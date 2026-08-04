package handlers

import (
	"net/http"
	"strconv"
	"time"

	"admin_backend/internal/models"
	"admin_backend/pkg/database"

	"github.com/gin-gonic/gin"
)

// WithdrawalHandler handles withdrawal-related requests
type WithdrawalHandler struct {
	db *database.PostgresDB
}

// NewWithdrawalHandler creates a new withdrawal handler
func NewWithdrawalHandler(db *database.PostgresDB) *WithdrawalHandler {
	return &WithdrawalHandler{db: db}
}

// ListWithdrawals lists all withdrawals
func (h *WithdrawalHandler) ListWithdrawals(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	status := c.Query("status")
	chain := c.Query("chain")
	token := c.Query("token")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var withdrawals []models.Withdrawal
	var total int64

	query := h.db.Model(&models.Withdrawal{}).Preload("User")

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if chain != "" {
		query = query.Where("chain = ?", chain)
	}
	if token != "" {
		query = query.Where("token = ?", token)
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC")

	if err := query.Find(&withdrawals).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch withdrawals"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        withdrawals,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

// GetWithdrawal gets a withdrawal by ID
func (h *WithdrawalHandler) GetWithdrawal(c *gin.Context) {
	withdrawalID := c.Param("id")

	var withdrawal models.Withdrawal
	if err := h.db.Preload("User").First(&withdrawal, withdrawalID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Withdrawal not found"})
		return
	}

	c.JSON(http.StatusOK, withdrawal)
}

// ApproveWithdrawal approves a withdrawal
func (h *WithdrawalHandler) ApproveWithdrawal(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	withdrawalID := c.Param("id")

	var req struct {
		Notes string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		req.Notes = ""
	}

	var withdrawal models.Withdrawal
	if err := h.db.Preload("User").First(&withdrawal, withdrawalID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Withdrawal not found"})
		return
	}

	if withdrawal.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Withdrawal is not pending"})
		return
	}

	now := time.Now()

	// Update withdrawal
	if err := h.db.Model(&withdrawal).Updates(map[string]interface{}{
		"status":    "approved",
		"approved_at": now,
		"approved_by": adminID,
		"notes":     req.Notes,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to approve withdrawal"})
		return
	}

	// TODO: Initiate blockchain transaction

	// Log activity
	logAdminActivity(h.db, adminID, "approve_withdrawal", "withdrawal", withdrawalID, "Withdrawal approved", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Withdrawal approved successfully"})
}

// RejectWithdrawal rejects a withdrawal
func (h *WithdrawalHandler) RejectWithdrawal(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	withdrawalID := c.Param("id")

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Rejection reason is required"})
		return
	}

	var withdrawal models.Withdrawal
	if err := h.db.Preload("User").First(&withdrawal, withdrawalID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Withdrawal not found"})
		return
	}

	if withdrawal.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Withdrawal is not pending"})
		return
	}

	now := time.Now()

	// Update withdrawal
	if err := h.db.Model(&withdrawal).Updates(map[string]interface{}{
		"status":           "rejected",
		"rejected_at":     now,
		"rejected_by":      adminID,
		"rejection_reason": req.Reason,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reject withdrawal"})
		return
	}

	// TODO: Refund user's balance

	// Log activity
	logAdminActivity(h.db, adminID, "reject_withdrawal", "withdrawal", withdrawalID, "Withdrawal rejected: "+req.Reason, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Withdrawal rejected"})
}

// ProcessWithdrawal processes a withdrawal (marks as completed)
func (h *WithdrawalHandler) ProcessWithdrawal(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	withdrawalID := c.Param("id")

	var req struct {
		TxHash string `json:"tx_hash" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Transaction hash is required"})
		return
	}

	var withdrawal models.Withdrawal
	if err := h.db.Preload("User").First(&withdrawal, withdrawalID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Withdrawal not found"})
		return
	}

	if withdrawal.Status != "approved" && withdrawal.Status != "processing" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Withdrawal must be approved first"})
		return
	}

	now := time.Now()

	// Update withdrawal
	if err := h.db.Model(&withdrawal).Updates(map[string]interface{}{
		"status":      "completed",
		"processed_at": now,
		"tx_hash":     req.TxHash,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process withdrawal"})
		return
	}

	// Log activity
	logAdminActivity(h.db, adminID, "process_withdrawal", "withdrawal", withdrawalID, "Withdrawal processed: "+req.TxHash, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Withdrawal processed successfully"})
}

// GetWithdrawalStats gets withdrawal statistics
func (h *WithdrawalHandler) GetWithdrawalStats(c *gin.Context) {
	var stats struct {
		Pending   int64   `json:"pending"`
		Approved  int64   `json:"approved"`
		Completed int64   `json:"completed"`
		Rejected  int64   `json:"rejected"`
		Total     float64 `json:"total"`
	}

	h.db.Model(&models.Withdrawal{}).Where("status = ?", "pending").Count(&stats.Pending)
	h.db.Model(&models.Withdrawal{}).Where("status = ?", "approved").Count(&stats.Approved)
	h.db.Model(&models.Withdrawal{}).Where("status = ?", "completed").Count(&stats.Completed)
	h.db.Model(&models.Withdrawal{}).Where("status = ?", "rejected").Count(&stats.Rejected)

	// Get total amount
	var result []struct {
		Total string
	}
	h.db.Model(&models.Withdrawal{}).Where("status IN ?", []string{"completed", "processing"}).Pluck("COALESCE(SUM(amount), 0)", &result)
	if len(result) > 0 {
		stats.Total, _ = strconv.ParseFloat(result[0].Total)
	}

	c.JSON(http.StatusOK, stats)
}

// BulkApproveWithdrawals approves multiple withdrawals at once
func (h *WithdrawalHandler) BulkApproveWithdrawals(c *gin.Context) {
	adminID := c.GetUint("admin_id")

	var req struct {
		WithdrawalIDs []uint `json:"withdrawal_ids" binding:"required"`
		Notes         string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Withdrawal IDs are required"})
		return
	}

	if len(req.WithdrawalIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No withdrawal IDs provided"})
		return
	}

	if len(req.WithdrawalIDs) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot approve more than 50 withdrawals at once"})
		return
	}

	now := time.Now()

	// Update withdrawals
	result := h.db.Model(&models.Withdrawal{}).
		Where("id IN ? AND status = ?", req.WithdrawalIDs, "pending").
		Updates(map[string]interface{}{
			"status":     "approved",
			"approved_at": now,
			"approved_by": adminID,
			"notes":      req.Notes,
		})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to approve withdrawals"})
		return
	}

	// Log activity
	logAdminActivity(h.db, adminID, "bulk_approve_withdrawals", "withdrawal", "", 
		"Approved "+strconv.Itoa(int(result.RowsAffected))+" withdrawals", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{
		"message":            "Withdrawals approved successfully",
		"approved_count":     result.RowsAffected,
		"requested_count":    len(req.WithdrawalIDs),
	})
}
