package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tigerwallet/license-service/internal/config"
	"github.com/tigerwallet/license-service/internal/crypto"
	"github.com/tigerwallet/license-service/internal/store"
	"golang.org/x/crypto/bcrypt"
)

const tokenTTL = 5 * time.Minute // short-lived license tokens; renewed on heartbeat

// Handlers bundles the control-plane dependencies.
type Handlers struct {
	cfg   *config.Config
	store *store.Store
	hub   *store.Hub
	keys  *crypto.KeyPair
}

func New(cfg *config.Config, st *store.Store, hub *store.Hub, keys *crypto.KeyPair) *Handlers {
	return &Handlers{cfg: cfg, store: st, hub: hub, keys: keys}
}

// --- auth: SuperAdmin login (bootstrap credential seeded at startup) ---

func (h *Handlers) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	id, role, err := h.store.VerifySuperAdmin(ctx, req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	tok, err := IssueJWT(h.cfg, id, req.Email, role, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token issue failed"})
		return
	}
	h.store.Audit(ctx, id, "auth.login", "sa_admin", id.String(), gin.H{"email": req.Email})
	c.JSON(http.StatusOK, gin.H{"token": tok, "admin": gin.H{"id": id, "email": req.Email, "role": role}})
}

// --- WL client login (limited role; can request withdrawals + heartbeat, NOT govern) ---

func (h *Handlers) WLClientLogin(c *gin.Context) {
	var req struct {
		Slug     string `json:"slug" binding:"required"`
		APIKey   string `json:"api_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// The WL client authenticates with its license key (its API key). Resolve
	// the license -> wl_client, then issue a wl_client-scoped JWT.
	ctx := c.Request.Context()
	lic, err := h.store.GetLicenseByKey(ctx, req.APIKey)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid api_key (license key)"})
		return
	}
	if lic.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "license is " + lic.Status, "status": lic.Status})
		return
	}
	wlc, err := h.store.GetWLClient(ctx, lic.WLClientID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "wl client not found"})
		return
	}
	if wlc.Status != "approved" && wlc.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "wl client is " + wlc.Status})
		return
	}
	tok, err := IssueJWT(h.cfg, lic.WLClientID, wlc.ContactEmail, "wl_client", &lic.WLClientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token issue failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": tok, "wl_client": wlc, "license": lic})
}

// ==================== SuperAdmin: WL client lifecycle ====================

func (h *Handlers) CreateWLClient(c *gin.Context) {
	var req struct {
		Name         string   `json:"name" binding:"required"`
		Slug         string   `json:"slug" binding:"required"`
		ContactEmail string   `json:"contact_email" binding:"required"`
		Tier         string   `json:"tier"`
		Products     []string `json:"products"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Tier == "" {
		req.Tier = "basic"
	}
	if len(req.Products) == 0 {
		req.Products = []string{"master_wallet", "user_wallet", "bots", "project_party"}
	}
	ctx := c.Request.Context()
	wlc, err := h.store.CreateWLClient(ctx, req.Name, req.Slug, req.ContactEmail, req.Tier, req.Products)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.store.Audit(ctx, adminID(c), "wl_client.create", "wl_client", wlc.ID.String(), gin.H{"name": req.Name})
	c.JSON(http.StatusCreated, wlc)
}

func (h *Handlers) ListWLClients(c *gin.Context) {
	ctx := c.Request.Context()
	clients, err := h.store.ListWLClients(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"clients": clients})
}

func (h *Handlers) ApproveWLClient(c *gin.Context) {
	h.transitionWLClient(c, "approved")
}

func (h *Handlers) SuspendWLClient(c *gin.Context) {
	h.transitionWLClient(c, "suspended")
	// When a WL client is suspended, ALL its products must die. Queue halt
	// commands for every product and publish immediately.
	h.killAllProducts(c)
}

func (h *Handlers) HaltWLClient(c *gin.Context) {
	h.transitionWLClient(c, "halted")
	h.killAllProducts(c)
}

func (h *Handlers) RevokeWLClient(c *gin.Context) {
	h.transitionWLClient(c, "revoked")
	h.killAllProducts(c)
}

// ResumeWLClient — SuperAdmin-ONLY reactivation. A WL client can never call
// this (RequireSuperAdmin gate). This is the single allowed path back to
// 'active' from suspended/halted/revoked.
func (h *Handlers) ResumeWLClient(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	if err := h.store.SetWLClientStatus(ctx, id, "active", adminID(c).String(), true); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Resume all the client's licenses too.
	lics, _ := h.store.ListLicenses(ctx, &id)
	for _, l := range lics {
		_ = h.store.SetLicenseStatus(ctx, l.ID, "active", true)
	}
	// Notify products to come back online.
	for _, l := range lics {
		_, _ = h.store.QueueCommand(ctx, id, l.Product, "resume", nil, adminID(c))
		_ = h.hub.PublishCommand(ctx, id, l.Product, map[string]any{"command": "resume"})
	}
	h.store.Audit(ctx, adminID(c), "wl_client.resume", "wl_client", id.String(), nil)
	c.JSON(http.StatusOK, gin.H{"resumed": id})
}

func (h *Handlers) transitionWLClient(c *gin.Context, status string) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	if err := h.store.SetWLClientStatus(ctx, id, status, adminID(c).String(), false); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Cascade to all the client's licenses.
	lcs, _ := h.store.ListLicenses(ctx, &id)
	for _, l := range lcs {
		_ = h.store.SetLicenseStatus(ctx, l.ID, status, false)
	}
	h.store.Audit(ctx, adminID(c), "wl_client."+status, "wl_client", id.String(), nil)
	c.JSON(http.StatusOK, gin.H{"transitioned": id, "status": status})
}

// killAllProducts queues halt commands for every product of a WL client and
// publishes them in real-time so the externally-hosted products stop serving
// immediately (fail-closed on next heartbeat AND via pub/sub).
func (h *Handlers) killAllProducts(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return
	}
	ctx := c.Request.Context()
	lcs, _ := h.store.ListLicenses(ctx, &id)
	for _, l := range lcs {
		_, _ = h.store.QueueCommand(ctx, id, l.Product, "halt", map[string]any{"reason": "wl_client_suspended"}, adminID(c))
		_ = h.hub.PublishCommand(ctx, id, l.Product, map[string]any{"command": "halt", "reason": "wl_client_suspended"})
	}
}

// UpdateWLClient — SuperAdmin updates tier + allowed products (add/remove a
// whole product from a WL client). This is the "add/remove any feature" hook
// at the product granularity.
func (h *Handlers) UpdateWLClient(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Tier     string   `json:"tier"`
		Products []string `json:"products"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	if err := h.store.UpdateWLClient(ctx, id, req.Tier, req.Products); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.store.Audit(ctx, adminID(c), "wl_client.update", "wl_client", id.String(), gin.H{"tier": req.Tier, "products": req.Products})
	c.JSON(http.StatusOK, gin.H{"updated": id})
}

// ==================== SuperAdmin: license ops ====================

func (h *Handlers) IssueLicense(c *gin.Context) {
	var req struct {
		WLClientID string   `json:"wl_client_id" binding:"required"`
		Product    string   `json:"product" binding:"required"`
		Plan       string   `json:"plan"`
		DurationDays int    `json:"duration_days"`
		MaxUsers   int      `json:"max_users"`
		MaxWallets int      `json:"max_wallets"`
		MaxBots    int      `json:"max_bots"`
		Features   []string `json:"features"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Plan == "" {
		req.Plan = "basic"
	}
	if req.DurationDays == 0 {
		req.DurationDays = 365
	}
	if req.MaxUsers == 0 {
		req.MaxUsers = 100
	}
	if req.MaxWallets == 0 {
		req.MaxWallets = 500
	}
	if req.MaxBots == 0 {
		req.MaxBots = 50
	}
	wlID, err := uuid.Parse(req.WLClientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wl_client_id"})
		return
	}
	key := generateLicenseKey()
	until := time.Now().AddDate(0, 0, req.DurationDays)
	ctx := c.Request.Context()
	lic, err := h.store.CreateLicense(ctx, wlID, req.Product, req.Plan, key, until, req.MaxUsers, req.MaxWallets, req.MaxBots, req.Features, adminID(c))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.store.Audit(ctx, adminID(c), "license.issue", "license", lic.ID.String(), gin.H{"product": req.Product, "plan": req.Plan})
	c.JSON(http.StatusCreated, lic)
}

func (h *Handlers) ListLicenses(c *gin.Context) {
	ctx := c.Request.Context()
	var wlID *uuid.UUID
	if wl := c.Query("wl_client_id"); wl != "" {
		id, err := uuid.Parse(wl)
		if err == nil {
			wlID = &id
		}
	}
	lics, err := h.store.ListLicenses(ctx, wlID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"licenses": lics})
}

func (h *Handlers) SuspendLicense(c *gin.Context)  { h.transitionLicense(c, "suspended") }
func (h *Handlers) HaltLicense(c *gin.Context)     { h.transitionLicense(c, "halted") }
func (h *Handlers) RevokeLicense(c *gin.Context)   { h.transitionLicense(c, "revoked") }
func (h *Handlers) ResumeLicense(c *gin.Context)   { h.transitionLicense(c, "active") }

func (h *Handlers) transitionLicense(c *gin.Context, status string) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	allowResume := status == "active" // only SuperAdmin reaches here (route gate)
	if err := h.store.SetLicenseStatus(ctx, id, status, allowResume); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	lic, _ := h.store.GetLicense(ctx, id)
	if lic != nil {
		cmd := status
		if status == "active" {
			cmd = "resume"
		}
		_, _ = h.store.QueueCommand(ctx, lic.WLClientID, lic.Product, cmd, nil, adminID(c))
		_ = h.hub.PublishCommand(ctx, lic.WLClientID, lic.Product, map[string]any{"command": cmd})
	}
	h.store.Audit(ctx, adminID(c), "license."+status, "license", id.String(), nil)
	c.JSON(http.StatusOK, gin.H{"transitioned": id, "status": status})
}

// ==================== SuperAdmin: feature flags (per-fetcher) ====================

func (h *Handlers) SetFeatureFlag(c *gin.Context) {
	var req struct {
		WLClientID string `json:"wl_client_id" binding:"required"`
		Product    string `json:"product" binding:"required"`
		Fetcher    string `json:"fetcher"`
		Enabled    bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Fetcher == "" {
		req.Fetcher = "*" // whole-product toggle
	}
	wlID, err := uuid.Parse(req.WLClientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wl_client_id"})
		return
	}
	ctx := c.Request.Context()
	if err := h.store.SetFeatureFlag(ctx, wlID, req.Product, req.Fetcher, req.Enabled, adminID(c)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_ = h.hub.PublishFlagChange(ctx, wlID, req.Product)
	h.store.Audit(ctx, adminID(c), "flag.set", "feature_flag", req.Product+"/"+req.Fetcher, gin.H{"enabled": req.Enabled})
	c.JSON(http.StatusOK, gin.H{"updated": true, "product": req.Product, "fetcher": req.Fetcher, "enabled": req.Enabled})
}

func (h *Handlers) ListFeatureFlags(c *gin.Context) {
	wlID, err := uuid.Parse(c.Query("wl_client_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wl_client_id required"})
		return
	}
	product := c.Query("product")
	flags, err := h.store.ListFeatureFlags(c.Request.Context(), wlID, product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"flags": flags})
}

// ==================== WL-product-facing: validate + heartbeat ====================

// ValidateLicense is called by an externally-hosted WL product on startup and
// on each token renewal. Returns a fresh signed license token + the current
// flag set + pending commands. Fail-closed: any non-active status returns 403.
func (h *Handlers) ValidateLicense(c *gin.Context) {
	var req struct {
		LicenseKey string `json:"license_key" binding:"required"`
		Product    string `json:"product" binding:"required"`
		InstanceID string `json:"instance_id" binding:"required"`
		Version    string `json:"version"`
		Hostname   string `json:"hostname"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	lic, err := h.store.GetLicenseByKey(ctx, req.LicenseKey)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"valid": false, "error": "license not found"})
		return
	}
	// Authoritative fail-closed check: WL client + license + heartbeat all alive.
	alive, reason, err := h.store.IsProductAlive(ctx, lic.WLClientID, req.Product, h.cfg.HeartbeatTimeout)
	if err != nil || !alive {
		c.JSON(http.StatusForbidden, gin.H{"valid": false, "alive": false, "reason": reason})
		return
	}
	// Record this validation as a heartbeat (the product is alive right now).
	_ = h.store.RecordHeartbeat(ctx, lic.WLClientID, req.Product, req.InstanceID, 0, req.Version, req.Hostname, nil)
	// Mint a fresh signed token.
	tok := crypto.NewToken(lic.ID.String(), lic.WLClientID.String(), lic.Product, lic.Plan, lic.Status,
		lic.ValidFrom.Unix(), lic.ValidUntil.Unix(), lic.MaxUsers, lic.MaxWallets, lic.MaxBots, lic.Features, tokenTTL)
	signed, err := h.keys.SignToken(tok)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token sign failed"})
		return
	}
	// Attach current flag set so the product refreshes its cache.
	flags, _ := h.store.ListFeatureFlags(ctx, lic.WLClientID, lic.Product)
	// Deliver pending commands.
	cmds, _ := h.store.DeliverPendingCommands(ctx, lic.WLClientID, req.Product)
	c.JSON(http.StatusOK, gin.H{
		"valid":          true,
		"alive":          true,
		"token":          signed,
		"verify_key_hex": h.keys.PublicKeyHex(),
		"flags":          flags,
		"commands":       cmds,
		"heartbeat_timeout_seconds": int(h.cfg.HeartbeatTimeout.Seconds()),
	})
}

// Heartbeat is the periodic check-in. A WL product MUST call this within the
// heartbeat_timeout window or it is considered dead (and IsProductAlive will
// fail-closed on the next validate). Returns pending commands + refreshed flags.
func (h *Handlers) Heartbeat(c *gin.Context) {
	var req struct {
		LicenseKey string `json:"license_key" binding:"required"`
		Product    string `json:"product" binding:"required"`
		InstanceID string `json:"instance_id" binding:"required"`
		LatencyMs  int    `json:"latency_ms"`
		Version    string `json:"version"`
		Hostname   string `json:"hostname"`
		Metrics    map[string]any `json:"metrics"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	lic, err := h.store.GetLicenseByKey(ctx, req.LicenseKey)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"alive": false, "reason": "license not found"})
		return
	}
	alive, reason, _ := h.store.IsProductAlive(ctx, lic.WLClientID, req.Product, h.cfg.HeartbeatTimeout+h.cfg.GracePeriod)
	if !alive {
		c.JSON(http.StatusForbidden, gin.H{"alive": false, "reason": reason, "command": "halt"})
		return
	}
	_ = h.store.RecordHeartbeat(ctx, lic.WLClientID, req.Product, req.InstanceID, req.LatencyMs, req.Version, req.Hostname, req.Metrics)
	cmds, _ := h.store.DeliverPendingCommands(ctx, lic.WLClientID, req.Product)
	flags, _ := h.store.ListFeatureFlags(ctx, lic.WLClientID, req.Product)
	// Renew the token on each heartbeat.
	tok := crypto.NewToken(lic.ID.String(), lic.WLClientID.String(), lic.Product, lic.Plan, lic.Status,
		lic.ValidFrom.Unix(), lic.ValidUntil.Unix(), lic.MaxUsers, lic.MaxWallets, lic.MaxBots, lic.Features, tokenTTL)
	signed, _ := h.keys.SignToken(tok)
	c.JSON(http.StatusOK, gin.H{
		"alive":    true,
		"token":    signed,
		"flags":    flags,
		"commands": cmds,
	})
}

// CommandAck — a WL product acknowledges executing a command.
func (h *Handlers) CommandAck(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Result string `json:"result"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := h.store.MarkCommandExecuted(c.Request.Context(), id, req.Result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"acked": id})
}

// IsFetcherEnabled — a WL product's fetcher checks whether SuperAdmin still
// permits it to serve. This is the per-fetcher granularity gate.
func (h *Handlers) IsFetcherEnabled(c *gin.Context) {
	wlID, err := uuid.Parse(c.Query("wl_client_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wl_client_id required"})
		return
	}
	product := c.Query("product")
	fetcher := c.Query("fetcher")
	if product == "" || fetcher == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product and fetcher required"})
		return
	}
	enabled, err := h.store.IsFetcherEnabled(c.Request.Context(), wlID, product, fetcher)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"enabled": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"enabled": enabled})
}

// ==================== Two-party withdrawal collaboration ====================

// RequestWithdrawal — a WL client requests a fund/revenue withdrawal. This
// creates a 'wl_approved' record (the WL side has approved). It does NOT move
// funds. Funds move ONLY after SuperAdminApproveWithdrawal flips it to
// 'approved' AND the master-wallet backend enforces the two-party gate.
func (h *Handlers) RequestWithdrawal(c *gin.Context) {
	wlClientID, ok := c.Get("wl_client_id")
	var wlID uuid.UUID
	if ok {
		wlID = wlClientID.(uuid.UUID)
	} else {
		// SuperAdmin may also file on behalf.
		id, err := uuid.Parse(c.Query("wl_client_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "wl_client_id required"})
			return
		}
		wlID = id
	}
	var req struct {
		Product      string `json:"product" binding:"required"`
		ResourceType string `json:"resource_type" binding:"required"`
		ResourceID   string `json:"resource_id" binding:"required"`
		AmountWei    string `json:"amount_wei" binding:"required"`
		ToAddress    string `json:"to_address" binding:"required"`
		ChainID      int64  `json:"chain_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	now := time.Now()
	wa := &store.WithdrawalApproval{
		WLClientID:   wlID,
		Product:      req.Product,
		ResourceType: req.ResourceType,
		ResourceID:   req.ResourceID,
		AmountWei:    req.AmountWei,
		ToAddress:    req.ToAddress,
		ChainID:      req.ChainID,
		WLApproverID: &wlID,
		WLApprovedAt: &now,
	}
	ctx := c.Request.Context()
	if err := h.store.CreateWithdrawalApproval(ctx, wa); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.store.Audit(ctx, adminID(c), "withdrawal.request", "withdrawal", wa.ID.String(), gin.H{"amount": req.AmountWei, "to": req.ToAddress})
	c.JSON(http.StatusCreated, wa)
}

// SuperAdminApproveWithdrawal — the mandatory second signature. Only after
// this is the withdrawal 'approved' and executable by the master-wallet.
func (h *Handlers) SuperAdminApproveWithdrawal(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	if err := h.store.SuperAdminApproveWithdrawal(ctx, id, adminID(c)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.store.Audit(ctx, adminID(c), "withdrawal.superadmin_approve", "withdrawal", id.String(), nil)
	c.JSON(http.StatusOK, gin.H{"approved": id, "two_party_complete": true})
}

func (h *Handlers) SuperAdminRejectWithdrawal(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	_ = h.store.SuperAdminRejectWithdrawal(ctx, id, adminID(c), "rejected by superadmin")
	h.store.Audit(ctx, adminID(c), "withdrawal.superadmin_reject", "withdrawal", id.String(), nil)
	c.JSON(http.StatusOK, gin.H{"rejected": id})
}

// MarkWithdrawalExecuted — the master-wallet backend calls this after it has
// broadcast the tx (gated by IsWithdrawalApproved) to record the tx hash.
func (h *Handlers) MarkWithdrawalExecuted(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		TxHash string `json:"tx_hash" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.store.MarkWithdrawalExecuted(c.Request.Context(), id, req.TxHash); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"executed": id, "tx_hash": req.TxHash})
}

// IsWithdrawalApproved — the master-wallet backend calls this right before
// broadcasting to enforce the two-party gate. Returns 200 + approved:true ONLY
// when both the WL side and SuperAdmin have approved.
func (h *Handlers) IsWithdrawalApproved(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	approved, err := h.store.IsWithdrawalApproved(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"approved": false, "error": err.Error()})
		return
	}
	if !approved {
		c.JSON(http.StatusForbidden, gin.H{"approved": false, "error": "two-party approval not complete (requires SuperAdmin co-sign)"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"approved": true})
}

func (h *Handlers) ListWithdrawalApprovals(c *gin.Context) {
	var wlID *uuid.UUID
	if wl := c.Query("wl_client_id"); wl != "" {
		id, err := uuid.Parse(wl)
		if err == nil {
			wlID = &id
		}
	}
	was, err := h.store.ListWithdrawalApprovals(c.Request.Context(), wlID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"withdrawals": was})
}

// --- SuperAdmin signer address (consumed by master-wallet multisig injection) ---

func (h *Handlers) GetSuperAdminSigner(c *gin.Context) {
	if h.cfg.SuperAdminSignerAddress == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "SUPER_ADMIN_SIGNER_ADDRESS not configured"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"address": h.cfg.SuperAdminSignerAddress})
}

// --- helpers ---

func (h *Handlers) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "license-control-plane", "verify_key": h.keys.PublicKeyHex()})
}

func generateLicenseKey() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// rand.Read must not fail on Linux; fall back to time-mix (still unique-ish).
		return fmt.Sprintf("twl-%d", time.Now().UnixNano())
	}
	return "twl-" + hex.EncodeToString(b)
}

// bootstrapSuperAdmin seeds the first superadmin if none exists.
func BootstrapSuperAdmin(ctx context.Context, st *store.Store, email, password string) error {
	if email == "" || password == "" {
		return errors.New("bootstrap credential required")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return err
	}
	_, err = st.DB().Exec(ctx,
		`INSERT INTO sa_admins (email, password_hash, role) VALUES ($1,$2,'superadmin')
		 ON CONFLICT (email) DO NOTHING`, email, string(hash))
	return err
}
