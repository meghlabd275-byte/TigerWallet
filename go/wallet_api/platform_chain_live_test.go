package main

// platform_chain_live_test.go — end-to-end checks against a running
// TigerSmartChain node.
//
// These tests are skipped unless TSC_RPC_URL points at a live node, so the
// normal unit run stays hermetic. When it is set they exercise the same code
// path a user hits: resolve the chain through the public registry, then read
// the native TGR balance and the RUSD/WTGR predeploy balances over real
// JSON-RPC. Nothing here is mocked - a wrong address, a missing predeploy or a
// broken endpoint fails the test instead of being papered over.

import (
	"context"

	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// liveTSCEndpoint returns the configured endpoint and skips the test when the
// operator has not pointed the suite at a node.
func liveTSCEndpoint(t *testing.T) string {
	t.Helper()
	url := os.Getenv("TSC_RPC_URL")
	if url == "" {
		t.Skip("TSC_RPC_URL is not set; skipping live TigerSmartChain checks")
	}
	return url
}

// TestLiveTSCChainRegistryResolvesEndpoint proves the public chain registry
// carries the operator's endpoint. Before this was fixed the entry shipped with
// an empty rpc_endpoint, so every client rendered TigerSmartChain as
// unreachable even while a node was running.
func TestLiveTSCChainRegistryResolvesEndpoint(t *testing.T) {
	url := liveTSCEndpoint(t)

	var found *ChainConfig
	for _, c := range listSupportedChains() {
		if c.ID == tigerSmartChainID {
			cc := c
			found = &cc
			break
		}
	}
	if found == nil {
		t.Fatalf("chain %d is not in the public registry", tigerSmartChainID)
	}
	if found.RPCEndpoint != url {
		t.Fatalf("registry rpc_endpoint = %q, want %q", found.RPCEndpoint, url)
	}
	if found.Symbol != tgrNativeSymbol {
		t.Fatalf("registry symbol = %q, want %q", found.Symbol, tgrNativeSymbol)
	}
}

// TestLiveTSCNativeBalance reads the TGR balance of the funded devnet
// validator. TGR is the native coin, so it must be readable through the same
// fetcher every other EVM chain uses.
func TestLiveTSCNativeBalance(t *testing.T) {
	url := liveTSCEndpoint(t)

	addr := os.Getenv("TSC_TEST_ADDRESS")
	if addr == "" {
		t.Skip("TSC_TEST_ADDRESS is not set; skipping live balance check")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	bal, err := FetchNativeBalance(ctx, url, common.HexToAddress(addr))
	if err != nil {
		t.Fatalf("FetchNativeBalance: %v", err)
	}
	if bal == nil {
		t.Fatal("FetchNativeBalance returned a nil balance with no error")
	}
	if bal.Sign() <= 0 {
		t.Fatalf("TGR balance = %s, want > 0 for the funded validator", bal)
	}
}

// TestLiveTSCRUSDPredeployBalance reads RUSD and WTGR through the real ERC-20
// balanceOf call. Both are genesis predeploys, so their code must exist from
// block zero and balanceOf must answer with a decoded uint256 rather than
// reverting or returning empty data. That is what makes the addresses in the
// token registry safe for every client to hard-code.
func TestLiveTSCRUSDPredeployBalance(t *testing.T) {
	url := liveTSCEndpoint(t)

	addr := os.Getenv("TSC_TEST_ADDRESS")
	if addr == "" {
		t.Skip("TSC_TEST_ADDRESS is not set; skipping live predeploy check")
	}

	tokens := defaultTokenRegistry[tigerSmartChainID]
	if len(tokens) == 0 {
		t.Fatalf("no tokens registered for chain %d", tigerSmartChainID)
	}

	holder := common.HexToAddress(addr)
	sawRUSD := false
	for _, tok := range tokens {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		bal, err := FetchERC20Balance(ctx, url, common.HexToAddress(tok.Contract), holder)
		cancel()
		if err != nil {
			t.Fatalf("%s (%s) is registered but balanceOf failed: %v",
				tok.Symbol, tok.Contract, err)
		}
		if bal == nil {
			t.Fatalf("%s balanceOf returned nil with no error", tok.Symbol)
		}
		if tok.Symbol == "RUSD" {
			sawRUSD = true
		}
	}
	if !sawRUSD {
		t.Fatal("RUSD was not among the platform token registry entries")
	}
}
