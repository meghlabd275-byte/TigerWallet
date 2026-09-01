package main

// non_evm_utxo.go — generalized UTXO-family SDK for the 17 seeded UTXO
// chains. Legacy chains (BTC/LTC/DOGE/DASH/RVN/ZEC-legacy/GRS/DGB/QTUM/XVG/
// NMC/MONA/BLK/KMD) use the standard double-SHA256 SIGHASH_ALL
// self-signed here. BCH-descendants (BCH/BSV/XEC) use BIP-143
// SIGHASH_ALL|FORKID. Zcash supports SAPLING v4 with ZIP-243
// (BLAKE2b-256, person "ZcashSigHash"+branchID); legacy ZEC formats are
// consensus-rejected since Overwinter, so v3/v4 only.
//
// Signing derives the secp256k1 key via the shared BIP-44 path from the
// wallet seed; nothing is fabricated: if required input amounts for forkID
// chains are absent, the call fails closed.

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	dblake2b "github.com/dchest/blake2b"
	"github.com/ethereum/go-ethereum/crypto"
)

// UTXOInput describes one input; ForkID chains additionally require Satoshis.
type UTXOInput struct {
	TxID         string `json:"txid"`
	Vout         uint32 `json:"vout"`
	ScriptPubKey []byte `json:"script_pubkey"`
	Satoshis     int64  `json:"satoshis"` // required for forkid (BCH/BSV/XEC)
}

// UTXOOutput is one tx output.
type UTXOOutput struct {
	Address   string `json:"address"`
	AmountSat int64  `json:"amount_sat"`
}

// UTXOAddressFromSeed derives the P2PKH address for a UTXO chain.
func UTXOAddressFromSeed(seed []byte, derivationPath, chainType string) (string, error) {
	params, ok := utxoTable[chainType]
	if !ok {
		return "", fmt.Errorf("unsupported UTXO chain %q", chainType)
	}
	priv, err := hdDerive(seed, derivationPath)
	if err != nil {
		return "", err
	}
	_, pub := btcec.PrivKeyFromBytes(crypto.FromECDSA(priv))
	pkh := hash160(pub.SerializeCompressed())
	return base58checkEncode(append(params.p2pkhVersion, pkh...)), nil
}

// UTXOSign builds a signed UTXO transaction for the chain; returns raw hex.
func UTXOSign(seed []byte, derivationPath, chainType string, inputs []UTXOInput, outputs []UTXOOutput) (string, error) {
	params, ok := utxoTable[chainType]
	if !ok {
		return "", fmt.Errorf("unsupported UTXO chain %q", chainType)
	}
	privEcdsa, err := hdDerive(seed, derivationPath)
	if err != nil {
		return "", fmt.Errorf("key derivation: %w", err)
	}
	priv, pubKey := btcec.PrivKeyFromBytes(crypto.FromECDSA(privEcdsa))

	var txOuts []*btcTxOut
	for _, o := range outputs {
		pkh, err := decodeUTXOAddress(o.Address, params.p2pkhVersion)
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
		tx.inputs = append(tx.inputs, &btcTxIn{prevHash: hash, prevIndex: in.Vout, sequence: 0xffffffff})
	}
	for _, o := range txOuts {
		tx.outputs = append(tx.outputs, o)
	}

	if params.forkID {
		if err := signForkIDInputs(tx, inputs, priv, pubKey); err != nil {
			return "", err
		}
	} else {
		if err := signLegacyInputs(tx, inputs, priv, pubKey); err != nil {
			return "", err
		}
	}
	raw, err := tx.serialize()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

// signLegacyInputs signs standard double-SHA256 SIGHASH_ALL inputs.
func signLegacyInputs(tx *btcTx, inputs []UTXOInput, priv *btcec.PrivateKey, pubKey *btcec.PublicKey) error {
	const sighashAll = 0x01
	for i, in := range inputs {
		sigHash, err := tx.legacySigHash(i, in.ScriptPubKey)
		if err != nil {
			return fmt.Errorf("sighash input %d: %w", i, err)
		}
		sig := ecdsa.Sign(priv, sigHash)
		der := sig.Serialize()
		scriptSig, err := p2pkhScriptSig(append(der, byte(sighashAll)), pubKey.SerializeCompressed())
		if err != nil {
			return err
		}
		tx.inputs[i].scriptSig = scriptSig
	}
	return nil
}

// signForkIDInputs signs BIP-143 SIGHASH_ALL|FORKID inputs (BCH/BSV/XEC).
func signForkIDInputs(tx *btcTx, inputs []UTXOInput, priv *btcec.PrivateKey, pubKey *btcec.PublicKey) error {
	const sighashAllForkID = 0x41 // SIGHASH_ALL | SIGHASH_FORKID
	for i, in := range inputs {
		if in.Satoshis <= 0 {
			return fmt.Errorf("forkid signing requires satoshis for every input (input %d missing)", i)
		}
		preimage := bip143Preimage(tx, i, in.ScriptPubKey, in.Satoshis)
		h := sha256.Sum256(preimage)
		h2 := sha256.Sum256(h[:])
		sig := ecdsa.Sign(priv, h2[:])
		der := sig.Serialize()
		scriptSig, err := p2pkhScriptSig(append(der, byte(sighashAllForkID)), pubKey.SerializeCompressed())
		if err != nil {
			return err
		}
		tx.inputs[i].scriptSig = scriptSig
	}
	return nil
}

// bip143Preimage builds the BIP-143 sighash preimage for a P2PKH input.
func bip143Preimage(tx *btcTx, idx int, subScript []byte, satoshis int64) []byte {
	var hashPrevouts, hashSequence, hashOutputs []byte
	{
		var b bytes.Buffer
		for _, in := range tx.inputs {
			b.Write(in.prevHash)
			var idxBuf [4]byte
			binary.LittleEndian.PutUint32(idxBuf[:], in.prevIndex)
			b.Write(idxBuf[:])
		}
		h := sha256.Sum256(b.Bytes())
		h2 := sha256.Sum256(h[:])
		hashPrevouts = append(hashPrevouts, h2[:]...)
	}
	{
		var b bytes.Buffer
		for _, in := range tx.inputs {
			var seqBuf [4]byte
			binary.LittleEndian.PutUint32(seqBuf[:], in.sequence)
			b.Write(seqBuf[:])
		}
		h := sha256.Sum256(b.Bytes())
		h2 := sha256.Sum256(h[:])
		hashSequence = append(hashSequence, h2[:]...)
	}
	{
		var b bytes.Buffer
		for _, o := range tx.outputs {
			var valBuf [8]byte
			binary.LittleEndian.PutUint64(valBuf[:], uint64(o.value))
			b.Write(valBuf[:])
			writeVarBytes(&b, o.pkScript)
		}
		h := sha256.Sum256(b.Bytes())
		h2 := sha256.Sum256(h[:])
		hashOutputs = append(hashOutputs, h2[:]...)
	}

	in := tx.inputs[idx]
	var b bytes.Buffer
	writeU32LE(&b, uint32(tx.version))
	b.Write(hashPrevouts)
	b.Write(hashSequence)
	b.Write(in.prevHash)
	writeU32LE(&b, in.prevIndex)
	writeVarBytes(&b, subScript)
	writeU64LE(&b, uint64(satoshis))
	writeU32LE(&b, in.sequence)
	b.Write(hashOutputs)
	writeU32LE(&b, tx.lockTime)
	writeU32LE(&b, 0x41) // SIGHASH_ALL|FORKID
	return b.Bytes()
}

// ---------------------------------------------------------------
// Zcash Sapling v4 (ZIP-243) — reachable, transparent-only.
// ---------------------------------------------------------------

const zcashSaplingBranchID = 0x76b80bad // consensus branchId for Sapling

// ZECSign builds a Zcash v4 (Sapling, transparent-only) signed tx using
// ZIP-243 sighash (BLAKE2b-256, person "ZcashSigHash").
// ZECAddressFromSeed derives the mainnet transparent P2PKH Zcash address
// (version bytes 0x1c 0xb8 — "t1").
func ZECAddressFromSeed(seed []byte, derivationPath string) (string, error) {
	priv, err := hdDerive(seed, derivationPath)
	if err != nil {
		return "", err
	}
	_, pub := btcec.PrivKeyFromBytes(crypto.FromECDSA(priv))
	pkh := hash160(pub.SerializeCompressed())
	return base58checkEncode(append([]byte{0x1c, 0xb8}, pkh...)), nil
}

func ZECSign(seed []byte, derivationPath string, inputs []UTXOInput, outputs []UTXOOutput) (string, error) {
	privEcdsa, err := hdDerive(seed, derivationPath)
	if err != nil {
		return "", fmt.Errorf("zcash key derivation: %w", err)
	}
	priv, pubKey := btcec.PrivKeyFromBytes(crypto.FromECDSA(privEcdsa))

	tx := &btcTx{version: 4, lockTime: 0} // v4 header: 0x04000080|versionGroupId handled in ser
	for _, in := range inputs {
		hash, err := reverseHexHash(in.TxID)
		if err != nil {
			return "", fmt.Errorf("invalid input txid %q: %w", in.TxID, err)
		}
		tx.inputs = append(tx.inputs, &btcTxIn{prevHash: hash, prevIndex: in.Vout, sequence: 0xffffffff})
	}
	for _, o := range outputs {
		pkh, err := decodeUTXOAddress(o.Address, utxoTable["zcash"].p2pkhVersion)
		if err != nil {
			return "", fmt.Errorf("invalid zcash address %q: %w", o.Address, err)
		}
		script, err := p2pkhScript(pkh)
		if err != nil {
			return "", err
		}
		tx.outputs = append(tx.outputs, &btcTxOut{value: o.AmountSat, pkScript: script})
	}

	const sighashAll = 0x01
	for i, in := range inputs {
		if in.Satoshis <= 0 {
			return "", fmt.Errorf("zcash zip-243 sighash requires satoshis per input (input %d missing)", i)
		}
		preimage := zip243Preimage(tx, i, in.ScriptPubKey, in.Satoshis)
		sighash, err := blake2b256Personal(preimage, blakePerson(zcashSaplingBranchID))
		if err != nil {
			return "", err
		}
		sig := ecdsa.Sign(priv, sighash)
		der := sig.Serialize()
		scriptSig, err := p2pkhScriptSig(append(der, byte(sighashAll)), pubKey.SerializeCompressed())
		if err != nil {
			return "", err
		}
		tx.inputs[i].scriptSig = scriptSig
	}

	// ZEC v4 serialization: header (0x04000080), versionGroupId 0x892FC208,
	// transparent bundle, empty sapling bundle, no joinsplit.
	raw, err := serializeZECv4(tx)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

// zip243Preimage builds the ZIP-243 sighash preimage for one transparent
// input of a v4 Zcash transaction.
func zip243Preimage(tx *btcTx, idx int, subScript []byte, satoshis int64) []byte {
	var hashPrevouts, hashSequence, hashOutputs []byte
	hf := func(data []byte) []byte {
		h, _ := blake2b256Personal(data, blakePerson(zcashSaplingBranchID))
		return h
	}
	{
		var b bytes.Buffer
		for _, in := range tx.inputs {
			b.Write(in.prevHash)
			writeU32LE(&b, in.prevIndex)
		}
		hashPrevouts = hf(b.Bytes())
	}
	{
		var b bytes.Buffer
		for _, in := range tx.inputs {
			writeU32LE(&b, in.sequence)
		}
		hashSequence = hf(b.Bytes())
	}
	{
		var b bytes.Buffer
		for _, o := range tx.outputs {
			writeU64LE(&b, uint64(o.value))
			writeVarBytes(&b, o.pkScript)
		}
		hashOutputs = hf(b.Bytes())
	}

	in := tx.inputs[idx]
	var b bytes.Buffer
	writeU32LE(&b, 0x04000080) // v4 header
	b.Write([]byte{0x70, 0x85, 0xC8, 0x92}) // versionGroupId 0x892FC208 LE
	b.Write(hashPrevouts)
	b.Write(hashSequence)
	b.Write(hashOutputs)
	b.Write(in.prevHash)
	writeU32LE(&b, in.prevIndex)
	writeVarBytes(&b, subScript)
	writeU64LE(&b, uint64(satoshis))
	writeU32LE(&b, in.sequence)
	writeU32LE(&b, tx.lockTime)
	writeU32LE(&b, 0x01) // SIGHASH_ALL
	return b.Bytes()
}

// blakePerson normalizes the BLAKE2b person string for ZIP-243:
// "ZcashSigHash" (12B) || branchId (4B LE).
func blakePerson(branchID uint32) []byte {
	p := make([]byte, 16)
	copy(p, []byte("ZcashSigHash"))
	binary.LittleEndian.PutUint32(p[12:], branchID)
	return p
}

// blake2b256Personal computes BLAKE2b-256 with a 16-byte person string.
func blake2b256Personal(data, person []byte) ([]byte, error) {
	if len(person) != 16 {
		return nil, errors.New("blake2b person must be 16 bytes")
	}
	h, err := dblake2b.New(&dblake2b.Config{Size: 32, Person: person})
	if err != nil {
		return nil, err
	}
	h.Write(data)
	return h.Sum(nil), nil
}

// serializeZECv4 serializes a Zcash Sapling v4 tx (transparent bundle only).
func serializeZECv4(tx *btcTx) ([]byte, error) {
	var b bytes.Buffer
	if err := writeU32LE(&b, 0x04000080); err != nil { // header (v4 | overwinter)
		return nil, err
	}
	b.Write([]byte{0x70, 0x85, 0xC8, 0x92}) // versionGroupId 0x892FC208 LE
	if err := writeVarInt(&b, uint64(len(tx.inputs))); err != nil {
		return nil, err
	}
	for _, in := range tx.inputs {
		b.Write(in.prevHash)
		writeU32LE(&b, in.prevIndex)
		writeVarBytes(&b, in.scriptSig)
		writeU32LE(&b, in.sequence)
	}
	if err := writeVarInt(&b, uint64(len(tx.outputs))); err != nil {
		return nil, err
	}
	for _, o := range tx.outputs {
		writeU64LE(&b, uint64(o.value))
		writeVarBytes(&b, o.pkScript)
	}
	if err := writeU32LE(&b, tx.lockTime); err != nil {
		return nil, err
	}
	// expiryHeight 0, valueBalance=0, 0 spends, 0 outputs (no sapling ops),
	// then no joinsplits (compact size 0).
	writeU32LE(&b, 0)
	writeU64LE(&b, 0)
	b.Write([]byte{0x00, 0x00}) // nShieldedSpend, nShieldedOutput
	return b.Bytes(), nil
}

// decodeUTXOAddress decodes a base58check P2PKH address with a specific
// (potentially two-byte) version.
func decodeUTXOAddress(addr string, version []byte) ([]byte, error) {
	decoded, err := base58Decode(addr)
	if err != nil {
		return nil, err
	}
	want := len(version) + 20 + 4
	if len(decoded) != want {
		return nil, fmt.Errorf("address must decode to %d bytes, got %d", want, len(decoded))
	}
	for i, v := range version {
		if decoded[i] != v {
			return nil, fmt.Errorf("address version mismatch (got 0x%02x)", decoded[i])
		}
	}
	end := len(decoded) - 4
	ck := doubleSHA256(decoded[:end])
	if !bytes.Equal(ck[:4], decoded[end:]) {
		return nil, errors.New("base58check checksum mismatch")
	}
	return decoded[len(version) : len(version)+20], nil
}
