package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Token registration used to invent a contract address with generateHex(40) and
// store it, so the console listed tokens that no node had ever seen. These tests
// pin the replacement: the operator supplies a real address and the handler
// confirms the contract exists before the row is written.

func TestNormalizeEVMAddress(t *testing.T) {
	got, err := normalizeEVMAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	if err != nil {
		t.Fatalf("rejected a valid address: %v", err)
	}
	// Lower-casing keeps the unique index meaningful: the same contract must not
	// be storable twice under different casing.
	if got != "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48" {
		t.Errorf("expected the canonical lower-case form, got %q", got)
	}

	for _, bad := range []string{
		"",
		"0x",
		"a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",       // no 0x
		"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb4",      // 39 nibbles
		"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb488",    // 41 nibbles
		"0xZZb86991c6218b36c1d19d4a2e9eb0ce3606eb48",     // not hex
		"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48; --", // trailing junk
	} {
		if _, err := normalizeEVMAddress(bad); err == nil {
			t.Errorf("accepted %q as an EVM address", bad)
		}
	}
}

func TestChainRPCEndpointFailsClosedForUnknownChain(t *testing.T) {
	t.Setenv("WL_RPC_999999", "")
	if _, err := chainRPCEndpoint(999999); err == nil {
		t.Error("chainRPCEndpoint returned an endpoint for a chain with no configured node")
	}

	// A per-chain override wins over the built-in default.
	t.Setenv("WL_RPC_1", "http://127.0.0.1:8545")
	got, err := chainRPCEndpoint(1)
	if err != nil {
		t.Fatalf("chainRPCEndpoint(1) returned %v", err)
	}
	if got != "http://127.0.0.1:8545" {
		t.Errorf("expected the WL_RPC_1 override, got %q", got)
	}
}

// codeNode serves eth_getCode with a canned result, so the verification path is
// exercised over a real HTTP round trip.
func codeNode(t *testing.T, result string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req wlRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("node received an undecodable request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if req.Method != "eth_getCode" {
			t.Errorf("expected eth_getCode, got %q", req.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0", "id": 1, "result": result,
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestVerifyContractDeployedAcceptsDeployedContract(t *testing.T) {
	srv := codeNode(t, "0x608060405234801561001057600080fd5b50")
	t.Setenv("WL_RPC_1", srv.URL)

	if err := verifyContractDeployed(context.Background(), 1, "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"); err != nil {
		t.Errorf("rejected a deployed contract: %v", err)
	}
}

func TestVerifyContractDeployedRejectsAddressWithNoCode(t *testing.T) {
	// eth_getCode returns "0x" for both a never-deployed address and an
	// externally-owned account; neither is a token contract.
	srv := codeNode(t, "0x")
	t.Setenv("WL_RPC_1", srv.URL)

	if err := verifyContractDeployed(context.Background(), 1, "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"); err == nil {
		t.Error("accepted an address with no contract code")
	}
}

func TestVerifyContractDeployedFailsClosedWhenNodeIsUnreachable(t *testing.T) {
	srv := codeNode(t, "0x6080")
	url := srv.URL
	srv.Close() // the node is now down
	t.Setenv("WL_RPC_1", url)

	if err := verifyContractDeployed(context.Background(), 1, "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"); err == nil {
		t.Error("verification succeeded while the node was unreachable; it must fail closed")
	}
}

func TestVerifyContractDeployedRejectsUnknownChain(t *testing.T) {
	t.Setenv("WL_RPC_999999", "")
	if err := verifyContractDeployed(context.Background(), 999999, "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"); err == nil {
		t.Error("verification succeeded on a chain with no configured node")
	}
}

func TestVerifyContractDeployedRejectsMalformedAddress(t *testing.T) {
	srv := codeNode(t, "0x6080")
	t.Setenv("WL_RPC_1", srv.URL)
	if err := verifyContractDeployed(context.Background(), 1, "not-an-address"); err == nil {
		t.Error("verification accepted a malformed address")
	}
}

// The fabricated address is gone. Pin that no handler invents a contract
// address again, which is what made the console lie about deployed tokens.
func TestNoHandlerFabricatesAContractAddress(t *testing.T) {
	src := readMainSource(t)
	if strings.Contains(src, `"0x" + generateHex(40)`) {
		t.Error("a handler still fabricates a 20-byte contract address")
	}
	for _, fn := range []string{"createToken", "createNewToken"} {
		body := functionBody(t, src, fn)
		if !strings.Contains(body, "normalizeEVMAddress(") {
			t.Errorf("%s does not validate the operator-supplied address", fn)
		}
	}
	if !strings.Contains(functionBody(t, src, "createNewToken"), "verifyContractDeployed(") {
		t.Error("createNewToken does not verify the contract exists on chain before storing it")
	}
}
