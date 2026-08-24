/**
 * TigerWallet Payment Card - Go API
 * Distributed worldwide payment card API service
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

	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ServerPort    string
	RedisAddr     string
	RateLimitRPS  float64
	RateLimitBurst int
}

var cfg = Config{
	ServerPort:    ":8444",
	RedisAddr:     "localhost:6379",
	RateLimitRPS:  200,
	RateLimitBurst: 400,
}

// ============================================================================
// Data Models
// ============================================================================

type CardToken struct {
	TokenID           string    `json:"token_id"`
	LastFour         string    `json:"last_four"`
	CardType         string    `json:"card_type"` // debit, credit
	Network           string    `json:"network"`   // visa, mastercard, amex
	ExpMonth         uint8     `json:"exp_month"`
	ExpYear          uint8     `json:"exp_year"`
	CardholderName   string    `json:"cardholder_name"`
	CreatedAt        uint64    `json:"created_at"`
	LastUsedAt       uint64    `json:"last_used_at"`
	IsFrozen         bool      `json:"is_frozen"`
	DailyLimit       uint32    `json:"daily_limit"`
	MonthlyLimit     uint32    `json:"monthly_limit"`
}

type Transaction struct {
	TransactionID    uint64    `json:"transaction_id"`
	UserID           uint64    `json:"user_id"`
	CardID           string    `json:"card_id"`
	TransactionType  string    `json:"transaction_type"` // purchase, withdrawal, refund
	Status           string    `json:"status"`          // pending, processing, approved, declined, completed
	Amount           uint32    `json:"amount"`
	OriginalAmount   uint32    `json:"original_amount"`
	Currency         string    `json:"currency"`
	MerchantID       uint64    `json:"merchant_id"`
	MerchantName     string    `json:"merchant_name"`
	MerchantCategory string    `json:"merchant_category"`
	Country          string    `json:"country"`
	ExchangeRate     int32     `json:"exchange_rate"`
	Fees             uint32    `json:"fees"`
	Cashback         uint32    `json:"cashback"`
	Timestamp        uint64    `json:"timestamp"`
	ProcessedAt      uint64    `json:"processed_at"`
	RiskScore        uint8     `json:"risk_score"`
	RejectionReason  string    `json:"rejection_reason,omitempty"`
	AuthCode         string    `json:"auth_code"`
}

type CardLimits struct {
	DailyLimit         uint32 `json:"daily_limit"`
	MonthlyLimit       uint32 `json:"monthly_limit"`
	PerTransactionLimit uint32 `json:"per_transaction_limit"`
	DailySpent         uint32 `json:"daily_spent"`
	MonthlySpent       uint32 `json:"monthly_spent"`
	TransactionCount   uint32 `json:"transaction_count"`
}

type AuthorizationRequest struct {
	UserID          uint64  `json:"user_id"`
	CardToken       string  `json:"card_token"`
	Amount          uint32  `json:"amount"`
	Currency        string  `json:"currency"`
	MerchantID      uint64  `json:"merchant_id"`
	MerchantName    string  `json:"merchant_name"`
	MerchantCategory string `json:"merchant_category"`
	Country         string  `json:"country"`
	TerminalID      string  `json:"terminal_id"`
	Latitude        float64 `json:"latitude"`
	Longitude       float64 `json:"longitude"`
}

type AuthorizationResponse struct {
	TransactionID uint64  `json:"transaction_id"`
	Status        string  `json:"status"`
	ApprovedAmount uint32 `json:"approved_amount"`
	AuthCode      string  `json:"auth_code"`
	ResponseCode  string  `json:"response_code"`
	RiskScore     uint8   `json:"risk_score"`
	Message       string  `json:"message"`
}

type CardStats struct {
	TotalCards      int     `json:"total_cards"`
	ActiveCards     int     `json:"active_cards"`
	TotalVolume    uint64  `json:"total_volume"`
	Volume24h      uint64  `json:"volume_24h"`
	TotalTransactions uint64 `json:"total_transactions"`
	ApprovedCount   uint64  `json:"approved_count"`
	DeclinedCount   uint64  `json:"declined_count"`
	AvgTransactionSize uint64 `json:"avg_transaction_size"`
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
// Payment Service
// ============================================================================

type PaymentService struct {
	redis         *redis.Client
	cards         map[string]map[string]*CardToken  // userID -> tokenID -> CardToken
	transactions  map[uint64]*Transaction
	limits        map[string]*CardLimits
	nextTxID      uint64
	mu            sync.RWMutex
	feePercent    float32
	cashbackPercent float32
}

func NewPaymentService(redisAddr string) *PaymentService {
	svc := &PaymentService{
		cards:         make(map[string]map[string]*CardToken),
		transactions:  make(map[uint64]*Transaction),
		limits:        make(map[string]*CardLimits),
		nextTxID:      1,
		feePercent:    0.029,  // 2.9%
		cashbackPercent: 0.03,   // 3%
	}

	svc.redis = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       1,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := svc.redis.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
		svc.redis = nil
	}

	// Seed demo cards

	return svc
}



func (s *PaymentService) CreateCard(userID, cardType, network, cardholderName string, expMonth, expYear uint8) *CardToken {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := uint64(time.Now().UnixMilli())
	
	// Generate token
	lastFour := fmt.Sprintf("%04d", 1000+len(s.transactions)%9000)
	tokenID := fmt.Sprintf("tok_%s_%s", network[:2], lastFour)

	card := &CardToken{
		TokenID:         tokenID,
		LastFour:        lastFour,
		CardType:        cardType,
		Network:         network,
		ExpMonth:        expMonth,
		ExpYear:         expYear,
		CardholderName:  cardholderName,
		CreatedAt:        now,
		LastUsedAt:       now,
		IsFrozen:        false,
		DailyLimit:      10000,
		MonthlyLimit:     50000,
	}

	if s.cards[userID] == nil {
		s.cards[userID] = make(map[string]*CardToken)
	}
	s.cards[userID][tokenID] = card

	// Initialize limits
	s.limits[tokenID] = &CardLimits{
		DailyLimit:         10000,
		MonthlyLimit:       50000,
		PerTransactionLimit: 5000,
	}

	return card
}

func (s *PaymentService) GetUserCards(userID string) []*CardToken {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cards, ok := s.cards[userID]
	if !ok {
		return []*CardToken{}
	}

	result := make([]*CardToken, 0, len(cards))
	for _, card := range cards {
		result = append(result, card)
	}
	return result
}

func (s *PaymentService) GetCard(userID, tokenID string) *CardToken {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if cards, ok := s.cards[userID]; ok {
		return cards[tokenID]
	}
	return nil
}

func (s *PaymentService) FreezeCard(userID, tokenID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cards, ok := s.cards[userID]; ok {
		if card, ok := cards[tokenID]; ok {
			card.IsFrozen = true
			return true
		}
	}
	return false
}

func (s *PaymentService) UnfreezeCard(userID, tokenID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cards, ok := s.cards[userID]; ok {
		if card, ok := cards[tokenID]; ok {
			card.IsFrozen = false
			return true
		}
	}
	return false
}

func (s *PaymentService) UpdateCardLimits(userID, tokenID string, limits *CardLimits) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cards, ok := s.cards[userID]; ok {
		if card, ok := cards[tokenID]; ok {
			card.DailyLimit = limits.DailyLimit
			card.MonthlyLimit = limits.MonthlyLimit
			if s.limits[tokenID] != nil {
				s.limits[tokenID].DailyLimit = limits.DailyLimit
				s.limits[tokenID].MonthlyLimit = limits.MonthlyLimit
				s.limits[tokenID].PerTransactionLimit = limits.PerTransactionLimit
			}
			return true
		}
	}
	return false
}

func (s *PaymentService) DeleteCard(userID, tokenID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cards, ok := s.cards[userID]; ok {
		if _, ok := cards[tokenID]; ok {
			delete(cards, tokenID)
			delete(s.limits, tokenID)
			return true
		}
	}
	return false
}

func (s *PaymentService) Authorize(req *AuthorizationRequest) *AuthorizationResponse {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := uint64(time.Now().UnixMilli())

	// Validate card
	card := s.cards[fmt.Sprintf("%d", req.UserID)][req.CardToken]
	if card == nil {
		return &AuthorizationResponse{
			Status:      "declined",
			ResponseCode: "CARD_NOT_FOUND",
			Message:     "Card not found",
		}
	}

	// Check if frozen
	if card.IsFrozen {
		return &AuthorizationResponse{
			Status:      "declined",
			ResponseCode: "CARD_FROZEN",
			Message:     "Card is frozen",
		}
	}

	// Check expiry
	if card.ExpYear < 26 || (card.ExpYear == 26 && card.ExpMonth < 6) {
		return &AuthorizationResponse{
			Status:      "declined",
			ResponseCode: "CARD_EXPIRED",
			Message:     "Card has expired",
		}
	}

	// Check limits
	limits := s.limits[req.CardToken]
	if limits != nil {
		if req.Amount > limits.PerTransactionLimit {
			return &AuthorizationResponse{
				Status:      "declined",
				ResponseCode: "AMOUNT_EXCEEDS_LIMIT",
				Message:     "Amount exceeds per-transaction limit",
			}
		}
		if req.Amount+limits.DailySpent > limits.DailyLimit {
			return &AuthorizationResponse{
				Status:      "declined",
				ResponseCode: "DAILY_LIMIT_EXCEEDED",
				Message:     "Daily limit exceeded",
			}
		}
	}

	// Generate transaction
	txID := s.nextTxID
	s.nextTxID++

	tx := &Transaction{
		TransactionID:    txID,
		UserID:           req.UserID,
		CardID:           req.CardToken,
		TransactionType:  "purchase",
		Status:           "approved",
		Amount:           req.Amount,
		OriginalAmount:   req.Amount,
		Currency:         req.Currency,
		MerchantID:       req.MerchantID,
		MerchantName:     req.MerchantName,
		MerchantCategory: req.MerchantCategory,
		Country:          req.Country,
		Timestamp:        now,
		ProcessedAt:      now,
		RiskScore:        10, // Low risk for demo
	}

	// Calculate fees and cashback
	tx.Fees = uint32(float32(req.Amount) * s.feePercent)
	tx.Cashback = uint32(float32(req.Amount) * s.cashbackPercent)

	// Generate auth code
	authCode := fmt.Sprintf("%06d", 100000+(txID%900000))

	s.transactions[txID] = tx

	// Update limits
	if limits != nil {
		limits.DailySpent += req.Amount
		limits.MonthlySpent += req.Amount
		limits.TransactionCount++
	}

	// Update card last used
	card.LastUsedAt = now

	return &AuthorizationResponse{
		TransactionID: txID,
		Status:        "approved",
		ApprovedAmount: req.Amount,
		AuthCode:      authCode,
		ResponseCode:  "APPROVED",
		RiskScore:    10,
		Message:      "Transaction approved",
	}
}

func (s *PaymentService) Refund(txID uint64, amount uint32) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, ok := s.transactions[txID]
	if !ok || tx.Status != "completed" {
		return false
	}

	if amount > tx.Amount {
		return false
	}

	// Create refund transaction
	refundID := s.nextTxID
	s.nextTxID++

	now := uint64(time.Now().UnixMilli())
	refund := &Transaction{
		TransactionID:   refundID,
		UserID:         tx.UserID,
		CardID:         tx.CardID,
		TransactionType: "refund",
		Status:          "completed",
		Amount:         amount,
		OriginalAmount: amount,
		Currency:       tx.Currency,
		MerchantID:     tx.MerchantID,
		MerchantName:   tx.MerchantName,
		Timestamp:      now,
		ProcessedAt:    now,
	}

	s.transactions[refundID] = refund

	// Update original transaction
	tx.Amount -= amount

	return true
}

func (s *PaymentService) GetTransaction(txID uint64) *Transaction {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.transactions[txID]
}

func (s *PaymentService) GetUserTransactions(userID string, limit int) []*Transaction {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Transaction
	count := 0
	for _, tx := range s.transactions {
		if tx.UserID == uint64(parseUserID(userID)) && count < limit {
			result = append(result, tx)
			count++
		}
	}

	// Sort by timestamp descending
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].Timestamp < result[j].Timestamp {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

func parseUserID(s string) int {
	var id int
	fmt.Sscanf(s, "%d", &id)
	return id
}

func (s *PaymentService) GetStats() *CardStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var totalVol, vol24h uint64
	var approved, declined uint64

	now := uint64(time.Now().UnixMilli())
	dayAgo := now - 24*60*60*1000

	for _, tx := range s.transactions {
		totalVol += uint64(tx.Amount)
		if tx.Timestamp > dayAgo {
			vol24h += uint64(tx.Amount)
		}
		if tx.Status == "approved" || tx.Status == "completed" {
			approved++
		} else if tx.Status == "declined" {
			declined++
		}
	}

	var activeCards int
	for _, userCards := range s.cards {
		for _, card := range userCards {
			if !card.IsFrozen {
				activeCards++
			}
		}
	}

	totalCards := len(s.cards)

	avgSize := uint64(0)
	if approved > 0 {
		avgSize = totalVol / approved
	}

	return &CardStats{
		TotalCards:          totalCards,
		ActiveCards:         activeCards,
		TotalVolume:        totalVol,
		Volume24h:          vol24h,
		TotalTransactions:   approved + declined,
		ApprovedCount:      approved,
		DeclinedCount:      declined,
		AvgTransactionSize: avgSize,
	}
}

// ============================================================================
// HTTP Handlers
// ============================================================================

type Handler struct {
	service *PaymentService
	limiter *rate.Limiter
}

func NewHandler(service *PaymentService) *Handler {
	return &Handler{
		service: service,
		limiter: rate.NewLimiter(rate.Limit(cfg.RateLimitRPS), cfg.RateLimitBurst),
	}
}

func (h *Handler) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.limiter.Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Error:   &APIError{Code: "RATE_LIMIT", Message: "Too many requests"},
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) CreateCard(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "1"
	}

	var req struct {
		CardType       string `json:"card_type"`
		Network        string `json:"network"`
		CardholderName string `json:"cardholder_name"`
		ExpMonth       uint8  `json:"exp_month"`
		ExpYear        uint8  `json:"exp_year"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_REQUEST", Message: err.Error()},
		})
		return
	}

	card := h.service.CreateCard(userID, req.CardType, req.Network, req.CardholderName, req.ExpMonth, req.ExpYear)

	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    card,
	})
}

func (h *Handler) GetCards(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "1"
	}

	cards := h.service.GetUserCards(userID)

	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    cards,
	})
}

func (h *Handler) GetCard(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "1"
	}

	card := h.service.GetCard(userID, vars["token"])
	if card == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   &APIError{Code: "NOT_FOUND", Message: "Card not found"},
		})
		return
	}

	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    card,
	})
}

func (h *Handler) FreezeCard(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "1"
	}

	success := h.service.FreezeCard(userID, vars["token"])
	if !success {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   &APIError{Code: "NOT_FOUND", Message: "Card not found"},
		})
		return
	}

	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
	})
}

func (h *Handler) UnfreezeCard(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "1"
	}

	success := h.service.UnfreezeCard(userID, vars["token"])
	if !success {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   &APIError{Code: "NOT_FOUND", Message: "Card not found"},
		})
		return
	}

	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
	})
}

func (h *Handler) Authorize(w http.ResponseWriter, r *http.Request) {
	var req AuthorizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_REQUEST", Message: err.Error()},
		})
		return
	}

	if req.UserID == 0 {
		req.UserID = 1
	}

	response := h.service.Authorize(&req)

	json.NewEncoder(w).Encode(APIResponse{
		Success: response.Status == "approved",
		Data:    response,
	})
}

func (h *Handler) GetTransactions(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "1"
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	transactions := h.service.GetUserTransactions(userID, limit)

	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    transactions,
	})
}

func (h *Handler) Refund(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var txID uint64
	fmt.Sscanf(vars["id"], "%d", &txID)

	var req struct {
		Amount uint32 `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_REQUEST", Message: err.Error()},
		})
		return
	}

	success := h.service.Refund(txID, req.Amount)
	if !success {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   &APIError{Code: "REFUND_FAILED", Message: "Refund failed"},
		})
		return
	}

	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
	})
}

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats := h.service.GetStats()

	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    stats,
	})
}

func (h *Handler) GetTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var txID uint64
	fmt.Sscanf(vars["id"], "%d", &txID)

	tx := h.service.GetTransaction(txID)
	if tx == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   &APIError{Code: "NOT_FOUND", Message: "Transaction not found"},
		})
		return
	}

	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    tx,
	})
}

// ============================================================================
// Main
// ============================================================================

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("Starting TigerWallet Payment Card API...")

	service := NewPaymentService(cfg.RedisAddr)
	handler := NewHandler(service)

	router := mux.NewRouter()
	router.Use(handler.RateLimit)

	// Card management
	router.HandleFunc("/api/v1/cards", handler.CreateCard).Methods("POST")
	router.HandleFunc("/api/v1/cards", handler.GetCards).Methods("GET")
	router.HandleFunc("/api/v1/cards/{token}", handler.GetCard).Methods("GET")
	router.HandleFunc("/api/v1/cards/{token}/freeze", handler.FreezeCard).Methods("POST")
	router.HandleFunc("/api/v1/cards/{token}/unfreeze", handler.UnfreezeCard).Methods("POST")

	// Transactions
	router.HandleFunc("/api/v1/authorize", handler.Authorize).Methods("POST")
	router.HandleFunc("/api/v1/transactions", handler.GetTransactions).Methods("GET")
	router.HandleFunc("/api/v1/transactions/{id}", handler.GetTransaction).Methods("GET")
	router.HandleFunc("/api/v1/transactions/{id}/refund", handler.Refund).Methods("POST")

	// Stats
	router.HandleFunc("/api/v1/stats", handler.GetStats).Methods("GET")

	// Health
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	}).Methods("GET")

	srv := &http.Server{
		Addr:         cfg.ServerPort,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("Server listening on %s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
