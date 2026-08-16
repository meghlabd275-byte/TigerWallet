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

	router := gin.Default()
	router.Use(cors.Default())
	router.GET("/health", svc.Health)

	api := router.Group("/api/v1")
	{
		api.POST("/auth/register", svc.Register)
		api.POST("/auth/login", svc.Login)

		// Every protected route is gated by JWT auth + the fail-closed license
		// gate (503 when the product is not authorized or a fetcher is disabled).
		mw := api.Group("")
		mw.Use(wlgate.JWTAuth(cfg.JWTSecret))
		mw.Use(gate.Middleware(cfg.Product, wlgate.SimpleFetcher))
		{
			mw.POST("/tokens", svc.CreateToken)
			mw.GET("/tokens", svc.ListTokens)
			mw.GET("/tokens/:id", svc.GetToken)
			mw.PUT("/tokens/:id", svc.UpdateToken)
			mw.DELETE("/tokens/:id", svc.DeleteToken)

			mw.POST("/listings", svc.CreateListing)
			mw.GET("/listings", svc.ListListings)

			mw.POST("/launchpad", svc.CreateLaunchpadProject)
			mw.GET("/launchpad", svc.ListLaunchpadProjects)
			mw.GET("/launchpad/:id", svc.GetLaunchpadProject)
			mw.POST("/launchpad/:id/participate", svc.ParticipateInLaunchpad)
			mw.GET("/launchpad/:id/participations", svc.ListParticipations)

			mw.POST("/market-making", svc.CreateMarketMakingConfig)
			mw.GET("/market-making", svc.ListMarketMakingConfigs)

			mw.POST("/fees", svc.CreateFeeConfig)
			mw.GET("/fees", svc.ListFeeConfigs)

			mw.POST("/favorites", svc.AddFavorite)
			mw.GET("/favorites", svc.ListFavorites)
			mw.DELETE("/favorites/:id", svc.RemoveFavorite)
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
