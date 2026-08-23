package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tigerwallet/wl-user-wallet/internal/onchain"
)

// GET /gas?chain_id= — real eth_gasPrice + eth_maxPriorityFeePerGas from the
// chain's RPC node. Fail-closed 503 if no RPC configured.
func (s *Svc) GetGas(c *gin.Context) {
	chainID, _ := strconv.ParseInt(c.Query("chain_id"), 10, 64)
	if chainID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chain_id required"})
		return
	}
	rpc := rpcForChain(chainID)
	if rpc == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no RPC configured for chain"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	gasPrice, maxFee, prioFee, err := onchain.FetchGasPrice(ctx, rpc)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "gas fetch failed: " + err.Error()})
		return
	}
	resp := gin.H{
		"chain_id":         chainID,
		"gas_price":        gasPrice.String(),
		"max_priority_fee": nil,
		"max_fee_per_gas":  nil,
	}
	if prioFee != nil {
		resp["max_priority_fee"] = prioFee.String()
	}
	if maxFee != nil {
		resp["max_fee_per_gas"] = maxFee.String()
	}
	c.JSON(http.StatusOK, resp)
}
