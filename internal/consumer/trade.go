package consumer

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/segmentio/kafka-go"

	"github.com/mantis-exchange/mantis-gateway/internal/grpcclient"
	"github.com/mantis-exchange/mantis-gateway/internal/ws"
	pb "github.com/mantis-exchange/mantis-gateway/pkg/proto/mantis"
)

const tradeTopic = "mantis.trades"

type tradeMessage struct {
	ID           string `json:"id"`
	Symbol       string `json:"symbol"`
	Price        string `json:"price"`
	Quantity     string `json:"quantity"`
	MakerOrderID string `json:"maker_order_id"`
	TakerOrderID string `json:"taker_order_id"`
	MakerSide    string `json:"maker_side"`
	CreatedAt    int64  `json:"created_at"`
}

// TradeConsumer reads trade events from Kafka and broadcasts them to WebSocket
// clients, then fetches fresh depth from the matching engine after each trade.
type TradeConsumer struct {
	hub      *ws.Hub
	matching *grpcclient.Client
	order    *grpcclient.OrderClient
	brokers  string
}

// NewTradeConsumer creates a new consumer that bridges Kafka trade events to the
// WebSocket hub.
func NewTradeConsumer(hub *ws.Hub, matching *grpcclient.Client, order *grpcclient.OrderClient, brokers string) *TradeConsumer {
	return &TradeConsumer{hub: hub, matching: matching, order: order, brokers: brokers}
}

// Start begins consuming trade messages. It blocks forever and should be called
// in a goroutine.
func (c *TradeConsumer) Start() {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     strings.Split(c.brokers, ","),
		Topic:       tradeTopic,
		GroupID:     "mantis-gateway-ws",
		MinBytes:    1,
		MaxBytes:    10e6,
		StartOffset: kafka.LastOffset,
	})
	defer reader.Close()

	log.Printf("gateway trade consumer started (brokers: %s, topic: %s)", c.brokers, tradeTopic)

	for {
		msg, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("gateway consumer read error: %v", err)
			continue
		}

		var trade tradeMessage
		if err := json.Unmarshal(msg.Value, &trade); err != nil {
			log.Printf("gateway consumer unmarshal error: %v", err)
			continue
		}

		// Broadcast trade to WS clients subscribed to "trades:BTC-USDT".
		c.hub.Broadcast("trades", trade.Symbol, map[string]interface{}{
			"id":             trade.ID,
			"symbol":         trade.Symbol,
			"price":          trade.Price,
			"quantity":       trade.Quantity,
			"maker_order_id": trade.MakerOrderID,
			"taker_order_id": trade.TakerOrderID,
			"maker_side":     trade.MakerSide,
			"created_at":     trade.CreatedAt,
		})

		// Fetch fresh depth from the matching engine and broadcast to depth
		// subscribers so they see the updated order book immediately.
		c.broadcastDepth(trade.Symbol)
	}
}

func (c *TradeConsumer) broadcastDepth(symbol string) {
	resp, err := c.matching.GetDepth(context.Background(), &pb.GetDepthRequest{
		Symbol:    symbol,
		MaxLevels: 20,
	})
	if err != nil {
		log.Printf("failed to fetch depth for %s: %v", symbol, err)
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

	c.hub.Broadcast("depth", symbol, map[string]interface{}{
		"symbol": symbol,
		"bids":   bids,
		"asks":   asks,
	})
}
