// Migration test — the launchpad service migrates four models during startup,
// so the migration must succeed both when the tables are new and when they
// already exist. The already-exists path is the one every restart takes, and
// it failed outright with gorm driver v1.5.4.
package launchpad

import (
	"os"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// migrateAll mirrors the AutoMigrate call in NewLaunchpadService.
func migrateAll(db *gorm.DB) error {
	return db.AutoMigrate(
		&LaunchpadProject{},
		&LaunchpoolProject{},
		&Allocation{},
		&Stake{},
	)
}

func TestMigrateIsIdempotent(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL/DATABASE_URL not set")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:  logger.Default.LogMode(logger.Silent),
		NowFunc: func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		t.Skipf("database unavailable: %v", err)
	}

	// A dedicated schema keeps the test independent of any other service that
	// shares the database, and guarantees the first migrate creates the tables.
	schema := "launchpadmig_" + time.Now().UTC().Format("20060102150405")
	if err := db.Exec("CREATE SCHEMA IF NOT EXISTS " + schema).Error; err != nil {
		t.Skipf("cannot create scratch schema: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Exec("DROP SCHEMA IF EXISTS " + schema + " CASCADE").Error
	})
	if err := db.Exec("SET search_path TO " + schema).Error; err != nil {
		t.Fatalf("set search_path: %v", err)
	}

	if err := migrateAll(db); err != nil {
		t.Fatalf("first migrate (creates tables) failed: %v", err)
	}
	if err := migrateAll(db); err != nil {
		t.Fatalf("second migrate (tables already exist) failed: %v", err)
	}
}
