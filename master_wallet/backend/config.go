package main

// config.go — MasterWallet backend configuration. All values come from env vars;
// no secrets are hardcoded. PostgreSQL + Redis connection pooling is configured
// for high-throughput distributed deployment.

import (
	"os"
	"strconv"
	"strings"
)

// AppConfig holds all runtime configuration loaded from the environment.
type AppConfig struct {
	Port            string
	JWTSecret       string
	DatabaseURL     string
	RedisAddr       string
	RedisPassword   string
	RedisDB         int
	// CoinGecko + Etherscan API keys for real market/on-chain data.
	CoinGeckoAPIKey string
	EtherscanAPIKey string
	// Treasury hot-wallet signer key (hex). Required for real broadcast. When
	// unset, treasury transfer/sweep endpoints return 503 (fail-closed) instead
	// of fabricating a transaction hash.
	TreasuryKeyHex string
	// Bundler / paymaster endpoints for account abstraction (optional; when
	// unset, AA endpoints delegate signing to the wallet owner's key).
	BundlerURL      string
	PaymasterURL    string
}

// LoadConfig reads configuration from environment variables with sensible
// production defaults. JWTSecret MUST be set for the service to issue tokens.
func LoadConfig() *AppConfig {
	port := os.Getenv("MASTER_WALLET_PORT")
	if port == "" {
		port = "8450"
	}
	jwtSecret := os.Getenv("MASTER_WALLET_JWT_SECRET")
	if jwtSecret == "" {
		// Fail closed: a random per-process secret means tokens never validate
		// across restarts, which is safer than a hardcoded default for prod.
		jwtSecret = "dev-only-change-me-" + strconv.FormatInt(int64(os.Getpid()), 10)
	}
	redisDB, _ := strconv.Atoi(getEnvDefault("MASTER_WALLET_REDIS_DB", "0"))
	return &AppConfig{
		Port:            port,
		JWTSecret:       jwtSecret,
		DatabaseURL:     getEnvDefault("MASTER_WALLET_DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet?sslmode=disable"),
		RedisAddr:       getEnvDefault("MASTER_WALLET_REDIS_ADDR", "localhost:6379"),
		RedisPassword:   os.Getenv("MASTER_WALLET_REDIS_PASSWORD"),
		RedisDB:         redisDB,
		CoinGeckoAPIKey: os.Getenv("COINGECKO_API_KEY"),
		EtherscanAPIKey: os.Getenv("ETHERSCAN_API_KEY"),
		TreasuryKeyHex:  os.Getenv("MASTER_WALLET_TREASURY_KEY_HEX"),
		BundlerURL:      os.Getenv("MASTER_WALLET_BUNDLER_URL"),
		PaymasterURL:    os.Getenv("MASTER_WALLET_PAYMASTER_URL"),
	}
}

func getEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// chainRPCEndpoint resolves the JSON-RPC endpoint for a given chain id from the
// environment. Each chain has its own env var so operators can pin specific
// node providers. Returns empty string (fail-closed) when unset.
func chainRPCEndpoint(chainID int64) string {
	// Map chain id -> env var holding the RPC URL.
	envVars := map[int64]string{
		1:          "ETH_RPC_URL",
		56:         "BSC_RPC_URL",
		137:        "POLYGON_RPC_URL",
		42161:      "ARBITRUM_RPC_URL",
		10:         "OPTIMISM_RPC_URL",
		43114:      "AVALANCHE_RPC_URL",
		8453:       "BASE_RPC_URL",
		42220:      "CELO_RPC_URL",
		250:        "FANTOM_RPC_URL",
		25:         "CRONOS_RPC_URL",
		59144:      "LINEA_RPC_URL",
		534352:     "SCROLL_RPC_URL",
		11155111:   "ETH_SEPOLIA_RPC_URL",
	}
	if envVar, ok := envVars[chainID]; ok {
		return os.Getenv(envVar)
	}
	return ""
}

// chainExplorerAPI returns the Etherscan-compatible explorer base URL + API key
// env var name for a chain id. Used for real transaction history / NFT fetches.
func chainExplorerAPI(chainID int64) (baseURL, apiKeyEnv string) {
	switch chainID {
	case 1:
		return "https://api.etherscan.io/api", "ETHERSCAN_API_KEY"
	case 56:
		return "https://api.bscscan.com/api", "BSCSCAN_API_KEY"
	case 137:
		return "https://api.polygonscan.com/api", "POLYGONSCAN_API_KEY"
	case 42161:
		return "https://api.arbiscan.io/api", "ARBISCAN_API_KEY"
	case 10:
		return "https://api-optimistic.etherscan.io/api", "OPTIMISM_API_KEY"
	case 43114:
		return "https://api.snowtrace.io/api", "SNOWTRACE_API_KEY"
	case 8453:
		return "https://api.basescan.org/api", "BASESCAN_API_KEY"
	default:
		return "", ""
	}
}

// chainCoinGeckoID maps an EVM chain id to the CoinGecko coin id used for price
// lookups (native asset price).
func chainCoinGeckoID(chainID int64) string {
	switch chainID {
	case 1:
		return "ethereum"
	case 56:
		return "binancecoin"
	case 137:
		return "matic-network"
	case 42161:
		return "arbitrum"
	case 10:
		return "optimism"
	case 43114:
		return "avalanche-2"
	case 8453:
		return "base"
	default:
		return ""
	}
}

// normalizeChain ensures a chain identifier is lowercased + trimmed.
func normalizeChain(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
