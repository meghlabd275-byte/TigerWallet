/**
 * TigerWallet Prediction Markets - Go API
 * Distributed worldwide API service for prediction markets
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
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ServerPort       string
	RedisAddr        string
	PostgresAddr     string
	RateLimitRPS     float64
	RateLimitBurst   int
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	IdleTimeout      time.Duration
	MaxHeaderBytes   int
	EnableCors       bool
	AllowedOrigins   []string
	EnableLogging    bool
	LogLevel         string
}

var config = Config{
	ServerPort:     ":8443",
	RedisAddr:      "localhost:6379",
	RateLimitRPS:   100,
	RateLimitBurst: 200,
	ReadTimeout:    30 * time.Second,
	WriteTimeout:   30 * time.Second,
	IdleTimeout:    120 * time.Second,
	MaxHeaderBytes: 1 << 20,
	EnableCors:     true,
	AllowedOrigins: []string{"*"},
	EnableLogging:  true,
	LogLevel:       "info",
}

// ============================================================================
// Data Models
// ============================================================================

type Market struct {
	MarketID        uint64    `json:"market_id"`
	Question        string    `json:"question"`
	Description    string    `json:"description"`
	Category        string    `json:"category"`
	OutcomeType     string    `json:"outcome_type"`
	Outcomes        []Outcome `json:"outcomes"`
	Status          string    `json:"status"`
	ResolutionTime  uint64    `json:"resolution_time"`
	ResolvedOutcome *uint32   `json:"resolved_outcome,omitempty"`
	Volume24h       uint64    `json:"volume_24h"`
	TotalVolume     uint64    `json:"total_volume"`
	Featured        bool      `json:"featured"`
	ImageURL        *string   `json:"image_url,omitempty"`
	CreatedAt       uint64    `json:"created_at"`
	UpdatedAt       uint64    `json:"updated_at"`
}

type Outcome struct {
	OutcomeID   uint32  `json:"outcome_id"`
	Name        string  `json:"name"`
	Price       uint64  `json:"price"`
	Volume      uint64  `json:"volume"`
	Probability float64 `json:"probability"`
}

type Order struct {
	OrderID       uint64 `json:"order_id"`
	MarketID       uint64 `json:"market_id"`
	OutcomeID     uint32 `json:"outcome_id"`
	UserID        uint32 `json:"user_id"`
	OrderType     string `json:"order_type"`
	Side          string `json:"side"`
	Price         uint64 `json:"price"`
	Amount        uint64 `json:"amount"`
	FilledAmount  uint64 `json:"filled_amount"`
	Status        string `json:"status"`
	Timestamp     uint64 `json:"timestamp"`
	ExpiresAt     uint64 `json:"expires_at"`
}

type Position struct {
	MarketID     uint64 `json:"market_id"`
	OutcomeID    uint32 `json:"outcome_id"`
	UserID       uint32 `json:"user_id"`
	Quantity     uint64 `json:"quantity"`
	AvgPrice     uint64 `json:"avg_price"`
	Invested     uint64 `json:"invested"`
	CurrentValue uint64 `json:"current_value"`
	ProfitLoss   int64  `json:"profit_loss"`
}

type Trade struct {
	TradeID    uint64  `json:"trade_id"`
	OrderID    uint64  `json:"order_id"`
	MarketID   uint64  `json:"market_id"`
	OutcomeID  uint32  `json:"outcome_id"`
	Side       string  `json:"side"`
	Price      uint64  `json:"price"`
	Amount     uint64  `json:"amount"`
	Fees       uint64  `json:"fees"`
	Timestamp  uint64  `json:"timestamp"`
	UserID     uint32  `json:"user_id"`
	TxHash     *string `json:"tx_hash,omitempty"`
}

type BetSlip struct {
	MarketID          uint64 `json:"market_id"`
	OutcomeID         uint32 `json:"outcome_id"`
	Side              string `json:"side"`
	Price             uint64 `json:"price"`
	Amount            uint64 `json:"amount"`
	PotentialWinnings uint64 `json:"potential_winnings"`
	Fees              uint64 `json:"fees"`
	TotalCost         uint64 `json:"total_cost"`
}

type MarketStats struct {
	TotalMarkets   int     `json:"total_markets"`
	ActiveMarkets  int     `json:"active_markets"`
	TotalVolume    uint64  `json:"total_volume"`
	Volume24h      uint64  `json:"volume_24h"`
	TotalUsers     int     `json:"total_users"`
	TotalTrades    uint64  `json:"total_trades"`
	AvgTradeSize   uint64  `json:"avg_trade_size"`
}

// ============================================================================
// Request/Response Types
// ============================================================================

type CreateMarketRequest struct {
	Question        string   `json:"question"`
	Description     string   `json:"description"`
	OutcomeType     string   `json:"outcome_type"`
	OutcomeNames    []string `json:"outcome_names"`
	ResolutionTime  uint64   `json:"resolution_time"`
	Category        string   `json:"category"`
	ImageURL        *string  `json:"image_url,omitempty"`
}

type PlaceOrderRequest struct {
	MarketID   uint64 `json:"market_id"`
	OutcomeID  uint32 `json:"outcome_id"`
	OrderType  string `json:"order_type"`
	Side       string `json:"side"`
	Price      uint64 `json:"price"`
	Amount     uint64 `json:"amount"`
	ExpiresAt  *uint64 `json:"expires_at,omitempty"`
}

type CancelOrderRequest struct {
	OrderID uint64 `json:"order_id"`
}

type ResolveMarketRequest struct {
	OutcomeID uint32 `json:"outcome_id"`
}

type AddFundsRequest struct {
	Amount uint64 `json:"amount"`
}

type GetMarketsRequest struct {
	Status    *string `json:"status,omitempty"`
	Category  *string `json:"category,omitempty"`
	Featured  *bool   `json:"featured,omitempty"`
	Offset    uint32  `json:"offset"`
	Limit     uint32  `json:"limit"`
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

type PaginationParams struct {
	Page     uint32 `json:"page"`
	PageSize uint32 `json:"page_size"`
}

// ============================================================================
// Service Layer
// ============================================================================

type PredictionService struct {
	redis          *redis.Client
	markets        map[uint64]*Market
	orders         map[uint64]*Order
	positions      map[string]*Position
	trades         map[uint32][]*Trade
	balances       map[uint32]uint64
	nextMarketID   uint64
	nextOrderID    uint64
	nextTradeID    uint64
	mu             sync.RWMutex
	feeBPS         uint32
}

func NewPredictionService(redisAddr string) *PredictionService {
	svc := &PredictionService{
		markets:      make(map[uint64]*Market),
		orders:       make(map[uint64]*Order),
		positions:    make(map[string]*Position),
		trades:       make(map[uint32][]*Trade),
		balances:     make(map[uint32]uint64),
		nextMarketID: 1,
		nextOrderID:  1,
		nextTradeID:  1,
		feeBPS:       30, // 0.3%
	}

	// Initialize Redis connection
	svc.redis = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := svc.redis.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
		// Continue without Redis - use in-memory fallback
		svc.redis = nil
	}

	// Seed some demo markets
	svc.seedMarkets()

	return svc
}

func (s *PredictionService) seedMarkets() {
	now := uint64(time.Now().UnixMilli())

	// Binary market: BTC prediction
	btcMarket := &Market{
		MarketID:       s.nextMarketID,
		Question:       "Will Bitcoin reach $150,000 by December 2026?",
		Description:    "Prediction on Bitcoin price target for end of 2026",
		Category:       "Crypto",
		OutcomeType:    "binary",
		Status:         "active",
		ResolutionTime: now + 180*24*60*60*1000, // 180 days
		Volume24h:      50000000,
		TotalVolume:    150000000,
		Featured:       true,
		Outcomes: []Outcome{
			{OutcomeID: 0, Name: "Yes", Price: 450000, Volume: 75000000, Probability: 0.45},
			{OutcomeID: 1, Name: "No", Price: 600000, Volume: 75000000, Probability: 0.55},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.markets[btcMarket.MarketID] = btcMarket
	s.nextMarketID++

	// Election market
	electionMarket := &Market{
		MarketID:       s.nextMarketID,
		Question:       "Which party will win the 2028 US Presidential Election?",
		Description:    "Prediction on US Presidential election outcome",
		Category:       "Politics",
		OutcomeType:    "categorical",
		Status:         "active",
		ResolutionTime: now + 365*24*60*60*1000,
		Volume24h:      25000000,
		TotalVolume:    75000000,
		Featured:       true,
		Outcomes: []Outcome{
			{OutcomeID: 0, Name: "Democratic", Price: 480000, Volume: 35000000, Probability: 0.48},
			{OutcomeID: 1, Name: "Republican", Price: 520000, Volume: 40000000, Probability: 0.52},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.markets[electionMarket.MarketID] = electionMarket
	s.nextMarketID++

	// Sports market
	sportsMarket := &Market{
		MarketID:       s.nextMarketID,
		Question:       "Will Team A win the Championship?",
		Description:    "Finals outcome prediction",
		Category:       "Sports",
		OutcomeType:    "binary",
		Status:         "active",
		ResolutionTime: now + 30*24*60*60*1000,
		Volume24h:      10000000,
		TotalVolume:    30000000,
		Featured:       false,
		Outcomes: []Outcome{
			{OutcomeID: 0, Name: "Yes", Price: 550000, Volume: 16500000, Probability: 0.55},
			{OutcomeID: 1, Name: "No", Price: 500000, Volume: 13500000, Probability: 0.45},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.markets[sportsMarket.MarketID] = sportsMarket
	s.nextMarketID++

	log.Printf("Seeded %d prediction markets", len(s.markets))
}

func (s *PredictionService) CreateMarket(req *CreateMarketRequest) (*Market, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := uint64(time.Now().UnixMilli())

	outcomes := make([]Outcome, len(req.OutcomeNames))
	for i, name := range req.OutcomeNames {
		outcomes[i] = Outcome{
			OutcomeID:   uint32(i),
			Name:        name,
			Price:       500000, // Start at 0.5
			Volume:      0,
			Probability: 1.0 / float64(len(req.OutcomeNames)),
		}
	}

	market := &Market{
		MarketID:       s.nextMarketID,
		Question:       req.Question,
		Description:    req.Description,
		Category:       req.Category,
		OutcomeType:    req.OutcomeType,
		Outcomes:       outcomes,
		Status:         "active",
		ResolutionTime: req.ResolutionTime,
		Volume24h:      0,
		TotalVolume:    0,
		Featured:       false,
		ImageURL:       req.ImageURL,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	s.markets[market.MarketID] = market
	s.nextMarketID++

	return market, nil
}

func (s *PredictionService) GetMarket(marketID uint64) (*Market, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	market, ok := s.markets[marketID]
	if !ok {
		return nil, fmt.Errorf("market not found: %d", marketID)
	}
	return market, nil
}

func (s *PredictionService) GetMarkets(req *GetMarketsRequest) []*Market {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Market
	for _, m := range s.markets {
		if req.Status != nil && m.Status != *req.Status {
			continue
		}
		if req.Category != nil && m.Category != *req.Category {
			continue
		}
		if req.Featured != nil && m.Featured != *req.Featured {
			continue
		}
		result = append(result, m)
	}

	// Apply pagination
	start := req.Offset
	if start > uint32(len(result)) {
		start = uint32(len(result))
	}
	end := start + req.Limit
	if end > uint32(len(result)) {
		end = uint32(len(result))
	}

	return result[start:end]
}

func (s *PredictionService) GetFeaturedMarkets() []*Market {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Market
	for _, m := range s.markets {
		if m.Featured && m.Status == "active" {
			result = append(result, m)
		}
	}
	return result
}

func (s *PredictionService) PlaceOrder(userID uint32, req *PlaceOrderRequest) (*Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	market, ok := s.markets[req.MarketID]
	if !ok {
		return nil, fmt.Errorf("market not found: %d", req.MarketID)
	}

	if market.Status != "active" {
		return nil, fmt.Errorf("market is not active")
	}

	if req.OutcomeID >= uint32(len(market.Outcomes)) {
		return nil, fmt.Errorf("invalid outcome: %d", req.OutcomeID)
	}

	if req.Price == 0 || req.Price > 1000000 {
		return nil, fmt.Errorf("invalid price: %d", req.Price)
	}

	if req.Amount == 0 {
		return nil, fmt.Errorf("invalid amount")
	}

	totalCost := (req.Amount * req.Price) / 1000000
	fee := (totalCost * s.feeBPS) / 10000
	totalRequired := totalCost + fee

	balance := s.balances[userID]
	if balance < totalRequired {
		return nil, fmt.Errorf("insufficient balance: need %d, have %d", totalRequired, balance)
	}

	s.balances[userID] = balance - totalRequired

	now := uint64(time.Now().UnixMilli())
	expiresAt := now + 86400000 // 24 hours default
	if req.ExpiresAt != nil {
		expiresAt = *req.ExpiresAt
	}

	order := &Order{
		OrderID:       s.nextOrderID,
		MarketID:      req.MarketID,
		OutcomeID:     req.OutcomeID,
		UserID:        userID,
		OrderType:     req.OrderType,
		Side:          req.Side,
		Price:         req.Price,
		Amount:        req.Amount,
		FilledAmount:  0,
		Status:        "pending",
		Timestamp:     now,
		ExpiresAt:     expiresAt,
	}

	s.orders[order.OrderID] = order
	s.nextOrderID++

	// Update market volume
	market.TotalVolume += totalCost
	market.Volume24h += totalCost
	market.Outcomes[req.OutcomeID].Volume += req.Amount
	market.UpdatedAt = now

	return order, nil
}

func (s *PredictionService) CancelOrder(userID, orderID uint32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, ok := s.orders[orderID]
	if !ok {
		return fmt.Errorf("order not found: %d", orderID)
	}

	if order.UserID != userID {
		return fmt.Errorf("order not found: %d", orderID)
	}

	if order.Status == "filled" || order.Status == "cancelled" {
		return fmt.Errorf("order already %s", order.Status)
	}

	// Refund remaining
	remaining := order.Amount - order.FilledAmount
	refund := (remaining * order.Price) / 1000000

	s.balances[userID] += refund
	order.Status = "cancelled"

	return nil
}

func (s *PredictionService) GetUserPositions(userID uint32) []*Position {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Position
	for _, p := range s.positions {
		if p.UserID == userID {
			result = append(result, p)
		}
	}
	return result
}

func (s *PredictionService) GetUserTrades(userID uint32, page, pageSize uint32) ([]*Trade, uint64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	trades := s.trades[userID]
	totalCount := uint64(len(trades))

	start := page * pageSize
	if start > uint32(len(trades)) {
		start = uint32(len(trades))
	}
	end := start + pageSize
	if end > uint32(len(trades)) {
		end = uint32(len(trades))
	}

	return trades[start:end], totalCount
}

func (s *PredictionService) ResolveMarket(marketID uint32, outcomeID uint32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	market, ok := s.markets[marketID]
	if !ok {
		return fmt.Errorf("market not found: %d", marketID)
	}

	if market.Status != "active" {
		return fmt.Errorf("market is not active")
	}

	if outcomeID >= uint32(len(market.Outcomes)) {
		return fmt.Errorf("invalid outcome: %d", outcomeID)
	}

	now := uint64(time.Now().UnixMilli())
	market.Status = "resolved"
	market.ResolvedOutcome = &outcomeID
	market.UpdatedAt = now

	// Update positions
	resolutionPrice := market.Outcomes[outcomeID].Price
	for key, pos := range s.positions {
		if key == fmt.Sprintf("%d_%d_%d", pos.UserID, marketID, outcomeID) {
			pos.CurrentValue = (pos.Quantity * resolutionPrice) / 1000000
			pos.ProfitLoss = int64(pos.CurrentValue) - int64(pos.Invested)
		}
	}

	return nil
}

func (s *PredictionService) AddFunds(userID uint32, amount uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.balances[userID] += amount
}

func (s *PredictionService) GetBalance(userID uint32) uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.balances[userID]
}

func (s *PredictionService) CalculateBetSlip(userID uint32, marketID uint64, outcomeID uint32, side string, amount uint64) (*BetSlip, error) {
	s.mu.RLock()
	market, ok := s.markets[marketID]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("market not found: %d", marketID)
	}

	if market.Status != "active" {
		return nil, fmt.Errorf("market is not active")
	}

	if outcomeID >= uint32(len(market.Outcomes)) {
		return nil, fmt.Errorf("invalid outcome: %d", outcomeID)
	}

	price := market.Outcomes[outcomeID].Price
	totalCost := (amount * price) / 1000000
	fee := (totalCost * s.feeBPS) / 10000

	var potentialWinnings uint64
	if side == "buy" {
		potentialWinnings = ((amount * 1000000) / price) - amount
	} else {
		potentialWinnings = amount
	}

	return &BetSlip{
		MarketID:          marketID,
		OutcomeID:         outcomeID,
		Side:              side,
		Price:             price,
		Amount:            amount,
		PotentialWinnings: potentialWinnings,
		Fees:              fee,
		TotalCost:         totalCost + fee,
	}, nil
}

func (s *PredictionService) GetStats() *MarketStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var activeCount int
	var totalVolume, volume24h uint64

	for _, m := range s.markets {
		if m.Status == "active" {
			activeCount++
		}
		totalVolume += m.TotalVolume
		volume24h += m.Volume24h
	}

	var totalTrades uint64
	for _, trades := range s.trades {
		totalTrades += uint64(len(trades))
	}

	avgTradeSize := uint64(0)
	if totalTrades > 0 {
		avgTradeSize = totalVolume / totalTrades
	}

	return &MarketStats{
		TotalMarkets:  len(s.markets),
		ActiveMarkets: activeCount,
		TotalVolume:   totalVolume,
		Volume24h:     volume24h,
		TotalUsers:    len(s.balances),
		TotalTrades:   totalTrades,
		AvgTradeSize:  avgTradeSize,
	}
}

// ============================================================================
// HTTP Handlers
// ============================================================================

type Handler struct {
	service   *PredictionService
	limiter   *rate.Limiter
	stats     *RequestStats
}

type RequestStats struct {
	TotalRequests   uint64
	ActiveRequests  uint64
	TotalErrors     uint64
	TotalLatencyUs  uint64
	mu             sync.Mutex
}

func NewHandler(service *PredictionService) *Handler {
	return &Handler{
		service: service,
		limiter: rate.NewLimiter(
			rate.Limit(config.RateLimitRPS),
			config.RateLimitBurst,
		),
		stats: &RequestStats{},
	}
}

func (h *Handler) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.limiter.Allow() {
			h.stats.mu.Lock()
			h.stats.TotalErrors++
			h.stats.mu.Unlock()

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Error: &APIError{
					Code:    "RATE_LIMIT_EXCEEDED",
					Message: "Too many requests",
				},
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) JSONMiddleware(next http.Handler) http.Handler {
	return handlers.ContentTypeHandler(next, "application/json")
}

func (h *Handler) LoggingMiddleware(next http.Handler) http.Handler {
	return handlers.LoggingHandler(os.Stdout, next)
}

func (h *Handler) CORSMiddleware(next http.Handler) http.Handler {
	if !config.EnableCors {
		return next
	}
	return handlers.CORS(
		handlers.AllowedOrigins(config.AllowedOrigins),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
	)(next)
}

func (h *Handler) GetMarkets(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()

	offset := uint32(0)
	limit := uint32(20)
	if o := vars.Get("offset"); o != "" {
		if parsed, err := fmt.Sscanf(o, "%d", &offset); err == nil && parsed > 0 {
			offset = uint32(parsed)
		}
	}
	if l := vars.Get("limit"); l != "" {
		if parsed, err := fmt.Sscanf(l, "%d", &limit); err == nil && parsed > 0 && parsed <= 100 {
			limit = uint32(parsed)
		}
	}

	var status, category *string
	if s := vars.Get("status"); s != "" {
		status = &s
	}
	if c := vars.Get("category"); c != "" {
		category = &c
	}

	req := &GetMarketsRequest{
		Status:   status,
		Category: category,
		Offset:   offset,
		Limit:    limit,
	}

	markets := h.service.GetMarkets(req)

	h.stats.mu.Lock()
	h.stats.TotalRequests++
	h.stats.mu.Unlock()

	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    markets,
	})
}

func (h *Handler) GetMarket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var marketID uint64
	if _, err := fmt.Sscanf(vars["id"], "%d", &marketID); err != nil {
		h.stats.mu.Lock()
		h.stats.TotalErrors++
		h.stats.mu.Unlock()

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_ID", Message: "Invalid market ID"},
		})
		return
	}

	market, err := h.service.GetMarket(marketID)
	if err != nil {
		h.stats.mu.Lock()
		h.stats.TotalErrors++
		h.stats.mu.Unlock()

		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   &APIError{Code: "NOT_FOUND", Message: err.Error()},
		})
		return
	}

	h.stats.mu.Lock()
	h.stats.TotalRequests++
	h.stats.mu.Unlock()

	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    market,
	})
}

func (h *Handler) CreateMarket(w http.ResponseWriter, r *http.Request) {
	var req CreateMarketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.stats.mu.Lock()
		h.stats.TotalErrors++
		h.stats.mu.Unlock()

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_REQUEST", Message: err.Error()},
		})
		return
	}

	market, err := h.service.CreateMarket(&req)
	if err != nil {
		h.stats.mu.Lock()
		h.stats.TotalErrors++
		h.stats.mu.Unlock()

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   &APIError{Code: "CREATE_FAILED", Message: err.Error()},
		})
		return
	}

	h.stats.mu.Lock()
	h.stats.TotalRequests++
	h.stats.mu.Unlock()

	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    market,
	})
}

func (h *Handler) PlaceOrder(w http.ResponseWriter, r *http.Request) {
	// Get user ID from header (in production, extract from JWT)
	userIDStr := r.Header.Get("X-User-ID")
	var userID uint32 = 1 // Default for demo

	if userIDStr != "" {
		fmt.Sscanf(userIDStr, "%d", &userID)
	}

	var req PlaceOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.stats.mu.Lock()
		h.stats.TotalErrors++
		h.stats.mu.Unlock()

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_REQUEST", Message: err.Error()},
		})
		return
	}

	order, err := h.service.PlaceOrder(userID, &req)
	if err != nil {
		h.stats.mu.Lock()
		h.stats.TotalErrors++
		h.stats.mu.Unlock()

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   &APIError{Code: "ORDER_FAILED", Message: err.Error()},
		})
		return
	}

	h.stats.mu.Lock()
	h.stats.TotalRequests++
	h.stats.mu.Unlock()

	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    order,
	})
}

func (h *Handler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var orderID uint64
	if _, err := fmt.Sscanf(vars["id"], "%d", &orderID); err != nil {
		h.stats.mu.Lock()
		h.stats.TotalErrors++
		h.stats.mu.Unlock()

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_ID", Message: "Invalid order ID"},
		})
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	var userID uint32 = 1
	if userIDStr != "" {
		fmt.Sscanf(userIDStr, "%d", &userID)
	}

	if err := h.service.CancelOrder(userID, uint32(orderID)); err != nil {
		h.stats.mu.Lock()
		h.stats.TotalErrors++
		h.stats.mu.Unlock()

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   &APIError{Code: "CANCEL_FAILED", Message: err.Error()},
		})
		return
	}

	h.stats.mu.Lock()
	h.stats.TotalRequests++
	h.stats.mu.Unlock()

	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
	})
}

func (h *Handler) GetPositions(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	var userID uint32 = 1
	if userIDStr != "" {
		fmt.Sscanf(userIDStr, "%d", &userID)
	}

	positions := h.service.GetUserPositions(userID)

	h.stats.mu.Lock()
	h.stats.TotalRequests++
	h.stats.mu.Unlock()

	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    positions,
	})
}

func (h *Handler) GetTrades(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	var userID uint32 = 1
	if userIDStr != "" {
		fmt.Sscanf(userIDStr, "%d", &userID)
	}

	page := uint32(0)
	pageSize := uint32(50)

	if p := r.URL.Query().Get("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		fmt.Sscanf(ps, "%d", &pageSize)
	}

	trades, total := h.service.GetUserTrades(userID, page, pageSize)

	h.stats.mu.Lock()
	h.stats.TotalRequests++
	h.stats.mu.Unlock()

	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"trades":     trades,
			"total":      total,
			"page":       page,
			"page_size":  pageSize,
		},
	})
}

func (h *Handler) GetBalance(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	var userID uint32 = 1
	if userIDStr != "" {
		fmt.Sscanf(userIDStr, "%d", &userID)
	}

	balance := h.service.GetBalance(userID)

	h.stats.mu.Lock()
	h.stats.TotalRequests++
	h.stats.mu.Unlock()

	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"balance": balance,
		},
	})
}

func (h *Handler) AddFunds(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	var userID uint32 = 1
	if userIDStr != "" {
		fmt.Sscanf(userIDStr, "%d", &userID)
	}

	var req AddFundsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.stats.mu.Lock()
		h.stats.TotalErrors++
		h.stats.mu.Unlock()

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_REQUEST", Message: err.Error()},
		})
		return
	}

	h.service.AddFunds(userID, req.Amount)

	h.stats.mu.Lock()
	h.stats.TotalRequests++
	h.stats.mu.Unlock()

	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
	})
}

func (h *Handler) CalculateBetSlip(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	var userID uint32 = 1
	if userIDStr != "" {
		fmt.Sscanf(userIDStr, "%d", &userID)
	}

	var req struct {
		MarketID  uint64 `json:"market_id"`
		OutcomeID uint32 `json:"outcome_id"`
		Side      string `json:"side"`
		Amount    uint64 `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.stats.mu.Lock()
		h.stats.TotalErrors++
		h.stats.mu.Unlock()

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   &APIError{Code: "INVALID_REQUEST", Message: err.Error()},
		})
		return
	}

	slip, err := h.service.CalculateBetSlip(userID, req.MarketID, req.OutcomeID, req.Side, req.Amount)
	if err != nil {
		h.stats.mu.Lock()
		h.stats.TotalErrors++
		h.stats.mu.Unlock()

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   &APIError{Code: "CALC_FAILED", Message: err.Error()},
		})
		return
	}

	h.stats.mu.Lock()
	h.stats.TotalRequests++
	h.stats.mu.Unlock()

	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    slip,
	})
}

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats := h.service.GetStats()

	h.stats.mu.Lock()
	h.stats.TotalRequests++
	h.stats.mu.Unlock()

	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    stats,
	})
}

func (h *Handler) GetFeatured(w http.ResponseWriter, r *http.Request) {
	markets := h.service.GetFeaturedMarkets()

	h.stats.mu.Lock()
	h.stats.TotalRequests++
	h.stats.mu.Unlock()

	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    markets,
	})
}

// ============================================================================
// Main
// ============================================================================

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("Starting TigerWallet Prediction Markets API...")

	// Initialize service
	service := NewPredictionService(config.RedisAddr)
	handler := NewHandler(service)

	// Create router
	router := mux.NewRouter()

	// Apply middlewares
	router.Use(handler.RateLimit)
	router.Use(handler.JSONMiddleware)
	if config.EnableLogging {
		router.Use(handler.LoggingMiddleware)
	}
	router.Use(handler.CORSMiddleware)

	// Routes
	router.HandleFunc("/api/v1/markets", handler.GetMarkets).Methods("GET")
	router.HandleFunc("/api/v1/markets/featured", handler.GetFeatured).Methods("GET")
	router.HandleFunc("/api/v1/markets", handler.CreateMarket).Methods("POST")
	router.HandleFunc("/api/v1/markets/{id}", handler.GetMarket).Methods("GET")
	router.HandleFunc("/api/v1/markets/{id}/resolve", handler.ResolveMarket).Methods("POST")
	router.HandleFunc("/api/v1/orders", handler.PlaceOrder).Methods("POST")
	router.HandleFunc("/api/v1/orders/{id}", handler.CancelOrder).Methods("DELETE")
	router.HandleFunc("/api/v1/positions", handler.GetPositions).Methods("GET")
	router.HandleFunc("/api/v1/trades", handler.GetTrades).Methods("GET")
	router.HandleFunc("/api/v1/balance", handler.GetBalance).Methods("GET")
	router.HandleFunc("/api/v1/balance/add", handler.AddFunds).Methods("POST")
	router.HandleFunc("/api/v1/slip", handler.CalculateBetSlip).Methods("POST")
	router.HandleFunc("/api/v1/stats", handler.GetStats).Methods("GET")

	// Health check
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	}).Methods("GET")

	// Create server
	srv := &http.Server{
		Addr:           config.ServerPort,
		Handler:        router,
		ReadTimeout:    config.ReadTimeout,
		WriteTimeout:   config.WriteTimeout,
		IdleTimeout:    config.IdleTimeout,
		MaxHeaderBytes: config.MaxHeaderBytes,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server listening on %s", config.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
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
