package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	Port        string
	RedisURL     string
	CardNetwork string
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

type Card struct {
	ID            string    `json:"id"`
	User          string    `json:"user"`
	Type         string    `json:"type"` // virtual, physical
	Status       string    `json:"status"` // active, blocked, expired
	CardNumber   string    `json:"cardNumber"`
	CVV          string    `json:"cvv"`
	ExpiryMonth  int       `json:"expiryMonth"`
	ExpiryYear   int       `json:"expiryYear"`
	Balance      float64   `json:"balance"`
	Limit        float64   `json:"limit"`
	Currency     string    `json:"currency"`
	Network      string    `json:"network"` // visa, mastercard
	CreatedAt    time.Time `json:"createdAt"`
	BlockedAt   *time.Time `json:"blockedAt"`
}

type Transaction struct {
	ID          string    `json:"id"`
	CardID      string    `json:"cardId"`
	User        string    `json:"user"`
	Amount      float64   `json:"amount"`
	Currency    string    `json:"currency"`
	Type        string    `json:"type"` // purchase, withdrawal, refund
	Merchant   string    `json:"merchant"`
	Status     string    `json:"status"` // pending, completed, failed
	Location   string    `json:"location"`
	CreatedAt   time.Time `json:"createdAt"`
}

type PaymentService struct {
	config *Config
	redis  *redis.Client
}

func NewPaymentService(cfg *Config) (*PaymentService, error) {
	r := redis.NewClient(&redis.Options{Addr: cfg.RedisURL, Password: "", DB: 0})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := r.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return &PaymentService{config: cfg, redis: r}, nil
}

func (s *PaymentService) CreateCard(ctx context.Context, user, cardType string) (*Card, error) {
	card := &Card{
		ID:            uuid.New().String(),
		User:          user,
		Type:          cardType,
		Status:        "active",
		CardNumber:    generateCardNumber(),
		CVV:           generateCVV(),
		ExpiryMonth:   int(time.Now().Month()),
		ExpiryYear:    int(time.Now().Year() + 3),
		Balance:       0,
		Limit:        10000,
		Currency:      "USD",
		Network:      "visa",
		CreatedAt:    time.Now(),
	}

	data, _ := json.Marshal(card)
	s.redis.Set(ctx, "payment:card:"+card.ID, data, 0)
	s.redis.SAdd(ctx, "payment:user:"+user, card.ID)

	return card, nil
}

func (s *PaymentService) GetCards(ctx context.Context, user string) []Card {
	ids, _ := s.redis.SMembers(ctx, "payment:user:"+user).Result()
	var cards []Card
	for _, id := range ids {
		data, _ := s.redis.Get(ctx, "payment:card:"+id).Bytes()
		var card Card
		json.Unmarshal(data, &card)
		cards = append(cards, card)
	}
	return cards
}

func (s *PaymentService) ProcessPayment(ctx context.Context, cardID, merchant string, amount float64) (*Transaction, error) {
	cardData, _ := s.redis.Get(ctx, "payment:card:"+cardID).Bytes()
	var card Card
	json.Unmarshal(cardData, &card)

	if card.Status != "active" {
		return nil, fmt.Errorf("card is not active")
	}

	if card.Balance+amount > card.Limit {
		return nil, fmt.Errorf("insufficient limit")
	}

	tx := &Transaction{
		ID:        uuid.New().String(),
		CardID:    cardID,
		User:      card.User,
		Amount:    amount,
		Currency:  card.Currency,
		Type:      "purchase",
		Status:    "completed",
		Merchant: merchant,
		CreatedAt: time.Now(),
	}

	card.Balance += amount
	cardData, _ = json.Marshal(card)
	s.redis.Set(ctx, "payment:card:"+cardID, cardData, 0)

	txData, _ := json.Marshal(tx)
	s.redis.Set(ctx, "payment:tx:"+tx.ID, txData, 0)

	return tx, nil
}

func (s *PaymentService) BlockCard(ctx context.Context, cardID string) error {
	cardData, _ := s.redis.Get(ctx, "payment:card:"+cardID).Bytes()
	var card Card
	json.Unmarshal(cardData, &card)

	now := time.Now()
	card.Status = "blocked"
	card.BlockedAt = &now

	cardData, _ = json.Marshal(card)
	s.redis.Set(ctx, "payment:card:"+cardID, cardData, 0)
	return nil
}

func generateCardNumber() string {
	return "4532" + fmt.Sprintf("%012d", time.Now().UnixNano()%1000000000000)
}

func generateCVV() string {
	return fmt.Sprintf("%03d", time.Now().UnixNano()%1000)
}

func main() {
	cfg := &Config{
		Port:        getEnv("PORT", "8080"),
		RedisURL:    getEnv("REDIS_URL", "localhost:6379"),
		CardNetwork: "visa",
	}

	svc, _ := NewPaymentService(cfg)

	r := gin.Default()
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	api := r.Group("/api/v1")
	{
		api.POST("/cards", func(c *gin.Context) {
			var req struct {
				User string `json:"user"`
				Type string `json:"type"`
			}
			c.ShouldBindJSON(&req)
			card, _ := svc.CreateCard(c.Request.Context(), req.User, req.Type)
			c.JSON(200, card)
		})
		api.GET("/cards/:user", func(c *gin.Context) {
			user := c.Param("user")
			c.JSON(200, svc.GetCards(c.Request.Context(), user))
		})
		api.POST("/pay", func(c *gin.Context) {
			var req struct {
				CardID   string  `json:"cardId"`
				Merchant string  `json:"merchant"`
				Amount   float64 `json:"amount"`
			}
			c.ShouldBindJSON(&req)
			tx, err := svc.ProcessPayment(c.Request.Context(), req.CardID, req.Merchant, req.Amount)
			if err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, tx)
		})
		api.POST("/cards/:id/block", func(c *gin.Context) {
			cardID := c.Param("id")
			svc.BlockCard(c.Request.Context(), cardID)
			c.JSON(200, gin.H{"status": "blocked"})
		})
	}

	log.Printf("Starting Payment service on %s", cfg.Port)
	r.Run(":" + cfg.Port)
}
