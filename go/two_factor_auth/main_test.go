package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret"

func token(t *testing.T, sub, role string) string {
	t.Helper()
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  sub,
		"role": role,
		"exp":  time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return s
}

func newTestService(t *testing.T) *TwoFactorService {
	t.Helper()
	gin.SetMode(gin.TestMode)
	return NewTwoFactorService(&Config{Port: "0", IssuerName: "TigerWallet", JWTSecret: testSecret})
}

func routerFor(s *TwoFactorService) *gin.Engine {
	r := gin.New()
	s.RegisterRoutes(r)
	return r
}

// do issues a request with an optional bearer token.
func do(r *gin.Engine, method, path, body, bearer string) *httptest.ResponseRecorder {
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRoutesRejectMissingToken(t *testing.T) {
	s := newTestService(t)
	r := routerFor(s)
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodPost, "/api/v1/2fa/setup", `{"userId":"victim","email":"v@example.com"}`},
		{http.MethodPost, "/api/v1/2fa/disable", `{"userId":"victim","code":"123456"}`},
		{http.MethodGet, "/api/v1/2fa/status/victim", ""},
		{http.MethodPost, "/api/v1/2fa/webauthn/register", `{"userId":"victim","deviceName":"d","publicKey":"pk"}`},
	} {
		w := do(r, tc.method, tc.path, tc.body, "")
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without token: got %d, want 401", tc.method, tc.path, w.Code)
		}
	}
}

// A caller must not be able to disable another account's second factor. This
// was possible before the routes authenticated the caller.
func TestDisableRejectsOtherUser(t *testing.T) {
	s := newTestService(t)
	r := routerFor(s)
	w := do(r, http.MethodPost, "/api/v1/2fa/disable",
		`{"userId":"victim","code":"123456"}`, token(t, "attacker", "user"))
	if w.Code != http.StatusForbidden {
		t.Fatalf("cross-user disable: got %d, want 403 (%s)", w.Code, w.Body.String())
	}
}

func TestSetupRejectsOtherUser(t *testing.T) {
	s := newTestService(t)
	r := routerFor(s)
	w := do(r, http.MethodPost, "/api/v1/2fa/setup",
		`{"userId":"victim","email":"v@example.com"}`, token(t, "attacker", "user"))
	if w.Code != http.StatusForbidden {
		t.Fatalf("cross-user setup: got %d, want 403 (%s)", w.Code, w.Body.String())
	}
}

func TestStatusRejectsOtherUser(t *testing.T) {
	s := newTestService(t)
	r := routerFor(s)
	w := do(r, http.MethodGet, "/api/v1/2fa/status/victim", "", token(t, "attacker", "user"))
	if w.Code != http.StatusForbidden {
		t.Fatalf("cross-user status: got %d, want 403", w.Code)
	}
}

func TestWebAuthnCredentialListRejectsOtherUser(t *testing.T) {
	s := newTestService(t)
	r := routerFor(s)
	w := do(r, http.MethodGet, "/api/v1/2fa/webauthn/credentials/victim", "", token(t, "attacker", "user"))
	if w.Code != http.StatusForbidden {
		t.Fatalf("cross-user credential list: got %d, want 403", w.Code)
	}
}

func TestOwnerCanReachOwnRoutes(t *testing.T) {
	s := newTestService(t)
	r := routerFor(s)
	// Setup for self succeeds and returns a secret.
	w := do(r, http.MethodPost, "/api/v1/2fa/setup",
		`{"userId":"alice","email":"alice@example.com"}`, token(t, "alice", "user"))
	if w.Code != http.StatusOK {
		t.Fatalf("own setup: got %d (%s)", w.Code, w.Body.String())
	}
	var resp struct {
		Secret string `json:"secret"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Secret == "" {
		t.Fatal("setup did not return a secret")
	}
}

func TestAdminMayActOnAnotherUser(t *testing.T) {
	s := newTestService(t)
	r := routerFor(s)
	if w := do(r, http.MethodPost, "/api/v1/2fa/setup",
		`{"userId":"bob","email":"bob@example.com"}`, token(t, "bob", "user")); w.Code != http.StatusOK {
		t.Fatalf("seed bob: %d", w.Code)
	}
	// Admin reaching another user's status is a support workflow, not a bypass.
	w := do(r, http.MethodGet, "/api/v1/2fa/status/bob", "", token(t, "root", "super_admin"))
	if w.Code != http.StatusOK {
		t.Fatalf("admin status: got %d (%s)", w.Code, w.Body.String())
	}
}

func TestUnsignedTokenRejected(t *testing.T) {
	s := newTestService(t)
	r := routerFor(s)
	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"sub": "victim",
		"exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign none: %v", err)
	}
	w := do(r, http.MethodGet, "/api/v1/2fa/status/victim", "", unsigned)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("alg=none: got %d, want 401", w.Code)
	}
}
