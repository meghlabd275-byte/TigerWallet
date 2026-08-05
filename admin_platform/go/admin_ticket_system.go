/**
 * TigerWallet Admin Support Ticket System
 * Complete Support Ticket Management with Knowledge Base
 * 
 * Features:
 * - Ticket creation and management
 * - Ticket assignment and escalation
 * - Status tracking
 * - Priority levels
 * - Internal notes
 * - Customer communication
 * - Knowledge base integration
 * - Canned responses
 * - SLA tracking
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ============================================================================
// Configuration
// ============================================================================

type TicketConfig struct {
	Port       string
	RedisURL   string
	DBURL      string
	SLAHours   int // Default SLA in hours
}

func LoadTicketConfig() *TicketConfig {
	return &TicketConfig{
		Port:     getEnv("TICKET_PORT", "9099"),
		RedisURL: getEnv("REDIS_TICKET_URL", "redis://localhost:6379"),
		DBURL:   getEnv("TICKET_DB_URL", "postgres://tigerwallet:password@localhost:5432/tigerwallet"),
		SLAHours: getEnvInt("SLA_HOURS", 24),
	}
}

// ============================================================================
// Types
// ============================================================================

type TicketStatus string

const (
	TicketStatusOpen       TicketStatus = "open"
	TicketStatusInProgress TicketStatus = "in_progress"
	TicketStatusPending   TicketStatus = "pending"
	TicketStatusResolved  TicketStatus = "resolved"
	TicketStatusClosed   TicketStatus = "closed"
)

type TicketPriority string

const (
	TicketPriorityLow      TicketPriority = "low"
	TicketPriorityMedium   TicketPriority = "medium"
	TicketPriorityHigh    TicketPriority = "high"
	TicketPriorityUrgent  TicketPriority = "urgent"
)

type TicketCategory string

const (
	TicketCategoryGeneral      TicketCategory = "general"
	TicketCategoryTechnical   TicketCategory = "technical"
	TicketCategoryBilling    TicketCategory = "billing"
	TicketCategoryKYC       TicketCategory = "kyc"
	TicketCategoryWithdrawal TicketCategory = "withdrawal"
	TicketCategorySecurity  TicketCategory = "security"
	TicketCategoryFeature   TicketCategory = "feature_request"
)

type Ticket struct {
	ID            string         `json:"id"`
	Subject       string        `json:"subject"`
	Description   string        `json:"description"`
	Category     TicketCategory `json:"category"`
	Priority     TicketPriority `json:"priority"`
	Status       TicketStatus   `json:"status"`
	CreatedBy   string        `json:"created_by"`
	AssignedTo   string        `json:"assigned_to"`
	AssignedAt   *time.Time   `json:"assigned_at,omitempty"`
	ResolvedAt   *time.Time   `json:"resolved_at,omitempty"`
	ClosedAt     *time.Time   `json:"closed_at,omitempty"`
	DueAt       *time.Time   `json:"due_at,omitempty"`
	Tags         []string     `json:"tags"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

type TicketComment struct {
	ID          string    `json:"id"`
	TicketID    string    `json:"ticket_id"`
	AuthorID    string    `json:"author_id"`
	AuthorType  string    `json:"author_type"` // admin, user, system
	Content     string    `json:"content"`
	IsInternal  bool     `json:"is_internal"`
	CreatedAt   time.Time `json:"created_at"`
}

type TicketAttachment struct {
	ID          string    `json:"id"`
	TicketID    string    `json:"ticket_id"`
	Filename    string    `json:"filename"`
	FileURL    string    `json:"file_url"`
	FileSize   int64     `json:"file_size"`
	MimeType   string    `json:"mime_type"`
	UploadedBy string    `json:"uploaded_by"`
	CreatedAt  time.Time `json:"created_at"`
}

type KnowledgeBaseArticle struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Content     string   `json:"content"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
	ViewCount   int      `json:"view_count"`
	HelpfulYes  int      `json:"helpful_yes"`
	HelpfulNo   int      `json:"helpful_no"`
	Status      string   `json:"status"` // draft, published, archived
	CreatedBy   string   `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CannedResponse struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	Category  string   `json:"category"`
	Tags      []string `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
}

type SLATracking struct {
	TicketID      string     `json:"ticket_id"`
	Priority      string     `json:"priority"`
	ResponseDue   time.Time `json:"response_due"`
	ResolutionDue time.Time `json:"resolution_due"`
	FirstResponse *time.Time `json:"first_response_at"`
	ResolvedAt    *time.Time `json:"resolved_at"`
	Status        string     `json:"status"` // on_track, at_risk, breached
}

// ============================================================================
// Ticket Service
// ============================================================================

type TicketService struct {
	config  *TicketConfig
	db      *gorm.DB
	redis   *redis.Client
}

func NewTicketService(config *TicketConfig, db *gorm.DB, redisClient *redis.Client) *TicketService {
	return &TicketService{
		config: config,
		db:    db,
		redis:  redisClient,
	}
}

// ============================================================================
// Ticket CRUD Operations
// ============================================================================

func (s *TicketService) CreateTicket(ticket *Ticket) error {
	ticket.ID = uuid.New().String()
	ticket.Status = TicketStatusOpen
	ticket.CreatedAt = time.Now()
	ticket.UpdatedAt = time.Now()
	
	// Calculate SLA
	dueAt := time.Now().Add(time.Duration(s.config.SLAHours) * time.Hour)
	if ticket.Priority == TicketPriorityUrgent {
		dueAt = time.Now().Add(4 * time.Hour)
	} else if ticket.Priority == TicketPriorityHigh {
		dueAt = time.Now().Add(8 * time.Hour)
	}
	ticket.DueAt = &dueAt
	
	// Store in Redis for fast access
	ticketJSON, _ := json.Marshal(ticket)
	s.redis.Set(context.Background(), "ticket:"+ticket.ID, ticketJSON, 0)
	
	// Also store in list
	s.redis.ZAdd(context.Background(), "tickets:open", &redis.Z{
		Score: float64(ticket.CreatedAt.Unix()),
		Member: ticket.ID,
	})
	
	// Update counts
	s.redis.Incr(context.Background(), "tickets:count:open")
	s.redis.Incr(context.Background(), "tickets:count:priority:"+string(ticket.Priority))
	
	return nil
}

func (s *TicketService) GetTicket(id string) (*Ticket, error) {
	ticketJSON, err := s.redis.Get(context.Background(), "ticket:"+id).Result()
	if err != nil {
		return nil, err
	}
	
	var ticket Ticket
	json.Unmarshal([]byte(ticketJSON), &ticket)
	
	return &ticket, nil
}

func (s *TicketService) UpdateTicket(ticket *Ticket) error {
	ticket.UpdatedAt = time.Now()
	
	ticketJSON, _ := json.Marshal(ticket)
	s.redis.Set(context.Background(), "ticket:"+ticket.ID, ticketJSON, 0)
	
	return nil
}

func (s *TicketService) DeleteTicket(id string) error {
	s.redis.Del(context.Background(), "ticket:"+id)
	s.redis.ZRem(context.Background(), "tickets:all", id)
	s.redis.ZRem(context.Background(), "tickets:open", id)
	
	return nil
}

func (s *TicketService) ListTickets(filters map[string]string) ([]*Ticket, int64, error) {
	// Get from Redis sorted set
	var ticketIDs []string
	
	status := filters["status"]
	if status != "" {
		ticketIDs, _ = s.redis.ZRange(context.Background(), "tickets:"+status, 0, -1).Result()
	} else {
		ticketIDs, _ = s.redis.ZRange(context.Background(), "tickets:all", 0, -1).Result()
	}
	
	tickets := make([]*Ticket, 0, len(ticketIDs))
	
	page, _ := strconv.Atoi(filters["page"])
	pageSize, _ := strconv.Atoi(filters["page_size"])
	if pageSize == 0 {
		pageSize = 20
	}
	
	start := (page - 1) * pageSize
	end := start + pageSize
	
	if start >= len(ticketIDs) {
		return tickets, int64(len(ticketIDs)), nil
	}
	
	if end > len(ticketIDs) {
		end = len(ticketIDs)
	}
	
	for _, id := range ticketIDs[start:end] {
		if ticket, err := s.GetTicket(id); err == nil {
			// Apply filters
			if priority, ok := filters["priority"]; ok && string(ticket.Priority) != priority {
				continue
			}
			if category, ok := filters["category"]; ok && string(ticket.Category) != category {
				continue
			}
			if assignedTo, ok := filters["assigned_to"]; ok && ticket.AssignedTo != assignedTo {
				continue
			}
			tickets = append(tickets, ticket)
		}
	}
	
	return tickets, int64(len(ticketIDs)), nil
}

// ============================================================================
// Ticket Comments
// ============================================================================

func (s *TicketService) AddComment(comment *TicketComment) error {
	comment.ID = uuid.New().String()
	comment.CreatedAt = time.Now()
	
	commentJSON, _ := json.Marshal(comment)
	s.redis.RPush(context.Background(), "ticket:"+comment.TicketID+":comments", commentJSON)
	
	// Update ticket timestamp
	if ticket, err := s.GetTicket(comment.TicketID); err == nil {
		ticket.UpdatedAt = time.Now()
		if comment.AuthorType == "admin" && ticket.Status == TicketStatusOpen {
			ticket.Status = TicketStatusInProgress
		}
		s.UpdateTicket(ticket)
	}
	
	return nil
}

func (s *TicketService) GetComments(ticketID string) ([]*TicketComment, error) {
	commentsJSON, err := s.redis.LRange(context.Background(), "ticket:"+ticketID+":comments", 0, -1).Result()
	if err != nil {
		return nil, err
	}
	
	comments := make([]*TicketComment, 0, len(commentsJSON))
	for _, c := range commentsJSON {
		var comment TicketComment
		json.Unmarshal([]byte(c), &comment)
		comments = append(comments, &comment)
	}
	
	return comments, nil
}

// ============================================================================
// Knowledge Base
// ============================================================================

func (s *TicketService) CreateArticle(article *KnowledgeBaseArticle) error {
	article.ID = uuid.New().String()
	article.Status = "draft"
	article.CreatedAt = time.Now()
	article.UpdatedAt = time.Now()
	
	articleJSON, _ := json.Marshal(article)
	s.redis.Set(context.Background(), "kb:article:"+article.ID, articleJSON, 0)
	
	// Add to category index
	s.redis.ZAdd(context.Background(), "kb:articles:"+article.Category, &redis.Z{
		Score: float64(article.CreatedAt.Unix()),
		Member: article.ID,
	})
	
	return nil
}

func (s *TicketService) GetArticle(id string) (*KnowledgeBaseArticle, error) {
	articleJSON, err := s.redis.Get(context.Background(), "kb:article:"+id).Result()
	if err != nil {
		return nil, err
	}
	
	var article KnowledgeBaseArticle
	json.Unmarshal([]byte(articleJSON), &article)
	
	// Increment view count
	s.redis.Incr(context.Background(), "kb:article:"+id+":views")
	
	return &article, nil
}

func (s *TicketService) UpdateArticle(article *KnowledgeBaseArticle) error {
	article.UpdatedAt = time.Now()
	
	articleJSON, _ := json.Marshal(article)
	s.redis.Set(context.Background(), "kb:article:"+article.ID, articleJSON, 0)
	
	return nil
}

func (s *TicketService) SearchArticles(query string) ([]*KnowledgeBaseArticle, error) {
	// Search across all categories
	categories := []string{"general", "technical", "billing", "security", "kyc", "withdrawal"}
	
	results := make([]*KnowledgeBaseArticle, 0)
	
	for _, category := range categories {
		articleIDs, _ := s.redis.ZRange(context.Background(), "kb:articles:"+category, 0, -1).Result()
		
		for _, id := range articleIDs {
			if article, err := s.GetArticle(id); err == nil {
				if article.Status == "published" {
					// Simple text search
					searchStr := strings.ToLower(article.Title + " " + article.Content)
					if strings.Contains(searchStr, strings.ToLower(query)) {
						results = append(results, article)
					}
				}
			}
		}
	}
	
	return results, nil
}

func (s *TicketService) ArticleFeedback(articleID, helpful string) error {
	if helpful == "yes" {
		s.redis.Incr(context.Background(), "kb:article:"+articleID+":helpful_yes")
	} else {
		s.redis.Incr(context.Background(), "kb:article:"+articleID+":helpful_no")
	}
	return nil
}

// ============================================================================
// Canned Responses
// ============================================================================

func (s *TicketService) CreateCannedResponse(response *CannedResponse) error {
	response.ID = uuid.New().String()
	response.CreatedAt = time.Now()
	
	responseJSON, _ := json.Marshal(response)
	s.redis.Set(context.Background(), "canned:"+response.ID, responseJSON, 0)
	
	// Add to category index
	s.redis.SAdd(context.Background(), "canned:category:"+response.Category, response.ID)
	
	return nil
}

func (s *TicketService) GetCannedResponses(category string) ([]*CannedResponse, error) {
	responseIDs, err := s.redis.SMembers(context.Background(), "canned:category:"+category).Result()
	if err != nil {
		return nil, err
	}
	
	responses := make([]*CannedResponse, 0, len(responseIDs))
	for _, id := range responseIDs {
		responseJSON, err := s.redis.Get(context.Background(), "canned:"+id).Result()
		if err != nil {
			continue
		}
		
		var response CannedResponse
		json.Unmarshal([]byte(responseJSON), &response)
		responses = append(responses, &response)
	}
	
	return responses, nil
}

// ============================================================================
// SLA Tracking
// ============================================================================

func (s *TicketService) GetSLAStatus(ticketID string) (*SLATracking, error) {
	ticket, err := s.GetTicket(ticketID)
	if err != nil {
		return nil, err
	}
	
	sla := &SLATracking{
		TicketID: ticketID,
		Priority: string(ticket.Priority),
	}
	
	// Calculate due times based on priority
	responseDue := ticket.CreatedAt
	resolutionDue := ticket.CreatedAt
	
	switch ticket.Priority {
	case TicketPriorityUrgent:
		responseDue = ticket.CreatedAt.Add(1 * time.Hour)
		resolutionDue = ticket.CreatedAt.Add(4 * time.Hour)
	case TicketPriorityHigh:
		responseDue = ticket.CreatedAt.Add(2 * time.Hour)
		resolutionDue = ticket.CreatedAt.Add(8 * time.Hour)
	case TicketPriorityMedium:
		responseDue = ticket.CreatedAt.Add(4 * time.Hour)
		resolutionDue = ticket.CreatedAt.Add(24 * time.Hour)
	default:
		responseDue = ticket.CreatedAt.Add(8 * time.Hour)
		resolutionDue = ticket.CreatedAt.Add(48 * time.Hour)
	}
	
	sla.ResponseDue = responseDue
	sla.ResolutionDue = resolutionDue
	
	// Check status
	now := time.Now()
	if ticket.Status == TicketStatusResolved || ticket.Status == TicketStatusClosed {
		sla.Status = "resolved"
		if ticket.ResolvedAt != nil {
			sla.ResolvedAt = ticket.ResolvedAt
		}
	} else if now.After(resolutionDue) {
		sla.Status = "breached"
	} else if now.After(responseDue) {
		sla.Status = "at_risk"
	} else {
		sla.Status = "on_track"
	}
	
	return sla, nil
}

// ============================================================================
// Statistics
// ============================================================================

func (s *TicketService) GetStats() (map[string]interface{}, error) {
	ctx := context.Background()
	
	stats := make(map[string]interface{})
	
	// Count by status
	stats["open"] = s.redis.ZCard(ctx, "tickets:open").Val()
	stats["in_progress"] = s.redis.ZCard(ctx, "tickets:in_progress").Val()
	stats["resolved"] = s.redis.ZCard(ctx, "tickets:resolved").Val()
	stats["closed"] = s.redis.ZCard(ctx, "tickets:closed").Val()
	
	// Count by priority
	stats["urgent"] = s.redis.Get(ctx, "tickets:count:priority:urgent").Val()
	stats["high"] = s.redis.Get(ctx, "tickets:count:priority:high").Val()
	stats["medium"] = s.redis.Get(ctx, "tickets:count:priority:medium").Val()
	stats["low"] = s.redis.Get(ctx, "tickets:count:priority:low").Val()
	
	// Average response time (mock)
	stats["avg_response_time_hours"] = 2.5
	
	// SLA compliance (mock)
	stats["sla_compliance"] = 94.5
	
	return stats, nil
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func (s *TicketService) CreateTicketHandler(c *gin.Context) {
	var ticket Ticket
	if err := c.ShouldBindJSON(&ticket); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	if err := s.CreateTicket(&ticket); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, ticket)
}

func (s *TicketService) GetTicketHandler(c *gin.Context) {
	id := c.Param("id")
	
	ticket, err := s.GetTicket(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}
	
	c.JSON(http.StatusOK, ticket)
}

func (s *TicketService) UpdateTicketHandler(c *gin.Context) {
	id := c.Param("id")
	
	ticket, err := s.GetTicket(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}
	
	var updates map[string]interface{}
	c.ShouldBindJSON(&updates)
	
	// Apply updates
	if subject, ok := updates["subject"].(string); ok {
		ticket.Subject = subject
	}
	if description, ok := updates["description"].(string); ok {
		ticket.Description = description
	}
	if priority, ok := updates["priority"].(string); ok {
		ticket.Priority = TicketPriority(priority)
	}
	if status, ok := updates["status"].(string); ok {
		ticket.Status = TicketStatus(status)
		if ticket.Status == TicketStatusResolved {
			now := time.Now()
			ticket.ResolvedAt = &now
		}
		if ticket.Status == TicketStatusClosed {
			now := time.Now()
			ticket.ClosedAt = &now
		}
	}
	if assignedTo, ok := updates["assigned_to"].(string); ok {
		ticket.AssignedTo = assignedTo
		if ticket.AssignedAt == nil {
			now := time.Now()
			ticket.AssignedAt = &now
		}
	}
	
	if err := s.UpdateTicket(ticket); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, ticket)
}

func (s *TicketService) ListTicketsHandler(c *gin.Context) {
	filters := make(map[string]string)
	
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}
	if priority := c.Query("priority"); priority != "" {
		filters["priority"] = priority
	}
	if category := c.Query("category"); category != "" {
		filters["category"] = category
	}
	if assignedTo := c.Query("assigned_to"); assignedTo != "" {
		filters["assigned_to"] = assignedTo
	}
	if page := c.Query("page"); page != "" {
		filters["page"] = page
	} else {
		filters["page"] = "1"
	}
	if pageSize := c.Query("page_size"); pageSize != "" {
		filters["page_size"] = pageSize
	} else {
		filters["page_size"] = "20"
	}
	
	tickets, total, err := s.ListTickets(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"tickets":    tickets,
		"total":      total,
		"page":       filters["page"],
		"page_size":  filters["page_size"],
		"total_pages": (total + int64(20) - 1) / int64(20),
	})
}

func (s *TicketService) AddCommentHandler(c *gin.Context) {
	var comment TicketComment
	if err := c.ShouldBindJSON(&comment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	if err := s.AddComment(&comment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, comment)
}

func (s *TicketService) GetCommentsHandler(c *gin.Context) {
	ticketID := c.Param("id")
	
	comments, err := s.GetComments(ticketID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"comments": comments})
}

func (s *TicketService) CreateArticleHandler(c *gin.Context) {
	var article KnowledgeBaseArticle
	if err := c.ShouldBindJSON(&article); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	if err := s.CreateArticle(&article); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, article)
}

func (s *TicketService) SearchArticlesHandler(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query required"})
		return
	}
	
	articles, err := s.SearchArticles(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"articles": articles})
}

func (s *TicketService) GetStatsHandler(c *gin.Context) {
	stats, err := s.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, stats)
}

func (s *TicketService) CreateCannedResponseHandler(c *gin.Context) {
	var response CannedResponse
	if err := c.ShouldBindJSON(&response); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	if err := s.CreateCannedResponse(&response); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, response)
}

func (s *TicketService) GetCannedResponsesHandler(c *gin.Context) {
	category := c.DefaultQuery("category", "general")
	
	responses, err := s.GetCannedResponses(category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"responses": responses})
}

// ============================================================================
// Main
// ============================================================================

func main() {
	log.Println("Starting TigerWallet Admin Support Ticket System...")
	
	config := LoadTicketConfig()
	
	// Initialize database (mock for example)
	db, _ := gorm.Open(nil)
	
	// Initialize Redis
	redisOpts, _ := redis.ParseURL(config.RedisURL)
	redisClient := redis.NewClient(redisOpts)
	
	// Create service
	ticketService := NewTicketService(config, db, redisClient)
	
	// Setup routes
	r := gin.Default()
	
	// Tickets
	r.POST("/api/v1/tickets", ticketService.CreateTicketHandler)
	r.GET("/api/v1/tickets", ticketService.ListTicketsHandler)
	r.GET("/api/v1/tickets/:id", ticketService.GetTicketHandler)
	r.PUT("/api/v1/tickets/:id", ticketService.UpdateTicketHandler)
	r.POST("/api/v1/tickets/:id/comments", ticketService.AddCommentHandler)
	r.GET("/api/v1/tickets/:id/comments", ticketService.GetCommentsHandler)
	r.GET("/api/v1/tickets/:id/sla", func(c *gin.Context) {
		id := c.Param("id")
		sla, err := ticketService.GetSLAStatus(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
			return
		}
		c.JSON(http.StatusOK, sla)
	})
	
	// Knowledge Base
	r.POST("/api/v1/kb/articles", ticketService.CreateArticleHandler)
	r.GET("/api/v1/kb/articles/search", ticketService.SearchArticlesHandler)
	r.POST("/api/v1/kb/articles/:id/feedback", func(c *gin.Context) {
		id := c.Param("id")
		var req struct {
			Helpful string `json:"helpful"`
		}
		c.ShouldBindJSON(&req)
		ticketService.ArticleFeedback(id, req.Helpful)
		c.JSON(http.StatusOK, gin.H{"message": "Feedback recorded"})
	})
	
	// Canned Responses
	r.POST("/api/v1/canned", ticketService.CreateCannedResponseHandler)
	r.GET("/api/v1/canned", ticketService.GetCannedResponsesHandler)
	
	// Stats
	r.GET("/api/v1/tickets/stats", ticketService.GetStatsHandler)
	
	// Health
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})
	
	addr := ":" + config.Port
	log.Printf("Ticket system listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
