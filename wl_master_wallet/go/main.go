// Standalone WL-MasterWallet backend entry point. Runs INDEPENDENTLY in the WL
// client's own cloud — own PG, own signing, own DB. Phones home to the license
// control plane on heartbeat; gates every request fail-closed via wlgate.
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
	"github.com/tigerwallet/wl-master-wallet/internal/config"
	"github.com/tigerwallet/wl-master-wallet/internal/handlers"
	"github.com/tigerwallet/wl-master-wallet/internal/store"
	"github.com/tigerwallet/wl-shared/wlgate"
)

func main() {
	cfg := config.Load()
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}
	if cfg.WLClientID == "" {
		log.Fatal("WL_CLIENT_ID environment variable is required (your white-label client UUID)")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer st.Close()
	log.Println("Database initialized (standalone wl_masterwallet)")

	// Fail-closed license gate. Starts dead until the first successful heartbeat
	// validates the license against the SuperAdmin control plane.
	gate := wlgate.New()

	// Phone home to the license control plane at the heartbeat interval. If the
	// control plane is unreachable or the license is invalid/suspended, the gate
	// goes dead and every protected request 503s.
	go gate.HeartbeatLoop(ctx, cfg.ControlPlaneURL, cfg.ControlPlaneToken, cfg.LicenseKey, cfg.Product, cfg.InstanceID, cfg.HeartbeatInterval)

	svc := handlers.New(cfg, st, gate)

	router := gin.Default()
	router.Use(cors.Default())
	router.GET("/health", svc.Health)

	// Public (read-only) market-data endpoints — no license gate, no auth.
	router.GET("/api/v1/health", svc.PublicHealth)
	router.GET("/api/v1/chains", svc.PublicChains)
	router.GET("/api/v1/gas", svc.PublicGas)
	router.GET("/api/v1/price", svc.PublicPrice)
	router.GET("/api/v1/transactions/history", svc.PublicTransactionHistory)

	// WebSocket — real gorilla/websocket (JWT verified per-connection).
	router.GET("/ws", svc.WebSocket)

	api := router.Group("/api/v1")
	{
		api.POST("/auth/register", svc.Register)
		api.POST("/auth/login", svc.Login)

		// Every protected route is gated by JWT auth + the fail-closed license
		// gate (503 when the product is not authorized or a fetcher is disabled).
		mw := api.Group("")
		mw.Use(wlgate.JWTAuth(cfg.JWTSecret))
		mw.Use(gate.Middleware(cfg.Product, wlgate.CategoryFetcher))
		{
			// Master-wallet CRUD + core operations.
			mw.POST("/master-wallet", svc.CreateMasterWallet)
			mw.GET("/master-wallet", svc.ListMasterWallets)
			mw.GET("/master-wallet/:id", svc.GetMasterWallet)
			mw.DELETE("/master-wallet/:id", svc.DeleteMasterWallet)
			mw.GET("/master-wallet/:id/balance", svc.GetBalance)
			mw.POST("/master-wallet/:id/sign", svc.SignTransaction)
			mw.POST("/master-wallet/:id/sign-message", svc.SignMessage)
			mw.POST("/master-wallet/:id/revenue-payout", svc.RevenuePayout)
			mw.POST("/master-wallet/:id/withdrawal-request", svc.WithdrawalRequest)

			// Transactions.
			mw.GET("/master-wallet/:id/transactions", svc.ListTransactions)
			mw.POST("/master-wallet/:id/transactions", svc.CreatePendingTransaction)
			mw.POST("/master-wallet/:id/transactions/:tid/approve", svc.ApproveTransaction)
			mw.POST("/master-wallet/:id/transactions/:tid/reject", svc.RejectTransaction)
			mw.POST("/transactions/:tid/execute", svc.ExecuteTransaction)
			mw.POST("/transactions/:tid/sign", svc.SignPendingTransaction)

			// Sub-wallets (balance + transfer use real ethclient + EIP-1559).
			mw.POST("/master-wallet/:id/sub-wallets", svc.CreateSubWallet)
			mw.GET("/master-wallet/:id/sub-wallets", svc.ListSubWallets)
			mw.GET("/master-wallet/:id/sub-wallets/:sid/balance", svc.GetSubWalletBalance)
			mw.POST("/master-wallet/:id/sub-wallets/:sid/transfer", svc.TransferFromSubWallet)

			// Policies / fees / auto-sign rules.
			mw.POST("/master-wallet/:id/policies", svc.CreatePolicy)
			mw.GET("/master-wallet/:id/policies", svc.ListPolicies)
			mw.DELETE("/master-wallet/:id/policies/:pid", svc.DeletePolicy)
			mw.POST("/master-wallet/:id/fees", svc.CreateFeeConfig)
			mw.GET("/master-wallet/:id/fees", svc.ListFeeConfigs)
			mw.DELETE("/master-wallet/:id/fees/:fid", svc.DeleteFeeConfig)
			mw.POST("/master-wallet/:id/auto-sign", svc.CreateAutoSignRule)
			mw.GET("/master-wallet/:id/auto-sign", svc.ListAutoSignRules)
			mw.DELETE("/master-wallet/:id/auto-sign/:rid", svc.DeleteAutoSignRule)

			// Users (admin-only writes — role gate enforces).
			mw.GET("/master-wallet/:id/users", svc.ListMasterWalletUsers)
			mw.POST("/master-wallet/:id/users", svc.CreateMasterWalletUser)
			mw.DELETE("/master-wallet/:id/users/:uid", svc.DeleteMasterWalletUser)

			// Scoped-admin role assignment — WL client owner only (wl_client scope).
			mw.PUT("/users/:id/scopes", svc.UpdateAdminScopes)

			// Analytics (real SQL aggregates).
			mw.GET("/master-wallet/:id/analytics/transactions", svc.AnalyticsTransactions)
			mw.GET("/master-wallet/:id/analytics/volume", svc.AnalyticsVolume)
			mw.GET("/master-wallet/:id/analytics/wallets", svc.AnalyticsWallets)

			// Notifications / Webhooks / Audit.
			mw.GET("/master-wallet/:id/notifications", svc.ListNotifications)
			mw.POST("/master-wallet/:id/notifications", svc.CreateNotification)
			mw.GET("/master-wallet/:id/webhooks", svc.ListWebhooks)
			mw.POST("/master-wallet/:id/webhooks", svc.CreateWebhook)
			mw.DELETE("/master-wallet/:id/webhooks/:wid", svc.DeleteWebhook)
			mw.GET("/master-wallet/:id/audit", svc.AuditLog)

			// Auto-sign (transaction execution + logs).
			mw.POST("/auto-sign-transaction", svc.AutoSignTransaction)
			mw.GET("/auto-sign-logs", svc.ListAutoSignLogs)

			// Multisig wallets (threshold-governed wallets + tx flow).
			mw.POST("/master-wallet/:id/multisig", svc.CreateMultisigWallet)
			mw.GET("/master-wallet/:id/multisig", svc.GetMultisigWallets)
			mw.POST("/master-wallet/:id/multisig/:wid/transactions", svc.CreateMultisigTransaction)
			mw.GET("/master-wallet/:id/multisig/:wid/transactions", svc.GetMultisigTransactions)
			mw.POST("/multisig/transactions/:tid/sign", svc.SignMultisigTransaction)
			mw.POST("/multisig/transactions/:tid/execute", svc.ExecuteMultisigTransaction)

			// Treasury (two-party gate REQUIRED before broadcast — fail-closed).
			mw.POST("/transfer", svc.TreasuryTransfer)
			mw.POST("/sweep", svc.TreasurySweep)

			// UserWallet-management governance layer.
			mw.GET("/user-chains/evm", svc.ListUserChainsEVM)
			mw.POST("/user-chains/evm", svc.CreateUserChainEVM)
			mw.PUT("/user-chains/evm/:chainId", svc.UpdateUserChainEVM)
			mw.DELETE("/user-chains/evm/:chainId", svc.DeleteUserChainEVM)
			mw.GET("/user-chains/nonevm", svc.ListUserChainsNonEVM)
			mw.POST("/user-chains/nonevm", svc.CreateUserChainNonEVM)
			mw.PUT("/user-chains/nonevm/:chainId", svc.UpdateUserChainNonEVM)
			mw.DELETE("/user-chains/nonevm/:chainId", svc.DeleteUserChainNonEVM)
			mw.GET("/user-tokens", svc.ListUserTokens)
			mw.POST("/user-tokens", svc.CreateUserToken)
			mw.PUT("/user-tokens/:tokenId", svc.UpdateUserToken)
			mw.DELETE("/user-tokens/:tokenId", svc.DeleteUserToken)
			mw.GET("/user-wallet-addresses", svc.ListUserWalletAddresses)
			mw.POST("/derive-user-address", svc.DeriveUserAddress)

			// Feature-flag governance (full CRUD).
			mw.GET("/feature-flags/:flagId", svc.GetFeatureFlag)
			mw.PUT("/feature-flags/:flagId", svc.UpsertFeatureFlag)
			mw.DELETE("/feature-flags/:flagId", svc.DeleteFeatureFlag)
		}
	}

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	go func() {
		log.Printf("Standalone WL-MasterWallet API starting on port %s (WL client %s)", cfg.Port, cfg.WLClientID)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")
	cancel()
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	_ = srv.Shutdown(shutCtx)
	log.Println("Server exited")
}
