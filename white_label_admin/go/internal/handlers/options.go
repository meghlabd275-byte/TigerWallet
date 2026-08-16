package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tigerwallet/white-label-admin/internal/middleware"
)

// ==================== Options contracts (trading_admin scope) ====================
// Governance records only — no fund movement.

func (s *Svc) ListOptionsContracts(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	ctx := c.Request.Context()
	rows, err := s.db.Query(ctx,
		`SELECT id, COALESCE(user_id::text,''), underlying, option_type, strike, expiry, premium, size,
		        status, COALESCE(chain_id,0), created_at, updated_at
		 FROM options_contracts WHERE white_label_id=$1 ORDER BY created_at DESC LIMIT 100`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var userID, underlying, optionType, status string
		var strike, premium, size string
		var chainID int64
		var expiry, created, updated time.Time
		_ = rows.Scan(&id, &userID, &underlying, &optionType, &strike, &expiry, &premium, &size, &status, &chainID, &created, &updated)
		out = append(out, gin.H{"id": id, "user_id": userID, "underlying": underlying, "option_type": optionType,
			"strike": strike, "expiry": expiry, "premium": premium, "size": size, "status": status, "chain_id": chainID,
			"created_at": created, "updated_at": updated})
	}
	c.JSON(http.StatusOK, gin.H{"contracts": out})
}

func (s *Svc) GetOptionsContract(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	var userID, underlying, optionType, status string
	var strike, premium, size string
	var chainID int64
	var expiry, created, updated time.Time
	err = s.db.QueryRow(ctx,
		`SELECT COALESCE(user_id::text,''), underlying, option_type, strike, expiry, premium, size,
		        status, COALESCE(chain_id,0), created_at, updated_at
		 FROM options_contracts WHERE id=$1 AND white_label_id=$2`, id, tenantID).
		Scan(&userID, &underlying, &optionType, &strike, &expiry, &premium, &size, &status, &chainID, &created, &updated)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "contract not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "user_id": userID, "underlying": underlying, "option_type": optionType,
		"strike": strike, "expiry": expiry, "premium": premium, "size": size, "status": status, "chain_id": chainID,
		"created_at": created, "updated_at": updated})
}

func (s *Svc) CreateOptionsContract(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	var req struct {
		UserID     string  `json:"user_id"`
		Underlying string  `json:"underlying" binding:"required"`
		OptionType string  `json:"option_type" binding:"required"`
		Strike     float64 `json:"strike"`
		Expiry     string  `json:"expiry" binding:"required"`
		Premium    float64 `json:"premium"`
		Size       float64 `json:"size"`
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
	expiry, err := time.Parse(time.RFC3339, req.Expiry)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expiry (use RFC3339)"})
		return
	}
	id := uuid.New()
	ctx := c.Request.Context()
	_, err = s.db.Exec(ctx,
		`INSERT INTO options_contracts (id, user_id, underlying, option_type, strike, expiry, premium, size, chain_id, status, white_label_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'active',$10)`,
		id, uid, req.Underlying, req.OptionType, req.Strike, expiry, req.Premium, req.Size, req.ChainID, tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "options.create", "options_contract", id.String(), gin.H{"underlying": req.Underlying})
	c.JSON(http.StatusCreated, gin.H{"id": id, "underlying": req.Underlying, "status": "active"})
}

func (s *Svc) UpdateOptionsContract(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Underlying string  `json:"underlying"`
		OptionType string  `json:"option_type"`
		Strike     float64 `json:"strike"`
		Expiry     string  `json:"expiry"`
		Premium    float64 `json:"premium"`
		Size       float64 `json:"size"`
		ChainID    int64   `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	if req.Underlying != "" {
		_, _ = s.db.Exec(ctx, `UPDATE options_contracts SET underlying=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.Underlying, id, tenantID)
	}
	if req.OptionType != "" {
		_, _ = s.db.Exec(ctx, `UPDATE options_contracts SET option_type=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.OptionType, id, tenantID)
	}
	var expiryArg any
	if req.Expiry != "" {
		if parsed, e := time.Parse(time.RFC3339, req.Expiry); e == nil {
			expiryArg = parsed
			_, _ = s.db.Exec(ctx, `UPDATE options_contracts SET expiry=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, expiryArg, id, tenantID)
		}
	}
	_, err = s.db.Exec(ctx,
		`UPDATE options_contracts SET strike=$1, premium=$2, size=$3, chain_id=$4, updated_at=NOW()
		 WHERE id=$5 AND white_label_id=$6`,
		req.Strike, req.Premium, req.Size, req.ChainID, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "options.update", "options_contract", id.String(), nil)
	c.JSON(http.StatusOK, gin.H{"updated": id})
}

func (s *Svc) DeleteOptionsContract(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	ct, err := s.db.Exec(ctx, `DELETE FROM options_contracts WHERE id=$1 AND white_label_id=$2`, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "contract not found"})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "options.delete", "options_contract", id.String(), nil)
	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

func (s *Svc) UpdateOptionsContractStatus(c *gin.Context) {
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
	ct, err := s.db.Exec(ctx, `UPDATE options_contracts SET status=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.Status, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "contract not found"})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "options.status", "options_contract", id.String(), gin.H{"status": req.Status})
	c.JSON(http.StatusOK, gin.H{"updated": id, "status": req.Status})
}
