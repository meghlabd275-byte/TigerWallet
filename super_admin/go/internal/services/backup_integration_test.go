package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tigerwallet/super-admin/internal/database"
)

// A backup that cannot be restored is worthless, so this exercises the real
// round trip against a live server: create tables with awkward values, dump
// them, delete the rows, restore, and compare byte for byte. Set
// TEST_DATABASE_URL to run it; without that the test skips. The fixture
// overwrites its own tables, so the database must be disposable.
func TestBackupRestoreRoundTrip(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping backup integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	previous := database.Pool
	database.Pool = pool
	defer func() { database.Pool = previous }()

	// The export walks every table in the public schema, so drop anything a
	// previous run or test left behind.
	rows, err := pool.Query(ctx, `SELECT tablename FROM pg_tables WHERE schemaname='public'`)
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	var existing []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan table: %v", err)
		}
		existing = append(existing, name)
	}
	rows.Close()
	for _, name := range existing {
		if _, err := pool.Exec(ctx, "DROP TABLE IF EXISTS "+quoteIdent(name)+" CASCADE"); err != nil {
			t.Fatalf("drop %s: %v", name, err)
		}
	}

	for _, stmt := range []string{
		`CREATE TABLE backup_items (
			id SERIAL PRIMARY KEY,
			label TEXT NOT NULL,
			amount NUMERIC(20,8),
			note TEXT,
			active BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE backup_empty (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			payload JSONB
		)`,
	} {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Fatalf("schema: %v", err)
		}
	}

	// Values chosen to break naive dump/restore: a tab, a newline, a backslash,
	// a double quote, a SQL statement, a NULL, and an explicit numeric.
	fixtures := []struct {
		label  string
		amount string
		note   any
	}{
		{"plain", "1.50000000", "hello"},
		{"tabs\tand\nnewlines", "2.00000000", "back\\slash"},
		{`quote"and 'apostrophe'`, "3.25000000", "drop table users; --"},
		{"null-note", "4.00000000", nil},
		{"unicode-₹-日本", "5.12345678", "emoji 🐯"},
	}
	for _, f := range fixtures {
		if _, err := pool.Exec(ctx,
			`INSERT INTO backup_items (label, amount, note) VALUES ($1,$2,$3)`,
			f.label, f.amount, f.note); err != nil {
			t.Fatalf("insert fixture: %v", err)
		}
	}

	dir := t.TempDir()
	svc := &BackupService{BackupDir: dir}
	backupID := uuid.New()

	path, size, err := svc.CreateBackup(ctx, backupID)
	if err != nil {
		t.Fatalf("CreateBackup: %v", err)
	}
	if size <= 0 {
		t.Fatal("CreateBackup reported a non-positive size")
	}
	if want := filepath.Join(dir, backupID.String()+".sql"); path != want {
		t.Errorf("backup path = %q, want %q", path, want)
	}

	// The dump holds every row in the database, so it must not be world
	// readable.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat backup: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("backup permissions = %o, want 600", perm)
	}

	// Wipe the rows but keep the schema, then restore.
	if _, err := pool.Exec(ctx, `TRUNCATE backup_items, backup_empty`); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	if err := svc.RestoreBackup(ctx, path); err != nil {
		t.Fatalf("RestoreBackup: %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM backup_items`).Scan(&count); err != nil {
		t.Fatalf("count after restore: %v", err)
	}
	if count != len(fixtures) {
		t.Fatalf("restored %d rows, want %d", count, len(fixtures))
	}

	for _, f := range fixtures {
		var label, amount string
		var note *string
		if err := pool.QueryRow(ctx,
			`SELECT label, amount::text, note FROM backup_items WHERE label = $1`, f.label).
			Scan(&label, &amount, &note); err != nil {
			t.Fatalf("read back %q: %v", f.label, err)
		}
		if label != f.label {
			t.Errorf("label = %q, want %q", label, f.label)
		}
		if amount != f.amount {
			t.Errorf("amount for %q = %q, want %q", f.label, amount, f.amount)
		}
		wantNote := ""
		if f.note != nil {
			wantNote = f.note.(string)
		}
		gotNote := ""
		if note != nil {
			gotNote = *note
		}
		if gotNote != wantNote {
			t.Errorf("note for %q = %q, want %q", f.label, gotNote, wantNote)
		}
	}

	// A restore must refuse a path outside the backup directory.
	outside := filepath.Join(t.TempDir(), "evil.sql")
	if err := os.WriteFile(outside, []byte("SELECT 1;"), 0o600); err != nil {
		t.Fatalf("write outside file: %v", err)
	}
	if err := svc.RestoreBackup(ctx, outside); err == nil {
		t.Error("RestoreBackup accepted a path outside the backup directory")
	}
}

// A backup file that does not exist must fail loudly rather than reporting a
// successful restore.
func TestRestoreBackupMissingFile(t *testing.T) {
	svc := &BackupService{BackupDir: t.TempDir()}
	err := svc.RestoreBackup(context.Background(), filepath.Join(svc.BackupDir, "nope.sql"))
	if err == nil {
		t.Fatal("RestoreBackup accepted a missing file")
	}
}

func TestParseCopyHeaderRejectsUnsafeNames(t *testing.T) {
	if _, _, err := parseCopyHeader(`COPY users; DROP TABLE x; (id) FROM STDIN`); err == nil {
		t.Error("parseCopyHeader accepted a malformed header")
	}
	// A schema-qualified name is refused: the dump only ever writes bare table
	// names, so anything else is a tampered file.
	if _, _, err := parseCopyHeader(`COPY "public"."users" (id, name) FROM STDIN`); err == nil {
		t.Error("parseCopyHeader accepted a schema-qualified table name")
	}
	if _, _, err := parseCopyHeader(`COPY "users" (id, "name; DROP TABLE x") FROM STDIN`); err == nil {
		t.Error("parseCopyHeader accepted an unsafe column name")
	}

	table, cols, err := parseCopyHeader(`COPY "audit_logs" ("id", "action", "resource_id") FROM STDIN WITH (FORMAT text);`)
	if err != nil {
		t.Fatalf("parseCopyHeader: %v", err)
	}
	if table != `"audit_logs"` {
		t.Errorf("table = %q", table)
	}
	// Column order must survive, because COPY data is positional.
	if cols != `"id", "action", "resource_id"` {
		t.Errorf("cols = %q, want %q", cols, `"id", "action", "resource_id"`)
	}
}

// The scheduled loop is what makes backups exist at all in production, so it is
// exercised against a live database: the loop must record a completed row, write
// the file, and then delete both once the file falls outside the retention
// window.
func TestScheduledBackupLoopWritesAndPrunes(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping scheduled backup test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	previous := database.Pool
	database.Pool = pool
	defer func() { database.Pool = previous }()

	if _, err := pool.Exec(ctx, backupsTableDDL); err != nil {
		t.Fatalf("create backups table: %v", err)
	}
	// The schema check runs the real migrations, which use CREATE TABLE IF NOT
	// EXISTS. A table left behind here would shadow the production definition, so
	// remove it when the test ends.
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DROP TABLE IF EXISTS backups`); err != nil {
			t.Logf("drop backups table: %v", err)
		}
	})
	if _, err := pool.Exec(ctx, `DELETE FROM backups`); err != nil {
		t.Fatalf("clear backups: %v", err)
	}

	dir := t.TempDir()
	svc := &BackupService{BackupDir: dir}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 20ms is far below any real schedule; it only has to fire while we wait.
	go svc.ScheduledBackupLoop(runCtx, 20*time.Millisecond, 0)

	var status, filePath string
	var size int64
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		err := pool.QueryRow(ctx,
			`SELECT status, file_path, file_size FROM backups
			 WHERE status='completed' ORDER BY completed_at DESC LIMIT 1`).
			Scan(&status, &filePath, &size)
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	cancel()

	if status != "completed" {
		t.Fatalf("scheduled loop never recorded a completed backup (status %q)", status)
	}
	if size <= 0 {
		t.Errorf("scheduled backup size = %d, want > 0", size)
	}
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("scheduled backup file missing: %v", err)
	}
	if info.Size() != size {
		t.Errorf("recorded size %d does not match file size %d", size, info.Size())
	}

	// Age the row past a one day window; the next run must remove row and file.
	if _, err := pool.Exec(ctx,
		`UPDATE backups SET completed_at = NOW() - INTERVAL '3 days' WHERE file_path = $1`, filePath); err != nil {
		t.Fatalf("age backup: %v", err)
	}
	svc.pruneOldBackups(ctx, 1)

	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("expired backup file still present (stat err = %v)", err)
	}
	var remaining int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM backups WHERE file_path = $1`, filePath).Scan(&remaining); err != nil {
		t.Fatalf("count after prune: %v", err)
	}
	if remaining != 0 {
		t.Errorf("expired backup row still present: %d", remaining)
	}
}

// A retention window that keeps everything must not delete a recent backup.
func TestPruneOldBackupsKeepsRecent(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping retention test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	previous := database.Pool
	database.Pool = pool
	defer func() { database.Pool = previous }()

	if _, err := pool.Exec(ctx, backupsTableDDL); err != nil {
		t.Fatalf("create backups table: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DROP TABLE IF EXISTS backups`); err != nil {
			t.Logf("drop backups table: %v", err)
		}
	})

	dir := t.TempDir()
	svc := &BackupService{BackupDir: dir}
	fresh := filepath.Join(dir, uuid.New().String()+".sql")
	if err := os.WriteFile(fresh, []byte("SELECT 1;\n"), 0o600); err != nil {
		t.Fatalf("write fresh backup: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO backups (backup_type, file_path, file_size, status, completed_at)
		 VALUES ('full',$1,$2,'completed',NOW())`, fresh, 9); err != nil {
		t.Fatalf("insert fresh backup: %v", err)
	}

	svc.pruneOldBackups(ctx, 30)

	if _, err := os.Stat(fresh); err != nil {
		t.Errorf("recent backup file was removed: %v", err)
	}
	var remaining int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM backups WHERE file_path = $1`, fresh).Scan(&remaining); err != nil {
		t.Fatalf("count: %v", err)
	}
	if remaining != 1 {
		t.Errorf("recent backup row count = %d, want 1", remaining)
	}
}

// backupsTableDDL mirrors the production schema for the table the scheduler
// records its jobs in, including columns the test does not itself set.
const backupsTableDDL = `CREATE TABLE IF NOT EXISTS backups (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	backup_type VARCHAR(50) NOT NULL,
	file_path TEXT NOT NULL,
	file_size BIGINT,
	status VARCHAR(50) DEFAULT 'pending',
	created_by UUID,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	completed_at TIMESTAMPTZ
)`
