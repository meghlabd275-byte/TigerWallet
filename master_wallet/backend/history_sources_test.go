package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// historySourcesFor is historySources with the key env stubbed.
func historySourcesFor(t *testing.T, chainID int64, etherscanKey string) []explorerHistorySource {
	t.Helper()
	t.Setenv("ETHERSCAN_API_KEY", etherscanKey)
	return historySources(chainID)
}

// Without a key the Etherscan V2 source must be absent: it can only answer
// "Missing/Invalid API Key", and an empty history would be indistinguishable
// from an address that has never transacted. Ethereum must instead fall through
// to its keyless Blockscout backend.
func TestHistorySourcesOmitKeylessEtherscan(t *testing.T) {
	srcs := historySourcesFor(t, 1, "")
	if len(srcs) == 0 {
		t.Fatal("chain 1 must have at least one history source without a key")
	}
	for _, s := range srcs {
		if s.etherscanV2 {
			t.Fatal("the Etherscan V2 source must not be used without ETHERSCAN_API_KEY")
		}
	}
	if !srcs[0].blockscoutV2 {
		t.Fatalf("expected the keyless Blockscout backend first, got %+v", srcs[0])
	}
	if !strings.Contains(srcs[0].baseURL, "eth.blockscout.com") {
		t.Fatalf("unexpected keyless host for chain 1: %s", srcs[0].baseURL)
	}
}

// With a key the Etherscan V2 source is preferred, and Blockscout remains as a
// fallback for when the key is rate-limited or the chain is not covered.
func TestHistorySourcesPreferEtherscanWithKey(t *testing.T) {
	srcs := historySourcesFor(t, 1, "a-key")
	if len(srcs) < 2 {
		t.Fatalf("expected Etherscan plus a fallback, got %d sources", len(srcs))
	}
	if !srcs[0].etherscanV2 || srcs[0].apiKeyEnv != "ETHERSCAN_API_KEY" {
		t.Fatalf("expected the Etherscan V2 source first, got %+v", srcs[0])
	}
	if !srcs[1].blockscoutV2 {
		t.Fatalf("expected the keyless backend as fallback, got %+v", srcs[1])
	}
}

// A chain outside the Etherscan family resolves through the registry explorer.
func TestHistorySourcesRegistryFallback(t *testing.T) {
	srcs := historySourcesFor(t, 246, "")
	if len(srcs) != 1 || srcs[0].blockscoutV2 || srcs[0].etherscanV2 {
		t.Fatalf("expected a single registry explorer source, got %+v", srcs)
	}
	if !strings.Contains(srcs[0].baseURL, "/api") {
		t.Fatalf("registry explorer must use the Etherscan-compatible /api path: %s", srcs[0].baseURL)
	}
}

// An unknown chain has no source at all; the caller must surface that rather
// than reporting an empty history.
func TestHistorySourcesUnknownChain(t *testing.T) {
	if srcs := historySourcesFor(t, 999999, ""); len(srcs) != 0 {
		t.Fatalf("unknown chain must have no sources, got %+v", srcs)
	}
}

// The Blockscout decoder must map its response onto the same shape the
// Etherscan path produces, including the RFC3339 timestamp and the
// success/failure word.
func TestFetchBlockscoutV2DecodesTransactions(t *testing.T) {
	body := `{"items":[
  {"hash":"0xaaa","from":{"hash":"0x111"},"to":{"hash":"0x222"},
   "value":"1000000000000000000","status":"ok","result":"success",
   "gas_used":"21000","timestamp":"2026-04-16T02:46:57.000000Z","block_number":5},
  {"hash":"0xbbb","from":{"hash":"0x111"},"to":null,
   "value":"0","status":"error","result":"reverted",
   "gas_used":"33000","timestamp":"2026-04-16T03:00:00.000000Z","block_number":6}
]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/api/v2/addresses/0xabc/transactions") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("User-Agent") == "" {
			t.Error("a User-Agent is required: the public explorers reject requests without one")
		}
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	txs, err := fetchBlockscoutV2(context.Background(), srv.URL, 1, "0xabc", 50)
	if err != nil {
		t.Fatalf("fetchBlockscoutV2: %v", err)
	}
	if len(txs) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(txs))
	}
	if txs[0].Hash != "0xaaa" || txs[0].From != "0x111" || txs[0].To != "0x222" {
		t.Fatalf("first transaction decoded wrong: %+v", txs[0])
	}
	if txs[0].Value != "1000000000000000000" || txs[0].GasUsed != "21000" {
		t.Fatalf("value/gas not preserved: %+v", txs[0])
	}
	if txs[0].Time != 1776307617 {
		t.Fatalf("RFC3339 timestamp not converted, got %d", txs[0].Time)
	}
	if txs[0].IsError != 0 {
		t.Fatalf("a successful tx must not be marked as an error: %+v", txs[0])
	}
	// A contract-creation transaction has no recipient; it must not panic and
	// must not invent an address.
	if txs[1].To != "" || txs[1].IsError != 1 {
		t.Fatalf("failed/contract-creation tx decoded wrong: %+v", txs[1])
	}
	if txs[1].ChainID != 1 {
		t.Fatalf("chain id not stamped: %+v", txs[1])
	}
}

// The limit must actually bound the response.
func TestFetchBlockscoutV2HonoursLimit(t *testing.T) {
	items := make([]string, 0, 5)
	for i := 0; i < 5; i++ {
		items = append(items, `{"hash":"0x`+string(rune('a'+i))+`","from":{"hash":"0x1"},"to":{"hash":"0x2"},"value":"1","status":"ok","timestamp":"2026-01-01T00:00:00Z"}`)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"items":[` + strings.Join(items, ",") + `]}`))
	}))
	defer srv.Close()

	txs, err := fetchBlockscoutV2(context.Background(), srv.URL, 1, "0xabc", 2)
	if err != nil {
		t.Fatalf("fetchBlockscoutV2: %v", err)
	}
	if len(txs) != 2 {
		t.Fatalf("limit ignored: got %d transactions", len(txs))
	}
}

// A non-200 must be an error, never an empty history.
func TestFetchBlockscoutV2FailsClosedOnErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("<html>Just a moment...</html>"))
	}))
	defer srv.Close()

	if _, err := fetchBlockscoutV2(context.Background(), srv.URL, 1, "0xabc", 50); err == nil {
		t.Fatal("a 403 must be an error, not an empty history")
	}
}

// The Etherscan decoder must still work, and must return the real upstream
// message rather than an empty list when the API rejects the request.
func TestFetchEtherscanTxListReturnsUpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"0","message":"NOTOK","result":"Missing/Invalid API Key"}`))
	}))
	defer srv.Close()

	_, err := fetchEtherscanTxList(context.Background(), srv.URL, "", 1, "0xabc", 50)
	if err == nil {
		t.Fatal("an API-key rejection must be an error")
	}
	if !strings.Contains(err.Error(), "Missing/Invalid API Key") {
		t.Fatalf("the real upstream message must be preserved, got: %v", err)
	}
}

// "No transactions found" is a legitimate empty result, not a failure.
func TestFetchEtherscanTxListEmptyIsNotAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"0","message":"No transactions found","result":[]}`))
	}))
	defer srv.Close()

	txs, err := fetchEtherscanTxList(context.Background(), srv.URL, "", 1, "0xabc", 50)
	if err != nil {
		t.Fatalf("an address with no history is not an error: %v", err)
	}
	if len(txs) != 0 {
		t.Fatalf("expected no transactions, got %d", len(txs))
	}
}

// The V2 request shape must carry the chainid parameter, or the multichain API
// answers for the wrong chain.
func TestFetchEtherscanTxListSendsChainIDForV2(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"status":"1","message":"OK","result":[]}`))
	}))
	defer srv.Close()

	// The V2 detection keys off "/v2/" in the base URL.
	if _, err := fetchEtherscanTxList(context.Background(), srv.URL+"/v2/api", "k", 137, "0xabc", 50); err != nil {
		t.Fatalf("fetchEtherscanTxList: %v", err)
	}
	if !strings.Contains(got, "chainid=137") {
		t.Fatalf("the chainid parameter is missing from %q", got)
	}
	if !strings.Contains(got, "apikey=k") {
		t.Fatalf("the apikey parameter is missing from %q", got)
	}
}

// The chain-aware entry point must report every backend it tried, so an
// operator can see why history is unavailable.
func TestFetchTransactionHistoryForChainReportsAllFailures(t *testing.T) {
	ctx := context.Background()
	_, err := FetchTransactionHistoryForChain(ctx, 999999, "0xabc", 10)
	if err == nil {
		t.Fatal("an unknown chain must fail")
	}
	if !strings.Contains(err.Error(), "no transaction-history backend") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// The Blockscout payload must decode from the same JSON a caller would send,
// proving the struct tags line up with the wire format.
func TestBlockscoutTxTagsMatchTheWireFormat(t *testing.T) {
	var tx blockscoutTx
	raw := `{"hash":"0x1","from":{"hash":"0xf"},"to":{"hash":"0xt"},"value":"5",
         "status":"ok","gas_used":"21000","timestamp":"2026-01-01T00:00:00Z"}`
	if err := json.Unmarshal([]byte(raw), &tx); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if tx.GasUsed != "21000" || tx.From.Hash != "0xf" || tx.To == nil || tx.To.Hash != "0xt" {
		t.Fatalf("struct tags do not match the wire format: %+v", tx)
	}
}
