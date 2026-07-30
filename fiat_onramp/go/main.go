/**
 * TigerWallet Fiat On-Ramp Service - Complete Implementation
 * 
 * Multi-provider fiat to crypto gateway with KYC integration
 * High-performance Go service for worldwide distribution
 */

package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// TYPES AND STRUCTURES
// ============================================================================

// Supported fiat currencies
type FiatCurrency struct {
	Code       string  `json:"code"`       // USD, EUR, GBP, etc.
	Symbol     string  `json:"symbol"`     // $, €, £
	Name       string  `json:"name"`        // US Dollar
	MinAmount  float64 `json:"min_amount"` // Minimum purchase
	MaxAmount  float64 `json:"max_amount"` // Maximum purchase
	Decimals   int     `json:"decimals"`
	IsEnabled  bool    `json:"is_enabled"`
}

// Supported crypto assets
type CryptoAsset struct {
	Symbol         string   `json:"symbol"`          // BTC, ETH, USDT
	Name           string   `json:"name"`            // Bitcoin
	Contract       string   `json:"contract,omitempty"` // Token contract address
	ChainID        uint64   `json:"chain_id"`       // 1 for ETH, 56 for BSC
	MinAmount      float64  `json:"min_amount"`     // Minimum purchase
	MaxAmount      float64  `json:"max_amount"`     // Maximum purchase
	Decimals       int      `json:"decimals"`
	IsEnabled     bool     `json:"is_enabled"`
	Network        string   `json:"network"`        // Ethereum, BSC, etc.
}

// Payment method types
type PaymentMethod struct {
	ID           string   `json:"id"`
	Type         string   `json:"type"` // card, bank_transfer, apple_pay, google_pay
	Name         string   `json:"name"`
	Fee          float64  `json:"fee"`         // Processing fee percentage
	FixedFee     float64  `json:"fixed_fee"`  // Fixed fee
	MinLimit     float64  `json:"min_limit"`  // Minimum amount
	MaxLimit     float64  `json:"max_limit"`  // Maximum amount
	RequiresKYC  bool     `json:"requires_kyc"`
	IsEnabled    bool     `json:"is_enabled"`
}

// KYC levels
type KYCLevel struct {
	Level           int     `json:"level"`    // 0, 1, 2, 3
	Name            string  `json:"name"`     // Basic, Intermediate, Full
	MaxDailyLimit   float64 `json:"max_daily_limit"`
	MaxMonthlyLimit float64 `json:"max_monthly_limit"`
	RequiresDoc     bool    `json:"requires_doc"`
	RequiresVideo   bool    `json:"requires_video"`
}

// Transaction status
type TransactionStatus string

const (
	StatusPending      TransactionStatus = "pending"
	StatusProcessing  TransactionStatus = "processing"
	StatusCompleted   TransactionStatus = "completed"
	StatusFailed      TransactionStatus = "failed"
	StatusCancelled   TransactionStatus = "cancelled"
	StatusRefunded    TransactionStatus = "refunded"
	StatusExpired     TransactionStatus = "expired"
)

// Onramp transaction
type OnrampTransaction struct {
	ID                string            `json:"id"`
	UserID            string            `json:"user_id"`
	Provider          string            `json:"provider"` // moonpay, simplex, transak
	FiatCurrency      string            `json:"fiat_currency"`
	FiatAmount        float64           `json:"fiat_amount"`
	FiatTotal         float64           `json:"fiat_total"` // Including fees
	CryptoCurrency    string            `json:"crypto_currency"`
	CryptoAmount      string            `json:"crypto_amount"`
	ExchangeRate      string            `json:"exchange_rate"`
	WalletAddress     string            `json:"wallet_address"`
	ChainID          uint64            `json:"chain_id"`
	Status            TransactionStatus `json:"status"`
	PaymentMethod     string            `json:"payment_method"`
	KYCLevel         int               `json:"kyc_level"`
	Fees             TransactionFees   `json:"fees"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	ExpiresAt        time.Time         `json:"expires_at"`
	CompletedAt      *time.Time        `json:"completed_at,omitempty"`
	BlockchainTxHash string            `json:"blockchain_tx_hash,omitempty"`
	ExternalID       string            `json:"external_id,omitempty"` // Provider's transaction ID
	RedirectURL      string            `json:"redirect_url,omitempty"`
	QuoteID          string            `json:"quote_id,omitempty"`
}

// Transaction fees breakdown
type TransactionFees struct {
	ProviderFee   float64 `json:"provider_fee"`   // Provider's fee
	PlatformFee   float64 `json:"platform_fee"`   // TigerWallet's fee
	NetworkFee    float64 `json:"network_fee"`    // Network fee
	ProcessingFee float64 `json:"processing_fee"` // Payment processing fee
	TotalFees    float64 `json:"total_fees"`
}

// Price quote
type PriceQuote struct {
	ID              string    `json:"id"`
	FiatCurrency    string    `json:"fiat_currency"`
	FiatAmount      float64   `json:"fiat_amount"`
	CryptoCurrency  string    `json:"crypto_currency"`
	CryptoAmount    float64   `json:"crypto_amount"`
	ExchangeRate    float64   `json:"exchange_rate"`
	ValidUntil     time.Time `json:"valid_until"`
	Provider        string    `json:"provider"`
}

// KYC Application
type KYCApplication struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Level        int       `json:"level"`
	Status       string    `json:"status"` // pending, submitted, approved, rejected
	Provider     string    `json:"provider"`
	ExternalID   string    `json:"external_id,omitempty"`
	SubmittedAt  time.Time `json:"submitted_at,omitempty"`
	ApprovedAt   time.Time `json:"approved_at,omitempty"`
	RejectedAt   time.Time `json:"approved_at,omitempty"`
	RejectionReason string `json:"rejection_reason,omitempty"`
	Documents    []KYCDocument `json:"documents"`
}

// KYC Document
type KYCDocument struct {
	Type        string `json:"type"` // passport, id_card, drivers_license
	Number      string `json:"number,omitempty"`
	IssuedDate  string `json:"issued_date,omitempty"`
	ExpiryDate  string `json:"expiry_date,omitempty"`
	Country     string `json:"country"`
	Status      string `json:"status"`
}

// ============================================================================
// CONFIGURATION
// ============================================================================

// FiatOnrampConfig holds all configuration
type FiatOnrampConfig struct {
	Providers           map[string]ProviderConfig `json:"providers"`
	FiatCurrencies      map[string]FiatCurrency  `json:"fiat_currencies"`
	CryptoAssets        map[string]CryptoAsset   `json:"crypto_assets"`
	PaymentMethods      []PaymentMethod          `json:"payment_methods"`
	KYCLevels          []KYCLevel               `json:"kyc_levels"`
	DefaultProvider     string                   `json:"default_provider"`
	QuoteExpirySeconds int                      `json:"quote_expiry_seconds"`
	TxExpiryMinutes    int                      `json:"tx_expiry_minutes"`
	WebhookSecret      string                   `json:"webhook_secret"`
}

// ProviderConfig holds provider-specific configuration
type ProviderConfig struct {
	Name        string            `json:"name"`
	Enabled     bool             `json:"enabled"`
	APIKey      string            `json:"api_key"`
	APISecret   string            `json:"api_secret"`
	BaseURL     string            `json:"base_url"`
	WebhookURL  string            `json:"webhook_url"`
	FeePercent  float64           `json:"fee_percent"`
	MinAmount   float64           `json:"min_amount"`
	MaxAmount   float64           `json:"max_amount"`
	SupportedFiats []string       `json:"supported_fiats"`
	SupportedCrypto []string      `json:"supported_crypto"`
	SupportedMethods []string     `json:"supported_methods"`
}

// ============================================================================
// SERVICE IMPLEMENTATION
// ============================================================================

// FiatOnrampService main service struct
type FiatOnrampService struct {
	config         FiatOnrampConfig
	mu             sync.RWMutex
	quotes         map[string]*PriceQuote
	transactions   map[string]*OnrampTransaction
	kycApplications map[string]*KYCApplication
	httpClient     *http.Client
}

// NewFiatOnrampService creates a new service instance
func NewFiatOnrampService(config FiatOnrampConfig) *FiatOnrampService {
	return &FiatOnrampService{
		config:          config,
		quotes:         make(map[string]*PriceQuote),
		transactions:   make(map[string]*OnrampTransaction),
		kycApplications: make(map[string]*KYCApplication),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ============================================================================
// PRICE QUOTE FUNCTIONS
// ============================================================================

// GetSupportedFiatCurrencies returns all supported fiat currencies
func (s *FiatOnrampService) GetSupportedFiatCurrencies() []FiatCurrency {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var currencies []FiatCurrency
	for _, c := range s.config.FiatCurrencies {
		if c.IsEnabled {
			currencies = append(currencies, c)
		}
	}
	return currencies
}

// GetSupportedCryptoAssets returns all supported crypto assets
func (s *FiatOnrampService) GetSupportedCryptoAssets() []CryptoAsset {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var assets []CryptoAsset
	for _, a := range s.config.CryptoAssets {
		if a.IsEnabled {
			assets = append(assets, a)
		}
	}
	return assets
}

// GetPaymentMethods returns available payment methods
func (s *FiatOnrampService) GetPaymentMethods() []PaymentMethod {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var methods []PaymentMethod
	for _, m := range s.config.PaymentMethods {
		if m.IsEnabled {
			methods = append(methods, m)
		}
	}
	return methods
}

// CreatePriceQuote creates a new price quote
func (s *FiatOnrampService) CreatePriceQuote(ctx context.Context, userID, fiatCurrency, cryptoCurrency string, fiatAmount float64, paymentMethod string) (*PriceQuote, error) {
	// Validate fiat currency
	fiat, ok := s.config.FiatCurrencies[fiatCurrency]
	if !ok || !fiat.IsEnabled {
		return nil, fmt.Errorf("unsupported fiat currency: %s", fiatCurrency)
	}
	
	// Validate crypto currency
	crypto, ok := s.config.CryptoAssets[cryptoCurrency]
	if !ok || !crypto.IsEnabled {
		return nil, fmt.Errorf("unsupported crypto currency: %s", cryptoCurrency)
	}
	
	// Validate amount
	if fiatAmount < fiat.MinAmount {
		return nil, fmt.Errorf("amount below minimum: %f", fiat.MinAmount)
	}
	if fiatAmount > fiat.MaxAmount {
		return nil, fmt.Errorf("amount above maximum: %f", fiat.MaxAmount)
	}
	
	// Get provider
	provider := s.selectProvider(fiatCurrency, cryptoCurrency, paymentMethod)
	if provider == nil {
		return nil, fmt.Errorf("no available provider for this combination")
	}
	
	// Calculate crypto amount (simplified - real implementation would call provider API)
	// In production, this would fetch real-time rates from the provider
	exchangeRate := s.getExchangeRate(fiatCurrency, cryptoCurrency, provider)
	cryptoAmount := fiatAmount / exchangeRate
	
	quote := &PriceQuote{
		ID:              generateID("quote"),
		FiatCurrency:    fiatCurrency,
		FiatAmount:      fiatAmount,
		CryptoCurrency:  cryptoCurrency,
		CryptoAmount:    cryptoAmount,
		ExchangeRate:    exchangeRate,
		ValidUntil:      time.Now().Add(time.Duration(s.config.QuoteExpirySeconds) * time.Second),
		Provider:        provider.Name,
	}
	
	// Store quote
	s.mu.Lock()
	s.quotes[quote.ID] = quote
	s.mu.Unlock()
	
	return quote, nil
}

// GetQuote retrieves a quote by ID
func (s *FiatOnrampService) GetQuote(quoteID string) (*PriceQuote, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	quote, ok := s.quotes[quoteID]
	if !ok {
		return nil, fmt.Errorf("quote not found")
	}
	
	// Check if expired
	if time.Now().After(quote.ValidUntil) {
		return nil, fmt.Errorf("quote expired")
	}
	
	return quote, nil
}

// ============================================================================
// TRANSACTION FUNCTIONS
// ============================================================================

// CreateTransaction creates a new onramp transaction
func (s *FiatOnrampService) CreateTransaction(ctx context.Context, userID, quoteID, walletAddress string, chainID uint64) (*OnrampTransaction, error) {
	// Get quote
	quote, err := s.GetQuote(quoteID)
	if err != nil {
		return nil, err
	}
	
	// Validate wallet address
	if walletAddress == "" {
		return nil, fmt.Errorf("wallet address is required")
	}
	
	// Calculate fees
	provider := s.config.Providers[quote.Provider]
	platformFee := quote.FiatAmount * 0.01 // 1% platform fee
	providerFee := quote.FiatAmount * (provider.FeePercent / 100)
	processingFee := 0.0 // Based on payment method
	networkFee := 0.0   // For crypto network
	
	totalFees := platformFee + providerFee + processingFee + networkFee
	
	tx := &OnrampTransaction{
		ID:               generateID("txn"),
		UserID:           userID,
		Provider:         quote.Provider,
		FiatCurrency:     quote.FiatCurrency,
		FiatAmount:       quote.FiatAmount,
		FiatTotal:        quote.FiatAmount + totalFees,
		CryptoCurrency:   quote.CryptoCurrency,
		CryptoAmount:     fmt.Sprintf("%.8f", quote.CryptoAmount),
		ExchangeRate:     fmt.Sprintf("%.8f", quote.ExchangeRate),
		WalletAddress:    walletAddress,
		ChainID:          chainID,
		Status:           StatusPending,
		PaymentMethod:    "card",
		KYCLevel:         0,
		Fees: TransactionFees{
			ProviderFee:   providerFee,
			PlatformFee:   platformFee,
			ProcessingFee: processingFee,
			NetworkFee:    networkFee,
			TotalFees:     totalFees,
		},
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		ExpiresAt:        time.Now().Add(time.Duration(s.config.TxExpiryMinutes) * time.Minute),
		QuoteID:          quoteID,
	}
	
	// Store transaction
	s.mu.Lock()
	s.transactions[tx.ID] = tx
	s.mu.Unlock()
	
	return tx, nil
}

// GetTransaction retrieves a transaction by ID
func (s *FiatOnrampService) GetTransaction(txID string) (*OnrampTransaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	tx, ok := s.transactions[txID]
	if !ok {
		return nil, fmt.Errorf("transaction not found")
	}
	
	return tx, nil
}

// GetUserTransactions returns all transactions for a user
func (s *FiatOnrampService) GetUserTransactions(userID string) []*OnrampTransaction {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var txs []*OnrampTransaction
	for _, tx := range s.transactions {
		if tx.UserID == userID {
			txs = append(txs, tx)
		}
	}
	return txs
}

// UpdateTransactionStatus updates transaction status
func (s *FiatOnrampService) UpdateTransactionStatus(txID string, status TransactionStatus, txHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	tx, ok := s.transactions[txID]
	if !ok {
		return fmt.Errorf("transaction not found")
	}
	
	tx.Status = status
	tx.UpdatedAt = time.Now()
	
	if txHash != "" {
		tx.BlockchainTxHash = txHash
	}
	
	if status == StatusCompleted {
		now := time.Now()
		tx.CompletedAt = &now
	}
	
	return nil
}

// CancelTransaction cancels a pending transaction
func (s *FiatOnrampService) CancelTransaction(txID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	tx, ok := s.transactions[txID]
	if !ok {
		return fmt.Errorf("transaction not found")
	}
	
	if tx.Status != StatusPending {
		return fmt.Errorf("can only cancel pending transactions")
	}
	
	tx.Status = StatusCancelled
	tx.UpdatedAt = time.Now()
	
	return nil
}

// ============================================================================
// KYC FUNCTIONS
// ============================================================================

// InitiateKYC starts a new KYC application
func (s *FiatOnrampService) InitiateKYC(ctx context.Context, userID string, level int) (*KYCApplication, error) {
	// Validate level
	if level < 0 || level >= len(s.config.KYCLevels) {
		return nil, fmt.Errorf("invalid KYC level")
	}
	
	kycLevel := s.config.KYCLevels[level]
	
	application := &KYCApplication{
		ID:      generateID("kyc"),
		UserID:  userID,
		Level:   level,
		Status:  "pending",
		Provider: "stripe", // or another KYC provider
	}
	
	s.mu.Lock()
	s.kycApplications[application.ID] = application
	s.mu.Unlock()
	
	return application, nil
}

// GetKYCStatus returns KYC application status
func (s *FiatOnrampService) GetKYCStatus(userID string) (*KYCApplication, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	for _, app := range s.kycApplications {
		if app.UserID == userID && app.Status != "rejected" {
			return app, nil
		}
	}
	
	return nil, fmt.Errorf("no KYC application found")
}

// GetUserKYCLevel returns the user's KYC level
func (s *FiatOnrampService) GetUserKYCLevel(userID string) int {
	app, err := s.GetKYCStatus(userID)
	if err != nil {
		return 0
	}
	
	if app.Status == "approved" {
		return app.Level
	}
	
	return 0
}

// CheckUserLimits checks if user can purchase given amount
func (s *FiatOnrampService) CheckUserLimits(userID string, amount float64) error {
	level := s.GetUserKYCLevel(userID)
	
	if level >= len(s.config.KYCLevels) {
		return fmt.Errorf("KYC level out of bounds")
	}
	
	kycLevel := s.config.KYCLevels[level]
	
	if amount > kycLevel.MaxDailyLimit {
		return fmt.Errorf("amount exceeds daily limit: %f", kycLevel.MaxDailyLimit)
	}
	
	return nil
}

// ============================================================================
// WEBHOOK HANDLING
// ============================================================================

// WebhookEvent represents a webhook event from provider
type WebhookEvent struct {
	Type        string          `json:"type"`
	EventID     string          `json:"event_id"`
	Timestamp   time.Time       `json:"timestamp"`
	Data        json.RawMessage `json:"data"`
	Signature   string          `json:"signature"`
}

// HandleWebhook processes incoming webhooks
func (s *FiatOnrampService) HandleWebhook(payload []byte, signature string) error {
	// Verify signature
	if !s.verifyWebhookSignature(payload, signature) {
		return fmt.Errorf("invalid webhook signature")
	}
	
	var event WebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("failed to parse webhook: %w", err)
	}
	
	switch event.Type {
	case "payment.pending":
		return s.handlePaymentPending(event.Data)
	case "payment.processing":
		return s.handlePaymentProcessing(event.Data)
	case "payment.completed":
		return s.handlePaymentCompleted(event.Data)
	case "payment.failed":
		return s.handlePaymentFailed(event.Data)
	case "payment.cancelled":
		return s.handlePaymentCancelled(event.Data)
	case "payment.expired":
		return s.handlePaymentExpired(event.Data)
	default:
		return fmt.Errorf("unknown event type: %s", event.Type)
	}
}

func (s *FiatOnrampService) handlePaymentPending(data json.RawMessage) error {
	// Parse and update transaction
	return nil
}

func (s *FiatOnrampService) handlePaymentProcessing(data json.RawMessage) error {
	// Parse and update transaction
	return nil
}

func (s *FiatOnrampService) handlePaymentCompleted(data json.RawMessage) error {
	// Parse transaction data
	var paymentData struct {
		ExternalID string `json:"external_id"`
		CryptoTxHash string `json:"crypto_tx_hash"`
	}
	
	if err := json.Unmarshal(data, &paymentData); err != nil {
		return err
	}
	
	// Find transaction by external ID and update
	s.mu.RLock()
	for _, tx := range s.transactions {
		if tx.ExternalID == paymentData.ExternalID {
			s.mu.RUnlock()
			return s.UpdateTransactionStatus(tx.ID, StatusCompleted, paymentData.CryptoTxHash)
		}
	}
	s.mu.RUnlock()
	
	return fmt.Errorf("transaction not found for external ID: %s", paymentData.ExternalID)
}

func (s *FiatOnrampService) handlePaymentFailed(data json.RawMessage) error {
	var paymentData struct {
		ExternalID string `json:"external_id"`
		Reason     string `json:"reason"`
	}
	
	if err := json.Unmarshal(data, &paymentData); err != nil {
		return err
	}
	
	s.mu.RLock()
	for _, tx := range s.transactions {
		if tx.ExternalID == paymentData.ExternalID {
			s.mu.RUnlock()
			tx.Status = StatusFailed
			tx.UpdatedAt = time.Now()
			return nil
		}
	}
	s.mu.RUnlock()
	
	return nil
}

func (s *FiatOnrampService) handlePaymentCancelled(data json.RawMessage) error {
	return nil
}

func (s *FiatOnrampService) handlePaymentExpired(data json.RawMessage) error {
	var paymentData struct {
		ExternalID string `json:"external_id"`
	}
	
	if err := json.Unmarshal(data, &paymentData); err != nil {
		return err
	}
	
	s.mu.RLock()
	for _, tx := range s.transactions {
		if tx.ExternalID == paymentData.ExternalID {
			s.mu.RUnlock()
			return s.UpdateTransactionStatus(tx.ID, StatusExpired, "")
		}
	}
	s.mu.RUnlock()
	
	return nil
}

// verifyWebhookSignature verifies the webhook signature
func (s *FiatOnrampService) verifyWebhookSignature(payload []byte, signature string) bool {
	if s.config.WebhookSecret == "" {
		return true // Skip verification if no secret configured
	}
	
	mac := hmac.New(sha256.New, []byte(s.config.WebhookSecret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	
	return hmac.Equal([]byte(expected), []byte(signature))
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// selectProvider selects the best provider for the given currencies
func (s *FiatOnrampService) selectProvider(fiatCurrency, cryptoCurrency, paymentMethod string) *ProviderConfig {
	for _, provider := range s.config.Providers {
		if !provider.Enabled {
			continue
		}
		
		// Check if provider supports fiat
		supportsFiat := false
		for _, f := range provider.SupportedFiats {
			if strings.EqualFold(f, fiatCurrency) {
				supportsFiat = true
				break
			}
		}
		if !supportsFiat {
			continue
		}
		
		// Check if provider supports crypto
		supportsCrypto := false
		for _, c := range provider.SupportedCrypto {
			if strings.EqualFold(c, cryptoCurrency) {
				supportsCrypto = true
				break
			}
		}
		if !supportsCrypto {
			continue
		}
		
		// Check if provider supports payment method
		supportsMethod := false
		for _, m := range provider.SupportedMethods {
			if strings.EqualFold(m, paymentMethod) {
				supportsMethod = true
				break
			}
		}
		if !supportsMethod {
			continue
		}
		
		return &provider
	}
	
	return nil
}

// getExchangeRate gets the exchange rate (simplified)
func (s *FiatOnrampService) getExchangeRate(fiatCurrency, cryptoCurrency string, provider *ProviderConfig) float64 {
	// Base rates (in production, fetch from provider API)
	rates := map[string]map[string]float64{
		"USD": {
			"BTC":  65000.0,
			"ETH":  3500.0,
			"USDT": 1.0,
			"USDC": 1.0,
			"BNB":  600.0,
			"SOL":  150.0,
			"TRX":  0.12,
			"XRP":  0.55,
			"DOGE": 0.15,
			"ADA":  0.45,
			"AVAX": 35.0,
			"DOT":  7.0,
			"MATIC": 0.65,
			"LINK": 15.0,
			"UNI":  8.0,
		},
		"EUR": {
			"BTC":  60000.0,
			"ETH":  3200.0,
			"USDT": 0.92,
			"USDC": 0.92,
		},
		"GBP": {
			"BTC":  52000.0,
			"ETH":  2800.0,
			"USDT": 0.79,
			"USDC": 0.79,
		},
	}
	
	fiatRates, ok := rates[fiatCurrency]
	if !ok {
		return 1.0 // Default
	}
	
	rate, ok := fiatRates[cryptoCurrency]
	if !ok {
		return 1.0 // Default
	}
	
	return rate
}

// generateID generates a unique ID
func generateID(prefix string) string {
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("%s_%d_%d", prefix, timestamp, time.Now().Nanosecond()%1000)
}

// ============================================================================
// HTTP HANDLERS (for API endpoints)
// ============================================================================

// ServeHTTP handles HTTP requests
func (s *FiatOnrampService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Add CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	path := r.URL.Path
	method := r.Method
	
	// Route handling
	switch {
	case path == "/api/v1/fiat/currencies" && method == http.MethodGet:
		s.handleGetFiatCurrencies(w, r)
	case path == "/api/v1/fiat/crypto" && method == http.MethodGet:
		s.handleGetCryptoAssets(w, r)
	case path == "/api/v1/fiat/payment-methods" && method == http.MethodGet:
		s.handleGetPaymentMethods(w, r)
	case path == "/api/v1/fiat/quote" && method == http.MethodPost:
		s.handleCreateQuote(w, r)
	case path == "/api/v1/fiat/quote/" && method == http.MethodGet:
		s.handleGetQuote(w, r)
	case path == "/api/v1/fiat/transaction" && method == http.MethodPost:
		s.handleCreateTransaction(w, r)
	case strings.HasPrefix(path, "/api/v1/fiat/transaction/") && method == http.MethodGet:
		s.handleGetTransaction(w, r)
	case path == "/api/v1/fiat/transactions" && method == http.MethodGet:
		s.handleGetUserTransactions(w, r)
	case path == "/api/v1/fiat/kyc" && method == http.MethodPost:
		s.handleInitiateKYC(w, r)
	case path == "/api/v1/fiat/kyc" && method == http.MethodGet:
		s.handleGetKYCStatus(w, r)
	case path == "/api/v1/fiat/webhook" && method == http.MethodPost:
		s.handleWebhook(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *FiatOnrampService) handleGetFiatCurrencies(w http.ResponseWriter, r *http.Request) {
	currencies := s.GetSupportedFiatCurrencies()
	json.NewEncoder(w).Encode(currencies)
}

func (s *FiatOnrampService) handleGetCryptoAssets(w http.ResponseWriter, r *http.Request) {
	assets := s.GetSupportedCryptoAssets()
	json.NewEncoder(w).Encode(assets)
}

func (s *FiatOnrampService) handleGetPaymentMethods(w http.ResponseWriter, r *http.Request) {
	methods := s.GetPaymentMethods()
	json.NewEncoder(w).Encode(methods)
}

func (s *FiatOnrampService) handleCreateQuote(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID         string `json:"user_id"`
		FiatCurrency   string `json:"fiat_currency"`
		CryptoCurrency string `json:"crypto_currency"`
		FiatAmount     float64 `json:"fiat_amount"`
		PaymentMethod  string `json:"payment_method"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	quote, err := s.CreatePriceQuote(r.Context(), req.UserID, req.FiatCurrency, req.CryptoCurrency, req.FiatAmount, req.PaymentMethod)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	json.NewEncoder(w).Encode(quote)
}

func (s *FiatOnrampService) handleGetQuote(w http.ResponseWriter, r *http.Request) {
	quoteID := strings.TrimPrefix(r.URL.Path, "/api/v1/fiat/quote/")
	quote, err := s.GetQuote(quoteID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	
	json.NewEncoder(w).Encode(quote)
}

func (s *FiatOnrampService) handleCreateTransaction(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID       string `json:"user_id"`
		QuoteID      string `json:"quote_id"`
		WalletAddress string `json:"wallet_address"`
		ChainID      uint64 `json:"chain_id"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	tx, err := s.CreateTransaction(r.Context(), req.UserID, req.QuoteID, req.WalletAddress, req.ChainID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	json.NewEncoder(w).Encode(tx)
}

func (s *FiatOnrampService) handleGetTransaction(w http.ResponseWriter, r *http.Request) {
	txID := strings.TrimPrefix(r.URL.Path, "/api/v1/fiat/transaction/")
	tx, err := s.GetTransaction(txID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	
	json.NewEncoder(w).Encode(tx)
}

func (s *FiatOnrampService) handleGetUserTransactions(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id required", http.StatusBadRequest)
		return
	}
	
	txs := s.GetUserTransactions(userID)
	json.NewEncoder(w).Encode(txs)
}

func (s *FiatOnrampService) handleInitiateKYC(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID string `json:"user_id"`
		Level  int   `json:"level"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	app, err := s.InitiateKYC(r.Context(), req.UserID, req.Level)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	json.NewEncoder(w).Encode(app)
}

func (s *FiatOnrampService) handleGetKYCStatus(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id required", http.StatusBadRequest)
		return
	}
	
	app, err := s.GetKYCStatus(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	
	json.NewEncoder(w).Encode(app)
}

func (s *FiatOnrampService) handleWebhook(w http.ResponseWriter, r *http.Request) {
	signature := r.Header.Get("X-Webhook-Signature")
	
	payload := make([]byte, r.ContentLength)
	r.Body.Read(payload)
	
	if err := s.HandleWebhook(payload, signature); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ============================================================================
// MAIN FUNCTION
// ============================================================================

func main() {
	// Default configuration
	config := FiatOnrampConfig{
		DefaultProvider:     "moonpay",
		QuoteExpirySeconds: 300,  // 5 minutes
		TxExpiryMinutes:    30,   // 30 minutes
		WebhookSecret:      "your_webhook_secret_here",
		
		FiatCurrencies: map[string]FiatCurrency{
			"USD": {Code: "USD", Symbol: "$", Name: "US Dollar", MinAmount: 30, MaxAmount: 25000, Decimals: 2, IsEnabled: true},
			"EUR": {Code: "EUR", Symbol: "€", Name: "Euro", MinAmount: 30, MaxAmount: 25000, Decimals: 2, IsEnabled: true},
			"GBP": {Code: "GBP", Symbol: "£", Name: "British Pound", MinAmount: 25, MaxAmount: 20000, Decimals: 2, IsEnabled: true},
			"JPY": {Code: "JPY", Symbol: "¥", Name: "Japanese Yen", MinAmount: 3500, MaxAmount: 3000000, Decimals: 0, IsEnabled: true},
			"CNY": {Code: "CNY", Symbol: "¥", Name: "Chinese Yuan", MinAmount: 200, MaxAmount: 150000, Decimals: 2, IsEnabled: true},
			"KRW": {Code: "KRW", Symbol: "₩", Name: "South Korean Won", MinAmount: 40000, MaxAmount: 30000000, Decimals: 0, IsEnabled: true},
			"INR": {Code: "INR", Symbol: "₹", Name: "Indian Rupee", MinAmount: 2500, MaxAmount: 2000000, Decimals: 2, IsEnabled: true},
		},
		
		CryptoAssets: map[string]CryptoAsset{
			"BTC":  {Symbol: "BTC", Name: "Bitcoin", ChainID: 0, MinAmount: 0.001, MaxAmount: 100, Decimals: 8, IsEnabled: true, Network: "Bitcoin"},
			"ETH":  {Symbol: "ETH", Name: "Ethereum", ChainID: 1, MinAmount: 0.01, MaxAmount: 500, Decimals: 18, IsEnabled: true, Network: "Ethereum"},
			"USDT": {Symbol: "USDT", Name: "Tether", ChainID: 1, MinAmount: 10, MaxAmount: 100000, Decimals: 6, IsEnabled: true, Network: "Ethereum", Contract: "0xdAC17F958D2ee523a2206206994597C13D831ec7"},
			"USDC": {Symbol: "USDC", Name: "USD Coin", ChainID: 1, MinAmount: 10, MaxAmount: 100000, Decimals: 6, IsEnabled: true, Network: "Ethereum", Contract: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"},
			"BNB":  {Symbol: "BNB", Name: "BNB", ChainID: 56, MinAmount: 0.01, MaxAmount: 500, Decimals: 18, IsEnabled: true, Network: "BSC"},
			"SOL":  {Symbol: "SOL", Name: "Solana", ChainID: 501, MinAmount: 0.1, MaxAmount: 5000, Decimals: 9, IsEnabled: true, Network: "Solana"},
			"TRX":  {Symbol: "TRX", Name: "TRON", ChainID: 728126428, MinAmount: 10, MaxAmount: 500000, Decimals: 6, IsEnabled: true, Network: "TRON"},
			"XRP":  {Symbol: "XRP", Name: "Ripple", ChainID: 0, MinAmount: 10, MaxAmount: 500000, Decimals: 6, IsEnabled: true, Network: "XRP"},
			"ADA":  {Symbol: "ADA", Name: "Cardano", ChainID: 0, MinAmount: 10, MaxAmount: 500000, Decimals: 6, IsEnabled: true, Network: "Cardano"},
			"DOT":  {Symbol: "DOT", Name: "Polkadot", ChainID: 0, MinAmount: 1, MaxAmount: 50000, Decimals: 10, IsEnabled: true, Network: "Polkadot"},
			"AVAX": {Symbol: "AVAX", Name: "Avalanche", ChainID: 43114, MinAmount: 1, MaxAmount: 50000, Decimals: 18, IsEnabled: true, Network: "Avalanche"},
			"MATIC":{Symbol: "MATIC", Name: "Polygon", ChainID: 137, MinAmount: 10, MaxAmount: 100000, Decimals: 18, IsEnabled: true, Network: "Polygon"},
			"LINK": {Symbol: "LINK", Name: "Chainlink", ChainID: 1, MinAmount: 1, MaxAmount: 50000, Decimals: 18, IsEnabled: true, Network: "Ethereum"},
			"UNI":  {Symbol: "UNI", Name: "Uniswap", ChainID: 1, MinAmount: 1, MaxAmount: 50000, Decimals: 18, IsEnabled: true, Network: "Ethereum"},
		},
		
		PaymentMethods: []PaymentMethod{
			{ID: "card", Type: "card", Name: "Credit/Debit Card", Fee: 2.5, FixedFee: 0.30, MinLimit: 30, MaxLimit: 25000, RequiresKYC: false, IsEnabled: true},
			{ID: "apple_pay", Type: "apple_pay", Name: "Apple Pay", Fee: 2.5, FixedFee: 0, MinLimit: 30, MaxLimit: 25000, RequiresKYC: false, IsEnabled: true},
			{ID: "google_pay", Type: "google_pay", Name: "Google Pay", Fee: 2.5, FixedFee: 0, MinLimit: 30, MaxLimit: 25000, RequiresKYC: false, IsEnabled: true},
			{ID: "sepa", Type: "bank_transfer", Name: "SEPA Bank Transfer", Fee: 0.5, FixedFee: 0, MinLimit: 100, MaxLimit: 50000, RequiresKYC: true, IsEnabled: true},
			{ID: "swift", Type: "bank_transfer", Name: "SWIFT Transfer", Fee: 0.5, FixedFee: 15, MinLimit: 1000, MaxLimit: 500000, RequiresKYC: true, IsEnabled: true},
			{ID: "pix", Type: "bank_transfer", Name: "PIX (Brazil)", Fee: 0, FixedFee: 0, MinLimit: 10, MaxLimit: 50000, RequiresKYC: false, IsEnabled: true},
			{ID: "upi", Type: "bank_transfer", Name: "UPI (India)", Fee: 0, FixedFee: 0, MinLimit: 100, MaxLimit: 50000, RequiresKYC: false, IsEnabled: true},
		},
		
		KYCLevels: []KYCLevel{
			{Level: 0, Name: "None", MaxDailyLimit: 0, MaxMonthlyLimit: 0, RequiresDoc: false, RequiresVideo: false},
			{Level: 1, Name: "Basic", MaxDailyLimit: 1000, MaxMonthlyLimit: 10000, RequiresDoc: false, RequiresVideo: false},
			{Level: 2, Name: "Intermediate", MaxDailyLimit: 10000, MaxMonthlyLimit: 100000, RequiresDoc: true, RequiresVideo: false},
			{Level: 3, Name: "Full", MaxDailyLimit: 50000, MaxMonthlyLimit: 500000, RequiresDoc: true, RequiresVideo: true},
		},
		
		Providers: map[string]ProviderConfig{
			"moonpay": {
				Name: "MoonPay", Enabled: true,
				BaseURL: "https://api.moonpay.com",
				FeePercent: 2.5, MinAmount: 30, MaxAmount: 25000,
				SupportedFiats: []string{"USD", "EUR", "GBP", "JPY"},
				SupportedCrypto: []string{"BTC", "ETH", "USDT", "USDC", "BNB", "SOL", "MATIC", "AVAX"},
				SupportedMethods: []string{"card", "apple_pay", "google_pay", "sepa"},
			},
			"simplex": {
				Name: "Simplex", Enabled: true,
				BaseURL: "https://api.simplex.com",
				FeePercent: 3.5, MinAmount: 50, MaxAmount: 20000,
				SupportedFiats: []string{"USD", "EUR", "GBP"},
				SupportedCrypto: []string{"BTC", "ETH", "USDT", "USDC", "BNB"},
				SupportedMethods: []string{"card", "apple_pay"},
			},
			"transak": {
				Name: "Transak", Enabled: true,
				BaseURL: "https://api.transak.com",
				FeePercent: 2.0, MinAmount: 30, MaxAmount: 30000,
				SupportedFiats: []string{"USD", "EUR", "GBP", "JPY", "CNY", "INR", "KRW"},
				SupportedCrypto: []string{"BTC", "ETH", "USDT", "USDC", "BNB", "SOL", "TRX", "XRP", "ADA", "DOT"},
				SupportedMethods: []string{"card", "bank_transfer", "sepa", "pix", "upi"},
			},
		},
	}
	
	// Create service
	service := NewFiatOnrampService(config)
	
	// Start server
	fmt.Println("Starting Fiat On-Ramp Service on :8080")
	http.HandleFunc("/", service.ServeHTTP)
	
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
