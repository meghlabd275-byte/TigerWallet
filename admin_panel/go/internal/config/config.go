// Config - Application configuration for Admin Panel Backend
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	// Server
	ServerPort         string
	ServerReadTimeout  time.Duration
	ServerWriteTimeout time.Duration
	ServerIdleTimeout  time.Duration

	// Database
	DatabaseURL             string
	DatabaseMaxConns        int32
	DatabaseMinConns        int32
	DatabaseMaxConnLifetime time.Duration

	// Redis
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	RedisPoolSize int

	// JWT
	JWTSecret        string
	JWTExpiry        time.Duration
	JWTRefreshExpiry time.Duration

	// Security
	BCryptCost        int
	EnableIPWhitelist bool
	AllowedIPs        []string
	RateLimitRequests int
	RateLimitWindow   time.Duration

	// 2FA
	TwoFactorIssuer string
	TwoFactorDigits int
	TwoFactorPeriod time.Duration

	// Backup
	BackupEnabled       bool
	BackupPath          string
	BackupInterval      time.Duration
	BackupRetentionDays int

	// Integrations
	PagerDutyAPIKey    string
	DatadogAPIKey      string
	DatadogAppKey      string
	DatadogSite        string
	CloudflareAPIKey   string
	CloudflareAPIToken string
	CloudflareEmail    string
	CloudflareZoneID   string

	// Logging
	LogLevel string
}

func Load() *Config {
	return &Config{
		// Server
		ServerPort:         getEnv("SERVER_PORT", "8081"),
		ServerReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 30*time.Second),
		ServerWriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 30*time.Second),
		ServerIdleTimeout:  getDurationEnv("SERVER_IDLE_TIMEOUT", 120*time.Second),

		// Database
		DatabaseURL:             getEnv("DATABASE_URL", "postgres://tigerwallet:password@localhost:5432/admin_panel?sslmode=disable"),
		DatabaseMaxConns:        int32(getIntEnv("DATABASE_MAX_CONNS", 25)),
		DatabaseMinConns:        int32(getIntEnv("DATABASE_MIN_CONNS", 5)),
		DatabaseMaxConnLifetime: getDurationEnv("DATABASE_MAX_CONN_LIFETIME", time.Hour),

		// Redis
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getIntEnv("REDIS_DB", 0),
		RedisPoolSize: getIntEnv("REDIS_POOL_SIZE", 100),

		// JWT
		JWTSecret:        getEnv("JWT_SECRET", ""),
		JWTExpiry:        getDurationEnv("JWT_EXPIRY", 24*time.Hour),
		JWTRefreshExpiry: getDurationEnv("JWT_REFRESH_EXPIRY", 7*24*time.Hour),

		// Security
		BCryptCost:        getIntEnv("BCRYPT_COST", 14),
		EnableIPWhitelist: getBoolEnv("ENABLE_IP_WHITELIST", false),
		AllowedIPs:        getEnvSlice("ALLOWED_IPS", []string{}),
		RateLimitRequests: getIntEnv("RATE_LIMIT_REQUESTS", 100),
		RateLimitWindow:   getDurationEnv("RATE_LIMIT_WINDOW", time.Minute),

		// 2FA
		TwoFactorIssuer: getEnv("TWO_FACTOR_ISSUER", "TigerWallet"),
		TwoFactorDigits: 6,
		TwoFactorPeriod: 30 * time.Second,

		// Backup
		BackupEnabled:       getBoolEnv("BACKUP_ENABLED", true),
		BackupPath:          getEnv("BACKUP_PATH", "/var/backups/tigerwallet"),
		BackupInterval:      getDurationEnv("BACKUP_INTERVAL", 24*time.Hour),
		BackupRetentionDays: getIntEnv("BACKUP_RETENTION_DAYS", 30),

		// Integrations
		PagerDutyAPIKey:    getEnv("PAGERDUTY_API_KEY", ""),
		DatadogAPIKey:      getEnv("DATADOG_API_KEY", ""),
		DatadogAppKey:      getEnv("DATADOG_APP_KEY", ""),
		DatadogSite:        getEnv("DATADOG_SITE", "datadoghq.com"),
		CloudflareAPIKey:   getEnv("CLOUDFLARE_API_KEY", ""),
		CloudflareAPIToken: getEnv("CLOUDFLARE_API_TOKEN", ""),
		CloudflareEmail:    getEnv("CLOUDFLARE_EMAIL", ""),
		CloudflareZoneID:   getEnv("CLOUDFLARE_ZONE_ID", ""),

		// Logging
		LogLevel: getEnv("LOG_LEVEL", "info"),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		return value == "true" || value == "1"
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvSlice(key string, defaultValue []string) []string {
	if value, exists := os.LookupEnv(key); exists {
		// Simple comma-separated list
		if value != "" {
			parts := strings.Split(value, ",")
			result := make([]string, 0, len(parts))
			for _, part := range parts {
				if trimmed := strings.TrimSpace(part); trimmed != "" {
					result = append(result, trimmed)
				}
			}
			return result
		}
	}
	return defaultValue
}
