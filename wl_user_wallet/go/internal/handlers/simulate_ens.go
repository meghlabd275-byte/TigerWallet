package handlers

// simulate_ens.go — REAL transaction simulation and ENS resolution for the
// standalone WL-UserWallet backend. Mirrors go/wallet_api/simulate_ens.go so
// the Android/iOS clients (which target this service) have feature parity with
// the web/desktop/extension clients (which target go/wallet_api). Simulation
// uses eth_estimateGas + eth_call against the live chain RPC; ENS uses the
// canonical registry on Ethereum mainnet. No mocks, no stubs.

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/tigerwallet/wl-user-wallet/internal/chains"
	"github.com/tigerwallet/wl-user-wallet/internal/onchain"
)

// ensRegistry is the canonical ENS registry (immutable on Ethereum mainnet).
const ensRegistry = "0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e"

func ensMainnetRPCEndpoint() string { return rpcForChain(1) }

// gweiToWeiString parses a gwei string (e.g. "20", "20.5") into wei (big.Int).
// Returns nil for empty/unparsable input so callers fall back to chain fees.
func gweiToWeiString(gwei string) *big.Int {
	gwei = strings.TrimSpace(gwei)
	if gwei == "" {
		return nil
	}
	f, ok := new(big.Float).SetString(gwei)
	if !ok || f.Sign() < 0 {
		return nil
	}
	wei, _ := new(big.Float).Mul(f, big.NewFloat(1e9)).Int(nil)
	return wei
}

// namehash computes the ENS namehash for a dotted name (EIP-137).
func ensNamehash(name string) []byte {
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

func ensPad32(b []byte) []byte {
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out
}

// resolveENS resolves an ENS name to an address via registry+resolver eth_call.
func resolveENS(ctx context.Context, name string) (common.Address, error) {
	rpc := ensMainnetRPCEndpoint()
	if rpc == "" {
		return common.Address{}, fmt.Errorf("no mainnet RPC configured for ENS")
	}
	node := ensNamehash(name)
	var zero common.Address

	resOut, err := onchain.EthCall(ctx, rpc, common.HexToAddress(ensRegistry),
		append(common.Hex2Bytes("0178b8bf"), ensPad32(node)...))
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

	addrOut, err := onchain.EthCall(ctx, rpc, resolver,
		append(common.Hex2Bytes("3b3b57de"), ensPad32(node)...))
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

func decodeENSStr(data []byte) string {
	if len(data) < 64 {
		return ""
	}
	length := new(big.Int).SetBytes(data[32:64]).Int64()
	if length <= 0 || 64+length > int64(len(data)) {
		return ""
	}
	return strings.TrimRight(string(data[64:64+length]), "\x00")
}

// GET /ens/resolve?name=alice.eth -> { name, address }
func (s *Svc) ENSResolve(c *gin.Context) {
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

// GET /ens/lookup?address=0x... -> { address, name } (reverse ENS).
func (s *Svc) ENSLookup(c *gin.Context) {
	address := strings.TrimSpace(c.Query("address"))
	if !common.IsHexAddress(address) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid address"})
		return
	}
	rpc := ensMainnetRPCEndpoint()
	if rpc == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no mainnet RPC configured"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	node := ensNamehash(strings.TrimPrefix(strings.ToLower(address), "0x") + ".addr.reverse")
	resOut, err := onchain.EthCall(ctx, rpc, common.HexToAddress(ensRegistry),
		append(common.Hex2Bytes("0178b8bf"), ensPad32(node)...))
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
	nameOut, err := onchain.EthCall(ctx, rpc, resolver,
		append(common.Hex2Bytes("691f3431"), ensPad32(node)...))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "lookup failed: " + err.Error()})
		return
	}
	name := decodeENSStr(nameOut)
	if name == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "no reverse record"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"address": common.HexToAddress(address).Hex(), "name": name})
}

// POST /simulate — dry-run a transaction before signing (success/gas/revert).
func (s *Svc) SimulateTransaction(c *gin.Context) {
	var req struct {
		ChainID int64  `json:"chain_id" binding:"required"`
		From    string `json:"from" binding:"required"`
		To      string `json:"to" binding:"required"`
		Value   string `json:"value"`
		Data    string `json:"data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !common.IsHexAddress(req.From) || !common.IsHexAddress(req.To) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from/to address"})
		return
	}
	if cfg := chains.ChainByID(req.ChainID); cfg == nil || !cfg.IsEVM() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or non-EVM chain_id"})
		return
	}
	rpc := rpcForChain(req.ChainID)
	if rpc == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no RPC configured for chain"})
		return
	}
	from := common.HexToAddress(req.From)
	to := common.HexToAddress(req.To)

	value := big.NewInt(0)
	if strings.TrimSpace(req.Value) != "" {
		f, ok := new(big.Float).SetString(req.Value)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid value"})
			return
		}
		i, _ := new(big.Float).Mul(f, big.NewFloat(1e18)).Int(nil)
		value = i
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

	client, err := ethclient.DialContext(ctx, rpc)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "RPC unavailable"})
		return
	}
	defer client.Close()

	call := ethereum.CallMsg{From: from, To: &to, Value: value, Data: data}
	gasEstimate, estErr := client.EstimateGas(ctx, call)
	_, callErr := client.CallContract(ctx, call, nil)
	gasPrice, maxFee, maxPrioFee, gpErr := onchain.FetchGasPrice(ctx, rpc)

	success := callErr == nil && estErr == nil
	resp := gin.H{
		"chain_id":     req.ChainID,
		"success":      success,
		"gas_estimate": gasEstimate,
		"will_revert":  callErr != nil,
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
		cost := new(big.Int).Mul(new(big.Int).SetUint64(gasEstimate), maxFee)
		resp["estimated_cost_wei"] = cost.String()
	}
	c.JSON(http.StatusOK, resp)
}
