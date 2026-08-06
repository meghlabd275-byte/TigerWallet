// Client Onboarding Service - Go Implementation
// Complete onboarding workflow for White Level clients

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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
}

type OnboardingStatus string

const (
	StatusPending    OnboardingStatus = "pending"
	StatusInProgress OnboardingStatus = "in_progress"
	StatusApproved   OnboardingStatus = "approved"
	StatusRejected   OnboardingStatus = "rejected"
	StatusSuspended  OnboardingStatus = "suspended"
)

type OnboardingRequest struct {
	ID              uuid.UUID        `json:"id"`
	CompanyName     string          `json:"company_name"`
	ContactName     string          `json:"contact_name"`
	ContactEmail    string          `json:"contact_email"`
	ContactPhone    string          `json:"contact_phone"`
	Products        string          `json:"products"` // JSON array
	Website         string          `json:"website"`
	Status          OnboardingStatus `json:"status"`
	AssignedTo      *uuid.UUID      `json:"assigned_to"` // admin who reviews
	Notes           string          `json:"notes"`
	Documents       string          `json:"documents"` // JSON object
	SubmittedAt     time.Time       `json:"submitted_at"`
	ReviewedAt     *time.Time      `json:"reviewed_at"`
}

var db *pgxpool.Pool
var redis *redis.Client
var config Config
var logger *log.Logger

func main() {
	logger = log.New(os.Stdout, "Onboarding: ", log.LstdFlags)
	logger.Println("Starting Client Onboarding Service...")

	config.Port = getEnv("ONBOARDING_PORT", "8101")
	config.DatabaseURL = getEnv("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet_admin")
	config.RedisURL = getEnv("REDIS_URL", "redis://localhost:6379")

	var err error
	db, err = pgxpool.New(context.Background(), config.DatabaseURL)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	logger.Println("Database connected")

	opt, _ := redis.ParseURL(config.RedisURL)
	redis = redis.NewClient(opt)
	redis.Ping(context.Background())
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

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "onboarding"})
	})

	// Onboarding requests
	router.POST("/api/v1/onboarding", submitOnboarding)
	router.GET("/api/v1/onboarding/:id", getOnboarding)
	router.GET("/api/v1/onboarding", listOnboardings)
	router.PUT("/api/v1/onboarding/:id/status", updateStatus)
	router.POST("/api/v1/onboarding/:id/approve", approveOnboarding)
	router.POST("/api/v1/onboarding/:id/reject", rejectOnboarding)

	// Configuration
	router.GET("/api/v1/config", getOnboardingConfig)
	router.PUT("/api/v1/config", updateOnboardingConfig)

	logger.Printf("Starting server on port %s", config.Port)
	srv := &http.Server{Addr: ":" + config.Port, Handler: router}

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
	srv.Shutdown(ctx)
	db.Close()
	redis.Close()
	logger.Println("Server exited")
}

func submitOnboarding(c *gin.Context) {
	var req struct {
		CompanyName string `json:"company_name" binding:"required"`
		ContactName  string `json:"contact_name" binding:"required"`
		ContactEmail string `json:"contact_email" binding:"required"`
		ContactPhone string `json:"contact_phone"`
		Products     string `json:"products"`
		Website      string `json:"website"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	onboarding := OnboardingRequest{
		ID:           uuid.New(),
		CompanyName:  req.CompanyName,
		ContactName:  req.ContactName,
		ContactEmail: req.ContactEmail,
		ContactPhone: req.ContactPhone,
		Products:     req.Products,
		Website:      req.Website,
		Status:       StatusPending,
		SubmittedAt:  time.Now(),
	}

	_, err := db.Exec(context.Background(), `
		INSERT INTO onboarding_requests (id, company_name, contact_name, contact_email, contact_phone, products, website, status, submitted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, onboarding.ID, onboarding.CompanyName, onboarding.ContactName, onboarding.ContactEmail, onboarding.ContactPhone, onboarding.Products, onboarding.Website, onboarding.Status, onboarding.SubmittedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Send notification to admin
	notif := map[string]interface{}{
		"type":    "new_onboarding",
		"request": onboarding,
	}
	notifJSON, _ := json.Marshal(notif)
	redis.Publish(context.Background(), "notifications", notifJSON)

	c.JSON(http.StatusCreated, onboarding)
}

func getOnboarding(c *gin.Context) {
	id := c.Param("id")
	uid, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	var req OnboardingRequest
	err = db.QueryRow(context.Background(), `
		SELECT id, company_name, contact_name, contact_email, contact_phone, products, website, status, assigned_to, notes, documents, submitted_at, reviewed_at
		FROM onboarding_requests WHERE id = $1
	`, uid).Scan(&req.ID, &req.CompanyName, &req.ContactName, &req.ContactEmail, &req.ContactPhone, &req.Products, &req.Website, &req.Status, &req.AssignedTo, &req.Notes, &req.Documents, &req.SubmittedAt, &req.ReviewedAt)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	c.JSON(http.StatusOK, req)
}

func listOnboardings(c *gin.Context) {
	status := c.Query("status")

	query := "SELECT id, company_name, contact_name, contact_email, contact_phone, products, website, status, submitted_at FROM onboarding_requests"
	if status != "" {
		query += fmt.Sprintf(" WHERE status = '%s'", status)
	}
	query += " ORDER BY submitted_at DESC LIMIT 100"

	rows, err := db.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var requests []OnboardingRequest
	for rows.Next() {
		var req OnboardingRequest
		if err := rows.Scan(&req.ID, &req.CompanyName, &req.ContactName, &req.ContactEmail, &req.ContactPhone, &req.Products, &req.Website, &req.Status, &req.SubmittedAt); err != nil {
			continue
		}
		requests = append(requests, req)
	}

	c.JSON(http.StatusOK, gin.H{"requests": requests})
}

func updateStatus(c *gin.Context) {
	id := c.Param("id")
	uid, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	var req struct {
		Status    OnboardingStatus `json:"status" binding:"required"`
		Notes     string           `json:"notes"`
		AssignedTo *uuid.UUID      `json:"assigned_to"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err = db.Exec(context.Background(), `
		UPDATE onboarding_requests SET status = $1, notes = $2, assigned_to = $3, reviewed_at = NOW() WHERE id = $4
	`, req.Status, req.Notes, req.AssignedTo, uid)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "status updated"})
}

func approveOnboarding(c *gin.Context) {
	id := c.Param("id")
	uid, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	_, err = db.Exec(context.Background(), `
		UPDATE onboarding_requests SET status = 'approved', reviewed_at = NOW() WHERE id = $1
	`, uid)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "onboarding approved"})
}

func rejectOnboarding(c *gin.Context) {
	id := c.Param("id")
	uid, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)

	_, err = db.Exec(context.Background(), `
		UPDATE onboarding_requests SET status = 'rejected', notes = $1, reviewed_at = NOW() WHERE id = $2
	`, req.Reason, uid)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "onboarding rejected"})
}

func getOnboardingConfig(c *gin.Context) {
	configJSON, err := redis.Get(context.Background(), "onboarding_config").Result()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"products": []string{"master_wallet", "user_wallet", "bots", "project_party"},
			"approval_required": true,
			"auto_approve": false,
		})
		return
	}

	c.JSON(http.StatusOK, json.RawMessage(configJSON))
}

func updateOnboardingConfig(c *gin.Context) {
	var config map[string]interface{}
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	configJSON, _ := json.Marshal(config)
	redis.Set(context.Background(), "onboarding_config", configJSON, 0)

	c.JSON(http.StatusOK, gin.H{"message": "config updated"})
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
