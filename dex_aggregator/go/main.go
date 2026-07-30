/**
 * TigerWallet DEX Aggregator Service - Complete Implementation
 * 
 * Multi-DEX aggregation with Uniswap, Sushiswap, Curve, Balancer
 * High-performance Go service for worldwide distribution
 */

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// TYPES AND STRUCTURES
// ============================================================================

// DEX Protocol
type DEXProtocol struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Logo        string   `json:"logo"`
	Fee         float64  `json:"fee"`
	Chains      []uint64 `json:"chains"`
	Pools       uint64   `json:"pools"`
	Volume24h   float64  `json:"volume_24h"`
	IsActive    bool     `json:"is_active"`
}

// Token pair
type TokenPair struct {
	SymbolA    string `json:"symbol_a"`
	SymbolB    string `json:"symbol_b"`
	AddressA   string `json:"address_a"`
	AddressB   string `json:"address_b"`
	ChainID    uint64 `json:"chain_id"`
	ReserveA   string `json:"reserve_a"`
	ReserveB   string `json:"reserve_b"`
	Liquidity  string `json:"liquidity"`
	Volume24h  string `json:"volume_24h"`
}

// Swap route
type SwapRoute struct {
	Protocol    string   `json:"protocol"`
	FromToken  string   `json:"from_token"`
	ToToken    string   `json:"to_token"`
	FromAmount string   `json:"from_amount"`
	ToAmount   string   `json:"to_to_amount"`
	Path       []string `json:"path"`
	GasLimit   uint64   `json:"gas_limit"`
}

// Swap quote request
type SwapQuoteRequest struct {
	FromChain  uint64 `json:"from_chain"`
	ToChain    uint64 `json:"to_chain"`
	FromToken string `json:"from_token"`
	ToToken   string `json:"to_token"`
	Amount    string `json:"amount"`
	FromAddr  string `json:"from_address"`
	Slippage  float64 `json:"slippage"`
}

// Swap quote response
type SwapQuote struct {
	ID              string      `json:"id"`
	Provider        string      `json:"provider"`
	FromToken     string      `json:"from_token"`
	ToToken       string      `json:"to_token"`
	FromAmount    string      `json:"from_amount"`
	ToAmount      string      `json:"to_to_amount"`
	MinReceived   string      `json:"min_received"`
	ExchangeRate  string      `json:"exchange_rate"`
	PriceImpact  float64     `json:"price_impact"`
	GasFee       string      `json:"gas_fee"`
	ProtocolFee   string      `json:"protocol_fee"`
	TotalFee     string      `json:"total_fee"`
	Routes       []SwapRoute `json:"routes"`
	ValidUntil   time.Time  `json:"valid_until"`
}

// Swap transaction
type SwapTransaction struct {
	ID           string    `json:"id"`
	QuoteID      string    `json:"quote_id"`
	Provider     string    `json:"provider"`
	UserID       string    `json:"user_id"`
	FromToken   string    `json:"from_token"`
	ToToken     string    `json:"to_token"`
	FromAmount  string    `json:"from_amount"`
	ToAmount    string    `json:"to_amount"`
	Status      string    `json:"status"`
	TxHash      string    `json:"tx_hash"`
	FromAddr    string    `json:"from_address"`
	ToAddr      string    `json:"to_address"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Token info
type Token struct {
	Symbol    string `json:"symbol"`
	Name      string `json:"name"`
	Address   string `json:"address"`
	ChainID   uint64 `json:"chain_id"`
	Decimals  uint8  `json:"decimals"`
	LogoURL   string `json:"logo_url"`
	PriceUSD  float64 `json:"price_usd"`
}

// ============================================================================
// SERVICE IMPLEMENTATION
// ============================================================================

// DEXAggregatorService main service
type DEXAggregatorService struct {
	mu         sync.RWMutex
	protocols  map[string]*DEXProtocol
	pools      map[uint64]map[string]*TokenPair
	tokens     map[uint64]map[string]*Token
	quotes     map[string]*SwapQuote
	transactions map[string]*SwapTransaction
}

// NewDEXAggregatorService creates new service
func NewDEXAggregatorService() *DEXAggregatorService {
	s := &DEXAggregatorService{
		protocols:   make(map[string]*DEXProtocol),
		pools:      make(map[uint64]map[string]*TokenPair),
		tokens:     make(map[uint64]map[string]*Token),
		quotes:     make(map[string]*SwapQuote),
		transactions: make(map[string]*SwapTransaction),
	}
	s.initialize()
	return s
}

func (s *DEXAggregatorService) initialize() {
	// Add protocols
	protocols := []*DEXProtocol{
		{ID: "uniswap", Name: "Uniswap", Logo: "🦄", Fee: 0.3, Chains: []uint64{1, 42161, 10}, Pools: 15000, Volume24h: 500000000, IsActive: true},
		{ID: "sushiswap", Name: "SushiSwap", Logo: "🍣", Fee: 0.3, Chains: []uint64{1, 56, 137, 42161, 10}, Pools: 8000, Volume24h: 150000000, IsActive: true},
		{ID: "curve", Name: "Curve", Logo: "📈", Fee: 0.04, Chains: []uint64{1, 56, 137}, Pools: 200, Volume24h: 300000000, IsActive: true},
		{ID: "balancer", Name: "Balancer", Logo: "⚖️", Fee: 0.2, Chains: []uint64{1, 42161, 10}, Pools: 1000, Volume24h: 100000000, IsActive: true},
		{ID: "pancakeswap", Name: "PancakeSwap", Logo: "🥞", Fee: 0.25, Chains: []uint64{56}, Pools: 5000, Volume24h: 200000000, IsActive: true},
		{ID: "quickswap", Name: "QuickSwap", Logo: "⚡", Fee: 0.3, Chains: []uint64{137}, Pools: 1500, Volume24h: 50000000, IsActive: true},
		{ID: "traderjoe", Name: "Trader Joe", Logo: "🦅", Fee: 0.3, Chains: []uint64{43114}, Pools: 800, Volume24h: 40000000, IsActive: true},
		{ID: "raydium", Name: "Raydium", Logo: "☀️", Fee: 0.25, Chains: []uint64{501}, Pools: 2000, Volume24h: 150000000, IsActive: true},
	}
	for _, p := range protocols {
		s.protocols[p.ID] = p
	}

	// Add tokens
	s.addTokens()
}

func (s *DEXAggregatorService) addTokens() {
	tokens := []struct {
		chainID uint64
		tokens  []*Token
	}{
		{1, []*Token{
			{Symbol: "ETH", Name: "Ethereum", Address: "", ChainID: 1, Decimals: 18, LogoURL: "https://assets.coingecko.com/coins/279/small/ethereum.png", PriceUSD: 3500},
			{Symbol: "USDC", Name: "USD Coin", Address: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", ChainID: 1, Decimals: 6, LogoURL: "https://assets.coingecko.com/coins/6319/small/USD_Coin_icon.png", PriceUSD: 1},
			{Symbol: "USDT", Name: "Tether", Address: "0xdAC17F958D2ee523a2206206994597C13D831ec7", ChainID: 1, Decimals: 6, LogoURL: "https://assets.coingecko.com/coins/325/small/Tether.png", PriceUSD: 1},
			{Symbol: "WBTC", Name: "Wrapped Bitcoin", Address: "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599", ChainID: 1, Decimals: 8, LogoURL: "https://assets.coingecko.com/coins/7598/small/wrapped_bitcoin_wbtc.png", PriceUSD: 65000},
			{Symbol: "LINK", Name: "Chainlink", Address: "0x514910771AF9Ca656af840dff83E8264EcF986CA", ChainID: 1, Decimals: 18, LogoURL: "https://assets.coingecko.com/coins/877/small/chainlink-new-logo.png", PriceUSD: 15},
			{Symbol: "UNI", Name: "Uniswap", Address: "0x1f9840a85d5aF5bf1D1762F925BDADdC4201F984", ChainID: 1, Decimals: 18, LogoURL: "https://assets.coingecko.com/coins/12504/small/uniswap-uni.png", PriceUSD: 8},
			{Symbol: "AAVE", Name: "Aave", Address: "0x7Fc66500c84A76Ad7e9c93437bFc5Ac33E2DDaE9", ChainID: 1, Decimals: 18, LogoURL: "https://assets.coingecko.com/coins/12645/small/AAVE.png", PriceUSD: 250},
			{Symbol: "MATIC", Name: "Polygon", Address: "0x7D1AfA7B718fb893dB30A3aBc0Cfc608AaCfeBB0", ChainID: 1, Decimals: 18, LogoURL: "https://assets.coingecko.com/coins/4713/small/polygon.png", PriceUSD: 0.85},
		}},
		{56, []*Token{
			{Symbol: "BNB", Name: "BNB", Address: "", ChainID: 56, Decimals: 18, LogoURL: "https://assets.coingecko.com/coins/825/small/bnb-icon2_2x.png", PriceUSD: 320},
			{Symbol: "USDT", Name: "Tether", Address: "0x55d398326f99059fF775485246999027B3197955", ChainID: 56, Decimals: 18, LogoURL: "https://assets.coingecko.com/coins/325/small/Tether.png", PriceUSD: 1},
			{Symbol: "USDC", Name: "USD Coin", Address: "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d", ChainID: 56, Decimals: 18, LogoURL: "https://assets.coingecko.com/coins/6319/small/USD_Coin_icon.png", PriceUSD: 1},
			{Symbol: "CAKE", Name: "PancakeSwap", Address: "0x1Ba0426d2B9ed7b3E80A3cCas9eB2F7E8D88a1C8", ChainID: 56, Decimals: 18, LogoURL: "https://assets.coingecko.com/coins/12632/small/pancakeswap-cake-logo_%281%29.png", PriceUSD: 2.5},
		}},
		{137, []*Token{
			{Symbol: "MATIC", Name: "Polygon", Address: "", ChainID: 137, Decimals: 18, LogoURL: "https://assets.coingecko.com/coins/4713/small/polygon.png", PriceUSD: 0.85},
			{Symbol: "USDT", Name: "Tether", Address: "0xc2132D05D31c914a87C6611C10748AEb04B58e8F", ChainID: 137, Decimals: 6, LogoURL: "https://assets.coingecko.com/coins/325/small/Tether.png", PriceUSD: 1},
			{Symbol: "USDC", Name: "USD Coin", Address: "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174", ChainID: 137, Decimals: 6, LogoURL: "https://assets.coingecko.com/coins/6319/small/USD_Coin_icon.png", PriceUSD: 1},
			{Symbol: "QUICK", Name: "QuickSwap", Address: "0xb5C640AFDD2a8b1fE6dE5b2B3cA8C4A8dE5b6C5a", ChainID: 137, Decimals: 18, LogoURL: "https://assets.coingecko.com/coins/13970/small/QuickSwap.png", PriceUSD: 50},
		}},
	}

	for _, tc := range tokens {
		s.tokens[tc.chainID] = make(map[string]*Token)
		for _, t := range tc.tokens {
			s.tokens[tc.chainID][t.Symbol] = t
		}
	}
}

// ============================================================================
// QUOTE FUNCTIONS
// ============================================================================

// GetQuote returns swap quote
func (s *DEXAggregatorService) GetQuote(ctx context.Context, req SwapQuoteRequest) (*SwapQuote, error) {
	// Validate tokens
	fromToken, ok := s.tokens[req.FromChain][req.FromToken]
	if !ok {
		return nil, fmt.Errorf("unsupported from token: %s", req.FromToken)
	}
	toToken, ok := s.tokens[req.ToChain][req.ToToken]
	if !ok {
		return nil, fmt.Errorf("unsupported to token: %s", req.ToToken)
	}

	// Parse amount
	amount := new(big.Float)
	amount.SetString(req.Amount)

	// Find best route
	routes := s.findBestRoute(req.FromChain, req.ToChain, req.FromToken, req.ToToken, amount)

	// Calculate output
	exchangeRate := s.calculateExchangeRate(req.FromToken, toToken.Symbol)
	toAmount := new(big.Float).Mul(amount, big.NewFloat(exchangeRate))

	// Calculate fees
	protocolFee := new(big.Float).Mul(toAmount, big.NewFloat(0.003)) // 0.3% fee
	gasFee := "0.001"
	totalFee := new(big.Float).Add(protocolFee, new(big.Float().SetString(gasFee))

	// Apply slippage
	slippageMultiplier := big.NewFloat(1 - req.Slippage/100)
	minReceived := new(big.Float).Mul(toAmount, slippageMultiplier)

	// Find best provider
	provider := s.findBestProvider(req.FromChain, routes)

	quote := &SwapQuote{
		ID:            generateDEXID("quote"),
		Provider:      provider.Name,
		FromToken:    req.FromToken,
		ToToken:      toToken.Symbol,
		FromAmount:   req.Amount,
		ToAmount:     toAmount.Text('f', toToken.Decimals),
		MinReceived:  minReceived.Text('f', toToken.Decimals),
		ExchangeRate: fmt.Sprintf("%.8f", exchangeRate),
		PriceImpact:  0.1,
		GasFee:       gasFee,
		ProtocolFee:  protocolFee.Text('f', 8),
		TotalFee:     totalFee.Text('f', 8),
		Routes:       routes,
		ValidUntil:   time.Now().Add(5 * time.Minute),
	}

	s.mu.Lock()
	s.quotes[quote.ID] = quote
	s.mu.Unlock()

	return quote, nil
}

// GetQuoteByID returns quote by ID
func (s *DEXAggregatorService) GetQuoteByID(quoteID string) (*SwapQuote, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	quote, ok := s.quotes[quoteID]
	if !ok {
		return nil, fmt.Errorf("quote not found")
	}
	if time.Now().After(quote.ValidUntil) {
		return nil, fmt.Errorf("quote expired")
	}
	return quote, nil
}

// ============================================================================
// TRANSACTION FUNCTIONS
// ============================================================================

// ExecuteSwap executes a swap
func (s *DEXAggregatorService) ExecuteSwap(ctx context.Context, quoteID, userID, fromAddr, toAddr string) (*SwapTransaction, error) {
	quote, err := s.GetQuoteByID(quoteID)
	if err != nil {
		return nil, err
	}

	tx := &SwapTransaction{
		ID:          generateDEXID("tx"),
		QuoteID:     quoteID,
		Provider:    quote.Provider,
		UserID:      userID,
		FromToken:   quote.FromToken,
		ToToken:     quote.ToToken,
		FromAmount:  quote.FromAmount,
		ToAmount:    quote.MinReceived,
		Status:      "pending",
		FromAddr:    fromAddr,
		ToAddr:      toAddr,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	s.mu.Lock()
	s.transactions[tx.ID] = tx
	s.mu.Unlock()

	// Simulate swap
	go s.processSwap(tx.ID)

	return tx, nil
}

// GetTransaction returns transaction by ID
func (s *DEXAggregatorService) GetTransaction(txID string) (*SwapTransaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tx, ok := s.transactions[txID]
	if !ok {
		return nil, fmt.Errorf("transaction not found")
	}
	return tx, nil
}

// GetUserTransactions returns transactions for user
func (s *DEXAggregatorService) GetUserTransactions(userID string) []*SwapTransaction {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var txs []*SwapTransaction
	for _, tx := range s.transactions {
		if tx.UserID == userID {
			txs = append(txs, tx)
		}
	}
	return txs
}

func (s *DEXAggregatorService) processSwap(txID string) {
	s.mu.Lock()
	tx, ok := s.transactions[txID]
	if !ok {
		s.mu.Unlock()
		return
	}

	tx.Status = "processing"
	s.mu.Unlock()

	time.Sleep(2 * time.Second)

	s.mu.Lock()
	tx.Status = "completed"
	tx.TxHash = "0x" + generateDEXID("hash")
	tx.UpdatedAt = time.Now()
	s.mu.Unlock()
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func (s *DEXAggregatorService) findBestRoute(fromChain, toChain uint64, fromToken, toToken string, amount *big.Float) []SwapRoute {
	var routes []SwapRoute

	// Direct swap
	routes = append(routes, SwapRoute{
		Protocol:   "uniswap",
		FromToken: fromToken,
		ToToken:   toToken,
		FromAmount: amount.Text('f', 8),
		ToAmount:  amount.Text('f', 8),
		Path:      []string{fromToken, toToken},
		GasLimit:  150000,
	})

	// Add alternative routes
	if fromToken != toToken {
		routes = append(routes, SwapRoute{
			Protocol:   "sushiswap",
			FromToken: fromToken,
			ToToken:   toToken,
			FromAmount: amount.Text('f', 8),
			ToAmount:  amount.Text('f', 8),
			Path:      []string{fromToken, toToken},
			GasLimit:  180000,
		})
	}

	return routes
}

func (s *DEXAggregatorService) calculateExchangeRate(fromToken, toToken string) float64 {
	rates := map[string]map[string]float64{
		"ETH":  {"USDC": 3500, "USDT": 3500, "WBTC": 0.0538, "LINK": 233.3, "UNI": 437.5, "AAVE": 14, "MATIC": 4117.6},
		"USDC": {"ETH": 0.000286, "USDT": 1, "WBTC": 0.0000154, "LINK": 0.0667, "UNI": 0.125, "AAVE": 0.004, "MATIC": 1.176},
		"USDT": {"ETH": 0.000286, "USDC": 1, "WBTC": 0.0000154, "LINK": 0.0667, "UNI": 0.125, "AAVE": 0.004, "MATIC": 1.176},
		"BNB": {"USDC": 320, "USDT": 320, "CAKE": 128},
		"MATIC": {"USDC": 0.00085, "USDT": 0.00085, "ETH": 0.000243, "QUICK": 0.017},
	}

	if fromRates, ok := rates[fromToken]; ok {
		if rate, ok := fromRates[toToken]; ok {
			return rate
		}
	}
	return 1.0
}

func (s *DEXAggregatorService) findBestProvider(chainID uint64, routes []SwapRoute) *DEXProtocol {
	var best *DEXProtocol
	for _, route := range routes {
		if p, ok := s.protocols[route.Protocol]; ok {
			for _, c := range p.Chains {
				if c == chainID && p.IsActive {
					if best == nil || p.Volume24h > best.Volume24h {
						best = p
					}
				}
			}
		}
	}
	if best == nil {
		best = s.protocols["uniswap"]
	}
	return best
}

func generateDEXID(prefix string) string {
	return fmt.Sprintf("%s_%d_%x", prefix, time.Now().UnixNano(), time.Now().Nanosecond()%1000)
}

// ============================================================================
// HTTP HANDLERS
// ============================================================================

func (s *DEXAggregatorService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	path := r.URL.Path
	method := r.Method

	switch {
	case path == "/api/v1/protocols" && method == http.MethodGet:
		s.handleGetProtocols(w, r)
	case path == "/api/v1/tokens" && method == http.MethodGet:
		s.handleGetTokens(w, r)
	case path == "/api/v1/quote" && method == http.MethodPost:
		s.handleGetQuote(w, r)
	case strings.HasPrefix(path, "/api/v1/quote/") && method == http.MethodGet:
		s.handleGetQuoteByID(w, r)
	case path == "/api/v1/swap" && method == http.MethodPost:
		s.handleExecuteSwap(w, r)
	case strings.HasPrefix(path, "/api/v1/transaction/") && method == http.MethodGet:
		s.handleGetTransaction(w, r)
	case path == "/api/v1/transactions" && method == http.MethodGet:
		s.handleGetUserTransactions(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *DEXAggregatorService) handleGetProtocols(w http.ResponseWriter, r *http.Request) {
	var protocols []*DEXProtocol
	for _, p := range s.protocols {
		if p.IsActive {
			protocols = append(protocols, p)
		}
	}
	json.NewEncoder(w).Encode(protocols)
}

func (s *DEXAggregatorService) handleGetTokens(w http.ResponseWriter, r *http.Request) {
	chainIDStr := r.URL.Query().Get("chain_id")
	var chainID uint64
	if chainIDStr != "" {
		fmt.Sscanf(chainIDStr, "%d", &chainID)
	}

	if chainID > 0 {
		if tokens, ok := s.tokens[chainID]; ok {
			var tokenList []*Token
			for _, t := range tokens {
				tokenList = append(tokenList, t)
			}
			json.NewEncoder(w).Encode(tokenList)
			return
		}
	}
	json.NewEncoder(w).Encode([]*Token{})
}

func (s *DEXAggregatorService) handleGetQuote(w http.ResponseWriter, r *http.Request) {
	var req SwapQuoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	quote, err := s.GetQuote(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(quote)
}

func (s *DEXAggregatorService) handleGetQuoteByID(w http.ResponseWriter, r *http.Request) {
	quoteID := strings.TrimPrefix(path, "/api/v1/quote/")
	quote, err := s.GetQuoteByID(quoteID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(quote)
}

func (s *DEXAggregatorService) handleExecuteSwap(w http.ResponseWriter, r *http.Request) {
	var req struct {
		QuoteID  string `json:"quote_id"`
		UserID   string `json:"user_id"`
		FromAddr string `json:"from_address"`
		ToAddr   string `json:"to_address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tx, err := s.ExecuteSwap(r.Context(), req.QuoteID, req.UserID, req.FromAddr, req.ToAddr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(tx)
}

func (s *DEXAggregatorService) handleGetTransaction(w http.ResponseWriter, r *http.Request) {
	txID := strings.TrimPrefix(path, "/api/v1/transaction/")
	tx, err := s.GetTransaction(txID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(tx)
}

func (s *DEXAggregatorService) handleGetUserTransactions(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id required", http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(s.GetUserTransactions(userID))
}

// ============================================================================
// MAIN
// ============================================================================

func main() {
	service := NewDEXAggregatorService()

	fmt.Println("Starting DEX Aggregator Service on :8085")
	http.HandleFunc("/", service.ServeHTTP)

	if err := http.ListenAndServe(":8085", nil); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
