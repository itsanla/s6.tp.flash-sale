package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// RoleAdmin adalah satu-satunya role terproteksi pada Flash Sale Mini — sistem
// ini sengaja tidak membangun tabel user/RBAC penuh (di luar scope demo),
// cukup satu akun Admin tetap yang dikonfigurasi lewat environment variable.
const RoleAdmin = "ADMIN"

var ErrInvalidToken = errors.New("token tidak valid atau kedaluwarsa")

type claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken menerbitkan JWT (HS256) untuk akun admin, berlaku selama expiry.
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

// RequireAdmin adalah middleware Gin yang memvalidasi header
// "Authorization: Bearer <token>" dan menolak akses (401) bila token tidak
// ada/tidak valid/kedaluwarsa/bukan role ADMIN.
func RequireAdmin(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		tokenString := strings.TrimPrefix(header, "Bearer ")
		if tokenString == "" || tokenString == header {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": "autentikasi admin diperlukan"})
			return
		}
		if _, err := parseToken(secret, tokenString); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": ErrInvalidToken.Error()})
			return
		}
		c.Next()
	}
}
