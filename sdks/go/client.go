package tigerwallet

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client represents a TigerWallet API client
type Client struct {
	baseURL     string
	apiKey      string
	apiSecret   string
	httpClient  *http.Client
	tenantID    string
	rateLimiter *RateLimiter
}

// Option is a functional option for configuring the client
type Option func(*Client)

// WithBaseURL sets the base URL for the API
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithTenantID sets the tenant ID
func WithTenantID(tenantID string) Option {
	return func(c *Client) {
		c.tenantID = tenantID
	}
}

// WithRateLimiter enables rate limiting
func WithRateLimiter(requestsPerSecond int) Option {
	return func(c *Client) {
		c.rateLimiter = NewRateLimiter(requestsPerSecond)
	}
}

// NewClient creates a new TigerWallet API client
func NewClient(apiKey, apiSecret string, opts ...Option) *Client {
	c := &Client{
		baseURL:    "https://api.tigerwallet.com",
		apiKey:     apiKey,
		apiSecret:  apiSecret,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Request makes an authenticated request to the API
func (c *Client) Request(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	// Apply rate limiting
	if c.rateLimiter != nil {
		c.rateLimiter.Wait(ctx)
	}

	// Serialize body
	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = json.Marshal(body)
	}

	// Generate signature
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	signature := c.generateSignature(method, path, timestamp, bodyBytes)

	// Build request
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("X-Timestamp", timestamp)
	req.Header.Set("X-Signature", signature)
	if c.tenantID != "" {
		req.Header.Set("X-Tenant-ID", c.tenantID)
	}

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		json.Unmarshal(respBody, &errResp)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, errResp.Message)
	}

	return respBody, nil
}

func (c *Client) generateSignature(method, path, timestamp string, body []byte) string {
	message := fmt.Sprintf("%s\n%s\n%s\n%s", method, path, timestamp, string(body))
	h := hmac.New(sha256.New, []byte(c.apiSecret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// Fetcher Service
type FetcherService struct {
	client *Client
}

// NewFetcherService creates a new fetcher service
func NewFetcherService(client *Client) *FetcherService {
	return &FetcherService{client: client}
}

// GetPrices retrieves token prices
func (s *FetcherService) GetPrices(ctx context.Context, symbols []string) (*PriceResponse, error) {
	path := fmt.Sprintf("/api/v1/fetcher/prices?symbols=%s", strings.Join(symbols, ","))
	respBody, err := s.client.Request(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var resp PriceResponse
	json.Unmarshal(respBody, &resp)
	return &resp, nil
}

// GetWalletBalance retrieves wallet balance
func (s *FetcherService) GetWalletBalance(ctx context.Context, chain, address string) (*WalletBalanceResponse, error) {
	path := fmt.Sprintf("/api/v1/fetcher/wallet/%s/%s", chain, address)
	respBody, err := s.client.Request(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var resp WalletBalanceResponse
	json.Unmarshal(respBody, &resp)
	return &resp, nil
}

// GetTransactions retrieves transactions
func (s *FetcherService) GetTransactions(ctx context.Context, chain, address string, limit int) (*TransactionsResponse, error) {
	path := fmt.Sprintf("/api/v1/fetcher/transactions/%s/%s?limit=%d", chain, address, limit)
	respBody, err := s.client.Request(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var resp TransactionsResponse
	json.Unmarshal(respBody, &resp)
	return &resp, nil
}

// GetTokenInfo retrieves token information
func (s *FetcherService) GetTokenInfo(ctx context.Context, chain, tokenAddress string) (*TokenInfoResponse, error) {
	path := fmt.Sprintf("/api/v1/fetcher/token/%s/%s", chain, tokenAddress)
	respBody, err := s.client.Request(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var resp TokenInfoResponse
	json.Unmarshal(respBody, &resp)
	return &resp, nil
}

// GetMarketData retrieves market data
func (s *FetcherService) GetMarketData(ctx context.Context, symbols []string) (*MarketDataResponse, error) {
	path := fmt.Sprintf("/api/v1/fetcher/market?symbols=%s", strings.Join(symbols, ","))
	respBody, err := s.client.Request(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var resp MarketDataResponse
	json.Unmarshal(respBody, &resp)
	return &resp, nil
}

// Permission Service
type PermissionService struct {
	client *Client
}

// NewPermissionService creates a new permission service
func NewPermissionService(client *Client) *PermissionService {
	return &PermissionService{client: client}
}

// GetPermissions retrieves permissions
func (s *PermissionService) GetPermissions(ctx context.Context) (*PermissionsResponse, error) {
	respBody, err := s.client.Request(ctx, "GET", "/api/v1/permissions", nil)
	if err != nil {
		return nil, err
	}

	var resp PermissionsResponse
	json.Unmarshal(respBody, &resp)
	return &resp, nil
}

// CheckPermission checks if a feature is enabled
func (s *PermissionService) CheckPermission(ctx context.Context, feature string) (bool, error) {
	path := fmt.Sprintf("/api/v1/permissions/%s", feature)
	respBody, err := s.client.Request(ctx, "GET", path, nil)
	if err != nil {
		return false, err
	}

	var resp PermissionCheckResponse
	json.Unmarshal(respBody, &resp)
	return resp.Enabled, nil
}

// SyncPermissions syncs permissions from server
func (s *PermissionService) SyncPermissions(ctx context.Context) (*PermissionsResponse, error) {
	respBody, err := s.client.Request(ctx, "POST", "/api/v1/permissions/sync", nil)
	if err != nil {
		return nil, err
	}

	var resp PermissionsResponse
	json.Unmarshal(respBody, &resp)
	return &resp, nil
}

// License Service
type LicenseService struct {
	client *Client
}

// NewLicenseService creates a new license service
func NewLicenseService(client *Client) *LicenseService {
	return &LicenseService{client: client}
}

// ValidateLicense validates a license key
func (s *LicenseService) ValidateLicense(ctx context.Context, licenseKey, hardwareID string) (*LicenseValidationResponse, error) {
	body := map[string]string{
		"license_key":  licenseKey,
		"hardware_id": hardwareID,
	}
	respBody, err := s.client.Request(ctx, "POST", "/api/v1/licenses/validate", body)
	if err != nil {
		return nil, err
	}

	var resp LicenseValidationResponse
	json.Unmarshal(respBody, &resp)
	return &resp, nil
}

// GetLicenseInfo retrieves license information
func (s *LicenseService) GetLicenseInfo(ctx context.Context) (*LicenseInfoResponse, error) {
	respBody, err := s.client.Request(ctx, "GET", "/api/v1/licenses/info", nil)
	if err != nil {
		return nil, err
	}

	var resp LicenseInfoResponse
	json.Unmarshal(respBody, &resp)
	return &resp, nil
}

// Webhook Service
type WebhookService struct {
	client *Client
}

// NewWebhookService creates a new webhook service
func NewWebhookService(client *Client) *WebhookService {
	return &WebhookService{client: client}
}

// RegisterWebhook registers a webhook endpoint
func (s *WebhookService) RegisterWebhook(ctx context.Context, eventType, url, secret string) (*WebhookResponse, error) {
	body := map[string]string{
		"event_type": eventType,
		"url":        url,
		"secret":     secret,
	}
	respBody, err := s.client.Request(ctx, "POST", "/api/v1/webhooks", body)
	if err != nil {
		return nil, err
	}

	var resp WebhookResponse
	json.Unmarshal(respBody, &resp)
	return &resp, nil
}

// VerifyWebhook verifies a webhook signature
func (s *WebhookService) VerifyWebhook(payload, signature, secret string) bool {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(payload))
	expectedSig := hex.EncodeToString(h.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expectedSig))
}

// Response Types
type PriceResponse struct {
	Success bool              `json:"success"`
	Prices  map[string]Price `json:"prices"`
}

type Price struct {
	Symbol    string  `json:"symbol"`
	Price     float64 `json:"price"`
	Change24h float64 `json:"change_24h"`
	Volume24h float64 `json:"volume_24h"`
	Timestamp int64   `json:"timestamp"`
}

type WalletBalanceResponse struct {
	Success  bool           `json:"success"`
	Address  string         `json:"address"`
	Chain    string         `json:"chain"`
	Balance  float64        `json:"balance"`
	Tokens   []TokenBalance `json:"tokens"`
}

type TokenBalance struct {
	Symbol    string  `json:"symbol"`
	Address   string  `json:"address"`
	Balance   float64 `json:"balance"`
	Price     float64 `json:"price"`
	ValueUSD  float64 `json:"value_usd"`
}

type TransactionsResponse struct {
	Success      bool          `json:"success"`
	Transactions []Transaction `json:"transactions"`
	Total        int           `json:"total"`
}

type Transaction struct {
	Hash        string    `json:"hash"`
	From        string    `json:"from"`
	To          string    `json:"to"`
	Value       float64   `json:"value"`
	Token       string    `json:"token"`
	Timestamp   int64     `json:"timestamp"`
	Status      string    `json:"status"`
	BlockNumber int64     `json:"block_number"`
}

type TokenInfoResponse struct {
	Success    bool    `json:"success"`
	Address    string  `json:"address"`
	Chain      string  `json:"chain"`
	Name       string  `json:"name"`
	Symbol     string  `json:"symbol"`
	Decimals   int     `json:"decimals"`
	TotalSupply float64 `json:"total_supply"`
	Price      float64 `json:"price"`
	MarketCap  float64 `json:"market_cap"`
}

type MarketDataResponse struct {
	Success bool           `json:"success"`
	Data    []MarketData   `json:"data"`
}

type MarketData struct {
	Symbol         string  `json:"symbol"`
	Price          float64 `json:"price"`
	PriceChange1h  float64 `json:"price_change_1h"`
	PriceChange24h float64 `json:"price_change_24h"`
	Volume24h      float64 `json:"volume_24h"`
	MarketCap      float64 `json:"market_cap"`
}

type PermissionsResponse struct {
	Success     bool                   `json:"success"`
	Permissions map[string]Permission   `json:"permissions"`
	SyncedAt    int64                  `json:"synced_at"`
}

type Permission struct {
	Enabled bool     `json:"enabled"`
	Limit   int      `json:"limit,omitempty"`
	Usage   int      `json:"usage,omitempty"`
}

type PermissionCheckResponse struct {
	Enabled bool   `json:"enabled"`
	Feature string `json:"feature"`
}

type LicenseValidationResponse struct {
	Valid       bool   `json:"valid"`
	LicenseKey  string `json:"license_key"`
	Product     string `json:"product"`
	ExpiresAt   int64  `json:"expires_at"`
	TenantID    string `json:"tenant_id"`
}

type LicenseInfoResponse struct {
	Product      string   `json:"product"`
	Plan         string   `json:"plan"`
	Status       string   `json:"status"`
	Features     []string `json:"features"`
	ExpiresAt    int64    `json:"expires_at"`
	ApiCallLimit int     `json:"api_call_limit"`
	ApiCallsUsed int     `json:"api_calls_used"`
}

type WebhookResponse struct {
	ID        string `json:"id"`
	EventType string `json:"event_type"`
	URL       string `json:"url"`
	Status    string `json:"status"`
}

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	tokens    chan struct{}
	rate      int
	 refill   time.Duration
}

func NewRateLimiter(requestsPerSecond int) *RateLimiter {
	rl := &RateLimiter{
		tokens: make(chan struct{}, requestsPerSecond),
		rate:   requestsPerSecond,
		refill: time.Second,
	}
	
	// Fill initial tokens
	for i := 0; i < requestsPerSecond; i++ {
		rl.tokens <- struct{}{}
	}
	
	// Refill tokens
	go func() {
		for {
			time.Sleep(rl.refill)
			select {
			case rl.tokens <- struct{}{}:
			default:
			}
		}
	}()
	
	return rl
}

func (rl *RateLimiter) Wait(ctx context.Context) {
	select {
	case <-rl.tokens:
		return
	case <-ctx.Done():
		return
	}
}
