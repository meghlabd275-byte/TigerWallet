// TigerWallet Fiat On-Ramp Service
// High-Load Distributed Go Implementation
// Buy crypto with fiat currency via multiple payment providers

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

// FiatOrder represents a fiat purchase order
type FiatOrder struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	OrderID        string    `gorm:"uniqueIndex" json:"order_id"`
	UserAddress    string    `gorm:"index" json:"user_address"`
	Provider       string    `json:"provider"` // MOONPAY, TRANSAK, STRIPE
	FiatAmount    float64   `json:"fiat_amount"`
	FiatCurrency  string    `json:"fiat_currency"`
	CryptoAmount  float64   `json:"crypto_amount"`
	CryptoToken   string    `json:"crypto_token"`
	CryptoChain  int64     `json:"crypto_chain"`
	RecipientAddress string `json:"recipient_address"`
	Status         string    `json:"status"` // PENDING, PROCESSING, COMPLETED, FAILED, CANCELLED
	PaymentURL    string    `json:"payment_url"`
	PaymentStatus string    `json:"payment_status"` // PENDING, PAID, FAILED
	TxHash        string    `json:"tx_hash"`
	Fee           float64   `json:"fee"`
	Rate          float64   `json:"rate"`
	PartnerID     string    `json:"partner_id"`
	ExternalID    string    `json:"external_id"`
	ChainID       int64     `json:"chain_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	CompletedAt   *time.Time `json:"completed_at"`
}

// FiatRate represents exchange rates
type FiatRate struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Provider     string    `json:"provider"`
	FiatCurrency string    `json:"fiat_currency"`
	CryptoToken  string    `json:"crypto_token"`
	BuyRate      float64   `json:"buy_rate"`
	SellRate     float64   `json:"sell_rate"`
	MinAmount    float64   `json:"min_amount"`
	MaxAmount    float64   `json:"max_amount"`
	IsActive     bool      `json:"is_active"`
	ChainID      int64     `json:"chain_id"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// PaymentMethod represents available payment methods
type PaymentMethod struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Provider     string    `json:"provider"`
	MethodType   string    `json:"method_type"` // CARD, BANK_TRANSFER, APPLE_PAY, GOOGLE_PAY
	MethodName   string    `json:"method_name"`
	FeePercent   float64   `json:"fee_percent"`
	FeeFixed    float64   `json:"fee_fixed"`
	ProcessingTime string `json:"processing_time"`
	IsActive     bool      `json:"is_active"`
	Countries    string    `json:"countries"` // JSON array
}

// ============================================================================
// Service Implementation
// ============================================================================

type FiatOnRampService struct {
	db     *gorm.DB
	redis *redis.Client
	config Config
	mu    sync.RWMutex
}

func NewFiatOnRampService(config Config) (*FiatOnRampService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	err = db.AutoMigrate(
		&FiatOrder{},
		&FiatRate{},
		&PaymentMethod{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort),
	})

	service := &FiatOnRampService{
		db:     db,
		redis:  rdb,
		config: config,
	}

	// Initialize rates and payment methods
	go service.initializeData()

	return service, nil
}

func (s *FiatOnRampService) initializeData() {
	// Initialize rates
	rates := []FiatRate{
		{Provider: "MOONPAY", FiatCurrency: "USD", CryptoToken: "ETH", BuyRate: 1.0, MinAmount: 50, MaxAmount: 25000, IsActive: true, ChainID: 1},
		{Provider: "MOONPAY", FiatCurrency: "USD", CryptoToken: "BTC", BuyRate: 1.0, MinAmount: 50, MaxAmount: 25000, IsActive: true, ChainID: 1},
		{Provider: "MOONPAY", FiatCurrency: "USD", CryptoToken: "USDT", BuyRate: 1.0, MinAmount: 50, MaxAmount: 25000, IsActive: true, ChainID: 1},
		{Provider: "MOONPAY", FiatCurrency: "EUR", CryptoToken: "ETH", BuyRate: 0.92, MinAmount: 50, MaxAmount: 25000, IsActive: true, ChainID: 1},
		{Provider: "TRANSAK", FiatCurrency: "USD", CryptoToken: "ETH", BuyRate: 1.0, MinAmount: 30, MaxAmount: 30000, IsActive: true, ChainID: 1},
		{Provider: "TRANSAK", FiatCurrency: "USD", CryptoToken: "BTC", BuyRate: 1.0, MinAmount: 30, MaxAmount: 30000, IsActive: true, ChainID: 1},
		{Provider: "TRANSAK", FiatCurrency: "GBP", CryptoToken: "ETH", BuyRate: 0.79, MinAmount: 30, MaxAmount: 20000, IsActive: true, ChainID: 1},
	}

	for _, rate := range rates {
		var existing FiatRate
		query := s.db.Where("provider = ? AND fiat_currency = ? AND crypto_token = ?", 
			rate.Provider, rate.FiatCurrency, rate.CryptoToken)
		if query.First(&existing).RowsAffected == 0 {
			s.db.Create(&rate)
		}
	}

	// Initialize payment methods
	methods := []PaymentMethod{
		{Provider: "MOONPAY", MethodType: "CARD", MethodName: "Credit/Debit Card", FeePercent: 2.5, FeeFixed: 0.50, ProcessingTime: "Instant", IsActive: true, Countries: "[\"US\",\"UK\",\"EU\",\"AU\"]"},
		{Provider: "MOONPAY", MethodType: "BANK_TRANSFER", MethodName: "Bank Transfer", FeePercent: 1.0, FeeFixed: 0, ProcessingTime: "2-3 days", IsActive: true, Countries: "[\"US\",\"UK\",\"EU\"]"},
		{Provider: "TRANSAK", MethodType: "CARD", MethodName: "Credit/Debit Card", FeePercent: 2.99, FeeFixed: 0, ProcessingTime: "Instant", IsActive: true, Countries: "[\"US\",\"UK\",\"EU\",\"ASIA\"]"},
		{Provider: "TRANSAK", MethodType: "APPLE_PAY", MethodName: "Apple Pay", FeePercent: 1.5, FeeFixed: 0, ProcessingTime: "Instant", IsActive: true, Countries: "[\"US\",\"UK\",\"EU\"]"},
		{Provider: "STRIPE", MethodType: "CARD", MethodName: "Credit Card", FeePercent: 2.9, FeeFixed: 0.30, ProcessingTime: "Instant", IsActive: true, Countries: "[\"US\",\"UK\",\"EU\",\"CA\"]"},
	}

	for _, method := range methods {
		var existing PaymentMethod
		query := s.db.Where("provider = ? AND method_type = ?", method.Provider, method.MethodType)
		if query.First(&existing).RowsAffected == 0 {
			s.db.Create(&method)
		}
	}
}

// ============================================================================
// Order Operations
// ============================================================================

type CreateOrderRequest struct {
	UserAddress     string  `json:"user_address" binding:"required"`
	Provider       string  `json:"provider" binding:"required"`
	FiatAmount     float64 `json:"fiat_amount" binding:"required"`
	FiatCurrency   string  `json:"fiat_currency" binding:"required"`
	CryptoToken    string  `json:"crypto_token" binding:"required"`
	CryptoChain    int64   `json:"crypto_chain"`
	RecipientAddress string `json:"recipient_address" binding:"required"`
	PaymentMethod  string  `json:"payment_method" binding:"required"`
	Email          string  `json:"email"`
}

type CreateOrderResponse struct {
	Success     bool    `json:"success"`
	OrderID     string  `json:"order_id"`
	PaymentURL  string  `json:"payment_url"`
	CryptoAmount float64 `json:"crypto_amount"`
	Rate        float64 `json:"rate"`
	Fee         float64 `json:"fee"`
	ExpiresAt   int64   `json:"expires_at"`
	Error       string  `json:"error,omitempty"`
}

func (s *FiatOnRampService) CreateOrder(ctx *gin.Context) {
	var req CreateOrderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, CreateOrderResponse{Success: false, Error: err.Error()})
		return
	}

	// Get rate
	var rate FiatRate
	query := s.db.Where("provider = ? AND fiat_currency = ? AND crypto_token = ? AND is_active = ?",
		req.Provider, req.FiatCurrency, req.CryptoToken, true)
	if query.First(&rate).RowsAffected == 0 {
		ctx.JSON(400, CreateOrderResponse{Success: false, Error: "Rate not available for this pair"})
		return
	}

	// Validate amount
	if req.FiatAmount < rate.MinAmount {
		ctx.JSON(400, CreateOrderResponse{Success: false, Error: fmt.Sprintf("Minimum amount is %.2f", rate.MinAmount)})
		return
	}
	if req.FiatAmount > rate.MaxAmount {
		ctx.JSON(400, CreateOrderResponse{Success: false, Error: fmt.Sprintf("Maximum amount is %.2f", rate.MaxAmount)})
		return
	}

	// Get payment method fee
	var method PaymentMethod
	s.db.Where("provider = ? AND method_type = ?", req.Provider, req.PaymentMethod).First(&method)

	// Calculate crypto amount
	cryptoAmount := req.FiatAmount / rate.BuyRate

	// Calculate fee
	fee := req.FiatAmount*(method.FeePercent/100) + method.FeeFixed

	// Generate order
	orderID := generateOrderID(req.UserAddress, req.Provider)
	expiresAt := time.Now().Add(30 * time.Minute).Unix()

	order := FiatOrder{
		OrderID:          orderID,
		UserAddress:       req.UserAddress,
		Provider:         req.Provider,
		FiatAmount:       req.FiatAmount,
		FiatCurrency:     req.FiatCurrency,
		CryptoAmount:     cryptoAmount,
		CryptoToken:      req.CryptoToken,
		CryptoChain:      req.CryptoChain,
		RecipientAddress: req.RecipientAddress,
		Status:           "PENDING",
		PaymentStatus:    "PENDING",
		Fee:              fee,
		Rate:             rate.BuyRate,
		ChainID:          req.CryptoChain,
	}

	if err := s.db.Create(&order).Error; err != nil {
		ctx.JSON(500, CreateOrderResponse{Success: false, Error: "Failed to create order"})
		return
	}

	// Generate payment URL (in production, this would call provider API)
	paymentURL := fmt.Sprintf("https://%s.com/pay/%s", req.Provider, orderID)

	// Update order with payment URL
	order.PaymentURL = paymentURL
	s.db.Save(&order)

	// Cache order for status checks
	s.redis.Set(ctx.Request.Context(), fmt.Sprintf("fiat_order:%s", orderID), orderID, 30*time.Minute)

	ctx.JSON(200, CreateOrderResponse{
		Success:     true,
		OrderID:    orderID,
		PaymentURL: paymentURL,
		CryptoAmount: cryptoAmount,
		Rate:       rate.BuyRate,
		Fee:        fee,
		ExpiresAt:  expiresAt,
	})
}

type GetOrderStatusRequest struct {
	OrderID string `json:"order_id" binding:"required"`
}

func (s *FiatOnRampService) GetOrderStatus(ctx *gin.Context) {
	orderID := ctx.Param("id")

	var order FiatOrder
	if err := s.db.Where("order_id = ?", orderID).First(&order).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "Order not found"})
		return
	}

	ctx.JSON(200, gin.H{
		"order_id":          order.OrderID,
		"status":            order.Status,
		"payment_status":    order.PaymentStatus,
		"fiat_amount":       order.FiatAmount,
		"fiat_currency":    order.FiatCurrency,
		"crypto_amount":     order.CryptoAmount,
		"crypto_token":      order.CryptoToken,
		"recipient_address": order.RecipientAddress,
		"tx_hash":          order.TxHash,
		"created_at":        order.CreatedAt,
		"updated_at":        order.UpdatedAt,
	})
}

// Webhook handler for payment providers
type WebhookRequest struct {
	OrderID      string `json:"order_id"`
	Status       string `json:"status"`
	TxHash       string `json:"tx_hash"`
	ExternalID   string `json:"external_id"`
}

func (s *FiatOnRampService) HandleWebhook(ctx *gin.Context) {
	var req WebhookRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	var order FiatOrder
	if err := s.db.Where("order_id = ?", req.OrderID).First(&order).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "Order not found"})
		return
	}

	// Update order based on webhook
	order.PaymentStatus = req.Status

	switch req.Status {
	case "PAID":
		order.Status = "PROCESSING"
		// In production, would trigger crypto transfer
	case "COMPLETED":
		order.Status = "COMPLETED"
		order.TxHash = req.TxHash
		now := time.Now()
		order.CompletedAt = &now
	case "FAILED":
		order.Status = "FAILED"
	}

	s.db.Save(&order)

	ctx.JSON(200, gin.H{"success": true})
}

// Cancel order
func (s *FiatOnRampService) CancelOrder(ctx *gin.Context) {
	var req GetOrderStatusRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	var order FiatOrder
	if err := s.db.Where("order_id = ?", req.OrderID).First(&order).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "Order not found"})
		return
	}

	if order.Status != "PENDING" {
		ctx.JSON(400, gin.H{"error": "Order cannot be cancelled"})
		return
	}

	order.Status = "CANCELLED"
	s.db.Save(&order)

	ctx.JSON(200, gin.H{"success": true, "status": "CANCELLED"})
}

// ============================================================================
// Rate Queries
// ============================================================================

func (s *FiatOnRampService) GetRates(ctx *gin.Context) {
	provider := ctx.Query("provider")
	fiatCurrency := ctx.Query("currency")
	cryptoToken := ctx.Query("token")

	query := s.db.Where("is_active = ?", true)

	if provider != "" {
		query = query.Where("provider = ?", provider)
	}
	if fiatCurrency != "" {
		query = query.Where("fiat_currency = ?", fiatCurrency)
	}
	if cryptoToken != "" {
		query = query.Where("crypto_token = ?", cryptoToken)
	}

	var rates []FiatRate
	query.Find(&rates)

	ctx.JSON(200, gin.H{"rates": rates})
}

func (s *FiatOnRampService) GetPaymentMethods(ctx *gin.Context) {
	provider := ctx.Query("provider")
	country := ctx.Query("country")

	query := s.db.Where("is_active = ?", true)

	if provider != "" {
		query = query.Where("provider = ?", provider)
	}

	var methods []PaymentMethod
	query.Find(&methods)

	// Filter by country if provided
	if country != "" {
		filtered := make([]PaymentMethod, 0)
		for _, m := range methods {
			// In production, would parse JSON countries array
			filtered = append(filtered, m)
		}
		methods = filtered
	}

	ctx.JSON(200, gin.H{"methods": methods})
}

func (s *FiatOnRampService) GetSupportedFiat(ctx *gin.Context) {
	provider := ctx.Query("provider")

	query := s.db.Model(&FiatRate{}).Select("DISTINCT fiat_currency")
	if provider != "" {
		query = query.Where("provider = ?", provider)
	}

	var currencies []string
	query.Pluck("fiat_currency", &currencies)

	ctx.JSON(200, gin.H{"currencies": currencies})
}

func (s *FiatOnRampService) GetSupportedCrypto(ctx *gin.Context) {
	provider := ctx.Query("provider")

	query := s.db.Model(&FiatRate{}).Select("DISTINCT crypto_token").Where("is_active = ?", true)
	if provider != "" {
		query = query.Where("provider = ?", provider)
	}

	var tokens []string
	query.Pluck("crypto_token", &tokens)

	ctx.JSON(200, gin.H{"tokens": tokens})
}

// ============================================================================
// User Orders
// ============================================================================

func (s *FiatOnRampService) GetUserOrders(ctx *gin.Context) {
	userAddress := ctx.Query("user_address")
	status := ctx.Query("status")

	query := s.db.Where("user_address = ?", userAddress)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var orders []FiatOrder
	query.Order("created_at DESC").Find(&orders)

	ctx.JSON(200, gin.H{"orders": orders})
}

// ============================================================================
// Helper Functions
// ============================================================================

func generateOrderID(user, provider string) string {
	data := fmt.Sprintf("%s:%s:%d", user, provider, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return "fiat_" + hex.EncodeToString(hash[:])[0:16]
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := Config{
		ServerPort: "8100",
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "tigerwallet"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "tigerwallet_fiat"),
		RedisHost:  getEnv("REDIS_HOST", "localhost"),
		RedisPort:  getEnv("REDIS_PORT", "6379"),
	}

	service, err := NewFiatOnRampService(config)
	if err != nil {
		fmt.Printf("Failed to start fiat on-ramp service: %v\n", err)
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

	api := router.Group("/api/v1/fiat")
	{
		api.POST("/create-order", service.CreateOrder)
		api.GET("/orders/:id", service.GetOrderStatus)
		api.POST("/orders/:id/cancel", service.CancelOrder)
		api.GET("/orders", service.GetUserOrders)
		api.GET("/rates", service.GetRates)
		api.GET("/methods", service.GetPaymentMethods)
		api.GET("/fiat-currencies", service.GetSupportedFiat)
		api.GET("/crypto-tokens", service.GetSupportedCrypto)
		api.POST("/webhook", service.HandleWebhook)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "fiat_onramp"})
	})

	go func() {
		fmt.Printf("Fiat on-ramp service starting on port %s\n", config.ServerPort)
		if err := router.Run(":" + config.ServerPort); err != nil {
			fmt.Printf("Failed to start server: %v\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down fiat on-ramp service...")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
