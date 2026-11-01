package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/gin-gonic/gin"
)

// networkStatusResponse is the real per-chain network status returned by
// GET /api/v1/network-status?chain_id=N. It performs a REAL eth_blockNumber +
// eth_chainId RPC call against the chain's configured RPC endpoint — never a
// fabricated block number.
type networkStatusResponse struct {
	ChainID        int64  `json:"chain_id"`
	BlockNumber    string `json:"block_number"` // hex string, as returned by eth_blockNumber
	BlockNumberInt uint64 `json:"block_number_int"`
	Syncing        bool   `json:"syncing"`
	RPCEndpoint    string `json:"rpc_endpoint"`
	LatencyMS      int64  `json:"latency_ms"`
	Timestamp      int64  `json:"timestamp"`
}

// handleNetworkStatus performs a real eth_blockNumber RPC call against the
// requested chain's RPC endpoint. This closes Gap G: previously all clients
// derived a fake block_number:0 from /chains. Now they call this real endpoint.
//
// Auth: public (read-only chain status, like /chains and /gas).
func handleNetworkStatus(c *gin.Context) {
	chainIDStr := c.Query("chain_id")
	if chainIDStr == "" {
		chainIDStr = "1" // default to Ethereum mainnet
	}
	chainID, err := strconv.ParseInt(chainIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chain_id"})
		return
	}
	chain := chainByID(chainID)
	if chain == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "unsupported chain"})
		return
	}
	if !chain.IsEVM() {
		// Non-EVM chains don't expose eth_blockNumber; return honestly.
		c.JSON(http.StatusOK, gin.H{
			"chain_id":     chainID,
			"block_number": "0",
			"note":         "non-EVM chain; block number requires chain-native RPC",
			"chain_type":   chain.ChainType,
		})
		return
	}
	endpoint := chain.RPCEndpoint
	if endpoint == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no RPC endpoint configured for chain"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()

	start := time.Now()
	var blockNum uint64
	err = rpcCall(ctx, endpoint, func(client *rpc.Client) error {
		var bn hexutil.Uint64
		if err := client.CallContext(ctx, &bn, "eth_blockNumber"); err != nil {
			return err
		}
		blockNum = uint64(bn)
		return nil
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "eth_blockNumber failed", "detail": err.Error()})
		return
	}
	latency := time.Since(start).Milliseconds()

	// eth_chainId and eth_syncing are best-effort: a node that does not expose
	// them still reports a real block number, and inventing a value here would
	// misreport the network.
	var chainIDResult *big.Int
	if err := rpcCall(ctx, endpoint, func(client *rpc.Client) error {
		var id hexutil.Big
		if err := client.CallContext(ctx, &id, "eth_chainId"); err != nil {
			return err
		}
		chainIDResult = id.ToInt()
		return nil
	}); err != nil {
		chainIDResult = nil
	}
	syncing := false
	if err := rpcCall(ctx, endpoint, func(client *rpc.Client) error {
		var progress *struct {
			CurrentBlock uint64 `json:"currentBlock"`
			HighestBlock uint64 `json:"highestBlock"`
		}
		if err := client.CallContext(ctx, &progress, "eth_syncing"); err != nil {
			return err
		}
		// A non-nil result means the node is syncing; false means synced.
		if progress != nil {
			syncing = progress.CurrentBlock < progress.HighestBlock
		}
		return nil
	}); err != nil {
		syncing = false
	}

	resp := networkStatusResponse{
		ChainID:        chainID,
		BlockNumber:    fmt.Sprintf("0x%x", blockNum),
		BlockNumberInt: blockNum,
		Syncing:        syncing,
		RPCEndpoint:    endpoint,
		LatencyMS:      latency,
		Timestamp:      time.Now().Unix(),
	}
	if chainIDResult != nil {
		resp.ChainID = chainIDResult.Int64()
	}
	c.JSON(http.StatusOK, resp)
}
