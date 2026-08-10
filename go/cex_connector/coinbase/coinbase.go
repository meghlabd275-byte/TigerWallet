/**
 * TigerWallet CEX Connectors - Coinbase
 * Go Implementation for Coinbase Exchange API
 */

package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// Coinbase Connector
// ============================================================================

type CoinbaseConnector struct {
	apiKey      string
	apiSecret   string
	passphrase  string
	baseURL     string
	httpClient  *http.Client
	symbols     map[string]*Product
	orderCache  map[string]*Order
	mu          sync.RWMutex
	rateLimiter *RateLimiter
}

func NewCoinbaseConnector(apiKey, apiSecret, passphrase string) *CoinbaseConnector {
	return &CoinbaseConnector{
		apiKey:      apiKey,
		apiSecret:   apiSecret,
		passphrase:  passphrase,
		baseURL:     "https://api.coinbase.com",
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		symbols:     make(map[string]*Product),
		orderCache:  make(map[string]*Order),
		rateLimiter: NewRateLimiter(10, time.Second),
	}
}

func (c *CoinbaseConnector) Connect() error {
	// Fetch available products
	products, err := c.GetProducts()
	if err != nil {
		return fmt.Errorf("failed to fetch products: %v", err)
	}

	c.mu.Lock()
	for _, p := range products {
		c.symbols[p.ID] = p
	}
	c.mu.Unlock()

	return nil
}

func (c *CoinbaseConnector) Disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.symbols = make(map[string]*Product)
}

func (c *CoinbaseConnector) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.symbols) > 0
}

type Product struct {
	ID               string  `json:"id"`
	BaseCurrency     string  `json:"base_currency"`
	QuoteCurrency    string  `json:"quote_currency"`
	BaseMinSize      float64 `json:"base_min_size,string"`
	BaseMaxSize      float64 `json:"base_max_size,string"`
	QuoteMinSize     float64 `json:"quote_min_size,string"`
	QuoteIncrement   float64 `json:"quote_increment,string"`
	DisplayName      string  `json:"display_name"`
	Status           string  `json:"status"`
	StatusMessage    string  `json:"status_message"`
	MinMarketFunds   float64 `json:"min_market_funds,string"`
	MaxMarketFunds   float64 `json:"max_market_funds,string"`
	PostOnly         bool    `json:"post_only"`
	LimitOnly        bool    `json:"limit_only"`
	CancelOnly       bool    `json:"cancel_only"`
	TradingDisabled  bool    `json:"trading_disabled"`
	AuctionMode      bool    `json:"auction_mode"`
	AuctionStartTime *string `json:"auction_start_time"`
	AuctionEndTime   *string `json:"auction_end_time"`
}

func (c *CoinbaseConnector) GetProducts() ([]*Product, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/api/v3/brokerage/products", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data struct {
		Products []Product `json:"products"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	products := make([]*Product, len(data.Products))
	for i := range data.Products {
		products[i] = &data.Products[i]
	}

	return products, nil
}

type Symbol struct {
	BaseAsset  string
	QuoteAsset string
	Symbol     string
	MinPrice   float64
	MaxPrice   float64
	MinQty     float64
	MaxQty     float64
	StepSize   float64
	TickSize   float64
}

func (c *CoinbaseConnector) GetSymbols() ([]Symbol, error) {
	c.mu.RLock()
	symbols := make([]Symbol, 0, len(c.symbols))
	for _, p := range c.symbols {
		symbols = append(symbols, Symbol{
			BaseAsset:  p.BaseCurrency,
			QuoteAsset: p.QuoteCurrency,
			Symbol:     p.ID,
			MinQty:     p.BaseMinSize,
			MaxQty:     p.BaseMaxSize,
			StepSize:   p.QuoteIncrement,
		})
	}
	c.mu.RUnlock()
	return symbols, nil
}

type Ticker struct {
	TradeID   int64  `json:"trade_id,string"`
	Price     string `json:"price"`
	Size      string `json:"size"`
	Side      string `json:"side"`
	Time      string `json:"time"`
	ProductID string `json:"product_id"`
}

func (c *CoinbaseConnector) GetTicker(productID string) (*Ticker, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/api/v3/brokerage/products/"+productID+"/ticker", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var ticker Ticker
	if err := json.Unmarshal(body, &ticker); err != nil {
		return nil, err
	}

	return &ticker, nil
}

type OrderBookEntry struct {
	PriceLevel string `json:"price_level"`
	Proceeds   string `json:"proceeds"`
	Size       string `json:"size"`
	NumOrders  int    `json:"num_orders,string"`
}

type OrderBook struct {
	ProductID   string           `json:"product_id"`
	Bids        []OrderBookEntry `json:"bids"`
	Asks        []OrderBookEntry `json:"asks"`
	Time        string           `json:"time"`
	SequenceNum int64            `json:"sequence_num,string"`
}

func (c *CoinbaseConnector) GetOrderBook(productID string, level int) (*OrderBook, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/api/v3/brokerage/products/"+productID+"/book?level="+strconv.Itoa(level), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var orderBook OrderBook
	if err := json.Unmarshal(body, &orderBook); err != nil {
		return nil, err
	}

	return &orderBook, nil
}

type Candle struct {
	Start     float64
	High      float64
	Low       float64
	Open      float64
	Close     float64
	Volume    float64
	Timestamp time.Time
}

func (c *CoinbaseConnector) GetCandles(productID string, start, end time.Time, granularity string) ([]Candle, error) {
	req, err := http.NewRequest("GET",
		fmt.Sprintf("%s/api/v3/brokerage/products/%s/candles?start=%s&end=%s&granularity=%s",
			c.baseURL, productID, start.Format(time.RFC3339), end.Format(time.RFC3339), granularity), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data struct {
		Candles [][]float64 `json:"candles"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	candles := make([]Candle, len(data.Candles))
	for i, c := range data.Candles {
		if len(c) >= 6 {
			candles[i] = Candle{
				Start:     c[0],
				High:      c[1],
				Low:       c[2],
				Open:      c[3],
				Close:     c[4],
				Volume:    c[5],
				Timestamp: time.Unix(int64(c[0]), 0),
			}
		}
	}

	return candles, nil
}

// Trading

func (c *CoinbaseConnector) CreateOrder(order *Order) (*Order, error) {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	endpoint := c.baseURL + "/api/v3/brokerage/orders"

	side := "BUY"
	if order.Side == SELL {
		side = "SELL"
	}

	orderType := "MARKET"
	if order.Type == LIMIT {
		orderType = "LIMIT"
	}

	orderRequest := map[string]interface{}{
		"type":          orderType,
		"side":          side,
		"product_id":    order.Symbol,
		"size":          strconv.FormatFloat(order.OriginalQty, 'f', -1, 64),
		"post_only":     true,
		"time_in_force": "GTC",
	}

	if order.Type == LIMIT && order.Price > 0 {
		orderRequest["limit_price"] = strconv.FormatFloat(order.Price, 'f', -1, 64)
	}

	jsonBody, _ := json.Marshal(orderRequest)

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	// Sign request
	message := timestamp + "POST" + "/api/v3/brokerage/orders" + string(jsonBody)
	signature := c.sign(message)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("CB-ACCESS-KEY", c.apiKey)
	req.Header.Set("CB-ACCESS-SIGN", signature)
	req.Header.Set("CB-ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("CB-ACCESS-PASSPHRASE", c.passphrase)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response struct {
		Order OrderResponse `json:"order"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	order.OrderID = response.Order.OrderID
	order.Status = OrderStatus(response.Order.Status)
	order.CreateTime = time.Now().UnixMilli()

	c.mu.Lock()
	c.orderCache[order.OrderID] = order
	c.mu.Unlock()

	return order, nil
}

type OrderResponse struct {
	OrderID            string `json:"order_id"`
	ProductID          string `json:"product_id"`
	Side               string `json:"side"`
	OrderType          string `json:"order_type"`
	Status             string `json:"status"`
	TimeInForce        string `json:"time_in_force"`
	PostOnly           bool   `json:"post_only"`
	CreatedAt          string `json:"created_at"`
	FillFee            string `json:"fill_fee"`
	FilledSize         string `json:"filled_size"`
	RemainingSize      string `json:"remaining_size"`
	AverageFilledPrice string `json:"average_filled_price"`
}

func (c *CoinbaseConnector) CancelOrder(orderID string) error {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	endpoint := c.baseURL + "/api/v3/brokerage/orders/" + orderID

	req, err := http.NewRequest("DELETE", endpoint, nil)
	if err != nil {
		return err
	}

	message := timestamp + "DELETE" + "/api/v3/brokerage/orders/" + orderID
	signature := c.sign(message)

	req.Header.Set("CB-ACCESS-KEY", c.apiKey)
	req.Header.Set("CB-ACCESS-SIGN", signature)
	req.Header.Set("CB-ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("CB-ACCESS-PASSPHRASE", c.passphrase)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to cancel order: %s", resp.Status)
	}

	return nil
}

func (c *CoinbaseConnector) GetOrder(orderID string) (*Order, error) {
	c.mu.RLock()
	if order, ok := c.orderCache[orderID]; ok {
		c.mu.RUnlock()
		return order, nil
	}
	c.mu.RUnlock()

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	endpoint := c.baseURL + "/api/v3/brokerage/orders/" + orderID

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	message := timestamp + "GET" + "/api/v3/brokerage/orders/" + orderID
	signature := c.sign(message)

	req.Header.Set("CB-ACCESS-KEY", c.apiKey)
	req.Header.Set("CB-ACCESS-SIGN", signature)
	req.Header.Set("CB-ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("CB-ACCESS-PASSPHRASE", c.passphrase)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response struct {
		Order OrderResponse `json:"order"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	order := &Order{
		OrderID:     response.Order.OrderID,
		Symbol:      response.Order.ProductID,
		Side:        BUY,
		Status:      OrderStatus(response.Order.Status),
		Price:       0,
		ExecutedQty: 0,
		CreateTime:  time.Now().UnixMilli(),
	}

	return order, nil
}

func (c *CoinbaseConnector) GetOpenOrders() ([]Order, error) {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	endpoint := c.baseURL + "/api/v3/brokerage/orders?status=open"

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	message := timestamp + "GET" + "/api/v3/brokerage/orders?status=open"
	signature := c.sign(message)

	req.Header.Set("CB-ACCESS-KEY", c.apiKey)
	req.Header.Set("CB-ACCESS-SIGN", signature)
	req.Header.Set("CB-ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("CB-ACCESS-PASSPHRASE", c.passphrase)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response struct {
		Orders []OrderResponse `json:"orders"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	orders := make([]Order, len(response.Orders))
	for i, o := range response.Orders {
		side := BUY
		if o.Side == "SELL" {
			side = SELL
		}
		orders[i] = Order{
			OrderID:    o.OrderID,
			Symbol:     o.ProductID,
			Side:       side,
			Status:     OrderStatus(o.Status),
			CreateTime: time.Now().UnixMilli(),
		}
	}

	return orders, nil
}

// Account

type Account struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Balance   float64 `json:"balance,string"`
	Currency  string  `json:"currency"`
	Available float64 `json:"available,string"`
	Hold      float64 `json:"hold,string"`
}

func (c *CoinbaseConnector) GetAccounts() ([]Account, error) {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	endpoint := c.baseURL + "/api/v3/brokerage/accounts"

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	message := timestamp + "GET" + "/api/v3/brokerage/accounts"
	signature := c.sign(message)

	req.Header.Set("CB-ACCESS-KEY", c.apiKey)
	req.Header.Set("CB-ACCESS-SIGN", signature)
	req.Header.Set("CB-ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("CB-ACCESS-PASSPHRASE", c.passphrase)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response struct {
		Accounts []Account `json:"accounts"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return response.Accounts, nil
}

func (c *CoinbaseConnector) GetAccount(currency string) (*Account, error) {
	accounts, err := c.GetAccounts()
	if err != nil {
		return nil, err
	}

	for _, a := range accounts {
		if a.Currency == currency {
			return &a, nil
		}
	}

	return nil, fmt.Errorf("account not found for currency: %s", currency)
}

func (c *CoinbaseConnector) GetBalance(currency string) (*Balance, error) {
	account, err := c.GetAccount(currency)
	if err != nil {
		return nil, err
	}

	return &Balance{
		Asset: currency,
		Free:  account.Available,
		Total: account.Balance,
	}, nil
}

// WebSocket (placeholder - requires separate connection)

func (c *CoinbaseConnector) SubscribeChannel(channel string, callback func(interface{})) error {
	// Coinbase requires separate WebSocket connection
	// This would establish a WebSocket connection to:
	// wss://ws-feed.exchange.coinbase.com
	return nil
}

func (c *CoinbaseConnector) UnsubscribeChannel(channel string) error {
	return nil
}

func (c *CoinbaseConnector) sign(message string) string {
	hmac256 := hmac.New(sha256.New, []byte(c.apiSecret))
	hmac256.Write([]byte(message))
	signature := base64.StdEncoding.EncodeToString(hmac256.Sum(nil))
	return signature
}

// Helper for conversions

func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// Main function for testing

func main() {
	fmt.Println("TigerWallet - Coinbase Connector")
	fmt.Println("================================")

	// Example: Create connector (would use real API keys in production)
	coinbase := NewCoinbaseConnector("your_api_key", "your_api_secret", "your_passphrase")

	// Connect
	if err := coinbase.Connect(); err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}

	fmt.Println("Connected to Coinbase")

	// Get BTC-USD ticker
	ticker, err := coinbase.GetTicker("BTC-USD")
	if err != nil {
		fmt.Printf("Failed to get ticker: %v\n", err)
	} else {
		fmt.Printf("BTC-USD: %s\n", ticker.Price)
	}

	// Get accounts
	accounts, err := coinbase.GetAccounts()
	if err != nil {
		fmt.Printf("Failed to get accounts: %v\n", err)
	} else {
		fmt.Printf("Accounts: %d\n", len(accounts))
	}
}
