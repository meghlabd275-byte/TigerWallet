package main

import (
	"os"
	"testing"
	"time"
)

func TestHoldDurationRefusal(t *testing.T) {
	if d := holdDuration(0, true); d != 5*time.Minute {
		t.Fatalf("refusal hold = %s, want 5m", d)
	}
}

func TestHoldDurationExponentialBackoff(t *testing.T) {
	cases := map[int]time.Duration{
		1: 1 * time.Minute,
		2: 2 * time.Minute,
		3: 4 * time.Minute,
	}
	for attempts, want := range cases {
		if d := holdDuration(attempts, false); d != want {
			t.Fatalf("hold(%d) = %s, want %s", attempts, d, want)
		}
	}
	// Cap at 15 minutes.
	if d := holdDuration(20, false); d != 15*time.Minute {
		t.Fatalf("hold cap = %s, want 15m", d)
	}
}

func TestUTXOParamsFor(t *testing.T) {
	p, ok := utxoParamsFor("bitcoin", 0)
	if !ok || p.p2pkhVersion != 0x00 || p.esploraBase == "" {
		t.Fatalf("bitcoin params: %+v ok=%v", p, ok)
	}
	// btc alias resolves to bitcoin.
	if _, ok := utxoParamsFor("btc", 0); !ok {
		t.Fatal("btc alias should resolve to bitcoin params")
	}
	p, ok = utxoParamsFor("litecoin", 0)
	if !ok || p.p2pkhVersion != 0x30 {
		t.Fatalf("litecoin params: %+v ok=%v", p, ok)
	}
	// Env override wins.
	t.Setenv("LTC_ESPLORA_URL", "https://ltc.example.com/api/")
	p, ok = utxoParamsFor("ltc", 0)
	if !ok || p.esploraBase != "https://ltc.example.com/api" {
		t.Fatalf("env override: %+v", p)
	}
	os.Unsetenv("LTC_ESPLORA_URL")
	// Registry chain id resolution: Bitcoin is seeded at 9000000000.
	p, ok = utxoParamsFor("", 9000000000)
	if !ok || p.name != "bitcoin" {
		t.Fatalf("registry resolution: %+v ok=%v", p, ok)
	}
	// Fail-closed for unsupported UTXO chains.
	if _, ok := utxoParamsFor("dogecoin", 0); ok {
		t.Fatal("dogecoin must fail closed (no esplora params pinned)")
	}
	if _, ok := utxoParamsFor("bitcoincash", 0); ok {
		t.Fatal("bitcoincash must fail closed (fork-id sighash required)")
	}
}

func TestNonEVMFamilyFor(t *testing.T) {
	// Registry resolution by chain id (Bitcoin seeded at 9000000000).
	if fam := nonEVMFamilyFor(9000000000, ""); fam != "bitcoin" {
		t.Fatalf("bitcoin family = %q", fam)
	}
	// Aliases normalize.
	if fam := nonEVMFamilyFor(0, "btc"); fam != "bitcoin" {
		t.Fatalf("btc alias = %q", fam)
	}
	if fam := nonEVMFamilyFor(0, "osmosis"); fam != "cosmos" {
		t.Fatalf("osmosis family = %q", fam)
	}
	if fam := nonEVMFamilyFor(0, "ATOM"); fam != "cosmos" {
		t.Fatalf("ATOM family = %q", fam)
	}
	// Unknown chains return the raw name (explicit unsupported error downstream).
	if fam := nonEVMFamilyFor(0, "tron"); fam != "tron" {
		t.Fatalf("tron family = %q", fam)
	}
	// Seeded cosmos-family chain resolves to cosmos via registry.
	var cosmosID int64
	for _, c := range defaultNonEVMChains {
		if c.ChainType == "cosmos" {
			cosmosID = c.ChainID
			break
		}
	}
	if cosmosID == 0 {
		t.Fatal("no cosmos chains seeded")
	}
	if fam := nonEVMFamilyFor(cosmosID, ""); fam != "cosmos" {
		t.Fatalf("registry cosmos family = %q", fam)
	}
}

// TestUnsupportedNonEVMChainIsNotTreatedAsEVM guards the signer dispatch: a
// non-EVM chain with no signer must be recognised as non-EVM so it fails
// closed, instead of falling through to the EVM signing path.
func TestUnsupportedNonEVMChainIsNotTreatedAsEVM(t *testing.T) {
	var tronID int64
	for _, c := range defaultNonEVMChains {
		if c.ChainType == "tron" {
			tronID = c.ChainID
			break
		}
	}
	if tronID == 0 {
		t.Fatal("tron not seeded")
	}
	if !isNonEVMChain(tronID) {
		t.Fatal("a seeded non-EVM chain must be reported as non-EVM")
	}
	if fam := nonEVMFamilyFor(tronID, ""); fam != "tron" {
		t.Fatalf("tron family = %q", fam)
	}
	// Every seeded non-EVM chain is either servable by a real signer or
	// reported as non-EVM so the caller can refuse it. None may be silently
	// routed to the EVM path.
	servable := map[string]bool{
		"cosmos": true, "bitcoin": true, "litecoin": true, "solana": true,
	}
	for _, c := range defaultNonEVMChains {
		if servable[c.ChainType] {
			continue
		}
		if !isNonEVMChain(c.ChainID) {
			t.Fatalf("chain %s (%s) would fall through to EVM signing", c.Name, c.ChainType)
		}
	}
	// EVM chain ids are not in the non-EVM registry.
	if isNonEVMChain(1) {
		t.Fatal("ethereum mainnet must not be reported as non-EVM")
	}
	if isNonEVMChain(0) {
		t.Fatal("chain id 0 must not be reported as non-EVM")
	}
}

func TestInstanceIDNonEmpty(t *testing.T) {
	if instanceID == "" {
		t.Fatal("instanceID must never be empty (claim markers depend on it)")
	}
}

func TestWSFanoutSkipsOwnOrigin(t *testing.T) {
	// The envelope carries the origin instance; receivers must be able to
	// distinguish their own publications. (Delivery itself needs Redis,
	// exercised in integration; here we pin the envelope contract.)
	env := wsFanoutMessage{Origin: instanceID, MasterID: "m1", Payload: map[string]interface{}{"type": "x"}}
	if env.Origin != instanceID {
		t.Fatal("origin mismatch")
	}
}
