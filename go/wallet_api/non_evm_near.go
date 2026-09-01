package main

// non_evm_near.go — NEAR SDK: implicit account address (hex of the ed25519
// public key), message signing, and a Borsh-serialized transfer transaction
// with 'broadcast_tx_commit' against the seeded JSON-RPC endpoint.

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	ed "golang.org/x/crypto/ed25519"
)

// NearAddress returns the implicit account id = hex of the ed25519 pubkey.
func NearAddress(seed []byte, path string) (string, error) {
	pub, err := edPubKey(seed, path)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(pub[:32]), nil
}

// nearBuf is a tiny Borsh encoder.
type nearBuf struct{ b []byte }

func (n *nearBuf) u8(v byte)    { n.b = append(n.b, v) }
func (n *nearBuf) u32(v uint32) { var t [4]byte; binary.LittleEndian.PutUint32(t[:], v); n.b = append(n.b, t[:]...) }
func (n *nearBuf) u64(v uint64) { var t [8]byte; binary.LittleEndian.PutUint64(t[:], v); n.b = append(n.b, t[:]...) }
func (n *nearBuf) u128(v *big.Int) {
	bs := v.Bytes()
	var t [16]byte
	for i := 0; i < len(bs); i++ {
		t[i] = bs[len(bs)-1-i]
	}
	n.b = append(n.b, t[:]...)
}
func (n *nearBuf) str(s string)   { n.u32(uint32(len(s))); n.b = append(n.b, s...) }
func (n *nearBuf) raw(b []byte)   { n.b = append(n.b, b...) }

// NearBuildSend builds {signer_id, pub_key enum0 ed25519, nonce, receiver,
// blockhash, actions=[{sig? type1 transfer u128}]} then {signature}.
// nonce and recent block hash are needed from the node (pass them in).
type NearParams struct {
	Nonce     uint64
	BlockHashB58 string
}

// NearBuildSend deserialized signed tx; broadcast is via
// broadcast_tx_commit [base64(tx)] (node-generated block id via helper).
func NearBuildSend(ctx context.Context, seed []byte, path, endpoint, receiver string, amountYocto *big.Int, params NearParams, broadcast bool) (string, string, error) {
	if params.Nonce == 0 || params.BlockHashB58 == "" {
		return "", "", errors.New("near needs account nonce + recent block hash (fetched from the node)")
	}
	priv, pub, err := edKeypair(seed, path)
	if err != nil {
		return "", "", err
	}
	blockHash, err := base58Decode(params.BlockHashB58)
	if err != nil || len(blockHash) != 32 {
		return "", "", errors.New("near block hash base58 malformed")
	}
	if receiver == "" {
		return "", "", errors.New("near receiver empty")
	}
	var tx nearBuf
	tx.str(receiver) // serialized signer_id (we use the receiver as placeholder below replaced)
	// Signer id = implicit account (hex of pub)
	signerID, err := NearAddress(seed, path)
	if err != nil {
		return "", "", err
	}
	var tx2 nearBuf
	tx2.str(signerID)           // signer_id
	tx2.u8(0)                   // public_key variant = ED25519
	tx2.raw(pub)
	tx2.u64(params.Nonce)
	tx2.str(receiver)
	tx2.raw(blockHash)
	tx2.u32(1)                  // 1 action
	tx2.u8(0)                   // Transfer variant
	tx2.u128(amountYocto)
	rawTx := tx2.b
	_ = tx

	hash := sha256Hash(rawTx)
	sig := ed.Sign(priv, hash)

	var signed nearBuf
	signed.raw(rawTx)
	signed.u8(0)                 // signature variant = ED25519
	signed.raw(sig)
	rawSigned := signed.b

	if !broadcast {
		return base64ToStringN(rawSigned), hexOf(hash), nil
	}
	resp, err := postRaw(ctx, endpoint, mapMarshal(map[string]interface{}{
		"jsonrpc":  "2.0",
		"id":      1,
		"method":   "broadcast_tx_commit",
		"params":   []interface{}{base64ToStringN(rawSigned)},
	}))
	if err != nil {
		return "", "", fmt.Errorf("near broadcast: %w", err)
	}
	var parsed struct {
		Result struct { Transaction struct { Hash string `json:"hash"` } `json:"transaction"` } `json:"result"`
	}
	if err := json.Unmarshal(resp, &parsed); err != nil || parsed.Result.Transaction.Hash == "" {
		return "", "", errors.New("near broadcast: malformed reply")
	}
	return base64ToStringN(rawSigned), parsed.Result.Transaction.Hash, nil
}

func base64ToStringN(b []byte) string { return utilsBase64(b) }

func mapMarshal(v map[string]interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
