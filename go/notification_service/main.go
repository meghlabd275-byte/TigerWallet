// TigerWallet Notification Service
// Push notifications, email, SMS alerts

package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Config struct {
	Port int
}

var cfg = Config{Port: 8011}

type Notification struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	Type      string                 `json:"type"` // push, email, sms
	Title     string                 `json:"title"`
	Body      string                 `json:"body"`
	Data      map[string]interface{} `json:"data"`
	Status    string                 `json:"status"` // pending, sent, failed
	Channel   string                 `json:"channel"`
	Priority  string                 `json:"priority"` // low, normal, high, urgent
	Read      bool                   `json:"read"`
	CreatedAt time.Time              `json:"created_at"`
	SentAt    *time.Time             `json:"sent_at"`
}

type Subscription struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Channels  []string  `json:"channels"` // push, email, sms
	Events    []string  `json:"events"`   // transaction, swap, staking, etc
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

type NotificationService struct {
	notifications []Notification
	subscriptions map[string]*Subscription
}

func NewNotificationService() *NotificationService {
	return &NotificationService{
		notifications: make([]Notification, 0),
		subscriptions: make(map[string]*Subscription),
	}
}

func (ns *NotificationService) SendNotification(c *gin.Context) {
	var req struct {
		UserID   string                 `json:"user_id" binding:"required"`
		Type     string                 `json:"type" binding:"required"`
		Title    string                 `json:"title" binding:"required"`
		Body     string                 `json:"body" binding:"required"`
		Data     map[string]interface{} `json:"data"`
		Priority string                 `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	notif := Notification{
		ID:        uuid.New().String(),
		UserID:    req.UserID,
		Type:      req.Type,
		Title:     req.Title,
		Body:      req.Body,
		Data:      req.Data,
		Status:    "sent",
		Channel:   req.Type,
		Priority:  req.Priority,
		CreatedAt: time.Now(),
	}

	ns.notifications = append(ns.notifications, notif)

	c.JSON(http.StatusOK, gin.H{
		"success":         true,
		"notification_id": notif.ID,
		"status":          "sent",
	})
}

func (ns *NotificationService) GetNotifications(c *gin.Context) {
	userID := c.Param("user_id")

	notifs := make([]Notification, 0)
	for _, n := range ns.notifications {
		if n.UserID == userID {
			notifs = append(notifs, n)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"notifications": notifs,
	})
}

// MarkAsRead marks a single notification as read by ID.
func (ns *NotificationService) MarkAsRead(c *gin.Context) {
	notifID := c.Param("id")
	for i := range ns.notifications {
		if ns.notifications[i].ID == notifID {
			ns.notifications[i].Read = true
			c.JSON(http.StatusOK, gin.H{"success": true, "notification": ns.notifications[i]})
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
}

// MarkAllAsRead marks all notifications for a user as read.
func (ns *NotificationService) MarkAllAsRead(c *gin.Context) {
	userID := c.Param("user_id")
	count := 0
	for i := range ns.notifications {
		if ns.notifications[i].UserID == userID && !ns.notifications[i].Read {
			ns.notifications[i].Read = true
			count++
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "marked_read": count})
}

// DeleteNotification removes a single notification by ID.
func (ns *NotificationService) DeleteNotification(c *gin.Context) {
	notifID := c.Param("id")
	for i := range ns.notifications {
		if ns.notifications[i].ID == notifID {
			ns.notifications = append(ns.notifications[:i], ns.notifications[i+1:]...)
			c.JSON(http.StatusOK, gin.H{"success": true})
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
}

// ClearAll removes all notifications for a user.
func (ns *NotificationService) ClearAll(c *gin.Context) {
	userID := c.Param("user_id")
	filtered := ns.notifications[:0]
	for _, n := range ns.notifications {
		if n.UserID != userID {
			filtered = append(filtered, n)
		}
	}
	ns.notifications = filtered
	c.JSON(http.StatusOK, gin.H{"success": true, "cleared": true})
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

	ns.subscriptions[sub.ID] = sub

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"subscription": sub,
	})
}

func (ns *NotificationService) Unsubscribe(c *gin.Context) {
	subID := c.Param("id")

	if _, ok := ns.subscriptions[subID]; !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}

	delete(ns.subscriptions, subID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"status":  "unsubscribed",
	})
}

func (ns *NotificationService) SendBatch(c *gin.Context) {
	var req struct {
		UserIDs []string `json:"user_ids" binding:"required"`
		Title   string   `json:"title" binding:"required"`
		Body    string   `json:"body" binding:"required"`
		Type    string   `json:"type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sent := 0
	for _, userID := range req.UserIDs {
		notif := Notification{
			ID:        uuid.New().String(),
			UserID:    userID,
			Type:      req.Type,
			Title:     req.Title,
			Body:      req.Body,
			Status:    "sent",
			CreatedAt: time.Now(),
		}
		ns.notifications = append(ns.notifications, notif)
		sent++
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"sent":    sent,
		"total":   len(req.UserIDs),
	})
}

func (ns *NotificationService) GetTemplates(c *gin.Context) {
	templates := []map[string]string{
		{"id": "tx_sent", "title": "Transaction Sent", "body": "Your transaction of {{amount}} {{token}} has been sent."},
		{"id": "tx_received", "title": "Transaction Received", "body": "You received {{amount}} {{token}} from {{from}}."},
		{"id": "swap_complete", "title": "Swap Complete", "body": "Your swap of {{from_amount}} {{from_token}} to {{to_amount}} {{to_token}} is complete."},
		{"id": "staking_reward", "title": "Staking Reward", "body": "You received {{amount}} {{token}} in staking rewards."},
		{"id": "nft_sold", "title": "NFT Sold", "body": "Your NFT {{nft_name}} was sold for {{price}} {{token}}."},
		{"id": "price_alert", "title": "Price Alert", "body": "{{token}} has reached {{price}}."},
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"templates": templates,
	})
}

func main() {
	log.Println("TigerWallet Notification Service")
	log.Printf("Starting on port %d", cfg.Port)

	ns := NewNotificationService()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
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

	log.Printf("Server starting on :%d", cfg.Port)
	r.Run(fmt.Sprintf(":%d", cfg.Port))
}
