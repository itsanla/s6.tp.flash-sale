package handler

import (
	"errors"
	"io/fs"
	"net/http"
	"strings"

	"wahanapark/auth"
	"wahanapark/config"
	"wahanapark/domain"
	"wahanapark/qris"
	"wahanapark/queue"
	"wahanapark/usecase"

	"github.com/gin-gonic/gin"
)

// Handler menampung seluruh dependensi yang dipakai lapisan HTTP.
type Handler struct {
	cfg      *config.Config
	catalog  *usecase.CatalogUsecase
	orders   *usecase.OrderUsecase
	accounts *usecase.AccountUsecase
	mq       *queue.RabbitMQ
	qrisGen  *qris.Generator
}

// Register memasang seluruh rute API dan penyajian aplikasi React.
func Register(
	r *gin.Engine,
	cfg *config.Config,
	catalog *usecase.CatalogUsecase,
	orders *usecase.OrderUsecase,
	accounts *usecase.AccountUsecase,
	mq *queue.RabbitMQ,
	qrisGen *qris.Generator,
	webFS fs.FS,
) {
	h := &Handler{cfg: cfg, catalog: catalog, orders: orders, accounts: accounts, mq: mq, qrisGen: qrisGen}

	r.GET("/health", h.health)

	api := r.Group("/api/v1")
	{
		api.GET("/config", h.appConfig)
		api.GET("/categories", h.listCategories)
		api.GET("/rides", h.listRides)
		api.GET("/rides/:slug", h.getRide)

		// Checkout memakai auth opsional: tanpa masuk akun tetap bisa memesan, tetapi
		// bila sedang masuk, pesanan otomatis tercatat pada akun pengunjung.
		api.POST("/orders", auth.OptionalUser(cfg.JWTSecret), h.checkout)
		api.GET("/orders/:code", h.getOrder)
		api.POST("/orders/:code/cancel", h.cancelOrder)
		api.GET("/orders/:code/tickets", h.getTickets)

		api.POST("/tickets/:code/scan", h.scanTicket)

		// Endpoint uji pembayaran. Menggantikan notifikasi dari penyedia pembayaran
		// sungguhan, dipakai oleh halaman /test/qris-list.
		test := api.Group("/test")
		{
			test.GET("/pending-orders", h.listPendingOrders)
			test.POST("/orders/:code/settle", h.settlePayment)
			test.GET("/system", h.systemStatus)
		}

		// Akun pengunjung
		api.POST("/auth/register", h.register)
		api.POST("/auth/login", h.login)
		api.POST("/auth/admin", h.adminLogin)

		me := api.Group("/me")
		me.Use(auth.RequireUser(cfg.JWTSecret))
		{
			me.GET("", h.profile)
			me.PUT("", h.updateProfile)
			me.GET("/orders", h.myOrders)
		}

		admin := api.Group("/admin")
		admin.Use(auth.RequireAdmin(cfg.JWTSecret))
		{
			admin.GET("/stats", h.adminStats)
			admin.GET("/orders", h.adminOrders)
			admin.POST("/rides", h.createRide)
			admin.PUT("/rides/:id", h.updateRide)
			admin.DELETE("/rides/:id", h.deleteRide)
		}
	}

	registerSPA(r, webFS)
}

// registerSPA menyajikan hasil build React. Berkas statis dilayani langsung, sedangkan
// rute lain dikembalikan ke index.html agar navigasi sisi klien tetap berfungsi saat
// halaman dimuat ulang atau dibuka langsung lewat tautan.
func registerSPA(r *gin.Engine, webFS fs.FS) {
	indexBytes, err := fs.ReadFile(webFS, "index.html")
	if err != nil {
		indexBytes = []byte("<h1>Antarmuka belum tersedia</h1><p>Jalankan build frontend terlebih dahulu.</p>")
	}
	fileServer := http.FileServer(http.FS(webFS))

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "endpoint tidak ditemukan"})
			return
		}
		clean := strings.TrimPrefix(path, "/")
		if clean != "" {
			if f, err := webFS.Open(clean); err == nil {
				f.Close()
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexBytes)
	})
}

func (h *Handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "taman-wahana",
		"mode":    h.cfg.AppMode,
	})
}

func (h *Handler) appConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"payment_ttl_minutes": h.cfg.PaymentTTLMinutes,
		"merchant_name":       h.cfg.MerchantName,
		"categories":          domain.CategoryLabels,
	})
}

// respondErr memetakan error domain ke kode status HTTP yang sesuai.
func respondErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrQuotaNotEnough):
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, domain.ErrRideHasOrders), errors.Is(err, domain.ErrTicketUsed):
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, domain.ErrEmailTaken):
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, domain.ErrInvalidCredentials):
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, domain.ErrRideNotFound), errors.Is(err, domain.ErrOrderNotFound),
		errors.Is(err, domain.ErrTicketNotFound), errors.Is(err, domain.ErrUserNotFound):
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, domain.ErrOrderNotPending), errors.Is(err, domain.ErrOrderExpired),
		errors.Is(err, domain.ErrInvalidInput), errors.Is(err, domain.ErrRideInactive),
		errors.Is(err, domain.ErrPasswordTooShort):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"success": false, "message": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
	}
}
