package main

// crypto_core_test.go — verifies the real crypto primitives against known
// BIP-39/32/44 test vectors. No mocks; real secp256k1/keccak.

import (
	"strings"
	"testing"
)

// TestBIP44AbandonVector verifies the canonical BIP-44 test vector:
// mnemonic "abandon abandon ... about" at m/44'/60'/0'/0/0 derives the
// well-known address 0x9858EfFD232B4033E47d90003D41EC34EcaEda94.
func TestBIP44AbandonVector(t *testing.T) {
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	if !ValidateMnemonic(mnemonic) {
		t.Fatal("mnemonic failed BIP-39 validation")
	}
	seed := MnemonicToSeed(mnemonic, "")
	priv, err := DeriveEVMPrivateKey(seed, 0)
	if err != nil {
		t.Fatalf("derive key: %v", err)
	}
	addr := PrivateKeyToAddress(priv)
	expected := "0x9858EfFD232B4033E47d90003D41EC34EcaEda94"
	if !strings.EqualFold(addr.Hex(), expected) {
		t.Fatalf("address mismatch: got %s want %s", addr.Hex(), expected)
	}
}

// TestSeedEncryptDecryptRoundtrip verifies the scrypt+AES-GCM seed encryption
// round-trips and that a wrong password fails (GCM auth tag).
func TestSeedEncryptDecryptRoundtrip(t *testing.T) {
	seed := MnemonicToSeed("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about", "")
	enc, err := EncryptSeed(seed, "correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	dec, err := DecryptSeed(enc, "correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(dec) != string(seed) {
		t.Fatal("decrypted seed does not match original")
	}
	// Wrong password must fail.
	if _, err := DecryptSeed(enc, "wrong-password"); err == nil {
		t.Fatal("wrong password should have failed decryption")
	}
}

// TestMnemonicGeneration verifies generated mnemonics are valid BIP-39.
func TestMnemonicGeneration(t *testing.T) {
	m, err := GenerateMnemonic(256)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	words := strings.Fields(m)
	if len(words) != 24 {
		t.Fatalf("expected 24 words, got %d", len(words))
	}
	if !ValidateMnemonic(m) {
		t.Fatal("generated mnemonic failed validation")
	}
}
