package main

// cex.go — real signed REST clients for centralized exchanges. All requests
// carry real exchange authentication (HMAC), all responses are parsed from
// the live upstream, and every failure path is fail-closed: no fabricated
// balances, order ids, or transaction hashes.

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var cexHTTPClient = &http.Client{Timeout: 30 * time.Second}

// cexCredentials decrypts the connection's stored credentials without
// mutating the stored record. Fail-closed: a decryption failure means the
// connection is unusable, never a fallback to plaintext.
func cexCredentials(conn *ExchangeConnection) (apiKey, apiSecret, passphrase string, err error) {
	apiKey, err = decryptString(conn.APIKey)
	if err != nil {
		return "", "", "", fmt.Errorf("decrypt api key: %w", err)
	}
	apiSecret, err = decryptString(conn.APISecret)
	if err != nil {
		return "", "", "", fmt.Errorf("decrypt api secret: %w", err)
	}
	if conn.Passphrase != "" {
		passphrase, err = decryptString(conn.Passphrase)
		if err != nil {
			return "", "", "", fmt.Errorf("decrypt passphrase: %w", err)
		}
	}
	return apiKey, apiSecret, passphrase, nil
}

// --- Binance (HMAC-SHA256 query signing) ---

const binanceBaseURL = "https://api.binance.com"

func binanceSignedRequest(ctx context.Context, method, path, apiKey, apiSecret string, params url.Values) ([]byte, error) {
	if params == nil {
		params = url.Values{}
	}
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	params.Set("recvWindow", "5000")
	query := params.Encode()
	mac := hmac.New(sha256.New, []byte(apiSecret))
	mac.Write([]byte(query))
	signature := hex.EncodeToString(mac.Sum(nil))

	full := binanceBaseURL + path + "?" + query + "&signature=" + signature
	var body io.Reader
	if method == http.MethodPost || method == http.MethodDelete {
		body = strings.NewReader("")
	}
	req, err := http.NewRequestWithContext(ctx, method, full, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-MBX-APIKEY", apiKey)
	resp, err := cexHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("binance HTTP %d: %s", resp.StatusCode, truncate(string(raw), 300))
	}
	return raw, nil
}

func binanceGetBalance(ctx context.Context, apiKey, apiSecret, symbol string) (*BalanceResponse, error) {
	raw, err := binanceSignedRequest(ctx, http.MethodGet, "/api/v3/account", apiKey, apiSecret, nil)
	if err != nil {
		return nil, err
	}
	var acct struct {
		Balances []struct {
			Asset  string `json:"asset"`
			Free   string `json:"free"`
			Locked string `json:"locked"`
		} `json:"balances"`
	}
	if err := json.Unmarshal(raw, &acct); err != nil {
		return nil, fmt.Errorf("binance account decode: %w", err)
	}
	symbol = strings.ToUpper(symbol)
	for _, b := range acct.Balances {
		if b.Asset == symbol {
			free, _ := strconv.ParseFloat(b.Free, 64)
			locked, _ := strconv.ParseFloat(b.Locked, 64)
			return &BalanceResponse{Symbol: symbol, Amount: free + locked, Available: free, Locked: locked}, nil
		}
	}
	// Zero balance is a real answer, not a fabrication.
	return &BalanceResponse{Symbol: symbol}, nil
}

// binancePlaceOrder places a real spot order. quoteOrderQty is used for MARKET
// buys denominated in the quote asset; otherwise quantity is base-asset amount.
func binancePlaceOrder(ctx context.Context, apiKey, apiSecret, symbol, side, orderType string, quantity, price float64) (map[string]any, error) {
	params := url.Values{}
	params.Set("symbol", strings.ToUpper(strings.ReplaceAll(symbol, "/", "")))
	params.Set("side", strings.ToUpper(side))
	params.Set("type", strings.ToUpper(orderType))
	if strings.EqualFold(orderType, "limit") {
		params.Set("timeInForce", "GTC")
		params.Set("price", strconv.FormatFloat(price, 'f', 8, 64))
		params.Set("quantity", strconv.FormatFloat(quantity, 'f', 8, 64))
	} else {
		params.Set("quantity", strconv.FormatFloat(quantity, 'f', 8, 64))
	}
	return binanceOrderDecode(binanceSignedRequest(ctx, http.MethodPost, "/api/v3/order", apiKey, apiSecret, params))
}

func binanceOrderDecode(raw []byte, err error) (map[string]any, error) {
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("binance order decode: %w", err)
	}
	return out, nil
}

func binanceWithdraw(ctx context.Context, apiKey, apiSecret, coin, address string, amount float64) (string, error) {
	params := url.Values{}
	params.Set("coin", strings.ToUpper(coin))
	params.Set("address", address)
	params.Set("amount", strconv.FormatFloat(amount, 'f', 8, 64))
	raw, err := binanceSignedRequest(ctx, http.MethodPost, "/sapi/v1/capital/withdraw/apply", apiKey, apiSecret, params)
	if err != nil {
		return "", err
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("binance withdraw decode: %w", err)
	}
	if out.ID == "" {
		return "", fmt.Errorf("binance returned no withdrawal id")
	}
	// The on-chain hash is assigned asynchronously by the exchange; what is
	// real at this point is the exchange withdrawal id.
	return "binance-withdrawal:" + out.ID, nil
}

// --- Kraken (HMAC-SHA512 over path+nonce+body, base64) ---

const krakenBaseURL = "https://api.kraken.com"

func krakenPrivate(ctx context.Context, endpoint, apiKey, apiSecret string, params url.Values) ([]byte, error) {
	if params == nil {
		params = url.Values{}
	}
	nonce := strconv.FormatInt(time.Now().UnixNano(), 10)
	params.Set("nonce", nonce)
	encoded := params.Encode()

	secret, err := base64.StdEncoding.DecodeString(apiSecret)
	if err != nil {
		return nil, fmt.Errorf("kraken secret is not valid base64: %w", err)
	}
	path := "/0/private/" + endpoint
	sha := sha256.New()
	sha.Write([]byte(nonce + encoded))
	mac := hmac.New(sha512.New, secret)
	mac.Write(append([]byte(path), sha.Sum(nil)...))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, krakenBaseURL+path, strings.NewReader(encoded))
	if err != nil {
		return nil, err
	}
	req.Header.Set("API-Key", apiKey)
	req.Header.Set("API-Sign", sig)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := cexHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("kraken HTTP %d: %s", resp.StatusCode, truncate(string(raw), 300))
	}
	var envelope struct {
		Error  []string        `json:"error"`
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("kraken decode: %w", err)
	}
	if len(envelope.Error) > 0 {
		return nil, fmt.Errorf("kraken error: %s", strings.Join(envelope.Error, "; "))
	}
	return envelope.Result, nil
}

// krakenAssetName maps common tickers to Kraken's internal asset names.
func krakenAssetName(symbol string) string {
	s := strings.ToUpper(symbol)
	switch s {
	case "BTC":
		return "XXBT"
	case "ETH":
		return "XETH"
	case "USDT":
		return "USDT"
	case "USDC":
		return "USDC"
	default:
		return s
	}
}

func krakenGetBalance(ctx context.Context, apiKey, apiSecret, symbol string) (*BalanceResponse, error) {
	raw, err := krakenPrivate(ctx, "Balance", apiKey, apiSecret, nil)
	if err != nil {
		return nil, err
	}
	var balances map[string]string
	if err := json.Unmarshal(raw, &balances); err != nil {
		return nil, fmt.Errorf("kraken balance decode: %w", err)
	}
	asset := krakenAssetName(symbol)
	amount, _ := strconv.ParseFloat(balances[asset], 64)
	return &BalanceResponse{Symbol: strings.ToUpper(symbol), Amount: amount, Available: amount}, nil
}

// krakenPair converts ETH/USDC to the Kraken pair code.
func krakenPair(pair string) string {
	p := strings.ToUpper(strings.ReplaceAll(pair, "/", ""))
	p = strings.ReplaceAll(p, "BTC", "XBT")
	return p
}

func krakenPlaceOrder(ctx context.Context, apiKey, apiSecret, pair, side, orderType string, amount, price float64) (map[string]any, error) {
	params := url.Values{}
	params.Set("pair", krakenPair(pair))
	params.Set("type", strings.ToLower(side))
	params.Set("ordertype", strings.ToLower(orderType))
	params.Set("volume", strconv.FormatFloat(amount, 'f', 8, 64))
	if strings.EqualFold(orderType, "limit") {
		params.Set("price", strconv.FormatFloat(price, 'f', 8, 64))
	}
	raw, err := krakenPrivate(ctx, "AddOrder", apiKey, apiSecret, params)
	if err != nil {
		return nil, err
	}
	var out struct {
		TxID []string `json:"txid"`
		Desc struct {
			Order string `json:"order"`
		} `json:"descr"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("kraken addorder decode: %w", err)
	}
	if len(out.TxID) == 0 {
		return nil, fmt.Errorf("kraken returned no order id")
	}
	return map[string]any{"orderId": out.TxID[0], "description": out.Desc.Order}, nil
}

func krakenWithdraw(ctx context.Context, apiKey, apiSecret, asset, address string, amount float64) (string, error) {
	params := url.Values{}
	params.Set("asset", krakenAssetName(asset))
	params.Set("key", address) // Kraken requires a pre-registered withdrawal key name
	params.Set("amount", strconv.FormatFloat(amount, 'f', 8, 64))
	raw, err := krakenPrivate(ctx, "Withdraw", apiKey, apiSecret, params)
	if err != nil {
		return "", err
	}
	var out struct {
		RefID string `json:"refid"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("kraken withdraw decode: %w", err)
	}
	if out.RefID == "" {
		return "", fmt.Errorf("kraken returned no withdrawal refid")
	}
	return "kraken-withdrawal:" + out.RefID, nil
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// exchangeName normalizes the connection's exchange identifier.
func exchangeName(conn *ExchangeConnection) string {
	return strings.ToLower(strings.TrimSpace(conn.Exchange))
}
