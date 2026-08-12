package main

// wallet_engine_test.go — verifies the REAL crypto pipeline:
// BIP-39 mnemonic -> BIP-32 HD derivation -> secp256k1 key -> EVM address.
// Uses the canonical BIP-44 test vector from the BIP-44 spec (ethereum.org).

import (
	"math/big"
	"strings"
	"testing"
)

// TestBIP39MnemonicGeneration verifies a real mnemonic is generated and valid.
func TestBIP39MnemonicGeneration(t *testing.T) {
	mnemonic, err := GenerateMnemonic(256)
	if err != nil {
		t.Fatalf("GenerateMnemonic failed: %v", err)
	}
	words := strings.Fields(mnemonic)
	if len(words) != 24 {
		t.Errorf("expected 24 words, got %d", len(words))
	}
	if !ValidateMnemonic(mnemonic) {
		t.Error("generated mnemonic failed BIP-39 validation")
	}
}

func TestBIP3912WordMnemonic(t *testing.T) {
	mnemonic, err := GenerateMnemonic(128)
	if err != nil {
		t.Fatalf("GenerateMnemonic(128) failed: %v", err)
	}
	words := strings.Fields(mnemonic)
	if len(words) != 12 {
		t.Errorf("expected 12 words, got %d", len(words))
	}
	if !ValidateMnemonic(mnemonic) {
		t.Error("12-word mnemonic failed validation")
	}
}

// TestBIP44TestVector verifies derivation against the canonical BIP-44 Ethereum
// test vector: mnemonic "abandon abandon abandon ... art" m/44'/60'/0'/0/0
// must derive address 0x9858EfFD232B4033E47d90003D41EC34EcaEda94.
func TestBIP44TestVector(t *testing.T) {
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	if !ValidateMnemonic(mnemonic) {
		t.Fatal("test vector mnemonic invalid")
	}
	chain := ChainConfig{ID: 1, DerivationPath: "m/44'/60'/0'/0/0"}
	privKey, err := DeriveEVMPrivateKey(mnemonic, chain, 0)
	if err != nil {
		t.Fatalf("DeriveEVMPrivateKey failed: %v", err)
	}
	addr := PrivateKeyToAddress(privKey)
	// Canonical address from BIP-44 ethereum test vector
	expected := "0x9858EfFD232B4033E47d90003D41EC34EcaEda94"
	if addr.Hex() != expected {
		t.Errorf("BIP-44 test vector mismatch:\n  got:      %s\n  expected: %s", addr.Hex(), expected)
	}
}

// TestEncryptDecryptSeed verifies the AES-GCM/scrypt seed encryption round-trips.
func TestEncryptDecryptSeed(t *testing.T) {
	seed := []byte("test-seed-for-encryption-round-trip-32bytes!")
	// pad to 64
	seed64 := make([]byte, 64)
	copy(seed64, seed)
	password := "super-secret-password-123"
	enc, err := EncryptSeed(seed64, password)
	if err != nil {
		t.Fatalf("EncryptSeed failed: %v", err)
	}
	if enc == "" || len(enc) < 128 {
		t.Errorf("ciphertext is empty or too short: len=%d", len(enc))
	}
	dec, err := DecryptSeed(enc, password)
	if err != nil {
		t.Fatalf("DecryptSeed failed: %v", err)
	}
	if string(dec) != string(seed64) {
		t.Error("decrypted seed does not match original")
	}
}

// TestDecryptSeedWrongPassword ensures wrong password fails (auth tag mismatch).
func TestDecryptSeedWrongPassword(t *testing.T) {
	seed64 := make([]byte, 64)
	copy(seed64, []byte("another-seed-value-for-testing!!"))
	enc, _ := EncryptSeed(seed64, "correct-password")
	_, err := DecryptSeed(enc, "wrong-password")
	if err == nil {
		t.Error("DecryptSeed should fail with wrong password")
	}
}

// TestSignPersonalMessage verifies a real ECDSA signature is produced.
func TestSignPersonalMessage(t *testing.T) {
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	chain := ChainConfig{ID: 1, DerivationPath: "m/44'/60'/0'/0/0"}
	priv, err := DeriveEVMPrivateKey(mnemonic, chain, 0)
	if err != nil {
		t.Fatalf("derive failed: %v", err)
	}
	sig, err := SignPersonalMessage(priv, []byte("Hello TigerWallet"))
	if err != nil {
		t.Fatalf("SignPersonalMessage failed: %v", err)
	}
	if len(sig) != 65 {
		t.Errorf("expected 65-byte signature, got %d", len(sig))
	}
	// v must be 27 or 28 for personal_sign
	if sig[64] != 27 && sig[64] != 28 {
		t.Errorf("expected recovery byte 27/28, got %d", sig[64])
	}
}

// TestParsePath verifies BIP-32 path parsing.
func TestParsePath(t *testing.T) {
	indices, err := parsePath("m/44'/60'/0'/0/0")
	if err != nil {
		t.Fatalf("parsePath failed: %v", err)
	}
	if len(indices) != 5 {
		t.Fatalf("expected 5 segments, got %d", len(indices))
	}
	if indices[0] != 44+hardening {
		t.Errorf("expected 44', got %d", indices[0])
	}
	if indices[1] != 60+hardening {
		t.Errorf("expected 60', got %d", indices[1])
	}
	if indices[4] != 0 {
		t.Errorf("expected 0, got %d", indices[4])
	}
}

// TestParsePathHardenedVariants verifies ' / h / H suffixes.
func TestParsePathHardenedVariants(t *testing.T) {
	for _, p := range []string{"m/44'/60'/0'/0/0", "m/44h/60h/0h/0/0", "m/44H/60H/0H/0/0"} {
		idx, err := parsePath(p)
		if err != nil {
			t.Errorf("parsePath(%q) failed: %v", p, err)
		}
		if idx[0] != 44+hardening {
			t.Errorf("parsePath(%q): expected hardened 44, got %d", p, idx[0])
		}
	}
}

// TestWeiToFloat verifies unit conversion.
func TestWeiToFloat(t *testing.T) {
	// 1 ether = 1e18 wei
	big18 := new(big.Int)
	big18.SetString("1000000000000000000", 10)
	f := weiToFloat(big18, 18)
	if f != 1.0 {
		t.Errorf("expected 1.0 ETH, got %f", f)
	}
}

// TestTokenRegistry verifies tokens are registered for mainnet.
func TestTokenRegistry(t *testing.T) {
	tokens := tokensForChain(1)
	if len(tokens) == 0 {
		t.Error("no tokens registered for Ethereum mainnet")
	}
	found := false
	for _, tk := range tokens {
		if tk.Symbol == "USDC" {
			found = true
			if tk.Decimals != 6 {
				t.Errorf("USDC decimals expected 6, got %d", tk.Decimals)
			}
		}
	}
	if !found {
		t.Error("USDC not in mainnet token registry")
	}
}

// TestSupportedChains verifies the chain registry.
func TestSupportedChains(t *testing.T) {
	if c := chainByID(1); c == nil || c.Name != "Ethereum Mainnet" {
		t.Error("Ethereum Mainnet chain not found")
	}
	if c := chainByID(137); c == nil || c.Name != "Polygon Mainnet" {
		t.Error("Polygon chain not found")
	}
	if c := chainByID(99999); c != nil {
		t.Error("unknown chain should return nil")
	}
	// Verify the preinstalled mainnet minimums: >=100 EVM, >=50 non-EVM.
	if n := evmChainCount(); n < 100 {
		t.Errorf("expected >=100 EVM mainnet chains, got %d", n)
	}
	if n := nonEvmChainCount(); n < 50 {
		t.Errorf("expected >=50 non-EVM mainnet chains, got %d", n)
	}
	// Verify Pi Network is preinstalled (per requirement).
	piFound := false
	for _, c := range listSupportedChains() {
		if c.ChainType == "pi" {
			piFound = true
			break
		}
	}
	if !piFound {
		t.Error("Pi Network chain not found")
	}
	// Verify no testnets shipped in the static registry.
	for _, c := range listSupportedChains() {
		if c.IsTestnet {
			t.Errorf("static registry must be mainnet-only; found testnet: %s", c.Name)
		}
	}
}
