// Standalone WL-ProjectParty backend entry point. Runs INDEPENDENTLY in the WL
// client's own cloud — own PG, own DB. Phones home to the license control
// plane on heartbeat; gates every request fail-closed via wlgate. A clone of
// the TigerWallet project_party token-listing / launchpad platform.
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
	"github.com/tigerwallet/wl-project-party/internal/config"
	"github.com/tigerwallet/wl-project-party/internal/handlers"
	"github.com/tigerwallet/wl-project-party/internal/store"
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
	log.Println("Database initialized (standalone wl_projectparty)")

	// Fail-closed license gate. Starts dead until the first successful heartbeat
	// validates the license against the SuperAdmin control plane.
	gate := wlgate.New()

	// Phone home to the license control plane at the heartbeat interval. If the
	// control plane is unreachable or the license is invalid/suspended, the gate
	// goes dead and every protected request 503s.
	go gate.HeartbeatLoop(ctx, cfg.ControlPlaneURL, cfg.ControlPlaneToken, cfg.LicenseKey, cfg.Product, cfg.InstanceID, cfg.HeartbeatInterval)

	svc := handlers.New(cfg, st, gate)

	router := buildRouter(cfg, svc, gate)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	go func() {
		log.Printf("Standalone WL-ProjectParty API starting on port %s (WL client %s)", cfg.Port, cfg.WLClientID)
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

// buildRouter wires the full REST route tree. Extracted from main so it can be
// exercised by tests without a live PostgreSQL connection — route registration
// (and any httprouter conflict panics) happen here, before any DB call.
func buildRouter(cfg *config.Config, svc *handlers.Handlers, gate *wlgate.Gate) *gin.Engine {
	router := gin.Default()
	router.Use(cors.Default())
	router.GET("/health", svc.Health)

	api := router.Group("/api/v1")
	{
		api.POST("/auth/register", svc.Register)
		api.POST("/auth/login", svc.Login)

		// Public discovery reads (no JWT required, matching the canonical
		// frontend's public market pages). Real PostgreSQL queries only.
		api.GET("/coins", svc.ListCoins)
		api.GET("/search", svc.SearchTokens)
		api.GET("/featured", svc.FeaturedTokens)
		api.GET("/trending", svc.TrendingTokens)
		api.GET("/market", svc.MarketOverview)

		// Every protected route is gated by JWT auth + the fail-closed license
		// gate (503 when the product is not authorized or a fetcher is disabled).
		mw := api.Group("")
		mw.Use(wlgate.JWTAuth(cfg.JWTSecret))
		mw.Use(gate.Middleware(cfg.Product, wlgate.CategoryFetcher))
		{
			mw.POST("/tokens", svc.CreateToken)
			mw.GET("/tokens", svc.ListTokens)
			mw.GET("/tokens/:id", svc.GetToken)
			mw.PUT("/tokens/:id", svc.UpdateToken)
			mw.DELETE("/tokens/:id", svc.DeleteToken)

			// Token listing workflow (canonical tokens/:id/* path shapes).
			// submit + status are user-facing; approve/reject/featured are
			// admin-gated and registered in the admin group below.
			mw.POST("/tokens/:id/submit", svc.SubmitToken)
			// Flat aliases the rebranded frontend also hits.
			mw.GET("/status/:token_id", svc.TokenListingStatus)
			mw.GET("/:id", svc.GetToken)

			mw.POST("/listings", svc.CreateListing)
			mw.GET("/listings", svc.ListListings)

			mw.POST("/launchpad", svc.CreateLaunchpadProject)
			mw.GET("/launchpad", svc.ListLaunchpadProjects)
			mw.GET("/launchpad/:id", svc.GetLaunchpadProject)
			mw.POST("/launchpad/:id/participate", svc.ParticipateInLaunchpad)
			mw.GET("/launchpad/:id/participations", svc.ListParticipations)

			// Launchpad contribution workflow (canonical launchpad/:id/* shapes).
			mw.POST("/launchpad/:id/contribute", svc.Contribute)
			mw.POST("/launchpad/:id/claim", svc.Claim)
			mw.POST("/launchpad/:id/cancel", svc.CancelContribution)
			mw.GET("/launchpad/:id/contribution-status", svc.GetContributionStatus)
			// Flat alias: contribution history for a token.
			mw.GET("/history/:token_id", svc.ContributionHistory)

			// Market-making (canonical marketmaking/* + flat /orders aliases).
			mw.GET("/orders", svc.ListMMOrders)
			mw.POST("/orders", svc.CreateMMOrder)
			mw.PUT("/orders/:id/status", svc.UpdateMMOrderStatus)
			mw.GET("/marketmaking/orders", svc.ListMMOrders)
			mw.POST("/marketmaking/orders", svc.CreateMMOrder)
			mw.PUT("/marketmaking/orders/:id/status", svc.UpdateMMOrderStatus)
			mw.GET("/marketmaking/status/:token_id", svc.MarketMakerStatus)
			mw.POST("/marketmaking/liquidity/add", svc.AddLiquidity)
			mw.POST("/marketmaking/liquidity/remove", svc.RemoveLiquidity)

			// Flat liquidity aliases.
			mw.GET("/liquidity", svc.LiquidityState)
			mw.POST("/liquidity/add", svc.AddLiquidity)
			mw.POST("/liquidity/remove", svc.RemoveLiquidity)

			// Pricing (canonical pricing/* shapes). set/update are admin-gated
			// (admin group); get + history are open to authenticated users.
			mw.GET("/pricing/:token_id", svc.GetTokenPrice)
			mw.GET("/pricing/history/:token_id", svc.PriceHistory)

			// Analytics (canonical analytics/* shapes + flat aliases).
			mw.GET("/volume", svc.VolumeStats)
			mw.GET("/transactions", svc.TransactionCount)
			mw.GET("/holders", svc.HolderCount)
			mw.GET("/analytics/volume", svc.VolumeStats)
			mw.GET("/analytics/liquidity", svc.LiquidityState)
			mw.GET("/analytics/holders", svc.HolderCount)
			mw.GET("/analytics/transactions", svc.TransactionCount)

			// Compliance: audit + KYC (canonical compliance/* shapes).
			// audit-create is admin-gated (admin group); reads + kyc/submit are open.
			mw.GET("/audit/:token_id", svc.AuditStatus)
			mw.POST("/kyc/submit", svc.SubmitKYC)
			mw.GET("/kyc/:token_id", svc.KYCStatus)
			mw.GET("/compliance/audit/:token_id", svc.AuditStatus)
			mw.POST("/compliance/kyc/submit", svc.SubmitKYC)
			mw.GET("/compliance/kyc/:token_id", svc.KYCStatus)

			// Fees (canonical fees/* shapes + flat aliases).
			// set/update are admin-gated (admin group); calculate/pay/list are open.
			mw.GET("/fees", svc.ListFees)
			mw.POST("/fees/calculate", svc.CalculateFees)
			mw.POST("/fees/pay", svc.PayFees)
			mw.GET("/calculate", svc.CalculateFees)
			mw.POST("/pay", svc.PayFees)

			mw.POST("/market-making", svc.CreateMarketMakingConfig)
			mw.GET("/market-making", svc.ListMarketMakingConfigs)
			// Plural /configs alias (used by bot_api getMMConfigs + web).
			mw.POST("/market-making/configs", svc.CreateMarketMakingConfig)
			mw.GET("/market-making/configs", svc.ListMarketMakingConfigs)
			mw.DELETE("/market-making/configs/:id", svc.DeleteMarketMakingConfig)
			// Query-param pricing form + aggregated compliance status.
			mw.GET("/pricing", svc.GetTokenPriceQuery)
			mw.GET("/compliance/status/:token_id", svc.ComplianceStatus)

			mw.POST("/fees", svc.CreateFeeConfig)
			mw.GET("/fees/configs", svc.ListFeeConfigs)

			mw.POST("/favorites", svc.AddFavorite)
			mw.GET("/favorites", svc.ListFavorites)
			mw.DELETE("/favorites/:id", svc.RemoveFavorite)
		}

		// Admin-gated routes: JWT + license gate + RequireRole("admin","super_admin").
		// Role is read from the real users.role column (no stub). Registered on a
		// sibling group so the role middleware composes after JWTAuth + gate.
		admin := api.Group("")
		admin.Use(wlgate.JWTAuth(cfg.JWTSecret))
		admin.Use(gate.Middleware(cfg.Product, wlgate.CategoryFetcher))
		admin.Use(svc.RequireRole("admin", "super_admin"))
		{
			admin.POST("/tokens/:id/approve", svc.ApproveToken)
			admin.POST("/tokens/:id/reject", svc.RejectToken)
			admin.POST("/tokens/:id/featured", svc.ToggleFeatured)
			admin.POST("/tokens/:id/verify-contract", svc.VerifyTokenContract)
			admin.POST("/fees/set", svc.SetFeeConfig)
			admin.PUT("/fees/set", svc.SetFeeConfig)
			admin.POST("/fees/update", svc.UpdateFeeConfig)
			admin.PUT("/fees/update", svc.UpdateFeeConfig)
			admin.POST("/set", svc.SetFeeConfig)
			admin.POST("/update", svc.UpdateFeeConfig)
			admin.POST("/audit", svc.CreateAuditLog)
			admin.POST("/compliance/audit", svc.CreateAuditLog)
			admin.POST("/pricing/set", svc.SetTokenPrice)
			admin.POST("/pricing/update", svc.UpdatePrice)
			// Fee verification: real on-chain receipt check before a pending
			// fee payment counts as paid.
			admin.POST("/fees/verify/:id", svc.VerifyFeePayment)
			admin.GET("/fees/payments", svc.ListFeePayments)
			// Scoped-admin role assignment — WL client owner only (wl_client
			// scope). RequireRole passes wl_client (full tenancy control); the
			// handler re-checks HasScope("wl_client") so a listing_admin cannot
			// escalate.
			admin.PUT("/users/:id/scopes", svc.UpdateAdminScopes)
		}
	}
	return router
}
