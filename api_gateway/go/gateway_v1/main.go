package main

import (
	"bytes"
	"context"
	cryptoRand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
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
	logger      zerolog.Logger
	redisClient *redis.Client
	cfg         Config
	upgrader    = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

// Configuration
type Config struct {
	Port              string
	RedisURL          string
	WalletServiceURL  string
	MarketDataURL     string
	PortfolioURL      string
	SwapServiceURL    string
	StakingServiceURL string
	NFTServiceURL     string
	BridgeServiceURL  string
}

// ============================================================================
// MAIN
// ============================================================================

func main() {
	// Initialize logger
	logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	logger.Info().Msg("Starting TigerWallet API Gateway")

	// Load configuration
	cfg = *loadConfig()

	// Initialize Redis
	redisClient = initRedis(cfg.RedisURL)
	defer redisClient.Close()

	// Setup router
	router := setupRouter(&cfg)

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
			wallet.POST("/create", createWallet(cfg))
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
			market.GET("/price/:token", getPrice(cfg))
			market.GET("/prices", getPrices(cfg))
			market.GET("/charts/:token", getPriceChart(cfg))
		}

		// Portfolio
		portfolio := v1.Group("/portfolio")
		{
			portfolio.GET("/:address", getPortfolio(cfg))
			portfolio.GET("/:address/history", getPortfolioHistory(cfg))
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
	ChainID  uint64 `json:"chain_id" binding:"required"`
	Type     string `json:"type" binding:"required"` // "mnemonic", "private_key"
	Password string `json:"password" binding:"required"`
}

type CreateWalletResponse struct {
	Address   string `json:"address"`
	PublicKey string `json:"public_key"`
	Mnemonic  string `json:"mnemonic,omitempty"`
	ChainID   uint64 `json:"chain_id"`
	CreatedAt int64  `json:"created_at"`
}

func createWallet(cfg *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateWalletRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if cfg.WalletServiceURL == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "wallet service is not configured; refusing to create mock wallets",
			})
			return
		}

		body, err := json.Marshal(req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wallet creation request"})
			return
		}

		httpReq, err := http.NewRequestWithContext(
			c.Request.Context(),
			http.MethodPost,
			cfg.WalletServiceURL+"/wallet/create",
			bytes.NewReader(body),
		)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "wallet service request could not be created"})
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(httpReq)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "wallet service unavailable"})
			return
		}
		defer resp.Body.Close()

		var response map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "wallet service returned invalid JSON"})
			return
		}

		delete(response, "mnemonic")
		delete(response, "seed_phrase")
		delete(response, "private_key")

		if address, ok := response["address"].(string); ok && address != "" {
			cacheKey := fmt.Sprintf("wallet:%s", address)
			redisClient.Set(context.Background(), cacheKey, address, 24*time.Hour)
		}

		c.JSON(resp.StatusCode, response)
	}
}

type ImportWalletRequest struct {
	Mnemonic   string `json:"mnemonic"`
	PrivateKey string `json:"private_key"`
	Password   string `json:"password"`
	ChainID    uint64 `json:"chain_id"`
}

// importWallet proxies to the canonical wallet service. It never fabricates
// an address/mnemonic; if no upstream is configured or reachable it fails
// closed with an honest error.
func importWallet(c *gin.Context) {
	if cfg.WalletServiceURL == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "wallet service is not configured; refusing to fabricate an imported wallet",
		})
		return
	}
	proxyServicePOST(c, cfg.WalletServiceURL, "/wallet/import")
}

// exportWallet proxies a keystore-V3 export to the canonical wallet service.
// The wallet service derives the encrypted keystore from the stored seed; this
// gateway never fabricates export material.
func exportWallet(c *gin.Context) {
	if cfg.WalletServiceURL == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "wallet service is not configured; refusing to fabricate a wallet export",
		})
		return
	}
	proxyServicePOST(c, cfg.WalletServiceURL, "/wallet/export")
}

// getWalletInfo proxies wallet lookup to the canonical wallet service.
func getWalletInfo(c *gin.Context) {
	if cfg.WalletServiceURL == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "wallet service is not configured; refusing to return mock wallet info",
		})
		return
	}
	proxyServiceGET(c, cfg.WalletServiceURL, "/wallet/"+c.Param("address"))
}

type BalanceResponse struct {
	Address   string                  `json:"address"`
	ChainID   uint64                  `json:"chain_id"`
	Native    string                  `json:"native"`
	Tokens    map[string]TokenBalance `json:"tokens"`
	Timestamp int64                   `json:"timestamp"`
}

type TokenBalance struct {
	Balance  string `json:"balance"`
	Symbol   string `json:"symbol"`
	Decimals uint8  `json:"decimals"`
	ValueUSD string `json:"value_usd"`
}

// getBalance proxies a live balance read to the canonical wallet service.
func getBalance(c *gin.Context) {
	if cfg.WalletServiceURL == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "wallet service is not configured; refusing to return a fabricated balance",
		})
		return
	}
	proxyServiceGET(c, cfg.WalletServiceURL, "/wallet/"+c.Param("address")+"/balance")
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
	TxHash  string `json:"tx_hash"`
	From    string `json:"from"`
	To      string `json:"to"`
	Amount  string `json:"amount"`
	ChainID uint64 `json:"chain_id"`
	Status  string `json:"status"`
}

// transfer proxies the signed+broadcast transfer to the canonical wallet
// service. The wallet service is the sole key holder and returns the real
// on-chain tx hash; this gateway never invents one.
func transfer(c *gin.Context) {
	if cfg.WalletServiceURL == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "wallet service is not configured; refusing to fabricate a transfer tx hash",
		})
		return
	}
	proxyServicePOST(c, cfg.WalletServiceURL, "/wallet/transfer")
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
	DEX        string `json:"dex"`
	FromToken  string `json:"from_token"`
	ToToken    string `json:"to_token"`
	FromAmount string `json:"from_amount"`
	ToAmount   string `json:"to_amount"`
}

type SwapQuoteResponse struct {
	FromToken   string      `json:"from_token"`
	ToToken     string      `json:"to_token"`
	FromAmount  string      `json:"from_amount"`
	ToAmount    string      `json:"to_amount"`
	MinReceived string      `json:"min_received"`
	Route       []SwapRoute `json:"route"`
	GasEstimate string      `json:"gas_estimate"`
	PriceImpact float64     `json:"price_impact"`
}

// getSwapQuote proxies to the real swap service for a live DEX quote.
func getSwapQuote(c *gin.Context) {
	if cfg.SwapServiceURL == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "swap service is not configured; refusing to return a fabricated quote",
		})
		return
	}
	proxyServicePOST(c, cfg.SwapServiceURL, "/swap/quote")
}

type ExecuteSwapRequest struct {
	FromToken   string  `json:"from_token" binding:"required"`
	ToToken     string  `json:"to_token" binding:"required"`
	Amount      string  `json:"amount" binding:"required"`
	MinReceive  string  `json:"min_receive" binding:"required"`
	FromAddress string  `json:"from_address" binding:"required"`
	ChainID     uint64  `json:"chain_id" binding:"required"`
	Slippage    float64 `json:"slippage"`
}

// executeSwap proxies swap execution to the real swap service, which returns
// the on-chain action to broadcast via the canonical wallet service.
func executeSwap(c *gin.Context) {
	if cfg.SwapServiceURL == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "swap service is not configured; refusing to fabricate a swap tx hash",
		})
		return
	}
	proxyServicePOST(c, cfg.SwapServiceURL, "/swap/execute")
}

// getSwapRoutes proxies the available DEX routes from the real swap service.
func getSwapRoutes(c *gin.Context) {
	if cfg.SwapServiceURL == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "swap service is not configured; refusing to return fabricated routes",
		})
		return
	}
	proxyServiceGET(c, cfg.SwapServiceURL, "/swap/routes")
}

// ============================================================================
// STAKING HANDLERS
// ============================================================================

type Validator struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Commission   float64 `json:"commission"`
	StakedAmount string  `json:"staked_amount"`
	APY          float64 `json:"apy"`
	Status       string  `json:"status"`
}

// getValidators proxies validator list to the real staking service.
func getValidators(c *gin.Context) {
	if cfg.StakingServiceURL == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "staking service is not configured; refusing to return fabricated validators",
		})
		return
	}
	proxyServiceGET(c, cfg.StakingServiceURL, "/staking/"+c.Param("chain")+"/validators")
}

type DelegateRequest struct {
	ValidatorID string `json:"validator_id" binding:"required"`
	Amount      string `json:"amount" binding:"required"`
	ChainID     uint64 `json:"chain_id" binding:"required"`
}

// delegate proxies the stake delegation to the real staking service.
func delegate(c *gin.Context) {
	if cfg.StakingServiceURL == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "staking service is not configured; refusing to fabricate a delegation tx hash",
		})
		return
	}
	proxyServicePOST(c, cfg.StakingServiceURL, "/staking/delegate")
}

// undelegate proxies the unstake to the real staking service.
func undelegate(c *gin.Context) {
	if cfg.StakingServiceURL == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "staking service is not configured; refusing to fabricate an undelegation tx hash",
		})
		return
	}
	proxyServicePOST(c, cfg.StakingServiceURL, "/staking/undelegate")
}

// getStakingRewards proxies reward lookup to the real staking service.
func getStakingRewards(c *gin.Context) {
	if cfg.StakingServiceURL == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "staking service is not configured; refusing to return fabricated rewards",
		})
		return
	}
	proxyServiceGET(c, cfg.StakingServiceURL, "/staking/rewards")
}

// ============================================================================
// NFT HANDLERS
// ============================================================================

type NFT struct {
	TokenID    string  `json:"token_id"`
	Contract   string  `json:"contract"`
	Name       string  `json:"name"`
	Symbol     string  `json:"symbol"`
	ImageURL   string  `json:"image_url"`
	Owner      string  `json:"owner"`
	Attributes []gin.H `json:"attributes"`
}

// getNFTs proxies NFT enumeration to the real NFT service (on-chain reads).
func getNFTs(c *gin.Context) {
	if cfg.NFTServiceURL == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "NFT service is not configured; refusing to return fabricated NFTs",
		})
		return
	}
	proxyServiceGET(c, cfg.NFTServiceURL, "/nft/"+c.Param("address"))
}

// getNFTDetails proxies NFT metadata to the real NFT service.
func getNFTDetails(c *gin.Context) {
	if cfg.NFTServiceURL == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "NFT service is not configured; refusing to return fabricated NFT metadata",
		})
		return
	}
	proxyServiceGET(c, cfg.NFTServiceURL, "/nft/"+c.Param("address")+"/"+c.Param("token_id"))
}

// transferNFT proxies the ERC-721 transfer to the real wallet service.
func transferNFT(c *gin.Context) {
	if cfg.WalletServiceURL == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "wallet service is not configured; refusing to fabricate an NFT transfer tx hash",
		})
		return
	}
	proxyServicePOST(c, cfg.WalletServiceURL, "/nft/transfer")
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

// getBridgeQuote proxies a live bridge quote to the real bridge service.
func getBridgeQuote(c *gin.Context) {
	if cfg.BridgeServiceURL == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "bridge service is not configured; refusing to return a fabricated quote",
		})
		return
	}
	proxyServicePOST(c, cfg.BridgeServiceURL, "/bridge/quote")
}

// executeBridge proxies bridge execution to the real bridge service.
func executeBridge(c *gin.Context) {
	if cfg.BridgeServiceURL == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "bridge service is not configured; refusing to fabricate a bridge tx hash",
		})
		return
	}
	proxyServicePOST(c, cfg.BridgeServiceURL, "/bridge/execute")
}

// ============================================================================
// MARKET DATA HANDLERS
// ============================================================================

func getPrice(cfg *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Param("token")

		cacheKey := fmt.Sprintf("price:%s", token)
		cached, err := redisClient.Get(context.Background(), cacheKey).Result()
		if err == nil && cached != "" {
			var cachedPayload map[string]interface{}
			if err := json.Unmarshal([]byte(cached), &cachedPayload); err == nil {
				c.JSON(http.StatusOK, cachedPayload)
				return
			}
			redisClient.Del(context.Background(), cacheKey)
		}

		payload, ok := proxyServiceGET(c, cfg.MarketDataURL, "/price/"+token)
		if !ok {
			return
		}
		if encoded, err := json.Marshal(payload); err == nil {
			redisClient.Set(context.Background(), cacheKey, string(encoded), 60*time.Second)
		}
		c.JSON(http.StatusOK, payload)
	}
}

func getPrices(cfg *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		payload, ok := proxyServiceGET(c, cfg.MarketDataURL, "/prices")
		if !ok {
			return
		}
		c.JSON(http.StatusOK, payload)
	}
}

func getPriceChart(cfg *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Param("token")
		payload, ok := proxyServiceGET(c, cfg.MarketDataURL, "/charts/"+token)
		if !ok {
			return
		}
		c.JSON(http.StatusOK, payload)
	}
}

// ============================================================================
// PORTFOLIO HANDLERS
// ============================================================================

type PortfolioAsset struct {
	Token      string  `json:"token"`
	Balance    string  `json:"balance"`
	ValueUSD   string  `json:"value_usd"`
	Allocation float64 `json:"allocation"`
	Change24h  float64 `json:"change_24h"`
}

func getPortfolio(cfg *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		address := c.Param("address")
		payload, ok := proxyServiceGET(c, cfg.PortfolioURL, "/portfolio/"+address)
		if !ok {
			return
		}
		c.JSON(http.StatusOK, payload)
	}
}

func getPortfolioHistory(cfg *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		address := c.Param("address")
		payload, ok := proxyServiceGET(c, cfg.PortfolioURL, "/portfolio/"+address+"/history")
		if !ok {
			return
		}
		c.JSON(http.StatusOK, payload)
	}
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

		conn.WriteJSON(gin.H{
			"type":    "heartbeat",
			"message": "connected",
		})
	}
}

// ============================================================================
// HELPERS
// ============================================================================

func loadConfig() *Config {
	return &Config{
		Port:             getEnv("PORT", "8080"),
		RedisURL:         getEnv("REDIS_URL", "redis://localhost:6379"),
		WalletServiceURL: getEnv("WALLET_SERVICE_URL", "http://localhost:8081"),
		MarketDataURL:    getEnv("MARKET_DATA_URL", ""),
		PortfolioURL:     getEnv("PORTFOLIO_SERVICE_URL", ""),
		SwapServiceURL:   getEnv("SWAP_SERVICE_URL", ""),
		StakingServiceURL: getEnv("STAKING_SERVICE_URL", ""),
		NFTServiceURL:    getEnv("NFT_SERVICE_URL", ""),
		BridgeServiceURL: getEnv("BRIDGE_SERVICE_URL", ""),
	}
}

func proxyServiceGET(c *gin.Context, baseURL, path string) (map[string]interface{}, bool) {
	if baseURL == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "required upstream service is not configured; refusing to return mock data",
		})
		return nil, false
	}

	targetURL := baseURL + path
	if c.Request.URL.RawQuery != "" {
		targetURL += "?" + c.Request.URL.RawQuery
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, targetURL, nil)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "upstream request could not be created"})
		return nil, false
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "upstream service unavailable"})
		return nil, false
	}
	defer resp.Body.Close()

	var payload map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "upstream service returned invalid JSON"})
		return nil, false
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		c.JSON(resp.StatusCode, payload)
		return nil, false
	}
	return payload, true
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
	// Kept for any future non-cryptographic correlation-ID need. It is NEVER
	// used for transaction hashes, signatures, or wallet secrets — the gateway
	// only ever returns hashes it receives from a real upstream service.
	bytes := make([]byte, length/2)
	if _, err := cryptoRand.Read(bytes); err != nil {
		panic(fmt.Sprintf("secure random generation failed: %v", err))
	}
	return hex.EncodeToString(bytes)[:length]
}

var _ = generateRandomHex // suppress unused-warning; retained for tracing IDs

// proxyServicePOST forwards the inbound JSON body to an upstream canonical
// service via POST and streams its JSON response (or an honest 5xx when the
// upstream is unreachable). It never fabricates a response.
func proxyServicePOST(c *gin.Context, baseURL, path string) {
	if baseURL == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "required upstream service is not configured; refusing to return mock data",
		})
		return
	}

	targetURL := baseURL + path
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, targetURL, c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "upstream request could not be created"})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if auth := c.GetHeader("Authorization"); auth != "" {
		req.Header.Set("Authorization", auth)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "upstream service unavailable"})
		return
	}
	defer resp.Body.Close()

	c.Status(resp.StatusCode)
	c.Header("Content-Type", "application/json")
	io.Copy(c.Writer, resp.Body)
}
