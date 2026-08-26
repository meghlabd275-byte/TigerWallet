package handlers

import (
        "net/http"
        "time"

        "github.com/gin-gonic/gin"
        "github.com/google/uuid"
        "github.com/tigerwallet/white-label-admin/internal/middleware"
)

// ==================== WL MasterWallet + UserWallet governance (wallet_admin) ====================
//
// The WL MasterWallet and WL UserWallet are INDEPENDENT, license-gated
// processes (wl_master_wallet/go, wl_user_wallet/go). These handlers let a WL
// client govern them from the WL-admin panel. They persist GOVERNANCE RECORDS
// (chain/token registration, fee schedules, account status) in the WL-admin's
// own PostgreSQL, tenant-scoped via middleware.TenantID. Real signing/balance
// state lives in the product backends; these rows are the audit/governance
// layer, and fund-moving operations (withdrawals) remain SuperAdmin co-signed.

// ---------------- master_wallet ----------------

func (s *Svc) ListWLMasterWallets(c *gin.Context) {
        tenantID := middleware.TenantID(c)
        ctx := c.Request.Context()
        rows, err := s.db.Query(ctx,
                `SELECT id, name, chain_id, chain_type, symbol, status, auto_sign_enabled, fee_percent, created_at, updated_at
                 FROM wl_master_wallets WHERE white_label_id=$1 ORDER BY chain_id ASC`, tenantID)
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
        }
        defer rows.Close()
        out := []gin.H{}
        for rows.Next() {
                var id uuid.UUID
                var name, chainType, symbol, status string
                var chainID int64
                var autoSign bool
                var feePercent float64
                var created, updated time.Time
                _ = rows.Scan(&id, &name, &chainID, &chainType, &symbol, &status, &autoSign, &feePercent, &created, &updated)
                out = append(out, gin.H{
                        "id": id, "name": name, "chain_id": chainID, "chain_type": chainType,
                        "symbol": symbol, "status": status, "auto_sign_enabled": autoSign,
                        "fee_percent": feePercent, "created_at": created, "updated_at": updated,
                })
        }
        c.JSON(http.StatusOK, gin.H{"master_wallets": out})
}

func (s *Svc) RegisterWLMasterWallet(c *gin.Context) {
        tenantID := middleware.TenantID(c)
        var req struct {
                Name            string  `json:"name" binding:"required"`
                ChainID         int64   `json:"chain_id" binding:"required"`
                ChainType       string  `json:"chain_type"`
                Symbol          string  `json:"symbol"`
                AutoSignEnabled *bool   `json:"auto_sign_enabled"`
                FeePercent      float64 `json:"fee_percent"`
        }
        if err := c.ShouldBindJSON(&req); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }
        if req.ChainType == "" {
                req.ChainType = "evm"
        }
        autoSign := true
        if req.AutoSignEnabled != nil {
                autoSign = *req.AutoSignEnabled
        }
        id := uuid.New()
        ctx := c.Request.Context()
        if _, err := s.db.Exec(ctx,
                `INSERT INTO wl_master_wallets (id, name, chain_id, chain_type, symbol, status, auto_sign_enabled, fee_percent, white_label_id)
                 VALUES ($1,$2,$3,$4,$5,'active',$6,$7,$8)`,
                id, req.Name, req.ChainID, req.ChainType, req.Symbol, autoSign, req.FeePercent, tenantID); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }
        s.audit(ctx, middleware.AdminID(c), "wl_master_wallet.register", "wl_master_wallet", id.String(), gin.H{"name": req.Name, "chain_id": req.ChainID})
        c.JSON(http.StatusCreated, gin.H{"id": id, "name": req.Name, "chain_id": req.ChainID, "status": "active"})
}

func (s *Svc) UpdateWLMasterWallet(c *gin.Context) {
        tenantID := middleware.TenantID(c)
        id, err := uuid.Parse(c.Param("id"))
        if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
                return
        }
        var req struct {
                Name            string  `json:"name"`
                Symbol          string  `json:"symbol"`
                Status          string  `json:"status"`
                AutoSignEnabled *bool   `json:"auto_sign_enabled"`
                FeePercent      *float64 `json:"fee_percent"`
        }
        if err := c.ShouldBindJSON(&req); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }
        ctx := c.Request.Context()
        // Halt is recorded here; resume requires TigerWallet SuperAdmin
        // collaboration (fail-closed), matching the rest of the WL governance.
        if req.Status == "active" {
                c.JSON(http.StatusForbidden, gin.H{"error": "SuperAdmin collaboration required to resume"})
                return
        }
        if req.Name != "" {
                _, _ = s.db.Exec(ctx, `UPDATE wl_master_wallets SET name=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.Name, id, tenantID)
        }
        if req.Symbol != "" {
                _, _ = s.db.Exec(ctx, `UPDATE wl_master_wallets SET symbol=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.Symbol, id, tenantID)
        }
        if req.Status != "" && req.Status != "active" {
                _, _ = s.db.Exec(ctx, `UPDATE wl_master_wallets SET status=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.Status, id, tenantID)
        }
        if req.AutoSignEnabled != nil {
                _, _ = s.db.Exec(ctx, `UPDATE wl_master_wallets SET auto_sign_enabled=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, *req.AutoSignEnabled, id, tenantID)
        }
        if req.FeePercent != nil {
                _, _ = s.db.Exec(ctx, `UPDATE wl_master_wallets SET fee_percent=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, *req.FeePercent, id, tenantID)
        }
        s.audit(ctx, middleware.AdminID(c), "wl_master_wallet.update", "wl_master_wallet", id.String(), nil)
        c.JSON(http.StatusOK, gin.H{"updated": id})
}

func (s *Svc) DeleteWLMasterWallet(c *gin.Context) {
        tenantID := middleware.TenantID(c)
        id, err := uuid.Parse(c.Param("id"))
        if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
                return
        }
        ctx := c.Request.Context()
        ct, err := s.db.Exec(ctx, `DELETE FROM wl_master_wallets WHERE id=$1 AND white_label_id=$2`, id, tenantID)
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
        }
        if ct.RowsAffected() == 0 {
                c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
                return
        }
        s.audit(ctx, middleware.AdminID(c), "wl_master_wallet.delete", "wl_master_wallet", id.String(), nil)
        c.JSON(http.StatusOK, gin.H{"deleted": id})
}

// ---------------- user_wallet ----------------

func (s *Svc) ListWLUserWallets(c *gin.Context) {
        tenantID := middleware.TenantID(c)
        ctx := c.Request.Context()
        rows, err := s.db.Query(ctx,
                `SELECT id, user_ref, wallet_address, chain_id, status, created_at, updated_at
                 FROM wl_user_wallets WHERE white_label_id=$1 ORDER BY created_at DESC LIMIT 500`, tenantID)
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
        }
        defer rows.Close()
        out := []gin.H{}
        for rows.Next() {
                var id uuid.UUID
                var userRef, addr, status string
                var chainID int64
                var created, updated time.Time
                _ = rows.Scan(&id, &userRef, &addr, &chainID, &status, &created, &updated)
                out = append(out, gin.H{
                        "id": id, "user_ref": userRef, "wallet_address": addr,
                        "chain_id": chainID, "status": status, "created_at": created, "updated_at": updated,
                })
        }
        c.JSON(http.StatusOK, gin.H{"user_wallets": out})
}

func (s *Svc) RegisterWLUserWallet(c *gin.Context) {
        tenantID := middleware.TenantID(c)
        var req struct {
                UserRef       string `json:"user_ref" binding:"required"`
                WalletAddress string `json:"wallet_address" binding:"required"`
                ChainID       int64  `json:"chain_id"`
        }
        if err := c.ShouldBindJSON(&req); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }
        id := uuid.New()
        ctx := c.Request.Context()
        if _, err := s.db.Exec(ctx,
                `INSERT INTO wl_user_wallets (id, user_ref, wallet_address, chain_id, status, white_label_id)
                 VALUES ($1,$2,$3,$4,'active',$5)`,
                id, req.UserRef, req.WalletAddress, req.ChainID, tenantID); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }
        s.audit(ctx, middleware.AdminID(c), "wl_user_wallet.register", "wl_user_wallet", id.String(), gin.H{"user_ref": req.UserRef})
        c.JSON(http.StatusCreated, gin.H{"id": id, "user_ref": req.UserRef, "status": "active"})
}

func (s *Svc) UpdateWLUserWalletStatus(c *gin.Context) {
        tenantID := middleware.TenantID(c)
        id, err := uuid.Parse(c.Param("id"))
        if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
                return
        }
        var req struct {
                Status string `json:"status" binding:"required"`
        }
        if err := c.ShouldBindJSON(&req); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }
        if req.Status == "active" {
                c.JSON(http.StatusForbidden, gin.H{"error": "SuperAdmin collaboration required to reactivate"})
                return
        }
        if req.Status != "suspended" && req.Status != "frozen" && req.Status != "closed" {
                c.JSON(http.StatusBadRequest, gin.H{"error": "status must be suspended, frozen, or closed"})
                return
        }
        ctx := c.Request.Context()
        ct, err := s.db.Exec(ctx,
                `UPDATE wl_user_wallets SET status=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.Status, id, tenantID)
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
        }
        if ct.RowsAffected() == 0 {
                c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
                return
        }
        s.audit(ctx, middleware.AdminID(c), "wl_user_wallet."+req.Status, "wl_user_wallet", id.String(), nil)
        c.JSON(http.StatusOK, gin.H{"id": id, "status": req.Status})
}