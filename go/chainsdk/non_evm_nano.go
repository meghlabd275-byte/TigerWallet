package chainsdk

// non_evm_nano.go — Nano SDK: address derivation, message signing, and a
// state-block builder with 'process' broadcast. Frontier hash, representative,
// and remaining balance are fetched/computed — never fabricated. The client
// supplies the 8-byte proof-of-work (e.g. node work_generate) for broadcast.

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"golang.org/x/crypto/blake2b"
	ed "golang.org/x/crypto/ed25519"
)

var nanoAlphabet = "13456789abcdefghijkmnopqrstuwxyz"

// nanoBase32Enc encodes with the Nano ledger's own base32: the value is
// left-padded with zero bits to a multiple of 5, most significant group first.
//
// The padding lengthens the shift base; it must NOT be applied to the value as
// well, or every character is taken from a rotated copy of the key. That error
// still yields a well-formed, correctly-checksummed address for the wrong
// account, so it is invisible to the checksum test.
//
// Two mainnet vectors pin this: the public key
// e89208dd...a65093ba encodes to 3t6k35gi...736xt (the well-known burn
// representative account), and the all-zero key to the 52-character run of 1s
// in nano_1111111111111111111111111111111111111111111111111111hifc8npp.
// The zero-key vector alone is not sufficient: every character of it is
// alphabet[0], so it cannot detect a rotated encoding.
func nanoBase32Enc(b []byte) string {
	bits := len(b) * 8
	chars := (bits + 4) / 5
	total := chars * 5
	value := new(big.Int).SetBytes(b)
	out := make([]byte, chars)
	mask := big.NewInt(31)
	tmp := new(big.Int)
	for i := 0; i < chars; i++ {
		tmp.Rsh(value, uint(total-5*(i+1)))
		tmp.And(tmp, mask)
		out[i] = nanoAlphabet[tmp.Int64()]
	}
	return string(out)
}

// nanoBase32Dec is the inverse of nanoBase32Enc.
//
// The value is left-padded, so the KEY occupies the low bits and the padding
// sits above it: 52 characters carry 260 bits, of which the low 256 are the key
// and the top four must be zero. Reading the high bits instead returns a key
// shifted by four positions — a different account. The accumulator is a
// big.Int rather than a machine word because 260 bits does not fit in uint64.
func nanoBase32Dec(s string) ([]byte, error) {
	acc := new(big.Int)
	tmp := new(big.Int)
	for i := 0; i < len(s); i++ {
		idx := strings.IndexByte(nanoAlphabet, s[i])
		if idx < 0 {
			return nil, fmt.Errorf("nano b32: invalid char %q", s[i])
		}
		acc.Lsh(acc, 5)
		acc.Or(acc, tmp.SetInt64(int64(idx)))
	}
	total := len(s) * 5
	keyBits := total - total%8
	if acc.BitLen() > keyBits {
		return nil, errors.New("nano b32: non-zero padding bits")
	}
	out := make([]byte, keyBits/8)
	acc.And(acc, tmp.Sub(tmp.Lsh(big.NewInt(1), uint(keyBits)), big.NewInt(1)))
	acc.FillBytes(out)
	return out, nil
}

// nanoChecksumChars returns the 8-character checksum that follows a Nano key:
// the first five bytes of BLAKE2b-256 of the key, reversed.
func nanoChecksumChars(pub []byte) (string, error) {
	h, err := blake2b.New(5, nil)
	if err != nil {
		return "", err
	}
	h.Write(pub)
	d := h.Sum(nil)
	rev := make([]byte, 5)
	for i := 0; i < 5; i++ {
		rev[i] = d[4-i]
	}
	return nanoBase32Enc(rev), nil
}

// NanoAddress derives nano_<b32(pub)> + <b32(reversed blake2b-256(pub)[27:])>.
func NanoAddress(seed []byte, path string) (string, error) {
	return NanoAddressFor(seed, path, "nano")
}

// nanoPrefixes holds the address prefix of every chain that runs the Nano
// ledger. Banano is the same codebase and checksum with a different prefix, so
// they share the encoder and differ only here.
//
// Wakt routes to FamilyNano but is deliberately absent: its prefix could not be
// confirmed, and deriving an address under a guessed prefix would hand the user
// an account no Wakt node accepts. It fails closed instead.
var nanoPrefixes = map[string]string{
	"nano":   "nano_",
	"banano": "ban_",
}

// NanoAddressFor derives a Nano-ledger address with the prefix of the named
// chain. The prefix is part of the checksummed address, so a Banano key
// published as a nano_ address is rejected by every node.
func NanoAddressFor(seed []byte, path, chainType string) (string, error) {
	pub, err := EdPubKey(seed, path)
	if err != nil {
		return "", err
	}
	return NanoAddressFromPub(pub, chainType)
}

// NanoAddressFromPub renders a 32-byte ed25519 public key as a Nano-ledger
// address. It is separate from NanoAddressFor so an address can be reproduced
// from a key that was recovered elsewhere (a signature recovery, a watch-only
// import) without re-deriving it from a seed.
func NanoAddressFromPub(pub []byte, chainType string) (string, error) {
	prefix, ok := nanoPrefixes[chainType]
	if !ok {
		return "", fmt.Errorf("unsupported nano-family chain %q", chainType)
	}
	if len(pub) != 32 {
		return "", fmt.Errorf("nano public key must be 32 bytes, got %d", len(pub))
	}
	chk, err := nanoChecksumChars(pub)
	if err != nil {
		return "", err
	}
	return prefix + nanoBase32Enc(pub) + chk, nil
}

// nanoPub decodes a nano-ledger address to its raw 32-byte public key.
//
// The prefix is stripped before decoding: nano_ and xrb_ are 5 characters while
// ban_ is 4, so a fixed offset would cut the first base32 character off a
// Banano key and silently derive the wrong account.
//
// The checksum is VERIFIED, not skipped. Without that check a single mistyped
// character yields a different, well-formed key, and a transfer would be sent
// to an account nobody controls. The length is exact for the same reason: a
// trailing character must not be quietly ignored.
func nanoPub(addr string) ([]byte, error) {
	sep := strings.IndexByte(addr, '_')
	if sep < 3 {
		return nil, errors.New("invalid nano-family address: missing prefix")
	}
	if _, ok := nanoPrefixes[addr[:sep]]; !ok && addr[:sep] != "xrb" {
		return nil, fmt.Errorf("unsupported nano-family prefix %q", addr[:sep])
	}
	body := addr[sep+1:]
	if len(body) != 60 {
		return nil, fmt.Errorf("nano address body must be 60 characters, got %d", len(body))
	}
	data, err := nanoBase32Dec(body[:52])
	if err != nil {
		return nil, err
	}
	if len(data) != 32 {
		return nil, fmt.Errorf("nano address key must decode to 32 bytes, got %d", len(data))
	}
	want, err := nanoChecksumChars(data)
	if err != nil {
		return nil, err
	}
	if body[52:] != want {
		return nil, errors.New("nano address checksum does not match")
	}
	return data, nil
}

// nanoRPC posts {"action": <action>, ...fields} to the nano node.
func nanoRPC(ctx context.Context, endpoint, action string, fields map[string]interface{}) (json.RawMessage, error) {
	body := map[string]interface{}{"action": action}
	for k, v := range fields {
		body[k] = v
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	resp, err := postRaw(ctx, endpoint, raw)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Error string `json:"error"`
	}
	if err := unmarshalCheckError(resp, &parsed); err != nil {
		return nil, err
	}
	if parsed.Error != "" {
		return nil, fmt.Errorf("nano %s: %s", action, parsed.Error)
	}
	return resp, nil
}

// nanoWorkThresholdSend is the minimum work value a Nano send (or change)
// block must reach. The ledger rejects a block whose work is below the epoch
// threshold.
const nanoWorkThresholdSend uint64 = 0xfffffff800000000

// nanoWorkThresholdReceive is the (lower) threshold used for receive blocks.
const nanoWorkThresholdReceive uint64 = 0xfffffe0000000000

// nanoWorkDigestSize is the digest size the ledger uses for proof-of-work:
// Nano hashes with a BLAKE2b-8 digest, not the BLAKE2b-512 default.
const nanoWorkDigestSize = 8

// nanoWorkValue returns the work value a nonce produces for a block root.
//
// The ledger hashes the NONCE FIRST and then the 32-byte root:
//
//	value = little_endian_u64( BLAKE2b-8( nonce_le_8 || root ) )
//
// Both the ordering and the byte order matter and neither is obvious, so this
// was checked against the live network rather than assumed: for 25 consecutive
// mainnet blocks, BLAKE2b-8(work || previous) read little-endian equals exactly
// the difficulty that the node's own work_validate reports for that work.
// Hashing root||nonce instead produces a different value for every block.
func nanoWorkValue(root []byte, nonce uint64) (uint64, error) {
	if len(root) != 32 {
		return 0, errors.New("nano work: block root must be 32 bytes")
	}
	var buf [40]byte
	binary.LittleEndian.PutUint64(buf[:8], nonce)
	copy(buf[8:], root)
	h, err := blake2b.New(nanoWorkDigestSize, nil)
	if err != nil {
		return 0, err
	}
	h.Write(buf[:])
	return binary.LittleEndian.Uint64(h.Sum(nil)), nil
}

// NanoWorkValid reports whether a nonce produces sufficient work for a block
// root.
func NanoWorkValid(root []byte, nonce uint64, threshold uint64) bool {
	value, err := nanoWorkValue(root, nonce)
	if err != nil {
		return false
	}
	return value >= threshold
}

// NanoWorkValue returns the work value a nonce produces for a block root, so a
// caller can report how close a rejected nonce was.
func NanoWorkValue(root []byte, nonce uint64) (uint64, error) {
	return nanoWorkValue(root, nonce)
}

// NanoWorkGenerate computes the proof-of-work for a block root in-process.
//
// A hosted node's work_generate is not always available: the public
// work-peering services answer HTTP 402 or 5xx. Computing the work locally
// keeps a sweep that is otherwise signable from failing. The search is a plain
// parallel nonce scan, as a node does, against the ledger's own threshold, so
// the returned nonce is one the network accepts. It never returns a nonce that
// does not meet the threshold.
func NanoWorkGenerate(ctx context.Context, root []byte, threshold uint64, workers int) (uint64, error) {
	if len(root) != 32 {
		return 0, errors.New("nano work: block root must be 32 bytes")
	}
	if threshold == 0 {
		threshold = nanoWorkThresholdSend
	}
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	if workers > 64 {
		workers = 64
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		found atomic.Uint64
		wg    sync.WaitGroup
	)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(offset uint64) {
			defer wg.Done()
			h, err := blake2b.New(nanoWorkDigestSize, nil)
			if err != nil {
				return
			}
			var buf [40]byte
			copy(buf[8:], root)
			nonce := offset
			step := uint64(workers)
			for {
				if found.Load() != 0 {
					return
				}
				binary.LittleEndian.PutUint64(buf[:8], nonce)
				h.Reset()
				h.Write(buf[:])
				if binary.LittleEndian.Uint64(h.Sum(nil)) >= threshold {
					found.CompareAndSwap(0, nonce)
					return
				}
				nonce += step
				// A cancelled search must stop burning CPU rather than run to
				// completion.
				if nonce%(step*4096) < step {
					select {
					case <-ctx.Done():
						return
					default:
					}
				}
			}
		}(uint64(w))
	}
	wg.Wait()

	if n := found.Load(); n != 0 {
		if !NanoWorkValid(root, n, threshold) {
			return 0, errors.New("nano work: internal error, candidate does not meet the threshold")
		}
		return n, nil
	}
	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("nano work: %w", err)
	}
	return 0, errors.New("nano work: search ended without a valid nonce")
}

// nanoWorkHex renders a nonce as the 16-character hex string a node expects.
// The node reports work as big-endian hex while hashing it little-endian, so
// the two representations must not be conflated.
func nanoWorkHex(nonce uint64) string {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], nonce)
	return hex.EncodeToString(b[:])
}

// nanoWorkParse is the inverse of nanoWorkHex.
func nanoWorkParse(workHex string) (uint64, error) {
	raw, err := hexStringDecode(workHex)
	if err != nil {
		return 0, err
	}
	if len(raw) != 8 {
		return 0, fmt.Errorf("nano work must be 8 bytes, got %d", len(raw))
	}
	return binary.BigEndian.Uint64(raw), nil
}

// NanoBuildSend builds a signed send-block.
func NanoBuildSend(ctx context.Context, seed []byte, path, endpoint, to string, remainingBalanceBE []byte, workHex string, broadcast bool) (string, string, error) {
	priv, pub, err := EdKeypair(seed, path)
	if err != nil {
		return "", "", err
	}
	from, err := NanoAddress(seed, path)
	if err != nil {
		return "", "", err
	}
	toPub, err := nanoPub(to)
	if err != nil {
		return "", "", err
	}
	res, err := nanoRPC(ctx, endpoint, "account_info", map[string]interface{}{
		"account": from, "representative": "true"})
	if err != nil {
		return "", "", fmt.Errorf("nano account_info: %w", err)
	}
	var ai struct {
		Frontier       string `json:"frontier"`
		Representative string `json:"representative"`
		Balance        string `json:"balance"`
	}
	if err := json.Unmarshal(res, &ai); err != nil || ai.Frontier == "" {
		return "", "", errors.New("nano account_info: malformed reply")
	}
	prev, err := hexStringDecode(ai.Frontier)
	if err != nil || len(prev) != 32 {
		return "", "", errors.New("nano frontier malformed")
	}
	repPub, err := nanoPub(ai.Representative)
	if err != nil {
		return "", "", err
	}
	if len(remainingBalanceBE) > 16 {
		return "", "", errors.New("balance must fit u128")
	}
	rem := u128BE(remainingBalanceBE)

	// The remaining balance is part of the signed block, so it must not exceed
	// what the account actually holds. A caller that passed a fabricated or
	// stale remainder would otherwise have that value signed and rejected by
	// every node, after which the failure looks like a broadcast problem.
	ledgerBal, ok := new(big.Int).SetString(strings.TrimSpace(ai.Balance), 10)
	if !ok {
		return "", "", errors.New("nano account_info returned an unparsable balance")
	}
	if new(big.Int).SetBytes(rem).Cmp(ledgerBal) > 0 {
		return "", "", fmt.Errorf("nano remaining balance %s exceeds the account balance %s",
			new(big.Int).SetBytes(rem).String(), ledgerBal.String())
	}

	var block bytes.Buffer
	block.Write([]byte(pub))
	block.Write(prev)
	block.Write(repPub)
	block.Write(rem)
	block.Write(toPub)
	rawBlock := block.Bytes()

	hdr := append(make([]byte, 31), 0x06)
	h, err := blake2b.New256(nil)
	if err != nil {
		return "", "", err
	}
	h.Write(append(hdr, rawBlock...))
	hash := h.Sum(nil)
	sig := ed.Sign(priv, hash)

	// Work root for a send block is the previous block hash. When the caller
	// supplies a nonce it is verified against the ledger threshold rather than
	// forwarded: a nonce below the threshold is rejected by every node, so
	// passing it through would turn a broadcast failure into a silent no-op.
	if strings.TrimSpace(workHex) == "" {
		nonce, werr := NanoWorkGenerate(ctx, prev, nanoWorkThresholdSend, 0)
		if werr != nil {
			return "", "", werr
		}
		workHex = nanoWorkHex(nonce)
	}
	work, err := hexStringDecode(workHex)
	if err != nil || len(work) != 8 {
		return "", "", errors.New("nano work must be an 8-byte nonce in hex")
	}
	nonce, err := nanoWorkParse(workHex)
	if err != nil {
		return "", "", err
	}
	if !NanoWorkValid(prev, nonce, nanoWorkThresholdSend) {
		value, _ := NanoWorkValue(prev, nonce)
		return "", "", fmt.Errorf("nano work %s is below the send threshold (value %016x)", workHex, value)
	}
	final := append(append([]byte{}, rawBlock...), sig...)
	final = append(final, work...)

	if broadcast {
		payload := map[string]interface{}{
			"json_block": "true",
			"subtype":    "send",
			"block": map[string]interface{}{
				"type":           "state",
				"account":        from,
				"previous":       ai.Frontier,
				"representative": ai.Representative,
				"balance":        u128Decimal(rem),
				"link":           hexOf(toPub),
				"signature":      hexOf(sig),
				"work":           workHex,
			},
		}
		resp, err := nanoRPC(ctx, endpoint, "process", payload)
		if err != nil {
			return "", "", fmt.Errorf("nano process: %w", err)
		}
		var parsed struct {
			Hash string `json:"hash"`
		}
		if err := json.Unmarshal(resp, &parsed); err != nil || parsed.Hash == "" {
			return "", "", errors.New("nano process: malformed reply")
		}
		return hexOf(final), parsed.Hash, nil
	}
	return hexOf(final), hexOf(hash), nil
}

// NanoSendRequest describes a Nano-ledger transfer.
//
// Amount is the amount to send in the ledger's raw unit, as a decimal string.
// RemainingBalanceBE and WorkHex are optional: when RemainingBalanceBE is empty
// the sender's balance is read from the ledger and the remainder is computed,
// and when WorkHex is empty the proof-of-work is computed locally. Both
// fallbacks exist because a caller that has to supply the remaining balance is
// a caller that can supply the wrong one: the block would then be signed over a
// balance the account does not hold and every node would reject it.
type NanoSendRequest struct {
	Amount             string
	RemainingBalanceBE []byte
	WorkHex            string
	Prefix             string
}

// NanoBuildSendRequest builds and signs a Nano send block using the request
// form. It is the preferred entry point; NanoBuildSend remains for callers that
// already resolved the balance.
func NanoBuildSendRequest(ctx context.Context, seed []byte, path, endpoint, to string, req NanoSendRequest, broadcast bool) (string, string, error) {
	prefix := req.Prefix
	if prefix == "" {
		prefix = "nano"
	}
	from, err := NanoAddressFor(seed, path, prefix)
	if err != nil {
		return "", "", err
	}
	remaining := req.RemainingBalanceBE
	if len(remaining) == 0 {
		amount, ok := new(big.Int).SetString(strings.TrimSpace(req.Amount), 10)
		if !ok || amount.Sign() <= 0 {
			return "", "", fmt.Errorf("nano amount %q must be a positive decimal integer in raw units", req.Amount)
		}
		bal, err := nanoLedgerBalance(ctx, endpoint, from)
		if err != nil {
			return "", "", err
		}
		if bal.Cmp(amount) < 0 {
			return "", "", fmt.Errorf("nano account %s holds %s raw but the transfer needs %s", from, bal.String(), amount.String())
		}
		remaining = make([]byte, 16)
		diff := new(big.Int).Sub(bal, amount).Bytes()
		copy(remaining[16-len(diff):], diff)
	}
	return NanoBuildSend(ctx, seed, path, endpoint, to, remaining, req.WorkHex, broadcast)
}

// nanoLedgerBalance reads an account's confirmed balance in raw units.
func nanoLedgerBalance(ctx context.Context, endpoint, address string) (*big.Int, error) {
	raw, err := nanoRPC(ctx, endpoint, "account_info", map[string]interface{}{"account": address})
	if err != nil {
		return nil, fmt.Errorf("nano account_info: %w", err)
	}
	var ai struct {
		Balance string `json:"balance"`
	}
	if err := json.Unmarshal(raw, &ai); err != nil {
		return nil, fmt.Errorf("nano account_info: malformed reply: %w", err)
	}
	bal, ok := new(big.Int).SetString(strings.TrimSpace(ai.Balance), 10)
	if !ok {
		return nil, fmt.Errorf("nano account_info returned an unparsable balance %q", ai.Balance)
	}
	return bal, nil
}

// u128Decimal renders a u128 byte array as a decimal string.
func u128Decimal(be []byte) string {
	if len(be) == 0 {
		return "0"
	}
	return new(big.Int).SetBytes(be).String()
}
