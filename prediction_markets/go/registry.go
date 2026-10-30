package main

// registry.go — the configured market sources and the poller that keeps the
// stored markets current.
//
// A source is registered only when its endpoint is configured. The poller
// writes what the providers return; if a provider is unreachable the stored
// markets are left untouched and the failure is logged, so the wallet never
// shows a market that does not exist.

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"
)

// marketSource reads markets from one upstream provider.
type marketSource struct {
	name  string
	fetch func(ctx context.Context, client *http.Client, endpoint string, limit, offset int) ([]Market, []Outcome, error)
}

// newMarketSourceRegistry returns the sources whose endpoints are configured.
func newMarketSourceRegistry(cfg *Config) []marketSource {
	var sources []marketSource

	if cfg.PolymarketAPI != "" {
		sources = append(sources, marketSource{
			name:  "polymarket",
			fetch: fetchPolymarketMarkets,
		})
	}

	return sources
}

// endpointForSource returns the configured endpoint for a source name.
func (c *Config) endpointForSource(name string) string {
	switch name {
	case "polymarket":
		return c.PolymarketAPI
	default:
		return ""
	}
}

// marketPollPageSize bounds how many markets are read per provider call.
const marketPollPageSize = 100

// marketPollPages bounds how many pages are read per source per cycle.
const marketPollPages = 5

// cacheGeneration is incremented after every refresh so the cached read paths
// are bypassed. A counter is used because Redis cannot delete a key by prefix
// without scanning the whole keyspace.
var cacheGeneration atomic.Uint64

// pollMarkets refreshes the stored markets on an interval until ctx is done.
func pollMarkets(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 60 * time.Second
	}

	client := &http.Client{Timeout: sourceTimeout}

	// An initial refresh runs immediately so a fresh deployment is not empty
	// until the first tick.
	refreshMarkets(ctx, client)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			refreshMarkets(ctx, client)
		}
	}
}

// refreshMarkets reads every configured source and stores the result.
func refreshMarkets(ctx context.Context, client *http.Client) {
	if len(marketSources) == 0 {
		return
	}

	for _, source := range marketSources {
		endpoint := config.endpointForSource(source.name)
		if endpoint == "" {
			continue
		}

		var total int
		for page := 0; page < marketPollPages; page++ {
			markets, outcomes, err := source.fetch(ctx, client, endpoint, marketPollPageSize, page*marketPollPageSize)
			if err != nil {
				logger.Warn().Err(err).Str("source", source.name).Int("page", page).
					Msg("market refresh failed; the stored markets are unchanged")
				break
			}

			for i := range markets {
				if err := store.UpsertMarket(ctx, markets[i]); err != nil {
					logger.Error().Err(err).Str("marketId", markets[i].ID).Msg("store market failed")
				}
			}
			for i := range outcomes {
				if err := store.UpsertOutcome(ctx, outcomes[i]); err != nil {
					logger.Error().Err(err).Str("outcomeId", outcomes[i].ID).Msg("store outcome failed")
				}
			}

			total += len(markets)
			if len(markets) < marketPollPageSize {
				break
			}
		}

		logger.Info().Str("source", source.name).Int("markets", total).Msg("market refresh complete")
	}

	// Invalidate the cached read paths.
	cacheGeneration.Add(1)
}
