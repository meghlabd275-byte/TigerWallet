package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The /master-wallet group is protected by JWT auth only, so each handler that
// takes a wallet id from the path must enforce ownership itself. These tests
// pin the gate on the handlers that previously read the id from a query
// parameter and performed no ownership check at all.
//
// A request without a resolvable identity must be refused before any database
// work happens, which is what the 400/403 responses below demonstrate.
func TestWalletScopedHandlersRejectUnidentifiedCaller(t *testing.T) {
	svc := &Service{}

	cases := []struct {
		name    string
		path    string
		handler func(*gin.Context)
	}{
		{"GetPolicies", "/api/v1/master-wallet/abc/policies", svc.GetPolicies},
		{"GetUsers", "/api/v1/master-wallet/abc/users", svc.GetUsers},
		{"GetAuditLogs", "/api/v1/master-wallet/abc/audit", svc.GetAuditLogs},
		{"GetVolumeAnalytics", "/api/v1/master-wallet/abc/analytics/volume", svc.GetVolumeAnalytics},
		{"GetTransactionAnalytics", "/api/v1/master-wallet/abc/analytics/transactions", svc.GetTransactionAnalytics},
		{"GetWalletAnalytics", "/api/v1/master-wallet/abc/analytics/wallets", svc.GetWalletAnalytics},
		{"GetTransactions", "/api/v1/master-wallet/abc/transactions", svc.GetTransactions},
		{"GetAutoSignPolicy", "/api/v1/master-wallet/abc/auto-sign-policy", svc.GetAutoSignPolicy},
		{"UpdatePolicy", "/api/v1/master-wallet/abc/policies/p1", svc.UpdatePolicy},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, tc.path, nil)
			// Route params are normally populated by the router; set them here
			// so requireMasterWalletAccess sees the same values it would in
			// production.
			c.Params = gin.Params{
				{Key: "id", Value: "abc"},
				{Key: "pid", Value: "p1"},
			}

			tc.handler(c)

			// No identity on the context, so the ownership check must fail.
			// Reaching the database instead would surface as a 500 from the
			// nil pool, and reaching it as an unauthenticated caller is exactly
			// the bug these handlers used to have.
			if w.Code != http.StatusForbidden && w.Code != http.StatusBadRequest {
				t.Fatalf("expected the ownership gate to refuse the request, got %d: %s",
					w.Code, w.Body.String())
			}
		})
	}
}
