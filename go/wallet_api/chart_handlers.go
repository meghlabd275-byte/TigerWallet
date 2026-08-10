package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// handleChartHistory returns real OHLC candles for a token from CoinGecko.
// Query params: coin (CoinGecko coin id, e.g. "ethereum"), days ("1","7","30","90").
func handleChartHistory(c *gin.Context) {
	coin := c.Query("coin")
	if coin == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "coin parameter required"})
		return
	}
	days := c.DefaultQuery("days", "30")
	candles, err := FetchOHLC(c.Request.Context(), coin, days)
	if err != nil || len(candles) == 0 {
		// Fall back to the market_chart endpoint which aggregates daily prices.
		mc, mErr := FetchMarketChart(c.Request.Context(), coin, days)
		if mErr != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "price history unavailable"})
			return
		}
		candles = mc
	}
	c.JSON(http.StatusOK, gin.H{"candles": candles})
}
