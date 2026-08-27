package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tigerwallet/billing/internal/database"
	"github.com/tigerwallet/billing/internal/models"
	"github.com/tigerwallet/billing/internal/services"
)

type BillingHandler struct {
	service *services.BillingService
}

func NewBillingHandler() *BillingHandler {
	return &BillingHandler{
		service: services.NewBillingService(),
	}
}

// Plan handlers

func (h *BillingHandler) GetPlans(c *gin.Context) {
	plans, err := h.service.GetAllPlans(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"plans": plans})
}

func (h *BillingHandler) GetPlan(c *gin.Context) {
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan ID"})
		return
	}

	plan, err := h.service.GetPlanByID(c.Request.Context(), planID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		return
	}
	c.JSON(http.StatusOK, plan)
}

// Tenant handlers

type CreateTenantRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
	Slug  string `json:"slug" binding:"required"`
}

func (h *BillingHandler) CreateTenant(c *gin.Context) {
	var req CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant, err := h.service.CreateTenant(c.Request.Context(), req.Name, req.Email, req.Slug)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, tenant)
}

func (h *BillingHandler) GetTenant(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant ID"})
		return
	}

	tenant, err := h.service.GetTenantByID(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
		return
	}
	c.JSON(http.StatusOK, tenant)
}

func (h *BillingHandler) GetTenantBySlug(c *gin.Context) {
	slug := c.Param("slug")

	tenant, err := h.service.GetTenantBySlug(c.Request.Context(), slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
		return
	}
	c.JSON(http.StatusOK, tenant)
}

type UpdateTenantStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

func (h *BillingHandler) UpdateTenantStatus(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant ID"})
		return
	}

	var req UpdateTenantStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdateTenantStatus(c.Request.Context(), tenantID, req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "status updated"})
}

// Subscription handlers

func (h *BillingHandler) GetSubscription(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant ID"})
		return
	}

	subscription, err := h.service.GetSubscriptionByTenantID(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}
	c.JSON(http.StatusOK, subscription)
}

type UpgradeSubscriptionRequest struct {
	PlanID   uuid.UUID `json:"plan_id" binding:"required"`
	BillingCycle string `json:"billing_cycle" binding:"required,oneof=monthly yearly"`
}

func (h *BillingHandler) UpgradeSubscription(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant ID"})
		return
	}

	var req UpgradeSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get current subscription
	sub, err := h.service.GetSubscriptionByTenantID(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}

	// Update subscription plan (in real implementation, this would call Stripe)
	sub.PlanID = req.PlanID
	sub.Status = models.SubStatusActive

	// Update period based on billing cycle
	if req.BillingCycle == "yearly" {
		sub.CurrentPeriodStart = time.Now()
		sub.CurrentPeriodEnd = time.Now().AddDate(1, 0, 0)
	} else {
		sub.CurrentPeriodStart = time.Now()
		sub.CurrentPeriodEnd = time.Now().AddDate(0, 1, 0)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "subscription upgraded",
		"subscription": sub,
	})
}

func (h *BillingHandler) CancelSubscription(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant ID"})
		return
	}

	sub, err := h.service.GetSubscriptionByTenantID(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}

	// Cancel at period end (Stripe would handle this)
	sub.CancelAtPeriodEnd = true

	c.JSON(http.StatusOK, gin.H{
		"message": "subscription will be canceled at period end",
		"subscription": sub,
	})
}

// Usage handlers

func (h *BillingHandler) GetUsage(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant ID"})
		return
	}

	// Get current month period
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	periodEnd := periodStart.AddDate(0, 1, 0)

	summary, err := h.service.GetUsageSummary(c.Request.Context(), tenantID, periodStart, periodEnd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, summary)
}

type RecordUsageRequest struct {
	APIMethod string `json:"api_method" binding:"required"`
	Count     int64  `json:"count" binding:"required,min=1"`
}

func (h *BillingHandler) RecordUsage(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant ID"})
		return
	}

	var req RecordUsageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check quota first
	hasQuota, err := h.service.CheckQuota(c.Request.Context(), tenantID, "api")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !hasQuota {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "API quota exceeded"})
		return
	}

	if err := h.service.RecordUsage(c.Request.Context(), tenantID, req.APIMethod, req.Count); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "usage recorded"})
}

func (h *BillingHandler) CheckQuota(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant ID"})
		return
	}

	resourceType := c.Query("type")
	if resourceType == "" {
		resourceType = "api"
	}

	hasQuota, err := h.service.CheckQuota(c.Request.Context(), tenantID, resourceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"has_quota": hasQuota,
		"resource":  resourceType,
	})
}

// Invoice handlers

func (h *BillingHandler) GetInvoices(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant ID"})
		return
	}

	rows, err := database.Pool.Query(c.Request.Context(), `
		SELECT id, tenant_id, invoice_number, stripe_invoice_id, amount, amount_due, 
			amount_paid, currency, status, due_date, paid_at, invoice_url, invoice_pdf, 
			created_at, updated_at
		FROM invoices WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT 50
	`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var invoices []models.Invoice
	for rows.Next() {
		var inv models.Invoice
		if err := rows.Scan(
			&inv.ID, &inv.TenantID, &inv.InvoiceNumber, &inv.StripeInvoiceID,
			&inv.Amount, &inv.AmountDue, &inv.AmountPaid, &inv.Currency,
			&inv.Status, &inv.DueDate, &inv.PaidAt, &inv.InvoiceURL, &inv.InvoicePDF,
			&inv.CreatedAt, &inv.UpdatedAt,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		invoices = append(invoices, inv)
	}

	c.JSON(http.StatusOK, gin.H{"invoices": invoices})
}

func (h *BillingHandler) GetInvoice(c *gin.Context) {
	invoiceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invoice ID"})
		return
	}

	var inv models.Invoice
	err = database.Pool.QueryRow(c.Request.Context(), `
		SELECT id, tenant_id, invoice_number, stripe_invoice_id, amount, amount_due, 
			amount_paid, currency, status, due_date, paid_at, invoice_url, invoice_pdf, 
			created_at, updated_at
		FROM invoices WHERE id = $1
	`).Scan(
		&inv.ID, &inv.TenantID, &inv.InvoiceNumber, &inv.StripeInvoiceID,
		&inv.Amount, &inv.AmountDue, &inv.AmountPaid, &inv.Currency,
		&inv.Status, &inv.DueDate, &inv.PaidAt, &inv.InvoiceURL, &inv.InvoicePDF,
		&inv.CreatedAt, &inv.UpdatedAt,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
		return
	}

	// Get line items
	rows, err := database.Pool.Query(c.Request.Context(), `
		SELECT id, invoice_id, description, quantity, unit_price, amount, created_at
		FROM line_items WHERE invoice_id = $1
	`, invoiceID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var item models.LineItem
			rows.Scan(&item.ID, &item.InvoiceID, &item.Description, &item.Quantity,
				&item.UnitPrice, &item.Amount, &item.CreatedAt)
			inv.LineItems = append(inv.LineItems, item)
		}
	}

	c.JSON(http.StatusOK, inv)
}
