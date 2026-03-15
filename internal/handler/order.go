package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// PlaceOrderRequest represents a new order submission.
type PlaceOrderRequest struct {
	Symbol string  `json:"symbol" binding:"required"`
	Side   string  `json:"side" binding:"required,oneof=buy sell"`
	Type   string  `json:"type" binding:"required,oneof=limit market"`
	Price  float64 `json:"price"`
	Qty    float64 `json:"qty" binding:"required,gt=0"`
}

// PlaceOrder handles POST /api/v1/orders.
// Placeholder: returns a mock accepted order. Will forward to matching engine via gRPC.
func PlaceOrder(c *gin.Context) {
	var req PlaceOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")

	// TODO: Forward to matching engine via gRPC.
	c.JSON(http.StatusCreated, gin.H{
		"order_id":   "mock-order-001",
		"user_id":    userID,
		"symbol":     req.Symbol,
		"side":       req.Side,
		"type":       req.Type,
		"price":      req.Price,
		"qty":        req.Qty,
		"status":     "accepted",
		"created_at": time.Now().UTC().Format(time.RFC3339),
	})
}

// CancelOrder handles DELETE /api/v1/orders/:id.
// Placeholder: returns a mock cancellation. Will forward to matching engine via gRPC.
func CancelOrder(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order id is required"})
		return
	}

	userID, _ := c.Get("user_id")

	// TODO: Forward cancellation to matching engine via gRPC.
	c.JSON(http.StatusOK, gin.H{
		"order_id":     orderID,
		"user_id":      userID,
		"status":       "cancelled",
		"cancelled_at": time.Now().UTC().Format(time.RFC3339),
	})
}
