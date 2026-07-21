package auth

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Peran yang dikenali sistem. Admin mengelola katalog wahana, sedangkan pengunjung
// memakai akunnya untuk melihat riwayat pemesanan dan tiket miliknya sendiri.
const (
	RoleAdmin = "ADMIN"
	RoleUser  = "USER"
)

// ContextUserID adalah kunci penyimpanan id pengunjung pada konteks permintaan.
const ContextUserID = "user_id"

var ErrInvalidToken = errors.New("sesi tidak valid atau sudah berakhir")

type claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken menerbitkan JWT bertanda tangan HS256 untuk sebuah peran.
// Subject berisi username untuk admin, atau id akun untuk pengunjung.
func GenerateToken(secret, subject, role string, expiry time.Duration) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
		},
	})
	return token.SignedString([]byte(secret))
}

func parseToken(secret, tokenString, wantRole string) (*claims, error) {
	parsed, err := jwt.ParseWithClaims(tokenString, &claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(secret), nil
	})
	if err != nil || !parsed.Valid {
		return nil, ErrInvalidToken
	}
	c, ok := parsed.Claims.(*claims)
	if !ok || c.Role != wantRole {
		return nil, ErrInvalidToken
	}
	return c, nil
}

func bearer(c *gin.Context) string {
	header := c.GetHeader("Authorization")
	token := strings.TrimPrefix(header, "Bearer ")
	if token == header {
		return ""
	}
	return token
}

// RequireAdmin menolak permintaan yang tidak menyertakan token admin yang sah.
func RequireAdmin(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearer(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": "login admin diperlukan"})
			return
		}
		if _, err := parseToken(secret, token, RoleAdmin); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": ErrInvalidToken.Error()})
			return
		}
		c.Next()
	}
}

// RequireUser menolak permintaan tanpa token pengunjung yang sah, lalu menyimpan id
// akun pada konteks agar handler tahu data siapa yang boleh dibaca.
func RequireUser(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearer(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": "silakan masuk ke akun Anda terlebih dahulu"})
			return
		}
		claims, err := parseToken(secret, token, RoleUser)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": ErrInvalidToken.Error()})
			return
		}
		id, err := strconv.ParseInt(claims.Subject, 10, 64)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": ErrInvalidToken.Error()})
			return
		}
		c.Set(ContextUserID, id)
		c.Next()
	}
}

// OptionalUser membaca token pengunjung bila ada, tanpa pernah menolak permintaan.
// Dipakai pada checkout supaya pemesanan tetap bisa dilakukan tanpa masuk akun,
// tetapi otomatis tercatat pada akun bila pengunjung sedang masuk.
func OptionalUser(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if token := bearer(c); token != "" {
			if claims, err := parseToken(secret, token, RoleUser); err == nil {
				if id, err := strconv.ParseInt(claims.Subject, 10, 64); err == nil {
					c.Set(ContextUserID, id)
				}
			}
		}
		c.Next()
	}
}

// UserID membaca id pengunjung dari konteks. Bernilai nol bila tidak sedang masuk.
func UserID(c *gin.Context) int64 {
	if v, ok := c.Get(ContextUserID); ok {
		if id, ok := v.(int64); ok {
			return id
		}
	}
	return 0
}
