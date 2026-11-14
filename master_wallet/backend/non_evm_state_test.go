package main

// non_evm_state_test.go — the account-model state fetchers are exercised
// against a real HTTP server so the request shape, the decode and the
// fail-closed behaviour are all covered. No internal function is stubbed: the
// server returns the same payloads the live nodes return.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNearSendParamsReadsNonceAndBlockHash(t *testing.T) {
	var gotAccessKey, gotBlock bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(body)
		payload := string(body)
		switch {
		case strings.Contains(payload, "view_access_key"):
			gotAccessKey = true
			if !strings.Contains(payload, "ed25519:abc") {
				t.Errorf("public key not forwarded: %s", payload)
			}
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"nonce":41,"block_height":1}}`))
		case strings.Contains(payload, `"block"`):
			gotBlock = true
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"header":{"hash":"EXkxGCoHhJGfihGZPEBCM7ZHN2n7x9RmM63Lrbb2agLj"}}}`))
		default:
			t.Errorf("unexpected NEAR call: %s", payload)
		}
	}))
	defer srv.Close()

	params, err := nearSendParams(context.Background(), srv.URL, "acct.near", "ed25519:abc")
	if err != nil {
		t.Fatalf("nearSendParams: %v", err)
	}
	if !gotAccessKey || !gotBlock {
		t.Fatalf("expected both RPC calls, access_key=%v block=%v", gotAccessKey, gotBlock)
	}
	// NEAR requires a nonce strictly greater than the access key's nonce.
	if params.Nonce != 42 {
		t.Fatalf("nonce = %d, want 42 (stored nonce + 1)", params.Nonce)
	}
	if params.BlockHashB58 == "" {
		t.Fatal("block hash not populated")
	}
}

func TestNearSendParamsFailsClosedOnRPCError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"message":"access key does not exist"}}`))
	}))
	defer srv.Close()

	if _, err := nearSendParams(context.Background(), srv.URL, "acct.near", "ed25519:abc"); err == nil {
		t.Fatal("expected an error when the node reports a failure")
	}
}

func TestNearSendParamsFailsClosedOnMissingBlockHash(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(body)
		if strings.Contains(string(body), "view_access_key") {
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"nonce":1}}`))
			return
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"header":{}}}`))
	}))
	defer srv.Close()

	if _, err := nearSendParams(context.Background(), srv.URL, "acct.near", "ed25519:abc"); err == nil {
		t.Fatal("expected an error when the block header has no hash")
	}
}

func TestStellarSequenceParsesDecimalString(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/accounts/GABCD") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"account_id":"GABCD","sequence":"123456789012345"}`))
	}))
	defer srv.Close()

	seq, err := stellarSequence(context.Background(), srv.URL, "GABCD")
	if err != nil {
		t.Fatalf("stellarSequence: %v", err)
	}
	if seq != 123456789012345 {
		t.Fatalf("sequence = %d, want 123456789012345", seq)
	}
}

func TestStellarSequenceFailsClosedOnBadSequence(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"sequence":"not-a-number"}`))
	}))
	defer srv.Close()

	if _, err := stellarSequence(context.Background(), srv.URL, "GABCD"); err == nil {
		t.Fatal("expected an error for a non-numeric sequence")
	}
}

func TestAlgoTxParamsBuildsValidRoundWindow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/v2/transactions/params") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"genesis-hash":"wGHE2Pwdvd7S12BL5FaOP20EGYesN73ktiC1qzkkit8=","last-round":66947572,"min-fee":1000,"fee":0}`))
	}))
	defer srv.Close()

	genesis, first, last, fee, err := algoTxParams(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("algoTxParams: %v", err)
	}
	if genesis == "" {
		t.Fatal("genesis hash not populated")
	}
	if first != 66947573 {
		t.Fatalf("first valid = %d, want last-round + 1", first)
	}
	if last <= first {
		t.Fatalf("last valid %d must exceed first valid %d", last, first)
	}
	if fee != 1000 {
		t.Fatalf("fee = %d, want the node min-fee 1000", fee)
	}
}

func TestAlgoTxParamsFailsClosedWithoutGenesis(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"last-round":100,"min-fee":1000}`))
	}))
	defer srv.Close()

	if _, _, _, _, err := algoTxParams(context.Background(), srv.URL); err == nil {
		t.Fatal("expected an error when the node omits the genesis hash")
	}
}

func TestTezosCounterIncrements(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/contracts/tz1abc/counter") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`"7"`))
	}))
	defer srv.Close()

	counter, err := tezosCounter(context.Background(), srv.URL, "tz1abc")
	if err != nil {
		t.Fatalf("tezosCounter: %v", err)
	}
	if counter != 8 {
		t.Fatalf("counter = %d, want stored counter + 1", counter)
	}
}

func TestTezosManagerKeyDistinguishesUnrevealed(t *testing.T) {
	// An unrevealed account answers 404 with a body that is not a key. That is a
	// normal state and must be reported as "empty", not as a transport failure.
	unrevealed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"Not found"}`))
	}))
	defer unrevealed.Close()

	key, err := tezosManagerKey(context.Background(), unrevealed.URL, "tz1abc")
	if err != nil {
		t.Fatalf("tezosManagerKey on unrevealed account: %v", err)
	}
	if key != "" {
		t.Fatalf("expected an empty manager key, got %q", key)
	}

	revealed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`"edpkuabc"`))
	}))
	defer revealed.Close()

	key, err = tezosManagerKey(context.Background(), revealed.URL, "tz1abc")
	if err != nil {
		t.Fatalf("tezosManagerKey on revealed account: %v", err)
	}
	if key != "edpkuabc" {
		t.Fatalf("manager key = %q, want edpkuabc", key)
	}
}

func TestAptosSequenceParsesDecimalString(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"sequence_number":"9"}`))
	}))
	defer srv.Close()

	seq, err := aptosSequence(context.Background(), srv.URL, "0x1")
	if err != nil {
		t.Fatalf("aptosSequence: %v", err)
	}
	if seq != 9 {
		t.Fatalf("sequence = %d, want 9", seq)
	}
}

func TestStateFetchersFailClosedOnHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("upstream down"))
	}))
	defer srv.Close()

	ctx := context.Background()
	if _, err := stellarSequence(ctx, srv.URL, "GABCD"); err == nil {
		t.Fatal("stellarSequence must fail closed on HTTP 500")
	}
	if _, _, _, _, err := algoTxParams(ctx, srv.URL); err == nil {
		t.Fatal("algoTxParams must fail closed on HTTP 500")
	}
	if _, err := tezosCounter(ctx, srv.URL, "tz1abc"); err == nil {
		t.Fatal("tezosCounter must fail closed on HTTP 500")
	}
	if _, err := aptosSequence(ctx, srv.URL, "0x1"); err == nil {
		t.Fatal("aptosSequence must fail closed on HTTP 500")
	}
	if _, err := jsonRPCCall(ctx, srv.URL, "block", nil); err == nil {
		t.Fatal("jsonRPCCall must fail closed on HTTP 500")
	}
}
