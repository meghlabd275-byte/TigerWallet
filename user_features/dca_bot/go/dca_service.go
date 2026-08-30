// TigerWallet DCA (Dollar-Cost Averaging) Bot Service
// Automated periodic buying of assets at regular intervals
// Reduces risk by averaging into positions over time

package main

import (
	"fmt"
	"math"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ServerPort string `json:"server_port"`
	DBHost     string `json:"db_host"`
	DBPort     string `json:"db_port"`
	DBUser     string `json:"db_user"`
	DBPassword string `json:"db_password"`
	DBName     string `json:"db_name"`
	RedisHost  string `json:"redis_host"`
	RedisPort  string `json:"redis_port"`
}

// ============================================================================
// Data Models
// ============================================================================

// DCAStrategy represents a DCA strategy
type DCAStrategy struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	UserAddress      string    `gorm:"index" json:"user_address"`
	Symbol           string    `json:"symbol"`
	Exchange         string    `json:"exchange"`
	InvestmentTotal  float64   `json:"investment_total"` // Total investment planned
	InvestmentPerBuy float64   `json:"investment_per_buy"`
	TotalBought      float64   `json:"total_bought"`
	TotalSpent       float64   `json:"total_spent"`
	BuyInterval      int       `json:"buy_interval"` // seconds
	TotalBuys        int       `json:"total_buys"`
	CompletedBuys    int       `json:"completed_buys"`
	Status           string    `json:"status"` // ACTIVE, PAUSED, COMPLETED, CANCELLED
	TakeProfitPct    float64   `json:"take_profit_pct"`
	StopLossPct      float64   `json:"stop_loss_pct"`
	StartTime        int64     `json:"start_time"`
	NextBuyTime      int64     `json:"next_buy_time"`
	ChainID          int64     `json:"chain_id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// DCABuy represents individual buy orders
type DCABuy struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	StrategyID  uint       `gorm:"index" json:"strategy_id"`
	OrderNumber int        `json:"order_number"`
	Amount      float64    `json:"amount"`
	Price       float64    `json:"price"`
	TotalSpent  float64    `json:"total_spent"`
	Status      string     `json:"status"` // PENDING, EXECUTED, SKIPPED, FAILED
	ExecutedAt  *time.Time `json:"executed_at"`
	TxHash      string     `json:"tx_hash"`
	ChainID     int64      `json:"chain_id"`
	CreatedAt   time.Time  `json:"created_at"`
}

// ============================================================================
// Service Implementation
// ============================================================================

type DCAService struct {
	db               *gorm.DB
	redis            *redis.Client
	config           Config
	mu               sync.RWMutex
	activeStrategies map[uint]*DCAStrategy
}

func NewDCAService(config Config) (*DCAService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	err = db.AutoMigrate(
		&DCAStrategy{},
		&DCABuy{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort),
	})

	service := &DCAService{
		db:               db,
		redis:            rdb,
		config:           config,
		activeStrategies: make(map[uint]*DCAStrategy),
	}

	go service.loadActiveStrategies()
	go service.startBuyScheduler()

	return service, nil
}

func (s *DCAService) loadActiveStrategies() {
	var strategies []DCAStrategy
	s.db.Where("status = ?", "ACTIVE").Find(&strategies)
	for i := range strategies {
		s.activeStrategies[strategies[i].ID] = &strategies[i]
	}
}

// ============================================================================
// Strategy Management
// ============================================================================

type CreateStrategyRequest struct {
	UserAddress      string  `json:"user_address" binding:"required"`
	Symbol           string  `json:"symbol" binding:"required"`
	Exchange         string  `json:"exchange"`
	InvestmentTotal  float64 `json:"investment_total" binding:"required"`
	InvestmentPerBuy float64 `json:"investment_per_buy" binding:"required"`
	BuyInterval      int     `json:"buy_interval" binding:"required"` // in seconds
	TakeProfitPct    float64 `json:"take_profit_pct"`
	StopLossPct      float64 `json:"stop_loss_pct"`
	ChainID          int64   `json:"chain_id"`
}

func (s *DCAService) CreateStrategy(ctx *gin.Context) {
	var req CreateStrategyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	// Validate inputs
	if req.InvestmentPerBuy <= 0 {
		ctx.JSON(400, gin.H{"success": false, "error": "Investment per buy must be positive"})
		return
	}

	if req.BuyInterval < 300 { // Minimum 5 minutes
		ctx.JSON(400, gin.H{"success": false, "error": "Buy interval must be at least 5 minutes"})
		return
	}

	totalBuys := int(math.Ceil(req.InvestmentTotal / req.InvestmentPerBuy))
	startTime := time.Now().Unix()
	nextBuyTime := startTime + int64(req.BuyInterval)

	strategy := DCAStrategy{
		UserAddress:      req.UserAddress,
		Symbol:           req.Symbol,
		Exchange:         req.Exchange,
		InvestmentTotal:  req.InvestmentTotal,
		InvestmentPerBuy: req.InvestmentPerBuy,
		TotalBought:      0,
		TotalSpent:       0,
		BuyInterval:      req.BuyInterval,
		TotalBuys:        totalBuys,
		CompletedBuys:    0,
		Status:           "ACTIVE",
		TakeProfitPct:    req.TakeProfitPct,
		StopLossPct:      req.StopLossPct,
		StartTime:        startTime,
		NextBuyTime:      nextBuyTime,
		ChainID:          req.ChainID,
	}

	if err := s.db.Create(&strategy).Error; err != nil {
		ctx.JSON(500, gin.H{"success": false, "error": "Failed to create strategy"})
		return
	}

	s.activeStrategies[strategy.ID] = &strategy

	ctx.JSON(200, gin.H{
		"success":            true,
		"strategy_id":        strategy.ID,
		"total_buys":         strategy.TotalBuys,
		"investment_per_buy": strategy.InvestmentPerBuy,
		"buy_interval":       strategy.BuyInterval,
		"next_buy_time":      strategy.NextBuyTime,
	})
}

// ============================================================================
// Strategy Control
// ============================================================================

type ControlRequest struct {
	UserAddress string `json:"user_address" binding:"required"`
	StrategyID  uint   `json:"strategy_id" binding:"required"`
}

func (s *DCAService) PauseStrategy(ctx *gin.Context) {
	var req ControlRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	var strategy DCAStrategy
	if err := s.db.First(&strategy, req.StrategyID).Error; err != nil {
		ctx.JSON(404, gin.H{"success": false, "error": "Strategy not found"})
		return
	}

	if strategy.UserAddress != req.UserAddress {
		ctx.JSON(403, gin.H{"success": false, "error": "Unauthorized"})
		return
	}

	strategy.Status = "PAUSED"
	s.db.Save(&strategy)

	delete(s.activeStrategies, strategy.ID)

	ctx.JSON(200, gin.H{"success": true, "status": "PAUSED"})
}

func (s *DCAService) ResumeStrategy(ctx *gin.Context) {
	var req ControlRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	var strategy DCAStrategy
	if err := s.db.First(&strategy, req.StrategyID).Error; err != nil {
		ctx.JSON(404, gin.H{"success": false, "error": "Strategy not found"})
		return
	}

	if strategy.UserAddress != req.UserAddress {
		ctx.JSON(403, gin.H{"success": false, "error": "Unauthorized"})
		return
	}

	strategy.Status = "ACTIVE"
	strategy.NextBuyTime = time.Now().Unix() + int64(strategy.BuyInterval)
	s.db.Save(&strategy)

	s.activeStrategies[strategy.ID] = &strategy

	ctx.JSON(200, gin.H{"success": true, "status": "ACTIVE"})
}

func (s *DCAService) StopStrategy(ctx *gin.Context) {
	var req ControlRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	var strategy DCAStrategy
	if err := s.db.First(&strategy, req.StrategyID).Error; err != nil {
		ctx.JSON(404, gin.H{"success": false, "error": "Strategy not found"})
		return
	}

	if strategy.UserAddress != req.UserAddress {
		ctx.JSON(403, gin.H{"success": false, "error": "Unauthorized"})
		return
	}

	strategy.Status = "CANCELLED"
	s.db.Save(&strategy)

	delete(s.activeStrategies, strategy.ID)

	ctx.JSON(200, gin.H{
		"success":      true,
		"status":       "CANCELLED",
		"total_bought": strategy.TotalBought,
		"total_spent":  strategy.TotalSpent,
	})
}

// ============================================================================
// Buy Execution Scheduler
// ============================================================================

func (s *DCAService) startBuyScheduler() {
	ticker := time.NewTicker(10 * time.Second) // Check every 10 seconds
	defer ticker.Stop()

	for range ticker.C {
		currentTime := time.Now().Unix()

		for _, strategy := range s.activeStrategies {
			if strategy.Status != "ACTIVE" {
				continue
			}

			if strategy.CompletedBuys >= strategy.TotalBuys {
				strategy.Status = "COMPLETED"
				s.db.Save(strategy)
				delete(s.activeStrategies, strategy.ID)
				continue
			}

			if currentTime >= strategy.NextBuyTime {
				s.executeBuy(strategy)
			}
		}
	}
}

func (s *DCAService) executeBuy(strategy *DCAStrategy) {
	// Get real current price. Fail-closed: skip buy if no live price (no divide-by-zero).
	currentPrice := s.getCurrentPrice(strategy.Symbol)
	if currentPrice <= 0 {
		return
	}

	amount := strategy.InvestmentPerBuy / currentPrice
	buy := DCABuy{
		StrategyID:  strategy.ID,
		OrderNumber: strategy.CompletedBuys + 1,
		Amount:      amount,
		Price:       currentPrice,
		TotalSpent:  strategy.InvestmentPerBuy,
		Status:      "pending_broadcast",
		TxHash:      "",
		ExecutedAt:  new(time.Time),
	}
	*buy.ExecutedAt = time.Now()

	s.db.Create(&buy)

	// Update strategy
	strategy.TotalBought += amount
	strategy.TotalSpent += strategy.InvestmentPerBuy
	strategy.CompletedBuys++
	strategy.NextBuyTime = time.Now().Unix() + int64(strategy.BuyInterval)

	// Check take profit / stop loss
	if strategy.TakeProfitPct > 0 || strategy.StopLossPct > 0 {
		avgPrice := strategy.TotalSpent / strategy.TotalBought
		profitPct := (currentPrice - avgPrice) / avgPrice * 100

		if strategy.TakeProfitPct > 0 && profitPct >= strategy.TakeProfitPct {
			strategy.Status = "COMPLETED"
		} else if strategy.StopLossPct > 0 && profitPct <= -strategy.StopLossPct {
			strategy.Status = "COMPLETED"
		}
	}

	if strategy.CompletedBuys >= strategy.TotalBuys {
		strategy.Status = "COMPLETED"
	}

	s.db.Save(strategy)

	if strategy.Status == "COMPLETED" {
		delete(s.activeStrategies, strategy.ID)
	}
}

// getCurrentPrice returns the real USD price for a symbol from the CoinGecko
// oracle. Fail-closed: returns 0 when no live price is available (callers
// must treat 0 as "no price", never a fabricated number).
func (s *DCAService) getCurrentPrice(symbol string) float64 {
	base := strings.SplitN(symbol, "-", 2)[0]
	live, err := fetchLivePricesUSD([]string{base})
	if err != nil {
		return 0
	}
	return live[base]
}

// ============================================================================
// Queries
// ============================================================================

func (s *DCAService) GetStrategies(ctx *gin.Context) {
	userAddress := ctx.Query("user_address")
	chainID := ctx.GetInt64("chain_id")

	if userAddress == "" {
		ctx.JSON(400, gin.H{"error": "user_address required"})
		return
	}

	var strategies []DCAStrategy
	s.db.Where("user_address = ? AND chain_id = ?", userAddress, chainID).Find(&strategies)

	ctx.JSON(200, gin.H{"strategies": strategies})
}

func (s *DCAService) GetStrategyDetails(ctx *gin.Context) {
	strategyID := ctx.Param("id")

	var strategy DCAStrategy
	if err := s.db.First(&strategy, strategyID).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "Strategy not found"})
		return
	}

	var buys []DCABuy
	s.db.Where("strategy_id = ?", strategy.ID).Order("order_number").Find(&buys)

	// Calculate average price
	var avgPrice float64
	if strategy.TotalBought > 0 {
		avgPrice = strategy.TotalSpent / strategy.TotalBought
	}

	ctx.JSON(200, gin.H{
		"strategy":      strategy,
		"buys":          buys,
		"average_price": avgPrice,
		"current_price": s.getCurrentPrice(strategy.Symbol),
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

// ============================================================================
// Main
// ============================================================================

func main() {
	config := Config{
		ServerPort: "8094",
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "tigerwallet"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "tigerwallet_dca"),
		RedisHost:  getEnv("REDIS_HOST", "localhost"),
		RedisPort:  getEnv("REDIS_PORT", "6379"),
	}

	service, err := NewDCAService(config)
	if err != nil {
		fmt.Printf("Failed to start DCA service: %v\n", err)
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

	api := router.Group("/api/v1/dca")
	{
		api.GET("/strategies", service.GetStrategies)
		api.GET("/strategies/:id", service.GetStrategyDetails)
		api.POST("/create", service.CreateStrategy)
		api.POST("/pause", service.PauseStrategy)
		api.POST("/resume", service.ResumeStrategy)
		api.POST("/stop", service.StopStrategy)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "dca"})
	})

	go func() {
		fmt.Printf("DCA service starting on port %s\n", config.ServerPort)
		if err := router.Run(":" + config.ServerPort); err != nil {
			fmt.Printf("Failed to start server: %v\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down DCA service...")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
