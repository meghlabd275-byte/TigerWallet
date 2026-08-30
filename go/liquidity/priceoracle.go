package main

// priceoracle.go - real CoinGecko price oracle. Fail-closed: returns only
// live prices; no fabricated fallback.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

var cgIDBySymbol = map[string]string{
	"BTC": "bitcoin", "ETH": "ethereum", "BNB": "binancecoin", "SOL": "solana",
	"XRP": "ripple", "DOGE": "dogecoin", "ADA": "cardano", "AVAX": "avalanche-2",
	"DOT": "polkadot", "LINK": "chainlink", "MATIC": "matic-network", "LTC": "litecoin",
	"UNI": "uniswap", "ATOM": "cosmos", "XLM": "stellar", "NEAR": "near",
	"APT": "aptos", "ARB": "arbitrum", "OP": "optimism", "INJ": "injective-protocol",
	"USDT": "tether", "USDC": "usd-coin", "TRX": "tron", "TON": "the-open-network",
}

type priceCache struct {
	mu    sync.RWMutex
	items map[string]cachedPrice
}
type cachedPrice struct {
	price float64
	at    time.Time
}

var cgCache = &priceCache{items: map[string]cachedPrice{}}

func cgHTTPGet(url string, out interface{}) error {
	client := &http.Client{Timeout: 12 * time.Second}
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("coingecko HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// fetchLivePricesUSD returns real USD prices keyed by UPPER symbol.
// Unknown symbols are omitted; on total failure returns an error.
func fetchLivePricesUSD(symbols []string) (map[string]float64, error) {
	ids := []string{}
	symByID := map[string]string{}
	for _, s := range symbols {
		up := strings.ToUpper(s)
		id, ok := cgIDBySymbol[up]
		if !ok {
			continue
		}
		ids = append(ids, id)
		symByID[id] = up
	}
	if len(ids) == 0 {
		return map[string]float64{}, nil
	}
	url := "https://api.coingecko.com/api/v3/simple/price?ids=" + strings.Join(ids, ",") + "&vs_currencies=usd&include_24hr_change=true"
	var raw map[string]map[string]float64
	if err := cgHTTPGet(url, &raw); err != nil {
		return nil, err
	}
	out := map[string]float64{}
	for id, fields := range raw {
		if p, ok := fields["usd"]; ok && p > 0 {
			out[symByID[id]] = p
			cgCache.mu.Lock()
			cgCache.items[symByID[id]] = cachedPrice{price: p, at: time.Now()}
			cgCache.mu.Unlock()
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no prices returned")
	}
	return out, nil
}

// livePriceUSD returns a single real price, preferring a fresh cache entry.
func livePriceUSD(symbol string) (float64, error) {
	up := strings.ToUpper(symbol)
	cgCache.mu.RLock()
	if cp, ok := cgCache.items[up]; ok && time.Since(cp.at) < 60*time.Second {
		cgCache.mu.RUnlock()
		return cp.price, nil
	}
	cgCache.mu.RUnlock()
	m, err := fetchLivePricesUSD([]string{up})
	if err != nil {
		return 0, err
	}
	p, ok := m[up]
	if !ok {
		return 0, fmt.Errorf("price unavailable for %s", up)
	}
	return p, nil
}
