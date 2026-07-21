package domain

import "time"

// Status order dalam siklus hidup pemesanan tiket.
const (
	StatusPending   = "PENDING"   // QRIS terbit, menunggu pembayaran
	StatusPaid      = "PAID"      // pembayaran diterima, tiket diterbitkan asinkron
	StatusExpired   = "EXPIRED"   // tidak dibayar sampai batas waktu, kuota dikembalikan
	StatusCancelled = "CANCELLED" // dibatalkan pengunjung, kuota dikembalikan
)

// Status tiket setelah diterbitkan.
const (
	TicketIssued = "ISSUED" // sudah terbit, belum dipakai masuk wahana
	TicketUsed   = "USED"   // sudah dipindai di gerbang wahana
)

// Kategori wahana. Dipakai untuk filter di katalog.
const (
	CategoryEkstrem     = "ekstrem"
	CategoryKeluarga    = "keluarga"
	CategoryAnak        = "anak"
	CategoryAir         = "air"
	CategoryPetualangan = "petualangan"
	CategoryIndoor      = "indoor"
)

// CategoryLabels memetakan slug kategori ke label tampilan Bahasa Indonesia.
var CategoryLabels = map[string]string{
	CategoryEkstrem:     "Ekstrem",
	CategoryKeluarga:    "Keluarga",
	CategoryAnak:        "Anak",
	CategoryAir:         "Wahana Air",
	CategoryPetualangan: "Petualangan",
	CategoryIndoor:      "Indoor",
}

// Ride adalah satu wahana yang tiketnya dijual.
type Ride struct {
	ID          int64  `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Tagline     string `json:"tagline"`
	Description string `json:"description"`
	Emoji       string `json:"emoji"`
	Price       int64  `json:"price"`         // rupiah per tiket
	DurationMin int    `json:"duration_min"`  // durasi wahana dalam menit
	MinHeightCm int    `json:"min_height_cm"` // syarat tinggi minimum, 0 berarti bebas
	ThrillLevel int    `json:"thrill_level"`  // 1 (santai) sampai 5 (ekstrem)
	DailyQuota  int    `json:"daily_quota"`   // kapasitas tiket per hari
	IsActive    bool   `json:"is_active"`

	// Diisi saat runtime dari Redis, bukan kolom database.
	Available int64 `json:"available"`
}

// OrderItem adalah satu baris pesanan (satu wahana beserta jumlah tiketnya).
type OrderItem struct {
	ID        int64  `json:"id"`
	OrderID   int64  `json:"-"`
	RideID    int64  `json:"ride_id"`
	RideSlug  string `json:"ride_slug"`
	RideName  string `json:"ride_name"`
	RideEmoji string `json:"ride_emoji"`
	UnitPrice int64  `json:"unit_price"`
	Quantity  int    `json:"quantity"`
	Subtotal  int64  `json:"subtotal"`
}

// Order adalah satu transaksi pembelian tiket oleh pengunjung.
type Order struct {
	ID            int64       `json:"-"`
	Code          string      `json:"code"`
	CustomerName  string      `json:"customer_name"`
	CustomerEmail string      `json:"customer_email"`
	CustomerPhone string      `json:"customer_phone"`
	VisitDate     string      `json:"visit_date"` // format YYYY-MM-DD
	Status        string      `json:"status"`
	TotalAmount   int64       `json:"total_amount"`
	QRISPayload   string      `json:"qris_payload"`
	CreatedAt     time.Time   `json:"created_at"`
	ExpiresAt     time.Time   `json:"expires_at"`
	PaidAt        *time.Time  `json:"paid_at,omitempty"`
	Items         []OrderItem `json:"items"`
}

// Ticket adalah satu tiket masuk wahana, terbit setelah pembayaran sukses.
type Ticket struct {
	ID        int64      `json:"-"`
	Code      string     `json:"code"`
	OrderID   int64      `json:"-"`
	OrderCode string     `json:"order_code"`
	RideID    int64      `json:"ride_id"`
	RideName  string     `json:"ride_name"`
	RideEmoji string     `json:"ride_emoji"`
	VisitDate string     `json:"visit_date"`
	Status    string     `json:"status"`
	IssuedAt  time.Time  `json:"issued_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
}

// Stats adalah ringkasan angka untuk dashboard admin.
type Stats struct {
	TotalRides    int   `json:"total_rides"`
	TotalOrders   int   `json:"total_orders"`
	PaidOrders    int   `json:"paid_orders"`
	PendingOrders int   `json:"pending_orders"`
	TotalTickets  int   `json:"total_tickets"`
	Revenue       int64 `json:"revenue"`
}
