package main

// hd_derive.go — REAL BIP-32 hierarchical deterministic key derivation.
// Implements HMAC-SHA512 CKD (child key derivation) per BIP-32, with
// hardened (index >= 0x80000000) and normal derivation, using secp256k1
// via go-ethereum/crypto. No shortcuts, no SHA256-of-seed fakes.

import (
	"crypto/ecdsa"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

const hardening uint32 = 0x80000000

// hdDerive derives the secp256k1 private key for a BIP-32 path (e.g.
// "m/44'/60'/0'/0/0") from a BIP-39 seed.
func hdDerive(seed []byte, path string) (*ecdsa.PrivateKey, error) {
	// Master key = HMAC-SHA512(key="Bitcoin seed", msg=seed)
	mac := hmac.New(sha512.New, []byte("Bitcoin seed"))
	mac.Write(seed)
	I := mac.Sum(nil)
	il := I[:32]
	ir := I[32:]

	parentKey := il
	parentChain := ir

	segments, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	for _, idx := range segments {
		child, childChain, err := ckdPriv(parentKey, parentChain, idx)
		if err != nil {
			return nil, err
		}
		parentKey = child
		parentChain = childChain
	}

	return crypto.ToECDSA(parentKey)
}

// parsePath parses a BIP-32 path string like "m/44'/60'/0'/0/0" into indices.
func parsePath(path string) ([]uint32, error) {
	path = strings.TrimSpace(path)
	if path == "m" || path == "m/" || path == "" {
		return nil, nil
	}
	if strings.HasPrefix(path, "m/") {
		path = path[2:]
	} else if path == "m" {
		return nil, nil
	}
	parts := strings.Split(path, "/")
	out := make([]uint32, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		hardened := strings.HasSuffix(p, "'") || strings.HasSuffix(p, "h") || strings.HasSuffix(p, "H")
		if hardened {
			p = p[:len(p)-1]
		}
		n, err := strconv.ParseUint(p, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid path segment %q: %w", p, err)
		}
		idx := uint32(n)
		if hardened {
			idx += hardening
		}
		out = append(out, idx)
	}
	return out, nil
}

// ckdPriv performs a single BIP-32 child key derivation (private -> private).
func ckdPriv(parentKey, parentChain []byte, index uint32) (childKey, childChain []byte, err error) {
	if len(parentKey) != 32 || len(parentChain) != 32 {
		return nil, nil, errors.New("invalid parent key/chain length")
	}

	mac := hmac.New(sha512.New, parentChain)
	if index >= hardening {
		// Hardened: data = 0x00 || parentKey || index
		mac.Write([]byte{0x00})
		mac.Write(parentKey)
	} else {
		// Normal: data = serP(parentPubkey) || index
		priv, err := crypto.ToECDSA(parentKey)
		if err != nil {
			return nil, nil, err
		}
		pub := crypto.CompressPubkey(&priv.PublicKey)
		mac.Write(pub)
	}
	var idxBuf [4]byte
	binary.BigEndian.PutUint32(idxBuf[:], index)
	mac.Write(idxBuf[:])
	I := mac.Sum(nil)
	il := I[:32]
	ir := I[32:]

	// childKey = (parse256(IL) + parentKey) mod n
	curveOrder := crypto.S256().Params().N
	ilInt := new(big.Int).SetBytes(il)
	parentInt := new(big.Int).SetBytes(parentKey)
	childInt := new(big.Int).Add(ilInt, parentInt)
	childInt.Mod(childInt, curveOrder)
	if childInt.Sign() == 0 {
		return nil, nil, errors.New("derived child key is zero (invalid)")
	}
	if childInt.Cmp(curveOrder) >= 0 {
		return nil, nil, errors.New("derived child key >= curve order (invalid)")
	}
	childKey = childInt.FillBytes(make([]byte, 32))
	childChain = ir
	return childKey, childChain, nil
}
