/**
 * TigerWallet CEX Connectors - Bybit
 * Go Implementation for Bybit Exchange API
 */

package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Bybit Connector
type BybitConnector struct {
	apiKey      string
	apiSecret   string
	baseURL     string
	httpClient  *http.Client
	symbols     map[string]*BybitSymbol
	orderCache  map[string]*Order
	mu          sync.RWMutex
	rateLimiter *RateLimiter
}

func NewBybitConnector(apiKey, apiSecret string) *BybitConnector {
	return &BybitConnector{
		apiKey:      apiKey,
		apiSecret:   apiSecret,
		baseURL:     "https://api.bybit.com",
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		symbols:     make(map[string]*BybitSymbol),
		orderCache:  make(map[string]*Order),
		rateLimiter: NewRateLimiter(10, time.Second),
	}
}

type BybitSymbol struct {
	Name           string `json:"name"`
	Alias          string `json:"alias"`
	BaseCurrency   string `json:"baseCurrency"`
	QuoteCurrency  string `json:"quoteCurrency"`
	QuotePrecision int    `json:"quotePrecision"`
	Turnover       string `json:"turnover"`
	MinOrderQty    string `json:"minOrderQty"`
	MaxOrderQty    string `json:"maxOrderQty"`
	MinPrice       string `json:"minPrice"`
	MaxPrice       string `json:"maxPrice"`
}

func (b *BybitConnector) Connect() error {
	resp, err := b.httpClient.Get(b.baseURL + "/v5/market/instruments-info?category=spot")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var data struct {
		Result struct {
			List []BybitSymbol `json:"list"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return err
	}

	b.mu.Lock()
	for i := range data.Result.List {
		b.symbols[data.Result.List[i].Name] = &data.Result.List[i]
	}
	b.mu.Unlock()

	return nil
}

func (b *BybitConnector) GetSymbols() ([]Symbol, error) {
	b.mu.RLock()
	symbols := make([]Symbol, 0, len(b.symbols))
	for _, s := range b.symbols {
		symbols = append(symbols, Symbol{
			BaseAsset:  s.BaseCurrency,
			QuoteAsset: s.QuoteCurrency,
			Symbol:     s.Name,
			MinQty:     parseFloat(s.MinOrderQty),
			MaxQty:     parseFloat(s.MaxOrderQty),
		})
	}
	b.mu.RUnlock()
	return symbols, nil
}

type BybitTicker struct {
	Symbol      string `json:"symbol"`
	LastPrice   string `json:"lastPrice"`
	Bid1Price   string `json:"bid1Price"`
	Ask1Price   string `json:"ask1Price"`
	Volume24h   string `json:"volume24h"`
	Turnover24h string `json:"turnover24h"`
	Time        int64  `json:"time,string"`
}

func (b *BybitConnector) GetTicker(symbol string) (*BybitTicker, error) {
	resp, err := b.httpClient.Get(b.baseURL + "/v5/market/tickers?category=spot&symbol=" + symbol)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data struct {
		Result struct {
			List []BybitTicker `json:"list"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	if len(data.Result.List) > 0 {
		return &data.Result.List[0], nil
	}

	return nil, fmt.Errorf("ticker not found")
}

func (b *BybitConnector) CreateOrder(order *Order) (*Order, error) {
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)

	side := "Buy"
	if order.Side == SELL {
		side = "Sell"
	}

	orderType := "Market"
	if order.Type == LIMIT {
		orderType = "Limit"
	}

	params := map[string]interface{}{
		"category":    "spot",
		"symbol":      order.Symbol,
		"side":        side,
		"orderType":   orderType,
		"qty":         strconv.FormatFloat(order.OriginalQty, 'f', -1, 64),
		"orderLinkId": order.OrderID,
	}

	if order.Type == LIMIT && order.Price > 0 {
		params["price"] = strconv.FormatFloat(order.Price, 'f', -1, 64)
	}

	jsonBody, _ := json.Marshal(params)
	path := "/v5/order/create"

	req, err := http.NewRequest("POST", b.baseURL+path, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	b.signRequest(req, path, string(jsonBody), timestamp)

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Result struct {
			OrderID string `json:"orderId"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	order.OrderID = result.Result.OrderID
	order.Status = ORDER_NEW

	b.mu.Lock()
	b.orderCache[order.OrderID] = order
	b.mu.Unlock()

	return order, nil
}

func (b *BybitConnector) CancelOrder(orderID string) error {
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)

	params := map[string]interface{}{
		"category": "spot",
		"orderId":  orderID,
	}

	jsonBody, _ := json.Marshal(params)
	path := "/v5/order/cancel"

	req, err := http.NewRequest("POST", b.baseURL+path, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	b.signRequest(req, path, string(jsonBody), timestamp)

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (b *BybitConnector) GetBalance(coin string) (*Balance, error) {
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	path := "/v5/account/wallet-balance?accountType=SPOT"

	req, err := http.NewRequest("GET", b.baseURL+path, nil)
	if err != nil {
		return nil, err
	}

	b.signRequest(req, path, "", timestamp)

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Result struct {
			List []struct {
				Coins []struct {
					Coin   string `json:"coin"`
					Wallet string `json:"walletBalance"`
					Avail  string `json:"availableToWithdraw"`
				} `json:"coin"`
			} `json:"list"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	for _, account := range result.Result.List {
		for _, c := range account.Coins {
			if c.Coin == coin {
				wallet, _ := strconv.ParseFloat(c.Wallet, 64)
				avail, _ := strconv.ParseFloat(c.Avail, 64)
				return &Balance{
					Asset: coin,
					Free:  avail,
					Total: wallet,
				}, nil
			}
		}
	}

	return &Balance{Asset: coin}, nil
}

func (b *BybitConnector) signRequest(req *http.Request, path, body, timestamp string) {
	message := timestamp + req.Method + path + body
	hmac := hmac.New(sha256.New, []byte(b.apiSecret))
	hmac.Write([]byte(message))
	signature := fmt.Sprintf("%x", hmac.Sum(nil))

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-BAPI-API-KEY", b.apiKey)
	req.Header.Set("X-BAPI-SIGN", signature)
	req.Header.Set("X-BAPI-TIMESTAMP", timestamp)
	req.Header.Set("X-BAPI-RECV-WINDOW", "5000")
}

func (b *BybitConnector) IsConnected() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.symbols) > 0
}

func (b *BybitConnector) Disconnect() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.symbols = make(map[string]*BybitSymbol)
}

func main() {
	fmt.Println("TigerWallet - Bybit Connector")
	fmt.Println("==============================")

	connector := NewBybitConnector("your_api_key", "your_api_secret")

	if err := connector.Connect(); err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}

	fmt.Println("Connected to Bybit")

	ticker, err := connector.GetTicker("BTCUSDT")
	if err != nil {
		fmt.Printf("Failed to get ticker: %v\n", err)
	} else {
		fmt.Printf("BTCUSDT: %s\n", ticker.LastPrice)
	}
}
