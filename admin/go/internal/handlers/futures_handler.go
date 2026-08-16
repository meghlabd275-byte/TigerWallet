/**
 * TigerWallet Admin - Futures Handler
 * Governance records only — no fund movement.
 */

package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type FuturesHandler struct {
	db *gorm.DB
}

func NewFuturesHandler(db *gorm.DB) *FuturesHandler {
	return &FuturesHandler{db: db}
}

// FuturesPosition mirrors the futures_positions table.
type FuturesPosition struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	UserID           *string   `gorm:"index" json:"user_id"`
	Pair             string    `gorm:"not null" json:"pair"`
	Side             string    `gorm:"not null" json:"side"`
	Size             float64   `gorm:"not null;default:0" json:"size"`
	Leverage         float64   `gorm:"not null;default:1" json:"leverage"`
	EntryPrice       float64   `gorm:"not null;default:0" json:"entry_price"`
	LiquidationPrice float64   `gorm:"not null;default:0" json:"liquidation_price"`
	Margin           float64   `gorm:"not null;default:0" json:"margin"`
	Status           string    `gorm:"not null;default:'open'" json:"status"`
	ChainID          *int64    `json:"chain_id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (FuturesPosition) TableName() string { return "futures_positions" }

func (h *FuturesHandler) List(c *gin.Context) {
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	query := h.db.Model(&FuturesPosition{})
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	var items []FuturesPosition
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"positions": items})
}

func (h *FuturesHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var item FuturesPosition
	if err := h.db.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "position not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"position": item})
}

func (h *FuturesHandler) Create(c *gin.Context) {
	var item FuturesPosition
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if item.Status == "" {
		item.Status = "open"
	}
	if err := h.db.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"position": item})
}

func (h *FuturesHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var item FuturesPosition
	if err := h.db.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "position not found"})
		return
	}
	var input FuturesPosition
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.db.Model(&item).Updates(input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"position": item})
}

func (h *FuturesHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	result := h.db.Delete(&FuturesPosition{}, id)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "position not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "position deleted"})
}

// UpdateStatus sets the status of a futures position (governance record only).
func (h *FuturesHandler) UpdateStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result := h.db.Model(&FuturesPosition{}).Where("id = ?", id).Update("status", req.Status)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "position not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "status updated", "status": req.Status})
}
