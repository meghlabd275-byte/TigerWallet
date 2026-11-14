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
	"github.com/tigerwallet/chainsdk"
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
	return mwUTXOAddressFromSeed(seed, derivationPath, []byte{0x00})
}

// mwUTXOAddressFromSeed derives a legacy P2PKH address for any UTXO-family
// chain (the version byte is the only per-chain difference).
func mwUTXOAddressFromSeed(seed []byte, derivationPath string, version []byte) (string, error) {
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
	// Version prefixes are 1-2 bytes (Zcash is 0x1cb8); prepend the whole
	// prefix before the base58check payload.
	return base58CheckEncode(append(append([]byte{}, version...), hash160...)), nil
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

// mwCosmosSign signs a Cosmos SignDoc with secp256k1 (SIGN_MODE_LEGACY_AMINO_JSON).
// The signDoc is the canonical amino JSON. Returns a 64-byte secp256k1 signature
// (r||s, no recovery byte) over SHA-256(signDoc). Real crypto — not a hash.
func mwCosmosSign(seed []byte, derivationPath, signDoc string) ([]byte, []byte, error) {
	privKey, err := DerivePrivateKeyFromPath(seed, derivationPath)
	if err != nil {
		return nil, nil, fmt.Errorf("cosmos key derivation: %w", err)
	}
	// SIGN_MODE_LEGACY_AMINO_JSON: sign over SHA-256(canonical amino JSON).
	msgHash := sha256.Sum256([]byte(signDoc))
	sig, err := crypto.Sign(msgHash[:], privKey)
	if err != nil {
		return nil, nil, fmt.Errorf("cosmos sign: %w", err)
	}
	// crypto.Sign returns 65 bytes (r||s||v); Cosmos uses 64 (r||s, no recovery).
	pubKeyBytes := crypto.CompressPubkey(&privKey.PublicKey)
	return sig[:64], pubKeyBytes, nil
}

// mwBTCSignTx builds and signs a real Bitcoin P2PKH transaction using SIGHASH_ALL.
// It fetches UTXOs from the blockstream.info API (public, no auth), constructs
// a 1-input 2-output (transfer + change) legacy transaction, signs with the
// derived secp256k1 key, and returns the raw signed transaction hex ready for
// broadcast. If insufficient UTXOs, returns an error (no fake tx).
func mwBTCSignTx(seed []byte, derivationPath, toAddress, valueStr string) (string, string, error) {
	p, _ := utxoParamsFor("bitcoin", 0)
	return mwUTXOSignTx(seed, derivationPath, toAddress, valueStr, p)
}

// mwUTXOSignTx builds and signs a real transfer on any UTXO-family chain.
//
// The sighash implementation lives in the shared chainsdk module so the
// MasterWallet, go/wallet_api and the SDKs cannot drift: chainsdk.UTXOSign
// covers the standard double-SHA256 SIGHASH_ALL family and the BCH-descendant
// BIP-143 SIGHASH_ALL|FORKID family, and chainsdk.ZECSign covers Zcash's
// Sapling v4 / ZIP-243 sighash. This function only selects the signer, feeds
// it the real UTXOs from the chain's indexer, and returns the raw signed
// transaction together with its locally computed txid.
func mwUTXOSignTx(seed []byte, derivationPath, toAddress, valueStr string, p utxoChainParams) (string, string, error) {
	idx, err := p.indexer()
	if err != nil {
		return "", "", fmt.Errorf("%s indexer: %w", p.name, err)
	}
	fromAddr, err := mwUTXOAddressFromSeed(seed, derivationPath, p.p2pkhVersion)
	if err != nil {
		return "", "", fmt.Errorf("%s from-address: %w", p.name, err)
	}
	utxos, err := idx.ListUTXOs(fromAddr)
	if err != nil {
		return "", "", fmt.Errorf("%s UTXO fetch: %w", p.name, err)
	}
	if len(utxos) == 0 {
		return "", "", fmt.Errorf("no confirmed UTXOs for %s on %s", fromAddr, p.name)
	}
	valueSat, ok := new(big.Int).SetString(strings.TrimSpace(valueStr), 10)
	if !ok || valueSat.Sign() <= 0 {
		return "", "", fmt.Errorf("invalid %s value %q", p.name, valueStr)
	}

	// Fee is derived from the real serialized size of the inputs we select.
	// A single-input estimate picks the UTXO set, then the fee is recomputed
	// against the selected input count so multi-input sweeps stay relayable.
	fee := p.feeForSize(estimateUTXOTxSize(1, p.sighash))
	totalNeeded := new(big.Int).Add(valueSat, big.NewInt(fee))

	selected := make([]btcUTXO, 0, len(utxos))
	selectedValue := big.NewInt(0)
	for _, u := range utxos {
		selected = append(selected, u)
		selectedValue.Add(selectedValue, u.Value)
		if selectedValue.Cmp(totalNeeded) >= 0 {
			break
		}
	}
	if selectedValue.Cmp(totalNeeded) < 0 {
		return "", "", fmt.Errorf("insufficient %s UTXOs: have %s need %s", p.name, selectedValue.String(), totalNeeded.String())
	}
	fee = p.feeForSize(estimateUTXOTxSize(int64(len(selected)), p.sighash))
	totalNeeded = new(big.Int).Add(valueSat, big.NewInt(fee))
	if selectedValue.Cmp(totalNeeded) < 0 {
		return "", "", fmt.Errorf("insufficient %s UTXOs after fee adjustment: have %s need %s", p.name, selectedValue.String(), totalNeeded.String())
	}
	change := new(big.Int).Sub(selectedValue, totalNeeded)

	inputs := make([]chainsdk.UTXOInput, 0, len(selected))
	for _, u := range selected {
		// The ForkID and Zcash sighashes commit to each input's amount, so the
		// value travels with the input; the legacy sighash ignores it.
		inputs = append(inputs, chainsdk.UTXOInput{
			TxID:     u.TxID,
			Vout:     u.Vout,
			Satoshis: u.Value.Int64(),
		})
	}
	outputs := []chainsdk.UTXOOutput{{Address: toAddress, AmountSat: valueSat.Int64()}}
	if change.Sign() > 0 {
		outputs = append(outputs, chainsdk.UTXOOutput{Address: fromAddr, AmountSat: change.Int64()})
	}

	var rawTx string
	if p.sighash == "zcash" {
		rawTx, err = chainsdk.ZECSign(seed, derivationPath, inputs, outputs)
	} else {
		rawTx, err = chainsdk.UTXOSign(seed, derivationPath, p.name, inputs, outputs)
	}
	if err != nil {
		return "", "", fmt.Errorf("%s tx build+sign: %w", p.name, err)
	}
	txHash, err := btcTxHash(rawTx)
	if err != nil {
		return rawTx, "", nil
	}
	return rawTx, txHash, nil
}

// estimateUTXOTxSize estimates the serialized size of a 1-2 output P2PKH
// transaction with the given number of inputs, in bytes. It is used only to
// price the fee; the real transaction is built and signed by chainsdk.
func estimateUTXOTxSize(inputs int64, sighash string) int64 {
	if inputs < 1 {
		inputs = 1
	}
	// version(4) + count(1) + inputs*(36 outpoint + 1 scriptlen + 108 scriptSig + 4 sequence)
	// + count(1) + 2*(8 value + 1 scriptlen + 25 P2PKH) + locktime(4)
	const perInput = 36 + 1 + 108 + 4
	size := int64(4+1+1+4) + inputs*perInput + 2*(8+1+25)
	if sighash == "zcash" {
		// Zcash v4 adds versionGroupId(4), expiryHeight(4), valueBalance(8)
		// and the empty shielded-bundle counts(2).
		size += 18
	}
	return size
}

// btcUTXO represents an unspent transaction output from the blockstream API.
type btcUTXO struct {
	TxID  string
	Vout  uint32
	Value *big.Int
}

// btcTxHash computes the display txid (double-SHA256 of the serialized tx, reversed).
func btcTxHash(rawTxHex string) (string, error) {
	raw := parseHex(rawTxHex)
	if raw == nil {
		return "", fmt.Errorf("invalid hex")
	}
	h := doubleSHA256(raw)
	// Reverse bytes for display.
	for i, j := 0, len(h)-1; i < j; i, j = i+1, j-1 {
		h[i], h[j] = h[j], h[i]
	}
	return hexEncode(h), nil
}

// ----------------------------------------------------------------------------
// base58check encoding (Bitcoin P2PKH)
// ----------------------------------------------------------------------------

// Base58 and bech32 encoding are shared with every other TigerWallet backend
// through chainsdk. The implementations here were byte-identical (verified by
// TestEquivalenceProbe), so they delegate rather than maintain a second copy
// that can drift.

// base58CheckEncode encodes version||payload with a 4-byte double-SHA256
// checksum. chainsdk takes the version already prepended to the payload.
// base58CheckEncode encodes a version-prefixed payload with the standard
// 4-byte double-SHA256 checksum. The version prefix is 1-2 bytes and must
// already be prepended to the payload by the caller.
func base58CheckEncode(versioned []byte) string {
	return chainsdk.Base58CheckEncode(versioned)
}

func bech32Encode(hrp string, data []byte) (string, error) {
	if len(data) != 20 {
		return "", fmt.Errorf("expected 20-byte hash, got %d", len(data))
	}
	return chainsdk.Bech32Encode(hrp, data)
}

func base58Encode(data []byte) string {
	return chainsdk.Base58Encode(data)
}

func convertBits(data []byte, fromBits, toBits uint, pad bool) ([]byte, error) {
	return chainsdk.ConvertBits(data, fromBits, toBits, pad)
}

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
