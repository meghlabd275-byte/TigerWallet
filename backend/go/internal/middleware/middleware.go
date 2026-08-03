package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/tigerwallet/backend/internal/config"
)

type RateLimiter struct {
	requests map[string]*clientRequests
	mu       sync.Mutex
	limit    int
	window   time.Duration
}

type clientRequests struct {
	count     int
	resetTime time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string]*clientRequests),
		limit:    limit,
		window:   window,
	}

	// Cleanup old entries
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			rl.cleanup()
		}
	}()

	return rl
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for key, client := range rl.requests {
		if client.resetTime.Before(now) {
			delete(rl.requests, key)
		}
	}
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	client, exists := rl.requests[key]

	if !exists || client.resetTime.Before(now) {
		rl.requests[key] = &clientRequests{
			count:     1,
			resetTime: now.Add(rl.window),
		}
		return true
	}

	if client.count >= rl.limit {
		return false
	}

	client.count++
	return true
}

func (rl *RateLimiter) Remaining(key string) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	client, exists := rl.requests[key]
	if !exists {
		return rl.limit
	}

	return rl.limit - client.count
}

var rateLimiter = NewRateLimiter(100, 15*time.Minute)

func RateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()

		if !rateLimiter.Allow(key) {
			remaining := rateLimiter.Remaining(key)
			c.Header("X-RateLimit-Limit", "100")
			c.Header("X-RateLimit-Remaining", "0")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests",
			})
			c.Abort()
			return
		}

		c.Header("X-RateLimit-Limit", "100")
		c.Header("X-RateLimit-Remaining", string(rune(rateLimiter.Remaining(key))))
		c.Next()
	}
}

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		c.Next()
	}
}

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// In production, validate against allowed origins
		allowedOrigins := []string{
			"https://tigerwallet.com",
			"http://localhost:3001",
		}

		isAllowed := false
		for _, allowed := range allowedOrigins {
			if origin == allowed || strings.HasPrefix(origin, "http://localhost") {
				isAllowed = true
				break
			}
		}

		if isAllowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func Authenticate(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for API key first
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" {
			// Validate API key
			// In production: check against database
			c.Set("userID", "api-user")
			c.Set("isAPIKey", true)
			c.Next()
			return
		}

		// Check for JWT token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "No token provided"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(cfg.JWT.Secret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		userIDStr, ok := claims["userId"].(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
			c.Abort()
			return
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID format"})
			c.Abort()
			return
		}

		c.Set("userID", userID)
		c.Set("isAPIKey", false)
		c.Next()
	}
}

func RequireKYC(level string) gin.HandlerFunc {
	levels := map[string]int{
		"BASIC":        1,
		"INTERMEDIATE": 2,
		"FULL":         3,
	}

	return func(c *gin.Context) {
		userID := c.MustGet("userID").(uuid.UUID)

		// In production: fetch user's KYC level from database
		// For now, allow all authenticated users
		_ = userID

		c.Next()
	}
}

func SanitizeInput() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Basic input sanitization can be added here
		// In production: use a library like bluemonday
		c.Next()
	}
}

func AuditLog(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Store original path and method
		path := c.Request.URL.Path
		method := c.Request.Method

		// Get user ID if authenticated
		var userID *uuid.UUID
		if uid, exists := c.Get("userID"); exists {
			if u, ok := uid.(uuid.UUID); ok {
				userID = &u
			}
		}

		// Log after request
		c.Next()

		// Log to database (async)
		go func() {
			// In production: insert into audit_logs table
			_ = userID
			_ = action
			_ = path
			_ = method
		}()
	}
}
