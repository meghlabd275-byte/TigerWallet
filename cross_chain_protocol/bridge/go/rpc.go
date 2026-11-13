package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/sha3"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"
)

// rpcCallTimeout bounds a single upstream JSON-RPC call.
const rpcCallTimeout = 15 * time.Second

type jsonRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type jsonRPCResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// rpcCall performs a JSON-RPC call against an EVM endpoint.
func rpcCall(ctx context.Context, endpoint, method string, params ...interface{}) (json.RawMessage, error) {
	if strings.TrimSpace(endpoint) == "" {
		return nil, fmt.Errorf("no RPC endpoint configured")
	}

	payload, err := json.Marshal(jsonRPCRequest{JSONRPC: "2.0", ID: 1, Method: method, Params: params})
	if err != nil {
		return nil, fmt.Errorf("encode RPC request: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, rpcCallTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: rpcCallTimeout}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d from RPC endpoint", resp.StatusCode)
	}

	var decoded jsonRPCResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, fmt.Errorf("decode RPC response: %w", err)
	}
	if decoded.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", decoded.Error.Code, decoded.Error.Message)
	}
	return decoded.Result, nil
}

// ethCallRaw performs eth_call and returns the raw hex result.
func ethCallRaw(ctx context.Context, rpcURL, to, data string) (string, error) {
	raw, err := rpcCall(ctx, rpcURL, "eth_call", map[string]string{
		"to":   to,
		"data": data,
	}, "latest")
	if err != nil {
		return "", err
	}
	var result string
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("decode eth_call result: %w", err)
	}
	if result == "0x" || result == "" {
		return "", fmt.Errorf("eth_call returned no data for %s", to)
	}
	return result, nil
}

// ethReceipt is the subset of an EVM transaction receipt this service needs.
type ethReceipt struct {
	TransactionHash string `json:"transactionHash"`
	BlockNumber     string `json:"blockNumber"`
	Status          string `json:"status"`
	GasUsed         string `json:"gasUsed"`
}

// ethGetTransactionReceipt returns the receipt, or nil when the node has not
// mined the transaction yet.
func ethGetTransactionReceipt(ctx context.Context, rpcURL, txHash string) (*ethReceipt, error) {
	raw, err := rpcCall(ctx, rpcURL, "eth_getTransactionReceipt", txHash)
	if err != nil {
		return nil, err
	}
	if string(raw) == "null" {
		return nil, nil
	}
	var receipt ethReceipt
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return nil, fmt.Errorf("decode receipt: %w", err)
	}
	return &receipt, nil
}

// isHexHash reports whether s is a 32-byte 0x-prefixed hash.
func isHexHash(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) != 66 || !strings.HasPrefix(s, "0x") {
		return false
	}
	_, err := hex.DecodeString(s[2:])
	return err == nil
}

// encodeSelectorHex returns the 4-byte function selector for a signature.
func encodeSelectorHex(signature string) string {
	h := sha3.NewLegacyKeccak256()
	h.Write([]byte(signature))
	sum := h.Sum(nil)
	return hex.EncodeToString(sum[:4])
}

// encodeUintArg left-pads a value into a 32-byte ABI word.
func encodeUintArg(value int64) string {
	n := big.NewInt(value)
	return fmt.Sprintf("%064x", n)
}

// encodeAddressArg left-pads an address into a 32-byte ABI word.
func encodeAddressArg(address string) string {
	return strings.Repeat("0", 24) + strings.TrimPrefix(strings.ToLower(address), "0x")
}

// encodeDynamicBytesArg encodes a dynamic bytes argument as an offset into the
// argument block followed by the length-prefixed payload.
func encodeDynamicBytesArg(wordIndex int, payload []byte) string {
	offset := wordIndex * 32
	encoded := fmt.Sprintf("%064x", big.NewInt(int64(offset)))
	encoded += fmt.Sprintf("%064x", big.NewInt(int64(len(payload))))
	if len(payload) > 0 {
		encoded += hex.EncodeToString(payload)
	}
	return encoded
}

// decodeABIUintAt decodes the 32-byte word at the given index of a return blob.
func decodeABIUintAt(raw string, wordIndex int) (*big.Int, error) {
	value := strings.TrimPrefix(strings.TrimSpace(raw), "0x")
	start := wordIndex * 64
	if len(value) < start+64 {
		return nil, fmt.Errorf("return value has no word %d", wordIndex)
	}
	n, ok := new(big.Int).SetString(value[start:start+64], 16)
	if !ok {
		return nil, fmt.Errorf("word %d is not a uint", wordIndex)
	}
	return n, nil
}

// weiToFloat converts a wei amount into ether units.
func weiToFloat(wei *big.Int) float64 {
	if wei == nil {
		return 0
	}
	out, _ := new(big.Float).Quo(new(big.Float).SetInt(wei), big.NewFloat(1e18)).Float64()
	return out
}

// formatAmount normalizes a user-supplied decimal amount.
func formatAmount(amount float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.8f", amount), "0"), ".")
}

// nativeSymbolForChain maps a chain id onto its native coin symbol.
func nativeSymbolForChain(chainID int64) string {
	switch chainID {
	case 1, 42161, 10, 8453:
		return "ETH"
	case 56:
		return "BNB"
	case 137:
		return "MATIC"
	case 43114:
		return "AVAX"
	case 25:
		return "CRO"
	case 42220:
		return "CELO"
	case 1284:
		return "GLMR"
	default:
		return "ETH"
	}
}
