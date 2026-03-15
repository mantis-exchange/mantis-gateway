package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeaders adds security-related HTTP headers.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Next()
	}
}

// CORS handles Cross-Origin Resource Sharing.
func CORS(allowedOrigins string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}

		// In production, validate against allowedOrigins list.
		// For development, allow the configured origins.
		c.Header("Access-Control-Allow-Origin", allowedOrigins)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-API-Key")
		c.Header("Access-Control-Max-Age", "86400")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
