// wl-bots is the standalone white-label bot management backend. It runs
// INDEPENDENTLY in the WL client's own cloud — own PostgreSQL, own database —
// and phones home to the TigerWallet license control plane on heartbeat. Every
// protected request is gated fail-closed via wlgate. It is a standalone clone
// of the TigerWallet mm_bot_platform/bot_api platform.
//
// REAL PostgreSQL only. REAL crypto only (bcrypt passwords, JWT auth,
// AES-GCM at-rest key encryption). No stubs, no fakes, no mocks, no demos.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/tigerwallet/wl-bots/internal/config"
	"github.com/tigerwallet/wl-bots/internal/handlers"
	"github.com/tigerwallet/wl-bots/internal/store"
	"github.com/tigerwallet/wl-shared/wlgate"
)

func main() {
	cfg := config.Load()

	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required (WL client secret — used for auth tokens AND at-rest key encryption)")
	}
	if cfg.ControlPlaneURL == "" || cfg.LicenseKey == "" || cfg.WLClientID == "" {
		log.Fatal("TWO_PARTY_GATE_URL, WL_CLIENT_ID, and WL_LICENSE_KEY are required (license control plane phone-home)")
	}

	gate := wlgate.New()

	// Phone-home loop. Populates the license cache that the fail-closed
	// middleware consults on every protected request.
	go gate.HeartbeatLoop(context.Background(),
		cfg.ControlPlaneURL, cfg.ControlPlaneToken, cfg.LicenseKey,
		cfg.Product, cfg.InstanceID, cfg.HeartbeatInterval)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	s, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("store init: %v", err)
	}
	defer s.Close()

	svc := handlers.New(cfg, s, gate)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/health", svc.Health)

	// Public surface (no auth, no license gate) — mirrors canonical /api/v1/public.
	pub := r.Group("/api/v1/public")
	{
		pub.GET("/tiers", svc.PublicTiers)
	}

	api := r.Group("/api/v1")
	auth := api.Group("/auth")
	{
		auth.POST("/register", svc.Register)
		auth.POST("/login", svc.Login)
		// POST /auth/logout — audit-only stateless logout (records a real PG
		// audit event; JWT is not server-side invalidated). License-gated like
		// the other protected auth actions.
	}

	// Protected routes: every request must present a valid JWT AND pass the
	// fail-closed license gate. If the gate has no live license, the middleware
	// rejects the request (HTTP 402/503) before any business logic runs.
	protected := api.Group("")
	protected.Use(wlgate.JWTAuth(cfg.JWTSecret))
	protected.Use(gate.Middleware("bots", wlgate.CategoryFetcher))
	{
		// Stateless logout (audit-only) — sits under protected so the JWT is
		// validated and the license gate enforced before we record the event.
		protected.POST("/auth/logout", svc.Logout)

		// Bot CRUD + lifecycle (existing).
		protected.POST("/bots", svc.CreateBot)
		protected.POST("/bots/create", svc.CreateBot) // canonical alias of POST /bots
		protected.GET("/bots", svc.ListBots)
		protected.GET("/bots/instances", svc.ListBotInstances) // canonical alias of GET /bots
		protected.GET("/bots/me", svc.CurrentBotUser)
		protected.GET("/bots/transactions", svc.ListBotTransactions)
		protected.GET("/bots/:id", svc.GetBot)
		protected.DELETE("/bots/:id", svc.DeleteBot)
		protected.POST("/bots/:id/start", svc.StartBot)
		protected.POST("/bots/:id/stop", svc.StopBot)
		protected.POST("/bots/:id/pause", svc.PauseBot)
		protected.POST("/bots/:id/status", svc.SetBotStatus) // direct status set (distinct from lifecycle)
		protected.GET("/bots/:id/executions", svc.ListBotExecutions)
		protected.GET("/bots/:id/logs", svc.ListBotLogs)

		protected.POST("/subscriptions", svc.CreateSubscription)
		protected.GET("/subscriptions", svc.ListSubscriptions)
		protected.GET("/subscription", svc.GetSubscription) // canonical singular alias (current user)

		protected.POST("/fees", svc.CreateFeeConfig)
		protected.GET("/fees", svc.ListFeeConfigs)
		protected.PUT("/fees", svc.UpdateFeeConfig)

		// Per-user API keys — full CRUD. /api-keys and /keys both map to the
		// same api_keys table.
		protected.POST("/api-keys", svc.CreateApiKey)
		protected.GET("/api-keys", svc.ListApiKeys)
		protected.POST("/keys", svc.CreateApiKey)
		protected.GET("/keys", svc.ListApiKeys)
		protected.DELETE("/keys/:id", svc.DeleteApiKey)
	}

	// Admin/operator routes — protected (JWT + license gate) AND role-gated.
	// Mirrors canonical requireRole: stats, fee-addresses, cex/dex CRUD, and
	// user management require an admin role (super_admin / finance_admin /
	// bot_operator, depending on surface).
	admin := protected.Group("")
	{
		// Platform stats — real COUNT queries (super_admin / finance_admin).
		admin.GET("/stats", svc.RequireRole("super_admin", "finance_admin"), svc.Stats)

		// User management (wl_client / bot_admin — canonical scopes; legacy
		// super_admin/bot_operator role strings still honored via fallback).
		admin.GET("/users", svc.RequireRole("super_admin", "bot_operator"), svc.ListUsers)
		admin.POST("/bots/users", svc.RequireRole("super_admin", "bot_operator"), svc.CreateBotUser)
		admin.DELETE("/bots/users/:id", svc.RequireRole("super_admin"), svc.DeleteBotUser)
		admin.PUT("/users/:id/status", svc.RequireRole("super_admin", "bot_operator"), svc.UpdateUserStatus)
		// Scoped-admin role assignment — WL client owner only (wl_client scope).
		admin.PUT("/users/:id/scopes", svc.UpdateAdminScopes)

		// CEX connector configs — AES-GCM at rest (super_admin / finance_admin).
		admin.GET("/cex", svc.RequireRole("super_admin", "finance_admin"), svc.ListCEX)
		admin.POST("/cex", svc.RequireRole("super_admin", "finance_admin"), svc.CreateCEX)
		admin.DELETE("/cex/:id", svc.RequireRole("super_admin", "finance_admin"), svc.DeleteCEX)

		// DEX connector configs (super_admin / finance_admin).
		admin.GET("/dex", svc.RequireRole("super_admin", "finance_admin"), svc.ListDEX)
		admin.POST("/dex", svc.RequireRole("super_admin", "finance_admin"), svc.CreateDEX)
		admin.DELETE("/dex/:id", svc.RequireRole("super_admin", "finance_admin"), svc.DeleteDEX)

		// Admin fee-collection addresses (super_admin / finance_admin).
		admin.GET("/fee-addresses", svc.RequireRole("super_admin", "finance_admin"), svc.ListFeeAddresses)
		admin.POST("/fee-addresses", svc.RequireRole("super_admin", "finance_admin"), svc.CreateFeeAddress)
		admin.DELETE("/fee-addresses/:id", svc.RequireRole("super_admin", "finance_admin"), svc.DeleteFeeAddress)
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("wl-bots listening on :%s (product=%s wl_client=%s)", cfg.Port, cfg.Product, cfg.WLClientID)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")
	shutdownCtx, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
