package database

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/havspect/walk-the-city/internal/trip"
)

func TestOpen_InMemory(t *testing.T) {
	dsn := fmt.Sprintf("file:memdb_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := OpenDSN(dsn)
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}

	// Verify table was created and can insert/query
	testTrip := trip.Trip{
		Destination:  "Tokyo",
		DurationDays: 5,
		Notes:        "Shinjuku, Shibuya, Asakusa",
	}

	if err := db.Create(&testTrip).Error; err != nil {
		t.Fatalf("failed to insert trip: %v", err)
	}

	if testTrip.ID == 0 {
		t.Errorf("expected auto-incremented ID, got 0")
	}

	var found trip.Trip
	if err := db.First(&found, testTrip.ID).Error; err != nil {
		t.Fatalf("failed to retrieve trip: %v", err)
	}

	if found.Destination != "Tokyo" || found.DurationDays != 5 {
		t.Errorf("unexpected trip data: %+v", found)
	}
}

func TestOpen_DirectoryAutoCreation(t *testing.T) {
	tempDir := t.TempDir()
	nestedPath := filepath.Join(tempDir, "nested", "storage", "test.db")

	db, err := Open(nestedPath)
	if err != nil {
		t.Fatalf("failed to open database with nested path: %v", err)
	}

	// Verify directory exists
	parentDir := filepath.Dir(nestedPath)
	if stat, err := os.Stat(parentDir); err != nil || !stat.IsDir() {
		t.Errorf("expected directory %q to be created, stat error: %v", parentDir, err)
	}

	// Verify we can write to it
	tRecord := trip.Trip{Destination: "Rome", DurationDays: 3}
	if err := db.Create(&tRecord).Error; err != nil {
		t.Fatalf("failed to write record: %v", err)
	}
}
