package main

// non_evm_ed25519_extra.go — Ed25519 chains whose address encoding differs from
// the plain public-key families. Each encoder is written against the chain's own
// protocol documentation; no third-party wallet library is involved.
//
//   graphene (steem, hive, bitshares) — RIPEMD-160 of the raw 33-byte compressed
//     public key (0x02 || x, i.e. the Ed25519 key with a sign byte prepended),
//     base58 with the chain's address prefix.
//   nem / symbol — the NEM "account id": version byte, network byte, 20-byte
//     RIPEMD-160(pubkey), then a 4-byte checksum taken from the first four bytes
//     of Keccak-256 of the first 21 bytes, base32 encoded.
//   siacoin — blake2b-256(pubkey), base32 encoded, with a per-network prefix.

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/ripemd160" //nolint:staticcheck // required by the graphene and NEM address formats
	"golang.org/x/crypto/sha3"
)

// ---------- graphene family (steem / hive / bitshares) ----------

// graphenePrefixes maps the chain to its base58 address prefix. Steem and Hive
// share the prefix but are separate chains with separate public keys.
var graphenePrefixes = map[string]string{
	"steem":     "STM",
	"hive":      "STM",
	"bitshares": "BTS",
}

// grapheneAddress encodes an Ed25519 public key in the graphene address format.
// The address is the chain prefix followed by base58 of RIPEMD-160 over the
// compressed public key. The 0x02 sign byte is prepended because Ed25519 keys
// have no sign bit, and graphene's address derivation always includes it.
func grapheneAddress(seed []byte, path, chainType string) (string, error) {
	prefix, ok := graphenePrefixes[chainType]
	if !ok {
		return "", fmt.Errorf("unsupported graphene-family chain %q", chainType)
	}
	pub, err := edPubKey(seed, path)
	if err != nil {
		return "", err
	}
	compressed := append([]byte{0x02}, pub...)
	digest := ripemd160.New()
	digest.Write(compressed)
	return prefix + base58Encode(digest.Sum(nil)), nil
}

// graphenePub decodes a graphene address back to the raw Ed25519 public key.
func graphenePub(addr string) ([]byte, error) {
	for _, prefix := range []string{"STM", "BTS"} {
		if !strings.HasPrefix(addr, prefix) {
			continue
		}
		raw, err := base58Decode(strings.TrimPrefix(addr, prefix))
		if err != nil {
			return nil, err
		}
		if len(raw) != 20 {
			return nil, errors.New("graphene address must hash to 20 bytes")
		}
		return raw, nil
	}
	return nil, fmt.Errorf("unsupported graphene prefix in %q", addr)
}

// ---------- NEM / Symbol ----------

// nemNetworks holds the network byte of each NEM-family chain's mainnet.
var nemNetworks = map[string]byte{
	"nem":    0x68, // NEM mainnet
	"symbol": 0x68, // Symbol mainnet
}

// nemAddress builds the NEM account id: version(1) || network(1) ||
// RIPEMD-160(pubkey)(20) || checksum(4), base32 encoded. The checksum is the
// first four bytes of Keccak-256 over the preceding 21 bytes.
func nemAddress(seed []byte, path, chainType string) (string, error) {
	network, ok := nemNetworks[chainType]
	if !ok {
		return "", fmt.Errorf("unsupported NEM-family chain %q", chainType)
	}
	pub, err := edPubKey(seed, path)
	if err != nil {
		return "", err
	}
	digest := ripemd160.New()
	digest.Write(pub)
	body := make([]byte, 0, 21)
	body = append(body, 0x68, network)
	body = append(body, digest.Sum(nil)...)

	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(body)
	checksum := hasher.Sum(nil)[:4]
	return nemBase32Encode(append(body, checksum...)), nil
}

// nemBase32Encode encodes with the NEM alphabet, which is the RFC 4648 alphabet
// with the digits moved after the letters.
const nemAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"

func nemBase32Encode(data []byte) string {
	var out strings.Builder
	var acc uint32
	bits := 0
	for _, b := range data {
		acc = (acc << 8) | uint32(b)
		bits += 8
		for bits >= 5 {
			bits -= 5
			out.WriteByte(nemAlphabet[(acc>>bits)&0x1f])
		}
	}
	if bits > 0 {
		out.WriteByte(nemAlphabet[(acc<<(5-bits))&0x1f])
	}
	return out.String()
}

// ---------- Siacoin ----------

// siacoinPrefix is the mainnet address prefix. The hash is blake2b-256 of the
// Ed25519 public key, base32 encoded with Siacoin's own alphabet.
const (
	siacoinPrefix   = "sia:"
	siacoinAlphabet = "abcdefghijklmnopqrstuvwxyz234567"
)

// siacoinAddress derives the Siacoin unlock hash address.
func siacoinAddress(seed []byte, path string) (string, error) {
	pub, err := edPubKey(seed, path)
	if err != nil {
		return "", err
	}
	sum := blake2b.Sum256(pub)
	return siacoinPrefix + siaBase32Encode(sum[:]), nil
}

func siaBase32Encode(data []byte) string {
	var out strings.Builder
	var acc uint32
	bits := 0
	for _, b := range data {
		acc = (acc << 8) | uint32(b)
		bits += 8
		for bits >= 5 {
			bits -= 5
			out.WriteByte(siacoinAlphabet[(acc>>bits)&0x1f])
		}
	}
	if bits > 0 {
		out.WriteByte(siacoinAlphabet[(acc<<(5-bits))&0x1f])
	}
	return out.String()
}
