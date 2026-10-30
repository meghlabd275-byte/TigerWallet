package main

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// A connection key on its own must never authorise a request. Previously
// Heartbeat only checked the token when the header happened to be present,
// so omitting it bypassed the check entirely.
func TestRequireSessionRejectsMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldRedis := redis
	redis = nil
	defer func() { redis = oldRedis }()

	r := gin.New()
	var status int
	r.POST("/heartbeat", func(c *gin.Context) {
		if _, ok := requireSession(c, "conn_abc"); !ok {
			status = c.Writer.Status()
			return
		}
		status = http.StatusOK
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/heartbeat", strings.NewReader("{}"))
	r.ServeHTTP(w, req)

	if status != http.StatusUnauthorized {
		t.Fatalf("missing token returned %d, want 401", status)
	}
}

// An empty header value is still a missing credential.
func TestRequireSessionRejectsBlankToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldRedis := redis
	redis = nil
	defer func() { redis = oldRedis }()

	r := gin.New()
	var status int
	r.POST("/heartbeat", func(c *gin.Context) {
		if _, ok := requireSession(c, "conn_abc"); !ok {
			status = c.Writer.Status()
			return
		}
		status = http.StatusOK
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/heartbeat", strings.NewReader("{}"))
	req.Header.Set("X-Session-Token", "   ")
	r.ServeHTTP(w, req)

	if status != http.StatusUnauthorized {
		t.Fatalf("blank token returned %d, want 401", status)
	}
}

// The rate limiter previously compared requests against 0 after incrementing,
// so it allowed unlimited traffic. It must now enforce its limit.
func TestRateLimiterEnforcesLimit(t *testing.T) {
	rl := newRateLimiter(3, time.Minute)

	for i := 0; i < 3; i++ {
		if !rl.Allow() {
			t.Fatalf("request %d unexpectedly denied", i+1)
		}
	}
	if rl.Allow() {
		t.Fatal("4th request allowed, limit was 3")
	}
}

// The window must reset once it elapses.
func TestRateLimiterWindowResets(t *testing.T) {
	rl := newRateLimiter(1, 10*time.Millisecond)

	if !rl.Allow() {
		t.Fatal("first request denied")
	}
	if rl.Allow() {
		t.Fatal("second request allowed within window")
	}
	time.Sleep(20 * time.Millisecond)
	if !rl.Allow() {
		t.Fatal("request denied after window elapsed")
	}
}

// Stored API keys must be salted and must not be recoverable by equality
// lookup, so the same key produces a different digest each time.
func TestAPIKeyHashIsSalted(t *testing.T) {
	a, err := APIKeyHash("tw_live_abcdef0123456789")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	b, err := APIKeyHash("tw_live_abcdef0123456789")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if a == b {
		t.Fatal("identical digests for the same key: hash is not salted")
	}
	if strings.Contains(a, "tw_live_abcdef0123456789") {
		t.Fatal("digest contains the plaintext key")
	}
}

// The connection record is serialised to Redis and returned by GetConnection,
// so it must not carry the session token.
func TestConnectionStateHasNoSessionToken(t *testing.T) {
	typ := reflect.TypeOf(ConnectionState{})
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if strings.Contains(strings.ToLower(f.Name), "sessiontoken") {
			t.Fatalf("ConnectionState still exposes a session token field: %s", f.Name)
		}
		jsonTag := strings.ToLower(f.Tag.Get("json"))
		if strings.Contains(jsonTag, "session_token") {
			t.Fatalf("ConnectionState still serialises a session token: %s", f.Name)
		}
	}
}
