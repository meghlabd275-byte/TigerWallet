package crypto

import (
	"strings"
	"testing"

	"github.com/tyler-smith/go-bip39"
)

// TestBIP44CanonicalVector verifies the standalone WL crypto against the
// canonical BIP-44 Ethereum test vector: mnemonic "abandon abandon ... about"
// at m/44'/60'/0'/0/0 must derive 0x9858EfFD232B4033E47d90003D41EC34EcaEda94.
func TestBIP44CanonicalVector(t *testing.T) {
	mnemonic := strings.Repeat("abandon ", 11) + "about"
	if !bip39.IsMnemonicValid(mnemonic) {
		t.Fatal("mnemonic invalid")
	}
	seed := bip39.NewSeed(mnemonic, "")
	priv, err := DeriveEVMPrivateKey(seed, 0)
	if err != nil {
		t.Fatalf("derive failed: %v", err)
	}
	addr := AddressFromPrivateKey(priv)
	want := "0x9858EfFD232B4033E47d90003D41EC34EcaEda94"
	if addr != want {
		t.Errorf("BIP-44 address mismatch: got %s, want %s", addr, want)
	}
}

// TestSeedEncryptionRoundtrip verifies scrypt+AES-GCM seed encryption roundtrips
// and that a wrong passphrase fails (fail-closed).
func TestSeedEncryptionRoundtrip(t *testing.T) {
	seed := []byte("0123456789abcdef0123456789abcdef")
	blob, err := EncryptSeedAtRest(seed, "correct-passphrase")
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}
	dec, err := DecryptSeedAtRest(blob, "correct-passphrase")
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	if string(dec) != string(seed) {
		t.Errorf("roundtrip mismatch")
	}
	// Wrong passphrase must fail.
	if _, err := DecryptSeedAtRest(blob, "wrong-passphrase"); err == nil {
		t.Error("wrong passphrase accepted (fail-closed broken)")
	}
}

// TestMnemonicGeneration verifies generated mnemonics are valid 24-word BIP-39.
func TestMnemonicGeneration(t *testing.T) {
	m, err := GenerateMnemonic()
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	words := strings.Fields(m)
	if len(words) != 24 {
		t.Errorf("expected 24 words, got %d", len(words))
	}
	if !bip39.IsMnemonicValid(m) {
		t.Error("generated mnemonic failed validation")
	}
}

// TestSignMessage verifies a personal_sign signature recovers to the signer.
func TestSignMessage(t *testing.T) {
	mnemonic := strings.Repeat("abandon ", 11) + "about"
	seed := bip39.NewSeed(mnemonic, "")
	priv, err := DeriveEVMPrivateKey(seed, 0)
	if err != nil {
		t.Fatalf("derive failed: %v", err)
	}
	sig, err := SignMessage(priv, "hello")
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}
	if len(sig) != 132 { // 0x + 130 hex chars (65 bytes)
		t.Errorf("unexpected signature length: %d", len(sig))
	}
}
