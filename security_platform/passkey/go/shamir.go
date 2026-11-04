/**
 * TigerWallet Passkey Authentication Service - Shamir Secret Sharing
 *
 * Threshold secret sharing over GF(2^8) with the AES reduction polynomial
 * 0x11b. Each byte of the secret is the constant term of an independent random
 * degree-(threshold-1) polynomial; share i is that polynomial evaluated at
 * x = i. Any `threshold` distinct shares rebuild the secret by Lagrange
 * interpolation at x = 0, and any smaller set yields a value that is
 * information-theoretically independent of the secret.
 *
 * This operates on arbitrary bytes, so it can split a wallet seed, a raw key,
 * or an arbitrary recovery secret.
 */

package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// gfMul multiplies two elements of GF(2^8) modulo the AES polynomial 0x11b.
// The multiplication is the carry-less ("Russian peasant") product reduced by
// the polynomial, which is why the XOR-and-shift form below is correct.
func gfMul(a, b byte) byte {
	var p byte
	for i := 0; i < 8; i++ {
		if b&1 != 0 {
			p ^= a
		}
		hi := a & 0x80
		a <<= 1
		if hi != 0 {
			a ^= 0x1b // low byte of 0x11b; the x^8 term cancels
		}
		b >>= 1
	}
	return p
}

// gfInv returns the multiplicative inverse of a in GF(2^8). Every non-zero
// element of the field satisfies a^255 = 1, so a^254 = a^-1.
func gfInv(a byte) (byte, error) {
	if a == 0 {
		return 0, errors.New("shamir: zero has no multiplicative inverse")
	}
	// a^254 by square-and-multiply over the exponent 254 = 0b11111110.
	result := byte(1)
	base := a
	for e := 254; e > 0; e >>= 1 {
		if e&1 != 0 {
			result = gfMul(result, base)
		}
		base = gfMul(base, base)
	}
	return result, nil
}

// gfDiv returns a/b in GF(2^8).
func gfDiv(a, b byte) (byte, error) {
	inv, err := gfInv(b)
	if err != nil {
		return 0, err
	}
	return gfMul(a, inv), nil
}

// gfEval evaluates the polynomial whose coefficients are coeffs (coeffs[0] is
// the constant term) at x, using Horner's method.
func gfEval(coeffs []byte, x byte) byte {
	var y byte
	for i := len(coeffs) - 1; i >= 0; i-- {
		y = gfMul(y, x) ^ coeffs[i]
	}
	return y
}

// ShamirShare is one share of a split secret.
type ShamirShare struct {
	Index byte
	Value []byte
}

// shamirSplit divides secret into total shares, any threshold of which
// reconstruct it. total must be between 1 and 255 and threshold between 1 and
// total; share indices are 1..total so that no share lands on x = 0, where the
// polynomial value would be the secret itself.
func shamirSplit(secret []byte, threshold, total int) ([]ShamirShare, error) {
	if len(secret) == 0 {
		return nil, errors.New("shamir: secret is empty")
	}
	if threshold < 1 {
		return nil, fmt.Errorf("shamir: threshold must be at least 1, got %d", threshold)
	}
	if total < 1 || total > 255 {
		return nil, fmt.Errorf("shamir: total shares must be between 1 and 255, got %d", total)
	}
	if threshold > total {
		return nil, fmt.Errorf("shamir: threshold %d exceeds total shares %d", threshold, total)
	}

	// One polynomial per secret byte, all sharing the same x-coordinates.
	coeffs := make([][]byte, len(secret))
	for i, b := range secret {
		row := make([]byte, threshold)
		row[0] = b
		if threshold > 1 {
			if _, err := rand.Read(row[1:]); err != nil {
				return nil, fmt.Errorf("shamir: random coefficients: %w", err)
			}
		}
		coeffs[i] = row
	}

	shares := make([]ShamirShare, total)
	for i := 0; i < total; i++ {
		x := byte(i + 1)
		value := make([]byte, len(secret))
		for j := range secret {
			value[j] = gfEval(coeffs[j], x)
		}
		shares[i] = ShamirShare{Index: x, Value: value}
	}
	return shares, nil
}

// shamirRecover rebuilds the secret from shares by Lagrange interpolation at
// x = 0. Duplicate indices are rejected: two shares with the same x would make
// the denominator of the Lagrange basis vanish.
func shamirRecover(shares []ShamirShare) ([]byte, error) {
	if len(shares) == 0 {
		return nil, errors.New("shamir: no shares supplied")
	}

	seen := make(map[byte]struct{}, len(shares))
	ordered := make([]ShamirShare, 0, len(shares))
	for _, s := range shares {
		if s.Index == 0 {
			return nil, errors.New("shamir: share index 0 is reserved for the secret")
		}
		if _, dup := seen[s.Index]; dup {
			return nil, fmt.Errorf("shamir: duplicate share index %d", s.Index)
		}
		seen[s.Index] = struct{}{}
		ordered = append(ordered, s)
	}

	size := len(ordered[0].Value)
	for _, s := range ordered {
		if len(s.Value) != size {
			return nil, fmt.Errorf("shamir: share %d has length %d, expected %d", s.Index, len(s.Value), size)
		}
	}

	sort.Slice(ordered, func(a, b int) bool { return ordered[a].Index < ordered[b].Index })

	secret := make([]byte, size)
	for pos := 0; pos < size; pos++ {
		var acc byte
		for i := range ordered {
			// Lagrange basis polynomial l_i evaluated at 0:
			//   l_i(0) = prod_{j != i} x_j / (x_i - x_j)
			// In characteristic 2, x_i - x_j is x_i ^ x_j and -x_j is x_j.
			num := byte(1)
			den := byte(1)
			for j := range ordered {
				if i == j {
					continue
				}
				num = gfMul(num, ordered[j].Index)
				den = gfMul(den, ordered[i].Index^ordered[j].Index)
			}
			basis, err := gfDiv(num, den)
			if err != nil {
				return nil, err
			}
			acc ^= gfMul(ordered[i].Value[pos], basis)
		}
		secret[pos] = acc
	}
	return secret, nil
}

// encodeShare renders a share as "<index>-<hex>". The index is stored in the
// clear because reconstruction needs it; the value is the secret-bearing part.
func encodeShare(s ShamirShare) string {
	return fmt.Sprintf("%02d-%s", s.Index, hex.EncodeToString(s.Value))
}

// decodeShare parses the form produced by encodeShare.
func decodeShare(encoded string) (ShamirShare, error) {
	parts := strings.SplitN(strings.TrimSpace(encoded), "-", 2)
	if len(parts) != 2 {
		return ShamirShare{}, fmt.Errorf("shamir: malformed share %q", encoded)
	}
	var idx int
	if _, err := fmt.Sscanf(parts[0], "%d", &idx); err != nil {
		return ShamirShare{}, fmt.Errorf("shamir: bad share index in %q: %w", encoded, err)
	}
	if idx < 1 || idx > 255 {
		return ShamirShare{}, fmt.Errorf("shamir: share index %d out of range", idx)
	}
	value, err := hex.DecodeString(parts[1])
	if err != nil {
		return ShamirShare{}, fmt.Errorf("shamir: bad share payload in %q: %w", encoded, err)
	}
	if len(value) == 0 {
		return ShamirShare{}, fmt.Errorf("shamir: share %q carries no data", encoded)
	}
	return ShamirShare{Index: byte(idx), Value: value}, nil
}
