package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

"github.com/tigerwallet/account-abstraction/abi"
)

// entryPointDefaults holds the deployed EntryPoint and factory addresses.
//
// Both are deployed at the same address on every major network, so one pair of
// constants covers every supported chain. The factory is the canonical
// `SimpleAccountFactory`, which is the counterpart of the EntryPoint.
var entryPointDefaults = map[string]string{
	"entry_point": "0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789",
	"factory":     "0x9406Cc6185a346906296840746125a0E44976454",
}

// rpcClients caches one JSON-RPC client per endpoint.
var (
	rpcClientsMu sync.Mutex
	rpcClients   = map[string]*rpc.Client{}
)

// rpcURLFor returns the JSON-RPC endpoint for a chain.
//
// The mapping is configured through environment variables of the form
// RPC_URL_<chainID>, so a deployment points the service at its own nodes. There
// is no built-in public endpoint: an unconfigured chain fails closed rather
// than silently depending on a third party.
func (s *AccountAbstractionService) rpcURLFor(chainID uint64) string {
	if chainID == 0 {
		chainID = s.defaultChainID
	}

	if url := os.Getenv(fmt.Sprintf("RPC_URL_%d", chainID)); url != "" {
		return url
	}
	if url := os.Getenv("RPC_URL"); url != "" {
		return url
	}

	return ""
}

// rpcClientFor returns a cached JSON-RPC client for an endpoint.
func rpcClientFor(url string) *rpc.Client {
	rpcClientsMu.Lock()
	defer rpcClientsMu.Unlock()

	if client, ok := rpcClients[url]; ok {
		return client
	}

	client, err := rpc.Dial(url)
	if err != nil {
		// The dial is lazy for HTTP, so a failure here means the URL itself is
		// unusable. Callers surface the resulting request error.
		client = nil
	}

	rpcClients[url] = client
	return client
}

// ethCallObject is the transaction object accepted by eth_call and
// eth_estimateGas.
type ethCallObject struct {
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
	Gas  string `json:"gas,omitempty"`
	Data string `json:"data,omitempty"`
}

// ethClientFor returns an ethclient for a chain, for typed contract calls.
func (s *AccountAbstractionService) ethClientFor(chainID uint64) (*ethclient.Client, error) {
	url := s.rpcURLFor(chainID)
	if url == "" {
		return nil, fmt.Errorf("no RPC endpoint configured for chain %d", chainID)
	}

	client, err := ethclient.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("cannot reach the RPC endpoint for chain %d: %w", chainID, err)
	}

	return client, nil
}

// callContract performs an eth_call and returns the raw return data.
func callContract(ctx context.Context, rpcURL, to string, data []byte) ([]byte, error) {
	client := rpcClientFor(rpcURL)
	if client == nil {
		return nil, fmt.Errorf("no usable JSON-RPC client for %s", rpcURL)
	}

	call := ethCallObject{
		To:   to,
		Data: hexutil.Encode(data),
	}

	var result hexutil.Bytes
	if err := client.CallContext(ctx, &result, "eth_call", call, "latest"); err != nil {
		return nil, fmt.Errorf("eth_call to %s failed: %w", to, err)
	}

	return result, nil
}

// computeAccountAddress asks the deployed factory for the counterfactual
// address of an owner and salt.
//
// The canonical factory derives the address from CREATE2 over its proxy
// creation code, which is large contract bytecode. Rather than reimplement that
// encoding off-chain, the service calls `getAddress(owner, salt)`, so the value
// it reports is the one the factory will actually deploy to.
func (s *AccountAbstractionService) computeAccountAddress(ctx context.Context, owner string, salt *big.Int, chainID uint64) (string, error) {
	if s.factoryAddress == "" {
		return "", fmt.Errorf("no account factory is configured")
	}
	if !common.IsHexAddress(owner) {
		return "", fmt.Errorf("%s is not a valid address", owner)
	}

	selector := crypto.Keccak256([]byte("getAddress(address,uint256)"))[:4]

	data := make([]byte, 0, 4+64)
	data = append(data, selector...)
	data = append(data, common.LeftPadBytes(common.HexToAddress(owner).Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(salt.Bytes(), 32)...)

	rpcURL := s.rpcURLFor(chainID)
	if rpcURL == "" {
		return "", fmt.Errorf("no RPC endpoint configured for chain %d", chainID)
	}

	returned, err := callContract(ctx, rpcURL, s.factoryAddress, data)
	if err != nil {
		return "", err
	}

	if len(returned) < 32 {
		return "", fmt.Errorf("getAddress returned %d bytes, expected at least 32", len(returned))
	}

	address := common.BytesToAddress(returned[12:32])
	if address == (common.Address{}) {
		return "", fmt.Errorf("the factory returned the zero address")
	}

	return address.Hex(), nil
}

// generateUserOpHash computes the ERC-4337 user operation hash.
//
// The hash covers the packed operation, the EntryPoint and the chain id, which
// is exactly what the deployed EntryPoint computes, so a signature over it is
// valid on-chain.
func (s *AccountAbstractionService) generateUserOpHash(userOp *UserOperation) string {
	op, err := userOp.toABI()
	if err != nil {
		// A malformed operation has no hash. Returning a hash over the raw
		// strings would produce a value the EntryPoint would reject.
		return ""
	}

	entryPoint := common.HexToAddress(s.entryPointAddress)
	chainID := new(big.Int).SetUint64(userOp.chainIDOr(s.defaultChainID))

	var hash common.Hash
	if s.usesPackedEncoding() {
		hash = abi.UserOpHashV07(op, entryPoint, chainID)
	} else {
		hash = abi.UserOpHashV06(op, entryPoint, chainID)
	}

	return hash.Hex()
}

// usesPackedEncoding reports whether the configured EntryPoint packs the gas
// fields into two words (v0.7 and later) rather than keeping one word per field
// (v0.6).
func (s *AccountAbstractionService) usesPackedEncoding() bool {
	return common.HexToAddress(s.entryPointAddress) != abi.EntryPointV06
}

// toABI converts the wire form of a user operation into the ABI form.
func (u *UserOperation) toABI() (*abi.UserOperation, error) {
	if !common.IsHexAddress(u.Sender) {
		return nil, fmt.Errorf("sender %q is not a valid address", u.Sender)
	}

	parse := func(name, value string) (*big.Int, error) {
		if value == "" {
			return new(big.Int), nil
		}
		v, ok := new(big.Int).SetString(value, 10)
		if !ok {
			return nil, fmt.Errorf("%s %q is not a number", name, value)
		}
		return v, nil
	}

	nonce, err := parse("nonce", u.Nonce)
	if err != nil {
		return nil, err
	}
	callGasLimit, err := parse("callGasLimit", u.CallGasLimit)
	if err != nil {
		return nil, err
	}
	verificationGasLimit, err := parse("verificationGasLimit", u.VerificationGasLimit)
	if err != nil {
		return nil, err
	}
	preVerificationGas, err := parse("preVerificationGas", u.PreVerificationGas)
	if err != nil {
		return nil, err
	}
	maxFeePerGas, err := parse("maxFeePerGas", u.MaxFeePerGas)
	if err != nil {
		return nil, err
	}
	maxPriorityFeePerGas, err := parse("maxPriorityFeePerGas", u.MaxPriorityFeePerGas)
	if err != nil {
		return nil, err
	}

	decode := func(name, value string) ([]byte, error) {
		if value == "" {
			return nil, nil
		}
		bytes, err := hexutil.Decode(value)
		if err != nil {
			return nil, fmt.Errorf("%s %q is not valid hex: %w", name, value, err)
		}
		return bytes, nil
	}

	initCode, err := decode("initCode", u.InitCode)
	if err != nil {
		return nil, err
	}
	callData, err := decode("callData", u.CallData)
	if err != nil {
		return nil, err
	}
	paymasterAndData, err := decode("paymasterAndData", u.PaymasterAndData)
	if err != nil {
		return nil, err
	}
	signature, err := decode("signature", u.Signature)
	if err != nil {
		return nil, err
	}

	return &abi.UserOperation{
		Sender:               common.HexToAddress(u.Sender),
		Nonce:                nonce,
		InitCode:             initCode,
		CallData:             callData,
		CallGasLimit:         callGasLimit,
		VerificationGasLimit: verificationGasLimit,
		PreVerificationGas:   preVerificationGas,
		MaxFeePerGas:         maxFeePerGas,
		MaxPriorityFeePerGas: maxPriorityFeePerGas,
		PaymasterAndData:     paymasterAndData,
		Signature:            signature,
	}, nil
}

// chainIDOr returns the operation's chain id, or the fallback when unset.
func (u *UserOperation) chainIDOr(fallback uint64) uint64 {
	if u.ChainID != 0 {
		return u.ChainID
	}
	return fallback
}

// normalizeAddress checks a value is a usable address and returns it in
// checksummed form.
func normalizeAddress(value string) (string, error) {
	if !common.IsHexAddress(value) {
		return "", fmt.Errorf("%q is not a valid address", value)
	}
	return common.HexToAddress(value).Hex(), nil
}

// trimHexPrefix removes a leading 0x from a hex string.
func trimHexPrefix(value string) string {
	return strings.TrimPrefix(value, "0x")
}

// defaultChainID returns the chain the service targets when an operation does
// not name one. It is configurable so a deployment can default to its own
// network.
func defaultChainID() uint64 {
if value := os.Getenv("AA_DEFAULT_CHAIN_ID"); value != "" {
if parsed, err := strconv.ParseUint(value, 10, 64); err == nil && parsed != 0 {
return parsed
}
}
return 1
}
