package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "mpc-test-secret"

func mintToken(t *testing.T, sub, role string) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": sub, "role": role, "exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, err := tok.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return signed
}

func mpcRequest(t *testing.T, h http.HandlerFunc, method, path, token string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h(w, req)
	return w
}

func createWalletAs(t *testing.T, owner string) *mpcWallet {
	t.Helper()
	h := secure(testSecret, nil, handleCreate)
	w := mpcRequest(t, h, http.MethodPost, "/api/v1/mpc/create", mintToken(t, owner, "user"),
		map[string]uint32{"threshold": 2, "totalShards": 3})
	if w.Code != http.StatusCreated {
		t.Fatalf("create returned %d: %s", w.Code, w.Body.String())
	}
	var wallet mpcWallet
	if err := json.Unmarshal(w.Body.Bytes(), &wallet); err != nil {
		t.Fatalf("decode wallet: %v", err)
	}
	return &wallet
}

// TestMPCRequiresAuthentication is the core regression guard: the service signs
// and exports key material, so no route may answer an anonymous caller.
func TestMPCRequiresAuthentication(t *testing.T) {
	cases := []struct {
		name    string
		handler func(http.ResponseWriter, *http.Request, caller)
		method  string
		path    string
	}{
		{"create", handleCreate, http.MethodPost, "/api/v1/mpc/create"},
		{"sign", handleSign, http.MethodPost, "/api/v1/mpc/sign"},
		{"wallet", handleGetWallet, http.MethodGet, "/api/v1/mpc/wallet/key_abc"},
		{"export", handleExportShares, http.MethodPost, "/api/v1/mpc/keygen/export"},
		{"recover", handleRecoverWallet, http.MethodPost, "/api/v1/mpc/keygen/recover"},
	}

	for _, tc := range cases {
		h := secure(testSecret, nil, tc.handler)
		if w := mpcRequest(t, h, tc.method, tc.path, "", nil); w.Code != http.StatusUnauthorized {
			t.Fatalf("%s without a token returned %d, want 401", tc.name, w.Code)
		}
	}
}

// TestMPCRejectsForgedAndExpiredTokens proves token validation is real.
func TestMPCRejectsForgedAndExpiredTokens(t *testing.T) {
	h := secure(testSecret, nil, handleGetWallet)

	forged := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "attacker", "exp": time.Now().Add(time.Hour).Unix(),
	})
	signedForged, _ := forged.SignedString([]byte("the-wrong-secret"))
	if w := mpcRequest(t, h, http.MethodGet, "/api/v1/mpc/wallet/key_abc", signedForged, nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("a token signed with the wrong secret returned %d, want 401", w.Code)
	}

	expired := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user", "exp": time.Now().Add(-time.Hour).Unix(),
	})
	signedExpired, _ := expired.SignedString([]byte(testSecret))
	if w := mpcRequest(t, h, http.MethodGet, "/api/v1/mpc/wallet/key_abc", signedExpired, nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("an expired token returned %d, want 401", w.Code)
	}

	// alg=none must not be accepted.
	unsigned := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"sub": "user"})
	noneToken, _ := unsigned.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if w := mpcRequest(t, h, http.MethodGet, "/api/v1/mpc/wallet/key_abc", noneToken, nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("an alg=none token returned %d, want 401", w.Code)
	}
}

// TestMPCSignRequiresOwnership proves one subject cannot sign with another
// subject's wallet. This is the highest-impact gap: signing authority is
// spendable.
func TestMPCSignRequiresOwnership(t *testing.T) {
	wallet := createWalletAs(t, "owner-a")

	h := secure(testSecret, nil, handleSign)
	body := map[string]string{
		"keyId":       wallet.KeyID,
		"messageHash": strings.Repeat("ab", 32),
	}

	// A different subject must not be able to sign.
	if w := mpcRequest(t, h, http.MethodPost, "/api/v1/mpc/sign", mintToken(t, "attacker", "user"), body); w.Code != http.StatusNotFound {
		t.Fatalf("a non-owner signed with the wallet: %d %s", w.Code, w.Body.String())
	}

	// The owner may.
	w := mpcRequest(t, h, http.MethodPost, "/api/v1/mpc/sign", mintToken(t, "owner-a", "user"), body)
	if w.Code != http.StatusOK {
		t.Fatalf("the owner could not sign: %d %s", w.Code, w.Body.String())
	}
	var resp signResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Signature) != 65 {
		t.Fatalf("signature length = %d, want 65", len(resp.Signature))
	}
}

// TestMPCGetWalletRequiresOwnership proves wallet metadata is not readable by
// other subjects, and that the answer does not confirm the wallet exists.
func TestMPCGetWalletRequiresOwnership(t *testing.T) {
	wallet := createWalletAs(t, "owner-b")
	h := secure(testSecret, nil, handleGetWallet)

	if w := mpcRequest(t, h, http.MethodGet, "/api/v1/mpc/wallet/"+wallet.KeyID, mintToken(t, "attacker", "user"), nil); w.Code != http.StatusNotFound {
		t.Fatalf("a non-owner read the wallet: %d", w.Code)
	}
	if w := mpcRequest(t, h, http.MethodGet, "/api/v1/mpc/wallet/"+wallet.KeyID, mintToken(t, "owner-b", "user"), nil); w.Code != http.StatusOK {
		t.Fatalf("the owner could not read the wallet: %d", w.Code)
	}
}

// TestMPCExportRequiresOwnership proves encrypted shares are owner-only.
func TestMPCExportRequiresOwnership(t *testing.T) {
	wallet := createWalletAs(t, "owner-c")
	h := secure(testSecret, nil, handleExportShares)
	body := map[string]interface{}{"keyId": wallet.KeyID, "passphrase": "a-strong-passphrase"}

	if w := mpcRequest(t, h, http.MethodPost, "/api/v1/mpc/keygen/export", mintToken(t, "attacker", "user"), body); w.Code != http.StatusNotFound {
		t.Fatalf("a non-owner exported the shares: %d %s", w.Code, w.Body.String())
	}

	w := mpcRequest(t, h, http.MethodPost, "/api/v1/mpc/keygen/export", mintToken(t, "owner-c", "user"), body)
	if w.Code != http.StatusOK {
		t.Fatalf("the owner could not export: %d %s", w.Code, w.Body.String())
	}
	var resp exportShareResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Shares) != 3 {
		t.Fatalf("exported %d shares, want 3", len(resp.Shares))
	}
	for _, blob := range resp.Shares {
		if blob.Ciphertext == "" {
			t.Fatal("a share was exported without ciphertext")
		}
	}
}

// TestMPCOperatorMayActAcrossWallets proves the service role still works, which
// is how the custody backend drives the engine.
func TestMPCOperatorMayActAcrossWallets(t *testing.T) {
	wallet := createWalletAs(t, "owner-d")

	h := secure(testSecret, nil, handleSign)
	w := mpcRequest(t, h, http.MethodPost, "/api/v1/mpc/sign", mintToken(t, "custody-service", "service"),
		map[string]string{"keyId": wallet.KeyID, "messageHash": strings.Repeat("cd", 32)})
	if w.Code != http.StatusOK {
		t.Fatalf("an operator could not sign: %d %s", w.Code, w.Body.String())
	}
}

// TestMPCRecoverRequiresOperator proves share submission, which reconstructs a
// signing key, is not available to ordinary users.
func TestMPCRecoverRequiresOperator(t *testing.T) {
	h := secure(testSecret, nil, handleRecoverWallet)
	body := map[string]interface{}{
		"passphrase": "a-strong-passphrase",
		"shares":     []map[string]interface{}{{"version": 1, "keyId": "key_x", "partyId": 1, "threshold": 2, "totalShards": 3}},
	}
	if w := mpcRequest(t, h, http.MethodPost, "/api/v1/mpc/keygen/recover", mintToken(t, "user", "user"), body); w.Code != http.StatusForbidden {
		t.Fatalf("a non-operator reached recovery: %d %s", w.Code, w.Body.String())
	}
}

// TestMPCCORSIsNotWildcard proves the service no longer advertises itself to
// every origin, and that only allowlisted origins receive CORS headers.
func TestMPCCORSIsNotWildcard(t *testing.T) {
	origins := map[string]bool{"https://wallet.tigerwallet.io": true}
	h := secure(testSecret, origins, handleGetWallet)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/mpc/wallet/key_abc", nil)
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Authorization", "Bearer "+mintToken(t, "user", "user"))
	w := httptest.NewRecorder()
	h(w, req)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("a non-allowlisted origin received %q", got)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/mpc/wallet/key_abc", nil)
	req2.Header.Set("Origin", "https://wallet.tigerwallet.io")
	req2.Header.Set("Authorization", "Bearer "+mintToken(t, "user", "user"))
	w2 := httptest.NewRecorder()
	h(w2, req2)
	if got := w2.Header().Get("Access-Control-Allow-Origin"); got != "https://wallet.tigerwallet.io" {
		t.Fatalf("the allowlisted origin received %q", got)
	}
}

// TestMPCSecretFailsClosed proves an unset secret is fatal rather than silently
// serving unauthenticated traffic.
func TestMPCSecretFailsClosed(t *testing.T) {
	t.Setenv("MPC_JWT_SECRET", "")
	t.Setenv("JWT_SECRET", "")
	if _, err := jwtSecret(); err == nil {
		t.Fatal("an unset secret was accepted")
	}

	t.Setenv("JWT_SECRET", "shared")
	if secret, err := jwtSecret(); err != nil || secret != "shared" {
		t.Fatalf("JWT_SECRET fallback failed: %v", err)
	}

	t.Setenv("MPC_JWT_SECRET", "dedicated")
	if secret, err := jwtSecret(); err != nil || secret != "dedicated" {
		t.Fatalf("MPC_JWT_SECRET did not take precedence: %v", err)
	}
}
