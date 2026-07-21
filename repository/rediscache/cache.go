package rediscache

import (
	"context"
	"time"

	"wahanapark/domain"

	"github.com/redis/go-redis/v9"
)

// Kunci cache yang dipakai aplikasi.
const (
	KeyRideCatalog = "wahana:cache:katalog"
	KeyStats       = "wahana:cache:statistik"
)

type cache struct{ client *redis.Client }

// NewCache membungkus Redis sebagai cache baca cepat. Kegagalan cache sengaja tidak
// pernah dianggap fatal: bila Redis bermasalah, pemanggil cukup membaca ulang dari SQLite.
func NewCache(client *redis.Client) domain.Cache { return &cache{client: client} }

func (c *cache) Get(ctx context.Context, key string) ([]byte, bool) {
	b, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, false
	}
	return b, true
}

func (c *cache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) {
	_ = c.client.Set(ctx, key, value, ttl).Err()
}

func (c *cache) Delete(ctx context.Context, keys ...string) {
	if len(keys) == 0 {
		return
	}
	_ = c.client.Del(ctx, keys...).Err()
}
