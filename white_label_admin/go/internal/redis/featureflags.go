package redis

import "fmt"

// Feature flag states. These are the string values written to the shared
// Redis key `tigerwallet:feature:<name>`, read fail-closed by downstream
// services (wallet_api etc.).
const (
	StateEnabled  = "enabled"
	StateDisabled = "disabled"
	StatePaused   = "paused"
)

const featureKeyPrefix = "tigerwallet:feature:"

// FeatureKey returns the Redis key for a feature flag.
func FeatureKey(name string) string {
	return featureKeyPrefix + name
}

// FeatureStateFromBool maps a boolean is_enabled to the canonical state string.
func FeatureStateFromBool(isEnabled bool) string {
	if isEnabled {
		return StateEnabled
	}
	return StateDisabled
}

// PublishFeatureState writes the feature flag's live state to Redis (no TTL).
func (r *RedisClient) PublishFeatureState(name, state string) error {
	if r == nil || r.Client == nil || name == "" {
		return fmt.Errorf("redis client or feature name not set")
	}
	return r.Client.Set(r.Ctx, FeatureKey(name), state, 0).Err()
}

// DeleteFeatureState removes the feature flag's live state from Redis.
func (r *RedisClient) DeleteFeatureState(name string) error {
	if r == nil || r.Client == nil || name == "" {
		return fmt.Errorf("redis client or feature name not set")
	}
	return r.Client.Del(r.Ctx, FeatureKey(name)).Err()
}

// GetFeatureState reads the live state of a feature flag. Returns ("", false)
// if missing or on error (fail-closed).
func (r *RedisClient) GetFeatureState(name string) (string, bool) {
	if r == nil || r.Client == nil || name == "" {
		return "", false
	}
	val, err := r.Client.Get(r.Ctx, FeatureKey(name)).Result()
	if err != nil {
		return "", false
	}
	return val, true
}
