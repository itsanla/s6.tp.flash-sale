package handler

import (
	"net/http"
	"time"

	"wahanapark/auth"
	"wahanapark/domain"
	"wahanapark/qris"
	"wahanapark/queue"
	"wahanapark/usecase"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// orderResponse membungkus order beserta gambar QR yang siap ditampilkan.
func (h *Handler) orderResponse(o *domain.Order) gin.H {
	data := gin.H{"order": o}
	if o.Status == domain.StatusPending && o.QRISPayload != "" {
		if img, err := qris.RenderPNGBase64(o.QRISPayload, 420); err == nil {
			data["qris_image"] = img
		}
	}
	data["seconds_left"] = int64(time.Until(o.ExpiresAt).Seconds())
	return data
}

func (h *Handler) checkout(c *gin.Context) {
	var req usecase.CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErr(c, domain.ErrInvalidInput)
		return
	}
	order, err := h.orders.Checkout(c.Request.Context(), req)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "order dibuat, silakan selesaikan pembayaran QRIS",
		"data":    h.orderResponse(order),
	})
}

func (h *Handler) getOrder(c *gin.Context) {
	order, err := h.orders.GetByCode(c.Request.Context(), c.Param("code"))
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": h.orderResponse(order)})
}

func (h *Handler) cancelOrder(c *gin.Context) {
	order, err := h.orders.Cancel(c.Request.Context(), c.Param("code"))
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "order dibatalkan", "data": gin.H{"order": order}})
}

func (h *Handler) getTickets(c *gin.Context) {
	tickets, err := h.orders.Tickets(c.Request.Context(), c.Param("code"))
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": tickets})
}

func (h *Handler) scanTicket(c *gin.Context) {
	ticket, err := h.orders.ScanTicket(c.Request.Context(), c.Param("code"))
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "tiket berhasil diverifikasi", "data": ticket})
}

// listPendingOrders dipakai halaman uji untuk menampilkan seluruh order yang menunggu
// pembayaran beserta gambar QRIS masing masing.
func (h *Handler) listPendingOrders(c *gin.Context) {
	orders, err := h.orders.ListPending(c.Request.Context(), 50)
	if err != nil {
		respondErr(c, err)
		return
	}
	out := make([]gin.H, 0, len(orders))
	for i := range orders {
		o := orders[i]
		item := gin.H{"order": o, "seconds_left": int64(time.Until(o.ExpiresAt).Seconds())}
		if img, err := qris.RenderPNGBase64(o.QRISPayload, 200); err == nil {
			item["qris_image"] = img
		}
		out = append(out, item)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": out})
}

// settlePayment melunasi pembayaran sebuah order. Endpoint ini adalah pengganti
// notifikasi dari penyedia pembayaran, khusus untuk keperluan pengujian dan demonstrasi.
func (h *Handler) settlePayment(c *gin.Context) {
	order, err := h.orders.SettlePayment(c.Request.Context(), c.Param("code"))
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "pembayaran berhasil, tiket sedang diterbitkan lewat antrean",
		"data":    gin.H{"order": order},
	})
}

// systemStatus melaporkan kondisi antrean RabbitMQ supaya proses asinkron terlihat nyata
// saat aplikasi didemonstrasikan.
func (h *Handler) systemStatus(c *gin.Context) {
	depth := func(name string) int {
		n, err := h.mq.QueueDepth(name)
		if err != nil {
			return -1
		}
		return n
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"queues": []gin.H{
			{"name": "Notifikasi", "queue": queue.NotifyQueue, "messages": depth(queue.NotifyQueue)},
			{"name": "Penerbitan tiket", "queue": queue.TicketQueue, "messages": depth(queue.TicketQueue)},
			{"name": "Penunda kedaluwarsa", "queue": queue.ExpiryWaitQueue, "messages": depth(queue.ExpiryWaitQueue)},
			{"name": "Pemroses kedaluwarsa", "queue": queue.ExpiryProcessQueue, "messages": depth(queue.ExpiryProcessQueue)},
		},
	}})
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErr(c, domain.ErrInvalidInput)
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
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"token":            token,
		"expires_in_hours": h.cfg.JWTExpiryHours,
	}})
}

func (h *Handler) adminStats(c *gin.Context) {
	stats, err := h.orders.Stats(c.Request.Context(), time.Duration(h.cfg.CacheTTLSeconds)*time.Second)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": stats})
}

func (h *Handler) adminOrders(c *gin.Context) {
	orders, err := h.orders.ListRecent(c.Request.Context(), 50)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": orders})
}
