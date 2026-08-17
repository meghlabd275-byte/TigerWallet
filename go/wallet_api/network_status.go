package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
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
	rpc := chain.RPCEndpoint
	if rpc == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no RPC endpoint configured for chain"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()

	start := time.Now()
	client, err := ethclient.DialContext(ctx, rpc)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "RPC dial failed", "detail": err.Error()})
		return
	}
	defer client.Close()

	blockNum, err := client.BlockNumber(ctx)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "eth_blockNumber failed", "detail": err.Error()})
		return
	}

	chainIDResult, err := client.ChainID(ctx)
	if err != nil {
		chainIDResult = nil
	}
	syncing := false
	if progress, err := client.SyncProgress(ctx); err == nil && progress != nil {
		// A non-nil SyncProgress means the node is syncing; nil = synced.
		syncing = progress.CurrentBlock < progress.HighestBlock
	}

	resp := networkStatusResponse{
		ChainID:        chainID,
		BlockNumber:    fmt.Sprintf("0x%x", blockNum),
		BlockNumberInt: blockNum,
		Syncing:        syncing,
		RPCEndpoint:    rpc,
		LatencyMS:      latency,
		Timestamp:      time.Now().Unix(),
	}
	if chainIDResult != nil {
		resp.ChainID = chainIDResult.Int64()
	}
	c.JSON(http.StatusOK, resp)
}
