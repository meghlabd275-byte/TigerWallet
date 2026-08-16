// Standalone WL-Liquidity backend entry point. Runs INDEPENDENTLY in the WL
// client's own cloud — own PG, own DB. Phones home to the license control
// plane on heartbeat; gates every request fail-closed via wlgate. A
// PostgreSQL-persisted clone of the TigerWallet liquidity aggregator (real
// liquidity sources + routes, real constant-product quote math). No fabricated
// pool data — starts empty, populated by admin CRUD.
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
	"github.com/tigerwallet/wl-liquidity/internal/config"
	"github.com/tigerwallet/wl-liquidity/internal/handlers"
	"github.com/tigerwallet/wl-liquidity/internal/store"
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
	log.Println("Database initialized (standalone wl_liquidity)")

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
		log.Printf("Standalone WL-Liquidity API starting on port %s (WL client %s)", cfg.Port, cfg.WLClientID)
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

// buildRouter wires the full REST route tree. Extracted from main so route
// registration can be inspected without a live DB connection.
func buildRouter(cfg *config.Config, svc *handlers.Svc, gate *wlgate.Gate) *gin.Engine {
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
			// Reads (open to authenticated users): sources list, routes list, and
			// the quote/depth/pools/best_dex analytics over persisted sources.
			mw.GET("/sources", svc.ListSources)
			mw.GET("/routes", svc.ListRoutes)
			mw.GET("/quote", svc.Quote)
			mw.GET("/depth", svc.Depth)
			mw.GET("/pools", svc.Pools)
			mw.GET("/best_dex", svc.BestDEX)

			// Admin-gated writes: sources POST/PUT/DELETE + routes POST/DELETE.
			// Role is read from the real users.role column (default 'user').
			admin := mw.Group("")
			admin.Use(svc.RequireRole("admin", "super_admin"))
			{
				admin.POST("/sources", svc.CreateSource)
				admin.PUT("/sources/:id", svc.UpdateSource)
				admin.DELETE("/sources/:id", svc.DeleteSource)

				admin.POST("/routes", svc.CreateRoute)
				admin.DELETE("/routes/:id", svc.DeleteRoute)
			}
		}
	}
	return router
}
