// feature_flags.go — downstream feature-flag enforcement layer for wallet_api.
//
// Redis is the SHARED feature-flag store (LaunchDarkly-style). Admin backends
// (admin/go, super_admin/go, white_label_admin/go) WRITE flag state to Redis;
// downstream services (this one) READ it. No code imports cross between admin
// apps and wallet apps — only the shared Redis namespace:
//
//	Key:   tigerwallet:feature:<name>
//	Value: "enabled" | "disabled" | "paused"   (string)
//	TTL:   none (persistent; admin-controlled)
//
// Enforcement is default-ENABLED (blacklist semantics): builtin features run
// continuously with no bootstrap step; only an explicit operator
// "disabled"/"paused" decision gates a feature. A Redis outage fails open so
// an infra blip never halts trading by itself.
package main

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// Canonical feature-flag states (must match the admin backends' redis package
// constants). These are the only values a downstream service will honor.
const (
	featureStateEnabled  = "enabled"
	featureStateDisabled = "disabled"
	featureStatePaused   = "paused"
)

const featureKeyPrefix = "tigerwallet:feature:"

func featureKey(name string) string {
	return featureKeyPrefix + name
}

// Gated feature names. These gate the matching wallet_api operations.
const (
	FeatureSwapTrading         = "swap_trading"         // /swap/quote, /swap/execute, /amm/*
	FeatureSendTransactions    = "send_transactions"    // /send
	FeatureStaking             = "staking"              // /staking/*
	FeatureLending             = "lending"              // /lending/* (when present)
	FeatureNFTTransfer         = "nft_transfer"         // /nft/transfer
	FeatureAccountAbstraction  = "account_abstraction"  // /aa/*
	FeatureBridge              = "bridge"               // /bridge/*
	FeatureFiatOnramp          = "fiat_onramp"          // /ramp/* onramp
	FeatureFiatOfframp         = "fiat_offramp"         // /ramp/* offramp
)

// featureCacheTTL bounds how long a fetched state is trusted in-memory before
// re-querying Redis. Keeps hot paths from hammering Redis on every request
// while still converging within a few seconds of an admin toggle.
const featureCacheTTL = 5 * time.Second

type featureCacheEntry struct {
	state     string
	fetchedAt time.Time
}

var (
	featureCacheMu sync.Mutex
	featureCache   = make(map[string]featureCacheEntry)
)

// redisClientForFlags returns the wallet_api Redis client (store.Redis) used to
// read flag state. Returns nil when the store / Redis is not initialized
// (fail-closed downstream).
func redisClientForFlags() *redis.Client {
	if store == nil || store.Redis == nil {
		return nil
	}
	return store.Redis
}

// FeatureState returns the raw live state string ("enabled" | "disabled" |
// "paused") for the named feature, as read from Redis. Default-enabled:
// missing/unknown/erroring keys resolve to "enabled" (seamless continuous
// trading); only an explicit operator stop/pause is enforced.
func FeatureState(featureName string) string {
	if featureName == "" {
		return featureStateDisabled
	}

	now := time.Now()
	featureCacheMu.Lock()
	if entry, ok := featureCache[featureName]; ok && now.Sub(entry.fetchedAt) < featureCacheTTL {
		featureCacheMu.Unlock()
		return entry.state
	}
	featureCacheMu.Unlock()

	state := fetchFeatureState(featureName)

	featureCacheMu.Lock()
	featureCache[featureName] = featureCacheEntry{state: state, fetchedAt: now}
	featureCacheMu.Unlock()
	return state
}

// fetchFeatureState reads the live state from Redis. Default-ENABLED: a
// missing key or Redis outage resolves to "enabled" — owner policy is that
// every user can perform all swap and trading continuously with no bootstrap
// step; an infra blip never halts trading by itself. Only an explicit
// operator decision ("disabled"/"paused") gates a feature.
func fetchFeatureState(featureName string) string {
	rdb := redisClientForFlags()
	if rdb == nil {
		return featureStateEnabled
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	val, err := rdb.Get(ctx, featureKey(featureName)).Result()
	if err != nil {
		// redis.Nil (never explicitly set) or network error -> default-on.
		return featureStateEnabled
	}
	switch strings.ToLower(val) {
	case featureStateEnabled, featureStateDisabled, featureStatePaused:
		return strings.ToLower(val)
	default:
		// Unknown value -> default-on (continuity); only the three
		// explicit operator states are enforced.
		return featureStateEnabled
	}
}

// IsFeatureEnabled returns true unless the feature was explicitly disabled or
// paused by an operator. Missing/unknown/unset flags default to enabled so
// builtin swap/trading is seamless and continuous for every user.
func IsFeatureEnabled(featureName string) bool {
	return FeatureState(featureName) == featureStateEnabled
}

// InvalidateFeatureCache drops the cached state for a feature, forcing the next
// read to hit Redis. Useful in tests; in production the 5s TTL suffices.
func InvalidateFeatureCache(featureName string) {
	featureCacheMu.Lock()
	delete(featureCache, featureName)
	featureCacheMu.Unlock()
}

// clearFeatureCacheForTests resets all cached state. Test-only.
func clearFeatureCacheForTests() {
	featureCacheMu.Lock()
	featureCache = make(map[string]featureCacheEntry)
	featureCacheMu.Unlock()
}

// enforceFeature returns false and writes the HTTP 423 Locked response when the
// feature was explicitly disabled/paused by an operator. Returns true when the
// feature is enabled — including the default-on unset state (caller may proceed). Designed to sit at the top of gated handlers:
//
//	if !enforceFeature(c, FeatureSwapTrading) { return }
func enforceFeature(c *gin.Context, featureName string) bool {
	if IsFeatureEnabled(featureName) {
		return true
	}
	state := FeatureState(featureName)
	if state == "" {
		state = featureStateDisabled
	}
	c.AbortWithStatusJSON(http.StatusLocked, gin.H{
		"error": "feature " + featureName + " is currently " + state,
	})
	return false
}
