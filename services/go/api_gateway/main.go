package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
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

// apiKeys is the in-process API-key store backing VerifyAPIKey.
var (
	apiKeys   = make(map[string]*APIKey)
	apiKeysMu sync.RWMutex
)

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
	apiKeysMu.Lock()
	apiKeys[apiKey.Key] = apiKey
	apiKeysMu.Unlock()

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

	// Verify IP whitelist (fail-closed when the key carries one and the
	// request does not originate from a whitelisted IP).
	if len(apiKey.IPWhitelist) > 0 {
		if req.Headers["X-Real-IP"] == "" || !stringSliceContains(apiKey.IPWhitelist, req.Headers["X-Real-IP"]) {
			return fmt.Errorf("request IP not whitelisted")
		}
	}

	// Verify rate limit
	if !checkRateLimit(apiKey) {
		return fmt.Errorf("rate limit exceeded")
	}

	return nil
}

func stringSliceContains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
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
	// OrderID is the real exchange order id for CEX fills.
	OrderID string `json:"order_id,omitempty"`
	// Hash is the on-chain transaction hash. Empty for CEX fills (a CEX fill
	// is off-chain); never fabricated.
	Hash string `json:"hash"`
}

// ExecuteSwap executes a real spot conversion on the connected CEX as a
// market order (from_token -> to_token). The response carries the real
// executed amounts and the exchange order id; a CEX fill has no on-chain
// hash, so Hash stays empty rather than being fabricated. DEX connections
// fail closed: DEX execution requires the wallet signing path, which lives in
// the wallet backend by design (keys never enter the gateway).
func ExecuteSwap(conn *ExchangeConnection, req *SwapRequest) (*SwapResponse, error) {
	if conn.Type == "dex" {
		return nil, fmt.Errorf("dex swap execution requires on-chain signing; use the wallet backend /swap/execute path (gateway holds no keys)")
	}
	apiKey, apiSecret, _, err := cexCredentials(conn)
	if err != nil {
		return nil, err
	}
	pair := strings.ToUpper(req.FromToken + req.ToToken)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var order map[string]any
	switch exchangeName(conn) {
	case "binance":
		// Sell `amount` of FromToken for ToToken at market.
		order, err = binancePlaceOrder(ctx, apiKey, apiSecret, pair, "sell", "market", req.Amount, 0)
	case "kraken":
		order, err = krakenPlaceOrder(ctx, apiKey, apiSecret, req.FromToken+"/"+req.ToToken, "sell", "market", req.Amount, 0)
	default:
		return nil, fmt.Errorf("exchange %q not supported for swap (supported: binance, kraken)", conn.Exchange)
	}
	if err != nil {
		return nil, err
	}
	resp := &SwapResponse{
		FromToken:  req.FromToken,
		ToToken:    req.ToToken,
		FromAmount: req.Amount,
	}
	if id, ok := order["orderId"]; ok {
		resp.OrderID = fmt.Sprint(id)
	}
	// Real executed amounts from the exchange fill report when present.
	if v, ok := order["executedQty"].(string); ok {
		resp.FromAmount, _ = strconv.ParseFloat(v, 64)
	}
	if v, ok := order["cummulativeQuoteQty"].(string); ok {
		resp.ToAmount, _ = strconv.ParseFloat(v, 64)
	}
	return resp, nil
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

// ExecuteOrder places a real limit/market order on the connected CEX and
// returns the real exchange order id and status. No on-chain hash exists for
// a CEX order, so Hash stays empty rather than being fabricated.
func ExecuteOrder(conn *ExchangeConnection, req *OrderRequest) (*OrderResponse, error) {
	if conn.Type == "dex" {
		return nil, fmt.Errorf("dex order execution requires on-chain signing; use the wallet backend (gateway holds no keys)")
	}
	apiKey, apiSecret, _, err := cexCredentials(conn)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var order map[string]any
	switch exchangeName(conn) {
	case "binance":
		order, err = binancePlaceOrder(ctx, apiKey, apiSecret, req.Pair, req.Side, req.Type, req.Amount, req.Price)
	case "kraken":
		order, err = krakenPlaceOrder(ctx, apiKey, apiSecret, req.Pair, req.Side, req.Type, req.Amount, req.Price)
	default:
		return nil, fmt.Errorf("exchange %q not supported for orders (supported: binance, kraken)", conn.Exchange)
	}
	if err != nil {
		return nil, err
	}
	resp := &OrderResponse{
		Pair:   req.Pair,
		Type:   req.Type,
		Side:   req.Side,
		Price:  req.Price,
		Amount: req.Amount,
		Status: "submitted",
	}
	if id, ok := order["orderId"]; ok {
		resp.OrderID = fmt.Sprint(id)
	}
	if v, ok := order["executedQty"].(string); ok {
		resp.Filled, _ = strconv.ParseFloat(v, 64)
	}
	if s, ok := order["status"].(string); ok {
		resp.Status = s
	}
	return resp, nil
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

// GetBalance reads the real account balance from the connected exchange API.
func GetBalance(conn *ExchangeConnection, symbol string) (*BalanceResponse, error) {
	if conn.Type == "dex" {
		return nil, fmt.Errorf("dex balances are on-chain; query the wallet backend /balance endpoint")
	}
	apiKey, apiSecret, _, err := cexCredentials(conn)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	switch exchangeName(conn) {
	case "binance":
		return binanceGetBalance(ctx, apiKey, apiSecret, symbol)
	case "kraken":
		return krakenGetBalance(ctx, apiKey, apiSecret, symbol)
	default:
		return nil, fmt.Errorf("exchange %q not supported for balances (supported: binance, kraken)", conn.Exchange)
	}
}

// TransferRequest represents transfer request
type TransferRequest struct {
	ToAddress string  `json:"to_address"`
	Symbol    string  `json:"symbol"`
	Amount    float64 `json:"amount"`
}

// ExecuteTransfer performs a real withdrawal on the connected exchange. The
// exchange assigns the on-chain hash asynchronously, so what is returned is
// the real exchange withdrawal reference (prefixed by venue), never a
// fabricated hash.
func ExecuteTransfer(conn *ExchangeConnection, req *TransferRequest) (string, error) {
	if conn.Type == "dex" {
		return "", fmt.Errorf("dex transfers are on-chain sends; use the wallet backend /send endpoint")
	}
	apiKey, apiSecret, _, err := cexCredentials(conn)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	switch exchangeName(conn) {
	case "binance":
		return binanceWithdraw(ctx, apiKey, apiSecret, req.Symbol, req.ToAddress, req.Amount)
	case "kraken":
		return krakenWithdraw(ctx, apiKey, apiSecret, req.Symbol, req.ToAddress, req.Amount)
	default:
		return "", fmt.Errorf("exchange %q not supported for withdrawals (supported: binance, kraken)", conn.Exchange)
	}
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
	apiKeysMu.RLock()
	defer apiKeysMu.RUnlock()
	return apiKeys[key]
}

// ============================================================================
// Main
// ============================================================================

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeAPIJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]string{"status": "healthy", "service": "api-gateway"}})
	})
	mux.HandleFunc("POST /v1/keys", handleCreateAPIKey)
	mux.HandleFunc("POST /v1/connections", handleAddConnection)
	mux.HandleFunc("GET /v1/connections", handleListConnections)
	mux.HandleFunc("POST /v1/swap", handleSwap)
	mux.HandleFunc("POST /v1/order", handleOrder)
	mux.HandleFunc("GET /v1/balance", handleBalance)
	mux.HandleFunc("POST /v1/transfer", handleTransfer)

	addr := ":" + envOr("GATEWAY_PORT", "8480")
	log.Printf("api-gateway listening on %s", addr)
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func writeAPIJSON(w http.ResponseWriter, status int, resp APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

func writeAPIError(w http.ResponseWriter, status int, code, msg string) {
	writeAPIJSON(w, status, APIResponse{Success: false, Error: &APIError{Code: code, Message: msg}})
}

// requireAdminToken gates administrative routes on the GATEWAY_ADMIN_TOKEN
// env var (fail-closed: unset token => 503, wrong token => 403).
func requireAdminToken(w http.ResponseWriter, r *http.Request) bool {
	want := os.Getenv("GATEWAY_ADMIN_TOKEN")
	if want == "" {
		writeAPIError(w, http.StatusServiceUnavailable, "config", "GATEWAY_ADMIN_TOKEN not configured")
		return false
	}
	if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Admin-Token")), []byte(want)) != 1 {
		writeAPIError(w, http.StatusForbidden, "forbidden", "invalid admin token")
		return false
	}
	return true
}

// authenticate verifies the caller's API key + secret pair.
func authenticate(w http.ResponseWriter, r *http.Request) *APIKey {
	key, secret := r.Header.Get("X-API-Key"), r.Header.Get("X-API-Secret")
	apiKey, err := VerifyAPIKey(key, secret)
	if err != nil {
		writeAPIError(w, http.StatusUnauthorized, "auth", "invalid API credentials")
		return nil
	}
	return apiKey
}

func hasPermission(k *APIKey, perm string) bool {
	for _, p := range k.Permissions {
		if p == perm || p == "admin" {
			return true
		}
	}
	return false
}

// handleCreateAPIKey issues a new API key pair (admin-gated).
func handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	if !requireAdminToken(w, r) {
		return
	}
	var req struct {
		UserID      string     `json:"user_id"`
		Name        string     `json:"name"`
		KeyType     APIKeyType `json:"key_type"`
		Permissions []string   `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" || req.Name == "" {
		writeAPIError(w, http.StatusBadRequest, "bad_request", "user_id and name are required")
		return
	}
	if req.KeyType == "" {
		req.KeyType = APIKeyTypeDeveloper
	}
	k := GenerateAPIKey(req.UserID, req.Name, req.KeyType, req.Permissions)
	// The plaintext secret is returned exactly once; only its AES-256-GCM
	// ciphertext is stored.
	secret, _ := decryptString(k.Secret)
	writeAPIJSON(w, http.StatusCreated, APIResponse{Success: true, Data: map[string]any{
		"id": k.ID, "key": k.Key, "secret": secret, "key_type": k.KeyType, "permissions": k.Permissions,
	}})
}

// handleAddConnection registers an exchange connection with real credentials
// (stored AES-256-GCM encrypted).
func handleAddConnection(w http.ResponseWriter, r *http.Request) {
	k := authenticate(w, r)
	if k == nil {
		return
	}
	if !hasPermission(k, "admin") {
		writeAPIError(w, http.StatusForbidden, "forbidden", "admin permission required")
		return
	}
	var req struct {
		Name        string   `json:"name"`
		Type        string   `json:"type"`
		Exchange    string   `json:"exchange"`
		APIKey      string   `json:"api_key"`
		APISecret   string   `json:"api_secret"`
		Passphrase  string   `json:"passphrase"`
		Permissions []string `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || req.Exchange == "" {
		writeAPIError(w, http.StatusBadRequest, "bad_request", "name and exchange are required")
		return
	}
	if req.Type == "" {
		req.Type = "cex"
	}
	conn := AddExchangeConnection(req.Name, req.Type, req.Exchange, req.APIKey, req.APISecret, req.Passphrase, req.Permissions)
	writeAPIJSON(w, http.StatusCreated, APIResponse{Success: true, Data: map[string]any{
		"id": conn.ID, "name": conn.Name, "exchange": conn.Exchange, "type": conn.Type, "status": conn.Status,
	}})
}

func handleListConnections(w http.ResponseWriter, r *http.Request) {
	k := authenticate(w, r)
	if k == nil {
		return
	}
	out := make([]map[string]any, 0, len(exchangeConnections))
	for _, c := range exchangeConnections {
		out = append(out, map[string]any{
			"id": c.ID, "name": c.Name, "exchange": c.Exchange, "type": c.Type, "status": c.Status,
		})
	}
	writeAPIJSON(w, http.StatusOK, APIResponse{Success: true, Data: out})
}

// resolveConnection finds a connection by id or name for swap/order/etc.
func resolveConnection(ref string) *ExchangeConnection {
	if c, ok := exchangeConnections[ref]; ok {
		return c
	}
	return GetExchangeConnection(ref)
}

func handleSwap(w http.ResponseWriter, r *http.Request) {
	k := authenticate(w, r)
	if k == nil {
		return
	}
	if !hasPermission(k, "swap") && !hasPermission(k, "trade") {
		writeAPIError(w, http.StatusForbidden, "forbidden", "swap permission required")
		return
	}
	var req struct {
		Connection string `json:"connection"`
		SwapRequest
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	conn := resolveConnection(req.Connection)
	if conn == nil {
		writeAPIError(w, http.StatusNotFound, "not_found", "connection not found")
		return
	}
	resp, err := ExecuteSwap(conn, &req.SwapRequest)
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, "upstream", err.Error())
		return
	}
	writeAPIJSON(w, http.StatusOK, APIResponse{Success: true, Data: resp})
}

func handleOrder(w http.ResponseWriter, r *http.Request) {
	k := authenticate(w, r)
	if k == nil {
		return
	}
	if !hasPermission(k, "trade") {
		writeAPIError(w, http.StatusForbidden, "forbidden", "trade permission required")
		return
	}
	var req struct {
		Connection string `json:"connection"`
		OrderRequest
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	conn := resolveConnection(req.Connection)
	if conn == nil {
		writeAPIError(w, http.StatusNotFound, "not_found", "connection not found")
		return
	}
	resp, err := ExecuteOrder(conn, &req.OrderRequest)
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, "upstream", err.Error())
		return
	}
	writeAPIJSON(w, http.StatusOK, APIResponse{Success: true, Data: resp})
}

func handleBalance(w http.ResponseWriter, r *http.Request) {
	k := authenticate(w, r)
	if k == nil {
		return
	}
	conn := resolveConnection(r.URL.Query().Get("connection"))
	if conn == nil {
		writeAPIError(w, http.StatusNotFound, "not_found", "connection not found")
		return
	}
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		writeAPIError(w, http.StatusBadRequest, "bad_request", "symbol required")
		return
	}
	resp, err := GetBalance(conn, symbol)
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, "upstream", err.Error())
		return
	}
	writeAPIJSON(w, http.StatusOK, APIResponse{Success: true, Data: resp})
}

func handleTransfer(w http.ResponseWriter, r *http.Request) {
	k := authenticate(w, r)
	if k == nil {
		return
	}
	if !hasPermission(k, "withdraw") {
		writeAPIError(w, http.StatusForbidden, "forbidden", "withdraw permission required")
		return
	}
	var req struct {
		Connection string `json:"connection"`
		TransferRequest
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	conn := resolveConnection(req.Connection)
	if conn == nil {
		writeAPIError(w, http.StatusNotFound, "not_found", "connection not found")
		return
	}
	ref, err := ExecuteTransfer(conn, &req.TransferRequest)
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, "upstream", err.Error())
		return
	}
	writeAPIJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]string{"withdrawal_ref": ref}})
}
