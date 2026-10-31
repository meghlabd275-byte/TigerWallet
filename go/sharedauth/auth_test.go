package sharedauth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func sign(t *testing.T, secret string, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return s
}

func TestParseRejectsUnsignedToken(t *testing.T) {
	// alg=none must never be accepted, even though it parses.
	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"sub": "attacker",
		"exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign none: %v", err)
	}
	if _, err := Parse("secret", unsigned); err == nil {
		t.Fatal("alg=none token was accepted")
	}
}

func TestParseRejectsWrongSecret(t *testing.T) {
	tok := sign(t, "other-secret", jwt.MapClaims{
		"sub": "victim",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	if _, err := Parse("secret", tok); err == nil {
		t.Fatal("token signed with a different secret was accepted")
	}
}

func TestParseRejectsExpiredToken(t *testing.T) {
	tok := sign(t, "secret", jwt.MapClaims{
		"sub": "victim",
		"exp": time.Now().Add(-time.Minute).Unix(),
	})
	if _, err := Parse("secret", tok); err == nil {
		t.Fatal("expired token was accepted")
	}
}

func TestParseRejectsTokenWithoutExpiry(t *testing.T) {
	// WithExpirationRequired: a token that never expires must not be accepted.
	tok := sign(t, "secret", jwt.MapClaims{"sub": "victim"})
	if _, err := Parse("secret", tok); err == nil {
		t.Fatal("token without exp was accepted")
	}
}

func TestParseReturnsSubjectAndRole(t *testing.T) {
	tok := sign(t, "secret", jwt.MapClaims{
		"sub":  "user-1",
		"role": "admin",
		"exp":  time.Now().Add(time.Hour).Unix(),
	})
	id, err := Parse("secret", tok)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if id.UserID != "user-1" || !id.IsAdmin() {
		t.Fatalf("got %+v", id)
	}
}

func TestSecretHasNoFallback(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	if _, err := Secret(); err == nil {
		t.Fatal("Secret returned a value with JWT_SECRET unset")
	}
	t.Setenv("JWT_SECRET", "abc")
	got, err := Secret()
	if err != nil || got != "abc" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestAuthenticateRejectsMissingAndMalformedHeader(t *testing.T) {
	t.Setenv("JWT_SECRET", "secret")
	for _, h := range []string{"", "Basic abc", "Bearer", "Bearer   "} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if h != "" {
			req.Header.Set("Authorization", h)
		}
		w := httptest.NewRecorder()
		if _, ok := Authenticate(w, req); ok {
			t.Fatalf("header %q was accepted", h)
		}
		if w.Code != http.StatusUnauthorized && w.Code != http.StatusServiceUnavailable {
			t.Fatalf("header %q gave status %d", h, w.Code)
		}
	}
}

func TestAuthenticateAcceptsValidToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "secret")
	tok := sign(t, "secret", jwt.MapClaims{
		"sub": "user-9",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	id, ok := Authenticate(w, req)
	if !ok {
		t.Fatalf("valid token rejected: %d %s", w.Code, w.Body.String())
	}
	if id.UserID != "user-9" {
		t.Fatalf("got %+v", id)
	}
}

func TestRequireAdmin(t *testing.T) {
	t.Setenv("JWT_SECRET", "secret")
	user := sign(t, "secret", jwt.MapClaims{
		"sub": "u", "role": "user", "exp": time.Now().Add(time.Hour).Unix(),
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+user)
	w := httptest.NewRecorder()
	if _, ok := RequireAdmin(w, req); ok {
		t.Fatal("non-admin passed RequireAdmin")
	}
	if w.Code != http.StatusForbidden {
		t.Fatalf("status %d", w.Code)
	}

	admin := sign(t, "secret", jwt.MapClaims{
		"sub": "a", "role": "super_admin", "exp": time.Now().Add(time.Hour).Unix(),
	})
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+admin)
	w = httptest.NewRecorder()
	if _, ok := RequireAdmin(w, req); !ok {
		t.Fatalf("admin rejected: %d %s", w.Code, w.Body.String())
	}
}

func TestWriteErrorEscapesQuotes(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, http.StatusBadRequest, `bad "quote"`)
	body := w.Body.String()
	if body != `{"success":false,"error":"bad \"quote\""}` {
		t.Fatalf("got %s", body)
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
