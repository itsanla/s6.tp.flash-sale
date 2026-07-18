package handler

import (
	"errors"
	"net/http"
	"time"

	"flashsale/auth"
	"flashsale/config"
	"flashsale/domain"
	"flashsale/usecase"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type FlashSaleHandler struct {
	uc  *usecase.FlashSaleUsecase
	cfg *config.Config
}

// Register memasang seluruh route API di bawah /api/v1.
func Register(r *gin.Engine, uc *usecase.FlashSaleUsecase, cfg *config.Config) {
	h := &FlashSaleHandler{uc: uc, cfg: cfg}

	api := r.Group("/api/v1")
	{
		api.POST("/auth/login", h.login)

		api.GET("/products", h.listProducts)
		api.GET("/products/:id", h.getProduct)

		api.POST("/checkout", h.checkout)
		api.POST("/orders/:id/pay", h.pay)
		api.POST("/orders/:id/cancel", h.cancel)
		api.GET("/orders/:id", h.getOrder)
		api.GET("/orders", h.listOrders)

		api.POST("/loadtest", h.startLoadTest)
		api.GET("/loadtest/:batch_id", h.getLoadTestStatus)

		// Kelola produk (Admin) — satu-satunya endpoint yang butuh login.
		admin := api.Group("/products")
		admin.Use(auth.RequireAdmin(cfg.JWTSecret))
		{
			admin.POST("", h.createProduct)
			admin.PUT("/:id", h.updateProduct)
			admin.DELETE("/:id", h.deleteProduct)
		}
	}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// login memvalidasi kredensial admin tunggal (bcrypt) lalu menerbitkan JWT.
func (h *FlashSaleHandler) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "body tidak valid"})
		return
	}
	if req.Username != h.cfg.AdminUsername {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": domain.ErrInvalidCredentials.Error()})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(h.cfg.AdminPasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": domain.ErrInvalidCredentials.Error()})
		return
	}
	token, err := auth.GenerateToken(h.cfg.JWTSecret, req.Username, time.Duration(h.cfg.JWTExpiryHours)*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"token": token, "expires_in_hours": h.cfg.JWTExpiryHours}})
}

func (h *FlashSaleHandler) listProducts(c *gin.Context) {
	products, err := h.uc.ListProducts(c.Request.Context())
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": products})
}

func (h *FlashSaleHandler) getProduct(c *gin.Context) {
	p, err := h.uc.GetProduct(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": p})
}

type createProductRequest struct {
	Name  string `json:"name"`
	Stock int64  `json:"stock"`
}

func (h *FlashSaleHandler) createProduct(c *gin.Context) {
	var req createProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "body tidak valid"})
		return
	}
	p, err := h.uc.CreateProduct(c.Request.Context(), req.Name, req.Stock)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "message": "produk ditambahkan", "data": p})
}

type updateProductRequest struct {
	Name  string `json:"name"`
	Stock *int64 `json:"stock"`
}

func (h *FlashSaleHandler) updateProduct(c *gin.Context) {
	var req updateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "body tidak valid"})
		return
	}
	if err := h.uc.UpdateProduct(c.Request.Context(), c.Param("id"), req.Name, req.Stock); err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "produk diperbarui"})
}

func (h *FlashSaleHandler) deleteProduct(c *gin.Context) {
	if err := h.uc.DeleteProduct(c.Request.Context(), c.Param("id")); err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "produk dihapus"})
}

type checkoutRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

func (h *FlashSaleHandler) checkout(c *gin.Context) {
	var req checkoutRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.ProductID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "product_id wajib diisi"})
		return
	}
	if req.Quantity == 0 {
		req.Quantity = 1
	}
	order, err := h.uc.Checkout(c.Request.Context(), req.ProductID, req.Quantity)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "order dibuat, silakan bayar sebelum kedaluwarsa",
		"data":    order,
	})
}

func (h *FlashSaleHandler) pay(c *gin.Context) {
	order, err := h.uc.Pay(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "pembayaran sukses, tiket terbit", "data": order})
}

func (h *FlashSaleHandler) cancel(c *gin.Context) {
	order, err := h.uc.Cancel(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "order dibatalkan, stok dikembalikan", "data": order})
}

func (h *FlashSaleHandler) getOrder(c *gin.Context) {
	order, err := h.uc.GetOrder(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": order})
}

func (h *FlashSaleHandler) listOrders(c *gin.Context) {
	orders, err := h.uc.ListOrders(c.Request.Context(), 50)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": orders})
}

type loadTestRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int64  `json:"quantity"`
}

// startLoadTest menerima permintaan uji beban dan langsung merespons sukses;
// pemrosesan aktual (reservasi stok + order + notifikasi) berjalan asinkron
// lewat RabbitMQ, progresnya dipantau via GET /loadtest/:batch_id.
func (h *FlashSaleHandler) startLoadTest(c *gin.Context) {
	var req loadTestRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Quantity <= 0 || req.ProductID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "product_id wajib diisi dan quantity lebih dari 0"})
		return
	}
	batchID, err := h.uc.StartBulkLoadTest(c.Request.Context(), req.ProductID, req.Quantity)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{
		"success":  true,
		"message":  "batch diterima, diproses asinkron di belakang layar via RabbitMQ",
		"batch_id": batchID,
	})
}

func (h *FlashSaleHandler) getLoadTestStatus(c *gin.Context) {
	status, err := h.uc.GetBatch(c.Request.Context(), c.Param("batch_id"))
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": status})
}

// respondErr memetakan error domain ke HTTP status yang sesuai.
func respondErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrOutOfStock):
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, domain.ErrProductNotFound), errors.Is(err, domain.ErrOrderNotFound), errors.Is(err, domain.ErrBatchNotFound):
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, domain.ErrProductExists), errors.Is(err, domain.ErrProductHasOrders):
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, domain.ErrOrderNotPending), errors.Is(err, domain.ErrOrderExpired), errors.Is(err, domain.ErrInvalidQuantity), errors.Is(err, domain.ErrBatchTooLarge):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"success": false, "message": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
	}
}
