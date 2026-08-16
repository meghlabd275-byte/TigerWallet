// Package config holds the standalone WL-UserWallet backend configuration.
// This backend runs INDEPENDENTLY in the WL client's own cloud/OS — it does
// NOT depend on TigerWallet cloud. It phones home to the license control
// plane (TWO_PARTY_GATE_URL) for license validation + heartbeat + SuperAdmin
// commands, and gates every request via the C++ WlGate.
package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port              string
	DatabaseURL       string
	JWTSecret         string
	ControlPlaneURL   string // license control plane (TWO_PARTY_GATE_URL)
	ControlPlaneToken string // WL client service token
	WLClientID        string // the WL client's UUID (tenant identity)
	LicenseKey        string // the product's license key (issued by SuperAdmin)
	Product           string // product identifier (e.g. "user_wallet")
	InstanceID        string // unique instance id (hostname/container id)
	BCryptCost        int
	JWTExpiry         time.Duration
	HeartbeatInterval time.Duration
	CoinGeckoAPIKey   string
	EtherscanAPIKey   string
}

func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "8443"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/wl_userwallet?sslmode=disable"),
		JWTSecret:          getEnv("JWT_SECRET", ""),
		ControlPlaneURL:    getEnv("TWO_PARTY_GATE_URL", ""),
		ControlPlaneToken:  getEnv("TWO_PARTY_GATE_TOKEN", ""),
		WLClientID:         getEnv("WL_CLIENT_ID", ""),
		LicenseKey:         getEnv("WL_LICENSE_KEY", ""),
		Product:            getEnv("WL_PRODUCT", "user_wallet"),
		InstanceID:         getEnv("WL_INSTANCE_ID", "default"),
		BCryptCost:         getInt("BCRYPT_COST", 12),
		JWTExpiry:          getDuration("JWT_EXPIRY", 24*time.Hour),
		HeartbeatInterval:  getDuration("HEARTBEAT_INTERVAL", 30*time.Second),
		CoinGeckoAPIKey:    getEnv("COINGECKO_API_KEY", ""),
		EtherscanAPIKey:    getEnv("ETHERSCAN_API_KEY", ""),
	}
}

func getEnv(k, d string) string {
	if v, ok := os.LookupEnv(k); ok {
		return v
	}
	return d
}
func getInt(k string, d int) int {
	if v, ok := os.LookupEnv(k); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return d
}
func getDuration(k string, d time.Duration) time.Duration {
	if v, ok := os.LookupEnv(k); ok {
		if dur, err := time.ParseDuration(v); err == nil {
			return dur
		}
	}
	return d
}
