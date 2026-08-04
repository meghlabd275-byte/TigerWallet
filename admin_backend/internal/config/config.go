package config

import (
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
	DBHost        string
	DBPort        int
	DBUser        string
	DBPassword    string
	DBName        string
	DBMaxOpenConns int
	DBMaxIdleConns int
	DBMaxLifetime  time.Duration
	
	// Redis
	RedisHost     string
	RedisPort     int
	RedisPassword string
	RedisDB       int
	RedisPoolSize int
	
	// JWT
	JWTSecret          string
	JWTExpirationHours int
	RefreshTokenExpiry  int
	
	// Security
	EncryptionKey      string
	PasswordPepper     string
	MaxLoginAttempts   int
	LockoutDuration    time.Duration
	
	// Rate Limiting
	RateLimitRequests  int
	RateLimitWindow    time.Duration
	
	// Logging
	LogLevel       string
	LogOutput      string
	
	// External Services
	ExternalAPITimeout time.Duration
	
	// Admin Settings
	DefaultAdminEmail    string
	DefaultAdminPassword string
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		// Server
		ServerPort: getEnv("ADMIN_PORT", "9093"),
		ServerHost: getEnv("ADMIN_HOST", "0.0.0.0"),
		
		// Database - PostgreSQL
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnvAsInt("DB_PORT", 5432),
		DBUser:        getEnv("DB_USER", "tigerwallet"),
		DBPassword:    getEnv("DB_PASSWORD", ""),
		DBName:        getEnv("DB_NAME", "tigerwallet_admin"),
		DBMaxOpenConns: getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns: getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
		DBMaxLifetime:  time.Duration(getEnvAsInt("DB_MAX_LIFETIME_MINUTES", 5)) * time.Minute,
		
		// Redis
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnvAsInt("REDIS_PORT", 6379),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvAsInt("REDIS_DB", 0),
		RedisPoolSize: getEnvAsInt("REDIS_POOL_SIZE", 10),
		
		// JWT
		JWTSecret:          getEnv("JWT_SECRET", ""),
		JWTExpirationHours:  getEnvAsInt("JWT_EXPIRATION_HOURS", 24),
		RefreshTokenExpiry: getEnvAsInt("REFRESH_TOKEN_EXPIRY_DAYS", 30),
		
		// Security
		EncryptionKey:   getEnv("ENCRYPTION_KEY", ""),
		PasswordPepper: getEnv("PASSWORD_PEPPER", ""),
		MaxLoginAttempts: getEnvAsInt("MAX_LOGIN_ATTEMPTS", 5),
		LockoutDuration: time.Duration(getEnvAsInt("LOCKOUT_DURATION_MINUTES", 15)) * time.Minute,
		
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
		DefaultAdminPassword: getEnv("DEFAULT_ADMIN_PASSWORD", "ChangeMe123!"),
	}
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
