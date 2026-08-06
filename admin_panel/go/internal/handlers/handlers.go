// Handlers - HTTP handlers for Admin Panel
package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tigerwallet/admin_panel/internal/middleware"
	"github.com/tigerwallet/admin_panel/internal/models"
	"github.com/tigerwallet/admin_panel/internal/services"
)

type Handler struct {
	authService     *services.AuthService
	userService     *services.UserService
	kycService      *services.KYCService
	transactionService *services.TransactionService
	withdrawalService *services.WithdrawalService
	tokenService    *services.TokenService
	blockchainService *services.BlockchainService
	feeService      *services.FeeService
	webhookService  *services.WebhookService
	notificationService *services.NotificationService
	auditService    *services.AuditService
	sessionService  *services.SessionService
	featureFlagService *services.FeatureFlagService
	ipWhitelistService *services.IPWhitelistService
	reportService   *services.ReportService
	ticketService   *services.TicketService
	whiteLabelService *services.WhiteLabelService
	slaService      *services.SLAService
	integrationService *services.IntegrationService
}

func NewHandler(
	authService *services.AuthService,
	userService *services.UserService,
	kycService *services.KYCService,
	transactionService *services.TransactionService,
	withdrawalService *services.WithdrawalService,
	tokenService *services.TokenService,
	blockchainService *services.BlockchainService,
	feeService *services.FeeService,
	webhookService *services.WebhookService,
	notificationService *services.NotificationService,
	auditService *services.AuditService,
	sessionService *services.SessionService,
	featureFlagService *services.FeatureFlagService,
	ipWhitelistService *services.IPWhitelistService,
	reportService *services.ReportService,
	ticketService *services.TicketService,
	whiteLabelService *services.WhiteLabelService,
	slaService *services.SLAService,
	integrationService *services.IntegrationService,
) *Handler {
	return &Handler{
		authService:          authService,
		userService:          userService,
		kycService:           kycService,
		transactionService:   transactionService,
		withdrawalService:    withdrawalService,
		tokenService:         tokenService,
		blockchainService:    blockchainService,
		feeService:           feeService,
		webhookService:       webhookService,
		notificationService:  notificationService,
		auditService:         auditService,
		sessionService:       sessionService,
		featureFlagService:   featureFlagService,
		ipWhitelistService:   ipWhitelistService,
		reportService:        reportService,
		ticketService:        ticketService,
		whiteLabelService:   whiteLabelService,
		slaService:         slaService,
		integrationService: integrationService,

	}
// HealthCheck returns health status
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"service": "admin_panel",
	})
}

// ==================== AUTHENTICATION ====================

// Register creates a new admin user
func (h *Handler) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=8"`
		Role     string `json:"role"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Role == "" {
		req.Role = "admin"
	}

	admin, err := h.authService.Register(c.Request.Context(), req.Username, req.Email, req.Password, req.Role)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"admin": admin})
}

// Login authenticates an admin user
func (h *Handler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	admin, accessToken, refreshToken, err := h.authService.Login(
		c.Request.Context(),
		req.Email,
		req.Password,
		c.ClientIP(),
		c.Request.UserAgent(),
	)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Log audit
	h.auditService.Log(c.Request.Context(), admin.ID, "login", "admin", admin.ID.String(), nil, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{
		"admin":         admin,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// RefreshToken refreshes JWT token
func (h *Handler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"access_token": token})
}

// Logout logs out an admin user
func (h *Handler) Logout(c *gin.Context) {
	adminID, _ := middleware.GetAdminID(c)
	h.authService.Logout(c.Request.Context(), adminID, "")
	h.auditService.Log(c.Request.Context(), adminID, "logout", "admin", adminID.String(), nil, c.ClientIP(), c.Request.UserAgent())
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// GetAdmins lists all admin users
func (h *Handler) GetAdmins(c *gin.Context) {
	admins, err := h.authService.ListAdmins(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"admins": admins})
}

// GetAdmin gets a single admin user
func (h *Handler) GetAdmin(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	admin, err := h.authService.GetAdminByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"admin": admin})
}

// UpdateAdmin updates an admin user
func (h *Handler) UpdateAdmin(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Role     string `json:"role"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.authService.UpdateAdmin(c.Request.Context(), id, req.Username, req.Email, req.Role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "admin updated"})
}

// DeleteAdmin deletes an admin user
func (h *Handler) DeleteAdmin(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.authService.DeleteAdmin(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "admin deleted"})
}

// SuspendAdmin suspends an admin user
func (h *Handler) SuspendAdmin(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.authService.SuspendAdmin(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "admin suspended"})
}

// ActivateAdmin activates an admin user
func (h *Handler) ActivateAdmin(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.authService.ActivateAdmin(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "admin activated"})
}

// ChangePassword changes admin password
func (h *Handler) ChangePassword(c *gin.Context) {
	adminID, _ := middleware.GetAdminID(c)

	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.authService.ChangePassword(c.Request.Context(), adminID, req.OldPassword, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.auditService.Log(c.Request.Context(), adminID, "change_password", "admin", adminID.String(), nil, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"message": "password changed"})
}

// Enable2FA enables 2FA for admin
func (h *Handler) Enable2FA(c *gin.Context) {
	adminID, _ := middleware.GetAdminID(c)

	var req struct {
		Secret string `json:"secret" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.authService.Enable2FA(c.Request.Context(), adminID, req.Secret); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "2FA enabled"})
}

// Disable2FA disables 2FA for admin
func (h *Handler) Disable2FA(c *gin.Context) {
	adminID, _ := middleware.GetAdminID(c)

	if err := h.authService.Disable2FA(c.Request.Context(), adminID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "2FA disabled"})
}

// ==================== USER MANAGEMENT ====================

// GetUsers lists all users
func (h *Handler) GetUsers(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	users, total, err := h.userService.ListUsers(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users, "total": total})
}

// GetUser gets a single user
func (h *Handler) GetUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	user, err := h.userService.GetUserByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// SearchUsers searches for users
func (h *Handler) SearchUsers(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query required"})
		return
	}

	users, err := h.userService.SearchUsers(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

// UpdateUserStatus updates user status
func (h *Handler) UpdateUserStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.userService.UpdateUserStatus(c.Request.Context(), id, req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "status updated"})
}

// BanUser bans a user
func (h *Handler) BanUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.userService.BanUser(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user banned"})
}

// UnbanUser unbans a user
func (h *Handler) UnbanUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.userService.UnbanUser(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user unbanned"})
}

// SuspendUser suspends a user
func (h *Handler) SuspendUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.userService.SuspendUser(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user suspended"})
}

// ==================== KYC MANAGEMENT ====================

// GetKYCRequests lists all KYC requests
func (h *Handler) GetKYCRequests(c *gin.Context) {
	status := c.Query("status")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	requests, total, err := h.kycService.ListKYCRequests(c.Request.Context(), status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"kyc_requests": requests, "total": total})
}

// ApproveKYC approves a KYC request
func (h *Handler) ApproveKYC(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	adminID, _ := middleware.GetAdminID(c)

	if err := h.kycService.ApproveKYC(c.Request.Context(), id, adminID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "KYC approved"})
}

// RejectKYC rejects a KYC request
func (h *Handler) RejectKYC(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminID, _ := middleware.GetAdminID(c)

	if err := h.kycService.RejectKYC(c.Request.Context(), id, adminID, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "KYC rejected"})
}

// ==================== TRANSACTION MANAGEMENT ====================

// GetTransactions lists all transactions
func (h *Handler) GetTransactions(c *gin.Context) {
	status := c.Query("status")
	userID := c.Query("user_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	transactions, total, err := h.transactionService.ListTransactions(c.Request.Context(), status, userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"transactions": transactions, "total": total})
}

// GetTransaction gets a single transaction
func (h *Handler) GetTransaction(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	tx, err := h.transactionService.GetTransactionByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"transaction": tx})
}

// FlagTransaction flags a transaction
func (h *Handler) FlagTransaction(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.transactionService.FlagTransaction(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "transaction flagged"})
}

// UnflagTransaction unflags a transaction
func (h *Handler) UnflagTransaction(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.transactionService.UnflagTransaction(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "transaction unflagged"})
}

// ==================== WITHDRAWAL MANAGEMENT ====================

// GetWithdrawals lists all withdrawals
func (h *Handler) GetWithdrawals(c *gin.Context) {
	status := c.Query("status")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	withdrawals, total, err := h.withdrawalService.ListWithdrawals(c.Request.Context(), status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"withdrawals": withdrawals, "total": total})
}

// ApproveWithdrawal approves a withdrawal
func (h *Handler) ApproveWithdrawal(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	adminID, _ := middleware.GetAdminID(c)

	if err := h.withdrawalService.ApproveWithdrawal(c.Request.Context(), id, adminID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "withdrawal approved"})
}

// RejectWithdrawal rejects a withdrawal
func (h *Handler) RejectWithdrawal(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.withdrawalService.RejectWithdrawal(c.Request.Context(), id, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "withdrawal rejected"})
}

// ProcessWithdrawal processes a withdrawal
func (h *Handler) ProcessWithdrawal(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		TXHash string `json:"tx_hash" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.withdrawalService.ProcessWithdrawal(c.Request.Context(), id, req.TXHash); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "withdrawal processed"})
}

// ==================== TOKEN MANAGEMENT ====================

// GetTokens lists all tokens
func (h *Handler) GetTokens(c *gin.Context) {
	tokens, err := h.tokenService.ListTokens(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tokens": tokens})
}

// CreateToken creates a new token
func (h *Handler) CreateToken(c *gin.Context) {
	var req models.Token
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.tokenService.CreateToken(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"token": token})
}

// UpdateToken updates a token
func (h *Handler) UpdateToken(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Name      string `json:"name"`
		IsActive  *bool  `json:"is_active"`
		IsVerified *bool `json:"is_verified"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.tokenService.UpdateToken(c.Request.Context(), id, req.Name, req.IsActive, req.IsVerified); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "token updated"})
}

// DeleteToken deletes a token
func (h *Handler) DeleteToken(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.tokenService.DeleteToken(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "token deleted"})
}

// ==================== BLOCKCHAIN MANAGEMENT ====================

// GetBlockchains lists all blockchains
func (h *Handler) GetBlockchains(c *gin.Context) {
	blockchains, err := h.blockchainService.ListBlockchains(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"blockchains": blockchains})
}

// CreateBlockchain creates a new blockchain
func (h *Handler) CreateBlockchain(c *gin.Context) {
	var req models.Blockchain
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	blockchain, err := h.blockchainService.CreateBlockchain(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"blockchain": blockchain})
}

// UpdateBlockchain updates a blockchain
func (h *Handler) UpdateBlockchain(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Name            string `json:"name"`
		RPCURL          string `json:"rpc_url"`
		ExplorerURL     string `json:"explorer_url"`
		AvgGasPriceGwei string `json:"avg_gas_price_gwei"`
		IsActive        *bool  `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.blockchainService.UpdateBlockchain(c.Request.Context(), id, req.Name, req.RPCURL, req.ExplorerURL, req.AvgGasPriceGwei, req.IsActive); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "blockchain updated"})
}

// SetBlockchainStatus sets blockchain status
func (h *Handler) SetBlockchainStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		IsActive bool `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.blockchainService.SetStatus(c.Request.Context(), id, req.IsActive); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "status updated"})
}

// ==================== FEE MANAGEMENT ====================

// GetFeeStructures lists all fee structures
func (h *Handler) GetFeeStructures(c *gin.Context) {
	fees, err := h.feeService.ListFeeStructures(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"fees": fees})
}

// CreateFeeStructure creates a new fee structure
func (h *Handler) CreateFeeStructure(c *gin.Context) {
	var req models.FeeStructure
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fee, err := h.feeService.CreateFeeStructure(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"fee": fee})
}

// UpdateFeeStructure updates a fee structure
func (h *Handler) UpdateFeeStructure(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		FeePercent string `json:"fee_percent"`
		FeeFixed   string `json:"fee_fixed"`
		MinFee     string `json:"min_fee"`
		MaxFee     string `json:"max_fee"`
		IsActive   *bool  `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.feeService.UpdateFeeStructure(c.Request.Context(), id, req.FeePercent, req.FeeFixed, req.MinFee, req.MaxFee, req.IsActive); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "fee updated"})
}

// ==================== TRADING PAIRS ====================

// GetTradingPairs lists all trading pairs
func (h *Handler) GetTradingPairs(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	pairs, total, err := h.tokenService.ListTradingPairs(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"trading_pairs": pairs, "total": total})
}

// CreateTradingPair creates a new trading pair
func (h *Handler) CreateTradingPair(c *gin.Context) {
	var req models.TradingPair
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pair, err := h.tokenService.CreateTradingPair(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"trading_pair": pair})
}

// UpdatePairStatus updates trading pair status
func (h *Handler) UpdatePairStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.tokenService.UpdatePairStatus(c.Request.Context(), id, req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "status updated"})
}

// ==================== WEBHOOKS ====================

// GetWebhooks lists all webhooks
func (h *Handler) GetWebhooks(c *gin.Context) {
	webhooks, err := h.webhookService.ListWebhooks(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"webhooks": webhooks})
}

// CreateWebhook creates a new webhook
func (h *Handler) CreateWebhook(c *gin.Context) {
	var req models.Webhook
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminID, _ := middleware.GetAdminID(c)
	webhook, err := h.webhookService.CreateWebhook(c.Request.Context(), &req, adminID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"webhook": webhook})
}

// TestWebhook tests a webhook
func (h *Handler) TestWebhook(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.webhookService.TestWebhook(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "webhook test sent"})
}

// DeleteWebhook deletes a webhook
func (h *Handler) DeleteWebhook(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.webhookService.DeleteWebhook(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "webhook deleted"})
}

// ==================== NOTIFICATIONS ====================

// GetNotifications lists notifications for current admin
func (h *Handler) GetNotifications(c *gin.Context) {
	adminID, _ := middleware.GetAdminID(c)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	notifications, total, err := h.notificationService.ListNotifications(c.Request.Context(), adminID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"notifications": notifications, "total": total})
}

// MarkNotificationRead marks a notification as read
func (h *Handler) MarkNotificationRead(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.notificationService.MarkAsRead(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "notification marked as read"})
}

// SendNotification sends a notification to an admin
func (h *Handler) SendNotification(c *gin.Context) {
	var req struct {
		AdminID          uuid.UUID `json:"admin_id"`
		Title            string    `json:"title" binding:"required"`
		Message          string    `json:"message" binding:"required"`
		NotificationType string    `json:"notification_type" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.notificationService.SendNotification(c.Request.Context(), req.AdminID, req.Title, req.Message, req.NotificationType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "notification sent"})
}

// BroadcastNotification broadcasts a notification to all admins
func (h *Handler) BroadcastNotification(c *gin.Context) {
	var req struct {
		Title            string `json:"title" binding:"required"`
		Message          string `json:"message" binding:"required"`
		NotificationType string `json:"notification_type" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.notificationService.Broadcast(c.Request.Context(), req.Title, req.Message, req.NotificationType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "notification broadcasted"})
}

// ==================== AUDIT LOGS ====================

// GetAuditLogs lists audit logs
func (h *Handler) GetAuditLogs(c *gin.Context) {
	adminID := c.Query("admin_id")
	action := c.Query("action")
	resourceType := c.Query("resource_type")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	logs, total, err := h.auditService.ListAuditLogs(c.Request.Context(), adminID, action, resourceType, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"audit_logs": logs, "total": total})
}

// ExportAuditLogs exports audit logs
func (h *Handler) ExportAuditLogs(c *gin.Context) {
	var req struct {
		Format   string `json:"format" binding:"required,oneof=csv json"`
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminID, _ := middleware.GetAdminID(c)
	filePath, err := h.auditService.ExportAuditLogs(c.Request.Context(), adminID, req.Format, req.StartDate, req.EndDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"file_path": filePath})
}

// ==================== SESSIONS ====================

// GetSessions lists sessions for an admin
func (h *Handler) GetSessions(c *gin.Context) {
	adminID, _ := middleware.GetAdminID(c)

	sessions, err := h.sessionService.ListSessions(c.Request.Context(), adminID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sessions": sessions})
}

// RevokeSession revokes a session
func (h *Handler) RevokeSession(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.sessionService.RevokeSession(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "session revoked"})
}

// RevokeAllSessions revokes all sessions for an admin
func (h *Handler) RevokeAllSessions(c *gin.Context) {
	adminID, _ := middleware.GetAdminID(c)

	if err := h.sessionService.RevokeAllSessions(c.Request.Context(), adminID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "all sessions revoked"})
}

// ==================== FEATURE FLAGS ====================

// GetFeatureFlags lists all feature flags
func (h *Handler) GetFeatureFlags(c *gin.Context) {
	flags, err := h.featureFlagService.ListFeatureFlags(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"feature_flags": flags})
}

// CreateFeatureFlag creates a new feature flag
func (h *Handler) CreateFeatureFlag(c *gin.Context) {
	var req models.FeatureFlag
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminID, _ := middleware.GetAdminID(c)
	flag, err := h.featureFlagService.CreateFeatureFlag(c.Request.Context(), &req, adminID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"feature_flag": flag})
}

// UpdateFeatureFlag updates a feature flag
func (h *Handler) UpdateFeatureFlag(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		IsEnabled         *bool `json:"is_enabled"`
		RolloutPercentage *int  `json:"rollout_percentage"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminID, _ := middleware.GetAdminID(c)
	if err := h.featureFlagService.UpdateFeatureFlag(c.Request.Context(), id, adminID, req.IsEnabled, req.RolloutPercentage); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "feature flag updated"})
}

// DeleteFeatureFlag deletes a feature flag
func (h *Handler) DeleteFeatureFlag(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.featureFlagService.DeleteFeatureFlag(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "feature flag deleted"})
}

// ==================== IP WHITELIST ====================

// GetIPWhitelist lists all IP whitelist entries
func (h *Handler) GetIPWhitelist(c *gin.Context) {
	entries, err := h.ipWhitelistService.ListIPWhitelist(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ip_whitelist": entries})
}

// AddIPToWhitelist adds an IP to whitelist
func (h *Handler) AddIPToWhitelist(c *gin.Context) {
	var req models.IPWhitelist
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminID, _ := middleware.GetAdminID(c)
	entry, err := h.ipWhitelistService.AddIP(c.Request.Context(), &req, adminID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"ip_whitelist": entry})
}

// RemoveIPFromWhitelist removes an IP from whitelist
func (h *Handler) RemoveIPFromWhitelist(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.ipWhitelistService.RemoveIP(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "IP removed from whitelist"})
}

// ==================== REPORTS ====================

// GenerateReport generates a new report
func (h *Handler) GenerateReport(c *gin.Context) {
	var req models.Report
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminID, _ := middleware.GetAdminID(c)
	report, err := h.reportService.GenerateReport(c.Request.Context(), &req, adminID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"report": report})
}

// GetReports lists all reports
func (h *Handler) GetReports(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	reports, total, err := h.reportService.ListReports(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"reports": reports, "total": total})
}

// ==================== TICKETS ====================

// GetTickets lists all tickets
func (h *Handler) GetTickets(c *gin.Context) {
	status := c.Query("status")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	tickets, total, err := h.ticketService.ListTickets(c.Request.Context(), status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tickets": tickets, "total": total})
}

// GetTicket gets a single ticket with messages
func (h *Handler) GetTicket(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	ticket, messages, err := h.ticketService.GetTicket(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ticket": ticket, "messages": messages})
}

// CreateTicket creates a new ticket
func (h *Handler) CreateTicket(c *gin.Context) {
	var req struct {
		Title       string `json:"title" binding:"required"`
		Description string `json:"description"`
		TicketType  string `json:"ticket_type" binding:"required"`
		Priority    string `json:"priority"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminID, _ := middleware.GetAdminID(c)
	ticket, err := h.ticketService.CreateTicket(c.Request.Context(), adminID, req.Title, req.Description, req.TicketType, req.Priority)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"ticket": ticket})
}

// UpdateTicketStatus updates ticket status
func (h *Handler) UpdateTicketStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.ticketService.UpdateTicketStatus(c.Request.Context(), id, req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ticket status updated"})
}

// AddTicketMessage adds a message to a ticket
func (h *Handler) AddTicketMessage(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Message    string `json:"message" binding:"required"`
		IsInternal bool   `json:"is_internal"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminID, _ := middleware.GetAdminID(c)
	message, err := h.ticketService.AddMessage(c.Request.Context(), id, adminID, req.Message, req.IsInternal)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": message})
}

// AssignTicket assigns a ticket to an admin
func (h *Handler) AssignTicket(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		AssignedTo uuid.UUID `json:"assigned_to" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.ticketService.AssignTicket(c.Request.Context(), id, req.AssignedTo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ticket assigned"})
}

// ==================== WHITE LABELS ====================

// GetWhiteLabels lists all white labels
func (h *Handler) GetWhiteLabels(c *gin.Context) {
	whiteLabels, err := h.whiteLabelService.ListWhiteLabels(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"white_labels": whiteLabels})
}

// CreateWhiteLabel creates a new white label
func (h *Handler) CreateWhiteLabel(c *gin.Context) {
	var req models.WhiteLabel
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	whiteLabel, err := h.whiteLabelService.CreateWhiteLabel(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"white_label": whiteLabel})
}

// UpdateWhiteLabel updates a white label
func (h *Handler) UpdateWhiteLabel(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Name           string `json:"name"`
		Domain         string `json:"domain"`
		LogoURL        string `json:"logo_url"`
		PrimaryColor   string `json:"primary_color"`
		SecondaryColor string `json:"secondary_color"`
		IsActive       *bool  `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.whiteLabelService.UpdateWhiteLabel(c.Request.Context(), id, req.Name, req.Domain, req.LogoURL, req.PrimaryColor, req.SecondaryColor, req.IsActive); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "white label updated"})
}

// DeleteWhiteLabel deletes a white label
func (h *Handler) DeleteWhiteLabel(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.whiteLabelService.DeleteWhiteLabel(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "white label deleted"})
}

// ==================== PLATFORM STATS ====================

// GetPlatformStats returns platform statistics
func (h *Handler) GetPlatformStats(c *gin.Context) {
	stats, err := h.userService.GetPlatformStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}
