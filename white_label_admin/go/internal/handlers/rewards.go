package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tigerwallet/white-label-admin/internal/middleware"
)

// ==================== Reward campaigns (reward_admin scope) ====================
// Governance records only — no fund movement.

func (s *Svc) ListRewardCampaigns(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	ctx := c.Request.Context()
	rows, err := s.db.Query(ctx,
		`SELECT id, name, reward_type, amount, COALESCE(token,''), status, start_at, end_at, created_at, updated_at
		 FROM reward_campaigns WHERE white_label_id=$1 ORDER BY created_at DESC LIMIT 100`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var name, rewardType, token, status, amount string
		var startAt, endAt, created, updated time.Time
		_ = rows.Scan(&id, &name, &rewardType, &amount, &token, &status, &startAt, &endAt, &created, &updated)
		out = append(out, gin.H{"id": id, "name": name, "reward_type": rewardType, "amount": amount, "token": token,
			"status": status, "start_at": startAt, "end_at": endAt, "created_at": created, "updated_at": updated})
	}
	c.JSON(http.StatusOK, gin.H{"campaigns": out})
}

func (s *Svc) GetRewardCampaign(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	var name, rewardType, token, status, amount string
	var startAt, endAt, created, updated time.Time
	err = s.db.QueryRow(ctx,
		`SELECT name, reward_type, amount, COALESCE(token,''), status, start_at, end_at, created_at, updated_at
		 FROM reward_campaigns WHERE id=$1 AND white_label_id=$2`, id, tenantID).
		Scan(&name, &rewardType, &amount, &token, &status, &startAt, &endAt, &created, &updated)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "campaign not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "name": name, "reward_type": rewardType, "amount": amount, "token": token,
		"status": status, "start_at": startAt, "end_at": endAt, "created_at": created, "updated_at": updated})
}

func (s *Svc) CreateRewardCampaign(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	var req struct {
		Name       string  `json:"name" binding:"required"`
		RewardType string  `json:"reward_type" binding:"required"`
		Amount     float64 `json:"amount"`
		Token      string  `json:"token"`
		StartAt    string  `json:"start_at"`
		EndAt      string  `json:"end_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var startAt, endAt any
	if req.StartAt != "" {
		if parsed, e := time.Parse(time.RFC3339, req.StartAt); e == nil {
			startAt = parsed
		}
	}
	if req.EndAt != "" {
		if parsed, e := time.Parse(time.RFC3339, req.EndAt); e == nil {
			endAt = parsed
		}
	}
	id := uuid.New()
	ctx := c.Request.Context()
	_, err := s.db.Exec(ctx,
		`INSERT INTO reward_campaigns (id, name, reward_type, amount, token, status, start_at, end_at, white_label_id)
		 VALUES ($1,$2,$3,$4,$5,'active',$6,$7,$8)`,
		id, req.Name, req.RewardType, req.Amount, req.Token, startAt, endAt, tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "reward.create", "reward_campaign", id.String(), gin.H{"name": req.Name})
	c.JSON(http.StatusCreated, gin.H{"id": id, "name": req.Name, "status": "active"})
}

func (s *Svc) UpdateRewardCampaign(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Name       string  `json:"name"`
		RewardType string  `json:"reward_type"`
		Amount     float64 `json:"amount"`
		Token      string  `json:"token"`
		StartAt    string  `json:"start_at"`
		EndAt      string  `json:"end_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	if req.Name != "" {
		_, _ = s.db.Exec(ctx, `UPDATE reward_campaigns SET name=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.Name, id, tenantID)
	}
	if req.RewardType != "" {
		_, _ = s.db.Exec(ctx, `UPDATE reward_campaigns SET reward_type=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.RewardType, id, tenantID)
	}
	var startAt, endAt any
	if req.StartAt != "" {
		if parsed, e := time.Parse(time.RFC3339, req.StartAt); e == nil {
			startAt = parsed
			_, _ = s.db.Exec(ctx, `UPDATE reward_campaigns SET start_at=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, startAt, id, tenantID)
		}
	}
	if req.EndAt != "" {
		if parsed, e := time.Parse(time.RFC3339, req.EndAt); e == nil {
			endAt = parsed
			_, _ = s.db.Exec(ctx, `UPDATE reward_campaigns SET end_at=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, endAt, id, tenantID)
		}
	}
	_, err = s.db.Exec(ctx,
		`UPDATE reward_campaigns SET amount=$1, token=$2, updated_at=NOW()
		 WHERE id=$3 AND white_label_id=$4`,
		req.Amount, req.Token, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "reward.update", "reward_campaign", id.String(), nil)
	c.JSON(http.StatusOK, gin.H{"updated": id})
}

func (s *Svc) DeleteRewardCampaign(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	ct, err := s.db.Exec(ctx, `DELETE FROM reward_campaigns WHERE id=$1 AND white_label_id=$2`, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "campaign not found"})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "reward.delete", "reward_campaign", id.String(), nil)
	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

func (s *Svc) UpdateRewardCampaignStatus(c *gin.Context) {
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
	ct, err := s.db.Exec(ctx, `UPDATE reward_campaigns SET status=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.Status, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "campaign not found"})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "reward.status", "reward_campaign", id.String(), gin.H{"status": req.Status})
	c.JSON(http.StatusOK, gin.H{"updated": id, "status": req.Status})
}
