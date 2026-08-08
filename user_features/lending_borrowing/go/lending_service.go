// TigerWallet Lending & Borrowing Service
// High-Load Distributed Go Implementation
// Handles lending, borrowing, liquidation, and interest calculations

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
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
	ChainRPCURL  string `json:"chain_rpc_url"`
	PrivateKey   string `json:"private_key"` // Encrypted
}

// ============================================================================
// Data Models
// ============================================================================

// LendingMarket represents a lending market for an asset
type LendingMarket struct {
	ID                    uint      `gorm:"primaryKey" json:"id"`
	AssetAddress         string    `gorm:"uniqueIndex;not null" json:"asset_address"`
	AssetSymbol          string    `json:"asset_symbol"`
	AssetName            string    `json:"asset_name"`
	AssetDecimals        uint8     `json:"asset_decimals"`
	TotalSupply          string    `json:"total_supply"`
	TotalBorrows         string    `json:"total_borrows"`
	SupplyRate           string    `json:"supply_rate"`
	BorrowRate           string    `json:"borrow_rate"`
	SupplyAPY            float64   `json:"supply_apy"`
	BorrowAPY            float64   `json:"borrow_apy"`
	UtilizationRate      float64   `json:"utilization_rate"`
	LTV                  float64   `json:"ltv"` // Loan to Value
	LiquidationThreshold float64   `json:"liquidation_threshold"`
	LiquidationBonus     float64   `json:"liquidation_bonus"`
	IsActive             bool      `json:"is_active"`
	ChainID              int64     `json:"chain_id"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// UserSupply represents a user's supply position
type UserSupply struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserAddress   string    `gorm:"index;not null" json:"user_address"`
	MarketID      uint      `gorm:"index;not null" json:"market_id"`
	AssetAddress  string    `json:"asset_address"`
	Balance       string    `json:"balance"`
	BalanceUSD    float64   `json:"balance_usd"`
	AccruedRewards string   `json:"accrued_rewards"`
	APY           float64   `json:"apy"`
	ChainID       int64     `json:"chain_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// UserBorrow represents a user's borrow position
type UserBorrow struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	UserAddress     string    `gorm:"index;not null" json:"user_address"`
	MarketID        uint      `gorm:"index;not null" json:"market_id"`
	AssetAddress    string    `json:"asset_address"`
	Balance         string    `json:"balance"`
	BalanceUSD      float64   `json:"balance_usd"`
	AccruedInterest string    `json:"accrued_interest"`
	APY             float64   `json:"apy"`
	ChainID         int64     `json:"chain_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// UserCollateral represents user's collateral positions
type UserCollateral struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserAddress   string    `gorm:"index;not null" json:"user_address"`
	AssetAddress string    `json:"asset_address"`
	AssetSymbol  string    `json:"asset_symbol"`
	ValueUSD     float64   `json:"value_usd"`
	IsCollateral  bool      `json:"is_collateral"`
	ChainID      int64     `json:"chain_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Liquidation represents a liquidation event
type Liquidation struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Liquidator     string    `json:"liquidator"`
	UserAddress    string    `json:"user_address"`
	RepayAsset     string    `json:"repay_asset"`
	RepayAmount    string    `json:"repay_amount"`
	CollateralAsset string   `json:"collateral_asset"`
	CollateralAmount string  `json:"collateral_amount"`
	ProfitUSD      float64   `json:"profit_usd"`
	TransactionHash string   `json:"transaction_hash"`
	ChainID        int64     `json:"chain_id"`
	CreatedAt      time.Time `json:"created_at"`
}

// InterestRate holds interest rate configuration
type InterestRate struct {
	BaseRate       float64 `json:"base_rate"`
	Slope1         float64 `json:"slope1"`
	Slope2         float64 `json:"slope2"`
	OptimalUtil    float64 `json:"optimal_util"`
}

// ============================================================================
// Service Implementation
// ============================================================================

type LendingService struct {
	db           *gorm.DB
	redis        *redis.Client
	config       Config
	interestRate InterestRate
	markets      map[string]*LendingMarket
	mu           sync.RWMutex
}

// NewLendingService creates a new lending service
func NewLendingService(config Config) (*LendingService, error) {
	// Connect to PostgreSQL
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto migrate
	err = db.AutoMigrate(
		&LendingMarket{},
		&UserSupply{},
		&UserBorrow{},
		&UserCollateral{},
		&Liquidation{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort),
	})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		fmt.Printf("Warning: Redis connection failed: %v\n", err)
	}

	service := &LendingService{
		db:    db,
		redis: rdb,
		config: config,
		interestRate: InterestRate{
			BaseRate:    0.02,   // 2% base rate
			Slope1:      0.10,   // 10% slope 1
			Slope2:      0.60,   // 60% slope 2
			OptimalUtil: 0.80,   // 80% optimal utilization
		},
		markets: make(map[string]*LendingMarket),
	}

	// Initialize markets
	go service.initializeMarkets()

	return service, nil
}

// ============================================================================
// Market Management
// ============================================================================

func (s *LendingService) initializeMarkets() {
	// Initialize default markets
	defaultMarkets := []LendingMarket{
		{
			AssetAddress:    "0x0000000000000000000000000000000000000000", // ETH
			AssetSymbol:     "ETH",
			AssetName:       "Ethereum",
			AssetDecimals:   18,
			LTV:             0.80,
			LiquidationThreshold: 0.85,
			LiquidationBonus: 0.05,
			IsActive:        true,
			ChainID:         1,
		},
		{
			AssetAddress:    "0xdAC17F958D2ee523a2206206994597C13D831ec7", // USDT
			AssetSymbol:     "USDT",
			AssetName:       "Tether USD",
			AssetDecimals:   6,
			LTV:             0.90,
			LiquidationThreshold: 0.95,
			LiquidationBonus: 0.02,
			IsActive:        true,
			ChainID:         1,
		},
		{
			AssetAddress:    "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", // USDC
			AssetSymbol:     "USDC",
			AssetName:       "USD Coin",
			AssetDecimals:   6,
			LTV:             0.90,
			LiquidationThreshold: 0.95,
			LiquidationBonus: 0.02,
			IsActive:        true,
			ChainID:         1,
		},
		{
			AssetAddress:    "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599", // WBTC
			AssetSymbol:     "WBTC",
			AssetName:       "Wrapped Bitcoin",
			AssetDecimals:   8,
			LTV:             0.70,
			LiquidationThreshold: 0.80,
			LiquidationBonus: 0.05,
			IsActive:        true,
			ChainID:         1,
		},
	}

	for _, market := range defaultMarkets {
		var existing LendingMarket
		if s.db.Where("asset_address = ?", market.AssetAddress).First(&existing).RowsAffected == 0 {
			s.db.Create(&market)
		}
	}
}

// Calculate interest rate based on utilization
func (s *LendingService) calculateInterestRate(utilization float64) (supplyRate, borrowRate float64) {
	ir := s.interestRate

	if utilization <= ir.OptimalUtil {
		// Linear increase from baseRate to baseRate + slope1
		slope := (ir.BaseRate + ir.Slope1 - ir.BaseRate) / ir.OptimalUtil
		borrowRate = ir.BaseRate + slope*utilization
	} else {
		// Non-linear increase beyond optimal
		excess := utilization - ir.OptimalUtil
		slope := (ir.BaseRate + ir.Slope2 - (ir.BaseRate + ir.Slope1)) / (1 - ir.OptimalUtil)
		borrowRate = ir.BaseRate + ir.Slope1 + slope*excess
	}

	// Supply rate is borrow rate * utilization * (1 - reserve factor)
	reserveFactor := 0.10 // 10% to protocol
	supplyRate = borrowRate * utilization * (1 - reserveFactor)

	return supplyRate, borrowRate
}

// UpdateMarketRates updates the supply and borrow rates for a market
func (s *LendingService) UpdateMarketRates(marketID uint) error {
	var market LendingMarket
	if err := s.db.First(&market, marketID).Error; err != nil {
		return err
	}

	totalSupply, _ := big.NewInt(0).SetString(market.TotalSupply, 10)
	totalBorrows, _ := big.NewInt(0).SetString(market.TotalBorrows, 10)

	if totalSupply.Cmp(big.NewInt(0)) == 0 {
		market.UtilizationRate = 0
		market.SupplyAPY = 0
		market.BorrowAPY = s.interestRate.BaseRate
	} else {
		utilization := float64(totalBorrows.Int64()) / float64(totalSupply.Int64())
		market.UtilizationRate = utilization

		supplyRate, borrowRate := s.calculateInterestRate(utilization)
		market.SupplyAPY = supplyRate * 100
		market.BorrowAPY = borrowRate * 100
	}

	return s.db.Save(&market).Error
}

// ============================================================================
// Supply (Lend) Operations
// ============================================================================

type SupplyRequest struct {
	UserAddress   string `json:"user_address" binding:"required"`
	AssetAddress  string `json:"asset_address" binding:"required"`
	Amount        string `json:"amount" binding:"required"`
	ChainID       int64  `json:"chain_id"`
}

type SupplyResponse struct {
	Success         bool    `json:"success"`
	TransactionHash string  `json:"transaction_hash,omitempty"`
	NewBalance      string  `json:"new_balance"`
	NewBalanceUSD  float64  `json:"new_balance_usd"`
	APY             float64 `json:"apy"`
	Error           string  `json:"error,omitempty"`
}

// Supply assets to the lending market
func (s *LendingService) Supply(ctx *gin.Context) {
	var req SupplyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, SupplyResponse{Success: false, Error: err.Error()})
		return
	}

	// Get market
	var market LendingMarket
	if err := s.db.Where("asset_address = ? AND chain_id = ?", req.AssetAddress, req.ChainID).First(&market).Error; err != nil {
		ctx.JSON(404, SupplyResponse{Success: false, Error: "Market not found"})
		return
	}

	amount, ok := new(big.Int).SetString(req.Amount, 10)
	if !ok {
		ctx.JSON(400, SupplyResponse{Success: false, Error: "Invalid amount"})
		return
	}

	// Update market total supply
	currentTotal, _ := big.NewInt(0).SetString(market.TotalSupply, 10)
	newTotal := new(big.Int).Add(currentTotal, amount)
	market.TotalSupply = newTotal.String()
	s.db.Save(&market)

	// Update user's supply position
	var userSupply UserSupply
	result := s.db.Where("user_address = ? AND market_id = ?", req.UserAddress, market.ID).First(&userSupply)

	if result.RowsAffected == 0 {
		userSupply = UserSupply{
			UserAddress:  req.UserAddress,
			MarketID:    market.ID,
			AssetAddress: req.AssetAddress,
			Balance:     amount.String(),
			APY:         market.SupplyAPY,
			ChainID:     req.ChainID,
		}
		s.db.Create(&userSupply)
	} else {
		currentBalance, _ := big.NewInt(0).SetString(userSupply.Balance, 10)
		newBalance := new(big.Int).Add(currentBalance, amount)
		userSupply.Balance = newBalance.String()
		userSupply.APY = market.SupplyAPY
		s.db.Save(&userSupply)
	}

	// Transaction hash is only known after broadcasting; do not fabricate one.
	// Update Redis cache
	cacheKey := fmt.Sprintf("supply:%s:%s", req.UserAddress, req.AssetAddress)
	s.redis.Set(ctx, cacheKey, userSupply.Balance, time.Hour)

	ctx.JSON(200, SupplyResponse{
		Success:         true,
		TransactionHash: "",
		NewBalance:      userSupply.Balance,
		APY:             market.SupplyAPY,
	})
}

// ============================================================================
// Borrow Operations
// ============================================================================

type BorrowRequest struct {
	UserAddress   string `json:"user_address" binding:"required"`
	AssetAddress  string `json:"asset_address" binding:"required"`
	Amount        string `json:"amount" binding:"required"`
	ChainID       int64  `json:"chain_id"`
}

type BorrowResponse struct {
	Success          bool    `json:"success"`
	TransactionHash  string  `json:"transaction_hash,omitempty"`
	NewBorrowBalance string  `json:"new_borrow_balance"`
	NewBorrowUSD    float64 `json:"new_borrow_usd"`
	APY             float64 `json:"apy"`
	HealthFactor    float64 `json:"health_factor"`
	Error           string  `json:"error,omitempty"`
}

// Borrow assets from the lending market
func (s *LendingService) Borrow(ctx *gin.Context) {
	var req BorrowRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, BorrowResponse{Success: false, Error: err.Error()})
		return
	}

	// Get market
	var market LendingMarket
	if err := s.db.Where("asset_address = ? AND chain_id = ?", req.AssetAddress, req.ChainID).First(&market).Error; err != nil {
		ctx.JSON(404, BorrowResponse{Success: false, Error: "Market not found"})
		return
	}

	// Check user's collateral
	collateralUSD := s.calculateUserCollateral(req.UserAddress, req.ChainID)
	borrowUSD := s.calculateUserBorrows(req.UserAddress, req.ChainID)
	amountUSD := s.calculateAssetUSD(req.AssetAddress, req.Amount, req.ChainID)

	newBorrowUSD := borrowUSD + amountUSD

	// Check LTV
	maxBorrowUSD := collateralUSD * market.LTV
	if newBorrowUSD > maxBorrowUSD {
		ctx.JSON(400, BorrowResponse{Success: false, Error: "Insufficient collateral"})
		return
	}

	// Check health factor
	healthFactor := s.calculateHealthFactor(req.UserAddress, req.ChainID)
	if healthFactor < 1.0 {
		ctx.JSON(400, BorrowResponse{Success: false, Error: "Health factor too low"})
		return
	}

	// Update market total borrows
	currentTotal, _ := big.NewInt(0).SetString(market.TotalBorrows, 10)
	newTotal := new(big.Int).Add(currentTotal, new(big.Int).SetString(req.Amount, 10))
	market.TotalBorrows = newTotal.String()
	s.db.Save(&market)

	// Update user's borrow position
	var userBorrow UserBorrow
	result := s.db.Where("user_address = ? AND market_id = ?", req.UserAddress, market.ID).First(&userBorrow)

	if result.RowsAffected == 0 {
		userBorrow = UserBorrow{
			UserAddress:  req.UserAddress,
			MarketID:    market.ID,
			AssetAddress: req.AssetAddress,
			Balance:     req.Amount,
			APY:         market.BorrowAPY,
			ChainID:     req.ChainID,
		}
		s.db.Create(&userBorrow)
	} else {
		currentBalance, _ := big.NewInt(0).SetString(userBorrow.Balance, 10)
		newBalance := new(big.Int).Add(currentBalance, new(big.Int).SetString(req.Amount, 10))
		userBorrow.Balance = newBalance.String()
		userBorrow.APY = market.BorrowAPY
		s.db.Save(&userBorrow)
	}

	ctx.JSON(200, BorrowResponse{
		Success:          true,
		TransactionHash:  "",
		NewBorrowBalance: userBorrow.Balance,
		NewBorrowUSD:    newBorrowUSD,
		APY:             market.BorrowAPY,
		HealthFactor:    healthFactor,
	})
}

// ============================================================================
// Health Factor & Liquidation
// ============================================================================

func (s *LendingService) calculateUserCollateral(userAddress string, chainID int64) float64 {
	var collaterals []UserCollateral
	s.db.Where("user_address = ? AND chain_id = ? AND is_collateral = ?", userAddress, chainID, true).Find(&collaterals)

	totalUSD := 0.0
	for _, c := range collaterals {
		totalUSD += c.ValueUSD
	}
	return totalUSD
}

func (s *LendingService) calculateUserBorrows(userAddress string, chainID int64) float64 {
	var borrows []UserBorrow
	s.db.Where("user_address = ? AND chain_id = ?", userAddress, chainID).Find(&borrows)

	totalUSD := 0.0
	for _, b := range borrows {
		totalUSD += b.BalanceUSD
	}
	return totalUSD
}

func (s *LendingService) calculateHealthFactor(userAddress string, chainID int64) float64 {
	collateralUSD := s.calculateUserCollateral(userAddress, chainID)
	borrowUSD := s.calculateUserBorrows(userAddress, chainID)

	if borrowUSD == 0 {
		return math.MaxFloat64 // Infinite health factor if no borrows
	}

	// Weighted average of liquidation thresholds
	var weightedThreshold float64
	var collaterals []UserCollateral
	s.db.Where("user_address = ? AND chain_id = ? AND is_collateral = ?", userAddress, chainID, true).Find(&collaterals)

	if len(collaterals) == 0 {
		return 0
	}

	for _, c := range collaterals {
		var market LendingMarket
		if s.db.Where("asset_address = ? AND chain_id = ?", c.AssetAddress, chainID).First(&market).Error == nil {
			weight := c.ValueUSD / collateralUSD
			weightedThreshold += weight * market.LiquidationThreshold
		}
	}

	return (collateralUSD * weightedThreshold) / borrowUSD
}

func (s *LendingService) calculateAssetUSD(assetAddress, amount string, chainID int64) float64 {
	// Get price from oracle (simplified)
	prices := map[string]float64{
		"0x0000000000000000000000000000000000000000": 3500.0,  // ETH
		"0xdAC17F958D2ee523a2206206994597C13D831ec7": 1.0,    // USDT
		"0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48": 1.0,  // USDC
		"0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599": 65000.0, // WBTC
	}

	price, ok := prices[assetAddress]
	if !ok {
		price = 0 // Unknown asset
	}

	amountFloat, _ := big.NewInt(0).SetString(amount, 10)
	decimals := 18
	if assetAddress == "0xdAC17F958D2ee523a2206206994597C13D831ec7" ||
		assetAddress == "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48" {
		decimals = 6
	}

	amountEth := float64(amountFloat.Int64()) / math.Pow10(decimals)
	return amountEth * price
}

// ============================================================================
// Repay Operations
// ============================================================================

type RepayRequest struct {
	UserAddress   string `json:"user_address" binding:"required"`
	AssetAddress  string `json:"asset_address" binding:"required"`
	Amount        string `json:"amount" binding:"required"`
	ChainID       int64  `json:"chain_id"`
}

// Repay borrowed assets
func (s *LendingService) Repay(ctx *gin.Context) {
	var req RepayRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	var market LendingMarket
	if err := s.db.Where("asset_address = ? AND chain_id = ?", req.AssetAddress, req.ChainID).First(&market).Error; err != nil {
		ctx.JSON(404, gin.H{"success": false, "error": "Market not found"})
		return
	}

	var userBorrow UserBorrow
	if err := s.db.Where("user_address = ? AND market_id = ?", req.UserAddress, market.ID).First(&userBorrow).Error; err != nil {
		ctx.JSON(404, gin.H{"success": false, "error": "No borrow position"})
		return
	}

	amount, _ := new(big.Int).SetString(req.Amount, 10)
	currentBalance, _ := new(big.Int).SetString(userBorrow.Balance, 10)

	// Cannot repay more than borrowed
	if amount.Cmp(currentBalance) > 0 {
		amount = currentBalance
	}

	newBalance := new(big.Int).Sub(currentBalance, amount)
	userBorrow.Balance = newBalance.String()
	s.db.Save(&userBorrow)

	// Update market total borrows
	marketTotal, _ := big.NewInt(0).SetString(market.TotalBorrows, 10)
	market.TotalBorrows = new(big.Int).Sub(marketTotal, amount).String()
	s.db.Save(&market)

	ctx.JSON(200, gin.H{
		"success":           true,
		"transaction_hash":  "",
		"remaining_balance":  userBorrow.Balance,
	})
}

// ============================================================================
// Withdraw Operations
// ============================================================================

type WithdrawRequest struct {
	UserAddress   string `json:"user_address" binding:"required"`
	AssetAddress  string `json:"asset_address" binding:"required"`
	Amount        string `json:"amount" binding:"required"`
	ChainID       int64  `json:"chain_id"`
}

// Withdraw supplied assets
func (s *LendingService) Withdraw(ctx *gin.Context) {
	var req WithdrawRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	var market LendingMarket
	if err := s.db.Where("asset_address = ? AND chain_id = ?", req.AssetAddress, req.ChainID).First(&market).Error; err != nil {
		ctx.JSON(404, gin.H{"success": false, "error": "Market not found"})
		return
	}

	var userSupply UserSupply
	if err := s.db.Where("user_address = ? AND market_id = ?", req.UserAddress, market.ID).First(&userSupply).Error; err != nil {
		ctx.JSON(404, gin.H{"success": false, "error": "No supply position"})
		return
	}

	// Check if withdrawal would cause undercollateralization
	amountUSD := s.calculateAssetUSD(req.AssetAddress, req.Amount, req.ChainID)
	collateralUSD := s.calculateUserCollateral(req.UserAddress, req.ChainID)
	newCollateralUSD := collateralUSD - amountUSD
	borrowUSD := s.calculateUserBorrows(req.UserAddress, req.ChainID)

	if borrowUSD > 0 {
		maxWithdrawUSD := collateralUSD - (borrowUSD / 0.8) // 80% LTV
		if amountUSD > maxWithdrawUSD {
			ctx.JSON(400, gin.H{"success": false, "error": "Would cause undercollateralization"})
			return
		}
	}

	amount, _ := new(big.Int).SetString(req.Amount, 10)
	currentBalance, _ := big.NewInt(0).SetString(userSupply.Balance, 10)

	if amount.Cmp(currentBalance) > 0 {
		amount = currentBalance
	}

	newBalance := new(big.Int).Sub(currentBalance, amount)
	userSupply.Balance = newBalance.String()
	s.db.Save(&userSupply)

	// Update market total supply
	marketTotal, _ := big.NewInt(0).SetString(market.TotalSupply, 10)
	market.TotalSupply = new(big.Int).Sub(marketTotal, amount).String()
	s.db.Save(&market)

	ctx.JSON(200, gin.H{
		"success":           true,
		"transaction_hash":  "",
		"remaining_balance": userSupply.Balance,
	})
}

// ============================================================================
// User Position Queries
// ============================================================================

type UserPositionResponse struct {
	Supplies    []UserSupply   `json:"supplies"`
	Borrows     []UserBorrow   `json:"borrows"`
	Collateral  float64        `json:"collateral_usd"`
	BorrowsUSD  float64        `json:"borrows_usd"`
	HealthFactor float64       `json:"health_factor"`
	NetAPY      float64        `json:"net_apy"`
}

// Get user's lending positions
func (s *LendingService) GetUserPosition(ctx *gin.Context) {
	userAddress := ctx.Query("user_address")
	chainID := ctx.GetInt64("chain_id")

	if userAddress == "" {
		ctx.JSON(400, gin.H{"error": "user_address required"})
		return
	}

	var supplies []UserSupply
	s.db.Where("user_address = ? AND chain_id = ?", userAddress, chainID).Find(&supplies)

	var borrows []UserBorrow
	s.db.Where("user_address = ? AND chain_id = ?", userAddress, chainID).Find(&borrows)

	collateralUSD := s.calculateUserCollateral(userAddress, chainID)
	borrowUSD := s.calculateUserBorrows(userAddress, chainID)
	healthFactor := s.calculateHealthFactor(userAddress, chainID)

	// Calculate net APY
	supplyAPY := 0.0
	for _, supply := range supplies {
		supplyAPY += supply.BalanceUSD * supply.APY
	}
	if collateralUSD > 0 {
		supplyAPY /= collateralUSD
	}

	borrowAPY := 0.0
	if borrowUSD > 0 {
		for _, borrow := range borrows {
			borrowAPY += borrow.BalanceUSD * borrow.APY
		}
		borrowAPY /= borrowUSD
	}

	netAPY := supplyAPY - borrowAPY

	ctx.JSON(200, UserPositionResponse{
		Supplies:     supplies,
		Borrows:      borrows,
		Collateral:   collateralUSD,
		BorrowsUSD:   borrowUSD,
		HealthFactor: healthFactor,
		NetAPY:       netAPY,
	})
}

// Get all markets
func (s *LendingService) GetMarkets(ctx *gin.Context) {
	var markets []LendingMarket
	s.db.Where("is_active = ?", true).Find(&markets)

	ctx.JSON(200, gin.H{"markets": markets})
}

// ============================================================================
// Helper Functions
// ============================================================================

// ============================================================================
// Main
// ============================================================================

func main() {
	config := Config{
		ServerPort:  "8090",
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBPort:      getEnv("DB_PORT", "5432"),
		DBUser:      getEnv("DB_USER", "tigerwallet"),
		DBPassword:  getEnv("DB_PASSWORD", "password"),
		DBName:      getEnv("DB_NAME", "tigerwallet_lending"),
		RedisHost:   getEnv("REDIS_HOST", "localhost"),
		RedisPort:   getEnv("REDIS_PORT", "6379"),
		ChainRPCURL: getEnv("CHAIN_RPC_URL", "https://eth.llamarpc.com"),
	}

	service, err := NewLendingService(config)
	if err != nil {
		fmt.Printf("Failed to start lending service: %v\n", err)
		os.Exit(1)
	}

	// Setup Gin router
	router := gin.Default()

	// CORS middleware
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
	api := router.Group("/api/v1/lending")
	{
		api.GET("/markets", service.GetMarkets)
		api.GET("/position", service.GetUserPosition)
		api.POST("/supply", service.Supply)
		api.POST("/borrow", service.Borrow)
		api.POST("/repay", service.Repay)
		api.POST("/withdraw", service.Withdraw)
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "lending"})
	})

	// Start server
	go func() {
		fmt.Printf("Lending service starting on port %s\n", config.ServerPort)
		if err := router.Run(":" + config.ServerPort); err != nil {
			fmt.Printf("Failed to start server: %v\n", err)
		}
	}()

	// Market rate update loop
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			var markets []LendingMarket
			service.db.Find(&markets)
			for _, market := range markets {
				service.UpdateMarketRates(market.ID)
			}
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down lending service...")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
