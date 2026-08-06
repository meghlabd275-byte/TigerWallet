package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"admin_system/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound      = errors.New("user not found")
	ErrTokenExpired      = errors.New("token expired")
	ErrInvalidToken      = errors.New("invalid token")
)

type AuthService struct {
	db    *pgxpool.Pool
	redis *redis.Client
	cfg   JWTConfig
}

type JWTConfig struct {
	Secret         string
	ExpirationTime time.Duration
	RefreshExpires time.Duration
	Issuer         string
}

func NewAuthService(db *pgxpool.Pool, redis *redis.Client) *AuthService {
	return &AuthService{
		db: db,
		redis: redis,
		cfg: JWTConfig{
			Secret:         "tigerwallet-admin-system-secret",
			ExpirationTime: 24 * time.Hour,
			RefreshExpires: 7 * 24 * time.Hour,
			Issuer:         "tigerwallet",
		},
	}
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*models.AuthResponse, error) {
	var user models.SystemUser
	err := s.db.QueryRow(ctx, `
		SELECT id, email, username, password_hash, role, status, last_login_at
		FROM system_users WHERE email = $1
	`, email).Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.Role, &user.Status, &user.LastLoginAt)

	if err == pgx.ErrNoRows {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	s.db.Exec(ctx, `UPDATE system_users SET last_login_at = NOW() WHERE id = $1`, user.ID)

	accessToken, _ := s.generateAccessToken(user)
	refreshToken, _ := s.generateRefreshToken(ctx, user.ID)

	return &models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.cfg.ExpirationTime.Seconds()),
		User:         &user,
	}, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*models.AuthResponse, error) {
	tokenHash := hashToken(refreshToken)
	var userID uuid.UUID
	var expiresAt time.Time

	err := s.db.QueryRow(ctx, `
		SELECT user_id, expires_at FROM refresh_tokens 
		WHERE token_hash = $1 AND expires_at > NOW()
	`, tokenHash).Scan(&userID, &expiresAt)
	if err == pgx.ErrNoRows {
		return nil, ErrInvalidToken
	}
	if err != nil {
		return nil, fmt.Errorf("failed to verify refresh token: %w", err)
	}

	var user models.SystemUser
	err = s.db.QueryRow(ctx, `SELECT id, email, username, role, status FROM system_users WHERE id = $1`, userID).Scan(&user.ID, &user.Email, &user.Username, &user.Role, &user.Status)
	if err != nil {
		return nil, ErrUserNotFound
	}

	s.db.Exec(ctx, `DELETE FROM refresh_tokens WHERE token_hash = $1`, tokenHash)

	accessToken, _ := s.generateAccessToken(user)
	newRefreshToken, _ := s.generateRefreshToken(ctx, user.ID)

	return &models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int64(s.cfg.ExpirationTime.Seconds()),
		User:         &user,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID) error {
	s.db.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_id = $1`, userID)
	return nil
}

func (s *AuthService) ValidateToken(ctx context.Context, tokenString string) (*models.SystemUser, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.cfg.Secret), nil
	})

	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	userIDStr, ok := claims["sub"].(string)
	if !ok {
		return nil, ErrInvalidToken
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, ErrInvalidToken
	}

	var user models.SystemUser
	err = s.db.QueryRow(ctx, `SELECT id, email, username, role, status FROM system_users WHERE id = $1`, userID).Scan(&user.ID, &user.Email, &user.Username, &user.Role, &user.Status)
	if err == pgx.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func (s *AuthService) generateAccessToken(user models.SystemUser) (string, error) {
	claims := jwt.MapClaims{
		"sub":  user.ID.String(),
		"email": user.Email,
		"role": user.Role,
		"exp":  time.Now().Add(s.cfg.ExpirationTime).Unix(),
		"iat":  time.Now().Unix(),
		"iss":  s.cfg.Issuer,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.Secret))
}

func (s *AuthService) generateRefreshToken(ctx context.Context, userID uuid.UUID) (string, error) {
	refreshToken := generateRandomToken(64)
	tokenHash := hashToken(refreshToken)
	expiresAt := time.Now().Add(s.cfg.RefreshExpires)

	s.db.Exec(ctx, `INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`, userID, tokenHash, expiresAt)

	return refreshToken, nil
}

// System Service
type SystemService struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewSystemService(db *pgxpool.Pool, redis *redis.Client) *SystemService {
	return &SystemService{db: db, redis: redis}
}

func (s *SystemService) GetInfo() (*models.SystemInfo, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	hostname, _ := os.Hostname()
	
	info := &models.SystemInfo{
		Hostname:    hostname,
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		CPUCores:    runtime.NumCPU(),
		MemoryTotal: m.TotalAlloc,
		Uptime:      time.Now().Unix(),
		LoadAverage: []float64{0.0, 0.0, 0.0},
	}

	return info, nil
}

func (s *SystemService) GetStatus() (*models.SystemStatus, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	status := &models.SystemStatus{
		Status:      "running",
		CPU:         0.0,
		Memory:      float64(m.Alloc) / float64(m.Sys) * 100,
		Disk:        0.0,
		Network:     make(map[string]int64),
		Processes:   0,
		Connections: 0,
	}

	return status, nil
}

func (s *SystemService) Restart() error {
	cmd := exec.Command("shutdown", "-r", "now")
	return cmd.Run()
}

func (s *SystemService) Shutdown() error {
	cmd := exec.Command("shutdown", "-h", "now")
	return cmd.Run()
}

// Monitoring Service
type MonitoringService struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewMonitoringService(db *pgxpool.Pool, redis *redis.Client) *MonitoringService {
	return &MonitoringService{db: db, redis: redis}
}

func (s *MonitoringService) GetMetrics() (map[string]interface{}, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics := map[string]interface{}{
		"goroutines":    runtime.NumGoroutine(),
		"memory_alloc":  m.Alloc,
		"memory_total":  m.TotalAlloc,
		"memory_sys":    m.Sys,
		"gc_runs":       m.NumGC,
		"cpu_count":     runtime.NumCPU(),
	}

	return metrics, nil
}

func (s *MonitoringService) GetResources() ([]models.MonitoringData, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	data := []models.MonitoringData{
		{ResourceType: "memory", MetricName: "alloc", Value: float64(m.Alloc), Unit: ptr("bytes"), RecordedAt: time.Now()},
		{ResourceType: "memory", MetricName: "total", Value: float64(m.TotalAlloc), Unit: ptr("bytes"), RecordedAt: time.Now()},
		{ResourceType: "memory", MetricName: "sys", Value: float64(m.Sys), Unit: ptr("bytes"), RecordedAt: time.Now()},
		{ResourceType: "cpu", MetricName: "goroutines", Value: float64(runtime.NumGoroutine()), Unit: ptr("count"), RecordedAt: time.Now()},
		{ResourceType: "cpu", MetricName: "cpu_count", Value: float64(runtime.NumCPU()), Unit: ptr("count"), RecordedAt: time.Now()},
	}

	return data, nil
}

func (s *MonitoringService) GetProcesses() ([]models.ProcessInfo, error) {
	return []models.ProcessInfo{
		{PID: os.Getpid(), Name: "admin_system", CPUPercent: ptr(0.0), MemoryPercent: ptr(0.0), Status: ptr("running"), RecordedAt: time.Now()},
	}, nil
}

func (s *MonitoringService) GetNetworkStats() ([]models.NetworkStats, error) {
	return []models.NetworkStats{
		{InterfaceName: ptr("eth0"), BytesSent: ptr(int64(0)), BytesReceived: ptr(int64(0)), RecordedAt: time.Now()},
	}, nil
}

// Config Service
type ConfigService struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewConfigService(db *pgxpool.Pool, redis *redis.Client) *ConfigService {
	return &ConfigService{db: db, redis: redis}
}

func (s *ConfigService) GetAll(ctx context.Context) ([]models.SystemConfig, error) {
	rows, err := s.db.Query(ctx, `SELECT key, value, value_type, description, is_secret, updated_at FROM system_config`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []models.SystemConfig
	for rows.Next() {
		var c models.SystemConfig
		rows.Scan(&c.Key, &c.Value, &c.ValueType, &c.Description, &c.IsSecret, &c.UpdatedAt)
		configs = append(configs, c)
	}
	return configs, nil
}

func (s *ConfigService) Get(ctx context.Context, key string) (*models.SystemConfig, error) {
	var c models.SystemConfig
	err := s.db.QueryRow(ctx, `SELECT key, value, value_type, description, is_secret, updated_at FROM system_config WHERE key = $1`, key).Scan(&c.Key, &c.Value, &c.ValueType, &c.Description, &c.IsSecret, &c.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("config not found")
	}
	return &c, err
}

func (s *ConfigService) Update(ctx context.Context, key, value string, userID uuid.UUID) error {
	_, err := s.db.Exec(ctx, `INSERT INTO system_config (key, value, updated_by, updated_at) VALUES ($1, $2, $3, NOW()) ON CONFLICT(key) DO UPDATE SET value = $2, updated_by = $3, updated_at = NOW()`, key, value, userID)
	return err
}

func (s *ConfigService) Delete(ctx context.Context, key string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM system_config WHERE key = $1`, key)
	return err
}

// Backup Service
type BackupService struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewBackupService(db *pgxpool.Pool, redis *redis.Client) *BackupService {
	return &BackupService{db: db, redis: redis}
}

func (s *BackupService) List(ctx context.Context, params *models.PaginationParams) (*models.PaginatedResponse, error) {
	rows, err := s.db.Query(ctx, `SELECT id, name, type, file_path, file_size, status, created_at, completed_at FROM system_backups ORDER BY created_at DESC LIMIT $1 OFFSET $2`, params.PageSize, (params.Page-1)*params.PageSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var backups []models.SystemBackup
	for rows.Next() {
		var b models.SystemBackup
		rows.Scan(&b.ID, &b.Name, &b.Type, &b.FilePath, &b.FileSize, &b.Status, &b.CreatedAt, &b.CompletedAt)
		backups = append(backups, b)
	}

	return &models.PaginatedResponse{Total: len(backups), Page: params.Page, PageSize: params.PageSize, Data: backups}, nil
}

func (s *BackupService) Create(ctx context.Context, name, backupType string, userID uuid.UUID) (*models.SystemBackup, error) {
	var b models.SystemBackup
	err := s.db.QueryRow(ctx, `INSERT INTO system_backups (name, type, status, created_by) VALUES ($1, $2, 'in_progress', $3) RETURNING id, name, type, status, created_at`, name, backupType, userID).Scan(&b.ID, &b.Name, &b.Type, &b.Status, &b.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *BackupService) Get(ctx context.Context, id uuid.UUID) (*models.SystemBackup, error) {
	var b models.SystemBackup
	err := s.db.QueryRow(ctx, `SELECT id, name, type, file_path, file_size, status, created_at, completed_at FROM system_backups WHERE id = $1`, id).Scan(&b.ID, &b.Name, &b.Type, &b.FilePath, &b.FileSize, &b.Status, &b.CreatedAt, &b.CompletedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("backup not found")
	}
	return &b, err
}

func (s *BackupService) Restore(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, `UPDATE system_backups SET status = 'restoring' WHERE id = $1`, id)
	return err
}

func (s *BackupService) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, `DELETE FROM system_backups WHERE id = $1`, id)
	return err
}

// Log Service
type LogService struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewLogService(db *pgxpool.Pool, redis *redis.Client) *LogService {
	return &LogService{db: db, redis: redis}
}

func (s *LogService) List(ctx context.Context, params *models.PaginationParams, level string) (*models.PaginatedResponse, error) {
	query := `SELECT id, level, message, component, user_id, ip_address, created_at FROM system_logs`
	args := []interface{}{}
	argIndex := 1

	if level != "" {
		query += fmt.Sprintf(" WHERE level = $%d", argIndex)
		args = append(args, level)
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, params.PageSize, (params.Page-1)*params.PageSize)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.SystemLog
	for rows.Next() {
		var l models.SystemLog
		rows.Scan(&l.ID, &l.Level, &l.Message, &l.Component, &l.UserID, &l.IPAddress, &l.CreatedAt)
		logs = append(logs, l)
	}

	return &models.PaginatedResponse{Total: len(logs), Page: params.Page, PageSize: params.PageSize, Data: logs}, nil
}

func (s *LogService) Get(ctx context.Context, id uuid.UUID) (*models.SystemLog, error) {
	var l models.SystemLog
	err := s.db.QueryRow(ctx, `SELECT id, level, message, component, user_id, ip_address, created_at FROM system_logs WHERE id = $1`, id).Scan(&l.ID, &l.Level, &l.Message, &l.Component, &l.UserID, &l.IPAddress, &l.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("log not found")
	}
	return &l, err
}

func (s *LogService) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, `DELETE FROM system_logs WHERE id = $1`, id)
	return err
}

func (s *LogService) DeleteOld(ctx context.Context, days int) error {
	_, err := s.db.Exec(ctx, `DELETE FROM system_logs WHERE created_at < NOW() - INTERVAL '1 day' * $1`, days)
	return err
}

// Metrics Service
type MetricsService struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewMetricsService(db *pgxpool.Pool, redis *redis.Client) *MetricsService {
	return &MetricsService{db: db, redis: redis}
}

func (s *MetricsService) Get(ctx context.Context) ([]models.SystemMetric, error) {
	rows, err := s.db.Query(ctx, `SELECT id, metric_type, metric_name, value, unit, recorded_at FROM system_metrics ORDER BY recorded_at DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []models.SystemMetric
	for rows.Next() {
		var m models.SystemMetric
		rows.Scan(&m.ID, &m.MetricType, &m.MetricName, &m.Value, &m.Unit, &m.RecordedAt)
		metrics = append(metrics, m)
	}
	return metrics, nil
}

func (s *MetricsService) GetByType(ctx context.Context, metricType string) ([]models.SystemMetric, error) {
	rows, err := s.db.Query(ctx, `SELECT id, metric_type, metric_name, value, unit, recorded_at FROM system_metrics WHERE metric_type = $1 ORDER BY recorded_at DESC LIMIT 100`, metricType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []models.SystemMetric
	for rows.Next() {
		var m models.SystemMetric
		rows.Scan(&m.ID, &m.MetricType, &m.MetricName, &m.Value, &m.Unit, &m.RecordedAt)
		metrics = append(metrics, m)
	}
	return metrics, nil
}

// Alert Service
type AlertService struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewAlertService(db *pgxpool.Pool, redis *redis.Client) *AlertService {
	return &AlertService{db: db, redis: redis}
}

func (s *AlertService) List(ctx context.Context, params *models.PaginationParams, status string) (*models.PaginatedResponse, error) {
	query := `SELECT id, title, message, severity, status, source, created_at FROM system_alerts`
	args := []interface{}{}
	argIndex := 1

	if status != "" {
		query += fmt.Sprintf(" WHERE status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, params.PageSize, (params.Page-1)*params.PageSize)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []models.SystemAlert
	for rows.Next() {
		var a models.SystemAlert
		rows.Scan(&a.ID, &a.Title, &a.Message, &a.Severity, &a.Status, &a.Source, &a.CreatedAt)
		alerts = append(alerts, a)
	}

	return &models.PaginatedResponse{Total: len(alerts), Page: params.Page, PageSize: params.PageSize, Data: alerts}, nil
}

func (s *AlertService) Create(ctx context.Context, alert *models.SystemAlert) (*models.SystemAlert, error) {
	err := s.db.QueryRow(ctx, `INSERT INTO system_alerts (title, message, severity, status, source) VALUES ($1, $2, $3, 'active', $4) RETURNING id, created_at`, alert.Title, alert.Message, alert.Severity, alert.Source).Scan(&alert.ID, &alert.CreatedAt)
	return alert, err
}

func (s *AlertService) Acknowledge(ctx context.Context, id, userID uuid.UUID) error {
	_, err := s.db.Exec(ctx, `UPDATE system_alerts SET status = 'acknowledged', acknowledged_by = $1, acknowledged_at = NOW() WHERE id = $2`, userID, id)
	return err
}

func (s *AlertService) Resolve(ctx context.Context, id, userID uuid.UUID) error {
	_, err := s.db.Exec(ctx, `UPDATE system_alerts SET status = 'resolved', resolved_by = $1, resolved_at = NOW() WHERE id = $2`, userID, id)
	return err
}

// Helper functions
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func generateRandomToken(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)[:length]
}

func ptr[T any](v T) *T {
	return &v
}
