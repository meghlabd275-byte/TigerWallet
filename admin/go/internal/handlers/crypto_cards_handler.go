/**
 * TigerWallet Admin - Crypto Cards Handler
 * Complete backend implementation for crypto card management
 */

package handlers

import (
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CryptoCardHandler struct {
	db *gorm.DB
}

func NewCryptoCardHandler(db *gorm.DB) *CryptoCardHandler {
	return &CryptoCardHandler{db: db}
}

// CryptoCard model
type CryptoCard struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"index" json:"user_id"`
	UserName   string    `json:"user_name"`
	CardNumber string    `gorm:"uniqueIndex" json:"card_number"`
	Currency   string    `json:"currency"`
	Balance    float64   `json:"balance"`
	Limit      float64   `json:"limit"`
	Status     string    `gorm:"default:'pending'" json:"status"` // active, blocked, pending
	CardType   string    `json:"card_type"`                       // virtual, physical
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (CryptoCard) TableName() string {
	return "crypto_cards"
}

// GetAll handles GET /crypto-cards
func (h *CryptoCardHandler) GetAll(c *gin.Context) {
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	query := h.db.Model(&CryptoCard{})

	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var cards []CryptoCard
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&cards).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": cards,
		"meta": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// GetByID handles GET /crypto-cards/:id
func (h *CryptoCardHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid card id"})
		return
	}

	var card CryptoCard
	if err := h.db.First(&card, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "card not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, card)
}

// Create handles POST /crypto-cards
func (h *CryptoCardHandler) Create(c *gin.Context) {
	var card CryptoCard
	if err := c.ShouldBindJSON(&card); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate card number (in production, use proper card generation)
	card.CardNumber = generateCardNumber()
	card.Status = "pending"

	if err := h.db.Create(&card).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, card)
}

// Block handles POST /crypto-cards/:id/block
func (h *CryptoCardHandler) Block(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid card id"})
		return
	}

	result := h.db.Model(&CryptoCard{}).Where("id = ?", id).Update("status", "blocked")
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "card not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "card blocked successfully"})
}

// Activate handles POST /crypto-cards/:id/activate
func (h *CryptoCardHandler) Activate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid card id"})
		return
	}

	result := h.db.Model(&CryptoCard{}).Where("id = ?", id).Update("status", "active")
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "card not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "card activated successfully"})
}

// SetLimit handles PUT /crypto-cards/:id/limit
func (h *CryptoCardHandler) SetLimit(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid card id"})
		return
	}

	var input struct {
		Limit float64 `json:"limit" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := h.db.Model(&CryptoCard{}).Where("id = ?", id).Update("limit", input.Limit)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "card not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "limit updated successfully"})
}

// Helper function to generate card number
func generateCardNumber() string {
	// In production, use proper Luhn algorithm
	return "4532" + randomDigits(12)
}

func randomDigits(n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += strconv.Itoa(int(rand.Intn(10)))
	}
	return result
}
