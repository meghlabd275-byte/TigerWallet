/**
 * TigerWallet Options Trading Service
 * Complete Options Trading Platform
 * 
 * Features:
 * - Call/Put options
 * - American/European style
 * - Expiration management
 * - Strike price management
 * - Premium calculation
 * - Exercise/Settlement
 * - Greeks calculation
 * - Open interest tracking
 * - Position management
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ServerPort  string
	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string
	RedisHost   string
	RedisPort   string
}

func LoadConfig() *Config {
	return &Config{
		ServerPort: getEnv("OPTIONS_PORT", "9097"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "tigerwallet"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "tigerwallet"),
		RedisHost:  getEnv("REDIS_HOST", "localhost"),
		RedisPort:  getEnv("REDIS_PORT", "6379"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============================================================================
// Database Models
// ============================================================================

type OptionContract struct {
	ID                uint           `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
	ContractID        string         `gorm:"uniqueIndex;size:36" json:"contract_id"`
	UnderlyingAsset   string         `json:"underlying_asset"` // BTC, ETH, etc.
	StrikePrice       float64        `json:"strike_price"`
	ExpirationDate    time.Time      `json:"expiration_date"`
	OptionType        string         `json:"option_type"` // call, put
	ExerciseStyle     string         `json:"exercise_style"` // american, european
	ContractSize      int            `json:"contract_size"` // Usually 1 for crypto
	IsActive          bool           `json:"is_active"`
	OpenInterest      int            `json:"open_interest"`
	Volume24h         int            `json:"volume_24h"`
	CurrentPremium    float64        `json:"current_premium"`
	ImpliedVolatility float64        `json:"implied_volatility"`
	Delta             float64        `json:"delta"`
	Gamma             float64        `json:"gamma"`
	Theta             float64        `json:"theta"`
	Vega              float64        `json:"vega"`
	 Rho              float64        `json:"rho"`
}

type OptionPosition struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	UserID            uint      `gorm:"index" json:"user_id"`
	ContractID        string    `gorm:"index" json:"contract_id"`
	Contract          OptionContract `gorm:"foreignKey:ContractID" json:"-"`
	PositionType      string    `json:"position_type"` // long, short
	Quantity          int       `json:"quantity"`
	EntryPrice        float64   `json:"entry_price"`
	CurrentPrice      float64   `json:"current_price"`
	UnrealizedPL      float64   `json:"unrealized_pl"`
	MarginRequired    float64   `json:"margin_required"`
	LiquidationPrice  float64   `json:"liquidation_price"`
	LastUpdated       time.Time `json:"last_updated"`
}

type OptionOrder struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	OrderID           string    `gorm:"uniqueIndex;size:36" json:"order_id"`
	UserID            uint      `gorm:"index" json:"user_id"`
	ContractID        string    `gorm:"index" json:"contract_id"`
	OrderType         string    `json:"order_type"` // market, limit
	Side              string    `json:"side"` // buy, sell
	PositionType      string    `json:"position_type"` // long, short
	Quantity          int       `json:"quantity"`
	LimitPrice        float64   `json:"limit_price"`
	FilledQuantity    int       `json:"filled_quantity"`
	AverageFillPrice  float64   `json:"average_fill_price"`
	Status            string    `json:"status"` // pending, partial, filled, cancelled, expired
	ExpiresAt         time.Time `json:"expires_at"`
	FilledAt          *time.Time `json:"filled_at"`
	CancelledAt       *time.Time `json:"cancelled_at"`
	ErrorMessage      string    `json:"error_message"`
}

type OptionTrade struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	TradeID           string    `gorm:"uniqueIndex;size:36" json:"trade_id"`
	OrderID           string    `gorm:"index" json:"order_id"`
	UserID            uint      `gorm:"index" json:"user_id"`
	ContractID        string    `gorm:"index" json:"contract_id"`
	Side              string    `json:"side"` // buy, sell
	PositionType      string    `json:"position_type"`
	Quantity          int       `json:"quantity"`
	Price             float64   `json:"price"`
	Total             float64   `json:"total"`
	Fee               float64   `json:"fee"`
	TxHash            string    `gorm:"uniqueIndex;size:66" json:"tx_hash"`
}

type OptionSettlement struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	SettlementID      string    `gorm:"uniqueIndex;size:36" json:"settlement_id"`
	ContractID        string    `gorm:"index" json:"contract_id"`
	UserID            uint      `gorm:"index" json:"user_id"`
	PositionID        uint      `gorm:"index" json:"position_id"`
	SettlementType    string    `json:"settlement_type"` // exercise, assign, expire, settle
	Quantity          int       `json:"quantity"`
	SettlementPrice   float64   `json:"settlement_price"`
	SettlementAmount  float64   `json:"settlement_amount"`
	Profit            float64   `json:"profit"`
	TxHash            string    `gorm:"uniqueIndex;size:66" json:"tx_hash"`
	Status            string    `json:"status"` // pending, completed, failed
}

// ============================================================================
// Black-Scholes Model
// ============================================================================

func BlackScholesCall(S, K, T, r, sigma float64) float64 {
	// S: Current stock price
	// K: Strike price
	// T: Time to expiration (in years)
	// r: Risk-free rate
	// sigma: Volatility
	
	d1 := (math.Log(S/K) + (r + sigma*sigma/2)*T) / (sigma * math.Sqrt(T))
	d2 := d1 - sigma*math.Sqrt(T)
	
	callPrice := S * NormalCDF(d1) - K * math.Exp(-r*T) * NormalCDF(d2)
	
	return callPrice
}

func BlackScholesPut(S, K, T, r, sigma float64) float64 {
	d1 := (math.Log(S/K) + (r + sigma*sigma/2)*T) / (sigma * math.Sqrt(T))
	d2 := d1 - sigma * math.Sqrt(T)
	
	putPrice := K * math.Exp(-r*T) * NormalCDF(-d2) - S * NormalCDF(-d1)
	
	return putPrice
}

func NormalCDF(x float64) float64 {
	return 0.5 * (1 + math.Erf(x/math.Sqrt(2)))
}

func NormalPDF(x float64) float64 {
	return math.Exp(-0.5*x*x) / math.Sqrt(2*math.Pi)
}

// Calculate Greeks
func CalculateGreeks(S, K, T, r, sigma float64, optionType string) (delta, gamma, theta, vega, rho float64) {
	d1 := (math.Log(S/K) + (r + sigma*sigma/2)*T) / (sigma * math.Sqrt(T))
	d2 := d1 - sigma*math.Sqrt(T)
	
	sqrtT := math.Sqrt(T)
	nd1 := NormalCDF(d1)
	nd2 := NormalCDF(d2)
	n_d1 := NormalPDF(d1)
	
	// Delta
	if optionType == "call" {
		delta = nd1
	} else {
		delta = nd1 - 1
	}
	
	// Gamma (same for call and put)
	gamma = n_d1 / (S * sigma * sqrtT)
	
	// Theta (annual)
	if optionType == "call" {
		theta = - (S * n_d1 * sigma / (2 * sqrtT)) - r * K * math.Exp(-r*T) * nd2
	} else {
		theta = - (S * n_d1 * sigma / (2 * sqrtT)) + r * K * math.Exp(-r*T) * NormalCDF(-d2)
	}
	theta = theta / 365 // Daily theta
	
	// Vega (same for call and put)
	vega = S * sqrtT * n_d1 / 100
	
	// Rho
	if optionType == "call" {
		rho = K * T * math.Exp(-r*T) * nd2 / 100
	} else {
		rho = -K * T * math.Exp(-r*T) * NormalCDF(-d2) / 100
	}
	
	return
}

// ============================================================================
// Options Service
// ============================================================================

type OptionsService struct {
	config     *Config
	db         *gorm.DB
	redis      *redis.Client
	contracts  map[string]OptionContract
	positions  map[uint][]OptionPosition
	mu         sync.RWMutex
}

func NewOptionsService(cfg *Config) (*OptionsService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	
	db.AutoMigrate(&OptionContract{}, &OptionPosition{}, &OptionOrder{}, &OptionTrade{}, &OptionSettlement{})
	
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: "",
		DB: 0,
	})
	
	service := &OptionsService{
		config:    cfg,
		db:        db,
		redis:     rdb,
		contracts: make(map[string]OptionContract),
		positions: make(map[uint][]OptionPosition),
	}
	
	service.loadContracts()
	
	return service, nil
}

func (s *OptionsService) loadContracts() {
	var contracts []OptionContract
	s.db.Where("is_active = ?", true).Find(&contracts)
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	for _, contract := range contracts {
		s.contracts[contract.ContractID] = contract
	}
}

// ============================================================================
// Contract Management
// ============================================================================

func (s *OptionsService) CreateOptionContract(
	underlying string,
	strikePrice float64,
	expiration time.Time,
	optionType string,
	exerciseStyle string,
) (*OptionContract, error) {
	
	contractID := fmt.Sprintf("%s-%s-%.0f-%s", 
		underlying, 
		optionType, 
		strikePrice, 
		expiration.Format("2006-01-02"))
	
	// Calculate initial premium using Black-Scholes
	// In production, get actual spot price and volatility
	currentPrice := strikePrice // Simplified
	timeToExpiry := expiration.Sub(time.Now()).Hours() / (24 * 365)
	volatility := 0.5 // 50% IV
	riskFreeRate := 0.05
	
	var premium float64
	if optionType == "call" {
		premium = BlackScholesCall(currentPrice, strikePrice, timeToExpiry, riskFreeRate, volatility)
	} else {
		premium = BlackScholesPut(currentPrice, strikePrice, timeToExpiry, riskFreeRate, volatility)
	}
	
	// Calculate Greeks
	delta, gamma, theta, vega, rho := CalculateGreeks(currentPrice, strikePrice, timeToExpiry, riskFreeRate, volatility, optionType)
	
	contract := OptionContract{
		ContractID:         contractID,
		UnderlyingAsset:     underlying,
		StrikePrice:         strikePrice,
		ExpirationDate:      expiration,
		OptionType:          optionType,
		ExerciseStyle:       exerciseStyle,
		ContractSize:        1,
		IsActive:            true,
		OpenInterest:        0,
		Volume24h:           0,
		CurrentPremium:      premium,
		ImpliedVolatility:   volatility,
		Delta:               delta,
		Gamma:               gamma,
		Theta:               theta,
		Vega:               vega,
		Rho:                rho,
	}
	
	s.db.Create(&contract)
	
	s.mu.Lock()
	s.contracts[contractID] = contract
	s.mu.Unlock()
	
	return &contract, nil
}

func (s *OptionsService) GetOptionContracts(underlying string) ([]OptionContract, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	contracts := make([]OptionContract, 0)
	for _, c := range s.contracts {
		if c.UnderlyingAsset == underlying && c.IsActive {
			contracts = append(contracts, c)
		}
	}
	
	// Sort by expiration
	sort.Slice(contracts, func(i, j int) bool {
		return contracts[i].ExpirationDate.Before(contracts[j].ExpirationDate)
	})
	
	return contracts, nil
}

func (s *OptionsService) GetOptionChain(underlying string) (map[string][]OptionContract, error) {
	contracts, err := s.GetOptionContracts(underlying)
	if err != nil {
		return nil, err
	}
	
	// Group by expiration date
	chain := make(map[string][]OptionContract)
	for _, c := range contracts {
		expStr := c.ExpirationDate.Format("2006-01-02")
		chain[expStr] = append(chain[expStr], c)
	}
	
	return chain, nil
}

// ============================================================================
// Order Management
// ============================================================================

func (s *OptionsService) PlaceOrder(
	userID uint,
	contractID string,
	orderType string,
	side string,
	positionType string,
	quantity int,
	limitPrice float64,
) (*OptionOrder, error) {
	
	// Get contract
	s.mu.RLock()
	contract, ok := s.contracts[contractID]
	s.mu.RUnlock()
	
	if !ok {
		return nil, fmt.Errorf("contract not found: %s", contractID)
	}
	
	orderID := uuid.New().String()
	
	order := OptionOrder{
		OrderID:       orderID,
		UserID:        userID,
		ContractID:    contractID,
		OrderType:     orderType,
		Side:          side,
		PositionType:  positionType,
		Quantity:      quantity,
		LimitPrice:    limitPrice,
		Status:        "pending",
		ExpiresAt:     time.Now().Add(24 * time.Hour),
	}
	
	s.db.Create(&order)
	
	// Execute market order immediately
	if orderType == "market" {
		s.executeOrder(&order, &contract)
	}
	
	return &order, nil
}

func (s *OptionsService) executeOrder(order *OptionOrder, contract *OptionContract) error {
	var fillPrice float64
	
	if order.OrderType == "market" {
		fillPrice = contract.CurrentPremium
	} else {
		fillPrice = order.LimitPrice
	}
	
	order.FilledQuantity = order.Quantity
	order.AverageFillPrice = fillPrice
	order.Status = "filled"
	now := time.Now()
	order.FilledAt = &now
	
	s.db.Save(order)
	
	// Create position
	position := OptionPosition{
		UserID:         order.UserID,
		ContractID:     order.ContractID,
		PositionType:   order.PositionType,
		Quantity:       order.Quantity,
		EntryPrice:     fillPrice,
		CurrentPrice:   fillPrice,
		MarginRequired: s.calculateMarginRequired(contract, order.Quantity, order.PositionType),
		LastUpdated:    time.Now(),
	}
	s.db.Create(&position)
	
	// Update contract open interest
	if order.Side == "buy" {
		contract.OpenInterest += order.Quantity
	} else {
		contract.OpenInterest -= order.Quantity
	}
	contract.Volume24h += order.Quantity
	s.db.Save(contract)
	
	// Create trade record
	trade := OptionTrade{
		TradeID:         uuid.New().String(),
		OrderID:         order.OrderID,
		UserID:          order.UserID,
		ContractID:      order.ContractID,
		Side:            order.Side,
		PositionType:    order.PositionType,
		Quantity:        order.Quantity,
		Price:           fillPrice,
		Total:           fillPrice * float64(order.Quantity),
		Fee:             fillPrice * float64(order.Quantity) * 0.001, // 0.1% fee
		TxHash:          generateTxHash(),
	}
	s.db.Create(&trade)
	
	return nil
}

func (s *OptionsService) calculateMarginRequired(contract *OptionContract, quantity int, positionType string) float64 {
	// Simplified margin calculation
	// In production, use proper risk-based margin
	
	marginPerContract := contract.CurrentPremium * float64(contract.ContractSize)
	
	if positionType == "short" {
		// Short positions require margin
		marginPerContract += contract.StrikePrice * 0.1 // 10% of strike
	}
	
	return marginPerContract * float64(quantity)
}

// ============================================================================
// Position Management
// ============================================================================

func (s *OptionsService) GetUserPositions(userID uint) ([]OptionPosition, error) {
	var positions []OptionPosition
	s.db.Where("user_id = ?", userID).Find(&positions)
	
	// Update current prices and P/L
	for i := range positions {
		s.mu.RLock()
		contract, ok := s.contracts[positions[i].ContractID]
		s.mu.RUnlock()
		
		if ok {
			positions[i].CurrentPrice = contract.CurrentPremium
			positions[i].UnrealizedPL = (positions[i].CurrentPrice - positions[i].EntryPrice) * float64(positions[i].Quantity)
			s.db.Save(&positions[i])
		}
	}
	
	return positions, nil
}

func (s *OptionsService) ClosePosition(userID uint, positionID uint, quantity int) (*OptionTrade, error) {
	var position OptionPosition
	if err := s.db.First(&position, positionID).Error; err != nil {
		return nil, fmt.Errorf("position not found")
	}
	
	if position.UserID != userID {
		return nil, fmt.Errorf("unauthorized")
	}
	
	if quantity > position.Quantity {
		quantity = position.Quantity
	}
	
	// Get contract
	s.mu.RLock()
	contract := s.contracts[position.ContractID]
	s.mu.RUnlock()
	
	// Calculate close price
	closePrice := contract.CurrentPremium
	side := "sell"
	if position.PositionType == "short" {
		side = "buy"
	}
	
	// Create closing trade
	trade := OptionTrade{
		TradeID:         uuid.New().String(),
		UserID:          userID,
		ContractID:      position.ContractID,
		Side:            side,
		PositionType:    position.PositionType,
		Quantity:        quantity,
		Price:           closePrice,
		Total:           closePrice * float64(quantity),
		Fee:             closePrice * float64(quantity) * 0.001,
		TxHash:          generateTxHash(),
	}
	s.db.Create(&trade)
	
	// Update position
	position.Quantity -= quantity
	position.CurrentPrice = closePrice
	position.UnrealizedPL = (position.CurrentPrice - position.EntryPrice) * float64(position.Quantity)
	position.LastUpdated = time.Now()
	
	if position.Quantity == 0 {
		s.db.Delete(&position)
	} else {
		s.db.Save(&position)
	}
	
	// Update contract open interest
	contract.OpenInterest -= quantity
	s.db.Save(&contract)
	
	return &trade, nil
}

// ============================================================================
// Exercise & Settlement
// ============================================================================

func (s *OptionsService) ExerciseOption(userID uint, positionID uint) (*OptionSettlement, error) {
	var position OptionPosition
	if err := s.db.First(&position, positionID).Error; err != nil {
		return nil, fmt.Errorf("position not found")
	}
	
	if position.UserID != userID {
		return nil, fmt.Errorf("unauthorized")
	}
	
	s.mu.RLock()
	contract := s.contracts[position.ContractID]
	s.mu.RUnlock()
	
	// Get current underlying price (simplified)
	underlyingPrice := contract.StrikePrice * 1.1 // Mock price
	
	var settlementAmount float64
	if contract.OptionType == "call" {
		settlementAmount = (underlyingPrice - contract.StrikePrice) * float64(position.Quantity)
	} else {
		settlementAmount = (contract.StrikePrice - underlyingPrice) * float64(position.Quantity)
	}
	
	if settlementAmount < 0 {
		settlementAmount = 0
	}
	
	settlement := OptionSettlement{
		SettlementID:      uuid.New().String(),
		ContractID:         position.ContractID,
		UserID:             userID,
		PositionID:         positionID,
		SettlementType:     "exercise",
		Quantity:           position.Quantity,
		SettlementPrice:   underlyingPrice,
		SettlementAmount:   settlementAmount,
		Profit:             settlementAmount - (position.EntryPrice * float64(position.Quantity)),
		TxHash:             generateTxHash(),
		Status:             "completed",
	}
	
	s.db.Create(&settlement)
	
	// Close position
	s.db.Delete(&position)
	
	// Update contract
	contract.OpenInterest -= position.Quantity
	s.db.Save(&contract)
	
	return &settlement, nil
}

func (s *OptionsService) ProcessExpiration() error {
	s.mu.RLock()
	contracts := make([]OptionContract, 0, len(s.contracts))
	for _, c := range s.contracts {
		if c.IsActive && c.ExpirationDate.Before(time.Now()) {
			contracts = append(contracts, c)
		}
	}
	s.mu.RUnlock()
	
	for _, contract := range contracts {
		// Get all positions for this contract
		var positions []OptionPosition
		s.db.Where("contract_id = ?", contract.ContractID).Find(&positions)
		
		for _, position := range positions {
			// Calculate settlement
			settlementAmount := 0.0
			if contract.OptionType == "call" {
				settlementAmount = math.Max(0, contract.StrikePrice*0.1) // Simplified
			} else {
				settlementAmount = math.Max(0, contract.StrikePrice*0.05)
			}
			
			settlement := OptionSettlement{
				SettlementID:      uuid.New().String(),
				ContractID:        contract.ContractID,
				UserID:            position.UserID,
				PositionID:        position.ID,
				SettlementType:    "expire",
				Quantity:          position.Quantity,
				SettlementPrice:   contract.StrikePrice,
				SettlementAmount:  settlementAmount * float64(position.Quantity),
				Profit:           -position.EntryPrice * float64(position.Quantity),
				TxHash:           generateTxHash(),
				Status:           "completed",
			}
			s.db.Create(&settlement)
			
			// Delete position
			s.db.Delete(&position)
		}
		
		// Deactivate contract
		contract.IsActive = false
		s.db.Save(&contract)
		
		s.mu.Lock()
		s.contracts[contract.ContractID] = contract
		s.mu.Unlock()
	}
	
	return nil
}

// ============================================================================
// API Handlers
// ============================================================================

func (s *OptionsService) setupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	{
		// Contracts
		api.GET("/contracts", s.getContracts)
		api.GET("/contracts/:underlying/chain", s.getOptionChain)
		
		// Orders
		api.POST("/orders", s.placeOrder)
		api.GET("/orders/:order_id", s.getOrder)
		api.POST("/orders/:order_id/cancel", s.cancelOrder)
		
		// Positions
		api.GET("/user/:user_id/positions", s.getUserPositions)
		api.POST("/positions/:position_id/close", s.closePosition)
		api.POST("/positions/:position_id/exercise", s.exercisePosition)
		
		// Market data
		api.GET("/market/:underlying", s.getMarketData)
		api.GET("/history/:contract_id", s.getTradeHistory)
	}
	
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "options-trading"})
	})
}

func (s *OptionsService) getContracts(c *gin.Context) {
	underlying := c.Query("underlying")
	
	if underlying == "" {
		s.mu.RLock()
		contracts := make([]OptionContract, 0, len(s.contracts))
		for _, c := range s.contracts {
			if c.IsActive {
				contracts = append(contracts, c)
			}
		}
		s.mu.RUnlock()
		c.JSON(http.StatusOK, gin.H{"contracts": contracts})
		return
	}
	
	contracts, err := s.GetOptionContracts(underlying)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"contracts": contracts})
}

func (s *OptionsService) getOptionChain(c *gin.Context) {
	underlying := c.Param("underlying")
	
	chain, err := s.GetOptionChain(underlying)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"chain": chain})
}

func (s *OptionsService) placeOrder(c *gin.Context) {
	var req struct {
		UserID       uint    `json:"user_id" binding:"required"`
		ContractID   string  `json:"contract_id" binding:"required"`
		OrderType    string  `json:"order_type" binding:"required"` // market, limit
		Side         string  `json:"side" binding:"required"` // buy, sell
		PositionType string  `json:"position_type" binding:"required"` // long, short
		Quantity     int     `json:"quantity" binding:"required"`
		LimitPrice   float64 `json:"limit_price"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	order, err := s.PlaceOrder(
		req.UserID,
		req.ContractID,
		req.OrderType,
		req.Side,
		req.PositionType,
		req.Quantity,
		req.LimitPrice,
	)
	
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{"order": order})
}

func (s *OptionsService) getOrder(c *gin.Context) {
	orderID := c.Param("order_id")
	
	var order OptionOrder
	if err := s.db.Where("order_id = ?", orderID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"order": order})
}

func (s *OptionsService) cancelOrder(c *gin.Context) {
	orderID := c.Param("order_id")
	
	var order OptionOrder
	if err := s.db.Where("order_id = ?", orderID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	
	if order.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order cannot be cancelled"})
		return
	}
	
	order.Status = "cancelled"
	now := time.Now()
	order.CancelledAt = &now
	s.db.Save(&order)
	
	c.JSON(http.StatusOK, gin.H{"message": "order cancelled"})
}

func (s *OptionsService) getUserPositions(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Param("user_id"), 10, 32)
	
	positions, err := s.GetUserPositions(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"positions": positions})
}

func (s *OptionsService) closePosition(c *gin.Context) {
	positionID, _ := strconv.ParseUint(c.Param("position_id"), 10, 32)
	
	var req struct {
		UserID   uint `json:"user_id" binding:"required"`
		Quantity int  `json:"quantity" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	trade, err := s.ClosePosition(req.UserID, uint(positionID), req.Quantity)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"trade": trade})
}

func (s *OptionsService) exercisePosition(c *gin.Context) {
	positionID, _ := strconv.ParseUint(c.Param("position_id"), 10, 32)
	
	var req struct {
		UserID uint `json:"user_id" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	settlement, err := s.ExerciseOption(req.UserID, uint(positionID))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"settlement": settlement})
}

func (s *OptionsService) getMarketData(c *gin.Context) {
	underlying := c.Param("underlying")
	
	s.mu.RLock()
	contracts := make([]OptionContract, 0)
	for _, c := range s.contracts {
		if c.UnderlyingAsset == underlying && c.IsActive {
			contracts = append(contracts, c)
		}
	}
	s.mu.RUnlock()
	
	c.JSON(http.StatusOK, gin.H{
		"underlying":     underlying,
		"contracts":      len(contracts),
		"open_interest":  getTotalOpenInterest(contracts),
		"volume_24h":    getTotalVolume(contracts),
	})
}

func (s *OptionsService) getTradeHistory(c *gin.Context) {
	contractID := c.Param("contract_id")
	
	var trades []OptionTrade
	s.db.Where("contract_id = ?", contractID).Order("created_at DESC").Limit(100).Find(&trades)
	
	c.JSON(http.StatusOK, gin.H{"trades": trades})
}

// ============================================================================
// Helper Functions
// ============================================================================

func getTotalOpenInterest(contracts []OptionContract) int {
	total := 0
	for _, c := range contracts {
		total += c.OpenInterest
	}
	return total
}

func getTotalVolume(contracts []OptionContract) int {
	total := 0
	for _, c := range contracts {
		total += c.Volume24h
	}
	return total
}

func generateTxHash() string {
	data := fmt.Sprintf("%d:%s", time.Now().UnixNano(), uuid.New().String())
	return fmt.Sprintf("0x%x", data)
}

// ============================================================================
// Main
// ============================================================================

func main() {
	cfg := LoadConfig()
	
	service, err := NewOptionsService(cfg)
	if err != nil {
		log.Fatalf("Failed to create options service: %v", err)
	}
	
	// Start expiration processor
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		for range ticker.C {
			service.ProcessExpiration()
		}
	}()
	
	router := gin.Default()
	service.setupRoutes(router)
	
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		<-quit
		log.Println("Shutting down options service...")
		os.Exit(0)
	}()
	
	log.Printf("Options Trading Service starting on port %s", cfg.ServerPort)
	if err := router.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
