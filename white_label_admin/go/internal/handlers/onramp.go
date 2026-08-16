package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tigerwallet/white-label-admin/internal/middleware"
)

// ==================== Onramp orders (p2p_admin scope) ====================
// Governance records only — no fund movement. Approve/Reject are WL-side
// governance decisions; actual fiat/crypto settlement is handled by the
// fiat-ramp backend.

func (s *Svc) ListOnrampOrders(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	ctx := c.Request.Context()
	rows, err := s.db.Query(ctx,
		`SELECT id, COALESCE(user_id::text,''), provider, fiat_currency, crypto_token, fiat_amount, crypto_amount,
		        status, COALESCE(payment_ref,''), created_at, updated_at
		 FROM onramp_orders WHERE white_label_id=$1 ORDER BY created_at DESC LIMIT 100`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var userID, provider, fiatCurrency, cryptoToken, status, paymentRef string
		var fiatAmount, cryptoAmount string
		var created, updated time.Time
		_ = rows.Scan(&id, &userID, &provider, &fiatCurrency, &cryptoToken, &fiatAmount, &cryptoAmount, &status, &paymentRef, &created, &updated)
		out = append(out, gin.H{"id": id, "user_id": userID, "provider": provider, "fiat_currency": fiatCurrency,
			"crypto_token": cryptoToken, "fiat_amount": fiatAmount, "crypto_amount": cryptoAmount, "status": status,
			"payment_ref": paymentRef, "created_at": created, "updated_at": updated})
	}
	c.JSON(http.StatusOK, gin.H{"orders": out})
}

func (s *Svc) GetOnrampOrder(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	var userID, provider, fiatCurrency, cryptoToken, status, paymentRef string
	var fiatAmount, cryptoAmount string
	var created, updated time.Time
	err = s.db.QueryRow(ctx,
		`SELECT COALESCE(user_id::text,''), provider, fiat_currency, crypto_token, fiat_amount, crypto_amount,
		        status, COALESCE(payment_ref,''), created_at, updated_at
		 FROM onramp_orders WHERE id=$1 AND white_label_id=$2`, id, tenantID).
		Scan(&userID, &provider, &fiatCurrency, &cryptoToken, &fiatAmount, &cryptoAmount, &status, &paymentRef, &created, &updated)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "user_id": userID, "provider": provider, "fiat_currency": fiatCurrency,
		"crypto_token": cryptoToken, "fiat_amount": fiatAmount, "crypto_amount": cryptoAmount, "status": status,
		"payment_ref": paymentRef, "created_at": created, "updated_at": updated})
}

func (s *Svc) CreateOnrampOrder(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	var req struct {
		UserID       string  `json:"user_id"`
		Provider     string  `json:"provider" binding:"required"`
		FiatCurrency string  `json:"fiat_currency" binding:"required"`
		CryptoToken  string  `json:"crypto_token" binding:"required"`
		FiatAmount   float64 `json:"fiat_amount"`
		CryptoAmount float64 `json:"crypto_amount"`
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
		`INSERT INTO onramp_orders (id, user_id, provider, fiat_currency, crypto_token, fiat_amount, crypto_amount, status, white_label_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,'pending',$8)`,
		id, uid, req.Provider, req.FiatCurrency, req.CryptoToken, req.FiatAmount, req.CryptoAmount, tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "onramp.create", "onramp_order", id.String(), gin.H{"provider": req.Provider})
	c.JSON(http.StatusCreated, gin.H{"id": id, "status": "pending"})
}

func (s *Svc) UpdateOnrampOrder(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Provider     string  `json:"provider"`
		FiatCurrency string  `json:"fiat_currency"`
		CryptoToken  string  `json:"crypto_token"`
		FiatAmount   float64 `json:"fiat_amount"`
		CryptoAmount float64 `json:"crypto_amount"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	if req.Provider != "" {
		_, _ = s.db.Exec(ctx, `UPDATE onramp_orders SET provider=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.Provider, id, tenantID)
	}
	if req.FiatCurrency != "" {
		_, _ = s.db.Exec(ctx, `UPDATE onramp_orders SET fiat_currency=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.FiatCurrency, id, tenantID)
	}
	if req.CryptoToken != "" {
		_, _ = s.db.Exec(ctx, `UPDATE onramp_orders SET crypto_token=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.CryptoToken, id, tenantID)
	}
	_, err = s.db.Exec(ctx,
		`UPDATE onramp_orders SET fiat_amount=$1, crypto_amount=$2, updated_at=NOW()
		 WHERE id=$3 AND white_label_id=$4`,
		req.FiatAmount, req.CryptoAmount, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "onramp.update", "onramp_order", id.String(), nil)
	c.JSON(http.StatusOK, gin.H{"updated": id})
}

func (s *Svc) DeleteOnrampOrder(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	ct, err := s.db.Exec(ctx, `DELETE FROM onramp_orders WHERE id=$1 AND white_label_id=$2`, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "onramp.delete", "onramp_order", id.String(), nil)
	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

func (s *Svc) reviewOnrampOrder(c *gin.Context, status, action string) {
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
	var ct int64
	if status == "completed" {
		tag, e := s.db.Exec(ctx, `UPDATE onramp_orders SET status='completed', updated_at=NOW() WHERE id=$1 AND white_label_id=$2`, id, tenantID)
		ct, err = tag.RowsAffected(), e
	} else {
		tag, e := s.db.Exec(ctx, `UPDATE onramp_orders SET status='rejected', payment_ref=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.Reason, id, tenantID)
		ct, err = tag.RowsAffected(), e
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ct == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	s.audit(ctx, adminID, action, "onramp_order", id.String(), gin.H{"reason": req.Reason})
	c.JSON(http.StatusOK, gin.H{action: id})
}

func (s *Svc) ApproveOnrampOrder(c *gin.Context) { s.reviewOnrampOrder(c, "completed", "onramp.approve") }
func (s *Svc) RejectOnrampOrder(c *gin.Context)  { s.reviewOnrampOrder(c, "rejected", "onramp.reject") }
