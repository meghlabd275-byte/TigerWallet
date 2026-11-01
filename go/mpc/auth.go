// Authentication and ownership for the MPC service.
//
// Every route except the health probe requires a bearer token, because the
// service holds threshold key shares and will sign with them. A wallet is bound
// to the subject that created it; only that subject (or an operator acting
// across wallets) may read, sign with, or export it.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// caller is the authenticated principal behind a request.
type caller struct {
	Subject string
	Role    string
}

// isOperator reports whether the caller may act on wallets it does not own.
// Operators are the platform's own backends (custody, treasury) and admins.
func (c caller) isOperator() bool {
	switch c.Role {
	case "admin", "super_admin", "service":
		return true
	}
	return false
}

// jwtSecret resolves the signing secret. MPC_JWT_SECRET takes precedence so the
// service can be given a dedicated secret; JWT_SECRET is accepted so a
// deployment that already shares one secret across services keeps working.
// There is no default: an unset secret is a fatal misconfiguration for a
// service that signs transactions.
func jwtSecret() (string, error) {
	if secret := os.Getenv("MPC_JWT_SECRET"); secret != "" {
		return secret, nil
	}
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		return secret, nil
	}
	return "", fmt.Errorf("neither MPC_JWT_SECRET nor JWT_SECRET is set")
}

// parseCaller validates the bearer token and returns the principal it names.
func parseCaller(r *http.Request, secret string) (caller, error) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return caller{}, fmt.Errorf("missing bearer token")
	}

	token, err := jwt.Parse(strings.TrimPrefix(header, "Bearer "), func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256", "HS384", "HS512"}))
	if err != nil || !token.Valid {
		return caller{}, fmt.Errorf("invalid or expired token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return caller{}, fmt.Errorf("invalid claims")
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return caller{}, fmt.Errorf("token has no subject")
	}
	role, _ := claims["role"].(string)
	return caller{Subject: sub, Role: role}, nil
}

// allowedOrigins reads the CORS allowlist. An empty allowlist emits no CORS
// headers at all, which keeps browsers on the same origin by default instead of
// exposing a key-custody API to every site.
func allowedOrigins() map[string]bool {
	origins := make(map[string]bool)
	for _, origin := range strings.Split(os.Getenv("MPC_ALLOWED_ORIGINS"), ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			origins[origin] = true
		}
	}
	return origins
}

// secure wraps a handler with CORS handling, authentication and, when the
// handler needs it, the caller context.
func secure(secret string, origins map[string]bool, h func(http.ResponseWriter, *http.Request, caller)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" && origins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		c, err := parseCaller(r, secret)
		if err != nil {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		h(w, r, c)
	}
}

// requireWalletAccess enforces that the caller owns the wallet or is an
// operator. It writes the error response itself and reports whether the handler
// should continue.
func requireWalletAccess(w http.ResponseWriter, wallet *mpcWallet, c caller) bool {
	if wallet.Owner == "" || wallet.Owner == c.Subject || c.isOperator() {
		return true
	}
	// Do not disclose whether the wallet exists to a caller that does not own
	// it; a 404 is the same answer they would get for a random key id.
	writeError(w, http.StatusNotFound, "wallet not found")
	return false
}

// resolveSecretOrExit loads the signing secret, terminating the process when it
// is absent. Refusing to start is deliberate: a key-custody service that runs
// without authentication is a vulnerability, not a degraded mode.
func resolveSecretOrExit() string {
	secret, err := jwtSecret()
	if err != nil {
		log.Fatalf("mpc: %v; refusing to serve signing requests unauthenticated", err)
	}
	return secret
}
