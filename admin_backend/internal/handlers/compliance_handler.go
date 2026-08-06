package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"admin_backend/internal/models"
	"admin_backend/pkg/database"

	"github.com/gin-gonic/gin"
)

// ComplianceHandler handles compliance operations
type ComplianceHandler struct {
	db *database.PostgresDB
}

// NewComplianceHandler creates a new compliance handler
func NewComplianceHandler(db *database.PostgresDB) *ComplianceHandler {
	return &ComplianceHandler{db: db}
}

// AMLReportRequest represents AML report request
type AMLReportRequest struct {
	PeriodStart string `json:"period_start" binding:"required"`
	PeriodEnd   string `json:"period_end" binding:"required"`
	ReportType  string `json:"report_type"` // transaction, user, summary
	Format      string `json:"format"` // json, pdf, excel
}

// TaxReportRequest represents tax report request
type TaxReportRequest struct {
	UserID uint   `json:"user_id" binding:"required"`
	Year   int    `json:"year" binding:"required"`
	Format string `json:"format"` // json, pdf, excel
}

// GDPRRequest represents GDPR data request
type GDPRRequest struct {
	UserID      uint   `json:"user_id" binding:"required"`
	RequestType string `json:"request_type" binding:"required"` // access, delete, portability
	Reason      string `json:"reason"`
}

// GenerateAMLReport generates an AML report
// POST /api/v1/admin/compliance/aml
func (h *ComplianceHandler) GenerateAMLReport(c *gin.Context) {
	var req AMLReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Parse dates
	periodStart, err := time.Parse("2006-01-02", req.PeriodStart)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start date format"})
		return
	}

	periodEnd, err := time.Parse("2006-01-02", req.PeriodEnd)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end date format"})
		return
	}

	// Get transaction data for the period
	var transactions []models.Transaction
	err = h.db.Where("created_at BETWEEN ? AND ?", periodStart, periodEnd).Find(&transactions).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions"})
		return
	}

	// Calculate AML metrics
	highRiskCount := 0
	flaggedCount := 0
	totalAmount := 0.0

	for _, tx := range transactions {
		totalAmount += tx.Amount

		// Check for high-risk indicators
		riskScore := 0
		if tx.Amount > 10000 {
			riskScore += 30
		}
		if tx.Fee > 100 {
			riskScore += 20
		}

		if riskScore > 50 {
			flaggedCount++
		}

		// Check user risk score
		var user models.User
		if h.db.First(&user, tx.UserID).Error == nil && user.RiskScore > 70 {
			highRiskCount++
		}
	}

	// Generate report
	report := map[string]interface{}{
		"report_id":        fmt.Sprintf("AML-%d", time.Now().Unix()),
		"generated_at":     time.Now().Format(time.RFC3339),
		"period_start":    periodStart.Format(time.RFC3339),
		"period_end":      periodEnd.Format(time.RFC3339),
		"total_transactions": len(transactions),
		"total_amount":    totalAmount,
		"high_risk_users": highRiskCount,
		"flagged_transactions": flaggedCount,
		"flagged_rate":    float64(flaggedCount) / float64(len(transactions)+1),
		"transactions":     transactions,
	}

	// Save to database
	complianceReport := models.ComplianceReport{
		Type:        "aml",
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		Status:      "completed",
		GeneratedBy: c.GetUint("admin_id"),
	}

	if req.Format == "pdf" || req.Format == "excel" {
		complianceReport.Status = "generating"
	}

	h.db.Create(&complianceReport)

	adminID := c.GetUint("admin_id")
	logAdminActivity(h.db, adminID, "generate_aml_report", "compliance", 
		fmt.Sprintf("%d", complianceReport.ID), "Generated AML report", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, report)
}

// GenerateTaxReport generates a tax report
// POST /api/v1/admin/compliance/tax
func (h *ComplianceHandler) GenerateTaxReport(c *gin.Context) {
	var req TaxReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Get user
	var user models.User
	if err := h.db.First(&user, req.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Calculate date range
	startDate := time.Date(req.Year, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(req.Year, 12, 31, 23, 59, 59, 0, time.UTC)

	// Get transactions for the year
	var transactions []models.Transaction
	err := h.db.Where("user_id = ? AND created_at BETWEEN ? AND ?", 
		req.UserID, startDate, endDate).Find(&transactions).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions"})
		return
	}

	// Calculate tax summary
	totalBuys := 0.0
	totalSells := 0.0
	totalFees := 0.0

	for _, tx := range transactions {
		switch tx.Type {
		case "buy":
			totalBuys += tx.Amount
		case "sell":
			totalSells += tx.Amount
		}
		totalFees += tx.Fee
	}

	netGainLoss := totalSells - totalBuys - totalFees

	report := map[string]interface{}{
		"report_id":      fmt.Sprintf("TAX-%d-%d", req.UserID, req.Year),
		"generated_at":   time.Now().Format(time.RFC3339),
		"user_id":        req.UserID,
		"user_email":     user.Email,
		"year":           req.Year,
		"total_buys":     totalBuys,
		"total_sells":    totalSells,
		"total_fees":     totalFees,
		"net_gain_loss":  netGainLoss,
		"transactions":   transactions,
	}

	// Save to database
	complianceReport := models.ComplianceReport{
		Type:        "tax",
		PeriodStart: startDate,
		PeriodEnd:   endDate,
		Status:      "completed",
		GeneratedBy: c.GetUint("admin_id"),
	}

	h.db.Create(&complianceReport)

	adminID := c.GetUint("admin_id")
	logAdminActivity(h.db, adminID, "generate_tax_report", "compliance", 
		fmt.Sprintf("%d", complianceReport.ID), "Generated tax report for user: "+user.Email, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, report)
}

// ProcessGDPRRequest processes a GDPR request
// POST /api/v1/admin/compliance/gdpr
func (h *ComplianceHandler) ProcessGDPRRequest(c *gin.Context) {
	var req GDPRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Get user
	var user models.User
	if err := h.db.First(&user, req.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Create GDPR request record
	gdprRequest := models.GDPRRequest{
		UserID:       req.UserID,
		Email:        user.Email,
		RequestType:  req.RequestType,
		Status:       "pending",
		RequestedAt:  time.Now(),
		Reason:       req.Reason,
	}

	if err := h.db.Create(&gdprRequest).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create GDPR request"})
		return
	}

	// Process based on type
	switch req.RequestType {
	case "access", "portability":
		// Generate data export
		var userData map[string]interface{}
		h.db.First(&user, req.UserID)

		userData = map[string]interface{}{
			"user":         user,
			"exported_at": time.Now().Format(time.RFC3339),
		}

		// Return the data
		response := map[string]interface{}{
			"request_id":  gdprRequest.ID,
			"status":      "completed",
			"request_type": req.RequestType,
			"data":        userData,
		}

		// Update status
		now := time.Now()
		h.db.Model(&gdprRequest).Updates(map[string]interface{}{
			"status":       "completed",
			"completed_at": now,
		})

		c.JSON(http.StatusOK, response)

	case "delete":
		// Anonymize user data
		h.db.Model(&user).Updates(map[string]interface{}{
			"email":    fmt.Sprintf("deleted-%d@example.com", req.UserID),
			"username": fmt.Sprintf("deleted-%d", req.UserID),
			"status":   "deleted",
		})

		// Update request
		now := time.Now()
		h.db.Model(&gdprRequest).Updates(map[string]interface{}{
			"status":       "completed",
			"completed_at": now,
		})

		c.JSON(http.StatusOK, map[string]interface{}{
			"request_id": gdprRequest.ID,
			"status":     "completed",
			"message":    "User data has been anonymized",
		})

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request type"})
	}
}

// GetComplianceReports lists compliance reports
// GET /api/v1/admin/compliance/reports
func (h *ComplianceHandler) GetComplianceReports(c *gin.Context) {
	reportType := c.Query("type")
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var reports []models.ComplianceReport
	var total int64

	query := h.db.Model(&models.ComplianceReport{})
	if reportType != "" {
		query = query.Where("type = ?", reportType)
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

// GetComplianceStats gets compliance statistics
// GET /api/v1/admin/compliance/stats
func (h *ComplianceHandler) GetComplianceStats(c *gin.Context) {
	var stats struct {
		TotalReports   int64 `json:"total_reports"`
		AMLReports     int64 `json:"aml_reports"`
		TaxReports     int64 `json:"tax_reports"`
		GDPRRequests   int64 `json:"gdpr_requests"`
		PendingGDPR   int64 `json:"pending_gdpr"`
		CompletedGDPR int64 `json:"completed_gdpr"`
		HighRiskUsers int64 `json:"high_risk_users"`
	}

	h.db.Model(&models.ComplianceReport{}).Count(&stats.TotalReports)
	h.db.Model(&models.ComplianceReport{}).Where("type = ?", "aml").Count(&stats.AMLReports)
	h.db.Model(&models.ComplianceReport{}).Where("type = ?", "tax").Count(&stats.TaxReports)
	h.db.Model(&models.GDPRRequest{}).Count(&stats.GDPRRequests)
	h.db.Model(&models.GDPRRequest{}).Where("status = ?", "pending").Count(&stats.PendingGDPR)
	h.db.Model(&models.GDPRRequest{}).Where("status = ?", "completed").Count(&stats.CompletedGDPR)
	h.db.Model(&models.User{}).Where("risk_score > ?", 70).Count(&stats.HighRiskUsers)

	c.JSON(http.StatusOK, stats)
}

// ExportGDPRData exports all GDPR data for a user
// GET /api/v1/admin/compliance/gdpr/:user_id/export
func (h *ComplianceHandler) ExportGDPRData(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get all user data
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var transactions []models.Transaction
	h.db.Where("user_id = ?", userID).Find(&transactions)

	var kycRecords []models.KYCRecord
	h.db.Where("user_id = ?", userID).Find(&kycRecords)

	exportData := map[string]interface{}{
		"user":         user,
		"transactions": transactions,
		"kyc_records":  kycRecords,
		"exported_at": time.Now().Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, exportData)
}

// AnonymizeUserData anonymizes user data (for GDPR delete)
// DELETE /api/v1/admin/compliance/anonymize/:user_id
func (h *ComplianceHandler) AnonymizeUserData(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Anonymize user data
	updates := map[string]interface{}{
		"email":      fmt.Sprintf("deleted-%d@example.com", userID),
		"username":   fmt.Sprintf("deleted-%d", userID),
		"phone":      nil,
		"first_name": "Deleted",
		"last_name":  "User",
		"status":     "deleted",
	}

	if err := h.db.Model(&user).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to anonymize user data"})
		return
	}

	adminID := c.GetUint("admin_id")
	logAdminActivity(h.db, adminID, "anonymize_user", "compliance", 
		fmt.Sprintf("%d", userID), "Anonymized user data", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "User data has been anonymized"})
}
