package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tigerwallet/white-label-admin/internal/middleware"
)

// ==================== Futures positions (trading_admin scope) ====================
// Governance records only — no fund movement. The WL admin manages the
// position record metadata; actual leverage/margin is enforced by the
// trading engine, not here.

func (s *Svc) ListFuturesPositions(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	ctx := c.Request.Context()
	rows, err := s.db.Query(ctx,
		`SELECT id, COALESCE(user_id::text,''), pair, side, size, leverage, entry_price, liquidation_price,
		        margin, status, COALESCE(chain_id,0), created_at, updated_at
		 FROM futures_positions WHERE white_label_id=$1 ORDER BY created_at DESC LIMIT 100`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var userID, pair, side, status string
		var size, leverage, entry, liq, margin string
		var chainID int64
		var created, updated time.Time
		_ = rows.Scan(&id, &userID, &pair, &side, &size, &leverage, &entry, &liq, &margin, &status, &chainID, &created, &updated)
		out = append(out, gin.H{"id": id, "user_id": userID, "pair": pair, "side": side, "size": size, "leverage": leverage,
			"entry_price": entry, "liquidation_price": liq, "margin": margin, "status": status, "chain_id": chainID,
			"created_at": created, "updated_at": updated})
	}
	c.JSON(http.StatusOK, gin.H{"positions": out})
}

func (s *Svc) GetFuturesPosition(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	var userID, pair, side, status string
	var size, leverage, entry, liq, margin string
	var chainID int64
	var created, updated time.Time
	err = s.db.QueryRow(ctx,
		`SELECT COALESCE(user_id::text,''), pair, side, size, leverage, entry_price, liquidation_price,
		        margin, status, COALESCE(chain_id,0), created_at, updated_at
		 FROM futures_positions WHERE id=$1 AND white_label_id=$2`, id, tenantID).
		Scan(&userID, &pair, &side, &size, &leverage, &entry, &liq, &margin, &status, &chainID, &created, &updated)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "position not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "user_id": userID, "pair": pair, "side": side, "size": size, "leverage": leverage,
		"entry_price": entry, "liquidation_price": liq, "margin": margin, "status": status, "chain_id": chainID,
		"created_at": created, "updated_at": updated})
}

func (s *Svc) CreateFuturesPosition(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	var req struct {
		UserID           string  `json:"user_id"`
		Pair             string  `json:"pair" binding:"required"`
		Side             string  `json:"side" binding:"required"`
		Size             float64 `json:"size"`
		Leverage         float64 `json:"leverage"`
		EntryPrice       float64 `json:"entry_price"`
		LiquidationPrice float64 `json:"liquidation_price"`
		Margin           float64 `json:"margin"`
		ChainID          int64   `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var uid *uuid.UUID
	if req.UserID != "" {
		if parsed, e := uuid.Parse(req.UserID); e == nil {
			uid = &parsed
		}
	}
	if req.Leverage == 0 {
		req.Leverage = 1
	}
	id := uuid.New()
	ctx := c.Request.Context()
	_, err := s.db.Exec(ctx,
		`INSERT INTO futures_positions (id, user_id, pair, side, size, leverage, entry_price, liquidation_price, margin, chain_id, status, white_label_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'open',$11)`,
		id, uid, req.Pair, req.Side, req.Size, req.Leverage, req.EntryPrice, req.LiquidationPrice, req.Margin, req.ChainID, tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "futures.create", "futures_position", id.String(), gin.H{"pair": req.Pair})
	c.JSON(http.StatusCreated, gin.H{"id": id, "pair": req.Pair, "status": "open"})
}

func (s *Svc) UpdateFuturesPosition(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Pair             string  `json:"pair"`
		Side             string  `json:"side"`
		Size             float64 `json:"size"`
		Leverage         float64 `json:"leverage"`
		EntryPrice       float64 `json:"entry_price"`
		LiquidationPrice float64 `json:"liquidation_price"`
		Margin           float64 `json:"margin"`
		ChainID          int64   `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	if req.Pair != "" {
		_, _ = s.db.Exec(ctx, `UPDATE futures_positions SET pair=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.Pair, id, tenantID)
	}
	if req.Side != "" {
		_, _ = s.db.Exec(ctx, `UPDATE futures_positions SET side=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.Side, id, tenantID)
	}
	_, err = s.db.Exec(ctx,
		`UPDATE futures_positions SET size=$1, leverage=$2, entry_price=$3, liquidation_price=$4, margin=$5, chain_id=$6, updated_at=NOW()
		 WHERE id=$7 AND white_label_id=$8`,
		req.Size, req.Leverage, req.EntryPrice, req.LiquidationPrice, req.Margin, req.ChainID, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "futures.update", "futures_position", id.String(), nil)
	c.JSON(http.StatusOK, gin.H{"updated": id})
}

func (s *Svc) DeleteFuturesPosition(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	ct, err := s.db.Exec(ctx, `DELETE FROM futures_positions WHERE id=$1 AND white_label_id=$2`, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "position not found"})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "futures.delete", "futures_position", id.String(), nil)
	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

func (s *Svc) UpdateFuturesPositionStatus(c *gin.Context) {
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
	ctx := c.Request.Context()
	ct, err := s.db.Exec(ctx, `UPDATE futures_positions SET status=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.Status, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "position not found"})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "futures.status", "futures_position", id.String(), gin.H{"status": req.Status})
	c.JSON(http.StatusOK, gin.H{"updated": id, "status": req.Status})
}
