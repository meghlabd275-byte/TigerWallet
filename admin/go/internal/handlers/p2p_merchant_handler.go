/**
 * TigerWallet Admin - P2P Merchant Handler
 * Complete backend implementation for P2P merchant management
 */

package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type P2PMerchantHandler struct {
	db *gorm.DB
}

func NewP2PMerchantHandler(db *gorm.DB) *P2PMerchantHandler {
	return &P2PMerchantHandler{db: db}
}

// P2PMerchant model
type P2PMerchant struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	BusinessName     string    `gorm:"not null" json:"business_name"`
	Email            string    `gorm:"not null" json:"email"`
	Phone            string    `json:"phone"`
	Country          string    `json:"country"`
	Status           string    `gorm:"default:'pending'" json:"status"` // pending, approved, rejected, suspended
	Verified         bool      `gorm:"default:false" json:"verified"`
	TotalVolume      float64   `json:"total_volume"`
	TransactionCount int       `json:"transaction_count"`
	Rating           float64   `gorm:"default:0" json:"rating"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (P2PMerchant) TableName() string {
	return "p2p_merchants"
}

// P2PTransaction model
type P2PTransaction struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	MerchantID    uint       `gorm:"index" json:"merchant_id"`
	BuyerID       uint       `gorm:"index" json:"buyer_id"`
	SellerID      uint       `gorm:"index" json:"seller_id"`
	Amount        float64    `json:"amount"`
	Currency      string     `json:"currency"`
	Status        string     `gorm:"default:'pending'" json:"status"` // pending, completed, cancelled, disputed
	PaymentMethod string     `json:"payment_method"`
	CreatedAt     time.Time  `json:"created_at"`
	CompletedAt   *time.Time `json:"completed_at"`
}

func (P2PTransaction) TableName() string {
	return "p2p_transactions"
}

// GetMerchants handles GET /p2p/merchants
func (h *P2PMerchantHandler) GetMerchants(c *gin.Context) {
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	query := h.db.Model(&P2PMerchant{})

	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	var merchants []P2PMerchant
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&merchants).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, merchants)
}

// GetMerchantByID handles GET /p2p/merchants/:id
func (h *P2PMerchantHandler) GetMerchantByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid merchant id"})
		return
	}

	var merchant P2PMerchant
	if err := h.db.First(&merchant, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "merchant not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, merchant)
}

// ApproveMerchant handles POST /p2p/merchants/:id/approve
func (h *P2PMerchantHandler) ApproveMerchant(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid merchant id"})
		return
	}

	result := h.db.Model(&P2PMerchant{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":   "approved",
		"verified": true,
	})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "merchant not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "merchant approved successfully"})
}

// RejectMerchant handles POST /p2p/merchants/:id/reject
func (h *P2PMerchantHandler) RejectMerchant(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid merchant id"})
		return
	}

	var input struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := h.db.Model(&P2PMerchant{}).Where("id = ?", id).Update("status", "rejected")

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "merchant not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "merchant rejected", "reason": input.Reason})
}

// GetTransactions handles GET /p2p/merchants/:id/transactions
func (h *P2PMerchantHandler) GetTransactions(c *gin.Context) {
	merchantID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid merchant id"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	var transactions []P2PTransaction
	if err := h.db.Where("merchant_id = ?", merchantID).Offset(offset).Limit(limit).Order("created_at DESC").Find(&transactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, transactions)
}

// CreateMerchant handles POST /p2p/merchants
func (h *P2PMerchantHandler) CreateMerchant(c *gin.Context) {
	var merchant P2PMerchant
	if err := c.ShouldBindJSON(&merchant); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	merchant.Status = "pending"
	if err := h.db.Create(&merchant).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, merchant)
}

// UpdateMerchant handles PUT /p2p/merchants/:id
func (h *P2PMerchantHandler) UpdateMerchant(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid merchant id"})
		return
	}

	var merchant P2PMerchant
	if err := h.db.First(&merchant, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "merchant not found"})
		return
	}

	var input P2PMerchant
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.Model(&merchant).Updates(input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, merchant)
}
