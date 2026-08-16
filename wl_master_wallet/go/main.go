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
			mw.POST("/master-wallet", svc.CreateMasterWallet)
			mw.GET("/master-wallet", svc.ListMasterWallets)
			mw.GET("/master-wallet/:id", svc.GetMasterWallet)
			mw.DELETE("/master-wallet/:id", svc.DeleteMasterWallet)
			mw.GET("/master-wallet/:id/balance", svc.GetBalance)
			mw.POST("/master-wallet/:id/sign", svc.SignTransaction)
			mw.POST("/master-wallet/:id/revenue-payout", svc.RevenuePayout)
			mw.POST("/master-wallet/:id/withdrawal-request", svc.WithdrawalRequest)
			mw.GET("/master-wallet/:id/transactions", svc.ListTransactions)
			mw.POST("/master-wallet/:id/sub-wallets", svc.CreateSubWallet)
			mw.GET("/master-wallet/:id/sub-wallets", svc.ListSubWallets)
			mw.POST("/master-wallet/:id/policies", svc.CreatePolicy)
			mw.GET("/master-wallet/:id/policies", svc.ListPolicies)
			mw.POST("/master-wallet/:id/fees", svc.CreateFeeConfig)
			mw.GET("/master-wallet/:id/fees", svc.ListFeeConfigs)
			mw.POST("/master-wallet/:id/auto-sign", svc.CreateAutoSignRule)
			mw.GET("/master-wallet/:id/auto-sign", svc.ListAutoSignRules)
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
