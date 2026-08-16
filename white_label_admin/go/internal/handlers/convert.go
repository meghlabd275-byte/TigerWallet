package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tigerwallet/white-label-admin/internal/middleware"
)

// ==================== Convert orders (trading_admin scope) ====================
// Governance records only — no fund movement.

func (s *Svc) ListConvertOrders(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	ctx := c.Request.Context()
	rows, err := s.db.Query(ctx,
		`SELECT id, COALESCE(user_id::text,''), from_token, to_token, from_amount, to_amount, rate,
		        status, COALESCE(chain_id,0), created_at, updated_at
		 FROM convert_orders WHERE white_label_id=$1 ORDER BY created_at DESC LIMIT 100`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var userID, fromToken, toToken, status string
		var fromAmount, toAmount, rate string
		var chainID int64
		var created, updated time.Time
		_ = rows.Scan(&id, &userID, &fromToken, &toToken, &fromAmount, &toAmount, &rate, &status, &chainID, &created, &updated)
		out = append(out, gin.H{"id": id, "user_id": userID, "from_token": fromToken, "to_token": toToken,
			"from_amount": fromAmount, "to_amount": toAmount, "rate": rate, "status": status, "chain_id": chainID,
			"created_at": created, "updated_at": updated})
	}
	c.JSON(http.StatusOK, gin.H{"orders": out})
}

func (s *Svc) GetConvertOrder(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	var userID, fromToken, toToken, status string
	var fromAmount, toAmount, rate string
	var chainID int64
	var created, updated time.Time
	err = s.db.QueryRow(ctx,
		`SELECT COALESCE(user_id::text,''), from_token, to_token, from_amount, to_amount, rate,
		        status, COALESCE(chain_id,0), created_at, updated_at
		 FROM convert_orders WHERE id=$1 AND white_label_id=$2`, id, tenantID).
		Scan(&userID, &fromToken, &toToken, &fromAmount, &toAmount, &rate, &status, &chainID, &created, &updated)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "user_id": userID, "from_token": fromToken, "to_token": toToken,
		"from_amount": fromAmount, "to_amount": toAmount, "rate": rate, "status": status, "chain_id": chainID,
		"created_at": created, "updated_at": updated})
}

func (s *Svc) CreateConvertOrder(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	var req struct {
		UserID     string  `json:"user_id"`
		FromToken  string  `json:"from_token" binding:"required"`
		ToToken    string  `json:"to_token" binding:"required"`
		FromAmount float64 `json:"from_amount"`
		ToAmount   float64 `json:"to_amount"`
		Rate       float64 `json:"rate"`
		ChainID    int64   `json:"chain_id"`
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
		`INSERT INTO convert_orders (id, user_id, from_token, to_token, from_amount, to_amount, rate, chain_id, status, white_label_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'pending',$9)`,
		id, uid, req.FromToken, req.ToToken, req.FromAmount, req.ToAmount, req.Rate, req.ChainID, tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "convert.create", "convert_order", id.String(), gin.H{"from_token": req.FromToken})
	c.JSON(http.StatusCreated, gin.H{"id": id, "status": "pending"})
}

func (s *Svc) UpdateConvertOrder(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		FromToken  string  `json:"from_token"`
		ToToken    string  `json:"to_token"`
		FromAmount float64 `json:"from_amount"`
		ToAmount   float64 `json:"to_amount"`
		Rate       float64 `json:"rate"`
		ChainID    int64   `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	if req.FromToken != "" {
		_, _ = s.db.Exec(ctx, `UPDATE convert_orders SET from_token=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.FromToken, id, tenantID)
	}
	if req.ToToken != "" {
		_, _ = s.db.Exec(ctx, `UPDATE convert_orders SET to_token=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.ToToken, id, tenantID)
	}
	_, err = s.db.Exec(ctx,
		`UPDATE convert_orders SET from_amount=$1, to_amount=$2, rate=$3, chain_id=$4, updated_at=NOW()
		 WHERE id=$5 AND white_label_id=$6`,
		req.FromAmount, req.ToAmount, req.Rate, req.ChainID, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "convert.update", "convert_order", id.String(), nil)
	c.JSON(http.StatusOK, gin.H{"updated": id})
}

func (s *Svc) DeleteConvertOrder(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	ct, err := s.db.Exec(ctx, `DELETE FROM convert_orders WHERE id=$1 AND white_label_id=$2`, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "convert.delete", "convert_order", id.String(), nil)
	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

func (s *Svc) UpdateConvertOrderStatus(c *gin.Context) {
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
	ct, err := s.db.Exec(ctx, `UPDATE convert_orders SET status=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.Status, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "convert.status", "convert_order", id.String(), gin.H{"status": req.Status})
	c.JSON(http.StatusOK, gin.H{"updated": id, "status": req.Status})
}
