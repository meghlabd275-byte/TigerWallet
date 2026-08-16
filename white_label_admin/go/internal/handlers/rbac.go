package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tigerwallet/white-label-admin/internal/middleware"
	"github.com/tigerwallet/white-label-admin/internal/roles"
)

// ==================== Structured RBAC (wl_client scope) ====================
//
// Complements the existing scope system (roles.go + RequireScope): an
// admin_role bundles a set of whitelisted scope strings (roles.IsValid).
// Assigning a role to an admin MERGES the role's scopes into that admin's
// admin_users.scopes column, so the JWT issued at the admin's next login
// carries the role's scopes and RequireScope keeps working unchanged. Revoking
// a role removes its scopes from the admin (only those not granted by another
// active role). admins/:id/permissions returns the effective scope set
// (direct scopes + scopes from every active assigned role).
//
// This deliberately reuses the existing roles whitelist rather than inventing
// a parallel permission namespace, so there is one authorization source of
// truth (the scopes in the JWT) and RequireScope is NOT bypassed.

// ---- admin_roles CRUD ----

func (s *Svc) ListAdminRoles(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	ctx := c.Request.Context()
	rows, err := s.db.Query(ctx,
		`SELECT id, name, COALESCE(description,''), permissions, is_system, is_active, created_at, updated_at
		 FROM admin_roles WHERE white_label_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var name, desc string
		var perms []string
		var isSystem, isActive bool
		var created, updated time.Time
		_ = rows.Scan(&id, &name, &desc, &perms, &isSystem, &isActive, &created, &updated)
		out = append(out, gin.H{"id": id, "name": name, "description": desc, "permissions": perms,
			"is_system": isSystem, "is_active": isActive, "created_at": created, "updated_at": updated})
	}
	c.JSON(http.StatusOK, gin.H{"roles": out})
}

func (s *Svc) GetAdminRole(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	var name, desc string
	var perms []string
	var isSystem, isActive bool
	var created, updated time.Time
	err = s.db.QueryRow(ctx,
		`SELECT name, COALESCE(description,''), permissions, is_system, is_active, created_at, updated_at
		 FROM admin_roles WHERE id=$1 AND white_label_id=$2`, id, tenantID).
		Scan(&name, &desc, &perms, &isSystem, &isActive, &created, &updated)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "name": name, "description": desc, "permissions": perms,
		"is_system": isSystem, "is_active": isActive, "created_at": created, "updated_at": updated})
}

func (s *Svc) CreateAdminRole(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	var req struct {
		Name        string   `json:"name" binding:"required"`
		Description string   `json:"description"`
		Permissions []string `json:"permissions"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !s.validateScopes(req.Permissions) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "permissions must be valid scopes (see /api/v1/scopes)"})
		return
	}
	id := uuid.New()
	ctx := c.Request.Context()
	_, err := s.db.Exec(ctx,
		`INSERT INTO admin_roles (id, name, description, permissions, is_system, is_active, white_label_id)
		 VALUES ($1,$2,$3,$4,false,true,$5)`,
		id, req.Name, req.Description, req.Permissions, tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "admin_role.create", "admin_role", id.String(), gin.H{"name": req.Name, "permissions": req.Permissions})
	c.JSON(http.StatusCreated, gin.H{"id": id, "name": req.Name, "permissions": req.Permissions})
}

func (s *Svc) UpdateAdminRole(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Permissions []string `json:"permissions"`
		IsActive    *bool    `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Permissions != nil && !s.validateScopes(req.Permissions) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "permissions must be valid scopes"})
		return
	}
	ctx := c.Request.Context()
	if req.Name != "" {
		_, _ = s.db.Exec(ctx, `UPDATE admin_roles SET name=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3 AND is_system=false`, req.Name, id, tenantID)
	}
	if req.Description != "" {
		_, _ = s.db.Exec(ctx, `UPDATE admin_roles SET description=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3 AND is_system=false`, req.Description, id, tenantID)
	}
	if req.Permissions != nil {
		_, _ = s.db.Exec(ctx, `UPDATE admin_roles SET permissions=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3 AND is_system=false`, req.Permissions, id, tenantID)
	}
	if req.IsActive != nil {
		// System roles cannot be deleted but may be toggled active.
		_, err = s.db.Exec(ctx, `UPDATE admin_roles SET is_active=$1, updated_at=NOW() WHERE id=$2 AND white_label_id=$3`, *req.IsActive, id, tenantID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	s.audit(ctx, middleware.AdminID(c), "admin_role.update", "admin_role", id.String(), gin.H{"permissions": req.Permissions})
	c.JSON(http.StatusOK, gin.H{"updated": id})
}

func (s *Svc) DeleteAdminRole(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	ct, err := s.db.Exec(ctx, `DELETE FROM admin_roles WHERE id=$1 AND white_label_id=$2 AND is_system=false`, id, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found or is system role"})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "admin_role.delete", "admin_role", id.String(), nil)
	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

// ---- admin_permissions CRUD ----

func (s *Svc) ListAdminPermissions(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	ctx := c.Request.Context()
	rows, err := s.db.Query(ctx,
		`SELECT id, name, COALESCE(description,''), category, is_active, created_at
		 FROM admin_permissions WHERE white_label_id=$1 ORDER BY category, name`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var name, desc, category string
		var active bool
		var created time.Time
		_ = rows.Scan(&id, &name, &desc, &category, &active, &created)
		out = append(out, gin.H{"id": id, "name": name, "description": desc, "category": category, "is_active": active, "created_at": created})
	}
	c.JSON(http.StatusOK, gin.H{"permissions": out})
}

func (s *Svc) CreateAdminPermission(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Category    string `json:"category"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Category == "" {
		req.Category = "general"
	}
	id := uuid.New()
	ctx := c.Request.Context()
	_, err := s.db.Exec(ctx,
		`INSERT INTO admin_permissions (id, name, description, category, is_active, white_label_id)
		 VALUES ($1,$2,$3,$4,true,$5)`,
		id, req.Name, req.Description, req.Category, tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "admin_permission.create", "admin_permission", id.String(), gin.H{"name": req.Name})
	c.JSON(http.StatusCreated, gin.H{"id": id, "name": req.Name, "category": req.Category})
}

// ---- admins/:id/role assign + admins/:id/permissions GET ----

// AssignAdminRole assigns a role to an admin and merges the role's scopes into
// the admin's admin_users.scopes so the next-issued JWT (RequireScope) honors
// them. Tenant-isolated: both the admin and the role must belong to the
// caller's white_label_id.
func (s *Svc) AssignAdminRole(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	adminID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid admin id"})
		return
	}
	var req struct {
		RoleID string `json:"role_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	roleID, err := uuid.Parse(req.RoleID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role id"})
		return
	}
	ctx := c.Request.Context()
	// Verify both the admin and the role are in the caller's tenancy and the
	// role is active; load the role's scopes to merge.
	var roleWL uuid.UUID
	var rolePerms []string
	var roleActive bool
	err = s.db.QueryRow(ctx, `SELECT white_label_id, permissions, is_active FROM admin_roles WHERE id=$1`, roleID).
		Scan(&roleWL, &rolePerms, &roleActive)
	if err != nil || roleWL != tenantID || !roleActive {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found in your tenancy or inactive"})
		return
	}
	var adminWL uuid.UUID
	err = s.db.QueryRow(ctx, `SELECT white_label_id FROM admin_users WHERE id=$1`, adminID).Scan(&adminWL)
	if err != nil || adminWL != tenantID {
		c.JSON(http.StatusNotFound, gin.H{"error": "admin not found in your tenancy"})
		return
	}
	grantedBy := middleware.AdminID(c)
	_, err = s.db.Exec(ctx,
		`INSERT INTO admin_role_assignments (id, admin_id, role_id, granted_by) VALUES ($1,$2,$3,$4)
		 ON CONFLICT (admin_id, role_id) DO NOTHING`,
		uuid.New(), adminID, roleID, grantedBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Merge role scopes into admin_users.scopes (union, dedup) so the next JWT
	// carries them and RequireScope enforces them — no parallel auth path.
	if len(rolePerms) > 0 {
		_, err = s.db.Exec(ctx,
			`UPDATE admin_users SET scopes = ARRAY(SELECT DISTINCT unnest(scopes || $1::text[])) WHERE id=$2`,
			rolePerms, adminID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	s.audit(ctx, grantedBy, "admin_role.assign", "admin", adminID.String(), gin.H{"role_id": roleID, "scopes": rolePerms})
	c.JSON(http.StatusOK, gin.H{"assigned": adminID, "role_id": roleID, "merged_scopes": rolePerms})
}

// RevokeAdminRole removes a role assignment and prunes scopes that are no
// longer backed by any active role (preserving scopes still granted by another
// active role).
func (s *Svc) RevokeAdminRole(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	adminID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid admin id"})
		return
	}
	roleID, err := uuid.Parse(c.Param("roleId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role id"})
		return
	}
	ctx := c.Request.Context()
	// Tenant-isolate: confirm the admin belongs to the caller's tenancy.
	var adminWL uuid.UUID
	err = s.db.QueryRow(ctx, `SELECT white_label_id FROM admin_users WHERE id=$1`, adminID).Scan(&adminWL)
	if err != nil || adminWL != tenantID {
		c.JSON(http.StatusNotFound, gin.H{"error": "admin not found in your tenancy"})
		return
	}
	if _, err := s.db.Exec(ctx, `DELETE FROM admin_role_assignments WHERE admin_id=$1 AND role_id=$2`, adminID, roleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Recompute effective scopes = union of scopes across remaining active
	// roles assigned to this admin. Direct scopes (not via a role) are
	// preserved by reading the union of the current scopes minus the revoked
	// role's scopes plus scopes still backed by an active role. Simplest
	// correct recomputation: set scopes to (scopes still backed by an active
	// role) union (any scopes the admin had that are NOT in any role's
	// permission set, i.e. directly-assigned scopes).
	var backed []string
	bRows, qerr := s.db.Query(ctx,
		`SELECT DISTINCT unnest(r.permissions) FROM admin_role_assignments a
		 JOIN admin_roles r ON r.id = a.role_id
		 WHERE a.admin_id=$1 AND r.is_active=true AND r.white_label_id=$2`, adminID, tenantID)
	if qerr == nil {
		for bRows.Next() {
			var p string
			_ = bRows.Scan(&p)
			backed = append(backed, p)
		}
		bRows.Close()
	}
	// Direct scopes = admin's current scopes that are not present in any role's
	// permissions for this tenancy (so they can't have come from a role).
	_, err = s.db.Exec(ctx,
		`UPDATE admin_users SET scopes =
		   ARRAY(SELECT DISTINCT s FROM unnest(scopes) s
		         WHERE s NOT IN (SELECT unnest(permissions) FROM admin_roles WHERE white_label_id=$1)
		            OR s = ANY($2::text[]))
		 WHERE id=$3`,
		tenantID, backed, adminID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.audit(ctx, middleware.AdminID(c), "admin_role.revoke", "admin", adminID.String(), gin.H{"role_id": roleID})
	c.JSON(http.StatusOK, gin.H{"revoked": adminID, "role_id": roleID, "effective_scopes": backed})
}

// GetAdminPermissions returns the admin's effective scope set: the union of
// the admin's direct scopes (admin_users.scopes) and the scopes of every
// active role assigned to the admin. This is the authoritative answer to
// "what can this admin do?" — and because assigned-role scopes are merged into
// admin_users.scopes at assign time, it matches what RequireScope enforces via
// the JWT.
func (s *Svc) GetAdminPermissions(c *gin.Context) {
	tenantID := middleware.TenantID(c)
	adminID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid admin id"})
		return
	}
	ctx := c.Request.Context()
	var adminWL uuid.UUID
	err = s.db.QueryRow(ctx, `SELECT white_label_id FROM admin_users WHERE id=$1`, adminID).Scan(&adminWL)
	if err != nil || adminWL != tenantID {
		c.JSON(http.StatusNotFound, gin.H{"error": "admin not found in your tenancy"})
		return
	}
	var scopes []string
	err = s.db.QueryRow(ctx, `SELECT scopes FROM admin_users WHERE id=$1`, adminID).Scan(&scopes)
	if err != nil {
		scopes = []string{}
	}
	perms := map[string]bool{}
	for _, sc := range scopes {
		perms[sc] = true
	}
	// Include scopes from active assigned roles (defensive — they are already
	// merged into admin_users.scopes, but this guarantees parity even if the
	// merge was bypassed).
	rRows, qerr := s.db.Query(ctx,
		`SELECT DISTINCT unnest(r.permissions) FROM admin_role_assignments a
		 JOIN admin_roles r ON r.id = a.role_id
		 WHERE a.admin_id=$1 AND r.is_active=true AND r.white_label_id=$2`, adminID, tenantID)
	if qerr == nil {
		for rRows.Next() {
			var p string
			_ = rRows.Scan(&p)
			perms[p] = true
		}
		rRows.Close()
	}
	out := []string{}
	for p := range perms {
		if roles.IsValid(p) {
			out = append(out, p)
		}
	}
	c.JSON(http.StatusOK, gin.H{"admin_id": adminID, "permissions": out, "count": len(out)})
}

// validateScopes returns false if any scope is not in the roles whitelist.
func (s *Svc) validateScopes(scopes []string) bool {
	for _, sc := range scopes {
		if !roles.IsValid(sc) {
			return false
		}
	}
	return true
}
