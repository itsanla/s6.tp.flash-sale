package worker

import (
	"context"
	"encoding/json"
	"log"

	"wahanapark/domain"
	"wahanapark/queue"
	"wahanapark/usecase"
)

// Worker menjalankan seluruh consumer RabbitMQ:
//
//  1. notifyConsumer  mensimulasikan pengiriman notifikasi ke pengunjung.
//  2. ticketConsumer  menerbitkan tiket setelah pembayaran diterima.
//  3. expiryConsumer  membatalkan order yang tidak dibayar sampai batas waktu.
type Worker struct {
	mq *queue.RabbitMQ
	uc *usecase.OrderUsecase
}

func New(mq *queue.RabbitMQ, uc *usecase.OrderUsecase) *Worker {
	return &Worker{mq: mq, uc: uc}
}

// Start menjalankan semua consumer sebagai goroutine dan langsung kembali.
func (w *Worker) Start(ctx context.Context) error {
	if err := w.startNotifyConsumer(ctx); err != nil {
		return err
	}
	if err := w.startTicketConsumer(ctx); err != nil {
		return err
	}
	if err := w.startExpiryConsumer(ctx); err != nil {
		return err
	}
	log.Println("Worker berjalan: consumer notifikasi, tiket, dan kedaluwarsa aktif")
	return nil
}

func (w *Worker) startNotifyConsumer(ctx context.Context) error {
	deliveries, err := w.mq.Consume(queue.NotifyQueue, 8)
	if err != nil {
		return err
	}
	go func() {
		for d := range deliveries {
			var msg domain.NotifyMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				log.Printf("[notifikasi] pesan rusak, dibuang: %v", err)
				_ = d.Nack(false, false)
				continue
			}
			switch msg.Event {
			case domain.EventOrderCreated:
				log.Printf("[notifikasi] Order %s dibuat atas nama %s. Kirim instruksi pembayaran QRIS.", msg.OrderCode, msg.Customer)
			case domain.EventOrderPaid:
				log.Printf("[notifikasi] Pembayaran order %s diterima. Kirim struk pembayaran.", msg.OrderCode)
			case domain.EventTicketsIssued:
				log.Printf("[notifikasi] Tiket order %s sudah terbit. Kirim tiket ke pengunjung.", msg.OrderCode)
			case domain.EventOrderExpired:
				log.Printf("[notifikasi] Order %s kedaluwarsa. Kirim pemberitahuan pembatalan.", msg.OrderCode)
			case domain.EventOrderCancelled:
				log.Printf("[notifikasi] Order %s dibatalkan pengunjung.", msg.OrderCode)
			default:
				log.Printf("[notifikasi] %s untuk order %s", msg.Event, msg.OrderCode)
			}
			_ = d.Ack(false)
		}
	}()
	return nil
}

// startTicketConsumer memproses penerbitan tiket. Prefetch dibuat kecil karena penulisan
// tiket masuk ke SQLite yang hanya melayani satu penulis pada satu waktu.
func (w *Worker) startTicketConsumer(ctx context.Context) error {
	deliveries, err := w.mq.Consume(queue.TicketQueue, 4)
	if err != nil {
		return err
	}
	go func() {
		for d := range deliveries {
			var job domain.TicketJob
			if err := json.Unmarshal(d.Body, &job); err != nil {
				log.Printf("[tiket] pesan rusak, dibuang: %v", err)
				_ = d.Nack(false, false)
				continue
			}
			if err := w.uc.IssueTickets(ctx, job.OrderCode); err != nil {
				log.Printf("[tiket] gagal menerbitkan tiket order %s: %v", job.OrderCode, err)
				_ = d.Nack(false, false)
				continue
			}
			_ = d.Ack(false)
		}
	}()
	return nil
}

func (w *Worker) startExpiryConsumer(ctx context.Context) error {
	deliveries, err := w.mq.Consume(queue.ExpiryProcessQueue, 4)
	if err != nil {
		return err
	}
	go func() {
		for d := range deliveries {
			var msg domain.ExpiryMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				log.Printf("[kedaluwarsa] pesan rusak, dibuang: %v", err)
				_ = d.Nack(false, false)
				continue
			}
			if err := w.uc.ExpireOrder(ctx, msg.OrderCode); err != nil {
				log.Printf("[kedaluwarsa] gagal memproses order %s: %v", msg.OrderCode, err)
				_ = d.Nack(false, false)
				continue
			}
			_ = d.Ack(false)
		}
	}()
	return nil
}
