package main

// non_evm_algorand.go — Algorand SDK: real address derivation (base32 no-
// padding with sha512-256 checksum), message signing, and MSGPACK pay-tx
// build plus algod /v2/transactions broadcast. Fails closed on node errors.

import (
	"context"
	"crypto/sha512"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	ed "golang.org/x/crypto/ed25519"
)

// AlgoAddress computes base32(pub || sha512_256(pub)[28:]) with no padding.
func AlgoAddress(seed []byte, path string) (string, error) {
	pub, err := edPubKey(seed, path)
	if err != nil {
		return "", err
	}
	sum := sha512.Sum512_256(pub)
	body := append(append([]byte{}, pub...), sum[28:]...) // 4-byte checksum
	return b32NoPad(body), nil
}

func b32NoPad(b []byte) string {
	const alpha = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
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

func b32NoPadDecode(s string) ([]byte, error) {
	const alpha = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
	var out []byte
	var acc uint32
	bits := 0
	for _, c := range []byte(strings.ToUpper(s)) {
		idx := strings.IndexByte(alpha, c)
		if idx < 0 {
			return nil, fmt.Errorf("base32: invalid char %q", c)
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

// msgpackBuf encodes the small msgpack subset Algorand needs.
type mpiBuf struct{ b []byte }

func (m *mpiBuf) raw(b []byte) { m.b = append(m.b, b...) }
func (m *mpiBuf) str(s string) {
	if len(s) <= 31 {
		m.raw([]byte{0xa0 | byte(len(s))})
	} else {
		m.raw([]byte{0xd9, byte(len(s))})
	}
	m.raw([]byte(s))
}
func (m *mpiBuf) bin(b []byte) {
	if len(b) <= 255 {
		m.raw([]byte{0xc4, byte(len(b))})
	} else {
		m.raw([]byte{0xc5, byte(len(b) >> 8), byte(len(b))})
	}
	m.raw(b)
}
func (m *mpiBuf) u64(v uint64) {
	if v <= 0x7f {
		m.raw([]byte{byte(v)})
	} else {
		m.raw([]byte{0xcf})
		for i := 0; i < 8; i++ {
			m.raw([]byte{byte(v >> (56 - i))})
		}
	}
}
func (m *mpiBuf) mapHeader(n int) { m.raw([]byte{0x80 | byte(n)}) }

// AlgoBuildRequest describes a pay transaction.
type AlgoBuildRequest struct {
	GenesisHashB64 string
	Fee            uint64
	FirstValid     uint64
	LastValid      uint64
	Amount         uint64
}

// AlgoBuildSend builds a signed 'pay' STX message. Broadcast attempts
// algod /v2/transactions; requires ALGOD_TOKEN on tokend nodes (fail-closed).
func AlgoBuildSend(ctx context.Context, seed []byte, path, endpoint, algodToken, to string, req AlgoBuildRequest, broadcast bool) (string, string, error) {
	if req.FirstValid == 0 || req.LastValid == 0 || req.GenesisHashB64 == "" {
		return "", "", errors.New("algorand needs genesis hash + first/last valid rounds")
	}
	priv, pub, err := edKeypair(seed, path)
	if err != nil {
		return "", "", err
	}
	toDecoded, err := b32NoPadDecode(to)
	if err != nil || len(toDecoded) < 32 {
		return "", "", errors.New("invalid algorand destination address")
	}
	gh, err := base64Decode(req.GenesisHashB64)
	if err != nil || len(gh) != 32 {
		return "", "", errors.New("algorand genesis hash must be 32 bytes (base64)")
	}
	txn := &mpiBuf{}
	txn.mapHeader(8)
	txn.str("amt");  txn.u64(req.Amount)
	txn.str("fee");  txn.u64(req.Fee)
	txn.str("fv");   txn.u64(req.FirstValid)
	txn.str("gh");   txn.bin(gh)
	txn.str("lv");   txn.u64(req.LastValid)
	txn.str("rcv");  txn.bin(toDecoded[:32])
	txn.str("snd");  txn.bin([]byte(pub))
	txn.str("type"); txn.str("pay")

	full := append([]byte("TX"), txn.b...)
	sig := ed.Sign(priv, full)

	envelope := &mpiBuf{}
	envelope.mapHeader(2)
	envelope.str("sig")
	envelope.bin(sig)
	envelope.str("txn")
	envelope.raw(txn.b)
	if !broadcast {
		return hexOf(envelope.b), "", nil
	}
	url := strings.TrimRight(endpoint, "/") + "/v2/transactions"
	hdr := http.Header{"Content-Type": []string{"application/x-binary"}}
	if algodToken != "" {
		hdr.Set("X-Algo-API-Token", algodToken)
	}
	resp, err := postRawHdr(ctx, url, hdr, envelope.b)
	if err != nil {
		return "", "", fmt.Errorf("algorand broadcast: %w", err)
	}
	var parsed struct {
		TxId string `json:"txId"`
	}
	if err := json.Unmarshal(resp, &parsed); err != nil || parsed.TxId == "" {
		return "", "", errors.New("algorand broadcast: malformed reply")
	}
	return hexOf(envelope.b), parsed.TxId, nil
}

// postRawHdr POSTs raw bytes with an explicit header.
func postRawHdr(ctx context.Context, url string, hdr http.Header, body []byte) ([]byte, error) {
	req, err := newRequestBytes(ctx, url, body)
	if err != nil {
		return nil, err
	}
	for k, v := range hdr {
		req.Header[k] = v
	}
	return doRead(req)
}
