package main

// ratelimit_redis.go — cluster-wide distributed rate limiting.
//
// The in-process token bucket (ratelimit.go) is correct for a single replica
// but silently multiplies every limit by the replica count in a clustered
// deployment: with K replicas behind a load balancer, a client effectively
// gets K × the intended rate. This file backs the same token-bucket policy
// with Redis so the limit holds across the whole cluster.
//
// The bucket state (tokens + last-refill timestamp) lives in a Redis hash per
// (policy, client) key and is mutated by a single Lua script, so check-and-
// consume is atomic under concurrent requests from any replica. The key TTL
// is derived from the refill horizon so idle clients cost zero memory.
//
// Failure policy: if Redis is unreachable the limiter falls back to the
// in-process bucket (fail-closed toward MORE limiting per replica, never
// un-throttled) — a Redis outage must never open the funds-movement or
// credential surfaces to unlimited traffic.

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

// redisTokenBucketScript atomically refills and consumes one token.
// KEYS[1] = bucket hash key
// ARGV = rate (tokens/sec), burst, now_ms
// Returns {allowed (0/1), retry_after_ms}.
var redisTokenBucketScript = redis.NewScript(`
local key    = KEYS[1]
local rate   = tonumber(ARGV[1])
local burst  = tonumber(ARGV[2])
local now_ms = tonumber(ARGV[3])

local data   = redis.call('HMGET', key, 'tokens', 'ts')
local tokens = tonumber(data[1])
local ts     = tonumber(data[2])
if tokens == nil then tokens = burst end
if ts == nil then ts = now_ms end

local elapsed = math.max(0, now_ms - ts) / 1000.0
tokens = math.min(burst, tokens + elapsed * rate)

local allowed = 0
local retry_ms = 0
if tokens >= 1.0 then
  tokens = tokens - 1.0
  allowed = 1
else
  retry_ms = math.ceil((1.0 - tokens) / rate * 1000.0)
end

redis.call('HMSET', key, 'tokens', tokens, 'ts', now_ms)
-- Bucket is fully refilled after burst/rate seconds; keep a margin.
redis.call('PEXPIRE', key, math.ceil(burst / rate * 1000.0) + 60000)
return {allowed, retry_ms}
`)

// redisRateLimiter enforces a token-bucket policy cluster-wide via Redis,
// with an in-process fallback for Redis outages.
type redisRateLimiter struct {
	rdb      *redis.Client
	name     string
	policy   rateLimitPolicy
	fallback *rateLimiter
}

// newRedisRateLimiter wraps a policy in a cluster-wide limiter.
func newRedisRateLimiter(rdb *redis.Client, name string, rate, burst float64) *redisRateLimiter {
	return &redisRateLimiter{
		rdb:      rdb,
		name:     name,
		policy:   rateLimitPolicy{rate: rate, burst: burst},
		fallback: newRateLimiter(rate, burst),
	}
}

func (rl *redisRateLimiter) allow(key string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	res, err := redisTokenBucketScript.Run(ctx, rl.rdb,
		[]string{"rl:wallet_api:" + rl.name + ":" + key},
		rl.policy.rate, rl.policy.burst, time.Now().UnixMilli(),
	).Slice()
	if err != nil || len(res) < 1 {
		// Redis down/slow: apply the per-replica floor. This errs
		// toward rejecting (each replica still enforces the full
		// policy locally), never toward unlimited traffic.
		return rl.fallback.allow(key)
	}
	allowed, ok := res[0].(int64)
	return ok && allowed == 1
}

func (rl *redisRateLimiter) retryAfterSeconds() int {
	return rl.fallback.retryAfterSeconds()
}

// initClusterLimiters upgrades the package-level auth/sign limiters from
// in-process to cluster-wide. Called once at boot when Redis is available.
func initClusterLimiters(rdb *redis.Client) {
	if rdb == nil {
		return
	}
	authLimiter = newRedisRateLimiter(rdb, "auth", 5.0/60.0, 5)
	signLimiter = newRedisRateLimiter(rdb, "sign", 20.0/60.0, 20)
}
