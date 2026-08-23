// Command license-service is the TigerWallet SuperAdmin license & kill-switch
// control plane. It is the ONLY authority that can authorize an externally-
// hosted white-label product to run. Without a valid, active license + a live
// heartbeat to this service, every WL product fails closed (all routes 403).
//
// Language placement: Go (high-load, distributed, PostgreSQL + Redis). The
// cryptographic license-token signing (Ed25519) is in internal/crypto. The
// ultra-low-latency per-request checker that WL products embed is the C++ shared
// library (wl_control_plane/cpp). The fail-closed heartbeat client + token
// verifier that WL products embed is the Rust SDK (white_label/sdk/rust).
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
	"github.com/tigerwallet/license-service/internal/config"
	"github.com/tigerwallet/license-service/internal/crypto"
	"github.com/tigerwallet/license-service/internal/handlers"
	"github.com/tigerwallet/license-service/internal/store"
)

func main() {
	cfg := config.Load()

	st, err := store.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("store init: %v", err)
	}
	defer st.Close()

	// Bootstrap the first SuperAdmin (idempotent).
	ctxBoot, cancelBoot := context.WithTimeout(context.Background(), 10*time.Second)
	if err := handlers.BootstrapSuperAdmin(ctxBoot, st, cfg.SuperAdminEmail, cfg.SuperAdminPassword); err != nil {
		log.Printf("WARN: bootstrap superadmin: %v", err)
	}
	cancelBoot()

	hub := store.NewHub(cfg.RedisAddr, cfg.RedisPassword)
	defer hub.Close()

	keys, err := crypto.LoadKeyPair(cfg.LicenseSignKeyHex, cfg.LicenseVerifyKeyHex)
	if err != nil {
		log.Fatalf("load license key pair: %v", err)
	}
	log.Printf("License verify key (distribute to WL products): %s", keys.PublicKeyHex())

	h := handlers.New(cfg, st, hub, keys)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-API-Key"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/health", h.Health)

	api := r.Group("/api/v1")
	{
		// Public auth endpoints.
		api.POST("/auth/login", h.Login)            // SuperAdmin login
		api.POST("/auth/wl-login", h.WLClientLogin) // WL client login (scoped JWT)

		// PUBLIC branding endpoint — no auth. A WL-branded app fetches its
		// branding config on startup by slug (injected at build time). Returns
		// 404 when no WL client matches so the app falls back to TigerWallet.
		api.GET("/branding/:slug", h.GetBrandingBySlug)

		// WL-product-facing endpoints (called by the Rust SDK / C++ checker).
		// These authenticate via the license_key in the body, not a JWT.
		api.POST("/license/validate", h.ValidateLicense)
		api.POST("/license/heartbeat", h.Heartbeat)
		api.POST("/license/command/:id/ack", h.CommandAck)
		api.GET("/license/fetcher-enabled", h.IsFetcherEnabled)

		// Authenticated area.
		authed := api.Group("")
		authed.Use(handlers.AuthMiddleware(cfg))

		// Two-party withdrawal collaboration — SERVICE endpoints called by the
		// wl_master_wallet / wl_user_wallet backends (machine-to-machine via
		// SERVICE_AUTH_TOKEN + X-WL-Client-ID header) OR by a human WL client /
		// SuperAdmin via JWT. These are the gate calls:
		//   POST /withdrawals/request         -> WL side files a withdrawal
		//   GET  /withdrawals/:id/approved    -> gate check before broadcast
		//   POST /withdrawals/:id/executed    -> record tx hash after broadcast
		// The SuperAdmin co-sign actions (list / approve / reject) live in the
		// sa group below and are JWT+RequireSuperAdmin-gated (a service token
		// can NEVER approve — only a human SuperAdmin can).
		twoParty := api.Group("")
		twoParty.Use(handlers.ServiceOrUserAuth(cfg))
		twoParty.Use(handlers.RequireWLClientOrSuperAdmin())
		{
			twoParty.POST("/withdrawals/request", h.RequestWithdrawal)
			twoParty.GET("/withdrawals/:id/approved", h.IsWithdrawalApproved)
			twoParty.POST("/withdrawals/:id/executed", h.MarkWithdrawalExecuted)
		}

		// SuperAdmin-only governance area. A WL client can NEVER reach these.
		sa := authed.Group("/super-admin")
		sa.Use(handlers.RequireSuperAdmin())
		{
			// WL client lifecycle
			sa.POST("/wl-clients", h.CreateWLClient)
			sa.GET("/wl-clients", h.ListWLClients)
			sa.PUT("/wl-clients/:id", h.UpdateWLClient)
			sa.POST("/wl-clients/:id/approve", h.ApproveWLClient)
			sa.POST("/wl-clients/:id/suspend", h.SuspendWLClient)
			sa.POST("/wl-clients/:id/halt", h.HaltWLClient)
			sa.POST("/wl-clients/:id/revoke", h.RevokeWLClient)
			sa.POST("/wl-clients/:id/resume", h.ResumeWLClient) // SuperAdmin-only resume

			// License ops
			sa.POST("/licenses", h.IssueLicense)
			sa.GET("/licenses", h.ListLicenses)
			sa.POST("/licenses/:id/suspend", h.SuspendLicense)
			sa.POST("/licenses/:id/halt", h.HaltLicense)
			sa.POST("/licenses/:id/revoke", h.RevokeLicense)
			sa.POST("/licenses/:id/resume", h.ResumeLicense) // SuperAdmin-only resume

			// Feature flags (per-fetcher granularity)
			sa.POST("/feature-flags", h.SetFeatureFlag)
			sa.GET("/feature-flags", h.ListFeatureFlags)

			// Two-party withdrawal SuperAdmin co-sign (governance, NOT gate).
			// A service token can NEVER reach these — only a human SuperAdmin JWT.
			sa.GET("/withdrawals", h.ListWithdrawalApprovals)
			sa.POST("/withdrawals/:id/approve", h.SuperAdminApproveWithdrawal)
			sa.POST("/withdrawals/:id/reject", h.SuperAdminRejectWithdrawal)

			// AutoApprover policy snapshot management (the security boundary
			// that defines fee/revenue => Manual, user tx => Auto).
			sa.POST("/treasury-addresses", h.AddTreasuryAddress)
			sa.GET("/treasury-addresses", h.ListTreasuryAddresses)
			sa.DELETE("/treasury-addresses/:id", h.DeleteTreasuryAddress)
			sa.POST("/auto-sign-rules", h.SetAutoSignRule)
			sa.GET("/auto-sign-rules", h.ListAutoSignRules)
			sa.DELETE("/auto-sign-rules/:id", h.DeleteAutoSignRule)

			// SuperAdmin signer address (consumed by master-wallet multisig)
			sa.GET("/signer-address", h.GetSuperAdminSigner)
		}
	}

	port := cfg.Port
	if port == "" {
		port = "9008"
	}
	srv := &http.Server{Addr: ":" + port, Handler: r, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second}
	go func() {
		log.Printf("License control plane listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Println("Stopped")
}
