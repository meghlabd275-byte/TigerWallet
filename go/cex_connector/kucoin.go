/**
 * TigerWallet CEX Connectors - KuCoin
 * Go Implementation for KuCoin Exchange API
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

// KuCoin Connector
type KuCoinConnector struct {
	apiKey        string
	apiSecret     string
	apiPassphrase string
	baseURL       string
	httpClient    *http.Client
	symbols       map[string]*KuCoinPair
	orderCache    map[string]*Order
	mu            sync.RWMutex
	rateLimiter   *RateLimiter
}

func NewKuCoinConnector(apiKey, apiSecret, apiPassphrase string) *KuCoinConnector {
	return &KuCoinConnector{
		apiKey:        apiKey,
		apiSecret:     apiSecret,
		apiPassphrase: apiPassphrase,
		baseURL:       "https://api.kucoin.com",
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		symbols:       make(map[string]*KuCoinPair),
		orderCache:    make(map[string]*Order),
		rateLimiter:   NewRateLimiter(15, time.Second),
	}
}

type KuCoinPair struct {
	Symbol       string `json:"symbol"`
	Name         string `json:"name"`
	BaseCurrency string `json:"baseCurrency"`
	QuoteCurrency string `json:"quoteCurrency"`
	BaseMinSize  string `json:"baseMinSize"`
	BaseMaxSize  string `json:"baseMaxSize"`
	QuoteMinSize string `json:"quoteMinSize"`
	QuoteMaxSize string `json:"quoteMaxSize"`
	MinPrice     string `json:"minPrice"`
	MaxPrice     string `json:"maxPrice"`
}

func (k *KuCoinConnector) Connect() error {
	resp, err := k.httpClient.Get(k.baseURL + "/api/v1/symbols")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var data struct {
		Data []KuCoinPair `json:"data"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return err
	}

	k.mu.Lock()
	for i := range data.Data {
		k.symbols[data.Data[i].Symbol] = &data.Data[i]
	}
	k.mu.Unlock()

	return nil
}

func (k *KuCoinConnector) GetSymbols() ([]Symbol, error) {
	k.mu.RLock()
	symbols := make([]Symbol, 0, len(k.symbols))
	for _, p := range k.symbols {
		symbols = append(symbols, Symbol{
			BaseAsset:  p.BaseCurrency,
			QuoteAsset: p.QuoteCurrency,
			Symbol:     p.Symbol,
		})
	}
	k.mu.RUnlock()
	return symbols, nil
}

type KuCoinTicker struct {
	Symbol     string `json:"symbol"`
	Buy        string `json:"buy"`
	Sell       string `json:"sell"`
	Last       string `json:"last"`
	Volume     string `json:"vol"`
	VolumeValue string `json:"volValue"`
	Time       int64  `json:"time,string"`
}

func (k *KuCoinConnector) GetTicker(symbol string) (*KuCoinTicker, error) {
	resp, err := k.httpClient.Get(k.baseURL + "/api/v1/market/orderbook/level1?symbol=" + symbol)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data struct {
		Data KuCoinTicker `json:"data"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	return &data.Data, nil
}

func (k *KuCoinConnector) CreateOrder(order *Order) (*Order, error) {
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)

	side := "buy"
	if order.Side == SELL {
		side = "sell"
	}

	orderType := "market"
	if order.Type == LIMIT {
		orderType = "limit"
	}

	params := map[string]string{
		"clientOid":   order.OrderID,
		"type":        side,
		"side":        orderType,
		"symbol":      order.Symbol,
		"size":        strconv.FormatFloat(order.OriginalQty, 'f', -1, 64),
	}

	if order.Type == LIMIT && order.Price > 0 {
		params["price"] = strconv.FormatFloat(order.Price, 'f', -1, 64)
	}

	// Sign and make request
	path := "/api/v1/orders"
	jsonBody, _ := json.Marshal(params)

	req, err := http.NewRequest("POST", k.baseURL+path, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	k.signRequest(req, path, string(jsonBody), timestamp)

	resp, err := k.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			OrderID string `json:"orderId"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	order.OrderID = result.Data.OrderID
	order.Status = ORDER_NEW

	k.mu.Lock()
	k.orderCache[order.OrderID] = order
	k.mu.Unlock()

	return order, nil
}

func (k *KuCoinConnector) CancelOrder(orderID string) error {
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	path := "/api/v1/orders/" + orderID

	req, err := http.NewRequest("DELETE", k.baseURL+path, nil)
	if err != nil {
		return err
	}

	k.signRequest(req, path, "", timestamp)

	resp, err := k.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (k *KuCoinConnector) GetBalance(currency string) (*Balance, error) {
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	path := "/api/v1/accounts"

	req, err := http.NewRequest("GET", k.baseURL+path, nil)
	if err != nil {
		return nil, err
	}

	k.signRequest(req, path, "", timestamp)

	resp, err := k.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []struct {
			Currency string `json:"currency"`
			Balance  string `json:"balance"`
			Avail    string `json:"available"`
			Type     string `json:"type"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	for _, a := range result.Data {
		if a.Currency == currency && a.Type == "main" {
			balance, _ := strconv.ParseFloat(a.Balance, 64)
			avail, _ := strconv.ParseFloat(a.Avail, 64)
			return &Balance{
				Asset: currency,
				Free:  avail,
				Total: balance,
			}, nil
		}
	}

	return &Balance{Asset: currency}, nil
}

func (k *KuCoinConnector) signRequest(req *http.Request, path, body, timestamp string) {
	message := timestamp + req.Method + path + body
	hmac256 := hmac.New(sha256.New, []byte(k.apiSecret))
	hmac256.Write([]byte(message))
	signature := base64.StdEncoding.EncodeToString(hmac256.Sum(nil))

	passphrase, _ := base64.StdEncoding.DecodeString(k.apiPassphrase)
	hmacPass := hmac.New(sha256.New, passphrase)
	hmacPass.Write([]byte(k.apiPassphrase))
	passphraseHash := base64.StdEncoding.EncodeToString(hmacPass.Sum(nil))

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("KC-API-KEY", k.apiKey)
	req.Header.Set("KC-API-SIGN", signature)
	req.Header.Set("KC-API-TIMESTAMP", timestamp)
	req.Header.Set("KC-API-PASSPHRASE", passphraseHash)
	req.Header.Set("KC-API-KEY-VERSION", "2")
}

func (k *KuCoinConnector) IsConnected() bool {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return len(k.symbols) > 0
}

func (k *KuCoinConnector) Disconnect() {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.symbols = make(map[string]*KuCoinPair)
}

func main() {
	fmt.Println("TigerWallet - KuCoin Connector")
	fmt.Println("================================")

	connector := NewKuCoinConnector("your_api_key", "your_api_secret", "your_passphrase")

	if err := connector.Connect(); err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}

	fmt.Println("Connected to KuCoin")

	ticker, err := connector.GetTicker("BTC-USDT")
	if err != nil {
		fmt.Printf("Failed to get ticker: %v\n", err)
	} else {
		fmt.Printf("BTC-USDT: %s\n", ticker.Last)
	}
}
