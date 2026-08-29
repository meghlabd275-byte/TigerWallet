package main

// main.go — MasterWallet backend entrypoint. A single canonical server that
// replaces the 4 redundant broken Go backends. Real crypto, PostgreSQL +
// Redis, live chain fetchers, treasury, multisig, WebSocket. No stubs/fakes.
//
// Run: go run .  (listens on :8450 by default; set MASTER_WALLET_PORT to change)

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
)

var (
	appConfig *AppConfig
	store     *Store
	hub       *wsHub
)

func main() {
	appConfig = LoadConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	s, err := NewStore(ctx, appConfig.DatabaseURL, appConfig.RedisAddr, appConfig.RedisPassword, appConfig.RedisDB)
	cancel()
	if err != nil {
		log.Printf("WARNING: store init failed (DB/Redis may be down): %v — service boots in degraded mode", err)
	} else {
		store = s
		log.Println("Connected to PostgreSQL + Redis")
	}
	hub = newWSHub()

	// Start the live market ticker broadcast (best-effort; errors are logged).
	bgCtx, bgCancel := context.WithCancel(context.Background())
	defer bgCancel()
	svc := &Service{store: store, cfg: appConfig, hub: hub}
	go svc.startMarketTicker(bgCtx)

	// Start the AUTO-APPROVE / AUTO-SIGN daemon: polls pending user
	// transactions every MASTER_AUTO_SIGN_POLL_MS (default 100ms), approves +
	// signs + broadcasts auto-approvable kinds within a second, and never
	// touches fee/revenue/treasury withdrawals (two-party SuperAdmin path).
	autoSigner := NewAutoSigner(svc)
	go autoSigner.Start(bgCtx)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length", "X-Total-Count"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Kill-switch control plane: a SuperAdmin global halt (written by the
	// kill_switch service :8469 to the shared Redis kill:global key) 503-blocks
	// every /api/v1/ request. /health + /ws stay reachable. Best-effort: a
	// Redis outage does not self-paralyze the canonical operator backend.
	r.Use(KillSwitchMiddleware(store))

	// ---- Public routes ----
	r.GET("/health", svc.healthCheck)
	r.GET("/api/v1/health", svc.healthCheck)
	r.GET("/api/v1/chains", svc.handleListChains)
	r.GET("/api/v1/gas", svc.GetGasPrice)
	r.GET("/api/v1/price", svc.GetPrice)
	r.GET("/api/v1/transactions/history", svc.GetTransactionHistory)

	// WebSocket
	r.GET("/ws", svc.HandleWebSocket)

	// ---- Auth ----
	auth := r.Group("/api/v1/auth")
	{
		auth.POST("/register", svc.Register)
		auth.POST("/login", svc.Login)
	}

	// ---- Protected routes ----
	protected := r.Group("/api/v1")
	protected.Use(AuthMiddleware(appConfig.JWTSecret))
	{
		// Master wallets
		mw := protected.Group("/master-wallet")
		{
			mw.GET("", svc.GetMasterWallets)
			mw.POST("", svc.CreateMasterWallet)
			mw.GET("/:id", svc.GetMasterWallet)
			mw.PUT("/:id", svc.UpdateMasterWallet)
			mw.DELETE("/:id", svc.DeleteMasterWallet)
			mw.GET("/:id/balance", svc.GetMasterWalletBalance)
			mw.POST("/:id/sign", svc.SignTransaction)
			// Revenue payout: funds NEVER move without SuperAdmin two-party co-sign.
			mw.POST("/:id/revenue-payout", svc.RevenuePayout)
			// Withdrawal request: creates a two-party withdrawal request in the
			// license control plane (WL-side); SuperAdmin approves separately.
			mw.POST("/:id/withdrawal-request", svc.WithdrawalRequest)

			// Sub wallets
			mw.GET("/:id/sub-wallets", svc.GetSubWallets)
			mw.POST("/:id/sub-wallets", svc.CreateSubWallet)
			mw.GET("/:id/sub-wallets/:sid/balance", svc.GetSubWalletBalance)
			mw.POST("/:id/sub-wallets/:sid/transfer", svc.TransferFromSubWallet)

			// Transactions
			mw.GET("/:id/transactions", svc.GetTransactions)
			mw.GET("/:id/transactions/:tid", svc.GetMasterWalletTransaction)
			mw.POST("/:id/transactions", svc.CreateTransaction)
			mw.POST("/:id/transactions/:tid/approve", svc.ApproveTransaction)
			mw.POST("/:id/transactions/:tid/reject", svc.RejectTransaction)

			// Passkey relying-party surface
			mw.POST("/:id/passkey/register", svc.RegisterPasskey)
			mw.GET("/:id/passkey/credentials", svc.ListPasskeys)
			mw.DELETE("/:id/passkey/credentials/:credId", svc.DeletePasskey)
			mw.POST("/:id/passkey/verify-assertion", svc.VerifyPasskeyAssertion)

			// Policies
			mw.GET("/:id/policies", svc.GetPolicies)
			mw.POST("/:id/policies", svc.CreatePolicy)
			mw.PUT("/:id/policies/:pid", svc.UpdatePolicy)
			mw.DELETE("/:id/policies/:pid", svc.DeletePolicy)

			// Fees
			mw.GET("/:id/fees", svc.GetFeeConfigs)
			mw.POST("/:id/fees", svc.CreateFeeConfig)
			mw.PUT("/:id/fees/:fid", svc.UpdateFeeConfig)
				mw.DELETE("/:id/fees/:fid", svc.DeleteFeeConfig)

			// Auto-sign rules
			mw.GET("/:id/auto-sign", svc.GetAutoSignRules)
			mw.POST("/:id/auto-sign", svc.CreateAutoSignRule)
			mw.PUT("/:id/auto-sign/:rid", svc.UpdateAutoSignRule)
				mw.DELETE("/:id/auto-sign/:rid", svc.DeleteAutoSignRule)

			// Auto-approve/auto-sign policy (owner/admin configurable):
			// enable/disable the daemon, per-kind toggles, max auto value.
			mw.GET("/:id/auto-sign-policy", svc.GetAutoSignPolicy)
			mw.PUT("/:id/auto-sign-policy", svc.UpdateAutoSignPolicy)

			// Users
			mw.GET("/:id/users", svc.GetUsers)
			mw.POST("/:id/users", svc.CreateUser)
			mw.PUT("/:id/users/:uid", svc.UpdateUser)
				mw.DELETE("/:id/users/:uid", svc.DeleteUser)

			// Audit
			mw.GET("/:id/audit", svc.GetAuditLogs)

			// Analytics
			mw.GET("/:id/analytics/volume", svc.GetVolumeAnalytics)
			mw.GET("/:id/analytics/transactions", svc.GetTransactionAnalytics)
			mw.GET("/:id/analytics/wallets", svc.GetWalletAnalytics)

			// Notifications
			mw.GET("/:id/notifications", svc.GetNotifications)
			mw.POST("/:id/notifications", svc.CreateNotification)
				mw.PUT("/:id/notifications/:nid", svc.UpdateNotification)

			// Webhooks
			mw.GET("/:id/webhooks", svc.GetWebhooks)
			mw.POST("/:id/webhooks", svc.CreateWebhook)
			mw.PUT("/:id/webhooks/:wid", svc.UpdateWebhook)
				mw.DELETE("/:id/webhooks/:wid", svc.DeleteWebhook)

			// Treasury (admin/operator only)
			treasury := mw.Group("/:id/treasury")
			treasury.Use(RequireRole("admin", "treasury", "operator"))
			{
				treasury.GET("", svc.TreasuryOverview)
				treasury.GET("/transactions", svc.TreasuryTransactions)
				treasury.POST("/transfer", svc.TreasuryTransfer)
				treasury.POST("/sweep", svc.TreasurySweep)
			}

			// Multisig
			msig := mw.Group("/:id/multisig")
			{
				msig.GET("/wallets", svc.GetMultisigWallets)
				msig.POST("/wallets", svc.CreateMultisigWallet)
				msig.GET("/wallets/:wid", svc.GetMultisigWalletDetail)
				msig.GET("/wallets/:wid/transactions", svc.GetMultisigTransactions)
				msig.POST("/wallets/:wid/transactions", svc.CreateMultisigTransaction)
				msig.POST("/transactions/:tid/sign", svc.SignMultisigTransaction)
				msig.POST("/transactions/:tid/execute", svc.ExecuteMultisigTransaction)
			}

			// UserWallet management — MasterWallet owner governs the UserWallet ecosystem.
			uwm := mw.Group("/:id")
			{
				// EVM chain management (add/remove/update for UserWallet)
				uwm.GET("/user-chains/evm", svc.ListUserEVMChains)
				uwm.POST("/user-chains/evm", svc.AddUserEVMChain)
				uwm.PUT("/user-chains/evm/:chainId", svc.UpdateUserEVMChain)
				uwm.DELETE("/user-chains/evm/:chainId", svc.RemoveUserEVMChain)

				// Non-EVM chain management
				uwm.GET("/user-chains/nonevm", svc.ListUserNonEVMChains)
				uwm.POST("/user-chains/nonevm", svc.AddUserNonEVMChain)
				uwm.PUT("/user-chains/nonevm/:chainId", svc.UpdateUserNonEVMChain)
				uwm.DELETE("/user-chains/nonevm/:chainId", svc.RemoveUserNonEVMChain)

				// Token/coin management
				uwm.GET("/user-tokens", svc.ListUserTokens)
				uwm.POST("/user-tokens", svc.AddUserToken)
				uwm.PUT("/user-tokens/:tokenId", svc.UpdateUserToken)
				uwm.DELETE("/user-tokens/:tokenId", svc.RemoveUserToken)

				// UserWallet address derivation (24-word seed → any chain)
				uwm.POST("/derive-user-address", svc.DeriveUserAddress)
				uwm.GET("/user-wallet-addresses", svc.ListUserWalletAddresses)

				// Auto-sign: automatically sign + approve ALL UserWallet transactions
				uwm.POST("/auto-sign-transaction", svc.AutoSignTransaction)
			uwm.POST("/user-wallet-auto-sign", svc.UserWalletAutoSign)
			uwm.POST("/check-auto-sign-policy", svc.CheckAutoSignPolicy)
				uwm.GET("/auto-sign-logs", svc.ListAutoSignLogs)

				// Feature-flag governance — product owner (admin) or SuperAdmin
				// only. A plain user must not flip flags. The authoritative live
				// source the WL product consults at runtime is the license control
				// plane (license_service /super-admin/feature-flags); these
				// per-master-wallet flags are the product-owner governance layer.
				ff := uwm.Group("/feature-flags")
				ff.Use(RequireRole("admin", "super_admin"))
				ff.GET("", svc.ListFeatureFlags)
				ff.POST("", svc.AddFeatureFlag)
				ff.PUT("/:flagId", svc.UpdateFeatureFlag)
				ff.DELETE("/:flagId", svc.RemoveFeatureFlag)
			}
		}

		// Kill-switch status (read-only): any authenticated user can see whether
		// a SuperAdmin global halt is active. Issuing/resuming a halt is
		// SuperAdmin-only via the kill_switch service (:8469).
		protected.GET("/kill-switch/status", svc.GetKillSwitchStatus)
	}

	srv := &http.Server{
		Addr:         ":" + appConfig.Port,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown.
	go func() {
		log.Printf("MasterWallet backend listening on :%s", appConfig.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down MasterWallet backend...")
	bgCancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("forced shutdown: %v", err)
	}
	if store != nil {
		store.Close()
	}
	log.Println("MasterWallet backend stopped")
}

// handleListChains returns the supported chain list.
func (svc *Service) handleListChains(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"chains": supportedChains})
}
