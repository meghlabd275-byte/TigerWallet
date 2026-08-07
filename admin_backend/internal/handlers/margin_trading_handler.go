/**
 * TigerWallet Admin - Margin Trading Handler
 * Complete backend implementation for margin trading management
 */

package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MarginTradingHandler struct {
	db *gorm.DB
}

func NewMarginTradingHandler(db *gorm.DB) *MarginTradingHandler {
	return &MarginTradingHandler{db: db}
}

// MarginPosition model
type MarginPosition struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	UserID          uint      `gorm:"index" json:"user_id"`
	UserName        string    `json:"user_name"`
	Pair            string    `json:"pair"`
	Side            string    `json:"side"` // long, short
	Size            float64   `json:"size"`
	Leverage        int       `json:"leverage"`
	EntryPrice      float64   `json:"entry_price"`
	CurrentPrice    float64   `json:"current_price"`
	PnL             float64   `json:"pnl"`
	LiquidationPrice float64  `json:"liquidation_price"`
	Status          string    `gorm:"default:'open'" json:"status"` // open, liquidated, closed
	OpenedAt        time.Time `json:"opened_at"`
	ClosedAt        *time.Time `json:"closed_at"`
}

func (MarginPosition) TableName() string {
	return "margin_positions"
}

// GetPositions handles GET /margin/positions
func (h *MarginTradingHandler) GetPositions(c *gin.Context) {
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	query := h.db.Model(&MarginPosition{})

	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	var positions []MarginPosition
	if err := query.Offset(offset).Limit(limit).Order("opened_at DESC").Find(&positions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, positions)
}

// GetHistory handles GET /margin/history
func (h *MarginTradingHandler) GetHistory(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	var positions []MarginPosition
	if err := h.db.Where("status != ?", "open").Offset(offset).Limit(limit).Order("closed_at DESC").Find(&positions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, positions)
}

// Liquidate handles POST /margin/positions/:id/liquidate
func (h *MarginTradingHandler) Liquidate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid position id"})
		return
	}

	now := time.Now()
	result := h.db.Model(&MarginPosition{}).Where("id = ? AND status = ?", id, "open").Updates(map[string]interface{}{
		"status":    "liquidated",
		"closed_at": now,
	})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "position not found or already closed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "position liquidated successfully"})
}

// GetLiquidationStats handles GET /margin/liquidation-stats
func (h *MarginTradingHandler) GetLiquidationStats(c *gin.Context) {
	var totalPositions int64
	var totalVolume float64
	var liquidationsToday int64
	var liquidatedVolume float64

	h.db.Model(&MarginPosition{}).Count(&totalPositions)
	h.db.Model(&MarginPosition{}).Where("status = ?", "open").Select("COALESCE(SUM(size * entry_price), 0)").Scan(&totalVolume)

	today := time.Now().Truncate(24 * time.Hour)
	h.db.Model(&MarginPosition{}).Where("status = ? AND closed_at >= ?", "liquidated", today).Count(&liquidationsToday)
	h.db.Model(&MarginPosition{}).Where("status = ? AND closed_at >= ?", "liquidated", today).Select("COALESCE(SUM(size * entry_price), 0)").Scan(&liquidatedVolume)

	c.JSON(http.StatusOK, gin.H{
		"total_positions":      totalPositions,
		"total_volume":        totalVolume,
		"liquidations_today": liquidationsToday,
		"liquidated_volume":   liquidatedVolume,
	})
}

// UpdatePrices handles POST /margin/update-prices
func (h *MarginTradingHandler) UpdatePrices(c *gin.Context) {
	var input struct {
		Pair   string  `json:"pair" binding:"required"`
		Price  float64 `json:"price" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update current prices and calculate PnL for all open positions
	var positions []MarginPosition
	h.db.Where("pair = ? AND status = ?", input.Pair, "open").Find(&positions)

	for _, pos := range positions {
		var pnl float64
		if pos.Side == "long" {
			pnl = (input.Price - pos.EntryPrice) * pos.Size
		} else {
			pnl = (pos.EntryPrice - input.Price) * pos.Size
		}

		h.db.Model(&pos).Updates(map[string]interface{}{
			"current_price": input.Price,
			"pnl":           pnl,
		})
	}

	c.JSON(http.StatusOK, gin.H{"message": "prices updated successfully"})
}
