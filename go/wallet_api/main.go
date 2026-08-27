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
	r.POST("/api/v1/gas/estimate", handleEstimateGas)
	r.GET("/api/v1/network-status", handleNetworkStatus)
	r.GET("/api/v1/chart/history", handleChartHistory)

	// ---- Transaction simulation (pre-sign dry-run, read-only) ----
	r.POST("/api/v1/simulate", handleSimulateTransaction)

	// ---- ENS resolution (real on-chain registry lookup on mainnet) ----
	r.GET("/api/v1/ens/resolve", handleENSResolve)
	r.GET("/api/v1/ens/lookup", handleENSLookup)

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
	// Rate-limited per IP (throttles brute-force credential guessing). 5/min,
	// burst 5 — generous for normal use, caps automated attempts.
	auth := r.Group("/api/v1/auth")
	auth.Use(RateLimit(authLimiter))
	{
		auth.POST("/register", handleRegister)
		auth.POST("/login", handleLogin)
		auth.POST("/guest", handleGuestAuth)
	}

	// ---- Protected wallet routes ----
	wallet := r.Group("/api/v1")
	wallet.Use(AuthMiddleware(appConfig.JWTSecret))
	{
		wallet.POST("/wallets", handleCreateWallet)
		wallet.GET("/wallets", handleListWallets)
		// Watch-only: track an address without its seed. Read-only; every
		// signing/funds-movement path rejects these wallets fail-closed.
		wallet.POST("/wallets/watch-only", handleCreateWatchOnlyWallet)
		// Price alerts (real CoinGecko spot quotes, evaluated on-read).
		wallet.GET("/price-alerts", handleListPriceAlerts)
		wallet.POST("/price-alerts", handleCreatePriceAlert)
		wallet.PUT("/price-alerts/:id", handleUpdatePriceAlert)
		wallet.DELETE("/price-alerts/:id", handleDeletePriceAlert)
		// Google Drive backup: export the encrypted seed blob (password-verified)
		// for upload to Drive; restore from a downloaded blob + password.
		wallet.POST("/wallets/:id/export-encrypted-seed", handleExportEncryptedSeed)
		wallet.POST("/wallets/import-encrypted-seed", handleImportEncryptedSeed)

		// ---- App lock + passkey (passwordless UserWallet) ----
		// Passkey wallet creation: create a wallet whose seed is encrypted with a
		// passkey-releasable unlock key (no user password).
		wallet.POST("/passkey/wallet", handlePasskeyCreateWallet)
		// Set/replace a per-wallet app-lock credential (passcode and/or passkey).
		wallet.POST("/wallets/:id/lock", handleLockSetup)
		// Unlock: verify passcode/passkey/nothing → issue a short-lived passwordless
		// unlock_token accepted by /send, /sign, /nft/transfer, /auto-send.
		wallet.POST("/wallets/:id/unlock", handleUnlock)

		// ---- KYC (proxied to listing_service) + P2P (KYC-gated, proxied to p2p_trading) ----
		wallet.GET("/kyc/status", handleKYCStatus)
		wallet.POST("/kyc/register", handleKYCRegister)
		wallet.POST("/kyc/submit", handleKYCSubmit)
		wallet.POST("/kyc/document", handleKYCDocument)
		wallet.GET("/kyc/session/:id", handleKYCDetail)
		wallet.GET("/p2p/adverts", handleP2PAdverts)
		// P2P order creation is KYC-gated (requires verified KYC status); registered
		// with the rate-limited signing group below (signLimited).

		wallet.GET("/balance", handleBalance)
		wallet.GET("/tokens", handleTokenBalances)
		wallet.GET("/transactions", handleTransactions)
		wallet.GET("/nfts", handleNFTs)
		// ---- Signing/funds-movement routes are rate-limited per user ----
		// (~20/min, burst 20): generous for normal use, caps automated drain.
		signLimited := wallet.Group("")
		signLimited.Use(RateLimit(signLimiter))
		signLimited.POST("/send", handleSendTransaction)
		signLimited.POST("/auto-send", handleAutoSend)
		signLimited.POST("/sign", handleSignMessage)
		signLimited.POST("/nft/transfer", handleNFTTransfer)

		// ---- Non-EVM signing (Solana Ed25519, Bitcoin secp256k1, Cosmos secp256k1) ----
		// Real key derivation + signing; mainnet only. See non_evm_signing.go.
		// Also funds-movement, so share the signing rate limit.
		signLimited.POST("/non_evm/sign", handleNonEvmSign)
		// P2P order creation — KYC-gated (fail-closed 403 if KYC not verified).
		signLimited.POST("/p2p/orders", handleP2PCreateOrder)
		signLimited.POST("/non_evm/send", handleNonEvmSend)
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

		// ---- Auxiliary DeFi service reverse-proxies ----
		// Forwards to the standalone microservices so every UserWallet client
		// (web/desktop/android/ios/rust/extension) can target a single port
		// (:8443) and reach the full DeFi surface. Auth is preserved via the
		// Bearer JWT header; no data is fabricated.
		//   lending_service      :8009  (/api/v1/lending/*)
		//   copy_trading_service :8006  (/api/v1/copytrading/*)
		//   governance_service   :8454  (/api/v1/governance/*)
		//   prediction_service   :8455  (/api/v1/prediction/*)
		wallet.Any("/lending/*path", deFiProxy("LENDING_SERVICE_URL", "http://localhost:8009", "lending"))
		wallet.Any("/copytrading/*path", deFiProxy("COPYTRADING_SERVICE_URL", "http://localhost:8006", "copytrading"))
		wallet.Any("/governance/*path", deFiProxy("GOVERNANCE_SERVICE_URL", "http://localhost:8454", "governance"))
		wallet.Any("/prediction/*path", deFiProxy("PREDICTION_SERVICE_URL", "http://localhost:8455", "prediction"))
		// Bridge: the canonical go/bridge_service (:8007) exposes
		// /api/v1/bridge/{routes,quote,transfer,tx/:id,history}. Proxying it
		// through :8443 closes Gap E (no dedicated bridge backend) so every
		// UserWallet client reaches real cross-chain routing via one port.
		wallet.Any("/bridge/*path", deFiProxy("BRIDGE_SERVICE_URL", "http://localhost:8007", "bridge"))
		// dApp browser / WalletConnect: the canonical dapp_browser/go service
		// (:8083) exposes /{pairings,sessions,ws/*,health} for WalletConnect-style
		// dApp pairing + signed-request relay. Proxying it through :8443 closes
		// Gap F so every UserWallet client reaches dApp pairing via one port.
		// The upstream mounts those routes at the ROOT (no /api/v1 prefix), so
		// the incoming /api/v1/dapp[/walletconnect] prefix is rewritten away.
		// Note: WebSocket upgrade (/ws/:topic) is handled by the reverse proxy
		// transparently (httputil.ReverseProxy supports WS).
		wallet.Any("/dapp/*path", deFiProxyRewrite("DAPP_BROWSER_SERVICE_URL", "http://localhost:8083", "dapp", ""))
		wallet.Any("/walletconnect/*path", deFiProxyRewrite("DAPP_BROWSER_SERVICE_URL", "http://localhost:8083", "walletconnect", ""))

		// Crypto card: the canonical go/card_service (:8457) exposes
		// /api/v1/card/{balance,transactions,rates}. Clients use the plural
		// /api/v1/cards/* surface (incl. /cards/:id/balance per-card style); the
		// cardsProxy maps it to the upstream per-user card account.
		wallet.Any("/cards/*path", cardsProxy("CARD_SERVICE_URL", "http://localhost:8457"))
		// Singular form (production/react client uses /card/{balance,transactions}).
		wallet.Any("/card/*path", deFiProxyRewrite("CARD_SERVICE_URL", "http://localhost:8457", "card", "/api/v1/card"))

		// Fiat ramp: the canonical go/fiat_ramp (:8451) exposes
		// /api/v1/ramp/{providers,quote,offramp-quote,order,...} — identical
		// prefix on both sides, plain proxy.
		wallet.Any("/ramp/*path", deFiProxy("FIAT_RAMP_SERVICE_URL", "http://localhost:8451", "ramp"))

		// MasterWallet multisig: reverse-proxied to the master wallet backend
		// (:8450) so UserWallet clients (extension/iOS/android/web) use multisig
		// without ever calling :8450 directly (separation rule).
		//   master_wallet/backend :8450  (/api/v1/master-wallet/<id>/multisig/*)
		wallet.Any("/wallet/multisig/*path", masterWalletMultisigProxy())

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
