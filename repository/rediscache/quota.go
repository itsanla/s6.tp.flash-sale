package rediscache

import (
	"context"
	"fmt"
	"time"

	"wahanapark/domain"

	"github.com/redis/go-redis/v9"
)

type quotaStore struct {
	client *redis.Client
	ttl    time.Duration
}

// NewQuotaStore membuat penyimpan kuota harian berbasis Redis.
//
// Kuota disimpan per kombinasi wahana dan tanggal kunjungan, lalu dikurangi memakai
// Lua script. Lua dijalankan Redis sebagai satu operasi tak terbagi, sehingga
// pengecekan sisa kuota dan pengurangannya tidak dapat disisipi permintaan lain.
// Inilah yang menjamin tiket tidak pernah terjual melebihi kuota (zero oversell).
func NewQuotaStore(client *redis.Client, ttlDays int) domain.QuotaStore {
	return &quotaStore{client: client, ttl: time.Duration(ttlDays) * 24 * time.Hour}
}

func quotaKey(rideID int64, date string) string {
	return fmt.Sprintf("wahana:kuota:%d:%s", rideID, date)
}

// reserveScript menyiapkan kuota bila kunci belum ada, lalu menguranginya.
// Mengembalikan sisa kuota bila berhasil, atau -1 bila kuota tidak mencukupi.
var reserveScript = redis.NewScript(`
local current = redis.call('GET', KEYS[1])
if current == false then
  redis.call('SET', KEYS[1], ARGV[2], 'EX', ARGV[3])
  current = ARGV[2]
end
local qty = tonumber(ARGV[1])
if tonumber(current) < qty then
  return -1
end
return redis.call('DECRBY', KEYS[1], qty)
`)

// restoreScript hanya menambah kembali kuota bila kuncinya masih ada. Bila kunci sudah
// kedaluwarsa berarti hari kunjungan sudah lewat dan pengembalian kuota tidak relevan.
var restoreScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[1]) == 1 then
  return redis.call('INCRBY', KEYS[1], ARGV[1])
end
return -1
`)

func (q *quotaStore) TryReserve(ctx context.Context, rideID int64, date string, qty, dailyQuota int) (int64, error) {
	res, err := reserveScript.Run(ctx, q.client,
		[]string{quotaKey(rideID, date)}, qty, dailyQuota, int(q.ttl.Seconds())).Int64()
	if err != nil {
		return 0, err
	}
	if res < 0 {
		return 0, domain.ErrQuotaNotEnough
	}
	return res, nil
}

func (q *quotaStore) Restore(ctx context.Context, rideID int64, date string, qty int) error {
	return restoreScript.Run(ctx, q.client, []string{quotaKey(rideID, date)}, qty).Err()
}

func (q *quotaStore) Available(ctx context.Context, rideID int64, date string, dailyQuota int) (int64, error) {
	v, err := q.client.Get(ctx, quotaKey(rideID, date)).Int64()
	if err == redis.Nil {
		// Belum ada transaksi pada tanggal tersebut, kuota masih penuh.
		return int64(dailyQuota), nil
	}
	if err != nil {
		return 0, err
	}
	return v, nil
}
