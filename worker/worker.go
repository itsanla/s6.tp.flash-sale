package worker

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"flashsale/domain"
	"flashsale/queue"
	"flashsale/usecase"
)

// Worker menjalankan consumer RabbitMQ (topik m2):
//   1. notifyConsumer -> simulasi kirim notifikasi (email/sms) saat order dibuat/dibayar
//   2. expiryConsumer -> auto-expire order yang tidak dibayar (dari DLX)
//   3. bulkConsumer    -> proses pesanan uji beban (load test) secara paralel
type Worker struct {
	mq              *queue.RabbitMQ
	uc              *usecase.FlashSaleUsecase
	bulkConcurrency int
	bulkDelay       time.Duration
}

func New(mq *queue.RabbitMQ, uc *usecase.FlashSaleUsecase, bulkConcurrency int, bulkDelay time.Duration) *Worker {
	return &Worker{mq: mq, uc: uc, bulkConcurrency: bulkConcurrency, bulkDelay: bulkDelay}
}

// Start menjalankan seluruh consumer sebagai goroutine. Non-blocking.
func (w *Worker) Start(ctx context.Context) error {
	if err := w.startNotifyConsumer(ctx); err != nil {
		return err
	}
	if err := w.startExpiryConsumer(ctx); err != nil {
		return err
	}
	if err := w.startBulkConsumer(ctx); err != nil {
		return err
	}
	log.Println("Worker berjalan: notify, expiry & bulk consumer aktif")
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

// startBulkConsumer menjalankan N goroutine yang berbagi satu channel delivery
// (competing consumers) — RabbitMQ otomatis membagi pesan di antara mereka,
// sehingga throughput proses meningkat linear terhadap bulkConcurrency.
// Prefetch dibuka lebih besar dari concurrency agar antrean lokal tiap
// goroutine tidak cepat kosong menunggu jaringan.
func (w *Worker) startBulkConsumer(ctx context.Context) error {
	deliveries, err := w.mq.ConsumeWithPrefetch(queue.BulkQueue, w.bulkConcurrency*4)
	if err != nil {
		return err
	}
	for i := 0; i < w.bulkConcurrency; i++ {
		go func() {
			for d := range deliveries {
				var msg domain.BulkMessage
				if err := json.Unmarshal(d.Body, &msg); err != nil {
					log.Printf("[loadtest] pesan rusak, dibuang: %v", err)
					_ = d.Nack(false, false)
					continue
				}
				if w.bulkDelay > 0 {
					time.Sleep(w.bulkDelay) // simulasi waktu proses pesanan
				}
				w.uc.ProcessBulkOrder(ctx, msg.BatchID)
				_ = d.Ack(false)
			}
		}()
	}
	return nil
}
