package middleware

// Tests for API key authentication and the rate limiter. Both are security
// controls, so the important cases are that they fail closed when unconfigured
// and that they reject rather than allow.

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func runAPIKeyAuth(t *testing.T, header string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, engine := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	if header != "" {
		c.Request.Header.Set("X-API-Key", header)
	}

	engine.Use(APIKeyAuth())
	engine.GET("/api/v1/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	engine.HandleContext(c)

	return rec
}

func TestAPIKeyAuthFailsClosedWhenUnconfigured(t *testing.T) {
	t.Setenv("BILLING_API_KEY_HASHES", "")
	t.Setenv("BILLING_API_KEYS", "")

	rec := runAPIKeyAuth(t, "any-key-at-all")
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503 when no key is configured", rec.Code)
	}
}

func TestAPIKeyAuthRequiresAKey(t *testing.T) {
	t.Setenv("BILLING_API_KEYS", "secret-key")
	t.Setenv("BILLING_API_KEY_HASHES", "")

	rec := runAPIKeyAuth(t, "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 without a key", rec.Code)
	}
}

func TestAPIKeyAuthRejectsWrongKey(t *testing.T) {
	t.Setenv("BILLING_API_KEYS", "secret-key")
	t.Setenv("BILLING_API_KEY_HASHES", "")

	rec := runAPIKeyAuth(t, "wrong-key")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 for an unknown key", rec.Code)
	}
}

func TestAPIKeyAuthAcceptsConfiguredKey(t *testing.T) {
	t.Setenv("BILLING_API_KEYS", "secret-key")
	t.Setenv("BILLING_API_KEY_HASHES", "")

	rec := runAPIKeyAuth(t, "secret-key")
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 for the configured key (body %s)", rec.Code, rec.Body.String())
	}
}

func TestAPIKeyAuthAcceptsHashedConfiguration(t *testing.T) {
	sum := sha256.Sum256([]byte("hashed-key"))
	t.Setenv("BILLING_API_KEYS", "")
	t.Setenv("BILLING_API_KEY_HASHES", hex.EncodeToString(sum[:]))

	if rec := runAPIKeyAuth(t, "hashed-key"); rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 for a key matching the configured hash", rec.Code)
	}
	if rec := runAPIKeyAuth(t, "other-key"); rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 for a non-matching key", rec.Code)
	}
}

func TestAPIKeyAuthIgnoresMalformedHashEntries(t *testing.T) {
	t.Setenv("BILLING_API_KEYS", "")
	// "not-a-hash" is neither valid hex nor the right length.
	t.Setenv("BILLING_API_KEY_HASHES", "not-a-hash")

	// With every entry rejected there is no usable configuration, so the
	// endpoint must refuse rather than fall open.
	if rec := runAPIKeyAuth(t, "anything"); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503 when no valid hash is configured", rec.Code)
	}
}

func TestAPIKeyAuthAcceptsOneOfSeveralKeys(t *testing.T) {
	t.Setenv("BILLING_API_KEYS", "first-key,second-key,third-key")
	t.Setenv("BILLING_API_KEY_HASHES", "")

	for _, key := range []string{"first-key", "second-key", "third-key"} {
		if rec := runAPIKeyAuth(t, key); rec.Code != http.StatusOK {
			t.Errorf("key %q: status = %d, want 200", key, rec.Code)
		}
	}
}

func TestAPIKeyAuthTrimsWhitespace(t *testing.T) {
	t.Setenv("BILLING_API_KEYS", "  spaced-key  ")
	t.Setenv("BILLING_API_KEY_HASHES", "")

	if rec := runAPIKeyAuth(t, "spaced-key"); rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (surrounding whitespace should be trimmed)", rec.Code)
	}
}

func runRateLimiter(t *testing.T, maxRequests int, window time.Duration, requests int) []int {
	t.Helper()
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(RateLimiter(maxRequests, window))
	engine.GET("/api/v1/limited", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	codes := make([]int, 0, requests)
	for i := 0; i < requests; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/limited", nil)
		req.RemoteAddr = "203.0.113.7:12345"
		engine.ServeHTTP(rec, req)
		codes = append(codes, rec.Code)
	}
	return codes
}

func TestRateLimiterAllowsUpToLimitThenRejects(t *testing.T) {
	codes := runRateLimiter(t, 3, time.Minute, 5)

	for i, code := range codes {
		want := http.StatusOK
		if i >= 3 {
			want = http.StatusTooManyRequests
		}
		if code != want {
			t.Errorf("request %d: status = %d, want %d", i+1, code, want)
		}
	}
}

func TestRateLimiterResetsAfterWindow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(RateLimiter(1, 50*time.Millisecond))
	engine.GET("/api/v1/limited", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	call := func() int {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/limited", nil)
		req.RemoteAddr = "203.0.113.9:1234"
		engine.ServeHTTP(rec, req)
		return rec.Code
	}

	if code := call(); code != http.StatusOK {
		t.Fatalf("first call status = %d, want 200", code)
	}
	if code := call(); code != http.StatusTooManyRequests {
		t.Fatalf("second call status = %d, want 429", code)
	}

	time.Sleep(80 * time.Millisecond)

	if code := call(); code != http.StatusOK {
		t.Errorf("call after the window status = %d, want 200", code)
	}
}

func TestRateLimiterFailsClosedWhenMisconfigured(t *testing.T) {
	for _, tc := range []struct {
		name        string
		maxRequests int
		window      time.Duration
	}{
		{"zero limit", 0, time.Minute},
		{"negative limit", -1, time.Minute},
		{"zero window", 10, 0},
	} {
		gin.SetMode(gin.TestMode)
		engine := gin.New()
		engine.Use(RateLimiter(tc.maxRequests, tc.window))
		engine.GET("/api/v1/limited", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/limited", nil))

		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("%s: status = %d, want 503 rather than allowing unbounded traffic", tc.name, rec.Code)
		}
	}
}

func TestRateLimiterReportsRemaining(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(RateLimiter(2, time.Minute))
	engine.GET("/api/v1/limited", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/limited", nil)
	req.RemoteAddr = "203.0.113.11:1"
	engine.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-RateLimit-Limit"); got != "2" {
		t.Errorf("X-RateLimit-Limit = %q, want 2", got)
	}
	if got := rec.Header().Get("X-RateLimit-Remaining"); got != "1" {
		t.Errorf("X-RateLimit-Remaining = %q, want 1", got)
	}
}

func TestRateLimiterKeysByClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(RateLimiter(1, time.Minute))
	engine.GET("/api/v1/limited", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	call := func(addr string) int {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/limited", nil)
		req.RemoteAddr = addr
		engine.ServeHTTP(rec, req)
		return rec.Code
	}

	if code := call("203.0.113.20:1"); code != http.StatusOK {
		t.Fatalf("first client status = %d", code)
	}
	// A different client must not be affected by the first client's usage.
	if code := call("203.0.113.21:1"); code != http.StatusOK {
		t.Errorf("second client status = %d, want 200 (limits must be per client)", code)
	}
	if code := call("203.0.113.20:1"); code != http.StatusTooManyRequests {
		t.Errorf("first client's second call status = %d, want 429", code)
	}
}

func TestValidateTokenRejectsAlgNone(t *testing.T) {
	// A token signed with "none" must not be accepted. The key function rejects
	// any method that is not HMAC.
	noneToken := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJ1c2VyX2lkIjoiMDAwMDAwMDAtMDAwMC0wMDAwLTAwMDAtMDAwMDAwMDAwMDAwIn0."

	if _, err := ValidateToken(noneToken, "any-secret"); err == nil {
		t.Fatal("a token with alg=none was accepted")
	}
}

func TestValidateTokenRejectsWrongSecret(t *testing.T) {
	// A well-formed but unrelated token must not validate against this secret.
	// The signature is deliberately not computed with the server secret.
	parts := strings.Split("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMDAwMDAwMDAtMDAwMC0wMDAwLTAwMDAtMDAwMDAwMDAwMDAwIn0.AAAA", ".")
	if len(parts) != 3 {
		t.Fatal("test fixture is malformed")
	}

	if _, err := ValidateToken(strings.Join(parts, "."), "the-real-secret"); err == nil {
		t.Fatal("a token with an invalid signature was accepted")
	}
}
