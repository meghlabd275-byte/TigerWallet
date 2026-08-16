package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tigerwallet/white-label-admin/internal/middleware"
)

// ==================== Partners (listing_admin scope) ====================
// Governance records only — no fund movement. Approve/Reject are WL-side
// governance decisions for partner onboarding.

func (s *Svc) ListPartners(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	ctx := c.Request.Context()
	rows, err := s.db.Query(ctx,
		`SELECT id, name, COALESCE(contact_email,''), COALESCE(api_key,''), status, revenue_share, created_at, updated_at
		 FROM partners WHERE white_label_id=$1 ORDER BY created_at DESC LIMIT 100`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var name, email, apiKey, status string
		var revenueShare string
		var created, updated time.Time
		_ = rows.Scan(&id, &name, &email, &apiKey, &status, &revenueShare, &created, &updated)
		out = append(out, gin.H{"id": id, "name": name, "contact_email": email, "api_key": apiKey,
			"status": status, "revenue_share": revenueShare, "created_at": created, "updated_at": updated})
	}
	c.JSON(http.StatusOK, gin.H{"partners": out})
}

func (s *Svc) GetPartner(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	var name, email, apiKey, status string
	var revenueShare string
	var created, updated time.Time
	err = s.db.QueryRow(ctx,
		`SELECT name, COALESCE(contact_email,''), COALESCE(api_key,''), status, revenue_share, created_at, updated_at
		 FROM partners WHERE id=$1 AND white_label_id=$2`, id, tenantID).
		Scan(&name, &email, &apiKey, &status, &revenueShare, &created, &updated)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "partner not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "name": name, "contact_email": email, "api_key": apiKey,
		"status": status, "revenue_share": revenueShare, "created_at": created, "updated_at": updated})
}

func (s *Svc) CreatePartner(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	var req struct {
		Name         string  `json:"name" binding:"required"`
		ContactEmail string  `json:"contact_email"`
		RevenueShare float64 `json:"revenue_share"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id := uuid.New()
	ctx := c.Request.Context()
	_, err := s.db.Exec(ctx,
		`INSERT INTO partners (id, name, contact_email, revenue_share, status, white_label_id)
		 VALUES ($1,$2,$3,$4,'pending',$5)`,
		id, req.Name, req.ContactEmail, req.RevenueShare, tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "partner.create", "partner", id.String(), gin.H{"name": req.Name})
	c.JSON(http.StatusCreated, gin.H{"id": id, "name": req.Name, "status": "pending"})
}

func (s *Svc) UpdatePartner(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Name         string  `json:"name"`
		ContactEmail string  `json:"contact_email"`
		RevenueShare float64 `json:"revenue_share"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	if req.Name != "" {
		_, _ = s.db.Exec(ctx, `UPDATE partners SET name=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.Name, id, tenantID)
	}
	_, err = s.db.Exec(ctx,
		`UPDATE partners SET contact_email=$1, revenue_share=$2, updated_at=NOW()
		 WHERE id=$3 AND white_label_id=$4`,
		req.ContactEmail, req.RevenueShare, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "partner.update", "partner", id.String(), nil)
	c.JSON(http.StatusOK, gin.H{"updated": id})
}

func (s *Svc) DeletePartner(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	ct, err := s.db.Exec(ctx, `DELETE FROM partners WHERE id=$1 AND white_label_id=$2`, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "partner not found"})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "partner.delete", "partner", id.String(), nil)
	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

func (s *Svc) UpdatePartnerStatus(c *gin.Context) {
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
	ct, err := s.db.Exec(ctx, `UPDATE partners SET status=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.Status, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "partner not found"})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "partner.status", "partner", id.String(), gin.H{"status": req.Status})
	c.JSON(http.StatusOK, gin.H{"updated": id, "status": req.Status})
}

func (s *Svc) reviewPartner(c *gin.Context, status, action string) {
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
	tag, err := s.db.Exec(ctx, `UPDATE partners SET status=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, status, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "partner not found"})
		return
	}
	s.audit(ctx, adminID, action, "partner", id.String(), gin.H{"reason": req.Reason})
	c.JSON(http.StatusOK, gin.H{action: id})
}

func (s *Svc) ApprovePartner(c *gin.Context) { s.reviewPartner(c, "approved", "partner.approve") }
func (s *Svc) RejectPartner(c *gin.Context)  { s.reviewPartner(c, "rejected", "partner.reject") }
