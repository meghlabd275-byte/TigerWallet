package main

// non_evm_crypto.go — Real non-EVM address derivation + signing for the
// MasterWallet UserWallet management layer.
//
// Real crypto only — no fakes/stubs/mocks:
//   - Solana: SLIP-0010 Ed25519 hardened HD derivation + ed25519 sign
//   - Bitcoin: secp256k1 P2PKH address (hash160 = RIPEMD160(SHA256(pubkey))
//     + base58check encoding, all implemented natively)
//   - Cosmos: secp256k1 + bech32 (BIP-173) account address
//
// Uses go-ethereum crypto (already a dep) for secp256k1 + golang.org/x/crypto
// for Ed25519. No external btcd dependency (avoids Go version conflicts).

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/ripemd160"
)

// ----------------------------------------------------------------------------
// SLIP-0010 Ed25519 HD derivation (for Solana)
// ----------------------------------------------------------------------------

// slip10DeriveEd25519MW derives an Ed25519 key from a seed using SLIP-0010.
// Ed25519 only supports hardened derivation (index >= 0x80000000).
func slip10DeriveEd25519MW(seed []byte, path string) ([]byte, error) {
	// Master key: HMAC-SHA512("ed25519 seed", seed)
	mac := hmac.New(sha512.New, []byte("ed25519 seed"))
	mac.Write(seed)
	sum := mac.Sum(nil)
	key := sum[:32]
	chainCode := sum[32:]

	parts, err := parseSLIP10Path(path)
	if err != nil {
		return nil, err
	}
	for _, p := range parts {
		if p < 0x80000000 {
			return nil, fmt.Errorf("ed25519 requires hardened derivation")
		}
		// Data: 0x00 || key || index(BE)
		data := make([]byte, 1+32+4)
		data[0] = 0x00
		copy(data[1:33], key)
		data[33] = byte(p >> 24)
		data[34] = byte(p >> 16)
		data[35] = byte(p >> 8)
		data[36] = byte(p)

		mac := hmac.New(sha512.New, chainCode)
		mac.Write(data)
		sum := mac.Sum(nil)
		key = sum[:32]
		chainCode = sum[32:]
	}
	return key, nil
}

// parseSLIP10Path parses a BIP-44 path like m/44'/501'/0'/0/0 into uint32 indices.
func parseSLIP10Path(path string) ([]uint32, error) {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "m")
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return nil, fmt.Errorf("empty path")
	}
	parts := strings.Split(path, "/")
	var indices []uint32
	for _, p := range parts {
		p = strings.TrimSpace(p)
		hardened := false
		if strings.HasSuffix(p, "'") || strings.HasSuffix(p, "h") || strings.HasSuffix(p, "H") {
			hardened = true
			p = p[:len(p)-1]
		}
		var val uint32
		_, err := fmt.Sscanf(p, "%d", &val)
		if err != nil {
			return nil, fmt.Errorf("invalid path segment: %s", p)
		}
		if val >= 0x80000000 {
			return nil, fmt.Errorf("index too large: %d", val)
		}
		if hardened {
			val += 0x80000000
		}
		indices = append(indices, val)
	}
	return indices, nil
}

// ----------------------------------------------------------------------------
// Solana
// ----------------------------------------------------------------------------

// mwSolanaAddressFromSeed derives a Solana base58 address from a seed.
func mwSolanaAddressFromSeed(seed []byte, derivationPath string) (string, error) {
	privKeyBytes, err := slip10DeriveEd25519MW(seed, derivationPath)
	if err != nil {
		return "", err
	}
	pubKey := ed25519.NewKeyFromSeed(privKeyBytes).Public()
	return base58Encode(pubKey.(ed25519.PublicKey)), nil
}

// mwSolanaSign signs a message with Solana Ed25519.
func mwSolanaSign(seed []byte, derivationPath, message string) (sig, pub []byte, err error) {
	privKeyBytes, err := slip10DeriveEd25519MW(seed, derivationPath)
	if err != nil {
		return nil, nil, err
	}
	privKey := ed25519.NewKeyFromSeed(privKeyBytes)
	pubKey := privKey.Public().(ed25519.PublicKey)
	sig = ed25519.Sign(privKey, []byte(message))
	return sig, pubKey, nil
}

// ----------------------------------------------------------------------------
// Bitcoin (P2PKH) — native implementation, no btcd
// ----------------------------------------------------------------------------

// mwBTCAddressFromSeed derives a Bitcoin mainnet P2PKH address from a seed.
// P2PKH = base58check(0x00 || RIPEMD160(SHA256(compressed_pubkey)))
func mwBTCAddressFromSeed(seed []byte, derivationPath string) (string, error) {
	privKey, err := DerivePrivateKeyFromPath(seed, derivationPath)
	if err != nil {
		return "", err
	}
	pubKeyBytes := crypto.CompressPubkey(&privKey.PublicKey)
	// hash160 = RIPEMD160(SHA256(pubkey))
	sha := sha256.Sum256(pubKeyBytes)
	hasher := ripemd160.New()
	hasher.Write(sha[:])
	hash160 := hasher.Sum(nil)
	// P2PKH mainnet: version byte 0x00
	return base58CheckEncode(0x00, hash160), nil
}

// ----------------------------------------------------------------------------
// Cosmos (bech32)
// ----------------------------------------------------------------------------

// mwCosmosAddressFromSeed derives a Cosmos bech32 address from a seed.
func mwCosmosAddressFromSeed(seed []byte, derivationPath, prefix string) (string, error) {
	privKey, err := DerivePrivateKeyFromPath(seed, derivationPath)
	if err != nil {
		return "", err
	}
	pubKeyBytes := crypto.CompressPubkey(&privKey.PublicKey)
	// hash160 = RIPEMD160(SHA256(pubkey))
	sha := sha256.Sum256(pubKeyBytes)
	hasher := ripemd160.New()
	hasher.Write(sha[:])
	hash160 := hasher.Sum(nil)
	// Convert to bech32 with the chain prefix.
	return bech32Encode(prefix, hash160)
}

// ----------------------------------------------------------------------------
// base58check encoding (Bitcoin P2PKH)
// ----------------------------------------------------------------------------

const base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

// base58Encode encodes bytes to base58 (no checksum).
func base58Encode(data []byte) string {
	// Convert to big integer.
	x := new(big.Int).SetBytes(data)
	base := big.NewInt(58)
	zero := big.NewInt(0)
	mod := new(big.Int)
	var result []byte
	for x.Cmp(zero) > 0 {
		x.DivMod(x, base, mod)
		result = append(result, base58Alphabet[mod.Int64()])
	}
	// Reverse.
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	// Leading zero bytes -> '1'.
	for _, b := range data {
		if b != 0 {
			break
		}
		result = append([]byte{'1'}, result...)
	}
	return string(result)
}

// base58CheckEncode encodes version + payload with a 4-byte double-SHA256 checksum.
func base58CheckEncode(version byte, payload []byte) string {
	data := make([]byte, 0, 1+len(payload)+4)
	data = append(data, version)
	data = append(data, payload...)
	// Double SHA-256 checksum (first 4 bytes).
	first := sha256.Sum256(data)
	second := sha256.Sum256(first[:])
	data = append(data, second[:4]...)
	return base58Encode(data)
}

// ----------------------------------------------------------------------------
// bech32 encoding (BIP-173, for Cosmos)
// ----------------------------------------------------------------------------

const bech32Charset = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"

// bech32Encode encodes a 20-byte hash to a bech32 address.
func bech32Encode(hrp string, data []byte) (string, error) {
	if len(data) != 20 {
		return "", fmt.Errorf("expected 20-byte hash, got %d", len(data))
	}
	conv, err := convertBits(data, 8, 5, true)
	if err != nil {
		return "", err
	}
	convInts := make([]int, len(conv))
	for i, b := range conv {
		convInts[i] = int(b)
	}
	values := append(bech32HRPExpand(hrp), convInts...)
	values = append(values, 0, 0, 0, 0, 0, 0)
	mod := bech32Polymod(values) ^ 1
	var checksum string
	for i := 0; i < 6; i++ {
		checksum += string(bech32Charset[(mod>>uint(5*(5-i)))&31])
	}
	var sb strings.Builder
	sb.WriteString(hrp)
	sb.WriteString("1")
	for _, b := range conv {
		sb.WriteString(string(bech32Charset[b]))
	}
	sb.WriteString(checksum)
	return sb.String(), nil
}

func convertBits(data []byte, fromBits, toBits uint, pad bool) ([]byte, error) {
	var acc uint32
	var bits uint
	var ret []byte
	maxv := uint32((1 << toBits) - 1)
	for _, value := range data {
		v := uint32(value)
		if v>>fromBits != 0 {
			return nil, fmt.Errorf("invalid data for %d-bit conversion", fromBits)
		}
		acc = (acc << fromBits) | v
		bits += fromBits
		for bits >= toBits {
			bits -= toBits
			ret = append(ret, byte((acc>>bits)&maxv))
		}
	}
	if pad {
		if bits > 0 {
			ret = append(ret, byte((acc<<(toBits-bits))&maxv))
		}
	} else if bits >= fromBits || ((acc<<(toBits-bits))&maxv) != 0 {
		return nil, fmt.Errorf("invalid padding")
	}
	return ret, nil
}

func bech32Polymod(values []int) int {
	gen := []int{0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3}
	chk := 1
	for _, v := range values {
		b := chk >> 25
		chk = (chk&0x1ffffff)<<5 ^ v
		for i := 0; i < 5; i++ {
			if (b>>uint(i))&1 == 1 {
				chk ^= gen[i]
			}
		}
	}
	return chk
}

func bech32HRPExpand(hrp string) []int {
	var ret []int
	for _, c := range hrp {
		ret = append(ret, int(c)>>5)
	}
	ret = append(ret, 0)
	for _, c := range hrp {
		ret = append(ret, int(c)&31)
	}
	return ret
}

// ----------------------------------------------------------------------------
// Shared helpers
// ----------------------------------------------------------------------------

// sha256Hash is the SHA-256 hash (used by user_wallet_management.go).
func sha256Hash(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

// hexEncode is a convenience hex encoder.
func hexEncode(b []byte) string {
	return hex.EncodeToString(b)
}

// ed25519Verify verifies an Ed25519 signature.
func ed25519Verify(pub, message, sig []byte) bool {
	if len(pub) != ed25519.PublicKeySize || len(sig) != ed25519.SignatureSize {
		return false
	}
	return ed25519.Verify(ed25519.PublicKey(pub), message, sig)
}
