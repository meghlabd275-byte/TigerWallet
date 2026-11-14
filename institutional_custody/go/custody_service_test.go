package main

// Tests for the custody withdrawal path. The signing/relay half lives in
// wallet_api and is covered there; these tests pin the parts that decide how
// much money moves, which is where a silent error is most expensive.

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
)

// tokenToWei must be exact. A rounding error here moves the wrong amount.
func TestTokenToWeiExact(t *testing.T) {
	cases := []struct {
		in       string
		decimals int
		want     string
	}{
		{"1", 18, "1000000000000000000"},
		{"1.0", 18, "1000000000000000000"},
		{"0.1", 18, "100000000000000000"},
		{"0.000000000000000001", 18, "1"},
		{"123.456", 18, "123456000000000000000"},
		{"1", 6, "1000000"},
		{"1.5", 6, "1500000"},
		{"1", 0, "1"},
		{"0", 18, "0"},
		{"0.0", 18, "0"},
		// More precision than the chain supports is truncated, never
		// rounded up: we must not move more than authorised.
		{"1.9999999999999999999", 18, "1999999999999999999"},
		{"1.1234567", 6, "1123456"},
	}
	for _, c := range cases {
		got, err := tokenToWei(c.in, c.decimals)
		if err != nil {
			t.Fatalf("tokenToWei(%q,%d): %v", c.in, c.decimals, err)
		}
		if got != c.want {
			t.Errorf("tokenToWei(%q,%d) = %q, want %q", c.in, c.decimals, got, c.want)
		}
	}
}

// Values that would move the wrong amount or trip the parser must be refused.
func TestTokenToWeiRejectsBadInput(t *testing.T) {
	for _, in := range []string{"", "   ", "-1", "-0.5", "abc", "1.2.3"} {
		if got, err := tokenToWei(in, 18); err == nil {
			t.Errorf("tokenToWei(%q) = %q, want error", in, got)
		}
	}
}

// Very large amounts must not overflow: big.Int handles them, and the value
// round-trips through the relay as a decimal string.
func TestTokenToWeiHandlesLargeAmounts(t *testing.T) {
	got, err := tokenToWei("123456789012345678901234567890", 18)
	if err != nil {
		t.Fatalf("large amount rejected: %v", err)
	}
	want := new(big.Int)
	want.SetString("123456789012345678901234567890000000000000000000", 10)
	if got != want.String() {
		t.Fatalf("got %s, want %s", got, want.String())
	}
}

// executeTransaction must fail closed when it cannot reach the pieces it
// needs. A custody withdrawal that silently "succeeds" is the worst outcome.
func TestExecuteTransactionFailsClosedWithoutConfig(t *testing.T) {
	svc := &CustodyService{config: Config{}}
	acct := &CustodyAccount{WalletAddress: "0xabc", ChainID: 1, MPCKeyID: "k1"}
	tx := &TransactionRequest{ToAddress: "0xdef", Amount: "1"}

	if _, err := svc.executeTransaction(acct, tx); err == nil {
		t.Fatal("expected failure with no MPC backend configured")
	}

	svc.config.MPCBackendURL = "http://mpc"
	if _, err := svc.executeTransaction(acct, tx); err == nil {
		t.Fatal("expected failure with no wallet_api configured")
	}

	svc.config.WalletAPIURL = "http://wallet"
	if _, err := svc.executeTransaction(acct, tx); err == nil {
		t.Fatal("expected failure with no service token configured")
	}
}

func TestExecuteTransactionFailsClosedWithoutMPCKey(t *testing.T) {
	svc := &CustodyService{config: Config{
		MPCBackendURL: "http://mpc",
		WalletAPIURL:  "http://wallet",
		ServiceToken:  "tok",
	}}
	acct := &CustodyAccount{WalletAddress: "0xabc", ChainID: 1}
	tx := &TransactionRequest{ToAddress: "0xdef", Amount: "1"}

	if _, err := svc.executeTransaction(acct, tx); err == nil {
		t.Fatal("expected failure when the account has no MPC key")
	}
}

// prepareTransfer must send the amount the operator authorised, in wei, to the
// relay's prepare endpoint.
func TestPrepareTransferSendsExactWei(t *testing.T) {
	var gotPath string
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"chain_id": 1, "from": "0xabc", "to": "0xdef",
			"value_wei": "1500000000000000000", "gas_limit": 21000,
			"tx_type": "legacy", "nonce": 4,
			"gas_price_wei": "20000000000",
			"signing_hash":  "0x" + string(make([]byte, 0)),
		})
	}))
	defer srv.Close()

	svc := &CustodyService{config: Config{WalletAPIURL: srv.URL}, httpClient: srv.Client()}
	acct := &CustodyAccount{WalletAddress: "0xabc", ChainID: 1}
	tx := &TransactionRequest{ToAddress: "0xdef", Amount: "1.5"}

	if _, err := svc.prepareTransfer(acct, tx); err != nil {
		t.Fatalf("prepareTransfer: %v", err)
	}
	if gotPath != "/api/v1/service/tx/prepare" {
		t.Fatalf("path %q, want /api/v1/service/tx/prepare", gotPath)
	}
	if gotBody["value_wei"] != "1500000000000000000" {
		t.Fatalf("value_wei %v, want 1500000000000000000", gotBody["value_wei"])
	}
	if gotBody["to"] != "0xdef" {
		t.Fatalf("to %v", gotBody["to"])
	}
}

// submitTransfer must echo back the exact fields that were signed, plus the
// signature; the relay re-derives the hash from them.
func TestSubmitTransferEchoesSignedFields(t *testing.T) {
	var gotPath string
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"tx_hash": "0xrealhash"})
	}))
	defer srv.Close()

	svc := &CustodyService{config: Config{WalletAPIURL: srv.URL}, httpClient: srv.Client()}
	prepared := &preparedTx{
		ChainID: 1, From: "0xabc", To: "0xdef",
		ValueWei: "1000000000000000000", Data: "0x",
		GasLimit: 21000, TxType: "legacy", Nonce: 4,
		GasPriceWei: "20000000000", SigningHash: "0xhash",
	}

	hash, err := svc.submitTransfer(prepared, "0xsig")
	if err != nil {
		t.Fatalf("submitTransfer: %v", err)
	}
	if hash != "0xrealhash" {
		t.Fatalf("hash %q, want 0xrealhash", hash)
	}
	if gotPath != "/api/v1/service/tx/submit" {
		t.Fatalf("path %q", gotPath)
	}
	for k, want := range map[string]interface{}{
		"signing_hash": "0xhash",
		"signature":    "0xsig",
		"nonce":        float64(4),
		"value_wei":    "1000000000000000000",
	} {
		if gotBody[k] != want {
			t.Errorf("body[%q] = %v, want %v", k, gotBody[k], want)
		}
	}
}

// An error response from the relay must surface as an error, never as a
// successful (fabricated) hash.
func TestSubmitTransferRejectsMissingHash(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{})
	}))
	defer srv.Close()

	svc := &CustodyService{config: Config{WalletAPIURL: srv.URL}, httpClient: srv.Client()}
	if _, err := svc.submitTransfer(&preparedTx{}, "0xsig"); err == nil {
		t.Fatal("expected error when the relay returns no tx_hash")
	}
}

func TestSubmitTransferSurfacesRelayError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "signing_hash does not match the supplied transaction fields",
		})
	}))
	defer srv.Close()

	svc := &CustodyService{config: Config{WalletAPIURL: srv.URL}, httpClient: srv.Client()}
	_, err := svc.submitTransfer(&preparedTx{}, "0xsig")
	if err == nil {
		t.Fatal("expected the relay rejection to surface")
	}
}

// ---------------------------------------------------------------------------
// Route authentication
// ---------------------------------------------------------------------------

// Every custody route must be unreachable without the operator's key. This
// service approves withdrawals, so an unauthenticated caller is a fund-loss
// path, not a hardening nicety.
func TestRequireAPIKeyBlocksUnauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reached := false
	r := gin.New()
	g := r.Group("/api/v1/custody")
	g.Use(requireAPIKey("operator-key"))
	g.POST("/approve", func(c *gin.Context) { reached = true; c.Status(200) })

	for _, hdr := range []map[string]string{
		{},
		{"Authorization": ""},
		{"Authorization": "Bearer wrong"},
		{"X-API-Key": "wrong"},
		{"Authorization": "Bearer "},
	} {
		reached = false
		req := httptest.NewRequest(http.MethodPost, "/api/v1/custody/approve", nil)
		for k, v := range hdr {
			req.Header.Set(k, v)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("headers %v: expected 401, got %d", hdr, w.Code)
		}
		if reached {
			t.Errorf("headers %v: handler ran despite failing auth", hdr)
		}
	}
}

func TestRequireAPIKeyAcceptsBothHeaderForms(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/api/v1/custody")
	g.Use(requireAPIKey("operator-key"))
	g.POST("/approve", func(c *gin.Context) { c.Status(200) })

	for _, hdr := range []map[string]string{
		{"Authorization": "Bearer operator-key"},
		{"X-API-Key": "operator-key"},
	} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/custody/approve", nil)
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

// An empty configured key must fail closed rather than accept everything.
func TestRequireAPIKeyFailsClosedWhenUnconfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/api/v1/custody")
	g.Use(requireAPIKey(""))
	g.POST("/approve", func(c *gin.Context) { c.Status(200) })

	req := httptest.NewRequest(http.MethodPost, "/api/v1/custody/approve", nil)
	req.Header.Set("X-API-Key", "")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when no key is configured, got %d", w.Code)
	}
}
