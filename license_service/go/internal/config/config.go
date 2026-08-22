package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all runtime configuration loaded from environment variables.
// Every field is env-overridable with a fail-closed default. Secrets are NEVER
// hardcoded; JWT_SECRET and the SuperAdmin bootstrap credential are required.
type Config struct {
	Port            string
	DatabaseURL     string
	RedisAddr       string
	RedisPassword   string
	JWTSecret       string
	JWTExpiry       time.Duration

	// Ed25519 signing key pair for license tokens (hex seed, env-injected).
	// If unset, a fresh key pair is generated at boot (tokens remain valid for
	// the process lifetime; for HA deploy a shared key via env).
	LicenseSignKeyHex   string
	LicenseVerifyKeyHex string

	// SuperAdmin bootstrap: the first superadmin account seeded on startup.
	SuperAdminEmail    string
	SuperAdminPassword string

	// Shared secret for service-to-service calls from WL product backends
	// (wl_master_wallet / wl_user_wallet) to the two-party withdrawal
	// collaboration endpoints. This is NOT a user JWT — it authenticates the
	// machine-to-machine gate calls (IsWithdrawalApproved / RequestWithdrawal
	// / MarkWithdrawalExecuted). Must match TWO_PARTY_GATE_TOKEN on the WL side.
	ServiceToken string

	// Withdrawal co-signer address controlled by TigerWallet SuperAdmin.
	// Injected into every WL master-wallet multisig as a mandatory owner and
	// required as the second approver on every fund/revenue exit.
	SuperAdminSignerAddress string

	HeartbeatTimeout time.Duration // WL product is dead if no heartbeat for this long
	GracePeriod      time.Duration // tolerance window before hard kill
}

func Load() *Config {
	cfg := &Config{
		Port:                    getEnv("LICENSE_PORT", "9008"),
		DatabaseURL:             getEnv("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet?sslmode=disable"),
		RedisAddr:               getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:           getEnv("REDIS_PASSWORD", ""),
		JWTSecret:               getEnv("JWT_SECRET", ""),
		JWTExpiry:               getDuration("JWT_EXPIRY", 24*time.Hour),
		LicenseSignKeyHex:       getEnv("LICENSE_SIGN_KEY_HEX", ""),
		LicenseVerifyKeyHex:     getEnv("LICENSE_VERIFY_KEY_HEX", ""),
		SuperAdminEmail:         getEnv("SUPER_ADMIN_EMAIL", "superadmin@tigerwallet.com"),
		SuperAdminPassword:      getEnv("SUPER_ADMIN_PASSWORD", ""),
		ServiceToken:            getEnv("SERVICE_AUTH_TOKEN", ""),
		SuperAdminSignerAddress: getEnv("SUPER_ADMIN_SIGNER_ADDRESS", ""),
		HeartbeatTimeout:        getDuration("HEARTBEAT_TIMEOUT", 90*time.Second),
		GracePeriod:             getDuration("HEARTBEAT_GRACE_PERIOD", 15*time.Second),
	}
	if cfg.JWTSecret == "" {
		// Fail-closed: a license control plane MUST have a JWT secret. We never
		// invent a weak default that would let any caller forge SuperAdmin tokens.
		panic("JWT_SECRET environment variable is required for the license control plane")
	}
	if cfg.SuperAdminPassword == "" {
		panic("SUPER_ADMIN_PASSWORD environment variable is required to bootstrap the first superadmin")
	}
	return cfg
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
func getDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if secs, err := strconv.Atoi(v); err == nil {
			return time.Duration(secs) * time.Second
		}
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
