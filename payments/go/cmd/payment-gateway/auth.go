// auth.go — bearer-token authentication for the payment gateway.
//
// The gateway charges cards and reads card metadata, so every route requires an
// authenticated caller and every user-scoped lookup is checked against the
// authenticated principal. Previously the routes were open and handlers trusted
// a user_id taken from the request body or the URL path.

package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var (
	// errAuthNotConfigured is returned when no signing key is set.
	errAuthNotConfigured = errors.New("authentication is not configured (PAYMENT_GATEWAY_JWT_SECRET)")
	// errInvalidToken is returned for a token that fails validation.
	errInvalidToken = errors.New("invalid token")
	// errForbidden is returned when a caller targets another user's data.
	errForbidden = errors.New("forbidden")
)

// contextUserID is the key holding the authenticated user id.
const contextUserID = "auth_user_id"

// jwtSecret returns the configured signing key.
func jwtSecret() ([]byte, error) {
	secret := strings.TrimSpace(os.Getenv("PAYMENT_GATEWAY_JWT_SECRET"))
	if secret == "" {
		return nil, errAuthNotConfigured
	}
	return []byte(secret), nil
}

// parseToken validates a bearer token and returns the authenticated user id.
func parseToken(raw string) (uint, error) {
	secret, err := jwtSecret()
	if err != nil {
		return 0, err
	}

	parsed, err := jwt.Parse(raw, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	})
	if err != nil || !parsed.Valid {
		return 0, errInvalidToken
	}

	mapClaims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return 0, errInvalidToken
	}

	// The subject may arrive as a number or a string depending on the issuer.
	switch subject := mapClaims["user_id"].(type) {
	case float64:
		if subject <= 0 {
			return 0, errInvalidToken
		}
		return uint(subject), nil
	case string:
		parsed, err := strconv.ParseUint(subject, 10, 32)
		if err != nil || parsed == 0 {
			return 0, errInvalidToken
		}
		return uint(parsed), nil
	default:
		return 0, errInvalidToken
	}
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

		userID, err := parseToken(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": errInvalidToken.Error()})
			return
		}

		c.Set(contextUserID, userID)
		c.Next()
	}
}

// currentUserID returns the authenticated user id.
func currentUserID(c *gin.Context) uint {
	if value, ok := c.Get(contextUserID); ok {
		if userID, ok := value.(uint); ok {
			return userID
		}
	}
	return 0
}

// requireSelf aborts unless the path parameter names the authenticated user.
// Without this check any caller could read another user's cards and payments.
func requireSelf(c *gin.Context) {
	requested, err := strconv.ParseUint(c.Param("user_id"), 10, 32)
	if err != nil || uint(requested) != currentUserID(c) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": errForbidden.Error()})
		return
	}
	c.Next()
}
