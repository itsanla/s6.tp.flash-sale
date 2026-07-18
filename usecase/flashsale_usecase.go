package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"time"

	"flashsale/domain"
)

// FlashSaleUsecase memuat logika bisnis flash sale: menggabungkan reservasi stok
// atomik (Redis / m1) dengan pemrosesan asinkron via message queue (RabbitMQ / m2).
type FlashSaleUsecase struct {
	repo             domain.StockRepository
	pub              domain.Publisher
	ttl              time.Duration
	productID        string
	loadTestMaxQty   int64
}

func NewFlashSaleUsecase(repo domain.StockRepository, pub domain.Publisher, ttl time.Duration, productID string, loadTestMaxQty int) *FlashSaleUsecase {
	return &FlashSaleUsecase{repo: repo, pub: pub, ttl: ttl, productID: productID, loadTestMaxQty: int64(loadTestMaxQty)}
}

func (u *FlashSaleUsecase) GetProduct(ctx context.Context, id string) (*domain.Product, error) {
	return u.repo.GetProduct(ctx, id)
}

func (u *FlashSaleUsecase) ListOrders(ctx context.Context, limit int) ([]domain.Order, error) {
	return u.repo.ListOrders(ctx, limit)
}

func (u *FlashSaleUsecase) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	return u.repo.GetOrder(ctx, id)
}

// Checkout: reservasi stok atomik lalu buat order PENDING.
// Bila reservasi gagal (stok habis) tidak ada order yang dibuat -> zero-oversell.
func (u *FlashSaleUsecase) Checkout(ctx context.Context, productID string, qty int) (*domain.Order, error) {
	if qty <= 0 {
		return nil, domain.ErrInvalidQuantity
	}

	// 1. Kurangi stok secara atomik di Redis (Lua). Ini inti jaminan zero-oversell.
	if _, err := u.repo.TryReserve(ctx, productID, qty); err != nil {
		return nil, err
	}

	// 2. Buat order PENDING dengan batas waktu bayar.
	now := time.Now()
	order := domain.Order{
		ID:        "ORD-" + randomID(),
		ProductID: productID,
		Quantity:  qty,
		Status:    domain.StatusPending,
		CreatedAt: now,
		ExpiresAt: now.Add(u.ttl),
	}
	if err := u.repo.SaveOrder(ctx, order); err != nil {
		// Kompensasi: kembalikan stok bila order gagal disimpan.
		_ = u.repo.RestoreStock(ctx, productID, qty)
		return nil, err
	}

	// 3. Publish notifikasi (async) + jadwalkan auto-expire lewat queue ber-TTL.
	u.publishNotify(ctx, order, "ORDER_CREATED")
	if err := u.pub.PublishExpiry(ctx, domain.ExpiryMessage{OrderID: order.ID}); err != nil {
		log.Printf("[usecase] gagal menjadwalkan expiry order %s: %v", order.ID, err)
	}

	return &order, nil
}

// Pay: tandai order PAID bila masih PENDING dan belum kedaluwarsa.
func (u *FlashSaleUsecase) Pay(ctx context.Context, orderID string) (*domain.Order, error) {
	order, err := u.repo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.Status != domain.StatusPending {
		return nil, domain.ErrOrderNotPending
	}
	if time.Now().After(order.ExpiresAt) {
		return nil, domain.ErrOrderExpired
	}
	if err := u.repo.UpdateOrderStatus(ctx, orderID, domain.StatusPaid); err != nil {
		return nil, err
	}
	order.Status = domain.StatusPaid
	u.publishNotify(ctx, *order, "ORDER_PAID")
	return order, nil
}

// Cancel: batalkan order PENDING milik pengguna & kembalikan stok.
func (u *FlashSaleUsecase) Cancel(ctx context.Context, orderID string) (*domain.Order, error) {
	order, err := u.repo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.Status != domain.StatusPending {
		return nil, domain.ErrOrderNotPending
	}
	if err := u.repo.UpdateOrderStatus(ctx, orderID, domain.StatusCancelled); err != nil {
		return nil, err
	}
	if err := u.repo.RestoreStock(ctx, order.ProductID, order.Quantity); err != nil {
		log.Printf("[usecase] gagal mengembalikan stok order %s: %v", order.ID, err)
	}
	order.Status = domain.StatusCancelled
	return order, nil
}

// ExpireOrder dipanggil oleh worker saat pesan order di-dead-letter (TTL habis).
// Idempotent: hanya order yang masih PENDING yang di-expire.
func (u *FlashSaleUsecase) ExpireOrder(ctx context.Context, orderID string) error {
	order, err := u.repo.GetOrder(ctx, orderID)
	if err == domain.ErrOrderNotFound {
		return nil // order sudah hilang, abaikan
	}
	if err != nil {
		return err
	}
	if order.Status != domain.StatusPending {
		// Sudah PAID/CANCELLED/EXPIRED -> abaikan (idempotent).
		return nil
	}
	if err := u.repo.UpdateOrderStatus(ctx, orderID, domain.StatusExpired); err != nil {
		return err
	}
	if err := u.repo.RestoreStock(ctx, order.ProductID, order.Quantity); err != nil {
		return err
	}
	log.Printf("[expiry] order %s kedaluwarsa, %d stok dikembalikan", orderID, order.Quantity)
	return nil
}

// StartBulkLoadTest memulai uji beban: menambah stok sejumlah qty (agar seluruh
// batch bisa berhasil, murni menguji throughput queue bukan zero-oversell —
// yang sudah dibuktikan lewat checkout normal), lalu MENGEMBALIKAN batch ID
// segera sambil mengirim pesan ke antrean bulk di background (goroutine).
// Klien mendapat respons sukses instan; pemrosesan aktual terjadi asinkron.
func (u *FlashSaleUsecase) StartBulkLoadTest(ctx context.Context, qty int64) (string, error) {
	if qty <= 0 {
		return "", domain.ErrInvalidQuantity
	}
	if qty > u.loadTestMaxQty {
		return "", domain.ErrBatchTooLarge
	}

	batchID := "BATCH-" + randomID()
	if err := u.repo.RestoreStock(ctx, u.productID, int(qty)); err != nil {
		return "", err
	}
	if err := u.repo.CreateBatch(ctx, batchID, qty); err != nil {
		return "", err
	}

	// Publish berjalan di background (context sendiri, terpisah dari request HTTP
	// yang sudah selesai) agar submit 50.000 pesan tidak memblokir respons klien.
	go u.submitBulkMessages(batchID, qty)

	return batchID, nil
}

func (u *FlashSaleUsecase) submitBulkMessages(batchID string, qty int64) {
	ctx := context.Background()
	// Flush progres "submitted" secara berkala (bukan tiap pesan) agar tidak
	// membebani Redis dengan puluhan ribu round-trip yang tidak perlu.
	flushEvery := qty / 50
	if flushEvery < 1 {
		flushEvery = 1
	}

	var i int64
	for i = 1; i <= qty; i++ {
		if err := u.pub.PublishBulk(ctx, domain.BulkMessage{BatchID: batchID, Seq: int(i)}); err != nil {
			log.Printf("[loadtest] gagal publish pesan %d/%d batch %s: %v", i, qty, batchID, err)
			continue
		}
		if i%flushEvery == 0 || i == qty {
			if err := u.repo.SetBatchSubmitted(ctx, batchID, i); err != nil {
				log.Printf("[loadtest] gagal update progres submitted batch %s: %v", batchID, err)
			}
		}
	}
	log.Printf("[loadtest] batch %s: %d pesan selesai dikirim ke antrean", batchID, qty)
}

// GetBatch mengembalikan progres real-time sebuah batch uji beban.
func (u *FlashSaleUsecase) GetBatch(ctx context.Context, batchID string) (*domain.BatchStatus, error) {
	return u.repo.GetBatch(ctx, batchID)
}

// ProcessBulkOrder dipanggil worker bulk untuk setiap pesan di antrean: membuat
// order nyata (reservasi atomik + simpan) lalu auto-pay (simulasi pembayaran
// instan) supaya hasilnya "berhasil" secara permanen, bukan menunggu bayar
// manual seperti alur checkout normal.
func (u *FlashSaleUsecase) ProcessBulkOrder(ctx context.Context, batchID string) {
	success := true
	order, err := u.Checkout(ctx, u.productID, 1)
	if err != nil {
		success = false
	} else if _, err := u.Pay(ctx, order.ID); err != nil {
		success = false
	}
	if err := u.repo.IncrBatchProcessed(ctx, batchID, success); err != nil {
		log.Printf("[loadtest] gagal update progres processed batch %s: %v", batchID, err)
	}
}

func (u *FlashSaleUsecase) publishNotify(ctx context.Context, o domain.Order, event string) {
	msg := domain.NotifyMessage{
		OrderID:   o.ID,
		ProductID: o.ProductID,
		Quantity:  o.Quantity,
		Event:     event,
		Timestamp: time.Now(),
	}
	if err := u.pub.PublishNotify(ctx, msg); err != nil {
		log.Printf("[usecase] gagal publish notify order %s: %v", o.ID, err)
	}
}

func randomID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
