// TigerWallet Perpetual Trading Service
// High-Speed Go Implementation for Margin/Perpetual Trading
// Supports long/short positions, leverage, funding, liquidation

package main

import (
	"fmt"
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

// PerpetualMarket represents a perpetual market
type PerpetualMarket struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	Symbol            string    `gorm:"uniqueIndex" json:"symbol"`
	Name              string    `json:"name"`
	BaseAsset         string    `json:"base_asset"`
	QuoteAsset        string    `json:"quote_asset"`
	ContractSize      float64   `json:"contract_size"`
	MinOrderSize      float64   `json:"min_order_size"`
	MaxOrderSize      float64   `json:"max_order_size"`
	MaxLeverage       float64   `json:"max_leverage"`
	MaintenanceMargin float64   `json:"maintenance_margin"`
	MarkPrice         float64   `json:"mark_price"`
	IndexPrice        float64   `json:"index_price"`
	LastPrice         float64   `json:"last_price"`
	FundingRate       float64   `json:"funding_rate"`
	NextFundingTime   int64     `json:"next_funding_time"`
	OpenInterest      float64   `json:"open_interest"`
	Volume24h         float64   `json:"volume_24h"`
	Change24h         float64   `json:"change_24h"`
	High24h           float64   `json:"high_24h"`
	Low24h            float64   `json:"low_24h"`
	IsActive          bool      `json:"is_active"`
	ChainID           int64     `json:"chain_id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// Position represents a user's position
type Position struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	UserAddress      string     `gorm:"index" json:"user_address"`
	MarketID         uint       `gorm:"index" json:"market_id"`
	Symbol           string     `json:"symbol"`
	Side             string     `json:"side"` // LONG or SHORT
	Size             float64    `json:"size"`
	EntryPrice       float64    `json:"entry_price"`
	MarkPrice        float64    `json:"mark_price"`
	Leverage         float64    `json:"leverage"`
	Collateral       float64    `json:"collateral"`
	UnrealizedPNL    float64    `json:"unrealized_pnl"`
	RealizedPNL      float64    `json:"realized_pnl"`
	LiquidationPrice float64    `json:"liquidation_price"`
	StopLossPrice    float64    `json:"stop_loss_price"`
	TakeProfitPrice  float64    `json:"take_profit_price"`
	Status           string     `json:"status"` // OPEN, CLOSED, LIQUIDATED
	ChainID          int64      `json:"chain_id"`
	OpenedAt         time.Time  `json:"opened_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	ClosedAt         *time.Time `json:"closed_at"`
}

// Order represents a trading order
type Order struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	UserAddress  string     `gorm:"index" json:"user_address"`
	MarketID     uint       `gorm:"index" json:"market_id"`
	Symbol       string     `json:"symbol"`
	Side         string     `json:"side"`       // BUY or SELL
	OrderType    string     `json:"order_type"` // MARKET, LIMIT, STOP
	Price        float64    `json:"price"`
	Size         float64    `json:"size"`
	FilledSize   float64    `json:"filled_size"`
	AvgFillPrice float64    `json:"avg_fill_price"`
	Leverage     float64    `json:"leverage"`
	Status       string     `json:"status"` // PENDING, FILLED, PARTIALLY_FILLED, CANCELLED
	ChainID      int64      `json:"chain_id"`
	CreatedAt    time.Time  `json:"created_at"`
	FilledAt     *time.Time `json:"filled_at"`
}

// Trade represents a filled trade
type Trade struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	OrderID     uint      `json:"order_id"`
	UserAddress string    `json:"user_address"`
	MarketID    uint      `json:"market_id"`
	Symbol      string    `json:"symbol"`
	Side        string    `json:"side"`
	Size        float64   `json:"size"`
	Price       float64   `json:"price"`
	Fee         float64   `json:"fee"`
	FeeAsset    string    `json:"fee_asset"`
	RealizedPNL float64   `json:"realized_pnl"`
	ChainID     int64     `json:"chain_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// FundingPayment represents funding rate payments
type FundingPayment struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserAddress string    `json:"user_address"`
	MarketID    uint      `json:"market_id"`
	Symbol      string    `json:"symbol"`
	Rate        float64   `json:"rate"`
	Payment     float64   `json:"payment"`
	Side        string    `json:"side"` // LONG pays SHORT or SHORT pays LONG
	ChainID     int64     `json:"chain_id"`
	PeriodStart int64     `json:"period_start"`
	PeriodEnd   int64     `json:"period_end"`
	CreatedAt   time.Time `json:"created_at"`
}

// ============================================================================
// Service Implementation
// ============================================================================

type PerpetualService struct {
	db         *gorm.DB
	redis      *redis.Client
	config     Config
	markets    map[string]*PerpetualMarket
	mu         sync.RWMutex
	priceFeeds map[string]float64
}

func NewPerpetualService(config Config) (*PerpetualService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	err = db.AutoMigrate(
		&PerpetualMarket{},
		&Position{},
		&Order{},
		&Trade{},
		&FundingPayment{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort),
	})

	service := &PerpetualService{
		db:         db,
		redis:      rdb,
		config:     config,
		markets:    make(map[string]*PerpetualMarket),
		priceFeeds: make(map[string]float64),
	}

	go service.initializeMarkets()
	go service.startPriceFeeds()

	return service, nil
}

// Initialize default markets
// initializeMarkets seeds default perpetual markets with REAL spot prices
// from the CoinGecko oracle (Mark/Index/Last/High/Low). Open-interest and
// 24h volume start at 0 and are filled by real trading activity. Fail-closed:
// an asset with no live price is not listed.
func (s *PerpetualService) initializeMarkets() {
	type mkt struct {
		symbol, name, base string
		contractSize       float64
		minOrder, maxOrder float64
		maxLev             float64
		maintMargin        float64
	}
	defs := []mkt{
		{"ETH-USDT", "Ethereum Perpetual", "ETH", 1, 0.01, 1000, 100, 0.005},
		{"BTC-USDT", "Bitcoin Perpetual", "BTC", 0.001, 0.001, 100, 100, 0.005},
		{"SOL-USDT", "Solana Perpetual", "SOL", 1, 0.1, 10000, 50, 0.01},
		{"ARB-USDT", "Arbitrum Perpetual", "ARB", 1, 1, 100000, 50, 0.01},
	}
	bases := []string{}
	for _, d := range defs {
		bases = append(bases, d.base)
	}
	spot, err := fetchLivePricesUSD(bases)
	if err != nil {
		spot = map[string]float64{}
	}
	for _, d := range defs {
		price := spot[d.base]
		if price <= 0 {
			continue
		}
		market := PerpetualMarket{
			Symbol:            d.symbol,
			Name:              d.name,
			BaseAsset:         d.base,
			QuoteAsset:        "USDT",
			ContractSize:      d.contractSize,
			MinOrderSize:      d.minOrder,
			MaxOrderSize:      d.maxOrder,
			MaxLeverage:       d.maxLev,
			MaintenanceMargin: d.maintMargin,
			MarkPrice:         price,
			IndexPrice:        price,
			LastPrice:         price,
			FundingRate:       0,
			OpenInterest:      0,
			Volume24h:         0,
			Change24h:         0,
			High24h:           price,
			Low24h:            price,
			IsActive:          true,
			ChainID:           1,
		}
		var existing PerpetualMarket
		if s.db.Where("symbol = ?", market.Symbol).First(&existing).RowsAffected == 0 {
			s.db.Create(&market)
		}
		s.markets[market.Symbol] = &market
	}
}

// Simulated price feeds
// startPriceFeeds refreshes mark/index/last prices from the real CoinGecko
// oracle every 10s. No simulated random walk. Fail-closed: keeps last known
// real price on upstream error.
func (s *PerpetualService) startPriceFeeds() {
	refresh := func() {
		s.mu.RLock()
		bases := []string{}
		for _, m := range s.markets {
			bases = append(bases, m.BaseAsset)
		}
		s.mu.RUnlock()
		spot, err := fetchLivePricesUSD(bases)
		if err != nil || len(spot) == 0 {
			return
		}
		s.mu.Lock()
		for symbol, market := range s.markets {
			price := spot[market.BaseAsset]
			if price <= 0 {
				continue
			}
			s.priceFeeds[symbol] = price
			old := market.MarkPrice
			market.MarkPrice = price
			market.IndexPrice = price
			market.LastPrice = price
			if old > 0 {
				market.Change24h = ((price - old) / old) * 100
			}
			if price > market.High24h || market.High24h == 0 {
				market.High24h = price
			}
			if price < market.Low24h || market.Low24h == 0 {
				market.Low24h = price
			}
			s.db.Save(market)
		}
		s.mu.Unlock()
	}
	refresh()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		refresh()
	}
}

// ============================================================================
// Trading Operations
// ============================================================================

type OpenPositionRequest struct {
	UserAddress string  `json:"user_address" binding:"required"`
	Symbol      string  `json:"symbol" binding:"required"`
	Side        string  `json:"side" binding:"required"` // LONG or SHORT
	Size        float64 `json:"size" binding:"required"`
	Leverage    float64 `json:"leverage" binding:"required"`
	OrderType   string  `json:"order_type"` // MARKET or LIMIT
	Price       float64 `json:"price"`
	ChainID     int64   `json:"chain_id"`
}

type OpenPositionResponse struct {
	Success          bool    `json:"success"`
	PositionID       uint    `json:"position_id,omitempty"`
	TransactionHash  string  `json:"transaction_hash,omitempty"`
	EntryPrice       float64 `json:"entry_price"`
	Size             float64 `json:"size"`
	Leverage         float64 `json:"leverage"`
	LiquidationPrice float64 `json:"liquidation_price"`
	Error            string  `json:"error,omitempty"`
}

// Open a new position
func (s *PerpetualService) OpenPosition(ctx *gin.Context) {
	var req OpenPositionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, OpenPositionResponse{Success: false, Error: err.Error()})
		return
	}

	// Get market
	market, ok := s.markets[req.Symbol]
	if !ok {
		ctx.JSON(404, OpenPositionResponse{Success: false, Error: "Market not found"})
		return
	}

	// Validate leverage
	if req.Leverage > market.MaxLeverage {
		ctx.JSON(400, OpenPositionResponse{Success: false, Error: "Leverage exceeds maximum"})
		return
	}

	// Calculate position details
	collateral := req.Size * market.MarkPrice / req.Leverage
	entryPrice := market.MarkPrice
	if req.OrderType == "LIMIT" && req.Price > 0 {
		entryPrice = req.Price
	}

	// Calculate liquidation price
	var liqPrice float64
	if req.Side == "LONG" {
		liqPrice = entryPrice * (1 - (1 / req.Leverage) + market.MaintenanceMargin)
	} else {
		liqPrice = entryPrice * (1 + (1 / req.Leverage) - market.MaintenanceMargin)
	}

	// Create position
	position := Position{
		UserAddress:      req.UserAddress,
		MarketID:         market.ID,
		Symbol:           req.Symbol,
		Side:             req.Side,
		Size:             req.Size,
		EntryPrice:       entryPrice,
		MarkPrice:        entryPrice,
		Leverage:         req.Leverage,
		Collateral:       collateral,
		UnrealizedPNL:    0,
		RealizedPNL:      0,
		LiquidationPrice: liqPrice,
		Status:           "OPEN",
		ChainID:          req.ChainID,
		OpenedAt:         time.Now(),
	}

	if err := s.db.Create(&position).Error; err != nil {
		ctx.JSON(500, OpenPositionResponse{Success: false, Error: "Failed to create position"})
		return
	}

	// Update market open interest
	market.OpenInterest += req.Size * market.MarkPrice
	s.db.Save(market)

	ctx.JSON(200, OpenPositionResponse{
		Success:          true,
		PositionID:       position.ID,
		EntryPrice:       entryPrice,
		Size:             req.Size,
		Leverage:         req.Leverage,
		LiquidationPrice: liqPrice,
	})
}

// Close position
type ClosePositionRequest struct {
	UserAddress string  `json:"user_address" binding:"required"`
	PositionID  uint    `json:"position_id" binding:"required"`
	Size        float64 `json:"size"` // 0 = close all
	ChainID     int64   `json:"chain_id"`
}

func (s *PerpetualService) ClosePosition(ctx *gin.Context) {
	var req ClosePositionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	var position Position
	if err := s.db.First(&position, req.PositionID).Error; err != nil {
		ctx.JSON(404, gin.H{"success": false, "error": "Position not found"})
		return
	}

	if position.UserAddress != req.UserAddress {
		ctx.JSON(403, gin.H{"success": false, "error": "Unauthorized"})
		return
	}

	// Get current mark price
	market, ok := s.markets[position.Symbol]
	if !ok {
		ctx.JSON(404, gin.H{"success": false, "error": "Market not found"})
		return
	}

	currentPrice := market.MarkPrice
	closeSize := req.Size
	if closeSize == 0 || closeSize > position.Size {
		closeSize = position.Size
	}

	// Calculate PnL
	var pnl float64
	if position.Side == "LONG" {
		pnl = (currentPrice - position.EntryPrice) * closeSize
	} else {
		pnl = (position.EntryPrice - currentPrice) * closeSize
	}

	// Update position
	newSize := position.Size - closeSize
	if newSize <= 0 {
		position.Status = "CLOSED"
		now := time.Now()
		position.ClosedAt = &now
	} else {
		// Partial close - recalculate average entry price
		position.EntryPrice = ((position.EntryPrice * position.Size) - (currentPrice * closeSize)) / newSize
		position.Size = newSize
	}

	position.UnrealizedPNL = pnl
	position.RealizedPNL += pnl
	position.Collateral += pnl
	position.MarkPrice = currentPrice

	s.db.Save(&position)

	// Create trade record
	trade := Trade{
		UserAddress: req.UserAddress,
		MarketID:    position.MarketID,
		Symbol:      position.Symbol,
		Side:        position.Side,
		Size:        closeSize,
		Price:       currentPrice,
		Fee:         closeSize * currentPrice * 0.0005, // 0.05% fee
		FeeAsset:    "USDT",
		RealizedPNL: pnl,
		ChainID:     req.ChainID,
		CreatedAt:   time.Now(),
	}
	s.db.Create(&trade)

	// Update market
	market.OpenInterest -= closeSize * market.MarkPrice
	s.db.Save(market)

	ctx.JSON(200, gin.H{
		"success":        true,
		"realized_pnl":   pnl,
		"remaining_size": position.Size,
	})
}

// ============================================================================
// Position Queries
// ============================================================================

type PositionResponse struct {
	ID               uint    `json:"id"`
	Symbol           string  `json:"symbol"`
	Side             string  `json:"side"`
	Size             float64 `json:"size"`
	EntryPrice       float64 `json:"entry_price"`
	MarkPrice        float64 `json:"mark_price"`
	Leverage         float64 `json:"leverage"`
	Collateral       float64 `json:"collateral"`
	UnrealizedPNL    float64 `json:"unrealized_pnl"`
	RealizedPNL      float64 `json:"realized_pnl"`
	LiquidationPrice float64 `json:"liquidation_price"`
	Status           string  `json:"status"`
}

// Get user positions
func (s *PerpetualService) GetPositions(ctx *gin.Context) {
	userAddress := ctx.Query("user_address")
	chainID := ctx.GetInt64("chain_id")

	if userAddress == "" {
		ctx.JSON(400, gin.H{"error": "user_address required"})
		return
	}

	var positions []Position
	s.db.Where("user_address = ? AND chain_id = ? AND status = ?", userAddress, chainID, "OPEN").Find(&positions)

	// Calculate unrealized PnL
	response := make([]PositionResponse, len(positions))
	for i, pos := range positions {
		market, ok := s.markets[pos.Symbol]
		var currentPrice float64
		if ok {
			currentPrice = market.MarkPrice
		} else {
			currentPrice = pos.EntryPrice
		}

		var unrealizedPNL float64
		if pos.Side == "LONG" {
			unrealizedPNL = (currentPrice - pos.EntryPrice) * pos.Size
		} else {
			unrealizedPNL = (pos.EntryPrice - currentPrice) * pos.Size
		}

		response[i] = PositionResponse{
			ID:               pos.ID,
			Symbol:           pos.Symbol,
			Side:             pos.Side,
			Size:             pos.Size,
			EntryPrice:       pos.EntryPrice,
			MarkPrice:        currentPrice,
			Leverage:         pos.Leverage,
			Collateral:       pos.Collateral,
			UnrealizedPNL:    unrealizedPNL,
			RealizedPNL:      pos.RealizedPNL,
			LiquidationPrice: pos.LiquidationPrice,
			Status:           pos.Status,
		}
	}

	ctx.JSON(200, gin.H{"positions": response})
}

// Get markets
func (s *PerpetualService) GetMarkets(ctx *gin.Context) {
	var markets []PerpetualMarket
	s.db.Where("is_active = ?", true).Find(&markets)

	ctx.JSON(200, gin.H{"markets": markets})
}

// Get market by symbol
func (s *PerpetualService) GetMarket(ctx *gin.Context) {
	symbol := ctx.Param("symbol")

	market, ok := s.markets[symbol]
	if !ok {
		ctx.JSON(404, gin.H{"error": "Market not found"})
		return
	}

	ctx.JSON(200, gin.H{"market": market})
}

// ============================================================================
// Funding Rate
// ============================================================================

type FundingRateResponse struct {
	Symbol          string  `json:"symbol"`
	FundingRate     float64 `json:"funding_rate"`
	NextFundingTime int64   `json:"next_funding_time"`
}

func (s *PerpetualService) GetFundingRates(ctx *gin.Context) {
	response := make([]FundingRateResponse, 0, len(s.markets))
	for symbol, market := range s.markets {
		response = append(response, FundingRateResponse{
			Symbol:          symbol,
			FundingRate:     market.FundingRate,
			NextFundingTime: market.NextFundingTime,
		})
	}
	ctx.JSON(200, gin.H{"funding_rates": response})
}

// ============================================================================
// Liquidation Check
// ============================================================================

func (s *PerpetualService) checkLiquidation(position *Position) bool {
	market, ok := s.markets[position.Symbol]
	if !ok {
		return false
	}

	currentPrice := market.MarkPrice
	var marginRatio float64

	if position.Side == "LONG" {
		marginRatio = (position.EntryPrice - currentPrice) / position.EntryPrice
	} else {
		marginRatio = (currentPrice - position.EntryPrice) / position.EntryPrice
	}

	// Add leverage to the ratio
	marginRatio = marginRatio * position.Leverage

	// Check if below maintenance margin
	return marginRatio < market.MaintenanceMargin
}

// ============================================================================
// Helper Functions
// ============================================================================

// ============================================================================
// Main
// ============================================================================

func main() {
	config := Config{
		ServerPort: "8091",
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "tigerwallet"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "tigerwallet_perpetual"),
		RedisHost:  getEnv("REDIS_HOST", "localhost"),
		RedisPort:  getEnv("REDIS_PORT", "6379"),
	}

	service, err := NewPerpetualService(config)
	if err != nil {
		fmt.Printf("Failed to start perpetual service: %v\n", err)
		os.Exit(1)
	}

	router := gin.Default()

	// CORS
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

	// Routes
	api := router.Group("/api/v1/perpetual")
	{
		api.GET("/markets", service.GetMarkets)
		api.GET("/markets/:symbol", service.GetMarket)
		api.GET("/positions", service.GetPositions)
		api.GET("/funding-rates", service.GetFundingRates)
		api.POST("/open-position", service.OpenPosition)
		api.POST("/close-position", service.ClosePosition)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "perpetual"})
	})

	go func() {
		fmt.Printf("Perpetual trading service starting on port %s\n", config.ServerPort)
		if err := router.Run(":" + config.ServerPort); err != nil {
			fmt.Printf("Failed to start server: %v\n", err)
		}
	}()

	// Liquidation check loop
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			var positions []Position
			service.db.Where("status = ?", "OPEN").Find(&positions)
			for i := range positions {
				if service.checkLiquidation(&positions[i]) {
					positions[i].Status = "LIQUIDATED"
					now := time.Now()
					positions[i].ClosedAt = &now
					service.db.Save(&positions[i])
					fmt.Printf("Liquidated position %d for user %s\n", positions[i].ID, positions[i].UserAddress)
				}
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down perpetual service...")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
