package main

// non_evm_state.go — chain-state fetchers for the account-model non-EVM
// families.
//
// An account-chain transfer cannot be signed correctly without fresh state:
// NEAR needs the access-key nonce plus a recent block hash, Stellar needs the
// source account sequence, Algorand needs the genesis hash and a valid round
// window, Tezos needs the contract counter, MultiversX and Aptos need the
// account nonce/sequence. Passing zeros produces a transaction the network
// rejects, so every value below is read from the chain's own node. A node that
// cannot answer makes the transfer fail closed — no value is ever invented.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/tigerwallet/chainsdk"
)

// jsonRPCCall performs a JSON-RPC 2.0 call and returns the raw `result`.
func jsonRPCCall(ctx context.Context, endpoint, method string, params interface{}) (json.RawMessage, error) {
	body, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	})
	if err != nil {
		return nil, err
	}
	raw, err := postRawJSON(ctx, endpoint, body)
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Message string `json:"message"`
			Data    string `json:"data"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("rpc reply decode: %w", err)
	}
	if envelope.Error != nil {
		msg := envelope.Error.Message
		if envelope.Error.Data != "" {
			msg += ": " + envelope.Error.Data
		}
		return nil, fmt.Errorf("rpc %s: %s", method, msg)
	}
	if len(envelope.Result) == 0 {
		return nil, fmt.Errorf("rpc %s returned no result", method)
	}
	return envelope.Result, nil
}

// nearSendParams reads the access-key nonce and a recent block hash. NEAR
// requires the nonce to be strictly greater than the access key's stored
// nonce, so it is incremented by one.
func nearSendParams(ctx context.Context, endpoint, accountID, publicKey string) (chainsdk.NearParams, error) {
	nonceRaw, err := jsonRPCCall(ctx, endpoint, "query", map[string]interface{}{
		"request_type": "view_access_key",
		"finality":     "final",
		"account_id":   accountID,
		"public_key":   publicKey,
	})
	if err != nil {
		return chainsdk.NearParams{}, fmt.Errorf("near access key: %w", err)
	}
	var key struct {
		Nonce uint64 `json:"nonce"`
	}
	if err := json.Unmarshal(nonceRaw, &key); err != nil {
		return chainsdk.NearParams{}, fmt.Errorf("near access key decode: %w", err)
	}
	blockRaw, err := jsonRPCCall(ctx, endpoint, "block", map[string]interface{}{"finality": "final"})
	if err != nil {
		return chainsdk.NearParams{}, fmt.Errorf("near block: %w", err)
	}
	var block struct {
		Header struct {
			Hash string `json:"hash"`
		} `json:"header"`
	}
	if err := json.Unmarshal(blockRaw, &block); err != nil || block.Header.Hash == "" {
		return chainsdk.NearParams{}, fmt.Errorf("near block decode: missing header hash")
	}
	return chainsdk.NearParams{Nonce: key.Nonce + 1, BlockHashB58: block.Header.Hash}, nil
}

// stellarSequence reads the source account's current sequence number.
func stellarSequence(ctx context.Context, horizon, address string) (uint64, error) {
	raw, err := httpGetCtx(ctx, strings.TrimRight(horizon, "/")+"/accounts/"+url.PathEscape(address))
	if err != nil {
		return 0, fmt.Errorf("stellar account: %w", err)
	}
	var acc struct {
		Sequence string `json:"sequence"`
	}
	if err := json.Unmarshal(raw, &acc); err != nil {
		return 0, fmt.Errorf("stellar account decode: %w", err)
	}
	seq, err := strconv.ParseUint(strings.TrimSpace(acc.Sequence), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("stellar sequence %q: %w", acc.Sequence, err)
	}
	return seq, nil
}

// algoTxParams reads the genesis hash, current round and minimum fee so a pay
// transaction lands inside a valid round window.
func algoTxParams(ctx context.Context, endpoint string) (genesisB64 string, firstValid, lastValid, fee uint64, err error) {
	raw, err := httpGetCtx(ctx, strings.TrimRight(endpoint, "/")+"/v2/transactions/params")
	if err != nil {
		return "", 0, 0, 0, fmt.Errorf("algorand params: %w", err)
	}
	var p struct {
		GenesisHash string `json:"genesis-hash"`
		LastRound   uint64 `json:"last-round"`
		MinFee      uint64 `json:"min-fee"`
		Fee         uint64 `json:"fee"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return "", 0, 0, 0, fmt.Errorf("algorand params decode: %w", err)
	}
	if p.GenesisHash == "" || p.LastRound == 0 {
		return "", 0, 0, 0, fmt.Errorf("algorand params incomplete (genesis-hash/last-round missing)")
	}
	// Valid window: the next round through +1000 rounds, the standard window
	// algod accepts. Fee: algod's min-fee, falling back to the suggested fee.
	fee = p.MinFee
	if fee == 0 {
		fee = p.Fee
	}
	if fee == 0 {
		return "", 0, 0, 0, fmt.Errorf("algorand node reported no minimum fee")
	}
	return p.GenesisHash, p.LastRound + 1, p.LastRound + 1000, fee, nil
}

// tezosCounter reads the contract's current counter; a new operation must use
// counter+1.
func tezosCounter(ctx context.Context, endpoint, address string) (uint64, error) {
	u := strings.TrimRight(endpoint, "/") + "/chains/main/blocks/head/context/contracts/" +
		url.PathEscape(address) + "/counter"
	raw, err := httpGetCtx(ctx, u)
	if err != nil {
		return 0, fmt.Errorf("tezos counter: %w", err)
	}
	c, err := strconv.ParseUint(strings.Trim(strings.TrimSpace(string(raw)), "\""), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("tezos counter %q: %w", string(raw), err)
	}
	return c + 1, nil
}

// aptosSequence reads the account's next sequence number.
func aptosSequence(ctx context.Context, endpoint, address string) (uint64, error) {
	raw, err := httpGetCtx(ctx, strings.TrimRight(endpoint, "/")+"/accounts/"+url.PathEscape(address))
	if err != nil {
		return 0, fmt.Errorf("aptos account: %w", err)
	}
	var acc struct {
		Sequence string `json:"sequence_number"`
	}
	if err := json.Unmarshal(raw, &acc); err != nil {
		return 0, fmt.Errorf("aptos account decode: %w", err)
	}
	seq, err := strconv.ParseUint(strings.TrimSpace(acc.Sequence), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("aptos sequence %q: %w", acc.Sequence, err)
	}
	return seq, nil
}
