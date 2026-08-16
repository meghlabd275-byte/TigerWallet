// Package config holds the standalone WL-Card backend configuration. This
// backend runs INDEPENDENTLY in the WL client's own cloud — own PG, own DB.
// It phones home to the license control plane (TWO_PARTY_GATE_URL) on
// heartbeat and gates every request fail-closed via wlgate. It is a
// standalone clone of the TigerWallet CryptoCard service, but
// PostgreSQL-persisted (real cards + transactions) instead of in-memory.
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
	CardEncKey        string // AES-GCM at-rest encryption key for PAN/CVV (CARD_ENC_KEY)
	ControlPlaneURL   string // license control plane (TWO_PARTY_GATE_URL)
	ControlPlaneToken string // WL client service token
	WLClientID        string // the WL client's UUID (tenant identity)
	LicenseKey        string // product license key (issued by SuperAdmin)
	Product           string // product identifier ("card")
	InstanceID        string // unique instance id (hostname/container id)
	BCryptCost        int
	JWTExpiry         time.Duration
	HeartbeatInterval time.Duration
}

func Load() *Config {
	return &Config{
		Port:              getEnv("CARD_PORT", getEnv("PORT", "8463")),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/wl_card?sslmode=disable"),
		JWTSecret:         getEnv("JWT_SECRET", ""),
		CardEncKey:        getEnv("CARD_ENC_KEY", ""),
		ControlPlaneURL:   getEnv("TWO_PARTY_GATE_URL", "http://localhost:8460"),
		ControlPlaneToken: getEnv("TWO_PARTY_GATE_TOKEN", ""),
		WLClientID:        getEnv("WL_CLIENT_ID", ""),
		LicenseKey:        getEnv("WL_LICENSE_KEY", ""),
		Product:           getEnv("WL_PRODUCT", "card"),
		InstanceID:        getEnv("WL_INSTANCE_ID", "default"),
		BCryptCost:        getInt("BCRYPT_COST", 12),
		JWTExpiry:         getDuration("JWT_EXPIRY", 24*time.Hour),
		HeartbeatInterval: getDuration("HEARTBEAT_INTERVAL", 30*time.Second),
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
