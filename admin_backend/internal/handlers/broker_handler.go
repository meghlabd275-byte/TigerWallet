package handlers

import (
	"net/http"
	"strconv"
	"time"

	"admin_backend/internal/models"
	"admin_backend/pkg/database"

	"github.com/gin-gonic/gin"
)

// BrokerHandler handles broker-related requests - COMPLETE IMPLEMENTATION
type BrokerHandler struct {
	db *database.PostgresDB
}

// NewBrokerHandler creates a new broker handler
func NewBrokerHandler(db *database.PostgresDB) *BrokerHandler {
	return &BrokerHandler{db: db}
}

// ListBrokers lists all brokers
func (h *BrokerHandler) ListBrokers(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	status := c.Query("status")
	search := c.Query("search")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var brokers []models.Broker
	var total int64

	query := h.db.Model(&models.Broker{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if search != "" {
		query = query.Where("name ILIKE ? OR email ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC")

	if err := query.Find(&brokers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch brokers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        brokers,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

// GetBroker gets a broker by ID
func (h *BrokerHandler) GetBroker(c *gin.Context) {
	brokerID := c.Param("id")

	var broker models.Broker
	if err := h.db.Preload("Clients").First(&broker, brokerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Broker not found"})
		return
	}

	c.JSON(http.StatusOK, broker)
}

// CreateBroker creates a new broker
func (h *BrokerHandler) CreateBroker(c *gin.Context) {
	adminID := c.GetUint("admin_id")

	var req struct {
		Name            string  `json:"name" binding:"required"`
		Email           string  `json:"email" binding:"required,email"`
		Phone           string  `json:"phone"`
		Company         string  `json:"company"`
		Address         string  `json:"address"`
		Commission      float64 `json:"commission"`
		MinTradeAmount  float64 `json:"min_trade_amount"`
		MaxTradeAmount  float64 `json:"max_trade_amount"`
		AllowedChains   []string `json:"allowed_chains"`
		AllowedTokens   []string `json:"allowed_tokens"`
		KYCRequired     bool    `json:"kyc_required"`
		IsActive        bool    `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Check if email already exists
	var existingBroker models.Broker
	if err := h.db.Where("email = ?", req.Email).First(&existingBroker).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
		return
	}

	broker := models.Broker{
		Name:            req.Name,
		Email:           req.Email,
		Phone:           req.Phone,
		Company:         req.Company,
		Address:         req.Address,
		Commission:      req.Commission,
		MinTradeAmount:  req.MinTradeAmount,
		MaxTradeAmount:  req.MaxTradeAmount,
		AllowedChains:   req.AllowedChains,
		AllowedTokens:   req.AllowedTokens,
		KYCRequired:     req.KYCRequired,
		IsActive:        req.IsActive,
		Status:          "pending",
		CreatedBy:       adminID,
	}

	if err := h.db.Create(&broker).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create broker"})
		return
	}

	logAdminActivity(h.db, adminID, "create_broker", "broker", strconv.FormatUint(uint64(broker.ID)), "Created broker: "+broker.Name, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusCreated, broker)
}

// UpdateBroker updates a broker
func (h *BrokerHandler) UpdateBroker(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	brokerID := c.Param("id")

	var req struct {
		Name            string   `json:"name"`
		Phone           string   `json:"phone"`
		Company         string   `json:"company"`
		Address         string   `json:"address"`
		Commission      float64  `json:"commission"`
		MinTradeAmount  float64  `json:"min_trade_amount"`
		MaxTradeAmount  float64  `json:"max_trade_amount"`
		AllowedChains   []string `json:"allowed_chains"`
		AllowedTokens   []string `json:"allowed_tokens"`
		KYCRequired     *bool    `json:"kyc_required"`
		IsActive        *bool    `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var broker models.Broker
	if err := h.db.First(&broker, brokerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Broker not found"})
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
	if req.Commission > 0 {
		updates["commission"] = req.Commission
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
	if req.KYCRequired != nil {
		updates["kyc_required"] = *req.KYCRequired
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if err := h.db.Model(&broker).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update broker"})
		return
	}

	logAdminActivity(h.db, adminID, "update_broker", "broker", brokerID, "Updated broker: "+broker.Name, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, broker)
}

// DeleteBroker deletes a broker
func (h *BrokerHandler) DeleteBroker(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	brokerID := c.Param("id")

	var broker models.Broker
	if err := h.db.First(&broker, brokerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Broker not found"})
		return
	}

	if err := h.db.Delete(&broker).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete broker"})
		return
	}

	logAdminActivity(h.db, adminID, "delete_broker", "broker", brokerID, "Deleted broker: "+broker.Name, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Broker deleted successfully"})
}

// ApproveBroker approves a broker
func (h *BrokerHandler) ApproveBroker(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	brokerID := c.Param("id")

	var broker models.Broker
	if err := h.db.First(&broker, brokerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Broker not found"})
		return
	}

	if err := h.db.Model(&broker).Updates(map[string]interface{}{
		"status":       "approved",
		"approved_at":  time.Now(),
		"approved_by":  adminID,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to approve broker"})
		return
	}

	logAdminActivity(h.db, adminID, "approve_broker", "broker", brokerID, "Approved broker: "+broker.Name, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Broker approved successfully"})
}

// SuspendBroker suspends a broker
func (h *BrokerHandler) SuspendBroker(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	brokerID := c.Param("id")

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Reason is required"})
		return
	}

	var broker models.Broker
	if err := h.db.First(&broker, brokerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Broker not found"})
		return
	}

	if err := h.db.Model(&broker).Updates(map[string]interface{}{
		"status":         "suspended",
		"suspended_at":    time.Now(),
		"suspended_by":    adminID,
		"suspend_reason": req.Reason,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to suspend broker"})
		return
	}

	logAdminActivity(h.db, adminID, "suspend_broker", "broker", brokerID, "Suspended broker: "+broker.Name+" - Reason: "+req.Reason, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Broker suspended successfully"})
}

// SetBrokerCommission sets broker commission
func (h *BrokerHandler) SetBrokerCommission(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	brokerID := c.Param("id")

	var req struct {
		Commission float64 `json:"commission" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var broker models.Broker
	if err := h.db.First(&broker, brokerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Broker not found"})
		return
	}

	if err := h.db.Model(&broker).Update("commission", req.Commission).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update commission"})
		return
	}

	logAdminActivity(h.db, adminID, "set_broker_commission", "broker", brokerID, "Set commission to: "+strconv.FormatFloat(req.Commission, 'f', 2, 64), c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "Commission updated successfully"})
}

// GetBrokerClients gets broker's clients
func (h *BrokerHandler) GetBrokerClients(c *gin.Context) {
	brokerID := c.Param("id")
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var clients []models.BrokerClient
	var total int64

	query := h.db.Model(&models.BrokerClient{}).Where("broker_id = ?", brokerID)
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

// GetBrokerStats gets broker statistics
func (h *BrokerHandler) GetBrokerStats(c *gin.Context) {
	var stats struct {
		TotalBrokers   int64   `json:"total_brokers"`
		ActiveBrokers int64   `json:"active_brokers"`
		PendingBrokers int64   `json:"pending_brokers"`
		SuspendedBrokers int64 `json:"suspended_brokers"`
		TotalClients   int64   `json:"total_clients"`
		TotalRevenue  float64 `json:"total_revenue"`
	}

	h.db.Model(&models.Broker{}).Count(&stats.TotalBrokers)
	h.db.Model(&models.Broker{}).Where("status = ? AND is_active = ?", "approved", true).Count(&stats.ActiveBrokers)
	h.db.Model(&models.Broker{}).Where("status = ?", "pending").Count(&stats.PendingBrokers)
	h.db.Model(&models.Broker{}).Where("status = ?", "suspended").Count(&stats.SuspendedBrokers)
	h.db.Model(&models.BrokerClient{}).Count(&stats.TotalClients)

	h.db.Model(&models.Broker{}).Select("COALESCE(SUM(total_revenue), 0)").Scan(&stats.TotalRevenue)

	c.JSON(http.StatusOK, stats)
}
