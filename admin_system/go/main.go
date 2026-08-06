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

	"admin_system/internal/config"
	"admin_system/internal/database"
	"admin_system/internal/handlers"
	"admin_system/internal/middleware"
	"admin_system/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.Load()
	log.Printf("Starting TigerWallet Admin System Backend...")
	log.Printf("Environment: %s", cfg.Environment)
	log.Printf("Port: %s", cfg.Server.Port)

	// Initialize PostgreSQL
	pgDB, err := database.NewPostgres(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pgDB.Close()

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

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
	} else {
		log.Println("Connected to Redis")
	}

	// Initialize services
	authService := services.NewAuthService(pgDB, rdb)
	systemService := services.NewSystemService(pgDB, rdb)
	monitoringService := services.NewMonitoringService(pgDB, rdb)
	configService := services.NewConfigService(pgDB, rdb)
	backupService := services.NewBackupService(pgDB, rdb)
	logService := services.NewLogService(pgDB, rdb)
	metricsService := services.NewMetricsService(pgDB, rdb)
	alertService := services.NewAlertService(pgDB, rdb)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	systemHandler := handlers.NewSystemHandler(systemService)
	monitoringHandler := handlers.NewMonitoringHandler(monitoringService)
	configHandler := handlers.NewConfigHandler(configService)
	backupHandler := handlers.NewBackupHandler(backupService)
	logHandler := handlers.NewLogHandler(logService)
	metricsHandler := handlers.NewMetricsHandler(metricsService)
	alertHandler := handlers.NewAlertHandler(alertService)

	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()
	router.Use(middleware.CORS())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
			"version":   "2.0.0",
			"service":   "admin_system",
		})
	})

	v1 := router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/logout", authHandler.Logout)
		}

		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(authService))
		{
			// System management
			system := protected.Group("/system")
			{
				system.GET("/info", systemHandler.GetInfo)
				system.GET("/status", systemHandler.GetStatus)
				system.POST("/restart", systemHandler.Restart)
				system.POST("/shutdown", systemHandler.Shutdown)
			}

			// Monitoring
			monitoring := protected.Group("/monitoring")
			{
				monitoring.GET("/metrics", monitoringHandler.GetMetrics)
				monitoring.GET("/resources", monitoringHandler.GetResources)
				monitoring.GET("/processes", monitoringHandler.GetProcesses)
				monitoring.GET("/network", monitoringHandler.GetNetworkStats)
			}

			// Configuration
			config := protected.Group("/config")
			{
				config.GET("", configHandler.GetAll)
				config.GET("/:key", configHandler.Get)
				config.PUT("/:key", configHandler.Update)
				config.DELETE("/:key", configHandler.Delete)
			}

			// Backups
			backup := protected.Group("/backup")
			{
				backup.GET("", backupHandler.List)
				backup.POST("", backupHandler.Create)
				backup.GET("/:id", backupHandler.Get)
				backup.POST("/:id/restore", backupHandler.Restore)
				backup.DELETE("/:id", backupHandler.Delete)
			}

			// Logs
			logs := protected.Group("/logs")
			{
				logs.GET("", logHandler.List)
				logs.GET("/:id", logHandler.Get)
				logs.DELETE("/:id", logHandler.Delete)
				logs.DELETE("", logHandler.DeleteOld)
			}

			// Metrics
			metrics := protected.Group("/metrics")
			{
				metrics.GET("", metricsHandler.Get)
				metrics.GET("/:type", metricsHandler.GetByType)
			}

			// Alerts
			alerts := protected.Group("/alerts")
			{
				alerts.GET("", alertHandler.List)
				alerts.POST("", alertHandler.Create)
				alerts.PUT("/:id/acknowledge", alertHandler.Acknowledge)
				alerts.PUT("/:id/resolve", alertHandler.Resolve)
			}
		}

		// Admin only
		admin := v1.Group("")
		admin.Use(middleware.AdminMiddleware(authService))
		{
			admin.GET("/admin/stats", handlers.GetAdminStats)
		}
	}

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("Server starting on port %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited properly")
}
