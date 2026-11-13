package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tigerwallet/multi-tenancy/internal/models"
	"github.com/tigerwallet/multi-tenancy/internal/services"
)

// Handlers exposes the tenant control plane over REST. Every handler delegates
// to services.TenantService, which performs the real PostgreSQL work; nothing
// here fabricates a response.
type Handlers struct {
	svc *services.TenantService
}

func New(svc *services.TenantService) *Handlers {
	return &Handlers{svc: svc}
}

func fail(c *gin.Context, code int, err error) {
	c.JSON(code, gin.H{"error": err.Error()})
}

func parseUUID(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return uuid.Nil, false
	}
	return id, true
}

// Health reports process liveness. The database probe is separate so an
// orchestrator can distinguish "process up" from "database reachable".
func (h *Handlers) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "multi-tenancy"})
}

func (h *Handlers) Ready(c *gin.Context) {
	ctx := c.Request.Context()
	if services.Pool == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "reason": "database pool not initialised"})
		return
	}
	if err := services.Pool.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "reason": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

// ---- Tenants ----

type createTenantReq struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required"`
	Slug  string `json:"slug" binding:"required"`
}

func (h *Handlers) CreateTenant(c *gin.Context) {
	var req createTenantReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	t, err := h.svc.CreateTenant(c.Request.Context(), req.Name, req.Email, req.Slug)
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"tenant": t})
}

func (h *Handlers) ListTenants(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var status *models.TenantStatus
	if s := c.Query("status"); s != "" {
		st := models.TenantStatus(s)
		status = &st
	}

	tenants, err := h.svc.ListTenants(c.Request.Context(), status, limit, offset)
	if err != nil {
		fail(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"tenants": tenants, "limit": limit, "offset": offset})
}

func (h *Handlers) GetTenant(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}
	t, err := h.svc.GetTenantByID(c.Request.Context(), id)
	if err != nil {
		fail(c, http.StatusNotFound, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"tenant": t})
}

func (h *Handlers) GetTenantBySlug(c *gin.Context) {
	t, err := h.svc.GetTenantBySlug(c.Request.Context(), c.Param("slug"))
	if err != nil {
		fail(c, http.StatusNotFound, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"tenant": t})
}

func (h *Handlers) UpdateTenant(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	if err := h.svc.UpdateTenant(c.Request.Context(), id, updates); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "tenant updated"})
}

type statusReq struct {
	Status models.TenantStatus `json:"status" binding:"required"`
}

func (h *Handlers) UpdateTenantStatus(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}
	var req statusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	if err := h.svc.UpdateTenantStatus(c.Request.Context(), id, req.Status); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "status updated", "status": req.Status})
}

func (h *Handlers) DeleteTenant(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.DeleteTenant(c.Request.Context(), id); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "tenant deleted"})
}

// ---- Users ----

type addUserReq struct {
	UserID uuid.UUID `json:"user_id" binding:"required"`
	Role   string    `json:"role"`
}

func (h *Handlers) AddUser(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}
	var req addUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	if req.Role == "" {
		req.Role = "member"
	}
	u, err := h.svc.AddUserToTenant(c.Request.Context(), id, req.UserID, req.Role)
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"user": u})
}

func (h *Handlers) ListUsers(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}
	users, err := h.svc.GetTenantUsers(c.Request.Context(), id)
	if err != nil {
		fail(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": users})
}

type roleReq struct {
	Role string `json:"role" binding:"required"`
}

func (h *Handlers) UpdateUserRole(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}
	userID, ok := parseUUID(c, "user_id")
	if !ok {
		return
	}
	var req roleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	if err := h.svc.UpdateUserRole(c.Request.Context(), id, userID, req.Role); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "role updated"})
}

func (h *Handlers) RemoveUser(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}
	userID, ok := parseUUID(c, "user_id")
	if !ok {
		return
	}
	if err := h.svc.RemoveUserFromTenant(c.Request.Context(), id, userID); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "user removed"})
}

// ---- Invitations ----

type inviteReq struct {
	Email     string    `json:"email" binding:"required"`
	Role      string    `json:"role"`
	InvitedBy uuid.UUID `json:"invited_by" binding:"required"`
}

func (h *Handlers) CreateInvitation(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}
	var req inviteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	if req.Role == "" {
		req.Role = "member"
	}
	inv, err := h.svc.CreateInvitation(c.Request.Context(), id, req.InvitedBy, req.Email, req.Role)
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"invitation": inv})
}

type acceptReq struct {
	UserID uuid.UUID `json:"user_id" binding:"required"`
}

func (h *Handlers) AcceptInvitation(c *gin.Context) {
	var req acceptReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	if err := h.svc.AcceptInvitation(c.Request.Context(), c.Param("token"), req.UserID); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "invitation accepted"})
}

// ---- Quotas ----

func (h *Handlers) GetQuota(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}
	resource := c.Query("resource")
	if resource == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "resource query parameter is required"})
		return
	}
	q, err := h.svc.GetQuota(c.Request.Context(), id.String(), resource)
	if err != nil {
		fail(c, http.StatusNotFound, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"quota": q})
}

type quotaReq struct {
	Resource string `json:"resource" binding:"required"`
	Amount   int64  `json:"amount" binding:"required"`
}

func (h *Handlers) IncrementQuota(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}
	var req quotaReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	if err := h.svc.IncrementQuota(c.Request.Context(), id.String(), req.Resource, req.Amount); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "quota incremented"})
}

func (h *Handlers) CheckQuota(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}
	resource := c.Query("resource")
	if resource == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "resource query parameter is required"})
		return
	}
	allowed, err := h.svc.CheckQuota(c.Request.Context(), id.String(), resource)
	if err != nil {
		fail(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"allowed": allowed, "resource": resource})
}

// ---- Config ----

func (h *Handlers) GetConfig(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}
	cfg, err := h.svc.GetTenantConfig(c.Request.Context(), id)
	if err != nil {
		fail(c, http.StatusNotFound, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"config": cfg})
}

func (h *Handlers) UpdateConfig(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}
	var cfg models.TenantConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	if err := h.svc.UpdateTenantConfig(c.Request.Context(), id, &cfg); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "config updated"})
}

// ---- Audit ----

func (h *Handlers) GetAuditLogs(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	logs, err := h.svc.GetAuditLogs(c.Request.Context(), id, limit)
	if err != nil {
		fail(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

type auditReq struct {
	UserID   uuid.UUID              `json:"user_id" binding:"required"`
	Action   string                 `json:"action" binding:"required"`
	Resource string                 `json:"resource" binding:"required"`
	Details  map[string]interface{} `json:"details"`
}

func (h *Handlers) CreateAuditLog(c *gin.Context) {
	id, ok := parseUUID(c, "id")
	if !ok {
		return
	}
	var req auditReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	if err := h.svc.CreateAuditLog(c.Request.Context(), id, req.UserID, req.Action, req.Resource, req.Details); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "audit log recorded"})
}
