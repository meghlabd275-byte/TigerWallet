package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func newAuthRouter(secret string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	svc := &NFTMarketplaceService{jwtSecret: []byte(secret)}

	r := gin.New()
	r.GET("/whoami", svc.AuthMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"address": c.GetString("address")})
	})
	return r
}

func signToken(t *testing.T, secret string, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func TestAuthMiddlewareRejectsUnconfiguredSecret(t *testing.T) {
	r := newAuthRouter("")

	req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
	req.Header.Set("Authorization", "Bearer anything")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 when no secret is configured", w.Code)
	}
}

func TestAuthMiddlewareRejectsMissingHeader(t *testing.T) {
	r := newAuthRouter("test-secret")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/whoami", nil))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestAuthMiddlewareAcceptsValidToken(t *testing.T) {
	const secret = "test-secret"
	const addr = "0xAbCdEf0123456789aBcDeF0123456789AbCdEf01"

	r := newAuthRouter(secret)
	token := signToken(t, secret, jwt.MapClaims{
		"sub": addr,
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body.String())
	}
	// Addresses are normalised to lowercase.
	if want := "0xabcdef0123456789abcdef0123456789abcdef01"; w.Body.String() != `{"address":"`+want+`"}` {
		t.Fatalf("body = %s, want the normalised address %s", w.Body.String(), want)
	}
}

// TestAuthMiddlewareRejectsWrongSecret is the core signature check.
func TestAuthMiddlewareRejectsWrongSecret(t *testing.T) {
	r := newAuthRouter("expected-secret")
	token := signToken(t, "attacker-secret", jwt.MapClaims{
		"sub": "0x1111111111111111111111111111111111111111",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 for a token signed with the wrong key", w.Code)
	}
}

// TestAuthMiddlewareRejectsNonHMACAlgorithm guards the algorithm-confusion hole
// where a token could arrive signed with "none" or an asymmetric method.
func TestAuthMiddlewareRejectsNonHMACAlgorithm(t *testing.T) {
	const secret = "test-secret"
	r := newAuthRouter(secret)

	token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"sub": "0x1111111111111111111111111111111111111111",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	unsigned, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("build unsigned token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
	req.Header.Set("Authorization", "Bearer "+unsigned)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 for an alg=none token", w.Code)
	}
}

// TestAuthMiddlewareRejectsNonAddressSubject covers the previous behaviour that
// synthesised an address out of any subject string, padding it to 40 chars.
func TestAuthMiddlewareRejectsNonAddressSubject(t *testing.T) {
	const secret = "test-secret"
	r := newAuthRouter(secret)

	for _, sub := range []string{"", "alice", "0x1234", "0x" + string(make([]byte, 40))} {
		token := signToken(t, secret, jwt.MapClaims{
			"sub": sub,
			"exp": time.Now().Add(time.Hour).Unix(),
		})
		req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("subject %q: status = %d, want 401", sub, w.Code)
		}
	}
}

func TestNormaliseEthAddress(t *testing.T) {
	got, ok := normaliseEthAddress("0xAbC0000000000000000000000000000000000001")
	if !ok {
		t.Fatal("valid address rejected")
	}
	if got != "0xabc0000000000000000000000000000000000001" {
		t.Fatalf("normalised = %s", got)
	}

	for _, bad := range []string{"", "0x", "abcdef", "0xzz00000000000000000000000000000000000000", "0x123"} {
		if _, ok := normaliseEthAddress(bad); ok {
			t.Fatalf("normaliseEthAddress(%q) should have failed", bad)
		}
	}
}

func TestParseTraitFilter(t *testing.T) {
	terms, err := parseTraitFilter("Background:Blue, Eyes:Green")
	if err != nil {
		t.Fatalf("parseTraitFilter: %v", err)
	}
	if len(terms) != 2 {
		t.Fatalf("got %d terms, want 2", len(terms))
	}
	if terms[0] != `[{"trait_type":"Background","value":"Blue"}]` {
		t.Fatalf("first term = %s", terms[0])
	}

	if got, err := parseTraitFilter("  "); err != nil || got != nil {
		t.Fatalf("blank filter = %v err=%v, want nil/nil", got, err)
	}

	for _, bad := range []string{"Background", ":Blue", "Background:"} {
		if _, err := parseTraitFilter(bad); err == nil {
			t.Fatalf("parseTraitFilter(%q) should have failed", bad)
		}
	}

	// A trailing comma is tolerated: empty pairs are skipped rather than
	// treated as malformed input.
	terms, err = parseTraitFilter("Background:Blue,")
	if err != nil {
		t.Fatalf("trailing comma should be tolerated, got %v", err)
	}
	if len(terms) != 1 {
		t.Fatalf("trailing comma produced %d terms, want 1", len(terms))
	}
}
