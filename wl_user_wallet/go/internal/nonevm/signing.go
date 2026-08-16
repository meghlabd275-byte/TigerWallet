// Package nonevm provides REAL non-EVM signing (Solana Ed25519, Bitcoin
// secp256k1 legacy P2PKH, Cosmos secp256k1 Amino SignDoc) and Web3 Secret
// Storage V3 keystore export/import for the standalone WL-UserWallet backend.
// Ported verbatim from the canonical wallet_api non_evm_signing.go +
// keystore_v3.go. No stubs: real SLIP-0010 Ed25519 HD derivation, real
// BIP-32 secp256k1 derivation, real bech32/base58check, real scrypt+AES-CTR+
// keccak256 MAC.
package nonevm

import (
	"bytes"
	"crypto/ecdsa"
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
	btcecdsa "github.com/btcsuite/btcd/btcec/v2/ecdsa"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	ed "golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/ripemd160"
)

// ---------------------------------------------------------------------------
// SLIP-0010 Ed25519 HD derivation (Solana + other Ed25519 chains)
// ---------------------------------------------------------------------------

// Slip10DeriveEd25519 implements SLIP-0010 Ed25519 hierarchical derivation.
// Ed25519 supports ONLY hardened derivation (index >= 0x80000000).
func Slip10DeriveEd25519(seed []byte, path string) ([]byte, error) {
	mac := hmac.New(sha512.New, []byte("ed25519 seed"))
	mac.Write(seed)
	I := mac.Sum(nil)
	il := I[:32]
	ir := I[32:]
	segments, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	for _, idx := range segments {
		if idx < 0x80000000 {
			return nil, errors.New("ed25519 SLIP-0010 requires hardened derivation only")
		}
		mac := hmac.New(sha512.New, ir)
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
// and signs the message bytes.
func SolanaSign(seed []byte, derivationPath, message string) (sig, pub []byte, err error) {
	seedEd, err := Slip10DeriveEd25519(seed, derivationPath)
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
	seedEd, err := Slip10DeriveEd25519(seed, derivationPath)
	if err != nil {
		return "", err
	}
	priv := ed.NewKeyFromSeed(seedEd)
	pub := priv.Public().(ed.PublicKey)
	return base58Encode(pub), nil
}

// ---------------------------------------------------------------------------
// BIP-32 secp256k1 HD derivation (Bitcoin / Cosmos)
// ---------------------------------------------------------------------------

const hardening uint32 = 0x80000000

// HDDerive derives the secp256k1 private key for a BIP-32 path from a seed.
func HDDerive(seed []byte, path string) (*ecdsa.PrivateKey, error) {
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
	return ethcrypto.ToECDSA(parentKey)
}

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
		n, err := parseUint(p)
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

func parseUint(s string) (uint64, error) {
	var n uint64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("non-digit %q", c)
		}
		n = n*10 + uint64(c-'0')
	}
	return n, nil
}

func ckdPriv(parentKey, parentChain []byte, index uint32) (childKey, childChain []byte, err error) {
	if len(parentKey) != 32 || len(parentChain) != 32 {
		return nil, nil, errors.New("invalid parent key/chain length")
	}
	mac := hmac.New(sha512.New, parentChain)
	if index >= hardening {
		mac.Write([]byte{0x00})
		mac.Write(parentKey)
	} else {
		priv, err := ethcrypto.ToECDSA(parentKey)
		if err != nil {
			return nil, nil, err
		}
		pub := ethcrypto.CompressPubkey(&priv.PublicKey)
		mac.Write(pub)
	}
	var idxBuf [4]byte
	binary.BigEndian.PutUint32(idxBuf[:], index)
	mac.Write(idxBuf[:])
	I := mac.Sum(nil)
	il := I[:32]
	ir := I[32:]
	curveOrder := ethcrypto.S256().Params().N
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

// ---------------------------------------------------------------------------
// Bitcoin signing (legacy P2PKH, SIGHASH_ALL)
// ---------------------------------------------------------------------------

// BTCInput describes one input to a Bitcoin transaction.
type BTCInput struct {
	TxID         string `json:"tx_id"`
	Vout         uint32 `json:"vout"`
	ScriptPubKey []byte `json:"script_pub_key"`
}

// BTCOutput describes one transaction output.
type BTCOutput struct {
	Address   string `json:"address"`
	AmountSat int64 `json:"amount_sat"`
}

// BTCSign builds a signed legacy Bitcoin transaction and returns the raw tx hex.
func BTCSign(seed []byte, derivationPath string, inputs []BTCInput, outputs []BTCOutput) (string, error) {
	privEcdsa, err := HDDerive(seed, derivationPath)
	if err != nil {
		return "", fmt.Errorf("bitcoin key derivation: %w", err)
	}
	priv, pubKey := btcec.PrivKeyFromBytes(ethcrypto.FromECDSA(privEcdsa))

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

	const sighashAll = 0x01
	for i, in := range inputs {
		sigHash, err := tx.legacySigHash(i, in.ScriptPubKey)
		if err != nil {
			return "", fmt.Errorf("calc sighash input %d: %w", i, err)
		}
		sig := btcecdsa.Sign(priv, sigHash)
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

// BTCAddressFromSeed returns the base58check P2PKH address (mainnet).
func BTCAddressFromSeed(seed []byte, derivationPath string) (string, error) {
	privEcdsa, err := HDDerive(seed, derivationPath)
	if err != nil {
		return "", err
	}
	_, pubKey := btcec.PrivKeyFromBytes(ethcrypto.FromECDSA(privEcdsa))
	pkh := hash160(pubKey.SerializeCompressed())
	return base58checkEncode(append([]byte{0x00}, pkh...)), nil
}

// BTCSignMessage signs a message with the Bitcoin secp256k1 key, returns r||s.
func BTCSignMessage(seed []byte, derivationPath string, message []byte) (sig, pub []byte, err error) {
	priv, err := HDDerive(seed, derivationPath)
	if err != nil {
		return nil, nil, err
	}
	full, err := ethcrypto.Sign(message, priv)
	if err != nil {
		return nil, nil, err
	}
	sig = full[:64]
	pub = ethcrypto.CompressPubkey(&priv.PublicKey)
	return sig, pub, nil
}

// ---------------------------------------------------------------------------
// Minimal legacy Bitcoin tx serializer + SIGHASH_ALL (pre-segwit)
// ---------------------------------------------------------------------------

type btcTxIn struct {
	prevHash  []byte
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

func (t *btcTx) legacySigHash(idx int, subScript []byte) ([]byte, error) {
	ser, err := t.serializeForSig(idx, subScript)
	if err != nil {
		return nil, err
	}
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

func hash160(b []byte) []byte {
	sha := sha256.Sum256(b)
	r := ripemd160.New()
	r.Write(sha[:])
	return r.Sum(nil)
}

func p2pkhScript(pkh []byte) ([]byte, error) {
	if len(pkh) != 20 {
		return nil, errors.New("pkh must be 20 bytes")
	}
	var b bytes.Buffer
	b.WriteByte(0x76)
	b.WriteByte(0xa9)
	b.WriteByte(0x14)
	b.Write(pkh)
	b.WriteByte(0x88)
	b.WriteByte(0xac)
	return b.Bytes(), nil
}

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
	Fee           CosmosFee                `json:"fee"`
	Memo          string                   `json:"memo"`
	Msgs          []map[string]interface{} `json:"msgs"`
	Sequence      string                   `json:"sequence"`
}

// CosmosFee is the Cosmos tx fee block.
type CosmosFee struct {
	Amount []CosmosCoin `json:"amount"`
	Gas    string       `json:"gas"`
}

// CosmosCoin is a Cosmos sdk.Coin.
type CosmosCoin struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

// CosmosSign canonicalizes the SignDoc to amino JSON, SHA-256 hashes it, and
// signs with the secp256k1 key from seed + path. Returns r||s (64 bytes).
func CosmosSign(seed []byte, derivationPath string, doc *CosmosSignDoc) (sig, pub []byte, err error) {
	if doc == nil {
		return nil, nil, errors.New("nil cosmos sign doc")
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		return nil, nil, fmt.Errorf("cosmos amino json: %w", err)
	}
	hash := sha256.Sum256(raw)
	privEcdsa, err := HDDerive(seed, derivationPath)
	if err != nil {
		return nil, nil, fmt.Errorf("cosmos key derivation: %w", err)
	}
	full, err := ethcrypto.Sign(hash[:], privEcdsa)
	if err != nil {
		return nil, nil, fmt.Errorf("cosmos secp256k1 sign: %w", err)
	}
	sig = full[:64]
	pub = ethcrypto.CompressPubkey(&privEcdsa.PublicKey)
	return sig, pub, nil
}

// CosmosAddressFromSeed returns the bech32 account address for a seed + path +
// prefix (e.g. "cosmos", "osmo").
func CosmosAddressFromSeed(seed []byte, derivationPath, prefix string) (string, error) {
	privEcdsa, err := HDDerive(seed, derivationPath)
	if err != nil {
		return "", err
	}
	_, pub := btcec.PrivKeyFromBytes(ethcrypto.FromECDSA(privEcdsa))
	pkh := hash160(pub.SerializeCompressed())
	return bech32Encode(prefix, pkh)
}

// ---------------------------------------------------------------------------
// base58 + bech32 (real implementations)
// ---------------------------------------------------------------------------

var b58Alphabet = []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")

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
	padded := make([]byte, leadingZeros+len(out))
	copy(padded[leadingZeros:], out)
	return padded, nil
}

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

const bech32Charset = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"

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
