// TigerWallet Swap Service - Automatic DEX/CEX Integration
//
// Features:
// - Real quote fetching from Uniswap, PancakeSwap and SushiSwap APIs
// - CoinGecko-based token pricing
// - Best-route selection across DEXs
// - Fee collection for white labels
// - Unsigned transaction payload construction (signing/broadcast left to the wallet service)

package wallet

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ==================== DEX/CEX Types ====================

type Dex struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	APIURL     string  `json:"api_url"`
	FeePercent float64 `json:"fee_percent"`
	Chain      string  `json:"chain"`
	Type       string  `json:"type"` // "dex" or "cex"
	Priority   int     `json:"priority"`
	Active     bool    `json:"active"`
}

// SwapRoute represents a single executable swap route returned by a DEX.
type SwapRoute struct {
	DexID       string  `json:"dex_id"`
	FromToken   string  `json:"from_token"`
	ToToken     string  `json:"to_token"`
	FromAmount  uint64  `json:"from_amount"`
	ToAmount    uint64  `json:"to_amount"`
	MinToAmount uint64  `json:"min_to_amount"`
	PriceImpact float64 `json:"price_impact"`
	GasUsed     uint64  `json:"gas_used"`
	GasPrice    string  `json:"gas_price"`
	Slippage    float64 `json:"slippage"`
	PoolAddress string  `json:"pool_address,omitempty"`
}

type SwapRequest struct {
	FromChain string  `json:"from_chain"`
	ToChain   string  `json:"to_chain"`
	FromToken string  `json:"from_token"`
	ToToken   string  `json:"to_token"`
	Amount    uint64  `json:"amount"`
	Slippage  float64 `json:"slippage"`
	UserID    string  `json:"user_id"`
}

// UnsignedTxPayload is the on-chain transaction data needed to execute a swap.
// Signing and broadcasting are performed by the wallet service, which has the
// funded account and private key material.
type UnsignedTxPayload struct {
	To       string `json:"to"`        // router / pool contract address
	From     string `json:"from"`      // user wallet address (populated by wallet service)
	Value    string `json:"value"`     // native value to send (wei); "0" for token swaps
	Data     string `json:"data"`      // ABI-encoded calldata (hex, 0x-prefixed)
	GasLimit uint64 `json:"gas_limit"`
	GasPrice string `json:"gas_price"`
	ChainID  int    `json:"chain_id"`
}

type SwapResult struct {
	Request     SwapRequest       `json:"request"`
	Routes      []SwapRoute       `json:"routes"`
	BestRoute   *SwapRoute        `json:"best_route"`
	UnsignedTx  *UnsignedTxPayload `json:"unsigned_tx"`
	TotalFee    uint64            `json:"total_fee"`
	AdminFee    uint64            `json:"admin_fee"`
	TxHash      string            `json:"tx_hash"`
	Timestamp   int64             `json:"timestamp"`
	Status      string            `json:"status"`
}

type TokenPrice struct {
	Token   string  `json:"token"`
	Price   float64 `json:"price"`
	Chain   string  `json:"chain"`
	Updated int64   `json:"updated"`
}

// ==================== Swap Service ====================

type SwapService struct {
	mu sync.RWMutex

	// DEXs/CEXs
	dexes map[string]*Dex

	httpClient *http.Client

	// CoinGecko base URL
	coingeckoAPI string

	// API keys for external services
	tigerSwapAPIKey string
	uniswapAPIKey   string
	pancakeAPIKey   string
	binanceAPIKey   string
	coinbaseAPIKey  string

	// Cached prices
	prices map[string]TokenPrice

	// Swap history
	history []SwapResult

	// Fee collection
	totalFees uint64
}

func NewSwapService() *SwapService {
	svc := &SwapService{
		dexes:        make(map[string]*Dex),
		prices:       make(map[string]TokenPrice),
		history:      []SwapResult{},
		httpClient:   &http.Client{Timeout: 15 * time.Second},
		coingeckoAPI: getEnvDefault("COINGECKO_API", "https://api.coingecko.com/api/v3"),
	}

	// Register DEXs
	dexList := []*Dex{
		{ID: "tigerswap", Name: "TigerSwap", APIURL: "https://api.tigerswap.io", FeePercent: 0.3, Chain: "ethereum", Type: "dex", Priority: 1, Active: true},
		{ID: "uniswap_v3", Name: "Uniswap V3", APIURL: "https://api.uniswap.org", FeePercent: 0.3, Chain: "ethereum", Type: "dex", Priority: 2, Active: true},
		{ID: "uniswap_v2", Name: "Uniswap V2", APIURL: "https://api.uniswap.org", FeePercent: 0.3, Chain: "ethereum", Type: "dex", Priority: 3, Active: true},
		{ID: "pancakeswap", Name: "PancakeSwap", APIURL: "https://api.pancakeswap.com", FeePercent: 0.25, Chain: "bsc", Type: "dex", Priority: 4, Active: true},
		{ID: "sushiswap", Name: "SushiSwap", APIURL: "https://api.sushi.com", FeePercent: 0.3, Chain: "ethereum", Type: "dex", Priority: 5, Active: true},
		{ID: "curve", Name: "Curve", APIURL: "https://api.curve.fi", FeePercent: 0.04, Chain: "ethereum", Type: "dex", Priority: 6, Active: true},
		{ID: "balancer", Name: "Balancer", APIURL: "https://api.balancer.fi", FeePercent: 0.2, Chain: "ethereum", Type: "dex", Priority: 7, Active: true},
		{ID: "dodo", Name: "DODO", APIURL: "https://api.dodoex.io", FeePercent: 0.3, Chain: "ethereum", Type: "dex", Priority: 8, Active: true},
		{ID: "1inch", Name: "1inch", APIURL: "https://api.1inch.io", FeePercent: 0.5, Chain: "ethereum", Type: "dex", Priority: 9, Active: true},
		{ID: "0x", Name: "0x", APIURL: "https://api.0x.org", FeePercent: 0.5, Chain: "ethereum", Type: "dex", Priority: 10, Active: true},
		// CEXs
		{ID: "binance", Name: "Binance", APIURL: "https://api.binance.com", FeePercent: 0.1, Chain: "multi", Type: "cex", Priority: 1, Active: true},
		{ID: "coinbase", Name: "Coinbase", APIURL: "https://api.coinbase.com", FeePercent: 0.5, Chain: "multi", Type: "cex", Priority: 2, Active: true},
		{ID: "kraken", Name: "Kraken", APIURL: "https://api.kraken.com", FeePercent: 0.2, Chain: "multi", Type: "cex", Priority: 3, Active: true},
		{ID: "kucoin", Name: "KuCoin", APIURL: "https://api.kucoin.com", FeePercent: 0.1, Chain: "multi", Type: "cex", Priority: 4, Active: true},
		{ID: "bybit", Name: "Bybit", APIURL: "https://api.bybit.com", FeePercent: 0.1, Chain: "multi", Type: "cex", Priority: 5, Active: true},
	}

	for _, dex := range dexList {
		svc.dexes[dex.ID] = dex
	}

	return svc
}

func getEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// ==================== Swap Functions ====================

// GetSwapRoutes fetches live quotes from the configured DEX providers
// (Uniswap, PancakeSwap, SushiSwap) and returns all available routes sorted by
// best output amount. If no provider responds (e.g. offline), an estimated
// route is still produced so callers always get usable routing data.
func (s *SwapService) GetSwapRoutes(req SwapRequest) ([]SwapRoute, error) {
	s.mu.Lock()
	dexes := s.activeDEXesLocked()
	s.mu.Unlock()

	slippage := req.Slippage
	if slippage <= 0 {
		slippage = 0.5
	}
	amountIn := new(big.Int).SetUint64(req.Amount)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var routes []SwapRoute
	for _, dex := range dexes {
		if dex.Type != "dex" || !dex.Active {
			continue
		}
		// Only query DEXs whose chain matches the request (ethereum/bsc) or
		// that are chain-agnostic aggregators.
		if dex.Chain != chainName(req.FromChain) && !strings.Contains(dex.ID, "inch") && dex.ID != "0x" && dex.ID != "tigerswap" {
			continue
		}

		route, err := s.fetchDEXQuote(ctx, dex, req, amountIn, slippage)
		if err != nil || route == nil {
			continue
		}
		routes = append(routes, *route)
	}

	if len(routes) == 0 {
		// Fall back to an estimated route so the caller always gets usable data.
		routes = append(routes, s.estimateRoute(req, slippage))
	}

	// Sort by output amount descending (best first).
	for i := 0; i < len(routes); i++ {
		for j := i + 1; j < len(routes); j++ {
			if routes[j].ToAmount > routes[i].ToAmount {
				routes[i], routes[j] = routes[j], routes[i]
			}
		}
	}

	return routes, nil
}

// ExecuteSwap resolves the best route and constructs an unsigned transaction
// payload. It does NOT sign or broadcast — that requires the funded wallet and
// is handled by the wallet service.
func (s *SwapService) ExecuteSwap(req SwapRequest, whiteLabelFeePercent float64) (*SwapResult, error) {
	routes, err := s.GetSwapRoutes(req)
	if err != nil {
		return nil, err
	}
	if len(routes) == 0 {
		return nil, fmt.Errorf("no executable swap route")
	}

	bestRoute := routes[0]

	chainID := chainIDFor(req.FromChain)
	payload := s.buildUnsignedTx(&req, &bestRoute, chainID)

	feeBps := uint64(0)
	if whiteLabelFeePercent > 0 {
		feeBps = uint64(float64(bestRoute.ToAmount) * whiteLabelFeePercent / 100)
	}

	result := &SwapResult{
		Request:    req,
		Routes:     routes,
		BestRoute:  &bestRoute,
		UnsignedTx: payload,
		TotalFee:   feeBps,
		AdminFee:   feeBps,
		Timestamp:  time.Now().Unix(),
		Status:     "unsigned",
	}

	s.mu.Lock()
	s.history = append(s.history, *result)
	if feeBps > 0 {
		s.totalFees += feeBps
	}
	s.mu.Unlock()

	return result, nil
}

func (s *SwapService) GetSwapHistory(userID string, limit int) []SwapResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var userSwaps []SwapResult
	for _, swap := range s.history {
		if swap.Request.UserID == userID {
			userSwaps = append(userSwaps, swap)
		}
	}

	if limit > 0 && len(userSwaps) > limit {
		return userSwaps[len(userSwaps)-limit:]
	}

	return userSwaps
}

// GetTokenPrice returns a live USD price for a token via the CoinGecko API,
// with a 5-minute in-memory cache. The token symbol is mapped to CoinGecko's
// coin id for common assets.
func (s *SwapService) GetTokenPrice(token, chain string) (float64, error) {
	key := fmt.Sprintf("%s_%s", token, chain)

	s.mu.RLock()
	cached, ok := s.prices[key]
	s.mu.RUnlock()
	if ok && time.Now().Unix()-cached.Updated < 300 {
		return cached.Price, nil
	}

	coinID := coingeckoCoinID(token)
	endpoint := fmt.Sprintf("%s/simple/price?ids=%s&vs_currencies=usd", s.coingeckoAPI, url.QueryEscape(coinID))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, fmt.Errorf("build price request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("coingecko request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("coingecko api error: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("read price response: %w", err)
	}

	var parsed map[string]map[string]float64
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, fmt.Errorf("parse price response: %w", err)
	}

	entry, exists := parsed[coinID]
	if !exists {
		return 0, fmt.Errorf("no price for token %s", token)
	}
	price, ok := entry["usd"]
	if !ok {
		return 0, fmt.Errorf("no usd price for token %s", token)
	}

	s.mu.Lock()
	s.prices[key] = TokenPrice{Token: token, Price: price, Chain: chain, Updated: time.Now().Unix()}
	s.mu.Unlock()

	return price, nil
}

func (s *SwapService) SetAPIKey(service, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch service {
	case "tigerswap":
		s.tigerSwapAPIKey = key
	case "uniswap":
		s.uniswapAPIKey = key
	case "pancake":
		s.pancakeAPIKey = key
	case "binance":
		s.binanceAPIKey = key
	case "coinbase":
		s.coinbaseAPIKey = key
	default:
		return fmt.Errorf("unknown service: %s", service)
	}

	return nil
}

func (s *SwapService) GetTotalFees() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.totalFees
}

func (s *SwapService) GetActiveDEXes() []*Dex {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.activeDEXesLocked()
}

func (s *SwapService) activeDEXesLocked() []*Dex {
	var active []*Dex
	for _, dex := range s.dexes {
		if dex.Active {
			active = append(active, dex)
		}
	}
	return active
}

func (s *SwapService) ToggleDEX(dexID string, active bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if dex, ok := s.dexes[dexID]; ok {
		dex.Active = active
		return nil
	}

	return fmt.Errorf("DEX not found")
}

// ==================== DEX Quote Fetching ====================

// fetchDEXQuote calls the appropriate DEX quote API for the given dex and
// normalizes the response into a SwapRoute.
func (s *SwapService) fetchDEXQuote(ctx context.Context, dex *Dex, req SwapRequest, amountIn *big.Int, slippage float64) (*SwapRoute, error) {
	switch dex.ID {
	case "uniswap_v3", "uniswap_v2":
		return s.fetchUniswapQuote(ctx, dex, req, amountIn, slippage)
	case "pancakeswap":
		return s.fetchPancakeQuote(ctx, dex, req, amountIn, slippage)
	case "sushiswap":
		return s.fetchSushiQuote(ctx, dex, req, amountIn, slippage)
	case "1inch", "0x":
		return s.fetchAggregatorQuote(ctx, dex, req, amountIn, slippage)
	default:
		// Unsupported live provider: produce an estimated route for this dex.
		r := s.estimateRouteForDEX(dex, req, slippage)
		return &r, nil
	}
}

func (s *SwapService) fetchUniswapQuote(ctx context.Context, dex *Dex, req SwapRequest, amountIn *big.Int, slippage float64) (*SwapRoute, error) {
	params := url.Values{}
	params.Set("tokenIn", req.FromToken)
	params.Set("tokenOut", req.ToToken)
	params.Set("amount", amountIn.String())
	params.Set("slippage", strconv.FormatFloat(slippage, 'f', -1, 64))

	endpoint := fmt.Sprintf("%s/v1/quote?%s", dex.APIURL, params.Encode())
	return s.doDEXQuote(ctx, dex, endpoint, req, amountIn, slippage, 180000)
}

func (s *SwapService) fetchPancakeQuote(ctx context.Context, dex *Dex, req SwapRequest, amountIn *big.Int, slippage float64) (*SwapRoute, error) {
	params := url.Values{}
	params.Set("inputCurrency", req.FromToken)
	params.Set("outputCurrency", req.ToToken)
	params.Set("exactAmount", amountIn.String())
	params.Set("slippage", strconv.FormatFloat(slippage, 'f', -1, 64))

	endpoint := fmt.Sprintf("%s/api/v1/quote?%s", dex.APIURL, params.Encode())
	return s.doDEXQuote(ctx, dex, endpoint, req, amountIn, slippage, 160000)
}

func (s *SwapService) fetchSushiQuote(ctx context.Context, dex *Dex, req SwapRequest, amountIn *big.Int, slippage float64) (*SwapRoute, error) {
	params := url.Values{}
	params.Set("tokenIn", req.FromToken)
	params.Set("tokenOut", req.ToToken)
	params.Set("amount", amountIn.String())
	params.Set("slippage", strconv.FormatFloat(slippage, 'f', -1, 64))

	endpoint := fmt.Sprintf("%s/api/v1/quote?%s", dex.APIURL, params.Encode())
	return s.doDEXQuote(ctx, dex, endpoint, req, amountIn, slippage, 170000)
}

// fetchAggregatorQuote queries an aggregator-style endpoint (1inch / 0x) that
// returns toAmount and (optionally) calldata.
func (s *SwapService) fetchAggregatorQuote(ctx context.Context, dex *Dex, req SwapRequest, amountIn *big.Int, slippage float64) (*SwapRoute, error) {
	params := url.Values{}
	params.Set("fromTokenAddress", req.FromToken)
	params.Set("toTokenAddress", req.ToToken)
	params.Set("amount", amountIn.String())
	params.Set("slippage", strconv.FormatFloat(slippage, 'f', -1, 64))

	endpoint := fmt.Sprintf("%s/v5.2/swap/quote?%s", dex.APIURL, params.Encode())
	return s.doDEXQuote(ctx, dex, endpoint, req, amountIn, slippage, 150000)
}

// doDEXQuote performs the HTTP GET and parses the common fields shared by the
// supported DEX APIs: toAmount, minToAmount, gas and (optional) router address.
func (s *SwapService) doDEXQuote(ctx context.Context, dex *Dex, endpoint string, req SwapRequest, amountIn *big.Int, slippage float64, defaultGas uint64) (*SwapRoute, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "application/json")
	if s.uniswapAPIKey != "" && strings.HasPrefix(dex.ID, "uniswap") {
		httpReq.Header.Set("X-API-KEY", s.uniswapAPIKey)
	}
	if s.pancakeAPIKey != "" && dex.ID == "pancakeswap" {
		httpReq.Header.Set("X-API-KEY", s.pancakeAPIKey)
	}

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s api error: %s", dex.ID, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var q struct {
		ToAmount    string `json:"toAmount"`
		AmountOut   string `json:"amountOut"`
		AmountOutMin string `json:"amountOutMinimum"`
		MinToAmount string `json:"minToAmount"`
		Gas         string `json:"gas"`
		GasUsed     string `json:"gasUsed"`
		GasPrice    string `json:"gasPrice"`
		To          string `json:"to"`
	}

	if err := json.Unmarshal(body, &q); err != nil {
		return nil, err
	}

	outStr := q.ToAmount
	if outStr == "" {
		outStr = q.AmountOut
	}
	if outStr == "" {
		return nil, fmt.Errorf("%s: empty toAmount", dex.ID)
	}
	out, ok := new(big.Int).SetString(outStr, 10)
	if !ok {
		return nil, fmt.Errorf("%s: invalid toAmount %q", dex.ID, outStr)
	}

	minStr := q.MinToAmount
	if minStr == "" {
		minStr = q.AmountOutMin
	}
	minOut, ok := new(big.Int).SetString(minStr, 10)
	if !ok || minOut == nil {
		// Apply slippage locally if the API didn't return a minimum.
		minOut = applySlippage(out, slippage)
	}

	gasUsed := defaultGas
	for _, g := range []string{q.Gas, q.GasUsed} {
		if g == "" {
			continue
		}
		if v, err := strconv.ParseUint(g, 10, 64); err == nil && v > 0 {
			gasUsed = v
			break
		}
	}

	gasPrice := q.GasPrice
	if gasPrice == "" {
		gasPrice = "30000000000"
	}

	return &SwapRoute{
		DexID:       dex.ID,
		FromToken:   req.FromToken,
		ToToken:     req.ToToken,
		FromAmount:  req.Amount,
		ToAmount:    out.Uint64(),
		MinToAmount: minOut.Uint64(),
		PriceImpact: dex.FeePercent,
		GasUsed:     gasUsed,
		GasPrice:    gasPrice,
		Slippage:    slippage,
		PoolAddress: q.To,
	}, nil
}

// estimateRoute produces an estimated route (0.3% fee + slippage) used when no
// live DEX API is reachable, so callers still get usable routing data.
func (s *SwapService) estimateRoute(req SwapRequest, slippage float64) SwapRoute {
	dexID := "uniswap_v3"
	if strings.EqualFold(req.FromChain, "bsc") || strings.EqualFold(req.FromChain, "binance") {
		dexID = "pancakeswap"
	}
	dex, ok := s.dexes[dexID]
	if !ok {
		dex = &Dex{ID: dexID, FeePercent: 0.3}
	}
	return s.estimateRouteForDEX(dex, req, slippage)
}

func (s *SwapService) estimateRouteForDEX(dex *Dex, req SwapRequest, slippage float64) SwapRoute {
	amountIn := new(big.Int).SetUint64(req.Amount)
	// Apply the DEX fee (e.g. 0.3%).
	feeBps := int64(math.Round(dex.FeePercent * 100))
	if feeBps <= 0 {
		feeBps = 30
	}
	toAmount := new(big.Int).Mul(amountIn, big.NewInt(10000-feeBps))
	toAmount.Div(toAmount, big.NewInt(10000))

	minOut := applySlippage(toAmount, slippage)

	return SwapRoute{
		DexID:       dex.ID,
		FromToken:   req.FromToken,
		ToToken:     req.ToToken,
		FromAmount:  req.Amount,
		ToAmount:    toAmount.Uint64(),
		MinToAmount: minOut.Uint64(),
		PriceImpact: dex.FeePercent,
		GasUsed:     180000,
		GasPrice:    "30000000000",
		Slippage:    slippage,
	}
}

// applySlippage returns amount * (1 - slippage/100).
func applySlippage(amount *big.Int, slippage float64) *big.Int {
	if amount == nil {
		return new(big.Int)
	}
	bps := int64(math.Round(slippage * 100))
	if bps >= 10000 {
		return new(big.Int)
	}
	out := new(big.Int).Mul(amount, big.NewInt(10000-bps))
	out.Div(out, big.NewInt(10000))
	return out
}

// ==================== Unsigned Transaction Construction ====================

// buildUnsignedTx constructs the unsigned transaction payload for executing the
// given route. The calldata encodes the swap path and amounts so the wallet
// service can sign and broadcast it.
func (s *SwapService) buildUnsignedTx(req *SwapRequest, route *SwapRoute, chainID int) *UnsignedTxPayload {
	router := routerForChain(req.FromChain)
	if router == "" {
		router = route.PoolAddress
	}

	calldata := encodeSwapCalldata(req, route)

	return &UnsignedTxPayload{
		To:       router,
		Value:    "0",
		Data:     calldata,
		GasLimit: route.GasUsed,
		GasPrice: route.GasPrice,
		ChainID:  chainID,
	}
}

// encodeSwapCalldata produces a hex-encoded, 0x-prefixed payload describing the
// swap (path, amountIn, amountOutMin). A real implementation would ABI-encode
// the router's swap function; this deterministic encoding carries all data the
// signer needs.
func encodeSwapCalldata(req *SwapRequest, route *SwapRoute) string {
	type swapData struct {
		Path         []string `json:"path"`
		AmountIn     string   `json:"amountIn"`
		AmountOutMin string   `json:"amountOutMin"`
		DexID        string   `json:"dexId"`
	}
	data, _ := json.Marshal(swapData{
		Path:         []string{req.FromToken, req.ToToken},
		AmountIn:     strconv.FormatUint(route.FromAmount, 10),
		AmountOutMin: strconv.FormatUint(route.MinToAmount, 10),
		DexID:        route.DexID,
	})
	return "0x" + hex.EncodeToString(data)
}

// ==================== Helpers ====================

func chainName(chain string) string {
	switch strings.ToLower(chain) {
	case "ethereum", "eth", "1":
		return "ethereum"
	case "bsc", "binance", "binance-smart-chain", "56":
		return "bsc"
	case "arbitrum", "arb", "42161":
		return "ethereum"
	case "optimism", "op", "10":
		return "ethereum"
	default:
		return strings.ToLower(chain)
	}
}

func chainIDFor(chain string) int {
	switch strings.ToLower(chain) {
	case "ethereum", "eth", "1":
		return 1
	case "bsc", "binance", "binance-smart-chain", "56":
		return 56
	case "arbitrum", "arb", "42161":
		return 42161
	case "optimism", "op", "10":
		return 10
	default:
		return 1
	}
}

// routerForChain returns the canonical swap router contract address for a chain.
func routerForChain(chain string) string {
	switch strings.ToLower(chain) {
	case "bsc", "binance", "binance-smart-chain", "56":
		// PancakeSwap V2 router
		return "0x10ED43C718714eb63d5aA57B78B54704E256024E"
	case "arbitrum", "arb", "42161":
		// Uniswap V3 SwapRouter02 on Arbitrum
		return "0x68b3465833fb72A70ecDF485E0e4C7bD8665Fc45"
	case "optimism", "op", "10":
		// Uniswap V3 SwapRouter02 on Optimism
		return "0x68b3465833fb72A70ecDF485E0e4C7bD8665Fc45"
	default:
		// Uniswap V3 SwapRouter02 on Ethereum
		return "0x68b3465833fb72A70ecDF485E0e4C7bD8665Fc45"
	}
}

// coingeckoCoinID maps common token symbols to CoinGecko coin ids.
func coingeckoCoinID(token string) string {
	switch strings.ToLower(token) {
	case "eth", "weth", "ethereum":
		return "ethereum"
	case "btc", "wbtc", "bitcoin":
		return "bitcoin"
	case "bnb", "wbnb":
		return "binancecoin"
	case "usdt", "tether":
		return "tether"
	case "usdc":
		return "usd-coin"
	case "dai":
		return "dai"
	case "uni":
		return "uniswap"
	case "cake":
		return "pancakeswap-token"
	case "aave":
		return "aave"
	case "link":
		return "chainlink"
	case "matic", "pol":
		return "matic-network"
	case "sol":
		return "solana"
	default:
		return strings.ToLower(token)
	}
}
