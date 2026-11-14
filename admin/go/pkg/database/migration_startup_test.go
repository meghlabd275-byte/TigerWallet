// Migration test — NewPostgresDB runs AutoMigrate at startup, so a failure
// here means the admin API cannot boot at all. The service migrates Blockchain
// twice (once in autoMigrate, once alongside TradingPair), so the second call
// always hits an already-existing table even on a brand-new database. Both the
// first boot and every restart must therefore succeed.
package database

import (
	"database/sql"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	_ "github.com/lib/pq"
	"github.com/tigerwallet/admin/internal/config"
)

// testConfig returns a config pointing at the test database. It skips when no
// database is configured so the suite stays CI-safe, and resets the schema so
// every run starts from an empty database.
//
// The connection parameters are derived from TEST_DATABASE_URL rather than
// hardcoded, so the suite works against whatever PostgreSQL the caller points
// it at (the previous fixed port made the test unusable outside one sandbox).
func testConfig(t *testing.T) *config.Config {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL/DATABASE_URL not set")
	}

	cfg := config.Load()
	host, port, user, password, dbname := parseDSN(dsn)
	cfg.DBHost = host
	cfg.DBPort = port
	cfg.DBUser = user
	cfg.DBPassword = password
	cfg.DBName = dbname

	raw, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("database unavailable: %v", err)
	}
	defer raw.Close()
	if err := raw.Ping(); err != nil {
		t.Skipf("database unreachable: %v", err)
	}
	if _, err := raw.Exec("DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public"); err != nil {
		t.Skipf("cannot reset schema: %v", err)
	}
	return cfg
}

// parseDSN extracts the connection parameters from a keyword/value DSN
// ("host=... port=... user=...") or a postgres:// URL. Missing values keep the
// libpq defaults so a partial DSN still works.
func parseDSN(dsn string) (host string, port int, user, password, dbname string) {
	host, port, user, password, dbname = "127.0.0.1", 5432, "postgres", "", "postgres"

	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		if u, err := url.Parse(dsn); err == nil {
			if u.Hostname() != "" {
				host = u.Hostname()
			}
			if p := u.Port(); p != "" {
				if n, err := strconv.Atoi(p); err == nil {
					port = n
				}
			}
			if u.User != nil {
				user = u.User.Username()
				if pw, ok := u.User.Password(); ok {
					password = pw
				}
			}
			if name := strings.TrimPrefix(u.Path, "/"); name != "" {
				dbname = name
			}
		}
		return host, port, user, password, dbname
	}

	for _, field := range strings.Fields(dsn) {
		k, v, ok := strings.Cut(field, "=")
		if !ok {
			continue
		}
		switch k {
		case "host":
			host = v
		case "port":
			if n, err := strconv.Atoi(v); err == nil {
				port = n
			}
		case "user":
			user = v
		case "password":
			password = v
		case "dbname":
			dbname = v
		}
	}
	return host, port, user, password, dbname
}

func TestNewPostgresDBOnFreshDatabase(t *testing.T) {
	cfg := testConfig(t)

	db, err := NewPostgresDB(cfg)
	if err != nil {
		t.Fatalf("NewPostgresDB failed on a fresh database: %v", err)
	}
	if db == nil {
		t.Fatal("NewPostgresDB returned a nil database")
	}
}

// TestMigrateIsIdempotent covers the restart path: the tables already exist, so
// every AutoMigrate call inspects existing columns.
func TestMigrateIsIdempotent(t *testing.T) {
	cfg := testConfig(t)

	for attempt := 1; attempt <= 2; attempt++ {
		if _, err := NewPostgresDB(cfg); err != nil {
			t.Fatalf("attempt %d: NewPostgresDB failed: %v", attempt, err)
		}
	}
}
