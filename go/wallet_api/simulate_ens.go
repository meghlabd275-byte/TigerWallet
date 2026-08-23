package main

// simulate_ens.go — REAL transaction simulation and ENS resolution against live
// chain RPC endpoints. Simulation estimates gas and dry-runs the call via
// eth_estimateGas + eth_call so the client can show success/failure and a revert
// reason BEFORE the user signs. ENS resolution performs real on-chain lookups
// against the canonical ENS registry on Ethereum mainnet. No mocks, no stubs.

import (
        "context"
        "fmt"
        "math/big"
        "net/http"
        "strings"
        "time"

        "github.com/ethereum/go-ethereum/common"
        "github.com/ethereum/go-ethereum/common/hexutil"
        "github.com/ethereum/go-ethereum/crypto"
        "github.com/gin-gonic/gin"
)

// ensRegistry is the canonical ENS registry (immutable on Ethereum mainnet).
const ensRegistry = "0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e"

// ensMainnetRPC is the public chain ENS lives on. Resolution always targets
// Ethereum mainnet regardless of the chain the user is sending on.
var ensMainnetRPC = chainsForNetworkDefault("ethereum")

// ---- namehash (EIP-137) ----

// namehash computes the ENS namehash for a dotted name (e.g. "alice.eth").
func namehash(name string) []byte {
        node := make([]byte, 32)
        if name == "" {
                return node
        }
        labels := strings.Split(strings.ToLower(name), ".")
        for i := len(labels) - 1; i >= 0; i-- {
                labelHash := crypto.Keccak256([]byte(labels[i]))
                node = crypto.Keccak256(append(node, labelHash...))
        }
        return node
}

// encode strings/bytes32 for eth_call.
func pad32bytes(b []byte) []byte {
        out := make([]byte, 32)
        copy(out[32-len(b):], b)
        return out
}

// resolver(node) selector 0x0178b8bf
func ensResolverData(node []byte) []byte {
        return append(common.Hex2Bytes("0178b8bf"), pad32bytes(node)...)
}

// addr(bytes32 node) selector 0x3b3b57de
func ensAddrData(node []byte) []byte {
        return append(common.Hex2Bytes("3b3b57de"), pad32bytes(node)...)
}

// name(bytes32 node) selector 0x691f3431 (reverse record lookup)
func ensNameData(node []byte) []byte {
        return append(common.Hex2Bytes("691f3431"), pad32bytes(node)...)
}

func chainsForNetworkDefault(name string) string {
        for _, ch := range SupportedChains {
                if ch.IsEVM() && strings.EqualFold(ch.Name, name) {
                        return ch.RPCEndpoint
                }
        }
        // Fall back to chain id 1 if present.
        for _, ch := range SupportedChains {
                if ch.IsEVM() && ch.ID == 1 {
                        return ch.RPCEndpoint
                }
        }
        return ""
}

// resolveENS resolves an ENS name to an Ethereum address using the registry +
// resolver eth_call sequence on Ethereum mainnet.
func resolveENS(ctx context.Context, name string) (common.Address, error) {
        rpc := ensMainnetRPC
        if rpc == "" {
                return common.Address{}, fmt.Errorf("no mainnet RPC configured for ENS")
        }
        node := namehash(name)
        var zero common.Address

        // Step 1: registry -> resolver address for the name.
        resOut, err := ethCall(ctx, rpc, common.HexToAddress(ensRegistry), ensResolverData(node))
        if err != nil {
                return zero, fmt.Errorf("registry resolver lookup: %w", err)
        }
        if len(resOut) < 32 {
                return zero, fmt.Errorf("no resolver for %s", name)
        }
        var resolver common.Address
        copy(resolver[:], resOut[len(resOut)-20:])
        if resolver == zero {
                return zero, fmt.Errorf("no resolver for %s", name)
        }

        // Step 2: resolver -> addr(node).
        addrOut, err := ethCall(ctx, rpc, resolver, ensAddrData(node))
        if err != nil {
                return zero, fmt.Errorf("resolver addr lookup: %w", err)
        }
        if len(addrOut) < 32 {
                return zero, fmt.Errorf("unresolved %s", name)
        }
        var addr common.Address
        copy(addr[:], addrOut[len(addrOut)-20:])
        if addr == zero {
                return zero, fmt.Errorf("unresolved %s", name)
        }
        return addr, nil
}

// decodeENSString decodes an ABI-encoded string (offset + len + data).
func decodeENSString(data []byte) string {
        if len(data) < 64 {
                return ""
        }
        length := new(big.Int).SetBytes(data[32:64]).Int64()
        if length <= 0 || 64+length > int64(len(data)) {
                return ""
        }
        return strings.TrimRight(string(data[64:64+length]), "\x00")
}

// handleENSResolve resolves an ENS name to an address.
// GET /api/v1/ens/resolve?name=alice.eth
func handleENSResolve(c *gin.Context) {
        name := strings.TrimSpace(c.Query("name"))
        if name == "" || !strings.HasSuffix(strings.ToLower(name), ".eth") {
                c.JSON(http.StatusBadRequest, gin.H{"error": "name must end in .eth"})
                return
        }
        ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
        defer cancel()
        addr, err := resolveENS(ctx, name)
        if err != nil {
                c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
                return
        }
        c.JSON(http.StatusOK, gin.H{"name": name, "address": addr.Hex()})
}

// handleENSLookup does a reverse ENS lookup for an address.
// GET /api/v1/ens/lookup?address=0x...
func handleENSLookup(c *gin.Context) {
        address := strings.TrimSpace(c.Query("address"))
        if !common.IsHexAddress(address) {
                c.JSON(http.StatusBadRequest, gin.H{"error": "invalid address"})
                return
        }
        rpc := ensMainnetRPC
        if rpc == "" {
                c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no mainnet RPC configured"})
                return
        }
        ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
        defer cancel()

        reverseName := strings.TrimPrefix(strings.ToLower(address), "0x") + ".addr.reverse"
        node := namehash(reverseName)

        // registry -> resolver
        resOut, err := ethCall(ctx, rpc, common.HexToAddress(ensRegistry), ensResolverData(node))
        if err != nil || len(resOut) < 32 {
                c.JSON(http.StatusNotFound, gin.H{"error": "no reverse resolver"})
                return
        }
        var resolver common.Address
        copy(resolver[:], resOut[len(resOut)-20:])
        if resolver == (common.Address{}) {
                c.JSON(http.StatusNotFound, gin.H{"error": "no reverse resolver"})
                return
        }
        // resolver -> name(node)
        nameOut, err := ethCall(ctx, rpc, resolver, ensNameData(node))
        if err != nil {
                c.JSON(http.StatusNotFound, gin.H{"error": "lookup failed: " + err.Error()})
                return
        }
        name := decodeENSString(nameOut)
        if name == "" {
                c.JSON(http.StatusNotFound, gin.H{"error": "no reverse record"})
                return
        }
        c.JSON(http.StatusOK, gin.H{"address": common.HexToAddress(address).Hex(), "name": name})
}

// ---- Transaction simulation ----

type simulateReq struct {
        ChainID int64  `json:"chain_id" binding:"required"`
        From    string `json:"from" binding:"required"`
        To      string `json:"to" binding:"required"`
        Value   string `json:"value"`   // human-readable ether (optional)
        Data    string `json:"data"`    // hex-encoded calldata (optional)
}

// EstimateGas performs a real eth_estimateGas against the chain RPC.
func estimateGas(ctx context.Context, endpoint string, from, to common.Address, value *big.Int, data []byte) (uint64, error) {
        client, err := rpcClient(endpoint)
        if err != nil {
                return 0, err
        }
        defer client.Close()
        call := map[string]interface{}{
                "from": from.Hex(),
                "to":   to.Hex(),
        }
        if value != nil && value.Sign() > 0 {
                call["value"] = hexutil.EncodeBig(value)
        }
        if len(data) > 0 {
                call["data"] = hexutil.Encode(data)
        }
        var result hexutil.Uint64
        if err := client.CallContext(ctx, &result, "eth_estimateGas", call, "latest"); err != nil {
                return 0, err
        }
        return uint64(result), nil
}

// dryRunCall performs an eth_call with an explicit value/from to predict
// success/failure and capture any revert reason.
func dryRunCall(ctx context.Context, endpoint string, from, to common.Address, value *big.Int, data []byte) (string, error) {
        client, err := rpcClient(endpoint)
        if err != nil {
                return "", err
        }
        defer client.Close()
        call := map[string]interface{}{
                "from": from.Hex(),
                "to":   to.Hex(),
        }
        if value != nil && value.Sign() > 0 {
                call["value"] = hexutil.EncodeBig(value)
        }
        if len(data) > 0 {
                call["data"] = hexutil.Encode(data)
        }
        var result hexutil.Bytes
        err = client.CallContext(ctx, &result, "eth_call", call, "latest")
        return hexutil.Encode(result), err
}

// handleSimulateTransaction dry-runs a transaction to predict outcome + gas.
// POST /api/v1/simulate
func handleSimulateTransaction(c *gin.Context) {
        var req simulateReq
        if err := c.ShouldBindJSON(&req); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }
        if !common.IsHexAddress(req.From) || !common.IsHexAddress(req.To) {
                c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from/to address"})
                return
        }
        chain := evmChainByChainID(req.ChainID)
        if chain == nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or non-EVM chain_id"})
                return
        }
        from := common.HexToAddress(req.From)
        to := common.HexToAddress(req.To)

        var value *big.Int
        if strings.TrimSpace(req.Value) != "" {
                f, ok := new(big.Float).SetString(req.Value)
                if !ok {
                        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid value"})
                        return
                }
                wei := new(big.Float).Mul(f, big.NewFloat(1e18))
                i, _ := wei.Int(nil)
                value = i
        } else {
                value = big.NewInt(0)
        }

        var data []byte
        if strings.TrimSpace(req.Data) != "" {
                d, err := hexutil.Decode(req.Data)
                if err != nil {
                        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid calldata hex"})
                        return
                }
                data = d
        }

        ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
        defer cancel()

        // Gas estimate.
        gasEstimate, estErr := estimateGas(ctx, chain.RPCEndpoint, from, to, value, data)
        // Dry-run the call to detect reverts.
        _, callErr := dryRunCall(ctx, chain.RPCEndpoint, from, to, value, data)

        // Gas pricing for cost projection.
        gasPrice, maxFee, maxPrioFee, gpErr := FetchGasPrice(ctx, chain.RPCEndpoint)

        success := callErr == nil && estErr == nil
        resp := gin.H{
                "chain_id":      req.ChainID,
                "success":       success,
                "gas_estimate":  gasEstimate,
                "will_revert":   callErr != nil,
        }
        if callErr != nil {
                resp["revert_reason"] = callErr.Error()
        }
        if estErr != nil {
                resp["estimate_error"] = estErr.Error()
        }
        if gpErr == nil {
                resp["gas_price"] = gasPrice.String()
                resp["max_fee_per_gas"] = maxFee.String()
                resp["max_priority_fee"] = maxPrioFee.String()
                // projected cost in wei at the safe max fee.
                cost := new(big.Int).Mul(new(big.Int).SetUint64(gasEstimate), maxFee)
                resp["estimated_cost_wei"] = cost.String()
        }
        c.JSON(http.StatusOK, resp)
}
