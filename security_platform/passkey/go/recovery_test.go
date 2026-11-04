package main

import (
	"encoding/base64"
	"strings"
	"testing"
)

func newTestService(t *testing.T) *PasskeyService {
	t.Helper()
	return &PasskeyService{
		config: &Config{EncryptionKey: strings.Repeat("k", 32)},
		encKey: []byte(strings.Repeat("k", 32)),
	}
}

func TestEncryptShareRoundTrip(t *testing.T) {
	s := newTestService(t)
	plain := "01-aabbccddeeff00112233445566778899"

	sealed, err := s.encryptShare(plain)
	if err != nil {
		t.Fatalf("encryptShare: %v", err)
	}
	if sealed == plain {
		t.Fatal("ciphertext equals plaintext")
	}

	got, err := s.decryptShare(sealed)
	if err != nil {
		t.Fatalf("decryptShare: %v", err)
	}
	if got != plain {
		t.Fatalf("round trip = %q, want %q", got, plain)
	}
}

func TestEncryptShareUsesAFreshNonce(t *testing.T) {
	s := newTestService(t)
	plain := "02-00112233445566778899001122334455"

	first, err := s.encryptShare(plain)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.encryptShare(plain)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("two encryptions of the same share produced identical ciphertext; the nonce is not fresh")
	}
	// Both must still decrypt to the same share.
	for i, blob := range []string{first, second} {
		got, err := s.decryptShare(blob)
		if err != nil {
			t.Fatalf("decryptShare(%d): %v", i, err)
		}
		if got != plain {
			t.Fatalf("decryptShare(%d) = %q, want %q", i, got, plain)
		}
	}
}

func TestDecryptShareRejectsTampering(t *testing.T) {
	s := newTestService(t)
	sealed, err := s.encryptShare("03-deadbeefdeadbeefdeadbeefdeadbeef")
	if err != nil {
		t.Fatal(err)
	}

	raw, err := base64.StdEncoding.DecodeString(sealed)
	if err != nil {
		t.Fatal(err)
	}
	// Flip one bit in the middle of the payload; GCM must reject it rather
	// than returning corrupted plaintext.
	raw[len(raw)/2] ^= 0x01
	tampered := base64.StdEncoding.EncodeToString(raw)

	if _, err := s.decryptShare(tampered); err == nil {
		t.Fatal("decryptShare accepted a tampered share")
	}
}

func TestDecryptShareRejectsTruncatedAndGarbage(t *testing.T) {
	s := newTestService(t)

	for name, in := range map[string]string{
		"not base64": "!!!!",
		"too short":  base64.StdEncoding.EncodeToString([]byte{1, 2, 3}),
		"empty":      "",
	} {
		if _, err := s.decryptShare(in); err == nil {
			t.Errorf("decryptShare accepted %s input", name)
		}
	}
}

func TestEncryptShareRejectsWrongKeyLength(t *testing.T) {
	s := &PasskeyService{encKey: []byte("short")}
	if _, err := s.encryptShare("01-aa"); err == nil {
		t.Fatal("encryptShare accepted a key that is not 32 bytes")
	}
	if _, err := s.decryptShare(base64.StdEncoding.EncodeToString([]byte("x"))); err == nil {
		t.Fatal("decryptShare accepted a key that is not 32 bytes")
	}
}
