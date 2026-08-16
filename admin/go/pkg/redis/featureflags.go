package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Feature-flag enforcement shared contract.
//
// Redis is the shared feature-flag store (LaunchDarkly-style) that downstream
// services consult to enforce feature behavior. Admin backends WRITE flag
// state here; downstream services READ it. No code imports cross between
// admin apps and wallet apps — only this shared Redis namespace.
//
// Key:   tigerwallet:feature:<name>
// Value: "enabled" | "disabled" | "paused"   (string)
// TTL:   none (persistent; admin-controlled)
//
// A "paused" state means the feature is temporarily suspended (resume -> enabled).

const (
	FeatureStateEnabled  = "enabled"
	FeatureStateDisabled = "disabled"
	FeatureStatePaused   = "paused"

	featureKeyPrefix = "tigerwallet:feature:"
)

// FeatureStateKey returns the Redis key for a feature flag's live state.
func FeatureStateKey(name string) string {
	return featureKeyPrefix + name
}

// FeatureStateFromBool maps a boolean enabled flag to the canonical state
// string. true -> "enabled", false -> "disabled".
func FeatureStateFromBool(enabled bool) string {
	if enabled {
		return FeatureStateEnabled
	}
	return FeatureStateDisabled
}

// PublishFeatureState writes a feature flag's state to Redis (persistent, no
// TTL). This is the canonical write path that downstream services read.
func (r *RedisClient) PublishFeatureState(name, state string) error {
	if r == nil || r.Client == nil {
		return fmt.Errorf("redis client not initialized")
	}
	ctx := r.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return r.Client.Set(ctx, FeatureStateKey(name), state, 0).Err()
}

// GetFeatureState reads a feature flag's live state from Redis. Returns
// ("", nil) when the key is missing (fail-closed: callers treat unknown as
// disabled).
func (r *RedisClient) GetFeatureState(name string) (string, error) {
	if r == nil || r.Client == nil {
		return "", fmt.Errorf("redis client not initialized")
	}
	ctx := r.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	val, err := r.Client.Get(ctx, FeatureStateKey(name)).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil
		}
		return "", err
	}
	return val, nil
}

// DeleteFeatureState removes a feature flag's live state from Redis (used when
// a flag is deleted so downstream services fail closed).
func (r *RedisClient) DeleteFeatureState(name string) error {
	if r == nil || r.Client == nil {
		return fmt.Errorf("redis client not initialized")
	}
	ctx := r.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return r.Client.Del(ctx, FeatureStateKey(name)).Err()
}

// ValidFeatureState reports whether s is one of the canonical state values.
func ValidFeatureState(s string) bool {
	switch s {
	case FeatureStateEnabled, FeatureStateDisabled, FeatureStatePaused:
		return true
	}
	return false
}
