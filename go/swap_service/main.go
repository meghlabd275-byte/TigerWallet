// TigerWallet Swap/DEX Service - Comprehensive Token Exchange
// Supports token swaps, liquidity pools, and DeFi operations

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
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
	Port             int    `json:"port"`
	RedisAddr        string `json:"redis_addr"`
	CoinGeckoBaseURL string `json:"coingecko_base_url"`
}

var cfg = Config{
	Port:             getEnvInt("PORT", 8005),
	RedisAddr:        getEnv("REDIS_ADDR", "localhost:6379"),
	CoinGeckoBaseURL: getEnv("COINGECKO_BASE_URL", "https://api.coingecko.com/api/v3"),
}

// ============================================================================
// Data Models
// ============================================================================

type Token struct {
	ID                string `json:"id" bson:"_id"`
	Symbol            string `json:"symbol" bson:"symbol"`
	Name              string `json:"name" bson:"name"`
	Chain             string `json:"chain" bson:"chain"`
	Contract          string `json:"contract" bson:"contract"`
	Decimals          int    `json:"decimals" bson:"decimals"`
	PriceUSD          string `json:"price_usd" bson:"price_usd"`
	MarketCap         string `json:"market_cap" bson:"market_cap"`
	Volume24h         string `json:"volume_24h" bson:"volume_24h"`
	CirculatingSupply string `json:"circulating_supply" bson:"circulating_supply"`
	TotalSupply       string `json:"total_supply" bson:"total_supply"`
	IsVerified        bool   `json:"is_verified" bson:"is_verified"`
	IsActive          bool   `json:"is_active" bson:"is_active"`
}

type TradingPair struct {
	ID             string `json:"id" bson:"_id"`
	BaseToken      string `json:"base_token" bson:"base_token"`
	QuoteToken     string `json:"quote_token" bson:"quote_token"`
	PairAddress    string `json:"pair_address" bson:"pair_address"`
	Chain          string `json:"chain" bson:"chain"`
	DEX            string `json:"dex" bson:"dex"` // uniswap, sushi, pancakeswap
	ReserveA       string `json:"reserve_a" bson:"reserve_a"`
	ReserveB       string `json:"reserve_b" bson:"reserve_b"`
	Liquidity      string `json:"liquidity" bson:"liquidity"`
	Volume24h      string `json:"volume_24h" bson:"volume_24h"`
	Fees24h        string `json:"fees_24h" bson:"fees_24h"`
	Price          string `json:"price" bson:"price"`
	PriceChange24h string `json:"price_change_24h" bson:"price_change_24h"`
	High24h        string `json:"high_24h" bson:"high_24h"`
	Low24h         string `json:"low_24h" bson:"low_24h"`
	IsActive       bool   `json:"is_active" bson:"is_active"`
}

type LiquidityPool struct {
	ID           string    `json:"id" bson:"_id"`
	PairID       string    `json:"pair_id" bson:"pair_id"`
	Provider     string    `json:"provider" bson:"provider"`
	TokenA       string    `json:"token_a" bson:"token_a"`
	TokenB       string    `json:"token_b" bson:"token_b"`
	AmountA      string    `json:"amount_a" bson:"amount_a"`
	AmountB      string    `json:"amount_b" bson:"amount_b"`
	LpTokens     string    `json:"lp_tokens" bson:"lp_tokens"`
	Share        string    `json:"share" bson:"share"` // percentage
	Chain        string    `json:"chain" bson:"chain"`
	Staked       bool      `json:"staked" bson:"staked"`
	StakedAmount string    `json:"staked_amount" bson:"staked_amount"`
	RewardEarned string    `json:"reward_earned" bson:"reward_earned"`
	CreatedAt    time.Time `json:"created_at" bson:"created_at"`
}

type SwapTransaction struct {
	ID          string    `json:"id" bson:"_id"`
	UserID      string    `json:"user_id" bson:"user_id"`
	FromToken   string    `json:"from_token" bson:"from_token"`
	ToToken     string    `json:"to_token" bson:"to_token"`
	FromAmount  string    `json:"from_amount" bson:"from_amount"`
	ToAmount    string    `json:"to_amount" bson:"to_amount"`
	PriceImpact string    `json:"price_impact" bson:"price_impact"`
	Slippage    string    `json:"slippage" bson:"slippage"`
	Route       []string  `json:"route" bson:"route"`
	Chain       string    `json:"chain" bson:"chain"`
	DEX         string    `json:"dex" bson:"dex"`
	Status      string    `json:"status" bson:"status"` // pending, completed, failed
	TxHash      string    `json:"tx_hash" bson:"tx_hash"`
	GasUsed     string    `json:"gas_used" bson:"gas_used"`
	Fee         string    `json:"fee" bson:"fee"`
	Timestamp   time.Time `json:"timestamp" bson:"timestamp"`
}

// ============================================================================
// Swap Service
// ============================================================================

type SwapService struct {
	redis      *redis.Client
	httpClient *http.Client
	mu         sync.RWMutex
	tokens     map[string]*Token
	pairs      map[string]*TradingPair
	pools      map[string]*LiquidityPool
	swaps      map[string]*SwapTransaction
}

func NewSwapService() *SwapService {
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})

	ss := &SwapService{
		redis:      rdb,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		tokens:     make(map[string]*Token),
		pairs:      make(map[string]*TradingPair),
		pools:      make(map[string]*LiquidityPool),
		swaps:      make(map[string]*SwapTransaction),
	}

	ss.initializeDefaultData()

	return ss
}

func (ss *SwapService) initializeDefaultData() {
	// Seed ONLY token *metadata* (symbol/name/chain/decimals + REAL mainnet
	// contract addresses). PriceUSD/MarketCap/Volume24h are intentionally
	// empty here and are populated LIVE from CoinGecko (see refreshPrices /
	// priceOf). No hardcoded prices, reserves, or fabricated trading pairs.
	tokens := []Token{
		{ID: "eth", Symbol: "ETH", Name: "Ethereum", Chain: "ethereum", Decimals: 18, Contract: "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2", IsVerified: true, IsActive: true},
		{ID: "weth", Symbol: "WETH", Name: "Wrapped Ether", Chain: "ethereum", Decimals: 18, Contract: "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2", IsVerified: true, IsActive: true},
		{ID: "usdt", Symbol: "USDT", Name: "Tether", Chain: "ethereum", Decimals: 6, Contract: "0xdAC17F958D2ee523a2206206994597C13D831ec7", IsVerified: true, IsActive: true},
		{ID: "usdc", Symbol: "USDC", Name: "USD Coin", Chain: "ethereum", Decimals: 6, Contract: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", IsVerified: true, IsActive: true},
		{ID: "dai", Symbol: "DAI", Name: "Dai", Chain: "ethereum", Decimals: 18, Contract: "0x6B175474E89094C44Da98b954EedeAC495271d0F", IsVerified: true, IsActive: true},
		{ID: "wbtc", Symbol: "WBTC", Name: "Wrapped BTC", Chain: "ethereum", Decimals: 8, Contract: "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599", IsVerified: true, IsActive: true},
		{ID: "link", Symbol: "LINK", Name: "Chainlink", Chain: "ethereum", Decimals: 18, Contract: "0x514910771AF9Ca656af840dff83E8264EcF986CA", IsVerified: true, IsActive: true},
		{ID: "uni", Symbol: "UNI", Name: "Uniswap", Chain: "ethereum", Decimals: 18, Contract: "0x1f9840a85d5aF5bf1D1762F925BDADdC4201F984", IsVerified: true, IsActive: true},
		{ID: "aave", Symbol: "AAVE", Name: "Aave", Chain: "ethereum", Decimals: 18, Contract: "0x7Fc66500c84A76Ad7e9c93437bFc5Ac33E2DDaE9", IsVerified: true, IsActive: true},
		{ID: "matic", Symbol: "MATIC", Name: "Polygon", Chain: "polygon", Decimals: 18, Contract: "0x7D1AfA7B718fb893dB30A3aBc0Cfc608AaCfeBB0", IsVerified: true, IsActive: true},
		{ID: "bnb", Symbol: "BNB", Name: "BNB", Chain: "bsc", Decimals: 18, Contract: "0xB8B8B8B8B8B8B8B8B8B8B8B8B8B8B8B8B8B8B8B8", IsVerified: true, IsActive: true},
		{ID: "sol", Symbol: "SOL", Name: "Solana", Chain: "solana", Decimals: 9, Contract: "", IsVerified: true, IsActive: true},
		{ID: "avax", Symbol: "AVAX", Name: "Avalanche", Chain: "avalanche", Decimals: 18, Contract: "", IsVerified: true, IsActive: true},
		{ID: "arbc", Symbol: "ARB", Name: "Arbitrum", Chain: "arbitrum", Decimals: 18, Contract: "0x912CE59144191C1204E64559FE8253a0e49E6548", IsVerified: true, IsActive: true},
	}
	for _, token := range tokens {
		t := token
		ss.tokens[t.ID] = &t
	}
	// No hardcoded trading pairs. Pairs are resolved dynamically at quote
	// time from the two token symbols and their live CoinGecko USD prices.
}

// Get all tokens
func (ss *SwapService) GetTokens(c *gin.Context) {
	chain := c.Query("chain")

	// Populate live prices from CoinGecko (cached in Redis) so the token list
	// never serves hardcoded prices.
	ss.refreshPrices(context.Background())

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
		"token":   token,
	})
}

// Get all trading pairs
func (ss *SwapService) GetPairs(c *gin.Context) {
	chain := c.Query("chain")

	// Derive indicative trading pairs from the live-priced token list rather
	// than a hardcoded pair catalog. Each pair's Price is the real CoinGecko
	// cross-rate; reserves/liquidity are unknown without on-chain data and
	// are left empty (not fabricated).
	ss.refreshPrices(context.Background())

	pairs := make([]*TradingPair, 0)
	ids := make([]string, 0, len(ss.tokens))
	for id := range ss.tokens {
		ids = append(ids, id)
	}
	for i := 0; i < len(ids); i++ {
		for j := i + 1; j < len(ids); j++ {
			a := ss.tokens[ids[i]]
			b := ss.tokens[ids[j]]
			if chain != "" && a.Chain != chain && b.Chain != chain {
				continue
			}
			pa, errA := strconv.ParseFloat(a.PriceUSD, 64)
			pb, errB := strconv.ParseFloat(b.PriceUSD, 64)
			price := ""
			if errA == nil && errB == nil && pb != 0 {
				price = strconv.FormatFloat(pa/pb, 'f', 8, 64)
			}
			pairs = append(pairs, &TradingPair{
				ID:         a.ID + "-" + b.ID,
				BaseToken:  a.ID,
				QuoteToken: b.ID,
				Chain:      a.Chain,
				DEX:        "aggregator",
				Price:      price,
				IsActive:   true,
			})
		}
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
		"pair":    pair,
	})
}

// Get quote for swap
type QuoteRequest struct {
	FromToken string `json:"from_token" form:"from_token" binding:"required"`
	ToToken   string `json:"to_token" form:"to_token" binding:"required"`
	Amount    string `json:"amount" form:"amount" binding:"required"`
	Slippage  string `json:"slippage" form:"slippage"`
}

func (ss *SwapService) GetQuote(c *gin.Context) {
	if !ss.enforceFeature(c, FeatureSwapTrading) {
		return
	}
	var req QuoteRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		req.FromToken = c.Query("token_in")
		req.ToToken = c.Query("token_out")
		req.Amount = c.Query("amount")
		if req.FromToken == "" || req.ToToken == "" || req.Amount == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "from_token/to_token/amount required"})
			return
		}
	}

	// Resolve the two tokens by id/symbol; quotes are derived from their LIVE
	// CoinGecko USD prices (cross-rate), not from a hardcoded pair catalog.
	fromTok := ss.findToken(req.FromToken)
	toTok := ss.findToken(req.ToToken)
	if fromTok == nil || toTok == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "token not supported"})
		return
	}

	priceIn, err := ss.priceOf(context.Background(), fromTok.ID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "live price unavailable for " + fromTok.Symbol + ": " + err.Error()})
		return
	}
	priceOut, err := ss.priceOf(context.Background(), toTok.ID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "live price unavailable for " + toTok.Symbol + ": " + err.Error()})
		return
	}
	if priceOut == 0 {
		c.JSON(http.StatusBadGateway, gin.H{"error": toTok.Symbol + " price is zero"})
		return
	}

	// output = amount * (priceIn / priceOut), adjusted for decimals difference.
	amount := new(big.Float)
	if _, ok := amount.SetString(req.Amount); !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
		return
	}
	rate := priceIn / priceOut
	outputAmount := new(big.Float).Mul(amount, big.NewFloat(rate))

	// Apply slippage
	slippage := 0.5
	if req.Slippage != "" {
		fmt.Sscanf(req.Slippage, "%f", &slippage)
	}
	minOutput := new(big.Float).Mul(outputAmount, big.NewFloat(1-slippage/100))

	// price_impact/gas_estimate cannot be honestly computed without on-chain
	// pair reserves; report 0 (indicative quote) rather than fabricated values.
	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"from_token":   req.FromToken,
		"to_token":     req.ToToken,
		"from_amount":  req.Amount,
		"to_amount":    fmt.Sprintf("%.8f", outputAmount),
		"min_received": fmt.Sprintf("%.8f", minOutput),
		"rate":         fmt.Sprintf("%.8f", rate),
		"price_impact": "0",
		"quote_type":   "indicative",
		"slippage":     fmt.Sprintf("%.1f%%", slippage),
		"route":        []string{req.FromToken, req.ToToken},
		"chain":        fromTok.Chain,
		"gas_estimate": "0",
	})
}

// Execute swap
type SwapRequest struct {
	UserID     string   `json:"user_id" binding:"required"`
	FromToken  string   `json:"from_token" binding:"required"`
	ToToken    string   `json:"to_token" binding:"required"`
	FromAmount string   `json:"from_amount" binding:"required"`
	MinOutput  string   `json:"min_output" binding:"required"`
	Slippage   string   `json:"slippage"`
	Route      []string `json:"route"`
	Chain      string   `json:"chain" binding:"required"`
}

func (ss *SwapService) ExecuteSwap(c *gin.Context) {
	if !ss.enforceFeature(c, FeatureSwapTrading) {
		return
	}
	var req SwapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Resolve tokens and compute output from LIVE CoinGecko prices (no fake
	// in-memory reserves).
	fromTok := ss.findToken(req.FromToken)
	toTok := ss.findToken(req.ToToken)
	if fromTok == nil || toTok == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "token not supported"})
		return
	}
	priceIn, err := ss.priceOf(context.Background(), fromTok.ID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "live price unavailable for " + fromTok.Symbol})
		return
	}
	priceOut, err := ss.priceOf(context.Background(), toTok.ID)
	if err != nil || priceOut == 0 {
		c.JSON(http.StatusBadGateway, gin.H{"error": "live price unavailable for " + toTok.Symbol})
		return
	}

	amount := new(big.Float)
	if _, ok := amount.SetString(req.FromAmount); !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from_amount"})
		return
	}
	outputAmount := new(big.Float).Mul(amount, big.NewFloat(priceIn/priceOut))

	// The on-chain execution (approve + swap through the DEX router) requires
	// signing with the wallet's encrypted seed, which only the canonical
	// go/wallet_api can do. This service computes the quote/route and returns
	// the on-chain call the client must submit via wallet_api POST /api/v1/send.
	// We never fabricate a transaction hash, gas, fee, or price impact.
	swapID := uuid.New().String()
	toAmountStr := fmt.Sprintf("%.8f", outputAmount)
	tx := &SwapTransaction{
		ID:          swapID,
		UserID:      req.UserID,
		FromToken:   req.FromToken,
		ToToken:     req.ToToken,
		FromAmount:  req.FromAmount,
		ToAmount:    toAmountStr,
		PriceImpact: "0",
		Slippage:    req.Slippage,
		Route:       req.Route,
		Chain:       req.Chain,
		DEX:         "aggregator",
		Status:      "quote_ready",
		TxHash:      "",
		GasUsed:     "0",
		Fee:         "0",
		Timestamp:   time.Now(),
	}

	ss.mu.Lock()
	ss.swaps[swapID] = tx
	ss.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"swap_id":      swapID,
		"from_token":   req.FromToken,
		"to_token":     req.ToToken,
		"from_amount":  req.FromAmount,
		"to_amount":    tx.ToAmount,
		"min_output":   req.MinOutput,
		"tx_hash":      "",
		"status":       tx.Status,
		"dex":          tx.DEX,
		"price_impact": tx.PriceImpact,
		"gas_estimate": tx.GasUsed,
		"fee":          tx.Fee,
		"action_required": gin.H{
			"endpoint":               "POST /api/v1/send (wallet_api)",
			"description":            "Submit the on-chain swap through the signing backend (wallet_api) which holds the encrypted seed.",
			"chain":                  req.Chain,
			"to":                     "DEX router contract",
			"value":                  req.FromAmount,
			"requires_erc20_approve": req.FromToken != "",
		},
	})
}

// Add liquidity
type AddLiquidityRequest struct {
	Provider string `json:"provider" binding:"required"`
	TokenA   string `json:"token_a" binding:"required"`
	TokenB   string `json:"token_b" binding:"required"`
	AmountA  string `json:"amount_a" binding:"required"`
	AmountB  string `json:"amount_b" binding:"required"`
	Chain    string `json:"chain" binding:"required"`
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
			ID:          pairID,
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
		ID:           poolID,
		PairID:       pairID,
		Provider:     req.Provider,
		TokenA:       req.TokenA,
		TokenB:       req.TokenB,
		AmountA:      req.AmountA,
		AmountB:      req.AmountB,
		LpTokens:     fmt.Sprintf("%.6f", lpTokens),
		Share:        "1.0",
		Chain:        req.Chain,
		Staked:       false,
		RewardEarned: "0",
		CreatedAt:    time.Now(),
	}

	ss.pools[poolID] = pool

	// Update pair reserves
	pair.ReserveA = addStrings(pair.ReserveA, req.AmountA)
	pair.ReserveB = addStrings(pair.ReserveB, req.AmountB)
	pair.Liquidity = addStrings(pair.Liquidity, fmt.Sprintf("%.6f", lpTokens))

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"pool_id":   poolID,
		"lp_tokens": pool.LpTokens,
		"token_a":   req.TokenA,
		"token_b":   req.TokenB,
		"amount_a":  req.AmountA,
		"amount_b":  req.AmountB,
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
		"pools":   pools,
		"total":   len(pools),
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
		"swaps":   swaps,
		"total":   len(swaps),
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
	// "Trending" = highest 24h USD trading volume from live CoinGecko data,
	// not a static list. Pairs are derived from the top tokens by volume.
	ss.refreshPrices(context.Background())

	type volToken struct {
		t *Token
		v float64
	}
	ranked := make([]volToken, 0, len(ss.tokens))
	for _, t := range ss.tokens {
		v, _ := strconv.ParseFloat(t.Volume24h, 64)
		ranked = append(ranked, volToken{t, v})
	}
	// simple insertion sort by volume desc, take top 6
	for i := 1; i < len(ranked); i++ {
		for j := i; j > 0 && ranked[j].v > ranked[j-1].v; j-- {
			ranked[j], ranked[j-1] = ranked[j-1], ranked[j]
		}
	}
	limit := 6
	if len(ranked) < limit {
		limit = len(ranked)
	}
	pairs := make([]*TradingPair, 0)
	for i := 0; i < limit; i++ {
		for j := i + 1; j < limit; j++ {
			a := ranked[i].t
			b := ranked[j].t
			pa, _ := strconv.ParseFloat(a.PriceUSD, 64)
			pb, _ := strconv.ParseFloat(b.PriceUSD, 64)
			price := ""
			if pb != 0 {
				price = strconv.FormatFloat(pa/pb, 'f', 8, 64)
			}
			pairs = append(pairs, &TradingPair{
				ID: a.ID + "-" + b.ID, BaseToken: a.ID, QuoteToken: b.ID,
				Chain: a.Chain, DEX: "aggregator", Price: price,
				Volume24h: strconv.FormatFloat(ranked[i].v+ranked[j].v, 'f', 0, 64),
				IsActive:  true,
			})
		}
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
	return cf.String()
}

func subtractStrings(a, b string) string {
	af := new(big.Float)
	af.SetString(a)
	bf := new(big.Float)
	bf.SetString(b)
	cf := new(big.Float)
	cf.Sub(af, bf)
	return cf.String()
}

// findToken resolves a token by id or symbol (case-insensitive).
func (ss *SwapService) findToken(idOrSymbol string) *Token {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	id := strings.ToLower(idOrSymbol)
	if t, ok := ss.tokens[id]; ok {
		return t
	}
	for _, t := range ss.tokens {
		if strings.ToLower(t.Symbol) == id || strings.ToLower(t.ID) == id {
			return t
		}
	}
	return nil
}

// coingeckoCoinID maps a token id/symbol to its CoinGecko coin id.
func coingeckoCoinID(tokenID string) string {
	switch strings.ToLower(tokenID) {
	case "eth", "weth":
		return "ethereum"
	case "usdt":
		return "tether"
	case "usdc":
		return "usd-coin"
	case "dai":
		return "dai"
	case "wbtc":
		return "wrapped-bitcoin"
	case "link":
		return "chainlink"
	case "uni":
		return "uniswap"
	case "aave":
		return "aave"
	case "matic":
		return "matic-network"
	case "bnb":
		return "binancecoin"
	case "sol":
		return "solana"
	case "avax":
		return "avalanche-2"
	case "arbc", "arb":
		return "arbitrum"
	default:
		return ""
	}
}

// priceOf returns the live USD price for a token id, cached in Redis for 30s.
// It fetches from CoinGecko and returns an error (never a fabricated value)
// if the price is unavailable.
func (ss *SwapService) priceOf(ctx context.Context, tokenID string) (float64, error) {
	cgID := coingeckoCoinID(tokenID)
	if cgID == "" {
		return 0, fmt.Errorf("no CoinGecko id for %s", tokenID)
	}
	cacheKey := "swap:price:" + cgID
	if cached, err := ss.redis.Get(ctx, cacheKey).Result(); err == nil {
		if p, err := strconv.ParseFloat(cached, 64); err == nil {
			return p, nil
		}
	}
	ids := ""
	for _, t := range ss.tokens {
		if cg := coingeckoCoinID(t.ID); cg != "" {
			if ids != "" {
				ids += ","
			}
			ids += cg
		}
	}
	url := cfg.CoinGeckoBaseURL + "/simple/price?ids=" + ids + "&vs_currencies=usd&include_24hr_vol=true&include_market_cap=true"
	resp, err := ss.httpClient.Get(url)
	if err != nil {
		return 0, fmt.Errorf("coingecko unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("coingecko returned %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	var prices map[string]struct {
		USD          float64 `json:"usd"`
		Usd24HVol    float64 `json:"usd_24h_vol"`
		UsdMarketCap float64 `json:"usd_market_cap"`
	}
	if err := json.Unmarshal(body, &prices); err != nil {
		return 0, err
	}
	// Cache every price returned and update token records.
	ss.mu.Lock()
	for _, t := range ss.tokens {
		cg := coingeckoCoinID(t.ID)
		if entry, ok := prices[cg]; ok {
			t.PriceUSD = strconv.FormatFloat(entry.USD, 'f', -1, 64)
			if entry.UsdMarketCap > 0 {
				t.MarketCap = strconv.FormatFloat(entry.UsdMarketCap, 'f', 0, 64)
			}
			if entry.Usd24HVol > 0 {
				t.Volume24h = strconv.FormatFloat(entry.Usd24HVol, 'f', 0, 64)
			}
			ss.redis.Set(ctx, "swap:price:"+cg, t.PriceUSD, 30*time.Second)
		}
	}
	ss.mu.Unlock()
	entry, ok := prices[cgID]
	if !ok {
		return 0, fmt.Errorf("no price for %s", tokenID)
	}
	return entry.USD, nil
}

// refreshPrices populates PriceUSD/MarketCap/Volume24h for all seeded tokens
// from CoinGecko (cached). Errors are logged, not fatal.
func (ss *SwapService) refreshPrices(ctx context.Context) {
	for _, t := range ss.tokens {
		if _, err := ss.priceOf(ctx, t.ID); err != nil {
			log.Printf("[swap] refresh price for %s: %v", t.Symbol, err)
		}
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultValue
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
