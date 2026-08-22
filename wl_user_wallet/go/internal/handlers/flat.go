package handlers

import (
	"context"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tigerwallet/wl-user-wallet/internal/crypto"
	"github.com/tigerwallet/wl-user-wallet/internal/middleware"
)

// GET /balance?wallet_id=&address=&chain_id= — flat canonical balance read.
// Real eth_getBalance. Fail-closed 503 if no RPC. If wallet_id is provided,
// the wallet's own chain/address is used (must be owned by caller).
func (s *Svc) FlatBalance(c *gin.Context) {
	addr := c.Query("address")
	chainID, _ := strconv.ParseInt(c.Query("chain_id"), 10, 64)
	if wID := c.Query("wallet_id"); wID != "" {
		id, err := uuid.Parse(wID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wallet_id"})
			return
		}
		w, err := s.store.GetWallet(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
			return
		}
		if w.UserID != middleware.UserID(c) {
			c.JSON(http.StatusForbidden, gin.H{"error": "not your wallet"})
			return
		}
		addr = w.Address
		chainID = w.ChainID
	}
	if addr == "" || chainID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "address and chain_id (or wallet_id) required"})
		return
	}
	rpc := rpcForChain(chainID)
	if rpc == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no RPC configured for chain"})
		return
	}
	client, err := ethclient.DialContext(c.Request.Context(), rpc)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "RPC unavailable"})
		return
	}
	defer client.Close()
	bal, err := client.BalanceAt(c.Request.Context(), common.HexToAddress(addr), nil)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "balance fetch failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"address": addr, "balance_wei": bal.String(), "chain_id": chainID})
}

// POST /send — flat canonical send. Accepts wallet_id in the body and uses
// the wallet's own chain/signer. Real EIP-1559 signing + broadcast.
func (s *Svc) FlatSend(c *gin.Context) {
	var req struct {
		WalletID     uuid.UUID `json:"wallet_id" binding:"required"`
		To           string    `json:"to" binding:"required"`
		Amount       string    `json:"amount" binding:"required"`
		Password     string    `json:"password" binding:"required"`
		GasLimit     uint64    `json:"gas_limit"`
		Token        string    `json:"token"`
		WithdrawalID string    `json:"withdrawal_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	w, err := s.store.GetWallet(c.Request.Context(), req.WalletID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	if w.UserID != middleware.UserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your wallet"})
		return
	}
	wid, ok := s.requireApproval(c, "transfer", req.To, req.Token, req.Amount, req.WithdrawalID)
	if !ok {
		return
	}
	seed, err := crypto.DecryptSeedAtRest(w.EncryptedSeed, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid password"})
		return
	}
	priv, err := crypto.DeriveEVMPrivateKey(seed, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "key derivation failed"})
		return
	}
	rpc := rpcForChain(w.ChainID)
	if rpc == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no RPC configured for chain"})
		return
	}
	client, err := ethclient.DialContext(c.Request.Context(), rpc)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "RPC unavailable"})
		return
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	nonce, err := client.PendingNonceAt(ctx, common.HexToAddress(w.Address))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "nonce fetch failed"})
		return
	}
	amount, ok := new(big.Float).SetString(req.Amount)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
		return
	}
	amount = amount.Mul(amount, big.NewFloat(1e18))
	amountInt, _ := amount.Int(nil)
	chainID := big.NewInt(w.ChainID)
	gasTipCap, _ := client.SuggestGasTipCap(ctx)
	head, _ := client.HeaderByNumber(ctx, nil)
	if head == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "head fetch failed"})
		return
	}
	rawTx, err := crypto.SignTransaction(priv, chainID, common.HexToAddress(req.To), amountInt, req.GasLimit, head.BaseFee, gasTipCap, nonce, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	txHash, err := broadcastRawTx(ctx, client, rawTx)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "broadcast failed: " + err.Error()})
		return
	}
	_ = s.store.CreateTransaction(c.Request.Context(), w.ID, txHash, "transfer", "broadcast", w.Address, req.To, req.Amount, "", w.ChainID)
	if wid != uuid.Nil {
		if g := middleware.GetTwoPartyGate(); g != nil {
			_ = g.MarkWithdrawalExecuted(c.Request.Context(), wid, txHash)
		}
	}
	c.JSON(http.StatusOK, gin.H{"transaction_hash": txHash, "status": "broadcast", "from": w.Address})
}

// POST /sign — flat canonical EIP-191 sign. wallet_id in body.
func (s *Svc) FlatSign(c *gin.Context) {
	var req struct {
		WalletID uuid.UUID `json:"wallet_id" binding:"required"`
		Message  string    `json:"message" binding:"required"`
		Password string    `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	w, err := s.store.GetWallet(c.Request.Context(), req.WalletID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	if w.UserID != middleware.UserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your wallet"})
		return
	}
	// personal_sign is non-value => Auto mode (license-alive = approved).
	if _, ok := s.requireApproval(c, "personal_sign", "", "", "", ""); !ok {
		return
	}
	seed, err := crypto.DecryptSeedAtRest(w.EncryptedSeed, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid password"})
		return
	}
	priv, err := crypto.DeriveEVMPrivateKey(seed, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "key derivation failed"})
		return
	}
	sig, err := crypto.SignMessage(priv, req.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"signature": sig, "address": w.Address})
}

// GET /transactions?wallet_id= — flat canonical list (wallet_id in query).
func (s *Svc) FlatTransactions(c *gin.Context) {
	id, err := uuid.Parse(c.Query("wallet_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wallet_id required"})
		return
	}
	w, err := s.store.GetWallet(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	if w.UserID != middleware.UserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your wallet"})
		return
	}
	txs, err := s.store.ListTransactions(c.Request.Context(), w.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"transactions": txs})
}
