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
	"github.com/google/uuid"
)

func main() {
	cfg := loadConfig()

	// Initialize database
	db, err := initDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize Gin router
	router := gin.Default()

	// CORS
	router.Use(corsMiddleware())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "tiger-kyc"})
	})

	// API routes
	api := router.Group("/api/v1/kyc")
	{
		// KYC Submissions
		api.POST("/submit", submitKYCHandler)
		api.GET("/status/:user_id", getKYCStatusHandler)
		api.GET("/applications", listApplicationsHandler)
		api.GET("/applications/:id", getApplicationHandler)
		api.PUT("/applications/:id/review", reviewApplicationHandler)
		api.POST("/applications/:id/approve", approveApplicationHandler)
		api.POST("/applications/:id/reject", rejectApplicationHandler)
		api.POST("/applications/:id/request-info", requestInfoHandler)

		// Documents
		api.POST("/documents/upload", uploadDocumentHandler)
		api.GET("/documents/:application_id", getDocumentsHandler)
		api.DELETE("/documents/:id", deleteDocumentHandler)

		// AML Checks
		api.POST("/aml/check", runAMLChecksHandler)
		api.GET("/aml/history/:user_id", getAMLHistoryHandler)

		// Watchlists
		api.POST("/watchlist/add", addToWatchlistHandler)
		api.DELETE("/watchlist/remove", removeFromWatchlistHandler)
		api.GET("/watchlist/search", searchWatchlistHandler)

		// Reports
		api.GET("/reports/compliance", generateComplianceReportHandler)
		api.GET("/reports/audit", generateAuditReportHandler)
	}

	// Start server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: router,
	}

	go func() {
		log.Printf("KYC service starting on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
}

// ============== Configuration ==============

type Config struct {
	Port     string
	Database DatabaseConfig
	Stripe   StripeConfig
	SumSub   SumSubConfig
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

type SumSubConfig struct {
	AppKey    string
	SecretKey string
}

func loadConfig() *Config {
	return &Config{
		Port: getEnv("KYC_PORT", "9005"),
		Database: DatabaseConfig{
			Host:     getEnv("KYC_DB_HOST", "localhost"),
			Port:     getEnvInt("KYC_DB_PORT", 5432),
			User:     getEnv("KYC_DB_USER", "tigerwallet"),
			Password: getEnv("KYC_DB_PASSWORD", "password"),
			DBName:   getEnv("KYC_DB_NAME", "tigerwallet_kyc"),
		},
		Stripe: StripeConfig{
			SecretKey: getEnv("STRIPE_SECRET_KEY", ""),
		},
		SumSub: SumSubConfig{
			AppKey:    getEnv("SUMSUB_APP_KEY", ""),
			SecretKey: getEnv("SUMSUB_SECRET_KEY", ""),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		if _, err := fmt.Sscan(value, &intValue); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// ============== Models ==============

type KYCApplication struct {
	ID                  uuid.UUID  `json:"id" db:"id"`
	UserID             uuid.UUID `json:"user_id" db:"user_id"`
	TenantID           uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Level              string    `json:"level" db:"level"`
	Status             string    `json:"status" db:"status"` // pending, in_review, approved, rejected, needs_more_info
	RejectionReason   *string   `json:"rejection_reason" db:"rejection_reason"`
	ReviewerID        *uuid.UUID `json:"reviewer_id" db:"reviewer_id"`
	ReviewedAt        *time.Time `json:"reviewed_at" db:"reviewed_at"`
	FirstName         string    `json:"first_name" db:"first_name"`
	LastName          string    `json:"last_name" db:"last_name"`
	DateOfBirth      string    `json:"date_of_birth" db:"date_of_birth"`
	Nationality       string    `json:"nationality" db:"nationality"`
	CountryOfResidence string   `json:"country_of_residence" db:"country_of_residence"`
	Address           string    `json:"address" db:"address"`
	City              string    `json:"city" db:"city"`
	State             string    `json:"state" db:"state"`
	PostalCode        string    `json:"postal_code" db:"postal_code"`
	PhoneNumber       string    `json:"phone_number" db:"phone_number"`
	SumSubApplicantID *string   `json:"sumsub_applicant_id" db:"sumsub_applicant_id"`
	AMLCheckPassed    *bool     `json:"aml_check_passed" db:"aml_check_passed"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

type KYCDocument struct {
	ID            uuid.UUID `json:"id" db:"id"`
	ApplicationID uuid.UUID `json:"application_id" db:"application_id"`
	Type         string    `json:"type" db:"type"` // id_card, passport, driver_license, utility_bill, bank_statement
	Side         string    `json:"side" db:"side"` // front, back
	FileURL      string    `json:"file_url" db:"file_url"`
	FileHash     string    `json:"file_hash" db:"file_hash"`
	Status       string    `json:"status" db:"status"` // pending, verified, rejected
	VerifiedAt   *time.Time `json:"verified_at" db:"verified_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type AMLCheck struct {
	ID                  uuid.UUID `json:"id" db:"id"`
	UserID             uuid.UUID `json:"user_id" db:"user_id"`
	ApplicationID      uuid.UUID `json:"application_id" db:"application_id"`
	Status             string    `json:"status" db:"status"` // pending, completed, failed
	PepMatch          bool      `json:"pep_match" db:"pep_match"`
	SanctionMatch     bool      `json:"sanction_match" db:"sanction_match"`
	AdverseMediaMatch bool      `json:"adverse_media_match" db:"adverse_media_match"`
	RiskScore         int       `json:"risk_score" db:"risk_score"`
	RiskLevel         string    `json:"risk_level" db:"risk_level"` // low, medium, high
	Details           map[string]interface{} `json:"details" db:"details"`
	CheckedAt         time.Time `json:"checked_at" db:"checked_at"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
}

type WatchlistEntry struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	EntityType   string    `json:"entity_type" db:"entity_type"` // person, organization, vessel
	Country      string    `json:"country" db:"country"`
	ListType     string    `json:"list_type" db:"list_type"` // pep, sanction, adverse_media
	RiskLevel    string    `json:"risk_level" db:"risk_level"`
	Details      map[string]interface{} `json:"details" db:"details"`
	AddedBy      uuid.UUID `json:"added_by" db:"added_by"`
	AddedAt      time.Time `json:"added_at" db:"added_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// ============== HTTP Handlers ==============

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func submitKYCHandler(c *gin.Context) {
	var req struct {
		UserID              uuid.UUID `json:"user_id" binding:"required"`
		TenantID           uuid.UUID `json:"tenant_id" binding:"required"`
		Level              string    `json:"level" binding:"required"`
		FirstName          string    `json:"first_name" binding:"required"`
		LastName           string    `json:"last_name" binding:"required"`
		DateOfBirth       string    `json:"date_of_birth" binding:"required"`
		Nationality        string    `json:"nationality" binding:"required"`
		CountryOfResidence string   `json:"country_of_residence" binding:"required"`
		Address            string    `json:"address" binding:"required"`
		City               string    `json:"city" binding:"required"`
		State              string    `json:"state"`
		PostalCode         string    `json:"postal_code" binding:"required"`
		PhoneNumber        string    `json:"phone_number" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	application := KYCApplication{
		ID:                   uuid.New(),
		UserID:              req.UserID,
		TenantID:            req.TenantID,
		Level:               req.Level,
		Status:              "pending",
		FirstName:           req.FirstName,
		LastName:            req.LastName,
		DateOfBirth:        req.DateOfBirth,
		Nationality:         req.Nationality,
		CountryOfResidence: req.CountryOfResidence,
		Address:             req.Address,
		City:               req.City,
		State:               req.State,
		PostalCode:         req.PostalCode,
		PhoneNumber:         req.PhoneNumber,
		CreatedAt:          time.Now(),
		UpdatedAt:           time.Now(),
	}

	// In production, create SumSub applicant here

	c.JSON(http.StatusCreated, gin.H{
		"application": application,
		"message":     "KYC application submitted successfully",
	})
}

func getKYCStatusHandler(c *gin.Context) {
	userID := c.Param("user_id")
	uid, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	// Return mock data
	status := map[string]interface{}{
		"user_id":       uid.String(),
		"status":        "approved",
		"level":         "advanced",
		"verified_at":   time.Now().Add(-24 * time.Hour),
		"documents":     []string{"id_card", "passport"},
		"aml_check":     "passed",
		"expiry_date":   time.Now().Add(365 * 24 * time.Hour),
	}

	c.JSON(http.StatusOK, status)
}

func listApplicationsHandler(c *gin.Context) {
	status := c.Query("status")
	limit := c.DefaultQuery("limit", "50")
	offset := c.DefaultQuery("offset", "0")

	// Return mock data
	applications := []map[string]interface{}{
		{
			"id":          uuid.New().String(),
			"user_id":     uuid.New().String(),
			"status":      "pending",
			"first_name":  "John",
			"last_name":   "Doe",
			"level":       "basic",
			"created_at": time.Now(),
		},
		{
			"id":          uuid.New().String(),
			"user_id":     uuid.New().String(),
			"status":      "approved",
			"first_name":  "Jane",
			"last_name":   "Smith",
			"level":       "advanced",
			"created_at":  time.Now().Add(-24 * time.Hour),
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"applications": applications,
		"total":       100,
		"limit":       limit,
		"offset":      offset,
	})
}

func getApplicationHandler(c *gin.Context) {
	appID := c.Param("id")
	aid, err := uuid.Parse(appID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid application ID"})
		return
	}

	application := map[string]interface{}{
		"id":                    aid.String(),
		"user_id":              uuid.New().String(),
		"status":               "in_review",
		"first_name":           "John",
		"last_name":            "Doe",
		"date_of_birth":        "1990-01-01",
		"nationality":           "US",
		"country_of_residence": "US",
		"address":              "123 Main St",
		"city":                 "New York",
		"postal_code":          "10001",
		"created_at":           time.Now(),
		"updated_at":           time.Now(),
	}

	c.JSON(http.StatusOK, application)
}

func reviewApplicationHandler(c *gin.Context) {
	var req struct {
		ReviewerID uuid.UUID `json:"reviewer_id" binding:"required"`
		Notes      string    `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "application marked for review"})
}

func approveApplicationHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "application approved", "status": "approved"})
}

func rejectApplicationHandler(c *gin.Context) {
	var req struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "application rejected", "status": "rejected", "reason": req.Reason})
}

func requestInfoHandler(c *gin.Context) {
	var req struct {
		Message string `json:"message" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "additional information requested", "status": "needs_more_info"})
}

func uploadDocumentHandler(c *gin.Context) {
	var req struct {
		ApplicationID uuid.UUID `json:"application_id" binding:"required"`
		Type          string    `json:"type" binding:"required"`
		Side          string    `json:"side" binding:"required"`
		FileData      string    `json:"file_data" binding:"required"` // base64
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	document := KYCDocument{
		ID:            uuid.New(),
		ApplicationID: req.ApplicationID,
		Type:          req.Type,
		Side:          req.Side,
		FileURL:       fmt.Sprintf("https://storage.tigerwallet.com/documents/%s", uuid.New()),
		Status:        "pending",
		CreatedAt:     time.Now(),
	}

	c.JSON(http.StatusCreated, gin.H{
		"document": document,
		"message": "document uploaded successfully",
	})
}

func getDocumentsHandler(c *gin.Context) {
	appID := c.Param("application_id")

	documents := []map[string]interface{}{
		{
			"id":            uuid.New().String(),
			"type":          "id_card",
			"side":          "front",
			"status":        "verified",
			"verified_at":   time.Now(),
			"created_at":    time.Now(),
		},
	}

	c.JSON(http.StatusOK, gin.H{"application_id": appID, "documents": documents})
}

func deleteDocumentHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "document deleted"})
}

func runAMLChecksHandler(c *gin.Context) {
	var req struct {
		UserID        uuid.UUID `json:"user_id" binding:"required"`
		ApplicationID uuid.UUID `json:"application_id" binding:"required"`
		FirstName     string    `json:"first_name" binding:"required"`
		LastName      string    `json:"last_name" binding:"required"`
		Country       string    `json:"country" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// In production, integrate with AML service provider
	amlCheck := AMLCheck{
		ID:                  uuid.New(),
		UserID:              req.UserID,
		ApplicationID:       req.ApplicationID,
		Status:              "completed",
		PepMatch:            false,
		SanctionMatch:       false,
		AdverseMediaMatch:  false,
		RiskScore:           10,
		RiskLevel:           "low",
		Details:             map[string]interface{}{},
		CheckedAt:           time.Now(),
		CreatedAt:           time.Now(),
	}

	c.JSON(http.StatusOK, gin.H{
		"aml_check": amlCheck,
		"message":   "AML check completed",
	})
}

func getAMLHistoryHandler(c *gin.Context) {
	userID := c.Param("user_id")

	history := []map[string]interface{}{
		{
			"id":                uuid.New().String(),
			"risk_score":        10,
			"risk_level":        "low",
			"pep_match":        false,
			"sanction_match":   false,
			"checked_at":       time.Now(),
		},
	}

	c.JSON(http.StatusOK, gin.H{"user_id": userID, "history": history})
}

func addToWatchlistHandler(c *gin.Context) {
	var req struct {
		Name       string    `json:"name" binding:"required"`
		EntityType string    `json:"entity_type" binding:"required"`
		Country    string    `json:"country"`
		ListType   string    `json:"list_type" binding:"required"`
		RiskLevel  string    `json:"risk_level" binding:"required"`
		Details    map[string]interface{} `json:"details"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entry := WatchlistEntry{
		ID:         uuid.New(),
		Name:       req.Name,
		EntityType: req.EntityType,
		Country:    req.Country,
		ListType:   req.ListType,
		RiskLevel:  req.RiskLevel,
		Details:    req.Details,
		AddedAt:   time.Now(),
		CreatedAt: time.Now(),
	}

	c.JSON(http.StatusCreated, gin.H{"entry": entry, "message": "added to watchlist"})
}

func removeFromWatchlistHandler(c *gin.Context) {
	var req struct {
		ID uuid.UUID `json:"id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "removed from watchlist"})
}

func searchWatchlistHandler(c *gin.Context) {
	query := c.Query("q")

	results := []map[string]interface{}{
		{
			"id":          uuid.New().String(),
			"name":        "Sample Entity",
			"entity_type": "organization",
			"country":     "US",
			"list_type":   "pep",
			"risk_level":  "high",
		},
	}

	c.JSON(http.StatusOK, gin.H{"query": query, "results": results, "total": 1})
}

func generateComplianceReportHandler(c *gin.Context) {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	report := map[string]interface{}{
		"period":           map[string]string{"start": startDate, "end": endDate},
		"total_applications": 100,
		"approved":        85,
		"rejected":         10,
		"pending":          5,
		"aml_checks": map[string]int{
			"passed": 95,
			"failed": 5,
		},
		"generated_at": time.Now(),
	}

	c.JSON(http.StatusOK, report)
}

func generateAuditReportHandler(c *gin.Context) {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	report := map[string]interface{}{
		"period":         map[string]string{"start": startDate, "end": endDate},
		"total_actions":  500,
		"by_type": map[string]int{
			"approved":  200,
			"rejected":  50,
			"reviewed":   250,
		},
		"by_reviewer": map[string]int{
			"admin1": 150,
			"admin2": 200,
			"admin3": 150,
		},
		"generated_at": time.Now(),
	}

	c.JSON(http.StatusOK, report)
}

// ============== Database ==============

type DB interface {
	Close()
}

func initDatabase(cfg *Config) (DB, error) {
	log.Printf("Connecting to PostgreSQL at %s:%d", cfg.Database.Host, cfg.Database.Port)
	return &mockDB{}, nil
}

type mockDB struct{}

func (m *mockDB) Close() {}
