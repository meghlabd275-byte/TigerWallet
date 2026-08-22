// Package middleware provides JWT auth + the fail-closed license gate for the
// standalone WL-UserWallet backend. The gate mirrors the C++ WlGate semantics
// (wait-free atomic liveness + flag snapshot) but in pure Go so the standalone
// backend has no cgo dependency. The Rust SDK / control plane heartbeat pushes
// the liveness + flags into this gate.
package middleware

import (
	"context"
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

// Claims for the WL-UserWallet JWT. Scopes carries the canonical scoped-admin
// taxonomy (wl_client, wallet_admin, ...) issued at login and enforced by
// HasScope on admin routes — mirrors wl_shared/go/wlgate.Claims.
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Scopes []string  `json:"scopes"`
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
		c.Set("scopes", claims.Scopes)
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

// HasScope reports whether the caller holds the given scope. wl_client (the WL
// owner) is always honored — full tenancy control — mirroring wlgate.HasScope.
func HasScope(c *gin.Context, scope string) bool {
	if v, ok := c.Get("scopes"); ok {
		scopes, _ := v.([]string)
		for _, s := range scopes {
			if s == scope || s == "wl_client" {
				return true
			}
		}
	}
	return false
}

// activeUserChecker is set by main.go (wired to the store) so the middleware
// package does NOT import the store package (avoids a circular dep). When a
// wallet_admin suspends a user, the user's existing stateless JWT still
// validates, but RequireActiveUser re-checks is_active from PostgreSQL on
// every fund-moving request — so the suspended user is immediately locked out.
var (
	activeUserChecker func(ctx context.Context, id uuid.UUID) (bool, error)
	activeUserOnce    sync.Once
)

// SetActiveUserChecker wires the store-backed is_active lookup. Called once
// from main.go at startup.
func SetActiveUserChecker(fn func(ctx context.Context, id uuid.UUID) (bool, error)) {
	activeUserOnce.Do(func() { activeUserChecker = fn })
}

// RequireActiveUser is middleware that 403-rejects a suspended (is_active=false)
// user. It runs AFTER JWTAuth (which sets user_id) on every fund-moving route
// (send/sign/swap/staking/non_evm). Admin-oversight routes (wallet_admin) are
// exempt — a suspended user with wallet_admin scope can still be inspected, but
// cannot move their own funds. Fail-closed: if the checker is unset or the DB
// lookup errors, the request is 403-rejected (never silently allowed).
func RequireActiveUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		// wl_client (WL owner) + wallet_admin are governance callers, not the
		// wallet's user — they are not subject to user-suspension gating here
		// (their own access is governed by the scope check in each admin handler).
		if HasScope(c, "wl_client") || HasScope(c, "wallet_admin") {
			c.Next()
			return
		}
		uid := UserID(c)
		if uid == uuid.Nil || activeUserChecker == nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "user not authenticated or active-checker unavailable"})
			return
		}
		active, err := activeUserChecker(c.Request.Context(), uid)
		if err != nil || !active {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "account suspended"})
			return
		}
		c.Next()
	}
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

// CategoryFetcher derives the fetcher key from the first functional path
// segment after /api/v1. This gives SuperAdmin per-feature granularity: the
// flag key becomes "user_wallet\x1fswap" or "user_wallet\x1fstaking", so
// SuperAdmin can disable only swap while leaving staking/send running.
// Routes map to these functional categories:
//   /wallets, /wallets/:id/* -> wallets   (core wallet mgmt + balance)
//   /send, /sign              -> send      (fund movement — EIP-191 + tx)
//   /transactions             -> transactions (history)
//   /balance,/tokens,/nfts,/gas,/price,/chains -> market  (read-only market data)
//   /swap/*                   -> swap      (DEX swap)
//   /staking/*                -> staking   (stake/unstake/claim)
//   /non_evm/*                -> non_evm   (Solana/Bitcoin/Cosmos signing)
//   /address-book/*           -> address_book
//   /devices/*                -> devices
//   /keystore/*               -> keystore
//   /admin/*                  -> admin     (governance oversight)
//   /users/:id/scopes         -> admin     (scope assignment)
// Unknown first segment falls back to "*" (whole-product flag).
func CategoryFetcher(path string) string {
	path = strings.Trim(path, "/")
	if path == "" {
		return "*"
	}
	// Strip the /api/v1 prefix if present.
	path = strings.TrimPrefix(path, "api/v1/")
	path = strings.TrimPrefix(path, "api/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return "*"
	}
	first := parts[0]
	switch first {
	case "wallets", "send", "sign", "transactions", "balance", "tokens",
		"nfts", "gas", "price", "chains", "swap", "staking", "non_evm",
		"address-book", "devices", "keystore", "admin", "users":
		if first == "users" {
			// /users/:id/scopes is a governance action -> admin fetcher.
			return "admin"
		}
		return first
	default:
		return "*"
	}
}

// IssueJWT mints a JWT for a user. The scopes claim carries the canonical
// scoped-admin taxonomy (set by the WL client via UpdateUserScopes) and is
// enforced by HasScope on admin routes.
func IssueJWT(secret string, userID uuid.UUID, email string, scopes []string, expiry time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		Scopes: scopes,
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
