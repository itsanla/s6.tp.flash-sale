package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"flashsale/config"
	"flashsale/domain"
	"flashsale/handler"
	"flashsale/middleware"
	"flashsale/queue"
	"flashsale/repository"
	"flashsale/usecase"
	"flashsale/worker"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.Load()
	log.Printf("Flash Sale Mini | mode=%s env=%s", cfg.AppMode, cfg.AppEnv)

	// --- Redis (m1) ---
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	if err := pingRedis(rdb); err != nil {
		log.Fatalf("Gagal terhubung ke Redis pada %s: %v", cfg.RedisAddr, err)
	}
	log.Printf("Redis terhubung pada %s", cfg.RedisAddr)

	stockRepo := repository.NewRedisStockRepository(rdb)

	// Seed produk flash sale (hanya jika belum ada).
	seedCtx, cancelSeed := context.WithTimeout(context.Background(), 5*time.Second)
	if err := stockRepo.SeedProduct(seedCtx, domain.Product{
		ID:    cfg.ProductID,
		Name:  cfg.ProductName,
		Stock: int64(cfg.ProductStock),
	}); err != nil {
		log.Fatalf("Gagal seed produk: %v", err)
	}
	cancelSeed()
	log.Printf("Produk siap: %s (stok awal %d)", cfg.ProductName, cfg.ProductStock)

	// --- RabbitMQ (m2) ---
	mq, err := queue.Connect(cfg.RabbitMQURL, time.Duration(cfg.OrderTTLSeconds)*time.Second)
	if err != nil {
		log.Fatalf("Gagal terhubung ke RabbitMQ: %v", err)
	}
	defer mq.Close()

	uc := usecase.NewFlashSaleUsecase(stockRepo, mq, time.Duration(cfg.OrderTTLSeconds)*time.Second)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// --- Worker (consumer) ---
	if cfg.AppMode == "worker" || cfg.AppMode == "all" {
		w := worker.New(mq, uc)
		if err := w.Start(ctx); err != nil {
			log.Fatalf("Gagal menjalankan worker: %v", err)
		}
	}

	// --- HTTP server + UI ---
	var srv *http.Server
	if cfg.AppMode == "server" || cfg.AppMode == "all" {
		srv = startServer(cfg, uc, rdb)
	}

	if cfg.AppMode == "worker" {
		log.Println("Mode worker: menunggu pesan...")
	}

	<-ctx.Done()
	log.Println("Sinyal shutdown diterima, mematikan dengan graceful...")

	if srv != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server gagal shutdown: %v", err)
		}
	}
	_ = rdb.Close()
	log.Println("Selesai.")
}

func startServer(cfg *config.Config, uc *usecase.FlashSaleUsecase, rdb *redis.Client) *http.Server {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(middleware.Logger(), middleware.CORS(), gin.Recovery())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "flash-sale", "mode": cfg.AppMode})
	})

	// Beri UI info produk mana yang aktif.
	r.GET("/api/v1/config", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"product_id":        cfg.ProductID,
			"order_ttl_seconds": cfg.OrderTTLSeconds,
		})
	})

	handler.Register(r, uc, cfg)

	// UI statis (tanpa framework) dilayani dari folder web/.
	r.StaticFile("/", "./web/index.html")
	r.Static("/static", "./web")

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: r}
	go func() {
		log.Printf("HTTP server berjalan di port %s — buka http://localhost:%s", cfg.Port, cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server gagal berjalan: %v", err)
		}
	}()
	return srv
}

func pingRedis(rdb *redis.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return rdb.Ping(ctx).Err()
}
