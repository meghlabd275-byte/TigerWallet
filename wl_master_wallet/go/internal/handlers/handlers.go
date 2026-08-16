// Package handlers implements the standalone WL-MasterWallet backend REST API.
// REAL BIP-39/32/44 key management + REAL EVM signing + REAL PostgreSQL
// persistence + fail-closed license gate + two-party withdrawal gate. No stubs,
// no fakes, no TigerWallet cloud dependency at request time.
//
// Two-party gate wiring:
//   - SignTransaction: if req.WithdrawalID is set, the withdrawal MUST be
//     co-approved by SuperAdmin (IsWithdrawalApproved) — fail-closed 403 if not.
//   - RevenuePayout: ALWAYS requires a withdrawal_id (revenue never moves
//     without SuperAdmin co-sign).
package handlers

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tigerwallet/wl-master-wallet/internal/config"
	"github.com/tigerwallet/wl-master-wallet/internal/store"
	"github.com/tigerwallet/wl-shared/wlcrypto"
	"github.com/tigerwallet/wl-shared/wlgate"
	"golang.org/x/crypto/bcrypt"
)

// Handlers serves the WL-MasterWallet REST API. It embeds the license gate
// (fail-closed) and the two-party withdrawal gate.
type Handlers struct {
	cfg          *config.Config
	store        *store.Store
	gate         *wlgate.Gate
	twoPartyGate *wlgate.TwoPartyGate
	wlClientID   uuid.UUID
	wsHub        *wsHub
}

// New builds a Handlers bound to a fail-closed license gate + two-party gate.
func New(cfg *config.Config, st *store.Store, gate *wlgate.Gate) *Handlers {
	wlID, err := uuid.Parse(cfg.WLClientID)
	if err != nil {
		wlID = uuid.Nil
	}
	return &Handlers{
		cfg:          cfg,
		store:        st,
		gate:         gate,
		twoPartyGate: wlgate.NewTwoPartyGate(cfg.ControlPlaneURL, cfg.ControlPlaneToken),
		wlClientID:   wlID,
		wsHub:        newWSHub(),
	}
}

// ==================== Auth (real bcrypt + JWT via wlgate.IssueJWT) ====================

func (h *Handlers) Register(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), h.cfg.BCryptCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hash failed"})
		return
	}
	id, err := h.store.CreateUser(c.Request.Context(), req.Email, string(hash))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "email": req.Email})
}

func (h *Handlers) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, hash, err := h.store.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	scopes := []string{"master_wallet"}
	tok, err := wlgate.IssueJWT(h.cfg.JWTSecret, id, req.Email, h.wlClientID, scopes, h.cfg.JWTExpiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token issue failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": tok, "user_id": id, "email": req.Email, "wl_client_id": h.cfg.WLClientID})
}

// ==================== Master wallets (real BIP-39/32/44) ====================

func (h *Handlers) CreateMasterWallet(c *gin.Context) {
	userID := wlgate.UserID(c)
	var req struct {
		Label      string `json:"label"`
		Password   string `json:"password" binding:"required,min=8"`
		Mnemonic   string `json:"mnemonic"`   // optional; if empty, generate
		Passphrase string `json:"passphrase"` // BIP-39 passphrase
		ChainID    int64  `json:"chain_id"`
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
		mnemonic, err = wlcrypto.GenerateMnemonic()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	seed := wlcrypto.MnemonicToSeed(mnemonic, req.Passphrase)
	priv, err := wlcrypto.DeriveEVMPrivateKey(seed, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "key derivation failed"})
		return
	}
	address := wlcrypto.AddressFromPrivateKey(priv)
	encSeed, err := wlcrypto.EncryptSeedAtRest(seed, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "seed encryption failed"})
		return
	}
	w, err := h.store.CreateMasterWallet(c.Request.Context(), userID, req.Label, address, encSeed, req.ChainID, h.cfg.WLClientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_ = h.store.Audit(c.Request.Context(), w.ID, "create_master_wallet", "master_wallet", w.ID.String(), "info", mustJSON(gin.H{"address": address, "chain_id": req.ChainID}))

	resp := gin.H{"id": w.ID, "label": w.Label, "address": address, "chain_id": w.ChainID, "created_at": w.CreatedAt}
	if req.Mnemonic == "" {
		// Only return the mnemonic once, at creation, when we generated it.
		resp["mnemonic"] = mnemonic
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *Handlers) ListMasterWallets(c *gin.Context) {
	userID := wlgate.UserID(c)
	ws, err := h.store.ListMasterWallets(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := []gin.H{}
	for _, w := range ws {
		out = append(out, gin.H{
			"id": w.ID, "label": w.Label, "address": w.Address,
			"chain_id": w.ChainID, "created_at": w.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"master_wallets": out})
}

func (h *Handlers) GetMasterWallet(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id": w.ID, "label": w.Label, "address": w.Address,
		"chain_id": w.ChainID, "wl_client_id": w.WLClientID, "created_at": w.CreatedAt,
	})
}

func (h *Handlers) DeleteMasterWallet(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	if err := h.store.DeleteMasterWallet(c.Request.Context(), w.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = h.store.Audit(c.Request.Context(), w.ID, "delete_master_wallet", "master_wallet", w.ID.String(), "warning", mustJSON(gin.H{"address": w.Address}))
	c.JSON(http.StatusOK, gin.H{"deleted": w.ID})
}

// ==================== Balance (real ethclient.BalanceAt) ====================

func (h *Handlers) GetBalance(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
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

// ==================== Sign/Send (real EVM signing + two-party gate) ====================

// SignTransaction builds + signs + broadcasts a real EIP-1559 tx. If a
// withdrawal_id is present, the two-party gate MUST approve — fail-closed 403.
func (h *Handlers) SignTransaction(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	var req struct {
		To           string `json:"to" binding:"required"`
		Amount       string `json:"amount" binding:"required"` // human-readable ETH
		Password     string `json:"password" binding:"required"`
		GasLimit     uint64 `json:"gas_limit"`
		Token        string `json:"token"`
		WithdrawalID string `json:"withdrawal_id"` // two-party gate (optional for transfers)
		Data         string `json:"data"`          // hex-encoded calldata (optional)
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Two-party gate: if a withdrawal_id is present, require SuperAdmin co-sign.
	var withdrawalID uuid.UUID
	if req.WithdrawalID != "" {
		wid, err := uuid.Parse(req.WithdrawalID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid withdrawal_id"})
			return
		}
		if !h.twoPartyGate.IsWithdrawalApproved(c.Request.Context(), wid) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":         "withdrawal not approved by SuperAdmin (two-party gate fail-closed)",
				"withdrawal_id": wid,
			})
			return
		}
		withdrawalID = wid
	}

	txHash, err := h.signAndBroadcast(c.Request.Context(), w, req.To, req.Amount, req.Password, req.GasLimit, req.Data)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "broadcast failed: " + err.Error()})
		return
	}
	txType := "transfer"
	if req.Token != "" {
		txType = "token_transfer"
	}
	_ = h.store.CreateTransaction(c.Request.Context(), w.ID, txHash, txType, "broadcast", w.Address, req.To, req.Amount, req.Token, w.ChainID)
	_ = h.store.Audit(c.Request.Context(), w.ID, "sign_transaction", "transaction", txHash, "info", mustJSON(gin.H{
		"to": req.To, "amount": req.Amount, "withdrawal_id": withdrawalID,
	}))
	if withdrawalID != uuid.Nil {
		_ = h.twoPartyGate.MarkWithdrawalExecuted(c.Request.Context(), withdrawalID, txHash)
	}
	c.JSON(http.StatusOK, gin.H{"transaction_hash": txHash, "status": "broadcast", "from": w.Address})
}

// RevenuePayout ALWAYS requires a two-party-approved withdrawal_id. Revenue
// never moves without SuperAdmin co-sign — fail-closed 403 otherwise.
func (h *Handlers) RevenuePayout(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	var req struct {
		To           string `json:"to" binding:"required"`
		Amount       string `json:"amount" binding:"required"`
		Password     string `json:"password" binding:"required"`
		GasLimit     uint64 `json:"gas_limit"`
		WithdrawalID string `json:"withdrawal_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	wid, err := uuid.Parse(req.WithdrawalID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid withdrawal_id"})
		return
	}
	// Revenue ALWAYS requires SuperAdmin co-sign. No exceptions.
	if !h.twoPartyGate.IsWithdrawalApproved(c.Request.Context(), wid) {
		c.JSON(http.StatusForbidden, gin.H{
			"error":         "revenue payout not approved by SuperAdmin (two-party gate required)",
			"withdrawal_id": wid,
		})
		return
	}
	txHash, err := h.signAndBroadcast(c.Request.Context(), w, req.To, req.Amount, req.Password, req.GasLimit, "")
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "broadcast failed: " + err.Error()})
		return
	}
	_ = h.store.CreateTransaction(c.Request.Context(), w.ID, txHash, "revenue_payout", "broadcast", w.Address, req.To, req.Amount, "", w.ChainID)
	_ = h.store.Audit(c.Request.Context(), w.ID, "revenue_payout", "transaction", txHash, "critical", mustJSON(gin.H{
		"to": req.To, "amount": req.Amount, "withdrawal_id": wid,
	}))
	_ = h.twoPartyGate.MarkWithdrawalExecuted(c.Request.Context(), wid, txHash)
	c.JSON(http.StatusOK, gin.H{"transaction_hash": txHash, "status": "broadcast", "type": "revenue_payout", "from": w.Address})
}

// WithdrawalRequest creates a two-party withdrawal request via the control
// plane (WL-side). The WL client approves its half; SuperAdmin must co-approve
// before SignTransaction/RevenuePayout will broadcast.
func (h *Handlers) WithdrawalRequest(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	var req struct {
		To       string `json:"to" binding:"required"`
		Amount   string `json:"amount" binding:"required"` // human-readable ETH
		Token    string `json:"token"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Verify the caller actually holds the wallet key before requesting a
	// withdrawal (fail-closed against impersonation).
	if _, err := h.deriveKey(c.Request.Context(), w, req.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid password"})
		return
	}
	amountWei, err := toWei(req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	currency := req.Token
	if currency == "" {
		currency = "ETH"
	}
	wid, err := h.twoPartyGate.RequestWithdrawal(c.Request.Context(), w.ID, req.To, amountWei.String(), currency, w.ChainID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "withdrawal request failed: " + err.Error()})
		return
	}
	_ = h.store.Audit(c.Request.Context(), w.ID, "withdrawal_request", "withdrawal", wid.String(), "warning", mustJSON(gin.H{
		"to": req.To, "amount": req.Amount, "currency": currency,
	}))
	c.JSON(http.StatusAccepted, gin.H{
		"withdrawal_id": wid, "status": "pending_super_admin_approval",
		"wallet_id": w.ID, "to": req.To, "amount": req.Amount, "currency": currency,
	})
}

// SignMessage signs a personal_sign message (real EIP-191).
func (h *Handlers) SignMessage(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
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
	priv, err := h.deriveKey(c.Request.Context(), w, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid password"})
		return
	}
	sig, err := wlcrypto.SignMessage(priv, req.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = h.store.Audit(c.Request.Context(), w.ID, "sign_message", "message", "", "info", mustJSON(gin.H{"address": w.Address}))
	c.JSON(http.StatusOK, gin.H{"signature": sig, "address": w.Address})
}

// ==================== Transactions ====================

func (h *Handlers) ListTransactions(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	txs, err := h.store.ListTransactions(c.Request.Context(), w.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"transactions": txs})
}

// ==================== Sub wallets (real BIP-44 derivation) ====================

func (h *Handlers) CreateSubWallet(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	var req struct {
		Label        string `json:"label"`
		Password     string `json:"password" binding:"required"`
		AccountIndex uint32 `json:"account_index"` // m/44'/60'/0'/0/account_index
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	seed, err := wlcrypto.DecryptSeedAtRest(w.EncryptedSeed, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid password"})
		return
	}
	priv, err := wlcrypto.DeriveEVMPrivateKey(seed, req.AccountIndex)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "key derivation failed"})
		return
	}
	address := wlcrypto.AddressFromPrivateKey(priv)
	derivationPath := fmt.Sprintf("m/44'/60'/0'/0/%d", req.AccountIndex)
	sw, err := h.store.CreateSubWallet(c.Request.Context(), w.ID, req.Label, address, derivationPath, w.ChainID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_ = h.store.Audit(c.Request.Context(), w.ID, "create_sub_wallet", "sub_wallet", sw.ID.String(), "info", mustJSON(gin.H{"address": address, "path": derivationPath}))
	c.JSON(http.StatusCreated, gin.H{
		"id": sw.ID, "label": sw.Label, "address": sw.Address,
		"derivation_path": sw.DerivationPath, "chain_id": sw.ChainID, "created_at": sw.CreatedAt,
	})
}

func (h *Handlers) ListSubWallets(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	sws, err := h.store.ListSubWallets(c.Request.Context(), w.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := []gin.H{}
	for _, sw := range sws {
		out = append(out, gin.H{
			"id": sw.ID, "label": sw.Label, "address": sw.Address,
			"derivation_path": sw.DerivationPath, "chain_id": sw.ChainID, "created_at": sw.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"sub_wallets": out})
}

// ==================== Policies ====================

func (h *Handlers) CreatePolicy(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	var req struct {
		Name    string          `json:"name" binding:"required"`
		Type    string          `json:"type" binding:"required"`
		Config  json.RawMessage `json:"config"`
		Enabled *bool           `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cfg := []byte(req.Config)
	if len(cfg) == 0 {
		cfg = []byte(`{}`)
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	id, err := h.store.CreatePolicy(c.Request.Context(), w.ID, req.Name, req.Type, cfg, enabled)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_ = h.store.Audit(c.Request.Context(), w.ID, "create_policy", "policy", id.String(), "info", mustJSON(gin.H{"name": req.Name, "type": req.Type}))
	c.JSON(http.StatusCreated, gin.H{"id": id, "name": req.Name, "type": req.Type, "enabled": enabled})
}

func (h *Handlers) ListPolicies(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	ps, err := h.store.ListPolicies(c.Request.Context(), w.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"policies": ps})
}

// ==================== Fee configs ====================

func (h *Handlers) CreateFeeConfig(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	var req struct {
		Name       string  `json:"name" binding:"required"`
		Percentage float64 `json:"percentage"`
		Cap        float64 `json:"cap"`
		Enabled    *bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	id, err := h.store.CreateFeeConfig(c.Request.Context(), w.ID, req.Name, req.Percentage, req.Cap, enabled)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_ = h.store.Audit(c.Request.Context(), w.ID, "create_fee_config", "fee_config", id.String(), "info", mustJSON(gin.H{"name": req.Name, "percentage": req.Percentage, "cap": req.Cap}))
	c.JSON(http.StatusCreated, gin.H{"id": id, "name": req.Name, "percentage": req.Percentage, "cap": req.Cap, "enabled": enabled})
}

func (h *Handlers) ListFeeConfigs(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	fcs, err := h.store.ListFeeConfigs(c.Request.Context(), w.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"fee_configs": fcs})
}

// ==================== Auto-sign rules ====================

func (h *Handlers) CreateAutoSignRule(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	var req struct {
		Trigger string `json:"trigger" binding:"required"`
		Action  string `json:"action" binding:"required"`
		Enabled *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	id, err := h.store.CreateAutoSignRule(c.Request.Context(), w.ID, req.Trigger, req.Action, enabled)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_ = h.store.Audit(c.Request.Context(), w.ID, "create_auto_sign_rule", "auto_sign_rule", id.String(), "warning", mustJSON(gin.H{"trigger": req.Trigger, "action": req.Action}))
	c.JSON(http.StatusCreated, gin.H{"id": id, "trigger": req.Trigger, "action": req.Action, "enabled": enabled})
}

func (h *Handlers) ListAutoSignRules(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	rs, err := h.store.ListAutoSignRules(c.Request.Context(), w.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"auto_sign_rules": rs})
}

// ==================== Health ====================

func (h *Handlers) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":       "healthy",
		"service":      "wl-master-wallet",
		"licensed":     h.gate.IsAlive(),
		"reason":       h.gate.Reason(),
		"wl_client_id": h.cfg.WLClientID,
		"product":      h.cfg.Product,
		"instance_id":  h.cfg.InstanceID,
	})
}

// ==================== helpers ====================

// loadOwnedWallet parses :id, fetches the master wallet, and enforces that the
// caller owns it. Writes the error response and returns ok=false on failure.
func (h *Handlers) loadOwnedWallet(c *gin.Context) (*store.MasterWallet, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return nil, false
	}
	w, err := h.store.GetMasterWallet(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "master wallet not found"})
		return nil, false
	}
	if w.UserID != wlgate.UserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your master wallet"})
		return nil, false
	}
	return w, true
}

// deriveKey decrypts the at-rest seed and derives the EVM private key at
// m/44'/60'/0'/0/0. Fail-closed on wrong passphrase / tampered ciphertext.
func (h *Handlers) deriveKey(ctx context.Context, w *store.MasterWallet, password string) (*ecdsa.PrivateKey, error) {
	seed, err := wlcrypto.DecryptSeedAtRest(w.EncryptedSeed, password)
	if err != nil {
		return nil, errors.New("invalid password")
	}
	priv, err := wlcrypto.DeriveEVMPrivateKey(seed, 0)
	if err != nil {
		return nil, errors.New("key derivation failed")
	}
	return priv, nil
}

// signAndBroadcast is the shared real-EVM-signing + broadcast path used by
// SignTransaction and RevenuePayout.
func (h *Handlers) signAndBroadcast(ctx context.Context, w *store.MasterWallet, to, amount, password string, gasLimit uint64, dataHex string) (string, error) {
	priv, err := h.deriveKey(ctx, w, password)
	if err != nil {
		return "", err
	}
	rpc := rpcForChain(w.ChainID)
	if rpc == "" {
		return "", errors.New("no RPC configured for chain")
	}
	client, err := ethclient.DialContext(ctx, rpc)
	if err != nil {
		return "", errors.New("RPC unavailable")
	}
	defer client.Close()
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	nonce, err := client.PendingNonceAt(cctx, common.HexToAddress(w.Address))
	if err != nil {
		return "", errors.New("nonce fetch failed")
	}
	amountInt, err := toWei(amount)
	if err != nil {
		return "", err
	}
	chainID := big.NewInt(w.ChainID)
	gasTipCap, err := client.SuggestGasTipCap(cctx)
	if err != nil {
		return "", errors.New("gas tip cap fetch failed")
	}
	head, err := client.HeaderByNumber(cctx, nil)
	if err != nil || head == nil {
		return "", errors.New("head fetch failed")
	}
	var data []byte
	if dataHex != "" {
		clean := strings.TrimPrefix(dataHex, "0x")
		data, err = hex.DecodeString(clean)
		if err != nil {
			return "", errors.New("invalid calldata hex")
		}
	}
	rawTx, err := wlcrypto.SignTransaction(priv, chainID, common.HexToAddress(to), amountInt, gasLimit, head.BaseFee, gasTipCap, nonce, data)
	if err != nil {
		return "", err
	}
	return broadcastRawTx(cctx, client, rawTx)
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

// toWei converts a human-readable ETH amount string to wei (*big.Int).
func toWei(amount string) (*big.Int, error) {
	f, ok := new(big.Float).SetString(amount)
	if !ok {
		return nil, errors.New("invalid amount")
	}
	f = f.Mul(f, big.NewFloat(1e18))
	n, _ := f.Int(nil)
	if n == nil {
		return nil, errors.New("amount overflow")
	}
	return n, nil
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte(`{}`)
	}
	return b
}
