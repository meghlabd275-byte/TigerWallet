// TigerWallet Fiat On-Ramp Service
// Fiat gateway for buying crypto

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

var cfg = Config{Port: 8008}

type FiatOrder struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	FiatAmount    string    `json:"fiat_amount"`
	FiatCurrency  string    `json:"fiat_currency"`
	CryptoAmount string    `json:"crypto_amount"`
	CryptoToken  string    `json:"crypto_token"`
	Chain        string    `json:"chain"`
	Recipient    string    `json:"recipient"`
	Provider     string    `json:"provider"` // stripe, moonpay, simplic
	Status       string    `json:"status"` // pending, processing, completed, failed
	PaymentURL   string    `json:"payment_url"`
	TxHash       string    `json:"tx_hash"`
	CreatedAt    time.Time `json:"created_at"`
}

type FiatService struct {
	orders map[string]*FiatOrder
}

func NewFiatService() *FiatService {
	fs := &FiatService{
		orders: make(map[string]*FiatOrder),
	}
	return fs
}

func (fs *FiatService) GetQuote(c *gin.Context) {
	var req struct {
		Amount     string `json:"amount" binding:"required"`
		Currency   string `json:"currency" binding:"required"`
		Token      string `json:"token" binding:"required"`
		Chain      string `json:"chain" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Simulate quote (in production, call provider APIs)
	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"fiat_amount":   req.Amount,
		"fiat_currency": req.Currency,
		"crypto_token":  req.Token,
		"crypto_amount": fmt.Sprintf("%.6f", 0.0), // Would calculate
		"chain":         req.Chain,
		"fee":           "2.5%",
		"provider":      "stripe",
	})
}

func (fs *FiatService) CreateOrder(c *gin.Context) {
	var req struct {
		UserID       string `json:"user_id" binding:"required"`
		FiatAmount   string `json:"fiat_amount" binding:"required"`
		FiatCurrency string `json:"fiat_currency" binding:"required"`
		CryptoToken  string `json:"crypto_token" binding:"required"`
		Chain       string `json:"chain" binding:"required"`
		Recipient   string `json:"recipient" binding:"required"`
		Provider    string `json:"provider"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order := &FiatOrder{
		ID:            uuid.New().String(),
		UserID:        req.UserID,
		FiatAmount:    req.FiatAmount,
		FiatCurrency:  req.FiatCurrency,
		CryptoToken:   req.CryptoToken,
		Chain:         req.Chain,
		Recipient:     req.Recipient,
		Status:        "pending",
		PaymentURL:    "https://pay.stripe.com/" + uuid.New().String(),
		Provider:      req.Provider,
		CreatedAt:     time.Now(),
	}

	fs.orders[order.ID] = order

	c.JSON(http.StatusCreated, gin.H{
		"success":     true,
		"order_id":   order.ID,
		"status":     order.Status,
		"payment_url": order.PaymentURL,
	})
}

func (fs *FiatService) GetOrder(c *gin.Context) {
	orderID := c.Param("id")
	order, ok := fs.orders[orderID]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "order": order})
}

func (fs *FiatService) GetSupportedFiat(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"currencies": []string{"USD", "EUR", "GBP", "AUD", "CAD", "JPY"},
		"providers": []string{"stripe", "moonpay", "simplic"},
	})
}

func (fs *FiatService) GetSupportedCrypto(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"tokens": []map[string]string{
			{"symbol": "ETH", "name": "Ethereum", "networks": "ethereum,polygon,arbitrum"},
			{"symbol": "BTC", "name": "Bitcoin", "networks": "bitcoin"},
			{"symbol": "USDT", "name": "Tether", "networks": "ethereum,polygon,tron"},
			{"symbol": "USDC", "name": "USD Coin", "networks": "ethereum,polygon,arbitrum"},
			{"symbol": "MATIC", "name": "Polygon", "networks": "polygon"},
		},
	})
}

func main() {
	log.Println("TigerWallet Fiat On-Ramp Service")
	log.Printf("Starting on port %d", cfg.Port)

	fs := NewFiatService()

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
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "fiat-onramp"})
	})

	api := r.Group("/api/v1/fiat")
	{
		api.POST("/quote", fs.GetQuote)
		api.POST("/order", fs.CreateOrder)
		api.GET("/order/:id", fs.GetOrder)
		api.GET("/currencies", fs.GetSupportedFiat)
		api.GET("/crypto", fs.GetSupportedCrypto)
	}

	log.Printf("Server starting on :%d", cfg.Port)
	r.Run(fmt.Sprintf(":%d", cfg.Port))
}
