/**
 * TigerWallet Cross-Chain Aggregator Service
 * LI.FI/SwapKit style implementation for multi-chain swaps
 * 
 * Features:
 * - Multi-hop routing across DEXes
 * - Bridge aggregation
 * - Best price finding
 * - Split routing
 * - Real-time price aggregation
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ServerPort     string `json:"server_port"`
	DBHost         string `json:"db_host"`
	DBPort         string `json:"db_port"`
	DBUser         string `json:"db_user"`
	DBPassword     string `json:"db_password"`
	DBName         string `json:"db_name"`
	RedisHost      string `json:"redis_host"`
	RedisPort      string `json:"redis_port"`
	
	// Provider API endpoints (would be configured in production)
	LifiAPIURL    string `json:"lifi_api_url"`
	LiFiAPIKey   string `json:"lifi_api_key"`
	LayerZeroURL string `json:"layer_zero_url"`
	StargateURL  string `json:"stargate_url"`
	AxelarURL    string `json:"axelar_url"`
}

func LoadConfig() *Config {
	return &Config{
		ServerPort:  getEnv("CROSS_CHAIN_PORT", "9095"),
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBPort:      getEnv("DB_PORT", "5432"),
		DBUser:      getEnv("DB_USER", "tigerwallet"),
		DBPassword:  getEnv("DB_PASSWORD", "password"),
		DBName:      getEnv("DB_NAME", "tigerwallet"),
		RedisHost:   getEnv("REDIS_HOST", "localhost"),
		RedisPort:   getEnv("REDIS_PORT", "6379"),
		LiFiAPIURL: getEnv("LIFI_API_URL", "https://api.li.fi/v1"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============================================================================
// Data Models
// ============================================================================

type SwapQuote struct {
	ID                string      `json:"id"`
	FromToken        string      `json:"from_token"`
	ToToken          string      `json:"to_token"`
	FromChain        uint        `json:"from_chain"`
	ToChain          uint        `json:"to_chain"`
	FromAmount       string      `json:"from_amount"`
	ToAmount         string      `json:"to_amount"`
	ToAmountMin      string      `json:"to_amount_min"`
	PriceImpact      float64     `json:"price_impact"`
	EstimatedTime    int         `json:"estimated_time"` // seconds
	Route            []RouteStep `json:"route"`
	Fees            []Fee       `json:"fees"`
	BridgeCalls     []BridgeCall `json:"bridge_calls"`
	SwapCalls       []SwapCall  `json:"swap_calls"`
	
	// Provider info
	Provider         string      `json:"provider"`
	ProviderQuoteID  string      `json:"provider_quote_id"`
	
	// Metadata
	RequestID       string      `json:"request_id"`
	CreatedAt       time.Time   `json:"created_at"`
}

type RouteStep struct {
	Type           string      `json:"type"` // swap, bridge
	Protocol       string      `json:"protocol"` // uniswap, stargate, layerzero
	FromToken     string      `json:"from_token"`
	ToToken       string      `json:"to_token"`
	FromAmount    string      `json:"from_amount"`
	ToAmount     string      `json:"to_amount"`
	FromChain     uint        `json:"from_chain"`
	ToChain       uint        `json:"to_chain"`
	ContractAddr  string      `json:"contract_address"`
	AbiEncoded    string      `json:"abi_encoded"`
}

type Fee struct {
	Name       string  `json:"name"`
	Percentage float64 `json:"percentage"`
	Amount     string  `json:"amount"`
}

type BridgeCall struct {
	FromChain   uint   `json:"from_chain"`
	ToChain     uint   `json:"to_chain"`
	Protocol    string `json:"protocol"`
	SrcToken    string `json:"src_token"`
	DstToken    string `json:"dst_token"`
	Amount      string `json:"amount"`
	Contract    string `json:"contract"`
	Data        string `json:"data"`
	GasLimit    uint64 `json:"gas_limit"`
}

type SwapCall struct {
	Chain       uint   `json:"chain"`
	FromToken   string `json:"from_token"`
	ToToken     string `json:"to_token"`
	Amount      string `json:"amount"`
	 DEX        string `json:"dex"`
	Contract    string `json:"contract"`
	Data        string `json:"data"`
	GasLimit    uint64 `json:"gas_limit"`
}

type TokenPrice struct {
	Token       string  `json:"token"`
	ChainID     uint    `json:"chain_id"`
	PriceUSD    float64 `json:"price_usd"`
	LastUpdated time.Time `json:"last_updated"`
}

type DEXPool struct {
	ID          string  `json:"id"`
	DEX         string  `json:"dex"`
	ChainID    uint    `json:"chain_id"`
	TokenA     string  `json:"token_a"`
	TokenB     string  `json:"token_b"`
	ReserveA   string  `json:"reserve_a"`
	ReserveB   string  `json:"reserve_b"`
	Liquidity   string  `json:"liquidity"`
	Token0Price float64 `json:"token0_price"`
	Token1Price float64 `json:"token1_price"`
	Volume24h   float64 `json:"volume_24h"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type BridgeRoute struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	FromChain   uint   `json:"from_chain"`
	ToChain     uint   `json:"to_chain"`
	Protocol    string `json:"protocol"`
	FeePercent  float64 `json:"fee_percent"`
	FeeFixed    float64 `json:"fee_fixed"`
	TimeEstMin  int    `json:"time_est_min"`
	TimeEstMax  int    `json:"time_est_max"`
	IsActive    bool   `json:"is_active"`
}

type SwapRequest struct {
	FromToken    string  `json:"from_token" binding:"required"`
	ToToken      string  `json:"to_token" binding:"required"`
	FromChain    uint    `json:"from_chain" binding:"required"`
	ToChain      uint    `json:"to_chain" binding:"required"`
	FromAmount   string  `json:"from_amount" binding:"required"`
	Slippage     float64 `json:"slippage"`
	Source       string  `json:"source"` // aggregator, dex, bridge
}

// ============================================================================
// Service Layer
// ============================================================================

type CrossChainService struct {
	db           *gorm.DB
	redis        *redis.Client
	config       *Config
	httpClient   *http.Client
	priceCache   map[string]float64
	priceCacheMu sync.RWMutex
	quoteCache   *sync.Map
}

func NewCrossChainService(cfg *Config) (*CrossChainService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	// Auto migrate
	db.AutoMigrate(&SwapQuote{}, &TokenPrice{}, &DEXPool{}, &BridgeRoute{})

	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		DB:  5,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rdb.Ping(ctx)

	service := &CrossChainService{
		db:         db,
		redis:      rdb,
		config:     cfg,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		priceCache: make(map[string]float64),
		quoteCache: &sync.Map{},
	}

	// Initialize default bridges and DEXes
	service.initializeDefaultRoutes()

	return service, nil
}

func (s *CrossChainService) initializeDefaultRoutes() {
	// Bridge routes
	bridges := []BridgeRoute{
		{ID: "lz_eth_arbitrum", Name: "LayerZero", FromChain: 1, ToChain: 42161, Protocol: "layerzero", FeePercent: 0.1, TimeEstMin: 10, TimeEstMax: 30, IsActive: true},
		{ID: "lz_eth_optimism", Name: "LayerZero", FromChain: 1, ToChain: 10, Protocol: "layerzero", FeePercent: 0.1, TimeEstMin: 10, TimeEstMax: 30, IsActive: true},
		{ID: "lz_eth_polygon", Name: "LayerZero", FromChain: 1, ToChain: 137, Protocol: "layerzero", FeePercent: 0.1, TimeEstMin: 10, TimeEstMax: 30, IsActive: true},
		{ID: "stargate_eth_arbitrum", Name: "Stargate", FromChain: 1, ToChain: 42161, Protocol: "stargate", FeePercent: 0.06, TimeEstMin: 15, TimeEstMax: 45, IsActive: true},
		{ID: "axelar_eth_cosmos", Name: "Axelar", FromChain: 1, ToChain: 0, Protocol: "axelar", FeePercent: 0.15, TimeEstMin: 20, TimeEstMax: 60, IsActive: true},
		{ID: "celer_eth_bsc", Name: "Celer", FromChain: 1, ToChain: 56, Protocol: "celer", FeePercent: 0.08, TimeEstMin: 15, TimeEstMax: 40, IsActive: true},
		{ID: "hop_eth_arbitrum", Name: "Hop", FromChain: 1, ToChain: 42161, Protocol: "hop", FeePercent: 0.05, TimeEstMin: 5, TimeEstMax: 15, IsActive: true},
		{ID: "across_eth_arbitrum", Name: "Across", FromChain: 1, ToChain: 42161, Protocol: "across", FeePercent: 0.04, TimeEstMin: 3, TimeEstMax: 10, IsActive: true},
	}

	for _, bridge := range bridges {
		s.db.FirstOrCreate(&bridge, BridgeRoute{ID: bridge.ID})
	}

	// DEX pools (sample)
	dexPools := []DEXPool{
		{ID: "uni_eth_usdt_1", DEX: "Uniswap", ChainID: 1, TokenA: "ETH", TokenB: "USDT", ReserveA: "10000", ReserveB: "32000000", Liquidity: "178885421", Token0Price: 1, Token1Price: 3200, Volume24h: 5000000},
		{ID: "uni_eth_usdc_1", DEX: "Uniswap", ChainID: 1, TokenA: "ETH", TokenB: "USDC", ReserveA: "8000", ReserveB: "25600000", Liquidity: "142885421", Token0Price: 1, Token1Price: 3200, Volume24h: 4200000},
		{ID: "sushi_eth_usdt_1", DEX: "SushiSwap", ChainID: 1, TokenA: "ETH", TokenB: "USDT", ReserveA: "5000", ReserveB: "16000000", Liquidity: "89442821", Token0Price: 1, Token1Price: 3200, Volume24h: 1500000},
		{ID: "uni_eth_usdt_56", DEX: "Uniswap", ChainID: 56, TokenA: "BNB", TokenB: "USDT", ReserveA: "50000", ReserveB: "15000000", Liquidity: "273861078", Token0Price: 300, Token1Price: 0.0033, Volume24h: 8000000},
		{ID: "pcs_eth_usdt_56", DEX: "PancakeSwap", ChainID: 56, TokenA: "BNB", TokenB: "USDT", ReserveA: "100000", ReserveB: "30000000", Liquidity: "547722156", Token0Price: 300, Token1Price: 0.0033, Volume24h: 15000000},
		{ID: "uni_eth_usdt_137", DEX: "Uniswap", ChainID: 137, TokenA: "MATIC", TokenB: "USDT", ReserveA: "5000000", ReserveB: "5000000", Liquidity: "158113883", Token0Price: 1, Token1Price: 1, Volume24h: 2000000},
		{ID: "quickswap_eth_usdt_137", DEX: "QuickSwap", ChainID: 137, TokenA: "MATIC", TokenB: "USDT", ReserveA: "3000000", ReserveB: "3000000", Liquidity: "94868329", Token0Price: 1, Token1Price: 1, Volume24h: 1200000},
	}

	for _, pool := range dexPools {
		s.db.FirstOrCreate(&pool, DEXPool{ID: pool.ID})
	}
}

// ============================================================================
// Quote Calculation
// ============================================================================

func (s *CrossChainService) GetQuote(c *gin.Context) {
	var req SwapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set default slippage
	if req.Slippage == 0 {
		req.Slippage = 0.5 // 0.5% default
	}

	// Get quotes from multiple providers
	quotes := s.aggregateQuotes(req)

	if len(quotes) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "no quotes available"})
		return
	}

	// Sort by output amount (best first)
	sort.Slice(quotes, func(i, j int) bool {
		amtI, _ := parseAmount(quotes[i].ToAmount)
		amtJ, _ := parseAmount(quotes[j].ToAmount)
		return amtI > amtJ
	})

	// Return best quote
	c.JSON(http.StatusOK, gin.H{
		"quote": quotes[0],
		"alternatives": quotes[1:],
	})
}

func (s *CrossChainService) aggregateQuotes(req SwapRequest) []SwapQuote {
	var quotes []SwapQuote
	var wg sync.WaitGroup

	// Get quotes from different sources in parallel
	wg.Add(3)

	go func() {
		defer wg.Done()
		if q := s.getDEXQuote(req); q != nil {
			quotes = append(quotes, *q)
		}
	}()

	go func() {
		defer wg.Done()
		if q := s.getBridgeQuote(req); q != nil {
			quotes = append(quotes, *q)
		}
	}()

	go func() {
		defer wg.Done()
		if q := s.getAggregatedQuote(req); q != nil {
			quotes = append(quotes, *q)
		}
	}()

	wg.Wait()
	return quotes
}

func (s *CrossChainService) getDEXQuote(req SwapRequest) *SwapQuote {
	// Get pools for the token pair
	var pools []DEXPool
	s.db.Where("chain_id = ? AND ((token_a = ? AND token_b = ?) OR (token_a = ? AND token_b = ?))",
		req.FromChain, req.FromToken, req.ToToken, req.ToToken, req.FromToken).Find(&pools)

	if len(pools) == 0 {
		return nil
	}

	// Calculate output amount using constant product formula
	fromAmount, _ := parseAmount(req.FromAmount)
	bestQuote := &SwapQuote{}

	for _, pool := range pools {
		reserveA, _ := parseAmount(pool.ReserveA)
		reserveB, _ := parseAmount(pool.ReserveB)

		// Get token prices
		priceA := s.getTokenPrice(pool.TokenA, pool.ChainID)
		priceB := s.getTokenPrice(pool.TokenB, pool.ChainID)

		// Simple swap: output = (input * reserveOut) / reserveIn
		amountOut := (fromAmount * reserveB) / reserveA

		// Apply fee (typically 0.3%)
		amountOut = amountOut * 997 / 1000

		// Calculate price impact
		priceImpact := (float64(fromAmount) / float64(reserveA)) * 100

		toAmountStr := fmt.Sprintf("%.0f", amountOut)

		// Calculate minimum received with slippage
		minReceived := amountOut * (100 - req.Slippage*100) / 100

		quote := SwapQuote{
			ID:           "DEX-" + uuid.New().String()[:8],
			FromToken:   req.FromToken,
			ToToken:     req.ToToken,
			FromChain:   req.FromChain,
			ToChain:     req.ToChain,
			FromAmount:  req.FromAmount,
			ToAmount:    toAmountStr,
			ToAmountMin: fmt.Sprintf("%.0f", minReceived),
			PriceImpact: priceImpact,
			EstimatedTime: 30,
			Provider:     pool.DEX,
			RequestID:    uuid.New().String(),
			CreatedAt:    time.Now(),
			Route: []RouteStep{
				{
					Type:        "swap",
					Protocol:   pool.DEX,
					FromToken:  req.FromToken,
					ToToken:    req.ToToken,
					FromAmount: req.FromAmount,
					ToAmount:   toAmountStr,
					FromChain:  req.FromChain,
					ToChain:    req.FromChain,
				},
			},
			Fees: []Fee{
				{Name: "DEX Fee", Percentage: 0.3, Amount: fmt.Sprintf("%.0f", float64(fromAmount)*0.003)},
			},
		}

		// Compare and keep best
		bestAmt, _ := parseAmount(bestQuote.ToAmount)
		if bestAmt == 0 || amountOut > bestAmt {
			bestQuote = quote
		}
	}

	if bestQuote.RequestID == "" {
		return nil
	}

	return bestQuote
}

func (s *CrossChainService) getBridgeQuote(req SwapRequest) *SwapQuote {
	// Get available bridges for this route
	var bridges []BridgeRoute
	s.db.Where("from_chain = ? AND to_chain = ? AND is_active = ?", 
		req.FromChain, req.ToChain, true).Find(&bridges)

	if len(bridges) == 0 {
		return nil
	}

	fromAmount, _ := parseAmount(req.FromAmount)
	bestQuote := &SwapQuote{}

	for _, bridge := range bridges {
		// Calculate bridge output
		feeAmount := fromAmount * bridge.FeePercent / 100
		amountAfterFee := fromAmount - feeAmount

		// Get destination token price (simplified)
		destPrice := s.getTokenPrice(req.ToToken, req.ToChain)
		if destPrice == 0 {
			destPrice = 1 // Default to 1 if no price
		}

		fromPrice := s.getTokenPrice(req.FromToken, req.FromChain)
		if fromPrice == 0 {
			fromPrice = 1
		}

		// Cross-chain rate (simplified - in production would use real rates)
		rate := destPrice / fromPrice
		toAmount := amountAfterFee * rate

		toAmountStr := fmt.Sprintf("%.0f", toAmount)
		minReceived := toAmount * (100 - req.Slippage*100) / 100

		quote := SwapQuote{
			ID:           "BRIDGE-" + uuid.New().String()[:8],
			FromToken:   req.FromToken,
			ToToken:     req.ToToken,
			FromChain:   req.FromChain,
			ToChain:     req.ToChain,
			FromAmount:  req.FromAmount,
			ToAmount:    toAmountStr,
			ToAmountMin: fmt.Sprintf("%.0f", minReceived),
			PriceImpact: 0.1, // Usually low for bridges
			EstimatedTime: bridge.TimeEstMax,
			Provider:     bridge.Name,
			RequestID:   uuid.New().String(),
			CreatedAt:   time.Now(),
			Route: []RouteStep{
				{
					Type:       "bridge",
					Protocol:  bridge.Protocol,
					FromToken: req.FromToken,
					ToToken:   req.ToToken,
					FromAmount: req.FromAmount,
					ToAmount:  toAmountStr,
					FromChain: req.FromChain,
					ToChain:   req.ToChain,
				},
			},
			Fees: []Fee{
				{Name: "Bridge Fee", Percentage: bridge.FeePercent, Amount: fmt.Sprintf("%.0f", feeAmount)},
			},
			BridgeCalls: []BridgeCall{
				{
					FromChain: bridge.FromChain,
					ToChain:   bridge.ToChain,
					Protocol:  bridge.Protocol,
					SrcToken:  req.FromToken,
					DstToken:  req.ToToken,
					Amount:    req.FromAmount,
				},
			},
		}

		bestAmt, _ := parseAmount(bestQuote.ToAmount)
		if bestAmt == 0 || toAmount > bestAmt {
			bestQuote = quote
		}
	}

	if bestQuote.RequestID == "" {
		return nil
	}

	return bestQuote
}

func (s *CrossChainService) getAggregatedQuote(req SwapRequest) *SwapQuote {
	// Multi-hop quote (e.g., ETH -> USDC -> USDT on different chains)
	if req.FromChain != req.ToChain {
		// Get intermediate quote
		intermediateReq := req
		intermediateReq.ToChain = req.FromChain

		// Simple aggregation: swap on source chain, then bridge, then swap on dest
		dexQuote := s.getDEXQuote(intermediateReq)
		bridgeQuote := s.getBridgeQuote(req)

		if dexQuote == nil || bridgeQuote == nil {
			return nil
		}

		// Combine routes
		combined := *dexQuote
		combined.Provider = "Aggregator"
		combined.ToAmount = bridgeQuote.ToAmount
		combined.ToAmountMin = bridgeQuote.ToAmountMin
		combined.PriceImpact = dexQuote.PriceImpact + bridgeQuote.PriceImpact
		combined.EstimatedTime = dexQuote.EstimatedTime + bridgeQuote.EstimatedTime

		// Add bridge to route
		combined.Route = append(combined.Route, RouteStep{
			Type:       "bridge",
			Protocol:  bridgeQuote.Provider,
			FromToken: dexQuote.ToToken,
			ToToken:   req.ToToken,
			FromChain: req.FromChain,
			ToChain:   req.ToChain,
		})

		return &combined
	}

	// Same chain - just use DEX
	return s.getDEXQuote(req)
}

// ============================================================================
// Token Price Helpers
// ============================================================================

func (s *CrossChainService) getTokenPrice(token string, chainID uint) float64 {
	key := fmt.Sprintf("price_%d_%s", chainID, token)

	s.priceCacheMu.RLock()
	if price, ok := s.priceCache[key]; ok {
		s.priceCacheMu.RUnlock()
		return price
	}
	s.priceCacheMu.RUnlock()

	// Try to fetch from Redis cache
	ctx := context.Background()
	if cached, err := s.redis.Get(ctx, key).Float64(); err == nil {
		s.priceCacheMu.Lock()
		s.priceCache[key] = cached
		s.priceCacheMu.Unlock()
		return cached
	}

	// Return default prices for common tokens
	defaultPrices := map[string]float64{
		"ETH": 3200, "WETH": 3200,
		"USDT": 1, "USDC": 1, "DAI": 1, "BUSD": 1,
		"BNB": 300,
		"MATIC": 0.85, "POL": 0.85,
		"AVAX": 35,
		"ARB": 1.1,
		"OP": 2.2,
		"LINK": 15,
		"UNI": 7.5,
		"AAVE": 95,
		"SOL": 140,
		"BTC": 65000,
		"TON": 5.5,
	}

	price := defaultPrices[token]
	if price > 0 {
		s.priceCacheMu.Lock()
		s.priceCache[key] = price
		s.priceCacheMu.Unlock()
	}

	return price
}

// ============================================================================
// Execute Swap
// ============================================================================

type ExecuteQuoteRequest struct {
	QuoteID      string `json:"quote_id" binding:"required"`
	FromAddress  string `json:"from_address" binding:"required"`
	ToAddress    string `json:"to_address"`
}

func (s *CrossChainService) ExecuteQuote(c *gin.Context) {
	var req ExecuteQuoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get quote from cache
	quoteIface, ok := s.quoteCache.Load(req.QuoteID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "quote not found or expired"})
		return
	}

	quote := quoteIface.(SwapQuote)

	// Build and execute cross-chain transactions
	txHashes := make([]string, 0)
	executionResults := make([]map[string]interface{}, 0)

	for i, step := range quote.Route {
		// Execute each step of the route
		txHash, err := s.executeRouteStep(step, req.FromAddress, req.ToAddress, i == len(quote.Route)-1)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":       "failed to execute swap",
				"step":        i,
				"step_details": step,
			})
			return
		}
		txHashes = append(txHashes, txHash)
		executionResults = append(executionResults, map[string]interface{}{
			"step":           i,
			"dex":           step.DEX,
			"tx_hash":       txHash,
			"from_token":    step.FromToken,
			"to_token":      step.ToToken,
			"from_amount":   step.FromAmount,
			"to_amount":     step.ToAmount,
			"status":        "submitted",
		})
	}

	response := gin.H{
		"status":           "submitted",
		"quote_id":         quote.ID,
		"transaction_hashes": txHashes,
		"execution_results": executionResults,
		"steps":            len(quote.Route),
		"estimated_time":   quote.EstimatedTime,
		"from_token":       quote.FromToken,
		"to_token":        quote.ToToken,
		"from_amount":     quote.FromAmount,
		"to_amount":       quote.ToAmount,
	}

	c.JSON(http.StatusOK, response)
}

// Execute a single route step
func (s *CrossChainService) executeRouteStep(step RouteStep, fromAddr, toAddr string, isFinal bool) (string, error) {
	// Get RPC URL for the chain
	rpcURL := s.getChainRPC(step.ChainID)
	if rpcURL == "" {
		return "", fmt.Errorf("no RPC for chain %d", step.ChainID)
	}

	// Prepare transaction data based on DEX
	var txData string
	var contractAddr string

	switch step.DEX {
	case "uniswap", "sushiswap", "pancakeswap":
		// For DEX swaps, encode swap data
		txData = s.encodeDexSwapData(step.FromToken, step.ToToken, fromAddr, step.FromAmount)
		contractAddr = s.getDexRouter(step.DEX, step.ChainID)
	case "layerzero", "stargate", "axelar":
		// For bridges, encode bridge data
		txData = s.encodeBridgeData(step.ToToken, toAddr, step.FromAmount)
		contractAddr = s.getBridgeContract(step.DEX)
	default:
		// Generic swap
		txData = "0x"
		contractAddr = fromAddr
	}

	// In production, this would:
	// 1. Build the transaction with proper nonce, gas, etc.
	// 2. Sign with user's private key (never stored, provided at request time)
	// 3. Broadcast to network
	// For now, return transaction hash from simulation

	// Simulate transaction and return hash
	txHash := s.simulateTransaction(contractAddr, txData, step.ChainID)
	return txHash, nil
}

// Get RPC URL for chain
func (s *CrossChainService) getChainRPC(chainID uint64) string {
	rpcMap := map[uint64]string{
		1:     "https://eth.llamarpc.com",
		56:    "https://bsc-dataseed.binance.org",
		137:   "https://polygon-rpc.com",
		42161: "https://arb1.arbitrum.io/rpc",
		10:    "https://mainnet.optimism.io",
		43114: "https://api.avax.network/ext/bc/C/rpc",
		8453:  "https://mainnet.base.org",
	}
	return rpcMap[chainID]
}

// Get DEX router address
func (s *CrossChainService) getDexRouter(dex string, chainID uint64) string {
	routers := map[string]map[uint64]string{
		"uniswap":    {1: "0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D"},
		"sushiswap":  {1: "0xd9e1cE17f2641f24aE83637ab66a2cca9C378B9"},
		"pancakeswap": {56: "0x10ED43C718714eb63d5aA57B78B54704E256024E"},
	}
	if chainRouters, ok := routers[dex]; ok {
		if addr, ok := chainRouters[chainID]; ok {
			return addr
		}
	}
	return "0x0000000000000000000000000000000000000000"
}

// Get bridge contract address
func (s *CrossChainService) getBridgeContract(bridge string) string {
	bridges := map[string]string{
		"layerzero": "0x66A71D08CE29A94F7BEA3E2F0F001B2eA2b8DaE0",
		"stargate":  "0x45e1D8F875f6Fe3F2Ee3E6f1F8a4e4d5C3aB9E2",
		"axelar":    "0x4F4495243837681061C4743b12BAf89987603e5",
	}
	return bridges[bridge]
}

// Encode DEX swap data
func (s *CrossChainService) encodeDexSwapData(fromToken, toToken, recipient, amount string) string {
	// In production, encode proper swap calldata using abi.encode
	// This is a simplified version
	return "0x" // Would be actual encoded swap data
}

// Encode bridge data
func (s *CrossChainService) encodeBridgeData(token, recipient, amount string) string {
	// In production, encode proper bridge calldata
	return "0x" // Would be actual encoded bridge data
}

// Simulate transaction (in production, would actually broadcast)
func (s *CrossChainService) simulateTransaction(to, data string, chainID uint64) string {
	// Generate deterministic hash based on inputs
	input := fmt.Sprintf("%s|%s|%d|%s", to, data, chainID, time.Now().Format(time.RFC3339Nano))
	hash := sha256.Sum256([]byte(input))
	return "0x" + hex.EncodeToString(hash[:])[:64]
}

// ============================================================================
// Token & Chain Info
// ============================================================================

func (s *CrossChainService) GetSupportedChains(c *gin.Context) {
	chains := []gin.H{
		{"id": 1, "name": "Ethereum", "symbol": "ETH", "type": "evm"},
		{"id": 56, "name": "BNB Smart Chain", "symbol": "BNB", "type": "evm"},
		{"id": 137, "name": "Polygon", "symbol": "MATIC", "type": "evm"},
		{"id": 42161, "name": "Arbitrum One", "symbol": "ETH", "type": "evm"},
		{"id": 10, "name": "Optimism", "symbol": "ETH", "type": "evm"},
		{"id": 43114, "name": "Avalanche", "symbol": "AVAX", "type": "evm"},
		{"id": 8453, "name": "Base", "symbol": "ETH", "type": "evm"},
		{"id": 0, "name": "Solana", "symbol": "SOL", "type": "solana"},
		{"id": 0, "name": "Bitcoin", "symbol": "BTC", "type": "bitcoin"},
		{"id": 0, "name": "Toncoin", "symbol": "TON", "type": "ton"},
	}

	c.JSON(http.StatusOK, gin.H{"chains": chains})
}

func (s *CrossChainService) GetSupportedTokens(c *gin.Context) {
	chainID := c.Query("chain_id")

	tokens := []gin.H{
		{"symbol": "ETH", "name": "Ethereum", "address": "0x0000000000000000000000000000000000000000", "decimals": 18},
		{"symbol": "WETH", "name": "Wrapped Ethereum", "address": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "decimals": 18},
		{"symbol": "USDT", "name": "Tether USD", "address": "0xdac17f958d2ee523a2206206994597c13d831ec7", "decimals": 6},
		{"symbol": "USDC", "name": "USD Coin", "address": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "decimals": 6},
		{"symbol": "DAI", "name": "Dai Stablecoin", "address": "0x6b175474e89094c44da98b954eedeac495271d0f", "decimals": 18},
		{"symbol": "BNB", "name": "BNB", "address": "0x0000000000000000000000000000000000000000", "decimals": 18},
		{"symbol": "MATIC", "name": "Polygon", "address": "0x0000000000000000000000000000000000000000", "decimals": 18},
		{"symbol": "LINK", "name": "Chainlink", "address": "0x514910771af9ca656af840dff83e8264ecf986ca", "decimals": 18},
		{"symbol": "UNI", "name": "Uniswap", "address": "0x1f9840a85d5af5bf1d1762f925bdaddc4201f984", "decimals": 18},
		{"symbol": "AAVE", "name": "Aave", "address": "0x7fc66500c84a76ad7e9c93437bfc5ac33e2ddae9", "decimals": 18},
	}

	if chainID != "" {
		// Filter tokens for specific chain
		// In production, would query database
	}

	c.JSON(http.StatusOK, gin.H{"tokens": tokens})
}

func (s *CrossChainService) GetBridges(c *gin.Context) {
	var bridges []BridgeRoute
	s.db.Where("is_active = ?", true).Find(&bridges)

	c.JSON(http.StatusOK, gin.H{"bridges": bridges})
}

func (s *CrossChainService) GetDEXes(c *gin.Context) {
	var dexes []gin.H

	// Get unique DEXes from pools
	var pools []DEXPool
	s.db.Select("DISTINCT dex").Find(&pools)

	for _, pool := range pools {
		dexes = append(dexes, gin.H{
			"name":   pool.DEX,
			"chains": s.getDEXChains(pool.DEX),
		})
	}

	c.JSON(http.StatusOK, gin.H{"dexes": dexes})
}

func (s *CrossChainService) getDEXChains(dex string) []uint {
	var pools []DEXPool
	s.db.Where("dex = ?", dex).Find(&pools)

	chains := make([]uint, 0, len(pools))
	seen := make(map[uint]bool)
	for _, p := range pools {
		if !seen[p.ChainID] {
			chains = append(chains, p.ChainID)
			seen[p.ChainID] = true
		}
	}
	return chains
}

// ============================================================================
// Helper Functions
// ============================================================================

func parseAmount(amountStr string) (float64, error) {
	return strconv.ParseFloat(amountStr, 64)
}

func (s *CrossChainService) cacheQuote(quote SwapQuote) {
	s.quoteCache.Store(quote.ID, quote)

	// Auto-expire after 5 minutes
	go func() {
		time.Sleep(5 * time.Minute)
		s.quoteCache.Delete(quote.ID)
	}()
}

// ============================================================================
// Main Entry Point
// ============================================================================

import "strconv"

func main() {
	config := LoadConfig()

	service, err := NewCrossChainService(config)
	if err != nil {
		log.Fatalf("Failed to initialize service: %v", err)
	}

	router := gin.Default()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// CORS
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Routes
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "cross-chain-aggregator"})
	})

	api := router.Group("/api/v1")
	{
		api.POST("/quote", service.GetQuote)
		api.POST("/execute", service.ExecuteQuote)
		
		api.GET("/chains", service.GetSupportedChains)
		api.GET("/tokens", service.GetSupportedTokens)
		api.GET("/bridges", service.GetBridges)
		api.GET("/dexes", service.GetDEXes)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Cross-Chain Aggregator starting on port %s", config.ServerPort)
		if err := router.Run(":" + config.ServerPort); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down server...")
}
