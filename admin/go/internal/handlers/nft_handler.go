package handlers

import (
	"net/http"
	"strconv"

	"github.com/tigerwallet/admin/internal/models"
	"github.com/tigerwallet/admin/pkg/database"

	"github.com/gin-gonic/gin"
)

// NFTHandler handles NFT-related requests - COMPLETE IMPLEMENTATION
type NFTHandler struct {
	db *database.PostgresDB
}

// NewNFTHandler creates a new NFT handler
func NewNFTHandler(db *database.PostgresDB) *NFTHandler {
	return &NFTHandler{db: db}
}

// ListNFTs lists all NFTs
func (h *NFTHandler) ListNFTs(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	collection := c.Query("collection")
	chain := c.Query("chain")
	status := c.Query("status")
	tokenType := c.Query("token_type")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var nfts []models.NFT
	var total int64

	query := h.db.Model(&models.NFT{})

	if collection != "" {
		query = query.Where("collection_address = ?", collection)
	}
	if chain != "" {
		query = query.Where("chain = ?", chain)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if tokenType != "" {
		query = query.Where("token_type = ?", tokenType)
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC")

	if err := query.Find(&nfts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch NFTs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        nfts,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

// GetNFT gets an NFT by ID
func (h *NFTHandler) GetNFT(c *gin.Context) {
	nftID := c.Param("id")

	var nft models.NFT
	if err := h.db.First(&nft, nftID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "NFT not found"})
		return
	}

	c.JSON(http.StatusOK, nft)
}

// CreateNFT creates a new NFT
func (h *NFTHandler) CreateNFT(c *gin.Context) {
	adminID := c.GetUint("admin_id")

	var req struct {
		Name              string  `json:"name" binding:"required"`
		Description       string  `json:"description"`
		CollectionAddress string  `json:"collection_address"`
		TokenID           string  `json:"token_id"`
		Chain             string  `json:"chain" binding:"required"`
		TokenType         string  `json:"token_type"`
		ContractType      string  `json:"contract_type"`
		MetadataURL       string  `json:"metadata_url"`
		ImageURL          string  `json:"image_url"`
		ExternalURL       string  `json:"external_url"`
		Creator           string  `json:"creator"`
		Owner             string  `json:"owner"`
		Royalty           float64 `json:"royalty"`
		Attributes        string  `json:"attributes"`
		IsActive          bool    `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	nft := models.NFT{
		Name:              req.Name,
		Description:       req.Description,
		CollectionAddress: req.CollectionAddress,
		TokenID:           req.TokenID,
		Chain:             req.Chain,
		TokenType:         req.TokenType,
		ContractType:      req.ContractType,
		MetadataURL:       req.MetadataURL,
		ImageURL:          req.ImageURL,
		ExternalURL:       req.ExternalURL,
		Creator:           req.Creator,
		Owner:             req.Owner,
		Royalty:           req.Royalty,
		Attributes:        req.Attributes,
		Status:            "active",
		IsActive:          req.IsActive,
		CreatedBy:         adminID,
	}

	if err := h.db.Create(&nft).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create NFT"})
		return
	}

	logAdminActivity(h.db, adminID, "create_nft", "nft", strconv.FormatUint(uint64(nft.ID), 10),
		"Created NFT: "+nft.Name, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusCreated, nft)
}

// UpdateNFT updates an NFT
func (h *NFTHandler) UpdateNFT(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	nftID := c.Param("id")

	var req struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		MetadataURL string  `json:"metadata_url"`
		ImageURL    string  `json:"image_url"`
		ExternalURL string  `json:"external_url"`
		Owner       string  `json:"owner"`
		Royalty     float64 `json:"royalty"`
		Attributes  string  `json:"attributes"`
		IsActive    *bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var nft models.NFT
	if err := h.db.First(&nft, nftID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "NFT not found"})
		return
	}

	updates := map[string]interface{}{}

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.MetadataURL != "" {
		updates["metadata_url"] = req.MetadataURL
	}
	if req.ImageURL != "" {
		updates["image_url"] = req.ImageURL
	}
	if req.ExternalURL != "" {
		updates["external_url"] = req.ExternalURL
	}
	if req.Owner != "" {
		updates["owner"] = req.Owner
	}
	if req.Royalty > 0 {
		updates["royalty"] = req.Royalty
	}
	if req.Attributes != "" {
		updates["attributes"] = req.Attributes
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if err := h.db.Model(&nft).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update NFT"})
		return
	}

	logAdminActivity(h.db, adminID, "update_nft", "nft", nftID,
		"Updated NFT: "+nft.Name, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, nft)
}

// DeleteNFT deletes an NFT
func (h *NFTHandler) DeleteNFT(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	nftID := c.Param("id")

	var nft models.NFT
	if err := h.db.First(&nft, nftID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "NFT not found"})
		return
	}

	if err := h.db.Delete(&nft).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete NFT"})
		return
	}

	logAdminActivity(h.db, adminID, "delete_nft", "nft", nftID,
		"Deleted NFT: "+nft.Name, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "NFT deleted successfully"})
}

// SuspendNFT suspends an NFT
func (h *NFTHandler) SuspendNFT(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	nftID := c.Param("id")

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Reason is required"})
		return
	}

	var nft models.NFT
	if err := h.db.First(&nft, nftID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "NFT not found"})
		return
	}

	if err := h.db.Model(&nft).Updates(map[string]interface{}{
		"status":         "suspended",
		"suspend_reason": req.Reason,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to suspend NFT"})
		return
	}

	logAdminActivity(h.db, adminID, "suspend_nft", "nft", nftID,
		"Suspended NFT: "+nft.Name+" - Reason: "+req.Reason, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "NFT suspended successfully"})
}

// GetNFTStats gets NFT statistics
func (h *NFTHandler) GetNFTStats(c *gin.Context) {
	var stats struct {
		TotalNFTs     int64   `json:"total_nfts"`
		ActiveNFTs    int64   `json:"active_nfts"`
		SuspendedNFTs int64   `json:"suspended_nfts"`
		TotalVolume   float64 `json:"total_volume"`
	}

	h.db.Model(&models.NFT{}).Count(&stats.TotalNFTs)
	h.db.Model(&models.NFT{}).Where("status = ?", "active").Count(&stats.ActiveNFTs)
	h.db.Model(&models.NFT{}).Where("status = ?", "suspended").Count(&stats.SuspendedNFTs)

	c.JSON(http.StatusOK, stats)
}

// FlagNFT flags an NFT for review.
func (h *NFTHandler) FlagNFT(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	nftID := c.Param("id")

	var nft models.NFT
	if err := h.db.First(&nft, nftID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "NFT not found"})
		return
	}

	if err := h.db.Model(&nft).Update("status", "flagged").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to flag NFT"})
		return
	}

	// Reload to reflect the updated status in the response.
	h.db.First(&nft, nftID)

	logAdminActivity(h.db, adminID, "flag_nft", "nft", nftID,
		"Flagged NFT: "+nft.Name, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, nft)
}
