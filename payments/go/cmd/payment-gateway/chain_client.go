package main

// chain_client.go - on-chain balance reads for deposit verification.
//
// Verification has to observe the chain. Anything less cannot distinguish a
// customer who paid from a customer who only opened the payment page.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// baseUnitDecimals is the number of decimal places each currency's smallest
// unit uses. A balance read from the chain is in these units.
var baseUnitDecimals = map[string]int{
	"BTC":   8,
	"ETH":   18,
	"USDT":  6, // ERC-20 USDT uses 6 decimals
	"USDC":  6,
	"BNB":   18,
	"MATIC": 18,
	"TRX":   6,
	"LTC":   8,
	"DOGE":  8,
}

type chainClient struct {
	rpcURL string
	http   *http.Client
}

func newChainClient(rpcURL string) *chainClient {
	if strings.TrimSpace(rpcURL) == "" {
		return nil
	}
	return &chainClient{
		rpcURL: rpcURL,
		http:   &http.Client{Timeout: 15 * time.Second},
	}
}

type rpcRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *chainClient) call(method string, params ...interface{}) (json.RawMessage, error) {
	body, err := json.Marshal(rpcRequest{JSONRPC: "2.0", ID: 1, Method: method, Params: params})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.rpcURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rpc returned HTTP %d", resp.StatusCode)
	}

	var decoded rpcResponse
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, err
	}
	if decoded.Error != nil {
		return nil, fmt.Errorf("rpc error %d: %s", decoded.Error.Code, decoded.Error.Message)
	}
	return decoded.Result, nil
}

// nativeBalance reads the native-coin balance of an address in base units.
func (c *chainClient) nativeBalance(address string) (*big.Int, error) {
	result, err := c.call("eth_getBalance", address, "latest")
	if err != nil {
		return nil, err
	}

	var hexBalance string
	if err := json.Unmarshal(result, &hexBalance); err != nil {
		return nil, fmt.Errorf("unexpected balance response: %w", err)
	}
	return hexToBigInt(hexBalance)
}

// erc20Balance reads an ERC-20 balance. The token contract must be configured
// per currency; there is no default, because guessing a contract address would
// silently read the wrong token.
func (c *chainClient) erc20Balance(tokenContract, holder string) (*big.Int, error) {
	if !isHexAddress(tokenContract) {
		return nil, fmt.Errorf("invalid token contract address %q", tokenContract)
	}
	holder = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(holder)), "0x")
	if len(holder) != 40 || !isHex(holder) {
		return nil, fmt.Errorf("invalid holder address %q", holder)
	}

	// balanceOf(address) selector = 0x70a08231, then the address left-padded
	// to 32 bytes.
	data := "0x70a08231" + strings.Repeat("0", 24) + holder

	result, err := c.call("eth_call", map[string]string{
		"to":   tokenContract,
		"data": data,
	}, "latest")
	if err != nil {
		return nil, err
	}

	var hexBalance string
	if err := json.Unmarshal(result, &hexBalance); err != nil {
		return nil, fmt.Errorf("unexpected token balance response: %w", err)
	}
	return hexToBigInt(hexBalance)
}

// balanceFor reads the balance for a currency, dispatching on whether the
// currency is a native coin or a token.
func (c *chainClient) balanceFor(currency, address string) (*big.Int, error) {
	upper := strings.ToUpper(strings.TrimSpace(currency))
	if _, known := baseUnitDecimals[upper]; !known {
		return nil, fmt.Errorf("unsupported currency %q", currency)
	}

	if contract, ok := erc20Contracts[upper]; ok {
		return c.erc20Balance(contract, address)
	}

	switch upper {
	case "BTC", "LTC", "DOGE", "TRX":
		return nil, fmt.Errorf(
			"%s deposits need an %s RPC endpoint; this gateway is configured for EVM JSON-RPC only",
			upper, upper)
	}
	return c.nativeBalance(address)
}

// erc20Contracts maps a token currency to its contract address. Populated from
// configuration so a deployment can point at the right chain; empty means the
// currency must be treated as native.
var erc20Contracts = map[string]string{}

// configureERC20 registers token contract addresses from configuration. An
// empty address is skipped rather than registered: a registered-but-empty
// entry would make the currency look configured and then fail at call time.
func configureERC20(pairs map[string]string) {
	for currency, contract := range pairs {
		trimmed := strings.TrimSpace(contract)
		if trimmed == "" {
			continue
		}
		erc20Contracts[strings.ToUpper(strings.TrimSpace(currency))] = trimmed
	}
}

func hexToBigInt(raw string) (*big.Int, error) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(raw), "0x")
	if trimmed == "" {
		return big.NewInt(0), nil
	}
	if !isHex(trimmed) {
		return nil, fmt.Errorf("invalid hex quantity %q", raw)
	}
	value, ok := new(big.Int).SetString(trimmed, 16)
	if !ok {
		return nil, fmt.Errorf("invalid hex quantity %q", raw)
	}
	return value, nil
}

func isHex(s string) bool {
	for _, c := range s {
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		case c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}

func isHexAddress(s string) bool {
	trimmed := strings.TrimPrefix(strings.TrimSpace(s), "0x")
	return len(trimmed) == 40 && isHex(trimmed)
}

// toBaseUnits converts a human-readable amount into the currency's smallest
// unit. It works on the decimal string rather than a float so that values such
// as 0.1 ETH do not lose precision on the way in.
func toBaseUnits(amount float64, currency string) (*big.Int, error) {
	if amount < 0 {
		return nil, errors.New("amount must not be negative")
	}
	decimals, ok := baseUnitDecimals[strings.ToUpper(strings.TrimSpace(currency))]
	if !ok {
		return nil, fmt.Errorf("unsupported currency %q", currency)
	}

	// Formatting a float directly to the currency's precision reintroduces
	// binary rounding error (0.1 ETH formats as 0.10000000000000000555...).
	// strconv's shortest representation round-trips the float, so the digits
	// the caller wrote are the digits used here.
	text := strconv.FormatFloat(amount, 'f', -1, 64)
	if strings.Contains(text, "e") || strings.Contains(text, "E") {
		// Scientific notation would break the digit split below.
		text = strconv.FormatFloat(amount, 'f', decimals, 64)
	}

	parts := strings.SplitN(text, ".", 2)
	whole := parts[0]
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
	}
	if len(fraction) > decimals {
		fraction = fraction[:decimals]
	}
	fraction += strings.Repeat("0", decimals-len(fraction))

	combined := whole + fraction
	combined = strings.TrimLeft(combined, "0")
	if combined == "" {
		return big.NewInt(0), nil
	}
	value, ok := new(big.Int).SetString(combined, 10)
	if !ok {
		return nil, fmt.Errorf("could not convert %v %s to base units", amount, currency)
	}
	return value, nil
}
