package main

// analyzer.go — real EVM token contract analysis.
//
// Capabilities are detected from the deployed bytecode, not assumed. Selectors
// are computed by hashing the canonical signatures so they cannot drift from
// the ABI, and the opcode walk skips PUSH immediates so operand bytes are never
// misread as instructions.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/sha3"
)

// EVM opcodes used by the analysis.
const (
	opPush1        = 0x60
	opPush4        = 0x63
	opPush32       = 0x7f
	opCallCode     = 0xf2
	opDelegateCall = 0xf4
	opCreate2      = 0xf5
	opSelfDestruct = 0xff
)

// keccakSelector returns the four-byte selector for a canonical signature.
func keccakSelector(signature string) [4]byte {
	h := sha3.NewLegacyKeccak256()
	h.Write([]byte(signature))
	var out [4]byte
	copy(out[:], h.Sum(nil)[:4])
	return out
}

// capabilitySignatures maps a detected capability to the signatures that
// indicate it. A capability is reported when any of its signatures is present
// in the dispatcher.
var capabilitySignatures = map[string][]string{
	"transfer":           {"transfer(address,uint256)"},
	"approve":            {"approve(address,uint256)"},
	"balance_of":         {"balanceOf(address)"},
	"total_supply":       {"totalSupply()"},
	"mint":               {"mint(address,uint256)", "mint(uint256)", "mint(address,uint256,bytes)"},
	"pause":              {"pause()", "unpause()", "paused()"},
	"blacklist":          {"setBlacklist(address,bool)", "addBlackList(address)", "destroyBlackFunds(address)", "isBlackListed(address)", "blacklist(address)"},
	"anti_whale":         {"setMaxTxAmount(uint256)", "setMaxWalletAmount(uint256)", "maxTxAmount()", "maxWalletAmount()"},
	"cooldown":           {"setCooldown(uint256)", "cooldownSeconds()", "setTradingCooldown(uint256)"},
	"fee_control":        {"setFees(uint256)", "setTaxFee(uint256)", "setBuyFee(uint256)", "setSellFee(uint256)"},
	"ownership":          {"owner()", "transferOwnership(address)", "renounceOwnership()"},
	"proxy_upgrade":      {"upgradeTo(address)", "upgradeToAndCall(address,bytes)", "implementation()"},
	"trading_switch":     {"enableTrading()", "disableTrading()", "tradingEnabled()"},
	"erc721":             {"safeTransferFrom(address,address,uint256,bytes)", "ownerOf(uint256)"},
	"erc165":             {"supportsInterface(bytes4)"},
	"external_call_risk": {"multicall(bytes[])", "execute(address,bytes)"},
}

// capabilityTable maps each computed selector to its capability.
var capabilityTable = func() map[[4]byte]string {
	table := map[[4]byte]string{}
	for capability, signatures := range capabilitySignatures {
		for _, signature := range signatures {
			table[keccakSelector(signature)] = capability
		}
	}
	return table
}()

// detectCapabilities finds the capabilities present in deployed code.
func detectCapabilities(code []byte) map[string]bool {
	found := map[string]bool{}
	for i := 0; i+4 < len(code); i++ {
		if code[i] != opPush4 {
			continue
		}
		var key [4]byte
		copy(key[:], code[i+1:i+5])
		if capability, ok := capabilityTable[key]; ok {
			found[capability] = true
		}
	}
	return found
}

// opcodeFacts summarizes the instructions in deployed code.
type opcodeFacts struct {
	Instructions int
	DelegateCall int
	CallCode     int
	SelfDestruct int
	Create2      int
	DataBytes    int
}

// scanOpcodes walks the code, skipping PUSH immediates.
func scanOpcodes(code []byte) opcodeFacts {
	var f opcodeFacts
	for i := 0; i < len(code); {
		op := code[i]
		f.Instructions++

		switch op {
		case opDelegateCall:
			f.DelegateCall++
		case opCallCode:
			f.CallCode++
		case opSelfDestruct:
			f.SelfDestruct++
		case opCreate2:
			f.Create2++
		}

		if op >= opPush1 && op <= opPush32 {
			operand := int(op-opPush1) + 1
			f.DataBytes += operand
			i += 1 + operand
			continue
		}
		i++
	}
	return f
}

// codeAnalysis is the result of inspecting deployed bytecode.
type codeAnalysis struct {
	IsContract   bool
	CodeSize     int
	Capabilities map[string]bool
	Facts        opcodeFacts
	IsProxy      bool
}

// analyzeCode inspects deployed bytecode. Empty code means the address holds no
// contract, which is reported as such rather than as a clean token.
func analyzeCode(code []byte) codeAnalysis {
	a := codeAnalysis{Capabilities: map[string]bool{}}
	if len(code) == 0 {
		return a
	}
	a.IsContract = true
	a.CodeSize = len(code)
	a.Capabilities = detectCapabilities(code)
	a.Facts = scanOpcodes(code)
	a.IsProxy = a.Facts.DelegateCall > 0 || a.Capabilities["proxy_upgrade"]
	return a
}

// ---------------------------------------------------------------------------
// JSON-RPC
// ---------------------------------------------------------------------------

// rpcTimeout bounds a single upstream call.
const rpcTimeout = 15 * time.Second

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

// callRPC performs a JSON-RPC call against an endpoint.
func callRPC(ctx context.Context, client *http.Client, endpoint, method string, params ...interface{}) (json.RawMessage, error) {
	payload, err := json.Marshal(jsonRPCRequest{JSONRPC: "2.0", ID: 1, Method: method, Params: params})
	if err != nil {
		return nil, fmt.Errorf("encode RPC request: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, endpoint)
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

// fetchContractCode reads the deployed bytecode at an address.
func fetchContractCode(ctx context.Context, client *http.Client, endpoint, address string) ([]byte, error) {
	raw, err := callRPC(ctx, client, endpoint, "eth_getCode", address, "latest")
	if err != nil {
		return nil, err
	}
	var hexCode string
	if err := json.Unmarshal(raw, &hexCode); err != nil {
		return nil, fmt.Errorf("decode eth_getCode result: %w", err)
	}
	return decodeHexBytes(hexCode)
}

// decodeHexBytes converts a hex string to bytes.
func decodeHexBytes(raw string) ([]byte, error) {
	value := strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(raw), "0x"), "0X")
	if value == "" {
		return nil, nil
	}
	if len(value)%2 != 0 {
		value = "0" + value
	}
	out := make([]byte, len(value)/2)
	for i := range out {
		hi, ok := hexValue(value[i*2])
		if !ok {
			return nil, fmt.Errorf("invalid hex at offset %d", i*2)
		}
		lo, ok := hexValue(value[i*2+1])
		if !ok {
			return nil, fmt.Errorf("invalid hex at offset %d", i*2+1)
		}
		out[i] = hi<<4 | lo
	}
	return out, nil
}

// hexValue converts one hex character to its value.
func hexValue(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	default:
		return 0, false
	}
}

// isValidAddress reports whether s is a 0x-prefixed 20-byte hex address.
func isValidAddress(s string) bool {
	if !strings.HasPrefix(s, "0x") || len(s) != 42 {
		return false
	}
	for i := 2; i < len(s); i++ {
		if _, ok := hexValue(s[i]); !ok {
			return false
		}
	}
	return true
}

// encodeUint256Arg left-pads a hex quantity into a 32-byte ABI word.
func encodeUint256Arg(hexQuantity string) string {
	value := strings.TrimPrefix(strings.TrimPrefix(hexQuantity, "0x"), "0X")
	if value == "" {
		value = "0"
	}
	if len(value) >= 64 {
		return value[len(value)-64:]
	}
	return strings.Repeat("0", 64-len(value)) + value
}

// encodeAddressArg left-pads an address into a 32-byte ABI word.
func encodeAddressArg(address string) string {
	return strings.Repeat("0", 24) + strings.ToLower(strings.TrimPrefix(address, "0x"))
}

// hexQuantityToBigInt parses a JSON-RPC hex quantity.
func hexQuantityToBigInt(raw string) (*big.Int, error) {
	value := strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(raw), "0x"), "0X")
	if value == "" {
		return nil, fmt.Errorf("empty hex quantity")
	}
	n, ok := new(big.Int).SetString(value, 16)
	if !ok {
		return nil, fmt.Errorf("not a hex quantity: %q", raw)
	}
	return n, nil
}

// ethCall performs an eth_call against an endpoint.
func ethCall(ctx context.Context, client *http.Client, endpoint, to, data string) (string, error) {
	raw, err := callRPC(ctx, client, endpoint, "eth_call", map[string]string{
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
	return result, nil
}

// decodeABIString decodes a solidity string return value: a 32-byte offset, a
// 32-byte length, then the UTF-8 payload.
func decodeABIString(raw string) (string, error) {
	value := strings.TrimPrefix(raw, "0x")
	if len(value) < 128 {
		return "", fmt.Errorf("return value too short for a string")
	}
	offset, err := hexQuantityToBigInt("0x" + value[:64])
	if err != nil {
		return "", err
	}
	start := int(offset.Int64()) * 2
	if start+64 > len(value) {
		return "", fmt.Errorf("string offset out of range")
	}
	length, err := hexQuantityToBigInt("0x" + value[start:start+64])
	if err != nil {
		return "", err
	}
	byteLen := int(length.Int64())
	if byteLen == 0 {
		return "", nil
	}
	payloadStart := start + 64
	payloadEnd := payloadStart + byteLen*2
	if payloadEnd > len(value) {
		return "", fmt.Errorf("string payload out of range")
	}
	decoded, err := decodeHexBytes(value[payloadStart:payloadEnd])
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// decodeABIUint decodes a 32-byte uint return value.
func decodeABIUint(raw string) (*big.Int, error) {
	value := strings.TrimPrefix(raw, "0x")
	if len(value) < 64 {
		return nil, fmt.Errorf("return value too short for a uint")
	}
	return hexQuantityToBigInt("0x" + value[:64])
}
