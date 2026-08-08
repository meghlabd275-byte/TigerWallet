package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the admin backend
type Config struct {
	// Server
	ServerPort string
	ServerHost string

	// Database
	DBHost           string
	DBPort           int
	DBUser           string
	DBPassword       string
	DBName           string
	DatabaseURL      string
	DatabaseMaxConns int
	DatabaseMinConns int
	DBMaxOpenConns   int
	DBMaxIdleConns   int
	DBMaxLifetime    time.Duration

	// Redis
	RedisHost     string
	RedisPort     int
	RedisPassword string
	RedisDB       int
	RedisPoolSize int

	// JWT
	JWTSecret          string
	JWTExpirationHours int
	RefreshTokenExpiry int

	// Security
	EncryptionKey    string
	PasswordPepper   string
	MaxLoginAttempts int
	LockoutDuration  time.Duration

	// Rate Limiting
	RateLimitRequests int
	RateLimitWindow   time.Duration

	// Logging
	LogLevel  string
	LogOutput string

	// External Services
	ExternalAPITimeout time.Duration

	// Admin Settings
	DefaultAdminEmail    string
	DefaultAdminPassword string

	// System Settings
	SiteName               string
	MaintenanceMode        bool
	RegistrationEnabled    bool
	PasswordMinLength      int
	PasswordRequireNumber  bool
	PasswordRequireSpecial bool
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		// Server
		ServerPort: getEnv("ADMIN_PORT", "9093"),
		ServerHost: getEnv("ADMIN_HOST", "0.0.0.0"),

		// Database - PostgreSQL
		DBHost:           getEnv("DB_HOST", "localhost"),
		DBPort:           getEnvAsInt("DB_PORT", 5432),
		DBUser:           getEnv("DB_USER", "tigerwallet"),
		DBPassword:       getEnv("DB_PASSWORD", ""),
		DBName:           getEnv("DB_NAME", "tigerwallet_admin"),
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://tigerwallet@localhost:5432/tigerwallet_admin?sslmode=disable"),
		DatabaseMaxConns: getEnvAsInt("DB_MAX_CONNS", 25),
		DatabaseMinConns: getEnvAsInt("DB_MIN_CONNS", 5),
		DBMaxOpenConns:   getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:   getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
		DBMaxLifetime:    time.Duration(getEnvAsInt("DB_MAX_LIFETIME_MINUTES", 5)) * time.Minute,

		// Redis
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnvAsInt("REDIS_PORT", 6379),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvAsInt("REDIS_DB", 0),
		RedisPoolSize: getEnvAsInt("REDIS_POOL_SIZE", 10),

		// JWT
		JWTSecret:          getEnv("JWT_SECRET", ""),
		JWTExpirationHours: getEnvAsInt("JWT_EXPIRATION_HOURS", 24),
		RefreshTokenExpiry: getEnvAsInt("REFRESH_TOKEN_EXPIRY_DAYS", 30),

		// Security
		EncryptionKey:    getEnv("ENCRYPTION_KEY", ""),
		PasswordPepper:   getEnv("PASSWORD_PEPPER", ""),
		MaxLoginAttempts: getEnvAsInt("MAX_LOGIN_ATTEMPTS", 5),
		LockoutDuration:  time.Duration(getEnvAsInt("LOCKOUT_DURATION_MINUTES", 15)) * time.Minute,

		// Rate Limiting
		RateLimitRequests: getEnvAsInt("RATE_LIMIT_REQUESTS", 100),
		RateLimitWindow:   time.Duration(getEnvAsInt("RATE_LIMIT_WINDOW_SECONDS", 60)) * time.Second,

		// Logging
		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogOutput: getEnv("LOG_OUTPUT", "stdout"),

		// External Services
		ExternalAPITimeout: time.Duration(getEnvAsInt("EXTERNAL_API_TIMEOUT_SECONDS", 30)) * time.Second,

		// Admin Settings
		DefaultAdminEmail:    getEnv("DEFAULT_ADMIN_EMAIL", "admin@tigerwallet.com"),
		DefaultAdminPassword: getRequiredEnv("DEFAULT_ADMIN_PASSWORD"),
	}
}

// getRequiredEnv reads an environment variable that must be set, or fatally exits.
func getRequiredEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("%s environment variable must be set", key)
	}
	return value
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	intVal, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intVal
}
