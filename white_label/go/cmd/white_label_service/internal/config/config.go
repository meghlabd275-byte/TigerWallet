package config

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type Config struct {
	// Server
	Port string
	
	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBMaxConns int32
	
	// Redis
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int
	
	// Security
	JWTSecret      string
	SessionExpiry  time.Duration
	MaxLoginAttempts int
	LockoutDuration time.Duration
	
	// Super Admin
	SuperAdminEmail string
}

var (
	Logger  zerolog.Logger
	DBPool  *pgxpool.Pool
	RedisClient *redis.Client
)

func Load() *Config {
	return &Config{
		Port:           getEnv("PORT", "8090"),
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnv("DB_PORT", "5432"),
		DBUser:         getEnv("DB_USER", "tigerwallet"),
		DBPassword:     getEnv("DB_PASSWORD", "tigerwallet"),
		DBName:         getEnv("DB_NAME", "tigerwallet_white_label"),
		DBMaxConns:     50,
		RedisHost:      getEnv("REDIS_HOST", "localhost"),
		RedisPort:      getEnv("REDIS_PORT", "6379"),
		RedisPassword:  getEnv("REDIS_PASSWORD", ""),
		RedisDB:        0,
		JWTSecret:      getEnv("JWT_SECRET", "tigerwallet-super-secret-key-change-in-production"),
		SessionExpiry:  24 * time.Hour,
		MaxLoginAttempts: 3,
		LockoutDuration: 15 * time.Minute,
		SuperAdminEmail: "superadmin@tigerwallet.com",
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func InitDatabase(cfg *Config) error {
	Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	
	// First connect without database to create it if needed
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/?sslmode=disable",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort)
	
	tempPool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		Logger.Warn().Err(err).Msg("Could not connect to default database, will use SQLite fallback")
		return err
	}
	
	// Create database if not exists
	_, _ = tempPool.Exec(context.Background(), fmt.Sprintf(
		"CREATE DATABASE %s", cfg.DBName))
	tempPool.Close()
	
	// Now connect to the actual database
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&pool_max_conns=%d",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBMaxConns)
	
	DBPool, err = pgxpool.New(context.Background(), dsn)
	if err != nil {
		Logger.Error().Err(err).Msg("Failed to create connection pool")
		return err
	}
	
	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := DBPool.Ping(ctx); err != nil {
		Logger.Error().Err(err).Msg("Failed to ping database")
		return err
	}
	
	Logger.Info().Msg("Connected to PostgreSQL database")
	return nil
}

func InitRedis(cfg *Config) error {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := RedisClient.Ping(ctx).Err(); err != nil {
		Logger.Warn().Err(err).Msg("Redis connection failed, using in-memory fallback")
		// Continue without Redis - use in-memory fallback
		return err
	}
	
	Logger.Info().Msg("Connected to Redis")
	return nil
}

func Close() {
	if DBPool != nil {
		DBPool.Close()
	}
	if RedisClient != nil {
		RedisClient.Close()
	}
}
