package domain

import "time"

// Jenis event notifikasi yang dipublikasikan ke RabbitMQ.
const (
	EventOrderCreated   = "ORDER_CREATED"
	EventOrderPaid      = "ORDER_PAID"
	EventTicketsIssued  = "TICKETS_ISSUED"
	EventOrderExpired   = "ORDER_EXPIRED"
	EventOrderCancelled = "ORDER_CANCELLED"
)

// NotifyMessage dikonsumsi worker notifikasi (simulasi kirim email atau WhatsApp).
type NotifyMessage struct {
	Event     string    `json:"event"`
	OrderCode string    `json:"order_code"`
	Customer  string    `json:"customer"`
	Detail    string    `json:"detail"`
	Timestamp time.Time `json:"timestamp"`
}

// TicketJob memerintahkan worker menerbitkan tiket untuk sebuah order yang sudah dibayar.
// Pekerjaan ini sengaja dilakukan asinkron supaya respons pembayaran tetap cepat.
type TicketJob struct {
	OrderCode string `json:"order_code"`
}

// ExpiryMessage dikirim ke antrean ber-TTL saat order dibuat. Setelah TTL habis pesan
// di-dead-letter ke antrean pemroses untuk membatalkan order yang belum dibayar.
type ExpiryMessage struct {
	OrderCode string `json:"order_code"`
}
