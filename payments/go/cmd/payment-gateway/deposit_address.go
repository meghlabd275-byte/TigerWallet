package main

// deposit_address.go - deterministic deposit addresses for the payment
// gateway.
//
// The gateway previously produced "payment addresses" by hashing a timestamp
// and encoding the digest: a string that looks like an address but has no
// corresponding private key. Funds sent to such an address are unrecoverable,
// so that code could only ever have lost customer money.
//
// Addresses are now derived the way a wallet derives them. A per-user account
// seed comes from HKDF-SHA256 over the master seed, the BIP-32 chain is walked
// to the coin's account path, and the public key is encoded per chain family.
// The gateway can therefore re-derive any address it issued and, with the
// master seed, spend what arrives there.
//
// Fail-closed: with no master seed the gateway refuses to issue an address
// rather than inventing one.

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strconv"
	"strings"

	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/ripemd160"
)

// ---------------------------------------------------------------------------
// Keccak-256 (Ethereum variant; original 0x01 padding, not SHA3's 0x06)
// ---------------------------------------------------------------------------

var keccakRoundConstants = [24]uint64{
	0x0000000000000001, 0x0000000000008082, 0x800000000000808A,
	0x8000000080008000, 0x000000000000808B, 0x0000000080000001,
	0x8000000080008081, 0x8000000000008009, 0x000000000000008A,
	0x0000000000000088, 0x0000000080008009, 0x000000008000000A,
	0x000000008000808B, 0x800000000000008B, 0x8000000000008089,
	0x8000000000008003, 0x8000000000008002, 0x8000000000000080,
	0x000000000000800A, 0x800000008000000A, 0x8000000080008081,
	0x8000000000008080, 0x0000000080000001, 0x8000000080008008,
}

var keccakRotation = [5][5]uint{
	{0, 36, 3, 41, 18},
	{1, 44, 10, 45, 2},
	{62, 6, 43, 15, 61},
	{28, 55, 25, 21, 56},
	{27, 20, 39, 8, 14},
}

func rotl64(x uint64, n uint) uint64 {
	if n == 0 {
		return x
	}
	return (x << n) | (x >> (64 - n))
}

func keccakF1600(a *[25]uint64) {
	for round := 0; round < 24; round++ {
		var c, d [5]uint64
		for x := 0; x < 5; x++ {
			c[x] = a[x] ^ a[x+5] ^ a[x+10] ^ a[x+15] ^ a[x+20]
		}
		for x := 0; x < 5; x++ {
			d[x] = c[(x+4)%5] ^ rotl64(c[(x+1)%5], 1)
		}
		for x := 0; x < 5; x++ {
			for y := 0; y < 5; y++ {
				a[x+5*y] ^= d[x]
			}
		}

		var b [25]uint64
		for x := 0; x < 5; x++ {
			for y := 0; y < 5; y++ {
				b[y+5*((2*x+3*y)%5)] = rotl64(a[x+5*y], keccakRotation[x][y])
			}
		}

		for x := 0; x < 5; x++ {
			for y := 0; y < 5; y++ {
				a[x+5*y] = b[x+5*y] ^ (^b[(x+1)%5+5*y] & b[(x+2)%5+5*y])
			}
		}

		a[0] ^= keccakRoundConstants[round]
	}
}

// keccak256 returns the 32-byte Keccak-256 digest of data.
func keccak256(data []byte) []byte {
	const rate = 136
	padded := make([]byte, 0, len(data)+rate)
	padded = append(padded, data...)
	padded = append(padded, 0x01)
	for len(padded)%rate != 0 {
		padded = append(padded, 0x00)
	}
	padded[len(padded)-1] ^= 0x80

	var state [25]uint64
	for offset := 0; offset < len(padded); offset += rate {
		block := padded[offset : offset+rate]
		for i := 0; i < rate/8; i++ {
			state[i] ^= binary.LittleEndian.Uint64(block[i*8 : (i+1)*8])
		}
		keccakF1600(&state)
	}

	out := make([]byte, 32)
	for i := 0; i < 4; i++ {
		binary.LittleEndian.PutUint64(out[i*8:(i+1)*8], state[i])
	}
	return out
}

// ---------------------------------------------------------------------------
// secp256k1
// ---------------------------------------------------------------------------

var (
	secpP, _  = new(big.Int).SetString("fffffffffffffffffffffffffffffffffffffffffffffffffffffffefffffc2f", 16)
	secpN, _  = new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
	secpGx, _ = new(big.Int).SetString("79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798", 16)
	secpGy, _ = new(big.Int).SetString("483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8", 16)
)

type curvePoint struct {
	x, y *big.Int
	inf  bool
}

func secpAdd(p, q curvePoint) curvePoint {
	if p.inf {
		return q
	}
	if q.inf {
		return p
	}

	var lambda *big.Int
	if p.x.Cmp(q.x) == 0 {
		if p.y.Cmp(q.y) != 0 || p.y.Sign() == 0 {
			return curvePoint{inf: true}
		}
		// slope = 3x^2 / 2y
		num := new(big.Int).Mul(p.x, p.x)
		num.Mul(num, big.NewInt(3))
		den := new(big.Int).Lsh(p.y, 1)
		inv := new(big.Int).ModInverse(den, secpP)
		if inv == nil {
			return curvePoint{inf: true}
		}
		lambda = num.Mul(num, inv)
	} else {
		num := new(big.Int).Sub(q.y, p.y)
		den := new(big.Int).Sub(q.x, p.x)
		inv := new(big.Int).ModInverse(den, secpP)
		if inv == nil {
			return curvePoint{inf: true}
		}
		lambda = num.Mul(num, inv)
	}
	lambda.Mod(lambda, secpP)

	x3 := new(big.Int).Mul(lambda, lambda)
	x3.Sub(x3, p.x)
	x3.Sub(x3, q.x)
	x3.Mod(x3, secpP)

	y3 := new(big.Int).Sub(p.x, x3)
	y3.Mul(y3, lambda)
	y3.Sub(y3, p.y)
	y3.Mod(y3, secpP)

	return curvePoint{x: x3, y: y3}
}

func secpMul(k *big.Int, point curvePoint) curvePoint {
	result := curvePoint{inf: true}
	addend := point
	for i := 0; i < k.BitLen(); i++ {
		if k.Bit(i) == 1 {
			result = secpAdd(result, addend)
		}
		addend = secpAdd(addend, addend)
	}
	return result
}

func secpBaseMul(k *big.Int) curvePoint {
	return secpMul(k, curvePoint{x: new(big.Int).Set(secpGx), y: new(big.Int).Set(secpGy)})
}

func padded32(v *big.Int) []byte {
	out := make([]byte, 32)
	b := v.Bytes()
	copy(out[32-len(b):], b)
	return out
}

// ---------------------------------------------------------------------------
// BIP-32
// ---------------------------------------------------------------------------

const hardenedOffset uint32 = 0x80000000

// ckdPriv performs the private-parent child key derivation of BIP-32.
func ckdPriv(parentKey, parentChain []byte, index uint32) ([]byte, []byte, error) {
	var data []byte
	if index >= hardenedOffset {
		data = append([]byte{0x00}, parentKey...)
	} else {
		d := new(big.Int).SetBytes(parentKey)
		point := secpBaseMul(d)
		if point.inf {
			return nil, nil, errors.New("bip32: parent key is the point at infinity")
		}
		data = append([]byte{0x02 | byte(point.y.Bit(0))}, padded32(point.x)...)
	}
	idxBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(idxBytes, index)
	data = append(data, idxBytes...)

	mac := hmac.New(sha512.New, parentChain)
	mac.Write(data)
	I := mac.Sum(nil)
	IL := new(big.Int).SetBytes(I[:32])
	if IL.Cmp(secpN) >= 0 {
		return nil, nil, errors.New("bip32: derived key is out of range")
	}
	ki := new(big.Int).SetBytes(parentKey)
	ki.Add(ki, IL)
	ki.Mod(ki, secpN)
	if ki.Sign() == 0 {
		return nil, nil, errors.New("bip32: derived key is zero")
	}
	return padded32(ki), I[32:], nil
}

// hdDerivePath walks a BIP-32 path such as "m/44'/60'/0'/0/0" from a seed and
// returns the private key and the compressed public key.
func hdDerivePath(seed []byte, path string) ([]byte, []byte, error) {
	if len(seed) == 0 {
		return nil, nil, errors.New("hd: empty seed")
	}

	mac := hmac.New(sha512.New, []byte("Bitcoin seed"))
	mac.Write(seed)
	I := mac.Sum(nil)
	key, chain := I[:32], I[32:]

	trimmed := strings.TrimSpace(path)
	if trimmed == "m" || trimmed == "m/" {
		trimmed = "m/"
	} else if !strings.HasPrefix(trimmed, "m/") {
		return nil, nil, fmt.Errorf("hd: path %q must start with m/", path)
	}

	for _, segment := range strings.Split(strings.TrimPrefix(trimmed, "m/"), "/") {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		hardened := strings.HasSuffix(segment, "'") || strings.HasSuffix(segment, "h") || strings.HasSuffix(segment, "H")
		if hardened {
			segment = segment[:len(segment)-1]
		}
		idx, err := strconv.ParseUint(segment, 10, 32)
		if err != nil {
			return nil, nil, fmt.Errorf("hd: bad path segment %q", segment)
		}
		index := uint32(idx)
		if hardened {
			index += hardenedOffset
		}
		key, chain, err = ckdPriv(key, chain, index)
		if err != nil {
			return nil, nil, err
		}
	}

	d := new(big.Int).SetBytes(key)
	point := secpBaseMul(d)
	if point.inf {
		return nil, nil, errors.New("hd: derived the point at infinity")
	}
	compressed := append([]byte{0x02 | byte(point.y.Bit(0))}, padded32(point.x)...)
	return key, compressed, nil
}

// ---------------------------------------------------------------------------
// Address encodings
// ---------------------------------------------------------------------------

const base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

func base58Encode(input []byte) string {
	x := new(big.Int).SetBytes(input)
	base := big.NewInt(58)
	mod := new(big.Int)
	var out []byte
	for x.Sign() > 0 {
		x.DivMod(x, base, mod)
		out = append([]byte{base58Alphabet[mod.Int64()]}, out...)
	}
	for _, b := range input {
		if b != 0 {
			break
		}
		out = append([]byte{'1'}, out...)
	}
	return string(out)
}

func base58checkEncode(payload []byte) string {
	first := sha256.Sum256(payload)
	second := sha256.Sum256(first[:])
	return base58Encode(append(append([]byte{}, payload...), second[:4]...))
}

func hash160(b []byte) []byte {
	sha := sha256.Sum256(b)
	r := ripemd160.New()
	r.Write(sha[:])
	return r.Sum(nil)
}

const bech32Charset = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"

func bech32Polymod(values []byte) uint32 {
	generator := [5]uint32{0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3}
	chk := uint32(1)
	for _, v := range values {
		top := chk >> 25
		chk = (chk&0x1ffffff)<<5 ^ uint32(v)
		for i := 0; i < 5; i++ {
			if (top>>uint(i))&1 == 1 {
				chk ^= generator[i]
			}
		}
	}
	return chk
}

func bech32HrpExpand(hrp string) []byte {
	out := make([]byte, 0, len(hrp)*2+1)
	for _, c := range []byte(hrp) {
		out = append(out, c>>5)
	}
	out = append(out, 0)
	for _, c := range []byte(hrp) {
		out = append(out, c&31)
	}
	return out
}

func convertBits(data []byte, fromBits, toBits uint, pad bool) ([]byte, error) {
	var acc uint32
	var bits uint
	maxv := uint32(1)<<toBits - 1
	out := make([]byte, 0, len(data)*int(fromBits)/int(toBits)+1)
	for _, b := range data {
		acc = acc<<fromBits | uint32(b)
		bits += fromBits
		for bits >= toBits {
			bits -= toBits
			out = append(out, byte(acc>>bits&maxv))
		}
	}
	if pad {
		if bits > 0 {
			out = append(out, byte(acc<<(toBits-bits)&maxv))
		}
	} else if bits >= fromBits || (acc<<(toBits-bits))&maxv != 0 {
		return nil, errors.New("bech32: non-zero padding")
	}
	return out, nil
}

// bech32Encode encodes a segwit address. The witness version is already a
// 5-bit value, so it is prepended after the conversion rather than converted
// with the program: converting 21 bytes together yields 34 groups instead of
// the correct 33 and produces an address no node accepts.
func bech32Encode(hrp string, witnessVersion byte, program []byte) (string, error) {
	if witnessVersion > 16 {
		return "", fmt.Errorf("bech32: witness version %d is out of range", witnessVersion)
	}
	if len(program) < 2 || len(program) > 40 {
		return "", fmt.Errorf("bech32: witness program must be 2..40 bytes, got %d", len(program))
	}
	converted, err := convertBits(program, 8, 5, true)
	if err != nil {
		return "", err
	}
	conv := append([]byte{witnessVersion}, converted...)
	combined := append(bech32HrpExpand(hrp), conv...)
	combined = append(combined, 0, 0, 0, 0, 0, 0)
	polymod := bech32Polymod(combined) ^ 1
	checksum := make([]byte, 6)
	for i := 0; i < 6; i++ {
		checksum[i] = byte(polymod >> uint(5*(5-i)) & 31)
	}
	var sb strings.Builder
	sb.WriteString(hrp)
	sb.WriteByte('1')
	for _, p := range append(conv, checksum...) {
		sb.WriteByte(bech32Charset[p])
	}
	return sb.String(), nil
}

// ---------------------------------------------------------------------------
// Per-currency derivation
// ---------------------------------------------------------------------------

type coinSpec struct {
	CoinType uint32
	Kind     string // evm | bech32 | base58 | tron
	Version  byte   // base58 P2PKH version byte
	HRP      string // bech32 human-readable part
}

var coinSpecs = map[string]coinSpec{
	"BTC":   {CoinType: 0, Kind: "bech32", HRP: "bc"},
	"ETH":   {CoinType: 60, Kind: "evm"},
	"USDT":  {CoinType: 60, Kind: "evm"}, // ERC-20 on Ethereum
	"USDC":  {CoinType: 60, Kind: "evm"},
	"BNB":   {CoinType: 714, Kind: "evm"},
	"MATIC": {CoinType: 966, Kind: "evm"},
	"TRX":   {CoinType: 195, Kind: "tron"},
	"LTC":   {CoinType: 2, Kind: "base58", Version: 0x30},
	"DOGE":  {CoinType: 3, Kind: "base58", Version: 0x1e},
}

// derivePaymentAddress returns the address the gateway controls for one
// (user, currency) pair. It is a pure function of the master seed, so the
// gateway can always re-derive it.
func derivePaymentAddress(masterSeed []byte, userID uint, currency string) (string, error) {
	if len(masterSeed) == 0 {
		return "", errors.New("crypto payments are not configured: master seed is unset")
	}
	spec, ok := coinSpecs[strings.ToUpper(currency)]
	if !ok {
		return "", fmt.Errorf("unsupported payment currency %q", currency)
	}

	// Per-user account seed, domain-separated from any other use of the
	// master seed.
	info := fmt.Sprintf("payment-gateway|user|%d|%s", userID, strings.ToUpper(currency))
	reader := hkdf.New(sha256.New, masterSeed, []byte("tigerwallet-payments-deposit-v1"), []byte(info))
	accountSeed := make([]byte, 32)
	if _, err := io.ReadFull(reader, accountSeed); err != nil {
		return "", err
	}

	path := fmt.Sprintf("m/44'/%d'/0'/0/0", spec.CoinType)
	_, compressed, err := hdDerivePath(accountSeed, path)
	if err != nil {
		return "", err
	}

	switch spec.Kind {
	case "evm":
		// The EVM address is the last 20 bytes of keccak256(uncompressed pubkey).
		uncompressed := uncompressedPubkey(compressed)
		digest := keccak256(uncompressed[1:])
		return "0x" + fmt.Sprintf("%x", digest[12:]), nil
	case "tron":
		uncompressed := uncompressedPubkey(compressed)
		digest := keccak256(uncompressed[1:])
		payload := append([]byte{0x41}, digest[12:]...)
		return base58checkEncode(payload), nil
	case "bech32":
		pkh := hash160(compressed)
		return bech32Encode(spec.HRP, 0x00, pkh)
	case "base58":
		pkh := hash160(compressed)
		return base58checkEncode(append([]byte{spec.Version}, pkh...)), nil
	default:
		return "", fmt.Errorf("unsupported address kind %q", spec.Kind)
	}
}

// uncompressedPubkey expands a compressed secp256k1 point to 0x04||X||Y.
func uncompressedPubkey(compressed []byte) []byte {
	if len(compressed) != 33 {
		return nil
	}
	x := new(big.Int).SetBytes(compressed[1:])
	// y^2 = x^3 + 7 (mod p)
	y2 := new(big.Int).Mul(x, x)
	y2.Mul(y2, x)
	y2.Add(y2, big.NewInt(7))
	y2.Mod(y2, secpP)

	// y = y2^((p+1)/4) for p = 3 mod 4
	exp := new(big.Int).Add(secpP, big.NewInt(1))
	exp.Rsh(exp, 2)
	y := new(big.Int).Exp(y2, exp, secpP)

	// Pick the root whose parity matches the prefix byte.
	if byte(y.Bit(0)) != compressed[0]&1 {
		y.Sub(secpP, y)
	}

	return append(append([]byte{0x04}, padded32(x)...), padded32(y)...)
}
