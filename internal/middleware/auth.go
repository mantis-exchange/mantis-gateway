package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Auth returns middleware that validates JWT tokens or API key signatures.
func Auth(jwtSecret string, accountServiceAddr string) gin.HandlerFunc {
	secretBytes := []byte(jwtSecret)

	return func(c *gin.Context) {
		// Try JWT auth first
		if header := c.GetHeader("Authorization"); header != "" {
			parts := strings.SplitN(header, " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
				tokenStr := parts[1]
				token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
					if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
						return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
					}
					return secretBytes, nil
				})
				if err == nil && token.Valid {
					if claims, ok := token.Claims.(jwt.MapClaims); ok {
						if sub, ok := claims["sub"].(string); ok && sub != "" {
							c.Set("user_id", sub)
							c.Next()
							return
						}
					}
				}
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
				return
			}
		}

		// Try API Key auth
		apiKey := c.GetHeader("X-API-Key")
		signature := c.GetHeader("X-Signature")
		timestamp := c.GetHeader("X-Timestamp")

		if apiKey != "" && signature != "" && timestamp != "" {
			// Validate timestamp (within 30 seconds)
			ts, err := time.Parse(time.RFC3339, timestamp)
			if err != nil {
				// Try unix millis
				var tsMs int64
				fmt.Sscanf(timestamp, "%d", &tsMs)
				ts = time.UnixMilli(tsMs)
			}
			if math.Abs(time.Since(ts).Seconds()) > 30 {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "timestamp expired"})
				return
			}

			// Lookup API key from account service
			resp, err := http.Get(accountServiceAddr + "/internal/v1/api-key/lookup?api_key=" + apiKey)
			if err != nil || resp.StatusCode != 200 {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
				return
			}
			defer resp.Body.Close()

			var result struct {
				UserID    string `json:"user_id"`
				APISecret string `json:"api_secret"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
				return
			}

			// Verify HMAC signature: HMAC-SHA256(secret, timestamp + method + path + body)
			body, _ := io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(strings.NewReader(string(body)))

			payload := timestamp + c.Request.Method + c.Request.URL.Path + string(body)
			mac := hmac.New(sha256.New, []byte(result.APISecret))
			mac.Write([]byte(payload))
			expectedSig := hex.EncodeToString(mac.Sum(nil))

			if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
				return
			}

			c.Set("user_id", result.UserID)
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization"})
	}
}
