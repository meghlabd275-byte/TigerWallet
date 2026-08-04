/**
 * TigerWallet Master Wallet - Admin API Service
 * 
 * This service provides administrative functionality for managing master wallets
 * and is separated from the admin platform to maintain proper architecture boundaries.
 */

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ============================================================================
// Master Wallet Models
// ============================================================================

// MasterWallet represents the main wallet controlled by the platform
type MasterWallet struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	WalletID         string    `gorm:"uniqueIndex" json:"wallet_id"`
	Name             string    `json:"name"`
	Address          string    `gorm:"uniqueIndex" json:"address"`
	ChainID          int64     `json:"chain_id"`
	WalletType       string    `json:"wallet_type"` // hot, cold, warm
	PublicKey        string    `json:"public_key"`
	EncryptedPrivateKey string   `json:"-"` // Never exposed via API
	IsActive          bool      `gorm:"default:true" json:"is_active"`
	Balance           string    `json:"balance"`
	LockedBalance     string    `json:"locked_balance"`
	LastSyncedAt     *time.Time `json:"last_synced_at"`
}

// UserWallet represents user wallets under master wallet
type UserWallet struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	WalletID         string    `gorm:"uniqueIndex" json:"wallet_id"`
	UserID           uint      `gorm:"index" json:"user_id"`
	MasterWalletID   uint      `gorm:"index" json:"master_wallet_id"`
	Address          string    `gorm:"uniqueIndex" json:"address"`
	ChainID          int64     `json:"chain_id"`
	WalletType       string    `json:"wallet_type"` // user, sub
	PublicKey        string    `json:"public_key"`
	IsActive          bool      `gorm:"default:true" json:"is_active"`
	Balance           string    `json:"balance"`
}

// MasterWalletTransaction represents transactions involving master wallet
type MasterWalletTransaction struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt        time.Time `json:"created_at"`
	TxHash           string    `gorm:"uniqueIndex" json:"tx_hash"`
	MasterWalletID   uint      `gorm:"index" json:"master_wallet_id"`
	Type             string    `json:"type"` // deposit, withdrawal, transfer, distribution
	Status           string    `json:"status"` // pending, confirmed, failed
	FromAddress      string    `json:"from_address"`
	ToAddress        string    `json:"to_address"`
	Amount           string    `json:"amount"`
	Token            string    `json:"token"`
	ChainID          int64     `json:"chain_id"`
	Fee              string    `json:"fee"`
	BlockNumber     *int64    `json:"block_number"`
	Confirmations    int       `json:"confirmations"`
}

// DistributionConfig represents profit distribution configuration
type DistributionConfig struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt        time.Time `json:"created_at"`
	ConfigID         string    `gorm:"uniqueIndex" json:"config_id"`
	Name             string    `json:"name"`
	MasterWalletID   uint      `gorm:"index" json:"master_wallet_id"`
	DistributionType string    `json:"distribution_type"` // percentage, fixed
	DistributionPercent float64 `json:"distribution_percent"` // 0-50
	Recipients       JSON      `gorm:"type:jsonb" json:"recipients"`
	IsActive        bool      `gorm:"default:true" json:"is_active"`
	CreatedBy        uint      `json:"created_by"`
}

// ============================================================================
// Master Wallet Handlers
// ============================================================================

// ListMasterWallets returns all master wallets
func (s *MasterWalletService) ListMasterWallets(c *gin.Context) {
	var wallets []MasterWallet
	query := s.db.Model(&MasterWallet{})

	status := c.Query("status")
	chainID := c.Query("chain_id")
	walletType := c.Query("wallet_type")

	if status == "active" {
		query = query.Where("is_active = ?", true)
	} else if status == "inactive" {
		query = query.Where("is_active = ?", false)
	}
	if chainID != "" {
		chain, _ := strconv.ParseInt(chainID, 10, 64)
		query = query.Where("chain_id = ?", chain)
	}
	if walletType != "" {
		query = query.Where("wallet_type = ?", walletType)
	}

	if err := query.Order("created_at DESC").Find(&wallets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch master wallets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": wallets,
		"total": len(wallets),
	})
}

// GetMasterWallet returns a single master wallet
func (s *MasterWalletService) GetMasterWallet(c *gin.Context) {
	walletID := c.Param("id")
	var wallet MasterWallet
	if err := s.db.Where("wallet_id = ?", walletID).First(&wallet).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Master wallet not found"})
		return
	}
	c.JSON(http.StatusOK, wallet)
}

// CreateMasterWallet creates a new master wallet
func (s *MasterWalletService) CreateMasterWallet(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	var wallet MasterWallet
	if err := c.ShouldBindJSON(&wallet); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	wallet.WalletID = "mw_" + uuid.New().String()[:8]
	wallet.IsActive = true

	if err := s.db.Create(&wallet).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create master wallet"})
		return
	}

	logAudit(adminID, "MASTER_WALLET_CREATED", "master_wallet", wallet.WalletID, c.ClientIP(), true, "")
	c.JSON(http.StatusCreated, wallet)
}

// UpdateMasterWallet updates a master wallet
func (s *MasterWalletService) UpdateMasterWallet(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	walletID := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Don't allow updating sensitive fields
	delete(updates, "encrypted_private_key")
	delete(updates, "address")
	delete(updates, "public_key")

	result := s.db.Model(&MasterWallet{}).Where("wallet_id = ?", walletID).Updates(updates)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Master wallet not found"})
		return
	}

	logAudit(adminID, "MASTER_WALLET_UPDATED", "master_wallet", walletID, c.ClientIP(), true, "")
	c.JSON(http.StatusOK, gin.H{"message": "Master wallet updated"})
}

// SyncMasterWalletBalance syncs the balance from blockchain
func (s *MasterWalletService) SyncMasterWalletBalance(c *gin.Context) {
	walletID := c.Param("id")
	
	var wallet MasterWallet
	if err := s.db.Where("wallet_id = ?", walletID).First(&wallet).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Master wallet not found"})
		return
	}

	// In production, this would call the blockchain node to get the actual balance
	// For now, we'll just update the last_synced_at timestamp
	now := time.Now()
	s.db.Model(&wallet).Update("last_synced_at", now)

	c.JSON(http.StatusOK, gin.H{
		"message":       "Balance synced",
		"last_synced_at": now,
	})
}

// ============================================================================
// User Wallet Handlers (Under Master Wallet)
// ============================================================================

// ListUserWallets returns all user wallets under a master wallet
func (s *MasterWalletService) ListUserWallets(c *gin.Context) {
	masterWalletID := c.Param("id")
	
	var masterWallet MasterWallet
	if err := s.db.Where("wallet_id = ?", masterWalletID).First(&masterWallet).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Master wallet not found"})
		return
	}

	var wallets []UserWallet
	if err := s.db.Where("master_wallet_id = ?", masterWalletID).Find(&wallets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user wallets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": wallets,
		"total": len(wallets),
	})
}

// CreateUserWallet creates a new user wallet under a master wallet
func (s *MasterWalletService) CreateUserWallet(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	masterWalletID := c.Param("id")

	var masterWallet MasterWallet
	if err := s.db.Where("wallet_id = ?", masterWalletID).First(&masterWallet).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Master wallet not found"})
		return
	}

	var wallet UserWallet
	if err := c.ShouldBindJSON(&wallet); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	wallet.WalletID = "uw_" + uuid.New().String()[:8]
	wallet.MasterWalletID = masterWallet.ID
	wallet.IsActive = true

	if err := s.db.Create(&wallet).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user wallet"})
		return
	}

	logAudit(adminID, "USER_WALLET_CREATED", "user_wallet", wallet.WalletID, c.ClientIP(), true, "")
	c.JSON(http.StatusCreated, wallet)
}

// ============================================================================
// Transaction Handlers
// ============================================================================

// ListMasterWalletTransactions returns all transactions for a master wallet
func (s *MasterWalletService) ListMasterWalletTransactions(c *gin.Context) {
	walletID := c.Param("id")
	
	var wallet MasterWallet
	if err := s.db.Where("wallet_id = ?", walletID).First(&wallet).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Master wallet not found"})
		return
	}

	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	status := c.Query("status")
	txType := c.Query("type")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var transactions []MasterWalletTransaction
	var total int64

	query := s.db.Model(&MasterWalletTransaction{}).Where("master_wallet_id = ?", wallet.ID)

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if txType != "" {
		query = query.Where("type = ?", txType)
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC")

	if err := query.Find(&transactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        transactions,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

// ============================================================================
// Distribution Config Handlers
// ============================================================================

// ListDistributionConfigs returns all distribution configurations
func (s *MasterWalletService) ListDistributionConfigs(c *gin.Context) {
	masterWalletID := c.Query("master_wallet_id")

	var configs []DistributionConfig
	query := s.db.Model(&DistributionConfig{})

	if masterWalletID != "" {
		query = query.Where("master_wallet_id = ?", masterWalletID)
	}

	if err := query.Order("created_at DESC").Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch distribution configs"})
		return
	}

	c.JSON(http.StatusOK, configs)
}

// CreateDistributionConfig creates a new distribution configuration
func (s *MasterWalletService) CreateDistributionConfig(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	var config DistributionConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Validate distribution percent (0-50)
	if config.DistributionPercent < 0 || config.DistributionPercent > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Distribution percent must be between 0 and 50"})
		return
	}

	config.ConfigID = "dist_" + uuid.New().String()[:8]
	config.IsActive = true
	config.CreatedBy = adminID

	if err := s.db.Create(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create distribution config"})
		return
	}

	logAudit(adminID, "DISTRIBUTION_CONFIG_CREATED", "distribution", config.ConfigID, c.ClientIP(), true, "")
	c.JSON(http.StatusCreated, config)
}

// ExecuteDistribution executes profit distribution
func (s *MasterWalletService) ExecuteDistribution(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	configID := c.Param("id")

	var config DistributionConfig
	if err := s.db.Where("config_id = ?", configID).First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Distribution config not found"})
		return
	}

	if !config.IsActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Distribution config is inactive"})
		return
	}

	// In production, this would:
	// 1. Calculate profits
	// 2. Distribute to recipients based on config
	// 3. Create transaction records
	
	now := time.Now()
	s.db.Model(&config).Update("last_executed_at", now)

	logAudit(adminID, "DISTRIBUTION_EXECUTED", "distribution", configID, c.ClientIP(), true, "")
	c.JSON(http.StatusOK, gin.H{"message": "Distribution executed successfully"})
}

// ============================================================================
// Helper Functions
// ============================================================================

func logAudit(adminID uint, action, resourceType, resourceID, ip string, success bool, details string) {
	// This would create an audit log entry
}

// ============================================================================
// Master Wallet Service
// ============================================================================

type MasterWalletService struct {
	db *gorm.DB
}

func NewMasterWalletService(db *gorm.DB) *MasterWalletService {
	return &MasterWalletService{db: db}
}

// Auto migrate models
func init() {
	// Models would be migrated here
}
