package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/tigerwallet/admin/internal/models"
	"github.com/tigerwallet/admin/pkg/database"

	"github.com/gin-gonic/gin"
)

// UserHandler handles user-related requests
type UserHandler struct {
	db *database.PostgresDB
}

// NewUserHandler creates a new user handler
func NewUserHandler(db *database.PostgresDB) *UserHandler {
	return &UserHandler{db: db}
}

// ListUsers lists all users with pagination and filters
func (h *UserHandler) ListUsers(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	search := c.Query("search")
	status := c.Query("status")
	kycStatus := c.Query("kyc_status")
	whiteLabelID := c.Query("white_label_id")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var users []models.User
	var total int64

	query := h.db.Model(&models.User{})

	// Apply filters
	if search != "" {
		query = query.Where("email ILIKE ? OR username ILIKE ? OR wallet_address ILIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if kycStatus != "" {
		query = query.Where("kyc_status = ?", kycStatus)
	}
	if whiteLabelID != "" {
		query = query.Where("white_label_id = ?", whiteLabelID)
	}

	// Count total
	query.Count(&total)

	// Paginate
	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC")

	if err := query.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        users,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

// GetUser gets a user by ID
func (h *UserHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")

	var user models.User
	if err := h.db.Preload("KYCApplications").First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateUser updates a user
func (h *UserHandler) UpdateUser(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	userID := c.Param("id")

	var req struct {
		Status   string `json:"status"`
		KYCLevel int    `json:"kyc_level"`
		Tags     string `json:"tags"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	updates := map[string]interface{}{}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if req.KYCLevel > 0 {
		updates["kyc_level"] = req.KYCLevel
	}

	if err := h.db.Model(&user).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	// Log activity
	logAdminActivity(h.db, adminID, "update_user", "user", userID, "User updated", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, user)
}

// DeleteUser soft deletes a user
func (h *UserHandler) DeleteUser(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	userID := c.Param("id")

	if err := h.db.Model(&models.User{}).Where("id = ?", userID).Update("status", "deleted").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	// Log activity
	logAdminActivity(h.db, adminID, "delete_user", "user", userID, "User deleted", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// VerifyKYC verifies a user's KYC
func (h *UserHandler) VerifyKYC(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	userID := c.Param("id")

	var req struct {
		Level int    `json:"level" binding:"required"`
		Notes string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	now := time.Now()
	if err := h.db.Model(&user).Updates(map[string]interface{}{
		"kyc_status":      "level" + strconv.Itoa(req.Level),
		"kyc_level":       req.Level,
		"kyc_verified_at": now,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify KYC"})
		return
	}

	// Create KYC application record
	kycApp := models.KYCApplication{
		UserID:      user.ID,
		Level:       req.Level,
		Status:      "approved",
		SubmittedAt: now,
		ReviewedAt:  &now,
		ReviewedBy:  &adminID,
		Notes:       req.Notes,
		IPAddress:   c.ClientIP(),
	}
	h.db.Create(&kycApp)

	// Log activity
	logAdminActivity(h.db, adminID, "verify_kyc", "user", userID, "KYC verified at level "+strconv.Itoa(req.Level), c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "KYC verified successfully"})
}

// DashboardHandler handles dashboard requests
type DashboardHandler struct {
	db *database.PostgresDB
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(db *database.PostgresDB) *DashboardHandler {
	return &DashboardHandler{db: db}
}

// GetDashboard gets dashboard statistics
func (h *DashboardHandler) GetDashboard(c *gin.Context) {
	var stats DashboardStats

	// Total users
	h.db.Model(&models.User{}).Count(&stats.TotalUsers)

	// Active users
	h.db.Model(&models.User{}).Where("status = ?", "active").Count(&stats.ActiveUsers)

	// Pending KYC
	h.db.Model(&models.User{}).Where("kyc_status = ?", "pending").Count(&stats.PendingKYC)

	// Today's new users
	today := time.Now().Truncate(24 * time.Hour)
	h.db.Model(&models.User{}).Where("created_at >= ?", today).Count(&stats.NewUsersToday)

	// Today's transactions
	h.db.Model(&models.Transaction{}).Where("created_at >= ?", today).Count(&stats.TodayTransactions)

	stats.SystemHealth = "99.9%"

	c.JSON(http.StatusOK, stats)
}

// TransactionHandler handles transaction requests
type TransactionHandler struct {
	db *database.PostgresDB
}

// NewTransactionHandler creates a new transaction handler
func NewTransactionHandler(db *database.PostgresDB) *TransactionHandler {
	return &TransactionHandler{db: db}
}

// ListTransactions lists all transactions
func (h *TransactionHandler) ListTransactions(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	txType := c.Query("type")
	status := c.Query("status")
	chain := c.Query("chain")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var transactions []models.Transaction
	var total int64

	query := h.db.Model(&models.Transaction{})

	if txType != "" {
		query = query.Where("type = ?", txType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if chain != "" {
		query = query.Where("chain = ?", chain)
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC").Preload("User")

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

// GetTransaction gets a transaction by ID
func (h *TransactionHandler) GetTransaction(c *gin.Context) {
	txID := c.Param("id")

	var tx models.Transaction
	if err := h.db.Preload("User").First(&tx, txID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
		return
	}

	c.JSON(http.StatusOK, tx)
}

// FlagTransaction flags a transaction
func (h *TransactionHandler) FlagTransaction(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	txID := c.Param("id")

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.db.Model(&models.Transaction{}).Where("id = ?", txID).Updates(map[string]interface{}{
		"flagged":     true,
		"flag_reason": req.Reason,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to flag transaction"})
		return
	}

	logAdminActivity(h.db, adminID, "flag_transaction", "transaction", txID, "Transaction flagged: "+req.Reason, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Transaction flagged"})
}

// Helper function to log admin activity
func logAdminActivity(db *database.PostgresDB, adminID uint, action, resource, resourceID, details, ip, userAgent string) {
	activity := models.AdminActivity{
		AdminID:    adminID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		IPAddress:  ip,
		UserAgent:  userAgent,
		Status:     "success",
	}
	db.Create(&activity)
}
