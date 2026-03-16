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

// GetKlines handles GET /api/v1/klines by proxying to the market-data service.
func (h *MarketHandler) GetKlines(c *gin.Context) {
	symbol := c.Query("symbol")
	interval := c.DefaultQuery("interval", "1m")
	limit := c.DefaultQuery("limit", "200")
	url := fmt.Sprintf("%s/api/v1/klines?symbol=%s&interval=%s&limit=%s", h.marketDataAddr, symbol, interval, limit)

	resp, err := http.Get(url)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "market data service unavailable"})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, "application/json", body)
}

// GetSymbols handles GET /api/v1/symbols.
func (h *MarketHandler) GetSymbols(c *gin.Context) {
	symbols := []gin.H{
		{"symbol": "BTC-USDT", "base": "BTC", "quote": "USDT", "price": "65000"},
		{"symbol": "ETH-USDT", "base": "ETH", "quote": "USDT", "price": "3500"},
		{"symbol": "SOL-USDT", "base": "SOL", "quote": "USDT", "price": "150"},
		{"symbol": "BNB-USDT", "base": "BNB", "quote": "USDT", "price": "600"},
		{"symbol": "XRP-USDT", "base": "XRP", "quote": "USDT", "price": "0.55"},
		{"symbol": "ADA-USDT", "base": "ADA", "quote": "USDT", "price": "0.45"},
		{"symbol": "DOGE-USDT", "base": "DOGE", "quote": "USDT", "price": "0.15"},
		{"symbol": "DOT-USDT", "base": "DOT", "quote": "USDT", "price": "7.5"},
		{"symbol": "AVAX-USDT", "base": "AVAX", "quote": "USDT", "price": "35"},
		{"symbol": "LINK-USDT", "base": "LINK", "quote": "USDT", "price": "15"},
		{"symbol": "MATIC-USDT", "base": "MATIC", "quote": "USDT", "price": "0.70"},
		{"symbol": "UNI-USDT", "base": "UNI", "quote": "USDT", "price": "10"},
		{"symbol": "ATOM-USDT", "base": "ATOM", "quote": "USDT", "price": "9"},
		{"symbol": "LTC-USDT", "base": "LTC", "quote": "USDT", "price": "85"},
		{"symbol": "FIL-USDT", "base": "FIL", "quote": "USDT", "price": "5.5"},
		{"symbol": "APT-USDT", "base": "APT", "quote": "USDT", "price": "9.5"},
		{"symbol": "ARB-USDT", "base": "ARB", "quote": "USDT", "price": "1.1"},
		{"symbol": "OP-USDT", "base": "OP", "quote": "USDT", "price": "2.5"},
		{"symbol": "NEAR-USDT", "base": "NEAR", "quote": "USDT", "price": "5"},
		{"symbol": "QFC-USDT", "base": "QFC", "quote": "USDT", "price": "1.5"},
	}
	c.JSON(http.StatusOK, gin.H{"symbols": symbols})
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
