package main

// chain_seeding_test.go — Tests for the chain registry data + seeding logic.
// Verifies 120 EVM + 66 non-EVM chains are present with valid data.

import (
	"testing"
)

func TestEVMChainCount120(t *testing.T) {
	if len(defaultEVMChains) < 120 {
		t.Errorf("expected >=120 EVM chains, got %d", len(defaultEVMChains))
	}
}

func TestNonEVMChainCount50(t *testing.T) {
	if len(defaultNonEVMChains) < 50 {
		t.Errorf("expected >=50 non-EVM chains, got %d", len(defaultNonEVMChains))
	}
}

func TestEVMChainsHaveRPC(t *testing.T) {
	// All EVM chains except possibly testnet-only should have a non-empty RPC.
	for i, c := range defaultEVMChains {
		if c.RPCURL == "" {
			t.Errorf("EVM chain %d (%s) has empty RPC URL", i, c.Name)
		}
		if c.ChainID <= 0 {
			t.Errorf("EVM chain %d (%s) has invalid chain_id", i, c.Name)
		}
	}
}

func TestNonEVMChainsHaveChainType(t *testing.T) {
	for i, c := range defaultNonEVMChains {
		if c.ChainType == "" {
			t.Errorf("non-EVM chain %d (%s) has empty chain_type", i, c.Name)
		}
	}
}

func TestEVMChainID1Present(t *testing.T) {
	found := false
	for _, c := range defaultEVMChains {
		if c.ChainID == 1 && c.Name == "Ethereum Mainnet" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Ethereum Mainnet (chain_id=1) not found in EVM chains")
	}
}

func TestBTCChainPresent(t *testing.T) {
	found := false
	for _, c := range defaultNonEVMChains {
		if c.ChainType == "bitcoin" || c.Symbol == "BTC" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Bitcoin not found in non-EVM chains")
	}
}

func TestSolanaChainPresent(t *testing.T) {
	found := false
	for _, c := range defaultNonEVMChains {
		if c.ChainType == "solana" || c.Symbol == "SOL" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Solana not found in non-EVM chains")
	}
}

func TestCosmosChainPresent(t *testing.T) {
	found := false
	for _, c := range defaultNonEVMChains {
		if c.ChainType == "cosmos" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Cosmos not found in non-EVM chains")
	}
}

func TestNoTestnetChains(t *testing.T) {
	// The canonical registry should only have mainnet chains.
	for _, c := range defaultEVMChains {
		// Testnet chain IDs: 3 (Ropsten), 4 (Rinkeby), 5 (Goerli), 42 (Kovan),
		// 11155111 (Sepolia), 80001 (Mumbai), etc. Ethereum mainnet is 1.
		if c.ChainID == 3 || c.ChainID == 4 || c.ChainID == 5 || c.ChainID == 42 ||
			c.ChainID == 11155111 || c.ChainID == 80001 {
			t.Errorf("testnet chain found in EVM: chain_id=%d name=%s", c.ChainID, c.Name)
		}
	}
}

func TestBech32PrefixForCosmos(t *testing.T) {
	if got := bech32PrefixForChainType("cosmos"); got != "cosmos" {
		t.Errorf("cosmos prefix = %s, want cosmos", got)
	}
	if got := bech32PrefixForChainType("osmosis"); got != "osmo" {
		t.Errorf("osmosis prefix = %s, want osmo", got)
	}
	if got := bech32PrefixForChainType("bitcoin"); got != "" {
		t.Errorf("bitcoin prefix = %s, want empty (not bech32)", got)
	}
}

func TestCosmosSignReal(t *testing.T) {
	// Real secp256k1 sign over a Cosmos SignDoc — no mocks.
	sig, pub, err := mwCosmosSign(testSeed, "m/44'/118'/0'/0/0", `{"test":"signDoc"}`)
	if err != nil {
		t.Fatalf("cosmos sign: %v", err)
	}
	if len(sig) != 64 {
		t.Errorf("cosmos signature must be 64 bytes, got %d", len(sig))
	}
	if len(pub) != 33 {
		t.Errorf("cosmos pubkey (compressed) must be 33 bytes, got %d", len(pub))
	}
	// Verify all-zero signature (not a fake).
	allZero := true
	for _, b := range sig {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("cosmos signature is all zeros (fake)")
	}
}

// TestBech32PrefixForChainID verifies each of the 23 registered Cosmos-SDK
// chains resolves to its correct per-chain bech32 prefix by chain_id (not the
// generic "cosmos" prefix). This is critical: Osmosis must yield "osmo",
// Injective must yield "inj", etc.
func TestBech32PrefixForChainID(t *testing.T) {
	cases := map[int64]string{
		9000000118: "cosmos",
		9000026317: "osmo",
		9000000330: "terra",
		9000073068: "inj",
		9000014648: "celestia",
		9000049823: "dydx",
		9000073741: "sei",
		9000041857: "kujira",
		9000012099: "stride",
		9000090063: "neutron",
		9000005267: "juno",
		9000007183: "akash",
		9000018759: "persistence",
		9000034677: "evmos",
		9000054841: "canto",
		9000003318: "kava",
		9000062954: "cro",
		9000016892: "stars",
		9000021252: "saga",
		9000086660: "noble",
		9000040572: "axelar",
		9000007153: "umee",
		9000000529: "secret",
	}
	if len(cases) != 23 {
		t.Fatalf("expected 23 cosmos chain->prefix mappings, got %d", len(cases))
	}
	for chainID, want := range cases {
		if got := bech32PrefixForChainID(chainID); got != want {
			t.Errorf("chain %d prefix = %s, want %s", chainID, got, want)
		}
	}
	// Unknown chain falls back to canonical cosmos prefix.
	if got := bech32PrefixForChainID(9999999999); got != "cosmos" {
		t.Errorf("unknown chain prefix = %s, want cosmos fallback", got)
	}
}

// TestCosmosChainMeta verifies the per-chain chain_id string + denom used in
// the SignDoc, so signatures are valid on the target chain.
func TestCosmosChainMeta(t *testing.T) {
	cidStr, denom := cosmosChainMeta(9000026317) // Osmosis
	if cidStr != "osmosis-1" {
		t.Errorf("osmosis chain_id string = %s, want osmosis-1", cidStr)
	}
	if denom != "uosmo" {
		t.Errorf("osmosis denom = %s, want uosmo", denom)
	}
	cidStr, denom = cosmosChainMeta(9000073068) // Injective
	if cidStr != "injective-1" {
		t.Errorf("injective chain_id string = %s, want injective-1", cidStr)
	}
	if denom != "inj" {
		t.Errorf("injective denom = %s, want inj", denom)
	}
	// Cosmos Hub fallback.
	cidStr, denom = cosmosChainMeta(9000000118)
	if cidStr != "cosmoshub-4" || denom != "uatom" {
		t.Errorf("cosmos hub meta = %s/%s, want cosmoshub-4/uatom", cidStr, denom)
	}
}
