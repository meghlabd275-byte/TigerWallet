/**
 * TigerWallet Liquidity Aggregator Service
 * High-Load Distributed Go Implementation
 * 
 * Features:
 * - Multi-DEX aggregation
 * - Best price finding
 * - Liquidity depth
 * - Smart routing
 * - Slippage protection
 */

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"sync"
	"time"
)

// ============== Data Structures ==============

type DEXPool struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Chain       string  `json:"chain"`
	TokenA      string  `json:"token_a"`
	TokenB      string  `json:"token_b"`
	ReserveA    float64 `json:"reserve_a"`
	ReserveB    float64 `json:"reserve_b"`
	Liquidity   float64 `json:"liquidity"`
	Volume24h   float64 `json:"volume_24h"`
	Fee         float64 `json:"fee"` // in percent
	APY         float64 `json:"apy"`
}

type SwapQuote struct {
	FromToken   string   `json:"from_token"`
	ToToken     string   `json:"to_token"`
	FromAmount  float64  `json:"from_amount"`
	ToAmount    float64  `json:"to_amount"`
	PriceImpact float64  `json:"price_impact"`
	Slippage    float64  `json:"slippage"`
	Route       []RouteStep `json:"route"`
	DEX         string   `json:"dex"`
	GasEstimate float64  `json:"gas_estimate"`
	ValidUntil  int64    `json:"valid_until"`
}

type RouteStep struct {
	DEX       string  `json:"dex"`
	Pool      string  `json:"pool"`
	FromToken string  `json:"from_token"`
	ToToken   string  `json:"to_token"`
}

type LiquidityDepth struct {
	TokenA     string     `json:"token_a"`
	TokenB     string     `json:"token_b"`
	Chain      string     `json:"chain"`
	TotalDepth float64    `json:"total_depth"`
	Pools      []PoolDepth `json:"pools"`
}

type PoolDepth struct {
	DEX       string  `json:"dex"`
	ReserveA  float64 `json:"reserve_a"`
	ReserveB  float64 `json:"reserve_b"`
	ValueUSD  float64 `json:"value_usd"`
	Share     float64 `json:"share"`
}

// ============== Service ==============

type LiquidityService struct {
	pools map[string][]*DEXPool // token pair -> pools
	tokens map[string]*TokenInfo

	mu         sync.RWMutex
	server     *http.Server
}

type TokenInfo struct {
	Symbol    string  `json:"symbol"`
	Name      string  `json:"name"`
	Decimals  int     `json:"decimals"`
	PriceUSD  float64 `json:"price_usd"`
	Chain     string  `json:"chain"`
}

func NewLiquidityService() *LiquidityService {
	s := &LiquidityService{
		pools:  make(map[string][]*DEXPool),
		tokens: make(map[string]*TokenInfo),
	}

	s.initData()
	return s
}

func (s *LiquidityService) initData() {
	// Initialize tokens
	tokens := []*TokenInfo{
		{symbol: "ETH", name: "Ethereum", decimals: 18, price_usd: 3500, chain: "ethereum"},
		{symbol: "USDT", name: "Tether", decimals: 6, price_usd: 1.0, chain: "ethereum"},
		{symbol: "USDC", name: "USD Coin", decimals: 6, price_usd: 1.0, chain: "ethereum"},
		{symbol: "WBTC", name: "Wrapped Bitcoin", decimals: 8, price_usd: 65000, chain: "ethereum"},
		{symbol: "BNB", name: "BNB", decimals: 18, price_usd: 600, chain: "bsc"},
		{symbol: "SOL", name: "Solana", decimals: 9, price_usd: 145, chain: "solana"},
		{symbol: "MATIC", name: "Polygon", decimals: 18, price_usd: 0.8, chain: "polygon"},
	}

	for _, t := range tokens {
		s.tokens[t.symbol] = t
	}

	// Initialize pools
	pools := []*DEXPool{
		// ETH pairs
		{id: "uni_eth_usdt", name: "Uniswap V3", chain: "ethereum", token_a: "ETH", token_b: "USDT", reserve_a: 50000, reserve_b: 175000000, fee: 0.3, apy: 15.5},
		{id: "sushi_eth_usdt", name: "SushiSwap", chain: "ethereum", token_a: "ETH", token_b: "USDT", reserve_a: 25000, reserve_b: 87500000, fee: 0.3, apy: 12.0},
		{id: "curve_eth_usdt", name: "Curve", chain: "ethereum", token_a: "ETH", token_b: "USDT", reserve_a: 100000, reserve_b: 350000000, fee: 0.04, apy: 8.5},

		// BSC pairs
		{id: "pcs_bnb_usdt", name: "PancakeSwap", chain: "bsc", token_a: "BNB", token_b: "USDT", reserve_a: 50000, reserve_b: 30000000, fee: 0.25, apy: 18.0},
		{id: "biswap_bnb_usdt", name: "BiSwap", chain: "bsc", token_a: "BNB", token_b: "USDT", reserve_a: 20000, reserve_b: 12000000, fee: 0.2, apy: 22.0},

		// WBTC
		{id: "uni_wbtc_eth", name: "Uniswap V3", chain: "ethereum", token_a: "WBTC", token_b: "ETH", reserve_a: 500, reserve_b: 15000, fee: 0.3, apy: 5.0},

		// MATIC
		{id: "quickswap_matic_usdt", name: "QuickSwap", chain: "polygon", token_a: "MATIC", token_b: "USDT", reserve_a: 5000000, reserve_b: 4000000, fee: 0.3, apy: 25.0},
	}

	for _, p := range pools {
		p.Liquidity = s.calculatePoolLiquidity(p)
		p.Volume24h = p.Liquidity * 0.1 * (0.5 + math.random())
		key := p.TokenA + "_" + p.TokenB
		s.pools[key] = append(s.pools[key], p)
	}
}

func (s *LiquidityService) calculatePoolLiquidity(p *DEXPool) float64 {
	tokenA, ok := s.tokens[p.TokenA]
	if !ok {
		return 0
	}
	tokenB, ok := s.tokens[p.TokenB]
	if !ok {
		return 0
	}

	return p.ReserveA*tokenA.PriceUSD + p.ReserveB*tokenB.PriceUSD
}

func (s *LiquidityService) Run() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/quote", s.handleQuote)
	mux.HandleFunc("/api/route", s.handleRoute)
	mux.HandleFunc("/api/pools", s.handlePools)
	mux.HandleFunc("/api/depth", s.handleDepth)
	mux.HandleFunc("/api/best_dex", s.handleBestDEX)
	mux.HandleFunc("/health", s.handleHealth)

	s.server = &http.Server{
		Addr:    ":8090",
		Handler: mux,
	}

	log.Println("Liquidity service starting on :8090")
	return s.server.ListenAndServe()
}

// ============== Handlers ==============

func (s *LiquidityService) handleQuote(w http.ResponseWriter, r *http.Request) {
	fromToken := r.URL.Query().Get("from")
	toToken := r.URL.Query().Get("to")
	amountStr := r.URL.QueryGet("amount")

	var amount float64
	if amountStr != "" {
		fmt.Sscanf(amountStr, "%f", &amount)
	} else {
		amount = 1.0
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Find best quote across all pools
	key := fromToken + "_" + toToken
	pools := s.pools[key]

	var bestQuote *SwapQuote
	for _, pool := range pools {
		quote := s.calculateQuote(pool, fromToken, toToken, amount)
		if bestQuote == nil || quote.ToAmount > bestQuote.ToAmount {
			bestQuote = quote
		}
	}

	if bestQuote == nil {
		http.Error(w, "No liquidity found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bestQuote)
}

func (s *LiquidityService) handleRoute(w http.ResponseWriter, r *http.Request) {
	fromToken := r.URL.Query().Get("from")
	toToken := r.URL.Query().Get("to")
	amountStr := r.URL.Query().Get("amount")

	var amount float64
	if amountStr != "" {
		fmt.Sscanf(amountStr, "%f", &amount)
	} else {
		amount = 1.0
	}

	routes := s.findBestRoute(fromToken, toToken, amount)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(routes)
}

func (s *LiquidityService) handlePools(w http.ResponseWriter, r *http.Request) {
	chain := r.URL.Query().Get("chain")
	token := r.URL.Query().Get("token")

	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*DEXPool
	for _, pools := range s.pools {
		for _, p := range pools {
			if chain != "" && p.Chain != chain {
				continue
			}
			if token != "" && p.TokenA != token && p.TokenB != token {
				continue
			}
			result = append(result, p)
		}
	}

	// Sort by liquidity
	sort.Slice(result, func(i, j int) bool {
		return result[i].Liquidity > result[j].Liquidity
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *LiquidityService) handleDepth(w http.ResponseWriter, r *http.Request) {
	tokenA := r.URL.Query().Get("token_a")
	tokenB := r.URL.Query().Get("token_b")
	chain := r.URL.Query().Get("chain")

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := tokenA + "_" + tokenB
	pools := s.pools[key]

	var totalDepth float64
	var poolDepths []PoolDepth

	for _, p := range pools {
		if chain != "" && p.Chain != chain {
			continue
		}

		depth := p.Liquidity
		totalDepth += depth

		poolDepths = append(poolDepths, PoolDepth{
			DEX:      p.Name,
			ReserveA: p.ReserveA,
			ReserveB: p.ReserveB,
			ValueUSD: depth,
		})
	}

	// Calculate shares
	for i := range poolDepths {
		if totalDepth > 0 {
			poolDepths[i].Share = poolDepths[i].ValueUSD / totalDepth * 100
		}
	}

	depth := LiquidityDepth{
		TokenA:     tokenA,
		TokenB:     tokenB,
		Chain:      chain,
		TotalDepth: totalDepth,
		Pools:      poolDepths,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(depth)
}

func (s *LiquidityService) handleBestDEX(w http.ResponseWriter, r *http.Request) {
	tokenA := r.URL.Query().Get("token_a")
	tokenB := r.URL.Query().Get("token_b")

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := tokenA + "_" + tokenB
	pools := s.pools[key]

	if len(pools) == 0 {
		http.Error(w, "No pools found", http.StatusNotFound)
		return
	}

	// Sort by liquidity
	sort.Slice(pools, func(i, j int) bool {
		return pools[i].Liquidity > pools[j].Liquidity
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pools[0])
}

func (s *LiquidityService) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var totalLiquidity float64
	var poolCount int
	for _, pools := range s.pools {
		for _, p := range pools {
			totalLiquidity += p.Liquidity
			poolCount++
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "healthy",
		"total_pools":    poolCount,
		"total_liquidity": totalLiquidity,
		"timestamp":      time.Now().Unix(),
	})
}

// ============== Calculations ==============

func (s *LiquidityService) calculateQuote(pool *DEXPool, fromToken, toToken string, amount float64) *SwapQuote {
	// Constant product formula: x * y = k
	// output = (input * reserveOut * (1 - fee)) / (reserveIn + input * (1 - fee))

	var reserveIn, reserveOut float64
	if fromToken == pool.TokenA {
		reserveIn = pool.ReserveA
		reserveOut = pool.ReserveB
	} else {
		reserveIn = pool.ReserveB
		reserveOut = pool.ReserveA
	}

	fee := pool.Fee / 100.0
	inputWithFee := amount * (1 - fee)
	numerator := inputWithFee * reserveOut
	denominator := reserveIn + inputWithFee
	output := numerator / denominator

	priceImpact := (amount / (reserveIn + amount)) * 100

	return &SwapQuote{
		FromToken:   fromToken,
		ToToken:     toToken,
		FromAmount:  amount,
		ToAmount:    output,
		PriceImpact: priceImpact,
		Slippage:    priceImpact * 1.5,
		Route: []RouteStep{
			{DEX: pool.Name, Pool: pool.ID, FromToken: fromToken, ToToken: toToken},
		},
		DEX:         pool.Name,
		GasEstimate: 150000,
		ValidUntil:  time.Now().Add(30 * time.Second).UnixMilli(),
	}
}

func (s *LiquidityService) findBestRoute(fromToken, toToken string, amount float64) [][]RouteStep {
	// Simple BFS to find routes (in production, use Dijkstra)
	var routes [][]RouteStep

	// Direct route
	key := fromToken + "_" + toToken
	if pools, ok := s.pools[key]; ok && len(pools) > 0 {
		quote := s.calculateQuote(pools[0], fromToken, toToken, amount)
		routes = append(routes, quote.Route)
	}

	return routes
}

// ============== Main ==============

func main() {
	log.Println("Starting TigerWallet Liquidity Aggregator Service...")

	service := NewLiquidityService()
	if err := service.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
