package main

// non_evm_aptos.go — Aptos SDK: real BCS RawTransaction for coin::transfer,
// signed with Ed25519, and broadcast via the fullnode REST API. SHA3-256
// (not legacy Keccak) is used exactly as the BCS raw-tx signing prefix spec
// requires.

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/crypto/sha3"
	ed "golang.org/x/crypto/ed25519"
)

// ------ BCS encoding helpers ------

type bcsBuf struct{ b []byte }

func (b *bcsBuf) uleb(v uint64) { b.b = append(b.b, encodeUvarint(v)...) }
func (b *bcsBuf) u8(v byte)     { b.b = append(b.b, v) }
func (b *bcsBuf) raw(by []byte) { b.b = append(b.b, by...) }
func (b *bcsBuf) u64(v uint64) {
	var t [8]byte
	// 8 bytes little-endian by hand to avoid extra imports
	for i := 0; i < 8; i++ {
		t[i] = byte(v >> (8 * uint(i)))
	}
	b.raw(t[:])
}
func (b *bcsBuf) u128BytesLe(v uint64) { b.u64(v) } // amounts fit in u64 here
func (b *bcsBuf) seq(by []byte)        { b.uleb(uint64(len(by))); b.raw(by) }
func (b *bcsBuf) str(s string)         { b.seq([]byte(s)) }
func (b *bcsBuf) address(a []byte)     { b.raw(a) }

// AptosAddress derives 0x || SHA3-256(pub || 0x00) (single-signature
// authenticator variant byte 0x00, as per the Aptos address spec).
func AptosAddress(seed []byte, path string) (string, error) {
	_, pub, err := edKeypair(seed, path)
	if err != nil {
		return "", err
	}
	raw := append(append([]byte{}, pub...), 0x00)
	sum := sha3.Sum256(raw)
	return "0x" + hex.EncodeToString(sum[:]), nil
}

// aptosBuildTransferBCS builds the BCS RawTransaction for 0x1::coin::transfer.
func aptosBuildTransferBCS(senderAddressHex, toAddressHex string, sequence, maxGas, gasPrice, expiration uint64, chainID byte, amount uint64) ([]byte, []byte, error) {
	sender, err := hex.DecodeString(strip0x(senderAddressHex))
	if err != nil || len(sender) != 32 {
		return nil, nil, errors.New("aptos sender address must be 32 bytes hex")
	}
	to, err := hex.DecodeString(strip0x(toAddressHex))
	if err != nil || len(to) != 32 {
		return nil, nil, errors.New("aptos destination must be 32 bytes hex")
	}

	// EntryFunction payload: module 0x1::coin, function transfer,
	// no type args, args = [receiver(u8 bytes), amount(u64)]
	payload := &bcsBuf{}
	payload.u8(2) // variant: EntryFunction
	module := hexDecodeFixed("1")
	payload.address(module)          // module address
	payload.str("coin")              // module name
	payload.str("transfer")          // function name
	payload.uleb(0)                  // no type args
	payload.uleb(2)                  // 2 args
	payload.seq(to)                  // arg 0: receiver address as bytes
	amt := &bcsBuf{}                 // arg 1: amount as BCS-u64 bytes
	amt.u64(amount)
	payload.seq(amt.b)

	rawTx := &bcsBuf{}
	rawTx.address(sender)
	rawTx.u64(sequence)
	rawTx.raw(payload.b)
	rawTx.u64(maxGas)
	rawTx.u64(gasPrice)
	rawTx.u64(expiration)
	rawTx.u8(chainID)
	return rawTx.b, to, nil
}

func strip0x(s string) string {
	if len(s) >= 2 && s[:2] == "0x" {
		return s[2:]
	}
	return s
}

func hexDecodeFixed(s string) []byte {
	if len(s)%2 == 1 {
		s = "0" + s
	}
	b, _ := hex.DecodeString(s)
	return b
}

// AptosBuildSend builds + signs the BCS raw transaction; broadcasts when
// requested through the fullnode REST endpoint (fallible → fail closed).
func AptosBuildSend(ctx context.Context, seed []byte, path, endpoint, to string, amount, sequence, maxGas, gasPrice, expiration uint64, chainID byte, broadcast bool) (signedJSON, txHash string, err error) {
	priv, pub, err := edKeypair(seed, path)
	if err != nil {
		return "", "", err
	}
	from, err := AptosAddress(seed, path)
	if err != nil {
		return "", "", err
	}
	rawBCS, _, err := aptosBuildTransferBCS(from, to, sequence, maxGas, gasPrice, expiration, chainID, amount)
	if err != nil {
		return "", "", err
	}
	// Signed message = sha3_256("APTOS::RawTransaction") || rawTxnBCS
	prefix := sha3.Sum256([]byte("APTOS::RawTransaction"))
	toSign := append(prefix[:], rawBCS...)
	sig := ed.Sign(priv, toSign)

	payload := map[string]interface{}{
		"sender":            from,
		"sequence_number":   numToString(sequence),
		"max_gas_amount":    numToString(maxGas),
		"gas_unit_price":    numToString(gasPrice),
		"expiration_timestamp_secs": numToString(expiration),
		"payload": map[string]interface{}{
			"type":           "entry_function_payload",
			"function":       "0x1::coin::transfer",
			"type_arguments": []interface{}{},
			"arguments": []interface{}{
				to,
				numToString(amount),
			},
		},
		"signature": map[string]interface{}{
			"type":       "ed25519_signature",
			"public_key": hex0x(pub),
			"signature":  hex0x(sig),
		},
	}
	raw, _ := json.Marshal(payload)
	if !broadcast {
		return string(raw), "", nil
	}
	url := strings_TrimRight(endpoint, "/") + "/v1/transactions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := solanaHTTP.Do(req) // shared bounded client
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusCreated {
		return "", "", fmt.Errorf("aptos broadcast: HTTP %d: %s", resp.StatusCode, string(body[:minInt(len(body), 300)]))
	}
	var parsed struct {
		Hash string `json:"hash"`
	}
	if uerr := json.Unmarshal(body, &parsed); uerr != nil || parsed.Hash == "" {
		return "", "", errors.New("aptos broadcast: malformed reply")
	}
	return string(raw), parsed.Hash, nil
}

func numToString(v uint64) string { return u64ToStr(v) }

func u64ToStr(v uint64) string {
	if v == 0 {
		return "0"
	}
	s := ""
	for v > 0 {
		s = string(rune('0'+v%10)) + s
		v /= 10
	}
	return s
}

func strings_TrimRight(s, cut string) string {
	for len(s) > 0 && strings_EndsWith(s, cut) {
		s = s[:len(s)-len(cut)]
	}
	return s
}

func strings_EndsWith(s, suf string) bool {
	return len(s) >= len(suf) && s[len(s)-len(suf):] == suf
}
