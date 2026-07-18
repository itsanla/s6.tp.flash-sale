package domain

import (
	"context"
	"errors"
	"time"
)

// Status order dalam siklus hidup flash sale.
const (
	StatusPending   = "PENDING"   // order dibuat, stok direservasi, menunggu pembayaran
	StatusPaid      = "PAID"      // pembayaran sukses, tiket terbit
	StatusExpired   = "EXPIRED"   // tidak dibayar sampai batas waktu, stok dikembalikan
	StatusCancelled = "CANCELLED" // dibatalkan pengguna, stok dikembalikan
)

// Error domain yang dipetakan ke HTTP status oleh handler.
var (
	ErrProductNotFound   = errors.New("produk tidak ditemukan")
	ErrOutOfStock        = errors.New("stok tiket habis")
	ErrOrderNotFound     = errors.New("order tidak ditemukan")
	ErrOrderNotPending   = errors.New("order sudah tidak berstatus PENDING")
	ErrOrderExpired      = errors.New("order sudah kedaluwarsa")
	ErrInvalidQuantity   = errors.New("jumlah tiket tidak valid")
	ErrBatchNotFound     = errors.New("batch uji beban tidak ditemukan")
	ErrBatchTooLarge     = errors.New("jumlah pesanan melebihi batas maksimum uji beban")
	ErrInvalidCredentials = errors.New("username atau password salah")
	ErrProductExists      = errors.New("produk dengan id tersebut sudah ada")
	ErrProductHasOrders   = errors.New("produk tidak dapat dihapus karena sudah memiliki order")
)

// Product adalah produk/tiket yang dijual saat flash sale.
type Product struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Stock int64  `json:"stock"` // stok tersisa, dibaca real-time dari Redis
}

// Order merepresentasikan satu pemesanan tiket.
type Order struct {
	ID        string    `json:"id"`
	ProductID string    `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// NotifyMessage adalah payload yang dikirim ke RabbitMQ untuk consumer notifikasi.
type NotifyMessage struct {
	OrderID   string    `json:"order_id"`
	ProductID string    `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Event     string    `json:"event"` // "ORDER_CREATED" | "ORDER_PAID"
	Timestamp time.Time `json:"timestamp"`
}

// ExpiryMessage adalah payload yang di-dead-letter setelah TTL untuk memicu auto-expire.
type ExpiryMessage struct {
	OrderID string `json:"order_id"`
}

// BulkMessage adalah satu "tiket antrean" pada uji beban: mewakili satu niat
// pemesanan yang dikirim ke RabbitMQ untuk diproses worker secara asinkron.
type BulkMessage struct {
	BatchID   string `json:"batch_id"`
	ProductID string `json:"product_id"`
	Seq       int    `json:"seq"`
}

// BatchStatus adalah progres real-time sebuah batch uji beban, dibaca UI via polling.
type BatchStatus struct {
	BatchID   string `json:"batch_id"`
	Requested int64  `json:"requested"` // total pesanan yang diminta
	Submitted int64  `json:"submitted"` // sudah dikirim ke antrean RabbitMQ
	Processed int64  `json:"processed"` // sudah selesai diproses worker
	Success   int64  `json:"success"`
	Failed    int64  `json:"failed"`
}

// StockRepository adalah kontrak penyimpanan stok & order berbasis Redis (topik m1).
type StockRepository interface {
	SeedProduct(ctx context.Context, p Product) error
	GetProduct(ctx context.Context, id string) (*Product, error)
	// ListProducts mengembalikan seluruh produk pada katalog flash sale.
	ListProducts(ctx context.Context) ([]Product, error)
	// CreateProduct menambah produk baru (Admin). Gagal bila ID sudah dipakai.
	CreateProduct(ctx context.Context, p Product) error
	// UpdateProduct memperbarui nama dan/atau stok (Admin). newStock nil berarti stok tidak diubah.
	UpdateProduct(ctx context.Context, id, name string, newStock *int64) error
	// DeleteProduct menghapus produk (Admin). Gagal bila produk sudah memiliki order.
	DeleteProduct(ctx context.Context, id string) error
	// TryReserve mengurangi stok secara atomik (Lua). Mengembalikan sisa stok.
	TryReserve(ctx context.Context, productID string, qty int) (remaining int64, err error)
	// RestoreStock mengembalikan stok saat order expired/cancelled.
	RestoreStock(ctx context.Context, productID string, qty int) error

	SaveOrder(ctx context.Context, o Order) error
	GetOrder(ctx context.Context, id string) (*Order, error)
	UpdateOrderStatus(ctx context.Context, id, status string) error
	ListOrders(ctx context.Context, limit int) ([]Order, error)

	// CreateBatch menginisialisasi tracker progres batch uji beban.
	CreateBatch(ctx context.Context, batchID string, requested int64) error
	// SetBatchSubmitted memperbarui jumlah pesanan yang sudah dikirim ke antrean.
	SetBatchSubmitted(ctx context.Context, batchID string, submitted int64) error
	// IncrBatchProcessed dipanggil worker setiap satu pesanan selesai diproses.
	IncrBatchProcessed(ctx context.Context, batchID string, success bool) error
	GetBatch(ctx context.Context, batchID string) (*BatchStatus, error)
}

// Publisher adalah kontrak pengiriman pesan ke RabbitMQ (topik m2).
type Publisher interface {
	PublishNotify(ctx context.Context, msg NotifyMessage) error
	// PublishExpiry mengirim pesan ke antrean ber-TTL; akan di-dead-letter setelah ttl.
	PublishExpiry(ctx context.Context, msg ExpiryMessage) error
	// PublishBulk mengirim satu pesanan uji beban ke antrean bulk.
	PublishBulk(ctx context.Context, msg BulkMessage) error
}
