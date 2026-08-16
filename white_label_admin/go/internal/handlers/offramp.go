package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tigerwallet/white-label-admin/internal/middleware"
)

// ==================== Offramp orders (p2p_admin scope) ====================
// Governance records only — no fund movement.

func (s *Svc) ListOfframpOrders(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	ctx := c.Request.Context()
	rows, err := s.db.Query(ctx,
		`SELECT id, COALESCE(user_id::text,''), provider, crypto_token, fiat_currency, crypto_amount, fiat_amount,
		        status, COALESCE(payout_ref,''), created_at, updated_at
		 FROM offramp_orders WHERE white_label_id=$1 ORDER BY created_at DESC LIMIT 100`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var userID, provider, cryptoToken, fiatCurrency, status, payoutRef string
		var cryptoAmount, fiatAmount string
		var created, updated time.Time
		_ = rows.Scan(&id, &userID, &provider, &cryptoToken, &fiatCurrency, &cryptoAmount, &fiatAmount, &status, &payoutRef, &created, &updated)
		out = append(out, gin.H{"id": id, "user_id": userID, "provider": provider, "crypto_token": cryptoToken,
			"fiat_currency": fiatCurrency, "crypto_amount": cryptoAmount, "fiat_amount": fiatAmount, "status": status,
			"payout_ref": payoutRef, "created_at": created, "updated_at": updated})
	}
	c.JSON(http.StatusOK, gin.H{"orders": out})
}

func (s *Svc) GetOfframpOrder(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	var userID, provider, cryptoToken, fiatCurrency, status, payoutRef string
	var cryptoAmount, fiatAmount string
	var created, updated time.Time
	err = s.db.QueryRow(ctx,
		`SELECT COALESCE(user_id::text,''), provider, crypto_token, fiat_currency, crypto_amount, fiat_amount,
		        status, COALESCE(payout_ref,''), created_at, updated_at
		 FROM offramp_orders WHERE id=$1 AND white_label_id=$2`, id, tenantID).
		Scan(&userID, &provider, &cryptoToken, &fiatCurrency, &cryptoAmount, &fiatAmount, &status, &payoutRef, &created, &updated)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "user_id": userID, "provider": provider, "crypto_token": cryptoToken,
		"fiat_currency": fiatCurrency, "crypto_amount": cryptoAmount, "fiat_amount": fiatAmount, "status": status,
		"payout_ref": payoutRef, "created_at": created, "updated_at": updated})
}

func (s *Svc) CreateOfframpOrder(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	var req struct {
		UserID       string  `json:"user_id"`
		Provider     string  `json:"provider" binding:"required"`
		CryptoToken  string  `json:"crypto_token" binding:"required"`
		FiatCurrency string  `json:"fiat_currency" binding:"required"`
		CryptoAmount float64 `json:"crypto_amount"`
		FiatAmount   float64 `json:"fiat_amount"`
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
		`INSERT INTO offramp_orders (id, user_id, provider, crypto_token, fiat_currency, crypto_amount, fiat_amount, status, white_label_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,'pending',$8)`,
		id, uid, req.Provider, req.CryptoToken, req.FiatCurrency, req.CryptoAmount, req.FiatAmount, tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "offramp.create", "offramp_order", id.String(), gin.H{"provider": req.Provider})
	c.JSON(http.StatusCreated, gin.H{"id": id, "status": "pending"})
}

func (s *Svc) UpdateOfframpOrder(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Provider     string  `json:"provider"`
		CryptoToken  string  `json:"crypto_token"`
		FiatCurrency string  `json:"fiat_currency"`
		CryptoAmount float64 `json:"crypto_amount"`
		FiatAmount   float64 `json:"fiat_amount"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	if req.Provider != "" {
		_, _ = s.db.Exec(ctx, `UPDATE offramp_orders SET provider=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.Provider, id, tenantID)
	}
	if req.CryptoToken != "" {
		_, _ = s.db.Exec(ctx, `UPDATE offramp_orders SET crypto_token=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.CryptoToken, id, tenantID)
	}
	if req.FiatCurrency != "" {
		_, _ = s.db.Exec(ctx, `UPDATE offramp_orders SET fiat_currency=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.FiatCurrency, id, tenantID)
	}
	_, err = s.db.Exec(ctx,
		`UPDATE offramp_orders SET crypto_amount=$1, fiat_amount=$2, updated_at=NOW()
		 WHERE id=$3 AND white_label_id=$4`,
		req.CryptoAmount, req.FiatAmount, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "offramp.update", "offramp_order", id.String(), nil)
	c.JSON(http.StatusOK, gin.H{"updated": id})
}

func (s *Svc) DeleteOfframpOrder(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	ct, err := s.db.Exec(ctx, `DELETE FROM offramp_orders WHERE id=$1 AND white_label_id=$2`, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "offramp.delete", "offramp_order", id.String(), nil)
	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

func (s *Svc) reviewOfframpOrder(c *gin.Context, status, action string) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	ctx := c.Request.Context()
	adminID := middleware.AdminID(c)
	if status == "completed" {
		tag, e := s.db.Exec(ctx, `UPDATE offramp_orders SET status='completed', updated_at=NOW() WHERE id=$1 AND white_label_id=$2`, id, tenantID)
		err = e
		if err == nil && tag.RowsAffected() == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
	} else {
		tag, e := s.db.Exec(ctx, `UPDATE offramp_orders SET status='rejected', payout_ref=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.Reason, id, tenantID)
		err = e
		if err == nil && tag.RowsAffected() == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.audit(ctx, adminID, action, "offramp_order", id.String(), gin.H{"reason": req.Reason})
	c.JSON(http.StatusOK, gin.H{action: id})
}

func (s *Svc) ApproveOfframpOrder(c *gin.Context) { s.reviewOfframpOrder(c, "completed", "offramp.approve") }
func (s *Svc) RejectOfframpOrder(c *gin.Context)  { s.reviewOfframpOrder(c, "rejected", "offramp.reject") }
