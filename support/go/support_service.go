// Support Ticket System
// Complete ticket management, SLA tracking, and customer support

package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// SupportConfig - Support System Configuration
type SupportConfig struct {
	// Email Settings
	EmailFrom     string `json:"email_from"`
	EmailReplyTo  string `json:"email_reply_to"`
	
	// SLA Settings
	SLAFirstResponse time.Duration `json:"sla_first_response"` // default 4 hours
	SLAResolution    time.Duration `json:"sla_resolution"`     // default 24 hours
	
	// Database Settings
	DBHost     string `json:"db_host"`
	DBPort     string `json:"db_port"`
	DBUser     string `json:"db_user"`
	DBPassword string `json:"db_password"`
	DBName     string `json:"db_name"`
	
	// Redis Settings
	RedisHost string `json:"redis_host"`
	RedisPort string `json:"redis_port"`
	
	// Server
	ServerPort string `json:"server_port"`
}

// Ticket - Support ticket
type Ticket struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	TicketID       string    `gorm:"uniqueIndex" json:"ticket_id"`
	UserID         string    `gorm:"index" json:"user_id"`
	UserEmail      string    `gorm:"index" json:"user_email"`
	UserName       string    `json:"user_name"`
	Subject        string    `json:"subject"`
	Description   string    `gorm:"type:text" json:"description"`
	Category       string    `json:"category"` // technical, billing, kyc, withdrawal, general
	Priority       string    `json:"priority"` // low, medium, high, urgent
	Status         string    `json:"status"` // open, pending, in_progress, resolved, closed
	AssignedTo     string    `gorm:"index" json:"assigned_to"`
	AssignedTeam   string    `json:"assigned_team"`
	Tags           string    `json:"tags"` // JSON array
	Channel        string    `json:"channel"` // email, chat, phone, web
	
	// SLA
	SLAFirstResponseBy *time.Time `json:"sla_first_response_by"`
	SLAResolutionBy    *time.Time `json:"sla_resolution_by"`
	FirstResponseAt    *time.Time `json:"first_response_at"`
	ResolvedAt         *time.Time `json:"resolved_at"`
	
	// Metadata
	IPAddress     string    `json:"ip_address"`
	UserAgent     string    `json:"user_agent"`
	RelatedOrder  string    `json:"related_order"`
	RelatedTx     string    `json:"related_tx"`
	
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// TicketMessage - Ticket message/reply
type TicketMessage struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	TicketID    string    `gorm:"index" json:"ticket_id"`
	MessageID   string    `gorm:"uniqueIndex" json:"message_id"`
	SenderID    string    `gorm:"index" json:"sender_id"`
	SenderType  string    `json:"sender_type"` // user, agent, system
	SenderName  string    `json:"sender_name"`
	SenderEmail string    `json:"sender_email"`
	Content     string    `gorm:"type:text" json:"content"`
	IsInternal  bool      `gorm:"default:false" json:"is_internal"`
	Attachments string    `json:"attachments"` // JSON array of file URLs
	CreatedAt   time.Time `json:"created_at"`
}

// TicketAttachment - Ticket attachment
type TicketAttachment struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	MessageID   string    `gorm:"index" json:"message_id"`
	Filename    string    `json:"filename"`
	FileURL     string    `json:"file_url"`
	FileSize    int64     `json:"file_size"`
	MimeType    string    `json:"mime_type"`
	UploadedBy  string    `json:"uploaded_by"`
	CreatedAt   time.Time `json:"created_at"`
}

// Agent - Support agent
type Agent struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      string    `gorm:"uniqueIndex" json:"user_id"`
	Email       string    `gorm:"uniqueIndex" json:"email"`
	Name        string    `json:"name"`
	Role        string    `json:"role"` // admin, lead, agent
	Team        string    `json:"team"`
	Skills      string    `json:"skills"` // JSON array
	Status      string    `json:"status"` // online, away, offline
	MaxTickets  int       `json:"max_tickets"` // max concurrent tickets
	CurrentLoad int       `json:"current_load"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AgentStatus - Agent status change log
type AgentStatus struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	AgentID   string    `gorm:"index" json:"agent_id"`
	Status    string    `json:"status"`
	Reason    string    `json:"reason"`
	ChangedAt time.Time `json:"changed_at"`
}

// CannedResponse - Canned response template
type CannedResponse struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Title       string    `json:"title"`
	Category    string    `json:"category"`
	Content     string    `gorm:"type:text" json:"content"`
	Tags        string    `json:"tags"` // JSON array
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	UsageCount  int       `gorm:"default:0" json:"usage_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// KnowledgeBaseCategory - Knowledge base category
type KnowledgeBaseCategory struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `json:"name"`
	Slug        string    `gorm:"uniqueIndex" json:"slug"`
	Description string    `json:"description"`
	ParentID    *uint     `json:"parent_id"`
	SortOrder   int       `json:"sort_order"`
	ArticleCount int      `gorm:"default:0" json:"article_count"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// KnowledgeBaseArticle - Knowledge base article
type KnowledgeBaseArticle struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CategoryID  uint      `gorm:"index" json:"category_id"`
	Title       string    `json:"title"`
	Slug        string    `gorm:"uniqueIndex" json:"slug"`
	Content     string    `gorm:"type:text" json:"content"`
	Summary     string    `json:"summary"`
	Tags        string    `json:"tags"` // JSON array
	Views       int       `gorm:"default:0" json:"views"`
	IsPublished bool      `gorm:"default:false" json:"is_published"`
	IsFeatured  bool      `gorm:"default:false" json:"is_featured"`
	AuthorID    string    `json:"author_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SupportService - Main support service
type SupportService struct {
	config  SupportConfig
	db      *gorm.DB
	redis   *redis.Client
	templates *template.Template
}

// NewSupportService - Create new support service
func NewSupportService(cfg SupportConfig) (*SupportService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	
	err = db.AutoMigrate(
		&Ticket{}, &TicketMessage{}, &TicketAttachment{},
		&Agent{}, &AgentStatus{}, &CannedResponse{},
		&KnowledgeBaseCategory{}, &KnowledgeBaseArticle{},
	)
	if err != nil {
		return nil, err
	}
	
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
	})
	
	if cfg.SLAFirstResponse == 0 {
		cfg.SLAFirstResponse = 4 * time.Hour
	}
	if cfg.SLAResolution == 0 {
		cfg.SLAResolution = 24 * time.Hour
	}
	
	// Seed canned responses
	seedCannedResponses(db)
	
	// Seed knowledge base
	seedKnowledgeBase(db)
	
	return &SupportService{
		config:    cfg,
		db:        db,
		redis:     rdb,
	}, nil
}

// seedCannedResponses - Seed default canned responses
func seedCannedResponses(db *gorm.DB) {
	responses := []CannedResponse{
		{
			Title:    "Greeting",
			Category: "general",
			Content:  "Hello! Thank you for contacting TigerWallet support. How can I help you today?",
			Tags:     `["greeting", "welcome"]`,
			IsActive: true,
		},
		{
			Title:    "Account Verification",
			Category: "kyc",
			Content:  "To verify your account, please submit the following documents:\n1. Government-issued ID (passport, driver's license)\n2. Proof of address (utility bill, bank statement)\n\nOur team typically reviews documents within 24-48 hours.",
			Tags:     `["kyc", "verification", "documents"]`,
			IsActive: true,
		},
		{
			Title:    "Withdrawal Processing",
			Category: "withdrawal",
			Content:  "Withdrawal requests are typically processed within 1-24 hours. The processing time may vary depending on:\n- Blockchain network congestion\n- Security verification\n- Amount (larger amounts may require additional verification)\n\nYou'll receive an email notification once your withdrawal is processed.",
			Tags:     `["withdrawal", "processing", "time"]`,
			IsActive: true,
		},
	}
	
	for _, r := range responses {
		var existing CannedResponse
		if db.Where("title = ?", r.Title).First(&existing).Error != nil {
			db.Create(&r)
		}
	}
}

// seedKnowledgeBase - Seed knowledge base
func seedKnowledgeBase(db *gorm.DB) {
	categories := []KnowledgeBaseCategory{
		{Name: "Getting Started", Slug: "getting-started", Description: "Learn the basics of TigerWallet"},
		{Name: "Account & Security", Slug: "account-security", Description: "Account management and security"},
		{Name: "Trading", Slug: "trading", Description: "Trading guides and tips"},
		{Name: "Deposits & Withdrawals", Slug: "deposits-withdrawals", Description: "Managing your funds"},
		{Name: "Fees & Limits", Slug: "fees-limits", Description: "Understanding fees and limits"},
	}
	
	for _, c := range categories {
		var existing KnowledgeBaseCategory
		if db.Where("slug = ?", c.Slug).First(&existing).Error != nil {
			db.Create(&c)
		}
	}
	
	articles := []KnowledgeBaseArticle{
		{
			Title:       "How to Create an Account",
			Slug:        "how-to-create-account",
			Summary:     "Step-by-step guide to creating your TigerWallet account",
			Content:     "## Creating Your Account\n\n1. Visit our website\n2. Click 'Sign Up'\n3. Enter your email and create a password\n4. Verify your email\n5. Complete KYC verification\n\nThat's it! You're ready to start trading.",
			Tags:        `["account", "signup", "getting-started"]`,
			IsPublished: true,
			IsFeatured:  true,
		},
		{
			Title:       "Two-Factor Authentication Setup",
			Slug:        "two-factor-authentication",
			Summary:     "Secure your account with 2FA",
			Content:     "## Setting Up 2FA\n\nTwo-factor authentication adds an extra layer of security to your account.\n\n### Steps:\n1. Go to Settings > Security\n2. Click 'Enable 2FA'\n3. Scan the QR code with your authenticator app\n4. Enter the verification code\n5. Save your backup codes!\n\nNever share your 2FA codes with anyone.",
			Tags:        `["security", "2fa", "authentication"]`,
			IsPublished: true,
			IsFeatured:  true,
		},
	}
	
	for _, a := range articles {
		var existing KnowledgeBaseArticle
		if db.Where("slug = ?", a.Slug).First(&existing).Error != nil {
			db.Create(&a)
		}
	}
}

// GenerateTicketID - Generate unique ticket ID
func (s *SupportService) GenerateTicketID() string {
	return fmt.Sprintf("TKT-%d-%s", time.Now().Unix(), randomString(6))
}

// GenerateMessageID - Generate unique message ID
func (s *SupportService) GenerateMessageID() string {
	return fmt.Sprintf("MSG-%d-%s", time.Now().Unix(), randomString(6))
}

// CreateTicket - Create new ticket
func (s *SupportService) CreateTicket(userID, userEmail, userName, subject, description, category, priority, channel, ipAddress, userAgent string) (*Ticket, error) {
	ticketID := s.GenerateTicketID()
	
	// Set priority
	if priority == "" {
		priority = "medium"
	}
	
	// Set category
	if category == "" {
		category = "general"
	}
	
	// Calculate SLA
	now := time.Now()
	slaFirstResponse := now.Add(s.config.SLAFirstResponse)
	slaResolution := now.Add(s.config.SLAResolution)
	
	// Upgrade priority based on keywords
	priority = s.calculatePriority(subject, description, priority)
	
	ticket := &Ticket{
		TicketID:           ticketID,
		UserID:             userID,
		UserEmail:          userEmail,
		UserName:           userName,
		Subject:            subject,
		Description:        description,
		Category:           category,
		Priority:           priority,
		Status:             "open",
		Channel:            channel,
		IPAddress:          ipAddress,
		UserAgent:          userAgent,
		SLAFirstResponseBy: &slaFirstResponse,
		SLAResolutionBy:    &slaResolution,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	
	err := s.db.Create(ticket).Error
	if err != nil {
		return nil, err
	}
	
	// Create initial message
	s.CreateMessage(ticketID, userID, userName, userEmail, description, "user")
	
	// Cache ticket
	s.cacheTicket(ticket)
	
	return ticket, nil
}

// calculatePriority - Auto-calculate priority based on content
func (s *SupportService) calculatePriority(subject, description, currentPriority string) string {
	text := strings.ToLower(subject + " " + description)
	
	urgentKeywords := []string{"urgent", "emergency", "lost", "hacked", "stolen", "money gone", "can't access"}
	highKeywords := []string{"withdrawal", "can't withdraw", "pending", "bug", "error", "not working"}
	
	for _, kw := range urgentKeywords {
		if strings.Contains(text, kw) {
			return "urgent"
		}
	}
	
	if currentPriority == "low" {
		for _, kw := range highKeywords {
			if strings.Contains(text, kw) {
				return "high"
			}
		}
	}
	
	return currentPriority
}

// CreateMessage - Add message to ticket
func (s *SupportService) CreateMessage(ticketID, senderID, senderName, senderEmail, content string, senderType string) (*TicketMessage, error) {
	messageID := s.GenerateMessageID()
	
	message := &TicketMessage{
		TicketID:    ticketID,
		MessageID:   messageID,
		SenderID:    senderID,
		SenderType:  senderType,
		SenderName:  senderName,
		SenderEmail: senderEmail,
		Content:     content,
		CreatedAt:   time.Now(),
	}
	
	err := s.db.Create(message).Error
	if err != nil {
		return nil, err
	}
	
	// Update ticket first response time if first agent response
	if senderType == "agent" {
		var ticket Ticket
		if s.db.Where("ticket_id = ?", ticketID).First(&ticket).Error == nil {
			if ticket.FirstResponseAt == nil {
				now := time.Now()
				s.db.Model(&ticket).Update("first_response_at", now)
			}
		}
	}
	
	// Update ticket timestamp
	s.db.Model(&Ticket{}).Where("ticket_id = ?", ticketID).Update("updated_at", time.Now())
	
	return message, nil
}

// UpdateTicketStatus - Update ticket status
func (s *SupportService) UpdateTicketStatus(ticketID, status, assignedTo string) error {
	updates := map[string]interface{}{
		"status":      status,
		"assigned_to": assignedTo,
		"updated_at":  time.Now(),
	}
	
	if status == "resolved" || status == "closed" {
		now := time.Now()
		updates["resolved_at"] = now
	}
	
	return s.db.Model(&Ticket{}).Where("ticket_id = ?", ticketID).Updates(updates).Error
}

// AssignTicket - Assign ticket to agent
func (s *SupportService) AssignTicket(ticketID, agentID, team string) error {
	return s.db.Model(&Ticket{}).Where("ticket_id = ?", ticketID).Updates(map[string]interface{}{
		"assigned_to":   agentID,
		"assigned_team": team,
		"status":        "in_progress",
		"updated_at":    time.Now(),
	}).Error
}

// GetTicket - Get ticket by ID
func (s *SupportService) GetTicket(ticketID string) (*Ticket, error) {
	var ticket Ticket
	err := s.db.Where("ticket_id = ?", ticketID).First(&ticket).Error
	return &ticket, err
}

// GetTicketMessages - Get ticket messages
func (s *SupportService) GetTicketMessages(ticketID string) ([]TicketMessage, error) {
	var messages []TicketMessage
	err := s.db.Where("ticket_id = ?", ticketID).Order("created_at ASC").Find(&messages).Error
	return messages, err
}

// GetUserTickets - Get user tickets
func (s *SupportService) GetUserTickets(userID string) ([]Ticket, error) {
	var tickets []Ticket
	err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&tickets).Error
	return tickets, err
}

// GetOpenTickets - Get open tickets
func (s *SupportService) GetOpenTickets(assignedTo, status string) ([]Ticket, error) {
	query := s.db.Where("1=1")
	
	if assignedTo != "" {
		query = query.Where("assigned_to = ?", assignedTo)
	}
	
	if status != "" {
		query = query.Where("status = ?", status)
	} else {
		query = query.Where("status IN ?", []string{"open", "pending", "in_progress"})
	}
	
	var tickets []Ticket
	err := query.Order("priority DESC, created_at ASC").Find(&tickets).Error
	return tickets, err
}

// GetCannedResponses - Get canned responses
func (s *SupportService) GetCannedResponses(category string) ([]CannedResponse, error) {
	query := s.db.Where("is_active = ?", true)
	
	if category != "" {
		query = query.Where("category = ?", category)
	}
	
	var responses []CannedResponse
	err := query.Order("usage_count DESC").Find(&responses).Error
	return responses, err
}

// UseCannedResponse - Use canned response and increment counter
func (s *SupportService) UseCannedResponse(responseID uint) error {
	return s.db.Model(&CannedResponse{}).Where("id = ?", responseID).Update("usage_count", gorm.Expr("usage_count + 1")).Error
}

// GetKnowledgeBaseCategories - Get knowledge base categories
func (s *SupportService) GetKnowledgeBaseCategories() ([]KnowledgeBaseCategory, error) {
	var categories []KnowledgeBaseCategory
	err := s.db.Where("is_active = ?", true).Order("sort_order ASC").Find(&categories).Error
	return categories, err
}

// GetKnowledgeBaseArticles - Get knowledge base articles
func (s *SupportService) GetKnowledgeBaseArticles(categoryID uint, search string) ([]KnowledgeBaseArticle, error) {
	query := s.db.Where("is_published = ?", true)
	
	if categoryID > 0 {
		query = query.Where("category_id = ?", categoryID)
	}
	
	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where("title LIKE ? OR content LIKE ? OR tags LIKE ?", searchPattern, searchPattern, searchPattern)
	}
	
	var articles []KnowledgeBaseArticle
	err := query.Order("is_featured DESC, views DESC, created_at DESC").Find(&articles).Error
	return articles, err
}

// GetKnowledgeBaseArticle - Get single article
func (s *SupportService) GetKnowledgeBaseArticle(slug string) (*KnowledgeBaseArticle, error) {
	var article KnowledgeBaseArticle
	err := s.db.Where("slug = ? AND is_published = ?", slug, true).First(&article).Error
	if err == nil {
		// Increment view count
		s.db.Model(&article).Update("views", gorm.Expr("views + 1"))
	}
	return &article, err
}

// GetSLAStatus - Get SLA status for ticket
func (s *SupportService) GetSLAStatus(ticketID string) (map[string]interface{}, error) {
	ticket, err := s.GetTicket(ticketID)
	if err != nil {
		return nil, err
	}
	
	now := time.Now()
	status := map[string]interface{}{
		"ticket_id": ticket.TicketID,
		"status":    ticket.Status,
	}
	
	// First response SLA
	if ticket.FirstResponseAt != nil {
		status["first_response"] = map[string]interface{}{
			"met":       true,
			"at":        ticket.FirstResponseAt,
			"required":  ticket.SLAFirstResponseBy,
		}
	} else if ticket.SLAFirstResponseBy != nil && now.After(*ticket.SLAFirstResponseBy) {
		status["first_response"] = map[string]interface{}{
			"met":      false,
			"breached": true,
			"by":       now.Sub(*ticket.SLAFirstResponseBy),
		}
	} else if ticket.SLAFirstResponseBy != nil {
		status["first_response"] = map[string]interface{}{
			"met":         false,
			"due":         ticket.SLAFirstResponseBy,
			"time_remaining": time.Until(*ticket.SLAFirstResponseBy),
		}
	}
	
	// Resolution SLA
	if ticket.ResolvedAt != nil {
		status["resolution"] = map[string]interface{}{
			"met":      true,
			"at":       ticket.ResolvedAt,
			"required": ticket.SLAResolutionBy,
		}
	} else if ticket.SLAResolutionBy != nil && now.After(*ticket.SLAResolutionBy) {
		status["resolution"] = map[string]interface{}{
			"met":      false,
			"breached": true,
			"by":       now.Sub(*ticket.SLAResolutionBy),
		}
	} else if ticket.SLAResolutionBy != nil {
		status["resolution"] = map[string]interface{}{
			"met":             false,
			"due":             ticket.SLAResolutionBy,
			"time_remaining":  time.Until(*ticket.SLAResolutionBy),
		}
	}
	
	return status, nil
}

// cacheTicket - Cache ticket in Redis
func (s *SupportService) cacheTicket(ticket *Ticket) {
	// Implementation would cache ticket in Redis
}

// HTTP Handlers

type CreateTicketRequest struct {
	UserID      string `json:"user_id" binding:"required"`
	UserEmail   string `json:"user_email" binding:"required,email"`
	UserName    string `json:"user_name"`
	Subject     string `json:"subject" binding:"required"`
	Description string `json:"description" binding:"required"`
	Category    string `json:"category"`
	Priority    string `json:"priority"`
	Channel     string `json:"channel"`
}

func (s *SupportService) CreateTicketHandler(c *gin.Context) {
	var req CreateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	if req.Channel == "" {
		req.Channel = "web"
	}
	
	channel := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	
	ticket, err := s.CreateTicket(
		req.UserID, req.UserEmail, req.UserName,
		req.Subject, req.Description, req.Category, req.Priority,
		req.Channel, channel, userAgent,
	)
	
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(201, ticket)
}

type AddMessageRequest struct {
	TicketID    string `json:"ticket_id" binding:"required"`
	SenderID    string `json:"sender_id" binding:"required"`
	SenderName  string `json:"sender_name"`
	SenderEmail string `json:"sender_email" binding:"required"`
	Content     string `json:"content" binding:"required"`
	SenderType  string `json:"sender_type"` // user, agent
	IsInternal  bool   `json:"is_internal"`
}

func (s *SupportService) AddMessageHandler(c *gin.Context) {
	var req AddMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	senderType := req.SenderType
	if senderType == "" {
		senderType = "user"
	}
	
	message, err := s.CreateMessage(req.TicketID, req.SenderID, req.SenderName, req.SenderEmail, req.Content, senderType)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(201, message)
}

type UpdateStatusRequest struct {
	TicketID   string `json:"ticket_id" binding:"required"`
	Status     string `json:"status" binding:"required"`
	AssignedTo string `json:"assigned_to"`
}

func (s *SupportService) UpdateStatusHandler(c *gin.Context) {
	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	err := s.UpdateTicketStatus(req.TicketID, req.Status, req.AssignedTo)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"status": "updated"})
}

func (s *SupportService) GetTicketHandler(c *gin.Context) {
	ticketID := c.Param("ticket_id")
	
	ticket, err := s.GetTicket(ticketID)
	if err != nil {
		c.JSON(404, gin.H{"error": "ticket not found"})
		return
	}
	
	messages, _ := s.GetTicketMessages(ticketID)
	
	c.JSON(200, gin.H{
		"ticket":   ticket,
		"messages":  messages,
	})
}

func (s *SupportService) GetUserTicketsHandler(c *gin.Context) {
	userID := c.Param("user_id")
	
	tickets, err := s.GetUserTickets(userID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"tickets": tickets})
}

func (s *SupportService) GetOpenTicketsHandler(c *gin.Context) {
	assignedTo := c.Query("assigned_to")
	status := c.Query("status")
	
	tickets, err := s.GetOpenTickets(assignedTo, status)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"tickets": tickets})
}

func (s *SupportService) GetCannedResponsesHandler(c *gin.Context) {
	category := c.Query("category")
	
	responses, err := s.GetCannedResponses(category)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"responses": responses})
}

func (s *SupportService) GetKnowledgeBaseHandler(c *gin.Context) {
	categories, _ := s.GetKnowledgeBaseCategories()
	
	c.JSON(200, gin.H{
		"categories": categories,
	})
}

func (s *SupportService) SearchKnowledgeBaseHandler(c *gin.Context) {
	categoryID := c.Query("category_id")
	search := c.Query("search")
	
	var catID uint
	fmt.Sscanf(categoryID, "%d", &catID)
	
	articles, err := s.GetKnowledgeBaseArticles(catID, search)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"articles": articles})
}

func (s *SupportService) GetArticleHandler(c *gin.Context) {
	slug := c.Param("slug")
	
	article, err := s.GetKnowledgeBaseArticle(slug)
	if err != nil {
		c.JSON(404, gin.H{"error": "article not found"})
		return
	}
	
	c.JSON(200, article)
}

func (s *SupportService) GetSLAStatusHandler(c *gin.Context) {
	ticketID := c.Param("ticket_id")
	
	status, err := s.GetSLAStatus(ticketID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, status)
}

// Utility

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

// Main

func main() {
	cfg := SupportConfig{
		EmailFrom:        getEnv("SUPPORT_EMAIL_FROM", "support@tigerwallet.com"),
		EmailReplyTo:    getEnv("SUPPORT_EMAIL_REPLY_TO", "support@tigerwallet.com"),
		SLAFirstResponse: getEnvDuration("SLA_FIRST_RESPONSE", 4*time.Hour),
		SLAResolution:    getEnvDuration("SLA_RESOLUTION", 24*time.Hour),
		DBHost:          getEnv("DB_HOST", "localhost"),
		DBPort:          getEnv("DB_PORT", "5432"),
		DBUser:          getEnv("DB_USER", "postgres"),
		DBPassword:      getEnv("DB_PASSWORD", "password"),
		DBName:          getEnv("DB_NAME", "support_db"),
		RedisHost:       getEnv("REDIS_HOST", "localhost"),
		RedisPort:       getEnv("REDIS_PORT", "6379"),
		ServerPort:      getEnv("SUPPORT_SERVER_PORT", "8093"),
	}
	
	service, err := NewSupportService(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize support service: %v", err)
	}
	
	r := gin.Default()
	
	r.POST("/tickets", service.CreateTicketHandler)
	r.GET("/tickets/:ticket_id", service.GetTicketHandler)
	r.GET("/tickets/user/:user_id", service.GetUserTicketsHandler)
	r.GET("/tickets/open", service.GetOpenTicketsHandler)
	r.POST("/tickets/:ticket_id/messages", service.AddMessageHandler)
	r.PUT("/tickets/:ticket_id/status", service.UpdateStatusHandler)
	r.GET("/tickets/:ticket_id/sla", service.GetSLAStatusHandler)
	
	r.GET("/canned-responses", service.GetCannedResponsesHandler)
	
	r.GET("/knowledgebase", service.GetKnowledgeBaseHandler)
	r.GET("/knowledgebase/search", service.SearchKnowledgeBaseHandler)
	r.GET("/knowledgebase/articles/:slug", service.GetArticleHandler)
	
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "support"})
	})
	
	log.Printf("Support Service starting on port %s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		d, err := time.ParseDuration(value)
		if err == nil {
			return d
		}
	}
	return defaultValue
}

// Need to import gorm
import "gorm.io/gorm"
