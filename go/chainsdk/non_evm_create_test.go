package chainsdk

import (
	"strings"
	"testing"
)

// nonEvmAddressNotDerivable lists the chains whose address cannot be produced
// from an HD seed by this backend, with the reason. They must fail closed with
// a descriptive error: an invented address would resolve to no account and
// would lose any funds sent to it.
//
//   - monero  : keys are sc_reduce(keccak256(seed)) scalars, not BIP32
//   - arweave : address is SHA-256 of an RSA-4096 modulus
//   - starknet: address is a deployed account contract; needs the class hash
//   - ton     : needs the compiled wallet v4r2 state-init cell
//   - aleo/hedera/flow: no HD address scheme (on-chain assignment or a
//     different curve)
//   - handshake/kadena/chia/nexus/ark/aion/nuls/pascalcoin/signum/supra/wakt:
//     chain-specific address encodings with no implementation in this backend
var nonEvmAddressNotDerivable = map[string]string{
	"monero":     "not BIP32/secp256k1",
	"arweave":    "RSA-4096 modulus",
	"starknet":   "needs account class hash",
	"ton":        "needs wallet code cell",
	"aleo":       "no HD address scheme",
	"hedera":     "no HD address scheme",
	"flow":       "no HD address scheme",
	"handshake":  "no address encoding",
	"kadena":     "no address encoding",
	"chia":       "no address encoding",
	"nexus":      "no address encoding",
	"ark":        "no address encoding",
	"aion":       "no address encoding",
	"nuls":       "no address encoding",
	"pascalcoin": "no address encoding",
	"signum":     "no address encoding",
	"supra":      "no address encoding",
	"wakt":       "no address encoding",
}

// TestNonEvmWalletCreationCoversRegistry is the guard for the bug where
// handleCreateWallet consulted only the EVM lookup and answered
// "unsupported chain" for every non-EVM network, so no Bitcoin, Solana,
// Cosmos or Tron wallet could be created.
//
// It walks the whole non-EVM registry and asserts two things: every chain
// either derives a well-formed address or fails closed with a descriptive
// error, and the number of working chains never regresses.
func TestNonEvmWalletCreationCoversRegistry(t *testing.T) {
	seed := seedFromMnemonic(t, abandonMnemonic)

	// Floor measured after fixing the ed25519 derivation paths and removing
	// the entries duplicated from the EVM registry. Raise it when coverage
	// improves; never lower it to make a failure pass.
	const minimumDerivable = 95

	derived, failClosed := 0, map[string]bool{}
	for _, chain := range NonEVMMainnet {
		family, ct, err := ResolveFamily(chain.ChainType, chain.ID)
		if err != nil {
			t.Errorf("%s (id %d, type %q): resolve failed: %v",
				chain.Name, chain.ID, chain.ChainType, err)
			continue
		}
		if strings.TrimSpace(chain.DerivationPath) == "" {
			t.Errorf("%s (id %d): registry has no derivation path", chain.Name, chain.ID)
			continue
		}

		addr, err := AddressFor(family, ct, chain.ID, seed, chain.DerivationPath, 0)
		if err != nil {
			if _, expected := nonEvmAddressNotDerivable[ct]; !expected {
				t.Errorf("%s (id %d, family %s): unexpected failure: %v",
					chain.Name, chain.ID, family, err)
				continue
			}
			failClosed[ct] = true
			continue
		}
		if reason, expected := nonEvmAddressNotDerivable[ct]; expected {
			t.Errorf("%s (id %d): expected fail-closed (%s) but got address %q",
				chain.Name, chain.ID, reason, addr)
			continue
		}
		if strings.TrimSpace(addr) == "" {
			t.Errorf("%s (id %d): empty address", chain.Name, chain.ID)
			continue
		}
		derived++
	}

	if derived < minimumDerivable {
		t.Errorf("only %d chains derived an address, want at least %d", derived, minimumDerivable)
	}
	for ct := range failClosed {
		if _, ok := nonEvmAddressNotDerivable[ct]; !ok {
			t.Errorf("chain type %q failed closed but is not documented", ct)
		}
	}
	t.Logf("%d chains derived, %d chain types failed closed by design", derived, len(failClosed))
}

// TestNonEvmRegistryHasNoEVMDuplicates guards the duplication found during the

// TestNonEvmEd25519PathsAreHardened pins that every ed25519-based chain uses a
// fully hardened path. SLIP-0010 forbids unhardened derivation for ed25519, so
// the registry previously advertised paths that the derivation layer refused.
func TestNonEvmEd25519PathsAreHardened(t *testing.T) {
	ed25519Chains := map[string]bool{
		"algorand": true, "aptos": true, "sui": true, "near": true,
		"nano": true, "banano": true, "stellar": true, "pi": true,
		"tezos": true, "waves": true, "elrond": true, "multiversx": true,
	}
	for _, chain := range NonEVMMainnet {
		ct := normChainType(chain.ChainType)
		if !ed25519Chains[ct] {
			continue
		}
		for _, seg := range strings.Split(chain.DerivationPath, "/") {
			if seg == "" || seg == "m" {
				continue
			}
			if !strings.HasSuffix(seg, "'") {
				t.Errorf("%s (id %d): ed25519 path %q has unhardened segment %q",
					chain.Name, chain.ID, chain.DerivationPath, seg)
			}
		}
	}
}

// TestNonEvmCreationDeterministic pins that the same mnemonic and path always
// yield the same address, so a restored wallet recovers its funds.
func TestNonEvmCreationDeterministic(t *testing.T) {
	seed := seedFromMnemonic(t, abandonMnemonic)
	for _, tc := range []struct {
		ct, path string
		id       int64
	}{
		{"bitcoin", "m/44'/0'/0'/0/0", 9000000000},
		{"solana", "m/44'/501'/0'/0'/0'", 9000000501},
		{"cosmos", "m/44'/118'/0'/0/0", 9000000118},
		{"tron", "m/44'/195'/0'/0/0", 9000000195},
		{"ripple", "m/44'/144'/0'/0/0", 9000000144},
		{"cardano", "m/1852'/1815'/0'/0/0", 9000001815},
	} {
		family, norm, err := ResolveFamily(tc.ct, tc.id)
		if err != nil {
			t.Fatalf("%s: resolve: %v", tc.ct, err)
		}
		first, err := AddressFor(family, norm, tc.id, seed, tc.path, 0)
		if err != nil {
			t.Fatalf("%s: %v", tc.ct, err)
		}
		second, err := AddressFor(family, norm, tc.id, seed, tc.path, 0)
		if err != nil {
			t.Fatalf("%s: %v", tc.ct, err)
		}
		if first != second {
			t.Errorf("%s: not deterministic: %q != %q", tc.ct, first, second)
		}
	}
}

// TestWithAccountIndex covers the trailing-segment rewrite used for account N.
func TestWithAccountIndex(t *testing.T) {
	for _, tc := range []struct {
		in   string
		idx  int
		want string
	}{
		{"m/44'/0'/0'/0/0", 3, "m/44'/0'/0'/0/3"},
		{"m/44'/501'/0'/0'/0'", 2, "m/44'/501'/0'/0'/2'"},
		{"m/1852'/1815'/0'/0/0", 1, "m/1852'/1815'/0'/0/1"},
		{"m/44'/148'/0'", 5, "m/44'/148'/5'"},
	} {
		if got := WithAccountIndex(tc.in, tc.idx); got != tc.want {
			t.Errorf("WithAccountIndex(%q, %d) = %q, want %q", tc.in, tc.idx, got, tc.want)
		}
	}
}
