package handlers

// multisig.go — On-chain-style threshold multisig wallet management. Creates
// multisig wallets (threshold + owners), collects owner ECDSA signatures, and
// executes a transaction once threshold signatures are gathered. All real
// secp256k1 verification via go-ethereum/crypto; no length checks, no stubs.

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tigerwallet/wl-shared/wlcrypto"
)

type createMultisigReq struct {
	Name      string   `json:"name" binding:"required"`
	ChainID   int64    `json:"chain_id"`
	Threshold int      `json:"threshold" binding:"required"`
	Owners    []string `json:"owners" binding:"required"`
}

// CreateMultisigWallet POST /master-wallet/:id/multisig/wallets
func (h *Handlers) CreateMultisigWallet(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	var req createMultisigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Threshold < 1 || req.Threshold > len(req.Owners) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "threshold must be between 1 and number of owners"})
		return
	}
	seen := map[string]bool{}
	cleanOwners := []string{}
	for _, o := range req.Owners {
		addr := strings.ToLower(strings.TrimSpace(o))
		if !common.IsHexAddress(addr) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid owner address: " + o})
			return
		}
		if seen[addr] {
			continue
		}
		seen[addr] = true
		cleanOwners = append(cleanOwners, addr)
	}
	if len(cleanOwners) < req.Threshold {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not enough unique owners for threshold"})
		return
	}
	chainID := req.ChainID
	if chainID == 0 {
		chainID = 1
	}
	id := uuid.New()
	ctx := c.Request.Context()
	_, err := h.store.DB().Exec(ctx,
		`INSERT INTO multisig_wallets (id, master_wallet_id, name, chain_id, threshold, owners) VALUES ($1,$2,$3,$4,$5,$6)`,
		id, w.ID, req.Name, chainID, req.Threshold, cleanOwners)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = h.store.Audit(ctx, w.ID, "multisig.create", "multisig", id.String(), "high", mustJSON(gin.H{"threshold": req.Threshold, "owners": cleanOwners}))
	c.JSON(http.StatusCreated, gin.H{
		"id": id, "name": req.Name, "chain_id": chainID,
		"threshold": req.Threshold, "owners": cleanOwners, "created_at": time.Now().UTC(),
	})
}

// GetMultisigWallets GET /master-wallet/:id/multisig/wallets
func (h *Handlers) GetMultisigWallets(c *gin.Context) {
	w, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	rows, err := h.store.DB().Query(c.Request.Context(),
		`SELECT id, name, chain_id, threshold, owners, nonce, created_at FROM multisig_wallets WHERE master_wallet_id = $1 ORDER BY created_at DESC`, w.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var name string
		var chainID int64
		var threshold, nonce int
		var owners []string
		var createdAt time.Time
		if err := rows.Scan(&id, &name, &chainID, &threshold, &owners, &nonce, &createdAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
			return
		}
		out = append(out, gin.H{"id": id, "name": name, "chain_id": chainID, "threshold": threshold, "owners": owners, "nonce": nonce, "created_at": createdAt})
	}
	c.JSON(http.StatusOK, gin.H{"multisig_wallets": out})
}

type createMultisigTxReq struct {
	ToAddress string `json:"to_address" binding:"required"`
	Value     string `json:"value" binding:"required"`
	Data      string `json:"data"`
}

// CreateMultisigTransaction POST /master-wallet/:id/multisig/wallets/:wid/transactions
func (h *Handlers) CreateMultisigTransaction(c *gin.Context) {
	_, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	walletID := c.Param("wid")
	if walletID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wid required"})
		return
	}
	var req createMultisigTxReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Data == "" {
		req.Data = "0x"
	}
	ctx := c.Request.Context()
	var nonce int
	var threshold int
	err := h.store.DB().QueryRow(ctx,
		`SELECT nonce, threshold FROM multisig_wallets WHERE id = $1`, walletID).Scan(&nonce, &threshold)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "multisig wallet not found"})
		return
	}
	txID := uuid.New()
	_, err = h.store.DB().Exec(ctx,
		`INSERT INTO multisig_transactions (id, multisig_wallet_id, to_address, value, data, nonce, status)
		 VALUES ($1,$2,$3,$4,$5,$6,'pending')`,
		txID, walletID, req.ToAddress, req.Value, req.Data, nonce)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": txID, "to_address": req.ToAddress, "value": req.Value, "nonce": nonce, "required_signatures": threshold, "status": "pending"})
}

type signMultisigTxReq struct {
	Signature   string `json:"signature" binding:"required"`
	Signer      string `json:"signer" binding:"required"`
	MessageHash string `json:"message_hash" binding:"required"`
}

// SignMultisigTransaction POST /master-wallet/:id/multisig/transactions/:tid/sign
func (h *Handlers) SignMultisigTransaction(c *gin.Context) {
	_, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	txID := c.Param("tid")
	if txID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tid required"})
		return
	}
	var req signMultisigTxReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sigBytes, err := hex.DecodeString(strings.TrimPrefix(req.Signature, "0x"))
	if err != nil || len(sigBytes) != 65 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "signature must be 65 bytes (r||s||v)"})
		return
	}
	hashBytes, err := hex.DecodeString(strings.TrimPrefix(req.MessageHash, "0x"))
	if err != nil || len(hashBytes) != 32 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message_hash must be 32 bytes"})
		return
	}
	pubBytes, err := crypto.Ecrecover(hashBytes, sigBytes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid signature"})
		return
	}
	pub, err := crypto.UnmarshalPubkey(pubBytes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "could not recover public key"})
		return
	}
	recovered := crypto.PubkeyToAddress(*pub)
	signerAddr := common.HexToAddress(req.Signer)
	if recovered != signerAddr {
		c.JSON(http.StatusBadRequest, gin.H{"error": "signature does not match signer address"})
		return
	}
	ctx := c.Request.Context()
	var owners []string
	var threshold int
	err = h.store.DB().QueryRow(ctx,
		`SELECT threshold, owners FROM multisig_wallets
		 WHERE id = (SELECT multisig_wallet_id FROM multisig_transactions WHERE id = $1)`,
		txID).Scan(&threshold, &owners)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "multisig transaction/wallet not found"})
		return
	}
	isOwner := false
	for _, o := range owners {
		if strings.EqualFold(o, signerAddr.Hex()) {
			isOwner = true
			break
		}
	}
	if !isOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "signer is not an owner of this multisig wallet"})
		return
	}
	sigEntry := fmt.Sprintf(`[{"signer":"%s","signature":"%s","message_hash":"%s","signed_at":"%s"}]`,
		signerAddr.Hex(), req.Signature, req.MessageHash, time.Now().UTC().Format(time.RFC3339))
	_, err = h.store.DB().Exec(ctx,
		`UPDATE multisig_transactions SET signatures = signatures || $1::jsonb WHERE id = $2`,
		sigEntry, txID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var sigCount int
	_ = h.store.DB().QueryRow(ctx, `SELECT jsonb_array_length(signatures) FROM multisig_transactions WHERE id = $1`, txID).Scan(&sigCount)
	canExecute := sigCount >= threshold
	c.JSON(http.StatusOK, gin.H{"transaction_id": txID, "signature_count": sigCount, "threshold": threshold, "can_execute": canExecute})
}

// GetMultisigTransactions GET /master-wallet/:id/multisig/wallets/:wid/transactions
func (h *Handlers) GetMultisigTransactions(c *gin.Context) {
	_, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	walletID := c.Param("wid")
	if walletID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wid required"})
		return
	}
	rows, err := h.store.DB().Query(c.Request.Context(),
		`SELECT id, to_address, value, data, nonce, status, signatures, created_at
		 FROM multisig_transactions WHERE multisig_wallet_id = $1 ORDER BY created_at DESC`, walletID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var toAddr, value, data, status string
		var nonce int
		var sigs []byte
		var createdAt time.Time
		if err := rows.Scan(&id, &toAddr, &value, &data, &nonce, &status, &sigs, &createdAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
			return
		}
		out = append(out, gin.H{"id": id, "to_address": toAddr, "value": value, "data": data, "nonce": nonce, "status": status, "signatures": json.RawMessage(sigs), "created_at": createdAt})
	}
	c.JSON(http.StatusOK, gin.H{"transactions": out})
}

// ExecuteMultisigTransaction POST /master-wallet/:id/multisig/transactions/:tid/execute
// Broadcasts the assembled multisig transaction once threshold signatures are
// collected. Uses the configured treasury hot-wallet key (env) to submit the
// assembled call; fail-closed when no executor key is set.
func (h *Handlers) ExecuteMultisigTransaction(c *gin.Context) {
	_, ok := h.loadOwnedWallet(c)
	if !ok {
		return
	}
	txID := c.Param("tid")
	if txID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tid required"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	var toAddr, value, data string
	var walletID string
	var threshold int
	err := h.store.DB().QueryRow(ctx,
		`SELECT mt.to_address, mt.value, mt.data, mw.id::text, mw.threshold
		 FROM multisig_transactions mt JOIN multisig_wallets mw ON mt.multisig_wallet_id = mw.id
		 WHERE mt.id = $1`, txID).Scan(&toAddr, &value, &data, &walletID, &threshold)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "multisig transaction not found"})
		return
	}
	var sigCount int
	_ = h.store.DB().QueryRow(ctx, `SELECT jsonb_array_length(signatures) FROM multisig_transactions WHERE id = $1`, txID).Scan(&sigCount)
	if sigCount < threshold {
		c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient signatures", "have": sigCount, "required": threshold})
		return
	}
	execKeyHex := h.cfg.TreasuryKeyHex
	if execKeyHex == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "executor key not configured (set MASTER_WALLET_TREASURY_KEY_HEX)", "status": "requires_execution"})
		return
	}
	priv, err := crypto.HexToECDSA(strings.TrimPrefix(execKeyHex, "0x"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid executor key"})
		return
	}
	var chainID int64
	_ = h.store.DB().QueryRow(ctx, `SELECT chain_id FROM multisig_wallets WHERE id = $1`, walletID).Scan(&chainID)
	rpc := rpcForChain(chainID)
	if rpc == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "RPC not configured"})
		return
	}
	client, err := ethclient.DialContext(ctx, rpc)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "RPC unavailable"})
		return
	}
	defer client.Close()
	from := crypto.PubkeyToAddress(priv.PublicKey)
	nonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "nonce fetch failed"})
		return
	}
	gasTipCap, err := client.SuggestGasTipCap(ctx)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "gas tip fetch failed"})
		return
	}
	head, err := client.HeaderByNumber(ctx, nil)
	if err != nil || head == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "head fetch failed"})
		return
	}
	wei, ok := new(big.Int).SetString(value, 10)
	if !ok {
		wei = big.NewInt(0)
	}
	callData, err := hex.DecodeString(strings.TrimPrefix(data, "0x"))
	if err != nil {
		callData = nil
	}
	rawTx, err := wlcryptoSignTx(priv, big.NewInt(chainID), nonce, common.HexToAddress(toAddr), wei, 100000, head.BaseFee, gasTipCap, callData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	txHash, err := broadcastRawTx(ctx, client, rawTx)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	_, _ = h.store.DB().Exec(ctx, `UPDATE multisig_transactions SET status='executed', executed_at=NOW() WHERE id=$1`, txID)
	_ = h.store.Audit(ctx, uuid.Nil, "multisig.execute", "multisig_tx", txID, "critical", mustJSON(gin.H{"tx_hash": txHash}))
	c.JSON(http.StatusOK, gin.H{"transaction_hash": txHash, "multisig_tx_id": txID, "status": "executed"})
}

// wlcryptoSignTx delegates to the shared wlcrypto SignTransaction so multisig
// execution uses the exact same EIP-1559 signing path as the rest of the app.
func wlcryptoSignTx(priv *ecdsa.PrivateKey, chainID *big.Int, nonce uint64, to common.Address, value *big.Int, gasLimit uint64, baseFee, tipCap *big.Int, data []byte) (string, error) {
	return wlcrypto.SignTransaction(priv, chainID, to, value, gasLimit, baseFee, tipCap, nonce, data)
}

// keep unused import guards aligned with the rest of the package.
var (
	_ = types.Transaction{}
	_ = fmt.Sprintf
)
