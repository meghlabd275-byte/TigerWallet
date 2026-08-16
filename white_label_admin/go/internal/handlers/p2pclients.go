package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tigerwallet/white-label-admin/internal/middleware"
)

// ==================== P2P clients (p2p_admin scope) ====================
// Governance records only — no fund movement.

func (s *Svc) ListP2PClients(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	ctx := c.Request.Context()
	rows, err := s.db.Query(ctx,
		`SELECT id, COALESCE(user_id::text,''), username, status, rating, trades_count, created_at, updated_at
		 FROM p2p_clients WHERE white_label_id=$1 ORDER BY created_at DESC LIMIT 100`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var userID, username, status string
		var rating string
		var tradesCount int32
		var created, updated time.Time
		_ = rows.Scan(&id, &userID, &username, &status, &rating, &tradesCount, &created, &updated)
		out = append(out, gin.H{"id": id, "user_id": userID, "username": username, "status": status,
			"rating": rating, "trades_count": tradesCount, "created_at": created, "updated_at": updated})
	}
	c.JSON(http.StatusOK, gin.H{"clients": out})
}

func (s *Svc) GetP2PClient(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	var userID, username, status string
	var rating string
	var tradesCount int32
	var created, updated time.Time
	err = s.db.QueryRow(ctx,
		`SELECT COALESCE(user_id::text,''), username, status, rating, trades_count, created_at, updated_at
		 FROM p2p_clients WHERE id=$1 AND white_label_id=$2`, id, tenantID).
		Scan(&userID, &username, &status, &rating, &tradesCount, &created, &updated)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "user_id": userID, "username": username, "status": status,
		"rating": rating, "trades_count": tradesCount, "created_at": created, "updated_at": updated})
}

func (s *Svc) CreateP2PClient(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	var req struct {
		UserID   string `json:"user_id"`
		Username string `json:"username" binding:"required"`
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
	id := uuid.New()
	ctx := c.Request.Context()
	_, err := s.db.Exec(ctx,
		`INSERT INTO p2p_clients (id, user_id, username, status, white_label_id)
		 VALUES ($1,$2,$3,'active',$4)`,
		id, uid, req.Username, tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "p2p_client.create", "p2p_client", id.String(), gin.H{"username": req.Username})
	c.JSON(http.StatusCreated, gin.H{"id": id, "username": req.Username, "status": "active"})
}

func (s *Svc) UpdateP2PClient(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Username string `json:"username"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	if req.Username != "" {
		_, err = s.db.Exec(ctx, `UPDATE p2p_clients SET username=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.Username, id, tenantID)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "p2p_client.update", "p2p_client", id.String(), nil)
	c.JSON(http.StatusOK, gin.H{"updated": id})
}

func (s *Svc) DeleteP2PClient(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	ct, err := s.db.Exec(ctx, `DELETE FROM p2p_clients WHERE id=$1 AND white_label_id=$2`, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "p2p_client.delete", "p2p_client", id.String(), nil)
	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

func (s *Svc) UpdateP2PClientStatus(c *gin.Context) {
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
	ct, err := s.db.Exec(ctx, `UPDATE p2p_clients SET status=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.Status, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "p2p_client.status", "p2p_client", id.String(), gin.H{"status": req.Status})
	c.JSON(http.StatusOK, gin.H{"updated": id, "status": req.Status})
}
