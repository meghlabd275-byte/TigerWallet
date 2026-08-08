package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

type Config struct {
	// Server
	Port        int
	ReleaseMode bool
	TLSEnabled  bool
	TLSCert     string
	TLSKey      string

	// Database
	DatabaseURL string

	// Redis
	RedisURL      string
	RedisPassword string

	// JWT
	JWTSecret          string
	JWTExpirationHours int

	// CORS
	CORSOrigins []string

	// Services
	WalletServiceURL     string
	SwapServiceURL       string
	PerpetualServiceURL  string
	CopyTradingURL       string
	BlockchainServiceURL string
	TokenServiceURL      string
	PortfolioServiceURL  string
	NotificationURL      string

	// Security
	MaxRequestSize int64
	RateLimit      int
	BurstLimit     int

	// Blockchain
	DefaultChainID uint64
	Confirmations  int

	// Admin
	SuperAdminAddresses []string
	AdminAPIKey         string
}

func Load() *Config {
	return &Config{
		Port:        getEnvInt("PORT", 8080),
		ReleaseMode: getEnvBool("RELEASE_MODE", false),
		TLSEnabled:  getEnvBool("TLS_ENABLED", false),
		TLSCert:     getEnv("TLS_CERT", ""),
		TLSKey:      getEnv("TLS_KEY", ""),

		DatabaseURL: getRequiredEnv("DATABASE_URL"),

		RedisURL:      getEnv("REDIS_URL", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),

		JWTSecret:          getRequiredEnv("JWT_SECRET"),
		JWTExpirationHours: getEnvInt("JWT_EXPIRATION_HOURS", 24),

		CORSOrigins: []string{
			"http://localhost:3000",
			"http://localhost:3001",
			"https://tigerwallet.io",
			"https://www.tigerwallet.io",
		},

		WalletServiceURL:     getEnv("WALLET_SERVICE_URL", "http://localhost:8081"),
		SwapServiceURL:       getEnv("SWAP_SERVICE_URL", "http://localhost:8082"),
		PerpetualServiceURL:  getEnv("PERPETUAL_SERVICE_URL", "http://localhost:8083"),
		CopyTradingURL:        getEnv("COPY_TRADING_SERVICE_URL", "http://localhost:8084"),
		BlockchainServiceURL: getEnv("BLOCKCHAIN_SERVICE_URL", "http://localhost:8085"),
		TokenServiceURL:      getEnv("TOKEN_SERVICE_URL", "http://localhost:8086"),
		PortfolioServiceURL:  getEnv("PORTFOLIO_SERVICE_URL", "http://localhost:8087"),
		NotificationURL:      getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8088"),

		MaxRequestSize: getEnvInt64("MAX_REQUEST_SIZE", 10485760), // 10MB
		RateLimit:      getEnvInt("RATE_LIMIT", 100),
		BurstLimit:     getEnvInt("BURST_LIMIT", 200),

		DefaultChainID: uint64(getEnvInt("DEFAULT_CHAIN_ID", 1)),
		Confirmations:  getEnvInt("CONFIRMATIONS", 12),

		SuperAdminAddresses: []string{
			getEnv("SUPER_ADMIN_ADDRESS", "0x0000000000000000000000000000000000000001"),
		},
		AdminAPIKey: getRequiredEnv("ADMIN_API_KEY"),
	}
}

// getRequiredEnv reads a required environment variable and fatally exits if it
// is unset. Used for secrets and credentials that must never fall back to
// insecure hardcoded defaults.
func getRequiredEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("%s environment variable must be set", key)
	}
	return value
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
