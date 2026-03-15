package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/mantis-exchange/mantis-gateway/internal/grpcclient"
	pb "github.com/mantis-exchange/mantis-gateway/pkg/proto/mantis"
)

// OrderHandler holds dependencies for order-related HTTP handlers.
type OrderHandler struct {
	matching *grpcclient.Client
}

// NewOrderHandler creates a new OrderHandler backed by the given gRPC client.
func NewOrderHandler(mc *grpcclient.Client) *OrderHandler {
	return &OrderHandler{matching: mc}
}

// PlaceOrderRequest represents a new order submission.
type PlaceOrderRequest struct {
	Symbol      string  `json:"symbol" binding:"required"`
	Side        string  `json:"side" binding:"required,oneof=buy sell"`
	Type        string  `json:"type" binding:"required,oneof=limit market"`
	TimeInForce string  `json:"time_in_force"`
	Price       string  `json:"price"`
	Qty         string  `json:"qty" binding:"required"`
}

// PlaceOrder handles POST /api/v1/orders.
func (h *OrderHandler) PlaceOrder(c *gin.Context) {
	var req PlaceOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	side, err := parseSide(req.Side)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orderType, err := parseOrderType(req.Type)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tif := parseTimeInForce(req.TimeInForce)

	resp, err := h.matching.SubmitOrder(c.Request.Context(), &pb.SubmitOrderRequest{
		Symbol:      req.Symbol,
		Side:        side,
		OrderType:   orderType,
		TimeInForce: tif,
		Price:       req.Price,
		Quantity:    req.Qty,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("matching engine error: %v", err)})
		return
	}

	order := resp.GetOrder()
	result := gin.H{
		"order_id":        order.GetId(),
		"symbol":          order.GetSymbol(),
		"side":            order.GetSide().String(),
		"type":            order.GetOrderType().String(),
		"time_in_force":   order.GetTimeInForce().String(),
		"price":           order.GetPrice(),
		"quantity":        order.GetQuantity(),
		"filled_quantity": order.GetFilledQuantity(),
		"status":          order.GetStatus().String(),
		"created_at":      order.GetCreatedAt(),
	}

	// Include any immediate trades (e.g. market orders or crossing limit orders).
	if len(resp.GetTrades()) > 0 {
		trades := make([]gin.H, 0, len(resp.GetTrades()))
		for _, t := range resp.GetTrades() {
			trades = append(trades, gin.H{
				"id":       t.GetId(),
				"price":    t.GetPrice(),
				"quantity": t.GetQuantity(),
			})
		}
		result["trades"] = trades
	}

	c.JSON(http.StatusCreated, result)
}

// CancelOrder handles DELETE /api/v1/orders/:id.
func (h *OrderHandler) CancelOrder(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order id is required"})
		return
	}

	// Symbol is required by the matching engine to locate the order book.
	symbol := c.Query("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol query parameter is required"})
		return
	}

	resp, err := h.matching.CancelOrder(c.Request.Context(), &pb.CancelOrderRequest{
		Symbol:  symbol,
		OrderId: orderID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("matching engine error: %v", err)})
		return
	}

	if !resp.GetSuccess() {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found or already cancelled"})
		return
	}

	order := resp.GetOrder()
	c.JSON(http.StatusOK, gin.H{
		"order_id":   order.GetId(),
		"symbol":     order.GetSymbol(),
		"status":     order.GetStatus().String(),
		"cancelled":  true,
		"created_at": order.GetCreatedAt(),
	})
}

// parseSide converts a string side to the protobuf enum.
func parseSide(s string) (pb.Side, error) {
	switch s {
	case "buy":
		return pb.Side_SIDE_BUY, nil
	case "sell":
		return pb.Side_SIDE_SELL, nil
	default:
		return pb.Side_SIDE_UNSPECIFIED, fmt.Errorf("invalid side: %s", s)
	}
}

// parseOrderType converts a string order type to the protobuf enum.
func parseOrderType(t string) (pb.OrderType, error) {
	switch t {
	case "limit":
		return pb.OrderType_ORDER_TYPE_LIMIT, nil
	case "market":
		return pb.OrderType_ORDER_TYPE_MARKET, nil
	default:
		return pb.OrderType_ORDER_TYPE_UNSPECIFIED, fmt.Errorf("invalid order type: %s", t)
	}
}

// parseTimeInForce converts a string time-in-force to the protobuf enum.
func parseTimeInForce(tif string) pb.TimeInForce {
	switch tif {
	case "GTC", "gtc":
		return pb.TimeInForce_TIME_IN_FORCE_GTC
	case "IOC", "ioc":
		return pb.TimeInForce_TIME_IN_FORCE_IOC
	case "FOK", "fok":
		return pb.TimeInForce_TIME_IN_FORCE_FOK
	default:
		return pb.TimeInForce_TIME_IN_FORCE_GTC // default to GTC
	}
}
