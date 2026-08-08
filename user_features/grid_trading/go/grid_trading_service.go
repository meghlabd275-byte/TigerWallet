// TigerWallet Grid Trading Bot Service
// Automated grid trading strategy execution
// Buys low, sells high within a price range

package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/signal"
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
	ServerPort    string `json:"server_port"`
	DBHost       string `json:"db_host"`
	DBPort       string `json:"db_port"`
	DBUser       string `json:"db_user"`
	DBPassword   string `json:"db_password"`
	DBName       string `json:"db_name"`
	RedisHost    string `json:"redis_host"`
	RedisPort    string `json:"redis_port"`
}

// ============================================================================
// Data Models
// ============================================================================

// GridStrategy represents a grid trading strategy
type GridStrategy struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	UserAddress     string    `gorm:"index" json:"user_address"`
	Symbol          string    `json:"symbol"`
	Exchange        string    `json:"exchange"` // BINANCE, BYBIT, OKX
	GridLevels      int       `json:"grid_levels"`
	MinPrice        float64   `json:"min_price"`
	MaxPrice        float64   `json:"max_price"`
	InvestmentTotal float64   `json:"investment_total"`
	InvestmentPerGrid float64 `json:"investment_per_grid"`
	CurrentPrice    float64   `json:"current_price"`
	TotalProfit    float64   `json:"total_profit"`
	Status         string    `json:"status"` // ACTIVE, PAUSED, STOPPED
	Side           string    `json:"side"` // LONG, SHORT, BOTH
	TakeProfitPct  float64   `json:"take_profit_pct"`
	StopLossPct    float64   `json:"stop_loss_pct"`
	ChainID        int64     `json:"chain_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// GridOrder represents a grid level order
type GridOrder struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	StrategyID    uint      `gorm:"index" json:"strategy_id"`
	Level         int       `json:"level"`
	Side          string    `json:"side"` // BUY or SELL
	Price         float64   `json:"price"`
	Amount        float64   `json:"amount"`
	Status        string    `json:"status"` // PENDING, FILLED, CANCELLED
	FilledPrice   float64   `json:"filled_price"`
	Profit        float64   `json:"profit"`
	FilledAt      *time.Time `json:"filled_at"`
	CreatedAt     time.Time `json:"created_at"`
}

// GridTrade represents executed trades
type GridTrade struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	StrategyID    uint      `json:"strategy_id"`
	OrderID       uint      `json:"order_id"`
	Level         int       `json:"level"`
	Side          string    `json:"side"`
	Price         float64   `json:"price"`
	Amount        float64   `json:"amount"`
	Profit        float64   `json:"profit"`
	ChainID       int64     `json:"chain_id"`
	CreatedAt     time.Time `json:"created_at"`
}

// ============================================================================
// Service Implementation
// ============================================================================

type GridTradingService struct {
	db     *gorm.DB
	redis  *redis.Client
	config Config
	mu     sync.RWMutex
	activeStrategies map[uint]*GridStrategy
}

func NewGridTradingService(config Config) (*GridTradingService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	err = db.AutoMigrate(
		&GridStrategy{},
		&GridOrder{},
		&GridTrade{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort),
	})

	service := &GridTradingService{
		db:     db,
		redis:  rdb,
		config: config,
		activeStrategies: make(map[uint]*GridStrategy),
	}

	// Load active strategies
	go service.loadActiveStrategies()

	return service, nil
}

func (s *GridTradingService) loadActiveStrategies() {
	var strategies []GridStrategy
	s.db.Where("status = ?", "ACTIVE").Find(&strategies)
	for i := range strategies {
		s.activeStrategies[strategies[i].ID] = &strategies[i]
	}
}

// ============================================================================
// Strategy Management
// ============================================================================

type CreateStrategyRequest struct {
	UserAddress     string  `json:"user_address" binding:"required"`
	Symbol         string  `json:"symbol" binding:"required"`
	Exchange       string  `json:"exchange"`
	GridLevels     int     `json:"grid_levels" binding:"required"`
	MinPrice       float64 `json:"min_price" binding:"required"`
	MaxPrice       float64 `json:"max_price" binding:"required"`
	InvestmentTotal float64 `json:"investment_total" binding:"required"`
	Side           string  `json:"side"`
	TakeProfitPct  float64 `json:"take_profit_pct"`
	StopLossPct    float64 `json:"stop_loss_pct"`
	ChainID        int64   `json:"chain_id"`
}

func (s *GridTradingService) CreateStrategy(ctx *gin.Context) {
	var req CreateStrategyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	// Validate inputs
	if req.GridLevels < 2 || req.GridLevels > 100 {
		ctx.JSON(400, gin.H{"success": false, "error": "Grid levels must be between 2 and 100"})
		return
	}

	if req.MinPrice >= req.MaxPrice {
		ctx.JSON(400, gin.H{"success": false, "error": "Min price must be less than max price"})
		return
	}

	investmentPerGrid := req.InvestmentTotal / float64(req.GridLevels)

	strategy := GridStrategy{
		UserAddress:      req.UserAddress,
		Symbol:           req.Symbol,
		Exchange:         req.Exchange,
		GridLevels:       req.GridLevels,
		MinPrice:         req.MinPrice,
		MaxPrice:         req.MaxPrice,
		InvestmentTotal:  req.InvestmentTotal,
		InvestmentPerGrid: investmentPerGrid,
		CurrentPrice:     (req.MinPrice + req.MaxPrice) / 2,
		TotalProfit:      0,
		Status:           "ACTIVE",
		Side:            req.Side,
		TakeProfitPct:   req.TakeProfitPct,
		StopLossPct:     req.StopLossPct,
		ChainID:         req.ChainID,
	}

	if err := s.db.Create(&strategy).Error; err != nil {
		ctx.JSON(500, gin.H{"success": false, "error": "Failed to create strategy"})
		return
	}

	// Create grid orders
	s.createGridOrders(&strategy)

	s.activeStrategies[strategy.ID] = &strategy

	ctx.JSON(200, gin.H{
		"success":       true,
		"strategy_id":    strategy.ID,
		"grid_levels":    strategy.GridLevels,
		"min_price":      strategy.MinPrice,
		"max_price":      strategy.MaxPrice,
		"investment_per_grid": strategy.InvestmentPerGrid,
	})
}

func (s *GridTradingService) createGridOrders(strategy *GridStrategy) {
	priceRange := strategy.MaxPrice - strategy.MinPrice
	levelStep := priceRange / float64(strategy.GridLevels-1)

	for i := 0; i < strategy.GridLevels; i++ {
		price := strategy.MinPrice + float64(i)*levelStep

		// Create buy order at this level
		buyOrder := GridOrder{
			StrategyID: strategy.ID,
			Level:     i,
			Side:      "BUY",
			Price:     price,
			Amount:    strategy.InvestmentPerGrid / price,
			Status:    "PENDING",
		}
		s.db.Create(&buyOrder)

		// Create sell order at this level
		sellOrder := GridOrder{
			StrategyID: strategy.ID,
			Level:     i,
			Side:      "SELL",
			Price:     price,
			Amount:    strategy.InvestmentPerGrid / price,
			Status:    "PENDING",
		}
		s.db.Create(&sellOrder)
	}
}

// ============================================================================
// Strategy Control
// ============================================================================

type ControlRequest struct {
	UserAddress string `json:"user_address" binding:"required"`
	StrategyID  uint   `json:"strategy_id" binding:"required"`
}

func (s *GridTradingService) PauseStrategy(ctx *gin.Context) {
	var req ControlRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	var strategy GridStrategy
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

func (s *GridTradingService) ResumeStrategy(ctx *gin.Context) {
	var req ControlRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	var strategy GridStrategy
	if err := s.db.First(&strategy, req.StrategyID).Error; err != nil {
		ctx.JSON(404, gin.H{"success": false, "error": "Strategy not found"})
		return
	}

	if strategy.UserAddress != req.UserAddress {
		ctx.JSON(403, gin.H{"success": false, "error": "Unauthorized"})
		return
	}

	strategy.Status = "ACTIVE"
	s.db.Save(&strategy)

	s.activeStrategies[strategy.ID] = &strategy

	ctx.JSON(200, gin.H{"success": true, "status": "ACTIVE"})
}

func (s *GridTradingService) StopStrategy(ctx *gin.Context) {
	var req ControlRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	var strategy GridStrategy
	if err := s.db.First(&strategy, req.StrategyID).Error; err != nil {
		ctx.JSON(404, gin.H{"success": false, "error": "Strategy not found"})
		return
	}

	if strategy.UserAddress != req.UserAddress {
		ctx.JSON(403, gin.H{"success": false, "error": "Unauthorized"})
		return
	}

	strategy.Status = "STOPPED"
	s.db.Save(&strategy)

	// Cancel all pending orders
	s.db.Model(&GridOrder{}).Where("strategy_id = ? AND status = ?", strategy.ID, "PENDING").
		Update("status", "CANCELLED")

	delete(s.activeStrategies, strategy.ID)

	ctx.JSON(200, gin.H{
		"success":    true,
		"status":     "STOPPED",
		"total_profit": strategy.TotalProfit,
	})
}

// ============================================================================
// Price Update & Order Execution
// ============================================================================

type UpdatePriceRequest struct {
	Symbol string  `json:"symbol" binding:"required"`
	Price  float64 `json:"price" binding:"required"`
}

func (s *GridTradingService) UpdatePrice(ctx *gin.Context) {
	var req UpdatePriceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Find strategies for this symbol
	var strategies []GridStrategy
	s.db.Where("symbol = ? AND status = ?", req.Symbol, "ACTIVE").Find(&strategies)

	for i := range strategies {
		strategy := &strategies[i]
		previousPrice := strategy.CurrentPrice
		strategy.CurrentPrice = req.Price
		s.db.Save(strategy)

		// Check if any orders should be executed
		s.executeGridOrders(strategy, previousPrice, req.Price)
	}

	ctx.JSON(200, gin.H{"success": true})
}

func (s *GridTradingService) executeGridOrders(strategy *GridStrategy, prevPrice, currentPrice float64) {
	var pendingOrders []GridOrder
	s.db.Where("strategy_id = ? AND status = ?", strategy.ID, "PENDING").Find(&pendingOrders)

	for i := range pendingOrders {
		order := &pendingOrders[i]
		
		var shouldExecute bool
		if order.Side == "BUY" {
			// Execute if price crossed below or at the buy price
			shouldExecute = currentPrice <= order.Price
		} else {
			// Execute if price crossed above or at the sell price
			shouldExecute = currentPrice >= order.Price
		}

		if shouldExecute {
			order.Status = "FILLED"
			order.FilledPrice = order.Price
			order.FilledAt = new(time.Time)
			*order.FilledAt = time.Now()

			// Calculate profit (from the opposite order)
			var profit float64
			var oppositeOrders []GridOrder
			s.db.Where("strategy_id = ? AND level = ? AND side != ? AND status = ?",
				order.StrategyID, order.Level, order.Side, "FILLED").
				Order("filled_at DESC").Limit(1).Find(&oppositeOrders)

			if len(oppositeOrders) > 0 {
				opposite := oppositeOrders[0]
				if order.Side == "BUY" && opposite.Side == "SELL" {
					// Profited from sell - buy low, sell high happened
					profit = (opposite.FilledPrice - order.Price) * order.Amount
				} else if order.Side == "SELL" && opposite.Side == "BUY" {
					// Profited from buy - sell high, buy low happened
					profit = (order.Price - opposite.FilledPrice) * order.Amount
				}
				order.Profit = profit
			}

			s.db.Save(order)

			// Update strategy total profit
			strategy.TotalProfit += profit
			s.db.Save(strategy)

			// Create trade record
			trade := GridTrade{
				StrategyID: strategy.ID,
				OrderID:    order.ID,
				Level:      order.Level,
				Side:       order.Side,
				Price:      order.FilledPrice,
				Amount:     order.Amount,
				Profit:     profit,
				ChainID:    strategy.ChainID,
				CreatedAt:  time.Now(),
			}
			s.db.Create(&trade)

			// Create replacement order at next level
			s.createReplacementOrder(strategy, order)
		}
	}
}

func (s *GridTradingService) createReplacementOrder(strategy *GridStrategy, executedOrder *GridOrder) {
	// For simplicity, we'll recreate the same order
	newOrder := GridOrder{
		StrategyID: executedOrder.StrategyID,
		Level:      executedOrder.Level,
		Side:       executedOrder.Side,
		Price:      executedOrder.Price,
		Amount:     executedOrder.Amount,
		Status:     "PENDING",
	}
	s.db.Create(&newOrder)
}

// ============================================================================
// Queries
// ============================================================================

func (s *GridTradingService) GetStrategies(ctx *gin.Context) {
	userAddress := ctx.Query("user_address")
	chainID := ctx.GetInt64("chain_id")

	if userAddress == "" {
		ctx.JSON(400, gin.H{"error": "user_address required"})
		return
	}

	var strategies []GridStrategy
	s.db.Where("user_address = ? AND chain_id = ?", userAddress, chainID).Find(&strategies)

	ctx.JSON(200, gin.H{"strategies": strategies})
}

func (s *GridTradingService) GetStrategyDetails(ctx *gin.Context) {
	strategyID := ctx.Param("id")

	var strategy GridStrategy
	if err := s.db.First(&strategy, strategyID).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "Strategy not found"})
		return
	}

	var orders []GridOrder
	s.db.Where("strategy_id = ?", strategy.ID).Order("level, side").Find(&orders)

	var trades []GridTrade
	s.db.Where("strategy_id = ?", strategy.ID).Order("created_at DESC").Limit(50).Find(&trades)

	ctx.JSON(200, gin.H{
		"strategy": strategy,
		"orders":   orders,
		"trades":   trades,
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
		ServerPort: "8093",
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "tigerwallet"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "tigerwallet_grid"),
		RedisHost:  getEnv("REDIS_HOST", "localhost"),
		RedisPort:  getEnv("REDIS_PORT", "6379"),
	}

	service, err := NewGridTradingService(config)
	if err != nil {
		fmt.Printf("Failed to start grid trading service: %v\n", err)
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

	api := router.Group("/api/v1/grid")
	{
		api.GET("/strategies", service.GetStrategies)
		api.GET("/strategies/:id", service.GetStrategyDetails)
		api.POST("/create", service.CreateStrategy)
		api.POST("/pause", service.PauseStrategy)
		api.POST("/resume", service.ResumeStrategy)
		api.POST("/stop", service.StopStrategy)
		api.POST("/update-price", service.UpdatePrice)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "grid_trading"})
	})

	go func() {
		fmt.Printf("Grid trading service starting on port %s\n", config.ServerPort)
		if err := router.Run(":" + config.ServerPort); err != nil {
			fmt.Printf("Failed to start server: %v\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down grid trading service...")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
