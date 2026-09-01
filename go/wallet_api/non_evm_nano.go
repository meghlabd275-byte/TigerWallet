package main

// non_evm_nano.go — Nano SDK: address derivation, message signing, and a
// state-block builder with 'process' broadcast. Frontier hash, representative,
// and remaining balance are fetched/computed — never fabricated. The client
// supplies the 8-byte proof-of-work (e.g. node work_generate) for broadcast.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"golang.org/x/crypto/blake2b"
	ed "golang.org/x/crypto/ed25519"
)

var nanoAlphabet = "13456789abcdefghijkmnopqrstuwxyz"

func nanoBase32Enc(b []byte) string {
	var acc uint32
	bits := 0
	var out strings.Builder
	for _, x := range b {
		acc = (acc << 8) | uint32(x)
		bits += 8
		for bits >= 5 {
			bits -= 5
			out.WriteByte(nanoAlphabet[(acc>>bits)&31])
		}
	}
	return out.String()
}

func nanoBase32Dec(s string) ([]byte, error) {
	var acc uint32
	bits := 0
	var out []byte
	for _, c := range []byte(s) {
		idx := strings.IndexByte(nanoAlphabet, c)
		if idx < 0 {
			return nil, fmt.Errorf("nano b32: invalid char %q", c)
		}
		acc = (acc << 5) | uint32(idx)
		bits += 5
		for bits >= 8 {
			bits -= 8
			out = append(out, byte((acc>>bits)&0xff))
		}
	}
	return out, nil
}

// NanoAddress derives nano_<b32(pub)> + <b32(reversed blake2b-256(pub)[27:])>.
func NanoAddress(seed []byte, path string) (string, error) {
	pub, err := edPubKey(seed, path)
	if err != nil {
		return "", err
	}
	h, err := blake2b.New256(nil)
	if err != nil {
		return "", err
	}
	h.Write(pub)
	d := h.Sum(nil)
	rev := make([]byte, 5)
	for i := 0; i < 5; i++ {
		rev[i] = d[31-i]
	}
	return "nano_" + nanoBase32Enc(pub) + nanoBase32Enc(rev), nil
}

// nanoPub decodes a nano address to its raw 32-byte public key.
func nanoPub(addr string) ([]byte, error) {
	if !(strings.HasPrefix(addr, "nano_") || strings.HasPrefix(addr, "xrb_")) || len(addr) < 60 {
		return nil, errors.New("invalid nano address")
	}
	body := addr[5:]
	data, err := nanoBase32Dec(body[:52])
	if len(data) < 32 {
		return nil, err
	}
	return data[:32], err
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
		Error string          `json:"error"`
	}
	if err := unmarshalCheckError(resp, &parsed); err != nil {
		return nil, err
	}
	if parsed.Error != "" {
		return nil, fmt.Errorf("nano %s: %s", action, parsed.Error)
	}
	return resp, nil
}

// NanoBuildSend builds a signed send-block.
func NanoBuildSend(ctx context.Context, seed []byte, path, endpoint, to string, remainingBalanceBE []byte, workHex string, broadcast bool) (string, string, error) {
	priv, pub, err := edKeypair(seed, path)
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

	work, err := hexStringDecode(workHex)
	if err != nil || len(work) != 8 {
		return "", "", errors.New("nano requires an 8-byte client proof-of-work (e.g. from node's work_generate)")
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

// u128Decimal renders a u128 byte array as a decimal string.
func u128Decimal(be []byte) string {
	if len(be) == 0 {
		return "0"
	}
	return new(big.Int).SetBytes(be).String()
}
