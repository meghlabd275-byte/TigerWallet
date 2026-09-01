package main

// non_evm_solana.go — Solana SDK: full SystemProgram.transfer transaction
// build, Ed25519 signature, and broadcast via the node's JSON-RPC. Recent
// blockhash is fetched from the node; if it is unreachable the call fails
// closed (we never sign a blockhash we didn't get from the network).

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	ed "golang.org/x/crypto/ed25519"
)

var solanaHTTP = &http.Client{Timeout: 25 * time.Second}

// solanaAccountFamily — getFromSeed returns address + sign message.
func SolanaAddressFromKey(pub []byte) (string, error) {
	if len(pub) != 32 {
		return "", errors.New("solana pubkey must be 32 bytes")
	}
	return base58Encode(pub), nil
}

// solanaRPCResp is the envelope for solana JSON-RPC replies.
type solanaRPCResp struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// solanaRPC performs a JSON-RPC call against the seeded endpoint.
func solanaRPC(ctx context.Context, endpoint, method string, params interface{}) (json.RawMessage, error) {
	payload := map[string]interface{}{"jsonrpc": "2.0", "id": 1, "method": method, "params": params}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := solanaHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var parsed solanaRPCResp
	if uerr := json.Unmarshal(body, &parsed); uerr != nil {
		return nil, uerr
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("solana rpc %s: %s", method, parsed.Error.Message)
	}
	return parsed.Result, nil
}

// SolanaBuildSend builds a signed SystemProgram.transfer tx; fetches the
// recent blockhash from the node and optionally broadcasts.
func SolanaBuildSend(ctx context.Context, seed []byte, path, endpoint, to string, lamports uint64, broadcast bool) (signedB64, txSig string, err error) {
	priv, pub, err := edKeypair(seed, path)
	if err != nil {
		return "", "", err
	}
	if lamports == 0 {
		return "", "", errors.New("lamports must be > 0")
	}
	toRaw, err := base58Decode(to)
	if err != nil || len(toRaw) != 32 {
		return "", "", errors.New("invalid solana destination address")
	}
	res, err := solanaRPC(ctx, endpoint, "getLatestBlockhash", nil)
	if err != nil {
		return "", "", fmt.Errorf("solana getLatestBlockhash: %w", err)
	}
	var bh struct {
		Value struct {
			Blockhash string `json:"blockhash"`
		} `json:"value"`
	}
	if err := json.Unmarshal(res, &bh); err != nil || bh.Value.Blockhash == "" {
		return "", "", errors.New("solana getLatestBlockhash: malformed reply")
	}
	blockhash, err := base58Decode(bh.Value.Blockhash)
	if err != nil || len(blockhash) != 32 {
		return "", "", errors.New("solana blockhash: malformed base58")
	}

	systemProg, _ := base58Decode("11111111111111111111111111111111")
	accounts := [][]byte{[]byte(pub), toRaw, systemProg}

	ixData := make([]byte, 12)
	binary.LittleEndian.PutUint32(ixData, 2) // SystemProgram.transfer enum
	binary.LittleEndian.PutUint64(ixData[4:], lamports)

	var msg bytes.Buffer
	msg.Write([]byte{0x01, 0x00, 0x01}) // header: 1 sig, 0 ro-signed, 1 ro-unsigned
	msg.Write(encodeUvarint(uint64(len(accounts))))
	for _, a := range accounts {
		msg.Write(a)
	}
	msg.Write(blockhash)
	msg.Write(encodeUvarint(1)) // one instruction
	var ix bytes.Buffer
	ix.WriteByte(2)                      // programIdIndex
	ix.Write(encodeUvarint(2))           // account count
	ix.Write([]byte{0, 1})               // account indices
	ix.Write(encodeUvarint(uint64(len(ixData))))
	ix.Write(ixData)
	msg.Write(ix.Bytes())

	sig := ed.Sign(priv, msg.Bytes())
	var tx bytes.Buffer
	tx.Write(encodeUvarint(1)) // one signature
	tx.Write(sig)
	tx.Write(msg.Bytes())

	if broadcast {
		payload := []interface{}{
			base64.StdEncoding.EncodeToString(tx.Bytes()),
			map[string]string{"encoding": "base64"},
		}
		res, err := solanaRPC(ctx, endpoint, "sendTransaction", payload)
		if err != nil {
			return "", "", fmt.Errorf("solana broadcast: %w", err)
		}
		var sigStr string
		if uerr := json.Unmarshal(res, &sigStr); uerr != nil {
			return "", "", uerr
		}
		return base64.StdEncoding.EncodeToString(tx.Bytes()), sigStr, nil
	}
	return base64.StdEncoding.EncodeToString(tx.Bytes()), "", nil
}
