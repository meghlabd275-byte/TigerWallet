package handlers

import (
	"net/http"
	"strconv"
	"time"

	"admin_backend/internal/models"
	"admin_backend/pkg/database"

	"github.com/gin-gonic/gin"
)

// FeeHandler handles fee-related requests - COMPLETE IMPLEMENTATION
type FeeHandler struct {
	db *database.PostgresDB
}

// NewFeeHandler creates a new fee handler
func NewFeeHandler(db *database.PostgresDB) *FeeHandler {
	return &FeeHandler{db: db}
}

// ListFees lists all fee structures
func (h *FeeHandler) ListFees(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	feeType := c.Query("type")
	chain := c.Query("chain")
	status := c.Query("status")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var fees []models.FeeStructure
	var total int64

	query := h.db.Model(&models.FeeStructure{})

	if feeType != "" {
		query = query.Where("fee_type = ?", feeType)
	}
	if chain != "" {
		query = query.Where("chain = ?", chain)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC")

	if err := query.Find(&fees).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch fees"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        fees,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

// GetFee gets a fee by ID
func (h *FeeHandler) GetFee(c *gin.Context) {
	feeID := c.Param("id")

	var fee models.FeeStructure
	if err := h.db.First(&fee, feeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Fee not found"})
		return
	}

	c.JSON(http.StatusOK, fee)
}

// CreateFee creates a new fee structure
func (h *FeeHandler) CreateFee(c *gin.Context) {
	adminID := c.GetUint("admin_id")

	var req struct {
		Name            string  `json:"name" binding:"required"`
		FeeType         string  `json:"fee_type" binding:"required"`
		Chain           string  `json:"chain"`
		Token           string  `json:"token"`
		FeePercent      float64 `json:"fee_percent"`
		FeeFixed        float64 `json:"fee_fixed"`
		MinFee          float64 `json:"min_fee"`
		MaxFee          float64 `json:"max_fee"`
		Tier            string  `json:"tier"`
		VolumeThreshold float64 `json:"volume_threshold"`
		IsActive        bool    `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	fee := models.FeeStructure{
		Name:            req.Name,
		FeeType:         req.FeeType,
		Chain:           req.Chain,
		Token:           req.Token,
		FeePercent:      req.FeePercent,
		FeeFixed:        req.FeeFixed,
		MinFee:          req.MinFee,
		MaxFee:          req.MaxFee,
		Tier:            req.Tier,
		VolumeThreshold: req.VolumeThreshold,
		IsActive:        req.IsActive,
		CreatedBy:       adminID,
	}

	if err := h.db.Create(&fee).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create fee"})
		return
	}

	logAdminActivity(h.db, adminID, "create_fee", "fee", strconv.FormatUint(uint64(fee.ID)), "Created fee: "+fee.Name, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusCreated, fee)
}

// UpdateFee updates a fee structure
func (h *FeeHandler) UpdateFee(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	feeID := c.Param("id")

	var req struct {
		Name            string  `json:"name"`
		FeePercent      float64 `json:"fee_percent"`
		FeeFixed        float64 `json:"fee_fixed"`
		MinFee          float64 `json:"min_fee"`
		MaxFee          float64 `json:"max_fee"`
		Tier            string  `json:"tier"`
		VolumeThreshold float64 `json:"volume_threshold"`
		IsActive        *bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var fee models.FeeStructure
	if err := h.db.First(&fee, feeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Fee not found"})
		return
	}

	updates := map[string]interface{}{}

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.FeePercent > 0 {
		updates["fee_percent"] = req.FeePercent
	}
	if req.FeeFixed > 0 {
		updates["fee_fixed"] = req.FeeFixed
	}
	if req.MinFee > 0 {
		updates["min_fee"] = req.MinFee
	}
	if req.MaxFee > 0 {
		updates["max_fee"] = req.MaxFee
	}
	if req.Tier != "" {
		updates["tier"] = req.Tier
	}
	if req.VolumeThreshold > 0 {
		updates["volume_threshold"] = req.VolumeThreshold
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if err := h.db.Model(&fee).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update fee"})
		return
	}

	logAdminActivity(h.db, adminID, "update_fee", "fee", feeID, "Updated fee: "+fee.Name, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, fee)
}

// DeleteFee deletes a fee structure
func (h *FeeHandler) DeleteFee(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	feeID := c.Param("id")

	var fee models.FeeStructure
	if err := h.db.First(&fee, feeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Fee not found"})
		return
	}

	if err := h.db.Delete(&fee).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete fee"})
		return
	}

	logAdminActivity(h.db, adminID, "delete_fee", "fee", feeID, "Deleted fee: "+fee.Name, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Fee deleted successfully"})
}

// CalculateFee calculates fee for a given amount
func (h *FeeHandler) CalculateFee(c *gin.Context) {
	var req struct {
		FeeType string  `json:"fee_type" binding:"required"`
		Chain   string  `json:"chain"`
		Token   string  `json:"token"`
		Amount  float64 `json:"amount" binding:"required"`
		Tier    string  `json:"tier"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Find applicable fee structure
	var fee models.FeeStructure
	query := h.db.Where("fee_type = ? AND is_active = ?", req.FeeType, true)
	
	if req.Chain != "" {
		query = query.Where("chain = ? OR chain = ''", req.Chain)
	}
	if req.Token != "" {
		query = query.Where("token = ? OR token = ''", req.Token)
	}
	if req.Tier != "" {
		query = query.Where("tier = ? OR tier = ''", req.Tier)
	}

	if err := query.First(&fee).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No applicable fee structure found"})
		return
	}

	// Calculate fee
	var calculatedFee float64
	if fee.FeePercent > 0 {
		calculatedFee = req.Amount * (fee.FeePercent / 100)
	}
	if fee.FeeFixed > 0 {
		calculatedFee += fee.FeeFixed
	}

	// Apply min/max bounds
	if fee.MinFee > 0 && calculatedFee < fee.MinFee {
		calculatedFee = fee.MinFee
	}
	if fee.MaxFee > 0 && calculatedFee > fee.MaxFee {
		calculatedFee = fee.MaxFee
	}

	c.JSON(http.StatusOK, gin.H{
		"amount":         req.Amount,
		"fee_type":       fee.FeeType,
		"fee_percent":   fee.FeePercent,
		"fee_fixed":      fee.FeeFixed,
		"calculated_fee": calculatedFee,
		"net_amount":     req.Amount - calculatedFee,
	})
}

// GetFeeStats gets fee statistics
func (h *FeeHandler) GetFeeStats(c *gin.Context) {
	var stats struct {
		TotalFees     int64   `json:"total_fees"`
		ActiveFees   int64   `json:"active_fees"`
		InactiveFees int64   `json:"inactive_fees"`
		TotalRevenue float64 `json:"total_revenue"`
	}

	h.db.Model(&models.FeeStructure{}).Count(&stats.TotalFees)
	h.db.Model(&models.FeeStructure{}).Where("is_active = ?", true).Count(&stats.ActiveFees)
	h.db.Model(&models.FeeStructure{}).Where("is_active = ?", false).Count(&stats.InactiveFees)

	// Calculate total revenue from transactions
	h.db.Model(&models.Transaction{}).Where("status = ?", "completed").Select("COALESCE(SUM(fee_amount), 0)").Scan(&stats.TotalRevenue)

	c.JSON(http.StatusOK, stats)
}
