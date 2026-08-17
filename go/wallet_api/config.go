package main

import (
	"fmt"
	"os"
	"strconv"
)

// AppConfig holds runtime configuration sourced from environment with sane
// defaults so the service is runnable without an env file.
type AppConfig struct {
	Port            string
	DatabaseURL     string
	RedisAddr       string
	JWTSecret       string
	CoinGeckoAPIKey string
	EtherscanAPIKey string
	AdminBootstrapEmail string
	// MasterWalletBackendURL is the canonical MasterWallet backend (Go, :8450).
	// When set, the UserWallet /send flow optionally asks the MasterWallet owner's
	// auto-sign policy to sponsor/approve the tx within a second (gas-sponsorship +
	// policy gate). Server-to-server only — UserWallet clients never talk to the
	// MasterWallet backend directly (app separation preserved).
	MasterWalletBackendURL string
	// ListingServiceURL is the canonical listing/KYC backend (:8010). Used by
	// the /kyc/* proxy + the P2P KYC gate. Server-to-server only.
	ListingServiceURL string
	// P2PServiceURL is the canonical P2P trading backend (p2p_trading, :8475).
	// Used by the /p2p/* proxy (orders KYC-gated). Server-to-server only.
	P2PServiceURL string
}

// LoadConfig reads configuration from environment variables. Every secret has
// a fallback ONLY for local development; production must set them explicitly.
func LoadConfig() *AppConfig {
	return &AppConfig{
		Port:            envOr("WALLET_API_PORT", "8443"),
		DatabaseURL:     envOr("DATABASE_URL", "postgres://tiger:tiger@localhost:5432/tigerwallet?sslmode=disable"),
		RedisAddr:       envOr("REDIS_ADDR", "localhost:6379"),
		JWTSecret:       envOr("JWT_SECRET", "tigerwallet-dev-secret-change-in-production"),
		CoinGeckoAPIKey: os.Getenv("COINGECKO_API_KEY"),
		EtherscanAPIKey: os.Getenv("ETHERSCAN_API_KEY"),
		AdminBootstrapEmail: os.Getenv("ADMIN_BOOTSTRAP_EMAIL"),
		MasterWalletBackendURL: envOr("MASTER_WALLET_API_URL", "http://localhost:8450"),
		ListingServiceURL:      envOr("LISTING_SERVICE_URL", "http://localhost:8010"),
		P2PServiceURL:          envOr("P2P_SERVICE_URL", "http://localhost:8475"),
	}
}

// envOr returns the env value or a fallback.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// envInt returns the env value as int or a fallback.
func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

// Ensure fmt is used (chains.go references envOr + fmt.Sprintf).
var _ = fmt.Sprintf
