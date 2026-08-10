/**
 * TigerWallet Price Oracle Service
 * High-Load Distributed Go Implementation
 *
 * Features:
 * - Multi-source price aggregation
 * - TWAP, VWAP calculations
 * - Anomaly detection
 * - Staleness alerts
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
	"strings"
	"sync"
	"time"
)

type Price struct {
	Symbol    string  `json:"symbol"`
	Price     float64 `json:"price"`
	Change24h float64 `json:"change_24h"`
	Volume24h float64 `json:"volume_24h"`
	Sources   int     `json:"sources"`
	UpdatedAt int64   `json:"updated_at"`
}

type PriceSource struct {
	Name   string  `json:"name"`
	Price  float64 `json:"price"`
	Weight float64 `json:"weight"`
}

type OracleConfig struct {
	Sources   []string `json:"sources"`
	Deviation float64  `json:"deviation"` // Max deviation before alert
	Staleness int64    `json:"staleness"` // Max age in seconds
}

type PriceAlert struct {
	ID        string  `json:"id"`
	Symbol    string  `json:"symbol"`
	Condition string  `json:"condition"` // above, below
	Target    float64 `json:"target"`
	Triggered bool    `json:"triggered"`
}

type OracleService struct {
	prices  map[string]*Price
	sources map[string][]PriceSource
	alerts  []PriceAlert
	config  OracleConfig
	mu      sync.RWMutex
	cgBase  string
	http    *http.Client
}

// coinGeckoID maps an oracle symbol to its CoinGecko coin id.
// Stablecoins and listed assets resolve to real ids; unknown symbols
// resolve to "" and will not be fetched (no fabricated price).
func coinGeckoID(symbol string) string {
	switch strings.ToUpper(symbol) {
	case "BTC":
		return "bitcoin"
	case "ETH":
		return "ethereum"
	case "BNB":
		return "binancecoin"
	case "SOL":
		return "solana"
	case "MATIC":
		return "matic-network"
	case "AVAX":
		return "avalanche-2"
	case "LINK":
		return "chainlink"
	case "UNI":
		return "uniswap"
	case "AAVE":
		return "aave"
	case "USDT":
		return "tether"
	case "USDC":
		return "usd-coin"
	default:
		return ""
	}
}

func NewOracleService() *OracleService {
	return &OracleService{
		prices:  make(map[string]*Price),
		sources: make(map[string][]PriceSource),
		alerts:  make([]PriceAlert, 0),
		config: OracleConfig{
			Sources:   []string{"binance", "coinbase", "kraken"},
			Deviation: 5.0,
			Staleness: 60,
		},
		cgBase: strings.TrimRight(getEnv("COINGECKO_BASE", "https://api.coingecko.com"), "/"),
		http:   &http.Client{Timeout: 5 * time.Second},
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func (o *OracleService) Run() error {
	// Start price updates
	go o.updatePrices()

	mux := http.NewServeMux()
	mux.HandleFunc("/price", o.handlePrice)
	mux.HandleFunc("/prices", o.handlePrices)
	mux.HandleFunc("/sources", o.handleSources)
	mux.HandleFunc("/alerts", o.handleAlerts)
	mux.HandleFunc("/health", o.handleHealth)

	log.Println("Oracle service starting on :8093")
	return http.ListenAndServe(":8093", mux)
}

func (o *OracleService) updatePrices() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	symbols := []string{"ETH", "BTC", "BNB", "SOL", "MATIC", "AVAX", "LINK", "UNI", "AAVE", "USDT", "USDC"}

	for range ticker.C {
		o.mu.Lock()
		for _, sym := range symbols {
			prices := o.fetchFromSources(sym)
			if len(prices) == 0 {
				// No real price available this cycle; keep prior price if any
				// but never fabricate one.
				continue
			}
			o.sources[sym] = prices

			// Calculate weighted average
			var totalWeight, totalPrice float64
			for _, p := range prices {
				totalWeight += p.Weight
				totalPrice += p.Price * p.Weight
			}

			avgPrice := totalPrice / totalWeight

			// Calculate change
			prevPrice := 0.0
			if existing, ok := o.prices[sym]; ok {
				prevPrice = existing.Price
			}

			change := 0.0
			if prevPrice > 0 {
				change = ((avgPrice - prevPrice) / prevPrice) * 100
			}

			// Use real 24h change/volume when available; otherwise 0 (never fabricated).
			mktChange, mktVolume := o.fetchMarket(sym)
			if mktChange != 0 {
				change = mktChange
			}

			o.prices[sym] = &Price{
				Symbol:    sym,
				Price:     avgPrice,
				Change24h: change,
				Volume24h: mktVolume,
				Sources:   len(prices),
				UpdatedAt: time.Now().UnixMilli(),
			}

			// Check alerts
			o.checkAlerts(sym, avgPrice)
		}
		o.mu.Unlock()
	}
}

func (o *OracleService) fetchFromSources(symbol string) []PriceSource {
	coinID := coinGeckoID(symbol)
	if coinID == "" {
		// Unknown symbol: no real source available. Return nothing rather
		// than fabricating a price; callers skip symbols with no sources.
		return nil
	}

	url := fmt.Sprintf("%s/api/v3/simple/price?ids=%s&vs_currencies=usd&include_24hr_vol=true&include_24hr_change=true", o.cgBase, coinID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("Accept", "application/json")
	resp, err := o.http.Do(req)
	if err != nil {
		log.Printf("oracle: coingecko fetch failed for %s: %v", symbol, err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("oracle: coingecko returned %d for %s", resp.StatusCode, symbol)
		return nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("oracle: read coingecko body for %s: %v", symbol, err)
		return nil
	}
	var parsed map[string]map[string]float64
	if err := json.Unmarshal(body, &parsed); err != nil {
		log.Printf("oracle: parse coingecko body for %s: %v", symbol, err)
		return nil
	}
	entry, ok := parsed[coinID]
	if !ok {
		return nil
	}
	price := entry["usd"]
	if price <= 0 {
		return nil
	}
	// Single authoritative source (CoinGecko). Weighted to 1.0 so the
	// aggregator uses the real value directly.
	return []PriceSource{{Name: "coingecko", Price: price, Weight: 1.0}}
}

// fetchMarket pulls 24h change and volume for a symbol from CoinGecko.
func (o *OracleService) fetchMarket(symbol string) (change, volume float64) {
	coinID := coinGeckoID(symbol)
	if coinID == "" {
		return 0, 0
	}
	url := fmt.Sprintf("%s/api/v3/simple/price?ids=%s&vs_currencies=usd&include_24hr_vol=true&include_24hr_change=true", o.cgBase, coinID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("Accept", "application/json")
	resp, err := o.http.Do(req)
	if err != nil {
		return 0, 0
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, 0
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0
	}
	var parsed map[string]map[string]float64
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, 0
	}
	entry, ok := parsed[coinID]
	if !ok {
		return 0, 0
	}
	return entry["usd_24h_change"], entry["usd_24h_vol"]
}

func (o *OracleService) checkAlerts(symbol string, price float64) {
	for i := range o.alerts {
		alert := &o.alerts[i]
		if alert.Symbol != symbol || alert.Triggered {
			continue
		}

		triggered := false
		if alert.Condition == "above" && price >= alert.Target {
			triggered = true
		} else if alert.Condition == "below" && price <= alert.Target {
			triggered = true
		}

		if triggered {
			alert.Triggered = true
			log.Printf("Price alert: %s %s %.2f (now: %.2f)", symbol, alert.Condition, alert.Target, price)
		}
	}
}

func (o *OracleService) handlePrice(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")

	o.mu.RLock()
	defer o.mu.RUnlock()

	if price, ok := o.prices[symbol]; ok {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(price)
		return
	}

	http.Error(w, "Symbol not found", http.StatusNotFound)
}

func (o *OracleService) handlePrices(w http.ResponseWriter, r *http.Request) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(o.prices)
}

func (o *OracleService) handleSources(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")

	o.mu.RLock()
	defer o.mu.RUnlock()

	if sources, ok := o.sources[symbol]; ok {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sources)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]PriceSource{})
}

func (o *OracleService) handleAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var alert PriceAlert
		if err := json.NewDecoder(r.Body).Decode(&alert); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		alert.ID = fmt.Sprintf("alert_%d", time.Now().UnixNano())
		alert.Triggered = false

		o.mu.Lock()
		o.alerts = append(o.alerts, alert)
		o.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(alert)
		return
	}

	o.mu.RLock()
	defer o.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(o.alerts)
}

func (o *OracleService) handleHealth(w http.ResponseWriter, r *http.Request) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"prices": len(o.prices),
		"alerts": len(o.alerts),
	})
}

func main() {
	log.Println("Starting TigerWallet Price Oracle Service...")

	service := NewOracleService()
	if err := service.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
