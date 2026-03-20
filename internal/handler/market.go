package handler

import (
	"encoding/json"
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
	// Static metadata for all supported trading pairs
	meta := map[string][2]string{
		"BTC-USDT":   {"BTC", "USDT"},
		"ETH-USDT":   {"ETH", "USDT"},
		"SOL-USDT":   {"SOL", "USDT"},
		"BNB-USDT":   {"BNB", "USDT"},
		"XRP-USDT":   {"XRP", "USDT"},
		"ADA-USDT":   {"ADA", "USDT"},
		"DOGE-USDT":  {"DOGE", "USDT"},
		"DOT-USDT":   {"DOT", "USDT"},
		"AVAX-USDT":  {"AVAX", "USDT"},
		"LINK-USDT":  {"LINK", "USDT"},
		"MATIC-USDT": {"MATIC", "USDT"},
		"UNI-USDT":   {"UNI", "USDT"},
		"ATOM-USDT":  {"ATOM", "USDT"},
		"LTC-USDT":   {"LTC", "USDT"},
		"FIL-USDT":   {"FIL", "USDT"},
		"APT-USDT":   {"APT", "USDT"},
		"ARB-USDT":   {"ARB", "USDT"},
		"OP-USDT":    {"OP", "USDT"},
		"NEAR-USDT":  {"NEAR", "USDT"},
		"QFC-USDT":   {"QFC", "USDT"},
	}

	// Fetch live ticker data from market-data service
	type ticker struct {
		Symbol    string `json:"symbol"`
		LastPrice string `json:"last_price"`
		High24h   string `json:"high_24h"`
		Low24h    string `json:"low_24h"`
		Volume24h string `json:"volume_24h"`
		Change24h string `json:"change_24h"`
	}

	tickerMap := make(map[string]ticker)
	resp, err := http.Get(h.marketDataAddr + "/api/v1/tickers")
	if err == nil && resp.StatusCode == 200 {
		defer resp.Body.Close()
		var result struct {
			Tickers []ticker `json:"tickers"`
		}
		body, _ := io.ReadAll(resp.Body)
		if json.Unmarshal(body, &result) == nil {
			for _, t := range result.Tickers {
				tickerMap[t.Symbol] = t
			}
		}
	}

	symbols := make([]gin.H, 0, len(meta))
	for sym, bq := range meta {
		entry := gin.H{
			"symbol": sym,
			"base":   bq[0],
			"quote":  bq[1],
		}
		if t, ok := tickerMap[sym]; ok {
			entry["price"] = t.LastPrice
			entry["change_24h"] = t.Change24h
			entry["high_24h"] = t.High24h
			entry["low_24h"] = t.Low24h
			entry["volume_24h"] = t.Volume24h
		} else {
			entry["price"] = "0"
			entry["change_24h"] = "0.00"
			entry["volume_24h"] = "0"
		}
		symbols = append(symbols, entry)
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
