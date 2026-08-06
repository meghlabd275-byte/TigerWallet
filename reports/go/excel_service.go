// Excel Report Generation Service
// Generate Excel reports for transactions, analytics, tax, and compliance

package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/xuri/excelize/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ReportConfig - Report Service Configuration
type ReportConfig struct {
	StoragePath string `json:"storage_path"`
	DBHost     string `json:"db_host"`
	DBPort     string `json:"db_port"`
	DBUser     string `json:"db_user"`
	DBPassword string `json:"db_password"`
	DBName     string `json:"db_name"`
	RedisHost  string `json:"redis_host"`
	RedisPort  string `json:"redis_port"`
	ServerPort string `json:"server_port"`
}

// ExcelReport - Excel report definition
type ExcelReport struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ReportID    string    `gorm:"uniqueIndex" json:"report_id"`
	ReportType  string    `json:"report_type"`
	Title       string    `json:"title"`
	UserID      string    `gorm:"index" json:"user_id"`
	Status      string    `json:"status"`
	FilePath    string    `json:"file_path"`
	FileSize    int64     `json:"file_size"`
	Parameters  string    `gorm:"type:jsonb" json:"parameters"`
	CreatedAt   time.Time `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at"`
}

// ExcelReportService - Main Excel report service
type ExcelReportService struct {
	config    ReportConfig
	db        *gorm.DB
	redis     *redis.Client
}

// NewExcelReportService - Create new report service
func NewExcelReportService(cfg ReportConfig) (*ExcelReportService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	
	err = db.AutoMigrate(&ExcelReport{})
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
	
	if cfg.StoragePath == "" {
		cfg.StoragePath = "./reports/excel"
	}
	os.MkdirAll(cfg.StoragePath, 0755)
	
	return &ExcelReportService{
		config: cfg,
		db:     db,
		redis:  rdb,
	}, nil
}

// GenerateReportID - Generate unique report ID
func (s *ExcelReportService) GenerateReportID() string {
	return fmt.Sprintf("XLS-%d-%s", time.Now().Unix(), randomString(6))
}

// CreateTransactionReport - Create transaction report in Excel
func (s *ExcelReportService) CreateTransactionReport(reportType, title, userID string, params map[string]interface{}) (*ExcelReport, error) {
	reportID := s.GenerateReportID()
	
	report := &ExcelReport{
		ReportID:   reportID,
		ReportType: reportType,
		Title:      title,
		UserID:      userID,
		Status:      "generating",
		CreatedAt:  time.Now(),
	}
	
	s.db.Create(report)
	
	// Generate Excel in background
	go s.generateTransactionExcel(report, params)
	
	return report, nil
}

func (s *ExcelReportService) generateTransactionExcel(report *ExcelReport, params map[string]interface{}) {
	f := excelize.NewFile()
	defer f.Close()
	
	// Set default sheet name
	sheetName := "Transactions"
	
	// Styles
	headerStyle, _ := f.NewStyle(`{"font":{"bold":true,"color":"#FFFFFF"},"fill":{"type":"pattern","color":["#4F46E5"],"pattern":1}}`)
	numberStyle, _ := f.NewStyle(`{"num_format":"#,##0.00"}`)
	dateStyle, _ := f.NewStyle(`{"num_format":"yyyy-mm-dd hh:mm:ss"}`)
	
	// Title
	f.SetCellValue(sheetName, "A1", "Transaction Report")
	f.SetCellStyle(sheetName, "A1", "A1", headerStyle)
	f.MergeCell(sheetName, "A1", "G1")
	
	// Date range
	f.SetCellValue(sheetName, "A2", "Generated:")
	f.SetCellValue(sheetName, "B2", time.Now().Format("2006-01-02 15:04:05"))
	
	// Headers
	headers := []string{"Date", "Type", "Asset", "Amount", "Fee", "Status", "TX Hash"}
	for i, header := range headers {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetCellValue(sheetName, col+"4", header)
		f.SetCellStyle(sheetName, col+"4", col+"4", headerStyle)
	}
	
	// Sample data (in production, fetch from database)
	transactions := [][]interface{}{
		{time.Now().AddDate(0, 0, -1), "Deposit", "BTC", 1.5, 0.0001, "Completed", "0x1234...5678"},
		{time.Now().AddDate(0, 0, -2), "Withdrawal", "ETH", 10.0, 0.001, "Completed", "0xabcd...efgh"},
		{time.Now().AddDate(0, 0, -3), "Trade", "USDT", 5000.0, 5.0, "Completed", "0x9876...5432"},
		{time.Now().AddDate(0, 0, -4), "Deposit", "ETH", 5.0, 0.0005, "Pending", "0xdef0...1234"},
		{time.Now().AddDate(0, 0, -5), "Trade", "BTC", 0.5, 0.0001, "Completed", "0x5678...90ab"},
	}
	
	for rowIdx, tx := range transactions {
		row := rowIdx + 5
		for colIdx, value := range tx {
			col, _ := excelize.ColumnNumberToName(colIdx + 1)
			f.SetCellValue(sheetName, col+fmt.Sprintf("%d", row), value)
			
			// Apply styles based on column
			if colIdx == 3 { // Amount
				f.SetCellStyle(sheetName, col+fmt.Sprintf("%d", row), col+fmt.Sprintf("%d", row), numberStyle)
			}
			if colIdx == 4 { // Fee
				f.SetCellStyle(sheetName, col+fmt.Sprintf("%d", row), col+fmt.Sprintf("%d", row), numberStyle)
			}
			if colIdx == 0 { // Date
				f.SetCellStyle(sheetName, col+fmt.Sprintf("%d", row), col+fmt.Sprintf("%d", row), dateStyle)
			}
		}
	}
	
	// Auto-fit columns
	f.SetColWidth(sheetName, "A", "A", 18)
	f.SetColWidth(sheetName, "B", "B", 12)
	f.SetColWidth(sheetName, "C", "C", 10)
	f.SetColWidth(sheetName, "D", "D", 15)
	f.SetColWidth(sheetName, "E", "E", 10)
	f.SetColWidth(sheetName, "F", "F", 12)
	f.SetColWidth(sheetName, "G", "G", 20)
	
	// Summary sheet
	summarySheet := "Summary"
	f.NewSheet(summarySheet)
	
	f.SetCellValue(summarySheet, "A1", "Transaction Summary")
	f.SetCellStyle(summarySheet, "A1", "A1", headerStyle)
	
	f.SetCellValue(summarySheet, "A3", "Total Transactions:")
	f.SetCellValue(summarySheet, "B3", len(transactions))
	
	f.SetCellValue(summarySheet, "A4", "Total Deposits:")
	f.SetCellValue(summarySheet, "B4", 2)
	
	f.SetCellValue(summarySheet, "A5", "Total Withdrawals:")
	f.SetCellValue(summarySheet, "B5", 1)
	
	f.SetCellValue(summarySheet, "A6", "Total Trades:")
	f.SetCellValue(summarySheet, "B6", 2)
	
	// Save file
	filename := fmt.Sprintf("%s/%s.xlsx", s.config.StoragePath, report.ReportID)
	if err := f.SaveAs(filename); err != nil {
		s.db.Model(report).Update("status", "failed")
		return
	}
	
	// Update report
	info, _ := os.Stat(filename)
	now := time.Now()
	s.db.Model(report).Updates(map[string]interface{}{
		"status":        "completed",
		"file_path":     filename,
		"file_size":     info.Size(),
		"completed_at": now,
	})
}

// CreateTaxReport - Create tax report in Excel
func (s *ExcelReportService) CreateTaxReport(year, userID string, params map[string]interface{}) (*ExcelReport, error) {
	reportID := s.GenerateReportID()
	
	report := &ExcelReport{
		ReportID:   reportID,
		ReportType: "tax",
		Title:      fmt.Sprintf("Tax Report %s", year),
		UserID:      userID,
		Status:      "generating",
		CreatedAt:  time.Now(),
	}
	
	s.db.Create(report)
	
	go s.generateTaxExcel(report, year, params)
	
	return report, nil
}

func (s *ExcelReportService) generateTaxExcel(report *ExcelReport, year string, params map[string]interface{}) {
	f := excelize.NewFile()
	defer f.Close()
	
	sheetName := "Tax Report"
	
	headerStyle, _ := f.NewStyle(`{"font":{"bold":true,"color":"#FFFFFF"},"fill":{"type":"pattern","color":["#10B981"],"pattern":1}}`)
	
	// Title
	f.SetCellValue(sheetName, "A1", fmt.Sprintf("Tax Report - Year %s", year))
	f.SetCellStyle(sheetName, "A1", "A1", headerStyle)
	f.MergeCell(sheetName, "A1", "F1")
	
	// Holdings section
	f.SetCellValue(sheetName, "A3", "Current Holdings")
	f.SetCellValue(sheetName, "A4", "Asset")
	f.SetCellValue(sheetName, "B4", "Quantity")
	f.SetCellValue(sheetName, "C4", "Cost Basis")
	f.SetCellValue(sheetName, "D4", "Current Value")
	f.SetCellValue(sheetName, "E4", "Gain/Loss")
	f.SetCellValue(sheetName, "F4", "Holding Period")
	
	for _, col := range []string{"A4", "B4", "C4", "D4", "E4", "F4"} {
		f.SetCellStyle(sheetName, col, col, headerStyle)
	}
	
	holdings := [][]interface{}{
		{"BTC", 1.5, 45000.00, 67500.00, 22500.00, "Long-term"},
		{"ETH", 10.0, 15000.00, 20000.00, 5000.00, "Long-term"},
		{"USDT", 5000.0, 5000.00, 5000.00, 0.00, "N/A"},
	}
	
	for rowIdx, h := range holdings {
		row := rowIdx + 5
		for colIdx, value := range h {
			col, _ := excelize.ColumnNumberToName(colIdx + 1)
			f.SetCellValue(sheetName, col+fmt.Sprintf("%d", row), value)
		}
	}
	
	// Transactions section
	f.SetCellValue(sheetName, "A11", "Taxable Transactions")
	f.SetCellValue(sheetName, "A12", "Date")
	f.SetCellValue(sheetName, "B12", "Asset")
	f.SetCellValue(sheetName, "C12", "Type")
	f.SetCellValue(sheetName, "D12", "Proceeds")
	f.SetCellValue(sheetName, "E12", "Cost Basis")
	f.SetCellValue(sheetName, "F12", "Gain/Loss")
	
	for _, col := range []string{"A12", "B12", "C12", "D12", "E12", "F12"} {
		f.SetCellStyle(sheetName, col, col, headerStyle)
	}
	
	txns := [][]interface{}{
		{time.Now().AddDate(0, -6, 0), "BTC", "Sale", 30000.00, 20000.00, 10000.00},
		{time.Now().AddDate(0, -3, 0), "ETH", "Sale", 8000.00, 5000.00, 3000.00},
	}
	
	for rowIdx, tx := range txns {
		row := rowIdx + 13
		for colIdx, value := range tx {
			col, _ := excelize.ColumnNumberToName(colIdx + 1)
			f.SetCellValue(sheetName, col+fmt.Sprintf("%d", row), value)
		}
	}
	
	// Summary
	f.SetCellValue(sheetName, "A17", "Summary")
	f.SetCellValue(sheetName, "A18", "Total Proceeds:")
	f.SetCellValue(sheetName, "B18", 38000.00)
	
	f.SetCellValue(sheetName, "A19", "Total Cost Basis:")
	f.SetCellValue(sheetName, "B19", 25000.00)
	
	f.SetCellValue(sheetName, "A20", "Total Capital Gains:")
	f.SetCellValue(sheetName, "B20", 13000.00)
	
	filename := fmt.Sprintf("%s/%s.xlsx", s.config.StoragePath, report.ReportID)
	if err := f.SaveAs(filename); err != nil {
		s.db.Model(report).Update("status", "failed")
		return
	}
	
	info, _ := os.Stat(filename)
	now := time.Now()
	s.db.Model(report).Updates(map[string]interface{}{
		"status":        "completed",
		"file_path":     filename,
		"file_size":     info.Size(),
		"completed_at": now,
	})
}

// CreateAnalyticsReport - Create analytics report in Excel
func (s *ExcelReportService) CreateAnalyticsReport(userID string, params map[string]interface{}) (*ExcelReport, error) {
	reportID := s.GenerateReportID()
	
	report := &ExcelReport{
		ReportID:   reportID,
		ReportType: "analytics",
		Title:      "Analytics Report",
		UserID:      userID,
		Status:      "generating",
		CreatedAt:  time.Now(),
	}
	
	s.db.Create(report)
	
	go s.generateAnalyticsExcel(report, params)
	
	return report, nil
}

func (s *ExcelReportService) generateAnalyticsExcel(report *ExcelReport, params map[string]interface{}) {
	f := excelize.NewFile()
	defer f.Close()
	
	sheetName := "Analytics"
	
	headerStyle, _ := f.NewStyle(`{"font":{"bold":true,"color":"#FFFFFF"},"fill":{"type":"pattern","color":["#F59E0B"],"pattern":1}}`)
	
	// Title
	f.SetCellValue(sheetName, "A1", "Analytics Report")
	f.SetCellStyle(sheetName, "A1", "A1", headerStyle)
	f.MergeCell(sheetName, "A1", "D1")
	
	// Key Metrics
	f.SetCellValue(sheetName, "A3", "Key Metrics")
	f.SetCellValue(sheetName, "A4", "Metric")
	f.SetCellValue(sheetName, "B4", "Value")
	f.SetCellValue(sheetName, "C4", "Change")
	f.SetCellValue(sheetName, "D4", "Period")
	
	for _, col := range []string{"A4", "B4", "C4", "D4"} {
		f.SetCellStyle(sheetName, col, col, headerStyle)
	}
	
	metrics := [][]interface{}{
		{"Total Users", 12500, "+15%", "Last 30 days"},
		{"Active Users", 8200, "+8%", "Last 30 days"},
		{"Total Volume", "$15.2M", "+22%", "Last 30 days"},
		{"Revenue", "$450K", "+12%", "Last 30 days"},
		{"Transaction Count", 45000, "+18%", "Last 30 days"},
	}
	
	for rowIdx, m := range metrics {
		row := rowIdx + 5
		for colIdx, value := range m {
			col, _ := excelize.ColumnNumberToName(colIdx + 1)
			f.SetCellValue(sheetName, col+fmt.Sprintf("%d", row), value)
		}
	}
	
	// Top Assets
	f.SetCellValue(sheetName, "A12", "Top Assets by Volume")
	f.SetCellValue(sheetName, "A13", "Asset")
	f.SetCellValue(sheetName, "B13", "Volume")
	f.SetCellValue(sheetName, "C13", "Share")
	f.SetCellValue(sheetName, "D13", "Transactions")
	
	for _, col := range []string{"A13", "B13", "C13", "D13"} {
		f.SetCellStyle(sheetName, col, col, headerStyle)
	}
	
	assets := [][]interface{}{
		{"BTC", "$8.5M", "56%", 15000},
		{"ETH", "$4.2M", "28%", 12000},
		{"USDT", "$1.8M", "12%", 8000},
		{"BNB", "$0.7M", "4%", 1000},
	}
	
	for rowIdx, a := range assets {
		row := rowIdx + 14
		for colIdx, value := range a {
			col, _ := excelize.ColumnNumberToName(colIdx + 1)
			f.SetCellValue(sheetName, col+fmt.Sprintf("%d", row), value)
		}
	}
	
	filename := fmt.Sprintf("%s/%s.xlsx", s.config.StoragePath, report.ReportID)
	if err := f.SaveAs(filename); err != nil {
		s.db.Model(report).Update("status", "failed")
		return
	}
	
	info, _ := os.Stat(filename)
	now := time.Now()
	s.db.Model(report).Updates(map[string]interface{}{
		"status":        "completed",
		"file_path":     filename,
		"file_size":     info.Size(),
		"completed_at": now,
	})
}

// GetReport - Get report by ID
func (s *ExcelReportService) GetReport(reportID string) (*ExcelReport, error) {
	var report ExcelReport
	err := s.db.Where("report_id = ?", reportID).First(&report).Error
	return &report, err
}

// GetReportFile - Get report file as base64
func (s *ExcelReportService) GetReportFile(reportID string) (string, error) {
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
	Title      string                 `json:"title"`
	UserID     string                 `json:"user_id" binding:"required"`
	Parameters map[string]interface{} `json:"parameters"`
}

func (s *ExcelReportService) CreateReportHandler(c *gin.Context) {
	var req CreateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	title := req.Title
	if title == "" {
		title = fmt.Sprintf("%s Report", req.ReportType)
	}
	
	var report *ExcelReport
	var err error
	
	switch req.ReportType {
	case "transaction":
		report, err = s.CreateTransactionReport(req.ReportType, title, req.UserID, req.Parameters)
	case "tax":
		year := "2024"
		if y, ok := req.Parameters["year"].(string); ok {
			year = y
		}
		report, err = s.CreateTaxReport(year, req.UserID, req.Parameters)
	case "analytics":
		report, err = s.CreateAnalyticsReport(req.UserID, req.Parameters)
	default:
		c.JSON(400, gin.H{"error": "invalid report_type"})
		return
	}
	
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(202, report)
}

func (s *ExcelReportService) GetReportHandler(c *gin.Context) {
	reportID := c.Param("report_id")
	
	report, err := s.GetReport(reportID)
	if err != nil {
		c.JSON(404, gin.H{"error": "report not found"})
		return
	}
	
	c.JSON(200, report)
}

func (s *ExcelReportService) DownloadReportHandler(c *gin.Context) {
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
	
	filename := fmt.Sprintf("%s.xlsx", report.ReportID)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

// Utility

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
		StoragePath: getEnv("EXCEL_STORAGE_PATH", "./reports/excel"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "excel_reports_db"),
		RedisHost:  getEnv("REDIS_HOST", "localhost"),
		RedisPort:  getEnv("REDIS_PORT", "6379"),
		ServerPort: getEnv("EXCEL_SERVER_PORT", "8097"),
	}
	
	service, err := NewExcelReportService(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize Excel report service: %v", err)
	}
	
	r := gin.Default()
	
	r.POST("/reports", service.CreateReportHandler)
	r.GET("/reports/:report_id", service.GetReportHandler)
	r.GET("/reports/:report_id/download", service.DownloadReportHandler)
	
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "excel-reports"})
	})
	
	log.Printf("Excel Report Service starting on port %s", cfg.ServerPort)
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
