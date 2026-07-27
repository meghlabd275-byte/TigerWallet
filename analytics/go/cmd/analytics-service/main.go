/**
 * TigerWallet Analytics Service
 * Real-time Analytics & Reporting Platform
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ServerPort string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	RedisHost  string
	RedisPort  string
}

func LoadConfig() *Config {
	return &Config{
		ServerPort: getEnv("ANALYTICS_PORT", "9104"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "tigerwallet"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "tigerwallet"),
		RedisHost:  getEnv("REDIS_HOST", "localhost"),
		RedisPort:  getEnv("REDIS_PORT", "6379"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============================================================================
// Database Models
// ============================================================================

type Event struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	EventID     string    `gorm:"uniqueIndex;size:36" json:"event_id"`
	EventType   string    `gorm:"index" json:"event_type"`
	UserID      uint      `gorm:"index" json:"user_id"`
	SessionID   string    `gorm:"index" json:"session_id"`
	Properties  string    `json:"properties"`
	Timestamp   time.Time `json:"timestamp"`
}

type Metric struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	MetricName  string    `gorm:"index" json:"metric_name"`
	MetricValue float64   `json:"metric_value"`
	Dimension   string    `json:"dimension"`
	Timestamp   time.Time `gorm:"index" json:"timestamp"`
}

type Report struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	ReportID    string    `gorm:"uniqueIndex;size:36" json:"report_id"`
	ReportName  string    `json:"report_name"`
	ReportType  string    `json:"report_type"`
	Period      string    `json:"period"`
	Data        string    `json:"data"`
	CreatedAt   time.Time `json:"created_at"`
}

type Dashboard struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	DashboardID string    `gorm:"uniqueIndex;size:36" json:"dashboard_id"`
	Name        string    `json:"name"`
	OwnerID     uint      `gorm:"index" json:"owner_id"`
	Widgets     string    `json:"widgets"`
	IsPublic    bool      `json:"is_public"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ============================================================================
// Analytics Service
// ============================================================================

type AnalyticsService struct {
	config *Config
	db     *gorm.DB
	redis  *redis.Client
	mu     sync.RWMutex
}

func NewAnalyticsService(cfg *Config) (*AnalyticsService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	db.AutoMigrate(&Event{}, &Metric{}, &Report{}, &Dashboard{})

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: "",
		DB:       0,
	})

	return &AnalyticsService{
		config: cfg,
		db:     db,
		redis:  rdb,
	}, nil
}

// ============================================================================
// Event Tracking
// ============================================================================

func (s *AnalyticsService) TrackEvent(eventType string, userID uint, sessionID string, properties map[string]interface{}) error {
	event := Event{
		EventID:   uuid.New().String(),
		EventType: eventType,
		UserID:    userID,
		SessionID: sessionID,
		Timestamp: time.Now(),
	}

	if properties != nil {
		propsJSON, _ := json.Marshal(properties)
		event.Properties = string(propsJSON)
	}

	s.db.Create(&event)

	// Update real-time counters in Redis
	ctx := context.Background()
	s.redis.Incr(ctx, fmt.Sprintf("event:%s:count", eventType))
	s.redis.ZIncrBy(ctx, "events:timeline", 1, event.EventID)

	return nil
}

func (s *AnalyticsService) GetEventCount(eventType string, startTime, endTime time.Time) (int64, error) {
	var count int64
	s.db.Model(&Event{}).
		Where("event_type = ? AND timestamp BETWEEN ? AND ?", eventType, startTime, endTime).
		Count(&count)
	return count, nil
}

func (s *AnalyticsService) GetUserEvents(userID uint, limit int) ([]Event, error) {
	var events []Event
	s.db.Where("user_id = ?", userID).Order("timestamp desc").Limit(limit).Find(&events)
	return events, nil
}

// ============================================================================
// Metrics Aggregation
// ============================================================================

func (s *AnalyticsService) RecordMetric(metricName string, value float64, dimension string) error {
	metric := Metric{
		MetricName:  metricName,
		MetricValue: value,
		Dimension:  dimension,
		Timestamp:  time.Now(),
	}
	return s.db.Create(&metric).Error
}

func (s *AnalyticsService) GetMetricSum(metricName string, startTime, endTime time.Time) (float64, error) {
	var result struct {
		Sum float64
	}
	s.db.Model(&Metric{}).
		Select("COALESCE(SUM(metric_value), 0) as sum").
		Where("metric_name = ? AND timestamp BETWEEN ? AND ?", metricName, startTime, endTime).
		Scan(&result)
	return result.Sum, nil
}

func (s *AnalyticsService) GetMetricAverage(metricName string, startTime, endTime time.Time) (float64, error) {
	var result struct {
		Avg float64
	}
	s.db.Model(&Metric{}).
		Select("COALESCE(AVG(metric_value), 0) as avg").
		Where("metric_name = ? AND timestamp BETWEEN ? AND ?", metricName, startTime, endTime).
		Scan(&result)
	return result.Avg, nil
}

func (s *AnalyticsService) GetMetricsByDimension(metricName string, startTime, endTime time.Time) (map[string]float64, error) {
	var metrics []Metric
	s.db.Where("metric_name = ? AND timestamp BETWEEN ? AND ?", metricName, startTime, endTime).
		Find(&metrics)

	result := make(map[string]float64)
	for _, m := range metrics {
		result[m.Dimension] += m.MetricValue
	}
	return result, nil
}

// ============================================================================
// Dashboard & Reports
// ============================================================================

func (s *AnalyticsService) CreateDashboard(name string, ownerID uint, widgets []map[string]interface{}) (*Dashboard, error) {
	widgetsJSON, _ := json.Marshal(widgets)

	dashboard := Dashboard{
		DashboardID: uuid.New().String(),
		Name:        name,
		OwnerID:     ownerID,
		Widgets:     string(widgetsJSON),
		IsPublic:    false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := s.db.Create(&dashboard).Error
	return &dashboard, err
}

func (s *AnalyticsService) GetDashboard(dashboardID string) (*Dashboard, error) {
	var dashboard Dashboard
	err := s.db.Where("dashboard_id = ?", dashboardID).First(&dashboard).Error
	return &dashboard, err
}

func (s *AnalyticsService) GetUserDashboards(userID uint) ([]Dashboard, error) {
	var dashboards []Dashboard
	err := s.db.Where("owner_id = ?", userID).Find(&dashboards).Error
	return dashboards, err
}

func (s *AnalyticsService) GenerateReport(reportName, reportType, period string, data map[string]interface{}) (*Report, error) {
	dataJSON, _ := json.Marshal(data)

	report := Report{
		ReportID:   uuid.New().String(),
		ReportName: reportName,
		ReportType: reportType,
		Period:     period,
		Data:       string(dataJSON),
		CreatedAt:  time.Now(),
	}

	err := s.db.Create(&report).Error
	return &report, err
}

// ============================================================================
// Pre-built Analytics
// ============================================================================

type OverviewStats struct {
	TotalUsers        int64   `json:"total_users"`
	ActiveUsers24h    int64   `json:"active_users_24h"`
	ActiveUsers7d     int64   `json:"active_users_7d"`
	TotalTransactions int64   `json:"total_transactions"`
	Volume24h         float64 `json:"volume_24h"`
	Volume7d          float64 `json:"volume_7d"`
	Revenue24h        float64 `json:"revenue_24h"`
	Revenue7d         float64 `json:"revenue_7d"`
}

func (s *AnalyticsService) GetOverviewStats() (*OverviewStats, error) {
	now := time.Now()
	dayAgo := now.AddDate(0, 0, -1)
	weekAgo := now.AddDate(0, 0, -7)

	stats := &OverviewStats{}

	// Total users
	s.db.Model(&Event{}).Where("event_type = ?", "user_created").Count(&stats.TotalUsers)

	// Active users 24h
	s.db.Model(&Event{}).Where("timestamp > ?", dayAgo).Distinct("user_id").Count((*gorm.DB)(&stats.ActiveUsers24h)).(*gorm.DB).Count(&stats.ActiveUsers24h)

	// Active users 7d
	s.db.Model(&Event{}).Where("timestamp > ?", weekAgo).Distinct("user_id").Count((*gorm.DB)(&stats.ActiveUsers7d)).(*gorm.DB).Count(&stats.ActiveUsers7d)

	// Total transactions
	s.db.Model(&Event{}).Where("event_type IN ?", []string{"swap", "transfer", "buy", "sell"}).Count(&stats.TotalTransactions)

	// Volume 24h
	s.GetMetricSum("volume", dayAgo, now).Then(func(v float64) { stats.Volume24h = v })

	// Volume 7d
	s.GetMetricSum("volume", weekAgo, now).Then(func(v float64) { stats.Volume7d = v })

	return stats, nil
}

type UserActivity struct {
	Date       string `json:"date"`
	NewUsers   int64  `json:"new_users"`
	ActiveUsers int64 `json:"active_users"`
	Transactions int64 `json:"transactions"`
}

func (s *AnalyticsService) GetUserActivity(days int) ([]UserActivity, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	var events []Event
	s.db.Where("timestamp > ?", startDate).Order("timestamp asc").Find(&events)

	activityMap := make(map[string]*UserActivity)

	for _, event := range events {
		date := event.Timestamp.Format("2006-01-02")
		if activityMap[date] == nil {
			activityMap[date] = &UserActivity{Date: date}
		}

		switch event.EventType {
		case "user_created":
			activityMap[date].NewUsers++
		case "login", "swap", "transfer":
			activityMap[date].ActiveUsers++
		case "swap", "transfer", "buy", "sell":
			activityMap[date].Transactions++
		}
	}

	activities := make([]UserActivity, 0, len(activityMap))
	for _, a := range activityMap {
		activities = append(activities, *a)
	}

	sort.Slice(activities, func(i, j int) bool {
		return activities[i].Date < activities[j].Date
	})

	return activities, nil
}

type TopAsset struct {
	Symbol   string  `json:"symbol"`
	Volume   float64 `json:"volume"`
	Trades   int64   `json:"trades"`
	Revenue  float64 `json:"revenue"`
}

func (s *AnalyticsService) GetTopAssets(limit int) ([]TopAsset, error) {
	var metrics []Metric
	s.db.Where("metric_name = ?", "asset_volume").
		Order("metric_value desc").
		Limit(limit).
		Find(&metrics)

	topAssets := make([]TopAsset, 0, limit)
	for _, m := range metrics {
		topAssets = append(topAssets, TopAsset{
			Symbol:   m.Dimension,
			Volume:   m.MetricValue,
			Trades:   0,
			Revenue:  0,
		})
	}

	return topAssets, nil
}

type GeographicData struct {
	Country     string  `json:"country"`
	Users      int64   `json:"users"`
	Volume     float64 `json:"volume"`
	Percentage float64 `json:"percentage"`
}

func (s *AnalyticsService) GetGeographicDistribution() ([]GeographicData, error) {
	// Mock geographic data
	data := []GeographicData{
		{Country: "United States", Users: 15000, Volume: 50000000, Percentage: 35},
		{Country: "United Kingdom", Users: 8000, Volume: 25000000, Percentage: 18},
		{Country: "Germany", Users: 6000, Volume: 18000000, Percentage: 13},
		{Country: "Japan", Users: 5000, Volume: 15000000, Percentage: 11},
		{Country: "South Korea", Users: 4000, Volume: 12000000, Percentage: 9},
		{Country: "Others", Users: 12000, Volume: 20000000, Percentage: 14},
	}
	return data, nil
}

type RevenueBreakdown struct {
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
	Percent  float64 `json:"percent"`
}

func (s *AnalyticsService) GetRevenueBreakdown() ([]RevenueBreakdown, error) {
	return []RevenueBreakdown{
		{Category: "Trading Fees", Amount: 1500000, Percent: 45},
		{Category: "Swap Fees", Amount: 800000, Percent: 24},
		{Category: "Withdrawal Fees", Amount: 500000, Percent: 15},
		{Category: "Staking Rewards", Amount: 300000, Percent: 9},
		{Category: "NFT Marketplace", Amount: 200000, Percent: 6},
		{Category: "Other", Amount: 35000, Percent: 1},
	}, nil
}

// ============================================================================
// API Handlers
// ============================================================================

func (s *AnalyticsService) setupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	{
		// Events
		api.POST("/events", s.trackEvent)
		api.GET("/events/:user_id", s.getUserEvents)

		// Metrics
		api.POST("/metrics", s.recordMetric)
		api.GET("/metrics/:metric_name/sum", s.getMetricSum)
		api.GET("/metrics/:metric_name/avg", s.getMetricAverage)
		api.GET("/metrics/:metric_name/dimension", s.getMetricByDimension)

		// Dashboard
		api.POST("/dashboards", s.createDashboard)
		api.GET("/dashboards/:id", s.getDashboard)
		api.GET("/dashboards/user/:user_id", s.getUserDashboards)

		// Reports
		api.POST("/reports", s.createReport)
		api.GET("/reports/:id", s.getReport)

		// Analytics
		api.GET("/overview", s.getOverview)
		api.GET("/activity", s.getActivity)
		api.GET("/top-assets", s.getTopAssets)
		api.GET("/geographic", s.getGeographic)
		api.GET("/revenue", s.getRevenue)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "analytics"})
	})
}

func (s *AnalyticsService) trackEvent(c *gin.Context) {
	var req struct {
		EventType  string                 `json:"event_type" binding:"required"`
		UserID     uint                   `json:"user_id"`
		SessionID  string                 `json:"session_id"`
		Properties map[string]interface{} `json:"properties"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := s.TrackEvent(req.EventType, req.UserID, req.SessionID, req.Properties)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "event tracked"})
}

func (s *AnalyticsService) getUserEvents(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Param("user_id"), 10, 32)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	events, err := s.GetUserEvents(uint(userID), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"events": events})
}

func (s *AnalyticsService) recordMetric(c *gin.Context) {
	var req struct {
		MetricName string  `json:"metric_name" binding:"required"`
		Value      float64 `json:"value" binding:"required"`
		Dimension  string  `json:"dimension"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := s.RecordMetric(req.MetricName, req.Value, req.Dimension)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "metric recorded"})
}

func (s *AnalyticsService) getMetricSum(c *gin.Context) {
	metricName := c.Param("metric_name")
	startTime := time.Now().AddDate(0, 0, -1)
	endTime := time.Now()

	sum, err := s.GetMetricSum(metricName, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"metric": metricName, "sum": sum})
}

func (s *AnalyticsService) getMetricAverage(c *gin.Context) {
	metricName := c.Param("metric_name")
	startTime := time.Now().AddDate(0, 0, -1)
	endTime := time.Now()

	avg, err := s.GetMetricAverage(metricName, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"metric": metricName, "average": avg})
}

func (s *AnalyticsService) getMetricByDimension(c *gin.Context) {
	metricName := c.Param("metric_name")
	startTime := time.Now().AddDate(0, 0, -1)
	endTime := time.Now()

	dimensions, err := s.GetMetricsByDimension(metricName, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"metric": metricName, "dimensions": dimensions})
}

func (s *AnalyticsService) createDashboard(c *gin.Context) {
	var req struct {
		Name     string                   `json:"name" binding:"required"`
		OwnerID  uint                    `json:"owner_id" binding:"required"`
		Widgets  []map[string]interface{} `json:"widgets"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dashboard, err := s.CreateDashboard(req.Name, req.OwnerID, req.Widgets)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"dashboard": dashboard})
}

func (s *AnalyticsService) getDashboard(c *gin.Context) {
	dashboardID := c.Param("id")

	dashboard, err := s.GetDashboard(dashboardID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "dashboard not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"dashboard": dashboard})
}

func (s *AnalyticsService) getUserDashboards(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Param("user_id"), 10, 32)

	dashboards, err := s.GetUserDashboards(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"dashboards": dashboards})
}

func (s *AnalyticsService) createReport(c *gin.Context) {
	var req struct {
		ReportName string                 `json:"report_name" binding:"required"`
		ReportType string                 `json:"report_type" binding:"required"`
		Period     string                 `json:"period" binding:"required"`
		Data       map[string]interface{} `json:"data"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	report, err := s.GenerateReport(req.ReportName, req.ReportType, req.Period, req.Data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"report": report})
}

func (s *AnalyticsService) getReport(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "report details"})
}

func (s *AnalyticsService) getOverview(c *gin.Context) {
	stats, err := s.GetOverviewStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"overview": stats})
}

func (s *AnalyticsService) getActivity(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))

	activity, err := s.GetUserActivity(days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"activity": activity})
}

func (s *AnalyticsService) getTopAssets(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	assets, err := s.GetTopAssets(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"assets": assets})
}

func (s *AnalyticsService) getGeographic(c *gin.Context) {
	data, err := s.GetGeographicDistribution()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"distribution": data})
}

func (s *AnalyticsService) getRevenue(c *gin.Context) {
	data, err := s.GetRevenueBreakdown()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"breakdown": data})
}

// ============================================================================
// Main
// ============================================================================

func main() {
	cfg := LoadConfig()

	service, err := NewAnalyticsService(cfg)
	if err != nil {
		log.Fatalf("Failed to create analytics service: %v", err)
	}

	router := gin.Default()
	service.setupRoutes(router)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down analytics service...")
		os.Exit(0)
	}()

	log.Printf("Analytics Service starting on port %s", cfg.ServerPort)
	if err := router.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
