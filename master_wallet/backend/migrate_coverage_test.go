package main

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// runMigrations must produce every table and column the handlers name. The
// queries below were each taken from production code and previously failed at
// runtime: audit_logs has no user_id/action column, and signers was queried by
// the auto-signer but never created.
func TestMigrationsCoverHandlerQueries(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping migration coverage test")
	}
	ctx := context.Background()

	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer admin.Close()

	// A dedicated database keeps the run independent of whatever is already
	// installed in the shared one, including extension placement.
	const dbName = "mw_migrate_check"
	if _, err := admin.Exec(ctx, `DROP DATABASE IF EXISTS `+dbName); err != nil {
		t.Fatalf("drop database: %v", err)
	}
	if _, err := admin.Exec(ctx, `CREATE DATABASE `+dbName); err != nil {
		t.Fatalf("create database: %v", err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(context.Background(), `DROP DATABASE IF EXISTS `+dbName); err != nil {
			t.Logf("drop database: %v", err)
		}
	})

	scopedDSN := replaceDatabase(dsn, dbName)
	pool, err := pgxpool.New(ctx, scopedDSN)
	if err != nil {
		t.Fatalf("connect scoped: %v", err)
	}
	defer pool.Close()

	// Extensions must exist before the schema statements that use them run.
	for _, ext := range []string{"pgcrypto", `"uuid-ossp"`, "citext"} {
		if _, err := pool.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS "+ext); err != nil {
			t.Fatalf("extension %s: %v", ext, err)
		}
	}

	if err := runMigrations(ctx, pool); err != nil {
		t.Fatalf("runMigrations on empty database: %v", err)
	}
	// A restart must be a no-op, since migrations run on every boot.
	if err := runMigrations(ctx, pool); err != nil {
		t.Fatalf("second runMigrations: %v", err)
	}

	// Each statement is one the application issues at runtime. PREPARE resolves
	// the names against the migrated schema, so a missing table or column fails
	// here rather than in production.
	statements := map[string]string{
		"audit write": `INSERT INTO audit_logs (master_wallet_id, event_type, event_category, actor_type, actor_id, target_type, severity, details)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		"signer lookup": `SELECT id::text FROM signers WHERE master_wallet_id = $1 AND is_active
			ORDER BY CASE signer_type WHEN 'owner' THEN 0 WHEN 'cosigner' THEN 1 ELSE 2 END, created_at LIMIT 1`,
		"signer insert": `INSERT INTO signers (master_wallet_id, user_id, label, signer_type, address)
			VALUES ($1,$2,$3,'owner',$4)`,
		"approval request insert": `INSERT INTO approval_requests (transaction_id, required_approvals, current_approvals, metadata)
			VALUES ($1,$2,0,$3)`,
		"approval threshold read": `SELECT COALESCE(ar.required_approvals, 1) FROM transactions t
			LEFT JOIN approval_requests ar ON ar.transaction_id = t.id
			WHERE t.id = $1 AND t.master_wallet_id = $2`,
		"signature insert": `INSERT INTO transaction_signatures (transaction_id, signer_id, signature_status, approved_at)
			VALUES ($1, $2, 'signed', NOW())`,
	}
	for name, sql := range statements {
		probe := "probe_" + strings.ReplaceAll(name, " ", "_")
		if _, err := pool.Exec(ctx, "PREPARE "+probe+" AS "+sql); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// replaceDatabase swaps the database segment of a PostgreSQL URL, keeping any
// query string intact.
func replaceDatabase(dsn, dbName string) string {
	slash := strings.LastIndex(dsn, "/")
	if slash < 0 {
		return dsn
	}
	rest := dsn[slash+1:]
	if q := strings.Index(rest, "?"); q >= 0 {
		return dsn[:slash+1] + dbName + rest[q:]
	}
	return dsn[:slash+1] + dbName
}
