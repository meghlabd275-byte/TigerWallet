// Backup - Automated backup and restore service
package backup

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type BackupService struct {
	dbURL     string
	backupDir string
	retention int // days
}

type Backup struct {
	ID          uuid.UUID
	BackupType  string    // full, incremental, config
	FilePath    string
	FileSize    int64
	Status      string    // pending, running, completed, failed
	CreatedBy   uuid.UUID
	CreatedAt   time.Time
	CompletedAt *time.Time
}

type BackupConfig struct {
	FullBackupSchedule    string // cron expression
	IncrementalSchedule   string
	BackupPath           string
	RetentionDays        int
	CompressionEnabled   bool
	EncryptionEnabled    bool
	ExcludeTables        []string
}

func NewBackupService(dbURL, backupDir string, retentionDays int) *BackupService {
	return &BackupService{
		dbURL:     dbURL,
		backupDir: backupDir,
		retention: retentionDays,
	}
}

// CreateFullBackup creates a full database backup
func (s *BackupService) CreateFullBackup(ctx context.Context, adminID uuid.UUID) (*Backup, error) {
	backup := &Backup{
		ID:         uuid.New(),
		BackupType: "full",
		Status:     "running",
		CreatedBy:  adminID,
		CreatedAt:  time.Now(),
	}

	// Ensure backup directory exists
	if err := os.MkdirAll(s.backupDir, 0755); err != nil {
		backup.Status = "failed"
		return backup, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Generate backup filename
	filename := fmt.Sprintf("backup_full_%s_%s.sql.zip",
		backup.ID.String()[:8],
		time.Now().Format("20060102150405"))
	backup.FilePath = filepath.Join(s.backupDir, filename)

	// Create the backup
	if err := s.performBackup(ctx, backup); err != nil {
		backup.Status = "failed"
		return backup, err
	}

	backup.Status = "completed"
	now := time.Now()
	backup.CompletedAt = &now

	// Get file size
	if info, err := os.Stat(backup.FilePath); err == nil {
		backup.FileSize = info.Size()
	}

	return backup, nil
}

// performBackup performs the actual backup operation
func (s *BackupService) performBackup(ctx context.Context, backup *Backup) error {
	// Connect to database
	conn, err := pgx.Connect(ctx, s.dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close(ctx)

	// Get list of tables
	rows, err := conn.Query(ctx, `
		SELECT tablename FROM pg_tables 
		WHERE schemaname = 'public'
		ORDER BY tablename
	`)
	if err != nil {
		return fmt.Errorf("failed to get tables: %w", err)
	}
	defer rows.Close(ctx)

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return err
		}
		tables = append(tables, table)
	}

	// Create zip file
	file, err := os.Create(backup.FilePath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	// Create SQL file in zip
	sqlFile, err := zipWriter.Create("backup.sql")
	if err != nil {
		return fmt.Errorf("failed to create zip entry: %w", err)
	}

	// Write schema
	schema, err := s.dumpSchema(ctx, conn, tables)
	if err != nil {
		return err
	}
	if _, err := sqlFile.Write([]byte(schema)); err != nil {
		return fmt.Errorf("failed to write schema: %w", err)
	}

	// Write data for each table
	for _, table := range tables {
		data, err := s.dumpTableData(ctx, conn, table)
		if err != nil {
			return fmt.Errorf("failed to dump table %s: %w", table, err)
		}
		if _, err := sqlFile.Write(data); err != nil {
			return fmt.Errorf("failed to write table data: %w", err)
		}
	}

	return nil
}

func (s *BackupService) dumpSchema(ctx context.Context, conn *pgx.Conn, tables []string) (string, error) {
	var schema strings.Builder

	// Get sequence definitions
	rows, err := conn.Query(ctx, `
		SELECT sequence_name FROM information_schema.sequences 
		WHERE sequence_schema = 'public'
	`)
	if err != nil {
		return "", err
	}
	defer rows.Close(ctx)

	for rows.Next() {
		var seq string
		if err := rows.Scan(&seq); err != nil {
			return "", err
		}
		schema.WriteString(fmt.Sprintf("SELECT last_value FROM %s;\n\n", seq))
	}

	// Get table schemas
	for _, table := range tables {
		rows, err := conn.Query(ctx, fmt.Sprintf("SELECT pg_get_ddl('%s')", table))
		if err != nil {
			return "", err
		}
		for rows.Next() {
			var ddl string
			if err := rows.Scan(&ddl); err != nil {
				return "", err
			}
			schema.WriteString(ddl)
			schema.WriteString("\n\n")
		}
		rows.Close()
	}

	return schema.String(), nil
}

func (s *BackupService) dumpTableData(ctx context.Context, conn *pgx.Conn, table string) (string, error) {
	var data strings.Builder

	rows, err := conn.Query(ctx, fmt.Sprintf("SELECT * FROM %s", table))
	if err != nil {
		return "", err
	}
	defer rows.Close(ctx)

	fieldDescriptions := rows.FieldDescriptions()
	columns := make([]string, len(fieldDescriptions))
	for i, fd := range fieldDescriptions {
		columns[i] = string(fd.Name)
	}

	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return "", err
		}

		data.WriteString("INSERT INTO ")
		data.WriteString(table)
		data.WriteString(" (")
		data.WriteString(strings.Join(columns, ", "))
		data.WriteString(") VALUES (")

		for i, v := range values {
			if i > 0 {
				data.WriteString(", ")
			}
			switch v := v.(type) {
			case nil:
				data.WriteString("NULL")
			case string:
				data.WriteString(fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''")))
			case []byte:
				data.WriteString(fmt.Sprintf("'\\x%x'", fmt.Sprintf("%x", v)))
			default:
				data.WriteString(fmt.Sprintf("%v", v))
			}
		}
		data.WriteString(");\n")
	}

	return data.String(), nil
}

// RestoreBackup restores a database from backup
func (s *BackupService) RestoreBackup(ctx context.Context, backupPath string) error {
	// Open the zip file
	reader, err := zip.OpenReader(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer reader.Close()

	// Connect to database
	conn, err := pgx.Connect(ctx, s.dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close(ctx)

	// Find SQL file in zip
	var sqlFile *zip.File
	for _, f := range reader.File {
		if f.Name == "backup.sql" {
			sqlFile = f
			break
		}
	}

	if sqlFile == nil {
		return fmt.Errorf("backup.sql not found in archive")
	}

	// Read SQL content
	rc, err := sqlFile.Open()
	if err != nil {
		return fmt.Errorf("failed to open SQL file: %w", err)
	}
	defer rc.Close()

	sqlContent, err := io.ReadAll(rc)
	if err != nil {
		return fmt.Errorf("failed to read SQL content: %w", err)
	}

	// Execute SQL statements (simplified - in production use proper splitting)
	statements := strings.Split(string(sqlContent), ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}
		if _, err := conn.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("failed to execute statement: %w\nStatement: %s", err, stmt[:min(100, len(stmt))])
		}
	}

	return nil
}

// ListBackups lists all available backups
func (s *BackupService) ListBackups() ([]Backup, error) {
	entries, err := os.ReadDir(s.backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Backup{}, nil
		}
		return nil, err
	}

	var backups []Backup
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".zip" {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		backup := Backup{
			ID:         uuid.New(),
			BackupType: "full",
			FilePath:   filepath.Join(s.backupDir, entry.Name()),
			FileSize:   info.Size(),
			Status:     "completed",
			CreatedAt:  info.ModTime(),
		}
		backups = append(backups, backup)
	}

	return backups, nil
}

// DeleteOldBackups removes backups older than retention period
func (s *BackupService) DeleteOldBackups() error {
	backups, err := s.ListBackups()
	if err != nil {
		return err
	}

	cutoff := time.Now().AddDate(0, 0, -s.retention)

	for _, backup := range backups {
		if backup.CreatedAt.Before(cutoff) {
			if err := os.Remove(backup.FilePath); err != nil {
				fmt.Printf("Warning: failed to delete old backup %s: %v\n", backup.FilePath, err)
			}
		}
	}

	return nil
}

// ExportConfig exports system configuration
func (s *BackupService) ExportConfig() ([]byte, error) {
	config := map[string]interface{}{
		"exported_at":    time.Now().Format(time.RFC3339),
		"version":        "1.0",
		"backup_retention_days": s.retention,
	}

	return json.MarshalIndent(config, "", "  ")
}

// ImportConfig imports system configuration
func (s *BackupService) ImportConfig(data []byte) error {
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate config
	if version, ok := config["version"].(string); !ok || version != "1.0" {
		return fmt.Errorf("unsupported config version")
	}

	return nil
}

// AutoBackup runs automatic backup on schedule
func (s *BackupService) AutoBackup(ctx context.Context, adminID uuid.UUID) error {
	backup, err := s.CreateFullBackup(ctx, adminID)
	if err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	if backup.Status != "completed" {
		return fmt.Errorf("backup completed with status: %s", backup.Status)
	}

	// Clean up old backups
	return s.DeleteOldBackups()
}
