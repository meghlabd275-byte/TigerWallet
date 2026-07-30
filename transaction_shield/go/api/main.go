/**
 * TigerWallet Transaction Shield - Go API Service
 * Enterprise-grade fraud protection API
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type Config struct {
	ServerPort string
}

var cfg = Config{ServerPort: ":8447"}

// ============================================================================
// Data Models
// ============================================================================

type TransactionRequest struct {
	RequestID       uint64    `json:"request_id"`
	UserID         uint32    `json:"user_id"`
	FromAddress    string    `json:"from_address"`
	ToAddress      string    `json:"to_address"`
	TokenAddress   string    `json:"token_address"`
	ChainID        uint64    `json:"chain_id"`
	Amount         uint64    `json:"amount"`
	Timestamp      uint64    `json:"timestamp"`
	TxType         string    `json:"tx_type"`
	DappOrigin     string    `json:"dapp_origin"`
	IPAddress      string    `json:"ip_address"`
	DeviceFingerprint string `json:"device_fingerprint"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
}

type RiskAssessment struct {
	RequestID         uint64   `json:"request_id"`
	RiskLevel        string   `json:"risk_level"` // NONE, LOW, MEDIUM, HIGH, CRITICAL, BLOCKED
	RiskScore        uint64   `json:"risk_score"` // 0-10000
	TriggeredRules    []string `json:"triggered_rules"`
	Recommendation   string   `json:"recommendation"` // APPROVE, WARN, REVIEW, BLOCK
	RequiresReview   bool     `json:"requires_review"`
	IsApproved       bool     `json:"is_approved"`
	AssessedAt       uint64   `json:"assessed_at"`
}

type UserShield struct {
	UserID                uint32   `json:"user_id"`
	Status               string   `json:"status"` // INACTIVE, ACTIVE, SUSPENDED, CLAIMED, EXPIRED
	ProtectionLimit      uint64   `json:"protection_limit"`
	CurrentCoveredAmount uint64   `json:"current_covered_amount"`
	TotalClaims          uint64   `json:"total_claims"`
	TotalClaimedAmount   uint64   `json:"total_claimed_amount"`
	ActivatedAt          uint64   `json:"activated_at"`
	ExpiresAt           uint64   `json:"expires_at"`
	AutoProtect         bool     `json:"auto_protect"`
	ProtectionLevel     uint8    `json:"protection_level"` // 1-5
}

type ShieldRule struct {
	RuleID          uint64            `json:"rule_id"`
	RuleType        string            `json:"rule_type"` // AMOUNT_LIMIT, VELOCITY_LIMIT, RECIPIENT_BLACKLIST, etc.
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	IsActive        bool              `json:"is_active"`
	Priority        uint8             `json:"priority"`
	ActionOnTrigger string            `json:"action_on_trigger"` // NONE, LOW, MEDIUM, HIGH, BLOCKED
	Parameters      map[string]string `json:"parameters"`
	CreatedAt       uint64            `json:"created_at"`
	UpdatedAt       uint64            `json:"updated_at"`
}

type Claim struct {
	ClaimID        uint64   `json:"claim_id"`
	UserID        uint32   `json:"user_id"`
	RequestID     uint64   `json:"request_id"`
	TxHash        uint64   `json:"tx_hash"`
	Amount        uint64   `json:"amount"`
	Description   string   `json:"description"`
	Status        string   `json:"status"` // pending, approved, rejected, paid
	FiledAt       uint64   `json:"filed_at"`
	ResolvedAt    uint64   `json:"resolved_at"`
	PaidAt        uint64   `json:"paid_at"`
	ResolvedBy    uint32   `json:"resolved_by"`
}

type ShieldStats struct {
	TotalTransactionsScanned uint64 `json:"total_transactions_scanned"`
	TransactionsApproved     uint64 `json:"transactions_approved"`
	TransactionsBlocked     uint64 `json:"transactions_blocked"`
	TransactionsFlagged     uint64 `json:"transactions_flagged"`
	TotalProtectionClaims  uint64 `json:"total_protection_claims"`
	TotalProtectionPaid   uint64 `json:"total_protection_paid"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ============================================================================
// Shield Service
// ============================================================================

type ShieldService struct {
	userShields   map[uint32]*UserShield
	rules        map[uint64]*ShieldRule
	claims       map[uint64]*Claim
	assessments  map[uint64]*RiskAssessment
	scamAddresses map[string]bool
	nextRuleID   uint64
	nextClaimID  uint64
	nextRequestID uint64
	mu           sync.RWMutex
}

func NewShieldService() *ShieldService {
	svc := &ShieldService{
		userShields:   make(map[uint32]*UserShield),
		rules:        make(map[uint64]*ShieldRule),
		claims:       make(map[uint64]*Claim),
		assessments:  make(map[uint64]*RiskAssessment),
		scamAddresses: make(map[string]bool),
		nextRuleID:   1,
		nextClaimID:  1,
		nextRequestID: 1,
	}

	// Initialize default rules
	svc.initDefaultRules()

	return svc
}

func (s *ShieldService) initDefaultRules() {
	now := uint64(time.Now().UnixMilli())

	// High amount rule
	s.rules[s.nextRuleID] = &ShieldRule{
		RuleID:          s.nextRuleID,
		RuleType:        "AMOUNT_LIMIT",
		Name:            "High Amount Alert",
		Description:     "Flag transactions over $10,000",
		IsActive:        true,
		Priority:        100,
		ActionOnTrigger: "HIGH",
		Parameters:      map[string]string{"max_amount": "1000000000"},
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	s.nextRuleID++

	// Velocity rule
	s.rules[s.nextRuleID] = &ShieldRule{
		RuleID:          s.nextRuleID,
		RuleType:        "VELOCITY_LIMIT",
		Name:            "High Velocity Alert",
		Description:     "Flag transactions within 10 seconds of each other",
		IsActive:        true,
		Priority:        90,
		ActionOnTrigger: "MEDIUM",
		Parameters:      map[string]string{"min_seconds": "10"},
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	s.nextRuleID++

	// Scam address rule
	s.rules[s.nextRuleID] = &ShieldRule{
		RuleID:          s.nextRuleID,
		RuleType:        "RECIPIENT_BLACKLIST",
		Name:            "Known Scam Address",
		Description:     "Block transactions to known scam addresses",
		IsActive:        true,
		Priority:        200,
		ActionOnTrigger: "CRITICAL",
		Parameters:      map[string]string{},
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	s.nextRuleID++

	// Initialize known scam addresses (demo)
	s.scamAddresses["0xSCAM0000000000000000000000000000000000001"] = true
	s.scamAddresses["0xSCAM0000000000000000000000000000000000002"] = true

	log.Printf("Initialized %d default rules", len(s.rules))
}

func (s *ShieldService) ActivateShield(userID uint32, protectionLimit uint64, protectionLevel uint8, durationDays uint64) *UserShield {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := uint64(time.Now().UnixMilli())

	shield := &UserShield{
		UserID:                userID,
		Status:               "ACTIVE",
		ProtectionLimit:      protectionLimit,
		CurrentCoveredAmount: 0,
		TotalClaims:          0,
		TotalClaimedAmount:   0,
		ActivatedAt:          now,
		ExpiresAt:           now + durationDays*24*60*60*1000,
		AutoProtect:         true,
		ProtectionLevel:     protectionLevel,
	}

	s.userShields[userID] = shield
	return shield
}

func (s *ShieldService) GetShield(userID uint32) *UserShield {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.userShields[userID]
}

func (s *ShieldService) AnalyzeTransaction(req *TransactionRequest) *RiskAssessment {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := uint64(time.Now().UnixMilli())

	req.RequestID = s.nextRequestID
	s.nextRequestID++

	assessment := &RiskAssessment{
		RequestID:     req.RequestID,
		AssessedAt:    now,
		TriggeredRules: []string{},
	}

	// Get user shield
	shield := s.userShields[req.UserID]

	// Check against rules
	for _, rule := range s.rules {
		if !rule.IsActive {
			continue
		}

		triggered := false

		switch rule.RuleType {
		case "AMOUNT_LIMIT":
			maxAmt := uint64(1000000000) // $10,000 default
			if v, ok := rule.Parameters["max_amount"]; ok {
				fmt.Sscanf(v, "%d", &maxAmt)
			}
			if req.Amount > maxAmt {
				triggered = true
			}

		case "RECIPIENT_BLACKLIST":
			if s.scamAddresses[req.ToAddress] {
				triggered = true
			}
		}

		if triggered {
			assessment.TriggeredRules = append(assessment.TriggeredRules, rule.Name)
			if rule.ActionOnTrigger == "CRITICAL" || rule.ActionOnTrigger == "BLOCKED" {
				assessment.RiskLevel = "CRITICAL"
				assessment.RiskScore = 9000
			} else if rule.ActionOnTrigger == "HIGH" {
				if assessment.RiskScore < 7000 {
					assessment.RiskLevel = "HIGH"
					assessment.RiskScore = 7000
				}
			}
		}
	}

	// Set default if no rules triggered
	if assessment.RiskLevel == "" {
		if req.Amount > 100000000 { // > $1,000
			assessment.RiskLevel = "MEDIUM"
			assessment.RiskScore = 4000
		} else if req.Amount > 10000000 { // > $100
			assessment.RiskLevel = "LOW"
			assessment.RiskScore = 2000
		} else {
			assessment.RiskLevel = "NONE"
			assessment.RiskScore = 0
		}
	}

	// Set recommendation
	switch assessment.RiskLevel {
	case "CRITICAL", "BLOCKED":
		assessment.Recommendation = "BLOCK"
		assessment.IsApproved = false
	case "HIGH":
		assessment.Recommendation = "REVIEW"
		assessment.RequiresReview = true
		assessment.IsApproved = false
	case "MEDIUM":
		assessment.Recommendation = "WARN"
		assessment.IsApproved = true
	default:
		assessment.Recommendation = "APPROVE"
		assessment.IsApproved = true
	}

	// Store assessment
	s.assessments[req.RequestID] = assessment

	return assessment
}

func (s *ShieldService) GetAssessment(requestID uint64) *RiskAssessment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.assessments[requestID]
}

func (s *ShieldService) FileClaim(userID, requestID uint64, txHash, amount uint64, description string) *Claim {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := uint64(time.Now().UnixMilli())

	claim := &Claim{
		ClaimID:      s.nextClaimID,
		UserID:       uint32(userID),
		RequestID:    requestID,
		TxHash:       txHash,
		Amount:       amount,
		Description:  description,
		Status:       "pending",
		FiledAt:      now,
	}

	s.claims[s.nextClaimID] = claim
	s.nextClaimID++

	return claim
}

func (s *ShieldService) ProcessClaim(claimID uint64, reviewerID uint32, approved bool, notes string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	claim, ok := s.claims[claimID]
	if !ok {
		return false
	}

	now := uint64(time.Now().UnixMilli())

	if approved {
		claim.Status = "approved"
	} else {
		claim.Status = "rejected"
	}

	claim.ResolvedAt = now
	claim.ResolvedBy = reviewerID

	return true
}

func (s *ShieldService) GetStats() *ShieldStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var approved, blocked, flagged uint64

	for _, a := range s.assessments {
		if !a.IsApproved {
			blocked++
		} else if a.RequiresReview {
			flagged++
		} else {
			approved++
		}
	}

	return &ShieldStats{
		TotalTransactionsScanned: uint64(len(s.assessments)),
		TransactionsApproved:     approved,
		TransactionsBlocked:     blocked,
		TransactionsFlagged:     flagged,
		TotalProtectionClaims:  uint64(len(s.claims)),
	}
}

// ============================================================================
// Handlers
// ============================================================================

type Handler struct {
	service *ShieldService
}

func NewHandler(svc *ShieldService) *Handler {
	return &Handler{service: svc}
}

func (h *Handler) ActivateShield(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID           uint32 `json:"user_id"`
		ProtectionLimit  uint64 `json:"protection_limit"`
		ProtectionLevel uint8  `json:"protection_level"`
		DurationDays    uint64 `json:"duration_days"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Error: &APIError{Code: "INVALID_REQUEST", Message: err.Error()}})
		return
	}

	shield := h.service.ActivateShield(req.UserID, req.ProtectionLimit, req.ProtectionLevel, req.DurationDays)
	json.NewEncoder(w).Encode(APIResponse{Success: true, Data: shield})
}

func (h *Handler) GetShield(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var userID uint32
	fmt.Sscanf(vars["user_id"], "%d", &userID)

	shield := h.service.GetShield(userID)
	if shield == nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Error: &APIError{Code: "NOT_FOUND", Message: "Shield not found"}})
		return
	}

	json.NewEncoder(w).Encode(APIResponse{Success: true, Data: shield})
}

func (h *Handler) AnalyzeTransaction(w http.ResponseWriter, r *http.Request) {
	var req TransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Error: &APIError{Code: "INVALID_REQUEST", Message: err.Error()}})
		return
	}

	assessment := h.service.AnalyzeTransaction(&req)
	json.NewEncoder(w).Encode(APIResponse{Success: true, Data: assessment})
}

func (h *Handler) GetAssessment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var requestID uint64
	fmt.Sscanf(vars["id"], "%d", &requestID)

	assessment := h.service.GetAssessment(requestID)
	if assessment == nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Error: &APIError{Code: "NOT_FOUND", Message: "Assessment not found"}})
		return
	}

	json.NewEncoder(w).Encode(APIResponse{Success: true, Data: assessment})
}

func (h *Handler) FileClaim(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID      uint64  `json:"user_id"`
		RequestID   uint64  `json:"request_id"`
		TxHash     uint64  `json:"tx_hash"`
		Amount     uint64  `json:"amount"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Error: &APIError{Code: "INVALID_REQUEST", Message: err.Error()}})
		return
	}

	claim := h.service.FileClaim(req.UserID, req.RequestID, req.TxHash, req.Amount, req.Description)
	json.NewEncoder(w).Encode(APIResponse{Success: true, Data: claim})
}

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats := h.service.GetStats()
	json.NewEncoder(w).Encode(APIResponse{Success: true, Data: stats})
}

// ============================================================================
// Main
// ============================================================================

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("Starting TigerWallet Transaction Shield API...")

	service := NewShieldService()
	handler := NewHandler(service)

	router := mux.NewRouter()
	router.Use(handlers.ContentTypeHandler(handlers.LoggingHandler(os.Stdout, router), "application/json"))

	// Routes
	router.HandleFunc("/api/v1/shield/activate", handler.ActivateShield).Methods("POST")
	router.HandleFunc("/api/v1/shield/{user_id}", handler.GetShield).Methods("GET")
	router.HandleFunc("/api/v1/analyze", handler.AnalyzeTransaction).Methods("POST")
	router.HandleFunc("/api/v1/assessment/{id}", handler.GetAssessment).Methods("GET")
	router.HandleFunc("/api/v1/claims", handler.FileClaim).Methods("POST"])
	router.HandleFunc("/api/v1/stats", handler.GetStats).Methods("GET")

	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	}).Methods("GET")

	srv := &http.Server{Addr: cfg.ServerPort, Handler: router}

	go func() {
		log.Printf("Server listening on %s", cfg.ServerPort)
		srv.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
