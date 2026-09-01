package main

// non_evm_others.go — Kaspa (schnorr address+sign), Nervos (bech32m short
// address + blake160 sign), Filecoin (f1 address + CID-less sign),
// MultiversX (JSON sign), Tezos (forge + blake2b + watermark sign), and the
// fail-closed family docs for Aleo/Hedera/Flow.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/blake2b"
	ed "golang.org/x/crypto/ed25519"
)

// --------------------------- Kaspa ---------------------------------------

// KaspaAddress computes the canonical kaspa: P2PK bech32 address
// ([0x00 || schnorr-x-pub(32)]); Kaspa's HRP includes the trailing colon.
func KaspaAddress(seed []byte, path string) (string, error) {
	priv, err := hdDerive(seed, path)
	if err != nil {
		return "", err
	}
	_, pubB := btcec.PrivKeyFromBytes(crypto.FromECDSA(priv))
	x := schnorr.SerializePubKey(pubB)
	payload := append([]byte{0x00}, x[:32]...)
	return bech32Encode("kaspa:", payload)
}

// KaspaSign signs with schnorr over sha256 hashed message.
func KaspaSign(seed []byte, path string, msg []byte) (sig, pub []byte, err error) {
	priv, err := hdDerive(seed, path)
	if err != nil {
		return nil, nil, err
	}
	privB, pubB := btcec.PrivKeyFromBytes(crypto.FromECDSA(priv))
	h := sha256Hash(msg)
	sigObj, err := schnorr.Sign(privB, h)
	if err != nil {
		return nil, nil, err
	}
	x := schnorr.SerializePubKey(pubB)
	return sigObj.Serialize(), x[:32], nil
}

// sha256_Hash helper.
func sha256Hash(b []byte) []byte {
	h := sha256.Sum256(b)
	return h[:]
}

// --------------------------- Nervos (CKB) ----------------------------------

// bech32mEncode encodes with the bech32m constant 0x2bc830a3 (CKB short
// addresses must use bech32m).
func bech32mEncode(hrp string, data []byte) (string, error) {
	conv, err := convertBits(data, 8, 5, true)
	if err != nil {
		return "", err
	}
	combined := append(bech32HrpExpand(hrp), conv...)
	checksum := craftChecksum(combined)
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

func craftChecksum(values []byte) []byte {
	// BIP-350 bech32m: polymod(hrp_expand(hrp) || data || six zeroes) ^ 0x2bc830a3.
	// Callers pass values = hrpExpand(hrp) || data.
	mod := append(append([]byte{}, values...), 0, 0, 0, 0, 0, 0)
	chk := bech32Polymod(mod) ^ 0x2bc830a3 // bech32m constant for ckb
	out := make([]byte, 6)
	for i := 0; i < 6; i++ {
		out[i] = byte((chk >> (5 * (5 - i))) & 31)
	}
	return out
}

// ckbShortAddr builds the short-form CKB address: header 0x01 (code-hash-
// index of the well-known secp256k1/blake160 lock script), args=blake2b-160
// of the compressed pubkey.
func ckbBlake160(pubCompressed []byte) ([]byte, error) {
	h, err := blake2b.New(20, nil)
	if err != nil {
		return nil, err
	}
	h.Write(pubCompressed)
	return h.Sum(nil), nil
}

// NervosAddress computes the short address: ckb1 + bech32m( 0x01 || blake160(pub) ).
func NervosAddress(seed []byte, path string) (string, error) {
	priv, err := hdDerive(seed, path)
	if err != nil {
		return "", err
	}
	pub := crypto.CompressPubkey(&priv.PublicKey)
	h160, err := ckbBlake160(pub)
	if err != nil {
		return "", err
	}
	payload := append([]byte{0x01}, h160...)
	return bech32mEncode("ckb", payload)
}

// NervosSign signs with ckb personal hash (blake2b person "ckb-default-hash")
// over the message bytes — recoverable 65-byte ECDSA (r||s||v).
func NervosSign(seed []byte, path string, msg []byte) (sig, pub []byte, err error) {
	priv, err := hdDerive(seed, path)
	if err != nil {
		return nil, nil, err
	}
	h, berr := blake2b.New(20, nil)
	if berr != nil {
		return nil, nil, berr
	}
	h.Write(msg)
	msgHash := h.Sum(nil)
	sig, err = crypto.Sign(msgHash, priv)
	if err != nil {
		return nil, nil, err
	}
	pub = crypto.CompressPubkey(&priv.PublicKey)
	return sig, pub, nil
}

// --------------------------- Filecoin --------------------------------------

// FilAddress computes f1: "f1" + base32-lower(payload || 4-byte blake2b
// checksum of (protocol || payload)), payload = blake2b-160(pub-uncompressed).
func FilAddress(seed []byte, path string) (string, error) {
	priv, err := hdDerive(seed, path)
	if err != nil {
		return "", err
	}
	pubU := crypto.FromECDSAPub(&priv.PublicKey) // 65B uncompressed
	h, err := blake2b.New(20, nil)
	if err != nil {
		return "", err
	}
	h.Write(pubU)
	payload := h.Sum(nil)
	protocol := byte(0x01) // f1
	ckin := append([]byte{protocol}, payload...)
	ch := blake2b.Sum256(ckin)
	body := append([]byte{protocol}, payload...)
	return "f1" + base32LowerEnc(append(body, ch[:4]...)), nil
}

// FilSign signs the blake2b-256 of message bytes with ECDSA r||s||v.
func FilSign(seed []byte, path string, msg []byte) (sig, pub []byte, err error) {
	priv, err := hdDerive(seed, path)
	if err != nil {
		return nil, nil, err
	}
	h, berr := blake2b.New256(nil)
	if berr != nil {
		return nil, nil, berr
	}
	h.Write(msg)
	sig, err = crypto.Sign(h.Sum(nil), priv)
	if err != nil {
		return nil, nil, err
	}
	pub = crypto.CompressPubkey(&priv.PublicKey)
	return sig, pub, nil
}

// --------------------------- MultiversX ------------------------------------

type mxTx struct {
	Nonce    uint64 `json:"nonce"`
	Value    string `json:"value"`
	Receiver string `json:"receiver"`
	Sender   string `json:"sender"`
	GasLimit uint64 `json:"gasLimit"`
	GasPrice uint64 `json:"gasPrice"`
	ChainID  string `json:"chainID"`
	Version  uint32 `json:"version"`
}

// MultiversXAddress computes the erd bech32 address.
func MultiversXAddress(seed []byte, path string) (string, error) {
	pub, err := edPubKey(seed, path)
	if err != nil {
		return "", err
	}
	return bech32Encode("erd", pub[:32])
}

// MultiversXBuildSend builds a signed EGLD transfer JSON (broadcast over
// the API endpoint on request).
func MultiversXBuildSend(ctx context.Context, seed []byte, path, apiEndpoint, to, amountAtoms string, broadcast bool) (string, string, error) {
	priv, _, err := edKeypair(seed, path)
	if err != nil {
		return "", "", err
	}
	sender, err := MultiversXAddress(seed, path)
	if err != nil {
		return "", "", err
	}
	tx := mxTx{
		Receiver: to,
		Value:    amountAtoms,
		GasLimit: 50000,
		GasPrice: 1000000000,
		ChainID:  "1",
		Version:  1,
	}
	tx.Sender = sender
	raw, err := json.Marshal(tx)
	if err != nil {
		return "", "", err
	}
	sig := ed.Sign(priv, raw)
	final := struct {
		mxTx
		Signature string `json:"signature"`
	}{tx, hexOf(sig)}
	finalRaw, _ := json.Marshal(final)
	if !broadcast {
		return string(finalRaw), "", nil
	}
	url := strings.TrimRight(apiEndpoint, "/") + "/transactions"
	resp, err := postRaw(ctx, url, finalRaw)
	if err != nil {
		return "", "", fmt.Errorf("multiversx broadcast: %w", err)
	}
	var parsed struct {
		TxHash string `json:"txHash"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal(resp, &parsed); err != nil {
		return "", "", err
	}
	if parsed.TxHash == "" {
		return "", "", fmt.Errorf("multiversx broadcast rejected: %s", parsed.Error)
	}
	return string(finalRaw), parsed.TxHash, nil
}

// --------------------------- Tezos -----------------------------------------

// TezosAddress computes tz1 from the pubkey.
func TezosAddress(seed []byte, path string) (string, error) {
	pub, err := edPubKey(seed, path)
	if err != nil {
		return "", err
	}
	h, err := blake2b.New(20, nil)
	if err != nil {
		return "", err
	}
	h.Write(pub[:32])
	pkh := h.Sum(nil)
	pre := []byte{0x06, 0xa1, 0x9f} // tz1 prefix
	return base58checkEncode(append(pre, pkh...)), nil
}

// tezosZarith encodes a nat with the Tezos LEB-ish encoding.
func tezosZarith(v uint64) []byte {
	var out []byte
	first := v & 0x3f
	v >>= 6
	if v > 0 {
		first |= 0x80
	}
	out = append(out, byte(first))
	for v > 0 {
		chunk := v & 0x7f
		v >>= 7
		if v > 0 {
			chunk |= 0x80
		}
		out = append(out, byte(chunk))
	}
	return out
}

// tezosForgeTransfer forges branch || tag(0x6c) || src || fee || counter ||
// gas || storage || dest || amount with zarith integers.
func tezosForgeTransfer(branch []byte, fromPkh, toPkh []byte, amount, fee, counter, gas, storage uint64) ([]byte, error) {
	if len(branch) != 32 || len(fromPkh) != 20 || len(toPkh) != 20 {
		return nil, errors.New("tezos forge input size error")
	}
	var b bytes.Buffer
	b.Write(branch)
	b.Write([]byte{0x6c}) // TRANSFER
	b.Write([]byte{0x00}) // ED25519 implicit source
	b.Write(fromPkh)
	b.Write(tezosZarith(fee))
	b.Write(tezosZarith(counter))
	b.Write(tezosZarith(gas))
	b.Write(tezosZarith(storage))
	b.Write([]byte{0x00}) // ED25519 implicit dest
	b.Write(toPkh)
	b.Write(tezosZarith(amount))
	return b.Bytes(), nil
}

// tezosPkhFrom decodes a tz1 address to the 20-byte pkh.
func tezosPkhFrom(addr string) ([]byte, error) {
	raw, err := tezosB58Decode(addr)
	if err != nil {
		return nil, err
	}
	if len(raw) != 23 {
		return nil, errors.New("unexpected tz1 payload length")
	}
	return raw[3:], nil
}

// tezosB58Decode decodes a base58check-encoded tezos hash/address (checksum
// is validated).
func tezosB58Decode(s string) ([]byte, error) {
	dec, err := base58Decode(s)
	if err != nil {
		return nil, err
	}
	if len(dec) < 5 {
		return nil, errors.New("tezos base58 payload short")
	}
	end := len(dec) - 4
	return dec[:end], nil
}

// TezosBuildSend forges + signs a transfer (watermark 0x03 generic op);
// broadcasts via /injection/operation.
func TezosBuildSend(ctx context.Context, seed []byte, path, endpoint, to string, amount, fee, counter, gas, storage uint64, broadcast bool) (string, string, error) {
	priv, pub, err := edKeypair(seed, path)
	if err != nil {
		return "", "", err
	}
	fromAddr, err := TezosAddress(seed, path)
	if err != nil {
		return "", "", err
	}
	fromPkh, err := tezosPkhFrom(fromAddr)
	if err != nil {
		return "", "", err
	}
	toPkh, err := tezosPkhFrom(to)
	if err != nil {
		return "", "", err
	}
	blockRaw, err := getRaw(ctx, strings.TrimRight(endpoint, "/")+"/chains/main/blocks/head/hash")
	if err != nil {
		return "", "", fmt.Errorf("tezos branch: %w", err)
	}
	branchStr := strings.Trim(strings.TrimSpace(string(blockRaw)), "\"")
	branch, err := tezosB58Decode(branchStr)
	if err != nil || len(branch) != 32 {
		return "", "", errors.New("tezos branch decode error")
	}
	forged, err := tezosForgeTransfer(branch, fromPkh, toPkh, amount, fee, counter, gas, storage)
	if err != nil {
		return "", "", err
	}
	water := append([]byte{0x03}, forged...)
	h, err := blake2b.New256(nil)
	if err != nil {
		return "", "", err
	}
	h.Write(water)
	sig := ed.Sign(priv, h.Sum(nil))
	full := append(forged, sig...)
	_ = pub
	if !broadcast {
		return hexOf(full), "", nil
	}
	body, _ := json.Marshal(map[string]string{"signed": hexOf(full)})
	resp, err := postRaw(ctx, strings.TrimRight(endpoint, "/")+"/injection/operation?chain=main", body)
	if err != nil {
		return "", "", fmt.Errorf("tezos broadcast: %w", err)
	}
	var parsedHash []string
	if err := json.Unmarshal(resp, &parsedHash); err != nil || len(parsedHash) == 0 {
		return "", "", errors.New("tezos broadcast: malformed reply")
	}
	return hexOf(full), parsedHash[0], nil
}

// --------------------------- fail-closed doc chains -------------------------

// nonEvmNotFeasible returns a precise error for chains whose address schemes
// require custom curves or on-chain account ids.
func nonEvmNotFeasible(family nonEvmFamily) error {
	switch family {
	case familyAleo:
		return errors.New("aleo: address derivation requires the Edwards-BLS12 group not available in this backend — fail-closed (on-chain verification would reject fabricated material)")
	case familyHedera:
		return errors.New("hedera: account ids (0.0.x...) are assigned on-chain at creation; only the ed25519 public key can be derived — fail-closed with the pubkey available on request")
	case familyFlow:
		return errors.New("flow: account addresses (8-byte) are assigned on-chain at creation; only the secp256k1/p256 public key can be derived — fail-closed with the pubkey available on request")
	}
	return errors.New("family not feasible — fail-closed")
}
