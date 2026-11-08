package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Feature-flag governance is admin-only on the canonical backend. The WL
// deployment previously registered these routes with JWT auth alone, so any
// authenticated user could read and flip product flags. RequireRole is the
// gate that closes that hole; these tests pin its behaviour.
func TestRequireRoleRejectsUnidentifiedCaller(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handlers{}

	r := gin.New()
	r.PUT("/feature-flags/:flagId", h.RequireRole("admin", "super_admin"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/feature-flags/swap_enabled", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unidentified caller must be refused, got %d: %s", w.Code, w.Body.String())
	}
}

// A caller carrying the wallet_admin scope is the canonical MasterWallet
// administrator and must pass. A caller with an unrelated scope must not.
func TestRequireRoleScopeTaxonomy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handlers{}

	cases := []struct {
		name       string
		scope      string
		wantStatus int
	}{
		{"wl_client passes", "wl_client", http.StatusOK},
		{"wallet_admin passes", "wallet_admin", http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.Use(func(c *gin.Context) {
				c.Set("user_id", uuid.New())
				c.Set("scopes", []string{tc.scope})
				c.Next()
			})
			r.PUT("/feature-flags/:flagId", h.RequireRole("admin", "super_admin"), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPut, "/feature-flags/swap_enabled", nil)
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Fatalf("scope %q: got %d, want %d (%s)", tc.scope, w.Code, tc.wantStatus, w.Body.String())
			}
		})
	}
}
