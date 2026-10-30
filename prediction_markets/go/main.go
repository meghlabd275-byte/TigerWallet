package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

// ============================================================================
// TIGERWALLET PREDICTION MARKETS SYSTEM
// Production-ready prediction markets integration (Polymarket, etc.)
// ============================================================================

var (
	logger        zerolog.Logger
	redisClient   *redis.Client
	wsHub         *WebSocketHub
	store         *Store
	config        *Config
	marketSources []marketSource
)

func main() {
	// Initialize logger
	output := zerolog.ConsoleWriter{Out: os.Stdout}
	logger = zerolog.New(output).With().Timestamp().Logger()

	// Load configuration
	cfg := loadConfig()
	config = cfg

	// Without a signing key the trading endpoints could not tell one account
	// from another, so the service refuses to start rather than run open.
	if cfg.JWTSecret == "" {
		log.Fatal(ErrNoSigningKey)
	}

	// Initialize Redis. The cache is optional; the database is not.
	redisClient = redis.NewClient(&redis.Options{
		Addr:     cfg.RedisURL,
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       0,
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Warn().Err(err).Msg("Redis is unreachable; the cache is disabled")
	}

	// Trading cannot be served without durable storage, so an unavailable
	// database is fatal rather than a degraded mode.
	var err error
	store, err = NewStore(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database is required: %v", err)
	}
	defer store.Close()
	logger.Info().Msg("database ready")

	// Load the configured market sources.
	marketSources = newMarketSourceRegistry(cfg)
	if len(marketSources) == 0 {
		logger.Warn().Msg("no market source is configured; the market list stays as stored")
	}
	go pollMarkets(ctx, cfg.MarketPollInterval)

	// Initialize WebSocket hub
	wsHub = NewWebSocketHub()

	// Start WebSocket hub
	go wsHub.Run()

	// Setup router
	router := setupRouter(cfg)

	// Start server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	logger.Info().Str("port", cfg.Port).Msg("Prediction Markets service started")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("Server forced to shutdown")
	}

	logger.Info().Msg("Server exited")
}

// Configuration
type Config struct {
	Port                 string
	RedisURL             string
	DatabaseURL          string
	JWTSecret            string
	PolymarketAPI        string
	PredictFunAPI        string
	HyperliquidAPI       string
	WSOrigins            []string
	MarketUpdateInterval time.Duration
	MarketPollInterval   time.Duration
}

func loadConfig() *Config {
	return &Config{
		Port:        getEnv("PREDICTION_PORT", "9207"),
		RedisURL:    getEnv("REDIS_URL", "localhost:6379"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		// The Gamma API serves market metadata. The CLOB API serves only
		// order-book endpoints and does not answer a market listing.
		PolymarketAPI:  getEnv("POLYMARKET_API", "https://gamma-api.polymarket.com"),
		PredictFunAPI:  getEnv("PREDICTFUN_API", ""),
		HyperliquidAPI: getEnv("HYPERLIQUID_API", ""),
		// A wildcard origin is refused: the permitted origins must be listed
		// explicitly so an unrelated page cannot read authenticated responses.
		WSOrigins:            splitAndTrim(getEnv("WS_ORIGINS", "")),
		MarketUpdateInterval: 5 * time.Second,
		MarketPollInterval:   getDurationEnv("MARKET_POLL_INTERVAL", 60*time.Second),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// getDurationEnv reads a duration such as "90s" or "5m".
func getDurationEnv(key string, def time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return def
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed <= 0 {
		log.Printf("%s=%q is not a valid duration; using %s", key, raw, def)
		return def
	}
	return parsed
}

// splitAndTrim splits a comma-separated list and drops blank entries.
func splitAndTrim(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// originAllowed reports whether an origin is on the allow-list. A wildcard is
// deliberately not honoured, because the service accepts credentials.
func originAllowed(origins []string, origin string) bool {
	if origin == "" {
		// A non-browser client sends no Origin header. It carries no ambient
		// credentials, so the request is not a cross-site forgery.
		return true
	}
	for _, allowed := range origins {
		if allowed == origin {
			return true
		}
	}
	return false
}

// ============================================================================
// DATA MODELS
// ============================================================================

// Market represents a prediction market
type Market struct {
	ID           string  `json:"id"`
	Question     string  `json:"question"`
	Description  string  `json:"description"`
	Source       string  `json:"source"` // polymarket, predictfun, hyperliquid
	URL          string  `json:"url"`
	ImageURL     string  `json:"imageUrl"`
	EndDate      int64   `json:"endDate"`
	CreatedAt    int64   `json:"createdAt"`
	UpdatedAt    int64   `json:"updatedAt"`
	Volume       float64 `json:"volume"`
	VolumeUSD    float64 `json:"volumeUSD"`
	OpenInterest float64 `json:"openInterest"`
	Active       bool    `json:"active"`
	Closed       bool    `json:"closed"`
	Resolved     bool    `json:"resolved"`
	Outcome      string  `json:"outcome,omitempty"`
}

// Outcome represents a market outcome
type Outcome struct {
	ID          string  `json:"id"`
	MarketID    string  `json:"marketId"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Probability float64 `json:"probability"`
	Volume      float64 `json:"volume"`
	YesVolume   float64 `json:"yesVolume,omitempty"`
	NoVolume    float64 `json:"noVolume,omitempty"`
	Active      bool    `json:"active"`
}

// Trade represents a prediction trade
type Trade struct {
	ID        string  `json:"id"`
	UserID    string  `json:"userId"`
	MarketID  string  `json:"marketId"`
	OutcomeID string  `json:"outcomeId"`
	Side      string  `json:"side"` // yes, no
	Amount    float64 `json:"amount"`
	Price     float64 `json:"price"`
	Total     float64 `json:"total"`
	ChainID   uint64  `json:"chainId"`
	TxHash    string  `json:"txHash,omitempty"`
	Status    string  `json:"status"` // pending, confirmed, failed
	CreatedAt int64   `json:"createdAt"`
}

// Position represents a user's position in a market
type Position struct {
	UserID       string  `json:"userId"`
	MarketID     string  `json:"marketId"`
	OutcomeID    string  `json:"outcomeId"`
	Quantity     float64 `json:"quantity"`
	AvgPrice     float64 `json:"avgPrice"`
	CurrentValue float64 `json:"currentValue"`
	ProfitLoss   float64 `json:"profitLoss"`
	UpdatedAt    int64   `json:"updatedAt"`
}

// ============================================================================
// API HANDLERS
// ============================================================================

func setupRouter(cfg *Config) *gin.Engine {
	r := gin.Default()

	// CORS middleware
	r.Use(corsMiddleware(cfg.WSOrigins))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
		})
	})

	// API v1
	v1 := r.Group("/api/v1")
	{
		// Markets
		markets := v1.Group("/markets")
		{
			markets.GET("", getMarkets)
			markets.GET("/:id", getMarket)
			markets.GET("/:id/outcomes", getOutcomes)
			markets.GET("/trending", getTrendingMarkets)
			markets.GET("/recent", getRecentMarkets)
		}

		// Trading. Every route here acts on an account, so every route
		// requires a verified token.
		trading := v1.Group("/trading", requireUser([]byte(cfg.JWTSecret)))
		{
			trading.POST("/buy", placeTrade)
			trading.POST("/sell", placeTrade)
			trading.GET("/history/:userId", getTradeHistory)
			trading.GET("/positions/:userId", getPositions)
		}

		// Prices
		prices := v1.Group("/prices")
		{
			prices.GET("/:marketId", getMarketPrices)
			prices.GET("/realtime/:marketId", streamPrices)
		}

		// Analytics. The aggregate volume is public; the per-user figures
		// are not.
		analytics := v1.Group("/analytics")
		{
			analytics.GET("/volume", getVolumeAnalytics)
			analytics.GET("/user/:userId", requireUser([]byte(cfg.JWTSecret)), getUserAnalytics)
		}
	}

	// WebSocket
	r.GET("/ws", handleWebSocket)

	return r
}

// CORS middleware
func corsMiddleware(origins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		allowed := false
		for _, o := range origins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}

		// Access is granted only to an origin that is explicitly permitted.
		// An empty allow-list previously matched every origin while still
		// sending Allow-Credentials, which let any site read authenticated
		// responses.
		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			c.Header("Access-Control-Allow-Credentials", "true")
		} else if origin != "" {
			logger.Warn().Str("origin", origin).Msg("rejected a request from an origin that is not allowed")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// ============================================================================
// MARKET HANDLERS
// ============================================================================

// Market, trade and position handlers live in handlers_markets.go.

// Trading, price and analytics handlers live in handlers_markets.go.

// ============================================================================
// WEBSOCKET HUB
// ============================================================================

type WebSocketHub struct {
	// MarketID -> Clients
	marketClients map[string]map[*WebSocketClient]bool
	register      chan *WebSocketClient
	unregister    chan *WebSocketClient
	broadcast     chan *MarketBroadcast
	mutex         sync.RWMutex
}

type WebSocketClient struct {
	marketID string
	conn     *websocket.Conn
	send     chan []byte
}

type MarketBroadcast struct {
	MarketID string
	Message  map[string]interface{}
}

func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		marketClients: make(map[string]map[*WebSocketClient]bool),
		register:      make(chan *WebSocketClient),
		unregister:    make(chan *WebSocketClient),
		broadcast:     make(chan *MarketBroadcast),
	}
}

func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			if h.marketClients[client.marketID] == nil {
				h.marketClients[client.marketID] = make(map[*WebSocketClient]bool)
			}
			h.marketClients[client.marketID][client] = true
			h.mutex.Unlock()

		case client := <-h.unregister:
			h.mutex.Lock()
			if clients, ok := h.marketClients[client.marketID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.send)
				}
			}
			h.mutex.Unlock()

		case broadcast := <-h.broadcast:
			h.mutex.RLock()
			clients := h.marketClients[broadcast.MarketID]
			h.mutex.RUnlock()

			for client := range clients {
				select {
				case client.send <- mustJSON(broadcast.Message):
				default:
					close(client.send)
					h.mutex.Lock()
					delete(clients, client)
					h.mutex.Unlock()
				}
			}
		}
	}
}

func (h *WebSocketHub) Register(marketID string, conn *websocket.Conn) *WebSocketClient {
	client := &WebSocketClient{
		marketID: marketID,
		conn:     conn,
		send:     make(chan []byte, 256),
	}
	h.register <- client
	return client
}

func (h *WebSocketHub) Unregister(marketID string, client *WebSocketClient) {
	h.unregister <- client
}

func (h *WebSocketHub) BroadcastMarketUpdate(marketID string, message map[string]interface{}) {
	h.broadcast <- &MarketBroadcast{
		MarketID: marketID,
		Message:  message,
	}
}

func (c *WebSocketClient) ReadPump() {
	defer func() {
		c.conn.Close()
	}()
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *WebSocketClient) WritePump() {
	defer c.conn.Close()
	for {
		message, ok := <-c.send
		if !ok {
			c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}
		c.conn.WriteMessage(websocket.TextMessage, message)
	}
}

func handleWebSocket(c *gin.Context) {
	marketID := c.Query("market")

	// A WebSocket handshake is not covered by the CORS policy, so the origin
	// has to be checked here. Returning true unconditionally allowed any page
	// to open an authenticated socket against this service.
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return originAllowed(config.WSOrigins, r.Header.Get("Origin"))
		},
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := wsHub.Register(marketID, conn)
	defer wsHub.Unregister(marketID, client)

	go client.WritePump()
	client.ReadPump()
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// generateID returns a random identifier.
//
// crypto/rand is used rather than hashing the clock, because two identifiers
// requested in the same nanosecond would otherwise be identical.
func generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// A failing entropy source must not silently produce a duplicate id.
		panic("crypto/rand is unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}

func mustJSON(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}

func getIntParam(c *gin.Context, name string, def int) int {
	if val := c.Query(name); val != "" {
		var parsed int
		if _, err := fmt.Sscanf(val, "%d", &parsed); err == nil {
			return parsed
		}
	}
	return def
}
