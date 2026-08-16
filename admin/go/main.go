// TigerWallet Admin - Main Entry Point
// Canonical admin backend: Go for high-load distributed admin operations.
// All handlers are real, DB-backed (PostgreSQL via GORM) with Redis caching.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tigerwallet/admin/internal/config"
	"github.com/tigerwallet/admin/internal/handlers"
	"github.com/tigerwallet/admin/internal/middleware"
	"github.com/tigerwallet/admin/internal/models"
	"github.com/tigerwallet/admin/pkg/auth"
	"github.com/tigerwallet/admin/pkg/database"
	"github.com/tigerwallet/admin/pkg/redis"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	_ "gorm.io/gorm"
)

func main() {
	cfg := config.Load()

	// Initialize PostgreSQL (GORM + connection pool + auto-migrate)
	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("Connected to PostgreSQL")

	// Initialize Redis
	redisClient, err := redis.NewRedisClient(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()
	log.Println("Connected to Redis")

	// Auth service (JWT + bcrypt + pepper)
	authSvc := auth.NewAuthService(cfg)

	// Auto-migrate handler-package governance models (avoid import cycle by
	// migrating here rather than in pkg/database, which handlers import).
	if err := db.DB.AutoMigrate(
		&handlers.FuturesPosition{},
		&handlers.OptionsContract{},
		&handlers.CopyTradingConfig{},
		&handlers.ConvertOrder{},
		&handlers.OnRampOrder{},
		&handlers.OffRampOrder{},
		&handlers.P2PClient{},
		&handlers.Partner{},
		&handlers.RewardCampaign{},
		&handlers.MarketingCampaign{},
		&handlers.AdminRole{},
		&handlers.AdminRoleAssignment{},
		&handlers.AdminPermission{},
	); err != nil {
		log.Fatalf("Failed to migrate admin domain governance models: %v", err)
	}

	// Create default super admin if none exists
	createDefaultAdmin(db, cfg, authSvc)

	// --- Initialize all handlers (real, DB-backed) ---
	adminHandler := handlers.NewAdminHandler(db, redisClient, cfg, authSvc)
	userHandler := handlers.NewUserHandler(db)
	dashboardHandler := handlers.NewDashboardHandler(db)
	transactionHandler := handlers.NewTransactionHandler(db)
	kycHandler := handlers.NewKYCHandler(db, redisClient.Client)
	tokenHandler := handlers.NewTokenHandler(db, authSvc)
	withdrawalHandler := handlers.NewWithdrawalHandler(db, redisClient.Client)
	whitelabelHandler := handlers.NewWhiteLabelHandler(db)
	feeHandler := handlers.NewFeeHandler(db)
	pairHandler := handlers.NewPairHandler(db)
	apiKeyHandler := handlers.NewAPIKeyHandler(db)
	analyticsHandler := handlers.NewAnalyticsHandler(db)
	superAdminHandler := handlers.NewSuperAdminHandler(db, redisClient, cfg, authSvc)
	twoFactorHandler := handlers.NewTwoFactorHandler(db, redisClient.Client)
	notificationHandler := handlers.NewNotificationHandler(db)
	supportHandler := handlers.NewSupportHandler(db)
	integrationHandler := handlers.NewIntegrationHandler(db, redisClient.Client)
	brokerHandler := handlers.NewBrokerHandler(db)
	institutionalHandler := handlers.NewInstitutionalHandler(db)
	complianceHandler := handlers.NewComplianceHandler(db)
	knowledgeBaseHandler := handlers.NewKnowledgeBaseHandler(db)
	multisigHandler := handlers.NewMultisigHandler(db)
	nftHandler := handlers.NewNFTHandler(db)
	masterWalletHandler := handlers.NewMasterWalletHandler(db.DB)
	billingHandler := handlers.NewBillingHandler()
	cryptoCardHandler := handlers.NewCryptoCardHandler(db.DB)
	featuresHandler := handlers.NewFeaturesHandler(db.DB, redisClient)
	liquidityHandler := handlers.NewLiquidityHandler(db.DB)
	marginTradingHandler := handlers.NewMarginTradingHandler(db.DB)
	p2pMerchantHandler := handlers.NewP2PMerchantHandler(db.DB)

	// Trading admin domain governance handlers (mirror super_admin/go commit 0cb13d7).
	futuresHandler := handlers.NewFuturesHandler(db.DB)
	optionsHandler := handlers.NewOptionsHandler(db.DB)
	copyTradingHandler := handlers.NewCopyTradingHandler(db.DB)
	convertHandler := handlers.NewConvertHandler(db.DB)
	onrampHandler := handlers.NewOnRampHandler(db.DB)
	offrampHandler := handlers.NewOffRampHandler(db.DB)
	p2pClientsHandler := handlers.NewP2PClientsHandler(db.DB)
	partnersHandler := handlers.NewPartnersHandler(db.DB)
	rewardsHandler := handlers.NewRewardsHandler(db.DB)
	marketingHandler := handlers.NewMarketingHandler(db.DB)
	// Structured RBAC (mirror super_admin/go commit 15e99eb).
	rbacHandler := handlers.NewRBACHandler(db.DB)

	blockchainHandler := handlers.NewBlockchainHandler(db)
	exportHandler := handlers.NewExportHandler(db)

	// --- Gin router ---
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestIDMiddleware())
	router.Use(middleware.SecurityHeadersMiddleware())
	router.Use(middleware.CORSMiddleware())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "tiger-admin", "timestamp": time.Now().Unix()})
	})

	api := router.Group("/api/v1")
	{
		// Public auth endpoints
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/login", adminHandler.Login)
			authGroup.POST("/refresh", adminHandler.RefreshToken)
		}

		// Protected endpoints
		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware(authSvc))
		{
			// Auth / profile
			protected.POST("/auth/logout", adminHandler.Logout)
			protected.GET("/auth/profile", adminHandler.GetProfile)
			protected.PUT("/auth/profile", adminHandler.UpdateProfile)
			protected.POST("/auth/change-password", adminHandler.ChangePassword)

			// 2FA
			twofa := protected.Group("/2fa")
			{
				twofa.POST("/setup", twoFactorHandler.Setup2FA)
				twofa.POST("/verify", twoFactorHandler.Verify2FA)
				twofa.POST("/enable", twoFactorHandler.Enable2FA)
				twofa.POST("/disable", twoFactorHandler.Disable2FA)
				twofa.GET("/status", twoFactorHandler.Get2FAStatus)
				twofa.POST("/backup-codes", twoFactorHandler.RegenerateBackupCodes)
				twofa.GET("/users", twoFactorHandler.List2FAUsers)
				twofa.GET("/stats", twoFactorHandler.Get2FAStats)
			}

			// Admin management (super_admin only)
			admins := protected.Group("/admins")
			admins.Use(middleware.SuperAdminMiddleware())
			{
				admins.GET("", superAdminHandler.ListAdmins)
				admins.POST("", superAdminHandler.CreateAdmin)
				admins.GET("/:id", superAdminHandler.GetAdmin)
				admins.PUT("/:id", superAdminHandler.UpdateAdmin)
				admins.DELETE("/:id", superAdminHandler.DeleteAdmin)
				admins.POST("/:id/suspend", superAdminHandler.SuspendAdmin)
				admins.POST("/:id/activate", superAdminHandler.ActivateAdmin)
				admins.GET("/:id/activities", adminHandler.GetAdminActivities)
			}

			// Dashboard & analytics
			protected.GET("/dashboard", dashboardHandler.GetDashboard)
			protected.GET("/analytics/users", analyticsHandler.GetUserAnalytics)
			protected.GET("/analytics/transactions", analyticsHandler.GetTransactionAnalytics)
			protected.GET("/analytics/revenue", analyticsHandler.GetRevenueAnalytics)
			protected.GET("/analytics/custom", analyticsHandler.GetCustomDateRangeAnalytics)
			protected.GET("/system/metrics", analyticsHandler.GetSystemMetrics)

			// Users
			users := protected.Group("/users")
			{
				users.GET("", userHandler.ListUsers)
				users.GET("/:id", userHandler.GetUser)
				users.PUT("/:id", userHandler.UpdateUser)
				users.DELETE("/:id", userHandler.DeleteUser)
				users.POST("/:id/verify-kyc", userHandler.VerifyKYC)
			}

			// KYC
			kyc := protected.Group("/kyc")
			{
				kyc.GET("", kycHandler.ListKYC)
				kyc.GET("/:id", kycHandler.GetKYC)
				kyc.POST("/:id/approve", kycHandler.ApproveKYC)
				kyc.POST("/:id/reject", kycHandler.RejectKYC)
				kyc.GET("/stats", kycHandler.GetKYCStats)
			}

			// Transactions
			transactions := protected.Group("/transactions")
			{
				transactions.GET("", transactionHandler.ListTransactions)
				transactions.GET("/:id", transactionHandler.GetTransaction)
				transactions.POST("/:id/flag", transactionHandler.FlagTransaction)
			}

			// Tokens
			tokens := protected.Group("/tokens")
			{
				tokens.GET("", tokenHandler.ListTokens)
				tokens.POST("", tokenHandler.CreateToken)
				tokens.GET("/:id", tokenHandler.GetToken)
				tokens.PUT("/:id", tokenHandler.UpdateToken)
				tokens.DELETE("/:id", tokenHandler.DeleteToken)
				tokens.POST("/:id/activate", tokenHandler.ActivateToken)
				tokens.POST("/:id/deactivate", tokenHandler.DeactivateToken)
				tokens.POST("/:id/verify", tokenHandler.VerifyToken)
				tokens.GET("/stats", tokenHandler.GetTokenStats)
				tokens.PUT("/:id/price", tokenHandler.UpdateTokenPrice)
			}

			// Withdrawals
			withdrawals := protected.Group("/withdrawals")
			{
				withdrawals.GET("", withdrawalHandler.ListWithdrawals)
				withdrawals.GET("/:id", withdrawalHandler.GetWithdrawal)
				withdrawals.POST("/:id/approve", withdrawalHandler.ApproveWithdrawal)
				withdrawals.POST("/:id/reject", withdrawalHandler.RejectWithdrawal)
				withdrawals.POST("/:id/process", withdrawalHandler.ProcessWithdrawal)
				withdrawals.GET("/stats", withdrawalHandler.GetWithdrawalStats)
				withdrawals.POST("/bulk-approve", withdrawalHandler.BulkApproveWithdrawals)
			}

			// White labels
			whiteLabels := protected.Group("/white-labels")
			{
				whiteLabels.GET("", whitelabelHandler.ListWhiteLabels)
				whiteLabels.POST("", whitelabelHandler.CreateWhiteLabel)
				whiteLabels.GET("/:id", whitelabelHandler.GetWhiteLabel)
				whiteLabels.PUT("/:id", whitelabelHandler.UpdateWhiteLabel)
				whiteLabels.DELETE("/:id", whitelabelHandler.DeleteWhiteLabel)
				whiteLabels.POST("/:id/approve", whitelabelHandler.ApproveWhiteLabel)
				whiteLabels.POST("/:id/suspend", whitelabelHandler.SuspendWhiteLabel)
				whiteLabels.GET("/stats", whitelabelHandler.GetWhiteLabelStats)
			}

			// Trading pairs
			pairs := protected.Group("/pairs")
			{
				pairs.GET("", pairHandler.ListPairs)
				pairs.POST("", pairHandler.CreatePair)
				pairs.POST("/import", pairHandler.ImportPairs)
				pairs.GET("/:id", pairHandler.GetPair)
				pairs.PUT("/:id", pairHandler.UpdatePair)
				pairs.DELETE("/:id", pairHandler.DeletePair)
				pairs.PUT("/:id/status", pairHandler.UpdatePairStatus)
				pairs.PUT("/:id/price", pairHandler.UpdatePairPrice)
				pairs.GET("/stats", pairHandler.GetPairStats)
			}

			// Fees
			fees := protected.Group("/fees")
			{
				fees.GET("", feeHandler.ListFees)
				fees.POST("", feeHandler.CreateFee)
				fees.GET("/:id", feeHandler.GetFee)
				fees.PUT("/:id", feeHandler.UpdateFee)
				fees.DELETE("/:id", feeHandler.DeleteFee)
				fees.POST("/calculate", feeHandler.CalculateFee)
				fees.GET("/stats", feeHandler.GetFeeStats)
			}

			// API keys
			apiKeys := protected.Group("/api-keys")
			{
				apiKeys.GET("", apiKeyHandler.ListAPIKeys)
				apiKeys.POST("", apiKeyHandler.CreateAPIKey)
				apiKeys.GET("/:id", apiKeyHandler.GetAPIKey)
				apiKeys.PUT("/:id", apiKeyHandler.UpdateAPIKey)
				apiKeys.DELETE("/:id", apiKeyHandler.DeleteAPIKey)
				apiKeys.POST("/:id/revoke", apiKeyHandler.RevokeAPIKey)
				apiKeys.POST("/:id/reactivate", apiKeyHandler.ReactivateAPIKey)
				apiKeys.POST("/:id/regenerate", apiKeyHandler.RegenerateAPIKey)
				apiKeys.GET("/stats", apiKeyHandler.GetAPIKeyStats)
			}

			// System config (super_admin)
			sysConfig := protected.Group("/system")
			sysConfig.Use(middleware.SuperAdminMiddleware())
			{
				sysConfig.GET("/config", superAdminHandler.GetSystemConfig)
				sysConfig.PUT("/config", superAdminHandler.UpdateSystemConfig)
				sysConfig.GET("/rate-limits", superAdminHandler.GetRateLimits)
				sysConfig.PUT("/rate-limits", superAdminHandler.UpdateRateLimits)
				sysConfig.GET("/master-wallets", superAdminHandler.ListMasterWallets)
				sysConfig.GET("/master-wallets/:id", superAdminHandler.GetMasterWallet)
				sysConfig.GET("/master-wallets/:id/balance", superAdminHandler.GetMasterWalletBalance)
			}

			// Feature flags
			featureFlags := protected.Group("/feature-flags")
			{
				featureFlags.GET("", superAdminHandler.ListFeatureFlags)
				featureFlags.POST("", superAdminHandler.CreateFeatureFlag)
				featureFlags.PUT("/:id", superAdminHandler.UpdateFeatureFlag)
				featureFlags.DELETE("/:id", superAdminHandler.DeleteFeatureFlag)
			}

			// Notifications
			notifications := protected.Group("/notifications")
			{
				notifications.GET("", notificationHandler.GetNotifications)
				notifications.GET("/:id", notificationHandler.GetNotification)
				notifications.PUT("/:id/read", notificationHandler.MarkAsRead)
				notifications.DELETE("/:id", notificationHandler.DeleteNotification)
				notifications.GET("/stats", notificationHandler.GetNotificationStats)
				notifications.POST("/send", notificationHandler.SendNotification)
				notifications.POST("/broadcast", notificationHandler.SendToAllUsers)
				notifications.POST("/template", notificationHandler.SendTemplateNotification)
			}

			// Support tickets
			tickets := protected.Group("/tickets")
			{
				tickets.GET("", supportHandler.ListTickets)
				tickets.POST("", supportHandler.CreateTicket)
				tickets.GET("/:id", supportHandler.GetTicket)
				tickets.PUT("/:id", supportHandler.UpdateTicket)
				tickets.POST("/:id/messages", supportHandler.AddMessage)
				tickets.POST("/:id/close", supportHandler.CloseTicket)
				tickets.GET("/stats", supportHandler.GetTicketStats)
				tickets.GET("/sla-violations", supportHandler.GetSLAViolations)
			}

			// Integrations
			integrations := protected.Group("/integrations")
			{
				integrations.GET("", integrationHandler.ListIntegrations)
				integrations.POST("", integrationHandler.CreateIntegration)
				integrations.PUT("/:id", integrationHandler.UpdateIntegration)
				integrations.DELETE("/:id", integrationHandler.DeleteIntegration)
				integrations.POST("/:id/test", integrationHandler.TestIntegration)
				integrations.POST("/slack", integrationHandler.SendSlackNotification)
				integrations.POST("/pagerduty", integrationHandler.SendPagerDutyAlert)
				integrations.POST("/datadog", integrationHandler.SendDatadogEvent)
				integrations.POST("/webhook", integrationHandler.WebhookHandler)
				integrations.GET("/stats", integrationHandler.GetIntegrationStats)
			}

			// Brokers
			brokers := protected.Group("/brokers")
			{
				brokers.GET("", brokerHandler.ListBrokers)
				brokers.POST("", brokerHandler.CreateBroker)
				brokers.GET("/:id", brokerHandler.GetBroker)
				brokers.PUT("/:id", brokerHandler.UpdateBroker)
				brokers.DELETE("/:id", brokerHandler.DeleteBroker)
				brokers.POST("/:id/approve", brokerHandler.ApproveBroker)
				brokers.POST("/:id/suspend", brokerHandler.SuspendBroker)
				brokers.PUT("/:id/commission", brokerHandler.SetBrokerCommission)
				brokers.GET("/:id/clients", brokerHandler.GetBrokerClients)
				brokers.GET("/stats", brokerHandler.GetBrokerStats)
			}

			// Institutional clients
			institutional := protected.Group("/institutional")
			{
				institutional.GET("", institutionalHandler.ListClients)
				institutional.POST("", institutionalHandler.CreateClient)
				institutional.GET("/:id", institutionalHandler.GetClient)
				institutional.PUT("/:id", institutionalHandler.UpdateClient)
				institutional.DELETE("/:id", institutionalHandler.DeleteClient)
				institutional.POST("/:id/approve", institutionalHandler.ApproveClient)
				institutional.POST("/:id/suspend", institutionalHandler.SuspendClient)
				institutional.PUT("/:id/limits", institutionalHandler.SetClientLimits)
				institutional.PUT("/:id/account-manager", institutionalHandler.AssignAccountManager)
				institutional.GET("/stats", institutionalHandler.GetClientStats)
			}

			// Compliance
			compliance := protected.Group("/compliance")
			{
				compliance.POST("/aml-report", complianceHandler.GenerateAMLReport)
				compliance.POST("/tax-report", complianceHandler.GenerateTaxReport)
				compliance.POST("/gdpr", complianceHandler.ProcessGDPRRequest)
				compliance.GET("/reports", complianceHandler.GetComplianceReports)
				compliance.GET("/stats", complianceHandler.GetComplianceStats)
				compliance.POST("/gdpr/export", complianceHandler.ExportGDPRData)
				compliance.POST("/gdpr/anonymize", complianceHandler.AnonymizeUserData)
			}

			// Knowledge base
			kb := protected.Group("/knowledge-base")
			{
				kb.GET("/categories", knowledgeBaseHandler.ListCategories)
				kb.POST("/categories", knowledgeBaseHandler.CreateCategory)
				kb.PUT("/categories/:id", knowledgeBaseHandler.UpdateCategory)
				kb.DELETE("/categories/:id", knowledgeBaseHandler.DeleteCategory)
				kb.GET("/articles", knowledgeBaseHandler.ListArticles)
				kb.POST("/articles", knowledgeBaseHandler.CreateArticle)
				kb.GET("/articles/:id", knowledgeBaseHandler.GetArticle)
				kb.PUT("/articles/:id", knowledgeBaseHandler.UpdateArticle)
				kb.DELETE("/articles/:id", knowledgeBaseHandler.DeleteArticle)
				kb.GET("/stats", knowledgeBaseHandler.GetKnowledgeBaseStats)
			}

			// Multisig
			multisig := protected.Group("/multisig")
			{
				multisig.GET("", multisigHandler.ListWallets)
				multisig.POST("", multisigHandler.CreateWallet)
				multisig.GET("/:id", multisigHandler.GetWallet)
				multisig.PUT("/:id", multisigHandler.UpdateWallet)
				multisig.DELETE("/:id", multisigHandler.DeleteWallet)
			}

			// NFT management
			nfts := protected.Group("/nfts")
			{
				nfts.GET("", nftHandler.ListNFTs)
				nfts.GET("/:id", nftHandler.GetNFT)
				nfts.DELETE("/:id", nftHandler.DeleteNFT)
				nfts.POST("/:id/flag", nftHandler.FlagNFT)
				nfts.GET("/stats", nftHandler.GetNFTStats)
			}

			// Master wallet
			masterWallet := protected.Group("/master-wallet")
			{
				masterWallet.GET("/stats", masterWalletHandler.GetStats)
				masterWallet.GET("/balances", masterWalletHandler.GetBalances)
				masterWallet.GET("/transactions", masterWalletHandler.GetTransactions)
			}

			// Billing
			billing := protected.Group("/billing")
			{
				billing.GET("/plans", billingHandler.GetPlans)
				billing.POST("/plans", billingHandler.CreatePlan)
				billing.PUT("/plans/:id", billingHandler.UpdatePlan)
				billing.DELETE("/plans/:id", billingHandler.DeletePlan)
				billing.GET("/subscription", billingHandler.GetSubscription)
				billing.POST("/subscription", billingHandler.CreateSubscription)
				billing.DELETE("/subscription", billingHandler.CancelSubscription)
				billing.GET("/invoices", billingHandler.GetInvoices)
				billing.GET("/payment-methods", billingHandler.GetPaymentMethods)
				billing.POST("/payment-methods", billingHandler.AddPaymentMethod)
				billing.DELETE("/payment-methods/:id", billingHandler.DeletePaymentMethod)
				billing.PUT("/payment-methods/:id/default", billingHandler.SetDefaultPaymentMethod)
			}

			// Crypto cards
			cryptoCards := protected.Group("/crypto-cards")
			{
				cryptoCards.GET("", cryptoCardHandler.GetAll)
				cryptoCards.POST("", cryptoCardHandler.Create)
				cryptoCards.GET("/:id", cryptoCardHandler.GetByID)
				cryptoCards.POST("/:id/block", cryptoCardHandler.Block)
				cryptoCards.POST("/:id/activate", cryptoCardHandler.Activate)
				cryptoCards.PUT("/:id/limit", cryptoCardHandler.SetLimit)
			}

			// Features
			features := protected.Group("/features")
			{
				features.GET("", featuresHandler.GetAll)
				features.POST("", featuresHandler.Create)
				features.GET("/:id", featuresHandler.GetByID)
				features.PUT("/:id", featuresHandler.Update)
				features.POST("/:id/toggle", featuresHandler.Toggle)
				features.PUT("/:id/rollout", featuresHandler.SetRollout)
				features.DELETE("/:id", featuresHandler.Delete)
				features.GET("/:id/check", featuresHandler.CheckFeature)
				features.PATCH("/:id/status", featuresHandler.SetStatus)
				features.PUT("/:id/status", featuresHandler.SetStatus)
			}

			// Liquidity pools
			liquidity := protected.Group("/liquidity")
			{
				liquidity.GET("/pools", liquidityHandler.GetPools)
				liquidity.GET("/pools/:id", liquidityHandler.GetPoolByID)
				liquidity.POST("/pools", liquidityHandler.CreatePool)
				liquidity.POST("/pools/:id/add", liquidityHandler.AddLiquidity)
				liquidity.POST("/pools/:id/remove", liquidityHandler.RemoveLiquidity)
				liquidity.GET("/stats", liquidityHandler.GetStats)
			}

			// Margin trading
			margin := protected.Group("/margin-trading")
			{
				margin.GET("/positions", marginTradingHandler.GetPositions)
				margin.POST("/positions", marginTradingHandler.OpenPosition)
				margin.POST("/positions/:id/close", marginTradingHandler.ClosePosition)
				margin.PUT("/positions/:id/liquidation", marginTradingHandler.UpdateLiquidationPrice)
				margin.GET("/stats", marginTradingHandler.GetStats)
			}

			// P2P merchants
			p2p := protected.Group("/p2p-merchants")
			{
				p2p.GET("", p2pMerchantHandler.GetMerchants)
				p2p.POST("", p2pMerchantHandler.CreateMerchant)
				p2p.GET("/:id", p2pMerchantHandler.GetMerchantByID)
				p2p.PUT("/:id", p2pMerchantHandler.UpdateMerchant)
				p2p.POST("/:id/approve", p2pMerchantHandler.ApproveMerchant)
				p2p.POST("/:id/reject", p2pMerchantHandler.RejectMerchant)
				p2p.GET("/:id/transactions", p2pMerchantHandler.GetTransactions)
			}

			// P2P clients
			p2pClients := protected.Group("/p2p-clients")
			{
				p2pClients.GET("", p2pClientsHandler.List)
				p2pClients.POST("", p2pClientsHandler.Create)
				p2pClients.GET("/:id", p2pClientsHandler.Get)
				p2pClients.PUT("/:id", p2pClientsHandler.Update)
				p2pClients.DELETE("/:id", p2pClientsHandler.Delete)
				p2pClients.PUT("/:id/status", p2pClientsHandler.UpdateStatus)
			}

			// Futures (governance records only)
			futures := protected.Group("/futures")
			{
				futures.GET("", futuresHandler.List)
				futures.POST("", futuresHandler.Create)
				futures.GET("/:id", futuresHandler.Get)
				futures.PUT("/:id", futuresHandler.Update)
				futures.DELETE("/:id", futuresHandler.Delete)
				futures.PUT("/:id/status", futuresHandler.UpdateStatus)
			}

			// Options (governance records only)
			opts := protected.Group("/options")
			{
				opts.GET("", optionsHandler.List)
				opts.POST("", optionsHandler.Create)
				opts.GET("/:id", optionsHandler.Get)
				opts.PUT("/:id", optionsHandler.Update)
				opts.DELETE("/:id", optionsHandler.Delete)
				opts.PUT("/:id/status", optionsHandler.UpdateStatus)
			}

			// Copy trading (governance records only)
			copyTrading := protected.Group("/copy-trading")
			{
				copyTrading.GET("", copyTradingHandler.List)
				copyTrading.POST("", copyTradingHandler.Create)
				copyTrading.GET("/:id", copyTradingHandler.Get)
				copyTrading.PUT("/:id", copyTradingHandler.Update)
				copyTrading.DELETE("/:id", copyTradingHandler.Delete)
				copyTrading.PUT("/:id/status", copyTradingHandler.UpdateStatus)
			}

			// Convert (governance records only)
			convert := protected.Group("/convert")
			{
				convert.GET("", convertHandler.List)
				convert.POST("", convertHandler.Create)
				convert.GET("/:id", convertHandler.Get)
				convert.PUT("/:id", convertHandler.Update)
				convert.DELETE("/:id", convertHandler.Delete)
				convert.PUT("/:id/status", convertHandler.UpdateStatus)
			}

			// OnRamp (governance records only; approve/reject are record-only)
			onramp := protected.Group("/onramp")
			{
				onramp.GET("", onrampHandler.List)
				onramp.POST("", onrampHandler.Create)
				onramp.GET("/:id", onrampHandler.Get)
				onramp.PUT("/:id", onrampHandler.Update)
				onramp.DELETE("/:id", onrampHandler.Delete)
				onramp.POST("/:id/approve", onrampHandler.Approve)
				onramp.POST("/:id/reject", onrampHandler.Reject)
			}

			// OffRamp (governance records only; approve/reject are record-only)
			offramp := protected.Group("/offramp")
			{
				offramp.GET("", offrampHandler.List)
				offramp.POST("", offrampHandler.Create)
				offramp.GET("/:id", offrampHandler.Get)
				offramp.PUT("/:id", offrampHandler.Update)
				offramp.DELETE("/:id", offrampHandler.Delete)
				offramp.POST("/:id/approve", offrampHandler.Approve)
				offramp.POST("/:id/reject", offrampHandler.Reject)
			}

			// Partners (governance records only; api_key generated on create)
			partners := protected.Group("/partners")
			{
				partners.GET("", partnersHandler.List)
				partners.POST("", partnersHandler.Create)
				partners.GET("/:id", partnersHandler.Get)
				partners.PUT("/:id", partnersHandler.Update)
				partners.DELETE("/:id", partnersHandler.Delete)
				partners.PUT("/:id/status", partnersHandler.UpdateStatus)
				partners.POST("/:id/approve", partnersHandler.Approve)
				partners.POST("/:id/reject", partnersHandler.Reject)
			}

			// Rewards (governance records only)
			rewards := protected.Group("/rewards")
			{
				rewards.GET("", rewardsHandler.List)
				rewards.POST("", rewardsHandler.Create)
				rewards.GET("/:id", rewardsHandler.Get)
				rewards.PUT("/:id", rewardsHandler.Update)
				rewards.DELETE("/:id", rewardsHandler.Delete)
				rewards.PUT("/:id/status", rewardsHandler.UpdateStatus)
			}

			// Marketing (governance records only)
			marketing := protected.Group("/marketing")
			{
				marketing.GET("", marketingHandler.List)
				marketing.POST("", marketingHandler.Create)
				marketing.GET("/:id", marketingHandler.Get)
				marketing.PUT("/:id", marketingHandler.Update)
				marketing.DELETE("/:id", marketingHandler.Delete)
				marketing.PUT("/:id/status", marketingHandler.UpdateStatus)
			}

			// Structured RBAC: roles, permissions, assignments.
			roles := protected.Group("/roles")
			{
				roles.GET("", rbacHandler.ListRoles)
				roles.POST("", rbacHandler.CreateRole)
				roles.GET("/:id", rbacHandler.GetRole)
				roles.PUT("/:id", rbacHandler.UpdateRole)
				roles.DELETE("/:id", rbacHandler.DeleteRole)
			}
			permissions := protected.Group("/permissions")
			{
				permissions.GET("", rbacHandler.ListPermissions)
				permissions.POST("", rbacHandler.CreatePermission)
			}
			// Per-admin role assignments + effective permissions.
			protected.GET("/admins/:id/roles", rbacHandler.ListRoles)
			protected.POST("/admins/:id/roles", rbacHandler.AssignRole)
			protected.DELETE("/admins/:id/roles/:roleId", rbacHandler.RevokeRole)
			protected.GET("/admins/:id/permissions", rbacHandler.GetEffectivePermissions)

			// Audit logs
			protected.GET("/audit-logs", superAdminHandler.GetAuditLogs)

			// Activities (admin action audit log from admin_activities table)
			protected.GET("/activities", blockchainHandler.ListActivities)

			// Blockchains (registry CRUD)
			blockchains := protected.Group("/blockchains")
			{
				blockchains.GET("", blockchainHandler.ListBlockchains)
				blockchains.POST("", blockchainHandler.CreateBlockchain)
				blockchains.GET("/:id", blockchainHandler.GetBlockchain)
				blockchains.PUT("/:id", blockchainHandler.UpdateBlockchain)
				blockchains.DELETE("/:id", blockchainHandler.DeleteBlockchain)
				blockchains.POST("/:id/test-rpc", blockchainHandler.TestBlockchainRpc)
			}

			// Data exports (CSV)
			export := protected.Group("/export")
			{
				export.GET("/users", exportHandler.ExportUsers)
				export.GET("/tokens", exportHandler.ExportTokens)
				export.GET("/withdrawals", exportHandler.ExportWithdrawals)
				export.GET("/transactions", exportHandler.ExportTransactions)
			}
		}
	}

	// --- HTTP server with timeouts ---
	srv := &http.Server{
		Addr:           ":" + cfg.ServerPort,
		Handler:        router,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited properly")
}

// createDefaultAdmin creates a default super admin account on first run.
// The password is hashed with bcrypt + pepper; the plaintext is never stored.
func createDefaultAdmin(db *database.PostgresDB, cfg *config.Config, authSvc *auth.AuthService) {
	var count int64
	db.Model(&models.Admin{}).Count(&count)
	if count > 0 {
		return
	}

	hashedPassword := cfg.DefaultAdminPassword + cfg.PasswordPepper
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(hashedPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Warning: Failed to hash default admin password: %v", err)
		return
	}

	admin := models.Admin{
		Username:      "admin",
		Email:         cfg.DefaultAdminEmail,
		PasswordHash:  string(hashedBytes),
		FirstName:     "Super",
		LastName:      "Admin",
		Role:          "super_admin",
		Status:        "active",
		EmailVerified: true,
	}

	if err := db.Create(&admin).Error; err != nil {
		log.Printf("Warning: Failed to create default admin: %v", err)
		return
	}
	log.Println("Created default super admin")
}
