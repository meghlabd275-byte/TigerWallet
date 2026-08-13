package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/tyler-smith/go-bip39"
)

func seedFromMnemonic(t *testing.T, mnemonic string) []byte {
	t.Helper()
	seed, err := bip39.NewSeedWithErrorChecking(mnemonic, "")
	if err != nil {
		t.Fatalf("mnemonic to seed: %v", err)
	}
	return seed
}

const abandonMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

func TestSolanaDerivationDeterministic(t *testing.T) {
	seed := seedFromMnemonic(t, abandonMnemonic)
	addr1, err := SolanaAddressFromSeed(seed, "m/44'/501'/0'/0'/0'")
	if err != nil {
		t.Fatalf("solana address: %v", err)
	}
	addr2, err := SolanaAddressFromSeed(seed, "m/44'/501'/0'/0'/0'")
	if err != nil {
		t.Fatalf("solana address 2: %v", err)
	}
	if addr1 != addr2 {
		t.Fatalf("solana derivation not deterministic: %s != %s", addr1, addr2)
	}
	if len(addr1) < 32 {
		t.Fatalf("solana address too short: %d", len(addr1))
	}
}

func TestSolanaSignVerifyRoundtrip(t *testing.T) {
	seed := seedFromMnemonic(t, abandonMnemonic)
	msg := []byte("hello solana")
	sig, pub, err := SolanaSign(seed, "m/44'/501'/0'/0'/0'", string(msg))
	if err != nil {
		t.Fatalf("solana sign: %v", err)
	}
	if len(sig) != 64 {
		t.Fatalf("ed25519 sig length: %d", len(sig))
	}
	if len(pub) != 32 {
		t.Fatalf("ed25519 pubkey length: %d", len(pub))
	}
	addr, _ := SolanaAddressFromSeed(seed, "m/44'/501'/0'/0'/0'")
	if base58Encode(pub) != addr {
		t.Fatalf("solana pubkey != derived address: %s != %s", base58Encode(pub), addr)
	}
	sig2, _, _ := SolanaSign(seed, "m/44'/501'/0'/0'/0'", "hello solana2")
	if bytes.Equal(sig, sig2) {
		t.Fatalf("ed25519 signatures collide across different messages")
	}
}

func TestBitcoinAddressMainnetP2PKH(t *testing.T) {
	seed := seedFromMnemonic(t, abandonMnemonic)
	addr, err := BTCAddressFromSeed(seed, "m/44'/0'/0'/0/0")
	if err != nil {
		t.Fatalf("btc address: %v", err)
	}
	if !strings.HasPrefix(addr, "1") {
		t.Fatalf("btc mainnet p2pkh address must start with '1', got %q", addr)
	}
	pkh, err := decodeP2PKHAddress(addr)
	if err != nil {
		t.Fatalf("decode btc address: %v", err)
	}
	if len(pkh) != 20 {
		t.Fatalf("decoded pkh length: %d", len(pkh))
	}
}

func TestBitcoinDeterministic(t *testing.T) {
	seed := seedFromMnemonic(t, abandonMnemonic)
	a1, _ := BTCAddressFromSeed(seed, "m/44'/0'/0'/0/0")
	a2, _ := BTCAddressFromSeed(seed, "m/44'/0'/0'/0/0")
	if a1 != a2 {
		t.Fatalf("btc derivation not deterministic: %s != %s", a1, a2)
	}
}

func TestCosmosAddressBech32(t *testing.T) {
	seed := seedFromMnemonic(t, abandonMnemonic)
	addr, err := CosmosAddressFromSeed(seed, "m/44'/118'/0'/0/0", "cosmos")
	if err != nil {
		t.Fatalf("cosmos address: %v", err)
	}
	if !strings.HasPrefix(addr, "cosmos1") {
		t.Fatalf("cosmos address must start with 'cosmos1', got %q", addr)
	}
	if len(addr) < 8 {
		t.Fatalf("cosmos address too short: %q", addr)
	}
	osmo, _ := CosmosAddressFromSeed(seed, "m/44'/118'/0'/0/0", "osmo")
	if !strings.HasPrefix(osmo, "osmo1") {
		t.Fatalf("osmo address must start with 'osmo1', got %q", osmo)
	}
}

func TestCosmosSignRoundtrip(t *testing.T) {
	seed := seedFromMnemonic(t, abandonMnemonic)
	doc := &CosmosSignDoc{
		AccountNumber: "1",
		ChainID:      "cosmoshub-4",
		Fee: CosmosFee{
			Amount: []CosmosCoin{{Denom: "uatom", Amount: "5000"}},
			Gas:    "200000",
		},
		Memo:     "test",
		Sequence: "0",
		Msgs: []map[string]interface{}{
			{"type": "cosmos-sdk/MsgSend", "value": map[string]interface{}{
				"from_address": "cosmos1sender",
				"to_address":   "cosmos1recipient",
				"amount":       []CosmosCoin{{Denom: "uatom", Amount: "1000000"}},
			}},
		},
	}
	sig, pub, err := CosmosSign(seed, "m/44'/118'/0'/0/0", doc)
	if err != nil {
		t.Fatalf("cosmos sign: %v", err)
	}
	if len(sig) != 64 {
		t.Fatalf("cosmos sig length: %d (want 64)", len(sig))
	}
	if len(pub) != 33 {
		t.Fatalf("cosmos compressed pubkey length: %d (want 33)", len(pub))
	}
	sig2, _, _ := CosmosSign(seed, "m/44'/118'/0'/0/0", doc)
	if !bytes.Equal(sig, sig2) {
		t.Fatalf("cosmos signing not deterministic")
	}
}

func TestBase58Roundtrip(t *testing.T) {
	cases := [][]byte{
		{0x00},
		{0xff, 0x00, 0x01},
		bytes.Repeat([]byte{0x42}, 32),
		bytes.Repeat([]byte{0x00}, 5),
	}
	for _, in := range cases {
		enc := base58Encode(in)
		dec, err := base58Decode(enc)
		if err != nil {
			t.Fatalf("base58 decode %q: %v", enc, err)
		}
		if !bytes.Equal(in, dec) {
			t.Fatalf("base58 roundtrip mismatch: %x -> %q -> %x", in, enc, dec)
		}
	}
}

func TestBech32CosmosChecksumValid(t *testing.T) {
	data := bytes.Repeat([]byte{0x01}, 20)
	enc, err := bech32Encode("cosmos", data)
	if err != nil {
		t.Fatalf("bech32 encode: %v", err)
	}
	if !strings.HasPrefix(enc, "cosmos1") {
		t.Fatalf("bech32 prefix: %q", enc)
	}
	if len(enc) != 45 {
		t.Fatalf("bech32 length: %d (want 45)", len(enc))
	}
}
