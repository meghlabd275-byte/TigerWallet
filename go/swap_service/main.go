// TigerWallet Swap/DEX Service - Comprehensive Token Exchange
// Supports token swaps, liquidity pools, and DeFi operations

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Port     int    `json:"port"`
	RedisAddr string `json:"redis_addr"`
}

var cfg = Config{
	Port:     8005,
	RedisAddr: "localhost:6379",
}

// ============================================================================
// Data Models
// ============================================================================

type Token struct {
	ID            string `json:"id" bson:"_id"`
	Symbol        string `json:"symbol" bson:"symbol"`
	Name          string `json:"name" bson:"name"`
	Chain         string `json:"chain" bson:"chain"`
	Contract      string `json:"contract" bson:"contract"`
	Decimals     int    `json:"decimals" bson:"decimals"`
	PriceUSD     string `json:"price_usd" bson:"price_usd"`
	MarketCap    string `json:"market_cap" bson:"market_cap"`
	Volume24h    string `json:"volume_24h" bson:"volume_24h"`
	CirculatingSupply string `json:"circulating_supply" bson:"circulating_supply"`
	TotalSupply  string `json:"total_supply" bson:"total_supply"`
	IsVerified   bool   `json:"is_verified" bson:"is_verified"`
	IsActive     bool   `json:"is_active" bson:"is_active"`
}

type TradingPair struct {
	ID            string   `json:"id" bson:"_id"`
	BaseToken    string   `json:"base_token" bson:"base_token"`
	QuoteToken   string   `json:"quote_token" bson:"quote_token"`
	PairAddress  string   `json:"pair_address" bson:"pair_address"`
	Chain        string   `json:"chain" bson:"chain"`
	 DEX         string   `json:"dex" bson:"dex"` // uniswap, sushi, pancakeswap
	ReserveA     string   `json:"reserve_a" bson:"reserve_a"`
	ReserveB     string   `json:"reserve_b" bson:"reserve_b"`
	Liquidity    string   `json:"liquidity" bson:"liquidity"`
	Volume24h    string   `json:"volume_24h" bson:"volume_24h"`
	Fees24h      string   `json:"fees_24h" bson:"fees_24h"`
	Price        string   `json:"price" bson:"price"`
	PriceChange24h string `json:"price_change_24h" bson:"price_change_24h"`
	High24h      string   `json:"high_24h" bson:"high_24h"`
	Low24h       string   `json:"low_24h" bson:"low_24h"`
	IsActive     bool     `json:"is_active" bson:"is_active"`
}

type LiquidityPool struct {
	ID           string `json:"id" bson:"_id"`
	PairID       string `json:"pair_id" bson:"pair_id"`
	Provider     string `json:"provider" bson:"provider"`
	TokenA       string `json:"token_a" bson:"token_a"`
	TokenB       string `json:"token_b" bson:"token_b"`
	AmountA      string `json:"amount_a" bson:"amount_a"`
	AmountB      string `json:"amount_b" bson:"amount_b"`
	LpTokens     string `json:"lp_tokens" bson:"lp_tokens"`
	Share        string `json:"share" bson:"share"` // percentage
	Chain        string `json:"chain" bson:"chain"`
	Staked       bool   `json:"staked" bson:"staked"`
	StakedAmount string `json:"staked_amount" bson:"staked_amount"`
	RewardEarned string `json:"reward_earned" bson:"reward_earned"`
	CreatedAt    time.Time `json:"created_at" bson:"created_at"`
}

type SwapTransaction struct {
	ID            string    `json:"id" bson:"_id"`
	UserID        string    `json:"user_id" bson:"user_id"`
	FromToken    string    `json:"from_token" bson:"from_token"`
	ToToken      string    `json:"to_token" bson:"to_token"`
	FromAmount   string    `json:"from_amount" bson:"from_amount"`
	ToAmount     string    `json:"to_amount" bson:"to_amount"`
	PriceImpact  string    `json:"price_impact" bson:"price_impact"`
	Slippage     string    `json:"slippage" bson:"slippage"`
	Route        []string `json:"route" bson:"route"`
	Chain        string   `json:"chain" bson:"chain"`
	 DEX         string   `json:"dex" bson:"dex"`
	Status       string   `json:"status" bson:"status"` // pending, completed, failed
	TxHash       string   `json:"tx_hash" bson:"tx_hash"`
	GasUsed      string   `json:"gas_used" bson:"gas_used"`
	Fee          string   `json:"fee" bson:"fee"`
	Timestamp    time.Time `json:"timestamp" bson:"timestamp"`
}

// ============================================================================
// Swap Service
// ============================================================================

type SwapService struct {
	redis       *redis.Client
	mu          sync.RWMutex
	tokens      map[string]*Token
	pairs       map[string]*TradingPair
	pools       map[string]*LiquidityPool
	swaps       map[string]*SwapTransaction
}

func NewSwapService() *SwapService {
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})

	ss := &SwapService{
		redis: rdb,
		tokens: make(map[string]*Token),
		pairs:  make(map[string]*TradingPair),
		pools:  make(map[string]*LiquidityPool),
		swaps:  make(map[string]*SwapTransaction),
	}

	ss.initializeDefaultData()

	return ss
}

func (ss *SwapService) initializeDefaultData() {
	// Initialize tokens
	tokens := []Token{
		{ID: "eth", Symbol: "ETH", Name: "Ethereum", Chain: "ethereum", Decimals: 18, PriceUSD: "2500.00", MarketCap: "300B", Volume24h: "15B", IsVerified: true, IsActive: true},
		{ID: "weth", Symbol: "WETH", Name: "Wrapped Ethereum", Chain: "ethereum", Decimals: 18, PriceUSD: "2500.00", MarketCap: "10B", Volume24h: "500M", IsVerified: true, IsActive: true},
		{ID: "usdt", Symbol: "USDT", Name: "Tether", Chain: "ethereum", Decimals: 6, PriceUSD: "1.00", MarketCap: "100B", Volume24h: "50B", IsVerified: true, IsActive: true},
		{ID: "usdc", Symbol: "USDC", Name: "USD Coin", Chain: "ethereum", Decimals: 6, PriceUSD: "1.00", MarketCap: "40B", Volume24h: "20B", IsVerified: true, IsActive: true},
		{ID: "dai", Symbol: "DAI", Name: "Dai", Chain: "ethereum", Decimals: 18, PriceUSD: "1.00", MarketCap: "5B", Volume24h: "500M", IsVerified: true, IsActive: true},
		{ID: "wbtc", Symbol: "WBTC", Name: "Wrapped Bitcoin", Chain: "ethereum", Decimals: 8, PriceUSD: "45000.00", MarketCap: "10B", Volume24h: "1B", IsVerified: true, IsActive: true},
		{ID: "link", Symbol: "LINK", Name: "Chainlink", Chain: "ethereum", Decimals: 18, PriceUSD: "15.00", MarketCap: "8B", Volume24h: "800M", IsVerified: true, IsActive: true},
		{ID: "uni", Symbol: "UNI", Name: "Uniswap", Chain: "ethereum", Decimals: 18, PriceUSD: "7.50", MarketCap: "5B", Volume24h: "300M", IsVerified: true, IsActive: true},
		{ID: "aave", Symbol: "AAVE", Name: "Aave", Chain: "ethereum", Decimals: 18, PriceUSD: "250.00", MarketCap: "3.5B", Volume24h: "200M", IsVerified: true, IsActive: true},
		{ID: "matic", Symbol: "MATIC", Name: "Polygon", Chain: "polygon", Decimals: 18, PriceUSD: "0.80", MarketCap: "7B", Volume24h: "500M", IsVerified: true, IsActive: true},
		{ID: "bnb", Symbol: "BNB", Name: "BNB", Chain: "bsc", Decimals: 18, PriceUSD: "350.00", MarketCap: "50B", Volume24h: "2B", IsVerified: true, IsActive: true},
		{ID: "sol", Symbol: "SOL", Name: "Solana", Chain: "solana", Decimals: 9, PriceUSD: "100.00", MarketCap: "40B", Volume24h: "3B", IsVerified: true, IsActive: true},
		{ID: "dot", Symbol: "DOT", Name: "Polkadot", Chain: "polkadot", Decimals: 10, PriceUSD: "7.00", MarketCap: "10B", Volume24h: "500M", IsVerified: true, IsActive: true},
		{ID: "avax", Symbol: "AVAX", Name: "Avalanche", Chain: "avalanche", Decimals: 18, PriceUSD: "35.00", MarketCap: "12B", Volume24h: "800M", IsVerified: true, IsActive: true},
		{ID: "arbc", Symbol: "ARB", Name: "Arbitrum", Chain: "arbitrum", Decimals: 18, PriceUSD: "1.10", MarketCap: "3B", Volume24h: "400M", IsVerified: true, IsActive: true},
	}

	for _, token := range tokens {
		ss.tokens[token.ID] = &token
	}

	// Initialize trading pairs
	pairs := []TradingPair{
		{ID: "eth-usdt", BaseToken: "eth", QuoteToken: "usdt", PairAddress: "0x0d4a11d5EEaaC28acE9F7B2eC56A2c5eB3wWv5", Chain: "ethereum", DEX: "uniswap", ReserveA: "50000", ReserveB: "125000000", Liquidity: "2500000", Volume24h: "150000000", Price: "2500.00", PriceChange24h: "2.5%", High24h: "2550.00", Low24h: "2450.00", IsActive: true},
		{ID: "weth-usdt", BaseToken: "weth", QuoteToken: "usdt", PairAddress: "0xC2b7B2a9A4f2D3e4f5a6b7c8d9e0f1a2b3c4d5", Chain: "ethereum", DEX: "uniswap", ReserveA: "10000", ReserveB: "25000000", Liquidity: "500000", Volume24h: "50000000", Price: "2500.00", PriceChange24h: "2.5%", High24h: "2550.00", Low24h: "2450.00", IsActive: true},
		{ID: "usdc-eth", BaseToken: "usdc", QuoteToken: "eth", PairAddress: "0xB4e16d0168e52d35CaCD2c6185b44381D4C5f23", Chain: "ethereum", DEX: "uniswap", ReserveA: "25000000", ReserveB: "10000", Liquidity: "500000", Volume24h: "30000000", Price: "0.0004", PriceChange24h: "-1.2%", High24h: "0.00041", Low24h: "0.00039", IsActive: true},
		{ID: "link-eth", BaseToken: "link", QuoteToken: "eth", PairAddress: "0xA2107a5D05B7b9fC3b0c4F3D3e4F5A6B7C8D9E0", Chain: "ethereum", DEX: "uniswap", ReserveA: "500000", ReserveB: "200", Liquidity: "316227", Volume24h: "1000000", Price: "0.0004", PriceChange24h: "3.5%", High24h: "0.00042", Low24h: "0.00038", IsActive: true},
		{ID: "uni-eth", BaseToken: "uni", QuoteToken: "eth", PairAddress: "0x1D415aa39D647834786EB9B5a33C8b9d2c0e7f3", Chain: "ethereum", DEX: "uniswap", ReserveA: "1000000", ReserveB: "133", Liquidity: "115470", Volume24h: "500000", Price: "0.000133", PriceChange24h: "-2.1%", High24h: "0.00014", Low24h: "0.00012", IsActive: true},
		{ID: "bnb-busd", BaseToken: "bnb", QuoteToken: "usdt", PairAddress: "0x58F876857a02D18D2685E9d23B6A9B4C8d5E6F7", Chain: "bsc", DEX: "pancakeswap", ReserveA: "10000", ReserveB: "3500000", Liquidity: "187082", Volume24h: "5000000", Price: "350.00", PriceChange24h: "1.8%", High24h: "360.00", Low24h: "340.00", IsActive: true},
		{ID: "matic-usdt", BaseToken: "matic", QuoteToken: "usdt", PairAddress: "0x9FBa5aB7C8D9E0F1A2B3C4D5E6F7A8B9C0D1E2", Chain: "polygon", DEX: "quickswap", ReserveA: "10000000", ReserveB: "8000000", Liquidity: "894427", Volume24h: "2000000", Price: "0.80", PriceChange24h: "4.2%", High24h: "0.85", Low24h: "0.75", IsActive: true},
		{ID: "sol-usdc", BaseToken: "sol", QuoteToken: "usdc", PairAddress: "sol1...", Chain: "solana", DEX: "raydium", ReserveA: "1000000", ReserveB: "100000000", Liquidity: "10000000", Volume24h: "50000000", Price: "100.00", PriceChange24h: "5.5%", High24h: "105.00", Low24h: "95.00", IsActive: true},
		{ID: "arb-eth", BaseToken: "arbc", QuoteToken: "eth", PairAddress: "0xabc...", Chain: "arbitrum", DEX: "uniswap", ReserveA: "5000000", ReserveB: "2000", Liquidity: "316227", Volume24h: "2000000", Price: "0.0004", PriceChange24h: "6.8%", High24h: "0.00043", Low24h: "0.00037", IsActive: true},
	}

	for _, pair := range pairs {
		ss.pairs[pair.ID] = &pair
	}
}

// ============================================================================
// API Handlers
// ============================================================================

// Get all tokens
func (ss *SwapService) GetTokens(c *gin.Context) {
	chain := c.Query("chain")

	tokens := make([]*Token, 0)
	for _, token := range ss.tokens {
		if chain == "" || token.Chain == chain {
			tokens = append(tokens, token)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"tokens":  tokens,
		"total":   len(tokens),
	})
}

// Get token by symbol
func (ss *SwapService) GetToken(c *gin.Context) {
	symbol := c.Param("symbol")

	var token *Token
	for _, t := range ss.tokens {
		if strings.ToLower(t.Symbol) == strings.ToLower(symbol) {
			token = t
			break
		}
	}

	if token == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "token not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"token":    token,
	})
}

// Get all trading pairs
func (ss *SwapService) GetPairs(c *gin.Context) {
	chain := c.Query("chain")
	dex := c.Query("dex")

	pairs := make([]*TradingPair, 0)
	for _, pair := range ss.pairs {
		if chain != "" && pair.Chain != chain {
			continue
		}
		if dex != "" && pair.DEX != dex {
			continue
		}
		pairs = append(pairs, pair)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"pairs":   pairs,
		"total":   len(pairs),
	})
}

// Get pair by ID
func (ss *SwapService) GetPair(c *gin.Context) {
	pairID := c.Param("id")

	pair, exists := ss.pairs[pairID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "pair not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"pair":     pair,
	})
}

// Get quote for swap
type QuoteRequest struct {
	FromToken string `json:"from_token" binding:"required"`
	ToToken   string `json:"to_token" binding:"required"`
	Amount    string `json:"amount" binding:"required"`
	Slippage  string `json:"slippage"`
}

func (ss *SwapService) GetQuote(c *gin.Context) {
	var req QuoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find pair
	var pair *TradingPair
	for _, p := range ss.pairs {
		if (p.BaseToken == req.FromToken && p.QuoteToken == req.ToToken) ||
			(p.BaseToken == req.ToToken && p.QuoteToken == req.FromToken) {
			pair = p
			break
		}
	}

	if pair == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pair not found"})
		return
	}

	// Calculate quote (simplified AMM formula)
	amount := new(big.Float)
	amount.SetString(req.Amount)

	reserveA := new(big.Float)
	reserveA.SetString(pair.ReserveA)
	reserveB := new(big.Float)
	reserveB.SetString(pair.ReserveB)

	var outputAmount *big.Float
	if pair.BaseToken == req.FromToken {
		outputAmount = new(big.Float).Quo(
			new(big.Float).Mul(amount, reserveB),
			reserveA,
		)
	} else {
		outputAmount = new(big.Float).Quo(
			new(big.Float).Mul(amount, reserveA),
			reserveB,
		)
	}

	// Apply slippage
	slippage := 0.5 // default 0.5%
	if req.Slippage != "" {
		fmt.Sscanf(req.Slippage, "%f", &slippage)
	}

	minOutput := new(big.Float).Mul(outputAmount, big.NewFloat(1-slippage/100))

	// Calculate price impact
	priceImpact := new(big.Float).Quo(amount, reserveA)
	priceImpact.Mul(priceImpact, big.NewFloat(100))

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"from_token":    req.FromToken,
		"to_token":      req.ToToken,
		"from_amount":   req.Amount,
		"to_amount":     fmt.Sprintf("%.6f", outputAmount),
		"min_received":  fmt.Sprintf("%.6f", minOutput),
		"price_impact":  fmt.Sprintf("%.4f%%", priceImpact),
		"slippage":      fmt.Sprintf("%.1f%%", slippage),
		"route":         []string{req.FromToken, req.ToToken},
		"pair":          pair.ID,
		"dex":           pair.DEX,
		"gas_estimate":  "150000",
	})
}

// Execute swap
type SwapRequest struct {
	UserID    string   `json:"user_id" binding:"required"`
	FromToken string   `json:"from_token" binding:"required"`
	ToToken   string   `json:"to_token" binding:"required"`
	FromAmount string  `json:"from_amount" binding:"required"`
	MinOutput  string  `json:"min_output" binding:"required"`
	Slippage   string  `json:"slippage"`
	Route     []string `json:"route"`
	Chain     string   `json:"chain" binding:"required"`
}

func (ss *SwapService) ExecuteSwap(c *gin.Context) {
	var req SwapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find pair
	var pair *TradingPair
	for _, p := range ss.pairs {
		if (p.BaseToken == req.FromToken && p.QuoteToken == req.ToToken) ||
			(p.BaseToken == req.ToToken && p.QuoteToken == req.FromToken) {
			pair = p
			break
		}
	}

	if pair == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pair not found"})
		return
	}

	// Calculate output amount (simplified)
	amount := new(big.Float)
	amount.SetString(req.FromAmount)
	reserveA := new(big.Float)
	reserveA.SetString(pair.ReserveA)
	reserveB := new(big.Float)
	reserveB.SetString(pair.ReserveB)

	var outputAmount *big.Float
	if pair.BaseToken == req.FromToken {
		outputAmount = new(big.Float).Quo(
			new(big.Float).Mul(amount, reserveB),
			reserveA,
		)
	} else {
		outputAmount = new(big.Float).Quo(
			new(big.Float).Mul(amount, reserveA),
			reserveB,
		)
	}

	// Create swap transaction
	swapID := uuid.New().String()
	tx := &SwapTransaction{
		ID:           swapID,
		UserID:       req.UserID,
		FromToken:   req.FromToken,
		ToToken:     req.ToToken,
		FromAmount:  req.FromAmount,
		ToAmount:    fmt.Sprintf("%.6f", outputAmount),
		PriceImpact: "0.5",
		Slippage:    "0.5",
		Route:       req.Route,
		Chain:       req.Chain,
		DEX:         pair.DEX,
		Status:      "pending",
		TxHash:      "", // not broadcast via RPC; real hash requires on-chain broadcast
		GasUsed:     "150000",
		Fee:         "0.003",
		Timestamp:   time.Now(),
	}

	ss.swaps[swapID] = tx

	// Update pair reserves
	if pair.BaseToken == req.FromToken {
		pair.ReserveA = addStrings(pair.ReserveA, req.FromAmount)
		pair.ReserveB = subtractStrings(pair.ReserveB, fmt.Sprintf("%.2f", outputAmount))
	} else {
		pair.ReserveB = addStrings(pair.ReserveB, req.FromAmount)
		pair.ReserveA = subtractStrings(pair.ReserveA, fmt.Sprintf("%.2f", outputAmount))
	}
	pair.Volume24h = addStrings(pair.Volume24h, req.FromAmount)

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"swap_id":    swapID,
		"from_token": req.FromToken,
		"to_token":   req.ToToken,
		"from_amount": req.FromAmount,
		"to_amount":  tx.ToAmount,
		"tx_hash":    tx.TxHash,
		"status":     tx.Status,
		"dex":        tx.DEX,
	})
}

// Add liquidity
type AddLiquidityRequest struct {
	Provider   string `json:"provider" binding:"required"`
	TokenA    string `json:"token_a" binding:"required"`
	TokenB    string `json:"token_b" binding:"required"`
	AmountA   string `json:"amount_a" binding:"required"`
	AmountB   string `json:"amount_b" binding:"required"`
	Chain     string `json:"chain" binding:"required"`
}

func (ss *SwapService) AddLiquidity(c *gin.Context) {
	var req AddLiquidityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find or create pair
	pairID := req.TokenA + "-" + req.TokenB
	pair, exists := ss.pairs[pairID]
	if !exists {
		pair = &TradingPair{
			ID:           pairID,
			BaseToken:   req.TokenA,
			QuoteToken:  req.TokenB,
			PairAddress: "", // pair address requires on-chain contract deployment
			Chain:       req.Chain,
			DEX:         "tigerswap",
			ReserveA:    req.AmountA,
			ReserveB:    req.AmountB,
			Liquidity:   req.AmountA,
			IsActive:    true,
		}
		ss.pairs[pairID] = pair
	}

	// Calculate LP tokens (simplified)
	amountA := new(big.Float)
	amountA.SetString(req.AmountA)
	amountB := new(big.Float)
	amountB.SetString(req.AmountB)

	reserveA := new(big.Float)
	reserveA.SetString(pair.ReserveA)
	reserveB := new(big.Float)
	reserveB.SetString(pair.ReserveB)

	lpTokens := new(big.Float).Mul(
		new(big.Float).Mul(amountA, amountB),
		big.NewFloat(0.5),
	)

	// Create liquidity position
	poolID := uuid.New().String()
	pool := &LiquidityPool{
		ID:         poolID,
		PairID:     pairID,
		Provider:   req.Provider,
		TokenA:     req.TokenA,
		TokenB:     req.TokenB,
		AmountA:    req.AmountA,
		AmountB:    req.AmountB,
		LpTokens:   fmt.Sprintf("%.6f", lpTokens),
		Share:      "1.0",
		Chain:      req.Chain,
		Staked:     false,
		RewardEarned: "0",
		CreatedAt:  time.Now(),
	}

	ss.pools[poolID] = pool

	// Update pair reserves
	pair.ReserveA = addStrings(pair.ReserveA, req.AmountA)
	pair.ReserveB = addStrings(pair.ReserveB, req.AmountB)
	pair.Liquidity = addStrings(pair.Liquidity, fmt.Sprintf("%.6f", lpTokens))

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"pool_id":   poolID,
		"lp_tokens":  pool.LpTokens,
		"token_a":   req.TokenA,
		"token_b":   req.B_token,
		"amount_a":  req.AmountA,
		"amount_b":  req.AmountB,
	})
}

// Get user's liquidity positions
		"share":     pool.Share,
	})
}

// Get user's liquidity positions
func (ss *SwapService) GetUserPools(c *gin.Context) {
	userID := c.Param("user_id")

	pools := make([]*LiquidityPool, 0)
	for _, pool := range ss.pools {
		if pool.Provider == userID {
			pools = append(pools, pool)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"pools":  pools,
		"total":  len(pools),
	})
}

// Get user's swap history
func (ss *SwapService) GetUserSwaps(c *gin.Context) {
	userID := c.Param("user_id")

	swaps := make([]*SwapTransaction, 0)
	for _, swap := range ss.swaps {
		if swap.UserID == userID {
			swaps = append(swaps, swap)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"swaps":  swaps,
		"total":  len(swaps),
	})
}

// Get popular tokens
func (ss *SwapService) GetPopularTokens(c *gin.Context) {
	popular := []string{"eth", "weth", "usdt", "usdc", "dai", "wbtc", "link", "uni", "aave", "matic", "bnb", "sol"}

	tokens := make([]*Token, 0)
	for _, symbol := range popular {
		if token, ok := ss.tokens[symbol]; ok {
			tokens = append(tokens, token)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"tokens":  tokens,
	})
}

// Get trending pairs
func (ss *SwapService) GetTrendingPairs(c *gin.Context) {
	pairs := make([]*TradingPair, 0)
	for _, pair := range ss.pairs {
		pairs = append(pairs, pair)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"pairs":   pairs,
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

func addStrings(a, b string) string {
	af := new(big.Float)
	af.SetString(a)
	bf := new(big.Float)
	bf.SetString(b)
	cf := new(big.Float)
	cf.Add(af, bf)
	res, _ := cf.String()
	return res
}

func subtractStrings(a, b string) string {
	af := new(big.Float)
	af.SetString(a)
	bf := new(big.Float)
	bf.SetString(b)
	cf := new(big.Float)
	cf.Sub(af, bf)
	res, _ := cf.String()
	return res
}

// ============================================================================
// Main
// ============================================================================

func main() {
	log.Println("TigerWallet Swap/DEX Service")
	log.Println("==============================")
	log.Printf("Starting on port %d", cfg.Port)

	ss := NewSwapService()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "swap-service",
			"timestamp": time.Now().Unix(),
		})
	})

	// Public routes
	r.GET("/api/v1/swap/tokens", ss.GetTokens)
	r.GET("/api/v1/swap/tokens/:symbol", ss.GetToken)
	r.GET("/api/v1/swap/pairs", ss.GetPairs)
	r.GET("/api/v1/swap/pairs/:id", ss.GetPair)
	r.GET("/api/v1/swap/popular", ss.GetPopularTokens)
	r.GET("/api/v1/swap/trending", ss.GetTrendingPairs)

	// Protected routes
	api := r.Group("/api/v1/swap")
	{
		api.POST("/quote", ss.GetQuote)
		api.POST("/execute", ss.ExecuteSwap)
		api.POST("/add-liquidity", ss.AddLiquidity)
		api.GET("/users/:user_id/pools", ss.GetUserPools)
		api.GET("/users/:user_id/swaps", ss.GetUserSwaps)
	}

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
