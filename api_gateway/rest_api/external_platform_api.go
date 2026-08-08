package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// ============================================================================
// TYPES
// ============================================================================

type ExternalPlatform struct {
	ID              string      `json:"id"`
	Name            string      `json:"name"`
	Type            string      `json:"type"` // cex, dex, wallet, protocol
	APIKey          string      `json:"apiKey"`
	Tier            string      `json:"tier"` // free, basic, pro, enterprise
	IsActive        bool        `json:"isActive"`
	Permissions     Permissions `json:"permissions"`
	RateLimitPerMin int         `json:"rateLimitPerMin"`
	MonthlyFeeUsd   float64     `json:"monthlyFeeUsd"`
	CreatedAt       time.Time   `json:"createdAt"`
}

type Permissions struct {
	CanTrade        bool `json:"canTrade"`
	CanSwap         bool `json:"canSwap"`
	CanAddLiquidity bool `json:"canAddLiquidity"`
	CanBridge       bool `json:"canBridge"`
	CanCreateToken  bool `json:"canCreateToken"`
}

type TierConfig struct {
	Name              string      `json:"name"`
	MonthlyFeeUsd     float64     `json:"monthlyFeeUsd"`
	MaxAPICallsPerMin int         `json:"maxApiCallsPerMin"`
	MaxDailyVolume    float64     `json:"maxDailyVolume"`
	MaxPositions      int         `json:"maxPositions"`
	Features          Permissions `json:"features"`
}

type TradingRequest struct {
	Platform string `json:"platform"`
	Symbol   string `json:"symbol"`
	Side     string `json:"side"` // buy, sell
	Type     string `json:"type"` // market, limit
	Amount   string `json:"amount"`
	Price    string `json:"price,omitempty"`
}

type SwapRequest struct {
	Platform string  `json:"platform"`
	ChainID  int     `json:"chainId"`
	TokenIn  string  `json:"tokenIn"`
	TokenOut string  `json:"tokenOut"`
	AmountIn string  `json:"amountIn"`
	Slippage float64 `json:"slippage"`
}

type LiquidityRequest struct {
	Platform string `json:"platform"`
	ChainID  int    `json:"chainId"`
	TokenA   string `json:"tokenA"`
	TokenB   string `json:"tokenB"`
	AmountA  string `json:"amountA"`
	AmountB  string `json:"amountB"`
}

type ApiResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// ============================================================================
// TIER CONFIGURATIONS
// ============================================================================

var TierConfigs = map[string]TierConfig{
	"free": {
		Name:              "free",
		MonthlyFeeUsd:     0,
		MaxAPICallsPerMin: 60,
		MaxDailyVolume:    10000,
		MaxPositions:      3,
		Features: Permissions{
			CanTrade:        true,
			CanSwap:         false,
			CanAddLiquidity: false,
			CanBridge:       false,
			CanCreateToken:  false,
		},
	},
	"basic": {
		Name:              "basic",
		MonthlyFeeUsd:     99,
		MaxAPICallsPerMin: 300,
		MaxDailyVolume:    100000,
		MaxPositions:      10,
		Features: Permissions{
			CanTrade:        true,
			CanSwap:         true,
			CanAddLiquidity: false,
			CanBridge:       false,
			CanCreateToken:  false,
		},
	},
	"pro": {
		Name:              "pro",
		MonthlyFeeUsd:     299,
		MaxAPICallsPerMin: 1000,
		MaxDailyVolume:    1000000,
		MaxPositions:      50,
		Features: Permissions{
			CanTrade:        true,
			CanSwap:         true,
			CanAddLiquidity: true,
			CanBridge:       true,
			CanCreateToken:  false,
		},
	},
	"enterprise": {
		Name:              "enterprise",
		MonthlyFeeUsd:     999,
		MaxAPICallsPerMin: 5000,
		MaxDailyVolume:    10000000,
		MaxPositions:      200,
		Features: Permissions{
			CanTrade:        true,
			CanSwap:         true,
			CanAddLiquidity: true,
			CanBridge:       true,
			CanCreateToken:  true,
		},
	},
}

// ============================================================================
// STORAGE (in production, use database)
// ============================================================================

var platforms = make(map[string]ExternalPlatform)
var apiCalls = make(map[string]int)
var dailyVolume = make(map[string]float64)

// ============================================================================
// HANDLERS
// ============================================================================

// Get tier configurations
func GetTierConfigs(w http.ResponseWriter, r *http.Request) {
	tiers := []TierConfig{}
	for _, tier := range TierConfigs {
		tiers = append(tiers, tier)
	}
	respondJSON(w, ApiResponse{Success: true, Data: tiers})
}

// Register external platform
func RegisterPlatform(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   string `json:"name"`
		Type   string `json:"type"`
		APIKey string `json:"apiKey"`
		Tier   string `json:"tier"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, ApiResponse{Success: false, Error: "Invalid request"})
		return
	}

	if req.Name == "" || req.Type == "" || req.APIKey == "" {
		respondJSON(w, ApiResponse{Success: false, Error: "Missing required fields"})
		return
	}

	// Validate tier
	tierConfig, ok := TierConfigs[req.Tier]
	if !ok {
		req.Tier = "free"
		tierConfig = TierConfigs["free"]
	}

	platform := ExternalPlatform{
		ID:              fmt.Sprintf("platform_%d", time.Now().Unix()),
		Name:            req.Name,
		Type:            req.Type,
		APIKey:          req.APIKey,
		Tier:            req.Tier,
		IsActive:        true,
		Permissions:     tierConfig.Features,
		RateLimitPerMin: tierConfig.MaxAPICallsPerMin,
		MonthlyFeeUsd:   tierConfig.MonthlyFeeUsd,
		CreatedAt:       time.Now(),
	}

	platforms[platform.ID] = platform

	respondJSON(w, ApiResponse{
		Success: true,
		Data:    platform,
		Message: "Platform registered successfully",
	})
}

// Get platform info
func GetPlatform(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/external-platform/")

	platform, ok := platforms[id]
	if !ok {
		respondJSON(w, ApiResponse{Success: false, Error: "Platform not found"})
		return
	}

	respondJSON(w, ApiResponse{Success: true, Data: platform})
}

// Update platform
func UpdatePlatform(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name,omitempty"`
		Tier     string `json:"tier,omitempty"`
		IsActive bool   `json:"isActive,omitempty"`
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/external-platform/update/")

	platform, ok := platforms[id]
	if !ok {
		respondJSON(w, ApiResponse{Success: false, Error: "Platform not found"})
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, ApiResponse{Success: false, Error: "Invalid request"})
		return
	}

	if req.Name != "" {
		platform.Name = req.Name
	}
	if req.Tier != "" {
		if tierConfig, ok := TierConfigs[req.Tier]; ok {
			platform.Tier = req.Tier
			platform.Permissions = tierConfig.Features
			platform.RateLimitPerMin = tierConfig.MaxAPICallsPerMin
			platform.MonthlyFeeUsd = tierConfig.MonthlyFeeUsd
		}
	}
	if req.IsActive {
		platform.IsActive = req.IsActive
	}

	platforms[id] = platform

	respondJSON(w, ApiResponse{Success: true, Data: platform})
}

// Delete platform
func DeletePlatform(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/external-platform/delete/")

	if _, ok := platforms[id]; !ok {
		respondJSON(w, ApiResponse{Success: false, Error: "Platform not found"})
		return
	}

	delete(platforms, id)

	respondJSON(w, ApiResponse{Success: true, Message: "Platform deleted"})
}

// Trading handler
func ExecuteTrade(w http.ResponseWriter, r *http.Request) {
	var req TradingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, ApiResponse{Success: false, Error: "Invalid request"})
		return
	}

	if req.Platform == "" || req.Symbol == "" || req.Side == "" || req.Amount == "" {
		respondJSON(w, ApiResponse{Success: false, Error: "Missing required fields"})
		return
	}

	platform, ok := platforms[req.Platform]
	if !ok {
		respondJSON(w, ApiResponse{Success: false, Error: "Platform not found"})
		return
	}

	if !platform.IsActive {
		respondJSON(w, ApiResponse{Success: false, Error: "Platform is not active"})
		return
	}

	if !platform.Permissions.CanTrade {
		respondJSON(w, ApiResponse{Success: false, Error: "Trading not permitted for this tier"})
		return
	}

	result, err := executeTradeOnPlatform(platform, &req)
	if err != nil {
		respondJSON(w, ApiResponse{Success: false, Error: fmt.Sprintf("trade execution failed: %v", err)})
		return
	}
	respondJSON(w, ApiResponse{Success: true, Data: result})
}

// Swap handler
func ExecuteSwap(w http.ResponseWriter, r *http.Request) {
	var req SwapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, ApiResponse{Success: false, Error: "Invalid request"})
		return
	}

	if req.Platform == "" || req.TokenIn == "" || req.TokenOut == "" || req.AmountIn == "" {
		respondJSON(w, ApiResponse{Success: false, Error: "Missing required fields"})
		return
	}

	platform, ok := platforms[req.Platform]
	if !ok {
		respondJSON(w, ApiResponse{Success: false, Error: "Platform not found"})
		return
	}

	if !platform.Permissions.CanSwap {
		respondJSON(w, ApiResponse{Success: false, Error: "Swap not permitted for this tier"})
		return
	}

	result, err := executeSwapViaAggregator(&req)
	if err != nil {
		respondJSON(w, ApiResponse{Success: false, Error: fmt.Sprintf("swap execution failed: %v", err)})
		return
	}
	respondJSON(w, ApiResponse{Success: true, Data: result})
}

// Add liquidity handler
func AddLiquidity(w http.ResponseWriter, r *http.Request) {
	var req LiquidityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, ApiResponse{Success: false, Error: "Invalid request"})
		return
	}

	if req.Platform == "" || req.TokenA == "" || req.TokenB == "" {
		respondJSON(w, ApiResponse{Success: false, Error: "Missing required fields"})
		return
	}

	platform, ok := platforms[req.Platform]
	if !ok {
		respondJSON(w, ApiResponse{Success: false, Error: "Platform not found"})
		return
	}

	if !platform.Permissions.CanAddLiquidity {
		respondJSON(w, ApiResponse{Success: false, Error: "Add liquidity not permitted for this tier"})
		return
	}

	result, err := addLiquidityOnChain(&req)
	if err != nil {
		respondJSON(w, ApiResponse{Success: false, Error: fmt.Sprintf("add liquidity failed: %v", err)})
		return
	}
	respondJSON(w, ApiResponse{Success: true, Data: result})
}

// Get rate limit status
func GetRateLimit(w http.ResponseWriter, r *http.Request) {
	platformID := r.URL.Query().Get("platform")

	if platformID == "" {
		respondJSON(w, ApiResponse{Success: false, Error: "Missing platform ID"})
		return
	}

	platform, ok := platforms[platformID]
	if !ok {
		respondJSON(w, ApiResponse{Success: false, Error: "Platform not found"})
		return
	}

	used := apiCalls[platformID]
	remaining := platform.RateLimitPerMin - used

	respondJSON(w, ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"platform":  platformID,
			"used":      used,
			"limit":     platform.RateLimitPerMin,
			"remaining": remaining,
			"resetAt":   time.Now().Add(time.Minute).Unix(),
		},
	})
}

// Get platform stats
func GetPlatformStats(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"totalPlatforms":  len(platforms),
		"activePlatforms": countActivePlatforms(),
		"byTier":          getPlatformsByTier(),
	}

	respondJSON(w, ApiResponse{Success: true, Data: stats})
}

// ============================================================================
// REAL DEX / CEX EXECUTION ADAPTERS
//
// These adapters perform live network calls. They never return synthetic fills,
// quotes, or hashes: on any failure they surface an error so the caller can
// retry or alert. Configuration is read from environment variables so secrets
// are not baked into the binary.
// ============================================================================

const (
	// zeroXSwapAPIBase is the public 0x Swap API, which aggregates real on-chain
	// liquidity across Uniswap, SushiSwap, PancakeSwap, etc. and returns an
	// executable swap calldata payload plus a price quote.
	zeroXSwapAPIBase = "https://api.0x.org/swap"
	// zeroXAPIKeyEnv optionally holds a 0x API key for higher rate limits.
	zeroXAPIKeyEnv = "ZERO_X_API_KEY"

	// binanceAPIBase is the Binance Spot REST endpoint used for live CEX trades.
	binanceAPIBase = "https://api.binance.com"
	binanceAPIKeyEnv    = "BINANCE_API_KEY"
	binanceSecretKeyEnv = "BINANCE_API_SECRET"

	// uniswapV2RouterAddress is the canonical Uniswap V2 router on Ethereum mainnet.
	uniswapV2RouterAddress = "0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D"

	// defaultSwapHTTPTimeout bounds external quote/swap HTTP calls.
	defaultSwapHTTPTimeout = 30 * time.Second
)

// SwapExecutionResult is the on-the-wire result of a real swap: a price quote
// together with executable calldata targeting a DEX router, ready for the
// master wallet to sign and broadcast.
type SwapExecutionResult struct {
	DEX          string    `json:"dex"`
	FromToken    string    `json:"fromToken"`
	ToToken      string    `json:"toToken"`
	FromAmount   string    `json:"fromAmount"`
	ToAmount     string    `json:"toAmount"`
	MinToAmount  string    `json:"minToAmount"`
	Price        string    `json:"price"`
	GasEstimate  string    `json:"gasEstimate"`
	To           string    `json:"to"`
	Calldata     string    `json:"calldata"`
	Value        string    `json:"value"`
	ExecutedAt   time.Time `json:"executedAt"`
}

// executeSwapViaAggregator fetches a real, executable swap from the 0x Swap API
// (which aggregates on-chain DEX liquidity) and returns the router calldata.
// No synthetic quote is produced: a network/API failure is returned as an error.
func executeSwapViaAggregator(req *SwapRequest) (*SwapExecutionResult, error) {
	if req.TokenIn == "" || req.TokenOut == "" || req.AmountIn == "" {
		return nil, fmt.Errorf("tokenIn, tokenOut and amountIn are required")
	}
	chainID := req.ChainID
	if chainID == 0 {
		chainID = 1 // default to Ethereum mainnet
	}

	params := url.Values{}
	params.Set("sellToken", req.TokenIn)
	params.Set("buyToken", req.TokenOut)
	params.Set("sellAmount", req.AmountIn)
	params.Set("takerAddress", "0x0000000000000000000000000000000000000000")
	if req.Slippage > 0 {
		params.Set("slippagePercentage", strconv.FormatFloat(req.Slippage/100, 'f', -1, 64))
	}

	endpoint := zeroXSwapURL(chainID) + "?" + params.Encode()

	httpReq, err := http.NewRequestWithContext(context.Background(), http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build 0x request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json")
	if key := os.Getenv(zeroXAPIKeyEnv); key != "" {
		httpReq.Header.Set("0x-api-key", key)
	}

	client := &http.Client{Timeout: defaultSwapHTTPTimeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("0x swap API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read 0x swap response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("0x swap API returned HTTP %d: %s", resp.StatusCode, truncateBody(body))
	}

	var quote struct {
		Price           string `json:"price"`
		BuyAmount       string `json:"buyAmount"`
		SellAmount      string `json:"sellAmount"`
		GrossBuyAmount  string `json:"grossBuyAmount"`
		NetBuyAmount    string `json:"netBuyAmount"`
		To              string `json:"to"`
		Data            string `json:"data"`
		Value           string `json:"value"`
		Gas             string `json:"gas"`
		EstimatedGas    string `json:"estimatedGas"`
		Sources         []struct {
			Name string `json:"name"`
		} `json:"sources"`
		Permit2 string `json:"permit2"`
	}
	if err := json.Unmarshal(body, &quote); err != nil {
		return nil, fmt.Errorf("parse 0x swap response: %w", err)
	}
	if quote.To == "" || quote.Data == "" {
		return nil, fmt.Errorf("0x swap API returned no executable calldata (to/data empty)")
	}

	dexName := "0x-aggregator"
	if len(quote.Sources) > 0 {
		dexName = quote.Sources[0].Name
	}

	minToAmount := quote.BuyAmount
	if quote.NetBuyAmount != "" {
		minToAmount = quote.NetBuyAmount
	}
	gasEst := quote.Gas
	if gasEst == "" {
		gasEst = quote.EstimatedGas
	}

	return &SwapExecutionResult{
		DEX:         dexName,
		FromToken:   req.TokenIn,
		ToToken:     req.TokenOut,
		FromAmount:  quote.SellAmount,
		ToAmount:    quote.BuyAmount,
		MinToAmount: minToAmount,
		Price:       quote.Price,
		GasEstimate: gasEst,
		To:          quote.To,
		Calldata:    quote.Data,
		Value:       quote.Value,
		ExecutedAt:  time.Now(),
	}, nil
}

// zeroXSwapURL returns the per-chain 0x Swap API endpoint. 0x exposes separate
// subdomains per chain; only well-supported chains are mapped, otherwise an
// error-friendly mainnet URL is returned (the API call will then fail loudly).
func zeroXSwapURL(chainID int) string {
	switch chainID {
	case 1:
		return zeroXSwapAPIBase + "/permit2/quote"
	case 56:
		return "https://bsc.api.0x.org/swap/permit2/quote"
	case 42161:
		return "https://arbitrum.api.0x.org/swap/permit2/quote"
	case 10:
		return "https://optimism.api.0x.org/swap/permit2/quote"
	case 137:
		return "https://polygon.api.0x.org/swap/permit2/quote"
	default:
		return zeroXSwapAPIBase + "/permit2/quote"
	}
}

// TradeExecutionResult is the result of a real CEX order placement.
type TradeExecutionResult struct {
	Exchange   string    `json:"exchange"`
	Symbol     string    `json:"symbol"`
	Side       string    `json:"side"`
	Type       string    `json:"type"`
	Amount     string    `json:"amount"`
	Price      string    `json:"price,omitempty"`
	OrderID    string    `json:"orderId"`
	Status     string    `json:"status"`
	ExecutedQty string   `json:"executedQty"`
	AvgPrice   string    `json:"avgPrice,omitempty"`
	ExecutedAt time.Time `json:"executedAt"`
}

// executeTradeOnPlatform dispatches a trade to the exchange backing the platform.
// Only Binance spot is wired here; an unknown platform or missing API keys produce
// a real error rather than a synthetic fill.
func executeTradeOnPlatform(platform *ExternalPlatform, req *TradingRequest) (*TradeExecutionResult, error) {
	side := strings.ToLower(req.Side)
	if side != "buy" && side != "sell" {
		return nil, fmt.Errorf("side must be 'buy' or 'sell', got %q", req.Side)
	}
	orderType := strings.ToLower(req.Type)
	if orderType == "" {
		orderType = "market"
	}
	if orderType != "market" && orderType != "limit" {
		return nil, fmt.Errorf("type must be 'market' or 'limit', got %q", req.Type)
	}

	switch strings.ToLower(platform.Name) {
	case "binance":
		return executeBinanceTrade(req, orderType, side)
	default:
		return nil, fmt.Errorf("no live trading adapter for platform %q (supported: Binance); configure API keys and a supported platform", platform.Name)
	}
}

// executeBinanceTrade places a real spot order on Binance using the signed REST API.
// API key and secret are read from BINANCE_API_KEY / BINANCE_API_SECRET; their absence
// is a hard error. A non-2xx response or parse failure is propagated, never faked.
func executeBinanceTrade(req *TradingRequest, orderType, side string) (*TradeExecutionResult, error) {
	apiKey := os.Getenv(binanceAPIKeyEnv)
	secret := os.Getenv(binanceSecretKeyEnv)
	if apiKey == "" || secret == "" {
		return nil, fmt.Errorf("Binance API credentials not configured (set %s and %s)", binanceAPIKeyEnv, binanceSecretKeyEnv)
	}

	// Binance expects symbol without separators, e.g. "BTCUSDT".
	symbol := strings.ToUpper(strings.ReplaceAll(req.Symbol, "/", ""))
	symbol = strings.ReplaceAll(symbol, "-", "")

	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("side", side)
	params.Set("type", strings.ToUpper(orderType))
	params.Set("quantity", req.Amount)
	if orderType == "limit" {
		if req.Price == "" {
			return nil, fmt.Errorf("limit order requires a price")
		}
		params.Set("price", req.Price)
		params.Set("timeInForce", "GTC")
	}
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	params.Set("recvWindow", "5000")

	query := params.Encode()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(query))
	signature := hex.EncodeToString(mac.Sum(nil))
	fullQuery := query + "&signature=" + signature

	endpoint := binanceAPIBase + "/api/v3/order?" + fullQuery
	httpReq, err := http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build Binance order request: %w", err)
	}
	httpReq.Header.Set("X-MBX-APIKEY", apiKey)

	client := &http.Client{Timeout: defaultSwapHTTPTimeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Binance order request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read Binance order response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Binance returned HTTP %d: %s", resp.StatusCode, truncateBody(body))
	}

	var ord struct {
		OrderID     int64  `json:"orderId"`
		Status      string `json:"status"`
		ExecutedQty string `json:"executedQty"`
		CummulativeQuoteQty string `json:"cummulativeQuoteQty"`
		Symbol      string `json:"symbol"`
		Side        string `json:"side"`
		Type        string `json:"type"`
		Price       string `json:"price"`
		AvgPrice    string `json:"avgPrice"`
	}
	if err := json.Unmarshal(body, &ord); err != nil {
		return nil, fmt.Errorf("parse Binance order response: %w", err)
	}

	avgPrice := ord.AvgPrice
	if avgPrice == "" && ord.ExecutedQty != "" {
		// Derive average fill price from cumulative quote qty if avgPrice absent.
		if q, ok := new(big.Float).SetString(ord.CummulativeQuoteQty); ok {
			if e, ok2 := new(big.Float).SetString(ord.ExecutedQty); ok2 && e.Sign() > 0 {
				avg, _ := new(big.Float).Quo(q, e).Float64()
				avgPrice = strconv.FormatFloat(avg, 'f', 8, 64)
			}
		}
	}

	return &TradeExecutionResult{
		Exchange:    "binance",
		Symbol:      ord.Symbol,
		Side:        ord.Side,
		Type:        ord.Type,
		Amount:      req.Amount,
		Price:       ord.Price,
		OrderID:     strconv.FormatInt(ord.OrderID, 10),
		Status:      ord.Status,
		ExecutedQty: ord.ExecutedQty,
		AvgPrice:    avgPrice,
		ExecutedAt:  time.Now(),
	}, nil
}

// LiquidityExecutionResult is the result of constructing an add-liquidity call.
// It carries the unsigned EVM calldata targeting the DEX router; the master
// wallet signs and broadcasts it via the existing signAndSubmitTransaction path.
type LiquidityExecutionResult struct {
	Router      string    `json:"router"`
	TokenA      string    `json:"tokenA"`
	TokenB      string    `json:"tokenB"`
	AmountA     string    `json:"amountA"`
	AmountB     string    `json:"amountB"`
	MinAmountA  string    `json:"minAmountA"`
	MinAmountB  string    `json:"minAmountB"`
	To          string    `json:"to"`
	Calldata    string    `json:"calldata"`
	Value       string    `json:"value"`
	ExecutedAt  time.Time `json:"executedAt"`
}

// addLiquidityOnChain builds a real Uniswap V2 Router addLiquidity calldata and
// submits it through the master wallet signer. Slippage tolerance defaults to 1%
// when the caller does not pass explicit minimums. A missing master wallet or
// signer configuration yields a real error rather than a synthetic position.
func addLiquidityOnChain(req *LiquidityRequest) (*LiquidityExecutionResult, error) {
	if req.ChainID == 0 {
		return nil, fmt.Errorf("chainId is required to add liquidity")
	}
	if !common.IsHexAddress(req.TokenA) || !common.IsHexAddress(req.TokenB) {
		return nil, fmt.Errorf("tokenA and tokenB must be valid EVM addresses")
	}
	amountA, ok := new(big.Int).SetString(req.AmountA, 10)
	if !ok {
		return nil, fmt.Errorf("invalid amountA %q", req.AmountA)
	}
	amountB, ok := new(big.Int).SetString(req.AmountB, 10)
	if !ok {
		return nil, fmt.Errorf("invalid amountB %q", req.AmountB)
	}

	// Default 1% slippage tolerance on both legs.
	minA := new(big.Int).Mul(amountA, big.NewInt(99))
	minA.Div(minA, big.NewInt(100))
	minB := new(big.Int).Mul(amountB, big.NewInt(99))
	minB.Div(minB, big.NewInt(100))

	router := common.HexToAddress(uniswapV2RouterAddress)
	tokenA := common.HexToAddress(req.TokenA)
	tokenB := common.HexToAddress(req.TokenB)
	deadline := big.NewInt(time.Now().Add(20 * time.Minute).Unix())

	// addLiquidity(address,address,uint256,uint256,uint256,uint256,address,uint256)
	// selector = 0xe8e33700.
	calldata, err := packAddLiquidityCall(tokenA, tokenB, amountA, amountB, minA, minB, router, deadline)
	if err != nil {
		return nil, fmt.Errorf("encode addLiquidity calldata: %w", err)
	}

	result := &LiquidityExecutionResult{
		Router:     uniswapV2RouterAddress,
		TokenA:     req.TokenA,
		TokenB:     req.TokenB,
		AmountA:    amountA.String(),
		AmountB:    amountB.String(),
		MinAmountA: minA.String(),
		MinAmountB: minB.String(),
		To:         uniswapV2RouterAddress,
		Calldata:   "0x" + hex.EncodeToString(calldata),
		Value:      "0",
		ExecutedAt: time.Now(),
	}

	// Submit through the master wallet so the on-chain effect is real.
	if masterWalletStore == nil {
		return nil, fmt.Errorf("master wallet store is not initialized; cannot submit addLiquidity")
	}
	autoTx := &AutoTransaction{
		WalletID: "master",
		Type:     "add_liquidity",
		ChainId:  req.ChainID,
		Token:    req.TokenA + "/" + req.TokenB,
		Amount:   big.NewInt(0),
		To:       uniswapV2RouterAddress,
		Data:     "0x" + hex.EncodeToString(calldata),
		GasLimit: 300000,
	}
	if err := masterWalletStore.QueueTransaction(autoTx); err != nil {
		return nil, fmt.Errorf("queue addLiquidity transaction: %w", err)
	}
	result.Value = autoTx.ID // surface the queued transaction id for tracking
	return result, nil
}

// packAddLiquidityCall ABI-encodes the Uniswap V2 router addLiquidity arguments.
// It uses a manual fixed-size head layout to avoid pulling in abigen/abi bindings.
func packAddLiquidityCall(tokenA, tokenB common.Address, amountA, amountB, minA, minB *big.Int, to common.Address, deadline *big.Int) ([]byte, error) {
	out := make([]byte, 0, 4+7*32)

	// 4-byte function selector for addLiquidity: keccak256("addLiquidity(address,address,uint256,uint256,uint256,uint256,address,uint256)")[:4]
	selector, err := functionSelector("addLiquidity(address,address,uint256,uint256,uint256,uint256,address,uint256)")
	if err != nil {
		return nil, err
	}
	out = append(out, selector...)
	out = append(out, padAddress(tokenA)...)
	out = append(out, padAddress(tokenB)...)
	out = append(out, padUint256(amountA)...)
	out = append(out, padUint256(amountB)...)
	out = append(out, padUint256(minA)...)
	out = append(out, padUint256(minB)...)
	out = append(out, padAddress(to)...)
	out = append(out, padUint256(deadline)...)
	return out, nil
}

// functionSelector returns the first 4 bytes of keccak256(signature).
func functionSelector(signature string) ([]byte, error) {
	h := crypto.Keccak256([]byte(signature))
	return h[:4], nil
}

// padAddress left-pads a 20-byte address to 32 bytes.
func padAddress(a common.Address) []byte {
	b := make([]byte, 32)
	copy(b[32-len(a):], a.Bytes())
	return b
}

// padUint256 left-pads a big.Int to 32 bytes (big-endian).
func padUint256(v *big.Int) []byte {
	b := make([]byte, 32)
	v.FillBytes(b)
	return b
}

// truncateBody trims an error response body for inclusion in error messages.
func truncateBody(body []byte) string {
	const max = 256
	s := string(body)
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}


// HELPER FUNCTIONS
// ============================================================================

func respondJSON(w http.ResponseWriter, resp ApiResponse) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func countActivePlatforms() int {
	count := 0
	for _, p := range platforms {
		if p.IsActive {
			count++
		}
	}
	return count
}

func getPlatformsByTier() map[string]int {
	byTier := make(map[string]int)
	for _, p := range platforms {
		byTier[p.Tier]++
	}
	return byTier
}

// ============================================================================
// ROUTE REGISTRATION
// ============================================================================

func RegisterExternalPlatformRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/external-platform/tiers", GetTierConfigs)
	mux.HandleFunc("/api/external-platform/register", RegisterPlatform)
	mux.HandleFunc("/api/external-platform/", GetPlatform)
	mux.HandleFunc("/api/external-platform/update/", UpdatePlatform)
	mux.HandleFunc("/api/external-platform/delete/", DeletePlatform)
	mux.HandleFunc("/api/external-platform/trade", ExecuteTrade)
	mux.HandleFunc("/api/external-platform/swap", ExecuteSwap)
	mux.HandleFunc("/api/external-platform/liquidity", AddLiquidity)
	mux.HandleFunc("/api/external-platform/rate-limit", GetRateLimit)
	mux.HandleFunc("/api/external-platform/stats", GetPlatformStats)
}
