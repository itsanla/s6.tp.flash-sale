package domain

import (
	"context"
	"errors"
	"time"
)

// Error domain yang dipetakan handler ke HTTP status yang sesuai.
var (
	ErrRideNotFound       = errors.New("wahana tidak ditemukan")
	ErrRideInactive       = errors.New("wahana sedang tidak beroperasi")
	ErrRideHasOrders      = errors.New("wahana tidak dapat dihapus karena sudah pernah dipesan")
	ErrQuotaNotEnough     = errors.New("kuota tiket wahana pada tanggal tersebut tidak mencukupi")
	ErrOrderNotFound      = errors.New("order tidak ditemukan")
	ErrOrderNotPending    = errors.New("order sudah tidak berstatus menunggu pembayaran")
	ErrOrderExpired       = errors.New("batas waktu pembayaran order sudah lewat")
	ErrTicketNotFound     = errors.New("tiket tidak ditemukan")
	ErrTicketUsed         = errors.New("tiket sudah pernah dipakai")
	ErrInvalidInput       = errors.New("data yang dikirim tidak valid")
	ErrInvalidCredentials = errors.New("username atau password salah")
)

// RideRepository adalah kontrak penyimpanan katalog wahana pada SQLite.
type RideRepository interface {
	List(ctx context.Context, category string, activeOnly bool) ([]Ride, error)
	GetBySlug(ctx context.Context, slug string) (*Ride, error)
	GetByID(ctx context.Context, id int64) (*Ride, error)
	Create(ctx context.Context, r *Ride) error
	Update(ctx context.Context, r *Ride) error
	Delete(ctx context.Context, id int64) error
	CountOrderItems(ctx context.Context, rideID int64) (int, error)
}

// OrderRepository adalah kontrak penyimpanan order dan itemnya pada SQLite.
type OrderRepository interface {
	Create(ctx context.Context, o *Order) error
	GetByCode(ctx context.Context, code string) (*Order, error)
	ListByStatus(ctx context.Context, status string, limit int) ([]Order, error)
	ListRecent(ctx context.Context, limit int) ([]Order, error)
	MarkPaid(ctx context.Context, code string, paidAt time.Time) error
	UpdateStatus(ctx context.Context, code, status string) error
	Stats(ctx context.Context) (*Stats, error)
}

// TicketRepository adalah kontrak penyimpanan tiket yang sudah terbit.
type TicketRepository interface {
	CreateBatch(ctx context.Context, tickets []Ticket) error
	ListByOrderCode(ctx context.Context, orderCode string) ([]Ticket, error)
	CountByOrderID(ctx context.Context, orderID int64) (int, error)
	GetByCode(ctx context.Context, code string) (*Ticket, error)
	MarkUsed(ctx context.Context, code string, usedAt time.Time) error
}

// QuotaStore adalah kontrak kuota harian per wahana yang disimpan di Redis.
// Reservasi dilakukan atomik supaya kuota tidak pernah terjual berlebih (zero-oversell)
// walaupun banyak pengunjung memesan pada saat bersamaan.
type QuotaStore interface {
	TryReserve(ctx context.Context, rideID int64, date string, qty, dailyQuota int) (remaining int64, err error)
	Restore(ctx context.Context, rideID int64, date string, qty int) error
	Available(ctx context.Context, rideID int64, date string, dailyQuota int) (int64, error)
}

// Cache adalah kontrak cache sederhana berbasis Redis untuk data yang sering dibaca.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration)
	Delete(ctx context.Context, keys ...string)
}

// Publisher adalah kontrak pengiriman pesan ke RabbitMQ.
type Publisher interface {
	PublishNotify(ctx context.Context, msg NotifyMessage) error
	PublishTicketJob(ctx context.Context, msg TicketJob) error
	PublishExpiry(ctx context.Context, msg ExpiryMessage) error
}
