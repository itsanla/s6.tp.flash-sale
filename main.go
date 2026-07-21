package main

import (
	"context"
	"io/fs"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"wahanapark/config"
	"wahanapark/handler"
	"wahanapark/middleware"
	"wahanapark/qris"
	"wahanapark/queue"
	"wahanapark/repository/rediscache"
	"wahanapark/repository/sqlite"
	"wahanapark/usecase"
	"wahanapark/web"
	"wahanapark/worker"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg := config.Load()
	log.Printf("Taman Wahana Nusantara | mode=%s env=%s", cfg.AppMode, cfg.AppEnv)

	// Password admin di-hash sekali saat startup supaya runtime tidak pernah
	// membandingkan password dalam bentuk teks biasa.
	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Gagal menyiapkan kredensial admin: %v", err)
	}
	cfg.AdminPasswordHash = string(hash)

	// --- SQLite: sumber kebenaran data wahana, order, dan tiket ---
	db, err := sqlite.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Gagal menyiapkan SQLite: %v", err)
	}
	defer db.Close()
	log.Printf("SQLite siap pada %s", cfg.DatabasePath)

	rideRepo := sqlite.NewRideRepository(db)
	orderRepo := sqlite.NewOrderRepository(db)
	ticketRepo := sqlite.NewTicketRepository(db)

	// --- Redis: kuota atomik dan cache ---
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	pingCtx, cancelPing := context.WithTimeout(context.Background(), 5*time.Second)
	if err := rdb.Ping(pingCtx).Err(); err != nil {
		cancelPing()
		log.Fatalf("Gagal terhubung ke Redis pada %s: %v", cfg.RedisAddr, err)
	}
	cancelPing()
	defer rdb.Close()
	log.Printf("Redis terhubung pada %s", cfg.RedisAddr)

	quotaStore := rediscache.NewQuotaStore(rdb, cfg.QuotaTTLDays)
	cache := rediscache.NewCache(rdb)

	// --- RabbitMQ: pemrosesan asinkron ---
	paymentTTL := time.Duration(cfg.PaymentTTLMinutes) * time.Minute
	mq, err := queue.Connect(cfg.RabbitMQURL, paymentTTL)
	if err != nil {
		log.Fatalf("Gagal terhubung ke RabbitMQ: %v", err)
	}
	defer mq.Close()

	qrisGen := qris.NewGenerator(cfg.MerchantName, cfg.MerchantCity, cfg.MerchantID)

	catalogUC := usecase.NewCatalogUsecase(rideRepo, quotaStore, cache,
		time.Duration(cfg.CacheTTLSeconds)*time.Second)
	orderUC := usecase.NewOrderUsecase(orderRepo, rideRepo, ticketRepo, quotaStore,
		cache, mq, qrisGen, paymentTTL)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if cfg.AppMode == "worker" || cfg.AppMode == "all" {
		w := worker.New(mq, orderUC)
		if err := w.Start(ctx); err != nil {
			log.Fatalf("Gagal menjalankan worker: %v", err)
		}
	}

	var srv *http.Server
	if cfg.AppMode == "server" || cfg.AppMode == "all" {
		srv = startServer(cfg, catalogUC, orderUC, mq, qrisGen)
	}
	if cfg.AppMode == "worker" {
		log.Println("Mode worker: menunggu pesan dari antrean")
	}

	<-ctx.Done()
	log.Println("Sinyal shutdown diterima, mematikan aplikasi dengan rapi")

	if srv != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server gagal dimatikan dengan rapi: %v", err)
		}
	}
	log.Println("Selesai.")
}

func startServer(
	cfg *config.Config,
	catalogUC *usecase.CatalogUsecase,
	orderUC *usecase.OrderUsecase,
	mq *queue.RabbitMQ,
	qrisGen *qris.Generator,
) *http.Server {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(middleware.Logger(), middleware.CORS(), gin.Recovery())

	// Hasil build React ikut tertanam di dalam binary, sehingga satu container sudah
	// berisi antarmuka, backend, dan basis data sekaligus.
	webFS, err := fs.Sub(web.DistFS, "dist")
	if err != nil {
		log.Fatalf("Gagal membaca berkas antarmuka: %v", err)
	}

	handler.Register(r, cfg, catalogUC, orderUC, mq, qrisGen, webFS)

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: r}
	go func() {
		log.Printf("Server berjalan di port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server gagal berjalan: %v", err)
		}
	}()
	return srv
}
