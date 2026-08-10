// TigerWallet - WebSocket Service
// High-performance Go WebSocket server for real-time updates
// Supports: Trading, Order Book, Tickers, Bot Status, Notifications

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

const (
	SERVER_PORT      = "8095"
	WRITE_TIMEOUT    = 10 * time.Second
	READ_TIMEOUT     = 60 * time.Second
	PING_INTERVAL    = 30 * time.Second
	MAX_MESSAGE_SIZE = 1024 * 1024 // 1MB
)

var (
	upgrader    websocket.Upgrader
	redisClient *redis.Client
	hub         *Hub
	channels    = make(map[string]map[*Client]bool)
	channelsMux sync.RWMutex
)

type Hub struct {
	// Registered clients
	clients map[*Client]bool
	// Registered clients by user ID
	clientsByUser map[string]map[*Client]bool
	// Inbound messages from clients
	broadcast chan *Message
	// Register requests from clients
	register chan *Client
	// Unregister requests from clients
	unregister chan *Client
	// Mutex
	mu sync.RWMutex
}

type Client struct {
	hub     *Hub
	conn    *websocket.Conn
	send    chan []byte
	userID  string
	email   string
	roles   []string
	isAdmin bool
}

type Message struct {
	Type      string          `json:"type"`
	Channel   string          `json:"channel,omitempty"`
	Payload   json.RawMessage `json:"payload"`
	UserID    string          `json:"user_id,omitempty"`
	Timestamp int64           `json:"timestamp"`
}

type WSMessage struct {
	Type    string      `json:"type"`
	Channel string      `json:"channel"`
	Data    interface{} `json:"data"`
}

type SubscriptionRequest struct {
	Channel string   `json:"channel"`
	Token   string   `json:"token,omitempty"`
	Roles   []string `json:"roles,omitempty"`
}

type AuthPayload struct {
	UserID string   `json:"user_id"`
	Email  string   `json:"email"`
	Roles  []string `json:"roles"`
}

// Channel types
const (
	ChannelTicker       = "ticker"
	ChannelOrderBook    = "orderbook"
	ChannelTrade        = "trade"
	ChannelBot          = "bot"
	ChannelListing      = "listing"
	ChannelPayment      = "payment"
	ChannelWallet       = "wallet"
	ChannelNotification = "notification"
	ChannelAdmin        = "admin"
)

// ============================================================================
// HUB
// ============================================================================

func newHub() *Hub {
	return &Hub{
		clients:       make(map[*Client]bool),
		clientsByUser: make(map[string]map[*Client]bool),
		broadcast:     make(chan *Message, 256),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			if client.userID != "" {
				if h.clientsByUser[client.userID] == nil {
					h.clientsByUser[client.userID] = make(map[*Client]bool)
				}
				h.clientsByUser[client.userID][client] = true
			}
			h.mu.Unlock()
			log.Printf("✅ Client connected: %s (%s)", client.userID, client.conn.RemoteAddr().String())

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				if client.userID != "" && h.clientsByUser[client.userID] != nil {
					delete(h.clientsByUser[client.userID], client)
				}
			}
			h.mu.Unlock()
			log.Printf("❌ Client disconnected: %s (%s)", client.userID, client.conn.RemoteAddr().String())

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				if shouldSendToClient(client, message) {
					select {
					case client.send <- encodeMessage(message):
					default:
						close(client.send)
						delete(h.clients, client)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

func shouldSendToClient(client *Client, message *Message) bool {
	// Admin receives all
	if client.isAdmin {
		return true
	}

	// Check if user is subscribed to channel
	channelsMux.RLock()
	defer channelsMux.RUnlock()

	// Get subscribers for this channel
	subs, ok := channels[message.Channel]
	if !ok {
		return true // No restriction on this channel
	}

	// Check if client is subscribed
	_, subscribed := subs[client]
	return subscribed
}

func encodeMessage(msg *Message) []byte {
	data, err := json.Marshal(msg)
	if err != nil {
		return []byte(fmt.Sprintf(`{"type":"error","error":"%s"}`, err.Error()))
	}
	return data
}

// ============================================================================
// CLIENT
// ============================================================================

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(MAX_MESSAGE_SIZE)
	c.conn.SetReadDeadline(time.Now().Add(READ_TIMEOUT))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(READ_TIMEOUT))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			c.send <- []byte(fmt.Sprintf(`{"type":"error","error":"%s"}`, err.Error()))
			continue
		}

		msg.Timestamp = time.Now().UnixMilli()
		c.handleMessage(&msg)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(PING_INTERVAL)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(WRITE_TIMEOUT))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(WRITE_TIMEOUT))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) handleMessage(msg *Message) {
	switch msg.Type {
	case "subscribe":
		var sub SubscriptionRequest
		if err := json.Unmarshal(msg.Payload, &sub); err != nil {
			c.send <- []byte(fmt.Sprintf(`{"type":"error","error":"%s"}`, err.Error()))
			return
		}
		c.subscribe(sub.Channel)

	case "unsubscribe":
		var sub SubscriptionRequest
		if err := json.Unmarshal(msg.Payload, &sub); err != nil {
			c.send <- []byte(fmt.Sprintf(`{"type":"error","error":"%s"}`, err.Error()))
			return
		}
		c.unsubscribe(sub.Channel)

	case "auth":
		var auth AuthPayload
		if err := json.Unmarshal(msg.Payload, &auth); err != nil {
			c.send <- []byte(fmt.Sprintf(`{"type":"error","error":"%s"}`, err.Error()))
			return
		}
		c.userID = auth.UserID
		c.email = auth.Email
		c.roles = auth.Roles
		for _, role := range auth.Roles {
			if role == "admin" || role == "super_admin" {
				c.isAdmin = true
				break
			}
		}
		c.send <- []byte(`{"type":"auth","status":"authenticated"}`)

	case "ping":
		c.send <- []byte(`{"type":"pong","timestamp":` + strconv.FormatInt(time.Now().UnixMilli(), 10) + `}`)

	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}
}

func (c *Client) subscribe(channel string) {
	channelsMux.Lock()
	defer channelsMux.Unlock()

	if channels[channel] == nil {
		channels[channel] = make(map[*Client]bool)
	}
	channels[channel][c] = true

	c.send <- []byte(fmt.Sprintf(`{"type":"subscribed","channel":"%s"}`, channel))
	log.Printf("📡 Client %s subscribed to %s", c.userID, channel)
}

func (c *Client) unsubscribe(channel string) {
	channelsMux.Lock()
	defer channelsMux.Unlock()

	if channels[channel] != nil {
		delete(channels[channel], c)
	}

	c.send <- []byte(fmt.Sprintf(`{"type":"unsubscribed","channel":"%s"}`, channel))
	log.Printf("📡 Client %s unsubscribed from %s", c.userID, channel)
}

// ============================================================================
// BROADCAST FUNCTIONS
// ============================================================================

func BroadcastToChannel(channel string, data interface{}) {
	msg := Message{
		Type:      "message",
		Channel:   channel,
		Payload:   mustMarshal(data),
		Timestamp: time.Now().UnixMilli(),
	}

	hub.broadcast <- &msg
}

func BroadcastToUser(userID string, data interface{}) {
	msg := Message{
		Type:      "message",
		UserID:    userID,
		Payload:   mustMarshal(data),
		Timestamp: time.Now().UnixMilli(),
	}

	hub.broadcast <- &msg
}

func BroadcastToAll(data interface{}) {
	msg := Message{
		Type:      "message",
		Payload:   mustMarshal(data),
		Timestamp: time.Now().UnixMilli(),
	}

	hub.broadcast <- &msg
}

func mustMarshal(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}

// ============================================================================
// WEBSOCKET HANDLERS
// ============================================================================

func handleWebSocket(c *gin.Context) {
	// Get token from query or header
	token := c.Query("token")
	if token == "" {
		token = c.GetHeader("Authorization")
	}

	// Upgrade connection
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Create client
	client := &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: "",
	}

	// Register client
	hub.register <- client

	// Start pumps
	go client.writePump()
	go client.readPump()
}

// ============================================================================
// TICKER STREAMING
// ============================================================================

type TickerUpdate struct {
	Symbol    string  `json:"symbol"`
	Price     float64 `json:"price"`
	Change24h float64 `json:"change24h"`
	Volume24h float64 `json:"volume24h"`
	Timestamp int64   `json:"timestamp"`
}

func StartTickerStream(exchanges []string) {
	// Simulate ticker updates (in production would connect to exchanges)
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		symbols := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "SOLUSDT"}

		for range ticker.C {
			for _, symbol := range symbols {
				update := TickerUpdate{
					Symbol:    symbol,
					Price:     40000 + float64(time.Now().Unix()%1000),
					Change24h: float64(time.Now().Unix()%10) - 5,
					Volume24h: 1000000,
					Timestamp: time.Now().UnixMilli(),
				}

				BroadcastToChannel(ChannelTicker, update)
			}
		}
	}()
}

// ============================================================================
// ORDER BOOK STREAMING
// ============================================================================

type OrderBookUpdate struct {
	Symbol    string      `json:"symbol"`
	Bids      [][]float64 `json:"bids"`
	Asks      [][]float64 `json:"asks"`
	Exchange  string      `json:"exchange"`
	Timestamp int64       `json:"timestamp"`
}

func StartOrderBookStream() {
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		symbols := []string{"BTCUSDT", "ETHUSDT"}

		for range ticker.C {
			for _, symbol := range symbols {
				update := OrderBookUpdate{
					Symbol:    symbol,
					Bids:      [][]float64{{40000, 1.5}, {39999, 2.0}, {39998, 3.0}},
					Asks:      [][]float64{{40001, 1.0}, {40002, 2.5}, {40003, 1.5}},
					Exchange:  "binance",
					Timestamp: time.Now().UnixMilli(),
				}

				BroadcastToChannel(ChannelOrderBook, update)
			}
		}
	}()
}

// ============================================================================
// BOT STATUS STREAMING
// ============================================================================

type BotStatusUpdate struct {
	BotID     string  `json:"bot_id"`
	BotType   string  `json:"bot_type"`
	Status    string  `json:"status"` // running, stopped, error
	PnL       float64 `json:"pnl"`
	Volume    float64 `json:"volume"`
	Orders    int     `json:"orders"`
	Timestamp int64   `json:"timestamp"`
}

func StartBotStatusStream() {
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		botIDs := []string{"bot-001", "bot-002", "bot-003"}

		for range ticker.C {
			for _, botID := range botIDs {
				update := BotStatusUpdate{
					BotID:     botID,
					BotType:   "grid",
					Status:    "running",
					PnL:       float64(time.Now().Unix() % 1000),
					Volume:    50000,
					Orders:    int(time.Now().Unix() % 100),
					Timestamp: time.Now().UnixMilli(),
				}

				BroadcastToChannel(ChannelBot, update)
			}
		}
	}()
}

// ============================================================================
// PAYMENT STATUS STREAMING
// ============================================================================

type PaymentStatusUpdate struct {
	PaymentID     string `json:"payment_id"`
	Status        string `json:"status"` // pending, confirmed, completed, failed
	Confirmations int    `json:"confirmations"`
	TxHash        string `json:"tx_hash,omitempty"`
	Timestamp     int64  `json:"timestamp"`
}

// ============================================================================
// LISTING STATUS STREAMING
// ============================================================================

type ListingStatusUpdate struct {
	ListingID string `json:"listing_id"`
	Status    string `json:"status"` // pending, approved, rejected
	Timestamp int64  `json:"timestamp"`
}

// ============================================================================
// ADMIN BROADCAST
// ============================================================================

type AdminBroadcast struct {
	Title   string `json:"title"`
	Message string `json:"message"`
	Level   string `json:"level"` // info, warning, critical
}

func BroadcastAdminMessage(title, message, level string) {
	update := AdminBroadcast{
		Title:   title,
		Message: message,
		Level:   level,
	}

	BroadcastToChannel(ChannelAdmin, update)
}

// ============================================================================
// API ENDPOINTS
// ============================================================================

func GetStats(c *gin.Context) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	totalClients := len(hub.clients)
	userCount := len(hub.clientsByUser)

	channelsMux.RLock()
	channelCount := len(channels)
	channelsMux.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"total_connections": totalClients,
			"unique_users":      userCount,
			"channels":          channelCount,
			"uptime":            time.Since(startTime).Seconds(),
		},
	})
}

func SendTestMessage(c *gin.Context) {
	var req struct {
		Channel string `json:"channel"`
		Message string `json:"message"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	BroadcastToChannel(req.Channel, gin.H{"message": req.Message})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Message sent to channel",
	})
}

func SendToUser(c *gin.Context) {
	var req struct {
		UserID  string `json:"user_id" binding:"required"`
		Message string `json:"message" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	BroadcastToUser(req.UserID, gin.H{"message": req.Message})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Message sent to user",
	})
}

func GetChannelSubscribers(c *gin.Context) {
	channel := c.Param("channel")

	channelsMux.RLock()
	defer channelsMux.RUnlock()

	subs, ok := channels[channel]
	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"success":     true,
			"subscribers": 0,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"subscribers": len(subs),
	})
}

// ============================================================================
// MAIN
// ============================================================================

var startTime time.Time

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("🚀 Starting TigerWallet WebSocket Service...")

	startTime = time.Now()

	// Initialize Redis
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	ctx := context.Background()
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		log.Printf("⚠️  Redis not available: %v", err)
	} else {
		log.Println("✅ Redis connected")
	}

	// Initialize hub
	hub = newHub()
	go hub.run()

	// Start streaming services
	StartTickerStream([]string{"binance", "bybit", "okx"})
	StartOrderBookStream()
	StartBotStatusStream()

	// Configure upgrader
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins in development
		},
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
	}

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// CORS
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "websocket",
			"time":    time.Now().Unix(),
		})
	})

	// WebSocket endpoint
	r.GET("/ws", handleWebSocket)

	// REST endpoints for broadcasting
	r.GET("/api/stats", GetStats)
	r.POST("/api/broadcast/channel", SendTestMessage)
	r.POST("/api/broadcast/user", SendToUser)
	r.GET("/api/channels/:channel/subscribers", GetChannelSubscribers)

	// Channels info
	r.GET("/api/channels", func(c *gin.Context) {
		channelsMux.RLock()
		defer channelsMux.RUnlock()

		channelInfo := make(map[string]int)
		for ch, subs := range channels {
			channelInfo[ch] = len(subs)
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    channelInfo,
		})
	})

	log.Printf("✅ WebSocket server running on port %s", SERVER_PORT)
	log.Printf("📡 WebSocket endpoint: ws://localhost:%s/ws", SERVER_PORT)

	if err := r.Run(":" + SERVER_PORT); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
