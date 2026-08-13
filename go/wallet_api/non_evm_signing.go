package main

// non_evm_signing.go — REAL non-EVM transaction + message signing for the
// canonical wallet backend. Three families, all mainnet, no fakes, no testnet
// branches, no stubs:
//
//   1. Solana — Ed25519 (golang.org/x/crypto/ed25519) over SLIP-0010 hardened
//      HD derivation (m/44'/501'/0'/0/0). Signs arbitrary message bytes; the
//      64-byte Ed25519 signature is verifiable on-chain.
//   2. Bitcoin — legacy P2PKH transaction builder + SIGHASH_ALL signer using
//      btcec/v2 secp256k1. Produces a broadcast-ready raw transaction (hex)
//      with canonical DER ECDSA signatures. Legacy (pre-segwit) inputs — the
//      standard for m/44'/0'/0'/0/0 wallets.
//   3. Cosmos — secp256k1 (go-ethereum/crypto) over a Cosmos SDK SignDoc
//      serialized as canonical Amino JSON (SIGN_MODE_LEGACY_AMINO_JSON).
//      Produces a secp256k1 r||s signature verifiable by any Cosmos SDK chain.
//
// All key material is derived from the same BIP-39 seed the EVM path uses
// (DecryptSeed -> SLIP-0010/BIP-44 path per family). No private keys are
// logged or returned; only signed payloads + signatures leave this module.

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/ethereum/go-ethereum/crypto"
	ed "golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/ripemd160"
)

// ---------------------------------------------------------------------------
// SLIP-0010 Ed25519 HD derivation (Solana + other Ed25519 chains)
// ---------------------------------------------------------------------------

// slip10DeriveEd25519 implements SLIP-0010 Ed25519 hierarchical derivation.
// Ed25519 supports ONLY hardened derivation (index >= 0x80000000).
//
// Reference: https://github.com/satoshilabs/slips/blob/master/slip-0010.md
func slip10DeriveEd25519(seed []byte, path string) ([]byte, error) {
	// Master: HMAC-SHA512(key="ed25519 seed", msg=seed) -> {IL, IR}
	mac := hmac.New(sha512.New, []byte("ed25519 seed"))
	mac.Write(seed)
	I := mac.Sum(nil)
	il := I[:32]  // master private key
	ir := I[32:]  // master chain code

	segments, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	for _, idx := range segments {
		if idx < 0x80000000 {
			return nil, errors.New("ed25519 SLIP-0010 requires hardened derivation only")
		}
		mac := hmac.New(sha512.New, ir) // chain code as key
		var buf [37]byte
		buf[0] = 0x00
		copy(buf[1:33], il)
		binary.BigEndian.PutUint32(buf[33:], idx)
		mac.Write(buf[:])
		I = mac.Sum(nil)
		il = I[:32]
		ir = I[32:]
	}
	if len(il) != 32 {
		return nil, errors.New("ed25519 derived seed length != 32")
	}
	return il, nil
}

// ---------------------------------------------------------------------------
// Solana signing
// ---------------------------------------------------------------------------

// SolanaSign derives the Solana Ed25519 keypair from the seed + BIP-44 path
// and signs the message bytes. Returns the 64-byte Ed25519 signature + pubkey.
func SolanaSign(seed []byte, derivationPath, message string) (sig, pub []byte, err error) {
	seedEd, err := slip10DeriveEd25519(seed, derivationPath)
	if err != nil {
		return nil, nil, fmt.Errorf("solana key derivation: %w", err)
	}
	priv := ed.NewKeyFromSeed(seedEd)
	pub = priv.Public().(ed.PublicKey)
	sig = ed.Sign(priv, []byte(message))
	if len(sig) != ed.SignatureSize {
		return nil, nil, errors.New("ed25519 signature length mismatch")
	}
	return sig, pub, nil
}

// SolanaAddressFromSeed returns the base58-encoded Solana public key address.
func SolanaAddressFromSeed(seed []byte, derivationPath string) (string, error) {
	seedEd, err := slip10DeriveEd25519(seed, derivationPath)
	if err != nil {
		return "", err
	}
	priv := ed.NewKeyFromSeed(seedEd)
	pub := priv.Public().(ed.PublicKey)
	return base58Encode(pub), nil
}

// ---------------------------------------------------------------------------
// Bitcoin signing (legacy P2PKH, SIGHASH_ALL)
// ---------------------------------------------------------------------------

// BTCInput describes one input to a Bitcoin transaction.
type BTCInput struct {
	TxID         string // hex big-endian funding txid
	Vout         uint32 // output index
	ScriptPubKey []byte // previous output P2PKH scriptPubKey
}

// BTCOutput describes one transaction output.
type BTCOutput struct {
	Address   string // base58check P2PKH destination
	AmountSat int64  // satoshis
}

// BTCSign builds a signed legacy Bitcoin transaction and returns the raw tx
// hex. Each input is signed with the secp256k1 key from seed + path.
func BTCSign(seed []byte, derivationPath string, inputs []BTCInput, outputs []BTCOutput) (string, error) {
	privEcdsa, err := hdDerive(seed, derivationPath)
	if err != nil {
		return "", fmt.Errorf("bitcoin key derivation: %w", err)
	}
	priv, pubKey := btcec.PrivKeyFromBytes(crypto.FromECDSA(privEcdsa))

	// Resolve destination P2PKH scriptPubKeys from addresses.
	var txOuts []*btcTxOut
	for _, o := range outputs {
		pkh, err := decodeP2PKHAddress(o.Address)
		if err != nil {
			return "", fmt.Errorf("invalid output address %q: %w", o.Address, err)
		}
		script, err := p2pkhScript(pkh)
		if err != nil {
			return "", err
		}
		txOuts = append(txOuts, &btcTxOut{value: o.AmountSat, pkScript: script})
	}

	// Build the legacy transaction (version 1).
	tx := &btcTx{version: 1, lockTime: 0}
	for _, in := range inputs {
		hash, err := reverseHexHash(in.TxID)
		if err != nil {
			return "", fmt.Errorf("invalid input txid %q: %w", in.TxID, err)
		}
		tx.inputs = append(tx.inputs, &btcTxIn{
			prevHash:  hash,
			prevIndex: in.Vout,
			sequence:  0xffffffff,
		})
	}
	for _, o := range txOuts {
		tx.outputs = append(tx.outputs, o)
	}

	// Sign each input with SIGHASH_ALL (legacy pre-segwit).
	const sighashAll = 0x01
	for i, in := range inputs {
		sigHash, err := tx.legacySigHash(i, in.ScriptPubKey)
		if err != nil {
			return "", fmt.Errorf("calc sighash input %d: %w", i, err)
		}
		sig := ecdsa.Sign(priv, sigHash)
		derSig := sig.Serialize()
		scriptSigWithHash := append(derSig, byte(sighashAll))
		scriptSig, err := p2pkhScriptSig(scriptSigWithHash, pubKey.SerializeCompressed())
		if err != nil {
			return "", fmt.Errorf("build scriptSig: %w", err)
		}
		tx.inputs[i].scriptSig = scriptSig
	}

	raw, err := tx.serialize()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

// BTCAddressFromSeed returns the base58check P2PKH address (mainnet) for a
// seed + derivation path.
func BTCAddressFromSeed(seed []byte, derivationPath string) (string, error) {
	privEcdsa, err := hdDerive(seed, derivationPath)
	if err != nil {
		return "", err
	}
	_, pubKey := btcec.PrivKeyFromBytes(crypto.FromECDSA(privEcdsa))
	pkh := hash160(pubKey.SerializeCompressed())
	return base58checkEncode(append([]byte{0x00}, pkh...)), nil
}

// ---------------------------------------------------------------------------
// Minimal legacy Bitcoin tx serializer + SIGHASH_ALL (pre-segwit)
// ---------------------------------------------------------------------------

type btcTxIn struct {
	prevHash  []byte // 32 bytes little-endian
	prevIndex uint32
	scriptSig []byte
	sequence  uint32
}

type btcTxOut struct {
	value    int64
	pkScript []byte
}

type btcTx struct {
	version  int32
	inputs   []*btcTxIn
	outputs  []*btcTxOut
	lockTime uint32
}

// legacySigHash computes the legacy SIGHASH_ALL sighash for input `idx`,
// substituting that input's scriptSig with subScript and zeroing all others.
func (t *btcTx) legacySigHash(idx int, subScript []byte) ([]byte, error) {
	// Build a shallow copy with modified scriptSigs.
	ser, err := t.serializeForSig(idx, subScript)
	if err != nil {
		return nil, err
	}
	// Append sighash type (4 bytes LE, SIGHASH_ALL = 1).
	var ht [4]byte
	binary.LittleEndian.PutUint32(ht[:], 0x01)
	ser = append(ser, ht[:]...)
	h := sha256.Sum256(ser)
	h2 := sha256.Sum256(h[:])
	return h2[:], nil
}

func (t *btcTx) serializeForSig(idx int, subScript []byte) ([]byte, error) {
	var b bytes.Buffer
	if err := writeU32LE(&b, uint32(t.version)); err != nil {
		return nil, err
	}
	if err := writeVarInt(&b, uint64(len(t.inputs))); err != nil {
		return nil, err
	}
	for i, in := range t.inputs {
		if _, err := b.Write(in.prevHash); err != nil {
			return nil, err
		}
		if err := writeU32LE(&b, in.prevIndex); err != nil {
			return nil, err
		}
		// Only the input being signed carries subScript; others are empty.
		if i == idx {
			if err := writeVarBytes(&b, subScript); err != nil {
				return nil, err
			}
		} else {
			if err := writeVarBytes(&b, nil); err != nil {
				return nil, err
			}
		}
		if err := writeU32LE(&b, in.sequence); err != nil {
			return nil, err
		}
	}
	if err := writeVarInt(&b, uint64(len(t.outputs))); err != nil {
		return nil, err
	}
	for _, o := range t.outputs {
		if err := writeU64LE(&b, uint64(o.value)); err != nil {
			return nil, err
		}
		if err := writeVarBytes(&b, o.pkScript); err != nil {
			return nil, err
		}
	}
	if err := writeU32LE(&b, t.lockTime); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (t *btcTx) serialize() ([]byte, error) {
	var b bytes.Buffer
	if err := writeU32LE(&b, uint32(t.version)); err != nil {
		return nil, err
	}
	if err := writeVarInt(&b, uint64(len(t.inputs))); err != nil {
		return nil, err
	}
	for _, in := range t.inputs {
		if _, err := b.Write(in.prevHash); err != nil {
			return nil, err
		}
		if err := writeU32LE(&b, in.prevIndex); err != nil {
			return nil, err
		}
		if err := writeVarBytes(&b, in.scriptSig); err != nil {
			return nil, err
		}
		if err := writeU32LE(&b, in.sequence); err != nil {
			return nil, err
		}
	}
	if err := writeVarInt(&b, uint64(len(t.outputs))); err != nil {
		return nil, err
	}
	for _, o := range t.outputs {
		if err := writeU64LE(&b, uint64(o.value)); err != nil {
			return nil, err
		}
		if err := writeVarBytes(&b, o.pkScript); err != nil {
			return nil, err
		}
	}
	if err := writeU32LE(&b, t.lockTime); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func writeU32LE(b *bytes.Buffer, v uint32) error {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], v)
	_, err := b.Write(buf[:])
	return err
}

func writeU64LE(b *bytes.Buffer, v uint64) error {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], v)
	_, err := b.Write(buf[:])
	return err
}

func writeVarInt(b *bytes.Buffer, v uint64) error {
	switch {
	case v < 0xfd:
		return b.WriteByte(byte(v))
	case v <= 0xffff:
		if err := b.WriteByte(0xfd); err != nil {
			return err
		}
		var buf [2]byte
		binary.LittleEndian.PutUint16(buf[:], uint16(v))
		_, err := b.Write(buf[:])
		return err
	case v <= 0xffffffff:
		if err := b.WriteByte(0xfe); err != nil {
			return err
		}
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(v))
		_, err := b.Write(buf[:])
		return err
	default:
		if err := b.WriteByte(0xff); err != nil {
			return err
		}
		var buf [8]byte
		binary.LittleEndian.PutUint64(buf[:], v)
		_, err := b.Write(buf[:])
		return err
	}
}

func writeVarBytes(b *bytes.Buffer, data []byte) error {
	if err := writeVarInt(b, uint64(len(data))); err != nil {
		return err
	}
	_, err := b.Write(data)
	return err
}

// reverseHexHash reverses a big-endian hex txid into little-endian bytes.
func reverseHexHash(hexStr string) ([]byte, error) {
	if len(hexStr) != 64 {
		return nil, fmt.Errorf("txid must be 64 hex chars, got %d", len(hexStr))
	}
	b, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, err
	}
	rev := make([]byte, 32)
	for i := 0; i < 32; i++ {
		rev[i] = b[31-i]
	}
	return rev, nil
}

// hash160 = RIPEMD160(SHA256(b)).
func hash160(b []byte) []byte {
	sha := sha256.Sum256(b)
	r := ripemd160.New()
	r.Write(sha[:])
	return r.Sum(nil)
}

// p2pkhScript builds OP_DUP OP_HASH160 <20> <pkh> OP_EQUALVERIFY OP_CHECKSIG.
func p2pkhScript(pkh []byte) ([]byte, error) {
	if len(pkh) != 20 {
		return nil, errors.New("pkh must be 20 bytes")
	}
	var b bytes.Buffer
	b.WriteByte(0x76) // OP_DUP
	b.WriteByte(0xa9) // OP_HASH160
	b.WriteByte(0x14)
	b.Write(pkh)
	b.WriteByte(0x88) // OP_EQUALVERIFY
	b.WriteByte(0xac) // OP_CHECKSIG
	return b.Bytes(), nil
}

// p2pkhScriptSig builds PUSH(<sig>) PUSH(<pubkey>).
func p2pkhScriptSig(sig, pub []byte) ([]byte, error) {
	var b bytes.Buffer
	if len(sig) > 75 {
		return nil, errors.New("sig too long for direct push")
	}
	b.WriteByte(byte(len(sig)))
	b.Write(sig)
	b.WriteByte(byte(len(pub)))
	b.Write(pub)
	return b.Bytes(), nil
}

// decodeP2PKHAddress decodes a base58check P2PKH address to its 20-byte
// payload (version byte stripped). Validates the checksum.
func decodeP2PKHAddress(addr string) ([]byte, error) {
	decoded, err := base58Decode(addr)
	if err != nil {
		return nil, err
	}
	if len(decoded) != 25 {
		return nil, fmt.Errorf("p2pkh address must decode to 25 bytes, got %d", len(decoded))
	}
	if decoded[0] != 0x00 {
		return nil, errors.New("not a mainnet P2PKH address (version != 0x00)")
	}
	cksum := doubleSHA256(decoded[:21])
	if !bytes.Equal(cksum[:4], decoded[21:]) {
		return nil, errors.New("base58check checksum mismatch")
	}
	return decoded[1:21], nil
}

// ---------------------------------------------------------------------------
// Cosmos signing (SIGN_MODE_LEGACY_AMINO_JSON, secp256k1)
// ---------------------------------------------------------------------------

// CosmosSignDoc is the canonical Amino JSON payload for a Cosmos SDK tx.
type CosmosSignDoc struct {
	AccountNumber string                   `json:"account_number"`
	ChainID       string                   `json:"chain_id"`
	Fee           CosmosFee                 `json:"fee"`
	Memo          string                   `json:"memo"`
	Msgs          []map[string]interface{} `json:"msgs"`
	Sequence      string                   `json:"sequence"`
}

// CosmosFee is the Cosmos tx fee block.
type CosmosFee struct {
	Amount []CosmosCoin `json:"amount"`
	Gas    string       `json:"gas"`
}

// CosmosCoin is a Cosmos sdk.Coin (denom + amount).
type CosmosCoin struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

// CosmosSign canonicalizes the SignDoc to amino JSON, SHA-256 hashes it, and
// signs with the secp256k1 key from seed + path. Returns r||s (64 bytes) + the
// compressed pubkey.
func CosmosSign(seed []byte, derivationPath string, doc *CosmosSignDoc) (sig, pub []byte, err error) {
	if doc == nil {
		return nil, nil, errors.New("nil cosmos sign doc")
	}
	raw, err := json.Marshal(doc) // Go json.Marshal sorts struct keys alphabetically
	if err != nil {
		return nil, nil, fmt.Errorf("cosmos amino json: %w", err)
	}
	hash := sha256.Sum256(raw)
	privEcdsa, err := hdDerive(seed, derivationPath)
	if err != nil {
		return nil, nil, fmt.Errorf("cosmos key derivation: %w", err)
	}
	full, err := crypto.Sign(hash[:], privEcdsa)
	if err != nil {
		return nil, nil, fmt.Errorf("cosmos secp256k1 sign: %w", err)
	}
	sig = full[:64] // Cosmos uses r||s (no recovery byte)
	pub = crypto.CompressPubkey(&privEcdsa.PublicKey)
	return sig, pub, nil
}

// CosmosAddressFromSeed returns the bech32 account address for a seed + path +
// prefix (e.g. "cosmos", "osmo"). Uses real bech32 (BIP-173).
func CosmosAddressFromSeed(seed []byte, derivationPath, prefix string) (string, error) {
	privEcdsa, err := hdDerive(seed, derivationPath)
	if err != nil {
		return "", err
	}
	_, pub := btcec.PrivKeyFromBytes(crypto.FromECDSA(privEcdsa))
	pkh := hash160(pub.SerializeCompressed())
	return bech32Encode(prefix, pkh)
}

// ---------------------------------------------------------------------------
// base58 + bech32 (real implementations)
// ---------------------------------------------------------------------------

var b58Alphabet = []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")

// base58Encode encodes bytes to a base58 string.
func base58Encode(input []byte) string {
	x := new(big.Int).SetBytes(input)
	base := big.NewInt(58)
	mod := new(big.Int)
	var result []byte
	for x.Sign() > 0 {
		x.DivMod(x, base, mod)
		result = append([]byte{b58Alphabet[mod.Int64()]}, result...)
	}
	for _, b := range input {
		if b != 0 {
			break
		}
		result = append([]byte{'1'}, result...)
	}
	return string(result)
}

// base58Decode decodes a base58 string to bytes.
func base58Decode(s string) ([]byte, error) {
	dec := new(big.Int)
	base := big.NewInt(58)
	for _, c := range []byte(s) {
		idx := strings.IndexByte(string(b58Alphabet), c)
		if idx < 0 {
			return nil, fmt.Errorf("invalid base58 char %q", c)
		}
		dec.Mul(dec, base)
		dec.Add(dec, big.NewInt(int64(idx)))
	}
	// Count leading '1's as leading zero bytes.
	leadingZeros := 0
	for _, c := range []byte(s) {
		if c != '1' {
			break
		}
		leadingZeros++
	}
	out := dec.Bytes()
	if len(out) == 0 && dec.Sign() == 0 {
		out = []byte{}
	}
	// Prepend leading zero bytes.
	padded := make([]byte, leadingZeros+len(out))
	copy(padded[leadingZeros:], out)
	return padded, nil
}

// base58checkEncode appends a 4-byte double-SHA256 checksum.
func base58checkEncode(payload []byte) string {
	cksum := doubleSHA256(payload)
	encoded := append(payload, cksum[:4]...)
	return base58Encode(encoded)
}

func doubleSHA256(b []byte) []byte {
	h := sha256.Sum256(b)
	h2 := sha256.Sum256(h[:])
	return h2[:]
}

// bech32Encode encodes an hrp + 8-bit data to a bech32 string (BIP-173).
func bech32Encode(hrp string, data []byte) (string, error) {
	conv, err := convertBits(data, 8, 5, true)
	if err != nil {
		return "", err
	}
	combined := append(bech32HrpExpand(hrp), conv...)
	checksum := bech32CreateChecksum(combined)
	var sb strings.Builder
	sb.WriteString(hrp)
	sb.WriteByte('1')
	for _, p := range conv {
		sb.WriteByte(bech32Charset[p])
	}
	for _, p := range checksum {
		sb.WriteByte(bech32Charset[p])
	}
	return sb.String(), nil
}

const bech32Charset = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"

func bech32Polymod(values []byte) int {
	gen := []int{0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3}
	chk := 1
	for _, v := range values {
		top := chk >> 25
		chk = (chk&0x1ffffff)<<5 ^ int(v)
		for i := 0; i < 5; i++ {
			if (top>>i)&1 == 1 {
				chk ^= gen[i]
			}
		}
	}
	return chk
}

func bech32HrpExpand(hrp string) []byte {
	out := make([]byte, 0, len(hrp)*2+1)
	for i := 0; i < len(hrp); i++ {
		out = append(out, hrp[i]>>5)
	}
	out = append(out, 0)
	for i := 0; i < len(hrp); i++ {
		out = append(out, hrp[i]&31)
	}
	return out
}

func bech32CreateChecksum(values []byte) []byte {
	mod := append(bech32HrpExpand(""), values...)
	mod = append(mod, 0, 0, 0, 0, 0, 0)
	chk := bech32Polymod(mod) ^ 1
	out := make([]byte, 6)
	for i := 0; i < 6; i++ {
		out[i] = byte((chk >> (5 * (5 - i))) & 31)
	}
	return out
}

// convertBits regroups bits between byte widths (8->5 for bech32).
func convertBits(data []byte, fromBits, toBits uint, pad bool) ([]byte, error) {
	var acc int
	var bits uint
	out := make([]byte, 0, len(data)*int(fromBits)/int(toBits)+1)
	maxv := (1 << toBits) - 1
	for _, b := range data {
		acc = (acc << fromBits) | int(b)
		bits += fromBits
		for bits >= toBits {
			bits -= toBits
			out = append(out, byte((acc>>bits)&maxv))
		}
	}
	if pad {
		if bits > 0 {
			out = append(out, byte((acc<<(toBits-bits))&maxv))
		}
	} else if bits >= fromBits || ((acc<<(toBits-bits))&maxv) != 0 {
		return nil, errors.New("bech32: non-zero padding")
	}
	return out, nil
}
