// Package handlers implements the standalone WL-UserWallet backend REST API.
// REAL BIP-39/32/44 key management + REAL EVM signing + REAL PostgreSQL
// persistence + fail-closed license gate. No stubs, no fakes, no TigerWallet
// cloud dependency at request time.
package handlers

import (
	"context"
	"encoding/hex"
	"errors"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tigerwallet/wl-user-wallet/internal/config"
	"github.com/tigerwallet/wl-user-wallet/internal/crypto"
	"github.com/tigerwallet/wl-user-wallet/internal/middleware"
	"github.com/tigerwallet/wl-user-wallet/internal/store"
	"golang.org/x/crypto/bcrypt"
)

type Svc struct {
	cfg  *config.Config
	store *store.Store
}

func New(cfg *config.Config, s *store.Store) *Svc {
	return &Svc{cfg: cfg, store: s}
}

// ==================== Auth (real bcrypt + JWT) ====================

func (s *Svc) Register(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.cfg.BCryptCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hash failed"})
		return
	}
	id, err := s.store.CreateUser(c.Request.Context(), req.Email, string(hash))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "email": req.Email})
}

func (s *Svc) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, hash, err := s.store.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	tok, err := middleware.IssueJWT(s.cfg.JWTSecret, id, req.Email, s.cfg.JWTExpiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token issue failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": tok, "user_id": id, "email": req.Email})
}

// ==================== Wallets (real BIP-39/32/44) ====================

func (s *Svc) CreateWallet(c *gin.Context) {
	userID := middleware.UserID(c)
	var req struct {
		Label     string `json:"label"`
		Password  string `json:"password" binding:"required,min=8"`
		Mnemonic  string `json:"mnemonic"` // optional; if empty, generate
		Passphrase string `json:"passphrase"` // BIP-39 passphrase
		ChainID   int64  `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ChainID == 0 {
		req.ChainID = 1
	}
	mnemonic := req.Mnemonic
	if mnemonic == "" {
		var err error
		mnemonic, err = crypto.GenerateMnemonic()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	seed := crypto.MnemonicToSeed(mnemonic, req.Passphrase)
	priv, err := crypto.DeriveEVMPrivateKey(seed, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "key derivation failed"})
		return
	}
	address := crypto.AddressFromPrivateKey(priv)
	encSeed, err := crypto.EncryptSeedAtRest(seed, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "seed encryption failed"})
		return
	}
	w, err := s.store.CreateWallet(c.Request.Context(), userID, req.Label, address, encSeed, req.ChainID, s.cfg.WLClientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp := gin.H{"id": w.ID, "label": w.Label, "address": address, "chain_id": w.ChainID}
	if req.Mnemonic == "" {
		// Only return the mnemonic once, at creation, when we generated it.
		resp["mnemonic"] = mnemonic
	}
	c.JSON(http.StatusCreated, resp)
}

func (s *Svc) ListWallets(c *gin.Context) {
	userID := middleware.UserID(c)
	ws, err := s.store.ListWallets(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := []gin.H{}
	for _, w := range ws {
		out = append(out, gin.H{"id": w.ID, "label": w.Label, "address": w.Address, "chain_id": w.ChainID, "created_at": w.CreatedAt})
	}
	c.JSON(http.StatusOK, gin.H{"wallets": out})
}

func (s *Svc) GetBalance(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
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
	bal, err := client.BalanceAt(c.Request.Context(), common.HexToAddress(w.Address), nil)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "balance fetch failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"address": w.Address, "balance_wei": bal.String(), "chain_id": w.ChainID})
}

// ==================== Send (real EVM signing + broadcast) ====================

func (s *Svc) SendTransaction(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		To           string `json:"to" binding:"required"`
		Amount       string `json:"amount" binding:"required"` // human-readable
		Password     string `json:"password" binding:"required"`
		GasLimit     uint64 `json:"gas_limit"`
		Token        string `json:"token"`
		WithdrawalID string `json:"withdrawal_id"` // required for fee/revenue/treasury
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
	// Two-mode fund-movement gate: classify the tx. User transfers are
	// auto-approved (fast path); fee/revenue/treasury require two-party.
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
	amountWei, ok := new(big.Float).SetString(req.Amount)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
		return
	}
	amountWei = amountWei.Mul(amountWei, big.NewFloat(1e18))
	amountInt, _ := amountWei.Int(nil)
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

// ==================== Sign message (real EIP-191) ====================

func (s *Svc) SignMessage(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Message  string `json:"message" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
	// personal_sign is a non-value operation => Auto mode (license-alive = approved).
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

// ==================== Transactions ====================

func (s *Svc) ListTransactions(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
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

// ==================== helpers ====================

// requireApproval is the two-mode fund-movement gate. EVERY outgoing value
// transfer or sign in this backend MUST call it before touching the seed.
//
// It runs the AutoApprover classifier:
//   - ModeAuto + approved  => proceed (the <1s fast path; license-alive IS the
//     approval from the MasterWallet owner / SuperAdmin).
//   - ModeManual           => require a two-party-approved withdrawal_id (the
//     slow path for fee/revenue/treasury). The caller passes the withdrawal_id
//     in the request body; if missing or not SuperAdmin-approved => 403.
//   - ModeAuto + denied    => 403/503 (license dead or rule blocked).
//
// This reconciles "user txs auto-approved within a second" with "no
// fee/revenue withdrawal without SuperAdmin collaboration."
func (s *Svc) requireApproval(c *gin.Context, txType, to, token, amount, withdrawalIDStr string) (uuid.UUID, bool) {
	d := middleware.ClassifyTx(txType, to, token, amount)
	if d.Mode == middleware.ModeManual {
		// Fee/revenue/treasury => MUST have a SuperAdmin two-party-approved withdrawal_id.
		wid, err := uuid.Parse(withdrawalIDStr)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{
				"error":  "this transaction requires a SuperAdmin two-party-approved withdrawal_id (fee/revenue/treasury)",
				"reason": d.Reason,
			})
			return uuid.Nil, false
		}
		gate := middleware.GetTwoPartyGate()
		if gate == nil || !gate.IsWithdrawalApproved(c.Request.Context(), wid) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":         "withdrawal not approved by SuperAdmin (two-party gate fail-closed)",
				"withdrawal_id": wid,
			})
			return uuid.Nil, false
		}
		return wid, true
	}
	if !d.Approved {
		code := http.StatusForbidden
		if !middleware.IsAlive() {
			code = http.StatusServiceUnavailable
		}
		c.JSON(code, gin.H{"error": d.Reason, "reason": d.Reason, "rule_id": d.RuleID})
		return uuid.Nil, false
	}
	return uuid.Nil, true
}

func broadcastRawTx(ctx context.Context, client *ethclient.Client, rawTxHex string) (string, error) {
	rawTxHex = strings.TrimPrefix(rawTxHex, "0x")
	rawTx, err := hex.DecodeString(rawTxHex)
	if err != nil {
		return "", err
	}
	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(rawTx); err != nil {
		return "", errors.New("invalid signed tx encoding: " + err.Error())
	}
	if err := client.SendTransaction(ctx, tx); err != nil {
		return "", err
	}
	return tx.Hash().Hex(), nil
}

func (s *Svc) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":      "healthy",
		"service":     "wl-user-wallet",
		"licensed":    middleware.IsAlive(),
		"wl_client_id": s.cfg.WLClientID,
	})
}
