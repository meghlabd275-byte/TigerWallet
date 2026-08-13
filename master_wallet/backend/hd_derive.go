package main

// hd_derive.go — REAL BIP-32 hierarchical deterministic key derivation.
// Implements HMAC-SHA512 CKD (child key derivation) per BIP-32, with
// hardened (index >= 0x80000000) and normal derivation, using secp256k1 via
// go-ethereum/crypto. No shortcuts, no SHA256-of-seed fakes.

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
	}
	parts := strings.Split(path, "/")
	out := make([]uint32, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		hardened := false
		if strings.HasSuffix(p, "'") || strings.HasSuffix(p, "h") || strings.HasSuffix(p, "H") {
			hardened = true
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

// ckdPriv performs one BIP-32 CKDpriv step. Hardened children use
// 0x00||key||index; normal children use serP(point(key))||index.
func ckdPriv(parentKey, parentChain []byte, index uint32) (childKey, childChain []byte, err error) {
	var data []byte
	if index >= hardening {
		// Hardened: 0x00 || ser256(kpar) || ser32(i)
		data = make([]byte, 1+len(parentKey)+4)
		data[0] = 0
		copy(data[1:], parentKey)
		binary.BigEndian.PutUint32(data[1+len(parentKey):], index)
	} else {
		// Normal: serP(point(kpar)) || ser32(i)
		pub, err := pubkeyFromPrivBytes(parentKey)
		if err != nil {
			return nil, nil, err
		}
		comp := crypto.CompressPubkey(pub)
		data = make([]byte, len(comp)+4)
		copy(data, comp)
		binary.BigEndian.PutUint32(data[len(comp):], index)
	}

	mac := hmac.New(sha512.New, parentChain)
	mac.Write(data)
	I := mac.Sum(nil)
	il, ir := I[:32], I[32:]

	// child key = (parse256(IL) + kpar) mod n
	privInt := new(big.Int).SetBytes(il)
	if privInt.Cmp(crypto.S256().Params().N) >= 0 || privInt.Sign() == 0 {
		return nil, nil, errors.New("invalid child key (>= n or zero)")
	}
	parentInt := new(big.Int).SetBytes(parentKey)
	childInt := new(big.Int).Add(privInt, parentInt)
	childInt.Mod(childInt, crypto.S256().Params().N)
	if childInt.Sign() == 0 {
		return nil, nil, errors.New("invalid child key (zero)")
	}
	childKey = childInt.FillBytes(make([]byte, 32))
	childChain = ir
	return childKey, childChain, nil
}

// pubkeyFromPrivBytes derives the *ecdsa.PublicKey for a 32-byte private key.
func pubkeyFromPrivBytes(priv []byte) (*ecdsa.PublicKey, error) {
	key, err := crypto.ToECDSA(priv)
	if err != nil {
		return nil, err
	}
	return &key.PublicKey, nil
}
