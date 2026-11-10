package store

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Migration must succeed on an empty database. Statements are ordered so a
// table is created before anything alters it, and a fresh deployment has no
// tables at all, which is exactly when an ordering mistake shows up.
func TestMigrateOnEmptyDatabase(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping migration test")
	}

	ctx := context.Background()

	// A dedicated schema keeps the run isolated from other tests.
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer admin.Close()
	if _, err := admin.Exec(ctx, `DROP SCHEMA IF EXISTS migrate_check CASCADE`); err != nil {
		t.Fatalf("reset schema: %v", err)
	}
	if _, err := admin.Exec(ctx, `CREATE SCHEMA migrate_check`); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	if _, err := admin.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS pgcrypto`); err != nil {
		t.Fatalf("create pgcrypto: %v", err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(context.Background(), `DROP SCHEMA IF EXISTS migrate_check CASCADE`); err != nil {
			t.Logf("drop schema: %v", err)
		}
	})

	// search_path pins every unqualified name to the throwaway schema.
	sep := "?"
	if contains(dsn, "?") {
		sep = "&"
	}
	scoped := dsn + sep + "search_path=migrate_check"
	pool, err := pgxpool.New(ctx, scoped)
	if err != nil {
		t.Fatalf("connect scoped: %v", err)
	}
	defer pool.Close()

	s := &Store{db: pool}
	if err := s.migrate(ctx); err != nil {
		t.Fatalf("migrate on empty database: %v", err)
	}

	// Running again must be a no-op: migrations are the deployment path, and a
	// restart has to work.
	if err := s.migrate(ctx); err != nil {
		t.Fatalf("second migrate: %v", err)
	}

	// The columns the handlers rely on must exist after migration.
	for _, check := range []struct{ table, column string }{
		{"mm_orders", "owner_id"},
		{"fee_payments", "verified_at"},
		{"tokens", "owner_id"},
		{"tokens", "contract_verified"},
		{"launchpad_contributions", "tx_hash"},
		{"launchpad_contributions", "confirmed_at"},
	} {
		var exists bool
		err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = 'migrate_check' AND table_name = $1 AND column_name = $2
			)`, check.table, check.column).Scan(&exists)
		if err != nil {
			t.Fatalf("column check %s.%s: %v", check.table, check.column, err)
		}
		if !exists {
			t.Errorf("column %s.%s missing after migration", check.table, check.column)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
