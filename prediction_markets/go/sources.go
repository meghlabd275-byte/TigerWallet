package main

// sources.go — upstream prediction-market feeds.
//
// Markets are read from the configured provider APIs and stored locally. When a
// provider is not configured or does not answer, the fetch reports an error and
// the caller serves whatever is already stored; nothing is invented to fill the
// gap.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// sourceTimeout bounds one upstream request.
const sourceTimeout = 20 * time.Second

// polymarketMarket mirrors the Gamma API market object.
type polymarketMarket struct {
	ID            string          `json:"id"`
	Question      string          `json:"question"`
	Description   string          `json:"description"`
	EndDate       string          `json:"endDate"`
	Image         string          `json:"image"`
	Icon          string          `json:"icon"`
	Slug          string          `json:"slug"`
	Active        bool            `json:"active"`
	Closed        bool            `json:"closed"`
	Archived      bool            `json:"archived"`
	Volume        json.RawMessage `json:"volume"`
	VolumeNum     float64         `json:"volumeNum"`
	LiquidityNum  float64         `json:"liquidityNum"`
	Outcomes      string          `json:"outcomes"`
	OutcomePrices string          `json:"outcomePrices"`
	ClobTokenIDs  string          `json:"clobTokenIds"`
	ConditionID   string          `json:"conditionId"`
}

// fetchPolymarketMarkets reads active markets from the Polymarket Gamma API.
func fetchPolymarketMarkets(ctx context.Context, client *http.Client, endpoint string, limit, offset int) ([]Market, []Outcome, error) {
	if endpoint == "" {
		return nil, nil, fmt.Errorf("the Polymarket API endpoint is not configured")
	}

	url := fmt.Sprintf("%s/markets?active=true&closed=false&limit=%d&offset=%d",
		strings.TrimRight(endpoint, "/"), limit, offset)

	reqCtx, cancel := context.WithTimeout(ctx, sourceTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("polymarket request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("polymarket returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, nil, err
	}

	var upstream []polymarketMarket
	if err := json.Unmarshal(body, &upstream); err != nil {
		return nil, nil, fmt.Errorf("decode polymarket response: %w", err)
	}

	now := time.Now().Unix()
	markets := make([]Market, 0, len(upstream))
	var outcomes []Outcome

	for _, m := range upstream {
		if m.ID == "" || m.Question == "" {
			continue
		}

		volume := m.VolumeNum
		if volume == 0 {
			volume = rawNumber(m.Volume)
		}

		webURL := fmt.Sprintf("https://polymarket.com/event/%s", m.Slug)
		if m.Slug == "" {
			webURL = fmt.Sprintf("https://polymarket.com/market/%s", m.ID)
		}

		markets = append(markets, Market{
			ID:          "polymarket:" + m.ID,
			Question:    m.Question,
			Description: m.Description,
			Source:      "polymarket",
			URL:         webURL,
			ImageURL:    firstNonEmpty(m.Icon, m.Image),
			EndDate:     parseRFC3339(m.EndDate),
			CreatedAt:   now,
			UpdatedAt:   now,
			Volume:      volume,
			VolumeUSD:   volume,
			Active:      m.Active && !m.Archived,
			Closed:      m.Closed,
		})

		outcomes = append(outcomes, decodePolymarketOutcomes(m)...)
	}

	return markets, outcomes, nil
}

// decodePolymarketOutcomes converts the JSON-encoded outcome arrays.
//
// The Gamma API returns outcomes and their prices as JSON strings inside the
// JSON document, so they need a second decode.
func decodePolymarketOutcomes(m polymarketMarket) []Outcome {
	var names, prices []string
	if m.Outcomes != "" {
		_ = json.Unmarshal([]byte(m.Outcomes), &names)
	}
	if m.OutcomePrices != "" {
		_ = json.Unmarshal([]byte(m.OutcomePrices), &prices)
	}
	if len(names) == 0 {
		return nil
	}

	outcomes := make([]Outcome, 0, len(names))
	for i, name := range names {
		var price float64
		if i < len(prices) {
			price, _ = strconv.ParseFloat(prices[i], 64)
		}
		outcomes = append(outcomes, Outcome{
			ID:          fmt.Sprintf("polymarket:%s:%d", m.ID, i),
			MarketID:    "polymarket:" + m.ID,
			Name:        name,
			Price:       price,
			Probability: price,
			Active:      m.Active && !m.Closed,
		})
	}
	return outcomes
}

// rawNumber reads a JSON value that may be a number or a numeric string.
func rawNumber(raw json.RawMessage) float64 {
	if len(raw) == 0 {
		return 0
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return f
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		parsed, _ := strconv.ParseFloat(s, 64)
		return parsed
	}
	return 0
}

// parseRFC3339 converts a timestamp to unix seconds, returning 0 when the value
// is absent or unparseable.
func parseRFC3339(value string) int64 {
	if value == "" {
		return 0
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05Z", "2006-01-02"} {
		if t, err := time.Parse(layout, value); err == nil {
			return t.Unix()
		}
	}
	return 0
}

// firstNonEmpty returns the first non-empty string.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
