/**
 * TigerWallet CEX Connectors - Kraken
 * Go Implementation for Kraken Exchange API
 */

package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// Kraken Connector
// ============================================================================

type KrakenConnector struct {
	apiKey      string
	apiSecret   string
	baseURL     string
	wsURL       string
	httpClient  *http.Client
	symbols     map[string]*KrakenPair
	orderCache  map[string]*Order
	mu          sync.RWMutex
	rateLimiter *RateLimiter
}

func NewKrakenConnector(apiKey, apiSecret string) *KrakenConnector {
	return &KrakenConnector{
		apiKey:      apiKey,
		apiSecret:   apiSecret,
		baseURL:     "https://api.kraken.com",
		wsURL:       "wss://ws.kraken.com",
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		symbols:     make(map[string]*KrakenPair),
		orderCache:  make(map[string]*Order),
		rateLimiter: NewRateLimiter(10, time.Second),
	}
}

func (k *KrakenConnector) Connect() error {
	// Fetch asset pairs
	pairs, err := k.GetAssetPairs()
	if err != nil {
		return fmt.Errorf("failed to fetch asset pairs: %v", err)
	}

	k.mu.Lock()
	for _, p := range pairs {
		k.symbols[p.Name] = p
	}
	k.mu.Unlock()

	return nil
}

func (k *KrakenConnector) Disconnect() {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.symbols = make(map[string]*KrakenPair)
}

func (k *KrakenConnector) IsConnected() bool {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return len(k.symbols) > 0
}

type KrakenPair struct {
	Name        string `json:"altname"`
	Base        string `json:"base"`
	Quote       string `json:"quote"`
	Pair        string `json:"pair"`
	Lot         string `json:"lot"`
	PairDecimal int    `json:"pair_decimals"`
	LotDecimal  int    `json:"lot_decimals"`
	MinLot      string `json:"minlot"`
	MinPrice    string `json:"minprice"`
	MaxPrice    string `json:"maxprice"`
	MinOrder    string `json:"minorder"`
	MaxOrder    string `json:"maxorder"`
	FeeVolume   string `json:"feevolume"`
}

func (k *KrakenConnector) GetAssetPairs() (map[string]*KrakenPair, error) {
	resp, err := k.httpClient.Get(k.baseURL + "/0/public/AssetPairs")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data struct {
		Error  []interface{}                     `json:"error"`
		Result map[string]map[string]interface{} `json:"result"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	pairs := make(map[string]*KrakenPair)
	for name, info := range data.Result {
		lot := "unit"
		if l, ok := info["lot"].(string); ok {
			lot = l
		}

		pairDecimal := 0
		if pd, ok := info["pair_decimals"].(float64); ok {
			pairDecimal = int(pd)
		}

		lotDecimal := 0
		if ld, ok := info["lot_decimals"].(float64); ok {
			lotDecimal = int(ld)
		}

		pairs[name] = &KrakenPair{
			Name:        name,
			Base:        info["base"].(string),
			Quote:       info["quote"].(string),
			Pair:        info["pair"].(string),
			Lot:         lot,
			PairDecimal: pairDecimal,
			LotDecimal:  lotDecimal,
			MinLot:      info["minlot"].(string),
			MinPrice:    info["minprice"].(string),
			MaxPrice:    info["maxprice"].(string),
			MinOrder:    info["minorder"].(string),
			MaxOrder:    info["maxorder"].(string),
			FeeVolume:   info["feevolume"].(string),
		}
	}

	return pairs, nil
}

func (k *KrakenConnector) GetSymbols() ([]Symbol, error) {
	k.mu.RLock()
	symbols := make([]Symbol, 0, len(k.symbols))
	for _, p := range k.symbols {
		symbols = append(symbols, Symbol{
			BaseAsset:  p.Base,
			QuoteAsset: p.Quote,
			Symbol:     p.Name,
			MinQty:     parseKrakenFloat(p.MinLot),
			MaxQty:     parseKrakenFloat(p.MaxOrder),
			StepSize:   1.0 / pow10(p.LotDecimal),
		})
	}
	k.mu.RUnlock()
	return symbols, nil
}

type Ticker struct {
	Pair      string
	Ask       []string // [price, wholeLotVolume, lotVolume]
	Bid       []string // [price, wholeLotVolume, lotVolume]
	Last      []string // [price, wholeLotVolume, lotVolume]
	Volume    []string // [today, last24h]
	VolumeAvg []string
	Trades    [][]string
	High      []string // [today, last24h]
	Low       []string // [today, last24h]
	Open      float64
	Today     float64
}

func (k *KrakenConnector) GetTicker(pair string) (*Ticker, error) {
	resp, err := k.httpClient.Get(k.baseURL + "/0/public/Ticker?pair=" + pair)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data struct {
		Error  []interface{}              `json:"error"`
		Result map[string]json.RawMessage `json:"result"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	for name, tickerData := range data.Result {
		var ticker Ticker
		if err := json.Unmarshal(tickerData, &ticker); err != nil {
			return nil, err
		}
		ticker.Pair = name
		return &ticker, nil
	}

	return nil, fmt.Errorf("ticker not found for pair: %s", pair)
}

type OrderBook struct {
	Pair string
	Bids [][]string // [price, volume, timestamp]
	Asks [][]string // [price, volume, timestamp]
}

func (k *KrakenConnector) GetOrderBook(pair string, count int) (*OrderBook, error) {
	if count == 0 {
		count = 100
	}

	resp, err := k.httpClient.Get(fmt.Sprintf("%s/0/public/Depth?pair=%s&count=%d", k.baseURL, pair, count))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data struct {
		Error  []interface{}              `json:"error"`
		Result map[string]json.RawMessage `json:"result"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	for name, depthData := range data.Result {
		var book OrderBook
		if err := json.Unmarshal(depthData, &book); err != nil {
			return nil, err
		}
		book.Pair = name
		return &book, nil
	}

	return nil, fmt.Errorf("orderbook not found for pair: %s", pair)
}

type OHLC struct {
	Timestamp int64
	Open      float64
	High      float64
	Low       float64
	Close     float64
	VWAP      float64
	Volume    float64
	Count     int
}

func (k *KrakenConnector) GetOHLC(pair, interval string) ([]OHLC, error) {
	resp, err := k.httpClient.Get(fmt.Sprintf("%s/public/OHLC?pair=%s&interval=%s", k.baseURL, pair, interval))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data struct {
		Error  []interface{}            `json:"error"`
		Result map[string][]interface{} `json:"result"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	for name, candles := range data.Result {
		if name == "last" {
			continue
		}

		ohlc := make([]OHLC, 0, len(candles))
		for _, c := range candles {
			cc := c.([]interface{})
			if len(cc) < 8 {
				continue
			}

			ts, _ := strconv.ParseInt(cc[0].(string), 10, 64)
			ohlc = append(ohlc, OHLC{
				Timestamp: ts,
				Open:      parseKrakenFloat(cc[1].(string)),
				High:      parseKrakenFloat(cc[2].(string)),
				Low:       parseKrakenFloat(cc[3].(string)),
				Close:     parseKrakenFloat(cc[4].(string)),
				VWAP:      parseKrakenFloat(cc[5].(string)),
				Volume:    parseKrakenFloat(cc[7].(string)),
			})
		}

		return ohlc, nil
	}

	return nil, fmt.Errorf("OHLC not found for pair: %s", pair)
}

// Trading

func (k *KrakenConnector) CreateOrder(order *Order) (*Order, error) {
	nonce := time.Now().UnixNano()

	pair := order.Symbol
	if _, ok := k.symbols[pair]; !ok {
		// Try alternate name
		for name, p := range k.symbols {
			if p.Pair == pair {
				pair = name
				break
			}
		}
	}

	postData := map[string]string{
		"pair":      pair,
		"type":      strings.ToLower(string(order.Side)),
		"ordertype": strings.ToLower(string(order.Type)),
		"volume":    strconv.FormatFloat(order.OriginalQty, 'f', -1, 64),
		"nonce":     strconv.FormatInt(nonce, 10),
	}

	if order.Type == LIMIT && order.Price > 0 {
		postData["price"] = strconv.FormatFloat(order.Price, 'f', -1, 64)
	}

	// Sign request
	path := "/0/private/AddOrder"
	signature := k.sign(path, postData, nonce)

	req, err := http.NewRequest("POST", k.baseURL+path, strings.NewReader(k.encodePostData(postData)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("API-Key", k.apiKey)
	req.Header.Set("API-Sign", signature)

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
		Error  []string `json:"error"`
		Result struct {
			TransactionIDs []string `json:"txid"`
			OrderDescr     struct {
				Order string `json:"order"`
			} `json:"orderDescr"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if len(result.Error) > 0 {
		return nil, fmt.Errorf("Kraken API error: %s", strings.Join(result.Error, ", "))
	}

	if len(result.Result.TransactionIDs) > 0 {
		order.OrderID = result.Result.TransactionIDs[0]
		order.Status = ORDER_NEW
	}

	order.CreateTime = time.Now().UnixMilli()

	k.mu.Lock()
	k.orderCache[order.OrderID] = order
	k.mu.Unlock()

	return order, nil
}

func (k *KrakenConnector) CancelOrder(orderID string) error {
	nonce := time.Now().UnixNano()

	postData := map[string]string{
		"txid":  orderID,
		"nonce": strconv.FormatInt(nonce, 10),
	}

	path := "/0/private/CancelOrder"
	signature := k.sign(path, postData, nonce)

	req, err := http.NewRequest("POST", k.baseURL+path, strings.NewReader(k.encodePostData(postData)))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("API-Key", k.apiKey)
	req.Header.Set("API-Sign", signature)

	resp, err := k.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to cancel order: %s", resp.Status)
	}

	return nil
}

func (k *KrakenConnector) GetOrder(orderID string) (*Order, error) {
	k.mu.RLock()
	if order, ok := k.orderCache[orderID]; ok {
		k.mu.RUnlock()
		return order, nil
	}
	k.mu.RUnlock()

	nonce := time.Now().UnixNano()

	postData := map[string]string{
		"txid":  orderID,
		"nonce": strconv.FormatInt(nonce, 10),
	}

	path := "/0/private/QueryOrders"
	signature := k.sign(path, postData, nonce)

	req, err := http.NewRequest("POST", k.baseURL+path, strings.NewReader(k.encodePostData(postData)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("API-Key", k.apiKey)
	req.Header.Set("API-Sign", signature)

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
		Error  []string       `json:"error"`
		Result map[string]any `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	for id, orderData := range result.Result {
		order := &Order{OrderID: id}

		if data, ok := orderData.(map[string]any); ok {
			if status, ok := data["status"].(string); ok {
				switch status {
				case "pending", "open":
					order.Status = ORDER_NEW
				case "closed":
					order.Status = ORDER_FILLED
				case "canceled":
					order.Status = ORDER_CANCELED
				}
			}
		}

		return order, nil
	}

	return nil, fmt.Errorf("order not found: %s", orderID)
}

func (k *KrakenConnector) GetOpenOrders() ([]Order, error) {
	nonce := time.Now().UnixNano()

	postData := map[string]string{
		"nonce": strconv.FormatInt(nonce, 10),
	}

	path := "/0/private/OpenOrders"
	signature := k.sign(path, postData, nonce)

	req, err := http.NewRequest("POST", k.baseURL+path, strings.NewReader(k.encodePostData(postData)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("API-Key", k.apiKey)
	req.Header.Set("API-Sign", signature)

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
		Error  []string       `json:"error"`
		Result map[string]any `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	orders := make([]Order, 0)
	for id := range result.Result {
		orders = append(orders, Order{OrderID: id, Status: ORDER_NEW})
	}

	return orders, nil
}

// Account

type KrakenBalance struct {
	Asset  string
	Free   float64
	Locked float64
}

func (k *KrakenConnector) GetBalance() (map[string]float64, error) {
	nonce := time.Now().UnixNano()

	postData := map[string]string{
		"nonce": strconv.FormatInt(nonce, 10),
	}

	path := "/0/private/Balance"
	signature := k.sign(path, postData, nonce)

	req, err := http.NewRequest("POST", k.baseURL+path, strings.NewReader(k.encodePostData(postData)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("API-Key", k.apiKey)
	req.Header.Set("API-Sign", signature)

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
		Error  []string          `json:"error"`
		Result map[string]string `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	balances := make(map[string]float64)
	for asset, balance := range result.Result {
		balances[asset] = parseKrakenFloat(balance)
	}

	return balances, nil
}

func (k *KrakenConnector) GetAccount() (*Account, error) {
	balances, err := k.GetBalance()
	if err != nil {
		return nil, err
	}

	account := &Account{
		Balances: make([]Balance, 0, len(balances)),
	}

	for asset, balance := range balances {
		account.Balances = append(account.Balances, Balance{
			Asset: asset,
			Total: balance,
			Free:  balance,
		})
	}

	return account, nil
}

func (k *KrakenConnector) GetAssetBalance(asset string) (*Balance, error) {
	balances, err := k.GetBalance()
	if err != nil {
		return nil, err
	}

	balance, ok := balances[asset]
	if !ok {
		return &Balance{Asset: asset}, nil
	}

	return &Balance{
		Asset: asset,
		Free:  balance,
		Total: balance,
	}, nil
}

// WebSocket

func (k *KrakenConnector) SubscribeChannel(channel string, callback func(interface{})) error {
	return nil
}

func (k *KrakenConnector) UnsubscribeChannel(channel string) error {
	return nil
}

// Helpers

func (k *KrakenConnector) sign(path string, postData map[string]string, nonce int64) string {
	// Create nonce + postdata string
	postDataString := strconv.FormatInt(nonce, 10) + k.encodePostData(postData)

	// SHA256(nonce + postdata)
	sha256Hash := sha256.New()
	sha256Hash.Write([]byte(postDataString))
	message := path + hex.EncodeToString(sha256Hash.Sum(nil))

	// HMAC-SHA512 with secret
	secret, err := base64.StdEncoding.DecodeString(k.apiSecret)
	if err != nil {
		return ""
	}
	hmacHash := hmac.New(sha512.New, secret)
	hmacHash.Write([]byte(message))

	return base64.StdEncoding.EncodeToString(hmacHash.Sum(nil))
}

func (k *KrakenConnector) encodePostData(postData map[string]string) string {
	var pairs []string
	for key, value := range postData {
		pairs = append(pairs, key+"="+value)
	}
	return strings.Join(pairs, "&")
}

func parseKrakenFloat(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func pow10(n int) float64 {
	result := big.NewFloat(1)
	for i := 0; i < n; i++ {
		result.Mul(result, big.NewFloat(10))
	}
	f, _ := result.Float64()
	return f
}

// Main

func main() {
	fmt.Println("TigerWallet - Kraken Connector")
	fmt.Println("================================")

	// Example: Create connector
	kraken := NewKrakenConnector("your_api_key", "your_api_secret")

	// Connect
	if err := kraken.Connect(); err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}

	fmt.Println("Connected to Kraken")

	// Get ticker
	ticker, err := kraken.GetTicker("XBTUSD")
	if err != nil {
		fmt.Printf("Failed to get ticker: %v\n", err)
	} else if len(ticker.Bid) > 0 {
		fmt.Printf("XBT/USD Bid: %s\n", ticker.Bid[0])
	}

	// Get balance
	balances, err := kraken.GetBalance()
	if err != nil {
		fmt.Printf("Failed to get balance: %v\n", err)
	} else {
		fmt.Printf("Assets: %d\n", len(balances))
	}
}
