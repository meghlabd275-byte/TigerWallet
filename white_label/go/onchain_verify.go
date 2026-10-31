package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// On-chain verification for white-label token registration.
//
// A token only exists because a contract was deployed to a chain. The
// registration endpoint used to invent an address and store it, so the console
// listed tokens that no node had ever seen and any downstream balance or
// transfer call against that address failed. Registration now takes the
// deployed address from the operator and confirms the contract is really there
// before the row is written.

const (
	// wlRPCTimeout bounds every upstream node call.
	wlRPCTimeout = 10 * time.Second
	// wlMaxRPCPayload caps the response we are willing to read from a node.
	wlMaxRPCPayload = 4 << 20
)

var wlHTTPClient = &http.Client{Timeout: wlRPCTimeout}

// wlDefaultChainRPC maps well-known chain IDs to a public endpoint. Operators
// override any of them with WL_RPC_<chainID>, and chain 1 with EVM_RPC_URL.
var wlDefaultChainRPC = map[uint64]string{
	1:     "https://eth.llamarpc.com",
	10:    "https://mainnet.optimism.io",
	56:    "https://bsc-dataseed.binance.org",
	137:   "https://polygon-rpc.com",
	8453:  "https://mainnet.base.org",
	42161: "https://arb1.arbitrum.io/rpc",
	43114: "https://api.avax.network/ext/bc/C/rpc",
}

// chainRPCEndpoint resolves the endpoint used to verify a token contract on
// chainID. It returns an error when the chain has no configured node, so the
// caller fails closed instead of writing an unverified address.
func chainRPCEndpoint(chainID uint64) (string, error) {
	if v := strings.TrimSpace(os.Getenv(fmt.Sprintf("WL_RPC_%d", chainID))); v != "" {
		return v, nil
	}
	if chainID == 1 {
		if v := strings.TrimSpace(os.Getenv("EVM_RPC_URL")); v != "" {
			return v, nil
		}
	}
	if v, ok := wlDefaultChainRPC[chainID]; ok {
		return v, nil
	}
	return "", fmt.Errorf("no RPC endpoint configured for chain %d; set WL_RPC_%d", chainID, chainID)
}

// evmAddressPattern matches a 20-byte hex address.
var evmAddressPattern = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)

// normalizeEVMAddress validates an EVM address and returns it in the canonical
// lower-case 0x form. Lower-casing keeps the database unique index meaningful:
// the same contract must not be storable twice under different casing.
func normalizeEVMAddress(addr string) (string, error) {
	addr = strings.TrimSpace(addr)
	if !evmAddressPattern.MatchString(addr) {
		return "", fmt.Errorf("address %q is not a 20-byte hex EVM address", addr)
	}
	return strings.ToLower(addr), nil
}

type wlRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type wlRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// wlRPCCall performs one JSON-RPC 2.0 call and unmarshals the result into out.
func wlRPCCall(ctx context.Context, endpoint, method string, params []interface{}, out interface{}) error {
	body, err := json.Marshal(wlRPCRequest{JSONRPC: "2.0", ID: 1, Method: method, Params: params})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := wlHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("node %s returned HTTP %d", endpoint, resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, wlMaxRPCPayload))
	if err != nil {
		return err
	}
	var r wlRPCResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return fmt.Errorf("invalid JSON-RPC response from %s: %w", endpoint, err)
	}
	if r.Error != nil {
		return fmt.Errorf("node error %d: %s", r.Error.Code, r.Error.Message)
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(r.Result, out)
}

// verifyContractDeployed confirms that address holds contract code on chainID.
//
// eth_getCode returns "0x" for an address with no code. That covers both a
// never-deployed address and an externally-owned account, and either way the
// address is not a token contract, so both are rejected.
func verifyContractDeployed(ctx context.Context, chainID uint64, address string) error {
	endpoint, err := chainRPCEndpoint(chainID)
	if err != nil {
		return err
	}
	addr, err := normalizeEVMAddress(address)
	if err != nil {
		return err
	}
	var code string
	if err := wlRPCCall(ctx, endpoint, "eth_getCode", []interface{}{addr, "latest"}, &code); err != nil {
		return fmt.Errorf("could not read contract code for %s on chain %d: %w", addr, chainID, err)
	}
	if strings.TrimSpace(code) == "" || strings.EqualFold(code, "0x") {
		return fmt.Errorf("no contract is deployed at %s on chain %d", addr, chainID)
	}
	return nil
}
