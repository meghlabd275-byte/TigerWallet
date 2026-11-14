package chainsdk

// non_evm_substrate_tx.go — real Substrate (Polkadot/Kusama) extrinsic
// construction.
//
// A signed transfer extrinsic is:
//
//	[compact(len) | 0x84 | MultiAddress::Id(signer) | MultiSignature::Sr25519 |
//	 signed-extra | call]
//
// For current Polkadot/Kusama runtime metadata the signed-extra is
// (CheckMortality, CheckNonce, ChargeTransactionPayment) and the additional
// signed payload is (CheckSpecVersion, CheckTxVersion, CheckGenesis,
// CheckEra). Spec version, transaction version and genesis hash are read from
// the node because they change on every runtime upgrade; a hardcoded value
// would produce an extrinsic the chain rejects.
//
// SCALE encoding is implemented here rather than pulled from a dependency so
// the exact byte layout stays visible and testable.

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"golang.org/x/crypto/blake2b"
)

// ---------- SCALE writer ----------

// scaleWriter accumulates SCALE-encoded bytes. Every method records an error
// rather than panicking, so encoding failures surface at the call site.
type scaleWriter struct {
	buf []byte
	err error
}

func (w *scaleWriter) fail(err error) {
	if w.err == nil {
		w.err = err
	}
}

func (w *scaleWriter) U8(v byte) {
	if w.err == nil {
		w.buf = append(w.buf, v)
	}
}

func (w *scaleWriter) U16(v uint16) {
	if w.err != nil {
		return
	}
	var b [2]byte
	binary.LittleEndian.PutUint16(b[:], v)
	w.buf = append(w.buf, b[:]...)
}

func (w *scaleWriter) U32(v uint32) {
	if w.err != nil {
		return
	}
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	w.buf = append(w.buf, b[:]...)
}

// Fixed appends a fixed-size byte array with no length prefix.
func (w *scaleWriter) Fixed(b []byte) {
	if w.err == nil {
		w.buf = append(w.buf, b...)
	}
}

// Compact writes a SCALE compact-encoded unsigned integer.
//
//	0..2^6-1     : single byte, value << 2
//	2^6..2^14-1  : two bytes,  (value << 2) | 0b01
//	2^14..2^30-1 : four bytes, (value << 2) | 0b10
//	otherwise    : big-integer form, ((len-4) << 2) | 0b11 then LE bytes
func (w *scaleWriter) Compact(v uint64) {
	if w.err != nil {
		return
	}
	switch {
	case v < 1<<6:
		w.U8(byte(v) << 2)
	case v < 1<<14:
		w.U16(uint16(v)<<2 | 0b01)
	case v < 1<<30:
		w.U32(uint32(v)<<2 | 0b10)
	default:
		var le [8]byte
		binary.LittleEndian.PutUint64(le[:], v)
		n := 8
		for n > 4 && le[n-1] == 0 {
			n--
		}
		w.U8(byte(n-4)<<2 | 0b11)
		w.buf = append(w.buf, le[:n]...)
	}
}

// CompactBig writes a compact-encoded unsigned integer of any width, used for
// u128 balances.
func (w *scaleWriter) CompactBig(v *big.Int) {
	if w.err != nil {
		return
	}
	if v == nil {
		w.fail(errors.New("scale: nil compact value"))
		return
	}
	if v.Sign() < 0 {
		w.fail(errors.New("scale: negative compact value"))
		return
	}
	// The four-byte form covers everything below 2^30, so a value with exactly
	// 30 significant bits (2^30-1) still fits it.
	if v.BitLen() <= 30 {
		w.Compact(v.Uint64())
		return
	}
	be := v.Bytes()
	le := make([]byte, 0, len(be))
	for i := len(be) - 1; i >= 0; i-- {
		le = append(le, be[i])
	}
	n := len(le)
	if n < 4 {
		for len(le) < 4 {
			le = append(le, 0)
		}
		n = 4
	}
	w.U8(byte(n-4)<<2 | 0b11)
	w.buf = append(w.buf, le...)
}

func (w *scaleWriter) BytesOut() ([]byte, error) {
	if w.err != nil {
		return nil, w.err
	}
	return w.buf, nil
}

// ---------- extrinsic construction ----------

// SubstrateChainParams carries the runtime values that must come from the node
// because they change across runtime upgrades.
type SubstrateChainParams struct {
	SpecVersion uint32 `json:"spec_version"`
	TxVersion   uint32 `json:"tx_version"`
	GenesisHash string `json:"genesis_hash"`
	BlockHash   string `json:"block_hash"`
	Nonce       uint64 `json:"nonce"`
	Tip         uint64 `json:"tip"`
	EraPeriod   uint64 `json:"era_period"`
}

// encodeMortalEra encodes a mortal era. period must be a power of two in
// [4, 65536]; current is the current block number.
func encodeMortalEra(current, period uint64) ([]byte, error) {
	if period < 4 || period > 65536 || (period&(period-1)) != 0 {
		return nil, fmt.Errorf("substrate: era period %d must be a power of two in [4,65536]", period)
	}
	var tz uint64
	for p := period; p > 1 && p&1 == 0; p >>= 1 {
		tz++
	}
	return []byte{byte(tz - 1), byte((current / period) % 256)}, nil
}

// substrateDecodeSS58 decodes an SS58 address to its 32-byte account id,
// verifying the two-byte checksum.
func substrateDecodeSS58(addr string) ([]byte, error) {
	raw, err := Base58Decode(strings.TrimSpace(addr))
	if err != nil {
		return nil, err
	}
	if len(raw) < 3 {
		return nil, errors.New("ss58 address too short")
	}
	payload := raw[:len(raw)-2]
	got := raw[len(raw)-2:]
	if len(payload) < 2 {
		return nil, errors.New("ss58 payload too short")
	}
	h, err := blake2b.New(64, nil)
	if err != nil {
		return nil, err
	}
	h.Write([]byte("SS58PRE"))
	h.Write(payload)
	want := h.Sum(nil)
	if got[0] != want[0] || got[1] != want[1] {
		return nil, errors.New("ss58 checksum mismatch")
	}
	// The account id is the 32 bytes after the 1-byte network prefix.
	id := payload[1:]
	if len(id) != 32 {
		return nil, fmt.Errorf("ss58 account id must be 32 bytes, got %d", len(id))
	}
	return id, nil
}

// SubstrateBuildTransfer builds and signs a balances.transferKeepAlive
// extrinsic, broadcasting it when requested.
//
// callIndex is the (pallet, call) index pair from the runtime metadata. It is
// runtime-specific, so it is supplied by the caller rather than guessed; a
// wrong index produces an extrinsic the chain rejects.
func SubstrateBuildTransfer(
	ctx context.Context,
	seed []byte,
	path, endpoint string,
	callIndex [2]byte,
	to string,
	amount *big.Int,
	params SubstrateChainParams,
	broadcast bool,
) (string, string, error) {
	if endpoint == "" {
		return "", "", errors.New("substrate: no node endpoint configured")
	}
	if params.SpecVersion == 0 || params.TxVersion == 0 {
		return "", "", errors.New("substrate: spec_version and tx_version must come from the node's runtime metadata")
	}
	if amount == nil || amount.Sign() <= 0 {
		return "", "", errors.New("substrate: transfer amount must be positive")
	}
	genesis, err := hexToFixed32(params.GenesisHash)
	if err != nil {
		return "", "", fmt.Errorf("substrate genesis_hash: %w", err)
	}
	toPub, err := substrateDecodeSS58(to)
	if err != nil {
		return "", "", fmt.Errorf("substrate recipient %q: %w", to, err)
	}

	// --- call data: [pallet, call, MultiAddress::Id, Compact<u128>] ---
	call := &scaleWriter{}
	call.U8(callIndex[0])
	call.U8(callIndex[1])
	call.U8(0x00) // MultiAddress::Id
	call.Fixed(toPub)
	call.CompactBig(amount)
	callData, err := call.BytesOut()
	if err != nil {
		return "", "", err
	}

	// --- signed extra ---
	var era []byte
	var eraFull []byte
	if params.EraPeriod == 0 {
		era = []byte{0x00} // immortal
	} else {
		current, err := substrateBlockNumber(ctx, endpoint)
		if err != nil {
			return "", "", fmt.Errorf("substrate: read block number for era: %w", err)
		}
		era, err = encodeMortalEra(current, params.EraPeriod)
		if err != nil {
			return "", "", err
		}
		// The signed payload replaces the 2-byte era with an 8-byte one
		// whose first two bytes are identical.
		eraFull = make([]byte, 8)
		copy(eraFull, era)
	}

	extra := &scaleWriter{}
	extra.Fixed(era)
	extra.Compact(params.Nonce)
	extra.Compact(params.Tip)
	extraBytes, err := extra.BytesOut()
	if err != nil {
		return "", "", err
	}

	// --- additional signed payload ---
	additional := &scaleWriter{}
	additional.U32(params.SpecVersion)
	additional.U32(params.TxVersion)
	additional.Fixed(genesis)
	if params.EraPeriod == 0 {
		additional.Fixed(genesis)
	} else {
		bh, err := hexToFixed32(params.BlockHash)
		if err != nil {
			return "", "", fmt.Errorf("substrate block_hash: %w", err)
		}
		additional.Fixed(bh)
	}
	additionalBytes, err := additional.BytesOut()
	if err != nil {
		return "", "", err
	}

	// --- payload to sign: call ++ extra ++ additional ---
	payload := append([]byte(nil), callData...)
	if eraFull != nil {
		payload = append(payload, eraFull...)
		payload = append(payload, extraBytes[2:]...)
	} else {
		payload = append(payload, extraBytes...)
	}
	payload = append(payload, additionalBytes...)

	// Payloads over 256 bytes are hashed before signing.
	toSign := payload
	if len(payload) > 256 {
		h := blake2b.Sum256(payload)
		toSign = h[:]
	}

	signature, pubKey, err := SubstrateSign(seed, path, toSign, 0)
	if err != nil {
		return "", "", fmt.Errorf("substrate sign: %w", err)
	}
	if len(signature) != 64 || len(pubKey) != 32 {
		return "", "", fmt.Errorf("substrate signer returned an unexpected key/signature size (%d/%d)", len(pubKey), len(signature))
	}

	// --- assemble ---
	body := &scaleWriter{}
	body.U8(0x84) // signed, version 4
	body.U8(0x00) // MultiAddress::Id
	body.Fixed(pubKey)
	body.U8(0x01) // MultiSignature::Sr25519
	body.Fixed(signature)
	body.Fixed(extraBytes)
	body.Fixed(callData)
	bodyBytes, err := body.BytesOut()
	if err != nil {
		return "", "", err
	}

	extrinsic := &scaleWriter{}
	extrinsic.Compact(uint64(len(bodyBytes)))
	extrinsic.Fixed(bodyBytes)
	extBytes, err := extrinsic.BytesOut()
	if err != nil {
		return "", "", err
	}

	extHex := "0x" + hex.EncodeToString(extBytes)
	extHash := blake2b.Sum256(extBytes)
	txHash := "0x" + hex.EncodeToString(extHash[:])

	if !broadcast {
		return extHex, txHash, nil
	}
	if err := substrateSubmit(ctx, endpoint, extHex); err != nil {
		return "", "", err
	}
	return extHex, txHash, nil
}

// hexToFixed32 decodes a 0x-prefixed 32-byte hex string.
func hexToFixed32(s string) ([]byte, error) {
	t := strings.TrimPrefix(strings.TrimSpace(s), "0x")
	b, err := hex.DecodeString(t)
	if err != nil {
		return nil, err
	}
	if len(b) != 32 {
		return nil, fmt.Errorf("expected 32 bytes, got %d", len(b))
	}
	return b, nil
}

// substrateBlockNumber reads the current best block number via chain_getHeader.
func substrateBlockNumber(ctx context.Context, endpoint string) (uint64, error) {
	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"chain_getHeader","params":[]}`)
	resp, err := postRaw(ctx, endpoint, body)
	if err != nil {
		return 0, err
	}
	var parsed struct {
		Result struct {
			Number string `json:"number"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := unmarshalCheckError(resp, &parsed); err != nil {
		return 0, err
	}
	if parsed.Error != nil {
		return 0, errors.New("chain_getHeader: " + parsed.Error.Message)
	}
	n, ok := new(big.Int).SetString(strings.TrimPrefix(parsed.Result.Number, "0x"), 16)
	if !ok {
		return 0, errors.New("chain_getHeader returned a malformed block number")
	}
	return n.Uint64(), nil
}

// substrateSubmit submits a signed extrinsic and confirms the node accepted it.
func substrateSubmit(ctx context.Context, endpoint, extHex string) error {
	body := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"author_submitExtrinsic","params":["%s"]}`, extHex)
	resp, err := postRaw(ctx, endpoint, []byte(body))
	if err != nil {
		return fmt.Errorf("substrate submit: %w", err)
	}
	var parsed struct {
		Result string `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := unmarshalCheckError(resp, &parsed); err != nil {
		return err
	}
	if parsed.Error != nil {
		return errors.New("substrate submit rejected: " + parsed.Error.Message)
	}
	if parsed.Result == "" {
		return errors.New("substrate submit returned no extrinsic hash")
	}
	return nil
}
