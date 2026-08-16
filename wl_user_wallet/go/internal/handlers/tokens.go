package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tigerwallet/wl-user-wallet/internal/onchain"
)

// GET /tokens?address=&chain_id= — real ERC-20 balanceOf eth_call across the
// token registry for that chain. Fail-closed 503 if no RPC configured.
func (s *Svc) GetTokens(c *gin.Context) {
	address := c.Query("address")
	chainID, _ := strconv.ParseInt(c.Query("chain_id"), 10, 64)
	if address == "" || chainID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "address and chain_id required"})
		return
	}
	rpc := rpcForChain(chainID)
	if rpc == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no RPC configured for chain"})
		return
	}
	holder, err := onchain.HexAddress(address)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	tokens := onchain.TokensForChain(chainID)
	balances := onchain.FetchTokenBalances(ctx, rpc, holder, tokens)
	c.JSON(http.StatusOK, gin.H{"chain_id": chainID, "address": address, "tokens": balances})
}
