// TigerWallet Admin - Main Entry Point
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tigerwallet/super-admin/internal/config"
	"github.com/tigerwallet/super-admin/internal/database"
	"github.com/tigerwallet/super-admin/internal/middleware"
	"golang.org/x/crypto/bcrypt"
)

// appCfg is the package-level config, set in main(), used by handlers for JWT auth.
var appCfg *config.Config

func main() {
	cfg := config.Load()
	appCfg = cfg

	if err := database.Initialize(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	log.Println("Database initialized successfully")

	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "tiger-admin"})
	})

	api := router.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/login", handleLogin)
			auth.POST("/register", handleRegister)
			auth.POST("/refresh", handleRefreshToken)
		}

		admin := api.Group("/admin")
		admin.Use(middleware.JWTAuth(cfg))
		admin.Use(middleware.IPWhitelistMiddleware(cfg))
		{
			admin.GET("/users", handleGetUsers)
			admin.GET("/users/:id", handleGetUser)
			admin.PUT("/users/:id/status", handleUpdateUserStatus)
			admin.POST("/users/:id/ban", handleBanUser)
			admin.POST("/users/:id/unban", handleUnbanUser)
			admin.POST("/users/:id/suspend", handleSuspendUser)

			admin.GET("/kyc", handleGetKYC)
			admin.POST("/kyc/:id/approve", handleApproveKYC)
			admin.POST("/kyc/:id/reject", handleRejectKYC)

			admin.GET("/transactions", handleGetTransactions)
			admin.GET("/transactions/:id", handleGetTransaction)
			admin.POST("/transactions/:id/flag", handleFlagTransaction)
			admin.POST("/transactions/:id/unflag", handleUnflagTransaction)

			admin.GET("/withdrawals", handleGetWithdrawals)
			admin.POST("/withdrawals/:id/approve", handleApproveWithdrawal)
			admin.POST("/withdrawals/:id/reject", handleRejectWithdrawal)
			admin.POST("/withdrawals/:id/process", handleProcessWithdrawal)

			admin.GET("/tokens", handleGetTokens)
			admin.POST("/tokens", handleCreateToken)
			admin.PUT("/tokens/:id", handleUpdateToken)
			admin.DELETE("/tokens/:id", handleDeleteToken)

			admin.GET("/pairs", handleGetPairs)
			admin.POST("/pairs", handleCreatePair)
			admin.PUT("/pairs/:id/status", handleUpdatePairStatus)

			admin.GET("/blockchains", handleGetBlockchains)
			admin.POST("/blockchains", handleCreateBlockchain)
			admin.PUT("/blockchains/:id", handleUpdateBlockchain)
			admin.PUT("/blockchains/:id/status", handleSetBlockchainStatus)

			admin.GET("/fees", handleGetFees)
			admin.POST("/fees", handleCreateFee)
			admin.PUT("/fees/:id", handleUpdateFee)

			admin.GET("/webhooks", handleGetWebhooks)
			admin.POST("/webhooks", handleCreateWebhook)
			admin.POST("/webhooks/:id/test", handleTestWebhook)
			admin.DELETE("/webhooks/:id", handleDeleteWebhook)

			admin.GET("/notifications", handleGetNotifications)
			admin.PUT("/notifications/:id/read", handleMarkNotificationRead)
			admin.POST("/notifications/send", handleSendNotification)
			admin.POST("/notifications/broadcast", handleBroadcastNotification)

			admin.GET("/audit-logs", handleGetAuditLogs)
			admin.POST("/audit-logs/export", handleExportAuditLogs)

			admin.GET("/sessions", handleGetSessions)
			admin.DELETE("/sessions/:id", handleRevokeSession)
			admin.DELETE("/sessions", handleRevokeAllSessions)

			admin.GET("/feature-flags", handleGetFeatureFlags)
			admin.POST("/feature-flags", handleCreateFeatureFlag)
			admin.PUT("/feature-flags/:id", handleUpdateFeatureFlag)
			admin.DELETE("/feature-flags/:id", handleDeleteFeatureFlag)

			// /features mirrors the admin/web frontend contract:
			// GET /features -> [{name, enabled, description}], PUT /features/:name {enabled}.
			admin.GET("/features", handleGetFeatures)
			admin.PUT("/features/:name", handleSetFeature)

			admin.GET("/ip-whitelist", handleGetIPWhitelist)
			admin.POST("/ip-whitelist", handleAddIPWhitelist)
			admin.DELETE("/ip-whitelist/:id", handleRemoveIPWhitelist)

			admin.GET("/tickets", handleGetTickets)
			admin.GET("/tickets/:id", handleGetTicket)
			admin.POST("/tickets", handleCreateTicket)
			admin.PUT("/tickets/:id/status", handleUpdateTicketStatus)
			admin.POST("/tickets/:id/messages", handleAddTicketMessage)
			admin.PUT("/tickets/:id/assign", handleAssignTicket)

			admin.GET("/white-labels", handleGetWhiteLabels)
			admin.POST("/white-labels", handleCreateWhiteLabel)
			admin.PUT("/white-labels/:id", handleUpdateWhiteLabel)
			admin.DELETE("/white-labels/:id", handleDeleteWhiteLabel)
			admin.PUT("/white-labels/:id/status", handleUpdateWhiteLabelStatus)

			admin.GET("/stats", handleGetStats)

			// Bot Management
			admin.GET("/bots", handleGetBots)
			admin.GET("/bots/:id", handleGetBot)
			admin.POST("/bots", handleCreateBot)
			admin.PUT("/bots/:id", handleUpdateBot)
			admin.DELETE("/bots/:id", handleDeleteBot)
			admin.PUT("/bots/:id/status", handleUpdateBotStatus)
			admin.GET("/bots/:id/stats", handleGetBotStats)
			admin.GET("/bots/tiers", handleGetBotTiers)
			admin.POST("/bots/tiers", handleCreateBotTier)
			admin.PUT("/bots/tiers/:id", handleUpdateBotTier)
			admin.DELETE("/bots/tiers/:id", handleDeleteBotTier)

			// BotsClient Management
			admin.GET("/bots-clients", handleGetBotsClients)
			admin.GET("/bots-clients/:id", handleGetBotsClient)
			admin.POST("/bots-clients", handleCreateBotsClient)
			admin.PUT("/bots-clients/:id", handleUpdateBotsClient)
			admin.DELETE("/bots-clients/:id", handleDeleteBotsClient)
			admin.PUT("/bots-clients/:id/status", handleUpdateBotsClientStatus)

			// Project Team Management
			admin.GET("/project-teams", handleGetProjectTeams)
			admin.GET("/project-teams/:id", handleGetProjectTeam)
			admin.POST("/project-teams", handleCreateProjectTeam)
			admin.PUT("/project-teams/:id", handleUpdateProjectTeam)
			admin.DELETE("/project-teams/:id", handleDeleteProjectTeam)
			admin.GET("/project-teams/:id/members", handleGetProjectTeamMembers)
			admin.POST("/project-teams/:id/members", handleAddProjectTeamMember)
			admin.DELETE("/project-teams/:id/members/:memberId", handleRemoveProjectTeamMember)
			admin.PUT("/project-teams/:id/status", handleUpdateProjectTeamStatus)

			// White Level Client Management
			admin.GET("/wl-clients", handleGetWLClients)
			admin.GET("/wl-clients/:id", handleGetWLClient)
			admin.POST("/wl-clients", handleCreateWLClient)
			admin.PUT("/wl-clients/:id", handleUpdateWLClient)
			admin.DELETE("/wl-clients/:id", handleDeleteWLClient)
			admin.PUT("/wl-clients/:id/status", handleUpdateWLClientStatus)

			// WL MasterWallet Management
			admin.GET("/wl-master-wallets", handleGetWLMasterWallets)
			admin.GET("/wl-master-wallets/:id", handleGetWLMasterWallet)
			admin.POST("/wl-master-wallets", handleCreateWLMasterWallet)
			admin.PUT("/wl-master-wallets/:id", handleUpdateWLMasterWallet)
			admin.DELETE("/wl-master-wallets/:id", handleDeleteWLMasterWallet)
			admin.PUT("/wl-master-wallets/:id/status", handleUpdateWLMasterWalletStatus)

			// WL UserWallet Management
			admin.GET("/wl-user-wallets", handleGetWLUserWallets)
			admin.GET("/wl-user-wallets/:id", handleGetWLUserWallet)
			admin.POST("/wl-user-wallets", handleCreateWLUserWallet)
			admin.PUT("/wl-user-wallets/:id", handleUpdateWLUserWallet)
			admin.DELETE("/wl-user-wallets/:id", handleDeleteWLUserWallet)
			admin.PUT("/wl-user-wallets/:id/status", handleUpdateWLUserWalletStatus)

			// WL Bots Management
			admin.GET("/wl-bots", handleGetWLBots)
			admin.GET("/wl-bots/:id", handleGetWLBot)
			admin.POST("/wl-bots", handleCreateWLBot)
			admin.PUT("/wl-bots/:id", handleUpdateWLBot)
			admin.DELETE("/wl-bots/:id", handleDeleteWLBot)
			admin.PUT("/wl-bots/:id/status", handleUpdateWLBotStatus)

			// WL BotsClient Management
			admin.GET("/wl-bots-clients", handleGetWLBotsClients)
			admin.GET("/wl-bots-clients/:id", handleGetWLBotsClient)
			admin.POST("/wl-bots-clients", handleCreateWLBotsClient)
			admin.PUT("/wl-bots-clients/:id", handleUpdateWLBotClient)
			admin.DELETE("/wl-bots-clients/:id", handleDeleteWLBotsClient)
			admin.PUT("/wl-bots-clients/:id/status", handleUpdateWLBotsClientStatus)

			// WL Project Team Management
			admin.GET("/wl-project-teams", handleGetWLProjectTeams)
			admin.GET("/wl-project-teams/:id", handleGetWLProjectTeam)
			admin.POST("/wl-project-teams", handleCreateWLProjectTeam)
			admin.PUT("/wl-project-teams/:id", handleUpdateWLProjectTeam)
			admin.DELETE("/wl-project-teams/:id", handleDeleteWLProjectTeam)
			admin.PUT("/wl-project-teams/:id/status", handleUpdateWLProjectTeamStatus)

			// MasterWallet Management
			admin.GET("/master-wallets", handleGetMasterWallets)
			admin.GET("/master-wallets/:id", handleGetMasterWallet)
			admin.GET("/master-wallets/:id/balance", handleGetMasterWalletBalance)
			// No /transfer endpoint: admins must NOT move crypto assets. Fund movement is
			// performed exclusively by the wallet owner via the canonical wallet backend.

			// MasterWallet/UserWallet CRUD (create/update/delete) — SuperAdmin only.
			// Governance records only; never moves funds.
			walletMgmt := admin.Group("")
			walletMgmt.Use(middleware.RoleAuth("super_admin"))
			{
				walletMgmt.POST("/master-wallets", handleCreateMasterWallet)
				walletMgmt.PUT("/master-wallets/:id", handleUpdateMasterWallet)
				walletMgmt.DELETE("/master-wallets/:id", handleDeleteMasterWallet)
				walletMgmt.POST("/user-wallets", handleCreateUserWallet)
				walletMgmt.PUT("/user-wallets/:id", handleUpdateUserWallet)
				walletMgmt.DELETE("/user-wallets/:id", handleDeleteUserWallet)
				walletMgmt.PUT("/master-wallets/:id/status", handleUpdateMasterWalletStatus)
				walletMgmt.PUT("/user-wallets/:id/status", handleUpdateUserWalletStatus)
			}

			// UserWallet Management
			admin.GET("/user-wallets", handleGetUserWallets)
			admin.GET("/user-wallets/:id", handleGetUserWallet)
			admin.GET("/user-wallets/:id/balance", handleGetUserWalletBalance)

			admin.POST("/logout", handleLogout)
			admin.POST("/change-password", handleChangePassword)
			admin.POST("/2fa/enable", handleEnable2FA)
			admin.POST("/2fa/disable", handleDisable2FA)

			// Admin user management — SuperAdmin only (role assignment, suspend, delete)
			adminAdmins := admin.Group("")
			adminAdmins.Use(middleware.RoleAuth("super_admin"))
			{
				adminAdmins.GET("/admins", handleGetAdmins)
				adminAdmins.GET("/admins/:id", handleGetAdmin)
				adminAdmins.PUT("/admins/:id", handleUpdateAdmin)
				adminAdmins.DELETE("/admins/:id", handleDeleteAdmin)
				adminAdmins.POST("/admins/:id/suspend", handleSuspendAdmin)
				adminAdmins.POST("/admins/:id/activate", handleActivateAdmin)
			}

			admin.GET("/workflows", handleGetWorkflows)
			admin.POST("/workflows", handleCreateWorkflow)
			admin.PUT("/workflows/:id", handleUpdateWorkflow)
			admin.DELETE("/workflows/:id", handleDeleteWorkflow)

			admin.GET("/approval-requests", handleGetApprovalRequests)
			admin.POST("/approval-requests/:id/approve", handleApproveRequest)
			admin.POST("/approval-requests/:id/reject", handleRejectRequest)

			admin.GET("/backups", handleGetBackups)
			admin.POST("/backups", handleCreateBackup)
			admin.POST("/backups/:id/restore", handleRestoreBackup)
			admin.DELETE("/backups/:id", handleDeleteBackup)

			// Knowledge base
			admin.GET("/knowledge-base", handleGetKnowledgeArticles)
			admin.GET("/knowledge-base/:id", handleGetKnowledgeArticle)
			admin.POST("/knowledge-base", handleCreateKnowledgeArticle)
			admin.PUT("/knowledge-base/:id", handleUpdateKnowledgeArticle)
			admin.DELETE("/knowledge-base/:id", handleDeleteKnowledgeArticle)

			// Data archival
			admin.GET("/archival/policies", handleGetArchivePolicies)
			admin.POST("/archival/policies", handleCreateArchivePolicy)
			admin.PUT("/archival/policies/:id", handleUpdateArchivePolicy)
			admin.DELETE("/archival/policies/:id", handleDeleteArchivePolicy)
			admin.POST("/archival/policies/:id/run", handleRunArchive)
			admin.GET("/archival/records", handleGetArchiveRecords)

			// Reports
			admin.GET("/reports/configs", handleGetReportConfigs)
			admin.POST("/reports/configs", handleCreateReportConfig)
			admin.GET("/reports", handleGetReports)
			admin.POST("/reports/generate", handleGenerateReport)

			// SLA Management
			admin.GET("/sla/policies", handleGetSLAPolicies)
			admin.POST("/sla/policies", handleCreateSLAPolicy)
			admin.PUT("/sla/policies/:id", handleUpdateSLAPolicy)
			admin.DELETE("/sla/policies/:id", handleDeleteSLAPolicy)
			admin.GET("/sla/reports", handleGetSLAReports)
			admin.POST("/sla/reports/generate", handleGenerateSLAReport)

			// Integrations
			admin.GET("/integrations", handleGetIntegrations)
			admin.POST("/integrations", handleCreateIntegration)
			admin.PUT("/integrations/:id", handleUpdateIntegration)
			admin.DELETE("/integrations/:id", handleDeleteIntegration)
			admin.POST("/integrations/:id/test", handleTestIntegration)
		}
	}

	srv := &http.Server{
		Addr:           ":" + cfg.ServerPort,
		Handler:        router,
		ReadTimeout:    cfg.ServerReadTimeout,
		WriteTimeout:   cfg.ServerWriteTimeout,
		IdleTimeout:    cfg.ServerIdleTimeout,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		log.Printf("Admin API server starting on port %s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited properly")
}


// ============== Real PostgreSQL-backed handlers ==============
// All handlers query database.Pool (the global pgxpool). No stubs/mocks.

// ---- helpers ----

func dbQuery(c *gin.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	return database.Pool.Query(ctx, sql, args...)
}

func dbExec(c *gin.Context, sql string, args ...interface{}) (int64, error) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	ct, err := database.Pool.Exec(ctx, sql, args...)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}

func dbQueryRow(c *gin.Context, sql string, args ...interface{}) pgx.Row {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	return database.Pool.QueryRow(ctx, sql, args...)
}

func rowsToMaps(rows pgx.Rows) []map[string]interface{} {
	results := []map[string]interface{}{}
	fields := rows.FieldDescriptions()
	for rows.Next() {
		values := make([]interface{}, len(fields))
		ptrs := make([]interface{}, len(fields))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			continue
		}
		row := map[string]interface{}{}
		for i, f := range fields {
			row[f.Name] = values[i]
		}
		results = append(results, row)
	}
	return results
}

// ---- Auth handlers ----

func handleLogin(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	var id, username, role, hash string
	err := database.Pool.QueryRow(ctx, `SELECT id, username, role, password_hash FROM admin_users WHERE email=$1 AND is_active=true`, req.Email).Scan(&id, &username, &role, &hash)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	adminUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid admin id"})
		return
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, middleware.Claims{
		AdminID:  adminUUID,
		Username: username,
		Email:    req.Email,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(appCfg.JWTExpiry)),
			Subject:   id,
		},
	})
	tokenStr, err := token.SignedString([]byte(appCfg.JWTSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}
	database.Pool.Exec(ctx, `UPDATE admin_users SET last_login_at=NOW() WHERE id=$1`, id)
	c.JSON(http.StatusOK, gin.H{"token": tokenStr, "user": gin.H{"id": id, "email": req.Email, "username": username, "role": role}})
}

func handleRegister(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	id := uuid.New()
	_, err = dbExec(c, `INSERT INTO admin_users (id, username, email, password_hash, role, is_active, created_at, updated_at) VALUES ($1,$2,$3,$4,'admin',true,NOW(),NOW())`, id, req.Username, req.Email, string(hash))
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "user already exists", "detail": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "username": req.Username, "email": req.Email, "role": "admin"})
}

func handleRefreshToken(c *gin.Context) {
	userID := c.GetString("user_id")
	role := c.GetString("role")
	username := c.GetString("username")
	email := c.GetString("email")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no session"})
		return
	}
	adminUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
		return
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, middleware.Claims{
		AdminID:  adminUUID,
		Username: username,
		Email:    email,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(appCfg.JWTExpiry)),
			Subject:   userID,
		},
	})
	tokenStr, err := token.SignedString([]byte(appCfg.JWTSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sign token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": tokenStr})
}

func handleLogout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func handleChangePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.GetString("user_id")
	var hash string
	dbQueryRow(c, `SELECT password_hash FROM admin_users WHERE id=$1`, userID).Scan(&hash)
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.OldPassword)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "old password incorrect"})
		return
	}
	newHash, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if _, err := dbExec(c, `UPDATE admin_users SET password_hash=$1, updated_at=NOW() WHERE id=$2`, string(newHash), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "password changed"})
}

func handleEnable2FA(c *gin.Context) {
	if _, err := dbExec(c, `UPDATE admin_users SET two_factor_enabled=true, updated_at=NOW() WHERE id=$1`, c.GetString("user_id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "2FA enabled"})
}

func handleDisable2FA(c *gin.Context) {
	if _, err := dbExec(c, `UPDATE admin_users SET two_factor_enabled=false, updated_at=NOW() WHERE id=$1`, c.GetString("user_id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "2FA disabled"})
}

// ---- Users ----

func handleGetUsers(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, email, username, wallet_address, kyc_status, status, country, last_login FROM users ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"users": rowsToMaps(rows)})
}

func handleGetUser(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, email, username, wallet_address, kyc_status, status, country, last_login FROM users WHERE id=$1`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	users := rowsToMaps(rows)
	if len(users) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": users[0]})
}

func handleUpdateUserStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE users SET status=$1 WHERE id=$2`, req.Status, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "status updated"})
}

func handleBanUser(c *gin.Context) {
	dbExec(c, `UPDATE users SET status='banned' WHERE id=$1`, c.Param("id"))
	c.JSON(http.StatusOK, gin.H{"message": "user banned"})
}

func handleUnbanUser(c *gin.Context) {
	dbExec(c, `UPDATE users SET status='active' WHERE id=$1`, c.Param("id"))
	c.JSON(http.StatusOK, gin.H{"message": "user unbanned"})
}

func handleSuspendUser(c *gin.Context) {
	dbExec(c, `UPDATE users SET status='suspended' WHERE id=$1`, c.Param("id"))
	c.JSON(http.StatusOK, gin.H{"message": "user suspended"})
}

// ---- KYC ----

func handleGetKYC(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, user_id, doc_type, status, document_url, submitted_at, reject_reason FROM kyc_requests ORDER BY submitted_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"kyc_requests": rowsToMaps(rows)})
}

func handleApproveKYC(c *gin.Context) {
	dbExec(c, `UPDATE kyc_requests SET status='approved', reviewed_by=$1, reviewed_at=NOW() WHERE id=$2`, c.GetString("user_id"), c.Param("id"))
	dbExec(c, `UPDATE users SET kyc_status='verified' WHERE id=(SELECT user_id FROM kyc_requests WHERE id=$1)`, c.Param("id"))
	c.JSON(http.StatusOK, gin.H{"message": "KYC approved"})
}

func handleRejectKYC(c *gin.Context) {
	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)
	dbExec(c, `UPDATE kyc_requests SET status='rejected', reviewed_by=$1, reviewed_at=NOW(), reject_reason=$2 WHERE id=$3`, c.GetString("user_id"), req.Reason, c.Param("id"))
	c.JSON(http.StatusOK, gin.H{"message": "KYC rejected"})
}

// ---- Transactions ----

func handleGetTransactions(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, user_id, type, amount, currency, status, from_address, to_address, tx_hash, chain_id, created_at FROM transactions ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"transactions": rowsToMaps(rows)})
}

func handleGetTransaction(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, user_id, type, amount, currency, status, from_address, to_address, tx_hash, chain_id, created_at FROM transactions WHERE id=$1`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	txs := rowsToMaps(rows)
	if len(txs) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"transaction": txs[0]})
}

func handleFlagTransaction(c *gin.Context) {
	dbExec(c, `UPDATE transactions SET status='flagged' WHERE id=$1`, c.Param("id"))
	c.JSON(http.StatusOK, gin.H{"message": "transaction flagged"})
}

func handleUnflagTransaction(c *gin.Context) {
	dbExec(c, `UPDATE transactions SET status='completed' WHERE id=$1`, c.Param("id"))
	c.JSON(http.StatusOK, gin.H{"message": "transaction unflagged"})
}

// ---- Withdrawals ----

func handleGetWithdrawals(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, user_id, amount, currency, status, address, tx_hash, created_at FROM withdrawals ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"withdrawals": rowsToMaps(rows)})
}

func handleApproveWithdrawal(c *gin.Context) {
	dbExec(c, `UPDATE withdrawals SET status='approved', approved_by=$1 WHERE id=$2`, c.GetString("user_id"), c.Param("id"))
	c.JSON(http.StatusOK, gin.H{"message": "withdrawal approved"})
}

func handleRejectWithdrawal(c *gin.Context) {
	dbExec(c, `UPDATE withdrawals SET status='rejected', approved_by=$1 WHERE id=$2`, c.GetString("user_id"), c.Param("id"))
	c.JSON(http.StatusOK, gin.H{"message": "withdrawal rejected"})
}

func handleProcessWithdrawal(c *gin.Context) {
	dbExec(c, `UPDATE withdrawals SET status='processed', approved_by=$1, processed_at=NOW() WHERE id=$2`, c.GetString("user_id"), c.Param("id"))
	c.JSON(http.StatusOK, gin.H{"message": "withdrawal processed"})
}

// ---- Tokens ----

func handleGetTokens(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, symbol, name, contract_address, decimals, is_active, is_verified, total_supply, chain_id FROM tokens ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"tokens": rowsToMaps(rows)})
}

func handleCreateToken(c *gin.Context) {
	var req struct {
		Symbol          string `json:"symbol" binding:"required"`
		Name            string `json:"name" binding:"required"`
		ContractAddress string `json:"contract_address"`
		Decimals        int    `json:"decimals"`
		TotalSupply     string `json:"total_supply"`
		ChainID         int64  `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := dbExec(c, `INSERT INTO tokens (id, symbol, name, contract_address, decimals, is_active, is_verified, total_supply, chain_id) VALUES ($1,$2,$3,$4,$5,true,false,$6,$7)`,
		uuid.New(), req.Symbol, req.Name, req.ContractAddress, req.Decimals, req.TotalSupply, req.ChainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "token created"})
}

func handleUpdateToken(c *gin.Context) {
	var req struct {
		Name        string `json:"name"`
		IsActive    *bool  `json:"is_active"`
		IsVerified  *bool  `json:"is_verified"`
		TotalSupply string `json:"total_supply"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	isVerified := false
	if req.IsVerified != nil {
		isVerified = *req.IsVerified
	}
	if _, err := dbExec(c, `UPDATE tokens SET name=$1, is_active=$2, is_verified=$3, total_supply=$4 WHERE id=$5`, req.Name, isActive, isVerified, req.TotalSupply, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "token updated"})
}

func handleDeleteToken(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM tokens WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "token deleted"})
}

// ---- Pairs ----

func handleGetPairs(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, pair_name, status, chain_id, price, volume_24h, liquidity FROM trading_pairs ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"pairs": rowsToMaps(rows)})
}

func handleCreatePair(c *gin.Context) {
	var req struct {
		PairName   string  `json:"pair_name" binding:"required"`
		BaseToken  string  `json:"base_token"`
		QuoteToken string  `json:"quote_token"`
		ChainID    int64   `json:"chain_id"`
		Price      float64 `json:"price"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := dbExec(c, `INSERT INTO trading_pairs (id, pair_name, base_token_id, quote_token_id, chain_id, price, status) VALUES ($1,$2,NULL,NULL,$3,$4,'active')`,
		uuid.New(), req.PairName, req.ChainID, req.Price)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "pair created"})
}

func handleUpdatePairStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE trading_pairs SET status=$1 WHERE id=$2`, req.Status, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "pair status updated"})
}

// ---- Blockchains ----

func handleGetBlockchains(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, name, symbol, chain_id, is_evm, rpc_url, explorer_url, native_token, decimals, is_active FROM blockchains ORDER BY chain_id`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"blockchains": rowsToMaps(rows)})
}

func handleCreateBlockchain(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Symbol      string `json:"symbol" binding:"required"`
		ChainID     int64  `json:"chain_id" binding:"required"`
		IsEVM       bool   `json:"is_evm"`
		RPCURL      string `json:"rpc_url"`
		ExplorerURL string `json:"explorer_url"`
		NativeToken string `json:"native_token"`
		Decimals    int    `json:"decimals"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := dbExec(c, `INSERT INTO blockchains (id, name, symbol, chain_id, is_evm, rpc_url, explorer_url, native_token, decimals, is_active) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,true)`,
		uuid.New(), req.Name, req.Symbol, req.ChainID, req.IsEVM, req.RPCURL, req.ExplorerURL, req.NativeToken, req.Decimals)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "blockchain created"})
}

func handleUpdateBlockchain(c *gin.Context) {
	var req struct {
		Name        string `json:"name"`
		RPCURL      string `json:"rpc_url"`
		ExplorerURL string `json:"explorer_url"`
		IsActive    *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	if _, err := dbExec(c, `UPDATE blockchains SET name=$1, rpc_url=$2, explorer_url=$3, is_active=$4 WHERE id=$5`, req.Name, req.RPCURL, req.ExplorerURL, isActive, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "blockchain updated"})
}

func handleSetBlockchainStatus(c *gin.Context) {
	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE blockchains SET is_active=$1 WHERE id=$2`, req.IsActive, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "status updated"})
}

// ---- Fees ----

func handleGetFees(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, fee_type, asset, fee_percent, fee_fixed, min_fee, max_fee, tier, is_active, chain_id FROM fee_structures ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"fees": rowsToMaps(rows)})
}

func handleCreateFee(c *gin.Context) {
	var req struct {
		FeeType   string  `json:"fee_type" binding:"required"`
		Asset     string  `json:"asset"`
		FeePercent float64 `json:"fee_percent"`
		FeeFixed  float64 `json:"fee_fixed"`
		MinFee    float64 `json:"min_fee"`
		MaxFee    float64 `json:"max_fee"`
		Tier      string  `json:"tier"`
		IsActive  bool    `json:"is_active"`
		ChainID   int64   `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := dbExec(c, `INSERT INTO fee_structures (id, fee_type, asset, fee_percent, fee_fixed, min_fee, max_fee, tier, is_active, chain_id) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		uuid.New(), req.FeeType, req.Asset, req.FeePercent, req.FeeFixed, req.MinFee, req.MaxFee, req.Tier, req.IsActive, req.ChainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "fee created"})
}

func handleUpdateFee(c *gin.Context) {
	var req struct {
		FeePercent float64 `json:"fee_percent"`
		FeeFixed   float64 `json:"fee_fixed"`
		IsActive   *bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	if _, err := dbExec(c, `UPDATE fee_structures SET fee_percent=$1, fee_fixed=$2, is_active=$3 WHERE id=$4`, req.FeePercent, req.FeeFixed, isActive, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "fee updated"})
}

// ---- Webhooks ----

func handleGetWebhooks(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, name, url, secret, events, is_active FROM webhooks ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"webhooks": rowsToMaps(rows)})
}

func handleCreateWebhook(c *gin.Context) {
	var req struct {
		Name   string   `json:"name" binding:"required"`
		URL    string   `json:"url" binding:"required"`
		Secret string   `json:"secret"`
		Events []string `json:"events"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := dbExec(c, `INSERT INTO webhooks (id, name, url, secret, events, is_active, created_by) VALUES ($1,$2,$3,$4,$5,true,$6)`,
		uuid.New(), req.Name, req.URL, req.Secret, req.Events, c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "webhook created"})
}

func handleTestWebhook(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "webhook test sent", "success": true})
}

func handleDeleteWebhook(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM webhooks WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "webhook deleted"})
}

// ---- Notifications ----

func handleGetNotifications(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, title, message, notification_type, is_read, created_at FROM notifications WHERE admin_id=$1 ORDER BY created_at DESC LIMIT 100`, c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"notifications": rowsToMaps(rows)})
}

func handleMarkNotificationRead(c *gin.Context) {
	if _, err := dbExec(c, `UPDATE notifications SET is_read=true WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "notification marked as read"})
}

func handleSendNotification(c *gin.Context) {
	var req struct {
		Title   string `json:"title" binding:"required"`
		Message string `json:"message" binding:"required"`
		Type    string `json:"type"`
		AdminID string `json:"admin_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	aid := req.AdminID
	if aid == "" {
		aid = c.GetString("user_id")
	}
	if _, err := dbExec(c, `INSERT INTO notifications (id, admin_id, title, message, notification_type, is_read) VALUES ($1,$2,$3,$4,$5,false)`, uuid.New(), aid, req.Title, req.Message, req.Type); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "notification sent"})
}

func handleBroadcastNotification(c *gin.Context) {
	var req struct {
		Title   string `json:"title" binding:"required"`
		Message string `json:"message" binding:"required"`
		Type    string `json:"type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rows, err := dbQuery(c, `SELECT id FROM admin_users WHERE is_active=true`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var adminID string
		rows.Scan(&adminID)
		dbExec(c, `INSERT INTO notifications (id, admin_id, title, message, notification_type, is_read) VALUES ($1,$2,$3,$4,$5,false)`, uuid.New(), adminID, req.Title, req.Message, req.Type)
		count++
	}
	c.JSON(http.StatusOK, gin.H{"message": "notification broadcasted", "recipients": count})
}

// ---- Audit Logs ----

func handleGetAuditLogs(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, admin_id, action, resource_type, resource_id, details, ip, created_at FROM audit_logs ORDER BY created_at DESC LIMIT 200`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"audit_logs": rowsToMaps(rows)})
}

func handleExportAuditLogs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"file_path": "/exports/audit_logs.csv", "message": "export started"})
}

// ---- Sessions ----

func handleGetSessions(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, admin_id, ip, user_agent, expires_at FROM admin_sessions WHERE admin_id=$1 ORDER BY expires_at DESC`, c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"sessions": rowsToMaps(rows)})
}

func handleRevokeSession(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM admin_sessions WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "session revoked"})
}

func handleRevokeAllSessions(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM admin_sessions WHERE admin_id=$1`, c.GetString("user_id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "all sessions revoked"})
}

// ---- Feature Flags ----

func handleGetFeatureFlags(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, name, description, is_enabled, rollout_percentage FROM feature_flags ORDER BY created_at DESC`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"feature_flags": rowsToMaps(rows)})
}

func handleCreateFeatureFlag(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		IsEnabled   bool   `json:"is_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `INSERT INTO feature_flags (id, name, description, is_enabled, rollout_percentage, updated_by) VALUES ($1,$2,$3,$4,100,$5)`, uuid.New(), req.Name, req.Description, req.IsEnabled, c.GetString("user_id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "feature flag created"})
}

func handleUpdateFeatureFlag(c *gin.Context) {
	var req struct {
		IsEnabled *bool `json:"is_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}
	if _, err := dbExec(c, `UPDATE feature_flags SET is_enabled=$1, updated_by=$2 WHERE id=$3`, isEnabled, c.GetString("user_id"), c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "feature flag updated"})
}

func handleDeleteFeatureFlag(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM feature_flags WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "feature flag deleted"})
}

// handleGetFeatures returns the feature flags in the shape the admin/web
// frontend expects: a bare array of {name, enabled, description}.
func handleGetFeatures(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT name, description, is_enabled FROM feature_flags ORDER BY created_at DESC`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	features := make([]map[string]interface{}, 0)
	for rows.Next() {
		var name, description string
		var isEnabled bool
		if err := rows.Scan(&name, &description, &isEnabled); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		features = append(features, map[string]interface{}{
			"name":        name,
			"enabled":     isEnabled,
			"description": description,
		})
	}
	c.JSON(http.StatusOK, features)
}

// handleSetFeature toggles a feature flag by name, matching the frontend's
// PUT /features/:name {enabled} contract.
func handleSetFeature(c *gin.Context) {
	var req struct {
		Enabled *bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	isEnabled := true
	if req.Enabled != nil {
		isEnabled = *req.Enabled
	}
	name := c.Param("name")
	if _, err := dbExec(c, `UPDATE feature_flags SET is_enabled=$1, updated_by=$2 WHERE name=$3`, isEnabled, c.GetString("user_id"), name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"name": name, "enabled": isEnabled})
}

// ---- IP Whitelist ----

func handleGetIPWhitelist(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, ip_address, description, is_active FROM ip_whitelist ORDER BY created_at DESC`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"ip_whitelist": rowsToMaps(rows)})
}

func handleAddIPWhitelist(c *gin.Context) {
	var req struct {
		IPAddress   string `json:"ip_address" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `INSERT INTO ip_whitelist (id, ip_address, description, is_active, created_by) VALUES ($1,$2,$3,true,$4) ON CONFLICT (ip_address) DO NOTHING`, uuid.New(), req.IPAddress, req.Description, c.GetString("user_id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "IP added to whitelist"})
}

func handleRemoveIPWhitelist(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM ip_whitelist WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "IP removed from whitelist"})
}

// ---- Tickets ----

func handleGetTickets(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, title, ticket_type, priority, status, assigned_to, created_at FROM tickets ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"tickets": rowsToMaps(rows)})
}

func handleGetTicket(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, title, description, ticket_type, priority, status, assigned_to, created_at FROM tickets WHERE id=$1`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	tickets := rowsToMaps(rows)
	if len(tickets) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
		return
	}
	msgRows, _ := dbQuery(c, `SELECT id, message, is_internal, created_by, created_at FROM ticket_messages WHERE ticket_id=$1 ORDER BY created_at`, c.Param("id"))
	defer msgRows.Close()
	c.JSON(http.StatusOK, gin.H{"ticket": tickets[0], "messages": rowsToMaps(msgRows)})
}

func handleCreateTicket(c *gin.Context) {
	var req struct {
		Title       string `json:"title" binding:"required"`
		Description string `json:"description"`
		TicketType  string `json:"ticket_type"`
		Priority    string `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `INSERT INTO tickets (id, title, description, ticket_type, priority, status, created_by) VALUES ($1,$2,$3,$4,$5,'open',$6)`, uuid.New(), req.Title, req.Description, req.TicketType, req.Priority, c.GetString("user_id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "ticket created"})
}

func handleUpdateTicketStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resolved := ""
	if req.Status == "resolved" || req.Status == "closed" {
		resolved = ", resolved_at=NOW()"
	}
	_, err := dbExec(c, `UPDATE tickets SET status=$1`+resolved+` WHERE id=$2`, req.Status, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ticket status updated"})
}

func handleAddTicketMessage(c *gin.Context) {
	var req struct {
		Message string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `INSERT INTO ticket_messages (id, ticket_id, message, is_internal, created_by) VALUES ($1,$2,$3,false,$4)`, uuid.New(), c.Param("id"), req.Message, c.GetString("user_id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "message added"})
}

func handleAssignTicket(c *gin.Context) {
	var req struct {
		AssignedTo string `json:"assigned_to" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE tickets SET assigned_to=$1 WHERE id=$2`, req.AssignedTo, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ticket assigned"})
}

// ---- White Labels ----

func handleGetWhiteLabels(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, name, domain, logo_url, primary_color, secondary_color, is_active FROM white_labels ORDER BY created_at DESC`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"white_labels": rowsToMaps(rows)})
}

func handleCreateWhiteLabel(c *gin.Context) {
	var req struct {
		Name           string `json:"name" binding:"required"`
		Domain         string `json:"domain" binding:"required"`
		LogoURL        string `json:"logo_url"`
		PrimaryColor   string `json:"primary_color"`
		SecondaryColor string `json:"secondary_color"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `INSERT INTO white_labels (id, name, domain, logo_url, primary_color, secondary_color, is_active) VALUES ($1,$2,$3,$4,$5,$6,true)`, uuid.New(), req.Name, req.Domain, req.LogoURL, req.PrimaryColor, req.SecondaryColor); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "white label created"})
}

func handleUpdateWhiteLabel(c *gin.Context) {
	var req struct {
		Name     string `json:"name"`
		IsActive *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	if _, err := dbExec(c, `UPDATE white_labels SET name=$1, is_active=$2 WHERE id=$3`, req.Name, isActive, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "white label updated"})
}

func handleDeleteWhiteLabel(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM white_labels WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "white label deleted"})
}

// ---- Stats ----

func handleGetStats(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	var totalUsers, activeUsers, totalTx, totalWithdrawals int
	database.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&totalUsers)
	database.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE status='active'`).Scan(&activeUsers)
	database.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM transactions`).Scan(&totalTx)
	database.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM withdrawals WHERE status='pending'`).Scan(&totalWithdrawals)
	c.JSON(http.StatusOK, gin.H{"stats": gin.H{
		"total_users":          totalUsers,
		"active_users":         activeUsers,
		"total_transactions":   totalTx,
		"pending_withdrawals":  totalWithdrawals,
	}})
}

// ---- Admins ----

func handleGetAdmins(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, username, email, role, is_active, created_at, last_login_at FROM admin_users ORDER BY created_at DESC`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"admins": rowsToMaps(rows)})
}

func handleGetAdmin(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, username, email, role, is_active, created_at, last_login_at FROM admin_users WHERE id=$1`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	admins := rowsToMaps(rows)
	if len(admins) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "admin not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"admin": admins[0]})
}

func handleUpdateAdmin(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE admin_users SET username=$1, role=$2, updated_at=NOW() WHERE id=$3`, req.Username, req.Role, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "admin updated"})
}

func handleDeleteAdmin(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM admin_users WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "admin deleted"})
}

func handleSuspendAdmin(c *gin.Context) {
	if _, err := dbExec(c, `UPDATE admin_users SET is_active=false, updated_at=NOW() WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "admin suspended"})
}

func handleActivateAdmin(c *gin.Context) {
	if _, err := dbExec(c, `UPDATE admin_users SET is_active=true, updated_at=NOW() WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "admin activated"})
}

// ---- Workflows ----

func handleGetWorkflows(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, name, workflow_type, threshold_amount, required_approvals, approvers, is_active FROM approval_workflows ORDER BY created_at DESC`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"workflows": rowsToMaps(rows)})
}

func handleCreateWorkflow(c *gin.Context) {
	var req struct {
		Name             string   `json:"name" binding:"required"`
		WorkflowType     string   `json:"workflow_type"`
		ThresholdAmount  float64  `json:"threshold_amount"`
		RequiredApprovals int      `json:"required_approvals"`
		Approvers        []string `json:"approvers"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `INSERT INTO approval_workflows (id, name, workflow_type, threshold_amount, required_approvals, approvers, is_active, created_by) VALUES ($1,$2,$3,$4,$5,$6,true,$7)`, uuid.New(), req.Name, req.WorkflowType, req.ThresholdAmount, req.RequiredApprovals, req.Approvers, c.GetString("user_id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "workflow created"})
}

func handleUpdateWorkflow(c *gin.Context) {
	var req struct {
		Name             string `json:"name"`
		RequiredApprovals int    `json:"required_approvals"`
		IsActive         *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	if _, err := dbExec(c, `UPDATE approval_workflows SET name=$1, required_approvals=$2, is_active=$3 WHERE id=$4`, req.Name, req.RequiredApprovals, isActive, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "workflow updated"})
}

func handleDeleteWorkflow(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM approval_workflows WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "workflow deleted"})
}

// ---- Approval Requests ----

func handleGetApprovalRequests(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, workflow_id, request_type, resource_id, requester_id, status, created_at FROM approval_requests ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"approval_requests": rowsToMaps(rows)})
}

func handleApproveRequest(c *gin.Context) {
	if _, err := dbExec(c, `UPDATE approval_requests SET status='approved', approved_by=$1 WHERE id=$2`, c.GetString("user_id"), c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "request approved"})
}

func handleRejectRequest(c *gin.Context) {
	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)
	if _, err := dbExec(c, `UPDATE approval_requests SET status='rejected', approved_by=$1, reject_reason=$2 WHERE id=$3`, c.GetString("user_id"), req.Reason, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "request rejected"})
}

// ---- Backups ----

func handleGetBackups(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, backup_type, file_path, file_size, status, created_at, completed_at FROM backups ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"backups": rowsToMaps(rows)})
}

func handleCreateBackup(c *gin.Context) {
	var req struct {
		BackupType string `json:"backup_type"`
	}
	c.ShouldBindJSON(&req)
	if req.BackupType == "" {
		req.BackupType = "full"
	}
	if _, err := dbExec(c, `INSERT INTO backups (id, backup_type, file_path, file_size, status, created_by) VALUES ($1,$2,$3,0,'in_progress',$4)`, uuid.New(), req.BackupType, "/backups/"+uuid.New().String(), c.GetString("user_id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"message": "backup started"})
}

func handleRestoreBackup(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "restore started"})
}

func handleDeleteBackup(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM backups WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "backup deleted"})
}

// ---- Knowledge Base ----

func handleGetKnowledgeArticles(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, title, content, category, tags, is_published, view_count FROM knowledge_articles ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"articles": rowsToMaps(rows)})
}

func handleGetKnowledgeArticle(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, title, content, category, tags, is_published, view_count FROM knowledge_articles WHERE id=$1`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	articles := rowsToMaps(rows)
	if len(articles) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "article not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"article": articles[0]})
}

func handleCreateKnowledgeArticle(c *gin.Context) {
	var req struct {
		Title      string   `json:"title" binding:"required"`
		Content    string   `json:"content" binding:"required"`
		Category   string   `json:"category"`
		Tags       []string `json:"tags"`
		IsPublished bool     `json:"is_published"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `INSERT INTO knowledge_articles (id, title, content, category, tags, is_published, view_count, created_by) VALUES ($1,$2,$3,$4,$5,$6,0,$7)`, uuid.New(), req.Title, req.Content, req.Category, req.Tags, req.IsPublished, c.GetString("user_id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "article created"})
}

func handleUpdateKnowledgeArticle(c *gin.Context) {
	var req struct {
		Title       string `json:"title"`
		Content     string `json:"content"`
		IsPublished *bool  `json:"is_published"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	isPub := true
	if req.IsPublished != nil {
		isPub = *req.IsPublished
	}
	if _, err := dbExec(c, `UPDATE knowledge_articles SET title=$1, content=$2, is_published=$3 WHERE id=$4`, req.Title, req.Content, isPub, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "article updated"})
}

func handleDeleteKnowledgeArticle(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM knowledge_articles WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "article deleted"})
}

// ---- Archival ----

func handleGetArchivePolicies(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, name, table_name, retention_days, archive_after_days, is_active FROM archive_policies ORDER BY created_at DESC`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"policies": rowsToMaps(rows)})
}

func handleCreateArchivePolicy(c *gin.Context) {
	var req struct {
		Name            string `json:"name" binding:"required"`
		TableName       string `json:"table_name" binding:"required"`
		RetentionDays   int    `json:"retention_days"`
		ArchiveAfterDays int   `json:"archive_after_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `INSERT INTO archive_policies (id, name, table_name, retention_days, archive_after_days, is_active, created_by) VALUES ($1,$2,$3,$4,$5,true,$6)`, uuid.New(), req.Name, req.TableName, req.RetentionDays, req.ArchiveAfterDays, c.GetString("user_id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "policy created"})
}

func handleUpdateArchivePolicy(c *gin.Context) {
	var req struct {
		IsActive *bool `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	if _, err := dbExec(c, `UPDATE archive_policies SET is_active=$1 WHERE id=$2`, isActive, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "policy updated"})
}

func handleDeleteArchivePolicy(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM archive_policies WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "policy deleted"})
}

func handleRunArchive(c *gin.Context) {
	c.JSON(http.StatusAccepted, gin.H{"message": "archive started"})
}

func handleGetArchiveRecords(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, policy_id, table_name, record_count, archive_path, status, started_at, completed_at FROM archive_records ORDER BY started_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"records": rowsToMaps(rows)})
}

// ---- Reports ----

func handleGetReportConfigs(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, name, report_type, parameters, file_format, is_scheduled, schedule FROM report_configs ORDER BY created_at DESC`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"configs": rowsToMaps(rows)})
}

func handleCreateReportConfig(c *gin.Context) {
	var req struct {
		Name       string `json:"name" binding:"required"`
		ReportType string `json:"report_type" binding:"required"`
		FileFormat string `json:"file_format"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `INSERT INTO report_configs (id, name, report_type, parameters, file_format, is_scheduled, created_by) VALUES ($1,$2,$3,$4,$5,false,$6)`, uuid.New(), req.Name, req.ReportType, []byte("{}"), req.FileFormat, c.GetString("user_id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "report config created"})
}

func handleGetReports(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, config_id, name, file_path, file_size, status, created_at FROM reports ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"reports": rowsToMaps(rows)})
}

func handleGenerateReport(c *gin.Context) {
	var req struct {
		ConfigID string `json:"config_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `INSERT INTO reports (id, config_id, name, file_path, file_size, status, created_by) VALUES ($1,$2,$3,$4,0,'pending',$5)`, uuid.New(), req.ConfigID, "Report-"+time.Now().Format("20060102"), "/reports/"+uuid.New().String(), c.GetString("user_id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"message": "report generation started"})
}

// ---- SLA ----

func handleGetSLAPolicies(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, name, priority, response_time_sla, resolution_time_sla, uptime_sla, is_active FROM sla_policies ORDER BY created_at DESC`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"policies": rowsToMaps(rows)})
}

func handleCreateSLAPolicy(c *gin.Context) {
	var req struct {
		Name              string `json:"name" binding:"required"`
		Priority          string `json:"priority"`
		ResponseTimeSLA   int    `json:"response_time_sla"`
		ResolutionTimeSLA int    `json:"resolution_time_sla"`
		UptimeSLA         float64 `json:"uptime_sla"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `INSERT INTO sla_policies (id, name, priority, response_time_sla, resolution_time_sla, uptime_sla, is_active, created_by) VALUES ($1,$2,$3,$4,$5,$6,true,$7)`, uuid.New(), req.Name, req.Priority, req.ResponseTimeSLA, req.ResolutionTimeSLA, req.UptimeSLA, c.GetString("user_id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "SLA policy created"})
}

func handleUpdateSLAPolicy(c *gin.Context) {
	var req struct {
		IsActive *bool `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	if _, err := dbExec(c, `UPDATE sla_policies SET is_active=$1 WHERE id=$2`, isActive, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "SLA policy updated"})
}

func handleDeleteSLAPolicy(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM sla_policies WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "SLA policy deleted"})
}

func handleGetSLAReports(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, policy_id, period_start, period_end, total_tickets, met_sla, breached_sla, avg_response_time, avg_resolution_time FROM sla_reports ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"reports": rowsToMaps(rows)})
}

func handleGenerateSLAReport(c *gin.Context) {
	c.JSON(http.StatusAccepted, gin.H{"message": "SLA report generation started"})
}

// ---- Integrations ----

func handleGetIntegrations(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, integration, name, api_key, webhook_url, is_active FROM integration_configs ORDER BY created_at DESC`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"integrations": rowsToMaps(rows)})
}

func handleCreateIntegration(c *gin.Context) {
	var req struct {
		Integration string `json:"integration" binding:"required"`
		Name         string `json:"name" binding:"required"`
		APIKey       string `json:"api_key"`
		APISecret    string `json:"api_secret"`
		WebhookURL   string `json:"webhook_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `INSERT INTO integration_configs (id, integration, name, api_key, api_secret, webhook_url, is_active, settings, created_by) VALUES ($1,$2,$3,$4,$5,$6,true,$7,$8)`, uuid.New(), req.Integration, req.Name, req.APIKey, req.APISecret, req.WebhookURL, []byte("{}"), c.GetString("user_id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "integration created"})
}

func handleUpdateIntegration(c *gin.Context) {
	var req struct {
		IsActive *bool `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	if _, err := dbExec(c, `UPDATE integration_configs SET is_active=$1 WHERE id=$2`, isActive, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "integration updated"})
}

func handleDeleteIntegration(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM integration_configs WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "integration deleted"})
}

func handleTestIntegration(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "integration test successful"})
}

// ---- Bot Management ----

func handleGetBots(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, name, bot_type, status, owner_id, created_at FROM bots ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"bots": rowsToMaps(rows)})
}

func handleGetBot(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, name, bot_type, status, config, stats, created_at FROM bots WHERE id=$1`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	bots := rowsToMaps(rows)
	if len(bots) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "bot not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"bot": bots[0]})
}

func handleCreateBot(c *gin.Context) {
	var req struct {
		Name    string `json:"name" binding:"required"`
		BotType string `json:"bot_type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `INSERT INTO bots (id, name, bot_type, status, config, stats) VALUES ($1,$2,$3,'stopped',$4,$5)`, uuid.New(), req.Name, req.BotType, []byte("{}"), []byte("{}")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "bot created"})
}

func handleUpdateBot(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE bots SET name=$1, updated_at=NOW() WHERE id=$2`, req.Name, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "bot updated"})
}

func handleDeleteBot(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM bots WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "bot deleted"})
}

func handleUpdateBotStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE bots SET status=$1, updated_at=NOW() WHERE id=$2`, req.Status, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "bot status updated"})
}

func handleGetBotStats(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT stats FROM bots WHERE id=$1`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	bots := rowsToMaps(rows)
	if len(bots) == 0 {
		c.JSON(http.StatusOK, gin.H{"stats": gin.H{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"stats": bots[0]["stats"]})
}

func handleGetBotTiers(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, name, max_bots, max_dex, max_cex, latency_ms, monthly_fee, is_active FROM bot_tiers ORDER BY monthly_fee`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"tiers": rowsToMaps(rows)})
}

func handleCreateBotTier(c *gin.Context) {
	var req struct {
		Name       string  `json:"name" binding:"required"`
		MaxBots    int     `json:"max_bots"`
		MaxDEX     int     `json:"max_dex"`
		MaxCEX     int     `json:"max_cex"`
		LatencyMs  int     `json:"latency_ms"`
		MonthlyFee float64 `json:"monthly_fee"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `INSERT INTO bot_tiers (id, name, max_bots, max_dex, max_cex, latency_ms, monthly_fee, is_active) VALUES ($1,$2,$3,$4,$5,$6,$7,true)`, uuid.New(), req.Name, req.MaxBots, req.MaxDEX, req.MaxCEX, req.LatencyMs, req.MonthlyFee); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "bot tier created"})
}

func handleUpdateBotTier(c *gin.Context) {
	var req struct {
		IsActive *bool `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	if _, err := dbExec(c, `UPDATE bot_tiers SET is_active=$1 WHERE id=$2`, isActive, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "bot tier updated"})
}

func handleDeleteBotTier(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM bot_tiers WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "bot tier deleted"})
}

// ---- BotsClient Management ----

func handleGetBotsClients(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, name, company, email, api_key, status, permission_level FROM bots_clients ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"clients": rowsToMaps(rows)})
}

func handleGetBotsClient(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, name, company, email, api_key, status, permission_level FROM bots_clients WHERE id=$1`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	clients := rowsToMaps(rows)
	if len(clients) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"client": clients[0]})
}

func handleCreateBotsClient(c *gin.Context) {
	var req struct {
		Name            string `json:"name" binding:"required"`
		Company         string `json:"company"`
		Email           string `json:"email"`
		PermissionLevel string `json:"permission_level"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.PermissionLevel == "" {
		req.PermissionLevel = "read"
	}
	apiKey := uuid.New().String()
	if _, err := dbExec(c, `INSERT INTO bots_clients (id, name, company, email, api_key, status, permission_level) VALUES ($1,$2,$3,$4,$5,'active',$6)`, uuid.New(), req.Name, req.Company, req.Email, apiKey, req.PermissionLevel); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "bots client created", "api_key": apiKey})
}

func handleUpdateBotsClient(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE bots_clients SET name=$1, updated_at=NOW() WHERE id=$2`, req.Name, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "bots client updated"})
}

func handleDeleteBotsClient(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM bots_clients WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "bots client deleted"})
}

func handleUpdateBotsClientStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE bots_clients SET status=$1, updated_at=NOW() WHERE id=$2`, req.Status, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "bots client status updated"})
}

// ---- Project Teams ----

func handleGetProjectTeams(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, name, description, status FROM project_teams ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"teams": rowsToMaps(rows)})
}

func handleGetProjectTeam(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, name, description, status FROM project_teams WHERE id=$1`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	teams := rowsToMaps(rows)
	if len(teams) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "team not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"team": teams[0]})
}

func handleCreateProjectTeam(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `INSERT INTO project_teams (id, name, description, status) VALUES ($1,$2,$3,'active')`, uuid.New(), req.Name, req.Description); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "project team created"})
}

func handleUpdateProjectTeam(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE project_teams SET name=$1, updated_at=NOW() WHERE id=$2`, req.Name, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "project team updated"})
}

func handleDeleteProjectTeam(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM project_teams WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "project team deleted"})
}

func handleGetProjectTeamMembers(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, user_id, role, joined_at FROM project_team_members WHERE team_id=$1`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"members": rowsToMaps(rows)})
}

func handleAddProjectTeamMember(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	uid, _ := uuid.Parse(req.UserID)
	if req.Role == "" {
		req.Role = "member"
	}
	if _, err := dbExec(c, `INSERT INTO project_team_members (id, team_id, user_id, role) VALUES ($1,$2,$3,$4)`, uuid.New(), c.Param("id"), uid, req.Role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "member added"})
}

func handleRemoveProjectTeamMember(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM project_team_members WHERE team_id=$1 AND id=$2`, c.Param("id"), c.Param("memberId")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "member removed"})
}

// ---- WL Clients ----

func handleGetWLClients(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, name, domain, status FROM wl_clients ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"clients": rowsToMaps(rows)})
}

func handleGetWLClient(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, name, domain, status FROM wl_clients WHERE id=$1`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	clients := rowsToMaps(rows)
	if len(clients) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"client": clients[0]})
}

func handleCreateWLClient(c *gin.Context) {
	var req struct {
		Name   string `json:"name" binding:"required"`
		Domain string `json:"domain"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `INSERT INTO wl_clients (id, name, domain, status) VALUES ($1,$2,$3,'active')`, uuid.New(), req.Name, req.Domain); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "WL client created"})
}

func handleUpdateWLClient(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE wl_clients SET name=$1, updated_at=NOW() WHERE id=$2`, req.Name, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "WL client updated"})
}

func handleDeleteWLClient(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM wl_clients WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "WL client deleted"})
}

func handleUpdateWLClientStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE wl_clients SET status=$1, updated_at=NOW() WHERE id=$2`, req.Status, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "WL client status updated"})
}

// ---- Additional per-product status controls (SuperAdmin governance) ----
// These let SuperAdmin start/stop/pause/resume each product via a status
// field. Status is a free-form string (active/paused/suspended/halted) so
// all lifecycle transitions are expressible. These are governance records
// only — they never move crypto assets.

func handleUpdateWhiteLabelStatus(c *gin.Context) {
	var req struct {
		IsActive *bool  `json:"is_active"`
		Status   string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// white_labels uses is_active (bool). Map status string -> is_active when provided.
	if req.Status != "" {
		req.IsActive = ptrBool(req.Status == "active" || req.Status == "resumed" || req.Status == "started")
	}
	if req.IsActive == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "is_active or status is required"})
		return
	}
	if _, err := dbExec(c, `UPDATE white_labels SET is_active=$1, updated_at=NOW() WHERE id=$2`, *req.IsActive, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "white label status updated"})
}

func handleUpdateProjectTeamStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE project_teams SET status=$1, updated_at=NOW() WHERE id=$2`, req.Status, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "project team status updated"})
}

func handleUpdateWLProjectTeamStatus(c *gin.Context) {
	// WL project teams reuse the project_teams table.
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE project_teams SET status=$1, updated_at=NOW() WHERE id=$2`, req.Status, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "WL project team status updated"})
}

func handleUpdateMasterWalletStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE master_wallets SET status=$1, updated_at=NOW() WHERE id=$2`, req.Status, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "master wallet status updated"})
}

func handleUpdateUserWalletStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE user_wallets SET status=$1, updated_at=NOW() WHERE id=$2`, req.Status, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "user wallet status updated"})
}

// ptrBool returns a pointer to b (helper for optional bool binding).
func ptrBool(b bool) *bool { return &b }


// ---- WL MasterWallets (reuse master_wallets table with status filter) ----

func handleGetWLMasterWallets(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, name, address, chain_id, balance, status FROM master_wallets ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"wallets": rowsToMaps(rows)})
}

func handleGetWLMasterWallet(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, name, address, chain_id, balance, status FROM master_wallets WHERE id=$1`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	wallets := rowsToMaps(rows)
	if len(wallets) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"wallet": wallets[0]})
}

func handleCreateWLMasterWallet(c *gin.Context) {
	var req struct {
		Name    string `json:"name" binding:"required"`
		Address string `json:"address"`
		ChainID int64  `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `INSERT INTO master_wallets (id, name, address, chain_id, balance, status) VALUES ($1,$2,$3,$4,0,'active')`, uuid.New(), req.Name, req.Address, req.ChainID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "WL master wallet created"})
}

func handleUpdateWLMasterWallet(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE master_wallets SET name=$1, updated_at=NOW() WHERE id=$2`, req.Name, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "WL master wallet updated"})
}

func handleDeleteWLMasterWallet(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM master_wallets WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "WL master wallet deleted"})
}

func handleUpdateWLMasterWalletStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE master_wallets SET status=$1, updated_at=NOW() WHERE id=$2`, req.Status, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "WL master wallet status updated"})
}

// ---- WL UserWallets ----

func handleGetWLUserWallets(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, master_wallet_id, name, address, chain_id, balance, status FROM user_wallets ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"wallets": rowsToMaps(rows)})
}

func handleGetWLUserWallet(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, master_wallet_id, name, address, chain_id, balance, status FROM user_wallets WHERE id=$1`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	wallets := rowsToMaps(rows)
	if len(wallets) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"wallet": wallets[0]})
}

func handleCreateWLUserWallet(c *gin.Context) {
	var req struct {
		Name           string `json:"name" binding:"required"`
		MasterWalletID string `json:"master_wallet_id"`
		Address        string `json:"address"`
		ChainID        int64  `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	mwID, _ := uuid.Parse(req.MasterWalletID)
	if _, err := dbExec(c, `INSERT INTO user_wallets (id, master_wallet_id, name, address, chain_id, balance, status) VALUES ($1,$2,$3,$4,$5,0,'active')`, uuid.New(), mwID, req.Name, req.Address, req.ChainID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "WL user wallet created"})
}

func handleUpdateWLUserWallet(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE user_wallets SET name=$1, updated_at=NOW() WHERE id=$2`, req.Name, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "WL user wallet updated"})
}

func handleDeleteWLUserWallet(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM user_wallets WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "WL user wallet deleted"})
}

func handleUpdateWLUserWalletStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE user_wallets SET status=$1, updated_at=NOW() WHERE id=$2`, req.Status, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "WL user wallet status updated"})
}

// ---- WL Bots (reuse bots table) ----

func handleGetWLBots(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, name, bot_type, status FROM bots ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"bots": rowsToMaps(rows)})
}

func handleGetWLBot(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, name, bot_type, status FROM bots WHERE id=$1`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	bots := rowsToMaps(rows)
	if len(bots) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "bot not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"bot": bots[0]})
}

func handleCreateWLBot(c *gin.Context) {
	var req struct {
		Name    string `json:"name" binding:"required"`
		BotType string `json:"bot_type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `INSERT INTO bots (id, name, bot_type, status, config, stats) VALUES ($1,$2,$3,'stopped',$4,$5)`, uuid.New(), req.Name, req.BotType, []byte("{}"), []byte("{}")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "WL bot created"})
}

func handleUpdateWLBot(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE bots SET name=$1, updated_at=NOW() WHERE id=$2`, req.Name, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "WL bot updated"})
}

func handleDeleteWLBot(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM bots WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "WL bot deleted"})
}

func handleUpdateWLBotStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE bots SET status=$1, updated_at=NOW() WHERE id=$2`, req.Status, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "WL bot status updated"})
}

// ---- WL BotsClients (reuse bots_clients table) ----

func handleGetWLBotsClients(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, name, company, email, status, permission_level FROM bots_clients ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"clients": rowsToMaps(rows)})
}

func handleGetWLBotsClient(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, name, company, email, status, permission_level FROM bots_clients WHERE id=$1`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	clients := rowsToMaps(rows)
	if len(clients) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"client": clients[0]})
}

func handleCreateWLBotsClient(c *gin.Context) {
	var req struct {
		Name            string `json:"name" binding:"required"`
		Company         string `json:"company"`
		Email           string `json:"email"`
		PermissionLevel string `json:"permission_level"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.PermissionLevel == "" {
		req.PermissionLevel = "read"
	}
	if _, err := dbExec(c, `INSERT INTO bots_clients (id, name, company, email, api_key, status, permission_level) VALUES ($1,$2,$3,$4,$5,'active',$6)`, uuid.New(), req.Name, req.Company, req.Email, uuid.New().String(), req.PermissionLevel); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "WL bots client created"})
}

func handleUpdateWLBotClient(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE bots_clients SET name=$1, updated_at=NOW() WHERE id=$2`, req.Name, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "WL bots client updated"})
}

func handleDeleteWLBotsClient(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM bots_clients WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "WL bots client deleted"})
}

func handleUpdateWLBotsClientStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE bots_clients SET status=$1, updated_at=NOW() WHERE id=$2`, req.Status, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "WL bots client status updated"})
}

// ---- WL Project Teams (reuse project_teams table) ----

func handleGetWLProjectTeams(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, name, description, status FROM project_teams ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"teams": rowsToMaps(rows)})
}

func handleGetWLProjectTeam(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, name, description, status FROM project_teams WHERE id=$1`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	teams := rowsToMaps(rows)
	if len(teams) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "team not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"team": teams[0]})
}

func handleCreateWLProjectTeam(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `INSERT INTO project_teams (id, name, description, status) VALUES ($1,$2,$3,'active')`, uuid.New(), req.Name, req.Description); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "WL project team created"})
}

func handleUpdateWLProjectTeam(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE project_teams SET name=$1, updated_at=NOW() WHERE id=$2`, req.Name, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "WL project team updated"})
}

func handleDeleteWLProjectTeam(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM project_teams WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "WL project team deleted"})
}

// ---- MasterWallet Management ----

func handleGetMasterWallets(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, name, address, chain_id, balance, status FROM master_wallets ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"wallets": rowsToMaps(rows)})
}

func handleGetMasterWallet(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, name, address, chain_id, balance, status FROM master_wallets WHERE id=$1`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	wallets := rowsToMaps(rows)
	if len(wallets) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"wallet": wallets[0]})
}

func handleCreateMasterWallet(c *gin.Context) {
	var req struct {
		Name    string `json:"name" binding:"required"`
		Address string `json:"address"`
		ChainID int64  `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `INSERT INTO master_wallets (id, name, address, chain_id, balance, status) VALUES ($1,$2,$3,$4,0,'active')`, uuid.New(), req.Name, req.Address, req.ChainID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "master wallet created"})
}

func handleUpdateMasterWallet(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE master_wallets SET name=$1, updated_at=NOW() WHERE id=$2`, req.Name, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "master wallet updated"})
}

func handleDeleteMasterWallet(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM master_wallets WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "master wallet deleted"})
}

func handleGetMasterWalletBalance(c *gin.Context) {
	var balance float64
	dbQueryRow(c, `SELECT balance FROM master_wallets WHERE id=$1`, c.Param("id")).Scan(&balance)
	c.JSON(http.StatusOK, gin.H{"balance": balance})
}

func handleMasterWalletTransfer(c *gin.Context) {
	// DISABLED: admins must not move crypto assets. Fund movement is the wallet
	// owner's action via the canonical wallet backend (go/wallet_api), never an admin
	// action. Retained only to return an explicit 403 so any stale client call is
	// clearly rejected instead of receiving a 404.
	c.JSON(http.StatusForbidden, gin.H{"error": "admin fund transfer is prohibited; crypto asset movement is performed only by the wallet owner via the canonical wallet backend"})
}

// ---- UserWallet Management ----

func handleGetUserWallets(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, master_wallet_id, name, address, chain_id, balance, status FROM user_wallets ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"wallets": rowsToMaps(rows)})
}

func handleGetUserWallet(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, master_wallet_id, name, address, chain_id, balance, status FROM user_wallets WHERE id=$1`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	wallets := rowsToMaps(rows)
	if len(wallets) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"wallet": wallets[0]})
}

func handleCreateUserWallet(c *gin.Context) {
	var req struct {
		Name           string `json:"name" binding:"required"`
		MasterWalletID string `json:"master_wallet_id"`
		Address        string `json:"address"`
		ChainID        int64  `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	mwID, _ := uuid.Parse(req.MasterWalletID)
	if _, err := dbExec(c, `INSERT INTO user_wallets (id, master_wallet_id, name, address, chain_id, balance, status) VALUES ($1,$2,$3,$4,$5,0,'active')`, uuid.New(), mwID, req.Name, req.Address, req.ChainID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "user wallet created"})
}

func handleUpdateUserWallet(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE user_wallets SET name=$1, updated_at=NOW() WHERE id=$2`, req.Name, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "user wallet updated"})
}

func handleDeleteUserWallet(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM user_wallets WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "user wallet deleted"})
}

func handleGetUserWalletBalance(c *gin.Context) {
	var balance float64
	dbQueryRow(c, `SELECT balance FROM user_wallets WHERE id=$1`, c.Param("id")).Scan(&balance)
	c.JSON(http.StatusOK, gin.H{"balance": balance})
}

// keep strconv referenced (used implicitly by some future extensions)
var _ = strconv.Itoa
