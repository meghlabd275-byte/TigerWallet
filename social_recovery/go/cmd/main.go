package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ============ Configuration ============

type Config struct {
	Port             string
	RedisURL         string
	AdminPrivateKey  string
	RecoveryDelay    time.Duration
	MaxGuardians    int
	MinGuardians    int
	EmailEnabled     bool
	SMSEnabled      bool
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============ Models ============

type Wallet struct {
	ID            string   `json:"id"`
	Address       string   `json:"address"`
	Chain         string   `json:"chain"`
	Guardians     []Guardian `json:"guardians"`
	Threshold     int      `json:"threshold"`
	Status        string   `json:"status"`
	RecoveryCount int      `json:"recoveryCount"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type Guardian struct {
	ID          string     `json:"id"`
	Address     string     `json:"address"`      // Ethereum address
	Email       string     `json:"email,omitempty"`
	Phone       string     `json:"phone,omitempty"`
	Type        GuardianType `json:"type"`
	Confirmed   bool       `json:"confirmed"`
	InvitedAt   time.Time  `json:"invitedAt"`
	ConfirmedAt *time.Time `json:"confirmedAt,omitempty"`
}

type GuardianType string

const (
	GuardianTypeAddress GuardianType = "address"
	GuardianTypeEmail  GuardianType = "email"
	GuardianTypePhone  GuardianType = "phone"
	GuardianTypeSocial GuardianType = "social"
)

type RecoveryRequest struct {
	ID              string           `json:"id"`
	WalletID        string           `json:"walletId"`
	NewOwnerAddress string           `json:"newOwnerAddress"`
	Status          RecoveryStatus  `json:"status"`
	ThresholdMet    bool            `json:"thresholdMet"`
	Signatures      []GuardianSignature `json:"signatures"`
	DelayEnd        time.Time       `json:"delayEnd"`
	ExecutedAt      *time.Time      `json:"executedAt,omitempty"`
	CancelledAt     *time.Time      `json:"cancelledAt,omitempty"`
	CreatedAt       time.Time       `json:"createdAt"`
	ExpiresAt       time.Time       `json:"expiresAt"`
}

type RecoveryStatus string

const (
	RecoveryPending   RecoveryStatus = "pending"
	RecoveryApproved  RecoveryStatus = "approved"
	RecoveryExecuting RecoveryStatus = "executing"
	RecoveryCompleted RecoveryStatus = "completed"
	RecoveryFailed    RecoveryStatus = "failed"
	RecoveryCancelled RecoveryStatus = "cancelled"
)

type GuardianSignature struct {
	GuardianID  string    `json:"guardianId"`
	Signature   string    `json:"signature"`
	SignedAt    time.Time `json:"signedAt"`
	Message     string    `json:"message"`
}

type RecoverySession struct {
	WalletID     string          `json:"walletId"`
	Challenge     string          `json:"challenge"`
	ExpiresAt    time.Time       `json:"expiresAt"`
	MaxAttempts  int             `json:"maxAttempts"`
	Attempts     int             `json:"attempts"`
}

// ============ Service ============

type SocialRecoveryService struct {
	config *Config
	redis  *redis.Client
}

func NewSocialRecoveryService(config *Config) (*SocialRecoveryService, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.RedisURL,
		Password: "",
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis not available: %v", err)
	}

	return &SocialRecoveryService{
		config: config,
		redis:  redisClient,
	}, nil
}

// Create Wallet
func (s *SocialRecoveryService) CreateWallet(ctx context.Context, address, chain string, threshold int, guardians []Guardian) (*Wallet, error) {
	wallet := &Wallet{
		ID:            uuid.New().String(),
		Address:       address,
		Chain:         chain,
		Guardians:     guardians,
		Threshold:     threshold,
		Status:        "active",
		RecoveryCount: 0,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if s.redis != nil {
		data, _ := json.Marshal(wallet)
		s.redis.Set(ctx, fmt.Sprintf("wallet:%s", wallet.ID), data, 0)
		s.redis.Set(ctx, fmt.Sprintf("wallet:address:%s:%s", chain, address), wallet.ID, 0)
	}

	return wallet, nil
}

// Add Guardian
func (s *SocialRecoveryService) AddGuardian(ctx context.Context, walletID string, guardian Guardian) (*Wallet, error) {
	wallet, err := s.GetWallet(ctx, walletID)
	if err != nil {
		return nil, err
	}

	if len(wallet.Guardians) >= s.config.MaxGuardians {
		return nil, fmt.Errorf("maximum guardians reached")
	}

	guardian.ID = uuid.New().String()
	guardian.InvitedAt = time.Now()

	wallet.Guardians = append(wallet.Guardians, guardian)
	wallet.UpdatedAt = time.Now()

	if s.redis != nil {
		data, _ := json.Marshal(wallet)
		s.redis.Set(ctx, fmt.Sprintf("wallet:%s", wallet.ID), data, 0)
	}

	// Send invitation (in production, would send email/SMS)
	go s.sendGuardianInvitation(guardian)

	return wallet, nil
}

// Remove Guardian
func (s *SocialRecoveryService) RemoveGuardian(ctx context.Context, walletID, guardianID string) (*Wallet, error) {
	wallet, err := s.GetWallet(ctx, walletID)
	if err != nil {
		return nil, err
	}

	for i, g := range wallet.Guardians {
		if g.ID == guardianID {
			wallet.Guardians = append(wallet.Guardians[:i], wallet.Guardians[i+1:]...)
			break
		}
	}

	wallet.UpdatedAt = time.Now()

	if s.redis != nil {
		data, _ := json.Marshal(wallet)
		s.redis.Set(ctx, fmt.Sprintf("wallet:%s", wallet.ID), data, 0)
	}

	return wallet, nil
}

// Initiate Recovery
func (s *SocialRecoveryService) InitiateRecovery(ctx context.Context, walletID, newOwnerAddress string) (*RecoveryRequest, error) {
	wallet, err := s.GetWallet(ctx, walletID)
	if err != nil {
		return nil, err
	}

	if len(wallet.Guardians) < wallet.Threshold {
		return nil, fmt.Errorf("not enough guardians")
	}

	// Check all guardians are confirmed
	confirmedCount := 0
	for _, g := range wallet.Guardians {
		if g.Confirmed {
			confirmedCount++
		}
	}

	if confirmedCount < wallet.Threshold {
		return nil, fmt.Errorf("not enough confirmed guardians")
	}

	request := &RecoveryRequest{
		ID:              uuid.New().String(),
		WalletID:        walletID,
		NewOwnerAddress: newOwnerAddress,
		Status:          RecoveryPending,
		ThresholdMet:    false,
		Signatures:      []GuardianSignature{},
		DelayEnd:        time.Now().Add(s.config.RecoveryDelay),
		CreatedAt:       time.Now(),
		ExpiresAt:       time.Now().Add(7 * 24 * time.Hour),
	}

	// Generate challenge
	challenge := generateChallenge()
	request.Signatures = append(request.Signatures, GuardianSignature{
		Message: fmt.Sprintf("Recovery challenge: %s", challenge),
	})

	if s.redis != nil {
		data, _ := json.Marshal(request)
		s.redis.Set(ctx, fmt.Sprintf("recovery:%s", request.ID), data, 7*24*time.Hour)
		s.redis.Set(ctx, fmt.Sprintf("recovery:wallet:%s", walletID), request.ID, 7*24*time.Hour)
	}

	// Notify guardians
	go s.notifyGuardiansOfRecovery(*wallet, *request)

	return request, nil
}

// Sign Recovery
func (s *SocialRecoveryService) SignRecovery(ctx context.Context, requestID, guardianID, signature string) (*RecoveryRequest, error) {
	request, err := s.GetRecoveryRequest(ctx, requestID)
	if err != nil {
		return nil, err
	}

	if request.Status != RecoveryPending {
		return nil, fmt.Errorf("recovery not pending")
	}

	wallet, err := s.GetWallet(ctx, request.WalletID)
	if err != nil {
		return nil, err
	}

	// Find guardian
	var guardian Guardian
	for _, g := range wallet.Guardians {
		if g.ID == guardianID {
			guardian = g
			break
		}
	}

	if guardian.ID == "" {
		return nil, fmt.Errorf("guardian not found")
	}

	// Verify signature (in production, would verify with guardian address)
	// For now, just add the signature
	request.Signatures = append(request.Signatures, GuardianSignature{
		GuardianID: guardianID,
		Signature:  signature,
		SignedAt:   time.Now(),
		Message:    fmt.Sprintf("Approve recovery to %s", request.NewOwnerAddress),
	})

	// Check threshold
	if len(request.Signatures) >= wallet.Threshold {
		request.ThresholdMet = true
		request.Status = RecoveryApproved
	}

	if s.redis != nil {
		data, _ := json.Marshal(request)
		s.redis.Set(ctx, fmt.Sprintf("recovery:%s", request.ID), data, 7*24*time.Hour)
	}

	return request, nil
}

// Execute Recovery (after delay)
func (s *SocialRecoveryService) ExecuteRecovery(ctx context.Context, requestID string) (*RecoveryRequest, error) {
	request, err := s.GetRecoveryRequest(ctx, requestID)
	if err != nil {
		return nil, err
	}

	if !request.ThresholdMet {
		return nil, fmt.Errorf("threshold not met")
	}

	if time.Now().Before(request.DelayEnd) {
		return nil, fmt.Errorf("delay period not complete")
	}

	request.Status = RecoveryExecuting

	// In production, would execute the actual blockchain transaction
	now := time.Now()
	request.ExecutedAt = &now
	request.Status = RecoveryCompleted

	// Update wallet
	wallet, err := s.GetWallet(ctx, request.WalletID)
	if err == nil {
		wallet.RecoveryCount++
		wallet.UpdatedAt = time.Now()
		if s.redis != nil {
			data, _ := json.Marshal(wallet)
			s.redis.Set(ctx, fmt.Sprintf("wallet:%s", wallet.ID), data, 0)
		}
	}

	if s.redis != nil {
		data, _ := json.Marshal(request)
		s.redis.Set(ctx, fmt.Sprintf("recovery:%s", request.ID), data, 7*24*time.Hour)
	}

	return request, nil
}

// Cancel Recovery
func (s *SocialRecoveryService) CancelRecovery(ctx context.Context, requestID string) (*RecoveryRequest, error) {
	request, err := s.GetRecoveryRequest(ctx, requestID)
	if err != nil {
		return nil, err
	}

	request.Status = RecoveryCancelled
	now := time.Now()
	request.CancelledAt = &now

	if s.redis != nil {
		data, _ := json.Marshal(request)
		s.redis.Set(ctx, fmt.Sprintf("recovery:%s", request.ID), data, 7*24*time.Hour)
	}

	return request, nil
}

// Get Wallet
func (s *SocialRecoveryService) GetWallet(ctx context.Context, walletID string) (*Wallet, error) {
	if s.redis == nil {
		return nil, fmt.Errorf("wallet not found")
	}

	data, err := s.redis.Get(ctx, fmt.Sprintf("wallet:%s", walletID)).Bytes()
	if err != nil {
		return nil, fmt.Errorf("wallet not found")
	}

	var wallet Wallet
	json.Unmarshal(data, &wallet)
	return &wallet, nil
}

// Get Wallet by Address
func (s *SocialRecoveryService) GetWalletByAddress(ctx context.Context, chain, address string) (*Wallet, error) {
	if s.redis == nil {
		return nil, fmt.Errorf("wallet not found")
	}

	walletID, err := s.redis.Get(ctx, fmt.Sprintf("wallet:address:%s:%s", chain, address)).Result()
	if err != nil {
		return nil, fmt.Errorf("wallet not found")
	}

	return s.GetWallet(ctx, walletID)
}

// Get Recovery Request
func (s *SocialRecoveryService) GetRecoveryRequest(ctx context.Context, requestID string) (*RecoveryRequest, error) {
	if s.redis == nil {
		return nil, fmt.Errorf("request not found")
	}

	data, err := s.redis.Get(ctx, fmt.Sprintf("recovery:%s", requestID)).Bytes()
	if err != nil {
		return nil, fmt.Errorf("request not found")
	}

	var request RecoveryRequest
	json.Unmarshal(data, &request)
	return &request, nil
}

// Get Active Recovery Request for Wallet
func (s *SocialRecoveryService) GetActiveRecoveryRequest(ctx context.Context, walletID string) (*RecoveryRequest, error) {
	if s.redis == nil {
		return nil, nil
	}

	requestID, err := s.redis.Get(ctx, fmt.Sprintf("recovery:wallet:%s", walletID)).Result()
	if err != nil {
		return nil, nil
	}

	return s.GetRecoveryRequest(ctx, requestID)
}

// Verify Guardian
func (s *SocialRecoveryService) VerifyGuardian(ctx context.Context, walletID, guardianID, code string) error {
	// In production, would verify the code via email/SMS
	wallet, err := s.GetWallet(ctx, walletID)
	if err != nil {
		return err
	}

	for i, g := range wallet.Guardians {
		if g.ID == guardianID {
			now := time.Now()
			wallet.Guardians[i].Confirmed = true
			wallet.Guardians[i].ConfirmedAt = &now
			wallet.UpdatedAt = time.Now()

			if s.redis != nil {
				data, _ := json.Marshal(wallet)
				s.redis.Set(ctx, fmt.Sprintf("wallet:%s", wallet.ID), data, 0)
			}

			return nil
		}
	}

	return fmt.Errorf("guardian not found")
}

// Helper functions
func generateChallenge() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *SocialRecoveryService) sendGuardianInvitation(guardian Guardian) {
	log.Printf("Sending invitation to guardian: %s", guardian.Address)
	// In production, would send email/SMS
}

func (s *SocialRecoveryService) notifyGuardiansOfRecovery(wallet Wallet, request RecoveryRequest) {
	log.Printf("Notifying guardians of recovery request: %s", request.ID)
	// In production, would send notifications to all guardians
}

// ============ HTTP Handlers ============

type Handler struct {
	service *SocialRecoveryService
}

func NewHandler(service *SocialRecoveryService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateWallet(c *gin.Context) {
	var req struct {
		Address   string      `json:"address" binding:"required"`
		Chain     string      `json:"chain" binding:"required"`
		Threshold int         `json:"threshold" binding:"required"`
		Guardians []Guardian  `json:"guardians"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wallet, err := h.service.CreateWallet(c.Request.Context(), req.Address, req.Chain, req.Threshold, req.Guardians)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, wallet)
}

func (h *Handler) AddGuardian(c *gin.Context) {
	walletID := c.Param("walletId")

	var req struct {
		Guardian Guardian `json:"guardian" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wallet, err := h.service.AddGuardian(c.Request.Context(), walletID, req.Guardian)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, wallet)
}

func (h *Handler) InitiateRecovery(c *gin.Context) {
	walletID := c.Param("walletId")

	var req struct {
		NewOwnerAddress string `json:"newOwnerAddress" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	request, err := h.service.InitiateRecovery(c.Request.Context(), walletID, req.NewOwnerAddress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, request)
}

func (h *Handler) SignRecovery(c *gin.Context) {
	requestID := c.Param("requestId")

	var req struct {
		GuardianID string `json:"guardianId" binding:"required"`
		Signature  string `json:"signature" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	request, err := h.service.SignRecovery(c.Request.Context(), requestID, req.GuardianID, req.Signature)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, request)
}

func (h *Handler) ExecuteRecovery(c *gin.Context) {
	requestID := c.Param("requestId")

	request, err := h.service.ExecuteRecovery(c.Request.Context(), requestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, request)
}

func (h *Handler) GetWallet(c *gin.Context) {
	walletID := c.Param("walletId")

	wallet, err := h.service.GetWallet(c.Request.Context(), walletID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, wallet)
}

func (h *Handler) GetRecoveryRequest(c *gin.Context) {
	requestID := c.Param("requestId")

	request, err := h.service.GetRecoveryRequest(c.Request.Context(), requestID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, request)
}

// ============ Main ============

func main() {
	config := &Config{
		Port:            getEnv("PORT", "8080"),
		RedisURL:        getEnv("REDIS_URL", "localhost:6379"),
		RecoveryDelay:   24 * time.Hour,
		MaxGuardians:    10,
		MinGuardians:    2,
		EmailEnabled:    getEnv("EMAIL_ENABLED", "true") == "true",
		SMSEnabled:     getEnv("SMS_ENABLED", "true") == "true",
	}

	service, err := NewSocialRecoveryService(config)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	handler := NewHandler(service)

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API routes
	api := router.Group("/api/v1")
	{
		api.POST("/wallets", handler.CreateWallet)
		api.GET("/wallets/:walletId", handler.GetWallet)
		api.POST("/wallets/:walletId/guardians", handler.AddGuardian)

		api.POST("/wallets/:walletId/recovery", handler.InitiateRecovery)
		api.POST("/recovery/:requestId/sign", handler.SignRecovery)
		api.POST("/recovery/:requestId/execute", handler.ExecuteRecovery)
		api.GET("/recovery/:requestId", handler.GetRecoveryRequest)
	}

	// Start server
	addr := fmt.Sprintf(":%s", config.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		log.Printf("Starting Social Recovery service on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
