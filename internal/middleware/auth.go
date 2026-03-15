package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Auth returns a Gin middleware that performs JWT authentication.
//
// This is a placeholder implementation. It checks that the Authorization header
// contains a Bearer token and sets a dummy user ID on the context. In production
// this will validate the JWT signature and claims using the configured secret.
func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing authorization header",
			})
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization header format, expected: Bearer <token>",
			})
			return
		}

		token := parts[1]
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "empty token",
			})
			return
		}

		// TODO: Validate JWT signature and claims using jwtSecret.
		// For now accept any non-empty token and set a placeholder user ID.
		_ = jwtSecret

		c.Set("user_id", "user-placeholder-123")
		c.Next()
	}
}
