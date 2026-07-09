package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ============ Configuration ============

type Config struct {
	Port              string
	MoonPayAPIKey     string
	MoonPaySecretKey  string
	MoonPayURL        string
	RampAPIKey        string
	RampURL           string
	TransakAPIKey     string
	TransakURL        string
	RedisURL          string
	StripeSecretKey   string
	ApplePayMerchantID string
	GooglePayMerchantID string
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============ Models ============

type FiatProvider string

const (
	ProviderMoonPay FiatProvider = "moonpay"
	ProviderRamp    FiatProvider = "ramp"
	ProviderTransak FiatProvider = "transak"
	ProviderStripe  FiatProvider = "stripe"
)

type TransactionStatus string

const (
	StatusPending   TransactionStatus = "pending"
	StatusProcessing TransactionStatus = "processing"
	StatusCompleted TransactionStatus = "completed"
	StatusFailed    TransactionStatus = "failed"
	StatusCancelled TransactionStatus = "cancelled"
)

type CryptoNetwork string

const (
	NetworkEthereum CryptoNetwork = "ethereum"
	NetworkPolygon CryptoNetwork = "polygon"
	NetworkBSC      CryptoNetwork = "bsc"
	NetworkArbitrum CryptoNetwork = "arbitrum"
	NetworkOptimism CryptoNetwork = "optimism"
	NetworkSolana   CryptoNetwork = "solana"
	NetworkBitcoin  CryptoNetwork = "bitcoin"
)

// On-ramp request
type OnRampRequest struct {
	Provider       FiatProvider `json:"provider" binding:"required"`
	WalletAddress  string       `json:"walletAddress" binding:"required"`
	CryptoCurrency string       `json:"cryptoCurrency" binding:"required"`
	CryptoNetwork  CryptoNetwork `json:"cryptoNetwork" binding:"required"`
	FiatCurrency   string       `json:"fiatCurrency" binding:"required"`
	FiatAmount     float64      `json:"fiatAmount" binding:"required,gt=0"`
	PaymentMethod  string       `json:"paymentMethod"`
	Email          string       `json:"email"`
	ExternalID     string       `json:"externalId"`
}

// On-ramp response
type OnRampResponse struct {
	OrderID         string             `json:"orderId"`
	Provider        FiatProvider       `json:"provider"`
	QuoteID         string             `json:"quoteId"`
	WalletAddress   string             `json:"walletAddress"`
	CryptoAmount    float64            `json:"cryptoAmount"`
	CryptoCurrency string             `json:"cryptoCurrency"`
	FiatAmount      float64            `json:"fiatAmount"`
	FiatCurrency    string             `json:"fiatCurrency"`
	ExchangeRate   float64            `json:"exchangeRate"`
	Status         TransactionStatus  `json:"status"`
	RedirectURL    string             `json:"redirectUrl"`
	CreatedAt      time.Time          `json:"createdAt"`
	ExpiresAt      time.Time          `json:"expiresAt"`
}

// Off-ramp request
type OffRampRequest struct {
	Provider       FiatProvider `json:"provider" binding:"required"`
	WalletAddress  string       `json:"walletAddress" binding:"required"`
	CryptoCurrency string       `json:"cryptoCurrency" binding:"required"`
	CryptoNetwork  CryptoNetwork `json:"cryptoNetwork" binding:"required"`
	FiatCurrency   string       `json:"fiatCurrency" binding:"required"`
	CryptoAmount   float64      `json:"cryptoAmount" binding:"required,gt=0"`
	IBAN           string       `json:"iban" binding:"required"`
	SWIFTBIC       string       `json:"swiftBic"`
	BankName       string       `json:"bankName"`
	AccountName    string       `json:"accountName" binding:"required"`
	Email          string       `json:"email"`
}

// ============ Fiat Service ============

type FiatService struct {
	config      *Config
	httpClient  *http.Client
	redis       *redis.Client
	orderCache  map[string]*OnRampResponse
}

func NewFiatService(config *Config) (*FiatService, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.RedisURL,
		Password: "",
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		// Continue without Redis
		fmt.Printf("Warning: Redis not available: %v\n", err)
	}

	return &FiatService{
		config:     config,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		redis:      redisClient,
		orderCache: make(map[string]*OnRampResponse),
	}, nil
}

// Create On-Ramp Order
func (s *FiatService) CreateOnRampOrder(ctx context.Context, req OnRampRequest) (*OnRampResponse, error) {
	orderID := uuid.New().String()

	// Get quote from provider
	var quote struct {
		price         float64
		quoteID       string
		expiresAt     time.Time
	}

	switch req.Provider {
	case ProviderMoonPay:
		q, err := s.getMoonPayQuote(ctx, req)
		if err != nil {
			return nil, err
		}
		quote = *q

	case ProviderRamp:
		q, err := s.getRampQuote(ctx, req)
		if err != nil {
			return nil, err
		}
		quote = *q

	case ProviderTransak:
		q, err := s.getTransakQuote(ctx, req)
		if err != nil {
			return nil, err
		}
		quote = *q

	default:
		return nil, fmt.Errorf("unsupported provider: %s", req.Provider)
	}

	// Calculate crypto amount
	cryptoAmount := req.FiatAmount / quote.price

	response := &OnRampResponse{
		OrderID:         orderID,
		Provider:        req.Provider,
		QuoteID:         quote.quoteID,
		WalletAddress:   req.WalletAddress,
		CryptoAmount:    cryptoAmount,
		CryptoCurrency:  req.CryptoCurrency,
		FiatAmount:      req.FiatAmount,
		FiatCurrency:    req.FiatCurrency,
		ExchangeRate:    quote.price,
		Status:          StatusPending,
		RedirectURL:     fmt.Sprintf("%s?orderId=%s", s.getRedirectURL(req.Provider), orderID),
		CreatedAt:       time.Now(),
		ExpiresAt:       quote.expiresAt,
	}

	// Store in cache
	s.orderCache[orderID] = response

	// Store in Redis if available
	if s.redis != nil {
		data, _ := json.Marshal(response)
		s.redis.Set(ctx, fmt.Sprintf("fiat:order:%s", orderID), data, 30*time.Minute)
	}

	return response, nil
}

// Get MoonPay Quote
func (s *FiatService) getMoonPayQuote(ctx context.Context, req OnRampRequest) (*struct {
	price     float64
	quoteID   string
	expiresAt time.Time
}, error) {
	apiKey := s.config.MoonPayAPIKey
	url := fmt.Sprintf("%s/v1/currencies/%s/buy_quote?apiKey=%s&baseCurrencyAmount=%f&baseCurrencyCode=%s&fixedCurrencyCode=%s",
		s.config.MoonPayURL,
		strings.ToLower(req.CryptoCurrency),
		apiKey,
		req.FiatAmount,
		strings.ToLower(req.FiatCurrency),
		strings.ToLower(req.CryptoCurrency),
	)

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get MoonPay quote: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		QuoteID      string  `json:"quoteId"`
		Price        float64 `json:"price"`
		ExpiresAt    string  `json:"expiresAt"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse MoonPay response: %w", err)
	}

	expiresAt, _ := time.Parse(time.RFC3339, result.ExpiresAt)

	return &struct {
		price     float64
		quoteID   string
		expiresAt time.Time
	}{
		price:     result.Price,
		quoteID:   result.QuoteID,
		expiresAt: expiresAt,
	}, nil
}

// Get Ramp Quote
func (s *FiatService) getRampQuote(ctx context.Context, req OnRampRequest) (*struct {
	price     float64
	quoteID   string
	expiresAt time.Time
}, error) {
	url := fmt.Sprintf("%s/v1/hosted-on-ramp-orders", s.config.RampURL)

	payload := map[string]interface{}{
		"app_id":            s.config.RampAPIKey,
		"fiat_currency":     req.FiatCurrency,
		"crypto_currency":   req.CryptoCurrency,
		"fiat_amount":       req.FiatAmount,
		"wallet_address":    req.WalletAddress,
		"network":           string(req.CryptoNetwork),
	}

	data, _ := json.Marshal(payload)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(data)))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get Ramp quote: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		ID                string  `json:"id"`
		CryptoAmount      float64 `json:"crypto_amount"`
		FiatAmount        float64 `json:"fiat_amount"`
		ExchangeRate      float64 `json:"exchange_rate"`
		CreatedAt         int64   `json:"created_at"`
		ValidityPeriod    int     `json:"validity_period"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse Ramp response: %w", err)
	}

	price := result.FiatAmount / result.CryptoAmount

	return &struct {
		price     float64
		quoteID   string
		expiresAt time.Time
	}{
		price:     price,
		quoteID:   result.ID,
		expiresAt: time.Unix(result.CreatedAt, 0).Add(time.Duration(result.ValidityPeriod) * time.Second),
	}, nil
}

// Get Transak Quote
func (s *FiatService) getTransakQuote(ctx context.Context, req OnRampRequest) (*struct {
	price     float64
	quoteID   string
	expiresAt time.Time
}, error) {
	url := fmt.Sprintf("%s/v1/c报价", s.config.TransakURL)

	payload := map[string]interface{}{
		"apiKey":           s.config.TransakAPIKey,
		"fiatCurrency":     req.FiatCurrency,
		"cryptoCurrency":   req.CryptoCurrency,
		"fiatAmount":       req.FiatAmount,
		"network":          string(req.CryptoNetwork),
	}

	data, _ := json.Marshal(payload)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(data)))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get Transak quote: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		QuoteID      string  `json:"quoteId"`
		Price        float64 `json:"price"`
		ExpiresAt    int64   `json:"expiresAt"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse Transak response: %w", err)
	}

	return &struct {
		price     float64
		quoteID   string
		expiresAt time.Time
	}{
		price:     result.Price,
		quoteID:   result.QuoteID,
		expiresAt: time.Unix(result.ExpiresAt, 0),
	}, nil
}

// Get Order Status
func (s *FiatService) GetOrderStatus(ctx context.Context, orderID string) (*OnRampResponse, error) {
	// Check cache first
	if order, ok := s.orderCache[orderID]; ok {
		return order, nil
	}

	// Check Redis
	if s.redis != nil {
		data, err := s.redis.Get(ctx, fmt.Sprintf("fiat:order:%s", orderID)).Bytes()
		if err == nil {
			var order OnRampResponse
			json.Unmarshal(data, &order)
			return &order, nil
		}
	}

	return nil, fmt.Errorf("order not found: %s", orderID)
}

// Webhook Handler
func (s *FiatService) HandleWebhook(ctx context.Context, provider FiatProvider, payload []byte, signature string) error {
	// Verify signature
	if !s.verifyWebhookSignature(provider, payload, signature) {
		return fmt.Errorf("invalid webhook signature")
	}

	var webhookData map[string]interface{}
	if err := json.Unmarshal(payload, &webhookData); err != nil {
		return err
	}

	orderID, ok := webhookData["orderId"].(string)
	if !ok {
		return fmt.Errorf("missing order ID")
	}

	// Update order status
	order, err := s.GetOrderStatus(ctx, orderID)
	if err != nil {
		return err
	}

	status, ok := webhookData["status"].(string)
	if !ok {
		return fmt.Errorf("missing status")
	}

	switch status {
	case "completed":
		order.Status = StatusCompleted
	case "failed":
		order.Status = StatusFailed
	case "cancelled":
		order.Status = StatusCancelled
	case "processing":
		order.Status = StatusProcessing
	}

	// Update cache
	s.orderCache[orderID] = order

	// Update Redis
	if s.redis != nil {
		data, _ := json.Marshal(order)
		s.redis.Set(ctx, fmt.Sprintf("fiat:order:%s", orderID), data, 30*time.Minute)
	}

	return nil
}

// Verify Webhook Signature
func (s *FiatService) verifyWebhookSignature(provider FiatProvider, payload []byte, signature string) bool {
	var secret string
	switch provider {
	case ProviderMoonPay:
		secret = s.config.MoonPaySecretKey
	default:
		return true // Skip verification for others
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(signature))
}

func (s *FiatService) getRedirectURL(provider FiatProvider) string {
	switch provider {
	case ProviderMoonPay:
		return "https://buy.moonpay.com"
	case ProviderRamp:
		return "https://buy.ramp.network"
	case ProviderTransak:
		return "https://buy.transak.com"
	default:
		return ""
	}
}

// ============ HTTP Handlers ============

type Handler struct {
	fiatService *FiatService
}

func NewHandler(fiatService *FiatService) *Handler {
	return &Handler{fiatService: fiatService}
}

func (h *Handler) CreateOnRampOrder(c *gin.Context) {
	var req OnRampRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := h.fiatService.CreateOnRampOrder(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, order)
}

func (h *Handler) GetOrderStatus(c *gin.Context) {
	orderID := c.Param("orderId")

	order, err := h.fiatService.GetOrderStatus(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, order)
}

func (h *Handler) Webhook(c *gin.Context) {
	provider := FiatProvider(c.Param("provider"))

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	signature := c.GetHeader("X-Signature")

	err = h.fiatService.HandleWebhook(c.Request.Context(), provider, body, signature)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ============ Main ============

func main() {
	config := &Config{
		Port:              getEnv("PORT", "8080"),
		MoonPayAPIKey:     getEnv("MOONPAY_API_KEY", ""),
		MoonPaySecretKey:  getEnv("MOONPAY_SECRET_KEY", ""),
		MoonPayURL:        getEnv("MOONPAY_URL", "https://api.moonpay.com"),
		RampAPIKey:        getEnv("RAMP_API_KEY", ""),
		RampURL:           getEnv("RAMP_URL", "https://api.ramp.network"),
		TransakAPIKey:     getEnv("TRANSAK_API_KEY", ""),
		TransakURL:        getEnv("TRANSAK_URL", "https://api.transak.com"),
		RedisURL:          getEnv("REDIS_URL", "localhost:6379"),
		StripeSecretKey:   getEnv("STRIPE_SECRET_KEY", ""),
		ApplePayMerchantID: getEnv("APPLE_PAY_MERCHANT_ID", ""),
		GooglePayMerchantID: getEnv("GOOGLE_PAY_MERCHANT_ID", ""),
	}

	fiatService, err := NewFiatService(config)
	if err != nil {
		log.Fatalf("Failed to create fiat service: %v", err)
	}

	handler := NewHandler(fiatService)

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API routes
	api := router.Group("/api/v1")
	{
		// On-ramp
		api.POST("/onramp", handler.CreateOnRampOrder)
		api.GET("/onramp/:orderId", handler.GetOrderStatus)

		// Off-ramp
		api.POST("/offramp", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "not implemented"})
		})

		// Webhooks
		api.POST("/webhook/:provider", handler.Webhook)
	}

	// Start server
	addr := fmt.Sprintf(":%s", config.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		log.Printf("Starting Fiat On-Ramp service on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
