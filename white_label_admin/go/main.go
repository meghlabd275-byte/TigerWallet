// TigerWallet White-Label Admin — main entry point.
//
// REAL PostgreSQL-backed handlers, REAL bcrypt + JWT auth carrying tenant +
// scopes, per-endpoint RequireScope authorization, per-WL-tenant isolation.
// No stubs, no mocks, no fake data. Every handler queries PostgreSQL and
// filters by the caller's white_label_id (tenant isolation).
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
	"github.com/tigerwallet/white-label-admin/internal/config"
	"github.com/tigerwallet/white-label-admin/internal/database"
	"github.com/tigerwallet/white-label-admin/internal/handlers"
	"github.com/tigerwallet/white-label-admin/internal/middleware"
	"github.com/tigerwallet/white-label-admin/internal/roles"
)

func main() {
	cfg := config.Load()
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}
	if err := database.Initialize(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()
	log.Println("Database initialized successfully")

	svc := handlers.New(cfg, database.Pool)

	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	router.GET("/health", svc.Health)
	router.GET("/api/v1/scopes", func(c *gin.Context) {
		groups := []gin.H{}
		for _, sc := range roles.AllScopes() {
			groups = append(groups, gin.H{"scope": sc, "label": roles.ScopeGroups[sc]})
		}
		c.JSON(http.StatusOK, gin.H{"scopes": groups})
	})

	api := router.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/login", svc.Login)
		}

		admin := api.Group("/admin")
		admin.Use(middleware.JWTAuth(cfg))
		admin.Use(middleware.IPWhitelistMiddleware(cfg))
		admin.Use(middleware.TenantScope())
		{
			admin.POST("/auth/register", svc.Register)
			admin.POST("/auth/logout", svc.Logout)
			admin.POST("/auth/refresh", svc.RefreshToken)
			admin.POST("/auth/change-password", svc.ChangePassword)
			admin.POST("/auth/2fa/enable", svc.Enable2FA)
			admin.POST("/auth/2fa/disable", svc.Disable2FA)

			admin.GET("/admins", middleware.RequireScope(roles.WLClient), svc.ListAdmins)
			admin.POST("/admins", middleware.RequireScope(roles.WLClient), svc.Register)
			admin.GET("/admins/:id", middleware.RequireScope(roles.WLClient), svc.GetAdmin)
			admin.PUT("/admins/:id", middleware.RequireScope(roles.WLClient), svc.UpdateAdmin)
			admin.DELETE("/admins/:id", middleware.RequireScope(roles.WLClient), svc.DeleteAdmin)
			admin.POST("/admins/:id/suspend", middleware.RequireScope(roles.WLClient), svc.SuspendAdmin)
			admin.POST("/admins/:id/activate", middleware.RequireScope(roles.WLClient), svc.ActivateAdmin)

			admin.GET("/users", svc.ListUsers)
			admin.GET("/users/:id", svc.GetUser)
			admin.PUT("/users/:id/status", middleware.RequireScope(roles.WalletAdmin, roles.KYCAdmin, roles.SecurityAdmin), svc.UpdateUserStatus)
			admin.POST("/users/:id/ban", middleware.RequireScope(roles.WalletAdmin, roles.SecurityAdmin), svc.BanUser)
			admin.POST("/users/:id/unban", middleware.RequireScope(roles.WalletAdmin, roles.SecurityAdmin), svc.UnbanUser)
			admin.POST("/users/:id/suspend", middleware.RequireScope(roles.WalletAdmin, roles.SecurityAdmin), svc.SuspendUser)

			admin.GET("/kyc", middleware.RequireScope(roles.KYCAdmin, roles.ComplianceAdmin), svc.ListKYC)
			admin.POST("/kyc/:id/approve", middleware.RequireScope(roles.KYCAdmin, roles.ComplianceAdmin), svc.ApproveKYC)
			admin.POST("/kyc/:id/reject", middleware.RequireScope(roles.KYCAdmin, roles.ComplianceAdmin), svc.RejectKYC)

			admin.GET("/transactions", svc.ListTransactions)
			admin.GET("/transactions/:id", svc.GetTransaction)
			admin.POST("/transactions/:id/flag", middleware.RequireScope(roles.SecurityAdmin, roles.ComplianceAdmin), svc.FlagTransaction)
			admin.POST("/transactions/:id/unflag", middleware.RequireScope(roles.SecurityAdmin, roles.ComplianceAdmin), svc.UnflagTransaction)

			admin.GET("/withdrawals", svc.ListWithdrawals)
			admin.POST("/withdrawals/:id/approve", middleware.RequireScope(roles.WalletAdmin), svc.ApproveWithdrawal)
			admin.POST("/withdrawals/:id/reject", middleware.RequireScope(roles.WalletAdmin), svc.RejectWithdrawal)
			admin.POST("/withdrawals/:id/process", middleware.RequireScope(roles.WalletAdmin), svc.ProcessWithdrawal)

			admin.GET("/tokens", svc.ListTokens)
			admin.POST("/tokens", middleware.RequireScope(roles.ListingAdmin), svc.CreateToken)
			admin.PUT("/tokens/:id", middleware.RequireScope(roles.ListingAdmin), svc.UpdateToken)
			admin.DELETE("/tokens/:id", middleware.RequireScope(roles.ListingAdmin), svc.DeleteToken)

			admin.GET("/pairs", svc.ListPairs)
			admin.POST("/pairs", middleware.RequireScope(roles.ListingAdmin), svc.CreatePair)
			admin.PUT("/pairs/:id/status", middleware.RequireScope(roles.ListingAdmin), svc.UpdatePairStatus)

			admin.GET("/blockchains", svc.ListBlockchains)
			admin.POST("/blockchains", middleware.RequireScope(roles.ListingAdmin), svc.CreateBlockchain)
			admin.PUT("/blockchains/:id", middleware.RequireScope(roles.ListingAdmin), svc.UpdateBlockchain)
			admin.PUT("/blockchains/:id/status", middleware.RequireScope(roles.ListingAdmin), svc.SetBlockchainStatus)

			admin.GET("/fees", svc.ListFees)
			admin.POST("/fees", middleware.RequireScope(roles.WalletAdmin, roles.ListingAdmin), svc.CreateFee)
			admin.PUT("/fees/:id", middleware.RequireScope(roles.WalletAdmin, roles.ListingAdmin), svc.UpdateFee)

			admin.GET("/notifications", svc.ListNotifications)
			admin.PUT("/notifications/:id/read", svc.MarkNotificationRead)
			admin.POST("/notifications/send", middleware.RequireScope(roles.WLClient, roles.MarketingAdmin), svc.SendNotification)
			admin.POST("/notifications/broadcast", middleware.RequireScope(roles.WLClient, roles.MarketingAdmin), svc.BroadcastNotification)

			admin.GET("/audit-logs", middleware.RequireScope(roles.ComplianceAdmin, roles.SecurityAdmin), svc.ListAuditLogs)
			admin.POST("/audit-logs/export", middleware.RequireScope(roles.ComplianceAdmin, roles.SecurityAdmin), svc.ExportAuditLogs)

			admin.GET("/sessions", svc.ListSessions)
			admin.DELETE("/sessions/:id", svc.RevokeSession)
			admin.DELETE("/sessions", svc.RevokeAllSessions)

			admin.GET("/feature-flags", svc.ListFeatureFlags)
			admin.POST("/feature-flags", middleware.RequireScope(roles.WLClient), svc.CreateFeatureFlag)
			admin.PUT("/feature-flags/:id", middleware.RequireScope(roles.WLClient), svc.UpdateFeatureFlag)
			admin.DELETE("/feature-flags/:id", middleware.RequireScope(roles.WLClient), svc.DeleteFeatureFlag)

			admin.GET("/ip-whitelist", middleware.RequireScope(roles.SecurityAdmin), svc.ListIPWhitelist)
			admin.POST("/ip-whitelist", middleware.RequireScope(roles.SecurityAdmin), svc.AddIPWhitelist)
			admin.DELETE("/ip-whitelist/:id", middleware.RequireScope(roles.SecurityAdmin), svc.RemoveIPWhitelist)

			admin.GET("/tickets", svc.ListTickets)
			admin.GET("/tickets/:id", svc.GetTicket)
			admin.POST("/tickets", svc.CreateTicket)
			admin.PUT("/tickets/:id/status", middleware.RequireScope(roles.CustomerServiceAdmin), svc.UpdateTicketStatus)
			admin.POST("/tickets/:id/messages", middleware.RequireScope(roles.CustomerServiceAdmin), svc.AddTicketMessage)
			admin.PUT("/tickets/:id/assign", middleware.RequireScope(roles.CustomerServiceAdmin), svc.AssignTicket)

			// ---- Trading admin domains (governance records; no fund movement) ----
			// Futures positions
			admin.GET("/futures", middleware.RequireScope(roles.TradingAdmin), svc.ListFuturesPositions)
			admin.GET("/futures/:id", middleware.RequireScope(roles.TradingAdmin), svc.GetFuturesPosition)
			admin.POST("/futures", middleware.RequireScope(roles.TradingAdmin), svc.CreateFuturesPosition)
			admin.PUT("/futures/:id", middleware.RequireScope(roles.TradingAdmin), svc.UpdateFuturesPosition)
			admin.DELETE("/futures/:id", middleware.RequireScope(roles.TradingAdmin), svc.DeleteFuturesPosition)
			admin.PUT("/futures/:id/status", middleware.RequireScope(roles.TradingAdmin), svc.UpdateFuturesPositionStatus)

			// Options contracts
			admin.GET("/options", middleware.RequireScope(roles.TradingAdmin), svc.ListOptionsContracts)
			admin.GET("/options/:id", middleware.RequireScope(roles.TradingAdmin), svc.GetOptionsContract)
			admin.POST("/options", middleware.RequireScope(roles.TradingAdmin), svc.CreateOptionsContract)
			admin.PUT("/options/:id", middleware.RequireScope(roles.TradingAdmin), svc.UpdateOptionsContract)
			admin.DELETE("/options/:id", middleware.RequireScope(roles.TradingAdmin), svc.DeleteOptionsContract)
			admin.PUT("/options/:id/status", middleware.RequireScope(roles.TradingAdmin), svc.UpdateOptionsContractStatus)

			// Copy-trading configs
			admin.GET("/copy-trading", middleware.RequireScope(roles.TradingAdmin), svc.ListCopyTradingConfigs)
			admin.GET("/copy-trading/:id", middleware.RequireScope(roles.TradingAdmin), svc.GetCopyTradingConfig)
			admin.POST("/copy-trading", middleware.RequireScope(roles.TradingAdmin), svc.CreateCopyTradingConfig)
			admin.PUT("/copy-trading/:id", middleware.RequireScope(roles.TradingAdmin), svc.UpdateCopyTradingConfig)
			admin.DELETE("/copy-trading/:id", middleware.RequireScope(roles.TradingAdmin), svc.DeleteCopyTradingConfig)
			admin.PUT("/copy-trading/:id/status", middleware.RequireScope(roles.TradingAdmin), svc.UpdateCopyTradingConfigStatus)

			// Convert orders
			admin.GET("/convert", middleware.RequireScope(roles.TradingAdmin), svc.ListConvertOrders)
			admin.GET("/convert/:id", middleware.RequireScope(roles.TradingAdmin), svc.GetConvertOrder)
			admin.POST("/convert", middleware.RequireScope(roles.TradingAdmin), svc.CreateConvertOrder)
			admin.PUT("/convert/:id", middleware.RequireScope(roles.TradingAdmin), svc.UpdateConvertOrder)
			admin.DELETE("/convert/:id", middleware.RequireScope(roles.TradingAdmin), svc.DeleteConvertOrder)
			admin.PUT("/convert/:id/status", middleware.RequireScope(roles.TradingAdmin), svc.UpdateConvertOrderStatus)

			// Onramp orders (approve/reject governance; no fund movement)
			admin.GET("/onramp", middleware.RequireScope(roles.P2PAdmin), svc.ListOnrampOrders)
			admin.GET("/onramp/:id", middleware.RequireScope(roles.P2PAdmin), svc.GetOnrampOrder)
			admin.POST("/onramp", middleware.RequireScope(roles.P2PAdmin), svc.CreateOnrampOrder)
			admin.PUT("/onramp/:id", middleware.RequireScope(roles.P2PAdmin), svc.UpdateOnrampOrder)
			admin.DELETE("/onramp/:id", middleware.RequireScope(roles.P2PAdmin), svc.DeleteOnrampOrder)
			admin.POST("/onramp/:id/approve", middleware.RequireScope(roles.P2PAdmin), svc.ApproveOnrampOrder)
			admin.POST("/onramp/:id/reject", middleware.RequireScope(roles.P2PAdmin), svc.RejectOnrampOrder)

			// Offramp orders (approve/reject governance; no fund movement)
			admin.GET("/offramp", middleware.RequireScope(roles.P2PAdmin), svc.ListOfframpOrders)
			admin.GET("/offramp/:id", middleware.RequireScope(roles.P2PAdmin), svc.GetOfframpOrder)
			admin.POST("/offramp", middleware.RequireScope(roles.P2PAdmin), svc.CreateOfframpOrder)
			admin.PUT("/offramp/:id", middleware.RequireScope(roles.P2PAdmin), svc.UpdateOfframpOrder)
			admin.DELETE("/offramp/:id", middleware.RequireScope(roles.P2PAdmin), svc.DeleteOfframpOrder)
			admin.POST("/offramp/:id/approve", middleware.RequireScope(roles.P2PAdmin), svc.ApproveOfframpOrder)
			admin.POST("/offramp/:id/reject", middleware.RequireScope(roles.P2PAdmin), svc.RejectOfframpOrder)

			// P2P clients
			admin.GET("/p2p-clients", middleware.RequireScope(roles.P2PAdmin), svc.ListP2PClients)
			admin.GET("/p2p-clients/:id", middleware.RequireScope(roles.P2PAdmin), svc.GetP2PClient)
			admin.POST("/p2p-clients", middleware.RequireScope(roles.P2PAdmin), svc.CreateP2PClient)
			admin.PUT("/p2p-clients/:id", middleware.RequireScope(roles.P2PAdmin), svc.UpdateP2PClient)
			admin.DELETE("/p2p-clients/:id", middleware.RequireScope(roles.P2PAdmin), svc.DeleteP2PClient)
			admin.PUT("/p2p-clients/:id/status", middleware.RequireScope(roles.P2PAdmin), svc.UpdateP2PClientStatus)

			// Partners (status + approve/reject governance)
			admin.GET("/partners", middleware.RequireScope(roles.ListingAdmin), svc.ListPartners)
			admin.GET("/partners/:id", middleware.RequireScope(roles.ListingAdmin), svc.GetPartner)
			admin.POST("/partners", middleware.RequireScope(roles.ListingAdmin), svc.CreatePartner)
			admin.PUT("/partners/:id", middleware.RequireScope(roles.ListingAdmin), svc.UpdatePartner)
			admin.DELETE("/partners/:id", middleware.RequireScope(roles.ListingAdmin), svc.DeletePartner)
			admin.PUT("/partners/:id/status", middleware.RequireScope(roles.ListingAdmin), svc.UpdatePartnerStatus)
			admin.POST("/partners/:id/approve", middleware.RequireScope(roles.ListingAdmin), svc.ApprovePartner)
			admin.POST("/partners/:id/reject", middleware.RequireScope(roles.ListingAdmin), svc.RejectPartner)

			// Reward campaigns
			admin.GET("/rewards", middleware.RequireScope(roles.RewardAdmin), svc.ListRewardCampaigns)
			admin.GET("/rewards/:id", middleware.RequireScope(roles.RewardAdmin), svc.GetRewardCampaign)
			admin.POST("/rewards", middleware.RequireScope(roles.RewardAdmin), svc.CreateRewardCampaign)
			admin.PUT("/rewards/:id", middleware.RequireScope(roles.RewardAdmin), svc.UpdateRewardCampaign)
			admin.DELETE("/rewards/:id", middleware.RequireScope(roles.RewardAdmin), svc.DeleteRewardCampaign)
			admin.PUT("/rewards/:id/status", middleware.RequireScope(roles.RewardAdmin), svc.UpdateRewardCampaignStatus)

			// Marketing campaigns
			admin.GET("/marketing", middleware.RequireScope(roles.MarketingAdmin), svc.ListMarketingCampaigns)
			admin.GET("/marketing/:id", middleware.RequireScope(roles.MarketingAdmin), svc.GetMarketingCampaign)
			admin.POST("/marketing", middleware.RequireScope(roles.MarketingAdmin), svc.CreateMarketingCampaign)
			admin.PUT("/marketing/:id", middleware.RequireScope(roles.MarketingAdmin), svc.UpdateMarketingCampaign)
			admin.DELETE("/marketing/:id", middleware.RequireScope(roles.MarketingAdmin), svc.DeleteMarketingCampaign)
			admin.PUT("/marketing/:id/status", middleware.RequireScope(roles.MarketingAdmin), svc.UpdateMarketingCampaignStatus)

			// ---- Structured RBAC (wl_client only) ----
			// Integrates with the existing scope system: roles bundle whitelisted
			// scope strings; assigning a role merges scopes into admin_users.scopes
			// so RequireScope (JWT scopes) keeps working unchanged.
			admin.GET("/admin-roles", middleware.RequireScope(roles.WLClient), svc.ListAdminRoles)
			admin.POST("/admin-roles", middleware.RequireScope(roles.WLClient), svc.CreateAdminRole)
			admin.GET("/admin-roles/:id", middleware.RequireScope(roles.WLClient), svc.GetAdminRole)
			admin.PUT("/admin-roles/:id", middleware.RequireScope(roles.WLClient), svc.UpdateAdminRole)
			admin.DELETE("/admin-roles/:id", middleware.RequireScope(roles.WLClient), svc.DeleteAdminRole)
			admin.GET("/admin-permissions", middleware.RequireScope(roles.WLClient), svc.ListAdminPermissions)
			admin.POST("/admin-permissions", middleware.RequireScope(roles.WLClient), svc.CreateAdminPermission)
			admin.POST("/admins/:id/role", middleware.RequireScope(roles.WLClient), svc.AssignAdminRole)
			admin.DELETE("/admins/:id/role/:roleId", middleware.RequireScope(roles.WLClient), svc.RevokeAdminRole)
			admin.GET("/admins/:id/permissions", middleware.RequireScope(roles.WLClient), svc.GetAdminPermissions)

			admin.GET("/stats", svc.Stats)
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
		log.Printf("White-Label Admin API starting on port %s", cfg.ServerPort)
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
