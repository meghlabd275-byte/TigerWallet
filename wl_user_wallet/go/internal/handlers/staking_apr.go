package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// Staking APY is never a constant in this service. A staking protocol pays a
// variable rate that moves with network conditions, so the rate is read from
// the protocol's own API and cached briefly. When no provider answers, the
// rate is reported as null rather than an invented number.

// stakingAprProvider describes how to read the current rate for one protocol.
type stakingAprProvider struct {
	// URL is the provider's rate endpoint. A {chain_id} placeholder is
	// substituted when present.
	URL string
	// JSONField is the dot-path to the decimal rate inside the response body,
	// for example "data.apr" for {"data":{"apr":2.248}}.
	JSONField string
	// Percent means the provider already returns a percentage. Otherwise the
	// value is a ratio and is multiplied by 100.
	Percent bool
}

var stakingAprProviders = map[string]stakingAprProvider{
	"lido": {
		URL:       "https://eth-api.lido.fi/v1/protocol/steth/apr/last",
		JSONField: "data.apr",
		Percent:   true,
	},
	"rocketpool": {
		URL:       "https://api.rocketpool.net/api/apr",
		JSONField: "yearlyAPR",
		// The field is already expressed in percent, for example 1.927.
		Percent: true,
	},
}

const stakingAprTTL = 5 * time.Minute

type stakingAprCacheEntry struct {
	value   float64
	ok      bool
	expires time.Time
}

var (
	stakingAprMu    sync.Mutex
	stakingAprCache = map[string]stakingAprCacheEntry{}
)

// stakingApr returns the current rate in percent for a provider key. The
// second return value is false when no live rate could be read; callers must
// then report the rate as unknown instead of substituting a number.
func stakingApr(ctx context.Context, provider string) (float64, bool) {
	p, ok := stakingAprProviders[provider]
	if !ok {
		return 0, false
	}

	stakingAprMu.Lock()
	if e, hit := stakingAprCache[provider]; hit && time.Now().Before(e.expires) {
		stakingAprMu.Unlock()
		return e.value, e.ok
	}
	stakingAprMu.Unlock()

	value, ok := fetchStakingApr(ctx, p)

	stakingAprMu.Lock()
	stakingAprCache[provider] = stakingAprCacheEntry{value: value, ok: ok, expires: time.Now().Add(stakingAprTTL)}
	stakingAprMu.Unlock()

	return value, ok
}

func fetchStakingApr(ctx context.Context, p stakingAprProvider) (float64, bool) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.URL, nil)
	if err != nil {
		return 0, false
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "TigerWallet/1.0")

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return 0, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, false
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if err != nil {
		return 0, false
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return 0, false
	}

	v, found := jsonPath(raw, p.JSONField)
	if !found {
		return 0, false
	}

	value, err := strconv.ParseFloat(v, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, false
	}
	if !p.Percent {
		value *= 100
	}
	// A negative rate means the provider payload is not what we expect.
	if value < 0 || value > 1000 {
		return 0, false
	}
	return value, true
}

// jsonPath walks a decoded JSON object along a dot-separated path and returns
// the value as a string. Numbers and numeric strings are both accepted.
func jsonPath(obj map[string]any, path string) (string, bool) {
	cur := any(obj)
	for _, seg := range splitDots(path) {
		m, ok := cur.(map[string]any)
		if !ok {
			return "", false
		}
		cur, ok = m[seg]
		if !ok {
			return "", false
		}
	}
	switch v := cur.(type) {
	case string:
		return v, true
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), true
	case json.Number:
		return v.String(), true
	default:
		return "", false
	}
}

func splitDots(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == '.' {
			out = append(out, cur)
			cur = ""
			continue
		}
		cur += string(r)
	}
	return append(out, cur)
}

// stakingAprConfigured reports whether the operator has bound a provider key
// to a staking asset through STAKING_APR_PROVIDER_<SYMBOL>. The value is the
// provider key (lido, rocketpool), or a URL when it contains a scheme.
func stakingAprConfigured(symbol string) (string, bool) {
	v := os.Getenv("STAKING_APR_PROVIDER_" + sanitizeEnvSuffix(symbol))
	if v == "" {
		return "", false
	}
	return v, true
}

func sanitizeEnvSuffix(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z':
			out = append(out, c-32)
		case c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
			out = append(out, c)
		default:
			out = append(out, '_')
		}
	}
	return string(out)
}

// resolveStakingApr looks up the live rate for an asset. The provider is
// chosen by configuration; when the operator supplies a raw URL the JSON
// field can be selected with STAKING_APR_FIELD_<SYMBOL>.
func resolveStakingApr(ctx context.Context, symbol string) (float64, bool) {
	key, ok := stakingAprConfigured(symbol)
	if !ok {
		return 0, false
	}
	if _, known := stakingAprProviders[key]; known {
		return stakingApr(ctx, key)
	}
	// Treat the value as a direct URL.
	field := os.Getenv("STAKING_APR_FIELD_" + sanitizeEnvSuffix(symbol))
	percent := true
	if raw := os.Getenv("STAKING_APR_IS_RATIO_" + sanitizeEnvSuffix(symbol)); raw == "1" || raw == "true" {
		percent = false
	}
	value, ok := fetchStakingApr(ctx, stakingAprProvider{URL: key, JSONField: field, Percent: percent})
	if !ok {
		return 0, false
	}
	return value, true
}

// aprOrNil renders a rate for a JSON response, or nil when the rate is not
// known so the client shows "unavailable" rather than a stale constant.
func aprOrNil(value float64, ok bool) any {
	if !ok {
		return nil
	}
	return fmt.Sprintf("%.4f", value)
}
