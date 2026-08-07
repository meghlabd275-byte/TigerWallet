// TigerWallet Admin Panel - Main Entry Point
// Production-ready Go backend with all features
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
	"github.com/tigerwallet/admin_panel/internal/config"
	"github.com/tigerwallet/admin_panel/internal/database"
	"github.com/tigerwallet/admin_panel/internal/handlers"
	"github.com/tigerwallet/admin_panel/internal/middleware"
	"github.com/tigerwallet/admin_panel/internal/services"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	if err := database.Initialize(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	log.Println("Database initialized successfully")

	// Initialize services
	authService := services.NewAuthService(cfg)
	userService := services.NewUserService()
	kycService := services.NewKYCService()
	transactionService := services.NewTransactionService()
	withdrawalService := services.NewWithdrawalService()
	tokenService := services.NewTokenService()
	blockchainService := services.NewBlockchainService()
	feeService := services.NewFeeService()
	webhookService := services.NewWebhookService()
	notificationService := services.NewNotificationService()
	auditService := services.NewAuditService()
	sessionService := services.NewSessionService()
	featureFlagService := services.NewFeatureFlagService()
	ipWhitelistService := services.NewIPWhitelistService()
	reportService := services.NewReportService()
	ticketService := services.NewTicketService()
	whiteLabelService := services.NewWhiteLabelService()
	slaService := services.NewSLAService()
	integrationService := services.NewIntegrationService(cfg)

	// Initialize handler
	h := handlers.NewHandler(
		authService,
		userService,
		kycService,
		transactionService,
		withdrawalService,
		tokenService,
		blockchainService,
		feeService,
		webhookService,
		notificationService,
		auditService,
		sessionService,
		featureFlagService,
		ipWhitelistService,
		reportService,
		ticketService,
		whiteLabelService,
		slaService,
		integrationService,
	)

	// Setup Gin router
	router := gin.Default()

	// CORS configuration
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Health check
	router.GET("/health", h.HealthCheck)

	// Public routes
	api := router.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", h.Register)
			auth.POST("/login", h.Login)
			auth.POST("/refresh", h.RefreshToken)
		}
	}

	// Protected routes
	admin := api.Group("/admin")
	admin.Use(middleware.JWTAuth(cfg))
	admin.Use(middleware.IPWhitelistMiddleware(cfg))
	{
		// Auth endpoints
		admin.GET("/admins", h.GetAdmins)
		admin.GET("/admins/:id", h.GetAdmin)
		admin.PUT("/admins/:id", h.UpdateAdmin)
		admin.DELETE("/admins/:id", h.DeleteAdmin)
		admin.POST("/admins/:id/suspend", h.SuspendAdmin)
		admin.POST("/admins/:id/activate", h.ActivateAdmin)
		admin.POST("/change-password", h.ChangePassword)
		admin.POST("/2fa/enable", h.Enable2FA)
		admin.POST("/2fa/disable", h.Disable2FA)
		admin.POST("/logout", h.Logout)

		// Session management
		admin.GET("/sessions", h.GetSessions)
		admin.DELETE("/sessions/:id", h.RevokeSession)
		admin.DELETE("/sessions", h.RevokeAllSessions)

		// User management
		users := admin.Group("/users")
		{
			users.GET("", h.GetUsers)
			users.GET("/search", h.SearchUsers)
			users.GET("/:id", h.GetUser)
			users.PUT("/:id/status", h.UpdateUserStatus)
			users.POST("/:id/ban", h.BanUser)
			users.POST("/:id/unban", h.UnbanUser)
			users.POST("/:id/suspend", h.SuspendUser)
		}

		// KYC management
		kyc := admin.Group("/kyc")
		{
			kyc.GET("", h.GetKYCRequests)
			kyc.POST("/:id/approve", h.ApproveKYC)
			kyc.POST("/:id/reject", h.RejectKYC)
		}

		// Transaction management
		transactions := admin.Group("/transactions")
		{
			transactions.GET("", h.GetTransactions)
			transactions.GET("/:id", h.GetTransaction)
			transactions.POST("/:id/flag", h.FlagTransaction)
			transactions.POST("/:id/unflag", h.UnflagTransaction)
		}

		// Withdrawal management
		withdrawals := admin.Group("/withdrawals")
		{
			withdrawals.GET("", h.GetWithdrawals)
			withdrawals.POST("/:id/approve", h.ApproveWithdrawal)
			withdrawals.POST("/:id/reject", h.RejectWithdrawal)
			withdrawals.POST("/:id/process", h.ProcessWithdrawal)
		}

		// Token management
		tokens := admin.Group("/tokens")
		{
			tokens.GET("", h.GetTokens)
			tokens.POST("", h.CreateToken)
			tokens.PUT("/:id", h.UpdateToken)
			tokens.DELETE("/:id", h.DeleteToken)
		}

		// Trading pairs
		pairs := admin.Group("/pairs")
		{
			pairs.GET("", h.GetTradingPairs)
			pairs.POST("", h.CreateTradingPair)
			pairs.PUT("/:id/status", h.UpdatePairStatus)
		}

		// Blockchain management
		blockchains := admin.Group("/blockchains")
		{
			blockchains.GET("", h.GetBlockchains)
			blockchains.POST("", h.CreateBlockchain)
			blockchains.PUT("/:id", h.UpdateBlockchain)
			blockchains.PUT("/:id/status", h.SetBlockchainStatus)
		}

		// Fee management
		fees := admin.Group("/fees")
		{
			fees.GET("", h.GetFeeStructures)
			fees.POST("", h.CreateFeeStructure)
			fees.PUT("/:id", h.UpdateFeeStructure)
		}

		// Webhook management
		webhooks := admin.Group("/webhooks")
		{
			webhooks.GET("", h.GetWebhooks)
			webhooks.POST("", h.CreateWebhook)
			webhooks.POST("/:id/test", h.TestWebhook)
			webhooks.DELETE("/:id", h.DeleteWebhook)
		}

		// Notifications
		notifications := admin.Group("/notifications")
		{
			notifications.GET("", h.GetNotifications)
			notifications.PUT("/:id/read", h.MarkNotificationRead)
			notifications.POST("/send", h.SendNotification)
			notifications.POST("/broadcast", h.BroadcastNotification)
		}

		// Audit logs
		audit := admin.Group("/audit-logs")
		{
			audit.GET("", h.GetAuditLogs)
			audit.POST("/export", h.ExportAuditLogs)
		}

		// Feature flags
		featureFlags := admin.Group("/feature-flags")
		{
			featureFlags.GET("", h.GetFeatureFlags)
			featureFlags.POST("", h.CreateFeatureFlag)
			featureFlags.PUT("/:id", h.UpdateFeatureFlag)
			featureFlags.DELETE("/:id", h.DeleteFeatureFlag)
		}

		// IP Whitelist
		ipWhitelist := admin.Group("/ip-whitelist")
		{
			ipWhitelist.GET("", h.GetIPWhitelist)
			ipWhitelist.POST("", h.AddIPToWhitelist)
			ipWhitelist.DELETE("/:id", h.RemoveIPFromWhitelist)
		}

		// Reports
		reports := admin.Group("/reports")
		{
			reports.GET("", h.GetReports)
			reports.POST("/generate", h.GenerateReport)
		}

		// Tickets
		tickets := admin.Group("/tickets")
		{
			tickets.GET("", h.GetTickets)
			tickets.GET("/:id", h.GetTicket)
			tickets.POST("", h.CreateTicket)
			tickets.PUT("/:id/status", h.UpdateTicketStatus)
			tickets.POST("/:id/messages", h.AddTicketMessage)
			tickets.PUT("/:id/assign", h.AssignTicket)
		}

		// White labels
		whiteLabels := admin.Group("/white-labels")
		{
			whiteLabels.GET("", h.GetWhiteLabels)
			whiteLabels.POST("", h.CreateWhiteLabel)
			whiteLabels.PUT("/:id", h.UpdateWhiteLabel)
			whiteLabels.DELETE("/:id", h.DeleteWhiteLabel)
		}

		// SLA Management
		sla := admin.Group("/sla")
		{
			sla.GET("/policies", h.GetSLAPolicies)
			sla.POST("/policies", h.CreateSLAPolicy)
			sla.PUT("/policies/:id", h.UpdateSLAPolicy)
			sla.DELETE("/policies/:id", h.DeleteSLAPolicy)
			sla.GET("/reports", h.GetSLAReports)
			sla.POST("/reports/generate", h.GenerateSLAReport)
		}

		// Integrations
		integrations := admin.Group("/integrations")
		{
			integrations.GET("", h.GetIntegrations)
			integrations.POST("", h.CreateIntegration)
			integrations.PUT("/:id", h.UpdateIntegration)
			integrations.DELETE("/:id", h.DeleteIntegration)
			integrations.POST("/:id/test", h.TestIntegration)
		}

		// Platform stats
		admin.GET("/stats", h.GetPlatformStats)
	}

	// Create server
	srv := &http.Server{
		Addr:           ":" + cfg.ServerPort,
		Handler:        router,
		ReadTimeout:    cfg.ServerReadTimeout,
		WriteTimeout:   cfg.ServerWriteTimeout,
		IdleTimeout:    cfg.ServerIdleTimeout,
		MaxHeaderBytes: 1 << 20,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Admin Panel API server starting on port %s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited properly")
}
