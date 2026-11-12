package main

// Guards the pre-broadcast chain check: a transaction signed for one network
// must never be relayed to another, because the signature is valid there too.

import (
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func chainIDNode(t *testing.T, chainID string, calls *int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*calls++
		w.Header().Set("Content-Type", "application/json")
		// eth_sendRawTransaction would be the second call; answering it means
		// the guard let the request through.
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"` + chainID + `"}`))
	}))
}

func TestBroadcastRefusesMismatchedChain(t *testing.T) {
	calls := 0
	srv := chainIDNode(t, "0x1", &calls) // node reports Ethereum
	defer srv.Close()

	_, err := BroadcastTransaction(context.Background(), srv.URL, "0xdeadbeef", big.NewInt(7710))
	if err == nil {
		t.Fatal("broadcast to a node on a different chain was allowed")
	}
	if !strings.Contains(err.Error(), "refusing to broadcast") {
		t.Fatalf("error = %v, want a refusal naming the chain mismatch", err)
	}
	if calls != 1 {
		t.Fatalf("node received %d calls; the transaction must not be sent", calls)
	}
}

func TestBroadcastProceedsOnMatchingChain(t *testing.T) {
	calls := 0
	srv := chainIDNode(t, "0x1e1e", &calls) // node reports 7710
	defer srv.Close()

	if _, err := BroadcastTransaction(context.Background(), srv.URL, "0xdeadbeef", big.NewInt(7710)); err != nil {
		t.Fatalf("broadcast on the matching chain failed: %v", err)
	}
	if calls != 2 {
		t.Fatalf("node received %d calls, want chain check then send", calls)
	}
}

// An endpoint that cannot answer eth_chainId cannot be shown to be the intended
// network, so the send must not proceed.
func TestBroadcastFailsClosedWhenChainUnverifiable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}))
	defer srv.Close()

	if _, err := BroadcastTransaction(context.Background(), srv.URL, "0xdeadbeef", big.NewInt(7710)); err == nil {
		t.Fatal("broadcast proceeded although the chain could not be verified")
	}
}

// Against the live node, a transaction signed for a different network must be
// refused before anything reaches the mempool. The expected chain id is
// deliberately wrong, so the guard must reject the request and the node must
// never see eth_sendRawTransaction — nothing is broadcast by this test.
func TestBroadcastChainGuardAgainstLiveNode(t *testing.T) {
	endpoint := os.Getenv(tscRPCEnv)
	if endpoint == "" {
		t.Skipf("%s is not set; skipping live chain-guard check", tscRPCEnv)
	}

	_, err := BroadcastTransaction(context.Background(), endpoint, "0xdeadbeef", big.NewInt(1))
	if err == nil {
		t.Fatal("the live node accepted a transaction signed for chain 1")
	}
	if !strings.Contains(err.Error(), "refusing to broadcast") {
		t.Fatalf("error = %v, want a refusal naming the chain mismatch", err)
	}
}
