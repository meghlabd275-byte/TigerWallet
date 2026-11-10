// schema_test.go — applies schemaSQL to a real PostgreSQL database.
//
// The script is not ordered by dependency and previously contained a syntax
// error, either of which aborts startup with "failed to apply schema". Running
// it here proves the whole script lands.
package main

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// scratchPool connects to a dedicated schema so the test cannot collide with a
// real deployment or another test. Connection problems skip; schema problems
// are failures.
func scratchPool(t *testing.T) *pgxpool.Pool {
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
	schema := "pptest_" + strconv.FormatInt(time.Now().UnixNano(), 36)
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
	if _, err := pool.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS pgcrypto"); err != nil {
		pool.Close()
		t.Skipf("pgcrypto unavailable: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DROP SCHEMA IF EXISTS "+schema+" CASCADE")
		pool.Close()
	})
	return pool
}

func TestApplySchemaCompletes(t *testing.T) {
	pool := scratchPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := applySchema(ctx, pool); err != nil {
		t.Fatalf("applySchema failed: %v", err)
	}
	// Idempotency: a second run must also succeed.
	if err := applySchema(ctx, pool); err != nil {
		t.Fatalf("second applySchema failed: %v", err)
	}

	var tables int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM information_schema.tables WHERE table_schema = current_schema()`,
	).Scan(&tables); err != nil {
		t.Fatalf("count tables: %v", err)
	}
	if tables < 15 {
		t.Fatalf("expected at least 15 tables, got %d", tables)
	}
}

// TestDeferredAlterColumnsExist proves the ALTER statements that precede their
// CREATE TABLE in the script are eventually applied. Without the retry passes
// the schema aborts on the first one and every owner_id column stays missing,
// silently disabling the ownership checks that use them.
func TestDeferredAlterColumnsExist(t *testing.T) {
	pool := scratchPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := applySchema(ctx, pool); err != nil {
		t.Fatalf("applySchema failed: %v", err)
	}

	for _, table := range []string{"tokens", "token_listings", "launchpads", "market_maker_orders"} {
		var exists bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM information_schema.columns
			                WHERE table_schema = current_schema() AND table_name = $1
			                  AND column_name = 'owner_id')`,
			table).Scan(&exists); err != nil {
			t.Fatalf("probe %s.owner_id: %v", table, err)
		}
		if !exists {
			t.Fatalf("%s is missing owner_id — ownership checks would not apply", table)
		}
	}

	var verified bool
	if err := pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM information_schema.columns
		                WHERE table_schema = current_schema() AND table_name = 'fee_payments'
		                  AND column_name = 'verified_at')`,
	).Scan(&verified); err != nil {
		t.Fatalf("probe fee_payments.verified_at: %v", err)
	}
	if !verified {
		t.Fatal("fee_payments is missing verified_at")
	}
}
