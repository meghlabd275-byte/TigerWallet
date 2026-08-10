package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// coinGeckoID maps a trading symbol base (BTC, ETH, …) to a CoinGecko coin id.
// Falls back to the lowercased base for unknown symbols.
var symbolToCoinGecko = map[string]string{
	"BTC":   "bitcoin",
	"ETH":   "ethereum",
	"BNB":   "binancecoin",
	"SOL":   "solana",
	"XRP":   "ripple",
	"ADA":   "cardano",
	"DOT":   "polkadot",
	"MATIC": "matic-network",
	"AVAX":  "avalanche-2",
	"LINK":  "chainlink",
	"UNI":   "uniswap",
	"ATOM":  "cosmos",
	"DOGE":  "dogecoin",
	"LTC":   "litecoin",
	"TRX":   "tron",
	"ARB":   "arbitrum",
	"OP":    "optimism",
}

func coinIDForSymbol(sym string) string {
	base := strings.ToUpper(strings.Split(sym, "/")[0])
	if id, ok := symbolToCoinGecko[base]; ok {
		return id
	}
	return strings.ToLower(base)
}

// handleTerminalKline returns OHLC candles for a trading symbol (BASE/QUOTE).
// Query: ?days=1&interval=daily
func handleTerminalKline(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol required"})
		return
	}
	coinID := coinIDForSymbol(symbol)
	days := c.DefaultQuery("days", "1")
	candles, err := FetchOHLC(c.Request.Context(), coinID, days)
	if err != nil || len(candles) == 0 {
		mc, mErr := FetchMarketChart(c.Request.Context(), coinID, days)
		if mErr != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "kline unavailable", "symbol": symbol, "coin": coinID})
			return
		}
		candles = mc
	}
	c.JSON(http.StatusOK, gin.H{"symbol": symbol, "interval": "candle", "candles": candles})
}

// CoinGeckoMarket mirrors a row of the /coins/markets response.
type CoinGeckoMarket struct {
	Symbol        string  `json:"symbol"`
	CurrentPrice  float64 `json:"current_price"`
	MarketCap     float64 `json:"market_cap"`
	TotalVolume   float64 `json:"total_volume"`
	PriceChange24 float64 `json:"price_change_percentage_24h"`
	High24        float64 `json:"high_24h"`
	Low24         float64 `json:"low_24h"`
	LastUpdated   string  `json:"last_updated"`
}

// handleTerminalTicker returns the 24h market ticker for a trading symbol.
func handleTerminalTicker(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol required"})
		return
	}
	coinID := coinIDForSymbol(symbol)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	u := fmt.Sprintf("https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&ids=%s", coinID)
	req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
	if key := appConfig.CoinGeckoAPIKey; key != "" {
		req.Header.Set("x-cg-pro-api-key", key)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "ticker unavailable"})
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var markets []CoinGeckoMarket
	if err := json.Unmarshal(body, &markets); err != nil || len(markets) == 0 {
		c.JSON(http.StatusBadGateway, gin.H{"error": "ticker parse failed", "coin": coinID})
		return
	}
	m := markets[0]
	c.JSON(http.StatusOK, gin.H{
		"symbol": symbol, "coin": coinID,
		"last_price": m.CurrentPrice, "high_24h": m.High24, "low_24h": m.Low24,
		"change_24h_pct": m.PriceChange24, "volume_24h": m.TotalVolume,
		"market_cap": m.MarketCap, "updated": m.LastUpdated,
	})
}
