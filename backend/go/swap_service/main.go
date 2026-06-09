package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Swap Service - Cross-Chain DEX Aggregator & Swap Engine
// ============================================================================

// Configuration
const (
	SwapServicePort     = 8082
	MaxRouteHops        = 3
	PriceUpdateInterval = 5 * time.Second
	DefaultSlippage     = 1 // 1%
	MaxSlippage         = 50 // 50%
)

// ============================================================================
// Types
// ============================================================================

// Token represents a tradable token
type Token struct {
	ID           string `json:"id"`
	ChainID      int    `json:"chain_id"`
	Address     string `json:"address"` // Empty for native
	Symbol      string `json:"symbol"`
	Name        string `json:"name"`
	Decimals    int    `json:"decimals"`
	Standard    string `json:"standard"` // native, erc20, spl, trc20
	PriceUSD    float64 `json:"price_usd"`
	Volume24h   float64 `json:"volume_24h"`
	Liquidity   float64 `json:"liquidity"`
	ContractType string `json:"contract_type"`
}

// Pool represents a liquidity pool
type Pool struct {
	ID              string            `json:"id"`
	ChainID        int               `json:"chain_id"`
	DEX            string            `json:"dex"` // uniswap, sushiswap, curve, etc.
	TokenA         string            `json:"token_a"`
	TokenB         string            `json:"token_b"`
	ReserveA      string            `json:"reserve_a"`
	ReserveB      string            `json:"reserve_b"`
	Fee            float64          `json:"fee"` // 0.3 for 0.3%
	Liquidity      float64          `json:"liquidity"`
	Volume24h     float64          `json:"volume_24h"`
	APY            float64          `json:"apy"`
	Stable        bool              `json:"stable"`
}

// SwapRoute represents a swap route
type SwapRoute struct {
	FromToken     string    `json:"from_token"`
	ToToken       string    `json:"to_token"`
	FromChain     int       `json:"from_chain"`
	ToChain       int       `json:"to_chain"`
	AmountIn      string    `json:"amount_in"`
	AmountOut     string    `json:"amount_out"`
	MinimumOut   string    `json:"minimum_out"`
	Slippage      float64   `json:"slippage"`
	GasEstimate   string    `json:"gas_estimate"`
	GasPrice      float64   `json:"gas_price"`
	PriceImpact   float64   `json:"price_impact"`
	Hops          []RouteHop `json:"hops"`
	TotalFee      float64   `json:"total_fee"`
	Blocks        int       `json:"blocks"`
}

type RouteHop struct {
	DEX        string `json:"dex"`
	PoolID     string `json:"pool_id"`
	FromToken  string `json:"from_token"`
	ToToken   string `json:"to_token"`
	FromAmount string `json:"from_amount"`
	ToAmount  string `json:"to_amount"`
	Fee       float64 `json:"fee"`
}

// SwapTransaction represents a swap transaction
type SwapTransaction struct {
	ID          string    `json:"id"`
	WalletID   string    `json:"wallet_id"`
	Route      SwapRoute `json:"route"`
	Status     string    `json:"status"` // pending, confirmed, failed
	Hash       string    `json:"hash"`
	GasUsed    uint64    `json:"gas_used"`
	BlockNumber int64     `json:"block_number"`
	Timestamp  time.Time `json:"timestamp"`
}

// Order represents a limit order
type Order struct {
	ID          string    `json:"id"`
	WalletID   string    `json:"wallet_id"`
	FromToken  string    `json:"from_token"`
	ToToken   string    `json:"to_token"`
	AmountIn  string    `json:"amount_in"`
	Price     string    `json:"price"`
	Status    string    `json:"status"` // pending, filled, cancelled, expired
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// RecurringSwap represents a recurring buy
type RecurringSwap struct {
	ID           string    `json:"id"`
	WalletID    string    `json:"wallet_id"`
	FromToken   string    `json:"from_token"`
	ToToken    string    `json:"to_token"`
	Amount     string    `json:"amount"`
	Interval   time.Duration `json:"interval"` // daily, weekly, monthly
	NextRun    time.Time `json:"next_run"`
	Status    string    `json:"status"` // active, paused, stopped
	TotalRuns  int       `json:"total_runs"`
	CreatedAt  time.Time `json:"created_at"`
}

// Bridge represents a cross-chain bridge
type Bridge struct {
	ID            string `json:"id"`
	FromChain    int    `json:"from_chain"`
	ToChain      int    `json:"to_chain"`
	Name         string `json:"name"` // chainport, across, stargate, etc.
	Fee          float64 `json:"fee"`
	TimeEstimate string `json:"time_estimate"`
	Status     string `json:"status"` // active, paused
}

// ============================================================================
// Storage
// ============================================================================

var (
	swapMux      sync.RWMutex
	tokens       = make(map[string]*Token)
	pools       = make(map[string]*Pool)
	transactions = make(map[string]*SwapTransaction)
	orders      = make(map[string]*Order)
	recurring   = make(map[string]*RecurringSwap)
	bridges     = make(map[string]*Bridge)
	priceCache = make(map[string]priceEntry)
)

type priceEntry struct {
	price    float64
	expiresAt time.Time
}

// ============================================================================
// Core Swap Functions
// ============================================================================

// GetToken returns token info
func GetToken(chainID int, address string) (*Token, error) {
	key := fmt.Sprintf("%d:%s", chainID, address)
	
	swapMux.RLock()
	if token, ok := tokens[key]; ok {
		swapMux.RUnlock()
		return token, nil
	}
	swapMux.RUnlock()
	
	return nil, fmt.Errorf("token not found")
}

// GetTokens returns all tokens on a chain
func GetTokens(chainID int) ([]Token, error) {
	result := make([]Token, 0)
	
	swapMux.RLock()
	for _, token := range tokens {
		if token.ChainID == chainID {
			result = append(result, *token)
		}
	}
	swapMux.RUnlock()
	
	return result, nil
}

// FindBestRoute finds the best swap route
func FindBestRoute(fromChain, toChain int, fromToken, toToken, amount string, maxSlippage float64) (*SwapRoute, error) {
	amountIn, ok := new(big.Int).SetString(amount, 10)
	if !ok {
		return nil, fmt.Errorf("invalid amount")
	}
	
	// Get all pools for the token pair
	pools := findPoolsForSwap(fromChain, toChain, fromToken, toToken)
	if len(pools) == 0 {
		return nil, fmt.Errorf("no liquidity pools found")
	}
	
	// Calculate routes
	routes := calculateRoutes(fromChain, toChain, fromToken, toToken, amountIn, pools, maxSlippage)
	if len(routes) == 0 {
		return nil, fmt.Errorf("no valid routes found")
	}
	
	// Return best route (highest output)
	bestRoute := routes[0]
	return &bestRoute, nil
}

func findPoolsForSwap(chainID int, tokenA, tokenB string) []*Pool {
	result := make([]*Pool, 0)
	
	swapMux.RLock()
	for _, pool := range pools {
		if pool.ChainID == chainID {
			if (pool.TokenA == tokenA && pool.TokenB == tokenB) ||
			   (pool.TokenA == tokenB && pool.TokenB == tokenA) {
				result = append(result, pool)
			}
		}
	}
	swapMux.RUnlock()
	
	return result
}

func calculateRoutes(fromChain, toChain int, fromToken, toToken string, amount *big.Int, pools []*Pool, maxSlippage float64) []SwapRoute {
	routes := make([]SwapRoute, 0)
	
	// Single hop routes
	for _, pool := range pools {
		if pool.TokenA == fromToken && pool.TokenB == toToken {
			out := calculateSingleHop(pool, amount)
			if out.Cmp(big.NewInt(0)) > 0 {
				routes = append(routes, SwapRoute{
					FromToken:   fromToken,
					ToToken:    toToken,
					FromChain: fromChain,
					ToChain:   toChain,
					AmountIn: amount.String(),
					AmountOut: out.String(),
					MinimumOut: calculateMinOut(out, maxSlippage).String(),
					Slippage:  maxSlippage,
					Hops: []RouteHop{
						{
							DEX:       pool.DEX,
							PoolID:    pool.ID,
							FromToken: fromToken,
							ToToken:  toToken,
							FromAmount: amount.String(),
							ToAmount: out.String(),
							Fee:      pool.Fee,
						},
					},
					TotalFee: pool.Fee,
				})
			}
		}
	}
	
	// Sort by output (descending)
	sort.Slice(routes, func(i, j int) bool {
		iOut, _ := new(big.Int).SetString(routes[i].AmountOut, 10)
		jOut, _ := new(big.Int).SetString(routes[j].AmountOut, 10)
		return iOut.Cmp(jOut) > 0
	})
	
	// Return top routes
	if len(routes) > 10 {
		return routes[:10]
	}
	return routes
}

func calculateSingleHop(pool *Pool, amount *big.Int) *big.Int {
	// Constant product formula: x * y = k
	// Output = (amount * reserveB) / (reserveA + amount)
	// With fee: output = (amount * (1000 - fee)) * reserveB / (reserveA * 1000 + amount * (1000 - fee))
	
	reserveA, _ := new(big.Int).SetString(pool.ReserveA, 10)
	reserveB, _ := new(big.Int).SetString(pool.ReserveB, 10)
	
	feeMultiplier := big.NewInt(1000 - int64(pool.Fee*10))
	adjustedAmount := new(big.Int).Mul(amount, feeMultiplier)
	denominator := new(big.Int).Add(
		new(big.Int).Mul(reserveA, big.NewInt(1000)),
		adjustedAmount,
	)
	
	if denominator.Cmp(big.NewInt(0)) == 0 {
		return big.NewInt(0)
	}
	
	output := new(big.Int).Mul(reserveB, adjustedAmount)
	output.Div(output, denominator)
	
	return output
}

func calculateMinOut(amount *big.Int, slippage float64) *big.Int {
	slippageMultiplier := big.NewInt(10000 - int64(slippage*100))
	minOut := new(big.Int).Mul(amount, slippageMultiplier)
	minOut.Div(minOut, big.NewInt(10000))
	return minOut
}

// ExecuteSwap executes a swap
func ExecuteSwap(walletID string, route SwapRoute) (*SwapTransaction, error) {
	// Validate route
	if route.AmountIn == "" || route.AmountOut == "" {
		return nil, fmt.Errorf("invalid route")
	}
	
	tx := &SwapTransaction{
		ID:       uuid.New().String(),
		WalletID: walletID,
		Route:    route,
		Status:   "confirmed", // In production, would be "pending" until confirmed
		Hash:     "0x" + hex.EncodeToString(sha256.Sum256([]byte(uuid.New().String()))),
		Timestamp: time.Now(),
	}
	
	swapMux.Lock()
	transactions[tx.ID] = tx
	swapMux.Unlock()
	
	return tx, nil
}

// ============================================================================
// Limit Orders
// ============================================================================

// CreateOrder creates a limit order
func CreateOrder(walletID, fromToken, toToken, amount, price string, expiresIn time.Duration) (*Order, error) {
	order := &Order{
		ID:        uuid.New().String(),
		WalletID: walletID,
		FromToken: fromToken,
		ToToken:  toToken,
		AmountIn: amount,
		Price:   price,
		Status:   "pending",
		ExpiresAt: time.Now().Add(expiresIn),
		CreatedAt: time.Now(),
	}
	
	swapMux.Lock()
	orders[order.ID] = order
	swapMux.Unlock()
	
	return order, nil
}

// FillOrder fills a limit order (if price condition met)
func FillOrder(orderID string, currentPrice float64) (*SwapTransaction, error) {
	swapMux.Lock()
	order, ok := orders[orderID]
	if !ok {
		swapMux.Unlock()
		return nil, fmt.Errorf("order not found")
	}
	
	if order.Status != "pending" {
		swapMux.Unlock()
		return nil, fmt.Errorf("order not pending")
	}
	
	if time.Now().After(order.ExpiresAt) {
		order.Status = "expired"
		swapMux.Unlock()
		return nil, fmt.Errorf("order expired")
	}
	
	order.Status = "filled"
	swapMux.Unlock()
	
	// Create swap transaction
	tx := &SwapTransaction{
		ID:       uuid.New().String(),
		WalletID: order.WalletID,
		Status:   "confirmed",
		Hash:    "0x" + hex.EncodeToString(sha256.Sum256([]byte(uuid.New().String()))),
		Timestamp: time.Now(),
	}
	
	swapMux.Lock()
	transactions[tx.ID] = tx
	swapMux.Unlock()
	
	return tx, nil
}

// CancelOrder cancels a limit order
func CancelOrder(orderID string) error {
	swapMux.Lock()
	if order, ok := orders[orderID]; ok {
		order.Status = "cancelled"
	}
	swapMux.Unlock()
	
	return nil
}

// GetPendingOrders returns pending orders for a wallet
func GetPendingOrders(walletID string) ([]Order, error) {
	result := make([]Order, 0)
	
	swapMux.RLock()
	for _, order := range orders {
		if order.WalletID == walletID && order.Status == "pending" {
			result = append(result, *order)
		}
	}
	swapMux.RUnlock()
	
	return result, nil
}

// ============================================================================
// Recurring Buys
// ============================================================================

// CreateRecurringSwap creates a recurring buy
func CreateRecurringSwap(walletID, fromToken, toToken, amount string, interval time.Duration) (*RecurringSwap, error) {
	rec := &RecurringSwap{
		ID:        uuid.New().String(),
		WalletID: walletID,
		FromToken: fromToken,
		ToToken:  toToken,
		Amount:   amount,
		Interval: interval,
		NextRun:  time.Now().Add(interval),
		Status:   "active",
		CreatedAt: time.Now(),
	}
	
	swapMux.Lock()
	recurring[rec.ID] = rec
	swapMux.Unlock()
	
	return rec, nil
}

// ProcessRecurring processes due recurring swaps
func ProcessRecurring() []SwapTransaction {
	result := make([]SwapTransaction, 0)
	now := time.Now()
	
	swapMux.RLock()
	for _, rec := range recurring {
		if rec.Status == "active" && now.After(rec.NextRun) {
			result = append(result, SwapTransaction{
				ID:        uuid.New().String(),
				WalletID: rec.WalletID,
				Status:   "pending",
				Timestamp: now,
			})
		}
	}
	swapMux.RUnlock()
	
	// Update next run time
	swapMux.Lock()
	for _, rec := range recurring {
		if rec.Status == "active" && now.After(rec.NextRun) {
			rec.NextRun = now.Add(rec.Interval)
			rec.TotalRuns++
		}
	}
	swapMux.Unlock()
	
	return result
}

// PauseRecurring pauses a recurring swap
func PauseRecurring(id string) error {
	swapMux.Lock()
	if rec, ok := recurring[id]; ok {
		rec.Status = "paused"
	}
	swapMux.Unlock()
	
	return nil
}

// ResumeRecurring resumes a recurring swap
func ResumeRecurring(id string) error {
	swapMux.Lock()
	if rec, ok := recurring[id]; ok {
		rec.Status = "active"
	}
	swapMux.Unlock()
	
	return nil
}

// ============================================================================
// Cross-Chain Bridges
// ============================================================================

// GetBridgeQuote gets a bridge quote
func GetBridgeQuote(fromChain, toChain int, fromToken, toToken, amount string) (map[string]interface{}, error) {
	key := fmt.Sprintf("%d:%d", fromChain, toChain)
	
	swapMux.RLock()
	bridge, ok := bridges[key]
	swapMux.RUnlock()
	
	if !ok {
		// Simulate bridge quote
		return map[string]interface{}{
			"from_chain": fromChain,
			"to_chain": toChain,
			"amount_in": amount,
			"amount_out": amount, // In production, would account for bridge fee
			"fee": "0.001",
			"time_estimate": "10m",
		}, nil
	}
	
	return map[string]interface{}{
		"from_chain": fromChain,
		"to_chain": toChain,
		"amount_in": amount,
		"amount_out": amount,
		"fee": bridge.Fee,
		"time_estimate": bridge.TimeEstimate,
	}, nil
}

// ExecuteBridge executes a cross-chain bridge
func ExecuteBridge(walletID string, fromChain, toChain int, fromToken, toToken, amount string) (*SwapTransaction, error) {
	tx := &SwapTransaction{
		ID:       uuid.New().String(),
		WalletID: walletID,
		Status:   "pending",
		Hash:    "0x" + hex.EncodeToString(sha256.Sum256([]byte(uuid.New().String()))),
		Timestamp: time.Now(),
	}
	
	swapMux.Lock()
	transactions[tx.ID] = tx
	swapMux.Unlock()
	
	return tx, nil
}

// ============================================================================
// Analytics
// ============================================================================

// GetSwapAnalytics returns swap analytics
func GetSwapAnalytics(chainID int, fromToken, toToken string) (map[string]interface{}, error) {
	pools := findPoolsForSwap(chainID, fromToken, toToken)
	
	totalLiquidity := 0.0
	totalVolume := 0.0
	bestAPY := 0.0
	avgFee := 0.0
	
	for _, pool := range pools {
		totalLiquidity += pool.Liquidity
		totalVolume += pool.Volume24h
		if pool.APY > bestAPY {
			bestAPY = pool.APY
		}
		avgFee += pool.Fee
	}
	
	if len(pools) > 0 {
		avgFee /= float64(len(pools))
	}
	
	return map[string]interface{}{
		"chain_id": chainID,
		"from_token": fromToken,
		"to_token": toToken,
		"total_liquidity": totalLiquidity,
		"volume_24h": totalVolume,
		"best_apy": bestAPY,
		"avg_fee": avgFee,
		"pool_count": len(pools),
	}, nil
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "swap"})
}

func getTokensHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	chainID := r.URL.Query().Get("chain_id")
	
	tokens, err := GetTokens(parseInt(chainID))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(tokens)
}

func getQuoteHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		FromChain int     `json:"from_chain"`
		ToChain   int     `json:"to_chain"`
		FromToken string  `json:"from_token"`
		ToToken  string  `json:"to_token"`
		Amount   string  `json:"amount"`
		Slippage float64 `json:"slippage"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	route, err := FindBestRoute(req.FromChain, req.ToChain, req.FromToken, req.ToToken, req.Amount, req.Slippage)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	
	json.NewEncoder(w).Encode(route)
}

func executeSwapHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		WalletID string     `json:"wallet_id"`
		Route   SwapRoute  `json:"route"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	tx, err := ExecuteSwap(req.WalletID, req.Route)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(tx)
}

func createOrderHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		WalletID   string `json:"wallet_id"`
		FromToken  string `json:"from_token"`
		ToToken   string `json:"to_token"`
		Amount    string `json:"amount"`
		Price     string `json:"price"`
		ExpiresIn int64  `json:"expires_in"` // hours
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	order, err := CreateOrder(req.WalletID, req.FromToken, req.ToToken, req.Amount, req.Price, time.Duration(req.ExpiresIn)*time.Hour)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(order)
}

func cancelOrderHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		OrderID string `json:"order_id"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	if err := CancelOrder(req.OrderID); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
}

func createRecurringHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		WalletID  string `json:"wallet_id"`
		FromToken string `json:"from_token"`
		ToToken   string `json:"to_token"`
		Amount   string `json:"amount"`
		Interval int64  `json:"interval"` // hours
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	rec, err := CreateRecurringSwap(req.WalletID, req.FromToken, req.ToToken, req.Amount, time.Duration(req.Interval)*time.Hour)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(rec)
}

func getBridgeQuoteHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		FromChain int    `json:"from_chain"`
		ToChain   int    `json:"to_chain"`
		FromToken string `json:"from_token"`
		ToToken  string `json:"to_token"`
		Amount   string `json:"amount"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	quote, err := GetBridgeQuote(req.FromChain, req.ToChain, req.FromToken, req.ToToken, req.Amount)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(quote)
}

func executeBridgeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req struct {
		WalletID  string `json:"wallet_id"`
		FromChain int    `json:"from_chain"`
		ToChain   int    `json:"to_chain"`
		FromToken string `json:"from_token"`
		ToToken  string `json:"to_token"`
		Amount   string `json:"amount"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	tx, err := ExecuteBridge(req.WalletID, req.FromChain, req.ToChain, req.FromToken, req.ToToken, req.Amount)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(tx)
}

func getAnalyticsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	chainID := r.URL.Query().Get("chain_id")
	fromToken := r.URL.Query().Get("from_token")
	toToken := r.URL.Query().Get("to_token")
	
	analytics, err := GetSwapAnalytics(parseInt(chainID), fromToken, toToken)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(analytics)
}

// ============================================================================
// Router
// ============================================================================

func router() http.Handler {
	mux := http.NewServeMux()
	
	// Health
	mux.HandleFunc("/health", healthHandler)
	
	// Token queries
	mux.HandleFunc("/api/swap/tokens", getTokensHandler)
	
	// Quotes and execution
	mux.HandleFunc("/api/swap/quote", getQuoteHandler)
	mux.HandleFunc("/api/swap/execute", executeSwapHandler)
	
	// Limit orders
	mux.HandleFunc("/api/swap/order/create", createOrderHandler)
	mux.HandleFunc("/api/swap/order/cancel", cancelOrderHandler)
	
	// Recurring buys
	mux.HandleFunc("/api/swap/recurring/create", createRecurringHandler)
	
	// Cross-chain bridges
	mux.HandleFunc("/api/swap/bridge/quote", getBridgeQuoteHandler)
	mux.HandleFunc("/api/swap/bridge/execute", executeBridgeHandler)
	
	// Analytics
	mux.HandleFunc("/api/swap/analytics", getAnalyticsHandler)
	
	return mux
}

// ============================================================================
// Helpers
// ============================================================================

func parseInt(s string) int {
	if s == "" {
		return 0
	}
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

// ============================================================================
// Main
// ============================================================================

func main() {
	fmt.Printf("Swap Service starting on port %d\n", SwapServicePort)
	
	// Initialize sample data
	initSwapData()
	
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", SwapServicePort),
		Handler:      router(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	
	fmt.Printf("Swap Service ready on :%d\n", SwapServicePort)
	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

func initSwapData() {
	// Initialize sample tokens
	sampleTokens := []Token{
		{ID: "eth", ChainID: 1, Address: "", Symbol: "ETH", Name: "Ethereum", Decimals: 18, Standard: "native", PriceUSD: 3500},
		{ID: "usdt", ChainID: 1, Address: "0xdac17f958d2ee523a2206206994597c13d831ec7", Symbol: "USDT", Name: "Tether USD", Decimals: 6, Standard: "erc20", PriceUSD: 1},
		{ID: "usdc", ChainID: 1, Address: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", Symbol: "USDC", Name: "USD Coin", Decimals: 6, Standard: "erc20", PriceUSD: 1},
		{ID: "dai", ChainID: 1, Address: "0x6b175474e89094c44da98b954eedeac495271d0f", Symbol: "DAI", Name: "Dai", Decimals: 18, Standard: "erc20", PriceUSD: 1},
		{ID: "wbtc", ChainID: 1, Address: "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599", Symbol: "WBTC", Name: "Wrapped Bitcoin", Decimals: 8, Standard: "erc20", PriceUSD: 65000},
		{ID: "bnb", ChainID: 56, Address: "", Symbol: "BNB", Name: "BNB", Decimals: 18, Standard: "native", PriceUSD: 600},
		{ID: "sol", ChainID: 101, Address: "", Symbol: "SOL", Name: "Solana", Decimals: 9, Standard: "native", PriceUSD: 150},
	}
	
	swapMux.Lock()
	for _, token := range sampleTokens {
		key := fmt.Sprintf("%d:%s", token.ChainID, token.Address)
		if token.Address == "" {
			key = fmt.Sprintf("%d:native", token.ChainID)
		}
		tokens[key] = &token
	}
	
	// Initialize sample pools
	samplePools := []Pool{
		{
			ID: "pool_1", ChainID: 1, DEX: "uniswapv3", TokenA: "eth", TokenB: "usdt",
			ReserveA: "100000000000000000000", ReserveB: "350000000000000000000000",
			Fee: 0.3, Liquidity: 5000000, Volume24h: 2500000, APY: 25.5,
		},
		{
			ID: "pool_2", ChainID: 1, DEX: "sushiswap", TokenA: "eth", TokenB: "usdt",
			ReserveA: "50000000000000000000", ReserveB: "175000000000000000000000",
			Fee: 0.3, Liquidity: 2500000, Volume24h: 1200000, APY: 18.2,
		},
		{
			ID: "pool_3", ChainID: 1, DEX: "uniswapv3", TokenA: "usdc", TokenB: "usdt",
			ReserveA: "1000000000000", ReserveB: "1000000000000", Fee: 0.01, Liquidity: 10000000, Stable: true,
		},
	}
	
	for _, pool := range samplePools {
		pools[pool.ID] = &pool
	}
	swapMux.Unlock()
}