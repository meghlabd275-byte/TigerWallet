// BackupService - logical database backup and restore.
//
// A backup exports every table in the public schema to a directory as a
// pg_dump-compatible plain SQL file: schema DDL first, then row data as COPY
// blocks. Restore replays one of those files. Nothing here reports success
// unless the work actually happened.
//
// The export deliberately does not use pg_dump: the admin backend must be able
// to take a backup against a managed database without requiring the client
// binaries to be installed in the container.
package services

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tigerwallet/super-admin/internal/database"
)

type BackupService struct {
	// BackupDir is where export files are written. It defaults to BACKUP_DIR or
	// /var/lib/tigerwallet/backups.
	BackupDir string
}

func NewBackupService() *BackupService {
	dir := os.Getenv("BACKUP_DIR")
	if dir == "" {
		dir = "/var/lib/tigerwallet/backups"
	}
	return &BackupService{BackupDir: dir}
}

var safeTableName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// listTables returns the user tables in the public schema. Table names cannot
// be bound as query parameters, so every name is validated before it reaches a
// query string.
func listTables(ctx context.Context) ([]string, error) {
	rows, err := database.Pool.Query(ctx, `
		SELECT tablename FROM pg_tables
		WHERE schemaname = 'public'
		ORDER BY tablename
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		if !safeTableName.MatchString(name) {
			continue
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

// quoteIdent double-quotes an identifier so reserved words and mixed case are
// handled correctly.
func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// CreateBackup writes a full logical export and returns the file path and its
// size in bytes.
func (s *BackupService) CreateBackup(ctx context.Context, backupID uuid.UUID) (string, int64, error) {
	if err := os.MkdirAll(s.BackupDir, 0o700); err != nil {
		return "", 0, fmt.Errorf("create backup directory: %w", err)
	}

	tables, err := listTables(ctx)
	if err != nil {
		return "", 0, err
	}
	if len(tables) == 0 {
		return "", 0, errors.New("no tables found to back up")
	}

	path := filepath.Join(s.BackupDir, fmt.Sprintf("%s.sql", backupID))
	// The file holds every row in the database, so it must not be readable by
	// other users on the host.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return "", 0, fmt.Errorf("create backup file: %w", err)
	}
	defer f.Close()

	w := bufio.NewWriterSize(f, 1<<20)
	if err := s.writeDump(ctx, w, tables); err != nil {
		f.Close()
		os.Remove(path)
		return "", 0, err
	}
	if err := w.Flush(); err != nil {
		os.Remove(path)
		return "", 0, fmt.Errorf("flush backup: %w", err)
	}
	if err := f.Sync(); err != nil {
		os.Remove(path)
		return "", 0, fmt.Errorf("sync backup: %w", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return "", 0, err
	}
	if info.Size() == 0 {
		os.Remove(path)
		return "", 0, errors.New("backup file is empty")
	}
	return path, info.Size(), nil
}

func (s *BackupService) writeDump(ctx context.Context, w io.Writer, tables []string) error {
	fmt.Fprintf(w, "-- TigerWallet logical backup\n-- Generated %s\n-- Tables: %d\n\n",
		time.Now().UTC().Format(time.RFC3339), len(tables))
	if _, err := w.Write([]byte("BEGIN;\n\n")); err != nil {
		return err
	}

	// Schema first, so the data section only ever references tables that exist.
	for _, table := range tables {
		ddl, err := tableDDL(ctx, table)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "-- Table: %s\n%s\n\n", table, ddl); err != nil {
			return err
		}
	}

	// Data as COPY blocks, which restore far faster than INSERT statements.
	// The block is written in the pg_dump layout: a COPY ... FROM STDIN header
	// carrying the column list, the rows, then a \. terminator.
	for _, table := range tables {
		cols, err := tableColumns(ctx, table)
		if err != nil {
			return err
		}
		if len(cols) == 0 {
			continue
		}

		quoted := make([]string, len(cols))
		for i, c := range cols {
			quoted[i] = quoteIdent(c)
		}
		if _, err := fmt.Fprintf(w, "-- Data for %s\nCOPY %s (%s) FROM STDIN WITH (FORMAT text);\n",
			table, quoteIdent(table), strings.Join(quoted, ", ")); err != nil {
			return err
		}

		query := fmt.Sprintf("COPY %s (%s) TO STDOUT WITH (FORMAT text)",
			quoteIdent(table), strings.Join(quoted, ", "))
		conn, err := database.Pool.Acquire(ctx)
		if err != nil {
			return fmt.Errorf("acquire connection for %s: %w", table, err)
		}
		_, copyErr := conn.Conn().PgConn().CopyTo(ctx, w, query)
		conn.Release()
		if copyErr != nil {
			return fmt.Errorf("copy %s: %w", table, copyErr)
		}
		if _, err := w.Write([]byte("\\.\n\n")); err != nil {
			return err
		}
	}

	_, err := w.Write([]byte("COMMIT;\n"))
	return err
}

// tableColumns returns a table's columns in ordinal order. COPY data is
// positional, so the order must be preserved exactly between dump and restore.
func tableColumns(ctx context.Context, table string) ([]string, error) {
	rows, err := database.Pool.Query(ctx, `
		SELECT column_name FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1
		ORDER BY ordinal_position
	`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		cols = append(cols, name)
	}
	return cols, rows.Err()
}

// tableDDL reconstructs a CREATE TABLE statement from the catalog. It covers
// the column types, nullability and defaults the admin schema uses.
func tableDDL(ctx context.Context, table string) (string, error) {
	rows, err := database.Pool.Query(ctx, `
		SELECT column_name, data_type, character_maximum_length, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1
		ORDER BY ordinal_position
	`, table)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var cols []string
	for rows.Next() {
		var name, dataType, isNullable string
		var maxLen *int
		var def *string
		if err := rows.Scan(&name, &dataType, &maxLen, &isNullable, &def); err != nil {
			return "", err
		}

		typ := dataType
		if maxLen != nil && (dataType == "character varying" || dataType == "character") {
			typ = fmt.Sprintf("%s(%d)", dataType, *maxLen)
		}
		if dataType == "USER-DEFINED" || dataType == "ARRAY" {
			// Fall back to the catalog so enums and arrays keep their real type.
			var udt string
			if err := database.Pool.QueryRow(ctx, `
				SELECT udt_name FROM information_schema.columns
				WHERE table_schema = 'public' AND table_name = $1 AND column_name = $2
			`, table, name).Scan(&udt); err == nil && udt != "" {
				typ = udt
			}
		}

		col := fmt.Sprintf("    %s %s", quoteIdent(name), typ)
		if def != nil {
			col += " DEFAULT " + *def
		}
		if isNullable == "NO" {
			col += " NOT NULL"
		}
		cols = append(cols, col)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(cols) == 0 {
		return "", fmt.Errorf("no columns found for table %s", table)
	}

	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n%s\n);",
		quoteIdent(table), strings.Join(cols, ",\n")), nil
}

// RestoreBackup replays an export file. It runs inside one transaction, so a
// failure part way through leaves the database untouched.
func (s *BackupService) RestoreBackup(ctx context.Context, path string) error {
	clean := filepath.Clean(path)
	// Refuse anything outside the backup directory so a crafted path cannot
	// read arbitrary files.
	absDir, err := filepath.Abs(s.BackupDir)
	if err != nil {
		return err
	}
	absPath, err := filepath.Abs(clean)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(absPath, absDir+string(os.PathSeparator)) {
		return errors.New("backup path is outside the backup directory")
	}

	f, err := os.Open(absPath)
	if err != nil {
		return fmt.Errorf("open backup: %w", err)
	}
	defer f.Close()

	tx, err := database.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// COPY FROM STDIN needs the connection, so statements are executed one at a
	// time as they are read from the file.
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1<<20), 16<<20)

	var statement strings.Builder
	applied := 0
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "COPY ") {
			if err := flushStatement(ctx, tx, &statement); err != nil {
				return err
			}
			applied++

			table, columns, err := parseCopyHeader(line)
			if err != nil {
				return err
			}
			data, err := readCopyData(scanner)
			if err != nil {
				return err
			}
			query := fmt.Sprintf("COPY %s (%s) FROM STDIN WITH (FORMAT text)", table, columns)
			if _, err := tx.Conn().PgConn().CopyFrom(ctx, strings.NewReader(data), query); err != nil {
				return fmt.Errorf("copy into %s: %w", table, err)
			}
			continue
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}

		statement.WriteString(line)
		statement.WriteString("\n")
		if strings.HasSuffix(trimmed, ";") {
			if err := flushStatement(ctx, tx, &statement); err != nil {
				return err
			}
			applied++
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if err := flushStatement(ctx, tx, &statement); err != nil {
		return err
	}

	if applied == 0 {
		return errors.New("backup file contains no statements")
	}
	return tx.Commit(ctx)
}

func flushStatement(ctx context.Context, tx pgx.Tx, sb *strings.Builder) error {
	sql := strings.TrimSpace(sb.String())
	sb.Reset()
	if sql == "" {
		return nil
	}
	// The dump wraps everything in its own transaction; nesting is not allowed.
	upper := strings.ToUpper(sql)
	if upper == "BEGIN;" || upper == "COMMIT;" {
		return nil
	}
	if _, err := tx.Exec(ctx, sql); err != nil {
		return fmt.Errorf("execute %.60s: %w", sql, err)
	}
	return nil
}

var copyHeader = regexp.MustCompile(`^COPY\s+(.+?)\s*\((.+)\)\s+FROM\s+STDIN`)

// parseCopyHeader extracts the table and column list from a COPY header,
// re-quoting both so they are safe to interpolate. Column order is preserved
// because COPY data is positional.
func parseCopyHeader(line string) (string, string, error) {
	m := copyHeader.FindStringSubmatch(strings.TrimSpace(line))
	if m == nil {
		return "", "", fmt.Errorf("unrecognised COPY header: %.60s", line)
	}

	table := strings.Trim(strings.TrimSpace(m[1]), `"`)
	if !safeTableName.MatchString(table) {
		return "", "", fmt.Errorf("unsafe table name in backup: %q", table)
	}

	var cols []string
	for _, c := range strings.Split(m[2], ",") {
		name := strings.Trim(strings.TrimSpace(c), `"`)
		if !safeTableName.MatchString(name) {
			return "", "", fmt.Errorf("unsafe column name in backup: %q", name)
		}
		cols = append(cols, quoteIdent(name))
	}
	if len(cols) == 0 {
		return "", "", errors.New("COPY header has no columns")
	}
	return quoteIdent(table), strings.Join(cols, ", "), nil
}

// readCopyData consumes COPY text lines until the terminator. A line of "\."
// on its own ends the block.
func readCopyData(scanner *bufio.Scanner) (string, error) {
	var b strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if line == `\.` {
			return b.String(), nil
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return "", errors.New("backup ended inside a COPY block")
}
