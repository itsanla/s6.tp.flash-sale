package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// RoleAdmin adalah satu satunya peran terproteksi pada aplikasi ini. Sistem sengaja
// tidak membangun tabel pengguna penuh karena hanya butuh satu akun pengelola taman.
const RoleAdmin = "ADMIN"

var ErrInvalidToken = errors.New("sesi admin tidak valid atau sudah berakhir")

type claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken menerbitkan JWT bertanda tangan HS256 untuk akun admin.
func GenerateToken(secret, username string, expiry time.Duration) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		Role: RoleAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
		},
	})
	return token.SignedString([]byte(secret))
}

func parseToken(secret, tokenString string) (*claims, error) {
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
	if !ok || c.Role != RoleAdmin {
		return nil, ErrInvalidToken
	}
	return c, nil
}

// RequireAdmin menolak permintaan yang tidak menyertakan token admin yang sah.
func RequireAdmin(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		tokenString := strings.TrimPrefix(header, "Bearer ")
		if tokenString == "" || tokenString == header {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": "login admin diperlukan"})
			return
		}
		if _, err := parseToken(secret, tokenString); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": ErrInvalidToken.Error()})
			return
		}
		c.Next()
	}
}
