package handler

import (
	"net/http"
	"strconv"

	"wahanapark/domain"

	"github.com/gin-gonic/gin"
)

func (h *Handler) listCategories(c *gin.Context) {
	cats, err := h.catalog.Categories(c.Request.Context())
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": cats})
}

// listRides melayani katalog wahana. Parameter tanggal dipakai untuk menghitung sisa
// kuota pada hari kunjungan yang dipilih pengunjung.
func (h *Handler) listRides(c *gin.Context) {
	category := c.Query("category")
	date := c.Query("date")
	rides, err := h.catalog.List(c.Request.Context(), category, date, true)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": rides})
}

func (h *Handler) getRide(c *gin.Context) {
	ride, err := h.catalog.GetBySlug(c.Request.Context(), c.Param("slug"), c.Query("date"))
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": ride})
}

type rideRequest struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Tagline     string `json:"tagline"`
	Description string `json:"description"`
	Emoji       string `json:"emoji"`
	Price       int64  `json:"price"`
	DurationMin int    `json:"duration_min"`
	MinHeightCm int    `json:"min_height_cm"`
	ThrillLevel int    `json:"thrill_level"`
	DailyQuota  int    `json:"daily_quota"`
	IsActive    bool   `json:"is_active"`
}

func (r rideRequest) toDomain() *domain.Ride {
	return &domain.Ride{
		Slug: r.Slug, Name: r.Name, Category: r.Category, Tagline: r.Tagline,
		Description: r.Description, Emoji: r.Emoji, Price: r.Price,
		DurationMin: r.DurationMin, MinHeightCm: r.MinHeightCm,
		ThrillLevel: r.ThrillLevel, DailyQuota: r.DailyQuota, IsActive: r.IsActive,
	}
}

func (h *Handler) createRide(c *gin.Context) {
	var req rideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErr(c, domain.ErrInvalidInput)
		return
	}
	ride := req.toDomain()
	if err := h.catalog.Create(c.Request.Context(), ride); err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "message": "wahana ditambahkan", "data": ride})
}

func (h *Handler) updateRide(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respondErr(c, domain.ErrInvalidInput)
		return
	}
	var req rideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErr(c, domain.ErrInvalidInput)
		return
	}
	ride := req.toDomain()
	ride.ID = id
	if err := h.catalog.Update(c.Request.Context(), ride); err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "wahana diperbarui", "data": ride})
}

func (h *Handler) deleteRide(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respondErr(c, domain.ErrInvalidInput)
		return
	}
	if err := h.catalog.Delete(c.Request.Context(), id); err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "wahana dihapus"})
}
