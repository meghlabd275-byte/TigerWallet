package main

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/tigerwallet/chainsdk"
)

// TestNonEVMParityWithChainsdk guards the MasterWallet non-EVM layer against
// drift from the shared chainsdk. The MasterWallet keeps its own derivation
// entry points (they carry MW-specific path/version handling), but they must
// agree byte-for-byte with the SDK: a silent divergence here means two
// TigerWallet backends derive different addresses for the same seed, which
// would strand user funds.
func TestNonEVMParityWithChainsdk(t *testing.T) {
	seed, _ := hex.DecodeString("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")

	t.Run("addresses", func(t *testing.T) {
		cases := []struct {
			name string
			mw   func() (string, error)
			sdk  func() (string, error)
		}{
			{
				name: "solana",
				mw:   func() (string, error) { return mwSolanaAddressFromSeed(seed, "m/44'/501'/0'/0'") },
				sdk:  func() (string, error) { return chainsdk.SolanaAddressFromSeed(seed, "m/44'/501'/0'/0'") },
			},
			{
				name: "bitcoin-p2pkh",
				mw:   func() (string, error) { return mwBTCAddressFromSeed(seed, "m/44'/0'/0'/0/0") },
				sdk:  func() (string, error) { return chainsdk.BTCAddressFromSeed(seed, "m/44'/0'/0'/0/0") },
			},
			{
				name: "litecoin-version-byte",
				mw:   func() (string, error) { return mwUTXOAddressFromSeed(seed, "m/44'/2'/0'/0/0", 0x30) },
				sdk:  func() (string, error) { return chainsdk.UTXOAddressFromSeed(seed, "m/44'/2'/0'/0/0", "litecoin") },
			},
			{
				name: "dogecoin-version-byte",
				mw:   func() (string, error) { return mwUTXOAddressFromSeed(seed, "m/44'/3'/0'/0/0", 0x1e) },
				sdk:  func() (string, error) { return chainsdk.UTXOAddressFromSeed(seed, "m/44'/3'/0'/0/0", "dogecoin") },
			},
			{
				name: "cosmos",
				mw:   func() (string, error) { return mwCosmosAddressFromSeed(seed, "m/44'/118'/0'/0/0", "cosmos") },
				sdk:  func() (string, error) { return chainsdk.CosmosAddressFromSeed(seed, "m/44'/118'/0'/0/0", "cosmos") },
			},
			{
				name: "cosmos-osmosis-hrp",
				mw:   func() (string, error) { return mwCosmosAddressFromSeed(seed, "m/44'/118'/0'/0/0", "osmo") },
				sdk:  func() (string, error) { return chainsdk.CosmosAddressFromSeed(seed, "m/44'/118'/0'/0/0", "osmo") },
			},
		}
		for _, c := range cases {
			mwAddr, mwErr := c.mw()
			sdkAddr, sdkErr := c.sdk()
			if mwErr != nil || sdkErr != nil {
				t.Fatalf("%s: derivation error: mw=%v sdk=%v", c.name, mwErr, sdkErr)
			}
			if mwAddr != sdkAddr {
				t.Errorf("DIVERGENCE %s: mw=%q sdk=%q", c.name, mwAddr, sdkAddr)
			} else {
				t.Logf("%s agree: %s", c.name, mwAddr)
			}
		}
	})

	t.Run("primitives", func(t *testing.T) {
		payload := []byte{0xde, 0xad, 0xbe, 0xef, 0x00, 0x01, 0x02, 0x03}
		hash20 := []byte{
			0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09,
			0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13,
		}

		if got, want := base58Encode(payload), chainsdk.Base58Encode(payload); got != want {
			t.Errorf("base58Encode: mw=%q sdk=%q", got, want)
		} else {
			t.Logf("base58Encode agree: %s", got)
		}

		// MW takes the version byte separately; chainsdk expects it prepended.
		if got, want := base58CheckEncode(0x00, hash20), chainsdk.Base58CheckEncode(append([]byte{0x00}, hash20...)); got != want {
			t.Errorf("base58CheckEncode: mw=%q sdk=%q", got, want)
		} else {
			t.Logf("base58CheckEncode agree: %s", got)
		}

		mwConv, mwConvErr := convertBits(hash20, 8, 5, true)
		if mwConvErr != nil {
			t.Fatalf("mw convertBits: %v", mwConvErr)
		}
		if got, want := mwConv, mustConvert(t, hash20, 8, 5, true); !bytes.Equal(got, want) {
			t.Errorf("convertBits: mw=%x sdk=%x", got, want)
		} else {
			t.Logf("convertBits agree: %x", got)
		}

		mwB32, mwB32Err := bech32Encode("cosmos", hash20)
		if mwB32Err != nil {
			t.Fatalf("mw bech32Encode: %v", mwB32Err)
		}
		if got, want := mwB32, mustBech32(t, "cosmos", hash20); got != want {
			t.Errorf("bech32Encode: mw=%q sdk=%q", got, want)
		} else {
			t.Logf("bech32Encode agree: %s", got)
		}
	})
}

func mustConvert(t *testing.T, data []byte, from, to uint, pad bool) []byte {
	t.Helper()
	out, err := chainsdk.ConvertBits(data, from, to, pad)
	if err != nil {
		t.Fatalf("ConvertBits: %v", err)
	}
	return out
}

func mustBech32(t *testing.T, hrp string, data []byte) string {
	t.Helper()
	out, err := chainsdk.Bech32Encode(hrp, data)
	if err != nil {
		t.Fatalf("Bech32Encode: %v", err)
	}
	return out
}
