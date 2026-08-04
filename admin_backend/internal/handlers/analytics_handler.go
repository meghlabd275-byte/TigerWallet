package handlers

import (
	"net/http"
	"strconv"
	"time"

	"admin_backend/internal/models"
	"admin_backend/pkg/database"

	"github.com/gin-gonic/gin"
)

// AnalyticsHandler handles analytics-related requests
type AnalyticsHandler struct {
	db *database.PostgresDB
}

// NewAnalyticsHandler creates a new analytics handler
func NewAnalyticsHandler(db *database.PostgresDB) *AnalyticsHandler {
	return &AnalyticsHandler{db: db}
}

// DashboardStats represents dashboard statistics
type DashboardStats struct {
	TotalUsers       int64   `json:"total_users"`
	ActiveUsers     int64   `json:"active_users"`
	TotalVolume    float64 `json:"total_volume"`
	PendingKYC    int64   `json:"pending_kyc"`
	SystemHealth   string  `json:"system_health"`
	TodayTransactions int64 `json:"today_transactions"`
	TodayVolume     float64 `json:"today_volume"`
	NewUsersToday  int64   `json:"new_users_today"`
}

// GetDashboard gets dashboard statistics
func (h *AnalyticsHandler) GetDashboard(c *gin.Context) {
	var stats DashboardStats

	// Total users
	h.db.Model(&models.User{}).Count(&stats.TotalUsers)

	// Active users
	h.db.Model(&models.User{}).Where("status = ?", "active").Count(&stats.ActiveUsers)

	// Pending KYC
	h.db.Model(&models.User{}).Where("kyc_status = ?", "pending").Count(&stats.PendingKYC)

	// Today's new users
	today := time.Now().Truncate(24 * time.Hour)
	h.db.Model(&models.User{}).Where("created_at >= ?", today).Count(&stats.NewUsersToday)

	// Today's transactions
	h.db.Model(&models.Transaction{}).Where("created_at >= ?", today).Count(&stats.TodayTransactions)

	stats.SystemHealth = "99.9%"

	c.JSON(http.StatusOK, stats)
}

// UserAnalytics represents user analytics data
type UserAnalytics struct {
	TotalUsers       int64   `json:"total_users"`
	NewUsersToday   int64   `json:"new_users_today"`
	NewUsersWeek    int64   `json:"new_users_week"`
	NewUsersMonth   int64   `json:"new_users_month"`
	ActiveUsers     int64   `json:"active_users"`
	InactiveUsers   int64   `json:"inactive_users"`
	VerifiedUsers   int64   `json:"verified_users"`
	KYCBreakdown    KYCBreakdown `json:"kyc_breakdown"`
}

// KYCBreakdown represents KYC statistics
type KYCBreakdown struct {
	None    int64 `json:"none"`
	Pending int64 `json:"pending"`
	Level1  int64 `json:"level1"`
	Level2  int64 `json:"level2"`
	Level3  int64 `json:"level3"`
}

// GetUserAnalytics gets user analytics
func (h *AnalyticsHandler) GetUserAnalytics(c *gin.Context) {
	var analytics UserAnalytics

	now := time.Now()
	today := now.Truncate(24 * time.Hour)
	weekAgo := today.AddDate(0, 0, -7)
	monthAgo := today.AddDate(0, 0, -30)

	// Total users
	h.db.Model(&models.User{}).Count(&analytics.TotalUsers)

	// New users
	h.db.Model(&models.User{}).Where("created_at >= ?", today).Count(&analytics.NewUsersToday)
	h.db.Model(&models.User{}).Where("created_at >= ?", weekAgo).Count(&analytics.NewUsersWeek)
	h.db.Model(&models.User{}).Where("created_at >= ?", monthAgo).Count(&analytics.NewUsersMonth)

	// Active users
	h.db.Model(&models.User{}).Where("status = ?", "active").Count(&analytics.ActiveUsers)
	h.db.Model(&models.User{}).Where("status != ?", "active").Count(&analytics.InactiveUsers)

	// Verified users (KYC level 2 or higher)
	h.db.Model(&models.User{}).Where("kyc_level >= ?", 2).Count(&analytics.VerifiedUsers)

	// KYC Breakdown
	h.db.Model(&models.User{}).Where("kyc_status = ?", "none").Count(&analytics.KYCBreakdown.None)
	h.db.Model(&models.User{}).Where("kyc_status = ?", "pending").Count(&analytics.KYCBreakdown.Pending)
	h.db.Model(&models.User{}).Where("kyc_status = ?", "level1").Count(&analytics.KYCBreakdown.Level1)
	h.db.Model(&models.User{}).Where("kyc_status = ?", "level2").Count(&analytics.KYCBreakdown.Level2)
	h.db.Model(&models.User{}).Where("kyc_status = ?", "level3").Count(&analytics.KYCBreakdown.Level3)

	c.JSON(http.StatusOK, analytics)
}

// TransactionAnalytics represents transaction analytics
type TransactionAnalytics struct {
	TotalTransactions   int64   `json:"total_transactions"`
	TodayTransactions  int64   `json:"today_transactions"`
	WeekTransactions  int64   `json:"week_transactions"`
	MonthTransactions int64   `json:"month_transactions"`
	TotalVolume       float64 `json:"total_volume"`
	TodayVolume       float64 `json:"today_volume"`
	WeekVolume        float64 `json:"week_volume"`
	MonthVolume       float64 `json:"month_volume"`
	AvgTransactionSize float64 `json:"avg_transaction_size"`
	TypeBreakdown     TypeBreakdown `json:"type_breakdown"`
	StatusBreakdown   StatusBreakdown `json:"status_breakdown"`
}

// TypeBreakdown represents transaction type distribution
type TypeBreakdown struct {
	Transfer int64 `json:"transfer"`
	Swap     int64 `json:"swap"`
	Stake    int64 `json:"stake"`
	Unstake  int64 `json:"unstake"`
	Bridge   int64 `json:"bridge"`
}

// StatusBreakdown represents transaction status distribution
type StatusBreakdown struct {
	Pending   int64 `json:"pending"`
	Confirmed int64 `json:"confirmed"`
	Failed    int64 `json:"failed"`
}

// GetTransactionAnalytics gets transaction analytics
func (h *AnalyticsHandler) GetTransactionAnalytics(c *gin.Context) {
	var analytics TransactionAnalytics

	now := time.Now()
	today := now.Truncate(24 * time.Hour)
	weekAgo := today.AddDate(0, 0, -7)
	monthAgo := today.AddDate(0, 0, -30)

	// Total transactions
	h.db.Model(&models.Transaction{}).Count(&analytics.TotalTransactions)

	// Today's transactions
	h.db.Model(&models.Transaction{}).Where("created_at >= ?", today).Count(&analytics.TodayTransactions)

	// Week transactions
	h.db.Model(&models.Transaction{}).Where("created_at >= ?", weekAgo).Count(&analytics.WeekTransactions)

	// Month transactions
	h.db.Model(&models.Transaction{}).Where("created_at >= ?", monthAgo).Count(&analytics.MonthTransactions)

	// Type breakdown
	h.db.Model(&models.Transaction{}).Where("type = ?", "transfer").Count(&analytics.TypeBreakdown.Transfer)
	h.db.Model(&models.Transaction{}).Where("type = ?", "swap").Count(&analytics.TypeBreakdown.Swap)
	h.db.Model(&models.Transaction{}).Where("type = ?", "stake").Count(&analytics.TypeBreakdown.Stake)
	h.db.Model(&models.Transaction{}).Where("type = ?", "unstake").Count(&analytics.TypeBreakdown.Unstake)
	h.db.Model(&models.Transaction{}).Where("type = ?", "bridge").Count(&analytics.TypeBreakdown.Bridge)

	// Status breakdown
	h.db.Model(&models.Transaction{}).Where("status = ?", "pending").Count(&analytics.StatusBreakdown.Pending)
	h.db.Model(&models.Transaction{}).Where("status = ?", "confirmed").Count(&analytics.StatusBreakdown.Confirmed)
	h.db.Model(&models.Transaction{}).Where("status = ?", "failed").Count(&analytics.StatusBreakdown.Failed)

	// Get volume (in real implementation, would sum the amount column)
	// For now, returning placeholder values
	analytics.TotalVolume = 0
	analytics.TodayVolume = 0
	analytics.WeekVolume = 0
	analytics.MonthVolume = 0
	analytics.AvgTransactionSize = 0

	c.JSON(http.StatusOK, analytics)
}

// RevenueAnalytics represents revenue analytics
type RevenueAnalytics struct {
	TotalRevenue    float64 `json:"total_revenue"`
	TodayRevenue   float64 `json:"today_revenue"`
	WeekRevenue    float64 `json:"week_revenue"`
	MonthRevenue   float64 `json:"month_revenue"`
	FeeBreakdown   FeeBreakdown `json:"fee_breakdown"`
}

// FeeBreakdown represents fee distribution
type FeeBreakdown struct {
	TradingFees   float64 `json:"trading_fees"`
	WithdrawalFees float64 `json:"withdrawal_fees"`
	DepositFees   float64 `json:"deposit_fees"`
	OtherFees     float64 `json:"other_fees"`
}

// GetRevenueAnalytics gets revenue analytics
func (h *AnalyticsHandler) GetRevenueAnalytics(c *gin.Context) {
	var analytics RevenueAnalytics

	// In real implementation, would calculate from transaction fees
	// For now, returning placeholder values
	analytics.TotalRevenue = 0
	analytics.TodayRevenue = 0
	analytics.WeekRevenue = 0
	analytics.MonthRevenue = 0
	analytics.FeeBreakdown.TradingFees = 0
	analytics.FeeBreakdown.WithdrawalFees = 0
	analytics.FeeBreakdown.DepositFees = 0
	analytics.FeeBreakdown.OtherFees = 0

	c.JSON(http.StatusOK, analytics)
}

// GetSystemMetrics gets system metrics
func (h *AnalyticsHandler) GetSystemMetrics(c *gin.Context) {
	var metrics struct {
		Uptime        float64 `json:"uptime"`
		CPUUsage      float64 `json:"cpu_usage"`
		MemoryUsage   float64 `json:"memory_usage"`
		DiskUsage     float64 `json:"disk_usage"`
		APILatency    float64 `json:"api_latency"`
		ActiveConnections int64 `json:"active_connections"`
		RequestsPerSecond float64 `json:"requests_per_second"`
	}

	// In real implementation, would fetch from system monitoring
	metrics.Uptime = 99.99
	metrics.CPUUsage = 45.5
	metrics.MemoryUsage = 62.3
	metrics.DiskUsage = 38.7
	metrics.APILatency = 25.5
	metrics.ActiveConnections = 1250
	metrics.RequestsPerSecond = 450.0

	c.JSON(http.StatusOK, metrics)
}

// GetCustomDateRangeAnalytics gets analytics for a custom date range
func (h *AnalyticsHandler) GetCustomDateRangeAnalytics(c *gin.Context) {
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	if startDateStr == "" || endDateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Start date and end date are required"})
		return
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start date format"})
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end date format"})
		return
	}

	endDate = endDate.Add(24 * time.Hour) // Include the end date

	var analytics struct {
		StartDate       string `json:"start_date"`
		EndDate         string `json:"end_date"`
		NewUsers        int64  `json:"new_users"`
		NewTransactions int64  `json:"new_transactions"`
		TotalVolume     string `json:"total_volume"`
	}

	analytics.StartDate = startDateStr
	analytics.EndDate = endDateStr

	h.db.Model(&models.User{}).Where("created_at >= ? AND created_at < ?", startDate, endDate).Count(&analytics.NewUsers)
	h.db.Model(&models.Transaction{}).Where("created_at >= ? AND created_at < ?", startDate, endDate).Count(&analytics.NewTransactions)
	analytics.TotalVolume = "0"

	c.JSON(http.StatusOK, analytics)
}
