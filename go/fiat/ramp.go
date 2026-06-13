package fiat

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sort"
	"sync"
	"time"
)

// ============================================================================
// Fiat On-Ramp/Off-Ramp Service
// ============================================================================

// FiatService provides fiat on/off ramp functionality
type FiatService struct {
	mu           sync.RWMutex
	providers   map[string]Provider
	tokens      map[string]TokenInfo
	networks    map[uint64]NetworkInfo
	config      *Config
	httpClient *http.Client
}

// Provider represents a fiat provider
type Provider interface {
	Name() string
	GetName() string
	GetRamps(ctx context.Context, req RampRequest) ([]RampQuote, error)
	CreateRamp(ctx context.Context, req CreateRampRequest) (*RampOrder, error)
	GetOrder(ctx context.Context, orderID string) (*RampOrder, error)
	CancelOrder(ctx context.Context, orderID string) error
}

// Config for fiat service
type Config struct {
	APIKey      string
	APISecret   string
	WebhookURL string
	Timeout    time.Duration
}

// TokenInfo represents token information
type TokenInfo struct {
	Symbol    string
	Name     string
	Decimals uint8
}

// NetworkInfo represents network information
type NetworkInfo struct {
	ChainID     uint64
	Name        string
	Symbol      string
	Decimals    uint8
	MinAmount   *big.Rat
	MaxAmount   *big.Rat
	GasPrice    *big.Int
}

// RampRequest for getting quotes
type RampRequest struct {
	Amount        *big.Rat
	Currency     string
	Token        string
	ChainID      uint64
	Country      string
	PaymentMethod string
}

// RampQuote represents a quote
type RampQuote struct {
	Provider      string
	Amount       *big.Rat
	CryptoAmount *big.Rat
	Rate         *big.Rat
	Fee          *big.Rat
	Total        *big.Rat
	ExpiresAt    time.Time
}

// CreateRampRequest for creating an order
type CreateRampRequest struct {
	QuoteID     string
	Provider   string
	Address    string
	Email      string
	Phone      string
	Country    string
	PaymentMethod string
}

// RampOrder represents a ramp order
type RampOrder struct {
	OrderID      string
	Provider    string
	Status      RampStatus
	CryptoAmount *big.Rat
	FiatAmount  *big.Rat
	Address     string
	ExpiresAt   time.Time
	CreatedAt   time.Time
}

// RampStatus enum
type RampStatus string

const (
	RampStatusPending   RampStatus = "pending"
	RampStatusProcessing RampStatus = "processing"
	RampStatusCompleted RampStatus = "completed"
	RampStatusFailed   RampStatus = "failed"
	RampStatusCancelled RampStatus = "cancelled"
	RampStatusRefunded RampStatus = "refunded"
)

// NewFiatService creates new fiat service
func NewFiatService(cfg *Config) *FiatService {
	return &FiatService{
		providers: make(map[string]Provider),
		tokens:    make(map[string]TokenInfo),
		networks:  make(map[uint64]NetworkInfo),
		config:   cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// RegisterProvider registers a provider
func (f *FiatService) RegisterProvider(name string, provider Provider) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.providers[name] = provider
}

// GetRamps gets quotes from all providers
func (f *FiatService) GetRamps(ctx context.Context, req RampRequest) ([]RampQuote, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	type result struct {
		provider string
		quotes   []RampQuote
		err      error
	}
	
	ch := make(chan result, len(f.providers))
	
	for name, provider := range f.providers {
		go func(name string, provider Provider) {
			quotes, err := provider.GetRamps(ctx, req)
			ch <- result{name, quotes, err}
		}(name, provider)
	}
	
	var allQuotes []result
	timeout := time.After(10 * time.Second)
	
	for i := 0; i < len(f.providers); i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout:
			break
		case r := <-ch:
			if r.err == nil {
				allQuotes = append(allQuotes, r)
			}
		}
	}
	
	var quotes []RampQuote
	for _, r := range allQuotes {
		for _, q := range r.quotes {
			q.Provider = r.provider
			quotes = append(quotes, q)
		}
	}
	
	// Sort by crypto amount (descending)
	sort.Slice(quotes, func(i, j int) bool {
		return quotes[i].CryptoAmount.Cmp(quotes[j].CryptoAmount) > 0
	})
	
	return quotes, nil
}

// CreateOrder creates a ramp order
func (f *FiatService) CreateOrder(ctx context.Context, req CreateRampRequest) (*RampOrder, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	provider, ok := f.providers[req.Provider]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", req.Provider)
	}
	
	return provider.CreateRamp(ctx, req)
}

// GetOrder gets order status
func (f *FiatService) GetOrder(ctx context.Context, providerName, orderID string) (*RampOrder, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	provider, ok := f.providers[providerName]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", providerName)
	}
	
	return provider.GetOrder(ctx, orderID)
}

// CancelOrder cancels an order
func (f *FiatService) CancelOrder(ctx context.Context, providerName, orderID string) error {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	provider, ok := f.providers[providerName]
	if !ok {
		return fmt.Errorf("provider not found: %s", providerName)
	}
	
	return provider.CancelOrder(ctx, orderID)
}

// ============================================================================
// Provider Implementations
// ============================================================================

// MoonPayProvider represents MoonPay
type MoonPayProvider struct {
	apiKey    string
	secretKey string
	baseURL   string
	client   *http.Client
}

// NewMoonPayProvider creates MoonPay provider
func NewMoonPayProvider(apiKey, secretKey string) *MoonPayProvider {
	return &MoonPayProvider{
		apiKey:    apiKey,
		secretKey: secretKey,
		baseURL:   "https://api.moonpay.com",
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (m *MoonPayProvider) Name() string     { return "moonpay" }
func (m *MoonPayProvider) GetName() string  { return "MoonPay" }

func (m *MoonPayProvider) GetRamps(ctx context.Context, req RampRequest) ([]RampQuote, error) {
	// Simulate quote
	rate := big.NewFloat(2500) // Example rate
	
	quotes := []RampQuote{
		{
			Provider:      "MoonPay",
			Amount:       req.Amount,
			CryptoAmount: new(big.Rat).SetFloat64(0),
			Rate:         rate,
			Fee:          new(big.Rat).SetFloat64(0),
			Total:        req.Amount,
			ExpiresAt:    time.Now().Add(10 * time.Minute),
		},
	}
	
	return quotes, nil
}

func (m *MoonPayProvider) CreateRamp(ctx context.Context, req CreateRampRequest) (*RampOrder, error) {
	return &RampOrder{
		OrderID:     generateID("moonpay_"),
		Provider:   "MoonPay",
		Status:     RampStatusPending,
		ExpiresAt:  time.Now().Add(15 * time.Minute),
		CreatedAt:  time.Now(),
	}, nil
}

func (m *MoonPayProvider) GetOrder(ctx context.Context, orderID string) (*RampOrder, error) {
	return &RampOrder{
		OrderID:    orderID,
		Provider:  "MoonPay",
		Status:    RampStatusPending,
		CreatedAt: time.Now(),
	}, nil
}

func (m *MoonPayProvider) CancelOrder(ctx context.Context, orderID string) error {
	return nil
}

// RampProvider represents Ramp
type RampProvider struct {
	apiKey string
	baseURL string
	client *http.Client
}

// NewRampProvider creates Ramp provider
func NewRampProvider(apiKey string) *RampProvider {
	return &RampProvider{
		apiKey:   apiKey,
		baseURL:  "https://api.ramp.network",
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (r *RampProvider) Name() string    { return "ramp" }
func (r *RampProvider) GetName() string { return "Ramp" }

func (r *RampProvider) GetRamps(ctx context.Context, req RampRequest) ([]RampQuote, error) {
	quotes := []RampQuote{
		{
			Provider:   "Ramp",
			Amount:    req.Amount,
			CryptoAmount: new(big.Rat).SetFloat64(0),
			Rate:       big.NewFloat(2499),
			Fee:        new(big.Rat).SetFloat64(0),
			Total:      req.Amount,
			ExpiresAt: time.Now().Add(10 * time.Minute),
		},
	}
	return quotes, nil
}

func (r *RampProvider) CreateRamp(ctx context.Context, req CreateRampRequest) (*RampOrder, error) {
	return &RampOrder{
		OrderID:    generateID("ramp_"),
		Provider:  "Ramp",
		Status:    RampStatusPending,
		ExpiresAt: time.Now().Add(15 * time.Minute),
		CreatedAt: time.Now(),
	}, nil
}

func (r *RampProvider) GetOrder(ctx context.Context, orderID string) (*RampOrder, error) {
	return &RampOrder{
		OrderID:   orderID,
		Provider: "Ramp",
		Status:   RampStatusPending,
		CreatedAt: time.Now(),
	}, nil
}

func (r *RampProvider) CancelOrder(ctx context.Context, orderID string) error {
	return nil
}

// TransakProvider represents Transak
type TransakProvider struct {
	apiKey string
	baseURL string
	client *http.Client
}

// NewTransakProvider creates Transak provider
func NewTransakProvider(apiKey string) *TransakProvider {
	return &TransakProvider{
		apiKey:   apiKey,
		baseURL:  "https://api.transak.com",
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (t *TransakProvider) Name() string    { return "transak" }
func (t *TransakProvider) GetName() string { return "Transak" }

func (t *TransakProvider) GetRamps(ctx context.Context, req RampRequest) ([]RampQuote, error) {
	quotes := []RampQuote{
		{
			Provider:   "Transak",
			Amount:    req.Amount,
			CryptoAmount: new(big.Rat).SetFloat64(0),
			Rate:       big.NewFloat(2501),
			Fee:        new(big.Rat).SetFloat64(0),
			Total:      req.Amount,
			ExpiresAt: time.Now().Add(10 * time.Minute),
		},
	}
	return quotes, nil
}

func (t *TransakProvider) CreateRamp(ctx context.Context, req CreateRampRequest) (*RampOrder, error) {
	return &RampOrder{
		OrderID:    generateID("transak_"),
		Provider:  "Transak",
		Status:    RampStatusPending,
		ExpiresAt: time.Now().Add(15 * time.Minute),
		CreatedAt: time.Now(),
	}, nil
}

func (t *TransakProvider) GetOrder(ctx context.Context, orderID string) (*RampOrder, error) {
	return &RampOrder{
		OrderID:   orderID,
		Provider: "Transak",
		Status:   RampStatusPending,
		CreatedAt: time.Now(),
	}, nil
}

func (t *TransakProvider) CancelOrder(ctx context.Context, orderID string) error {
	return nil
}

// ============================================================================
// Utilities
// ============================================================================

func generateID(prefix string) string {
	return fmt.Sprintf("%s%x", prefix, time.Now().UnixNano())
}

// ============================================================================
// HTTP Handlers
// ============================================================================

// HandleGetRamps handles get quotes
func (f *FiatService) HandleGetRamps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	amountStr := r.URL.Query().Get("amount")
	currency := r.URL.Query().Get("currency")
	token := r.URL.Query().Get("token")
	
	amount := new(big.Rat)
	amount.SetString(amountStr)
	
	req := RampRequest{
		Amount:    amount,
		Currency: currency,
		Token:    token,
	}
	
	quotes, err := f.GetRamps(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(quotes)
}

// HandleCreateOrder handles create order
func (f *FiatService) HandleCreateOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req CreateRampRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	order, err := f.CreateOrder(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

// HandleGetOrder handles get order
func (f *FiatService) HandleGetOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	orderID := r.URL.Query().Get("orderId")
	provider := r.URL.Query().Get("provider")
	
	order, err := f.GetOrder(r.Context(), provider, orderID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

// ============================================================================
// Webhook Handling
// ============================================================================

// WebhookHandler handles webhooks from providers
type WebhookHandler struct {
	service *FiatService
	secret string
}

// NewWebhookHandler creates webhook handler
func NewWebhookHandler(service *FiatService, secret string) *WebhookHandler {
	return &WebhookHandler{
		service: service,
		secret:  secret,
	}
}

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Verify signature
	signature := r.Header.Get("X-Signature")
	if !h.verifySignature(r.Body, signature) {
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}
	
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	orderID, ok := payload["orderId"].(string)
	if !ok {
		http.Error(w, "Missing orderId", http.StatusBadRequest)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"received": true}`)
}

func (h *WebhookHandler) verifySignature(body io.Reader, signature string) bool {
	if signature == "" {
		return false
	}
	
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return false
	}
	
	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write(bodyBytes)
	expected := hex.EncodeToString(mac.Sum(nil))
	
	return hmac.Equal([]byte(signature), []byte(expected))
}

// ============================================================================
// Fiat Service HTTP Server
// ============================================================================

// Serve starts the fiat service HTTP server
func (f *FiatService) Serve(addr string) error {
	http.HandleFunc("/v1/ramps", f.HandleGetRamps)
	http.HandleFunc("/v1/orders", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			f.HandleCreateOrder(w, r)
		case http.MethodGet:
			f.HandleGetOrder(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	
	return http.ListenAndServe(addr, nil)
}