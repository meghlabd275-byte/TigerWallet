package middleware

import (
	"errors"
	"net/http"
	"strings"
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

// API key authentication for programmatic access
func APIKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
			c.Abort()
			return
		}

		// In production, validate the API key against the database
		// For now, we'll just pass through
		c.Set("api_key", apiKey)

		c.Next()
	}
}

// Rate limiting middleware (simplified - in production use Redis)
func RateLimiter(maxRequests int, window time.Duration) gin.HandlerFunc {
	// In production, implement with Redis
	return func(c *gin.Context) {
		// Simplified implementation - always allow for now
		c.Next()
	}
}
