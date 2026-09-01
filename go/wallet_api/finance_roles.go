package main

// finance_roles.go — superadmin-defined dynamic admin roles.
//
// The superadmin (users.role = 'superadmin' or 'admin') can create/delete
// roles with a permission set and grant/revoke them to any user. Finance
// admin endpoints check permissions dynamically through granted roles, so
// access can be delegated (e.g. a "withdrawal-officer" role with only
// withdrawals.sign) without handing out full admin. Every mutation is
// audit-logged.
//
// Permission catalog (closed set — unknown permissions are rejected):
//   withdrawals.sign  — approve/reject queued withdrawals
//   rates.manage      — manage convert-engine rates
//   switches.manage   — per-token deposit/withdraw/P2P/convert switches
//   p2p.resolve       — resolve escrow disputes
//   roles.manage      — create/delete roles, grant/revoke
//   audit.read        — read the finance audit log

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var financePermissionCatalog = map[string]bool{
	"withdrawals.sign": true,
	"rates.manage":     true,
	"switches.manage":  true,
	"p2p.resolve":      true,
	"roles.manage":     true,
	"audit.read":       true,
}

// requireFinancePerm authorizes a finance admin action. Full admins
// (superadmin / admin / wl_admin / master_wallet_admin) always pass; other
// users pass when any granted dynamic role contains the permission.
func requireFinancePerm(c *gin.Context, perm string) bool {
	role := getUserRole(c)
	switch role {
	case "superadmin", "admin", "wl_admin", "master_wallet_admin":
		return true
	}
	uid, err := uuid.Parse(getUserID(c))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return false
	}
	var count int
	if err := store.PG.QueryRow(c.Request.Context(),
		`SELECT COUNT(*) FROM admin_role_grant g
		 JOIN admin_role r ON r.name = g.role_name
		 WHERE g.user_id=$1 AND $2 = ANY(r.permissions)`, uid, perm).Scan(&count); err != nil || count == 0 {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "missing permission: " + perm})
		return false
	}
	return true
}

// requireRolesManage gates role administration itself.
func requireRolesManage(c *gin.Context) bool {
	return requireFinancePerm(c, "roles.manage")
}

// handleAdminListRoles lists all dynamic roles with their grants.
func handleAdminListRoles(c *gin.Context) {
	if !requireRolesManage(c) {
		return
	}
	rows, err := store.PG.Query(c.Request.Context(),
		`SELECT r.name, r.permissions, r.created_at,
		        COALESCE((SELECT COUNT(*) FROM admin_role_grant g WHERE g.role_name=r.name),0) AS grants
		 FROM admin_role r ORDER BY r.name`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var name string
		var perms []string
		var created time.Time
		var grants int
		if err := rows.Scan(&name, &perms, &created, &grants); err != nil {
			continue
		}
		out = append(out, gin.H{"name": name, "permissions": perms, "grants": grants, "created_at": created})
	}
	c.JSON(http.StatusOK, gin.H{"roles": out, "count": len(out), "permission_catalog": financePermissionCatalog})
}

// handleAdminCreateRole creates a role with a validated permission set.
func handleAdminCreateRole(c *gin.Context) {
	if !requireRolesManage(c) {
		return
	}
	adminID, _ := uuid.Parse(getUserID(c))
	var req struct {
		Name        string   `json:"name" binding:"required"`
		Permissions []string `json:"permissions" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Name = strings.ToLower(strings.TrimSpace(req.Name))
	if len(req.Name) < 3 || len(req.Name) > 64 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role name must be 3-64 characters"})
		return
	}
	for _, p := range req.Permissions {
		if !financePermissionCatalog[p] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unknown permission: " + p})
			return
		}
	}
	if _, err := store.PG.Exec(c.Request.Context(),
		`INSERT INTO admin_role(name, permissions, created_by) VALUES($1,$2,$3)`,
		req.Name, req.Permissions, adminID); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "role already exists or invalid"})
		return
	}
	auditFinance(c.Request.Context(), adminID, getUserRole(c), "role.create", req.Name,
		gin.H{"permissions": req.Permissions})
	c.JSON(http.StatusOK, gin.H{"status": "role_created", "name": req.Name, "permissions": req.Permissions})
}

// handleAdminDeleteRole deletes a role (grants cascade).
func handleAdminDeleteRole(c *gin.Context) {
	if !requireRolesManage(c) {
		return
	}
	adminID, _ := uuid.Parse(getUserID(c))
	name := c.Param("name")
	tag, err := store.PG.Exec(c.Request.Context(), `DELETE FROM admin_role WHERE name=$1`, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}
	auditFinance(c.Request.Context(), adminID, getUserRole(c), "role.delete", name, nil)
	c.JSON(http.StatusOK, gin.H{"status": "role_deleted", "name": name})
}

// handleAdminGrantRole grants a role to a user.
func handleAdminGrantRole(c *gin.Context) {
	if !requireRolesManage(c) {
		return
	}
	adminID, _ := uuid.Parse(getUserID(c))
	var req struct {
		UserID string `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	target, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}
	name := c.Param("name")
	if _, err := store.PG.Exec(c.Request.Context(),
		`INSERT INTO admin_role_grant(user_id, role_name, granted_by) VALUES($1,$2,$3)
		 ON CONFLICT (user_id, role_name) DO NOTHING`, target, name, adminID); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "grant failed (unknown role or user)"})
		return
	}
	auditFinance(c.Request.Context(), adminID, getUserRole(c), "role.grant", name,
		gin.H{"user_id": req.UserID})
	c.JSON(http.StatusOK, gin.H{"status": "role_granted", "role": name, "user_id": req.UserID})
}

// handleAdminRevokeRole revokes a role from a user.
func handleAdminRevokeRole(c *gin.Context) {
	if !requireRolesManage(c) {
		return
	}
	adminID, _ := uuid.Parse(getUserID(c))
	var req struct {
		UserID string `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	target, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}
	name := c.Param("name")
	tag, err := store.PG.Exec(c.Request.Context(),
		`DELETE FROM admin_role_grant WHERE user_id=$1 AND role_name=$2`, target, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "revoke failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "grant not found"})
		return
	}
	auditFinance(c.Request.Context(), adminID, getUserRole(c), "role.revoke", name,
		gin.H{"user_id": req.UserID})
	c.JSON(http.StatusOK, gin.H{"status": "role_revoked", "role": name, "user_id": req.UserID})
}

// handleAdminAuditLog reads the finance audit log (audit.read permission).
func handleAdminAuditLog(c *gin.Context) {
	if !requireFinancePerm(c, "audit.read") {
		return
	}
	rows, err := store.PG.Query(c.Request.Context(),
		`SELECT id, COALESCE(actor_user_id::text,''), actor_role, action, COALESCE(target,''),
		        COALESCE(detail::text,'null'), created_at
		 FROM finance_audit_log ORDER BY id DESC LIMIT 500`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id int64
		var actor, role, action, target, detail string
		var created time.Time
		if err := rows.Scan(&id, &actor, &role, &action, &target, &detail, &created); err != nil {
			continue
		}
		out = append(out, gin.H{
			"id": id, "actor_user_id": actor, "actor_role": role, "action": action,
			"target": target, "detail": detail, "created_at": created,
		})
	}
	c.JSON(http.StatusOK, gin.H{"audit": out, "count": len(out)})
}

// ---------------------------------------------------------------------------
// Per-token switch administration (switches.manage permission)
// ---------------------------------------------------------------------------

// handleAdminSetSwitch upserts the feature switches for one token.
func handleAdminSetSwitch(c *gin.Context) {
	if !requireFinancePerm(c, "switches.manage") {
		return
	}
	adminID, _ := uuid.Parse(getUserID(c))
	currency := strings.ToUpper(c.Param("currency"))
	var req struct {
		DepositEnabled  bool `json:"deposit_enabled"`
		WithdrawEnabled bool `json:"withdraw_enabled"`
		P2PEnabled      bool `json:"p2p_enabled"`
		ConvertEnabled  bool `json:"convert_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := store.PG.Exec(c.Request.Context(),
		`INSERT INTO token_switch(currency, deposit_enabled, withdraw_enabled, p2p_enabled, convert_enabled, updated_by, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,now())
		 ON CONFLICT (currency) DO UPDATE SET
		   deposit_enabled=$2, withdraw_enabled=$3, p2p_enabled=$4, convert_enabled=$5,
		   updated_by=$6, updated_at=now()`,
		currency, req.DepositEnabled, req.WithdrawEnabled, req.P2PEnabled, req.ConvertEnabled, adminID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "switch update failed"})
		return
	}
	auditFinance(c.Request.Context(), adminID, getUserRole(c), "switch.set", currency, req)
	c.JSON(http.StatusOK, gin.H{"status": "switch_set", "currency": currency})
}
