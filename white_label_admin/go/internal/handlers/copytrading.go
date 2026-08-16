package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tigerwallet/white-label-admin/internal/middleware"
)

// ==================== Copy-trading configs (trading_admin scope) ====================
// Governance records only — no fund movement.

func (s *Svc) ListCopyTradingConfigs(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	ctx := c.Request.Context()
	rows, err := s.db.Query(ctx,
		`SELECT id, COALESCE(follower_id::text,''), COALESCE(leader_id::text,''), allocation, max_leverage,
		        status, created_at, updated_at
		 FROM copy_trading_configs WHERE white_label_id=$1 ORDER BY created_at DESC LIMIT 100`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var followerID, leaderID, status string
		var allocation, maxLeverage string
		var created, updated time.Time
		_ = rows.Scan(&id, &followerID, &leaderID, &allocation, &maxLeverage, &status, &created, &updated)
		out = append(out, gin.H{"id": id, "follower_id": followerID, "leader_id": leaderID,
			"allocation": allocation, "max_leverage": maxLeverage, "status": status, "created_at": created, "updated_at": updated})
	}
	c.JSON(http.StatusOK, gin.H{"configs": out})
}

func (s *Svc) GetCopyTradingConfig(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	var followerID, leaderID, status string
	var allocation, maxLeverage string
	var created, updated time.Time
	err = s.db.QueryRow(ctx,
		`SELECT COALESCE(follower_id::text,''), COALESCE(leader_id::text,''), allocation, max_leverage,
		        status, created_at, updated_at
		 FROM copy_trading_configs WHERE id=$1 AND white_label_id=$2`, id, tenantID).
		Scan(&followerID, &leaderID, &allocation, &maxLeverage, &status, &created, &updated)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "follower_id": followerID, "leader_id": leaderID,
		"allocation": allocation, "max_leverage": maxLeverage, "status": status, "created_at": created, "updated_at": updated})
}

func (s *Svc) CreateCopyTradingConfig(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	var req struct {
		FollowerID  string  `json:"follower_id"`
		LeaderID    string  `json:"leader_id"`
		Allocation  float64 `json:"allocation"`
		MaxLeverage float64 `json:"max_leverage"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var follower, leader *uuid.UUID
	if req.FollowerID != "" {
		if parsed, e := uuid.Parse(req.FollowerID); e == nil {
			follower = &parsed
		}
	}
	if req.LeaderID != "" {
		if parsed, e := uuid.Parse(req.LeaderID); e == nil {
			leader = &parsed
		}
	}
	if req.MaxLeverage == 0 {
		req.MaxLeverage = 1
	}
	id := uuid.New()
	ctx := c.Request.Context()
	_, err := s.db.Exec(ctx,
		`INSERT INTO copy_trading_configs (id, follower_id, leader_id, allocation, max_leverage, status, white_label_id)
		 VALUES ($1,$2,$3,$4,$5,'active',$6)`,
		id, follower, leader, req.Allocation, req.MaxLeverage, tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "copy_trading.create", "copy_trading_config", id.String(), nil)
	c.JSON(http.StatusCreated, gin.H{"id": id, "status": "active"})
}

func (s *Svc) UpdateCopyTradingConfig(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Allocation  float64 `json:"allocation"`
		MaxLeverage float64 `json:"max_leverage"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	_, err = s.db.Exec(ctx,
		`UPDATE copy_trading_configs SET allocation=$1, max_leverage=$2, updated_at=NOW()
		 WHERE id=$3 AND white_label_id=$4`,
		req.Allocation, req.MaxLeverage, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "copy_trading.update", "copy_trading_config", id.String(), nil)
	c.JSON(http.StatusOK, gin.H{"updated": id})
}

func (s *Svc) DeleteCopyTradingConfig(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	ct, err := s.db.Exec(ctx, `DELETE FROM copy_trading_configs WHERE id=$1 AND white_label_id=$2`, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "copy_trading.delete", "copy_trading_config", id.String(), nil)
	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

func (s *Svc) UpdateCopyTradingConfigStatus(c *gin.Context) {
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
	ct, err := s.db.Exec(ctx, `UPDATE copy_trading_configs SET status=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.Status, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "copy_trading.status", "copy_trading_config", id.String(), gin.H{"status": req.Status})
	c.JSON(http.StatusOK, gin.H{"updated": id, "status": req.Status})
}
