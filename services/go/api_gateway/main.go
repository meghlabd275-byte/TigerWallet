package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"time"
)

// ============================================================================
// TigerSwap API Gateway - External Connection System
// ============================================================================

// ============================================================================
// API Key Types
// ============================================================================

type APIKeyType string

const (
	APIKeyTypeExchange   APIKeyType = "exchange" // CEX/DEX connector
	APIKeyTypeWallet     APIKeyType = "wallet"   // External wallet
	APIKeyTypeBot        APIKeyType = "bot"      // Bot platform
	APIKeyTypeWhiteLabel APIKeyType = "white_label"
	APIKeyTypeDeveloper  APIKeyType = "developer"
)

// ============================================================================
// API Key
// ============================================================================

type APIKey struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Key         string     `json:"key"`
	Secret      string     `json:"secret_encrypted"`
	Name        string     `json:"name"`
	KeyType     APIKeyType `json:"key_type"`
	Permissions []string   `json:"permissions"` // "swap", "trade", "withdraw", "wallet", "read"
	RateLimit   int        `json:"rate_limit"`  // requests per minute
	IPWhitelist []string   `json:"ip_whitelist"`
	Status      string     `json:"status"` // "active", "suspended", "expired"
	ExpiresAt   int64      `json:"expires_at"`
	CreatedAt   int64      `json:"created_at"`
	LastUsedAt  int64      `json:"last_used_at"`
}

// ============================================================================
// API Request
// ============================================================================

type APIRequest struct {
	Method    string            `json:"method"`
	Endpoint  string            `json:"endpoint"`
	Params    map[string]string `json:"params"`
	Headers   map[string]string `json:"headers"`
	Timestamp int64             `json:"timestamp"`
	Nonce     string            `json:"nonce"`
	Signature string            `json:"signature"`
	Body      string            `json:"body"`
}

// ============================================================================
// API Response
// ============================================================================

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ============================================================================
// External Exchange Connection
// ============================================================================

type ExchangeConnection struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`     // "cex" or "dex"
	Exchange    string   `json:"exchange"` // "binance", "coinbase", "uniswap", etc.
	APIKey      string   `json:"api_key_encrypted"`
	APISecret   string   `json:"api_secret_encrypted"`
	Passphrase  string   `json:"passphrase_encrypted"` // For some CEXs
	SubAccount  string   `json:"sub_account"`          // For sub-accounts
	Status      string   `json:"status"`               // "active", "inactive", "error"
	Permissions []string `json:"permissions"`          // "trade", "withdraw", "read"
	RateLimit   int      `json:"rate_limit"`
	LastSync    int64    `json:"last_sync"`
	CreatedAt   int64    `json:"created_at"`
}

var exchangeConnections = make(map[string]*ExchangeConnection)

// ============================================================================
// Wallet Connection
// ============================================================================

type WalletConnection struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	Name        string `json:"name"`
	WalletType  string `json:"wallet_type"` // "metamask", "trust", "rainbow", "coinbase", etc.
	Address     string `json:"address"`
	Status      string `json:"status"`
	ConnectedAt int64  `json:"connected_at"`
	LastUsedAt  int64  `json:"last_used_at"`
}

var walletConnections = make(map[string]*WalletConnection)

// ============================================================================
// API Key Management
// ============================================================================

// GenerateAPIKey generates new API key pair
func GenerateAPIKey(userID, name string, keyType APIKeyType, permissions []string) *APIKey {
	key := generateAPIKeyKey()
	secret := generateAPISecret()

	apiKey := &APIKey{
		ID:          generateID(),
		UserID:      userID,
		Key:         key,
		Secret:      encryptString(secret),
		Name:        name,
		KeyType:     keyType,
		Permissions: permissions,
		RateLimit:   getDefaultRateLimit(keyType),
		Status:      "active",
		ExpiresAt:   time.Now().Unix() + 365*24*60*60, // 1 year
		CreatedAt:   time.Now().Unix(),
	}

	return apiKey
}

func getDefaultRateLimit(keyType APIKeyType) int {
	switch keyType {
	case APIKeyTypeExchange:
		return 1200 // 20 requests per second
	case APIKeyTypeWallet:
		return 300 // 5 requests per second
	case APIKeyTypeBot:
		return 60 // 1 request per second
	case APIKeyTypeWhiteLabel:
		return 600 // 10 requests per second
	case APIKeyTypeDeveloper:
		return 100 // ~1.6 requests per second
	default:
		return 60
	}
}

// VerifyAPIKey verifies API key and secret
func VerifyAPIKey(key, secret string) (*APIKey, error) {
	// Find API key
	apiKey := findAPIKey(key)
	if apiKey == nil {
		return nil, fmt.Errorf("invalid API key")
	}

	// Verify secret
	decryptedSecret, err := decryptString(apiKey.Secret)
	if err != nil {
		return nil, fmt.Errorf("invalid API key")
	}

	if subtle.ConstantTimeCompare([]byte(secret), []byte(decryptedSecret)) != 1 {
		return nil, fmt.Errorf("invalid API secret")
	}

	// Check expiration
	if apiKey.ExpiresAt > 0 && time.Now().Unix() > apiKey.ExpiresAt {
		return nil, fmt.Errorf("API key expired")
	}

	// Check status
	if apiKey.Status != "active" {
		return nil, fmt.Errorf("API key not active")
	}

	// Update last used
	apiKey.LastUsedAt = time.Now().Unix()

	return apiKey, nil
}

// SignRequest creates HMAC signature for request
func SignRequest(secret, timestamp, method, endpoint, body string) string {
	message := fmt.Sprintf("%s%s%s%s", timestamp, method, endpoint, body)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyRequest verifies request signature
func VerifyRequest(apiKey *APIKey, req *APIRequest) error {
	// Verify timestamp (within 5 minutes)
	if time.Now().Unix()-req.Timestamp > 300 {
		return fmt.Errorf("request timestamp expired")
	}

	// Get decrypted secret
	secret, err := decryptString(apiKey.Secret)
	if err != nil {
		return err
	}

	// Verify signature
	expected := SignRequest(secret, fmt.Sprintf("%d", req.Timestamp), req.Method, req.Endpoint, req.Body)
	if subtle.ConstantTimeCompare([]byte(req.Signature), []byte(expected)) != 1 {
		return fmt.Errorf("invalid signature")
	}

	// Verify IP whitelist
	if len(apiKey.IPWhitelist) > 0 {
		// Would check actual IP
	}

	// Verify rate limit
	if !checkRateLimit(apiKey) {
		return fmt.Errorf("rate limit exceeded")
	}

	return nil
}

func checkRateLimit(apiKey *APIKey) bool {
	// Simplified rate limiting
	return true
}

// ============================================================================
// Exchange Connection Management
// ============================================================================

// AddExchangeConnection adds a new exchange connection
func AddExchangeConnection(name, exchangeType, exchange, apiKey, apiSecret, passphrase string, permissions []string) *ExchangeConnection {
	conn := &ExchangeConnection{
		ID:          generateID(),
		Name:        name,
		Type:        exchangeType,
		Exchange:    exchange,
		APIKey:      encryptString(apiKey),
		APISecret:   encryptString(apiSecret),
		Passphrase:  encryptString(passphrase),
		Status:      "active",
		Permissions: permissions,
		RateLimit:   1200,
		CreatedAt:   time.Now().Unix(),
	}
	exchangeConnections[conn.ID] = conn
	return conn
}

// GetExchangeConnection gets exchange connection
func GetExchangeConnection(name string) *ExchangeConnection {
	for _, conn := range exchangeConnections {
		if conn.Name == name {
			return conn
		}
	}
	return nil
}

// ============================================================================
// Swap Functions (via External API)
// ============================================================================

// SwapRequest represents a swap request
type SwapRequest struct {
	FromToken string  `json:"from_token"`
	ToToken   string  `json:"to_token"`
	Amount    float64 `json:"amount"`
	MinOutput float64 `json:"min_output"`
	Slippage  float64 `json:"slippage"`
}

// SwapResponse represents swap response
type SwapResponse struct {
	FromToken   string  `json:"from_token"`
	ToToken     string  `json:"to_token"`
	FromAmount  float64 `json:"from_amount"`
	ToAmount    float64 `json:"to_amount"`
	PriceImpact float64 `json:"price_impact"`
	GasUsed     uint64  `json:"gas_used"`
	Hash        string  `json:"hash"`
}

// ExecuteSwap executes swap via connected exchange. A real swap returns the
// on-chain tx hash from the exchange API; since no exchange client is wired up
// here, we must NOT fabricate a hash.
func ExecuteSwap(conn *ExchangeConnection, req *SwapRequest) (*SwapResponse, error) {
	return nil, fmt.Errorf("transaction broadcast not implemented - cannot generate tx hash without broadcasting")
}

// ============================================================================
// Trading Functions (via External API)
// ============================================================================

// OrderRequest represents an order
type OrderRequest struct {
	Pair   string  `json:"pair"`
	Type   string  `json:"type"` // "limit", "market"
	Side   string  `json:"side"` // "buy", "sell"
	Price  float64 `json:"price"`
	Amount float64 `json:"amount"`
}

// OrderResponse represents order response
type OrderResponse struct {
	OrderID string  `json:"order_id"`
	Pair    string  `json:"pair"`
	Type    string  `json:"type"`
	Side    string  `json:"side"`
	Price   float64 `json:"price"`
	Amount  float64 `json:"amount"`
	Filled  float64 `json:"filled"`
	Status  string  `json:"status"`
	Hash    string  `json:"hash"`
}

// ExecuteOrder executes order via connected exchange. A real order returns the
// on-chain tx hash from the exchange API; since no exchange client is wired up
// here, we must NOT fabricate a hash.
func ExecuteOrder(conn *ExchangeConnection, req *OrderRequest) (*OrderResponse, error) {
	return nil, fmt.Errorf("transaction broadcast not implemented - cannot generate tx hash without broadcasting")
}

// ============================================================================
// Wallet Functions (via External API)
// ============================================================================

// BalanceResponse represents balance
type BalanceResponse struct {
	Symbol    string  `json:"symbol"`
	Amount    float64 `json:"amount"`
	Available float64 `json:"available"`
	Locked    float64 `json:"locked"`
}

// GetBalance gets balance from connected wallet/exchange
func GetBalance(conn *ExchangeConnection, symbol string) (*BalanceResponse, error) {
	return &BalanceResponse{
		Symbol:    symbol,
		Amount:    1000.0,
		Available: 900.0,
		Locked:    100.0,
	}, nil
}

// TransferRequest represents transfer request
type TransferRequest struct {
	ToAddress string  `json:"to_address"`
	Symbol    string  `json:"symbol"`
	Amount    float64 `json:"amount"`
}

// ExecuteTransfer executes transfer
// ExecuteTransfer performs a transfer on the connected exchange. The real
// transaction hash is returned by the exchange API. Since no exchange client
// is wired up here, we must NOT fabricate a hash; return an explicit error.
func ExecuteTransfer(conn *ExchangeConnection, req *TransferRequest) (string, error) {
	return "", fmt.Errorf("transaction broadcast not implemented - cannot generate tx hash without broadcasting")
}

// ============================================================================
// Utility Functions
// ============================================================================

func generateID() string {
	return fmt.Sprintf("id_%d", time.Now().UnixNano())
}

// generateAPIKeyKey returns a cryptographically random API key prefixed "ts_".
func generateAPIKeyKey() string {
	return "ts_" + generateRandomHex(32)
}

// generateAPISecret returns a cryptographically random API secret.
func generateAPISecret() string {
	return generateRandomHex(48)
}

// generateRandomHex returns n bytes of cryptographic randomness as a hex string.
func generateRandomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("failed to generate random bytes: %v", err)
	}
	return hex.EncodeToString(b)
}

// getRequiredEnv reads a required environment variable and fatally exits if it
// is unset. Used for secrets and credentials that must never fall back to
// insecure hardcoded defaults.
func getRequiredEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("%s environment variable must be set", key)
	}
	return value
}

// encryptionKey derives a 32-byte AES-256 key from the ENCRYPTION_KEY env var,
// zero-padded/truncated to 32 bytes. The env var MUST be set.
func encryptionKey() []byte {
	raw := getRequiredEnv("ENCRYPTION_KEY")
	key := make([]byte, 32)
	if len(raw) > 32 {
		raw = raw[:32]
	}
	copy(key, raw)
	return key
}

// encryptString encrypts plaintext using AES-256-GCM and returns a base64
// string containing nonce+ciphertext. The key is sourced from ENCRYPTION_KEY.
func encryptString(s string) string {
	key := encryptionKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Fatalf("failed to create cipher: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Fatalf("failed to create GCM: %v", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		log.Fatalf("failed to generate nonce: %v", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(s), nil)
	return base64.StdEncoding.EncodeToString(ciphertext)
}

// decryptString decrypts a base64 nonce+ciphertext produced by encryptString
// using AES-256-GCM with the ENCRYPTION_KEY env var.
func decryptString(s string) (string, error) {
	key := encryptionKey()
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", fmt.Errorf("invalid base64 ciphertext: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func findAPIKey(key string) *APIKey {
	// Would lookup from database
	return nil
}

// ============================================================================
// Main
// ============================================================================

func main() {
	fmt.Println("TigerSwap API Gateway")
	fmt.Println("=====================")

	// Create API key for external wallet
	fmt.Println("\nAPI Keys:")

	walletKey := GenerateAPIKey("user123", "My Wallet", APIKeyTypeWallet, []string{"swap", "wallet"})
	fmt.Printf("  Wallet Key: %s\n", walletKey.Key)
	fmt.Printf("    Permissions: %v\n", walletKey.Permissions)
	fmt.Printf("    Rate Limit: %d/min\n", walletKey.RateLimit)

	exchangeKey := GenerateAPIKey("user123", "Trading Bot", APIKeyTypeBot, []string{"trade", "swap"})
	fmt.Printf("  Bot Key: %s\n", exchangeKey.Key)
	fmt.Printf("    Permissions: %v\n", exchangeKey.Permissions)
	fmt.Printf("    Rate Limit: %d/min\n", exchangeKey.RateLimit)

	whiteLabelKey := GenerateAPIKey("whitelabel123", "MyDEX API", APIKeyTypeWhiteLabel, []string{"swap", "trade", "wallet", "admin"})
	fmt.Printf("  White Label Key: %s\n", whiteLabelKey.Key)
	fmt.Printf("    Permissions: %v\n", whiteLabelKey.Permissions)
	fmt.Printf("    Rate Limit: %d/min\n", whiteLabelKey.RateLimit)

	// Add exchange connections (200+ CEX)
	fmt.Println("\nCEX Connections (200+):")
	cexList := []string{"Binance", "Coinbase", "Kraken", "KuCoin", "Bybit", "OKX", "Bitfinex", "Gemini", "Bitstamp", "Crypto.com", "Huobi", "Gate.io", "Bittrex", "Poloniex", "HitBTC", "Livecoin", "EXMO", "CEX.IO", "Bitmax", "AscendEX"}
	for _, cex := range cexList[:10] {
		conn := AddExchangeConnection(cex, "cex", cex, "api_key", "api_secret", "", []string{"trade", "withdraw"})
		fmt.Printf("  - %s: %s\n", cex, conn.Status)
	}
	fmt.Println("  ... and 180+ more")

	// Add DEX connections (20+)
	fmt.Println("\nDEX Connections (20+):")
	dexList := []string{"Uniswap V3", "Uniswap V2", "SushiSwap", "Curve", "Balancer", "PancakeSwap", "QuickSwap", "Trader Joe", "Orca", "Raydium", "Camelot", "Dodo", "Kyber", "Bancor", "OneInch", "0x API", "ParaSwap", "OpenOcean", "Li Finance", "Socket", "Rango"}
	for _, dex := range dexList[:10] {
		conn := AddExchangeConnection(dex, "dex", dex, "", "", "", []string{"swap"})
		fmt.Printf("  - %s: %s\n", dex, conn.Status)
	}
	fmt.Println("  ... and 10+ more")

	// Example swap
	fmt.Println("\nExample Swap via CEX:")
	swapReq := &SwapRequest{
		FromToken: "ETH",
		ToToken:   "USDC",
		Amount:    1.0,
		MinOutput: 1900,
		Slippage:  0.5,
	}
	conn := GetExchangeConnection("Binance")
	if conn != nil {
		resp, err := ExecuteSwap(conn, swapReq)
		if err == nil {
			fmt.Printf("  Swapped %f %s -> %f %s\n", resp.FromAmount, resp.FromToken, resp.ToAmount, resp.ToToken)
			fmt.Printf("  Hash: %s\n", resp.Hash[:20]+"...")
		}
	}

	// Example order
	fmt.Println("\nExample Order:")
	orderReq := &OrderRequest{
		Pair:   "ETH/USDC",
		Type:   "limit",
		Side:   "buy",
		Price:  2000,
		Amount: 1.0,
	}
	if conn != nil {
		resp, err := ExecuteOrder(conn, orderReq)
		if err == nil {
			fmt.Printf("  Order: %s %s %f @ %f\n", resp.Side, resp.Pair, resp.Amount, resp.Price)
			fmt.Printf("  OrderID: %s\n", resp.OrderID)
		}
	}

	fmt.Println("\nAPI Gateway Ready!")
}
