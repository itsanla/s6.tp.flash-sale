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

	// Load test (demo pembuktian throughput RabbitMQ + Redis di bawah beban tinggi)
	LoadTestMaxQuantity int // batas aman jumlah pesanan per batch
	LoadTestConcurrency int // jumlah worker paralel yang memproses antrean bulk
	LoadTestDelayMs     int // simulasi waktu proses per pesanan (ms)

	// Admin (login + RBAC minimal): satu akun tetap, tanpa tabel user.
	AdminUsername  string
	AdminPassword  string // dibaca sekali saat startup, langsung di-hash (bcrypt) — lihat main.go
	AdminPasswordHash string // diisi main.go setelah bcrypt.GenerateFromPassword
	JWTSecret      string
	JWTExpiryHours int
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

		LoadTestMaxQuantity: getEnvInt("LOADTEST_MAX_QUANTITY", 50000),
		LoadTestConcurrency: getEnvInt("LOADTEST_CONCURRENCY", 20),
		LoadTestDelayMs:     getEnvInt("LOADTEST_DELAY_MS", 15),

		AdminUsername:  getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:  getEnv("ADMIN_PASSWORD", "admin123"),
		JWTSecret:      getEnv("JWT_SECRET", "flashsale-dev-secret-change-me"),
		JWTExpiryHours: getEnvInt("JWT_EXPIRY_HOURS", 2),
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
