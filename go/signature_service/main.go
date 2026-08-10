package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Server   ServerConfig
	Redis    RedisConfig
	Security SecurityConfig
}

type ServerConfig struct {
	Port string
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type SecurityConfig struct {
	RequireApproval bool
	MaxSignaturesPerHour int
	AllowedChains   []uint64
}

func LoadConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8444"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     6379,
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       0,
		},
		Security: SecurityConfig{
			RequireApproval:     getEnv("REQUIRE_APPROVAL", "false") == "true",
			MaxSignaturesPerHour: 1000,
			AllowedChains:        []uint64{1, 5, 137, 42161, 10, 43114, 56, 8453},
		},
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

type SignatureRequest struct {
	ID              string    `json:"id"`
	UserID         string    `json:"userId"`
	WalletAddress  string    `json:"walletAddress"`
	ChainID        uint64    `json:"chainId"`
	Message        string    `json:"message"`
	MessageHash    string    `json:"messageHash"`
	Signature      string    `json:"signature,omitempty"`
	Status         string    `json:"status"` // pending, signed, failed, cancelled
	SignatureType  string    `json:"signatureType"` // personal_sign, eth_sign, typed_data
	IPAddress      string    `json:"ipAddress"`
	UserAgent      string    `json:"userAgent"`
	ApprovedBy     string    `json:"approvedBy,omitempty"`
	ApprovedAt     *time.Time `json:"approvedAt,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	CompletedAt    *time.Time `json:"completedAt,omitempty"`
}

type SignatureApproval struct {
	ID              string    `json:"id"`
	RequestID       string    `json:"requestId"`
	ApproverID      string    `json:"approverId"`
	ApproverEmail   string    `json:"approverEmail"`
	Status          string    `json:"status"` // approved, rejected
	Notes           string    `json:"notes"`
	CreatedAt       time.Time `json:"createdAt"`
}

type KeyRotation struct {
	ID              string    `json:"id"`
	UserID          string    `json:"userId"`
	OldPublicKey    string    `json:"oldPublicKey"`
	NewPublicKey    string    `json:"newPublicKey"`
	Status          string    `json:"status"` // pending, completed, failed
	RotationType    string    `json:"rotationType"` // scheduled, emergency, compromised
	CreatedAt       time.Time `json:"createdAt"`
	CompletedAt     *time.Time `json:"completedAt,omitempty"`
}

type AuditLog struct {
	ID          string    `json:"id"`
	UserID      string    `json:"userId"`
	Action      string    `json:"action"`
	Resource    string    `json:"resource"`
	ResourceID  string    `json:"resourceId"`
	Details     string    `json:"details"`
	IPAddress   string    `json:"ipAddress"`
	UserAgent   string    `json:"userAgent"`
	CreatedAt   time.Time `json:"createdAt"`
}

// ============================================================================
// Services
// ============================================================================

type SignatureService struct {
	config      *Config
	redis       *redis.Client
	requests    map[string]*SignatureRequest
	approvals   map[string]*SignatureApproval
	rotations   map[string]*KeyRotation
	auditLogs   []AuditLog
	mu          sync.RWMutex
	rateLimiter *RateLimiter
}

func NewSignatureService(config *Config, redisClient *redis.Client) *SignatureService {
	return &SignatureService{
		config:      config,
		redis:       redisClient,
		requests:    make(map[string]*SignatureRequest),
		approvals:   make(map[string]*SignatureApproval),
		rotations:   make(map[string]*KeyRotation),
		auditLogs:   make([]AuditLog, 0),
		rateLimiter: NewRateLimiter(config.Security.MaxSignaturesPerHour),
	}
}

// ============================================================================
// Rate Limiter
// ============================================================================

type RateLimiter struct {
	requests map[string][]time.Time
	mu       sync.Mutex
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   time.Hour,
	}
}

func (r *RateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	oneHourAgo := now.Add(-r.window)

	// Clean old requests
	var recent []time.Time
	for _, t := range r.requests[key] {
		if t.After(oneHourAgo) {
			recent = append(recent, t)
		}
	}

	if len(recent) >= r.limit {
		r.requests[key] = recent
		return false
	}

	r.requests[key] = append(recent, now)
	return true
}

// ============================================================================
// Signature Operations
// ============================================================================

func (s *SignatureService) CreateSignatureRequest(
	userID,
	walletAddress string,
	chainID uint64,
	message string,
	signatureType string,
	ipAddress,
	userAgent string,
) (*SignatureRequest, error) {
	// Check rate limit
	if !s.rateLimiter.Allow(userID) {
		return nil, fmt.Errorf("rate limit exceeded")
	}

	// Validate chain
	if !s.isChainAllowed(chainID) {
		return nil, fmt.Errorf("chain %d not allowed", chainID)
	}

	// Validate wallet address
	if !common.IsHexAddress(walletAddress) {
		return nil, fmt.Errorf("invalid wallet address")
	}

	// Calculate message hash
	messageHash := calculateMessageHash(message)

	request := &SignatureRequest{
		ID:             uuid.New().String(),
		UserID:         userID,
		WalletAddress:  walletAddress,
		ChainID:        chainID,
		Message:        message,
		MessageHash:    messageHash,
		Status:         "pending",
		SignatureType:  signatureType,
		IPAddress:      ipAddress,
		UserAgent:      userAgent,
		CreatedAt:      time.Now(),
	}

	s.mu.Lock()
	s.requests[request.ID] = request
	s.mu.Unlock()

	// Log audit
	s.logAudit(userID, "CREATE_SIGNATURE_REQUEST", "signature_request", request.ID,
		fmt.Sprintf("Created request for wallet %s on chain %d", walletAddress, chainID),
		ipAddress, userAgent)

	// If approval required, don't auto-sign
	if s.config.Security.RequireApproval {
		return request, nil
	}

	// Auto-sign for demo (in production would require approval)
	return request, nil
}

func (s *SignatureService) SignMessage(
	requestID string,
	privateKey *ecdsa.PrivateKey,
) (*SignatureRequest, error) {
	s.mu.RLock()
	request, ok := s.requests[requestID]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("request not found")
	}

	if request.Status != "pending" {
		return nil, fmt.Errorf("request already processed")
	}

	// Sign the message
	var signature []byte
	var err error

	switch request.SignatureType {
	case "personal_sign":
		signature, err = crypto.Sign(accounts.TextHash([]byte(request.Message)), privateKey)
	case "eth_sign":
		// eth_sign signs the raw keccak256 hash of the message (no Ethereum prefix).
		signature, err = crypto.Sign(crypto.Keccak256([]byte(request.Message)), privateKey)
	default:
		signature, err = crypto.Sign(accounts.TextHash([]byte(request.Message)), privateKey)
	}

	if err != nil {
		request.Status = "failed"
		return nil, fmt.Errorf("signing failed: %w", err)
	}

	request.Signature = hexutil.Encode(signature)
	request.Status = "signed"
	now := time.Now()
	request.CompletedAt = &now

	// Log audit
	s.logAudit(request.UserID, "SIGN_MESSAGE", "signature_request", request.ID,
		fmt.Sprintf("Signed message with hash %s", request.MessageHash),
		request.IPAddress, request.UserAgent)

	return request, nil
}

func (s *SignatureService) GetSignatureRequest(id string) (*SignatureRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	request, ok := s.requests[id]
	if !ok {
		return nil, fmt.Errorf("request not found")
	}

	return request, nil
}

func (s *SignatureService) GetUserSignatureRequests(userID string, limit int) ([]SignatureRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var requests []SignatureRequest
	count := 0

	for _, req := range s.requests {
		if req.UserID == userID {
			requests = append(requests, *req)
			count++
			if count >= limit {
				break
			}
		}
	}

	return requests, nil
}

func (s *SignatureService) CancelSignatureRequest(id, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	request, ok := s.requests[id]
	if !ok {
		return fmt.Errorf("request not found")
	}

	if request.UserID != userID {
		return fmt.Errorf("unauthorized")
	}

	if request.Status != "pending" {
		return fmt.Errorf("request cannot be cancelled")
	}

	request.Status = "cancelled"

	// Log audit
	s.logAudit(userID, "CANCEL_SIGNATURE_REQUEST", "signature_request", id,
		"Cancelled signature request", request.IPAddress, request.UserAgent)

	return nil
}

// ============================================================================
// Approval Management
// ============================================================================

func (s *SignatureService) ApproveSignatureRequest(
	requestID,
	approverID,
	approverEmail,
	notes string,
) (*SignatureApproval, error) {
	s.mu.RLock()
	request, ok := s.requests[requestID]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("request not found")
	}

	if request.Status != "pending" {
		return nil, fmt.Errorf("request already processed")
	}

	approval := &SignatureApproval{
		ID:            uuid.New().String(),
		RequestID:     requestID,
		ApproverID:    approverID,
		ApproverEmail: approverEmail,
		Status:        "approved",
		Notes:         notes,
		CreatedAt:     time.Now(),
	}

	request.ApprovedBy = approverID
	now := time.Now()
	request.ApprovedAt = &now

	s.mu.Lock()
	s.approvals[approval.ID] = approval
	s.mu.Unlock()

	// Log audit
	s.logAudit(approverID, "APPROVE_SIGNATURE", "signature_request", requestID,
		fmt.Sprintf("Approved request from user %s", request.UserID),
		"", "")

	return approval, nil
}

func (s *SignatureService) RejectSignatureRequest(
	requestID,
	approverID,
	approverEmail,
	notes string,
) error {
	s.mu.RLock()
	request, ok := s.requests[requestID]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("request not found")
	}

	request.Status = "rejected"

	approval := &SignatureApproval{
		ID:            uuid.New().String(),
		RequestID:     requestID,
		ApproverID:    approverID,
		ApproverEmail: approverEmail,
		Status:        "rejected",
		Notes:         notes,
		CreatedAt:     time.Now(),
	}

	s.mu.Lock()
	s.approvals[approval.ID] = approval
	s.mu.Unlock()

	// Log audit
	s.logAudit(approverID, "REJECT_SIGNATURE", "signature_request", requestID,
		fmt.Sprintf("Rejected request from user %s", request.UserID),
		"", "")

	return nil
}

// ============================================================================
// Key Rotation
// ============================================================================

func (s *SignatureService) InitiateKeyRotation(
	userID,
	oldPublicKey,
	newPublicKey,
	rotationType string,
) (*KeyRotation, error) {
	rotation := &KeyRotation{
		ID:           uuid.New().String(),
		UserID:       userID,
		OldPublicKey: oldPublicKey,
		NewPublicKey: newPublicKey,
		Status:       "pending",
		RotationType: rotationType,
		CreatedAt:    time.Now(),
	}

	s.mu.Lock()
	s.rotations[rotation.ID] = rotation
	s.mu.Unlock()

	// Log audit
	s.logAudit(userID, "INITIATE_KEY_ROTATION", "key_rotation", rotation.ID,
		fmt.Sprintf("Initiated %s key rotation", rotationType),
		"", "")

	return rotation, nil
}

func (s *SignatureService) CompleteKeyRotation(rotationID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rotation, ok := s.rotations[rotationID]
	if !ok {
		return fmt.Errorf("rotation not found")
	}

	if rotation.Status != "pending" {
		return fmt.Errorf("rotation already processed")
	}

	rotation.Status = "completed"
	now := time.Now()
	rotation.CompletedAt = &now

	// Log audit
	s.logAudit(rotation.UserID, "COMPLETE_KEY_ROTATION", "key_rotation", rotationID,
		"Completed key rotation",
		"", "")

	return nil
}

// ============================================================================
// Verification
// ============================================================================

func (s *SignatureService) VerifySignature(
	walletAddress,
	message,
	signature string,
) (bool, error) {
	// Parse signature
	sigBytes, err := hexutil.Decode(signature)
	if err != nil {
		return false, fmt.Errorf("invalid signature format")
	}

	// Recover public key
	if len(sigBytes) != 65 {
		return false, fmt.Errorf("invalid signature length")
	}

	sigBytes[64] = 0 // V value

	pubKey, err := crypto.SigToPub(accounts.TextHash([]byte(message)), sigBytes)
	if err != nil {
		return false, fmt.Errorf("signature verification failed")
	}

	recoveredAddress := crypto.PubkeyToAddress(*pubKey).Hex()

	// Compare addresses (case insensitive)
	return strings.EqualFold(recoveredAddress, walletAddress), nil
}

// ============================================================================
// Utilities
// ============================================================================

func (s *SignatureService) isChainAllowed(chainID uint64) bool {
	for _, allowed := range s.config.Security.AllowedChains {
		if chainID == allowed {
			return true
		}
	}
	return false
}

func (s *SignatureService) logAudit(userID, action, resource, resourceID, details, ipAddress, userAgent string) {
	log := AuditLog{
		ID:         uuid.New().String(),
		UserID:     userID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    details,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		CreatedAt:  time.Now(),
	}

	s.mu.Lock()
	s.auditLogs = append(s.auditLogs, log)
	s.mu.Unlock()
}

func (s *SignatureService) GetAuditLogs(userID string, limit int) []AuditLog {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var logs []AuditLog
	count := 0

	for i := len(s.auditLogs) - 1; i >= 0 && count < limit; i-- {
		if s.auditLogs[i].UserID == userID {
			logs = append(logs, s.auditLogs[i])
			count++
		}
	}

	return logs
}

func calculateMessageHash(message string) string {
	// Ethereum personal_sign hash: keccak256("\x19Ethereum Signed Message:\n" + len + msg).
	// Must use keccak256 (NOT sha256) so the stored hash matches the signature
	// produced by crypto.Sign(accounts.TextHash(...)).
	prefix := "\x19Ethereum Signed Message:\n"
	fullMessage := prefix + fmt.Sprintf("%d", len(message)) + message
	hash := crypto.Keccak256([]byte(fullMessage))
	return hex.EncodeToString(hash)
}

// ============================================================================
// Generate Key Pair
// ============================================================================

func GenerateKeyPair() (string, string, error) {
	// Ethereum uses secp256k1 (NOT NIST P-256). crypto.GenerateKey() produces a
	// cryptographically-secure secp256k1 key compatible with Ethereum addresses,
	// EIP-191/712 signing, and eth_sendRawTransaction.
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate key: %w", err)
	}

	privateKeyHex := hex.EncodeToString(privateKey.D.Bytes())
	publicKeyHex := hex.EncodeToString(crypto.FromECDSAPub(&privateKey.PublicKey))

	return privateKeyHex, publicKeyHex, nil
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func (s *SignatureService) RegisterRoutes(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "signature-service"})
	})

	api := r.Group("/api/v1")
	{
		// Signature requests
		api.POST("/signature/request", s.handleCreateRequest)
		api.GET("/signature/requests", s.handleGetUserRequests)
		api.GET("/signature/request/:id", s.handleGetRequest)
		api.POST("/signature/request/:id/cancel", s.handleCancelRequest)
		api.POST("/signature/request/:id/sign", s.handleSignRequest)
		api.POST("/signature/verify", s.handleVerifySignature)

		// Approvals
		api.POST("/signature/request/:id/approve", s.handleApproveRequest)
		api.POST("/signature/request/:id/reject", s.handleRejectRequest)

		// Key rotation
		api.POST("/key-rotation", s.handleInitiateRotation)
		api.POST("/key-rotation/:id/complete", s.handleCompleteRotation)

		// Audit
		api.GET("/audit-logs", s.handleGetAuditLogs)

		// Utility
		api.POST("/key/generate", s.handleGenerateKey)
	}
}

func (s *SignatureService) handleCreateRequest(c *gin.Context) {
	var req struct {
		UserID        string `json:"userId" binding:"required"`
		WalletAddress string `json:"walletAddress" binding:"required"`
		ChainID       uint64 `json:"chainId" binding:"required"`
		Message       string `json:"message" binding:"required"`
		SignatureType string `json:"signatureType"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sigType := req.SignatureType
	if sigType == "" {
		sigType = "personal_sign"
	}

	request, err := s.CreateSignatureRequest(
		req.UserID,
		req.WalletAddress,
		req.ChainID,
		req.Message,
		sigType,
		c.ClientIP(),
		c.Request.UserAgent(),
	)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, request)
}

func (s *SignatureService) handleGetUserRequests(c *gin.Context) {
	userID := c.Query("userId")
	limit := 50

	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId required"})
		return
	}

	requests, err := s.GetUserSignatureRequests(userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"requests": requests})
}

func (s *SignatureService) handleGetRequest(c *gin.Context) {
	id := c.Param("id")

	request, err := s.GetSignatureRequest(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, request)
}

func (s *SignatureService) handleCancelRequest(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		UserID string `json:"userId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.CancelSignatureRequest(id, req.UserID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Request cancelled"})
}

func (s *SignatureService) handleSignRequest(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		PrivateKey string `json:"privateKey" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse private key
	privateKeyBytes, err := hex.DecodeString(req.PrivateKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid private key"})
		return
	}

	privateKey, err := crypto.ToECDSA(privateKeyBytes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid private key"})
		return
	}

	request, err := s.SignMessage(id, privateKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, request)
}

func (s *SignatureService) handleVerifySignature(c *gin.Context) {
	var req struct {
		WalletAddress string `json:"walletAddress" binding:"required"`
		Message       string `json:"message" binding:"required"`
		Signature     string `json:"signature" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	valid, err := s.VerifySignature(req.WalletAddress, req.Message, req.Signature)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": valid})
}

func (s *SignatureService) handleApproveRequest(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		ApproverID   string `json:"approverId" binding:"required"`
		ApproverEmail string `json:"approverEmail" binding:"required"`
		Notes        string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	approval, err := s.ApproveSignatureRequest(id, req.ApproverID, req.ApproverEmail, req.Notes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, approval)
}

func (s *SignatureService) handleRejectRequest(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		ApproverID   string `json:"approverId" binding:"required"`
		ApproverEmail string `json:"approverEmail" binding:"required"`
		Notes        string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.RejectSignatureRequest(id, req.ApproverID, req.ApproverEmail, req.Notes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Request rejected"})
}

func (s *SignatureService) handleInitiateRotation(c *gin.Context) {
	var req struct {
		UserID       string `json:"userId" binding:"required"`
		OldPublicKey string `json:"oldPublicKey" binding:"required"`
		NewPublicKey string `json:"newPublicKey" binding:"required"`
		RotationType string `json:"rotationType"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rotType := req.RotationType
	if rotType == "" {
		rotType = "scheduled"
	}

	rotation, err := s.InitiateKeyRotation(req.UserID, req.OldPublicKey, req.NewPublicKey, rotType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, rotation)
}

func (s *SignatureService) handleCompleteRotation(c *gin.Context) {
	id := c.Param("id")

	if err := s.CompleteKeyRotation(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Key rotation completed"})
}

func (s *SignatureService) handleGetAuditLogs(c *gin.Context) {
	userID := c.Query("userId")
	limit := 100

	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId required"})
		return
	}

	logs := s.GetAuditLogs(userID, limit)
	c.JSON(http.StatusOK, gin.H{"auditLogs": logs})
}

func (s *SignatureService) handleGenerateKey(c *gin.Context) {
	privateKey, publicKey, err := GenerateKeyPair()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"privateKey": privateKey,
		"publicKey":  publicKey,
	})
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := LoadConfig()

	// Initialize Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Redis.Host, config.Redis.Port),
		Password: config.Redis.Password,
		DB:       config.Redis.DB,
	})

	// Initialize service
	service := NewSignatureService(config, redisClient)

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// Register routes
	service.RegisterRoutes(r)

	// Create server
	srv := &http.Server{
		Addr:    ":" + config.Server.Port,
		Handler: r,
	}

	// Start server
	go func() {
		log.Printf("Signature service starting on port %s", config.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx := context.Background()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}

func c() {
	// Dummy function to fix syntax
}
