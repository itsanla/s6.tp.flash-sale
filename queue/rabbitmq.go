package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"wahanapark/domain"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Nama exchange dan queue yang dipakai bersama oleh publisher dan consumer.
const (
	// Jalur notifikasi: fanout supaya mudah ditambah consumer baru di kemudian hari.
	NotifyExchange = "wahana.notify"
	NotifyQueue    = "wahana.notify.queue"

	// Jalur penerbitan tiket: pekerjaan berat dipindahkan dari alur permintaan HTTP.
	TicketExchange   = "wahana.ticket"
	TicketQueue      = "wahana.ticket.queue"
	TicketRoutingKey = "issue"

	// Jalur kedaluwarsa order memakai pola TTL dan Dead Letter Exchange.
	ExpiryExchange     = "wahana.expiry"
	ExpiryWaitQueue    = "wahana.expiry.wait"    // queue penunda, sengaja tanpa consumer
	DLX                = "wahana.dlx"            // menerima pesan yang sudah kedaluwarsa
	ExpiryProcessQueue = "wahana.expiry.process" // queue yang benar benar dikonsumsi worker
	ExpiryRoutingKey   = "expired"
)

// RabbitMQ membungkus koneksi AMQP beserta channel penerbit pesan.
type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	ttl     time.Duration
}

// Connect membuka koneksi dengan percobaan ulang, lalu mendeklarasikan seluruh topologi.
func Connect(url string, orderTTL time.Duration) (*RabbitMQ, error) {
	var conn *amqp.Connection
	var err error

	// Saat container start bersamaan, RabbitMQ biasanya belum siap menerima koneksi.
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
	log.Println("RabbitMQ terhubung dan topologi siap")
	return r, nil
}

func (r *RabbitMQ) declareTopology() error {
	ch := r.channel

	// Jalur notifikasi.
	if err := ch.ExchangeDeclare(NotifyExchange, "fanout", true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(NotifyQueue, true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind(NotifyQueue, "", NotifyExchange, false, nil); err != nil {
		return err
	}

	// Jalur penerbitan tiket.
	if err := ch.ExchangeDeclare(TicketExchange, "direct", true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(TicketQueue, true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind(TicketQueue, TicketRoutingKey, TicketExchange, false, nil); err != nil {
		return err
	}

	// Dead Letter Exchange beserta queue pemrosesnya.
	if err := ch.ExchangeDeclare(DLX, "direct", true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(ExpiryProcessQueue, true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind(ExpiryProcessQueue, ExpiryRoutingKey, DLX, false, nil); err != nil {
		return err
	}

	// Queue penunda ber-TTL. Pesan di sini tidak pernah dikonsumsi; begitu TTL habis
	// RabbitMQ memindahkannya sendiri ke DLX sehingga berfungsi sebagai penjadwal.
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
	return ch.QueueBind(ExpiryWaitQueue, ExpiryRoutingKey, ExpiryExchange, false, nil)
}

func (r *RabbitMQ) publish(ctx context.Context, exchange, key string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return r.channel.PublishWithContext(ctx, exchange, key, false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
	})
}

func (r *RabbitMQ) PublishNotify(ctx context.Context, msg domain.NotifyMessage) error {
	return r.publish(ctx, NotifyExchange, "", msg)
}

func (r *RabbitMQ) PublishTicketJob(ctx context.Context, msg domain.TicketJob) error {
	return r.publish(ctx, TicketExchange, TicketRoutingKey, msg)
}

func (r *RabbitMQ) PublishExpiry(ctx context.Context, msg domain.ExpiryMessage) error {
	return r.publish(ctx, ExpiryExchange, ExpiryRoutingKey, msg)
}

// Consume membuka channel terpisah untuk sebuah queue supaya pengaturan prefetch tiap
// consumer tidak saling memengaruhi.
func (r *RabbitMQ) Consume(queueName string, prefetch int) (<-chan amqp.Delivery, error) {
	ch, err := r.conn.Channel()
	if err != nil {
		return nil, err
	}
	if err := ch.Qos(prefetch, 0, false); err != nil {
		return nil, err
	}
	return ch.Consume(queueName, "", false, false, false, false, nil)
}

// QueueDepth melaporkan jumlah pesan yang masih menunggu pada sebuah queue.
func (r *RabbitMQ) QueueDepth(queueName string) (int, error) {
	ch, err := r.conn.Channel()
	if err != nil {
		return 0, err
	}
	defer ch.Close()

	q, err := ch.QueueInspect(queueName)
	if err != nil {
		return 0, err
	}
	return q.Messages, nil
}

func (r *RabbitMQ) Close() {
	if r.channel != nil {
		_ = r.channel.Close()
	}
	if r.conn != nil {
		_ = r.conn.Close()
	}
}
