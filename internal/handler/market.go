package handler

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/mantis-exchange/mantis-gateway/internal/grpcclient"
	pb "github.com/mantis-exchange/mantis-gateway/pkg/proto/mantis"
)

// MarketHandler holds dependencies for market-data HTTP handlers.
type MarketHandler struct {
	matching       *grpcclient.Client
	marketDataAddr string
}

// NewMarketHandler creates a new MarketHandler backed by the given gRPC client.
func NewMarketHandler(mc *grpcclient.Client, marketDataAddr string) *MarketHandler {
	return &MarketHandler{matching: mc, marketDataAddr: marketDataAddr}
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

// GetTrades handles GET /api/v1/trades/:symbol by proxying to the market-data service.
func (h *MarketHandler) GetTrades(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol is required"})
		return
	}

	limit := c.DefaultQuery("limit", "50")
	url := fmt.Sprintf("%s/api/v1/trades?symbol=%s&limit=%s", h.marketDataAddr, symbol, limit)

	resp, err := http.Get(url)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "market data service unavailable"})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, "application/json", body)
}
