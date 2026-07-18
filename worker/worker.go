package worker

import (
	"context"
	"encoding/json"
	"log"

	"flashsale/domain"
	"flashsale/queue"
	"flashsale/usecase"
)

// Worker menjalankan consumer RabbitMQ (topik m2):
//   1. notifyConsumer   -> simulasi kirim notifikasi (email/sms) saat order dibuat/dibayar
//   2. expiryConsumer   -> auto-expire order yang tidak dibayar (dari DLX)
type Worker struct {
	mq *queue.RabbitMQ
	uc *usecase.FlashSaleUsecase
}

func New(mq *queue.RabbitMQ, uc *usecase.FlashSaleUsecase) *Worker {
	return &Worker{mq: mq, uc: uc}
}

// Start menjalankan seluruh consumer sebagai goroutine. Non-blocking.
func (w *Worker) Start(ctx context.Context) error {
	if err := w.startNotifyConsumer(ctx); err != nil {
		return err
	}
	if err := w.startExpiryConsumer(ctx); err != nil {
		return err
	}
	log.Println("Worker berjalan: notify & expiry consumer aktif")
	return nil
}

func (w *Worker) startNotifyConsumer(ctx context.Context) error {
	deliveries, err := w.mq.Consume(queue.NotifyQueue)
	if err != nil {
		return err
	}
	go func() {
		for d := range deliveries {
			var msg domain.NotifyMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				log.Printf("[notify] pesan rusak, dibuang: %v", err)
				_ = d.Nack(false, false)
				continue
			}
			// Simulasi pengiriman notifikasi.
			switch msg.Event {
			case "ORDER_CREATED":
				log.Printf("[notify] 📧 Order %s dibuat (%d tiket). Email: 'Segera selesaikan pembayaran Anda.'", msg.OrderID, msg.Quantity)
			case "ORDER_PAID":
				log.Printf("[notify] 🎫 Order %s LUNAS. Email: 'Tiket Anda terbit, terima kasih!'", msg.OrderID)
			default:
				log.Printf("[notify] event %s untuk order %s", msg.Event, msg.OrderID)
			}
			_ = d.Ack(false)
		}
	}()
	return nil
}

func (w *Worker) startExpiryConsumer(ctx context.Context) error {
	deliveries, err := w.mq.Consume(queue.ExpiryProcessQueue)
	if err != nil {
		return err
	}
	go func() {
		for d := range deliveries {
			var msg domain.ExpiryMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				log.Printf("[expiry] pesan rusak, dibuang: %v", err)
				_ = d.Nack(false, false)
				continue
			}
			if err := w.uc.ExpireOrder(ctx, msg.OrderID); err != nil {
				log.Printf("[expiry] gagal memproses order %s: %v", msg.OrderID, err)
				_ = d.Nack(false, false) // hindari loop tak berujung
				continue
			}
			_ = d.Ack(false)
		}
	}()
	return nil
}
