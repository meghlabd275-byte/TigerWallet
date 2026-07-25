// TigerWallet Options Trading Service
// High-Load Distributed Go Implementation
// Supports CALL/PUT options with various expiration dates and strike prices

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

// OptionContract represents an options contract
type OptionContract struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Symbol         string    `gorm:"index" json:"symbol"` // e.g., ETH-3000-CALL-2024-03-15
	Name           string    `json:"name"`
	Underlying     string    `json:"underlying"` // ETH, BTC, etc.
	StrikePrice   float64   `json:"strike_price"`
	ExpirationDate time.Time `json:"expiration_date"`
	OptionType    string    `json:"option_type"` // CALL or PUT
	ContractSize  float64   `json:"contract_size"`
	CurrentPrice  float64   `json:"current_price"` // Premium
	OpenInterest  int       `json:"open_interest"`
	Volume24h     int       `json:"volume_24h"`
	IV            float64   `json:"iv"` // Implied Volatility
	Delta         float64   `json:"delta"`
	Gamma         float64   `json:"gamma"`
	Theta         float64   `json:"theta"`
	Vega          float64   `json:"vega"`
	ChainID       int64     `json:"chain_id"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// OptionPosition represents a user's option position
type OptionPosition struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	UserAddress     string    `gorm:"index" json:"user_address"`
	ContractID     uint      `gorm:"index" json:"contract_id"`
	ContractSymbol string    `json:"contract_symbol"`
	Side           string    `json:"side"` // LONG (bought) or SHORT (sold)
	Quantity       int       `json:"quantity"`
	EntryPrice    float64   `json:"entry_price"`
	CurrentPrice  float64   `json:"current_price"`
	UnrealizedPNL float64   `json:"unrealized_pnl"`
	RealizedPNL   float64   `json:"realized_pnl"`
	Status        string    `json:"status"` // OPEN, EXERCISED, EXPIRED, CLOSED
	ChainID       int64     `json:"chain_id"`
	OpenedAt      time.Time `json:"opened_at"`
	ClosedAt      *time.Time `json:"closed_at"`
}

// OptionOrder represents an order
type OptionOrder struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserAddress   string    `gorm:"index" json:"user_address"`
	ContractID   uint      `gorm:"index" json:"contract_id"`
	ContractSymbol string  `json:"contract_symbol"`
	Side         string    `json:"side"` // BUY or SELL
	OrderType    string    `json:"order_type"` // MARKET, LIMIT
	Price        float64   `json:"price"`
	Quantity     int       `json:"quantity"`
	FilledQty    int       `json:"filled_qty"`
	AvgFillPrice float64   `json:"avg_fill_price"`
	Status       string    `json:"status"` // PENDING, FILLED, PARTIAL, CANCELLED
	ChainID      int64     `json:"chain_id"`
	CreatedAt    time.Time `json:"created_at"`
	FilledAt     *time.Time `json:"filled_at"`
}

// OptionTrade represents a filled trade
type OptionTrade struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	OrderID        uint      `json:"order_id"`
	UserAddress    string    `json:"user_address"`
	ContractID     uint      `json:"contract_id"`
	ContractSymbol string    `json:"contract_symbol"`
	Side           string    `json:"side"`
	Quantity       int       `json:"quantity"`
	Price          float64   `json:"price"`
	Premium        float64   `json:"premium"` // Total premium paid/received
	Fee            float64   `json:"fee"`
	ChainID        int64     `json:"chain_id"`
	CreatedAt      time.Time `json:"created_at"`
}

// ExerciseEvent represents option exercise
type ExerciseEvent struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	PositionID     uint      `json:"position_id"`
	UserAddress    string    `json:"user_address"`
	ContractSymbol string    `json:"contract_symbol"`
	Quantity       int       `json:"quantity"`
	StrikePrice    float64   `json:"strike_price"`
	SettlementPrice float64  `json:"settlement_price"`
	Profit         float64   `json:"profit"`
	ChainID        int64     `json:"chain_id"`
	ExercisedAt    time.Time `json:"exercised_at"`
}

// ============================================================================
// Service Implementation
// ============================================================================

type OptionsService struct {
	db     *gorm.DB
	redis *redis.Client
	config Config
	mu     sync.RWMutex
	prices map[string]float64
}

func NewOptionsService(config Config) (*OptionsService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	err = db.AutoMigrate(
		&OptionContract{},
		&OptionPosition{},
		&OptionOrder{},
		&OptionTrade{},
		&ExerciseEvent{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort),
	})

	service := &OptionsService{
		db:     db,
		redis:  rdb,
		config: config,
		prices: make(map[string]float64),
	}

	go service.initializeOptions()
	go service.startPriceFeed()

	return service, nil
}

// Initialize default options contracts
func (s *OptionsService) initializeOptions() {
	expirations := []time.Time{
		time.Now().Add(24 * time.Hour),
		time.Now().Add(7 * 24 * time.Hour),
		time.Now().Add(30 * 24 * time.Hour),
		time.Now().Add(90 * 24 * time.Hour),
	}

	strikePrices := map[string][]float64{
		"ETH": {2500, 2750, 3000, 3250, 3500, 3750, 4000},
		"BTC": {50000, 55000, 60000, 65000, 70000, 75000, 80000},
		"SOL": {100, 125, 150, 175, 200, 225, 250},
	}

	underlyingPrices := map[string]float64{
		"ETH": 3500,
		"BTC": 65000,
		"SOL": 150,
	}

	for underlying, strikes := range strikePrices {
		basePrice := underlyingPrices[underlying]
		
		for _, expiry := range expirations {
			for _, strike := range strikes {
				// Create CALL option
				callSymbol := fmt.Sprintf("%s-%.0f-CALL-%s", underlying, strike, expiry.Format("2006-01-02"))
				var existing OptionContract
				if s.db.Where("symbol = ?", callSymbol).First(&existing).RowsAffected == 0 {
					callOption := OptionContract{
						Symbol:         callSymbol,
						Name:           callSymbol,
						Underlying:     underlying,
						StrikePrice:    strike,
						ExpirationDate: expiry,
						OptionType:     "CALL",
						ContractSize:   1,
						CurrentPrice:   s.calculateOptionPrice(basePrice, strike, 30, expiry, "CALL"),
						IV:             30 + math.Random()*20,
						ChainID:        1,
						IsActive:       true,
					}
					s.db.Create(&callOption)
				}

				// Create PUT option
				putSymbol := fmt.Sprintf("%s-%.0f-PUT-%s", underlying, strike, expiry.Format("2006-01-02"))
				if s.db.Where("symbol = ?", putSymbol).First(&existing).RowsAffected == 0 {
					putOption := OptionContract{
						Symbol:         putSymbol,
						Name:           putSymbol,
						Underlying:     underlying,
						StrikePrice:    strike,
						ExpirationDate: expiry,
						OptionType:     "PUT",
						ContractSize:   1,
						CurrentPrice:   s.calculateOptionPrice(basePrice, strike, 30, expiry, "PUT"),
						IV:             30 + math.Random()*20,
						ChainID:        1,
						IsActive:       true,
					}
					s.db.Create(&putOption)
				}
			}
		}
	}
}

// Black-Scholes option pricing
func (s *OptionsService) calculateOptionPrice(spotPrice, strikePrice float64, iv float64, expiration time.Time, optionType string) float64 {
	daysToExpiry := expiration.Sub(time.Now()).Hours() / 24
	if daysToExpiry <= 0 {
		return 0
	}

	timeToExpiry := daysToExpiry / 365.0
	ivDecimal := iv / 100.0

	// Simplified Black-Scholes
	d1 := (math.Log(spotPrice/strikePrice) + (0.05+ivDecimal*ivDecimal/2)*timeToExpiry) / (ivDecimal * math.Sqrt(timeToExpiry))
	d2 := d1 - ivDecimal*math.Sqrt(timeToExpiry)

	var price float64
	if optionType == "CALL" {
		price = spotPrice*normalCDF(d1) - strikePrice*math.Exp(-0.05*timeToExpiry)*normalCDF(d2)
	} else {
		price = strikePrice*math.Exp(-0.05*timeToExpiry)*normalCDF(-d2) - spotPrice*normalCDF(-d1)
	}

	// Minimum price of $0.01
	if price < 0.01 {
		price = 0.01
	}

	return price
}

func normalCDF(x float64) float64 {
	return 0.5 * (1 + math.Erf(x/math.Sqrt2))
}

// Start price feed simulation
func (s *OptionsService) startPriceFeed() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	underlyingPrices := map[string]float64{
		"ETH": 3500,
		"BTC": 65000,
		"SOL": 150,
	}

	for range ticker.C {
		for underlying, basePrice := range underlyingPrices {
			// Random walk
			change := (math.Random() - 0.5) * basePrice * 0.02
			newPrice := basePrice + change

			// Update option prices based on new underlying
			s.updateOptionPrices(underlying, newPrice)
		}
	}
}

func (s *OptionsService) updateOptionPrices(underlying string, spotPrice float64) {
	var contracts []OptionContract
	s.db.Where("underlying = ? AND is_active = ?", underlying, true).Find(&contracts)

	for i := range contracts {
		contract := &contracts[i]
		daysToExpiry := contract.ExpirationDate.Sub(time.Now()).Hours() / 24
		if daysToExpiry <= 0 {
			contract.IsActive = false
			s.db.Save(contract)
			continue
		}

		contract.CurrentPrice = s.calculateOptionPrice(
			spotPrice,
			contract.StrikePrice,
			contract.IV,
			contract.ExpirationDate,
			contract.OptionType,
		)

		// Calculate Greeks
		s.calculateGreeks(contract, spotPrice)

		s.db.Save(contract)
	}
}

func (s *OptionsService) calculateGreeks(contract *OptionContract, spotPrice float64) {
	daysToExpiry := contract.ExpirationDate.Sub(time.Now()).Hours() / 24
	timeToExpiry := daysToExpiry / 365.0
	iv := contract.IV / 100.0

	d1 := (math.Log(spotPrice/contract.StrikePrice) + (0.05+iv*iv/2)*timeToExpiry) / (iv * math.Sqrt(timeToExpiry))
	d2 := d1 - iv*math.Sqrt(timeToExpiry)

	contract.Delta = normalCDF(d1)
	if contract.OptionType == "PUT" {
		contract.Delta -= 1
	}

	// Gamma (same for calls and puts)
	contract.Gamma = normalCDF(d1) / (spotPrice * iv * math.Sqrt(timeToExpiry))

	// Theta
	thetaBase := -spotPrice*normalCDF(d1)*iv / (2*math.Sqrt(timeToExpiry))
	if contract.OptionType == "CALL" {
		contract.Theta = (thetaBase - 0.05*contract.StrikePrice*math.Exp(-0.05*timeToExpiry)*normalCDF(d2)) / 365
	} else {
		contract.Theta = (thetaBase + 0.05*contract.StrikePrice*math.Exp(-0.05*timeToExpiry)*normalCDF(-d2)) / 365
	}

	// Vega
	contract.Vega = spotPrice * math.Sqrt(timeToExpiry) * normalCDF(d1) / 100
}

// ============================================================================
// Trading Operations
// ============================================================================

type OpenPositionRequest struct {
	UserAddress   string  `json:"user_address" binding:"required"`
	ContractID   uint    `json:"contract_id" binding:"required"`
	Side         string  `json:"side" binding:"required"` // LONG or SHORT
	Quantity     int     `json:"quantity" binding:"required"`
	OrderType    string  `json:"order_type"` // MARKET or LIMIT
	Price        float64 `json:"price"`
	ChainID      int64   `json:"chain_id"`
}

func (s *OptionsService) OpenPosition(ctx *gin.Context) {
	var req OpenPositionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	var contract OptionContract
	if err := s.db.First(&contract, req.ContractID).Error; err != nil {
		ctx.JSON(404, gin.H{"success": false, "error": "Contract not found"})
		return
	}

	if !contract.IsActive {
		ctx.JSON(400, gin.H{"success": false, "error": "Contract is not active"})
		return
	}

	// Check expiration
	if time.Now().After(contract.ExpirationDate) {
		ctx.JSON(400, gin.H{"success": false, "error": "Contract has expired"})
		return
	}

	// Get execution price
	execPrice := contract.CurrentPrice
	if req.OrderType == "LIMIT" && req.Price > 0 {
		execPrice = req.Price
	}

	premium := execPrice * float64(req.Quantity) * contract.ContractSize

	// Create position
	position := OptionPosition{
		UserAddress:     req.UserAddress,
		ContractID:     contract.ID,
		ContractSymbol: contract.Symbol,
		Side:           req.Side,
		Quantity:       req.Quantity,
		EntryPrice:    execPrice,
		CurrentPrice:  execPrice,
		UnrealizedPNL: 0,
		RealizedPNL:   0,
		Status:         "OPEN",
		ChainID:        req.ChainID,
		OpenedAt:       time.Now(),
	}

	if err := s.db.Create(&position).Error; err != nil {
		ctx.JSON(500, gin.H{"success": false, "error": "Failed to open position"})
		return
	}

	// Create trade record
	trade := OptionTrade{
		OrderID:        0,
		UserAddress:    req.UserAddress,
		ContractID:     contract.ID,
		ContractSymbol: contract.Symbol,
		Side:           req.Side,
		Quantity:       req.Quantity,
		Price:          execPrice,
		Premium:        premium,
		Fee:            premium * 0.001, // 0.1% fee
		ChainID:        req.ChainID,
		CreatedAt:      time.Now(),
	}
	s.db.Create(&trade)

	// Update contract volume
	contract.Volume24h += req.Quantity
	contract.OpenInterest += req.Quantity
	s.db.Save(&contract)

	ctx.JSON(200, gin.H{
		"success":        true,
		"position_id":    position.ID,
		"symbol":        contract.Symbol,
		"quantity":      req.Quantity,
		"entry_price":   execPrice,
		"premium":       premium,
		"fee":           trade.Fee,
	})
}

type ClosePositionRequest struct {
	UserAddress string  `json:"user_address" binding:"required"`
	PositionID uint    `json:"position_id" binding:"required"`
	Quantity   int     `json:"quantity"`
	ChainID    int64   `json:"chain_id"`
}

func (s *OptionsService) ClosePosition(ctx *gin.Context) {
	var req ClosePositionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	var position OptionPosition
	if err := s.db.First(&position, req.PositionID).Error; err != nil {
		ctx.JSON(404, gin.H{"success": false, "error": "Position not found"})
		return
	}

	if position.UserAddress != req.UserAddress {
		ctx.JSON(403, gin.H{"success": false, "error": "Unauthorized"})
		return
	}

	if position.Status != "OPEN" {
		ctx.JSON(400, gin.H{"success": false, "error": "Position is not open"})
		return
	}

	var contract OptionContract
	if err := s.db.First(&contract, position.ContractID).Error; err != nil {
		ctx.JSON(404, gin.H{"success": false, "error": "Contract not found"})
		return
	}

	closeQty := req.Quantity
	if closeQty == 0 || closeQty > position.Quantity {
		closeQty = position.Quantity
	}

	// Calculate PnL
	var pnl float64
	entryPremium := position.EntryPrice * float64(closeQty) * contract.ContractSize
	exitPremium := contract.CurrentPrice * float64(closeQty) * contract.ContractSize

	if position.Side == "LONG" {
		pnl = exitPremium - entryPremium
	} else {
		pnl = entryPremium - exitPremium
	}

	// Update position
	position.Quantity -= closeQty
	position.UnrealizedPNL = pnl
	position.RealizedPNL += pnl

	if position.Quantity <= 0 {
		position.Status = "CLOSED"
		now := time.Now()
		position.ClosedAt = &now
	}

	s.db.Save(&position)

	// Update contract open interest
	contract.OpenInterest -= closeQty
	s.db.Save(&contract)

	ctx.JSON(200, gin.H{
		"success":       true,
		"realized_pnl": pnl,
		"remaining_qty": position.Quantity,
	})
}

// Exercise option
type ExerciseRequest struct {
	UserAddress string `json:"user_address" binding:"required"`
	PositionID uint   `json:"position_id" binding:"required"`
	Quantity   int    `json:"quantity"`
}

func (s *OptionsService) ExercisePosition(ctx *gin.Context) {
	var req ExerciseRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	var position OptionPosition
	if err := s.db.First(&position, req.PositionID).Error; err != nil {
		ctx.JSON(404, gin.H{"success": false, "error": "Position not found"})
		return
	}

	if position.UserAddress != req.UserAddress {
		ctx.JSON(403, gin.H{"success": false, "error": "Unauthorized"})
		return
	}

	var contract OptionContract
	s.db.First(&contract, position.ContractID)

	// Get settlement price (simulated)
	settlementPrice := s.prices[contract.Underlying]
	if settlementPrice == 0 {
		settlementPrice = contract.StrikePrice
	}

	exerciseQty := req.Quantity
	if exerciseQty == 0 || exerciseQty > position.Quantity {
		exerciseQty = position.Quantity
	}

	// Calculate exercise profit
	var profit float64
	if contract.OptionType == "CALL" {
		if settlementPrice > contract.StrikePrice {
			profit = (settlementPrice - contract.StrikePrice) * float64(exerciseQty) * contract.ContractSize
		}
	} else {
		if settlementPrice < contract.StrikePrice {
			profit = (contract.StrikePrice - settlementPrice) * float64(exerciseQty) * contract.ContractSize
		}
	}

	// Record exercise event
	exercise := ExerciseEvent{
		PositionID:       position.ID,
		UserAddress:      req.UserAddress,
		ContractSymbol:  contract.Symbol,
		Quantity:        exerciseQty,
		StrikePrice:     contract.StrikePrice,
		SettlementPrice: settlementPrice,
		Profit:          profit,
		ChainID:         position.ChainID,
		ExercisedAt:     time.Now(),
	}
	s.db.Create(&exercise)

	// Update position
	position.Quantity -= exerciseQty
	position.RealizedPNL += profit

	if position.Quantity <= 0 {
		position.Status = "EXERCISED"
		now := time.Now()
		position.ClosedAt = &now
	}

	s.db.Save(&position)

	// Update contract
	contract.OpenInterest -= exerciseQty
	s.db.Save(&contract)

	ctx.JSON(200, gin.H{
		"success":         true,
		"exercise_id":    exercise.ID,
		"profit":         profit,
		"settlement_price": settlementPrice,
	})
}

// ============================================================================
// Queries
// ============================================================================

func (s *OptionsService) GetContracts(ctx *gin.Context) {
	underlying := ctx.Query("underlying")
	optionType := ctx.Query("type")
	expiration := ctx.Query("expiration")

	query := s.db.Where("is_active = ?", true)

	if underlying != "" {
		query = query.Where("underlying = ?", underlying)
	}
	if optionType != "" {
		query = query.Where("option_type = ?", optionType)
	}
	if expiration != "" {
		expDate, _ := time.Parse("2006-01-02", expiration)
		query = query.Where("expiration_date >= ?", expDate)
	}

	var contracts []OptionContract
	query.Order("expiration_date, strike_price").Find(&contracts)

	ctx.JSON(200, gin.H{"contracts": contracts})
}

func (s *OptionsService) GetContract(ctx *gin.Context) {
	symbol := ctx.Param("symbol")

	var contract OptionContract
	if err := s.db.Where("symbol = ?", symbol).First(&contract).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "Contract not found"})
		return
	}

	ctx.JSON(200, gin.H{"contract": contract})
}

func (s *OptionsService) GetPositions(ctx *gin.Context) {
	userAddress := ctx.Query("user_address")
	chainID := ctx.GetInt64("chain_id")

	if userAddress == "" {
		ctx.JSON(400, gin.H{"error": "user_address required"})
		return
	}

	var positions []OptionPosition
	s.db.Where("user_address = ? AND chain_id = ? AND status = ?", userAddress, chainID, "OPEN").Find(&positions)

	// Update unrealized PnL
	response := make([]map[string]interface{}, len(positions))
	for i, pos := range positions {
		var contract OptionContract
		s.db.First(&contract, pos.ContractID)

		var pnl float64
		if pos.Side == "LONG" {
			pnl = (contract.CurrentPrice - pos.EntryPrice) * float64(pos.Quantity) * contract.ContractSize
		} else {
			pnl = (pos.EntryPrice - contract.CurrentPrice) * float64(pos.Quantity) * contract.ContractSize
		}

		response[i] = map[string]interface{}{
			"id":               pos.ID,
			"contract_symbol":  pos.ContractSymbol,
			"side":            pos.Side,
			"quantity":        pos.Quantity,
			"entry_price":     pos.EntryPrice,
			"current_price":   contract.CurrentPrice,
			"unrealized_pnl":  pnl,
			"realized_pnl":    pos.RealizedPNL,
			"status":          pos.Status,
		}
	}

	ctx.JSON(200, gin.H{"positions": response})
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := Config{
		ServerPort: "8095",
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "tigerwallet"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "tigerwallet_options"),
		RedisHost:  getEnv("REDIS_HOST", "localhost"),
		RedisPort:  getEnv("REDIS_PORT", "6379"),
	}

	service, err := NewOptionsService(config)
	if err != nil {
		fmt.Printf("Failed to start options service: %v\n", err)
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

	api := router.Group("/api/v1/options")
	{
		api.GET("/contracts", service.GetContracts)
		api.GET("/contracts/:symbol", service.GetContract)
		api.GET("/positions", service.GetPositions)
		api.POST("/open", service.OpenPosition)
		api.POST("/close", service.ClosePosition)
		api.POST("/exercise", service.ExercisePosition)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "options"})
	})

	go func() {
		fmt.Printf("Options trading service starting on port %s\n", config.ServerPort)
		if err := router.Run(":" + config.ServerPort); err != nil {
			fmt.Printf("Failed to start server: %v\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down options service...")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func generateOfferID(contract, token, seller string) string {
	data := fmt.Sprintf("%s:%s:%s:%d", contract, token, seller, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return "offer_" + hex.EncodeToString(hash[:])[:16]
}
