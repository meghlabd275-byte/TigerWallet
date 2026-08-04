package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"admin_backend/internal/config"
	"admin_backend/internal/handlers"
	"admin_backend/internal/middleware"
	"admin_backend/pkg/auth"
	"admin_backend/pkg/database"
	"admin_backend/pkg/redis"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize PostgreSQL
	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("Connected to PostgreSQL")

	// Initialize Redis
	redisClient, err := redis.NewRedisClient(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()
	log.Println("Connected to Redis")

	// Initialize auth service
	authSvc := auth.NewAuthService(cfg)

	// Create default super admin if not exists
	createDefaultAdmin(db, cfg, authSvc)

	// Initialize handlers
	adminHandler := handlers.NewAdminHandler(db, redisClient, cfg, authSvc)

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.SecurityHeadersMiddleware())
	r.Use(middleware.CORSMiddleware())

	// Health check endpoint (no auth required)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
		})
	})

	// API v1
	v1 := r.Group("/api/v1")
	{
		// Public endpoints
		public := v1.Group("")
		{
			public.POST("/auth/login", adminHandler.Login)
			public.POST("/auth/refresh", adminHandler.RefreshToken)
		}

		// Protected endpoints
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(authSvc))
		{
			// Auth endpoints
			auth := protected.Group("/auth")
			{
				auth.POST("/logout", adminHandler.Logout)
				auth.GET("/profile", adminHandler.GetProfile)
				auth.PUT("/profile", adminHandler.UpdateProfile)
				auth.POST("/change-password", adminHandler.ChangePassword)
			}

			// Admin management (super_admin only)
			admins := protected.Group("/admins")
			admins.Use(middleware.SuperAdminMiddleware())
			{
				admins.GET("", adminHandler.ListAdmins)
				admins.POST("", adminHandler.CreateAdmin)
				admins.GET("/:id", adminHandler.GetAdmin)
				admins.PUT("/:id", adminHandler.UpdateAdmin)
				admins.DELETE("/:id", adminHandler.DeleteAdmin)
				admins.GET("/:id/activities", adminHandler.GetAdminActivities)
			}

			// Dashboard
			protected.GET("/dashboard", adminHandler.GetDashboard)

			// Users
			protected.GET("/users", adminHandler.ListUsers)
			protected.GET("/users/:id", adminHandler.GetUser)
			protected.PUT("/users/:id", adminHandler.UpdateUser)
			protected.DELETE("/users/:id", adminHandler.DeleteUser)
			protected.POST("/users/:id/verify-kyc", adminHandler.VerifyKYC)

			// Transactions
			protected.GET("/transactions", adminHandler.ListTransactions)
			protected.GET("/transactions/:id", adminHandler.GetTransaction)
			protected.POST("/transactions/:id/flag", adminHandler.FlagTransaction)

			// Tokens
			protected.GET("/tokens", adminHandler.ListTokens)
			protected.POST("/tokens", adminHandler.CreateToken)
			protected.PUT("/tokens/:id", adminHandler.UpdateToken)
			protected.DELETE("/tokens/:id", adminHandler.DeleteToken)

			// Withdrawals
			protected.GET("/withdrawals", adminHandler.ListWithdrawals)
			protected.POST("/withdrawals/:id/approve", adminHandler.ApproveWithdrawal)
			protected.POST("/withdrawals/:id/reject", adminHandler.RejectWithdrawal)

			// KYC
			protected.GET("/kyc", adminHandler.ListKYC)
			protected.GET("/kyc/:id", adminHandler.GetKYC)
			protected.POST("/kyc/:id/approve", adminHandler.ApproveKYC)
			protected.POST("/kyc/:id/reject", adminHandler.RejectKYC)

			// White Labels
			protected.GET("/white-labels", adminHandler.ListWhiteLabels)
			protected.POST("/white-labels", adminHandler.CreateWhiteLabel)
			protected.PUT("/white-labels/:id", adminHandler.UpdateWhiteLabel)
			protected.DELETE("/white-labels/:id", adminHandler.DeleteWhiteLabel)

			// System
			protected.GET("/system/config", adminHandler.GetSystemConfig)
			protected.PUT("/system/config", adminHandler.UpdateSystemConfig)
			protected.GET("/system/status", adminHandler.GetSystemStatus)
			protected.GET("/system/metrics", adminHandler.GetSystemMetrics)

			// API Keys
			protected.GET("/api-keys", adminHandler.ListAPIKeys)
			protected.POST("/api-keys", adminHandler.CreateAPIKey)
			protected.DELETE("/api-keys/:id", adminHandler.DeleteAPIKey)

			// Analytics
			protected.GET("/analytics/users", adminHandler.GetUserAnalytics)
			protected.GET("/analytics/transactions", adminHandler.GetTransactionAnalytics)
			protected.GET("/analytics/revenue", adminHandler.GetRevenueAnalytics)
		}
	}

	// Start server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.ServerHost, cfg.ServerPort),
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout:  30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// Graceful shutdown
	go func() {
		log.Printf("Admin backend starting on %s:%s", cfg.ServerHost, cfg.ServerPort)
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

	log.Println("Server exited")
}

func createDefaultAdmin(db *database.PostgresDB, cfg *config.Config, authSvc *auth.AuthService) {
	var admin models.Admin
	result := db.Where("email = ?", cfg.DefaultAdminEmail).First(&admin)
	if result.Error == nil {
		return // Admin already exists
	}

	// Create default admin
	hashedPassword := cfg.DefaultAdminPassword + cfg.PasswordPepper
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(hashedPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Warning: Failed to hash default admin password: %v", err)
		return
	}

	admin = models.Admin{
		Username:     "admin",
		Email:        cfg.DefaultAdminEmail,
		PasswordHash: string(hashedBytes),
		FirstName:   "Super",
		LastName:     "Admin",
		Role:         "super_admin",
		Status:       "active",
		EmailVerified: true,
	}

	if err := db.Create(&admin).Error; err != nil {
		log.Printf("Warning: Failed to create default admin: %v", err)
		return
	}

	log.Println("Created default super admin")
}

// Import context
import "context"

// Import models
import "admin_backend/internal/models"
