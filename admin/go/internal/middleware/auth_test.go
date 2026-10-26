package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func newRateLimitRouter(limiter *RateLimiter, perMinute int) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(limiter.Middleware(perMinute))
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
	return r
}

func TestRateLimiterBlocksAfterLimit(t *testing.T) {
	r := newRateLimitRouter(NewRateLimiter(), 3)

	for i := 1; i <= 3; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ping", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: got status %d, want 200", i, w.Code)
		}
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ping", nil))
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("request 4: got status %d, want 429", w.Code)
	}
	if w.Header().Get("Retry-After") == "" {
		t.Fatal("a 429 response must include Retry-After")
	}
}

// TestRateLimiterWindowResets guards the regression where counters were never
// reset, permanently locking a client out after N lifetime requests.
func TestRateLimiterWindowResets(t *testing.T) {
	limiter := NewRateLimiter()
	r := newRateLimitRouter(limiter, 2)

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ping", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: got status %d, want 200", i+1, w.Code)
		}
	}

	// Expire the window for this client directly, then confirm the next request
	// is served rather than rejected.
	limiter.mu.Lock()
	for k := range limiter.resetAt {
		limiter.resetAt[k] = limiter.resetAt[k].Add(-2 * 60 * 1e9)
	}
	limiter.mu.Unlock()

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ping", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("after window expiry got status %d, want 200", w.Code)
	}
}

func TestRateLimiterPrunesExpiredWindows(t *testing.T) {
	limiter := NewRateLimiter()
	now := time.Now()

	// Seed more than the prune threshold: half expired, half live.
	limiter.mu.Lock()
	for i := 0; i < 1500; i++ {
		key := "rate_limit:10.0." + strconv.Itoa(i/256) + "." + strconv.Itoa(i%256)
		limiter.requests[key] = 1
		if i%2 == 0 {
			limiter.resetAt[key] = now.Add(-time.Minute)
		} else {
			limiter.resetAt[key] = now.Add(time.Minute)
		}
	}
	before := len(limiter.resetAt)
	limiter.pruneLocked(now)
	after := len(limiter.resetAt)

	live := 0
	for _, reset := range limiter.resetAt {
		if reset.After(now) {
			live++
		}
	}
	limiter.mu.Unlock()

	if before != 1500 {
		t.Fatalf("seeded %d windows, want 1500", before)
	}
	if after != 750 {
		t.Fatalf("after prune %d windows remain, want only the 750 live ones", after)
	}
	if live != after {
		t.Fatalf("pruned set contains %d expired windows", after-live)
	}
}

func TestCORSMiddlewareAllowsListedOriginWithCredentials(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://admin.tigerwallet.io, https://ops.tigerwallet.io")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORSMiddleware())
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://admin.tigerwallet.io")
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://admin.tigerwallet.io" {
		t.Fatalf("Allow-Origin = %q, want the requesting origin", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("Allow-Credentials = %q, want true for a listed origin", got)
	}
}

// TestCORSMiddlewareNeverReflectsUnknownOrigin is the regression test for the
// previous wildcard-with-credentials configuration.
func TestCORSMiddlewareNeverReflectsUnknownOrigin(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://admin.tigerwallet.io")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORSMiddleware())
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://evil.example")
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Allow-Origin = %q, want empty for an unlisted origin", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("Allow-Credentials = %q, want empty for an unlisted origin", got)
	}
}

func TestCORSMiddlewareWildcardNeverPairedWithCredentials(t *testing.T) {
	os.Unsetenv("CORS_ALLOWED_ORIGINS")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORSMiddleware())
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://anything.example")
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got == "*" {
		t.Fatal("wildcard origin must not be emitted alongside credentials")
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("Allow-Credentials = %q, want empty when no allowlist is configured", got)
	}
}

func TestCORSMiddlewarePreflight(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://admin.tigerwallet.io")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORSMiddleware())
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	req.Header.Set("Origin", "https://admin.tigerwallet.io")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204", w.Code)
	}
}

func TestParseAllowedOrigins(t *testing.T) {
	got := parseAllowedOrigins(" https://a.example , ,https://b.example ")

	if len(got) != 2 {
		t.Fatalf("parsed %d origins, want 2", len(got))
	}
	if !got["https://a.example"] || !got["https://b.example"] {
		t.Fatalf("parsed set missing entries: %v", got)
	}
}
