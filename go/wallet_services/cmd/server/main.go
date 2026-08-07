/**
 * TigerWallet Go Services - Main Entry Point
 * 
 * High-performance, distributed wallet services supporting:
 * - Multi-chain wallet operations (130+ blockchains)
 * - Real-time transaction processing
 * - Distributed caching with Redis
 * - PostgreSQL for persistent storage
 * - Horizontal scaling support
 */

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tigerwallet/wallet-services/internal/cache"
	"github.com/tigerwallet/wallet-services/internal/config"
	"github.com/tigerwallet/wallet-services/internal/database"
	"github.com/tigerwallet/wallet-services/internal/handlers"
	"github.com/tigerwallet/wallet-services/internal/middleware"
	"github.com/tigerwallet/wallet-services/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var (
	logger = logrus.New()
	server *http.Server
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("Failed to load config: %v", err)
	}

	// Setup logging
	setupLogging(cfg)

	logger.Info("=================================================")
	logger.Info("  TigerWallet Services Starting...")
	logger.Info("=================================================")
	logger.Infof("Environment: %s", cfg.Environment)
	logger.Infof("Server Port: %d", cfg.Server.Port)
	logger.Infof("Database: %s", cfg.Database.Host)
	logger.Infof("Redis: %s", cfg.Redis.Host)

	// Initialize database connection
	db, err := database.NewPostgresDB(cfg.Database)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.RunMigrations(db); err != nil {
		logger.Fatalf("Failed to run migrations: %v", err)
	}
	logger.Info("Database migrations completed")

	// Initialize Redis cache
	redisClient, err := cache.NewRedisClient(cfg.Redis)
	if err != nil {
		logger.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()
	logger.Info("Redis connection established")

	// Initialize services
	walletService := services.NewWalletService(db, redisClient)
	transactionService := services.NewTransactionService(db, redisClient)
	userService := services.NewUserService(db, redisClient)
	authService := services.NewAuthService(db, redisClient)
	blockchainService := services.NewBlockchainService(db, redisClient)
	tokenService := services.NewTokenService(db, redisClient)
	priceService := services.NewPriceService(db, redisClient)
	notificationService := services.NewNotificationService(db, redisClient)
	analyticsService := services.NewAnalyticsService(db, redisClient)

	// Initialize handlers
	walletHandler := handlers.NewWalletHandler(walletService)
	transactionHandler := handlers.NewTransactionHandler(transactionService)
	userHandler := handlers.NewUserHandler(userService)
	authHandler := handlers.NewAuthHandler(authService)
	blockchainHandler := handlers.NewBlockchainHandler(blockchainService)
	tokenHandler := handlers.NewTokenHandler(tokenService)
	priceHandler := handlers.NewPriceHandler(priceService)
	notificationHandler := handlers.NewNotificationHandler(notificationService)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsService)

	// Setup router
	router := setupRouter(cfg, authHandler, userHandler, walletHandler, 
		transactionHandler, blockchainHandler, tokenHandler, 
		priceHandler, notificationHandler, analyticsHandler)

	// Create HTTP server
	server = &http.Server{
		Addr:           fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:        router,
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
	}

	// Start server in goroutine
	go func() {
		logger.Infof("Server starting on port %d", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited gracefully")
}

func setupLogging(cfg *config.Config) {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
		logger.SetLevel(logrus.InfoLevel)
	} else {
		gin.SetMode(gin.DebugMode)
		logger.SetLevel(logrus.DebugLevel)
	}

	// Log format
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})
}

func setupRouter(cfg *config.Config, 
	authHandler *handlers.AuthHandler,
	userHandler *handlers.UserHandler,
	walletHandler *handlers.WalletHandler,
	transactionHandler *handlers.TransactionHandler,
	blockchainHandler *handlers.BlockchainHandler,
	tokenHandler *handlers.TokenHandler,
	priceHandler *handlers.PriceHandler,
	notificationHandler *handlers.NotificationHandler,
	analyticsHandler *handlers.AnalyticsHandler) *gin.Engine {

	router := gin.New()
	
	// Global middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())
	router.Use(middleware.RateLimiter())
	router.Use(middleware.RequestID())
	router.Use(middleware.Timeout())
	router.Use(middleware.Compression())

	// Health check
	router.GET("/health", handlers.HealthCheck)
	router.GET("/ready", handlers.ReadinessCheck)

	// API v1
	v1 := router.Group("/api/v1")
	{
		// Auth routes (public)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/forgot-password", authHandler.ForgotPassword)
			auth.POST("/reset-password", authHandler.ResetPassword)
			auth.POST("/verify-email", authHandler.VerifyEmail)
			auth.POST("/resend-verification", authHandler.ResendVerification)
		}

		// Protected routes
		protected := v1.Group("")
		protected.Use(middleware.AuthRequired(cfg.JWT.Secret))
		{
			// User management
			user := protected.Group("/user")
			{
				user.GET("/profile", userHandler.GetProfile)
				user.PUT("/profile", userHandler.UpdateProfile)
				user.PUT("/settings", userHandler.UpdateSettings)
				user.POST("/change-password", userHandler.ChangePassword)
				user.DELETE("/account", userHandler.DeleteAccount)
				user.POST("/enable-2fa", userHandler.Enable2FA)
				user.POST("/disable-2fa", userHandler.Disable2FA)
			}

			// Wallet management
			wallet := protected.Group("/wallets")
			{
				wallet.GET("", walletHandler.ListWallets)
				wallet.POST("", walletHandler.CreateWallet)
				wallet.GET("/:id", walletHandler.GetWallet)
				wallet.DELETE("/:id", walletHandler.DeleteWallet)
				wallet.GET("/:id/balance", walletHandler.GetBalance)
				wallet.GET("/:id/transactions", walletHandler.GetTransactions)
				wallet.POST("/:id/export", walletHandler.ExportWallet)
				wallet.POST("/:id/import", walletHandler.ImportWallet)
			}

			// Address management
			address := protected.Group("/addresses")
			{
				address.GET("", walletHandler.ListAddresses)
				address.POST("", walletHandler.CreateAddress)
				address.GET("/:id/qr", walletHandler.GetAddressQR)
				address.DELETE("/:id", walletHandler.DeleteAddress)
			}

			// Transaction management
			tx := protected.Group("/transactions")
			{
				tx.GET("", transactionHandler.ListTransactions)
				tx.POST("", transactionHandler.CreateTransaction)
				tx.GET("/:id", transactionHandler.GetTransaction)
				tx.POST("/:id/cancel", transactionHandler.CancelTransaction)
				tx.POST("/:id/accelerate", transactionHandler.AccelerateTransaction)
				tx.POST("/:id/replace", transactionHandler.ReplaceByFee)
				tx.GET("/pending", transactionHandler.GetPendingTransactions)
				tx.GET("/history", transactionHandler.GetTransactionHistory)
			}

			// Token management
			token := protected.Group("/tokens")
			{
				token.GET("", tokenHandler.ListTokens)
				token.POST("", tokenHandler.AddToken)
				token.DELETE("/:id", tokenHandler.RemoveToken)
				token.GET("/:id/price", tokenHandler.GetTokenPrice)
				token.GET("/:id/history", tokenHandler.GetTokenPriceHistory)
			}

			// Price/Portfolio
			price := protected.Group("/prices")
			{
				price.GET("", priceHandler.GetPrices)
				price.GET("/:symbol", priceHandler.GetPrice)
				price.GET("/historical", priceHandler.GetHistoricalPrices)
				price.GET("/markets", priceHandler.GetMarketData)
			}

			// Blockchain info
			blockchain := protected.Group("/blockchain")
			{
				blockchain.GET("/chains", blockchainHandler.ListChains)
				blockchain.GET("/chains/:id", blockchainHandler.GetChain)
				blockchain.GET("/chains/:id/gas", blockchainHandler.GetGasPrice)
				blockchain.GET("/chains/:id/nonce", blockchainHandler.GetNonce)
				blockchain.GET("/chains/:id/block", blockchainHandler.GetBlock)
				blockchain.POST("/chains/:id/broadcast", blockchainHandler.BroadcastTransaction)
			}

			// Notifications
			notification := protected.Group("/notifications")
			{
				notification.GET("", notificationHandler.ListNotifications)
				notification.PUT("/:id/read", notificationHandler.MarkAsRead)
				notification.PUT("/read-all", notificationHandler.MarkAllAsRead)
				notification.DELETE("/:id", notificationHandler.DeleteNotification)
				notification.POST("/settings", notificationHandler.UpdateSettings)
			}

			// Analytics
			analytics := protected.Group("/analytics")
			{
				analytics.GET("/portfolio", analyticsHandler.GetPortfolio)
				analytics.GET("/portfolio/history", analyticsHandler.GetPortfolioHistory)
				analytics.GET("/transactions/summary", analyticsHandler.GetTransactionSummary)
				analytics.GET("/transactions/by-chain", analyticsHandler.GetTransactionsByChain)
				analytics.GET("/gas/summary", analyticsHandler.GetGasSummary)
				analytics.GET("/tax/report", analyticsHandler.GetTaxReport)
			}
		}

		// WebSocket for real-time updates
		ws := v1.Group("/ws")
		{
			ws.GET("/prices", handlers.HandlePriceWebSocket)
			ws.GET("/transactions", handlers.HandleTransactionWebSocket)
			ws.GET("/notifications", handlers.HandleNotificationWebSocket)
		}
	}

	// Public API endpoints (no auth required)
	public := v1.Group("/public")
	{
		public.GET("/tokens", tokenHandler.ListPublicTokens)
		public.GET("/prices", priceHandler.GetPublicPrices)
		public.GET("/chains", blockchainHandler.ListPublicChains)
		public.GET("/gas-oracle", blockchainHandler.GetGasOracle)
	}

	// Documentation
	router.GET("/api/docs", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"version": "1.0.0",
			"endpoints": "/api/v1",
			"docs": "https://docs.tigerwallet.io",
		})
	})

	// 404 handler
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Endpoint not found",
			"code": 404,
		})
	})

	return router
}
