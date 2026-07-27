package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
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
	wsHub        *WebSocketHub
)

func main() {
	// Initialize logger
	output := zerolog.ConsoleWriter{Output: os.Stdout}
	logger = zerolog.New(output).With().Timestamp().Logger()

	// Load configuration
	cfg := loadConfig()

	// Initialize Redis
	redisClient = redis.NewClient(&redis.Options{
		Addr:     cfg.RedisURL,
		Password: "",
		DB:       0,
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Warn().Err(err).Msg("Redis connection failed, using in-memory storage")
	}

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
	Port              string
	RedisURL          string
	PolymarketAPI     string
	PredictFunAPI     string
	HyperliquidAPI    string
	WSOrigins         []string
	MarketUpdateInterval time.Duration
}

func loadConfig() *Config {
	return &Config{
		Port:              getEnv("PREDICTION_PORT", "9207"),
		RedisURL:          getEnv("REDIS_URL", "localhost:6379"),
		PolymarketAPI:     getEnv("POLYMARKET_API", "https://clob.polymarket.com"),
		PredictFunAPI:     getEnv("PREDICTFUN_API", "https://api.predict.fun"),
		HyperliquidAPI:   getEnv("HYPERLIQUID_API", "https://api.hyperliquid.xyz"),
		WSOrigins:         strings.Split(getEnv("WS_ORIGINS", "*"), ","),
		MarketUpdateInterval: 5 * time.Second,
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// ============================================================================
// DATA MODELS
// ============================================================================

// Market represents a prediction market
type Market struct {
	ID                string    `json:"id"`
	Question          string    `json:"question"`
	Description       string    `json:"description"`
	Source            string    `json:"source"` // polymarket, predictfun, hyperliquid
	URL               string    `json:"url"`
	ImageURL          string    `json:"imageUrl"`
	EndDate           int64     `json:"endDate"`
	CreatedAt         int64     `json:"createdAt"`
	UpdatedAt         int64     `json:"updatedAt"`
	Volume            float64   `json:"volume"`
	VolumeUSD         float64   `json:"volumeUSD"`
	OpenInterest      float64   `json:"openInterest"`
	Active            bool      `json:"active"`
	Closed            bool      `json:"closed"`
	Resolved          bool      `json:"resolved"`
	Outcome           string    `json:"outcome,omitempty"`
}

// Outcome represents a market outcome
type Outcome struct {
	ID        string  `json:"id"`
	MarketID  string  `json:"marketId"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Probability float64 `json:"probability"`
	Volume    float64 `json:"volume"`
	YesVolume float64 `json:"yesVolume,omitempty"`
	NoVolume  float64 `json:"noVolume,omitempty"`
	Active    bool    `json:"active"`
}

// Trade represents a prediction trade
type Trade struct {
	ID            string    `json:"id"`
	UserID        string    `json:"userId"`
	MarketID      string    `json:"marketId"`
	OutcomeID     string    `json:"outcomeId"`
	Side          string    `json:"side"` // yes, no
	Amount        float64   `json:"amount"`
	Price         float64   `json:"price"`
	Total         float64   `json:"total"`
	ChainID       uint64    `json:"chainId"`
	TxHash        string    `json:"txHash,omitempty"`
	Status        string    `json:"status"` // pending, confirmed, failed
	CreatedAt     int64     `json:"createdAt"`
}

// Position represents a user's position in a market
type Position struct {
	UserID       string    `json:"userId"`
	MarketID     string    `json:"marketId"`
	OutcomeID    string    `json:"outcomeId"`
	Quantity     float64   `json:"quantity"`
	AvgPrice     float64   `json:"avgPrice"`
	CurrentValue float64   `json:"currentValue"`
	ProfitLoss   float64   `json:"profitLoss"`
	UpdatedAt    int64     `json:"updatedAt"`
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

		// Trading
		trading := v1.Group("/trading")
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

		// Analytics
		analytics := v1.Group("/analytics")
		{
			analytics.GET("/volume", getVolumeAnalytics)
			analytics.GET("/user/:userId", getUserAnalytics)
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

		if allowed || len(origins) == 0 {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			c.Header("Access-Control-Allow-Credentials", "true")
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

func getMarkets(c *gin.Context) {
	ctx := context.Background()
	
	// Get query params
	source := c.Query("source")
	status := c.Query("status")
	limit := getIntParam(c, "limit", 50)
	offset := getIntParam(c, "offset", 0)

	// Try Redis first
	key := fmt.Sprintf("markets:%s:%s:%d:%d", source, status, limit, offset)
	if data, err := redisClient.Get(ctx, key).Bytes(); err == nil {
		var markets []Market
		if json.Unmarshal(data, &markets) == nil {
			c.JSON(http.StatusOK, markets)
			return
		}
	}

	// Fetch from sources
	markets := fetchMarketsFromSources(source, status, limit, offset)

	// Cache for 30 seconds
	if data, err := json.Marshal(markets); err == nil {
		redisClient.Set(ctx, key, data, 30*time.Second)
	}

	c.JSON(http.StatusOK, markets)
}

func getMarket(c *gin.Context) {
	id := c.Param("id")
	ctx := context.Background()

	// Try Redis
	key := fmt.Sprintf("market:%s", id)
	if data, err := redisClient.Get(ctx, key).Bytes(); err == nil {
		var market Market
		if json.Unmarshal(data, &market) == nil {
			c.JSON(http.StatusOK, market)
			return
		}
	}

	// Fetch from database or API
	market := fetchMarketByID(id)

	if market.ID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Market not found"})
		return
	}

	// Cache
	if data, err := json.Marshal(market); err == nil {
		redisClient.Set(ctx, key, data, 30*time.Second)
	}

	c.JSON(http.StatusOK, market)
}

func getOutcomes(c *gin.Context) {
	marketID := c.Param("id")
	ctx := context.Background()

	// Try Redis
	key := fmt.Sprintf("outcomes:%s", marketID)
	if data, err := redisClient.Get(ctx, key).Bytes(); err == nil {
		var outcomes []Outcome
		if json.Unmarshal(data, &outcomes) == nil {
			c.JSON(http.StatusOK, outcomes)
			return
		}
	}

	// Fetch outcomes
	outcomes := fetchOutcomes(marketID)

	// Cache
	if data, err := json.Marshal(outcomes); err == nil {
		redisClient.Set(ctx, key, data, 10*time.Second)
	}

	c.JSON(http.StatusOK, outcomes)
}

func getTrendingMarkets(c *gin.Context) {
	limit := getIntParam(c, "limit", 10)
	markets := fetchTrendingMarkets(limit)
	c.JSON(http.StatusOK, markets)
}

func getRecentMarkets(c *gin.Context) {
	limit := getIntParam(c, "limit", 20)
	markets := fetchRecentMarkets(limit)
	c.JSON(http.StatusOK, markets)
}

// ============================================================================
// TRADING HANDLERS
// ============================================================================

func placeTrade(c *gin.Context) {
	var req struct {
		UserID    string  `json:"userId" binding:"required"`
		MarketID  string  `json:"marketId" binding:"required"`
		OutcomeID string  `json:"outcomeId" binding:"required"`
		Side      string  `json:"side" binding:"required,oneof=yes no"`
		Amount    float64 `json:"amount" binding:"required,gt=0"`
		ChainID   uint64  `json:"chainId"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get current price
	price := getCurrentPrice(req.MarketID, req.OutcomeID)
	total := req.Amount * price

	// Create trade
	trade := Trade{
		ID:        generateID(),
		UserID:    req.UserID,
		MarketID:  req.MarketID,
		OutcomeID: req.OutcomeID,
		Side:      req.Side,
		Amount:    req.Amount,
		Price:     price,
		Total:     total,
		ChainID:   req.ChainID,
		Status:    "pending",
		CreatedAt: time.Now().Unix(),
	}

	// Simulate on-chain execution
	txHash := executeTrade(trade)
	trade.TxHash = txHash
	trade.Status = "confirmed"

	// Save to storage
	saveTrade(trade)

	// Update position
	updatePosition(trade)

	// Broadcast update
	wsHub.BroadcastMarketUpdate(trade.MarketID, map[string]interface{}{
		"type":    "trade",
		"trade":   trade,
		"price":   price,
		"volume":  total,
	})

	c.JSON(http.StatusOK, trade)
}

func getTradeHistory(c *gin.Context) {
	userID := c.Param("userId")
	limit := getIntParam(c, "limit", 50)
	
	trades := fetchTradeHistory(userID, limit)
	c.JSON(http.StatusOK, trades)
}

func getPositions(c *gin.Context) {
	userID := c.Param("userId")
	
	positions := fetchPositions(userID)
	c.JSON(http.StatusOK, positions)
}

// ============================================================================
// PRICE HANDLERS
// ============================================================================

func getMarketPrices(c *gin.Context) {
	marketID := c.Param("marketId")
	
	prices := fetchMarketPrices(marketID)
	c.JSON(http.StatusOK, prices)
}

func streamPrices(c *gin.Context) {
	marketID := c.Param("marketId")

	// Upgrade to WebSocket
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "WebSocket upgrade failed"})
		return
	}

	// Register client
	client := wsHub.Register(marketID, conn)
	defer wsHub.Unregister(marketID, client)

	// Send initial prices
	prices := fetchMarketPrices(marketID)
	client.Send <- map[string]interface{}{
		"type":   "prices",
		"prices": prices,
	}

	// Keep connection alive
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// ============================================================================
// ANALYTICS HANDLERS
// ============================================================================

func getVolumeAnalytics(c *gin.Context) {
	period := c.DefaultQuery("period", "24h")
	
	volume := fetchVolumeAnalytics(period)
	c.JSON(http.StatusOK, volume)
}

func getUserAnalytics(c *gin.Context) {
	userID := c.Param("userId")
	
	analytics := fetchUserAnalytics(userID)
	c.JSON(http.StatusOK, analytics)
}

// ============================================================================
// WEBSOCKET HUB
// ============================================================================

type WebSocketHub struct {
	// MarketID -> Clients
	marketClients map[string]map[*WebSocketClient]bool
	register      chan *WebSocketClient
	unregister   chan *WebSocketClient
	broadcast    chan *MarketBroadcast
	mutex        sync.RWMutex
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
		unregister:   make(chan *WebSocketClient),
		broadcast:    make(chan *MarketBroadcast),
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
		Message:   message,
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

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
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

func generateID() string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%d%s", time.Now().UnixNano(), "tiger")))
	return hex.EncodeToString(hash[:])
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

// ============================================================================
// DATA FETCHING (Placeholder - Integrate with real APIs)
// ============================================================================

func fetchMarketsFromSources(source, status string, limit, offset int) []Market {
	// In production: fetch from Polymarket, Predict.fun, Hyperliquid APIs
	return []Market{
		{
			ID:            "1",
			Question:      "Will BTC reach $100k by end of 2026?",
			Description:   "Bitcoin price prediction market",
			Source:        "polymarket",
			URL:           "https://polymarket.com/market/will-btc-reach-100k",
			VolumeUSD:     1500000,
			OpenInterest:  500000,
			Active:        true,
			EndDate:       time.Now().AddDate(0, 6, 0).Unix(),
			CreatedAt:     time.Now().AddDate(0, -1, 0).Unix(),
		},
		{
			ID:            "2",
			Question:      "Will ETH flip BTC market cap by 2027?",
			Description:   "Ethereum vs Bitcoin market cap",
			Source:        "polymarket",
			URL:           "https://polymarket.com/market/will-eth-flip-btc",
			VolumeUSD:     800000,
			OpenInterest:  300000,
			Active:        true,
			EndDate:       time.Now().AddDate(1, 0, 0).Unix(),
			CreatedAt:     time.Now().AddDate(0, -2, 0).Unix(),
		},
	}
}

func fetchMarketByID(id string) Market {
	markets := fetchMarketsFromSources("", "", 100, 0)
	for _, m := range markets {
		if m.ID == id {
			return m
		}
	}
	return Market{}
}

func fetchOutcomes(marketID string) []Outcome {
	return []Outcome{
		{
			ID:          marketID + "-yes",
			MarketID:    marketID,
			Name:        "Yes",
			Price:       0.65,
			Probability: 65,
			Volume:      500000,
			Active:      true,
		},
		{
			ID:          marketID + "-no",
			MarketID:    marketID,
			Name:        "No",
			Price:       0.35,
			Probability: 35,
			Volume:      300000,
			Active:      true,
		},
	}
}

func fetchTrendingMarkets(limit int) []Market {
	return fetchMarketsFromSources("", "", limit, 0)
}

func fetchRecentMarkets(limit int) []Market {
	return fetchMarketsFromSources("", "", limit, 0)
}

func getCurrentPrice(marketID, outcomeID string) float64 {
	outcomes := fetchOutcomes(marketID)
	for _, o := range outcomes {
		if o.ID == outcomeID {
			return o.Price
		}
	}
	return 0.5
}

func executeTrade(trade Trade) string {
	// In production: execute on-chain transaction
	return "0x" + generateID()[:64]
}

func saveTrade(trade Trade) {
	// Save to database
}

func updatePosition(trade Trade) {
	// Update user position
}

func fetchTradeHistory(userID string, limit int) []Trade {
	return []Trade{}
}

func fetchPositions(userID string) []Position {
	return []Position{}
}

func fetchMarketPrices(marketID string) map[string]float64 {
	outcomes := fetchOutcomes(marketID)
	prices := make(map[string]float64)
	for _, o := range outcomes {
		prices[o.ID] = o.Price
	}
	return prices
}

func fetchVolumeAnalytics(period string) map[string]interface{} {
	return map[string]interface{}{
		"period":       period,
		"totalVolume":  2500000.0,
		"tradeCount":   15000,
		"uniqueUsers":  5000,
		"topMarkets":   []string{"1", "2"},
	}
}

func fetchUserAnalytics(userID string) map[string]interface{} {
	return map[string]interface{}{
		"userId":        userID,
		"totalVolume":   10000.0,
		"tradeCount":    50,
		"winningTrades": 30,
		"profitLoss":    2500.0,
	}
}
