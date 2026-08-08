package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/tigerwallet/admin/internal/models"
	"github.com/tigerwallet/admin/pkg/database"

	"github.com/gin-gonic/gin"
)

// InstitutionalHandler handles institutional client requests - COMPLETE IMPLEMENTATION
type InstitutionalHandler struct {
	db *database.PostgresDB
}

// NewInstitutionalHandler creates a new institutional handler
func NewInstitutionalHandler(db *database.PostgresDB) *InstitutionalHandler {
	return &InstitutionalHandler{db: db}
}

// ListClients lists all institutional clients
func (h *InstitutionalHandler) ListClients(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	clientType := c.Query("type")
	status := c.Query("status")
	search := c.Query("search")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var clients []models.InstitutionalClient
	var total int64

	query := h.db.Model(&models.InstitutionalClient{})

	if clientType != "" {
		query = query.Where("client_type = ?", clientType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if search != "" {
		query = query.Where("name ILIKE ? OR email ILIKE ? OR company ILIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC")

	if err := query.Find(&clients).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch clients"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        clients,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

// GetClient gets an institutional client by ID
func (h *InstitutionalHandler) GetClient(c *gin.Context) {
	clientID := c.Param("id")

	var client models.InstitutionalClient
	if err := h.db.Preload("AccountManager").First(&client, clientID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		return
	}

	c.JSON(http.StatusOK, client)
}

// CreateClient creates a new institutional client
func (h *InstitutionalHandler) CreateClient(c *gin.Context) {
	adminID := c.GetUint("admin_id")

	var req struct {
		Name           string   `json:"name" binding:"required"`
		Email          string   `json:"email" binding:"required,email"`
		Phone          string   `json:"phone"`
		Company        string   `json:"company" binding:"required"`
		ClientType     string   `json:"client_type" binding:"required"`
		Address        string   `json:"address"`
		RegistrationNo string   `json:"registration_no"`
		TaxID          string   `json:"tax_id"`
		DailyLimit     float64  `json:"daily_limit"`
		MonthlyLimit   float64  `json:"monthly_limit"`
		YearlyLimit    float64  `json:"yearly_limit"`
		MinTradeAmount float64  `json:"min_trade_amount"`
		MaxTradeAmount float64  `json:"max_trade_amount"`
		AllowedChains  []string `json:"allowed_chains"`
		AllowedTokens  []string `json:"allowed_tokens"`
		RequiresKYC    bool     `json:"requires_kyc"`
		HasAPIAccess   bool     `json:"has_api_access"`
		IsActive       bool     `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Check if email already exists
	var existingClient models.InstitutionalClient
	if err := h.db.Where("email = ?", req.Email).First(&existingClient).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
		return
	}

	client := models.InstitutionalClient{
		Name:           req.Name,
		Email:          req.Email,
		Phone:          req.Phone,
		Company:        req.Company,
		ClientType:     req.ClientType,
		Address:        req.Address,
		RegistrationNo: req.RegistrationNo,
		TaxID:          req.TaxID,
		DailyLimit:     req.DailyLimit,
		MonthlyLimit:   req.MonthlyLimit,
		YearlyLimit:    req.YearlyLimit,
		MinTradeAmount: req.MinTradeAmount,
		MaxTradeAmount: req.MaxTradeAmount,
		AllowedChains:  req.AllowedChains,
		AllowedTokens:  req.AllowedTokens,
		RequiresKYC:    req.RequiresKYC,
		HasAPIAccess:   req.HasAPIAccess,
		IsActive:       req.IsActive,
		Status:         "pending",
		CreatedBy:      adminID,
	}

	if err := h.db.Create(&client).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create client"})
		return
	}

	logAdminActivity(h.db, adminID, "create_institutional_client", "client",
		strconv.FormatUint(uint64(client.ID), 10), "Created client: "+client.Name, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusCreated, client)
}

// UpdateClient updates an institutional client
func (h *InstitutionalHandler) UpdateClient(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	clientID := c.Param("id")

	var req struct {
		Name           string   `json:"name"`
		Phone          string   `json:"phone"`
		Company        string   `json:"company"`
		Address        string   `json:"address"`
		RegistrationNo string   `json:"registration_no"`
		TaxID          string   `json:"tax_id"`
		DailyLimit     float64  `json:"daily_limit"`
		MonthlyLimit   float64  `json:"monthly_limit"`
		YearlyLimit    float64  `json:"yearly_limit"`
		MinTradeAmount float64  `json:"min_trade_amount"`
		MaxTradeAmount float64  `json:"max_trade_amount"`
		AllowedChains  []string `json:"allowed_chains"`
		AllowedTokens  []string `json:"allowed_tokens"`
		RequiresKYC    *bool    `json:"requires_kyc"`
		HasAPIAccess   *bool    `json:"has_api_access"`
		IsActive       *bool    `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var client models.InstitutionalClient
	if err := h.db.First(&client, clientID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		return
	}

	updates := map[string]interface{}{}

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Phone != "" {
		updates["phone"] = req.Phone
	}
	if req.Company != "" {
		updates["company"] = req.Company
	}
	if req.Address != "" {
		updates["address"] = req.Address
	}
	if req.RegistrationNo != "" {
		updates["registration_no"] = req.RegistrationNo
	}
	if req.TaxID != "" {
		updates["tax_id"] = req.TaxID
	}
	if req.DailyLimit > 0 {
		updates["daily_limit"] = req.DailyLimit
	}
	if req.MonthlyLimit > 0 {
		updates["monthly_limit"] = req.MonthlyLimit
	}
	if req.YearlyLimit > 0 {
		updates["yearly_limit"] = req.YearlyLimit
	}
	if req.MinTradeAmount > 0 {
		updates["min_trade_amount"] = req.MinTradeAmount
	}
	if req.MaxTradeAmount > 0 {
		updates["max_trade_amount"] = req.MaxTradeAmount
	}
	if req.AllowedChains != nil {
		updates["allowed_chains"] = req.AllowedChains
	}
	if req.AllowedTokens != nil {
		updates["allowed_tokens"] = req.AllowedTokens
	}
	if req.RequiresKYC != nil {
		updates["requires_kyc"] = *req.RequiresKYC
	}
	if req.HasAPIAccess != nil {
		updates["has_api_access"] = *req.HasAPIAccess
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if err := h.db.Model(&client).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update client"})
		return
	}

	logAdminActivity(h.db, adminID, "update_institutional_client", "client", clientID,
		"Updated client: "+client.Name, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, client)
}

// DeleteClient deletes an institutional client
func (h *InstitutionalHandler) DeleteClient(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	clientID := c.Param("id")

	var client models.InstitutionalClient
	if err := h.db.First(&client, clientID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		return
	}

	if err := h.db.Delete(&client).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete client"})
		return
	}

	logAdminActivity(h.db, adminID, "delete_institutional_client", "client", clientID,
		"Deleted client: "+client.Name, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Client deleted successfully"})
}

// ApproveClient approves an institutional client
func (h *InstitutionalHandler) ApproveClient(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	clientID := c.Param("id")

	var client models.InstitutionalClient
	if err := h.db.First(&client, clientID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		return
	}

	if err := h.db.Model(&client).Updates(map[string]interface{}{
		"status":      "approved",
		"approved_at": time.Now(),
		"approved_by": adminID,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to approve client"})
		return
	}

	logAdminActivity(h.db, adminID, "approve_institutional_client", "client", clientID,
		"Approved client: "+client.Name, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Client approved successfully"})
}

// SuspendClient suspends an institutional client
func (h *InstitutionalHandler) SuspendClient(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	clientID := c.Param("id")

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Reason is required"})
		return
	}

	var client models.InstitutionalClient
	if err := h.db.First(&client, clientID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		return
	}

	if err := h.db.Model(&client).Updates(map[string]interface{}{
		"status":         "suspended",
		"suspended_at":   time.Now(),
		"suspended_by":   adminID,
		"suspend_reason": req.Reason,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to suspend client"})
		return
	}

	logAdminActivity(h.db, adminID, "suspend_institutional_client", "client", clientID,
		"Suspended client: "+client.Name+" - Reason: "+req.Reason, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Client suspended successfully"})
}

// SetClientLimits sets client limits
func (h *InstitutionalHandler) SetClientLimits(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	clientID := c.Param("id")

	var req struct {
		DailyLimit     float64 `json:"daily_limit"`
		MonthlyLimit   float64 `json:"monthly_limit"`
		YearlyLimit    float64 `json:"yearly_limit"`
		MinTradeAmount float64 `json:"min_trade_amount"`
		MaxTradeAmount float64 `json:"max_trade_amount"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var client models.InstitutionalClient
	if err := h.db.First(&client, clientID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		return
	}

	updates := map[string]interface{}{}
	if req.DailyLimit > 0 {
		updates["daily_limit"] = req.DailyLimit
	}
	if req.MonthlyLimit > 0 {
		updates["monthly_limit"] = req.MonthlyLimit
	}
	if req.YearlyLimit > 0 {
		updates["yearly_limit"] = req.YearlyLimit
	}
	if req.MinTradeAmount > 0 {
		updates["min_trade_amount"] = req.MinTradeAmount
	}
	if req.MaxTradeAmount > 0 {
		updates["max_trade_amount"] = req.MaxTradeAmount
	}

	if err := h.db.Model(&client).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update limits"})
		return
	}

	logAdminActivity(h.db, adminID, "set_client_limits", "client", clientID,
		"Updated limits for: "+client.Name, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, client)
}

// AssignAccountManager assigns an account manager to client
func (h *InstitutionalHandler) AssignAccountManager(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	clientID := c.Param("id")

	var req struct {
		AdminID uint `json:"admin_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var client models.InstitutionalClient
	if err := h.db.First(&client, clientID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		return
	}

	// Verify admin exists
	var admin models.Admin
	if err := h.db.First(&admin, req.AdminID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
		return
	}

	if err := h.db.Model(&client).Update("account_manager_id", req.AdminID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign account manager"})
		return
	}

	logAdminActivity(h.db, adminID, "assign_account_manager", "client", clientID,
		"Assigned account manager: "+admin.Username, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Account manager assigned successfully"})
}

// GetClientStats gets institutional client statistics
func (h *InstitutionalHandler) GetClientStats(c *gin.Context) {
	var stats struct {
		TotalClients     int64   `json:"total_clients"`
		ActiveClients    int64   `json:"active_clients"`
		PendingClients   int64   `json:"pending_clients"`
		SuspendedClients int64   `json:"suspended_clients"`
		TotalVolume      float64 `json:"total_volume"`
	}

	h.db.Model(&models.InstitutionalClient{}).Count(&stats.TotalClients)
	h.db.Model(&models.InstitutionalClient{}).Where("status = ? AND is_active = ?", "approved", true).Count(&stats.ActiveClients)
	h.db.Model(&models.InstitutionalClient{}).Where("status = ?", "pending").Count(&stats.PendingClients)
	h.db.Model(&models.InstitutionalClient{}).Where("status = ?", "suspended").Count(&stats.SuspendedClients)

	h.db.Model(&models.InstitutionalClient{}).Select("COALESCE(SUM(total_volume), 0)").Scan(&stats.TotalVolume)

	c.JSON(http.StatusOK, stats)
}
