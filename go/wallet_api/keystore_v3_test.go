package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

// TestKeystoreV3RoundTrip verifies export -> import returns the same key.
func TestKeystoreV3RoundTrip(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	wantAddr := crypto.PubkeyToAddress(key.PublicKey)

	blob, err := ExportKeystoreV3(key, "correct horse battery staple")
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	// Validate JSON structure + version.
	var ks KeystoreV3
	if err := json.Unmarshal(blob, &ks); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ks.Version != 3 {
		t.Fatalf("version = %d, want 3", ks.Version)
	}
	if ks.Crypto.KDF != "scrypt" {
		t.Fatalf("kdf = %q, want scrypt", ks.Crypto.KDF)
	}
	if ks.Crypto.Cipher != "aes-128-ctr" {
		t.Fatalf("cipher = %q, want aes-128-ctr", ks.Crypto.Cipher)
	}
	if !strings.EqualFold(ks.Address, wantAddr.Hex()) {
		t.Fatalf("address = %q, want %q", ks.Address, wantAddr.Hex())
	}

	// Import back with the correct password.
	got, err := ImportKeystoreV3(blob, "correct horse battery staple")
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	gotAddr := crypto.PubkeyToAddress(got.PublicKey)
	if gotAddr != wantAddr {
		t.Fatalf("imported address = %s, want %s", gotAddr, wantAddr)
	}
}

// TestKeystoreV3WrongPassword verifies a wrong password fails the MAC check.
func TestKeystoreV3WrongPassword(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	blob, err := ExportKeystoreV3(key, "the-right-password")
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if _, err := ImportKeystoreV3(blob, "the-wrong-password"); err == nil {
		t.Fatal("import with wrong password succeeded; expected mac mismatch")
	}
}
