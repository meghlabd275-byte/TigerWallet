package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"admin_console/internal/config"
	"admin_console/internal/database"
	"admin_console/internal/handlers"
	"admin_console/internal/middleware"
	"admin_console/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	log.Printf("Starting TigerWallet Admin Console Backend...")
	log.Printf("Environment: %s", cfg.Environment)
	log.Printf("Port: %s", cfg.Server.Port)

	// Initialize PostgreSQL
	pgDB, err := database.NewPostgres(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pgDB.Close()

	// Run migrations
	if err := database.RunMigrations(pgDB); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()

	// Test Redis connection
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
	} else {
		log.Println("Connected to Redis")
	}

	// Initialize services
	authService := services.NewAuthService(pgDB, rdb)
	userService := services.NewUserService(pgDB, rdb)
	kycService := services.NewKYCService(pgDB, rdb)
	tokenService := services.NewTokenService(pgDB, rdb)
	transactionService := services.NewTransactionService(pgDB, rdb)
	analyticsService := services.NewAnalyticsService(pgDB, rdb)
	auditService := services.NewAuditService(pgDB, rdb)
	notificationService := services.NewNotificationService(rdb)
	complianceService := services.NewComplianceService(pgDB, rdb)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService)
	kycHandler := handlers.NewKYCHandler(kycService)
	tokenHandler := handlers.NewTokenHandler(tokenService)
	transactionHandler := handlers.NewTransactionHandler(transactionService)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsService)
	auditHandler := handlers.NewAuditHandler(auditService)
	notificationHandler := handlers.NewNotificationHandler(notificationService)
	complianceHandler := handlers.NewComplianceHandler(complianceService)

	// Setup Gin router
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// CORS middleware
	router.Use(middleware.CORS())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
			"version":   "2.0.0",
		})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Auth routes (public)
		auth := v1.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/register", authHandler.Register)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/logout", authHandler.Logout)
			auth.POST("/forgot-password", authHandler.ForgotPassword)
			auth.POST("/reset-password", authHandler.ResetPassword)
			auth.POST("/verify-email", authHandler.VerifyEmail)
		}

		// Protected routes
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(authService))
		{
			// User management
			users := protected.Group("/users")
			{
				users.GET("", userHandler.List)
				users.GET("/:id", userHandler.Get)
				users.POST("", userHandler.Create)
				users.PUT("/:id", userHandler.Update)
				users.DELETE("/:id", userHandler.Delete)
				users.PUT("/:id/status", userHandler.UpdateStatus)
				users.GET("/:id/activity", userHandler.GetActivity)
			}

			// KYC management
			kyc := protected.Group("/kyc")
			{
				kyc.GET("", kycHandler.List)
				kyc.GET("/:id", kycHandler.Get)
				kyc.POST("/:id/approve", kycHandler.Approve)
				kyc.POST("/:id/reject", kycHandler.Reject)
				kyc.POST("/:id/request-info", kycHandler.RequestInfo)
				kyc.GET("/stats", kycHandler.GetStats)
			}

			// Token management
			tokens := protected.Group("/tokens")
			{
				tokens.GET("", tokenHandler.List)
				tokens.GET("/:id", tokenHandler.Get)
				tokens.POST("", tokenHandler.Create)
				tokens.PUT("/:id", tokenHandler.Update)
				tokens.DELETE("/:id", tokenHandler.Delete)
				tokens.POST("/:id/approve", tokenHandler.Approve)
				tokens.POST("/:id/reject", tokenHandler.Reject)
				tokens.GET("/:id/holders", tokenHandler.GetHolders)
			}

			// Transaction management
			transactions := protected.Group("/transactions")
			{
				transactions.GET("", transactionHandler.List)
				transactions.GET("/:id", transactionHandler.Get)
				transactions.POST("/:id/flag", transactionHandler.Flag)
				transactions.POST("/:id/approve", transactionHandler.Approve)
				transactions.POST("/:id/reject", transactionHandler.Reject)
				transactions.POST("/:id/cancel", transactionHandler.Cancel)
				transactions.GET("/stats", transactionHandler.GetStats)
				transactions.GET("/pending", transactionHandler.GetPending)
			}

			// Analytics
			analytics := protected.Group("/analytics")
			{
				analytics.GET("/dashboard", analyticsHandler.Dashboard)
				analytics.GET("/users", analyticsHandler.UserAnalytics)
				analytics.GET("/transactions", analyticsHandler.TransactionAnalytics)
				analytics.GET("/revenue", analyticsHandler.RevenueAnalytics)
				analytics.GET("/growth", analyticsHandler.GrowthAnalytics)
			}

			// Audit logs
			audit := protected.Group("/audit")
			{
				audit.GET("", auditHandler.List)
				audit.GET("/:id", auditHandler.Get)
				audit.GET("/user/:userId", auditHandler.GetByUser)
				audit.GET("/export", auditHandler.Export)
			}

			// Notifications
			notifications := protected.Group("/notifications")
			{
				notifications.GET("", notificationHandler.List)
				notifications.PUT("/:id/read", notificationHandler.MarkRead)
				notifications.PUT("/read-all", notificationHandler.MarkAllRead)
				notifications.POST("/send", notificationHandler.Send)
			}

			// Compliance
			compliance := protected.Group("/compliance")
			{
				compliance.GET("/reports", complianceHandler.ListReports)
				compliance.POST("/reports", complianceHandler.CreateReport)
				compliance.GET("/reports/:id", complianceHandler.GetReport)
				compliance.GET("/sanctions", complianceHandler.CheckSanctions)
				compliance.POST("/aml-check", complianceHandler.AMLCheck)
			}
		}

		// Admin only routes
		admin := v1.Group("")
		admin.Use(middleware.AdminMiddleware(authService))
		{
			admin.GET("/stats", handlers.GetSystemStats)
			admin.GET("/config", handlers.GetConfig)
			admin.PUT("/config", handlers.UpdateConfig)
		}
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server starting on port %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}
