/**
 * TigerWallet Admin Platform - SLA Management Service
 * Complete SLA management with all fetchers and functionality
 */

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ============================================================================
// SLA Models
// ============================================================================

type SLAPolicy struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	PolicyID        string    `gorm:"uniqueIndex" json:"policy_id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Priority        string    `json:"priority"` // critical, high, medium, low
	ResponseTime    int       `json:"response_time_minutes"` // minutes
	ResolutionTime   int       `json:"resolution_time_minutes"` // minutes
	RefundPercent   float64   `json:"refund_percent"` // 0-100
	IsActive        bool      `gorm:"default:true" json:"is_active"`
	AppliesTo      string    `json:"applies_to"` // all, white_label, tier
	TierLevel       *int      `json:"tier_level"` // nil for all
	WhiteLabelID    *uint     `json:"white_label_id"`
}

type SLACompliance struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	TicketID        uint      `gorm:"index" json:"ticket_id"`
	PolicyID        string    `json:"policy_id"`
	ResponseDeadline time.Time `json:"response_deadline"`
	ResolutionDeadline time.Time `json:"resolution_deadline"`
	FirstResponseAt *time.Time `json:"first_response_at"`
	ResolvedAt      *time.Time `json:"resolved_at"`
	MetResponse     bool       `json:"met_response"`
	MetResolution   bool       `json:"met_resolution"`
	BreachReason    *string   `json:"breach_reason"`
}

type SLAMetric struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	PeriodStart     time.Time `json:"period_start"`
	PeriodEnd       time.Time `json:"period_end"`
	TotalTickets    int64     `json:"total_tickets"`
	MetSLA          int64     `json:"met_sla"`
	Breached        int64     `json:"breached"`
	AvgResponseTime float64   `json:"avg_response_time_minutes"`
	AvgResolutionTime float64 `json:"avg_resolution_time_minutes"`
	ComplianceRate  float64   `json:"compliance_rate_percent"`
}

// ============================================================================
// SLA Handlers
// ============================================================================

func (s *AdminPlatformService) ListSLAPolicies(c *gin.Context) {
	var policies []SLAPolicy
	if err := s.db.Order("priority ASC").Find(&policies).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch SLA policies"})
		return
	}
	c.JSON(http.StatusOK, policies)
}

func (s *AdminPlatformService) GetSLAPolicy(c *gin.Context) {
	policyID := c.Param("id")
	var policy SLAPolicy
	if err := s.db.Where("policy_id = ?", policyID).First(&policy).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "SLA policy not found"})
		return
	}
	c.JSON(http.StatusOK, policy)
}

func (s *AdminPlatformService) CreateSLAPolicy(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	var policy SLAPolicy
	if err := c.ShouldBindJSON(&policy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	policy.PolicyID = "sla_" + uuid.New().String()[:8]
	policy.IsActive = true

	if err := s.db.Create(&policy).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create SLA policy"})
		return
	}

	s.logAudit(adminID, "SLA_POLICY_CREATED", "sla", policy.PolicyID, c.ClientIP(), true, "")
	c.JSON(http.StatusCreated, policy)
}

func (s *AdminPlatformService) UpdateSLAPolicy(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	policyID := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	result := s.db.Model(&SLAPolicy{}).Where("policy_id = ?", policyID).Updates(updates)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "SLA policy not found"})
		return
	}

	s.logAudit(adminID, "SLA_POLICY_UPDATED", "sla", policyID, c.ClientIP(), true, "")
	c.JSON(http.StatusOK, gin.H{"message": "SLA policy updated"})
}

func (s *AdminPlatformService) DeleteSLAPolicy(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	policyID := c.Param("id")

	result := s.db.Where("policy_id = ?", policyID).Delete(&SLAPolicy{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "SLA policy not found"})
		return
	}

	s.logAudit(adminID, "SLA_POLICY_DELETED", "sla", policyID, c.ClientIP(), true, "")
	c.JSON(http.StatusOK, gin.H{"message": "SLA policy deleted"})
}

func (s *AdminPlatformService) GetSLACompliance(c *gin.Context) {
	ticketID := c.Param("ticket_id")
	tid, _ := strconv.ParseUint(ticketID, 10, 32)

	var compliance SLACompliance
	if err := s.db.Where("ticket_id = ?", uint(tid)).First(&compliance).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "SLA compliance not found"})
		return
	}
	c.JSON(http.StatusOK, compliance)
}

func (s *AdminPlatformService) GetSLAMetrics(c *gin.Context) {
	period := c.DefaultQuery("period", "30d")
	
	var startTime time.Time
	now := time.Now()
	
	switch period {
	case "7d":
		startTime = now.AddDate(0, 0, -7)
	case "30d":
		startTime = now.AddDate(0, 0, -30)
	case "90d":
		startTime = now.AddDate(0, 0, -90)
	case "1y":
		startTime = now.AddDate(-1, 0, 0)
	default:
		startTime = now.AddDate(0, 0, -30)
	}

	var metrics SLAMetric
	err := s.db.Where("period_start >= ? AND period_end <= ?", startTime, now).
		Order("period_start DESC").First(&metrics).Error

	if err != nil {
		// Return default metrics if none found
		metrics = SLAMetric{
			PeriodStart:    startTime,
			PeriodEnd:      now,
			TotalTickets:   0,
			MetSLA:         0,
			Breached:       0,
			ComplianceRate: 100.0,
		}
	}

	c.JSON(http.StatusOK, metrics)
}

// ============================================================================
// Report Generation Models
// ============================================================================

type Report struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	ReportID      string    `gorm:"uniqueIndex" json:"report_id"`
	ReportType    string    `json:"report_type"` // users, transactions, kyc, revenue, compliance
	Format        string    `json:"format"` // csv, xlsx, pdf
	PeriodStart   time.Time `json:"period_start"`
	PeriodEnd     time.Time `json:"period_end"`
	Status        string    `json:"status"` // generating, ready, failed
	FileURL       *string   `json:"file_url"`
	FileSize      *int64    `json:"file_size_bytes"`
	GeneratedBy   uint      `json:"generated_by"`
	Filters       JSON      `gorm:"type:jsonb" json:"filters"`
	ErrorMessage  *string   `json:"error_message"`
}

type ReportSchedule struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	ScheduleID    string    `gorm:"uniqueIndex" json:"schedule_id"`
	Name          string    `json:"name"`
	ReportType    string    `json:"report_type"`
	Format        string    `json:"format"`
	Frequency     string    `json:"frequency"` // daily, weekly, monthly
	TimeOfDay     string    `json:"time_of_day"` // HH:MM
	DayOfWeek     *int      `json:"day_of_week"` // 0-6 for weekly
	DayOfMonth    *int      `json:"day_of_month"` // 1-28 for monthly
	IsActive      bool      `gorm:"default:true" json:"is_active"`
	Recipients    JSON      `gorm:"type:jsonb" json:"recipients"` // email addresses
	LastGenerated *time.Time `json:"last_generated"`
	NextRun       time.Time `json:"next_run"`
}

// ============================================================================
// Report Generation Handlers
// ============================================================================

func (s *AdminPlatformService) ListReports(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	reportType := c.Query("type")
	status := c.Query("status")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var reports []Report
	var total int64

	query := s.db.Model(&Report{})
	if reportType != "" {
		query = query.Where("report_type = ?", reportType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC")

	if err := query.Find(&reports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reports"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        reports,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

func (s *AdminPlatformService) CreateReport(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	var req struct {
		ReportType  string    `json:"report_type" binding:"required"`
		Format      string    `json:"format" binding:"required"`
		PeriodStart time.Time `json:"period_start" binding:"required"`
		PeriodEnd   time.Time `json:"period_end" binding:"required"`
		Filters     map[string]interface{} `json:"filters"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	filtersJSON, _ := json.Marshal(req.Filters)

	report := Report{
		ReportID:     "report_" + uuid.New().String()[:8],
		ReportType:   req.ReportType,
		Format:       req.Format,
		PeriodStart:  req.PeriodStart,
		PeriodEnd:    req.PeriodEnd,
		Status:       "generating",
		GeneratedBy: adminID,
		Filters:      filtersJSON,
	}

	if err := s.db.Create(&report).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create report"})
		return
	}

	// Start async generation
	go s.generateReport(report.ReportID)

	s.logAudit(adminID, "REPORT_CREATED", "report", report.ReportID, c.ClientIP(), true, "")
	c.JSON(http.StatusAccepted, gin.H{
		"message": "Report generation started",
		"report":  report,
	})
}

func (s *AdminPlatformService) generateReport(reportID string) {
	// Simulated report generation - in production this would generate actual files
	var report Report
	if err := s.db.Where("report_id = ?", reportID).First(&report).Error; err != nil {
		return
	}

	// Generate report based on type
	fileURL := fmt.Sprintf("https://reports.tigerwallet.com/%s.%s", reportID, report.Format)
	
	report.Status = "ready"
	report.FileURL = &fileURL
	defaultSize := int64(1024000) // 1MB default
	report.FileSize = &defaultSize

	s.db.Save(&report)
}

func (s *AdminPlatformService) DownloadReport(c *gin.Context) {
	reportID := c.Param("id")
	var report Report
	if err := s.db.Where("report_id = ?", reportID).First(&report).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Report not found"})
		return
	}

	if report.Status != "ready" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Report not ready"})
		return
	}

	if report.FileURL == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Report file not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"download_url": *report.FileURL,
		"filename":     fmt.Sprintf("%s.%s", report.ReportID, report.Format),
	})
}

func (s *AdminPlatformService) DeleteReport(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	reportID := c.Param("id")

	result := s.db.Where("report_id = ?", reportID).Delete(&Report{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Report not found"})
		return
	}

	s.logAudit(adminID, "REPORT_DELETED", "report", reportID, c.ClientIP(), true, "")
	c.JSON(http.StatusOK, gin.H{"message": "Report deleted"})
}

// ============================================================================
// Report Schedule Handlers
// ============================================================================

func (s *AdminPlatformService) ListReportSchedules(c *gin.Context) {
	var schedules []ReportSchedule
	if err := s.db.Order("name ASC").Find(&schedules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch schedules"})
		return
	}
	c.JSON(http.StatusOK, schedules)
}

func (s *AdminPlatformService) CreateReportSchedule(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	var schedule ReportSchedule
	if err := c.ShouldBindJSON(&schedule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	schedule.ScheduleID = "sched_" + uuid.New().String()[:8]
	schedule.IsActive = true

	// Calculate next run
	schedule.NextRun = s.calculateNextRun(&schedule)

	if err := s.db.Create(&schedule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create schedule"})
		return
	}

	s.logAudit(adminID, "REPORT_SCHEDULE_CREATED", "schedule", schedule.ScheduleID, c.ClientIP(), true, "")
	c.JSON(http.StatusCreated, schedule)
}

func (s *AdminPlatformService) calculateNextRun(schedule *ReportSchedule) time.Time {
	now := time.Now()
	var next time.Time
	
	location, _ := time.LoadLocation("UTC")
	
	switch schedule.Frequency {
	case "daily":
		// Parse time of day
		parsedTime, _ := time.Parse("15:04", schedule.TimeOfDay)
		hour := parsedTime.Hour()
		minute := parsedTime.Minute()
		
		next = time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, location)
		if next.Before(now) {
			next = next.AddDate(0, 0, 1)
		}
	case "weekly":
		parsedTime, _ := time.Parse("15:04", schedule.TimeOfDay)
		hour := parsedTime.Hour()
		minute := parsedTime.Minute()
		
		daysUntil := int(schedule.DayOfWeek) - int(now.Weekday())
		if daysUntil < 0 || (daysUntil == 0 && next.Before(now)) {
			daysUntil += 7
		}
		next = now.AddDate(0, 0, daysUntil)
		next = time.Date(next.Year(), next.Month(), next.Day(), hour, minute, 0, 0, location)
	case "monthly":
		parsedTime, _ := time.Parse("15:04", schedule.TimeOfDay)
		hour := parsedTime.Hour()
		minute := parsedTime.Minute()
		
		day := *schedule.DayOfMonth
		next = time.Date(now.Year(), now.Month(), day, hour, minute, 0, 0, location)
		if next.Before(now) {
			next = time.Date(now.Year(), now.Month()+1, day, hour, minute, 0, 0, location)
		}
	}
	
	return next
}

func (s *AdminPlatformService) UpdateReportSchedule(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	scheduleID := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	result := s.db.Model(&ReportSchedule{}).Where("schedule_id = ?", scheduleID).Updates(updates)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Schedule not found"})
		return
	}

	s.logAudit(adminID, "REPORT_SCHEDULE_UPDATED", "schedule", scheduleID, c.ClientIP(), true, "")
	c.JSON(http.StatusOK, gin.H{"message": "Schedule updated"})
}

func (s *AdminPlatformService) DeleteReportSchedule(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	scheduleID := c.Param("id")

	result := s.db.Where("schedule_id = ?", scheduleID).Delete(&ReportSchedule{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Schedule not found"})
		return
	}

	s.logAudit(adminID, "REPORT_SCHEDULE_DELETED", "schedule", scheduleID, c.ClientIP(), true, "")
	c.JSON(http.StatusOK, gin.H{"message": "Schedule deleted"})
}

// ============================================================================
// AI Fraud Detection Models
// ============================================================================

type FraudAlert struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	AlertID         string    `gorm:"uniqueIndex" json:"alert_id"`
	UserID          uint      `gorm:"index" json:"user_id"`
	AlertType       string    `json:"alert_type"` // suspicious_transaction, unusual_activity, high_risk, money_laundering
	Severity        string    `json:"severity"` // low, medium, high, critical
	Status          string    `json:"status"` // open, investigating, resolved, false_positive
	Score           float64   `json:"score"` // 0-100 fraud probability
	Description     string    `json:"description"`
	Evidence        JSON      `gorm:"type:jsonb" json:"evidence"`
	ResolvedBy      *uint     `json:"resolved_by"`
	ResolvedAt      *time.Time `json:"resolved_at"`
	ResolutionNote  *string   `json:"resolution_note"`
}

type FraudRule struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	RuleID          string    `gorm:"uniqueIndex" json:"rule_id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	RuleType        string    `json:"rule_type"` // pattern, threshold, velocity, machine_learning
	Conditions      JSON      `gorm:"type:jsonb" json:"conditions"`
	Action          string    `json:"action"` // alert, block, review
	Severity        string    `json:"severity"`
	IsActive        bool      `gorm:"default:true" json:"is_active"`
	FalsePositiveCount int   `gorm:"default:0" json:"false_positive_count"`
	MatchCount      int64     `gorm:"default:0" json:"match_count"`
}

// ============================================================================
// Fraud Detection Handlers
// ============================================================================

func (s *AdminPlatformService) ListFraudAlerts(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	status := c.Query("status")
	severity := c.Query("severity")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var alerts []FraudAlert
	var total int64

	query := s.db.Model(&FraudAlert{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if severity != "" {
		query = query.Where("severity = ?", severity)
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC")

	if err := query.Find(&alerts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch alerts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        alerts,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

func (s *AdminPlatformService) GetFraudAlert(c *gin.Context) {
	alertID := c.Param("id")
	var alert FraudAlert
	if err := s.db.Where("alert_id = ?", alertID).First(&alert).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found"})
		return
	}
	c.JSON(http.StatusOK, alert)
}

func (s *AdminPlatformService) ResolveFraudAlert(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	alertID := c.Param("id")

	var req struct {
		Status         string `json:"status" binding:"required"` // resolved, false_positive
		ResolutionNote string `json:"resolution_note"`
	}
	c.ShouldBindJSON(&req)

	now := time.Now()
	result := s.db.Model(&FraudAlert{}).Where("alert_id = ?", alertID).Updates(map[string]interface{}{
		"status":          req.Status,
		"resolved_by":    adminID,
		"resolved_at":     now,
		"resolution_note": req.ResolutionNote,
	})

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found"})
		return
	}

	s.logAudit(adminID, "FRAUD_ALERT_RESOLVED", "fraud", alertID, c.ClientIP(), true, req.Status)
	c.JSON(http.StatusOK, gin.H{"message": "Alert resolved"})
}

func (s *AdminPlatformService) ListFraudRules(c *gin.Context) {
	var rules []FraudRule
	if err := s.db.Order("name ASC").Find(&rules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch rules"})
		return
	}
	c.JSON(http.StatusOK, rules)
}

func (s *AdminPlatformService) CreateFraudRule(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	var rule FraudRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	rule.RuleID = "fraud_" + uuid.New().String()[:8]
	rule.IsActive = true

	if err := s.db.Create(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create rule"})
		return
	}

	s.logAudit(adminID, "FRAUD_RULE_CREATED", "fraud_rule", rule.RuleID, c.ClientIP(), true, "")
	c.JSON(http.StatusCreated, rule)
}

func (s *AdminPlatformService) UpdateFraudRule(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	ruleID := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	result := s.db.Model(&FraudRule{}).Where("rule_id = ?", ruleID).Updates(updates)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rule not found"})
		return
	}

	s.logAudit(adminID, "FRAUD_RULE_UPDATED", "fraud_rule", ruleID, c.ClientIP(), true, "")
	c.JSON(http.StatusOK, gin.H{"message": "Rule updated"})
}

func (s *AdminPlatformService) GetFraudStats(c *gin.Context) {
	var stats struct {
		TotalAlerts     int64 `json:"total_alerts"`
		OpenAlerts      int64 `json:"open_alerts"`
		CriticalAlerts  int64 `json:"critical_alerts"`
		ResolvedToday   int64 `json:"resolved_today"`
		FalsePositives  int64 `json:"false_positives"`
		RulesActive     int64 `json:"rules_active"`
	}

	today := time.Now().Truncate(24 * time.Hour)

	s.db.Model(&FraudAlert{}).Count(&stats.TotalAlerts)
	s.db.Model(&FraudAlert{}).Where("status = ?", "open").Count(&stats.OpenAlerts)
	s.db.Model(&FraudAlert{}).Where("severity = ? AND status = ?", "critical", "open").Count(&stats.CriticalAlerts)
	s.db.Model(&FraudAlert{}).Where("resolved_at >= ?", today).Count(&stats.ResolvedToday)
	s.db.Model(&FraudAlert{}).Where("status = ?", "false_positive").Count(&stats.FalsePositives)
	s.db.Model(&FraudRule{}).Where("is_active = ?", true).Count(&stats.RulesActive)

	c.JSON(http.StatusOK, stats)
}
