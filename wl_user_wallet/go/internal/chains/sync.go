package chains

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Registry sync — the canonical chain registry is the UserWallet backend's
// GET /api/v1/chains (go/wallet_api), which layers master-wallet/admin-managed
// chain governance (PostgreSQL) on top of the embedded defaults. WL
// user-wallets sync from it so chain add/update/remove propagates without
// redeploying every WL client. The embedded registry remains the fail-open
// baseline: the WL product keeps working standalone when the upstream is
// unreachable (platform liveness is gated by the license service, not by the
// chain registry).
var (
	registryMu       sync.RWMutex
	upstreamOverride map[int64]ChainConfig
)

// StartRegistrySync fetches the canonical registry immediately and then every
// 5 minutes. An empty upstreamURL disables sync (pure embedded mode).
func StartRegistrySync(ctx context.Context, upstreamURL string) {
	upstreamURL = strings.TrimRight(strings.TrimSpace(upstreamURL), "/")
	if upstreamURL == "" {
		return
	}
	client := &http.Client{Timeout: 5 * time.Second}
	fetch := func() {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, upstreamURL+"/api/v1/chains", nil)
		if err != nil {
			return
		}
		resp, err := client.Do(req)
		if err != nil {
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return
		}
		var out struct {
			Chains []ChainConfig `json:"chains"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return
		}
		if len(out.Chains) == 0 {
			return // never replace a good registry with an empty one
		}
		m := make(map[int64]ChainConfig, len(out.Chains))
		for _, ch := range out.Chains {
			m[ch.ID] = ch
		}
		registryMu.Lock()
		upstreamOverride = m
		registryMu.Unlock()
		log.Printf("chains: synced %d chains from canonical registry %s", len(m), upstreamURL)
	}
	go func() {
		fetch()
		t := time.NewTicker(5 * time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				fetch()
			}
		}
	}()
}

// mergedChains returns the embedded registry overlaid with the synced upstream
// registry (upstream wins per chain ID; upstream-only chains are added).
func mergedChains() map[int64]ChainConfig {
	registryMu.RLock()
	defer registryMu.RUnlock()
	if len(upstreamOverride) == 0 {
		return SupportedChains
	}
	out := make(map[int64]ChainConfig, len(upstreamOverride)+len(SupportedChains))
	for id, ch := range SupportedChains {
		out[id] = ch
	}
	for id, ch := range upstreamOverride {
		out[id] = ch
	}
	return out
}
