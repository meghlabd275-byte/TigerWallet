// Push Notifications Service - Go Implementation
// In-app notifications for market events

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Configuration
type NotificationConfig struct {
	ServerPort string `json:"server_port"`
	DBHost     string `json:"db_host"`
	DBPort     string `json:"db_port"`
	DBUser     string `json:"db_user"`
	DBPassword string `json:"db_password"`
	DBName     string `json:"db_name"`
	RedisHost  string `json:"redis_host"`
	RedisPort  string `json:"redis_port"`
}

// Notification types
const (
	TYPE_PRICE_ALERT   = "price_alert"
	TYPE_TRANSACTION  = "transaction"
	TYPE_STAKING      = "staking"
	TYPE_SWAP        = "swap"
	TYPE_SECURITY    = "security"
	TYPE_MARKET      = "market"
	TYPE_SYSTEM      = "system"
)

// NotificationPriority
const (
	PRIORITY_LOW    = "low"
	PRIORITY_MEDIUM = "medium"
	PRIORITY_HIGH   = "high"
)

// Notification represents a notification
type Notification struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	NotifID     string    `gorm:"uniqueIndex" json:"notif_id"`
	UserAddress string    `gorm:"index" json:"user_address"`
	Type       string    `json:"type"`
	Priority   string    `json:"priority"`
	Title      string    `json:"title"`
	Message    string    `json:"message"`
	Data       string    `gorm:"type:jsonb" json:"data"`
	IsRead     bool      `json:"is_read"`
	IsPushed   bool      `json:"is_pushed"`
	CreatedAt  time.Time `json:"created_at"`
	ReadAt     *time.Time `json:"read_at"`
}

// DeviceToken represents a device token
type DeviceToken struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserAddress string    `gorm:"index" json:"user_address"`
	Token      string    `gorm:"uniqueIndex" json:"token"`
	Platform   string    `json:"platform"` // ios, android, web
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// NotificationService
type NotificationService struct {
	db      *gorm.DB
	redis  *redis.Client
	config NotificationConfig
}

// NewNotificationService creates new service
func NewNotificationService(cfg NotificationConfig) (*NotificationService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&Notification{}, &DeviceToken{})
	if err != nil {
		return nil, err
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &NotificationService{
		db:      db,
		redis:  rdb,
		config: cfg,
	}, nil
}

// GenerateNotifID generates a unique notification ID
func (s *NotificationService) GenerateNotifID() string {
	return fmt.Sprintf("notif_%d", time.Now().UnixNano())
}

// CreateNotification creates a notification
func (s *NotificationService) CreateNotification(userAddress, notifType, priority, title, message string, data map[string]interface{}) (*Notification, error) {
	notifID := s.GenerateNotifID()

	dataJSON := "{}"
	if data != nil {
		b, _ := json.Marshal(data)
		dataJSON = string(b)
	}

	notif := &Notification{
		NotifID:     notifID,
		UserAddress: userAddress,
		Type:       notifType,
		Priority:   priority,
		Title:      title,
		Message:   message,
		Data:      dataJSON,
		IsRead:     false,
		IsPushed:   false,
		CreatedAt: time.Now(),
	}

	s.db.Create(notif)

	return notif, nil
}

// GetNotifications gets notifications for a user
func (s *NotificationService) GetNotifications(userAddress string, limit, offset int) ([]Notification, error) {
	var notifs []Notification
	if limit == 0 {
		limit = 50
	}
	err := s.db.Where("user_address = ?", userAddress).Order("created_at DESC").Limit(limit).Offset(offset).Find(&notifs).Error
	return notifs, err
}

// GetUnreadCount gets unread notification count
func (s *NotificationService) GetUnreadCount(userAddress string) (int64, error) {
	var count int64
	err := s.db.Model(&Notification{}).Where("user_address = ? AND is_read = ?", userAddress, false).Count(&count).Error
	return count, err
}

// MarkAsRead marks notifications as read
func (s *NotificationService) MarkAsRead(userAddress string, notifIDs []string) error {
	if len(notifIDs) == 0 {
		return s.db.Model(&Notification{}).Where("user_address = ?", userAddress).Update("is_read", true).Error
	}
	return s.db.Model(&Notification{}).Where("user_address = ? AND notif_id IN ?", userAddress, notifIDs).Update("is_read", true).Error
}

// DeleteNotification deletes a notification
func (s *NotificationService) DeleteNotification(notifID string) error {
	return s.db.Where("notif_id = ?", notifID).Delete(&Notification{}).Error
}

// RegisterDevice registers a device token
func (s *NotificationService) RegisterDevice(userAddress, token, platform string) error {
	result := s.db.Model(&DeviceToken{}).Where("token = ?", token).Updates(map[string]interface{}{
		"user_address": userAddress,
		"platform":   platform,
		"is_active":  true,
		"updated_at": time.Now(),
	})

	if result.RowsAffected == 0 {
		device := DeviceToken{
			UserAddress: userAddress,
			Token:      token,
			Platform:  platform,
			IsActive:   true,
			CreatedAt: time.Now(),
		}
		s.db.Create(&device)
	}

	return nil
}

// UnregisterDevice unregisters a device token
func (s *NotificationService) UnregisterDevice(token string) error {
	return s.db.Model(&DeviceToken{}).Where("token = ?", token).Update("is_active", false).Error
}

// GetDeviceTokens gets device tokens for a user
func (s *NotificationService) GetDeviceTokens(userAddress string) ([]DeviceToken, error) {
	var tokens []DeviceToken
	err := s.db.Where("user_address = ? AND is_active = ?", userAddress, true).Find(&tokens).Error
	return tokens, err
}

// SendPushNotification sends a push notification (placeholder - would integrate with FCM/APNS)
func (s *NotificationService) SendPushNotification(userAddress, title, message string) error {
	tokens, err := s.GetDeviceTokens(userAddress)
	if err != nil || len(tokens) == 0 {
		return fmt.Errorf("no device tokens")
	}

	// In production, integrate with FCM/APNS
	for _, token := range tokens {
		fmt.Printf("Sending push to %s: %s - %s\n", token.Token, title, message)
	}

	// Mark as pushed
	s.db.Model(&Notification{}).Where("user_address = ? AND title = ?", userAddress, title).Update("is_pushed", true)

	return nil
}

// Handlers

type CreateNotifRequest struct {
	UserAddress string                 `json:"user_address" binding:"required"`
	Type      string                 `json:"type" binding:"required"`
	Priority string                 `json:"priority"`
	Title    string                 `json:"title" binding:"required"`
	Message  string                 `json:"message" binding:"required"`
	Data     map[string]interface{} `json:"data"`
}

func (s *NotificationService) CreateHandler(c *gin.Context) {
	var req CreateNotifRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	priority := req.Priority
	if priority == "" {
		priority = PRIORITY_MEDIUM
	}

	notif, err := s.CreateNotification(req.UserAddress, req.Type, priority, req.Title, req.Message, req.Data)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, notif)
}

func (s *NotificationService) GetHandler(c *gin.Context) {
	address := c.Param("address")
	limit := parseInt(c.Query("limit"))
	offset := parseInt(c.Query("offset"))

	notifs, err := s.GetNotifications(address, limit, offset)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, notifs)
}

func (s *NotificationService) GetUnreadHandler(c *gin.Context) {
	address := c.Param("address")
	count, err := s.GetUnreadCount(address)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"count": count})
}

type MarkReadRequest struct {
	NotifIDs []string `json:"notif_ids"`
}

func (s *NotificationService) MarkReadHandler(c *gin.Context) {
	address := c.Param("address")

	var req MarkReadRequest
	c.ShouldBindJSON(&req)

	if err := s.MarkAsRead(address, req.NotifIDs); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "ok"})
}

func (s *NotificationService) DeleteHandler(c *gin.Context) {
	notifID := c.Param("notif_id")

	if err := s.DeleteNotification(notifID); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "deleted"})
}

type RegisterDeviceRequest struct {
	Token    string `json:"token" binding:"required"`
	Platform string `json:"platform" binding:"required"`
}

func (s *NotificationService) RegisterDeviceHandler(c *gin.Context) {
	address := c.Param("address")

	var req RegisterDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if err := s.RegisterDevice(address, req.Token, req.Platform); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "registered"})
}

func (s *NotificationService) UnregisterDeviceHandler(c *gin.Context) {
	token := c.Param("token")

	if err := s.UnregisterDevice(token); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "unregistered"})
}

// Utility functions

func parseInt(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}

// Main

func main() {
	cfg := NotificationConfig{
		ServerPort: getEnv("NOTIF_SERVER_PORT", "8086"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "notifications_db"),
		RedisHost:  getEnv("REDIS_HOST", "localhost"),
		RedisPort:  getEnv("REDIS_PORT", "6379"),
	}

	service, err := NewNotificationService(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize service: %v\n", err)
		os.Exit(1)
	}

	r := gin.Default()

	r.POST("/notifications", service.CreateHandler)
	r.GET("/notifications/:address", service.GetHandler)
	r.GET("/notifications/:address/unread", service.GetUnreadHandler)
	r.POST("/notifications/:address/read", service.MarkReadHandler)
	r.DELETE("/notifications/:notif_id", service.DeleteHandler)
	r.POST("/devices/:address", service.RegisterDeviceHandler)
	r.DELETE("/devices/:token", service.UnregisterDeviceHandler)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	go func() {
		fmt.Printf("Notification Service starting on port %s\n", cfg.ServerPort)
		if err := r.Run(":" + cfg.ServerPort); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down...")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}