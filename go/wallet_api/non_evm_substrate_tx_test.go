package main

// non_evm_substrate_tx_test.go — SCALE encoding and Substrate extrinsic tests.
//
// The expected bytes are taken from the SCALE specification and from the
// Polkadot JS implementation, so a regression in the encoder fails here rather
// than producing an extrinsic the chain rejects.

import (
	"context"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"
	"testing"
)

func TestScaleCompactUint(t *testing.T) {
	cases := []struct {
		name string
		in   uint64
		want string
	}{
		{"zero", 0, "00"},
		{"one", 1, "04"},
		{"max single byte", 63, "fc"},
		{"min two byte", 64, "0101"},
		{"max two byte", 16383, "fdff"},
		{"min four byte", 16384, "02000100"},
		{"max four byte", 1073741823, "feffffff"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := &scaleWriter{}
			w.Compact(c.in)
			got, err := w.BytesOut()
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			if hex.EncodeToString(got) != c.want {
				t.Fatalf("Compact(%d) = %s, want %s", c.in, hex.EncodeToString(got), c.want)
			}
		})
	}
}

// CompactBig must produce the same bytes as Compact for values that fit, and
// the big-integer form above 2^30.
func TestScaleCompactBig(t *testing.T) {
	t.Run("agrees with compact below 2^30", func(t *testing.T) {
		for _, v := range []uint64{0, 1, 63, 64, 16383, 16384, 1073741823} {
			a := &scaleWriter{}
			a.Compact(v)
			b := &scaleWriter{}
			b.CompactBig(new(big.Int).SetUint64(v))
			if hex.EncodeToString(a.buf) != hex.EncodeToString(b.buf) {
				t.Fatalf("disagreement at %d: %s vs %s",
					v, hex.EncodeToString(a.buf), hex.EncodeToString(b.buf))
			}
		}
	})

	t.Run("u128 above 2^30 uses the big form", func(t *testing.T) {
		// 1 DOT = 10^10 plancks. 10^10 < 2^30 is false (2^30 = 1073741824),
		// so 10^10 exercises the four-byte form.
		v, _ := new(big.Int).SetString("10000000000", 10)
		w := &scaleWriter{}
		w.CompactBig(v)
		// 10000000000 << 2 = 40000000000 = 0x9_502F9000, which needs the
		// big-integer form: mode 0b11 with 8 bytes.
		if len(w.buf) == 0 || w.buf[0]&0b11 != 0b11 {
			t.Fatalf("expected the big-integer mode, got %x", w.buf)
		}
	})

	t.Run("rejects negative", func(t *testing.T) {
		w := &scaleWriter{}
		w.CompactBig(big.NewInt(-1))
		if _, err := w.BytesOut(); err == nil {
			t.Fatal("expected an error for a negative value")
		}
	})

	t.Run("rejects nil", func(t *testing.T) {
		w := &scaleWriter{}
		w.CompactBig(nil)
		if _, err := w.BytesOut(); err == nil {
			t.Fatal("expected an error for a nil value")
		}
	})
}

func TestScaleFixedAndInts(t *testing.T) {
	w := &scaleWriter{}
	w.U8(0x01)
	w.U16(0x0302)
	w.U32(0x07060504)
	w.Fixed([]byte{0xaa, 0xbb})
	got, _ := w.BytesOut()
	want := "01020304050607aabb"
	if hex.EncodeToString(got) != want {
		t.Fatalf("got %s, want %s", hex.EncodeToString(got), want)
	}
}

// The mortal era encoding is a two-byte structure whose first byte is the
// quantisation exponent and second the phase.
func TestEncodeMortalEra(t *testing.T) {
	t.Run("period 64 at block 64", func(t *testing.T) {
		era, err := encodeMortalEra(64, 64)
		if err != nil {
			t.Fatalf("encode: %v", err)
		}
		// 64 = 2^6, so trailing zeros = 6 and the stored exponent is 5.
		// phase = (64/64) % 256 = 1.
		if era[0] != 5 || era[1] != 1 {
			t.Fatalf("era = %x, want 0501", era)
		}
	})

	t.Run("rejects non power of two", func(t *testing.T) {
		if _, err := encodeMortalEra(100, 100); err == nil {
			t.Fatal("expected an error for a non power-of-two period")
		}
	})

	t.Run("rejects period below 4", func(t *testing.T) {
		if _, err := encodeMortalEra(100, 2); err == nil {
			t.Fatal("expected an error for period 2")
		}
	})

	t.Run("rejects period above 65536", func(t *testing.T) {
		if _, err := encodeMortalEra(100, 131072); err == nil {
			t.Fatal("expected an error for period 131072")
		}
	})
}

// SS58 decoding must verify the checksum, so a mistyped address is rejected
// rather than sending funds to a derived-but-wrong account.
func TestSubstrateDecodeSS58(t *testing.T) {
	t.Run("round trips a real address", func(t *testing.T) {
		seed := make([]byte, 32)
		for i := range seed {
			seed[i] = byte(i + 1)
		}
		addr, err := SubstrateAddress(seed, 0) // Polkadot prefix
		if err != nil {
			t.Fatalf("derive address: %v", err)
		}
		id, err := substrateDecodeSS58(addr)
		if err != nil {
			t.Fatalf("decode %q: %v", addr, err)
		}
		if len(id) != 32 {
			t.Fatalf("account id must be 32 bytes, got %d", len(id))
		}
	})

	t.Run("rejects a corrupted checksum", func(t *testing.T) {
		seed := make([]byte, 32)
		for i := range seed {
			seed[i] = byte(i + 1)
		}
		addr, _ := SubstrateAddress(seed, 0)
		// Flip the final character to a different base58 digit.
		last := addr[len(addr)-1]
		repl := byte('1')
		if last == '1' {
			repl = '2'
		}
		broken := addr[:len(addr)-1] + string(repl)
		if _, err := substrateDecodeSS58(broken); err == nil {
			t.Fatal("expected a checksum failure for a corrupted address")
		}
	})

	t.Run("rejects junk", func(t *testing.T) {
		for _, bad := range []string{"", "!!!", "0x1234"} {
			if _, err := substrateDecodeSS58(bad); err == nil {
				t.Fatalf("expected %q to be rejected", bad)
			}
		}
	})
}

func TestHexToFixed32(t *testing.T) {
	good := "0x" + strings.Repeat("ab", 32)
	if _, err := hexToFixed32(good); err != nil {
		t.Fatalf("valid input rejected: %v", err)
	}
	for _, bad := range []string{"", "0x", "0xabcd", strings.Repeat("ab", 31)} {
		if _, err := hexToFixed32(bad); err == nil {
			t.Fatalf("expected %q to be rejected", bad)
		}
	}
}

// An offline (non-broadcast) build must produce a well-formed extrinsic and
// reject inputs it cannot encode.
func TestSubstrateBuildTransferOffline(t *testing.T) {
	seed := make([]byte, 32)
	for i := range seed {
		seed[i] = byte(i + 1)
	}
	to, err := SubstrateAddress(seed, 0)
	if err != nil {
		t.Fatalf("derive recipient: %v", err)
	}

	genesis := "0x" + strings.Repeat("11", 32)
	params := SubstrateChainParams{
		SpecVersion: 1000000,
		TxVersion:   26,
		GenesisHash: genesis,
		Nonce:       0,
		EraPeriod:   0, // immortal keeps the test offline
	}
	amount, _ := new(big.Int).SetString("10000000000", 10) // 1 DOT in plancks

	extHex, txHash, err := SubstrateBuildTransfer(context.Background(), seed,
		"", "http://127.0.0.1:1", [2]byte{0x05, 0x00}, to, amount, params, false)
	if err != nil {
		t.Fatalf("build offline: %v", err)
	}

	raw, err := hex.DecodeString(strings.TrimPrefix(extHex, "0x"))
	if err != nil {
		t.Fatalf("extrinsic is not hex: %v", err)
	}
	// The extrinsic starts with a compact length. Decode it to locate the
	// signed body rather than assuming a one-byte prefix.
	n, off, err := decodeScaleCompact(raw)
	if err != nil {
		t.Fatalf("decode extrinsic length: %v", err)
	}
	if int(n) != len(raw)-off {
		t.Fatalf("declared extrinsic length %d does not match the %d bytes present", n, len(raw)-off)
	}
	body := raw[off:]
	// 0x84 = signed, version 4.
	if body[0] != 0x84 {
		t.Fatalf("expected signed v4 marker 0x84, got %x", body[0])
	}
	// MultiAddress::Id (0x00) then a 32-byte account id.
	if body[1] != 0x00 {
		t.Fatalf("expected MultiAddress::Id marker, got %x", body[1])
	}
	// MultiSignature::Sr25519 (0x01) then a 64-byte signature.
	if body[2+32] != 0x01 {
		t.Fatalf("expected Sr25519 signature marker, got %x", body[2+32])
	}
	// The body holds the prefix, signer, signature, signed-extra and call,
	// which is at least 140 bytes.
	if len(body) < 140 {
		t.Fatalf("extrinsic body is implausibly short: %d bytes", len(body))
	}
	if !strings.HasPrefix(txHash, "0x") || len(txHash) != 66 {
		t.Fatalf("tx hash must be 32 bytes of 0x-prefixed hex, got %q", txHash)
	}
}

// errShortBuffer reports a truncated SCALE compact value.
var errShortBuffer = errors.New("scale: buffer too short for compact value")

// decodeScaleCompact decodes a SCALE compact integer, returning the value and
// the number of bytes consumed.
func decodeScaleCompact(b []byte) (uint64, int, error) {
	if len(b) == 0 {
		return 0, 0, errShortBuffer
	}
	switch b[0] & 0b11 {
	case 0b00:
		return uint64(b[0] >> 2), 1, nil
	case 0b01:
		if len(b) < 2 {
			return 0, 0, errShortBuffer
		}
		return uint64(b[0]>>2) | uint64(b[1])<<6, 2, nil
	case 0b10:
		if len(b) < 4 {
			return 0, 0, errShortBuffer
		}
		v := uint64(b[0]>>2) | uint64(b[1])<<6 | uint64(b[2])<<14 | uint64(b[3])<<22
		return v, 4, nil
	default:
		n := int(b[0]>>2) + 4
		if len(b) < 1+n {
			return 0, 0, errShortBuffer
		}
		var v uint64
		for i := 0; i < n; i++ {
			v |= uint64(b[1+i]) << (8 * uint(i))
		}
		return v, 1 + n, nil
	}
}

func TestSubstrateBuildTransferFailClosed(t *testing.T) {
	seed := make([]byte, 32)
	to, _ := SubstrateAddress(seed, 0)
	genesis := "0x" + strings.Repeat("11", 32)
	amount := big.NewInt(1000)

	t.Run("no endpoint", func(t *testing.T) {
		_, _, err := SubstrateBuildTransfer(context.Background(), seed, "", "",
			[2]byte{5, 0}, to, amount, SubstrateChainParams{SpecVersion: 1, TxVersion: 1, GenesisHash: genesis}, true)
		if err == nil || !strings.Contains(err.Error(), "no node endpoint") {
			t.Fatalf("expected a missing-endpoint error, got: %v", err)
		}
	})

	t.Run("missing runtime version", func(t *testing.T) {
		_, _, err := SubstrateBuildTransfer(context.Background(), seed, "", "http://127.0.0.1:1",
			[2]byte{5, 0}, to, amount, SubstrateChainParams{GenesisHash: genesis}, false)
		if err == nil || !strings.Contains(err.Error(), "runtime metadata") {
			t.Fatalf("expected a runtime-version error, got: %v", err)
		}
	})

	t.Run("non-positive amount", func(t *testing.T) {
		_, _, err := SubstrateBuildTransfer(context.Background(), seed, "", "http://127.0.0.1:1",
			[2]byte{5, 0}, to, big.NewInt(0),
			SubstrateChainParams{SpecVersion: 1, TxVersion: 1, GenesisHash: genesis}, false)
		if err == nil || !strings.Contains(err.Error(), "must be positive") {
			t.Fatalf("expected a positive-amount error, got: %v", err)
		}
	})

	t.Run("bad genesis hash", func(t *testing.T) {
		_, _, err := SubstrateBuildTransfer(context.Background(), seed, "", "http://127.0.0.1:1",
			[2]byte{5, 0}, to, amount,
			SubstrateChainParams{SpecVersion: 1, TxVersion: 1, GenesisHash: "0xzz"}, false)
		if err == nil {
			t.Fatal("expected an error for a malformed genesis hash")
		}
	})

	t.Run("invalid recipient", func(t *testing.T) {
		_, _, err := SubstrateBuildTransfer(context.Background(), seed, "", "http://127.0.0.1:1",
			[2]byte{5, 0}, "not-an-address", amount,
			SubstrateChainParams{SpecVersion: 1, TxVersion: 1, GenesisHash: genesis}, false)
		if err == nil {
			t.Fatal("expected an error for an invalid recipient address")
		}
	})
}
