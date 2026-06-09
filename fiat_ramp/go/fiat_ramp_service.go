// Fiat Ramp Service - Go Implementation
// On/off ramp for fiat currency conversion

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Configuration
type FiatRampConfig struct {
	ServerPort  string `json:"server_port"`
	DBHost     string `json:"db_host"`
	DBPort     string `json:"db_port"`
	DBUser     string `json:"db_user"`
	DBPassword string `json:"db_password"`
	DBName     string `json:"db_name"`
	RedisHost  string `json:"redis_host"`
	RedisPort  string `json:"redis_port"`
}

// Fiat Ramp Status
const (
	RAMP_STATUS_PENDING   = "pending"
	RAMP_STATUS_PROCESSING = "processing"
	RAMP_STATUS_COMPLETED = "completed"
	RAMP_STATUS_FAILED   = "failed"
	RAMP_STATUS_CANCELLED = "cancelled"
)

// Fiat Ramp Type
const (
	RAMP_TYPE_BUY  = "buy"  // Fiat -> Crypto
	RAMP_TYPE_SELL = "sell" // Crypto -> Fiat
)

// Supported Fiat Currencies
var FIAT_CURRENCIES = []string{"USD", "EUR", "GBP", "JPY", "KRW", "CNY", "INR"}

// Supported Crypto Currencies
var CRYPTO_CURRENCIES = []string{"ETH", "BTC", "USDT", "USDC", "MATIC", "BNB", "AVAX"}

// FiatRamp represents a fiat ramp order
type FiatRamp struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	OrderID      string    `gorm:"uniqueIndex" json:"order_id"`
	UserAddress  string    `gorm:"index" json:"user_address"`
	Type        string    `json:"type"`
	FiatAmount  string    `json:"fiat_amount"`
	FiatCurrency string   `json:"fiat_currency"`
	CryptoAmount string   `json:"crypto_amount"`
	CryptoToken  string   `json:"crypto_token"`
	ExchangeRate string   `json:"exchange_rate"`
	Provider    string    `json:"provider"`
	BankAccount string   `json:"bank_account"`
	Status      string    `json:"status"`
	TxHash      string    `json:"tx_hash"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at"`
}

// BankAccount represents a user's bank account
type BankAccount struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserAddress string    `gorm:"index" json:"user_address"`
	BankName   string    `json:"bank_name"`
	AccountNum string    `json:"account_number"` // Encrypted
	RoutingNum string    `json:"routing_number"`
	SwiftCode  string    `json:"swift_code"`
	IBAN       string    `json:"iban"`
	IsDefault  bool      `json:"is_default"`
	CreatedAt  time.Time `json:"created_at"`
}

// FiatRampService
type FiatRampService struct {
	db      *gorm.DB
	redis   *redis.Client
	config  FiatRampConfig
}

// NewFiatRampService creates new service
func NewFiatRampService(cfg FiatRampConfig) (*FiatRampService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&FiatRamp{}, &BankAccount{})
	if err != nil {
		return nil, err
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &FiatRampService{
		db:     db,
		redis:  rdb,
		config: cfg,
	}, nil
}

// GenerateOrderID generates a unique order ID
func (s *FiatRampService) GenerateOrderID() string {
	var b [16]byte
	rand.Read(b[:])
	return "ramp_" + hex.EncodeToString(b[:])
}

// GetExchangeRate gets current exchange rate
func (s *FiatRampService) GetExchangeRate(fiatCurrency, cryptoToken string) (string, error) {
	// Simulated rates - in production, fetch from price oracle
	rates := map[string]map[string]string{
		"USD": {"ETH": "3500", "BTC": "65000", "USDT": "1", "USDC": "1", "MATIC": "0.85", "BNB": "600", "AVAX": "35"},
		"EUR": {"ETH": "3200", "BTC": "59000", "USDT": "0.91", "USDC": "0.91", "MATIC": "0.78", "BNB": "550", "AVAX": "32"},
		"GBP": {"ETH": "2800", "BTC": "52000", "USDT": "0.79", "USDC": "0.79", "MATIC": "0.68", "BNB": "480", "AVAX": "28"},
	}

	if rates[fiatCurrency] != nil {
		if rate, ok := rates[fiatCurrency][cryptoToken]; ok {
			return rate, nil
		}
	}

	return "1", nil // Default to 1 for stablecoins
}

// CreateBuyOrder creates a buy order (fiat -> crypto)
func (s *FiatRampService) CreateBuyOrder(req CreateOrderRequest) (*FiatRamp, error) {
	orderID := s.GenerateOrderID()

	rate, err := s.GetExchangeRate(req.FiatCurrency, req.CryptoToken)
	if err != nil {
		return nil, err
	}

	// Calculate crypto amount
	fiatAmount := parseAmount(req.FiatAmount)
	rateFloat := parseAmount(rate)
	cryptoAmount := fiatAmount / rateFloat

	order := &FiatRamp{
		OrderID:       orderID,
		UserAddress:   req.UserAddress,
		Type:         RAMP_TYPE_BUY,
		FiatAmount:   req.FiatAmount,
		FiatCurrency: req.FiatCurrency,
		CryptoAmount: formatAmount(cryptoAmount),
		CryptoToken:  req.CryptoToken,
		ExchangeRate: rate,
		Provider:    req.Provider,
		BankAccount:  req.BankAccount,
		Status:      RAMP_STATUS_PENDING,
		CreatedAt:   time.Now(),
	}

	s.db.Create(order)
	return order, nil
}

// CreateSellOrder creates a sell order (crypto -> fiat)
func (s *FiatRampService) CreateSellOrder(req CreateOrderRequest) (*FiatRamp, error) {
	orderID := s.GenerateOrderID()

	rate, err := s.GetExchangeRate(req.FiatCurrency, req.CryptoToken)
	if err != nil {
		return nil, err
	}

	fiatAmount := parseAmount(req.CryptoAmount) * parseAmount(rate)

	order := &FiatRamp{
		OrderID:       orderID,
		UserAddress:   req.UserAddress,
		Type:         RAMP_TYPE_SELL,
		FiatAmount:   formatAmount(fiatAmount),
		FiatCurrency: req.FiatCurrency,
		CryptoAmount: req.CryptoAmount,
		CryptoToken:  req.CryptoToken,
		ExchangeRate: rate,
		Provider:    req.Provider,
		BankAccount: req.BankAccount,
		Status:      RAMP_STATUS_PENDING,
		CreatedAt:   time.Now(),
	}

	s.db.Create(order)
	return order, nil
}

// GetOrder gets an order
func (s *FiatRampService) GetOrder(orderID string) (*FiatRamp, error) {
	var order FiatRamp
	if err := s.db.Where("order_id = ?", orderID).First(&order).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

// GetUserOrders gets orders for a user
func (s *FiatRampService) GetUserOrders(userAddress string) ([]FiatRamp, error) {
	var orders []FiatRamp
	err := s.db.Where("user_address = ?", userAddress).Order("created_at DESC").Find(&orders).Error
	return orders, err
}

// UpdateOrderStatus updates order status
func (s *FiatRampService) UpdateOrderStatus(orderID, status, txHash string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if txHash != "" {
		updates["tx_hash"] = txHash
	}

	if status == RAMP_STATUS_COMPLETED {
		now := time.Now()
		updates["completed_at"] = now
	}

	result := s.db.Model(&FiatRamp{}).Where("order_id = ?", orderID).Updates(updates)

	if result.RowsAffected == 0 {
		return fmt.Errorf("order not found")
	}

	return nil
}

// CancelOrder cancels an order
func (s *FiatRampService) CancelOrder(orderID string) error {
	return s.UpdateOrderStatus(orderID, RAMP_STATUS_CANCELLED, "")
}

// AddBankAccount adds a bank account
func (s *FiatRampService) AddBankAccount(userAddress string, account BankAccount) error {
	// If this is default, unset other defaults
	if account.IsDefault {
		s.db.Model(&BankAccount{}).Where("user_address = ?", userAddress).Update("is_default", false)
	}

	result := s.db.Model(&BankAccount{}).Where("user_address = ? AND bank_name = ?", userAddress, account.BankName).Updates(map[string]interface{}{
		"account_number": account.AccountNum,
		"routing_number": account.RoutingNum,
		"swift_code":     account.SwiftCode,
		"iban":          account.IBAN,
		"is_default":    account.IsDefault,
	})

	if result.RowsAffected == 0 {
		account.UserAddress = userAddress
		s.db.Create(&account)
	}

	return nil
}

// GetBankAccounts gets bank accounts for a user
func (s *FiatRampService) GetBankAccounts(userAddress string) ([]BankAccount, error) {
	var accounts []BankAccount
	err := s.db.Where("user_address = ?", userAddress).Find(&accounts).Error
	return accounts, err
}

// GetSupportedCurrencies gets supported currencies
func (s *FiatRampService) GetSupportedCurrencies() (map[string][]string, error) {
	return map[string][]string{
		"fiat":  FIAT_CURRENCIES,
		"crypto": CRYPTO_CURRENCIES,
	}, nil
}

// Handlers

type CreateOrderRequest struct {
	UserAddress   string  `json:"user_address" binding:"required"`
	FiatAmount   string  `json:"fiat_amount"`
	CryptoAmount string  `json:"crypto_amount"`
	FiatCurrency string  `json:"fiat_currency" binding:"required"`
	CryptoToken  string  `json:"crypto_token" binding:"required"`
	Provider    string  `json:"provider"`
	BankAccount string  `json:"bank_account"`
}

func (s *FiatRampService) CreateBuyHandler(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	order, err := s.CreateBuyOrder(req)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, order)
}

func (s *FiatRampService) CreateSellHandler(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	order, err := s.CreateSellOrder(req)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, order)
}

func (s *FiatRampService) GetOrderHandler(c *gin.Context) {
	orderID := c.Param("order_id")

	order, err := s.GetOrder(orderID)
	if err != nil {
		c.JSON(404, gin.H{"error": "order not found"})
		return
	}
	c.JSON(200, order)
}

func (s *FiatRampService) GetUserOrdersHandler(c *gin.Context) {
	address := c.Param("address")

	orders, err := s.GetUserOrders(address)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, orders)
}

type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required"`
	TxHash string `json:"tx_hash"`
}

func (s *FiatRampService) UpdateStatusHandler(c *gin.Context) {
	orderID := c.Param("order_id")

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if err := s.UpdateOrderStatus(orderID, req.Status, req.TxHash); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": req.Status})
}

func (s *FiatRampService) CancelHandler(c *gin.Context) {
	orderID := c.Param("order_id")

	if err := s.CancelOrder(orderID); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "cancelled"})
}

func (s *FiatRampService) GetRatesHandler(c *gin.Context) {
	fiatCurrency := c.DefaultQuery("fiat", "USD")
	cryptoToken := c.Query("crypto")

	rate, err := s.GetExchangeRate(fiatCurrency, cryptoToken)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"rate": rate})
}

func (s *FiatRampService) GetCurrenciesHandler(c *gin.Context) {
	currencies, err := s.GetSupportedCurrencies()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, currencies)
}

func (s *FiatRampService) AddBankHandler(c *gin.Context) {
	address := c.Param("address")

	var account BankAccount
	if err := c.ShouldBindJSON(&account); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if err := s.AddBankAccount(address, account); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "added"})
}

func (s *FiatRampService) GetBanksHandler(c *gin.Context) {
	address := c.Param("address")

	accounts, err := s.GetBankAccounts(address)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, accounts)
}

// Utility functions

func parseAmount(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func formatAmount(f float64) string {
	return fmt.Sprintf("%.6f", f)
}

// Main

func main() {
	cfg := FiatRampConfig{
		ServerPort: getEnv("FIAT_RAMP_PORT", "8087"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "fiat_ramp_db"),
		RedisHost:  getEnv("REDIS_HOST", "localhost"),
		RedisPort:  getEnv("REDIS_PORT", "6379"),
	}

	service, err := NewFiatRampService(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize service: %v\n", err)
		os.Exit(1)
	}

	r := gin.Default()

	r.POST("/buy", service.CreateBuyHandler)
	r.POST("/sell", service.CreateSellHandler)
	r.GET("/order/:order_id", service.GetOrderHandler)
	r.GET("/orders/:address", service.GetUserOrdersHandler)
	r.POST("/order/:order_id/status", service.UpdateStatusHandler)
	r.POST("/order/:order_id/cancel", service.CancelHandler)
	r.GET("/rates", service.GetRatesHandler)
	r.GET("/currencies", service.GetCurrenciesHandler)
	r.POST("/banks/:address", service.AddBankHandler)
	r.GET("/banks/:address", service.GetBanksHandler)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	go func() {
		fmt.Printf("Fiat Ramp Service starting on port %s\n", cfg.ServerPort)
		if err := r.Run(":" + cfg.ServerPort); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start server: %n", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down...")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}