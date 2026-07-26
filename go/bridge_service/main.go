// TigerWallet Bridge Service
// Cross-chain bridge for token transfers

package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Config struct {
	Port int
}

var cfg = Config{Port: 8007}

type BridgeTransaction struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	FromChain      string    `json:"from_chain"`
	ToChain        string    `json:"to_chain"`
	Token          string    `json:"token"`
	Amount         string    `json:"amount"`
	Recipient      string    `json:"recipient"`
	Status         string    `json:"status"` // pending, processing, completed, failed
	FromTxHash     string    `json:"from_tx_hash"`
	ToTxHash        string    `json:"to_tx_hash"`
	Fee            string    `json:"fee"`
	EstimatedTime  int       `json:"estimated_time"`
	Timestamp      time.Time `json:"timestamp"`
}

type BridgeService struct {
	transactions map[string]*BridgeTransaction
}

func NewBridgeService() *BridgeService {
	bs := &BridgeService{
		transactions: make(map[string]*BridgeTransaction),
	}
	return bs
}

func (bs *BridgeService) GetQuote(c *gin.Context) {
	var req struct {
		FromChain string `json:"from_chain" binding:"required"`
		ToChain   string `json:"to_chain" binding:"required"`
		Token     string `json:"token" binding:"required"`
		Amount    string `json:"amount" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Calculate bridge fee (0.3% average)
	fee := fmt.Sprintf("%.6f", 0.0)

	c.JSON(http.StatusOK, gin.H{
		"success":         true,
		"from_chain":     req.FromChain,
		"to_chain":       req.ToChain,
		"token":          req.Token,
		"amount":          req.Amount,
		"fee":            fee,
		"estimated_time":  "600", // seconds
		"min_amount":     "10",
		"max_amount":     "100000",
	})
}

func (bs *BridgeService) InitiateTransfer(c *gin.Context) {
	var req struct {
		UserID     string `json:"user_id" binding:"required"`
		FromChain  string `json:"from_chain" binding:"required"`
		ToChain    string `json:"to_chain" binding:"required"`
		Token      string `json:"token" binding:"required"`
		Amount     string `json:"amount" binding:"required"`
		Recipient string `json:"recipient" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx := &BridgeTransaction{
		ID:            uuid.New().String(),
		UserID:        req.UserID,
		FromChain:     req.FromChain,
		ToChain:       req.ToChain,
		Token:         req.Token,
		Amount:        req.Amount,
		Recipient:     req.Recipient,
		Status:        "pending",
		Fee:           "0.3%",
		EstimatedTime: 600,
		Timestamp:     time.Now(),
	}

	bs.transactions[tx.ID] = tx

	c.JSON(http.StatusCreated, gin.H{
		"success":       true,
		"tx_id":        tx.ID,
		"from_chain":   req.FromChain,
		"to_chain":     req.ToChain,
		"amount":       req.Amount,
		"fee":          tx.Fee,
		"status":       tx.Status,
	})
}

func (bs *BridgeService) GetStatus(c *gin.Context) {
	txID := c.Param("id")
	tx, ok := bs.transactions[txID]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "transaction": tx})
}

func main() {
	log.Println("TigerWallet Bridge Service")
	log.Printf("Starting on port %d", cfg.Port)

	bs := NewBridgeService()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "bridge"})
	})

	api := r.Group("/api/v1/bridge")
	{
		api.POST("/quote", bs.GetQuote)
		api.POST("/transfer", bs.InitiateTransfer)
		api.GET("/tx/:id", bs.GetStatus)
	}

	log.Printf("Server starting on :%d", cfg.Port)
	r.Run(fmt.Sprintf(":%d", cfg.Port))
}
