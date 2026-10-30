// Package config holds the standalone WL-Bots backend configuration. This backend
// runs INDEPENDENTLY in the WL client's own cloud — own PG, own DB. It phones
// home to the license control plane on heartbeat and gates every request
// fail-closed via wlgate. It is a clone of the TigerWallet mm_bot_platform/
// bot_api bot management platform, but standalone (no TigerWallet cloud at
// request time).
package config

import (
	"os"
	"strings"
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
	ProjectPartyURL   string // WL ProjectParty backend for mm-configs linkage (PROJECT_PARTY_URL)
	PPServiceToken    string // shared service-to-service token with project-party (PP_SERVICE_TOKEN)
	// CORSAllowedOrigins is the browser-origin allowlist for the API.
	// Empty means no cross-origin browser access.
	CORSAllowedOrigins []string
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
		ProjectPartyURL:   getEnv("PROJECT_PARTY_URL", "http://localhost:8106"),
		PPServiceToken:    getEnv("PP_SERVICE_TOKEN", ""),
		CORSAllowedOrigins: splitCSV(getEnv("WL_BOTS_CORS_ORIGINS", "")),
	}
}

// splitCSV parses a comma-separated list, dropping blank entries so a trailing
// comma cannot introduce an empty origin or address.
func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
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
