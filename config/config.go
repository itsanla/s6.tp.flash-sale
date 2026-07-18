package config

import (
	"os"
	"strconv"
)

// Config menampung seluruh konfigurasi aplikasi yang dibaca dari environment variable.
type Config struct {
	// HTTP
	Port   string
	AppEnv string
	// Mode: "server" (HTTP + UI), "worker" (consumer RabbitMQ), atau "all" (keduanya)
	AppMode string

	// Redis (topik m1 — in-memory store & atomic counter)
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// RabbitMQ (topik m2 — message queue)
	RabbitMQURL string

	// Flash sale
	OrderTTLSeconds int    // batas waktu bayar sebelum order auto-expire
	ProductID       string // id produk yang dijual saat flash sale
	ProductName     string
	ProductStock    int // stok awal yang di-seed saat startup
}

// Load membaca konfigurasi dari environment variable dengan nilai default yang aman.
func Load() *Config {
	return &Config{
		Port:    getEnv("PORT", "8080"),
		AppEnv:  getEnv("APP_ENV", "development"),
		AppMode: getEnv("APP_MODE", "all"),

		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),

		RabbitMQURL: getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),

		OrderTTLSeconds: getEnvInt("ORDER_TTL_SECONDS", 60),
		ProductID:       getEnv("PRODUCT_ID", "TICKET-EVENTHUB-2026"),
		ProductName:     getEnv("PRODUCT_NAME", "Tiket Flash Sale EventHub 2026"),
		ProductStock:    getEnvInt("PRODUCT_STOCK", 20),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
