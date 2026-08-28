//go:build ignore

// Standalone reference/demo service. Run individually with: go run <file>
// (Tagged "ignore" so the services/go directory is not a broken package —
//  these files are not part of any deployed build; deployed services live
//  under their own modules, e.g. go/*, */go.)
package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// Exchange Integration Service
// Supports: Binance, Coinbase, OKX, Bybit, BitGet
// ============================================================================

// Exchange types
type Exchange string

const (
	ExchangeBinance Exchange = "binance"
	ExchangeCoinbase Exchange = "coinbase"
	ExchangeOKX     Exchange = "okx"
	ExchangeBybit   Exchange = "bybit"
	ExchangeBitGet Exchange = "bitget"
)

// ExchangeConfig holds API credentials for an exchange
type ExchangeConfig struct {
	APIKey        string `json:"api_key"`
	SecretKey    string `json:"secret_key"`
	Passphrase   string `json:"passphrase,omitempty"` // For exchanges that require it
	APIBase      string `json:"api_base"`
	WSBase       string `json:"ws_base"`
	UseProxy     bool   `json:"use_proxy"`
	HTTPTimeout  int    `json:"http_timeout"` // seconds
	RateLimit    int    `json:"rate_limit"`  // requests per second
}

// ExchangeClient is the main client for exchange operations
type ExchangeClient struct {
	exchange Exchange
	config   ExchangeConfig
	client   *http.Client
	mu       sync.RWMutex
	rates    map[string]time.Time
}

// NewExchangeClient creates a new exchange client
func NewExchangeClient(exchange Exchange, config ExchangeConfig) *ExchangeClient {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		MaxIdleConns:   100,
		IdleConnTimeout: 30 * time.Second,
	}

	return &ExchangeClient{
		exchange: exchange,
		config:   config,
		client: &http.Client{
			Transport: transport,
			Timeout: time.Duration(config.HTTPTimeout) * time.Second,
		},
		rates: make(map[string]time.Time),
	}
}

// ============================================================================
// Binance API Implementation
// ============================================================================

// BinanceAPI is the Binance-specific implementation
type BinanceAPI struct {
	client *ExchangeClient
}

// NewBinanceAPI creates a new Binance API client
func NewBinanceAPI(client *ExchangeClient) *BinanceAPI {
	return &BinanceAPI{client: client}
}

// Binance ticker response
type BinanceTickerResponse struct {
	Symbol           string `json:"symbol"`
	PriceChange      string `json:"priceChange"`
	PriceChangePercent string `json:"priceChangePercent"`
	LastPrice       string `json:"lastPrice"`
	BidPrice        string `json:"bidPrice"`
	AskPrice        string `json:"askPrice"`
	HighPrice       string `json:"highPrice"`
	LowPrice        string `json:"lowPrice"`
	Volume          string `json:"volume"`
	QuoteVolume     string `json:"quoteVolume"`
}

// GetTicker fetches ticker data for a symbol
func (b *BinanceAPI) GetTicker(symbol string) (*PriceData, error) {
	params := map[string]string{
		"symbol": symbol,
	}

	data, err := b.client.request("GET", "/api/v3/ticker/24hr", params, true)
	if err != nil {
		return nil, err
	}

	var ticker BinanceTickerResponse
	if err := json.Unmarshal(data, &ticker); err != nil {
		return nil, err
	}

	price, _ := strconv.ParseFloat(ticker.LastPrice, 64)
	change, _ := strconv.ParseFloat(ticker.PriceChangePercent, 64)
	volume, _ := strconv.ParseFloat(ticker.Volume, 64)
	high, _ := strconv.ParseFloat(ticker.HighPrice, 64)
	low, _ := strconv.ParseFloat(ticker.LowPrice, 64)

	return &PriceData{
		Symbol:    symbol,
		Price:     price,
		Change24h: change,
		Volume24h: volume,
		High24h:   high,
		Low24h:    low,
		Timestamp: time.Now().Unix(),
	}, nil
}

// Binance order book response
type BinanceOrderBook struct {
	LastUpdateID int64       `json:"lastUpdateId"`
	Bids       [][]string  `json:"bids"`
	Asks       [][]string  `json:"asks"`
}

// GetOrderBook fetches the order book for a symbol
func (b *BinanceAPI) GetOrderBook(symbol string, limit int) (*OrderBook, error) {
	params := map[string]string{
		"symbol": symbol,
		"limit":  fmt.Sprintf("%d", limit),
	}

	data, err := b.client.request("GET", "/api/v3/depth", params, true)
	if err != nil {
		return nil, err
	}

	var book BinanceOrderBook
	if err := json.Unmarshal(data, &book); err != nil {
		return nil, err
	}

	ob := &OrderBook{
		Symbol:     symbol,
		Bids:       make([][2]float64, len(book.Bids)),
		Asks:       make([][2]float64, len(book.Asks)),
		LastUpdate:  book.LastUpdateID,
	}

	for i, bid := range book.Bids {
		ob.Bids[i][0], _ = strconv.ParseFloat(bid[0], 64)
		ob.Bids[i][1], _ = strconv.ParseFloat(bid[1], 64)
	}

	for i, ask := range book.Asks {
		ob.Asks[i][0], _ = strconv.ParseFloat(ask[0], 64)
		ob.Asks[i][1], _ = strconv.ParseFloat(ask[1], 64)
	}

	return ob, nil
}

// Binance account response
type BinanceAccountResponse struct {
	Balances []struct {
		Asset  string `json:"asset"`
		Free   string `json:"free"`
		Locked string `json:"locked"`
	} `json:"balances"`
}

// GetAccountBalances fetches account balances
func (b *BinanceAPI) GetAccountBalances() (map[string]Balance, error) {
	data, err := b.client.request("GET", "/api/v3/account", nil, true)
	if err != nil {
		return nil, err
	}

	var account BinanceAccountResponse
	if err := json.Unmarshal(data, &account); err != nil {
		return nil, err
	}

	balances := make(map[string]Balance)
	for _, balance := range account.Balances {
		free, _ := strconv.ParseFloat(balance.Free, 64)
		locked, _ := strconv.ParseFloat(balance.Locked, 64)

		if free+locked > 0 {
			balances[balance.Asset] = Balance{
				Available: free,
				Locked:   locked,
				Total:    free + locked,
			}
		}
	}

	return balances, nil
}

// Binance order response
type BinanceOrderResponse struct {
	OrderID       int64   `json:"orderId"`
	Symbol        string  `json:"symbol"`
	Price        float64 `json:"price"`
	OrigQty      float64 `json:"origQty"`
	ExecutedQty  float64 `json:"executedQty"`
	Status       string  `json:"status"`
	Side        string  `json:"side"`
	Type         string  `json:"type"`
	TimeInForce  string  `json:"timeInForce"`
}

// PlaceOrder places a new order
func (b *BinanceAPI) PlaceOrder(order *Order) (string, error) {
	params := map[string]string{
		"symbol":            order.Symbol,
		"side":              order.Side,
		"type":              order.Type,
		"quantity":          fmt.Sprintf("%.8f", order.Quantity),
		"timeInForce":        "GTC",
	}

	if order.Price > 0 {
		params["price"] = fmt.Sprintf("%.8f", order.Price)
	}

	data, err := b.client.request("POST", "/api/v3/order", params, true)
	if err != nil {
		return "", err
	}

	var response BinanceOrderResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return "", err
	}

	return fmt.Sprintf("%d", response.OrderID), nil
}

// CancelOrder cancels an existing order
func (b *BinanceAPI) CancelOrder(symbol string, orderID int64) error {
	params := map[string]string{
		"symbol":  symbol,
		"orderId": fmt.Sprintf("%d", orderID),
	}

	_, err := b.client.request("DELETE", "/api/v3/order", params, true)
	return err
}

// GetOpenOrders fetches all open orders
func (b *BinanceAPI) GetOpenOrders(symbol string) ([]Order, error) {
	params := map[string]string{}
	if symbol != "" {
		params["symbol"] = symbol
	}

	data, err := b.client.request("GET", "/api/v3/openOrders", params, true)
	if err != nil {
		return nil, err
	}

	var orders []BinanceOrderResponse
	if err := json.Unmarshal(data, &orders); err != nil {
		return nil, err
	}

	result := make([]Order, len(orders))
	for i, o := range orders {
		result[i] = Order{
			ID:         fmt.Sprintf("%d", o.OrderID),
			Symbol:     o.Symbol,
			Side:       o.Side,
			Type:       o.Type,
			Quantity:   o.OrigQty,
			Price:     o.Price,
			Status:     o.Status,
			Timestamp:  time.Now().Unix(),
		}
	}

	return result, nil
}

// ============================================================================
// Coinbase API Implementation
// ============================================================================

// CoinbaseAPI is the Coinbase-specific implementation
type CoinbaseAPI struct {
	client *ExchangeClient
}

// NewCoinbaseAPI creates a new Coinbase API client
func NewCoinbaseAPI(client *ExchangeClient) *CoinbaseAPI {
	return &CoinbaseAPI{client: client}
}

// Coinbase product ticker
type CoinbaseProduct struct {
	TradeID       int64   `json:"trade_id"`
	Price        string `json:"price"`
	Size         string `json:"size"`
	Volume       string `json:"volume"`
	Time         string `json:"time"`
}

// GetTicker fetches ticker data
func (c *CoinbaseAPI) GetTicker(productID string) (*PriceData, error) {
	path := fmt.Sprintf("/products/%s/ticker", productID)

	data, err := c.client.request("GET", path, nil, false)
	if err != nil {
		return nil, err
	}

	var ticker CoinbaseProduct
	if err := json.Unmarshal(data, &ticker); err != nil {
		return nil, err
	}

	price, _ := strconv.ParseFloat(ticker.Price, 64)
	volume, _ := strconv.ParseFloat(ticker.Volume, 64)

	return &PriceData{
		Symbol:    productID,
		Price:     price,
		Volume24h: volume,
		Timestamp: time.Now().Unix(),
	}, nil
}

// Coinbase account
type CoinbaseAccount struct {
	ID        string `json:"id"`
	Balance   string `json:"balance"`
	Available string `json:"available"`
	Currency  string `json:"currency"`
}

// GetAccountBalances fetches account balances
func (c *CoinbaseAPI) GetAccountBalances() (map[string]Balance, error) {
	data, err := c.client.request("GET", "/accounts", nil, true)
	if err != nil {
		return nil, err
	}

	var accounts []CoinbaseAccount
	if err := json.Unmarshal(data, &accounts); err != nil {
		return nil, err
	}

	balances := make(map[string]Balance)
	for _, account := range accounts {
		available, _ := strconv.ParseFloat(account.Available, 64)
		balance, _ := strconv.ParseFloat(account.Balance, 64)

		if balance > 0 {
			balances[account.Currency] = Balance{
				Available: available,
				Locked:    balance - available,
				Total:     balance,
			}
		}
	}

	return balances, nil
}

// ============================================================================
// OKX API Implementation
// ============================================================================

// OKXAPI is the OKX-specific implementation
type OKXAPI struct {
	client *ExchangeClient
}

// NewOKXAPI creates a new OKX API client
func NewOKXAPI(client *ExchangeClient) *OKXAPI {
	return &OKXAPI{client: client}
}

// OKX ticker response
type OKXTicker struct {
	InstID  string `json:"instId"`
	Last   string `json:"last"`
	Open   string `json:"open"`
	High   string `json:"high"`
	Low    string `json:"low"`
	VolCcy string `json:"volCcy"`
	Vol   string `json:"vol"`
}

// GetTicker fetches ticker data
func (o *OKXAPI) GetTicker(instID string) (*PriceData, error) {
	params := map[string]string{
		"instId": instID,
	}

	data, err := o.client.request("GET", "/api/v5/market/ticker", params, true)
	if err != nil {
		return nil, err
	}

	var tickers []OKXTicker
	if err := json.Unmarshal(data, &tickers); err != nil {
		return nil, err
	}

	if len(tickers) == 0 {
		return nil, fmt.Errorf("no ticker data")
	}

	ticker := tickers[0]
	price, _ := strconv.ParseFloat(ticker.Last, 64)
	volume, _ := strconv.ParseFloat(ticker.VolCcy, 64)
	high, _ := strconv.ParseFloat(ticker.High, 64)
	low, _ := strconv.ParseFloat(ticker.Low, 64)
	change := ((price - 0) / 0) * 100 // Calculate properly in production

	return &PriceData{
		Symbol:    instID,
		Price:     price,
		Change24h: change,
		Volume24h: volume,
		High24h:   high,
		Low24h:    low,
		Timestamp: time.Now().Unix(),
	}, nil
}

// ============================================================================
// Bybit API Implementation
// ============================================================================

// BybitAPI is the Bybit-specific implementation
type BybitAPI struct {
	client *ExchangeClient
}

// NewBybitAPI creates a new Bybit API client
func NewBybitAPI(client *ExchangeClient) *BybitAPI {
	return &BybitAPI{client: client}
}

// Bybit ticker
type BybitTicker struct {
	Symbol         string `json:"symbol1"`
	LastPrice     string `json:"lastPrice"`
	Price24h      string `json:"price24h"`
	Volume24h     string `json:"volume24h1"`
	HighPrice24h  string `json:"highPrice24h"`
	LowPrice24h   string `json:"lowPrice24h"`
}

// GetTicker fetches ticker data
func (by *BybitAPI) GetTicker(symbol string) (*PriceData, error) {
	params := map[string]string{
		"category": "spot",
		"symbol":   symbol,
	}

	data, err := by.client.request("GET", "/v5/market/ticker", params, true)
	if err != nil {
		return nil, err
	}

	// Parse response (simplified)
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	price, _ := strconv.ParseFloat("0", 64)

	return &PriceData{
		Symbol:    symbol,
		Price:     price,
		Timestamp: time.Now().Unix(),
	}, nil
}

// ============================================================================
// BitGet API Implementation
// ============================================================================

// BitGetAPI is the BitGet-specific implementation
type BitGetAPI struct {
	client *ExchangeClient
}

// NewBitGetAPI creates a new BitGet API client
func NewBitGetAPI(client *ExchangeClient) *BitGetAPI {
	return &BitGetAPI{client: client}
}

// GetTicker fetches ticker data
func (bg *BitGetAPI) GetTicker(symbol string) (*PriceData, error) {
	params := map[string]string{
		"symbol": symbol,
	}

	data, err := bg.client.request("GET", "/spot/v1/ticker/24hr", params, true)
	if err != nil {
		return nil, err
	}

	price, _ := strconv.ParseFloat("0", 64)

	return &PriceData{
		Symbol:    symbol,
		Price:     price,
		Timestamp: time.Now().Unix(),
	}, nil
}

// ============================================================================
// Generic Exchange Client Methods
// ============================================================================

// PriceData represents price information
type PriceData struct {
	Symbol    string  `json:"symbol"`
	Price    float64 `json:"price"`
	Change24h float64 `json:"change_24h"`
	Volume24h float64 `json:"volume_24h"`
	High24h  float64 `json:"high_24h"`
	Low24h   float64 `json:"low_24h"`
	Timestamp int64  `json:"timestamp"`
}

// Balance represents a balance
type Balance struct {
	Available float64 `json:"available"`
	Locked    float64 `json:"locked"`
	Total     float64 `json:"total"`
}

// Order represents a trading order
type Order struct {
	ID        string  `json:"id"`
	Symbol    string  `json:"symbol"`
	Side      string  `json:"side"`       // BUY or SELL
	Type      string  `json:"type"`       // LIMIT or MARKET
	Quantity  float64 `json:"quantity"`
	Price     float64 `json:"price"`
	Status    string  `json:"status"`
	Timestamp int64  `json:"timestamp"`
}

// OrderBook represents an order book
type OrderBook struct {
	Symbol    string       `json:"symbol"`
	Bids      [][2]float64 `json:"bids"`
	Asks      [][2]float64 `json:"asks"`
	LastUpdate int64       `json:"last_update"`
}

// request makes an API request
func (e *ExchangeClient) request(method, path string, params map[string]string, sign bool) ([]byte, error) {
	// Build URL
	baseURL := e.config.APIBase + path

	// Add query parameters
	if len(params) > 0 {
		query := url.Values{}
		for k, v := range params {
			query.Add(k, v)
		}
		baseURL += "?" + query.Encode()
	}

	// Create request
	req, err := http.NewRequest(method, baseURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	// Sign request if required
	if sign {
		timestamp := strconv.FormatInt(time.Now().Unix()*1000, 10)
		message := timestamp + method + path

		if len(params) > 0 {
			query := url.Values{}
			for k, v := range params {
				query.Add(k, v)
			}
			message += "?" + query.Encode()
		}

		signature := e.sign(message)
		req.Header.Set("X-BAPI-API-KEY", e.config.APIKey)
		req.Header.Set("X-BAPI-SIGNATURE", signature)
		req.Header.Set("X-BAPI-TIMESTAMP", timestamp)
		req.Header.Set("X-BAPI-COIN", "G")
	}

	// Make request
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s", string(body))
	}

	return body, nil
}

// sign creates a signature for the request
func (e *ExchangeClient) sign(message string) string {
	mac := hmac.New(sha256.New, []byte(e.config.SecretKey))
	mac.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// ============================================================================
// Exchange Service Factory
// ============================================================================

// NewExchange creates a new exchange API instance
func NewExchange(exchange Exchange, config ExchangeConfig) interface{} {
	client := NewExchangeClient(exchange, config)

	switch exchange {
	case ExchangeBinance:
		return NewBinanceAPI(client)
	case ExchangeCoinbase:
		return NewCoinbaseAPI(client)
	case ExchangeOKX:
		return NewOKXAPI(client)
	case ExchangeBybit:
		return NewBybitAPI(client)
	case ExchangeBitGet:
		return NewBitGetAPI(client)
	default:
		return nil
	}
}

// ============================================================================
// Fiat On-Ramp Service
// ============================================================================

// FiatRampConfig holds configuration for fiat on-ramp providers
type FiatRampConfig struct {
	Provider    string `json:"provider"` // moonpay, transak, wyre
	APIKey     string `json:"api_key"`
	APIBase    string `json:"api_base"`
	WebhookURL string `json:"webhook_url"`
}

// FiatRampProvider is the interface for fiat on-ramp providers
type FiatRampProvider interface {
	CreateTransaction(req *FiatTransactionRequest) (*FiatTransaction, error)
	GetTransaction(id string) (*FiatTransaction, error)
	GetQuote(req *FiatQuoteRequest) (*FiatQuote, error)
}

// FiatTransactionRequest represents a fiat transaction request
type FiatTransactionRequest struct {
	WalletAddress string  `json:"wallet_address"`
	CryptoSymbol   string  `json:"crypto_symbol"`
	FiatSymbol    string  `json:"fiat_symbol"`
	Amount        float64 `json:"amount"`
	FiatAmount   float64 `json:"fiat_amount"`
	RedirectURL  string  `json:"redirect_url"`
}

// FiatTransaction represents a fiat transaction
type FiatTransaction struct {
	ID              string  `json:"id"`
	Status          string  `json:"status"`
	CryptoAmount    float64 `json:"crypto_amount"`
	FiatAmount     float64 `json:"fiat_amount"`
	CryptoSymbol   string  `json:"crypto_symbol"`
	FiatSymbol     string  `json:"fiat_symbol"`
	CryptoAddress  string  `json:"crypto_address"`
	RedirectURL    string  `json:"redirect_url"`
	Timestamp      int64   `json:"timestamp"`
}

// FiatQuoteRequest represents a fiat quote request
type FiatQuoteRequest struct {
	CryptoSymbol string  `json:"crypto_symbol"`
	FiatSymbol  string  `json:"fiat_symbol"`
	Amount     float64 `json:"amount"`
	Side       string  `json:"side"` // buy or sell
}

// FiatQuote represents a fiat quote
type FiatQuote struct {
	CryptoAmount   float64 `json:"crypto_amount"`
	FiatAmount    float64 `json:"fiat_amount"`
	CryptoSymbol  string  `json:"crypto_symbol"`
	FiatSymbol    string  `json:"fiat_symbol"`
	ExchangeRate float64 `json:"exchange_rate"`
	ExpiresAt    int64   `json:"expires_at"`
}

// MoonPayProvider implements fiat on-ramp via MoonPay
type MoonPayProvider struct {
	config FiatRampConfig
	client *http.Client
}

// NewMoonPayProvider creates a new MoonPay provider
func NewMoonPayProvider(config FiatRampConfig) *MoonPayProvider {
	return &MoonPayProvider{
		config: config,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// CreateTransaction creates a new MoonPay transaction
func (m *MoonPayProvider) CreateTransaction(req *FiatTransactionRequest) (*FiatTransaction, error) {
	body, _ := json.Marshal(map[string]interface{}{
		"walletAddress": req.WalletAddress,
		"baseCurrency":  req.FiatSymbol,
		"quoteCurrency": req.CryptoSymbol,
		"baseAmount":   req.FiatAmount,
		"redirectURL":  req.RedirectURL,
	})

	httpReq, err := http.NewRequest("POST", m.config.APIBase+"/v1/transactions", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "ApiKey "+m.config.APIKey)

	resp, err := m.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &FiatTransaction{
		ID:             result["id"].(string),
		Status:         "pending",
		CryptoAddress:  req.WalletAddress,
		FiatAmount:     req.FiatAmount,
		CryptoSymbol:  req.CryptoSymbol,
		FiatSymbol:     req.FiatSymbol,
		RedirectURL:    result["url"].(string),
		Timestamp:     time.Now().Unix(),
	}, nil
}

// GetTransaction gets transaction status
func (m *MoonPayProvider) GetTransaction(id string) (*FiatTransaction, error) {
	httpReq, err := http.NewRequest("GET", m.config.APIBase+"/v1/transactions/"+id, nil)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "ApiKey "+m.config.APIKey)

	resp, err := m.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return &FiatTransaction{
		ID:        id,
		Status:    "completed",
		Timestamp: time.Now().Unix(),
	}, nil
}

// GetQuote gets a fiat quote
func (m *MoonPayProvider) GetQuote(req *FiatQuoteRequest) (*FiatQuote, error) {
	// In production, this would call MoonPay's quote API
	return &FiatQuote{
		CryptoAmount:  req.Amount * 0.001, // Simplified
		FiatAmount:   req.Amount,
		CryptoSymbol: req.CryptoSymbol,
		FiatSymbol:   req.FiatSymbol,
		ExchangeRate: 0.001,
		ExpiresAt:    time.Now().Add(5 * time.Minute).Unix(),
	}, nil
}

// ============================================================================
// Notification Service
// ============================================================================

// NotificationConfig holds notification service configuration
type NotificationConfig struct {
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUser     string `json:"smtp_user"`
	SMTPPassword string `json:"smtp_password"`
	FromEmail    string `json:"from_email"`
	TelegramBot  string `json:"telegram_bot"`
	TelegramChat string `json:"telegram_chat"`
	PushAPIKey   string `json:"push_api_key"`
}

// NotificationService handles all notification types
type NotificationService struct {
	config NotificationConfig
	email  *SMTPClient
	push   *PushNotificationClient
	telegram *TelegramClient
}

// SMTPClient handles email notifications
type SMTPClient struct {
	host     string
	port     int
	username string
	password string
	from     string
}

// SendEmail sends an email notification
func (s *SMTPClient) SendEmail(to, subject, body string) error {
	// In production, implement SMTP sending
	return nil
}

// PushNotificationClient handles push notifications
type PushNotificationClient struct {
	apiKey string
}

// SendPush sends a push notification
func (p *PushNotificationClient) SendPush(token, title, body string) error {
	// In production, implement FCM/APNs sending
	return nil
}

// TelegramClient handles Telegram notifications
type TelegramClient struct {
	botToken string
	chatID   string
}

// SendTelegram sends a Telegram message
func (t *TelegramClient) SendTelegram(message string) error {
	// In production, implement Telegram Bot API
	return nil
}

// NewNotificationService creates a new notification service
func NewNotificationService(config NotificationConfig) *NotificationService {
	return &NotificationService{
		config: config,
		email: &SMTPClient{
			host:     config.SMTPHost,
			port:     config.SMTPPort,
			username: config.SMTPUser,
			password: config.SMTPPassword,
			from:     config.FromEmail,
		},
		push: &PushNotificationClient{
			apiKey: config.PushAPIKey,
		},
		telegram: &TelegramClient{
			botToken: config.TelegramBot,
			chatID:   config.TelegramChat,
		},
	}
}

// SendPriceAlert sends a price alert notification
func (n *NotificationService) SendPriceAlert(symbol string, price, target float64, direction string) error {
	title := fmt.Sprintf("%s Price Alert", symbol)
	body := fmt.Sprintf("%s has reached $%.2f. Target: $%.2f (%s)", symbol, price, target, direction)

	if err := n.email.SendEmail("", title, body); err != nil {
		return err
	}

	if err := n.telegram.SendTelegram(title + "\n" + body); err != nil {
		return err
	}

	return nil
}

// SendTransactionNotification sends a transaction notification
func (n *NotificationService) SendTransactionNotification(txHash, status, symbol string, amount float64) error {
	title := fmt.Sprintf("Transaction %s", status)
	body := fmt.Sprintf("%s: %s %f\nTx: %s", status, symbol, amount, txHash)

	return n.telegram.SendTelegram(title + "\n" + body)
}

// getRequiredEnv reads a required environment variable and fatally exits if it
// is unset. Used for secrets/credentials that must never fall back to insecure
// hardcoded defaults.
func getRequiredEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s environment variable must be set", key)
	}
	return v
}

// ============================================================================
// Main Entry Point
// ============================================================================

func main() {
	fmt.Println("TigerWallet Exchange Integration Service")
	fmt.Println("========================================")

	// Example: Create Binance client
	binanceConfig := ExchangeConfig{
		APIKey:       getRequiredEnv("EXCHANGE_API_KEY"),
		SecretKey:    getRequiredEnv("EXCHANGE_API_SECRET"),
		APIBase:      "https://api.binance.com",
		WSBase:       "wss://stream.binance.com:9443",
		HTTPTimeout:  30,
		RateLimit:    10,
	}

	binance := NewExchange(ExchangeBinance, binanceConfig)
	if b, ok := binance.(*BinanceAPI); ok {
		ticker, err := b.GetTicker("ETHUSDT")
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Printf("ETHUSDT: $%.2f (%.2f%%)\n", ticker.Price, ticker.Change24h)
		}
	}
}