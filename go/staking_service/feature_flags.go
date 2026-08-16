// feature_flags.go — downstream feature-flag enforcement layer for the
// staking service.
//
// Mirrors go/wallet_api/feature_flags.go. Redis is the SHARED feature-flag
// store (LaunchDarkly-style). Admin backends WRITE flag state to Redis;
// downstream services (this one) READ it. Only the shared Redis namespace
// crosses between admin and wallet apps:
//
//      Key:   tigerwallet:feature:<name>
//      Value: "enabled" | "disabled" | "paused"   (string)
//      TTL:   none (persistent; admin-controlled)
//
// Enforcement is fail-closed (matches wallet_api): any missing/unknown/erroring
// state is treated as disabled, so an admin toggling a feature off (or Redis
// being unreachable) halts the gated behavior rather than letting it through.
// For a downstream service this trades availability for the safer security
// posture of honoring admin intent consistently across the platform.
package main

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// Canonical feature-flag states (must match the admin backends' redis package
// constants). These are the only values a downstream service will honor.
const (
	flagStateEnabled  = "enabled"
	flagStateDisabled = "disabled"
	flagStatePaused   = "paused"
)

const flagKeyPrefix = "tigerwallet:feature:"

func flagKey(name string) string {
	return flagKeyPrefix + name
}

// Gated feature name for this service.
const FeatureStaking = "staking"

// flagCacheTTL bounds how long a fetched state is trusted in-memory before
// re-querying Redis. Keeps hot paths from hammering Redis on every request
// while still converging within a few seconds of an admin toggle.
const flagCacheTTL = 5 * time.Second

type flagCacheEntry struct {
	state     string
	fetchedAt time.Time
}

var (
	flagCacheMu sync.Mutex
	flagCache   = make(map[string]flagCacheEntry)
)

// redisClientForFlags returns the staking service Redis client used to read
// flag state. Returns nil when the service / Redis is not initialized
// (fail-closed downstream).
func (ss *StakingService) redisClientForFlags() *redis.Client {
	if ss == nil || ss.redis == nil {
		return nil
	}
	return ss.redis
}

// FeatureState returns the raw live state string ("enabled" | "disabled" |
// "paused") for the named feature, as read from Redis. Fail-closed: returns
// "disabled" for missing/unknown/erroring keys.
func (ss *StakingService) FeatureState(featureName string) string {
	if featureName == "" {
		return flagStateDisabled
	}

	now := time.Now()
	flagCacheMu.Lock()
	if entry, ok := flagCache[featureName]; ok && now.Sub(entry.fetchedAt) < flagCacheTTL {
		flagCacheMu.Unlock()
		return entry.state
	}
	flagCacheMu.Unlock()

	state := ss.fetchFeatureState(featureName)

	flagCacheMu.Lock()
	flagCache[featureName] = flagCacheEntry{state: state, fetchedAt: now}
	flagCacheMu.Unlock()
	return state
}

// fetchFeatureState reads the live state from Redis. Fail-closed: any error or
// missing key resolves to "disabled".
func (ss *StakingService) fetchFeatureState(featureName string) string {
	rdb := ss.redisClientForFlags()
	if rdb == nil {
		return flagStateDisabled
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	val, err := rdb.Get(ctx, flagKey(featureName)).Result()
	if err != nil {
		// redis.Nil (missing), network error, etc. -> fail-closed disabled.
		return flagStateDisabled
	}
	switch val {
	case flagStateEnabled, flagStateDisabled, flagStatePaused:
		return val
	default:
		// Unknown value -> fail-closed disabled.
		return flagStateDisabled
	}
}

// IsFeatureEnabled returns true ONLY when the feature's live state is exactly
// "enabled". disabled / paused / missing / unknown all return false (fail-closed).
func (ss *StakingService) IsFeatureEnabled(featureName string) bool {
	return ss.FeatureState(featureName) == flagStateEnabled
}

// InvalidateFeatureCache drops the cached state for a feature, forcing the next
// read to hit Redis. Useful in tests; in production the 5s TTL suffices.
func (ss *StakingService) InvalidateFeatureCache(featureName string) {
	flagCacheMu.Lock()
	delete(flagCache, featureName)
	flagCacheMu.Unlock()
}

// enforceFeature returns false and writes the HTTP 423 Locked response when the
// feature is disabled/paused/missing. Returns true when the feature is enabled
// (caller may proceed). Designed to sit at the top of gated handlers:
//
//	if !ss.enforceFeature(c, FeatureStaking) { return }
func (ss *StakingService) enforceFeature(c *gin.Context, featureName string) bool {
	if ss.IsFeatureEnabled(featureName) {
		return true
	}
	state := ss.FeatureState(featureName)
	if state == "" {
		state = flagStateDisabled
	}
	c.AbortWithStatusJSON(http.StatusLocked, gin.H{
		"error": "feature " + featureName + " is currently " + state,
	})
	return false
}
