package main

// non_evm_sui.go — Sui SDK: intent-scoped blake2b-256 hashing and Ed25519
// signing, node-assisted tx building via sui_moveCall on the fullnode
// JSON-RPC, and final submission via sui_executeTransactionBlock. Owned SUI
// coin objects are discovered via sui_getOwnedCoins; anything besides a
// plain gas-coin transfer fails closed.

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"golang.org/x/crypto/blake2b"
	ed "golang.org/x/crypto/ed25519"
)

// SuiAddress derives the SUI address: 0x || blake2b-256(0x00 || pub).
func SuiAddress(seed []byte, path string) (string, error) {
	_, pub, err := edKeypair(seed, path)
	if err != nil {
		return "", err
	}
	raw := append([]byte{0x00}, pub...)
	sum := blake2b.Sum256(raw)
	return "0x" + hex.EncodeToString(sum[:]), nil
}

// suiIntent prefixes [scope, version, appId] for tx signing: [0, 0, 0].
func suiSignTxBytes(priv ed.PrivateKey, txBytes []byte) ([]byte, error) {
	msg := append([]byte{0, 0, 0}, txBytes...)
	d := blake2b.Sum256(msg)
	return ed.Sign(priv, d[:]), nil
}

// suiOwnedCoins calls sui_getOwnedCoins for the SUI coin type.
type suiCoin struct {
	CoinType                  string `json:"coinType"`
	CoinObjectId              string `json:"coinObjectId"`
	Balance                   string `json:"balance"`
}

func suiOwnedCoins(ctx context.Context, endpoint, address string) ([]suiCoin, error) {
	params := []interface{}{address, "0x2::sui::SUI"}
	res, err := solanaRPC(ctx, endpoint, "sui_getOwnedCoins", params)
	if err != nil {
		return nil, err
	}
	var out []suiCoin
	if err := json.Unmarshal(res, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SuiBuildSend discovers the sender's owned SUI coin objects, asks the
// node to build the transfer (sui_moveCall public_transfer on 0x2::pay::pay
// module via transferSui), signs the intent-prefixed blake2b digest, and
// broadcasts via sui_executeTransactionBlock.
func SuiBuildSend(ctx context.Context, seed []byte, path, endpoint, to string, amount uint64, broadcast bool) (string, string, error) {
	priv, _, err := edKeypair(seed, path)
	if err != nil {
		return "", "", err
	}
	if amount == 0 {
		return "", "", errors.New("sui amount must be > 0")
	}
	fromAddr, err := SuiAddress(seed, path)
	if err != nil {
		return "", "", err
	}
	coins, err := suiOwnedCoins(ctx, endpoint, fromAddr)
	if err != nil {
		return "", "", fmt.Errorf("sui owned coins: %w", err)
	}
	if len(coins) == 0 {
		return "", "", fmt.Errorf("no SUI coins owned by %s", fromAddr)
	}
	// public_transfer(tx, obj, recipient) on module 0x2::transfer
	params := []interface{}{
		fromAddr,
		"0x2",
		"transfer",
		"public_transfer",
		[]string{},
		[]interface{}{coins[0].CoinObjectId, to},
		"", // gas budget unset — let node pick
		[2]interface{}{"0x2::sui::SUI", amount},
	}
	_ = params
	call := []interface{}{
		fromAddr,
		"0x2",
		"transfer",
		"public_transfer",
		[]string{},
		[]interface{}{coins[0].CoinObjectId, to},
		"5000000",                  // gas budget 0.005 SUI
	}
	res, err := solanaRPC(ctx, endpoint, "sui_moveCall", call)
	if err != nil {
		return "", "", fmt.Errorf("sui_moveCall: %w", err)
	}
	var built struct {
		TxBytes string `json:"txBytes"`
		Gas     []struct {
			CoinObjectId string `json:"coinObjectId"`
		} `json:"gas"`
	}
	if err := json.Unmarshal(res, &built); err != nil || built.TxBytes == "" {
		return "", "", errors.New("sui_moveCall: malformed reply")
	}
	txBytes, err := base64.StdEncoding.DecodeString(built.TxBytes)
	if err != nil {
		return "", "", err
	}
	sig, err := suiSignTxBytes(priv, txBytes)
	if err != nil {
		return "", "", err
	}
	if !broadcast {
		return base64.StdEncoding.EncodeToString(sig), "", nil
	}
	exeRes, err := solanaRPC(ctx, endpoint, "sui_executeTransactionBlock",
		[]interface{}{built.TxBytes, []string{base64.StdEncoding.EncodeToString(sig)}, map[string]bool{"showEffects": true, "showEvents": true}, "WaitForLocalExecution"})
	if err != nil {
		return "", "", fmt.Errorf("sui broadcast: %w", err)
	}
	var parsed struct {
		Digest string `json:"digest"`
	}
	if err := json.Unmarshal(exeRes, &parsed); err != nil || parsed.Digest == "" {
		return "", "", errors.New("sui broadcast: malformed reply")
	}
	return base64.StdEncoding.EncodeToString(sig), parsed.Digest, nil
}
