package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tigerwallet/wl-user-wallet/internal/onchain"
)

// symbolToCoinGeckoID maps common ticker symbols to CoinGecko coin ids for the
// /price endpoint when the client passes ?symbol=ETH rather than ?coin=bitcoin.
var symbolToCoinGeckoID = map[string]string{
	"ETH":   "ethereum",
	"BTC":   "bitcoin",
	"BNB":   "binancecoin",
	"MATIC": "matic-network",
	"POL":   "matic-network",
	"SOL":   "solana",
	"ADA":   "cardano",
	"XRP":   "ripple",
	"DOT":   "polkadot",
	"AVAX":  "avalanche-2",
	"LINK":  "chainlink",
	"UNI":   "uniswap",
	"USDC":  "usd-coin",
	"USDT":  "tether",
	"DAI":   "dai",
	"ATOM":  "cosmos",
	"ARB":   "arbitrum",
	"OP":    "optimism",
	"WBTC":  "wrapped-bitcoin",
}

// GET /price?symbol=ETH or /price?coin=bitcoin — real CoinGecko simple-price.
// Fail-closed 503 on upstream error.
func (s *Svc) GetPrice(c *gin.Context) {
	coinID := strings.ToLower(strings.TrimSpace(c.Query("coin")))
	if coinID == "" {
		if sym := strings.ToUpper(strings.TrimSpace(c.Query("symbol"))); sym != "" {
			if id, ok := symbolToCoinGeckoID[sym]; ok {
				coinID = id
			} else {
				coinID = strings.ToLower(sym)
			}
		}
	}
	if coinID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol or coin required"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 12*time.Second)
	defer cancel()
	price, err := onchain.FetchTokenPrice(ctx, coinID, s.cfg.CoinGeckoAPIKey)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "price fetch failed: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"coin": coinID, "usd": price.PriceUSD, "usd_24h_change": price.Change24})
}
