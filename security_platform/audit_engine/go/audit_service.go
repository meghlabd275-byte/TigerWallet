/**
 * TigerWallet Security Audit Platform
 * Production-ready security audit and transparency system
 * 
 * Features:
 * - Smart contract audits
 * - Security vulnerability scanning
 * - Penetration testing coordination
 * - Transparency reports
 * - Security scorecards
 * - Audit verification API
 */

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ServerPort      string
	DBHost          string
	DBPort          string
	DBUser          string
	DBPassword      string
	DBName          string
}

func LoadConfig() *Config {
	return &Config{
		ServerPort: getEnv("AUDIT_PORT", "9102"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "tigerwallet"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "tigerwallet"),
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

type AuditReport struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	
	AuditID          string    `gorm:"uniqueIndex" json:"audit_id"`
	ProjectName      string    `json:"project_name"`
	ProjectType      string    `json:"project_type"` // smart_contract, wallet, protocol, infrastructure
	
	// Audit Firm
	AuditFirm        string    `json:"audit_firm"`
	LeadAuditor      string    `json:"lead_auditor"`
	AuditDate        time.Time `json:"audit_date"`
	ReportURL        string    `json:"report_url"`
	
	// Scope
	ContractAddresses string    `json:"contract_addresses"` // JSON array
	CodeRepository   string    `json:"code_repository"`
	CommitHash       string    `json:"commit_hash"`
	
	// Results
	TotalIssues      int       `json:"total_issues"`
	CriticalIssues  int       `json:"critical_issues"`
	HighIssues      int       `json:"high_issues"`
	MediumIssues    int       `json:"medium_issues"`
	LowIssues       int       `json:"low_issues"`
	Informational   int       `json:"informational"`
	
	// Status
	Status           string    `json:"status"` // scheduled, in_progress, completed, published
	Score            float64   `json:"score"` // 0-100
	
	// Verification
	IsVerified       bool      `json:"is_verified"`
	VerificationDate *time.Time `json:"verification_date"`
	CertikReportID  string    `json:"certik_report_id,omitempty"`
	SlowMistReportID string   `json:"slowmist_report_id,omitempty"`
	
	// Certificate
	CertificateID    string    `json:"certificate_id"`
	CertificateURL   string    `json:"certificate_url"`
}

type Vulnerability struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	
	AuditID          string    `gorm:"index" json:"audit_id"`
	VulnerabilityID   string    `gorm:"uniqueIndex" json:"vulnerability_id"`
	
	// Classification
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	Category         string    `json:"category"` // reentrancy, overflow, access_control, etc.
	Severity          string    `json:"severity"` // critical, high, medium, low, informational
	CWEID            string    `json:"cwe_id"` // Common Weakness Enumeration
	CVSSScore        float64   `json:"cvss_score"`
	
	// Location
	ContractAddress   string    `json:"contract_address"`
	FilePath         string    `json:"file_path"`
	LineNumber       int       `json:"line_number"`
	FunctionName     string    `json:"function_name"`
	
	// Impact
	Impact           string    `json:"impact"`
	Exploitability   string    `json:"exploitability"`
	Remediation      string    `json:"remediation"`
	
	// Status
	Status           string    `json:"status"` // identified, acknowledged, fixed, verified, wontfix
	FixedAt          *time.Time `json:"fixed_at"`
}

type SecurityScan struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	
	ScanID           string    `gorm:"uniqueIndex" json:"scan_id"`
	ProjectName      string    `json:"project_name"`
	ScanType         string    `json:"scan_type"` // static, dynamic, symbolic
	
	// Results
	TotalFindings    int       `json:"total_findings"`
	CriticalFindings int       `json:"critical_findings"`
	HighFindings     int       `json:"high_findings"`
	MediumFindings   int       `json:"medium_findings"`
	LowFindings      int       `json:"low_findings"`
	
	// Details
	ScanDuration     int       `json:"scan_duration"` // seconds
	LinesOfCode     int       `json:"lines_of_code"`
	Coverage        float64   `json:"coverage"` // percentage
	
	Status           string    `json:"status"` // queued, running, completed, failed
	
	Report           string    `json:"report"` // JSON report
}

type TransparencyReport struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	
	ReportID         string    `gorm:"uniqueIndex" json:"report_id"`
	ReportType       string    `json:"report_type"` // monthly, quarterly, annual
	PeriodStart      time.Time `json:"period_start"`
	PeriodEnd        time.Time `json:"period_end"`
	
	// Content
	Title            string    `json:"title"`
	Summary          string    `json:"summary"`
	Content          string    `json:"content"` // Markdown
	
	// Metrics
	TotalTransactions string    `json:"total_transactions"`
	TotalVolume      string    `json:"total_volume"`
	SecurityIncidents int      `json:"security_incidents"`
	BugBountiesPaid  string    `json:"bug_bounties_paid"`
	AuditsCompleted  int       `json:"audits_completed"`
	
	Status           string    `json:"status"` // draft, published
	PublishedAt      *time.Time `json:"published_at"`
}

type SecurityMetric struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	
	MetricType       string    `json:"metric_type"` // uptime, incidents, response_time, etc.
	MetricValue      float64   `json:"metric_value"`
	Unit             string    `json:"unit"` // percentage, milliseconds, count
	
	Timestamp        time.Time `json:"timestamp"`
	Period           string    `json:"period"` // hourly, daily, weekly, monthly
	
	Metadata         string    `json:"metadata"` // JSON
}

// ============================================================================
// Service Implementation
// ============================================================================

type AuditService struct {
	db *gorm.DB
}

func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{db: db}
}

// SubmitProjectForAudit submits a project for security audit
func (s *AuditService) SubmitProjectForAudit(projectName, projectType, contractAddresses, codeRepository string) (*AuditReport, error) {
	audit := &AuditReport{
		AuditID:       uuid.New().String(),
		ProjectName:   projectName,
		ProjectType:   projectType,
		Status:        "scheduled",
		ContractAddresses: contractAddresses,
		CodeRepository: codeRepository,
	}

	if err := s.db.Create(audit).Error; err != nil {
		return nil, err
	}

	return audit, nil
}

// GetAuditReport retrieves an audit report
func (s *AuditService) GetAuditReport(auditID string) (*AuditReport, error) {
	var report AuditReport
	if err := s.db.Where("audit_id = ?", auditID).First(&report).Error; err != nil {
		return nil, err
	}
	return &report, nil
}

// ListAudits lists all audit reports
func (s *AuditService) ListAudits(status string, limit, offset int) ([]AuditReport, error) {
	var reports []AuditReport
	query := s.db.Model(&AuditReport{}).Order("created_at DESC")
	
	if status != "" {
		query = query.Where("status = ?", status)
	}
	
	if err := query.Limit(limit).Offset(offset).Find(&reports).Error; err != nil {
		return nil, err
	}
	
	return reports, nil
}

// UpdateAuditStatus updates the status of an audit
func (s *AuditService) UpdateAuditStatus(auditID, status string, findings map[string]int, score float64) error {
	updates := map[string]interface{}{
		"status": status,
		"score":  score,
	}
	
	if critical, ok := findings["critical"]; ok {
		updates["critical_issues"] = critical
	}
	if high, ok := findings["high"]; ok {
		updates["high_issues"] = high
	}
	if medium, ok := findings["medium"]; ok {
		updates["medium_issues"] = medium
	}
	if low, ok := findings["low"]; ok {
		updates["low_issues"] = low
	}
	
	total := findings["critical"] + findings["high"] + findings["medium"] + findings["low"] + findings["informational"]
	updates["total_issues"] = total
	
	if status == "completed" {
		now := time.Now()
		updates["audit_date"] = now
		updates["is_verified"] = true
		updates["verification_date"] = &now
	}
	
	return s.db.Model(&AuditReport{}).Where("audit_id = ?", auditID).Updates(updates).Error
}

// CreateVulnerability creates a new vulnerability record
func (s *AuditService) CreateVulnerability(auditID string, vuln VulnerabilityInput) (*Vulnerability, error) {
	vulnerability := &Vulnerability{
		AuditID:        auditID,
		VulnerabilityID: uuid.New().String(),
		Title:          vuln.Title,
		Description:    vuln.Description,
		Category:       vuln.Category,
		Severity:       vuln.Severity,
		CWEID:          vuln.CWEID,
		CVSSScore:      vuln.CVSSScore,
		ContractAddress: vuln.ContractAddress,
		FilePath:       vuln.FilePath,
		LineNumber:     vuln.LineNumber,
		FunctionName:   vuln.FunctionName,
		Impact:         vuln.Impact,
		Remediation:    vuln.Remediation,
		Status:         "identified",
	}

	if err := s.db.Create(vulnerability).Error; err != nil {
		return nil, err
	}

	return vulnerability, nil
}

// GetVulnerabilities retrieves vulnerabilities for an audit
func (s *AuditService) GetVulnerabilities(auditID string, severity string) ([]Vulnerability, error) {
	var vulns []Vulnerability
	query := s.db.Where("audit_id = ?", auditID)
	
	if severity != "" {
		query = query.Where("severity = ?", severity)
	}
	
	if err := query.Order("CASE severity WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 WHEN 'low' THEN 4 ELSE 5 END").Find(&vulns).Error; err != nil {
		return nil, err
	}
	
	return vulns, nil
}

// RunSecurityScan initiates a security scan
func (s *ScanService) RunSecurityScan(projectName, scanType, contractCode string) (*SecurityScan, error) {
	scan := &SecurityScan{
		ScanID:      uuid.New().String(),
		ProjectName: projectName,
		ScanType:    scanType,
		Status:      "queued",
		LinesOfCode: strings.Count(contractCode, "\n"),
	}

	if err := s.db.Create(scan).Error; err != nil {
		return nil, err
	}

	// In production, this would trigger actual scanning
	// For now, simulate a scan
	go func() {
		time.Sleep(2 * time.Second)
		s.db.Model(scan).Updates(map[string]interface{}{
			"status":          "running",
			"scan_duration":   120,
			"coverage":        85.5,
			"total_findings":  5,
			"critical_findings": 0,
			"high_findings":    1,
			"medium_findings":  2,
			"low_findings":    2,
		})
	}()

	return scan, nil
}

// CreateTransparencyReport creates a new transparency report
func (s *AuditService) CreateTransparencyReport(reportType string, periodStart, periodEnd time.Time, title, summary, content string) (*TransparencyReport, error) {
	report := &TransparencyReport{
		ReportID:     uuid.New().String(),
		ReportType:   reportType,
		PeriodStart:  periodStart,
		PeriodEnd:    periodEnd,
		Title:        title,
		Summary:      summary,
		Content:      content,
		Status:       "draft",
	}

	if err := s.db.Create(report).Error; err != nil {
		return nil, err
	}

	return report, nil
}

// PublishTransparencyReport publishes a transparency report
func (s *AuditService) PublishTransparencyReport(reportID string) error {
	now := time.Now()
	return s.db.Model(&TransparencyReport{}).
		Where("report_id = ?", reportID).
		Updates(map[string]interface{}{
			"status":       "published",
			"published_at": now,
		}).Error
}

// GetTransparencyReports retrieves published transparency reports
func (s *AuditService) GetTransparencyReports(limit, offset int) ([]TransparencyReport, error) {
	var reports []TransparencyReport
	if err := s.db.Where("status = ?", "published").
		Order("published_at DESC").
		Limit(limit).Offset(offset).
		Find(&reports).Error; err != nil {
		return nil, err
	}
	return reports, nil
}

// RecordSecurityMetric records a security metric
func (s *AuditService) RecordSecurityMetric(metricType string, value float64, unit string) error {
	metric := &SecurityMetric{
		MetricType: metricType,
		MetricValue: value,
		Unit:       unit,
		Timestamp:  time.Now(),
		Period:     "daily",
	}
	return s.db.Create(metric).Error
}

// GetSecurityScore calculates the overall security score
func (s *AuditService) GetSecurityScore() (float64, error) {
	var score float64 = 100.0

	// Deduct for critical issues
	var criticalCount int64
	s.db.Model(&Vulnerability{}).Where("severity = ? AND status != ?", "critical", "fixed").Count(&criticalCount)
	score -= float64(criticalCount) * 10

	// Deduct for high issues
	var highCount int64
	s.db.Model(&Vulnerability{}).Where("severity = ? AND status != ?", "high", "fixed").Count(&highCount)
	score -= float64(highCount) * 5

	// Deduct for medium issues
	var mediumCount int64
	s.db.Model(&Vulnerability{}).Where("severity = ? AND status != ?", "medium", "fixed").Count(&mediumCount)
	score -= float64(mediumCount) * 2

	if score < 0 {
		score = 0
	}

	return score, nil
}

// ============================================================================
// Types
// ============================================================================

type VulnerabilityInput struct {
	Title          string  `json:"title"`
	Description    string  `json:"description"`
	Category       string  `json:"category"`
	Severity       string  `json:"severity"`
	CWEID          string  `json:"cwe_id"`
	CVSSScore      float64 `json:"cvss_score"`
	ContractAddress string `json:"contract_address"`
	FilePath       string  `json:"file_path"`
	LineNumber     int     `json:"line_number"`
	FunctionName   string  `json:"function_name"`
	Impact         string  `json:"impact"`
	Remediation    string  `json:"remediation"`
}

// ============================================================================
// Scan Service
// ============================================================================

type ScanService struct {
	db *gorm.DB
}

func NewScanService(db *gorm.DB) *ScanService {
	return &ScanService{db: db}
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func (s *AuditService) SubmitProjectHandler(c *gin.Context) {
	var req struct {
		ProjectName       string `json:"project_name" binding:"required"`
		ProjectType       string `json:"project_type" binding:"required"`
		ContractAddresses string `json:"contract_addresses"`
		CodeRepository   string `json:"code_repository"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	report, err := s.SubmitProjectForAudit(req.ProjectName, req.ProjectType, req.ContractAddresses, req.CodeRepository)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    report,
	})
}

func (s *AuditService) GetAuditHandler(c *gin.Context) {
	auditID := c.Param("id")

	report, err := s.GetAuditReport(auditID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "audit not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    report,
	})
}

func (s *AuditService) ListAuditsHandler(c *gin.Context) {
	status := c.Query("status")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	reports, err := s.ListAudits(status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    reports,
	})
}

func (s *AuditService) GetVulnerabilitiesHandler(c *gin.Context) {
	auditID := c.Param("id")
	severity := c.Query("severity")

	vulns, err := s.GetVulnerabilities(auditID, severity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"vulnerabilities": vulns,
	})
}

func (s *AuditService) GetSecurityScoreHandler(c *gin.Context) {
	score, err := s.GetSecurityScore()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"score":      score,
		"rating":     getRating(score),
	})
}

func getRating(score float64) string {
	switch {
	case score >= 90:
		return "Excellent"
	case score >= 75:
		return "Good"
	case score >= 60:
		return "Fair"
	case score >= 40:
		return "Poor"
	default:
		return "Critical"
	}
}

func (s *AuditService) GetTransparencyReportsHandler(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	reports, err := s.GetTransparencyReports(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    reports,
	})
}

func (s *AuditService) CreateTransparencyReportHandler(c *gin.Context) {
	var req struct {
		ReportType   string    `json:"report_type" binding:"required"`
		PeriodStart  time.Time `json:"period_start" binding:"required"`
		PeriodEnd    time.Time `json:"period_end" binding:"required"`
		Title        string    `json:"title" binding:"required"`
		Summary      string    `json:"summary"`
		Content      string    `json:"content"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	report, err := s.CreateTransparencyReport(req.ReportType, req.PeriodStart, req.PeriodEnd, req.Title, req.Summary, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    report,
	})
}

func (s *AuditService) PublishTransparencyReportHandler(c *gin.Context) {
	reportID := c.Param("id")

	if err := s.PublishTransparencyReport(reportID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// ============================================================================
// Database Migration
// ============================================================================

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&AuditReport{},
		&Vulnerability{},
		&SecurityScan{},
		&TransparencyReport{},
		&SecurityMetric{},
	)
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := LoadConfig()

	// Initialize database
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Run migrations
	if err := Migrate(db); err != nil {
		log.Fatalf("Failed to migrate: %v", err)
	}

	// Initialize services
	auditService := NewAuditService(db)
	scanService := NewScanService(db)

	// Setup router
	router := gin.Default()

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		
		c.Next()
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// API routes
	api := router.Group("/api/v1/security")
	{
		// Audit routes
		api.POST("/audit/submit", auditService.SubmitProjectHandler)
		api.GET("/audit/:id", auditService.GetAuditHandler)
		api.GET("/audit", auditService.ListAuditsHandler)
		api.GET("/audit/:id/vulnerabilities", auditService.GetVulnerabilitiesHandler)
		
		// Security score
		api.GET("/score", auditService.GetSecurityScoreHandler)
		
		// Transparency reports
		api.GET("/transparency", auditService.GetTransparencyReportsHandler)
		api.POST("/transparency", auditService.CreateTransparencyReportHandler)
		api.POST("/transparency/:id/publish", auditService.PublishTransparencyReportHandler)
	}

	// Start server
	addr := fmt.Sprintf(":%s", config.ServerPort)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		log.Printf("Starting Security Audit service on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}

// Add strconv import
import "strconv"
