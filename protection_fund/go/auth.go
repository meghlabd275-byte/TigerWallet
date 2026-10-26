// auth.go — bearer-token authentication and role checks for the Protection
// Fund API.
//
// The service manages a real fund and pays real claims, so every mutating route
// requires an authenticated operator. The signing key must be configured; the
// algorithm is pinned to HMAC so a token cannot be forged with "alg":"none".

package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var (
	// errAuthNotConfigured is returned when no signing key is set.
	errAuthNotConfigured = errors.New("authentication is not configured (PROTECTION_FUND_JWT_SECRET)")
	// errInvalidToken is returned for a token that fails validation.
	errInvalidToken = errors.New("invalid token")
)

// Claims is the authenticated principal carried by a fund token.
type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

// jwtSecret returns the configured signing key.
func jwtSecret() ([]byte, error) {
	secret := strings.TrimSpace(os.Getenv("PROTECTION_FUND_JWT_SECRET"))
	if secret == "" {
		return nil, errAuthNotConfigured
	}
	return []byte(secret), nil
}

// parseToken validates a bearer token and returns its claims.
func parseToken(raw string) (*Claims, error) {
	secret, err := jwtSecret()
	if err != nil {
		return nil, err
	}

	parsed, err := jwt.Parse(raw, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	})
	if err != nil || !parsed.Valid {
		return nil, errInvalidToken
	}

	mapClaims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errInvalidToken
	}

	userID, _ := mapClaims["user_id"].(string)
	if userID == "" {
		if sub, _ := mapClaims["sub"].(string); sub != "" {
			userID = sub
		}
	}
	if userID == "" {
		return nil, errInvalidToken
	}

	role, _ := mapClaims["role"].(string)
	return &Claims{UserID: userID, Role: strings.ToLower(role)}, nil
}

// RequireAuth rejects unauthenticated requests and binds the principal.
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := jwtSecret(); err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}

		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}

		claims, err := parseToken(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": errInvalidToken.Error()})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// RequireRole restricts a route to the listed roles.
func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(roles))
	for _, role := range roles {
		allowed[strings.ToLower(role)] = true
	}

	return func(c *gin.Context) {
		if !allowed[c.GetString("role")] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}
		c.Next()
	}
}
