package main

// Authentication and authorization for the P2P trading service.
//
// This service moves value: releasing escrow pays the seller. It is published
// on its own host port, so the wallet-api KYC/JWT proxy in front of it is not a
// security boundary — a caller can reach this process directly. Every mutating
// request therefore has to carry its own identity, and the identity has to be
// the one recorded on the order or advert being touched.
//
// Tokens are the same HS256 bearer tokens wallet-api issues, validated with the
// shared JWT_SECRET.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// authSecret is the HMAC key shared with wallet-api. It is loaded once at
// startup by requireAuthSecret, which refuses to start the service without it.
var authSecret []byte

// requireAuthSecret loads JWT_SECRET, failing closed. A service that authorises
// value transfers must not run without a way to verify who is calling.
func requireAuthSecret() error {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if len(secret) < 32 {
		return fmt.Errorf("JWT_SECRET is required and must be at least 32 characters")
	}
	authSecret = []byte(secret)
	return nil
}

type caller struct {
	UserID string
	Role   string
}

type userClaims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// authenticate extracts and verifies the caller identity from the request.
func authenticate(r *http.Request) (*caller, error) {
	header := r.Header.Get("Authorization")
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return nil, errors.New("bearer token required")
	}

	claims := &userClaims{}
	token, err := jwt.ParseWithClaims(parts[1], claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return authSecret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !token.Valid {
		return nil, errors.New("invalid or expired token")
	}
	if claims.Subject == "" {
		return nil, errors.New("token has no subject")
	}
	role := claims.Role
	if role == "" {
		role = "user"
	}
	return &caller{UserID: claims.Subject, Role: role}, nil
}

func isAdminRole(role string) bool {
	switch role {
	case "admin", "super_admin", "wl_admin", "master_wallet_admin":
		return true
	}
	return false
}

// writeAuthError maps an authentication failure to a response. It deliberately
// does not echo the underlying reason, which would help an attacker.
func writeAuthError(w http.ResponseWriter, err error) {
	status := http.StatusUnauthorized
	if err.Error() == "bearer token required" {
		status = http.StatusUnauthorized
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":"authentication required"}`))
}

// requireCaller authenticates the request and returns the caller, writing the
// error response itself when authentication fails.
func requireCaller(w http.ResponseWriter, r *http.Request) (*caller, bool) {
	c, err := authenticate(r)
	if err != nil {
		writeAuthError(w, err)
		return nil, false
	}
	return c, true
}

// allow acts as an in-process rate limit so a single caller cannot hammer the
// order book. It is intentionally simple: this service runs one replica per
// deployment and the gateway applies the cluster-wide limit.
var lastAction = map[string]time.Time{}

func tooFast(userID string) bool {
	now := time.Now()
	if prev, ok := lastAction[userID]; ok && now.Sub(prev) < 100*time.Millisecond {
		return true
	}
	lastAction[userID] = now
	return false
}

// audit records a state change against the order it touched. Failures are
// logged but never block the operation, since the primary record is the order
// row itself.
func (s *Server) audit(ctx context.Context, orderID, actorID, action, from, to string) {
	if s.db == nil {
		return
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO p2p_order_audit (order_id, actor_id, action, from_status, to_status, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`, orderID, actorID, action, from, to); err != nil {
		logf("p2p audit write failed: %v", err)
	}
}

func logf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}
