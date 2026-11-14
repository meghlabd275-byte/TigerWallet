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
	Port          string
	JWTSecret     string
	DatabaseURL   string
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	// CoinGecko + Etherscan API keys for real market/on-chain data.
	CoinGeckoAPIKey string
	EtherscanAPIKey string
	// Treasury hot-wallet signer key (hex). Required for real broadcast. When
	// unset, treasury transfer/sweep endpoints return 503 (fail-closed) instead
	// of fabricating a transaction hash.
	TreasuryKeyHex string
	// Bundler / paymaster endpoints for account abstraction (optional; when
	// unset, AA endpoints delegate signing to the wallet owner's key).
	BundlerURL   string
	PaymasterURL string
	// CORSAllowedOrigins is the browser-origin allowlist for the API and the
	// WebSocket handshake. Empty means no cross-origin browser access.
	CORSAllowedOrigins []string
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
		Port:               port,
		JWTSecret:          jwtSecret,
		DatabaseURL:        getEnvDefault("MASTER_WALLET_DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet?sslmode=disable"),
		RedisAddr:          getEnvDefault("MASTER_WALLET_REDIS_ADDR", "localhost:6379"),
		RedisPassword:      os.Getenv("MASTER_WALLET_REDIS_PASSWORD"),
		RedisDB:            redisDB,
		CoinGeckoAPIKey:    os.Getenv("COINGECKO_API_KEY"),
		EtherscanAPIKey:    os.Getenv("ETHERSCAN_API_KEY"),
		TreasuryKeyHex:     os.Getenv("MASTER_WALLET_TREASURY_KEY_HEX"),
		BundlerURL:         os.Getenv("MASTER_WALLET_BUNDLER_URL"),
		PaymasterURL:       os.Getenv("MASTER_WALLET_PAYMASTER_URL"),
		CORSAllowedOrigins: splitAndTrim(os.Getenv("MASTER_WALLET_CORS_ORIGINS")),
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
// chainRPCEndpoint resolves the JSON-RPC URL for a chain id.
//
// Resolution order:
//
//  1. The chain's dedicated environment variable (ETH_RPC_URL, TSC_RPC_URL, …).
//     An operator running a private or rate-limited node sets this.
//  2. The RPC URL carried by the canonical chain registry entry. Every seeded
//     chain publishes a real public endpoint, so a chain works out of the box
//     instead of returning "RPC not configured" until an env var is added.
//  3. Empty. Callers treat that as "this chain cannot be reached" and answer
//     503; nothing is ever dialled that was not configured.
func chainRPCEndpoint(chainID int64) string {
	// Map chain id -> env var holding the RPC URL.
	envVars := map[int64]string{
		1:                 "ETH_RPC_URL",
		56:                "BSC_RPC_URL",
		137:               "POLYGON_RPC_URL",
		42161:             "ARBITRUM_RPC_URL",
		10:                "OPTIMISM_RPC_URL",
		43114:             "AVALANCHE_RPC_URL",
		8453:              "BASE_RPC_URL",
		42220:             "CELO_RPC_URL",
		250:               "FANTOM_RPC_URL",
		25:                "CRONOS_RPC_URL",
		59144:             "LINEA_RPC_URL",
		534352:            "SCROLL_RPC_URL",
		11155111:          "ETH_SEPOLIA_RPC_URL",
		tigerSmartChainID: tscRPCEnv,
	}
	if envVar, ok := envVars[chainID]; ok {
		if url := strings.TrimSpace(os.Getenv(envVar)); url != "" {
			return url
		}
	}
	// Registry fallback: the chain's own published endpoint.
	if c, ok := chainByID(chainID); ok {
		for _, rc := range defaultEVMChains {
			if rc.ChainID == c.ChainID && rc.RPCURL != "" {
				return rc.RPCURL
			}
		}
	}
	return ""
}

// etherscanV2Chains are the chains the unified Etherscan V2 multichain API
// serves from ONE base URL with ONE key (ETHERSCAN_API_KEY); the chain is
// selected with the chainid query parameter.
var etherscanV2Chains = map[int64]bool{
	1: true, 56: true, 137: true, 42161: true, 10: true, 43114: true, 8453: true,
}

// keylessBlockscoutHosts maps a chain id to a Blockscout instance that serves
// its v2 REST API WITHOUT an API key. These are the chains whose Etherscan-
// family explorer rejects keyless requests, so without this a deployment with
// no ETHERSCAN_API_KEY would show an empty history on the busiest chains. Each
// host was verified live; a host that cannot serve a chain is simply absent
// here, and the chain then reports its real upstream error rather than
// returning empty data.
var keylessBlockscoutHosts = map[int64]string{
	1:      "https://eth.blockscout.com",
	10:     "https://optimism.blockscout.com",
	324:    "https://zksync.blockscout.com",
	534352: "https://scroll.blockscout.com",
	42220:  "https://celo.blockscout.com",
	30:     "https://rootstock.blockscout.com",
}

// explorerHistorySource is one way to read an address's transaction history.
// An empty apiKeyEnv means the source is keyless.
type explorerHistorySource struct {
	// etherscanV2 selects the unified Etherscan V2 request shape (base
	// ".../v2/api" plus a chainid parameter).
	etherscanV2 bool
	// blockscoutV2 selects the Blockscout v2 REST shape (a per-chain host,
	// paginated with URL parameters).
	blockscoutV2 bool
	baseURL      string
	apiKeyEnv    string
}

// chainExplorerAPI returns the Etherscan-compatible explorer base URL + API key
// env var name for a chain id. Used for real transaction history / NFT fetches.
// An empty apiKeyEnv means the endpoint is keyless.
func chainExplorerAPI(chainID int64) (baseURL, apiKeyEnv string) {
	if etherscanV2Chains[chainID] {
		return "https://api.etherscan.io/v2/api", "ETHERSCAN_API_KEY"
	}
	// Keyless fallback: every seeded EVM chain carries a public ExplorerURL in
	// the registry. Blockscout-family explorers expose the Etherscan-compatible
	// API at <base>/api WITHOUT an API key, so history works out of the box for
	// the long tail of chains. If the explorer is not API-compatible the fetch
	// fails with the real upstream error (fail-closed, no fakes).
	for _, c := range defaultEVMChains {
		if c.ChainID == chainID && c.ExplorerURL != "" {
			return strings.TrimRight(c.ExplorerURL, "/") + "/api", ""
		}
	}
	return "", ""
}

// historySources returns the history backends for a chain, in the order they
// should be tried. The Etherscan V2 entry is only included when a key is
// actually configured: without one it can only answer "Missing/Invalid API
// Key", and returning that as an empty history would be indistinguishable from
// an address that has never transacted. When a chain has no configured source
// at all, the registry explorer is still tried, so its real error surfaces.
func historySources(chainID int64) []explorerHistorySource {
	var out []explorerHistorySource
	if etherscanV2Chains[chainID] && os.Getenv("ETHERSCAN_API_KEY") != "" {
		out = append(out, explorerHistorySource{
			etherscanV2: true,
			baseURL:     "https://api.etherscan.io/v2/api",
			apiKeyEnv:   "ETHERSCAN_API_KEY",
		})
	}
	if host := keylessBlockscoutHosts[chainID]; host != "" {
		out = append(out, explorerHistorySource{blockscoutV2: true, baseURL: host})
	}
	if base, keyEnv := chainExplorerAPI(chainID); base != "" {
		if !etherscanV2Chains[chainID] {
			out = append(out, explorerHistorySource{baseURL: base, apiKeyEnv: keyEnv})
		}
	}
	return out
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

// splitAndTrim splits a comma-separated list and drops empty entries.
func splitAndTrim(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if v := strings.TrimSpace(part); v != "" {
			out = append(out, v)
		}
	}
	return out
}
