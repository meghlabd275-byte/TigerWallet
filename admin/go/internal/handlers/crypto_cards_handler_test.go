package handlers

import (
	"strings"
	"testing"
)

func TestGeneratePANIsLuhnValid(t *testing.T) {
	for i := 0; i < 200; i++ {
		pan, err := generatePAN()
		if err != nil {
			t.Fatalf("generatePAN: %v", err)
		}
		if len(pan) != cryptoCardPANLength {
			t.Fatalf("PAN %q has length %d, want %d", pan, len(pan), cryptoCardPANLength)
		}
		if !strings.HasPrefix(pan, cryptoCardIssuerBIN) {
			t.Fatalf("PAN %q does not start with issuer BIN", pan)
		}
		if !luhnValid(pan) {
			t.Fatalf("PAN %q fails the Luhn checksum", pan)
		}
	}
}

func TestGeneratePANIsNotPredictable(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 500; i++ {
		pan, err := generatePAN()
		if err != nil {
			t.Fatalf("generatePAN: %v", err)
		}
		if seen[pan] {
			t.Fatalf("duplicate PAN generated: %q", pan)
		}
		seen[pan] = true
	}
}

func TestLuhnValidRejectsBadInput(t *testing.T) {
	cases := map[string]bool{
		"":                    false,
		"4532015112830366":    true,  // canonical Visa test number
		"4532015112830367":    false, // single-digit corruption
		"4532-0151-1283-0366": false, // non-digit characters are rejected
		"0000000000000000":    true,  // zeroes satisfy the checksum
	}
	for input, want := range cases {
		if got := luhnValid(input); got != want {
			t.Errorf("luhnValid(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestMaskPANNeverExposesMoreThanLastFour(t *testing.T) {
	const pan = "4532015112830366"
	masked := maskPAN(pan)

	if !strings.HasSuffix(masked, "0366") {
		t.Fatalf("masked PAN %q should end with the last four digits", masked)
	}
	if strings.Contains(masked, "4532015112") {
		t.Fatalf("masked PAN %q leaked the leading digits", masked)
	}
	if strings.Count(masked, "*") != 12 {
		t.Fatalf("masked PAN %q should hide 12 digits", masked)
	}
}

func TestMaskPANHandlesShortInput(t *testing.T) {
	if got := maskPAN("12"); got != "**** **** **** ****" {
		t.Fatalf("maskPAN(short) = %q, want full mask", got)
	}
}

func TestPANEncrypterRoundTrip(t *testing.T) {
	enc := newCardPANEncrypter("unit-test-secret")
	if !enc.configured() {
		t.Fatal("encrypter should be configured with a secret")
	}

	const pan = "4532015112830366"
	sealed, err := enc.encrypt(pan)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if strings.Contains(sealed, pan) {
		t.Fatal("ciphertext must not contain the plaintext PAN")
	}

	opened, err := enc.decrypt(sealed)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if opened != pan {
		t.Fatalf("round trip produced %q, want %q", opened, pan)
	}
}

func TestPANEncrypterUsesFreshNonce(t *testing.T) {
	enc := newCardPANEncrypter("unit-test-secret")
	first, err := enc.encrypt("4532015112830366")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	second, err := enc.encrypt("4532015112830366")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if first == second {
		t.Fatal("identical plaintexts must produce different ciphertexts")
	}
}

func TestPANEncrypterFailsClosedWithoutKey(t *testing.T) {
	enc := newCardPANEncrypter("")
	if enc.configured() {
		t.Fatal("encrypter must not be configured without a secret")
	}
	if _, err := enc.encrypt("4532015112830366"); err == nil {
		t.Fatal("encrypt must fail when no key is configured")
	}
	if _, err := enc.decrypt("anything"); err == nil {
		t.Fatal("decrypt must fail when no key is configured")
	}
}

func TestPANEncrypterRejectsWrongKey(t *testing.T) {
	sealed, err := newCardPANEncrypter("key-one").encrypt("4532015112830366")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if _, err := newCardPANEncrypter("key-two").decrypt(sealed); err == nil {
		t.Fatal("decrypt with the wrong key must fail")
	}
}
