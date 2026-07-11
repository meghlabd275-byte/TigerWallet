package routes

import (
	"github.com/gin-gonic/gin"
	"tigerwallet/backend/go/api/handlers"
	"tigerwallet/backend/go/api/middleware"
	"tigerwallet/backend/go/api/config"
)

func Setup(r *gin.Engine, h *handlers.Handler, cfg *config.Config) {
	// Public routes
	r.POST("/api/v1/auth/register", h.Register)
	r.POST("/api/v1/auth/login", h.Login)
	r.POST("/api/v1/auth/refresh", h.RefreshToken)
	
	// Blockchain routes
	r.GET("/api/v1/blockchains", h.GetBlockchains)
	r.GET("/api/v1/blockchains/:id", h.GetBlockchain)
	r.POST("/api/v1/blockchains", h.AddBlockchain)
	r.PUT("/api/v1/blockchains/:id", h.UpdateBlockchain)
	r.DELETE("/api/v1/blockchains/:id", h.DeleteBlockchain)

	// Token routes
	r.GET("/api/v1/tokens", h.GetTokens)
	r.GET("/api/v1/tokens/:id", h.GetToken)
	r.POST("/api/v1/tokens", h.AddToken)
	r.PUT("/api/v1/tokens/:id", h.UpdateToken)
	r.DELETE("/api/v1/tokens/:id", h.DeleteToken)

	// Wallet routes (authenticated)
	wallet := r.Group("/api/v1/wallets")
	wallet.Use(middleware.Auth(cfg.JWTSecret))
	{
		wallet.POST("", h.CreateWallet)
		wallet.GET("", h.GetWallets)
		wallet.GET("/:id", h.GetWallet)
		wallet.DELETE("/:id", h.DeleteWallet)
		wallet.GET("/:id/balance", h.GetBalance)
		wallet.POST("/:id/export", h.ExportWallet)
		wallet.POST("/import", h.ImportWallet)
	}

	// Transaction routes
	tx := r.Group("/api/v1/transactions")
	tx.Use(middleware.Auth(cfg.JWTSecret))
	{
		tx.POST("", h.CreateTransaction)
		tx.GET("", h.GetTransactions)
		tx.GET("/:id", h.GetTransaction)
		tx.POST("/:id/sign", h.SignTransaction)
		tx.POST("/:id/broadcast", h.BroadcastTransaction)
		tx.POST("/:id/cancel", h.CancelTransaction)
	}

	// Swap routes
	swap := r.Group("/api/v1/swap")
	swap.Use(middleware.Auth(cfg.JWTSecret))
	{
		swap.GET("/quote", h.GetSwapQuote)
		swap.POST("/execute", h.ExecuteSwap)
	}

	// Perpetual trading routes
	perp := r.Group("/api/v1/perpetual")
	perp.Use(middleware.Auth(cfg.JWTSecret))
	{
		perp.GET("/markets", h.GetPerpetualMarkets)
		perp.GET("/positions", h.GetPerpetualPositions)
		perp.POST("/orders", h.CreatePerpetualOrder)
		perp.DELETE("/orders/:id", h.CancelPerpetualOrder)
	}

	// Copy trading routes
	copy := r.Group("/api/v1/copy")
	copy.Use(middleware.Auth(cfg.JWTSecret))
	{
		copy.GET("/traders", h.GetCopyTraders)
		copy.POST("/follow", h.FollowTrader)
		copy.DELETE("/unfollow/:id", h.UnfollowTrader)
		copy.GET("/trades", h.GetCopyTrades)
	}

	// Staking routes
	staking := r.Group("/api/v1/staking")
	staking.Use(middleware.Auth(cfg.JWTSecret))
	{
		staking.GET("/pools", h.GetStakingPools)
		staking.POST("/stake", h.Stake)
		staking.POST("/unstake", h.Unstake)
		staking.POST("/claim", h.ClaimRewards)
	}

	// NFT routes
	nft := r.Group("/api/v1/nft")
	nft.Use(middleware.Auth(cfg.JWTSecret))
	{
		nft.GET("/collections", h.GetNFTCollections)
		nft.GET("", h.GetNFTs)
		nft.POST("/transfer", h.TransferNFT)
	}

	// Admin routes (admin only)
	admin := r.Group("/api/v1/admin")
	admin.Use(middleware.AdminAuth(cfg.AdminAPIKey))
	{
		admin.GET("/stats", h.GetStats)
		admin.POST("/fees", h.UpdateFeeConfig)
		admin.GET("/users", h.GetAllUsers)
	}
}
