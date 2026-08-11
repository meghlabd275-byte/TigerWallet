// TigerWallet Wallet API — canonical backend service.
//
// Real wallet operations: BIP-39 mnemonic, BIP-32/44 HD key derivation,
// secp256k1 signing, EVM transaction broadcast, on-chain balance/token/tx/nft
// fetchers, PostgreSQL persistence, Redis cache. No mocks, no stubs.
//
// Run: go run .  (listens on :8443 by default)
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
)

var (
	appConfig *AppConfig
	store     *Store
)

func main() {
	appConfig = LoadConfig()

	// Connect to PostgreSQL + Redis (best-effort; service still boots for
	// read-only/public endpoints if DB is temporarily unavailable).
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	s, err := NewStore(ctx, appConfig.DatabaseURL, appConfig.RedisAddr)
	cancel()
	if err != nil {
		log.Printf("WARNING: store init failed (DB/Redis may be down): %v — public endpoints only", err)
	} else {
		store = s
		log.Println("Connected to PostgreSQL + Redis")
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// CORS — allow the Next.js frontend and all wallet clients
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length", "X-Total-Count"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// ---- Public routes ----
	r.GET("/health", handleHealth)
	r.GET("/api/v1/health", handleHealth)
	r.GET("/api/v1/chains", handleSupportedChains)
	r.GET("/api/v1/price", handlePrice)
	r.GET("/api/v1/gas", handleGasPrice)
	r.GET("/api/v1/chart/history", handleChartHistory)

	// ---- Security / scam-scan routes (read-only public) ----
	r.GET("/api/v1/security/check-url", handleCheckURL)
	r.GET("/api/v1/security/check-address", handleCheckAddress)
	r.POST("/api/v1/security/scan", handleSecurityScan)

	// ---- Trading-terminal market data (read-only public) ----
	r.GET("/api/v1/terminal/kline/:symbol", handleTerminalKline)
	r.GET("/api/v1/terminal/ticker/:symbol", handleTerminalTicker)

	// ---- Auth routes ----
	auth := r.Group("/api/v1/auth")
	{
		auth.POST("/register", handleRegister)
		auth.POST("/login", handleLogin)
	}

	// ---- Protected wallet routes ----
	wallet := r.Group("/api/v1")
	wallet.Use(AuthMiddleware(appConfig.JWTSecret))
	{
		wallet.POST("/wallets", handleCreateWallet)
		wallet.GET("/wallets", handleListWallets)
		wallet.GET("/balance", handleBalance)
		wallet.GET("/tokens", handleTokenBalances)
		wallet.GET("/transactions", handleTransactions)
		wallet.GET("/nfts", handleNFTs)
		wallet.POST("/send", handleSendTransaction)
		wallet.POST("/sign", handleSignMessage)

		// ---- DeFi: swap & staking (real CoinGecko quotes + on-chain action) ----
		wallet.GET("/swap/quote", handleSwapQuote)
		wallet.POST("/swap/execute", handleSwapExecute)
		wallet.GET("/staking/quote", handleStakingQuote)
		wallet.POST("/staking/stake", handleStakingAction("stake"))
		wallet.POST("/staking/unstake", handleStakingAction("unstake"))
		wallet.POST("/staking/claim", handleStakingAction("claim"))
		wallet.GET("/transactions/:txHash", handleTransactionReceipt)

		// ---- Address book (per-user contacts) ----
		wallet.GET("/address-book/contacts", handleListContacts)
		wallet.POST("/address-book/contacts", handleCreateContact)
		wallet.PUT("/address-book/contacts/:id", handleUpdateContact)
		wallet.DELETE("/address-book/contacts/:id", handleDeleteContact)

		// ---- Admin / dashboard routes (authenticated) ----
		// Back the master-wallet dashboard with real PostgreSQL aggregates.
		admin := wallet.Group("/admin")
		{
			admin.GET("/stats", handleAdminStats)
			admin.GET("/wallets", handleAdminWallets)
			admin.GET("/transactions", handleAdminTransactions)

			admin.GET("/wallets/:id", handleAdminWalletDetail)
			admin.PUT("/wallets/:id", handleAdminUpdateWallet)
			admin.DELETE("/wallets/:id", handleAdminDeleteWallet)
			admin.GET("/wallets/transactions", handleAdminWalletTransactions)

			// ---- Admin chain configuration ----
			admin.GET("/chains", handleAdminListChains)
			admin.POST("/chains", handleAdminCreateChain)
			admin.PUT("/chains/:id", handleAdminUpdateChain)
			admin.DELETE("/chains/:id", handleAdminDeleteChain)
			admin.GET("/chains/bridges", handleAdminListBridges)
			admin.POST("/chains/bridges", handleAdminCreateBridge)
			admin.GET("/chains/validators", handleAdminListValidators)
			admin.POST("/chains/validators", handleAdminCreateValidator)
			admin.GET("/chains/metrics", handleAdminChainMetrics)
			admin.GET("/chains/token-deployments", handleAdminTokenDeployments)

			// ---- Admin fee configuration ----
			admin.GET("/fees", handleAdminListFees)
			admin.POST("/fees", handleAdminCreateFee)
			admin.PUT("/fees/:id", handleAdminUpdateFee)
			admin.DELETE("/fees/:id", handleAdminDeleteFee)
			admin.GET("/fees/transactions", handleAdminFeeTransactions)
			admin.GET("/fees/revenue", handleAdminFeeRevenue)
		}
	}

	// Also expose balance/tokens/tx/nfts/gas without auth for read-only public
	// data (useful for "view any address" features). Protected send/sign stay authed.
	r.GET("/api/v1/public/balance", handleBalance)
	r.GET("/api/v1/public/tokens", handleTokenBalances)
	r.GET("/api/v1/public/transactions", handleTransactions)
	r.GET("/api/v1/public/nfts", handleNFTs)

	srv := &http.Server{
		Addr:         ":" + appConfig.Port,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		log.Printf("TigerWallet API listening on :%s", appConfig.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Printf("forced shutdown: %v", err)
	}
	if store != nil {
		store.PG.Close()
		_ = store.Redis.Close()
	}
	log.Println("Server stopped")
}
