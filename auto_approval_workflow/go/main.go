package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
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
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
)

// ============================================================================
// TIGERWALLET AUTO-APPROVAL WORKFLOW SYSTEM
// Automated KYC and White Label approval with intelligent routing
// ============================================================================

var (
	logger         zerolog.Logger
	redisClient    *redis.Client
	dbPool         *pgxpool.Pool
	workflowEngine *WorkflowEngine
)

// Configuration
type Config struct {
	Port               string
	DatabaseURL        string
	RedisURL           string
	ApprovalQueue      string
	MaxRetries         int
	ProcessingInterval time.Duration
	WebhookURL         string
}

// KYC Application
type KYCApplication struct {
	ID             string     `json:"id"`
	UserID         string     `json:"userId"`
	WhiteLabelID   string     `json:"whiteLabelId"`
	Type           string     `json:"type"`   // identity, address, selfie, document
	Status         string     `json:"status"` // pending, processing, approved, rejected, needs_review
	RiskScore      float64    `json:"riskScore"`
	RiskLevel      string     `json:"riskLevel"` // low, medium, high, critical
	Confidence     float64    `json:"confidence"`
	Documents      []Document `json:"documents"`
	VerifiedAt     *time.Time `json:"verifiedAt,omitempty"`
	RejectedAt     *time.Time `json:"rejectedAt,omitempty"`
	RejectedReason string     `json:"rejectedReason,omitempty"`
	AutoApproved   bool       `json:"autoApproved"`
	ManualReview   bool       `json:"manualReview"`
	ReviewedBy     string     `json:"reviewedBy,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// Document
type Document struct {
	ID         string     `json:"id"`
	Type       string     `json:"type"` // passport, driver_license, national_id, utility_bill
	URL        string     `json:"url"`
	Hash       string     `json:"hash"`
	Verified   bool       `json:"verified"`
	VerifiedAt *time.Time `json:"verifiedAt,omitempty"`
}

// White Label Application
type WhiteLabelApplication struct {
	ID             string      `json:"id"`
	UserID         string      `json:"userId"`
	CompanyName    string      `json:"companyName"`
	Domain         string      `json:"domain"`
	BusinessType   string      `json:"businessType"`
	Status         string      `json:"status"` // pending, processing, approved, rejected, needs_review
	RiskScore      float64     `json:"riskScore"`
	RiskLevel      string      `json:"riskLevel"`
	Documents      []Document  `json:"documents"`
	ContactInfo    ContactInfo `json:"contactInfo"`
	KYCStatus      string      `json:"kycStatus"`
	AutoApproved   bool        `json:"autoApproved"`
	ApprovedAt     *time.Time  `json:"approvedAt,omitempty"`
	RejectedAt     *time.Time  `json:"rejectedAt,omitempty"`
	RejectedReason string      `json:"rejectedReason,omitempty"`
	CreatedAt      time.Time   `json:"createdAt"`
	UpdatedAt      time.Time   `json:"updatedAt"`
}

// Contact Information
type ContactInfo struct {
	Email   string `json:"email"`
	Phone   string `json:"phone"`
	Address string `json:"address"`
	City    string `json:"city"`
	Country string `json:"country"`
	Website string `json:"website"`
}

// Workflow
type Workflow struct {
	ID                   string         `json:"id"`
	Name                 string         `json:"name"`
	Type                 string         `json:"type"`   // kyc, white_label, user_approval
	Status               string         `json:"status"` // active, paused, archived
	Steps                []WorkflowStep `json:"steps"`
	Conditions           []Condition    `json:"conditions"`
	AutoApprove          bool           `json:"autoApprove"`
	AutoApproveThreshold float64        `json:"autoApproveThreshold"` // Risk score threshold
	CreatedAt            time.Time      `json:"createdAt"`
	UpdatedAt            time.Time      `json:"updatedAt"`
}

// Workflow Step
type WorkflowStep struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"` // verification, review, approval, notification, webhook
	Config     map[string]interface{} `json:"config"`
	Timeout    int                    `json:"timeout"` // seconds
	RetryCount int                    `json:"retryCount"`
	OnFailure  string                 `json:"onFailure"` // skip, retry, reject
	Order      int                    `json:"order"`
}

// Condition
type Condition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // eq, neq, gt, lt, gte, lte, in, not_in
	Value    interface{} `json:"value"`
	Action   string      `json:"action"` // approve, reject, review, skip
}

// Workflow Execution
type WorkflowExecution struct {
	ID          string                 `json:"id"`
	WorkflowID  string                 `json:"workflowId"`
	EntityID    string                 `json:"entityId"` // KYC or White Label application ID
	EntityType  string                 `json:"entityType"`
	Status      string                 `json:"status"` // pending, running, completed, failed
	CurrentStep int                    `json:"currentStep"`
	Results     map[string]interface{} `json:"results"`
	Error       string                 `json:"error,omitempty"`
	StartedAt   time.Time              `json:"startedAt"`
	CompletedAt *time.Time             `json:"completedAt,omitempty"`
}

// Approval Rule
type ApprovalRule struct {
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	Type          string      `json:"type"` // kyc, white_label
	Priority      int         `json:"priority"`
	Conditions    []Condition `json:"conditions"`
	Action        string      `json:"action"` // auto_approve, auto_reject, manual_review
	RiskThreshold float64     `json:"riskThreshold"`
	IsActive      bool        `json:"isActive"`
	CreatedAt     time.Time   `json:"createdAt"`
	UpdatedAt     time.Time   `json:"updatedAt"`
}

// Webhook Event
type WebhookEvent struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"` // kyc.approved, kyc.rejected, wl.approved, wl.rejected
	EntityID  string      `json:"entityId"`
	Data      interface{} `json:"data"`
	Retries   int         `json:"retries"`
	Status    string      `json:"status"` // pending, sent, failed
	CreatedAt time.Time   `json:"createdAt"`
	SentAt    *time.Time  `json:"sentAt,omitempty"`
}

// Workflow Engine
type WorkflowEngine struct {
	dbPool    *pgxpool.Pool
	redis     *redis.Client
	workflows map[string]*Workflow
	mu        sync.RWMutex
	running   bool
	stopChan  chan struct{}
}

func NewWorkflowEngine(db *pgxpool.Pool, redis *redis.Client) *WorkflowEngine {
	return &WorkflowEngine{
		dbPool:    db,
		redis:     redis,
		workflows: make(map[string]*Workflow),
		stopChan:  make(chan struct{}),
	}
}

func (e *WorkflowEngine) Start() {
	e.running = true
	go e.processQueue()
	go e.cleanupOldExecutions()
	logger.Info().Msg("Workflow engine started")
}

func (e *WorkflowEngine) Stop() {
	e.running = false
	close(e.stopChan)
	logger.Info().Msg("Workflow engine stopped")
}

func (e *WorkflowEngine) processQueue() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-e.stopChan:
			return
		case <-ticker.C:
			e.processNextItem()
		}
	}
}

func (e *WorkflowEngine) processNextItem() {
	ctx := context.Background()

	// Get next item from queue
	result, err := e.redis.LPop(ctx, "approval_queue").Result()
	if err != nil {
		return // Queue is empty
	}

	var item map[string]interface{}
	if err := json.Unmarshal([]byte(result), &item); err != nil {
		logger.Error().Err(err).Msg("Failed to parse queue item")
		return
	}

	entityType := item["entityType"].(string)
	entityID := item["entityID"].(string)

	// Get appropriate workflow
	workflow := e.getWorkflow(entityType)
	if workflow == nil {
		logger.Warn().Str("entityType", entityType).Msg("No workflow found")
		return
	}

	// Execute workflow
	execution := e.executeWorkflow(workflow, entityType, entityID)

	// Update entity status based on execution result
	if execution.Status == "completed" {
		e.updateEntityStatus(entityType, entityID, "approved")
	} else if execution.Status == "failed" {
		e.updateEntityStatus(entityType, entityID, "needs_review")
	}
}

func (e *WorkflowEngine) getWorkflow(entityType string) *Workflow {
	e.mu.RLock()
	defer e.mu.RUnlock()

	workflow, ok := e.workflows[entityType]
	if !ok {
		// Try to load from database
		workflow = e.loadWorkflow(entityType)
		if workflow != nil {
			e.workflows[entityType] = workflow
		}
	}
	return workflow
}

func (e *WorkflowEngine) loadWorkflow(entityType string) *Workflow {
	var workflow Workflow
	var stepsJSON, conditionsJSON []byte

	err := dbPool.QueryRow(context.Background(), `
		SELECT id, name, type, status, steps, conditions, auto_approve, auto_approve_threshold, created_at, updated_at
		FROM approval_workflows WHERE type = $1 AND status = 'active'
	`, entityType).Scan(
		&workflow.ID, &workflow.Name, &workflow.Type, &workflow.Status,
		&stepsJSON, &conditionsJSON, &workflow.AutoApprove, &workflow.AutoApproveThreshold,
		&workflow.CreatedAt, &workflow.UpdatedAt,
	)

	if err != nil {
		// Create default workflow
		return createDefaultWorkflow(entityType)
	}

	json.Unmarshal(stepsJSON, &workflow.Steps)
	json.Unmarshal(conditionsJSON, &workflow.Conditions)

	return &workflow
}

func createDefaultWorkflow(entityType string) *Workflow {
	steps := []WorkflowStep{
		{
			ID:         "verify_documents",
			Name:       "Verify Documents",
			Type:       "verification",
			Config:     map[string]interface{}{"provider": "internal"},
			Timeout:    30,
			RetryCount: 3,
			OnFailure:  "review",
			Order:      1,
		},
		{
			ID:         "risk_assessment",
			Name:       "Risk Assessment",
			Type:       "verification",
			Config:     map[string]interface{}{"model": "risk_v1"},
			Timeout:    15,
			RetryCount: 2,
			OnFailure:  "review",
			Order:      2,
		},
		{
			ID:         "decision",
			Name:       "Decision Engine",
			Type:       "approval",
			Config:     map[string]interface{}{},
			Timeout:    5,
			RetryCount: 1,
			OnFailure:  "reject",
			Order:      3,
		},
	}

	conditions := []Condition{
		{
			Field:    "riskScore",
			Operator: "lte",
			Value:    0.2,
			Action:   "approve",
		},
		{
			Field:    "riskScore",
			Operator: "gte",
			Value:    0.8,
			Action:   "reject",
		},
		{
			Field:    "riskScore",
			Operator: "gt",
			Value:    0.2,
			Action:   "review",
		},
	}

	return &Workflow{
		ID:                   entityType + "_default",
		Name:                 "Default " + entityType + " Workflow",
		Type:                 entityType,
		Status:               "active",
		Steps:                steps,
		Conditions:           conditions,
		AutoApprove:          true,
		AutoApproveThreshold: 0.3,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}
}

func (e *WorkflowEngine) executeWorkflow(workflow *Workflow, entityType, entityID string) *WorkflowExecution {
	execution := &WorkflowExecution{
		ID:          generateUUID(),
		WorkflowID:  workflow.ID,
		EntityID:    entityID,
		EntityType:  entityType,
		Status:      "running",
		CurrentStep: 0,
		Results:     make(map[string]interface{}),
		StartedAt:   time.Now(),
	}

	for i, step := range workflow.Steps {
		execution.CurrentStep = i

		// Execute step
		result := e.executeStep(step, entityType, entityID)
		execution.Results[step.ID] = result

		// Check if step failed
		if result["status"] == "failed" {
			if step.OnFailure == "reject" {
				execution.Status = "failed"
				execution.Error = fmt.Sprintf("Step %s failed", step.Name)
				break
			} else if step.OnFailure == "skip" {
				continue
			}
		}

		// Check conditions after verification step
		if step.Type == "approval" || step.Type == "verification" {
			action := e.evaluateConditions(workflow.Conditions, execution.Results)
			if action == "approve" {
				execution.Status = "completed"
				now := time.Now()
				execution.CompletedAt = &now
				break
			} else if action == "reject" {
				execution.Status = "failed"
				execution.Error = "Rejected by conditions"
				break
			}
		}
	}

	if execution.Status == "running" {
		execution.Status = "completed"
		now := time.Now()
		execution.CompletedAt = &now
	}

	// Save execution
	e.saveExecution(execution)

	return execution
}

func (e *WorkflowEngine) executeStep(step WorkflowStep, entityType, entityID string) map[string]interface{} {
	result := map[string]interface{}{
		"step":   step.Name,
		"status": "success",
	}

	switch step.Type {
	case "verification":
		result = e.runVerification(entityType, entityID, step)
	case "review":
		result = e.runReviewCheck(entityType, entityID, step)
	case "approval":
		result = e.runApprovalCheck(entityType, entityID, step)
	case "webhook":
		result = e.runWebhook(entityType, entityID, step)
	case "notification":
		result = e.runNotification(entityType, entityID, step)
	}

	return result
}

func (e *WorkflowEngine) runVerification(entityType, entityID string, step WorkflowStep) map[string]interface{} {
	// Run document verification
	result := map[string]interface{}{
		"step":   step.Name,
		"status": "success",
		"data": map[string]interface{}{
			"verified":   true,
			"confidence": 0.95,
		},
	}

	// Simulate verification (in production, call external provider)
	if entityType == "kyc" {
		var kyc KYCApplication
		err := dbPool.QueryRow(context.Background(), `
			SELECT id, user_id, white_label_id, type, status, risk_score, risk_level 
			FROM kyc_applications WHERE id = $1
		`, entityID).Scan(&kyc.ID, &kyc.UserID, &kyc.WhiteLabelID, &kyc.Type, &kyc.Status, &kyc.RiskScore, &kyc.RiskLevel)

		if err != nil {
			result["status"] = "failed"
			result["error"] = err.Error()
		}
	}

	return result
}

func (e *WorkflowEngine) runReviewCheck(entityType, entityID string, step WorkflowStep) map[string]interface{} {
	result := map[string]interface{}{
		"step":   step.Name,
		"status": "success",
		"data": map[string]interface{}{
			"needsReview": false,
		},
	}
	return result
}

func (e *WorkflowEngine) runApprovalCheck(entityType, entityID string, step WorkflowStep) map[string]interface{} {
	result := map[string]interface{}{
		"step":   step.Name,
		"status": "success",
		"data": map[string]interface{}{
			"approved": true,
		},
	}
	return result
}

func (e *WorkflowEngine) runWebhook(entityType, entityID string, step WorkflowStep) map[string]interface{} {
	result := map[string]interface{}{
		"step":   step.Name,
		"status": "success",
	}
	return result
}

func (e *WorkflowEngine) runNotification(entityType, entityID string, step WorkflowStep) map[string]interface{} {
	result := map[string]interface{}{
		"step":   step.Name,
		"status": "success",
	}
	return result
}

func (e *WorkflowEngine) evaluateConditions(conditions []Condition, results map[string]interface{}) string {
	for _, condition := range conditions {
		fieldValue := results[condition.Field]
		if fieldValue == nil {
			continue
		}

		result := evaluateCondition(condition, fieldValue)
		if result {
			return condition.Action
		}
	}
	return "review"
}

func evaluateCondition(condition Condition, value interface{}) bool {
	switch condition.Operator {
	case "eq":
		return value == condition.Value
	case "neq":
		return value != condition.Value
	case "gt":
		return toFloat(value) > toFloat(condition.Value)
	case "lt":
		return toFloat(value) < toFloat(condition.Value)
	case "gte":
		return toFloat(value) >= toFloat(condition.Value)
	case "lte":
		return toFloat(value) <= toFloat(condition.Value)
	}
	return false
}

func toFloat(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	default:
		return 0
	}
}

func (e *WorkflowEngine) updateEntityStatus(entityType, entityID, status string) {
	if entityType == "kyc" {
		dbPool.Exec(context.Background(), `
			UPDATE kyc_applications SET status = $1, updated_at = NOW() WHERE id = $2
		`, status, entityID)
	} else if entityType == "white_label" {
		dbPool.Exec(context.Background(), `
			UPDATE white_label_applications SET status = $1, updated_at = NOW() WHERE id = $2
		`, status, entityID)
	}
}

func (e *WorkflowEngine) saveExecution(execution *WorkflowExecution) {
	resultsJSON, _ := json.Marshal(execution.Results)

	dbPool.Exec(context.Background(), `
		INSERT INTO workflow_executions (id, workflow_id, entity_id, entity_type, status, current_step, results, error, started_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			current_step = EXCLUDED.current_step,
			results = EXCLUDED.results,
			error = EXCLUDED.error,
			completed_at = EXCLUDED.completed_at
	`, execution.ID, execution.WorkflowID, execution.EntityID, execution.EntityType, execution.Status,
		execution.CurrentStep, resultsJSON, execution.Error, execution.StartedAt, execution.CompletedAt)
}

func (e *WorkflowEngine) cleanupOldExecutions() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-e.stopChan:
			return
		case <-ticker.C:
			// Clean up executions older than 30 days
			dbPool.Exec(context.Background(), `
				DELETE FROM workflow_executions 
				WHERE completed_at < NOW() - INTERVAL '30 days'
			`)
		}
	}
}

// API Handlers

// Submit KYC application
func SubmitKYCApplication(c *gin.Context) {
	var application KYCApplication
	if err := c.ShouldBindJSON(&application); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	application.ID = generateUUID()
	application.Status = "pending"
	application.CreatedAt = time.Now()
	application.UpdatedAt = time.Now()

	_, err := dbPool.Exec(context.Background(), `
		INSERT INTO kyc_applications (id, user_id, white_label_id, type, status, risk_score, risk_level, documents, auto_approved, manual_review, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, application.ID, application.UserID, application.WhiteLabelID, application.Type, application.Status,
		application.RiskScore, application.RiskLevel, "[]", application.AutoApproved, application.ManualReview,
		application.CreatedAt, application.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Add to approval queue
	item := map[string]interface{}{
		"entityType": "kyc",
		"entityID":   application.ID,
	}
	itemJSON, _ := json.Marshal(item)
	redisClient.RPush(context.Background(), "approval_queue", string(itemJSON))

	c.JSON(http.StatusCreated, gin.H{"application": application})
}

// Get KYC application status
func GetKYCApplication(c *gin.Context) {
	id := c.Param("id")

	var application KYCApplication
	err := dbPool.QueryRow(context.Background(), `
		SELECT id, user_id, white_label_id, type, status, risk_score, risk_level, auto_approved, manual_review, reviewed_by, created_at, updated_at
		FROM kyc_applications WHERE id = $1
	`, id).Scan(
		&application.ID, &application.UserID, &application.WhiteLabelID, &application.Type, &application.Status,
		&application.RiskScore, &application.RiskLevel, &application.AutoApproved, &application.ManualReview,
		&application.ReviewedBy, &application.CreatedAt, &application.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Application not found"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"application": application})
}

// Submit White Label application
func SubmitWhiteLabelApplication(c *gin.Context) {
	var application WhiteLabelApplication
	if err := c.ShouldBindJSON(&application); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	application.ID = generateUUID()
	application.Status = "pending"
	application.KYCStatus = "pending"
	application.CreatedAt = time.Now()
	application.UpdatedAt = time.Now()

	_, err := dbPool.Exec(context.Background(), `
		INSERT INTO white_label_applications (id, user_id, company_name, domain, business_type, status, risk_score, risk_level, kyc_status, auto_approved, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, application.ID, application.UserID, application.CompanyName, application.Domain, application.BusinessType,
		application.Status, application.RiskScore, application.RiskLevel, application.KYCStatus, application.AutoApproved,
		application.CreatedAt, application.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Add to approval queue
	item := map[string]interface{}{
		"entityType": "white_label",
		"entityID":   application.ID,
	}
	itemJSON, _ := json.Marshal(item)
	redisClient.RPush(context.Background(), "approval_queue", string(itemJSON))

	c.JSON(http.StatusCreated, gin.H{"application": application})
}

// Get workflow status
func GetWorkflowStatus(c *gin.Context) {
	entityType := c.Query("entityType")
	entityID := c.Query("entityID")

	var execution WorkflowExecution
	err := dbPool.QueryRow(context.Background(), `
		SELECT id, workflow_id, entity_id, entity_type, status, current_step, error, started_at, completed_at
		FROM workflow_executions 
		WHERE entity_type = $1 AND entity_id = $2
		ORDER BY started_at DESC LIMIT 1
	`, entityType, entityID).Scan(
		&execution.ID, &execution.WorkflowID, &execution.EntityID, &execution.EntityType,
		&execution.Status, &execution.CurrentStep, &execution.Error, &execution.StartedAt, &execution.CompletedAt,
	)

	if err == pgx.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "No workflow execution found"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"execution": execution})
}

// Create approval rule
func CreateApprovalRule(c *gin.Context) {
	var rule ApprovalRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule.ID = generateUUID()
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	conditionsJSON, _ := json.Marshal(rule.Conditions)

	_, err := dbPool.Exec(context.Background(), `
		INSERT INTO approval_rules (id, name, type, priority, conditions, action, risk_threshold, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, rule.ID, rule.Name, rule.Type, rule.Priority, conditionsJSON, rule.Action, rule.RiskThreshold,
		rule.IsActive, rule.CreatedAt, rule.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"rule": rule})
}

// Get approval rules
func GetApprovalRules(c *gin.Context) {
	ruleType := c.Query("type")

	rows, err := dbPool.Query(context.Background(), `
		SELECT id, name, type, priority, conditions, action, risk_threshold, is_active, created_at, updated_at
		FROM approval_rules WHERE type = $1 ORDER BY priority DESC
	`, ruleType)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var rules []ApprovalRule
	for rows.Next() {
		var rule ApprovalRule
		var conditionsJSON []byte
		rows.Scan(&rule.ID, &rule.Name, &rule.Type, &rule.Priority, &conditionsJSON,
			&rule.Action, &rule.RiskThreshold, &rule.IsActive, &rule.CreatedAt, &rule.UpdatedAt)
		json.Unmarshal(conditionsJSON, &rule.Conditions)
		rules = append(rules, rule)
	}

	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

// Manual review override
func ManualReviewOverride(c *gin.Context) {
	var request struct {
		EntityType string `json:"entityType" binding:"required"`
		EntityID   string `json:"entityID" binding:"required"`
		Action     string `json:"action" binding:"required"` // approve, reject
		Reason     string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("userID")

	// Update entity status
	if request.EntityType == "kyc" {
		status := "approved"
		if request.Action == "reject" {
			status = "rejected"
		}

		_, err := dbPool.Exec(context.Background(), `
			UPDATE kyc_applications SET status = $1, reviewed_by = $2, rejected_reason = $3, updated_at = NOW() 
			WHERE id = $4
		`, status, userID, request.Reason, request.EntityID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else if request.EntityType == "white_label" {
		status := "approved"
		if request.Action == "reject" {
			status = "rejected"
		}

		_, err := dbPool.Exec(context.Background(), `
			UPDATE white_label_applications SET status = $1, rejected_reason = $2, updated_at = NOW() 
			WHERE id = $3
		`, status, request.Reason, request.EntityID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Review completed"})
}

// Get queue status
func GetQueueStatus(c *gin.Context) {
	ctx := context.Background()

	queueLength, _ := redisClient.LLen(ctx, "approval_queue").Result()
	processingCount, _ := redisClient.Get(ctx, "approval_processing").Int()

	c.JSON(http.StatusOK, gin.H{
		"queueLength": queueLength,
		"processing":  processingCount,
	})
}

// Helper functions
func generateUUID() string {
	b := make([]byte, 16)
	for i := 0; i < 16; i++ {
		b[i] = byte(i * 17 % 256)
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// Database schema
func createSchema() error {
	schema := `
	-- KYC Applications
	CREATE TABLE IF NOT EXISTS kyc_applications (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL,
		white_label_id UUID,
		type VARCHAR(50) NOT NULL,
		status VARCHAR(20) DEFAULT 'pending',
		risk_score DECIMAL(5,4) DEFAULT 0,
		risk_level VARCHAR(20) DEFAULT 'low',
		confidence DECIMAL(5,4) DEFAULT 0,
		documents JSONB DEFAULT '[]',
		verified_at TIMESTAMP,
		rejected_at TIMESTAMP,
		rejected_reason TEXT,
		auto_approved BOOLEAN DEFAULT false,
		manual_review BOOLEAN DEFAULT false,
		reviewed_by UUID,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);

	-- White Label Applications
	CREATE TABLE IF NOT EXISTS white_label_applications (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL,
		company_name VARCHAR(255) NOT NULL,
		domain VARCHAR(255) UNIQUE NOT NULL,
		business_type VARCHAR(50),
		status VARCHAR(20) DEFAULT 'pending',
		risk_score DECIMAL(5,4) DEFAULT 0,
		risk_level VARCHAR(20) DEFAULT 'low',
		documents JSONB DEFAULT '[]',
		contact_info JSONB,
		kyc_status VARCHAR(20) DEFAULT 'pending',
		auto_approved BOOLEAN DEFAULT false,
		approved_at TIMESTAMP,
		rejected_at TIMESTAMP,
		rejected_reason TEXT,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);

	-- Approval Workflows
	CREATE TABLE IF NOT EXISTS approval_workflows (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(255) NOT NULL,
		type VARCHAR(50) NOT NULL UNIQUE,
		status VARCHAR(20) DEFAULT 'active',
		steps JSONB NOT NULL,
		conditions JSONB DEFAULT '[]',
		auto_approve BOOLEAN DEFAULT true,
		auto_approve_threshold DECIMAL(5,4) DEFAULT 0.3,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);

	-- Approval Rules
	CREATE TABLE IF NOT EXISTS approval_rules (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(255) NOT NULL,
		type VARCHAR(50) NOT NULL,
		priority INTEGER DEFAULT 0,
		conditions JSONB NOT NULL,
		action VARCHAR(20) NOT NULL,
		risk_threshold DECIMAL(5,4) DEFAULT 0.5,
		is_active BOOLEAN DEFAULT true,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);

	-- Workflow Executions
	CREATE TABLE IF NOT EXISTS workflow_executions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		workflow_id VARCHAR(255) NOT NULL,
		entity_id UUID NOT NULL,
		entity_type VARCHAR(50) NOT NULL,
		status VARCHAR(20) DEFAULT 'pending',
		current_step INTEGER DEFAULT 0,
		results JSONB DEFAULT '{}',
		error TEXT,
		started_at TIMESTAMP DEFAULT NOW(),
		completed_at TIMESTAMP,
		UNIQUE(entity_id, entity_type, started_at)
	);

	-- Webhook Events
	CREATE TABLE IF NOT EXISTS webhook_events (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		type VARCHAR(50) NOT NULL,
		entity_id UUID NOT NULL,
		data JSONB,
		retries INTEGER DEFAULT 0,
		status VARCHAR(20) DEFAULT 'pending',
		created_at TIMESTAMP DEFAULT NOW(),
		sent_at TIMESTAMP
	);

	-- Indexes
	CREATE INDEX IF NOT EXISTS idx_kyc_user ON kyc_applications(user_id);
	CREATE INDEX IF NOT EXISTS idx_kyc_status ON kyc_applications(status);
	CREATE INDEX IF NOT EXISTS idx_wl_status ON white_label_applications(status);
	CREATE INDEX IF NOT EXISTS idx_executions_entity ON workflow_executions(entity_id, entity_type);
	`

	_, err := dbPool.Exec(context.Background(), schema)
	return err
}

// Router setup
func setupRouter() *gin.Engine {
	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "timestamp": time.Now().Unix()})
	})

	v1 := r.Group("/api/v1")
	{
		// KYC endpoints
		v1.POST("/kyc/applications", SubmitKYCApplication)
		v1.GET("/kyc/applications/:id", GetKYCApplication)

		// White Label endpoints
		v1.POST("/white-label/applications", SubmitWhiteLabelApplication)

		// Workflow endpoints
		v1.GET("/workflow/status", GetWorkflowStatus)
		v1.POST("/workflow/override", ManualReviewOverride)

		// Rules endpoints
		v1.POST("/approval/rules", CreateApprovalRule)
		v1.GET("/approval/rules", GetApprovalRules)

		// Queue endpoints
		v1.GET("/queue/status", GetQueueStatus)
	}

	return r
}

func main() {
	initLogger()
	logger.Info().Msg("Starting TigerWallet Auto-Approval Workflow System")

	config := Config{
		Port:        getEnv("PORT", "8087"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://tigerwallet:tigerpass@localhost:5432/tigerwallet?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "localhost:6379"),
	}

	// Initialize database
	var err error
	dbPool, err = pgxpool.Connect(context.Background(), config.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	// Create schema
	if err := createSchema(); err != nil {
		logger.Warn().Err(err).Msg("Schema creation warning")
	}

	// Initialize Redis
	redisClient = redis.NewClient(&redis.Options{
		Addr:     config.RedisURL,
		Password: "",
		DB:       0,
	})
	defer redisClient.Close()

	// Initialize workflow engine
	workflowEngine = NewWorkflowEngine(dbPool, redisClient)
	workflowEngine.Start()
	defer workflowEngine.Stop()

	// Setup router
	router := setupRouter()

	// Start server
	srv := &http.Server{
		Addr:         ":" + config.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		logger.Info().Str("port", config.Port).Msg("Server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
