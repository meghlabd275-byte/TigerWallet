package wlgate

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// runAuth drives a single request through JWTAuth and reports whether the
// middleware let it reach the handler.
func runAuth(t *testing.T, token string) (int, bool) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	reached := false
	r := gin.New()
	r.Use(JWTAuth("server-secret"))
	r.GET("/probe", func(c *gin.Context) {
		reached = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code, reached
}

// TestJWTAuthAcceptsValidToken is the control: a well-formed HS256 token with an
// unexpired exp must be accepted.
func TestJWTAuthAcceptsValidToken(t *testing.T) {
	tok, err := IssueJWT("server-secret", uuid.New(), "user@example.com", uuid.New(), []string{"wallet"}, time.Hour)
	if err != nil {
		t.Fatalf("IssueJWT: %v", err)
	}
	code, reached := runAuth(t, tok)
	if code != http.StatusOK || !reached {
		t.Fatalf("valid token rejected: status=%d reached=%v", code, reached)
	}
}

// TestJWTAuthRejectsWrongSecret ensures a token signed with a different key is
// refused rather than trusted.
func TestJWTAuthRejectsWrongSecret(t *testing.T) {
	tok, err := IssueJWT("attacker-secret", uuid.New(), "user@example.com", uuid.New(), []string{"wallet"}, time.Hour)
	if err != nil {
		t.Fatalf("IssueJWT: %v", err)
	}
	if code, reached := runAuth(t, tok); reached || code != http.StatusUnauthorized {
		t.Fatalf("token signed with wrong secret accepted: status=%d reached=%v", code, reached)
	}
}

// TestJWTAuthRejectsExpiredToken covers the expiry claim.
func TestJWTAuthRejectsExpiredToken(t *testing.T) {
	tok, err := IssueJWT("server-secret", uuid.New(), "user@example.com", uuid.New(), []string{"wallet"}, -time.Hour)
	if err != nil {
		t.Fatalf("IssueJWT: %v", err)
	}
	if code, reached := runAuth(t, tok); reached || code != http.StatusUnauthorized {
		t.Fatalf("expired token accepted: status=%d reached=%v", code, reached)
	}
}

// TestJWTAuthRejectsAlgNone covers the classic algorithm-confusion attack: a
// token that declares alg=none and carries no signature must never be trusted,
// even though its claims are well formed.
func TestJWTAuthRejectsAlgNone(t *testing.T) {
	unsigned := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{
		UserID: uuid.New(),
		Scopes: []string{"wl_client"},
	})
	tok, err := unsigned.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign none: %v", err)
	}
	if code, reached := runAuth(t, tok); reached || code != http.StatusUnauthorized {
		t.Fatalf("alg=none token accepted: status=%d reached=%v", code, reached)
	}
}

// TestJWTAuthRejectsNonHMACAlgorithm pins the algorithm allowlist: only HS256 is
// honoured, so a token declaring any other algorithm is refused before its
// signature is even considered.
func TestJWTAuthRejectsNonHMACAlgorithm(t *testing.T) {
	// HMAC variants can be genuinely signed with our secret, so these are real
	// signature-valid tokens that must still be refused on the alg allowlist.
	for _, alg := range []string{"HS384", "HS512"} {
		tok := jwt.NewWithClaims(jwt.GetSigningMethod(alg), Claims{
			UserID: uuid.New(),
			Scopes: []string{"wl_client"},
		})
		signed, err := tok.SignedString([]byte("server-secret"))
		if err != nil {
			t.Fatalf("sign %s: %v", alg, err)
		}
		if code, reached := runAuth(t, signed); reached || code != http.StatusUnauthorized {
			t.Fatalf("alg=%s token accepted: status=%d reached=%v", alg, code, reached)
		}
	}

	// Asymmetric algs cannot be signed with a byte secret, so hand-craft the
	// token to prove the keyfunc rejects the algorithm outright.
	for _, alg := range []string{"RS256", "ES256", "none"} {
		header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"` + alg + `","typ":"JWT"}`))
		body := base64.RawURLEncoding.EncodeToString([]byte(`{"user_id":"` + uuid.NewString() + `","scopes":["wl_client"]}`))
		forged := header + "." + body + ".AAAA"
		if code, reached := runAuth(t, forged); reached || code != http.StatusUnauthorized {
			t.Fatalf("alg=%s token accepted: status=%d reached=%v", alg, code, reached)
		}
	}
}

// TestJWTAuthRequiresBearerHeader covers the header-shape checks.
func TestJWTAuthRequiresBearerHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, hdr := range []string{"", "token", "Basic abc", "Bearer", "Bearer a b"} {
		r := gin.New()
		r.Use(JWTAuth("server-secret"))
		r.GET("/probe", func(c *gin.Context) { c.Status(http.StatusOK) })
		req := httptest.NewRequest(http.MethodGet, "/probe", nil)
		if hdr != "" {
			req.Header.Set("Authorization", hdr)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("header %q should be rejected, got %d", hdr, w.Code)
		}
	}
}
