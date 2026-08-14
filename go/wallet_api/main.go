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
		// Load any admin-managed chain config overrides into the live registry
		// so admin-added/updated chains are visible immediately at boot.
		bgCtx, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
		applyAdminChainOverrides(bgCtx)
		// Bootstrap the first admin from the ADMIN_BOOTSTRAP_EMAIL env so the
		// admin/wl-admin/master-wallet-admin role can be seeded without a
		// pre-existing admin (the first user to register with that email is
		// promoted). Subsequent role changes go through the admin API.
		bootstrapAdminRole(bgCtx, appConfig.AdminBootstrapEmail)
		cancel2()
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

	// ---- dApp directory (read-only public) ----
	r.GET("/api/v1/dapps", handleListDApps)
	r.GET("/api/v1/dapps/categories", handleDAppCategories)
	r.GET("/api/v1/dapps/:id", handleGetDApp)
	r.GET("/api/v1/defi/protocols", handleDefiProtocols)

	// ---- Token asset registry (read-only public) ----
	r.GET("/api/v1/tokens/registry", handleTokenRegistry)

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
		wallet.POST("/nft/transfer", handleNFTTransfer)

		// ---- Non-EVM signing (Solana Ed25519, Bitcoin secp256k1, Cosmos secp256k1) ----
		// Real key derivation + signing; mainnet only. See non_evm_signing.go.
		wallet.POST("/non_evm/sign", handleNonEvmSign)
		wallet.POST("/non_evm/send", handleNonEvmSend)
		wallet.POST("/non_evm/address", handleNonEvmAddress)

		// ---- Web3 Secret Storage V3 keystore import/export (geth/MetaMask interop) ----
		wallet.POST("/keystore/export", handleExportKeystore)
		wallet.POST("/keystore/import", handleImportKeystore)

		// ---- DeFi: swap & staking (real CoinGecko quotes + on-chain action) ----
		wallet.GET("/swap/quote", handleSwapQuote)
		wallet.POST("/swap/execute", handleSwapExecute)
		// ---- On-chain AMM router (real getAmountsOut via eth_call) ----
		wallet.GET("/amm/quote", handleAmmQuote)
		wallet.POST("/amm/swap", handleAmmSwap)
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

                // ---- Multi-device sync (PostgreSQL-backed, no mock data) ----
                wallet.GET("/devices", handleListDevices)
                wallet.POST("/devices", handleRegisterDevice)
                wallet.POST("/devices/:id/sync", handleSyncDevice)
                wallet.DELETE("/devices/:id", handleDeleteDevice)

		// ---- Portfolio features (PostgreSQL-backed, no mock data) ----
		wallet.GET("/approvals", handleListApprovals)
		wallet.DELETE("/approvals/:id", handleRevokeApproval)
		wallet.GET("/perpetual/positions", handleListPerpetualPositions)
		wallet.POST("/perpetual/positions", handleCreatePerpetualPosition)
		wallet.POST("/perpetual/positions/:id/close", handleClosePerpetualPosition)
		wallet.GET("/margin/positions", handleListMarginPositions)
		wallet.POST("/margin/positions", handleCreateMarginPosition)
		wallet.POST("/margin/positions/:id/close", handleCloseMarginPosition)
		wallet.GET("/token-sales", handleListTokenSales)
		wallet.POST("/token-sales/:id/participate", handleParticipateTokenSale)
		wallet.GET("/dao/proposals", handleListDAOProposals)
		wallet.POST("/dao/proposals", handleCreateDAOProposal)
		wallet.POST("/dao/proposals/:id/vote", handleVoteDAOProposal)
		wallet.GET("/dao/delegates", handleListDAODelegates)
		wallet.GET("/launchpool", handleListLaunchpool)
		wallet.GET("/launchpool/stakes", handleListLaunchpoolStakes)
		wallet.POST("/launchpool/stake", handleLaunchpoolStake)
		wallet.POST("/launchpool/unstake", handleLaunchpoolUnstake)

		// ---- Admin / dashboard routes (authenticated + admin-role) ----
		// Back the master-wallet dashboard with real PostgreSQL aggregates.
		admin := wallet.Group("/admin")
		admin.Use(RequireAdmin())
		{
			admin.GET("/stats", handleAdminStats)
			admin.GET("/wallets", handleAdminWallets)
			admin.GET("/transactions", handleAdminTransactions)
			admin.GET("/users", handleAdminUsers)

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

			// ---- Admin role management ----
			admin.PUT("/users/:id/role", handleAdminSetUserRole)

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
