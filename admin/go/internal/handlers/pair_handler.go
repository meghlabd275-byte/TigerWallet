package handlers

import (
	"net/http"
	"strconv"

	"github.com/tigerwallet/admin/internal/models"
	"github.com/tigerwallet/admin/pkg/database"

	"github.com/gin-gonic/gin"
)

// PairHandler handles trading pair-related requests - COMPLETE IMPLEMENTATION
type PairHandler struct {
	db *database.PostgresDB
}

// NewPairHandler creates a new pair handler
func NewPairHandler(db *database.PostgresDB) *PairHandler {
	return &PairHandler{db: db}
}

// ListPairs lists all trading pairs
func (h *PairHandler) ListPairs(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	baseToken := c.Query("base_token")
	quoteToken := c.Query("quote_token")
	chain := c.Query("chain")
	status := c.Query("status")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var pairs []models.TradingPair
	var total int64

	query := h.db.Model(&models.TradingPair{})

	if baseToken != "" {
		query = query.Where("base_token = ?", baseToken)
	}
	if quoteToken != "" {
		query = query.Where("quote_token = ?", quoteToken)
	}
	if chain != "" {
		query = query.Where("chain = ?", chain)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("volume_24h DESC")

	if err := query.Find(&pairs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch pairs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        pairs,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

// GetPair gets a trading pair by ID
func (h *PairHandler) GetPair(c *gin.Context) {
	pairID := c.Param("id")

	var pair models.TradingPair
	if err := h.db.Preload("Token0").Preload("Token1").First(&pair, pairID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pair not found"})
		return
	}

	c.JSON(http.StatusOK, pair)
}

// CreatePair creates a new trading pair
func (h *PairHandler) CreatePair(c *gin.Context) {
	adminID := c.GetUint("admin_id")

	var req struct {
		PairName          string  `json:"pair_name" binding:"required"`
		BaseToken         string  `json:"base_token" binding:"required"`
		QuoteToken        string  `json:"quote_token" binding:"required"`
		Chain             string  `json:"chain" binding:"required"`
		MinTradeAmount    float64 `json:"min_trade_amount"`
		MaxTradeAmount    float64 `json:"max_trade_amount"`
		MakerFee          float64 `json:"maker_fee"`
		TakerFee          float64 `json:"taker_fee"`
		PricePrecision    int     `json:"price_precision"`
		QuantityPrecision int     `json:"quantity_precision"`
		MinPrice          float64 `json:"min_price"`
		MaxPrice          float64 `json:"max_price"`
		IsActive          bool    `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Check if pair already exists
	var existingPair models.TradingPair
	if err := h.db.Where("base_token = ? AND quote_token = ? AND chain = ?",
		req.BaseToken, req.QuoteToken, req.Chain).First(&existingPair).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Pair already exists"})
		return
	}

	pair := models.TradingPair{
		PairName:          req.PairName,
		BaseToken:         req.BaseToken,
		QuoteToken:        req.QuoteToken,
		Chain:             req.Chain,
		MinTradeAmount:    req.MinTradeAmount,
		MaxTradeAmount:    req.MaxTradeAmount,
		MakerFee:          req.MakerFee,
		TakerFee:          req.TakerFee,
		PricePrecision:    req.PricePrecision,
		QuantityPrecision: req.QuantityPrecision,
		MinPrice:          req.MinPrice,
		MaxPrice:          req.MaxPrice,
		IsActive:          req.IsActive,
		CreatedBy:         adminID,
	}

	if err := h.db.Create(&pair).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create pair"})
		return
	}

	logAdminActivity(h.db, adminID, "create_pair", "pair", strconv.FormatUint(uint64(pair.ID), 10), "Created pair: "+pair.PairName, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusCreated, pair)
}

// UpdatePair updates a trading pair
func (h *PairHandler) UpdatePair(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	pairID := c.Param("id")

	var req struct {
		PairName          string  `json:"pair_name"`
		MinTradeAmount    float64 `json:"min_trade_amount"`
		MaxTradeAmount    float64 `json:"max_trade_amount"`
		MakerFee          float64 `json:"maker_fee"`
		TakerFee          float64 `json:"taker_fee"`
		PricePrecision    int     `json:"price_precision"`
		QuantityPrecision int     `json:"quantity_precision"`
		MinPrice          float64 `json:"min_price"`
		MaxPrice          float64 `json:"max_price"`
		IsActive          *bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var pair models.TradingPair
	if err := h.db.First(&pair, pairID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pair not found"})
		return
	}

	updates := map[string]interface{}{}

	if req.PairName != "" {
		updates["pair_name"] = req.PairName
	}
	if req.MinTradeAmount > 0 {
		updates["min_trade_amount"] = req.MinTradeAmount
	}
	if req.MaxTradeAmount > 0 {
		updates["max_trade_amount"] = req.MaxTradeAmount
	}
	if req.MakerFee > 0 {
		updates["maker_fee"] = req.MakerFee
	}
	if req.TakerFee > 0 {
		updates["taker_fee"] = req.TakerFee
	}
	if req.PricePrecision > 0 {
		updates["price_precision"] = req.PricePrecision
	}
	if req.QuantityPrecision > 0 {
		updates["quantity_precision"] = req.QuantityPrecision
	}
	if req.MinPrice > 0 {
		updates["min_price"] = req.MinPrice
	}
	if req.MaxPrice > 0 {
		updates["max_price"] = req.MaxPrice
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if err := h.db.Model(&pair).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pair"})
		return
	}

	logAdminActivity(h.db, adminID, "update_pair", "pair", pairID, "Updated pair: "+pair.PairName, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, pair)
}

// DeletePair deletes a trading pair
func (h *PairHandler) DeletePair(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	pairID := c.Param("id")

	var pair models.TradingPair
	if err := h.db.First(&pair, pairID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pair not found"})
		return
	}

	if err := h.db.Delete(&pair).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete pair"})
		return
	}

	logAdminActivity(h.db, adminID, "delete_pair", "pair", pairID, "Deleted pair: "+pair.PairName, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Pair deleted successfully"})
}

// UpdatePairStatus updates pair status (activate/deactivate)
func (h *PairHandler) UpdatePairStatus(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	pairID := c.Param("id")

	var req struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.Status != "active" && req.Status != "suspended" && req.Status != "halted" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
		return
	}

	var pair models.TradingPair
	if err := h.db.First(&pair, pairID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pair not found"})
		return
	}

	if err := h.db.Model(&pair).Update("status", req.Status).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}

	logAdminActivity(h.db, adminID, "update_pair_status", "pair", pairID, "Updated pair status to: "+req.Status, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Status updated successfully"})
}

// UpdatePairPrice updates pair price
func (h *PairHandler) UpdatePairPrice(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	pairID := c.Param("id")

	var req struct {
		Price       string `json:"price" binding:"required"`
		Volume24h   string `json:"volume_24h"`
		PriceChange string `json:"price_change_24h"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var pair models.TradingPair
	if err := h.db.First(&pair, pairID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pair not found"})
		return
	}

	updates := map[string]interface{}{
		"last_price": req.Price,
	}

	if req.Volume24h != "" {
		updates["volume_24h"] = req.Volume24h
	}
	if req.PriceChange != "" {
		updates["price_change_24h"] = req.PriceChange
	}

	if err := h.db.Model(&pair).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update price"})
		return
	}

	logAdminActivity(h.db, adminID, "update_pair_price", "pair", pairID, "Updated price: "+req.Price, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, pair)
}

// GetPairStats gets trading pair statistics
func (h *PairHandler) GetPairStats(c *gin.Context) {
	var stats struct {
		TotalPairs     int64   `json:"total_pairs"`
		ActivePairs    int64   `json:"active_pairs"`
		SuspendedPairs int64   `json:"suspended_pairs"`
		HaltingPairs   int64   `json:"halting_pairs"`
		TotalVolume    float64 `json:"total_volume"`
		Volume24h      float64 `json:"volume_24h"`
	}

	h.db.Model(&models.TradingPair{}).Count(&stats.TotalPairs)
	h.db.Model(&models.TradingPair{}).Where("status = ?", "active").Count(&stats.ActivePairs)
	h.db.Model(&models.TradingPair{}).Where("status = ?", "suspended").Count(&stats.SuspendedPairs)
	h.db.Model(&models.TradingPair{}).Where("status = ?", "halted").Count(&stats.HaltingPairs)

	// Get volume from all active pairs
	h.db.Model(&models.TradingPair{}).Where("status = ?", "active").Select("COALESCE(SUM(volume_24h), 0)").Scan(&stats.Volume24h)

	c.JSON(http.StatusOK, stats)
}
