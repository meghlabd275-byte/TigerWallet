/**
 * TigerWallet Admin - Bots Clients Handler
 * Governance records only — no fund movement. Admins monitor and control
 * bot-client connections (the client apps that connect to bot instances).
 */

package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type BotsClientsHandler struct {
	db *gorm.DB
}

func NewBotsClientsHandler(db *gorm.DB) *BotsClientsHandler {
	return &BotsClientsHandler{db: db}
}

// BotsClient mirrors the admin_bots_clients governance table.
type BotsClient struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	ClientID      string     `gorm:"uniqueIndex;not null" json:"client_id"`
	BotID         string     `gorm:"index;not null" json:"bot_id"`
	UserID        *string    `gorm:"index" json:"user_id"`
	ClientName    string     `json:"client_name"`
	ClientType    string     `json:"client_type"`
	APIKeyID      string     `json:"api_key_id"`
	Status        string     `gorm:"not null;default:'active';index" json:"status"`
	ConnectedAt   *time.Time `json:"connected_at"`
	LastSeenAt    *time.Time `json:"last_seen_at"`
	AllocatedUSD  float64    `gorm:"default:0" json:"allocated_usd"`
	PnlUSD        float64    `gorm:"default:0" json:"pnl_usd"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (BotsClient) TableName() string { return "admin_bots_clients" }

func (h *BotsClientsHandler) List(c *gin.Context) {
	status := c.Query("status")
	botID := c.Query("bot_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	query := h.db.Model(&BotsClient{})
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}
	if botID != "" {
		query = query.Where("bot_id = ?", botID)
	}

	var items []BotsClient
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"bots_clients": items})
}

func (h *BotsClientsHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var item BotsClient
	if err := h.db.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "bots client not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"bots_client": item})
}

func (h *BotsClientsHandler) Create(c *gin.Context) {
	var item BotsClient
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
	c.JSON(http.StatusCreated, gin.H{"bots_client": item})
}

func (h *BotsClientsHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var item BotsClient
	if err := h.db.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "bots client not found"})
		return
	}
	var input BotsClient
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.db.Model(&item).Updates(input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"bots_client": item})
}

func (h *BotsClientsHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	result := h.db.Delete(&BotsClient{}, id)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "bots client not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "bots client deleted"})
}

// UpdateStatus sets bots-client status (start/stop/pause/resume — governance record only).
func (h *BotsClientsHandler) UpdateStatus(c *gin.Context) {
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
	result := h.db.Model(&BotsClient{}).Where("id = ?", id).Update("status", req.Status)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "bots client not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "status updated", "status": req.Status})
}
