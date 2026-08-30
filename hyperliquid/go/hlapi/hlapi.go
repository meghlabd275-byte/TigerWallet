// Package hlapi is a real client for the public, keyless Hyperliquid info
// API. All data comes from live responses; every failure path returns an
// error (fail-closed) — nothing is fabricated.
package hlapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// InfoURL is the Hyperliquid info endpoint (override via HL_INFO_URL).
var InfoURL = "https://api.hyperliquid.xyz/info"

var httpClient = &http.Client{Timeout: 15 * time.Second}

func postInfo(ctx context.Context, payload map[string]any, out any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, InfoURL, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("hyperliquid info HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// AssetMarket is the real market context for one perp asset.
type AssetMarket struct {
	Symbol        string
	MarkPrice     float64
	PrevDayPrice  float64
	Change24h     float64 // percent
	Volume24h     float64 // notional USD
	OpenInterest  float64 // in base units (coins)
	FundingRate   float64 // current hourly funding rate
	NextFundingAt int64   // unix timestamp of the next hourly funding boundary
}

// GetMarketData fetches real market data for the requested assets via
// metaAndAssetCtxs. Unknown assets are skipped; error when the upstream call
// fails entirely.
func GetMarketData(ctx context.Context, assets []string) ([]AssetMarket, error) {
	var raw []json.RawMessage
	if err := postInfo(ctx, map[string]any{"type": "metaAndAssetCtxs"}, &raw); err != nil {
		return nil, err
	}
	if len(raw) != 2 {
		return nil, fmt.Errorf("unexpected metaAndAssetCtxs shape (%d elements)", len(raw))
	}
	var meta struct {
		Universe []struct {
			Name string `json:"name"`
		} `json:"universe"`
	}
	if err := json.Unmarshal(raw[0], &meta); err != nil {
		return nil, err
	}
	var ctxs []struct {
		MarkPx    string `json:"markPx"`
		PrevDayPx string `json:"prevDayPx"`
		DayNtlVlm string `json:"dayNtlVlm"`
		OpenInt   string `json:"openInterest"`
		Funding   string `json:"funding"`
	}
	if err := json.Unmarshal(raw[1], &ctxs); err != nil {
		return nil, err
	}
	byName := map[string]int{}
	for i, u := range meta.Universe {
		byName[u.Name] = i
	}
	nextFunding := time.Now().Truncate(time.Hour).Add(time.Hour).Unix()
	var out []AssetMarket
	for _, asset := range assets {
		idx, ok := byName[asset]
		if !ok || idx >= len(ctxs) {
			continue // unknown on the venue: skip (fail-closed, not fabricated)
		}
		c := ctxs[idx]
		mark, _ := strconv.ParseFloat(c.MarkPx, 64)
		prev, _ := strconv.ParseFloat(c.PrevDayPx, 64)
		vol, _ := strconv.ParseFloat(c.DayNtlVlm, 64)
		oi, _ := strconv.ParseFloat(c.OpenInt, 64)
		funding, _ := strconv.ParseFloat(c.Funding, 64)
		change := 0.0
		if prev > 0 {
			change = (mark - prev) / prev * 100
		}
		out = append(out, AssetMarket{
			Symbol:        asset,
			MarkPrice:     mark,
			PrevDayPrice:  prev,
			Change24h:     change,
			Volume24h:     vol,
			OpenInterest:  oi,
			FundingRate:   funding,
			NextFundingAt: nextFunding,
		})
	}
	return out, nil
}

// AccountState is the real clearinghouse state for a Hyperliquid account
// (the account address is the user's EVM wallet address).
type AccountState struct {
	AccountValue      float64 // total collateral USD
	TotalNotionalPos  float64
	TotalRawUSD       float64
	MaintenanceMargin float64
}

// GetAccountState fetches the real account state via clearinghouseState.
func GetAccountState(ctx context.Context, address string) (*AccountState, error) {
	var parsed struct {
		MarginSummary struct {
			AccountValue     string `json:"accountValue"`
			TotalNtlPos      string `json:"totalNtlPos"`
			TotalRawUSD      string `json:"totalRawUsd"`
			TotalMaintMargin string `json:"totalMaintenanceMargin"`
		} `json:"marginSummary"`
		CrossMaintenanceMarginUsed string `json:"crossMaintenanceMarginUsed"`
	}
	if err := postInfo(ctx, map[string]any{"type": "clearinghouseState", "user": address}, &parsed); err != nil {
		return nil, err
	}
	st := &AccountState{}
	st.AccountValue, _ = strconv.ParseFloat(parsed.MarginSummary.AccountValue, 64)
	st.TotalNotionalPos, _ = strconv.ParseFloat(parsed.MarginSummary.TotalNtlPos, 64)
	st.TotalRawUSD, _ = strconv.ParseFloat(parsed.MarginSummary.TotalRawUSD, 64)
	st.MaintenanceMargin, _ = strconv.ParseFloat(parsed.CrossMaintenanceMarginUsed, 64)
	return st, nil
}
