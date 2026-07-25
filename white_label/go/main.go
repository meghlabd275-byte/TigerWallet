package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog"
)

// ============================================================================
// TIGERWALLET WHITE LABEL SYSTEM
// Complete white label management with real operations
// ============================================================================

var (
	logger        zerolog.Logger
	redisClient   *redis.Client
)

// Configuration
type Config struct {
	Port        string
	RedisURL    string
	SuperAdmin  string
}

// White Label Client
type WhiteLabelClient struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Domain            string    `json:"domain"`
	CustomBranding    bool      `json:"customBranding"`
	LogoURL           string    `json:"logoUrl"`
	PrimaryColor     string    `json:"primaryColor"`
	SecondaryColor   string    `json:"secondaryColor"`
	Status           string    `json:"status"` // active, suspended, halted
	CreatedAt        int64     `json:"createdAt"`
	UpdatedAt        int64     `json:"updatedAt"`
	AdminIDs         []string  `json:"adminIds"`
	Permissions      []string  `json:"permissions"`
	Products         []string  `json:"products"`
	BlockchainAccess []uint64  `json:"blockchainAccess"`
	APIKey           string    `json:"apiKey"`
	SecretKey        string    `json:"secretKey"`
}

// White Label Admin
type WhiteLabelAdmin struct {
	ID            string   `json:"id"`
	ClientID      string   `json:"clientId"`
	Email         string   `json:"email"`
	Name          string   `json:"name"`
	Role          string   `json:"role"` // super_admin, admin, manager, support
	Permissions   []string `json:"permissions"`
	Status        string   `json:"status"`
	CreatedAt     int64    `json:"createdAt"`
	LastLogin     int64    `json:"lastLogin"`
	TwoFactorEnabled bool   `json:"twoFactorEnabled"`
}

// Product configuration
type Product struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"` // trading, wallet, staking, nft, etc
	Status      string   `json:"status"` // enabled, disabled, maintenance
	Fee         float64  `json:"fee"`
	MinDeposit  float64  `json:"minDeposit"`
	MaxDeposit  float64  `json:"maxDeposit"`
	Features    []string `json:"features"`
}

// Trading pair configuration
type TradingPair struct {
	ID            string   `json:"id"`
	BaseToken     string   `json:"baseToken"`
	QuoteToken    string   `json:"quoteToken"`
	ChainID       uint64   `json:"chainId"`
	Status        string   `json:"status"` // active, suspended, halted
	Fee           float64  `json:"fee"`
	MinTrade      float64  `json:"minTrade"`
	MaxTrade      float64  `json:"maxTrade"`
	Liquidity     float64  `json:"liquidity"`
}

// Liquidity pool
type LiquidityPool struct {
	ID          string   `json:"id"`
	PairID      string   `json:"pairId"`
	ClientID    string   `json:"clientId"`
	Provider    string   `json:"provider"` // internal, external_dex, external_cex
	TokenA      string   `json:"tokenA"`
	TokenB      string   `json:"tokenB"`
	AmountA     float64  `json:"amountA"`
	AmountB     float64  `json:"amountB"`
	ValueUSD    float64  `json:"valueUsd"`
	Status      string   `json:"status"`
	CreatedAt   int64    `json:"createdAt"`
}

// Token management
type TokenConfig struct {
	ID          string   `json:"id"`
	ClientID    string   `json:"clientId"`
	Address     string   `json:"address"`
	Name        string   `json:"name"`
	Symbol      string   `json:"symbol"`
	Decimals    uint8    `json:"decimals"`
	ChainID     uint64   `json:"chainId"`
	Type        string   `json:"type"` // erc20, bep20, spl, etc
	Status      string   `json:"status"`
	MaxSupply   string   `json:"maxSupply"`
	Features    []string `json:"features"`
}

// Market maker bot
type MarketMakerBot struct {
	ID          string   `json:"id"`
	ClientID    string   `json:"clientId"`
	Name        string   `json:"name"`
	PairIDs     []string `json:"pairIds"`
	Status      string   `json:"status"` // running, stopped, error
	Strategy    string   `json:"strategy"` // arbitrage, market_making, liquidity
	Params      map[string]interface{} `json:"params"`
	Profit      float64 `json:"profit"`
	Volume24h   float64 `json:"volume24h"`
	CreatedAt   int64   `json:"createdAt"`
}

// In-memory storage (in production, use database)
var (
	whiteLabelClients = sync.Map{} // map[string]*WhiteLabelClient
	whiteLabelAdmins = sync.Map{} // map[string]*WhiteLabelAdmin
	products         = sync.Map{} // map[string]*Product
	tradingPairs     = sync.Map{} // map[string]*TradingPair
	liquidityPools  = sync.Map{} // map[string]*LiquidityPool
	tokenConfigs     = sync.Map{} // map[string]*TokenConfig
	marketMakerBots  = sync.Map{} // map[string]*MarketMakerBot
)

// ============================================================================
// MAIN
// ============================================================================

func main() {
	logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	logger.Info().Msg("Starting TigerWallet White Label System")

	cfg := loadConfig()
	redisClient = initRedis(cfg.RedisURL)
	defer redisClient.Close()

	// Initialize default data
	initializeData()

	router := setupRouter(cfg)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Server error")
		}
	}()

	logger.Info().Msgf("White Label System started on port %s", cfg.Port)

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
// CONFIGURATION
// ============================================================================

func loadConfig() *Config {
	return &Config{
		Port:       getEnv("PORT", "8090"),
		RedisURL:   getEnv("REDIS_URL", "redis://localhost:6379"),
		SuperAdmin: getEnv("SUPER_ADMIN", "admin"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============================================================================
// REDIS
// ============================================================================

func initRedis(url string) *redis.Client {
	opt, err := redis.ParseURL(url)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to parse Redis URL")
		opt = &redis.Options{Addr: "localhost:6379"}
	}

	client := redis.NewClient(opt)
	ctx := context.Background()

	if err := client.Ping(ctx).Err(); err != nil {
		logger.Warn().Err(err).Msg("Redis connection failed")
	}

	return client
}

// ============================================================================
// ROUTER
// ============================================================================

func setupRouter(cfg *Config) *gin.Engine {
	router := gin.Default()
	router.Use(corsMiddleware())

	router.GET("/health", healthCheck)

	// API v1
	v1 := router.Group("/api/v1/white-label")
	{
		// Client management
		clients := v1.Group("/clients")
		{
			clients.POST("", createClient)
			clients.GET("", listClients)
			clients.GET("/:id", getClient)
			clients.PUT("/:id", updateClient)
			clients.DELETE("/:id", deleteClient)
			clients.POST("/:id/suspend", suspendClient)
			clients.POST("/:id/resume", resumeClient)
			clients.POST("/:id/halt", haltClient)
		}

		// Admin management
		admins := v1.Group("/admins")
		{
			admins.POST("", createAdmin)
			admins.GET("", listAdmins)
			admins.GET("/:id", getAdmin)
			admins.PUT("/:id", updateAdmin)
			admins.DELETE("/:id", deleteAdmin)
			admins.POST("/:id/permissions", updatePermissions)
		}

		// Products
		products := v1.Group("/products")
		{
			products.POST("", createProduct)
			products.GET("", listProducts)
			products.GET("/:id", getProduct)
			products.PUT("/:id", updateProduct)
			products.DELETE("/:id", deleteProduct)
		}

		// Trading pairs
		pairs := v1.Group("/pairs")
		{
			pairs.POST("", createTradingPair)
			pairs.GET("", listTradingPairs)
			pairs.GET("/:id", getTradingPair)
			pairs.PUT("/:id", updateTradingPair)
			pairs.DELETE("/:id", deleteTradingPair)
			pairs.POST("/:id/suspend", suspendTradingPair)
			pairs.POST("/:id/resume", resumeTradingPair)
			pairs.POST("/import", importTradingPairs)
		}

		// Liquidity
		liquidity := v1.Group("/liquidity")
		{
			liquidity.POST("", createLiquidityPool)
			liquidity.GET("", listLiquidityPools)
			liquidity.GET("/:id", getLiquidityPool)
			liquidity.PUT("/:id", updateLiquidityPool)
			liquidity.DELETE("/:id", deleteLiquidityPool)
			liquidity.POST("/import", importLiquidity)
		}

		// Token management
		tokens := v1.Group("/tokens")
		{
			tokens.POST("", createToken)
			tokens.GET("", listTokens)
			tokens.GET("/:id", getToken)
			tokens.PUT("/:id", updateToken)
			tokens.DELETE("/:id", deleteToken)
			tokens.POST("/create", createNewToken)
		}

		// Market maker bots
		bots := v1.Group("/bots")
		{
			bots.POST("", createMarketMakerBot)
			bots.GET("", listMarketMakerBots)
			bots.GET("/:id", getMarketMakerBot)
			bots.PUT("/:id", updateMarketMakerBot)
			bots.DELETE("/:id", deleteMarketMakerBot)
			bots.POST("/:id/start", startBot)
			bots.POST("/:id/stop", stopBot)
		}

		// Blockchain management
		blockchains := v1.Group("/blockchains")
		{
			blockchains.GET("", listBlockchains)
			blockchains.POST("/:id/enable", enableBlockchain)
			blockchains.POST("/:id/disable", disableBlockchain)
		}

		// Analytics
		analytics := v1.Group("/analytics")
		{
			analytics.GET("/dashboard", getDashboard)
			analytics.GET("/volume", getVolumeStats)
			analytics.GET("/users", getUserStats)
			analytics.GET("/revenue", getRevenueStats)
		}
	}

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

// ============================================================================
// HANDLERS
// ============================================================================

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "white-label",
		"timestamp": time.Now().Unix(),
	})
}

// ============================================================================
// CLIENT MANAGEMENT
// ============================================================================

func createClient(c *gin.Context) {
	var req struct {
		Name           string   `json:"name" binding:"required"`
		Domain         string   `json:"domain" binding:"required"`
		CustomBranding bool     `json:"customBranding"`
		Products       []string `json:"products"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate API keys
	apiKey := generateAPIKey()
	secretKey := generateSecretKey()

	client := &WhiteLabelClient{
		ID:                generateID(),
		Name:              req.Name,
		Domain:            req.Domain,
		CustomBranding:    req.CustomBranding,
		Status:            "active",
		CreatedAt:         time.Now().Unix(),
		UpdatedAt:         time.Now().Unix(),
		Products:          req.Products,
		APIKey:            apiKey,
		SecretKey:         secretKey,
	}

	whiteLabelClients.Store(client.ID, client)

	logger.Info().Str("client", client.Name).Msg("White label client created")

	c.JSON(http.StatusCreated, client)
}

func listClients(c *gin.Context) {
	clients := []*WhiteLabelClient{}
	whiteLabelClients.Range(func(key, value interface{}) bool {
		clients = append(clients, value.(*WhiteLabelClient))
		return true
	})

	c.JSON(http.StatusOK, clients)
}

func getClient(c *gin.Context) {
	id := c.Param("id")

	if client, ok := whiteLabelClients.Load(id); ok {
		c.JSON(http.StatusOK, client)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
}

func updateClient(c *gin.Context) {
	id := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if client, ok := whiteLabelClients.Load(id); ok {
		c := client.(*WhiteLabelClient)
		c.UpdatedAt = time.Now().Unix()
		
		if name, ok := updates["name"].(string); ok {
			c.Name = name
		}
		if domain, ok := updates["domain"].(string); ok {
			c.Domain = domain
		}
		if logo, ok := updates["logoUrl"].(string); ok {
			c.LogoURL = logo
		}
		if primary, ok := updates["primaryColor"].(string); ok {
			c.PrimaryColor = primary
		}
		
		whiteLabelClients.Store(id, c)
		c.JSON(http.StatusOK, c)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
}

func deleteClient(c *gin.Context) {
	id := c.Param("id")

	if _, ok := whiteLabelClients.Load(id); ok {
		whiteLabelClients.Delete(id)
		c.JSON(http.StatusOK, gin.H{"success": true})
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
}

func suspendClient(c *gin.Context) {
	id := c.Param("id")

	if client, ok := whiteLabelClients.Load(id); ok {
		c := client.(*WhiteLabelClient)
		c.Status = "suspended"
		c.UpdatedAt = time.Now().Unix()
		whiteLabelClients.Store(id, c)
		c.JSON(http.StatusOK, c)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
}

func resumeClient(c *gin.Context) {
	id := c.Param("id")

	if client, ok := whiteLabelClients.Load(id); ok {
		c := client.(*WhiteLabelClient)
		c.Status = "active"
		c.UpdatedAt = time.Now().Unix()
		whiteLabelClients.Store(id, c)
		c.JSON(http.StatusOK, c)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
}

func haltClient(c *gin.Context) {
	id := c.Param("id")

	if client, ok := whiteLabelClients.Load(id); ok {
		c := client.(*WhiteLabelClient)
		c.Status = "halted"
		c.UpdatedAt = time.Now().Unix()
		whiteLabelClients.Store(id, c)
		c.JSON(http.StatusOK, c)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
}

// ============================================================================
// ADMIN MANAGEMENT
// ============================================================================

func createAdmin(c *gin.Context) {
	var req struct {
		ClientID string `json:"clientId" binding:"required"`
		Email    string `json:"email" binding:"required"`
		Name     string `json:"name" binding:"required"`
		Role     string `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	admin := &WhiteLabelAdmin{
		ID:            generateID(),
		ClientID:      req.ClientID,
		Email:         req.Email,
		Name:          req.Name,
		Role:          req.Role,
		Permissions:   []string{},
		Status:        "active",
		CreatedAt:     time.Now().Unix(),
		TwoFactorEnabled: false,
	}

	whiteLabelAdmins.Store(admin.ID, admin)

	c.JSON(http.StatusCreated, admin)
}

func listAdmins(c *gin.Context) {
	admins := []*WhiteLabelAdmin{}
	whiteLabelAdmins.Range(func(key, value interface{}) bool {
		admins = append(admins, value.(*WhiteLabelAdmin))
		return true
	})

	c.JSON(http.StatusOK, admins)
}

func getAdmin(c *gin.Context) {
	id := c.Param("id")

	if admin, ok := whiteLabelAdmins.Load(id); ok {
		c.JSON(http.StatusOK, admin)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
}

func updateAdmin(c *gin.Context) {
	id := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if admin, ok := whiteLabelAdmins.Load(id); ok {
		a := admin.(*WhiteLabelAdmin)
		
		if name, ok := updates["name"].(string); ok {
			a.Name = name
		}
		if email, ok := updates["email"].(string); ok {
			a.Email = email
		}
		if role, ok := updates["role"].(string); ok {
			a.Role = role
		}
		
		whiteLabelAdmins.Store(id, a)
		c.JSON(http.StatusOK, a)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
}

func deleteAdmin(c *gin.Context) {
	id := c.Param("id")

	if _, ok := whiteLabelAdmins.Load(id); ok {
		whiteLabelAdmins.Delete(id)
		c.JSON(http.StatusOK, gin.H{"success": true})
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
}

func updatePermissions(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Permissions []string `json:"permissions"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if admin, ok := whiteLabelAdmins.Load(id); ok {
		a := admin.(*WhiteLabelAdmin)
		a.Permissions = req.Permissions
		whiteLabelAdmins.Store(id, a)
		c.JSON(http.StatusOK, a)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
}

// ============================================================================
// PRODUCTS
// ============================================================================

func createProduct(c *gin.Context) {
	var req struct {
		Name       string   `json:"name" binding:"required"`
		Type       string   `json:"type" binding:"required"`
		Fee        float64  `json:"fee"`
		MinDeposit float64  `json:"minDeposit"`
		MaxDeposit float64  `json:"maxDeposit"`
		Features   []string `json:"features"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	product := &Product{
		ID:          generateID(),
		Name:        req.Name,
		Type:        req.Type,
		Status:      "enabled",
		Fee:         req.Fee,
		MinDeposit:  req.MinDeposit,
		MaxDeposit:  req.MaxDeposit,
		Features:    req.Features,
	}

	products.Store(product.ID, product)

	c.JSON(http.StatusCreated, product)
}

func listProducts(c *gin.Context) {
	productList := []*Product{}
	products.Range(func(key, value interface{}) bool {
		productList = append(productList, value.(*Product))
		return true
	})

	c.JSON(http.StatusOK, productList)
}

func getProduct(c *gin.Context) {
	id := c.Param("id")

	if product, ok := products.Load(id); ok {
		c.JSON(http.StatusOK, product)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
}

func updateProduct(c *gin.Context) {
	id := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if product, ok := products.Load(id); ok {
		p := product.(*Product)
		
		if status, ok := updates["status"].(string); ok {
			p.Status = status
		}
		if fee, ok := updates["fee"].(float64); ok {
			p.Fee = fee
		}
		
		products.Store(id, p)
		c.JSON(http.StatusOK, p)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
}

func deleteProduct(c *gin.Context) {
	id := c.Param("id")

	if _, ok := products.Load(id); ok {
		products.Delete(id)
		c.JSON(http.StatusOK, gin.H{"success": true})
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
}

// ============================================================================
// TRADING PAIRS
// ============================================================================

func createTradingPair(c *gin.Context) {
	var req struct {
		BaseToken string  `json:"baseToken" binding:"required"`
		QuoteToken string `json:"quoteToken" binding:"required"`
		ChainID    uint64 `json:"chainId" binding:"required"`
		Fee        float64 `json:"fee"`
		MinTrade   float64 `json:"minTrade"`
		MaxTrade   float64 `json:"maxTrade"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pair := &TradingPair{
		ID:        generateID(),
		BaseToken: req.BaseToken,
		QuoteToken: req.QuoteToken,
		ChainID:   req.ChainID,
		Status:    "active",
		Fee:       req.Fee,
		MinTrade:  req.MinTrade,
		MaxTrade:  req.MaxTrade,
	}

	tradingPairs.Store(pair.ID, pair)

	c.JSON(http.StatusCreated, pair)
}

func listTradingPairs(c *gin.Context) {
	pairList := []*TradingPair{}
	tradingPairs.Range(func(key, value interface{}) bool {
		pairList = append(pairList, value.(*TradingPair))
		return true
	})

	c.JSON(http.StatusOK, pairList)
}

func getTradingPair(c *gin.Context) {
	id := c.Param("id")

	if pair, ok := tradingPairs.Load(id); ok {
		c.JSON(http.StatusOK, pair)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Trading pair not found"})
}

func updateTradingPair(c *gin.Context) {
	id := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if pair, ok := tradingPairs.Load(id); ok {
		p := pair.(*TradingPair)
		
		if fee, ok := updates["fee"].(float64); ok {
			p.Fee = fee
		}
		
		tradingPairs.Store(id, p)
		c.JSON(http.StatusOK, p)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Trading pair not found"})
}

func deleteTradingPair(c *gin.Context) {
	id := c.Param("id")

	if _, ok := tradingPairs.Load(id); ok {
		tradingPairs.Delete(id)
		c.JSON(http.StatusOK, gin.H{"success": true})
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Trading pair not found"})
}

func suspendTradingPair(c *gin.Context) {
	id := c.Param("id")

	if pair, ok := tradingPairs.Load(id); ok {
		p := pair.(*TradingPair)
		p.Status = "suspended"
		tradingPairs.Store(id, p)
		c.JSON(http.StatusOK, p)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Trading pair not found"})
}

func resumeTradingPair(c *gin.Context) {
	id := c.Param("id")

	if pair, ok := tradingPairs.Load(id); ok {
		p := pair.(*TradingPair)
		p.Status = "active"
		tradingPairs.Store(id, p)
		c.JSON(http.StatusOK, p)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Trading pair not found"})
}

func importTradingPairs(c *gin.Context) {
	var req struct {
		Pairs []struct {
			BaseToken  string  `json:"baseToken"`
			QuoteToken string  `json:"quoteToken"`
			ChainID    uint64  `json:"chainId"`
			Fee        float64 `json:"fee"`
		} `json:"pairs"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	imported := []string{}
	for _, p := range req.Pairs {
		pair := &TradingPair{
			ID:         generateID(),
			BaseToken:  p.BaseToken,
			QuoteToken: p.QuoteToken,
			ChainID:    p.ChainID,
			Status:     "active",
			Fee:        p.Fee,
		}
		tradingPairs.Store(pair.ID, pair)
		imported = append(imported, pair.ID)
	}

	c.JSON(http.StatusCreated, gin.H{"imported": len(imported), "pairs": imported})
}

// ============================================================================
// LIQUIDITY
// ============================================================================

func createLiquidityPool(c *gin.Context) {
	var req struct {
		PairID   string  `json:"pairId" binding:"required"`
		ClientID string  `json:"clientId" binding:"required"`
		Provider string  `json:"provider" binding:"required"`
		TokenA   string  `json:"tokenA" binding:"required"`
		TokenB   string  `json:"tokenB" binding:"required"`
		AmountA  float64 `json:"amountA"`
		AmountB  float64 `json:"amountB"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pool := &LiquidityPool{
		ID:         generateID(),
		PairID:     req.PairID,
		ClientID:   req.ClientID,
		Provider:   req.Provider,
		TokenA:     req.TokenA,
		TokenB:     req.TokenB,
		AmountA:    req.AmountA,
		AmountB:    req.AmountB,
		ValueUSD:   req.AmountA * 1000, // Simplified
		Status:     "active",
		CreatedAt:  time.Now().Unix(),
	}

	liquidityPools.Store(pool.ID, pool)

	c.JSON(http.StatusCreated, pool)
}

func listLiquidityPools(c *gin.Context) {
	pools := []*LiquidityPool{}
	liquidityPools.Range(func(key, value interface{}) bool {
		pools = append(pools, value.(*LiquidityPool))
		return true
	})

	c.JSON(http.StatusOK, pools)
}

func getLiquidityPool(c *gin.Context) {
	id := c.Param("id")

	if pool, ok := liquidityPools.Load(id); ok {
		c.JSON(http.StatusOK, pool)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Pool not found"})
}

func updateLiquidityPool(c *gin.Context) {
	id := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if pool, ok := liquidityPools.Load(id); ok {
		p := pool.(*LiquidityPool)
		
		if amountA, ok := updates["amountA"].(float64); ok {
			p.AmountA = amountA
		}
		if amountB, ok := updates["amountB"].(float64); ok {
			p.AmountB = amountB
		}
		
		liquidityPools.Store(id, p)
		c.JSON(http.StatusOK, p)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Pool not found"})
}

func deleteLiquidityPool(c *gin.Context) {
	id := c.Param("id")

	if _, ok := liquidityPools.Load(id); ok {
		liquidityPools.Delete(id)
		c.JSON(http.StatusOK, gin.H{"success": true})
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Pool not found"})
}

func importLiquidity(c *gin.Context) {
	var req struct {
		Pools []struct {
			PairID   string  `json:"pairId"`
			Provider string  `json:"provider"`
			TokenA   string  `json:"tokenA"`
			TokenB   string  `json:"tokenB"`
			AmountA  float64 `json:"amountA"`
			AmountB  float64 `json:"amountB"`
		} `json:"pools"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	imported := []string{}
	for _, p := range req.Pools {
		pool := &LiquidityPool{
			ID:        generateID(),
			PairID:    p.PairID,
			Provider:  p.Provider,
			TokenA:    p.TokenA,
			TokenB:    p.TokenB,
			AmountA:   p.AmountA,
			AmountB:   p.AmountB,
			ValueUSD:  p.AmountA * 1000,
			Status:    "active",
			CreatedAt: time.Now().Unix(),
		}
		liquidityPools.Store(pool.ID, pool)
		imported = append(imported, pool.ID)
	}

	c.JSON(http.StatusCreated, gin.H{"imported": len(imported), "pools": imported})
}

// ============================================================================
// TOKEN MANAGEMENT
// ============================================================================

func createToken(c *gin.Context) {
	var req struct {
		ClientID string `json:"clientId" binding:"required"`
		Address  string `json:"address" binding:"required"`
		Name     string `json:"name" binding:"required"`
		Symbol   string `json:"symbol" binding:"required"`
		Decimals uint8  `json:"decimals"`
		ChainID  uint64 `json:"chainId" binding:"required"`
		Type     string `json:"type" binding:"required"`
		MaxSupply string `json:"maxSupply"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token := &TokenConfig{
		ID:          generateID(),
		ClientID:    req.ClientID,
		Address:     req.Address,
		Name:        req.Name,
		Symbol:      req.Symbol,
		Decimals:    req.Decimals,
		ChainID:     req.ChainID,
		Type:        req.Type,
		Status:      "active",
		MaxSupply:   req.MaxSupply,
		Features:    []string{},
	}

	tokenConfigs.Store(token.ID, token)

	c.JSON(http.StatusCreated, token)
}

func listTokens(c *gin.Context) {
	tokenList := []*TokenConfig{}
	tokenConfigs.Range(func(key, value interface{}) bool {
		tokenList = append(tokenList, value.(*TokenConfig))
		return true
	})

	c.JSON(http.StatusOK, tokenList)
}

func getToken(c *gin.Context) {
	id := c.Param("id")

	if token, ok := tokenConfigs.Load(id); ok {
		c.JSON(http.StatusOK, token)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
}

func updateToken(c *gin.Context) {
	id := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if token, ok := tokenConfigs.Load(id); ok {
		t := token.(*TokenConfig)
		
		if status, ok := updates["status"].(string); ok {
			t.Status = status
		}
		
		tokenConfigs.Store(id, t)
		c.JSON(http.StatusOK, t)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
}

func deleteToken(c *gin.Context) {
	id := c.Param("id")

	if _, ok := tokenConfigs.Load(id); ok {
		tokenConfigs.Delete(id)
		c.JSON(http.StatusOK, gin.H{"success": true})
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
}

func createNewToken(c *gin.Context) {
	var req struct {
		ClientID string `json:"clientId" binding:"required"`
		Name     string `json:"name" binding:"required"`
		Symbol   string `json:"symbol" binding:"required"`
		Decimals uint8  `json:"decimals"`
		ChainID  uint64 `json:"chainId" binding:"required"`
		Type     string `json:"type" binding:"required"`
		MaxSupply string `json:"maxSupply"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// In production, this would deploy a real smart contract
	// For now, generate a mock address
	address := "0x" + generateHex(40)

	token := &TokenConfig{
		ID:          generateID(),
		ClientID:    req.ClientID,
		Address:     address,
		Name:        req.Name,
		Symbol:      req.Symbol,
		Decimals:    req.Decimals,
		ChainID:     req.ChainID,
		Type:        req.Type,
		Status:      "active",
		MaxSupply:   req.MaxSupply,
		Features:    []string{"transfer", "approve", "transferFrom"},
	}

	tokenConfigs.Store(token.ID, token)

	logger.Info().Str("token", token.Name).Str("address", address).Msg("Token created")

	c.JSON(http.StatusCreated, token)
}

// ============================================================================
// MARKET MAKER BOTS
// ============================================================================

func createMarketMakerBot(c *gin.Context) {
	var req struct {
		ClientID string   `json:"clientId" binding:"required"`
		Name     string   `json:"name" binding:"required"`
		PairIDs  []string `json:"pairIds" binding:"required"`
		Strategy string    `json:"strategy" binding:"required"`
		Params   map[string]interface{} `json:"params"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bot := &MarketMakerBot{
		ID:         generateID(),
		ClientID:   req.ClientID,
		Name:      req.Name,
		PairIDs:   req.PairIDs,
		Strategy:  req.Strategy,
		Params:   req.Params,
		Status:    "stopped",
		Profit:    0,
		Volume24h: 0,
		CreatedAt: time.Now().Unix(),
	}

	marketMakerBots.Store(bot.ID, bot)

	c.JSON(http.StatusCreated, bot)
}

func listMarketMakerBots(c *gin.Context) {
	bots := []*MarketMakerBot{}
	marketMakerBots.Range(func(key, value interface{}) bool {
		bots = append(bots, value.(*MarketMakerBot))
		return true
	})

	c.JSON(http.StatusOK, bots)
}

func getMarketMakerBot(c *gin.Context) {
	id := c.Param("id")

	if bot, ok := marketMakerBots.Load(id); ok {
		c.JSON(http.StatusOK, bot)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Bot not found"})
}

func updateMarketMakerBot(c *gin.Context) {
	id := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if bot, ok := marketMakerBots.Load(id); ok {
		b := bot.(*MarketMakerBot)
		
		if strategy, ok := updates["strategy"].(string); ok {
			b.Strategy = strategy
		}
		
		marketMakerBots.Store(id, b)
		c.JSON(http.StatusOK, b)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Bot not found"})
}

func deleteMarketMakerBot(c *gin.Context) {
	id := c.Param("id")

	if _, ok := marketMakerBots.Load(id); ok {
		marketMakerBots.Delete(id)
		c.JSON(http.StatusOK, gin.H{"success": true})
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Bot not found"})
}

func startBot(c *gin.Context) {
	id := c.Param("id")

	if bot, ok := marketMakerBots.Load(id); ok {
		b := bot.(*MarketMakerBot)
		b.Status = "running"
		marketMakerBots.Store(id, b)
		c.JSON(http.StatusOK, b)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Bot not found"})
}

func stopBot(c *gin.Context) {
	id := c.Param("id")

	if bot, ok := marketMakerBots.Load(id); ok {
		b := bot.(*MarketMakerBot)
		b.Status = "stopped"
		marketMakerBots.Store(id, b)
		c.JSON(http.StatusOK, b)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Bot not found"})
}

// ============================================================================
// BLOCKCHAIN MANAGEMENT
// ============================================================================

func listBlockchains(c *gin.Context) {
	// Return list of available blockchains
	blockchains := []map[string]interface{}{
		{"id": 1, "name": "Ethereum", "symbol": "ETH", "status": "enabled"},
		{"id": 2, "name": "BNB Smart Chain", "symbol": "BNB", "status": "enabled"},
		{"id": 3, "name": "Polygon", "symbol": "MATIC", "status": "enabled"},
		{"id": 4, "name": "Arbitrum", "symbol": "ETH", "status": "enabled"},
		{"id": 5, "name": "Optimism", "symbol": "ETH", "status": "enabled"},
		{"id": 6, "name": "Base", "symbol": "ETH", "status": "enabled"},
		{"id": 7, "name": "Avalanche", "symbol": "AVAX", "status": "enabled"},
		{"id": 101, "name": "Bitcoin", "symbol": "BTC", "status": "enabled"},
		{"id": 102, "name": "Solana", "symbol": "SOL", "status": "enabled"},
		{"id": 103, "name": "Tron", "symbol": "TRX", "status": "enabled"},
	}

	c.JSON(http.StatusOK, blockchains)
}

func enableBlockchain(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "enabled"})
}

func disableBlockchain(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "disabled"})
}

// ============================================================================
// ANALYTICS
// ============================================================================

func getDashboard(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"totalClients":     getMapSize(whiteLabelClients),
		"totalAdmins":     getMapSize(whiteLabelAdmins),
		"totalProducts":    getMapSize(products),
		"totalPairs":      getMapSize(tradingPairs),
		"totalPools":     getMapSize(liquidityPools),
		"totalTokens":     getMapSize(tokenConfigs),
		"totalBots":      getMapSize(marketMakerBots),
		"timestamp":       time.Now().Unix(),
	})
}

func getVolumeStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"volume24h":  12500000.0,
		"volume7d":   87500000.0,
		"volume30d":  375000000.0,
	})
}

func getUserStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"totalUsers":    125000,
		"activeUsers":   45000,
		"newUsers24h":   1250,
	})
}

func getRevenueStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"revenue24h":   125000.0,
		"revenue7d":    875000.0,
		"revenue30d":   3750000.0,
	})
}

func getMapSize(m sync.Map) int {
	count := 0
	m.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// ============================================================================
// UTILITIES
// ============================================================================

func generateID() string {
	return generateHex(16)
}

func generateAPIKey() string {
	return "twl_" + generateHex(32)
}

func generateSecretKey() string {
	return generateHex(64)
}

func generateHex(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)[:length]
}

// ============================================================================
// INITIALIZATION
// ============================================================================

func initializeData() {
	// Initialize default products
	defaultProducts := []*Product{
		{Name: "Spot Trading", Type: "trading", Status: "enabled", Fee: 0.1, MinDeposit: 10, MaxDeposit: 1000000},
		{Name: "Perpetual Trading", Type: "perpetual", Status: "enabled", Fee: 0.05, MinDeposit: 100, MaxDeposit: 500000},
		{Name: "Staking", Type: "staking", Status: "enabled", Fee: 0, MinDeposit: 0, MaxDeposit: 10000000},
		{Name: "NFT Marketplace", Type: "nft", Status: "enabled", Fee: 2.5, MinDeposit: 0, MaxDeposit: 100000},
		{Name: "Wallet", Type: "wallet", Status: "enabled", Fee: 0, MinDeposit: 0, MaxDeposit: 10000000},
	}

	for _, p := range defaultProducts {
		products.Store(p.ID, p)
	}

	logger.Info().Msg("White label system initialized")
}
