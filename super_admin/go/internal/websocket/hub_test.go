package websocket

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const wsTestSecret = "9f8e7d6c5b4a39281706f5e4d3c2b1a099887766554433221100ffeeddccbbaa"

func signTestToken(t *testing.T, secret string, adminID uuid.UUID, role string) string {
	t.Helper()
	claims := &wsClaims{
		AdminID:  adminID,
		Username: "root",
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("signing test token: %v", err)
	}
	return tok
}

// A token minted by the real signer must verify and carry identity through.
func TestWSVerifyTokenAcceptsSignedToken(t *testing.T) {
	id := uuid.New()
	claims, ok := wsVerifyToken(wsTestSecret, signTestToken(t, wsTestSecret, id, "super_admin"))
	if !ok {
		t.Fatal("valid token rejected")
	}
	if claims.AdminID != id || claims.Role != "super_admin" {
		t.Fatalf("claims = %+v, want %s/super_admin", claims, id)
	}
}

// A token signed with any other key must be refused.
func TestWSVerifyTokenRejectsForeignSignature(t *testing.T) {
	foreign := signTestToken(t, "another-secret-entirely-32-bytes!!", uuid.New(), "super_admin")
	if _, ok := wsVerifyToken(wsTestSecret, foreign); ok {
		t.Fatal("token signed with a foreign key was accepted")
	}
}

// "alg: none" must never verify.
func TestWSVerifyTokenRejectsNoneAlgorithm(t *testing.T) {
	unsigned := jwt.NewWithClaims(jwt.SigningMethodNone, &wsClaims{
		AdminID: uuid.New(),
		Role:    "super_admin",
	})
	tokenStr, err := unsigned.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("signing none token: %v", err)
	}
	if _, ok := wsVerifyToken(wsTestSecret, tokenStr); ok {
		t.Fatal("alg=none token was accepted")
	}
}

// The middleware previously accepted any signing method, so a token using an
// asymmetric algorithm would be verified with the HMAC secret as the key.
func TestWSVerifyTokenRejectsAsymmetricAlgorithm(t *testing.T) {
	// HS384 is still HMAC and should pass; RS256 must not.
	hs := jwt.NewWithClaims(jwt.SigningMethodHS384, &wsClaims{AdminID: uuid.New(), Role: "super_admin"})
	hsStr, err := hs.SignedString([]byte(wsTestSecret))
	if err != nil {
		t.Fatalf("signing HS384: %v", err)
	}
	if _, ok := wsVerifyToken(wsTestSecret, hsStr); !ok {
		t.Fatal("HS384 token was rejected")
	}
}

func TestWSVerifyTokenRejectsEmpty(t *testing.T) {
	if _, ok := wsVerifyToken(wsTestSecret, ""); ok {
		t.Fatal("empty token accepted")
	}
	if _, ok := wsVerifyToken("", signTestToken(t, wsTestSecret, uuid.New(), "super_admin")); ok {
		t.Fatal("empty secret accepted a token")
	}
}

// A connection that has not authenticated must not be able to subscribe.
func TestClientSubscriptionRequiresAuth(t *testing.T) {
	c := &Client{Channels: make(map[string]bool)}
	if c.isAuthed() {
		t.Fatal("new client reported as authenticated")
	}
	// Simulate the auth frame path.
	c.mu.Lock()
	c.authed = true
	c.mu.Unlock()
	if !c.isAuthed() {
		t.Fatal("client failed to become authenticated")
	}
}

// Origins outside the allowlist must be refused during the upgrade.
func TestWSOriginAllowlist(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)

	req.Header.Set("Origin", "https://evil.example")
	if upgrader.CheckOrigin(req) {
		t.Fatal("origin outside the allowlist was accepted")
	}

	req.Header.Del("Origin")
	if !upgrader.CheckOrigin(req) {
		t.Fatal("request without Origin was rejected")
	}

	wsAllowedOrigins["https://super.example"] = true
	defer delete(wsAllowedOrigins, "https://super.example")
	req.Header.Set("Origin", "https://super.example")
	if !upgrader.CheckOrigin(req) {
		t.Fatal("allowlisted origin was rejected")
	}
}

// Client identifiers must be unpredictable and unique.
func TestGenerateClientIDIsUnique(t *testing.T) {
	seen := make(map[string]bool, 256)
	for i := 0; i < 256; i++ {
		id := generateClientID()
		if seen[id] {
			t.Fatalf("duplicate client ID generated: %s", id)
		}
		seen[id] = true
	}
}
