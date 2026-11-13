package main

// kyc_status_test.go — tests for the KYC status endpoint the wallet gates call.
//
// The wallet backend's P2P and card gates read this endpoint. Before this was
// delegated to the in-house engine, the local implementation could only ever
// produce "pending_review", so no user could reach "verified" and P2P trading
// was permanently blocked. These tests pin the contract: an approving status
// passes through, a rejection does not, and an unreachable engine is reported
// as unavailable rather than as an approving status.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// withEngine points the delegation at a stub engine.
func withEngine(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	t.Setenv("KYC_ENGINE_URL", srv.URL)
}

func TestKYCStatusPassesThroughAVerifiedDecision(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withEngine(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-KYC-Token") != "tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "verified", "verified": true, "level": 2, "risk_score": 0,
		})
	})
	t.Setenv("KYC_SERVICE_TOKEN", "tok")

	r := gin.New()
	ls := &ListingService{}
	r.GET("/api/v1/kyc/status/:user_id", ls.GetKYCStatus)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/kyc/status/abc", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if out["status"] != "verified" || out["verified"] != true {
		t.Fatalf("the verified decision must pass through; got %s", w.Body.String())
	}
}

func TestKYCStatusDoesNotReportARejectionAsVerified(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withEngine(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "rejected", "verified": false, "level": 2,
		})
	})

	r := gin.New()
	ls := &ListingService{}
	r.GET("/api/v1/kyc/status/:user_id", ls.GetKYCStatus)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/kyc/status/abc", nil))

	var out map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if out["verified"] == true {
		t.Fatalf("a rejected applicant must not be reported verified; got %s", w.Body.String())
	}
}

func TestKYCStatusReportsAnUnreachableEngineAsUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Point at a closed port so the request fails.
	t.Setenv("KYC_ENGINE_URL", "http://127.0.0.1:1")

	r := gin.New()
	ls := &ListingService{}
	r.GET("/api/v1/kyc/status/:user_id", ls.GetKYCStatus)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/kyc/status/abc", nil))

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("an unreachable engine must be reported as unavailable; got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"verified":false`) {
		t.Fatalf("the response must explicitly deny verification; got %s", w.Body.String())
	}
}

func TestKYCStatusReportsAMissingEngineConfiguration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("KYC_ENGINE_URL", "")

	r := gin.New()
	ls := &ListingService{}
	r.GET("/api/v1/kyc/status/:user_id", ls.GetKYCStatus)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/kyc/status/abc", nil))

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("with no engine configured the endpoint must refuse; got %d", w.Code)
	}
}
