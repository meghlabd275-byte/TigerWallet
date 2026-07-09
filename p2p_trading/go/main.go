package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Port            string
	RedisURL        string
	ContractAddress string
	FeePercent      float64
	DisputeWindow   time.Duration
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

type P2POrder struct {
	ID              string    `json:"id"`
	Creator         string    `json:"creator"`
	Type            string    `json:"type"` // "buy" or "sell"
	Token           string    `json:"token"`
	Amount          float64   `json:"amount"`
	Price           float64   `json:"price"`
	PaymentMethod   string    `json:"paymentMethod"`
	Status          string    `json:"status"` // "open", "pending", "completed", "cancelled", "disputed"
	FiatCurrency    string    `json:"fiatCurrency"`
	MinAmount       float64   `json:"minAmount"`
	MaxAmount       float64   `json:"maxAmount"`
	Terms           string    `json:"terms"`
	CreatedAt       time.Time `json:"createdAt"`
	ExpiresAt       time.Time `json:"expiresAt"`
}

type Trade struct {
	ID              string    `json:"id"`
	OrderID         string    `json:"orderId"`
	Buyer           string    `json:"buyer"`
	Seller          string    `json:"seller"`
	Amount          float64   `json:"amount"`
	Price           float64   `json:"price"`
	Status          string    `json:"status"` // "initiated", "paid", "released", "completed", "disputed", "cancelled"
	EscrowAddress   string    `json:"escrowAddress"`
	BuyerConfirmAt  *time.Time `json:"buyerConfirmAt"`
	SellerConfirmAt *time.Time `json:"sellerConfirmAt"`
	DisputeReason   string    `json:"disputeReason"`
	CreatedAt       time.Time `json:"createdAt"`
	CompletedAt     *time.Time `json:"completedAt"`
}

type User struct {
	ID            string    `json:"id"`
	Address       string    `json:"address"`
	Username      string    `json:"username"`
	ReputationScore float64 `json:"reputationScore"`
	TotalTrades   int      `json:"totalTrades"`
	CompletedTrades int     `json:"completedTrades"`
	CancelledTrades int    `json:"cancelledTrades"`
	DisputedTrades int     `json:"disputedTrades"`
	BlockedUsers  []string `json:"blockedUsers"`
	Verified      bool      `json:"verified"`
	CreatedAt     time.Time `json:"createdAt"`
}

type ChatMessage struct {
	ID        string    `json:"id"`
	TradeID   string    `json:"tradeId"`
	Sender    string    `json:"sender"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"createdAt"`
}

type Dispute struct {
	ID          string    `json:"id"`
	TradeID     string    `json:"tradeId"`
	Reporter    string    `json:"reporter"`
	Reason      string    `json:"reason"`
	Evidence    string    `json:"evidence"`
	Status      string    `json:"status"` // "open", "under_review", "resolved", "closed"
	Resolution  string    `json:"resolution"`
	RefundTo    string    `json:"refundTo"`
	CreatedAt   time.Time `json:"createdAt"`
	ResolvedAt  *time.Time `json:"resolvedAt"`
}

// ============================================================================
// Service
// ============================================================================

type P2PService struct {
	config *Config
	redis  *redis.Client
}

func NewP2PService(config *Config) (*P2PService, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.RedisURL,
		Password: "",
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &P2PService{
		config: config,
		redis:  redisClient,
	}, nil
}

// CreateOrder creates a new P2P order
func (s *P2PService) CreateOrder(ctx context.Context, order P2POrder) (*P2POrder, error) {
	order.ID = uuid.New().String()
	order.Status = "open"
	order.CreatedAt = time.Now()
	order.ExpiresAt = time.Now().Add(24 * time.Hour)

	// Store in Redis
	data, _ := json.Marshal(order)
	s.redis.Set(ctx, fmt.Sprintf("p2p:order:%s", order.ID), data, 24*time.Hour)

	// Add to user's orders
	s.redis.SAdd(ctx, fmt.Sprintf("p2p:user:%s:orders", order.Creator), order.ID)

	return &order, nil
}

// GetOrder retrieves an order by ID
func (s *P2PService) GetOrder(ctx context.Context, orderID string) (*P2POrder, error) {
	data, err := s.redis.Get(ctx, fmt.Sprintf("p2p:order:%s", orderID)).Bytes()
	if err != nil {
		return nil, fmt.Errorf("order not found")
	}

	var order P2POrder
	json.Unmarshal(data, &order)
	return &order, nil
}

// GetOpenOrders returns all open orders with filters
func (s *P2PService) GetOpenOrders(ctx context.Context, token, orderType, paymentMethod, currency string) ([]P2POrder, error) {
	pattern := "p2p:order:*"
	keys, err := s.redis.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	var orders []P2POrder
	for _, key := range keys {
		data, err := s.redis.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}

		var order P2POrder
		json.Unmarshal(data, &order)

		if order.Status != "open" {
			continue
		}

		// Apply filters
		if token != "" && order.Token != token {
			continue
		}
		if orderType != "" && order.Type != orderType {
			continue
		}
		if paymentMethod != "" && order.PaymentMethod != paymentMethod {
			continue
		}
		if currency != "" && order.FiatCurrency != currency {
			continue
		}

		orders = append(orders, order)
	}

	return orders, nil
}

// InitiateTrade starts a P2P trade
func (s *P2PService) InitiateTrade(ctx context.Context, orderID, buyerAddress string) (*Trade, error) {
	// Get order
	order, err := s.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if order.Status != "open" {
		return nil, fmt.Errorf("order is not available")
	}

	// Generate escrow address (in production, this would be a real smart contract)
	escrowAddress := generateEscrowAddress()

	trade := &Trade{
		ID:            uuid.New().String(),
		OrderID:       orderID,
		Buyer:         buyerAddress,
		Seller:        order.Creator,
		Amount:        order.Amount,
		Price:         order.Price,
		Status:         "initiated",
		EscrowAddress: escrowAddress,
		CreatedAt:     time.Now(),
	}

	// Store trade
	data, _ := json.Marshal(trade)
	s.redis.Set(ctx, fmt.Sprintf("p2p:trade:%s", trade.ID), data, 7*24*time.Hour)

	// Update order status
	order.Status = "pending"
	orderData, _ := json.Marshal(order)
	s.redis.Set(ctx, fmt.Sprintf("p2p:order:%s", orderID), orderData, 24*time.Hour)

	return trade, nil
}

// ConfirmPayment marks payment as made
func (s *P2PService) ConfirmPayment(ctx context.Context, tradeID, confirmAddress string) (*Trade, error) {
	trade, err := s.GetTrade(ctx, tradeID)
	if err != nil {
		return nil, err
	}

	if trade.Status != "initiated" {
		return nil, fmt.Errorf("invalid trade status")
	}

	if confirmAddress != trade.Buyer {
		return nil, fmt.Errorf("only buyer can confirm payment")
	}

	now := time.Now()
	trade.Status = "paid"
	trade.BuyerConfirmAt = &now

	// Store updated trade
	data, _ := json.Marshal(trade)
	s.redis.Set(ctx, fmt.Sprintf("p2p:trade:%s", tradeID), data, 7*24*time.Hour)

	return trade, nil
}

// ReleaseCrypto releases crypto to buyer
func (s *P2PService) ReleaseCrypto(ctx context.Context, tradeID, releaseAddress string) (*Trade, error) {
	trade, err := s.GetTrade(ctx, tradeID)
	if err != nil {
		return nil, err
	}

	if trade.Status != "paid" {
		return nil, fmt.Errorf("payment not confirmed")
	}

	if releaseAddress != trade.Seller {
		return nil, fmt.Errorf("only seller can release crypto")
	}

	now := time.Now()
	trade.Status = "released"
	trade.SellerConfirmAt = &now
	trade.CompletedAt = &now

	// Store updated trade
	data, _ := json.Marshal(trade)
	s.redis.Set(ctx, fmt.Sprintf("p2p:trade:%s", tradeID), data, 7*24*time.Hour)

	// Update order status
	order, _ := s.GetOrder(ctx, trade.OrderID)
	order.Status = "completed"
	orderData, _ := json.Marshal(order)
	s.redis.Set(ctx, fmt.Sprintf("p2p:order:%s", trade.OrderID), orderData, 24*time.Hour)

	// Update user reputation
	s.updateReputation(ctx, trade.Buyer, true)
	s.updateReputation(ctx, trade.Seller, true)

	return trade, nil
}

// GetTrade retrieves a trade by ID
func (s *P2PService) GetTrade(ctx context.Context, tradeID string) (*Trade, error) {
	data, err := s.redis.Get(ctx, fmt.Sprintf("p2p:trade:%s", tradeID)).Bytes()
	if err != nil {
		return nil, fmt.Errorf("trade not found")
	}

	var trade Trade
	json.Unmarshal(data, &trade)
	return &trade, nil
}

// OpenDispute opens a dispute for a trade
func (s *P2PService) OpenDispute(ctx context.Context, tradeID, reporter, reason, evidence string) (*Dispute, error) {
	trade, err := s.GetTrade(ctx, tradeID)
	if err != nil {
		return nil, err
	}

	dispute := &Dispute{
		ID:        uuid.New().String(),
		TradeID:   tradeID,
		Reporter:  reporter,
		Reason:    reason,
		Evidence:  evidence,
		Status:    "open",
		CreatedAt: time.Now(),
	}

	// Update trade status
	trade.Status = "disputed"
	data, _ := json.Marshal(trade)
	s.redis.Set(ctx, fmt.Sprintf("p2p:trade:%s", tradeID), data, 7*24*time.Hour)

	// Store dispute
	disputeData, _ := json.Marshal(dispute)
	s.redis.Set(ctx, fmt.Sprintf("p2p:dispute:%s", dispute.ID), disputeData, 30*24*time.Hour)

	return dispute, nil
}

// SendMessage sends a chat message in a trade
func (s *P2PService) SendMessage(ctx context.Context, tradeID, sender, message string) (*ChatMessage, error) {
	msg := &ChatMessage{
		ID:        uuid.New().String(),
		TradeID:   tradeID,
		Sender:    sender,
		Message:   message,
		CreatedAt: time.Now(),
	}

	// Store in trade chat
	data, _ := json.Marshal(msg)
	s.redis.RPush(ctx, fmt.Sprintf("p2p:trade:%s:messages", tradeID), data)

	return msg, nil
}

// GetMessages retrieves chat messages for a trade
func (s *P2PService) GetMessages(ctx context.Context, tradeID string) ([]ChatMessage, error) {
	results, err := s.redis.LRange(ctx, fmt.Sprintf("p2p:trade:%s:messages", tradeID), 0, -1).Result()
	if err != nil {
		return nil, err
	}

	var messages []ChatMessage
	for _, data := range results {
		var msg ChatMessage
		json.Unmarshal([]byte(data), &msg)
		messages = append(messages, msg)
	}

	return messages, nil
}

// GetUser returns user profile
func (s *P2PService) GetUser(ctx context.Context, address string) (*User, error) {
	data, err := s.redis.Get(ctx, fmt.Sprintf("p2p:user:%s", address)).Bytes()
	if err != nil {
		// Create new user
		user := &User{
			ID:               uuid.New().String(),
			Address:          address,
			ReputationScore:  100.0,
			TotalTrades:      0,
			CompletedTrades:  0,
			CancelledTrades:  0,
			DisputedTrades:   0,
			BlockedUsers:     []string{},
			Verified:         false,
			CreatedAt:        time.Now(),
		}
		return user, nil
	}

	var user User
	json.Unmarshal(data, &user)
	return &user, nil
}

// Update user reputation
func (s *P2PService) updateReputation(ctx context.Context, address string, completed bool) {
	user, _ := s.GetUser(ctx, address)
	
	user.TotalTrades++
	if completed {
		user.CompletedTrades++
		// Increase reputation
		user.ReputationScore = math.Min(100, user.ReputationScore+2)
	} else {
		user.CancelledTrades++
		// Decrease reputation
		user.ReputationScore = math.Max(0, user.ReputationScore-5)
	}

	data, _ := json.Marshal(user)
	s.redis.Set(ctx, fmt.Sprintf("p2p:user:%s", address), data, 0)
}

// Helper functions
func generateEscrowAddress() string {
	bytes := make([]byte, 20)
	rand.Read(bytes)
	return "0x" + hex.EncodeToString(bytes)
}

// ============================================================================
// HTTP Handlers
// ============================================================================

type Handler struct {
	service *P2PService
}

func NewHandler(service *P2PService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateOrder(c *gin.Context) {
	var order P2POrder
	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.CreateOrder(c.Request.Context(), order)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, result)
}

func (h *Handler) GetOrders(c *gin.Context) {
	token := c.Query("token")
	orderType := c.Query("type")
	paymentMethod := c.Query("paymentMethod")
	currency := c.Query("currency")

	orders, err := h.service.GetOpenOrders(c.Request.Context(), token, orderType, paymentMethod, currency)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, orders)
}

func (h *Handler) InitiateTrade(c *gin.Context) {
	var req struct {
		OrderID string `json:"orderId" binding:"required"`
		Buyer   string `json:"buyer" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	trade, err := h.service.InitiateTrade(c.Request.Context(), req.OrderID, req.Buyer)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, trade)
}

func (h *Handler) ConfirmPayment(c *gin.Context) {
	tradeID := c.Param("tradeId")
	
	var req struct {
		Address string `json:"address" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	trade, err := h.service.ConfirmPayment(c.Request.Context(), tradeID, req.Address)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, trade)
}

func (h *Handler) ReleaseCrypto(c *gin.Context) {
	tradeID := c.Param("tradeId")
	
	var req struct {
		Address string `json:"address" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	trade, err := h.service.ReleaseCrypto(c.Request.Context(), tradeID, req.Address)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, trade)
}

func (h *Handler) GetTrade(c *gin.Context) {
	tradeID := c.Param("tradeId")
	
	trade, err := h.service.GetTrade(c.Request.Context(), tradeID)
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, trade)
}

func (h *Handler) OpenDispute(c *gin.Context) {
	tradeID := c.Param("tradeId")
	
	var req struct {
		Reporter string `json:"reporter" binding:"required"`
		Reason   string `json:"reason" binding:"required"`
		Evidence string `json:"evidence"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	dispute, err := h.service.OpenDispute(c.Request.Context(), tradeID, req.Reporter, req.Reason, req.Evidence)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, dispute)
}

func (h *Handler) SendMessage(c *gin.Context) {
	tradeID := c.Param("tradeId")
	
	var req struct {
		Sender  string `json:"sender" binding:"required"`
		Message string `json:"message" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	msg, err := h.service.SendMessage(c.Request.Context(), tradeID, req.Sender, req.Message)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, msg)
}

func (h *Handler) GetMessages(c *gin.Context) {
	tradeID := c.Param("tradeId")
	
	messages, err := h.service.GetMessages(c.Request.Context(), tradeID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, messages)
}

func (h *Handler) GetUser(c *gin.Context) {
	address := c.Param("address")
	
	user, err := h.service.GetUser(c.Request.Context(), address)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, user)
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := &Config{
		Port:            getEnv("PORT", "8080"),
		RedisURL:        getEnv("REDIS_URL", "localhost:6379"),
		ContractAddress: getEnv("CONTRACT_ADDRESS", "0x0000000000000000000000000000000000000000"),
		FeePercent:      0.5,
		DisputeWindow:   24 * time.Hour,
	}

	service, err := NewP2PService(config)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	handler := NewHandler(service)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API routes
	api := router.Group("/api/v1")
	{
		api.POST("/orders", handler.CreateOrder)
		api.GET("/orders", handler.GetOrders)
		api.GET("/orders/:orderId", func(c *gin.Context) {
			orderID := c.Param("orderId")
			order, err := service.GetOrder(c.Request.Context(), orderID)
			if err != nil {
				c.JSON(404, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, order)
		})

		api.POST("/trades", handler.InitiateTrade)
		api.GET("/trades/:tradeId", handler.GetTrade)
		api.POST("/trades/:tradeId/confirm", handler.ConfirmPayment)
		api.POST("/trades/:tradeId/release", handler.ReleaseCrypto)
		api.POST("/trades/:tradeId/dispute", handler.OpenDispute)
		api.GET("/trades/:tradeId/messages", handler.GetMessages)
		api.POST("/trades/:tradeId/messages", handler.SendMessage)

		api.GET("/users/:address", handler.GetUser)
	}

	log.Printf("Starting P2P Trading service on %s", config.Port)
	router.Run(":" + config.Port)
}
