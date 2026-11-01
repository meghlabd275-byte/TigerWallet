package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
)

const kycTestSecret = "kyc-test-secret"

func newKYCRouter(t *testing.T) (*gin.Engine, func(sub, role string) string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	logger = zerolog.Nop()

	mint := func(sub, role string) string {
		tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub":  sub,
			"role": role,
			"exp":  time.Now().Add(time.Hour).Unix(),
		})
		s, err := tok.SignedString([]byte(kycTestSecret))
		if err != nil {
			t.Fatal(err)
		}
		return s
	}

	return setupRouter(Config{JWTSecret: kycTestSecret}), mint
}

func kycReq(t *testing.T, r *gin.Engine, method, path, token string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestKYCAnonymousRejected proves the endpoints are no longer open.
func TestKYCAnonymousRejected(t *testing.T) {
	r, _ := newKYCRouter(t)
	for _, tc := range []struct{ method, path string }{
		{http.MethodPost, "/api/v1/workflow/override"},
		{http.MethodPost, "/api/v1/approval/rules"},
		{http.MethodGet, "/api/v1/queue/status"},
		{http.MethodGet, "/api/v1/kyc/applications/abc"},
	} {
		if w := kycReq(t, r, tc.method, tc.path, "", nil); w.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s without token returned %d, want 401", tc.method, tc.path, w.Code)
		}
	}
}

// TestKYCFailsClosedWithoutSecret ensures a misconfigured deployment does not
// serve unauthenticated traffic.
func TestKYCFailsClosedWithoutSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger = zerolog.Nop()
	r := setupRouter(Config{})
	if w := kycReq(t, r, http.MethodPost, "/api/v1/workflow/override", "anything", nil); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("missing secret returned %d, want 503", w.Code)
	}
}

// TestSelfApprovalBlocked is the core regression guard: an ordinary user must
// not be able to approve their own KYC application.
func TestSelfApprovalBlocked(t *testing.T) {
	r, mint := newKYCRouter(t)

	w := kycReq(t, r, http.MethodPost, "/api/v1/workflow/override", mint("attacker", "user"), map[string]interface{}{
		"entityType": "kyc",
		"entityID":   "some-application",
		"action":     "approve",
		"reason":     "self approved",
	})
	if w.Code != http.StatusForbidden {
		t.Fatalf("self-approval returned %d, want 403: %s", w.Code, w.Body.String())
	}
}

// TestWhiteLabelSelfApprovalBlocked covers the same escalation on the
// white-label path, which would grant tenant privileges.
func TestWhiteLabelSelfApprovalBlocked(t *testing.T) {
	r, mint := newKYCRouter(t)

	w := kycReq(t, r, http.MethodPost, "/api/v1/workflow/override", mint("attacker", "user"), map[string]interface{}{
		"entityType": "white_label",
		"entityID":   "wl-application",
		"action":     "approve",
	})
	if w.Code != http.StatusForbidden {
		t.Fatalf("white-label self-approval returned %d, want 403", w.Code)
	}
}

// TestNonReviewerCannotManageRules proves rule creation is also gated.
func TestNonReviewerCannotManageRules(t *testing.T) {
	r, mint := newKYCRouter(t)

	if w := kycReq(t, r, http.MethodPost, "/api/v1/approval/rules", mint("user-1", "user"),
		map[string]interface{}{"name": "auto approve everything"}); w.Code != http.StatusForbidden {
		t.Fatalf("rule creation by a plain user returned %d, want 403", w.Code)
	}
	if w := kycReq(t, r, http.MethodGet, "/api/v1/approval/rules", mint("user-1", "user"), nil); w.Code != http.StatusForbidden {
		t.Fatalf("rule listing by a plain user returned %d, want 403", w.Code)
	}
}

// TestReviewerRolesAllowed confirms legitimate compliance staff are not blocked.
func TestReviewerRolesAllowed(t *testing.T) {
	for _, role := range []string{"admin", "super_admin", "compliance", "reviewer"} {
		gin.SetMode(gin.TestMode)
		logger = zerolog.Nop()
		secret := "role-test-secret"
		tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "staff-1", "role": role, "exp": time.Now().Add(time.Hour).Unix(),
		})
		signed, _ := tok.SignedString([]byte(secret))
		r := setupRouter(Config{JWTSecret: secret})

		req := httptest.NewRequest(http.MethodGet, "/api/v1/approval/rules", nil)
		req.Header.Set("Authorization", "Bearer "+signed)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code == http.StatusForbidden || w.Code == http.StatusUnauthorized {
			t.Fatalf("role %q was rejected with %d", role, w.Code)
		}
	}
}

// TestForgedTokenRejected ensures a token signed with the wrong key fails.
func TestForgedTokenRejected(t *testing.T) {
	r, _ := newKYCRouter(t)
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "x", "role": "admin"})
	signed, _ := tok.SignedString([]byte("wrong-secret"))
	if w := kycReq(t, r, http.MethodGet, "/api/v1/approval/rules", signed, nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("forged token returned %d, want 401", w.Code)
	}
}

// TestNoneAlgorithmRejected guards against the classic "alg: none" bypass.
func TestNoneAlgorithmRejected(t *testing.T) {
	r, _ := newKYCRouter(t)
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"sub": "x", "role": "admin"})
	signed, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Skip("library refused to sign none algorithm")
	}
	if w := kycReq(t, r, http.MethodGet, "/api/v1/approval/rules", signed, nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("alg=none token returned %d, want 401", w.Code)
	}
}
