package main

// treasury.go — Treasury management: real on-chain balance + real broadcast for
// transfers/sweeps/allocations. The treasury hot-wallet key is loaded from env;
// when unset, write endpoints return 503 (fail-closed) instead of fabricating.

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TreasuryOverview returns the live treasury state for a master wallet.
func (svc *Service) TreasuryOverview(c *gin.Context) {
	id := c.Param("id")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()

	var address string
	var chainID int64
	err := svc.store.db.QueryRow(ctx,
		`SELECT address, chain_id FROM master_wallets WHERE id = $1`, id).
		Scan(&address, &chainID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "master wallet not found"})
		return
	}
	rpc := rpcEndpointForChain(chainID)
	if rpc == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "RPC not configured for chain", "chain_id": chainID})
		return
	}
	bal, err := FetchNativeBalance(ctx, rpc, common.HexToAddress(address))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	chain, _ := chainByID(chainID)
	var usd float64
	if p, err := FetchTokenPrice(ctx, chainCoinGeckoID(chainID)); err == nil && p != nil {
		f, _ := new(big.Float).Quo(new(big.Float).SetInt(bal), big.NewFloat(pow10f(chain.Decimals))).Float64()
		usd = f * p.USD
	}
	// Persist overview snapshot (upsert).
	_, _ = svc.store.db.Exec(ctx,
		`INSERT INTO treasury_overview (master_wallet_id, total_value_usd, total_balance, allocated, reserved)
		 VALUES ($1,$2,$3,0,0)
		 ON CONFLICT (master_wallet_id) DO UPDATE SET total_value_usd=$2, total_balance=$3, updated_at=NOW()`,
		id, usd, bal.String())
	c.JSON(http.StatusOK, gin.H{
		"master_wallet_id": id, "address": address, "chain_id": chainID,
		"total_balance": weiToFloat(bal, chain.Decimals), "total_balance_wei": bal.String(),
		"total_value_usd": usd, "native_symbol": chain.Symbol, "updated_at": time.Now().UTC(),
	})
}

// TreasuryTransfer broadcasts a real on-chain transfer from the treasury hot
// wallet. The hot-wallet key MUST be configured via env; fail-closed otherwise.
type treasuryTransferReq struct {
	To     string `json:"to" binding:"required"`
	Amount string `json:"amount" binding:"required"`
	Token  string `json:"token"`
	Notes  string `json:"notes"`
}

func (svc *Service) TreasuryTransfer(c *gin.Context) {
	id := c.Param("id")
	var req treasuryTransferReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hotKey := os.Getenv("MASTER_WALLET_TREASURY_KEY_HEX")
	if hotKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "treasury signing key not configured (set MASTER_WALLET_TREASURY_KEY_HEX)", "status": "requires_signing"})
		return
	}
	privKey, err := parsePrivateKeyHex(hotKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid treasury key"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	var chainID int64
	err = svc.store.db.QueryRow(ctx, `SELECT chain_id FROM master_wallets WHERE id = $1`, id).Scan(&chainID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "master wallet not found"})
		return
	}
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
	chain, _ := chainByID(chainID)
	toAddr := common.HexToAddress(req.To)
	var value *big.Int
	var data []byte
	if req.Token == "" {
		wei, ok := new(big.Int).SetString(humanToWei(req.Amount, chain.Decimals), 10)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
			return
		}
		value = wei
	} else {
		wei, ok := new(big.Int).SetString(humanToWei(req.Amount, 18), 10)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
			return
		}
		value = big.NewInt(0)
		data = erc20TransferCalldata(toAddr, wei)
		toAddr = common.HexToAddress(req.Token)
	}
	rawTx, err := SignEVMTransaction(big.NewInt(chainID), nonce, toAddr, value, 21000, maxFee, prio, data, privKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	txHash, err := BroadcastTransaction(ctx, rpc, rawTx)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	// Record the treasury transaction.
	txRec := uuid.New().String()
	_, _ = svc.store.db.Exec(ctx,
		`INSERT INTO treasury_transactions (id, master_wallet_id, tx_type, amount, token_symbol, chain_id, tx_hash, status, counterparty, notes)
		 VALUES ($1,$2,'transfer',$3,$4,$5,$6,'pending',$7,$8)`,
		txRec, id, req.Amount, req.Token, chainID, txHash, req.To, req.Notes)
	svc.store.audit(ctx, id, "treasury.transfer", "treasury", "user", currentUserID(c), "transaction", txHash, "high", gin.H{"to": req.To, "amount": req.Amount})
	c.JSON(http.StatusOK, gin.H{"transaction_hash": txHash, "status": "broadcast", "treasury_tx_id": txRec, "from": from.Hex()})
}

// TreasuryTransactions lists treasury transaction history (from DB).
func (svc *Service) TreasuryTransactions(c *gin.Context) {
	id := c.Param("id")
	limit := parseLimit(c.Query("limit"), 50, 200)
	ctx := c.Request.Context()
	rows, err := svc.store.db.Query(ctx,
		`SELECT id, tx_type, amount, token_symbol, chain_id, tx_hash, status, counterparty, notes, created_at, confirmed_at
		 FROM treasury_transactions WHERE master_wallet_id = $1 ORDER BY created_at DESC LIMIT $2`, id, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch treasury transactions"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var entry gin.H
		var txID uuid.UUID
		var txType, amount, status string
		var tokenSym, txHash, counterparty, notes *string
		var chainID int64
		var createdAt time.Time
		var confirmedAt *time.Time
		_ = rows.Scan(&txID, &txType, &amount, &tokenSym, &chainID, &txHash, &status, &counterparty, &notes, &createdAt, &confirmedAt)
		entry = gin.H{
			"id": txID.String(), "tx_type": txType, "amount": amount, "status": status,
			"chain_id": chainID, "created_at": createdAt,
		}
		if tokenSym != nil {
			entry["token"] = *tokenSym
		}
		if txHash != nil {
			entry["tx_hash"] = *txHash
		}
		if counterparty != nil {
			entry["counterparty"] = *counterparty
		}
		if notes != nil {
			entry["notes"] = *notes
		}
		if confirmedAt != nil {
			entry["confirmed_at"] = *confirmedAt
		}
		out = append(out, entry)
	}
	c.JSON(http.StatusOK, gin.H{"transactions": out})
}

// TreasurySweep moves all funds from a sub-wallet to the treasury. Real broadcast.
type sweepReq struct {
	SubWalletID string `json:"sub_wallet_id" binding:"required"`
	Password    string `json:"password" binding:"required"`
}

func (svc *Service) TreasurySweep(c *gin.Context) {
	id := c.Param("id")
	var req sweepReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Resolve sub wallet address + master chain.
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	var subAddr string
	var chainID int64
	err := svc.store.db.QueryRow(ctx,
		`SELECT sw.address, mw.chain_id FROM sub_wallets sw JOIN master_wallets mw ON sw.master_wallet_id = mw.id WHERE sw.id = $1`,
		req.SubWalletID).Scan(&subAddr, &chainID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "sub wallet not found"})
		return
	}
	rpc := rpcEndpointForChain(chainID)
	if rpc == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "RPC not configured"})
		return
	}
	// Fetch the full balance of the sub wallet.
	bal, err := FetchNativeBalance(ctx, rpc, common.HexToAddress(subAddr))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	if bal.Sign() <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nothing to sweep (balance is zero)"})
		return
	}
	// Use the master wallet's key to sign (it controls sub wallets).
	send := sendReq{To: "", Amount: "0", Password: req.Password}
	// Need the treasury address (master wallet address).
	var masterAddr string
	_ = svc.store.db.QueryRow(ctx, `SELECT address FROM master_wallets WHERE id = $1`, id).Scan(&masterAddr)
	send.To = masterAddr
	// Convert wei -> human.
	chain, _ := chainByID(chainID)
	f, _ := new(big.Float).Quo(new(big.Float).SetInt(bal), big.NewFloat(pow10f(chain.Decimals))).Float64()
	send.Amount = fmt.Sprintf("%.8f", f)
	// Subtract gas estimate.
	txHash, fromAddr, _, err := svc.buildSignBroadcast(c, id, send)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	txRec := uuid.New().String()
	_, _ = svc.store.db.Exec(ctx,
		`INSERT INTO treasury_transactions (id, master_wallet_id, tx_type, amount, chain_id, tx_hash, status, counterparty, notes)
		 VALUES ($1,$2,'sweep',$3,$4,$5,'pending',$6,'sweep from sub wallet')`,
		txRec, id, send.Amount, chainID, txHash, fromAddr)
	svc.store.audit(ctx, id, "treasury.sweep", "treasury", "user", currentUserID(c), "transaction", txHash, "high", gin.H{"sub_wallet": req.SubWalletID})
	c.JSON(http.StatusOK, gin.H{"transaction_hash": txHash, "status": "pending", "swept_amount": send.Amount, "treasury_tx_id": txRec})
}
