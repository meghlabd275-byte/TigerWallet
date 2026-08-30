package main

// price_fetcher.go — cluster-safe fiat valuation fetcher.
//
// FetchTokenPrice (fetchers.go) hits CoinGecko on every call. At global scale
// (billions of balance requests) that exhausts the upstream rate limit within
// seconds when every replica calls independently. This layer adds a two-level
// cache in front of it:
//
//   L1: in-process TTL cache (per replica, zero latency)
//   L2: shared Redis cache (cluster-wide single rate-limit budget — N
//       replicas serving the same coin only fetch upstream once per TTL)
//
// Fail-closed semantics are preserved: on any upstream/cache error the caller
// gets an error and omits USD fields — a price is never fabricated.

import (
        "context"
        "encoding/json"
        "fmt"
        "sync"
        "time"
)

const (
        // priceCacheTTL balances freshness against upstream rate limits.
        priceCacheTTL = 60 * time.Second
        // priceRedisKeyPrefix namespaces the shared cache entries.
        priceRedisKeyPrefix = "mw:price:"
)

type cachedPrice struct {
        p   *CoinGeckoPrice
        exp time.Time
}

var (
        priceCacheMu sync.Mutex
        priceCache   = map[string]cachedPrice{}
)

// FetchTokenPriceCached returns the CoinGecko price for coinID using the
// two-level cache. svc may have a nil store/redis (degraded mode): the cache
// degrades to L1-only with no error surface change.
func (svc *Service) FetchTokenPriceCached(ctx context.Context, coinID string) (*CoinGeckoPrice, error) {
        if coinID == "" {
                return nil, fmt.Errorf("no coin id")
        }
        now := time.Now()

        priceCacheMu.Lock()
        if cp, ok := priceCache[coinID]; ok && now.Before(cp.exp) {
                p := cp.p
                priceCacheMu.Unlock()
                return p, nil
        }
        priceCacheMu.Unlock()

        // L2: shared Redis cache (cluster-wide budget).
        if svc != nil && svc.store != nil && svc.store.redis != nil {
                rctx, cancel := context.WithTimeout(ctx, 2*time.Second)
                data, err := svc.store.redis.Get(rctx, priceRedisKeyPrefix+coinID).Bytes()
                cancel()
                if err == nil {
                        var p CoinGeckoPrice
                        if json.Unmarshal(data, &p) == nil && p.USD > 0 {
                                priceCacheMu.Lock()
                                priceCache[coinID] = cachedPrice{p: &p, exp: now.Add(priceCacheTTL)}
                                priceCacheMu.Unlock()
                                return &p, nil
                        }
                }
        }

        // Miss on both levels: single upstream fetch. Concurrent misses for the
        // same coin on one replica are single-flighted via a per-call lock strip.
        p, err := fetchTokenPriceSingleflight(ctx, coinID)
        if err != nil {
                return nil, err
        }

        priceCacheMu.Lock()
        priceCache[coinID] = cachedPrice{p: p, exp: now.Add(priceCacheTTL)}
        priceCacheMu.Unlock()

        if svc != nil && svc.store != nil && svc.store.redis != nil {
                if data, merr := json.Marshal(p); merr == nil {
                        rctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
                        _ = svc.store.redis.Set(rctx, priceRedisKeyPrefix+coinID, data, priceCacheTTL).Err()
                        cancel()
                }
        }
        return p, nil
}

// singleflight so a burst of concurrent requests for the same coin on one
// replica triggers exactly one upstream fetch.
var (
        sfMu     sync.Mutex
        sfCalls  = map[string]*sfCall{}
)

type sfCall struct {
        done chan struct{}
        p    *CoinGeckoPrice
        err  error
}

func fetchTokenPriceSingleflight(ctx context.Context, coinID string) (*CoinGeckoPrice, error) {
        sfMu.Lock()
        if c, ok := sfCalls[coinID]; ok {
                sfMu.Unlock()
                select {
                case <-c.done:
                        return c.p, c.err
                case <-ctx.Done():
                        return nil, ctx.Err()
                }
        }
        c := &sfCall{done: make(chan struct{})}
        sfCalls[coinID] = c
        sfMu.Unlock()

        c.p, c.err = FetchTokenPrice(ctx, coinID)
        close(c.done)

        sfMu.Lock()
        delete(sfCalls, coinID)
        sfMu.Unlock()
        return c.p, c.err
}
