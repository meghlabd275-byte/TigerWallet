/**
 * TigerWallet Payment Gateway
 * Multi-Payment Method Integration Service
 */

package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
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
	ServerPort    string
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	RedisHost     string
	RedisPort     string
	StripeKey     string
	PayPalClient  string
	PayPalSecret  string
}

func LoadConfig() *Config {
	return &Config{
		ServerPort:   getEnv("PAYMENT_PORT", "9105"),
		DBHost:       getEnv("DB_HOST", "localhost"),
		DBPort:       getEnv("DB_PORT", "5432"),
		DBUser:       getEnv("DB_USER", "tigerwallet"),
		DBPassword:   getEnv("DB_PASSWORD", "password"),
		DBName:       getEnv("DB_NAME", "tigerwallet"),
		RedisHost:    getEnv("REDIS_HOST", "localhost"),
		RedisPort:    getEnv("REDIS_PORT", "6379"),
		StripeKey:    getEnv("STRIPE_KEY", ""),
		PayPalClient: getEnv("PAYPAL_CLIENT", ""),
		PayPalSecret: getEnv("PAYPAL_SECRET", ""),
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

type PaymentMethod struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	MethodID        string    `gorm:"uniqueIndex;size:36" json:"method_id"`
	UserID          uint      `gorm:"index" json:"user_id"`
	MethodType      string    `json:"method_type"` // card, bank, wallet, crypto
	Provider        string    `json:"provider"` // stripe, paypal, coinbase
	CardLast4       string    `json:"card_last4"`
	CardBrand       string    `json:"card_brand"`
	CardExpMonth    int       `json:"card_exp_month"`
	CardExpYear     int       `json:"card_exp_year"`
	BankName        string    `json:"bank_name"`
	BankCode        string    `json:"bank_code"`
	IsDefault       bool      `json:"is_default"`
	IsVerified      bool      `json:"is_verified"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Transaction struct {
	ID                uint           `gorm:"primarykey" json:"id"`
	TransactionID    string         `gorm:"uniqueIndex;size:36" json:"transaction_id"`
	UserID            uint           `gorm:"index" json:"user_id"`
	OrderID          string         `gorm:"index" json:"order_id"`
	Amount           float64        `json:"amount"`
	Currency         string         `json:"currency"`
	PaymentMethod    string         `json:"payment_method"`
	Provider         string         `json:"provider"`
	Status           string         `json:"status"` // pending, processing, completed, failed, refunded
	Type             string         `json:"type"` // deposit, withdrawal, payment
	ProviderRef      string         `json:"provider_ref"`
	ProviderResponse string         `json:"provider_response"`
	Metadata         string         `json:"metadata"`
	FeeAmount        float64        `json:"fee_amount"`
	NetAmount        float64        `json:"net_amount"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	CompletedAt      *time.Time     `json:"completed_at"`
}

type Refund struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	RefundID        string    `gorm:"uniqueIndex;size:36" json:"refund_id"`
	TransactionID   string    `gorm:"index" json:"transaction_id"`
	Amount          float64   `json:"amount"`
	Reason          string    `json:"reason"`
	Status          string    `json:"status"` // pending, completed, failed
	ProviderRef     string    `json:"provider_ref"`
	CreatedAt       time.Time `json:"created_at"`
	CompletedAt     *time.Time `json:"completed_at"`
}

type Invoice struct {
	ID              uint           `gorm:"primarykey" json:"id"`
	InvoiceID       string         `gorm:"uniqueIndex;size:36" json:"invoice_id"`
	UserID          uint           `gorm:"index" json:"user_id"`
	Amount          float64        `json:"amount"`
	Currency        string         `json:"currency"`
	Description     string         `json:"description"`
	Status          string         `json:"status"` // draft, pending, paid, cancelled, expired
	DueDate         time.Time      `json:"due_date"`
	PaidAt          *time.Time     `json:"paid_at"`
	Transactions    string         `json:"transactions"` // JSON array
	CreatedAt       time.Time      `json:"created_at"`
}

// ============================================================================
// Payment Gateway Service
// ============================================================================

type PaymentGateway struct {
	config *Config
	db     *gorm.DB
	redis  *redis.Client
}

func NewPaymentGateway(cfg *Config) (*PaymentGateway, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	db.AutoMigrate(&PaymentMethod{}, &Transaction{}, &Refund{}, &Invoice{})

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: "",
		DB:       0,
	})

	return &PaymentGateway{
		config: cfg,
		db:     db,
		redis:  rdb,
	}, nil
}

// ============================================================================
// Payment Method Management
// ============================================================================

func (p *PaymentGateway) AddCard(userID uint, cardNumber, expMonth, expYear, cvv string) (*PaymentMethod, error) {
	// Validate card
	if !p.validateCardNumber(cardNumber) {
		return nil, fmt.Errorf("invalid card number")
	}

	// Get last 4 digits
	last4 := cardNumber[len(cardNumber)-4:]

	// Detect card brand
	brand := detectCardBrand(cardNumber)

	// In production, this would tokenize with payment provider
	methodID := uuid.New().String()

	method := PaymentMethod{
		MethodID:      methodID,
		UserID:        userID,
		MethodType:    "card",
		Provider:      "stripe",
		CardLast4:     last4,
		CardBrand:     brand,
		CardExpMonth:  p.parseInt(expMonth),
		CardExpYear:   p.parseInt(expYear),
		IsDefault:     false,
		IsVerified:    true,
		CreatedAt:     time.Now(),
		UpdatedAt:      time.Now(),
	}

	err := p.db.Create(&method).Error
	return &method, err
}

func (p *PaymentGateway) AddBankAccount(userID uint, bankName, accountNumber, routingNumber, country string) (*PaymentMethod, error) {
	// Validate bank account
	if !p.validateBankAccount(accountNumber) {
		return nil, fmt.Errorf("invalid account number")
	}

	methodID := uuid.New().String()

	method := PaymentMethod{
		MethodID:    methodID,
		UserID:      userID,
		MethodType:  "bank",
		Provider:    "stripe",
		BankName:    bankName,
		BankCode:    routingNumber,
		IsDefault:   false,
		IsVerified:  false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := p.db.Create(&method).Error
	return &method, err
}

func (p *PaymentGateway) GetUserPaymentMethods(userID uint) ([]PaymentMethod, error) {
	var methods []PaymentMethod
	err := p.db.Where("user_id = ?", userID).Find(&methods).Error
	return methods, err
}

func (p *PaymentGateway) SetDefaultPaymentMethod(userID uint, methodID string) error {
	// Unset all defaults
	p.db.Model(&PaymentMethod{}).Where("user_id = ?", userID).Update("is_default", false)

	// Set new default
	return p.db.Model(&PaymentMethod{}).Where("method_id = ?", methodID).Update("is_default", true).Error
}

func (p *PaymentGateway) DeletePaymentMethod(methodID string) error {
	return p.db.Where("method_id = ?", methodID).Delete(&PaymentMethod{}).Error
}

// ============================================================================
// Payment Processing
// ============================================================================

func (p *PaymentGateway) CreatePayment(userID uint, amount float64, currency, paymentMethod, methodType, description string) (*Transaction, error) {
	// Validate amount
	if amount <= 0 {
		return nil, fmt.Errorf("invalid amount")
	}

	// Validate currency
	if !p.validateCurrency(currency) {
		return nil, fmt.Errorf("unsupported currency")
	}

	// Calculate fee
	fee := p.calculateFee(amount, methodType)
	netAmount := amount - fee

	txn := Transaction{
		TransactionID: uuid.New().String(),
		UserID:        userID,
		Amount:        amount,
		Currency:      currency,
		PaymentMethod: paymentMethod,
		Provider:      "stripe",
		Status:        "pending",
		Type:          "payment",
		FeeAmount:     fee,
		NetAmount:     netAmount,
		CreatedAt:     time.Now(),
		UpdatedAt:      time.Now(),
	}

	err := p.db.Create(&txn).Error
	return &txn, err
}

func (p *PaymentGateway) ProcessPayment(transactionID string) (*Transaction, error) {
	var txn Transaction
	if err := p.db.Where("transaction_id = ?", transactionID).First(&txn).Error; err != nil {
		return nil, err
	}

	if txn.Status != "pending" {
		return nil, fmt.Errorf("transaction not in pending status")
	}

	// In production, this would call the payment provider
	// For now, simulate successful payment
	providerRef := generateProviderRef()

	txn.Status = "completed"
	txn.ProviderRef = providerRef
	txn.ProviderResponse = `{"status": "succeeded"}`
	txn.CompletedAt = new(time.Time)
	*txn.CompletedAt = time.Now()

	err := p.db.Save(&txn).Error
	return &txn, err
}

func (p *PaymentGateway) GetTransaction(transactionID string) (*Transaction, error) {
	var txn Transaction
	err := p.db.Where("transaction_id = ?", transactionID).First(&txn).Error
	return &txn, err
}

func (p *PaymentGateway) GetUserTransactions(userID uint, limit int) ([]Transaction, error) {
	var transactions []Transaction
	err := p.db.Where("user_id = ?", userID).Order("created_at desc").Limit(limit).Find(&transactions).Error
	return transactions, err
}

func (p *PaymentGateway) CancelTransaction(transactionID string) error {
	var txn Transaction
	if err := p.db.Where("transaction_id = ?", transactionID).First(&txn).Error; err != nil {
		return err
	}

	if txn.Status != "pending" {
		return fmt.Errorf("cannot cancel transaction in status: %s", txn.Status)
	}

	txn.Status = "cancelled"
	return p.db.Save(&txn).Error
}

// ============================================================================
// Refunds
// ============================================================================

func (p *PaymentGateway) CreateRefund(transactionID string, amount float64, reason string) (*Refund, error) {
	var txn Transaction
	if err := p.db.Where("transaction_id = ?", transactionID).First(&txn).Error; err != nil {
		return nil, err
	}

	if txn.Status != "completed" {
		return nil, fmt.Errorf("can only refund completed transactions")
	}

	if amount > txn.Amount {
		return nil, fmt.Errorf("refund amount exceeds transaction amount")
	}

	refund := Refund{
		RefundID:      uuid.New().String(),
		TransactionID: transactionID,
		Amount:        amount,
		Reason:        reason,
		Status:        "pending",
		CreatedAt:     time.Now(),
	}

	err := p.db.Create(&refund).Error
	return &refund, err
}

func (p *PaymentGateway) ProcessRefund(refundID string) (*Refund, error) {
	var refund Refund
	if err := p.db.Where("refund_id = ?", refundID).First(&refund).Error; err != nil {
		return nil, err
	}

	if refund.Status != "pending" {
		return nil, fmt.Errorf("refund not in pending status")
	}

	// Process refund with provider
	providerRef := generateProviderRef()

	refund.Status = "completed"
	refund.ProviderRef = providerRef
	refund.CompletedAt = new(time.Time)
	*refund.CompletedAt = time.Now()

	err := p.db.Save(&refund).Error
	return &refund, err
}

// ============================================================================
// Invoices
// ============================================================================

func (p *PaymentGateway) CreateInvoice(userID uint, amount float64, currency, description string, dueDate time.Time) (*Invoice, error) {
	invoice := Invoice{
		InvoiceID:   uuid.New().String(),
		UserID:      userID,
		Amount:      amount,
		Currency:    currency,
		Description: description,
		Status:      "draft",
		DueDate:     dueDate,
		CreatedAt:   time.Now(),
	}

	err := p.db.Create(&invoice).Error
	return &invoice, err
}

func (p *PaymentGateway) PayInvoice(invoiceID string, paymentMethodID string) (*Invoice, error) {
	var invoice Invoice
	if err := p.db.Where("invoice_id = ?", invoiceID).First(&invoice).Error; err != nil {
		return nil, err
	}

	if invoice.Status != "draft" && invoice.Status != "pending" {
		return nil, fmt.Errorf("invoice cannot be paid")
	}

	// Create payment transaction
	txn, err := p.CreatePayment(invoice.UserID, invoice.Amount, invoice.Currency, paymentMethodID, "card", invoice.Description)
	if err != nil {
		return nil, err
	}

	// Process payment
	_, err = p.ProcessPayment(txn.TransactionID)
	if err != nil {
		return nil, err
	}

	// Update invoice
	now := time.Now()
	invoice.Status = "paid"
	invoice.PaidAt = &now
	invoice.Transactions = fmt.Sprintf(`["%s"]`, txn.TransactionID)

	err = p.db.Save(&invoice).Error
	return &invoice, err
}

func (p *PaymentGateway) GetInvoice(invoiceID string) (*Invoice, error) {
	var invoice Invoice
	err := p.db.Where("invoice_id = ?", invoiceID).First(&invoice).Error
	return &invoice, err
}

// ============================================================================
// Crypto Payments
// ============================================================================

type CryptoPayment struct {
	Address     string  `json:"address"`
	Amount      float64 `json:"amount"`
	Network     string  `json:"network"`
	Currency    string  `json:"currency"`
	ExpiresAt   int64   `json:"expires_at"`
}

func (p *PaymentGateway) CreateCryptoPayment(userID uint, amount float64, currency, network string) (*CryptoPayment, error) {
	// Generate payment address (in production, this would be a real crypto address)
	address := generateCryptoAddress(currency, network)

	// Set expiration (30 minutes)
	expiresAt := time.Now().Add(30 * time.Minute).Unix()

	payment := CryptoPayment{
		Address:   address,
		Amount:    amount,
		Network:   network,
		Currency:  currency,
		ExpiresAt: expiresAt,
	}

	// Store in Redis with expiration
	paymentJSON, _ := json.Marshal(payment)
	p.redis.Set(context.Background(), fmt.Sprintf("crypto_payment:%s", address), paymentJSON, 30*time.Minute)

	// Create transaction record
	txn := Transaction{
		TransactionID: uuid.New().String(),
		UserID:        userID,
		Amount:        amount,
		Currency:      currency,
		PaymentMethod: "crypto",
		Provider:      "coinbase",
		Status:        "pending",
		Type:          "deposit",
		CreatedAt:     time.Now(),
		UpdatedAt:      time.Now(),
	}
	p.db.Create(&txn)

	return &payment, nil
}

func (p *PaymentGateway) VerifyCryptoPayment(address string) (bool, error) {
	ctx := context.Background()
	result, err := p.redis.Get(ctx, fmt.Sprintf("crypto_payment:%s", address)).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	var payment CryptoPayment
	if err := json.Unmarshal([]byte(result), &payment); err != nil {
		return false, err
	}

	// Check expiration
	if time.Now().Unix() > payment.ExpiresAt {
		return false, nil
	}

	return true, nil
}

// ============================================================================
// Validation Helpers
// ============================================================================

func (p *PaymentGateway) validateCardNumber(cardNumber string) bool {
	// Remove spaces and dashes
	cleaned := regexp.MustCompile(`[\s-]+`).ReplaceAllString(cardNumber, "")

	// Check length
	if len(cleaned) < 13 || len(cleaned) > 19 {
		return false
	}

	// Luhn algorithm
	return luhnCheck(cleaned)
}

func luhnCheck(cardNumber string) bool {
	sum := 0
	isSecond := false

	for i := len(cardNumber) - 1; i >= 0; i-- {
		d := int(cardNumber[i] - '0')

		if isSecond {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}

		sum += d
		isSecond = !isSecond
	}

	return sum%10 == 0
}

func detectCardBrand(cardNumber string) string {
	cleaned := regexp.MustCompile(`[\s-]+`).ReplaceAllString(cardNumber, "")

	if strings.HasPrefix(cleaned, "4") {
		return "Visa"
	} else if matched, _ := regexp.MatchString(`^5[1-5]`, cleaned); matched {
		return "Mastercard"
	} else if matched, _ := regexp.MatchString(`^3[47]`, cleaned); matched {
		return "Amex"
	} else if matched, _ := regexp.MatchString(`^6(?:011|5)`, cleaned); matched {
		return "Discover"
	}

	return "Unknown"
}

func (p *PaymentGateway) validateBankAccount(accountNumber string) bool {
	return len(accountNumber) >= 8 && len(accountNumber) <= 17
}

func (p *PaymentGateway) validateCurrency(currency string) bool {
	validCurrencies := map[string]bool{
		"USD": true, "EUR": true, "GBP": true, "JPY": true,
		"CNY": true, "KRW": true, "INR": true, "BRL": true,
		"USDT": true, "USDC": true, "BTC": true, "ETH": true,
	}
	return validCurrencies[currency]
}

func (p *PaymentGateway) calculateFee(amount float64, methodType string) float64 {
	feeRates := map[string]float64{
		"card":        0.029,
		"bank":        0.01,
		"crypto":      0.01,
		"wallet":      0.0,
	}

	rate := feeRates[methodType]
	if rate == 0 {
		rate = 0.025
	}

	return amount * rate
}

func (p *PaymentGateway) parseInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

// ============================================================================
// API Handlers
// ============================================================================

func (p *PaymentGateway) setupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	{
		// Payment methods
		api.POST("/payment-methods/card", p.addCard)
		api.POST("/payment-methods/bank", p.addBankAccount)
		api.GET("/payment-methods/:user_id", p.getPaymentMethods)
		api.POST("/payment-methods/:method_id/default", p.setDefaultMethod)
		api.DELETE("/payment-methods/:method_id", p.deleteMethod)

		// Payments
		api.POST("/payments", p.createPayment)
		api.POST("/payments/:transaction_id/process", p.processPayment)
		api.GET("/payments/:transaction_id", p.getTransaction)
		api.GET("/payments/user/:user_id", p.getUserTransactions)
		api.POST("/payments/:transaction_id/cancel", p.cancelPayment)

		// Refunds
		api.POST("/refunds", p.createRefund)
		api.POST("/refunds/:refund_id/process", p.processRefund)

		// Invoices
		api.POST("/invoices", p.createInvoice)
		api.GET("/invoices/:invoice_id", p.getInvoice)
		api.POST("/invoices/:invoice_id/pay", p.payInvoice)

		// Crypto
		api.POST("/crypto/payment", p.createCryptoPayment)
		api.POST("/crypto/verify", p.verifyCryptoPayment)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "payment-gateway"})
	})
}

func (p *PaymentGateway) addCard(c *gin.Context) {
	var req struct {
		UserID    uint   `json:"user_id" binding:"required"`
		CardNumber string `json:"card_number" binding:"required"`
		ExpMonth  string `json:"exp_month" binding:"required"`
		ExpYear   string `json:"exp_year" binding:"required"`
		CVV       string `json:"cvv" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	method, err := p.AddCard(req.UserID, req.CardNumber, req.ExpMonth, req.ExpYear, req.CVV)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"method": method})
}

func (p *PaymentGateway) addBankAccount(c *gin.Context) {
	var req struct {
		UserID       uint   `json:"user_id" binding:"required"`
		BankName     string `json:"bank_name" binding:"required"`
		AccountNumber string `json:"account_number" binding:"required"`
		RoutingNumber string `json:"routing_number" binding:"required"`
		Country      string `json:"country" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	method, err := p.AddBankAccount(req.UserID, req.BankName, req.AccountNumber, req.RoutingNumber, req.Country)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"method": method})
}

func (p *PaymentGateway) getPaymentMethods(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Param("user_id"), 10, 32)

	methods, err := p.GetUserPaymentMethods(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"methods": methods})
}

func (p *PaymentGateway) setDefaultMethod(c *gin.Context) {
	methodID := c.Param("method_id")

	var req struct {
		UserID uint `json:"user_id" binding:"required"`
	}
	c.ShouldBindJSON(&req)

	err := p.SetDefaultPaymentMethod(req.UserID, methodID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "default method updated"})
}

func (p *PaymentGateway) deleteMethod(c *gin.Context) {
	methodID := c.Param("method_id")

	err := p.DeletePaymentMethod(methodID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "method deleted"})
}

func (p *PaymentGateway) createPayment(c *gin.Context) {
	var req struct {
		UserID       uint   `json:"user_id" binding:"required"`
		Amount       float64 `json:"amount" binding:"required"`
		Currency     string  `json:"currency" binding:"required"`
		PaymentMethod string `json:"payment_method" binding:"required"`
		MethodType   string  `json:"method_type" binding:"required"`
		Description  string  `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	txn, err := p.CreatePayment(req.UserID, req.Amount, req.Currency, req.PaymentMethod, req.MethodType, req.Description)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"transaction": txn})
}

func (p *PaymentGateway) processPayment(c *gin.Context) {
	transactionID := c.Param("transaction_id")

	txn, err := p.ProcessPayment(transactionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"transaction": txn})
}

func (p *PaymentGateway) getTransaction(c *gin.Context) {
	transactionID := c.Param("transaction_id")

	txn, err := p.GetTransaction(transactionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"transaction": txn})
}

func (p *PaymentGateway) getUserTransactions(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Param("user_id"), 10, 32)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	txns, err := p.GetUserTransactions(uint(userID), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"transactions": txns})
}

func (p *PaymentGateway) cancelPayment(c *gin.Context) {
	transactionID := c.Param("transaction_id")

	err := p.CancelTransaction(transactionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "transaction cancelled"})
}

func (p *PaymentGateway) createRefund(c *gin.Context) {
	var req struct {
		TransactionID string  `json:"transaction_id" binding:"required"`
		Amount        float64 `json:"amount" binding:"required"`
		Reason        string  `json:"reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	refund, err := p.CreateRefund(req.TransactionID, req.Amount, req.Reason)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"refund": refund})
}

func (p *PaymentGateway) processRefund(c *gin.Context) {
	refundID := c.Param("refund_id")

	refund, err := p.ProcessRefund(refundID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"refund": refund})
}

func (p *PaymentGateway) createInvoice(c *gin.Context) {
	var req struct {
		UserID      uint      `json:"user_id" binding:"required"`
		Amount      float64   `json:"amount" binding:"required"`
		Currency    string    `json:"currency" binding:"required"`
		Description string    `json:"description"`
		DueDate    time.Time `json:"due_date"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dueDate := req.DueDate
	if dueDate.IsZero() {
		dueDate = time.Now().Add(30 * 24 * time.Hour)
	}

	invoice, err := p.CreateInvoice(req.UserID, req.Amount, req.Currency, req.Description, dueDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"invoice": invoice})
}

func (p *PaymentGateway) getInvoice(c *gin.Context) {
	invoiceID := c.Param("invoice_id")

	invoice, err := p.GetInvoice(invoiceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"invoice": invoice})
}

func (p *PaymentGateway) payInvoice(c *gin.Context) {
	invoiceID := c.Param("invoice_id")

	var req struct {
		PaymentMethodID string `json:"payment_method_id" binding:"required"`
	}
	c.ShouldBindJSON(&req)

	invoice, err := p.PayInvoice(invoiceID, req.PaymentMethodID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"invoice": invoice})
}

func (p *PaymentGateway) createCryptoPayment(c *gin.Context) {
	var req struct {
		UserID   uint   `json:"user_id" binding:"required"`
		Amount   float64 `json:"amount" binding:"required"`
		Currency string  `json:"currency" binding:"required"`
		Network  string  `json:"network" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payment, err := p.CreateCryptoPayment(req.UserID, req.Amount, req.Currency, req.Network)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"payment": payment})
}

func (p *PaymentGateway) verifyCryptoPayment(c *gin.Context) {
	var req struct {
		Address string `json:"address" binding:"required"`
	}
	c.ShouldBindJSON(&req)

	verified, err := p.VerifyCryptoPayment(req.Address)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"verified": verified})
}

// ============================================================================
// Helper Functions
// ============================================================================

func generateProviderRef() string {
	data := fmt.Sprintf("%d:%s", time.Now().UnixNano(), uuid.New().String())
	hash := sha256.Sum256([]byte(data))
	return "pay_" + hex.EncodeToString(hash[:])[:24]
}

func generateCryptoAddress(currency, network string) string {
	data := fmt.Sprintf("%s:%s:%d", currency, network, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	switch currency {
	case "BTC":
		return "bc1" + base64.StdEncoding.EncodeToString(hash[:])[:38]
	case "ETH":
		return "0x" + hex.EncodeToString(hash[:])[:40]
	default:
		return "0x" + hex.EncodeToString(hash[:])[:40]
	}
}

// ============================================================================
// Main
// ============================================================================

func main() {
	cfg := LoadConfig()

	service, err := NewPaymentGateway(cfg)
	if err != nil {
		log.Fatalf("Failed to create payment gateway: %v", err)
	}

	router := gin.Default()
	service.setupRoutes(router)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down payment gateway...")
		os.Exit(0)
	}()

	log.Printf("Payment Gateway starting on port %s", cfg.ServerPort)
	if err := router.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
