package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret-that-is-long-enough-to-be-realistic"

func init() {
	gin.SetMode(gin.TestMode)
}

// newTestRouter builds the same middleware chain the service registers, without
// the database, so the access-control decisions can be tested on their own.
func newTestRouter() *gin.Engine {
	svc := &ProtectionFundService{}
	r := gin.New()

	protected := r.Group("/api/v1")
	protected.Use(svc.authMiddleware())
	protected.POST("/claims", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	reviewer := protected.Group("")
	reviewer.Use(svc.requireReviewer())
	reviewer.POST("/claims/:id/approve", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	reviewer.POST("/claims/:id/pay", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	return r
}

func signToken(t *testing.T, secret, sub, role string) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  sub,
		"role": role,
		"exp":  time.Now().Add(time.Hour).Unix(),
	})
	signed, err := tok.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("signing token: %v", err)
	}
	return signed
}

func do(r *gin.Engine, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// A claim submission is money-adjacent, so an anonymous caller must be refused.
func TestAnonymousRequestIsRejected(t *testing.T) {
	cfg.JWTSecret = testSecret
	r := newTestRouter()

	if w := do(r, http.MethodPost, "/api/v1/claims", nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous submit: got %d, want 401", w.Code)
	}
}

// With no secret configured the service cannot authenticate anyone. It must
// refuse rather than silently treat every caller as verified.
func TestMissingSecretFailsClosed(t *testing.T) {
	cfg.JWTSecret = ""
	r := newTestRouter()

	w := do(r, http.MethodPost, "/api/v1/claims", map[string]string{"Authorization": "Bearer anything"})
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("unset secret: got %d, want 503", w.Code)
	}
}

// A token signed with a different key must not be accepted.
func TestTokenSignedWithWrongSecretIsRejected(t *testing.T) {
	cfg.JWTSecret = testSecret
	r := newTestRouter()

	token := signToken(t, "some-other-secret", "user-1", "user")
	w := do(r, http.MethodPost, "/api/v1/claims", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("foreign signature: got %d, want 401", w.Code)
	}
}

// "alg: none" must never be honoured.
func TestUnsignedTokenIsRejected(t *testing.T) {
	cfg.JWTSecret = testSecret
	r := newTestRouter()

	tok := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"sub":  "attacker",
		"role": "admin",
		"exp":  time.Now().Add(time.Hour).Unix(),
	})
	unsigned, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("building unsigned token: %v", err)
	}

	w := do(r, http.MethodPost, "/api/v1/claims", map[string]string{"Authorization": "Bearer " + unsigned})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("alg=none: got %d, want 401", w.Code)
	}
}

// An ordinary authenticated user may submit and read, but must not be able to
// approve or pay a claim.
func TestRegularUserCannotApproveOrPay(t *testing.T) {
	cfg.JWTSecret = testSecret
	cfg.AdminToken = "service-token-value"
	r := newTestRouter()

	token := signToken(t, testSecret, "user-1", "user")
	auth := map[string]string{"Authorization": "Bearer " + token}

	if w := do(r, http.MethodPost, "/api/v1/claims", auth); w.Code != http.StatusOK {
		t.Fatalf("submit as user: got %d, want 200", w.Code)
	}
	for _, path := range []string{"/api/v1/claims/c1/approve", "/api/v1/claims/c1/pay"} {
		if w := do(r, http.MethodPost, path, auth); w.Code != http.StatusForbidden {
			t.Fatalf("user hitting %s: got %d, want 403", path, w.Code)
		}
	}
}

// Staff roles reach the reviewer chain.
func TestAdminRoleReachesReviewerChain(t *testing.T) {
	cfg.JWTSecret = testSecret
	cfg.AdminToken = ""
	r := newTestRouter()

	for _, role := range []string{"admin", "superadmin"} {
		token := signToken(t, testSecret, "staff-1", role)
		w := do(r, http.MethodPost, "/api/v1/claims/c1/approve", map[string]string{"Authorization": "Bearer " + token})
		if w.Code != http.StatusOK {
			t.Fatalf("role %q approve: got %d, want 200", role, w.Code)
		}
	}
}

// The service token lets an internal payout job drive the reviewer chain, and
// only an exact match is accepted.
func TestServiceTokenGrantsReviewerAccess(t *testing.T) {
	cfg.JWTSecret = testSecret
	cfg.AdminToken = "service-token-value"
	r := newTestRouter()

	token := signToken(t, testSecret, "job-1", "user")

	good := map[string]string{"Authorization": "Bearer " + token, "X-Service-Token": "service-token-value"}
	if w := do(r, http.MethodPost, "/api/v1/claims/c1/pay", good); w.Code != http.StatusOK {
		t.Fatalf("valid service token: got %d, want 200", w.Code)
	}

	bad := map[string]string{"Authorization": "Bearer " + token, "X-Service-Token": "service-token-valuX"}
	if w := do(r, http.MethodPost, "/api/v1/claims/c1/pay", bad); w.Code != http.StatusForbidden {
		t.Fatalf("wrong service token: got %d, want 403", w.Code)
	}
}

// With no service token configured, a user token must not open the chain.
func TestUnsetServiceTokenDoesNotOpenReviewerChain(t *testing.T) {
	cfg.JWTSecret = testSecret
	cfg.AdminToken = ""
	r := newTestRouter()

	token := signToken(t, testSecret, "user-1", "user")
	w := do(r, http.MethodPost, "/api/v1/claims/c1/pay", map[string]string{
		"Authorization":   "Bearer " + token,
		"X-Service-Token": "",
	})
	if w.Code != http.StatusForbidden {
		t.Fatalf("empty service token: got %d, want 403", w.Code)
	}
}

// newScopeRouter registers the read handlers so claim ownership can be tested.
func newScopeRouter(svc *ProtectionFundService) *gin.Engine {
	r := gin.New()
	protected := r.Group("/api/v1")
	protected.Use(svc.authMiddleware())
	protected.GET("/claims", svc.GetClaims)
	protected.GET("/claims/:id", svc.GetClaim)
	protected.GET("/users/:user_id/coverage", svc.GetUserCoverage)
	return r
}

func seedClaims(svc *ProtectionFundService) {
	svc.claims["claim-alice"] = &Claim{ID: "claim-alice", UserID: "alice", Status: "pending", IncidentType: "hack"}
	svc.claims["claim-bob"] = &Claim{ID: "claim-bob", UserID: "bob", Status: "pending", IncidentType: "hack"}
	svc.users["cov-alice"] = &UserCoverage{ID: "cov-alice", UserID: "alice", Active: true}
}

// A user must not be able to read another user's claim by guessing its id.
func TestCannotReadAnotherUsersClaimByID(t *testing.T) {
	cfg.JWTSecret = testSecret
	svc := &ProtectionFundService{claims: map[string]*Claim{}, users: map[string]*UserCoverage{}}
	seedClaims(svc)
	r := newScopeRouter(svc)

	alice := map[string]string{"Authorization": "Bearer " + signToken(t, testSecret, "alice", "user")}

	if w := do(r, http.MethodGet, "/api/v1/claims/claim-alice", alice); w.Code != http.StatusOK {
		t.Fatalf("reading own claim: got %d, want 200", w.Code)
	}
	// 404 rather than 403 so the endpoint does not confirm the claim exists.
	if w := do(r, http.MethodGet, "/api/v1/claims/claim-bob", alice); w.Code != http.StatusNotFound {
		t.Fatalf("reading another user's claim: got %d, want 404", w.Code)
	}
}

// Listing claims must not leak other users' records, even when user_id is
// supplied as a query parameter.
func TestClaimListIsScopedToCaller(t *testing.T) {
	cfg.JWTSecret = testSecret
	svc := &ProtectionFundService{claims: map[string]*Claim{}, users: map[string]*UserCoverage{}}
	seedClaims(svc)
	r := newScopeRouter(svc)

	alice := map[string]string{"Authorization": "Bearer " + signToken(t, testSecret, "alice", "user")}

	w := do(r, http.MethodGet, "/api/v1/claims", alice)
	if w.Code != http.StatusOK {
		t.Fatalf("list own claims: got %d, want 200", w.Code)
	}
	if body := w.Body.String(); !contains(body, "claim-alice") || contains(body, "claim-bob") {
		t.Fatalf("list leaked foreign claims: %s", body)
	}

	// Asking for bob's claims must still return only alice's.
	w = do(r, http.MethodGet, "/api/v1/claims?user_id=bob", alice)
	if body := w.Body.String(); contains(body, "claim-bob") {
		t.Fatalf("user_id override leaked foreign claims: %s", body)
	}
}

// Staff legitimately need the full list.
func TestStaffSeesAllClaims(t *testing.T) {
	cfg.JWTSecret = testSecret
	svc := &ProtectionFundService{claims: map[string]*Claim{}, users: map[string]*UserCoverage{}}
	seedClaims(svc)
	r := newScopeRouter(svc)

	admin := map[string]string{"Authorization": "Bearer " + signToken(t, testSecret, "staff-1", "admin")}
	w := do(r, http.MethodGet, "/api/v1/claims", admin)
	if body := w.Body.String(); !contains(body, "claim-alice") || !contains(body, "claim-bob") {
		t.Fatalf("staff should see every claim: %s", body)
	}
}

// Coverage for another user must be refused.
func TestCannotReadAnotherUsersCoverage(t *testing.T) {
	cfg.JWTSecret = testSecret
	svc := &ProtectionFundService{claims: map[string]*Claim{}, users: map[string]*UserCoverage{}}
	seedClaims(svc)
	r := newScopeRouter(svc)

	alice := map[string]string{"Authorization": "Bearer " + signToken(t, testSecret, "alice", "user")}

	if w := do(r, http.MethodGet, "/api/v1/users/alice/coverage", alice); w.Code != http.StatusOK {
		t.Fatalf("own coverage: got %d, want 200", w.Code)
	}
	if w := do(r, http.MethodGet, "/api/v1/users/bob/coverage", alice); w.Code != http.StatusForbidden {
		t.Fatalf("foreign coverage: got %d, want 403", w.Code)
	}
}

func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
