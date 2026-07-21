package handler

import (
	"net/http"
	"strconv"
	"time"

	"wahanapark/auth"
	"wahanapark/domain"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type adminLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// userSession menyusun jawaban standar setelah mendaftar atau masuk akun.
func (h *Handler) userSession(c *gin.Context, user *domain.User) {
	token, err := auth.GenerateToken(h.cfg.JWTSecret, strconv.FormatInt(user.ID, 10),
		auth.RoleUser, time.Duration(h.cfg.JWTExpiryHours)*time.Hour)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"token": token,
		"user":  user,
	}})
}

func (h *Handler) register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErr(c, domain.ErrInvalidInput)
		return
	}
	user, err := h.accounts.Register(c.Request.Context(), req.Name, req.Email, req.Phone, req.Password)
	if err != nil {
		respondErr(c, err)
		return
	}
	h.userSession(c, user)
}

func (h *Handler) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErr(c, domain.ErrInvalidInput)
		return
	}
	user, err := h.accounts.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		respondErr(c, err)
		return
	}
	h.userSession(c, user)
}

func (h *Handler) profile(c *gin.Context) {
	user, err := h.accounts.Profile(c.Request.Context(), auth.UserID(c))
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": user})
}

type updateProfileRequest struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

func (h *Handler) updateProfile(c *gin.Context) {
	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErr(c, domain.ErrInvalidInput)
		return
	}
	user, err := h.accounts.UpdateProfile(c.Request.Context(), auth.UserID(c), req.Name, req.Phone)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "profil diperbarui", "data": user})
}

func (h *Handler) myOrders(c *gin.Context) {
	orders, err := h.accounts.MyOrders(c.Request.Context(), auth.UserID(c), 50)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": orders})
}

// adminLogin memakai username dan password terpisah dari akun pengunjung.
func (h *Handler) adminLogin(c *gin.Context) {
	var req adminLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErr(c, domain.ErrInvalidInput)
		return
	}
	if req.Username != h.cfg.AdminUsername {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "username atau password salah"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(h.cfg.AdminPasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "username atau password salah"})
		return
	}
	token, err := auth.GenerateToken(h.cfg.JWTSecret, req.Username, auth.RoleAdmin,
		time.Duration(h.cfg.JWTExpiryHours)*time.Hour)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"token":            token,
		"expires_in_hours": h.cfg.JWTExpiryHours,
	}})
}
