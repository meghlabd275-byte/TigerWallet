package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/tigerwallet/wl-project-party/internal/config"
	"github.com/tigerwallet/wl-project-party/internal/handlers"
	"github.com/tigerwallet/wl-project-party/internal/store"
	"github.com/tigerwallet/wl-shared/wlgate"
)

// TestBuildRouterNoConflict exercises the EXACT production route tree. Any
// httprouter conflict (duplicate method+path, or two different param names at
// the same tree position) panics inside buildRouter — failing the test. No DB
// or live license is required: route registration runs before any DB call and
// the dead gate only short-circuits protected handlers at request time.
func TestBuildRouterNoConflict(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	cfg := &config.Config{JWTSecret: "test", Product: "project_party", WLClientID: "00000000-0000-0000-0000-000000000000"}
	gate := wlgate.New() // dead gate (fail-closed) — fine for registration
	svc := handlers.New(cfg, &store.Store{}, gate)

	r := buildRouter(cfg, svc, gate)
	if r == nil {
		t.Fatal("buildRouter returned nil router")
	}

	// Count registered routes — must meet the canonical parity target (>=50).
	routes := r.Routes()
	if len(routes) < 50 {
		t.Fatalf("expected >= 50 routes, got %d", len(routes))
	}

	// Dispatch a few representative requests. The dead license gate
	// fail-closes protected routes (503) and public discovery routes hit the
	// store (nil DB) returning a 5xx — either way no panic, proving the tree
	// resolves these paths without conflict.
	probes := []struct{ method, path string }{
		{http.MethodGet, "/api/v1/coins"},
		{http.MethodGet, "/api/v1/search?q=abc"},
		{http.MethodGet, "/api/v1/featured"},
		{http.MethodGet, "/api/v1/trending"},
		{http.MethodGet, "/api/v1/market"},
		{http.MethodGet, "/api/v1/status/00000000-0000-0000-0000-000000000000"},
		{http.MethodGet, "/api/v1/00000000-0000-0000-0000-000000000000"},
		{http.MethodGet, "/api/v1/orders"},
		{http.MethodGet, "/api/v1/liquidity"},
		{http.MethodGet, "/api/v1/audit/00000000-0000-0000-0000-000000000000"},
		{http.MethodGet, "/api/v1/kyc/00000000-0000-0000-0000-000000000000"},
		{http.MethodPost, "/api/v1/tokens/00000000-0000-0000-0000-000000000000/submit"},
	}
	for _, p := range probes {
		req := httptest.NewRequest(p.method, p.path, nil)
		w := httptest.NewRecorder()
		// Should never panic; status only asserts the route matched (404 would
		// mean a routing conflict swallowed the path).
		r.ServeHTTP(w, req)
		if w.Code == http.StatusNotFound {
			t.Errorf("%s %s returned 404 (route not matched)", p.method, p.path)
		}
	}
}

