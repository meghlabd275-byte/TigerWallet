// TigerWallet Copy Trading Service
// High-Load Distributed Go Implementation
// Allows users to follow and copy trades from expert traders

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/tigerwallet/wl-shared/wlgate"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ServerPort    string `json:"server_port"`
	DBHost       string `json:"db_host"`
	DBPort       string `json:"db_port"`
	DBUser       string `json:"db_user"`
	DBPassword   string `json:"db_password"`
	DBName       string `json:"db_name"`
	RedisHost    string `json:"redis_host"`
	RedisPort    string `json:"redis_port"`
	// White-label license gate (fail-closed). The copy-trading product is a WL
	// product; without a valid license validated against the TigerWallet
	// SuperAdmin control plane, no request is served.
	ControlPlaneURL    string `json:"control_plane_url"`
	ControlPlaneToken  string `json:"control_plane_token"`
	WLClientID         string `json:"wl_client_id"`
	LicenseKey         string `json:"license_key"`
	Product            string `json:"product"`
	InstanceID         string `json:"instance_id"`
	HeartbeatInterval  time.Duration `json:"heartbeat_interval"`
	JWTSecret          string        `json:"jwt_secret"`
}

// ============================================================================
// Data Models
// ============================================================================

// Trader represents an expert trader
type Trader struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	TraderID       string    `gorm:"uniqueIndex" json:"trader_id"`
	UserAddress    string    `gorm:"index" json:"user_address"`
	Username       string    `json:"username"`
	Avatar         string    `json:"avatar"`
	Bio            string    `json:"bio"`
	TotalPnl       float64   `json:"total_pnl"`
	WinRate        float64   `json:"win_rate"`
	TotalTrades    int       `json:"total_trades"`
	Followers      int       `json:"followers"`
	TotalAum       float64   `json:"total_aum"` // Assets Under Management
	Verified      bool      `json:"verified"`
	Status        string    `json:"status"` // ACTIVE, PAUSED, BANNED
	Commission    float64   `json:"commission"` // % of profits
	ChainID       int64     `json:"chain_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Follower represents a user following a trader
type Follower struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	FollowerID    string    `gorm:"uniqueIndex" json:"follower_id"`
	TraderID     string    `gorm:"index" json:"trader_id"`
	TraderAddress string   `json:"trader_address"`
	UserAddress  string    `gorm:"index" json:"user_address"`
	Allocation   float64   `json:"allocation"` // Amount allocated to copy
	CopyRatio    float64   `json:"copy_ratio"` // 0.1 to 10.0
	Status       string    `json:"status"` // ACTIVE, PAUSED, STOPPED
	TotalPnl     float64   `json:"total_pnl"`
	ChainID      int64     `json:"chain_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CopiedTrade represents a copied trade
type CopiedTrade struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	CopiedTradeID string    `gorm:"uniqueIndex" json:"copied_trade_id"`
	FollowerID    string    `json:"follower_id"`
	TraderID      string    `json:"trader_id"`
	UserAddress   string    `json:"user_address"`
	Symbol        string    `json:"symbol"`
	Side          string    `json:"side"` // BUY or SELL
	Amount        float64   `json:"amount"`
	Price         float64   `json:"price"`
	Pnl           float64   `json:"pnl"`
	Status        string    `json:"status"` // OPEN, CLOSED
	ChainID       int64     `json:"chain_id"`
	OpenedAt      time.Time `json:"opened_at"`
	ClosedAt      *time.Time `json:"closed_at"`
}

// TraderTrade represents a trader's trade
type TraderTrade struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	TradeID       string    `gorm:"uniqueIndex" json:"trade_id"`
	TraderID      string    `gorm:"index" json:"trader_id"`
	UserAddress   string    `json:"user_address"`
	Symbol        string    `json:"symbol"`
	Side          string    `json:"side"`
	Amount        float64   `json:"amount"`
	Price         float64   `json:"price"`
	Pnl           float64   `json:"pnl"`
	OpenedAt      time.Time `json:"opened_at"`
	ClosedAt      *time.Time `json:"closed_at"`
	ChainID       int64     `json:"chain_id"`
	CreatedAt     time.Time `json:"created_at"`
}

// LeaderboardEntry for ranking traders
type LeaderboardEntry struct {
	TraderID   string  `json:"trader_id"`
	Username   string  `json:"username"`
	Avatar     string  `json:"avatar"`
	TotalPnl   float64 `json:"total_pnl"`
	WinRate    float64 `json:"win_rate"`
	TotalTrades int   `json:"total_trades"`
	Followers  int    `json:"followers"`
}

// ============================================================================
// Service Implementation
// ============================================================================

type CopyTradingService struct {
	db     *gorm.DB
	redis *redis.Client
	config Config
	mu    sync.RWMutex
}

func NewCopyTradingService(config Config) (*CopyTradingService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	err = db.AutoMigrate(
		&Trader{},
		&Follower{},
		&CopiedTrade{},
		&TraderTrade{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort),
	})

	service := &CopyTradingService{
		db:     db,
		redis:  rdb,
		config: config,
	}

	// Initialize sample traders
	go service.initializeSampleTraders()

	return service, nil
}

func (s *CopyTradingService) initializeSampleTraders() {
        // No sample/seed traders. Trader profiles must come from real on-chain
        // performance data (verified via the backend), not fabricated placeholder
        // addresses (0x1111...), PnL, or win rates. Do not seed fake copy-trading leaders.
}

// ====+
// Trader Management
// ============================================================================

type RegisterTraderRequest struct {
	UserAddress string  `json:"user_address" binding:"required"`
	Username   string  `json:"username" binding:"required"`
	Avatar    string  `json:"avatar"`
	Bio       string  `json:"bio"`
	Commission float64 `json:"commission"`
	ChainID   int64   `json:"chain_id"`
}

func (s *CopyTradingService) RegisterTrader(ctx *gin.Context) {
	var req RegisterTraderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	// Check if already a trader
	var existing Trader
	if s.db.Where("user_address = ?", req.UserAddress).First(&existing).RowsAffected > 0 {
		ctx.JSON(400, gin.H{"success": false, "error": "Already registered as trader"})
		return
	}

	commission := req.Commission
	if commission <= 0 {
		commission = 10.0 // Default 10%
	}

	trader := Trader{
		TraderID:    generateTraderID(req.UserAddress),
		UserAddress: req.UserAddress,
		Username:   req.Username,
		Avatar:     req.Avatar,
		Bio:        req.Bio,
		TotalPnl:   0,
		WinRate:     0,
		TotalTrades: 0,
		Followers:   0,
		TotalAum:   0,
		Verified:   false,
		Status:     "ACTIVE",
		Commission: commission,
		ChainID:    req.ChainID,
	}

	if err := s.db.Create(&trader).Error; err != nil {
		ctx.JSON(500, gin.H{"success": false, "error": "Failed to register"})
		return
	}

	ctx.JSON(200, gin.H{
		"success":    true,
		"trader_id":  trader.TraderID,
		"status":    "ACTIVE",
	})
}

// ============================================================================
// Follower Operations
// ============================================================================

type FollowTraderRequest struct {
	TraderID    string  `json:"trader_id" binding:"required"`
	UserAddress string  `json:"user_address" binding:"required"`
	Allocation  float64 `json:"allocation" binding:"required"`
	CopyRatio   float64 `json:"copy_ratio"`
	ChainID     int64   `json:"chain_id"`
}

func (s *CopyTradingService) FollowTrader(ctx *gin.Context) {
	var req FollowTraderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	// Verify trader exists
	var trader Trader
	if err := s.db.Where("trader_id = ?", req.TraderID).First(&trader).Error; err != nil {
		ctx.JSON(404, gin.H{"success": false, "error": "Trader not found"})
		return
	}

	if trader.Status != "ACTIVE" {
		ctx.JSON(400, gin.H{"success": false, "error": "Trader is not active"})
		return
	}

	copyRatio := req.CopyRatio
	if copyRatio <= 0 {
		copyRatio = 1.0 // Default 1:1
	}

	// Check if already following
	var existing Follower
	result := s.db.Where("trader_id = ? AND user_address = ?", req.TraderID, req.UserAddress).First(&existing)

	if result.RowsAffected > 0 {
		existing.Allocation = req.Allocation
		existing.CopyRatio = copyRatio
		existing.Status = "ACTIVE"
		s.db.Save(&existing)
	} else {
		follower := Follower{
			FollowerID:    generateFollowerID(req.UserAddress, req.TraderID),
			TraderID:      req.TraderID,
			TraderAddress: trader.UserAddress,
			UserAddress:  req.UserAddress,
			Allocation:   req.Allocation,
			CopyRatio:    copyRatio,
			Status:       "ACTIVE",
			TotalPnl:     0,
			ChainID:       req.ChainID,
		}
		s.db.Create(&follower)

		// Update trader followers count
		trader.Followers++
		trader.TotalAum += req.Allocation
		s.db.Save(&trader)
	}

	ctx.JSON(200, gin.H{
		"success":    true,
		"status":     "ACTIVE",
		"allocation": req.Allocation,
	})
}

type UnfollowRequest struct {
	TraderID    string `json:"trader_id" binding:"required"`
	UserAddress string `json:"user_address" binding:"required"`
}

func (s *CopyTradingService) UnfollowTrader(ctx *gin.Context) {
	var req UnfollowRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	var follower Follower
	if err := s.db.Where("trader_id = ? AND user_address = ?", req.TraderID, req.UserAddress).First(&follower).Error; err != nil {
		ctx.JSON(404, gin.H{"success": false, "error": "Not following this trader"})
		return
	}

	// Update follower status
	follower.Status = "STOPPED"
	s.db.Save(&follower)

	// Update trader stats
	var trader Trader
	s.db.Where("trader_id = ?", req.TraderID).First(&trader)
	trader.Followers--
	trader.TotalAum -= follower.Allocation
	s.db.Save(&trader)

	ctx.JSON(200, gin.H{
		"success": true,
		"status":  "STOPPED",
	})
}

// ============================================================================
// Trade Copying
// ============================================================================

type CopyTradeRequest struct {
	FollowerID string  `json:"follower_id" binding:"required"`
	Trade      TraderTrade `json:"trade" binding:"required"`
}

func (s *CopyTradingService) OnTraderTrade(ctx *gin.Context) {
	var req CopyTradeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Get all active followers of this trader
	var followers []Follower
	s.db.Where("trader_id = ? AND status = ?", req.Trade.TraderID, "ACTIVE").Find(&followers)

	for _, follower := range followers {
		// Calculate copied amount based on allocation and copy ratio
		copiedAmount := follower.Allocation * follower.CopyRatio

		// Ensure we don't exceed the follower's total allocation
		var currentExposure float64
		s.db.Model(&CopiedTrade{}).
			Where("follower_id = ? AND status = ?", follower.FollowerID, "OPEN").
			Select("COALESCE(SUM(amount), 0)").
			Scan(&currentExposure)

		if currentExposure+copiedAmount > follower.Allocation*10 { // Max 10x leverage
			continue
		}

		copiedTrade := CopiedTrade{
			CopiedTradeID: generateCopiedTradeID(follower.FollowerID, req.Trade.TradeID),
			FollowerID:   follower.FollowerID,
			TraderID:     req.Trade.TraderID,
			UserAddress:  follower.UserAddress,
			Symbol:       req.Trade.Symbol,
			Side:         req.Trade.Side,
			Amount:       copiedAmount,
			Price:        req.Trade.Price,
			Pnl:          0,
			Status:       "OPEN",
			ChainID:      req.Trade.ChainID,
			OpenedAt:     time.Now(),
		}

		s.db.Create(&copiedTrade)
	}

	ctx.JSON(200, gin.H{
		"success":     true,
		"copied_to":   len(followers),
	})
}

type CloseCopyTradeRequest struct {
	CopiedTradeID string  `json:"copied_trade_id" binding:"required"`
	UserAddress   string  `json:"user_address" binding:"required"`
	ClosePrice    float64 `json:"close_price"`
}

func (s *CopyTradingService) CloseCopiedTrade(ctx *gin.Context) {
	var req CloseCopyTradeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	var copiedTrade CopiedTrade
	if err := s.db.Where("copied_trade_id = ?", req.CopiedTradeID).First(&copiedTrade).Error; err != nil {
		ctx.JSON(404, gin.H{"success": false, "error": "Trade not found"})
		return
	}

	if copiedTrade.UserAddress != req.UserAddress {
		ctx.JSON(403, gin.H{"success": false, "error": "Unauthorized"})
		return
	}

	// Calculate PnL
	var pnl float64
	if copiedTrade.Side == "BUY" {
		pnl = (req.ClosePrice - copiedTrade.Price) * copiedTrade.Amount
	} else {
		pnl = (copiedTrade.Price - req.ClosePrice) * copiedTrade.Amount
	}

	now := time.Now()
	copiedTrade.Pnl = pnl
	copiedTrade.Status = "CLOSED"
	copiedTrade.ClosedAt = &now
	s.db.Save(&copiedTrade)

	// Update follower stats
	var follower Follower
	s.db.Where("follower_id = ?", copiedTrade.FollowerID).First(&follower)
	follower.TotalPnl += pnl
	s.db.Save(&follower)

	// Update trader stats (for commission calculation)
	var trader Trader
	s.db.Where("trader_id = ?", copiedTrade.TraderID).First(&trader)

	if pnl > 0 {
		commission := pnl * (trader.Commission / 100)
		trader.TotalPnl += commission
	}
	s.db.Save(&trader)

	ctx.JSON(200, gin.H{
		"success":    true,
		"pnl":        pnl,
		"commission": pnl * (trader.Commission / 100),
	})
}

// ============================================================================
// Queries
// ============================================================================

func (s *CopyTradingService) GetTraders(ctx *gin.Context) {
	limit := ctx.DefaultQuery("limit", "50")
	offset := ctx.DefaultQuery("offset", "0")

	var traders []Trader
	s.db.Where("status = ?", "ACTIVE").
		Order("total_pnl DESC").
		Limit(parseInt(limit)).
		Offset(parseInt(offset)).
		Find(&traders)

	ctx.JSON(200, gin.H{"traders": traders})
}

func (s *CopyTradingService) GetTrader(ctx *gin.Context) {
	traderID := ctx.Param("id")

	var trader Trader
	if err := s.db.Where("trader_id = ?", traderID).First(&trader).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "Trader not found"})
		return
	}

	// Get recent trades
	var trades []TraderTrade
	s.db.Where("trader_id = ?", traderID).
		Order("created_at DESC").
		Limit(20).
		Find(&trades)

	// Get follower count
	var followerCount int64
	s.db.Model(&Follower{}).Where("trader_id = ? AND status = ?", traderID, "ACTIVE").Count(&followerCount)

	ctx.JSON(200, gin.H{
		"trader":        trader,
		"recent_trades": trades,
		"follower_count": followerCount,
	})
}

func (s *CopyTradingService) GetLeaderboard(ctx *gin.Context) {
	var traders []Trader
	s.db.Where("status = ?", "ACTIVE").
		Order("total_pnl DESC").
		Limit(100).
		Find(&traders)

	entries := make([]LeaderboardEntry, len(traders))
	for i, trader := range traders {
		entries[i] = LeaderboardEntry{
			TraderID:   trader.TraderID,
			Username:   trader.Username,
			Avatar:     trader.Avatar,
			TotalPnl:   trader.TotalPnl,
			WinRate:    trader.WinRate,
			TotalTrades: trader.TotalTrades,
			Followers:  trader.Followers,
		}
	}

	// Sort by different criteria
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].TotalPnl > entries[j].TotalPnl
	})

	ctx.JSON(200, gin.H{"leaderboard": entries})
}

func (s *CopyTradingService) GetUserFollowings(ctx *gin.Context) {
	userAddress := ctx.Query("user_address")

	var followings []Follower
	s.db.Where("user_address = ? AND status = ?", userAddress, "ACTIVE").Find(&followings)

	// Get trader details
	type FollowingWithTrader struct {
		Follower
		Trader
	}

	result := make([]FollowingWithTrader, len(followings))
	for i, f := range followings {
		var t Trader
		s.db.Where("trader_id = ?", f.TraderID).First(&t)
		result[i] = FollowingWithTrader{
			Follower: f,
			Trader:  t,
		}
	}

	ctx.JSON(200, gin.H{"followings": result})
}

func (s *CopyTradingService) GetUserCopiedTrades(ctx *gin.Context) {
	userAddress := ctx.Query("user_address")

	var trades []CopiedTrade
	s.db.Where("user_address = ?", userAddress).
		Order("opened_at DESC").
		Find(&trades)

	ctx.JSON(200, gin.H{"trades": trades})
}

// ============================================================================
// Helper Functions
// ============================================================================

func generateTraderID(userAddress string) string {
	data := fmt.Sprintf("trader:%s:%d", userAddress, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return "t_" + hex.EncodeToString(hash[:])[0:12]
}

func generateFollowerID(userAddress, traderID string) string {
	data := fmt.Sprintf("follower:%s:%s:%d", userAddress, traderID, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return "f_" + hex.EncodeToString(hash[:])[0:12]
}

func generateCopiedTradeID(followerID, tradeID string) string {
	data := fmt.Sprintf("copy:%s:%s:%d", followerID, tradeID, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return "c_" + hex.EncodeToString(hash[:])[0:16]
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := Config{
		ServerPort:        "8099",
		DBHost:           getEnv("DB_HOST", "localhost"),
		DBPort:           getEnv("DB_PORT", "5432"),
		DBUser:           getEnv("DB_USER", "tigerwallet"),
		DBPassword:       getEnv("DB_PASSWORD", "password"),
		DBName:           getEnv("DB_NAME", "tigerwallet_copytrading"),
		RedisHost:        getEnv("REDIS_HOST", "localhost"),
		RedisPort:        getEnv("REDIS_PORT", "6379"),
		ControlPlaneURL:   getEnv("TWO_PARTY_GATE_URL", ""),
		ControlPlaneToken: getEnv("TWO_PARTY_GATE_TOKEN", ""),
		WLClientID:        getEnv("WL_CLIENT_ID", ""),
		LicenseKey:        getEnv("WL_LICENSE_KEY", ""),
		Product:           getEnv("WL_PRODUCT", "copy_trading"),
		InstanceID:        getEnv("WL_INSTANCE_ID", "default"),
		HeartbeatInterval: getDurationEnv("HEARTBEAT_INTERVAL", 30*time.Second),
		JWTSecret:         getEnv("JWT_SECRET", ""),
	}

	// Fail-closed license gate (mirrors wl_shared/wlgate). The copy-trading
	// product starts DEAD and only serves requests once a valid license has
	// been validated against the TigerWallet SuperAdmin control plane.
	gate := wlgate.New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go gate.HeartbeatLoop(ctx, config.ControlPlaneURL, config.ControlPlaneToken,
		config.LicenseKey, config.Product, config.InstanceID, config.HeartbeatInterval)

	service, err := NewCopyTradingService(config)
	if err != nil {
		fmt.Printf("Failed to start copy trading service: %v\n", err)
		os.Exit(1)
	}

	router := gin.Default()

	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// License gate on every API route (fail-closed 503 when not authorized).
	api := router.Group("/api/v1/copytrading", gate.Middleware(config.Product, wlgate.CategoryFetcher))
	{
		api.GET("/traders", service.GetTraders)
		api.GET("/traders/:id", service.GetTrader)
		api.GET("/leaderboard", service.GetLeaderboard)
		api.POST("/register-trader", service.RegisterTrader)
		api.POST("/follow", service.FollowTrader)
		api.POST("/unfollow", service.UnfollowTrader)
		api.POST("/on-trade", service.OnTraderTrade)
		api.POST("/close-trade", service.CloseCopiedTrade)
		api.GET("/followings", service.GetUserFollowings)
		api.GET("/trades", service.GetUserCopiedTrades)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "copytrading",
			"licensed": gate.IsAlive(),
		})
	})

	go func() {
		fmt.Printf("Copy trading service starting on port %s\n", config.ServerPort)
		if err := router.Run(":" + config.ServerPort); err != nil {
			fmt.Printf("Failed to start server: %v\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	cancel()
	fmt.Println("Shutting down copy trading service...")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getDurationEnv(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func parseInt(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}
