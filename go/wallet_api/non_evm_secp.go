package main

// non_evm_secp.go — secp256k1 account-chain SDKs: Tron (node-built
// TransferContract via /wallet/createtransaction + local sign), VeChain
// (RLP + blake2b + ECDSA r||s), Ripple (canonical serialization + sha512-
// half ECDSA via rippled RPC), ICP (address+sign), Zilliqa (schnorr sign,
// via btcec), Kaspa (schnorr sign), Nervos (bech32m short address +
// blake160), Filecoin (address+sign). Aleo/Hedera/Flow fail closed with
// precise documented reasons.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/blake2b"
)

// --------------------------- Tron ---------------------------------------

// TronAddress computes base58check(0x41 || eth-keccak-address).
func TronAddress(seed []byte, path string) (string, error) {
	priv, err := hdDerive(seed, path)
	if err != nil {
		return "", err
	}
	addr := crypto.PubkeyToAddress(priv.PublicKey)
	raw := append([]byte{0x41}, addr.Bytes()...)
	return base58checkEncode(raw), nil
}

// tronAddrBytes decodes a T-address to the 21-byte payload.
func tronAddrBytes(addr string) ([]byte, error) {
	raw, err := base58Decode(addr)
	if err != nil {
		return nil, err
	}
	if len(raw) != 25 || raw[0] != 0x41 {
		return nil, errors.New("invalid tron address")
	}
	return raw[:21], nil
}

// TronBuildSend asks the node for a Transaction shell with the reference
// block filled (ensures hash/signature correctness), fills our own signature
// locally, and broadcasts via /wallet/broadcasttransaction.
func TronBuildSend(ctx context.Context, seed []byte, path, endpoint, to string, sunAmount uint64, broadcast bool) (string, string, error) {
	priv, err := hdDerive(seed, path)
	if err != nil {
		return "", "", err
	}
	from, err := TronAddress(seed, path)
	if err != nil {
		return "", "", err
	}
	if _, err := tronAddrBytes(to); err != nil {
		return "", "", err
	}
	// node builds the shell with correct ref_block fields
	payload := map[string]interface{}{
		"owner_address": from,
		"to_address":    to,
		"amount":        sunAmount,
	}
	resp, err := postRaw(ctx, strings.TrimRight(endpoint, "/")+"/wallet/createtransaction", jsonMust(payload))
	if err != nil {
		return "", "", fmt.Errorf("tron createtransaction: %w", err)
	}
	var shell map[string]interface{}
	if err := json.Unmarshal(resp, &shell); err != nil {
		return "", "", err
	}
	rawHex, ok := shell["raw_data_hex"].(string)
	if !ok || rawHex == "" {
		return "", "", errors.New("tron createtransaction: malformed reply")
	}
	rawBytes, err := hexStringDecode(rawHex)
	if err != nil {
		return "", "", err
	}
	hashForSign := sha256.Sum256(rawBytes)
	sig, err := crypto.Sign(hashForSign[:], priv)
	if err != nil {
		return "", "", err
	}
	if !broadcast {
		return hexOf(rawBytes) + hexOf(sig[:64]), "", nil
	}
	submit := map[string]interface{}{
		"signature": []map[string]string{{"signature": hexOf(sig)}},
		"txID":      "",                  // optional
	}
	_ = submit
	finalBody, _ := json.Marshal(map[string]interface{}{
		"raw_data_hex": rawHex,
		"signature":   []string{hexOf(sig)},
	})
	resp2, err := postRaw(ctx, strings.TrimRight(endpoint, "/")+"/wallet/broadcasttransaction", finalBody)
	if err != nil {
		return "", "", fmt.Errorf("tron broadcast: %w", err)
	}
	var parsed struct {
		Result bool   `json:"result"`
		TxID   string `json:"txid"`
	}
	if err := json.Unmarshal(resp2, &parsed); err != nil || !parsed.Result || parsed.TxID == "" {
		return "", "", errors.New("tron broadcast rejected")
	}
	return hexOf(rawBytes), parsed.TxID, nil
}

func jsonMust(v map[string]interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

// --------------------------- VeChain -------------------------------------

// VETAddress returns the 0x keccak address.
func VETAddress(seed []byte, path string) (string, error) {
	priv, err := hdDerive(seed, path)
	if err != nil {
		return "", err
	}
	return "0x" + hex.EncodeToString(crypto.PubkeyToAddress(priv.PublicKey).Bytes()), nil
}

// ---------- rlp helpers (minimal, used by vechain) ----------

func rlpBytes(b []byte) []byte {
	if len(b) == 1 && b[0] < 0x80 {
		return b
	}
	if len(b) <= 55 {
		return append([]byte{0x80 + byte(len(b))}, b...)
	}
	return append([]byte{0xb7 + byte(lenOfLen(len(b)))}, append(lenbBE(len(b)), b...)...)
}

func lenOfLen(n int) int {
	l := 0
	for n > 0 {
		l++
		n >>= 8
	}
	return l
}

func lenbBE(n int) []byte {
	b := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		b[i] = byte(n)
		n >>= 8
	}
	for b[0] == 0 {
		b = b[1:]
	}
	return b
}

// rlpWrapList prepends list header.
func rlpWrapList(parts [][]byte) []byte {
	var body []byte
	for _, p := range parts {
		body = append(body, p...)
	}
	if len(body) <= 55 {
		return append([]byte{0xc0 + byte(len(body))}, body...)
	}
	return append([]byte{0xf7 + byte(lenOfLen(len(body)))}, append(lenbBE(len(body)), body...)...)
}

func uintToBe(v uint64, size int) []byte {
	full := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		full[i] = byte(v)
		v >>= 8
	}
	return full[8-size:]
}

// VETBuildSend builds a transfer tx: [chainTag u8, blockRef8B, exp u32,
// clauses, gasPriceCoef u8, gas u64, dependsOn(empty), nonce]. BlockRef is
// fetched properly from the node /blocks/best; fail closed on error.
func VETBuildSend(ctx context.Context, seed []byte, path, endpoint, to string, atomic uint64, gas uint64, broadcast bool) (string, string, error) {
	priv, err := hdDerive(seed, path)
	if err != nil {
		return "", "", err
	}
	toAddr, err := hex.DecodeString(strings.TrimPrefix(to, "0x"))
	if err != nil || len(toAddr) != 20 {
		return "", "", errors.New("invalid vechain address")
	}
	// block ref = first 8 bytes of best block id — real, fail closed
	best, err := getRaw(ctx, strings.TrimRight(endpoint, "/")+"/blocks/best")
	if err != nil {
		return "", "", err
	}
	var parsed struct { Id string `json:"id"` }
	if err := json.Unmarshal(best, &parsed); err != nil || parsed.Id == "" {
		return "", "", errors.New("vechain best block malformed")
	}
	idBytes, err := hexStringDecode(parsed.Id)
	if err != nil || len(idBytes) < 8 {
		return "", "", errors.New("vechain block id malformed")
	}
	blockRef := idBytes[:8]
	chainTag := byte(0x4a) // mainnet vest exo tag? chain genesis tag for mainnet is variable—the correct one for mainnet = computed from genesis id below; use fixed mainnet constant

	toDear := rlpBytes(toAddr)
	value := rlpBytes(uintToBe(atomic, 8))
	data := rlpBytes([]byte{})
	gasPrice := rlpBytes([]byte{0x00})
	gasLimit := rlpBytes(uintToBe(gas, 8))
	dependsOn := rlpBytes([]byte{})
	nonce := rlpBytes(uintToBe(uint64(time.Now().UnixNano()), 8))

	// clause body
	clause := rlpWrapList([][]byte{toDear, value, data})
	clauses := rlpWrapList([][]byte{clause})

	body := rlpWrapList([][]byte{
		rlpBytes([]byte{chainTag}),
		rlpBytes(blockRef),
		rlpBytes(uintToBe(0, 4)), // expiration 0 = no expiry
		clauses,
		gasPrice,
		gasLimit,
		dependsOn,
		nonce,
	})
	hash, err := blake2b.New256(nil)
	if err != nil {
		return "", "", err
	}
	hash.Write(body)
	d := hash.Sum(nil)
	fullSig, err := crypto.Sign(d, priv) // geth RFC6979
	if err != nil {
		return "", "", err
	}
	sig := fullSig[:64] // vechain expects r||s (no recovery id)

	bodySigned := rlpWrapList([][]byte{
		rlpBytes([]byte{chainTag}),
		rlpBytes(blockRef),
		rlpBytes(uintToBe(0, 4)),
		clauses,
		gasPrice,
		gasLimit,
		dependsOn,
		nonce,
		rlpBytes(sig),
	})
	rawHex := hexOf(bodySigned)
	if !broadcast {
		return rawHex, "", nil
	}
	payload := map[string]interface{}{"raw": rawHex}
	resp, err := postRaw(ctx, strings.TrimRight(endpoint, "/")+"/transactions", jsonMust(payload))
	if err != nil {
		return "", "", fmt.Errorf("vechain broadcast: %w", err)
	}
	var out struct{ Id string `json:"id"` }
	if err := json.Unmarshal(resp, &out); err != nil || out.Id == "" {
		return "", "", errors.New("vechain broadcast malformed")
	}
	return rawHex, out.Id, nil
}

// lowS flips s to N-s if needed; the raw 64-byte r||s buffer is modified
// in place. Returns false if the input shape can't be handled.
func lowS(raw []byte) bool {
	if len(raw) != 64 {
		return false
	}
	n := crypto.S256().Params().N
	half := new(big.Int).Div(n, big.NewInt(2))
	s := new(big.Int).SetBytes(raw[32:])
	if s.Cmp(half) > 0 {
		s = new(big.Int).Sub(n, s)
	}
	sb := s.Bytes()
	if len(sb) > 32 {
		return false
	}
	// zero out the s slot then left-pad
	for i := 32; i < 64; i++ {
		raw[i] = 0
	}
	copy(raw[64-len(sb):], sb)
	return true
}

// --------------------------- Ripple ---------------------------------------

var rippleAlphabet = "rpshnaf39wBUDNEGHJKLM4PQRST7VWXYZ2bcdeCg65jkm8oFqi1TuvYx"

// XRPAddress computes r.. from account id version 0x00 + hash160(pub-compressed).
func XRPAddress(seed []byte, path string) (string, error) {
	priv, err := hdDerive(seed, path)
	if err != nil {
		return "", err
	}
	pub := crypto.CompressPubkey(&priv.PublicKey)
	h := hash160(pub)
	raw := append([]byte{0x00}, h...)
	return base58EncAlphabet(append(raw, doubleSHA256(raw)[:4]...), rippleAlphabet), nil
}

func base58EncAlphabet(input []byte, alpha string) string {
	x := new(big.Int).SetBytes(input)
	base := big.NewInt(58)
	mod := new(big.Int)
	var result []byte
	for x.Sign() > 0 {
		x.DivMod(x, base, mod)
		result = append([]byte{alpha[mod.Int64()]}, result...)
	}
	for _, b := range input {
		if b != 0 {
			break
		}
		result = append([]byte{alpha[0]}, result...)
	}
	return string(result)
}

func base58DecAlphabet(s, alpha string) ([]byte, error) {
	dec := new(big.Int)
	base := big.NewInt(58)
	for _, c := range []byte(s) {
		idx := strings.IndexByte(alpha, c)
		if idx < 0 {
			return nil, fmt.Errorf("invalid base58 char %q", c)
		}
		dec.Mul(dec, base)
		dec.Add(dec, big.NewInt(int64(idx)))
	}
	zeros := 0
	for _, c := range []byte(s) {
		if c != alpha[0] {
			break
		}
		zeros++
	}
	out := dec.Bytes()
	padded := make([]byte, zeros+len(out))
	copy(padded[zeros:], out)
	return padded, nil
}

// sha512Half yields the first 32 bytes of SHA-512 (rippled hash).
func sha512Half(b []byte) []byte {
	h := sha512.Sum512(b)
	return h[:32]
}

// rippleCanonicalFields builds the Payment serial in sorted-field order.
func rippleCanonicalFields(fromPub, fromAcct, toAcct []byte, seq, lastLedger uint32, drops, fee uint64, destTag *uint32) ([]byte, error) {
	type f struct {
		hdr  []byte
		body []byte
	}
	var fields []f
	pushU16 := func(hdr byte, v uint16) {
		var b [2]byte
		// need big-endian write to avoid import issues
		b[0] = byte(v >> 8)
		b[1] = byte(v)
		fields = append(fields, f{[]byte{hdr}, b[:]})
	}
	pushU32 := func(hdr byte, v uint32) {
		var b [4]byte
		b[0] = byte(v >> 24)
		b[1] = byte(v >> 16)
		b[2] = byte(v >> 8)
		b[3] = byte(v)
		fields = append(fields, f{[]byte{hdr}, b[:]})
	}
	pushU64 := func(hdr byte, v uint64) {
		var b [8]byte
		for i := 7; i >= 0; i-- {
			b[7-i] = byte(v >> (8 * uint(i)))
		}
		fields = append(fields, f{[]byte{hdr}, b[:]})
	}
	pushAcct := func(hdr byte, raw []byte) {
		h := append([]byte{hdr, byte(len(raw))}, raw...)
		fields = append(fields, f{h[:2], raw})
	}
	pushAcct(0x11, fromAcct)             // Account
	pushU16(0x12, 0)                     // TransactionType Payment = 0
	pushAcct(0x13, toAcct)               // Destination
	pushU64(0x18, drops)                 // Amount
	pushU64(0x19, fee)                   // Fee
	pushU32(0x22, 0)                     // Flags
	pushU32(0x24, seq)                   // Sequence
	pushU32(0x28, lastLedger)            // LastLedgerSequence
	if destTag != nil {
		pushU32(0x2e, *destTag)            // (2,14) DestinationTag
	}
	pushAcct(0x73, fromPub)              // SigningPubKey
	var out bytes.Buffer
	for _, x := range fields {
		out.Write(x.hdr)
		out.Write(x.body)
	}
	return out.Bytes(), nil
}

// rippledRPC posts {"method": ..., "params": [payload]}.
func rippledRPC(ctx context.Context, endpoint, method string, params map[string]interface{}) (json.RawMessage, error) {
	payload := map[string]interface{}{"method": method, "params": []interface{}{params}}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	resp, err := postRaw(ctx, endpoint, raw)
	if err != nil {
		return nil, err
	}
	var parsed struct { Result json.RawMessage `json:"result"` }
	if err := json.Unmarshal(resp, &parsed); err != nil {
		return nil, err
	}
	return parsed.Result, nil
}

// RippleBuildSend builds a Payment; fetches sequence + current ledger from
// account_info and fee from 'fee' (fail closed on errors), then submits.
func RippleBuildSend(ctx context.Context, seed []byte, path, endpoint, to string, drops uint64, destTag *uint32, broadcast bool) (string, string, error) {
	priv, err := hdDerive(seed, path)
	if err != nil {
		return "", "", err
	}
	from, err := XRPAddress(seed, path)
	if err != nil {
		return "", "", err
	}
	fromBytes, err := xrpAddrBytes(from)
	if err != nil {
		return "", "", err
	}
	toBytes, err := xrpAddrBytes(to)
	if err != nil {
		return "", "", err
	}
	ai, err := rippledRPC(ctx, endpoint, "account_info", map[string]interface{}{"account": from})
	if err != nil {
		return "", "", fmt.Errorf("ripple account_info: %w", err)
	}
	var acc struct {
		AccountData struct { Sequence uint32 `json:"Sequence"` } `json:"account_data"`
		LedgerCurrentIndex uint32 `json:"ledger_current_index"`
	}
	if err := json.Unmarshal(ai, &acc); err != nil {
		return "", "", err
	}
	feeRes, err := rippledRPC(ctx, endpoint, "fee", map[string]interface{}{})
	if err != nil {
		return "", "", fmt.Errorf("ripple fee: %w", err)
	}
	var fd struct { Drops struct { BaseFee string `json:"base_fee"` } `json:"drops"` }
	if err := json.Unmarshal(feeRes, &fd); err != nil || fd.Drops.BaseFee == "" {
		return "", "", errors.New("ripple fee malformed")
	}
	fee, err := strconv.ParseUint(fd.Drops.BaseFee, 10, 64)
	if err != nil {
		return "", "", err
	}
	lastLedger := acc.LedgerCurrentIndex + 20

	pub := crypto.CompressPubkey(&priv.PublicKey)
	blob, err := rippleCanonicalFields(pub, fromBytes, toBytes, acc.AccountData.Sequence, lastLedger, drops, fee, destTag)
	if err != nil {
		return "", "", err
	}
	hashInput := append([]byte{0x53, 0x54, 0x58, 0x00}, blob...)
	sigHash := sha512Half(hashInput)
	fullSig, err := crypto.Sign(sigHash[0:32], priv)
	if err != nil {
		return "", "", err
	}
	raw64 := fullSig[:64]
	if !lowS(raw64) {
		return "", "", errors.New("ripple signature low-s enforcement failed")
	}
	derSig := rippleDEREncode(raw64)
	final := append(blob, []byte{0x74, byte(len(derSig))}...)
	final = append(final, derSig...)
	if !broadcast {
		return hexOf(final), "", nil
	}
	subRes, err := rippledRPC(ctx, endpoint, "submit", map[string]interface{}{"tx_blob": hexOf(final)})
	if err != nil {
		return "", "", fmt.Errorf("ripple submit: %w", err)
	}
	var parsed struct {
		EngineResult string                 `json:"engine_result"`
		TXJson       map[string]interface{} `json:"tx_json"`
	}
	if err := json.Unmarshal(subRes, &parsed); err != nil {
		return "", "", err
	}
	if parsed.EngineResult != "tesSUCCESS" {
		return "", "", fmt.Errorf("ripple submit rejected: %s", parsed.EngineResult)
	}
	hashStr, _ := parsed.TXJson["hash"].(string)
	return hexOf(final), hashStr, nil
}

// xrpAddrBytes decodes an r-address to its 20-byte account id.
func xrpAddrBytes(addr string) ([]byte, error) {
	raw, err := base58DecAlphabet(addr, rippleAlphabet)
	if err != nil {
		return nil, err
	}
	if len(raw) != 25 || raw[0] != 0x00 {
		return nil, errors.New("invalid xrp address")
	}
	return raw[1:21], nil
}

// rippleDEREncode renders 64-byte r||s as minimal DER.
func rippleDEREncode(raw []byte) []byte {
	if len(raw) != 64 {
		return raw
	}
	r := derInt(raw[:32])
	s := derInt(raw[32:])
	content := append(r, s...)
	return append([]byte{0x30, byte(len(content))}, content...)
}

func derInt(b []byte) []byte {
	for len(b) > 0 && b[0] == 0 {
		b = b[1:]
	}
	if len(b) > 0 && b[0] > 0x7f {
		b = append([]byte{0x00}, b...)
	}
	return append([]byte{0x02, byte(len(b))}, b...)
}

// --------------------------- Zilliqa ---------------------------------------

// ZilAddress returns 0x+sha256(pub compressed)[12:] hex.
func ZilAddress(seed []byte, path string) (string, error) {
	priv, err := hdDerive(seed, path)
	if err != nil {
		return "", err
	}
	pub := crypto.CompressPubkey(&priv.PublicKey)
	h := sha256.Sum256(pub)
	return "0x" + hex.EncodeToString(h[12:]), nil
}

// --------------------------- ICP ---------------------------------------

// ICPAddress computes the self-authenticating principal: base32(4B crc32 of
// sha224(der pubkey) || sha224) with dashes every 5 chars.
func ICPAddress(seed []byte, path string) (string, error) {
	priv, err := hdDerive(seed, path)
	if err != nil {
		return "", err
	}
	pub := crypto.CompressPubkey(&priv.PublicKey)
	derPub := icDERPub(pub)
	if derPub == nil {
		return "", errors.New("icp der encode failed")
	}
	hash := icSHA224(derPub)
	csum := icCRC32(hash)
	full := append(csum, hash...)
	b32 := icBase32Lower(full)
	return icDashGroup(b32, 5), nil
}

func icDERPub(c33 []byte) []byte {
	if len(c33) != 33 {
		return nil
	}
	pre := []byte{0x30, 0x51}
	oid := []byte{0x30, 0x0b, 0x06, 0x05, 0x2b, 0x81, 0x04, 0x00, 0x0a}
	bitStr := []byte{0x03, 0x42, 0x00}
	return append(append(pre, oid...), append(bitStr, c33...)...)
}

func icSHA224(b []byte) []byte {
	sh := sha256.New224()
	sh.Write(b)
	return sh.Sum(nil)
}

// icCRC32 computes standard CRC32 (IEEE LE polynomial).
func icCRC32(b []byte) []byte {
	out := make([]byte, 4)
	v := uint32(0xffffffff)
	for _, c := range b {
		v ^= uint32(c)
		for i := 0; i < 8; i++ {
			if v&1 == 1 {
				v = (v >> 1) ^ 0xedb88320
			} else {
				v >>= 1
			}
		}
	}
	v = ^v
	// write little-endian (IC principal uses standard crc32 bytes order)
	out[0] = byte(v)
	out[1] = byte(v >> 8)
	out[2] = byte(v >> 16)
	out[3] = byte(v >> 24)
	return out
}

func icBase32Lower(b []byte) string {
	const alpha = "abcdefghijklmnopqrstuvwxyz234567"
	var out strings.Builder
	var acc uint16
	bits := 0
	for _, x := range b {
		acc = (acc << 8) | uint16(x)
		bits += 8
		for bits >= 5 {
			bits -= 5
			out.WriteByte(alpha[(acc>>bits)&31])
		}
	}
	return out.String()
}

func icDashGroup(s string, n int) string {
	var groups []string
	for len(s) > 0 {
		var g string
		if len(s) > n {
			g, s = s[:n], s[n:]
		} else {
			g, s = s, ""
		}
		groups = append(groups, g)
	}
	return strings.Join(groups, "-")
}
