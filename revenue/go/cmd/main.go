package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func main() {
	cfg := loadConfig()

	db, err := initDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	router := gin.Default()
	router.Use(corsMiddleware())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "tiger-revenue"})
	})

	api := router.Group("/api/v1")
	{
		// Transaction Fees
		api.POST("/fees/calculate", calculateFeeHandler)
		api.POST("/fees/collect", collectFeeHandler)
		api.GET("/fees/history/:tenant_id", getFeeHistoryHandler)

		// API Usage Billing
		api.POST("/usage/track", trackUsageHandler)
		api.GET("/usage/:tenant_id", getUsageHandler)
		api.GET("/usage/summary/:tenant_id", getUsageSummaryHandler)

		// Invoices
		api.POST("/invoices", createInvoiceHandler)
		api.GET("/invoices/:tenant_id", listInvoicesHandler)
		api.GET("/invoices/:id", getInvoiceHandler)
		api.POST("/invoices/:id/pay", payInvoiceHandler)

		// Subscriptions
		api.GET("/subscriptions/:tenant_id", getSubscriptionHandler)
		api.POST("/subscriptions/:tenant_id/upgrade", upgradeSubscriptionHandler)
		api.POST("/subscriptions/:tenant_id/downgrade", downgradeSubscriptionHandler)

		// Revenue Analytics
		api.GET("/analytics/revenue", getRevenueAnalyticsHandler)
		api.GET("/analytics/usage", getUsageAnalyticsHandler)
		api.GET("/analytics/tenants", getTenantAnalyticsHandler)

		// Super Admin
		superAdmin := api.Group("/super-admin")
		{
			superAdmin.GET("/dashboard", getDashboardHandler)
			superAdmin.GET("/reports/revenue", generateRevenueReportHandler)
		}
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: router,
	}

	go func() {
		log.Printf("Revenue service starting on port %s", cfg.Port)
		srv.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}

type Config struct {
	Port     string
	Database DatabaseConfig
	Stripe   StripeConfig
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

type StripeConfig struct {
	SecretKey string
}

func loadConfig() *Config {
	return &Config{
		Port: getEnv("REVENUE_PORT", "9010"),
		Database: DatabaseConfig{
			Host:     getEnv("REVENUE_DB_HOST", "localhost"),
			Port:     getEnvInt("REVENUE_DB_PORT", 5432),
			User:     getEnv("REVENUE_DB_USER", "tigerwallet"),
			Password: getEnv("REVENUE_DB_PASSWORD", "password"),
			DBName:   getEnv("REVENUE_DB_NAME", "tigerwallet_revenue"),
		},
		Stripe: StripeConfig{
			SecretKey: getEnv("STRIPE_SECRET_KEY", ""),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	var v int
	_, err := fmt.Sscan(os.Getenv(key), &v)
	if err != nil {
		return defaultValue
	}
	return v
}

type FeeConfig struct {
	TransactionFeePercent float64 `json:"transaction_fee_percent"`
	ApiCallFee           float64 `json:"api_call_fee"`
	MonthlyBaseFee      float64 `json:"monthly_base_fee"`
	WithdrawalFee       float64 `json:"withdrawal_fee"`
}

type TransactionFee struct {
	ID            uuid.UUID `json:"id" db:"id"`
	TenantID     uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Product      string    `json:"product" db:"product"`
	TransactionID string    `json:"transaction_id" db:"transaction_id"`
	Amount       float64   `json:"amount" db:"amount"`
	FeeAmount    float64   `json:"fee_amount" db:"fee_amount"`
	FeePercent   float64   `json:"fee_percent" db:"fee_percent"`
	NetAmount    float64   `json:"net_amount" db:"net_amount"`
	Status       string    `json:"status" db:"status"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type APIUsage struct {
	ID          uuid.UUID `json:"id" db:"id"`
	TenantID   uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Product    string    `json:"product" db:"product"`
	Endpoint   string    `json:"endpoint" db:"endpoint"`
	Method     string    `json:"method" db:"method"`
	StatusCode int       `json:"status_code" db:"status_code"`
	LatencyMs  int       `json:"latency_ms" db:"latency_ms"`
	Cost       float64   `json:"cost" db:"cost"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

type Invoice struct {
	ID              uuid.UUID `json:"id" db:"id"`
	TenantID       uuid.UUID `json:"tenant_id" db:"tenant_id"`
	InvoiceNumber  string    `json:"invoice_number" db:"invoice_number"`
	Type           string    `json:"type" db:"type"` // subscription, usage, transaction
	PeriodStart   time.Time `json:"period_start" db:"period_start"`
	PeriodEnd     time.Time `json:"period_end" db:"period_end"`
	Subtotal      float64   `json:"subtotal" db:"subtotal"`
	Tax           float64   `json:"tax" db:"tax"`
	Total         float64   `json:"total" db:"total"`
	Status        string    `json:"status" db:"status"` // pending, paid, overdue, cancelled
	DueDate       time.Time `json:"due_date" db:"due_date"`
	PaidAt        *time.Time `json:"paid_at" db:"paid_at"`
	PaymentMethod string    `json:"payment_method" db:"payment_method"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func calculateFeeHandler(c *gin.Context) {
	var req struct {
		TenantID   string  `json:"tenant_id" binding:"required"`
		Product   string  `json:"product" binding:"required"`
		Amount    float64 `json:"amount" binding:"required"`
		FeeType   string  `json:"fee_type" binding:"required"` // transaction, withdrawal, api_call
	}
	c.ShouldBindJSON(&req)

	feeConfig := getFeeConfig(req.FeeType)
	feeAmount := req.Amount * feeConfig.TransactionFeePercent / 100

	if req.FeeType == "api_call" {
		feeAmount = feeConfig.ApiCallFee
	} else if req.FeeType == "withdrawal" {
		feeAmount = feeConfig.WithdrawalFee
	}

	netAmount := req.Amount - feeAmount

	c.JSON(http.StatusOK, gin.H{
		"amount":      req.Amount,
		"fee_amount":  feeAmount,
		"fee_percent": feeConfig.TransactionFeePercent,
		"net_amount":  netAmount,
	})
}

func collectFeeHandler(c *gin.Context) {
	var req struct {
		TenantID     string  `json:"tenant_id" binding:"required"`
		Product     string  `json:"product" binding:"required"`
		TransactionID string `json:"transaction_id" binding:"required"`
		Amount       float64 `json:"amount" binding:"required"`
		FeeType      string  `json:"fee_type" binding:"required"`
	}
	c.ShouldBindJSON(&req)

	feeConfig := getFeeConfig(req.FeeType)
	feeAmount := req.Amount * feeConfig.TransactionFeePercent / 100

	if req.FeeType == "withdrawal" {
		feeAmount = feeConfig.WithdrawalFee
	}

	fee := map[string]interface{}{
		"id":              uuid.New().String(),
		"tenant_id":       req.TenantID,
		"product":         req.Product,
		"transaction_id":  req.TransactionID,
		"amount":         req.Amount,
		"fee_amount":     feeAmount,
		"net_amount":     req.Amount - feeAmount,
		"status":         "collected",
		"created_at":     time.Now().Unix(),
	}

	c.JSON(http.StatusOK, gin.H{"fee": fee, "message": "Fee collected"})
}

func getFeeHistoryHandler(c *gin.Context) {
	tenantID := c.Param("tenant_id")

	fees := []map[string]interface{}{
		{
			"id":             uuid.New().String(),
			"transaction_id": "tx_123",
			"amount":         1000.0,
			"fee_amount":    10.0,
			"net_amount":    990.0,
			"created_at":    time.Now().Unix(),
		},
	}

	c.JSON(http.StatusOK, gin.H{"fees": fees, "total": 100.0})
}

func trackUsageHandler(c *gin.Context) {
	var req struct {
		TenantID  string `json:"tenant_id" binding:"required"`
		Product  string `json:"product" binding:"required"`
		Endpoint string `json:"endpoint" binding:"required"`
		Method  string `json:"method" binding:"required"`
		LatencyMs int   `json:"latency_ms"`
	}
	c.ShouldBindJSON(&req)

	cost := calculateAPICost(req.Endpoint, req.Method)

	usage := map[string]interface{}{
		"id":         uuid.New().String(),
		"tenant_id":  req.TenantID,
		"product":    req.Product,
		"endpoint":   req.Endpoint,
		"method":     req.Method,
		"latency_ms": req.LatencyMs,
		"cost":       cost,
		"created_at": time.Now().Unix(),
	}

	c.JSON(http.StatusOK, gin.H{"usage": usage, "message": "Usage tracked"})
}

func getUsageHandler(c *gin.Context) {
	tenantID := c.Param("tenant_id")

	usage := []map[string]interface{}{
		{
			"id":         uuid.New().String(),
			"endpoint":   "/api/v1/fetcher/prices",
			"method":     "GET",
			"calls":      1000,
			"cost":       0.001,
			"created_at": time.Now().Unix(),
		},
	}

	c.JSON(http.StatusOK, gin.H{"usage": usage, "total_calls": 1000, "total_cost": 1.0})
}

func getUsageSummaryHandler(c *gin.Context) {
	tenantID := c.Param("tenant_id")

	summary := map[string]interface{}{
		"tenant_id":       tenantID,
		"total_calls":     100000,
		"total_cost":      100.0,
		"calls_remaining": 900000,
		"limit":          1000000,
		"period":         "monthly",
		"reset_date":     time.Now().Add(30 * 24 * time.Hour).Unix(),
	}

	c.JSON(http.StatusOK, summary)
}

func createInvoiceHandler(c *gin.Context) {
	var req struct {
		TenantID   string    `json:"tenant_id" binding:"required"`
		Type       string    `json:"type" binding:"required"`
		PeriodStart time.Time `json:"period_start"`
		PeriodEnd   time.Time `json:"period_end"`
		Items      []map[string]interface{} `json:"items"`
	}
	c.ShouldBindJSON(&req)

	subtotal := 0.0
	for _, item := range req.Items {
		subtotal += item["amount"].(float64)
	}

	tax := subtotal * 0.1
	total := subtotal + tax

	invoice := map[string]interface{}{
		"id":             uuid.New().String(),
		"invoice_number": "INV-" + time.Now().Format("200601021504"),
		"tenant_id":      req.TenantID,
		"type":           req.Type,
		"subtotal":       subtotal,
		"tax":            tax,
		"total":          total,
		"status":         "pending",
		"due_date":       time.Now().Add(30 * 24 * time.Hour).Unix(),
		"created_at":     time.Now().Unix(),
	}

	c.JSON(http.StatusCreated, gin.H{"invoice": invoice})
}

func listInvoicesHandler(c *gin.Context) {
	tenantID := c.Param("tenant_id")

	invoices := []map[string]interface{}{
		{
			"id":            uuid.New().String(),
			"invoice_number": "INV-20240801001",
			"type":         "subscription",
			"total":        999.0,
			"status":       "paid",
			"due_date":     time.Now().Unix(),
		},
	}

	c.JSON(http.StatusOK, gin.H{"invoices": invoices, "total": 1})
}

func getInvoiceHandler(c *gin.Context) {
	invoiceID := c.Param("id")

	invoice := map[string]interface{}{
		"id":              invoiceID,
		"invoice_number":  "INV-20240801001",
		"tenant_id":      uuid.New().String(),
		"type":           "subscription",
		"subtotal":       909.09,
		"tax":            90.91,
		"total":          1000.0,
		"status":         "paid",
		"paid_at":        time.Now().Unix(),
		"created_at":     time.Now().Add(-7 * 24 * time.Hour).Unix(),
	}

	c.JSON(http.StatusOK, invoice)
}

func payInvoiceHandler(c *gin.Context) {
	var req struct {
		PaymentMethod string `json:"payment_method" binding:"required"`
	}
	c.ShouldBindJSON(&req)

	c.JSON(http.StatusOK, gin.H{
		"status":      "paid",
		"paid_at":    time.Now().Unix(),
		"message":    "Invoice paid successfully",
	})
}

func getSubscriptionHandler(c *gin.Context) {
	tenantID := c.Param("tenant_id")

	subscription := map[string]interface{}{
		"tenant_id":            tenantID,
		"plan":                "enterprise",
		"status":              "active",
		"monthly_price":       999.0,
		"api_call_limit":      1000000,
		"current_period_end":  time.Now().Add(30 * 24 * time.Hour).Unix(),
		"features": []string{
			"unlimited_wallets",
			"unlimited_bots",
			"priority_support",
			"custom_branding",
		},
	}

	c.JSON(http.StatusOK, subscription)
}

func upgradeSubscriptionHandler(c *gin.Context) {
	var req struct {
		NewPlan string `json:"new_plan" binding:"required"`
	}
	c.ShouldBindJSON(&req)

	c.JSON(http.StatusOK, gin.H{
		"status":     "upgraded",
		"new_plan":   req.NewPlan,
		"message":    "Subscription upgraded",
	})
}

func downgradeSubscriptionHandler(c *gin.Context) {
	var req struct {
		NewPlan string `json:"new_plan" binding:"required"`
	}
	c.ShouldBindJSON(&req)

	c.JSON(http.StatusOK, gin.H{
		"status":    "downgraded",
		"new_plan":  req.NewPlan,
		"message":   "Subscription downgraded",
	})
}

func getRevenueAnalyticsHandler(c *gin.Context) {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	analytics := map[string]interface{}{
		"period": map[string]string{
			"start": startDate,
			"end":   endDate,
		},
		"total_revenue":    150000.0,
		"subscription":     100000.0,
		"api_usage":        30000.0,
		"transaction_fees": 20000.0,
		"by_tenant": []map[string]interface{}{
			{"tenant_id": "t1", "revenue": 50000.0},
			{"tenant_id": "t2", "revenue": 40000.0},
			{"tenant_id": "t3", "revenue": 30000.0},
		},
		"by_product": map[string]float64{
			"master_wallet":   60000.0,
			"user_wallet":     40000.0,
			"bots":           30000.0,
			"project_party":  20000.0,
		},
	}

	c.JSON(http.StatusOK, analytics)
}

func getUsageAnalyticsHandler(c *gin.Context) {
	analytics := map[string]interface{}{
		"total_api_calls":     10000000,
		"total_cost":         10000.0,
		"avg_latency_ms":     45,
		"by_endpoint": map[string]int{
			"/api/v1/fetcher/prices":        5000000,
			"/api/v1/fetcher/wallet":       3000000,
			"/api/v1/fetcher/blockchain":    2000000,
		},
	}

	c.JSON(http.StatusOK, analytics)
}

func getTenantAnalyticsHandler(c *gin.Context) {
	analytics := map[string]interface{}{
		"total_tenants":       100,
		"active_tenants":      85,
		"new_tenants_30d":    15,
		"churned_tenants_30d": 5,
		"by_plan": map[string]int{
			"free":      30,
			"basic":     40,
			"pro":       20,
			"enterprise": 10,
		},
	}

	c.JSON(http.StatusOK, analytics)
}

func getDashboardHandler(c *gin.Context) {
	dashboard := map[string]interface{}{
		"total_revenue_today":    5000.0,
		"total_revenue_monthly":  150000.0,
		"active_tenants":         85,
		"total_api_calls":        10000000,
		"avg_response_time_ms":    45,
		"top_tenants": []map[string]interface{}{
			{"name": "Company A", "revenue": 50000.0},
			{"name": "Company B", "revenue": 40000.0},
			{"name": "Company C", "revenue": 30000.0},
		},
	}

	c.JSON(http.StatusOK, dashboard)
}

func generateRevenueReportHandler(c *gin.Context) {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	report := map[string]interface{}{
		"period":        map[string]string{"start": startDate, "end": endDate},
		"generated_at": time.Now().Unix(),
		"revenue": map[string]interface{}{
			"subscription":  100000.0,
			"api_usage":    30000.0,
			"transactions": 20000.0,
			"total":       150000.0,
		},
		"invoices": map[string]interface{}{
			"total":        100,
			"paid":        90,
			"pending":      8,
			"overdue":     2,
			"total_value":  150000.0,
		},
		"tenants": map[string]interface{}{
			"total":      100,
			"new_30d":   15,
			"churned_30d": 5,
		},
	}

	c.JSON(http.StatusOK, report)
}

func getFeeConfig(feeType string) FeeConfig {
	configs := map[string]FeeConfig{
		"transaction": {TransactionFeePercent: 1.0, ApiCallFee: 0.0, MonthlyBaseFee: 0.0, WithdrawalFee: 0.0},
		"withdrawal":  {TransactionFeePercent: 0.0, ApiCallFee: 0.0, MonthlyBaseFee: 0.0, WithdrawalFee: 5.0},
		"api_call":    {TransactionFeePercent: 0.0, ApiCallFee: 0.001, MonthlyBaseFee: 0.0, WithdrawalFee: 0.0},
	}

	if config, ok := configs[feeType]; ok {
		return config
	}
	return configs["transaction"]
}

func calculateAPICost(endpoint, method string) float64 {
	baseCost := 0.001

	if endpoint == "/api/v1/fetcher/prices" {
		baseCost = 0.001
	} else if endpoint == "/api/v1/fetcher/wallet" {
		baseCost = 0.002
	} else if endpoint == "/api/v1/fetcher/blockchain" {
		baseCost = 0.005
	}

	if method == "POST" {
		baseCost *= 2
	}

	return baseCost
}

type DB struct{}

func initDatabase(cfg *Config) (*DB, error) {
	log.Printf("Connecting to PostgreSQL at %s:%d", cfg.Database.Host, cfg.Database.Port)
	return &DB{}, nil
}

func (d *DB) Close() {}
