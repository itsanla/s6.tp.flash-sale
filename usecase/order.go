package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	"wahanapark/domain"
	"wahanapark/qris"
	"wahanapark/repository/rediscache"
)

// CheckoutItem adalah satu baris permintaan pembelian dari pengunjung.
type CheckoutItem struct {
	RideID   int64 `json:"ride_id"`
	Quantity int   `json:"quantity"`
}

// CheckoutRequest adalah data lengkap yang dikirim halaman keranjang.
type CheckoutRequest struct {
	CustomerName  string         `json:"customer_name"`
	CustomerEmail string         `json:"customer_email"`
	CustomerPhone string         `json:"customer_phone"`
	VisitDate     string         `json:"visit_date"`
	Items         []CheckoutItem `json:"items"`
}

// OrderUsecase memuat logika pemesanan tiket: reservasi kuota, penerbitan QRIS,
// pelunasan pembayaran, penerbitan tiket, sampai pembatalan otomatis.
type OrderUsecase struct {
	orders     domain.OrderRepository
	rides      domain.RideRepository
	tickets    domain.TicketRepository
	quota      domain.QuotaStore
	cache      domain.Cache
	publisher  domain.Publisher
	qrisGen    *qris.Generator
	paymentTTL time.Duration
}

func NewOrderUsecase(
	orders domain.OrderRepository,
	rides domain.RideRepository,
	tickets domain.TicketRepository,
	quota domain.QuotaStore,
	cache domain.Cache,
	publisher domain.Publisher,
	qrisGen *qris.Generator,
	paymentTTL time.Duration,
) *OrderUsecase {
	return &OrderUsecase{
		orders: orders, rides: rides, tickets: tickets, quota: quota,
		cache: cache, publisher: publisher, qrisGen: qrisGen, paymentTTL: paymentTTL,
	}
}

// Checkout membuat order baru berstatus menunggu pembayaran.
//
// Kuota setiap wahana direservasi lebih dulu secara atomik di Redis. Bila salah satu
// wahana kehabisan kuota, seluruh reservasi yang sudah terlanjur diambil dikembalikan
// sehingga tidak ada kuota yang tertahan sia sia.
func (u *OrderUsecase) Checkout(ctx context.Context, req CheckoutRequest) (*domain.Order, error) {
	if strings.TrimSpace(req.CustomerName) == "" || len(req.Items) == 0 {
		return nil, domain.ErrInvalidInput
	}
	if _, err := time.Parse("2006-01-02", req.VisitDate); err != nil {
		return nil, domain.ErrInvalidInput
	}

	type reserved struct {
		rideID int64
		qty    int
	}
	var taken []reserved

	// Mengembalikan seluruh kuota yang sempat direservasi bila proses gagal di tengah jalan.
	rollback := func() {
		for _, t := range taken {
			if err := u.quota.Restore(ctx, t.rideID, req.VisitDate, t.qty); err != nil {
				log.Printf("[order] gagal mengembalikan kuota wahana %d: %v", t.rideID, err)
			}
		}
	}

	items := make([]domain.OrderItem, 0, len(req.Items))
	var total int64

	for _, ci := range req.Items {
		if ci.Quantity <= 0 {
			rollback()
			return nil, domain.ErrInvalidInput
		}
		ride, err := u.rides.GetByID(ctx, ci.RideID)
		if err != nil {
			rollback()
			return nil, err
		}
		if !ride.IsActive {
			rollback()
			return nil, domain.ErrRideInactive
		}
		if _, err := u.quota.TryReserve(ctx, ride.ID, req.VisitDate, ci.Quantity, ride.DailyQuota); err != nil {
			rollback()
			if err == domain.ErrQuotaNotEnough {
				return nil, fmt.Errorf("%w: %s", domain.ErrQuotaNotEnough, ride.Name)
			}
			return nil, err
		}
		taken = append(taken, reserved{rideID: ride.ID, qty: ci.Quantity})

		subtotal := ride.Price * int64(ci.Quantity)
		total += subtotal
		items = append(items, domain.OrderItem{
			RideID: ride.ID, RideSlug: ride.Slug, RideName: ride.Name, RideEmoji: ride.Emoji,
			UnitPrice: ride.Price, Quantity: ci.Quantity, Subtotal: subtotal,
		})
	}

	now := time.Now()
	order := &domain.Order{
		Code:          "ORD-" + randomCode(6),
		CustomerName:  strings.TrimSpace(req.CustomerName),
		CustomerEmail: strings.TrimSpace(req.CustomerEmail),
		CustomerPhone: strings.TrimSpace(req.CustomerPhone),
		VisitDate:     req.VisitDate,
		Status:        domain.StatusPending,
		TotalAmount:   total,
		CreatedAt:     now,
		ExpiresAt:     now.Add(u.paymentTTL),
		Items:         items,
	}
	order.QRISPayload = u.qrisGen.BuildPayload(order.Code, total)

	if err := u.orders.Create(ctx, order); err != nil {
		rollback()
		return nil, err
	}
	u.cache.Delete(ctx, rediscache.KeyStats)

	// Notifikasi dan penjadwalan kedaluwarsa dikirim asinkron lewat RabbitMQ.
	u.notify(ctx, domain.EventOrderCreated, order, fmt.Sprintf("Order %s dibuat, menunggu pembayaran QRIS", order.Code))
	if err := u.publisher.PublishExpiry(ctx, domain.ExpiryMessage{OrderCode: order.Code}); err != nil {
		log.Printf("[order] gagal menjadwalkan kedaluwarsa order %s: %v", order.Code, err)
	}
	return order, nil
}

// GetByCode mengembalikan detail order beserta itemnya.
func (u *OrderUsecase) GetByCode(ctx context.Context, code string) (*domain.Order, error) {
	return u.orders.GetByCode(ctx, code)
}

// ListPending dipakai halaman uji QRIS untuk menampilkan order yang menunggu pembayaran.
func (u *OrderUsecase) ListPending(ctx context.Context, limit int) ([]domain.Order, error) {
	return u.orders.ListByStatus(ctx, domain.StatusPending, limit)
}

func (u *OrderUsecase) ListRecent(ctx context.Context, limit int) ([]domain.Order, error) {
	return u.orders.ListRecent(ctx, limit)
}

// SettlePayment menandai order sebagai lunas. Fungsi ini dipanggil halaman uji
// /test/qris-list sebagai pengganti notifikasi dari penyedia pembayaran sungguhan.
//
// Penerbitan tiket sengaja tidak dikerjakan di sini melainkan dikirim sebagai pekerjaan
// ke RabbitMQ, supaya respons pembayaran tetap cepat berapa pun jumlah tiket yang dibeli.
func (u *OrderUsecase) SettlePayment(ctx context.Context, code string) (*domain.Order, error) {
	order, err := u.orders.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if order.Status != domain.StatusPending {
		return nil, domain.ErrOrderNotPending
	}
	if time.Now().After(order.ExpiresAt) {
		return nil, domain.ErrOrderExpired
	}
	now := time.Now()
	if err := u.orders.MarkPaid(ctx, code, now); err != nil {
		return nil, err
	}
	order.Status = domain.StatusPaid
	order.PaidAt = &now
	u.cache.Delete(ctx, rediscache.KeyStats)

	u.notify(ctx, domain.EventOrderPaid, order, fmt.Sprintf("Pembayaran order %s diterima", order.Code))
	if err := u.publisher.PublishTicketJob(ctx, domain.TicketJob{OrderCode: order.Code}); err != nil {
		log.Printf("[order] gagal mengantrekan penerbitan tiket order %s: %v", order.Code, err)
	}
	return order, nil
}

// Cancel membatalkan order yang belum dibayar dan mengembalikan kuotanya.
func (u *OrderUsecase) Cancel(ctx context.Context, code string) (*domain.Order, error) {
	order, err := u.orders.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if order.Status != domain.StatusPending {
		return nil, domain.ErrOrderNotPending
	}
	if err := u.orders.UpdateStatus(ctx, code, domain.StatusCancelled); err != nil {
		return nil, err
	}
	u.releaseQuota(ctx, order)
	order.Status = domain.StatusCancelled
	u.cache.Delete(ctx, rediscache.KeyStats)
	u.notify(ctx, domain.EventOrderCancelled, order, fmt.Sprintf("Order %s dibatalkan pengunjung", order.Code))
	return order, nil
}

// ExpireOrder dijalankan worker saat pesan penjadwalan kedaluwarsa tiba dari Dead Letter
// Queue. Aman dipanggil berulang: order yang sudah dibayar atau dibatalkan diabaikan.
func (u *OrderUsecase) ExpireOrder(ctx context.Context, code string) error {
	order, err := u.orders.GetByCode(ctx, code)
	if err == domain.ErrOrderNotFound {
		return nil
	}
	if err != nil {
		return err
	}
	if order.Status != domain.StatusPending {
		return nil
	}
	if err := u.orders.UpdateStatus(ctx, code, domain.StatusExpired); err != nil {
		return err
	}
	u.releaseQuota(ctx, order)
	u.cache.Delete(ctx, rediscache.KeyStats)
	log.Printf("[expiry] order %s kedaluwarsa, kuota %d wahana dikembalikan", code, len(order.Items))
	u.notify(ctx, domain.EventOrderExpired, order, fmt.Sprintf("Order %s kedaluwarsa karena tidak dibayar", order.Code))
	return nil
}

// IssueTickets menerbitkan tiket untuk order yang sudah lunas. Dijalankan worker.
// Bila tiket sudah pernah terbit, pemanggilan berikutnya tidak menerbitkan tiket ganda.
func (u *OrderUsecase) IssueTickets(ctx context.Context, code string) error {
	order, err := u.orders.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	if order.Status != domain.StatusPaid {
		return nil
	}
	existing, err := u.tickets.CountByOrderID(ctx, order.ID)
	if err != nil {
		return err
	}
	if existing > 0 {
		return nil
	}

	now := time.Now()
	tickets := make([]domain.Ticket, 0, 8)
	for _, item := range order.Items {
		for i := 0; i < item.Quantity; i++ {
			tickets = append(tickets, domain.Ticket{
				Code:      "TKT-" + randomCode(6),
				OrderID:   order.ID,
				OrderCode: order.Code,
				RideID:    item.RideID,
				RideName:  item.RideName,
				RideEmoji: item.RideEmoji,
				VisitDate: order.VisitDate,
				Status:    domain.TicketIssued,
				IssuedAt:  now,
			})
		}
	}
	if err := u.tickets.CreateBatch(ctx, tickets); err != nil {
		return err
	}
	u.cache.Delete(ctx, rediscache.KeyStats)
	log.Printf("[tiket] %d tiket terbit untuk order %s", len(tickets), order.Code)
	u.notify(ctx, domain.EventTicketsIssued, order, fmt.Sprintf("%d tiket order %s sudah terbit", len(tickets), order.Code))
	return nil
}

// Tickets mengembalikan seluruh tiket milik sebuah order.
func (u *OrderUsecase) Tickets(ctx context.Context, orderCode string) ([]domain.Ticket, error) {
	if _, err := u.orders.GetByCode(ctx, orderCode); err != nil {
		return nil, err
	}
	return u.tickets.ListByOrderCode(ctx, orderCode)
}

// ScanTicket menandai satu tiket sudah dipakai masuk wahana.
func (u *OrderUsecase) ScanTicket(ctx context.Context, code string) (*domain.Ticket, error) {
	ticket, err := u.tickets.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if ticket.Status == domain.TicketUsed {
		return nil, domain.ErrTicketUsed
	}
	now := time.Now()
	if err := u.tickets.MarkUsed(ctx, code, now); err != nil {
		return nil, err
	}
	ticket.Status = domain.TicketUsed
	ticket.UsedAt = &now
	return ticket, nil
}

// Stats mengembalikan ringkasan angka untuk dashboard, dilayani dari cache bila ada.
func (u *OrderUsecase) Stats(ctx context.Context, ttl time.Duration) (*domain.Stats, error) {
	st, err := u.orders.Stats(ctx)
	if err != nil {
		return nil, err
	}
	return st, nil
}

func (u *OrderUsecase) releaseQuota(ctx context.Context, order *domain.Order) {
	for _, item := range order.Items {
		if err := u.quota.Restore(ctx, item.RideID, order.VisitDate, item.Quantity); err != nil {
			log.Printf("[order] gagal mengembalikan kuota wahana %d: %v", item.RideID, err)
		}
	}
}

func (u *OrderUsecase) notify(ctx context.Context, event string, order *domain.Order, detail string) {
	msg := domain.NotifyMessage{
		Event:     event,
		OrderCode: order.Code,
		Customer:  order.CustomerName,
		Detail:    detail,
		Timestamp: time.Now(),
	}
	if err := u.publisher.PublishNotify(ctx, msg); err != nil {
		log.Printf("[order] gagal mengirim notifikasi %s: %v", event, err)
	}
}

func randomCode(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return strings.ToUpper(hex.EncodeToString(b))
}
