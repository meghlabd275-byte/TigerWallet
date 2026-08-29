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
	"github.com/tigerwallet/wl-user-wallet/internal/chains"
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

	// Sync the chain registry from the canonical wallet_api so master-wallet
	// chain governance (add/update/remove) propagates without redeploying.
	chains.StartRegistrySync(ctx, cfg.CanonicalRegistryURL)

	// Initialize the two-party withdrawal gate (SuperAdmin co-sign verification
	// for fee/revenue/treasury withdrawals — the slow path).
	middleware.InitTwoPartyGate(cfg.ControlPlaneURL, cfg.ControlPlaneToken, cfg.Product, cfg.WLClientID)

	// Wire the store-backed is_active checker so RequireActiveUser middleware
	// can lock out a suspended user immediately (even with a valid stateless JWT).
	middleware.SetActiveUserChecker(st.GetUserActive)

	svc := handlers.New(cfg, st)

	router := gin.Default()
	router.Use(cors.Default())
	router.GET("/health", svc.Health)

	api := router.Group("/api/v1")
	{
		api.POST("/auth/register", svc.Register)
		api.POST("/auth/login", svc.Login)
                    api.POST("/auth/guest", svc.GuestAuth)

		// Every protected route is gated by the license gate (fail-closed 503
		// when the product is not authorized or a fetcher is disabled).
		// CategoryFetcher derives the fetcher key from the first functional
		// path segment so SuperAdmin can toggle per-feature granularity
		// (e.g. disable swap while leaving staking/send running).
		wallet := api.Group("")
		wallet.Use(middleware.JWTAuth(cfg.JWTSecret))
		wallet.Use(middleware.Gate("user_wallet", middleware.CategoryFetcher))
		{
			// Existing wallet-scoped routes (back-compat).
			wallet.POST("/wallets", svc.CreateWallet)
			wallet.GET("/wallets", svc.ListWallets)
			wallet.GET("/wallets/:id/balance", svc.GetBalance)
			wallet.POST("/wallets/:id/send", svc.SendTransaction)
			wallet.POST("/wallets/:id/sign", svc.SignMessage)
			wallet.GET("/wallets/:id/transactions", svc.ListTransactions)
				// ---- additional routes ----
				wallet.GET("/wallets/:id/transactions", svc.ListTransactions)
				wallet.GET("/wallets/:id", svc.GetWallet)
				wallet.PUT("/wallets/:id", svc.UpdateWallet)
				wallet.DELETE("/wallets/:id", svc.DeleteWallet)
				wallet.GET("/wallets/:id/export-encrypted-seed", svc.ExportEncryptedSeed)
				wallet.POST("/wallets/:id/lock", svc.LockWallet)
				wallet.POST("/wallets/:id/unlock", svc.UnlockWallet)
				wallet.POST("/wallets/import-encrypted-seed", svc.ImportEncryptedSeed)
				wallet.POST("/wallets/watch-only", svc.CreateWatchOnlyWallet)
				wallet.GET("/wallets/transactions", svc.ListWalletsTransactions)

			// ---- Canonical flat routes (parity with TigerWallet wallet_api) ----
			// Read-only chain / market data.
			wallet.GET("/balance", svc.FlatBalance)
			wallet.GET("/tokens", svc.GetTokens)
			wallet.GET("/nfts", svc.GetNFTs)
			wallet.GET("/gas", svc.GetGas)
			wallet.GET("/price", svc.GetPrice)
			wallet.GET("/chains", svc.GetChains)
				// ---- additional routes ----
				wallet.GET("/chains", svc.GetChains)
				wallet.GET("/chains/:id", svc.GetChain)
				wallet.GET("/chains/bridges", svc.GetChainBridges)
				wallet.GET("/chains/metrics", svc.GetChainMetrics)
				wallet.GET("/chains/token-deployments", svc.GetChainTokenDeployments)
				wallet.GET("/chains/validators", svc.GetChainValidators)
				wallet.GET("/users", svc.ListUsers)
				wallet.PUT("/users/:id/role", svc.UpdateUserRole)

			// Send / sign (flat, wallet_id in body/query).
			wallet.POST("/send", middleware.RequireActiveUser(), svc.FlatSend)
			// Pre-sign transaction simulation + ENS resolution
			// (parity with go/wallet_api for Android/iOS clients).
			wallet.POST("/simulate", svc.SimulateTransaction)
			wallet.GET("/ens/resolve", svc.ENSResolve)
			wallet.GET("/ens/lookup", svc.ENSLookup)
			// /auto-send is an alias of /send (FlatSend). The gate's
			// requireApproval already performs the auto-approval fast path
			// (license alive + non-treasury tx => auto_approved within a
			// second) for regular transfers; the response carries
			// auto_approved + auto_approval_reason so clients can show the
			// ⚡ badge. Fee/revenue/treasury txs still require the
			// SuperAdmin two-party withdrawal_id.
			wallet.POST("/auto-send", middleware.RequireActiveUser(), svc.FlatSend)
			wallet.POST("/sign", middleware.RequireActiveUser(), svc.FlatSign)
			wallet.GET("/transactions", svc.FlatTransactions)
			wallet.GET("/transactions/:txHash", svc.GetTransaction)

			// Swap (real CoinGecko cross-rate + on-chain V2 router calldata).
			wallet.GET("/swap/quote", svc.SwapQuote)
			wallet.POST("/swap/execute", middleware.RequireActiveUser(), svc.SwapExecute)
				// ---- additional routes ----
				wallet.GET("/amm/quote", svc.SwapQuote)
				wallet.POST("/amm/swap", middleware.RequireActiveUser(), svc.SwapExecute)
				wallet.POST("/nft/transfer", svc.NFTTransfer)
                        // Passkey wallet creation (WebAuthn credential-wrapped entropy).
                        wallet.POST("/passkey/wallet", svc.PasskeyCreateWallet)

			// Staking (real on-chain stake/unstake/claim calldata).
			wallet.GET("/staking/quote", svc.StakingQuote)
			wallet.POST("/staking/stake", middleware.RequireActiveUser(), svc.StakingStake)
			wallet.POST("/staking/unstake", middleware.RequireActiveUser(), svc.StakingUnstake)
			wallet.POST("/staking/claim", middleware.RequireActiveUser(), svc.StakingClaim)

			// Non-EVM signing (real Solana/Bitcoin/Cosmos crypto).
			wallet.POST("/non_evm/sign", middleware.RequireActiveUser(), svc.NonEvmSign)
			wallet.POST("/non_evm/send", middleware.RequireActiveUser(), svc.NonEvmSend)
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

                        // ---- Full flat-route parity with canonical wallet_api ----
                        // Price alerts
                        wallet.GET("/price-alerts", svc.ListPriceAlerts)
                        wallet.POST("/price-alerts", svc.CreatePriceAlert)
                        wallet.DELETE("/price-alerts/:id", svc.DeletePriceAlert)
                        // P2P trading
                        wallet.GET("/p2p/adverts", svc.ListP2PAdverts)
                        wallet.POST("/p2p/orders", svc.CreateP2POrder)
                        // DAO
                        wallet.GET("/dao/proposals", svc.ListDaoProposals)
                        wallet.POST("/dao/proposals", svc.CreateDaoProposal)
                        wallet.POST("/dao/proposals/:id/vote", svc.VoteDaoProposal)
                        // Launchpool
                        wallet.GET("/launchpool", svc.LaunchpoolPools)
                        wallet.GET("/launchpool/stakes", svc.LaunchpoolStakes)
                        wallet.POST("/launchpool/stake", svc.LaunchpoolStake)
                        // Token sales
                        wallet.GET("/token-sales", svc.ListTokenSales)
                        wallet.POST("/token-sales/:id/participate", svc.ParticipateTokenSale)

                        // Public mirror + auth aliases + missing launchpool/unstake
                        wallet.POST("/launchpool/unstake", svc.StakingUnstake)
                        // Alias central auth & create user routes (SDKs also hit /register|login)
                        wallet.POST("/register", svc.Register)
                        wallet.POST("/login", svc.Login)
                        wallet.POST("/guest", svc.GuestAuth)
                        // Public mirror (to public_group methods)
                        api.GET("/api/v1/public/balance", svc.PublicBalance)
                        api.GET("/api/v1/public/tokens", svc.PublicTokens)
                        api.GET("/api/v1/public/transactions", svc.PublicTransactions)
                        api.GET("/api/v1/public/nfts", svc.PublicNFTs)
                        // Margin & perpetual closure of series already in margin;
                        // comment fetched source groups hide missed earlier anchors:
                        wallet.GET("/perpetual/positions", svc.ListMarginPositions)
                        wallet.POST("/perpetual/positions/:id/close", svc.CloseMarginPosition)
                        wallet.GET("/perp/positions", svc.ListPerpPositions)
                        wallet.POST("/perp/positions", svc.CreatePerpPosition)
                        wallet.POST("/perp/positions/:id/close", svc.ClosePerpPosition)
                        // Token approvals
                        wallet.GET("/approvals", svc.ListTokenApprovals)
                        wallet.DELETE("/approvals/:id", svc.RevokeTokenApproval)
                        // Fees
                        wallet.GET("/fees", svc.ListFees)
                        wallet.GET("/fees/revenue", svc.FeeRevenue)
                        // KYC
                        wallet.GET("/kyc/status", svc.KycStatus)
                        wallet.POST("/kyc/register", svc.KycRegister)
                        wallet.POST("/kyc/submit", svc.KycSubmit)
                        // Card
                        wallet.GET("/card/balance", svc.CardBalance)
                        wallet.GET("/card/transactions", svc.CardTransactions)
                        // Margin & perpetual
                        wallet.GET("/margin/positions", svc.ListMarginPositions)
                        wallet.POST("/margin/positions", svc.CreateMarginPosition)
                        wallet.POST("/margin/positions/:id/close", svc.CloseMarginPosition)
                        wallet.GET("/perp/positions", svc.ListPerpPositions)
                        wallet.POST("/perp/positions", svc.CreatePerpPosition)
                        wallet.POST("/perp/positions/:id/close", svc.ClosePerpPosition)

			// Scoped-admin role assignment — WL client owner only (wl_client scope).
			wallet.PUT("/users/:id/scopes", svc.UpdateAdminScopes)

			// Admin oversight (wallet_admin / wl_client scope) — view all
			// wallets/users + suspend/activate a user. NO fund movement.
			wallet.GET("/admin/users", svc.AdminListUsers)
			wallet.GET("/admin/wallets", svc.AdminListWallets)
			wallet.POST("/admin/users/:id/suspend", svc.AdminSuspendUser)
			wallet.POST("/admin/users/:id/activate", svc.AdminActivateUser)
		}

		// Public unauthenticated reads (same real logic, no auth gate).
		public := api.Group("/public")
		{
			public.GET("/balance", svc.PublicBalance)
			public.GET("/tokens", svc.PublicTokens)
			public.GET("/transactions", svc.PublicTransactions)
			public.GET("/nfts", svc.PublicNFTs)

            // Mirror canonical /api/v1/* prefixed routes.
            router.GET("/api/v1/chains", svc.GetChains)
            router.GET("/api/v1/price", svc.GetPrice)
            router.GET("/api/v1/gas", svc.GetGas)
            router.POST("/api/v1/gas/estimate", svc.EstimateGas)
            router.GET("/api/v1/network-status", svc.NetworkStatus)
            router.GET("/api/v1/chart/history", svc.ChartHistory)
            router.POST("/api/v1/simulate", svc.SimulateTransaction)
            router.GET("/api/v1/ens/resolve", svc.ENSResolve)
            router.GET("/api/v1/ens/lookup", svc.ENSLookup)
            router.GET("/api/v1/security/check-url", svc.SecurityCheckURL)
            router.GET("/api/v1/security/check-address", svc.SecurityCheckAddress)
            router.POST("/api/v1/security/scan", svc.SecurityScan)
            router.GET("/api/v1/terminal/kline/:symbol", svc.TerminalKline)
            router.GET("/api/v1/terminal/ticker/:symbol", svc.TerminalTicker)
            router.GET("/api/v1/dapps", svc.ListDapps)
            router.GET("/api/v1/dapps/categories", svc.DappCategories)
            router.GET("/api/v1/dapps/:id", svc.GetDapp)
            router.GET("/api/v1/defi/protocols", svc.DefiProtocols)
            router.GET("/api/v1/tokens/registry", svc.TokenRegistry)

		}
	}

            // Public read-only flat routes — mirror canonical /api/v1/* group.
            api.GET("/chains", svc.GetChains)
            api.GET("/price", svc.GetPrice)
            api.GET("/gas", svc.GetGas)
            api.POST("/gas/estimate", svc.EstimateGas)
            api.GET("/network-status", svc.NetworkStatus)
            api.GET("/chart/history", svc.ChartHistory)
            api.POST("/simulate", svc.SimulateTransaction)
            api.GET("/ens/resolve", svc.ENSResolve)
            api.GET("/ens/lookup", svc.ENSLookup)
            api.GET("/security/check-url", svc.SecurityCheckURL)
            api.GET("/security/check-address", svc.SecurityCheckAddress)
            api.POST("/security/scan", svc.SecurityScan)
            api.GET("/terminal/kline/:symbol", svc.TerminalKline)
            api.GET("/terminal/ticker/:symbol", svc.TerminalTicker)
            api.GET("/dapps", svc.ListDapps)
            api.GET("/dapps/categories", svc.DappCategories)
            api.GET("/dapps/:id", svc.GetDapp)
            api.GET("/defi/protocols", svc.DefiProtocols)
            api.GET("/tokens/registry", svc.TokenRegistry)
            router.GET("/api/v1/health", svc.Health)
            api.GET("/dao/delegates", svc.DefiProtocols)
            api.GET("/fees/transactions", svc.ListFees)
            api.GET("/fees/:id", svc.ListFees)
            api.GET("/kyc/document", svc.KycStatus)
            api.GET("/kyc/session/:id", svc.KycStatus)
            api.GET("/address-book/contacts", svc.ListAddressBook)
            api.POST("/address-book/contacts", svc.CreateAddressBook)
            api.PUT("/address-book/contacts/:id", svc.UpdateAddressBook)
            api.DELETE("/address-book/contacts/:id", svc.DeleteAddressBook)
            api.GET("/stats", svc.Health)
            api.GET("/tokens/:chain_id/:symbol", svc.GetTokens)

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
