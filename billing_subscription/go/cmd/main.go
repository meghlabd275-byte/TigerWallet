// Billing & Subscription Service - Go Implementation
// High-performance, distributed billing system for TigerWallet ecosystem

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Configuration
type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
	JWTSecret   string
}

// ============ DATA MODELS ============

type PlanTier string

const (
	TierFree     PlanTier = "free"
	TierStarter PlanTier = "starter"
	TierBusiness PlanTier = "business"
	TierEnterprise PlanTier = "enterprise"
)

type BillingCycle string

const (
	CycleMonthly  BillingCycle = "monthly"
	CycleYearly  BillingCycle = "yearly"
	CycleLifetime BillingCycle = "lifetime"
)

type SubscriptionStatus string

const (
	SubActive    SubscriptionStatus = "active"
	SubTrial    SubscriptionStatus = "trial"
	SubPastDue  SubscriptionStatus = "past_due"
	SubCancelled SubscriptionStatus = "cancelled"
	SubExpired   SubscriptionStatus = "expired"
)

// Subscription Plan
type Plan struct {
	ID               uuid.UUID     `json:"id"`
	Name            string       `json:"name"`
	Tier            PlanTier     `json:"tier"`
	Description     string       `json:"description"`
	PriceMonthly    float64     `json:"price_monthly"`
	PriceYearly     float64     `json:"price_yearly"`
	Features        string       `json:"features"` // JSON array
	Limits          string       `json:"limits"` // JSON object
	IsActive        bool         `json:"is_active"`
	CreatedAt       time.Time    `json:"created_at"`
}

// Client Subscription
type Subscription struct {
	ID              uuid.UUID         `json:"id"`
	ClientID        uuid.UUID         `json:"client_id"`
	PlanID          uuid.UUID         `json:"plan_id"`
	Status          SubscriptionStatus `json:"status"`
	BillingCycle    BillingCycle      `json:"billing_cycle"`
	Amount          float64           `json:"amount"`
	Currency        string            `json:"currency"` // USD
	StartDate       time.Time         `json:"start_date"`
	EndDate         *time.Time        `json:"end_date"`
	TrialEndDate    *time.Time        `json:"trial_end_date"`
	AutoRenew       bool              `json:"auto_renew"`
	PaymentMethod   string            `json:"payment_method"` // card, bank, crypto
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// Usage Record
type UsageRecord struct {
	ID             uuid.UUID   `json:"id"`
	SubscriptionID uuid.UUID   `json:"subscription_id"`
	ClientID       uuid.UUID   `json:"client_id"`
	Metric         string     `json:"metric"` // api_calls, storage, bandwidth
	Count          int64      `json:"count"`
	Unit           string     `json:"unit"`
	Period         string     `json:"period"` // daily, monthly
	Timestamp      time.Time  `json:"timestamp"`
}

// Invoice
type Invoice struct {
	ID              uuid.UUID   `json:"id"`
	SubscriptionID  uuid.UUID   `json:"subscription_id"`
	ClientID       uuid.UUID   `json:"client_id"`
	InvoiceNumber  string     `json:"invoice_number"`
	Amount         float64    `json:"amount"`
	Currency       string     `json:"currency"`
	Status         string     `json:"status"` // draft, pending, paid, failed, refunded
	DueDate        time.Time  `json:"due_date"`
	PaidAt         *time.Time `json:"paid_at"`
	Items          string     `json:"items"` // JSON array
	CreatedAt      time.Time  `json:"created_at"`
}

// Payment
type Payment struct {
	ID              uuid.UUID   `json:"id"`
	InvoiceID       uuid.UUID   `json:"invoice_id"`
	ClientID        uuid.UUID   `json:"client_id"`
	Amount          float64    `json:"amount"`
	Currency        string     `json:"currency"`
	Status          string     `json:"status"` // pending, completed, failed, refunded
	PaymentMethod   string     `json:"payment_method"`
	TransactionID   string     `json:"transaction_id"`
	Metadata        string     `json:"metadata"` // JSON
	ProcessedAt     *time.Time `json:"processed_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

// Global variables
var (
	db     *pgxpool.Pool
	redis  *redis.Client
	config Config
	logger *log.Logger
	jwtSecret []byte
)

// ============ INITIALIZATION ============

func initDatabase() error {
	var err error
	dbURL := getEnv("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet_admin")

	db, err = pgxpool.New(context.Background(), dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	if err = db.Ping(context.Background()); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Create tables
	_, err = db.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS billing_plans (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			tier VARCHAR(50) NOT NULL,
			description TEXT,
			price_monthly DECIMAL(10,2) NOT NULL,
			price_yearly DECIMAL(10,2) NOT NULL,
			features JSONB,
			limits JSONB,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS subscriptions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			client_id UUID NOT NULL,
			plan_id UUID REFERENCES billing_plans(id),
			status VARCHAR(50) DEFAULT 'trial',
			billing_cycle VARCHAR(50) DEFAULT 'monthly',
			amount DECIMAL(10,2) NOT NULL,
			currency VARCHAR(10) DEFAULT 'USD',
			start_date TIMESTAMP NOT NULL,
			end_date TIMESTAMP,
			trial_end_date TIMESTAMP,
			auto_renew BOOLEAN DEFAULT true,
			payment_method VARCHAR(50),
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS usage_records (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			subscription_id UUID REFERENCES subscriptions(id),
			client_id UUID NOT NULL,
			metric VARCHAR(100) NOT NULL,
			count BIGINT DEFAULT 0,
			unit VARCHAR(50),
			period VARCHAR(50),
			timestamp TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS invoices (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			subscription_id UUID REFERENCES subscriptions(id),
			client_id UUID NOT NULL,
			invoice_number VARCHAR(50) UNIQUE NOT NULL,
			amount DECIMAL(10,2) NOT NULL,
			currency VARCHAR(10) DEFAULT 'USD',
			status VARCHAR(50) DEFAULT 'draft',
			due_date TIMESTAMP NOT NULL,
			paid_at TIMESTAMP,
			items JSONB,
			created_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS payments (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			invoice_id UUID REFERENCES invoices(id),
			client_id UUID NOT NULL,
			amount DECIMAL(10,2) NOT NULL,
			currency VARCHAR(10) DEFAULT 'USD',
			status VARCHAR(50) DEFAULT 'pending',
			payment_method VARCHAR(50),
			transaction_id VARCHAR(255),
			metadata JSONB,
			processed_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_subscriptions_client ON subscriptions(client_id);
		CREATE INDEX IF NOT EXISTS idx_usage_client ON usage_records(client_id);
		CREATE INDEX IF NOT EXISTS idx_invoices_client ON invoices(client_id);
		CREATE INDEX IF NOT EXISTS idx_payments_client ON payments(client_id);
	`)

	// Insert default plans
	_, err = db.Exec(context.Background(), `
		INSERT INTO billing_plans (name, tier, description, price_monthly, price_yearly, features, limits)
		VALUES 
			('Free', 'free', 'Free tier for testing', 0, 0, '["basic_api", "limited_users"]', '{"api_calls": 1000, "storage_mb": 100}'),
			('Starter', 'starter', 'Starter plan for small teams', 29.99, 299.99, '["full_api", "unlimited_users", "email_support"]', '{"api_calls": 10000, "storage_mb": 1000}'),
			('Business', 'business', 'Business plan with advanced features', 99.99, 999.99, '["full_api", "priority_support", "analytics", "webhooks"]', '{"api_calls": 100000, "storage_mb": 10000}'),
			('Enterprise', 'enterprise', 'Enterprise plan with custom solutions', 299.99, 2999.99, '["full_api", "dedicated_support", "custom_integrations", "sla"]', '{"api_calls": -1, "storage_mb": -1}')
		ON CONFLICT DO NOTHING
	`)

	return err
}

func initRedis() error {
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379")
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return err
	}
	redis = redis.NewClient(opt)
	return redis.Ping(context.Background()).Err()
}

// ============ HTTP HANDLERS ============

func HealthCheck(c *gin.Context) {
	ctx := context.Background()
	dbStatus := "healthy"
	if err := db.Ping(ctx); err != nil {
		dbStatus = "unhealthy"
	}
	
	redisStatus := "healthy"
	if err := redis.Ping(ctx).Err(); err != nil {
		redisStatus = "unhealthy"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "ok",
		"database": dbStatus,
		"redis":    redisStatus,
	})
}

// GetPlans - Get all billing plans
func GetPlans(c *gin.Context) {
	rows, err := db.Query(context.Background(), `
		SELECT id, name, tier, description, price_monthly, price_yearly, features, limits, is_active, created_at
		FROM billing_plans WHERE is_active = true ORDER BY price_monthly ASC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var plans []Plan
	for rows.Next() {
		var plan Plan
		var features, limits []byte
		if err := rows.Scan(&plan.ID, &plan.Name, &plan.Tier, &plan.Description, &plan.PriceMonthly, &plan.PriceYearly, &features, &limits, &plan.IsActive, &plan.CreatedAt); err != nil {
			continue
		}
		json.Unmarshal(features, &plan.Features)
		json.Unmarshal(limits, &plan.Limits)
		plans = append(plans, plan)
	}

	c.JSON(http.StatusOK, gin.H{"plans": plans})
}

// CreateSubscription - Create new subscription
func CreateSubscription(c *gin.Context) {
	var req struct {
		ClientID      uuid.UUID      `json:"client_id" binding:"required"`
		PlanID        uuid.UUID      `json:"plan_id" binding:"required"`
		BillingCycle  BillingCycle   `json:"billing_cycle"`
		PaymentMethod string         `json:"payment_method"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get plan details
	var plan Plan
	var priceMonthly float64
	err := db.QueryRow(context.Background(), `
		SELECT id, name, tier, price_monthly, price_yearly FROM billing_plans WHERE id = $1
	`, req.PlanID).Scan(&plan.ID, &plan.Name, &plan.Tier, &plan.PriceMonthly, &plan.PriceYearly)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		return
	}

	amount := plan.PriceMonthly
	if req.BillingCycle == CycleYearly {
		amount = plan.PriceYearly
	}

	subscription := Subscription{
		ID:             uuid.New(),
		ClientID:       req.ClientID,
		PlanID:         req.PlanID,
		Status:         SubTrial,
		BillingCycle:   req.BillingCycle,
		Amount:         amount,
		Currency:       "USD",
		StartDate:      time.Now(),
		TrialEndDate:   &[]time.Time{time.Now().Add(14 * 24 * time.Hour)}[0],
		AutoRenew:      true,
		PaymentMethod:  req.PaymentMethod,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	_, err = db.Exec(context.Background(), `
		INSERT INTO subscriptions (id, client_id, plan_id, status, billing_cycle, amount, currency, start_date, trial_end_date, auto_renew, payment_method, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, subscription.ID, subscription.ClientID, subscription.PlanID, subscription.Status, subscription.BillingCycle, subscription.Amount, subscription.Currency, subscription.StartDate, subscription.TrialEndDate, subscription.AutoRenew, subscription.PaymentMethod, subscription.CreatedAt, subscription.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, subscription)
}

// GetSubscription - Get client subscription
func GetSubscription(c *gin.Context) {
	clientID := c.Param("client_id")
	id, err := uuid.Parse(clientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client ID"})
		return
	}

	var sub Subscription
	err = db.QueryRow(context.Background(), `
		SELECT id, client_id, plan_id, status, billing_cycle, amount, currency, start_date, end_date, trial_end_date, auto_renew, payment_method, created_at, updated_at
		FROM subscriptions WHERE client_id = $1 ORDER BY created_at DESC LIMIT 1
	`, id).Scan(&sub.ID, &sub.ClientID, &sub.PlanID, &sub.Status, &sub.BillingCycle, &sub.Amount, &sub.Currency, &sub.StartDate, &sub.EndDate, &sub.TrialEndDate, &sub.AutoRenew, &sub.PaymentMethod, &sub.CreatedAt, &sub.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}

	c.JSON(http.StatusOK, sub)
}

// RecordUsage - Record API usage
func RecordUsage(c *gin.Context) {
	var req struct {
		SubscriptionID uuid.UUID `json:"subscription_id" binding:"required"`
		ClientID       uuid.UUID `json:"client_id" binding:"required"`
		Metric         string    `json:"metric" binding:"required"`
		Count          int64     `json:"count"`
		Unit           string    `json:"unit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	usage := UsageRecord{
		ID:             uuid.New(),
		SubscriptionID: req.SubscriptionID,
		ClientID:       req.ClientID,
		Metric:         req.Metric,
		Count:          req.Count,
		Unit:           req.Unit,
		Period:         "daily",
		Timestamp:      time.Now(),
	}

	_, err := db.Exec(context.Background(), `
		INSERT INTO usage_records (id, subscription_id, client_id, metric, count, unit, period, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, usage.ID, usage.SubscriptionID, usage.ClientID, usage.Metric, usage.Count, usage.Unit, usage.Period, usage.Timestamp)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update Redis counter
	redis.IncrBy(context.Background(), fmt.Sprintf("usage:%s:%s", req.ClientID, req.Metric), req.Count)

	c.JSON(http.StatusCreated, gin.H{"message": "usage recorded"})
}

// GetUsage - Get usage for client
func GetUsage(c *gin.Context) {
	clientID := c.Param("client_id")
	id, err := uuid.Parse(clientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client ID"})
		return
	}

	rows, err := db.Query(context.Background(), `
		SELECT id, metric, count, unit, period, timestamp
		FROM usage_records WHERE client_id = $1 ORDER BY timestamp DESC LIMIT 100
	`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var usage []UsageRecord
	for rows.Next() {
		var u UsageRecord
		if err := rows.Scan(&u.ID, &u.Metric, &u.Count, &u.Unit, &u.Period, &u.Timestamp); err != nil {
			continue
		}
		usage = append(usage, u)
	}

	c.JSON(http.StatusOK, gin.H{"usage": usage})
}

// CreateInvoice - Create invoice
func CreateInvoice(c *gin.Context) {
	var req struct {
		SubscriptionID uuid.UUID `json:"subscription_id" binding:"required"`
		ClientID       uuid.UUID `json:"client_id" binding:"required"`
		Amount         float64   `json:"amount" binding:"required"`
		Items          string    `json:"items"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate invoice number
	invoiceNum := fmt.Sprintf("INV-%d-%s", time.Now().Unix(), uuid.New().String()[:8])

	invoice := Invoice{
		ID:              uuid.New(),
		SubscriptionID:  req.SubscriptionID,
		ClientID:        req.ClientID,
		InvoiceNumber:   invoiceNum,
		Amount:          req.Amount,
		Currency:        "USD",
		Status:          "pending",
		DueDate:         time.Now().Add(30 * 24 * time.Hour),
		Items:           req.Items,
		CreatedAt:       time.Now(),
	}

	_, err := db.Exec(context.Background(), `
		INSERT INTO invoices (id, subscription_id, client_id, invoice_number, amount, currency, status, due_date, items, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, invoice.ID, invoice.SubscriptionID, invoice.ClientID, invoice.InvoiceNumber, invoice.Amount, invoice.Currency, invoice.Status, invoice.DueDate, invoice.Items, invoice.CreatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, invoice)
}

// GetInvoices - Get client invoices
func GetInvoices(c *gin.Context) {
	clientID := c.Param("client_id")
	id, err := uuid.Parse(clientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client ID"})
		return
	}

	rows, err := db.Query(context.Background(), `
		SELECT id, invoice_number, amount, currency, status, due_date, paid_at, created_at
		FROM invoices WHERE client_id = $1 ORDER BY created_at DESC
	`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var invoices []Invoice
	for rows.Next() {
		var inv Invoice
		if err := rows.Scan(&inv.ID, &inv.InvoiceNumber, &inv.Amount, &inv.Currency, &inv.Status, &inv.DueDate, &inv.PaidAt, &inv.CreatedAt); err != nil {
			continue
		}
		invoices = append(invoices, inv)
	}

	c.JSON(http.StatusOK, gin.H{"invoices": invoices})
}

// ProcessPayment - Process payment
func ProcessPayment(c *gin.Context) {
	var req struct {
		InvoiceID     uuid.UUID `json:"invoice_id" binding:"required"`
		ClientID      uuid.UUID `json:"client_id" binding:"required"`
		Amount        float64   `json:"amount" binding:"required"`
		PaymentMethod string    `json:"payment_method" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payment := Payment{
		ID:              uuid.New(),
		InvoiceID:       req.InvoiceID,
		ClientID:        req.ClientID,
		Amount:          req.Amount,
		Currency:        "USD",
		Status:          "completed",
		PaymentMethod:   req.PaymentMethod,
		TransactionID:   fmt.Sprintf("TXN-%s", uuid.New().String()[:16]),
		ProcessedAt:     &[]time.Time{time.Now()}[0],
		CreatedAt:       time.Now(),
	}

	tx, err := db.Begin(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Insert payment
	_, err = tx.Exec(context.Background(), `
		INSERT INTO payments (id, invoice_id, client_id, amount, currency, status, payment_method, transaction_id, processed_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, payment.ID, payment.InvoiceID, payment.ClientID, payment.Amount, payment.Currency, payment.Status, payment.PaymentMethod, payment.TransactionID, payment.ProcessedAt, payment.CreatedAt)

	if err != nil {
		tx.Rollback(context.Background())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update invoice status
	_, err = tx.Exec(context.Background(), `
		UPDATE invoices SET status = 'paid', paid_at = NOW() WHERE id = $1
	`, req.InvoiceID)

	if err != nil {
		tx.Rollback(context.Background())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	tx.Commit(context.Background())

	c.JSON(http.StatusCreated, payment)
}

// CancelSubscription - Cancel subscription
func CancelSubscription(c *gin.Context) {
	subscriptionID := c.Param("id")
	id, err := uuid.Parse(subscriptionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription ID"})
		return
	}

	_, err = db.Exec(context.Background(), `
		UPDATE subscriptions SET status = 'cancelled', auto_renew = false, updated_at = NOW() WHERE id = $1
	`, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "subscription cancelled"})
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// ============ MAIN ============

func main() {
	logger = log.New(os.Stdout, "Billing Service: ", log.LstdFlags)
	logger.Println("Starting Billing & Subscription Service...")

	config.Port = getEnv("BILLING_PORT", "8100")
	config.DatabaseURL = getEnv("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet_admin")
	config.RedisURL = getEnv("REDIS_URL", "redis://localhost:6379")
	config.JWTSecret = getEnv("JWT_SECRET", "tigerwallet-billing-secret")
	jwtSecret = []byte(config.JWTSecret)

	if err := initDatabase(); err != nil {
		logger.Fatalf("Failed to initialize database: %v", err)
	}
	logger.Println("Database connected")

	if err := initRedis(); err != nil {
		logger.Fatalf("Failed to initialize Redis: %v", err)
	}
	logger.Println("Redis connected")

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	router.GET("/health", HealthCheck)

	// Plans
	router.GET("/api/v1/plans", GetPlans)

	// Subscriptions
	router.POST("/api/v1/subscriptions", CreateSubscription)
	router.GET("/api/v1/subscriptions/:client_id", GetSubscription)
	router.DELETE("/api/v1/subscriptions/:id", CancelSubscription)

	// Usage
	router.POST("/api/v1/usage", RecordUsage)
	router.GET("/api/v1/usage/:client_id", GetUsage)

	// Invoices
	router.POST("/api/v1/invoices", CreateInvoice)
	router.GET("/api/v1/invoices/:client_id", GetInvoices)

	// Payments
	router.POST("/api/v1/payments", ProcessPayment)

	logger.Printf("Starting server on port %s", config.Port)
	srv := &http.Server{
		Addr:    ":" + config.Port,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	logger.Println("Server started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	db.Close()
	redis.Close()
	logger.Println("Server exited")
}
