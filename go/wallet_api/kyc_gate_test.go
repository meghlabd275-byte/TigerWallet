package main

// kyc_gate_test.go — tests for the P2P and card KYC gate.
//
// The gate reads a status from the in-house KYC service and decides whether a
// user may trade P2P or hold a card. These tests use a stub HTTP server that
// returns each status the service can emit, because the defect being guarded
// against was a status-interpretation bug, not a transport bug: the gate
// previously accepted `Level >= 2` on its own, so a REJECTED applicant at level
// 2 passed. Every rejection and pending state must now close the gate.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ensureConfig gives the gate a config object to read. The production binary
// loads it at startup; the test binary has not.
func ensureConfig() {
	if appConfig == nil {
		appConfig = &AppConfig{}
	}
}

// withStubKYC points the gate at a stub service returning `body` with `code`.
func withStubKYC(t *testing.T, code int, body map[string]interface{}, token string) {
	t.Helper()
	ensureConfig()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token != "" && r.Header.Get("X-KYC-Token") != token {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"invalid or missing service token"}`))
			return
		}
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(srv.Close)

	prevURL := appConfig.ListingServiceURL
	prevTok := appConfig.KYCServiceToken
	appConfig.ListingServiceURL = srv.URL
	appConfig.KYCServiceToken = token
	t.Cleanup(func() {
		appConfig.ListingServiceURL = prevURL
		appConfig.KYCServiceToken = prevTok
	})
}

func TestKYCGateAcceptsVerifiedStatus(t *testing.T) {
	withStubKYC(t, http.StatusOK, map[string]interface{}{
		"status": "verified", "verified": true, "level": 2,
	}, "tok")

	if !kycVerified(context.Background(), uuid.New()) {
		t.Fatal("a verified applicant must pass the gate")
	}
}

func TestKYCGateAcceptsApprovedStatus(t *testing.T) {
	withStubKYC(t, http.StatusOK, map[string]interface{}{
		"status": "approved", "verified": false, "level": 2,
	}, "tok")

	if !kycVerified(context.Background(), uuid.New()) {
		t.Fatal("an approved applicant at the required level must pass the gate")
	}
}

// This is the regression that mattered: a rejected applicant at level 2 used to
// pass, because `Level >= 2` was accepted without regard to status.
func TestKYCGateRejectsRejectedStatusEvenAtLevel2(t *testing.T) {
	withStubKYC(t, http.StatusOK, map[string]interface{}{
		"status": "rejected", "verified": false, "level": 2,
	}, "tok")

	if kycVerified(context.Background(), uuid.New()) {
		t.Fatal("a REJECTED applicant at level 2 must NOT pass the gate")
	}
}

func TestKYCGateRejectsEveryNonApprovingStatus(t *testing.T) {
	for _, status := range []string{
		"not_started", "pending", "pending_review", "reviewing", "in_review", "expired",
	} {
		t.Run(status, func(t *testing.T) {
			withStubKYC(t, http.StatusOK, map[string]interface{}{
				"status": status, "verified": false, "level": 3,
			}, "tok")
			if kycVerified(context.Background(), uuid.New()) {
				t.Fatalf("status %q must not pass the gate", status)
			}
		})
	}
}

func TestKYCGateRejectsVerifiedFlagWithNonApprovingStatus(t *testing.T) {
	// A contradictory payload must be resolved conservatively.
	withStubKYC(t, http.StatusOK, map[string]interface{}{
		"status": "rejected", "verified": true, "level": 2,
	}, "tok")

	if kycVerified(context.Background(), uuid.New()) {
		t.Fatal("a rejecting status must win over a contradictory verified flag")
	}
}

func TestKYCGateRequiresTheMinimumLevel(t *testing.T) {
	withStubKYC(t, http.StatusOK, map[string]interface{}{
		"status": "verified", "verified": true, "level": 1,
	}, "tok")

	if kycVerified(context.Background(), uuid.New()) {
		t.Fatal("a verified applicant below the required level must not pass")
	}
}

func TestKYCGateFailsClosedWithoutAConfiguredBackend(t *testing.T) {
	ensureConfig()
	prevURL := appConfig.ListingServiceURL
	appConfig.ListingServiceURL = ""
	t.Cleanup(func() { appConfig.ListingServiceURL = prevURL })

	if kycVerified(context.Background(), uuid.New()) {
		t.Fatal("with no KYC backend configured the gate must stay closed")
	}
}

func TestKYCGateFailsClosedOnServiceError(t *testing.T) {
	withStubKYC(t, http.StatusInternalServerError, map[string]interface{}{
		"error": "boom",
	}, "tok")

	if kycVerified(context.Background(), uuid.New()) {
		t.Fatal("a failing KYC service must close the gate, not open it")
	}
}

func TestKYCGateFailsClosedOnUnauthorized(t *testing.T) {
	// The stub demands "right"; the gate is configured with "wrong".
	withStubKYC(t, http.StatusOK, map[string]interface{}{
		"status": "verified", "verified": true, "level": 2,
	}, "right")
	appConfig.KYCServiceToken = "wrong"

	if kycVerified(context.Background(), uuid.New()) {
		t.Fatal("an unauthenticated status read must not be trusted")
	}
}

func TestKYCGateSendsTheConfiguredToken(t *testing.T) {
	var seen string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get("X-KYC-Token")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"verified","verified":true,"level":2}`))
	}))
	defer srv.Close()

	prevURL := appConfig.ListingServiceURL
	prevTok := appConfig.KYCServiceToken
	appConfig.ListingServiceURL = srv.URL
	appConfig.KYCServiceToken = "the-token"
	defer func() {
		appConfig.ListingServiceURL = prevURL
		appConfig.KYCServiceToken = prevTok
	}()

	if !kycVerified(context.Background(), uuid.New()) {
		t.Fatal("expected the gate to pass")
	}
	if seen != "the-token" {
		t.Fatalf("the gate must present the configured token; server saw %q", seen)
	}
}

func TestKYCGateFailsClosedOnMalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{not json`))
	}))
	defer srv.Close()

	prevURL := appConfig.ListingServiceURL
	appConfig.ListingServiceURL = srv.URL
	defer func() { appConfig.ListingServiceURL = prevURL }()

	if kycVerified(context.Background(), uuid.New()) {
		t.Fatal("a malformed status response must close the gate")
	}
}

// The card routes are wrapped in requireKYCMiddleware, so a card request from an
// unverified user must be refused before the upstream card service is reached.
func TestCardRouteRefusesUnverifiedUser(t *testing.T) {
	ensureConfig()
	withStubKYC(t, http.StatusOK, map[string]interface{}{
		"status": "not_started", "verified": false, "level": 0,
	}, "tok")

	r := gin.New()
	r.GET("/api/v1/cards/balance", func(c *gin.Context) {
		c.Set("userID", uuid.New().String())
	}, requireKYCMiddleware("card issuance"), func(c *gin.Context) {
		c.String(http.StatusOK, "reached upstream")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cards/balance", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("an unverified user must be refused a card; got HTTP %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "kyc_required") {
		t.Fatalf("the refusal must be machine-readable; body was %q", w.Body.String())
	}
}

// A verified user must pass the card gate.
func TestCardRouteAllowsVerifiedUser(t *testing.T) {
	ensureConfig()
	withStubKYC(t, http.StatusOK, map[string]interface{}{
		"status": "verified", "verified": true, "level": 2,
	}, "tok")

	r := gin.New()
	r.GET("/api/v1/cards/balance", func(c *gin.Context) {
		c.Set("userID", uuid.New().String())
	}, requireKYCMiddleware("card issuance"), func(c *gin.Context) {
		c.String(http.StatusOK, "reached upstream")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cards/balance", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("a verified user must reach the card service; got HTTP %d body %q", w.Code, w.Body.String())
	}
}

// When the KYC backend is down the card gate must refuse, not allow.
func TestCardRouteFailsClosedWhenKYCIsDown(t *testing.T) {
	ensureConfig()
	withStubKYC(t, http.StatusInternalServerError, map[string]interface{}{"error": "down"}, "tok")

	r := gin.New()
	r.GET("/api/v1/cards/balance", func(c *gin.Context) {
		c.Set("userID", uuid.New().String())
	}, requireKYCMiddleware("card issuance"), func(c *gin.Context) {
		c.String(http.StatusOK, "reached upstream")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cards/balance", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("a KYC outage must close the card gate; got HTTP %d", w.Code)
	}
}
