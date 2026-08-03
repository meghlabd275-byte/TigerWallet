package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Configuration
type Config struct {
	// Server
	Port string
	
	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBMaxConns int32
	
	// Redis
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int
	
	// Security
	JWTSecret     string
	SessionExpiry time.Duration
}

var (
	logger   *log.Logger
	dbPool   *pgxpool.Pool
	redisClient *redis.Client
)

func main() {
	logger = log.New(os.Stdout, "[AdminPlatform] ", log.LstdFlags)
	logger.Println("Starting TigerWallet Admin Platform...")
	
	cfg := loadConfig()
	
	// Initialize database
	if err := initDatabase(cfg); err != nil {
		logger.Printf("Database initialization failed: %v", err)
		logger.Println("Continuing with in-memory storage...")
	}
	
	// Initialize Redis
	if err := initRedis(cfg); err != nil {
		logger.Printf("Redis initialization failed: %v", err)
		logger.Println("Continuing without Redis cache...")
	}
	
	// Setup router
	router := setupRouter(cfg)
	
	// Start server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}
	
	go func() {
		logger.Printf("Admin Platform starting on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server error: %v", err)
		}
	}()
	
	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	logger.Println("Shutting down server...")
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := srv.Shutdown(ctx); err != nil {
		logger.Printf("Server forced to shutdown: %v", err)
	}
	
	if dbPool != nil {
		dbPool.Close()
		logger.Println("Database connection closed")
	}
	
	if redisClient != nil {
		redisClient.Close()
		logger.Println("Redis connection closed")
	}
	
	logger.Println("Server exited")
}

func loadConfig() *Config {
	return &Config{
		Port:       getEnv("ADMIN_PORT", "8081"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "tigerwallet"),
		DBPassword: getEnv("DB_PASSWORD", "tigerwallet"),
		DBName:     getEnv("DB_NAME", "tigerwallet_admin"),
		DBMaxConns: 50,
		RedisHost:  getEnv("REDIS_HOST", "localhost"),
		RedisPort:  getEnv("REDIS_PORT", "6379"),
		JWTSecret: getEnv("JWT_SECRET", "admin-platform-secret-key"),
		SessionExpiry: 24 * time.Hour,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func initDatabase(cfg *Config) error {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&pool_max_conns=%d",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBMaxConns)
	
	var err error
	dbPool, err = pgxpool.New(context.Background(), dsn)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := dbPool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	
	logger.Println("Connected to PostgreSQL database")
	return nil
}

func initRedis(cfg *Config) error {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}
	
	logger.Println("Connected to Redis")
	return nil
}

func setupRouter(cfg *Config) *gin.Engine {
	router := gin.Default()
	
	// CORS middleware
	router.Use(corsMiddleware())
	
	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "admin-platform",
			"timestamp": time.Now().Unix(),
		})
	})
	
	// API v1
	v1 := router.Group("/api/v1")
	{
		// Admin authentication
		auth := v1.Group("/auth")
		{
			auth.POST("/login", handleLogin)
			auth.POST("/logout", handleLogout)
			auth.POST("/refresh", handleRefreshToken)
		}
		
		// Protected routes
		protected := v1.Group("")
		protected.Use(authMiddleware(cfg.JWTSecret))
		{
			// Dashboard
			protected.GET("/dashboard", handleGetDashboard)
			protected.GET("/dashboard/stats", handleGetStats)
			
			// Users
			users := protected.Group("/users")
			{
				users.GET("", handleListUsers)
				users.GET("/:id", handleGetUser)
				users.PUT("/:id", handleUpdateUser)
				users.POST("/:id/ban", handleBanUser)
				users.POST("/:id/unban", handleUnbanUser)
				users.POST("/:id/suspend", handleSuspendUser)
			}
			
			// KYC
			kyc := protected.Group("/kyc")
			{
				kyc.GET("", handleListKYC)
				kyc.GET("/:id", handleGetKYC)
				kyc.POST("/:id/approve", handleApproveKYC)
				kyc.POST("/:id/reject", handleRejectKYC)
			}
			
			// Transactions
			tx := protected.Group("/transactions")
			{
				tx.GET("", handleListTransactions)
				tx.GET("/:id", handleGetTransaction)
			}
			
			// Trading Pairs
			pairs := protected.Group("/pairs")
			{
				pairs.GET("", handleListPairs)
				pairs.POST("", handleCreatePair)
				pairs.PUT("/:id", handleUpdatePair)
				pairs.POST("/:id/suspend", handleSuspendPair)
				pairs.POST("/:id/resume", handleResumePair)
				pairs.POST("/:id/halt", handleHaltPair)
			}
			
			// Blockchains
			blockchains := protected.Group("/blockchains")
			{
				blockchains.GET("", handleListBlockchains)
				blockchains.POST("", handleCreateBlockchain)
				blockchains.PUT("/:id", handleUpdateBlockchain)
				blockchains.POST("/:id/enable", handleEnableBlockchain)
				blockchains.POST("/:id/disable", handleDisableBlockchain)
			}
			
			// Fees
			fees := protected.Group("/fees")
			{
				fees.GET("", handleListFees)
				fees.POST("", handleCreateFee)
				fees.PUT("/:id", handleUpdateFee)
			}
			
			// Admins
			admins := protected.Group("/admins")
			{
				admins.GET("", handleListAdmins)
				admins.POST("", handleCreateAdmin)
				admins.PUT("/:id", handleUpdateAdmin)
				admins.DELETE("/:id", handleDeleteAdmin)
			}
			
			// White Labels
			wl := protected.Group("/white-labels")
			{
				wl.GET("", handleListWhiteLabels)
				wl.POST("", handleCreateWhiteLabel)
				wl.PUT("/:id", handleUpdateWhiteLabel)
				wl.DELETE("/:id", handleDeleteWhiteLabel)
				wl.POST("/:id/approve", handleApproveWhiteLabel)
				wl.POST("/:id/suspend", handleSuspendWhiteLabel)
			}
			
			// Audit Logs
			protected.GET("/audit-logs", handleListAuditLogs)
			
			// Settings
			settings := protected.Group("/settings")
			{
				settings.GET("", handleGetSettings)
				settings.PUT("", handleUpdateSettings)
			}
		}
	}
	
	return router
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	}
}
