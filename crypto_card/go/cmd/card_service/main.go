/**
 * TigerWallet Crypto Card Service
 * Go backend for crypto card operations
 * Provides REST API for card management, transactions, and processing
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
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// Configuration
type Config struct {
	ServerPort        string
	RedisAddr         string
	JWTSecret         string
	EncryptionKey     string
	FeeConfiguration  FeeConfig
	Limits            LimitsConfig
}

type FeeConfig struct {
	CardIssueFee          int64
	MonthlyFee            int64
	TransactionFee        float64
	CryptoConversionFee   float64
}

type LimitsConfig struct {
	DailySpendLimit      int64
	MonthlySpendLimit    int64
	MaxTransaction       int64
	MinTransaction       int64
	MaxCardsPerUser      int
}

// Card Types
type CardType string
type CardStatus string
type CardNetwork string
type TransactionType string
type TransactionStatus string

const (
	CardTypeVirtual         CardType = "VIRTUAL"
	CardTypePhysical       CardType = "PHYSICAL"
	CardTypeVirtualOneTime CardType = "VIRTUAL_ONE_TIME"
	CardTypeMetal          CardType = "METAL"

	CardStatusPending   CardStatus = "PENDING"
	CardStatusActive    CardStatus = "ACTIVE"
	CardStatusBlocked   CardStatus = "BLOCKED"
	CardStatusExpired   CardStatus = "EXPIRED"
	CardStatusCancelled CardStatus = "CANCELLED"
	CardStatusFrozen    CardStatus = "FROZEN"

	CardNetworkVisa      CardNetwork = "VISA"
	CardNetworkMastercard CardNetwork = "MASTERCARD"
	CardNetworkAmex      CardNetwork = "AMEX"
	CardNetworkUnionPay  CardNetwork = "UNIONPAY"

	TransactionTypePurchase   TransactionType = "PURCHASE"
	TransactionTypeWithdrawal TransactionType = "WITHDRAWAL"
	TransactionTypeRefund     TransactionType = "REFUND"
	TransactionTypeTransfer   TransactionType = "TRANSFER"
	TransactionTypeTopUp      TransactionType = "TOP_UP"
	TransactionTypeFee        TransactionType = "FEE"

	TransactionStatusPending   TransactionStatus = "PENDING"
	TransactionStatusCompleted TransactionStatus = "COMPLETED"
	TransactionStatusFailed    TransactionStatus = "FAILED"
	TransactionStatusCancelled TransactionStatus = "CANCELLED"
	TransactionStatusFlagged   TransactionStatus = "FLAGGED"
)

// Card Data Models
type CardHolder struct {
	UserID         string `json:"user_id"`
	Name           string `json:"name"`
	Email          string `json:"email"`
	Phone          string `json:"phone"`
	BillingAddress string `json:"billing_address"`
	Country        string `json:"country"`
	City           string `json:"city"`
	PostalCode     string `json:"postal_code"`
	KYCLevel       string `json:"kyc_level"`
	RiskLevel      uint8  `json:"risk_level"`
}

type CardData struct {
	CardID                  string    `json:"card_id"`
	UserID                  string    `json:"user_id"`
	MaskedNumber            string    `json:"masked_number"`
	LastFour                string    `json:"last_four"`
	ExpiryMonth             uint16    `json:"expiry_month"`
	ExpiryYear              uint16    `json:"expiry_year"`
	CardType                CardType  `json:"card_type"`
	Status                  CardStatus `json:"status"`
	Network                 CardNetwork `json:"network"`
	Currency                string    `json:"currency"`
	CardHolderName          string    `json:"card_holder_name"`
	BillingAddress          string    `json:"billing_address"`
	DailyLimit              int64     `json:"daily_limit"`
	MonthlyLimit            int64     `json:"monthly_limit"`
	DailySpent              int64     `json:"daily_spent"`
	MonthlySpent            int64     `json:"monthly_spent"`
	MaxSingleTransaction   int64     `json:"max_single_transaction"`
	MinSingleTransaction   int64     `json:"min_single_transaction"`
	ContactlessEnabled      bool      `json:"contactless_enabled"`
	OnlinePaymentsEnabled  bool      `json:"online_payments_enabled"`
	InternationalEnabled   bool      `json:"international_enabled"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
	ExpiresAt               time.Time `json:"expires_at"`
}

type Transaction struct {
	TransactionID      string            `json:"transaction_id"`
	CardID             string            `json:"card_id"`
	UserID             string            `json:"user_id"`
	Type               TransactionType  `json:"type"`
	Status             TransactionStatus `json:"status"`
	Currency           string            `json:"currency"`
	Amount             int64             `json:"amount"`
	Fee                int64             `json:"fee"`
	CryptoAmount       int64             `json:"crypto_amount,omitempty"`
	CryptoCurrency     string            `json:"crypto_currency,omitempty"`
	MerchantID         string            `json:"merchant_id"`
	MerchantName       string            `json:"merchant_name"`
	MerchantCategory   string            `json:"merchant_category"`
	TerminalID         string            `json:"terminal_id"`
	Location           string            `json:"location"`
	IPAddress          string            `json:"ip_address"`
	Description        string            `json:"description"`
	ReferenceID       string            `json:"reference_id"`
	AuthorizationCode  string            `json:"authorization_code"`
	Timestamp         time.Time         `json:"timestamp"`
	SettledAt         *time.Time        `json:"settled_at,omitempty"`
	BlockchainTxHash   string            `json:"blockchain_tx_hash,omitempty"`
	RiskScore          uint8             `json:"risk_score"`
	RiskReason         string            `json:"risk_reason"`
}

type CardLimits struct {
	DailyLimit            int64 `json:"daily_limit"`
	MonthlyLimit          int64 `json:"monthly_limit"`
	MaxSingleTransaction  int64 `json:"max_single_transaction"`
	MinSingleTransaction  int64 `json:"min_single_transaction"`
	DailyWithdrawalLimit  int64 `json:"daily_withdrawal_limit"`
	MonthlyWithdrawalLimit int64 `json:"monthly_withdrawal_limit"`
}

type CreateCardRequest struct {
	UserID   string      `json:"user_id" binding:"required"`
	CardType CardType    `json:"card_type" binding:"required"`
	Network  CardNetwork `json:"network" binding:"required"`
	Currency string      `json:"currency"`
	Holder   CardHolder  `json:"holder" binding:"required"`
}

type ProcessTransactionRequest struct {
	CardID            string          `json:"card_id" binding:"required"`
	UserID            string          `json:"user_id" binding:"required"`
	Type              TransactionType `json:"type" binding:"required"`
	Amount            int64           `json:"amount" binding:"required"`
	Currency          string          `json:"currency"`
	CryptoCurrency    string          `json:"crypto_currency"`
	MerchantID        string          `json:"merchant_id"`
	MerchantName      string          `json:"merchant_name"`
	MerchantCategory  string          `json:"merchant_category"`
	Location          string          `json:"location"`
	IPAddress         string          `json:"ip_address"`
	Description       string          `json:"description"`
}

type UpdateLimitsRequest struct {
	CardID string     `json:"card_id" binding:"required"`
	Limits CardLimits `json:"limits" binding:"required"`
}

// Card Service
type CardService struct {
	config       Config
	redis        *redis.Client
	cards        map[string]*CardData
	transactions map[string]*Transaction
	mu           sync.RWMutex
}

// NewCardService creates a new card service instance
func NewCardService(cfg Config) *CardService {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: "",
		DB:       0,
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
	}

	return &CardService{
		config:       cfg,
		redis:        redisClient,
		cards:        make(map[string]*CardData),
		transactions: make(map[string]*Transaction),
	}
}

// GenerateCardNumber generates a valid card number
func (s *CardService) GenerateCardNumber(network CardNetwork, userID string) (string, error) {
	var iin string
	switch network {
	case CardNetworkVisa:
		iin = "400000"
	case CardNetworkMastercard:
		iin = "510000"
	case CardNetworkAmex:
		iin = "340000"
	case CardNetworkUnionPay:
		iin = "620000"
	default:
		iin = "400000"
	}

	hash := uuid.NewSHA1(uuid.NameSpaceOID, []byte(userID+time.Now().String()))
	uniquePart := strings.ReplaceAll(hash.String(), "-", "")[0:9]

	partial := iin + uniquePart
	checkDigit := s.calculateLuhnCheckDigit(partial)

	return partial + checkDigit, nil
}

func (s *CardService) calculateLuhnCheckDigit(partial string) string {
	sum := 0
	alternate := false

	for i := len(partial) - 1; i >= 0; i-- {
		n := int(partial[i] - '0')

		if alternate {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}

		sum += n
		alternate = !alternate
	}

	return fmt.Sprintf("%d", (10-(sum%10))%10)
}

func (s *CardService) ValidateCardNumber(cardNumber string) bool {
	if len(cardNumber) != 16 {
		return false
	}

	sum := 0
	alternate := false

	for i := len(cardNumber) - 1; i >= 0; i-- {
		n := int(cardNumber[i] - '0')

		if alternate {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}

		sum += n
		alternate = !alternate
	}

	return sum%10 == 0
}

func (s *CardService) GenerateCVV(cardNumber, expiryMonth, expiryYear string) string {
	data := cardNumber + expiryMonth + expiryYear
	hash := uuid.NewSHA1(uuid.NameSpaceOID, []byte(data))
	return strings.ReplaceAll(hash.String(), "-", "")[0:3]
}

func (s *CardService) CreateCard(req CreateCardRequest) (*CardData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	userCardCount := 0
	for _, card := range s.cards {
		if card.UserID == req.UserID && card.Status != CardStatusCancelled {
			userCardCount++
		}
	}

	if userCardCount >= s.config.Limits.MaxCardsPerUser {
		return nil, fmt.Errorf("maximum cards per user reached")
	}

	cardNumber, err := s.GenerateCardNumber(req.Network, req.UserID)
	if err != nil {
		return nil, err
	}

	if !s.ValidateCardNumber(cardNumber) {
		return nil, fmt.Errorf("invalid card number generated")
	}

	cvv := s.GenerateCVV(cardNumber, "12", "2029")

	now := time.Now()
	card := &CardData{
		CardID:                  "CARD_" + uuid.New().String()[:8],
		UserID:                  req.UserID,
		MaskedNumber:            cardNumber[:4] + " **** **** " + cardNumber[12:],
		LastFour:                cardNumber[12:],
		ExpiryMonth:             12,
		ExpiryYear:              2029,
		CardType:                req.CardType,
		Status:                  CardStatusPending,
		Network:                 req.Network,
		Currency:                req.Currency,
		CardHolderName:          req.Holder.Name,
		BillingAddress:          req.Holder.BillingAddress,
		DailyLimit:              s.config.Limits.DailySpendLimit,
		MonthlyLimit:            s.config.Limits.MonthlySpendLimit,
		DailySpent:              0,
		MonthlySpent:            0,
		MaxSingleTransaction:   s.config.Limits.MaxTransaction,
		MinSingleTransaction:   s.config.Limits.MinTransaction,
		ContactlessEnabled:     true,
		OnlinePaymentsEnabled:  true,
		InternationalEnabled:    true,
		CreatedAt:               now,
		UpdatedAt:               now,
		ExpiresAt:               now.AddDate(3, 0, 0),
	}

	s.cards[card.CardID] = card

	ctx := context.Background()
	cardJSON, _ := json.Marshal(card)
	s.redis.Set(ctx, "card:"+card.CardID, cardJSON, 0)

	return card, nil
}

func (s *CardService) ActivateCard(cardID string) (*CardData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	card, ok := s.cards[cardID]
	if !ok {
		return nil, fmt.Errorf("card not found")
	}

	if card.Status != CardStatusPending {
		return nil, fmt.Errorf("card cannot be activated")
	}

	card.Status = CardStatusActive
	card.UpdatedAt = time.Now()

	ctx := context.Background()
	cardJSON, _ := json.Marshal(card)
	s.redis.Set(ctx, "card:"+card.CardID, cardJSON, 0)

	return card, nil
}

func (s *CardService) BlockCard(cardID string) (*CardData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	card, ok := s.cards[cardID]
	if !ok {
		return nil, fmt.Errorf("card not found")
	}

	card.Status = CardStatusBlocked
	card.UpdatedAt = time.Now()

	ctx := context.Background()
	cardJSON, _ := json.Marshal(card)
	s.redis.Set(ctx, "card:"+card.CardID, cardJSON, 0)

	return card, nil
}

func (s *CardService) FreezeCard(cardID string) (*CardData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	card, ok := s.cards[cardID]
	if !ok {
		return nil, fmt.Errorf("card not found")
	}

	card.Status = CardStatusFrozen
	card.UpdatedAt = time.Now()

	ctx := context.Background()
	cardJSON, _ := json.Marshal(card)
	s.redis.Set(ctx, "card:"+card.CardID, cardJSON, 0)

	return card, nil
}

func (s *CardService) UnfreezeCard(cardID string) (*CardData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	card, ok := s.cards[cardID]
	if !ok {
		return nil, fmt.Errorf("card not found")
	}

	card.Status = CardStatusActive
	card.UpdatedAt = time.Now()

	ctx := context.Background()
	cardJSON, _ := json.Marshal(card)
	s.redis.Set(ctx, "card:"+card.CardID, cardJSON, 0)

	return card, nil
}

func (s *CardService) CancelCard(cardID string) (*CardData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	card, ok := s.cards[cardID]
	if !ok {
		return nil, fmt.Errorf("card not found")
	}

	card.Status = CardStatusCancelled
	card.UpdatedAt = time.Now()

	ctx := context.Background()
	cardJSON, _ := json.Marshal(card)
	s.redis.Set(ctx, "card:"+card.CardID, cardJSON, 0)

	return card, nil
}

func (s *CardService) GetCard(cardID string) (*CardData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	card, ok := s.cards[cardID]
	if !ok {
		return nil, fmt.Errorf("card not found")
	}

	return card, nil
}

func (s *CardService) GetUserCards(userID string) ([]*CardData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var cards []*CardData
	for _, card := range s.cards {
		if card.UserID == userID && card.Status != CardStatusCancelled {
			cards = append(cards, card)
		}
	}

	return cards, nil
}

func (s *CardService) ProcessTransaction(req ProcessTransactionRequest) (*Transaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	card, ok := s.cards[req.CardID]
	if !ok {
		return nil, fmt.Errorf("card not found")
	}

	if card.Status != CardStatusActive {
		return nil, fmt.Errorf("card is not active")
	}

	if req.Amount < card.MinSingleTransaction {
		return nil, fmt.Errorf("amount below minimum transaction limit")
	}

	if req.Amount > card.MaxSingleTransaction {
		return nil, fmt.Errorf("amount exceeds maximum transaction limit")
	}

	if card.DailySpent+req.Amount > card.DailyLimit {
		return nil, fmt.Errorf("daily limit exceeded")
	}

	if card.MonthlySpent+req.Amount > card.MonthlyLimit {
		return nil, fmt.Errorf("monthly limit exceeded")
	}

	fee := int64(float64(req.Amount) * s.config.FeeConfiguration.TransactionFee)

	tx := &Transaction{
		TransactionID:     "TX_" + uuid.New().String()[:12],
		CardID:            req.CardID,
		UserID:            req.UserID,
		Type:              req.Type,
		Status:            TransactionStatusPending,
		Currency:          req.Currency,
		Amount:            req.Amount,
		Fee:               fee,
		MerchantID:        req.MerchantID,
		MerchantName:      req.MerchantName,
		MerchantCategory:  req.MerchantCategory,
		Location:          req.Location,
		IPAddress:         req.IPAddress,
		Description:       req.Description,
		AuthorizationCode: generateAuthorizationCode(),
		Timestamp:         time.Now(),
		RiskScore:         0,
	}

	riskApproved, riskScore, riskReason := s.assessRisk(tx, card)
	tx.RiskScore = riskScore
	tx.RiskReason = riskReason

	if !riskApproved {
		tx.Status = TransactionStatusFlagged
	} else {
		tx.Status = TransactionStatusCompleted
		card.DailySpent += req.Amount
		card.MonthlySpent += req.Amount
		card.UpdatedAt = time.Now()

		ctx := context.Background()
		cardJSON, _ := json.Marshal(card)
		s.redis.Set(ctx, "card:"+card.CardID, cardJSON, 0)
	}

	s.transactions[tx.TransactionID] = tx

	ctx := context.Background()
	txJSON, _ := json.Marshal(tx)
	s.redis.Set(ctx, "tx:"+tx.TransactionID, txJSON, 0)

	return tx, nil
}

func (s *CardService) assessRisk(tx *Transaction, card *CardData) (bool, uint8, string) {
	score := uint8(0)

	if tx.Amount > card.MaxSingleTransaction {
		score += 30
	}

	highRiskCategories := []string{"gambling", "casino", "adult", "weapons"}
	for _, cat := range highRiskCategories {
		if strings.Contains(strings.ToLower(tx.MerchantCategory), cat) {
			score += 40
			break
		}
	}

	if tx.Location == "" {
		score += 15
	}

	if tx.IPAddress == "" {
		score += 10
	}

	if score >= 50 {
		return false, score, "Transaction blocked due to high risk score"
	} else if score >= 25 {
		return true, score, "Manual review recommended"
	}

	return true, score, "Transaction approved"
}

func generateAuthorizationCode() string {
	return fmt.Sprintf("AUTH%08d", time.Now().UnixNano()%100000000)
}

func (s *CardService) GetCardTransactions(cardID string, days int) ([]*Transaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var txs []*Transaction
	since := time.Now().AddDate(0, 0, -days)

	for _, tx := range s.transactions {
		if tx.CardID == cardID && tx.Timestamp.After(since) {
			txs = append(txs, tx)
		}
	}

	return txs, nil
}

func (s *CardService) UpdateCardLimits(cardID string, limits CardLimits) (*CardData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	card, ok := s.cards[cardID]
	if !ok {
		return nil, fmt.Errorf("card not found")
	}

	card.DailyLimit = limits.DailyLimit
	card.MonthlyLimit = limits.MonthlyLimit
	card.MaxSingleTransaction = limits.MaxSingleTransaction
	card.MinSingleTransaction = limits.MinSingleTransaction
	card.UpdatedAt = time.Now()

	ctx := context.Background()
	cardJSON, _ := json.Marshal(card)
	s.redis.Set(ctx, "card:"+card.CardID, cardJSON, 0)

	return card, nil
}

func (s *CardService) GetCryptoRates() map[string]float64 {
	return map[string]float64{
		"USD_BTC":  0.000025,
		"USD_ETH":  0.0004,
		"USD_USDT": 1.0,
		"USD_EUR":  0.92,
		"USD_GBP":  0.79,
	}
}

func (s *CardService) ConvertToCrypto(amount int64, fiatCurrency, cryptoCurrency string) (int64, error) {
	rates := s.GetCryptoRates()

	key := fiatCurrency + "_" + cryptoCurrency
	rate, ok := rates[key]
	if !ok {
		return 0, fmt.Errorf("conversion rate not available for %s to %s", fiatCurrency, cryptoCurrency)
	}

	return int64(float64(amount) * rate * 100000000), nil
}

// Handlers
func (s *CardService) CreateCardHandler(c *gin.Context) {
	var req CreateCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	card, err := s.CreateCard(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, card)
}

func (s *CardService) GetCardHandler(c *gin.Context) {
	cardID := c.Param("id")

	card, err := s.GetCard(cardID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, card)
}

func (s *CardService) GetUserCardsHandler(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	cards, err := s.GetUserCards(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"cards": cards})
}

func (s *CardService) ActivateCardHandler(c *gin.Context) {
	cardID := c.Param("id")

	card, err := s.ActivateCard(cardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, card)
}

func (s *CardService) BlockCardHandler(c *gin.Context) {
	cardID := c.Param("id")

	card, err := s.BlockCard(cardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, card)
}

func (s *CardService) FreezeCardHandler(c *gin.Context) {
	cardID := c.Param("id")

	card, err := s.FreezeCard(cardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, card)
}

func (s *CardService) UnfreezeCardHandler(c *gin.Context) {
	cardID := c.Param("id")

	card, err := s.UnfreezeCard(cardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, card)
}

func (s *CardService) CancelCardHandler(c *gin.Context) {
	cardID := c.Param("id")

	card, err := s.CancelCard(cardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, card)
}

func (s *CardService) ProcessTransactionHandler(c *gin.Context) {
	var req ProcessTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := s.ProcessTransaction(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, tx)
}

func (s *CardService) GetCardTransactionsHandler(c *gin.Context) {
	cardID := c.Param("id")

	days := 30
	if d := c.Query("days"); d != "" {
		fmt.Sscanf(d, "%d", &days)
	}

	txs, err := s.GetCardTransactions(cardID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"transactions": txs})
}

func (s *CardService) UpdateLimitsHandler(c *gin.Context) {
	var req UpdateLimitsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	card, err := s.UpdateCardLimits(req.CardID, req.Limits)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, card)
}

func (s *CardService) GetRatesHandler(c *gin.Context) {
	rates := s.GetCryptoRates()
	c.JSON(http.StatusOK, gin.H{"rates": rates})
}

func (s *CardService) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func (s *CardService) SetupRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/cards")
	api.Use(s.AuthMiddleware())
	{
		api.POST("", s.CreateCardHandler)
		api.GET("", s.GetUserCardsHandler)
		api.GET("/rates", s.GetRatesHandler)
		api.GET("/:id", s.GetCardHandler)
		api.POST("/:id/activate", s.ActivateCardHandler)
		api.POST("/:id/block", s.BlockCardHandler)
		api.POST("/:id/freeze", s.FreezeCardHandler)
		api.POST("/:id/unfreeze", s.UnfreezeCardHandler)
		api.POST("/:id/cancel", s.CancelCardHandler)
		api.GET("/:id/transactions", s.GetCardTransactionsHandler)
		api.PUT("/:id/limits", s.UpdateLimitsHandler)
		api.POST("/transactions", s.ProcessTransactionHandler)
	}
}

func main() {
	cfg := Config{
		ServerPort:   getEnv("CARD_SERVICE_PORT", "8085"),
		RedisAddr:   getEnv("REDIS_ADDR", "localhost:6379"),
		JWTSecret:   getEnv("JWT_SECRET", ""),
		FeeConfiguration: FeeConfig{
			CardIssueFee:          0,
			MonthlyFee:            0,
			TransactionFee:        0.029,
			CryptoConversionFee:   0.01,
		},
		Limits: LimitsConfig{
			DailySpendLimit:     1000000,
			MonthlySpendLimit:   10000000,
			MaxTransaction:      500000,
			MinTransaction:      100,
			MaxCardsPerUser:     10,
		},
	}

	service := NewCardService(cfg)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

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
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "crypto-card-service",
			"timestamp": time.Now().Unix(),
		})
	})

	service.SetupRoutes(r)

	addr := ":" + cfg.ServerPort
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		log.Printf("Starting Crypto Card Service on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}

	log.Println("Server exited")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
