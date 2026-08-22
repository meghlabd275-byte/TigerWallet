// TigerWallet Launchpad Ecosystem - service entrypoint.
//
// This is the only `package main` in launchpad_ecosystem/go. It imports the
// launchpad subpackage (which owns Config, LaunchpadService and all models)
// and wires up the HTTP server + background status updater.

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"

	launchpad "github.com/tigerwallet/launchpad-ecosystem/launchpad"
)

func main() {
	config := launchpad.LoadConfig()

	service, err := launchpad.NewLaunchpadService(config)
	if err != nil {
		fmt.Printf("Failed to start launchpad service: %v\n", err)
		os.Exit(1)
	}

	router := gin.Default()

	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	service.RegisterRoutes(router)

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "launchpad"})
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service.StartStatusUpdater(ctx)

	go func() {
		fmt.Printf("Launchpad service starting on port %s\n", config.ServerPort)
		if err := router.Run(":" + config.ServerPort); err != nil {
			fmt.Printf("Failed to start server: %v\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	cancel()

	fmt.Println("Shutting down launchpad service...")
}
