/**
 * TigerWallet Admin - Liquidity Sources Handler
 * Governance records only — no fund movement. Admins manage external liquidity
 * sources (DEX/CEX connectors, aggregators, market makers) that the platform
 * routes through. The actual routing happens in the downstream services
 * (swap_service, bridge_service, mm_bot_platform/bot_api).
 */

package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type LiquiditySourcesHandler struct {
	db *gorm.DB
}

func NewLiquiditySourcesHandler(db *gorm.DB) *LiquiditySourcesHandler {
	return &LiquiditySourcesHandler{db: db}
}

// LiquiditySource mirrors the admin_liquidity_sources governance table.
type LiquiditySource struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	SourceID      string    `gorm:"uniqueIndex;not null" json:"source_id"`
	Name          string    `gorm:"not null" json:"name"`
	SourceType    string    `gorm:"not null" json:"source_type"` // dex, cex, aggregator, market_maker
	ChainID       *int64    `gorm:"index" json:"chain_id"`
	RouterAddress string    `json:"router_address"`
	APIEndpoint   string    `json:"api_endpoint"`
	APIKeyID      string    `json:"api_key_id"`
	Status        string    `gorm:"not null;default:'active';index" json:"status"`
	Priority      int       `gorm:"default:0" json:"priority"`
	FeeBps        float64   `gorm:"default:0" json:"fee_bps"`
	SlippageBps   float64   `gorm:"default:0" json:"slippage_bps"`
	MaxCapacity   float64   `gorm:"default:0" json:"max_capacity"`
	CurrentLiquidity float64 `gorm:"default:0" json:"current_liquidity"`
	SupportedPairs string   `json:"supported_pairs"`
	Metadata      string    `json:"metadata"`
	LastHealthCheck *time.Time `json:"last_health_check"`
	HealthStatus  string    `gorm:"default:'unknown'" json:"health_status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (LiquiditySource) TableName() string { return "admin_liquidity_sources" }

func (h *LiquiditySourcesHandler) List(c *gin.Context) {
	status := c.Query("status")
	sourceType := c.Query("type")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	query := h.db.Model(&LiquiditySource{})
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}
	if sourceType != "" && sourceType != "all" {
		query = query.Where("source_type = ?", sourceType)
	}

	var items []LiquiditySource
	if err := query.Offset(offset).Limit(limit).Order("priority DESC, created_at DESC").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"liquidity_sources": items})
}

func (h *LiquiditySourcesHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var item LiquiditySource
	if err := h.db.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "liquidity source not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"liquidity_source": item})
}

func (h *LiquiditySourcesHandler) Create(c *gin.Context) {
	var item LiquiditySource
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if item.Status == "" {
		item.Status = "active"
	}
	if err := h.db.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"liquidity_source": item})
}

func (h *LiquiditySourcesHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var item LiquiditySource
	if err := h.db.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "liquidity source not found"})
		return
	}
	var input LiquiditySource
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.db.Model(&item).Updates(input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"liquidity_source": item})
}

func (h *LiquiditySourcesHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	result := h.db.Delete(&LiquiditySource{}, id)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "liquidity source not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "liquidity source deleted"})
}

// UpdateStatus sets liquidity-source status (start/stop/pause/resume — governance record only).
func (h *LiquiditySourcesHandler) UpdateStatus(c *gin.Context) {
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
	result := h.db.Model(&LiquiditySource{}).Where("id = ?", id).Update("status", req.Status)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "liquidity source not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "status updated", "status": req.Status})
}

// UpdatePriority sets the routing priority of a liquidity source.
func (h *LiquiditySourcesHandler) UpdatePriority(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Priority int `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result := h.db.Model(&LiquiditySource{}).Where("id = ?", id).Update("priority", req.Priority)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "liquidity source not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "priority updated", "priority": req.Priority})
}

// GetStats returns aggregate liquidity-source statistics.
func (h *LiquiditySourcesHandler) GetStats(c *gin.Context) {
	var stats struct {
		Total             int64   `json:"total"`
		Active            int64   `json:"active"`
		Paused            int64   `json:"paused"`
		Stopped           int64   `json:"stopped"`
		TotalLiquidity    float64 `json:"total_liquidity"`
		DEXCount          int64   `json:"dex_count"`
		CEXCount          int64   `json:"cex_count"`
		AggregatorCount   int64   `json:"aggregator_count"`
		MarketMakerCount  int64   `json:"market_maker_count"`
	}
	h.db.Model(&LiquiditySource{}).Count(&stats.Total)
	h.db.Model(&LiquiditySource{}).Where("status = ?", "active").Count(&stats.Active)
	h.db.Model(&LiquiditySource{}).Where("status = ?", "paused").Count(&stats.Paused)
	h.db.Model(&LiquiditySource{}).Where("status = ?", "stopped").Count(&stats.Stopped)
	h.db.Model(&LiquiditySource{}).Select("COALESCE(SUM(current_liquidity), 0)").Scan(&stats.TotalLiquidity)
	h.db.Model(&LiquiditySource{}).Where("source_type = ?", "dex").Count(&stats.DEXCount)
	h.db.Model(&LiquiditySource{}).Where("source_type = ?", "cex").Count(&stats.CEXCount)
	h.db.Model(&LiquiditySource{}).Where("source_type = ?", "aggregator").Count(&stats.AggregatorCount)
	h.db.Model(&LiquiditySource{}).Where("source_type = ?", "market_maker").Count(&stats.MarketMakerCount)
	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

// HealthCheck records a health-check result for a liquidity source.
func (h *LiquiditySourcesHandler) HealthCheck(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		HealthStatus string `json:"health_status"`
		Liquidity    float64 `json:"liquidity"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	now := time.Now()
	updates := map[string]interface{}{
		"health_status":     req.HealthStatus,
		"last_health_check": now,
	}
	if req.Liquidity > 0 {
		updates["current_liquidity"] = req.Liquidity
	}
	result := h.db.Model(&LiquiditySource{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "liquidity source not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "health check recorded", "health_status": req.HealthStatus, "checked_at": now})
}
