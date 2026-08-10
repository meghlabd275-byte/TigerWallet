package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// FetchOHLC fetches real OHLC candles from CoinGecko's /coins/{id}/ohlc
// endpoint. Returns chronological daily candles [time, open, high, low, close].
type OHLCPoint struct {
	Time  int64   `json:"time"`
	Open  float64 `json:"open"`
	High  float64 `json:"high"`
	Low   float64 `json:"low"`
	Close float64 `json:"close"`
}

func FetchOHLC(ctx context.Context, coinID string, days string) ([]OHLCPoint, error) {
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/coins/%s/ohlc?vs_currency=usd&days=%s", coinID, days)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	if key := appConfig.CoinGeckoAPIKey; key != "" {
		req.Header.Set("x-cg-pro-api-key", key)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var raw [][]float64
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	out := make([]OHLCPoint, 0, len(raw))
	for _, r := range raw {
		if len(r) < 5 {
			continue
		}
		out = append(out, OHLCPoint{
			Time:  int64(r[0]) / 1000, // ms -> s
			Open:  r[1],
			High:  r[2],
			Low:   r[3],
			Close: r[4],
		})
	}
	return out, nil
}

// FetchMarketChart fetches real historical prices from CoinGecko's
// /coins/{id}/market_chart endpoint and aggregates them into daily OHLC
// candles.
func FetchMarketChart(ctx context.Context, coinID string, days string) ([]OHLCPoint, error) {
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/coins/%s/market_chart?vs_currency=usd&days=%s", coinID, days)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	if key := appConfig.CoinGeckoAPIKey; key != "" {
		req.Header.Set("x-cg-pro-api-key", key)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var raw struct {
		Prices [][]float64 `json:"prices"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	dayMap := map[int64]*OHLCPoint{}
	var dayKeys []int64
	for _, p := range raw.Prices {
		if len(p) < 2 {
			continue
		}
		ts := int64(p[0]) / 1000
		dayStart := ts - (ts % 86400)
		price := p[1]
		c, ok := dayMap[dayStart]
		if !ok {
			c = &OHLCPoint{Time: dayStart, Open: price, High: price, Low: price, Close: price}
			dayMap[dayStart] = c
			dayKeys = append(dayKeys, dayStart)
		}
		c.Close = price
		if price > c.High {
			c.High = price
		}
		if price < c.Low {
			c.Low = price
		}
	}
	// Sort ascending by day.
	for i := 0; i < len(dayKeys); i++ {
		for j := i + 1; j < len(dayKeys); j++ {
			if dayKeys[j] < dayKeys[i] {
				dayKeys[i], dayKeys[j] = dayKeys[j], dayKeys[i]
			}
		}
	}
	out := make([]OHLCPoint, 0, len(dayKeys))
	for _, k := range dayKeys {
		out = append(out, *dayMap[k])
	}
	return out, nil
}
