package middleware

import (
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger mencatat setiap permintaan API beserta durasinya. Permintaan berkas statis
// tidak dicatat supaya log tetap mudah dibaca saat memantau alur pemesanan.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/") || path == "/health" {
			log.Printf("%s %s -> %d (%s)", c.Request.Method, path, c.Writer.Status(), time.Since(start))
		}
	}
}

// CORS mengizinkan akses lintas origin agar API mudah diuji dari luar aplikasi.
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
