package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mantis-exchange/mantis-gateway/internal/config"
	"github.com/mantis-exchange/mantis-gateway/internal/consumer"
	"github.com/mantis-exchange/mantis-gateway/internal/grpcclient"
	"github.com/mantis-exchange/mantis-gateway/internal/handler"
	"github.com/mantis-exchange/mantis-gateway/internal/middleware"
	"github.com/mantis-exchange/mantis-gateway/internal/ws"
)

func main() {
	cfg := config.Load()

	// Connect to the matching engine via gRPC (for market data queries).
	matchingClient, err := grpcclient.New(cfg.MatchingEngineAddr)
	if err != nil {
		log.Fatalf("failed to connect to matching engine: %v", err)
	}
	defer matchingClient.Close()

	// Connect to the order service via gRPC.
	orderClient, err := grpcclient.NewOrderClient(cfg.OrderServiceAddr)
	if err != nil {
		log.Fatalf("failed to connect to order service: %v", err)
	}
	defer orderClient.Close()

	hub := ws.NewHub()
	go hub.Run()

	// Start WebSocket trade consumer for real-time broadcasting.
	tc := consumer.NewTradeConsumer(hub, matchingClient, cfg.KafkaBrokers)
	go tc.Start()

	orderHandler := handler.NewOrderHandler(orderClient)
	marketHandler := handler.NewMarketHandler(matchingClient)

	r := gin.Default()

	// Security middleware
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(cfg.CORSOrigins))

	// Global rate limiter: 100 requests/sec per IP, burst of 100.
	limiter := middleware.NewRateLimiter(100, 100)
	r.Use(limiter.Middleware())

	// Health check.
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// WebSocket endpoint (no auth required for market data).
	r.GET("/ws", hub.HandleWS)

	// Public API routes (no auth).
	public := r.Group("/api/v1")
	{
		public.GET("/depth/:symbol", marketHandler.GetDepth)
		public.GET("/trades/:symbol", handler.GetTrades)
	}

	// Authenticated API routes.
	auth := r.Group("/api/v1")
	auth.Use(middleware.Auth(cfg.JWTSecret))
	{
		auth.POST("/orders", orderHandler.PlaceOrder)
		auth.DELETE("/orders/:id", orderHandler.CancelOrder)
		auth.GET("/account", handler.GetAccount)
		auth.GET("/account/balances", handler.GetBalances)
	}

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("mantis-gateway starting on :%s (engine: %s, order-service: %s)", cfg.Port, cfg.MatchingEngineAddr, cfg.OrderServiceAddr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
