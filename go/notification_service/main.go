// TigerWallet Notification Service
//
// Stores notifications in PostgreSQL and delivers them through the channel the
// caller names. Delivery status reflects what the provider actually answered;
// a notification is never reported as sent when no provider accepted it.

package main

import (
	"context"
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
	"github.com/google/uuid"
)

type Config struct {
	Port int
}

func loadNotificationConfig() Config {
	port := 8011
	if v := os.Getenv("PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			port = n
		}
	}
	return Config{Port: port}
}

var cfg = loadNotificationConfig()

type Notification struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	Type      string                 `json:"type"`
	Title     string                 `json:"title"`
	Body      string                 `json:"body"`
	Data      map[string]interface{} `json:"data"`
	Status    string                 `json:"status"` // pending, sent, failed, unavailable
	Channel   string                 `json:"channel"`
	Priority  string                 `json:"priority"`
	Read      bool                   `json:"read"`
	CreatedAt time.Time              `json:"created_at"`
	SentAt    *time.Time             `json:"sent_at"`
}

type Subscription struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Channels  []string  `json:"channels"`
	Events    []string  `json:"events"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

type NotificationService struct {
	store      *store
	dispatcher *dispatcher
}

func NewNotificationService(st *store) *NotificationService {
	return &NotificationService{store: st, dispatcher: newDispatcher()}
}

// deliver persists a notification, attempts delivery and records the real
// outcome. The status returned to the caller is the one stored.
func (ns *NotificationService) deliver(ctx context.Context, n *Notification) *Notification {
	n.ID = uuid.New().String()
	n.CreatedAt = time.Now()
	if n.Priority == "" {
		n.Priority = "normal"
	}
	n.Status = "pending"
	if err := ns.store.insert(ctx, n); err != nil {
		log.Printf("notification insert failed: %v", err)
		n.Status = "failed"
		return n
	}

	deliverCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := ns.dispatcher.Send(deliverCtx, n); err != nil {
		status := "failed"
		if errors.Is(err, errChannelUnavailable) {
			status = "unavailable"
		}
		n.Status = status
		if markErr := ns.store.markResult(ctx, n.ID, status, err.Error()); markErr != nil {
			log.Printf("notification status update failed for %s: %v", n.ID, markErr)
		}
		return n
	}

	now := time.Now()
	n.Status = "sent"
	n.SentAt = &now
	if err := ns.store.markResult(ctx, n.ID, "sent", ""); err != nil {
		log.Printf("notification status update failed for %s: %v", n.ID, err)
	}
	return n
}

func (ns *NotificationService) SendNotification(c *gin.Context) {
	var req struct {
		UserID   string                 `json:"user_id" binding:"required"`
		Type     string                 `json:"type" binding:"required"`
		Channel  string                 `json:"channel"`
		Title    string                 `json:"title" binding:"required"`
		Body     string                 `json:"body" binding:"required"`
		Data     map[string]interface{} `json:"data"`
		Priority string                 `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	channel := req.Channel
	if channel == "" {
		channel = "in_app"
	}

	n := ns.deliver(c.Request.Context(), &Notification{
		UserID:   req.UserID,
		Type:     req.Type,
		Channel:  channel,
		Title:    req.Title,
		Body:     req.Body,
		Data:     req.Data,
		Priority: req.Priority,
	})

	// A notification that could not be handed to a provider is reported as an
	// error, so the caller does not assume the user was reached.
	switch n.Status {
	case "sent":
		c.JSON(http.StatusOK, gin.H{"success": true, "notification_id": n.ID, "status": n.Status})
	case "unavailable":
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success":         false,
			"notification_id": n.ID,
			"status":          n.Status,
			"error":           "no provider is configured for channel " + channel,
		})
	default:
		c.JSON(http.StatusBadGateway, gin.H{
			"success":         false,
			"notification_id": n.ID,
			"status":          n.Status,
			"error":           "delivery failed",
		})
	}
}

func (ns *NotificationService) GetNotifications(c *gin.Context) {
	userID := c.Param("user_id")
	limit, _ := strconv.Atoi(c.Query("limit"))
	notifs, err := ns.store.listForUser(c.Request.Context(), userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot load notifications"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "notifications": notifs})
}

// MarkAsRead marks a notification read. The owning user id is required so one
// user cannot modify another user's notification by guessing its id.
func (ns *NotificationService) MarkAsRead(c *gin.Context) {
	notifID := c.Param("id")
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}
	ok, err := ns.store.markRead(c.Request.Context(), notifID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot update notification"})
		return
	}
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (ns *NotificationService) MarkAllAsRead(c *gin.Context) {
	userID := c.Param("user_id")
	count, err := ns.store.markAllRead(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot update notifications"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "marked_read": count})
}

func (ns *NotificationService) DeleteNotification(c *gin.Context) {
	notifID := c.Param("id")
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}
	ok, err := ns.store.delete(c.Request.Context(), notifID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot delete notification"})
		return
	}
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (ns *NotificationService) ClearAll(c *gin.Context) {
	userID := c.Param("user_id")
	removed, err := ns.store.clearForUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot clear notifications"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "cleared": removed})
}

func (ns *NotificationService) Subscribe(c *gin.Context) {
	var req struct {
		UserID   string   `json:"user_id" binding:"required"`
		Channels []string `json:"channels" binding:"required"`
		Events   []string `json:"events" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sub := &Subscription{
		ID:        uuid.New().String(),
		UserID:    req.UserID,
		Channels:  req.Channels,
		Events:    req.Events,
		Enabled:   true,
		CreatedAt: time.Now(),
	}
	if err := ns.store.createSubscription(c.Request.Context(), sub); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot save subscription"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "subscription": sub})
}

func (ns *NotificationService) Unsubscribe(c *gin.Context) {
	subID := c.Param("id")
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}
	ok, err := ns.store.deleteSubscription(c.Request.Context(), subID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot remove subscription"})
		return
	}
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "status": "unsubscribed"})
}

// SendBatch fans a message out to the given users, attempting delivery for
// each and reporting how many were actually delivered.
func (ns *NotificationService) SendBatch(c *gin.Context) {
	var req struct {
		UserIDs []string `json:"user_ids" binding:"required"`
		Title   string   `json:"title" binding:"required"`
		Body    string   `json:"body" binding:"required"`
		Type    string   `json:"type" binding:"required"`
		Channel string   `json:"channel"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.UserIDs) > 10000 {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "batch is limited to 10000 recipients"})
		return
	}
	channel := req.Channel
	if channel == "" {
		channel = "in_app"
	}

	ctx := c.Request.Context()
	results := make([]map[string]string, 0, len(req.UserIDs))
	sent, failed := 0, 0
	for _, userID := range req.UserIDs {
		n := ns.deliver(ctx, &Notification{
			UserID:  userID,
			Type:    req.Type,
			Channel: channel,
			Title:   req.Title,
			Body:    req.Body,
		})
		results = append(results, map[string]string{"user_id": userID, "notification_id": n.ID, "status": n.Status})
		if n.Status == "sent" {
			sent++
		} else {
			failed++
		}
	}

	status := http.StatusOK
	if sent == 0 && failed > 0 {
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, gin.H{
		"success": failed == 0,
		"sent":    sent,
		"failed":  failed,
		"total":   len(req.UserIDs),
		"results": results,
	})
}

func (ns *NotificationService) GetTemplates(c *gin.Context) {
	templates, err := ns.store.templates(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot load templates"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "templates": templates})
}

// resumePending closes out notifications whose delivery attempt was interrupted
// by a restart between insert and provider response.
func (ns *NotificationService) resumePending(ctx context.Context) {
	ids, err := ns.store.pendingIDs(ctx, 500)
	if err != nil || len(ids) == 0 {
		return
	}
	log.Printf("closing out %d interrupted notification deliveries", len(ids))
	for _, id := range ids {
		if err := ns.store.markResult(ctx, id, "failed", "delivery interrupted by restart"); err != nil {
			log.Printf("could not close out pending notification %s: %v", id, err)
		}
	}
}

func parseInt32(v string) (int32, error) {
	n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 32)
	return int32(n), err
}

// corsOrigins parses the browser-origin allowlist from ALLOWED_ORIGINS (comma
// separated). An empty allowlist emits no CORS headers, which keeps browsers on
// the same origin by default instead of exposing the API to every site.
func corsOrigins() map[string]bool {
	origins := make(map[string]bool)
	for _, origin := range strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			origins[origin] = true
		}
	}
	return origins
}

func main() {
	log.Println("TigerWallet Notification Service")
	log.Printf("Starting on port %d", cfg.Port)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	st, err := newStore(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("cannot start without a database: %v", err)
	}
	defer st.Close()

	ns := NewNotificationService(st)
	ns.resumePending(ctx)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		// CORS: explicit origin allowlist. A wildcard would let any site read
		// this API with the visitor's credentials attached.
		if origin := c.Request.Header.Get("Origin"); origin != "" && corsOrigins()[origin] {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Vary", "Origin")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		if err := st.pool.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "service": "notification", "error": "database unreachable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "notification"})
	})

	api := r.Group("/api/v1/notifications")
	{
		api.POST("/send", ns.SendNotification)
		api.POST("/batch", ns.SendBatch)
		api.GET("/users/:user_id", ns.GetNotifications)
		api.PUT("/users/:user_id/read", ns.MarkAllAsRead)
		api.PUT("/users/:user_id/clear", ns.ClearAll)
		api.PUT("/:id/read", ns.MarkAsRead)
		api.DELETE("/:id", ns.DeleteNotification)
		api.POST("/subscribe", ns.Subscribe)
		api.DELETE("/subscribe/:id", ns.Unsubscribe)
		api.GET("/templates", ns.GetTemplates)
	}

	srv := &http.Server{Addr: fmt.Sprintf(":%d", cfg.Port), Handler: r}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
