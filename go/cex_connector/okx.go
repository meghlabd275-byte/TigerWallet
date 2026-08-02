/**
 * TigerWallet CEX Connectors - OKX
 * Go Implementation for OKX Exchange API
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

// OKX Connector
type OKXConnector struct {
	apiKey     string
	apiSecret  string
	passphrase string
	baseURL    string
	httpClient *http.Client
	symbols    map[string]*OKXSymbol
	orderCache map[string]*Order
	mu         sync.RWMutex
	rateLimiter *RateLimiter
}

func NewOKXConnector(apiKey, apiSecret, passphrase string) *OKXConnector {
	return &OKXConnector{
		apiKey:     apiKey,
		apiSecret:  apiSecret,
		passphrase: passphrase,
		baseURL:    "https://www.okx.com",
		httpClient: &http.Client{Timeout: 30 * time.Second},
		symbols:    make(map[string]*OKXSymbol),
		orderCache: make(map[string]*Order),
		rateLimiter: NewRateLimiter(10, time.Second),
	}
}

type OKXSymbol struct {
	InstID  string `json:"instId"`
	InstType string `json:"instType"`
	BaseCcy string `json:"baseCcy"`
	QuoteCcy string `json:"quoteCcy"`
	MinSize string `json:"minSz"`
	MaxSize string `json:"maxSz"`
	MinPrice string `json:"minPx"`
	MaxPrice string `json:"maxPx"`
}

func (o *OKXConnector) Connect() error {
	resp, err := o.httpClient.Get(o.baseURL + "/api/v5/public/instruments?instType=SPOT")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var data struct {
		Data []OKXSymbol `json:"data"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return err
	}

	o.mu.Lock()
	for i := range data.Data {
		o.symbols[data.Data[i].InstID] = &data.Data[i]
	}
	o.mu.Unlock()

	return nil
}

func (o *OKXConnector) GetSymbols() ([]Symbol, error) {
	o.mu.RLock()
	symbols := make([]Symbol, 0, len(o.symbols))
	for _, s := range o.symbols {
		symbols = append(symbols, Symbol{
			BaseAsset:  s.BaseCcy,
			QuoteAsset: s.QuoteCcy,
			Symbol:     s.InstID,
			MinQty:     parseFloat(s.MinSize),
			MaxQty:     parseFloat(s.MaxSize),
		})
	}
	o.mu.RUnlock()
	return symbols, nil
}

type OKXTicker struct {
	InstID    string `json:"instId"`
	Last      string `json:"last"`
	Bid1Price string `json:"bidPx"`
	Ask1Price string `json:"askPx"`
	Vol24h    string `json:"vol24h"`
	Time      int64  `json:"ts,string"`
}

func (o *OKXConnector) GetTicker(instID string) (*OKXTicker, error) {
	resp, err := o.httpClient.Get(o.baseURL + "/api/v5/market/ticker?instId=" + instID)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data struct {
		Data []OKXTicker `json:"data"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	if len(data.Data) > 0 {
		return &data.Data[0], nil
	}

	return nil, fmt.Errorf("ticker not found")
}

func (o *OKXConnector) CreateOrder(order *Order) (*Order, error) {
	timestamp := time.Now().UTC().Format(time.RFC3339)

	side := "buy"
	if order.Side == SELL {
		side = "sell"
	}

	ordType := "market"
	if order.Type == LIMIT {
		ordType = "limit"
	}

	params := []interface{}{
		map[string]interface{}{
			"instId":  order.Symbol,
			"tdMode":  "cash",
			"side":    side,
			"ordType": ordType,
			"sz":      strconv.FormatFloat(order.OriginalQty, 'f', -1, 64),
		},
	}

	if order.Type == LIMIT && order.Price > 0 {
		params[0].(map[string]interface{})["px"] = strconv.FormatFloat(order.Price, 'f', -1, 64)
	}

	jsonBody, _ := json.Marshal(params)
	path := "/api/v5/trade/order"

	req, err := http.NewRequest("POST", o.baseURL+path, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	o.signRequest(req, path, string(jsonBody), timestamp)

	resp, err := o.httpClient.Do(req)
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
			OrdID string `json:"ordId"`
			ClOrdID string `json:"clOrdId"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if len(result.Data) > 0 {
		order.OrderID = result.Data[0].OrdID
	}

	order.Status = ORDER_NEW

	o.mu.Lock()
	o.orderCache[order.OrderID] = order
	o.mu.Unlock()

	return order, nil
}

func (o *OKXConnector) CancelOrder(orderID, instID string) error {
	timestamp := time.Now().UTC().Format(time.RFC3339)

	params := []interface{}{
		map[string]interface{}{
			"instId": instID,
			"ordId":  orderID,
		},
	}

	jsonBody, _ := json.Marshal(params)
	path := "/api/v5/trade/cancel-order"

	req, err := http.NewRequest("POST", o.baseURL+path, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	o.signRequest(req, path, string(jsonBody), timestamp)

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (o *OKXConnector) GetBalance(ccy string) (*Balance, error) {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	path := "/api/v5/account/balance?ccy=" + ccy

	req, err := http.NewRequest("GET", o.baseURL+path, nil)
	if err != nil {
		return nil, err
	}

	o.signRequest(req, path, "", timestamp)

	resp, err := o.httpClient.Do(req)
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
			Details []struct {
				Ccy   string `json:"ccy"`
				Avail string `json:"availBal"`
				Frozen string `json:"frozenBal"`
			} `json:"details"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	for _, account := range result.Data {
		for _, detail := range account.Details {
			if detail.Ccy == ccy {
				avail, _ := strconv.ParseFloat(detail.Avail, 64)
				frozen, _ := strconv.ParseFloat(detail.Frozen, 64)
				return &Balance{
					Asset: ccy,
					Free:  avail,
					Total: avail + frozen,
				}, nil
			}
		}
	}

	return &Balance{Asset: ccy}, nil
}

func (o *OKXConnector) signRequest(req *http.Request, path, body, timestamp string) {
	var prehash string
	if body != "" {
		prehash = timestamp + req.Method + path + body
	} else {
		prehash = timestamp + req.Method + path
	}

	hmac256 := hmac.New(sha256.New, []byte(o.apiSecret))
	hmac256.Write([]byte(prehash))
	signature := base64.StdEncoding.EncodeToString(hmac256.Sum(nil))

	passHmac := hmac.New(sha256.New, []byte(o.passphrase))
	passHmac.Write([]byte(o.passphrase))
	passphraseHash := base64.StdEncoding.EncodeToString(passHmac.Sum(nil))

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("OK-ACCESS-KEY", o.apiKey)
	req.Header.Set("OK-ACCESS-SIGN", signature)
	req.Header.Set("OK-ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("OK-ACCESS-PASSPHRASE", passphraseHash)
}

func (o *OKXConnector) IsConnected() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return len(o.symbols) > 0
}

func (o *OKXConnector) Disconnect() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.symbols = make(map[string]*OKXSymbol)
}

func main() {
	fmt.Println("TigerWallet - OKX Connector")
	fmt.Println("============================")

	connector := NewOKXConnector("your_api_key", "your_api_secret", "your_passphrase")

	if err := connector.Connect(); err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}

	fmt.Println("Connected to OKX")

	ticker, err := connector.GetTicker("BTC-USDT")
	if err != nil {
		fmt.Printf("Failed to get ticker: %v\n", err)
	} else {
		fmt.Printf("BTC-USDT: %s\n", ticker.Last)
	}
}
