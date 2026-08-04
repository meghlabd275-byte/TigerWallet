package database

import (
	"context"
	"fmt"
	"time"

	"admin_backend/internal/config"
	"admin_backend/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// PostgresDB holds the PostgreSQL database connection
type PostgresDB struct {
	DB *gorm.DB
}

// NewPostgresDB creates a new PostgreSQL database connection
func NewPostgresDB(cfg *config.Config) (*PostgresDB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
	)

	// Custom logger
	customLogger := logger.New(
		fmtLogger{},
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  getLogLevel(cfg.LogLevel),
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: customLogger,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.DBMaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.DBMaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.DBMaxLifetime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Auto migrate
	if err := autoMigrate(db); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return &PostgresDB{DB: db}, nil
}

// autoMigrate runs database migrations
func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.Admin{},
		&models.AdminSession{},
		&models.AdminActivity{},
		&models.User{},
		&models.Transaction{},
		&models.Token{},
		&models.KYCApplication{},
		&models.Withdrawal{},
		&models.WhiteLabel{},
		&models.SystemConfig{},
		&models.APIKey{},
		&models.AuditLog{},
		&models.Notification{},
	)
}

// getLogLevel converts string log level to gorm log level
func getLogLevel(level string) logger.LogLevel {
	switch level {
	case "debug":
		return logger.Debug
	case "warn":
		return logger.Warn
	case "error":
		return logger.Error
	default:
		return logger.Info
	}
}

// fmtLogger implements gorm's logger interface
type fmtLogger struct{}

func (fmtLogger) Print(v ...interface{}) {
	fmt.Println(v...)
}

// Close closes the database connection
func (p *PostgresDB) Close() error {
	sqlDB, err := p.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Begin starts a new transaction
func (p *PostgresDB) Begin() *gorm.DB {
	return p.DB.Begin()
}

// Transaction executes a function within a transaction
func (p *PostgresDB) Transaction(fn func(*gorm.DB) error) error {
	return p.DB.Transaction(fn)
}

// Raw executes raw SQL
func (p *PostgresDB) Raw(sql string, values ...interface{}) *gorm.DB {
	return p.DB.Raw(sql, values...)
}

// Exec executes raw SQL
func (p *PostgresDB) Exec(sql string, values ...interface{}) *gorm.DB {
	return p.DB.Exec(sql, values...)
}

// Create creates a new record
func (p *PostgresDB) Create(value interface{}) *gorm.DB {
	return p.DB.Create(value)
}

// First retrieves the first record
func (p *PostgresDB) First(dest interface{}, conditions ...interface{}) *gorm.DB {
	return p.DB.First(dest, conditions...)
}

// Where adds where conditions
func (p *PostgresDB) Where(query interface{}, args ...interface{}) *gorm.DB {
	return p.DB.Where(query, args...)
}

// Find retrieves records matching conditions
func (p *PostgresDB) Find(dest interface{}, conditions ...interface{}) *gorm.DB {
	return p.DB.Find(dest, conditions...)
}

// Model changes the current model
func (p *PostgresDB) Model(model interface{}) *gorm.DB {
	return p.DB.Model(model)
}

// Updates updates records
func (p *PostgresDB) Updates(values interface{}) *gorm.DB {
	return p.DB.Updates(values)
}

// Delete deletes records
func (p *PostgresDB) Delete(model interface{}, conditions ...interface{}) *gorm.DB {
	return p.DB.Delete(model, conditions...)
}

// Count counts records
func (p *PostgresDB) Count(count *int64) *gorm.DB {
	return p.DB.Count(count)
}

// Preloads preloads relationships
func (p *PostgresDB) Preload(name string, conditions ...interface{}) *gorm.DB {
	return p.DB.Preload(name, conditions...)
}

// Joins joins with relationships
func (p *PostgresDB) Joins(query string, args ...interface{}) *gorm.DB {
	return p.DB.Joins(query, args...)
}

// Scopes adds scopes
func (p *PostgresDB) Scopes(scopes ...func(*gorm.DB) *gorm.DB) *gorm.DB {
	return p.DB.Scopes(scopes...)
}
