package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/tigerwallet/admin/internal/models"
	"github.com/tigerwallet/admin/pkg/auth"
	"github.com/tigerwallet/admin/pkg/database"

	"github.com/gin-gonic/gin"
)

// TokenHandler handles token-related requests
type TokenHandler struct {
	db      *database.PostgresDB
	authSvc *auth.AuthService
}

// NewTokenHandler creates a new token handler
func NewTokenHandler(db *database.PostgresDB, authSvc *auth.AuthService) *TokenHandler {
	return &TokenHandler{
		db:      db,
		authSvc: authSvc,
	}
}

// ListTokens lists all tokens
func (h *TokenHandler) ListTokens(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	search := c.Query("search")
	chain := c.Query("chain")
	status := c.Query("status")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var tokens []models.Token
	var total int64

	query := h.db.Model(&models.Token{})

	if search != "" {
		query = query.Where("name ILIKE ? OR symbol ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if chain != "" {
		query = query.Where("chain = ?", chain)
	}
	if status != "" {
		if status == "active" {
			query = query.Where("is_active = ?", true)
		} else if status == "inactive" {
			query = query.Where("is_active = ?", false)
		}
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC")

	if err := query.Find(&tokens).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tokens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        tokens,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

// GetToken gets a token by ID
func (h *TokenHandler) GetToken(c *gin.Context) {
	tokenID := c.Param("id")

	var token models.Token
	if err := h.db.First(&token, tokenID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
		return
	}

	c.JSON(http.StatusOK, token)
}

// CreateToken creates a new token
func (h *TokenHandler) CreateToken(c *gin.Context) {
	adminID := c.GetUint("admin_id")

	var req struct {
		Name            string `json:"name" binding:"required"`
		Symbol          string `json:"symbol" binding:"required"`
		ContractAddress string `json:"contract_address"`
		Chain           string `json:"chain" binding:"required"`
		Decimals        int    `json:"decimals"`
		TotalSupply     string `json:"total_supply"`
		LogoURL         string `json:"logo_url"`
		Website         string `json:"website"`
		Description     string `json:"description"`
		ListingFee      string `json:"listing_fee"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Check if token symbol already exists
	var existingToken models.Token
	if err := h.db.Where("symbol = ? AND chain = ?", req.Symbol, req.Chain).First(&existingToken).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Token with this symbol already exists on this chain"})
		return
	}

	if req.Decimals == 0 {
		req.Decimals = 18
	}

	now := time.Now()
	token := models.Token{
		Name:            req.Name,
		Symbol:          req.Symbol,
		ContractAddress: req.ContractAddress,
		Chain:           req.Chain,
		Decimals:        req.Decimals,
		TotalSupply:     req.TotalSupply,
		LogoURL:         req.LogoURL,
		Website:         req.Website,
		Description:     req.Description,
		IsActive:        true,
		IsVerified:      false,
		ListingFee:      req.ListingFee,
		ListedBy:        adminID,
		ListedAt:        &now,
	}

	if err := h.db.Create(&token).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create token"})
		return
	}

	// Log activity
	logAdminActivity(h.db, adminID, "create_token", "token", strconv.FormatUint(uint64(token.ID), 10), "Created token: "+token.Symbol, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusCreated, token)
}

// UpdateToken updates a token
func (h *TokenHandler) UpdateToken(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	tokenID := c.Param("id")

	var req struct {
		Name        string `json:"name"`
		LogoURL     string `json:"logo_url"`
		Website     string `json:"website"`
		Description string `json:"description"`
		Price       string `json:"price"`
		IsActive    *bool  `json:"is_active"`
		IsVerified  *bool  `json:"is_verified"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var token models.Token
	if err := h.db.First(&token, tokenID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
		return
	}

	updates := map[string]interface{}{}

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.LogoURL != "" {
		updates["logo_url"] = req.LogoURL
	}
	if req.Website != "" {
		updates["website"] = req.Website
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Price != "" {
		updates["price"] = req.Price
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.IsVerified != nil {
		updates["is_verified"] = *req.IsVerified
	}

	if err := h.db.Model(&token).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update token"})
		return
	}

	// Log activity
	logAdminActivity(h.db, adminID, "update_token", "token", tokenID, "Updated token: "+token.Symbol, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, token)
}

// DeleteToken soft deletes a token
func (h *TokenHandler) DeleteToken(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	tokenID := c.Param("id")

	var token models.Token
	if err := h.db.First(&token, tokenID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
		return
	}

	// Soft delete - deactivate
	if err := h.db.Model(&token).Update("is_active", false).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete token"})
		return
	}

	// Log activity
	logAdminActivity(h.db, adminID, "delete_token", "token", tokenID, "Deleted token: "+token.Symbol, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Token deleted successfully"})
}

// ActivateToken activates a token
func (h *TokenHandler) ActivateToken(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	tokenID := c.Param("id")

	var token models.Token
	if err := h.db.First(&token, tokenID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
		return
	}

	if err := h.db.Model(&token).Update("is_active", true).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to activate token"})
		return
	}

	// Log activity
	logAdminActivity(h.db, adminID, "activate_token", "token", tokenID, "Activated token: "+token.Symbol, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Token activated successfully"})
}

// DeactivateToken deactivates a token
func (h *TokenHandler) DeactivateToken(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	tokenID := c.Param("id")

	var token models.Token
	if err := h.db.First(&token, tokenID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
		return
	}

	if err := h.db.Model(&token).Update("is_active", false).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deactivate token"})
		return
	}

	// Log activity
	logAdminActivity(h.db, adminID, "deactivate_token", "token", tokenID, "Deactivated token: "+token.Symbol, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Token deactivated successfully"})
}

// VerifyToken verifies a token
func (h *TokenHandler) VerifyToken(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	tokenID := c.Param("id")

	var token models.Token
	if err := h.db.First(&token, tokenID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
		return
	}

	if err := h.db.Model(&token).Update("is_verified", true).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify token"})
		return
	}

	// Log activity
	logAdminActivity(h.db, adminID, "verify_token", "token", tokenID, "Verified token: "+token.Symbol, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Token verified successfully"})
}

// GetTokenStats gets token statistics
func (h *TokenHandler) GetTokenStats(c *gin.Context) {
	var stats struct {
		TotalTokens      int64 `json:"total_tokens"`
		ActiveTokens     int64 `json:"active_tokens"`
		InactiveTokens   int64 `json:"inactive_tokens"`
		VerifiedTokens   int64 `json:"verified_tokens"`
		UnverifiedTokens int64 `json:"unverified_tokens"`
	}

	h.db.Model(&models.Token{}).Count(&stats.TotalTokens)
	h.db.Model(&models.Token{}).Where("is_active = ?", true).Count(&stats.ActiveTokens)
	h.db.Model(&models.Token{}).Where("is_active = ?", false).Count(&stats.InactiveTokens)
	h.db.Model(&models.Token{}).Where("is_verified = ?", true).Count(&stats.VerifiedTokens)
	h.db.Model(&models.Token{}).Where("is_verified = ?", false).Count(&stats.UnverifiedTokens)

	c.JSON(http.StatusOK, stats)
}

// UpdateTokenPrice updates token price
func (h *TokenHandler) UpdateTokenPrice(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	tokenID := c.Param("id")

	var req struct {
		Price          string `json:"price" binding:"required"`
		MarketCap      string `json:"market_cap"`
		Volume24h      string `json:"volume_24h"`
		PriceChange24h string `json:"price_change_24h"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Price is required"})
		return
	}

	var token models.Token
	if err := h.db.First(&token, tokenID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
		return
	}

	updates := map[string]interface{}{
		"price": req.Price,
	}

	if req.MarketCap != "" {
		updates["market_cap"] = req.MarketCap
	}
	if req.Volume24h != "" {
		updates["volume_24h"] = req.Volume24h
	}
	if req.PriceChange24h != "" {
		updates["price_change_24h"] = req.PriceChange24h
	}

	if err := h.db.Model(&token).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update token price"})
		return
	}

	// Log activity
	logAdminActivity(h.db, adminID, "update_token_price", "token", tokenID, "Updated price: "+req.Price, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, token)
}
