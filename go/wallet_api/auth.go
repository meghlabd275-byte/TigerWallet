package main

// auth.go — JWT authentication middleware. Issues HS256-signed tokens after
// credential verification; middleware validates them on protected routes.

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a password with bcrypt (cost 12).
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword checks a bcrypt hash in constant time.
func VerifyPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// constantTimeEq is kept for any non-bcrypt constant-time comparisons.
func constantTimeEq(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// emailLocalPart returns the local part of an email address (before '@'),
// used to derive a default username when the client omits one.
func emailLocalPart(email string) string {
	if i := strings.IndexByte(email, '@'); i > 0 {
		return email[:i]
	}
	return ""
}

// IssueJWT creates a signed JWT for a user ID + role. The role claim is used
// by RequireAdmin to gate admin/wl-admin/master-wallet-admin endpoints.
func IssueJWT(secret string, userID string, role string) (string, error) {
	if role == "" {
		role = "user"
	}
	claims := jwt.MapClaims{
		"sub":  userID,
		"role": role,
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
		"iat":  time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseJWT validates and parses a JWT token, returning (userID, role).
func ParseJWT(secret, tokenStr string) (string, string, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return "", "", errors.New("invalid token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", errors.New("invalid claims")
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return "", "", errors.New("missing subject")
	}
	role, _ := claims["role"].(string)
	if role == "" {
		role = "user"
	}
	return sub, role, nil
}

// AuthMiddleware validates the JWT bearer token and sets userID + role in context.
func AuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			return
		}
		userID, role, err := ParseJWT(secret, parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set("userID", userID)
		c.Set("role", role)
		c.Next()
	}
}

// RequireAdmin is middleware that rejects requests unless the authenticated
// user holds an admin-level role (admin, wl_admin, or master_wallet_admin).
// Must be used after AuthMiddleware.
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := getUserRole(c)
		if role != "admin" && role != "wl_admin" && role != "master_wallet_admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin privileges required"})
			return
		}
		c.Next()
	}
}

// getUserID extracts the authenticated user ID from the gin context.
func getUserID(c *gin.Context) string {
	v, _ := c.Get("userID")
	s, _ := v.(string)
	return s
}

// getUserRole extracts the authenticated user's role from the gin context.
func getUserRole(c *gin.Context) string {
	v, _ := c.Get("role")
	s, _ := v.(string)
	return s
}
