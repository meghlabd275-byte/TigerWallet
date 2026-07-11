package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port       string
	Debug      bool
	Host       string
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBMaxOpenConns int
	DBMaxIdleConns int
	DBMaxLifetime  time.Duration
	RedisHost     string
	RedisPort     int
	RedisPassword string
	RedisDB       int
	JWTSecret          string
	JWTExpiration      time.Duration
	JWTRefreshExpires  time.Duration
	EthereumRPC    string
	BSCRPC         string
	PolygonRPC     string
	ArbitrumRPC    string
	OptimismRPC    string
	SolanaRPC      string
	TRONRPC        string
	AptosRPC       string
	InfuraAPIKey     string
	AlchemyAPIKey    string
	CoinGeckoAPI     string
	PriceAPI         string
	MaxRequestSize    int64
	RateLimitRequests int
	RateLimitWindow   time.Duration
	AdminAPIKey string
}

func Load() *Config {
	return &Config{
		Port:       getEnv("PORT", "8080"),
		Debug:      getEnvAsBool("DEBUG", false),
		Host:       getEnv("HOST", "0.0.0.0"),
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnvAsInt("DB_PORT", 5432),
		DBUser:         getEnv("DB_USER", "tigerwallet"),
		DBPassword:     getEnv("DB_PASSWORD", "password"),
		DBName:         getEnv("DB_NAME", "tigerwallet"),
		DBMaxOpenConns: getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns: getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
		DBMaxLifetime:  getEnvAsDuration("DB_MAX_LIFETIME", 5*time.Minute),
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnvAsInt("REDIS_PORT", 6379),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvAsInt("REDIS_DB", 0),
		JWTSecret:          getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		JWTExpiration:      getEnvAsDuration("JWT_EXPIRATION", 24*time.Hour),
		JWTRefreshExpires:  getEnvAsDuration("JWT_REFRESH_EXPIRES", 7*24*time.Hour),
		EthereumRPC:  getEnv("ETHEREUM_RPC", "https://eth.llamarpc.com"),
		BSCRPC:        getEnv("BSC_RPC", "https://bsc-dataseed.binance.org"),
		PolygonRPC:    getEnv("POLYGON_RPC", "https://polygon-rpc.com"),
		ArbitrumRPC:   getEnv("ARBITRUM_RPC", "https://arb1.arbitrum.io/rpc"),
		OptimismRPC:   getEnv("OPTIMISM_RPC", "https://mainnet.optimism.io"),
		SolanaRPC:     getEnv("SOLANA_RPC", "https://api.mainnet-beta.solana.com"),
		TRONRPC:       getEnv("TRON_RPC", "https://api.trongrid.io"),
		AptosRPC:      getEnv("APTOS_RPC", "https://fullnode.mainnet.aptoslabs.com"),
		InfuraAPIKey:    getEnv("INFURA_API_KEY", ""),
		AlchemyAPIKey:   getEnv("ALCHEMY_API_KEY", ""),
		CoinGeckoAPI:    getEnv("COINGECKO_API", "https://api.coingecko.com/api/v3"),
		PriceAPI:        getEnv("PRICE_API", "https://api.tigerwallet.io/v1/prices"),
		MaxRequestSize:    getEnvAsInt64("MAX_REQUEST_SIZE", 10485760),
		RateLimitRequests: getEnvAsInt("RATE_LIMIT_REQUESTS", 100),
		RateLimitWindow:   getEnvAsDuration("RATE_LIMIT_WINDOW", time.Minute),
		AdminAPIKey: getEnv("ADMIN_API_KEY", "admin-secret-key"),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists { return value }
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil { return value }
	return defaultValue
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseInt(valueStr, 10, 64); err == nil { return value }
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseBool(valueStr); err == nil { return value }
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := getEnv(key, "")
	if value, err := time.ParseDuration(valueStr); err == nil { return value }
	return defaultValue
}
