package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var (
store    *Store
screener *amlScreener
)

func main() {
	cfg := loadConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var err error
	store, err = NewStore(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer store.Close()

	if err := configureDocumentStorage(); err != nil {
		log.Fatalf("Failed to configure document storage: %v", err)
	}

	screener = newAMLScreener(store, &AMLProviderConfig{
		Endpoint: cfg.AMLProvider.Endpoint,
		APIKey:   cfg.AMLProvider.APIKey,
	})
	if cfg.AMLProvider.Endpoint == "" {
		log.Print("AML_PROVIDER_URL is not set; screening runs against the local watchlist only")
	}

	// Initialize Gin router
	router := gin.Default()

	// CORS
	router.Use(corsMiddleware())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "tiger-kyc"})
	})
	router.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		if pingErr := store.Ping(ctx); pingErr != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "database": pingErr.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready", "database": "ok"})
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
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
}

// ============== Configuration ==============

type Config struct {
	Port        string
	DatabaseURL string
	Stripe      StripeConfig
	SumSub      SumSubConfig
	AMLProvider AMLProviderConfig
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
		Port:        getEnv("KYC_PORT", "9005"),
		DatabaseURL: mustEnv("DATABASE_URL"),
		Stripe: StripeConfig{
			SecretKey: getEnv("STRIPE_SECRET_KEY", ""),
		},
		SumSub: SumSubConfig{
			AppKey:    getEnv("SUMSUB_APP_KEY", ""),
			SecretKey: getEnv("SUMSUB_SECRET_KEY", ""),
		},
		AMLProvider: AMLProviderConfig{
			Endpoint: getEnv("AML_PROVIDER_URL", ""),
			APIKey:   getEnv("AML_PROVIDER_API_KEY", ""),
		},
	}
}

// mustEnv returns the named environment variable or terminates the process.
// A secret has no safe default: a committed fallback would ship a known
// credential to every deployment.
func mustEnv(key string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	log.Fatalf("%s must be set", key)
	return ""
}

// getEnv returns the named environment variable or the supplied default.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============== Models ==============

type KYCApplication struct {
	ID                 uuid.UUID  `json:"id" db:"id"`
	UserID             uuid.UUID  `json:"user_id" db:"user_id"`
	TenantID           *uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Level              string     `json:"level" db:"level"`
	Status             string     `json:"status" db:"status"` // pending, in_review, approved, rejected, needs_more_info
	RejectionReason    *string    `json:"rejection_reason" db:"rejection_reason"`
	ReviewerID         *uuid.UUID `json:"reviewer_id" db:"reviewer_id"`
	ReviewedAt         *time.Time `json:"reviewed_at" db:"reviewed_at"`
	FirstName          string     `json:"first_name" db:"first_name"`
	LastName           string     `json:"last_name" db:"last_name"`
	DateOfBirth        string     `json:"date_of_birth" db:"date_of_birth"`
	Nationality        string     `json:"nationality" db:"nationality"`
	CountryOfResidence string     `json:"country_of_residence" db:"country_of_residence"`
	Address            string     `json:"address" db:"address"`
	City               string     `json:"city" db:"city"`
	State              string     `json:"state" db:"state"`
	PostalCode         string     `json:"postal_code" db:"postal_code"`
	PhoneNumber        string     `json:"phone_number" db:"phone_number"`
	SumSubApplicantID  *string    `json:"sumsub_applicant_id" db:"sumsub_applicant_id"`
	AMLCheckPassed     *bool      `json:"aml_check_passed" db:"aml_check_passed"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at" db:"updated_at"`
}

type KYCDocument struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	ApplicationID uuid.UUID  `json:"application_id" db:"application_id"`
	Type          string     `json:"type" db:"type"` // id_card, passport, driver_license, utility_bill, bank_statement
	Side          string     `json:"side" db:"side"` // front, back
	FileURL       string     `json:"file_url" db:"file_url"`
	FileHash      string     `json:"file_hash" db:"file_hash"`
	Status        string     `json:"status" db:"status"` // pending, verified, rejected
	VerifiedAt    *time.Time `json:"verified_at" db:"verified_at"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
}

type AMLCheck struct {
	ID                uuid.UUID              `json:"id" db:"id"`
	UserID            uuid.UUID              `json:"user_id" db:"user_id"`
	ApplicationID     uuid.UUID              `json:"application_id" db:"application_id"`
	Status            string                 `json:"status" db:"status"` // pending, completed, failed
	PepMatch          bool                   `json:"pep_match" db:"pep_match"`
	SanctionMatch     bool                   `json:"sanction_match" db:"sanction_match"`
	AdverseMediaMatch bool                   `json:"adverse_media_match" db:"adverse_media_match"`
	RiskScore         int                    `json:"risk_score" db:"risk_score"`
	RiskLevel         string                 `json:"risk_level" db:"risk_level"` // low, medium, high
	Details           map[string]interface{} `json:"details" db:"details"`
	CheckedAt         time.Time              `json:"checked_at" db:"checked_at"`
	CreatedAt         time.Time              `json:"created_at" db:"created_at"`
}

type WatchlistEntry struct {
	ID         uuid.UUID              `json:"id" db:"id"`
	Name       string                 `json:"name" db:"name"`
	EntityType string                 `json:"entity_type" db:"entity_type"` // person, organization, vessel
	Country    string                 `json:"country" db:"country"`
	ListType   string                 `json:"list_type" db:"list_type"` // pep, sanction, adverse_media
	RiskLevel  string                 `json:"risk_level" db:"risk_level"`
	Details    map[string]interface{} `json:"details" db:"details"`
	AddedBy    uuid.UUID              `json:"added_by" db:"added_by"`
	AddedAt    time.Time              `json:"added_at" db:"added_at"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
}

// ============== HTTP Handlers ==============

// allowedOrigin reports whether origin appears in
// TIGERWALLET_ALLOWED_ORIGINS (comma separated). Access is granted only to an
// origin that is explicitly listed: an empty allow-list must not match every
// caller, and "*" is refused because this service sends
// Access-Control-Allow-Credentials, which a browser will not honour alongside
// a wildcard origin.
func allowedOrigin(origin string) bool {
	if origin == "" {
		return false
	}
	for _, item := range strings.Split(os.Getenv("TIGERWALLET_ALLOWED_ORIGINS"), ",") {
		if candidate := strings.TrimSpace(item); candidate != "" && candidate != "*" && candidate == origin {
			return true
		}
	}
	return false
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if allowedOrigin(c.Request.Header.Get("Origin")) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", c.Request.Header.Get("Origin"))
			c.Writer.Header().Add("Vary", "Origin")
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
