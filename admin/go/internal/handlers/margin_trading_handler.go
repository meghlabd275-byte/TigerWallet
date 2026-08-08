/**
 * TigerWallet Admin - Margin Trading Handler
 * Complete backend implementation for margin trading management
 */

package handlers

import (
	"net/http"
	"strconv"
	"time"

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
	ID               uint       `gorm:"primaryKey" json:"id"`
	UserID           uint       `gorm:"index" json:"user_id"`
	UserName         string     `json:"user_name"`
	Pair             string     `json:"pair"`
	Side             string     `json:"side"` // long, short
	Size             float64    `json:"size"`
	Leverage         int        `json:"leverage"`
	EntryPrice       float64    `json:"entry_price"`
	CurrentPrice     float64    `json:"current_price"`
	PnL              float64    `json:"pnl"`
	LiquidationPrice float64    `json:"liquidation_price"`
	Status           string     `gorm:"default:'open'" json:"status"` // open, liquidated, closed
	OpenedAt         time.Time  `json:"opened_at"`
	ClosedAt         *time.Time `json:"closed_at"`
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
		"total_positions":    totalPositions,
		"total_volume":       totalVolume,
		"liquidations_today": liquidationsToday,
		"liquidated_volume":  liquidatedVolume,
	})
}

// UpdatePrices handles POST /margin/update-prices
func (h *MarginTradingHandler) UpdatePrices(c *gin.Context) {
	var input struct {
		Pair  string  `json:"pair" binding:"required"`
		Price float64 `json:"price" binding:"required"`
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

// OpenPosition opens a new margin trading position.
func (h *MarginTradingHandler) OpenPosition(c *gin.Context) {
	var input struct {
		UserID           uint    `json:"user_id" binding:"required"`
		UserName         string  `json:"user_name"`
		Pair             string  `json:"pair" binding:"required"`
		Side             string  `json:"side" binding:"required"`
		Size             float64 `json:"size" binding:"required"`
		Leverage         int     `json:"leverage" binding:"required"`
		EntryPrice       float64 `json:"entry_price" binding:"required"`
		LiquidationPrice float64 `json:"liquidation_price"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Side != "long" && input.Side != "short" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "side must be long or short"})
		return
	}
	if input.Leverage < 1 || input.Leverage > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "leverage must be between 1 and 100"})
		return
	}

	position := MarginPosition{
		UserID:           input.UserID,
		UserName:         input.UserName,
		Pair:             input.Pair,
		Side:             input.Side,
		Size:             input.Size,
		Leverage:         input.Leverage,
		EntryPrice:       input.EntryPrice,
		CurrentPrice:     input.EntryPrice,
		LiquidationPrice: input.LiquidationPrice,
		Status:           "open",
		OpenedAt:         time.Now(),
	}

	if err := h.db.Create(&position).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, position)
}

// ClosePosition closes an existing margin trading position.
func (h *MarginTradingHandler) ClosePosition(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid position id"})
		return
	}

	var input struct {
		ExitPrice float64 `json:"exit_price" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var position MarginPosition
	if err := h.db.Where("id = ? AND status = ?", id, "open").First(&position).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "open position not found"})
		return
	}

	var pnl float64
	if position.Side == "long" {
		pnl = (input.ExitPrice - position.EntryPrice) * position.Size
	} else {
		pnl = (position.EntryPrice - input.ExitPrice) * position.Size
	}

	now := time.Now()
	h.db.Model(&position).Updates(map[string]interface{}{
		"status":        "closed",
		"current_price": input.ExitPrice,
		"pnl":           pnl,
		"closed_at":     now,
	})

	c.JSON(http.StatusOK, gin.H{"message": "position closed", "pnl": pnl})
}

// UpdateLiquidationPrice updates the liquidation price of a position.
func (h *MarginTradingHandler) UpdateLiquidationPrice(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid position id"})
		return
	}

	var input struct {
		LiquidationPrice float64 `json:"liquidation_price" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := h.db.Model(&MarginPosition{}).Where("id = ? AND status = ?", id, "open").Update("liquidation_price", input.LiquidationPrice)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "open position not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "liquidation price updated"})
}

// GetStats returns margin trading statistics.
func (h *MarginTradingHandler) GetStats(c *gin.Context) {
	var openPositions int64
	var closedPositions int64
	var liquidatedPositions int64
	var totalPnL float64

	h.db.Model(&MarginPosition{}).Where("status = ?", "open").Count(&openPositions)
	h.db.Model(&MarginPosition{}).Where("status = ?", "closed").Count(&closedPositions)
	h.db.Model(&MarginPosition{}).Where("status = ?", "liquidated").Count(&liquidatedPositions)
	h.db.Model(&MarginPosition{}).Where("status = ?", "closed").Select("COALESCE(SUM(pnl), 0)").Scan(&totalPnL)

	c.JSON(http.StatusOK, gin.H{
		"open_positions":       openPositions,
		"closed_positions":     closedPositions,
		"liquidated_positions": liquidatedPositions,
		"total_pnl":            totalPnL,
	})
}
