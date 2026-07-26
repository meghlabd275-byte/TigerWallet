package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Port     string
	RedisURL string
}

func LoadConfig() *Config {
	return &Config{
		Port:     getEnv("PORT", "8450"),
		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============================================================================
// Models
// ============================================================================

type Guardian struct {
	ID          string    `json:"id"`
	Address     string    `json:"address"`
	Email       string    `json:"email"`
	Name        string    `json:"name"`
	IsFamily    bool      `json:"isFamily"`
	Weight      int       `json:"weight"` // Voting weight
	Confirmed   bool      `json:"confirmed"`
	ConfirmedAt *int64    `json:"confirmedAt"`
}

type RecoveryRequest struct {
	ID             string    `json:"id"`
	UserID         string    `json:"userId"`
	NewPublicKey   string    `json:"newPublicKey"`
	Threshold      int       `json:"threshold"`
	Guardians      []Guardian `json:"guardians"`
	ConfirmedCount int       `json:"confirmedCount"`
	Status         string    `json:"status"` // pending, confirmed, executed, cancelled
	CreatedAt      int64     `json:"createdAt"`
	ExecutedAt     *int64    `json:"executedAt"`
}

type GuardianConfirmation struct {
	RequestID   string `json:"requestId"`
	GuardianID  string `json:"guardianId"`
	Signature   string `json:"signature"`
	ConfirmedAt int64  `json:"confirmedAt"`
}

type SocialRecoveryService struct {
	config     *Config
	redis     *redis.Client
	recoveryRequests map[string]*RecoveryRequest
	guardians map[string][]Guardian
	mu        sync.RWMutex
}

func NewSocialRecoveryService(config *Config) *SocialRecoveryService {
	redisClient := redis.NewClient(&redis.Options{
		Addr: config.RedisURL,
	})

	return &SocialRecoveryService{
		config:           config,
		redis:           redisClient,
		recoveryRequests: make(map[string]*RecoveryRequest),
		guardians:       make(map[string][]Guardian),
	}
}

// ============================================================================
// Social Recovery Operations
// ============================================================================

func (s *SocialRecoveryService) SetupGuardians(userID string, guardianList []Guardian, threshold int) error {
	// Validate threshold
	if threshold < 1 || threshold > len(guardianList) {
		return fmt.Errorf("threshold must be between 1 and %d", len(guardianList))
	}

	// Validate guardian addresses
	for _, g := range guardianList {
		if g.Address == "" {
			return fmt.Errorf("guardian address is required")
		}
		g.Weight = 1 // Default weight
	}

	s.mu.Lock()
	s.guardians[userID] = guardianList
	s.mu.Unlock()

	return nil
}

func (s *SocialRecoveryService) InitiateRecovery(userID, newPublicKey string) (*RecoveryRequest, error) {
	s.mu.RLock()
	guardianList := s.guardians[userID]
	s.mu.RUnlock()

	if len(guardianList) == 0 {
		return nil, fmt.Errorf("no guardians configured for user")
	}

	// Calculate threshold (need > 50% of guardians)
	threshold := len(guardianList)/2 + 1

	recovery := &RecoveryRequest{
		ID:             generateID(),
		UserID:         userID,
		NewPublicKey:   newPublicKey,
		Threshold:      threshold,
		Guardians:      guardianList,
		ConfirmedCount: 0,
		Status:         "pending",
		CreatedAt:      time.Now().Unix(),
	}

	s.mu.Lock()
	s.recoveryRequests[recovery.ID] = recovery
	s.mu.Unlock()

	return recovery, nil
}

func (s *SocialRecoveryService) ConfirmRecovery(requestID, guardianID, signature string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	recovery, ok := s.recoveryRequests[requestID]
	if !ok {
		return fmt.Errorf("recovery request not found")
	}

	if recovery.Status != "pending" {
		return fmt.Errorf("recovery request is not pending")
	}

	// Verify guardian is part of this recovery
	isGuardian := false
	for _, g := range recovery.Guardians {
		if g.ID == guardianID {
			isGuardian = true
			break
		}
	}

	if !isGuardian {
		return fmt.Errorf("guardian not authorized for this recovery")
	}

	// Verify signature (in production, verify using guardian's address)
	if signature == "" {
		return fmt.Errorf("signature required")
	}

	// Check if already confirmed
	for _, g := range recovery.Guardians {
		if g.ID == guardianID && g.Confirmed {
			return fmt.Errorf("guardian already confirmed")
		}
	}

	// Mark guardian as confirmed
	for i := range recovery.Guardians {
		if recovery.Guardians[i].ID == guardianID {
			recovery.Guardians[i].Confirmed = true
			now := time.Now().Unix()
			recovery.Guardians[i].ConfirmedAt = &now
			break
		}
	}

	recovery.ConfirmedCount++

	// Check if threshold reached
	if recovery.ConfirmedCount >= recovery.Threshold {
		recovery.Status = "confirmed"
	}

	return nil
}

func (s *SocialRecoveryService) ExecuteRecovery(requestID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	recovery, ok := s.recoveryRequests[requestID]
	if !ok {
		return fmt.Errorf("recovery request not found")
	}

	if recovery.Status != "confirmed" {
		return fmt.Errorf("recovery not confirmed by required guardians")
	}

	// Execute recovery (in production, this would trigger key rotation)
	now := time.Now().Unix()
	recovery.ExecutedAt = &now
	recovery.Status = "executed"

	return nil
}

func (s *SocialRecoveryService) CancelRecovery(requestID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	recovery, ok := s.recoveryRequests[requestID]
	if !ok {
		return fmt.Errorf("recovery request not found")
	}

	if recovery.UserID != userID {
		return fmt.Errorf("unauthorized")
	}

	if recovery.Status != "pending" {
		return fmt.Errorf("cannot cancel non-pending recovery")
	}

	recovery.Status = "cancelled"

	return nil
}

func (s *SocialRecoveryService) GetRecoveryStatus(requestID string) *RecoveryRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.recoveryRequests[requestID]
}

func (s *SocialRecoveryService) GetUserGuardians(userID string) []Guardian {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.guardians[userID]
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func (s *SocialRecoveryService) RegisterRoutes(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "social-recovery"})
	})

	api := r.Group("/api/v1/recovery")
	{
		// Setup guardians
		api.POST("/guardians", s.handleSetupGuardians)
		
		// Get user guardians
		api.GET("/guardians/:userId", s.handleGetGuardians)
		
		// Initiate recovery
		api.POST("/initiate", s.handleInitiateRecovery)
		
		// Confirm recovery (by guardian)
		api.POST("/confirm", s.handleConfirmRecovery)
		
		// Execute recovery
		api.POST("/execute/:requestId", s.handleExecuteRecovery)
		
		// Cancel recovery
		api.POST("/cancel", s.handleCancelRecovery)
		
		// Get status
		api.GET("/status/:requestId", s.handleGetStatus)
	}
}

func (s *SocialRecoveryService) handleSetupGuardians(c *gin.Context) {
	var req struct {
		UserID    string     `json:"userId" binding:"required"`
		Guardians []Guardian `json:"guardians" binding:"required"`
		Threshold int        `json:"threshold"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	threshold := req.Threshold
	if threshold == 0 {
		threshold = len(req.Guardians)/2 + 1
	}

	if err := s.SetupGuardians(req.UserID, req.Guardians, threshold); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Guardians configured successfully"})
}

func (s *SocialRecoveryService) handleGetGuardians(c *gin.Context) {
	userID := c.Param("userId")
	guardians := s.GetUserGuardians(userID)

	c.JSON(http.StatusOK, gin.H{"guardians": guardians})
}

func (s *SocialRecoveryService) handleInitiateRecovery(c *gin.Context) {
	var req struct {
		UserID       string `json:"userId" binding:"required"`
		NewPublicKey string `json:"newPublicKey" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	recovery, err := s.InitiateRecovery(req.UserID, req.NewPublicKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, recovery)
}

func (s *SocialRecoveryService) handleConfirmRecovery(c *gin.Context) {
	var req GuardianConfirmation
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.ConfirmRecovery(req.RequestID, req.GuardianID, req.Signature); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	recovery := s.GetRecoveryStatus(req.RequestID)

	c.JSON(http.StatusOK, gin.H{
		"message":        "Recovery confirmed",
		"confirmedCount": recovery.ConfirmedCount,
		"threshold":      recovery.Threshold,
		"status":         recovery.Status,
	})
}

func (s *SocialRecoveryService) handleExecuteRecovery(c *gin.Context) {
	requestID := c.Param("requestId")

	if err := s.ExecuteRecovery(requestID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Recovery executed successfully"})
}

func (s *SocialRecoveryService) handleCancelRecovery(c *gin.Context) {
	var req struct {
		RequestID string `json:"requestId" binding:"required"`
		UserID    string `json:"userId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.CancelRecovery(req.RequestID, req.UserID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Recovery cancelled"})
}

func (s *SocialRecoveryService) handleGetStatus(c *gin.Context) {
	requestID := c.Param("requestId")
	recovery := s.GetRecoveryStatus(requestID)

	if recovery == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Recovery request not found"})
		return
	}

	c.JSON(http.StatusOK, recovery)
}

// ============================================================================
// Helper Functions
// ============================================================================

func generateID() string {
	hash := sha256.New()
	hash.Write([]byte(time.Now().Format(time.RFC3339Nano)))
	return hex.EncodeToString(hash.Sum(nil))[:16]
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := LoadConfig()
	service := NewSocialRecoveryService(config)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	service.RegisterRoutes(r)

	srv := &http.Server{
		Addr:    ":" + config.Port,
		Handler: r,
	}

	go func() {
		log.Printf("Social Recovery service starting on port %s", config.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}
