package handlers

// non_evm_crypto.go — REAL non-EVM address derivation + signing for the
// WL-MasterWallet UserWallet management layer. Ported from the canonical
// master_wallet/backend/non_evm_crypto.go + hd_derive.go + btc_helpers.go.
//
// Real crypto only — no fakes/stubs/mocks:
//   - EVM: secp256k1 + Keccak-256 (BIP-44 m/44'/60'/...) via go-ethereum
//   - Solana: SLIP-0010 Ed25519 hardened HD derivation + ed25519 sign
//   - Bitcoin: secp256k1 P2PKH (RIPEMD160(SHA256(pubkey)) + base58check)
//   - Cosmos: secp256k1 + bech32 (BIP-173) account address
//
// Uses go-ethereum crypto + golang.org/x/crypto (ed25519, ripemd160, sha3).

import (
	"crypto/ecdsa"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/ripemd160"
	"golang.org/x/crypto/sha3"
)

const hardening uint32 = 0x80000000

// ----------------------------------------------------------------------------
// BIP-32 HD derivation (secp256k1, generic path)
// ----------------------------------------------------------------------------

// derivePrivateKeyFromPath derives the secp256k1 private key for a BIP-32 path
// (e.g. "m/44'/60'/0'/0/0") from a BIP-39 seed. HMAC-SHA512 CKD per BIP-32.
func derivePrivateKeyFromPath(seed []byte, path string) (*ecdsa.PrivateKey, error) {
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
	path = strings.TrimPrefix(path, "m/")
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
		n, err := strconv.ParseUint(p, 10, 31)
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

// ckdPriv performs one BIP-32 CKD_priv step.
func ckdPriv(parentKey, parentChain []byte, index uint32) (childKey, childChain []byte, err error) {
	var data []byte
	if index >= hardening {
		// Hardened: data = 0x00 || parentKey || index(BE)
		data = make([]byte, 1+32+4)
		data[0] = 0x00
		copy(data[1:33], parentKey)
		binary.BigEndian.PutUint32(data[33:], index)
	} else {
		// Normal: data = serP(parentPubkey) || index(BE)
		pub, err := crypto.DecompressPubkey(parentKey)
		if err != nil {
			// parentKey is raw 32-byte scalar; derive pubkey.
			priv, err := crypto.ToECDSA(parentKey)
			if err != nil {
				return nil, nil, fmt.Errorf("ckdPriv: %w", err)
			}
			pub = &priv.PublicKey
		}
		pubBytes := crypto.CompressPubkey(pub)
		data = make([]byte, 33+4)
		copy(data[:33], pubBytes)
		binary.BigEndian.PutUint32(data[33:], index)
	}
	mac := hmac.New(sha512.New, parentChain)
	mac.Write(data)
	I := mac.Sum(nil)
	il, ir := I[:32], I[32:]
	// childKey = (il + parentKey) mod n
	curveOrder := crypto.S256().Params().N
	child := new(big.Int).SetBytes(il)
	child.Add(child, new(big.Int).SetBytes(parentKey))
	child.Mod(child, curveOrder)
	if child.Sign() == 0 || child.Cmp(curveOrder) >= 0 {
		return nil, nil, fmt.Errorf("ckdPriv: invalid child")
	}
	childKey = paddedBytes32(child.Bytes())
	childChain = ir
	return childKey, childChain, nil
}

func paddedBytes32(b []byte) []byte {
	if len(b) >= 32 {
		return b[len(b)-32:]
	}
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out
}

// privateKeyToAddress returns the EVM address for a secp256k1 key.
func privateKeyToAddress(key *ecdsa.PrivateKey) string {
	return crypto.PubkeyToAddress(key.PublicKey).Hex()
}

// ----------------------------------------------------------------------------
// SLIP-0010 Ed25519 HD derivation (for Solana)
// ----------------------------------------------------------------------------

// slip10DeriveEd25519 derives an Ed25519 seed from a BIP-39 seed using SLIP-0010.
// Ed25519 only supports hardened derivation (index >= 0x80000000).
func slip10DeriveEd25519(seed []byte, path string) ([]byte, error) {
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
		data := make([]byte, 1+32+4)
		data[0] = 0x00
		copy(data[1:33], key)
		binary.BigEndian.PutUint32(data[33:], p)

		mac := hmac.New(sha512.New, chainCode)
		mac.Write(data)
		sum := mac.Sum(nil)
		key = sum[:32]
		chainCode = sum[32:]
	}
	return key, nil
}

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
		if _, err := fmt.Sscanf(p, "%d", &val); err != nil {
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

func solanaAddressFromSeed(seed []byte, derivationPath string) (string, error) {
	privKeyBytes, err := slip10DeriveEd25519(seed, derivationPath)
	if err != nil {
		return "", err
	}
	pubKey := ed25519.NewKeyFromSeed(privKeyBytes).Public()
	return base58Encode(pubKey.(ed25519.PublicKey)), nil
}

// ----------------------------------------------------------------------------
// Bitcoin (P2PKH) — native implementation, no btcd
// P2PKH = base58check(0x00 || RIPEMD160(SHA256(compressed_pubkey)))
// ----------------------------------------------------------------------------

func btcAddressFromSeed(seed []byte, derivationPath string) (string, error) {
	privKey, err := derivePrivateKeyFromPath(seed, derivationPath)
	if err != nil {
		return "", err
	}
	pubKeyBytes := crypto.CompressPubkey(&privKey.PublicKey)
	sha := sha256.Sum256(pubKeyBytes)
	hasher := ripemd160.New()
	hasher.Write(sha[:])
	hash160 := hasher.Sum(nil)
	return base58CheckEncode(0x00, hash160), nil
}

// btcUTXO represents an unspent output from the blockstream API.
type btcUTXO struct {
	TxID  string
	Vout  uint32
	Value *big.Int
}

func fetchBTCUTXOs(address string) ([]btcUTXO, error) {
	url := "https://blockstream.info/api/address/" + address + "/utxo"
	resp, err := httpGet(url)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 || string(resp) == "[]" {
		return nil, nil
	}
	var raw []struct {
		TxID   string `json:"txid"`
		Vout   int    `json:"vout"`
		Value  int64  `json:"value"`
		Status struct {
			Confirmed bool `json:"confirmed"`
		} `json:"status"`
	}
	if err := json.Unmarshal(resp, &raw); err != nil {
		return nil, err
	}
	utxos := make([]btcUTXO, 0, len(raw))
	for _, r := range raw {
		if !r.Status.Confirmed {
			continue
		}
		utxos = append(utxos, btcUTXO{TxID: r.TxID, Vout: uint32(r.Vout), Value: big.NewInt(r.Value)})
	}
	return utxos, nil
}

// ----------------------------------------------------------------------------
// Cosmos (bech32)
// ----------------------------------------------------------------------------

func cosmosAddressFromSeed(seed []byte, derivationPath, prefix string) (string, error) {
	privKey, err := derivePrivateKeyFromPath(seed, derivationPath)
	if err != nil {
		return "", err
	}
	pubKeyBytes := crypto.CompressPubkey(&privKey.PublicKey)
	sha := sha256.Sum256(pubKeyBytes)
	hasher := ripemd160.New()
	hasher.Write(sha[:])
	hash160 := hasher.Sum(nil)
	return bech32Encode(prefix, hash160)
}

// cosmosSign signs a SignDoc with secp256k1 over SHA-256 (legacy amino). Returns
// 64-byte r||s + compressed pubkey.
func cosmosSign(seed []byte, derivationPath, signDoc string) ([]byte, []byte, error) {
	privKey, err := derivePrivateKeyFromPath(seed, derivationPath)
	if err != nil {
		return nil, nil, fmt.Errorf("cosmos key derivation: %w", err)
	}
	msgHash := sha256.Sum256([]byte(signDoc))
	sig, err := crypto.Sign(msgHash[:], privKey)
	if err != nil {
		return nil, nil, fmt.Errorf("cosmos sign: %w", err)
	}
	pubKeyBytes := crypto.CompressPubkey(&privKey.PublicKey)
	return sig[:64], pubKeyBytes, nil
}

// ----------------------------------------------------------------------------
// base58check encoding (Bitcoin P2PKH)
// ----------------------------------------------------------------------------

const base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

func base58Encode(data []byte) string {
	x := new(big.Int).SetBytes(data)
	base := big.NewInt(58)
	zero := big.NewInt(0)
	mod := new(big.Int)
	var result []byte
	for x.Cmp(zero) > 0 {
		x.DivMod(x, base, mod)
		result = append(result, base58Alphabet[mod.Int64()])
	}
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	for _, b := range data {
		if b != 0 {
			break
		}
		result = append([]byte{'1'}, result...)
	}
	return string(result)
}

func base58CheckEncode(version byte, payload []byte) string {
	data := make([]byte, 0, 1+len(payload)+4)
	data = append(data, version)
	data = append(data, payload...)
	first := sha256.Sum256(data)
	second := sha256.Sum256(first[:])
	data = append(data, second[:4]...)
	return base58Encode(data)
}

func base58Decode(s string) []byte {
	dec := []byte{}
	for i := 0; i < len(s); i++ {
		c := s[i]
		idx := -1
		for j := 0; j < 58; j++ {
			if base58Alphabet[j] == c {
				idx = j
				break
			}
		}
		if idx < 0 {
			continue
		}
		carry := big.NewInt(int64(idx))
		for j := len(dec) - 1; j >= 0; j-- {
			val := new(big.Int).Mul(big.NewInt(int64(dec[j])), big.NewInt(58))
			val.Add(val, carry)
			dec[j] = byte(val.Int64() & 0xFF)
			carry = new(big.Int).Rsh(val, 8)
		}
		for carry.Sign() > 0 {
			dec = append([]byte{byte(carry.Int64() & 0xFF)}, dec...)
			carry = new(big.Int).Rsh(carry, 8)
		}
	}
	for i := 0; i < len(s) && s[i] == '1'; i++ {
		dec = append([]byte{0}, dec...)
	}
	return dec
}

func base58CheckDecode(address string) (byte, []byte, bool) {
	decoded := base58Decode(address)
	if len(decoded) < 5 {
		return 0, nil, false
	}
	payload := decoded[:len(decoded)-4]
	checksum := decoded[len(decoded)-4:]
	expected := doubleSHA256(payload)
	for i := 0; i < 4; i++ {
		if checksum[i] != expected[i] {
			return 0, nil, false
		}
	}
	if len(payload) < 1 {
		return 0, nil, false
	}
	return payload[0], payload[1:], true
}

// ----------------------------------------------------------------------------
// bech32 encoding (BIP-173, for Cosmos)
// ----------------------------------------------------------------------------

const bech32Charset = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"

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

func sha256Hash(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

func doubleSHA256(data []byte) []byte {
	h1 := sha256Hash(data)
	h2 := sha256Hash(h1)
	return h2
}

func keccak256Hash(data []byte) []byte {
	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	return h.Sum(nil)
}

func hexEncode(b []byte) string { return hex.EncodeToString(b) }

func parseHex(s string) []byte {
	s = strings.TrimPrefix(s, "0x")
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil
	}
	return b
}

func httpGet(url string) ([]byte, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// bech32PrefixForChainID maps a non-EVM chain id to its bech32 account prefix.
func bech32PrefixForChainID(chainID int64) string {
	if p, ok := cosmosBech32Prefixes[chainID]; ok {
		return p
	}
	return "cosmos"
}

// cosmosBech32Prefixes is the real bech32 account-prefix map for known
// Cosmos-sdk chains. Only entries with verified canonical prefixes are listed;
// unknown chains fall back to "cosmos" (Cosmos Hub prefix) — callers must pass
// an explicit prefix via the chain record when known.
var cosmosBech32Prefixes = map[int64]string{
	118: "cosmos", // Cosmos Hub
}
