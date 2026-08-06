// Slack Integration Service
// Webhooks, notifications, and bot functionality for Slack

package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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

// SlackConfig - Slack Integration Configuration
type SlackConfig struct {
	// OAuth Settings
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	SigningSecret string `json:"signing_secret"`
	
	// Bot Settings
	BotToken     string `json:"bot_token"`
	AppToken     string `json:"app_token"`
	DefaultChannel string `json:"default_channel"`
	
	// Webhook Settings
	WebhookURL string `json:"webhook_url"`
	
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

// SlackWorkspace - Connected Slack workspace
type SlackWorkspace struct {
	ID              string    `gorm:"primaryKey" json:"id"`
	TeamID          string    `gorm:"uniqueIndex" json:"team_id"`
	TeamName        string    `json:"team_name"`
	AccessToken     string    `gorm:"type:text" json:"-"`
	BotUserID       string    `json:"bot_user_id"`
	BotToken        string    `gorm:"type:text" json:"-"`
	WebhookURL      string    `json:"webhook_url"`
	ChannelID       string    `json:"channel_id"`
	IsActive        bool      `gorm:"default:true" json:"is_active"`
	InstalledAt     time.Time `json:"installed_at"`
	InstalledBy     string    `json:"installed_by"`
}

// SlackChannel - Slack channel configuration
type SlackChannel struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	WorkspaceID   string    `gorm:"index" json:"workspace_id"`
	ChannelID     string    `gorm:"uniqueIndex" json:"channel_id"`
	ChannelName   string    `json:"channel_name"`
	Topic         string    `json:"topic"`
	Notifications string    `json:"notifications"` // JSON: which notifications to send
	IsActive      bool      `gorm:"default:true" json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// SlackMessage - Slack message log
type SlackMessage struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	WorkspaceID string    `gorm:"index" json:"workspace_id"`
	ChannelID   string    `gorm:"index" json:"channel_id"`
	MessageTS   string    `json:"message_ts"`
	MessageID   string    `gorm:"index" json:"message_id"`
	Content     string    `gorm:"type:text" json:"content"`
	MessageType string    `json:"message_type"` // alert, notification, command
	Status      string    `json:"status"` // sent, failed
	Error       string    `gorm:"type:text" json:"error"`
	CreatedAt   time.Time `json:"created_at"`
}

// SlackService - Main Slack integration service
type SlackService struct {
	config  SlackConfig
	db      *gorm.DB
	redis   *redis.Client
	client  *http.Client
}

// NewSlackService - Create new Slack service
func NewSlackService(cfg SlackConfig) (*SlackService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	
	err = db.AutoMigrate(&SlackWorkspace{}, &SlackChannel{}, &SlackMessage{})
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
	
	return &SlackService{
		config: cfg,
		db:     db,
		redis:  rdb,
		client: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// VerifySignature - Verify Slack request signature
func (s *SlackService) VerifySignature(timestamp, signature, body string) bool {
	if s.config.SigningSecret == "" {
		return true
	}
	
	baseString := fmt.Sprintf("v0:%s:%s", timestamp, body)
	mac := hmac.New(sha256.New, []byte(s.config.SigningSecret))
	mac.Write([]byte(baseString))
	expectedSig := "v0=" + base64.StdEncoding.EncodeToString(mac.Sum(nil))
	
	return hmac.Equal([]byte(signature), []byte(expectedSig))
}

// SendMessage - Send message to Slack channel
func (s *SlackService) SendMessage(channel, text string, blocks []interface{}) (string, error) {
	var payload map[string]interface{}
	
	if blocks != nil {
		payload = map[string]interface{}{
			"channel": channel,
			"text":    text,
			"blocks":  blocks,
		}
	} else {
		payload = map[string]interface{}{
			"channel": channel,
			"text":    text,
		}
	}
	
	body, _ := json.Marshal(payload)
	
	req, err := http.NewRequest("POST", "https://slack.com/api/chat.postMessage", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.config.BotToken)
	
	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	respBody, _ := io.ReadAll(resp.Body)
	
	var result map[string]interface{}
	json.Unmarshal(respBody, &result)
	
	if !result["ok"].(bool) {
		return "", fmt.Errorf("slack error: %v", result["error"])
	}
	
	ts := result["ts"].(string)
	
	// Log
	s.logMessage(channel, ts, text, "notification", "sent", "")
	
	return ts, nil
}

// SendAlert - Send alert to Slack
func (s *SlackService) SendAlert(channel, alertType, title, message, severity string) error {
	color := "#36a64f" // green
	if severity == "high" {
		color = "#ff0000"
	} else if severity == "medium" {
		color = "#ffa500"
	} else if severity == "low" {
		color = "#ffff00"
	}
	
	blocks := []interface{}{
		map[string]interface{}{
			"type": "header",
			"text": map[string]interface{}{
				"type": "plain_text",
				"text": title,
			},
		},
		map[string]interface{}{
			"type": "section",
			"text": map[string]interface{}{
				"type": "mrkdwn",
				"text": message,
			},
		},
		map[string]interface{}{
			"type": "context",
			"elements": []interface{}{
				map[string]interface{}{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*Type:* %s | *Severity:* %s | *Time:* %s", alertType, severity, time.Now().Format("15:04:05")),
				},
			},
		},
	}
	
	_, err := s.SendMessage(channel, title, blocks)
	return err
}

// SendTransactionAlert - Send transaction alert
func (s *SlackService) SendTransactionAlert(channel, txType, amount, currency, address, status string) error {
	title := fmt.Sprintf("Transaction %s", status)
	message := fmt.Sprintf("*Type:* %s\n*Amount:* %s %s\n*Address:* `%s`", txType, amount, currency, address)
	
	return s.SendAlert(channel, "transaction", title, message, "high")
}

// SendSecurityAlert - Send security alert
func (s *SlackService) SendSecurityAlert(channel, alertType, description, ipAddress string) error {
	title := "Security Alert - " + alertType
	message := fmt.Sprintf("%s\n\n*IP Address:* `%s`\n*Time:* %s", description, ipAddress, time.Now().Format("2006-01-02 15:04:05"))
	
	return s.SendAlert(channel, "security", title, message, "critical")
}

// SendKYCAlert - Send KYC alert
func (s *SlackService) SendKYCAlert(channel, status, userID, reason string) error {
	title := fmt.Sprintf("KYC Status: %s", status)
	message := fmt.Sprintf("*User ID:* %s\n*Status:* %s\n*Reason:* %s", userID, status, reason)
	severity := "medium"
	if status == "rejected" {
		severity = "high"
	}
	
	return s.SendAlert(channel, "kyc", title, message, severity)
}

// SendWithdrawalAlert - Send withdrawal alert
func (s *SlackService) SendWithdrawalAlert(channel, userID, amount, currency, status string) error {
	title := fmt.Sprintf("Withdrawal %s", status)
	message := fmt.Sprintf("*User:* %s\n*Amount:* %s %s\n*Status:* %s", userID, amount, currency, status)
	
	return s.SendAlert(channel, "withdrawal", title, message, "high")
}

// HandleSlashCommand - Handle Slack slash command
func (s *SlackService) HandleSlashCommand(command, text, userID, channelID string) (string, string, []interface{}) {
	parts := strings.Fields(text)
	
	switch command {
	case "/balance":
		// Return user balance
		return "Balance Check", "Your balance is being fetched...", nil
	
	case "/deposit":
		// Return deposit address
		return "Deposit Address", "Your deposit address is: `0x...`", nil
	
	case "/withdraw":
		// Process withdrawal
		if len(parts) < 2 {
			return "Withdraw", "Usage: /withdraw <amount> <currency>", nil
		}
		return "Withdraw", fmt.Sprintf("Withdrawal request for %s %s submitted for processing", parts[0], parts[1]), nil
	
	case "/price":
		// Return price
		if len(parts) < 1 {
			return "Price", "Usage: /price <token>", nil
		}
		return "Price", fmt.Sprintf("Fetching price for %s...", parts[0]), nil
	
	case "/help":
		blocks := []interface{}{
			{
				"type": "section",
				"text": map[string]interface{}{
					"type": "mrkdwn",
					"text": "*Available Commands:*\n• `/balance` - Check your balance\n• `/deposit` - Get deposit address\n• `/withdraw <amount> <currency>` - Withdraw funds\n• `/price <token>` - Get token price\n• `/help` - Show this help message",
				},
			},
		}
		return "Help", "Available commands:", blocks
	
	default:
		return "Unknown Command", "Type /help for available commands", nil
	}
}

// logMessage - Log Slack message
func (s *SlackService) logMessage(channelID, messageTS, content, msgType, status, errorMsg string) {
	msg := &SlackMessage{
		ChannelID:   channelID,
		MessageTS:   messageTS,
		MessageID:   messageTS,
		Content:     content,
		MessageType: msgType,
		Status:      status,
		Error:       errorMsg,
		CreatedAt:   time.Now(),
	}
	s.db.Create(msg)
}

// HTTP Handlers

type SendMessageRequest struct {
	Channel string                 `json:"channel" binding:"required"`
	Text    string                 `json:"text" binding:"required"`
	Blocks  []interface{}         `json:"blocks"`
}

func (s *SlackService) SendMessageHandler(c *gin.Context) {
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	ts, err := s.SendMessage(req.Channel, req.Text, req.Blocks)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"status": "sent", "message_ts": ts})
}

type SendAlertRequest struct {
	Channel   string `json:"channel" binding:"required"`
	AlertType string `json:"alert_type" binding:"required"`
	Title     string `json:"title" binding:"required"`
	Message   string `json:"message" binding:"required"`
	Severity  string `json:"severity"`
}

func (s *SlackService) SendAlertHandler(c *gin.Context) {
	var req SendAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	severity := req.Severity
	if severity == "" {
		severity = "medium"
	}
	
	err := s.SendAlert(req.Channel, req.AlertType, req.Title, req.Message, severity)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"status": "sent"})
}

func (s *SlackService) SlackEventsHandler(c *gin.Context) {
	// Verify signature
	timestamp := c.GetHeader("X-Slack-Request-Timestamp")
	signature := c.GetHeader("X-Slack-Signature")
	
	body, _ := io.ReadAll(c.Request.Body)
	
	if !s.VerifySignature(timestamp, signature, string(body)) {
		c.JSON(401, gin.H{"error": "invalid signature"})
		return
	}
	
	var payload map[string]interface{}
	json.Unmarshal(body, &payload)
	
	// Handle URL verification challenge
	if payload["type"] == "url_verification" {
		c.JSON(200, gin.H{"challenge": payload["challenge"]})
		return
	}
	
	// Handle event callbacks
	if payload["type"] == "event_callback" {
		event := payload["event"].(map[string]interface{})
		eventType := event["type"].(string)
		
		log.Printf("Received Slack event: %s", eventType)
	}
	
	c.JSON(200, gin.H{"status": "ok"})
}

func (s *SlackService) SlashCommandHandler(c *gin.Context) {
	command := c.PostForm("command")
	text := c.PostForm("text")
	userID := c.PostForm("user_id")
	channelID := c.PostForm("channel_id")
	
	responseText, ephemeralText, blocks := s.HandleSlashCommand(command, text, userID, channelID)
	
	if blocks != nil {
		c.JSON(200, gin.H{
			"response_type": "ephemeral",
			"text":         responseText,
			"blocks":       blocks,
		})
	} else {
		c.JSON(200, gin.H{
			"response_type": "ephemeral",
			"text":         ephemeralText,
		})
	}
}

func (s *SlackService) InteractiveHandler(c *gin.Context) {
	payload := c.PostForm("payload")
	
	var data map[string]interface{}
	json.Unmarshal([]byte(payload), &data)
	
	// Handle button clicks, menu selections, etc.
	action := data["actions"].([]interface{})[0].(map[string]interface{})
	actionType := action["type"].(string)
	
	log.Printf("Interactive action: %s", actionType)
	
	c.JSON(200, gin.H{"status": "ok"})
}

// Main

func main() {
	cfg := SlackConfig{
		ClientID:        getEnv("SLACK_CLIENT_ID", ""),
		ClientSecret:    getEnv("SLACK_CLIENT_SECRET", ""),
		SigningSecret:   getEnv("SLACK_SIGNING_SECRET", ""),
		BotToken:        getEnv("SLACK_BOT_TOKEN", ""),
		AppToken:        getEnv("SLACK_APP_TOKEN", ""),
		DefaultChannel:  getEnv("SLACK_DEFAULT_CHANNEL", "#alerts"),
		WebhookURL:      getEnv("SLACK_WEBHOOK_URL", ""),
		DBHost:          getEnv("DB_HOST", "localhost"),
		DBPort:          getEnv("DB_PORT", "5432"),
		DBUser:          getEnv("DB_USER", "postgres"),
		DBPassword:      getEnv("DB_PASSWORD", "password"),
		DBName:          getEnv("DB_NAME", "slack_db"),
		RedisHost:       getEnv("REDIS_HOST", "localhost"),
		RedisPort:       getEnv("REDIS_PORT", "6379"),
		ServerPort:      getEnv("SLACK_SERVER_PORT", "8091"),
	}
	
	service, err := NewSlackService(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize Slack service: %v", err)
	}
	
	r := gin.Default()
	
	r.POST("/slack/message", service.SendMessageHandler)
	r.POST("/slack/alert", service.SendAlertHandler)
	r.POST("/slack/events", service.SlackEventsHandler)
	r.POST("/slack/commands", service.SlashCommandHandler)
	r.POST("/slack/interactive", service.InteractiveHandler)
	
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "slack"})
	})
	
	log.Printf("Slack Service starting on port %s", cfg.ServerPort)
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
