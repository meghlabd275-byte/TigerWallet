package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"admin_backend/internal/models"
	"admin_backend/pkg/database"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// NotificationService handles sending notifications to users
type NotificationService struct {
	redis *redis.Client
}

// NewNotificationService creates a new notification service
func NewNotificationService(redisClient *redis.Client) *NotificationService {
	return &NotificationService{redis: redisClient}
}

// SendKYCNotification sends a KYC status notification to a user via Redis queue
func (s *NotificationService) SendKYCNotification(userID uint, status, message string) error {
	// Store notification in Redis queue for processing by notification service
	key := fmt.Sprintf("notifications:user:%d", userID)
	notification := map[string]interface{}{
		"type":    "kyc",
		"status":  status,
		"message": message,
		"time":    time.Now().Unix(),
	}
	
	notificationJSON, _ := json.Marshal(notification)
	s.redis.LPush(s.redis.Context(), key, notificationJSON)
	s.redis.Expire(s.redis.Context(), key, 7*24*time.Hour)
	
	return nil
}

// KYCHandler handles KYC-related requests
type KYCHandler struct {
	db               *database.PostgresDB
	notificationSvc *NotificationService
}

// NewKYCHandler creates a new KYC handler
func NewKYCHandler(db *database.PostgresDB, redisClient *redis.Client) *KYCHandler {
	return &KYCHandler{
		db:               db,
		notificationSvc: NewNotificationService(redisClient),
	}
}

// ListKYC lists all KYC applications
func (h *KYCHandler) ListKYC(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	status := c.Query("status")
	level := c.Query("level")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var applications []models.KYCApplication
	var total int64

	query := h.db.Model(&models.KYCApplication{}).Preload("User")

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if level != "" {
		levelInt, _ := strconv.Atoi(level)
		query = query.Where("level = ?", levelInt)
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC")

	if err := query.Find(&applications).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch KYC applications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        applications,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

// GetKYC gets a KYC application by ID
func (h *KYCHandler) GetKYC(c *gin.Context) {
	kycID := c.Param("id")

	var kyc models.KYCApplication
	if err := h.db.Preload("User").First(&kyc, kycID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "KYC application not found"})
		return
	}

	c.JSON(http.StatusOK, kyc)
}

// ApproveKYC approves a KYC application
func (h *KYCHandler) ApproveKYC(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	kycID := c.Param("id")

	var req struct {
		Notes string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		req.Notes = ""
	}

	var kyc models.KYCApplication
	if err := h.db.Preload("User").First(&kyc, kycID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "KYC application not found"})
		return
	}

	if kyc.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "KYC application is not pending"})
		return
	}

	now := time.Now()

	// Update KYC application
	if err := h.db.Model(&kyc).Updates(map[string]interface{}{
		"status":     "approved",
		"reviewed_at": now,
		"reviewed_by": adminID,
		"notes":      req.Notes,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to approve KYC"})
		return
	}

	// Update user KYC status
	if err := h.db.Model(&models.User{}).Where("id = ?", kyc.UserID).Updates(map[string]interface{}{
		"kyc_status":     "level" + strconv.Itoa(kyc.Level),
		"kyc_level":      kyc.Level,
		"kyc_verified_at": now,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user KYC status"})
		return
	}

	// Log activity
	logAdminActivity(h.db, adminID, "approve_kyc", "kyc", kycID, "KYC approved at level "+strconv.Itoa(kyc.Level), c.ClientIP(), c.Request.UserAgent())

	// Send notification to user
	notificationMsg := fmt.Sprintf("Your KYC application has been approved at Level %d. You now have access to higher withdrawal limits and additional features.", kyc.Level)
	_ = h.notificationSvc.SendKYCNotification(kyc.UserID, "approved", notificationMsg)

	c.JSON(http.StatusOK, gin.H{"message": "KYC approved successfully"})
}

// RejectKYC rejects a KYC application
func (h *KYCHandler) RejectKYC(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	kycID := c.Param("id")

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Rejection reason is required"})
		return
	}

	var kyc models.KYCApplication
	if err := h.db.Preload("User").First(&kyc, kycID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "KYC application not found"})
		return
	}

	if kyc.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "KYC application is not pending"})
		return
	}

	now := time.Now()

	// Update KYC application
	if err := h.db.Model(&kyc).Updates(map[string]interface{}{
		"status":            "rejected",
		"reviewed_at":       now,
		"reviewed_by":       adminID,
		"rejection_reason":  req.Reason,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reject KYC"})
		return
	}

	// Update user KYC status
	if err := h.db.Model(&models.User{}).Where("id = ?", kyc.UserID).Updates(map[string]interface{}{
		"kyc_status":           "rejected",
		"kyc_rejection_reason": req.Reason,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user KYC status"})
		return
	}

	// Log activity
	logAdminActivity(h.db, adminID, "reject_kyc", "kyc", kycID, "KYC rejected: "+req.Reason, c.ClientIP(), c.Request.UserAgent())

	// Send notification to user
	notificationMsg := fmt.Sprintf("Your KYC application has been rejected. Reason: %s. Please submit a new application with the required documents.", req.Reason)
	_ = h.notificationSvc.SendKYCNotification(kyc.UserID, "rejected", notificationMsg)

	c.JSON(http.StatusOK, gin.H{"message": "KYC rejected"})
}

// GetKYCStats gets KYC statistics
func (h *KYCHandler) GetKYCStats(c *gin.Context) {
	var stats struct {
		Pending  int64 `json:"pending"`
		Approved int64 `json:"approved"`
		Rejected int64 `json:"rejected"`
		Total    int64 `json:"total"`
	}

	h.db.Model(&models.KYCApplication{}).Where("status = ?", "pending").Count(&stats.Pending)
	h.db.Model(&models.KYCApplication{}).Where("status = ?", "approved").Count(&stats.Approved)
	h.db.Model(&models.KYCApplication{}).Where("status = ?", "rejected").Count(&stats.Rejected)
	h.db.Model(&models.KYCApplication{}).Count(&stats.Total)

	c.JSON(http.StatusOK, stats)
}
