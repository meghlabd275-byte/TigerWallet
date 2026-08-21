// policy_handlers.go — SuperAdmin handlers for the AutoApprover policy snapshot.
//
// These routes let SuperAdmin configure the security boundary that the WL
// products' AutoApprover consults on the fast path:
//   - Treasury addresses: any outgoing tx to one of these => MANUAL two-party.
//     This prevents the WL client or MasterWallet owner from routing
//     fee/revenue/treasury withdrawals through the <1s auto-approve fast path.
//   - Auto-sign rules: SuperAdmin can block a specific auto-approve even when
//     the license is alive (e.g. block auto-approve above a per-tx amount cap,
//     or block a specific token contract).
//
// All routes are SuperAdmin-only (RequireSuperAdmin middleware in main.go).
// Changes propagate to the WL products on the next heartbeat (the control
// plane pushes the snapshot alongside the feature flags).
package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// --- Treasury addresses ---

// AddTreasuryAddress (SuperAdmin): mark an address as a fee/revenue/treasury
// destination so any outgoing tx to it is forced to MANUAL two-party mode.
func (h *Handlers) AddTreasuryAddress(c *gin.Context) {
	var req struct {
		WLClientID uuid.UUID `json:"wl_client_id" binding:"required"`
		Product    string    `json:"product" binding:"required"`
		Address    string    `json:"address" binding:"required"`
		Label      string    `json:"label"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !strings.HasPrefix(req.Address, "0x") || len(req.Address) != 42 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "address must be a 0x-prefixed 20-byte hex"})
		return
	}
	adminID := superAdminID(c)
	addr, err := h.store.AddTreasuryAddress(c.Request.Context(), req.WLClientID, req.Product, req.Address, req.Label, &adminID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, addr)
}

// ListTreasuryAddresses (SuperAdmin): list all treasury addresses for a WL client + product.
func (h *Handlers) ListTreasuryAddresses(c *gin.Context) {
	wlID, err := uuid.Parse(c.Query("wl_client_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wl_client_id required"})
		return
	}
	product := c.Query("product")
	if product == "" {
		product = "%"
	}
	addrs, err := h.store.ListTreasuryAddresses(c.Request.Context(), wlID, product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"treasury_addresses": addrs})
}

// DeleteTreasuryAddress (SuperAdmin).
func (h *Handlers) DeleteTreasuryAddress(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.store.DeleteTreasuryAddress(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

// --- Auto-sign rules ---

// SetAutoSignRule (SuperAdmin): create/update a rule that can block an
// auto-approve even when the license is alive.
func (h *Handlers) SetAutoSignRule(c *gin.Context) {
	var req struct {
		WLClientID uuid.UUID `json:"wl_client_id" binding:"required"`
		Product    string    `json:"product" binding:"required"`
		Fetcher    string    `json:"fetcher"`
		TxType     string    `json:"tx_type"`
		Token      string    `json:"token"`
		MaxAmount  string    `json:"max_amount"`
		Block      bool      `json:"block"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Fetcher == "" {
		req.Fetcher = "*"
	}
	if req.TxType == "" {
		req.TxType = "*"
	}
	if req.Token == "" {
		req.Token = "*"
	}
	if req.MaxAmount == "" {
		req.MaxAmount = "0"
	}
	adminID := superAdminID(c)
	rule, err := h.store.SetAutoSignRule(c.Request.Context(), req.WLClientID, req.Product, req.Fetcher, req.TxType, req.Token, req.MaxAmount, req.Block, &adminID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, rule)
}

// ListAutoSignRules (SuperAdmin).
func (h *Handlers) ListAutoSignRules(c *gin.Context) {
	wlID, err := uuid.Parse(c.Query("wl_client_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wl_client_id required"})
		return
	}
	product := c.Query("product")
	if product == "" {
		product = "%"
	}
	rules, err := h.store.ListAutoSignRules(c.Request.Context(), wlID, product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"auto_sign_rules": rules})
}

// DeleteAutoSignRule (SuperAdmin).
func (h *Handlers) DeleteAutoSignRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.store.DeleteAutoSignRule(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

// superAdminID extracts the SuperAdmin's sa_admin id from the JWT context.
func superAdminID(c *gin.Context) uuid.UUID {
	if v, ok := c.Get("admin_id"); ok {
		if id, ok := v.(uuid.UUID); ok {
			return id
		}
	}
	return uuid.Nil
}
