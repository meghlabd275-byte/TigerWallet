package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"admin_backend/internal/models"
	"admin_backend/pkg/database"

	"github.com/gin-gonic/gin"
)

// SupportHandler handles support ticket operations
type SupportHandler struct {
	db *database.PostgresDB
}

// NewSupportHandler creates a new support handler
func NewSupportHandler(db *database.PostgresDB) *SupportHandler {
	return &SupportHandler{db: db}
}

// TicketRequest represents ticket creation request
type TicketRequest struct {
	UserID      uint     `json:"user_id" binding:"required"`
	UserEmail   string   `json:"user_email" binding:"required"`
	UserName    string   `json:"user_name"`
	Subject     string   `json:"subject" binding:"required"`
	Description string   `json:"description" binding:"required"`
	Category    string   `json:"category" binding:"required"`
	Priority    string   `json:"priority"`
	RelatedTx   string   `json:"related_tx"`
}

// MessageRequest represents ticket message request
type MessageRequest struct {
	Content      string `json:"content" binding:"required"`
	IsInternal   bool   `json:"is_internal"`
	Attachments  string `json:"attachments"`
}

// CreateTicket creates a new support ticket
// POST /api/v1/admin/support/tickets
func (h *SupportHandler) CreateTicket(c *gin.Context) {
	var req TicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	priority := req.Priority
	if priority == "" {
		priority = "medium"
	}

	// Generate ticket ID
	ticketID := fmt.Sprintf("TKT-%d-%s", time.Now().Unix(), generateRandomString(6))

	ticket := models.SupportTicket{
		TicketID:    ticketID,
		UserID:      req.UserID,
		Subject:     req.Subject,
		Description: req.Description,
		Category:    req.Category,
		Priority:    priority,
		Status:      "open",
		RelatedTx:   req.RelatedTx,
	}

	// Try to find user
	var user models.User
	if err := h.db.First(&user, req.UserID).Error; err == nil {
		ticket.UserName = user.Email
	}

	if err := h.db.Create(&ticket).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create ticket"})
		return
	}

	// Set SLA times
	slaFirstResponse := time.Now().Add(4 * time.Hour)
	slaResolution := time.Now().Add(24 * time.Hour)
	ticket.SLAFirstResponseBy = &slaFirstResponse
	ticket.SLAResolutionBy = &slaResolution
	h.db.Model(&ticket).Updates(map[string]interface{}{
		"sla_first_response_by": slaFirstResponse,
		"sla_resolution_by":    slaResolution,
	})

	// Create initial message
	message := models.SupportTicketMessage{
		TicketID:   ticket.ID,
		SenderID:   req.UserID,
		SenderType: "user",
		Message:    req.Description,
	}

	if req.UserName != "" {
		message.SenderName = req.UserName
	}

	h.db.Create(&message)

	adminID := c.GetUint("admin_id")
	logAdminActivity(h.db, adminID, "create_ticket", "support_ticket", 
		fmt.Sprintf("%d", ticket.ID), "Created ticket: "+ticketID, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusCreated, ticket)
}

// GetTicket gets a ticket by ID
// GET /api/v1/admin/support/tickets/:id
func (h *SupportHandler) GetTicket(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticket ID"})
		return
	}

	var ticket models.SupportTicket
	if err := h.db.Preload("User").First(&ticket, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	// Get messages
	var messages []models.SupportTicketMessage
	h.db.Where("ticket_id = ?", ticket.ID).Order("created_at ASC").Find(&messages)

	c.JSON(http.StatusOK, gin.H{
		"ticket":   ticket,
		"messages": messages,
	})
}

// ListTickets lists all tickets with filtering
// GET /api/v1/admin/support/tickets
func (h *SupportHandler) ListTickets(c *gin.Context) {
	status := c.Query("status")
	category := c.Query("category")
	priority := c.Query("priority")
	assignedTo := c.Query("assigned_to")
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var tickets []models.SupportTicket
	var total int64

	query := h.db.Model(&models.SupportTicket{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if priority != "" {
		query = query.Where("priority = ?", priority)
	}
	if assignedTo != "" {
		query = query.Where("assigned_to = ?", assignedTo)
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC")

	if err := query.Find(&tickets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tickets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        tickets,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

// UpdateTicket updates a ticket
// PUT /api/v1/admin/support/tickets/:id
func (h *SupportHandler) UpdateTicket(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticket ID"})
		return
	}

	var req struct {
		Status      string `json:"status"`
		Priority    string `json:"priority"`
		AssignedTo  uint   `json:"assigned_to"`
		Category    string `json:"category"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var ticket models.SupportTicket
	if err := h.db.First(&ticket, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	updates := map[string]interface{}{}

	if req.Status != "" {
		updates["status"] = req.Status
		if req.Status == "resolved" || req.Status == "closed" {
			now := time.Now()
			updates["resolved_at"] = now
		}
	}
	if req.Priority != "" {
		updates["priority"] = req.Priority
	}
	if req.Category != "" {
		updates["category"] = req.Category
	}
	if req.AssignedTo > 0 {
		updates["assigned_to"] = req.AssignedTo
	}

	// Check for first response SLA
	if ticket.FirstResponseAt == nil && req.AssignedTo > 0 {
		now := time.Now()
		updates["first_response_at"] = now
	}

	if err := h.db.Model(&ticket).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update ticket"})
		return
	}

	adminID := c.GetUint("admin_id")
	logAdminActivity(h.db, adminID, "update_ticket", "support_ticket", 
		fmt.Sprintf("%d", id), "Updated ticket status to: "+req.Status, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, ticket)
}

// AddMessage adds a message to a ticket
// POST /api/v1/admin/support/tickets/:id/messages
func (h *SupportHandler) AddMessage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticket ID"})
		return
	}

	var req MessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var ticket models.SupportTicket
	if err := h.db.First(&ticket, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	adminID := c.GetUint("admin_id")
	admin, _ := h.db.First(&models.Admin{}, adminID)

	message := models.SupportTicketMessage{
		TicketID:    ticket.ID,
		SenderID:    adminID,
		SenderType:  "admin",
		SenderName:  admin.Username,
		Message:     req.Content,
		IsInternal:  req.IsInternal,
		Attachments: req.Attachments,
	}

	if err := h.db.Create(&message).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add message"})
		return
	}

	// Update first response time if this is first admin response
	if ticket.FirstResponseAt == nil {
		now := time.Now()
		h.db.Model(&ticket).Update("first_response_at", now)
	}

	// Update ticket status to in_progress if it was open
	if ticket.Status == "open" {
		h.db.Model(&ticket).Update("status", "in_progress")
	}

	c.JSON(http.StatusCreated, message)
}

// CloseTicket closes a ticket
// POST /api/v1/admin/support/tickets/:id/close
func (h *SupportHandler) CloseTicket(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticket ID"})
		return
	}

	var ticket models.SupportTicket
	if err := h.db.First(&ticket, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":     "closed",
		"closed_at":  now,
		"resolved_at": now,
	}

	if err := h.db.Model(&ticket).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to close ticket"})
		return
	}

	adminID := c.GetUint("admin_id")
	logAdminActivity(h.db, adminID, "close_ticket", "support_ticket", 
		fmt.Sprintf("%d", id), "Closed ticket: "+ticket.TicketID, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Ticket closed successfully"})
}

// GetTicketStats gets ticket statistics
// GET /api/v1/admin/support/stats
func (h *SupportHandler) GetTicketStats(c *gin.Context) {
	var stats struct {
		TotalTickets     int64 `json:"total_tickets"`
		OpenTickets     int64 `json:"open_tickets"`
		InProgressTickets int64 `json:"in_progress_tickets"`
		ResolvedTickets int64 `json:"resolved_tickets"`
		ClosedTickets   int64 `json:"closed_tickets"`
		HighPriority    int64 `json:"high_priority"`
		UrgentPriority int64 `json:"urgent_priority"`
		BreachedSLA     int64 `json:"breached_sla"`
		AvgResponseTime float64 `json:"avg_response_time"`
	}

	h.db.Model(&models.SupportTicket{}).Count(&stats.TotalTickets)
	h.db.Model(&models.SupportTicket{}).Where("status = ?", "open").Count(&stats.OpenTickets)
	h.db.Model(&models.SupportTicket{}).Where("status = ?", "in_progress").Count(&stats.InProgressTickets)
	h.db.Model(&models.SupportTicket{}).Where("status = ?", "resolved").Count(&stats.ResolvedTickets)
	h.db.Model(&models.SupportTicket{}).Where("status = ?", "closed").Count(&stats.ClosedTickets)
	h.db.Model(&models.SupportTicket{}).Where("priority IN ?", []string{"high", "urgent"}).Count(&stats.HighPriority)

	// Calculate breached SLA
	var tickets []models.SupportTicket
	h.db.Find(&tickets)
	breached := int64(0)
	for _, t := range tickets {
		if t.SLAResolutionBy != nil && t.SLAResolutionBy.Before(time.Now()) && t.Status != "resolved" && t.Status != "closed" {
			breached++
		}
	}
	stats.BreachedSLA = breached

	// Calculate average response time
	var totalResponseTime int64
	var respondedCount int64
	h.db.Model(&models.SupportTicket{}).Where("first_response_at IS NOT NULL").Count(&respondedCount)
	if respondedCount > 0 {
		h.db.Raw("SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (first_response_at - created_at))), 0) FROM support_tickets WHERE first_response_at IS NOT NULL").Scan(&totalResponseTime)
		stats.AvgResponseTime = float64(totalResponseTime) / 3600 // Convert to hours
	}

	c.JSON(http.StatusOK, stats)
}

// GetSLAViolations gets tickets with SLA violations
// GET /api/v1/admin/support/sla-violations
func (h *SupportHandler) GetSLAViolations(c *gin.Context) {
	var tickets []models.SupportTicket
	
	err := h.db.Where("status NOT IN ? AND sla_resolution_by < ?", 
		[]string{"resolved", "closed"}, time.Now()).Find(&tickets).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch SLA violations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":  len(tickets),
		"tickets": tickets,
	})
}

// helper function
func generateRandomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
