package main

// non_evm_cardano_tx.go — real Cardano transaction construction.
//
// Cardano transactions are CBOR-encoded. This file implements the encoder and
// the Shelley-era transaction body for a simple ADA payment: build the body
// from the caller's selected UTXOs, sign the body hash with the CIP-1852
// ed25519-bip32 key (non_evm_cardano.go), assemble the witness set, and submit
// it to the node.
//
// The CBOR writer below is a strict subset encoder (unsigned integers of any
// width, byte strings, text strings, arrays, maps) rather than a general
// library, because the transaction shape is fixed and a general decoder would
// add surface area without adding capability. It refuses to encode values it
// cannot represent, so a malformed transaction cannot be produced silently.
//
// Input selection is the caller's responsibility: the backend does not pick
// UTXOs on the user's behalf and never invents an input. Outputs must balance
// against the inputs after the fee, and that is checked before signing.

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"

	"golang.org/x/crypto/blake2b"
)

// ---------- CBOR writer ----------

// cborWriter accumulates canonical CBOR. It is append-only and returns an
// error rather than panicking, so every construction failure is propagated to
// the caller instead of producing a truncated encoding.
type cborWriter struct {
	buf []byte
	err error
}

func (w *cborWriter) fail(err error) {
	if w.err == nil {
		w.err = err
	}
}

// major writes a CBOR type header: the 3-bit major type plus the argument,
// using the shortest legal representation for the value.
func (w *cborWriter) major(mt byte, v uint64) {
	if w.err != nil {
		return
	}
	switch {
	case v < 24:
		w.buf = append(w.buf, mt<<5|byte(v))
	case v <= 0xff:
		w.buf = append(w.buf, mt<<5|24, byte(v))
	case v <= 0xffff:
		w.buf = append(w.buf, mt<<5|25, byte(v>>8), byte(v))
	case v <= 0xffffffff:
		w.buf = append(w.buf, mt<<5|26, byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
	default:
		w.buf = append(w.buf, mt<<5|27,
			byte(v>>56), byte(v>>48), byte(v>>40), byte(v>>32),
			byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
	}
}

func (w *cborWriter) Uint(v uint64) { w.major(0, v) }

func (w *cborWriter) Bytes(b []byte) {
	w.major(2, uint64(len(b)))
	if w.err == nil {
		w.buf = append(w.buf, b...)
	}
}

func (w *cborWriter) Text(s string) {
	w.major(3, uint64(len(s)))
	if w.err == nil {
		w.buf = append(w.buf, s...)
	}
}

func (w *cborWriter) Array(n int) { w.major(4, uint64(n)) }

func (w *cborWriter) Map(n int) { w.major(5, uint64(n)) }

// Raw appends pre-encoded CBOR (used for nested structures built separately).
func (w *cborWriter) Raw(b []byte) {
	if w.err == nil {
		w.buf = append(w.buf, b...)
	}
}

func (w *cborWriter) BytesOut() ([]byte, error) {
	if w.err != nil {
		return nil, w.err
	}
	return w.buf, nil
}

// ---------- transaction types ----------

// CardanoUTXO is one unspent output selected by the caller.
type CardanoUTXO struct {
	TxHash     string `json:"tx_hash"`     // 64 hex chars
	Index      uint32 `json:"index"`       // output index
	Amount     uint64 `json:"amount"`      // lovelace
	AddressB32 string `json:"address_b32"` // bech32 address holding it
}

// CardanoOutput is one destination output.
type CardanoOutput struct {
	Address string `json:"address"` // bech32
	Amount  uint64 `json:"amount"`  // lovelace
}

// ---------- encoding ----------

// cborTxInput encodes a Shelley transaction input: [bytes(txhash), uint(index)].
func cborTxInput(txHash []byte, index uint32) []byte {
	w := &cborWriter{}
	w.Array(2)
	w.Bytes(txHash)
	w.Uint(uint64(index))
	return w.buf
}

// cborValueCoin encodes a value as plain lovelace.
func cborValueCoin(amount uint64) []byte {
	w := &cborWriter{}
	w.Uint(amount)
	return w.buf
}

// cborTxOutput encodes a Shelley output: [addressBytes, value].
func cborTxOutput(addressBytes []byte, amount uint64) []byte {
	w := &cborWriter{}
	w.Array(2)
	w.Bytes(addressBytes)
	w.Raw(cborValueCoin(amount))
	return w.buf
}

// buildCardanoBody assembles the transaction body map (Shelley era, keys 0-2).
//
//	0 : set of inputs
//	1 : list of outputs
//	2 : fee
//
// Inputs are sorted by (txhash, index) as the ledger requires for canonical
// ordering, and duplicate inputs are rejected outright: spending the same
// output twice would be rejected by the node, and silently de-duplicating
// would hide a caller bug that changes the fee calculation.
func buildCardanoBody(inputs []CardanoUTXO, outputs []CardanoOutput, fee uint64) ([]byte, error) {
	if len(inputs) == 0 {
		return nil, errors.New("cardano: no inputs selected")
	}
	if len(outputs) == 0 {
		return nil, errors.New("cardano: no outputs")
	}

	type encIn struct {
		hash []byte
		idx  uint32
	}
	encInputs := make([]encIn, 0, len(inputs))
	seen := make(map[string]bool, len(inputs))
	for i, in := range inputs {
		h, err := hex.DecodeString(strings.TrimPrefix(strings.TrimSpace(in.TxHash), "0x"))
		if err != nil {
			return nil, fmt.Errorf("cardano: input %d tx_hash is not hex: %w", i, err)
		}
		if len(h) != 32 {
			return nil, fmt.Errorf("cardano: input %d tx_hash must be 32 bytes, got %d", i, len(h))
		}
		// Cardano compares input references by hash then index, so key
		// the duplicate check the same way.
		key := hex.EncodeToString(h) + ":" + fmt.Sprint(in.Idx())
		if seen[key] {
			return nil, fmt.Errorf("cardano: input %s is selected twice", key)
		}
		seen[key] = true
		encInputs = append(encInputs, encIn{hash: h, idx: in.Idx()})
	}
	sort.Slice(encInputs, func(a, b int) bool {
		cmp := strings.Compare(hex.EncodeToString(encInputs[a].hash), hex.EncodeToString(encInputs[b].hash))
		if cmp != 0 {
			return cmp < 0
		}
		return encInputs[a].idx < encInputs[b].idx
	})

	encOutputs := make([][]byte, 0, len(outputs))
	for i, out := range outputs {
		if strings.TrimSpace(out.Address) == "" {
			return nil, fmt.Errorf("cardano: output %d has no address", i)
		}
		addrBytes, err := cardanoAddressBytes(out.Address)
		if err != nil {
			return nil, fmt.Errorf("cardano: output %d address: %w", i, err)
		}
		encOutputs = append(encOutputs, cborTxOutput(addrBytes, out.Amount))
	}

	w := &cborWriter{}
	w.Map(3)
	// key 0: inputs (set)
	w.Uint(0)
	w.Array(len(encInputs))
	for _, in := range encInputs {
		w.Raw(cborTxInput(in.hash, in.idx))
	}
	// key 1: outputs (list)
	w.Uint(1)
	w.Array(len(encOutputs))
	for _, o := range encOutputs {
		w.Raw(o)
	}
	// key 2: fee
	w.Uint(2)
	w.Uint(fee)
	return w.BytesOut()
}

// Idx returns the output index as a uint32. Present so the duplicate-input key
// reads the same value that gets encoded.
func (c CardanoUTXO) Idx() uint32 { return c.Index }

// cborWitnessSet encodes the witness set map containing only the vkey witnesses
// (key 0): a map from public key hash to [signature, public key].
func cborWitnessSet(witnesses map[string][]byte, pubKeys map[string][]byte) []byte {
	// Deterministic ordering keeps the encoding stable for identical inputs.
	keys := make([]string, 0, len(witnesses))
	for k := range witnesses {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	w := &cborWriter{}
	w.Map(1)
	w.Uint(0)
	w.Map(len(keys))
	for _, k := range keys {
		kh, _ := hex.DecodeString(k)
		w.Bytes(kh)
		w.Array(2)
		w.Bytes(witnesses[k])
		w.Bytes(pubKeys[k])
	}
	return w.buf
}

// CardanoBuildSend builds, signs, and optionally broadcasts a Cardano ADA
// payment. The transaction hash is the blake2b-256 of the body, which is the
// txid the network uses. Returns the hex-encoded signed transaction and the
// transaction hash.
func CardanoBuildSend(ctx context.Context, seed []byte, path, endpoint string, inputs []CardanoUTXO, outputs []CardanoOutput, fee uint64, broadcast bool) (string, string, error) {
	if endpoint == "" {
		return "", "", errors.New("cardano: no node endpoint configured")
	}

	// Check the arithmetic before encoding anything: a transaction whose inputs
	// cannot pay for its outputs plus the fee is invalid however it serializes,
	// so reporting that first gives the caller a usable error instead of an
	// encoding detail.
	var inTotal, outTotal uint64
	for _, in := range inputs {
		inTotal += in.Amount
	}
	for _, out := range outputs {
		outTotal += out.Amount
	}
	if inTotal < outTotal+fee {
		return "", "", fmt.Errorf("cardano: inputs %d do not cover outputs %d + fee %d", inTotal, outTotal, fee)
	}

	body, err := buildCardanoBody(inputs, outputs, fee)
	if err != nil {
		return "", "", err
	}

	node, err := cardanoDerive(seed, path)
	if err != nil {
		return "", "", err
	}
	pub := cardanoPubKey(node)
	pkh := cardanoKeyHash(pub)

	// Cardano signs the blake2b-256 hash of the transaction body.
	bodyHash := blake2b.Sum256(body)
	sig, _, err := CardanoSign(seed, path, bodyHash[:])
	if err != nil {
		return "", "", fmt.Errorf("cardano sign: %w", err)
	}

	ws := cborWitnessSet(
		map[string][]byte{hex.EncodeToString(pkh): sig},
		map[string][]byte{hex.EncodeToString(pkh): pub},
	)

	// Transaction = [body, witness_set, auxiliary_data(null)].
	tx := &cborWriter{}
	tx.Array(3)
	tx.Raw(body)
	tx.Raw(ws)
	tx.buf = append(tx.buf, 0xf6) // CBOR null for the absent auxiliary data
	txBytes, err := tx.BytesOut()
	if err != nil {
		return "", "", err
	}

	txHash := hex.EncodeToString(bodyHash[:])
	if !broadcast {
		return hex.EncodeToString(txBytes), txHash, nil
	}

	// Submit through the node's submit-api, which takes the raw CBOR.
	if _, err := postRawCBOR(ctx, strings.TrimRight(endpoint, "/")+"/api/submit/tx", txBytes); err != nil {
		return "", "", fmt.Errorf("cardano submit: %w", err)
	}
	return hex.EncodeToString(txBytes), txHash, nil
}

// cardanoKeyHash computes the 28-byte blake2b-224 key hash of a public key.
func cardanoKeyHash(pub []byte) []byte {
	h, err := blake2b.New(28, nil)
	if err != nil {
		return nil
	}
	h.Write(pub)
	return h.Sum(nil)
}

// cardanoAddressBytes decodes a bech32 Shelley address to its raw bytes.
func cardanoAddressBytes(addr string) ([]byte, error) {
	a := strings.TrimSpace(addr)
	if !strings.HasPrefix(strings.ToLower(a), "addr") && !strings.HasPrefix(strings.ToLower(a), "stake") {
		return nil, fmt.Errorf("not a cardano bech32 address: %q", a)
	}
	return bech32DecodeToBytes(a)
}
