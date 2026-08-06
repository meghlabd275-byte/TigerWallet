package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"admin_backend/internal/models"
	"admin_backend/pkg/database"

	"github.com/gin-gonic/gin"
)

// NotificationHandler handles notification operations
type NotificationHandler struct {
	db *database.PostgresDB
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(db *database.PostgresDB) *NotificationHandler {
	return &NotificationHandler{db: db}
}

// NotificationRequest represents notification request
type NotificationRequest struct {
	UserID    uint     `json:"user_id"`
	Title     string   `json:"title" binding:"required"`
	Message   string   `json:"message" binding:"required"`
	Type      string   `json:"type"` // info, warning, error, success
	Priority  string   `json:"priority"` // low, normal, high, urgent
	ActionURL string   `json:"action_url"`
	SendEmail bool     `json:"send_email"`
	SendSMS   bool     `json:"send_sms"`
	SendPush  bool     `json:"send_push"`
}

// SendNotification sends a notification
// POST /api/v1/admin/notifications
func (h *NotificationHandler) SendNotification(c *gin.Context) {
	var req NotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	notification := models.Notification{
		AdminID:   c.GetUint("admin_id"),
		Title:     req.Title,
		Message:   req.Message,
		Type:      req.Type,
		Priority:  req.Priority,
		ActionURL: req.ActionURL,
		IsRead:    false,
	}

	if req.UserID > 0 {
		notification.AdminID = req.UserID
	}

	if err := h.db.Create(&notification).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create notification"})
		return
	}

	// Send via different channels
	if req.SendEmail {
		// TODO: Send email notification
	}
	if req.SendSMS {
		// TODO: Send SMS notification
	}
	if req.SendPush {
		// TODO: Send push notification
	}

	adminID := c.GetUint("admin_id")
	logAdminActivity(h.db, adminID, "send_notification", "notification", 
		fmt.Sprintf("%d", notification.ID), "Sent notification: "+notification.Title, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusCreated, notification)
}

// SendBulkNotifications sends notifications to multiple users
// POST /api/v1/admin/notifications/bulk
func (h *NotificationHandler) SendBulkNotifications(c *gin.Context) {
	var req struct {
		UserIDs    []uint  `json:"user_ids" binding:"required"`
		Title     string  `json:"title" binding:"required"`
		Message   string  `json:"message" binding:"required"`
		Type      string  `json:"type"`
		SendEmail bool    `json:"send_email"`
		SendPush  bool    `json:"send_push"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	notifiedCount := 0
	for _, userID := range req.UserIDs {
		notification := models.Notification{
			AdminID: userID,
			Title:   req.Title,
			Message: req.Message,
			Type:    req.Type,
			IsRead:  false,
		}

		if err := h.db.Create(&notification).Error; err == nil {
			notifiedCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total_users":  len(req.UserIDs),
		"notified":     notifiedCount,
	})
}

// SendToAllUsers sends notification to all users
// POST /api/v1/admin/notifications/broadcast
func (h *NotificationHandler) SendToAllUsers(c *gin.Context) {
	var req struct {
		Title      string `json:"title" binding:"required"`
		Message    string `json:"message" binding:"required"`
		Type       string `json:"type"`
		MinKYCLevel int   `json:"min_kyc_level"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Get users based on filters
	var users []models.User
	query := h.db.Model(&models.User{})

	if req.MinKYCLevel > 0 {
		query = query.Where("kyc_level >= ?", req.MinKYCLevel)
	}

	if err := query.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	notifiedCount := 0
	for _, user := range users {
		notification := models.Notification{
			AdminID: user.ID,
			Title:   req.Title,
			Message: req.Message,
			Type:    req.Type,
			IsRead:  false,
		}

		if err := h.db.Create(&notification).Error; err == nil {
			notifiedCount++
		}
	}

	adminID := c.GetUint("admin_id")
	logAdminActivity(h.db, adminID, "broadcast_notification", "notification", 
		fmt.Sprintf("%d", notifiedCount), "Broadcast notification: "+req.Title, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{
		"total_users": len(users),
		"notified":    notifiedCount,
	})
}

// GetNotifications gets notifications
// GET /api/v1/admin/notifications
func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	isRead := c.Query("is_read")
	priority := c.Query("priority")
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var notifications []models.Notification
	var total int64

	query := h.db.Model(&models.Notification{})

	if isRead != "" {
		read := isRead == "true"
		query = query.Where("is_read = ?", read)
	}
	if priority != "" {
		query = query.Where("priority = ?", priority)
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC")

	if err := query.Find(&notifications).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        notifications,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

// MarkAsRead marks notification as read
// PUT /api/v1/admin/notifications/:id/read
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	now := time.Now()
	err = h.db.Model(&models.Notification{}).Where("id = ?", id).Updates(map[string]interface{}{
		"is_read": true,
		"read_at": now,
	}).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification marked as read"})
}

// MarkAllAsRead marks all notifications as read
// PUT /api/v1/admin/notifications/read-all
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	now := time.Now()
	err := h.db.Model(&models.Notification{}).Where("is_read = ?", false).Updates(map[string]interface{}{
		"is_read": true,
		"read_at": now,
	}).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark all as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All notifications marked as read"})
}

// DeleteNotification deletes a notification
// DELETE /api/v1/admin/notifications/:id
func (h *NotificationHandler) DeleteNotification(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	if err := h.db.Delete(&models.Notification{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete notification"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification deleted successfully"})
}

// GetNotificationStats gets notification statistics
// GET /api/v1/admin/notifications/stats
func (h *NotificationHandler) GetNotificationStats(c *gin.Context) {
	var stats struct {
		TotalNotifications int64 `json:"total_notifications"`
		UnreadCount     int64 `json:"unread_count"`
		InfoCount       int64 `json:"info_count"`
		WarningCount    int64 `json:"warning_count"`
		ErrorCount      int64 `json:"error_count"`
		SuccessCount    int64 `json:"success_count"`
		TodayCount     int64 `json:"today_count"`
	}

	h.db.Model(&models.Notification{}).Count(&stats.TotalNotifications)
	h.db.Model(&models.Notification{}).Where("is_read = ?", false).Count(&stats.UnreadCount)
	h.db.Model(&models.Notification{}).Where("type = ?", "info").Count(&stats.InfoCount)
	h.db.Model(&models.Notification{}).Where("type = ?", "warning").Count(&stats.WarningCount)
	h.db.Model(&models.Notification{}).Where("type = ?", "error").Count(&stats.ErrorCount)
	h.db.Model(&models.Notification{}).Where("type = ?", "success").Count(&stats.SuccessCount)

	today := time.Now().Truncate(24 * time.Hour)
	h.db.Model(&models.Notification{}).Where("created_at >= ?", today).Count(&stats.TodayCount)

	c.JSON(http.StatusOK, stats)
}

// TemplateNotificationRequest represents template notification request
type TemplateNotificationRequest struct {
	TemplateID string   `json:"template_id" binding:"required"`
	UserIDs    []uint   `json:"user_ids"`
	Variables  map[string]string `json:"variables"`
}

// SendTemplateNotification sends notification using template
// POST /api/v1/admin/notifications/template
func (h *NotificationHandler) SendTemplateNotification(c *gin.Context) {
	var req TemplateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Get template
	var template struct {
		Title string
		Message string
	}

	// In production, fetch from database
	// For now, use placeholder
	title := "Notification: " + req.TemplateID
	message := "Template notification"

	notifiedCount := 0
	for _, userID := range req.UserIDs {
		notification := models.Notification{
			AdminID: userID,
			Title:   title,
			Message: message,
			Type:    "info",
			IsRead:  false,
		}

		if err := h.db.Create(&notification).Error; err == nil {
			notifiedCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"notified": notifiedCount,
	})
}
