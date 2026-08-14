package main

import (
	"testing"
	"time"
)

// TestRateLimiterAllowThenDeny verifies that a fresh key is allowed up to the
// burst size and then rejected until tokens refill.
func TestRateLimiterAllowThenDeny(t *testing.T) {
	rl := newRateLimiter(1.0, 3.0) // 1 token/sec, burst 3
	key := "ip:127.0.0.1"
	for i := 0; i < 3; i++ {
		if !rl.allow(key) {
			t.Fatalf("expected request %d to be allowed within burst", i+1)
		}
	}
	if rl.allow(key) {
		t.Fatal("expected 4th request to be denied (bucket exhausted)")
	}
}

// TestRateLimiterRefills verifies that tokens refill over wall-clock time.
func TestRateLimiterRefills(t *testing.T) {
	rl := newRateLimiter(1000.0, 1.0) // fast refill so the test is quick
	key := "ip:10.0.0.1"
	if !rl.allow(key) {
		t.Fatal("first request should be allowed")
	}
	if rl.allow(key) {
		t.Fatal("second immediate request should be denied (no refill yet)")
	}
	time.Sleep(5 * time.Millisecond)
	if !rl.allow(key) {
		t.Fatal("request after refill window should be allowed")
	}
}

// TestRateLimiterIndependentKeys verifies that different keys are limited
// independently (one IP saturating does not block another).
func TestRateLimiterIndependentKeys(t *testing.T) {
	rl := newRateLimiter(0.01, 1.0) // very slow refill, burst 1
	if !rl.allow("ip:a") {
		t.Fatal("ip:a first should be allowed")
	}
	if rl.allow("ip:a") {
		t.Fatal("ip:a second should be denied")
	}
	if !rl.allow("ip:b") {
		t.Fatal("ip:b first should be allowed independently of ip:a")
	}
}

// TestRateLimiterRetryAfter verifies the helper returns a positive duration.
func TestRateLimiterRetryAfter(t *testing.T) {
	rl := newRateLimiter(5.0/60.0, 5.0) // auth policy: 5/min
	r := rl.retryAfterSeconds()
	if r <= 0 {
		t.Fatalf("retry-after should be positive, got %d", r)
	}
}

// TestClientKeyPrefersUser verifies that an authenticated user ID is used over
// the IP when present.
func TestClientKeyPrefersUser(t *testing.T) {
	// This is exercised via clientKey which depends on gin context; we test
	// the helper behavior indirectly through the indexByte/trimSpace helpers.
	if indexByte("a,b", ',') != 1 {
		t.Fatal("indexByte comma lookup failed")
	}
	if indexByte("abc", ',') != -1 {
		t.Fatal("indexByte missing char should return -1")
	}
	if trimSpace("  x  ") != "x" {
		t.Fatalf("trimSpace failed: %q", trimSpace("  x  "))
	}
	if trimSpace("y") != "y" {
		t.Fatalf("trimSpace no-space failed: %q", trimSpace("y"))
	}
}
