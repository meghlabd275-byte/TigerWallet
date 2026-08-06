// Cloudflare Integration Service
// WAF, DDoS protection, DNS management, and security features

package main

import (
	"bytes"
	"context"
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

// CloudflareConfig - Cloudflare Integration Configuration
type CloudflareConfig struct {
	// API Settings
	APIKey     string `json:"api_key"`
	APIEmail   string `json:"api_email"`
	AccountID  string `json:"account_id"`
	ZoneID     string `json:"zone_id"`
	
	// Security Settings
	EnableWAF        bool `json:"enable_waf"`
	EnableDDoS       bool `json:"enable_ddos"`
	EnableRateLimit  bool `json:"enable_rate_limit"`
	EnableBotFight   bool `json:"enable_bot_fight"`
	
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

// WAFRule - WAF Rule configuration
type WAFRule struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	RuleID      string    `gorm:"uniqueIndex" json:"rule_id"`
	Name        string    `json:"name"`
	Action      string    `json:"action"` // block, challenge, allow, log
	Expression  string    `gorm:"type:text" json:"expression"`
	Priority    int       `json:"priority"`
	IsEnabled   bool      `gorm:"default:true" json:"is_enabled"`
	ZoneID      string    `json:"zone_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// IPRule - IP Rule (blacklist/whitelist)
type IPRule struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	IP          string    `gorm:"index" json:"ip"`
	RuleType    string    `json:"rule_type"` // blacklist, whitelist
	Reason      string    `json:"reason"`
	ExpiresAt   *time.Time `json:"expires_at"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	ZoneID      string    `json:"zone_id"`
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   string    `json:"created_by"`
}

// RateLimitRule - Rate limiting rule
type RateLimitRule struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	RuleID      string    `gorm:"uniqueIndex" json:"rule_id"`
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	RequestsPerMinute int `json:"requests_per_minute"`
	Burst       int       `json:"burst"`
	Action      string    `json:"action"` // simulate, block
	IsEnabled   bool      `gorm:"default:true" json:"is_enabled"`
	ZoneID      string    `json:"zone_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// DNSRecord - DNS Record management
type DNSRecord struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	RecordID    string    `gorm:"uniqueIndex" json:"record_id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"` // A, AAAA, CNAME, MX, TXT, etc.
	Content     string    `json:"content"`
	Proxied     bool      `gorm:"default:false" json:"proxied"`
	TTL         int       `json:"ttl"` // 1 = auto
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	ZoneID      string    `json:"zone_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// FirewallEvent - Firewall event log
type FirewallEvent struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	EventID     string    `gorm:"uniqueIndex" json:"event_id"`
	Action      string    `json:"action"`
	SourceIP    string    `gorm:"index" json:"source_ip"`
	UserAgent   string    `json:"user_agent"`
	Country    string    `json:"country"`
	ASN         string    `json:"asn"`
	URL         string    `json:"url"`
	RuleID      string    `json:"rule_id"`
	Timestamp   time.Time `json:"timestamp"`
	CreatedAt   time.Time `json:"created_at"`
}

// CloudflareService - Main Cloudflare integration service
type CloudflareService struct {
	config  CloudflareConfig
	db      *gorm.DB
	redis   *redis.Client
	client  *http.Client
}

// NewCloudflareService - Create new Cloudflare service
func NewCloudflareService(cfg CloudflareConfig) (*CloudflareService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	
	err = db.AutoMigrate(&WAFRule{}, &IPRule{}, &RateLimitRule{}, &DNSRecord{}, &FirewallEvent{})
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
	
	return &CloudflareService{
		config: cfg,
		db:     db,
		redis:  rdb,
		client: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// getHeaders - Get Cloudflare API headers
func (s *CloudflareService) getHeaders() map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + s.config.APIKey,
		"Content-Type":  "application/json",
		"X-Auth-Email":  s.config.APIEmail,
	}
}

// getBaseURL - Get Cloudflare API base URL
func (s *CloudflareService) getBaseURL() string {
	return "https://api.cloudflare.com/client/v4"
}

// callAPI - Make API call to Cloudflare
func (s *CloudflareService) callAPI(method, endpoint string, body []byte) (map[string]interface{}, error) {
	url := s.getBaseURL() + endpoint
	
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	
	for k, v := range s.getHeaders() {
		req.Header.Set(k, v)
	}
	
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	respBody, _ := io.ReadAll(resp.Body)
	
	var result map[string]interface{}
	json.Unmarshal(respBody, &result)
	
	if !result["success"].(bool) {
		return nil, fmt.Errorf("Cloudflare API error: %v", result["errors"])
	}
	
	return result, nil
}

// ==================== WAF Rules ====================

// CreateWAFRule - Create WAF rule
func (s *CloudflareService) CreateWAFRule(name, action, expression string, priority int) (*WAFRule, error) {
	rule := &WAFRule{
		RuleID:     fmt.Sprintf("waf_%d", time.Now().Unix()),
		Name:       name,
		Action:     action,
		Expression: expression,
		Priority:   priority,
		IsEnabled:  true,
		ZoneID:     s.config.ZoneID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	
	// Deploy to Cloudflare if configured
	if s.config.EnableWAF && s.config.APIKey != "" {
		payload := map[string]interface{}{
			"action":     action,
			"expression": expression,
			"priority":   priority,
			"products":   []string{"waf"},
		}
		body, _ := json.Marshal(payload)
		
		result, err := s.callAPI("POST", "/zones/"+s.config.ZoneID+"/firewall/rules", body)
		if err == nil {
			if rules, ok := result["result"].(map[string]interface{}); ok {
				if id, ok := rules["id"].(string); ok {
					rule.RuleID = id
				}
			}
		}
	}
	
	err := s.db.Create(rule).Error
	return rule, err
}

// GetWAFRules - Get all WAF rules
func (s *CloudflareService) GetWAFRules() ([]WAFRule, error) {
	var rules []WAFRule
	err := s.db.Order("priority ASC").Find(&rules).Error
	return rules, err
}

// UpdateWAFRule - Update WAF rule
func (s *CloudflareService) UpdateWAFRule(ruleID string, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()
	return s.db.Model(&WAFRule{}).Where("rule_id = ?", ruleID).Updates(updates).Error
}

// DeleteWAFRule - Delete WAF rule
func (s *CloudflareService) DeleteWAFRule(ruleID string) error {
	// Delete from Cloudflare
	if s.config.APIKey != "" {
		s.callAPI("DELETE", "/zones/"+s.config.ZoneID+"/firewall/rules/"+ruleID, nil)
	}
	
	return s.db.Where("rule_id = ?", ruleID).Delete(&WAFRule{}).Error
}

// ==================== IP Rules ====================

// AddIPRule - Add IP to blacklist/whitelist
func (s *CloudflareService) AddIPRule(ip, ruleType, reason, createdBy string, expiresAt *time.Time) (*IPRule, error) {
	rule := &IPRule{
		IP:        ip,
		RuleType:  ruleType,
		Reason:    reason,
		ExpiresAt: expiresAt,
		IsActive:  true,
		ZoneID:    s.config.ZoneID,
		CreatedAt: time.Now(),
		CreatedBy: createdBy,
	}
	
	// Deploy to Cloudflare
	if s.config.APIKey != "" {
		action := "block"
		if ruleType == "whitelist" {
			action = "allow"
		}
		
		payload := map[string]interface{}{
			"target":     "ip",
			"value":      ip,
			"action":     action,
			"description": reason,
		}
		body, _ := json.Marshal(map[string]interface{}{"items": []map[string]interface{}{payload}})
		
		result, err := s.callAPI("POST", "/zones/"+s.config.ZoneID+"/firewall/access_rules/rules", body)
		if err == nil {
			if rules, ok := result["result"].(map[string]interface{}); ok {
				if id, ok := rules["id"].(string); ok {
					rule.IP = id
				}
			}
		}
	}
	
	err := s.db.Create(rule).Error
	return rule, err
}

// GetIPRules - Get all IP rules
func (s *CloudflareService) GetIPRules(ruleType string) ([]IPRule, error) {
	query := s.db.Where("is_active = ?", true)
	
	if ruleType != "" {
		query = query.Where("rule_type = ?", ruleType)
	}
	
	var rules []IPRule
	err := query.Order("created_at DESC").Find(&rules).Error
	return rules, err
}

// RemoveIPRule - Remove IP rule
func (s *CloudflareService) RemoveIPRule(ip string) error {
	rule := IPRule{}
	if err := s.db.Where("ip = ?", ip).First(&rule).Error; err != nil {
		return err
	}
	
	// Remove from Cloudflare
	if s.config.APIKey != "" {
		s.callAPI("DELETE", "/zones/"+s.config.ZoneID+"/firewall/access_rules/rules/"+ip, nil)
	}
	
	return s.db.Delete(&rule).Error
}

// CheckIP - Check if IP is blocked
func (s *CloudflareService) CheckIP(ip string) (bool, string, error) {
	var rule IPRule
	err := s.db.Where("ip = ? AND is_active = ? AND (expires_at IS NULL OR expires_at > ?)", 
		ip, true, time.Now()).First(&rule).Error
	
	if err == gorm.ErrRecordNotFound {
		return false, "", nil
	}
	
	if err != nil {
		return false, "", err
	}
	
	return true, rule.RuleType, nil
}

// ==================== Rate Limiting ====================

// CreateRateLimitRule - Create rate limiting rule
func (s *CloudflareService) CreateRateLimitRule(name, path string, requestsPerMinute, burst int, action string) (*RateLimitRule, error) {
	rule := &RateLimitRule{
		RuleID:            fmt.Sprintf("ratelimit_%d", time.Now().Unix()),
		Name:              name,
		Path:              path,
		RequestsPerMinute: requestsPerMinute,
		Burst:            burst,
		Action:           action,
		IsEnabled:        true,
		ZoneID:           s.config.ZoneID,
		CreatedAt:        time.Now(),
		UpdatedAt:         time.Now(),
	}
	
	// Deploy to Cloudflare
	if s.config.EnableRateLimit && s.config.APIKey != "" {
		payload := map[string]interface{}{
			"threshold":   requestsPerMinute,
			"period":      60,
			"action":      map[string]interface{}{"mode": action},
			"match":       map[string]interface{}{
				"request": map[string]interface{}{
					"url_pattern": path,
				},
			},
		}
		body, _ := json.Marshal(payload)
		
		result, err := s.callAPI("POST", "/zones/"+s.config.ZoneID+"/rate_limits", body)
		if err == nil {
			if rules, ok := result["result"].(map[string]interface{}); ok {
				if id, ok := rules["id"].(string); ok {
					rule.RuleID = id
				}
			}
		}
	}
	
	err := s.db.Create(rule).Error
	return rule, err
}

// GetRateLimitRules - Get all rate limit rules
func (s *CloudflareService) GetRateLimitRules() ([]RateLimitRule, error) {
	var rules []RateLimitRule
	err := s.db.Where("is_enabled = ?", true).Find(&rules).Error
	return rules, err
}

// DeleteRateLimitRule - Delete rate limit rule
func (s *CloudflareService) DeleteRateLimitRule(ruleID string) error {
	if s.config.APIKey != "" {
		s.callAPI("DELETE", "/zones/"+s.config.ZoneID+"/rate_limits/"+ruleID, nil)
	}
	
	return s.db.Where("rule_id = ?", ruleID).Delete(&RateLimitRule{}).Error
}

// ==================== DNS Management ====================

// CreateDNSRecord - Create DNS record
func (s *CloudflareService) CreateDNSRecord(name, recordType, content string, proxied bool, ttl int) (*DNSRecord, error) {
	record := &DNSRecord{
		RecordID: fmt.Sprintf("dns_%d", time.Now().Unix()),
		Name:     name,
		Type:     recordType,
		Content: content,
		Proxied:  proxied,
		TTL:      ttl,
		IsActive: true,
		ZoneID:   s.config.ZoneID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	// Deploy to Cloudflare
	if s.config.APIKey != "" {
		payload := map[string]interface{}{
			"name":    name,
			"type":    recordType,
			"content": content,
			"proxied": proxied,
			"ttl":     ttl,
		}
		body, _ := json.Marshal(payload)
		
		result, err := s.callAPI("POST", "/zones/"+s.config.ZoneID+"/dns_records", body)
		if err == nil {
			if dnsRecord, ok := result["result"].(map[string]interface{}); ok {
				if id, ok := dnsRecord["id"].(string); ok {
					record.RecordID = id
				}
			}
		}
	}
	
	err := s.db.Create(record).Error
	return record, err
}

// GetDNSRecords - Get all DNS records
func (s *CloudflareService) GetDNSRecords(recordType string) ([]DNSRecord, error) {
	query := s.db.Where("is_active = ?", true)
	
	if recordType != "" {
		query = query.Where("type = ?", recordType)
	}
	
	var records []DNSRecord
	err := query.Order("name ASC").Find(&records).Error
	return records, err
}

// UpdateDNSRecord - Update DNS record
func (s *CloudflareService) UpdateDNSRecord(recordID string, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()
	return s.db.Model(&DNSRecord{}).Where("record_id = ?", recordID).Updates(updates).Error
}

// DeleteDNSRecord - Delete DNS record
func (s *CloudflareService) DeleteDNSRecord(recordID string) error {
	if s.config.APIKey != "" {
		s.callAPI("DELETE", "/zones/"+s.config.ZoneID+"/dns_records/"+recordID, nil)
	}
	
	return s.db.Where("record_id = ?", recordID).Delete(&DNSRecord{}).Error
}

// ==================== Firewall Events ====================

// LogFirewallEvent - Log firewall event
func (s *CloudflareService) LogFirewallEvent(eventID, action, sourceIP, userAgent, country, asn, url, ruleID string) error {
	event := &FirewallEvent{
		EventID:   eventID,
		Action:    action,
		SourceIP:  sourceIP,
		UserAgent:  userAgent,
		Country:   country,
		ASN:       asn,
		URL:       url,
		RuleID:    ruleID,
		Timestamp: time.Now(),
		CreatedAt: time.Now(),
	}
	
	return s.db.Create(event).Error
}

// GetFirewallEvents - Get firewall events
func (s *CloudflareService) GetFirewallEvents(ip string, limit int) ([]FirewallEvent, error) {
	query := s.db
	
	if ip != "" {
		query = query.Where("source_ip = ?", ip)
	}
	
	if limit == 0 {
		limit = 100
	}
	
	var events []FirewallEvent
	err := query.Order("timestamp DESC").Limit(limit).Find(&events).Error
	return events, err
}

// ==================== Analytics ====================

// GetSecurityStats - Get security statistics
func (s *CloudflareService) GetSecurityStats() (map[string]interface{}, error) {
	var blocked, allowed, challenge int64
	
	s.db.Model(&FirewallEvent{}).Where("action = ?", "block").Count(&blocked)
	s.db.Model(&FirewallEvent{}).Where("action = ?", "allow").Count(&allowed)
	s.db.Model(&FirewallEvent{}).Where("action = ?", "challenge").Count(&challenge)
	
	var blacklistCount, whitelistCount int64
	s.db.Model(&IPRule{}).Where("rule_type = ? AND is_active = ?", "blacklist", true).Count(&blacklistCount)
	s.db.Model(&IPRule{}).Where("rule_type = ? AND is_active = ?", "whitelist", true).Count(&whitelistCount)
	
	return map[string]interface{}{
		"blocked_requests":   blocked,
		"allowed_requests":   allowed,
		"challenges":        challenge,
		"blacklisted_ips":   blacklistCount,
		"whitelisted_ips":   whitelistCount,
	}, nil
}

// HTTP Handlers

type CreateWAFRuleRequest struct {
	Name       string `json:"name" binding:"required"`
	Action     string `json:"action" binding:"required"`
	Expression string `json:"expression" binding:"required"`
	Priority   int    `json:"priority"`
}

func (s *CloudflareService) CreateWAFRuleHandler(c *gin.Context) {
	var req CreateWAFRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	priority := req.Priority
	if priority == 0 {
		priority = 1
	}
	
	rule, err := s.CreateWAFRule(req.Name, req.Action, req.Expression, priority)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(201, rule)
}

func (s *CloudflareService) GetWAFRulesHandler(c *gin.Context) {
	rules, err := s.GetWAFRules()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"rules": rules})
}

func (s *CloudflareService) DeleteWAFRuleHandler(c *gin.Context) {
	ruleID := c.Param("rule_id")
	
	err := s.DeleteWAFRule(ruleID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"status": "deleted"})
}

type AddIPRuleRequest struct {
	IP        string     `json:"ip" binding:"required"`
	RuleType  string     `json:"rule_type" binding:"required"` // blacklist, whitelist
	Reason    string     `json:"reason"`
	ExpiresAt *time.Time `json:"expires_at"`
}

func (s *CloudflareService) AddIPRuleHandler(c *gin.Context) {
	var req AddIPRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	createdBy := c.GetString("user_id")
	
	rule, err := s.AddIPRule(req.IP, req.RuleType, req.Reason, createdBy, req.ExpiresAt)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(201, rule)
}

func (s *CloudflareService) GetIPRulesHandler(c *gin.Context) {
	ruleType := c.Query("type")
	
	rules, err := s.GetIPRules(ruleType)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"rules": rules})
}

func (s *CloudflareService) RemoveIPRuleHandler(c *gin.Context) {
	ip := c.Param("ip")
	
	err := s.RemoveIPRule(ip)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"status": "removed"})
}

type CreateRateLimitRequest struct {
	Name              string `json:"name" binding:"required"`
	Path              string `json:"path" binding:"required"`
	RequestsPerMinute int    `json:"requests_per_minute" binding:"required"`
	Burst             int    `json:"burst"`
	Action            string `json:"action"`
}

func (s *CloudflareService) CreateRateLimitHandler(c *gin.Context) {
	var req CreateRateLimitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	action := req.Action
	if action == "" {
		action = "block"
	}
	
	burst := req.Burst
	if burst == 0 {
		burst = req.RequestsPerMinute
	}
	
	rule, err := s.CreateRateLimitRule(req.Name, req.Path, req.RequestsPerMinute, burst, action)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(201, rule)
}

func (s *CloudflareService) GetRateLimitsHandler(c *gin.Context) {
	rules, err := s.GetRateLimitRules()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"rules": rules})
}

type CreateDNSRecordRequest struct {
	Name     string `json:"name" binding:"required"`
	Type     string `json:"type" binding:"required"`
	Content  string `json:"content" binding:"required"`
	Proxied  bool   `json:"proxied"`
	TTL      int    `json:"ttl"`
}

func (s *CloudflareService) CreateDNSRecordHandler(c *gin.Context) {
	var req CreateDNSRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	ttl := req.TTL
	if ttl == 0 {
		ttl = 1 // auto
	}
	
	record, err := s.CreateDNSRecord(req.Name, req.Type, req.Content, req.Proxied, ttl)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(201, record)
}

func (s *CloudflareService) GetDNSRecordsHandler(c *gin.Context) {
	recordType := c.Query("type")
	
	records, err := s.GetDNSRecords(recordType)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"records": records})
}

func (s *CloudflareService) GetFirewallEventsHandler(c *gin.Context) {
	ip := c.Query("ip")
	limit := 100
	fmt.Sscanf(c.DefaultQuery("limit", "100"), "%d", &limit)
	
	events, err := s.GetFirewallEvents(ip, limit)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"events": events})
}

func (s *CloudflareService) GetSecurityStatsHandler(c *gin.Context) {
	stats, err := s.GetSecurityStats()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, stats)
}

// Main

func main() {
	cfg := CloudflareConfig{
		APIKey:       getEnv("CLOUDFLARE_API_KEY", ""),
		APIEmail:     getEnv("CLOUDFLARE_EMAIL", ""),
		AccountID:    getEnv("CLOUDFLARE_ACCOUNT_ID", ""),
		ZoneID:       getEnv("CLOUDFLARE_ZONE_ID", ""),
		EnableWAF:    getEnvBool("CLOUDFLARE_ENABLE_WAF", true),
		EnableDDoS:   getEnvBool("CLOUDFLARE_ENABLE_DDOS", true),
		EnableRateLimit: getEnvBool("CLOUDFLARE_ENABLE_RATE_LIMIT", true),
		DBHost:       getEnv("DB_HOST", "localhost"),
		DBPort:       getEnv("DB_PORT", "5432"),
		DBUser:       getEnv("DB_USER", "postgres"),
		DBPassword:   getEnv("DB_PASSWORD", "password"),
		DBName:       getEnv("DB_NAME", "cloudflare_db"),
		RedisHost:    getEnv("REDIS_HOST", "localhost"),
		RedisPort:    getEnv("REDIS_PORT", "6379"),
		ServerPort:   getEnv("CLOUDFLARE_SERVER_PORT", "8095"),
	}
	
	service, err := NewCloudflareService(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize Cloudflare service: %v", err)
	}
	
	r := gin.Default()
	
	r.POST("/waf/rules", service.CreateWAFRuleHandler)
	r.GET("/waf/rules", service.GetWAFRulesHandler)
	r.DELETE("/waf/rules/:rule_id", service.DeleteWAFRuleHandler)
	
	r.POST("/ip-rules", service.AddIPRuleHandler)
	r.GET("/ip-rules", service.GetIPRulesHandler)
	r.DELETE("/ip-rules/:ip", service.RemoveIPRuleHandler)
	
	r.POST("/rate-limits", service.CreateRateLimitHandler)
	r.GET("/rate-limits", service.GetRateLimitsHandler)
	
	r.POST("/dns-records", service.CreateDNSRecordHandler)
	r.GET("/dns-records", service.GetDNSRecordsHandler)
	
	r.GET("/firewall/events", service.GetFirewallEventsHandler)
	r.GET("/security/stats", service.GetSecurityStatsHandler)
	
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "cloudflare"})
	})
	
	log.Printf("Cloudflare Service starting on port %s", cfg.ServerPort)
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

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1"
	}
	return defaultValue
}
