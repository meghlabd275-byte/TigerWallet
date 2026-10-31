package main

// multisig.go — On-chain-style threshold multisig wallet management. Creates
// multisig wallets (threshold + owners), collects owner ECDSA signatures, and
// executes a transaction once threshold signatures are gathered. All real
// secp256k1 verification via go-ethereum/crypto; no length checks.

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type createMultisigReq struct {
	Name      string   `json:"name" binding:"required"`
	ChainID   int64    `json:"chain_id"`
	Threshold int      `json:"threshold" binding:"required"`
	Owners    []string `json:"owners" binding:"required"` // EVM addresses
}

func (svc *Service) CreateMultisigWallet(c *gin.Context) {
	masterID, ok := svc.requireMasterWalletAccess(c)
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
	// Validate + dedupe owners.
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
	mid := uuid.New().String()
	ctx := c.Request.Context()
	_, err := svc.store.db.Exec(ctx,
		`INSERT INTO multisig_wallets (id, master_wallet_id, name, chain_id, threshold, owners) VALUES ($1,$2,$3,$4,$5,$6)`,
		mid, masterID, req.Name, chainID, req.Threshold, cleanOwners)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	svc.store.audit(ctx, masterID, "multisig.create", "multisig", "user", currentUserID(c), "multisig", mid, "high", gin.H{"threshold": req.Threshold, "owners": cleanOwners})
	c.JSON(http.StatusCreated, gin.H{
		"id": mid, "name": req.Name, "chain_id": chainID,
		"threshold": req.Threshold, "owners": cleanOwners, "created_at": time.Now().UTC(),
	})
}

func (svc *Service) GetMultisigWallets(c *gin.Context) {
	masterID, ok := svc.requireMasterWalletAccess(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	rows, err := svc.store.db.Query(ctx,
		`SELECT id, name, chain_id, threshold, owners, nonce, created_at FROM multisig_wallets WHERE master_wallet_id = $1`, masterID)
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
		_ = rows.Scan(&id, &name, &chainID, &threshold, &owners, &nonce, &createdAt)
		out = append(out, gin.H{"id": id.String(), "name": name, "chain_id": chainID, "threshold": threshold, "owners": owners, "nonce": nonce, "created_at": createdAt})
	}
	c.JSON(http.StatusOK, gin.H{"multisig_wallets": out})
}

type createMultisigTxReq struct {
	ToAddress string `json:"to_address" binding:"required"`
	Value     string `json:"value" binding:"required"`
	Data      string `json:"data"`
}

func (svc *Service) CreateMultisigTransaction(c *gin.Context) {
	masterID, ok := svc.requireMasterWalletAccess(c)
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
	// Scoped by master_wallet_id so a wid belonging to another master wallet is
	// not addressable through this URL.
	err := svc.store.db.QueryRow(ctx,
		`SELECT nonce, threshold FROM multisig_wallets WHERE id = $1 AND master_wallet_id = $2`,
		walletID, masterID).Scan(&nonce, &threshold)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "multisig wallet not found"})
		return
	}
	txID := uuid.New().String()
	_, err = svc.store.db.Exec(ctx,
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
	Signature   string `json:"signature" binding:"required"`    // 65-byte hex r||s||v
	Signer      string `json:"signer" binding:"required"`       // address of signer
	MessageHash string `json:"message_hash" binding:"required"` // the hash that was signed
}

// SignMultisigTransaction verifies a real ECDSA signature against the message
// hash, checks the signer is an owner, and records it. When threshold is met,
// the transaction can be executed.
func (svc *Service) SignMultisigTransaction(c *gin.Context) {
	if _, ok := svc.requireMasterWalletAccess(c); !ok {
		return
	}
	// The route is /:id/multisig/transactions/:tid/sign. Reading :id here used the
	// master wallet id as the transaction id, so no transaction was ever found.
	txID := c.Param("tid")
	var req signMultisigTxReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Verify the signature is real secp256k1.
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
	// go-ethereum's crypto.Ecrecover returns the public key for (hash, sig) where
	// the last byte is the recovery id (0 or 1).
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
	// Verify signer is an owner.
	ctx := c.Request.Context()
	var owners []string
	var walletID string
	var threshold int
	err = svc.store.db.QueryRow(ctx,
		`SELECT id::text, threshold, owners FROM multisig_wallets mw
		 WHERE mw.id = (SELECT multisig_wallet_id FROM multisig_transactions WHERE id = $1)`,
		txID).Scan(&walletID, &threshold, &owners)
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
	// Append the signature, refusing a second entry from a signer who has already
	// signed. The NOT EXISTS is evaluated by PostgreSQL inside the UPDATE, so two
	// concurrent requests from the same owner cannot both append and inflate the
	// count towards the threshold.
	signerHex := signerAddr.Hex()
	sigEntry := `[{"signer":"` + signerHex + `","signature":"` + req.Signature +
		`","message_hash":"` + req.MessageHash + `","signed_at":"` + time.Now().UTC().Format(time.RFC3339) + `"}]`
	tag, err := svc.store.db.Exec(ctx,
		`UPDATE multisig_transactions
		 SET signatures = signatures || $1::jsonb
		 WHERE id = $2
		   AND NOT EXISTS (
		         SELECT 1 FROM jsonb_array_elements(signatures) e
		         WHERE lower(e->>'signer') = lower($3)
		       )`,
		sigEntry, txID, signerHex)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "this signer has already signed this transaction"})
		return
	}
	// Count signatures.
	var sigCount int
	_ = svc.store.db.QueryRow(ctx, `SELECT jsonb_array_length(signatures) FROM multisig_transactions WHERE id = $1`, txID).Scan(&sigCount)
	canExecute := sigCount >= threshold
	svc.store.audit(ctx, "", "multisig.sign", "multisig", "user", currentUserID(c), "multisig_tx", txID, "normal", gin.H{"signer": signerAddr.Hex(), "signature_count": sigCount})
	c.JSON(http.StatusOK, gin.H{"transaction_id": txID, "signature_count": sigCount, "threshold": threshold, "can_execute": canExecute})
}

func (svc *Service) GetMultisigTransactions(c *gin.Context) {
	masterID, ok := svc.requireMasterWalletAccess(c)
	if !ok {
		return
	}
	// The route is /:id/multisig/wallets/:wid/transactions. Reading :id here used
	// the master wallet id as the multisig wallet id, so the list was always empty.
	walletID := c.Param("wid")
	ctx := c.Request.Context()
	// Join through multisig_wallets so the wid must belong to the master wallet in
	// the URL, not merely exist.
	rows, err := svc.store.db.Query(ctx,
		`SELECT mt.id, mt.to_address, mt.value, mt.data, mt.nonce, mt.status, mt.signatures, mt.created_at
		 FROM multisig_transactions mt
		 JOIN multisig_wallets mw ON mw.id = mt.multisig_wallet_id
		 WHERE mt.multisig_wallet_id = $1 AND mw.master_wallet_id = $2
		 ORDER BY mt.created_at DESC`, walletID, masterID)
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
		_ = rows.Scan(&id, &toAddr, &value, &data, &nonce, &status, &sigs, &createdAt)
		out = append(out, gin.H{"id": id.String(), "to_address": toAddr, "value": value, "data": data, "nonce": nonce, "status": status, "signatures": rawJSON(sigs), "created_at": createdAt})
	}
	c.JSON(http.StatusOK, gin.H{"transactions": out})
}

// ExecuteMultisigTransaction broadcasts the assembled multisig transaction once
// threshold signatures are collected. Uses a configured executor key (env) to
// submit the assembled call; fail-closed when no executor key is set.
func (svc *Service) ExecuteMultisigTransaction(c *gin.Context) {
	if _, ok := svc.requireMasterWalletAccess(c); !ok {
		return
	}
	// Same :id/:tid mix-up as SignMultisigTransaction; the route parameter that
	// identifies the transaction is :tid.
	txID := c.Param("tid")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	var toAddr, value, data string
	var walletID string
	var threshold int
	var status string
	err := svc.store.db.QueryRow(ctx,
		`SELECT mt.to_address, mt.value, mt.data, mw.id::text, mw.threshold, mt.status
		 FROM multisig_transactions mt JOIN multisig_wallets mw ON mt.multisig_wallet_id = mw.id
		 WHERE mt.id = $1`, txID).Scan(&toAddr, &value, &data, &walletID, &threshold, &status)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "multisig transaction not found"})
		return
	}
	// Refuse to broadcast twice. Without this an execute replay would submit the
	// same call again under a fresh nonce, moving the funds a second time.
	if status != "pending" {
		c.JSON(http.StatusConflict, gin.H{"error": "transaction is not pending (status=" + status + ")"})
		return
	}
	var sigCount int
	_ = svc.store.db.QueryRow(ctx, `SELECT jsonb_array_length(signatures) FROM multisig_transactions WHERE id = $1`, txID).Scan(&sigCount)
	if sigCount < threshold {
		c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient signatures", "have": sigCount, "required": threshold})
		return
	}
	execKey := svc.cfg.TreasuryKeyHex
	if execKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "executor key not configured (set MASTER_WALLET_TREASURY_KEY_HEX)", "status": "requires_execution"})
		return
	}
	privKey, err := parsePrivateKeyHex(execKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid executor key"})
		return
	}
	var chainID int64
	_ = svc.store.db.QueryRow(ctx, `SELECT chain_id FROM multisig_wallets WHERE id = $1`, walletID).Scan(&chainID)
	rpc := rpcEndpointForChain(chainID)
	if rpc == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "RPC not configured"})
		return
	}
	from := PrivateKeyToAddress(privKey)
	nonce, err := FetchTransactionCount(ctx, rpc, from)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	_, maxFee, prio, err := FetchGasPrice(ctx, rpc)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
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
	rawTx, err := SignEVMTransaction(big.NewInt(chainID), nonce, common.HexToAddress(toAddr), wei, 100000, maxFee, prio, callData, privKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	txHash, err := BroadcastTransaction(ctx, rpc, rawTx)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	// The status flip is conditional on 'pending' and happens after the broadcast.
	// If it does not match, another replica executed this transaction between our
	// read and here; the tx is already on-chain, so report that rather than
	// pretending we are the one who executed it.
	tag, err := svc.store.db.Exec(ctx,
		`UPDATE multisig_transactions SET status='executed', executed_at=NOW()
		 WHERE id=$1 AND status='pending'`, txID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "broadcast succeeded but status update failed", "transaction_hash": txHash})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "transaction was already executed", "transaction_hash": txHash})
		return
	}
	svc.store.audit(ctx, "", "multisig.execute", "multisig", "user", currentUserID(c), "multisig_tx", txID, "high", gin.H{"tx_hash": txHash})
	c.JSON(http.StatusOK, gin.H{"transaction_hash": txHash, "multisig_tx_id": txID, "status": "executed"})
}

// used to satisfy unused import in big.Int when no other ref — safe to keep.
var _ = fmt.Sprintf
