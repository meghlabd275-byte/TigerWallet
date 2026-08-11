/**
 * TigerWallet Fiat On-Ramp Service
 * Complete fiat-to-crypto gateway with multi-provider support
 * High-load worldwide distributed Go service
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
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ============================================================================
// Configuration
// ============================================================================

type FiatConfig struct {
	Port            int                       `json:"port"`
	RedisAddr       string                    `json:"redis_addr"`
	SupportedFiat   []string                  `json:"supported_fiat"`
	SupportedCrypto []string                  `json:"supported_crypto"`
	SupportedChains []string                  `json:"supported_chains"`
	Providers       map[string]ProviderConfig `json:"providers"`
	WebhookSecret   string                    `json:"webhook_secret"`
	MinOrderUSD     float64                   `json:"min_order_usd"`
	MaxOrderUSD     float64                   `json:"max_order_usd"`
	DefaultCrypto   string                    `json:"default_crypto"`
	DefaultChain    string                    `json:"default_chain"`
}

type ProviderConfig struct {
	Name          string   `json:"name"`
	Enabled       bool     `json:"enabled"`
	APIKey        string   `json:"api_key"`
	APISecret     string   `json:"api_secret"`
	WebhookURL    string   `json:"webhook_url"`
	WebhookSecret string   `json:"webhook_secret"`
	BaseURL       string   `json:"base_url"`
	FeePercent    float64  `json:"fee_percent"`
	MinOrder      float64  `json:"min_order"`
	MaxOrder      float64  `json:"max_order"`
	Currencies    []string `json:"currencies"`
}

var defaultConfig = FiatConfig{
	Port:            8451,
	RedisAddr:       "localhost:6379",
	SupportedFiat:   []string{"USD", "EUR", "GBP", "AUD", "CAD", "JPY", "KRW", "INR", "BRL"},
	SupportedCrypto: []string{"BTC", "ETH", "USDT", "USDC", "MATIC", "BNB", "SOL", "AVAX", "DOT", "ADA"},
	SupportedChains: []string{"ethereum", "polygon", "bsc", "arbitrum", "optimism", "avalanche", "solana"},
	Providers: map[string]ProviderConfig{
		"moonpay": {Name: "MoonPay", Enabled: true, FeePercent: 2.5, MinOrder: 30, MaxOrder: 5000, Currencies: []string{"USD", "EUR", "GBP"}},
		"transak": {Name: "Transak", Enabled: true, FeePercent: 2.0, MinOrder: 20, MaxOrder: 10000, Currencies: []string{"USD", "EUR", "GBP", "INR"}},
		"stripe":  {Name: "Stripe", Enabled: true, FeePercent: 1.5, MinOrder: 10, MaxOrder: 25000, Currencies: []string{"USD", "EUR", "GBP"}},
		"wyre":    {Name: "Wyre", Enabled: true, FeePercent: 1.8, MinOrder: 20, MaxOrder: 2500, Currencies: []string{"USD", "EUR", "GBP", "AUD", "CAD"}},
	},
	MinOrderUSD:   20,
	MaxOrderUSD:   25000,
	DefaultCrypto: "USDT",
	DefaultChain:  "polygon",
}

// ============================================================================
// Data Models
// ============================================================================

type Order struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id"`
	Provider        string     `json:"provider"`
	Status          string     `json:"status"`
	FiatAmount      float64    `json:"fiat_amount"`
	FiatCurrency    string     `json:"fiat_currency"`
	CryptoAmount    float64    `json:"crypto_amount"`
	CryptoCurrency  string     `json:"crypto_currency"`
	Chain           string     `json:"chain"`
	CryptoAddress   string     `json:"crypto_address"`
	ProviderOrderID string     `json:"provider_order_id"`
	PaymentURL      string     `json:"payment_url"`
	FiatEquivalent  float64    `json:"fiat_equivalent"`
	ExchangeRate    float64    `json:"exchange_rate"`
	FeeAmount       float64    `json:"fee_amount"`
	ProviderFee     float64    `json:"provider_fee"`
	IPAddress       string     `json:"ip_address"`
	WalletAddress   string     `json:"wallet_address"`
	UserAgent       string     `json:"user_agent"`
	ErrorMessage    string     `json:"error_message"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	CompletedAt     *time.Time `json:"completed_at"`
}

type QuoteRequest struct {
	FiatAmount     float64 `json:"fiat_amount"`
	FiatCurrency   string  `json:"fiat_currency"`
	CryptoCurrency string  `json:"crypto_currency"`
	Chain          string  `json:"chain"`
}

type QuoteResponse struct {
	Provider       string  `json:"provider"`
	CryptoAmount   float64 `json:"crypto_amount"`
	ExchangeRate   float64 `json:"exchange_rate"`
	FiatEquivalent float64 `json:"fiat_equivalent"`
	FeeAmount      float64 `json:"fee_amount"`
	ProviderFee    float64 `json:"provider_fee"`
	ValidUntil     int64   `json:"valid_until"`
}

type CreateOrderRequest struct {
	UserID         string  `json:"user_id"`
	FiatAmount     float64 `json:"fiat_amount"`
	FiatCurrency   string  `json:"fiat_currency"`
	CryptoCurrency string  `json:"crypto_currency"`
	Chain          string  `json:"chain"`
	CryptoAddress  string  `json:"crypto_address"`
	WalletAddress  string  `json:"wallet_address"`
	ReturnURL      string  `json:"return_url"`
	CallbackURL    string  `json:"callback_url"`
	Provider       string  `json:"provider"`
	IPAddress      string  `json:"-"`
	UserAgent      string  `json:"-"`
}

type CreateOrderResponse struct {
	OrderID         string `json:"order_id"`
	Provider        string `json:"provider"`
	PaymentURL      string `json:"payment_url"`
	ProviderOrderID string `json:"provider_order_id"`
	ExpiresAt       int64  `json:"expires_at"`
}

// ============================================================================
// Fiat Service
// ============================================================================

type FiatService struct {
	redis           *redis.Client
	config          *FiatConfig
	providerClients map[string]ProviderClient
	orderCache      map[string]*Order
	mu              sync.RWMutex
	exchangeRates   map[string]float64
	rateMu          sync.RWMutex
}

type ProviderClient interface {
	GetQuote(ctx context.Context, req *QuoteRequest) (*QuoteResponse, error)
	CreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error)
	GetOrderStatus(ctx context.Context, providerOrderID string) (*Order, error)
	HandleWebhook(ctx context.Context, payload []byte) (*Order, error)
}

func NewFiatService(config *FiatConfig) *FiatService {
	if config == nil {
		config = &defaultConfig
	}
	return &FiatService{
		redis:           redis.NewClient(&redis.Options{Addr: config.RedisAddr}),
		config:          config,
		providerClients: make(map[string]ProviderClient),
		orderCache:      make(map[string]*Order),
		exchangeRates:   make(map[string]float64),
	}
}

func (s *FiatService) initProviders() {
	for name, cfg := range s.config.Providers {
		if !cfg.Enabled {
			continue
		}
		var client ProviderClient
		switch name {
		case "moonpay":
			client = NewMoonPayClient(cfg)
		case "transak":
			client = NewTransakClient(cfg)
		case "stripe":
			client = NewStripeClient(cfg)
		case "wyre":
			client = NewWyreClient(cfg)
		}
		if client != nil {
			s.providerClients[name] = client
		}
	}
	go s.refreshExchangeRates()
}

func (s *FiatService) refreshExchangeRates() {
	s.refreshFromCoinGecko()
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		s.refreshFromCoinGecko()
	}
}

// refreshFromCoinGecko fetches REAL fiat->crypto exchange rates from
// CoinGecko's simple/price endpoint. On any error it leaves the existing
// rates in place (or empty if never set) - never fabricates a rate.
func (s *FiatService) refreshFromCoinGecko() {
	coinIDs := map[string]string{
		"BTC": "bitcoin", "ETH": "ethereum", "USDT": "tether",
		"USDC": "usd-coin", "MATIC": "matic-network", "BNB": "binancecoin",
		"SOL": "solana", "AVAX": "avalanche-2", "DOT": "polkadot", "ADA": "cardano",
	}
	vsCurrencies := strings.Join(s.config.SupportedFiat, ",")
	ids := make([]string, 0, len(coinIDs))
	for _, id := range coinIDs {
		ids = append(ids, id)
	}
	url := "https://api.coingecko.com/api/v3/simple/price?ids=" + strings.Join(ids, ",") +
		"&vs_currencies=" + vsCurrencies

	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return
	}
	var data map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return
	}

	rates := make(map[string]float64)
	for symbol, id := range coinIDs {
		prices, ok := data[id]
		if !ok {
			continue
		}
		for _, fiat := range s.config.SupportedFiat {
			if price, ok := prices[strings.ToLower(fiat)]; ok && price > 0 {
				rates[fiat+"-"+symbol] = 1.0 / price
			}
		}
	}

	s.rateMu.Lock()
	if len(rates) > 0 {
		s.exchangeRates = rates
	}
	s.rateMu.Unlock()
}

// ============================================================================
// Quote Methods
// ============================================================================

func (s *FiatService) GetBestQuote(ctx context.Context, req *QuoteRequest) (*QuoteResponse, error) {
	if err := s.validateQuoteRequest(req); err != nil {
		return nil, err
	}

	type qr struct {
		provider string
		quote    *QuoteResponse
		err      error
	}

	results := make(chan qr, len(s.providerClients))
	var wg sync.WaitGroup

	for name, client := range s.providerClients {
		wg.Add(1)
		go func(name string, client ProviderClient) {
			defer wg.Done()
			quote, err := client.GetQuote(ctx, req)
			results <- qr{name, quote, err}
		}(name, client)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var bestQuote *QuoteResponse
	var bestProvider string

	for result := range results {
		if result.err != nil {
			continue
		}
		if bestQuote == nil || result.quote.CryptoAmount > bestQuote.CryptoAmount {
			bestQuote = result.quote
			bestProvider = result.provider
		}
	}

	if bestQuote == nil {
		return nil, fmt.Errorf("no providers available")
	}

	bestQuote.Provider = bestProvider
	return bestQuote, nil
}

func (s *FiatService) validateQuoteRequest(req *QuoteRequest) error {
	if req.FiatAmount < s.config.MinOrderUSD {
		return fmt.Errorf("minimum order amount is %.2f USD", s.config.MinOrderUSD)
	}
	if req.FiatAmount > s.config.MaxOrderUSD {
		return fmt.Errorf("maximum order amount is %.2f USD", s.config.MaxOrderUSD)
	}

	validFiat := false
	for _, c := range s.config.SupportedFiat {
		if strings.ToUpper(req.FiatCurrency) == c {
			validFiat = true
			break
		}
	}
	if !validFiat {
		return fmt.Errorf("unsupported fiat currency: %s", req.FiatCurrency)
	}

	validCrypto := false
	for _, c := range s.config.SupportedCrypto {
		if strings.ToUpper(req.CryptoCurrency) == c {
			validCrypto = true
			break
		}
	}
	if !validCrypto {
		return fmt.Errorf("unsupported crypto currency: %s", req.CryptoCurrency)
	}

	return nil
}

// ============================================================================
// Order Methods
// ============================================================================

func (s *FiatService) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
	if err := s.validateCreateOrderRequest(req); err != nil {
		return nil, err
	}

	provider := req.Provider
	if provider == "" {
		provider = s.selectBestProvider(req.FiatCurrency, req.CryptoCurrency)
	}

	client, ok := s.providerClients[provider]
	if !ok {
		return nil, fmt.Errorf("provider not available: %s", provider)
	}

	orderID := uuid.New().String()

	quoteReq := &QuoteRequest{
		FiatAmount:     req.FiatAmount,
		FiatCurrency:   req.FiatCurrency,
		CryptoCurrency: req.CryptoCurrency,
		Chain:          req.Chain,
	}

	quote, err := client.GetQuote(ctx, quoteReq)
	if err != nil {
		return nil, err
	}

	order := &Order{
		ID:             orderID,
		UserID:         req.UserID,
		Provider:       provider,
		Status:         "pending",
		FiatAmount:     req.FiatAmount,
		FiatCurrency:   strings.ToUpper(req.FiatCurrency),
		CryptoCurrency: strings.ToUpper(req.CryptoCurrency),
		Chain:          req.Chain,
		CryptoAddress:  req.CryptoAddress,
		WalletAddress:  req.WalletAddress,
		IPAddress:      req.IPAddress,
		UserAgent:      req.UserAgent,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		CryptoAmount:   quote.CryptoAmount,
		FiatEquivalent: quote.FiatEquivalent,
		ExchangeRate:   quote.ExchangeRate,
		FeeAmount:      quote.FeeAmount,
		ProviderFee:    quote.ProviderFee,
	}

	createReq := &CreateOrderRequest{
		UserID:         req.UserID,
		FiatAmount:     req.FiatAmount,
		FiatCurrency:   req.FiatCurrency,
		CryptoCurrency: req.CryptoCurrency,
		Chain:          req.Chain,
		CryptoAddress:  req.CryptoAddress,
		WalletAddress:  req.WalletAddress,
		ReturnURL:      req.ReturnURL,
		CallbackURL:    req.CallbackURL,
	}

	providerResp, err := client.CreateOrder(ctx, createReq)
	if err != nil {
		order.Status = "failed"
		order.ErrorMessage = err.Error()
		s.saveOrder(order)
		return nil, err
	}

	order.ProviderOrderID = providerResp.ProviderOrderID
	order.PaymentURL = providerResp.PaymentURL

	s.saveOrder(order)

	return &CreateOrderResponse{
		OrderID:         orderID,
		Provider:        provider,
		PaymentURL:      providerResp.PaymentURL,
		ProviderOrderID: providerResp.ProviderOrderID,
		ExpiresAt:       providerResp.ExpiresAt,
	}, nil
}

func (s *FiatService) GetOrder(ctx context.Context, orderID string) (*Order, error) {
	s.mu.RLock()
	order, ok := s.orderCache[orderID]
	s.mu.RUnlock()

	if ok {
		return order, nil
	}

	key := fmt.Sprintf("fiat:order:%s", orderID)
	data, err := s.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("order not found")
	}
	if err != nil {
		return nil, err
	}

	var cachedOrder Order
	if err := json.Unmarshal([]byte(data), &cachedOrder); err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.orderCache[orderID] = &cachedOrder
	s.mu.Unlock()

	return &cachedOrder, nil
}

func (s *FiatService) saveOrder(order *Order) error {
	data, err := json.Marshal(order)
	if err != nil {
		return err
	}

	ctx := context.Background()
	orderKey := fmt.Sprintf("fiat:order:%s", order.ID)
	if err := s.redis.Set(ctx, orderKey, data, 7*24*time.Hour).Err(); err != nil {
		return err
	}

	s.mu.Lock()
	s.orderCache[order.ID] = order
	s.mu.Unlock()

	return nil
}

func (s *FiatService) validateCreateOrderRequest(req *CreateOrderRequest) error {
	if req.FiatAmount < s.config.MinOrderUSD {
		return fmt.Errorf("minimum order amount is %.2f USD", s.config.MinOrderUSD)
	}
	if req.FiatAmount > s.config.MaxOrderUSD {
		return fmt.Errorf("maximum order amount is %.2f USD", s.config.MaxOrderUSD)
	}
	if req.CryptoAddress == "" {
		return fmt.Errorf("crypto address is required")
	}
	if !strings.HasPrefix(req.CryptoAddress, "0x") || len(req.CryptoAddress) != 42 {
		return fmt.Errorf("invalid crypto address format")
	}
	return nil
}

func (s *FiatService) selectBestProvider(fiatCurrency, cryptoCurrency string) string {
	var bestProvider string
	var bestRate float64

	for name, cfg := range s.config.Providers {
		if !cfg.Enabled {
			continue
		}
		valid := false
		for _, c := range cfg.Currencies {
			if strings.ToUpper(fiatCurrency) == c {
				valid = true
				break
			}
		}
		if !valid {
			continue
		}
		if bestProvider == "" || cfg.FeePercent < bestRate {
			bestProvider = name
			bestRate = cfg.FeePercent
		}
	}
	return bestProvider
}

// ============================================================================
// Webhook Handler
// ============================================================================

func (s *FiatService) HandleWebhook(c *gin.Context) {
	provider := c.Param("provider")

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	signature := c.GetHeader("X-Signature")
	if !s.verifyWebhookSignature(provider, body, signature) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	client, ok := s.providerClients[provider]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
		return
	}

	order, err := client.HandleWebhook(c.Request.Context(), body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.saveOrder(order)
	c.JSON(http.StatusOK, gin.H{"success": true, "order": order})
}

func (s *FiatService) verifyWebhookSignature(provider string, body []byte, signature string) bool {
	if signature == "" {
		return false
	}

	cfg, ok := s.config.Providers[provider]
	if !ok {
		return false
	}

	mac := hmac.New(sha256.New, []byte(cfg.WebhookSecret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expected))
}

// ============================================================================
// API Routes
// ============================================================================

func (s *FiatService) SetupRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/fiat")
	{
		api.GET("/quote", s.GetQuoteEndpoint)
		api.POST("/order", s.CreateOrderEndpoint)
		api.GET("/order/:id", s.GetOrderEndpoint)
		api.GET("/rates", s.GetExchangeRatesEndpoint)
		api.GET("/currencies", s.GetSupportedCurrenciesEndpoint)
		api.POST("/webhook/:provider", s.HandleWebhook)
	}
}

func (s *FiatService) GetQuoteEndpoint(c *gin.Context) {
	var req QuoteRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	quote, err := s.GetBestQuote(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "quote": quote})
}

func (s *FiatService) CreateOrderEndpoint(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.IPAddress = c.ClientIP()
	req.UserAgent = c.Request.UserAgent()

	resp, err := s.CreateOrder(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "order": resp})
}

func (s *FiatService) GetOrderEndpoint(c *gin.Context) {
	orderID := c.Param("id")

	order, err := s.GetOrder(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "order": order})
}

func (s *FiatService) GetExchangeRatesEndpoint(c *gin.Context) {
	s.rateMu.RLock()
	rates := s.exchangeRates
	s.rateMu.RUnlock()

	c.JSON(http.StatusOK, gin.H{"success": true, "rates": rates})
}

func (s *FiatService) GetSupportedCurrenciesEndpoint(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"fiat":    s.config.SupportedFiat,
		"crypto":  s.config.SupportedCrypto,
		"chains":  s.config.SupportedChains,
	})
}

// ============================================================================
// Provider Clients
// ============================================================================

type MoonPayClient struct{ config ProviderConfig }

func NewMoonPayClient(cfg ProviderConfig) *MoonPayClient { return &MoonPayClient{config: cfg} }

func (c *MoonPayClient) GetQuote(ctx context.Context, req *QuoteRequest) (*QuoteResponse, error) {
	if c.config.APIKey == "" {
		return nil, fmt.Errorf("MoonPay API key not configured; cannot fetch a real quote")
	}
	// Real MoonPay API: GET /v3/currencies/{currency}/quote?baseCurrencyAmount=...
	base := c.config.BaseURL
	if base == "" {
		base = "https://api.moonpay.com"
	}
	url := fmt.Sprintf("%s/v3/currencies/%s/quote?baseCurrencyCode=%s&baseCurrencyAmount=%.2f&areFeesIncluded=true",
		base, strings.ToLower(req.CryptoCurrency), strings.ToLower(req.FiatCurrency), req.FiatAmount)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Api-Key "+c.config.APIKey)
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("MoonPay API returned status %d", resp.StatusCode)
	}
	var apiResp struct {
		BaseCurrencyAmount float64 `json:"baseCurrencyAmount"`
		QuoteCurrencyAmount float64 `json:"quoteCurrencyAmount"`
		FeeAmount           float64 `json:"feeAmount"`
		TotalAmount         float64 `json:"totalAmount"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}
	rate := 0.0
	if apiResp.BaseCurrencyAmount > 0 {
		rate = apiResp.QuoteCurrencyAmount / apiResp.BaseCurrencyAmount
	}
	return &QuoteResponse{
		CryptoAmount:   apiResp.QuoteCurrencyAmount,
		ExchangeRate:   rate,
		FiatEquivalent: req.FiatAmount,
		FeeAmount:      apiResp.FeeAmount,
		ProviderFee:    apiResp.FeeAmount,
		ValidUntil:     time.Now().Add(5 * time.Minute).Unix(),
	}, nil
}

func (c *MoonPayClient) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
	if c.config.APIKey == "" {
		return nil, fmt.Errorf("MoonPay API key not configured; cannot create a real order")
	}
	// Real MoonPay hosted widget URL. The client opens this; MoonPay handles
	// payment + redirects. We do NOT fabricate an order id / payment URL.
	base := c.config.BaseURL
	if base == "" {
		base = "https://buy.moonpay.com"
	}
	params := url.Values{}
	params.Set("apiKey", c.config.APIKey)
	params.Set("baseCurrencyCode", strings.ToLower(req.FiatCurrency))
	params.Set("baseCurrencyAmount", fmt.Sprintf("%.2f", req.FiatAmount))
	params.Set("currencyCode", strings.ToLower(req.CryptoCurrency))
	params.Set("walletAddress", req.WalletAddress)
	return &CreateOrderResponse{
		OrderID:    uuid.New().String(),
		PaymentURL: base + "?" + params.Encode(),
		ExpiresAt: time.Now().Add(30 * time.Minute).Unix(),
	}, nil
}

func (c *MoonPayClient) GetOrderStatus(ctx context.Context, providerOrderID string) (*Order, error) {
	if c.config.APIKey == "" {
		return nil, fmt.Errorf("MoonPay API key not configured")
	}
	base := c.config.BaseURL
	if base == "" {
		base = "https://api.moonpay.com"
	}
	url := base + "/v1/transactions/" + providerOrderID
	httpReq, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	httpReq.Header.Set("Authorization", "Api-Key "+c.config.APIKey)
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("MoonPay API returned status %d", resp.StatusCode)
	}
	var tx struct {
		Status string `json:"status"`
		ID     string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tx); err != nil {
		return nil, err
	}
	return &Order{ProviderOrderID: tx.ID, Status: tx.Status}, nil
}

func (c *MoonPayClient) HandleWebhook(ctx context.Context, payload []byte) (*Order, error) {
	// MoonPay webhooks are signed with a secret in the X-Webhook-Signature
	// header. Without the secret we CANNOT verify the payload - returning
	// "completed" would be a payment-confirmation vulnerability.
	if c.config.WebhookSecret == "" {
		return nil, fmt.Errorf("MoonPay webhook secret not configured; payload cannot be verified")
	}
	// Caller must extract the signature header and pass it; this stub rejects
	// all webhooks until real signature verification is wired (fail-closed).
	return nil, fmt.Errorf("webhook signature verification not implemented; rejecting unverified payload")
}

type TransakClient struct{ config ProviderConfig }

func NewTransakClient(cfg ProviderConfig) *TransakClient { return &TransakClient{config: cfg} }

func (c *TransakClient) GetQuote(ctx context.Context, req *QuoteRequest) (*QuoteResponse, error) {
	if c.config.APIKey == "" {
		return nil, fmt.Errorf("Transak API key not configured; cannot fetch a real quote")
	}
	base := c.config.BaseURL
	if base == "" {
		base = "https://api.transak.com"
	}
	u := base + "/api/v1/currencies/price?baseCurrency=" + strings.ToLower(req.FiatCurrency) +
		"&baseAmount=" + fmt.Sprintf("%.2f", req.FiatAmount) +
		"&cryptoCurrency=" + strings.ToLower(req.CryptoCurrency)
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Transak API returned status %d", resp.StatusCode)
	}
	var apiResp struct {
		Response struct {
			TotalAmount float64 `json:"totalAmount"`
			CryptoAmount float64 `json:"cryptoAmount"`
			Fee         float64 `json:"fee"`
		} `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}
	rate := 0.0
	if req.FiatAmount > 0 {
		rate = apiResp.Response.CryptoAmount / req.FiatAmount
	}
	return &QuoteResponse{
		CryptoAmount:   apiResp.Response.CryptoAmount,
		ExchangeRate:   rate,
		FiatEquivalent: req.FiatAmount,
		FeeAmount:      apiResp.Response.Fee,
		ProviderFee:    apiResp.Response.Fee,
		ValidUntil:     time.Now().Add(5 * time.Minute).Unix(),
	}, nil
}

func (c *TransakClient) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
	if c.config.APIKey == "" {
		return nil, fmt.Errorf("Transak API key not configured; cannot create a real order")
	}
	base := c.config.BaseURL
	if base == "" {
		base = "https://global.transak.com"
	}
	params := url.Values{}
	params.Set("apiKey", c.config.APIKey)
	params.Set("defaultFiatCurrency", strings.ToLower(req.FiatCurrency))
	params.Set("fiatAmount", fmt.Sprintf("%.2f", req.FiatAmount))
	params.Set("cryptoCurrencyCode", strings.ToLower(req.CryptoCurrency))
	params.Set("walletAddress", req.WalletAddress)
	return &CreateOrderResponse{
		OrderID:    uuid.New().String(),
		PaymentURL: base + "?" + params.Encode(),
		ExpiresAt: time.Now().Add(30 * time.Minute).Unix(),
	}, nil
}

func (c *TransakClient) GetOrderStatus(ctx context.Context, providerOrderID string) (*Order, error) {
	if c.config.APIKey == "" {
		return nil, fmt.Errorf("Transak API key not configured")
	}
	base := c.config.BaseURL
	if base == "" {
		base = "https://api.transak.com"
	}
	resp, err := http.Get(base + "/api/v1/partner/order/" + providerOrderID)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Transak API returned status %d", resp.StatusCode)
	}
	var o struct {
		Response struct {
			Status string `json:"status"`
			ID     string `json:"id"`
		} `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&o); err != nil {
		return nil, err
	}
	return &Order{ProviderOrderID: o.Response.ID, Status: o.Response.Status}, nil
}

func (c *TransakClient) HandleWebhook(ctx context.Context, payload []byte) (*Order, error) {
	if c.config.WebhookSecret == "" {
		return nil, fmt.Errorf("Transak webhook secret not configured; payload cannot be verified")
	}
	return nil, fmt.Errorf("webhook signature verification not implemented; rejecting unverified payload")
}

type StripeClient struct{ config ProviderConfig }

func NewStripeClient(cfg ProviderConfig) *StripeClient { return &StripeClient{config: cfg} }

func (c *StripeClient) GetQuote(ctx context.Context, req *QuoteRequest) (*QuoteResponse, error) {
	if c.config.APIKey == "" {
		return nil, fmt.Errorf("Stripe API key not configured; cannot fetch a real quote")
	}
	// Stripe does not have a fiat->crypto quote endpoint directly; the
	// on-ramp partner (Stripe Crypto Onramp) requires the secret key to
	// create a session. Honest: reject until configured.
	return nil, fmt.Errorf("Stripe on-ramp requires a configured session; use CreateOrder")
}

func (c *StripeClient) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
	if c.config.APIKey == "" {
		return nil, fmt.Errorf("Stripe API key not configured; cannot create a real order")
	}
	// Real Stripe Crypto Onramp Session API: POST /v1/crypto_onramp_sessions
	base := c.config.BaseURL
	if base == "" {
		base = "https://api.stripe.com"
	}
	body := url.Values{}
	body.Set("source_currency", strings.ToLower(req.FiatCurrency))
	body.Set("source_amount", fmt.Sprintf("%.0f", req.FiatAmount))
	body.Set("destination_currency", strings.ToLower(req.CryptoCurrency))
	body.Set("destination_network", req.Chain)
	body.Set("wallet_address", req.WalletAddress)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", base+"/v1/crypto_onramp_sessions", strings.NewReader(body.Encode()))
	httpReq.SetBasicAuth(c.config.APIKey, "")
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Stripe API returned status %d", resp.StatusCode)
	}
	var sess struct {
		ID           string `json:"id"`
		RedirectURL  string `json:"redirect_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sess); err != nil {
		return nil, err
	}
	return &CreateOrderResponse{
		OrderID:         sess.ID,
		PaymentURL:      sess.RedirectURL,
		ProviderOrderID: sess.ID,
		ExpiresAt:       time.Now().Add(30 * time.Minute).Unix(),
	}, nil
}

func (c *StripeClient) GetOrderStatus(ctx context.Context, providerOrderID string) (*Order, error) {
	if c.config.APIKey == "" {
		return nil, fmt.Errorf("Stripe API key not configured")
	}
	base := c.config.BaseURL
	if base == "" {
		base = "https://api.stripe.com"
	}
	httpReq, _ := http.NewRequestWithContext(ctx, "GET", base+"/v1/crypto_onramp_sessions/"+providerOrderID, nil)
	httpReq.SetBasicAuth(c.config.APIKey, "")
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Stripe API returned status %d", resp.StatusCode)
	}
	var sess struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sess); err != nil {
		return nil, err
	}
	return &Order{ProviderOrderID: sess.ID, Status: sess.Status}, nil
}

func (c *StripeClient) HandleWebhook(ctx context.Context, payload []byte) (*Order, error) {
	if c.config.WebhookSecret == "" {
		return nil, fmt.Errorf("Stripe webhook secret not configured; payload cannot be verified")
	}
	return nil, fmt.Errorf("webhook signature verification not implemented; rejecting unverified payload")
}

type WyreClient struct{ config ProviderConfig }

func NewWyreClient(cfg ProviderConfig) *WyreClient { return &WyreClient{config: cfg} }

func (c *WyreClient) GetQuote(ctx context.Context, req *QuoteRequest) (*QuoteResponse, error) {
	if c.config.APIKey == "" {
		return nil, fmt.Errorf("Wyre API key not configured; cannot fetch a real quote")
	}
	base := c.config.BaseURL
	if base == "" {
		base = "https://api.sendwyre.com"
	}
	body, _ := json.Marshal(map[string]interface{}{
		"sourceAmount":    req.FiatAmount,
		"sourceCurrency":  strings.ToLower(req.FiatCurrency),
		"destCurrency":    strings.ToLower(req.CryptoCurrency),
	})
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", base+"/v3/orders/quote/partner", strings.NewReader(string(body)))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Api-Key", c.config.APIKey)
	httpReq.Header.Set("X-Api-Signature", c.config.APISecret)
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Wyre API returned status %d", resp.StatusCode)
	}
	var q struct {
		DestAmount float64 `json:"destAmount"`
		SourceAmount float64 `json:"sourceAmount"`
		Fee        float64 `json:"fee"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&q); err != nil {
		return nil, err
	}
	rate := 0.0
	if q.SourceAmount > 0 {
		rate = q.DestAmount / q.SourceAmount
	}
	return &QuoteResponse{
		CryptoAmount:   q.DestAmount,
		ExchangeRate:   rate,
		FiatEquivalent: req.FiatAmount,
		FeeAmount:      q.Fee,
		ProviderFee:    q.Fee,
		ValidUntil:     time.Now().Add(5 * time.Minute).Unix(),
	}, nil
}

func (c *WyreClient) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
	if c.config.APIKey == "" {
		return nil, fmt.Errorf("Wyre API key not configured; cannot create a real order")
	}
	base := c.config.BaseURL
	if base == "" {
		base = "https://pay.sendwyre.com"
	}
	params := url.Values{}
	params.Set("dest", req.WalletAddress)
	params.Set("destCurrency", strings.ToLower(req.CryptoCurrency))
	params.Set("sourceAmount", fmt.Sprintf("%.2f", req.FiatAmount))
	params.Set("sourceCurrency", strings.ToLower(req.FiatCurrency))
	params.Set("accountId", c.config.APIKey)
	return &CreateOrderResponse{
		OrderID:    uuid.New().String(),
		PaymentURL: base + "/purchase?" + params.Encode(),
		ExpiresAt: time.Now().Add(30 * time.Minute).Unix(),
	}, nil
}

func (c *WyreClient) GetOrderStatus(ctx context.Context, providerOrderID string) (*Order, error) {
	if c.config.APIKey == "" {
		return nil, fmt.Errorf("Wyre API key not configured")
	}
	base := c.config.BaseURL
	if base == "" {
		base = "https://api.sendwyre.com"
	}
	httpReq, _ := http.NewRequestWithContext(ctx, "GET", base+"/v3/orders/"+providerOrderID, nil)
	httpReq.Header.Set("X-Api-Key", c.config.APIKey)
	httpReq.Header.Set("X-Api-Signature", c.config.APISecret)
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Wyre API returned status %d", resp.StatusCode)
	}
	var o struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&o); err != nil {
		return nil, err
	}
	return &Order{ProviderOrderID: o.ID, Status: o.Status}, nil
}

func (c *WyreClient) HandleWebhook(ctx context.Context, payload []byte) (*Order, error) {
	if c.config.WebhookSecret == "" {
		return nil, fmt.Errorf("Wyre webhook secret not configured; payload cannot be verified")
	}
	return nil, fmt.Errorf("webhook signature verification not implemented; rejecting unverified payload")
}

// ============================================================================
// Main
// ============================================================================

func main() {
	log.Println("TigerWallet Fiat On-Ramp Service")
	log.Println("================================")

	service := NewFiatService(nil)
	service.initProviders()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "fiat-onramp", "timestamp": time.Now().Unix()})
	})

	service.SetupRoutes(r)

	addr := fmt.Sprintf(":%d", defaultConfig.Port)
	log.Printf("Fiat On-Ramp service starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
