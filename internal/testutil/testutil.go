// Package testutil provides shared helpers for tests.
package testutil

import (
	"testing"
	"yunyoumanager/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewDB returns an in-memory SQLite DB with the schema migrated.
// The DB is closed when the test ends.
func NewDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	if err := db.AutoMigrate(&model.Member{}, &model.DailyRecord{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_daily_member_date ON daily_records(member_id, date)`)
	return db
}
