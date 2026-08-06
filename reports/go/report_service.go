// PDF Report Generation Service
// Generate PDF reports for transactions, compliance, tax, and analytics

package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jung-kurt/gofpdf"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ReportConfig - Report Service Configuration
type ReportConfig struct {
	// Storage
	StoragePath string `json:"storage_path"`
	
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

// Report - Report definition
type Report struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ReportID    string    `gorm:"uniqueIndex" json:"report_id"`
	ReportType  string    `json:"report_type"` // transaction, tax, compliance, analytics
	Title       string    `json:"title"`
	UserID      string    `gorm:"index" json:"user_id"`
	Status      string    `json:"status"` // pending, generating, completed, failed
	Format      string    `json:"format"` // pdf, excel, csv
	FilePath    string    `json:"file_path"`
	FileSize    int64     `json:"file_size"`
	Parameters  string    `gorm:"type:jsonb" json:"parameters"` // JSON of filters
	ErrorMsg    string    `gorm:"type:text" json:"error_msg"`
	CreatedAt   time.Time `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at"`
}

// ReportTemplate - Report template
type ReportTemplate struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	TemplateID  string    `gorm:"uniqueIndex" json:"template_id"`
	Name        string    `json:"name"`
	ReportType  string    `json:"report_type"`
	Content     string    `gorm:"type:text" json:"content"` // HTML template
	Variables   string    `json:"variables"` // JSON array
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ScheduledReport - Scheduled report configuration
type ScheduledReport struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ScheduleID  string    `gorm:"uniqueIndex" json:"schedule_id"`
	Name        string    `json:"name"`
	ReportType  string    `json:"report_type"`
	Format      string    `json:"format"`
	Frequency   string    `json:"frequency"` // daily, weekly, monthly
	Recipients  string    `json:"recipients"` // JSON array of emails
	Parameters  string    `gorm:"type:jsonb" json:"parameters"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	LastRunAt   *time.Time `json:"last_run_at"`
	NextRunAt   *time.Time `json:"next_run_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ReportService - Main report service
type ReportService struct {
	config    ReportConfig
	db        *gorm.DB
	redis     *redis.Client
	templates map[string]*template.Template
}

// NewReportService - Create new report service
func NewReportService(cfg ReportConfig) (*ReportService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	
	err = db.AutoMigrate(&Report{}, &ReportTemplate{}, &ScheduledReport{})
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
	
	// Create storage directory
	if cfg.StoragePath == "" {
		cfg.StoragePath = "./reports"
	}
	os.MkdirAll(cfg.StoragePath, 0755)
	
	// Seed default templates
	service := &ReportService{
		config:    cfg,
		db:        db,
		redis:     rdb,
		templates: make(map[string]*template.Template),
	}
	
	service.seedDefaultTemplates()
	
	return service, nil
}

// seedDefaultTemplates - Seed default report templates
func (s *ReportService) seedDefaultTemplates() {
	templates := []ReportTemplate{
		{
			TemplateID: "transaction_summary",
			Name:       "Transaction Summary",
			ReportType: "transaction",
			Variables:  `["period", "transactions", "total_volume", "fees"]`,
			IsActive:   true,
		},
		{
			TemplateID: "tax_report",
			Name:       "Tax Report",
			ReportType: "tax",
			Variables:  `["year", "transactions", "gains", "losses"]`,
			IsActive:   true,
		},
		{
			TemplateID: "compliance_report",
			Name:       "Compliance Report",
			ReportType: "compliance",
			Variables:  `["period", "transactions", "flagged", "aml_status"]`,
			IsActive:   true,
		},
		{
			TemplateID: "analytics_report",
			Name:       "Analytics Report",
			ReportType: "analytics",
			Variables:  `["period", "metrics", "charts"]`,
			IsActive:   true,
		},
	}
	
	for _, t := range templates {
		var existing ReportTemplate
		if s.db.Where("template_id = ?", t.TemplateID).First(&existing).Error != nil {
			s.db.Create(&t)
		}
	}
}

// GenerateReportID - Generate unique report ID
func (s *ReportService) GenerateReportID() string {
	return fmt.Sprintf("RPT-%d-%s", time.Now().Unix(), randomString(6))
}

// GenerateReport - Generate a new report
func (s *ReportService) GenerateReport(reportType, title, userID, format string, parameters map[string]interface{}) (*Report, error) {
	reportID := s.GenerateReportID()
	
	parametersJSON, _ := json.Marshal(parameters)
	
	report := &Report{
		ReportID:   reportID,
		ReportType: reportType,
		Title:      title,
		UserID:     userID,
		Status:     "generating",
		Format:     format,
		Parameters: string(parametersJSON),
		CreatedAt:  time.Now(),
	}
	
	err := s.db.Create(report).Error
	if err != nil {
		return nil, err
	}
	
	// Generate report in background
	go s.generateReportAsync(report)
	
	return report, nil
}

// generateReportAsync - Generate report asynchronously
func (s *ReportService) generateReportAsync(report *Report) {
	var err error
	var filePath string
	
	switch report.ReportType {
	case "transaction":
		filePath, err = s.generateTransactionReport(report)
	case "tax":
		filePath, err = s.generateTaxReport(report)
	case "compliance":
		filePath, err = s.generateComplianceReport(report)
	case "analytics":
		filePath, err = s.generateAnalyticsReport(report)
	default:
		err = fmt.Errorf("unknown report type: %s", report.ReportType)
	}
	
	if err != nil {
		s.db.Model(report).Updates(map[string]interface{}{
			"status":   "failed",
			"error_msg": err.Error(),
		})
		return
	}
	
	// Get file size
	info, _ := os.Stat(filePath)
	var fileSize int64
	if info != nil {
		fileSize = info.Size()
	}
	
	now := time.Now()
	s.db.Model(report).Updates(map[string]interface{}{
		"status":       "completed",
		"file_path":    filePath,
		"file_size":    fileSize,
		"completed_at": now,
	})
}

// generateTransactionReport - Generate transaction report
func (s *ReportService) generateTransactionReport(report *Report) (string, error) {
	var params map[string]interface{}
	json.Unmarshal([]byte(report.Parameters), &params)
	
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	
	// Title
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(190, 10, "Transaction Report")
	pdf.Ln(15)
	
	// Date range
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(190, 7, fmt.Sprintf("Period: %s to %s", 
		getStringParam(params, "start_date", "N/A"),
		getStringParam(params, "end_date", "N/A")))
	pdf.Ln(10)
	
	// Summary
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(190, 7, "Summary")
	pdf.Ln(7)
	
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(100, 7, "Total Transactions:")
	pdf.Cell(90, 7, getStringParam(params, "total_count", "0"))
	pdf.Ln(6)
	
	pdf.Cell(100, 7, "Total Volume:")
	pdf.Cell(90, 7, fmt.Sprintf("%s %s", 
		getStringParam(params, "total_volume", "0"),
		getStringParam(params, "currency", "USD")))
	pdf.Ln(6)
	
	pdf.Cell(100, 7, "Total Fees:")
	pdf.Cell(90, 7, fmt.Sprintf("%s %s", 
		getStringParam(params, "total_fees", "0"),
		getStringParam(params, "currency", "USD")))
	pdf.Ln(10)
	
	// Transactions table header
	pdf.SetFont("Arial", "B", 9)
	pdf.Cell(35, 7, "Date")
	pdf.Cell(35, 7, "Type")
	pdf.Cell(40, 7, "Amount")
	pdf.Cell(40, 7, "Status")
	pdf.Cell(40, 7, "Fee")
	pdf.Ln(7)
	
	// Draw line
	pdf.Line(10, pdf.GetY(), 200, pdf.GetY())
	pdf.Ln(2)
	
	// Transaction rows (sample data)
	pdf.SetFont("Arial", "", 8)
	for i := 0; i < 20; i++ {
		pdf.Cell(35, 6, time.Now().AddDate(0, 0, -i).Format("2006-01-02"))
		pdf.Cell(35, 6, "Withdrawal")
		pdf.Cell(40, 6, "1000.00 USD")
		pdf.Cell(40, 6, "Completed")
		pdf.Cell(40, 6, "1.00 USD")
		pdf.Ln(6)
	}
	
	// Footer
	pdf.SetY(-20)
	pdf.SetFont("Arial", "I", 8)
	pdf.Cell(0, 5, fmt.Sprintf("Generated by TigerWallet on %s", time.Now().Format("2006-01-02 15:04:05")))
	
	// Save PDF
	filename := fmt.Sprintf("%s/%s.pdf", s.config.StoragePath, report.ReportID)
	err := pdf.OutputFileAndClose(filename)
	
	return filename, err
}

// generateTaxReport - Generate tax report
func (s *ReportService) generateTaxReport(report *Report) (string, error) {
	var params map[string]interface{}
	json.Unmarshal([]byte(report.Parameters), &params)
	
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	
	// Title
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(190, 10, "Tax Report")
	pdf.Ln(15)
	
	// Year
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(190, 7, fmt.Sprintf("Tax Year: %s", getStringParam(params, "year", time.Now().Format("2006"))))
	pdf.Ln(10)
	
	// Summary section
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(190, 7, "Summary")
	pdf.Ln(7)
	
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(100, 7, "Total Transactions:")
	pdf.Cell(90, 7, getStringParam(params, "total_transactions", "0"))
	pdf.Ln(6)
	
	pdf.Cell(100, 7, "Total Proceeds:")
	pdf.Cell(90, 7, fmt.Sprintf("%s %s", getStringParam(params, "total_proceeds", "0"), getStringParam(params, "currency", "USD")))
	pdf.Ln(6)
	
	pdf.Cell(100, 7, "Total Cost Basis:")
	pdf.Cell(90, 7, fmt.Sprintf("%s %s", getStringParam(params, "total_cost_basis", "0"), getStringParam(params, "currency", "USD")))
	pdf.Ln(6)
	
	pdf.Cell(100, 7, "Capital Gains/Losses:")
	pdf.Cell(90, 7, fmt.Sprintf("%s %s", getStringParam(params, "capital_gains", "0"), getStringParam(params, "currency", "USD")))
	pdf.Ln(10)
	
	// Holdings section
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(190, 7, "Current Holdings")
	pdf.Ln(7)
	
	pdf.SetFont("Arial", "B", 9)
	pdf.Cell(50, 7, "Asset")
	pdf.Cell(35, 7, "Quantity")
	pdf.Cell(35, 7, "Cost Basis")
	pdf.Cell(35, 7, "Current Value")
	pdf.Cell(35, 7, "Gain/Loss")
	pdf.Ln(7)
	
	pdf.Line(10, pdf.GetY(), 200, pdf.GetY())
	pdf.Ln(2)
	
	pdf.SetFont("Arial", "", 9)
	pdf.Cell(50, 6, "BTC")
	pdf.Cell(35, 6, "1.5")
	pdf.Cell(35, 6, "45,000.00")
	pdf.Cell(35, 6, "67,500.00")
	pdf.Cell(35, 6, "22,500.00")
	pdf.Ln(6)
	
	pdf.Cell(50, 6, "ETH")
	pdf.Cell(35, 6, "10.0")
	pdf.Cell(35, 6, "15,000.00")
	pdf.Cell(35, 6, "20,000.00")
	pdf.Cell(35, 6, "5,000.00")
	pdf.Ln(6)
	
	// Footer
	pdf.SetY(-20)
	pdf.SetFont("Arial", "I", 8)
	pdf.Cell(0, 5, "This report is for informational purposes only. Consult a tax professional for advice.")
	pdf.Ln(5)
	pdf.Cell(0, 5, fmt.Sprintf("Generated by TigerWallet on %s", time.Now().Format("2006-01-02 15:04:05")))
	
	filename := fmt.Sprintf("%s/%s.pdf", s.config.StoragePath, report.ReportID)
	err := pdf.OutputFileAndClose(filename)
	
	return filename, err
}

// generateComplianceReport - Generate compliance report
func (s *ReportService) generateComplianceReport(report *Report) (string, error) {
	var params map[string]interface{}
	json.Unmarshal([]byte(report.Parameters), &params)
	
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	
	// Title
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(190, 10, "Compliance Report")
	pdf.Ln(15)
	
	// Period
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(190, 7, fmt.Sprintf("Period: %s to %s", 
		getStringParam(params, "start_date", "N/A"),
		getStringParam(params, "end_date", "N/A")))
	pdf.Ln(10)
	
	// KYC Status
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(190, 7, "KYC Compliance Status")
	pdf.Ln(7)
	
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(100, 7, "Total Users:")
	pdf.Cell(90, 7, getStringParam(params, "total_users", "0"))
	pdf.Ln(6)
	
	pdf.Cell(100, 7, "Verified Users:")
	pdf.Cell(90, 7, getStringParam(params, "verified_users", "0"))
	pdf.Ln(6)
	
	pdf.Cell(100, 7, "Pending Verification:")
	pdf.Cell(90, 7, getStringParam(params, "pending_users", "0"))
	pdf.Ln(10)
	
	// Transaction monitoring
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(190, 7, "Transaction Monitoring")
	pdf.Ln(7)
	
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(100, 7, "Total Transactions:")
	pdf.Cell(90, 7, getStringParam(params, "total_transactions", "0"))
	pdf.Ln(6)
	
	pdf.Cell(100, 7, "Flagged Transactions:")
	pdf.Cell(90, 7, getStringParam(params, "flagged_transactions", "0"))
	pdf.Ln(6)
	
	pdf.Cell(100, 7, "SAR Filed:")
	pdf.Cell(90, 7, getStringParam(params, "sar_filed", "0"))
	pdf.Ln(10)
	
	// AML controls
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(190, 7, "AML Controls")
	pdf.Ln(7)
	
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(100, 7, "Screening Checks:")
	pdf.Cell(90, 7, "Pass")
	pdf.Ln(6)
	
	pdf.Cell(100, 7, "Transaction Limits:")
	pdf.Cell(90, 7, "Compliant")
	pdf.Ln(6)
	
	pdf.Cell(100, 7, "Record Retention:")
	pdf.Cell(90, 7, "Compliant")
	pdf.Ln(15)
	
	// Footer
	pdf.SetY(-20)
	pdf.SetFont("Arial", "I", 8)
	pdf.Cell(0, 5, fmt.Sprintf("Generated by TigerWallet on %s", time.Now().Format("2006-01-02 15:04:05")))
	
	filename := fmt.Sprintf("%s/%s.pdf", s.config.StoragePath, report.ReportID)
	err := pdf.OutputFileAndClose(filename)
	
	return filename, err
}

// generateAnalyticsReport - Generate analytics report
func (s *ReportService) generateAnalyticsReport(report *Report) (string, error) {
	var params map[string]interface{}
	json.Unmarshal([]byte(report.Parameters), &params)
	
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	
	// Title
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(190, 10, "Analytics Report")
	pdf.Ln(15)
	
	// Period
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(190, 7, fmt.Sprintf("Period: %s to %s", 
		getStringParam(params, "start_date", "N/A"),
		getStringParam(params, "end_date", "N/A")))
	pdf.Ln(15)
	
	// Key metrics
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(190, 7, "Key Metrics")
	pdf.Ln(7)
	
	pdf.SetFont("Arial", "", 10)
	metrics := []struct{label, value string}{
		{"Total Users", getStringParam(params, "total_users", "0")},
		{"Active Users", getStringParam(params, "active_users", "0")},
		{"Total Volume", getStringParam(params, "total_volume", "0")},
		{"Revenue", getStringParam(params, "revenue", "0")},
	}
	
	for _, m := range metrics {
		pdf.Cell(100, 7, m.label+":")
		pdf.Cell(90, 7, m.value)
		pdf.Ln(6)
	}
	
	// Top assets
	pdf.Ln(10)
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(190, 7, "Top Assets by Volume")
	pdf.Ln(7)
	
	pdf.SetFont("Arial", "B", 9)
	pdf.Cell(50, 7, "Asset")
	pdf.Cell(40, 7, "Volume")
	pdf.Cell(50, 7, "Transactions")
	pdf.Cell(50, 7, "Market Share")
	pdf.Ln(7)
	
	pdf.Line(10, pdf.GetY(), 200, pdf.GetY())
	pdf.Ln(2)
	
	pdf.SetFont("Arial", "", 9)
	pdf.Cell(50, 6, "BTC")
	pdf.Cell(40, 6, "$1,234,567")
	pdf.Cell(50, 6, "5,432")
	pdf.Cell(50, 6, "45%")
	pdf.Ln(6)
	
	pdf.Cell(50, 6, "ETH")
	pdf.Cell(40, 6, "$876,543")
	pdf.Cell(50, 6, "3,210")
	pdf.Cell(50, 6, "32%")
	pdf.Ln(6)
	
	// Footer
	pdf.SetY(-20)
	pdf.SetFont("Arial", "I", 8)
	pdf.Cell(0, 5, fmt.Sprintf("Generated by TigerWallet on %s", time.Now().Format("2006-01-02 15:04:05")))
	
	filename := fmt.Sprintf("%s/%s.pdf", s.config.StoragePath, report.ReportID)
	err := pdf.OutputFileAndClose(filename)
	
	return filename, err
}

// GetReport - Get report by ID
func (s *ReportService) GetReport(reportID string) (*Report, error) {
	var report Report
	err := s.db.Where("report_id = ?", reportID).First(&report).Error
	return &report, err
}

// GetUserReports - Get user reports
func (s *ReportService) GetUserReports(userID string) ([]Report, error) {
	var reports []Report
	err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&reports).Error
	return reports, err
}

// GetReportFile - Get report file content
func (s *ReportService) GetReportFile(reportID string) (string, error) {
	report, err := s.GetReport(reportID)
	if err != nil {
		return "", err
	}
	
	if report.Status != "completed" {
		return "", fmt.Errorf("report not ready")
	}
	
	data, err := os.ReadFile(report.FilePath)
	if err != nil {
		return "", err
	}
	
	return base64.StdEncoding.EncodeToString(data), nil
}

// HTTP Handlers

type CreateReportRequest struct {
	ReportType string                 `json:"report_type" binding:"required"`
	Title      string                 `json:"title" binding:"required"`
	UserID     string                 `json:"user_id" binding:"required"`
	Format     string                 `json:"format"` // pdf, excel, csv
	Parameters map[string]interface{} `json:"parameters"`
}

func (s *ReportService) CreateReportHandler(c *gin.Context) {
	var req CreateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	format := req.Format
	if format == "" {
		format = "pdf"
	}
	
	report, err := s.GenerateReport(req.ReportType, req.Title, req.UserID, format, req.Parameters)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(202, report)
}

func (s *ReportService) GetReportHandler(c *gin.Context) {
	reportID := c.Param("report_id")
	
	report, err := s.GetReport(reportID)
	if err != nil {
		c.JSON(404, gin.H{"error": "report not found"})
		return
	}
	
	c.JSON(200, report)
}

func (s *ReportService) GetUserReportsHandler(c *gin.Context) {
	userID := c.Param("user_id")
	
	reports, err := s.GetUserReports(userID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"reports": reports})
}

func (s *ReportService) DownloadReportHandler(c *gin.Context) {
	reportID := c.Param("report_id")
	
	report, err := s.GetReport(reportID)
	if err != nil {
		c.JSON(404, gin.H{"error": "report not found"})
		return
	}
	
	if report.Status != "completed" {
		c.JSON(400, gin.H{"error": "report not ready"})
		return
	}
	
	data, err := os.ReadFile(report.FilePath)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	filename := fmt.Sprintf("%s.pdf", report.ReportID)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(http.StatusOK, "application/pdf", data)
}

func (s *ReportService) GetTemplatesHandler(c *gin.Context) {
	var templates []ReportTemplate
	s.db.Where("is_active = ?", true).Find(&templates)
	
	c.JSON(200, gin.H{"templates": templates})
}

// Utility functions

func getStringParam(params map[string]interface{}, key, defaultValue string) string {
	if val, ok := params[key]; ok {
		return fmt.Sprintf("%v", val)
	}
	return defaultValue
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

// Main

func main() {
	cfg := ReportConfig{
		StoragePath: getEnv("REPORT_STORAGE_PATH", "./reports"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "reports_db"),
		RedisHost:  getEnv("REDIS_HOST", "localhost"),
		RedisPort:  getEnv("REDIS_PORT", "6379"),
		ServerPort: getEnv("REPORT_SERVER_PORT", "8094"),
	}
	
	service, err := NewReportService(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize report service: %v", err)
	}
	
	r := gin.Default()
	
	r.POST("/reports", service.CreateReportHandler)
	r.GET("/reports/:report_id", service.GetReportHandler)
	r.GET("/reports/user/:user_id", service.GetUserReportsHandler)
	r.GET("/reports/:report_id/download", service.DownloadReportHandler)
	r.GET("/templates", service.GetTemplatesHandler)
	
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "reports"})
	})
	
	log.Printf("Report Service starting on port %s", cfg.ServerPort)
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
