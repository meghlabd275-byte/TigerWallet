// Migration tests — apply the migration set to a real PostgreSQL database.
//
// Startup runs every statement in runMigrations and returns an error on the
// first failure, so a statement that can never succeed (an index on a table
// that is not created, for example) prevents the service from booting at all.
// Applying the set here catches that class of defect before deploy.
package database

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testPool connects to an isolated schema and installs the pool in the package
// variable the migration statements use. A dedicated schema keeps the test
// independent of any other service that shares the database. Connection
// problems skip; migration problems are failures.
func testPool(t *testing.T) {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL/DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		t.Skipf("bad database url: %v", err)
	}
	schema := "migtest_" + strconv.FormatInt(time.Now().UnixNano(), 36)
	if cfg.ConnConfig.RuntimeParams == nil {
		cfg.ConnConfig.RuntimeParams = map[string]string{}
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Skipf("database unavailable: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("database unreachable: %v", err)
	}
	if _, err := pool.Exec(ctx, "CREATE SCHEMA IF NOT EXISTS "+schema); err != nil {
		pool.Close()
		t.Skipf("cannot create scratch schema: %v", err)
	}
	Pool = pool
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DROP SCHEMA IF EXISTS "+schema+" CASCADE")
		pool.Close()
	})
}

func TestRunMigrationsAppliesCleanly(t *testing.T) {
	testPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := runMigrations(ctx); err != nil {
		t.Fatalf("migrations failed: %v", err)
	}
	// Idempotency: a second pass must also succeed.
	if err := runMigrations(ctx); err != nil {
		t.Fatalf("re-running migrations failed: %v", err)
	}
}

// TestIndexesReferenceExistingTables guards against the defect where the
// migration created an index on "sessions" while the table is "admin_sessions".
func TestIndexesReferenceExistingTables(t *testing.T) {
	testPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := runMigrations(ctx); err != nil {
		t.Fatalf("migrations failed: %v", err)
	}

	for _, table := range []string{"admin_sessions", "admin_users", "users", "audit_logs", "notifications"} {
		var exists bool
		if err := Pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM information_schema.tables
			                WHERE table_schema = current_schema() AND table_name = $1)`,
			table).Scan(&exists); err != nil {
			t.Fatalf("probe %s: %v", table, err)
		}
		if !exists {
			t.Fatalf("expected table %q to exist after migrations", table)
		}
	}

	var indexExists bool
	if err := Pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname = current_schema() AND indexname = 'idx_sessions_admin_id')`,
	).Scan(&indexExists); err != nil {
		t.Fatalf("probe index: %v", err)
	}
	if !indexExists {
		t.Fatal("idx_sessions_admin_id was not created")
	}
}
