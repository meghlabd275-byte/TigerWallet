// feature_flags.go — downstream feature-flag enforcement layer.
//
// Mirrors go/wallet_api/feature_flags.go. Redis is the SHARED feature-flag
// store. Admin backends WRITE flag state to Redis; downstream services (this
// one) READ it. Only the shared Redis namespace crosses between admin and
// wallet apps:
//
//	Key:   tigerwallet:feature:<name>
//	Value: "enabled" | "disabled" | "paused"
//	TTL:   none (persistent; admin-controlled)
//
// Enforcement is fail-closed: any missing/unknown/erroring state is treated as
// disabled, so an admin toggling a feature off (or Redis being unreachable)
// halts the gated behavior rather than letting it through.
package main

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const (
	flagStateEnabled  = "enabled"
	flagStateDisabled = "disabled"
	flagStatePaused   = "paused"
)

const flagKeyPrefix = "tigerwallet:feature:"

func flagKey(name string) string { return flagKeyPrefix + name }

// nft_marketplace is replaced per-service with the canonical feature flag name.
const GatedFeature = "nft_marketplace"

const flagCacheTTL = 5 * time.Second

type flagCacheEntry struct {
	state     string
	fetchedAt time.Time
}

var (
	flagCacheMu sync.Mutex
	flagCache   = make(map[string]flagCacheEntry)
)

func (ns *NFTService) fetchFeatureState(featureName string) string {
	if ns == nil || ns.redis == nil {
		return flagStateDisabled
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	val, err := ns.redis.Get(ctx, flagKey(featureName)).Result()
	if err != nil {
		return flagStateDisabled
	}
	switch val {
	case flagStateEnabled, flagStateDisabled, flagStatePaused:
		return val
	default:
		return flagStateDisabled
	}
}

// FeatureState returns the cached-then-live state string for the feature.
func (ns *NFTService) FeatureState(featureName string) string {
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
	state := ns.fetchFeatureState(featureName)
	flagCacheMu.Lock()
	flagCache[featureName] = flagCacheEntry{state: state, fetchedAt: now}
	flagCacheMu.Unlock()
	return state
}

func (ns *NFTService) isFeatureEnabled(featureName string) bool {
	return ns.FeatureState(featureName) == flagStateEnabled
}

// enforceFeature aborts with HTTP 423 when the feature is not enabled. Usage:
//
//	if !ns.enforceFeature(c, GatedFeature) { return }
func (ns *NFTService) enforceFeature(c *gin.Context, featureName string) bool {
	if ns.isFeatureEnabled(featureName) {
		return true
	}
	state := ns.FeatureState(featureName)
	if state == "" {
		state = flagStateDisabled
	}
	c.AbortWithStatusJSON(http.StatusLocked, gin.H{
		"error": "feature " + featureName + " is currently " + state,
	})
	return false
}

// featureGate is a gin middleware enforcing the flag on a whole route group.
func (ns *NFTService) featureGate(featureName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ns.enforceFeature(c, featureName) {
			return
		}
		c.Next()
	}
}

var _ = redis.Nil // keep the redis import referenced for the cache type above
