package main

import (
	"encoding/hex"
	"strings"
	"testing"
)

// The XRPL base58 dictionary excludes 0, O, I and l. A short or duplicated
// dictionary cannot round-trip: base58EncAlphabet indexes it with a value in
// [0,57], so a 56-entry table panics on mod 56/57 and silently emits the wrong
// character for every other residue. This test pins the table's shape.
func TestXRPDictionaryIsCanonical(t *testing.T) {
	if len(rippleAlphabet) != 58 {
		t.Fatalf("rippleAlphabet length = %d, want 58", len(rippleAlphabet))
	}
	seen := map[rune]bool{}
	for _, c := range rippleAlphabet {
		if seen[c] {
			t.Errorf("rippleAlphabet repeats %q", c)
		}
		seen[c] = true
	}
	if len(seen) != 58 {
		t.Fatalf("rippleAlphabet has %d unique symbols, want 58", len(seen))
	}
	for _, forbidden := range "0OIl" {
		if strings.ContainsRune(rippleAlphabet, forbidden) {
			t.Errorf("rippleAlphabet must not contain %q", forbidden)
		}
	}
	const canonical = "rpshnaf39wBUDNEGHJKLM4PQRST7VWXYZ2bcdeCg65jkm8oFqi1tuvAxyz"
	if rippleAlphabet != canonical {
		t.Errorf("rippleAlphabet is not the canonical XRPL dictionary:\n got %s\nwant %s",
			rippleAlphabet, canonical)
	}
}

// TestXRPGensisAccountVector reproduces the XRPL genesis account from its
// published public key. It exercises the whole path — hash160(pubkey),
// version byte, double-SHA256 checksum, base58 — so any defect in the
// dictionary or the encoder changes the result.
func TestXRPGensisAccountVector(t *testing.T) {
	const (
		pubHex = "0330E7FC9D56BB25D6893BA3F317AE5BCF33B3291BD63DB32654A313222F7FD020"
		want   = "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh"
	)
	pub, err := hex.DecodeString(pubHex)
	if err != nil {
		t.Fatalf("decode pubkey: %v", err)
	}
	acct := hash160(pub)
	if got := hex.EncodeToString(acct); got != "b5f762798a53d543a014caf8b297cff8f2f937e8" {
		t.Fatalf("account id = %s, want b5f762798a53d543a014caf8b297cff8f2f937e8", got)
	}
	raw := append([]byte{0x00}, acct...)
	got := base58EncAlphabet(append(raw, doubleSHA256(raw)[:4]...), rippleAlphabet)
	if got != want {
		t.Fatalf("address = %s, want %s", got, want)
	}
}

// TestXRPAddressDerivation checks the wallet-facing entry point: every derived
// address must be decodable back to its own account id (round-trip), which a
// 56-character dictionary cannot satisfy.
func TestXRPAddressDerivation(t *testing.T) {
	seed := seedFromMnemonic(t, abandonMnemonic)
	for _, path := range []string{
		"m/44'/144'/0'/0/0",
		"m/44'/144'/0'/0/1",
		"m/44'/144'/0'/0/7",
	} {
		addr, err := XRPAddress(seed, path)
		if err != nil {
			t.Fatalf("XRPAddress(%s): %v", path, err)
		}
		if !strings.HasPrefix(addr, "r") {
			t.Errorf("%s: address %q does not start with 'r'", path, addr)
		}
		raw, err := base58DecAlphabet(addr, rippleAlphabet)
		if err != nil {
			t.Fatalf("%s: decode %q: %v", path, addr, err)
		}
		if len(raw) != 25 {
			t.Fatalf("%s: decoded %d bytes, want 25", path, len(raw))
		}
		if raw[0] != 0x00 {
			t.Errorf("%s: version byte = %#x, want 0x00", path, raw[0])
		}
		if got, want := hex.EncodeToString(raw[21:]), hex.EncodeToString(doubleSHA256(raw[:21])[:4]); got != want {
			t.Errorf("%s: checksum = %s, want %s", path, got, want)
		}
	}
}

// TestBase58EncAlphabetRejectsBadDictionary guards the defensive length check:
// a malformed dictionary must yield "" rather than a panic or a corrupt string.
func TestBase58EncAlphabetRejectsBadDictionary(t *testing.T) {
	if got := base58EncAlphabet([]byte{1, 2, 3}, "short"); got != "" {
		t.Errorf("short dictionary returned %q, want empty", got)
	}
	if got := base58EncAlphabet([]byte{1, 2, 3}, strings.Repeat("a", 60)); got != "" {
		t.Errorf("long dictionary returned %q, want empty", got)
	}
}
