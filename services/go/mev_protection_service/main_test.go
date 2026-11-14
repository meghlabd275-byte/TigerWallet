package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// The /v1/mev/block endpoint is a censorship surface and bundle submission
// spends relay reputation, so neither may be driven anonymously.
func TestRequireAPIKeyBlocksUnauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/v1/mev")
	g.Use(requireAPIKey("operator-key"))
	g.POST("/block", func(c *gin.Context) { c.Status(http.StatusOK) })

	for _, hdr := range []map[string]string{
		{},
		{"Authorization": ""},
		{"Authorization": "Bearer wrong"},
		{"X-API-Key": "wrong"},
	} {
		req := httptest.NewRequest(http.MethodPost, "/v1/mev/block", nil)
		for k, v := range hdr {
			req.Header.Set(k, v)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("headers %v: expected 401, got %d", hdr, w.Code)
		}
	}
}

func TestRequireAPIKeyAcceptsBothHeaderForms(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/v1/mev")
	g.Use(requireAPIKey("operator-key"))
	g.POST("/block", func(c *gin.Context) { c.Status(http.StatusOK) })

	for _, hdr := range []map[string]string{
		{"Authorization": "Bearer operator-key"},
		{"X-API-Key": "operator-key"},
	} {
		req := httptest.NewRequest(http.MethodPost, "/v1/mev/block", nil)
		for k, v := range hdr {
			req.Header.Set(k, v)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("headers %v: expected 200, got %d", hdr, w.Code)
		}
	}
}

// An unconfigured key must deny everything rather than allow everything.
func TestRequireAPIKeyFailsClosedWhenUnconfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/v1/mev")
	g.Use(requireAPIKey(""))
	g.POST("/block", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodPost, "/v1/mev/block", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when no key is configured, got %d", w.Code)
	}
}
