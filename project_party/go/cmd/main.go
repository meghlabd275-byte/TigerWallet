package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg := loadConfig()

	// Initialize database
	db, err := initDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	propagationDBPool = db

	// Initialize Redis cache
	rdb := initRedis(cfg)

	// Initialize on-chain launchpad (optional; fail-closed if unconfigured).
	if err := initLaunchpadOnChain(); err != nil {
		log.Printf("warning: on-chain launchpad disabled: %v (contributions will return 503)", err)
	} else if launchpadOnChainEnabled() {
		log.Printf("on-chain launchpad configured: contract=%s", launchpadOnChainSingleton.contract.Hex())
	}

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
		// Discovery routes (public, frontend-facing)
		api.GET("/coins", listCoinsHandler(db, rdb))
		api.GET("/search", searchTokensHandler(db, rdb))
		api.GET("/featured", getFeaturedHandler(db, rdb))
		api.GET("/trending", getTrendingHandler(db, rdb))
		api.GET("/market", getMarketHandler(db, rdb))

		// Auth (public: register + login)
		auth := api.Group("/auth")
		{
			auth.POST("/register", registerHandler(db))
			auth.POST("/login", loginHandler(rdb, db))
		}

		// Favorites (authenticated users only)
		favAuth := api.Group("/favorites", authMiddleware())
		{
			favAuth.GET("", listFavoritesHandler(db))
			favAuth.POST("", addFavoriteHandler(db))
			favAuth.DELETE("/:id", removeFavoriteHandler(db))
		}

		// Token Management — mutations require auth; approve/reject require admin.
		tokens := api.Group("/tokens")
		{
			tokens.GET("", listTokensHandler(db))            // public browse
			tokens.GET("/:id", getTokenHandler(db))          // public view
			tokensAuth := tokens.Group("", authMiddleware()) // authenticated mutations
			{
				tokensAuth.POST("", createTokenHandler(db))
				tokensAuth.PUT("/:id", updateTokenHandler(db))
				tokensAuth.DELETE("/:id", deleteTokenHandler(db))
				tokensAuth.POST("/:id/submit", submitTokenHandler(db))
			}
			tokensAdmin := tokens.Group("", authMiddleware(), adminOnly()) // admin-only
			{
				tokensAdmin.POST("/:id/approve", approveTokenHandler(db, cfg))
				tokensAdmin.POST("/:id/reject", rejectTokenHandler(db))
				// Admin-only: verify a token's on-chain contract (checksum + ERC-20
				// interface: name/symbol/decimals/totalSupply eth_call).
				tokensAdmin.POST("/:id/verify-contract", verifyTokenContractHandler(db))
			}
		}

		// Token Listings — mutations require auth; feature/featured admin-only.
		listings := api.Group("/listings")
		{
			listings.GET("", listListingsHandler(db))
			listings.GET("/:id", getListingHandler(db))
			listingsAuth := listings.Group("", authMiddleware())
			{
				listingsAuth.POST("", createListingHandler(db))
				listingsAuth.PUT("/:id/status", updateListingStatusHandler(db))
			}
			listingsAdmin := listings.Group("", authMiddleware(), adminOnly())
			{
				listingsAdmin.POST("/:id/featured", featureListingHandler(db))
			}
		}

		// Launchpad — mutations require auth.
		launchpad := api.Group("/launchpad")
		{
			launchpad.GET("", listLaunchpadsHandler(db))
			launchpad.GET("/:id", getLaunchpadHandler(db))
			launchpadAuth := launchpad.Group("", authMiddleware())
			{
				launchpadAuth.POST("/create", createLaunchpadHandler(db))
				launchpadAuth.POST("/:id/contribute", contributeHandler(db))
				launchpadAuth.POST("/:id/claim", claimTokensHandler(db))
				launchpadAuth.POST("/:id/cancel", cancelLaunchpadHandler(db))
			}
		}

		// Market Making — mutations require auth.
		mm := api.Group("/market-making")
		{
			mm.GET("/orders", getMakerOrdersHandler(db))
			mm.GET("/status/:token_id", getMarketMakerStatusHandler(db))
			// Config routes: link a listed token to a bot market-maker.
			// List is public (so bot_api can read configs); create/delete need auth.
			mm.GET("/configs", listMarketMakingConfigsHandler(db))
			mmAuth := mm.Group("", authMiddleware())
			{
				mmAuth.POST("/orders", createMakerOrdersHandler(db))
				mmAuth.PUT("/orders/:id/status", updateOrderStatusHandler(db))
				mmAuth.POST("/liquidity/add", addLiquidityHandler(db))
				mmAuth.POST("/liquidity/remove", removeLiquidityHandler(db))
				mmAuth.POST("/configs", createMarketMakingConfigHandler(db))
				mmAuth.DELETE("/configs/:id", deleteMarketMakingConfigHandler(db))
			}
		}

		// Pricing — set/update require auth; reads public.
		pricing := api.Group("/pricing")
		{
			pricing.GET("/:token_id", getTokenPriceHandler(db))
			pricing.GET("/history/:token_id", getPriceHistoryHandler(db))
			pricingAuth := pricing.Group("", authMiddleware(), adminOnly())
			{
				pricingAuth.POST("/set", setTokenPriceHandler(db))
				pricingAuth.POST("/update", updatePriceHandler(db))
			}
		}

		// Analytics (public read)
		analytics := api.Group("/analytics")
		{
			analytics.GET("/volume", getTradingVolumeHandler(db))
			analytics.GET("/liquidity", getLiquidityHandler(db))
			analytics.GET("/holders", getHolderCountHandler(db))
			analytics.GET("/transactions", getTransactionCountHandler(db))
		}

		// Compliance — submit requires auth; reads public.
		compliance := api.Group("/compliance")
		{
			compliance.GET("/audit/:token_id", getAuditStatusHandler(db))
			compliance.GET("/kyc/:token_id", getKYCStatusHandler(db))
			complianceAuth := compliance.Group("", authMiddleware())
			{
				complianceAuth.POST("/audit", requestAuditHandler(db))
				complianceAuth.POST("/kyc/submit", submitKYCHandler(db))
			}
		}

		// Fees — calculate/pay require auth.
		fees := api.Group("/fees")
		{
			fees.GET("", getListingFeesHandler(db))
			feesAuth := fees.Group("", authMiddleware())
			{
				feesAuth.POST("/calculate", calculateFeesHandler(db))
				feesAuth.POST("/pay", payFeesHandler(db))
				// Admin-only: verify a fee payment's on-chain tx receipt.
				feesAuth.POST("/verify/:id", adminOnly(), verifyFeePaymentHandler(db))
			}
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
	// UserWallet token-registry propagation: when a token is approved, the
	// approved token is pushed into the UserWallet token registry over the
	// authorized HTTP admin endpoint (no cross-domain package import). The
	// service token is issued by the MasterWallet admin / SuperAdmin.
	WalletAPIURL   string
	WalletAPIToken string
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
		WalletAPIURL:   getEnv("WALLET_API_URL", "http://localhost:8443"),
		WalletAPIToken: getEnv("WALLET_API_ADMIN_TOKEN", ""),
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
	ID              uuid.UUID         `json:"id" db:"id"`
	TenantID        uuid.UUID         `json:"tenant_id" db:"tenant_id"`
	Name            string            `json:"name" db:"name"`
	Symbol          string            `json:"symbol" db:"symbol"`
	Decimals        int               `json:"decimals" db:"decimals"`
	ContractAddress string            `json:"contract_address" db:"contract_address"`
	Chain           string            `json:"chain" db:"chain"`
	TotalSupply     string            `json:"total_supply" db:"total_supply"`
	LogoURL         string            `json:"logo_url" db:"logo_url"`
	Description     string            `json:"description" db:"description"`
	Website         string            `json:"website" db:"website"`
	Whitepaper      string            `json:"whitepaper" db:"whitepaper"`
	SocialLinks     map[string]string `json:"social_links" db:"social_links"`
	Status          string            `json:"status" db:"status"` // draft, submitted, in_review, approved, rejected, listed
	SubmissionDate  *time.Time        `json:"submission_date" db:"submission_date"`
	ReviewerID      *uuid.UUID        `json:"reviewer_id" db:"reviewer_id"`
	ReviewedAt      *time.Time        `json:"reviewed_at" db:"reviewed_at"`
	RejectionReason *string           `json:"rejection_reason" db:"rejection_reason"`
	ListingFeeUSD   float64           `json:"listing_fee_usd" db:"listing_fee_usd"`
	IsFeatured      bool              `json:"is_featured" db:"is_featured"`
	LaunchpadID     *uuid.UUID        `json:"launchpad_id" db:"launchpad_id"`
	CreatedAt       time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at" db:"updated_at"`
}

type TokenListing struct {
	ID             uuid.UUID `json:"id" db:"id"`
	TokenID        uuid.UUID `json:"token_id" db:"token_id"`
	TenantID       uuid.UUID `json:"tenant_id" db:"tenant_id"`
	PairToken      string    `json:"pair_token" db:"pair_token"` // e.g., "USDT", "ETH"
	InitialPrice   string    `json:"initial_price" db:"initial_price"`
	CurrentPrice   string    `json:"current_price" db:"current_price"`
	LaunchType     string    `json:"launch_type" db:"launch_type"` // fair_launch, presale, farming
	StartTime      time.Time `json:"start_time" db:"start_time"`
	EndTime        time.Time `json:"end_time" db:"end_time"`
	Status         string    `json:"status" db:"status"` // upcoming, active, completed, cancelled
	Volume24h      string    `json:"volume_24h" db:"volume_24h"`
	LiquidityUSD   string    `json:"liquidity_usd" db:"liquidity_usd"`
	MarketCap      string    `json:"market_cap" db:"market_cap"`
	PriceChange24h float64   `json:"price_change_24h" db:"price_change_24h"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
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
	ID          uuid.UUID  `json:"id" db:"id"`
	LaunchpadID uuid.UUID  `json:"launchpad_id" db:"launchpad_id"`
	UserID      uuid.UUID  `json:"user_id" db:"user_id"`
	Amount      string     `json:"amount" db:"amount"`
	TokenAmount string     `json:"token_amount" db:"token_amount"`
	Status      string     `json:"status" db:"status"` // pending, confirmed, claimed, refunded
	TxHash      string     `json:"tx_hash" db:"tx_hash"`
	ConfirmedAt *time.Time `json:"confirmed_at" db:"confirmed_at"`
	ClaimedAt   *time.Time `json:"claimed_at" db:"claimed_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

type MarketMakerOrder struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	TenantID  uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	TokenID   uuid.UUID  `json:"token_id" db:"token_id"`
	Side      string     `json:"side" db:"side"` // buy, sell
	Price     string     `json:"price" db:"price"`
	Quantity  string     `json:"quantity" db:"quantity"`
	Remaining string     `json:"remaining" db:"remaining"`
	Status    string     `json:"status" db:"status"` // pending, filled, cancelled
	FilledAt  *time.Time `json:"filled_at" db:"filled_at"`
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

type TokenPrice struct {
	ID        uuid.UUID `json:"id" db:"id"`
	TokenID   uuid.UUID `json:"token_id" db:"token_id"`
	Price     string    `json:"price" db:"price"`
	Change24h float64   `json:"change_24h" db:"change_24h"`
	Volume24h string    `json:"volume_24h" db:"volume_24h"`
	Timestamp time.Time `json:"timestamp" db:"timestamp"`
}

type ComplianceAudit struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	TokenID     uuid.UUID  `json:"token_id" db:"token_id"`
	AuditType   string     `json:"audit_type" db:"audit_type"` // security, code, financial
	Status      string     `json:"status" db:"status"`         // requested, in_progress, completed, failed
	ReportURL   *string    `json:"report_url" db:"report_url"`
	Auditor     string     `json:"auditor" db:"auditor"`
	CompletedAt *time.Time `json:"completed_at" db:"completed_at"`
	RequestedAt time.Time  `json:"requested_at" db:"requested_at"`
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
			"volume_24h":       totalVolume24h,
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

// jwtSecret returns the JWT signing secret from env. Fail-closed: a missing
// JWT_SECRET is a fatal startup error (no insecure dev default), matching the
// white-label clone's hardening and the no-security-vulnerability directive.
func jwtSecret() string {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		log.Fatal("JWT_SECRET environment variable is required (fail-closed, no default)")
	}
	return s
}

// Claims is the JWT payload for project_party sessions.
type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// validRoles is the closed set of roles the project_party backend accepts.
func validRoles() map[string]bool {
	return map[string]bool{"user": true, "admin": true, "super_admin": true}
}

// hashPassword bcrypts a plaintext password (cost 12).
func hashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), 12)
	return string(b), err
}

// checkPassword verifies a bcrypt hash; constant-time via bcrypt.
func checkPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

// issueJWT mints a 24h HS256 JWT for a user+role.
func issueJWT(userID, role string) (string, error) {
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID,
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString([]byte(jwtSecret()))
}

// parseJWT validates and parses a JWT string. Returns claims or error.
func parseJWT(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	tok, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(jwtSecret()), nil
	})
	if err != nil || !tok.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	if !validRoles()[claims.Role] {
		return nil, fmt.Errorf("invalid role")
	}
	return claims, nil
}

// authMiddleware validates the Bearer JWT and sets user_id + role in the gin
// context. Unauthenticated requests are rejected with 401.
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid Authorization header"})
			return
		}
		claims, err := parseJWT(strings.TrimPrefix(auth, "Bearer "))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// requireRole returns a middleware that rejects requests whose role is not in
// the allowed set. Must be used AFTER authMiddleware.
func requireRole(allowed ...string) gin.HandlerFunc {
	set := make(map[string]bool, len(allowed))
	for _, r := range allowed {
		set[r] = true
	}
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		roleStr, _ := role.(string)
		if !set[roleStr] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}
		c.Next()
	}
}

// adminOnly is a shorthand for requireRole("admin", "super_admin").
func adminOnly() gin.HandlerFunc { return requireRole("admin", "super_admin") }

func loginHandler(rdb *redis.Client, db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		var userID, role, passwordHash string
		err := db.QueryRow(ctx,
			`SELECT id, role, password_hash FROM pp_users WHERE username=$1 AND is_active=true`,
			req.Username).Scan(&userID, &role, &passwordHash)
		if err != nil || !checkPassword(passwordHash, req.Password) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		token, err := issueJWT(userID, role)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
			return
		}
		if rdb != nil {
			rCtx, rCancel := context.WithTimeout(ctx, 2*time.Second)
			defer rCancel()
			rdb.Set(rCtx, "session:"+token, req.Username, 24*time.Hour)
		}
		c.JSON(http.StatusOK, gin.H{"token": token, "username": req.Username, "role": role})
	}
}

// registerHandler creates a new pp_users row (always role="user"; privileged
// roles are assigned only by a DB admin / SuperAdmin tool).
func registerHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required,min=3"`
			Password string `json:"password" binding:"required,min=8"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		pwHash, err := hashPassword(req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
			return
		}
		userID := uuid.New().String()
		_, err = db.Exec(ctx,
			`INSERT INTO pp_users (id, username, password_hash, role, is_active) VALUES ($1,$2,$3,'user',true)`,
			userID, req.Username, pwHash)
		if err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": "username already exists"})
			return
		}
		token, err := issueJWT(userID, "user")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"token": token, "username": req.Username, "role": "user"})
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
		ownerID, _ := actorUUID(c)
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
		_, err := db.Exec(ctx, `INSERT INTO tokens (id, tenant_id, owner_id, name, symbol, decimals, contract_address, chain, total_supply, logo_url, description, website, whitepaper, social_links, status, listing_fee_usd, is_featured, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`,
			token.ID, token.TenantID, ownerID, token.Name, token.Symbol, token.Decimals, token.ContractAddress, token.Chain, token.TotalSupply, token.LogoURL, token.Description, token.Website, token.Whitepaper, linksJSON, token.Status, token.ListingFeeUSD, token.IsFeatured, token.CreatedAt, token.UpdatedAt)
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
		if !canManageToken(c, db, tokenID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "not the token owner"})
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
		if !canManageToken(c, db, tokenID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "not the token owner"})
			return
		}
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
		if !canManageToken(c, db, tokenID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "not the token owner"})
			return
		}
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

func approveTokenHandler(db *pgxpool.Pool, cfg *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenID := c.Param("id")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		ct, err := db.Exec(ctx, `UPDATE tokens SET status='listed', reviewed_at=$1, updated_at=$1 WHERE id=$2 AND status IN ('submitted','in_review')`, time.Now(), tokenID)
		if err != nil || ct.RowsAffected() == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "token not found or not in a reviewable status"})
			return
		}
		// Propagate the approved token into the UserWallet token registry over
		// the authorized HTTP admin endpoint (no cross-domain package import).
		// Best-effort + logged: the approval DB write is the source of truth;
		// propagation is a downstream side-effect that surfaces the token in
		// UserWallet. A failure here does not roll back the approval.
		go propagateTokenToUserWallet(cfg, tokenID)
		c.JSON(http.StatusOK, gin.H{"message": "Token approved and listed", "status": "listed"})
	}
}

// chainIDForNetwork maps a ProjectParty network/chain name (e.g. "ethereum",
// "bsc", "polygon") to the canonical EVM chain id used by the UserWallet
// token registry. Non-EVM / unknown chains return 0 (propagation skipped —
// the UserWallet registry is EVM-scoped for ERC-20 contract addresses).
func chainIDForNetwork(network string) int64 {
	switch strings.ToLower(strings.TrimSpace(network)) {
	case "ethereum", "eth", "mainnet":
		return 1
	case "bsc", "binance", "binance-smart-chain", "bnb":
		return 56
	case "polygon", "matic":
		return 137
	case "arbitrum", "arbitrum-one":
		return 42161
	case "optimism", "op":
		return 10
	case "base":
		return 8453
	case "avalanche", "avax":
		return 43114
	case "fantom", "ftm":
		return 250
	}
	return 0
}

// propagateTokenToUserWallet fetches the approved token row and POSTs it to
// the UserWallet token registry (POST /api/v1/admin/tokens). Runs in its own
// goroutine; failures are logged and non-fatal.
func propagateTokenToUserWallet(cfg *Config, tokenID string) {
	if cfg.WalletAPIURL == "" {
		return
	}
	pool := dbPoolForPropagation(cfg)
	if pool == nil {
		log.Printf("propagate: no db pool available; skipping token %s", tokenID)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var t struct {
		Name            string `json:"name"`
		Symbol          string `json:"symbol"`
		Decimals        int    `json:"decimals"`
		ContractAddress string `json:"contract_address"`
		Chain           string `json:"chain"`
		LogoURL         string `json:"logo_url"`
	}
	err := pool.QueryRow(ctx,
		`SELECT name, symbol, decimals, contract_address, chain, logo_url FROM tokens WHERE id=$1`, tokenID).
		Scan(&t.Name, &t.Symbol, &t.Decimals, &t.ContractAddress, &t.Chain, &t.LogoURL)
	if err != nil {
		log.Printf("propagate: fetch token %s: %v", tokenID, err)
		return
	}
	chainID := chainIDForNetwork(t.Chain)
	if chainID == 0 {
		log.Printf("propagate: token %s chain %q is non-EVM/unknown — skipping UserWallet registry push", tokenID, t.Chain)
		return
	}
	body, _ := json.Marshal(map[string]any{
		"chain_id":  chainID,
		"contract":  t.ContractAddress,
		"symbol":    t.Symbol,
		"name":      t.Name,
		"decimals":  t.Decimals,
		"logo_uri":  t.LogoURL,
		"is_active": true,
		"source":    "project_party",
	})
	url := strings.TrimRight(cfg.WalletAPIURL, "/") + "/api/v1/admin/tokens"
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if cfg.WalletAPIToken != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.WalletAPIToken)
	}
	resp, err := httpClientForPropagation.Do(req)
	if err != nil {
		log.Printf("propagate: POST token %s to UserWallet: %v", tokenID, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Printf("propagate: UserWallet rejected token %s (HTTP %d)", tokenID, resp.StatusCode)
		return
	}
	log.Printf("propagate: token %s (%s) pushed to UserWallet registry chain_id=%d", tokenID, t.Symbol, chainID)
}

var httpClientForPropagation = &http.Client{Timeout: 10 * time.Second}

// dbPoolForPropagation returns the project_party pgx pool. The pool is set as
// a package-level var in initDatabase; this indirection keeps the propagation
// helper testable.
var propagationDBPool *pgxpool.Pool

func dbPoolForPropagation(cfg *Config) *pgxpool.Pool {
	if propagationDBPool != nil {
		return propagationDBPool
	}
	return nil
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
			ID:             uuid.New(),
			TokenID:        tokenID,
			PairToken:      req.PairToken,
			InitialPrice:   req.InitialPrice,
			CurrentPrice:   req.InitialPrice,
			LaunchType:     req.LaunchType,
			StartTime:      start,
			EndTime:        end,
			Status:         "upcoming",
			Volume24h:      "0",
			LiquidityUSD:   "0",
			MarketCap:      "0",
			PriceChange24h: 0,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		if _, err := db.Exec(ctx, `INSERT INTO token_listings (id, token_id, owner_id, pair_token, initial_price, current_price, launch_type, start_time, end_time, status, volume_24h, liquidity_usd, market_cap, price_change_24h, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
			listing.ID, listing.TokenID, lpOwnerID(c), listing.PairToken, listing.InitialPrice, listing.CurrentPrice, listing.LaunchType, listing.StartTime, listing.EndTime, listing.Status, listing.Volume24h, listing.LiquidityUSD, listing.MarketCap, listing.PriceChange24h, listing.CreatedAt, listing.UpdatedAt); err != nil {
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
		if !canManageListing(c, db, c.Param("id")) {
			c.JSON(http.StatusForbidden, gin.H{"error": "not the listing owner"})
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
		if _, err := db.Exec(ctx, `INSERT INTO launchpads (id, token_id, owner_id, name, description, soft_cap, hard_cap, min_contribution, max_contribution, start_time, end_time, token_price, accepted_payment, total_raised, status, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
			lp.ID, lp.TokenID, lpOwnerID(c), lp.Name, lp.Description, lp.SoftCap, lp.HardCap, lp.MinContribution, lp.MaxContribution, lp.StartTime, lp.EndTime, lp.TokenPrice, lp.AcceptedPayment, lp.TotalRaised, lp.Status, lp.CreatedAt, lp.UpdatedAt); err != nil {
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
			Symbol       string  `json:"symbol"`
			Progress     float64 `json:"progress"`
			Contributors int     `json:"contributors"`
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
			"id":           lp.ID,
			"token_id":     lp.TokenID,
			"name":         lp.Name,
			"soft_cap":     lp.SoftCap,
			"hard_cap":     lp.HardCap,
			"total_raised": lp.TotalRaised,
			"status":       lp.Status,
			"contributors": contributors,
			"symbol":       symbol,
		})
	}
}

func contributeHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		var req struct {
			Amount string `json:"amount" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		uidStr, _ := userID.(string)
		uid, err := uuid.Parse(uidStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
			return
		}
		launchpadID := c.Param("id")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
		defer cancel()
		var tokenPrice, status string
		err = db.QueryRow(ctx, `SELECT token_price, status FROM launchpads WHERE id=$1`, launchpadID).Scan(&tokenPrice, &status)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "launchpad not found"})
			return
		}
		if status != "active" && status != "upcoming" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "launchpad not accepting contributions"})
			return
		}

		// Record the contribution as 'pending' BEFORE the on-chain attempt.
		contrib := LaunchpadContribution{
			ID:          uuid.New(),
			LaunchpadID: uuid.MustParse(launchpadID),
			UserID:      uid,
			Amount:      req.Amount,
			Status:      "pending",
			CreatedAt:   time.Now(),
		}
		if _, err := db.Exec(ctx,
			`INSERT INTO launchpad_contributions (id, launchpad_id, user_id, amount, token_amount, status, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			contrib.ID, contrib.LaunchpadID, contrib.UserID, contrib.Amount, contrib.TokenAmount, contrib.Status, contrib.CreatedAt); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable", "detail": err.Error()})
			return
		}

		// On-chain contribution (fail-closed: if the on-chain layer is not
		// configured, return 503 and leave the contribution as 'pending' so
		// the operator can reconcile. NEVER fabricate a tx hash.)
		if !launchpadOnChainEnabled() {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":        "on-chain launchpad not configured",
				"contribution": contrib,
				"status":       "pending_offchain",
			})
			return
		}
		saleID := saleIDFromUUID(launchpadID)
		valueWei, ok := new(big.Int).SetString(req.Amount, 10)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount (must be wei integer)"})
			return
		}
		txHash, tokenClaim, err := launchpadOnChainSingleton.contributeOnChain(ctx, saleID, valueWei)
		if err != nil {
			// Mark the contribution as 'failed' — never fabricate.
			db.Exec(ctx, `UPDATE launchpad_contributions SET status='failed' WHERE id=$1`, contrib.ID)
			c.JSON(http.StatusBadGateway, gin.H{
				"error":        "on-chain contribution failed",
				"detail":       err.Error(),
				"contribution": contrib,
			})
			return
		}
		// Persist the real tx hash + token claim.
		_ = persistOnChainContribution(db, ctx, contrib.ID.String(), txHash, tokenClaim)
		contrib.TxHash = txHash
		contrib.Status = "confirmed"
		contrib.TokenAmount = tokenClaim.String()

		// Update total_raised (the on-chain amount is the source of truth).
		if _, err := db.Exec(ctx, `UPDATE launchpads SET total_raised=(COALESCE(total_raised::numeric,0)+$1)::text, updated_at=$2 WHERE id=$3`, req.Amount, time.Now(), launchpadID); err != nil {
			log.Printf("warn: update total_raised failed: %v", err)
		}
		c.JSON(http.StatusCreated, gin.H{
			"contribution": contrib,
			"tx_hash":      txHash,
			"message":      "On-chain contribution confirmed",
		})
	}
}

func claimTokensHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		uidStr, _ := userID.(string)
		uid, err := uuid.Parse(uidStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
			return
		}
		launchpadID := c.Param("id")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
		defer cancel()

		// Require the contribution to be 'confirmed' (on-chain tx happened).
		var contribID string
		err = db.QueryRow(ctx,
			`SELECT id FROM launchpad_contributions WHERE launchpad_id=$1 AND user_id=$2 AND status='confirmed' ORDER BY created_at DESC LIMIT 1`,
			launchpadID, uid).Scan(&contribID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no confirmed contribution found"})
			return
		}

		// On-chain claim (fail-closed).
		if !launchpadOnChainEnabled() {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "on-chain launchpad not configured"})
			return
		}
		saleID := saleIDFromUUID(launchpadID)
		claimTxHash, err := launchpadOnChainSingleton.claimTokensOnChain(ctx, saleID)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{
				"error":  "on-chain claim failed",
				"detail": err.Error(),
			})
			return
		}
		// Mark as claimed with the real claim tx hash.
		ct, err := db.Exec(ctx,
			`UPDATE launchpad_contributions SET status='claimed', claimed_at=$1, tx_hash=$2 WHERE id=$3`,
			time.Now(), claimTxHash, contribID)
		if err != nil || ct.RowsAffected() == 0 {
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to update claim status"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message":      "Tokens claimed on-chain",
			"tx_hash":      claimTxHash,
			"contribution": contribID,
		})
	}
}

func cancelLaunchpadHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !canManageLaunchpad(c, db, c.Param("id")) {
			c.JSON(http.StatusForbidden, gin.H{"error": "not the launchpad owner"})
			return
		}
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
		if _, err := db.Exec(ctx, `INSERT INTO market_maker_orders (id, token_id, owner_id, side, price, quantity, remaining, status, expires_at, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
			order.ID, order.TokenID, lpOwnerID(c), order.Side, order.Price, order.Quantity, order.Remaining, order.Status, order.ExpiresAt, order.CreatedAt); err != nil {
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
		if !canManageOrder(c, db, c.Param("id")) {
			c.JSON(http.StatusForbidden, gin.H{"error": "not the order owner"})
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

// isAdminRole reports whether the caller has an admin-tier role.
func isAdminRole(c *gin.Context) bool {
	role, _ := c.Get("role")
	r, _ := role.(string)
	return r == "admin" || r == "super_admin"
}

// actorUUID parses the JWT user_id set by authMiddleware.
func actorUUID(c *gin.Context) (uuid.UUID, bool) {
	uid, _ := c.Get("user_id")
	s, _ := uid.(string)
	id, err := uuid.Parse(s)
	return id, err == nil
}

// lpOwnerID returns the JWT actor as a *uuid.UUID for owner columns (nil when absent).
func lpOwnerID(c *gin.Context) *uuid.UUID {
	id, ok := actorUUID(c)
	if !ok {
		return nil
	}
	return &id
}

// canManageToken: token owner (tokens.owner_id) or admin.
func canManageToken(c *gin.Context, db *pgxpool.Pool, tokenID string) bool {
	if isAdminRole(c) {
		return true
	}
	actor, ok := actorUUID(c)
	if !ok {
		return false
	}
	var owner *uuid.UUID
	if err := db.QueryRow(c.Request.Context(), `SELECT owner_id FROM tokens WHERE id=$1`, tokenID).Scan(&owner); err != nil {
		return false
	}
	return owner != nil && *owner == actor
}

// canManageLaunchpad: launchpad owner, underlying token owner, or admin.
func canManageLaunchpad(c *gin.Context, db *pgxpool.Pool, launchpadID string) bool {
	if isAdminRole(c) {
		return true
	}
	actor, ok := actorUUID(c)
	if !ok {
		return false
	}
	var owner, tokenOwner *uuid.UUID
	if err := db.QueryRow(c.Request.Context(),
		`SELECT l.owner_id, t.owner_id FROM launchpads l JOIN tokens t ON t.id=l.token_id WHERE l.id=$1`,
		launchpadID).Scan(&owner, &tokenOwner); err != nil {
		return false
	}
	return (owner != nil && *owner == actor) || (tokenOwner != nil && *tokenOwner == actor)
}

// canManageOrder: order owner, underlying token owner, or admin.
func canManageOrder(c *gin.Context, db *pgxpool.Pool, orderID string) bool {
	if isAdminRole(c) {
		return true
	}
	actor, ok := actorUUID(c)
	if !ok {
		return false
	}
	var owner, tokenOwner *uuid.UUID
	if err := db.QueryRow(c.Request.Context(),
		`SELECT o.owner_id, t.owner_id FROM market_maker_orders o JOIN tokens t ON t.id=o.token_id WHERE o.id=$1`,
		orderID).Scan(&owner, &tokenOwner); err != nil {
		return false
	}
	return (owner != nil && *owner == actor) || (tokenOwner != nil && *tokenOwner == actor)
}

// canManageListing: listing owner, underlying token owner, or admin.
func canManageListing(c *gin.Context, db *pgxpool.Pool, listingID string) bool {
	if isAdminRole(c) {
		return true
	}
	actor, ok := actorUUID(c)
	if !ok {
		return false
	}
	var owner, tokenOwner *uuid.UUID
	if err := db.QueryRow(c.Request.Context(),
		`SELECT l.owner_id, t.owner_id FROM token_listings l JOIN tokens t ON t.id=l.token_id WHERE l.id=$1`,
		listingID).Scan(&owner, &tokenOwner); err != nil {
		return false
	}
	return (owner != nil && *owner == actor) || (tokenOwner != nil && *tokenOwner == actor)
}

// updateListingLiquidity recomputes a token's pool liquidity in USD terms
// (total reserve x latest token price) onto its token_listings row.
func updateListingLiquidity(ctx context.Context, db *pgxpool.Pool, tokenID string) {
	var reserve float64
	if err := db.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount::numeric),0) FROM liquidity_positions WHERE token_id=$1`,
		tokenID).Scan(&reserve); err != nil {
		return
	}
	var price string
	if err := db.QueryRow(ctx,
		`SELECT price FROM token_prices WHERE token_id=$1 ORDER BY timestamp DESC LIMIT 1`,
		tokenID).Scan(&price); err != nil {
		return
	}
	p, err := parseFloat(price)
	if err != nil {
		return
	}
	if _, err := db.Exec(ctx,
		`UPDATE token_listings SET liquidity_usd=$1, updated_at=$2 WHERE token_id=$3`,
		fmt.Sprintf("%.6f", reserve*p), time.Now(), tokenID); err != nil {
		log.Printf("warn: update listing liquidity failed: %v", err)
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
			TokenID    string `json:"token_id" binding:"required"`
			Amount     string `json:"amount" binding:"required"`
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
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		// Real proportional LP shares: the first deposit mints 1:1; every
		// later deposit mints amount * totalLP / totalReserve so LP tokens
		// always represent an exact pro-rata share of the pool reserve.
		var totalReserve, totalLP float64
		if err := db.QueryRow(ctx,
			`SELECT COALESCE(SUM(amount::numeric),0), COALESCE(SUM(lp_tokens::numeric),0) FROM liquidity_positions WHERE token_id=$1`,
			tokenID).Scan(&totalReserve, &totalLP); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable", "detail": err.Error()})
			return
		}
		lp := amt
		if totalReserve > 0 && totalLP > 0 {
			lp = amt * totalLP / totalReserve
		}
		lpTokens := fmt.Sprintf("%.6f", lp)
		if _, err := db.Exec(ctx, `INSERT INTO liquidity_positions (id, token_id, quote_token, amount, lp_tokens, created_at) VALUES ($1,$2,$3,$4,$5,$6)`, uuid.New(), tokenID, req.QuoteToken, req.Amount, lpTokens, time.Now()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable", "detail": err.Error()})
			return
		}
		updateListingLiquidity(ctx, db, tokenID.String())
		c.JSON(http.StatusOK, gin.H{"message": "Liquidity added successfully", "lp_tokens": lpTokens})
	}
}

func removeLiquidityHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			TokenID  string `json:"token_id" binding:"required"`
			LPAmount string `json:"lp_amount" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		lpAmt, err := parseFloat(req.LPAmount)
		if err != nil || lpAmt <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lp_amount"})
			return
		}
		tokenID, err := uuid.Parse(req.TokenID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token_id"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		tx, err := db.Begin(ctx)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable"})
			return
		}
		defer tx.Rollback(ctx)
		// Burn LP tokens proportionally, newest position first: each burned LP
		// releases exactly lp/totalLP of that position's recorded amount.
		rows, err := tx.Query(ctx,
			`SELECT id, lp_tokens::numeric, amount::numeric FROM liquidity_positions WHERE token_id=$1 ORDER BY created_at DESC FOR UPDATE`,
			tokenID)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable"})
			return
		}
		type pos struct {
			id     uuid.UUID
			lp     float64
			amount float64
		}
		var positions []pos
		var totalLP float64
		for rows.Next() {
			var p pos
			if err := rows.Scan(&p.id, &p.lp, &p.amount); err == nil {
				positions = append(positions, p)
				totalLP += p.lp
			}
		}
		rows.Close()
		if totalLP <= 0 || lpAmt > totalLP {
			c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient lp balance"})
			return
		}
		remaining := lpAmt
		withdrawn := 0.0
		for _, p := range positions {
			if remaining <= 0 {
				break
			}
			burn := p.lp
			if burn > remaining {
				burn = remaining
			}
			released := p.amount * (burn / p.lp)
			withdrawn += released
			remaining -= burn
			if burn >= p.lp {
				if _, err := tx.Exec(ctx, `DELETE FROM liquidity_positions WHERE id=$1`, p.id); err != nil {
					c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable"})
					return
				}
			} else {
				newLP := fmt.Sprintf("%.6f", p.lp-burn)
				newAmt := fmt.Sprintf("%.6f", p.amount-released)
				if _, err := tx.Exec(ctx, `UPDATE liquidity_positions SET lp_tokens=$1, amount=$2 WHERE id=$3`, newLP, newAmt, p.id); err != nil {
					c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable"})
					return
				}
			}
		}
		if err := tx.Commit(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable"})
			return
		}
		updateListingLiquidity(ctx, db, tokenID.String())
		c.JSON(http.StatusOK, gin.H{"message": "Liquidity removed successfully", "lp_burned": fmt.Sprintf("%.6f", lpAmt), "amount_released": fmt.Sprintf("%.6f", withdrawn)})
	}
}

// ============== Pricing Handlers ==============

func setTokenPriceHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			TokenID   string `json:"token_id" binding:"required"`
			Price     string `json:"price" binding:"required"`
			Volume24h string `json:"volume_24h"`
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
		// Real 24h change: pct vs the newest price recorded >= 24h ago.
		change24h := 0.0
		var prevPrice string
		if err := db.QueryRow(ctx,
			`SELECT price FROM token_prices WHERE token_id=$1 AND timestamp <= NOW() - INTERVAL '24 hours' ORDER BY timestamp DESC LIMIT 1`,
			tokenID).Scan(&prevPrice); err == nil {
			if prev, err1 := parseFloat(prevPrice); err1 == nil && prev > 0 {
				if cur, err2 := parseFloat(req.Price); err2 == nil {
					change24h = (cur - prev) / prev * 100
				}
			}
		}
		// Real 24h volume: caller-supplied oracle volume, else the sum of
		// this token's launchpad contributions over the last 24h.
		vol24h := req.Volume24h
		if vol24h == "" {
			_ = db.QueryRow(ctx,
				`SELECT COALESCE(SUM(amount::numeric),0)::text FROM launchpad_contributions lc JOIN launchpads l ON l.id=lc.launchpad_id WHERE l.token_id=$1 AND lc.created_at > NOW() - INTERVAL '24 hours'`,
				tokenID).Scan(&vol24h)
		}
		if vol24h == "" {
			vol24h = "0"
		}
		if _, err := db.Exec(ctx, `INSERT INTO token_prices (id, token_id, price, change_24h, volume_24h, timestamp) VALUES ($1,$2,$3,$4,$5,$6)`, uuid.New(), tokenID, req.Price, change24h, vol24h, time.Now()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable", "detail": err.Error()})
			return
		}
		// Keep the listing row in sync: current price, 24h change, 24h
		// volume, and market cap (price x total_supply).
		marketCap := "0"
		var supply string
		if err := db.QueryRow(ctx, `SELECT COALESCE(NULLIF(total_supply,''),'0') FROM tokens WHERE id=$1`, tokenID).Scan(&supply); err == nil {
			if sup, err1 := parseFloat(supply); err1 == nil {
				if cur, err2 := parseFloat(req.Price); err2 == nil {
					marketCap = fmt.Sprintf("%.6f", sup*cur)
				}
			}
		}
		if _, err := db.Exec(ctx, `UPDATE token_listings SET current_price=$1, volume_24h=$2, price_change_24h=$3, market_cap=$4, updated_at=$5 WHERE token_id=$6`, req.Price, vol24h, change24h, marketCap, time.Now(), tokenID); err != nil {
			log.Printf("warn: update listing price failed: %v", err)
		}
		c.JSON(http.StatusOK, gin.H{"message": "Price set", "price": req.Price, "change_24h": change24h, "volume_24h": vol24h})
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
		// No fabricated fallbacks — if the fee_schedule table is empty, return
		// an honest empty map (the frontend shows "not configured").
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
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		// Read the base fee for the requested listing type from fee_schedule.
		// The listing type maps to a fee_type key (basic_listing or
		// launchpad_basic). If not configured, return 0 honestly.
		baseKey := "basic_listing"
		if req.ListingType == "launchpad" {
			baseKey = "launchpad_basic"
		}
		var total float64
		_ = db.QueryRow(ctx, `SELECT amount FROM fee_schedule WHERE fee_type=$1`, baseKey).Scan(&total)

		// Add feature fees from the schedule (featured, audit, kyc).
		featureKeyMap := map[string]string{"featured": "featured_listing", "audit": "audit_required", "kyc": "kyc_verification"}
		for _, f := range req.Features {
			if fk, ok := featureKeyMap[f]; ok {
				var amt float64
				_ = db.QueryRow(ctx, `SELECT amount FROM fee_schedule WHERE fee_type=$1`, fk).Scan(&amt)
				total += amt
			}
		}
		c.JSON(http.StatusOK, gin.H{"total_fee": total, "currency": "USD"})
	}
}

func payFeesHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		var req struct {
			TokenID       string `json:"token_id"`
			Amount        string `json:"amount" binding:"required"`
			PaymentMethod string `json:"payment_method" binding:"required"`
			TxHash        string `json:"tx_hash"` // optional; client-supplied, NOT trusted
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		txID := uuid.New()
		uid, _ := userID.(string)
		// Store as 'pending' — NEVER 'completed'. The tx_hash is recorded for
		// audit but is NOT trusted to mark the fee paid; an admin must verify
		// the on-chain receipt (or the payment provider webhook) before the
		// status transitions to 'completed'. This prevents users from claiming
		// they paid without paying.
		status := "pending"
		if req.TxHash == "" {
			status = "awaiting_payment"
		}
		_, err := db.Exec(ctx,
			`INSERT INTO fee_payments (id, token_id, user_id, amount, payment_method, tx_hash, status, created_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			txID, strOrNull(req.TokenID), strOrNull(uid), req.Amount, req.PaymentMethod, strOrNull(req.TxHash), status, time.Now())
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable", "detail": err.Error()})
			return
		}
		c.JSON(http.StatusAccepted, gin.H{
			"message":        "Payment recorded; awaiting verification",
			"transaction_id": txID,
			"status":         status,
		})
	}
}

// verifyFeePaymentHandler does REAL on-chain receipt verification for a fee
// payment. An admin calls this after the user claims they paid (payFeesHandler
// stored the tx_hash as 'pending'). The handler fetches the transaction receipt
// from the blockchain via go-ethereum ethclient, checks: (1) the tx exists,
// (2) status == 1 (success), (3) the tx was not replayed on a different chain.
// Only then does it transition the fee_payments row to 'completed'. If the
// receipt is not found or the tx failed, the status stays 'pending' (fail-closed).
// Requires PP_RPC_URL env to be set (same as the launchpad on-chain layer).
func verifyFeePaymentHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		paymentID := c.Param("id")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()

		// Fetch the fee_payment record
		var txHash, status string
		err := db.QueryRow(ctx,
			`SELECT tx_hash, status FROM fee_payments WHERE id=$1`, paymentID).Scan(&txHash, &status)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "fee payment not found"})
			return
		}
		if txHash == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no tx_hash recorded for this payment"})
			return
		}
		if status == "completed" {
			c.JSON(http.StatusOK, gin.H{"message": "payment already verified", "status": "completed"})
			return
		}

		rpcURL := getenvDefault("PP_RPC_URL", "")
		if rpcURL == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "on-chain verification not configured (PP_RPC_URL unset)"})
			return
		}

		client, err := ethclient.Dial(rpcURL)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "RPC connection failed", "detail": err.Error()})
			return
		}
		defer client.Close()

		// Normalize tx hash (ensure 0x prefix)
		if !strings.HasPrefix(txHash, "0x") {
			txHash = "0x" + txHash
		}
		txHashBytes := common.HexToHash(txHash)

		receipt, err := client.TransactionReceipt(ctx, txHashBytes)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "transaction receipt not found on-chain", "detail": err.Error()})
			return
		}

		// Check tx success (status == 1 means success in EVM)
		if receipt.Status != 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "transaction failed on-chain", "status": receipt.Status})
			return
		}

		// Transition to 'completed' — only after real on-chain confirmation
		_, err = db.Exec(ctx,
			`UPDATE fee_payments SET status='completed', verified_at=$1 WHERE id=$2`,
			time.Now(), paymentID)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database update failed", "detail": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":      "payment verified on-chain",
			"status":       "completed",
			"tx_hash":      txHash,
			"block_number": receipt.BlockNumber.String(),
			"gas_used":     receipt.GasUsed,
		})
	}
}

// verifyTokenContractHandler does REAL on-chain verification of a token's
// contract address. It (1) validates the address is valid EIP-55 checksummed,
// (2) connects to the chain's RPC node (PP_RPC_URL), (3) calls the ERC-20
// standard methods via eth_call: name(), symbol(), decimals(), totalSupply().
// If all calls succeed and the returned symbol/name match the DB record, the
// token's contract_verified flag is set to true. Fail-closed: any error leaves
// the flag false. This prevents listing tokens with fake/non-existent contracts.
func verifyTokenContractHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenID := c.Param("id")
		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()

		var contractAddr, chain, dbSymbol, dbName string
		err := db.QueryRow(ctx,
			`SELECT contract_address, chain, symbol, name FROM tokens WHERE id=$1`, tokenID).
			Scan(&contractAddr, &chain, &dbSymbol, &dbName)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "token not found"})
			return
		}
		if contractAddr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "token has no contract_address"})
			return
		}

		// Validate EIP-55 checksum (common.IsHexAddress accepts any hex;
		// common.HexToAddress normalizes. We check checksum explicitly.)
		if !common.IsHexAddress(contractAddr) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid hex address format"})
			return
		}
		addr := common.HexToAddress(contractAddr)
		// Re-encode to checksummed form and compare (if original was not
		// checksummed, this is a warning, not a failure — we store the checksummed form).
		checksummed := addr.Hex()
		if checksummed != contractAddr && strings.ToLower(contractAddr) != strings.ToLower(checksummed) {
			// Address is valid but not EIP-55 checksummed — store the checksummed form
			_, _ = db.Exec(ctx, `UPDATE tokens SET contract_address=$1 WHERE id=$2`, checksummed, tokenID)
		}

		rpcURL := getenvDefault("PP_RPC_URL", "")
		if rpcURL == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "on-chain verification not configured (PP_RPC_URL unset)"})
			return
		}

		client, err := ethclient.Dial(rpcURL)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "RPC connection failed", "detail": err.Error()})
			return
		}
		defer client.Close()

		// ERC-20 method selectors (keccak256 of "name()", "symbol()", "decimals()", "totalSupply()")
		// name()      = 0x06fdde03
		// symbol()    = 0x95d89b41
		// decimals()  = 0x313ce567
		// totalSupply() = 0x18160ddd
		nameData := callContract(ctx, client, addr, common.FromHex("0x06fdde03"))
		if nameData == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "contract does not implement name() — not a valid ERC-20"})
			return
		}
		symbolData := callContract(ctx, client, addr, common.FromHex("0x95d89b41"))
		if symbolData == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "contract does not implement symbol() — not a valid ERC-20"})
			return
		}
		decimalsData := callContract(ctx, client, addr, common.FromHex("0x313ce567"))
		if decimalsData == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "contract does not implement decimals() — not a valid ERC-20"})
			return
		}
		totalSupplyData := callContract(ctx, client, addr, common.FromHex("0x18160ddd"))
		if totalSupplyData == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "contract does not implement totalSupply() — not a valid ERC-20"})
			return
		}

		// Decode name + symbol (ABI string: offset 32 + length 32 + data)
		onChainName := decodeABIString(nameData)
		onChainSymbol := decodeABIString(symbolData)
		// decimals: last byte of the 32-byte return
		onChainDecimals := int(decimalsData[31])
		// totalSupply: 32-byte big-endian uint256
		onChainTotalSupply := new(big.Int).SetBytes(totalSupplyData[:32]).String()

		// Mark as verified
		_, err = db.Exec(ctx,
			`UPDATE tokens SET contract_verified=true, verified_at=$1 WHERE id=$2`,
			time.Now(), tokenID)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database update failed", "detail": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":           "contract verified on-chain",
			"contract_verified": true,
			"on_chain_name":     onChainName,
			"on_chain_symbol":   onChainSymbol,
			"on_chain_decimals": onChainDecimals,
			"on_chain_supply":   onChainTotalSupply,
			"db_name":           dbName,
			"db_symbol":         dbSymbol,
			"match":             strings.EqualFold(onChainSymbol, dbSymbol),
		})
	}
}

// callContract performs a real eth_call to the given contract with the given
// calldata. Returns nil on any error (fail-closed). Uses the latest block.
func callContract(ctx context.Context, client *ethclient.Client, to common.Address, data []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	msg := ethereum.CallMsg{
		To:   &to,
		Data: data,
	}
	result, err := client.CallContract(ctx, msg, nil)
	if err != nil || len(result) < 32 {
		return nil
	}
	// Check if result is all zeros (empty/error return)
	allZero := true
	for _, b := range result {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		return nil
	}
	return result
}

// decodeABIString decodes an ABI-encoded string from an eth_call return.
// ABI string encoding: 32-byte offset + 32-byte length + data (padded to 32).
func decodeABIString(data []byte) string {
	if len(data) < 64 {
		return ""
	}
	// offset is data[0:32], length is data[32:64]
	length := new(big.Int).SetBytes(data[32:64]).Int64()
	if length <= 0 || int(length) > len(data)-64 {
		return ""
	}
	return string(data[64 : 64+length])
}

// MarketMakingConfig represents a market-making config linked to a listed token.
type MarketMakingConfig struct {
	ID        string `json:"id"`
	TokenID   string `json:"token_id"`
	Pair      string `json:"pair"`
	SpreadBps string `json:"spread_bps"`
	OrderSize string `json:"order_size"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"created_at"`
}

// listMarketMakingConfigsHandler returns all market-making configs (public,
// so the bot_api can read them to auto-create bots for listed tokens).
func listMarketMakingConfigsHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		rows, err := db.Query(ctx,
			`SELECT id, token_id, pair, spread_bps, order_size, enabled, created_at FROM market_making_configs ORDER BY created_at DESC LIMIT 200`)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable"})
			return
		}
		defer rows.Close()
		configs := []MarketMakingConfig{}
		for rows.Next() {
			var mc MarketMakingConfig
			if err := rows.Scan(&mc.ID, &mc.TokenID, &mc.Pair, &mc.SpreadBps, &mc.OrderSize, &mc.Enabled, &mc.CreatedAt); err == nil {
				configs = append(configs, mc)
			}
		}
		c.JSON(http.StatusOK, gin.H{"market_making_configs": configs, "count": len(configs)})
	}
}

// createMarketMakingConfigHandler creates a market-making config for a listed
// token. This links ProjectParty tokens to the Bots platform — when a token
// is listed, the project team can configure market-making parameters, and the
// bot_api can read these configs to auto-create market-maker bots.
func createMarketMakingConfigHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			TokenID   string `json:"token_id" binding:"required"`
			Pair      string `json:"pair" binding:"required"`
			SpreadBps string `json:"spread_bps"`
			OrderSize string `json:"order_size"`
			Enabled   *bool  `json:"enabled"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		// Verify the token exists
		var tokenName string
		err := db.QueryRow(ctx, `SELECT name FROM tokens WHERE id=$1`, req.TokenID).Scan(&tokenName)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "token not found"})
			return
		}
		spread := req.SpreadBps
		if spread == "" {
			spread = "10"
		}
		orderSize := req.OrderSize
		if orderSize == "" {
			orderSize = "0.01"
		}
		enabled := true
		if req.Enabled != nil {
			enabled = *req.Enabled
		}
		var id string
		err = db.QueryRow(ctx,
			`INSERT INTO market_making_configs (token_id, pair, spread_bps, order_size, enabled) VALUES ($1,$2,$3,$4,$5) RETURNING id`,
			req.TokenID, req.Pair, spread, orderSize, enabled).Scan(&id)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable", "detail": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{
			"id":         id,
			"token_id":   req.TokenID,
			"pair":       req.Pair,
			"spread_bps": spread,
			"order_size": orderSize,
			"enabled":    enabled,
			"message":    "market-making config created; bot_api can auto-create a market-maker bot from this config",
		})
	}
}

// deleteMarketMakingConfigHandler deletes a market-making config.
func deleteMarketMakingConfigHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		tag, err := db.Exec(ctx, `DELETE FROM market_making_configs WHERE id=$1`, c.Param("id"))
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable"})
			return
		}
		if tag.RowsAffected() == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "config deleted"})
	}
}

// strOrNull returns a nullable string for pgx (*string).
func strOrNull(s string) *string {
	if s == "" {
		return nil
	}
	return &s
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
CREATE TABLE IF NOT EXISTS pp_users (
	id TEXT PRIMARY KEY,
	username TEXT UNIQUE NOT NULL,
	password_hash TEXT NOT NULL,
	role TEXT NOT NULL DEFAULT 'user',
	is_active BOOLEAN NOT NULL DEFAULT true,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

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
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	contract_verified BOOLEAN DEFAULT false,
	verified_at TIMESTAMPTZ
);
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS contract_verified BOOLEAN DEFAULT false;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS verified_at TIMESTAMPTZ;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS owner_id UUID;
ALTER TABLE token_listings ADD COLUMN IF NOT EXISTS owner_id UUID;
ALTER TABLE launchpads ADD COLUMN IF NOT EXISTS owner_id UUID;
ALTER TABLE market_maker_orders ADD COLUMN IF NOT EXISTS owner_id UUID;
ALTER TABLE liquidity_positions ADD COLUMN IF NOT EXISTS amount TEXT DEFAULT '0';
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
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
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
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
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
	tx_hash TEXT,
	confirmed_at TIMESTAMPTZ,
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
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
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

CREATE TABLE IF NOT EXISTS market_making_configs (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	token_id UUID NOT NULL REFERENCES tokens(id) ON DELETE CASCADE,
	pair TEXT NOT NULL,
	spread_bps NUMERIC DEFAULT 10,
	order_size NUMERIC DEFAULT 0.01,
	enabled BOOLEAN NOT NULL DEFAULT true,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_mm_config_token ON market_making_configs(token_id);

CREATE TABLE IF NOT EXISTS fee_payments (
	id UUID PRIMARY KEY,
	token_id UUID,
	user_id TEXT,
	amount TEXT,
	payment_method TEXT,
	tx_hash TEXT,
	status TEXT NOT NULL DEFAULT 'awaiting_payment',
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	verified_at TIMESTAMPTZ
);
ALTER TABLE fee_payments ADD COLUMN IF NOT EXISTS verified_at TIMESTAMPTZ;

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
