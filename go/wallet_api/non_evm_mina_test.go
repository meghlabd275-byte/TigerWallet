package main

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"testing"
)

// The Pallas generator must satisfy the curve equation, and the group order
// must annihilate it. If either constant were wrong every Mina address would be
// silently wrong, so both are checked against the curve rather than trusted.
func TestPallasGeneratorAndOrder(t *testing.T) {
	g := pallasGenerator()
	if !pallasOnCurve(g) {
		t.Fatal("generator is not on the Pallas curve")
	}
	if pallasMul(pallasQ, g) != nil {
		t.Fatal("order * G must be the point at infinity")
	}
	if pallasMul(big.NewInt(1), g) == nil {
		t.Fatal("1 * G must not be the identity")
	}
}

// Adding a point to its negation must produce the identity rather than a
// division by zero. This is the branch that was missing from the Starknet
// implementation of the same construction.
func TestPallasInfinityBranch(t *testing.T) {
	g := pallasGenerator()
	neg := &pallasPoint{x: new(big.Int).Set(g.x), y: new(big.Int).Neg(g.y)}
	neg.y.Mod(neg.y, pallasP)
	if got := pallasAdd(g, neg); got != nil {
		t.Fatalf("G + (-G) must be the identity, got %v", got)
	}
	two := pallasAdd(g, g)
	if two == nil || !pallasOnCurve(two) {
		t.Fatal("2G must be a point on the curve")
	}
	if pallasAdd(two, neg) == nil {
		t.Fatal("2G + (-G) must not be the identity")
	}
}

// pallasMul must agree with repeated addition, which exercises the ladder
// independently of the doubling shortcut.
func TestPallasScalarMultiplicationAgreesWithAddition(t *testing.T) {
	g := pallasGenerator()
	acc := (*pallasPoint)(nil)
	for i := 1; i <= 12; i++ {
		acc = pallasAdd(acc, g)
		got := pallasMul(big.NewInt(int64(i)), g)
		if got == nil || acc == nil {
			t.Fatalf("%d * G produced the identity", i)
		}
		if got.x.Cmp(acc.x) != 0 || got.y.Cmp(acc.y) != 0 {
			t.Fatalf("%d * G disagrees with repeated addition", i)
		}
	}
}

// A Mina address is Base58Check with version byte 203 over a 35-byte payload
// whose first two bytes are 0x01 0x01. Every mainnet address begins with B62q.
func TestMinaAddressFormatAndVersionByte(t *testing.T) {
	seed := seedFromMnemonic(t, abandonMnemonic)
	addr, err := MinaAddress(seed, "m/44'/12586'/0'/0/0")
	if err != nil {
		t.Fatalf("mina address: %v", err)
	}
	if len(addr) != 55 {
		t.Fatalf("mina address must be 55 characters, got %d (%q)", len(addr), addr)
	}
	if addr[:4] != "B62q" {
		t.Fatalf("mina mainnet address must start with B62q, got %q", addr)
	}
	raw, err := base58checkDecode(addr)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if raw[0] != 203 {
		t.Fatalf("version byte = %d, want 203", raw[0])
	}
	if raw[1] != 0x01 || raw[2] != 0x01 {
		t.Fatalf("payload tag = %02x%02x, want 0101", raw[1], raw[2])
	}
	if len(raw) != 36 {
		t.Fatalf("payload must be 1 version + 35 key bytes, got %d", len(raw))
	}
}

// The serialised public key must round-trip back to a point on the curve: the
// x coordinate is little-endian and the final byte is the y parity.
func TestPallasPublicKeyEncodingRoundTrip(t *testing.T) {
	entropy := make([]byte, 32)
	for i := range entropy {
		entropy[i] = byte(0x30 + i)
	}
	scalar, err := pallasScalarFromEntropy(entropy)
	if err != nil {
		t.Fatalf("scalar: %v", err)
	}
	pt, err := pallasPublicKey(scalar)
	if err != nil {
		t.Fatalf("public key: %v", err)
	}
	enc := pallasEncodePublicKey(pt)
	if len(enc) != 35 {
		t.Fatalf("encoded public key must be 35 bytes, got %d", len(enc))
	}
	if enc[0] != 0x01 || enc[1] != 0x01 {
		t.Fatalf("tag bytes = %02x%02x, want 0101", enc[0], enc[1])
	}

	// Rebuild x from the little-endian bytes and confirm the point is on the
	// curve with the recorded parity.
	xb := make([]byte, 32)
	for i := 0; i < 32; i++ {
		xb[i] = enc[33-i]
	}
	x := new(big.Int).SetBytes(xb)
	if x.Cmp(pt.x) != 0 {
		t.Fatal("little-endian x did not round-trip")
	}
	if enc[34] != byte(pt.y.Bit(0)) {
		t.Fatal("parity byte does not match the y coordinate")
	}
	if !pallasOnCurve(&pallasPoint{x: x, y: pt.y}) {
		t.Fatal("decoded point is not on the curve")
	}
}

// The private key serialisation is 0x01 || scalar little-endian.
func TestPallasPrivateKeyEncoding(t *testing.T) {
	scalar := big.NewInt(0x0102030405060708)
	enc := pallasEncodePrivateKey(scalar)
	if len(enc) != 33 || enc[0] != 0x01 {
		t.Fatalf("private key encoding = %x, want 0x01 || 32 bytes", enc)
	}
	if enc[1] != 0x08 || enc[8] != 0x01 {
		t.Fatalf("scalar must be little-endian, got %s", hex.EncodeToString(enc[1:]))
	}
	if !bytes.Equal(enc[9:], make([]byte, 24)) {
		t.Fatal("high bytes of a small scalar must be zero")
	}
}

// A scalar of zero has no public key and must fail closed instead of returning
// the identity point.
func TestPallasZeroScalarFailsClosed(t *testing.T) {
	if _, err := pallasPublicKey(big.NewInt(0)); err == nil {
		t.Fatal("a zero scalar must not produce a public key")
	}
	if _, err := pallasScalarFromEntropy(make([]byte, 8)); err == nil {
		t.Fatal("short entropy must fail closed")
	}
}
