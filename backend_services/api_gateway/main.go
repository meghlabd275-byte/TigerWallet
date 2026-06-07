package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

// ============================================================================
// TIGERWALLET API GATEWAY
// Main entry point for all client requests
// ============================================================================

var (
	logger zerolog.Logger
	redisClient *redis.Client
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

// Configuration
type Config struct {
	Port            string
	RedisURL        string
	WalletServiceURL string
	SwapServiceURL  string
	StakingServiceURL string
}

// ============================================================================
// MAIN
// ============================================================================

func main() {
	// Initialize logger
	logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	logger.Info().Msg("Starting TigerWallet API Gateway")

	// Load configuration
	cfg := loadConfig()

	// Initialize Redis
	redisClient = initRedis(cfg.RedisURL)
	defer redisClient.Close()

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
			logger.Fatal().Err(err).Msg("Server error")
		}
	}()

	logger.Info().Msgf("Server started on port %s", cfg.Port)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	logger.Info().Msg("Server exited")
}

// ============================================================================
// ROUTER SETUP
// ============================================================================

func setupRouter(cfg *Config) *gin.Engine {
	router := gin.Default()

	// CORS middleware
	router.Use(corsMiddleware())

	// Rate limiting middleware
	router.Use(rateLimitMiddleware())

	// Health check
	router.GET("/health", healthCheck)

	// API v1
	v1 := router.Group("/api/v1")
	{
		// Wallet routes
		wallet := v1.Group("/wallet")
		{
			wallet.POST("/create", createWallet)
			wallet.POST("/import", importWallet)
			wallet.POST("/export", exportWallet)
			wallet.GET("/:address", getWalletInfo)
			wallet.GET("/:address/balance", getBalance)
			wallet.POST("/transfer", transfer)
		}

		// Swap routes
		swap := v1.Group("/swap")
		{
			swap.GET("/quote", getSwapQuote)
			swap.POST("/execute", executeSwap)
			swap.GET("/routes", getSwapRoutes)
		}

		// Staking routes
		staking := v1.Group("/staking")
		{
			staking.GET("/:chain/validators", getValidators)
			staking.POST("/delegate", delegate)
			staking.POST("/undelegate", undelegate)
			staking.GET("/rewards", getStakingRewards)
		}

		// NFT routes
		nft := v1.Group("/nft")
		{
			nft.GET("/:address", getNFTs)
			nft.GET("/:address/:token_id", getNFTDetails)
			nft.POST("/transfer", transferNFT)
		}

		// Bridge routes
		bridge := v1.Group("/bridge")
		{
			bridge.GET("/quote", getBridgeQuote)
			bridge.POST("/execute", executeBridge)
		}

		// Market data
		market := v1.Group("/market")
		{
			market.GET("/price/:token", getPrice)
			market.GET("/prices", getPrices)
			market.GET("/charts/:token", getPriceChart)
		}

		// Portfolio
		portfolio := v1.Group("/portfolio")
		{
			portfolio.GET("/:address", getPortfolio)
			portfolio.GET("/:address/history", getPortfolioHistory)
		}
	}

	// WebSocket for real-time updates
	router.GET("/ws", handleWebSocket)

	return router
}

// ============================================================================
// MIDDLEWARE
// ============================================================================

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func rateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client IP
		ip := c.ClientIP()

		// Check rate limit in Redis
		key := fmt.Sprintf("ratelimit:%s", ip)
		count, err := redisClient.Incr(context.Background(), key).Result()
		if err != nil {
			logger.Error().Err(err).Msg("Rate limit check failed")
			c.Next()
			return
		}

		// Set expiry on first request
		if count == 1 {
			redisClient.Expire(context.Background(), key, time.Minute)
		}

		// Rate limit: 100 requests per minute
		if count > 100 {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ============================================================================
// HANDLERS
// ============================================================================

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
	})
}

// ============================================================================
// WALLET HANDLERS
// ============================================================================

type CreateWalletRequest struct {
	ChainID uint64 `json:"chain_id" binding:"required"`
	Type    string `json:"type" binding:"required"` // "mnemonic", "private_key"
	Password string `json:"password" binding:"required"`
}

type CreateWalletResponse struct {
	Address    string   `json:"address"`
	PublicKey  string   `json:"public_key"`
	Mnemonic   string   `json:"mnemonic,omitempty"`
	ChainID    uint64  `json:"chain_id"`
	CreatedAt  int64   `json:"created_at"`
}

func createWallet(c *gin.Context) {
	var req CreateWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// In production, this would call the wallet service
	// For now, return mock data
	response := CreateWalletResponse{
		Address:   "0x" + generateRandomHex(40),
		PublicKey: "0x" + generateRandomHex(66),
		Mnemonic:  generateMnemonic(),
		ChainID:   req.ChainID,
		CreatedAt: time.Now().Unix(),
	}

	// Cache in Redis
	cacheKey := fmt.Sprintf("wallet:%s", response.Address)
	redisClient.Set(context.Background(), cacheKey, response.Address, 24*time.Hour)

	c.JSON(http.StatusCreated, response)
}

type ImportWalletRequest struct {
	Mnemonic   string `json:"mnemonic"`
	PrivateKey string `json:"private_key"`
	Password   string `json:"password"`
	ChainID    uint64 `json:"chain_id"`
}

func importWallet(c *gin.Context) {
	var req ImportWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response := CreateWalletResponse{
		Address:   "0x" + generateRandomHex(40),
		PublicKey: "0x" + generateRandomHex(66),
		ChainID:   req.ChainID,
		CreatedAt: time.Now().Unix(),
	}

	c.JSON(http.StatusCreated, response)
}

func getWalletInfo(c *gin.Context) {
	address := c.Param("address")

	// Check cache
	cacheKey := fmt.Sprintf("wallet:%s", address)
	cached, err := redisClient.Get(context.Background(), cacheKey).Result()
	if err == nil && cached != "" {
		c.JSON(http.StatusOK, gin.H{
			"address":    address,
			"chain_id":  1,
			"created_at": time.Now().Add(-30 * 24 * time.Hour).Unix(),
		})
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Wallet not found"})
}

type BalanceResponse struct {
	Address   string            `json:"address"`
	ChainID   uint64           `json:"chain_id"`
	Native    string           `json:"native"`
	Tokens    map[string]TokenBalance `json:"tokens"`
	Timestamp int64            `json:"timestamp"`
}

type TokenBalance struct {
	Balance   string `json:"balance"`
	Symbol    string `json:"symbol"`
	Decimals  uint8  `json:"decimals"`
	ValueUSD  string `json:"value_usd"`
}

func getBalance(c *gin.Context) {
	address := c.Param("address")
	chainID := c.Query("chain_id")

	response := BalanceResponse{
		Address:   address,
		ChainID:   1,
		Native:    "1000000000000000000", // 1 ETH in wei
		Tokens:    make(map[string]TokenBalance),
		Timestamp: time.Now().Unix(),
	}

	// Add some sample tokens
	response.Tokens["0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"] = TokenBalance{
		Balance:  "1000000000", // 1000 USDC
		Symbol:   "USDC",
		Decimals: 6,
		ValueUSD: "1000.00",
	}

	c.JSON(http.StatusOK, response)
}

type TransferRequest struct {
	From     string `json:"from" binding:"required"`
	To       string `json:"to" binding:"required"`
	Amount   string `json:"amount" binding:"required"`
	ChainID  uint64 `json:"chain_id" binding:"required"`
	Token    string `json:"token"` // empty for native
	GasPrice string `json:"gas_price"`
	Nonce    uint64 `json:"nonce"`
}

type TransferResponse struct {
	TxHash   string `json:"tx_hash"`
	From     string `json:"from"`
	To       string `json:"to"`
	Amount   string `json:"amount"`
	ChainID  uint64 `json:"chain_id"`
	Status   string `json:"status"`
}

func transfer(c *gin.Context) {
	var req TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// In production: sign and broadcast transaction
	response := TransferResponse{
		TxHash:  "0x" + generateRandomHex(64),
		From:    req.From,
		To:      req.To,
		Amount:  req.Amount,
		ChainID: req.ChainID,
		Status:  "pending",
	}

	// Cache transaction
	txKey := fmt.Sprintf("tx:%s", response.TxHash)
	txJSON, _ := json.Marshal(response)
	redisClient.Set(context.Background(), txKey, txJSON, 24*time.Hour)

	c.JSON(http.StatusCreated, response)
}

// ============================================================================
// SWAP HANDLERS
// ============================================================================

type SwapQuoteRequest struct {
	FromToken string  `json:"from_token" binding:"required"`
	ToToken   string  `json:"to_token" binding:"required"`
	Amount    string  `json:"amount" binding:"required"`
	Slippage  float64 `json:"slippage"`
	ChainID   uint64  `json:"chain_id" binding:"required"`
}

type SwapRoute struct {
	DEX       string `json:"dex"`
	FromToken string `json:"from_token"`
	ToToken   string `json:"to_token"`
	FromAmount string `json:"from_amount"`
	ToAmount  string `json:"to_amount"`
}

type SwapQuoteResponse struct {
	FromToken    string      `json:"from_token"`
	ToToken      string      `json:"to_token"`
	FromAmount   string      `json:"from_amount"`
	ToAmount     string      `json:"to_amount"`
	MinReceived  string      `json:"min_received"`
	Route        []SwapRoute `json:"route"`
	GasEstimate  string      `json:"gas_estimate"`
	PriceImpact  float64     `json:"price_impact"`
}

func getSwapQuote(c *gin.Context) {
	var req SwapQuoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Mock quote - in production call swap service
	response := SwapQuoteResponse{
		FromToken:    req.FromToken,
		ToToken:      req.ToToken,
		FromAmount:   req.Amount,
		ToAmount:     "1500000000000000000", // 1.5x output
		MinReceived:  "1485000000000000000", // with slippage
		Route: []SwapRoute{
			{
				DEX:        "uniswap_v3",
				FromToken:  req.FromToken,
				ToToken:    req.ToToken,
				FromAmount: req.Amount,
				ToAmount:   "1500000000000000000",
			},
		},
		GasEstimate: "150000",
		PriceImpact: 0.5,
	}

	c.JSON(http.StatusOK, response)
}

type ExecuteSwapRequest struct {
	FromToken  string `json:"from_token" binding:"required"`
	ToToken    string `json:"to_token" binding:"required"`
	Amount     string `json:"amount" binding:"required"`
	MinReceive string `json:"min_receive" binding:"required"`
	FromAddress string `json:"from_address" binding:"required"`
	ChainID    uint64 `json:"chain_id" binding:"required"`
	Slippage   float64 `json:"slippage"`
}

func executeSwap(c *gin.Context) {
	var req ExecuteSwapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tx_hash":   "0x" + generateRandomHex(64),
		"from":      req.FromAddress,
		"from_token": req.FromToken,
		"to_token":  req.ToToken,
		"status":    "pending",
	})
}

func getSwapRoutes(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"routes": []string{
			"uniswap_v3",
			"uniswap_v2",
			"sushiswap",
			"curve",
			"balancer",
			"pancakeswap",
		},
	})
}

// ============================================================================
// STAKING HANDLERS
// ============================================================================

type Validator struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Commission  float64 `json:"commission"`
	StakedAmount string `json:"staked_amount"`
	APY         float64 `json:"apy"`
	Status      string  `json:"status"`
}

func getValidators(c *gin.Context) {
	chain := c.Param("chain")

	response := []Validator{
		{
			ID:          "validator_1",
			Name:        "Lido",
			Commission:  10.0,
			StakedAmount: "1000000000000000000000",
			APY:         4.5,
			Status:      "active",
		},
		{
			ID:          "validator_2",
			Name:        "Rocket Pool",
			Commission:  15.0,
			StakedAmount: "500000000000000000000",
			APY:         4.2,
			Status:      "active",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"chain":     chain,
		"validators": response,
	})
}

type DelegateRequest struct {
	ValidatorID string `json:"validator_id" binding:"required"`
	Amount     string `json:"amount" binding:"required"`
	ChainID    uint64 `json:"chain_id" binding:"required"`
}

func delegate(c *gin.Context) {
	var req DelegateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tx_hash":      "0x" + generateRandomHex(64),
		"validator_id": req.ValidatorID,
		"amount":       req.Amount,
		"status":       "pending",
	})
}

func undelegate(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"tx_hash": "0x" + generateRandomHex(64),
		"status":  "pending",
	})
}

func getStakingRewards(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"rewards": []gin.H{
			{
				"amount":    "100000000000000000",
				"timestamp": time.Now().Unix(),
				"chain_id":  1,
			},
		},
	})
}

// ============================================================================
// NFT HANDLERS
// ============================================================================

type NFT struct {
	TokenID    string   `json:"token_id"`
	Contract   string   `json:"contract"`
	Name       string   `json:"name"`
	Symbol     string   `json:"symbol"`
	ImageURL   string   `json:"image_url"`
	Owner      string   `json:"owner"`
	Attributes []gin.H  `json:"attributes"`
}

func getNFTs(c *gin.Context) {
	address := c.Param("address")

	response := []NFT{
		{
			TokenID:  "1",
			Contract: "0xBC4CA0EdA7647A8aB7C2061c2E118A18a936f13D",
			Name:     "Bored Ape Yacht Club",
			Symbol:   "BAYC",
			ImageURL: "https://ipfs.io/ipfs/QmRRPWG96cmgTn2qSzjwr2qvfNEuhunv6FNeMFGa9bx6mQ",
			Owner:    address,
			Attributes: []gin.H{
				{"trait_type": "Background", "value": "Blue"},
				{"trait_type": "Fur", "value": "Dark Brown"},
			},
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"address": address,
		"nfts":    response,
		"total":   1,
	})
}

func getNFTDetails(c *gin.Context) {
	tokenID := c.Param("token_id")

	c.JSON(http.StatusOK, gin.H{
		"token_id": tokenID,
		"name":     "Bored Ape #1",
		"image_url": "https://ipfs.io/ipfs/QmRRPWG96cmgTn2qSzjwr2qvfNEuhunv6FNeMFGa9bx6mQ",
	})
}

func transferNFT(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"tx_hash": "0x" + generateRandomHex(64),
		"status":  "pending",
	})
}

// ============================================================================
// BRIDGE HANDLERS
// ============================================================================

type BridgeQuoteRequest struct {
	FromChain uint64 `json:"from_chain" binding:"required"`
	ToChain   uint64 `json:"to_chain" binding:"required"`
	Token     string `json:"token" binding:"required"`
	Amount    string `json:"amount" binding:"required"`
}

func getBridgeQuote(c *gin.Context) {
	var req BridgeQuoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"from_chain":     req.FromChain,
		"to_chain":       req.ToChain,
		"from_amount":    req.Amount,
		"to_amount":      "980000000000000000", // 2% fee
		"fee":            "20000000000000000",
		"estimated_time":  "15m",
		"protocol":       "stargate",
	})
}

func executeBridge(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"tx_hash":      "0x" + generateRandomHex(64),
		"status":       "pending",
		"bridge_tx_id": generateRandomHex(32),
	})
}

// ============================================================================
// MARKET DATA HANDLERS
// ============================================================================

func getPrice(c *gin.Context) {
	token := c.Param("token")

	// Check cache first
	cacheKey := fmt.Sprintf("price:%s", token)
	cached, err := redisClient.Get(context.Background(), cacheKey).Result()
	if err == nil && cached != "" {
		c.JSON(http.StatusOK, gin.H{
			"token":  token,
			"price":  cached,
			"change": "2.5",
		})
		return
	}

	// Mock price - in production fetch from API
	price := "2500.00"
	redisClient.Set(context.Background(), cacheKey, price, 60*time.Second)

	c.JSON(http.StatusOK, gin.H{
		"token":  token,
		"price":  price,
		"change": "2.5",
	})
}

func getPrices(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"prices": gin.H{
			"ETH": "2500.00",
			"BTC": "45000.00",
			"USDC": "1.00",
			"MATIC": "0.85",
		},
	})
}

func getPriceChart(c *gin.Context) {
	token := c.Param("token")

	// Generate mock chart data
	points := make([]gin.H, 24)
	for i := 0; i < 24; i++ {
		points[i] = gin.H{
			"timestamp": time.Now().Add(-time.Duration(24-i) * time.Hour).Unix(),
			"price":     2500 + float64(i)*10,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"token":  token,
		"chart":  points,
		"period": "24h",
	})
}

// ============================================================================
// PORTFOLIO HANDLERS
// ============================================================================

type PortfolioAsset struct {
	Token       string `json:"token"`
	Balance     string `json:"balance"`
	ValueUSD    string `json:"value_usd"`
	Allocation  float64 `json:"allocation"`
	Change24h   float64 `json:"change_24h"`
}

func getPortfolio(c *gin.Context) {
	address := c.Param("address")

	response := gin.H{
		"address": address,
		"total_value_usd": "12500.00",
		"assets": []PortfolioAsset{
			{
				Token:      "ETH",
				Balance:    "4.0",
				ValueUSD:   "10000.00",
				Allocation: 80.0,
				Change24h:  2.5,
			},
			{
				Token:      "USDC",
				Balance:    "2500.00",
				ValueUSD:   "2500.00",
				Allocation: 20.0,
				Change24h:  0.0,
			},
		},
		"timestamp": time.Now().Unix(),
	}

	c.JSON(http.StatusOK, response)
}

func getPortfolioHistory(c *gin.Context) {
	address := c.Param("address")

	// Generate mock history
	points := make([]gin.H, 30)
	for i := 0; i < 30; i++ {
		points[i] = gin.H{
			"timestamp":     time.Now().Add(-time.Duration(30-i) * 24 * time.Hour).Unix(),
			"total_value":   12000 + float64(i)*20,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"address":  address,
		"history":  points,
		"period":   "30d",
	})
}

// ============================================================================
// WEBSOCKET HANDLER
// ============================================================================

func handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Error().Err(err).Msg("WebSocket upgrade failed")
		return
	}
	defer conn.Close()

	logger.Info().Msg("WebSocket client connected")

	// Send initial data
	conn.WriteJSON(gin.H{
		"type":    "connected",
		"message": "Welcome to TigerWallet",
	})

	// Handle messages
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			logger.Error().Err(err).Msg("WebSocket read error")
			break
		}

		// Handle subscription
		var sub map[string]interface{}
		if err := json.Unmarshal(msg, &sub); err == nil {
			if subType, ok := sub["type"].(string); ok {
				switch subType {
				case "subscribe":
					conn.WriteJSON(gin.H{"type": "subscribed", "channel": sub["channel"]})
				case "ping":
					conn.WriteJSON(gin.H{"type": "pong"})
				}
			}
		}

		// Simulate price updates
		conn.WriteJSON(gin.H{
			"type":  "price_update",
			"token": "ETH",
			"price": "2500.00",
		})
	}
}

// ============================================================================
// HELPERS
// ============================================================================

func loadConfig() *Config {
	return &Config{
		Port:            getEnv("PORT", "8080"),
		RedisURL:        getEnv("REDIS_URL", "redis://localhost:6379"),
		WalletServiceURL: getEnv("WALLET_SERVICE_URL", "http://localhost:8081"),
		SwapServiceURL:  getEnv("SWAP_SERVICE_URL", "http://localhost:8082"),
	}
}

func initRedis(url string) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		logger.Warn().Err(err).Msg("Redis connection failed, using in-memory fallback")
	}

	return client
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func generateRandomHex(length int) string {
	bytes := make([]byte, length/2)
	for i := range bytes {
		bytes[i] = byte(time.Now().UnixNano() % 256)
		time.Sleep(time.Nanosecond)
	}
	return hex.EncodeToString(bytes)[:length]
}

func generateMnemonic() string {
	words := []string{
		"abandon", "ability", "able", "about", "above", "absent", "absorb", "abstract",
		"absurd", "abuse", "access", "accident", "account", "accuse", "achieve",
		"acid", "acoustic", "acquire", "across", "act", "action", "actor", "actress",
	}
	mnemonic := ""
	for i := 0; i < 24; i++ {
		if i > 0 {
			mnemonic += " "
		}
		mnemonic += words[i%len(words)]
	}
	return mnemonic
}
