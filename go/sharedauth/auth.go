// Package sharedauth authenticates callers of the standalone Go microservices.
//
// These services are reachable over the network and act on behalf of a user, so
// each one must establish who is calling before it reads or writes that user's
// data. The identity is never taken from the request body: it comes from the
// bearer token that wallet_api issued, using the same secret and claim layout
// (sub, role, exp) that go/wallet_api/auth.go produces.
package sharedauth

import (
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// Identity is the authenticated caller.
type Identity struct {
	UserID string
	Role   string
}

// IsAdmin reports whether the caller holds the admin role.
func (id Identity) IsAdmin() bool {
	return id.Role == "admin" || id.Role == "super_admin"
}

var ErrNoSecret = errors.New("JWT_SECRET is not set")

// Secret returns the shared signing secret.
//
// There is deliberately no fallback. A default would let anyone who has read
// this repository mint a token for any user, which is the same as having no
// authentication at all.
func Secret() (string, error) {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		return "", ErrNoSecret
	}
	return s, nil
}

// Parse validates a token and returns the caller it identifies.
func Parse(secret, token string) (Identity, error) {
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		// Reject alg=none and any asymmetric algorithm, which would let a
		// caller sign a token with the public half of a key.
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	}, jwt.WithExpirationRequired())
	if err != nil {
		return Identity{}, errors.New("invalid or expired token")
	}
	if !parsed.Valid {
		return Identity{}, errors.New("invalid or expired token")
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return Identity{}, errors.New("invalid claims")
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return Identity{}, errors.New("missing subject")
	}
	role, _ := claims["role"].(string)
	if role == "" {
		role = "user"
	}
	return Identity{UserID: sub, Role: role}, nil
}

// Bearer extracts the token from an Authorization header.
func Bearer(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	if len(h) < 8 || !strings.EqualFold(h[:7], "Bearer ") {
		return "", false
	}
	return strings.TrimSpace(h[7:]), true
}

// Authenticate resolves the caller, or writes 401 and reports false.
func Authenticate(w http.ResponseWriter, r *http.Request) (Identity, bool) {
	secret, err := Secret()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "authentication is not configured")
		return Identity{}, false
	}
	token, ok := Bearer(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing bearer token")
		return Identity{}, false
	}
	id, err := Parse(secret, token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return Identity{}, false
	}
	return id, true
}

// RequireAdmin authenticates and additionally requires an admin role.
func RequireAdmin(w http.ResponseWriter, r *http.Request) (Identity, bool) {
	id, ok := Authenticate(w, r)
	if !ok {
		return Identity{}, false
	}
	if !id.IsAdmin() {
		writeError(w, http.StatusForbidden, "admin role required")
		return Identity{}, false
	}
	return id, true
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"success":false,"error":"` + jsonEscape(msg) + `"}`))
}

func jsonEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 {
				b.WriteString(`\u00`)
				const hex = "0123456789abcdef"
				b.WriteByte(hex[(r>>4)&0xf])
				b.WriteByte(hex[r&0xf])
				continue
			}
			b.WriteRune(r)
		}
	}
	return b.String()
}
