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
