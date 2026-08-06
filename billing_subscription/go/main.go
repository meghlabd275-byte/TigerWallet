package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/tigerwallet/billing/internal/config"
	"github.com/tigerwallet/billing/internal/database"
	"github.com/tigerwallet/billing/internal/handlers"
	"github.com/tigerwallet/billing/internal/middleware"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	if err := database.Initialize(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	log.Println("Database initialized successfully")

	// Initialize Gin router
	router := gin.Default()

	// CORS configuration
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Tenant-ID"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "tiger-billing",
			"timestamp": time.Now().Unix(),
		})
	})

	// Initialize handlers
	billingHandler := handlers.NewBillingHandler()

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Public routes
		public := v1.Group("/public")
		{
			public.GET("/plans", billingHandler.GetPlans)
			public.GET("/plans/:id", billingHandler.GetPlan)
			public.GET("/tenants/:slug", billingHandler.GetTenantBySlug)
		}

		// Protected routes
		protected := v1.Group("")
		protected.Use(middleware.JWTAuth(cfg))
		{
			// Tenant management
			tenants := protected.Group("/tenants")
			{
				tenants.POST("", billingHandler.CreateTenant)
				tenants.GET("/:id", billingHandler.GetTenant)
				tenants.PUT("/:id/status", billingHandler.UpdateTenantStatus)
			}

			// Subscription management
			subs := protected.Group("/subscriptions")
			{
				subs.GET("/:tenant_id", billingHandler.GetSubscription)
				subs.POST("/:tenant_id/upgrade", billingHandler.UpgradeSubscription)
				subs.POST("/:tenant_id/cancel", billingHandler.CancelSubscription)
			}

			// Usage tracking
			usage := protected.Group("/usage")
			{
				usage.GET("/:tenant_id", billingHandler.GetUsage)
				usage.POST("/:tenant_id/record", billingHandler.RecordUsage)
				usage.GET("/:tenant_id/check", billingHandler.CheckQuota)
			}

			// Invoice management
			invoices := protected.Group("/invoices")
			{
				invoices.GET("/:tenant_id", billingHandler.GetInvoices)
				invoices.GET("/invoice/:id", billingHandler.GetInvoice)
			}
		}

		// Webhook routes (for Stripe, etc.)
		webhooks := v1.Group("/webhooks")
		{
			webhooks.POST("/stripe", handleStripeWebhook)
		}
	}

	// Start server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	log.Printf("Billing service started on port %s", cfg.Server.Port)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// Placeholder for Stripe webhook handler
func handleStripeWebhook(c *gin.Context) {
	// In production, this would verify the Stripe signature and process events
	c.JSON(http.StatusOK, gin.H{"message": "webhook received"})
}
