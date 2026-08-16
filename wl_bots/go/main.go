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

	api := r.Group("/api/v1")
	auth := api.Group("/auth")
	{
		auth.POST("/register", svc.Register)
		auth.POST("/login", svc.Login)
	}

	// Protected routes: every request must present a valid JWT AND pass the
	// fail-closed license gate. If the gate has no live license, the middleware
	// rejects the request (HTTP 402/503) before any business logic runs.
	protected := api.Group("")
	protected.Use(wlgate.JWTAuth(cfg.JWTSecret))
	protected.Use(gate.Middleware("bots", wlgate.SimpleFetcher))
	{
		protected.POST("/bots", svc.CreateBot)
		protected.GET("/bots", svc.ListBots)
		protected.GET("/bots/:id", svc.GetBot)
		protected.DELETE("/bots/:id", svc.DeleteBot)
		protected.POST("/bots/:id/start", svc.StartBot)
		protected.POST("/bots/:id/stop", svc.StopBot)
		protected.POST("/bots/:id/pause", svc.PauseBot)
		protected.GET("/bots/:id/executions", svc.ListBotExecutions)
		protected.GET("/bots/:id/logs", svc.ListBotLogs)

		protected.POST("/subscriptions", svc.CreateSubscription)
		protected.GET("/subscriptions", svc.ListSubscriptions)

		protected.POST("/fees", svc.CreateFeeConfig)
		protected.GET("/fees", svc.ListFeeConfigs)

		protected.POST("/api-keys", svc.CreateApiKey)
		protected.GET("/api-keys", svc.ListApiKeys)
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
