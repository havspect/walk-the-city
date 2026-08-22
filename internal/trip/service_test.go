package trip

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:memtrip_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open test sqlite: %v", err)
	}

	if err := db.AutoMigrate(&Trip{}); err != nil {
		t.Fatalf("failed to auto-migrate: %v", err)
	}
	return db
}

func TestTripService_CreateTrip(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// Happy path
	params := CreateTripParams{
		Destination:  "Paris",
		DurationDays: 4,
		Notes:        "Eiffel Tower, Louvre, Montmartre",
	}

	created, err := svc.CreateTrip(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error creating trip: %v", err)
	}

	if created.ID == 0 {
		t.Errorf("expected non-zero ID, got 0")
	}
	if created.Destination != "Paris" || created.DurationDays != 4 {
		t.Errorf("unexpected trip fields: %+v", created)
	}

	// Validation errors
	invalidParams := []struct {
		name        string
		params      CreateTripParams
		expectField string
	}{
		{
			name:        "empty destination",
			params:      CreateTripParams{Destination: "", DurationDays: 3},
			expectField: "destination",
		},
		{
			name:        "whitespace destination",
			params:      CreateTripParams{Destination: "   ", DurationDays: 3},
			expectField: "destination",
		},
		{
			name:        "duration zero",
			params:      CreateTripParams{Destination: "Berlin", DurationDays: 0},
			expectField: "duration_days",
		},
		{
			name:        "duration too large",
			params:      CreateTripParams{Destination: "Berlin", DurationDays: 31},
			expectField: "duration_days",
		},
	}

	for _, tt := range invalidParams {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.CreateTrip(ctx, tt.params)
			if err == nil {
				t.Fatalf("expected validation error, got nil")
			}
			var valErr *ValidationError
			if !errors.As(err, &valErr) {
				t.Fatalf("expected *ValidationError, got %T: %v", err, err)
			}
			if _, exists := valErr.FieldErrors[tt.expectField]; !exists {
				t.Errorf("expected error field %q, got: %+v", tt.expectField, valErr.FieldErrors)
			}
		})
	}
}

func TestTripService_ListAndGetTrips(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// Empty list
	trips, err := svc.ListTrips(ctx)
	if err != nil {
		t.Fatalf("unexpected error on empty list: %v", err)
	}
	if len(trips) != 0 {
		t.Errorf("expected 0 trips, got %d", len(trips))
	}

	// Insert trips
	t1, err := svc.CreateTrip(ctx, CreateTripParams{Destination: "London", DurationDays: 2})
	if err != nil {
		t.Fatalf("failed creating trip 1: %v", err)
	}
	time.Sleep(10 * time.Millisecond) // slight delay for ordering test
	t2, err := svc.CreateTrip(ctx, CreateTripParams{Destination: "Rome", DurationDays: 5})
	if err != nil {
		t.Fatalf("failed creating trip 2: %v", err)
	}

	// List trips should return in reverse chronological order
	trips, err = svc.ListTrips(ctx)
	if err != nil {
		t.Fatalf("failed listing trips: %v", err)
	}
	if len(trips) != 2 {
		t.Fatalf("expected 2 trips, got %d", len(trips))
	}
	if trips[0].ID != t2.ID || trips[1].ID != t1.ID {
		t.Errorf("expected trips ordered by created_at DESC (Rome first, then London)")
	}

	// Get by ID success
	found, err := svc.GetTripByID(ctx, t1.ID)
	if err != nil {
		t.Fatalf("failed getting trip %d: %v", t1.ID, err)
	}
	if found.Destination != "London" {
		t.Errorf("expected destination London, got %q", found.Destination)
	}

	// Get by ID not found
	_, err = svc.GetTripByID(ctx, 99999)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}
