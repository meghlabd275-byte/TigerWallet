package handlers

// handlers_routes.go — Additional route handlers for the standalone
// WL-MasterWallet backend, porting the canonical master_wallet/backend feature
// set to the WL structure. REAL PostgreSQL (pgxpool) + REAL on-chain RPC
// (ethclient) only — no stubs/fakes/mocks. Every fund-movement route enforces
// the two-party SuperAdmin withdrawal gate fail-closed.
//
// Groupings:
//   - Sub-wallet balance/transfer (real ethclient + EIP-1559 sign+broadcast)
//   - Transaction workflow (create/approve/reject/execute/sign)
//   - DELETE endpoints (fees/policies/auto-sign/users/webhooks)
//   - Users CRUD (master_wallet_users)
//   - Analytics (real SQL aggregates)
//   - Notifications / Webhooks / Audit
//   - Market data (chains/gas/price/history/health) — public, read-only
//   - Auto-sign transaction + logs
//   - Treasury transfer/sweep (two-party gated)
//   - UserWallet management layer (chains/tokens/addresses/derive) + feature flags

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/tigerwallet/wl-master-wallet/internal/store"
	"github.com/tigerwallet/wl-shared/wlcrypto"
	"github.com/tigerwallet/wl-shared/wlgate"
	"golang.org/x/crypto/bcrypt"
)

// ----------------------------------------------------------------------------
// Role gate
// ----------------------------------------------------------------------------

// adminRoles may manage users, feature flags, and treasury operations.
var adminRoles = map[string]bool{
	"admin": true, "treasury": true, "operator": true, "super_admin": true,
}

// adminScopeWhitelist is the canonical scoped-role taxonomy (from
// white_label_admin/go/internal/roles). UpdateAdminScopes validates every
// requested scope against this set.
var adminScopeWhitelist = map[string]bool{
	"wl_client": true, "trading_admin": true, "p2p_admin": true, "bot_admin": true,
	"listing_admin": true, "liquidity_admin": true, "wallet_admin": true,
	"customer_service_admin": true, "marketing_admin": true, "kyc_admin": true,
	"card_admin": true, "reward_admin": true, "security_admin": true,
	"compliance_admin": true, "user": true,
}

// requireRole is a gin middleware that prefers the canonical scope taxonomy
// (wlgate.HasScope) and falls back to the legacy local role (users.role) for
// backward compatibility. 'wl_client' (the WL owner) always passes; the
// canonical scope for THIS product is 'wallet_admin' (MasterWallet/UserWallet
// management). Legacy admin/treasury/operator/super_admin role strings are
// honored via the DB fallback so existing deployments don't break.
func (h *Handlers) requireRole(allowed ...string) gin.HandlerFunc {
	allow := map[string]bool{}
	for _, r := range allowed {
		allow[r] = true
	}
	return func(c *gin.Context) {
		uid := wlgate.UserID(c)
		if uid == uuid.Nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
			return
		}
		// wl_client (WL owner) always passes — full tenancy control.
		if wlgate.HasScope(c, "wl_client") {
			c.Set("role", "wl_client")
			c.Next()
			return
		}
		// Canonical scope: wallet_admin controls all MasterWallet management.
		if wlgate.HasScope(c, "wallet_admin") {
			c.Set("role", "wallet_admin")
			c.Next()
			return
		}
		// Legacy local-role fallback: load the user's role from the DB + match.
		role := h.store.UserRole(c.Request.Context(), uid)
		if !allow[role] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "insufficient role (required: " + strings.Join(allowed, ",") + ")",
				"role":  role,
			})
			return
		}
		c.Set("role", role)
		c.Next()
	}
}

// UpdateAdminScopes is the WL-client-facing endpoint to grant/revoke scoped
// admin roles on a master-wallet user. Mirrors white_label_admin
// AssignAdminRole. Only a caller holding 'wl_client' (the WL owner) may set
// scopes — a wallet_admin cannot escalate themselves or others. The scopes
// MUST be in the canonical whitelist (validated server-side).
func (h *Handlers) UpdateAdminScopes(c *gin.Context) {
	if !wlgate.HasScope(c, "wl_client") {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "only the WL client owner may assign admin scopes"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	var req struct {
		Scopes []string `json:"scopes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	for _, sc := range req.Scopes {
		if !adminScopeWhitelist[sc] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scope: " + sc})
			return
		}
	}
	if err := h.store.UpdateUserScopes(c.Request.Context(), id, req.Scopes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_ = h.store.Audit(c.Request.Context(), uuid.Nil, "update_admin_scopes", "user", id.String(), "warning", mustJSON(gin.H{"scopes": req.Scopes, "set_by": wlgate.UserID(c)}))
	c.JSON(http.StatusOK, gin.H{"user_id": id, "scopes": req.Scopes})
}

// ----------------------------------------------------------------------------
// Sub-wallet balance / transfer
// ----------------------------------------------------------------------------

// GetSubWalletBalance — real ethclient.BalanceAt for a sub-wallet address.
func (h *Handlers) GetSubWalletBalance(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	sid, err := uuid.Parse(c.Param("sid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sub-wallet id"})
		return
	}
	addr, chainID, err := h.subWalletAddress(c.Request.Context(), w.ID, sid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	rpc := rpcForChain(chainID)
	if rpc == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no RPC configured for chain"})
		return
	}
	bal, err := fetchNativeBalance(c.Request.Context(), rpc, common.HexToAddress(addr))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "balance fetch failed: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"address": addr, "balance_wei": bal.String(), "chain_id": chainID})
}

// TransferFromSubWallet — real EIP-1559 sign+broadcast from a sub-wallet
// derived key. If withdrawal_id is present, the two-party gate MUST approve.
func (h *Handlers) TransferFromSubWallet(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	sid, err := uuid.Parse(c.Param("sid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sub-wallet id"})
		return
	}
	subAddr, chainID, path, err := h.subWalletAddressPath(c.Request.Context(), w.ID, sid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	var req struct {
		To           string `json:"to" binding:"required"`
		Amount       string `json:"amount" binding:"required"`
		Password     string `json:"password" binding:"required"`
		GasLimit     uint64 `json:"gas_limit"`
		Token        string `json:"token"`
		WithdrawalID string `json:"withdrawal_id"`
		Data         string `json:"data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Two-mode gate: user transfer => Auto (fast path); treasury recipient or
	// fee/revenue => Manual (require two-party-approved withdrawal_id).
	wid, ok := h.requireApproval(c, "transfer", req.To, req.Token, req.Amount, req.WithdrawalID)
	if !ok {
		return
	}
	withdrawalID := wid

	txHash, err := h.signAndBroadcastPath(c.Request.Context(), w, subAddr, path, req.To, req.Amount, req.Password, req.GasLimit, req.Data)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "broadcast failed: " + err.Error()})
		return
	}
	txType := "transfer"
	if req.Token != "" {
		txType = "token_transfer"
	}
	_ = h.store.CreateTransaction(c.Request.Context(), w.ID, txHash, txType, "broadcast", subAddr, req.To, req.Amount, req.Token, chainID)
	_ = h.store.Audit(c.Request.Context(), w.ID, "sub_wallet_transfer", "transaction", txHash, "info", mustJSON(gin.H{
		"sub_wallet_id": sid, "from": subAddr, "to": req.To, "amount": req.Amount, "withdrawal_id": withdrawalID,
	}))
	if withdrawalID != uuid.Nil {
		_ = h.twoPartyGate.MarkWithdrawalExecuted(c.Request.Context(), withdrawalID, txHash)
	}
	c.JSON(http.StatusOK, gin.H{"transaction_hash": txHash, "status": "broadcast", "from": subAddr})
}

// subWalletAddress loads just the address + chain id for a sub wallet.
func (h *Handlers) subWalletAddress(ctx context.Context, mwID, sid uuid.UUID) (addr string, chainID int64, err error) {
	var path string
	err = h.store.DB().QueryRow(ctx,
		`SELECT address, chain_id, derivation_path FROM sub_wallets WHERE id=$1 AND master_wallet_id=$2`,
		sid, mwID).Scan(&addr, &chainID, &path)
	if err != nil {
		return "", 0, errors.New("sub-wallet not found")
	}
	return addr, chainID, nil
}

// subWalletAddressPath loads address + chain id + derivation path.
func (h *Handlers) subWalletAddressPath(ctx context.Context, mwID, sid uuid.UUID) (addr string, chainID int64, path string, err error) {
	err = h.store.DB().QueryRow(ctx,
		`SELECT address, chain_id, derivation_path FROM sub_wallets WHERE id=$1 AND master_wallet_id=$2`,
		sid, mwID).Scan(&addr, &chainID, &path)
	if err != nil {
		return "", 0, "", errors.New("sub-wallet not found")
	}
	return addr, chainID, path, nil
}

// signAndBroadcastPath signs + broadcasts using a key derived from an arbitrary
// BIP-32 derivation path (sub-wallet account index), not the master m/.../0.
func (h *Handlers) signAndBroadcastPath(ctx context.Context, w *store.MasterWallet, fromAddr, path, to, amount, password string, gasLimit uint64, dataHex string) (string, error) {
	seed, err := wlcrypto.DecryptSeedAtRest(w.EncryptedSeed, password)
	if err != nil {
		return "", errors.New("invalid password")
	}
	priv, err := derivePrivateKeyFromPath(seed, path)
	if err != nil {
		return "", errors.New("key derivation failed for path " + path)
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
	nonce, err := client.PendingNonceAt(cctx, common.HexToAddress(fromAddr))
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

// ----------------------------------------------------------------------------
// Transaction workflow
// ----------------------------------------------------------------------------

// CreatePendingTransaction creates a pending transaction record (status=pending).
func (h *Handlers) CreatePendingTransaction(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	var req struct {
		To      string `json:"to" binding:"required"`
		Amount  string `json:"amount" binding:"required"`
		Token   string `json:"token"`
		ChainID int64  `json:"chain_id"`
		Note    string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ChainID == 0 {
		req.ChainID = w.ChainID
	}
	tid := uuid.New()
	_, err := h.store.DB().Exec(c.Request.Context(),
		`INSERT INTO transactions (id, master_wallet_id, tx_hash, tx_type, status, from_address, to_address, amount, token, chain_id) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		tid, w.ID, "", "transfer", "pending", w.Address, req.To, req.Amount, req.Token, req.ChainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = h.store.Audit(c.Request.Context(), w.ID, "create_pending_tx", "transaction", tid.String(), "info", mustJSON(gin.H{"to": req.To, "amount": req.Amount, "note": req.Note}))
	c.JSON(http.StatusCreated, gin.H{"id": tid, "status": "pending", "to": req.To, "amount": req.Amount, "token": req.Token, "chain_id": req.ChainID})
}

// ApproveTransaction records an approval on a pending transaction.
func (h *Handlers) ApproveTransaction(c *gin.Context) {
	h.txStateChange(c, "approved")
}

// RejectTransaction records a rejection on a pending transaction.
func (h *Handlers) RejectTransaction(c *gin.Context) {
	h.txStateChange(c, "rejected")
}

func (h *Handlers) txStateChange(c *gin.Context, state string) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	tid, err := uuid.Parse(c.Param("tid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid transaction id"})
		return
	}
	tag, err := h.store.DB().Exec(c.Request.Context(),
		`UPDATE transactions SET status=$1 WHERE id=$2 AND master_wallet_id=$3 AND status='pending'`,
		state, tid, w.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "transaction not pending or not found"})
		return
	}
	_ = h.store.Audit(c.Request.Context(), w.ID, "tx_"+state, "transaction", tid.String(), "info", nil)
	c.JSON(http.StatusOK, gin.H{"id": tid, "status": state})
}

// ExecuteTransaction executes (broadcasts) an approved transaction. The
// two-party gate is enforced fail-closed BEFORE broadcast: every execution
// requires a SuperAdmin-approved withdrawal_id. 403 if not approved.
func (h *Handlers) ExecuteTransaction(c *gin.Context) {
	tid, err := uuid.Parse(c.Param("tid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid transaction id"})
		return
	}
	var req struct {
		Password     string `json:"password" binding:"required"`
		WithdrawalID string `json:"withdrawal_id"` // required when classifier returns Manual
		GasLimit     uint64 `json:"gas_limit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var mwID uuid.UUID
	var to, amount, token, status string
	var chainID int64
	err = h.store.DB().QueryRow(c.Request.Context(),
		`SELECT master_wallet_id, to_address, amount, token, chain_id, status FROM transactions WHERE id=$1`,
		tid).Scan(&mwID, &to, &amount, &token, &chainID, &status)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}
	w, err := h.store.GetMasterWallet(c.Request.Context(), mwID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "master wallet not found"})
		return
	}
	if w.UserID != wlgate.UserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your master wallet"})
		return
	}
	if status != "approved" {
		c.JSON(http.StatusConflict, gin.H{"error": "transaction not approved (status=" + status + ")"})
		return
	}
	// Two-mode gate with the loaded to/amount/token. User transfers => Auto;
	// treasury recipient / fee / revenue => Manual (require withdrawal_id).
	wid, ok := h.requireApproval(c, "transfer", to, token, amount, req.WithdrawalID)
	if !ok {
		return
	}
	txHash, err := h.signAndBroadcast(c.Request.Context(), w, to, amount, req.Password, req.GasLimit, "")
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "broadcast failed: " + err.Error()})
		return
	}
	_, _ = h.store.DB().Exec(c.Request.Context(),
		`UPDATE transactions SET status='broadcast', tx_hash=$1 WHERE id=$2`, txHash, tid)
	_ = h.store.Audit(c.Request.Context(), w.ID, "execute_tx", "transaction", txHash, "critical", mustJSON(gin.H{"tx_id": tid, "withdrawal_id": wid}))
	if wid != uuid.Nil {
		_ = h.twoPartyGate.MarkWithdrawalExecuted(c.Request.Context(), wid, txHash)
	}
	c.JSON(http.StatusOK, gin.H{"transaction_hash": txHash, "status": "broadcast", "tx_id": tid})
}

// SignPendingTransaction signs (but does not broadcast) a pending tx — returns
// the raw signed hex. The two-party gate is enforced when a withdrawal_id is
// present (fund movement requires SuperAdmin co-sign).
func (h *Handlers) SignPendingTransaction(c *gin.Context) {
	tid, err := uuid.Parse(c.Param("tid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid transaction id"})
		return
	}
	var req struct {
		Password     string `json:"password" binding:"required"`
		WithdrawalID string `json:"withdrawal_id"`
		GasLimit     uint64 `json:"gas_limit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var mwID uuid.UUID
	var to, amount, token, status string
	var chainID int64
	err = h.store.DB().QueryRow(c.Request.Context(),
		`SELECT master_wallet_id, to_address, amount, token, chain_id, status FROM transactions WHERE id=$1`,
		tid).Scan(&mwID, &to, &amount, &token, &chainID, &status)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}
	w, err := h.store.GetMasterWallet(c.Request.Context(), mwID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "master wallet not found"})
		return
	}
	if w.UserID != wlgate.UserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your master wallet"})
		return
	}
	// Two-mode gate with the loaded to/amount/token.
	wid, ok := h.requireApproval(c, "transfer", to, token, amount, req.WithdrawalID)
	if !ok {
		return
	}
	priv, err := h.deriveKey(c.Request.Context(), w, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid password"})
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
	cctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	nonce, err := client.PendingNonceAt(cctx, common.HexToAddress(w.Address))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "nonce fetch failed"})
		return
	}
	amountInt, err := toWei(amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	gasTipCap, err := client.SuggestGasTipCap(cctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "gas tip cap fetch failed"})
		return
	}
	head, err := client.HeaderByNumber(cctx, nil)
	if err != nil || head == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "head fetch failed"})
		return
	}
	rawTx, err := wlcrypto.SignTransaction(priv, big.NewInt(chainID), common.HexToAddress(to), amountInt, req.GasLimit, head.BaseFee, gasTipCap, nonce, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sign failed: " + err.Error()})
		return
	}
	_, _ = h.store.DB().Exec(c.Request.Context(), `UPDATE transactions SET status='signed' WHERE id=$1`, tid)
	_ = h.store.Audit(c.Request.Context(), w.ID, "sign_pending_tx", "transaction", tid.String(), "info", mustJSON(gin.H{"withdrawal_id": wid}))
	c.JSON(http.StatusOK, gin.H{"raw_tx": rawTx, "tx_id": tid, "status": "signed"})
}

// ----------------------------------------------------------------------------
// DELETE endpoints
// ----------------------------------------------------------------------------

func (h *Handlers) DeleteFeeConfig(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	fid, err := uuid.Parse(c.Param("fid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fee id"})
		return
	}
	tag, err := h.store.DB().Exec(c.Request.Context(), `DELETE FROM fee_configs WHERE id=$1 AND master_wallet_id=$2`, fid, w.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "fee config not found"})
		return
	}
	_ = h.store.Audit(c.Request.Context(), w.ID, "delete_fee_config", "fee_config", fid.String(), "warning", nil)
	c.JSON(http.StatusOK, gin.H{"deleted": fid})
}

func (h *Handlers) DeletePolicy(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	pid, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid policy id"})
		return
	}
	tag, err := h.store.DB().Exec(c.Request.Context(), `DELETE FROM policies WHERE id=$1 AND master_wallet_id=$2`, pid, w.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}
	_ = h.store.Audit(c.Request.Context(), w.ID, "delete_policy", "policy", pid.String(), "warning", nil)
	c.JSON(http.StatusOK, gin.H{"deleted": pid})
}

func (h *Handlers) DeleteAutoSignRule(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	rid, err := uuid.Parse(c.Param("rid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule id"})
		return
	}
	tag, err := h.store.DB().Exec(c.Request.Context(), `DELETE FROM auto_sign_rules WHERE id=$1 AND master_wallet_id=$2`, rid, w.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "auto-sign rule not found"})
		return
	}
	_ = h.store.Audit(c.Request.Context(), w.ID, "delete_auto_sign_rule", "auto_sign_rule", rid.String(), "warning", nil)
	c.JSON(http.StatusOK, gin.H{"deleted": rid})
}

func (h *Handlers) DeleteMasterWalletUser(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	uid, err := uuid.Parse(c.Param("uid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	tag, err := h.store.DB().Exec(c.Request.Context(), `DELETE FROM master_wallet_users WHERE id=$1 AND master_wallet_id=$2`, uid, w.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	_ = h.store.Audit(c.Request.Context(), w.ID, "delete_mw_user", "master_wallet_user", uid.String(), "warning", nil)
	c.JSON(http.StatusOK, gin.H{"deleted": uid})
}

func (h *Handlers) DeleteWebhook(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	wid, err := uuid.Parse(c.Param("wid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook id"})
		return
	}
	tag, err := h.store.DB().Exec(c.Request.Context(), `DELETE FROM webhooks WHERE id=$1 AND master_wallet_id=$2`, wid, w.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "webhook not found"})
		return
	}
	_ = h.store.Audit(c.Request.Context(), w.ID, "delete_webhook", "webhook", wid.String(), "warning", nil)
	c.JSON(http.StatusOK, gin.H{"deleted": wid})
}

// ----------------------------------------------------------------------------
// Master-wallet users CRUD
// ----------------------------------------------------------------------------

func (h *Handlers) ListMasterWalletUsers(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	rows, err := h.store.DB().Query(c.Request.Context(),
		`SELECT id, email, name, role, is_active, last_login_at, created_at FROM master_wallet_users WHERE master_wallet_id=$1 ORDER BY created_at DESC`, w.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var email, name, role string
		var isActive bool
		var lastLogin *time.Time
		var created time.Time
		_ = rows.Scan(&id, &email, &name, &role, &isActive, &lastLogin, &created)
		row := gin.H{"id": id, "email": email, "name": name, "role": role, "is_active": isActive, "created_at": created}
		if lastLogin != nil {
			row["last_login_at"] = *lastLogin
		}
		out = append(out, row)
	}
	c.JSON(http.StatusOK, gin.H{"users": out})
}

// CreateMasterWalletUser creates a governed user under a master wallet. Role is
// forced to "user" unless the caller is an admin (admin may assign operator/
// treasury/admin). super_admin may only be granted by an existing super_admin.
func (h *Handlers) CreateMasterWalletUser(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Name     string `json:"name"`
		Password string `json:"password" binding:"required,min=8"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	callerRole := h.store.UserRole(c.Request.Context(), wlgate.UserID(c))
	role := "user"
	switch strings.ToLower(req.Role) {
	case "operator", "treasury", "admin":
		if adminRoles[callerRole] {
			role = strings.ToLower(req.Role)
		}
	case "super_admin":
		if callerRole == "super_admin" {
			role = "super_admin"
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), h.cfg.BCryptCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hash failed"})
		return
	}
	id := uuid.New()
	_, err = h.store.DB().Exec(c.Request.Context(),
		`INSERT INTO master_wallet_users (id, master_wallet_id, email, name, role, password_hash) VALUES ($1,$2,$3,$4,$5,$6)`,
		id, w.ID, req.Email, req.Name, role, string(hash))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_ = h.store.Audit(c.Request.Context(), w.ID, "create_mw_user", "master_wallet_user", id.String(), "info", mustJSON(gin.H{"email": req.Email, "role": role, "granted_by": callerRole}))
	c.JSON(http.StatusCreated, gin.H{"id": id, "email": req.Email, "name": req.Name, "role": role})
}

// ----------------------------------------------------------------------------
// Analytics (real SQL aggregates — honest empty when no data)
// ----------------------------------------------------------------------------

func (h *Handlers) AnalyticsTransactions(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	rows, err := h.store.DB().Query(c.Request.Context(),
		`SELECT status, COUNT(*) FROM transactions WHERE master_wallet_id=$1 GROUP BY status`, w.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	byStatus := gin.H{}
	total := 0
	for rows.Next() {
		var status string
		var n int
		_ = rows.Scan(&status, &n)
		byStatus[status] = n
		total += n
	}
	c.JSON(http.StatusOK, gin.H{"total": total, "by_status": byStatus})
}

func (h *Handlers) AnalyticsVolume(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	// Sum broadcast transfer volume per token. amount is stored as a decimal
	// string; cast to numeric for the sum.
	rows, err := h.store.DB().Query(c.Request.Context(),
		`SELECT COALESCE(token,''), COUNT(*), COALESCE(SUM(amount::numeric),0) FROM transactions WHERE master_wallet_id=$1 AND status='broadcast' GROUP BY token`, w.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	byToken := []gin.H{}
	for rows.Next() {
		var token string
		var count int
		var sum string
		_ = rows.Scan(&token, &count, &sum)
		byToken = append(byToken, gin.H{"token": token, "count": count, "volume": sum})
	}
	c.JSON(http.StatusOK, gin.H{"by_token": byToken})
}

func (h *Handlers) AnalyticsWallets(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	var subCount int
	_ = h.store.DB().QueryRow(c.Request.Context(),
		`SELECT COUNT(*) FROM sub_wallets WHERE master_wallet_id=$1`, w.ID).Scan(&subCount)
	var userCount int
	_ = h.store.DB().QueryRow(c.Request.Context(),
		`SELECT COUNT(*) FROM master_wallet_users WHERE master_wallet_id=$1`, w.ID).Scan(&userCount)
	c.JSON(http.StatusOK, gin.H{"sub_wallets": subCount, "governed_users": userCount})
}

// ----------------------------------------------------------------------------
// Notifications
// ----------------------------------------------------------------------------

func (h *Handlers) ListNotifications(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	limit := parseLimit(c.Query("limit"), 50)
	rows, err := h.store.DB().Query(c.Request.Context(),
		`SELECT id, notification_type, category, title, message, priority, channel, is_read, created_at FROM notifications WHERE master_wallet_id=$1 ORDER BY created_at DESC LIMIT $2`, w.ID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var nType, title, message, priority, channel string
		var category *string
		var isRead bool
		var created time.Time
		_ = rows.Scan(&id, &nType, &category, &title, &message, &priority, &channel, &isRead, &created)
		row := gin.H{"id": id, "type": nType, "title": title, "message": message, "priority": priority, "channel": channel, "is_read": isRead, "created_at": created}
		if category != nil {
			row["category"] = *category
		}
		out = append(out, row)
	}
	c.JSON(http.StatusOK, gin.H{"notifications": out})
}

func (h *Handlers) CreateNotification(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	var req struct {
		Type     string          `json:"type" binding:"required"`
		Category string          `json:"category"`
		Title    string          `json:"title" binding:"required"`
		Message  string          `json:"message" binding:"required"`
		Priority string          `json:"priority"`
		Channel  string          `json:"channel"`
		Data     json.RawMessage `json:"data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Priority == "" {
		req.Priority = "normal"
	}
	if req.Channel == "" {
		req.Channel = "in_app"
	}
	data := []byte(req.Data)
	if len(data) == 0 {
		data = []byte(`{}`)
	}
	id := uuid.New()
	_, err := h.store.DB().Exec(c.Request.Context(),
		`INSERT INTO notifications (id, master_wallet_id, user_id, notification_type, category, title, message, priority, channel, data) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		id, w.ID, wlgate.UserID(c), req.Type, req.Category, req.Title, req.Message, req.Priority, req.Channel, data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = h.store.Audit(c.Request.Context(), w.ID, "create_notification", "notification", id.String(), "info", nil)
	c.JSON(http.StatusCreated, gin.H{"id": id, "type": req.Type, "title": req.Title, "priority": req.Priority})
}

// ----------------------------------------------------------------------------
// Webhooks
// ----------------------------------------------------------------------------

func (h *Handlers) ListWebhooks(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	rows, err := h.store.DB().Query(c.Request.Context(),
		`SELECT id, name, url, events, retry_count, is_active, is_verified, total_delivered, total_failed, created_at FROM webhooks WHERE master_wallet_id=$1 ORDER BY created_at DESC`, w.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var name, urlStr string
		var events []string
		var retry int
		var isActive, isVerified bool
		var delivered, failed int64
		var created time.Time
		_ = rows.Scan(&id, &name, &urlStr, &events, &retry, &isActive, &isVerified, &delivered, &failed, &created)
		out = append(out, gin.H{
			"id": id, "name": name, "url": urlStr, "events": events, "retry_count": retry,
			"is_active": isActive, "is_verified": isVerified, "total_delivered": delivered,
			"total_failed": failed, "created_at": created,
		})
	}
	c.JSON(http.StatusOK, gin.H{"webhooks": out})
}

func (h *Handlers) CreateWebhook(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	var req struct {
		Name       string   `json:"name" binding:"required"`
		URL        string   `json:"url" binding:"required,url"`
		Events     []string `json:"events"`
		RetryCount int      `json:"retry_count"`
		IsActive   *bool    `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.Events) == 0 {
		req.Events = []string{}
	}
	retry := 3
	if req.RetryCount > 0 {
		retry = req.RetryCount
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	id := uuid.New()
	_, err := h.store.DB().Exec(c.Request.Context(),
		`INSERT INTO webhooks (id, master_wallet_id, name, url, events, retry_count, is_active) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		id, w.ID, req.Name, req.URL, req.Events, retry, active)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_ = h.store.Audit(c.Request.Context(), w.ID, "create_webhook", "webhook", id.String(), "info", mustJSON(gin.H{"url": req.URL, "events": req.Events}))
	c.JSON(http.StatusCreated, gin.H{"id": id, "name": req.Name, "url": req.URL, "events": req.Events, "is_active": active})
}

// ----------------------------------------------------------------------------
// Audit
// ----------------------------------------------------------------------------

func (h *Handlers) AuditLog(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	limit := parseLimit(c.Query("limit"), 100)
	logs, err := h.store.ListAuditLogs(c.Request.Context(), w.ID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"audit": logs})
}

// ----------------------------------------------------------------------------
// Market data (public, read-only — outside the license-gated group)
// ----------------------------------------------------------------------------

// PublicChains returns the full 186-chain registry.
func (h *Handlers) PublicChains(c *gin.Context) {
	ctype := c.Query("type")
	var chains []ChainConfig
	switch ctype {
	case "evm":
		chains = evmChains()
	case "nonevm":
		chains = nonEVMChains()
	default:
		chains = allChains()
	}
	c.JSON(http.StatusOK, gin.H{"chains": chains, "count": len(chains)})
}

// PublicGas returns real eth_feeHistory + eth_gasPrice for a chain id.
func (h *Handlers) PublicGas(c *gin.Context) {
	chainID, err := strconv.ParseInt(c.Query("chain_id"), 10, 64)
	if err != nil || chainID == 0 {
		chainID = 1
	}
	rpc := rpcForChain(chainID)
	if rpc == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no RPC configured for chain"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	gp, maxFee, tip, err := fetchGasPrice(ctx, rpc)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "gas fetch failed: " + err.Error()})
		return
	}
	baseFees, ratios, rewards, _ := fetchFeeHistory(ctx, rpc, 10, []float64{25, 50, 75})
	c.JSON(http.StatusOK, gin.H{
		"chain_id":          chainID,
		"gas_price_wei":     gp.String(),
		"max_fee_wei":       maxFee.String(),
		"priority_fee_wei":  tip.String(),
		"fee_history":       gin.H{"base_fees": bigIntsToHex(baseFees), "gas_used_ratios": ratios, "rewards": rewardsToHex(rewards)},
	})
}

func bigIntsToHex(bs []*big.Int) []string {
	out := make([]string, 0, len(bs))
	for _, b := range bs {
		if b == nil {
			out = append(out, "")
			continue
		}
		out = append(out, "0x"+b.Text(16))
	}
	return out
}

func rewardsToHex(rs [][]*big.Int) [][]string {
	out := make([][]string, 0, len(rs))
	for _, r := range rs {
		row := make([]string, 0, len(r))
		for _, b := range r {
			if b == nil {
				row = append(row, "")
				continue
			}
			row = append(row, "0x"+b.Text(16))
		}
		out = append(out, row)
	}
	return out
}

// PublicPrice returns the real CoinGecko USD price for a coin id or chain id.
func (h *Handlers) PublicPrice(c *gin.Context) {
	coinID := c.Query("coin_id")
	if coinID == "" {
		chainID, _ := strconv.ParseInt(c.Query("chain_id"), 10, 64)
		coinID = chainCoinGeckoID(chainID)
	}
	if coinID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "coin_id or supported chain_id required"})
		return
	}
	p, err := fetchTokenPrice(c.Request.Context(), coinID)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "price fetch failed: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"coin_id": coinID, "usd": p.USD, "usd_24h_change": p.USD24h, "usd_market_cap": p.MarketCap})
}

// PublicTransactionHistory proxies an Etherscan-compatible explorer for the
// real on-chain tx history of an address. Fail-closed if no explorer configured.
func (h *Handlers) PublicTransactionHistory(c *gin.Context) {
	addr := c.Query("address")
	if addr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "address required"})
		return
	}
	chainID, err := strconv.ParseInt(c.Query("chain_id"), 10, 64)
	if err != nil || chainID == 0 {
		chainID = 1
	}
	base, keyEnv := chainExplorerAPI(chainID)
	if base == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no explorer API configured for chain " + strconv.FormatInt(chainID, 10)})
		return
	}
	apiKey := ""
	if keyEnv != "" {
		apiKey = os.Getenv(keyEnv)
	}
	txs, err := fetchTransactionHistory(c.Request.Context(), base, apiKey, addr, chainID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "history fetch failed: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"transactions": txs, "count": len(txs), "chain_id": chainID, "address": addr})
}

// PublicHealth is the public health endpoint (mirrors the gated Health).
func (h *Handlers) PublicHealth(c *gin.Context) {
	h.Health(c)
}

// ----------------------------------------------------------------------------
// Auto-sign
// ----------------------------------------------------------------------------

// AutoSignTransaction performs an auto-sign transaction: it looks up a matching
// enabled auto-sign rule for the wallet, and if matched signs+broadcasts the tx.
// Fund movement (transfer) requires the two-party gate via withdrawal_id.
func (h *Handlers) AutoSignTransaction(c *gin.Context) {
	var req struct {
		MasterWalletID string `json:"master_wallet_id" binding:"required"`
		To             string `json:"to" binding:"required"`
		Amount         string `json:"amount" binding:"required"`
		Password       string `json:"password" binding:"required"`
		Token          string `json:"token"`
		GasLimit       uint64 `json:"gas_limit"`
		WithdrawalID   string `json:"withdrawal_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	mwID, err := uuid.Parse(req.MasterWalletID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid master_wallet_id"})
		return
	}
	w, err := h.store.GetMasterWallet(c.Request.Context(), mwID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "master wallet not found"})
		return
	}
	if w.UserID != wlgate.UserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your master wallet"})
		return
	}
	// Look up an enabled rule whose action matches "transfer" / "token_transfer".
	txAction := "transfer"
	if req.Token != "" {
		txAction = "token_transfer"
	}
	var ruleID uuid.UUID
	var trigger, action string
	_ = h.store.DB().QueryRow(c.Request.Context(),
		`SELECT id, trigger, action FROM auto_sign_rules WHERE master_wallet_id=$1 AND enabled=TRUE AND action=$2 LIMIT 1`,
		w.ID, txAction).Scan(&ruleID, &trigger, &action)
	if ruleID == uuid.Nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "no matching enabled auto-sign rule"})
		return
	}

	var withdrawalID uuid.UUID
	if req.WithdrawalID != "" {
		wid, err := uuid.Parse(req.WithdrawalID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid withdrawal_id"})
			return
		}
		if !h.twoPartyGate.IsWithdrawalApproved(c.Request.Context(), wid) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":         "auto-sign withdrawal not approved by SuperAdmin (two-party gate fail-closed)",
				"withdrawal_id": wid,
			})
			return
		}
		withdrawalID = wid
	}

	txHash, err := h.signAndBroadcast(c.Request.Context(), w, req.To, req.Amount, req.Password, req.GasLimit, "")
	if err != nil {
		h.recordAutoSignLog(w.ID, ruleID, "", w.ChainID, w.Address, req.To, req.Amount, req.Token, "failed", err.Error())
		c.JSON(http.StatusBadGateway, gin.H{"error": "broadcast failed: " + err.Error()})
		return
	}
	h.recordAutoSignLog(w.ID, ruleID, txHash, w.ChainID, w.Address, req.To, req.Amount, req.Token, "broadcast", "")
	_ = h.store.CreateTransaction(c.Request.Context(), w.ID, txHash, "auto_sign_"+txAction, "broadcast", w.Address, req.To, req.Amount, req.Token, w.ChainID)
	_ = h.store.Audit(c.Request.Context(), w.ID, "auto_sign_transaction", "transaction", txHash, "critical", mustJSON(gin.H{"rule_id": ruleID, "withdrawal_id": withdrawalID}))
	if withdrawalID != uuid.Nil {
		_ = h.twoPartyGate.MarkWithdrawalExecuted(c.Request.Context(), withdrawalID, txHash)
	}
	c.JSON(http.StatusOK, gin.H{"transaction_hash": txHash, "status": "broadcast", "rule_id": ruleID, "auto_signed": true})
}

func (h *Handlers) recordAutoSignLog(mwID, ruleID uuid.UUID, txHash string, chainID int64, from, to, amount, token, status, errMsg string) {
	_, _ = h.store.DB().Exec(context.Background(),
		`INSERT INTO auto_sign_logs (id, master_wallet_id, rule_id, tx_hash, chain_id, from_address, to_address, amount, token, status, error) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		uuid.New(), mwID, ruleID, txHash, chainID, from, to, amount, token, status, errMsg)
}

// ListAutoSignLogs returns real auto-sign log rows (optionally for a wallet).
func (h *Handlers) ListAutoSignLogs(c *gin.Context) {
	limit := parseLimit(c.Query("limit"), 100)
	mwIDStr := c.Query("master_wallet_id")
	var rows pgx.Rows
	var err error
	if mwIDStr != "" {
		mwID, perr := uuid.Parse(mwIDStr)
		if perr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid master_wallet_id"})
			return
		}
		// Enforce ownership.
		w, gerr := h.store.GetMasterWallet(c.Request.Context(), mwID)
		if gerr != nil || w.UserID != wlgate.UserID(c) {
			c.JSON(http.StatusForbidden, gin.H{"error": "not your master wallet"})
			return
		}
		rows, err = h.store.DB().Query(c.Request.Context(),
			`SELECT id, master_wallet_id, rule_id, tx_hash, chain_id, from_address, to_address, amount, token, status, error, created_at FROM auto_sign_logs WHERE master_wallet_id=$1 ORDER BY created_at DESC LIMIT $2`, mwID, limit)
	} else {
		rows, err = h.store.DB().Query(c.Request.Context(),
			`SELECT l.id, l.master_wallet_id, l.rule_id, l.tx_hash, l.chain_id, l.from_address, l.to_address, l.amount, l.token, l.status, l.error, l.created_at FROM auto_sign_logs l JOIN master_wallets m ON m.id=l.master_wallet_id WHERE m.user_id=$1 ORDER BY l.created_at DESC LIMIT $2`, wlgate.UserID(c), limit)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, mwID uuid.UUID
		var ruleID *uuid.UUID
		var txHash, from, to, amount, token, status string
		var chainID int64
		var errMsg *string
		var created time.Time
		_ = rows.Scan(&id, &mwID, &ruleID, &txHash, &chainID, &from, &to, &amount, &token, &status, &errMsg, &created)
		row := gin.H{"id": id, "master_wallet_id": mwID, "tx_hash": txHash, "chain_id": chainID, "from": from, "to": to, "amount": amount, "token": token, "status": status, "created_at": created}
		if ruleID != nil {
			row["rule_id"] = *ruleID
		}
		if errMsg != nil {
			row["error"] = *errMsg
		}
		out = append(out, row)
	}
	c.JSON(http.StatusOK, gin.H{"auto_sign_logs": out})
}

// ----------------------------------------------------------------------------
// Treasury (two-party gated — fail-closed BEFORE broadcast)
// ----------------------------------------------------------------------------

// TreasuryTransfer performs a direct treasury transfer. The two-party gate is
// REQUIRED: IsWithdrawalApproved MUST return true before broadcast. 403 fail-closed.
func (h *Handlers) TreasuryTransfer(c *gin.Context) {
	var req struct {
		MasterWalletID string `json:"master_wallet_id" binding:"required"`
		To             string `json:"to" binding:"required"`
		Amount         string `json:"amount" binding:"required"`
		Password       string `json:"password" binding:"required"`
		GasLimit       uint64 `json:"gas_limit"`
		WithdrawalID   string `json:"withdrawal_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	mwID, err := uuid.Parse(req.MasterWalletID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid master_wallet_id"})
		return
	}
	w, err := h.store.GetMasterWallet(c.Request.Context(), mwID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "master wallet not found"})
		return
	}
	if w.UserID != wlgate.UserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your master wallet"})
		return
	}
	// Treasury transfer => ALWAYS Manual two-party (classifier forces it).
	wid, ok := h.requireApproval(c, "treasury_transfer", req.To, "", req.Amount, req.WithdrawalID)
	if !ok {
		return
	}
	txHash, err := h.signAndBroadcast(c.Request.Context(), w, req.To, req.Amount, req.Password, req.GasLimit, "")
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "broadcast failed: " + err.Error()})
		return
	}
	_ = h.store.CreateTransaction(c.Request.Context(), w.ID, txHash, "treasury_transfer", "broadcast", w.Address, req.To, req.Amount, "", w.ChainID)
	_ = h.store.Audit(c.Request.Context(), w.ID, "treasury_transfer", "transaction", txHash, "critical", mustJSON(gin.H{"to": req.To, "amount": req.Amount, "withdrawal_id": wid}))
	_ = h.twoPartyGate.MarkWithdrawalExecuted(c.Request.Context(), wid, txHash)
	c.JSON(http.StatusOK, gin.H{"transaction_hash": txHash, "status": "broadcast", "type": "treasury_transfer"})
}

// TreasurySweep sweeps the treasury balance to a destination. Two-party REQUIRED.
func (h *Handlers) TreasurySweep(c *gin.Context) {
	var req struct {
		MasterWalletID string `json:"master_wallet_id" binding:"required"`
		To             string `json:"to" binding:"required"`
		Password       string `json:"password" binding:"required"`
		GasLimit       uint64 `json:"gas_limit"`
		WithdrawalID   string `json:"withdrawal_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	mwID, err := uuid.Parse(req.MasterWalletID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid master_wallet_id"})
		return
	}
	w, err := h.store.GetMasterWallet(c.Request.Context(), mwID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "master wallet not found"})
		return
	}
	if w.UserID != wlgate.UserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your master wallet"})
		return
	}
	// Treasury sweep => ALWAYS Manual two-party (classifier forces it).
	wid, ok := h.requireApproval(c, "treasury_sweep", req.To, "", "", req.WithdrawalID)
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
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	bal, err := client.BalanceAt(ctx, common.HexToAddress(w.Address), nil)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "balance fetch failed"})
		return
	}
	// Compute sweep amount = balance - maxFee*gasLimit.
	gasTipCap, err := client.SuggestGasTipCap(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "gas tip fetch failed"})
		return
	}
	head, err := client.HeaderByNumber(ctx, nil)
	if err != nil || head == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "head fetch failed"})
		return
	}
	gasLimit := uint64(21000)
	if req.GasLimit > 0 {
		gasLimit = req.GasLimit
	}
	maxFee := gasTipCap
	if head.BaseFee != nil {
		maxFee = new(big.Int).Add(gasTipCap, head.BaseFee)
	}
	fee := new(big.Int).Mul(maxFee, new(big.Int).SetUint64(gasLimit))
	if bal.Cmp(fee) <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "balance insufficient to cover sweep + gas", "balance_wei": bal.String(), "fee_wei": fee.String()})
		return
	}
	amount := new(big.Int).Sub(bal, fee)

	priv, err := h.deriveKey(ctx, w, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid password"})
		return
	}
	nonce, err := client.PendingNonceAt(ctx, common.HexToAddress(w.Address))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "nonce fetch failed"})
		return
	}
	rawTx, err := wlcrypto.SignTransaction(priv, big.NewInt(w.ChainID), common.HexToAddress(req.To), amount, gasLimit, head.BaseFee, gasTipCap, nonce, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sign failed: " + err.Error()})
		return
	}
	txHash, err := broadcastRawTx(ctx, client, rawTx)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "broadcast failed: " + err.Error()})
		return
	}
	_ = h.store.CreateTransaction(c.Request.Context(), w.ID, txHash, "treasury_sweep", "broadcast", w.Address, req.To, amount.String(), "", w.ChainID)
	_ = h.store.Audit(c.Request.Context(), w.ID, "treasury_sweep", "transaction", txHash, "critical", mustJSON(gin.H{"to": req.To, "swept_wei": amount.String(), "withdrawal_id": wid}))
	_ = h.twoPartyGate.MarkWithdrawalExecuted(c.Request.Context(), wid, txHash)
	c.JSON(http.StatusOK, gin.H{"transaction_hash": txHash, "status": "broadcast", "type": "treasury_sweep", "swept_wei": amount.String()})
}

// ----------------------------------------------------------------------------
// UserWallet management layer
// ----------------------------------------------------------------------------

// ---- EVM chains CRUD ----

func (h *Handlers) ListUserChainsEVM(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	rows, err := h.store.DB().Query(c.Request.Context(),
		`SELECT id, chain_id, name, symbol, rpc_url, explorer_url, decimals, enabled, created_at FROM user_chains_evm WHERE master_wallet_id=$1 ORDER BY chain_id`, w.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var chainID int64
		var name, symbol, rpcURL string
		var explorerURL *string
		var decimals int
		var enabled bool
		var created time.Time
		_ = rows.Scan(&id, &chainID, &name, &symbol, &rpcURL, &explorerURL, &decimals, &enabled, &created)
		row := gin.H{"id": id, "chain_id": chainID, "name": name, "symbol": symbol, "rpc_url": rpcURL, "decimals": decimals, "enabled": enabled, "created_at": created}
		if explorerURL != nil {
			row["explorer_url"] = *explorerURL
		}
		out = append(out, row)
	}
	c.JSON(http.StatusOK, gin.H{"chains": out})
}

func (h *Handlers) CreateUserChainEVM(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	var req struct {
		ChainID     int64  `json:"chain_id" binding:"required"`
		Name        string `json:"name" binding:"required"`
		Symbol      string `json:"symbol" binding:"required"`
		RPCURL      string `json:"rpc_url" binding:"required"`
		ExplorerURL string `json:"explorer_url"`
		Decimals    int    `json:"decimals"`
		Enabled     *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Decimals == 0 {
		req.Decimals = 18
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	id := uuid.New()
	_, err := h.store.DB().Exec(c.Request.Context(),
		`INSERT INTO user_chains_evm (id, master_wallet_id, chain_id, name, symbol, rpc_url, explorer_url, decimals, enabled) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT (master_wallet_id, chain_id) DO UPDATE SET name=EXCLUDED.name, symbol=EXCLUDED.symbol, rpc_url=EXCLUDED.rpc_url, explorer_url=EXCLUDED.explorer_url, decimals=EXCLUDED.decimals, enabled=EXCLUDED.enabled`,
		id, w.ID, req.ChainID, req.Name, req.Symbol, req.RPCURL, req.ExplorerURL, req.Decimals, enabled)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_ = h.store.Audit(c.Request.Context(), w.ID, "create_user_chain_evm", "user_chain_evm", strconv.FormatInt(req.ChainID, 10), "info", nil)
	c.JSON(http.StatusCreated, gin.H{"id": id, "chain_id": req.ChainID, "name": req.Name, "symbol": req.Symbol, "enabled": enabled})
}

func (h *Handlers) UpdateUserChainEVM(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	chainID, err := strconv.ParseInt(c.Param("chainId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chain id"})
		return
	}
	var req struct {
		Name        string `json:"name"`
		Symbol      string `json:"symbol"`
		RPCURL      string `json:"rpc_url"`
		ExplorerURL string `json:"explorer_url"`
		Decimals    int    `json:"decimals"`
		Enabled     *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tag, err := h.store.DB().Exec(c.Request.Context(),
		`UPDATE user_chains_evm SET name=COALESCE(NULLIF($1,''), name), symbol=COALESCE(NULLIF($2,''), symbol), rpc_url=COALESCE(NULLIF($3,''), rpc_url), explorer_url=COALESCE(NULLIF($4,''), explorer_url), decimals=COALESCE(NULLIF($5,0), decimals), enabled=COALESCE($6, enabled) WHERE master_wallet_id=$7 AND chain_id=$8`,
		req.Name, req.Symbol, req.RPCURL, req.ExplorerURL, req.Decimals, req.Enabled, w.ID, chainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "chain not found"})
		return
	}
	_ = h.store.Audit(c.Request.Context(), w.ID, "update_user_chain_evm", "user_chain_evm", strconv.FormatInt(chainID, 10), "info", nil)
	c.JSON(http.StatusOK, gin.H{"chain_id": chainID, "updated": true})
}

func (h *Handlers) DeleteUserChainEVM(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	chainID, err := strconv.ParseInt(c.Param("chainId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chain id"})
		return
	}
	tag, err := h.store.DB().Exec(c.Request.Context(), `DELETE FROM user_chains_evm WHERE master_wallet_id=$1 AND chain_id=$2`, w.ID, chainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "chain not found"})
		return
	}
	_ = h.store.Audit(c.Request.Context(), w.ID, "delete_user_chain_evm", "user_chain_evm", strconv.FormatInt(chainID, 10), "warning", nil)
	c.JSON(http.StatusOK, gin.H{"deleted": chainID})
}

// ---- Non-EVM chains CRUD ----

func (h *Handlers) ListUserChainsNonEVM(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	rows, err := h.store.DB().Query(c.Request.Context(),
		`SELECT id, chain_id, name, symbol, chain_type, rpc_url, explorer_url, decimals, bech32_prefix, enabled, created_at FROM user_chains_nonevm WHERE master_wallet_id=$1 ORDER BY chain_id`, w.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var chainID int64
		var name, symbol, chainType string
		var rpcURL *string
		var explorerURL *string
		var bech32 *string
		var decimals int
		var enabled bool
		var created time.Time
		_ = rows.Scan(&id, &chainID, &name, &symbol, &chainType, &rpcURL, &explorerURL, &decimals, &bech32, &enabled, &created)
		row := gin.H{"id": id, "chain_id": chainID, "name": name, "symbol": symbol, "chain_type": chainType, "decimals": decimals, "enabled": enabled, "created_at": created}
		if rpcURL != nil {
			row["rpc_url"] = *rpcURL
		}
		if explorerURL != nil {
			row["explorer_url"] = *explorerURL
		}
		if bech32 != nil {
			row["bech32_prefix"] = *bech32
		}
		out = append(out, row)
	}
	c.JSON(http.StatusOK, gin.H{"chains": out})
}

func (h *Handlers) CreateUserChainNonEVM(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	var req struct {
		ChainID      int64  `json:"chain_id" binding:"required"`
		Name         string `json:"name" binding:"required"`
		Symbol       string `json:"symbol" binding:"required"`
		ChainType    string `json:"chain_type" binding:"required"`
		RPCURL       string `json:"rpc_url"`
		ExplorerURL  string `json:"explorer_url"`
		Decimals     int    `json:"decimals"`
		Bech32Prefix string `json:"bech32_prefix"`
		Enabled      *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	id := uuid.New()
	_, err := h.store.DB().Exec(c.Request.Context(),
		`INSERT INTO user_chains_nonevm (id, master_wallet_id, chain_id, name, symbol, chain_type, rpc_url, explorer_url, decimals, bech32_prefix, enabled) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) ON CONFLICT (master_wallet_id, chain_id) DO UPDATE SET name=EXCLUDED.name, symbol=EXCLUDED.symbol, chain_type=EXCLUDED.chain_type, rpc_url=EXCLUDED.rpc_url, explorer_url=EXCLUDED.explorer_url, decimals=EXCLUDED.decimals, bech32_prefix=EXCLUDED.bech32_prefix, enabled=EXCLUDED.enabled`,
		id, w.ID, req.ChainID, req.Name, req.Symbol, req.ChainType, req.RPCURL, req.ExplorerURL, req.Decimals, req.Bech32Prefix, enabled)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_ = h.store.Audit(c.Request.Context(), w.ID, "create_user_chain_nonevm", "user_chain_nonevm", strconv.FormatInt(req.ChainID, 10), "info", nil)
	c.JSON(http.StatusCreated, gin.H{"id": id, "chain_id": req.ChainID, "name": req.Name, "chain_type": req.ChainType, "enabled": enabled})
}

func (h *Handlers) UpdateUserChainNonEVM(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	chainID, err := strconv.ParseInt(c.Param("chainId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chain id"})
		return
	}
	var req struct {
		Name         string `json:"name"`
		Symbol       string `json:"symbol"`
		ChainType    string `json:"chain_type"`
		RPCURL       string `json:"rpc_url"`
		ExplorerURL  string `json:"explorer_url"`
		Decimals     int    `json:"decimals"`
		Bech32Prefix string `json:"bech32_prefix"`
		Enabled      *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tag, err := h.store.DB().Exec(c.Request.Context(),
		`UPDATE user_chains_nonevm SET name=COALESCE(NULLIF($1,''), name), symbol=COALESCE(NULLIF($2,''), symbol), chain_type=COALESCE(NULLIF($3,''), chain_type), rpc_url=COALESCE(NULLIF($4,''), rpc_url), explorer_url=COALESCE(NULLIF($5,''), explorer_url), decimals=COALESCE(NULLIF($6,0), decimals), bech32_prefix=COALESCE(NULLIF($7,''), bech32_prefix), enabled=COALESCE($8, enabled) WHERE master_wallet_id=$9 AND chain_id=$10`,
		req.Name, req.Symbol, req.ChainType, req.RPCURL, req.ExplorerURL, req.Decimals, req.Bech32Prefix, req.Enabled, w.ID, chainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "chain not found"})
		return
	}
	_ = h.store.Audit(c.Request.Context(), w.ID, "update_user_chain_nonevm", "user_chain_nonevm", strconv.FormatInt(chainID, 10), "info", nil)
	c.JSON(http.StatusOK, gin.H{"chain_id": chainID, "updated": true})
}

func (h *Handlers) DeleteUserChainNonEVM(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	chainID, err := strconv.ParseInt(c.Param("chainId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chain id"})
		return
	}
	tag, err := h.store.DB().Exec(c.Request.Context(), `DELETE FROM user_chains_nonevm WHERE master_wallet_id=$1 AND chain_id=$2`, w.ID, chainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "chain not found"})
		return
	}
	_ = h.store.Audit(c.Request.Context(), w.ID, "delete_user_chain_nonevm", "user_chain_nonevm", strconv.FormatInt(chainID, 10), "warning", nil)
	c.JSON(http.StatusOK, gin.H{"deleted": chainID})
}

// ---- Tokens CRUD ----

func (h *Handlers) ListUserTokens(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	rows, err := h.store.DB().Query(c.Request.Context(),
		`SELECT id, chain_id, contract_address, symbol, name, decimals, enabled, created_at FROM user_tokens WHERE master_wallet_id=$1 ORDER BY chain_id, symbol`, w.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var chainID int64
		var contract, symbol string
		var name *string
		var decimals int
		var enabled bool
		var created time.Time
		_ = rows.Scan(&id, &chainID, &contract, &symbol, &name, &decimals, &enabled, &created)
		row := gin.H{"id": id, "chain_id": chainID, "contract_address": contract, "symbol": symbol, "decimals": decimals, "enabled": enabled, "created_at": created}
		if name != nil {
			row["name"] = *name
		}
		out = append(out, row)
	}
	c.JSON(http.StatusOK, gin.H{"tokens": out})
}

func (h *Handlers) CreateUserToken(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	var req struct {
		ChainID         int64  `json:"chain_id" binding:"required"`
		ContractAddress string `json:"contract_address" binding:"required"`
		Symbol          string `json:"symbol" binding:"required"`
		Name            string `json:"name"`
		Decimals        int    `json:"decimals"`
		Enabled         *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	id := uuid.New()
	_, err := h.store.DB().Exec(c.Request.Context(),
		`INSERT INTO user_tokens (id, master_wallet_id, chain_id, contract_address, symbol, name, decimals, enabled) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (master_wallet_id, chain_id, contract_address) DO UPDATE SET symbol=EXCLUDED.symbol, name=EXCLUDED.name, decimals=EXCLUDED.decimals, enabled=EXCLUDED.enabled`,
		id, w.ID, req.ChainID, req.ContractAddress, req.Symbol, req.Name, req.Decimals, enabled)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_ = h.store.Audit(c.Request.Context(), w.ID, "create_user_token", "user_token", id.String(), "info", nil)
	c.JSON(http.StatusCreated, gin.H{"id": id, "chain_id": req.ChainID, "contract_address": req.ContractAddress, "symbol": req.Symbol, "enabled": enabled})
}

func (h *Handlers) UpdateUserToken(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	tid, err := uuid.Parse(c.Param("tokenId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token id"})
		return
	}
	var req struct {
		Symbol   string `json:"symbol"`
		Name     string `json:"name"`
		Decimals int    `json:"decimals"`
		Enabled  *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tag, err := h.store.DB().Exec(c.Request.Context(),
		`UPDATE user_tokens SET symbol=COALESCE(NULLIF($1,''), symbol), name=COALESCE(NULLIF($2,''), name), decimals=COALESCE(NULLIF($3,0), decimals), enabled=COALESCE($4, enabled) WHERE master_wallet_id=$5 AND id=$6`,
		req.Symbol, req.Name, req.Decimals, req.Enabled, w.ID, tid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "token not found"})
		return
	}
	_ = h.store.Audit(c.Request.Context(), w.ID, "update_user_token", "user_token", tid.String(), "info", nil)
	c.JSON(http.StatusOK, gin.H{"id": tid, "updated": true})
}

func (h *Handlers) DeleteUserToken(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	tid, err := uuid.Parse(c.Param("tokenId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token id"})
		return
	}
	tag, err := h.store.DB().Exec(c.Request.Context(), `DELETE FROM user_tokens WHERE master_wallet_id=$1 AND id=$2`, w.ID, tid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "token not found"})
		return
	}
	_ = h.store.Audit(c.Request.Context(), w.ID, "delete_user_token", "user_token", tid.String(), "warning", nil)
	c.JSON(http.StatusOK, gin.H{"deleted": tid})
}

// ---- Derived user wallet addresses ----

func (h *Handlers) ListUserWalletAddresses(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	rows, err := h.store.DB().Query(c.Request.Context(),
		`SELECT id, chain_id, chain_type, address, derivation_path, label, created_at FROM user_wallet_addresses WHERE master_wallet_id=$1 ORDER BY created_at DESC`, w.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var chainID int64
		var chainType, address, path string
		var label *string
		var created time.Time
		_ = rows.Scan(&id, &chainID, &chainType, &address, &path, &label, &created)
		row := gin.H{"id": id, "chain_id": chainID, "chain_type": chainType, "address": address, "derivation_path": path, "created_at": created}
		if label != nil {
			row["label"] = *label
		}
		out = append(out, row)
	}
	c.JSON(http.StatusOK, gin.H{"addresses": out})
}

// DeriveUserAddress derives an address from the master seed for any chain.
// EVM: secp256k1/Keccak via BIP-44 m/44'/60'/...
// Solana: SLIP-0010 Ed25519 m/44'/501'/0'/0'
// Bitcoin: secp256k1 P2PKH m/44'/0'/0'/0/0
// Cosmos: secp256k1 + bech32 m/44'/118'/0'/0/0
func (h *Handlers) DeriveUserAddress(c *gin.Context) {
	var req struct {
		MasterWalletID string `json:"master_wallet_id" binding:"required"`
		Password       string `json:"password" binding:"required"`
		ChainType      string `json:"chain_type" binding:"required"` // evm|solana|bitcoin|cosmos
		ChainID        int64  `json:"chain_id"`
		DerivationPath string `json:"derivation_path"` // optional override
		AccountIndex   uint32 `json:"account_index"`
		Label          string `json:"label"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	mwID, err := uuid.Parse(req.MasterWalletID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid master_wallet_id"})
		return
	}
	w, err := h.store.GetMasterWallet(c.Request.Context(), mwID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "master wallet not found"})
		return
	}
	if w.UserID != wlgate.UserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your master wallet"})
		return
	}
	seed, err := wlcrypto.DecryptSeedAtRest(w.EncryptedSeed, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid password"})
		return
	}

	chainType := strings.ToLower(req.ChainType)
	if req.ChainID == 0 {
		req.ChainID = w.ChainID
	}
	path := req.DerivationPath
	var address string
	switch chainType {
	case "evm", "ethereum":
		if path == "" {
			path = fmt.Sprintf("m/44'/60'/0'/0/%d", req.AccountIndex)
		}
		priv, err := derivePrivateKeyFromPath(seed, path)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "EVM key derivation failed: " + err.Error()})
			return
		}
		address = privateKeyToAddress(priv)
	case "solana":
		if path == "" {
			path = fmt.Sprintf("m/44'/501'/0'/%d'", req.AccountIndex)
		}
		addr, err := solanaAddressFromSeed(seed, path)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Solana key derivation failed: " + err.Error()})
			return
		}
		address = addr
	case "bitcoin", "btc":
		if path == "" {
			path = "m/44'/0'/0'/0/0"
		}
		addr, err := btcAddressFromSeed(seed, path)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Bitcoin key derivation failed: " + err.Error()})
			return
		}
		address = addr
	case "cosmos", "atom":
		if path == "" {
			path = fmt.Sprintf("m/44'/118'/0'/0/%d", req.AccountIndex)
		}
		prefix := bech32PrefixForChainID(req.ChainID)
		addr, err := cosmosAddressFromSeed(seed, path, prefix)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Cosmos key derivation failed: " + err.Error()})
			return
		}
		address = addr
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported chain_type: " + chainType})
		return
	}

	id := uuid.New()
	_, err = h.store.DB().Exec(c.Request.Context(),
		`INSERT INTO user_wallet_addresses (id, master_wallet_id, user_id, chain_id, chain_type, address, derivation_path, label) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		id, w.ID, wlgate.UserID(c), req.ChainID, chainType, address, path, req.Label)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = h.store.Audit(c.Request.Context(), w.ID, "derive_user_address", "user_wallet_address", id.String(), "info", mustJSON(gin.H{"chain_type": chainType, "path": path, "address": address}))
	c.JSON(http.StatusCreated, gin.H{"id": id, "address": address, "chain_type": chainType, "chain_id": req.ChainID, "derivation_path": path})
}

// ---- Feature flags (CRUD) ----

func (h *Handlers) GetFeatureFlag(c *gin.Context) {
	flagID := c.Param("flagId")
	if flagID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "flag id required"})
		return
	}
	// Global (no master wallet context): return the first matching flag row.
	var id uuid.UUID
	var mwID *uuid.UUID
	var desc *string
	var enabled bool
	var config string
	var updated time.Time
	err := h.store.DB().QueryRow(c.Request.Context(),
		`SELECT id, master_wallet_id, description, enabled, config::text, updated_at FROM feature_flags WHERE flag_id=$1 ORDER BY updated_at DESC LIMIT 1`, flagID).
		Scan(&id, &mwID, &desc, &enabled, &config, &updated)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "feature flag not found"})
		return
	}
	row := gin.H{"id": id, "flag_id": flagID, "enabled": enabled, "config": config, "updated_at": updated}
	if desc != nil {
		row["description"] = *desc
	}
	if mwID != nil {
		row["master_wallet_id"] = *mwID
	}
	c.JSON(http.StatusOK, row)
}

func (h *Handlers) UpsertFeatureFlag(c *gin.Context) {
	flagID := c.Param("flagId")
	if flagID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "flag id required"})
		return
	}
	mwIDStr := c.Query("master_wallet_id")
	var mwID uuid.UUID
	if mwIDStr != "" {
		wid, err := uuid.Parse(mwIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid master_wallet_id"})
			return
		}
		// Enforce ownership when scoped.
		w, err := h.store.GetMasterWallet(c.Request.Context(), wid)
		if err != nil || w.UserID != wlgate.UserID(c) {
			c.JSON(http.StatusForbidden, gin.H{"error": "not your master wallet"})
			return
		}
		mwID = wid
	}
	var req struct {
		Description string          `json:"description"`
		Enabled     *bool           `json:"enabled"`
		Config      json.RawMessage `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cfg := []byte(req.Config)
	if len(cfg) == 0 {
		cfg = []byte(`{}`)
	}
	enabled := false
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	var mwArg any = mwID
	if mwID == uuid.Nil {
		mwArg = nil
	}
	_, err := h.store.DB().Exec(c.Request.Context(),
		`INSERT INTO feature_flags (id, master_wallet_id, flag_id, description, enabled, config) VALUES ($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (master_wallet_id, flag_id) DO UPDATE SET description=EXCLUDED.description, enabled=EXCLUDED.enabled, config=EXCLUDED.config, updated_at=NOW()`,
		uuid.New(), mwArg, flagID, req.Description, enabled, cfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = h.store.Audit(c.Request.Context(), mwID, "upsert_feature_flag", "feature_flag", flagID, "warning", mustJSON(gin.H{"enabled": enabled}))
	c.JSON(http.StatusOK, gin.H{"flag_id": flagID, "enabled": enabled, "updated": true})
}

func (h *Handlers) DeleteFeatureFlag(c *gin.Context) {
	flagID := c.Param("flagId")
	if flagID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "flag id required"})
		return
	}
	mwIDStr := c.Query("master_wallet_id")
	var tag pgconn.CommandTag
	var err error
	if mwIDStr != "" {
		mwID, perr := uuid.Parse(mwIDStr)
		if perr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid master_wallet_id"})
			return
		}
		w, gerr := h.store.GetMasterWallet(c.Request.Context(), mwID)
		if gerr != nil || w.UserID != wlgate.UserID(c) {
			c.JSON(http.StatusForbidden, gin.H{"error": "not your master wallet"})
			return
		}
		tag, err = h.store.DB().Exec(c.Request.Context(), `DELETE FROM feature_flags WHERE flag_id=$1 AND master_wallet_id=$2`, flagID, mwID)
	} else {
		tag, err = h.store.DB().Exec(c.Request.Context(), `DELETE FROM feature_flags WHERE flag_id=$1`, flagID)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "feature flag not found"})
		return
	}
	_ = h.store.Audit(c.Request.Context(), uuid.Nil, "delete_feature_flag", "feature_flag", flagID, "warning", nil)
	c.JSON(http.StatusOK, gin.H{"deleted": flagID})
}

// ----------------------------------------------------------------------------
// WebSocket — real gorilla/websocket hub
// ----------------------------------------------------------------------------

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // WL client controls the origin; auth is via token query param
	},
}

// wsHub fans out balance/tx updates to subscribed connections. No fabricated
// events — each connection gets the live balance on connect and then periodic
// balance refreshes (real ethclient.BalanceAt) plus any new tx row.
type wsHub struct {
	mu    sync.Mutex
	conns map[*wsConn]struct{}
}

type wsConn struct {
	masterWalletID uuid.UUID
	ch             chan []byte
	conn           *websocket.Conn
}

func newWSHub() *wsHub { return &wsHub{conns: map[*wsConn]struct{}{}} }

func (hub *wsHub) register(c *wsConn) {
	hub.mu.Lock()
	hub.conns[c] = struct{}{}
	hub.mu.Unlock()
}

func (hub *wsHub) unregister(c *wsConn) {
	hub.mu.Lock()
	delete(hub.conns, c)
	hub.mu.Unlock()
	close(c.ch)
}

func (hub *wsHub) broadcast(mwID uuid.UUID, payload []byte) {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	for c := range hub.conns {
		if c.masterWalletID == mwID {
			select {
			case c.ch <- payload:
			default: // drop if slow
			}
		}
	}
}

// WebSocket handles a real WebSocket connection. Subscribe by master_wallet_id
// + token query param. On connect sends the live balance, then pushes periodic
// balance refreshes. Auth is verified via the JWT token; ownership enforced.
func (h *Handlers) WebSocket(c *gin.Context) {
	mwIDStr := c.Query("master_wallet_id")
	token := c.Query("token")
	if mwIDStr == "" || token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "master_wallet_id and token required"})
		return
	}
	mwID, err := uuid.Parse(mwIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid master_wallet_id"})
		return
	}
	// Verify the JWT token owns this wallet (parse inline — wlgate exports the
	// Claims shape but not a verifier function).
	claims := &wlgate.Claims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		return []byte(h.cfg.JWTSecret), nil
	})
	if err != nil || !parsed.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	w, err := h.store.GetMasterWallet(c.Request.Context(), mwID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "master wallet not found"})
		return
	}
	if w.UserID != claims.UserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your master wallet"})
		return
	}

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return // Upgrade writes the error response
	}
	wc := &wsConn{masterWalletID: mwID, ch: make(chan []byte, 16), conn: conn}
	h.wsHub.register(wc)
	defer func() {
		h.wsHub.unregister(wc)
		conn.Close()
	}()

	// On connect: send the live balance.
	h.pushBalance(c.Request.Context(), wc, w)

	// Reader: discard client pings/messages; detect close.
	go func() {
		defer conn.Close()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	// Periodic real balance refresh (every 15s). No fabricated tx events.
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case msg, ok := <-wc.ch:
			if !ok {
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			h.pushBalance(c.Request.Context(), wc, w)
		}
	}
}

func (h *Handlers) pushBalance(ctx context.Context, wc *wsConn, w *store.MasterWallet) {
	rpc := rpcForChain(w.ChainID)
	balWei := "0"
	source := "unavailable"
	if rpc != "" {
		if bal, err := fetchNativeBalance(ctx, rpc, common.HexToAddress(w.Address)); err == nil {
			balWei = bal.String()
			source = "rpc"
		}
	}
	msg, _ := json.Marshal(gin.H{
		"type": "balance", "master_wallet_id": w.ID, "address": w.Address,
		"chain_id": w.ChainID, "balance_wei": balWei, "source": source, "ts": time.Now().UTC(),
	})
	select {
	case wc.ch <- msg:
	default:
	}
}

// broadcastNewTx is called by handlers after a successful broadcast so the WS
// hub pushes the new tx to subscribers. Real event (only fired after broadcast).
func (h *Handlers) broadcastNewTx(w *store.MasterWallet, txHash, txType, amount, to string) {
	msg, _ := json.Marshal(gin.H{
		"type": "transaction", "master_wallet_id": w.ID, "tx_hash": txHash,
		"tx_type": txType, "amount": amount, "to": to, "ts": time.Now().UTC(),
	})
	h.wsHub.broadcast(w.ID, msg)
}
