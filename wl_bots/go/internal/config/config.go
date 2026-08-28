// Package config holds the standalone WL-Bots backend configuration. This backend
// runs INDEPENDENTLY in the WL client's own cloud — own PG, own DB. It phones
// home to the license control plane on heartbeat and gates every request
// fail-closed via wlgate. It is a clone of the TigerWallet mm_bot_platform/
// bot_api bot management platform, but standalone (no TigerWallet cloud at
// request time).
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
	Product           string // product identifier ("bots")
	InstanceID        string // unique instance id (hostname/container id)
	BCryptCost        int
	JWTExpiry         time.Duration
	HeartbeatInterval time.Duration
	BotCoreURL        string // Rust bot_core execution plane (BOT_CORE_URL)
}

func Load() *Config {
	return &Config{
		Port:              getEnv("PORT", "8471"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/wl_bots?sslmode=disable"),
		JWTSecret:         getEnv("JWT_SECRET", ""),
		ControlPlaneURL:   getEnv("TWO_PARTY_GATE_URL", ""),
		ControlPlaneToken: getEnv("TWO_PARTY_GATE_TOKEN", ""),
		WLClientID:        getEnv("WL_CLIENT_ID", ""),
		LicenseKey:        getEnv("WL_LICENSE_KEY", ""),
		Product:           getEnv("WL_PRODUCT", "bots"),
		InstanceID:        getEnv("WL_INSTANCE_ID", "default"),
		BCryptCost:        getInt("BCRYPT_COST", 12),
		JWTExpiry:         getDuration("JWT_EXPIRY", 24*time.Hour),
		HeartbeatInterval: getDuration("HEARTBEAT_INTERVAL", 30*time.Second),
		BotCoreURL:        getEnv("BOT_CORE_URL", "http://localhost:8472"),
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
