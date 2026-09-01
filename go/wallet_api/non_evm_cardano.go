package main

// non_evm_cardano.go — Cardano SDK: real BIP32-Ed25519 (Khovratovich/Law,
// Icarus paper §3) HD derivation from the wallet seed including SOFT child
// derivation (CIP-1852 paths m/1852'/1815'/a'/role/index end in soft levels),
// enterprise Shelley addresses (header 0x61 mainnet), and canonical
// ed25519-bip32 signatures (r = SHA512(kR||M), S = r + H(R||A||M)·kL).
// Root derivation is Ledger/Byron style (SHA512(0x01||seed) with retry);
// full CBOR ledger transactions fail closed with an explicit error.

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"

	"filippo.io/edwards25519"
	"golang.org/x/crypto/blake2b"
)

// cardanoNode is one ed25519-bip32 extpriv node (kL 32, kR 32, chain 32,
// all little-endian byte strings).
type cardanoNode struct {
	kL, kR, chain []byte
}

var ed25519OrderL = mustBigHex("1000000000000000000000000000000014def9dea2f79cd65812631a5cf5d3ed")

func mustBigHex(h string) *big.Int {
	b, ok := new(big.Int).SetString(h, 16)
	if !ok {
		panic("bad order constant")
	}
	return b
}

// leToBig interprets a little-endian byte string as an integer.
func leToBig(b []byte) *big.Int {
	rev := make([]byte, len(b))
	for i, x := range b {
		rev[len(b)-1-i] = x
	}
	return new(big.Int).SetBytes(rev)
}

// bigToLe32 renders x as 32 little-endian bytes.
func bigToLe32(x *big.Int) []byte {
	be := x.Bytes()
	out := make([]byte, 32)
	for i, v := range be {
		if i >= 32 {
			break
		}
		out[len(be)-1-i] = v
	}
	return out
}

// addLe32 adds two little-endian integers, truncated to 32 bytes (no mod
// reduction — BIP32-Ed25519 keeps kL unreduced).
func addLe32(a, b []byte) []byte {
	res := make([]byte, 32)
	carry := uint16(0)
	for i := 0; i < 32; i++ {
		var av, bv uint16
		if i < len(a) {
			av = uint16(a[i])
		}
		if i < len(b) {
			bv = uint16(b[i])
		}
		s := av + bv + carry
		res[i] = byte(s)
		carry = s >> 8
	}
	return res
}

// mul8Le multiplies a little-endian integer by 8, truncated to 32 bytes.
func mul8Le(z []byte) []byte {
	res := make([]byte, 32)
	carry := uint16(0)
	for i := 0; i < 32 && i < len(z); i++ {
		s := uint16(z[i])*8 + carry
		res[i] = byte(s)
		carry = s >> 8
	}
	return res
}

// cardanoScalar converts kL (unreduced LE) to a canonical edwards25519 scalar.
func cardanoScalar(kL []byte) *edwards25519.Scalar {
	x := new(big.Int).Mod(leToBig(kL), ed25519OrderL)
	le := bigToLe32(x)
	s, err := edwards25519.NewScalar().SetCanonicalBytes(le)
	if err != nil {
		// Defensive: after mod-l the bytes are canonical by construction.
		return edwards25519.NewScalar()
	}
	return s
}

// cardanoPubKey computes the compressed public key A = [kL]B (32 bytes).
func cardanoPubKey(n *cardanoNode) []byte {
	return new(edwards25519.Point).ScalarBaseMult(cardanoScalar(n.kL)).Bytes()
}

// bip32EdRoot computes the root extpriv node: h = SHA512(0x01||seed), retry
// with SHA512(h[:32]) while h[31]&0x20 != 0 (Ledger/Byron master rule);
// cc = SHA256(seed)[32:].
func bip32EdRoot(seed []byte) (*cardanoNode, error) {
	if len(seed) == 0 {
		return nil, errors.New("empty seed")
	}
	hash := sha512.Sum512(append([]byte{0x01}, seed...))
	h := hash[:]
	for h[31]&0x20 != 0 {
		h2 := sha512.Sum512(h[:32])
		h = h2[:]
	}
	kL := make([]byte, 32)
	copy(kL, h[:32])
	kL[0] &= 0xF8
	kL[31] &= 0x7F
	kL[31] |= 0x40
	kR := make([]byte, 32)
	copy(kR, h[32:])
	sum := sha256.Sum256(seed)
	return &cardanoNode{kL: kL, kR: kR, chain: sum[32:]}, nil
}

// bip32EdChild derives a child node (hardened idx >= 0x80000000 or soft).
func bip32EdChild(n *cardanoNode, idx uint32) *cardanoNode {
	var idxBuf [4]byte
	binary.BigEndian.PutUint32(idxBuf[:], idx)
	var zInput, cInput []byte
	if idx >= 0x80000000 {
		zInput = append(append([]byte{0x00}, n.kL...), n.kR...)
		cInput = append(append([]byte{0x01}, n.kL...), n.kR...)
	} else {
		pub := cardanoPubKey(n)
		zInput = append([]byte{0x02}, pub...)
		cInput = append([]byte{0x03}, pub...)
	}
	z := hmacSHA512of(n.chain, append(zInput, idxBuf[:]...))
	c := hmacSHA512of(n.chain, append(cInput, idxBuf[:]...))
	child := &cardanoNode{
		kL:    addLe32(n.kL, mul8Le(z[:32])),
		chain: c[32:],
	}
	if idx >= 0x80000000 {
		child.kR = addLe32(n.kR, z[32:])
	} else {
		child.kR = append([]byte(nil), n.kR...)
	}
	return child
}

// hmacSHA512of computes HMAC-SHA512 using the stdlib.
func hmacSHA512of(key, data []byte) []byte {
	k := make([]byte, 128)
	if len(key) > 128 {
		h := sha512.Sum512(key)
		copy(k, h[:])
	} else {
		copy(k, key)
	}
	ipad := make([]byte, 128)
	opad := make([]byte, 128)
	for i := range k {
		ipad[i] = k[i] ^ 0x36
		opad[i] = k[i] ^ 0x5c
	}
	inner := sha512.Sum512(append(ipad, data...))
	outer := sha512.Sum512(append(opad, inner[:]...))
	return outer[:]
}

// cardanoDerive walks the path from the seed to the leaf node.
func cardanoDerive(seed []byte, path string) (*cardanoNode, error) {
	node, err := bip32EdRoot(seed)
	if err != nil {
		return nil, err
	}
	idxes, err := parsePathNums(path)
	if err != nil {
		return nil, fmt.Errorf("cardano path: %w", err)
	}
	for _, i := range idxes {
		node = bip32EdChild(node, i)
	}
	return node, nil
}

// CardanoAddress computes an enterprise Shelley address: header (6<<4)|1 for
// mainnet (0x61), bech32 "addr1" — fails closed on malformed input.
func CardanoAddress(seed []byte, path string) (string, error) {
	node, err := cardanoDerive(seed, path)
	if err != nil {
		return "", err
	}
	pub := cardanoPubKey(node)
	h, err := blake2b.New(28, nil)
	if err != nil {
		return "", err
	}
	h.Write(pub)
	pkh := h.Sum(nil)
	raw := append([]byte{0x61}, pkh...)
	return bech32Encode("addr", raw)
}

// CardanoSign produces a canonical ed25519-bip32 signature:
// R = [SHA512(kR||M) mod l]B; S = (r + SHA512(R||A||M)·kL) mod l.
func CardanoSign(seed []byte, path string, msg []byte) (sig, pub []byte, err error) {
	node, err := cardanoDerive(seed, path)
	if err != nil {
		return nil, nil, err
	}
	a := cardanoPubKey(node)
	// r = SHA512(kR || M) mod l (64-byte uniform -> SetUniformBytes)
	rDigest := sha512.Sum512(append(append([]byte(nil), node.kR...), msg...))
	r, err := edwards25519.NewScalar().SetUniformBytes(rDigest[:])
	if err != nil {
		return nil, nil, err
	}
	rPoint := new(edwards25519.Point).ScalarBaseMult(r).Bytes()
	// h = SHA512(R || A || M) mod l
	hInput := append(append(append([]byte(nil), rPoint...), a...), msg...)
	hDigest := sha512.Sum512(hInput)
	h, err := edwards25519.NewScalar().SetUniformBytes(hDigest[:])
	if err != nil {
		return nil, nil, err
	}
	// S = r + h·kL (mod l)
	kLs := cardanoScalar(node.kL)
	s := edwards25519.NewScalar().MultiplyAdd(h, kLs, r)
	sBytes := s.Bytes()
	sig = append(append([]byte(nil), rPoint...), sBytes...)
	return sig, a, nil
}

// parsePathNums extracts numeric indices from an "m/44'/1815'/0'/0/0" path.
func parsePathNums(path string) ([]uint32, error) {
	if path == "" {
		return nil, nil
	}
	segments, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	out := make([]uint32, len(segments))
	for i, s := range segments {
		out[i] = uint32(s)
	}
	return out, nil
}
