package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetAccount handles GET /api/v1/account.
// Placeholder: returns mock account info. Will query account service.
func GetAccount(c *gin.Context) {
	userID, _ := c.Get("user_id")

	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"email":   "user@example.com",
		"status":  "active",
		"kyc":     "verified",
	})
}

// GetBalances handles GET /api/v1/account/balances.
// Placeholder: returns mock balances. Will query account service.
func GetBalances(c *gin.Context) {
	userID, _ := c.Get("user_id")

	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"balances": []gin.H{
			{"asset": "BTC", "free": "1.50000000", "locked": "0.25000000"},
			{"asset": "USDT", "free": "50000.00", "locked": "10000.00"},
			{"asset": "ETH", "free": "10.00000000", "locked": "0.00000000"},
		},
	})
}
