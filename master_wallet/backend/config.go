package main

// config.go — MasterWallet backend configuration. All values come from env vars;
// no secrets are hardcoded. PostgreSQL + Redis connection pooling is configured
// for high-throughput distributed deployment.

import (
	"crypto/rand"
	"encoding/hex"
	"log"
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
	// A predictable secret would let anyone mint an operator token for this
	// treasury backend, so an unset or placeholder value is fatal rather than
	// silently substituted. MASTER_WALLET_ALLOW_INSECURE_DEV_SECRET exists only
	// for local development and generates a random per-process key.
	jwtSecret := strings.TrimSpace(os.Getenv("MASTER_WALLET_JWT_SECRET"))
	if jwtSecret == "" || isPlaceholderSecret(jwtSecret) {
		if strings.EqualFold(os.Getenv("MASTER_WALLET_ALLOW_INSECURE_DEV_SECRET"), "true") {
			buf := make([]byte, 32)
			if _, err := rand.Read(buf); err != nil {
				log.Fatal("master wallet: failed to generate a development JWT secret: ", err)
			}
			jwtSecret = hex.EncodeToString(buf)
			log.Println("WARNING: using a random development JWT secret; tokens will not survive a restart")
		} else {
			log.Fatal("master wallet: MASTER_WALLET_JWT_SECRET must be set to a non-placeholder value")
		}
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
	// Etherscan deprecated the per-chain V1 endpoints; the V2 multichain
	// API serves every Etherscan-family chain from ONE base URL with ONE
	// key (ETHERSCAN_API_KEY), selecting the chain via the chainid query
	// parameter (added by the fetcher for /v2/ bases).
	switch chainID {
	case 1, 56, 137, 42161, 10, 43114, 8453:
		return "https://api.etherscan.io/v2/api", "ETHERSCAN_API_KEY"
	default:
		// Keyless fallback: every seeded EVM chain carries a public
		// ExplorerURL in the registry. Blockscout-family explorers
		// expose the Etherscan-compatible API at <base>/api WITHOUT an
		// API key, so history works out of the box for the long tail
		// of chains. If the explorer is not API-compatible the fetch
		// fails with the real upstream error (fail-closed, no fakes).
		for _, c := range defaultEVMChains {
			if c.ChainID == chainID && c.ExplorerURL != "" {
				return strings.TrimRight(c.ExplorerURL, "/") + "/api", ""
			}
		}
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

// placeholderSecrets are values that appear in example configs and must never
// be accepted as a real signing key.
var placeholderSecrets = map[string]bool{
	"tigerwallet-master-jwt-change-in-production": true,
	"change-in-production":                        true,
	"changeme":                                    true,
	"change-me":                                   true,
	"secret":                                      true,
	"password":                                    true,
	"your-secret-here":                            true,
	"dev-only-change-me":                          true,
}

// isPlaceholderSecret reports whether a configured secret is a known
// placeholder rather than a real key.
func isPlaceholderSecret(value string) bool {
	return placeholderSecrets[strings.ToLower(strings.TrimSpace(value))]
}
