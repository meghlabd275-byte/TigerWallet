package handlers

import (
	"net/http"

	"admin_console/internal/models"
	"admin_console/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "message": err.Error()})
		return
	}

	resp, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		switch err {
		case services.ErrInvalidCredentials:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		case services.ErrAccountLocked:
			c.JSON(http.StatusForbidden, gin.H{"error": "Account locked"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Login failed"})
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "message": err.Error()})
		return
	}

	resp, err := h.authService.Register(c.Request.Context(), &req)
	if err != nil {
		if err == services.ErrUserExists {
			c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Registration failed"})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	resp, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := h.authService.Logout(c.Request.Context(), userID.(uuid.UUID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Logout failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if err := h.authService.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a reset link has been sent"})
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if err := h.authService.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if err := h.authService.VerifyEmail(c.Request.Context(), req.Token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Verification failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Email verified successfully"})
}

// User Handler
type UserHandler struct {
	userService *services.UserService
}

func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

func (h *UserHandler) List(c *gin.Context) {
	var params models.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid parameters"})
		return
	}

	status := c.Query("status")
	role := c.Query("role")

	resp, err := h.userService.List(c.Request.Context(), &params, status, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *UserHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	user, profile, err := h.userService.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user, "profile": profile})
}

func (h *UserHandler) Create(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	user, err := h.userService.Create(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func (h *UserHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	user, err := h.userService.Update(c.Request.Context(), id, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if err := h.userService.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

func (h *UserHandler) UpdateStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required,oneof=active inactive suspended"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if err := h.userService.UpdateStatus(c.Request.Context(), id, req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Status updated successfully"})
}

func (h *UserHandler) GetActivity(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	activities, err := h.userService.GetActivity(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch activity"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"activities": activities})
}

// KYC Handler
type KYCHandler struct {
	kycService *services.KYCService
}

func NewKYCHandler(kycService *services.KYCService) *KYCHandler {
	return &KYCHandler{kycService: kycService}
}

func (h *KYCHandler) List(c *gin.Context) {
	var params models.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid parameters"})
		return
	}

	status := c.Query("status")

	resp, err := h.kycService.List(c.Request.Context(), &params, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch KYC records"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *KYCHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid KYC ID"})
		return
	}

	kyc, err := h.kycService.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "KYC not found"})
		return
	}

	c.JSON(http.StatusOK, kyc)
}

func (h *KYCHandler) Approve(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid KYC ID"})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	if err := h.kycService.Approve(c.Request.Context(), id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to approve KYC"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "KYC approved successfully"})
}

func (h *KYCHandler) Reject(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid KYC ID"})
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Reason required"})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	if err := h.kycService.Reject(c.Request.Context(), id, userID, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reject KYC"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "KYC rejected"})
}

func (h *KYCHandler) RequestInfo(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid KYC ID"})
		return
	}

	var req struct {
		Message string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Message required"})
		return
	}

	if err := h.kycService.RequestInfo(c.Request.Context(), id, req.Message); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to request info"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Info requested"})
}

func (h *KYCHandler) GetStats(c *gin.Context) {
	stats, err := h.kycService.GetStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// Token Handler
type TokenHandler struct {
	tokenService *services.TokenService
}

func NewTokenHandler(tokenService *services.TokenService) *TokenHandler {
	return &TokenHandler{tokenService: tokenService}
}

func (h *TokenHandler) List(c *gin.Context) {
	var params models.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid parameters"})
		return
	}

	status := c.Query("status")

	resp, err := h.tokenService.List(c.Request.Context(), &params, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tokens"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *TokenHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token ID"})
		return
	}

	token, err := h.tokenService.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
		return
	}

	c.JSON(http.StatusOK, token)
}

func (h *TokenHandler) Create(c *gin.Context) {
	var token models.Token
	if err := c.ShouldBindJSON(&token); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	created, err := h.tokenService.Create(c.Request.Context(), &token, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create token"})
		return
	}

	c.JSON(http.StatusCreated, created)
}

func (h *TokenHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token ID"})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	token, err := h.tokenService.Update(c.Request.Context(), id, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update token"})
		return
	}

	c.JSON(http.StatusOK, token)
}

func (h *TokenHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token ID"})
		return
	}

	if err := h.tokenService.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Token deleted successfully"})
}

func (h *TokenHandler) Approve(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token ID"})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	if err := h.tokenService.Approve(c.Request.Context(), id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to approve token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Token approved"})
}

func (h *TokenHandler) Reject(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token ID"})
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Reason required"})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	if err := h.tokenService.Reject(c.Request.Context(), id, userID, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reject token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Token rejected"})
}

func (h *TokenHandler) GetHolders(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token ID"})
		return
	}

	holders, err := h.tokenService.GetHolders(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get holders"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"holders": holders})
}

// Transaction Handler
type TransactionHandler struct {
	txService *services.TransactionService
}

func NewTransactionHandler(txService *services.TransactionService) *TransactionHandler {
	return &TransactionHandler{txService: txService}
}

func (h *TransactionHandler) List(c *gin.Context) {
	var params models.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid parameters"})
		return
	}

	status := c.Query("status")
	txType := c.Query("type")

	resp, err := h.txService.List(c.Request.Context(), &params, status, txType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *TransactionHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction ID"})
		return
	}

	tx, err := h.txService.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
		return
	}

	c.JSON(http.StatusOK, tx)
}

func (h *TransactionHandler) Flag(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction ID"})
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Reason required"})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	if err := h.txService.Flag(c.Request.Context(), id, userID, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to flag transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Transaction flagged"})
}

func (h *TransactionHandler) Approve(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction ID"})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	if err := h.txService.Approve(c.Request.Context(), id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to approve transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Transaction approved"})
}

func (h *TransactionHandler) Reject(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction ID"})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	if err := h.txService.Reject(c.Request.Context(), id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reject transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Transaction rejected"})
}

func (h *TransactionHandler) Cancel(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction ID"})
		return
	}

	if err := h.txService.Cancel(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Transaction cancelled"})
}

func (h *TransactionHandler) GetStats(c *gin.Context) {
	stats, err := h.txService.GetStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *TransactionHandler) GetPending(c *gin.Context) {
	resp, err := h.txService.GetPending(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get pending transactions"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Analytics Handler
type AnalyticsHandler struct {
	analyticsService *services.AnalyticsService
}

func NewAnalyticsHandler(analyticsService *services.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{analyticsService: analyticsService}
}

func (h *AnalyticsHandler) Dashboard(c *gin.Context) {
	stats, err := h.analyticsService.Dashboard(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get dashboard stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *AnalyticsHandler) UserAnalytics(c *gin.Context) {
	analytics, err := h.analyticsService.UserAnalytics(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user analytics"})
		return
	}

	c.JSON(http.StatusOK, analytics)
}

func (h *AnalyticsHandler) TransactionAnalytics(c *gin.Context) {
	analytics, err := h.analyticsService.TransactionAnalytics(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get transaction analytics"})
		return
	}

	c.JSON(http.StatusOK, analytics)
}

func (h *AnalyticsHandler) RevenueAnalytics(c *gin.Context) {
	analytics, err := h.analyticsService.RevenueAnalytics(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get revenue analytics"})
		return
	}

	c.JSON(http.StatusOK, analytics)
}

func (h *AnalyticsHandler) GrowthAnalytics(c *gin.Context) {
	analytics, err := h.analyticsService.GrowthAnalytics(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get growth analytics"})
		return
	}

	c.JSON(http.StatusOK, analytics)
}

// Audit Handler
type AuditHandler struct {
	auditService *services.AuditService
}

func NewAuditHandler(auditService *services.AuditService) *AuditHandler {
	return &AuditHandler{auditService: auditService}
}

func (h *AuditHandler) List(c *gin.Context) {
	var params models.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid parameters"})
		return
	}

	action := c.Query("action")
	resourceType := c.Query("resource_type")

	resp, err := h.auditService.List(c.Request.Context(), &params, action, resourceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch audit logs"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *AuditHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid audit log ID"})
		return
	}

	log, err := h.auditService.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Audit log not found"})
		return
	}

	c.JSON(http.StatusOK, log)
}

func (h *AuditHandler) GetByUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	resp, err := h.auditService.GetByUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch audit logs"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *AuditHandler) Export(c *gin.Context) {
	// Implementation for exporting audit logs
	c.JSON(http.StatusOK, gin.H{"message": "Export functionality"})
}

// Notification Handler
type NotificationHandler struct {
	notificationService *services.NotificationService
}

func NewNotificationHandler(notificationService *services.NotificationService) *NotificationHandler {
	return &NotificationHandler{notificationService: notificationService}
}

func (h *NotificationHandler) List(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	var params models.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid parameters"})
		return
	}

	resp, err := h.notificationService.List(c.Request.Context(), userID, &params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *NotificationHandler) MarkRead(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	if err := h.notificationService.MarkRead(c.Request.Context(), userID, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Marked as read"})
}

func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	if err := h.notificationService.MarkAllRead(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark all as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All marked as read"})
}

func (h *NotificationHandler) Send(c *gin.Context) {
	var notification models.Notification
	if err := c.ShouldBindJSON(&notification); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	notification.ID = uuid.New()

	if err := h.notificationService.Send(c.Request.Context(), &notification); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send notification"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Notification sent"})
}

// Compliance Handler
type ComplianceHandler struct {
	complianceService *services.ComplianceService
}

func NewComplianceHandler(complianceService *services.ComplianceService) *ComplianceHandler {
	return &ComplianceHandler{complianceService: complianceService}
}

func (h *ComplianceHandler) ListReports(c *gin.Context) {
	var params models.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid parameters"})
		return
	}

	resp, err := h.complianceService.ListReports(c.Request.Context(), &params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reports"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *ComplianceHandler) CreateReport(c *gin.Context) {
	var report models.ComplianceReport
	if err := c.ShouldBindJSON(&report); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	created, err := h.complianceService.CreateReport(c.Request.Context(), &report, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create report"})
		return
	}

	c.JSON(http.StatusCreated, created)
}

func (h *ComplianceHandler) GetReport(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid report ID"})
		return
	}

	report, err := h.complianceService.GetReport(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Report not found"})
		return
	}

	c.JSON(http.StatusOK, report)
}

func (h *ComplianceHandler) CheckSanctions(c *gin.Context) {
	var req struct {
		Address string `json:"address" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Address required"})
		return
	}

	result, err := h.complianceService.CheckSanctions(c.Request.Context(), req.Address)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check sanctions"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *ComplianceHandler) AMLCheck(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID required"})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	result, err := h.complianceService.AMLCheck(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to perform AML check"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// System handlers
func GetSystemStats(c *gin.Context) {
	stats := map[string]interface{}{
		"total_users":        0,
		"active_users":       0,
		"total_transactions": 0,
		"system_health":      "healthy",
		"uptime":             "24h",
	}
	c.JSON(http.StatusOK, stats)
}

func GetConfig(c *gin.Context) {
	config := map[string]interface{}{
		"maintenance_mode":     false,
		"registration_enabled": true,
		"kyc_required":         true,
		"withdrawal_limit":     "10000",
	}
	c.JSON(http.StatusOK, config)
}

func UpdateConfig(c *gin.Context) {
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Configuration updated", "updates": updates})
}
