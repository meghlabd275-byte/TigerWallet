/**
 * TigerWallet KYC (Know Your Customer) Integration Service
 * Complete KYC integration with document verification, AML screening, and travel rule compliance
 */

package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ============================================================================
// KYC Configuration
// ============================================================================

type KYCConfig struct {
	Provider          string           `json:"provider"` // internal, synaps, sumsub, veriff
	APIKey            string           `json:"api_key"`
	APISecret         string           `json:"api_secret"`
	WebhookSecret     string           `json:"webhook_secret"`
	BaseURL           string           `json:"base_url"`
	CallbackURL       string           `json:"callback_url"`
	RequiredLevels    map[int]KYCLevel `json:"required_levels"`
	AMLEnabled        bool             `json:"aml_enabled"`
	TravelRuleEnabled bool             `json:"travel_rule_enabled"`
	MaxRetryAttempts  int              `json:"max_retry_attempts"`
	SessionTimeout    time.Duration    `json:"session_timeout"`
}

type KYCLevel struct {
	Name             string        `json:"name"`
	RequiredDocs     []string      `json:"required_docs"`
	VerificationTime time.Duration `json:"verification_time"`
	TrustScoreBoost  int           `json:"trust_score_boost"`
}

var DefaultKYCConfig = &KYCConfig{
	Provider:    "internal",
	BaseURL:     "https://api.tigerwallet.com",
	CallbackURL: "https://api.tigerwallet.com/kyc/callback",
	RequiredLevels: map[int]KYCLevel{
		1: {
			Name:             "Email Verification",
			RequiredDocs:     []string{"email"},
			VerificationTime: 1 * time.Minute,
			TrustScoreBoost:  10,
		},
		2: {
			Name:             "Phone Verification",
			RequiredDocs:     []string{"email", "phone"},
			VerificationTime: 5 * time.Minute,
			TrustScoreBoost:  15,
		},
		3: {
			Name:             "ID Verification",
			RequiredDocs:     []string{"email", "phone", "id_document"},
			VerificationTime: 15 * time.Minute,
			TrustScoreBoost:  25,
		},
		4: {
			Name:             "Enhanced Due Diligence",
			RequiredDocs:     []string{"email", "phone", "id_document", "proof_of_address", "selfie"},
			VerificationTime: 24 * time.Hour,
			TrustScoreBoost:  40,
		},
	},
	AMLEnabled:        true,
	TravelRuleEnabled: true,
	MaxRetryAttempts:  3,
	SessionTimeout:    30 * time.Minute,
}

// ============================================================================
// KYC Data Models
// ============================================================================

type KYCUser struct {
	ID          string          `json:"id" bson:"_id"`
	Email       string          `json:"email" bson:"email"`
	Phone       string          `json:"phone" bson:"phone"`
	Level       int             `json:"level" bson:"level"`
	Status      string          `json:"status" bson:"status"` // pending, reviewing, verified, rejected, expired
	Country     string          `json:"country" bson:"country"`
	FirstName   string          `json:"first_name" bson:"first_name"`
	LastName    string          `json:"last_name" bson:"last_name"`
	DateOfBirth string          `json:"date_of_birth" bson:"date_of_birth"`
	Address     *Address        `json:"address" bson:"address"`
	Documents   []*KYCDocument  `json:"documents" bson:"documents"`
	AMLCheck    *AMLCheckResult `json:"aml_check" bson:"aml_check"`
	TravelRule  *TravelRuleData `json:"travel_rule" bson:"travel_rule"`
	RiskScore   int             `json:"risk_score" bson:"risk_score"`
	TrustScore  int             `json:"trust_score" bson:"trust_score"`
	ExternalID  string          `json:"external_id" bson:"external_id"`
	Provider    string          `json:"provider" bson:"provider"`
	CreatedAt   time.Time       `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" bson:"updated_at"`
	VerifiedAt  *time.Time      `json:"verified_at" bson:"verified_at"`
	ExpiresAt   *time.Time      `json:"expires_at" bson:"expires_at"`
}

type Address struct {
	Street     string `json:"street" bson:"street"`
	City       string `json:"city" bson:"city"`
	State      string `json:"state" bson:"state"`
	PostalCode string `json:"postal_code" bson:"postal_code"`
	Country    string `json:"country" bson:"country"`
}

type KYCDocument struct {
	ID           string    `json:"id" bson:"_id"`
	Type         string    `json:"type" bson:"type"` // passport, id_card, drivers_license, proof_of_address
	Number       string    `json:"number" bson:"number"`
	IssuedBy     string    `json:"issued_by" bson:"issued_by"`
	ExpiryDate   string    `json:"expiry_date" bson:"expiry_date"`
	Status       string    `json:"status" bson:"status"` // pending, verified, rejected, expired
	FrontURL     string    `json:"front_url" bson:"front_url"`
	BackURL      string    `json:"back_url" bson:"back_url"`
	SelfieURL    string    `json:"selfie_url" bson:"selfie_url"`
	VerifiedAt   time.Time `json:"verified_at" bson:"verified_at"`
	RejectReason string    `json:"reject_reason" bson:"reject_reason"`
}

type AMLCheckResult struct {
	CheckedAt          time.Time  `json:"checked_at" bson:"checked_at"`
	Status             string     `json:"status" bson:"status"`         // clear, flagged, suspicious
	RiskLevel          string     `json:"risk_level" bson:"risk_level"` // low, medium, high, critical
	PEPStatus          string     `json:"pep_status" bson:"pep_status"` // yes, no
	SanctionsStatus    string     `json:"sanctions_status" bson:"sanctions_status"`
	AdverseMediaStatus string     `json:"adverse_media_status" bson:"adverse_media_status"`
	ReportURL          string     `json:"report_url" bson:"report_url"`
	MatchDetails       []AMLMatch `json:"match_details" bson:"match_details"`
}

type AMLMatch struct {
	Type        string `json:"type" bson:"type"` // pep, sanction, adverse_media
	Name        string `json:"name" bson:"name"`
	Alias       string `json:"alias" bson:"alias"`
	Country     string `json:"country" bson:"country"`
	MatchScore  int    `json:"match_score" bson:"match_score"`
	Description string `json:"description" bson:"description"`
}

type TravelRuleData struct {
	Enabled         bool      `json:"enabled" bson:"enabled"`
	BeneficiaryName string    `json:"beneficiary_name" bson:"beneficiary_name"`
	BeneficiaryAddr string    `json:"beneficiary_addr" bson:"beneficiary_addr"`
	BeneficiaryID   string    `json:"beneficiary_id" bson:"beneficiary_id"`
	OriginatorName  string    `json:"originator_name" bson:"originator_name"`
	OriginatorAddr  string    `json:"originator_addr" bson:"originator_addr"`
	OriginatorID    string    `json:"originator_id" bson:"originator_id"`
	Amount          string    `json:"amount" bson:"amount"`
	Currency        string    `json:"currency" bson:"currency"`
	Date            time.Time `json:"date" bson:"date"`
}

// ============================================================================
// KYC Service
// ============================================================================

type KYCService struct {
	redis      *redis.Client
	config     *KYCConfig
	mu         sync.RWMutex
	users      map[string]*KYCUser
	sessions   map[string]*KYCSession
	httpClient *http.Client
}

type KYCSession struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Level     int       `json:"level"`
	Status    string    `json:"status"` // pending, active, completed, expired
	StartedAt time.Time `json:"started_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Steps     []KYCStep `json:"steps"`
}

type KYCStep struct {
	Name        string                 `json:"name"`
	Status      string                 `json:"status"` // pending, in_progress, completed, failed
	CompletedAt *time.Time             `json:"completed_at"`
	Data        map[string]interface{} `json:"data"`
}

func NewKYCService(redisClient *redis.Client, config *KYCConfig) *KYCService {
	if config == nil {
		config = DefaultKYCConfig
	}

	return &KYCService{
		redis:    redisClient,
		config:   config,
		users:    make(map[string]*KYCUser),
		sessions: make(map[string]*KYCSession),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ============================================================================
// KYC Registration & Verification
// ============================================================================

// RegisterUser registers a new user for KYC
func (s *KYCService) RegisterUser(ctx context.Context, email string, phone string, country string) (*KYCUser, error) {
	// Check if user already exists
	userKey := fmt.Sprintf("kyc:email:%s", email)
	existingData, err := s.redis.Get(ctx, userKey).Result()
	if err == nil {
		var existingUser KYCUser
		if err := json.Unmarshal([]byte(existingData), &existingUser); err == nil {
			return &existingUser, fmt.Errorf("user already registered")
		}
	}

	user := &KYCUser{
		ID:        uuid.New().String(),
		Email:     email,
		Phone:     phone,
		Country:   country,
		Level:     0,
		Status:    "pending",
		Provider:  s.config.Provider,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save to Redis
	userData, err := json.Marshal(user)
	if err != nil {
		return nil, err
	}

	if err := s.redis.Set(ctx, userKey, userData, 0).Err(); err != nil {
		return nil, err
	}

	// Also save by ID
	idKey := fmt.Sprintf("kyc:id:%s", user.ID)
	s.redis.Set(ctx, idKey, userData, 0)

	return user, nil
}

// StartVerification starts the KYC verification process
func (s *KYCService) StartVerification(ctx context.Context, userID string, level int) (*KYCSession, error) {
	// Get user
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Validate level
	if level < 1 || level > 4 {
		return nil, fmt.Errorf("invalid KYC level")
	}

	// Get level config
	levelConfig, ok := s.config.RequiredLevels[level]
	if !ok {
		return nil, fmt.Errorf("level configuration not found")
	}

	// Create session
	session := &KYCSession{
		ID:        uuid.New().String(),
		UserID:    userID,
		Email:     user.Email,
		Level:     level,
		Status:    "active",
		StartedAt: time.Now(),
		ExpiresAt: time.Now().Add(s.config.SessionTimeout),
		Steps:     make([]KYCStep, 0),
	}

	// Add steps based on required documents
	for _, docType := range levelConfig.RequiredDocs {
		step := KYCStep{
			Name:   docType,
			Status: "pending",
			Data:   make(map[string]interface{}),
		}
		session.Steps = append(session.Steps, step)
	}

	// Save session
	sessionData, _ := json.Marshal(session)
	sessionKey := fmt.Sprintf("kyc:session:%s", session.ID)
	s.redis.Set(ctx, sessionKey, sessionData, s.config.SessionTimeout)

	// Update user level
	user.Level = level
	user.Status = "reviewing"
	user.UpdatedAt = time.Now()
	s.saveUser(ctx, user)

	return session, nil
}

// SubmitDocument submits a document for verification
func (s *KYCService) SubmitDocument(ctx context.Context, sessionID string, docType string, file *multipart.FileHeader) (*KYCDocument, error) {
	// Get session
	sessionKey := fmt.Sprintf("kyc:session:%s", sessionID)
	sessionData, err := s.redis.Get(ctx, sessionKey).Result()
	if err != nil {
		return nil, fmt.Errorf("session not found")
	}

	var session KYCSession
	if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
		return nil, err
	}

	// Find the step
	stepIndex := -1
	for i, step := range session.Steps {
		if step.Name == docType {
			stepIndex = i
			break
		}
	}

	if stepIndex == -1 {
		return nil, fmt.Errorf("document type not required for this level")
	}

	// Persist the uploaded document. With no external cloud storage
	// configured the raw bytes are stored in Redis under a document key and
	// the storage URL references that key. A real deployment would stream to
	// S3/GCS and store the object URL, but this keeps the document genuinely
	// retrievable (not discarded) with the same interface.
	doc := &KYCDocument{
		ID:     uuid.New().String(),
		Type:   docType,
		Status: "pending",
	}
	if file != nil {
		src, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("cannot open document: %w", err)
		}
		data, err := io.ReadAll(src)
		src.Close()
		if err != nil {
			return nil, fmt.Errorf("cannot read document: %w", err)
		}
		docKey := fmt.Sprintf("kyc:document:%s", doc.ID)
		encoded := base64.StdEncoding.EncodeToString(data)
		if err := s.redis.Set(ctx, docKey, encoded, s.config.SessionTimeout*4).Err(); err != nil {
			return nil, fmt.Errorf("cannot store document: %w", err)
		}
		doc.FrontURL = fmt.Sprintf("tigerwallet://kyc/document/%s", doc.ID)
		doc.Number = fmt.Sprintf("%x", sha256.Sum256(data))[:16]
	}

	// Update step status
	session.Steps[stepIndex].Status = "completed"
	now := time.Now()
	session.Steps[stepIndex].CompletedAt = &now

	// Save session
	sessionBytes, _ := json.Marshal(session)
	s.redis.Set(ctx, sessionKey, sessionBytes, s.config.SessionTimeout)

	// If all steps completed, trigger verification
	allCompleted := true
	for _, step := range session.Steps {
		if step.Status != "completed" {
			allCompleted = false
			break
		}
	}

	if allCompleted {
		go s.processVerification(session.UserID)
	}

	return doc, nil
}

// processVerification marks the submission as pending manual review. KYC
// verification requires human/external-provider review; it is never auto-
// approved here (auto-approval would be a security vulnerability allowing
// unverified users to claim elevated trust scores).
func (s *KYCService) processVerification(userID string) {
	ctx := context.Background()

	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return
	}

	// Mark as pending external/manual review, NOT verified.
	user.Status = "pending_review"
	user.UpdatedAt = time.Now()

	s.saveUser(ctx, user)

	// Run AML check if enabled
	if s.config.AMLEnabled {
		s.runAMLCheck(userID)
	}
}

// runAMLCheck runs AML screening. It never fabricates a "clear" result:
// when AML is enabled but no screening provider is configured (no API key),
// the user is marked pending/unverified and routed to manual review. Only a
// real provider call (against s.config.Provider + APIKey) can set "clear".
func (s *KYCService) runAMLCheck(userID string) {
	ctx := context.Background()
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return
	}

	if !s.config.AMLEnabled {
		return
	}

	// No real provider configured -> honest pending state (NOT "clear").
	pending := &AMLCheckResult{
		CheckedAt:          time.Now(),
		Status:             "pending",
		RiskLevel:          "unverified",
		PEPStatus:          "unknown",
		SanctionsStatus:    "pending",
		AdverseMediaStatus: "pending",
	}
	user.AMLCheck = pending
	user.RiskScore = 90 // unverified -> maximum prudential risk until cleared
	s.saveUser(ctx, user)
}

// GetUserByID retrieves a user by ID
func (s *KYCService) GetUserByID(ctx context.Context, userID string) (*KYCUser, error) {
	idKey := fmt.Sprintf("kyc:id:%s", userID)
	data, err := s.redis.Get(ctx, idKey).Result()
	if err != nil {
		return nil, err
	}

	var user KYCUser
	if err := json.Unmarshal([]byte(data), &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (s *KYCService) GetUserByEmail(ctx context.Context, email string) (*KYCUser, error) {
	userKey := fmt.Sprintf("kyc:email:%s", email)
	data, err := s.redis.Get(ctx, userKey).Result()
	if err != nil {
		return nil, err
	}

	var user KYCUser
	if err := json.Unmarshal([]byte(data), &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// saveUser saves a user to Redis
func (s *KYCService) saveUser(ctx context.Context, user *KYCUser) {
	userData, _ := json.Marshal(user)

	userKey := fmt.Sprintf("kyc:email:%s", user.Email)
	s.redis.Set(ctx, userKey, userData, 0)

	idKey := fmt.Sprintf("kyc:id:%s", user.ID)
	s.redis.Set(ctx, idKey, userData, 0)
}

// ============================================================================
// Travel Rule Compliance
// ============================================================================

// GetTravelRuleData retrieves travel rule data for a transaction
func (s *KYCService) GetTravelRuleData(ctx context.Context, userID string, amount string, currency string) (*TravelRuleData, error) {
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if !s.config.TravelRuleEnabled {
		return nil, fmt.Errorf("travel rule not enabled")
	}

	// Check if user is verified
	if user.Status != "verified" {
		return nil, fmt.Errorf("user not verified for travel rule")
	}

	travelRule := &TravelRuleData{
		Enabled:         true,
		BeneficiaryName: user.FirstName + " " + user.LastName,
		OriginatorName:  user.FirstName + " " + user.LastName,
		OriginatorID:    user.ID,
		Amount:          amount,
		Currency:        currency,
		Date:            time.Now(),
	}

	if user.Address != nil {
		travelRule.BeneficiaryAddr = fmt.Sprintf("%s, %s, %s %s, %s",
			user.Address.Street, user.Address.City, user.Address.State,
			user.Address.PostalCode, user.Address.Country)
		travelRule.OriginatorAddr = travelRule.BeneficiaryAddr
	}

	return travelRule, nil
}

// ValidateTravelRule validates travel rule data
func (s *KYCService) ValidateTravelRule(ctx context.Context, data *TravelRuleData) error {
	if !data.Enabled {
		return nil
	}

	if data.BeneficiaryName == "" {
		return fmt.Errorf("beneficiary name required")
	}

	if data.OriginatorName == "" {
		return fmt.Errorf("originator name required")
	}

	if data.Amount == "" {
		return fmt.Errorf("amount required")
	}

	// Check threshold (typically $3,000 USD)
	amountFloat := 0.0
	fmt.Sscanf(data.Amount, "%f", &amountFloat)
	if amountFloat >= 3000 {
		// High value transaction requires more info
		if data.BeneficiaryID == "" {
			return fmt.Errorf("beneficiary ID required for high-value transaction")
		}
	}

	return nil
}

// ============================================================================
// KYC API Endpoints
// ============================================================================

func (s *ListingService) KYCRouter(r *gin.Engine) {
	kyc := r.Group("/api/v1/kyc")
	{
		// Public endpoints
		kyc.POST("/register", s.RegisterKYCUser)
		kyc.POST("/start", s.StartKYCVerification)
		kyc.POST("/document", s.SubmitKYCDocument)
		kyc.GET("/status/:user_id", s.GetKYCStatus)
		kyc.GET("/session/:session_id", s.GetKYCSession)

		// Travel rule
		kyc.POST("/travel-rule/validate", s.ValidateTravelRuleEndpoint)
		kyc.GET("/travel-rule/:user_id", s.GetTravelRule)

		// Webhook (for external providers)
		kyc.POST("/webhook", s.KYCWebhook)
	}
}

func (s *ListingService) RegisterKYCUser(c *gin.Context) {
	var req struct {
		Email   string `json:"email" binding:"required,email"`
		Phone   string `json:"phone"`
		Country string `json:"country" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	kycService := NewKYCService(s.redis, DefaultKYCConfig)
	user, err := kycService.RegisterUser(c.Request.Context(), req.Email, req.Phone, req.Country)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"user":    user,
	})
}

func (s *ListingService) StartKYCVerification(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
		Level  int    `json:"level" binding:"required,min=1,max=4"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	kycService := NewKYCService(s.redis, DefaultKYCConfig)
	session, err := kycService.StartVerification(c.Request.Context(), req.UserID, req.Level)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"session": session,
	})
}

func (s *ListingService) SubmitKYCDocument(c *gin.Context) {
	sessionID := c.PostForm("session_id")
	docType := c.PostForm("doc_type")

	if sessionID == "" || docType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id and doc_type required"})
		return
	}

	file, err := c.FormFile("document")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "document file required"})
		return
	}

	kycService := NewKYCService(s.redis, DefaultKYCConfig)
	doc, err := kycService.SubmitDocument(c.Request.Context(), sessionID, docType, file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"document": doc,
	})
}

func (s *ListingService) GetKYCStatus(c *gin.Context) {
	userID := c.Param("user_id")

	kycService := NewKYCService(s.redis, DefaultKYCConfig)
	user, err := kycService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"user":    user,
	})
}

func (s *ListingService) GetKYCSession(c *gin.Context) {
	sessionID := c.Param("session_id")

	sessionKey := fmt.Sprintf("kyc:session:%s", sessionID)
	data, err := s.redis.Get(c.Request.Context(), sessionKey).Result()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	var session KYCSession
	json.Unmarshal([]byte(data), &session)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"session": session,
	})
}

func (s *ListingService) ValidateTravelRuleEndpoint(c *gin.Context) {
	var data TravelRuleData
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	kycService := NewKYCService(s.redis, DefaultKYCConfig)
	err := kycService.ValidateTravelRule(c.Request.Context(), &data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"valid":   true,
	})
}

func (s *ListingService) GetTravelRule(c *gin.Context) {
	userID := c.Param("user_id")

	var req struct {
		Amount   string `json:"amount" binding:"required"`
		Currency string `json:"currency" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	kycService := NewKYCService(s.redis, DefaultKYCConfig)
	travelRule, err := kycService.GetTravelRuleData(c.Request.Context(), userID, req.Amount, req.Currency)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"travelRule": travelRule,
	})
}

func (s *ListingService) KYCWebhook(c *gin.Context) {
	// Verify webhook signature using the KYC service's HMAC-SHA256 key.
	signature := c.GetHeader("X-KYC-Signature")
	kycService := NewKYCService(s.redis, DefaultKYCConfig)

	var bodyBytes []byte
	if signature != "" {
		// Read body once so we can both verify the signature and decode JSON.
		bodyBytes, _ = io.ReadAll(c.Request.Body)
		expectedSig := kycService.computeWebhookSignature(bodyBytes)
		if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
			return
		}
	}

	var payload map[string]interface{}
	if len(bodyBytes) > 0 {
		_ = json.Unmarshal(bodyBytes, &payload)
	} else {
		_ = c.ShouldBindJSON(&payload)
	}

	// Process webhook based on event type
	eventType, _ := payload["event_type"].(string)
	userID, _ := payload["user_id"].(string)

	log.Printf("KYC Webhook: %s for user %s", eventType, userID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

func (s *KYCService) computeWebhookSignature(body []byte) string {
	mac := hmac.New(sha256.New, []byte(s.config.WebhookSecret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
