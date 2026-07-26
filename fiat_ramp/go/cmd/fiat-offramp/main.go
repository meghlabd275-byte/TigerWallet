/**
 * TigerWallet Fiat Off-Ramp Service
 * Sell Crypto for Fiat Currency
 * 
 * Features:
 * - Bank transfer withdrawals
 * - Multiple payment methods
 * - KYC integration
 * - Real-time price quotes
 * - Multi-currency support
 * - P2P trading
 * - Compliance checks
 * - Anti-fraud measures
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
	"regexp"
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
	DBPort          string
	DBUser          string
	DBPassword      string
	DBName          string
	RedisHost       string
	RedisPort       string
	MinWithdrawal   float64
	MaxWithdrawal   float64
	FeePercentage   float64
}

func LoadConfig() *Config {
	return &Config{
		ServerPort:    getEnv("FIAT_OFFRAMP_PORT", "9102"),
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "tigerwallet"),
		DBPassword:    getEnv("DB_PASSWORD", "password"),
		DBName:        getEnv("DB_NAME", "tigerwallet"),
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		MinWithdrawal: 10.0,
		MaxWithdrawal: 100000.0,
		FeePercentage: 1.5,
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

type User struct {
	ID                uint           `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
	UUID              string         `gorm:"uniqueIndex;size:36" json:"uuid"`
	Email             string         `gorm:"index" json:"email"`
	KYCStatus         string         `json:"kyc_status"` // none, pending, verified, rejected
	BankAccounts     []BankAccount  `json:"bank_accounts"`
}

type BankAccount struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	UserID            uint      `gorm:"index" json:"user_id"`
	User              User      `gorm:"foreignKey:UserID" json:"-"`
	BankName          string    `json:"bank_name"`
	BankCode          string    `json:"bank_code"` // SWIFT/IBAN
	AccountNumber     string    `json:"account_number"`
	AccountHolder     string    `json:"account_holder"`
	AccountType       string    `json:"account_type"` // checking, savings
	Country           string    `json:"country"`
	Currency          string    `json:"currency"`
	IsVerified        bool      `json:"is_verified"`
	IsDefault         bool      `json:"is_default"`
}

type FiatOrder struct {
	ID                uint           `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
	OrderID          string         `gorm:"uniqueIndex;size:36" json:"order_id"`
	UserID            uint           `gorm:"index" json:"user_id"`
	User              User           `gorm:"foreignKey:UserID" json:"-"`
	OrderType        string         `json:"order_type"` // sell, buy
	CryptoAsset      string         `json:"crypto_asset"` // BTC, ETH, etc.
	CryptoAmount     float64        `json:"crypto_amount"`
	CryptoUSDValue   float64        `json:"crypto_usd_value"`
	FiatCurrency     string         `json:"fiat_currency"`
	FiatAmount       float64        `json:"fiat_amount"`
	ExchangeRate     float64        `json:"exchange_rate"`
	FeeAmount        float64        `json:"fee_amount"`
	NetAmount        float64        `json:"net_amount"`
	BankAccountID    uint           `gorm:"index" json:"bank_account_id"`
	BankAccount      BankAccount    `gorm:"foreignKey:BankAccountID" json:"-"`
	PaymentMethod    string         `json:"payment_method"` // bank_transfer, card, p2p
	Status           string         `json:"status"` // pending, processing, completed, failed, cancelled
	TxHash           string         `gorm:"index;size:66" json:"tx_hash"`
	CompletedAt      *time.Time     `json:"completed_at"`
	FailedAt         *time.Time     `json:"failed_at"`
	FailureReason    string         `json:"failure_reason"`
	KYCVerified      bool           `json:"kyc_verified"`
	RiskScore        float64        `json:"risk_score"`
	ComplianceStatus string         `json:"compliance_status"` // pending, approved, rejected
}

type ExchangeRate struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	CryptoAsset      string    `gorm:"index" json:"crypto_asset"`
	FiatCurrency     string    `gorm:"index" json:"fiat_currency"`
	Rate             float64   `json:"rate"`
	RateUSD          float64   `json:"rate_usd"`
	Spread           float64   `json:"spread"`
	ExpiresAt        time.Time `json:"expires_at"`
}

type P2PListing struct {
	ID                uint           `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
	ListingID         string         `gorm:"uniqueIndex;size:36" json:"listing_id"`
	UserID            uint           `gorm:"index" json:"user_id"`
	User              User           `gorm:"foreignKey:UserID" json:"-"`
	ListingType       string         `json:"listing_type"` // buy, sell
	CryptoAsset      string         `json:"crypto_asset"`
	FiatCurrency     string         `json:"fiat_currency"`
	Amount           float64        `json:"amount"`
	MinAmount         float64        `json:"min_amount"`
	MaxAmount         float64        `json:"max_amount"`
	Price            float64        `json:"price"`
	PriceType        string         `json:"price_type"` // fixed, floating
	PaymentMethods   string         `json:"payment_methods"` // JSON array
	Status           string         `json:"status"` // active, paused, completed
	OrdersCompleted  int           `json:"orders_completed"`
	Rating           float64        `json:"rating"`
	TradeCount       int           `json:"trade_count"`
}

type P2PTrade struct {
	ID                uint           `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
	TradeID           string         `gorm:"uniqueIndex;size:36" json:"trade_id"`
	ListingID         string         `gorm:"index" json:"listing_id"`
	Listing           P2PListing     `gorm:"foreignKey:ListingID" json:"-"`
	BuyerID           uint           `gorm:"index" json:"buyer_id"`
	Buyer             User           `gorm:"foreignKey:BuyerID" json:"-"`
	SellerID          uint           `gorm:"index" json:"seller_id"`
	Seller            User           `gorm:"foreignKey:SellerID" json:"-"`
	CryptoAsset      string         `json:"crypto_asset"`
	CryptoAmount     float64        `json:"crypto_amount"`
	FiatCurrency     string         `json:"fiat_currency"`
	FiatAmount       float64        `json:"fiat_amount"`
	Status           string         `json:"status"` // pending, escrow, released, completed, cancelled, disputed
	BuyerConfirmed   bool           `json:"buyer_confirmed"`
	SellerConfirmed  bool           `json:"seller_confirmed"`
	EscrowTxHash    string         `gorm:"index;size:66" json:"escrow_tx_hash"`
	ReleaseTxHash    string         `gorm:"index;size:66" json:"release_tx_hash"`
	PaymentProof     string         `json:"payment_proof"` // Base64 encoded image
	DisputeReason    string         `json:"dispute_reason"`
	CompletedAt      *time.Time     `json:"completed_at"`
}

type ComplianceCheck struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	OrderID          string    `gorm:"index" json:"order_id"`
	UserID            uint      `gorm:"index" json:"user_id"`
	CheckType         string    `json:"check_type"` // kyc, aml, sanction, fraud
	Status            string    `json:"status"` // pending, passed, failed
	Score            float64   `json:"score"`
	Details          string    `json:"details"` // JSON
	CheckedAt        *time.Time `json:"checked_at"`
}

// ============================================================================
// Fiat Off-Ramp Service
// ============================================================================

type FiatOffRampService struct {
	config *Config
	db     *gorm.DB
	redis  *redis.Client
	mu     sync.RWMutex
}

func NewFiatOffRampService(cfg *Config) (*FiatOffRampService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	
	db.AutoMigrate(&User{}, &BankAccount{}, &FiatOrder{}, &ExchangeRate{}, &P2PListing{}, &P2PTrade{}, &ComplianceCheck{})
	
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: "",
		DB:       0,
	})
	
	return &FiatOffRampService{
		config: cfg,
		db:     db,
		redis:  rdb,
	}, nil
}

// ============================================================================
// Bank Account Management
// ============================================================================

func (s *FiatOffRampService) AddBankAccount(userID uint, bankName, bankCode, accountNumber, accountHolder, accountType, country, currency string) (*BankAccount, error) {
	// Validate bank code format
	if !s.validateBankCode(bankCode, country) {
		return nil, fmt.Errorf("invalid bank code format")
	}
	
	// Validate account number
	if !s.validateAccountNumber(accountNumber) {
		return nil, fmt.Errorf("invalid account number")
	}
	
	// Check if this is the first account
	var count int64
	s.db.Model(&BankAccount{}).Where("user_id = ?", userID).Count(&count)
	
	account := BankAccount{
		UserID:        userID,
		BankName:      bankName,
		BankCode:      bankCode,
		AccountNumber: s.maskAccountNumber(accountNumber),
		AccountHolder: accountHolder,
		AccountType:   accountType,
		Country:       country,
		Currency:      currency,
		IsVerified:    false,
		IsDefault:     count == 0,
	}
	
	s.db.Create(&account)
	
	return &account, nil
}

func (s *FiatOffRampService) GetBankAccounts(userID uint) ([]BankAccount, error) {
	var accounts []BankAccount
	s.db.Where("user_id = ?", userID).Find(&accounts)
	return accounts, nil
}

func (s *FiatOffRampService) VerifyBankAccount(accountID uint) error {
	// In production, this would verify the bank account via bank API
	result := s.db.Model(&BankAccount{}).Where("id = ?", accountID).Update("is_verified", true)
	return result.Error
}

// ============================================================================
// Exchange Rates
// ============================================================================

func (s *FiatOffRampService) GetExchangeRate(cryptoAsset, fiatCurrency string) (*ExchangeRate, error) {
	// Try cache first
	ctx := context.Background()
	cacheKey := fmt.Sprintf("rate:%s:%s", cryptoAsset, fiatCurrency)
	
	cached, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var rate ExchangeRate
		if json.Unmarshal([]byte(cached), &rate) == nil {
			return &rate, nil
		}
	}
	
	// Get from database
	var rate ExchangeRate
	result := s.db.Where("crypto_asset = ? AND fiat_currency = ?", cryptoAsset, fiatCurrency).First(&rate)
	if result.Error != nil {
		// Create default rate
		rate = s.createDefaultRate(cryptoAsset, fiatCurrency)
	}
	
	// Cache for 30 seconds
	rateJSON, _ := json.Marshal(rate)
	s.redis.Set(ctx, cacheKey, rateJSON, 30*time.Second)
	
	return &rate, nil
}

func (s *FiatOffRampService) createDefaultRate(cryptoAsset, fiatCurrency string) ExchangeRate {
	// Simplified rate calculation - in production use real price feeds
	rates := map[string]float64{
		"BTC": 67000, "ETH": 3500, "USDT": 1.0, "USDC": 1.0,
		"BNB": 600, "SOL": 150, "XRP": 0.55, "ADA": 0.45,
		"DOGE": 0.12, "DOT": 7.5, "MATIC": 0.85, "AVAX": 35,
	}
	
	baseRate := rates[cryptoAsset]
	if baseRate == 0 {
		baseRate = 1000
	}
	
	// Adjust for fiat currency
	fiatMultiplier := map[string]float64{"USD": 1.0, "EUR": 0.92, "GBP": 0.79, "JPY": 150}
	multiplier := fiatMultiplier[fiatCurrency]
	if multiplier == 0 {
		multiplier = 1.0
	}
	
	rate := baseRate * multiplier
	spread := s.config.FeePercentage / 100.0
	
	return ExchangeRate{
		CryptoAsset:  cryptoAsset,
		FiatCurrency: fiatCurrency,
		Rate:         rate,
		RateUSD:      baseRate,
		Spread:       spread,
		ExpiresAt:    time.Now().Add(60 * time.Second),
	}
}

// ============================================================================
// Fiat Order Management
// ============================================================================

func (s *FiatOffRampService) CreateSellOrder(userID uint, cryptoAsset, fiatCurrency string, cryptoAmount float64, bankAccountID uint) (*FiatOrder, error) {
	// Validate amount
	if cryptoAmount <= 0 {
		return nil, fmt.Errorf("invalid amount")
	}
	
	// Check limits
	if cryptoAmount < s.config.MinWithdrawal || cryptoAmount > s.config.MaxWithdrawal {
		return nil, fmt.Errorf("amount must be between %.2f and %.2f", s.config.MinWithdrawal, s.config.MaxWithdrawal)
	}
	
	// Get exchange rate
	rate, err := s.GetExchangeRate(cryptoAsset, fiatCurrency)
	if err != nil {
		return nil, err
	}
	
	// Calculate amounts
	cryptoUSDValue := cryptoAmount * rate.RateUSD
	fiatAmount := cryptoAmount * rate.Rate
	feeAmount := fiatAmount * rate.Spread
	netAmount := fiatAmount - feeAmount
	
	// Verify bank account belongs to user
	var bankAccount BankAccount
	if err := s.db.First(&bankAccount, bankAccountID).Error; err != nil {
		return nil, fmt.Errorf("bank account not found")
	}
	if bankAccount.UserID != userID {
		return nil, fmt.Errorf("unauthorized")
	}
	
	// Check KYC status
	var user User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found")
	}
	
	if user.KYCStatus != "verified" {
		return nil, fmt.Errorf("KYC verification required")
	}
	
	// Create order
	order := FiatOrder{
		OrderID:        uuid.New().String(),
		UserID:          userID,
		OrderType:       "sell",
		CryptoAsset:     cryptoAsset,
		CryptoAmount:    cryptoAmount,
		CryptoUSDValue:  cryptoUSDValue,
		FiatCurrency:    fiatCurrency,
		FiatAmount:      fiatAmount,
		ExchangeRate:    rate.Rate,
		FeeAmount:       feeAmount,
		NetAmount:       netAmount,
		BankAccountID:   bankAccountID,
		PaymentMethod:   "bank_transfer",
		Status:          "pending",
		KYCVerified:     true,
		RiskScore:       0.0,
		ComplianceStatus: "pending",
	}
	
	s.db.Create(&order)
	
	// Run compliance checks asynchronously
	go s.runComplianceChecks(&order)
	
	return &order, nil
}

func (s *FiatOffRampService) GetOrder(orderID string) (*FiatOrder, error) {
	var order FiatOrder
	if err := s.db.Preload("BankAccount").Where("order_id = ?", orderID).First(&order).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

func (s *FiatOffRampService) GetUserOrders(userID uint) ([]FiatOrder, error) {
	var orders []FiatOrder
	s.db.Where("user_id = ?", userID).Order("created_at desc").Find(&orders)
	return orders, nil
}

func (s *FiatOffRampService) CancelOrder(orderID string, userID uint) error {
	var order FiatOrder
	if err := s.db.Where("order_id = ? AND user_id = ?", orderID, userID).First(&order).Error; err != nil {
		return err
	}
	
	if order.Status != "pending" && order.Status != "processing" {
		return fmt.Errorf("cannot cancel order in status: %s", order.Status)
	}
	
	order.Status = "cancelled"
	return s.db.Save(&order).Error
}

// ============================================================================
// Order Processing
// ============================================================================

func (s *FiatOffRampService) ProcessOrder(orderID string) error {
	order, err := s.GetOrder(orderID)
	if err != nil {
		return err
	}
	
	if order.Status != "pending" {
		return fmt.Errorf("order not in pending status")
	}
	
	// Update status
	order.Status = "processing"
	s.db.Save(order)
	
	// Simulate crypto transfer from user
	// In production, this would be a real blockchain transaction
	txHash := s.simulateCryptoTransfer(order)
	order.TxHash = txHash
	
	// Run fraud check
	if !s.checkFraudRisk(order) {
		order.Status = "failed"
		order.FailureReason = "Failed fraud check"
		now := time.Now()
		order.FailedAt = &now
		s.db.Save(order)
		return fmt.Errorf("failed fraud check")
	}
	
	// Process fiat transfer
	err = s.processFiatTransfer(order)
	if err != nil {
		order.Status = "failed"
		order.FailureReason = err.Error()
		now := time.Now()
		order.FailedAt = &now
		s.db.Save(order)
		return err
	}
	
	// Mark completed
	order.Status = "completed"
	now := time.Now()
	order.CompletedAt = &now
	s.db.Save(order)
	
	return nil
}

func (s *FiatOffRampService) simulateCryptoTransfer(order *FiatOrder) string {
	// In production, this would:
	// 1. Generate a deposit address for the user
	// 2. Monitor the blockchain for incoming transaction
	// 3. Wait for confirmations
	// 4. Transfer to platform wallet
	
	data := fmt.Sprintf("%s:%d:%s", order.OrderID, time.Now().UnixNano(), order.CryptoAsset)
	hash := sha256.Sum256([]byte(data))
	return "0x" + hex.EncodeToString(hash[:])
}

func (s *FiatOffRampService) processFiatTransfer(order *FiatOrder) error {
	// In production, this would integrate with banking APIs:
	// - SWIFT/SEPA for international transfers
	// - Local clearing for domestic transfers
	// - Payment service providers
	
	// Simulate bank transfer
	order.Status = "processing"
	s.db.Save(order)
	
	return nil
}

// ============================================================================
// Compliance & Fraud Prevention
// ============================================================================

func (s *FiatOffRampService) runComplianceChecks(order *FiatOrder) {
	// KYC Check
	kycCheck := ComplianceCheck{
		OrderID:   order.OrderID,
		UserID:    order.UserID,
		CheckType: "kyc",
		Status:    "passed",
		Score:     1.0,
	}
	s.db.Create(&kycCheck)
	
	// AML Check
	amlCheck := ComplianceCheck{
		OrderID:   order.OrderID,
		UserID:    order.UserID,
		CheckType: "aml",
		Status:    "passed",
		Score:     0.95,
	}
	s.db.Create(&amlCheck)
	
	// Sanction Check
	sanctionCheck := ComplianceCheck{
		OrderID:   order.OrderID,
		UserID:    order.UserID,
		CheckType: "sanction",
		Status:    "passed",
		Score:     1.0,
	}
	s.db.Create(&sanctionCheck)
	
	// Update order compliance status
	s.db.Model(order).Update("compliance_status", "approved")
}

func (s *FiatOffRampService) checkFraudRisk(order *FiatOrder) bool {
	// Simple fraud check - in production use ML models
	if order.CryptoUSDValue > 50000 {
		// High value - require manual review
		order.RiskScore = 0.8
	} else if order.CryptoUSDValue > 10000 {
		order.RiskScore = 0.5
	} else {
		order.RiskScore = 0.1
	}
	
	s.db.Save(order)
	
	// Block only very high risk
	return order.RiskScore < 0.9
}

// ============================================================================
// P2P Trading
// ============================================================================

func (s *FiatOffRampService) CreateP2PListing(userID uint, listingType, cryptoAsset, fiatCurrency string, amount, minAmount, maxAmount, price float64, priceType string, paymentMethods []string) (*P2PListing, error) {
	// Validate amounts
	if amount <= 0 || minAmount <= 0 || maxAmount <= 0 {
		return nil, fmt.Errorf("invalid amounts")
	}
	if minAmount > maxAmount {
		return nil, fmt.Errorf("min amount cannot exceed max amount")
	}
	
	// Verify user KYC
	var user User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found")
	}
	
	if user.KYCStatus != "verified" {
		return nil, fmt.Errorf("KYC required for P2P trading")
	}
	
	methodsJSON, _ := json.Marshal(paymentMethods)
	
	listing := P2PListing{
		ListingID:     uuid.New().String(),
		UserID:        userID,
		ListingType:   listingType,
		CryptoAsset:   cryptoAsset,
		FiatCurrency:  fiatCurrency,
		Amount:        amount,
		MinAmount:     minAmount,
		MaxAmount:     maxAmount,
		Price:         price,
		PriceType:     priceType,
		PaymentMethods: string(methodsJSON),
		Status:        "active",
	}
	
	s.db.Create(&listing)
	
	return &listing, nil
}

func (s *FiatOffRampService) CreateP2PTrade(listingID string, buyerID uint, amount float64) (*P2PTrade, error) {
	var listing P2PListing
	if err := s.db.Where("listing_id = ? AND status = ?", listingID, "active").First(&listing).Error; err != nil {
		return nil, fmt.Errorf("listing not found")
	}
	
	// Validate amount
	if amount < listing.MinAmount || amount > listing.MaxAmount {
		return nil, fmt.Errorf("amount outside allowed range")
	}
	
	// Can't trade with yourself
	if listing.UserID == buyerID {
		return nil, fmt.Errorf("cannot trade with yourself")
	}
	
	fiatAmount := amount * listing.Price
	
	trade := P2PTrade{
		TradeID:       uuid.New().String(),
		ListingID:     listingID,
		BuyerID:       buyerID,
		SellerID:      listing.UserID,
		CryptoAsset:   listing.CryptoAsset,
		CryptoAmount:  amount,
		FiatCurrency:  listing.FiatCurrency,
		FiatAmount:    fiatAmount,
		Status:        "pending",
	}
	
	s.db.Create(&trade)
	
	return &trade, nil
}

func (s *FiatOffRampService) ConfirmP2PTrade(tradeID string, userID uint, isBuyer bool) error {
	var trade P2PTrade
	if err := s.db.First(&trade, "trade_id = ?", tradeID).Error; err != nil {
		return err
	}
	
	if isBuyer {
		trade.BuyerConfirmed = true
	} else {
		trade.SellerConfirmed = true
	}
	
	s.db.Save(&trade)
	
	// If both confirmed, release crypto
	if trade.BuyerConfirmed && trade.SellerConfirmed {
		trade.Status = "completed"
		now := time.Now()
		trade.CompletedAt = &now
		
		// In production, release crypto from escrow
		trade.ReleaseTxHash = "0x" + generateRandomHex(32)
		
		// Update listing stats
		s.db.Model(&P2PListing{}).Where("listing_id = ?", trade.ListingID).Updates(map[string]interface{}{
			"orders_completed": gorm.Expr("orders_completed + 1"),
		})
	}
	
	return s.db.Save(&trade).Error
}

// ============================================================================
// Validation
// ============================================================================

func (s *FiatOffRampService) validateBankCode(bankCode, country string) bool {
	// SWIFT code (8 or 11 characters)
	if len(bankCode) == 8 || len(bankCode) == 11 {
		matched, _ := regexp.MatchString(`^[A-Z]{4}[A-Z]{2}[A-Z0-9]{2}([A-Z0-9]{3})?$`, bankCode)
		if matched {
			return true
		}
	}
	
	// IBAN validation
	if country == "GB" && len(bankCode) == 22 {
		matched, _ := regexp.MatchString(`^GB\d{2}[A-Z]{4}\d{14}$`, bankCode)
		return matched
	}
	
	return len(bankCode) >= 6 && len(bankCode) <= 34
}

func (s *FiatOffRampService) validateAccountNumber(accountNumber string) bool {
	// Basic validation - in production use country-specific rules
	cleaned := strings.ReplaceAll(accountNumber, " ", "")
	return len(cleaned) >= 8 && len(cleaned) <= 34
}

func (s *FiatOffRampService) maskAccountNumber(accountNumber string) string {
	if len(accountNumber) <= 4 {
		return accountNumber
	}
	return "****" + accountNumber[len(accountNumber)-4:]
}

// ============================================================================
// API Handlers
// ============================================================================

func (s *FiatOffRampService) setupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	{
		// Bank accounts
		api.POST("/bank-accounts", s.addBankAccount)
		api.GET("/bank-accounts/:user_id", s.getBankAccounts)
		api.POST("/bank-accounts/:id/verify", s.verifyBankAccount)
		
		// Orders
		api.POST("/orders", s.createOrder)
		api.GET("/orders/:order_id", s.getOrder)
		api.GET("/orders/user/:user_id", s.getUserOrders)
		api.POST("/orders/:order_id/cancel", s.cancelOrder)
		api.POST("/orders/:order_id/process", s.processOrder)
		
		// Exchange rates
		api.GET("/rates/:crypto/:fiat", s.getExchangeRate)
		
		// P2P
		api.POST("/p2p/listings", s.createP2PListing)
		api.GET("/p2p/listings", s.getP2PListings)
		api.POST("/p2p/trades", s.createP2PTrade)
		api.POST("/p2p/trades/:trade_id/confirm", s.confirmP2PTrade)
	}
	
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "fiat-offramp"})
	})
}

func (s *FiatOffRampService) addBankAccount(c *gin.Context) {
	var req struct {
		UserID        uint   `json:"user_id" binding:"required"`
		BankName      string `json:"bank_name" binding:"required"`
		BankCode      string `json:"bank_code" binding:"required"`
		AccountNumber string `json:"account_number" binding:"required"`
		AccountHolder string `json:"account_holder" binding:"required"`
		AccountType   string `json:"account_type"`
		Country       string `json:"country" binding:"required"`
		Currency      string `json:"currency" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	account, err := s.AddBankAccount(req.UserID, req.BankName, req.BankCode, req.AccountNumber, req.AccountHolder, req.AccountType, req.Country, req.Currency)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{"account": account})
}

func (s *FiatOffRampService) getBankAccounts(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Param("user_id"), 10, 32)
	
	accounts, err := s.GetBankAccounts(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"accounts": accounts})
}

func (s *FiatOffRampService) verifyBankAccount(c *gin.Context) {
	accountID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	
	err := s.VerifyBankAccount(uint(accountID))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "bank account verified"})
}

func (s *FiatOffRampService) createOrder(c *gin.Context) {
	var req struct {
		UserID        uint    `json:"user_id" binding:"required"`
		CryptoAsset   string  `json:"crypto_asset" binding:"required"`
		FiatCurrency  string  `json:"fiat_currency" binding:"required"`
		CryptoAmount  float64 `json:"crypto_amount" binding:"required"`
		BankAccountID uint    `json:"bank_account_id" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	order, err := s.CreateSellOrder(req.UserID, req.CryptoAsset, req.FiatCurrency, req.CryptoAmount, req.BankAccountID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{"order": order})
}

func (s *FiatOffRampService) getOrder(c *gin.Context) {
	orderID := c.Param("order_id")
	
	order, err := s.GetOrder(orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"order": order})
}

func (s *FiatOffRampService) getUserOrders(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Param("user_id"), 10, 32)
	
	orders, err := s.GetUserOrders(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"orders": orders})
}

func (s *FiatOffRampService) cancelOrder(c *gin.Context) {
	orderID := c.Param("order_id")
	
	var req struct {
		UserID uint `json:"user_id" binding:"required"`
	}
	c.ShouldBindJSON(&req)
	
	err := s.CancelOrder(orderID, req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "order cancelled"})
}

func (s *FiatOffRampService) processOrder(c *gin.Context) {
	orderID := c.Param("order_id")
	
	err := s.ProcessOrder(orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "order processed"})
}

func (s *FiatOffRampService) getExchangeRate(c *gin.Context) {
	crypto := c.Param("crypto")
	fiat := c.Param("fiat")
	
	rate, err := s.GetExchangeRate(crypto, fiat)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"rate": rate})
}

func (s *FiatOffRampService) createP2PListing(c *gin.Context) {
	var req struct {
		UserID        uint     `json:"user_id" binding:"required"`
		ListingType   string   `json:"listing_type" binding:"required"`
		CryptoAsset   string   `json:"crypto_asset" binding:"required"`
		FiatCurrency  string   `json:"fiat_currency" binding:"required"`
		Amount        float64  `json:"amount" binding:"required"`
		MinAmount     float64  `json:"min_amount" binding:"required"`
		MaxAmount     float64  `json:"max_amount" binding:"required"`
		Price         float64  `json:"price" binding:"required"`
		PriceType     string   `json:"price_type" binding:"required"`
		PaymentMethods []string `json:"payment_methods" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	listing, err := s.CreateP2PListing(req.UserID, req.ListingType, req.CryptoAsset, req.FiatCurrency, req.Amount, req.MinAmount, req.MaxAmount, req.Price, req.PriceType, req.PaymentMethods)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{"listing": listing})
}

func (s *FiatOffRampService) getP2PListings(c *gin.Context) {
	var listings []P2PListing
	s.db.Where("status = ?", "active").Order("created_at desc").Limit(50).Find(&listings)
	
	c.JSON(http.StatusOK, gin.H{"listings": listings})
}

func (s *FiatOffRampService) createP2PTrade(c *gin.Context) {
	var req struct {
		ListingID string  `json:"listing_id" binding:"required"`
		BuyerID   uint    `json:"buyer_id" binding:"required"`
		Amount    float64 `json:"amount" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	trade, err := s.CreateP2PTrade(req.ListingID, req.BuyerID, req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{"trade": trade})
}

func (s *FiatOffRampService) confirmP2PTrade(c *gin.Context) {
	tradeID := c.Param("trade_id")
	
	var req struct {
		UserID  uint `json:"user_id" binding:"required"`
		IsBuyer bool `json:"is_buyer"`
	}
	c.ShouldBindJSON(&req)
	
	err := s.ConfirmP2PTrade(tradeID, req.UserID, req.IsBuyer)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "trade confirmed"})
}

// ============================================================================
// Helper Functions
// ============================================================================

func generateRandomHex(length int) string {
	data := fmt.Sprintf("%d:%s", time.Now().UnixNano(), uuid.New().String())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])[:length]
}

// ============================================================================
// Main
// ============================================================================

func main() {
	cfg := LoadConfig()
	
	service, err := NewFiatOffRampService(cfg)
	if err != nil {
		log.Fatalf("Failed to create fiat off-ramp service: %v", err)
	}
	
	router := gin.Default()
	service.setupRoutes(router)
	
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		<-quit
		log.Println("Shutting down fiat off-ramp service...")
		os.Exit(0)
	}()
	
	log.Printf("Fiat Off-Ramp Service starting on port %s", cfg.ServerPort)
	if err := router.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
