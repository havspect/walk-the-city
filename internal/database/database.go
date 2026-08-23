package database

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/glebarez/sqlite"
	"github.com/havspect/walk-the-city/internal/trip"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open initializes a SQLite database connection at the specified file path,
// ensuring the parent directory exists, configuring WAL mode and busy timeouts,
// and executing schema migrations.
func Open(dbPath string) (*gorm.DB, error) {
	if dbPath == "" {
		return nil, fmt.Errorf("database path cannot be empty")
	}

	// For non-memory databases, ensure the storage directory exists.
	if !isMemoryPath(dbPath) {
		dir := filepath.Dir(dbPath)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create database directory %q: %w", dir, err)
			}
		}
	}

	dsn := formatDSN(dbPath)
	return OpenDSN(dsn)
}

// OpenDSN opens a GORM connection using a raw SQLite DSN and performs auto-migration.
func OpenDSN(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enforce FK constraints for cascade deletes (SQLite defaults OFF per connection).
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	if err := db.AutoMigrate(&trip.Trip{}, &trip.Itinerary{}, &trip.Stop{}, &trip.Segment{}); err != nil {
		return nil, fmt.Errorf("failed to auto-migrate database schema: %w", err)
	}

	return db, nil
}

func isMemoryPath(path string) bool {
	return path == ":memory:" || strings.HasPrefix(path, "file::memory:") || strings.Contains(path, "mode=memory")
}

func formatDSN(dbPath string) string {
	fk := "_pragma=foreign_keys(1)"
	wal := "_pragma=journal_mode(WAL)"
	busy := "_pragma=busy_timeout(5000)"
	if isMemoryPath(dbPath) {
		if strings.Contains(dbPath, "?") {
			return dbPath + "&" + busy + "&" + wal + "&" + fk
		}
		return dbPath + "?" + busy + "&" + wal + "&" + fk
	}
	return fmt.Sprintf("file:%s?%s&%s&%s", dbPath, busy, wal, fk)
}
