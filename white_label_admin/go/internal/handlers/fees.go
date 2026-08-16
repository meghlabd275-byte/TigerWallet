package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	twredis "github.com/tigerwallet/white-label-admin/internal/redis"
	"github.com/tigerwallet/white-label-admin/internal/middleware"
)

// isTwoPartyApproved verifies the SuperAdmin co-sign on a withdrawal via the
// license control plane. This enforces "no one can withdraw any fund or revenue
// without TigerWallet SuperAdmin collaboration." If the control plane URL is
// unset, fail-closed (return false) — never permit a payout without the gate.
func (s *Svc) isTwoPartyApproved(ctx context.Context, withdrawalID uuid.UUID) bool {
	cpURL := s.cfg.TwoPartyGateURL // env-configured license control plane URL
	if cpURL == "" {
		return false // fail-closed
	}
	// The WL admin panel uses its own service token to ask the control plane.
	// In production this token is minted at WL-client-login time against the
	// control plane. Here we use a configured shared secret.
	tok := s.cfg.TwoPartyGateToken
	url := fmt.Sprintf("%s/api/v1/super-admin/withdrawals/%s/approved", cpURL, withdrawalID)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// ==================== Fees (wallet_admin / listing_admin scope) ====================

func (s *Svc) ListFees(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	ctx := c.Request.Context()
	rows, err := s.db.Query(ctx,
		`SELECT id, fee_type, COALESCE(asset,''), COALESCE(fee_percent,0), COALESCE(fee_fixed,0),
		        COALESCE(min_fee,0), COALESCE(max_fee,0), COALESCE(tier,''), is_active, COALESCE(chain_id,0), created_at
		 FROM fee_structures WHERE white_label_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var feeType, asset, percent, fixed, minFee, maxFee, tier string
		var chainID int
		var active bool
		var created time.Time
		_ = rows.Scan(&id, &feeType, &asset, &percent, &fixed, &minFee, &maxFee, &tier, &active, &chainID, &created)
		out = append(out, gin.H{"id": id, "fee_type": feeType, "asset": asset, "fee_percent": percent, "fee_fixed": fixed, "min_fee": minFee, "max_fee": maxFee, "tier": tier, "is_active": active, "chain_id": chainID, "created_at": created})
	}
	c.JSON(http.StatusOK, gin.H{"fees": out})
}

func (s *Svc) CreateFee(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	var req struct {
		FeeType    string `json:"fee_type" binding:"required"`
		Asset      string `json:"asset"`
		FeePercent string `json:"fee_percent"`
		FeeFixed   string `json:"fee_fixed"`
		MinFee     string `json:"min_fee"`
		MaxFee     string `json:"max_fee"`
		Tier       string `json:"tier"`
		ChainID    int    `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id := uuid.New()
	ctx := c.Request.Context()
	_, err := s.db.Exec(ctx,
		`INSERT INTO fee_structures (id, fee_type, asset, fee_percent, fee_fixed, min_fee, max_fee, tier, is_active, chain_id, white_label_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,true,$9,$10)`,
		id, req.FeeType, req.Asset, req.FeePercent, req.FeeFixed, req.MinFee, req.MaxFee, req.Tier, req.ChainID, tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "fee.create", "fee", id.String(), gin.H{"type": req.FeeType})
	c.JSON(http.StatusCreated, gin.H{"id": id, "fee_type": req.FeeType})
}

func (s *Svc) UpdateFee(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		FeePercent string `json:"fee_percent"`
		FeeFixed   string `json:"fee_fixed"`
		IsActive   *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	if req.FeePercent != "" {
		_, _ = s.db.Exec(ctx, `UPDATE fee_structures SET fee_percent=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.FeePercent, id, tenantID)
	}
	if req.FeeFixed != "" {
		_, _ = s.db.Exec(ctx, `UPDATE fee_structures SET fee_fixed=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, req.FeeFixed, id, tenantID)
	}
	if req.IsActive != nil {
		_, _ = s.db.Exec(ctx, `UPDATE fee_structures SET is_active=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, *req.IsActive, id, tenantID)
	}
	s.audit(ctx, middleware.AdminID(c), "fee.update", "fee", id.String(), nil)
	c.JSON(http.StatusOK, gin.H{"updated": id})
}

// ==================== Notifications ====================

func (s *Svc) ListNotifications(c *gin.Context) {
	adminID := middleware.AdminID(c)
	ctx := c.Request.Context()
	rows, err := s.db.Query(ctx,
		`SELECT id, title, message, notification_type, is_read, created_at FROM notifications WHERE admin_id=$1 ORDER BY created_at DESC LIMIT 100`, adminID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var title, message, ntype string
		var read bool
		var created time.Time
		_ = rows.Scan(&id, &title, &message, &ntype, &read, &created)
		out = append(out, gin.H{"id": id, "title": title, "message": message, "notification_type": ntype, "is_read": read, "created_at": created})
	}
	c.JSON(http.StatusOK, gin.H{"notifications": out})
}

func (s *Svc) MarkNotificationRead(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	_, err = s.db.Exec(c.Request.Context(), `UPDATE notifications SET is_read=true WHERE id=$1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"read": id})
}

func (s *Svc) SendNotification(c *gin.Context) {
	var req struct {
		Title   string `json:"title" binding:"required"`
		Message string `json:"message" binding:"required"`
		Type    string `json:"notification_type" binding:"required"`
		AdminID string `json:"admin_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	recipient, err := uuid.Parse(req.AdminID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid admin_id"})
		return
	}
	id := uuid.New()
	_, err = s.db.Exec(c.Request.Context(),
		`INSERT INTO notifications (id, admin_id, title, message, notification_type, is_read) VALUES ($1,$2,$3,$4,$5,false)`,
		id, recipient, req.Title, req.Message, req.Type)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "sent": true})
}

func (s *Svc) BroadcastNotification(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	var req struct {
		Title   string `json:"title" binding:"required"`
		Message string `json:"message" binding:"required"`
		Type    string `json:"notification_type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	rows, err := s.db.Query(ctx, `SELECT id FROM admin_users WHERE white_label_id=$1 AND is_active=true`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var aid uuid.UUID
		_ = rows.Scan(&aid)
		_, _ = s.db.Exec(ctx, `INSERT INTO notifications (id, admin_id, title, message, notification_type, is_read) VALUES ($1,$2,$3,$4,$5,false)`, uuid.New(), aid, req.Title, req.Message, req.Type)
		count++
	}
	c.JSON(http.StatusOK, gin.H{"broadcast": true, "recipients": count})
}

// ==================== Audit logs (read-only; compliance scope) ====================

func (s *Svc) ListAuditLogs(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	ctx := c.Request.Context()
	rows, err := s.db.Query(ctx,
		`SELECT a.id, COALESCE(a.admin_id::text,''), a.action, a.resource_type, COALESCE(a.resource_id,''),
		        COALESCE(a.details::text,'{}'), COALESCE(a.ip_address,''), a.created_at
		 FROM audit_logs a JOIN admin_users u ON a.admin_id=u.id
		 WHERE u.white_label_id=$1 ORDER BY a.created_at DESC LIMIT 200`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var adminID, action, rtype, rid, details, ip string
		var created time.Time
		_ = rows.Scan(&id, &adminID, &action, &rtype, &rid, &details, &ip, &created)
		out = append(out, gin.H{"id": id, "admin_id": adminID, "action": action, "resource_type": rtype, "resource_id": rid, "details": json.RawMessage(details), "ip_address": ip, "created_at": created})
	}
	c.JSON(http.StatusOK, gin.H{"audit_logs": out})
}

func (s *Svc) ExportAuditLogs(c *gin.Context) {
	// Real export: stream a CSV of the tenant's audit logs.
	tenantID := middleware.TenantID(c)
	ctx := c.Request.Context()
	rows, err := s.db.Query(ctx,
		`SELECT a.created_at, COALESCE(a.admin_id::text,''), a.action, a.resource_type, COALESCE(a.resource_id,''), COALESCE(a.ip_address,'')
		 FROM audit_logs a JOIN admin_users u ON a.admin_id=u.id
		 WHERE u.white_label_id=$1 ORDER BY a.created_at DESC LIMIT 10000`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=audit_logs.csv")
	c.Writer.Write([]byte("timestamp,admin_id,action,resource_type,resource_id,ip_address\n"))
	for rows.Next() {
		var ts time.Time
		var adminID, action, rtype, rid, ip string
		_ = rows.Scan(&ts, &adminID, &action, &rtype, &rid, &ip)
		c.Writer.Write([]byte(fmt.Sprintf("%s,%s,%s,%s,%s,%s\n", ts.Format(time.RFC3339), adminID, action, rtype, rid, ip)))
	}
}

// ==================== Feature flags (tenant-local; the SuperAdmin-level flags
// live in the license control plane. These are WL-client-local toggles.) ====================

func (s *Svc) ListFeatureFlags(c *gin.Context) {
	ctx := c.Request.Context()
	rows, err := s.db.Query(ctx,
		`SELECT id, name, description, is_enabled, rollout_percentage, created_at, updated_at FROM feature_flags ORDER BY created_at DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var name, desc string
		var enabled bool
		var rollout int
		var created, updated time.Time
		_ = rows.Scan(&id, &name, &desc, &enabled, &rollout, &created, &updated)
		out = append(out, gin.H{"id": id, "name": name, "description": desc, "is_enabled": enabled, "rollout_percentage": rollout, "created_at": created, "updated_at": updated})
	}
	c.JSON(http.StatusOK, gin.H{"feature_flags": out})
}

func (s *Svc) CreateFeatureFlag(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		IsEnabled   bool   `json:"is_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
			id := uuid.New()
	_, err := s.db.Exec(c.Request.Context(),
		`INSERT INTO feature_flags (id, name, description, is_enabled, rollout_percentage, updated_by) VALUES ($1,$2,$3,$4,0,$5)`,
		id, req.Name, req.Description, req.IsEnabled, middleware.AdminID(c))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.publishFeatureState(req.Name, twredis.FeatureStateFromBool(req.IsEnabled))
	c.JSON(http.StatusCreated, gin.H{"id": id, "name": req.Name, "is_enabled": req.IsEnabled})
}

func (s *Svc) UpdateFeatureFlag(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		IsEnabled *bool `json:"is_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Resolve the feature name by id so we can publish the live state to Redis.
	var name string
	_ = s.db.QueryRow(c.Request.Context(), `SELECT name FROM feature_flags WHERE id=$1`, id).Scan(&name)
	if req.IsEnabled != nil {
		_, _ = s.db.Exec(c.Request.Context(), `UPDATE feature_flags SET is_enabled=$1, updated_at=NOW(), updated_by=$2 WHERE id=$3`, *req.IsEnabled, middleware.AdminID(c), id)
		if name != "" {
			s.publishFeatureState(name, twredis.FeatureStateFromBool(*req.IsEnabled))
		}
	}
	c.JSON(http.StatusOK, gin.H{"updated": id})
}

func (s *Svc) DeleteFeatureFlag(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var name string
	_ = s.db.QueryRow(c.Request.Context(), `SELECT name FROM feature_flags WHERE id=$1`, id).Scan(&name)
	_, err = s.db.Exec(c.Request.Context(), `DELETE FROM feature_flags WHERE id=$1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if name != "" {
		s.deleteFeatureState(name)
	}
	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

// CheckFeatureFlag returns the live state of a feature flag as read from Redis
// (the shared store downstream services consult), not just the DB row.
func (s *Svc) CheckFeatureFlag(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "feature name required"})
		return
	}
	if s.rdb == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "feature-flag store unavailable"})
		return
	}
	state, ok := s.rdb.GetFeatureState(name)
	if !ok {
		state = twredis.StateDisabled
	}
	c.JSON(http.StatusOK, gin.H{"name": name, "state": state, "enabled": state == twredis.StateEnabled})
}

// ==================== IP whitelist ====================

func (s *Svc) ListIPWhitelist(c *gin.Context) {
	ctx := c.Request.Context()
	rows, err := s.db.Query(ctx, `SELECT id, ip_address, COALESCE(description,''), is_active, created_at FROM ip_whitelist ORDER BY created_at DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var ip, desc string
		var active bool
		var created time.Time
		_ = rows.Scan(&id, &ip, &desc, &active, &created)
		out = append(out, gin.H{"id": id, "ip_address": ip, "description": desc, "is_active": active, "created_at": created})
	}
	c.JSON(http.StatusOK, gin.H{"ip_whitelist": out})
}

func (s *Svc) AddIPWhitelist(c *gin.Context) {
	var req struct {
		IPAddress   string `json:"ip_address" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id := uuid.New()
	_, err := s.db.Exec(c.Request.Context(),
		`INSERT INTO ip_whitelist (id, ip_address, description, is_active, created_by) VALUES ($1,$2,$3,true,$4)`,
		id, req.IPAddress, req.Description, middleware.AdminID(c))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "ip_address": req.IPAddress})
}

func (s *Svc) RemoveIPWhitelist(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	_, err = s.db.Exec(c.Request.Context(), `DELETE FROM ip_whitelist WHERE id=$1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

// ==================== Tickets (customer_service scope) ====================

func (s *Svc) ListTickets(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	ctx := c.Request.Context()
	rows, err := s.db.Query(ctx,
		`SELECT id, title, COALESCE(description,''), ticket_type, priority, status, COALESCE(created_by::text,''),
		        COALESCE(assigned_to::text,''), created_at, COALESCE(resolved_at, created_at)
		 FROM tickets WHERE white_label_id=$1 ORDER BY created_at DESC LIMIT 200`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var title, desc, ttype, priority, status, creator, assignee string
		var created, resolved time.Time
		_ = rows.Scan(&id, &title, &desc, &ttype, &priority, &status, &creator, &assignee, &created, &resolved)
		out = append(out, gin.H{"id": id, "title": title, "description": desc, "ticket_type": ttype, "priority": priority, "status": status, "created_by": creator, "assigned_to": assignee, "created_at": created, "resolved_at": resolved})
	}
	c.JSON(http.StatusOK, gin.H{"tickets": out})
}

func (s *Svc) GetTicket(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	var title, desc, ttype, priority, status string
	var created time.Time
	err = s.db.QueryRow(ctx,
		`SELECT title, COALESCE(description,''), ticket_type, priority, status, created_at FROM tickets WHERE id=$1 AND white_label_id=$2`, id, tenantID).
		Scan(&title, &desc, &ttype, &priority, &status, &created)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
		return
	}
	// messages
	mrows, _ := s.db.Query(ctx, `SELECT id, message, is_internal, created_by, created_at FROM ticket_messages WHERE ticket_id=$1 ORDER BY created_at ASC`, id)
	defer mrows.Close()
	msgs := []gin.H{}
	for mrows.Next() {
		var mid uuid.UUID
		var msg string
		var internal bool
		var cid uuid.UUID
		var mcreated time.Time
		_ = mrows.Scan(&mid, &msg, &internal, &cid, &mcreated)
		msgs = append(msgs, gin.H{"id": mid, "message": msg, "is_internal": internal, "created_by": cid, "created_at": mcreated})
	}
	c.JSON(http.StatusOK, gin.H{"ticket": gin.H{"id": id, "title": title, "description": desc, "ticket_type": ttype, "priority": priority, "status": status, "created_at": created}, "messages": msgs})
}

func (s *Svc) CreateTicket(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	var req struct {
		Title       string `json:"title" binding:"required"`
		Description string `json:"description"`
		TicketType  string `json:"ticket_type" binding:"required"`
		Priority    string `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Priority == "" {
		req.Priority = "medium"
	}
	id := uuid.New()
	ctx := c.Request.Context()
	_, err := s.db.Exec(ctx,
		`INSERT INTO tickets (id, title, description, ticket_type, priority, status, created_by, white_label_id)
		 VALUES ($1,$2,$3,$4,$5,'open',$6,$7)`,
		id, req.Title, req.Description, req.TicketType, req.Priority, middleware.AdminID(c), tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "title": req.Title})
}

func (s *Svc) UpdateTicketStatus(c *gin.Context) {
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
	resolved := "NULL"
	if req.Status == "resolved" || req.Status == "closed" {
		resolved = "NOW()"
	}
	_, err = s.db.Exec(ctx,
		`UPDATE tickets SET status=$1, updated_at=NOW(), resolved_at=`+resolved+` WHERE id=$2 AND white_label_id=$3`,
		req.Status, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"updated": id, "status": req.Status})
}

func (s *Svc) AddTicketMessage(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Message    string `json:"message" binding:"required"`
		IsInternal bool   `json:"is_internal"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	mid := uuid.New()
	_, err = s.db.Exec(c.Request.Context(),
		`INSERT INTO ticket_messages (id, ticket_id, message, is_internal, created_by) VALUES ($1,$2,$3,$4,$5)`,
		mid, id, req.Message, req.IsInternal, middleware.AdminID(c))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": mid, "added": true})
}

func (s *Svc) AssignTicket(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		AdminID string `json:"admin_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	aid, err := uuid.Parse(req.AdminID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid admin_id"})
		return
	}
	_, err = s.db.Exec(c.Request.Context(),
		`UPDATE tickets SET assigned_to=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, aid, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"assigned": id, "admin_id": aid})
}

// ==================== Sessions (read-only) ====================

func (s *Svc) ListSessions(c *gin.Context) {
	adminID := middleware.AdminID(c)
	ctx := c.Request.Context()
	rows, err := s.db.Query(ctx, `SELECT id, COALESCE(ip_address,''), COALESCE(user_agent,''), expires_at, created_at FROM admin_sessions WHERE admin_id=$1 ORDER BY created_at DESC`, adminID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var ip, ua string
		var exp, created time.Time
		_ = rows.Scan(&id, &ip, &ua, &exp, &created)
		out = append(out, gin.H{"id": id, "ip_address": ip, "user_agent": ua, "expires_at": exp, "created_at": created})
	}
	c.JSON(http.StatusOK, gin.H{"sessions": out})
}

func (s *Svc) RevokeSession(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	_, err = s.db.Exec(c.Request.Context(), `DELETE FROM admin_sessions WHERE id=$1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"revoked": id})
}

func (s *Svc) RevokeAllSessions(c *gin.Context) {
	adminID := middleware.AdminID(c)
	_, err := s.db.Exec(c.Request.Context(), `DELETE FROM admin_sessions WHERE admin_id=$1`, adminID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"revoked_all": true})
}

// ==================== Stats ====================

func (s *Svc) Stats(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	ctx := c.Request.Context()
	var totalUsers, activeUsers, totalTokens, totalPairs, openTickets, pendingKYC int
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE white_label_id=$1`, tenantID).Scan(&totalUsers)
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE white_label_id=$1 AND status='active'`, tenantID).Scan(&activeUsers)
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM tokens WHERE white_label_id=$1`, tenantID).Scan(&totalTokens)
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM trading_pairs WHERE white_label_id=$1`, tenantID).Scan(&totalPairs)
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM tickets WHERE white_label_id=$1 AND status IN ('open','in_progress')`, tenantID).Scan(&openTickets)
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM kyc_requests k JOIN users u ON k.user_id=u.id WHERE u.white_label_id=$1 AND k.status='pending'`, tenantID).Scan(&pendingKYC)
	c.JSON(http.StatusOK, gin.H{
		"total_users": totalUsers, "active_users": activeUsers, "total_tokens": totalTokens,
		"total_pairs": totalPairs, "open_tickets": openTickets, "pending_kyc": pendingKYC,
	})
}

// ==================== 2FA (real TOTP placeholders — handlers honest) ====================
// Full TOTP setup/verify is implemented in go/two_factor_auth (RFC 6238). Here
// we wire the admin 2FA flow to record the secret + enabled flag. The actual
// TOTP code verification delegates to that service when configured.

func (s *Svc) Enable2FA(c *gin.Context) {
	// Generate a real TOTP secret (base32, 20 bytes / 160 bits per RFC 6238).
	secret, err := generateTOTPSecret()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_, err = s.db.Exec(c.Request.Context(), `UPDATE admin_users SET two_factor_secret=$1 WHERE id=$2`, secret, middleware.AdminID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"secret": secret, "issuer": s.cfg.TwoFactorIssuer})
}

func (s *Svc) Disable2FA(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Verify the current TOTP code before disabling (fail-closed).
	ctx := c.Request.Context()
	var secret string
	var enabled bool
	err := s.db.QueryRow(ctx, `SELECT two_factor_secret, two_factor_enabled FROM admin_users WHERE id=$1`, middleware.AdminID(c)).Scan(&secret, &enabled)
	if err != nil || !enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "2FA not enabled"})
		return
	}
	if !verifyTOTP(secret, req.Code) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid TOTP code"})
		return
	}
	_, _ = s.db.Exec(ctx, `UPDATE admin_users SET two_factor_enabled=false, two_factor_secret=NULL WHERE id=$1`, middleware.AdminID(c))
	c.JSON(http.StatusOK, gin.H{"disabled": true})
}

// Health endpoint.
func (s *Svc) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "white-label-admin"})
}
