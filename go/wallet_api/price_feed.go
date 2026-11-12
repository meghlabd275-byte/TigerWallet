package main

// price_feed.go — the canonical USD price feed used by every wallet surface.
//
// Design rules, in order of importance:
//
//  1. A price is either fetched from a public exchange API or derived from live
//     on-chain pool reserves. It is never a constant, never a random number and
//     never a "reasonable-looking" default.
//  2. Several independent sources are queried and the median is returned. One
//     exchange having an outage or returning a bad tick cannot move the price a
//     user sees, and no single provider is a hard dependency.
//  3. When no source can answer, the price is reported as unavailable (0 with
//     source "unavailable") and the caller must omit the fiat value rather than
//     show a wrong one.
//
// The stablecoin peg is the one declared value in this file, and it is declared
// because it is a contract fact rather than a market opinion: RUSD is minted
// against a USD reserve held by the treasury and redeemed 1:1. It is reported
// with source "declared_peg" so a caller can always tell a peg from a market
// price. Every other stablecoin still goes to the exchanges first, and only
// falls back to the peg if all of them are unreachable.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// PriceQuote is the result of a price lookup.
type PriceQuote struct {
	// Symbol is the upper-case asset symbol the quote is for.
	Symbol string `json:"symbol"`

	// USD is the price in US dollars. Zero when no source answered.
	USD float64 `json:"usd"`

	// Change24h is the 24 hour change as a fraction (0.0125 == +1.25%).
	Change24h float64 `json:"change_24h"`

	// Source names where the price came from: "binance", "okx", "coinbase",
	// "coingecko", "onchain_pool" or "declared_peg". A caller that must not
	// show an estimate can check this field.
	Source string `json:"source"`

	// Sources lists every provider that answered, before the median was taken.
	Sources []string `json:"sources,omitempty"`

	// UpdatedAt is when the quote was produced.
	UpdatedAt time.Time `json:"updated_at"`
}

// Available reports whether a real price was obtained.
func (q PriceQuote) Available() bool { return q.USD > 0 && q.Source != "unavailable" }

// declaredPegs are assets whose issuer redeems them 1:1 against the US dollar.
// Only stablecoins the platform itself issues are listed here. Market
// stablecoins such as USDT and USDC are NOT listed: their price is read from
// the exchanges like any other asset.
var declaredPegs = map[string]float64{
	"RUSD":  1.0, // Royal Tiger USD - genesis predeploy on TigerSmartChain
	"RYUSD": 1.0, // Royal Tiger yield USD
}

// symbolToCoinGeckoPriceFeed maps a symbol to the CoinGecko id used by the
// aggregator. The terminal uses its own table; this one is the price feed's.
var symbolToCoinGeckoPriceFeed = map[string]string{
	"BTC":   "bitcoin",
	"ETH":   "ethereum",
	"BNB":   "binancecoin",
	"SOL":   "solana",
	"XRP":   "ripple",
	"ADA":   "cardano",
	"DOT":   "polkadot",
	"MATIC": "matic-network",
	"POL":   "matic-network",
	"AVAX":  "avalanche-2",
	"LINK":  "chainlink",
	"UNI":   "uniswap",
	"ATOM":  "cosmos",
	"DOGE":  "dogecoin",
	"LTC":   "litecoin",
	"TRX":   "tron",
	"ARB":   "arbitrum",
	"OP":    "optimism",
	"USDT":  "tether",
	"USDC":  "usd-coin",
	"DAI":   "dai",
	"BUSD":  "binance-usd",
	"TUSD":  "true-usd",
	"WTGR":  "tiger-smart-chain",
}

// priceSourceTimeout bounds a single provider call. The aggregator queries the
// providers concurrently, so the total latency is one timeout, not four.
const priceSourceTimeout = 6 * time.Second

// priceHTTPClient is shared so connections are reused across lookups.
var priceHTTPClient = &http.Client{Timeout: priceSourceTimeout}

// FetchAggregatedPrice returns the median USD price for a symbol across every
// reachable source. Providers that fail are skipped; the quote is only marked
// unavailable when none succeed.
func FetchAggregatedPrice(ctx context.Context, symbol string) PriceQuote {
	sym := strings.ToUpper(strings.TrimSpace(symbol))
	if sym == "" {
		return PriceQuote{Symbol: sym, Source: "unavailable", UpdatedAt: time.Now().UTC()}
	}

	type result struct {
		name   string
		usd    float64
		change float64
		ok     bool
	}

	sources := []func(context.Context, string) (float64, float64, bool){
		fetchBinancePrice,
		fetchOKXPrice,
		fetchCoinbasePrice,
		func(c context.Context, s string) (float64, float64, bool) {
			return fetchCoinGeckoAggregated(c, s)
		},
	}

	results := make([]result, len(sources))
	var wg sync.WaitGroup
	for i, fn := range sources {
		wg.Add(1)
		go func(i int, fn func(context.Context, string) (float64, float64, bool)) {
			defer wg.Done()
			usd, change, ok := fn(ctx, sym)
			if ok && usd > 0 {
				results[i] = result{name: sourceName(i), usd: usd, change: change, ok: true}
			}
		}(i, fn)
	}
	wg.Wait()

	prices := make([]float64, 0, len(results))
	names := make([]string, 0, len(results))
	changeSum, changeN := 0.0, 0
	for _, r := range results {
		if !r.ok {
			continue
		}
		prices = append(prices, r.usd)
		names = append(names, r.name)
		if r.change != 0 {
			changeSum += r.change
			changeN++
		}
	}

	quote := PriceQuote{Symbol: sym, Sources: names, UpdatedAt: time.Now().UTC()}
	if len(prices) > 0 {
		sort.Float64s(prices)
		quote.USD = median(prices)
		quote.Source = "median:" + strings.Join(names, "+")
		if changeN > 0 {
			quote.Change24h = changeSum / float64(changeN)
		}
		return quote
	}

	// No exchange answered. A platform-issued stablecoin still has a price: the
	// issuer redeems it 1:1. That is a declared contract fact, reported as such.
	if peg, ok := declaredPegs[sym]; ok {
		quote.USD = peg
		quote.Source = "declared_peg"
		return quote
	}

	// Last resort for assets that only exist on-chain: read the AMM pool.
	if usd, ok := fetchOnChainPoolPrice(ctx, sym); ok {
		quote.USD = usd
		quote.Source = "onchain_pool"
		return quote
	}

	quote.Source = "unavailable"
	return quote
}

// median returns the middle value of a sorted slice (mean of the two middle
// values for an even count).
func median(sorted []float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

func sourceName(i int) string {
	switch i {
	case 0:
		return "binance"
	case 1:
		return "okx"
	case 2:
		return "coinbase"
	default:
		return "coingecko"
	}
}

// ---- Providers -----------------------------------------------------------
//
// Each provider speaks a public, unauthenticated market-data API. None of them
// requires an API key, so the wallet works out of the box; when a key is
// configured for CoinGecko it is used to raise the rate limit.

func fetchBinancePrice(ctx context.Context, symbol string) (float64, float64, bool) {
	// Binance quotes against USDT. Tether is itself a dollar instrument, so the
	// USDT price is used as the USD price; that is the market convention and it
	// is stated here rather than hidden.
	pair := symbol + "USDT"
	url := "https://api.binance.com/api/v3/ticker/24hr?symbol=" + pair
	var out struct {
		LastPrice     string `json:"lastPrice"`
		PriceChangePc string `json:"priceChangePercent"`
	}
	if !getJSON(ctx, url, &out) {
		return 0, 0, false
	}
	usd, err := parseFloat(out.LastPrice)
	if err != nil || usd <= 0 {
		return 0, 0, false
	}
	change, _ := parseFloat(out.PriceChangePc)
	return usd, change / 100, true
}

func fetchOKXPrice(ctx context.Context, symbol string) (float64, float64, bool) {
	url := "https://www.okx.com/api/v5/market/ticker?instId=" + symbol + "-USDT"
	var out struct {
		Data []struct {
			Last   string `json:"last"`
			Open24 string `json:"open24h"`
		} `json:"data"`
	}
	if !getJSON(ctx, url, &out) || len(out.Data) == 0 {
		return 0, 0, false
	}
	last, err := parseFloat(out.Data[0].Last)
	if err != nil || last <= 0 {
		return 0, 0, false
	}
	var change float64
	if open, err := parseFloat(out.Data[0].Open24); err == nil && open > 0 {
		change = (last - open) / open
	}
	return last, change, true
}

func fetchCoinbasePrice(ctx context.Context, symbol string) (float64, float64, bool) {
	url := "https://api.exchange.coinbase.com/products/" + symbol + "-USD/ticker"
	var out struct {
		Price  string `json:"price"`
		Open   string `json:"open"`
		Volume string `json:"volume"`
	}
	if !getJSON(ctx, url, &out) {
		return 0, 0, false
	}
	last, err := parseFloat(out.Price)
	if err != nil || last <= 0 {
		return 0, 0, false
	}
	var change float64
	if open, err := parseFloat(out.Open); err == nil && open > 0 {
		change = (last - open) / open
	}
	return last, change, true
}

func fetchCoinGeckoAggregated(ctx context.Context, symbol string) (float64, float64, bool) {
	id, ok := symbolToCoinGeckoPriceFeed[symbol]
	if !ok {
		id = strings.ToLower(symbol)
	}
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=usd&include_24hr_change=true", id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, 0, false
	}
	if appConfig.CoinGeckoAPIKey != "" {
		req.Header.Set("x-cg-pro-api-key", appConfig.CoinGeckoAPIKey)
	}
	resp, err := priceHTTPClient.Do(req)
	if err != nil {
		return 0, 0, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, 0, false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return 0, 0, false
	}
	var raw map[string]struct {
		USD    float64 `json:"usd"`
		Change float64 `json:"usd_24h_change"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return 0, 0, false
	}
	entry, ok := raw[id]
	if !ok || entry.USD <= 0 {
		return 0, 0, false
	}
	return entry.USD, entry.Change / 100, true
}

// getJSON performs a GET and decodes a JSON body, returning false on any
// failure. It never returns partial data.
func getJSON(ctx context.Context, url string, out any) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	// Public market-data APIs reject requests without a User-Agent.
	req.Header.Set("User-Agent", "TigerWallet/1.0")
	resp, err := priceHTTPClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return false
	}
	return json.Unmarshal(body, out) == nil
}

func parseFloat(s string) (float64, error) {
	var f float64
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	if _, err := fmt.Sscanf(strings.TrimSpace(s), "%g", &f); err != nil {
		return 0, err
	}
	return f, nil
}

// ---- On-chain pool price -------------------------------------------------

// fetchOnChainPoolPrice reads the price of an asset straight out of a
// Uniswap-V2 style pool on TigerSmartChain.
//
// This is the honest answer for TGR and WTGR: until a centralised exchange
// lists them there is no order book to quote from, but there is a real AMM
// pool whose reserves are public state. The price is derived from those
// reserves with the constant-product rule, so it is a real market price that a
// user can act on, not an estimate.
//
// Configuration (both must be set, otherwise no on-chain price is produced):
//
//	TSC_RPC_URL          TigerSmartChain JSON-RPC endpoint
//	TSC_TGR_RUSD_PAIR    address of the TGR/RUSD (or WTGR/RUSD) pair
func fetchOnChainPoolPrice(ctx context.Context, symbol string) (float64, bool) {
	switch symbol {
	case "TGR", "WTGR":
	default:
		return 0, false
	}

	rpcURL := strings.TrimSpace(envOr("TSC_RPC_URL", ""))
	pairAddr := strings.TrimSpace(envOr("TSC_TGR_RUSD_PAIR", ""))
	if rpcURL == "" || !common.IsHexAddress(pairAddr) {
		return 0, false
	}

	// getReserves() selector 0x0902f1ac -> (uint112 reserve0, uint112 reserve1, uint32 blockTimestampLast)
	raw, err := rpcEthCall(ctx, rpcURL, pairAddr, "0x0902f1ac")
	if err != nil || len(raw) < 2+3*64 {
		return 0, false
	}
	reserve0, ok0 := new(big.Int).SetString(strings.TrimPrefix(raw[2:66], "0x"), 16)
	reserve1, ok1 := new(big.Int).SetString(strings.TrimPrefix(raw[66:130], "0x"), 16)
	if !ok0 || !ok1 || reserve0.Sign() <= 0 || reserve1.Sign() <= 0 {
		return 0, false
	}

	// token0/token1 ordering decides which reserve is RUSD. Read it from the
	// pool instead of assuming: token0() selector 0x0dfe1681, token1() 0x4f1a4a83.
	rusd := strings.ToLower(common.HexToAddress(rusdPredeployAddress).Hex())
	token0Raw, err0 := rpcEthCall(ctx, rpcURL, pairAddr, "0x0dfe1681")
	token1Raw, err1 := rpcEthCall(ctx, rpcURL, pairAddr, "0x4f1a4a83")
	if err0 != nil || err1 != nil || len(token0Raw) < 66 || len(token1Raw) < 66 {
		return 0, false
	}
	token0 := strings.ToLower(common.HexToAddress("0x" + token0Raw[len(token0Raw)-40:]).Hex())
	token1 := strings.ToLower(common.HexToAddress("0x" + token1Raw[len(token1Raw)-40:]).Hex())

	// price(asset in USD) = reserveRUSD / reserveAsset, with both sides scaled
	// to 18 decimals (TGR, WTGR and RUSD are all 18-decimal assets).
	//
	// Exactly one leg must be RUSD. A pool holding RUSD on both sides has no
	// asset leg, and its ratio would always read 1.0; that is a malformed pool,
	// not a one-dollar price, so it is refused.
	var assetReserve, rusdReserve *big.Int
	switch {
	case token0 == rusd && token1 != rusd:
		rusdReserve, assetReserve = reserve0, reserve1
	case token1 == rusd && token0 != rusd:
		rusdReserve, assetReserve = reserve1, reserve0
	default:
		return 0, false
	}
	if assetReserve.Sign() <= 0 {
		return 0, false
	}

	ratio := new(big.Float).Quo(new(big.Float).SetInt(rusdReserve), new(big.Float).SetInt(assetReserve))
	usd, _ := ratio.Float64()
	if usd <= 0 {
		return 0, false
	}
	return usd, true
}

// rpcEthCall performs eth_call against a JSON-RPC endpoint and returns the
// 0x-prefixed hex result.
func rpcEthCall(ctx context.Context, endpoint, to, data string) (string, error) {
	callCtx, cancel := context.WithTimeout(ctx, priceSourceTimeout)
	defer cancel()

	payload := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"eth_call","params":[{"to":%q,"data":%q},"latest"]}`, to, data)
	req, err := http.NewRequestWithContext(callCtx, http.MethodPost, endpoint, strings.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "TigerWallet/1.0")

	resp, err := priceHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	var out struct {
		Result string `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", err
	}
	if out.Error != nil {
		return "", fmt.Errorf("eth_call: %s", out.Error.Message)
	}
	if out.Result == "" {
		return "", fmt.Errorf("eth_call: empty result")
	}
	return out.Result, nil
}
