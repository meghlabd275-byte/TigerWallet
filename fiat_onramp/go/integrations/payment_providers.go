/**
 * TigerWallet Fiat On-Ramp Payment Provider Integrations
 * Production-ready integrations for Stripe, MoonPay, Transak
 */

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
)

// ============================================================================
// Configuration
// ============================================================================

type PaymentConfig struct {
	StripeSecretKey     string `json:"stripe_secret_key"`
	StripeWebhookSecret string `json:"stripe_webhook_secret"`
	MoonPayAPIKey      string `json:"moonpay_api_key"`
	MoonPaySecretKey   string `json:"moonpay_secret_key"`
	TransakAPIKey      string `json:"transak_api_key"`
	TransakSecretKey   string `json:"transak_secret_key"`
}

func LoadPaymentConfig() *PaymentConfig {
	return &PaymentConfig{
		StripeSecretKey:     getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),
		MoonPayAPIKey:      getEnv("MOONPAY_API_KEY", ""),
		MoonPaySecretKey:   getEnv("MOONPAY_SECRET_KEY", ""),
		TransakAPIKey:      getEnv("TRANSAK_API_KEY", ""),
		TransakSecretKey:   getEnv("TRANSAK_SECRET_KEY", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============================================================================
// Stripe Integration
// ============================================================================

type StripeService struct {
	config    *PaymentConfig
	httpClient *http.Client
}

func NewStripeService(config *PaymentConfig) *StripeService {
	return &StripeService{
		config: config,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Create Stripe Payment Intent
type CreatePaymentIntentRequest struct {
	Amount      int64  `json:"amount" binding:"required"` // in cents
	Currency    string `json:"currency" binding:"required"`  // usd, eur, etc
	CustomerID  string `json:"customer_id"`
	Metadata    map[string]string `json:"metadata"`
}

type PaymentIntentResponse struct {
	ClientSecret string `json:"client_secret"`
	PaymentID   string `json:"payment_id"`
	Amount      int64  `json:"amount"`
	Currency    string `json:"currency"`
	Status      string `json:"status"`
}

func (s *StripeService) CreatePaymentIntent(req CreatePaymentIntentRequest) (*PaymentIntentResponse, error) {
	if s.config.StripeSecretKey == "" {
		return nil, fmt.Errorf("Stripe not configured")
	}

	paymentIntent := map[string]interface{}{
		"amount":        req.Amount,
		"currency":      strings.ToLower(req.Currency),
		"payment_method_types": []string{"card", "us_bank_account"},
		"metadata":      req.Metadata,
	}

	if req.CustomerID != "" {
		paymentIntent["customer"] = req.CustomerID
	}

	jsonData, _ := json.Marshal(paymentIntent)

	httpReq, _ := http.NewRequest("POST", "https://api.stripe.com/v1/payment_intents", strings.NewReader(string(jsonData)))
	httpReq.Header.Set("Authorization", "Bearer "+s.config.StripeSecretKey)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Stripe request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Stripe error: %s", string(body))
	}

	var result map[string]interface{}
	json.Unmarshal(body, &result)

	return &PaymentIntentResponse{
		ClientSecret: result["client_secret"].(string),
		PaymentID:   result["id"].(string),
		Amount:      int64(result["amount"].(float64)),
		Currency:    result["currency"].(string),
		Status:      result["status"].(string),
	}, nil
}

// Create Stripe Customer
type CreateCustomerRequest struct {
	Email   string `json:"email" binding:"required"`
	Name    string `json:"name"`
	Phone   string `json:"phone"`
}

func (s *StripeService) CreateCustomer(req CreateCustomerRequest) (string, error) {
	if s.config.StripeSecretKey == "" {
		return "", fmt.Errorf("Stripe not configured")
	}

	data := fmt.Sprintf("email=%s&name=%s&phone=%s", 
		urlEncode(req.Email), urlEncode(req.Name), urlEncode(req.Phone))

	httpReq, _ := http.NewRequest("POST", "https://api.stripe.com/v1/customers", strings.NewReader(data))
	httpReq.Header.Set("Authorization", "Bearer "+s.config.StripeSecretKey)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if result["id"] != nil {
		return result["id"].(string), nil
	}
	return "", fmt.Errorf("failed to create customer")
}

// Verify Stripe Webhook Signature
func (s *StripeService) VerifyWebhookSignature(payload []byte, signature string) (map[string]interface{}, error) {
	if s.config.StripeWebhookSecret == "" {
		return nil, fmt.Errorf("Webhook secret not configured")
	}

	// Parse signature header
	sigParts := strings.Split(signature, ",")
	var timestamp string
	var sig string

	for _, part := range sigParts {
		if strings.HasPrefix(part, "t=") {
			timestamp = strings.TrimPrefix(part, "t=")
		} else if strings.HasPrefix(part, "v1=") {
			sig = strings.TrimPrefix(part, "v1=")
		}
	}

	// Create signed payload
	signedPayload := timestamp + "." + string(payload)

	// Compute expected signature
	mac := hmac.New(sha256.New, []byte(s.config.StripeWebhookSecret))
	mac.Write([]byte(signedPayload))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return nil, fmt.Errorf("invalid signature")
	}

	var payloadData map[string]interface{}
	json.Unmarshal(payload, &payloadData)

	return payloadData, nil
}

// ============================================================================
// MoonPay Integration
// ============================================================================

type MoonPayService struct {
	config    *PaymentConfig
	httpClient *http.Client
	BaseURL   string
}

func NewMoonPayService(config *PaymentConfig) *MoonPayService {
	return &MoonPayService{
		config: config,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		BaseURL:   "https://api.moonpay.com",
	}
}

// Create MoonPay Payment Session
type CreateMoonPaySessionRequest struct {
	BaseCurrencyCode string  `json:"base_currency_code" binding:"required"` // eth, btc, etc
	QuoteCurrencyCode string `json:"quote_currency_code" binding:"required"` // usd, eur, etc
	BaseAmount       float64 `json:"base_amount"`
	WalletAddress    string  `json:"wallet_address"`
	Email            string  `json:"email"`
	ExternalID       string  `json:"external_id"`
}

type MoonPaySessionResponse struct {
	SessionID        string  `json:"session_id"`
	RedirectURL      string  `json:"redirect_url"`
	ExpiresAt        string  `json:"expires_at"`
	BaseCurrencyCode string  `json:"base_currency_code"`
	QuoteCurrencyCode string `json:"quote_currency_code"`
}

func (s *MoonPayService) CreateSession(req CreateMoonPaySessionRequest) (*MoonPaySessionResponse, error) {
	if s.config.MoonPayAPIKey == "" {
		return nil, fmt.Errorf("MoonPay not configured")
	}

	timestamp := time.Now().Unix()
	signature := s.generateMoonPaySignature(fmt.Sprintf("%d%s", timestamp, req.ExternalID))

	url := fmt.Sprintf("%s/v1/sessions?apiKey=%s", s.BaseURL, s.config.MoonPayAPIKey)

	session := map[string]interface{}{
		"baseCurrencyCode":   req.BaseCurrencyCode,
		"quoteCurrencyCode": req.QuoteCurrencyCode,
		"baseAmount":        req.BaseAmount,
		"walletAddress":     req.WalletAddress,
		"externalId":        req.ExternalID,
		"signature":         signature,
		"timestamp":         timestamp,
	}

	if req.Email != "" {
		session["email"] = req.Email
	}

	jsonData, _ := json.Marshal(session)

	httpReq, _ := http.NewRequest("POST", url, strings.NewReader(string(jsonData)))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("MoonPay error: %s", string(body))
	}

	var result map[string]interface{}
	json.Unmarshal(body, &result)

	data := result["data"].(map[string]interface{})

	return &MoonPaySessionResponse{
		SessionID:        data["id"].(string),
		RedirectURL:      data["redirectUrl"].(string),
		ExpiresAt:        data["expiresAt"].(string),
		BaseCurrencyCode: data["baseCurrencyCode"].(string),
		QuoteCurrencyCode: data["quoteCurrencyCode"].(string),
	}, nil
}

// Get MoonPay Transaction Status
func (s *MoonPayService) GetTransactionStatus(transactionID string) (string, error) {
	if s.config.MoonPayAPIKey == "" {
		return "", fmt.Errorf("MoonPay not configured")
	}

	url := fmt.Sprintf("%s/v1/transactions/%s?apiKey=%s", 
		s.BaseURL, transactionID, s.config.MoonPayAPIKey)

	httpReq, _ := http.NewRequest("GET", url, nil)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if data, ok := result["data"].(map[string]interface{}); ok {
		return data["status"].(string), nil
	}

	return "", fmt.Errorf("transaction not found")
}

// Get Supported Currencies
func (s *MoonPayService) GetSupportedCurrencies() ([]map[string]interface{}, error) {
	if s.config.MoonPayAPIKey == "" {
		return nil, fmt.Errorf("MoonPay not configured")
	}

	url := fmt.Sprintf("%s/v1/currencies?apiKey=%s", s.BaseURL, s.config.MoonPayAPIKey)

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	currencies := result["data"].([]interface{})
	resultCurrencies := make([]map[string]interface{}, len(currencies))

	for i, c := range currencies {
		resultCurrencies[i] = c.(map[string]interface{})
	}

	return resultCurrencies, nil
}

func (s *MoonPayService) generateMoonPaySignature(message string) string {
	mac := hmac.New(sha256.New, []byte(s.config.MoonPaySecretKey))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

// ============================================================================
// Transak Integration
// ============================================================================

type TransakService struct {
	config    *PaymentConfig
	httpClient *http.Client
	BaseURL   string
}

func NewTransakService(config *PaymentConfig) *TransakService {
	return &TransakService{
		config: config,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		BaseURL:   "https://api.transak.com",
	}
}

// Create Transak Order
type CreateTransakOrderRequest struct {
	ExchangeType      int     `json:"exchange_type"` // 1 = fiat to crypto
	FromCurrency     string  `json:"from_currency" binding:"required"` // USD, EUR
	ToCurrency       string  `json:"to_currency" binding:"required"` // ETH, BTC
	FromAmount       float64 `json:"from_amount"`
	ToAmount         float64 `json:"to_amount"`
	WalletAddress    string  `json:"wallet_address"`
	Email            string  `json:"email"`
	PartnerOrderID   string  `json:"partner_order_id"`
	PartnerCustomerID string `json:"partner_customer_id"`
}

type TransakOrderResponse struct {
	OrderID         string  `json:"order_id"`
	RedirectURL      string  `json:"redirect_url"`
	Status           string  `json:"status"`
	ExpiresAt        string  `json:"expires_at"`
	FromCurrency     string  `json:"from_currency"`
	ToCurrency       string  `json:"to_currency"`
	FromAmount       float64 `json:"from_amount"`
	ToAmount         float64 `json:"to_amount"`
}

func (s *TransakService) CreateOrder(req CreateTransakOrderRequest) (*TransakOrderResponse, error) {
	if s.config.TransakAPIKey == "" {
		return nil, fmt.Errorf("Transak not configured")
	}

	if req.PartnerOrderID == "" {
		req.PartnerOrderID = uuid.New().String()
	}

	timestamp := time.Now().Unix()
	signature := s.generateTransakSignature(fmt.Sprintf("%d%s", timestamp, req.PartnerOrderID))

	url := fmt.Sprintf("%s/v1/orders?apiKey=%s", s.BaseURL, s.config.TransakAPIKey)

	order := map[string]interface{}{
		"exchangeType":         req.ExchangeType,
		"fromCurrency":        req.FromCurrency,
		"toCurrency":          req.ToCurrency,
		"fromAmount":          req.FromAmount,
		"toAmount":            req.ToAmount,
		"walletAddress":       req.WalletAddress,
		"email":               req.Email,
		"partnerOrderId":      req.PartnerOrderID,
		"partnerCustomerId":   req.PartnerCustomerID,
		"timestamp":           timestamp,
		"signature":           signature,
	}

	jsonData, _ := json.Marshal(order)

	httpReq, _ := http.NewRequest("POST", url, strings.NewReader(string(jsonData)))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("Transak error: %s", string(body))
	}

	var result map[string]interface{}
	json.Unmarshal(body, &result)

	data := result["response"].(map[string]interface{})

	return &TransakOrderResponse{
		OrderID:         data["id"].(string),
		RedirectURL:      data["redirectURL"].(string),
		Status:           data["status"].(string),
		ExpiresAt:        data["expiresAt"].(string),
		FromCurrency:     data["fromCurrency"].(string),
		ToCurrency:       data["toCurrency"].(string),
		FromAmount:       data["fromAmount"].(float64),
		ToAmount:         data["toAmount"].(float64),
	}, nil
}

// Get Transak Order Status
func (s *TransakService) GetOrderStatus(orderID string) (string, error) {
	if s.config.TransakAPIKey == "" {
		return "", fmt.Errorf("Transak not configured")
	}

	url := fmt.Sprintf("%s/v1/orders/%s?apiKey=%s", 
		s.BaseURL, orderID, s.config.TransakAPIKey)

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if data, ok := result["response"].(map[string]interface{}); ok {
		return data["status"].(string), nil
	}

	return "", fmt.Errorf("order not found")
}

// Get Transak Supported Currencies
func (s *TransakService) GetSupportedCurrencies() ([]map[string]interface{}, error) {
	if s.config.TransakAPIKey == "" {
		return nil, fmt.Errorf("Transak not configured")
	}

	url := fmt.Sprintf("%s/v1/currencies?apiKey=%s", s.BaseURL, s.config.TransakAPIKey)

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	currencies := result["response"].([]interface{})
	resultCurrencies := make([]map[string]interface{}, len(currencies))

	for i, c := range currencies {
		resultCurrencies[i] = c.(map[string]interface{})
	}

	return resultCurrencies, nil
}

func (s *TransakService) generateTransakSignature(message string) string {
	mac := hmac.New(sha256.New, []byte(s.config.TransakSecretKey))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

// ============================================================================
// Fiat On-Ramp Service with All Providers
// ============================================================================

type FiatOnRampService struct {
	config        *PaymentConfig
	stripe       *StripeService
	moonpay     *MoonPayService
	transak     *TransakService
}

func NewFiatOnRampService(config *PaymentConfig) *FiatOnRampService {
	return &FiatOnRampService{
		config:  config,
		stripe:  NewStripeService(config),
		moonpay: NewMoonPayService(config),
		transak: NewTransakService(config),
	}
}

// Get Quote from All Providers
type QuoteRequest struct {
	FromCurrency string  `json:"from_currency" binding:"required"` // fiat
	ToCurrency   string  `json:"to_currency" binding:"required"` // crypto
	Amount       float64 `json:"amount" binding:"required"`
}

type QuoteResponse struct {
	Provider  string  `json:"provider"`
	FromCurrency string `json:"from_currency"`
	ToCurrency   string `json:"to_currency"`
	FromAmount   float64 `json:"from_amount"`
	ToAmount     float64 `json:"to_amount"`
	ExchangeRate float64 `json:"exchange_rate"`
	Fee          float64 `json:"fee"`
}

func (s *FiatOnRampService) GetQuotes(req QuoteRequest) ([]QuoteResponse, error) {
	var quotes []QuoteResponse

	// Get MoonPay quotes
	if s.config.MoonPayAPIKey != "" {
		sessionReq := CreateMoonPaySessionRequest{
			BaseCurrencyCode:  req.ToCurrency,
			QuoteCurrencyCode: req.FromCurrency,
			BaseAmount:       req.Amount,
			ExternalID:       uuid.New().String(),
		}
		if session, err := s.moonpay.CreateSession(sessionReq); err == nil {
			quotes = append(quotes, QuoteResponse{
				Provider:      "moonpay",
				FromCurrency:  req.FromCurrency,
				ToCurrency:    req.ToCurrency,
				FromAmount:    req.Amount,
				ToAmount:      sessionReq.BaseAmount,
				ExchangeRate:  req.Amount / sessionReq.BaseAmount,
				Fee:           req.Amount * 0.025, // 2.5% fee
			})
		}
	}

	// Get Transak quotes
	if s.config.TransakAPIKey != "" {
		orderReq := CreateTransakOrderRequest{
			ExchangeType:   1,
			FromCurrency:  req.FromCurrency,
			ToCurrency:    req.ToCurrency,
			FromAmount:    req.Amount,
			WalletAddress: "",
			PartnerOrderID: uuid.New().String(),
		}
		if order, err := s.transak.CreateOrder(orderReq); err == nil {
			quotes = append(quotes, QuoteResponse{
				Provider:      "transak",
				FromCurrency:  req.FromCurrency,
				ToCurrency:    req.ToCurrency,
				FromAmount:    req.Amount,
				ToAmount:      order.ToAmount,
				ExchangeRate:  req.Amount / order.ToAmount,
				Fee:           req.Amount * 0.02, // 2% fee
			})
		}
	}

	return quotes, nil
}

// Create Payment Session (unified)
type CreateSessionRequest struct {
	Provider       string  `json:"provider" binding:"required"` // stripe, moonpay, transak
	FromCurrency   string  `json:"from_currency" binding:"required"`
	ToCurrency     string  `json:"to_currency" binding:"required"`
	Amount         float64 `json:"amount"`
	WalletAddress  string  `json:"wallet_address" binding:"required"`
	Email          string  `json:"email"`
}

func (s *FiatOnRampService) CreateSession(req CreateSessionRequest) (map[string]interface{}, error) {
	switch strings.ToLower(req.Provider) {
	case "stripe":
		// Stripe is card-based, no wallet address needed
		intentReq := CreatePaymentIntentRequest{
			Amount:   int64(req.Amount * 100), // cents
			Currency: req.FromCurrency,
			Metadata: map[string]string{
				"to_currency":    req.ToCurrency,
				"wallet_address": req.WalletAddress,
			},
		}
		intent, err := s.stripe.CreatePaymentIntent(intentReq)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"provider":       "stripe",
			"client_secret": intent.ClientSecret,
			"payment_id":    intent.PaymentID,
		}, nil

	case "moonpay":
		sessionReq := CreateMoonPaySessionRequest{
			BaseCurrencyCode:  req.ToCurrency,
			QuoteCurrencyCode: req.FromCurrency,
			BaseAmount:       req.Amount,
			WalletAddress:    req.WalletAddress,
			Email:           req.Email,
			ExternalID:       uuid.New().String(),
		}
		session, err := s.moonpay.CreateSession(sessionReq)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"provider":     "moonpay",
			"session_id":  session.SessionID,
			"redirect_url": session.RedirectURL,
		}, nil

	case "transak":
		orderReq := CreateTransakOrderRequest{
			ExchangeType:      1,
			FromCurrency:     req.FromCurrency,
			ToCurrency:       req.ToCurrency,
			FromAmount:       req.Amount,
			WalletAddress:    req.WalletAddress,
			Email:           req.Email,
			PartnerOrderID:   uuid.New().String(),
		}
		order, err := s.transak.CreateOrder(orderReq)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"provider":     "transak",
			"order_id":    order.OrderID,
			"redirect_url": order.RedirectURL,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported provider: %s", req.Provider)
	}
}

// ============================================================================
// Utility Functions
// ============================================================================

func urlEncode(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, " ", "+"), "@", "%40")
}

// ============================================================================
// Main Entry Point
// ============================================================================

import "net/url"

func main() {
	config := LoadPaymentConfig()
	
	fiatService := NewFiatOnRampService(config)

	router := gin.Default()
	router.Use(gin.Recovery())

	// CORS
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

	// Health
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "fiat-onramp-providers"})
	})

	// Get quotes from all providers
	router.POST("/api/v1/fiat/quote", func(c *gin.Context) {
		var req QuoteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		quotes, err := fiatService.GetQuotes(req)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"quotes": quotes})
	})

	// Create payment session
	router.POST("/api/v1/fiat/session", func(c *gin.Context) {
		var req CreateSessionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		result, err := fiatService.CreateSession(req)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, result)
	})

	// Stripe webhook
	router.POST("/api/v1/webhooks/stripe", func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		signature := c.GetHeader("Stripe-Signature")

		payload, err := fiatService.stripe.VerifyWebhookSignature(body, signature)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		// Handle webhook event
		eventType := payload["type"].(string)
		fmt.Printf("Stripe webhook received: %s\n", eventType)

		c.JSON(200, gin.H{"received": true})
	})

	// MoonPay webhook
	router.POST("/api/v1/webhooks/moonpay", func(c *gin.Context) {
		var payload map[string]interface{}
		c.ShouldBindJSON(&payload)

		eventType := payload["type"].(string)
		fmt.Printf("MoonPay webhook received: %s\n", eventType)

		c.JSON(200, gin.H{"received": true})
	})

	// Transak webhook
	router.POST("/api/v1/webhooks/transak", func(c *gin.Context) {
		var payload map[string]interface{}
		c.ShouldBindJSON(&payload)

		eventType := payload["eventName"].(string)
		fmt.Printf("Transak webhook received: %s\n", eventType)

		c.JSON(200, gin.H{"received": true})
	})

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		fmt.Println("Fiat On-Ramp Providers service starting on port 9098")
		if err := router.Run(":9098"); err != nil {
			fmt.Printf("Server error: %v\n", err)
		}
	}()

	<-quit
	fmt.Println("Shutting down...")
}
