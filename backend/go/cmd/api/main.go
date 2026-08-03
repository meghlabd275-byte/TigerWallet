package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/tigerwallet/backend/internal/config"
	"github.com/tigerwallet/backend/internal/database"
	"github.com/tigerwallet/backend/internal/handlers"
	"github.com/tigerwallet/backend/internal/middleware"
	"github.com/tigerwallet/backend/internal/models"
	"github.com/tigerwallet/backend/internal/websocket"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize Redis for caching/sessions
	// redisClient := database.ConnectRedis(cfg)

	// Initialize WebSocket hub
	wsHub := websocket.NewHub()
	go wsHub.Run()

	// Initialize handlers
	p2pHandler := handlers.NewP2PHandler(db, wsHub)
	marginHandler := handlers.NewMarginHandler(db)
	walletHandler := handlers.NewWalletHandler(db)
	authHandler := handlers.NewAuthHandler(db, cfg)

	// Setup router
	router := gin.Default()

	// Security middleware
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.RateLimiter())

	// CORS
	router.Use(middleware.CORS())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "ok",
			"timestamp": "2024-01-01T00:00:00Z",
			"version":   "1.0.0",
		})
	})

	// API routes
	api := router.Group("/api/v1")
	{
		// Auth routes (public)
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
		}

		// Protected routes
		protected := api.Group("")
		protected.Use(middleware.Authenticate(cfg))
		{
			// P2P routes
			p2p := protected.Group("/p2p")
			{
				p2p.GET("/adverts", p2pHandler.GetAdverts)
				p2p.POST("/orders", p2pHandler.CreateOrder)
				p2p.GET("/orders", p2pHandler.GetOrders)
				p2p.GET("/orders/:id", p2pHandler.GetOrder)
				p2p.POST("/orders/:id/pay", p2pHandler.MarkAsPaid)
				p2p.POST("/orders/:id/confirm", p2pHandler.ConfirmPayment)
				p2p.POST("/orders/:id/cancel", p2pHandler.CancelOrder)
				p2p.POST("/orders/:id/dispute", p2pHandler.OpenDispute)
				p2p.GET("/payment-methods", p2pHandler.GetPaymentMethods)
				p2p.GET("/fiat-currencies", p2pHandler.GetFiatCurrencies)
			}

			// Merchant routes
			merchants := protected.Group("/merchants")
			{
				merchants.POST("/apply", p2pHandler.ApplyMerchant)
				merchants.GET("/profile", p2pHandler.GetMerchantProfile)
				merchants.POST("/collateral/add", p2pHandler.AddCollateral)
			}

			// Margin trading routes
			margin := protected.Group("/margin")
			{
				margin.GET("/account", marginHandler.GetAccount)
				margin.POST("/borrow", marginHandler.Borrow)
				margin.POST("/repay", marginHandler.Repay)
				margin.GET("/positions", marginHandler.GetPositions)
				margin.POST("/positions", marginHandler.OpenPosition)
				margin.DELETE("/positions/:id", marginHandler.ClosePosition)
			}

			// Wallet routes
			wallet := protected.Group("/wallet")
			{
				wallet.GET("/balance", walletHandler.GetBalance)
				wallet.POST("/transfer", walletHandler.Transfer)
				wallet.GET("/transactions", walletHandler.GetTransactions)
			}
		}

		// WebSocket
		api.GET("/ws", websocket.HandleWebSocket(wsHub, cfg))
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down server...")
		wsHub.Stop()
		log.Println("Server stopped")
	}()

	// Start server
	port := cfg.Server.Port
	log.Printf("TigerWallet API starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
