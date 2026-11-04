package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"testing"
)

func TestGFMultiplicationAgainstKnownProducts(t *testing.T) {
	// AES/Rijndael field identities and textbook products.
	cases := []struct{ a, b, want byte }{
		{0x00, 0x5a, 0x00}, // zero annihilates
		{0x01, 0x5a, 0x5a}, // one is the identity
		{0x02, 0x01, 0x02},
		{0x57, 0x83, 0xc1}, // from the AES specification (FIPS-197)
		{0x57, 0x13, 0xfe}, // from the AES specification
		{0x02, 0x80, 0x1b}, // reduction by 0x11b wraps exactly once
		{0xff, 0x01, 0xff},
	}
	for _, c := range cases {
		if got := gfMul(c.a, c.b); got != c.want {
			t.Errorf("gfMul(%#02x, %#02x) = %#02x, want %#02x", c.a, c.b, got, c.want)
		}
	}
}

func TestGFMultiplicationIsCommutative(t *testing.T) {
	for a := 0; a < 256; a++ {
		for b := 0; b < 256; b++ {
			if gfMul(byte(a), byte(b)) != gfMul(byte(b), byte(a)) {
				t.Fatalf("gfMul not commutative at %#02x,%#02x", a, b)
			}
		}
	}
}

func TestGFInverseSatisfiesUnity(t *testing.T) {
	for a := 1; a < 256; a++ {
		inv, err := gfInv(byte(a))
		if err != nil {
			t.Fatalf("gfInv(%#02x) returned error: %v", a, err)
		}
		if prod := gfMul(byte(a), inv); prod != 1 {
			t.Fatalf("gfMul(%#02x, gfInv(%#02x)) = %#02x, want 0x01", a, a, prod)
		}
	}
	if _, err := gfInv(0); err == nil {
		t.Fatal("gfInv(0) must fail: zero has no inverse")
	}
}

func TestGFDivisionInvertsMultiplication(t *testing.T) {
	for a := 1; a < 256; a += 7 {
		for b := 1; b < 256; b += 11 {
			q, err := gfDiv(byte(a), byte(b))
			if err != nil {
				t.Fatalf("gfDiv(%#02x,%#02x): %v", a, b, err)
			}
			if gfMul(q, byte(b)) != byte(a) {
				t.Fatalf("gfDiv(%#02x,%#02x) * %#02x != %#02x", a, b, b, a)
			}
		}
	}
}

func TestSplitRecoversEveryThresholdSubset(t *testing.T) {
	secret := []byte("correct horse battery staple: 32 bytes!")

	shares, err := shamirSplit(secret, 3, 5)
	if err != nil {
		t.Fatalf("shamirSplit: %v", err)
	}
	if len(shares) != 5 {
		t.Fatalf("got %d shares, want 5", len(shares))
	}

	// Every 3-of-5 combination must rebuild the secret, and the order of the
	// shares must not matter.
	for i := 0; i < 5; i++ {
		for j := i + 1; j < 5; j++ {
			for k := j + 1; k < 5; k++ {
				subset := []ShamirShare{shares[k], shares[i], shares[j]}
				got, err := shamirRecover(subset)
				if err != nil {
					t.Fatalf("recover(%d,%d,%d): %v", i, j, k, err)
				}
				if !bytes.Equal(got, secret) {
					t.Fatalf("recover(%d,%d,%d) = %q, want %q", i, j, k, got, secret)
				}
			}
		}
	}
}

func TestFewerThanThresholdSharesDoNotRevealTheSecret(t *testing.T) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		t.Fatal(err)
	}
	shares, err := shamirSplit(secret, 4, 6)
	if err != nil {
		t.Fatal(err)
	}

	// Interpolating 3 shares through a degree-3 polynomial does not land on
	// the secret. Check every 3-subset; none may equal the secret.
	matches := 0
	for i := 0; i < 6; i++ {
		for j := i + 1; j < 6; j++ {
			for k := j + 1; k < 6; k++ {
				got, err := shamirRecover([]ShamirShare{shares[i], shares[j], shares[k]})
				if err != nil {
					t.Fatalf("recover: %v", err)
				}
				if bytes.Equal(got, secret) {
					matches++
				}
			}
		}
	}
	if matches != 0 {
		t.Fatalf("%d three-share subsets reproduced the secret below the threshold", matches)
	}
}

func TestSplitWithThresholdOneRepeatsTheSecret(t *testing.T) {
	secret := []byte{0xde, 0xad, 0xbe, 0xef}
	shares, err := shamirSplit(secret, 1, 3)
	if err != nil {
		t.Fatal(err)
	}
	for _, sh := range shares {
		if !bytes.Equal(sh.Value, secret) {
			t.Fatalf("threshold 1 share %d = %x, want %x", sh.Index, sh.Value, secret)
		}
	}
	got, err := shamirRecover(shares[:1])
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, secret) {
		t.Fatalf("recover with threshold 1 = %x, want %x", got, secret)
	}
}

func TestSplitRejectsInvalidParameters(t *testing.T) {
	secret := []byte("s")
	if _, err := shamirSplit(secret, 0, 3); err == nil {
		t.Error("threshold 0 must be rejected")
	}
	if _, err := shamirSplit(secret, 4, 3); err == nil {
		t.Error("threshold above total must be rejected")
	}
	if _, err := shamirSplit(secret, 1, 0); err == nil {
		t.Error("total 0 must be rejected")
	}
	if _, err := shamirSplit(secret, 1, 256); err == nil {
		t.Error("total 256 must be rejected: share indices are a single byte")
	}
	if _, err := shamirSplit(nil, 1, 1); err == nil {
		t.Error("empty secret must be rejected")
	}
}

func TestRecoverRejectsMalformedShareSets(t *testing.T) {
	shares, err := shamirSplit([]byte("abcdefgh"), 2, 3)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := shamirRecover(nil); err == nil {
		t.Error("no shares must be rejected")
	}
	// Duplicate x-coordinates make the Lagrange denominator zero.
	if _, err := shamirRecover([]ShamirShare{shares[0], shares[0]}); err == nil {
		t.Error("duplicate share indices must be rejected")
	}
	// Index 0 is where the secret lives; it must never be accepted as a share.
	if _, err := shamirRecover([]ShamirShare{{Index: 0, Value: []byte("x")}}); err == nil {
		t.Error("share index 0 must be rejected")
	}
	// Ragged lengths indicate a corrupted set.
	if _, err := shamirRecover([]ShamirShare{shares[0], {Index: 9, Value: []byte("short")}}); err == nil {
		t.Error("mismatched share lengths must be rejected")
	}
}

func TestSharesDifferFromEachOtherAndFromTheSecret(t *testing.T) {
	secret := []byte("00000000000000000000000000000000")
	shares, err := shamirSplit(secret, 3, 5)
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	for _, sh := range shares {
		key := hex.EncodeToString(sh.Value)
		if key == hex.EncodeToString(secret) {
			t.Fatalf("share %d equals the secret", sh.Index)
		}
		if seen[key] {
			t.Fatalf("share %d duplicates another share", sh.Index)
		}
		seen[key] = true
	}
}

func TestShareEncodingRoundTrip(t *testing.T) {
	shares, err := shamirSplit([]byte("round trip payload"), 2, 3)
	if err != nil {
		t.Fatal(err)
	}
	for _, sh := range shares {
		encoded := encodeShare(sh)
		if !strings.Contains(encoded, "-") {
			t.Fatalf("encoded share %q lacks an index separator", encoded)
		}
		decoded, err := decodeShare(encoded)
		if err != nil {
			t.Fatalf("decodeShare(%q): %v", encoded, err)
		}
		if decoded.Index != sh.Index || !bytes.Equal(decoded.Value, sh.Value) {
			t.Fatalf("round trip changed the share: %+v vs %+v", decoded, sh)
		}
	}
}

func TestDecodeShareRejectsBadInput(t *testing.T) {
	bad := []string{
		"",
		"nodash",
		"00-aabb",   // index 0 is reserved
		"256-aabb",  // index out of byte range
		"01-",       // empty payload
		"01-zzzz",   // not hex
		"xx-aabbcc", // index not a number
	}
	for _, in := range bad {
		if _, err := decodeShare(in); err == nil {
			t.Errorf("decodeShare(%q) must fail", in)
		}
	}
}

func TestRandomSecretsRecoverAcrossManyThresholds(t *testing.T) {
	for _, tc := range []struct{ threshold, total int }{
		{2, 3}, {3, 5}, {5, 5}, {4, 9}, {2, 10}, {1, 4}, {7, 7},
	} {
		secret := make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			t.Fatal(err)
		}
		shares, err := shamirSplit(secret, tc.threshold, tc.total)
		if err != nil {
			t.Fatalf("split(%d,%d): %v", tc.threshold, tc.total, err)
		}
		// Use exactly the threshold, taking the last N shares to prove
		// position within the set is irrelevant.
		used := shares[tc.total-tc.threshold:]
		got, err := shamirRecover(used)
		if err != nil {
			t.Fatalf("recover(%d,%d): %v", tc.threshold, tc.total, err)
		}
		if !bytes.Equal(got, secret) {
			t.Fatalf("threshold %d of %d did not recover the secret", tc.threshold, tc.total)
		}
	}
}
