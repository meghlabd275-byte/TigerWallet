package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := loadConfig()

	// Initialize database
	db, err := initDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize Redis cache
	rdb := initRedis(cfg)

	// Initialize Gin router
	router := gin.Default()

	// CORS
	router.Use(corsMiddleware())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "tiger-project-party"})
	})

	// API routes
	api := router.Group("/api/v1")
	{
		// Discovery routes (frontend-facing)
		api.GET("/coins", listCoinsHandler(db, rdb))
		api.GET("/search", searchTokensHandler(db, rdb))
		api.GET("/featured", getFeaturedHandler(db, rdb))
		api.GET("/trending", getTrendingHandler(db, rdb))
		api.GET("/market", getMarketHandler(db, rdb))
		favorites := api.Group("/favorites")
		{
			favorites.GET("", listFavoritesHandler(db))
			favorites.POST("", addFavoriteHandler(db))
			favorites.DELETE("/:id", removeFavoriteHandler(db))
		}

		// Auth (lightweight JWT)
		auth := api.Group("/auth")
		{
			auth.POST("/login", loginHandler(rdb))
		}

		// Token Management
		tokens := api.Group("/tokens")
		{
			tokens.POST("", createTokenHandler(db))
			tokens.GET("", listTokensHandler(db))
			tokens.GET("/:id", getTokenHandler(db))
			tokens.PUT("/:id", updateTokenHandler(db))
			tokens.DELETE("/:id", deleteTokenHandler(db))
			tokens.POST("/:id/submit", submitTokenHandler(db))
			tokens.POST("/:id/approve", approveTokenHandler(db))
			tokens.POST("/:id/reject", rejectTokenHandler(db))
		}

		// Token Listings
		listings := api.Group("/listings")
		{
			listings.POST("", createListingHandler(db))
			listings.GET("", listListingsHandler(db))
			listings.GET("/:id", getListingHandler(db))
			listings.PUT("/:id/status", updateListingStatusHandler(db))
			listings.POST("/:id/featured", featureListingHandler(db))
		}

		// Launchpad
		launchpad := api.Group("/launchpad")
		{
			launchpad.POST("/create", createLaunchpadHandler(db))
			launchpad.GET("", listLaunchpadsHandler(db))
			launchpad.GET("/:id", getLaunchpadHandler(db))
			launchpad.POST("/:id/contribute", contributeHandler(db))
			launchpad.POST("/:id/claim", claimTokensHandler(db))
			launchpad.POST("/:id/cancel", cancelLaunchpadHandler(db))
		}

		// Market Making
		marketmaking := api.Group("/market-making")
		{
			marketmaking.POST("/orders", createMakerOrdersHandler(db))
			marketmaking.GET("/orders", getMakerOrdersHandler(db))
			marketmaking.PUT("/orders/:id/status", updateOrderStatusHandler(db))
			marketmaking.GET("/status/:token_id", getMarketMakerStatusHandler(db))
			marketmaking.POST("/liquidity/add", addLiquidityHandler(db))
			marketmaking.POST("/liquidity/remove", removeLiquidityHandler(db))
		}

		// Pricing
		pricing := api.Group("/pricing")
		{
			pricing.POST("/set", setTokenPriceHandler(db))
			pricing.GET("/:token_id", getTokenPriceHandler(db))
			pricing.GET("/history/:token_id", getPriceHistoryHandler(db))
			pricing.POST("/update", updatePriceHandler(db))
		}

		// Analytics
		analytics := api.Group("/analytics")
		{
			analytics.GET("/volume", getTradingVolumeHandler(db))
			analytics.GET("/liquidity", getLiquidityHandler(db))
			analytics.GET("/holders", getHolderCountHandler(db))
			analytics.GET("/transactions", getTransactionCountHandler(db))
		}

		// Compliance
		compliance := api.Group("/compliance")
		{
			compliance.POST("/audit", requestAuditHandler(db))
			compliance.GET("/audit/:token_id", getAuditStatusHandler(db))
			compliance.POST("/kyc/submit", submitKYCHandler(db))
			compliance.GET("/kyc/:token_id", getKYCStatusHandler(db))
		}

		// Fees
		fees := api.Group("/fees")
		{
			fees.GET("", getListingFeesHandler(db))
			fees.POST("/calculate", calculateFeesHandler(db))
			fees.POST("/pay", payFeesHandler(db))
		}
	}

	// Start server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: router,
	}

	go func() {
		log.Printf("ProjectParty service starting on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
}

// ============== Configuration ==============

type Config struct {
	Port     string
	Database DatabaseConfig
	Redis    RedisConfig
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

func loadConfig() *Config {
	return &Config{
		Port: getEnv("PROJECT_PARTY_PORT", "8106"),
		Database: DatabaseConfig{
			Host:     getEnv("PROJECT_PARTY_DB_HOST", "localhost"),
			Port:     getEnvInt("PROJECT_PARTY_DB_PORT", 5432),
			User:     getEnv("PROJECT_PARTY_DB_USER", "tigerwallet"),
			Password: getEnv("PROJECT_PARTY_DB_PASSWORD", "password"),
			DBName:   getEnv("PROJECT_PARTY_DB_NAME", "tigerwallet_project_party"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("PROJECT_PARTY_REDIS_ADDR", "localhost:6379"),
			Password: getEnv("PROJECT_PARTY_REDIS_PASSWORD", ""),
			DB:       getEnvInt("PROJECT_PARTY_REDIS_DB", 0),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		if _, err := fmt.Sscan(value, &intValue); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// ============== Models ==============

type Token struct {
	ID                uuid.UUID  `json:"id" db:"id"`
	TenantID         uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Name             string    `json:"name" db:"name"`
	Symbol           string    `json:"symbol" db:"symbol"`
	Decimals         int       `json:"decimals" db:"decimals"`
	ContractAddress  string    `json:"contract_address" db:"contract_address"`
	Chain            string    `json:"chain" db:"chain"`
	TotalSupply      string    `json:"total_supply" db:"total_supply"`
	LogoURL          string    `json:"logo_url" db:"logo_url"`
	Description      string    `json:"description" db:"description"`
	Website          string    `json:"website" db:"website"`
	Whitepaper      string    `json:"whitepaper" db:"whitepaper"`
	SocialLinks      map[string]string `json:"social_links" db:"social_links"`
	Status           string    `json:"status" db:"status"` // draft, submitted, in_review, approved, rejected, listed
	SubmissionDate   *time.Time `json:"submission_date" db:"submission_date"`
	ReviewerID       *uuid.UUID `json:"reviewer_id" db:"reviewer_id"`
	ReviewedAt       *time.Time `json:"reviewed_at" db:"reviewed_at"`
	RejectionReason  *string   `json:"rejection_reason" db:"rejection_reason"`
	ListingFeeUSD    float64   `json:"listing_fee_usd" db:"listing_fee_usd"`
	IsFeatured       bool      `json:"is_featured" db:"is_featured"`
	LaunchpadID      *uuid.UUID `json:"launchpad_id" db:"launchpad_id"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

type TokenListing struct {
	ID                uuid.UUID `json:"id" db:"id"`
	TokenID          uuid.UUID `json:"token_id" db:"token_id"`
	TenantID         uuid.UUID `json:"tenant_id" db:"tenant_id"`
	PairToken        string    `json:"pair_token" db:"pair_token"` // e.g., "USDT", "ETH"
	InitialPrice     string    `json:"initial_price" db:"initial_price"`
	CurrentPrice     string    `json:"current_price" db:"current_price"`
	LaunchType       string    `json:"launch_type" db:"launch_type"` // fair_launch, presale, farming
	StartTime        time.Time `json:"start_time" db:"start_time"`
	EndTime          time.Time `json:"end_time" db:"end_time"`
	Status           string    `json:"status" db:"status"` // upcoming, active, completed, cancelled
	Volume24h        string    `json:"volume_24h" db:"volume_24h"`
	LiquidityUSD     string    `json:"liquidity_usd" db:"liquidity_usd"`
	MarketCap        string    `json:"market_cap" db:"market_cap"`
	PriceChange24h   float64   `json:"price_change_24h" db:"price_change_24h"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

type Launchpad struct {
	ID              uuid.UUID `json:"id" db:"id"`
	TokenID         uuid.UUID `json:"token_id" db:"token_id"`
	TenantID        uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Name            string    `json:"name" db:"name"`
	Description     string    `json:"description" db:"description"`
	SoftCap         string    `json:"soft_cap" db:"soft_cap"`
	HardCap         string    `json:"hard_cap" db:"hard_cap"`
	MinContribution string    `json:"min_contribution" db:"min_contribution"`
	MaxContribution string    `json:"max_contribution" db:"max_contribution"`
	StartTime       time.Time `json:"start_time" db:"start_time"`
	EndTime         time.Time `json:"end_time" db:"end_time"`
	TokenPrice      string    `json:"token_price" db:"token_price"`
	AcceptedPayment string    `json:"accepted_payment" db:"accepted_payment"` // USDT, BNB, ETH
	TotalRaised     string    `json:"total_raised" db:"total_raised"`
	Status          string    `json:"status" db:"status"` // upcoming, active, completed, cancelled, refunded
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

type LaunchpadContribution struct {
	ID            uuid.UUID `json:"id" db:"id"`
	LaunchpadID  uuid.UUID `json:"launchpad_id" db:"launchpad_id"`
	UserID       uuid.UUID `json:"user_id" db:"user_id"`
	Amount       string    `json:"amount" db:"amount"`
	TokenAmount  string    `json:"token_amount" db:"token_amount"`
	Status       string    `json:"status" db:"status"` // pending, claimed, refunded
	ClaimedAt    *time.Time `json:"claimed_at" db:"claimed_at"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
}

type MarketMakerOrder struct {
	ID           uuid.UUID `json:"id" db:"id"`
	TenantID    uuid.UUID `json:"tenant_id" db:"tenant_id"`
	TokenID     uuid.UUID `json:"token_id" db:"token_id"`
	Side        string    `json:"side" db:"side"` // buy, sell
	Price       string    `json:"price" db:"price"`
	Quantity    string    `json:"quantity" db:"quantity"`
	Remaining   string    `json:"remaining" db:"remaining"`
	Status      string    `json:"status" db:"status"` // pending, filled, cancelled
	FilledAt    *time.Time `json:"filled_at" db:"filled_at"`
	ExpiresAt   time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type TokenPrice struct {
	ID        uuid.UUID `json:"id" db:"id"`
	TokenID  uuid.UUID `json:"token_id" db:"token_id"`
	Price    string    `json:"price" db:"price"`
	Change24h float64   `json:"change_24h" db:"change_24h"`
	Volume24h string    `json:"volume_24h" db:"volume_24h"`
	Timestamp time.Time `json:"timestamp" db:"timestamp"`
}

type ComplianceAudit struct {
	ID            uuid.UUID `json:"id" db:"id"`
	TokenID       uuid.UUID `json:"token_id" db:"token_id"`
	AuditType     string    `json:"audit_type" db:"audit_type"` // security, code, financial
	Status        string    `json:"status" db:"status"` // requested, in_progress, completed, failed
	ReportURL     *string   `json:"report_url" db:"report_url"`
	Auditor       string    `json:"auditor" db:"auditor"`
	CompletedAt   *time.Time `json:"completed_at" db:"completed_at"`
	RequestedAt   time.Time `json:"requested_at" db:"requested_at"`
}

// ============== HTTP Handlers ==============

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// ---- Redis cache helpers ----

func cacheGet(rdb *redis.Client, key string, dst interface{}) bool {
	if rdb == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	val, err := rdb.Get(ctx, key).Bytes()
	if err != nil {
		return false
	}
	return json.Unmarshal(val, dst) == nil
}

func cacheSet(rdb *redis.Client, key string, val interface{}, ttl time.Duration) {
	if rdb == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if b, err := json.Marshal(val); err == nil {
		rdb.Set(ctx, key, b, ttl)
	}
}

// ---- Discovery handlers (frontend-facing) ----

func listCoinsHandler(db *pgxpool.Pool, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		network := c.Query("network")
		cacheKey := "coins:" + network
		var cached []Token
		if cacheGet(rdb, cacheKey, &cached) {
			c.JSON(http.StatusOK, gin.H{"coins": cached})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		var rows pgx.Rows
		var err error
		if network != "" {
			rows, err = db.Query(ctx, `SELECT id, name, symbol, decimals, contract_address, chain, total_supply, logo_url, description, website, status, is_featured, created_at FROM tokens WHERE ($1='' OR chain=$1) ORDER BY is_featured DESC, created_at DESC LIMIT 200`, network)
		} else {
			rows, err = db.Query(ctx, `SELECT id, name, symbol, decimals, contract_address, chain, total_supply, logo_url, description, website, status, is_featured, created_at FROM tokens ORDER BY is_featured DESC, created_at DESC LIMIT 200`)
		}
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable", "detail": err.Error()})
			return
		}
		defer rows.Close()
		coins := []Token{}
		for rows.Next() {
			var t Token
			if err := rows.Scan(&t.ID, &t.Name, &t.Symbol, &t.Decimals, &t.ContractAddress, &t.Chain, &t.TotalSupply, &t.LogoURL, &t.Description, &t.Website, &t.Status, &t.IsFeatured, &t.CreatedAt); err == nil {
				coins = append(coins, t)
			}
		}
		cacheSet(rdb, cacheKey, coins, 30*time.Second)
		c.JSON(http.StatusOK, gin.H{"coins": coins})
	}
}

func searchTokensHandler(db *pgxpool.Pool, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		q := c.Query("q")
		if q == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "query param 'q' required"})
			return
		}
		cacheKey := "search:" + q
		var cached []Token
		if cacheGet(rdb, cacheKey, &cached) {
			c.JSON(http.StatusOK, gin.H{"results": cached})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		pattern := "%" + q + "%"
		rows, err := db.Query(ctx, `SELECT id, name, symbol, decimals, contract_address, chain, total_supply, logo_url, description, website, status, is_featured, created_at FROM tokens WHERE name ILIKE $1 OR symbol ILIKE $1 OR contract_address ILIKE $1 ORDER BY created_at DESC LIMIT 50`, pattern)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable", "detail": err.Error()})
			return
		}
		defer rows.Close()
		results := []Token{}
		for rows.Next() {
			var t Token
			if err := rows.Scan(&t.ID, &t.Name, &t.Symbol, &t.Decimals, &t.ContractAddress, &t.Chain, &t.TotalSupply, &t.LogoURL, &t.Description, &t.Website, &t.Status, &t.IsFeatured, &t.CreatedAt); err == nil {
				results = append(results, t)
			}
		}
		cacheSet(rdb, cacheKey, results, 30*time.Second)
		c.JSON(http.StatusOK, gin.H{"results": results})
	}
}

func getFeaturedHandler(db *pgxpool.Pool, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var cached []Token
		if cacheGet(rdb, "featured", &cached) {
			c.JSON(http.StatusOK, gin.H{"featured": cached})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		rows, err := db.Query(ctx, `SELECT id, name, symbol, decimals, contract_address, chain, total_supply, logo_url, description, website, status, is_featured, created_at FROM tokens WHERE is_featured=true AND status='listed' ORDER BY created_at DESC LIMIT 20`)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable", "detail": err.Error()})
			return
		}
		defer rows.Close()
		featured := []Token{}
		for rows.Next() {
			var t Token
			if err := rows.Scan(&t.ID, &t.Name, &t.Symbol, &t.Decimals, &t.ContractAddress, &t.Chain, &t.TotalSupply, &t.LogoURL, &t.Description, &t.Website, &t.Status, &t.IsFeatured, &t.CreatedAt); err == nil {
				featured = append(featured, t)
			}
		}
		cacheSet(rdb, "featured", featured, 60*time.Second)
		c.JSON(http.StatusOK, gin.H{"featured": featured})
	}
}

func getTrendingHandler(db *pgxpool.Pool, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var cached []Token
		if cacheGet(rdb, "trending", &cached) {
			c.JSON(http.StatusOK, gin.H{"trending": cached})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		// Trending = listed tokens ordered by 24h volume (from latest price record)
		rows, err := db.Query(ctx, `SELECT t.id, t.name, t.symbol, t.decimals, t.contract_address, t.chain, t.total_supply, t.logo_url, t.description, t.website, t.status, t.is_featured, t.created_at FROM tokens t JOIN token_prices tp ON tp.token_id=t.id WHERE t.status='listed' ORDER BY tp.volume_24h DESC NULLS LAST LIMIT 20`)
		if err != nil {
			// fallback: no price history yet -> listed tokens by created_at
			rows, err = db.Query(ctx, `SELECT id, name, symbol, decimals, contract_address, chain, total_supply, logo_url, description, website, status, is_featured, created_at FROM tokens WHERE status='listed' ORDER BY created_at DESC LIMIT 20`)
			if err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable", "detail": err.Error()})
				return
			}
		}
		defer rows.Close()
		trending := []Token{}
		for rows.Next() {
			var t Token
			if err := rows.Scan(&t.ID, &t.Name, &t.Symbol, &t.Decimals, &t.ContractAddress, &t.Chain, &t.TotalSupply, &t.LogoURL, &t.Description, &t.Website, &t.Status, &t.IsFeatured, &t.CreatedAt); err == nil {
				trending = append(trending, t)
			}
		}
		cacheSet(rdb, "trending", trending, 60*time.Second)
		c.JSON(http.StatusOK, gin.H{"trending": trending})
	}
}

func getMarketHandler(db *pgxpool.Pool, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var cached gin.H
		if cacheGet(rdb, "market", &cached) {
			c.JSON(http.StatusOK, cached)
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		var totalTokens int
		db.QueryRow(ctx, `SELECT COUNT(*) FROM tokens WHERE status='listed'`).Scan(&totalTokens)
		var totalListings int
		db.QueryRow(ctx, `SELECT COUNT(*) FROM token_listings WHERE status IN ('active','upcoming')`).Scan(&totalListings)
		var totalLaunchpads int
		db.QueryRow(ctx, `SELECT COUNT(*) FROM launchpads WHERE status IN ('active','upcoming')`).Scan(&totalLaunchpads)
		var totalVolume24h string
		db.QueryRow(ctx, `SELECT COALESCE(SUM(volume_24h::numeric),0)::text FROM token_listings WHERE status='active'`).Scan(&totalVolume24h)
		market := gin.H{
			"total_tokens":     totalTokens,
			"total_listings":   totalListings,
			"total_launchpads": totalLaunchpads,
			"volume_24h":        totalVolume24h,
		}
		cacheSet(rdb, "market", market, 30*time.Second)
		c.JSON(http.StatusOK, market)
	}
}

// ---- Favorites ----

func listFavoritesHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Query("user_id")
		if userID == "" {
			userID = "anonymous"
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		rows, err := db.Query(ctx, `SELECT t.id, t.name, t.symbol, t.decimals, t.contract_address, t.chain, t.total_supply, t.logo_url, t.description, t.website, t.status, t.is_featured, t.created_at FROM favorites f JOIN tokens t ON t.id=f.token_id WHERE f.user_id=$1 ORDER BY f.created_at DESC`, userID)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable", "detail": err.Error()})
			return
		}
		defer rows.Close()
		favs := []Token{}
		for rows.Next() {
			var t Token
			if err := rows.Scan(&t.ID, &t.Name, &t.Symbol, &t.Decimals, &t.ContractAddress, &t.Chain, &t.TotalSupply, &t.LogoURL, &t.Description, &t.Website, &t.Status, &t.IsFeatured, &t.CreatedAt); err == nil {
				favs = append(favs, t)
			}
		}
		c.JSON(http.StatusOK, gin.H{"favorites": favs})
	}
}

func addFavoriteHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			TokenID string `json:"token_id" binding:"required"`
			UserID  string `json:"user_id"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.UserID == "" {
			req.UserID = "anonymous"
		}
		tokenID, err := uuid.Parse(req.TokenID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token_id"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		if _, err := db.Exec(ctx, `INSERT INTO favorites (id, user_id, token_id, created_at) VALUES ($1,$2,$3,$4) ON CONFLICT (user_id, token_id) DO NOTHING`, uuid.New(), req.UserID, tokenID, time.Now()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable", "detail": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"message": "Favorite added"})
	}
}

func removeFavoriteHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenID := c.Param("id")
		userID := c.Query("user_id")
		if userID == "" {
			userID = "anonymous"
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		if _, err := db.Exec(ctx, `DELETE FROM favorites WHERE user_id=$1 AND token_id=$2`, userID, tokenID); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable", "detail": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Favorite removed"})
	}
}

// ---- Auth (lightweight) ----

func loginHandler(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// Real credential check is handled by the canonical auth service; here we issue a
		// session token bound to the operator so the admin panel can gate mutations.
		token := uuid.New().String()
		if rdb != nil {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			defer cancel()
			rdb.Set(ctx, "session:"+token, req.Username, 24*time.Hour)
		}
		c.JSON(http.StatusOK, gin.H{"token": token, "username": req.Username})
	}
}

// ============== Token Handlers ==============

func createTokenHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			TenantID        string            `json:"tenant_id"`
			Name            string            `json:"name" binding:"required"`
			Symbol          string            `json:"symbol" binding:"required"`
			Decimals        int               `json:"decimals"`
			ContractAddress string            `json:"contract_address"`
			Chain           string            `json:"chain" binding:"required"`
			TotalSupply     string            `json:"total_supply"`
			LogoURL         string            `json:"logo_url"`
			Description     string            `json:"description"`
			Website         string            `json:"website"`
			Whitepaper      string            `json:"whitepaper"`
			SocialLinks     map[string]string `json:"social_links"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		tenantID, _ := uuid.Parse(req.TenantID)
		linksJSON, _ := json.Marshal(req.SocialLinks)
		token := Token{
			ID:              uuid.New(),
			TenantID:        tenantID,
			Name:            req.Name,
			Symbol:          req.Symbol,
			Decimals:        req.Decimals,
			ContractAddress: req.ContractAddress,
			Chain:           req.Chain,
			TotalSupply:     req.TotalSupply,
			LogoURL:         req.LogoURL,
			Description:     req.Description,
			Website:         req.Website,
			Whitepaper:      req.Whitepaper,
			SocialLinks:     req.SocialLinks,
			Status:          "draft",
			ListingFeeUSD:   500.0,
			IsFeatured:      false,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		_, err := db.Exec(ctx, `INSERT INTO tokens (id, tenant_id, name, symbol, decimals, contract_address, chain, total_supply, logo_url, description, website, whitepaper, social_links, status, listing_fee_usd, is_featured, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
			token.ID, token.TenantID, token.Name, token.Symbol, token.Decimals, token.ContractAddress, token.Chain, token.TotalSupply, token.LogoURL, token.Description, token.Website, token.Whitepaper, linksJSON, token.Status, token.ListingFeeUSD, token.IsFeatured, token.CreatedAt, token.UpdatedAt)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable", "detail": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"token": token, "message": "Token created successfully"})
	}
}

func scanTokenRows(rows pgx.Rows) []Token {
	tokens := []Token{}
	for rows.Next() {
		var t Token
		var links []byte
		if err := rows.Scan(&t.ID, &t.TenantID, &t.Name, &t.Symbol, &t.Decimals, &t.ContractAddress, &t.Chain, &t.TotalSupply, &t.LogoURL, &t.Description, &t.Website, &t.Whitepaper, &links, &t.Status, &t.SubmissionDate, &t.ReviewerID, &t.ReviewedAt, &t.RejectionReason, &t.ListingFeeUSD, &t.IsFeatured, &t.LaunchpadID, &t.CreatedAt, &t.UpdatedAt); err == nil {
			if len(links) > 0 {
				json.Unmarshal(links, &t.SocialLinks)
			}
			tokens = append(tokens, t)
		}
	}
	return tokens
}

func listTokensHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := c.Query("status")
		limit := c.DefaultQuery("limit", "50")
		offset := c.DefaultQuery("offset", "0")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		var rows pgx.Rows
		var err error
		if status != "" {
			rows, err = db.Query(ctx, `SELECT id, tenant_id, name, symbol, decimals, contract_address, chain, total_supply, logo_url, description, website, whitepaper, social_links, status, submission_date, reviewer_id, reviewed_at, rejection_reason, listing_fee_usd, is_featured, launchpad_id, created_at, updated_at FROM tokens WHERE status=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, status, limit, offset)
		} else {
			rows, err = db.Query(ctx, `SELECT id, tenant_id, name, symbol, decimals, contract_address, chain, total_supply, logo_url, description, website, whitepaper, social_links, status, submission_date, reviewer_id, reviewed_at, rejection_reason, listing_fee_usd, is_featured, launchpad_id, created_at, updated_at FROM tokens ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
		}
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable", "detail": err.Error()})
			return
		}
		defer rows.Close()
		tokens := scanTokenRows(rows)
		var total int
		db.QueryRow(ctx, `SELECT COUNT(*) FROM tokens`).Scan(&total)
		c.JSON(http.StatusOK, gin.H{"tokens": tokens, "total": total, "limit": limit, "offset": offset})
	}
}

func getTokenHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenID := c.Param("id")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		var t Token
		var links []byte
		err := db.QueryRow(ctx, `SELECT id, tenant_id, name, symbol, decimals, contract_address, chain, total_supply, logo_url, description, website, whitepaper, social_links, status, submission_date, reviewer_id, reviewed_at, rejection_reason, listing_fee_usd, is_featured, launchpad_id, created_at, updated_at FROM tokens WHERE id=$1`, tokenID).Scan(&t.ID, &t.TenantID, &t.Name, &t.Symbol, &t.Decimals, &t.ContractAddress, &t.Chain, &t.TotalSupply, &t.LogoURL, &t.Description, &t.Website, &t.Whitepaper, &links, &t.Status, &t.SubmissionDate, &t.ReviewerID, &t.ReviewedAt, &t.RejectionReason, &t.ListingFeeUSD, &t.IsFeatured, &t.LaunchpadID, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "token not found"})
			return
		}
		if len(links) > 0 {
			json.Unmarshal(links, &t.SocialLinks)
		}
		c.JSON(http.StatusOK, t)
	}
}

func updateTokenHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenID := c.Param("id")
		var req struct {
			Name            string            `json:"name"`
			Symbol          string            `json:"symbol"`
			Decimals        *int              `json:"decimals"`
			ContractAddress string            `json:"contract_address"`
			Chain           string            `json:"chain"`
			TotalSupply     string            `json:"total_supply"`
			LogoURL         string            `json:"logo_url"`
			Description     string            `json:"description"`
			Website         string            `json:"website"`
			Whitepaper      string            `json:"whitepaper"`
			SocialLinks     map[string]string `json:"social_links"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		linksJSON, _ := json.Marshal(req.SocialLinks)
		decimals := 18
		if req.Decimals != nil {
			decimals = *req.Decimals
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		ct, err := db.Exec(ctx, `UPDATE tokens SET name=$1, symbol=$2, decimals=$3, contract_address=$4, chain=$5, total_supply=$6, logo_url=$7, description=$8, website=$9, whitepaper=$10, social_links=$11, updated_at=$12 WHERE id=$13`,
			req.Name, req.Symbol, decimals, req.ContractAddress, req.Chain, req.TotalSupply, req.LogoURL, req.Description, req.Website, req.Whitepaper, linksJSON, time.Now(), tokenID)
		if err != nil || ct.RowsAffected() == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "token not found or update failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Token updated"})
	}
}

func deleteTokenHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenID := c.Param("id")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		ct, err := db.Exec(ctx, `DELETE FROM tokens WHERE id=$1`, tokenID)
		if err != nil || ct.RowsAffected() == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "token not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Token deleted"})
	}
}

func submitTokenHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenID := c.Param("id")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		ct, err := db.Exec(ctx, `UPDATE tokens SET status='submitted', submission_date=$1, updated_at=$1 WHERE id=$2 AND status='draft'`, time.Now(), tokenID)
		if err != nil || ct.RowsAffected() == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "token not found or not in draft status"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Token submitted for review", "status": "submitted"})
	}
}

func approveTokenHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenID := c.Param("id")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		ct, err := db.Exec(ctx, `UPDATE tokens SET status='approved', reviewed_at=$1, updated_at=$1 WHERE id=$2 AND status IN ('submitted','in_review')`, time.Now(), tokenID)
		if err != nil || ct.RowsAffected() == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "token not found or not in a reviewable status"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Token approved", "status": "approved"})
	}
}

func rejectTokenHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenID := c.Param("id")
		var req struct {
			Reason string `json:"reason" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		ct, err := db.Exec(ctx, `UPDATE tokens SET status='rejected', rejection_reason=$1, reviewed_at=$2, updated_at=$2 WHERE id=$3 AND status IN ('submitted','in_review')`, req.Reason, time.Now(), tokenID)
		if err != nil || ct.RowsAffected() == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "token not found or not in a reviewable status"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Token rejected", "status": "rejected", "reason": req.Reason})
	}
}

// ============== Listing Handlers ==============

func createListingHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			TokenID      string `json:"token_id" binding:"required"`
			PairToken    string `json:"pair_token" binding:"required"`
			InitialPrice string `json:"initial_price" binding:"required"`
			LaunchType   string `json:"launch_type" binding:"required"`
			StartTime    string `json:"start_time" binding:"required"`
			EndTime      string `json:"end_time" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		tokenID, err := uuid.Parse(req.TokenID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token_id"})
			return
		}
		start, err := time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_time (use RFC3339)"})
			return
		}
		end, err := time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_time (use RFC3339)"})
			return
		}
		listing := TokenListing{
			ID:            uuid.New(),
			TokenID:       tokenID,
			PairToken:     req.PairToken,
			InitialPrice:  req.InitialPrice,
			CurrentPrice:  req.InitialPrice,
			LaunchType:    req.LaunchType,
			StartTime:     start,
			EndTime:       end,
			Status:        "upcoming",
			Volume24h:     "0",
			LiquidityUSD:  "0",
			MarketCap:     "0",
			PriceChange24h: 0,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		if _, err := db.Exec(ctx, `INSERT INTO token_listings (id, token_id, pair_token, initial_price, current_price, launch_type, start_time, end_time, status, volume_24h, liquidity_usd, market_cap, price_change_24h, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
			listing.ID, listing.TokenID, listing.PairToken, listing.InitialPrice, listing.CurrentPrice, listing.LaunchType, listing.StartTime, listing.EndTime, listing.Status, listing.Volume24h, listing.LiquidityUSD, listing.MarketCap, listing.PriceChange24h, listing.CreatedAt, listing.UpdatedAt); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable", "detail": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"listing": listing, "message": "Listing created"})
	}
}

func listListingsHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		rows, err := db.Query(ctx, `SELECT id, token_id, pair_token, initial_price, current_price, launch_type, start_time, end_time, status, volume_24h, liquidity_usd, market_cap, price_change_24h, created_at, updated_at FROM token_listings ORDER BY created_at DESC LIMIT 100`)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable", "detail": err.Error()})
			return
		}
		defer rows.Close()
		listings := []TokenListing{}
		for rows.Next() {
			var l TokenListing
			if err := rows.Scan(&l.ID, &l.TokenID, &l.PairToken, &l.InitialPrice, &l.CurrentPrice, &l.LaunchType, &l.StartTime, &l.EndTime, &l.Status, &l.Volume24h, &l.LiquidityUSD, &l.MarketCap, &l.PriceChange24h, &l.CreatedAt, &l.UpdatedAt); err == nil {
				listings = append(listings, l)
			}
		}
		c.JSON(http.StatusOK, gin.H{"listings": listings})
	}
}

func getListingHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		var l TokenListing
		err := db.QueryRow(ctx, `SELECT id, token_id, pair_token, initial_price, current_price, launch_type, start_time, end_time, status, volume_24h, liquidity_usd, market_cap, price_change_24h, created_at, updated_at FROM token_listings WHERE id=$1`, c.Param("id")).Scan(&l.ID, &l.TokenID, &l.PairToken, &l.InitialPrice, &l.CurrentPrice, &l.LaunchType, &l.StartTime, &l.EndTime, &l.Status, &l.Volume24h, &l.LiquidityUSD, &l.MarketCap, &l.PriceChange24h, &l.CreatedAt, &l.UpdatedAt)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
			return
		}
		c.JSON(http.StatusOK, l)
	}
}

func updateListingStatusHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Status string `json:"status" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		valid := map[string]bool{"upcoming": true, "active": true, "completed": true, "cancelled": true}
		if !valid[req.Status] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		ct, err := db.Exec(ctx, `UPDATE token_listings SET status=$1, updated_at=$2 WHERE id=$3`, req.Status, time.Now(), c.Param("id"))
		if err != nil || ct.RowsAffected() == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Listing status updated", "status": req.Status})
	}
}

func featureListingHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		ct, err := db.Exec(ctx, `UPDATE tokens SET is_featured=true, updated_at=$1 WHERE id=(SELECT token_id FROM token_listings WHERE id=$2)`, time.Now(), c.Param("id"))
		if err != nil || ct.RowsAffected() == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Listing featured"})
	}
}

// ============== Launchpad Handlers ==============

func createLaunchpadHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			TokenID         string `json:"token_id" binding:"required"`
			Name            string `json:"name" binding:"required"`
			Description     string `json:"description"`
			SoftCap         string `json:"soft_cap" binding:"required"`
			HardCap         string `json:"hard_cap" binding:"required"`
			MinContribution string `json:"min_contribution" binding:"required"`
			MaxContribution string `json:"max_contribution" binding:"required"`
			StartTime       string `json:"start_time" binding:"required"`
			EndTime         string `json:"end_time" binding:"required"`
			TokenPrice      string `json:"token_price" binding:"required"`
			AcceptedPayment string `json:"accepted_payment" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		tokenID, err := uuid.Parse(req.TokenID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token_id"})
			return
		}
		start, err := time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_time"})
			return
		}
		end, err := time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_time"})
			return
		}
		lp := Launchpad{
			ID:              uuid.New(),
			TokenID:         tokenID,
			Name:            req.Name,
			Description:     req.Description,
			SoftCap:         req.SoftCap,
			HardCap:         req.HardCap,
			MinContribution: req.MinContribution,
			MaxContribution: req.MaxContribution,
			StartTime:       start,
			EndTime:         end,
			TokenPrice:      req.TokenPrice,
			AcceptedPayment: req.AcceptedPayment,
			TotalRaised:     "0",
			Status:          "upcoming",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		if _, err := db.Exec(ctx, `INSERT INTO launchpads (id, token_id, name, description, soft_cap, hard_cap, min_contribution, max_contribution, start_time, end_time, token_price, accepted_payment, total_raised, status, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
			lp.ID, lp.TokenID, lp.Name, lp.Description, lp.SoftCap, lp.HardCap, lp.MinContribution, lp.MaxContribution, lp.StartTime, lp.EndTime, lp.TokenPrice, lp.AcceptedPayment, lp.TotalRaised, lp.Status, lp.CreatedAt, lp.UpdatedAt); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable", "detail": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"launchpad": lp, "message": "Launchpad created"})
	}
}

func listLaunchpadsHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		rows, err := db.Query(ctx, `SELECT l.id, l.token_id, l.name, l.description, l.soft_cap, l.hard_cap, l.min_contribution, l.max_contribution, l.start_time, l.end_time, l.token_price, l.accepted_payment, l.total_raised, l.status, l.created_at, l.updated_at, t.symbol FROM launchpads l LEFT JOIN tokens t ON t.id=l.token_id ORDER BY l.created_at DESC LIMIT 100`)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable", "detail": err.Error()})
			return
		}
		defer rows.Close()
		type LaunchpadView struct {
			Launchpad
			Symbol     string  `json:"symbol"`
			Progress   float64 `json:"progress"`
			Contributors int   `json:"contributors"`
		}
		launchpads := []LaunchpadView{}
		for rows.Next() {
			var lv LaunchpadView
			if err := rows.Scan(&lv.ID, &lv.TokenID, &lv.Name, &lv.Description, &lv.SoftCap, &lv.HardCap, &lv.MinContribution, &lv.MaxContribution, &lv.StartTime, &lv.EndTime, &lv.TokenPrice, &lv.AcceptedPayment, &lv.TotalRaised, &lv.Status, &lv.CreatedAt, &lv.UpdatedAt, &lv.Symbol); err == nil {
				launchpads = append(launchpads, lv)
			}
		}
		c.JSON(http.StatusOK, gin.H{"launchpads": launchpads})
	}
}

func getLaunchpadHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		var lp Launchpad
		var symbol string
		err := db.QueryRow(ctx, `SELECT l.id, l.token_id, l.name, l.description, l.soft_cap, l.hard_cap, l.min_contribution, l.max_contribution, l.start_time, l.end_time, l.token_price, l.accepted_payment, l.total_raised, l.status, l.created_at, l.updated_at, t.symbol FROM launchpads l LEFT JOIN tokens t ON t.id=l.token_id WHERE l.id=$1`, c.Param("id")).Scan(&lp.ID, &lp.TokenID, &lp.Name, &lp.Description, &lp.SoftCap, &lp.HardCap, &lp.MinContribution, &lp.MaxContribution, &lp.StartTime, &lp.EndTime, &lp.TokenPrice, &lp.AcceptedPayment, &lp.TotalRaised, &lp.Status, &lp.CreatedAt, &lp.UpdatedAt, &symbol)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "launchpad not found"})
			return
		}
		var contributors int
		db.QueryRow(ctx, `SELECT COUNT(DISTINCT user_id) FROM launchpad_contributions WHERE launchpad_id=$1`, lp.ID).Scan(&contributors)
		c.JSON(http.StatusOK, gin.H{
			"id":            lp.ID,
			"token_id":      lp.TokenID,
			"name":          lp.Name,
			"soft_cap":      lp.SoftCap,
			"hard_cap":      lp.HardCap,
			"total_raised":  lp.TotalRaised,
			"status":        lp.Status,
			"contributors":  contributors,
			"symbol":        symbol,
		})
	}
}

func contributeHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Amount string `json:"amount" binding:"required"`
			UserID string `json:"user_id"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.UserID == "" {
			req.UserID = "anonymous"
		}
		launchpadID := c.Param("id")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		var tokenPrice string
		var status string
		err := db.QueryRow(ctx, `SELECT token_price, status FROM launchpads WHERE id=$1`, launchpadID).Scan(&tokenPrice, &status)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "launchpad not found"})
			return
		}
		if status != "active" && status != "upcoming" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "launchpad not accepting contributions"})
			return
		}
		tokenAmount := "0"
		if tp, e := parseFloat(tokenPrice); e == nil && tp > 0 {
			if amt, e := parseFloat(req.Amount); e == nil {
				tokenAmount = fmt.Sprintf("%.6f", amt/tp)
			}
		}
		contrib := LaunchpadContribution{
			ID:          uuid.New(),
			LaunchpadID: uuid.MustParse(launchpadID),
			UserID:      uuid.New(),
			Amount:      req.Amount,
			TokenAmount: tokenAmount,
			Status:      "pending",
			CreatedAt:   time.Now(),
		}
		uid, _ := uuid.Parse(req.UserID)
		contrib.UserID = uid
		if _, err := db.Exec(ctx, `INSERT INTO launchpad_contributions (id, launchpad_id, user_id, amount, token_amount, status, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			contrib.ID, contrib.LaunchpadID, contrib.UserID, contrib.Amount, contrib.TokenAmount, contrib.Status, contrib.CreatedAt); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable", "detail": err.Error()})
			return
		}
		if _, err := db.Exec(ctx, `UPDATE launchpads SET total_raised=(COALESCE(total_raised::numeric,0)+$1)::text, updated_at=$2 WHERE id=$3`, req.Amount, time.Now(), launchpadID); err != nil {
			log.Printf("warn: update total_raised failed: %v", err)
		}
		c.JSON(http.StatusCreated, gin.H{"contribution": contrib, "message": "Contribution successful"})
	}
}

func claimTokensHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		launchpadID := c.Param("id")
		var req struct {
			UserID string `json:"user_id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		uid, _ := uuid.Parse(req.UserID)
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		ct, err := db.Exec(ctx, `UPDATE launchpad_contributions SET status='claimed', claimed_at=$1 WHERE launchpad_id=$2 AND user_id=$3 AND status='pending'`, time.Now(), launchpadID, uid)
		if err != nil || ct.RowsAffected() == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no claimable contribution found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Tokens claimed successfully"})
	}
}

func cancelLaunchpadHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		ct, err := db.Exec(ctx, `UPDATE launchpads SET status='cancelled', updated_at=$1 WHERE id=$2`, time.Now(), c.Param("id"))
		if err != nil || ct.RowsAffected() == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "launchpad not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Launchpad cancelled", "status": "cancelled"})
	}
}

// ============== Market Making Handlers ==============

func createMakerOrdersHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			TokenID  string `json:"token_id" binding:"required"`
			Side     string `json:"side" binding:"required"`
			Price    string `json:"price" binding:"required"`
			Quantity string `json:"quantity" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.Side != "buy" && req.Side != "sell" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "side must be buy or sell"})
			return
		}
		tokenID, err := uuid.Parse(req.TokenID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token_id"})
			return
		}
		order := MarketMakerOrder{
			ID:        uuid.New(),
			TokenID:   tokenID,
			Side:      req.Side,
			Price:     req.Price,
			Quantity:  req.Quantity,
			Remaining: req.Quantity,
			Status:    "pending",
			ExpiresAt: time.Now().Add(24 * time.Hour),
			CreatedAt: time.Now(),
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		if _, err := db.Exec(ctx, `INSERT INTO market_maker_orders (id, token_id, side, price, quantity, remaining, status, expires_at, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			order.ID, order.TokenID, order.Side, order.Price, order.Quantity, order.Remaining, order.Status, order.ExpiresAt, order.CreatedAt); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable", "detail": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"order": order})
	}
}

func getMakerOrdersHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenID := c.Query("token_id")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		var rows pgx.Rows
		var err error
		if tokenID != "" {
			rows, err = db.Query(ctx, `SELECT id, token_id, side, price, quantity, remaining, status, filled_at, expires_at, created_at FROM market_maker_orders WHERE token_id=$1 ORDER BY created_at DESC LIMIT 100`, tokenID)
		} else {
			rows, err = db.Query(ctx, `SELECT id, token_id, side, price, quantity, remaining, status, filled_at, expires_at, created_at FROM market_maker_orders ORDER BY created_at DESC LIMIT 100`)
		}
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable", "detail": err.Error()})
			return
		}
		defer rows.Close()
		orders := []MarketMakerOrder{}
		for rows.Next() {
			var o MarketMakerOrder
			if err := rows.Scan(&o.ID, &o.TokenID, &o.Side, &o.Price, &o.Quantity, &o.Remaining, &o.Status, &o.FilledAt, &o.ExpiresAt, &o.CreatedAt); err == nil {
				orders = append(orders, o)
			}
		}
		c.JSON(http.StatusOK, gin.H{"orders": orders})
	}
}

func updateOrderStatusHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Status string `json:"status" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		valid := map[string]bool{"pending": true, "filled": true, "cancelled": true}
		if !valid[req.Status] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		var ct pgconn.CommandTag
		var err error
		if req.Status == "filled" {
			ct, err = db.Exec(ctx, `UPDATE market_maker_orders SET status=$1, filled_at=$2, remaining='0' WHERE id=$3`, req.Status, time.Now(), c.Param("id"))
		} else {
			ct, err = db.Exec(ctx, `UPDATE market_maker_orders SET status=$1 WHERE id=$2`, req.Status, c.Param("id"))
		}
		if err != nil || ct.RowsAffected() == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Order status updated"})
	}
}

func getMarketMakerStatusHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenID := c.Param("token_id")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		var totalOrders, filledOrders int
		db.QueryRow(ctx, `SELECT COUNT(*), COUNT(*) FILTER (WHERE status='filled') FROM market_maker_orders WHERE token_id=$1`, tokenID).Scan(&totalOrders, &filledOrders)
		var buyHigh, sellLow *string
		db.QueryRow(ctx, `SELECT MAX(price::numeric) FILTER (WHERE side='buy'), MIN(price::numeric) FILTER (WHERE side='sell') FROM market_maker_orders WHERE token_id=$1 AND status='pending'`, tokenID).Scan(&buyHigh, &sellLow)
		spread := 0.0
		if buyHigh != nil && sellLow != nil {
			if bh, e := parseFloat(*buyHigh); e == nil {
				if sl, e := parseFloat(*sellLow); e == nil && sl > 0 {
					spread = (sl - bh) / sl * 100
				}
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"token_id":      tokenID,
			"active":        totalOrders > 0,
			"spread":        spread,
			"total_orders":  totalOrders,
			"filled_orders": filledOrders,
		})
	}
}

func addLiquidityHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			TokenID   string `json:"token_id" binding:"required"`
			Amount    string `json:"amount" binding:"required"`
			QuoteToken string `json:"quote_token"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		tokenID, err := uuid.Parse(req.TokenID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token_id"})
			return
		}
		amt, err := parseFloat(req.Amount)
		if err != nil || amt <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
			return
		}
		// LP tokens minted proportional to contribution (constant-product proxy)
		lpTokens := fmt.Sprintf("%.6f", amt*1000)
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		if _, err := db.Exec(ctx, `INSERT INTO liquidity_positions (id, token_id, quote_token, lp_tokens, created_at) VALUES ($1,$2,$3,$4,$5)`, uuid.New(), tokenID, req.QuoteToken, lpTokens, time.Now()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable", "detail": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Liquidity added successfully", "lp_tokens": lpTokens})
	}
}

func removeLiquidityHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			TokenID string `json:"token_id" binding:"required"`
			LPAmount string `json:"lp_amount" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		ct, err := db.Exec(ctx, `DELETE FROM liquidity_positions WHERE token_id=$1 AND lp_tokens::numeric <= $2::numeric ORDER BY created_at DESC LIMIT 1`, req.TokenID, req.LPAmount)
		if err != nil || ct.RowsAffected() == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no matching liquidity position"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Liquidity removed successfully"})
	}
}

// ============== Pricing Handlers ==============

func setTokenPriceHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			TokenID string `json:"token_id" binding:"required"`
			Price   string `json:"price" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		tokenID, err := uuid.Parse(req.TokenID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token_id"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		if _, err := db.Exec(ctx, `INSERT INTO token_prices (id, token_id, price, change_24h, volume_24h, timestamp) VALUES ($1,$2,$3,0,'0',$4)`, uuid.New(), tokenID, req.Price, time.Now()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable", "detail": err.Error()})
			return
		}
		if _, err := db.Exec(ctx, `UPDATE token_listings SET current_price=$1, updated_at=$2 WHERE token_id=$3`, req.Price, time.Now(), tokenID); err != nil {
			log.Printf("warn: update listing price failed: %v", err)
		}
		c.JSON(http.StatusOK, gin.H{"message": "Price set", "price": req.Price})
	}
}

func getTokenPriceHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenID := c.Param("token_id")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		var tp TokenPrice
		err := db.QueryRow(ctx, `SELECT id, token_id, price, change_24h, volume_24h, timestamp FROM token_prices WHERE token_id=$1 ORDER BY timestamp DESC LIMIT 1`, tokenID).Scan(&tp.ID, &tp.TokenID, &tp.Price, &tp.Change24h, &tp.Volume24h, &tp.Timestamp)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "price not found"})
			return
		}
		c.JSON(http.StatusOK, tp)
	}
}

func getPriceHistoryHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		rows, err := db.Query(ctx, `SELECT id, token_id, price, change_24h, volume_24h, timestamp FROM token_prices WHERE token_id=$1 ORDER BY timestamp DESC LIMIT 100`, c.Param("token_id"))
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable", "detail": err.Error()})
			return
		}
		defer rows.Close()
		history := []TokenPrice{}
		for rows.Next() {
			var tp TokenPrice
			if err := rows.Scan(&tp.ID, &tp.TokenID, &tp.Price, &tp.Change24h, &tp.Volume24h, &tp.Timestamp); err == nil {
				history = append(history, tp)
			}
		}
		c.JSON(http.StatusOK, gin.H{"history": history})
	}
}

func updatePriceHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			TokenID string `json:"token_id" binding:"required"`
			Price   string `json:"price" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		setTokenPriceHandler(db)(c)
	}
}

// ============== Analytics Handlers ==============

func getTradingVolumeHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		var total24h, total7d, total30d string
		db.QueryRow(ctx, `SELECT COALESCE(SUM(volume_24h::numeric),0)::text FROM token_listings WHERE updated_at > NOW() - INTERVAL '1 day'`).Scan(&total24h)
		db.QueryRow(ctx, `SELECT COALESCE(SUM(volume_24h::numeric),0)::text FROM token_listings WHERE updated_at > NOW() - INTERVAL '7 days'`).Scan(&total7d)
		db.QueryRow(ctx, `SELECT COALESCE(SUM(volume_24h::numeric),0)::text FROM token_listings WHERE updated_at > NOW() - INTERVAL '30 days'`).Scan(&total30d)
		c.JSON(http.StatusOK, gin.H{
			"total_24h": total24h,
			"total_7d":  total7d,
			"total_30d": total30d,
		})
	}
}

func getLiquidityHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		var total string
		db.QueryRow(ctx, `SELECT COALESCE(SUM(liquidity_usd::numeric),0)::text FROM token_listings`).Scan(&total)
		c.JSON(http.StatusOK, gin.H{"total_liquidity": total})
	}
}

func getHolderCountHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenID := c.Query("token_id")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		// Holder count is an aggregate of unique contributors per token's launchpads; honestly 0 when none.
		var total int
		if tokenID != "" {
			db.QueryRow(ctx, `SELECT COUNT(DISTINCT user_id) FROM launchpad_contributions lc JOIN launchpads l ON l.id=lc.launchpad_id WHERE l.token_id=$1`, tokenID).Scan(&total)
		}
		c.JSON(http.StatusOK, gin.H{"token_id": tokenID, "total": total})
	}
}

func getTransactionCountHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		var total24h, total7d int
		db.QueryRow(ctx, `SELECT COUNT(*) FROM launchpad_contributions WHERE created_at > NOW() - INTERVAL '1 day'`).Scan(&total24h)
		db.QueryRow(ctx, `SELECT COUNT(*) FROM launchpad_contributions WHERE created_at > NOW() - INTERVAL '7 days'`).Scan(&total7d)
		c.JSON(http.StatusOK, gin.H{"total_24h": total24h, "total_7d": total7d})
	}
}

// ============== Compliance Handlers ==============

func requestAuditHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			TokenID   string `json:"token_id" binding:"required"`
			AuditType string `json:"audit_type" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		tokenID, err := uuid.Parse(req.TokenID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token_id"})
			return
		}
		valid := map[string]bool{"security": true, "code": true, "financial": true}
		if !valid[req.AuditType] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid audit_type"})
			return
		}
		audit := ComplianceAudit{
			ID:          uuid.New(),
			TokenID:     tokenID,
			AuditType:   req.AuditType,
			Status:      "requested",
			RequestedAt: time.Now(),
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		if _, err := db.Exec(ctx, `INSERT INTO compliance_audits (id, token_id, audit_type, status, auditor, requested_at) VALUES ($1,$2,$3,$4,$5,$6)`, audit.ID, audit.TokenID, audit.AuditType, audit.Status, "", audit.RequestedAt); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable", "detail": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"audit": audit})
	}
}

func getAuditStatusHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenID := c.Param("token_id")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		rows, err := db.Query(ctx, `SELECT id, audit_type, status, report_url, auditor, completed_at, requested_at FROM compliance_audits WHERE token_id=$1 ORDER BY requested_at DESC`, tokenID)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable", "detail": err.Error()})
			return
		}
		defer rows.Close()
		audits := []ComplianceAudit{}
		for rows.Next() {
			var a ComplianceAudit
			if err := rows.Scan(&a.ID, &a.AuditType, &a.Status, &a.ReportURL, &a.Auditor, &a.CompletedAt, &a.RequestedAt); err == nil {
				audits = append(audits, a)
			}
		}
		c.JSON(http.StatusOK, gin.H{"token_id": tokenID, "audits": audits})
	}
}

func submitKYCHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			TokenID string `json:"token_id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		tokenID, err := uuid.Parse(req.TokenID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token_id"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		if _, err := db.Exec(ctx, `INSERT INTO kyc_records (id, token_id, status, submitted_at, expires_at) VALUES ($1,$2,'pending',$3,$4)`, uuid.New(), tokenID, time.Now(), time.Now().Add(365*24*time.Hour)); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable", "detail": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "KYC submitted", "status": "pending"})
	}
}

func getKYCStatusHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenID := c.Param("token_id")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		var status string
		var submittedAt, expiresAt *time.Time
		err := db.QueryRow(ctx, `SELECT status, submitted_at, expires_at FROM kyc_records WHERE token_id=$1 ORDER BY submitted_at DESC LIMIT 1`, tokenID).Scan(&status, &submittedAt, &expiresAt)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "kyc not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"token_id": tokenID, "status": status, "submitted_at": submittedAt, "expires_at": expiresAt})
	}
}

// ============== Fee Handlers ==============

func getListingFeesHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		fees := map[string]float64{}
		rows, err := db.Query(ctx, `SELECT fee_type, amount FROM fee_schedule`)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var ft string
				var amt float64
				if rows.Scan(&ft, &amt) == nil {
					fees[ft] = amt
				}
			}
		}
		if len(fees) == 0 {
			fees = map[string]float64{
				"basic_listing": 500, "featured_listing": 1500, "audit_required": 5000,
				"kyc_verification": 1000, "launchpad_basic": 5000, "launchpad_premium": 15000,
			}
		}
		c.JSON(http.StatusOK, fees)
	}
}

func calculateFeesHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			ListingType string   `json:"listing_type" binding:"required"`
			Features    []string `json:"features"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		total := 500.0
		if req.ListingType == "launchpad" {
			total = 5000.0
		}
		for _, f := range req.Features {
			switch f {
			case "featured":
				total += 1000
			case "audit":
				total += 5000
			case "kyc":
				total += 1000
			}
		}
		c.JSON(http.StatusOK, gin.H{"total_fee": total, "currency": "USD"})
	}
}

func payFeesHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Amount        string `json:"amount" binding:"required"`
			PaymentMethod string `json:"payment_method" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		txID := uuid.New()
		if _, err := db.Exec(ctx, `INSERT INTO fee_payments (id, amount, payment_method, status, created_at) VALUES ($1,$2,$3,'completed',$4)`, txID, req.Amount, req.PaymentMethod, time.Now()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable", "detail": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Payment processed", "transaction_id": txID})
	}
}

// ============== Helpers ==============

func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

var _ = sort.Strings

// ============== Database ==============

func initRedis(cfg *Config) *redis.Client {
	rdb := redis.NewClient(&redis.Options{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password, DB: cfg.Redis.DB})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis unavailable (caching disabled): %v", err)
		return nil
	}
	return rdb
}

var schemaSQL = `
CREATE TABLE IF NOT EXISTS tokens (
	id UUID PRIMARY KEY,
	tenant_id UUID,
	name TEXT NOT NULL,
	symbol TEXT NOT NULL,
	decimals INTEGER DEFAULT 18,
	contract_address TEXT,
	chain TEXT NOT NULL,
	total_supply TEXT,
	logo_url TEXT,
	description TEXT,
	website TEXT,
	whitepaper TEXT,
	social_links JSONB,
	status TEXT NOT NULL DEFAULT 'draft',
	submission_date TIMESTAMPTZ,
	reviewer_id UUID,
	reviewed_at TIMESTAMPTZ,
	rejection_reason TEXT,
	listing_fee_usd NUMERIC DEFAULT 500,
	is_featured BOOLEAN DEFAULT false,
	launchpad_id UUID,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_tokens_status ON tokens(status);
CREATE INDEX IF NOT EXISTS idx_tokens_chain ON tokens(chain);
CREATE INDEX IF NOT EXISTS idx_tokens_symbol ON tokens(symbol);

CREATE TABLE IF NOT EXISTS token_listings (
	id UUID PRIMARY KEY,
	token_id UUID NOT NULL REFERENCES tokens(id) ON DELETE CASCADE,
	tenant_id UUID,
	pair_token TEXT NOT NULL,
	initial_price TEXT,
	current_price TEXT,
	launch_type TEXT,
	start_time TIMESTAMPTZ,
	end_time TIMESTAMPTZ,
	status TEXT NOT NULL DEFAULT 'upcoming',
	volume_24h TEXT DEFAULT '0',
	liquidity_usd TEXT DEFAULT '0',
	market_cap TEXT DEFAULT '0',
	price_change_24h NUMERIC DEFAULT 0,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_listings_status ON token_listings(status);

CREATE TABLE IF NOT EXISTS launchpads (
	id UUID PRIMARY KEY,
	token_id UUID NOT NULL REFERENCES tokens(id) ON DELETE CASCADE,
	tenant_id UUID,
	name TEXT NOT NULL,
	description TEXT,
	soft_cap TEXT,
	hard_cap TEXT,
	min_contribution TEXT,
	max_contribution TEXT,
	start_time TIMESTAMPTZ,
	end_time TIMESTAMPTZ,
	token_price TEXT,
	accepted_payment TEXT,
	total_raised TEXT DEFAULT '0',
	status TEXT NOT NULL DEFAULT 'upcoming',
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_launchpads_status ON launchpads(status);

CREATE TABLE IF NOT EXISTS launchpad_contributions (
	id UUID PRIMARY KEY,
	launchpad_id UUID NOT NULL REFERENCES launchpads(id) ON DELETE CASCADE,
	user_id UUID,
	amount TEXT,
	token_amount TEXT,
	status TEXT DEFAULT 'pending',
	claimed_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_contrib_launchpad ON launchpad_contributions(launchpad_id);
CREATE INDEX IF NOT EXISTS idx_contrib_user ON launchpad_contributions(user_id);

CREATE TABLE IF NOT EXISTS market_maker_orders (
	id UUID PRIMARY KEY,
	tenant_id UUID,
	token_id UUID NOT NULL REFERENCES tokens(id) ON DELETE CASCADE,
	side TEXT NOT NULL,
	price TEXT,
	quantity TEXT,
	remaining TEXT,
	status TEXT DEFAULT 'pending',
	filled_at TIMESTAMPTZ,
	expires_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_mm_orders_token ON market_maker_orders(token_id);

CREATE TABLE IF NOT EXISTS token_prices (
	id UUID PRIMARY KEY,
	token_id UUID NOT NULL REFERENCES tokens(id) ON DELETE CASCADE,
	price TEXT,
	change_24h NUMERIC DEFAULT 0,
	volume_24h TEXT DEFAULT '0',
	timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_prices_token ON token_prices(token_id, timestamp DESC);

CREATE TABLE IF NOT EXISTS compliance_audits (
	id UUID PRIMARY KEY,
	token_id UUID NOT NULL REFERENCES tokens(id) ON DELETE CASCADE,
	audit_type TEXT,
	status TEXT DEFAULT 'requested',
	report_url TEXT,
	auditor TEXT,
	completed_at TIMESTAMPTZ,
	requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_audits_token ON compliance_audits(token_id);

CREATE TABLE IF NOT EXISTS kyc_records (
	id UUID PRIMARY KEY,
	token_id UUID NOT NULL REFERENCES tokens(id) ON DELETE CASCADE,
	status TEXT DEFAULT 'pending',
	submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	expires_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_kyc_token ON kyc_records(token_id);

CREATE TABLE IF NOT EXISTS favorites (
	id UUID PRIMARY KEY,
	user_id TEXT NOT NULL,
	token_id UUID NOT NULL REFERENCES tokens(id) ON DELETE CASCADE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	UNIQUE(user_id, token_id)
);
CREATE INDEX IF NOT EXISTS idx_favorites_user ON favorites(user_id);

CREATE TABLE IF NOT EXISTS liquidity_positions (
	id UUID PRIMARY KEY,
	token_id UUID NOT NULL REFERENCES tokens(id) ON DELETE CASCADE,
	quote_token TEXT,
	lp_tokens TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_liq_pos_token ON liquidity_positions(token_id);

CREATE TABLE IF NOT EXISTS fee_payments (
	id UUID PRIMARY KEY,
	amount TEXT,
	payment_method TEXT,
	status TEXT DEFAULT 'completed',
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS fee_schedule (
	fee_type TEXT PRIMARY KEY,
	amount NUMERIC NOT NULL
);
INSERT INTO fee_schedule (fee_type, amount) VALUES
	('basic_listing', 500), ('featured_listing', 1500), ('audit_required', 5000),
	('kyc_verification', 1000), ('launchpad_basic', 5000), ('launchpad_premium', 15000)
ON CONFLICT (fee_type) DO NOTHING;
`

func initDatabase(cfg *Config) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.Database.User, cfg.Database.Password, cfg.Database.Host,
		cfg.Database.Port, cfg.Database.DBName)

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Apply schema migrations (idempotent)
	if _, err := pool.Exec(context.Background(), schemaSQL); err != nil {
		return nil, fmt.Errorf("failed to apply schema: %w", err)
	}

	log.Println("ProjectParty database initialized with schema migrations applied")
	return pool, nil
}
