/**
 * TigerWallet Admin - Bots Handler
 * Governance records only — no fund movement. Admins monitor and control
 * bot lifecycle status (start/stop/pause/resume) via governance records.
 * The real bot execution lives in mm_bot_platform/bot_api + TigerBotPlatform.sol.
 */

package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type BotsHandler struct {
	db *gorm.DB
}

func NewBotsHandler(db *gorm.DB) *BotsHandler {
	return &BotsHandler{db: db}
}

// Bot mirrors the admin_bots governance table.
type Bot struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	BotID         string    `gorm:"uniqueIndex;not null" json:"bot_id"`
	Name          string    `gorm:"not null" json:"name"`
	OwnerID       *string   `gorm:"index" json:"owner_id"`
	BotType       string    `gorm:"not null" json:"bot_type"`
	Strategy      string    `json:"strategy"`
	ChainID       *int64    `json:"chain_id"`
	Status        string    `gorm:"not null;default:'active';index" json:"status"`
	Tier          string    `json:"tier"`
	Exchange      string    `json:"exchange"`
	Pair          string    `json:"pair"`
	AllocatedUSD  float64   `gorm:"default:0" json:"allocated_usd"`
	PnlUSD        float64   `gorm:"default:0" json:"pnl_usd"`
	WinRate       float64   `gorm:"default:0" json:"win_rate"`
	TotalTrades   int64     `gorm:"default:0" json:"total_trades"`
	LastActiveAt  *time.Time `json:"last_active_at"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (Bot) TableName() string { return "admin_bots" }

func (h *BotsHandler) List(c *gin.Context) {
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	query := h.db.Model(&Bot{})
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	var items []Bot
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"bots": items})
}

func (h *BotsHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var item Bot
	if err := h.db.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "bot not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"bot": item})
}

func (h *BotsHandler) Create(c *gin.Context) {
	var item Bot
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
	c.JSON(http.StatusCreated, gin.H{"bot": item})
}

func (h *BotsHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var item Bot
	if err := h.db.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "bot not found"})
		return
	}
	var input Bot
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.db.Model(&item).Updates(input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"bot": item})
}

func (h *BotsHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	result := h.db.Delete(&Bot{}, id)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "bot not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "bot deleted"})
}

// UpdateStatus sets bot status (start/stop/pause/resume — governance record only).
func (h *BotsHandler) UpdateStatus(c *gin.Context) {
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
	result := h.db.Model(&Bot{}).Where("id = ?", id).Update("status", req.Status)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "bot not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "status updated", "status": req.Status})
}

// GetStats returns aggregate bot statistics.
func (h *BotsHandler) GetStats(c *gin.Context) {
	var stats struct {
		Total       int64   `json:"total"`
		Active      int64   `json:"active"`
		Paused      int64   `json:"paused"`
		Stopped     int64   `json:"stopped"`
		TotalPnl    float64 `json:"total_pnl"`
		TotalTrades int64   `json:"total_trades"`
	}
	h.db.Model(&Bot{}).Count(&stats.Total)
	h.db.Model(&Bot{}).Where("status = ?", "active").Count(&stats.Active)
	h.db.Model(&Bot{}).Where("status = ?", "paused").Count(&stats.Paused)
	h.db.Model(&Bot{}).Where("status = ?", "stopped").Count(&stats.Stopped)
	h.db.Model(&Bot{}).Select("COALESCE(SUM(pnl_usd), 0)").Scan(&stats.TotalPnl)
	h.db.Model(&Bot{}).Select("COALESCE(SUM(total_trades), 0)").Scan(&stats.TotalTrades)
	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

// BotTier mirrors the admin_bot_tiers governance table.
type BotTier struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"uniqueIndex;not null" json:"name"`
	MaxBots     int       `gorm:"not null;default:1" json:"max_bots"`
	MaxAllocation float64 `gorm:"default:0" json:"max_allocation"`
	FeePercent  float64   `gorm:"default:0" json:"fee_percent"`
	Features    string    `json:"features"`
	Status      string    `gorm:"not null;default:'active'" json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (BotTier) TableName() string { return "admin_bot_tiers" }

func (h *BotsHandler) ListTiers(c *gin.Context) {
	var tiers []BotTier
	if err := h.db.Order("created_at DESC").Find(&tiers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tiers": tiers})
}

func (h *BotsHandler) CreateTier(c *gin.Context) {
	var tier BotTier
	if err := c.ShouldBindJSON(&tier); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if tier.Status == "" {
		tier.Status = "active"
	}
	if err := h.db.Create(&tier).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"tier": tier})
}

func (h *BotsHandler) UpdateTier(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var tier BotTier
	if err := h.db.First(&tier, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tier not found"})
		return
	}
	var input BotTier
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.db.Model(&tier).Updates(input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tier": tier})
}

func (h *BotsHandler) DeleteTier(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	result := h.db.Delete(&BotTier{}, id)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "tier not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "tier deleted"})
}
