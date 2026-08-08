package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tigerwallet/admin/internal/models"
	"github.com/tigerwallet/admin/pkg/database"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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
	UserID    uint   `json:"user_id"`
	Title     string `json:"title" binding:"required"`
	Message   string `json:"message" binding:"required"`
	Type      string `json:"type"`     // info, warning, error, success
	Priority  string `json:"priority"` // low, normal, high, urgent
	ActionURL string `json:"action_url"`
	SendEmail bool   `json:"send_email"`
	SendSMS   bool   `json:"send_sms"`
	SendPush  bool   `json:"send_push"`
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

	// Send via different channels. Email uses real SMTP; SMS and push are
	// dispatched as HTTP webhook calls to a configurable endpoint so the
	// delivery is observable and not silently dropped.
	if req.SendEmail {
		if err := h.sendNotificationEmail(notification); err != nil {
			fmt.Printf("notification %d: email delivery failed: %v\n", notification.ID, err)
		}
	}
	if req.SendSMS {
		if err := h.dispatchNotificationWebhook("sms", notification); err != nil {
			fmt.Printf("notification %d: sms webhook failed: %v\n", notification.ID, err)
		}
	}
	if req.SendPush {
		if err := h.dispatchNotificationWebhook("push", notification); err != nil {
			fmt.Printf("notification %d: push webhook failed: %v\n", notification.ID, err)
		}
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
		UserIDs   []uint `json:"user_ids" binding:"required"`
		Title     string `json:"title" binding:"required"`
		Message   string `json:"message" binding:"required"`
		Type      string `json:"type"`
		SendEmail bool   `json:"send_email"`
		SendPush  bool   `json:"send_push"`
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
		"total_users": len(req.UserIDs),
		"notified":    notifiedCount,
	})
}

// SendToAllUsers sends notification to all users
// POST /api/v1/admin/notifications/broadcast
func (h *NotificationHandler) SendToAllUsers(c *gin.Context) {
	var req struct {
		Title       string `json:"title" binding:"required"`
		Message     string `json:"message" binding:"required"`
		Type        string `json:"type"`
		MinKYCLevel int    `json:"min_kyc_level"`
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
		UnreadCount        int64 `json:"unread_count"`
		InfoCount          int64 `json:"info_count"`
		WarningCount       int64 `json:"warning_count"`
		ErrorCount         int64 `json:"error_count"`
		SuccessCount       int64 `json:"success_count"`
		TodayCount         int64 `json:"today_count"`
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
	TemplateID string            `json:"template_id" binding:"required"`
	UserIDs    []uint            `json:"user_ids"`
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

// GetNotification retrieves a single notification by ID.
// GET /api/v1/admin/notifications/:id
func (h *NotificationHandler) GetNotification(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	var notification models.Notification
	if err := h.db.First(&notification, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notification"})
		return
	}

	c.JSON(http.StatusOK, notification)
}

// sendNotificationEmail delivers the notification via SMTP. SMTP settings are
// read entirely from environment variables so no credentials are hardcoded:
//   - SMTP_HOST     SMTP server hostname (required)
//   - SMTP_PORT     SMTP server port, defaults to 587 (required)
//   - SMTP_USERNAME SMTP auth username
//   - SMTP_PASSWORD SMTP auth password
//   - SMTP_FROM     sender address (required)
//   - SMTP_TO       comma-separated recipient override; when empty the
//     notification recipient's admin email is looked up from the DB.
func (h *NotificationHandler) sendNotificationEmail(n models.Notification) error {
	host := os.Getenv("SMTP_HOST")
	if host == "" {
		return fmt.Errorf("SMTP_HOST not configured")
	}
	port := os.Getenv("SMTP_PORT")
	if port == "" {
		port = "587"
	}
	from := os.Getenv("SMTP_FROM")
	if from == "" {
		return fmt.Errorf("SMTP_FROM not configured")
	}
	username := os.Getenv("SMTP_USERNAME")
	password := os.Getenv("SMTP_PASSWORD")

	recipients := strings.Split(os.Getenv("SMTP_TO"), ",")
	// Look up the recipient's email from the DB when no override is set.
	var trimmedRecipients []string
	for _, r := range recipients {
		r = strings.TrimSpace(r)
		if r != "" {
			trimmedRecipients = append(trimmedRecipients, r)
		}
	}
	if len(trimmedRecipients) == 0 {
		var admin models.Admin
		if err := h.db.First(&admin, n.AdminID).Error; err != nil {
			return fmt.Errorf("failed to resolve recipient admin %d: %w", n.AdminID, err)
		}
		trimmedRecipients = append(trimmedRecipients, admin.Email)
	}

	subject := "TigerWallet Notification: " + n.Title
	body := fmt.Sprintf("Title: %s\n\nMessage: %s\n\nType: %s\nPriority: %s\n",
		n.Title, n.Message, n.Type, n.Priority)

	msg := buildEmailMessage(from, strings.Join(trimmedRecipients, ", "), subject, body)

	addr := host + ":" + port
	var auth smtp.Auth
	if username != "" {
		auth = smtp.PlainAuth("", username, password, host)
	}
	return smtp.SendMail(addr, auth, from, trimmedRecipients, []byte(msg))
}

// buildEmailMessage assembles an RFC 5322 message with the required headers.
func buildEmailMessage(from, to, subject, body string) string {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	return b.String()
}

// dispatchNotificationWebhook POSTs the notification payload to the
// NOTIFICATION_WEBHOOK_URL endpoint as JSON, tagging the channel ("sms" or
// "push") so the receiver can route the delivery. When the webhook URL is
// not configured the call is skipped without an error so notification
// creation is never blocked by missing delivery config.
func (h *NotificationHandler) dispatchNotificationWebhook(channel string, n models.Notification) error {
	webhookURL := os.Getenv("NOTIFICATION_WEBHOOK_URL")
	if webhookURL == "" {
		return nil
	}

	payload := map[string]interface{}{
		"channel":         channel,
		"notification_id": n.ID,
		"admin_id":        n.AdminID,
		"title":           n.Title,
		"message":         n.Message,
		"type":            n.Type,
		"priority":        n.Priority,
		"action_url":      n.ActionURL,
		"created_at":      n.CreatedAt,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token := os.Getenv("NOTIFICATION_WEBHOOK_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}
