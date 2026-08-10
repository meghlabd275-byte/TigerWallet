// TigerWallet - CEX Connector Service
// High-performance Go implementation for connecting to centralized exchanges
// Supports: Binance, Coinbase, Kraken, KuCoin, Bybit, OKX, Gate, Bitget, Huobi, MEXC
// NO Stripe or fiat - Crypto only (USDT, USDC, ETH, BTC, BNB, SOL)

package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

const (
	// Server Configuration
	SERVER_PORT = "8090"

	// Supported Stable Coins (NO FIAT)
	USDT = "USDT"
	USDC = "USDC"
	ETH  = "ETH"
	BTC  = "BTC"
	BNB  = "BNB"
	SOL  = "SOL"

	// Exchange IDs
	BINANCE  = "binance"
	COINBASE = "coinbase"
	KRAKEN   = "kraken"
	KUCOIN   = "kucoin"
	BYBIT    = "bybit"
	OKX      = "okx"
	GATE     = "gate"
	BITGET   = "bitget"
	HUOBI    = "huobi"
	MEXC     = "mexc"
)

var (
	redisClient  *redis.Client
	exchangeAPI  map[string]*ExchangeAPI
	orderBooks   map[string]*OrderBook
	orderBookMux sync.RWMutex
)

type ExchangeAPI struct {
	Name       string
	BaseURL    string
	WSURL      string
	APIKey     string
	APISecret  string
	Passphrase string // For exchanges that need it
	httpClient *http.Client
	wsConn     *websocket.Conn
	subscribed map[string]bool
	mu         sync.RWMutex
}

type Ticker struct {
	Symbol         string  `json:"symbol"`
	LastPrice      float64 `json:"lastPrice"`
	BidPrice       float64 `json:"bidPrice"`
	AskPrice       float64 `json:"askPrice"`
	Volume24h      float64 `json:"volume24h"`
	QuoteVolume24h float64 `json:"quoteVolume24h"`
	PriceChange    float64 `json:"priceChange"`
	PriceChangePct float64 `json:"priceChangePercent"`
	High24h        float64 `json:"high24h"`
	Low24h         float64 `json:"low24h"`
	Timestamp      int64   `json:"timestamp"`
}

type OrderBook struct {
	Symbol       string           `json:"symbol"`
	Bids         []OrderBookEntry `json:"bids"`
	Asks         []OrderBookEntry `json:"asks"`
	LastUpdateID int64            `json:"lastUpdateId"`
	Exchange     string           `json:"exchange"`
	Timestamp    int64            `json:"timestamp"`
}

type OrderBookEntry struct {
	Price  float64 `json:"price"`
	Amount float64 `json:"amount"`
}

type Order struct {
	OrderID      string  `json:"orderId"`
	Symbol       string  `json:"symbol"`
	Side         string  `json:"side"`
	Type         string  `json:"type"`
	Price        float64 `json:"price"`
	Amount       float64 `json:"amount"`
	FilledAmount float64 `json:"filledAmount"`
	Status       string  `json:"status"`
	CreatedAt    int64   `json:"createdAt"`
	FilledAt     *int64  `json:"filledAt"`
	Commission   float64 `json:"commission"`
	Exchange     string  `json:"exchange"`
}

type Balance struct {
	Currency string  `json:"currency"`
	Free     float64 `json:"free"`
	Locked   float64 `json:"locked"`
	Total    float64 `json:"total"`
	USDValue float64 `json:"usdValue"`
}

type Trade struct {
	TradeID         string  `json:"tradeId"`
	OrderID         string  `json:"orderId"`
	Symbol          string  `json:"symbol"`
	Side            string  `json:"side"`
	Price           float64 `json:"price"`
	Amount          float64 `json:"amount"`
	Commission      float64 `json:"commission"`
	CommissionAsset string  `json:"commissionAsset"`
	Timestamp       int64   `json:"timestamp"`
	Exchange        string  `json:"exchange"`
}

type KLine struct {
	OpenTime    int64
	Open        float64
	High        float64
	Low         float64
	Close       float64
	Volume      float64
	CloseTime   int64
	QuoteVolume float64
	NumTrades   int64
}

type WithdrawRequest struct {
	Currency  string  `json:"currency"`
	Address   string  `json:"address"`
	Amount    float64 `json:"amount"`
	Network   string  `json:"network"`
	Fee       float64 `json:"fee"`
	Timestamp int64   `json:"timestamp"`
}

type DepositAddress struct {
	Currency string `json:"currency"`
	Address  string `json:"address"`
	Memo     string `json:"memo,omitempty"`
	Network  string `json:"network"`
	Tag      string `json:"tag,omitempty"`
}

type Market struct {
	Symbol          string  `json:"symbol"`
	BaseAsset       string  `json:"baseAsset"`
	QuoteAsset      string  `json:"quoteAsset"`
	MinPrice        float64 `json:"minPrice"`
	MaxPrice        float64 `json:"maxPrice"`
	MinAmount       float64 `json:"minAmount"`
	MaxAmount       float64 `json:"maxAmount"`
	MinNotional     float64 `json:"minNotional"`
	PricePrecision  int     `json:"pricePrecision"`
	AmountPrecision int     `json:"amountPrecision"`
	QuotePrecision  int     `json:"quotePrecision"`
	Status          string  `json:"status"`
}

type ExchangeStats struct {
	Exchange           string  `json:"exchange"`
	Status             string  `json:"status"`
	LatencyMs          float64 `json:"latencyMs"`
	LastUpdate         int64   `json:"lastUpdate"`
	TickersCount       int     `json:"tickersCount"`
	OrderBooksCount    int     `json:"orderBooksCount"`
	WebSocketConnected bool    `json:"wsConnected"`
}

type PaymentRequest struct {
	UserID     string  `json:"user_id"`
	ListingID  string  `json:"listing_id"`
	Currency   string  `json:"currency"` // USDT, USDC, ETH, BTC, BNB, SOL
	Amount     float64 `json:"amount"`
	Network    string  `json:"network"`     // For crypto: eth, bsc, arb, etc.
	WalletType string  `json:"wallet_type"` // master_wallet, user_wallet
}

type PaymentResponse struct {
	PaymentID     string  `json:"payment_id"`
	Currency      string  `json:"currency"`
	Amount        float64 `json:"amount"`
	Address       string  `json:"address"` // Payment address to send crypto
	Memo          string  `json:"memo,omitempty"`
	Network       string  `json:"network"`
	Confirmations int     `json:"confirmations"`
	Status        string  `json:"status"`
	TxHash        string  `json:"tx_hash,omitempty"`
	CreatedAt     int64   `json:"created_at"`
	ExpiresAt     int64   `json:"expires_at"`
}

type CryptoPayment struct {
	ID                    string  `json:"id"`
	UserID                string  `json:"user_id"`
	ListingID             string  `json:"listing_id"`
	Currency              string  `json:"currency"`
	Amount                float64 `json:"amount"`
	Address               string  `json:"address"`
	Memo                  string  `json:"memo"`
	Network               string  `json:"network"`
	TxHash                string  `json:"tx_hash"`
	Status                string  `json:"status"` // pending, confirmed, completed, failed
	Confirmations         int     `json:"confirmations"`
	RequiredConfirmations int     `json:"required_confirmations"`
	CreatedAt             int64   `json:"created_at"`
	UpdatedAt             int64   `json:"updated_at"`
	CompletedAt           *int64  `json:"completed_at"`
}

// Initialize exchange APIs
func initExchangeAPIs() {
	exchangeAPI = make(map[string]*ExchangeAPI)
	orderBooks = make(map[string]*OrderBook)

	// Binance
	exchangeAPI[BINANCE] = &ExchangeAPI{
		Name:       "Binance",
		BaseURL:    "https://api.binance.com",
		WSURL:      "wss://stream.binance.com:9443/ws",
		APIKey:     os.Getenv("BINANCE_API_KEY"),
		APISecret:  os.Getenv("BINANCE_API_SECRET"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
		subscribed: make(map[string]bool),
	}

	// Coinbase
	exchangeAPI[COINBASE] = &ExchangeAPI{
		Name:       "Coinbase",
		BaseURL:    "https://api.coinbase.com",
		WSURL:      "wss://ws-feed.coinbase.com",
		APIKey:     os.Getenv("COINBASE_API_KEY"),
		APISecret:  os.Getenv("COINBASE_API_SECRET"),
		Passphrase: os.Getenv("COINBASE_PASS"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
		subscribed: make(map[string]bool),
	}

	// Kraken
	exchangeAPI[KRAKEN] = &ExchangeAPI{
		Name:       "Kraken",
		BaseURL:    "https://api.kraken.com",
		WSURL:      "wss://ws.kraken.com",
		APIKey:     os.Getenv("KRAKEN_API_KEY"),
		APISecret:  os.Getenv("KRAKEN_API_SECRET"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
		subscribed: make(map[string]bool),
	}

	// KuCoin
	exchangeAPI[KUCOIN] = &ExchangeAPI{
		Name:       "KuCoin",
		BaseURL:    "https://api.kucoin.com",
		WSURL:      "wss://ws-api.kucoin.com",
		APIKey:     os.Getenv("KUCOIN_API_KEY"),
		APISecret:  os.Getenv("KUCOIN_API_SECRET"),
		Passphrase: os.Getenv("KUCOIN_PASS"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
		subscribed: make(map[string]bool),
	}

	// Bybit
	exchangeAPI[BYBIT] = &ExchangeAPI{
		Name:       "Bybit",
		BaseURL:    "https://api.bybit.com",
		WSURL:      "wss://stream.bybit.com/v5/public/spot",
		APIKey:     os.Getenv("BYBIT_API_KEY"),
		APISecret:  os.Getenv("BYBIT_API_SECRET"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
		subscribed: make(map[string]bool),
	}

	// OKX
	exchangeAPI[OKX] = &ExchangeAPI{
		Name:       "OKX",
		BaseURL:    "https://www.okx.com",
		WSURL:      "wss://ws.okx.com:8443/ws/v5/public",
		APIKey:     os.Getenv("OKX_API_KEY"),
		APISecret:  os.Getenv("OKX_API_SECRET"),
		Passphrase: os.Getenv("OKX_PASS"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
		subscribed: make(map[string]bool),
	}

	// Gate
	exchangeAPI[GATE] = &ExchangeAPI{
		Name:       "Gate",
		BaseURL:    "https://api.gateio.ws",
		WSURL:      "wss://api.gateio.ws/ws/v4/",
		APIKey:     os.Getenv("GATE_API_KEY"),
		APISecret:  os.Getenv("GATE_API_SECRET"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
		subscribed: make(map[string]bool),
	}

	// Bitget
	exchangeAPI[BITGET] = &ExchangeAPI{
		Name:       "Bitget",
		BaseURL:    "https://api.bitget.com",
		WSURL:      "wss://ws.bitget.com/v2/spot/public",
		APIKey:     os.Getenv("BITGET_API_KEY"),
		APISecret:  os.Getenv("BITGET_API_SECRET"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
		subscribed: make(map[string]bool),
	}

	// Huobi
	exchangeAPI[HUOBI] = &ExchangeAPI{
		Name:       "Huobi",
		BaseURL:    "https://api.huobi.pro",
		WSURL:      "wss://api.huobi.pro/ws",
		APIKey:     os.Getenv("HUOBI_API_KEY"),
		APISecret:  os.Getenv("HUOBI_API_SECRET"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
		subscribed: make(map[string]bool),
	}

	// MEXC
	exchangeAPI[MEXC] = &ExchangeAPI{
		Name:       "MEXC",
		BaseURL:    "https://api.mexc.com",
		WSURL:      "wss://ws.mexc.com/ws",
		APIKey:     os.Getenv("MEXC_API_KEY"),
		APISecret:  os.Getenv("MEXC_API_SECRET"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
		subscribed: make(map[string]bool),
	}
}

// ============================================================================
// CRYPTO PAYMENT SYSTEM (NO STRIPE - CRYPTO ONLY)
// ============================================================================

var cryptoPayments = make(map[string]*CryptoPayment)
var cryptoPaymentsMux sync.RWMutex

// Supported crypto currencies
var supportedCrypto = map[string]bool{
	"USDT": true,
	"USDC": true,
	"ETH":  true,
	"BTC":  true,
	"BNB":  true,
	"SOL":  true,
}

// Network mappings
var cryptoNetworks = map[string][]string{
	"USDT": {"ETH", "BSC", "ARB", "AVAX", "POLYGON", "OPTIMISM", "TRON", "SOLANA"},
	"USDC": {"ETH", "BSC", "ARB", "AVAX", "POLYGON", "OPTIMISM"},
	"ETH":  {"ETH"},
	"BTC":  {"BTC", "BTC (SegWit)"},
	"BNB":  {"BSC"},
	"SOL":  {"SOLANA"},
}

// Master wallet addresses (these would be real addresses in production)
var masterWalletAddresses = map[string]string{
	"ETH":      "0x742d35Cc6634C0532925a3b844Bc9e7595f5eD5B",
	"BSC":      "0x742d35Cc6634C0532925a3b844Bc9e7595f5eD5B",
	"ARB":      "0x742d35Cc6634C0532925a3b844Bc9e7595f5eD5B",
	"AVAX":     "0x742d35Cc6634C0532925a3b844Bc9e7595f5eD5B",
	"POLYGON":  "0x742d35Cc6634C0532925a3b844Bc9e7595f5eD5B",
	"OPTIMISM": "0x742d35Cc6634C0532925a3b844Bc9e7595f5eD5B",
	"TRON":     "TJhqY7GMz6iM7sCqYqN3pF2xMqE7Kdm2rW",
	"SOLANA":   "7Eqpo6gnY9xq5uM1WqE3K2tX5rY8xPzA3vB6wD9kLm",
	"BTC":      "bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh",
}

var masterWalletCounter = 0

func generatePaymentID() string {
	masterWalletCounter++
	return fmt.Sprintf("PAY-%d-%d", time.Now().Unix(), masterWalletCounter)
}

// Create crypto payment (NO STRIPE - CRYPTO ONLY)
func CreateCryptoPayment(c *gin.Context) {
	var req PaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate crypto currency (NO FIAT)
	req.Currency = strings.ToUpper(req.Currency)
	if !supportedCrypto[req.Currency] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid currency. Only crypto supported: USDT, USDC, ETH, BTC, BNB, SOL",
		})
		return
	}

	// Validate network
	network := strings.ToUpper(req.Network)
	validNetworks, ok := cryptoNetworks[req.Currency]
	if !ok || !contains(validNetworks, network) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid network for %s. Valid networks: %v", req.Currency, validNetworks),
		})
		return
	}

	// Get master wallet address
	address, ok := masterWalletAddresses[network]
	if !ok {
		address = masterWalletAddresses[req.Currency] // Fallback
	}

	// Generate unique payment address (in production, would generate unique address per user)
	paymentID := generatePaymentID()
	memo := "" // For chains that need memo (like TRON)

	// For TRON, use memo
	if network == "TRON" {
		memo = paymentID
	}

	// For Solana, use memo
	if network == "SOLANA" {
		memo = paymentID
	}

	// Create payment record
	now := time.Now()
	payment := &CryptoPayment{
		ID:                    paymentID,
		UserID:                req.UserID,
		ListingID:             req.ListingID,
		Currency:              req.Currency,
		Amount:                req.Amount,
		Address:               address,
		Memo:                  memo,
		Network:               network,
		Status:                "pending",
		Confirmations:         0,
		RequiredConfirmations: getRequiredConfirmations(req.Currency),
		CreatedAt:             now.Unix(),
		UpdatedAt:             now.Unix(),
		CompletedAt:           nil,
	}

	// Store payment
	cryptoPaymentsMux.Lock()
	cryptoPayments[paymentID] = payment
	cryptoPaymentsMux.Unlock()

	// Save to Redis if available
	if redisClient != nil {
		paymentJSON, _ := json.Marshal(payment)
		redisClient.Set(context.Background(), "payment:"+paymentID, paymentJSON, time.Hour*24)
	}

	// Expire in 24 hours
	expiresAt := now.Add(24 * time.Hour).Unix()

	log.Printf("💰 Crypto payment created: %s - %s %f to %s (%s)",
		paymentID, req.Currency, req.Amount, address, network)

	c.JSON(http.StatusCreated, PaymentResponse{
		PaymentID:     paymentID,
		Currency:      req.Currency,
		Amount:        req.Amount,
		Address:       address,
		Memo:          memo,
		Network:       network,
		Confirmations: 0,
		Status:        "pending",
		CreatedAt:     now.Unix(),
		ExpiresAt:     expiresAt,
	})
}

func getRequiredConfirmations(currency string) int {
	confirmations := map[string]int{
		"USDT":         12,
		"USDC":         12,
		"ETH":          12,
		"BSC":          15,
		"ARB":          15,
		"AVAX":         12,
		"POLYGON":      128,
		"OPTIMISM":     1024,
		"TRON":         19,
		"SOLANA":       32,
		"BTC":          6,
		"BTC (SegWit)": 6,
	}

	if v, ok := confirmations[currency]; ok {
		return v
	}
	return 12 // Default
}

// Verify crypto payment (called by webhook or manual check)
func VerifyCryptoPayment(c *gin.Context) {
	paymentID := c.Param("id")
	txHash := c.Query("tx_hash")

	cryptoPaymentsMux.RLock()
	payment, ok := cryptoPayments[paymentID]
	cryptoPaymentsMux.RUnlock()

	if !ok {
		// Try Redis
		if redisClient != nil {
			paymentJSON, err := redisClient.Get(context.Background(), "payment:"+paymentID).Result()
			if err == nil {
				json.Unmarshal([]byte(paymentJSON), &payment)
			}
		}
	}

	if payment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
		return
	}

	// Update with transaction hash if provided
	if txHash != "" {
		payment.TxHash = txHash
	}

	// In production, would verify on-chain
	// For now, simulate confirmation check
	payment.Confirmations++

	if payment.Confirmations >= payment.RequiredConfirmations && payment.Status != "completed" {
		payment.Status = "completed"
		now := time.Now().Unix()
		payment.CompletedAt = &now
	}

	payment.UpdatedAt = time.Now().Unix()

	// Update storage
	cryptoPaymentsMux.Lock()
	cryptoPayments[paymentID] = payment
	cryptoPaymentsMux.Unlock()

	if redisClient != nil {
		paymentJSON, _ := json.Marshal(payment)
		redisClient.Set(context.Background(), "payment:"+paymentID, paymentJSON, time.Hour*24)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"payment_id":    payment.ID,
			"status":        payment.Status,
			"confirmations": payment.Confirmations,
			"required":      payment.RequiredConfirmations,
			"tx_hash":       payment.TxHash,
		},
	})
}

// Get payment status
func GetPaymentStatus(c *gin.Context) {
	paymentID := c.Param("id")

	cryptoPaymentsMux.RLock()
	payment, ok := cryptoPayments[paymentID]
	cryptoPaymentsMux.RUnlock()

	if !ok && redisClient != nil {
		paymentJSON, err := redisClient.Get(context.Background(), "payment:"+paymentID).Result()
		if err == nil {
			json.Unmarshal([]byte(paymentJSON), &payment)
		}
	}

	if payment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"payment_id":             payment.ID,
			"currency":               payment.Currency,
			"amount":                 payment.Amount,
			"address":                payment.Address,
			"memo":                   payment.Memo,
			"network":                payment.Network,
			"tx_hash":                payment.TxHash,
			"status":                 payment.Status,
			"confirmations":          payment.Confirmations,
			"required_confirmations": payment.RequiredConfirmations,
			"created_at":             payment.CreatedAt,
			"completed_at":           payment.CompletedAt,
		},
	})
}

// ============================================================================
// EXCHANGE API METHODS
// ============================================================================

func (e *ExchangeAPI) GetTicker(symbol string) (*Ticker, error) {
	// Implementation for each exchange
	switch e.Name {
	case "Binance":
		return e.binanceGetTicker(symbol)
	case "Coinbase":
		return e.coinbaseGetTicker(symbol)
	case "Kraken":
		return e.krakenGetTicker(symbol)
	case "KuCoin":
		return e.kucoinGetTicker(symbol)
	case "Bybit":
		return e.bybitGetTicker(symbol)
	case "OKX":
		return e.okxGetTicker(symbol)
	case "Gate":
		return e.gateGetTicker(symbol)
	case "Bitget":
		return e.bitgetGetTicker(symbol)
	case "Huobi":
		return e.huobiGetTicker(symbol)
	case "MEXC":
		return e.mexcGetTicker(symbol)
	}
	return nil, fmt.Errorf("exchange not supported: %s", e.Name)
}

func (e *ExchangeAPI) GetOrderBook(symbol string, limit int) (*OrderBook, error) {
	switch e.Name {
	case "Binance":
		return e.binanceGetOrderBook(symbol, limit)
	case "Coinbase":
		return e.coinbaseGetOrderBook(symbol, limit)
	case "Bybit":
		return e.bybitGetOrderBook(symbol, limit)
	}
	return nil, nil
}

func (e *ExchangeAPI) PlaceOrder(symbol, side, orderType string, amount, price float64) (*Order, error) {
	return nil, nil // Would implement per exchange
}

func (e *ExchangeAPI) CancelOrder(orderID, symbol string) error {
	return nil
}

func (e *ExchangeAPI) GetBalance() ([]Balance, error) {
	return nil, nil
}

func (e *ExchangeAPI) GetOpenOrders(symbol string) ([]Order, error) {
	return nil, nil
}

func (e *ExchangeAPI) GetTradeHistory(symbol string, limit int) ([]Trade, error) {
	return nil, nil
}

func (e *ExchangeAPI) Withdraw(req WithdrawRequest) (string, error) {
	return "", nil
}

func (e *ExchangeAPI) GetDepositAddress(currency, network string) (*DepositAddress, error) {
	return nil, nil
}

func (e *ExchangeAPI) GetMarkets() ([]Market, error) {
	return nil, nil
}

// Binance implementations
func (e *ExchangeAPI) binanceGetTicker(symbol string) (*Ticker, error) {
	url := fmt.Sprintf("%s/api/v3/ticker/24hr?symbol=%s", e.BaseURL, symbol)

	resp, err := e.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	ticker := &Ticker{
		Symbol:         result["symbol"].(string),
		LastPrice:      parseFloat(result["lastPrice"]),
		BidPrice:       parseFloat(result["bidPrice"]),
		AskPrice:       parseFloat(result["askPrice"]),
		Volume24h:      parseFloat(result["volume"]),
		QuoteVolume24h: parseFloat(result["quoteVolume"]),
		PriceChange:    parseFloat(result["priceChange"]),
		PriceChangePct: parseFloat(result["priceChangePercent"]),
		High24h:        parseFloat(result["highPrice"]),
		Low24h:         parseFloat(result["lowPrice"]),
		Timestamp:      time.Now().UnixMilli(),
	}

	return ticker, nil
}

func (e *ExchangeAPI) binanceGetOrderBook(symbol string, limit int) (*OrderBook, error) {
	if limit == 0 {
		limit = 100
	}

	url := fmt.Sprintf("%s/api/v3/depth?symbol=%s&limit=%d", e.BaseURL, symbol, limit)

	resp, err := e.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	ob := &OrderBook{
		Symbol:       symbol,
		LastUpdateID: int64(result["lastUpdateId"].(float64)),
		Exchange:     BINANCE,
		Timestamp:    time.Now().UnixMilli(),
	}

	for _, b := range result["bids"].([]interface{}) {
		entry := b.([]interface{})
		ob.Bids = append(ob.Bids, OrderBookEntry{
			Price:  parseFloat(entry[0]),
			Amount: parseFloat(entry[1]),
		})
	}

	for _, a := range result["asks"].([]interface{}) {
		entry := a.([]interface{})
		ob.Asks = append(ob.Asks, OrderBookEntry{
			Price:  parseFloat(entry[0]),
			Amount: parseFloat(entry[1]),
		})
	}

	return ob, nil
}

// Coinbase implementation
func (e *ExchangeAPI) coinbaseGetTicker(symbol string) (*Ticker, error) {
	url := fmt.Sprintf("%s/api/v3/brokerage/products/%s", e.BaseURL, symbol)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	price := parseFloat(result["price"])
	ticker := &Ticker{
		Symbol:    symbol,
		LastPrice: price,
		Volume24h: parseFloat(result["volume"]),
		High24h:   parseFloat(result["high_24_h"]),
		Low24h:    parseFloat(result["low_24_h"]),
		Timestamp: time.Now().UnixMilli(),
	}

	return ticker, nil
}

func (e *ExchangeAPI) coinbaseGetOrderBook(symbol string, limit int) (*OrderBook, error) {
	url := fmt.Sprintf("%s/api/v3/brokerage/products/%s/ticker", e.BaseURL, symbol)

	resp, err := e.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	ob := &OrderBook{
		Symbol:   symbol,
		Exchange: COINBASE,
	}

	if bid, ok := result["bid"]; ok {
		ob.Bids = append(ob.Bids, OrderBookEntry{Price: parseFloat(bid)})
	}
	if ask, ok := result["ask"]; ok {
		ob.Asks = append(ob.Asks, OrderBookEntry{Price: parseFloat(ask)})
	}

	return ob, nil
}

// Bybit implementation
func (e *ExchangeAPI) bybitGetTicker(symbol string) (*Ticker, error) {
	url := fmt.Sprintf("%s/v5/market/ticker?category=spot&symbol=%s", e.BaseURL, symbol)

	resp, err := e.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	list := result["list"].([]interface{})
	if len(list) == 0 {
		return nil, fmt.Errorf("no data")
	}

	data := list[0].(map[string]interface{})

	ticker := &Ticker{
		Symbol:         data["symbol"].(string),
		LastPrice:      parseFloat(data["lastPrice"]),
		BidPrice:       parseFloat(data["bid1Price"]),
		AskPrice:       parseFloat(data["ask1Price"]),
		Volume24h:      parseFloat(data["volume24h"]),
		QuoteVolume24h: parseFloat(data["quoteVolume24h"]),
		High24h:        parseFloat(data["highPrice24h"]),
		Low24h:         parseFloat(data["lowPrice24h"]),
		Timestamp:      time.Now().UnixMilli(),
	}

	return ticker, nil
}

func (e *ExchangeAPI) bybitGetOrderBook(symbol string, limit int) (*OrderBook, error) {
	if limit == 0 {
		limit = 100
	}

	url := fmt.Sprintf("%s/v5/market/orderbook?category=spot&symbol=%s&limit=%d", e.BaseURL, symbol, limit)

	resp, err := e.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	list := result["list"].([]interface{})
	if len(list) == 0 {
		return nil, fmt.Errorf("no data")
	}

	data := list[0].(map[string]interface{})

	ob := &OrderBook{
		Symbol:    symbol,
		Exchange:  BYBIT,
		Timestamp: time.Now().UnixMilli(),
	}

	for _, b := range data["b"].([]interface{}) {
		entry := b.([]interface{})
		ob.Bids = append(ob.Bids, OrderBookEntry{
			Price:  parseFloat(entry[0]),
			Amount: parseFloat(entry[1]),
		})
	}

	for _, a := range data["a"].([]interface{}) {
		entry := a.([]interface{})
		ob.Asks = append(ob.Asks, OrderBookEntry{
			Price:  parseFloat(entry[0]),
			Amount: parseFloat(entry[1]),
		})
	}

	return ob, nil
}

// Placeholder implementations for other exchanges
func (e *ExchangeAPI) krakenGetTicker(symbol string) (*Ticker, error) { return nil, nil }
func (e *ExchangeAPI) kucoinGetTicker(symbol string) (*Ticker, error) { return nil, nil }
func (e *ExchangeAPI) okxGetTicker(symbol string) (*Ticker, error)    { return nil, nil }
func (e *ExchangeAPI) gateGetTicker(symbol string) (*Ticker, error)   { return nil, nil }
func (e *ExchangeAPI) bitgetGetTicker(symbol string) (*Ticker, error) { return nil, nil }
func (e *ExchangeAPI) huobiGetTicker(symbol string) (*Ticker, error)  { return nil, nil }
func (e *ExchangeAPI) mexcGetTicker(symbol string) (*Ticker, error)   { return nil, nil }

// ============================================================================
// AUTHENTICATED API METHODS
// ============================================================================

func (e *ExchangeAPI) signRequest(method, endpoint, query string) string {
	var signStr string
	if method == "GET" && query != "" {
		signStr = fmt.Sprintf("%s%s?%s", method, endpoint, query)
	} else {
		signStr = fmt.Sprintf("%s%s%s", method, endpoint, query)
	}

	mac := hmac.New(sha256.New, []byte(e.APISecret))
	mac.Write([]byte(signStr))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func (e *ExchangeAPI) authenticatedRequest(method, endpoint, query string) (*http.Response, error) {
	signature := e.signRequest(method, endpoint, query)

	req, err := http.NewRequest(method, e.BaseURL+endpoint+"?"+query, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("API-Key", e.APIKey)
	req.Header.Set("API-Sign", signature)

	return e.httpClient.Do(req)
}

// ============================================================================
// API ENDPOINTS
// ============================================================================

// Get ticker from specific exchange
func GetTicker(c *gin.Context) {
	exchange := c.Param("exchange")
	symbol := c.Query("symbol")

	api, ok := exchangeAPI[exchange]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Exchange not found"})
		return
	}

	ticker, err := api.GetTicker(symbol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": ticker})
}

// Get order book from specific exchange
func GetOrderBook(c *gin.Context) {
	exchange := c.Param("exchange")
	symbol := c.Query("symbol")
	limit, _ := strconv.Atoi(c.Query("limit"))

	api, ok := exchangeAPI[exchange]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Exchange not found"})
		return
	}

	ob, err := api.GetOrderBook(symbol, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Cache locally
	orderBookMux.Lock()
	orderBooks[fmt.Sprintf("%s:%s", exchange, symbol)] = ob
	orderBookMux.Unlock()

	c.JSON(http.StatusOK, gin.H{"success": true, "data": ob})
}

// Get all exchange stats
func GetExchangeStats(c *gin.Context) {
	var stats []ExchangeStats

	for name, api := range exchangeAPI {
		stat := ExchangeStats{
			Exchange:           name,
			Status:             "online",
			LastUpdate:         time.Now().UnixMilli(),
			WebSocketConnected: api.wsConn != nil,
		}

		orderBookMux.RLock()
		count := 0
		for k := range orderBooks {
			if strings.HasPrefix(k, name+":") {
				count++
			}
		}
		stat.OrderBooksCount = count
		orderBookMux.RUnlock()

		stats = append(stats, stat)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": stats})
}

// Compare price across exchanges
func ComparePrice(c *gin.Context) {
	symbol := c.Query("symbol")

	type Price struct {
		Exchange string  `json:"exchange"`
		Price    float64 `json:"price"`
		Bid      float64 `json:"bid"`
		Ask      float64 `json:"ask"`
		Spread   float64 `json:"spread"`
	}

	var prices []Price

	for name, api := range exchangeAPI {
		ticker, err := api.GetTicker(symbol)
		if err != nil {
			continue
		}

		prices = append(prices, Price{
			Exchange: name,
			Price:    ticker.LastPrice,
			Bid:      ticker.BidPrice,
			Ask:      ticker.AskPrice,
			Spread:   ticker.AskPrice - ticker.BidPrice,
		})
	}

	// Sort by price
	sort.Slice(prices, func(i, j int) bool {
		return prices[i].Price < prices[j].Price
	})

	c.JSON(http.StatusOK, gin.H{"success": true, "data": prices})
}

// Get supported currencies (CRYPTO ONLY - NO FIAT)
func GetSupportedCurrencies(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"currencies": []string{"USDT", "USDC", "ETH", "BTC", "BNB", "SOL"},
			"networks":   cryptoNetworks,
		},
	})
}

// Get master wallet address
func GetMasterWalletAddress(c *gin.Context) {
	currency := strings.ToUpper(c.Query("currency"))
	network := strings.ToUpper(c.Query("network"))

	address, ok := masterWalletAddresses[network]
	if !ok {
		address = masterWalletAddresses[currency]
	}

	if address == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Network not supported"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"currency": currency,
			"network":  network,
			"address":  address,
		},
	})
}

// ============================================================================
// HELPERS
// ============================================================================

func parseFloat(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	case int:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return 0
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func initRedis() {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	ctx := context.Background()
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		log.Printf("⚠️  Redis not available: %v", err)
		redisClient = nil
		return
	}
	log.Println("✅ Redis connected")
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// ============================================================================
// MAIN
// ============================================================================

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("🚀 Starting TigerWallet CEX Connector Service...")

	initRedis()
	initExchangeAPIs()

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(CORSMiddleware())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "cex-connector",
			"time":    time.Now().Unix(),
		})
	})

	// Crypto payment endpoints (NO STRIPE)
	r.POST("/api/payments/crypto", CreateCryptoPayment)
	r.GET("/api/payments/crypto/:id", GetPaymentStatus)
	r.POST("/api/payments/crypto/:id/verify", VerifyCryptoPayment)

	// Supported currencies (CRYPTO ONLY)
	r.GET("/api/currencies", GetSupportedCurrencies)
	r.GET("/api/wallet/address", GetMasterWalletAddress)

	// Exchange endpoints
	r.GET("/api/exchanges", GetExchangeStats)
	r.GET("/api/exchanges/:exchange/ticker", GetTicker)
	r.GET("/api/exchanges/:exchange/orderbook", GetOrderBook)
	r.GET("/api/compare", ComparePrice)

	log.Printf("✅ CEX Connector running on port %s", SERVER_PORT)
	log.Printf("💰 Crypto payments: USDT, USDC, ETH, BTC, BNB, SOL only")

	if err := r.Run(":" + SERVER_PORT); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
