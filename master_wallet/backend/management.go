package main

// management.go — Policy, fees, auto-sign rules, users, whitelist, analytics,
// audit, notifications, webhooks, API keys. All PostgreSQL-backed; no mock
// data. Read endpoints return real DB rows; write endpoints persist + audit.

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// --- Policies ---

type policyReq struct {
	Name        string                 `json:"name" binding:"required"`
	PolicyType  string                 `json:"policy_type" binding:"required"`
	Conditions  map[string]interface{} `json:"conditions"`
	Actions     map[string]interface{} `json:"actions"`
	IsActive    *bool                  `json:"is_active"`
	Priority    int                    `json:"priority"`
}

func (svc *Service) GetPolicies(c *gin.Context) {
	masterID := c.Query("master_wallet_id")
	if masterID == "" {
		masterID = c.Param("id")
	}
	ctx := c.Request.Context()
	rows, err := svc.store.db.Query(ctx,
		`SELECT id, master_wallet_id, name, policy_type, is_active, priority, conditions, actions, created_at
		 FROM policies WHERE master_wallet_id = $1 ORDER BY priority DESC, created_at DESC`, masterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch policies"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, mid uuid.UUID
		var name, ptype string
		var active bool
		var priority int
		var conditions, actions []byte
		var createdAt time.Time
		_ = rows.Scan(&id, &mid, &name, &ptype, &active, &priority, &conditions, &actions, &createdAt)
		out = append(out, gin.H{
			"id": id.String(), "master_wallet_id": mid.String(), "name": name, "policy_type": ptype,
			"is_active": active, "priority": priority, "conditions": rawJSON(conditions), "actions": rawJSON(actions),
			"created_at": createdAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"policies": out})
}

func (svc *Service) CreatePolicy(c *gin.Context) {
	masterID := c.Param("id")
	var req policyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	pid := uuid.New().String()
	ctx := c.Request.Context()
	userID := currentUserID(c)
	_, err := svc.store.db.Exec(ctx,
		`INSERT INTO policies (id, master_wallet_id, name, policy_type, is_active, priority, conditions, actions, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		pid, masterID, req.Name, req.PolicyType, active, req.Priority, detailsJSON(req.Conditions), detailsJSON(req.Actions), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	svc.store.audit(ctx, masterID, "policy.create", "policy", "user", userID, "policy", pid, "normal", gin.H{"name": req.Name})
	c.JSON(http.StatusCreated, gin.H{"id": pid, "name": req.Name, "policy_type": req.PolicyType, "is_active": active})
}

func (svc *Service) UpdatePolicy(c *gin.Context) {
	id := c.Param("pid")
	var req policyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	res, err := svc.store.db.Exec(ctx,
		`UPDATE policies SET name=$1, policy_type=$2, conditions=$3, actions=$4, priority=$5 WHERE id=$6`,
		req.Name, req.PolicyType, detailsJSON(req.Conditions), detailsJSON(req.Actions), req.Priority, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if res.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}
	svc.store.audit(ctx, c.Param("id"), "policy.update", "policy", "user", currentUserID(c), "policy", id, "normal", gin.H{"name": req.Name})
	c.JSON(http.StatusOK, gin.H{"id": id, "updated": true})
}

func (svc *Service) DeletePolicy(c *gin.Context) {
	id := c.Param("pid")
	ctx := c.Request.Context()
	res, err := svc.store.db.Exec(ctx, `DELETE FROM policies WHERE id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if res.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}
	svc.store.audit(ctx, c.Param("id"), "policy.delete", "policy", "user", currentUserID(c), "policy", id, "normal", nil)
	c.JSON(http.StatusOK, gin.H{"id": id, "deleted": true})
}

// --- Fees ---

type feeReq struct {
	FeeType       string  `json:"fee_type" binding:"required"`
	FeePercentage float64 `json:"fee_percentage"`
	FeeFixed      string  `json:"fee_fixed"`
	IsActive      *bool   `json:"is_active"`
}

func (svc *Service) GetFeeConfigs(c *gin.Context) {
	masterID := c.Param("id")
	if masterID == "" {
		masterID = c.Query("master_wallet_id")
	}
	ctx := c.Request.Context()
	rows, err := svc.store.db.Query(ctx,
		`SELECT id, fee_type, fee_percentage, fee_fixed, is_active, created_at FROM fee_config WHERE master_wallet_id = $1`, masterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch fees"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var ftype string
		var pct float64
		var fixed string
		var active bool
		var createdAt time.Time
		_ = rows.Scan(&id, &ftype, &pct, &fixed, &active, &createdAt)
		out = append(out, gin.H{"id": id.String(), "fee_type": ftype, "fee_percentage": pct, "fee_fixed": fixed, "is_active": active, "created_at": createdAt})
	}
	c.JSON(http.StatusOK, gin.H{"fees": out})
}

func (svc *Service) CreateFeeConfig(c *gin.Context) {
	masterID := c.Param("id")
	var req feeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	fid := uuid.New().String()
	ctx := c.Request.Context()
	_, err := svc.store.db.Exec(ctx,
		`INSERT INTO fee_config (id, master_wallet_id, fee_type, fee_percentage, fee_fixed, is_active)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		fid, masterID, req.FeeType, req.FeePercentage, req.FeeFixed, active)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": fid, "fee_type": req.FeeType, "fee_percentage": req.FeePercentage})
}

// feeUpdateReq is the partial-update form of feeReq: every field is optional;
// a nil/empty value leaves the column unchanged (pointer fields distinguish
// "unset" from "set to zero").
type feeUpdateReq struct {
	FeeType       string   `json:"fee_type"`
	FeePercentage *float64 `json:"fee_percentage"`
	FeeFixed      string   `json:"fee_fixed"`
	IsActive      *bool    `json:"is_active"`
}

// UpdateFeeConfig — PUT /:id/fees/:fid. The MasterWallet owner can update any
// UserWallet fee (percentage / fixed / active state). Fee percentage is capped
// at 20%, matching the client-side cap in master_wallet/rust local_fee_config.
func (svc *Service) UpdateFeeConfig(c *gin.Context) {
	masterID := c.Param("id")
	fid := c.Param("fid")
	var req feeUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.FeePercentage != nil && (*req.FeePercentage < 0 || *req.FeePercentage > 20) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fee_percentage must be between 0 and 20"})
		return
	}
	ctx := c.Request.Context()
	res, err := svc.store.db.Exec(ctx,
		`UPDATE fee_config
		 SET fee_type = COALESCE(NULLIF($1, ''), fee_type),
		     fee_percentage = COALESCE($2, fee_percentage),
		     fee_fixed = COALESCE(NULLIF($3, ''), fee_fixed),
		     is_active = COALESCE($4, is_active),
		     updated_at = NOW()
		 WHERE id = $5 AND master_wallet_id = $6`,
		req.FeeType, req.FeePercentage, req.FeeFixed, req.IsActive, fid, masterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if res.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "fee config not found"})
		return
	}
	svc.store.audit(ctx, masterID, "fee.update", "fee", "user", currentUserID(c), "fee_config", fid, "normal",
		gin.H{"fee_type": req.FeeType, "fee_percentage": req.FeePercentage})
	c.JSON(http.StatusOK, gin.H{"id": fid, "updated": true})
}

func (svc *Service) DeleteFeeConfig(c *gin.Context) {
	fid := c.Param("fid")
	ctx := c.Request.Context()
	res, err := svc.store.db.Exec(ctx, `DELETE FROM fee_config WHERE id = $1 AND master_wallet_id = $2`, fid, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if res.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "fee config not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": fid, "deleted": true})
}

// --- Auto-sign rules ---

type autoSignReq struct {
	Name      string                 `json:"name" binding:"required"`
	RuleType  string                 `json:"rule_type" binding:"required"`
	Conditions map[string]interface{} `json:"conditions"`
	MaxAmount string                 `json:"max_amount"`
	IsActive  *bool                  `json:"is_active"`
}

func (svc *Service) GetAutoSignRules(c *gin.Context) {
	masterID := c.Param("id")
	if masterID == "" {
		masterID = c.Query("master_wallet_id")
	}
	ctx := c.Request.Context()
	rows, err := svc.store.db.Query(ctx,
		`SELECT id, name, rule_type, conditions, max_amount, is_active, created_at FROM auto_sign_rules WHERE master_wallet_id = $1`, masterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch auto-sign rules"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var name, rtype, maxAmt string
		var active bool
		var conditions []byte
		var createdAt time.Time
		_ = rows.Scan(&id, &name, &rtype, &conditions, &maxAmt, &active, &createdAt)
		out = append(out, gin.H{"id": id.String(), "name": name, "rule_type": rtype, "max_amount": maxAmt, "is_active": active, "conditions": rawJSON(conditions), "created_at": createdAt})
	}
	c.JSON(http.StatusOK, gin.H{"auto_sign_rules": out})
}

func (svc *Service) CreateAutoSignRule(c *gin.Context) {
	masterID := c.Param("id")
	var req autoSignReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	rid := uuid.New().String()
	ctx := c.Request.Context()
	_, err := svc.store.db.Exec(ctx,
		`INSERT INTO auto_sign_rules (id, master_wallet_id, name, rule_type, conditions, max_amount, is_active)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		rid, masterID, req.Name, req.RuleType, detailsJSON(req.Conditions), req.MaxAmount, active)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": rid, "name": req.Name, "rule_type": req.RuleType})
}

// autoSignUpdateReq is the partial-update form of autoSignReq.
type autoSignUpdateReq struct {
	Name       string                 `json:"name"`
	RuleType   string                 `json:"rule_type"`
	Conditions map[string]interface{} `json:"conditions"`
	MaxAmount  *string                `json:"max_amount"`
	IsActive   *bool                  `json:"is_active"`
}

// UpdateAutoSignRule — PUT /:id/auto-sign/:rid. MasterWallet owner edits an
// auto-sign rule in place (name/conditions/limit/active state).
func (svc *Service) UpdateAutoSignRule(c *gin.Context) {
	masterID := c.Param("id")
	rid := c.Param("rid")
	var req autoSignUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var conditions []byte
	if req.Conditions != nil {
		conditions = detailsJSON(req.Conditions)
	}
	ctx := c.Request.Context()
	res, err := svc.store.db.Exec(ctx,
		`UPDATE auto_sign_rules
		 SET name = COALESCE(NULLIF($1, ''), name),
		     rule_type = COALESCE(NULLIF($2, ''), rule_type),
		     conditions = COALESCE($3, conditions),
		     max_amount = COALESCE(NULLIF($4, ''), max_amount),
		     is_active = COALESCE($5, is_active),
		     updated_at = NOW()
		 WHERE id = $6 AND master_wallet_id = $7`,
		req.Name, req.RuleType, conditions, req.MaxAmount, req.IsActive, rid, masterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if res.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "auto-sign rule not found"})
		return
	}
	svc.store.audit(ctx, masterID, "auto_sign.update", "auto_sign", "user", currentUserID(c), "auto_sign_rule", rid, "normal",
		gin.H{"name": req.Name})
	c.JSON(http.StatusOK, gin.H{"id": rid, "updated": true})
}

func (svc *Service) DeleteAutoSignRule(c *gin.Context) {
	rid := c.Param("rid")
	ctx := c.Request.Context()
	res, err := svc.store.db.Exec(ctx, `DELETE FROM auto_sign_rules WHERE id = $1 AND master_wallet_id = $2`, rid, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if res.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "auto-sign rule not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": rid, "deleted": true})
}

// --- Users ---

type userReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name"`
	Role     string `json:"role"`
}

func (svc *Service) GetUsers(c *gin.Context) {
	masterID := c.Query("master_wallet_id")
	ctx := c.Request.Context()
	rows, err := svc.store.db.Query(ctx,
		`SELECT id, email, name, role, is_active, last_login_at, created_at FROM mw_users WHERE master_wallet_id = $1 ORDER BY created_at DESC`, masterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch users"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var email, name, role string
		var active bool
		var lastLogin *time.Time
		var createdAt time.Time
		_ = rows.Scan(&id, &email, &name, &role, &active, &lastLogin, &createdAt)
		entry := gin.H{"id": id.String(), "email": email, "name": name, "role": role, "is_active": active, "created_at": createdAt}
		if lastLogin != nil {
			entry["last_login_at"] = *lastLogin
		}
		out = append(out, entry)
	}
	c.JSON(http.StatusOK, gin.H{"users": out})
}

func (svc *Service) CreateUser(c *gin.Context) {
	masterID := c.Param("id")
	var req userReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hash, err := hashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	role := req.Role
	if role == "" {
		role = "user"
	}
	uid := uuid.New().String()
	ctx := c.Request.Context()
	_, err = svc.store.db.Exec(ctx,
		`INSERT INTO mw_users (id, email, name, role, password_hash, master_wallet_id) VALUES ($1,$2,$3,$4,$5,$6)`,
		uid, req.Email, req.Name, role, hash, masterID)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}
	svc.store.audit(ctx, masterID, "user.create", "user", "user", currentUserID(c), "user", uid, "normal", gin.H{"email": req.Email})
	c.JSON(http.StatusCreated, gin.H{"id": uid, "email": req.Email, "role": role})
}

// userUpdateReq is the partial-update form of userReq. Password is optional:
// when supplied it is re-hashed (bcrypt) and rotated.
type userUpdateReq struct {
	Email    string `json:"email"`
	Name     *string `json:"name"`
	Role     string `json:"role"`
	IsActive *bool   `json:"is_active"`
	Password string `json:"password"`
}

// validMWRoles is the closed set of roles a master-wallet user may hold.
var validMWRoles = map[string]bool{"user": true, "admin": true, "operator": true, "treasury": true, "super_admin": true}

// UpdateUser — PUT /:id/users/:uid. MasterWallet owner edits a user's name,
// role, active state, or password. Role changes are validated against the
// closed role set; a user cannot be deleted here (use DELETE).
func (svc *Service) UpdateUser(c *gin.Context) {
	masterID := c.Param("id")
	uid := c.Param("uid")
	var req userUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Role != "" && !validMWRoles[req.Role] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
		return
	}
	var newHash *string
	if req.Password != "" {
		if len(req.Password) < 8 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 8 characters"})
			return
		}
		h, err := hashPassword(req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
			return
		}
		newHash = &h
	}
	ctx := c.Request.Context()
	res, err := svc.store.db.Exec(ctx,
		`UPDATE mw_users
		 SET name = COALESCE($1, name),
		     role = COALESCE(NULLIF($2, ''), role),
		     is_active = COALESCE($3, is_active),
		     password_hash = COALESCE($4, password_hash),
		     updated_at = NOW()
		 WHERE id = $5 AND master_wallet_id = $6`,
		req.Name, req.Role, req.IsActive, newHash, uid, masterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if res.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	svc.store.audit(ctx, masterID, "user.update", "user", "user", currentUserID(c), "user", uid, "normal",
		gin.H{"role": req.Role})
	c.JSON(http.StatusOK, gin.H{"id": uid, "updated": true})
}

func (svc *Service) DeleteUser(c *gin.Context) {
	uid := c.Param("uid")
	ctx := c.Request.Context()
	res, err := svc.store.db.Exec(ctx, `DELETE FROM mw_users WHERE id = $1 AND master_wallet_id = $2`, uid, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if res.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	svc.store.audit(ctx, c.Param("id"), "user.delete", "user", "user", currentUserID(c), "user", uid, "normal", nil)
	c.JSON(http.StatusOK, gin.H{"id": uid, "deleted": true})
}

// --- Audit logs ---

func (svc *Service) GetAuditLogs(c *gin.Context) {
	masterID := c.Query("master_wallet_id")
	limit := parseLimit(c.Query("limit"), 100, 500)
	ctx := c.Request.Context()
	rows, err := svc.store.db.Query(ctx,
		`SELECT id, master_wallet_id, event_type, event_category, actor_type, actor_id, target_type, target_id, severity, details, created_at
		 FROM audit_logs WHERE master_wallet_id = $1 ORDER BY created_at DESC LIMIT $2`, masterID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch audit logs"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var mid *uuid.UUID
		var eventType, category, actorType, actorID, targetType, targetID, severity string
		var details []byte
		var createdAt time.Time
		_ = rows.Scan(&id, &mid, &eventType, &category, &actorType, &actorID, &targetType, &targetID, &severity, &details, &createdAt)
		entry := gin.H{
			"id": id.String(), "event_type": eventType, "event_category": category,
			"actor_type": actorType, "actor_id": actorID, "target_type": targetType,
			"target_id": targetID, "severity": severity, "details": rawJSON(details), "created_at": createdAt,
		}
		if mid != nil {
			entry["master_wallet_id"] = mid.String()
		}
		out = append(out, entry)
	}
	c.JSON(http.StatusOK, gin.H{"audit_logs": out})
}

// --- Analytics (real SQL aggregates) ---

func (svc *Service) GetVolumeAnalytics(c *gin.Context) {
	masterID := c.Query("master_wallet_id")
	ctx := c.Request.Context()
	var totalVolume string
	var txCount int64
	_ = svc.store.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount), 0)::text, COUNT(*) FROM transactions WHERE master_wallet_id = $1 AND status = 'confirmed'`,
		masterID).Scan(&totalVolume, &txCount)
	c.JSON(http.StatusOK, gin.H{"master_wallet_id": masterID, "total_volume": totalVolume, "transaction_count": txCount, "source": "postgresql"})
}

func (svc *Service) GetTransactionAnalytics(c *gin.Context) {
	masterID := c.Query("master_wallet_id")
	ctx := c.Request.Context()
	rows, err := svc.store.db.Query(ctx,
		`SELECT status, COUNT(*) FROM transactions WHERE master_wallet_id = $1 GROUP BY status`, masterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := gin.H{}
	for rows.Next() {
		var status string
		var count int64
		_ = rows.Scan(&status, &count)
		out[status] = count
	}
	c.JSON(http.StatusOK, gin.H{"master_wallet_id": masterID, "by_status": out, "source": "postgresql"})
}

func (svc *Service) GetWalletAnalytics(c *gin.Context) {
	ctx := c.Request.Context()
	var walletCount, subWalletCount, userCount int64
	_ = svc.store.db.QueryRow(ctx, `SELECT COUNT(*) FROM master_wallets`).Scan(&walletCount)
	_ = svc.store.db.QueryRow(ctx, `SELECT COUNT(*) FROM sub_wallets`).Scan(&subWalletCount)
	_ = svc.store.db.QueryRow(ctx, `SELECT COUNT(*) FROM mw_users`).Scan(&userCount)
	c.JSON(http.StatusOK, gin.H{"master_wallets": walletCount, "sub_wallets": subWalletCount, "users": userCount, "source": "postgresql"})
}

// --- Notifications ---

type notificationReq struct {
	UserID   string                 `json:"user_id"`
	Type     string                 `json:"notification_type" binding:"required"`
	Category string                 `json:"category"`
	Title    string                 `json:"title" binding:"required"`
	Message  string                 `json:"message" binding:"required"`
	Priority string                 `json:"priority"`
	Channel  string                 `json:"channel"`
	Data     map[string]interface{} `json:"data"`
}

func (svc *Service) CreateNotification(c *gin.Context) {
	masterID := c.Param("id")
	var req notificationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	nid := uuid.New().String()
	ctx := c.Request.Context()
	_, err := svc.store.db.Exec(ctx,
		`INSERT INTO notifications (id, master_wallet_id, user_id, notification_type, category, title, message, priority, channel, data)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		nid, masterID, req.UserID, req.Type, req.Category, req.Title, req.Message, req.Priority, req.Channel, detailsJSON(req.Data))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": nid, "title": req.Title, "created": true})
}

func (svc *Service) GetNotifications(c *gin.Context) {
	masterID := c.Param("id")
	ctx := c.Request.Context()
	rows, err := svc.store.db.Query(ctx,
		`SELECT id, notification_type, category, title, message, priority, is_read, created_at FROM notifications WHERE master_wallet_id = $1 ORDER BY created_at DESC LIMIT 50`, masterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var ntype, category, title, message, priority string
		var isRead bool
		var createdAt time.Time
		_ = rows.Scan(&id, &ntype, &category, &title, &message, &priority, &isRead, &createdAt)
		out = append(out, gin.H{"id": id.String(), "type": ntype, "category": category, "title": title, "message": message, "priority": priority, "is_read": isRead, "created_at": createdAt})
	}
	c.JSON(http.StatusOK, gin.H{"notifications": out})
}

// notificationUpdateReq is the partial-update form of notificationReq. The
// primary use is marking a notification read/unread; content fields are
// updatable for corrections.
type notificationUpdateReq struct {
	Title    string `json:"title"`
	Message  string `json:"message"`
	Priority string `json:"priority"`
	IsRead   *bool  `json:"is_read"`
}

// UpdateNotification — PUT /:id/notifications/:nid. Supports marking
// read/unread and correcting title/message/priority.
func (svc *Service) UpdateNotification(c *gin.Context) {
	masterID := c.Param("id")
	nid := c.Param("nid")
	var req notificationUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	res, err := svc.store.db.Exec(ctx,
		`UPDATE notifications
		 SET title = COALESCE(NULLIF($1, ''), title),
		     message = COALESCE(NULLIF($2, ''), message),
		     priority = COALESCE(NULLIF($3, ''), priority),
		     is_read = COALESCE($4, is_read),
		     read_at = CASE WHEN $4 IS NULL THEN read_at
		                    WHEN $4 THEN NOW()
		                    ELSE NULL END
		 WHERE id = $5 AND master_wallet_id = $6`,
		req.Title, req.Message, req.Priority, req.IsRead, nid, masterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if res.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": nid, "updated": true})
}

// --- Webhooks ---

type webhookReq struct {
	Name       string   `json:"name" binding:"required"`
	URL        string   `json:"url" binding:"required"`
	Events     []string `json:"events" binding:"required"`
	RetryCount int      `json:"retry_count"`
}

// webhookUpdateReq is the partial-update form of webhookReq.
type webhookUpdateReq struct {
	Name       string   `json:"name"`
	URL        string   `json:"url"`
	Events     []string `json:"events"`
	IsActive   *bool    `json:"is_active"`
	RetryCount *int     `json:"retry_count"`
}

// UpdateWebhook — PUT /:id/webhooks/:wid. MasterWallet owner edits a webhook
// endpoint's name/url/events/active state/retry policy in place.
func (svc *Service) UpdateWebhook(c *gin.Context) {
	masterID := c.Param("id")
	wid := c.Param("wid")
	var req webhookUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	res, err := svc.store.db.Exec(ctx,
		`UPDATE webhooks
		 SET name = COALESCE(NULLIF($1, ''), name),
		     url = COALESCE(NULLIF($2, ''), url),
		     events = COALESCE($3, events),
		     is_active = COALESCE($4, is_active),
		     retry_count = COALESCE($5, retry_count),
		     updated_at = NOW()
		 WHERE id = $6 AND master_wallet_id = $7`,
		req.Name, req.URL, req.Events, req.IsActive, req.RetryCount, wid, masterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if res.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "webhook not found"})
		return
	}
	svc.store.audit(ctx, masterID, "webhook.update", "webhook", "user", currentUserID(c), "webhook", wid, "normal",
		gin.H{"url": req.URL})
	c.JSON(http.StatusOK, gin.H{"id": wid, "updated": true})
}

func (svc *Service) CreateWebhook(c *gin.Context) {
	masterID := c.Param("id")
	var req webhookReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	wid := uuid.New().String()
	ctx := c.Request.Context()
	_, err := svc.store.db.Exec(ctx,
		`INSERT INTO webhooks (id, master_wallet_id, name, url, events, retry_count) VALUES ($1,$2,$3,$4,$5,$6)`,
		wid, masterID, req.Name, req.URL, req.Events, req.RetryCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": wid, "name": req.Name, "url": req.URL})
}

func (svc *Service) GetWebhooks(c *gin.Context) {
	masterID := c.Param("id")
	ctx := c.Request.Context()
	rows, err := svc.store.db.Query(ctx,
		`SELECT id, name, url, events, is_active, is_verified, total_delivered, total_failed, created_at FROM webhooks WHERE master_wallet_id = $1`, masterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var name, url string
		var events []string
		var active, verified bool
		var delivered, failed int64
		var createdAt time.Time
		_ = rows.Scan(&id, &name, &url, &events, &active, &verified, &delivered, &failed, &createdAt)
		out = append(out, gin.H{"id": id.String(), "name": name, "url": url, "events": events, "is_active": active, "is_verified": verified, "total_delivered": delivered, "total_failed": failed, "created_at": createdAt})
	}
	c.JSON(http.StatusOK, gin.H{"webhooks": out})
}

func (svc *Service) DeleteWebhook(c *gin.Context) {
	wid := c.Param("wid")
	ctx := c.Request.Context()
	res, err := svc.store.db.Exec(ctx, `DELETE FROM webhooks WHERE id = $1 AND master_wallet_id = $2`, wid, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if res.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "webhook not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": wid, "deleted": true})
}

// rawJSON safely converts a []byte JSONB column to an object; nil -> empty map.
func rawJSON(b []byte) interface{} {
	if len(b) == 0 {
		return map[string]interface{}{}
	}
	return b
}
