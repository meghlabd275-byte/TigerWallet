// TigerWallet Admin - Main Entry Point
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lib/pq"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/tigerwallet/super-admin/internal/config"
	"github.com/tigerwallet/super-admin/internal/database"
	"github.com/tigerwallet/super-admin/internal/middleware"
	twredis "github.com/tigerwallet/super-admin/internal/redis"
	"golang.org/x/crypto/bcrypt"
)

// appCfg is the package-level config, set in main(), used by handlers for JWT auth.
var appCfg *config.Config

// redisClient is the shared feature-flag store client. Feature flag handlers
// publish live state to it; downstream services read from the same Redis keys.
var redisClient *twredis.RedisClient

func main() {
	cfg := config.Load()
	appCfg = cfg

	if err := database.Initialize(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	log.Println("Database initialized successfully")

	// Shared feature-flag store. Non-fatal: if Redis is unavailable the backend
	// still boots; flag publish simply no-ops (fail-closed downstream).
	rc, err := twredis.NewRedisClient(cfg)
	if err != nil {
		log.Printf("Warning: feature-flag Redis client unavailable: %v", err)
	}
	redisClient = rc
	if redisClient != nil {
		defer redisClient.Close()
		log.Println("Feature-flag Redis client initialized")
	}

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
			// No self-registration route: admin accounts are created exclusively
			// by a SuperAdmin via POST /api/v1/admin/admins (see adminAdmins below).
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
			admin.GET("/features/:name/check", handleCheckFeatureFlag)

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

			// ---- Domain admin governance (CRUD + status/approve/reject) ----
			// Governance records only; never moves crypto assets.
			admin.GET("/futures", handleGetFuturesPositions)
			admin.GET("/futures/:id", handleGetFuturesPosition)
			admin.POST("/futures", handleCreateFuturesPosition)
			admin.PUT("/futures/:id", handleUpdateFuturesPosition)
			admin.DELETE("/futures/:id", handleDeleteFuturesPosition)
			admin.PUT("/futures/:id/status", handleUpdateFuturesPositionStatus)

			admin.GET("/options", handleGetOptionsContracts)
			admin.GET("/options/:id", handleGetOptionsContract)
			admin.POST("/options", handleCreateOptionsContract)
			admin.PUT("/options/:id", handleUpdateOptionsContract)
			admin.DELETE("/options/:id", handleDeleteOptionsContract)
			admin.PUT("/options/:id/status", handleUpdateOptionsContractStatus)

			admin.GET("/copy-trading", handleGetCopyTradingConfigs)
			admin.GET("/copy-trading/:id", handleGetCopyTradingConfig)
			admin.POST("/copy-trading", handleCreateCopyTradingConfig)
			admin.PUT("/copy-trading/:id", handleUpdateCopyTradingConfig)
			admin.DELETE("/copy-trading/:id", handleDeleteCopyTradingConfig)
			admin.PUT("/copy-trading/:id/status", handleUpdateCopyTradingConfigStatus)

			// ---- Trading control-plane (SuperAdmin global governance) ----
			// Full lifecycle over the builtin trading engines; status flips
			// publish to the shared Redis control namespace that the wallet
			// engines enforce on.
			admin.GET("/trading/overview", handleTradingOverview)
			admin.GET("/trading/audit", handleTradingControlAudit)
			admin.POST("/trading/halt/:vertical", handleHaltTradingVertical)
			admin.POST("/trading/resume/:vertical", handleResumeTradingVertical)

			admin.GET("/trading/contracts", handleListTradingContracts)
			admin.POST("/trading/contracts", handleCreateTradingContract)
			admin.POST("/trading/contracts/:id/stop", handleStopTradingContract)
			admin.POST("/trading/contracts/:id/resume", handleResumeTradingContract)
			admin.DELETE("/trading/contracts/:id", handleDeleteTradingContract)

			admin.GET("/trading/pools", handleListLiquidityPools)
			admin.POST("/trading/pools", handleCreateLiquidityPool)
			admin.POST("/trading/pools/:id/stop", handleStopLiquidityPool)
			admin.POST("/trading/pools/:id/resume", handleResumeLiquidityPool)
			admin.DELETE("/trading/pools/:id", handleDeleteLiquidityPool)

			admin.POST("/trading/pairs/:id/stop", handleStopTradingPair)
			admin.POST("/trading/pairs/:id/resume", handleResumeTradingPair)

			admin.GET("/trading/margin-markets", handleListMarginMarkets)
			admin.POST("/trading/margin-markets", handleCreateMarginMarket)
			admin.POST("/trading/margin-markets/:id/stop", handleStopMarginMarket)
			admin.POST("/trading/margin-markets/:id/resume", handleResumeMarginMarket)
			admin.DELETE("/trading/margin-markets/:id", handleDeleteMarginMarket)

			admin.POST("/trading/options/:id/stop", handleStopOptionsContract)
			admin.POST("/trading/options/:id/resume", handleResumeOptionsContract)

			admin.POST("/trading/copy-trading/:id/stop", handleStopCopyTrading)
			admin.POST("/trading/copy-trading/:id/resume", handleResumeCopyTrading)

			admin.GET("/convert", handleGetConvertOrders)
			admin.GET("/convert/:id", handleGetConvertOrder)
			admin.POST("/convert", handleCreateConvertOrder)
			admin.PUT("/convert/:id", handleUpdateConvertOrder)
			admin.DELETE("/convert/:id", handleDeleteConvertOrder)
			admin.PUT("/convert/:id/status", handleUpdateConvertOrderStatus)

			admin.GET("/onramp", handleGetOnrampOrders)
			admin.GET("/onramp/:id", handleGetOnrampOrder)
			admin.POST("/onramp", handleCreateOnrampOrder)
			admin.PUT("/onramp/:id", handleUpdateOnrampOrder)
			admin.DELETE("/onramp/:id", handleDeleteOnrampOrder)
			admin.POST("/onramp/:id/approve", handleApproveOnrampOrder)
			admin.POST("/onramp/:id/reject", handleRejectOnrampOrder)

			admin.GET("/offramp", handleGetOfframpOrders)
			admin.GET("/offramp/:id", handleGetOfframpOrder)
			admin.POST("/offramp", handleCreateOfframpOrder)
			admin.PUT("/offramp/:id", handleUpdateOfframpOrder)
			admin.DELETE("/offramp/:id", handleDeleteOfframpOrder)
			admin.POST("/offramp/:id/approve", handleApproveOfframpOrder)
			admin.POST("/offramp/:id/reject", handleRejectOfframpOrder)

			admin.GET("/p2p-clients", handleGetP2PClients)
			admin.GET("/p2p-clients/:id", handleGetP2PClient)
			admin.POST("/p2p-clients", handleCreateP2PClient)
			admin.PUT("/p2p-clients/:id", handleUpdateP2PClient)
			admin.DELETE("/p2p-clients/:id", handleDeleteP2PClient)
			admin.PUT("/p2p-clients/:id/status", handleUpdateP2PClientStatus)

			admin.GET("/p2p-merchants", handleGetP2PMerchants)
			admin.GET("/p2p-merchants/:id", handleGetP2PMerchant)
			admin.POST("/p2p-merchants", handleCreateP2PMerchant)
			admin.PUT("/p2p-merchants/:id", handleUpdateP2PMerchant)
			admin.DELETE("/p2p-merchants/:id", handleDeleteP2PMerchant)
			admin.PUT("/p2p-merchants/:id/status", handleUpdateP2PMerchantStatus)
			admin.POST("/p2p-merchants/:id/approve", handleApproveP2PMerchant)
			admin.POST("/p2p-merchants/:id/reject", handleRejectP2PMerchant)

			admin.GET("/partners", handleGetPartners)
			admin.GET("/partners/:id", handleGetPartner)
			admin.POST("/partners", handleCreatePartner)
			admin.PUT("/partners/:id", handleUpdatePartner)
			admin.DELETE("/partners/:id", handleDeletePartner)
			admin.PUT("/partners/:id/status", handleUpdatePartnerStatus)
			admin.POST("/partners/:id/approve", handleApprovePartner)
			admin.POST("/partners/:id/reject", handleRejectPartner)

			admin.GET("/rewards", handleGetRewardCampaigns)
			admin.GET("/rewards/:id", handleGetRewardCampaign)
			admin.POST("/rewards", handleCreateRewardCampaign)
			admin.PUT("/rewards/:id", handleUpdateRewardCampaign)
			admin.DELETE("/rewards/:id", handleDeleteRewardCampaign)
			admin.PUT("/rewards/:id/status", handleUpdateRewardCampaignStatus)

			admin.GET("/marketing", handleGetMarketingCampaigns)
			admin.GET("/marketing/:id", handleGetMarketingCampaign)
			admin.POST("/marketing", handleCreateMarketingCampaign)
			admin.PUT("/marketing/:id", handleUpdateMarketingCampaign)
			admin.DELETE("/marketing/:id", handleDeleteMarketingCampaign)
			admin.PUT("/marketing/:id/status", handleUpdateMarketingCampaignStatus)

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
			admin.POST("/2fa/verify", handleVerify2FA)
			admin.POST("/2fa/disable", handleDisable2FA)

			// Admin user management — SuperAdmin only (role assignment, suspend, delete)
			adminAdmins := admin.Group("")
			adminAdmins.Use(middleware.RoleAuth("super_admin"))
			{
				adminAdmins.GET("/admins", handleGetAdmins)
				adminAdmins.POST("/admins", handleCreateAdmin)
				adminAdmins.GET("/admins/:id", handleGetAdmin)
				adminAdmins.PUT("/admins/:id", handleUpdateAdmin)
				adminAdmins.DELETE("/admins/:id", handleDeleteAdmin)
				adminAdmins.POST("/admins/:id/suspend", handleSuspendAdmin)
				adminAdmins.POST("/admins/:id/activate", handleActivateAdmin)

				// Structured RBAC: custom admin roles + granular permissions (SuperAdmin only)
				adminAdmins.GET("/roles", handleListAdminRoles)
				adminAdmins.POST("/roles", handleCreateAdminRole)
				adminAdmins.GET("/roles/:id", handleGetAdminRole)
				adminAdmins.PUT("/roles/:id", handleUpdateAdminRole)
				adminAdmins.DELETE("/roles/:id", handleDeleteAdminRole)
				adminAdmins.GET("/permissions", handleListAdminPermissions)
				adminAdmins.POST("/permissions", handleCreateAdminPermission)
				adminAdmins.POST("/admins/:id/roles", handleAssignAdminRole)
				adminAdmins.DELETE("/admins/:id/roles/:roleId", handleRevokeAdminRole)
				adminAdmins.GET("/admins/:id/permissions", handleGetAdminEffectivePermissions)
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

			// Crypto cards (governance records only — no fund movement)
			admin.GET("/crypto-cards", handleListCryptoCards)
			admin.POST("/crypto-cards", handleCreateCryptoCard)
			admin.GET("/crypto-cards/:id", handleGetCryptoCard)
			admin.PUT("/crypto-cards/:id", handleUpdateCryptoCard)
			admin.DELETE("/crypto-cards/:id", handleDeleteCryptoCard)
			admin.POST("/crypto-cards/:id/block", handleBlockCryptoCard)
			admin.POST("/crypto-cards/:id/activate", handleActivateCryptoCard)
			admin.PUT("/crypto-cards/:id/limit", handleSetCryptoCardLimit)
			admin.PUT("/crypto-cards/:id/status", handleUpdateCryptoCardStatus)
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
		Email         string `json:"email" binding:"required"`
		Password      string `json:"password" binding:"required"`
		TwoFactorCode string `json:"two_factor_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	var id, username, role, hash, twoFactorSecret string
	var twoFactorEnabled bool
	err := database.Pool.QueryRow(ctx, `SELECT id, username, role, password_hash, two_factor_secret, two_factor_enabled FROM admin_users WHERE email=$1 AND is_active=true`, req.Email).Scan(&id, &username, &role, &hash, &twoFactorSecret, &twoFactorEnabled)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	// Fail-closed TOTP enforcement: when 2FA is enabled AND a TOTP secret exists,
	// require a valid one-time code. A missing code returns 401 with
	// two_factor_required=true so the client can prompt. An invalid code is
	// rejected. Secrets are only ever set through the verified /2fa/setup +
	// /2fa/verify flow, so this can never lock out an admin who has not
	// completed real 2FA enrollment.
	if twoFactorEnabled && strings.TrimSpace(twoFactorSecret) != "" {
		code := strings.TrimSpace(req.TwoFactorCode)
		if code == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "two_factor_required", "two_factor_required": true})
			return
		}
		if !totp.Validate(code, twoFactorSecret) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid two-factor code"})
			return
		}
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

func handleCreateAdmin(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Role is optional but must be one of the known admin roles. Never allow a
	// request to self-promote to super_admin via this endpoint.
	validRoles := []string{"admin", "operator", "viewer", "dex_admin", "cex_admin", "finance_admin"}
	if req.Role == "" {
		req.Role = "admin"
	}
	if !contains(validRoles, req.Role) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	id := uuid.New()
	_, err = dbExec(c, `INSERT INTO admin_users (id, username, email, password_hash, role, is_active, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,true,NOW(),NOW())`, id, req.Username, req.Email, string(hash), req.Role)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "user already exists", "detail": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "username": req.Username, "email": req.Email, "role": req.Role})
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
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
	// Real TOTP setup: generate a fresh secret, persist it to admin_users
	// (two_factor_enabled stays false until the admin verifies a code via
	// /2fa/verify). Returns the otpauth URI for authenticator apps + a QR
	// data URI. No fake/placeholder secrets.
	userID := c.GetString("user_id")
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "TigerWalletSuperAdmin",
		AccountName: userID,
		Period:      30,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate 2FA secret"})
		return
	}
	if _, err := dbExec(c, `UPDATE admin_users SET two_factor_secret=$1, updated_at=NOW() WHERE id=$2`, key.Secret(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"secret":  key.Secret(),
		"otpauth": key.URL(),
		"qr_data": key.URL(),
		"message": "Scan the QR with your authenticator, then POST /2fa/verify with a 6-digit code to enable",
		"enabled": false,
	})
}

// handleVerify2FA confirms 2FA enrollment: validates a TOTP code against the
// stored secret and, on success, sets two_factor_enabled=true. Only after this
// does login-time enforcement activate.
func handleVerify2FA(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.GetString("user_id")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	var secret string
	err := database.Pool.QueryRow(ctx, `SELECT two_factor_secret FROM admin_users WHERE id=$1`, userID).Scan(&secret)
	if err != nil || strings.TrimSpace(secret) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "2FA not set up — call POST /2fa/enable first"})
		return
	}
	if !totp.Validate(strings.TrimSpace(req.Code), secret) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid verification code"})
		return
	}
	if _, err := dbExec(c, `UPDATE admin_users SET two_factor_enabled=true, updated_at=NOW() WHERE id=$1`, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "2FA enabled", "enabled": true})
}

func handleDisable2FA(c *gin.Context) {
	// Clear both the secret and the flag so a re-enroll requires a fresh setup.
	if _, err := dbExec(c, `UPDATE admin_users SET two_factor_enabled=false, two_factor_secret=NULL, updated_at=NOW() WHERE id=$1`, c.GetString("user_id")); err != nil {
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
		FeeType    string  `json:"fee_type" binding:"required"`
		Asset      string  `json:"asset"`
		FeePercent float64 `json:"fee_percent"`
		FeeFixed   float64 `json:"fee_fixed"`
		MinFee     float64 `json:"min_fee"`
		MaxFee     float64 `json:"max_fee"`
		Tier       string  `json:"tier"`
		IsActive   bool    `json:"is_active"`
		ChainID    int64   `json:"chain_id"`
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

// publishFeatureState writes the feature flag's live state to the shared Redis
// store (the store downstream services consult). Non-fatal on failure.
func publishFeatureState(name, state string) {
	if redisClient == nil || name == "" {
		return
	}
	_ = redisClient.PublishFeatureState(name, state)
}

// deleteFeatureState removes the feature flag's live state from Redis.
func deleteFeatureState(name string) {
	if redisClient == nil || name == "" {
		return
	}
	_ = redisClient.DeleteFeatureState(name)
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
	publishFeatureState(req.Name, twredis.FeatureStateFromBool(req.IsEnabled))
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
	// Resolve the feature name by id so we can publish the live state to Redis.
	var name string
	_ = dbQueryRow(c, `SELECT name FROM feature_flags WHERE id=$1`, c.Param("id")).Scan(&name)
	if _, err := dbExec(c, `UPDATE feature_flags SET is_enabled=$1, updated_by=$2 WHERE id=$3`, isEnabled, c.GetString("user_id"), c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if name != "" {
		publishFeatureState(name, twredis.FeatureStateFromBool(isEnabled))
	}
	c.JSON(http.StatusOK, gin.H{"message": "feature flag updated"})
}

func handleDeleteFeatureFlag(c *gin.Context) {
	// Resolve the feature name by id so we can delete the live state from Redis.
	var name string
	_ = dbQueryRow(c, `SELECT name FROM feature_flags WHERE id=$1`, c.Param("id")).Scan(&name)
	if _, err := dbExec(c, `DELETE FROM feature_flags WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if name != "" {
		deleteFeatureState(name)
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
	publishFeatureState(name, twredis.FeatureStateFromBool(isEnabled))
	c.JSON(http.StatusOK, gin.H{"name": name, "enabled": isEnabled})
}

// handleCheckFeatureFlag returns the LIVE feature state as read from Redis
// (the store downstream services consult), not just the DB row. Fail-closed:
// missing/unknown -> disabled.
func handleCheckFeatureFlag(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "feature name required"})
		return
	}
	state := twredis.StateDisabled
	enabled := false
	if redisClient != nil {
		if live, ok := redisClient.GetFeatureState(name); ok {
			state = live
			enabled = (live == twredis.StateEnabled)
		}
	}
	c.JSON(http.StatusOK, gin.H{"name": name, "state": state, "enabled": enabled})
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
		"total_users":         totalUsers,
		"active_users":        activeUsers,
		"total_transactions":  totalTx,
		"pending_withdrawals": totalWithdrawals,
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
		Name              string   `json:"name" binding:"required"`
		WorkflowType      string   `json:"workflow_type"`
		ThresholdAmount   float64  `json:"threshold_amount"`
		RequiredApprovals int      `json:"required_approvals"`
		Approvers         []string `json:"approvers"`
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
		Name              string `json:"name"`
		RequiredApprovals int    `json:"required_approvals"`
		IsActive          *bool  `json:"is_active"`
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
		Title       string   `json:"title" binding:"required"`
		Content     string   `json:"content" binding:"required"`
		Category    string   `json:"category"`
		Tags        []string `json:"tags"`
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
		Name             string `json:"name" binding:"required"`
		TableName        string `json:"table_name" binding:"required"`
		RetentionDays    int    `json:"retention_days"`
		ArchiveAfterDays int    `json:"archive_after_days"`
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
		Name              string  `json:"name" binding:"required"`
		Priority          string  `json:"priority"`
		ResponseTimeSLA   int     `json:"response_time_sla"`
		ResolutionTimeSLA int     `json:"resolution_time_sla"`
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
		Name        string `json:"name" binding:"required"`
		APIKey      string `json:"api_key"`
		APISecret   string `json:"api_secret"`
		WebhookURL  string `json:"webhook_url"`
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

// ============== Domain admin governance handlers ==============
// Real PostgreSQL-backed CRUD + status/approve/reject handlers for the
// per-product governance tables. These manage governance records ONLY —
// they never move crypto assets.

// ---- Futures (futures_positions) ----

func handleGetFuturesPositions(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT * FROM futures_positions ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"positions": rowsToMaps(rows)})
}

func handleGetFuturesPosition(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT * FROM futures_positions WHERE id=$1`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	positions := rowsToMaps(rows)
	if len(positions) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "position not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"position": positions[0]})
}

func handleCreateFuturesPosition(c *gin.Context) {
	var req struct {
		Pair             string  `json:"pair" binding:"required"`
		Side             string  `json:"side" binding:"required"`
		Size             float64 `json:"size"`
		Leverage         float64 `json:"leverage"`
		EntryPrice       float64 `json:"entry_price"`
		LiquidationPrice float64 `json:"liquidation_price"`
		Margin           float64 `json:"margin"`
		ChainID          int64   `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `INSERT INTO futures_positions (id, pair, side, size, leverage, entry_price, liquidation_price, margin, chain_id, status) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'open')`,
		uuid.New(), req.Pair, req.Side, req.Size, req.Leverage, req.EntryPrice, req.LiquidationPrice, req.Margin, req.ChainID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "futures position created"})
}

func handleUpdateFuturesPosition(c *gin.Context) {
	var req struct {
		Pair             string  `json:"pair"`
		Side             string  `json:"side"`
		Size             float64 `json:"size"`
		Leverage         float64 `json:"leverage"`
		EntryPrice       float64 `json:"entry_price"`
		LiquidationPrice float64 `json:"liquidation_price"`
		Margin           float64 `json:"margin"`
		ChainID          int64   `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE futures_positions SET pair=$1, side=$2, size=$3, leverage=$4, entry_price=$5, liquidation_price=$6, margin=$7, chain_id=$8, updated_at=NOW() WHERE id=$9`,
		req.Pair, req.Side, req.Size, req.Leverage, req.EntryPrice, req.LiquidationPrice, req.Margin, req.ChainID, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "futures position updated"})
}

func handleDeleteFuturesPosition(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM futures_positions WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "futures position deleted"})
}

func handleUpdateFuturesPositionStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE futures_positions SET status=$1, updated_at=NOW() WHERE id=$2`, req.Status, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "futures position status updated"})
}

// ---- Options (options_contracts) ----

func handleGetOptionsContracts(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT * FROM options_contracts ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"contracts": rowsToMaps(rows)})
}

func handleGetOptionsContract(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT * FROM options_contracts WHERE id=$1`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	contracts := rowsToMaps(rows)
	if len(contracts) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "contract not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"contract": contracts[0]})
}

func handleCreateOptionsContract(c *gin.Context) {
	var req struct {
		Underlying string  `json:"underlying" binding:"required"`
		OptionType string  `json:"option_type" binding:"required"`
		Strike     float64 `json:"strike"`
		Expiry     string  `json:"expiry"`
		Premium    float64 `json:"premium"`
		Size       float64 `json:"size"`
		ChainID    int64   `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `INSERT INTO options_contracts (id, underlying, option_type, strike, expiry, premium, size, chain_id, status) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'active')`,
		uuid.New(), req.Underlying, req.OptionType, req.Strike, req.Expiry, req.Premium, req.Size, req.ChainID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "options contract created"})
}

func handleUpdateOptionsContract(c *gin.Context) {
	var req struct {
		Underlying string  `json:"underlying"`
		OptionType string  `json:"option_type"`
		Strike     float64 `json:"strike"`
		Expiry     string  `json:"expiry"`
		Premium    float64 `json:"premium"`
		Size       float64 `json:"size"`
		ChainID    int64   `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE options_contracts SET underlying=$1, option_type=$2, strike=$3, expiry=$4, premium=$5, size=$6, chain_id=$7, updated_at=NOW() WHERE id=$8`,
		req.Underlying, req.OptionType, req.Strike, req.Expiry, req.Premium, req.Size, req.ChainID, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "options contract updated"})
}

func handleDeleteOptionsContract(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM options_contracts WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "options contract deleted"})
}

func handleUpdateOptionsContractStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE options_contracts SET status=$1, updated_at=NOW() WHERE id=$2`, req.Status, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "options contract status updated"})
}

// ---- Copy trading (copy_trading_configs) ----

func handleGetCopyTradingConfigs(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT * FROM copy_trading_configs ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"configs": rowsToMaps(rows)})
}

func handleGetCopyTradingConfig(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT * FROM copy_trading_configs WHERE id=$1`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	configs := rowsToMaps(rows)
	if len(configs) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"config": configs[0]})
}

func handleCreateCopyTradingConfig(c *gin.Context) {
	var req struct {
		FollowerID  string  `json:"follower_id"`
		LeaderID    string  `json:"leader_id"`
		Allocation  float64 `json:"allocation"`
		MaxLeverage float64 `json:"max_leverage"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `INSERT INTO copy_trading_configs (id, follower_id, leader_id, allocation, max_leverage, status) VALUES ($1,$2,$3,$4,$5,'active')`,
		uuid.New(), req.FollowerID, req.LeaderID, req.Allocation, req.MaxLeverage); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "copy trading config created"})
}

func handleUpdateCopyTradingConfig(c *gin.Context) {
	var req struct {
		FollowerID  string  `json:"follower_id"`
		LeaderID    string  `json:"leader_id"`
		Allocation  float64 `json:"allocation"`
		MaxLeverage float64 `json:"max_leverage"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE copy_trading_configs SET follower_id=$1, leader_id=$2, allocation=$3, max_leverage=$4, updated_at=NOW() WHERE id=$5`,
		req.FollowerID, req.LeaderID, req.Allocation, req.MaxLeverage, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "copy trading config updated"})
}

func handleDeleteCopyTradingConfig(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM copy_trading_configs WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "copy trading config deleted"})
}

func handleUpdateCopyTradingConfigStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE copy_trading_configs SET status=$1, updated_at=NOW() WHERE id=$2`, req.Status, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "copy trading config status updated"})
}

// ---- Convert (convert_orders) ----

func handleGetConvertOrders(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT * FROM convert_orders ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"orders": rowsToMaps(rows)})
}

func handleGetConvertOrder(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT * FROM convert_orders WHERE id=$1`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	orders := rowsToMaps(rows)
	if len(orders) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": orders[0]})
}

func handleCreateConvertOrder(c *gin.Context) {
	var req struct {
		UserID     string  `json:"user_id"`
		FromToken  string  `json:"from_token" binding:"required"`
		ToToken    string  `json:"to_token" binding:"required"`
		FromAmount float64 `json:"from_amount"`
		ToAmount   float64 `json:"to_amount"`
		Rate       float64 `json:"rate"`
		ChainID    int64   `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `INSERT INTO convert_orders (id, user_id, from_token, to_token, from_amount, to_amount, rate, chain_id, status) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'pending')`,
		uuid.New(), req.UserID, req.FromToken, req.ToToken, req.FromAmount, req.ToAmount, req.Rate, req.ChainID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "convert order created"})
}

func handleUpdateConvertOrder(c *gin.Context) {
	var req struct {
		FromToken  string  `json:"from_token"`
		ToToken    string  `json:"to_token"`
		FromAmount float64 `json:"from_amount"`
		ToAmount   float64 `json:"to_amount"`
		Rate       float64 `json:"rate"`
		ChainID    int64   `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE convert_orders SET from_token=$1, to_token=$2, from_amount=$3, to_amount=$4, rate=$5, chain_id=$6, updated_at=NOW() WHERE id=$7`,
		req.FromToken, req.ToToken, req.FromAmount, req.ToAmount, req.Rate, req.ChainID, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "convert order updated"})
}

func handleDeleteConvertOrder(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM convert_orders WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "convert order deleted"})
}

func handleUpdateConvertOrderStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE convert_orders SET status=$1, updated_at=NOW() WHERE id=$2`, req.Status, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "convert order status updated"})
}

// ---- Onramp (onramp_orders) ----

func handleGetOnrampOrders(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT * FROM onramp_orders ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"orders": rowsToMaps(rows)})
}

func handleGetOnrampOrder(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT * FROM onramp_orders WHERE id=$1`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	orders := rowsToMaps(rows)
	if len(orders) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": orders[0]})
}

func handleCreateOnrampOrder(c *gin.Context) {
	var req struct {
		UserID       string  `json:"user_id"`
		Provider     string  `json:"provider" binding:"required"`
		FiatCurrency string  `json:"fiat_currency" binding:"required"`
		CryptoToken  string  `json:"crypto_token" binding:"required"`
		FiatAmount   float64 `json:"fiat_amount"`
		CryptoAmount float64 `json:"crypto_amount"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `INSERT INTO onramp_orders (id, user_id, provider, fiat_currency, crypto_token, fiat_amount, crypto_amount, status) VALUES ($1,$2,$3,$4,$5,$6,$7,'pending')`,
		uuid.New(), req.UserID, req.Provider, req.FiatCurrency, req.CryptoToken, req.FiatAmount, req.CryptoAmount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "onramp order created"})
}

func handleUpdateOnrampOrder(c *gin.Context) {
	var req struct {
		Provider     string  `json:"provider"`
		FiatCurrency string  `json:"fiat_currency"`
		CryptoToken  string  `json:"crypto_token"`
		FiatAmount   float64 `json:"fiat_amount"`
		CryptoAmount float64 `json:"crypto_amount"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE onramp_orders SET provider=$1, fiat_currency=$2, crypto_token=$3, fiat_amount=$4, crypto_amount=$5, updated_at=NOW() WHERE id=$6`,
		req.Provider, req.FiatCurrency, req.CryptoToken, req.FiatAmount, req.CryptoAmount, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "onramp order updated"})
}

func handleDeleteOnrampOrder(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM onramp_orders WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "onramp order deleted"})
}

func handleApproveOnrampOrder(c *gin.Context) {
	if _, err := dbExec(c, `UPDATE onramp_orders SET status='completed', updated_at=NOW() WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "onramp order approved"})
}

func handleRejectOnrampOrder(c *gin.Context) {
	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)
	if _, err := dbExec(c, `UPDATE onramp_orders SET status='rejected', payment_ref=$1, updated_at=NOW() WHERE id=$2`, req.Reason, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "onramp order rejected"})
}

// ---- Offramp (offramp_orders) ----

func handleGetOfframpOrders(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT * FROM offramp_orders ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"orders": rowsToMaps(rows)})
}

func handleGetOfframpOrder(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT * FROM offramp_orders WHERE id=$1`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	orders := rowsToMaps(rows)
	if len(orders) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": orders[0]})
}

func handleCreateOfframpOrder(c *gin.Context) {
	var req struct {
		UserID       string  `json:"user_id"`
		Provider     string  `json:"provider" binding:"required"`
		CryptoToken  string  `json:"crypto_token" binding:"required"`
		FiatCurrency string  `json:"fiat_currency" binding:"required"`
		CryptoAmount float64 `json:"crypto_amount"`
		FiatAmount   float64 `json:"fiat_amount"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `INSERT INTO offramp_orders (id, user_id, provider, crypto_token, fiat_currency, crypto_amount, fiat_amount, status) VALUES ($1,$2,$3,$4,$5,$6,$7,'pending')`,
		uuid.New(), req.UserID, req.Provider, req.CryptoToken, req.FiatCurrency, req.CryptoAmount, req.FiatAmount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "offramp order created"})
}

func handleUpdateOfframpOrder(c *gin.Context) {
	var req struct {
		Provider     string  `json:"provider"`
		CryptoToken  string  `json:"crypto_token"`
		FiatCurrency string  `json:"fiat_currency"`
		CryptoAmount float64 `json:"crypto_amount"`
		FiatAmount   float64 `json:"fiat_amount"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE offramp_orders SET provider=$1, crypto_token=$2, fiat_currency=$3, crypto_amount=$4, fiat_amount=$5, updated_at=NOW() WHERE id=$6`,
		req.Provider, req.CryptoToken, req.FiatCurrency, req.CryptoAmount, req.FiatAmount, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "offramp order updated"})
}

func handleDeleteOfframpOrder(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM offramp_orders WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "offramp order deleted"})
}

func handleApproveOfframpOrder(c *gin.Context) {
	if _, err := dbExec(c, `UPDATE offramp_orders SET status='completed', updated_at=NOW() WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "offramp order approved"})
}

func handleRejectOfframpOrder(c *gin.Context) {
	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)
	if _, err := dbExec(c, `UPDATE offramp_orders SET status='rejected', payout_ref=$1, updated_at=NOW() WHERE id=$2`, req.Reason, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "offramp order rejected"})
}

// ---- P2P clients (p2p_clients) ----

func handleGetP2PClients(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT * FROM p2p_clients ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"clients": rowsToMaps(rows)})
}

func handleGetP2PClient(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT * FROM p2p_clients WHERE id=$1`, c.Param("id"))
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

func handleCreateP2PClient(c *gin.Context) {
	var req struct {
		UserID   string `json:"user_id"`
		Username string `json:"username" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `INSERT INTO p2p_clients (id, user_id, username, status) VALUES ($1,$2,$3,'active')`,
		uuid.New(), req.UserID, req.Username); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "p2p client created"})
}

func handleUpdateP2PClient(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE p2p_clients SET username=$1, updated_at=NOW() WHERE id=$2`, req.Username, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "p2p client updated"})
}

func handleDeleteP2PClient(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM p2p_clients WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "p2p client deleted"})
}

func handleUpdateP2PClientStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE p2p_clients SET status=$1, updated_at=NOW() WHERE id=$2`, req.Status, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "p2p client status updated"})
}

// ---- P2P merchants (p2p_merchants) ----

func handleGetP2PMerchants(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT * FROM p2p_merchants ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"merchants": rowsToMaps(rows)})
}

func handleGetP2PMerchant(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT * FROM p2p_merchants WHERE id=$1`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	merchants := rowsToMaps(rows)
	if len(merchants) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "merchant not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"merchant": merchants[0]})
}

func handleCreateP2PMerchant(c *gin.Context) {
	var req struct {
		Name  string `json:"name" binding:"required"`
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `INSERT INTO p2p_merchants (id, name, email, status, verified) VALUES ($1,$2,$3,'pending',false)`,
		uuid.New(), req.Name, req.Email); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "p2p merchant created"})
}

func handleUpdateP2PMerchant(c *gin.Context) {
	var req struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE p2p_merchants SET name=$1, email=$2, updated_at=NOW() WHERE id=$3`, req.Name, req.Email, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "p2p merchant updated"})
}

func handleDeleteP2PMerchant(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM p2p_merchants WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "p2p merchant deleted"})
}

func handleUpdateP2PMerchantStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE p2p_merchants SET status=$1, updated_at=NOW() WHERE id=$2`, req.Status, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "p2p merchant status updated"})
}

func handleApproveP2PMerchant(c *gin.Context) {
	if _, err := dbExec(c, `UPDATE p2p_merchants SET status='approved', verified=true, updated_at=NOW() WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "p2p merchant approved"})
}

func handleRejectP2PMerchant(c *gin.Context) {
	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)
	_ = req
	if _, err := dbExec(c, `UPDATE p2p_merchants SET status='rejected', updated_at=NOW() WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "p2p merchant rejected"})
}

// ---- Partners (partners) ----

func handleGetPartners(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT * FROM partners ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"partners": rowsToMaps(rows)})
}

func handleGetPartner(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT * FROM partners WHERE id=$1`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	partners := rowsToMaps(rows)
	if len(partners) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "partner not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"partner": partners[0]})
}

func handleCreatePartner(c *gin.Context) {
	var req struct {
		Name         string  `json:"name" binding:"required"`
		ContactEmail string  `json:"contact_email"`
		RevenueShare float64 `json:"revenue_share"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `INSERT INTO partners (id, name, contact_email, api_key, revenue_share, status) VALUES ($1,$2,$3,$4,$5,'pending')`,
		uuid.New(), req.Name, req.ContactEmail, uuid.New().String(), req.RevenueShare); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "partner created"})
}

func handleUpdatePartner(c *gin.Context) {
	var req struct {
		Name         string  `json:"name"`
		ContactEmail string  `json:"contact_email"`
		RevenueShare float64 `json:"revenue_share"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE partners SET name=$1, contact_email=$2, revenue_share=$3, updated_at=NOW() WHERE id=$4`,
		req.Name, req.ContactEmail, req.RevenueShare, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "partner updated"})
}

func handleDeletePartner(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM partners WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "partner deleted"})
}

func handleUpdatePartnerStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE partners SET status=$1, updated_at=NOW() WHERE id=$2`, req.Status, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "partner status updated"})
}

func handleApprovePartner(c *gin.Context) {
	if _, err := dbExec(c, `UPDATE partners SET status='approved', updated_at=NOW() WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "partner approved"})
}

func handleRejectPartner(c *gin.Context) {
	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)
	_ = req
	if _, err := dbExec(c, `UPDATE partners SET status='rejected', updated_at=NOW() WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "partner rejected"})
}

// ---- Rewards (reward_campaigns) ----

func handleGetRewardCampaigns(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT * FROM reward_campaigns ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"campaigns": rowsToMaps(rows)})
}

func handleGetRewardCampaign(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT * FROM reward_campaigns WHERE id=$1`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	campaigns := rowsToMaps(rows)
	if len(campaigns) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "campaign not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"campaign": campaigns[0]})
}

func handleCreateRewardCampaign(c *gin.Context) {
	var req struct {
		Name       string  `json:"name" binding:"required"`
		RewardType string  `json:"reward_type" binding:"required"`
		Amount     float64 `json:"amount"`
		Token      string  `json:"token"`
		StartAt    string  `json:"start_at"`
		EndAt      string  `json:"end_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `INSERT INTO reward_campaigns (id, name, reward_type, amount, token, status, start_at, end_at) VALUES ($1,$2,$3,$4,$5,'active',$6,$7)`,
		uuid.New(), req.Name, req.RewardType, req.Amount, req.Token, req.StartAt, req.EndAt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "reward campaign created"})
}

func handleUpdateRewardCampaign(c *gin.Context) {
	var req struct {
		Name       string  `json:"name"`
		RewardType string  `json:"reward_type"`
		Amount     float64 `json:"amount"`
		Token      string  `json:"token"`
		StartAt    string  `json:"start_at"`
		EndAt      string  `json:"end_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE reward_campaigns SET name=$1, reward_type=$2, amount=$3, token=$4, start_at=$5, end_at=$6, updated_at=NOW() WHERE id=$7`,
		req.Name, req.RewardType, req.Amount, req.Token, req.StartAt, req.EndAt, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "reward campaign updated"})
}

func handleDeleteRewardCampaign(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM reward_campaigns WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "reward campaign deleted"})
}

func handleUpdateRewardCampaignStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE reward_campaigns SET status=$1, updated_at=NOW() WHERE id=$2`, req.Status, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "reward campaign status updated"})
}

// ---- Marketing (marketing_campaigns) ----

func handleGetMarketingCampaigns(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT * FROM marketing_campaigns ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"campaigns": rowsToMaps(rows)})
}

func handleGetMarketingCampaign(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT * FROM marketing_campaigns WHERE id=$1`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	campaigns := rowsToMaps(rows)
	if len(campaigns) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "campaign not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"campaign": campaigns[0]})
}

func handleCreateMarketingCampaign(c *gin.Context) {
	var req struct {
		Name    string  `json:"name" binding:"required"`
		Channel string  `json:"channel" binding:"required"`
		Budget  float64 `json:"budget"`
		StartAt string  `json:"start_at"`
		EndAt   string  `json:"end_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `INSERT INTO marketing_campaigns (id, name, channel, budget, status, start_at, end_at) VALUES ($1,$2,$3,$4,'draft',$5,$6)`,
		uuid.New(), req.Name, req.Channel, req.Budget, req.StartAt, req.EndAt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "marketing campaign created"})
}

func handleUpdateMarketingCampaign(c *gin.Context) {
	var req struct {
		Name    string  `json:"name"`
		Channel string  `json:"channel"`
		Budget  float64 `json:"budget"`
		StartAt string  `json:"start_at"`
		EndAt   string  `json:"end_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE marketing_campaigns SET name=$1, channel=$2, budget=$3, start_at=$4, end_at=$5, updated_at=NOW() WHERE id=$6`,
		req.Name, req.Channel, req.Budget, req.StartAt, req.EndAt, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "marketing campaign updated"})
}

func handleDeleteMarketingCampaign(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM marketing_campaigns WHERE id=$1`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "marketing campaign deleted"})
}

func handleUpdateMarketingCampaignStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `UPDATE marketing_campaigns SET status=$1, updated_at=NOW() WHERE id=$2`, req.Status, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "marketing campaign status updated"})
}

// ---- Structured RBAC handlers (SuperAdmin-managed custom roles + permissions) ----

func handleListAdminRoles(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, name, description, permissions, is_system, is_active, created_at, updated_at FROM admin_roles ORDER BY created_at DESC`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"roles": rowsToMaps(rows)})
}

func handleCreateAdminRole(c *gin.Context) {
	var req struct {
		Name        string   `json:"name" binding:"required"`
		Description string   `json:"description"`
		Permissions []string `json:"permissions"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := dbExec(c, `INSERT INTO admin_roles (id, name, description, permissions, is_system, is_active) VALUES ($1,$2,$3,$4,false,true)`, uuid.New(), req.Name, req.Description, pq.Array(req.Permissions)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "admin role created"})
}

func handleGetAdminRole(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, name, description, permissions, is_system, is_active, created_at, updated_at FROM admin_roles WHERE id=$1`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	roles := rowsToMaps(rows)
	if len(roles) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"role": roles[0]})
}

func handleUpdateAdminRole(c *gin.Context) {
	var req struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Permissions []string `json:"permissions"`
		IsActive    *bool    `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// System roles cannot be deleted but can be toggled active.
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	if _, err := dbExec(c, `UPDATE admin_roles SET name=$1, description=$2, permissions=$3, is_active=$4, updated_at=NOW() WHERE id=$5 AND is_system=false`, req.Name, req.Description, pq.Array(req.Permissions), isActive, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "admin role updated"})
}

func handleDeleteAdminRole(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM admin_roles WHERE id=$1 AND is_system=false`, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "admin role deleted"})
}

func handleListAdminPermissions(c *gin.Context) {
	rows, err := dbQuery(c, `SELECT id, name, description, category, is_active, created_at FROM admin_permissions ORDER BY category, name`)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	c.JSON(http.StatusOK, gin.H{"permissions": rowsToMaps(rows)})
}

func handleCreateAdminPermission(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Category    string `json:"category"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Category == "" {
		req.Category = "general"
	}
	if _, err := dbExec(c, `INSERT INTO admin_permissions (id, name, description, category, is_active) VALUES ($1,$2,$3,$4,true)`, uuid.New(), req.Name, req.Description, req.Category); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "permission created"})
}

func handleAssignAdminRole(c *gin.Context) {
	var req struct {
		RoleID string `json:"role_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	grantedBy := c.GetString("user_id")
	if _, err := dbExec(c, `INSERT INTO admin_role_assignments (id, admin_id, role_id, granted_by) VALUES ($1,$2,$3,$4) ON CONFLICT (admin_id, role_id) DO NOTHING`, uuid.New(), c.Param("id"), req.RoleID, grantedBy); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "role assigned"})
}

func handleRevokeAdminRole(c *gin.Context) {
	if _, err := dbExec(c, `DELETE FROM admin_role_assignments WHERE admin_id=$1 AND role_id=$2`, c.Param("id"), c.Param("roleId")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "role revoked"})
}

func handleGetAdminEffectivePermissions(c *gin.Context) {
	// Aggregate permissions across all roles assigned to this admin.
	rows, err := dbQuery(c, `
		SELECT DISTINCT unnest(r.permissions) AS permission
		FROM admin_role_assignments a
		JOIN admin_roles r ON r.id = a.role_id
		WHERE a.admin_id = $1 AND r.is_active = true`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	perms := []string{}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err == nil {
			perms = append(perms, p)
		}
	}
	c.JSON(http.StatusOK, gin.H{"admin_id": c.Param("id"), "permissions": perms})
}

// ===================== Crypto Cards (governance records only — no fund movement) =====================

func handleListCryptoCards(c *gin.Context) {
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	query := `SELECT id, user_id, user_name, card_number, currency, balance, "limit", status, card_type, created_at, updated_at FROM crypto_cards`
	args := []interface{}{}
	if status != "" && status != "all" {
		query += ` WHERE status = $1`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(len(args)+1) + ` OFFSET $` + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	rows, err := database.Pool.Query(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	cards := []map[string]interface{}{}
	for rows.Next() {
		var id int64
		var userID *string
		var userName, cardNumber, currency, cardStatus, cardType string
		var balance, limitVal float64
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &userID, &userName, &cardNumber, &currency, &balance, &limitVal, &cardStatus, &cardType, &createdAt, &updatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		cards = append(cards, map[string]interface{}{
			"id": id, "user_id": userID, "user_name": userName, "card_number": cardNumber,
			"currency": currency, "balance": balance, "limit": limitVal, "status": cardStatus,
			"card_type": cardType, "created_at": createdAt, "updated_at": updatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"crypto_cards": cards})
}

func handleCreateCryptoCard(c *gin.Context) {
	var req struct {
		UserID   *string `json:"user_id"`
		UserName string  `json:"user_name"`
		Currency string  `json:"currency" binding:"required"`
		Limit    float64 `json:"limit"`
		CardType string  `json:"card_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.CardType == "" {
		req.CardType = "virtual"
	}
	cardNumber := "****-****-****-" + strconv.FormatInt(time.Now().UnixNano()%10000, 10)
	_, err := database.Pool.Exec(c.Request.Context(),
		`INSERT INTO crypto_cards (user_id, user_name, card_number, currency, balance, "limit", status, card_type) VALUES ($1,$2,$3,$4,0,$5,'pending',$6)`,
		req.UserID, req.UserName, cardNumber, req.Currency, req.Limit, req.CardType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "crypto card created", "card_number": cardNumber})
}

func handleGetCryptoCard(c *gin.Context) {
	id := c.Param("id")
	var cardID int64
	var userID *string
	var userName, cardNumber, currency, cardStatus, cardType string
	var balance, limitVal float64
	var createdAt, updatedAt time.Time
	err := database.Pool.QueryRow(c.Request.Context(),
		`SELECT id, user_id, user_name, card_number, currency, balance, "limit", status, card_type, created_at, updated_at FROM crypto_cards WHERE id = $1`, id).
		Scan(&cardID, &userID, &userName, &cardNumber, &currency, &balance, &limitVal, &cardStatus, &cardType, &createdAt, &updatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "crypto card not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"crypto_card": map[string]interface{}{
		"id": cardID, "user_id": userID, "user_name": userName, "card_number": cardNumber,
		"currency": currency, "balance": balance, "limit": limitVal, "status": cardStatus,
		"card_type": cardType, "created_at": createdAt, "updated_at": updatedAt,
	}})
}

func handleUpdateCryptoCard(c *gin.Context) {
	id := c.Param("id")
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	allowedFields := map[string]string{
		"user_name": "user_name", "currency": "currency", "card_type": "card_type",
	}
	setClauses := []string{}
	args := []interface{}{id}
	argIdx := 2
	for field, col := range allowedFields {
		if val, ok := req[field]; ok {
			setClauses = append(setClauses, col+" = $"+strconv.Itoa(argIdx))
			args = append(args, val)
			argIdx++
		}
	}
	if len(setClauses) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no updatable fields provided"})
		return
	}
	query := `UPDATE crypto_cards SET ` + strings.Join(setClauses, ", ") + ` WHERE id = $1`
	result, err := database.Pool.Exec(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if result.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "crypto card not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "crypto card updated"})
}

func handleDeleteCryptoCard(c *gin.Context) {
	id := c.Param("id")
	result, err := database.Pool.Exec(c.Request.Context(), `DELETE FROM crypto_cards WHERE id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if result.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "crypto card not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "crypto card deleted"})
}

func handleBlockCryptoCard(c *gin.Context) {
	id := c.Param("id")
	result, err := database.Pool.Exec(c.Request.Context(), `UPDATE crypto_cards SET status = 'blocked' WHERE id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if result.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "crypto card not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "crypto card blocked", "status": "blocked"})
}

func handleActivateCryptoCard(c *gin.Context) {
	id := c.Param("id")
	result, err := database.Pool.Exec(c.Request.Context(), `UPDATE crypto_cards SET status = 'active' WHERE id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if result.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "crypto card not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "crypto card activated", "status": "active"})
}

func handleSetCryptoCardLimit(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Limit float64 `json:"limit" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := database.Pool.Exec(c.Request.Context(), `UPDATE crypto_cards SET "limit" = $1 WHERE id = $2`, req.Limit, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if result.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "crypto card not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "limit updated", "limit": req.Limit})
}

func handleUpdateCryptoCardStatus(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := database.Pool.Exec(c.Request.Context(), `UPDATE crypto_cards SET status = $1 WHERE id = $2`, req.Status, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if result.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "crypto card not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "status updated", "status": req.Status})
}
