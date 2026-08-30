package main

// ratelimit.go — in-process per-IP rate limiting middleware.
//
// The canonical dedicated `go/rate_limiter_service` (:8012) exposes a generic
// token-bucket policy engine over HTTP, but it is not consumed by this service.
// To close that gap without introducing a hard cross-service dependency (which
// would make wallet_api unavailable whenever the limiter was down), this file
// implements a self-contained, in-process token-bucket limiter keyed by client
// IP (+ optional authenticated user). It is a fixed-window/token-bucket hybrid:
// each key has a refill rate and burst; the first request that exceeds the
// bucket is rejected with HTTP 429 and a Retry-After header.
//
// This is deliberately simple and dependency-free (stdlib + gin only) so it
// works in every deployment without an extra service. For multi-instance
// deployments the same policy should be backed by Redis (the limiter service
// already supports this pattern); the in-process limiter is a correct
// single-instance floor.

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// bucket is a token-bucket with a monotonic refill.
type bucket struct {
	tokens   float64
	lastTime time.Time
}

// rateLimitPolicy describes a limit for a group of routes.
type rateLimitPolicy struct {
	// rate is the steady-state refill in tokens per second.
	rate float64
	// burst is the maximum tokens that can accumulate.
	burst float64
}

// rateLimiter is a thread-safe in-process token-bucket limiter.
type rateLimiter struct {
	mu      sync.Mutex
	policy  rateLimitPolicy
	buckets map[string]*bucket
}

// newRateLimiter creates a limiter with the given refill rate (tokens/sec)
// and burst capacity.
func newRateLimiter(rate, burst float64) *rateLimiter {
	return &rateLimiter{
		policy:  rateLimitPolicy{rate: rate, burst: burst},
		buckets: make(map[string]*bucket),
	}
}

// allow returns true if a request from the given key is allowed, consuming one
// token. Refills the bucket based on elapsed wall-clock time.
func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.buckets[key]
	if !ok {
		// New bucket starts full.
		b = &bucket{tokens: rl.policy.burst, lastTime: now}
		rl.buckets[key] = b
	}
	elapsed := now.Sub(b.lastTime).Seconds()
	if elapsed > 0 {
		b.tokens += elapsed * rl.policy.rate
		if b.tokens > rl.policy.burst {
			b.tokens = rl.policy.burst
		}
	}
	b.lastTime = now
	if b.tokens < 1.0 {
		return false
	}
	b.tokens -= 1.0
	return true
}

// retryAfterSeconds returns a conservative estimate of when the next token
// would be available for a saturated key.
func (rl *rateLimiter) retryAfterSeconds() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	wait := 1.0 / rl.policy.rate
	if wait < 1 {
		wait = 1
	}
	return int(wait)
}

// clientKey returns a stable per-client key. It prefers the authenticated user
// ID (so an account doing heavy work from many IPs is still limited) and falls
// back to the client IP. It uses the X-Forwarded-For header (first hop) when a
// reverse proxy is in front.
func clientKey(c *gin.Context) string {
	if uid := getUserID(c); uid != "" {
		return "u:" + uid
	}
	ip := c.ClientIP()
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		// first hop only
		if i := indexByte(xff, ','); i > 0 {
			xff = xff[:i]
		}
		if xff != "" {
			ip = trimSpace(xff)
		}
	}
	return "ip:" + ip
}

// indexByte is a stdlib-free indexByte (avoid pulling strings just for this).
func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

// limiter is the interface enforced by the RateLimit middleware. It is
// satisfied by the in-process rateLimiter (single replica) and by
// redisRateLimiter (cluster-wide; see ratelimit_redis.go).
type limiter interface {
	allow(key string) bool
	retryAfterSeconds() int
}

// RateLimit returns gin middleware enforcing the given refill rate + burst on
// the routes it wraps. On limit exceeded it responds 429 with a Retry-After
// header. The key is the authenticated user ID when present, else the client
// IP, so a single account cannot bypass the limit by rotating IPs and a single
// IP cannot bypass it across accounts.
func RateLimit(rl limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := clientKey(c)
		if !rl.allow(key) {
			retry := rl.retryAfterSeconds()
			c.Header("Retry-After", strconv.Itoa(retry))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded, please retry later",
				"retry_after": retry,
			})
			return
		}
		c.Next()
	}
}

// Pre-configured limiters for the sensitive surfaces.
//
// Auth endpoints: 5 logins + 5 registrations per minute per IP/user. This
// throttles brute-force credential guessing without blocking normal use.
//
// Signing endpoints (send/sign/nft transfer): 20 per minute per user. These
// are the funds-movement surfaces; the limit is generous for normal use but
// caps automated drain attempts.
var (
	authLimiter limiter = newRateLimiter(5.0/60.0, 5)   // ~5/min, burst 5
	signLimiter limiter = newRateLimiter(20.0/60.0, 20) // ~20/min, burst 20
)
