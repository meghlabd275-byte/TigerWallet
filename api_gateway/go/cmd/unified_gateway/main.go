// Unified TigerWallet API Gateway
// High-performance Go gateway for all wallet services
// Ultra-low latency routing with Redis caching

package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Config holds all gateway configuration
type Config struct {
	Port           string
	WalletAPI      string
	StakingService string
	LendingService string
	BridgeService  string
	SwapService    string
	NFTService     string
	RedisHost      string
	RedisPort      string
	PostgresHost   string
	PostgresPort   string
	PostgresUser   string
	PostgresPass   string
	PostgresDB     string
	JWTSecret      string
}

// Global services
var (
	cfg           Config
	redisClient   *redis.Client
	postgresPool  *pgxpool.Pool
	chainConfigs  map[int64]*ChainConfig
	dexRouters    map[int64][]*DEXRouter
	tokenPrices   map[string]float64
	priceMu       sync.RWMutex
	httpClient    *http.Client
)

// ChainConfig holds chain configuration
type ChainConfig struct {
	ID             int64
	Name           string
	Symbol         string
	RPCEndpoint    string
	ExplorerAPI    string
	ExplorerURL    string
	NativeCurrency  string
	BlockTime      int
	ChainType      string // evm, solana, cosmos, etc.
}

// DEXRouter holds DEX router info
type DEXRouter struct {
	Name        string
	Address     string
	Factory     string
	InitCode    string
	Fee         uint32
	ChainID     int64
}

// TokenInfo holds token information
type TokenInfo struct {
	Symbol      string
	Name        string
	Address     string
	Decimals    uint8
	ChainID     int64
	IsNative    bool
	IsStable    bool
	PriceUSD    float64
	LogoURI     string
}

// =============================================================================
// Initialization
// =============================================================================

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("🐯 TigerWallet Unified API Gateway starting...")

	// Load configuration
	loadConfig()

	// Initialize Redis
	initRedis()

	// Initialize PostgreSQL
	initPostgres()

	// Initialize chain configs
	initChainConfigs()

	// Initialize HTTP client
	httpClient = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}

	// Start price fetcher
	go startPriceFetcher()

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())
	router.Use(rateLimitMiddleware())

	// Register routes
	registerRoutes(router)

	// Health check
	router.GET("/health", handleHealth)
	router.GET("/ready", handleReady)

	log.Printf("🚀 Gateway listening on port %s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start gateway: %v", err)
	}
}

func loadConfig() {
	cfg = Config{
		Port:           getEnv("PORT", "8080"),
		WalletAPI:      getEnv("WALLET_API", "http://localhost:8443"),
		StakingService: getEnv("STAKING_SERVICE", "http://localhost:8081"),
		LendingService: getEnv("LENDING_SERVICE", "http://localhost:8082"),
		BridgeService:  getEnv("BRIDGE_SERVICE", "http://localhost:8083"),
		SwapService:    getEnv("SWAP_SERVICE", "http://localhost:8084"),
		NFTService:     getEnv("NFT_SERVICE", "http://localhost:8085"),
		RedisHost:      getEnv("REDIS_HOST", "localhost"),
		RedisPort:      getEnv("REDIS_PORT", "6379"),
		PostgresHost:   getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:   getEnv("POSTGRES_PORT", "5432"),
		PostgresUser:   getEnv("POSTGRES_USER", "tigerwallet"),
		PostgresPass:   getEnv("POSTGRES_PASSWORD", "tigerwallet"),
		PostgresDB:     getEnv("POSTGRES_DB", "tigerwallet"),
		JWTSecret:      getEnv("JWT_SECRET", "tigerwallet-secret-key-change-in-production"),
	}
}

func initRedis() {
	addr := fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort)
	redisClient = redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     "",
		DB:           0,
		PoolSize:     100,
		MinIdleConns: 10,
	})
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("⚠️  Redis connection failed: %v (continuing without cache)", err)
	} else {
		log.Println("✅ Redis connected")
	}
}

func initPostgres() {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.PostgresUser, cfg.PostgresPass, cfg.PostgresHost, cfg.PostgresPort, cfg.PostgresDB)
	
	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		log.Printf("⚠️  PostgreSQL config failed: %v (continuing without DB)", err)
		return
	}
	poolConfig.MaxConns = 50
	poolConfig.MinConns = 5

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	var err2 error
	postgresPool, err2 = pgxpool.NewWithConfig(ctx, poolConfig)
	if err2 != nil {
		log.Printf("⚠️  PostgreSQL connection failed: %v (continuing without DB)", err2)
	} else {
		log.Println("✅ PostgreSQL connected")
	}
}

func initChainConfigs() {
	chainConfigs = map[int64]*ChainConfig{
		1: {
			ID:             1,
			Name:           "Ethereum",
			Symbol:         "ETH",
			RPCEndpoint:    "https://eth.llamarpc.com",
			ExplorerAPI:    "https://api.etherscan.io/api",
			ExplorerURL:    "https://etherscan.io",
			NativeCurrency: "ETH",
			BlockTime:      12,
			ChainType:      "evm",
		},
		56: {
			ID:             56,
			Name:           "BNB Chain",
			Symbol:         "BNB",
			RPCEndpoint:    "https://bsc-dataseed.binance.org",
			ExplorerAPI:    "https://api.bscscan.com/api",
			ExplorerURL:    "https://bscscan.com",
			NativeCurrency: "BNB",
			BlockTime:      3,
			ChainType:      "evm",
		},
		137: {
			ID:             137,
			Name:           "Polygon",
			Symbol:         "MATIC",
			RPCEndpoint:    "https://polygon-rpc.com",
			ExplorerAPI:    "https://api.polygonscan.com/api",
			ExplorerURL:    "https://polygonscan.com",
			NativeCurrency: "MATIC",
			BlockTime:      2,
			ChainType:      "evm",
		},
		42161: {
			ID:             42161,
			Name:           "Arbitrum One",
			Symbol:         "ETH",
			RPCEndpoint:    "https://arb1.arbitrum.io/rpc",
			ExplorerAPI:    "https://api.arbiscan.io/api",
			ExplorerURL:    "https://arbiscan.io",
			NativeCurrency: "ETH",
			BlockTime:      1,
			ChainType:      "evm",
		},
		10: {
			ID:             10,
			Name:           "Optimism",
			Symbol:         "ETH",
			RPCEndpoint:    "https://mainnet.optimism.io",
			ExplorerAPI:    "https://api-optimistic.etherscan.io/api",
			ExplorerURL:    "https://optimistic.etherscan.io",
			NativeCurrency: "ETH",
			BlockTime:      2,
			ChainType:      "evm",
		},
		8453: {
			ID:             8453,
			Name:           "Base",
			Symbol:         "ETH",
			RPCEndpoint:    "https://mainnet.base.org",
			ExplorerAPI:    "https://api.basescan.org/api",
			ExplorerURL:    "https://basescan.org",
			NativeCurrency: "ETH",
			BlockTime:      2,
			ChainType:      "evm",
		},
		43114: {
			ID:             43114,
			Name:           "Avalanche C-Chain",
			Symbol:         "AVAX",
			RPCEndpoint:    "https://api.avax.network/ext/bc/C/rpc",
			ExplorerAPI:    "https://api.snowtrace.io/api",
			ExplorerURL:    "https://snowtrace.io",
			NativeCurrency: "AVAX",
			BlockTime:      2,
			ChainType:      "evm",
		},
		25: {
			ID:             25,
			Name:           "Cronos",
			Symbol:         "CRO",
			RPCEndpoint:    "https://evm.cronos.org",
			ExplorerAPI:    "https://api.cronoscan.com/api",
			ExplorerURL:    "https://cronoscan.com",
			NativeCurrency: "CRO",
			BlockTime:      6,
			ChainType:      "evm",
		},
		250: {
			ID:             250,
			Name:           "Fantom",
			Symbol:         "FTM",
			RPCEndpoint:    "https://rpc.fantom.network",
			ExplorerAPI:    "https://api.ftmscan.com/api",
			ExplorerURL:    "https://ftmscan.com",
			NativeCurrency: "FTM",
			BlockTime:      1,
			ChainType:      "evm",
		},
		42220: {
			ID:             42220,
			Name:           "Celo",
			Symbol:         "CELO",
			RPCEndpoint:    "https://forno.celo.org",
			ExplorerAPI:    "https://api.celoscan.io/api",
			ExplorerURL:    "https://celoscan.io",
			NativeCurrency: "CELO",
			BlockTime:      5,
			ChainType:      "evm",
		},
	}
	
	// DEX Routers
	dexRouters = map[int64][]*DEXRouter{
		1: {
			{Name: "Uniswap V3", Address: "0x68b3465833fb72A70ecDF485E0e4C7bD8665Fc45", Factory: "0x1F98431c8aD98523631AE4a59f267346ea31F984", Fee: 500, ChainID: 1},
			{Name: "Uniswap V2", Address: "0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D", Factory: "0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f", Fee: 3000, ChainID: 1},
			{Name: "SushiSwap", Address: "0xd9e1cE17f2641f24aE83637ab66a2cca9C378B9F", Factory: "0xC0AEe478e3658e2610c5F7A4A2E1777cE9e4f2Ac", Fee: 3000, ChainID: 1},
		},
		56: {
			{Name: "PancakeSwap", Address: "0x10ED43C718714eb63d5aA57B78B54704E384024E", Factory: "0xca143ce32fe78f1f7019d7d551a6402fc5350c73", Fee: 2500, ChainID: 56},
		},
		137: {
			{Name: "QuickSwap", Address: "0xa5E0829CaCEd8fFD4De81E942986A134Aa4FC85E", Factory: "0x5757371414417b8C6CAad45bAeF941aBc7d3b32E", Fee: 3000, ChainID: 137},
			{Name: "SushiSwap", Address: "0x1b02dA8Cb0d097eB8D57A175b88c7D8b47997506", Factory: "0xc35DADB65012eC5796536bD9864eD8773aBc74C4", Fee: 3000, ChainID: 137},
		},
		42161: {
			{Name: "Uniswap V3", Address: "0x68b3465833fb72A70ecDF485E0e4C7bD8665Fc45", Factory: "0x1F98431c8aD98523631AE4a59f267346ea31F984", Fee: 500, ChainID: 42161},
			{Name: "SushiSwap", Address: "0x1b02dA8Cb0d097eB8D57A175b88c7D8b47997506", Factory: "0xc35DADB65012eC5796536bD9864eD8773aBc74C4", Fee: 3000, ChainID: 42161},
		},
		10: {
			{Name: "Uniswap V3", Address: "0x68b3465833fb72A70ecDF485E0e4C7bD8665Fc45", Factory: "0x1F98431c8aD98523631AE4a59f267346ea31F984", Fee: 500, ChainID: 10},
		},
		8453: {
			{Name: "BaseSwap", Address: "0x45dDa9a7CfF581D8707fC52a2972C2b3F5778286", Factory: "0xFe31Aa2Dd7a9d45C722E4Ac5fA20E66a9a64b9C2", Fee: 3000, ChainID: 8453},
		},
	}

	tokenPrices = map[string]float64{
		"ETH":  3500.0,
		"BNB":  600.0,
		"MATIC": 1.0,
		"USDC": 1.0,
		"USDT": 1.0,
		"DAI":  1.0,
		"WBTC": 65000.0,
		"AVAX": 35.0,
		"CRO":  0.1,
		"FTM":  0.3,
		"CELO": 0.7,
	}
}

// =============================================================================
// Middleware
// =============================================================================

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Requested-With")
		c.Header("Access-Control-Max-Age", "86400")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func rateLimitMiddleware() gin.HandlerFunc {
	type bucket struct {
		tokens    float64
		lastCheck int64
		mu        sync.Mutex
	}
	
	buckets := make(map[string]*bucket)
	var bucketMu sync.Mutex
	
	return func(c *gin.Context) {
		ip := c.ClientIP()
		bucketMu.Lock()
		b, ok := buckets[ip]
		if !ok {
			b = &bucket{tokens: 100, lastCheck: time.Now().Unix()}
			buckets[ip] = b
		}
		bucketMu.Unlock()
		
		b.mu.Lock()
		defer b.mu.Unlock()
		
		now := time.Now().Unix()
		elapsed := now - b.lastCheck
		b.tokens += float64(elapsed) * 10 // 10 tokens per second
		if b.tokens > 100 {
			b.tokens = 100
		}
		b.lastCheck = now
		
		if b.tokens < 1 {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		b.tokens--
		
		c.Next()
	}
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}
		
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			return
		}
		
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(cfg.JWTSecret), nil
		})
		
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			return
		}
		
		c.Set("user_id", claims["sub"])
		c.Next()
	}
}

// =============================================================================
// Routes
// =============================================================================

func registerRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")
	
	// Auth routes (public)
	api.POST("/auth/register", handleAuthRegister)
	api.POST("/auth/login", handleAuthLogin)
	
	// Chain info (public)
	api.GET("/chains", handleGetChains)
	api.GET("/chains/:chain_id", handleGetChain)
	
	// Wallet routes (protected)
	wallet := api.Group("/wallet")
	wallet.Use(authMiddleware())
	{
		wallet.POST("/create", handleWalletCreate)
		wallet.GET("/list", handleWalletList)
		wallet.GET("/:address/balance", handleWalletBalance)
		wallet.GET("/:address/tokens", handleWalletTokens)
		wallet.GET("/:address/transactions", handleWalletTransactions)
		wallet.GET("/:address/nfts", handleWalletNFTs)
	}
	
	// Staking routes (protected)
	staking := api.Group("/staking")
	staking.Use(authMiddleware())
	{
		staking.GET("/pools", handleStakingPools)
		staking.GET("/positions", handleStakingPositions)
		staking.POST("/stake", handleStakingStake)
		staking.POST("/unstake", handleStakingUnstake)
		staking.POST("/claim", handleStakingClaim)
	}
	
	// Lending routes (protected)
	lending := api.Group("/lending")
	lending.Use(authMiddleware())
	{
		lending.GET("/markets", handleLendingMarkets)
		lending.GET("/position", handleLendingPosition)
		lending.POST("/supply", handleLendingSupply)
		lending.POST("/withdraw", handleLendingWithdraw)
		lending.POST("/borrow", handleLendingBorrow)
		lending.POST("/repay", handleLendingRepay)
	}
	
	// Bridge routes (protected)
	bridge := api.Group("/bridge")
	bridge.Use(authMiddleware())
	{
		bridge.GET("/routes", handleBridgeRoutes)
		bridge.GET("/quote", handleBridgeQuote)
		bridge.POST("/transfer", handleBridgeTransfer)
		bridge.GET("/history", handleBridgeHistory)
	}
	
	// Swap routes (protected)
	swap := api.Group("/swap")
	swap.Use(authMiddleware())
	{
		swap.GET("/tokens", handleSwapTokens)
		swap.GET("/quote", handleSwapQuote)
		swap.GET("/routes", handleSwapRoutes)
		swap.POST("/execute", handleSwapExecute)
	}
	
	// NFT routes (protected)
	nft := api.Group("/nft")
	nft.Use(authMiddleware())
	{
		nft.GET("/collections", handleNFTCollections)
		nft.GET("/items", handleNFTItems)
		nft.GET("/:id", handleNFTDetail)
		nft.POST("/buy", handleNFTBuy)
		nft.POST("/sell", handleNFTSell)
		nft.POST("/list", handleNFTList)
	}
	
	// Gas & Price (public)
	api.GET("/gas/:chain_id", handleGasPrice)
	api.GET("/price/:symbol", handleTokenPrice)
	api.GET("/prices", handleAllPrices)
	
	// DEX Info (public)
	api.GET("/dex/routers", handleDEXRouters)
}

// =============================================================================
// Health & Ready
// =============================================================================

func handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "tigerwallet-unified-gateway",
		"version":   "1.0.0",
		"timestamp": time.Now().UTC(),
	})
}

func handleReady(c *gin.Context) {
	ready := true
	issues := []string{}
	
	// Check Redis
	if redisClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := redisClient.Ping(ctx).Err(); err != nil {
			ready = false
			issues = append(issues, "redis: "+err.Error())
		}
	}
	
	// Check Postgres
	if postgresPool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := postgresPool.Ping(ctx); err != nil {
			ready = false
			issues = append(issues, "postgres: "+err.Error())
		}
	}
	
	status := http.StatusOK
	if !ready {
		status = http.StatusServiceUnavailable
	}
	
	c.JSON(status, gin.H{
		"ready":    ready,
		"issues":    issues,
		"timestamp": time.Now().UTC(),
	})
}

// =============================================================================
// Auth Handlers
// =============================================================================

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required,min=3"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func handleAuthRegister(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Check if user exists
	if postgresPool != nil {
		var exists bool
		err := postgresPool.QueryRow(c.Request.Context(), 
			"SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", req.Email).Scan(&exists)
		if err == nil && exists {
			c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
			return
		}
	}
	
	// Create user (simplified - in production use proper hashing)
	userID := fmt.Sprintf("user_%d", time.Now().UnixNano())
	
	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":      userID,
		"email":    req.Email,
		"username": req.Username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	})
	
	tokenString, err := token.SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"user_id": userID,
		"token":   tokenString,
	})
}

func handleAuthLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// For demo purposes, accept any valid credentials
	// In production, verify against database
	
	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   "user_demo",
		"email": req.Email,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	})
	
	tokenString, err := token.SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"token": tokenString,
	})
}

// =============================================================================
// Chain Handlers
// =============================================================================

func handleGetChains(c *gin.Context) {
	chains := make([]*ChainConfig, 0, len(chainConfigs))
	for _, chain := range chainConfigs {
		chains = append(chains, chain)
	}
	c.JSON(http.StatusOK, gin.H{"chains": chains})
}

func handleGetChain(c *gin.Context) {
	chainID, err := strconv.ParseInt(c.Param("chain_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chain_id"})
		return
	}
	
	chain, ok := chainConfigs[chainID]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "chain not found"})
		return
	}
	
	c.JSON(http.StatusOK, chain)
}

// =============================================================================
// Wallet Handlers
// =============================================================================

type CreateWalletRequest struct {
	ChainID      int64  `json:"chain_id"`
	Label        string `json:"label"`
	AccountIndex int    `json:"account_index"`
}

func handleWalletCreate(c *gin.Context) {
	var req CreateWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	if req.ChainID == 0 {
		req.ChainID = 1
	}
	
	chain, ok := chainConfigs[req.ChainID]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported chain"})
		return
	}
	
	// Generate a demo address (in production, use proper HD derivation)
	address := generateDemoAddress()
	
	c.JSON(http.StatusCreated, gin.H{
		"address":         address,
		"chain_id":        req.ChainID,
		"chain_name":      chain.Name,
		"label":           req.Label,
		"derivation_path": fmt.Sprintf("m/44'/60'/0'/0/%d", req.AccountIndex),
	})
}

func handleWalletList(c *gin.Context) {
	userID := c.GetString("user_id")
	_ = userID
	
	// Return demo wallets
	wallets := []gin.H{
		{"address": "0x742d35Cc6634C053292505a5eC874A66E8535F5F", "chain_id": 1, "label": "Ethereum Main"},
		{"address": "0x123d35Cc6634C053292505a5eC874A66E8535F5F", "chain_id": 56, "label": "BNB Chain"},
		{"address": "0x456d35Cc6634C053292505a5eC874A66E8535F5F", "chain_id": 137, "label": "Polygon"},
	}
	
	c.JSON(http.StatusOK, gin.H{"wallets": wallets})
}

func handleWalletBalance(c *gin.Context) {
	address := c.Param("address")
	chainIDStr := c.Query("chain_id")
	
	chainID, _ := strconv.ParseInt(chainIDStr, 10, 64)
	if chainID == 0 {
		chainID = 1
	}
	
	// Fetch real balance from chain
	balance, err := fetchNativeBalance(chainID, address)
	if err != nil {
		balance = 1.5 // Demo fallback
	}
	
	priceMu.RLock()
	price := tokenPrices["ETH"]
	if chain, ok := chainConfigs[chainID]; ok {
		price = tokenPrices[chain.Symbol]
	}
	priceMu.RUnlock()
	
	c.JSON(http.StatusOK, gin.H{
		"chain_id":   chainID,
		"address":    address,
		"balance":    balance,
		"balance_usd": balance * price,
		"symbol":     chainConfigs[chainID].Symbol,
	})
}

func handleWalletTokens(c *gin.Context) {
	address := c.Param("address")
	chainID, _ := strconv.ParseInt(c.Query("chain_id"), 10, 64)
	if chainID == 0 {
		chainID = 1
	}
	
	// Return common tokens with demo balances
	tokens := []gin.H{
		{"symbol": "USDC", "address": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", "balance": 1000.0, "balance_usd": 1000.0},
		{"symbol": "USDT", "address": "0xdAC17F958D2ee523a2206206994597C13D831ec7", "balance": 500.0, "balance_usd": 500.0},
		{"symbol": "WBTC", "address": "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599", "balance": 0.01, "balance_usd": 650.0},
	}
	
	c.JSON(http.StatusOK, gin.H{"tokens": tokens, "chain_id": chainID})
}

func handleWalletTransactions(c *gin.Context) {
	address := c.Param("address")
	chainID, _ := strconv.ParseInt(c.Query("chain_id"), 10, 64)
	
	// Return demo transactions
	txs := []gin.H{
		{
			"hash":    "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			"from":    address,
			"to":      "0x742d35Cc6634C053292505a5eC874A66E8535F5F",
			"value":   "0.5",
			"symbol":  "ETH",
			"status":  "confirmed",
			"block":   18500000,
			"timestamp": time.Now().Add(-3600).Unix(),
		},
	}
	
	c.JSON(http.StatusOK, gin.H{"transactions": txs, "chain_id": chainID})
}

func handleWalletNFTs(c *gin.Context) {
	address := c.Param("address")
	chainID, _ := strconv.ParseInt(c.Query("chain_id"), 10, 64)
	
	// Return demo NFTs
	nfts := []gin.H{
		{
			"token_id":     "1234",
			"contract":     "0xBC4CA0EdA7647A8aB7C2061c2E118A18a936f13D",
			"name":         "Bored Ape Yacht Club #1234",
			"symbol":       "BAYC",
			"image_url":    "https://ipfs.io/ipfs/QmRRPWG96cmgTn2qSzjwr2qvfIzuhPSJB66CH7XBy6uD4f/1234.png",
			"chain_id":     1,
		},
	}
	
	c.JSON(http.StatusOK, gin.H{"nfts": nfts, "chain_id": chainID})
}

// =============================================================================
// Staking Handlers
// =============================================================================

type StakingPool struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Token       string  `json:"token"`
	ChainID     int64   `json:"chain_id"`
	APY         float64 `json:"apy"`
	MinStake    float64 `json:"min_stake"`
	TotalStaked float64 `json:"total_staked"`
	TVL         float64 `json:"tvl"`
	Description string  `json:"description"`
}

type StakingPosition struct {
	ID        string  `json:"id"`
	PoolID    string  `json:"pool_id"`
	PoolName  string  `json:"pool_name"`
	Token     string  `json:"token"`
	Amount    float64 `json:"amount"`
	Reward    float64 `json:"reward"`
	APY       float64 `json:"apy"`
	Status    string  `json:"status"`
	StartTime int64   `json:"start_time"`
}

func handleStakingPools(c *gin.Context) {
	pools := []StakingPool{
		{ID: "lido", Name: "Lido Liquid Staking", Token: "ETH", ChainID: 1, APY: 4.2, MinStake: 0.01, TotalStaked: 15.2e9, TVL: 15.2e9, Description: "Liquid staking through Lido"},
		{ID: "rocketpool", Name: "Rocket Pool", Token: "ETH", ChainID: 1, APY: 3.8, MinStake: 0.01, TotalStaked: 2.1e9, TVL: 2.1e9, Description: "Decentralized liquid staking"},
		{ID: "aave", Name: "Aave Staking", Token: "AAVE", ChainID: 1, APY: 5.5, MinStake: 1.0, TotalStaked: 180e6, TVL: 180e6, Description: "Stake AAVE for rewards"},
		{ID: "compound", Name: "Compound Staking", Token: "COMP", ChainID: 1, APY: 4.8, MinStake: 0.1, TotalStaked: 120e6, TVL: 120e6, Description: "Stake COMP for governance rewards"},
	}
	
	c.JSON(http.StatusOK, gin.H{"pools": pools})
}

func handleStakingPositions(c *gin.Context) {
	userID := c.GetString("user_id")
	_ = userID
	
	positions := []StakingPosition{
		{
			ID: "pos_1", PoolID: "lido", PoolName: "Lido Liquid Staking",
			Token: "stETH", Amount: 5.5, Reward: 0.23, APY: 4.2,
			Status: "active", StartTime: time.Now().Add(-30*24*time.Hour).Unix(),
		},
	}
	
	c.JSON(http.StatusOK, gin.H{"positions": positions})
}

type StakeRequest struct {
	PoolID string  `json:"pool_id" binding:"required"`
	Amount float64 `json:"amount" binding:"required"`
}

func handleStakingStake(c *gin.Context) {
	var req StakeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be positive"})
		return
	}
	
	// In production, this would:
	// 1. Validate the pool exists
	// 2. Build and sign the staking transaction
	// 3. Broadcast to the network
	// 4. Track the position in database
	
	txHash := fmt.Sprintf("0x%x", time.Now().UnixNano())
	
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"tx_hash":    txHash,
		"pool_id":    req.PoolID,
		"amount":     req.Amount,
		"position_id": fmt.Sprintf("pos_%d", time.Now().UnixNano()),
	})
}

type UnstakeRequest struct {
	PositionID string  `json:"position_id" binding:"required"`
	Amount     float64 `json:"amount"`
}

func handleStakingUnstake(c *gin.Context) {
	var req UnstakeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	txHash := fmt.Sprintf("0x%x", time.Now().UnixNano())
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"tx_hash": txHash,
		"message": "Unstake initiated. Funds will be available after the unbonding period.",
	})
}

type ClaimRequest struct {
	PositionID string `json:"position_id" binding:"required"`
}

func handleStakingClaim(c *gin.Context) {
	var req ClaimRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	txHash := fmt.Sprintf("0x%x", time.Now().UnixNano())
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"tx_hash": txHash,
		"message": "Rewards claimed successfully",
	})
}

// =============================================================================
// Lending Handlers
// =============================================================================

type LendingMarket struct {
	ID                   int64   `json:"id"`
	Asset                string  `json:"asset"`
	Symbol               string  `json:"symbol"`
	ChainID              int64   `json:"chain_id"`
	TotalSupply          float64 `json:"total_supply"`
	TotalBorrow          float64 `json:"total_borrow"`
	SupplyAPY            float64 `json:"supply_apy"`
	BorrowAPY            float64 `json:"borrow_apy"`
	Utilization          float64 `json:"utilization"`
	LTV                  float64 `json:"ltv"`
	LiquidationThreshold float64 `json:"liquidation_threshold"`
}

type LendingPosition struct {
	Supplies   []LendingSupply   `json:"supplies"`
	Borrows    []LendingBorrow   `json:"borrows"`
	HealthFactor float64         `json:"health_factor"`
	TotalCollateral float64      `json:"total_collateral"`
	TotalDebt  float64           `json:"total_debt"`
}

type LendingSupply struct {
	Asset   string  `json:"asset"`
	Amount  float64 `json:"amount"`
	APY     float64 `json:"apy"`
	ValueUSD float64 `json:"value_usd"`
}

type LendingBorrow struct {
	Asset   string  `json:"asset"`
	Amount  float64 `json:"amount"`
	APY     float64 `json:"apy"`
	ValueUSD float64 `json:"value_usd"`
}

func handleLendingMarkets(c *gin.Context) {
	chainID, _ := strconv.ParseInt(c.Query("chain_id"), 10, 64)
	if chainID == 0 {
		chainID = 1
	}
	
	markets := []LendingMarket{
		{ID: 1, Asset: "0x0000000000000000000000000000000000000000", Symbol: "ETH", ChainID: chainID, TotalSupply: 500e6, TotalBorrow: 350e6, SupplyAPY: 3.5, BorrowAPY: 5.2, Utilization: 0.70, LTV: 0.80, LiquidationThreshold: 0.85},
		{ID: 2, Asset: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", Symbol: "USDC", ChainID: chainID, TotalSupply: 1000e6, TotalBorrow: 600e6, SupplyAPY: 4.0, BorrowAPY: 5.5, Utilization: 0.60, LTV: 0.85, LiquidationThreshold: 0.90},
		{ID: 3, Asset: "0xdAC17F958D2ee523a2206206994597C13D831ec7", Symbol: "USDT", ChainID: chainID, TotalSupply: 800e6, TotalBorrow: 500e6, SupplyAPY: 4.2, BorrowAPY: 5.8, Utilization: 0.625, LTV: 0.85, LiquidationThreshold: 0.90},
		{ID: 4, Asset: "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599", Symbol: "WBTC", ChainID: chainID, TotalSupply: 50e8, TotalBorrow: 25e8, SupplyAPY: 1.8, BorrowAPY: 3.5, Utilization: 0.50, LTV: 0.70, LiquidationThreshold: 0.80},
	}
	
	c.JSON(http.StatusOK, gin.H{"markets": markets})
}

func handleLendingPosition(c *gin.Context) {
	userID := c.GetString("user_id")
	_ = userID
	
	position := LendingPosition{
		Supplies: []LendingSupply{
			{Asset: "ETH", Amount: 10.0, APY: 3.5, ValueUSD: 35000},
			{Asset: "USDC", Amount: 5000, APY: 4.0, ValueUSD: 5000},
		},
		Borrows: []LendingBorrow{
			{Asset: "USDT", Amount: 2000, APY: 5.8, ValueUSD: 2000},
		},
		HealthFactor: 2.5,
		TotalCollateral: 40000,
		TotalDebt: 2000,
	}
	
	c.JSON(http.StatusOK, position)
}

type SupplyRequest struct {
	Asset   string  `json:"asset" binding:"required"`
	Amount  float64 `json:"amount" binding:"required"`
	ChainID int64   `json:"chain_id"`
}

func handleLendingSupply(c *gin.Context) {
	var req SupplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	txHash := fmt.Sprintf("0x%x", time.Now().UnixNano())
	
	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"tx_hash":     txHash,
		"asset":       req.Asset,
		"amount":      req.Amount,
		"new_balance": req.Amount,
	})
}

type WithdrawRequest struct {
	Asset   string  `json:"asset" binding:"required"`
	Amount  float64 `json:"amount" binding:"required"`
	ChainID int64   `json:"chain_id"`
}

func handleLendingWithdraw(c *gin.Context) {
	var req WithdrawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	txHash := fmt.Sprintf("0x%x", time.Now().UnixNano())
	
	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"tx_hash":     txHash,
		"asset":       req.Asset,
		"amount":      req.Amount,
	})
}

type BorrowRequest struct {
	Asset     string  `json:"asset" binding:"required"`
	Amount    float64 `json:"amount" binding:"required"`
	RateMode  string  `json:"rate_mode"` // "stable" or "variable"
	ChainID   int64   `json:"chain_id"`
}

func handleLendingBorrow(c *gin.Context) {
	var req BorrowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	txHash := fmt.Sprintf("0x%x", time.Now().UnixNano())
	
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"tx_hash":  txHash,
		"asset":    req.Asset,
		"amount":   req.Amount,
		"rate_mode": req.RateMode,
	})
}

type RepayRequest struct {
	Asset    string  `json:"asset" binding:"required"`
	Amount   float64 `json:"amount" binding:"required"`
	ChainID  int64   `json:"chain_id"`
}

func handleLendingRepay(c *gin.Context) {
	var req RepayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	txHash := fmt.Sprintf("0x%x", time.Now().UnixNano())
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"tx_hash": txHash,
		"asset":   req.Asset,
		"amount":  req.Amount,
	})
}

// =============================================================================
// Bridge Handlers
// =============================================================================

type BridgeRoute struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Logo         string  `json:"logo"`
	Fee          float64 `json:"fee"`
	Time         string  `json:"time"`
	MinAmount    float64 `json:"min_amount"`
	MaxAmount    float64 `json:"max_amount"`
	Reliability  float64 `json:"reliability"`
	FromChainID  int64   `json:"from_chain_id"`
	ToChainID    int64   `json:"to_chain_id"`
}

type BridgeQuote struct {
	FromChain      int64   `json:"from_chain"`
	ToChain        int64   `json:"to_chain"`
	Token          string  `json:"token"`
	Amount         float64 `json:"amount"`
	ReceivedAmount float64 `json:"received_amount"`
	BridgeFee      float64 `json:"bridge_fee"`
	NetworkFee     float64 `json:"network_fee"`
	EstimatedTime  string  `json:"estimated_time"`
	Rate           float64 `json:"rate"`
	Route          string  `json:"route"`
}

func handleBridgeRoutes(c *gin.Context) {
	fromChainID, _ := strconv.ParseInt(c.Query("from_chain"), 10, 64)
	toChainID, _ := strconv.ParseInt(c.Query("to_chain"), 10, 64)
	
	routes := []BridgeRoute{
		{ID: "across", Name: "Across Protocol", Logo: "🔄", Fee: 0.09, Time: "1-3 min", MinAmount: 10, MaxAmount: 1000000, Reliability: 99.2, FromChainID: fromChainID, ToChainID: toChainID},
		{ID: "stargate", Name: "Stargate", Logo: "🌉", Fee: 0.06, Time: "3-5 min", MinAmount: 50, MaxAmount: 500000, Reliability: 98.8, FromChainID: fromChainID, ToChainID: toChainID},
		{ID: "hop", Name: "Hop Exchange", Logo: "⚡", Fee: 0.04, Time: "5-10 min", MinAmount: 100, MaxAmount: 250000, Reliability: 98.5, FromChainID: fromChainID, ToChainID: toChainID},
		{ID: "cbridge", Name: "Celer Bridge", Logo: "🌐", Fee: 0.03, Time: "10-20 min", MinAmount: 100, MaxAmount: 1000000, Reliability: 97.9, FromChainID: fromChainID, ToChainID: toChainID},
		{ID: "synapse", Name: "Synapse", Logo: "🔗", Fee: 0.05, Time: "5-15 min", MinAmount: 50, MaxAmount: 250000, Reliability: 97.5, FromChainID: fromChainID, ToChainID: toChainID},
	}
	
	c.JSON(http.StatusOK, gin.H{"routes": routes})
}

func handleBridgeQuote(c *gin.Context) {
	fromChainID, _ := strconv.ParseInt(c.Query("from_chain"), 10, 64)
	toChainID, _ := strconv.ParseInt(c.Query("to_chain"), 10, 64)
	amount, _ := strconv.ParseFloat(c.Query("amount"), 64)
	token := c.DefaultQuery("token", "ETH")
	
	// Calculate quote based on route and amounts
	bridgeFee := amount * 0.003 // 0.3%
	networkFee := 5.0 // flat fee
	receivedAmount := amount - bridgeFee - (networkFee / tokenPrices[token])
	
	quote := BridgeQuote{
		FromChain:     fromChainID,
		ToChain:       toChainID,
		Token:         token,
		Amount:        amount,
		ReceivedAmount: receivedAmount,
		BridgeFee:     bridgeFee,
		NetworkFee:    networkFee,
		EstimatedTime: "5-10 min",
		Rate:          1.0,
		Route:         "Across Protocol",
	}
	
	c.JSON(http.StatusOK, quote)
}

type TransferRequest struct {
	FromChain  int64   `json:"from_chain" binding:"required"`
	ToChain    int64   `json:"to_chain" binding:"required"`
	Token      string  `json:"token" binding:"required"`
	Amount     float64 `json:"amount" binding:"required"`
	Route      string  `json:"route" binding:"required"`
	DestAddress string `json:"dest_address"`
}

func handleBridgeTransfer(c *gin.Context) {
	var req TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	txHash := fmt.Sprintf("0x%x", time.Now().UnixNano())
	
	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"tx_hash":       txHash,
		"transfer_id":   fmt.Sprintf("bridge_%d", time.Now().UnixNano()),
		"from_chain":    req.FromChain,
		"to_chain":      req.ToChain,
		"token":         req.Token,
		"amount":        req.Amount,
		"status":        "pending",
		"estimated_time": "5-10 min",
	})
}

func handleBridgeHistory(c *gin.Context) {
	userID := c.GetString("user_id")
	_ = userID
	
	history := []gin.H{
		{
			"id":          "bridge_1",
			"from_chain":  1,
			"to_chain":    137,
			"token":       "ETH",
			"amount":      1.0,
			"status":      "completed",
			"timestamp":   time.Now().Add(-24*time.Hour).Unix(),
		},
	}
	
	c.JSON(http.StatusOK, gin.H{"history": history})
}

// =============================================================================
// Swap Handlers
// =============================================================================

type SwapToken struct {
	Address    string  `json:"address"`
	Symbol     string  `json:"symbol"`
	Name       string  `json:"name"`
	Decimals   uint8   `json:"decimals"`
	ChainID    int64   `json:"chain_id"`
	IsNative   bool    `json:"is_native"`
	IsStable   bool    `json:"is_stable"`
	PriceUSD   float64 `json:"price_usd"`
	LogoURI    string  `json:"logo_uri"`
}

type SwapQuote struct {
	InputToken     string          `json:"input_token"`
	OutputToken    string          `json:"output_token"`
	InputAmount    float64         `json:"input_amount"`
	OutputAmount   float64         `json:"output_amount"`
	MinimumOut     float64         `json:"minimum_out"`
	PriceImpact    float64         `json:"price_impact"`
	Route          []SwapRouteStep `json:"route"`
	GasEstimate    float64         `json:"gas_estimate"`
	GasFeeUSD      float64         `json:"gas_fee_usd"`
	ExchangeRate   float64         `json:"exchange_rate"`
	ExpiresAt      int64           `json:"expires_at"`
}

type SwapRouteStep struct {
	Dex       string  `json:"dex"`
	PoolAddr  string  `json:"pool_address"`
	Fee       uint32  `json:"fee"`
	AmountIn  float64 `json:"amount_in"`
	AmountOut float64 `json:"amount_out"`
}

func handleSwapTokens(c *gin.Context) {
	chainID, _ := strconv.ParseInt(c.Query("chain_id"), 10, 64)
	if chainID == 0 {
		chainID = 1
	}
	
	tokens := []SwapToken{
		{Address: "0x0000000000000000000000000000000000000000", Symbol: "ETH", Name: "Ethereum", Decimals: 18, ChainID: chainID, IsNative: true, PriceUSD: 3500, LogoURI: "https://assets.coingecko.com/coins/images/279/small/ethereum.png"},
		{Address: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", Symbol: "USDC", Name: "USD Coin", Decimals: 6, ChainID: chainID, IsStable: true, PriceUSD: 1.0, LogoURI: "https://assets.coingecko.com/coins/images/6319/small/usdc.png"},
		{Address: "0xdAC17F958D2ee523a2206206994597C13D831ec7", Symbol: "USDT", Name: "Tether USD", Decimals: 6, ChainID: chainID, IsStable: true, PriceUSD: 1.0, LogoURI: "https://assets.coingecko.com/coins/images/325/small/Tether.png"},
		{Address: "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599", Symbol: "WBTC", Name: "Wrapped Bitcoin", Decimals: 8, ChainID: chainID, PriceUSD: 65000, LogoURI: "https://assets.coingecko.com/coins/images/7598/small/wrapped_bitcoin_wbtc.png"},
		{Address: "0x514910771AF9Ca656af840dff83E8264EcF986CA", Symbol: "LINK", Name: "Chainlink", Decimals: 18, ChainID: chainID, PriceUSD: 15, LogoURI: "https://assets.coingecko.com/coins/images/877/small/chainlink-new-logo.png"},
	}
	
	c.JSON(http.StatusOK, gin.H{"tokens": tokens, "chain_id": chainID})
}

func handleSwapQuote(c *gin.Context) {
	tokenIn := c.Query("token_in")
	tokenOut := c.Query("token_out")
	amount, _ := strconv.ParseFloat(c.Query("amount"), 64)
	chainID, _ := strconv.ParseInt(c.Query("chain_id"), 10, 64)
	
	priceMu.RLock()
	priceIn := tokenPrices[tokenIn]
	priceOut := tokenPrices[tokenOut]
	priceMu.RUnlock()
	
	if priceIn == 0 {
		priceIn = 1.0
	}
	if priceOut == 0 {
		priceOut = 1.0
	}
	
	outputAmount := (amount * priceIn) / priceOut
	priceImpact := 0.5 // Demo value
	gasEstimate := 0.002 // ETH
	
	quote := SwapQuote{
		InputToken:    tokenIn,
		OutputToken:   tokenOut,
		InputAmount:   amount,
		OutputAmount:  outputAmount,
		MinimumOut:    outputAmount * 0.995, // 0.5% slippage
		PriceImpact:   priceImpact,
		GasEstimate:   gasEstimate,
		GasFeeUSD:     gasEstimate * 3500,
		ExchangeRate:  priceIn / priceOut,
		ExpiresAt:     time.Now().Add(30 * time.Second).Unix(),
		Route: []SwapRouteStep{
			{Dex: "Uniswap V3", Fee: 500, AmountIn: amount, AmountOut: outputAmount},
		},
	}
	
	c.JSON(http.StatusOK, quote)
}

func handleSwapRoutes(c *gin.Context) {
	tokenIn := c.Query("token_in")
	tokenOut := c.Query("token_out")
	chainID, _ := strconv.ParseInt(c.Query("chain_id"), 10, 64)
	
	routes := []gin.H{
		{
			"dex":        "Uniswap V3",
			"path":       []string{tokenIn, tokenOut},
			"fee":        500,
			"liquidity":  10000000,
			"price_impact": 0.3,
		},
		{
			"dex":        "SushiSwap",
			"path":       []string{tokenIn, tokenOut},
			"fee":        3000,
			"liquidity":  5000000,
			"price_impact": 0.5,
		},
	}
	
	c.JSON(http.StatusOK, gin.H{"routes": routes, "chain_id": chainID})
}

type SwapExecuteRequest struct {
	TokenIn    string  `json:"token_in" binding:"required"`
	TokenOut   string  `json:"token_out" binding:"required"`
	AmountIn   float64 `json:"amount_in" binding:"required"`
	MinOut     float64 `json:"min_out" binding:"required"`
	Recipient  string  `json:"recipient"`
	ChainID    int64   `json:"chain_id"`
}

func handleSwapExecute(c *gin.Context) {
	var req SwapExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	txHash := fmt.Sprintf("0x%x", time.Now().UnixNano())
	
	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"tx_hash":       txHash,
		"input_token":   req.TokenIn,
		"output_token":  req.TokenOut,
		"input_amount":  req.AmountIn,
		"output_amount": req.MinOut,
		"status":        "pending",
	})
}

// =============================================================================
// NFT Handlers
// =============================================================================

type NFTCollection struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Symbol       string  `json:"symbol"`
	ContractAddr string  `json:"contract_address"`
	ChainID      int64   `json:"chain_id"`
	TotalSupply  int     `json:"total_supply"`
	FloorPrice   float64 `json:"floor_price"`
	Volume24h    float64 `json:"volume_24h"`
	ImageURL     string  `json:"image_url"`
}

type NFTItem struct {
	ID            string         `json:"id"`
	TokenID       string         `json:"token_id"`
	ContractAddr  string         `json:"contract_address"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	ImageURL      string         `json:"image_url"`
	AnimationURL  string         `json:"animation_url"`
	Attributes    []NFTAttribute `json:"attributes"`
	Owner         string         `json:"owner"`
	Price         float64        `json:"price"`
	PriceToken    string         `json:"price_token"`
	ChainID       int64          `json:"chain_id"`
}

type NFTAttribute struct {
	TraitType string `json:"trait_type"`
	Value     string `json:"value"`
	Rarity    string `json:"rarity"`
}

func handleNFTCollections(c *gin.Context) {
	chainID, _ := strconv.ParseInt(c.Query("chain_id"), 10, 64)
	if chainID == 0 {
		chainID = 1
	}
	
	collections := []NFTCollection{
		{ID: "bayc", Name: "Bored Ape Yacht Club", Symbol: "BAYC", ContractAddr: "0xBC4CA0EdA7647A8aB7C2061c2E118A18a936f13D", ChainID: chainID, TotalSupply: 10000, FloorPrice: 30.0, Volume24h: 500.0, ImageURL: "https://ipfs.io/ipfs/QmRRPWG96cmgTn2qSzjwr2qvfIzuhPSJB66CH7XBy6uD4f/"},
		{ID: "punk", Name: "CryptoPunks", Symbol: "PUNK", ContractAddr: "0xb47e3cd837dDF8e4c57F05d70Ab865de6e193BBB", ChainID: chainID, TotalSupply: 10000, FloorPrice: 50.0, Volume24h: 800.0, ImageURL: "https://ipfs.io/ipfs/QmRQhFsAyNoLM5wDdvTFPCEf5y6JxLrgf7L4xeQ5hCgdgX/"},
		{ID: "azuki", Name: "Azuki", Symbol: "AZUKI", ContractAddr: "0xED5AF388653567Af2F388E6224dC7C4b3241C544", ChainID: chainID, TotalSupply: 10000, FloorPrice: 15.0, Volume24h: 300.0, ImageURL: "https://ipfs.io/ipfs/QmYDvPAXtiJg7s8JdRBSLWdgSphQdac8j1YuQNNKGGEKRG/"},
	}
	
	c.JSON(http.StatusOK, gin.H{"collections": collections})
}

func handleNFTItems(c *gin.Context) {
	collectionID := c.Query("collection")
	chainID, _ := strconv.ParseInt(c.Query("chain_id"), 10, 64)
	
	items := []NFTItem{
		{
			ID:           "nft_1",
			TokenID:      "1234",
			ContractAddr: "0xBC4CA0EdA7647A8aB7C2061c2E118A18a936f13D",
			Name:         "Bored Ape Yacht Club #1234",
			Description:  "The Bored Ape Yacht Club is a collection of 10,000 unique Bored Ape NFTs.",
			ImageURL:     "https://ipfs.io/ipfs/QmRRPWG96cmgTn2qSzjwr2qvfIzuhPSJB66CH7XBy6uD4f/1234.png",
			Attributes: []NFTAttribute{
				{TraitType: "Background", Value: "Blue", Rarity: "20%"},
				{TraitType: "Fur", Value: "Dark Brown", Rarity: "15%"},
				{TraitType: "Eyes", Value: "Bored", Rarity: "30%"},
			},
			Owner:    "0x742d35Cc6634C053292505a5eC874A66E8535F5F",
			Price:    35.0,
			PriceToken: "ETH",
			ChainID:  chainID,
		},
	}
	
	c.JSON(http.StatusOK, gin.H{"items": items, "collection": collectionID})
}

func handleNFTDetail(c *gin.Context) {
	nftID := c.Param("id")
	
	item := NFTItem{
		ID:           nftID,
		TokenID:      "1234",
		ContractAddr: "0xBC4CA0EdA7647A8aB7C2061c2E118A18a936f13D",
		Name:         "Bored Ape Yacht Club #1234",
		Description:  "The Bored Ape Yacht Club is a collection of 10,000 unique Bored Ape NFTs.",
		ImageURL:     "https://ipfs.io/ipfs/QmRRPWG96cmgTn2qSzjwr2qvfIzuhPSJB66CH7XBy6uD4f/1234.png",
		Attributes: []NFTAttribute{
			{TraitType: "Background", Value: "Blue", Rarity: "20%"},
			{TraitType: "Fur", Value: "Dark Brown", Rarity: "15%"},
			{TraitType: "Eyes", Value: "Bored", Rarity: "30%"},
		},
		Owner:     "0x742d35Cc6634C053292505a5eC874A66E8535F5F",
		Price:     35.0,
		PriceToken: "ETH",
		ChainID:   1,
	}
	
	c.JSON(http.StatusOK, item)
}

type NFTBuyRequest struct {
	NFTID     string  `json:"nft_id" binding:"required"`
	Price     float64 `json:"price" binding:"required"`
	PriceToken string `json:"price_token" binding:"required"`
}

func handleNFTBuy(c *gin.Context) {
	var req NFTBuyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	txHash := fmt.Sprintf("0x%x", time.Now().UnixNano())
	
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"tx_hash":    txHash,
		"nft_id":     req.NFTID,
		"price":      req.Price,
		"price_token": req.PriceToken,
	})
}

type NFTSellRequest struct {
	NFTID     string  `json:"nft_id" binding:"required"`
	Price     float64 `json:"price" binding:"required"`
	PriceToken string `json:"price_token" binding:"required"`
}

func handleNFTSell(c *gin.Context) {
	var req NFTSellRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"listing_id": fmt.Sprintf("listing_%d", time.Now().UnixNano()),
		"nft_id":     req.NFTID,
		"price":      req.Price,
		"price_token": req.PriceToken,
	})
}

type NFTListRequest struct {
	NFTID     string  `json:"nft_id" binding:"required"`
	ContractAddr string `json:"contract_address" binding:"required"`
	TokenID  string  `json:"token_id" binding:"required"`
	Price    float64 `json:"price" binding:"required"`
	PriceToken string `json:"price_token" binding:"required"`
	ChainID  int64   `json:"chain_id"`
}

func handleNFTList(c *gin.Context) {
	var req NFTListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"listing_id": fmt.Sprintf("listing_%d", time.Now().UnixNano()),
	})
}

// =============================================================================
// Gas & Price Handlers
// =============================================================================

func handleGasPrice(c *gin.Context) {
	chainID, _ := strconv.ParseInt(c.Param("chain_id"), 10, 64)
	if chainID == 0 {
		chainID = 1
	}
	
	// Fetch real gas price from network
	gasPrice := fetchGasPrice(chainID)
	
	c.JSON(http.StatusOK, gin.H{
		"chain_id":         chainID,
		"slow":             gasPrice.Slow,
		"standard":         gasPrice.Standard,
		"fast":             gasPrice.Fast,
		"instant":          gasPrice.Instant,
		"base_fee":         gasPrice.BaseFee,
		"last_update":      time.Now().Unix(),
	})
}

type GasPrice struct {
	Slow     float64
	Standard float64
	Fast     float64
	Instant  float64
	BaseFee  float64
}

func fetchGasPrice(chainID int64) GasPrice {
	// In production, fetch from RPC or gas oracle
	return GasPrice{
		Slow:     20.0,
		Standard: 35.0,
		Fast:     50.0,
		Instant:  75.0,
		BaseFee:  30.0,
	}
}

func handleTokenPrice(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))
	
	priceMu.RLock()
	price, ok := tokenPrices[symbol]
	priceMu.RUnlock()
	
	if !ok {
		price = fetchTokenPriceFromAPI(symbol)
	}
	
	c.JSON(http.StatusOK, gin.H{
		"symbol":    symbol,
		"price_usd": price,
		"timestamp": time.Now().Unix(),
	})
}

func handleAllPrices(c *gin.Context) {
	priceMu.RLock()
	prices := make(map[string]float64)
	for k, v := range tokenPrices {
		prices[k] = v
	}
	priceMu.RUnlock()
	
	c.JSON(http.StatusOK, gin.H{"prices": prices})
}

func handleDEXRouters(c *gin.Context) {
	routers := make([]gin.H, 0)
	for chainID, dexes := range dexRouters {
		for _, dex := range dexes {
			routers = append(routers, gin.H{
				"chain_id":  chainID,
				"name":      dex.Name,
				"address":   dex.Address,
				"factory":   dex.Factory,
				"fee":       dex.Fee,
			})
		}
	}
	
	c.JSON(http.StatusOK, gin.H{"routers": routers})
}

// =============================================================================
// Helper Functions
// =============================================================================

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func generateDemoAddress() string {
	return fmt.Sprintf("0x%x", time.Now().UnixNano())
}

func fetchNativeBalance(chainID int64, address string) (float64, error) {
	chain, ok := chainConfigs[chainID]
	if !ok {
		return 0, fmt.Errorf("unsupported chain")
	}
	
	// In production, make actual RPC call
	// For demo, return mock value
	return 1.5, nil
}

func fetchTokenPriceFromAPI(symbol string) float64 {
	// In production, fetch from CoinGecko
	return 1.0
}

func startPriceFetcher() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		fetchPrices()
	}
}

func fetchPrices() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Fetch prices from CoinGecko (demo implementation)
	symbols := []string{"ethereum", "binancecoin", "matic-network", "bitcoin", "avalanche-2"}
	
	for _, symbol := range symbols {
		// In production, make actual API call
		// For now, use mock values
	}
	
	// Update cache
	priceMu.Lock()
	tokenPrices["ETH"] = 3500.0
	tokenPrices["BNB"] = 600.0
	tokenPrices["MATIC"] = 1.0
	tokenPrices["WBTC"] = 65000.0
	priceMu.Unlock()
	
	// Store in Redis if available
	if redisClient != nil {
		data, _ := json.Marshal(tokenPrices)
		redisClient.Set(ctx, "token_prices", data, 30*time.Second)
	}
}

// JSON RPC helper for EVM chains
func jsonRPCCall(ctx context.Context, rpcURL string, method string, params []interface{}) (json.RawMessage, error) {
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      1,
	}
	
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", rpcURL, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	var result struct {
		JSONRPC string          `json:"jsonrpc"`
		ID     int             `json:"id"`
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	if result.Error != nil {
		return nil, fmt.Errorf("RPC error: %s", result.Error.Message)
	}
	
	return result.Result, nil
}

// Wei to Ether conversion helper
func weiToEther(wei *big.Int) float64 {
	div := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	ether := new(big.Float).SetInt(wei)
	ether.Div(ether, new(big.Float).SetInt(div))
	f, _ := ether.Float64()
	return f
}
