// Config - Application configuration for Admin Backend
package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServerPort         string
	ServerReadTimeout  time.Duration
	ServerWriteTimeout time.Duration
	ServerIdleTimeout  time.Duration
	DatabaseURL        string
	DatabaseMaxConns   int32
	DatabaseMinConns   int32
	RedisAddr         string
	RedisPassword     string
	RedisDB           int
	JWTSecret         string
	JWTExpiry          time.Duration
	JWTRefreshExpiry   time.Duration
	BCryptCost         int
	EnableIPWhitelist  bool
	AllowedIPs         []string
	RateLimitRequests  int
	RateLimitWindow    time.Duration
	TwoFactorIssuer    string
	BackupEnabled      bool
	BackupPath         string
	BackupInterval     time.Duration
	BackupRetentionDays int
	PagerDutyAPIKey    string
	DatadogAPIKey      string
	DatadogAppKey      string
	DatadogSite        string
	CloudflareAPIKey   string
	CloudflareEmail    string
	CloudflareZoneID   string
	LogLevel           string
}

func Load() *Config {
	return &Config{
		ServerPort:         getEnv("SERVER_PORT", "8082"),
		ServerReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 30*time.Second),
		ServerWriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 30*time.Second),
		ServerIdleTimeout:  getDurationEnv("SERVER_IDLE_TIMEOUT", 120*time.Second),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://tigerwallet:password@localhost:5432/admin?sslmode=disable"),
		DatabaseMaxConns:   int32(getIntEnv("DATABASE_MAX_CONNS", 25)),
		DatabaseMinConns:   int32(getIntEnv("DATABASE_MIN_CONNS", 5)),
		RedisAddr:          getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:     getEnv("REDIS_PASSWORD", ""),
		RedisDB:           getIntEnv("REDIS_DB", 0),
		JWTSecret:          getEnv("JWT_SECRET", "tigerwallet-admin-secret-key"),
		JWTExpiry:          getDurationEnv("JWT_EXPIRY", 24*time.Hour),
		JWTRefreshExpiry:   getDurationEnv("JWT_REFRESH_EXPIRY", 7*24*time.Hour),
		BCryptCost:         getIntEnv("BCRYPT_COST", 14),
		EnableIPWhitelist:  getBoolEnv("ENABLE_IP_WHITELIST", false),
		AllowedIPs:         getEnvSlice("ALLOWED_IPS"),
		RateLimitRequests:  getIntEnv("RATE_LIMIT_REQUESTS", 100),
		RateLimitWindow:    getDurationEnv("RATE_LIMIT_WINDOW", time.Minute),
		TwoFactorIssuer:    getEnv("TWO_FACTOR_ISSUER", "TigerWallet"),
		BackupEnabled:       getBoolEnv("BACKUP_ENABLED", true),
		BackupPath:          getEnv("BACKUP_PATH", "/var/backups/tigerwallet"),
		BackupInterval:      getDurationEnv("BACKUP_INTERVAL", 24*time.Hour),
		BackupRetentionDays: getIntEnv("BACKUP_RETENTION_DAYS", 30),
		PagerDutyAPIKey:    getEnv("PAGERDUTY_API_KEY", ""),
		DatadogAPIKey:      getEnv("DATADOG_API_KEY", ""),
		DatadogAppKey:      getEnv("DATADOG_APP_KEY", ""),
		DatadogSite:        getEnv("DATADOG_SITE", "datadoghq.com"),
		CloudflareAPIKey:   getEnv("CLOUDFLARE_API_KEY", ""),
		CloudflareEmail:    getEnv("CLOUDFLARE_EMAIL", ""),
		CloudflareZoneID:   getEnv("CLOUDFLARE_ZONE_ID", ""),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
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

func getEnvSlice(key string) []string {
	return []string{}
}
