package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mantis-exchange/mantis-gateway/internal/config"
	"github.com/mantis-exchange/mantis-gateway/internal/handler"
	"github.com/mantis-exchange/mantis-gateway/internal/middleware"
	"github.com/mantis-exchange/mantis-gateway/internal/ws"
)

func main() {
	cfg := config.Load()

	hub := ws.NewHub()
	go hub.Run()

	r := gin.Default()

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
		public.GET("/depth/:symbol", handler.GetDepth)
		public.GET("/trades/:symbol", handler.GetTrades)
	}

	// Authenticated API routes.
	auth := r.Group("/api/v1")
	auth.Use(middleware.Auth(cfg.JWTSecret))
	{
		auth.POST("/orders", handler.PlaceOrder)
		auth.DELETE("/orders/:id", handler.CancelOrder)
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

	log.Printf("mantis-gateway starting on :%s (matching-engine: %s)", cfg.Port, cfg.MatchingEngineAddr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
