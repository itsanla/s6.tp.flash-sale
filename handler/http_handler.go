package handler

import (
	"errors"
	"net/http"

	"flashsale/config"
	"flashsale/domain"
	"flashsale/usecase"

	"github.com/gin-gonic/gin"
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
		api.GET("/product", h.getProduct)
		api.POST("/checkout", h.checkout)
		api.POST("/orders/:id/pay", h.pay)
		api.POST("/orders/:id/cancel", h.cancel)
		api.GET("/orders/:id", h.getOrder)
		api.GET("/orders", h.listOrders)
	}
}

func (h *FlashSaleHandler) getProduct(c *gin.Context) {
	p, err := h.uc.GetProduct(c.Request.Context(), h.cfg.ProductID)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": p})
}

type checkoutRequest struct {
	Quantity int `json:"quantity"`
}

func (h *FlashSaleHandler) checkout(c *gin.Context) {
	var req checkoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "body tidak valid"})
		return
	}
	if req.Quantity == 0 {
		req.Quantity = 1
	}
	order, err := h.uc.Checkout(c.Request.Context(), h.cfg.ProductID, req.Quantity)
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

// respondErr memetakan error domain ke HTTP status yang sesuai.
func respondErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrOutOfStock):
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, domain.ErrProductNotFound), errors.Is(err, domain.ErrOrderNotFound):
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, domain.ErrOrderNotPending), errors.Is(err, domain.ErrOrderExpired), errors.Is(err, domain.ErrInvalidQuantity):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"success": false, "message": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
	}
}
