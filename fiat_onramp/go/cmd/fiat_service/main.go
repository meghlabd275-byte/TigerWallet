/**
 * TigerWallet Fiat On-Ramp Service
 * Go backend for fiat-to-crypto purchases
 * Integrates with MoonPay, Transak, and Stripe
 */

package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/go-redis/redis/v8"
)

// Configuration
type Config struct {
	ServerPort        string
	RedisAddr         string
	MoonPayAPIKey     string
	MoonPaySecretKey  string
	MoonPayWebhookKey string
	TransakAPIKey     string
	TransakSecretKey  string
	StripeSecretKey   string
	StripeWebhookKey  string
	SupportFiat       []string
	SupportCrypto     []string
}

// Provider types
type Provider string

const (
	ProviderMoonPay Provider = "MOONPAY"
	ProviderTransak Provider = "TRANSAK"
	ProviderStripe  Provider = "STRIPE"
)

// Fiat Currency
type FiatCurrency struct {
	Code         string  `json:"code"`
	Name         string  `json:"name"`
	Symbol       string  `json:"symbol"`
	MinAmount    float64 `json:"min_amount"`
	MaxAmount    float64 `json:"max_amount"`
	DecimalPlaces int    `json:"decimal_places"`
}

// Crypto Currency
type CryptoCurrency struct {
	Code         string   `json:"code"`
	Name         string   `json:"name"`
	Network      string   `json:"network"`
	MinAmount    float64  `json:"min_amount"`
	MaxAmount    float64  `json:"max_amount"`
	DecimalPlaces int     `json:"decimal_places"`
}

// Order Status
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "PENDING"
	OrderStatusProcessing OrderStatus = "PROCESSING"
	OrderStatusCompleted OrderStatus = "COMPLETED"
	OrderStatusFailed    OrderStatus = "FAILED"
	OrderStatusCancelled OrderStatus = "CANCELLED"
	OrderStatusRefunded  OrderStatus = "REFUNDED"
)

// Order Types
type OrderType string

const (
	OrderTypeBuy  OrderType = "BUY"
	OrderTypeSell OrderType = "SELL"
)

// Order
type Order struct {
	OrderID          string            `json:"order_id"`
	Provider         Provider          `json:"provider"`
	UserID           string            `json:"user_id"`
	Type             OrderType         `json:"type"`
	FiatCurrency     string            `json:"fiat_currency"`
	FiatAmount       float64           `json:"fiat_amount"`
	CryptoCurrency   string            `json:"crypto_currency"`
	CryptoAmount     float64           `json:"crypto_amount"`
	ExchangeRate     float64           `json:"exchange_rate"`
	ProviderFee      float64           `json:"provider_fee"`
	PlatformFee      float64           `json:"platform_fee"`
	TotalAmount      float64           `json:"total_amount"`
	Status           OrderStatus       `json:"status"`
	WalletAddress    string            `json:"wallet_address"`
	Network          string            `json:"network"`
	RedirectURL      string            `json:"redirect_url"`
	CallbackURL      string            `json:"callback_url"`
	ExternalID       string            `json:"external_id"`
	PaymentMethod    string            `json:"payment_method"`
	TransactionHash  string            `json:"transaction_hash"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	CompletedAt       *time.Time        `json:"completed_at,omitempty"`
	FailureReason    string            `json:"failure_reason,omitempty"`
}

// Quote
type Quote struct {
	QuoteID         string    `json:"quote_id"`
	Provider        Provider  `json:"provider"`
	FiatCurrency    string    `json:"fiat_currency"`
	FiatAmount      float64   `json:"fiat_amount"`
	CryptoCurrency  string    `json:"crypto_currency"`
	CryptoAmount    float64   `json:"crypto_amount"`
	ExchangeRate    float64   `json:"exchange_rate"`
	TotalFee        float64   `json:"total_fee"`
	ValidUntil      time.Time `json:"valid_until"`
}

// CreateOrderRequest
type CreateOrderRequest struct {
	UserID         string   `json:"user_id" binding:"required"`
	Type           OrderType `json:"type" binding:"required"`
	FiatCurrency   string   `json:"fiat_currency" binding:"required"`
	FiatAmount     float64  `json:"fiat_amount" binding:"required"`
	CryptoCurrency string   `json:"crypto_currency" binding:"required"`
	WalletAddress  string   `json:"wallet_address" binding:"required"`
	Network        string   `json:"network"`
	PaymentMethod  string   `json:"payment_method"`
	RedirectURL    string   `json:"redirect_url"`
	CallbackURL    string   `json:"callback_url"`
}

// GetQuoteRequest
type GetQuoteRequest struct {
	Provider       Provider `json:"provider"`
	FiatCurrency   string   `json:"fiat_currency" binding:"required"`
	FiatAmount     float64  `json:"fiat_amount" binding:"required"`
	CryptoCurrency string   `json:"crypto_currency" binding:"required"`
}

// Fiat On-Ramp Service
type FiatOnRampService struct {
	config      Config
	redis       *redis.Client
	orders      map[string]*Order
	quotes      map[string]*Quote
	mu          sync.RWMutex
}

// NewFiatOnRampService creates a new fiat on-ramp service
func NewFiatOnRampService(cfg Config) *FiatOnRampService {
	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
		DB:   1,
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
	}

	return &FiatOnRampService{
		config: cfg,
		redis:  redisClient,
		orders: make(map[string]*Order),
		quotes: make(map[string]*Quote),
	}
}

// SupportedCurrencies returns supported fiat and crypto currencies
func (s *FiatOnRampService) SupportedCurrencies() (map[string]FiatCurrency, map[string]CryptoCurrency) {
	fiat := map[string]FiatCurrency{
		"USD": {Code: "USD", Name: "US Dollar", Symbol: "$", MinAmount: 30, MaxAmount: 25000, DecimalPlaces: 2},
		"EUR": {Code: "EUR", Name: "Euro", Symbol: "€", MinAmount: 30, MaxAmount: 25000, DecimalPlaces: 2},
		"GBP": {Code: "GBP", Name: "British Pound", Symbol: "£", MinAmount: 25, MaxAmount: 20000, DecimalPlaces: 2},
		"CNY": {Code: "CNY", Name: "Chinese Yuan", Symbol: "¥", MinAmount: 200, MaxAmount: 150000, DecimalPlaces: 2},
		"JPY": {Code: "JPY", Name: "Japanese Yen", Symbol: "¥", MinAmount: 4000, MaxAmount: 3000000, DecimalPlaces: 0},
		"KRW": {Code: "KRW", Name: "Korean Won", Symbol: "₩", MinAmount: 40000, MaxAmount: 30000000, DecimalPlaces: 0},
		"AUD": {Code: "AUD", Name: "Australian Dollar", Symbol: "A$", MinAmount: 45, MaxAmount: 35000, DecimalPlaces: 2},
		"CAD": {Code: "CAD", Name: "Canadian Dollar", Symbol: "C$", MinAmount: 40, MaxAmount: 32000, DecimalPlaces: 2},
		"INR": {Code: "INR", Name: "Indian Rupee", Symbol: "₹", MinAmount: 2500, MaxAmount: 1800000, DecimalPlaces: 2},
		"BRL": {Code: "BRL", Name: "Brazilian Real", Symbol: "R$", MinAmount: 150, MaxAmount: 100000, DecimalPlaces: 2},
	}

	crypto := map[string]CryptoCurrency{
		"BTC":  {Code: "BTC", Name: "Bitcoin", Network: "Bitcoin", MinAmount: 0.0001, MaxAmount: 10, DecimalPlaces: 8},
		"ETH":  {Code: "ETH", Name: "Ethereum", Network: "Ethereum", MinAmount: 0.001, MaxAmount: 100, DecimalPlaces: 8},
		"USDT": {Code: "USDT", Name: "Tether", Network: "Ethereum", MinAmount: 10, MaxAmount: 500000, DecimalPlaces: 2},
		"USDC": {Code: "USDC", Name: "USD Coin", Network: "Ethereum", MinAmount: 10, MaxAmount: 500000, DecimalPlaces: 2},
		"BNB":  {Code: "BNB", Name: "BNB", Network: "BNB Smart Chain", MinAmount: 0.01, MaxAmount: 1000, DecimalPlaces: 8},
		"SOL":  {Code: "SOL", Name: "Solana", Network: "Solana", MinAmount: 0.1, MaxAmount: 5000, DecimalPlaces: 8},
		"XRP":  {Code: "XRP", Name: "XRP", Network: "XRP Ledger", MinAmount: 10, MaxAmount: 100000, DecimalPlaces: 6},
		"ADA":  {Code: "ADA", Name: "Cardano", Network: "Cardano", MinAmount: 10, MaxAmount: 100000, DecimalPlaces: 6},
		"DOGE": {Code: "DOGE", Name: "Dogecoin", Network: "Dogecoin", MinAmount: 100, MaxAmount: 5000000, DecimalPlaces: 8},
		"TRX":  {Code: "TRX", Name: "TRON", Network: "TRON", MinAmount: 100, MaxAmount: 10000000, DecimalPlaces: 6},
		"MATIC": {Code: "MATIC", Name: "Polygon", Network: "Polygon", MinAmount: 10, MaxAmount: 500000, DecimalPlaces: 8},
		"AVAX": {Code: "AVAX", Name: "Avalanche", Network: "Avalanche", MinAmount: 1, MaxAmount: 50000, DecimalPlaces: 8},
		"LINK": {Code: "LINK", Name: "Chainlink", Network: "Ethereum", MinAmount: 1, MaxAmount: 100000, DecimalPlaces: 8},
		"UNI":  {Code: "UNI", Name: "Uniswap", Network: "Ethereum", MinAmount: 1, MaxAmount: 100000, DecimalPlaces: 8},
		"ATOM": {Code: "ATOM", Name: "Cosmos", Network: "Cosmos", MinAmount: 1, MaxAmount: 50000, DecimalPlaces: 8},
	}

	return fiat, crypto
}

// GetQuote returns a quote for the specified amount
func (s *FiatOnRampService) GetQuote(req GetQuoteRequest) (*Quote, error) {
	// Validate currencies
	fiatCurrencies, cryptoCurrencies := s.SupportedCurrencies()
	
	if _, ok := fiatCurrencies[req.FiatCurrency]; !ok {
		return nil, fmt.Errorf("unsupported fiat currency: %s", req.FiatCurrency)
	}
	
	if _, ok := cryptoCurrencies[req.CryptoCurrency]; !ok {
		return nil, fmt.Errorf("unsupported crypto currency: %s", req.CryptoCurrency)
	}
	
	// Validate amount
	fiat := fiatCurrencies[req.FiatCurrency]
	if req.FiatAmount < fiat.MinAmount || req.FiatAmount > fiat.MaxAmount {
		return nil, fmt.Errorf("amount must be between %v and %v", fiat.MinAmount, fiat.MaxAmount)
	}
	
	// Get exchange rate from provider
	rate, fee, err := s.getExchangeRate(req.Provider, req.FiatCurrency, req.FiatAmount, req.CryptoCurrency)
	if err != nil {
		return nil, err
	}
	
	// Calculate crypto amount
	cryptoAmount := (req.FiatAmount - fee) / rate
	
	quote := &Quote{
		QuoteID:        "QUOTE_" + uuid.New().String()[:12],
		Provider:       req.Provider,
		FiatCurrency:   req.FiatCurrency,
		FiatAmount:     req.FiatAmount,
		CryptoCurrency: req.CryptoCurrency,
		CryptoAmount:   cryptoAmount,
		ExchangeRate:   rate,
		TotalFee:       fee,
		ValidUntil:     time.Now().Add(10 * time.Minute),
	}
	
	// Store quote
	s.mu.Lock()
	s.quotes[quote.QuoteID] = quote
	s.mu.Unlock()
	
	return quote, nil
}

// getExchangeRate gets exchange rate from provider
func (s *FiatOnRampService) getExchangeRate(provider Provider, fiatCurrency string, fiatAmount float64, cryptoCurrency string) (rate float64, fee float64, err error) {
	// In production, call provider APIs
	// For now, use mock rates with real fee structure
	
	baseRates := map[string]float64{
		"BTC":  45000.0,
		"ETH":  2500.0,
		"USDT": 1.0,
		"USDC": 1.0,
		"BNB":  300.0,
		"SOL":  100.0,
		"XRP":  0.55,
		"ADA":  0.45,
		"DOGE": 0.08,
		"TRX":  0.1,
		"MATIC": 0.8,
		"AVAX": 35.0,
		"LINK": 15.0,
		"UNI":  6.0,
		"ATOM": 9.0,
	}
	
	rate, ok := baseRates[cryptoCurrency]
	if !ok {
		return 0, 0, fmt.Errorf("rate not available for %s", cryptoCurrency)
	}
	
	// Apply provider fee (typically 1-5%)
	var providerFeeRate float64
	switch provider {
	case ProviderMoonPay:
		providerFeeRate = 0.029 // 2.9%
	case ProviderTransak:
		providerFeeRate = 0.025 // 2.5%
	case ProviderStripe:
		providerFeeRate = 0.034 // 3.4% + $0.30
	default:
		providerFeeRate = 0.03
	}
	
	fee = fiatAmount * providerFeeRate
	if provider == ProviderStripe {
		fee += 0.30 // Stripe fixed fee
	}
	
	return rate, fee, nil
}

// CreateOrder creates a new order
func (s *FiatOnRampService) CreateOrder(req CreateOrderRequest) (*Order, error) {
	// Validate currencies
	fiatCurrencies, cryptoCurrencies := s.SupportedCurrencies()
	
	if _, ok := fiatCurrencies[req.FiatCurrency]; !ok {
		return nil, fmt.Errorf("unsupported fiat currency: %s", req.FiatCurrency)
	}
	
	if _, ok := cryptoCurrencies[req.CryptoCurrency]; !ok {
		return nil, fmt.Errorf("unsupported crypto currency: %s", req.CryptoCurrency)
	}
	
	// Validate amount
	fiat := fiatCurrencies[req.FiatCurrency]
	if req.FiatAmount < fiat.MinAmount || req.FiatAmount > fiat.MaxAmount {
		return nil, fmt.Errorf("amount must be between %v and %v", fiat.MinAmount, fiat.MaxAmount)
	}
	
	// Determine provider based on currency/payment method
	provider := s.determineProvider(req.FiatCurrency, req.PaymentMethod)
	
	// Get rate and calculate amounts
	rate, providerFee, err := s.getExchangeRate(provider, req.FiatCurrency, req.FiatAmount, req.CryptoCurrency)
	if err != nil {
		return nil, err
	}
	
	platformFee := req.FiatAmount * 0.005 // 0.5% platform fee
	cryptoAmount := (req.FiatAmount - providerFee - platformFee) / rate
	
	// Create order
	now := time.Now()
	order := &Order{
		OrderID:         "ORDER_" + uuid.New().String()[:12],
		Provider:        provider,
		UserID:          req.UserID,
		Type:            req.Type,
		FiatCurrency:    req.FiatCurrency,
		FiatAmount:      req.FiatAmount,
		CryptoCurrency:  req.CryptoCurrency,
		CryptoAmount:    cryptoAmount,
		ExchangeRate:    rate,
		ProviderFee:     providerFee,
		PlatformFee:     platformFee,
		TotalAmount:     req.FiatAmount,
		Status:          OrderStatusPending,
		WalletAddress:   req.WalletAddress,
		Network:         req.Network,
		RedirectURL:     req.RedirectURL,
		CallbackURL:     req.CallbackURL,
		PaymentMethod:   req.PaymentMethod,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	
	// Store order
	s.mu.Lock()
	s.orders[order.OrderID] = order
	s.mu.Unlock()
	
	// Persist to Redis
	ctx := context.Background()
	orderJSON, _ := json.Marshal(order)
	s.redis.Set(ctx, "fiat_order:"+order.OrderID, orderJSON, 24*time.Hour)
	
	return order, nil
}

// determineProvider determines the best provider based on currency and payment method
func (s *FiatOnRampService) determineProvider(fiatCurrency, paymentMethod string) Provider {
	// Map of supported currencies per provider
	moonPayFiat := []string{"USD", "EUR", "GBP", "AUD", "CAD"}
	transakFiat := []string{"USD", "EUR", "GBP", "INR", "BRL", "KRW"}
	
	for _, c := range moonPayFiat {
		if c == fiatCurrency {
			return ProviderMoonPay
		}
	}
	
	for _, c := range transakFiat {
		if c == fiatCurrency {
			return ProviderTransak
		}
	}
	
	// Default to Stripe
	return ProviderStripe
}

// GetOrder returns order by ID
func (s *FiatOnRampService) GetOrder(orderID string) (*Order, error) {
	s.mu.RLock()
	order, ok := s.orders[orderID]
	s.mu.RUnlock()
	
	if !ok {
		// Try Redis
		ctx := context.Background()
		orderJSON, err := s.redis.Get(ctx, "fiat_order:"+orderID).Result()
		if err != nil {
			return nil, fmt.Errorf("order not found")
		}
		
		var order Order
		if err := json.Unmarshal([]byte(orderJSON), &order); err != nil {
			return nil, err
		}
		return &order, nil
	}
	
	return order, nil
}

// GetUserOrders returns all orders for a user
func (s *FiatOnRampService) GetUserOrders(userID string) ([]*Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var orders []*Order
	for _, order := range s.orders {
		if order.UserID == userID {
			orders = append(orders, order)
		}
	}
	
	return orders, nil
}

// UpdateOrderStatus updates order status (called by webhook handlers)
func (s *FiatOnRampService) UpdateOrderStatus(orderID string, status OrderStatus, txHash string, failureReason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	order, ok := s.orders[orderID]
	if !ok {
		return fmt.Errorf("order not found")
	}
	
	order.Status = status
	order.UpdatedAt = time.Now()
	
	if txHash != "" {
		order.TransactionHash = txHash
	}
	
	if failureReason != "" {
		order.FailureReason = failureReason
	}
	
	if status == OrderStatusCompleted {
		now := time.Now()
		order.CompletedAt = &now
	}
	
	// Persist to Redis
	ctx := context.Background()
	orderJSON, _ := json.Marshal(order)
	s.redis.Set(ctx, "fiat_order:"+order.OrderID, orderJSON, 24*time.Hour)
	
	return nil
}

// VerifyWebhookSignature verifies webhook signature from provider
func (s *FiatOnRampService) VerifyWebhookSignature(provider Provider, payload []byte, signature string) bool {
	var secretKey string
	
	switch provider {
	case ProviderMoonPay:
		secretKey = s.config.MoonPaySecretKey
	case ProviderTransak:
		secretKey = s.config.TransakSecretKey
	case ProviderStripe:
		secretKey = s.config.StripeSecretKey
	}
	
	if secretKey == "" {
		return false
	}
	
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write(payload)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))
	
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// Webhook handlers for providers
func (s *FiatOnRampService) HandleMoonPayWebhook(c *gin.Context) {
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Verify signature
	signature := c.GetHeader("MoonPay-Signature")
	if signature == "" || !s.VerifyWebhookSignature(ProviderMoonPay, []byte(c.Request.Body.(*strings.Reader).Read([]byte{})), signature) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}
	
	// Process webhook
	externalID, _ := payload["externalId"].(string)
	statusStr, _ := payload["status"].(string)
	
	var status OrderStatus
	switch statusStr {
	case "completed":
		status = OrderStatusCompleted
	case "failed":
		status = OrderStatusFailed
	case "cancelled":
		status = OrderStatusCancelled
	default:
		status = OrderStatusProcessing
	}
	
	txHash, _ := payload["transactionHash"].(string)
	
	if err := s.UpdateOrderStatus(externalID, status, txHash, ""); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"status": "received"})
}

func (s *FiatOnRampService) HandleTransakWebhook(c *gin.Context) {
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	externalID, _ := payload["orderId"].(string)
	statusStr, _ := payload["status"].(string)
	
	var status OrderStatus
	switch statusStr {
	case "COMPLETED":
		status = OrderStatusCompleted
	case "FAILED":
		status = OrderStatusFailed
	case "CANCELLED":
		status = OrderStatusCancelled
	default:
		status = OrderStatusProcessing
	}
	
	txHash, _ := payload["cryptoTransactionHash"].(string)
	
	if err := s.UpdateOrderStatus(externalID, status, txHash, ""); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"status": "received"})
}

func (s *FiatOnRampService) HandleStripeWebhook(c *gin.Context) {
	payload := c.Request.Body
	signature := c.GetHeader("Stripe-Signature")
	
	// Verify signature
	if signature == "" || !s.VerifyWebhookSignature(ProviderStripe, []byte(payload.(string)), signature) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}
	
	var event map[string]interface{}
	if err := json.Unmarshal([]byte(payload.(string)), &event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Process event
	c.JSON(http.StatusOK, gin.H{"status": "received"})
}

// API Handlers
func (s *FiatOnRampService) GetCurrenciesHandler(c *gin.Context) {
	fiat, crypto := s.SupportedCurrencies()
	c.JSON(http.StatusOK, gin.H{
		"fiat":  fiat,
		"crypto": crypto,
	})
}

func (s *FiatOnRampService) GetQuoteHandler(c *gin.Context) {
	var req GetQuoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	quote, err := s.GetQuote(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, quote)
}

func (s *FiatOnRampService) CreateOrderHandler(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	order, err := s.CreateOrder(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, order)
}

func (s *FiatOnRampService) GetOrderHandler(c *gin.Context) {
	orderID := c.Param("id")
	
	order, err := s.GetOrder(orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, order)
}

func (s *FiatOnRampService) GetUserOrdersHandler(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}
	
	orders, err := s.GetUserOrders(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"orders": orders})
}

func (s *FiatOnRampService) SetupRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/fiat")
	{
		api.GET("/currencies", s.GetCurrenciesHandler)
		api.POST("/quote", s.GetQuoteHandler)
		api.POST("/order", s.CreateOrderHandler)
		api.GET("/order/:id", s.GetOrderHandler)
		api.GET("/orders", s.GetUserOrdersHandler)
		
		// Webhooks
		api.POST("/webhooks/moonpay", s.HandleMoonPayWebhook)
		api.POST("/webhooks/transak", s.HandleTransakWebhook)
		api.POST("/webhooks/stripe", s.HandleStripeWebhook)
	}
}

func main() {
	cfg := Config{
		ServerPort:        getEnv("FIAT_SERVICE_PORT", "8086"),
		RedisAddr:         getEnv("REDIS_ADDR", "localhost:6379"),
		MoonPayAPIKey:    getEnv("MOONPAY_API_KEY", ""),
		MoonPaySecretKey: getEnv("MOONPAY_SECRET_KEY", ""),
		TransakAPIKey:    getEnv("TRANSAK_API_KEY", ""),
		TransakSecretKey: getEnv("TRANSAK_SECRET_KEY", ""),
		StripeSecretKey:  getEnv("STRIPE_SECRET_KEY", ""),
	}

	service := NewFiatOnRampService(cfg)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "fiat-onramp-service",
			"timestamp": time.Now().Unix(),
		})
	})

	service.SetupRoutes(r)

	addr := ":" + cfg.ServerPort
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		log.Printf("Starting Fiat On-Ramp Service on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}

	log.Println("Server exited")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
