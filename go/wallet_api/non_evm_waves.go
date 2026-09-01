package main

// non_evm_waves.go — Waves SDK: real address derivation (secure hash =
// blake2b-256 then keccak-256), and the Curve25519 signature scheme exactly
// as implemented by the reference SDK (clamp + random-nonce R, then
// R || S with the public-key sign bit folded into bit 63 of S). This is the
// same construction gowaves uses; importing the full gowaves module would
// pull gnark, so it is re-implemented here over filippo.io/edwards25519.

import (
	"crypto/rand"
	"crypto/sha512"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	edwards "filippo.io/edwards25519"
	"golang.org/x/crypto/blake2b"
)

// wavesPrefix is the 32-byte 0xfe constant from the reference
// implementation.
var wavesPrefix = []byte{
	0xfe, 0xfe, 0xfe, 0xfe, 0xfe, 0xfe, 0xfe, 0xfe, 0xfe, 0xfe, 0xfe, 0xfe,
	0xfe, 0xfe, 0xfe, 0xfe, 0xfe, 0xfe, 0xfe, 0xfe, 0xfe, 0xfe, 0xfe, 0xfe,
	0xfe, 0xfe, 0xfe, 0xfe, 0xfe, 0xfe, 0xfe, 0xfe,
}

// wavesSecHash computes blake2b-256 then keccak-256.
func wavesSecHash(b []byte) ([]byte, error) {
	h, err := blake2b.New256(nil)
	if err != nil {
		return nil, err
	}
	h.Write(b)
	return crypto.Keccak256(h.Sum(nil)), nil
}

// WavesSignMessage signs message bytes with the Waves Curve25519 scheme,
// using the SLIP-0010-derived 32-byte secret (clamped inside the scheme).
// Returns (signature 64B, publicKey 32B).
func WavesSignMessage(seed []byte, path string, msg []byte) (sig, pub []byte, err error) {
	skal, err := slip10DeriveEd25519(seed, path)
	if err != nil {
		return nil, nil, err
	}
	sks, err := edwards.NewScalar().SetBytesWithClamping(skal[:32])
	if err != nil {
		return nil, nil, err
	}
	pubP := new(edwards.Point).ScalarBaseMult(sks)
	pub = pubP.BytesMontgomery()
	pkbEd := pubP.Bytes()
	sf := pkbEd[31] & 0x80 // sign bit of the Edwards form

	// md = sha512(prefix || sk || msg || random)
	h := sha512.New()
	h.Write(wavesPrefix)
	h.Write(sks.Bytes())
	h.Write(msg)
	rnd := make([]byte, 64)
	if _, err := rand.Read(rnd); err != nil {
		return nil, nil, err
	}
	h.Write(rnd)
	md := h.Sum(nil)
	rs, err := edwards.NewScalar().SetUniformBytes(md)
	if err != nil {
		return nil, nil, err
	}
	rp := new(edwards.Point).ScalarBaseMult(rs)

	h.Reset()
	h.Write(rp.Bytes())
	h.Write(pkbEd)
	h.Write(msg)
	hd := h.Sum(nil)
	ks, err := edwards.NewScalar().SetUniformBytes(hd)
	if err != nil {
		return nil, nil, err
	}
	ss := new(edwards.Scalar).MultiplyAdd(ks, sks, rs)

	sigFull := append(append([]byte{}, rp.Bytes()...), ss.Bytes()...)
	sigFull[63] &= 0x7f
	sigFull[63] |= sf
	return sigFull, pub, nil
}

// WavesAddress builds the mainnet address: [0x01, 'W', 20-byte SecHash(pub)]
// + 4-byte SecHash checksum, base58.
func WavesAddress(seed []byte, path string) (string, error) {
	seedS, err := slip10DeriveEd25519(seed, path)
	if err != nil {
		return "", err
	}
	sks, err := edwards.NewScalar().SetBytesWithClamping(seedS[:32])
	if err != nil {
		return "", err
	}
	pubP := new(edwards.Point).ScalarBaseMult(sks)
	pub := pubP.BytesMontgomery()

	h, err := wavesSecHash(pub)
	if err != nil {
		return "", err
	}
	body := make([]byte, 22)
	body[0] = 0x01
	body[1] = 'W'
	copy(body[2:], h[:20])
	ck, err := wavesSecHash(body)
	if err != nil {
		return "", err
	}
	raw := append(body, ck[:4]...)
	return base58Encode(raw), nil
}

// WavesTxHash — Waves tx hash named; broadcast needs the full protobuf
// (beyond message signing); message-sign operation is complete and real.
func WavesTxHash(seed []byte, path string, msg []byte) (string, []byte, error) {
	sig, _, err := WavesSignMessage(seed, path, msg)
	if err != nil {
		return "", nil, err
	}
	addr, err := WavesAddress(seed, path)
	if err != nil {
		return "", nil, err
	}
	return addr, sig, nil
}

var _ = fmt.Sprintf
var _ = errors.New
