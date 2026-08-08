/**
 * Configuration Management
 * 
 * Loads and manages all configuration for the wallet services
 */

package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Environment string        `mapstructure:"environment"`
	Server      ServerConfig `mapstructure:"server"`
	Database    DatabaseConfig `mapstructure:"database"`
	Redis       RedisConfig `mapstructure:"redis"`
	JWT         JWTConfig `mapstructure:"jwt"`
	Security    SecurityConfig `mapstructure:"security"`
	Blockchain  BlockchainConfig `mapstructure:"blockchain"`
	External    ExternalConfig `mapstructure:"external"`
}

type ServerConfig struct {
	Port           int           `mapstructure:"port"`
	ReadTimeout    time.Duration `mapstructure:"read_timeout"`
	WriteTimeout   time.Duration `mapstructure:"write_timeout"`
	MaxHeaderBytes int           `mapstructure:"max_header_bytes"`
	MaxConnections int           `mapstructure:"max_connections"`
}

type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	Database     string `mapstructure:"database"`
	SSLMode      string `mapstructure:"ssl_mode"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxLifetime  time.Duration `mapstructure:"max_lifetime"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	Database int    `mapstructure:"database"`
	PoolSize int    `mapstructure:"pool_size"`
}

type JWTConfig struct {
	Secret           string        `mapstructure:"secret"`
	AccessExpiry    time.Duration `mapstructure:"access_expiry"`
	RefreshExpiry   time.Duration `mapstructure:"refresh_expiry"`
	Issuer          string        `mapstructure:"issuer"`
}

type SecurityConfig struct {
	PasswordMinLength    int  `mapstructure:"password_min_length"`
	PasswordRequireUpper bool `mapstructure:"password_require_upper"`
	PasswordRequireLower bool `mapstructure:"password_require_lower"`
	PasswordRequireDigit bool `mapstructure:"password_require_digit"`
	PasswordRequireSpecial bool `mapstructure:"password_require_special"`
	MaxLoginAttempts    int  `mapstructure:"max_login_attempts"`
	LockoutDuration     time.Duration `mapstructure:"lockout_duration"`
	SessionTimeout      time.Duration `mapstructure:"session_timeout"`
	Enable2FA           bool `mapstructure:"enable_2fa"`
	RateLimitRequests   int  `mapstructure:"rate_limit_requests"`
	RateLimitWindow     time.Duration `mapstructure:"rate_limit_window"`
}

type BlockchainConfig struct {
	RPCTimeout    time.Duration `mapstructure:"rpc_timeout"`
	MaxRetries   int           `mapstructure:"max_retries"`
	RetryDelay   time.Duration `mapstructure:"retry_delay"`
	Confirmations map[string]int `mapstructure:"confirmations"`
}

type ExternalConfig struct {
	EtherscanAPIKey    string `mapstructure:"etherscan_api_key"`
	BscScanAPIKey      string `mapstructure:"bscscan_api_key"`
	PolygonScanAPIKey  string `mapstructure:"polygonscan_api_key"`
	CoingeckoAPIKey    string `mapstructure:"coingecko_api_key"`
	InfuraProjectID    string `mapstructure:"infura_project_id"`
	InfuraProjectSecret string `mapstructure:"infura_project_secret"`
	AlchemyAPIKey      string `mapstructure:"alchemy_api_key"`
}

func Load() (*Config, error) {
	// Set default values
	viper.SetDefault("environment", "development")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.read_timeout", 30*time.Second)
	viper.SetDefault("server.write_timeout", 30*time.Second)
	viper.SetDefault("server.max_header_bytes", 1<<20) // 1MB

	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("database.max_open_conns", 100)
	viper.SetDefault("database.max_idle_conns", 10)
	viper.SetDefault("database.max_lifetime", 30*time.Minute)

	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.pool_size", 100)

	viper.SetDefault("jwt.secret", "")
	viper.SetDefault("jwt.access_expiry", 15*time.Minute)
	viper.SetDefault("jwt.refresh_expiry", 7*24*time.Hour)
	viper.SetDefault("jwt.issuer", "tigerwallet")

	viper.SetDefault("security.password_min_length", 8)
	viper.SetDefault("security.password_require_upper", true)
	viper.SetDefault("security.password_require_lower", true)
	viper.SetDefault("security.password_require_digit", true)
	viper.SetDefault("security.password_require_special", true)
	viper.SetDefault("security.max_login_attempts", 5)
	viper.SetDefault("security.lockout_duration", 15*time.Minute)
	viper.SetDefault("security.session_timeout", 24*time.Hour)
	viper.SetDefault("security.enable_2fa", true)
	viper.SetDefault("security.rate_limit_requests", 100)
	viper.SetDefault("security.rate_limit_window", 1*time.Minute)

	viper.SetDefault("blockchain.rpc_timeout", 30*time.Second)
	viper.SetDefault("blockchain.max_retries", 3)
	viper.SetDefault("blockchain.retry_delay", 1*time.Second)

	// Set default confirmations
	viper.SetDefault("blockchain.confirmations.bitcoin", 6)
	viper.SetDefault("blockchain.confirmations.ethereum", 12)
	viper.SetDefault("blockchain.confirmations.polygon", 15)
	viper.SetDefault("blockchain.confirmations.bsc", 15)

	// Try to read from config file
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/tigerwallet/")
	viper.AddConfigPath("$HOME/.tigerwallet/")

	// Environment variables take precedence
	viper.SetEnvPrefix("TW")
	viper.AutomaticEnv()

	// Read config
	if err := viper.ReadInConfig(); err != nil {
		// Config file not found, use defaults
		fmt.Printf("Config file not found, using defaults: %v\n", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func (c *DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
	)
}

func (c *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
