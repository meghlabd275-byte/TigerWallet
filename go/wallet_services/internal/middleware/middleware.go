/**
 * HTTP Middleware
 */

package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/tigerwallet/wallet-services/internal/config"
	"github.com/tigerwallet/wallet-services/internal/cache"
	"github.com/sirupsen/logrus"
)

var logger = logrus.WithField("component", "middleware")

// RequestID adds a unique request ID to each request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("requestID", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// Timeout adds request timeout
func Timeout() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip for health checks
		if c.Request.URL.Path == "/health" || c.Request.URL.Path == "/ready" {
			c.Next()
			return
		}

		done := make(chan struct{})
		go func() {
			c.Next()
			close(done)
		}()

		select {
		case <-done:
			return
		case <-time.After(30 * time.Second):
			c.AbortWithStatusJSON(http.StatusRequestTimeout, gin.H{
				"error": "Request timeout",
				"code":  "TIMEOUT",
			})
		}
	}
}

// CORS handles Cross-Origin Resource Sharing
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Request-ID, X-Client-Version")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID, X-Total-Count")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// Compression adds gzip compression
func Compression() gin.HandlerFunc {
	// In production, use actual gzip middleware
	return func(c *gin.Context) {
		c.Next()
	}
}

// RateLimiter implements rate limiting
func RateLimiter() gin.HandlerFunc {
	// In production, implement proper rate limiting with Redis
	return func(c *gin.Context) {
		c.Next()
	}
}

// AuthRequired validates JWT tokens
func AuthRequired(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header required",
				"code":  "UNAUTHORIZED",
			})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
				"code":  "UNAUTHORIZED",
			})
			return
		}

		tokenString := parts[1]

		// Parse and validate token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
				"code":  "TOKEN_INVALID",
			})
			return
		}

		// Extract claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token claims",
				"code":  "TOKEN_INVALID",
			})
			return
		}

		// Set user info in context
		userID, ok := claims["sub"].(string)
		if !ok || userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid user ID in token",
				"code":  "TOKEN_INVALID",
			})
			return
		}

		c.Set("userID", userID)
		c.Set("claims", claims)
		c.Next()
	}
}

// ValidateAPIKey validates API keys for programmatic access
func ValidateAPIKey(db interface{}, cache *cache.RedisClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "API key required",
				"code":  "API_KEY_REQUIRED",
			})
			return
		}

		// Validate API key (implementation depends on database)
		// For now, skip validation
		c.Next()
	}
}

// SecurityHeaders adds security headers
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Next()
	}
}

// Logger logs all requests
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		requestID, _ := c.Get("requestID")

		logger.WithFields(logrus.Fields{
			"method":      method,
			"path":        path,
			"status":     statusCode,
			"latency":    latency,
			"request_id": requestID,
			"client_ip":   c.ClientIP(),
		}).Info("HTTP Request")
	}
}

// Recovery handles panics
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Errorf("Panic recovered: %v", err)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
					"code":  "INTERNAL_ERROR",
				})
			}
		}()
		c.Next()
	}
}

// ValidateRequest validates request body
func ValidateRequest(schema interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		// In production, implement JSON schema validation
		c.Next()
	}
}

// MonitorPerformance tracks performance metrics
func MonitorPerformance() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		latency := time.Since(start)
		
		// Log slow requests
		if latency > 1*time.Second {
			logger.Warnf("Slow request: %s %s took %v", 
				c.Request.Method, 
				c.Request.URL.Path, 
				latency,
			)
		}
	}
}

// IPWhitelist restricts access to specific IPs
func IPWhitelist(whitelist []string) gin.HandlerFunc {
	whitelistMap := make(map[string]bool)
	for _, ip := range whitelist {
		whitelistMap[ip] = true
	}

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		
		if len(whitelistMap) > 0 && !whitelistMap[clientIP] && clientIP != "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Access denied",
				"code":  "FORBIDDEN",
			})
			return
		}

		c.Next()
	}
}

// CSRF protection
func CSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "GET" {
			// Generate CSRF token for GET requests
			csrfToken := uuid.New().String()
			c.Set("csrf_token", csrfToken)
			c.Header("X-CSRF-Token", csrfToken)
		} else {
			// Validate CSRF token for other methods
			// In production, implement proper CSRF validation
		}
		c.Next()
	}
}

// InputSanitization sanitizes user input
func InputSanitization() gin.HandlerFunc {
	return func(c *gin.Context) {
		// In production, sanitize all inputs to prevent XSS, SQL injection, etc.
		c.Next()
	}
}

// FeatureFlags enables feature flags
func FeatureFlags(flags map[string]bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("feature_flags", flags)
		c.Next()
	}
}

// RequireRole checks if user has required role
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// In production, check user role from claims
		userRole, exists := c.Get("user_role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Role check failed",
				"code":  "FORBIDDEN",
			})
			return
		}

		for _, role := range allowedRoles {
			if userRole == role {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "Insufficient permissions",
			"code":  "FORBIDDEN",
		})
	}
}

// RateLimitByUser implements per-user rate limiting
func RateLimitByUser(redisClient *cache.RedisClient, requests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("userID")
		if !exists {
			c.Next()
			return
		}

		key := "ratelimit:user:" + userID.(string)
		
		allowed, err := redisClient.CheckRateLimit(c.Request.Context(), key, requests, window)
		if err != nil {
			logger.Warnf("Rate limit check failed: %v", err)
			c.Next()
			return
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
				"code":  "RATE_LIMIT_EXCEEDED",
				"retry_after": window.Seconds(),
			})
			return
		}

		c.Next()
	}
}

// ValidateChain ensures valid blockchain is specified
func ValidateChain() gin.HandlerFunc {
	validChains := map[string]bool{
		"bitcoin": true,
		"ethereum": true,
		"polygon": true,
		"bsc": true,
		"avalanche": true,
		"arbitrum": true,
		"optimism": true,
		"solana": true,
		"cosmos": true,
		"tron": true,
		"algorand": true,
		"aptos": true,
		"sui": true,
		"ton": true,
		"polkadot": true,
		"kusama": true,
		"cardano": true,
	}

	return func(c *gin.Context) {
		chain := c.Param("chain")
		if chain != "" && !validChains[chain] {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "Invalid blockchain",
				"code":  "INVALID_CHAIN",
			})
			return
		}

		c.Next()
	}
}

// Config returns the config
func Config(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("config", cfg)
		c.Next()
	}
}
