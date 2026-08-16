/**
 * TigerWallet Admin - RBAC Handler
 * Structured admin roles + granular permissions (GORM-backed).
 * Mirrors super_admin/go RBAC (commit 15e99eb). System roles/permissions are
 * protected: they cannot be deleted or have their name/permissions mutated.
 */

package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type RBACHandler struct {
	db *gorm.DB
}

func NewRBACHandler(db *gorm.DB) *RBACHandler {
	return &RBACHandler{db: db}
}

// AdminRole mirrors the admin_roles table.
type AdminRole struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name        string         `gorm:"uniqueIndex;not null" json:"name"`
	Description string         `json:"description"`
	Permissions pq.StringArray `gorm:"type:text[];not null;default:'{}'" json:"permissions"`
	IsSystem    bool           `gorm:"not null;default:false" json:"is_system"`
	IsActive    bool           `gorm:"not null;default:true" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

func (AdminRole) TableName() string { return "admin_roles" }

// AdminRoleAssignment mirrors the admin_role_assignments table.
type AdminRoleAssignment struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	AdminID   uuid.UUID  `gorm:"type:uuid;index:idx_admin_role_assignments_admin,unique:uniq_admin_role" json:"admin_id"`
	RoleID    uuid.UUID  `gorm:"type:uuid;index:idx_admin_role_assignments_admin,unique:uniq_admin_role" json:"role_id"`
	GrantedBy *uuid.UUID `gorm:"type:uuid" json:"granted_by"`
	GrantedAt time.Time  `json:"granted_at"`
}

func (AdminRoleAssignment) TableName() string { return "admin_role_assignments" }

// AdminPermission mirrors the admin_permissions table.
type AdminPermission struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name        string    `gorm:"uniqueIndex;not null" json:"name"`
	Description string    `json:"description"`
	Category    string    `gorm:"not null;default:'general'" json:"category"`
	IsActive    bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

func (AdminPermission) TableName() string { return "admin_permissions" }

// -------- Roles --------

func (h *RBACHandler) ListRoles(c *gin.Context) {
	var roles []AdminRole
	if err := h.db.Order("created_at DESC").Find(&roles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"roles": roles})
}

func (h *RBACHandler) GetRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role id"})
		return
	}
	var role AdminRole
	if err := h.db.First(&role, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"role": role})
}

func (h *RBACHandler) CreateRole(c *gin.Context) {
	var req struct {
		Name        string   `json:"name" binding:"required"`
		Description string   `json:"description"`
		Permissions []string `json:"permissions"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	role := AdminRole{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
		Permissions: pq.StringArray(req.Permissions),
		IsSystem:    false,
		IsActive:    true,
	}
	if err := h.db.Create(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"role": role})
}

func (h *RBACHandler) UpdateRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role id"})
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

	// System roles cannot be edited (only active flag toggled).
	var existing AdminRole
	if err := h.db.First(&existing, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}
	if existing.IsSystem {
		updates := map[string]interface{}{"updated_at": time.Now()}
		if req.IsActive != nil {
			updates["is_active"] = *req.IsActive
		}
		if err := h.db.Model(&existing).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "system role active flag updated"})
		return
	}

	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	updates["description"] = req.Description
	updates["permissions"] = pq.StringArray(req.Permissions)
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if err := h.db.Model(&existing).Where("id = ? AND is_system = false", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "admin role updated"})
}

func (h *RBACHandler) DeleteRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role id"})
		return
	}
	result := h.db.Where("id = ? AND is_system = false", id).Delete(&AdminRole{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found or is system-protected"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "admin role deleted"})
}

// -------- Permissions --------

func (h *RBACHandler) ListPermissions(c *gin.Context) {
	var perms []AdminPermission
	if err := h.db.Order("category ASC, name ASC").Find(&perms).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"permissions": perms})
}

func (h *RBACHandler) CreatePermission(c *gin.Context) {
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
	perm := AdminPermission{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		IsActive:    true,
	}
	if err := h.db.Create(&perm).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"permission": perm})
}

// -------- Assignments --------

func (h *RBACHandler) AssignRole(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role_id"})
		return
	}

	grantedByStr := c.GetString("user_id")
	var grantedBy *uuid.UUID
	if grantedByStr != "" {
		if uid, err := uuid.Parse(grantedByStr); err == nil {
			grantedBy = &uid
		}
	}

	assignment := AdminRoleAssignment{
		ID:        uuid.New(),
		AdminID:   adminID,
		RoleID:    roleID,
		GrantedBy: grantedBy,
		GrantedAt: time.Now(),
	}
	// ON CONFLICT (admin_id, role_id) DO NOTHING — emulate via unique constraint + ignore.
	result := h.db.Where("admin_id = ? AND role_id = ?", adminID, roleID).
		FirstOrCreate(&assignment)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "role assigned", "assignment": assignment})
}

func (h *RBACHandler) RevokeRole(c *gin.Context) {
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
	result := h.db.Where("admin_id = ? AND role_id = ?", adminID, roleID).Delete(&AdminRoleAssignment{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "role revoked"})
}

// GetEffectivePermissions aggregates permissions across all active roles
// assigned to an admin (mirrors super_admin's DISTINCT unnest query).
func (h *RBACHandler) GetEffectivePermissions(c *gin.Context) {
	adminID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid admin id"})
		return
	}

	var perms []string
	// Join assignments -> roles, unnest the permissions array, distinct.
	rows, err := h.db.Raw(`
		SELECT DISTINCT unnest(r.permissions) AS permission
		FROM admin_role_assignments a
		JOIN admin_roles r ON r.id = a.role_id
		WHERE a.admin_id = ? AND r.is_active = true
	`, adminID).Rows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err == nil {
			perms = append(perms, p)
		}
	}
	if perms == nil {
		perms = []string{}
	}
	c.JSON(http.StatusOK, gin.H{"admin_id": adminID, "permissions": perms})
}

// ListAdminRoles is a convenience alias matching the super_admin naming.
func (h *RBACHandler) ListAdminRoles(c *gin.Context) { h.ListRoles(c) }
