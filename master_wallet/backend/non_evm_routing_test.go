package main

// non_evm_routing_test.go — pins the non-EVM custodial routing table.
//
// The point of these tests is that the dispatch and the set of real broadcast
// paths cannot drift apart: a family is advertised as supported only when a
// builder actually exists, and every seeded chain that has no builder is
// reported as unsupported rather than being signed with the wrong scheme.

import (
	"sort"
	"strings"
	"testing"

	"github.com/tigerwallet/chainsdk"
)

// TestNonEVMHasSignerMatchesBuilders asserts nonEVMHasSigner agrees with the
// two real dispatch paths (the UTXO chain table and the chainsdk family list).
func TestNonEVMHasSignerMatchesBuilders(t *testing.T) {
	for _, c := range defaultNonEVMChains {
		family := strings.ToLower(c.ChainType)
		want := false
		switch family {
		case "solana", "cosmos", "cardano":
			want = true
		default:
			if _, ok := utxoParamsFor(family, c.ChainID); ok {
				want = true
			} else if chainsdkFamilySupported(family) {
				want = true
			}
		}
		if got := nonEVMHasSigner(family); got != want {
			t.Errorf("chain %d (%s, family %q): nonEVMHasSigner=%v, want %v",
				c.ChainID, c.Name, family, got, want)
		}
	}
}

// TestEverySeededFamilyIsClassified asserts each seeded chain either has a
// signer or is explicitly unsupported — there is no third state.
func TestEverySeededFamilyIsClassified(t *testing.T) {
	supported := map[string]int{}
	unsupported := map[string]int{}
	for _, c := range defaultNonEVMChains {
		family := strings.ToLower(c.ChainType)
		if nonEVMHasSigner(family) {
			supported[family]++
		} else {
			unsupported[family]++
		}
	}
	if len(supported) == 0 {
		t.Fatal("no seeded non-EVM family has a signer")
	}
	// The families that must be supported at launch: the top non-EVM chains, by
	// the name the registry seeds them under.
	required := []string{
		"bitcoin", "litecoin", "dogecoin", "dash", "zcash", "bitcoincash",
		"solana", "cosmos", "cardano", "tron", "ripple", "stellar", "near",
		"algorand", "tezos", "vechain", "aptos", "sui", "elrond",
	}
	for _, f := range required {
		if supported[f] == 0 {
			t.Errorf("required family %q has no signer", f)
		}
	}
	var unsupportedNames []string
	for f := range unsupported {
		unsupportedNames = append(unsupportedNames, f)
	}
	sort.Strings(unsupportedNames)
	t.Logf("supported families: %d, explicitly unsupported: %v", len(supported), unsupportedNames)
}

// TestChainsdkFamilySupportedHasBuilder asserts the advertised chainsdk
// families are exactly the ones the autoSignChainSDK switch handles.
func TestChainsdkFamilySupportedHasBuilder(t *testing.T) {
	// Keep in step with the switch in autoSignChainSDK.
	handled := []string{"tron", "sui", "ripple", "vechain", "aptos", "near",
		"algorand", "stellar", "tezos", "multiversx", "elrond"}
	for _, f := range handled {
		if !chainsdkFamilySupported(f) {
			t.Errorf("family %q is handled by autoSignChainSDK but not advertised", f)
		}
	}
	for _, f := range []string{"xrp", "pi"} {
		if !chainsdkFamilySupported(f) {
			t.Errorf("alias %q must resolve to a supported family", f)
		}
	}
	// A family with no builder must not be advertised.
	for _, f := range []string{"ton", "hedera", "flow", "aleo", "nano", "kaspa"} {
		if chainsdkFamilySupported(f) {
			t.Errorf("family %q is advertised but has no builder in the switch", f)
		}
	}
}

// TestChainsdkAddressForCoversEveryRoutedFamily asserts an address can be
// derived for every family routed to autoSignChainSDK — a transfer cannot be
// built without the sender address.
func TestChainsdkAddressForCoversEveryRoutedFamily(t *testing.T) {
	seed := []byte("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about")
	// Use each chain's own seeded derivation path: the ed25519 families require
	// fully hardened paths, so a hand-written one would be rejected by design.
	checked := map[string]bool{}
	for _, c := range defaultNonEVMChains {
		family := strings.ToLower(c.ChainType)
		if !chainsdkFamilySupported(family) || checked[family] {
			continue
		}
		if c.DerivationPath == "" {
			continue
		}
		checked[family] = true
		addr, err := chainsdkAddressFor(family, seed, c.DerivationPath)
		if err != nil {
			t.Errorf("family %q (%s): %v", family, c.Name, err)
			continue
		}
		if strings.TrimSpace(addr) == "" {
			t.Errorf("family %q (%s): empty address", family, c.Name)
		}
	}
	if len(checked) == 0 {
		t.Fatal("no chainsdk family was checked")
	}
	t.Logf("derived an address for %d chainsdk families", len(checked))
	if _, err := chainsdkAddressFor("ton", seed, "m/44'/607'/0'/0/0"); err == nil {
		t.Error("an unsupported family must fail address derivation closed")
	}
}

// TestSeededEd25519PathsAreHardened is a funds-correctness guard. SLIP-0010
// forbids unhardened derivation for ed25519, so a seeded path with a plain
// segment makes that chain impossible to sweep: every signature attempt fails
// before a transaction is built. Ten registry rows shipped that way.
func TestSeededEd25519PathsAreHardened(t *testing.T) {
	// Cardano is excluded on purpose: it uses BIP32-Ed25519 (CIP-1852), which
	// does permit soft derivation, so its m/1852'/1815'/0'/0/0 path is valid.
	ed25519Families := map[string]bool{
		"algorand": true, "aptos": true, "sui": true, "near": true,
		"nano": true, "stellar": true, "pi": true, "tezos": true,
		"waves": true, "elrond": true, "multiversx": true,
	}
	for _, c := range defaultNonEVMChains {
		family := strings.ToLower(c.ChainType)
		if !ed25519Families[family] || c.DerivationPath == "" {
			continue
		}
		for _, seg := range strings.Split(c.DerivationPath, "/") {
			if seg == "" || seg == "m" {
				continue
			}
			if !strings.HasSuffix(seg, "'") {
				t.Errorf("%s (id %d, %s): ed25519 path %q has unhardened segment %q",
					c.Name, c.ChainID, family, c.DerivationPath, seg)
			}
		}
	}
}

// TestSeededPathsMatchSharedSDK asserts the MasterWallet registry and the
// shared chainsdk registry agree on every chain they both seed. A divergence
// means the address derived at creation differs from the address the signer
// derives later, which would make a balance unreachable.
func TestSeededPathsMatchSharedSDK(t *testing.T) {
	shared := map[int64]string{}
	for _, c := range chainsdk.NonEVMMainnet {
		shared[c.ID] = c.DerivationPath
	}
	var compared int
	for _, c := range defaultNonEVMChains {
		want, ok := shared[c.ChainID]
		if !ok || want == "" || c.DerivationPath == "" {
			continue
		}
		compared++
		if c.DerivationPath != want {
			t.Errorf("%s (id %d): master_wallet path %q, shared SDK path %q",
				c.Name, c.ChainID, c.DerivationPath, want)
		}
	}
	if compared == 0 {
		t.Fatal("no shared chain was compared; the registries no longer overlap")
	}
	t.Logf("compared %d seeded derivation paths against the shared SDK registry", compared)
}

// TestCardanoFeeIsTheLedgerMinimum pins the ledger fee formula so a change to
// the constants is deliberate.
func TestCardanoFeeIsTheLedgerMinimum(t *testing.T) {
	// a=155381, b=44: a 300-byte transaction costs 155381 + 44*300.
	if got, want := cardanoFeeForSize(300), uint64(155381+44*300); got != want {
		t.Fatalf("fee for 300 bytes = %d, want %d", got, want)
	}
	if got, want := cardanoFeeForSize(0), uint64(155381); got != want {
		t.Fatalf("fee for 0 bytes = %d, want the fixed component %d", got, want)
	}
}

// TestSelectCardanoInputsCoversTarget asserts the selector is deterministic and
// only reports coverage when the chosen inputs really cover the target.
func TestSelectCardanoInputsCoversTarget(t *testing.T) {
	utxos := []chainsdk.CardanoUTXO{
		{TxHash: "aa", Index: 0, Amount: 5000000},
		{TxHash: "bb", Index: 1, Amount: 2000000},
		{TxHash: "cc", Index: 2, Amount: 9000000},
	}
	chosen, total := selectCardanoInputs(utxos, 10000000)
	if total < 10000000 {
		t.Fatalf("selected %d lovelace, target was 10000000", total)
	}
	// Largest first: the 9 ADA output plus the 5 ADA output, not the 2 ADA one.
	if len(chosen) != 2 {
		t.Fatalf("chose %d inputs, want 2", len(chosen))
	}
	if chosen[0].Amount != 9000000 {
		t.Fatalf("first input is %d, want the largest (9000000)", chosen[0].Amount)
	}

	// A target the set cannot cover is reported honestly.
	_, total = selectCardanoInputs(utxos, 100000000)
	if total != 16000000 {
		t.Fatalf("total = %d, want the full 16000000 when the target is unreachable", total)
	}
}

// TestCardanoUTXOIsPureLovelace is a regression guard: a UTXO carrying a
// native asset must not be spent by a plain ADA transfer, because doing so
// destroys the asset.
func TestCardanoUTXOIsPureLovelace(t *testing.T) {
	amounts := func(units ...string) []cardanoAmount {
		out := make([]cardanoAmount, 0, len(units))
		for _, u := range units {
			out = append(out, cardanoAmount{Unit: u, Quantity: "1"})
		}
		return out
	}
	cases := []struct {
		name string
		row  cardanoUTXO
		want bool
	}{
		{"ada only", cardanoUTXO{TxHash: "aa", Amount: amounts("lovelace")}, true},
		{"ada plus native asset", cardanoUTXO{TxHash: "bb", Amount: amounts("lovelace", "abc")}, false},
		{"native asset only", cardanoUTXO{TxHash: "cc", Amount: amounts("abc")}, false},
		{"no amount entries", cardanoUTXO{TxHash: "dd", Amount: amounts()}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := cardanoUTXOIsPureLovelace(tc.row); got != tc.want {
				t.Fatalf("cardanoUTXOIsPureLovelace = %v, want %v", got, tc.want)
			}
		})
	}
}
