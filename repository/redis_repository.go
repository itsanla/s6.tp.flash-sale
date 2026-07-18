package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"flashsale/domain"

	"github.com/redis/go-redis/v9"
)

// redisStockRepository mengimplementasikan domain.StockRepository memakai Redis.
//
// Topik m1 (Redis): stok disimpan sebagai counter integer dan diubah secara
// atomik lewat Lua script sehingga tidak mungkin terjadi oversell walaupun
// banyak request checkout terjadi bersamaan (race condition).
type redisStockRepository struct {
	client *redis.Client
}

func NewRedisStockRepository(client *redis.Client) domain.StockRepository {
	return &redisStockRepository{client: client}
}

func stockKey(productID string) string      { return "flashsale:stock:" + productID }
func nameKey(productID string) string       { return "flashsale:product:name:" + productID }
func orderKey(orderID string) string        { return "flashsale:order:" + orderID }
func batchKey(batchID string) string        { return "flashsale:batch:" + batchID }
func orderCountKey(productID string) string { return "flashsale:product:ordercount:" + productID }

const orderIndexKey = "flashsale:orders"      // sorted set: member=orderID, score=unix ts
const productIndexKey = "flashsale:products:index" // set: seluruh ID produk pada katalog

// reserveScript: cek stok lalu kurangi secara atomik.
//   KEYS[1] = stock key
//   ARGV[1] = qty
// Return: sisa stok (>=0) jika sukses, -1 jika produk tidak ada, -2 jika stok kurang.
var reserveScript = redis.NewScript(`
local stock = redis.call('GET', KEYS[1])
if stock == false then
  return -1
end
local qty = tonumber(ARGV[1])
if tonumber(stock) < qty then
  return -2
end
return redis.call('DECRBY', KEYS[1], qty)
`)

func (r *redisStockRepository) SeedProduct(ctx context.Context, p domain.Product) error {
	// SETNX: hanya set jika belum ada, agar restart tidak mereset stok berjalan.
	if err := r.client.SetNX(ctx, stockKey(p.ID), p.Stock, 0).Err(); err != nil {
		return err
	}
	if err := r.client.Set(ctx, nameKey(p.ID), p.Name, 0).Err(); err != nil {
		return err
	}
	return r.client.SAdd(ctx, productIndexKey, p.ID).Err()
}

func (r *redisStockRepository) ListProducts(ctx context.Context) ([]domain.Product, error) {
	ids, err := r.client.SMembers(ctx, productIndexKey).Result()
	if err != nil {
		return nil, err
	}
	products := make([]domain.Product, 0, len(ids))
	for _, id := range ids {
		p, err := r.GetProduct(ctx, id)
		if err == domain.ErrProductNotFound {
			continue // index basi (mis. produk pernah dihapus manual), abaikan
		}
		if err != nil {
			return nil, fmt.Errorf("gagal membaca produk %s: %w", id, err)
		}
		products = append(products, *p)
	}
	return products, nil
}

func (r *redisStockRepository) CreateProduct(ctx context.Context, p domain.Product) error {
	ok, err := r.client.SetNX(ctx, stockKey(p.ID), p.Stock, 0).Result()
	if err != nil {
		return err
	}
	if !ok {
		return domain.ErrProductExists
	}
	if err := r.client.Set(ctx, nameKey(p.ID), p.Name, 0).Err(); err != nil {
		return err
	}
	return r.client.SAdd(ctx, productIndexKey, p.ID).Err()
}

func (r *redisStockRepository) UpdateProduct(ctx context.Context, id, name string, newStock *int64) error {
	exists, err := r.client.Exists(ctx, stockKey(id)).Result()
	if err != nil {
		return err
	}
	if exists == 0 {
		return domain.ErrProductNotFound
	}
	pipe := r.client.TxPipeline()
	if name != "" {
		pipe.Set(ctx, nameKey(id), name, 0)
	}
	if newStock != nil {
		pipe.Set(ctx, stockKey(id), *newStock, 0)
	}
	_, err = pipe.Exec(ctx)
	return err
}

func (r *redisStockRepository) DeleteProduct(ctx context.Context, id string) error {
	exists, err := r.client.Exists(ctx, stockKey(id)).Result()
	if err != nil {
		return err
	}
	if exists == 0 {
		return domain.ErrProductNotFound
	}
	count, err := r.client.Get(ctx, orderCountKey(id)).Int64()
	if err != nil && err != redis.Nil {
		return err
	}
	if count > 0 {
		return domain.ErrProductHasOrders
	}
	pipe := r.client.TxPipeline()
	pipe.Del(ctx, stockKey(id), nameKey(id), orderCountKey(id))
	pipe.SRem(ctx, productIndexKey, id)
	_, err = pipe.Exec(ctx)
	return err
}

func (r *redisStockRepository) GetProduct(ctx context.Context, id string) (*domain.Product, error) {
	stock, err := r.client.Get(ctx, stockKey(id)).Int64()
	if err == redis.Nil {
		return nil, domain.ErrProductNotFound
	}
	if err != nil {
		return nil, err
	}
	name, err := r.client.Get(ctx, nameKey(id)).Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}
	return &domain.Product{ID: id, Name: name, Stock: stock}, nil
}

func (r *redisStockRepository) TryReserve(ctx context.Context, productID string, qty int) (int64, error) {
	res, err := reserveScript.Run(ctx, r.client, []string{stockKey(productID)}, qty).Int64()
	if err != nil {
		return 0, err
	}
	switch res {
	case -1:
		return 0, domain.ErrProductNotFound
	case -2:
		return 0, domain.ErrOutOfStock
	default:
		return res, nil
	}
}

func (r *redisStockRepository) RestoreStock(ctx context.Context, productID string, qty int) error {
	return r.client.IncrBy(ctx, stockKey(productID), int64(qty)).Err()
}

func (r *redisStockRepository) SaveOrder(ctx context.Context, o domain.Order) error {
	data, err := json.Marshal(o)
	if err != nil {
		return err
	}
	pipe := r.client.TxPipeline()
	pipe.Set(ctx, orderKey(o.ID), data, 24*time.Hour)
	pipe.ZAdd(ctx, orderIndexKey, redis.Z{Score: float64(o.CreatedAt.Unix()), Member: o.ID})
	// Dilacak agar DeleteProduct bisa menolak penghapusan produk yang sudah terjual
	// (mencegah orphan order), mencontoh proteksi hapus TicketType pada EventHub.
	pipe.Incr(ctx, orderCountKey(o.ProductID))
	_, err = pipe.Exec(ctx)
	return err
}

func (r *redisStockRepository) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	data, err := r.client.Get(ctx, orderKey(id)).Bytes()
	if err == redis.Nil {
		return nil, domain.ErrOrderNotFound
	}
	if err != nil {
		return nil, err
	}
	var o domain.Order
	if err := json.Unmarshal(data, &o); err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *redisStockRepository) UpdateOrderStatus(ctx context.Context, id, status string) error {
	o, err := r.GetOrder(ctx, id)
	if err != nil {
		return err
	}
	o.Status = status
	data, err := json.Marshal(o)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, orderKey(id), data, 24*time.Hour).Err()
}

func (r *redisStockRepository) ListOrders(ctx context.Context, limit int) ([]domain.Order, error) {
	// Ambil order terbaru lebih dulu (skor tertinggi).
	ids, err := r.client.ZRevRange(ctx, orderIndexKey, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}
	orders := make([]domain.Order, 0, len(ids))
	for _, id := range ids {
		o, err := r.GetOrder(ctx, id)
		if err == domain.ErrOrderNotFound {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("gagal membaca order %s: %w", id, err)
		}
		orders = append(orders, *o)
	}
	return orders, nil
}

func (r *redisStockRepository) CreateBatch(ctx context.Context, batchID string, requested int64) error {
	pipe := r.client.TxPipeline()
	pipe.HSet(ctx, batchKey(batchID), map[string]any{
		"requested": requested,
		"submitted": 0,
		"processed": 0,
		"success":   0,
		"failed":    0,
	})
	pipe.Expire(ctx, batchKey(batchID), 24*time.Hour)
	_, err := pipe.Exec(ctx)
	return err
}

func (r *redisStockRepository) SetBatchSubmitted(ctx context.Context, batchID string, submitted int64) error {
	return r.client.HSet(ctx, batchKey(batchID), "submitted", submitted).Err()
}

func (r *redisStockRepository) IncrBatchProcessed(ctx context.Context, batchID string, success bool) error {
	pipe := r.client.TxPipeline()
	pipe.HIncrBy(ctx, batchKey(batchID), "processed", 1)
	if success {
		pipe.HIncrBy(ctx, batchKey(batchID), "success", 1)
	} else {
		pipe.HIncrBy(ctx, batchKey(batchID), "failed", 1)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (r *redisStockRepository) GetBatch(ctx context.Context, batchID string) (*domain.BatchStatus, error) {
	vals, err := r.client.HGetAll(ctx, batchKey(batchID)).Result()
	if err != nil {
		return nil, err
	}
	if len(vals) == 0 {
		return nil, domain.ErrBatchNotFound
	}
	get := func(k string) int64 {
		n, _ := strconv.ParseInt(vals[k], 10, 64)
		return n
	}
	return &domain.BatchStatus{
		BatchID:   batchID,
		Requested: get("requested"),
		Submitted: get("submitted"),
		Processed: get("processed"),
		Success:   get("success"),
		Failed:    get("failed"),
	}, nil
}
