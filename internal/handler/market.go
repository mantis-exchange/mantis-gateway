package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/mantis-exchange/mantis-gateway/internal/grpcclient"
	pb "github.com/mantis-exchange/mantis-gateway/pkg/proto/mantis"
)

// MarketHandler holds dependencies for market-data HTTP handlers.
type MarketHandler struct {
	matching *grpcclient.Client
}

// NewMarketHandler creates a new MarketHandler backed by the given gRPC client.
func NewMarketHandler(mc *grpcclient.Client) *MarketHandler {
	return &MarketHandler{matching: mc}
}

// GetDepth handles GET /api/v1/depth/:symbol.
func (h *MarketHandler) GetDepth(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol is required"})
		return
	}

	maxLevels := int32(20) // default
	if v := c.Query("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n > 0 {
			maxLevels = int32(n)
		}
	}

	resp, err := h.matching.GetDepth(c.Request.Context(), &pb.GetDepthRequest{
		Symbol:    symbol,
		MaxLevels: maxLevels,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("matching engine error: %v", err)})
		return
	}

	depth := resp.GetDepth()

	bids := make([][]string, 0, len(depth.GetBids()))
	for _, lvl := range depth.GetBids() {
		bids = append(bids, []string{lvl.GetPrice(), lvl.GetQuantity()})
	}

	asks := make([][]string, 0, len(depth.GetAsks()))
	for _, lvl := range depth.GetAsks() {
		asks = append(asks, []string{lvl.GetPrice(), lvl.GetQuantity()})
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol": depth.GetSymbol(),
		"bids":   bids,
		"asks":   asks,
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
