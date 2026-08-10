package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
