package handlers

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tigerwallet/wl-user-wallet/internal/crypto"
	"github.com/tigerwallet/wl-user-wallet/internal/middleware"
	"github.com/tigerwallet/wl-user-wallet/internal/store"
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
// Optional `data` (hex calldata) turns this into a contract call; optional
// `amount_wei` sends an exact wei value (no float rounding).
func (s *Svc) FlatSend(c *gin.Context) {
	var req struct {
		WalletID     uuid.UUID `json:"wallet_id" binding:"required"`
		To           string    `json:"to" binding:"required"`
		Amount       string    `json:"amount"`
		AmountWei    string    `json:"amount_wei,omitempty"`
		Data         string    `json:"data,omitempty"`
		Password     string    `json:"password" binding:"required"`
		GasLimit     uint64    `json:"gas_limit"`
		Token        string    `json:"token"`
		WithdrawalID string    `json:"withdrawal_id"`
		// Optional EIP-1559 editable fee overrides (gwei) — parity with
		// go/wallet_api /send. When set, overrides the chain suggestion.
		MaxFeeGwei      string `json:"max_fee_gwei,omitempty"`
		MaxPriorityGwei string `json:"max_priority_gwei,omitempty"`
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
	amountStr := req.Amount
	amountInt, err := parseAmountWei(req.Amount, req.AmountWei)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.AmountWei != "" {
		amountStr = new(big.Rat).SetFrac(amountInt, big.NewInt(1e18)).FloatString(18)
	}
	var data []byte
	if req.Data != "" {
		data, err = decodeHexData(req.Data)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	gasLimit := req.GasLimit
	if gasLimit == 0 && len(data) > 0 {
		gasLimit = 120000 // contract call default; plain-transfer default is in crypto.SignTransaction
	}
	wid, ok := s.requireApproval(c, "transfer", req.To, req.Token, amountStr, req.WithdrawalID)
	if !ok {
		return
	}
	autoApproved := ok && wid == uuid.Nil
	autoReason := ""
	if ok && wid != uuid.Nil {
		autoReason = "two-party approved by SuperAdmin"
	}
	s.execFlatEVMSend(c, w, req.Password, req.To, amountInt, gasLimit, data,
		req.MaxFeeGwei, req.MaxPriorityGwei, wid, autoApproved, autoReason, "transfer", amountStr)
}

// parseAmountWei resolves the tx value: exact wei when amount_wei is given,
// otherwise decimal ether units (amount) converted to wei. Fail-closed on
// unparseable input.
func parseAmountWei(amount, amountWei string) (*big.Int, error) {
	if amountWei != "" {
		v, ok := new(big.Int).SetString(amountWei, 10)
		if !ok || v.Sign() < 0 {
			return nil, fmt.Errorf("invalid amount_wei")
		}
		return v, nil
	}
	if amount == "" {
		return nil, fmt.Errorf("amount is required")
	}
	f, ok := new(big.Float).SetString(amount)
	if !ok {
		return nil, fmt.Errorf("invalid amount")
	}
	f = f.Mul(f, big.NewFloat(1e18))
	v, _ := f.Int(nil)
	return v, nil
}

// decodeHexData decodes 0x-prefixed (or bare) hex calldata. Fail-closed.
func decodeHexData(s string) ([]byte, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "0x")
	if s == "" {
		return nil, nil
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid data: not hex")
	}
	return b, nil
}

// execFlatEVMSend is the shared sign+broadcast executor for /send and
// /nft/transfer: decrypts the seed, derives the key, signs a real EIP-1559
// transaction and broadcasts it. Fail-closed at every step — never fabricates
// a transaction hash.
func (s *Svc) execFlatEVMSend(c *gin.Context, w *store.Wallet, password, to string,
	amountInt *big.Int, gasLimit uint64, data []byte,
	maxFeeGwei, maxPriorityGwei string, wid uuid.UUID,
	autoApproved bool, autoReason, txType, amountStr string) {
	seed, err := crypto.DecryptSeedAtRest(w.EncryptedSeed, password)
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
	chainID := big.NewInt(w.ChainID)
	gasTipCap, _ := client.SuggestGasTipCap(ctx)
	head, _ := client.HeaderByNumber(ctx, nil)
	if head == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "head fetch failed"})
		return
	}
	// EIP-1559 editable fee overrides (gwei -> wei). Defaults: fee cap is
	// 2*baseFee + tip (standard safe ceiling), tip from the node suggestion.
	feeCap := new(big.Int).Add(new(big.Int).Mul(head.BaseFee, big.NewInt(2)), gasTipCap)
	if v := gweiToWeiString(maxFeeGwei); v != nil {
		feeCap = v
	}
	if v := gweiToWeiString(maxPriorityGwei); v != nil {
		gasTipCap = v
	}
	rawTx, err := crypto.SignTransaction(priv, chainID, common.HexToAddress(to), amountInt, gasLimit, feeCap, gasTipCap, nonce, data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	txHash, err := broadcastRawTx(ctx, client, rawTx)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "broadcast failed: " + err.Error()})
		return
	}
	_ = s.store.CreateTransaction(c.Request.Context(), w.ID, txHash, txType, "broadcast", w.Address, to, amountStr, "", w.ChainID)
	if wid != uuid.Nil {
		if g := middleware.GetTwoPartyGate(); g != nil {
			_ = g.MarkWithdrawalExecuted(c.Request.Context(), wid, txHash)
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"transaction_hash":     txHash,
		"status":               "broadcast",
		"from":                 w.Address,
		"auto_approved":        autoApproved,
		"auto_approval_reason": autoReason,
	})
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
