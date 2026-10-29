package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/tigerwallet/billing/internal/config"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

type Claims struct {
	UserID   uuid.UUID `json:"user_id"`
	TenantID uuid.UUID `json:"tenant_id"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	jwt.RegisteredClaims
}

func JWTAuth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			c.Abort()
			return
		}

		claims, err := ValidateToken(tokenString, cfg.JWT.Secret)
		if err != nil {
			status := http.StatusUnauthorized
			message := "invalid token"
			if errors.Is(err, ErrExpiredToken) {
				message = "token has expired"
			}
			c.JSON(status, gin.H{"error": message})
			c.Abort()
			return
		}

		// Set user info in context
		c.Set("user_id", claims.UserID)
		c.Set("tenant_id", claims.TenantID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)

		c.Next()
	}
}

func GenerateToken(userID, tenantID uuid.UUID, email, role string, cfg *config.Config) (string, error) {
	claims := Claims{
		UserID:   userID,
		TenantID: tenantID,
		Email:    email,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(cfg.JWT.Expiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "tigerwallet",
			Subject:   userID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWT.Secret))
}

func GenerateRefreshToken(userID, tenantID uuid.UUID, email, role string, cfg *config.Config) (string, error) {
	claims := Claims{
		UserID:   userID,
		TenantID: tenantID,
		Email:    email,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(cfg.JWT.RefreshDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "tigerwallet",
			Subject:   userID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWT.Secret))
}

func ValidateToken(tokenString, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

func RefreshToken(tokenString string, cfg *config.Config) (string, error) {
	claims, err := ValidateToken(tokenString, cfg.JWT.Secret)
	if err != nil {
		return "", err
	}

	return GenerateToken(claims.UserID, claims.TenantID, claims.Email, claims.Role, cfg)
}

// Role-based access control
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "role not found"})
			c.Abort()
			return
		}

		roleStr := role.(string)
		for _, r := range roles {
			if roleStr == r {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		c.Abort()
	}
}

// Tenant isolation middleware
func TenantIsolation() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get tenant from context (set by JWT auth)
		tenantID, exists := c.Get("tenant_id")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "tenant not found"})
			c.Abort()
			return
		}

		// Check if request has tenant header and validate it matches the token
		headerTenantID := c.GetHeader("X-Tenant-ID")
		if headerTenantID != "" {
			headerUUID, err := uuid.Parse(headerTenantID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant ID"})
				c.Abort()
				return
			}
			if headerUUID != tenantID.(uuid.UUID) {
				c.JSON(http.StatusForbidden, gin.H{"error": "tenant mismatch"})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// APIKeyAuth authenticates programmatic access.
//
// The presented key is hashed with SHA-256 and compared against the configured
// key hashes in constant time. An unset or empty configuration refuses every
// request rather than accepting any key.
func APIKeyAuth() gin.HandlerFunc {
	hashes := configuredAPIKeyHashes()

	return func(c *gin.Context) {
		apiKey := strings.TrimSpace(c.GetHeader("X-API-Key"))
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
			c.Abort()
			return
		}
		if len(hashes) == 0 {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "API key authentication is not configured"})
			c.Abort()
			return
		}

		presented := sha256.Sum256([]byte(apiKey))
		authorized := false
		for _, expected := range hashes {
			if hmac.Equal(presented[:], expected) {
				authorized = true
				break
			}
		}
		if !authorized {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
			c.Abort()
			return
		}

		c.Set("api_key", apiKey)
		c.Next()
	}
}

// configuredAPIKeyHashes returns the SHA-256 hashes of the accepted API keys.
//
// Keys are configured as BILLING_API_KEY_HASHES, a comma-separated list of hex
// SHA-256 digests, so the raw keys are not held in the process environment.
// BILLING_API_KEYS is also accepted for deployments that can only supply the
// keys themselves; those values are hashed at startup.
func configuredAPIKeyHashes() [][]byte {
	var out [][]byte

	for _, raw := range strings.Split(os.Getenv("BILLING_API_KEY_HASHES"), ",") {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		decoded, err := hex.DecodeString(value)
		if err != nil || len(decoded) != sha256.Size {
			log.Printf("billing: ignoring malformed BILLING_API_KEY_HASHES entry (must be a hex SHA-256 digest)")
			continue
		}
		out = append(out, decoded)
	}

	for _, raw := range strings.Split(os.Getenv("BILLING_API_KEYS"), ",") {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		sum := sha256.Sum256([]byte(value))
		out = append(out, sum[:])
	}

	return out
}

// RateLimiter bounds how many requests a caller may make in a window.
//
// The window is a fixed interval: the counter resets when it elapses. Entries
// are keyed by client address and expired entries are swept so the map cannot
// grow without bound.
func RateLimiter(maxRequests int, window time.Duration) gin.HandlerFunc {
	if maxRequests <= 0 || window <= 0 {
		// A misconfigured limiter must not silently disable itself; refusing
		// traffic is the safe failure mode for a security control.
		return func(c *gin.Context) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "rate limiter is not configured"})
			c.Abort()
		}
	}

	type bucket struct {
		count     int
		expiresAt time.Time
	}

	var (
		mu      sync.Mutex
		buckets = map[string]*bucket{}
	)

	return func(c *gin.Context) {
		key := c.ClientIP()
		now := time.Now()

		mu.Lock()
		for k, b := range buckets {
			if now.After(b.expiresAt) {
				delete(buckets, k)
			}
		}
		b, ok := buckets[key]
		if !ok {
			b = &bucket{expiresAt: now.Add(window)}
			buckets[key] = b
		}
		b.count++
		exceeded := b.count > maxRequests
		remaining := maxRequests - b.count
		if remaining < 0 {
			remaining = 0
		}
		mu.Unlock()

		c.Header("X-RateLimit-Limit", strconv.Itoa(maxRequests))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))

		if exceeded {
			c.Header("Retry-After", strconv.Itoa(int(window.Seconds())))
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			c.Abort()
			return
		}

		c.Next()
	}
}
