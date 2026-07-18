package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"flashsale/domain"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Nama exchange & queue. Dipakai bersama oleh publisher dan consumer.
const (
	NotifyExchange = "flashsale.notify"        // fanout: broadcast notifikasi order
	NotifyQueue    = "flashsale.notify.queue"  // consumer notifikasi (email/sms/dsb.)

	// Mekanisme auto-expire memakai TTL + Dead Letter Exchange (DLX).
	ExpiryExchange     = "flashsale.expiry"           // tujuan publish pesan order baru
	ExpiryWaitQueue    = "flashsale.expiry.wait"      // queue ber-TTL, TANPA consumer
	DLX                = "flashsale.dlx"               // dead-letter exchange
	ExpiryProcessQueue = "flashsale.expiry.process"   // consumer pemroses order kedaluwarsa
	ExpiryRoutingKey   = "expired"
)

// RabbitMQ membungkus koneksi dan channel AMQP.
type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	ttl     time.Duration
}

// Connect membuka koneksi ke RabbitMQ dengan retry, lalu mendeklarasikan topologi.
func Connect(url string, orderTTL time.Duration) (*RabbitMQ, error) {
	var conn *amqp.Connection
	var err error

	// RabbitMQ sering belum siap saat container app start; retry beberapa kali.
	for i := 1; i <= 15; i++ {
		conn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		log.Printf("RabbitMQ belum siap (percobaan %d/15): %v", i, err)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("gagal terhubung ke RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	r := &RabbitMQ{conn: conn, channel: ch, ttl: orderTTL}
	if err := r.declareTopology(); err != nil {
		return nil, err
	}
	log.Println("RabbitMQ terhubung & topologi siap")
	return r, nil
}

func (r *RabbitMQ) declareTopology() error {
	ch := r.channel

	// --- Jalur notifikasi (fanout) ---
	if err := ch.ExchangeDeclare(NotifyExchange, "fanout", true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(NotifyQueue, true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind(NotifyQueue, "", NotifyExchange, false, nil); err != nil {
		return err
	}

	// --- Jalur auto-expire (TTL + DLX) ---
	// 1. DLX: menerima pesan yang sudah kedaluwarsa lalu meneruskan ke process queue.
	if err := ch.ExchangeDeclare(DLX, "direct", true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(ExpiryProcessQueue, true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind(ExpiryProcessQueue, ExpiryRoutingKey, DLX, false, nil); err != nil {
		return err
	}

	// 2. Exchange & wait queue ber-TTL. Pesan yang masuk TIDAK dikonsumsi;
	//    setelah TTL habis, RabbitMQ men-dead-letter-kan ke DLX -> ExpiryProcessQueue.
	if err := ch.ExchangeDeclare(ExpiryExchange, "direct", true, false, false, false, nil); err != nil {
		return err
	}
	waitArgs := amqp.Table{
		"x-message-ttl":             int32(r.ttl.Milliseconds()),
		"x-dead-letter-exchange":    DLX,
		"x-dead-letter-routing-key": ExpiryRoutingKey,
	}
	if _, err := ch.QueueDeclare(ExpiryWaitQueue, true, false, false, false, waitArgs); err != nil {
		return err
	}
	if err := ch.QueueBind(ExpiryWaitQueue, ExpiryRoutingKey, ExpiryExchange, false, nil); err != nil {
		return err
	}
	return nil
}

// PublishNotify mengirim notifikasi ke fanout exchange.
func (r *RabbitMQ) PublishNotify(ctx context.Context, msg domain.NotifyMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return r.channel.PublishWithContext(ctx, NotifyExchange, "", false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent,
		Timestamp:    msg.Timestamp,
	})
}

// PublishExpiry mengirim order ke wait queue; akan otomatis kedaluwarsa setelah TTL.
func (r *RabbitMQ) PublishExpiry(ctx context.Context, msg domain.ExpiryMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return r.channel.PublishWithContext(ctx, ExpiryExchange, ExpiryRoutingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent,
	})
}

// Consume mengembalikan channel delivery untuk sebuah queue.
func (r *RabbitMQ) Consume(queueName string) (<-chan amqp.Delivery, error) {
	// Proses satu per satu agar adil antar worker.
	if err := r.channel.Qos(1, 0, false); err != nil {
		return nil, err
	}
	return r.channel.Consume(queueName, "", false, false, false, false, nil)
}

func (r *RabbitMQ) Close() {
	if r.channel != nil {
		_ = r.channel.Close()
	}
	if r.conn != nil {
		_ = r.conn.Close()
	}
}
