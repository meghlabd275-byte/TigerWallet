// Package middleware provides JWT auth + the fail-closed license gate for the
// standalone WL-UserWallet backend. The gate mirrors the C++ WlGate semantics
// (wait-free atomic liveness + flag snapshot) but in pure Go so the standalone
// backend has no cgo dependency. The Rust SDK / control plane heartbeat pushes
// the liveness + flags into this gate.
package middleware

import (
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// gate is the in-process license gate (mirrors wl_control_plane/cpp WlGate).
type gate struct {
	alive atomic.Bool
	mu    sync.RWMutex
	reason string
	flags map[string]bool // product\x1ffetcher -> enabled
}

var licenseGate = &gate{flags: map[string]bool{}}

// SetAlive sets the liveness flag (called by the heartbeat loop after validate).
func SetAlive(alive bool, reason string) {
	licenseGate.alive.Store(alive)
	licenseGate.mu.Lock()
	licenseGate.reason = reason
	if alive {
		licenseGate.reason = ""
	}
	licenseGate.mu.Unlock()
}

// SetFlags pushes a flag snapshot.
type Flag struct {
	Product string `json:"product"`
	Fetcher string `json:"fetcher"`
	Enabled bool   `json:"enabled"`
}

func SetFlags(flags []Flag) {
	licenseGate.mu.Lock()
	defer licenseGate.mu.Unlock()
	licenseGate.flags = map[string]bool{}
	for _, f := range flags {
		licenseGate.flags[f.Product+"\x1f"+f.Fetcher] = f.Enabled
	}
}

// IsAlive returns the liveness flag.
func IsAlive() bool { return licenseGate.alive.Load() }

// Reason returns the fail-closed reason.
func Reason() string {
	licenseGate.mu.RLock()
	defer licenseGate.mu.RUnlock()
	return licenseGate.reason
}

// FetcherEnabled checks the per-fetcher gate.
func FetcherEnabled(product, fetcher string) bool {
	if !IsAlive() {
		return false
	}
	licenseGate.mu.RLock()
	defer licenseGate.mu.RUnlock()
	if en, ok := licenseGate.flags[product+"\x1f*"]; ok && !en {
		return false
	}
	if en, ok := licenseGate.flags[product+"\x1f"+fetcher]; ok && !en {
		return false
	}
	return true
}

// Claims for the WL-UserWallet JWT.
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	jwt.RegisteredClaims
}

func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if h == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}
		parts := strings.Split(h, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid header"})
			return
		}
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(parts[1], claims, func(t *jwt.Token) (any, error) {
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Next()
	}
}

// UserID extracts the caller's user id from context.
func UserID(c *gin.Context) uuid.UUID {
	if v, ok := c.Get("user_id"); ok {
		return v.(uuid.UUID)
	}
	return uuid.Nil
}

// Gate is the license-gate middleware. It fail-closeds (503) when the product
// is not alive or a fetcher is disabled by SuperAdmin. Every protected route
// is wrapped in this.
func Gate(product string, fetcherForPath func(string) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !IsAlive() {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error":  "product is not authorized to serve (license suspended/revoked or heartbeat stale)",
				"reason": Reason(),
			})
			return
		}
		fetcher := "*"
		if fetcherForPath != nil {
			if f := fetcherForPath(c.Request.URL.Path); f != "" {
				fetcher = f
			}
		}
		if !FetcherEnabled(product, fetcher) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "fetcher disabled by SuperAdmin",
				"product": product,
				"fetcher": fetcher,
			})
			return
		}
		c.Next()
	}
}

// IssueJWT mints a JWT for a user.
func IssueJWT(secret string, userID uuid.UUID, email string, expiry time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}

// SimpleFetcher derives the fetcher name from the last path segment.
func SimpleFetcher(path string) string {
	path = strings.Trim(path, "/")
	if path == "" {
		return "*"
	}
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}
