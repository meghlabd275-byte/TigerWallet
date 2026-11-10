// Support Ticket System - Go Implementation
//
// Customer-facing ticket intake plus the internal agent console. The public
// surface (create a ticket, read it back, reply) is reachable with a shared
// X-API-Key because it is called from the wallet front ends' own servers; the
// agent surface (list, assign, status, internal notes, stats) additionally
// requires an admin JWT. Internal notes are never returned to the public
// reader.
//
// Required environment:
//
//	SUPPORT_API_KEY     >= 16 chars, shared with the calling front end
//	SUPPORT_JWT_SECRET  >= 32 chars, same issuer as the admin tier
//	DATABASE_URL        PostgreSQL connection string
//	REDIS_URL           Redis connection string
package main

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	Port           string
	DatabaseURL    string
	RedisURL       string
	APIKey         string
	JWTSecret      string
	AllowedOrigins []string
}

type TicketStatus string

const (
	StatusOpen       TicketStatus = "open"
	StatusPending    TicketStatus = "pending"
	StatusInProgress TicketStatus = "in_progress"
	StatusResolved   TicketStatus = "resolved"
	StatusClosed     TicketStatus = "closed"
)

type TicketPriority string

const (
	PriorityLow      TicketPriority = "low"
	PriorityMedium   TicketPriority = "medium"
	PriorityHigh     TicketPriority = "high"
	PriorityCritical TicketPriority = "critical"
)

// maxBodyBytes bounds request bodies so a large payload cannot exhaust memory.
const maxBodyBytes = 64 << 10

var (
	validStatuses = map[TicketStatus]bool{
		StatusOpen: true, StatusPending: true, StatusInProgress: true,
		StatusResolved: true, StatusClosed: true,
	}
	validPriorities = map[TicketPriority]bool{
		PriorityLow: true, PriorityMedium: true, PriorityHigh: true, PriorityCritical: true,
	}
	validCategories = map[string]bool{
		"technical": true, "billing": true, "account": true, "feature_request": true, "other": true,
	}
	allowedAdminRoles = map[string]bool{
		"admin": true, "super_admin": true, "master_admin": true,
		"support": true, "support_agent": true, "white_label_admin": true,
	}
)

type Ticket struct {
	ID          uuid.UUID      `json:"id"`
	ClientID    uuid.UUID      `json:"client_id"`
	Subject     string         `json:"subject"`
	Description string         `json:"description"`
	Category    string         `json:"category"`
	Status      TicketStatus   `json:"status"`
	Priority    TicketPriority `json:"priority"`
	AssignedTo  *uuid.UUID     `json:"assigned_to"`
	CreatedBy   uuid.UUID      `json:"created_by"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	ResolvedAt  *time.Time     `json:"resolved_at"`
}

type TicketReply struct {
	ID         uuid.UUID `json:"id"`
	TicketID   uuid.UUID `json:"ticket_id"`
	UserID     uuid.UUID `json:"user_id"`
	UserType   string    `json:"user_type"`
	Message    string    `json:"message"`
	IsInternal bool      `json:"is_internal"`
	CreatedAt  time.Time `json:"created_at"`
}

var (
	db          *pgxpool.Pool
	redisClient *redis.Client
	config      Config
	logger      *log.Logger
)

func main() {
	logger = log.New(os.Stdout, "Support: ", log.LstdFlags)
	logger.Println("Starting Support Ticket Service...")

	config = loadConfig()

	var err error
	db, err = pgxpool.New(context.Background(), config.DatabaseURL)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	if err := db.Ping(context.Background()); err != nil {
		logger.Fatalf("Database is not reachable: %v", err)
	}
	logger.Println("Database connected")

	if err := ensureSchema(context.Background()); err != nil {
		logger.Fatalf("Failed to apply schema: %v", err)
	}
	logger.Println("Schema applied")

	opt, err := redis.ParseURL(config.RedisURL)
	if err != nil {
		logger.Fatalf("Invalid Redis URL: %v", err)
	}
	redisClient = redis.NewClient(opt)
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		logger.Fatalf("Redis is not reachable: %v", err)
	}
	logger.Println("Redis connected")

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())
	router.Use(bodyLimitMiddleware())

	router.GET("/health", healthHandler)

	// Public surface: called from the wallet front ends' own servers, so it is
	// gated by a shared key and a Redis-backed rate limit rather than a session.
	public := router.Group("/api/v1")
	public.Use(requireAPIKey())
	public.Use(rateLimit("support:pub:", 60, time.Minute))
	{
		public.POST("/tickets", createTicket)
		public.GET("/tickets/:id", getTicket)
		public.POST("/tickets/:id/replies", createReply)
		public.GET("/tickets/:id/replies", getReplies)
	}

	// Agent surface.
	agent := router.Group("/api/v1")
	agent.Use(requireAuth())
	{
		agent.GET("/tickets", listTickets)
		agent.PUT("/tickets/:id", updateTicket)
		agent.PUT("/tickets/:id/status", updateStatus)
		agent.PUT("/tickets/:id/assign", assignTicket)
		agent.GET("/tickets/:id/notes", getAllReplies)
		agent.POST("/tickets/:id/notes", createInternalNote)
		agent.GET("/stats", getStats)
	}

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
	_ = srv.Shutdown(ctx)
	db.Close()
	redisClient.Close()
	logger.Println("Server exited")
}

func loadConfig() Config {
	allowed := []string{}
	for _, o := range strings.Split(os.Getenv("SUPPORT_ALLOWED_ORIGINS"), ",") {
		if o = strings.TrimSpace(o); o != "" {
			allowed = append(allowed, o)
		}
	}

	cfg := Config{
		Port:           getEnv("SUPPORT_PORT", "8102"),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		RedisURL:       getEnv("REDIS_URL", "redis://localhost:6379"),
		APIKey:         os.Getenv("SUPPORT_API_KEY"),
		JWTSecret:      os.Getenv("SUPPORT_JWT_SECRET"),
		AllowedOrigins: allowed,
	}
	if cfg.DatabaseURL == "" {
		logger.Fatal("DATABASE_URL is required")
	}
	if len(cfg.APIKey) < 16 {
		logger.Fatal("SUPPORT_API_KEY is required and must be at least 16 characters")
	}
	if len(cfg.JWTSecret) < 32 {
		logger.Fatal("SUPPORT_JWT_SECRET is required and must be at least 32 characters")
	}
	return cfg
}

// ensureSchema creates the tables this service owns. Each statement is
// idempotent; the ALTERs carry explicit types so they are valid on an empty
// database as well as an existing one.
func ensureSchema(ctx context.Context) error {
	statements := []string{
		`CREATE EXTENSION IF NOT EXISTS "pgcrypto"`,
		`CREATE TABLE IF NOT EXISTS tickets (
			id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			client_id   UUID NOT NULL,
			subject     VARCHAR(255) NOT NULL,
			description TEXT NOT NULL,
			category    VARCHAR(50) NOT NULL,
			status      VARCHAR(50) NOT NULL DEFAULT 'open',
			priority    VARCHAR(50) NOT NULL DEFAULT 'medium',
			assigned_to UUID,
			created_by  UUID NOT NULL,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			resolved_at TIMESTAMPTZ
		)`,
		`CREATE TABLE IF NOT EXISTS ticket_replies (
			id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			ticket_id   UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
			user_id     UUID NOT NULL,
			user_type   VARCHAR(20) NOT NULL,
			message     TEXT NOT NULL,
			is_internal BOOLEAN NOT NULL DEFAULT FALSE,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS ticket_audit_log (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			actor_id   UUID,
			ticket_id  UUID,
			action     VARCHAR(100) NOT NULL,
			details    TEXT,
			ip_address VARCHAR(45),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`ALTER TABLE tickets ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMPTZ`,
		`ALTER TABLE tickets ADD COLUMN IF NOT EXISTS assigned_to UUID`,
		`CREATE INDEX IF NOT EXISTS idx_tickets_client ON tickets(client_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tickets_status ON tickets(status)`,
		`CREATE INDEX IF NOT EXISTS idx_tickets_created_at ON tickets(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_ticket_replies_ticket ON ticket_replies(ticket_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_ticket_audit_ticket ON ticket_audit_log(ticket_id, created_at DESC)`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(ctx, statement); err != nil {
			return fmt.Errorf("schema statement failed: %w", err)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// MIDDLEWARE
// ---------------------------------------------------------------------------

func corsMiddleware() gin.HandlerFunc {
	allowed := make(map[string]bool, len(config.AllowedOrigins))
	for _, o := range config.AllowedOrigins {
		allowed[o] = true
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && allowed[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
			c.Header("Access-Control-Max-Age", "600")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func bodyLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
		}
		c.Next()
	}
}

func requireAPIKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		presented := c.GetHeader("X-API-Key")
		if subtle.ConstantTimeCompare([]byte(presented), []byte(config.APIKey)) != 1 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// rateLimit is a Redis-backed fixed-window limiter so one budget is shared by
// every replica. It fails closed: if Redis is unreachable the request is
// refused rather than allowed through unmetered.
func rateLimit(prefix string, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		key := prefix + c.ClientIP()
		n, err := redisClient.Incr(ctx, key).Result()
		if err != nil {
			logger.Printf("rate limiter unavailable: %v", err)
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "service temporarily unavailable"})
			c.Abort()
			return
		}
		if n == 1 {
			redisClient.Expire(ctx, key, window)
		}
		if n > int64(limit) {
			c.Header("Retry-After", strconv.Itoa(int(window.Seconds())))
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			c.Abort()
			return
		}
		c.Next()
	}
}

type adminClaims struct {
	AdminID string `json:"admin_id"`
	Role    string `json:"role"`
	jwt.RegisteredClaims
}

func requireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "bearer token required"})
			c.Abort()
			return
		}
		claims := &adminClaims{}
		token, err := jwt.ParseWithClaims(parts[1], claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(config.JWTSecret), nil
		}, jwt.WithValidMethods([]string{"HS256"}))
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}
		if !allowedAdminRoles[claims.Role] {
			c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
			c.Abort()
			return
		}
		c.Set("admin_id", claims.AdminID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// ---------------------------------------------------------------------------
// HELPERS
// ---------------------------------------------------------------------------

func dbError(c *gin.Context, action string, err error) {
	logger.Printf("%s: %v", action, err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
}

// audit records an agent action against the ticket it touched.
func audit(actorID, ticketID, action, details, ip string) {
	if _, err := db.Exec(context.Background(), `
		INSERT INTO ticket_audit_log (actor_id, ticket_id, action, details, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`, actorID, ticketID, action, details, ip); err != nil {
		logger.Printf("audit write failed: %v", err)
	}
}

func healthHandler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	if err := db.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "service": "support"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "support"})
}

// ---------------------------------------------------------------------------
// PUBLIC HANDLERS
// ---------------------------------------------------------------------------

func createTicket(c *gin.Context) {
	var req struct {
		ClientID    uuid.UUID      `json:"client_id" binding:"required"`
		Subject     string         `json:"subject" binding:"required"`
		Description string         `json:"description" binding:"required"`
		Category    string         `json:"category" binding:"required"`
		Priority    TicketPriority `json:"priority"`
		CreatedBy   uuid.UUID      `json:"created_by" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if len(strings.TrimSpace(req.Subject)) == 0 || len(req.Subject) > 255 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subject must be 1-255 characters"})
		return
	}
	if len(strings.TrimSpace(req.Description)) == 0 || len(req.Description) > 20000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "description must be 1-20000 characters"})
		return
	}
	if !validCategories[req.Category] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown category"})
		return
	}
	priority := req.Priority
	if priority == "" {
		priority = PriorityMedium
	}
	if !validPriorities[priority] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown priority"})
		return
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

	if _, err := db.Exec(context.Background(), `
		INSERT INTO tickets (id, client_id, subject, description, category, status, priority, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, ticket.ID, ticket.ClientID, ticket.Subject, ticket.Description, ticket.Category,
		ticket.Status, ticket.Priority, ticket.CreatedBy, ticket.CreatedAt, ticket.UpdatedAt); err != nil {
		dbError(c, "create ticket", err)
		return
	}

	if err := redisClient.Incr(context.Background(), "tickets:open").Err(); err != nil {
		logger.Printf("open counter increment failed: %v", err)
	}

	c.JSON(http.StatusCreated, ticket)
}

func getTicket(c *gin.Context) {
	uid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	var ticket Ticket
	err = db.QueryRow(context.Background(), `
		SELECT id, client_id, subject, description, category, status, priority,
		       assigned_to, created_by, created_at, updated_at, resolved_at
		FROM tickets WHERE id = $1
	`, uid).Scan(&ticket.ID, &ticket.ClientID, &ticket.Subject, &ticket.Description,
		&ticket.Category, &ticket.Status, &ticket.Priority, &ticket.AssignedTo,
		&ticket.CreatedBy, &ticket.CreatedAt, &ticket.UpdatedAt, &ticket.ResolvedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err != nil {
		dbError(c, "read ticket", err)
		return
	}

	c.JSON(http.StatusOK, ticket)
}

func createReply(c *gin.Context) {
	ticketID, err := uuid.Parse(c.Param("id"))
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if req.UserType != "client" && req.UserType != "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_type must be client or admin"})
		return
	}
	if strings.TrimSpace(req.Message) == "" || len(req.Message) > 20000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message must be 1-20000 characters"})
		return
	}

	// A caller on the public surface may not write internal notes; those are for
	// agents only.
	if req.IsInternal {
		c.JSON(http.StatusForbidden, gin.H{"error": "internal notes require agent authentication"})
		return
	}

	var exists bool
	if err := db.QueryRow(context.Background(),
		`SELECT TRUE FROM tickets WHERE id = $1`, ticketID).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
			return
		}
		dbError(c, "check ticket", err)
		return
	}

	reply := TicketReply{
		ID:        uuid.New(),
		TicketID:  ticketID,
		UserID:    req.UserID,
		UserType:  req.UserType,
		Message:   req.Message,
		CreatedAt: time.Now(),
	}

	if _, err := db.Exec(context.Background(), `
		INSERT INTO ticket_replies (id, ticket_id, user_id, user_type, message, is_internal, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, reply.ID, reply.TicketID, reply.UserID, reply.UserType, reply.Message,
		reply.IsInternal, reply.CreatedAt); err != nil {
		dbError(c, "create reply", err)
		return
	}

	if _, err := db.Exec(context.Background(),
		`UPDATE tickets SET updated_at = NOW() WHERE id = $1`, ticketID); err != nil {
		logger.Printf("ticket touch failed: %v", err)
	}

	c.JSON(http.StatusCreated, reply)
}

func getReplies(c *gin.Context) {
	ticketID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	rows, err := db.Query(context.Background(), `
		SELECT id, user_id, user_type, message, is_internal, created_at
		FROM ticket_replies
		WHERE ticket_id = $1 AND is_internal = FALSE
		ORDER BY created_at ASC
	`, ticketID)
	if err != nil {
		dbError(c, "list replies", err)
		return
	}
	defer rows.Close()

	replies := []TicketReply{}
	for rows.Next() {
		var r TicketReply
		if err := rows.Scan(&r.ID, &r.UserID, &r.UserType, &r.Message, &r.IsInternal, &r.CreatedAt); err != nil {
			dbError(c, "scan reply", err)
			return
		}
		r.TicketID = ticketID
		replies = append(replies, r)
	}
	if err := rows.Err(); err != nil {
		dbError(c, "iterate replies", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"replies": replies})
}

// ---------------------------------------------------------------------------
// AGENT HANDLERS
// ---------------------------------------------------------------------------

func listTickets(c *gin.Context) {
	clientID := c.Query("client_id")
	status := c.Query("status")
	limit := clampLimit(c.Query("limit"))

	query := `SELECT id, client_id, subject, description, category, status, priority,
	                 assigned_to, created_by, created_at, updated_at, resolved_at
	          FROM tickets WHERE 1=1`
	args := []interface{}{}
	if clientID != "" {
		uid, err := uuid.Parse(clientID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client_id"})
			return
		}
		args = append(args, uid)
		query += fmt.Sprintf(" AND client_id = $%d", len(args))
	}
	if status != "" {
		if !validStatuses[TicketStatus(status)] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unknown status"})
			return
		}
		args = append(args, status)
		query += fmt.Sprintf(" AND status = $%d", len(args))
	}
	args = append(args, limit)
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", len(args))

	rows, err := db.Query(context.Background(), query, args...)
	if err != nil {
		dbError(c, "list tickets", err)
		return
	}
	defer rows.Close()

	tickets := []Ticket{}
	for rows.Next() {
		var t Ticket
		if err := rows.Scan(&t.ID, &t.ClientID, &t.Subject, &t.Description, &t.Category,
			&t.Status, &t.Priority, &t.AssignedTo, &t.CreatedBy, &t.CreatedAt,
			&t.UpdatedAt, &t.ResolvedAt); err != nil {
			dbError(c, "scan ticket", err)
			return
		}
		tickets = append(tickets, t)
	}
	if err := rows.Err(); err != nil {
		dbError(c, "iterate tickets", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"tickets": tickets})
}

func updateTicket(c *gin.Context) {
	uid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	var req struct {
		Subject     string         `json:"subject"`
		Description string         `json:"description"`
		Priority    TicketPriority `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if len(req.Subject) > 255 || len(req.Description) > 20000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "field too long"})
		return
	}
	if req.Priority != "" && !validPriorities[req.Priority] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown priority"})
		return
	}

	tag, err := db.Exec(context.Background(), `
		UPDATE tickets
		SET subject     = COALESCE(NULLIF($1, ''), subject),
		    description = COALESCE(NULLIF($2, ''), description),
		    priority    = COALESCE($3, priority),
		    updated_at  = NOW()
		WHERE id = $4
	`, req.Subject, req.Description, req.Priority, uid)
	if err != nil {
		dbError(c, "update ticket", err)
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	actor, _ := c.Get("admin_id")
	audit(toString(actor), uid.String(), "UPDATE_TICKET", "ticket fields updated", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "ticket updated"})
}

func updateStatus(c *gin.Context) {
	uid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	var req struct {
		Status TicketStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if !validStatuses[req.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown status"})
		return
	}

	// resolved_at is cleared when a ticket leaves the resolved state so the
	// column always reflects the current state.
	var resolvedAt *time.Time
	if req.Status == StatusResolved {
		now := time.Now()
		resolvedAt = &now
	}

	tag, err := db.Exec(context.Background(), `
		UPDATE tickets SET status = $1, resolved_at = $2, updated_at = NOW() WHERE id = $3
	`, req.Status, resolvedAt, uid)
	if err != nil {
		dbError(c, "update status", err)
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	if err := redisClient.Decr(context.Background(), "tickets:open").Err(); err != nil {
		logger.Printf("open counter decrement failed: %v", err)
	}
	if req.Status == StatusResolved {
		if err := redisClient.Incr(context.Background(), "tickets:resolved").Err(); err != nil {
			logger.Printf("resolved counter increment failed: %v", err)
		}
	}

	actor, _ := c.Get("admin_id")
	audit(toString(actor), uid.String(), "UPDATE_STATUS", "status -> "+string(req.Status), c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "status updated"})
}

func assignTicket(c *gin.Context) {
	uid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	var req struct {
		AssignedTo uuid.UUID `json:"assigned_to" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	tag, err := db.Exec(context.Background(), `
		UPDATE tickets SET assigned_to = $1, status = 'in_progress', updated_at = NOW() WHERE id = $2
	`, req.AssignedTo, uid)
	if err != nil {
		dbError(c, "assign ticket", err)
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	actor, _ := c.Get("admin_id")
	audit(toString(actor), uid.String(), "ASSIGN_TICKET", "assigned to "+req.AssignedTo.String(), c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "ticket assigned"})
}

// createInternalNote records an agent-only note. The note is never returned by
// the public reply endpoint.
func createInternalNote(c *gin.Context) {
	ticketID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	var req struct {
		Message string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if strings.TrimSpace(req.Message) == "" || len(req.Message) > 20000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message must be 1-20000 characters"})
		return
	}

	var exists bool
	if err := db.QueryRow(context.Background(),
		`SELECT TRUE FROM tickets WHERE id = $1`, ticketID).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
			return
		}
		dbError(c, "check ticket", err)
		return
	}

	actor, _ := c.Get("admin_id")
	note := TicketReply{
		ID:         uuid.New(),
		TicketID:   ticketID,
		UserID:     uuid.Nil,
		UserType:   "admin",
		Message:    req.Message,
		IsInternal: true,
		CreatedAt:  time.Now(),
	}
	if id, err := uuid.Parse(toString(actor)); err == nil {
		note.UserID = id
	}

	if _, err := db.Exec(context.Background(), `
		INSERT INTO ticket_replies (id, ticket_id, user_id, user_type, message, is_internal, created_at)
		VALUES ($1, $2, $3, $4, $5, TRUE, $6)
	`, note.ID, note.TicketID, note.UserID, note.UserType, note.Message, note.CreatedAt); err != nil {
		dbError(c, "create internal note", err)
		return
	}

	audit(toString(actor), ticketID.String(), "ADD_INTERNAL_NOTE", "internal note added", c.ClientIP())

	c.JSON(http.StatusCreated, note)
}

// getAllReplies returns every reply including internal notes. Agent access only.
func getAllReplies(c *gin.Context) {
	ticketID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	rows, err := db.Query(context.Background(), `
		SELECT id, user_id, user_type, message, is_internal, created_at
		FROM ticket_replies WHERE ticket_id = $1 ORDER BY created_at ASC
	`, ticketID)
	if err != nil {
		dbError(c, "list all replies", err)
		return
	}
	defer rows.Close()

	replies := []TicketReply{}
	for rows.Next() {
		var r TicketReply
		if err := rows.Scan(&r.ID, &r.UserID, &r.UserType, &r.Message, &r.IsInternal, &r.CreatedAt); err != nil {
			dbError(c, "scan reply", err)
			return
		}
		r.TicketID = ticketID
		replies = append(replies, r)
	}
	if err := rows.Err(); err != nil {
		dbError(c, "iterate replies", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"replies": replies})
}

func getStats(c *gin.Context) {
	var total, open, inProgress, resolved, closed int64
	ctx := context.Background()
	queries := []struct {
		sql string
		dst *int64
	}{
		{`SELECT COUNT(*) FROM tickets`, &total},
		{`SELECT COUNT(*) FROM tickets WHERE status = 'open'`, &open},
		{`SELECT COUNT(*) FROM tickets WHERE status = 'in_progress'`, &inProgress},
		{`SELECT COUNT(*) FROM tickets WHERE status = 'resolved'`, &resolved},
		{`SELECT COUNT(*) FROM tickets WHERE status = 'closed'`, &closed},
	}
	for _, q := range queries {
		if err := db.QueryRow(ctx, q.sql).Scan(q.dst); err != nil {
			dbError(c, "read stats", err)
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total":       total,
		"open":        open,
		"in_progress": inProgress,
		"resolved":    resolved,
		"closed":      closed,
	})
}

// ---------------------------------------------------------------------------
// UTILITIES
// ---------------------------------------------------------------------------

func clampLimit(raw string) int {
	if raw == "" {
		return 100
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 100
	}
	if n > 500 {
		return 500
	}
	return n
}

func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
