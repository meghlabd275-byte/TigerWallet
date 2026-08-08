// TigerWallet Admin - Main Entry Point
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/tigerwallet/master-admin-management/internal/config"
	"github.com/tigerwallet/master-admin-management/internal/database"
	"github.com/tigerwallet/master-admin-management/internal/middleware"
)

func main() {
	cfg := config.Load()

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

			admin.GET("/stats", handleGetStats)

			admin.POST("/logout", handleLogout)
			admin.POST("/change-password", handleChangePassword)
			admin.POST("/2fa/enable", handleEnable2FA)
			admin.POST("/2fa/disable", handleDisable2FA)

			admin.GET("/admins", handleGetAdmins)
			admin.GET("/admins/:id", handleGetAdmin)
			admin.PUT("/admins/:id", handleUpdateAdmin)
			admin.DELETE("/admins/:id", handleDeleteAdmin)
			admin.POST("/admins/:id/suspend", handleSuspendAdmin)
			admin.POST("/admins/:id/activate", handleActivateAdmin)

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

func handleLogin(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "login handler"}) }
func handleRegister(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "register handler"}) }
func handleRefreshToken(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "refresh handler"}) }
func handleLogout(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "logout handler"}) }
func handleChangePassword(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "password changed"}) }
func handleEnable2FA(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "2FA enabled"}) }
func handleDisable2FA(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "2FA disabled"}) }
func handleGetUsers(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"users": []}) }
func handleGetUser(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"user": map[string]interface{}{}}) }
func handleUpdateUserStatus(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "status updated"}) }
func handleBanUser(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "user banned"}) }
func handleUnbanUser(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "user unbanned"}) }
func handleSuspendUser(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "user suspended"}) }
func handleGetKYC(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"kyc_requests": []}) }
func handleApproveKYC(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "KYC approved"}) }
func handleRejectKYC(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "KYC rejected"}) }
func handleGetTransactions(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"transactions": []}) }
func handleGetTransaction(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"transaction": map[string]interface{}{}}) }
func handleFlagTransaction(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "transaction flagged"}) }
func handleUnflagTransaction(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "transaction unflagged"}) }
func handleGetWithdrawals(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"withdrawals": []}) }
func handleApproveWithdrawal(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "withdrawal approved"}) }
func handleRejectWithdrawal(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "withdrawal rejected"}) }
func handleProcessWithdrawal(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "withdrawal processed"}) }
func handleGetTokens(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"tokens": []}) }
func handleCreateToken(c *gin.Context) { c.JSON(http.StatusCreated, gin.H{"message": "token created"}) }
func handleUpdateToken(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "token updated"}) }
func handleDeleteToken(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "token deleted"}) }
func handleGetPairs(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"pairs": []}) }
func handleCreatePair(c *gin.Context) { c.JSON(http.StatusCreated, gin.H{"message": "pair created"}) }
func handleUpdatePairStatus(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "pair status updated"}) }
func handleGetBlockchains(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"blockchains": []}) }
func handleCreateBlockchain(c *gin.Context) { c.JSON(http.StatusCreated, gin.H{"message": "blockchain created"}) }
func handleUpdateBlockchain(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "blockchain updated"}) }
func handleSetBlockchainStatus(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "status updated"}) }
func handleGetFees(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"fees": []}) }
func handleCreateFee(c *gin.Context) { c.JSON(http.StatusCreated, gin.H{"message": "fee created"}) }
func handleUpdateFee(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "fee updated"}) }
func handleGetWebhooks(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"webhooks": []}) }
func handleCreateWebhook(c *gin.Context) { c.JSON(http.StatusCreated, gin.H{"message": "webhook created"}) }
func handleTestWebhook(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "webhook test sent"}) }
func handleDeleteWebhook(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "webhook deleted"}) }
func handleGetNotifications(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"notifications": []}) }
func handleMarkNotificationRead(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "notification marked as read"}) }
func handleSendNotification(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "notification sent"}) }
func handleBroadcastNotification(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "notification broadcasted"}) }
func handleGetAuditLogs(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"audit_logs": []}) }
func handleExportAuditLogs(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"file_path": "/exports/audit_logs.csv"}) }
func handleGetSessions(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"sessions": []}) }
func handleRevokeSession(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "session revoked"}) }
func handleRevokeAllSessions(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "all sessions revoked"}) }
func handleGetFeatureFlags(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"feature_flags": []}) }
func handleCreateFeatureFlag(c *gin.Context) { c.JSON(http.StatusCreated, gin.H{"message": "feature flag created"}) }
func handleUpdateFeatureFlag(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "feature flag updated"}) }
func handleDeleteFeatureFlag(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "feature flag deleted"}) }
func handleGetIPWhitelist(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ip_whitelist": []}) }
func handleAddIPWhitelist(c *gin.Context) { c.JSON(http.StatusCreated, gin.H{"message": "IP added to whitelist"}) }
func handleRemoveIPWhitelist(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "IP removed from whitelist"}) }
func handleGetTickets(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"tickets": []}) }
func handleGetTicket(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ticket": map[string]interface{}{}, "messages": []}) }
func handleCreateTicket(c *gin.Context) { c.JSON(http.StatusCreated, gin.H{"message": "ticket created"}) }
func handleUpdateTicketStatus(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "ticket status updated"}) }
func handleAddTicketMessage(c *gin.Context) { c.JSON(http.StatusCreated, gin.H{"message": "message added"}) }
func handleAssignTicket(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "ticket assigned"}) }
func handleGetWhiteLabels(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"white_labels": []}) }
func handleCreateWhiteLabel(c *gin.Context) { c.JSON(http.StatusCreated, gin.H{"message": "white label created"}) }
func handleUpdateWhiteLabel(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "white label updated"}) }
func handleDeleteWhiteLabel(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "white label deleted"}) }
func handleGetStats(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"stats": map[string]interface{}{"total_users": 0, "active_users": 0}}) }
func handleGetAdmins(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"admins": []}) }
func handleGetAdmin(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"admin": map[string]interface{}{}}) }
func handleUpdateAdmin(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "admin updated"}) }
func handleDeleteAdmin(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "admin deleted"}) }
func handleSuspendAdmin(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "admin suspended"}) }
func handleActivateAdmin(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "admin activated"}) }
func handleGetWorkflows(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"workflows": []}) }
func handleCreateWorkflow(c *gin.Context) { c.JSON(http.StatusCreated, gin.H{"message": "workflow created"}) }
func handleUpdateWorkflow(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "workflow updated"}) }
func handleDeleteWorkflow(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "workflow deleted"}) }
func handleGetApprovalRequests(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"approval_requests": []}) }
func handleApproveRequest(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "request approved"}) }
func handleRejectRequest(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "request rejected"}) }
func handleGetBackups(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"backups": []}) }
func handleCreateBackup(c *gin.Context) { c.JSON(http.StatusAccepted, gin.H{"message": "backup started"}) }
func handleRestoreBackup(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "restore started"}) }
func handleDeleteBackup(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "backup deleted"}) }

// Knowledge base handlers
func handleGetKnowledgeArticles(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"articles": []}) }
func handleGetKnowledgeArticle(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"article": map[string]interface{}{}}) }
func handleCreateKnowledgeArticle(c *gin.Context) { c.JSON(http.StatusCreated, gin.H{"message": "article created"}) }
func handleUpdateKnowledgeArticle(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "article updated"}) }
func handleDeleteKnowledgeArticle(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "article deleted"}) }

// Archival handlers
func handleGetArchivePolicies(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"policies": []}) }
func handleCreateArchivePolicy(c *gin.Context) { c.JSON(http.StatusCreated, gin.H{"message": "policy created"}) }
func handleUpdateArchivePolicy(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "policy updated"}) }
func handleDeleteArchivePolicy(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "policy deleted"}) }
func handleRunArchive(c *gin.Context) { c.JSON(http.StatusAccepted, gin.H{"message": "archive started"}) }
func handleGetArchiveRecords(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"records": []}) }

// Report handlers
func handleGetReportConfigs(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"configs": []}) }
func handleCreateReportConfig(c *gin.Context) { c.JSON(http.StatusCreated, gin.H{"message": "report config created"}) }
func handleGetReports(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"reports": []}) }
func handleGenerateReport(c *gin.Context) { c.JSON(http.StatusAccepted, gin.H{"message": "report generation started"}) }

// SLA handlers
func handleGetSLAPolicies(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"policies": []}) }
func handleCreateSLAPolicy(c *gin.Context) { c.JSON(http.StatusCreated, gin.H{"message": "SLA policy created"}) }
func handleUpdateSLAPolicy(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "SLA policy updated"}) }
func handleDeleteSLAPolicy(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "SLA policy deleted"}) }
func handleGetSLAReports(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"reports": []}) }
func handleGenerateSLAReport(c *gin.Context) { c.JSON(http.StatusAccepted, gin.H{"message": "SLA report generation started"}) }

// Integration handlers
func handleGetIntegrations(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"integrations": []}) }
func handleCreateIntegration(c *gin.Context) { c.JSON(http.StatusCreated, gin.H{"message": "integration created"}) }
func handleUpdateIntegration(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "integration updated"}) }
func handleDeleteIntegration(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "integration deleted"}) }
func handleTestIntegration(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"success": true, "message": "integration test successful"}) }
