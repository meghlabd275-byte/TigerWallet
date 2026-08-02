package main

import (
	"context"
	"fmt"
	"log"
	"master_wallet"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Initialize services
	_ = master_wallet.GetRegistry()
	_ = master_wallet.GetTokenRegistry()
	_ = master_wallet.GetMasterWalletService()
	_ = master_wallet.GetTigerWalletService()
	_ = master_wallet.GetCustomBrandingService()

	// Setup router
	adminAPI := master_wallet.GetAdminAPIService()
	router := adminAPI.SetupRouter()

	// Get port from environment or default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create server
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Print startup information
	registry := master_wallet.GetRegistry()
	tokenRegistry := master_wallet.GetTokenRegistry()
	masterService := master_wallet.GetMasterWalletService()

	fmt.Println("======================================")
	fmt.Println("  TigerWallet Master Wallet System")
	fmt.Println("======================================")
	fmt.Printf("\n✅ Services Initialized Successfully\n\n")
	fmt.Printf("📊 Network Statistics:\n")
	fmt.Printf("   - Total Networks: %d\n", registry.GetSupportedChains())
	fmt.Printf("   - Active Networks: %d\n", registry.GetActiveChainCount())
	fmt.Printf("   - Total Tokens: %d\n", tokenRegistry.GetTokenCount())
	
	// Get TigerWallet master wallet
	tigerWallet := masterService.GetMasterWalletByType("tiger")
	if tigerWallet != nil {
		fmt.Printf("\n🐯 TigerWallet Master:\n")
		fmt.Printf("   - ID: %s\n", tigerWallet.ID)
		fmt.Printf("   - Networks: %d\n", len(tigerWallet.NetworkIDs))
		fmt.Printf("   - Tokens: %d\n", len(tigerWallet.TokenIDs))
	}
	
	fmt.Printf("\n🌐 Server starting on port %s\n", port)
	fmt.Println("======================================\n")

	// Start server in goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited properly")
}
