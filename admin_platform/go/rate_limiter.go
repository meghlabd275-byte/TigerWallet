package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// ============================================================================
// RATE LIMITER TYPES
// ============================================================================

type RateLimiter struct {
	mu           sync.RWMutex
	redis        *redis.Client
	limits      map[string]*RateLimitConfig
	ipCounts     map[string]*WindowCounter
	windowSize   time.Duration
	cleanupInterval time.Duration
}

type RateLimitConfig struct {
	Requests int           `json:"requests"`
	Window   time.Duration `json:"window"`
	Burst   int           `json:"burst"`
}

type WindowCounter struct {
	Count     int
	ExpiresAt time.Time
}

type RateLimitResponse struct {
	Allowed    bool   `json:"allowed"`
	Remaining int    `json:"remaining"`
	ResetAt   int64  `json:"resetAt"`
	Limit     int    `json:"limit"`
}

// ============================================================================
// RATE LIMITER IMPLEMENTATION
// ============================================================================

func NewRateLimiter(redisClient *redis.Client) *RateLimiter {
	rl := &RateLimiter{
		redis:      redisClient,
		limits:     make(map[string]*RateLimitConfig),
		ipCounts:   make(map[string]*WindowCounter),
		windowSize: time.Minute,
	}

	// Default limits
	rl.limits["default"] = &RateLimitConfig{
		Requests: 100,
		Window:   time.Minute,
		Burst:    10,
	}

	rl.limits["auth"] = &RateLimitConfig{
		Requests: 5,
		Window:   time.Minute,
		Burst:    3,
	}

	rl.limits["api"] = &RateLimitConfig{
		Requests: 1000,
		Window:   time.Minute,
		Burst:    50,
	}

	rl.limits["admin"] = &RateLimitConfig{
		Requests: 500,
		Window:   time.Minute,
		Burst:    25,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, counter := range rl.ipCounts {
			if now.After(counter.ExpiresAt) {
				delete(rl.ipCounts, key)
			}
		}
		rl.mu.Unlock()
	}
}

// ============================================================================
// RATE LIMITING MIDDLEWARE
// ============================================================================

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip rate limiting for health checks
		if c.Request.URL.Path == "/health" {
			c.Next()
			return
		}

		// Get client identifier
		identifier := rl.getIdentifier(c)

		// Get rate limit config
		config := rl.getConfig(c)

		// Check rate limit
		allowed, remaining, resetAt := rl.checkRateLimit(identifier, config)

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", config.Requests))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetAt))

		if !allowed {
			c.JSON(http.StatusTooManyRequests, RateLimitResponse{
				Allowed:   false,
				Remaining: 0,
				ResetAt:   resetAt,
				Limit:     config.Requests,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func (rl *RateLimiter) getIdentifier(c *gin.Context) string {
	// Try to get from authenticated user first
	if userID, exists := c.Get("user_id"); exists {
		return fmt.Sprintf("user:%s", userID)
	}

	// Fall back to IP address
	ip := c.ClientIP()
	if forwarded := c.GetHeader("X-Forwarded-For"); forwarded != "" {
		ip = forwarded
	}

	// Add endpoint to differentiate
	return fmt.Sprintf("ip:%s:%s", ip, c.Request.URL.Path)
}

func (rl *RateLimiter) getConfig(c *gin.Context) *RateLimitConfig {
	// Determine which limit to apply based on endpoint
	path := c.Request.URL.Path

	switch {
	case contains(path, "/auth/login"), contains(path, "/auth/register"):
		return rl.limits["auth"]
	case contains(path, "/admin"), contains(path, "/api/v1/admins"):
		return rl.limits["admin"]
	case contains(path, "/api/"):
		return rl.limits["api"]
	default:
		return rl.limits["default"]
	}
}

func (rl *RateLimiter) checkRateLimit(identifier string, config *RateLimitConfig) (bool, int, int64) {
	// Try Redis first
	if rl.redis != nil {
		return rl.checkRateLimitRedis(identifier, config)
	}

	// Fall back to in-memory
	return rl.checkRateLimitMemory(identifier, config)
}

func (rl *RateLimiter) checkRateLimitRedis(identifier string, config *RateLimitConfig) (bool, int, int64) {
	ctx := context.Background()
	key := fmt.Sprintf("ratelimit:%s", identifier)

	// Increment counter
	count, err := rl.redis.Incr(ctx, key).Result()
	if err != nil {
		return true, config.Requests, time.Now().Add(config.Window).Unix()
	}

	// Set expiry on first request
	if count == 1 {
		rl.redis.Expire(ctx, key, config.Window)
	}

	resetAt := time.Now().Add(config.Window).Unix()
	allowed := count <= int64(config.Requests)
	remaining := config.Requests - int(count)

	if remaining < 0 {
		remaining = 0
	}

	return allowed, remaining, resetAt
}

func (rl *RateLimiter) checkRateLimitMemory(identifier string, config *RateLimitConfig) (bool, int, int64) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	counter, exists := rl.ipCounts[identifier]
	if !exists || now.After(counter.ExpiresAt) {
		// New window
		counter = &WindowCounter{
			Count:     1,
			ExpiresAt: now.Add(config.Window),
		}
		rl.ipCounts[identifier] = counter
		return true, config.Requests - 1, counter.ExpiresAt.Unix()
	}

	// Existing window
	counter.Count++
	allowed := counter.Count <= config.Requests
	remaining := config.Requests - counter.Count

	if remaining < 0 {
		remaining = 0
	}

	return allowed, remaining, counter.ExpiresAt.Unix()
}

// ============================================================================
// SLIDING WINDOW RATE LIMITER (More accurate)
// ============================================================================

type SlidingWindowLimiter struct {
	redis *redis.Client
}

func NewSlidingWindowLimiter(redisClient *redis.Client) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{redis: redisClient}
}

func (swl *SlidingWindowLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/health" {
			c.Next()
			return
		}

		identifier := c.ClientIP()
		key := fmt.Sprintf("sliding_ratelimit:%s", identifier)

		now := time.Now()
		windowStart := now.Unix() - 60 // 1 minute window

		ctx := context.Background()

		// Remove old entries
		swl.redis.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))

		// Count current requests
		count, _ := swl.redis.ZCard(ctx, key).Result()

		limit := 100 // requests per minute

		if int(count) >= limit {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
				"limit": limit,
			})
			c.Abort()
			return
		}

		// Add current request
		swl.redis.ZAdd(ctx, key, &redis.Z{
			Score:  float64(now.Unix()),
			Member: now.UnixNano(),
		})
		swl.redis.Expire(ctx, key, 2*time.Minute)

		// Set headers
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", limit-count-1))

		c.Next()
	}
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
