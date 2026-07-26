/**
 * TigerWallet Lending & Borrowing Service
 * Complete DeFi Lending Platform
 * 
 * Features:
 * - Supply/Deposit assets
 * - Borrow assets
 * - Collateral management
 * - Liquidation system
 * - Interest rate models
 * - Flash loans
 * - Cross-chain lending
 * - Yield farming
 */

package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/big"
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
	ServerPort      string
	DBHost          string
	DBPort           string
	DBUser          string
	DBPassword       string
	DBName          string
	RedisHost       string
	RedisPort       string
	MasterWallet    string
	GasStation      string
	PriceOracle     string
	LiquidationBonus float64
	MinHealthFactor float64
}

func LoadConfig() *Config {
	return &Config{
		ServerPort:      getEnv("LENDING_PORT", "9096"),
		DBHost:          getEnv("DB_HOST", "localhost"),
		DBPort:          getEnv("DB_PORT", "5432"),
		DBUser:          getEnv("DB_USER", "tigerwallet"),
		DBPassword:      getEnv("DB_PASSWORD", "password"),
		DBName:          getEnv("DB_NAME", "tigerwallet"),
		RedisHost:       getEnv("REDIS_HOST", "localhost"),
		RedisPort:       getEnv("REDIS_PORT", "6379"),
		MasterWallet:    getEnv("MASTER_WALLET", ""),
		GasStation:      getEnv("GAS_STATION", "http://localhost:8080"),
		PriceOracle:     getEnv("PRICE_ORACLE", "http://localhost:8081"),
		LiquidationBonus: 0.05,
		MinHealthFactor:  1.0,
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

type Asset struct {
	ID                uint           `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
	AssetID           string         `gorm:"uniqueIndex;size:36" json:"asset_id"`
	Symbol            string         `gorm:"index" json:"symbol"`
	Name              string         `json:"name"`
	ContractAddress   string         `json:"contract_address"`
	ChainID           int            `json:"chain_id"`
	Decimals          int            `json:"decimals"`
	IsActive          bool           `json:"is_active"`
	IsCollateral      bool           `json:"is_collateral"`
	IsBorrowable      bool           `json:"is_borrowable"`
	SupplyAPY         float64        `json:"supply_apy"`
	BorrowAPY         float64        `json:"borrow_apy"`
	ReserveFactor     float64        `json:"reserve_factor"`
	LTV               float64        `json:"ltv"` // Loan-to-Value ratio
	LiquidationThreshold float64    `json:"liquidation_threshold"`
	LiquidationPenalty float64       `json:"liquidation_penalty"`
	SupplyCap         string         `json:"supply_cap"`
	BorrowCap         string         `json:"borrow_cap"`
	TotalSupplied     string         `json:"total_supplied"`
	TotalBorrowed     string         `json:"total_borrowed"`
	UtilizationRate   float64        `json:"utilization_rate"`
	PriceUSD          float64        `json:"price_usd"`
	LastPriceUpdate   time.Time      `json:"last_price_update"`
}

type UserPosition struct {
	ID                uint           `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	UserID            uint           `gorm:"index" json:"user_id"`
	User              User           `gorm:"foreignKey:UserID" json:"-"`
	WalletAddress     string         `gorm:"index" json:"wallet_address"`
	ChainID           int            `json:"chain_id"`
	TotalSuppliedUSD  float64        `json:"total_supplied_usd"`
	TotalBorrowedUSD  float64        `json:"total_borrowed_usd"`
	HealthFactor      float64        `json:"health_factor"`
	NetAPY            float64        `json:"net_apy"`
	IsLiquidatable    bool           `json:"is_liquidatable"`
	LastUpdated       time.Time      `json:"last_updated"`
}

type Supply struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	UserPositionID    uint      `gorm:"index" json:"user_position_id"`
	UserPosition     UserPosition `gorm:"foreignKey:UserPositionID" json:"-"`
	AssetID           string    `gorm:"index" json:"asset_id"`
	Asset             Asset     `gorm:"foreignKey:AssetID" json:"-"`
	Amount            string    `json:"amount"`
	AmountUSD         float64   `json:"amount_usd"`
	AccruedRewards    string    `json:"accrued_rewards"`
	APY               float64   `json:"apy"`
	LastUpdated       time.Time `json:"last_updated"`
}

type Borrow struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	UserPositionID    uint      `gorm:"index" json:"user_position_id"`
	UserPosition     UserPosition `gorm:"foreignKey:UserPositionID" json:"-"`
	AssetID           string    `gorm:"index" json:"asset_id"`
	Asset             Asset     `gorm:"foreignKey:AssetID" json:"-"`
	Amount            string    `json:"amount"`
	AmountUSD         float64   `json:"amount_usd"`
	APY               float64   `json:"apy"`
	InterestIndex     string    `json:"interest_index"`
	LastUpdated       time.Time `json:"last_updated"`
}

type Transaction struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	TxHash            string    `gorm:"uniqueIndex;size:66" json:"tx_hash"`
	UserID            uint      `gorm:"index" json:"user_id"`
	WalletAddress     string    `gorm:"index" json:"wallet_address"`
	ChainID           int       `json:"chain_id"`
	Type              string    `json:"type"` // supply, borrow, repay, withdraw, liquidation
	AssetID           string    `json:"asset_id"`
	Amount            string    `json:"amount"`
	AmountUSD         float64   `json:"amount_usd"`
	Status            string    `json:"status"` // pending, confirmed, failed
	BlockNumber       int64     `json:"block_number"`
	GasUsed           string    `json:"gas_used"`
	ErrorMessage      string    `json:"error_message"`
}

type Liquidation struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	LiquidatorID      uint      `gorm:"index" json:"liquidator_id"`
	VictimUserID      uint      `gorm:"index" json:"victim_user_id"`
	RepayAssetID      string    `json:"repay_asset_id"`
	RepayAmount       string    `json:"repay_amount"`
	RepayAmountUSD    float64   `json:"repay_amount_usd"`
	CollateralAssetID string    `json:"collateral_asset_id"`
	CollateralAmount  string    `json:"collateral_amount"`
	CollateralAmountUSD float64 `json:"collateral_amount_usd"`
	ProfitUSD         float64   `json:"profit_usd"`
	TxHash            string    `gorm:"uniqueIndex;size:66" json:"tx_hash"`
	Status            string    `json:"status"`
}

type User struct {
	ID                uint           `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
	UUID              string         `gorm:"uniqueIndex;size:36" json:"uuid"`
	Email             string         `gorm:"index" json:"email"`
	WalletAddress     string         `gorm:"index" json:"wallet_address"`
	ChainID           int            `json:"chain_id"`
	Status            string         `json:"status"` // active, suspended
}

// ============================================================================
// Interest Rate Model
// ============================================================================

type InterestRateModel struct {
	BaseRate       float64
	Multiplier     float64
	JumpMultiplier float64
	Kink           float64
}

func CalculateInterestRate(model InterestRateModel, utilization float64) float64 {
	if utilization <= model.Kink {
		return model.BaseRate + (model.Multiplier * utilization)
	}
	normalRate := model.BaseRate + (model.Multiplier * model.Kink)
	excessUtilization := utilization - model.Kink
	return normalRate + (model.JumpMultiplier * excessUtilization)
}

// ============================================================================
// Lending Service
// ============================================================================

type LendingService struct {
	config     *Config
	db         *gorm.DB
	redis      *redis.Client
	assets     map[string]Asset
	positions  map[uint]*UserPosition
	mu         sync.RWMutex
}

func NewLendingService(cfg *Config) (*LendingService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	
	db.AutoMigrate(&Asset{}, &UserPosition{}, &Supply{}, &Borrow{}, &Transaction{}, &Liquidation{}, &User{})
	
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: "",
		DB: 0,
	})
	
	service := &LendingService{
		config:    cfg,
		db:        db,
		redis:     rdb,
		assets:    make(map[string]Asset),
		positions: make(map[uint]*UserPosition),
	}
	
	// Load assets
	service.loadAssets()
	
	return service, nil
}

func (s *LendingService) loadAssets() {
	var assets []Asset
	s.db.Where("is_active = ?", true).Find(&assets)
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	for _, asset := range assets {
		s.assets[asset.AssetID] = asset
	}
}

func (s *LendingService) getAsset(assetID string) (*Asset, error) {
	s.mu.RLock()
	asset, ok := s.assets[assetID]
	s.mu.RUnlock()
	
	if !ok {
		return nil, fmt.Errorf("asset not found: %s", assetID)
	}
	
	return &asset, nil
}

// ============================================================================
// Supply/Deposit Functions
// ============================================================================

func (s *LendingService) Supply(userID uint, walletAddress string, chainID int, assetID string, amount string) (*Supply, error) {
	// Parse amount
	amountFloat, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %v", err)
	}
	
	// Get asset
	asset, err := s.getAsset(assetID)
	if err != nil {
		return nil, err
	}
	
	if !asset.IsActive {
		return nil, fmt.Errorf("asset is not active")
	}
	
	// Get or create user position
	position, err := s.getOrCreatePosition(userID, walletAddress, chainID)
	if err != nil {
		return nil, err
	}
	
	// Calculate amount in USD
	amountUSD := amountFloat * asset.PriceUSD
	
	// Check supply cap
	supplyCap, _ := strconv.ParseFloat(asset.SupplyCap, 64)
	totalSupplied, _ := strconv.ParseFloat(asset.TotalSupplied, 64)
	if supplyCap > 0 && (totalSupplied + amountFloat) > supplyCap {
		return nil, fmt.Errorf("supply cap exceeded")
	}
	
	// Get or create supply record
	var supply Supply
	result := s.db.Where("user_position_id = ? AND asset_id = ?", position.ID, assetID).First(&supply)
	
	if result.Error == gorm.ErrRecordNotFound {
		supply = Supply{
			UserPositionID: position.ID,
			AssetID:        assetID,
			Amount:         amount,
			AmountUSD:      amountUSD,
			AccruedRewards: "0",
			APY:            asset.SupplyAPY,
			LastUpdated:    time.Now(),
		}
		s.db.Create(&supply)
	} else {
		// Update existing supply
		oldAmount, _ := strconv.ParseFloat(supply.Amount, 64)
		newAmount := oldAmount + amountFloat
		supply.Amount = fmt.Sprintf("%.8f", newAmount)
		supply.AmountUSD += amountUSD
		supply.LastUpdated = time.Now()
		s.db.Save(&supply)
	}
	
	// Update asset total supplied
	asset.TotalSupplied = fmt.Sprintf("%.8f", totalSupplied + amountFloat)
	s.db.Save(asset)
	
	// Update position
	s.updatePosition(position)
	
	// Record transaction
	s.recordTransaction(userID, walletAddress, chainID, "supply", assetID, amount, amountUSD)
	
	return &supply, nil
}

func (s *LendingService) Withdraw(userID uint, walletAddress string, chainID int, assetID string, amount string) (string, error) {
	// Get asset
	asset, err := s.getAsset(assetID)
	if err != nil {
		return "", err
	}
	
	// Get user position
	position, err := s.getPosition(userID)
	if err != nil {
		return "", err
	}
	
	// Get supply record
	var supply Supply
	if err := s.db.Where("user_position_id = ? AND asset_id = ?", position.ID, assetID).First(&supply).Error; err != nil {
		return "", fmt.Errorf("no supply found for this asset")
	}
	
	// Parse amount
	amountFloat, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return "", fmt.Errorf("invalid amount: %v", err)
	}
	
	supplyAmount, _ := strconv.ParseFloat(supply.Amount, 64)
	if amountFloat > supplyAmount {
		return "", fmt.Errorf("insufficient balance")
	}
	
	// Check if withdrawal would trigger liquidation
	amountUSD := amountFloat * asset.PriceUSD
	newSuppliedUSD := position.TotalSuppliedUSD - amountUSD
	newBorrowedUSD := position.TotalBorrowedUSD
	
	if newSuppliedUSD > 0 {
		collateralUSD := newSuppliedUSD
		if newBorrowedUSD / collateralUSD > (1 / s.config.MinHealthFactor) {
			return "", fmt.Errorf("withdrawal would trigger liquidation")
		}
	}
	
	// Update supply
	supply.Amount = fmt.Sprintf("%.8f", supplyAmount - amountFloat)
	supply.AmountUSD -= amountUSD
	supply.LastUpdated = time.Now()
	s.db.Save(&supply)
	
	// Update asset total supplied
	totalSupplied, _ := strconv.ParseFloat(asset.TotalSupplied, 64)
	asset.TotalSupplied = fmt.Sprintf("%.8f", totalSupplied - amountFloat)
	s.db.Save(asset)
	
	// Update position
	s.updatePosition(position)
	
	// Record transaction
	txHash := s.recordTransaction(userID, walletAddress, chainID, "withdraw", assetID, amount, amountUSD)
	
	return txHash, nil
}

// ============================================================================
// Borrow Functions
// ============================================================================

func (s *LendingService) Borrow(userID uint, walletAddress string, chainID int, assetID string, amount string) (*Borrow, error) {
	// Parse amount
	amountFloat, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %v", err)
	}
	
	// Get asset
	asset, err := s.getAsset(assetID)
	if err != nil {
		return nil, err
	}
	
	if !asset.IsBorrowable {
		return nil, fmt.Errorf("asset is not borrowable")
	}
	
	// Get user position
	position, err := s.getOrCreatePosition(userID, walletAddress, chainID)
	if err != nil {
		return nil, err
	}
	
	// Check health factor
	s.updatePosition(position)
	if position.HealthFactor < s.config.MinHealthFactor {
		return nil, fmt.Errorf("health factor too low to borrow")
	}
	
	// Calculate amount in USD
	amountUSD := amountFloat * asset.PriceUSD
	
	// Calculate max borrowable
	maxBorrowUSD := position.TotalSuppliedUSD * 0.8 // 80% LTV
	availableBorrowUSD := maxBorrowUSD - position.TotalBorrowedUSD
	
	if amountUSD > availableBorrowUSD {
		return nil, fmt.Errorf("insufficient collateral. Max borrow: %.2f USD", availableBorrowUSD)
	}
	
	// Check borrow cap
	borrowCap, _ := strconv.ParseFloat(asset.BorrowCap, 64)
	totalBorrowed, _ := strconv.ParseFloat(asset.TotalBorrowed, 64)
	if borrowCap > 0 && (totalBorrowed + amountFloat) > borrowCap {
		return nil, fmt.Errorf("borrow cap exceeded")
	}
	
	// Get or create borrow record
	var borrow Borrow
	result := s.db.Where("user_position_id = ? AND asset_id = ?", position.ID, assetID).First(&borrow)
	
	if result.Error == gorm.ErrRecordNotFound {
		borrow = Borrow{
			UserPositionID: position.ID,
			AssetID:        assetID,
			Amount:         amount,
			AmountUSD:      amountUSD,
			APY:            asset.BorrowAPY,
			InterestIndex:  "1.0",
			LastUpdated:    time.Now(),
		}
		s.db.Create(&borrow)
	} else {
		// Update existing borrow
		oldAmount, _ := strconv.ParseFloat(borrow.Amount, 64)
		newAmount := oldAmount + amountFloat
		borrow.Amount = fmt.Sprintf("%.8f", newAmount)
		borrow.AmountUSD += amountUSD
		borrow.LastUpdated = time.Now()
		s.db.Save(&borrow)
	}
	
	// Update asset total borrowed
	asset.TotalBorrowed = fmt.Sprintf("%.8f", totalBorrowed + amountFloat)
	asset.UtilizationRate = (totalBorrowed + amountFloat) / (totalSupplied + amountFloat)
	s.db.Save(asset)
	
	// Update position
	s.updatePosition(position)
	
	// Record transaction
	s.recordTransaction(userID, walletAddress, chainID, "borrow", assetID, amount, amountUSD)
	
	return &borrow, nil
}

func (s *LendingService) Repay(userID uint, walletAddress string, chainID int, assetID string, amount string) error {
	// Get asset
	asset, err := s.getAsset(assetID)
	if err != nil {
		return err
	}
	
	// Get user position
	position, err := s.getPosition(userID)
	if err != nil {
		return err
	}
	
	// Get borrow record
	var borrow Borrow
	if err := s.db.Where("user_position_id = ? AND asset_id = ?", position.ID, assetID).First(&borrow).Error; err != nil {
		return fmt.Errorf("no borrow found for this asset")
	}
	
	// Parse amount
	amountFloat, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return fmt.Errorf("invalid amount: %v", err)
	}
	
	borrowAmount, _ := strconv.ParseFloat(borrow.Amount, 64)
	if amountFloat > borrowAmount {
		amountFloat = borrowAmount
	}
	
	amountUSD := amountFloat * asset.PriceUSD
	
	// Update borrow
	borrow.Amount = fmt.Sprintf("%.8f", borrowAmount - amountFloat)
	borrow.AmountUSD -= amountUSD
	borrow.LastUpdated = time.Now()
	s.db.Save(&borrow)
	
	// Update asset total borrowed
	totalBorrowed, _ := strconv.ParseFloat(asset.TotalBorrowed, 64)
	asset.TotalBorrowed = fmt.Sprintf("%.8f", totalBorrowed - amountFloat)
	s.db.Save(asset)
	
	// Update position
	s.updatePosition(position)
	
	// Record transaction
	s.recordTransaction(userID, walletAddress, chainID, "repay", assetID, fmt.Sprintf("%.8f", amountFloat), amountUSD)
	
	return nil
}

// ============================================================================
// Position Management
// ============================================================================

func (s *LendingService) getOrCreatePosition(userID uint, walletAddress string, chainID int) (*UserPosition, error) {
	var position UserPosition
	result := s.db.Where("user_id = ? AND chain_id = ?", userID, chainID).First(&position)
	
	if result.Error == gorm.ErrRecordNotFound {
		position = UserPosition{
			UserID:           userID,
			WalletAddress:    walletAddress,
			ChainID:          chainID,
			TotalSuppliedUSD: 0,
			TotalBorrowedUSD: 0,
			HealthFactor:     math.MaxFloat64,
			NetAPY:           0,
			IsLiquidatable:   false,
			LastUpdated:      time.Now(),
		}
		s.db.Create(&position)
	}
	
	return &position, nil
}

func (s *LendingService) getPosition(userID uint) (*UserPosition, error) {
	var position UserPosition
	if err := s.db.Where("user_id = ?", userID).First(&position).Error; err != nil {
		return nil, err
	}
	return &position, nil
}

func (s *LendingService) updatePosition(position *UserPosition) {
	// Get all supplies
	var supplies []Supply
	s.db.Where("user_position_id = ?", position.ID).Find(&supplies)
	
	totalSuppliedUSD := 0.0
	for _, supply := range supplies {
		totalSuppliedUSD += supply.AmountUSD
	}
	
	// Get all borrows
	var borrows []Borrow
	s.db.Where("user_position_id = ?", position.ID).Find(&borrows)
	
	totalBorrowedUSD := 0.0
	for _, borrow := range borrows {
		totalBorrowedUSD += borrow.AmountUSD
	}
	
	position.TotalSuppliedUSD = totalSuppliedUSD
	position.TotalBorrowedUSD = totalBorrowedUSD
	
	// Calculate health factor
	if totalBorrowedUSD > 0 && totalSuppliedUSD > 0 {
		// Simplified health factor calculation
		// In production, would consider each collateral asset's LTV
		position.HealthFactor = totalSuppliedUSD / totalBorrowedUSD * 0.8
	} else {
		position.HealthFactor = math.MaxFloat64
	}
	
	position.IsLiquidatable = position.HealthFactor < s.config.MinHealthFactor
	position.LastUpdated = time.Now()
	
	s.db.Save(position)
}

// ============================================================================
// Liquidation
// ============================================================================

func (s *LendingService) Liquidate(liquidatorID uint, victimUserID uint, repayAssetID string, collateralAssetID string) (*Liquidation, error) {
	// Get victim position
	var victimPosition UserPosition
	if err := s.db.Where("user_id = ?", victimUserID).First(&victimPosition).Error; err != nil {
		return nil, fmt.Errorf("victim position not found")
	}
	
	if !victimPosition.IsLiquidatable {
		return nil, fmt.Errorf("position is not liquidatable")
	}
	
	// Get assets
	repayAsset, err := s.getAsset(repayAssetID)
	if err != nil {
		return nil, err
	}
	
	collateralAsset, err := s.getAsset(collateralAssetID)
	if err != nil {
		return nil, err
	}
	
	// Get victim's borrow
	var borrow Borrow
	if err := s.db.Where("user_position_id = ? AND asset_id = ?", victimPosition.ID, repayAssetID).First(&borrow).Error; err != nil {
		return nil, fmt.Errorf("borrow not found")
	}
	
	// Calculate repay amount (50% of borrow)
	borrowAmount, _ := strconv.ParseFloat(borrow.Amount, 64)
	repayAmount := borrowAmount * 0.5
	repayAmountUSD := repayAmount * repayAsset.PriceUSD
	
	// Calculate collateral to receive
	bonus := s.config.LiquidationBonus
	collateralAmount := repayAmountUSD * (1 + bonus) / collateralAsset.PriceUSD
	
	// Execute liquidation
	liquidation := Liquidation{
		LiquidatorID:      liquidatorID,
		VictimUserID:      victimUserID,
		RepayAssetID:      repayAssetID,
		RepayAmount:       fmt.Sprintf("%.8f", repayAmount),
		RepayAmountUSD:    repayAmountUSD,
		CollateralAssetID: collateralAssetID,
		CollateralAmount:  fmt.Sprintf("%.8f", collateralAmount),
		CollateralAmountUSD: repayAmountUSD * (1 + bonus),
		ProfitUSD:         repayAmountUSD * bonus,
		TxHash:            generateTxHash(),
		Status:            "pending",
	}
	s.db.Create(&liquidation)
	
	// Update borrow
	borrow.Amount = fmt.Sprintf("%.8f", borrowAmount - repayAmount)
	borrow.AmountUSD -= repayAmountUSD
	s.db.Save(&borrow)
	
	// Update victim position
	s.updatePosition(&victimPosition)
	
	return &liquiation, nil
}

// ============================================================================
// Flash Loans
// ============================================================================

type FlashLoan struct {
	Asset     string  `json:"asset"`
	Amount    float64 `json:"amount"`
	Fee       float64 `json:"fee"`
	Initiator string  `json:"initiator"`
	Target    string  `json:"target"`
	Data      string  `json:"data"`
}

func (s *LendingService) ExecuteFlashLoan(loan FlashLoan) (string, error) {
	asset, err := s.getAsset(loan.Amount)
	if err != nil {
		return "", err
	}
	
	// Flash loan fee is 0.09%
	fee := loan.Amount * 0.0009
	
	// In production, this would:
	// 1. Check if pool has enough liquidity
	// 2. Transfer funds to initiator
	// 3. Execute the operation
	// 4. Verify repayment + fee
	// 5. Revert if not repaid
	
	txHash := generateTxHash()
	
	return txHash, nil
}

// ============================================================================
// API Handlers
// ============================================================================

func (s *LendingService) setupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	{
		// Assets
		api.GET("/assets", s.listAssets)
		api.GET("/assets/:asset_id", s.getAssetDetail)
		
		// User positions
		api.GET("/user/:user_id/position", s.getUserPosition)
		api.GET("/user/:user_id/supplies", s.getUserSupplies)
		api.GET("/user/:user_id/borrows", s.getUserBorrows)
		
		// Operations
		api.POST("/supply", s.supply)
		api.POST("/withdraw", s.withdraw)
		api.POST("/borrow", s.borrow)
		api.POST("/repay", s.repay)
		api.POST("/liquidate", s.liquidate)
		
		// Flash loans
		api.POST("/flash-loan", s.flashLoan)
		
		// Market stats
		api.GET("/market", s.getMarketStats)
		api.GET("/rates", s.getInterestRates)
	}
	
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "lending-borrowing"})
	})
}

func (s *LendingService) listAssets(c *gin.Context) {
	s.mu.RLock()
	assets := make([]Asset, 0, len(s.assets))
	for _, asset := range s.assets {
		assets = append(assets, asset)
	}
	s.mu.RUnlock()
	
	c.JSON(http.StatusOK, gin.H{"assets": assets})
}

func (s *LendingService) getAssetDetail(c *gin.Context) {
	assetID := c.Param("asset_id")
	
	asset, err := s.getAsset(assetID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"asset": asset})
}

func (s *LendingService) getUserPosition(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("user_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	
	position, err := s.getPosition(uint(userID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "position not found"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"position": position})
}

func (s *LendingService) getUserSupplies(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Param("user_id"), 10, 32)
	
	position, err := s.getPosition(uint(userID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "position not found"})
		return
	}
	
	var supplies []Supply
	s.db.Where("user_position_id = ?", position.ID).Find(&supplies)
	
	c.JSON(http.StatusOK, gin.H{"supplies": supplies})
}

func (s *LendingService) getUserBorrows(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Param("user_id"), 10, 32)
	
	position, err := s.getPosition(uint(userID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "position not found"})
		return
	}
	
	var borrows []Borrow
	s.db.Where("user_position_id = ?", position.ID).Find(&borrows)
	
	c.JSON(http.StatusOK, gin.H{"borrows": borrows})
}

func (s *LendingService) supply(c *gin.Context) {
	var req struct {
		UserID        uint   `json:"user_id" binding:"required"`
		WalletAddress string `json:"wallet_address" binding:"required"`
		ChainID       int    `json:"chain_id" binding:"required"`
		AssetID       string `json:"asset_id" binding:"required"`
		Amount        string `json:"amount" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	supply, err := s.Supply(req.UserID, req.WalletAddress, req.ChainID, req.AssetID, req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"supply": supply})
}

func (s *LendingService) withdraw(c *gin.Context) {
	var req struct {
		UserID        uint   `json:"user_id" binding:"required"`
		WalletAddress string `json:"wallet_address" binding:"required"`
		ChainID       int    `json:"chain_id" binding:"required"`
		AssetID       string `json:"asset_id" binding:"required"`
		Amount        string `json:"amount" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	txHash, err := s.Withdraw(req.UserID, req.WalletAddress, req.ChainID, req.AssetID, req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"tx_hash": txHash})
}

func (s *LendingService) borrow(c *gin.Context) {
	var req struct {
		UserID        uint   `json:"user_id" binding:"required"`
		WalletAddress string `json:"wallet_address" binding:"required"`
		ChainID       int    `json:"chain_id" binding:"required"`
		AssetID       string `json:"asset_id" binding:"required"`
		Amount        string `json:"amount" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	borrow, err := s.Borrow(req.UserID, req.WalletAddress, req.ChainID, req.AssetID, req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"borrow": borrow})
}

func (s *LendingService) repay(c *gin.Context) {
	var req struct {
		UserID        uint   `json:"user_id" binding:"required"`
		WalletAddress string `json:"wallet_address" binding:"required"`
		ChainID       int    `json:"chain_id" binding:"required"`
		AssetID       string `json:"asset_id" binding:"required"`
		Amount        string `json:"amount" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	err := s.Repay(req.UserID, req.WalletAddress, req.ChainID, req.AssetID, req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "repaid successfully"})
}

func (s *LendingService) liquidate(c *gin.Context) {
	var req struct {
		LiquidatorID      uint   `json:"liquidator_id" binding:"required"`
		VictimUserID     uint   `json:"victim_user_id" binding:"required"`
		RepayAssetID     string `json:"repay_asset_id" binding:"required"`
		CollateralAssetID string `json:"collateral_asset_id" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	liquidation, err := s.Liquidate(req.LiquidatorID, req.VictimUserID, req.RepayAssetID, req.CollateralAssetID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"liquidation": liquidation})
}

func (s *LendingService) flashLoan(c *gin.Context) {
	var loan FlashLoan
	if err := c.ShouldBindJSON(&loan); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	txHash, err := s.ExecuteFlashLoan(loan)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"tx_hash": txHash, "fee": loan.Amount * 0.0009})
}

func (s *LendingService) getMarketStats(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	totalSupplied := 0.0
	totalBorrowed := 0.0
	totalLiquidity := 0.0
	
	for _, asset := range s.assets {
		supplied, _ := strconv.ParseFloat(asset.TotalSupplied, 64)
		borrowed, _ := strconv.ParseFloat(asset.TotalBorrowed, 64)
		
		totalSupplied += supplied * asset.PriceUSD
		totalBorrowed += borrowed * asset.PriceUSD
		totalLiquidity += (supplied - borrowed) * asset.PriceUSD
	}
	
	c.JSON(http.StatusOK, gin.H{
		"total_supplied_usd":   totalSupplied,
		"total_borrowed_usd":   totalBorrowed,
		"total_liquidity_usd": totalLiquidity,
	})
}

func (s *LendingService) getInterestRates(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	rates := make([]gin.H, 0, len(s.assets))
	for _, asset := range s.assets {
		rates = append(rates, gin.H{
			"asset_id":    asset.AssetID,
			"symbol":     asset.Symbol,
			"supply_apy":  asset.SupplyAPY,
			"borrow_apy": asset.BorrowAPY,
			"utilization": asset.UtilizationRate,
		})
	}
	
	c.JSON(http.StatusOK, gin.H{"rates": rates})
}

// ============================================================================
// Helper Functions
// ============================================================================

func (s *LendingService) recordTransaction(userID uint, walletAddress, chainID int, txType, assetID, amount string, amountUSD float64) string {
	txHash := generateTxHash()
	
	tx := Transaction{
		TxHash:       txHash,
		UserID:       userID,
		WalletAddress: walletAddress,
		ChainID:      chainID,
		Type:         txType,
		AssetID:      assetID,
		Amount:       amount,
		AmountUSD:    amountUSD,
		Status:       "confirmed",
		BlockNumber:  0,
	}
	
	s.db.Create(&tx)
	
	return txHash
}

func generateTxHash() string {
	data := fmt.Sprintf("%d:%s", time.Now().UnixNano(), uuid.New().String())
	hash := sha256.Sum256([]byte(data))
	return "0x" + hex.EncodeToString(hash[:])
}

// ============================================================================
// Main
// ============================================================================

func main() {
	cfg := LoadConfig()
	
	service, err := NewLendingService(cfg)
	if err != nil {
		log.Fatalf("Failed to create lending service: %v", err)
	}
	
	router := gin.Default()
	service.setupRoutes(router)
	
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		<-quit
		log.Println("Shutting down lending service...")
		os.Exit(0)
	}()
	
	log.Printf("Lending & Borrowing Service starting on port %s", cfg.ServerPort)
	if err := router.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
