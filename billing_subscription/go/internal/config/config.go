package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Stripe   StripeConfig
	Security SecurityConfig
}

type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
	MaxConns int32
	MinConns int32
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB      int
	PoolSize int
}

type JWTConfig struct {
	Secret          string
	Expiration      time.Duration
	RefreshDuration time.Duration
}

type StripeConfig struct {
	SecretKey      string
	WebhookSecret  string
	PublishableKey string
}

type SecurityConfig struct {
	BCryptCost         int
	MaxLoginAttempts   int
	LockoutDuration    time.Duration
	PasswordMinLength  int
	RequireUppercase   bool
	RequireLowercase   bool
	RequireNumbers    bool
	RequireSpecial    bool
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         getEnv("BILLING_PORT", "9001"),
			ReadTimeout:  getDurationEnv("READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getDurationEnv("WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  getDurationEnv("IDLE_TIMEOUT", 120*time.Second),
		},
		Database: DatabaseConfig{
			Host:     getEnv("BILLING_DB_HOST", "localhost"),
			Port:     getIntEnv("BILLING_DB_PORT", 5432),
			User:     getEnv("BILLING_DB_USER", "tigerwallet"),
			Password: getEnv("BILLING_DB_PASSWORD", "password"),
			DBName:   getEnv("BILLING_DB_NAME", "tigerwallet_billing"),
			SSLMode:  getEnv("BILLING_DB_SSLMODE", "require"),
			MaxConns: int32(getIntEnv("BILLING_DB_MAX_CONNS", 25)),
			MinConns: int32(getIntEnv("BILLING_DB_MIN_CONNS", 5)),
		},
		Redis: RedisConfig{
			Host:     getEnv("BILLING_REDIS_HOST", "localhost"),
			Port:     getIntEnv("BILLING_REDIS_PORT", 6379),
			Password: getEnv("BILLING_REDIS_PASSWORD", ""),
			DB:      getIntEnv("BILLING_REDIS_DB", 0),
			PoolSize: getIntEnv("BILLING_REDIS_POOL_SIZE", 50),
		},
		JWT: JWTConfig{
			Secret:          getEnv("BILLING_JWT_SECRET", ""),
			Expiration:      getDurationEnv("JWT_EXPIRATION", 24*time.Hour),
			RefreshDuration: getDurationEnv("JWT_REFRESH_DURATION", 7*24*time.Hour),
		},
		Stripe: StripeConfig{
			SecretKey:      getEnv("STRIPE_SECRET_KEY", ""),
			WebhookSecret:  getEnv("STRIPE_WEBHOOK_SECRET", ""),
			PublishableKey: getEnv("STRIPE_PUBLISHABLE_KEY", ""),
		},
		Security: SecurityConfig{
			BCryptCost:        getIntEnv("BCRYPT_COST", 12),
			MaxLoginAttempts:  getIntEnv("MAX_LOGIN_ATTEMPTS", 5),
			LockoutDuration:   getDurationEnv("LOCKOUT_DURATION", 15*time.Minute),
			PasswordMinLength: getIntEnv("PASSWORD_MIN_LENGTH", 12),
			RequireUppercase:  getBoolEnv("REQUIRE_UPPERCASE", true),
			RequireLowercase:  getBoolEnv("REQUIRE_LOWERCASE", true),
			RequireNumbers:    getBoolEnv("REQUIRE_NUMBERS", true),
			RequireSpecial:    getBoolEnv("REQUIRE_SPECIAL", true),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1"
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
