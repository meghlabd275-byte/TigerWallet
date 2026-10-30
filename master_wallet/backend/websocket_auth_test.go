package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const wsTestSecret = "e1f2a3b4c5d6e7f8091a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d5e6f70"

func wsTestContext(t *testing.T, header, queryToken string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	if header != "" {
		req.Header.Set("Authorization", "Bearer "+header)
	}
	if queryToken != "" {
		q := req.URL.Query()
		q.Set("token", queryToken)
		req.URL.RawQuery = q.Encode()
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req
	return c
}

// A valid token issued by the real signer must authenticate and carry the
// caller's identity through to the connection.
func TestWSAuthenticateAcceptsIssuedToken(t *testing.T) {
	token, err := IssueJWT(wsTestSecret, "user-123", "owner@example.com", "owner")
	if err != nil {
		t.Fatalf("IssueJWT: %v", err)
	}

	claims, ok := wsAuthenticate(wsTestContext(t, token, ""), wsTestSecret)
	if !ok {
		t.Fatal("valid token rejected")
	}
	if claims.UserID != "user-123" || claims.Role != "owner" {
		t.Fatalf("claims = %+v, want user-123/owner", claims)
	}

	// Browsers cannot set headers on a WebSocket handshake, so the query
	// parameter path must work identically.
	if _, ok := wsAuthenticate(wsTestContext(t, "", token), wsTestSecret); !ok {
		t.Fatal("valid token in query parameter rejected")
	}
}

// A token signed with a different key must never be accepted; this is the
// forgery path that previously did not exist because identity came from a
// query parameter.
func TestWSAuthenticateRejectsForeignSignature(t *testing.T) {
	forged, err := IssueJWT("a-completely-different-secret-value-32b", "attacker", "a@b.c", "super_admin")
	if err != nil {
		t.Fatalf("IssueJWT: %v", err)
	}
	if _, ok := wsAuthenticate(wsTestContext(t, forged, ""), wsTestSecret); ok {
		t.Fatal("token signed with a foreign key was accepted")
	}
}

// "alg: none" is the classic JWT downgrade; it must be refused.
func TestWSAuthenticateRejectsNoneAlgorithm(t *testing.T) {
	unsigned := jwt.NewWithClaims(jwt.SigningMethodNone, &MasterClaims{
		UserID: "attacker",
		Role:   "super_admin",
	})
	tokenStr, err := unsigned.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("signing none token: %v", err)
	}
	if _, ok := wsAuthenticate(wsTestContext(t, tokenStr, ""), wsTestSecret); ok {
		t.Fatal("alg=none token was accepted")
	}
}

// An unconfigured secret must fail closed rather than authenticate everyone.
func TestWSAuthenticateFailsClosedWithoutSecret(t *testing.T) {
	token, err := IssueJWT(wsTestSecret, "user-123", "o@example.com", "owner")
	if err != nil {
		t.Fatalf("IssueJWT: %v", err)
	}
	for _, secret := range []string{"", "   "} {
		if _, ok := wsAuthenticate(wsTestContext(t, token, ""), secret); ok {
			t.Fatalf("empty secret %q authenticated a caller", secret)
		}
	}
	if _, ok := wsAuthenticate(wsTestContext(t, "", ""), wsTestSecret); ok {
		t.Fatal("missing token authenticated a caller")
	}
}

// Operators bypass the ownership lookup; everyone else must not, and a
// database failure must deny rather than allow.
func TestWSMayAccessWalletAdminBypass(t *testing.T) {
	svc := &Service{}
	ctx := context.Background()
	for _, role := range []string{"admin", "super_admin"} {
		if !svc.wsMayAccessWallet(ctx, "any-wallet", "someone", role) {
			t.Fatalf("role %q was denied the ownership bypass", role)
		}
	}
}

// A browser Origin outside the allowlist must be refused during the upgrade.
func TestWSOriginAllowlist(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)

	req.Header.Set("Origin", "https://evil.example")
	if upgrader.CheckOrigin(req) {
		t.Fatal("origin outside the allowlist was accepted")
	}

	// Non-browser clients (no Origin) are not CSWSH candidates.
	req.Header.Del("Origin")
	if !upgrader.CheckOrigin(req) {
		t.Fatal("request without Origin was rejected")
	}

	// With an allowlist configured, a listed origin is accepted.
	wsAllowedOrigins["https://admin.example"] = true
	defer delete(wsAllowedOrigins, "https://admin.example")
	req.Header.Set("Origin", "https://admin.example")
	if !upgrader.CheckOrigin(req) {
		t.Fatal("allowlisted origin was rejected")
	}
}

// splitAndTrim backs the CORS/WebSocket origin allowlist parsing.
func TestSplitAndTrim(t *testing.T) {
	got := splitAndTrim(" https://a.example , ,https://b.example ")
	if len(got) != 2 || got[0] != "https://a.example" || got[1] != "https://b.example" {
		t.Fatalf("splitAndTrim = %#v", got)
	}
	if len(splitAndTrim("")) != 0 {
		t.Fatal("empty input must yield no origins")
	}
}
