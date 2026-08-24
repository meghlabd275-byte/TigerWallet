package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/tigerwallet/admin/pkg/database"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// BillingHandler handles subscription billing backed by real Postgres rows.
// Invoices are created as real records in "open" status; payment collection
// is performed against the stored tokenized payment method by an external
// processor integration and only then marked paid. Nothing here fabricates
// payment state.
type BillingHandler struct {
	db *database.PostgresDB
}

type SubscriptionPlan struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name      string    `gorm:"size:100;not null;uniqueIndex" json:"name"`
	Price     float64   `gorm:"not null" json:"price"`
	Period    string    `gorm:"size:20;not null;default:'month'" json:"period"`
	Features  string    `gorm:"type:jsonb;default:'[]'" json:"features"`
	MaxUsers  int       `gorm:"not null;default:0" json:"max_users"`
	IsActive  bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (SubscriptionPlan) TableName() string { return "billing_plans" }

type Subscription struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID             string     `gorm:"size:64;not null;index" json:"user_id"`
	PlanID             uuid.UUID  `gorm:"type:uuid;not null" json:"plan_id"`
	PlanName           string     `gorm:"size:100;not null" json:"plan_name"`
	Status             string     `gorm:"size:20;not null;default:'active';index" json:"status"`
	CurrentPeriodStart time.Time  `gorm:"not null" json:"current_period_start"`
	CurrentPeriodEnd   time.Time  `gorm:"not null" json:"current_period_end"`
	CanceledAt         *time.Time `json:"canceled_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

func (Subscription) TableName() string { return "billing_subscriptions" }

type Invoice struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	InvoiceNumber  string     `gorm:"size:40;not null;uniqueIndex" json:"invoice_number"`
	SubscriptionID uuid.UUID  `gorm:"type:uuid;not null;index" json:"subscription_id"`
	Amount         float64    `gorm:"not null" json:"amount"`
	Status         string     `gorm:"size:20;not null;default:'open';index" json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	PaidAt         *time.Time `json:"paid_at,omitempty"`
}

func (Invoice) TableName() string { return "billing_invoices" }

// PaymentMethod stores only a tokenized card reference (brand/last4/expiry).
// Full card numbers are never accepted or stored by this service.
type PaymentMethod struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID      string    `gorm:"size:64;not null;index" json:"user_id"`
	Type        string    `gorm:"size:20;not null" json:"type"`
	Brand       string    `gorm:"size:30" json:"brand"`
	Last4       string    `gorm:"size:4;not null" json:"last4"`
	ExpiryMonth string    `gorm:"size:2;not null" json:"expiry_month"`
	ExpiryYear  string    `gorm:"size:4;not null" json:"expiry_year"`
	IsDefault   bool      `gorm:"not null;default:false" json:"is_default"`
	CreatedAt   time.Time `json:"created_at"`
}

func (PaymentMethod) TableName() string { return "billing_payment_methods" }

func NewBillingHandler(db *database.PostgresDB) (*BillingHandler, error) {
	h := &BillingHandler{db: db}
	if err := db.DB.AutoMigrate(&SubscriptionPlan{}, &Subscription{}, &Invoice{}, &PaymentMethod{}); err != nil {
		return nil, fmt.Errorf("billing migration failed: %w", err)
	}
	if err := h.seedDefaultPlans(); err != nil {
		return nil, err
	}
	return h, nil
}

func (h *BillingHandler) seedDefaultPlans() error {
	var count int64
	if err := h.db.DB.Model(&SubscriptionPlan{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	defaults := []SubscriptionPlan{
		{ID: uuid.New(), Name: "Basic", Price: 99.0, Period: "month", Features: `["Up to 1,000 users","Basic analytics","Email support"]`, MaxUsers: 1000, IsActive: true},
		{ID: uuid.New(), Name: "Pro", Price: 299.0, Period: "month", Features: `["Up to 10,000 users","Advanced analytics","Priority support","API access"]`, MaxUsers: 10000, IsActive: true},
		{ID: uuid.New(), Name: "Enterprise", Price: 999.0, Period: "month", Features: `["Unlimited users","Custom analytics","24/7 support","Full API access"]`, MaxUsers: -1, IsActive: true},
	}
	return h.db.DB.Create(&defaults).Error
}

func (h *BillingHandler) GetPlans(c *gin.Context) {
	var plans []SubscriptionPlan
	if err := h.db.DB.Where("is_active = ?", true).Order("price ASC").Find(&plans).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": plans})
}

func (h *BillingHandler) CreatePlan(c *gin.Context) {
	var req struct {
		Name     string   `json:"name" binding:"required"`
		Price    float64  `json:"price" binding:"required"`
		Period   string   `json:"period"`
		Features []string `json:"features"`
		MaxUsers int      `json:"max_users"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Price < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "price must be non-negative"})
		return
	}
	period := req.Period
	if period == "" {
		period = "month"
	}
	features := "[]"
	if len(req.Features) > 0 {
		b, err := json.Marshal(req.Features)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid features"})
			return
		}
		features = string(b)
	}
	plan := SubscriptionPlan{
		ID:       uuid.New(),
		Name:     req.Name,
		Price:    req.Price,
		Period:   period,
		Features: features,
		MaxUsers: req.MaxUsers,
		IsActive: true,
	}
	if err := h.db.DB.Create(&plan).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "plan with this name already exists"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": plan})
}

func (h *BillingHandler) UpdatePlan(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan id"})
		return
	}
	var req struct {
		Name     *string   `json:"name"`
		Price    *float64  `json:"price"`
		Period   *string   `json:"period"`
		Features *[]string `json:"features"`
		MaxUsers *int      `json:"max_users"`
		IsActive *bool     `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Price != nil {
		if *req.Price < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "price must be non-negative"})
			return
		}
		updates["price"] = *req.Price
	}
	if req.Period != nil {
		updates["period"] = *req.Period
	}
	if req.Features != nil {
		b, err := json.Marshal(*req.Features)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid features"})
			return
		}
		updates["features"] = string(b)
	}
	if req.MaxUsers != nil {
		updates["max_users"] = *req.MaxUsers
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}
	res := h.db.DB.Model(&SubscriptionPlan{}).Where("id = ?", id).Updates(updates)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		return
	}
	var plan SubscriptionPlan
	h.db.DB.First(&plan, "id = ?", id)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": plan})
}

func (h *BillingHandler) DeletePlan(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan id"})
		return
	}
	var active int64
	h.db.DB.Model(&Subscription{}).Where("plan_id = ? AND status = ?", id, "active").Count(&active)
	if active > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "plan has active subscriptions and cannot be deleted"})
		return
	}
	res := h.db.DB.Delete(&SubscriptionPlan{}, "id = ?", id)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "plan deleted"})
}

func (h *BillingHandler) GetSubscription(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var sub Subscription
	if err := h.db.DB.Where("user_id = ? AND status = ?", userID, "active").Order("created_at DESC").First(&sub).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active subscription"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": sub})
}

func (h *BillingHandler) CreateSubscription(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req struct {
		PlanID string `json:"plan_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	planID, err := uuid.Parse(req.PlanID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan id"})
		return
	}
	var plan SubscriptionPlan
	if err := h.db.DB.First(&plan, "id = ? AND is_active = ?", planID, true).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "plan not found or inactive"})
		return
	}

	now := time.Now()
	sub := Subscription{
		ID:                 uuid.New(),
		UserID:             userID,
		PlanID:             plan.ID,
		PlanName:           plan.Name,
		Status:             "active",
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   addPeriod(now, plan.Period),
	}

	tx := h.db.DB.Begin()
	if err := tx.Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Cancel any existing active subscription before activating the new one.
	if err := tx.Model(&Subscription{}).
		Where("user_id = ? AND status = ?", userID, "active").
		Updates(map[string]interface{}{"status": "canceled", "canceled_at": now}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := tx.Create(&sub).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Real invoice record in "open" status. It is only marked paid after an
	// actual charge is confirmed by the payment processor integration.
	invoice := Invoice{
		ID:             uuid.New(),
		InvoiceNumber:  fmt.Sprintf("INV-%s", sub.ID.String()[:8]),
		SubscriptionID: sub.ID,
		Amount:         plan.Price,
		Status:         "open",
	}
	if err := tx.Create(&invoice).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"subscription": sub,
			"invoice":      invoice,
		},
	})
}

func (h *BillingHandler) CancelSubscription(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	now := time.Now()
	res := h.db.DB.Model(&Subscription{}).
		Where("user_id = ? AND status = ?", userID, "active").
		Updates(map[string]interface{}{"status": "canceled", "canceled_at": now})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active subscription"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "subscription canceled"})
}

func (h *BillingHandler) GetInvoices(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var invoices []Invoice
	err := h.db.DB.Joins("JOIN billing_subscriptions s ON s.id = billing_invoices.subscription_id").
		Where("s.user_id = ?", userID).
		Order("billing_invoices.created_at DESC").
		Find(&invoices).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": invoices})
}

func (h *BillingHandler) GetPaymentMethods(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var methods []PaymentMethod
	if err := h.db.DB.Where("user_id = ?", userID).Order("is_default DESC, created_at ASC").Find(&methods).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": methods})
}

func (h *BillingHandler) AddPaymentMethod(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req struct {
		Type        string `json:"type" binding:"required"`
		Brand       string `json:"brand"`
		Last4       string `json:"last4" binding:"required,len=4"`
		ExpiryMonth string `json:"expiry_month" binding:"required,len=2"`
		ExpiryYear  string `json:"expiry_year" binding:"required,len=4"`
		IsDefault   bool   `json:"is_default"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Type != "card" && req.Type != "bank" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be card or bank"})
		return
	}

	pm := PaymentMethod{
		ID:          uuid.New(),
		UserID:      userID,
		Type:        req.Type,
		Brand:       req.Brand,
		Last4:       req.Last4,
		ExpiryMonth: req.ExpiryMonth,
		ExpiryYear:  req.ExpiryYear,
	}

	tx := h.db.DB.Begin()
	var count int64
	tx.Model(&PaymentMethod{}).Where("user_id = ?", userID).Count(&count)
	// First method becomes the default automatically.
	pm.IsDefault = req.IsDefault || count == 0
	if pm.IsDefault {
		if err := tx.Model(&PaymentMethod{}).Where("user_id = ?", userID).Update("is_default", false).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if err := tx.Create(&pm).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": pm})
}

func (h *BillingHandler) DeletePaymentMethod(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payment method id"})
		return
	}
	var pm PaymentMethod
	if err := h.db.DB.First(&pm, "id = ? AND user_id = ?", id, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment method not found"})
		return
	}
	if pm.IsDefault {
		c.JSON(http.StatusConflict, gin.H{"error": "cannot delete the default payment method; set another default first"})
		return
	}
	if err := h.db.DB.Delete(&pm).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "payment method deleted"})
}

func (h *BillingHandler) SetDefaultPaymentMethod(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payment method id"})
		return
	}
	tx := h.db.DB.Begin()
	res := tx.Model(&PaymentMethod{}).Where("id = ? AND user_id = ?", id, userID).Update("is_default", true)
	if res.Error != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
		return
	}
	if res.RowsAffected == 0 {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "payment method not found"})
		return
	}
	if err := tx.Model(&PaymentMethod{}).Where("user_id = ? AND id != ?", userID, id).Update("is_default", false).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "default payment method updated"})
}

func addPeriod(from time.Time, period string) time.Time {
	switch period {
	case "year":
		return from.AddDate(1, 0, 0)
	case "week":
		return from.AddDate(0, 0, 7)
	default:
		return from.AddDate(0, 1, 0)
	}
}
