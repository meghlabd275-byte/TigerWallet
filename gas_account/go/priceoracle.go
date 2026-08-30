package main

// priceoracle.go — real CoinGecko USD price oracle with a 60s in-memory TTL
// cache. Fail-closed: a fetch/parse failure returns an error (never a
// fabricated price). Optional COINGECKO_API_KEY enables the pro endpoint.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// coinGeckoIDs maps ticker symbols to CoinGecko API ids.
var coinGeckoIDs = map[string]string{
	"BTC": "bitcoin", "ETH": "ethereum", "SOL": "solana", "BNB": "binancecoin",
	"MATIC": "matic-network", "POL": "matic-network", "AVAX": "avalanche-2",
	"FTM": "fantom", "ONE": "harmony", "MOVR": "moonriver", "XRP": "ripple",
	"DOGE": "dogecoin", "ADA": "cardano", "DOT": "polkadot", "LINK": "chainlink",
	"LTC": "litecoin", "UNI": "uniswap", "ATOM": "cosmos", "XLM": "stellar",
	"NEAR": "near", "APT": "aptos", "ARB": "arbitrum", "OP": "optimism",
	"INJ": "injective-protocol", "WBTC": "wrapped-bitcoin", "TRX": "tron",
	"USDT": "tether", "USDC": "usd-coin", "DAI": "dai",
}

// oracleStablecoins are pinned to exactly 1.0 USD for exact funding math.
var oracleStablecoins = map[string]bool{"USDT": true, "USDC": true, "DAI": true}

var oracleCache = struct {
	sync.RWMutex
	prices    map[string]float64
	fetchedAt time.Time
}{prices: map[string]float64{}}

// fetchLivePricesUSD pulls live USD prices for the given symbols from
// CoinGecko. Returns only successfully-priced symbols; error if none.
func fetchLivePricesUSD(symbols []string) (map[string]float64, error) {
	ids := map[string]bool{}
	idToSymbol := map[string]string{}
	for _, s := range symbols {
		s = strings.ToUpper(strings.TrimSpace(s))
		id, ok := coinGeckoIDs[s]
		if !ok {
			continue
		}
		if oracleStablecoins[s] {
			continue // pinned below
		}
		ids[id] = true
		idToSymbol[id] = s
	}
	out := map[string]float64{}
	for s := range oracleStablecoins {
		for _, want := range symbols {
			if strings.EqualFold(want, s) {
				out[strings.ToUpper(want)] = 1.0
			}
		}
	}
	if len(ids) > 0 {
		idList := make([]string, 0, len(ids))
		for id := range ids {
			idList = append(idList, id)
		}
		base := "https://api.coingecko.com/api/v3/simple/price"
		key := os.Getenv("COINGECKO_API_KEY")
		if key != "" {
			base = "https://pro-api.coingecko.com/api/v3/simple/price"
		}
		url := base + "?ids=" + strings.Join(idList, ",") + "&vs_currencies=usd&include_24hr_change=true"
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		if key != "" {
			req.Header.Set("x-cg-pro-api-key", key)
		}
		client := &http.Client{Timeout: 8 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("coingecko HTTP %d", resp.StatusCode)
		}
		var parsed map[string]map[string]float64
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			return nil, err
		}
		for id, sym := range idToSymbol {
			if usd, ok := parsed[id]["usd"]; ok && usd > 0 {
				out[sym] = usd
			}
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no live prices available")
	}
	oracleCache.Lock()
	for k, v := range out {
		oracleCache.prices[k] = v
	}
	oracleCache.fetchedAt = time.Now()
	oracleCache.Unlock()
	return out, nil
}

// livePriceUSD returns the live USD price for a symbol, using the 60s shared
// cache when fresh. Fail-closed: error when the price cannot be determined.
func livePriceUSD(symbol string) (float64, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if oracleStablecoins[symbol] {
		return 1.0, nil
	}
	oracleCache.RLock()
	price, ok := oracleCache.prices[symbol]
	fresh := time.Since(oracleCache.fetchedAt) < 60*time.Second
	oracleCache.RUnlock()
	if ok && fresh {
		return price, nil
	}
	fetched, err := fetchLivePricesUSD([]string{symbol})
	if err != nil {
		// Serve the stale cached price if we have one; otherwise fail closed.
		if ok {
			return price, nil
		}
		return 0, fmt.Errorf("price unavailable for %s: %w", symbol, err)
	}
	price, ok = fetched[symbol]
	if !ok {
		return 0, fmt.Errorf("price unavailable for %s", symbol)
	}
	return price, nil
}
