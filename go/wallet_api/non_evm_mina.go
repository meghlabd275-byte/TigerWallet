package main

// Pallas curve arithmetic for Mina. Mina does not use Ed25519: a Mina account
// public key is a point on Pallas (y^2 = x^3 + 5), serialised as the
// little-endian x coordinate followed by the y parity byte, and the account
// address is Base58Check with version byte 203. An address built from an
// Ed25519 key looks plausible but no Mina node recognises it.

import (
	"errors"
	"math/big"
)

var (
	// pallasP is the base field modulus.
	pallasP = mustParseBig("28948022309329048855892746252171976963363056481941560715954676764349967630337")
	// pallasQ is the order of the Pallas group (the scalar field modulus).
	pallasQ = mustParseBig("28948022309329048855892746252171976963363056481941647379679742748393362948097")
	// pallasA is the curve coefficient (0 for Pallas: y^2 = x^3 + 5).
	pallasA = big.NewInt(0)
	// pallasB is the constant term.
	pallasB = big.NewInt(5)
)

// pallasPoint is an affine point. A nil point is the identity.
type pallasPoint struct {
	x, y *big.Int
}

// pallasGenerator returns the standard Pallas generator (-1, 2).
func pallasGenerator() *pallasPoint {
	return &pallasPoint{
		x: new(big.Int).Sub(pallasP, big.NewInt(1)),
		y: big.NewInt(2),
	}
}

// pallasOnCurve reports whether the point satisfies y^2 = x^3 + 5 mod p.
func pallasOnCurve(p *pallasPoint) bool {
	if p == nil {
		return true
	}
	lhs := new(big.Int).Mul(p.y, p.y)
	lhs.Mod(lhs, pallasP)
	rhs := new(big.Int).Mul(p.x, p.x)
	rhs.Mul(rhs, p.x)
	rhs.Add(rhs, pallasB)
	rhs.Mod(rhs, pallasP)
	return lhs.Cmp(rhs) == 0
}

// pallasAdd returns p + q. Adding a point to its negation yields the identity,
// so that case is handled before the slope is computed: the slope denominator
// would otherwise be zero.
func pallasAdd(p, q *pallasPoint) *pallasPoint {
	if p == nil {
		return q
	}
	if q == nil {
		return p
	}
	if p.x.Cmp(q.x) == 0 {
		sum := new(big.Int).Add(p.y, q.y)
		sum.Mod(sum, pallasP)
		if sum.Sign() == 0 {
			return nil // P + (-P) = identity
		}
		return pallasDouble(p)
	}
	num := new(big.Int).Sub(q.y, p.y)
	den := new(big.Int).Sub(q.x, p.x)
	den.Mod(den, pallasP)
	den.ModInverse(den, pallasP)
	if den == nil {
		return nil
	}
	lambda := num.Mul(num, den)
	lambda.Mod(lambda, pallasP)

	x3 := new(big.Int).Mul(lambda, lambda)
	x3.Sub(x3, p.x)
	x3.Sub(x3, q.x)
	x3.Mod(x3, pallasP)

	y3 := new(big.Int).Sub(p.x, x3)
	y3.Mul(y3, lambda)
	y3.Sub(y3, p.y)
	y3.Mod(y3, pallasP)
	return &pallasPoint{x: x3, y: y3}
}

// pallasDouble returns 2p, or the identity when p is the 2-torsion point.
func pallasDouble(p *pallasPoint) *pallasPoint {
	if p == nil || p.y.Sign() == 0 {
		return nil
	}
	num := new(big.Int).Mul(p.x, p.x)
	num.Mul(num, big.NewInt(3))
	num.Add(num, pallasA)
	den := new(big.Int).Lsh(p.y, 1)
	den.Mod(den, pallasP)
	den.ModInverse(den, pallasP)
	if den == nil {
		return nil
	}
	lambda := num.Mul(num, den)
	lambda.Mod(lambda, pallasP)

	x3 := new(big.Int).Mul(lambda, lambda)
	two := new(big.Int).Lsh(p.x, 1)
	x3.Sub(x3, two)
	x3.Mod(x3, pallasP)

	y3 := new(big.Int).Sub(p.x, x3)
	y3.Mul(y3, lambda)
	y3.Sub(y3, p.y)
	y3.Mod(y3, pallasP)
	return &pallasPoint{x: x3, y: y3}
}

// pallasMul returns k * p using a left-to-right double-and-add ladder.
func pallasMul(k *big.Int, p *pallasPoint) *pallasPoint {
	if k.Sign() == 0 || p == nil {
		return nil
	}
	k = new(big.Int).Mod(k, pallasQ)
	var acc *pallasPoint
	for i := k.BitLen() - 1; i >= 0; i-- {
		acc = pallasDouble(acc)
		if k.Bit(i) == 1 {
			acc = pallasAdd(acc, p)
		}
	}
	return acc
}

// pallasScalarFromEntropy reduces 32 bytes of entropy into a valid Pallas
// scalar. A Mina private key is exactly this scalar, stored little-endian.
func pallasScalarFromEntropy(entropy []byte) (*big.Int, error) {
	if len(entropy) < 32 {
		return nil, errors.New("mina: private key entropy must be at least 32 bytes")
	}
	le := make([]byte, 32)
	for i := 0; i < 32; i++ {
		le[i] = entropy[31-i]
	}
	s := new(big.Int).SetBytes(le)
	s.Mod(s, pallasQ)
	if s.Sign() == 0 {
		return nil, errors.New("mina: derived scalar is zero")
	}
	return s, nil
}

// pallasPublicKey returns the account public key for a scalar: the affine point
// k*G. The result is never the identity for a non-zero scalar.
func pallasPublicKey(k *big.Int) (*pallasPoint, error) {
	if k.Sign() == 0 {
		return nil, errors.New("mina: zero scalar has no public key")
	}
	pt := pallasMul(k, pallasGenerator())
	if pt == nil {
		return nil, errors.New("mina: scalar multiplication produced the identity")
	}
	if !pallasOnCurve(pt) {
		return nil, errors.New("mina: derived point is not on the Pallas curve")
	}
	return pt, nil
}

// pallasEncodePublicKey serialises a public key the way Mina does:
// 0x01 (compressed flag) || 0x01 (parity tag) || x little-endian || parity.
func pallasEncodePublicKey(pt *pallasPoint) []byte {
	parity := byte(0)
	if pt.y.Bit(0) == 1 {
		parity = 1
	}
	out := make([]byte, 0, 35)
	out = append(out, 0x01, 0x01)
	xb := make([]byte, 32)
	pt.x.FillBytes(xb)
	for i := 31; i >= 0; i-- { // little-endian
		out = append(out, xb[i])
	}
	return append(out, parity)
}

// pallasEncodePrivateKey serialises a private key: 0x01 || scalar
// little-endian.
func pallasEncodePrivateKey(k *big.Int) []byte {
	out := make([]byte, 0, 33)
	out = append(out, 0x01)
	sb := make([]byte, 32)
	k.FillBytes(sb)
	for i := 31; i >= 0; i-- {
		out = append(out, sb[i])
	}
	return out
}
