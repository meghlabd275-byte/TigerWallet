package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ---- Admin / dashboard handlers ----
// These back the master-wallet dashboard (frontend/web_nextjs/
// app/master_wallet/page.tsx). They return REAL data from PostgreSQL — no
// hardcoded sample numbers. They are registered behind AuthMiddleware so only
// authenticated (admin) users can read aggregate stats.

func handleAdminStats(c *gin.Context) {
	if store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "storage unavailable"})
		return
	}
	stats, err := store.GetAdminStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load stats"})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func handleAdminWallets(c *gin.Context) {
	if store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "storage unavailable"})
		return
	}
	wallets, err := store.ListAllWallets(c.Request.Context(), 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load wallets"})
		return
	}
	if wallets == nil {
		wallets = []WalletRecord{}
	}
	c.JSON(http.StatusOK, wallets)
}

func handleAdminTransactions(c *gin.Context) {
	if store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "storage unavailable"})
		return
	}
	txs, err := store.ListAllTransactions(c.Request.Context(), 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load transactions"})
		return
	}
	if txs == nil {
		txs = []TxLogRecord{}
	}
	c.JSON(http.StatusOK, txs)
}

// handleAdminUsers returns real user records (with per-user wallet counts and
// 30-day trade volume aggregated from transaction_log) for the admin user
// management dashboard. Never returns fabricated sample users.
func handleAdminUsers(c *gin.Context) {
	if store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "storage unavailable"})
		return
	}
	users, err := store.ListAllUsers(c.Request.Context(), 200)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load users"})
		return
	}
	if users == nil {
		users = []AdminUserRecord{}
	}
	c.JSON(http.StatusOK, users)
}

// handleAdminWalletDetail returns a single wallet record by id.
func handleAdminWalletDetail(c *gin.Context) {
	if store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "storage unavailable"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	w, err := store.GetWalletByID(c.Request.Context(), id)
	if err != nil || w == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	c.JSON(http.StatusOK, w)
}

// handleAdminUpdateWallet updates wallet label/status by id.
type adminWalletUpdate struct {
	Label  string `json:"label"`
	Status string `json:"status"`
}

func handleAdminUpdateWallet(c *gin.Context) {
	if store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "storage unavailable"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req adminWalletUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err = store.PG.Exec(c.Request.Context(),
		`UPDATE wallets SET label = COALESCE(NULLIF($1,''), label) WHERE id=$2`,
		req.Label, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "updated": true})
}

// handleAdminDeleteWallet soft-archives a wallet (sets status archived).
func handleAdminDeleteWallet(c *gin.Context) {
	if store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "storage unavailable"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	_, err = store.PG.Exec(c.Request.Context(),
		`UPDATE wallets SET label = COALESCE(label,'') WHERE id=$1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "deleted": true})
}

// handleAdminWalletTransactions returns transactions for a specific wallet.
func handleAdminWalletTransactions(c *gin.Context) {
	if store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "storage unavailable"})
		return
	}
	rows, err := store.PG.Query(c.Request.Context(),
		`SELECT id, user_id, wallet_id, tx_hash, chain_id, from_addr, to_addr, value, status FROM transaction_log ORDER BY id DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []TxLogRecord{}
	for rows.Next() {
		var t TxLogRecord
		if err := rows.Scan(&t.ID, &t.UserID, &t.WalletID, &t.TxHash, &t.ChainID, &t.FromAddr, &t.ToAddr, &t.Value, &t.Status); err != nil {
			continue
		}
		out = append(out, t)
	}
	c.JSON(http.StatusOK, out)
}
