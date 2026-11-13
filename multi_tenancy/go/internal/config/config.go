package config

import (
	"fmt"
	"os"
)

// Config is loaded entirely from the environment. No credential has a
// hard-coded default: a missing value is a startup error, not a silent
// fallback to a shared password.
type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
	AdminToken  string
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:       env("TENANT_PORT", "9002"),
		RedisURL:   env("TENANT_REDIS_URL", ""),
		AdminToken: os.Getenv("TENANT_ADMIN_TOKEN"),
	}

	// TENANT_DATABASE_URL wins; otherwise assemble one from the discrete
	// TENANT_DB_* variables the compose file supplies.
	if dsn := os.Getenv("TENANT_DATABASE_URL"); dsn != "" {
		cfg.DatabaseURL = dsn
	} else {
		host := env("TENANT_DB_HOST", "localhost")
		port := env("TENANT_DB_PORT", "5432")
		user := env("TENANT_DB_USER", "")
		pass := os.Getenv("TENANT_DB_PASSWORD")
		name := env("TENANT_DB_NAME", "tigerwallet")
		if user == "" {
			return nil, fmt.Errorf("TENANT_DB_USER (or TENANT_DATABASE_URL) must be set")
		}
		cfg.DatabaseURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, pass, host, port, name)
	}

	if cfg.RedisURL == "" {
		rHost := env("TENANT_REDIS_HOST", "localhost")
		rPort := env("TENANT_REDIS_PORT", "6379")
		cfg.RedisURL = fmt.Sprintf("redis://%s:%s", rHost, rPort)
	}

	return cfg, nil
}
