// Support Ticket System - Go Implementation
// Complete support ticketing system for TigerWallet

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
}

type TicketStatus string
type TicketPriority string

const (
	StatusOpen     TicketStatus = "open"
	StatusPending TicketStatus = "pending"
	StatusInProgress TicketStatus = "in_progress"
	StatusResolved TicketStatus = "resolved"
	StatusClosed   TicketStatus = "closed"
)

const (
	PriorityLow      TicketPriority = "low"
	PriorityMedium   TicketPriority = "medium"
	PriorityHigh     TicketPriority = "high"
	PriorityCritical TicketPriority = "critical"
)

type Ticket struct {
	ID          uuid.UUID      `json:"id"`
	ClientID    uuid.UUID      `json:"client_id"`
	Subject     string        `json:"subject"`
	Description string        `json:"description"`
	Category    string        `json:"category"` // technical, billing, account, feature_request
	Status      TicketStatus  `json:"status"`
	Priority    TicketPriority `json:"priority"`
	AssignedTo  *uuid.UUID   `json:"assigned_to"`
	CreatedBy   uuid.UUID    `json:"created_by"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	ResolvedAt  *time.Time   `json:"resolved_at"`
}

type TicketReply struct {
	ID        uuid.UUID   `json:"id"`
	TicketID  uuid.UUID   `json:"ticket_id"`
	UserID    uuid.UUID   `json:"user_id"`
	UserType  string      `json:"user_type"` // client, admin
	Message   string      `json:"message"`
	IsInternal bool      `json:"is_internal"`
	CreatedAt time.Time  `json:"created_at"`
}

var db *pgxpool.Pool
var redis *redis.Client
var config Config
var logger *log.Logger

func main() {
	logger = log.New(os.Stdout, "Support: ", log.LstdFlags)
	logger.Println("Starting Support Ticket Service...")

	config.Port = getEnv("SUPPORT_PORT", "8102")
	config.DatabaseURL = getEnv("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet_admin")
	config.RedisURL = getEnv("REDIS_URL", "redis://localhost:6379")

	var err error
	db, err = pgxpool.New(context.Background(), config.DatabaseURL)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	logger.Println("Database connected")

	opt, _ := redis.ParseURL(config.RedisURL)
	redis = redis.NewClient(opt)
	redis.Ping(context.Background())
	logger.Println("Redis connected")

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "support"})
	})

	// Tickets
	router.POST("/api/v1/tickets", createTicket)
	router.GET("/api/v1/tickets/:id", getTicket)
	router.GET("/api/v1/tickets", listTickets)
	router.PUT("/api/v1/tickets/:id", updateTicket)
	router.PUT("/api/v1/tickets/:id/status", updateStatus)
	router.PUT("/api/v1/tickets/:id/assign", assignTicket)

	// Replies
	router.POST("/api/v1/tickets/:id/replies", createReply)
	router.GET("/api/v1/tickets/:id/replies", getReplies)

	// Stats
	router.GET("/api/v1/stats", getStats)

	logger.Printf("Starting server on port %s", config.Port)
	srv := &http.Server{Addr: ":" + config.Port, Handler: router}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	logger.Println("Server started")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	db.Close()
	redis.Close()
	logger.Println("Server exited")
}

func createTicket(c *gin.Context) {
	var req struct {
		ClientID    uuid.UUID      `json:"client_id" binding:"required"`
		Subject     string        `json:"subject" binding:"required"`
		Description string        `json:"description" binding:"required"`
		Category    string        `json:"category" binding:"required"`
		Priority    TicketPriority `json:"priority"`
		CreatedBy   uuid.UUID     `json:"created_by" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	priority := req.Priority
	if priority == "" {
		priority = PriorityMedium
	}

	ticket := Ticket{
		ID:          uuid.New(),
		ClientID:    req.ClientID,
		Subject:     req.Subject,
		Description: req.Description,
		Category:    req.Category,
		Status:      StatusOpen,
		Priority:    priority,
		CreatedBy:   req.CreatedBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err := db.Exec(context.Background(), `
		INSERT INTO tickets (id, client_id, subject, description, category, status, priority, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, ticket.ID, ticket.ClientID, ticket.Subject, ticket.Description, ticket.Category, ticket.Status, ticket.Priority, ticket.CreatedBy, ticket.CreatedAt, ticket.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update stats
	redis.Incr(context.Background(), "tickets:open")

	c.JSON(http.StatusCreated, ticket)
}

func getTicket(c *gin.Context) {
	id := c.Param("id")
	uid, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	var ticket Ticket
	err = db.QueryRow(context.Background(), `
		SELECT id, client_id, subject, description, category, status, priority, assigned_to, created_by, created_at, updated_at, resolved_at
		FROM tickets WHERE id = $1
	`, uid).Scan(&ticket.ID, &ticket.ClientID, &ticket.Subject, &ticket.Description, &ticket.Category, &ticket.Status, &ticket.Priority, &ticket.AssignedTo, &ticket.CreatedBy, &ticket.CreatedAt, &ticket.UpdatedAt, &ticket.ResolvedAt)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	c.JSON(http.StatusOK, ticket)
}

func listTickets(c *gin.Context) {
	clientID := c.Query("client_id")
	status := c.Query("status")

	query := "SELECT id, client_id, subject, category, status, priority, created_at, updated_at FROM tickets WHERE 1=1"
	if clientID != "" {
		query += fmt.Sprintf(" AND client_id = '%s'", clientID)
	}
	if status != "" {
		query += fmt.Sprintf(" AND status = '%s'", status)
	}
	query += " ORDER BY created_at DESC LIMIT 100"

	rows, err := db.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var tickets []Ticket
	for rows.Next() {
		var t Ticket
		if err := rows.Scan(&t.ID, &t.ClientID, &t.Subject, &t.Category, &t.Status, &t.Priority, &t.CreatedAt, &t.UpdatedAt); err != nil {
			continue
		}
		tickets = append(tickets, t)
	}

	c.JSON(http.StatusOK, gin.H{"tickets": tickets})
}

func updateTicket(c *gin.Context) {
	id := c.Param("id")
	uid, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	var req struct {
		Subject     string        `json:"subject"`
		Description string        `json:"description"`
		Priority    TicketPriority `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err = db.Exec(context.Background(), `
		UPDATE tickets SET subject = COALESCE(NULLIF($1, ''), subject), description = COALESCE(NULLIF($2, ''), description), priority = COALESCE($3, priority), updated_at = NOW() WHERE id = $4
	`, req.Subject, req.Description, req.Priority, uid)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ticket updated"})
}

func updateStatus(c *gin.Context) {
	id := c.Param("id")
	uid, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	var req struct {
		Status TicketStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var resolvedAt interface{}
	if req.Status == StatusResolved {
		now := time.Now()
		resolvedAt = now
	}

	_, err = db.Exec(context.Background(), `
		UPDATE tickets SET status = $1, resolved_at = $2, updated_at = NOW() WHERE id = $3
	`, req.Status, resolvedAt, uid)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update stats
	if req.Status == StatusResolved {
		redis.Decr(context.Background(), "tickets:open")
		redis.Incr(context.Background(), "tickets:resolved")
	}

	c.JSON(http.StatusOK, gin.H{"message": "status updated"})
}

func assignTicket(c *gin.Context) {
	id := c.Param("id")
	uid, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	var req struct {
		AssignedTo uuid.UUID `json:"assigned_to" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err = db.Exec(context.Background(), `
		UPDATE tickets SET assigned_to = $1, status = 'in_progress', updated_at = NOW() WHERE id = $2
	`, req.AssignedTo, uid)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ticket assigned"})
}

func createReply(c *gin.Context) {
	id := c.Param("id")
	ticketID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	var req struct {
		UserID     uuid.UUID `json:"user_id" binding:"required"`
		UserType   string    `json:"user_type" binding:"required"`
		Message    string    `json:"message" binding:"required"`
		IsInternal bool      `json:"is_internal"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	reply := TicketReply{
		ID:        uuid.New(),
		TicketID:  ticketID,
		UserID:    req.UserID,
		UserType:  req.UserType,
		Message:   req.Message,
		IsInternal: req.IsInternal,
		CreatedAt: time.Now(),
	}

	_, err = db.Exec(context.Background(), `
		INSERT INTO ticket_replies (id, ticket_id, user_id, user_type, message, is_internal, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, reply.ID, reply.TicketID, reply.UserID, reply.UserType, reply.Message, reply.IsInternal, reply.CreatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update ticket
	db.Exec(context.Background(), `UPDATE tickets SET updated_at = NOW() WHERE id = $1`, ticketID)

	c.JSON(http.StatusCreated, reply)
}

func getReplies(c *gin.Context) {
	id := c.Param("id")
	ticketID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	rows, err := db.Query(context.Background(), `
		SELECT id, user_id, user_type, message, is_internal, created_at
		FROM ticket_replies WHERE ticket_id = $1 ORDER BY created_at ASC
	`, ticketID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var replies []TicketReply
	for rows.Next() {
		var r TicketReply
		if err := rows.Scan(&r.ID, &r.UserID, &r.UserType, &r.Message, &r.IsInternal, &r.CreatedAt); err != nil {
			continue
		}
		replies = append(replies, r)
	}

	c.JSON(http.StatusOK, gin.H{"replies": replies})
}

func getStats(c *gin.Context) {
	var total, open, inProgress, resolved, closed int64
	db.QueryRow(context.Background(), "SELECT COUNT(*) FROM tickets").Scan(&total)
	db.QueryRow(context.Background(), "SELECT COUNT(*) FROM tickets WHERE status = 'open'").Scan(&open)
	db.QueryRow(context.Background(), "SELECT COUNT(*) FROM tickets WHERE status = 'in_progress'").Scan(&inProgress)
	db.QueryRow(context.Background(), "SELECT COUNT(*) FROM tickets WHERE status = 'resolved'").Scan(&resolved)
	db.QueryRow(context.Background(), "SELECT COUNT(*) FROM tickets WHERE status = 'closed'").Scan(&closed)

	c.JSON(http.StatusOK, gin.H{
		"total":       total,
		"open":        open,
		"in_progress": inProgress,
		"resolved":    resolved,
		"closed":      closed,
	})
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
