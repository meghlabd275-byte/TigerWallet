package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/tigerwallet/multi-tenancy/internal/config"
	"github.com/tigerwallet/multi-tenancy/internal/database"
	"github.com/tigerwallet/multi-tenancy/internal/handlers"
	"github.com/tigerwallet/multi-tenancy/internal/services"
)

// corsMiddleware sets CORS headers for the control-plane API. Written inline so
// the module needs no extra dependency.
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func buildRouter(h *handlers.Handlers) *gin.Engine {
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	router.Use(gin.Recovery(), gin.Logger(), corsMiddleware())

	registerMetrics(router)

	router.GET("/health", h.Health)
	router.GET("/readyz", h.Ready)

	api := router.Group("/api/v1")
	{
		api.POST("/tenants", h.CreateTenant)
		api.GET("/tenants", h.ListTenants)
		api.GET("/tenants/slug/:slug", h.GetTenantBySlug)
		api.GET("/tenants/:id", h.GetTenant)
		api.PUT("/tenants/:id", h.UpdateTenant)
		api.PUT("/tenants/:id/status", h.UpdateTenantStatus)
		api.DELETE("/tenants/:id", h.DeleteTenant)

		api.GET("/tenants/:id/users", h.ListUsers)
		api.POST("/tenants/:id/users", h.AddUser)
		api.PUT("/tenants/:id/users/:user_id/role", h.UpdateUserRole)
		api.DELETE("/tenants/:id/users/:user_id", h.RemoveUser)

		api.POST("/tenants/:id/invitations", h.CreateInvitation)
		api.POST("/invitations/:token/accept", h.AcceptInvitation)

		api.GET("/tenants/:id/quota", h.GetQuota)
		api.POST("/tenants/:id/quota", h.IncrementQuota)
		api.GET("/tenants/:id/quota/check", h.CheckQuota)

		api.GET("/tenants/:id/config", h.GetConfig)
		api.PUT("/tenants/:id/config", h.UpdateConfig)

		api.GET("/tenants/:id/audit", h.GetAuditLogs)
		api.POST("/tenants/:id/audit", h.CreateAuditLog)
	}
	return router
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to create PostgreSQL pool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("PostgreSQL is not reachable: %v", err)
	}

	if err := database.Migrate(ctx, pool); err != nil {
		log.Fatalf("failed to apply schema: %v", err)
	}

	rdb, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("invalid Redis URL: %v", err)
	}
	redisClient := redis.NewClient(rdb)
	defer redisClient.Close()

	metricsPool = pool
	metricsRedis = redisClient

	services.Initialize(pool, redisClient)

	h := handlers.New(services.NewTenantService())

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      buildRouter(h),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("multi-tenancy service listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}
