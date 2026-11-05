/**
 * Tests for the transaction shield threat intelligence and contract
 * inspection.
 *
 * The contract checker is exercised against a real HTTP server that speaks
 * JSON-RPC, so the request shape, the response parsing and the failure paths
 * are all covered without mocking the client.
 */

package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- threat intelligence ----------------------------------------------------

func writeThreatFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "threats.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write threat file: %v", err)
	}
	return path
}

func TestThreatDBLoadAndLookup(t *testing.T) {
	path := writeThreatFile(t, `{
		"entries": {
			"1": [
				{"address": "0x1111111111111111111111111111111111111111", "category": "phishing", "source": "internal"}
			],
			"137": [
				{"address": "0x2222222222222222222222222222222222222222", "category": "honeypot"}
			],
			"global": [
				{"address": "0x3333333333333333333333333333333333333333", "category": "sanctioned", "source": "ofac"}
			]
		}
	}`)

	db := NewThreatDB()
	if err := db.Load(path); err != nil {
		t.Fatalf("load: %v", err)
	}

	// Case-insensitive match on the listed chain.
	match := db.Lookup(1, "0x1111111111111111111111111111111111111111")
	if !match.Found {
		t.Fatal("phishing address on chain 1 was not found")
	}
	if match.Entry.Category != CategoryPhishing {
		t.Errorf("category = %q, want phishing", match.Entry.Category)
	}
	if match.Severity != 1.0 {
		t.Errorf("phishing severity = %v, want 1.0", match.Severity)
	}

	match = db.Lookup(1, strings.ToUpper("0x1111111111111111111111111111111111111111"))
	if !match.Found {
		t.Error("lookup should be case-insensitive")
	}

	// A chain-scoped entry must not match on a different chain.
	if db.Lookup(56, "0x1111111111111111111111111111111111111111").Found {
		t.Error("chain-1 entry matched on chain 56")
	}

	// A global entry matches on every chain.
	for _, chain := range []uint64{1, 56, 137, 42161} {
		if !db.Lookup(chain, "0x3333333333333333333333333333333333333333").Found {
			t.Errorf("global entry not matched on chain %d", chain)
		}
	}

	// An unknown address is not found. It is not reported as safe either; the
	// caller distinguishes the two.
	if db.Lookup(1, "0x4444444444444444444444444444444444444444").Found {
		t.Error("an unlisted address was reported as a threat")
	}
	if db.Lookup(1, "").Found {
		t.Error("an empty address was reported as a threat")
	}
}

func TestThreatDBStats(t *testing.T) {
	path := writeThreatFile(t, `{
		"entries": {
			"1": [{"address": "0xaa00000000000000000000000000000000000000", "category": "scam"}],
			"global": [{"address": "0xbb00000000000000000000000000000000000000", "category": "mixer"}]
		}
	}`)

	db := NewThreatDB()
	if err := db.Load(path); err != nil {
		t.Fatalf("load: %v", err)
	}

	stats := db.Stats()
	if stats["total_addresses"] != 2 {
		t.Errorf("total_addresses = %v, want 2", stats["total_addresses"])
	}
	if stats["global"] != 1 {
		t.Errorf("global = %v, want 1", stats["global"])
	}
	if stats["loaded"] == nil {
		t.Error("loaded timestamp was not recorded")
	}
}

func TestThreatDBRejectsBadInput(t *testing.T) {
	cases := map[string]string{
		"missing file":     "",
		"invalid json":     `{"entries": `,
		"empty address":    `{"entries": {"1": [{"address": "", "category": "scam"}]}}`,
		"missing category": `{"entries": {"1": [{"address": "0xaa00000000000000000000000000000000000000"}]}}`,
		"bad chain key":    `{"entries": {"notachain": [{"address": "0xaa00000000000000000000000000000000000000", "category": "scam"}]}}`,
	}

	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			db := NewThreatDB()
			var err error
			if body == "" {
				err = db.Load(filepath.Join(t.TempDir(), "does-not-exist.json"))
			} else {
				err = db.Load(writeThreatFile(t, body))
			}
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
		})
	}
}

func TestThreatDBSeverityOrdering(t *testing.T) {
	// A sanctioned address must not score lower than a scam token.
	if threatSeverity(CategorySanctioned) <= threatSeverity(CategoryScam) {
		t.Error("sanctioned severity should exceed scam severity")
	}
	if threatSeverity(CategoryPhishing) < threatSeverity(CategoryScam) {
		t.Error("phishing severity should be at least scam severity")
	}
}

// --- chain RPC URL parsing --------------------------------------------------

func TestParseChainRPCURLs(t *testing.T) {
	got, err := parseChainRPCURLs("1=https://eth.example,137=http://polygon.example:8545")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got[1] != "https://eth.example" {
		t.Errorf("chain 1 = %q", got[1])
	}
	if got[137] != "http://polygon.example:8545" {
		t.Errorf("chain 137 = %q", got[137])
	}

	if empty, err := parseChainRPCURLs("   "); err != nil || len(empty) != 0 {
		t.Errorf("an empty spec should yield no chains, got %v err %v", empty, err)
	}

	bad := []string{
		"no-equals-sign",
		"=https://example.com",
		"abc=https://example.com",
		"1=",
		"1=ftp://example.com",
		"1=example.com",
	}
	for _, spec := range bad {
		if _, err := parseChainRPCURLs(spec); err == nil {
			t.Errorf("spec %q should have been rejected", spec)
		}
	}
}

// --- contract inspection over a real JSON-RPC server ------------------------

// rpcStub is a minimal JSON-RPC endpoint that answers eth_getCode and
// eth_getStorageAt with the values the test supplies.
func rpcStub(t *testing.T, code string, storage string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		var result interface{}
		switch req.Method {
		case "eth_getCode":
			result = code
		case "eth_getStorageAt":
			result = storage
		default:
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"error":   map[string]interface{}{"code": -32601, "message": "method not found"},
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  result,
		})
	}))
}

const zeroStorage = "0x0000000000000000000000000000000000000000000000000000000000000000"

func TestInspectExternallyOwnedAccount(t *testing.T) {
	srv := rpcStub(t, "0x", zeroStorage)
	defer srv.Close()

	checker := NewContractChecker(map[uint64]string{1: srv.URL}, 2*time.Second)
	info, err := checker.Inspect(context.Background(), 1, "0x1111111111111111111111111111111111111111")
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if !info.Checked {
		t.Error("Checked should be true when the chain answered")
	}
	if info.IsContract {
		t.Error("an address with no code is not a contract")
	}
}

func TestInspectPlainContract(t *testing.T) {
	// 10 bytes of runtime code, no EIP-1967 implementation slot.
	srv := rpcStub(t, "0x6001600155", zeroStorage)
	defer srv.Close()

	checker := NewContractChecker(map[uint64]string{1: srv.URL}, 2*time.Second)
	info, err := checker.Inspect(context.Background(), 1, "0x2222222222222222222222222222222222222222")
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if !info.IsContract {
		t.Error("an address with code is a contract")
	}
	if info.CodeSize != 5 {
		t.Errorf("CodeSize = %d, want 5", info.CodeSize)
	}
	if info.IsProxy {
		t.Error("a zero implementation slot means not a proxy")
	}
}

func TestInspectProxyContract(t *testing.T) {
	// The implementation address sits in the low 20 bytes of the slot word.
	const impl = "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"
	slot := "0x000000000000000000000000" + impl[2:]

	srv := rpcStub(t, "0x6001600155", slot)
	defer srv.Close()

	checker := NewContractChecker(map[uint64]string{1: srv.URL}, 2*time.Second)
	info, err := checker.Inspect(context.Background(), 1, "0x3333333333333333333333333333333333333333")
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if !info.IsProxy {
		t.Fatal("a non-zero EIP-1967 slot means the address is a proxy")
	}
	if info.Implementation != impl {
		t.Errorf("Implementation = %q, want %q", info.Implementation, impl)
	}
}

func TestInspectFailsWhenChainIsUnconfigured(t *testing.T) {
	checker := NewContractChecker(map[uint64]string{1: "https://example.invalid"}, 2*time.Second)
	info, err := checker.Inspect(context.Background(), 999, "0x1111111111111111111111111111111111111111")
	if err == nil {
		t.Fatal("an unconfigured chain should report an error, not a result")
	}
	if info.Checked {
		t.Error("Checked must be false when the chain was never asked")
	}
}

func TestInspectFailsWhenNodeIsDown(t *testing.T) {
	srv := rpcStub(t, "0x", zeroStorage)
	url := srv.URL
	srv.Close() // the endpoint is now unreachable

	checker := NewContractChecker(map[uint64]string{1: url}, 500*time.Millisecond)
	info, err := checker.Inspect(context.Background(), 1, "0x1111111111111111111111111111111111111111")
	if err == nil {
		t.Fatal("a dead node should report an error")
	}
	if info.Checked {
		t.Error("Checked must be false when the node did not answer")
	}
}

func TestInspectRejectsNonAddress(t *testing.T) {
	srv := rpcStub(t, "0x", zeroStorage)
	defer srv.Close()

	checker := NewContractChecker(map[uint64]string{1: srv.URL}, 2*time.Second)
	if _, err := checker.Inspect(context.Background(), 1, "not-an-address"); err == nil {
		t.Error("a non-address should be rejected before the RPC call")
	}
}

func TestInspectPropagatesRPCError(t *testing.T) {
	// The stub returns a JSON-RPC error for any method it does not know.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"error":   map[string]interface{}{"code": -32000, "message": "execution reverted"},
		})
	}))
	defer srv.Close()

	checker := NewContractChecker(map[uint64]string{1: srv.URL}, 2*time.Second)
	info, err := checker.Inspect(context.Background(), 1, "0x1111111111111111111111111111111111111111")
	if err == nil {
		t.Fatal("a JSON-RPC error should surface as an error")
	}
	if !strings.Contains(err.Error(), "execution reverted") {
		t.Errorf("error should carry the node's message, got %v", err)
	}
	if info.Checked {
		t.Error("Checked must be false when the call failed")
	}
}

func TestConfiguredChains(t *testing.T) {
	checker := NewContractChecker(map[uint64]string{137: "http://b", 1: "http://a"}, time.Second)
	got := checker.ConfiguredChains()
	if len(got) != 2 || got[0] != 1 || got[1] != 137 {
		t.Errorf("ConfiguredChains = %v, want [1 137] sorted", got)
	}

	var nilChecker *ContractChecker
	if nilChecker.ConfiguredChains() != nil {
		t.Error("a nil checker should report no chains")
	}
}

// --- risk scoring -----------------------------------------------------------

func TestAnalyzeAddressRiskUsesThreatList(t *testing.T) {
	path := writeThreatFile(t, `{
		"entries": {
			"1": [{"address": "0xdead000000000000000000000000000000000000", "category": "drainer", "source": "internal"}]
		}
	}`)
	threats := NewThreatDB()
	if err := threats.Load(path); err != nil {
		t.Fatalf("load: %v", err)
	}

	engine := NewRiskEngine(&Config{}, threats, nil)

	score, desc := engine.analyzeAddressRisk(AnalyzeRequest{
		ChainID:   1,
		ToAddress: "0xdead000000000000000000000000000000000000",
	})
	if score != 1.0 {
		t.Errorf("a drainer address scored %v, want 1.0", score)
	}
	if !strings.Contains(desc, "drainer") {
		t.Errorf("description should name the category, got %q", desc)
	}
}

func TestAnalyzeAddressRiskRejectsMalformedAddress(t *testing.T) {
	engine := NewRiskEngine(&Config{}, NewThreatDB(), nil)

	score, _ := engine.analyzeAddressRisk(AnalyzeRequest{ChainID: 1, ToAddress: "0x123"})
	if score < 0.9 {
		t.Errorf("a malformed address scored %v, want at least 0.9", score)
	}
}

func TestAnalyzeAddressRiskUnverifiedIsNotSafe(t *testing.T) {
	// No threat list entry and no RPC endpoint: the destination is unknown.
	// It must not be scored as though it had been checked and found clean.
	engine := NewRiskEngine(&Config{}, NewThreatDB(), nil)

	score, desc := engine.analyzeAddressRisk(AnalyzeRequest{
		ChainID:   1,
		ToAddress: "0x1111111111111111111111111111111111111111",
	})
	if score < 0.3 {
		t.Errorf("an unverified destination scored %v, want at least 0.3", score)
	}
	if !strings.Contains(desc, "could not be verified") {
		t.Errorf("description should say the destination was unverified, got %q", desc)
	}
}

func TestAnalyzeAddressRiskEOAScoresLow(t *testing.T) {
	srv := rpcStub(t, "0x", zeroStorage)
	defer srv.Close()

	engine := NewRiskEngine(&Config{}, NewThreatDB(),
		NewContractChecker(map[uint64]string{1: srv.URL}, 2*time.Second))

	score, desc := engine.analyzeAddressRisk(AnalyzeRequest{
		ChainID:   1,
		ToAddress: "0x1111111111111111111111111111111111111111",
	})
	if score > 0.1 {
		t.Errorf("a verified EOA scored %v, want a low score", score)
	}
	if !strings.Contains(desc, "externally owned account") {
		t.Errorf("description = %q", desc)
	}
}

func TestAnalyzeNetworkRiskIsNotConstant(t *testing.T) {
	// The old implementation returned 0.2 for every address, which meant the
	// network factor added a fixed amount to every score.
	engine := NewRiskEngine(&Config{}, NewThreatDB(), nil)

	if got := engine.analyzeNetworkRisk("0x1111111111111111111111111111111111111111"); got != 0 {
		t.Errorf("an unlisted address should not carry network risk, got %v", got)
	}
}

func TestIsHexAddress(t *testing.T) {
	valid := []string{
		"0x1111111111111111111111111111111111111111",
		"0xABCDEFabcdefABCDEFabcdefABCDEFabcdefABCD",
	}
	for _, addr := range valid {
		if !isHexAddress(addr) {
			t.Errorf("%q should be a valid address", addr)
		}
	}

	invalid := []string{
		"",
		"0x",
		"1111111111111111111111111111111111111111",   // no 0x prefix
		"0x11111111111111111111111111111111111111",   // too short
		"0x11111111111111111111111111111111111111111", // too long
		"0xzz11111111111111111111111111111111111111", // non-hex
	}
	for _, addr := range invalid {
		if isHexAddress(addr) {
			t.Errorf("%q should not be a valid address", addr)
		}
	}
}
