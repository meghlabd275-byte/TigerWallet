package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tigerwallet/wl-user-wallet/internal/onchain"
)

// GET /transactions/:txHash?chain_id= — real tx receipt via the chain's RPC
// (eth_getTransactionReceipt). Fail-closed 503 if no RPC configured.
func (s *Svc) GetTransaction(c *gin.Context) {
	txHash := c.Param("txHash")
	chainID, _ := strconv.ParseInt(c.Query("chain_id"), 10, 64)
	if txHash == "" || chainID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "txHash and chain_id required"})
		return
	}
	rpc := rpcForChain(chainID)
	if rpc == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no RPC configured for chain"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	receipt, err := onchain.FetchTxReceipt(ctx, rpc, txHash)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "receipt fetch failed: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, receipt)
}
