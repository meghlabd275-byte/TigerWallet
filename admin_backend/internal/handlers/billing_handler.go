package handlers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type BillingHandler struct {
	db interface {
		Query(ctx context.Context, query string, args ...interface{}) (interface{}, error)
		Exec(ctx context.Context, query string, args ...interface{}) (interface{}, error)
	}
}

type SubscriptionPlan struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Price       float64   `json:"price"`
	Period      string    `json:"period"`
	Features    []string `json:"features"`
	MaxUsers    int      `json:"max_users"`
	IsActive    bool     `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Subscription struct {
	ID                   uuid.UUID `json:"id"`
	PlanID              uuid.UUID `json:"plan_id"`
	PlanName            string    `json:"plan_name"`
	Status              string    `json:"status"`
	CurrentPeriodStart  time.Time `json:"current_period_start"`
	CurrentPeriodEnd    time.Time `json:"current_period_end"`
	Users               int       `json:"users"`
	APICalls            int64     `json:"api_calls"`
	CreatedAt           time.Time `json:"created_at"`
}

type Invoice struct {
	ID              uuid.UUID `json:"id"`
	InvoiceNumber  string    `json:"invoice_number"`
	SubscriptionID uuid.UUID `json:"subscription_id"`
	Amount         float64   `json:"amount"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	PaidAt         *time.Time `json:"paid_at"`
}

type PaymentMethod struct {
	ID            uuid.UUID `json:"id"`
	Type          string    `json:"type"`
	Last4         string    `json:"last4"`
	ExpiryMonth   string    `json:"expiry_month"`
	ExpiryYear    string    `json:"expiry_year"`
	IsDefault     bool      `json:"is_default"`
	CreatedAt     time.Time `json:"created_at"`
}

func NewBillingHandler() *BillingHandler {
	return &BillingHandler{}
}

func (h *BillingHandler) GetPlans(c *gin.Context) {
	plans := []SubscriptionPlan{
		{
			ID:       uuid.New(),
			Name:     "Basic",
			Price:    99.0,
			Period:   "month",
			Features: []string{"Up to 1,000 users", "Basic analytics", "Email support"},
			MaxUsers: 1000,
			IsActive: true,
		},
		{
			ID:       uuid.New(),
			Name:     "Pro",
			Price:    299.0,
			Period:   "month",
			Features: []string{"Up to 10,000 users", "Advanced analytics", "Priority support", "API access"},
			MaxUsers: 10000,
			IsActive: true,
		},
		{
			ID:       uuid.New(),
			Name:     "Enterprise",
			Price:    999.0,
			Period:   "month",
			Features: []string{"Unlimited users", "Custom analytics", "24/7 support", "Full API access"},
			MaxUsers: -1,
			IsActive: true,
		},
	}

	c.JSON(200, gin.H{
		"success": true,
		"data":    plans,
	})
}

func (h *BillingHandler) GetSubscription(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}

	subscription := Subscription{
		ID:                  uuid.New(),
		PlanID:             uuid.New(),
		PlanName:           "Pro",
		Status:             "active",
		CurrentPeriodStart: time.Now().AddDate(0, 0, -15),
		CurrentPeriodEnd:   time.Now().AddDate(0, 0, 15),
		Users:              2500,
		APICalls:           50000,
		CreatedAt:          time.Now(),
	}

	c.JSON(200, gin.H{
		"success": true,
		"data":    subscription,
	})
}

func (h *BillingHandler) CreateSubscription(c *gin.Context) {
	var req struct {
		PlanID string `json:"plan_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}

	planID, err := uuid.Parse(req.PlanID)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid plan_id"})
		return
	}

	subscription := Subscription{
		ID:                  uuid.New(),
		PlanID:             planID,
		PlanName:           "Pro",
		Status:             "active",
		CurrentPeriodStart: time.Now(),
		CurrentPeriodEnd:   time.Now().AddDate(0, 1, 0),
		Users:              0,
		APICalls:           0,
		CreatedAt:          time.Now(),
	}

	c.JSON(201, gin.H{
		"success": true,
		"data":    subscription,
	})
}

func (h *BillingHandler) CancelSubscription(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "subscription cancelled",
	})
}

func (h *BillingHandler) GetInvoices(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}

	now := time.Now()
	invoices := []Invoice{
		{
			ID:             uuid.New(),
			InvoiceNumber:  "INV-001",
			SubscriptionID: uuid.New(),
			Amount:         299.0,
			Status:         "paid",
			CreatedAt:      now.AddDate(0, 0, -30),
			PaidAt:         &now,
		},
		{
			ID:             uuid.New(),
			InvoiceNumber:  "INV-002",
			SubscriptionID: uuid.New(),
			Amount:         299.0,
			Status:         "paid",
			CreatedAt:      now.AddDate(0, 0, -60),
			PaidAt:         &now,
		},
	}

	c.JSON(200, gin.H{
		"success": true,
		"data":    invoices,
	})
}

func (h *BillingHandler) GetPaymentMethods(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}

	methods := []PaymentMethod{
		{
			ID:           uuid.New(),
			Type:         "card",
			Last4:        "4242",
			ExpiryMonth:  "12",
			ExpiryYear:   "2025",
			IsDefault:    true,
			CreatedAt:    time.Now(),
		},
	}

	c.JSON(200, gin.H{
		"success": true,
		"data":    methods,
	})
}

func (h *BillingHandler) AddPaymentMethod(c *gin.Context) {
	var req struct {
		Type        string `json:"type" binding:"required"`
		Last4       string `json:"last4" binding:"required"`
		ExpiryMonth string `json:"expiry_month" binding:"required"`
		ExpiryYear  string `json:"expiry_year" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}

	method := PaymentMethod{
		ID:           uuid.New(),
		Type:         req.Type,
		Last4:        req.Last4,
		ExpiryMonth:  req.ExpiryMonth,
		ExpiryYear:   req.ExpiryYear,
		IsDefault:    false,
		CreatedAt:    time.Now(),
	}

	c.JSON(201, gin.H{
		"success": true,
		"data":    method,
	})
}

func (h *BillingHandler) DeletePaymentMethod(c *gin.Context) {
	methodID := c.Param("id")
	if methodID == "" {
		c.JSON(400, gin.H{"error": "method_id required"})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "payment method deleted",
	})
}

func (h *BillingHandler) SetDefaultPaymentMethod(c *gin.Context) {
	methodID := c.Param("id")
	if methodID == "" {
		c.JSON(400, gin.H{"error": "method_id required"})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "default payment method set",
	})
}

func hashPassword(password string) string {
	bytes, _ := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes)
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (h *BillingHandler) CreatePlan(c *gin.Context) {
	var req struct {
		Name     string   `json:"name" binding:"required"`
		Price    float64  `json:"price" binding:"required"`
		Period   string   `json:"period" binding:"required"`
		Features []string `json:"features"`
		MaxUsers int      `json:"max_users"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	plan := SubscriptionPlan{
		ID:       uuid.New(),
		Name:     req.Name,
		Price:    req.Price,
		Period:   req.Period,
		Features: req.Features,
		MaxUsers: req.MaxUsers,
		IsActive: true,
	}

	c.JSON(201, gin.H{
		"success": true,
		"data":    plan,
	})
}

func (h *BillingHandler) UpdatePlan(c *gin.Context) {
	planID := c.Param("id")
	if planID == "" {
		c.JSON(400, gin.H{"error": "plan_id required"})
		return
	}

	var req struct {
		Name     string   `json:"name"`
		Price    float64  `json:"price"`
		Features []string `json:"features"`
		MaxUsers int      `json:"max_users"`
		IsActive bool     `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "plan updated",
	})
}

func (h *BillingHandler) DeletePlan(c *gin.Context) {
	planID := c.Param("id")
	if planID == "" {
		c.JSON(400, gin.H{"error": "plan_id required"})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "plan deleted",
	})
}
