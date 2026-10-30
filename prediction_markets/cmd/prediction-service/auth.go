package main

// auth.go — bearer-token authentication.
//
// The trading endpoints act on a user's positions, so the user is taken from a
// verified token and never from the request body or the path. A caller that
// names somebody else's identifier is refused rather than obeyed.

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Claims is the subset of the token payload this service relies on.
type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// ErrNoSigningKey is returned when no signing key has been configured. It is
// fatal at startup, so a deployment cannot accidentally run unauthenticated.
var ErrNoSigningKey = errors.New("JWT_SECRET must be set; refusing to serve trading without authentication")

// verifyToken validates a bearer token and returns the authenticated user.
func verifyToken(secret []byte, header string) (string, error) {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", errors.New("a bearer token is required")
	}

	raw := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if raw == "" {
		return "", errors.New("the bearer token is empty")
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (interface{}, error) {
		// Only HMAC signatures are accepted. Without this check a token signed
		// with "none", or with an RSA key presented as an HMAC secret, would
		// verify.
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	}, jwt.WithValidMethods([]string{"HS256", "HS384", "HS512"}))
	if err != nil {
		return "", err
	}
	if !token.Valid {
		return "", errors.New("the token is not valid")
	}
	if claims.UserID == "" {
		return "", errors.New("the token carries no user_id")
	}

	return claims.UserID, nil
}

// requireUser authenticates the request and stores the user id on the context.
func requireUser(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := verifyToken(secret, c.GetHeader("Authorization"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		c.Set(contextUserKey, userID)
		c.Next()
	}
}

// contextUserKey is the context key holding the authenticated user id.
const contextUserKey = "authenticatedUserID"

// authenticatedUser returns the user id placed on the context by requireUser.
func authenticatedUser(c *gin.Context) (string, bool) {
	value, exists := c.Get(contextUserKey)
	if !exists {
		return "", false
	}
	userID, ok := value.(string)
	return userID, ok
}

// resolveUser returns the authenticated user, and reports a mismatch when the
// caller asked for a different account.
//
// The identifier in the path or query is treated as an assertion to be checked,
// not as the source of truth: honouring it would let any caller read any
// account's positions.
func resolveUser(c *gin.Context, requested string) (string, bool) {
	authenticated, ok := authenticatedUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return "", false
	}

	if requested != "" && requested != authenticated {
		c.JSON(http.StatusForbidden, gin.H{"error": "this account is not yours to read"})
		return "", false
	}

	return authenticated, true
}
