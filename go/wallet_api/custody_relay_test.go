package main

// Tests for the service-to-service custody relay. These exercise the real
// signature path: the test builds and signs genuine EIP-155 transactions with
// go-ethereum, so every sender the handler recovers is an actual signer.

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
)

// testSigner holds a generated key and the account it controls.
type testSigner struct {
	key     *ecdsa.PrivateKey
	addr    common.Address
	addrHex string
}

func newTestSigner(t *testing.T) *testSigner {
	t.Helper()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	addr := crypto.PubkeyToAddress(key.PublicKey)
	return &testSigner{key: key, addr: addr, addrHex: addr.Hex()}
}

// signBundle signs the hash carried by b, mirroring what an MPC backend does:
// it returns a raw 65-byte [R||S||V] signature over the digest.
func (s *testSigner) signBundle(t *testing.T, b txBundle) string {
	t.Helper()
	sig, err := crypto.Sign(common.FromHex(b.SigningHash), s.key)
	if err != nil {
		t.Fatalf("sign hash: %v", err)
	}
	return "0x" + common.Bytes2Hex(sig)
}

// signBundleLegacyV signs and then converts V to the 27/28 convention, which
// real MPC backends commonly emit. The relay must accept both.
func (s *testSigner) signBundleLegacyV(t *testing.T, b txBundle) string {
	t.Helper()
	sig, err := crypto.Sign(common.FromHex(b.SigningHash), s.key)
	if err != nil {
		t.Fatalf("sign hash: %v", err)
	}
	sig[64] += 27
	return "0x" + common.Bytes2Hex(sig)
}

func legacyBundle(from string) txBundle {
	return txBundle{
		ChainID:     1,
		From:        from,
		To:          "0x000000000000000000000000000000000000dEaD",
		ValueWei:    "1000000000000000000",
		Data:        "0x",
		GasLimit:    21000,
		TxType:      "legacy",
		Nonce:       7,
		GasPriceWei: "20000000000",
	}
}

func hashFor(t *testing.T, b txBundle) string {
	t.Helper()
	to, err := hexAddress(b.To)
	if err != nil {
		t.Fatalf("to: %v", err)
	}
	value, err := parseDecimalBig(b.ValueWei)
	if err != nil {
		t.Fatalf("value: %v", err)
	}
	gp, err := parseDecimalBig(b.GasPriceWei)
	if err != nil {
		t.Fatalf("gas: %v", err)
	}
	tx, signer, err := buildUnsignedTx(b, to, value, nil, gp, nil, nil)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	return signer.Hash(tx).Hex()
}

// ---------------------------------------------------------------------------
// Pure assembly: the tamper-detection property
// ---------------------------------------------------------------------------

func TestAssembleSignedTxAcceptsValidSignature(t *testing.T) {
	s := newTestSigner(t)
	b := legacyBundle(s.addrHex)
	b.SigningHash = hashFor(t, b)

	tx, sender, err := assembleSignedTx(b, s.signBundle(t, b))
	if err != nil {
		t.Fatalf("valid signature rejected: %v", err)
	}
	if sender != s.addr {
		t.Fatalf("recovered %s, want %s", sender.Hex(), s.addr.Hex())
	}
	if tx == nil {
		t.Fatal("no transaction returned")
	}
	if tx.Value().Cmp(big.NewInt(1e18)) != 0 {
		t.Fatalf("value %s, want 1e18", tx.Value())
	}
}

// A 27/28 V byte is what many TSS backends emit; rejecting it would make the
// relay unusable with them.
func TestAssembleSignedTxAcceptsLegacyVByte(t *testing.T) {
	s := newTestSigner(t)
	b := legacyBundle(s.addrHex)
	b.SigningHash = hashFor(t, b)

	_, sender, err := assembleSignedTx(b, s.signBundleLegacyV(t, b))
	if err != nil {
		t.Fatalf("27/28-style V rejected: %v", err)
	}
	if sender != s.addr {
		t.Fatalf("recovered %s, want %s", sender.Hex(), s.addr.Hex())
	}
}

// The core security property: changing the recipient after signing must not
// produce a broadcastable transaction.
func TestAssembleSignedTxRejectsTamperedRecipient(t *testing.T) {
	s := newTestSigner(t)
	b := legacyBundle(s.addrHex)
	b.SigningHash = hashFor(t, b)
	sig := s.signBundle(t, b)

	b.To = "0x1111111111111111111111111111111111111111"

	if _, _, err := assembleSignedTx(b, sig); err != errHashMismatch {
		t.Fatalf("expected errHashMismatch, got %v", err)
	}
}

func TestAssembleSignedTxRejectsTamperedAmount(t *testing.T) {
	s := newTestSigner(t)
	b := legacyBundle(s.addrHex)
	b.SigningHash = hashFor(t, b)
	sig := s.signBundle(t, b)

	b.ValueWei = "999000000000000000000"

	if _, _, err := assembleSignedTx(b, sig); err != errHashMismatch {
		t.Fatalf("expected errHashMismatch, got %v", err)
	}
}

func TestAssembleSignedTxRejectsTamperedFee(t *testing.T) {
	s := newTestSigner(t)
	b := legacyBundle(s.addrHex)
	b.SigningHash = hashFor(t, b)
	sig := s.signBundle(t, b)

	b.GasPriceWei = "1"

	if _, _, err := assembleSignedTx(b, sig); err != errHashMismatch {
		t.Fatalf("expected errHashMismatch, got %v", err)
	}
}

func TestAssembleSignedTxRejectsTamperedNonce(t *testing.T) {
	s := newTestSigner(t)
	b := legacyBundle(s.addrHex)
	b.SigningHash = hashFor(t, b)
	sig := s.signBundle(t, b)

	b.Nonce = 8

	if _, _, err := assembleSignedTx(b, sig); err != errHashMismatch {
		t.Fatalf("expected errHashMismatch, got %v", err)
	}
}

// A signature produced by a different key must never be accepted for the
// declared account.
func TestAssembleSignedTxRejectsForeignSigner(t *testing.T) {
	victim := newTestSigner(t)
	attacker := newTestSigner(t)

	b := legacyBundle(victim.addrHex)
	b.SigningHash = hashFor(t, b)

	// The attacker signs the correct hash but with the wrong key. The hash
	// matches, so only the sender recovery check can catch this.
	if _, _, err := assembleSignedTx(b, attacker.signBundle(t, b)); err != errSenderMismatch {
		t.Fatalf("expected errSenderMismatch, got %v", err)
	}
}

func TestAssembleSignedTxRejectsMalformedSignature(t *testing.T) {
	s := newTestSigner(t)
	b := legacyBundle(s.addrHex)
	b.SigningHash = hashFor(t, b)

	for _, sig := range []string{"0x", "0xdeadbeef", "not-hex-at-all"} {
		if _, _, err := assembleSignedTx(b, sig); err != errBadSignature {
			t.Fatalf("signature %q: expected errBadSignature, got %v", sig, err)
		}
	}
}

func TestAssembleSignedTxRejectsBadAddresses(t *testing.T) {
	s := newTestSigner(t)

	b := legacyBundle("not-an-address")
	b.SigningHash = "0x00"
	if _, _, err := assembleSignedTx(b, "0x00"); err != errBadFrom {
		t.Fatalf("expected errBadFrom, got %v", err)
	}

	b = legacyBundle(s.addrHex)
	b.To = "zzz"
	if _, _, err := assembleSignedTx(b, "0x00"); err != errBadTo {
		t.Fatalf("expected errBadTo, got %v", err)
	}
}

// EIP-1559 transactions must round-trip as well, and the hash must differ from
// the legacy hash of the same amounts.
func TestAssembleSignedTxHandlesEIP1559(t *testing.T) {
	s := newTestSigner(t)
	b := txBundle{
		ChainID:        1,
		From:           s.addrHex,
		To:             "0x000000000000000000000000000000000000dEaD",
		ValueWei:       "500000000000000000",
		Data:           "0x",
		GasLimit:       21000,
		TxType:         "eip1559",
		Nonce:          3,
		MaxFeeWei:      "30000000000",
		MaxPriorityWei: "2000000000",
	}
	to, _ := hexAddress(b.To)
	value, _ := parseDecimalBig(b.ValueWei)
	mf, _ := parseDecimalBig(b.MaxFeeWei)
	mp, _ := parseDecimalBig(b.MaxPriorityWei)
	tx, signer, err := buildUnsignedTx(b, to, value, nil, nil, mf, mp)
	if err != nil {
		t.Fatalf("build eip1559: %v", err)
	}
	b.SigningHash = signer.Hash(tx).Hex()

	signed, sender, err := assembleSignedTx(b, s.signBundle(t, b))
	if err != nil {
		t.Fatalf("eip1559 rejected: %v", err)
	}
	if sender != s.addr {
		t.Fatalf("recovered %s, want %s", sender.Hex(), s.addr.Hex())
	}
	if signed.Type() != types.DynamicFeeTxType {
		t.Fatalf("type %d, want DynamicFeeTxType", signed.Type())
	}
}

// ---------------------------------------------------------------------------
// HTTP surface
// ---------------------------------------------------------------------------

func relayRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/service/tx/prepare", handleServiceTxPrepare)
	r.POST("/api/v1/service/tx/submit", handleServiceTxSubmit)
	r.POST("/api/v1/service/broadcast", handleServiceBroadcast)
	return r
}

func postJSON(t *testing.T, r *gin.Engine, path, token string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRelayDisabledWithoutToken(t *testing.T) {
	t.Setenv("SERVICE_BROADCAST_TOKEN", "")
	r := relayRouter()
	s := newTestSigner(t)

	for _, path := range []string{
		"/api/v1/service/tx/prepare",
		"/api/v1/service/tx/submit",
		"/api/v1/service/broadcast",
	} {
		w := postJSON(t, r, path, "whatever", map[string]interface{}{})
		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s: expected 503 when disabled, got %d: %s", path, w.Code, w.Body.String())
		}
	}
	_ = s
}

func TestRelayRejectsBadToken(t *testing.T) {
	t.Setenv("SERVICE_BROADCAST_TOKEN", "correct-token")
	r := relayRouter()

	for _, path := range []string{
		"/api/v1/service/tx/prepare",
		"/api/v1/service/tx/submit",
		"/api/v1/service/broadcast",
	} {
		for _, tok := range []string{"", "wrong"} {
			w := postJSON(t, r, path, tok, map[string]interface{}{})
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("%s token %q: expected 401, got %d", path, tok, w.Code)
			}
		}
	}
}

func TestRelayPrepareRejectsUnknownChain(t *testing.T) {
	t.Setenv("SERVICE_BROADCAST_TOKEN", "tok")
	r := relayRouter()
	s := newTestSigner(t)

	w := postJSON(t, r, "/api/v1/service/tx/prepare", "tok", map[string]interface{}{
		"chain_id": 999999,
		"from":     s.addrHex,
		"to":       "0x000000000000000000000000000000000000dEaD",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRelaySubmitRequiresSigningHash(t *testing.T) {
	t.Setenv("SERVICE_BROADCAST_TOKEN", "tok")
	r := relayRouter()
	s := newTestSigner(t)

	b := legacyBundle(s.addrHex)
	body := map[string]interface{}{
		"chain_id": b.ChainID, "from": b.From, "to": b.To,
		"value_wei": b.ValueWei, "gas_limit": b.GasLimit,
		"tx_type": "legacy", "nonce": b.Nonce,
		"gas_price_wei": b.GasPriceWei, "signature": "0x00",
	}
	w := postJSON(t, r, "/api/v1/service/tx/submit", "tok", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing signing_hash, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "signing_hash") {
		t.Fatalf("error should mention signing_hash: %s", w.Body.String())
	}
}

// Submit must report 409 when the caller's fields do not hash to the value it
// claims to have signed — this is the tamper signal an operator sees.
func TestRelaySubmitRejectsTamperedFields(t *testing.T) {
	t.Setenv("SERVICE_BROADCAST_TOKEN", "tok")
	r := relayRouter()
	s := newTestSigner(t)

	b := legacyBundle(s.addrHex)
	b.SigningHash = hashFor(t, b)
	sig := s.signBundle(t, b)

	body := map[string]interface{}{
		"chain_id": b.ChainID, "from": b.From,
		"to":        "0x1111111111111111111111111111111111111111",
		"value_wei": b.ValueWei, "gas_limit": b.GasLimit,
		"tx_type": "legacy", "nonce": b.Nonce,
		"gas_price_wei": b.GasPriceWei,
		"signing_hash":  b.SigningHash, "signature": sig,
	}
	w := postJSON(t, r, "/api/v1/service/tx/submit", "tok", body)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for tampered fields, got %d: %s", w.Code, w.Body.String())
	}
}

// A foreign signature is caught even when the hash is correct.
func TestRelaySubmitRejectsForeignSignature(t *testing.T) {
	t.Setenv("SERVICE_BROADCAST_TOKEN", "tok")
	r := relayRouter()
	victim := newTestSigner(t)
	attacker := newTestSigner(t)

	b := legacyBundle(victim.addrHex)
	b.SigningHash = hashFor(t, b)
	sig := attacker.signBundle(t, b)

	body := map[string]interface{}{
		"chain_id": b.ChainID, "from": b.From, "to": b.To,
		"value_wei": b.ValueWei, "gas_limit": b.GasLimit,
		"tx_type": "legacy", "nonce": b.Nonce,
		"gas_price_wei": b.GasPriceWei,
		"signing_hash":  b.SigningHash, "signature": sig,
	}
	w := postJSON(t, r, "/api/v1/service/tx/submit", "tok", body)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for foreign signature, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Low-level raw relay
// ---------------------------------------------------------------------------

func TestRawBroadcastRejectsUndecodablePayload(t *testing.T) {
	t.Setenv("SERVICE_BROADCAST_TOKEN", "tok")
	r := relayRouter()

	w := postJSON(t, r, "/api/v1/service/broadcast", "tok",
		map[string]interface{}{"raw_tx": "0xdeadbeef", "chain_id": 1})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "not a decodable transaction") {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestRawBroadcastRejectsSenderMismatch(t *testing.T) {
	t.Setenv("SERVICE_BROADCAST_TOKEN", "tok")
	r := relayRouter()
	s := newTestSigner(t)

	b := legacyBundle(s.addrHex)
	b.SigningHash = hashFor(t, b)
	to, _ := hexAddress(b.To)
	value, _ := parseDecimalBig(b.ValueWei)
	gp, _ := parseDecimalBig(b.GasPriceWei)
	tx, signer, err := buildUnsignedTx(b, to, value, nil, gp, nil, nil)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	signed, err := tx.WithSignature(signer, common.FromHex(s.signBundle(t, b)))
	if err != nil {
		t.Fatalf("with signature: %v", err)
	}
	raw, _ := signed.MarshalBinary()

	other := common.HexToAddress("0x2222222222222222222222222222222222222222")
	w := postJSON(t, r, "/api/v1/service/broadcast", "tok", map[string]interface{}{
		"raw_tx":        "0x" + common.Bytes2Hex(raw),
		"chain_id":      1,
		"expected_from": other.Hex(),
	})
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(strings.ToLower(w.Body.String()), strings.ToLower(s.addrHex)) {
		t.Fatalf("response should name recovered sender: %s", w.Body.String())
	}
}

func TestRawBroadcastRejectsUnknownChain(t *testing.T) {
	t.Setenv("SERVICE_BROADCAST_TOKEN", "tok")
	r := relayRouter()
	s := newTestSigner(t)

	w := postJSON(t, r, "/api/v1/service/broadcast", "tok", map[string]interface{}{
		"raw_tx":   "0x02f8700180808080808080c001a0",
		"chain_id": 999999,
	})
	_ = s
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
