// TigerWallet Savings Service
// High-Load Distributed Go Implementation
// Interest-bearing savings accounts with flexible and fixed terms

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
}

// ============================================================================
// Data Models
// ============================================================================

// SavingsProduct represents a savings product
type SavingsProduct struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	AssetAddress    string    `gorm:"index" json:"asset_address"`
	AssetSymbol     string    `json:"asset_symbol"`
	AssetDecimals   uint8     `json:"asset_decimals"`
	ProductType     string    `json:"product_type"` // FLEXIBLE, FIXED
	TermDays        int       `json:"term_days"` // 0 for flexible
	APY             float64   `json:"apy"`
	MinDeposit      float64   `json:"min_deposit"`
	MaxDeposit      float64   `json:"max_deposit"`
	TotalDeposited  float64   `json:"total_deposited"`
	TotalWithdrawn  float64   `json:"total_withdrawn"`
	MinLockPeriod   int       `json:"min_lock_period"` // seconds
	EarlyWithdrawFee float64  `json:"early_withdraw_fee"` // percentage
	IsActive        bool      `json:"is_active"`
	ChainID         int64     `json:"chain_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// SavingsAccount represents a user's savings account
type SavingsAccount struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserAddress    string    `gorm:"index" json:"user_address"`
	ProductID     uint      `gorm:"index" json:"product_id"`
	AssetAddress  string    `json:"asset_address"`
	Balance       float64   `json:"balance"`
	AccruedInterest float64 `json:"accrued_interest"`
	TotalDeposited float64  `json:"total_deposited"`
	TotalWithdrawn float64  `json:"total_withdrawn"`
	APY           float64   `json:"apy"`
	Status        string    `json:"status"` // ACTIVE, WITHDRAWN, LIQUIDATED
	StartTime     int64     `json:"start_time"`
	TermEndTime   *int64    `json:"term_end_time"`
	ChainID       int64     `json:"chain_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// SavingsTransaction represents deposit/withdraw transactions
type SavingsTransaction struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserAddress   string    `gorm:"index" json:"user_address"`
	ProductID     uint      `json:"product_id"`
	AssetAddress  string    `json:"asset_address"`
	Amount        float64   `json:"amount"`
	Interest      float64   `json:"interest"`
	Type          string    `json:"type"` // DEPOSIT, WITHDRAW, INTEREST
	TxHash        string    `json:"tx_hash"`
	ChainID       int64     `json:"chain_id"`
	CreatedAt     time.Time `json:"created_at"`
}

// ============================================================================
// Service Implementation
// ============================================================================

type SavingsService struct {
	db     *gorm.DB
	redis  *redis.Client
	config Config
	mu     sync.RWMutex
}

func NewSavingsService(config Config) (*SavingsService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	err = db.AutoMigrate(
		&SavingsProduct{},
		&SavingsAccount{},
		&SavingsTransaction{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort),
	})

	service := &SavingsService{
		db:     db,
		redis:  rdb,
		config: config,
	}

	go service.initializeProducts()

	return service, nil
}

func (s *SavingsService) initializeProducts() {
	products := []SavingsProduct{
		{
			Name:             "Flexible USDT Savings",
			Description:      "Earn interest on your USDT with flexible withdrawal",
			AssetAddress:    "0xdAC17F958D2ee523a2206206994597C13D831ec7",
			AssetSymbol:     "USDT",
			AssetDecimals:   6,
			ProductType:     "FLEXIBLE",
			TermDays:        0,
			APY:             4.5,
			MinDeposit:      1,
			MaxDeposit:      1000000,
			MinLockPeriod:   0,
			EarlyWithdrawFee: 0,
			IsActive:        true,
			ChainID:         1,
		},
		{
			Name:             "Flexible USDC Savings",
			Description:      "Earn interest on your USDC with flexible withdrawal",
			AssetAddress:    "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
			AssetSymbol:     "USDC",
			AssetDecimals:   6,
			ProductType:     "FLEXIBLE",
			TermDays:        0,
			APY:             4.2,
			MinDeposit:      1,
			MaxDeposit:      1000000,
			MinLockPeriod:   0,
			EarlyWithdrawFee: 0,
			IsActive:        true,
			ChainID:         1,
		},
		{
			Name:             "Flexible ETH Savings",
			Description:      "Earn interest on your ETH with flexible withdrawal",
			AssetAddress:    "0x0000000000000000000000000000000000000000",
			AssetSymbol:     "ETH",
			AssetDecimals:   18,
			ProductType:     "FLEXIBLE",
			TermDays:        0,
			APY:             3.8,
			MinDeposit:      0.01,
			MaxDeposit:      10000,
			MinLockPeriod:   0,
			EarlyWithdrawFee: 0,
			IsActive:        true,
			ChainID:         1,
		},
		{
			Name:             "30-Day Fixed USDT",
			Description:      "Higher APY with 30-day lock period",
			AssetAddress:    "0xdAC17F958D2ee523a2206206994597C13D831ec7",
			AssetSymbol:     "USDT",
			AssetDecimals:   6,
			ProductType:     "FIXED",
			TermDays:        30,
			APY:             6.5,
			MinDeposit:      100,
			MaxDeposit:      1000000,
			MinLockPeriod:   2592000, // 30 days in seconds
			EarlyWithdrawFee: 0.5, // 0.5% fee
			IsActive:        true,
			ChainID:         1,
		},
		{
			Name:             "60-Day Fixed USDT",
			Description:      "Even higher APY with 60-day lock period",
			AssetAddress:    "0xdAC17F958D2ee523a2206206994597C13D831ec7",
			AssetSymbol:     "USDT",
			AssetDecimals:   6,
			ProductType:     "FIXED",
			TermDays:        60,
			APY:             7.5,
			MinDeposit:      100,
			MaxDeposit:      1000000,
			MinLockPeriod:   5184000, // 60 days in seconds
			EarlyWithdrawFee: 1.0, // 1% fee
			IsActive:        true,
			ChainID:         1,
		},
		{
			Name:             "90-Day Fixed USDT",
			Description:      "Best APY with 90-day lock period",
			AssetAddress:    "0xdAC17F958D2ee523a2206206994597C13D831ec7",
			AssetSymbol:     "USDT",
			AssetDecimals:   6,
			ProductType:     "FIXED",
			TermDays:        90,
			APY:             8.5,
			MinDeposit:      100,
			MaxDeposit:      1000000,
			MinLockPeriod:   7776000, // 90 days in seconds
			EarlyWithdrawFee: 1.5, // 1.5% fee
			IsActive:        true,
			ChainID:         1,
		},
	}

	for _, product := range products {
		var existing SavingsProduct
		if s.db.Where("asset_address = ? AND product_type = ? AND term_days = ?", 
			product.AssetAddress, product.ProductType, product.TermDays).First(&existing).RowsAffected == 0 {
			s.db.Create(&product)
		}
	}
}

// ============================================================================
// Deposit Operations
// ============================================================================

type DepositRequest struct {
	UserAddress   string  `json:"user_address" binding:"required"`
	ProductID     uint    `json:"product_id" binding:"required"`
	Amount        float64 `json:"amount" binding:"required"`
	ChainID       int64   `json:"chain_id"`
}

type DepositResponse struct {
	Success         bool    `json:"success"`
	AccountID       uint    `json:"account_id,omitempty"`
	TransactionHash string  `json:"transaction_hash,omitempty"`
	NewBalance      float64 `json:"new_balance"`
	APY             float64 `json:"apy"`
	InterestEarned  float64 `json:"interest_earned"`
	MaturityDate    *int64  `json:"maturity_date,omitempty"`
	Error           string  `json:"error,omitempty"`
}

func (s *SavingsService) Deposit(ctx *gin.Context) {
	var req DepositRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, DepositResponse{Success: false, Error: err.Error()})
		return
	}

	// Get product
	var product SavingsProduct
	if err := s.db.First(&product, req.ProductID).Error; err != nil {
		ctx.JSON(404, DepositResponse{Success: false, Error: "Product not found"})
		return
	}

	// Validate amount
	if req.Amount < product.MinDeposit {
		ctx.JSON(400, DepositResponse{Success: false, Error: "Below minimum deposit"})
		return
	}
	if req.Amount > product.MaxDeposit {
		ctx.JSON(400, DepositResponse{Success: false, Error: "Exceeds maximum deposit"})
		return
	}

	// Check existing account or create new
	var account SavingsAccount
	result := s.db.Where("user_address = ? AND product_id = ? AND status = ?", 
		req.UserAddress, req.ProductID, "ACTIVE").First(&account)

	var maturityDate *int64
	if product.ProductType == "FIXED" && product.TermDays > 0 {
		maturity := time.Now().Add(time.Duration(product.TermDays) * 24 * time.Hour).Unix()
		maturityDate = &maturity
	}

	if result.RowsAffected == 0 {
		account = SavingsAccount{
			UserAddress:       req.UserAddress,
			ProductID:         req.ProductID,
			AssetAddress:      product.AssetAddress,
			Balance:           req.Amount,
			AccruedInterest:   0,
			TotalDeposited:    req.Amount,
			TotalWithdrawn:    0,
			APY:               product.APY,
			Status:            "ACTIVE",
			StartTime:         time.Now().Unix(),
			TermEndTime:       maturityDate,
			ChainID:           req.ChainID,
		}
		s.db.Create(&account)
	} else {
		account.Balance += req.Amount
		account.TotalDeposited += req.Amount
		if product.ProductType == "FIXED" && maturityDate != nil {
			account.TermEndTime = maturityDate
		}
		s.db.Save(&account)
	}

	// Update product total
	product.TotalDeposited += req.Amount
	s.db.Save(&product)

	// Create transaction record
	txHash := s.generateTxHash(req.UserAddress, product.AssetAddress, req.Amount)
	transaction := SavingsTransaction{
		UserAddress:  req.UserAddress,
		ProductID:   req.ProductID,
		AssetAddress: product.AssetAddress,
		Amount:      req.Amount,
		Type:        "DEPOSIT",
		TxHash:      txHash,
		ChainID:     req.ChainID,
	}
	s.db.Create(&transaction)

	ctx.JSON(200, DepositResponse{
		Success:         true,
		AccountID:       account.ID,
		TransactionHash: txHash,
		NewBalance:      account.Balance,
		APY:             product.APY,
		MaturityDate:    maturityDate,
	})
}

// ============================================================================
// Withdraw Operations
// ============================================================================

type WithdrawRequest struct {
	UserAddress   string  `json:"user_address" binding:"required"`
	AccountID     uint    `json:"account_id" binding:"required"`
	Amount        float64 `json:"amount" binding:"required"`
	ChainID       int64   `json:"chain_id"`
}

type WithdrawResponse struct {
	Success          bool    `json:"success"`
	TransactionHash  string  `json:"transaction_hash,omitempty"`
	AmountReceived   float64 `json:"amount_received"`
	InterestReceived float64 `json:"interest_received"`
	FeeApplied      float64 `json:"fee_applied"`
	NewBalance      float64 `json:"new_balance"`
	Error            string  `json:"error,omitempty"`
}

func (s *SavingsService) Withdraw(ctx *gin.Context) {
	var req WithdrawRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, WithdrawResponse{Success: false, Error: err.Error()})
		return
	}

	var account SavingsAccount
	if err := s.db.First(&account, req.AccountID).Error; err != nil {
		ctx.JSON(404, WithdrawResponse{Success: false, Error: "Account not found"})
		return
	}

	if account.UserAddress != req.UserAddress {
		ctx.JSON(403, WithdrawResponse{Success: false, Error: "Unauthorized"})
		return
	}

	var product SavingsProduct
	if err := s.db.First(&product, account.ProductID).Error; err != nil {
		ctx.JSON(404, WithdrawResponse{Success: false, Error: "Product not found"})
		return
	}

	// Check if locked
	if product.ProductType == "FIXED" && account.TermEndTime != nil {
		currentTime := time.Now().Unix()
		if currentTime < *account.TermEndTime {
			ctx.JSON(400, WithdrawResponse{Success: false, Error: "Lock period not yet ended"})
			return
		}
	}

	// Calculate withdrawal amount
	withdrawAmount := req.Amount
	if withdrawAmount > account.Balance+account.AccruedInterest {
		withdrawAmount = account.Balance + account.AccruedInterest
	}

	// Calculate interest received
	var interestReceived float64
	var feeApplied float64

	if withdrawAmount <= account.AccruedInterest {
		interestReceived = withdrawAmount
	} else {
		interestReceived = account.AccruedInterest
		principalWithdraw := withdrawAmount - interestReceived

		// Check for early withdrawal fee
		if product.ProductType == "FIXED" && account.TermEndTime != nil {
			currentTime := time.Now().Unix()
			if currentTime < *account.TermEndTime {
				feeApplied = principalWithdraw * product.EarlyWithdrawFee / 100
				principalWithdraw -= feeApplied
			}
		}

		account.Balance -= principalWithdraw
	}

	account.AccruedInterest -= interestReceived
	account.TotalWithdrawn += withdrawAmount - feeApplied

	if account.Balance <= 0 {
		account.Status = "WITHDRAWN"
	}

	s.db.Save(&account)

	// Update product total
	product.TotalWithdrawn += withdrawAmount - feeApplied
	s.db.Save(&product)

	txHash := s.generateTxHash(req.UserAddress, product.AssetAddress, withdrawAmount)
	transaction := SavingsTransaction{
		UserAddress:  req.UserAddress,
		ProductID:   req.ProductID,
		AssetAddress: product.AssetAddress,
		Amount:      withdrawAmount,
		Interest:    interestReceived,
		Type:        "WITHDRAW",
		TxHash:      txHash,
		ChainID:     req.ChainID,
	}
	s.db.Create(&transaction)

	ctx.JSON(200, WithdrawResponse{
		Success:          true,
		TransactionHash:  txHash,
		AmountReceived:   withdrawAmount - feeApplied,
		InterestReceived: interestReceived,
		FeeApplied:       feeApplied,
		NewBalance:       account.Balance,
	})
}

// ============================================================================
// Claim Interest
// ============================================================================

func (s *SavingsService) ClaimInterest(ctx *gin.Context) {
	var req struct {
		UserAddress string `json:"user_address" binding:"required"`
		AccountID   uint   `json:"account_id" binding:"required"`
		ChainID     int64  `json:"chain_id"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	var account SavingsAccount
	if err := s.db.First(&account, req.AccountID).Error; err != nil {
		ctx.JSON(404, gin.H{"success": false, "error": "Account not found"})
		return
	}

	if account.AccruedInterest <= 0 {
		ctx.JSON(400, gin.H{"success": false, "error": "No interest to claim"})
		return
	}

	interest := account.AccruedInterest
	account.Balance += interest
	account.AccruedInterest = 0
	s.db.Save(&account)

	txHash := s.generateTxHash(req.UserAddress, account.AssetAddress, interest)

	ctx.JSON(200, gin.H{
		"success":         true,
		"transaction_hash": txHash,
		"amount_claimed":   interest,
	})
}

// ============================================================================
// Queries
// ============================================================================

type AccountResponse struct {
	ID                uint    `json:"id"`
	ProductID         uint    `json:"product_id"`
	ProductName       string  `json:"product_name"`
	AssetSymbol       string  `json:"asset_symbol"`
	Balance           float64 `json:"balance"`
	AccruedInterest   float64 `json:"accrued_interest"`
	APY               float64 `json:"apy"`
	TotalDeposited    float64 `json:"total_deposited"`
	TotalWithdrawn    float64 `json:"total_withdrawn"`
	Status            string  `json:"status"`
	StartTime         int64   `json:"start_time"`
	TermEndTime       *int64  `json:"term_end_time"`
}

func (s *SavingsService) GetAccounts(ctx *gin.Context) {
	userAddress := ctx.Query("user_address")
	chainID := ctx.GetInt64("chain_id")

	if userAddress == "" {
		ctx.JSON(400, gin.H{"error": "user_address required"})
		return
	}

	var accounts []SavingsAccount
	s.db.Where("user_address = ? AND chain_id = ? AND status = ?", userAddress, chainID, "ACTIVE").Find(&accounts)

	response := make([]AccountResponse, len(accounts))
	for i, account := range accounts {
		var product SavingsProduct
		s.db.First(&product, account.ProductID)

		response[i] = AccountResponse{
			ID:              account.ID,
			ProductID:       account.ProductID,
			ProductName:     product.Name,
			AssetSymbol:     product.AssetSymbol,
			Balance:         account.Balance,
			AccruedInterest: account.AccruedInterest,
			APY:             account.APY,
			TotalDeposited:  account.TotalDeposited,
			TotalWithdrawn:  account.TotalWithdrawn,
			Status:          account.Status,
			StartTime:       account.StartTime,
			TermEndTime:     account.TermEndTime,
		}
	}

	ctx.JSON(200, gin.H{"accounts": response})
}

func (s *SavingsService) GetProducts(ctx *gin.Context) {
	var products []SavingsProduct
	s.db.Where("is_active = ?", true).Find(&products)

	ctx.JSON(200, gin.H{"products": products})
}

// ============================================================================
// Interest Calculation (Called periodically)
// ============================================================================

func (s *SavingsService) CalculateInterest() {
	var accounts []SavingsAccount
	s.db.Where("status = ?", "ACTIVE").Find(&accounts)

	for _, account := range accounts {
		var product SavingsProduct
		if err := s.db.First(&product, account.ProductID).Error; err != nil {
			continue
		}

		// Daily interest = balance * APY / 365
		dailyInterest := account.Balance * (product.APY / 365 / 100)
		account.AccruedInterest += dailyInterest
		s.db.Save(&account)
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

func (s *SavingsService) generateTxHash(user, asset string, amount float64) string {
	data := fmt.Sprintf("%s:%s:%f:%d", user, asset, amount, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return "0x" + hex.EncodeToString(hash[:])
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := Config{
		ServerPort: "8092",
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "tigerwallet"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "tigerwallet_savings"),
		RedisHost:  getEnv("REDIS_HOST", "localhost"),
		RedisPort:  getEnv("REDIS_PORT", "6379"),
	}

	service, err := NewSavingsService(config)
	if err != nil {
		fmt.Printf("Failed to start savings service: %v\n", err)
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

	api := router.Group("/api/v1/savings")
	{
		api.GET("/products", service.GetProducts)
		api.GET("/accounts", service.GetAccounts)
		api.POST("/deposit", service.Deposit)
		api.POST("/withdraw", service.Withdraw)
		api.POST("/claim", service.ClaimInterest)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "savings"})
	})

	go func() {
		fmt.Printf("Savings service starting on port %s\n", config.ServerPort)
		if err := router.Run(":" + config.ServerPort); err != nil {
			fmt.Printf("Failed to start server: %v\n", err)
		}
	}()

	// Interest calculation loop (every hour)
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			service.CalculateInterest()
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down savings service...")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
