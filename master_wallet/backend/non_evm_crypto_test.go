package main

// non_evm_crypto_test.go — Tests for real non-EVM address derivation + signing.
// Uses the canonical BIP-39 "abandon...about" mnemonic (no mocks).

import (
	"testing"
)

// testSeed is the deterministic seed from "abandon abandon ... about".
var testSeed = MnemonicToSeed("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about", "")

func TestSolanaAddressFromSeed(t *testing.T) {
	// SLIP-0010 Ed25519 requires ALL path segments hardened.
	addr, err := mwSolanaAddressFromSeed(testSeed, "m/44'/501'/0'/0'/0'")
	if err != nil {
		t.Fatalf("Solana address derivation: %v", err)
	}
	if len(addr) < 32 || len(addr) > 44 {
		t.Errorf("Solana address length invalid: %d (%s)", len(addr), addr)
	}
	// Solana addresses are base58 — verify it only contains base58 chars.
	for _, c := range addr {
		if !isBase58Char(byte(c)) {
			t.Errorf("Solana address contains non-base58 char: %c in %s", c, addr)
		}
	}
}

func TestSolanaSignAndVerify(t *testing.T) {
	sig, pub, err := mwSolanaSign(testSeed, "m/44'/501'/0'/0'/0'", "test message")
	if err != nil {
		t.Fatalf("Solana sign: %v", err)
	}
	if len(sig) != 64 {
		t.Errorf("Ed25519 signature must be 64 bytes, got %d", len(sig))
	}
	if len(pub) != 32 {
		t.Errorf("Ed25519 public key must be 32 bytes, got %d", len(pub))
	}
	// Verify the signature (real Ed25519 verify).
	if !verifyEd25519Sig(pub, []byte("test message"), sig) {
		t.Error("Solana Ed25519 signature verification failed")
	}
	// Tamper detection.
	if verifyEd25519Sig(pub, []byte("tampered"), sig) {
		t.Error("Ed25519 signature should NOT verify for tampered message")
	}
}

func TestBTCAddressFromSeed(t *testing.T) {
	addr, err := mwBTCAddressFromSeed(testSeed, "m/44'/0'/0'/0/0")
	if err != nil {
		t.Fatalf("BTC address derivation: %v", err)
	}
	// Bitcoin mainnet P2PKH addresses start with '1'.
	if len(addr) == 0 || addr[0] != '1' {
		t.Errorf("BTC mainnet P2PKH address should start with '1', got: %s", addr)
	}
	if len(addr) < 26 || len(addr) > 35 {
		t.Errorf("BTC address length invalid: %d (%s)", len(addr), addr)
	}
}

func TestCosmosAddressFromSeed(t *testing.T) {
	addr, err := mwCosmosAddressFromSeed(testSeed, "m/44'/118'/0'/0/0", "cosmos")
	if err != nil {
		t.Fatalf("Cosmos address derivation: %v", err)
	}
	if len(addr) < 38 || len(addr) > 45 {
		t.Errorf("Cosmos address length invalid: %d (%s)", len(addr), addr)
	}
	// Cosmos addresses start with "cosmos1".
	if len(addr) < 7 || addr[:7] != "cosmos1" {
		t.Errorf("Cosmos address should start with 'cosmos1', got: %s", addr)
	}
}

func TestCosmosOsmosisPrefix(t *testing.T) {
	addr, err := mwCosmosAddressFromSeed(testSeed, "m/44'/118'/0'/0/0", "osmo")
	if err != nil {
		t.Fatalf("Osmosis address derivation: %v", err)
	}
	if len(addr) < 5 || addr[:4] != "osmo" {
		t.Errorf("Osmosis address should start with 'osmo', got: %s", addr)
	}
}

func TestSLIP10PathParsing(t *testing.T) {
	indices, err := parseSLIP10Path("m/44'/501'/0'/0/0")
	if err != nil {
		t.Fatalf("path parse: %v", err)
	}
	if len(indices) != 5 {
		t.Fatalf("expected 5 indices, got %d", len(indices))
	}
	// First 3 should be hardened (>= 0x80000000)
	if indices[0] < 0x80000000 || indices[1] < 0x80000000 {
		t.Error("first two indices should be hardened")
	}
}

func TestSLIP10RejectsNonHardenedForEd25519(t *testing.T) {
	// Ed25519 SLIP-10 only allows hardened derivation.
	_, err := slip10DeriveEd25519MW(testSeed, "m/44'/501'/0/0/0")
	if err == nil {
		t.Error("should reject non-hardened Ed25519 derivation")
	}
}

func TestBase58CheckEncoding(t *testing.T) {
	// Known test vector: empty payload with version 0x00 -> "1" (but that's
	// for all-zero). Test with a known 20-byte hash.
	hash := make([]byte, 20)
	hash[0] = 0x00
	hash[19] = 0xff
	addr := base58CheckEncode(0x00, hash)
	if len(addr) < 26 || len(addr) > 35 {
		t.Errorf("base58check address length invalid: %d (%s)", len(addr), addr)
	}
}

// isBase58Char checks if a byte is a valid base58 character.
func isBase58Char(c byte) bool {
	const base58 = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	for _, b := range []byte(base58) {
		if c == b {
			return true
		}
	}
	return false
}

// verifyEd25519Sig wraps the ed25519 Verify function.
func verifyEd25519Sig(pub, message, sig []byte) bool {
	return ed25519Verify(pub, message, sig)
}
