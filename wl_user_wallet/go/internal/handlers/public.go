package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tigerwallet/wl-user-wallet/internal/onchain"
)

// GET /public/balance?address=&chain_id= — unauthenticated native balance read.
// Real eth_getBalance. Fail-closed 503 if no RPC.
func (s *Svc) PublicBalance(c *gin.Context) {
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
	ctx, cancel := context.WithTimeout(c.Request.Context(), 12*time.Second)
	defer cancel()
	bal, err := onchain.FetchBalance(ctx, rpc, address)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "balance fetch failed: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"chain_id": chainID, "address": address, "balance": bal.String()})
}

// GET /public/tokens?address=&chain_id= — unauthenticated ERC-20 balances.
func (s *Svc) PublicTokens(c *gin.Context) {
	s.GetTokens(c)
}

// GET /public/nfts?address=&chain_id= — unauthenticated NFT holdings.
func (s *Svc) PublicNFTs(c *gin.Context) {
	s.GetNFTs(c)
}

// GET /public/transactions?address=&chain_id= — unauthenticated recent tx
// history via the chain's explorer API (real Etherscan-compatible HTTP).
func (s *Svc) PublicTransactions(c *gin.Context) {
	address := c.Query("address")
	chainID, _ := strconv.ParseInt(c.Query("chain_id"), 10, 64)
	if address == "" || chainID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "address and chain_id required"})
		return
	}
	explorer := explorerForChain(chainID)
	if explorer == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no explorer API configured for chain"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	txs, err := onchain.FetchTransactionHistory(ctx, explorer, s.cfg.EtherscanAPIKey, address, chainID)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "tx history fetch failed: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"chain_id": chainID, "address": address, "transactions": txs})
}
