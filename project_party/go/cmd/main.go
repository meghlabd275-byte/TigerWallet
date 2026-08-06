package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := loadConfig()

	// Initialize database
	db, err := initDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

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
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

func loadConfig() *Config {
	return &Config{
		Port: getEnv("PROJECT_PARTY_PORT", "9006"),
		Database: DatabaseConfig{
			Host:     getEnv("PROJECT_PARTY_DB_HOST", "localhost"),
			Port:     getEnvInt("PROJECT_PARTY_DB_PORT", 5432),
			User:     getEnv("PROJECT_PARTY_DB_USER", "tigerwallet"),
			Password: getEnv("PROJECT_PARTY_DB_PASSWORD", "password"),
			DBName:   getEnv("PROJECT_PARTY_DB_NAME", "tigerwallet_project_party"),
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
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

// Token Handlers
func createTokenHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			TenantID        string   `json:"tenant_id" binding:"required"`
			Name            string   `json:"name" binding:"required"`
			Symbol          string   `json:"symbol" binding:"required"`
			Decimals        int      `json:"decimals" binding:"required"`
			ContractAddress string   `json:"contract_address" binding:"required"`
			Chain           string   `json:"chain" binding:"required"`
			TotalSupply     string   `json:"total_supply" binding:"required"`
			LogoURL         string   `json:"logo_url"`
			Description     string   `json:"description"`
			Website         string   `json:"website"`
			Whitepaper     string   `json:"whitepaper"`
			SocialLinks     map[string]string `json:"social_links"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		token := Token{
			ID:               uuid.New(),
			Name:             req.Name,
			Symbol:           req.Symbol,
			Decimals:         req.Decimals,
			ContractAddress:  req.ContractAddress,
			Chain:            req.Chain,
			TotalSupply:      req.TotalSupply,
			LogoURL:          req.LogoURL,
			Description:      req.Description,
			Website:          req.Website,
			Whitepaper:      req.Whitepaper,
			SocialLinks:      req.SocialLinks,
			Status:           "draft",
			ListingFeeUSD:    500.0,
			IsFeatured:       false,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}

		c.JSON(http.StatusCreated, gin.H{"token": token, "message": "Token created successfully"})
	}
}

func listTokensHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := c.Query("status")
		limit := c.DefaultQuery("limit", "50")
		offset := c.DefaultQuery("offset", "0")

		tokens := []map[string]interface{}{
			{"id": uuid.New().String(), "name": "Bitcoin", "symbol": "BTC", "chain": "Bitcoin", "status": "listed", "is_featured": true},
			{"id": uuid.New().String(), "name": "Ethereum", "symbol": "ETH", "chain": "Ethereum", "status": "listed", "is_featured": true},
			{"id": uuid.New().String(), "name": "TigerWallet", "symbol": "TIGER", "chain": "Ethereum", "status": "in_review", "is_featured": false},
		}

		c.JSON(http.StatusOK, gin.H{"tokens": tokens, "total": 100, "limit": limit, "offset": offset})
	}
}

func getTokenHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenID := c.Param("id")
		token := map[string]interface{}{
			"id":               tokenID,
			"name":             "Sample Token",
			"symbol":           "SAMPLE",
			"decimals":         18,
			"contract_address":  "0x1234...",
			"chain":            "Ethereum",
			"total_supply":     "1000000000",
			"logo_url":         "https://example.com/logo.png",
			"description":     "Sample token description",
			"website":          "https://example.com",
			"whitepaper":       "https://example.com/whitepaper.pdf",
			"social_links":     map[string]string{"twitter": "@sampletoken", "telegram": "@sampletoken"},
			"status":           "listed",
			"listing_fee_usd":  500.0,
			"is_featured":      false,
			"created_at":       time.Now(),
		}
		c.JSON(http.StatusOK, token)
	}
}

func updateTokenHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Token updated"})
	}
}

func deleteTokenHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Token deleted"})
	}
}

func submitTokenHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Token submitted for review", "status": "submitted"})
	}
}

func approveTokenHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Token approved", "status": "approved"})
	}
}

func rejectTokenHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Reason string `json:"reason" binding:"required"`
		}
		c.ShouldBindJSON(&req)
		c.JSON(http.StatusOK, gin.H{"message": "Token rejected", "status": "rejected", "reason": req.Reason})
	}
}

// Listing Handlers
func createListingHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			TokenID      string `json:"token_id" binding:"required"`
			PairToken    string `json:"pair_token" binding:"required"`
			InitialPrice string `json:"initial_price" binding:"required"`
			LaunchType   string `json:"launch_type" binding:"required"`
			StartTime   string `json:"start_time" binding:"required"`
			EndTime     string `json:"end_time" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		listing := map[string]interface{}{
			"id":             uuid.New().String(),
			"token_id":       req.TokenID,
			"pair_token":     req.PairToken,
			"initial_price":  req.InitialPrice,
			"current_price":  req.InitialPrice,
			"launch_type":    req.LaunchType,
			"status":         "upcoming",
			"volume_24h":    "0",
			"liquidity_usd": "0",
			"market_cap":     "0",
			"created_at":    time.Now(),
		}

		c.JSON(http.StatusCreated, gin.H{"listing": listing, "message": "Listing created"})
	}
}

func listListingsHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		listings := []map[string]interface{}{
			{"id": uuid.New().String(), "symbol": "BTC", "pair_token": "USDT", "price": "45000", "volume_24h": "1000000", "status": "active"},
			{"id": uuid.New().String(), "symbol": "ETH", "pair_token": "USDT", "price": "2500", "volume_24h": "500000", "status": "active"},
		}
		c.JSON(http.StatusOK, gin.H{"listings": listings})
	}
}

func getListingHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		listing := map[string]interface{}{
			"id":              c.Param("id"),
			"symbol":          "SAMPLE",
			"pair_token":      "USDT",
			"current_price":   "1.50",
			"volume_24h":     "100000",
			"liquidity_usd":  "50000",
			"market_cap":      "1500000",
			"price_change_24h": 5.5,
			"status":          "active",
		}
		c.JSON(http.StatusOK, listing)
	}
}

func updateListingStatusHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Status string `json:"status" binding:"required"`
		}
		c.ShouldBindJSON(&req)
		c.JSON(http.StatusOK, gin.H{"message": "Listing status updated", "status": req.Status})
	}
}

func featureListingHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Listing featured"})
	}
}

// Launchpad Handlers
func createLaunchpadHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			TokenID          string `json:"token_id" binding:"required"`
			Name             string `json:"name" binding:"required"`
			Description      string `json:"description"`
			SoftCap          string `json:"soft_cap" binding:"required"`
			HardCap          string `json:"hard_cap" binding:"required"`
			MinContribution  string `json:"min_contribution" binding:"required"`
			MaxContribution  string `json:"max_contribution" binding:"required"`
			StartTime        string `json:"start_time" binding:"required"`
			EndTime          string `json:"end_time" binding:"required"`
			TokenPrice       string `json:"token_price" binding:"required"`
			AcceptedPayment  string `json:"accepted_payment" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		launchpad := map[string]interface{}{
			"id":              uuid.New().String(),
			"name":            req.Name,
			"token_id":       req.TokenID,
			"soft_cap":       req.SoftCap,
			"hard_cap":       req.HardCap,
			"total_raised":   "0",
			"status":         "upcoming",
			"created_at":    time.Now(),
		}

		c.JSON(http.StatusCreated, gin.H{"launchpad": launchpad, "message": "Launchpad created"})
	}
}

func listLaunchpadsHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		launchpads := []map[string]interface{}{
			{"id": uuid.New().String(), "name": "IDOToken Sale", "token": "IDO", "hard_cap": "500000", "status": "active", "progress": 75},
		}
		c.JSON(http.StatusOK, gin.H{"launchpads": launchpads})
	}
}

func getLaunchpadHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		launchpad := map[string]interface{}{
			"id":               c.Param("id"),
			"name":             "Sample Launchpad",
			"soft_cap":        "100000",
			"hard_cap":        "500000",
			"total_raised":     "350000",
			"status":          "active",
			"progress":        70,
			"contributors":    150,
		}
		c.JSON(http.StatusOK, launchpad)
	}
}

func contributeHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Amount string `json:"amount" binding:"required"`
		}
		c.ShouldBindJSON(&req)

		contribution := map[string]interface{}{
			"id":           uuid.New().String(),
			"amount":       req.Amount,
			"token_amount": "1000",
			"status":       "pending",
		}

		c.JSON(http.StatusCreated, gin.H{"contribution": contribution, "message": "Contribution successful"})
	}
}

func claimTokensHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Tokens claimed successfully"})
	}
}

func cancelLaunchpadHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Launchpad cancelled", "status": "cancelled"})
	}
}

// Market Making Handlers
func createMakerOrdersHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			TokenID  string `json:"token_id" binding:"required"`
			Side     string `json:"side" binding:"required"`
			Price    string `json:"price" binding:"required"`
			Quantity string `json:"quantity" binding:"required"`
		}
		c.ShouldBindJSON(&req)

		order := map[string]interface{}{
			"id":         uuid.New().String(),
			"token_id":   req.TokenID,
			"side":       req.Side,
			"price":      req.Price,
			"quantity":   req.Quantity,
			"remaining":  req.Quantity,
			"status":     "pending",
			"created_at": time.Now(),
		}

		c.JSON(http.StatusCreated, gin.H{"order": order})
	}
}

func getMakerOrdersHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		orders := []map[string]interface{}{
			{"id": uuid.New().String(), "side": "buy", "price": "1.50", "quantity": "1000", "filled": "500"},
		}
		c.JSON(http.StatusOK, gin.H{"orders": orders})
	}
}

func updateOrderStatusHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Order status updated"})
	}
}

func getMarketMakerStatusHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := map[string]interface{}{
			"token_id":      c.Param("token_id"),
			"active":        true,
			"spread":        0.5,
			"total_orders":  100,
			"filled_orders": 75,
			"volume_24h":    "50000",
		}
		c.JSON(http.StatusOK, status)
	}
}

func addLiquidityHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Liquidity added successfully", "lp_tokens": "150.5"})
	}
}

func removeLiquidityHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Liquidity removed successfully"})
	}
}

// Pricing Handlers
func setTokenPriceHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			TokenID string `json:"token_id" binding:"required"`
			Price   string `json:"price" binding:"required"`
		}
		c.ShouldBindJSON(&req)
		c.JSON(http.StatusOK, gin.H{"message": "Price set", "price": req.Price})
	}
}

func getTokenPriceHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		price := map[string]interface{}{
			"token_id":     c.Param("token_id"),
			"price":        "1.50",
			"change_24h":   5.5,
			"volume_24h":   "100000",
			"market_cap":   "1500000",
			"timestamp":    time.Now(),
		}
		c.JSON(http.StatusOK, price)
	}
}

func getPriceHistoryHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		history := []map[string]interface{}{
			{"price": "1.45", "timestamp": time.Now().Add(-1 * time.Hour)},
			{"price": "1.48", "timestamp": time.Now().Add(-2 * time.Hour)},
			{"price": "1.50", "timestamp": time.Now()},
		}
		c.JSON(http.StatusOK, gin.H{"history": history})
	}
}

func updatePriceHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Price updated"})
	}
}

// Analytics Handlers
func getTradingVolumeHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		volume := map[string]interface{}{
			"total_24h":   "1000000",
			"total_7d":    "7000000",
			"total_30d":   "30000000",
			"by_token":    map[string]string{"BTC": "500000", "ETH": "300000", "USDT": "200000"},
		}
		c.JSON(http.StatusOK, volume)
	}
}

func getLiquidityHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		liquidity := map[string]interface{}{
			"total_liquidity": "5000000",
			"by_pair":        map[string]string{"BTC/USDT": "2000000", "ETH/USDT": "1500000", "TIGER/USDT": "1500000"},
		}
		c.JSON(http.StatusOK, liquidity)
	}
}

func getHolderCountHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenID := c.Query("token_id")
		holders := map[string]interface{}{
			"token_id":   tokenID,
			"total":      5000,
			"new_24h":   50,
		}
		c.JSON(http.StatusOK, holders)
	}
}

func getTransactionCountHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		txs := map[string]interface{}{
			"total_24h": 10000,
			"total_7d":  70000,
		}
		c.JSON(http.StatusOK, txs)
	}
}

// Compliance Handlers
func requestAuditHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			TokenID   string `json:"token_id" binding:"required"`
			AuditType string `json:"audit_type" binding:"required"`
		}
		c.ShouldBindJSON(&req)

		audit := map[string]interface{}{
			"id":           uuid.New().String(),
			"token_id":     req.TokenID,
			"audit_type":   req.AuditType,
			"status":       "requested",
			"requested_at": time.Now(),
		}

		c.JSON(http.StatusCreated, gin.H{"audit": audit})
	}
}

func getAuditStatusHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		audit := map[string]interface{}{
			"token_id":    c.Param("token_id"),
			"security":    map[string]string{"status": "completed", "report_url": "https://audit.com/report.pdf"},
			"code":        map[string]string{"status": "in_progress"},
			"financial":   map[string]string{"status": "pending"},
		}
		c.JSON(http.StatusOK, audit)
	}
}

func submitKYCHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "KYC submitted", "status": "pending"})
	}
}

func getKYCStatusHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		kyc := map[string]interface{}{
			"token_id":      c.Param("token_id"),
			"status":        "approved",
			"verified_at":   time.Now(),
			"expires_at":    time.Now().Add(365 * 24 * time.Hour),
		}
		c.JSON(http.StatusOK, kyc)
	}
}

// Fee Handlers
func getListingFeesHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		fees := map[string]interface{}{
			"basic_listing":      500,
			"featured_listing":  1500,
			"audit_required":     5000,
			"kyc_verification":   1000,
			"launchpad_basic":   5000,
			"launchpad_premium":  15000,
		}
		c.JSON(http.StatusOK, fees)
	}
}

func calculateFeesHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			ListingType string `json:"listing_type" binding:"required"`
			Features   []string `json:"features"`
		}
		c.ShouldBindJSON(&req)

		// Calculate fees based on listing type and features
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
			Amount    string `json:"amount" binding:"required"`
			PaymentMethod string `json:"payment_method" binding:"required"`
		}
		c.ShouldBindJSON(&req)

		c.JSON(http.StatusOK, gin.H{"message": "Payment processed", "transaction_id": uuid.New().String()})
	}
}

// ============== Database ==============

func initDatabase(cfg *Config) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=require",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.DBName,
	)

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}
