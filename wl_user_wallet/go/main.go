// Standalone WL-UserWallet backend entry point. Runs INDEPENDENTLY in the WL
// client's own environment — own PG, own signing, own DB. Phones home to the
// license control plane on heartbeat; gates every request fail-closed.
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
	"github.com/tigerwallet/wl-user-wallet/internal/config"
	"github.com/tigerwallet/wl-user-wallet/internal/handlers"
	"github.com/tigerwallet/wl-user-wallet/internal/middleware"
	"github.com/tigerwallet/wl-user-wallet/internal/store"
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
	log.Println("Database initialized (standalone wl_userwallet)")

	// Start the license-control-plane heartbeat (fail-closed phone-home).
	go middleware.HeartbeatLoop(ctx, cfg.ControlPlaneURL, cfg.ControlPlaneToken, cfg.WLClientID, cfg.LicenseKey, cfg.Product, cfg.InstanceID, cfg.HeartbeatInterval)

	// Initialize the two-party withdrawal gate (SuperAdmin co-sign verification
	// for fee/revenue/treasury withdrawals — the slow path).
	middleware.InitTwoPartyGate(cfg.ControlPlaneURL, cfg.ControlPlaneToken)

	svc := handlers.New(cfg, st)

	router := gin.Default()
	router.Use(cors.Default())
	router.GET("/health", svc.Health)

	api := router.Group("/api/v1")
	{
		api.POST("/auth/register", svc.Register)
		api.POST("/auth/login", svc.Login)

		// Every protected route is gated by the license gate (fail-closed 503
		// when the product is not authorized or a fetcher is disabled).
		wallet := api.Group("")
		wallet.Use(middleware.JWTAuth(cfg.JWTSecret))
		wallet.Use(middleware.Gate("user_wallet", middleware.SimpleFetcher))
		{
			// Existing wallet-scoped routes (back-compat).
			wallet.POST("/wallets", svc.CreateWallet)
			wallet.GET("/wallets", svc.ListWallets)
			wallet.GET("/wallets/:id/balance", svc.GetBalance)
			wallet.POST("/wallets/:id/send", svc.SendTransaction)
			wallet.POST("/wallets/:id/sign", svc.SignMessage)
			wallet.GET("/wallets/:id/transactions", svc.ListTransactions)

			// ---- Canonical flat routes (parity with TigerWallet wallet_api) ----
			// Read-only chain / market data.
			wallet.GET("/balance", svc.FlatBalance)
			wallet.GET("/tokens", svc.GetTokens)
			wallet.GET("/nfts", svc.GetNFTs)
			wallet.GET("/gas", svc.GetGas)
			wallet.GET("/price", svc.GetPrice)
			wallet.GET("/chains", svc.GetChains)

			// Send / sign (flat, wallet_id in body/query).
			wallet.POST("/send", svc.FlatSend)
			wallet.POST("/sign", svc.FlatSign)
			wallet.GET("/transactions", svc.FlatTransactions)
			wallet.GET("/transactions/:txHash", svc.GetTransaction)

			// Swap (real CoinGecko cross-rate + on-chain V2 router calldata).
			wallet.GET("/swap/quote", svc.SwapQuote)
			wallet.POST("/swap/execute", svc.SwapExecute)

			// Staking (real on-chain stake/unstake/claim calldata).
			wallet.GET("/staking/quote", svc.StakingQuote)
			wallet.POST("/staking/stake", svc.StakingStake)
			wallet.POST("/staking/unstake", svc.StakingUnstake)
			wallet.POST("/staking/claim", svc.StakingClaim)

			// Non-EVM signing (real Solana/Bitcoin/Cosmos crypto).
			wallet.POST("/non_evm/sign", svc.NonEvmSign)
			wallet.POST("/non_evm/send", svc.NonEvmSend)
			wallet.POST("/non_evm/address", svc.NonEvmAddress)

			// Address book (real PG CRUD).
			wallet.GET("/address-book", svc.ListAddressBook)
			wallet.POST("/address-book", svc.CreateAddressBook)
			wallet.PUT("/address-book/:id", svc.UpdateAddressBook)
			wallet.DELETE("/address-book/:id", svc.DeleteAddressBook)

			// Device sync (real PG CRUD).
			wallet.GET("/devices", svc.ListDevices)
			wallet.POST("/devices", svc.RegisterDevice)
			wallet.POST("/devices/:id/sync", svc.SyncDevice)
			wallet.DELETE("/devices/:id", svc.DeleteDevice)

			// Keystore V3 (real Web3 Secret Storage scrypt+AES-CTR+keccak MAC).
			wallet.POST("/keystore/export", svc.ExportKeystore)
			wallet.POST("/keystore/import", svc.ImportKeystore)
		}

		// Public unauthenticated reads (same real logic, no auth gate).
		public := api.Group("/public")
		{
			public.GET("/balance", svc.PublicBalance)
			public.GET("/tokens", svc.PublicTokens)
			public.GET("/transactions", svc.PublicTransactions)
			public.GET("/nfts", svc.PublicNFTs)
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
		log.Printf("Standalone WL-UserWallet API starting on port %s (WL client %s)", cfg.Port, cfg.WLClientID)
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
