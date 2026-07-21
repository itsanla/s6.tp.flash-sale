package config

import (
	"os"
	"strconv"
)

// Config menampung seluruh konfigurasi aplikasi yang dibaca dari environment variable.
type Config struct {
	// HTTP
	Port    string
	AppEnv  string
	AppMode string // "server", "worker", atau "all"

	// SQLite (sumber kebenaran data: wahana, order, tiket)
	DatabasePath string

	// Redis (kuota atomik + cache)
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// RabbitMQ (pemrosesan asinkron)
	RabbitMQURL string

	// Aturan bisnis
	PaymentTTLMinutes int // batas waktu bayar QRIS sebelum order kedaluwarsa
	QuotaTTLDays      int // berapa lama kunci kuota harian disimpan di Redis
	CacheTTLSeconds   int // masa berlaku cache katalog wahana

	// Merchant QRIS (simulasi, bukan merchant sungguhan)
	MerchantName string
	MerchantCity string
	MerchantID   string

	// Admin
	AdminUsername     string
	AdminPassword     string
	AdminPasswordHash string // diisi saat startup hasil bcrypt
	JWTSecret         string
	JWTExpiryHours    int
}

// Load membaca konfigurasi dari environment variable dengan default yang aman.
func Load() *Config {
	return &Config{
		Port:    getEnv("PORT", "8080"),
		AppEnv:  getEnv("APP_ENV", "development"),
		AppMode: getEnv("APP_MODE", "all"),

		DatabasePath: getEnv("DATABASE_PATH", "data/wahana.db"),

		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),

		RabbitMQURL: getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),

		PaymentTTLMinutes: getEnvInt("PAYMENT_TTL_MINUTES", 10),
		QuotaTTLDays:      getEnvInt("QUOTA_TTL_DAYS", 45),
		CacheTTLSeconds:   getEnvInt("CACHE_TTL_SECONDS", 60),

		MerchantName: getEnv("MERCHANT_NAME", "TAMAN WAHANA SIMULASI"),
		MerchantCity: getEnv("MERCHANT_CITY", "PADANG"),
		MerchantID:   getEnv("MERCHANT_ID", "936000091100000001"),

		AdminUsername:  getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:  getEnv("ADMIN_PASSWORD", "admin123"),
		JWTSecret:      getEnv("JWT_SECRET", "wahana-dev-secret-change-me"),
		JWTExpiryHours: getEnvInt("JWT_EXPIRY_HOURS", 4),
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
