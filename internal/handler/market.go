package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetDepth handles GET /api/v1/depth/:symbol.
// Placeholder: returns mock order book depth. Will query matching engine via gRPC.
func GetDepth(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol is required"})
		return
	}

	// TODO: Fetch real depth from matching engine.
	c.JSON(http.StatusOK, gin.H{
		"symbol": symbol,
		"bids": [][]interface{}{
			{"50000.00", "1.5"},
			{"49999.00", "2.3"},
			{"49998.00", "0.8"},
		},
		"asks": [][]interface{}{
			{"50001.00", "1.2"},
			{"50002.00", "3.1"},
			{"50003.00", "0.5"},
		},
	})
}

// GetTrades handles GET /api/v1/trades/:symbol.
// Placeholder: returns mock recent trades. Will query matching engine via gRPC.
func GetTrades(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol is required"})
		return
	}

	// TODO: Fetch real trades from matching engine.
	c.JSON(http.StatusOK, gin.H{
		"symbol": symbol,
		"trades": []gin.H{
			{"id": "t1", "price": "50000.50", "qty": "0.1", "side": "buy", "time": 1700000000000},
			{"id": "t2", "price": "50000.00", "qty": "0.5", "side": "sell", "time": 1700000001000},
			{"id": "t3", "price": "50001.00", "qty": "0.25", "side": "buy", "time": 1700000002000},
		},
	})
}
