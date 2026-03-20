package ws

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins — Traefik handles origin validation in production.
		return true
	},
}

// SubscribeMessage is sent by clients to subscribe to a market data channel.
type SubscribeMessage struct {
	Action  string `json:"action"`  // "subscribe" or "unsubscribe"
	Channel string `json:"channel"` // e.g. "depth", "trades"
	Symbol  string `json:"symbol"`  // e.g. "BTCUSDT"
}

// Client represents a single WebSocket connection.
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	mu     sync.Mutex
	subs   map[string]bool // subscribed channels, keyed by "channel:symbol"
	userID string          // Set if client authenticated via JWT
}

// Hub maintains the set of active clients and broadcasts messages to them.
type Hub struct {
	mu         sync.RWMutex
	clients    map[*Client]bool
	broadcast  chan BroadcastMessage
	register   chan *Client
	unregister chan *Client
}

// BroadcastMessage carries data to be sent to clients subscribed to a channel.
type BroadcastMessage struct {
	Channel string // "channel:symbol"
	Data    []byte
}

// NewHub creates and returns a new Hub.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan BroadcastMessage, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub event loop. Should be called in a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				client.mu.Lock()
				if client.subs[msg.Channel] {
					select {
					case client.send <- msg.Data:
					default:
						// Client send buffer full; disconnect.
						go func(c *Client) {
							h.unregister <- c
						}(client)
					}
				}
				client.mu.Unlock()
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends a message to all clients subscribed to the given channel.
func (h *Hub) Broadcast(channel, symbol string, data interface{}) {
	payload, err := json.Marshal(gin.H{
		"channel": channel,
		"symbol":  symbol,
		"data":    data,
	})
	if err != nil {
		log.Printf("ws broadcast marshal error: %v", err)
		return
	}
	h.broadcast <- BroadcastMessage{
		Channel: channel + ":" + symbol,
		Data:    payload,
	}
}

// HandleWS returns a Gin handler that upgrades HTTP connections to WebSocket.
// It accepts an optional JWT token query parameter for authenticating private channels.
func (h *Hub) HandleWS(jwtSecret string) gin.HandlerFunc {
	secretBytes := []byte(jwtSecret)
	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("ws upgrade error: %v", err)
			return
		}

		// Optional JWT auth from query param
		var userID string
		if tokenStr := c.Query("token"); tokenStr != "" {
			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method")
				}
				return secretBytes, nil
			})
			if err == nil && token.Valid {
				if claims, ok := token.Claims.(jwt.MapClaims); ok {
					if sub, ok := claims["sub"].(string); ok {
						userID = sub
					}
				}
			}
		}

		client := &Client{
			hub:    h,
			conn:   conn,
			send:   make(chan []byte, 256),
			subs:   make(map[string]bool),
			userID: userID,
		}
		h.register <- client

		go client.writePump()
		go client.readPump()
	}
}

// BroadcastToUser sends a message to a specific user's private channel.
func (h *Hub) BroadcastToUser(userID string, channel string, data interface{}) {
	h.Broadcast(channel, userID, data)
}

// readPump reads messages from the WebSocket connection.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("ws read error: %v", err)
			}
			break
		}

		var msg SubscribeMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("ws unmarshal error: %v", err)
			continue
		}

		key := msg.Channel + ":" + msg.Symbol
		c.mu.Lock()
		switch msg.Action {
		case "subscribe":
			if msg.Channel == "orders" && c.userID == "" {
				errMsg, _ := json.Marshal(gin.H{
					"action":  "subscribe",
					"channel": msg.Channel,
					"status":  "error",
					"error":   "authentication required",
				})
				c.mu.Unlock()
				select {
				case c.send <- errMsg:
				default:
				}
				continue
			}
			// For private channels, use user-specific key
			if msg.Channel == "orders" {
				key = "orders:" + c.userID
			}
			c.subs[key] = true
			log.Printf("ws client subscribed to %s", key)
		case "unsubscribe":
			delete(c.subs, key)
			log.Printf("ws client unsubscribed from %s", key)
		}
		c.mu.Unlock()

		// Send acknowledgement.
		ack, _ := json.Marshal(gin.H{
			"action":  msg.Action,
			"channel": msg.Channel,
			"symbol":  msg.Symbol,
			"status":  "ok",
		})
		select {
		case c.send <- ack:
		default:
		}
	}
}

// writePump writes messages to the WebSocket connection.
func (c *Client) writePump() {
	defer c.conn.Close()

	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Printf("ws write error: %v", err)
			return
		}
	}
}
