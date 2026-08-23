package trip

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	if err := db.AutoMigrate(&Trip{}, &Itinerary{}, &Stop{}, &Segment{}); err != nil {
		t.Fatalf("failed to auto-migrate: %v", err)
	}
	return db
}

func TestTripService_CreateTrip(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

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
		{name: "empty destination", params: CreateTripParams{Destination: "", DurationDays: 3}, expectField: "destination"},
		{name: "whitespace destination", params: CreateTripParams{Destination: "   ", DurationDays: 3}, expectField: "destination"},
		{name: "duration zero", params: CreateTripParams{Destination: "Berlin", DurationDays: 0}, expectField: "duration_days"},
		{name: "duration too large", params: CreateTripParams{Destination: "Berlin", DurationDays: 31}, expectField: "duration_days"},
		{name: "notes too long", params: CreateTripParams{Destination: "Rome", DurationDays: 3, Notes: strings.Repeat("A", 2001)}, expectField: "notes"},
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

func TestTripService_CreateTrip_WithItineraries(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	gen := NewMockGenerator()
	itins, _ := gen.Generate(ctx, CreateTripParams{Destination: "Rome", City: "Rome", DurationDays: 2})

	created, err := svc.CreateTrip(ctx, CreateTripParams{
		Destination:  "Rome, Italy",
		City:         "Rome",
		DurationDays: 2,
		Itineraries:  itins,
	})
	if err != nil {
		t.Fatalf("create with itineraries: %v", err)
	}
	if len(created.Itineraries) != 2 {
		t.Fatalf("expected 2 itineraries, got %d", len(created.Itineraries))
	}
	for _, itin := range created.Itineraries {
		if len(itin.Stops) == 0 {
			t.Errorf("itinerary %q: expected stops", itin.Title)
		}
		if len(itin.Segments) != len(itin.Stops)-1 {
			t.Errorf("itinerary %q: N-1 broken: %d stops %d segs", itin.Title, len(itin.Stops), len(itin.Segments))
		}
		for _, st := range itin.Stops {
			if st.ImageURL == "" {
				t.Errorf("stop %q: image url empty", st.Title)
			}
		}
	}
	// Get by ID round-trips
	loaded, err := svc.GetTripByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if len(loaded.Itineraries) != 2 {
		t.Fatalf("preload itineraries: got %d", len(loaded.Itineraries))
	}
	if len(loaded.Itineraries[0].Stops) != len(created.Itineraries[0].Stops) {
		t.Errorf("stops count mismatch after reload")
	}
}

func TestTripService_ListAndGetTrips(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	trips, err := svc.ListTrips(ctx)
	if err != nil {
		t.Fatalf("unexpected error on empty list: %v", err)
	}
	if len(trips) != 0 {
		t.Errorf("expected 0 trips, got %d", len(trips))
	}

	t1, err := svc.CreateTrip(ctx, CreateTripParams{Destination: "London", DurationDays: 2})
	if err != nil {
		t.Fatalf("failed creating trip 1: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	t2, err := svc.CreateTrip(ctx, CreateTripParams{Destination: "Rome", DurationDays: 5})
	if err != nil {
		t.Fatalf("failed creating trip 2: %v", err)
	}

	trips, err = svc.ListTrips(ctx)
	if err != nil {
		t.Fatalf("failed listing trips: %v", err)
	}
	if len(trips) != 2 {
		t.Fatalf("expected 2 trips, got %d", len(trips))
	}
	if trips[0].ID != t2.ID || trips[1].ID != t1.ID {
		t.Errorf("expected trips ordered by created_at DESC")
	}

	found, err := svc.GetTripByID(ctx, t1.ID)
	if err != nil {
		t.Fatalf("failed getting trip %d: %v", t1.ID, err)
	}
	if found.Destination != "London" {
		t.Errorf("expected destination London, got %q", found.Destination)
	}

	_, err = svc.GetTripByID(ctx, 99999)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestTripService_GenerateAndSaveTrip(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()
	gen := NewMockGenerator()

	trip, err := svc.GenerateAndSaveTrip(ctx, CreateTripParams{
		Destination:  "Rome, Italy",
		City:         "Rome",
		DurationDays: 3,
		Month:        "September",
		Mobility:     "Walking + Public Transit",
	}, gen)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(trip.Itineraries) != 2 {
		t.Fatalf("expected 2 itineraries from mock, got %d", len(trip.Itineraries))
	}
}

func TestTripService_CreateTrip_ValidationItineraries(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// Invalid stop kind
	_, err := svc.CreateTrip(ctx, CreateTripParams{
		Destination:  "Rome",
		DurationDays: 2,
		Itineraries: []*Itinerary{
			{
				Title: "T", Stops: []Stop{{Kind: "Bogus", Title: "S", Body: "b", ImageURL: "https://example.com/x.jpg"}},
			},
		},
	})
	if err == nil {
		t.Fatalf("expected validation error for bogus kind")
	}
	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	// Wrong segment count (N-1 invariant)
	_, err = svc.CreateTrip(ctx, CreateTripParams{
		Destination:  "Rome",
		DurationDays: 2,
		Itineraries: []*Itinerary{
			{
				Title: "T",
				Stops: []Stop{
					{Kind: StopKindLandmark, Title: "S1", Body: "b", ImageURL: "https://example.com/a.jpg"},
					{Kind: StopKindLandmark, Title: "S2", Body: "b", ImageURL: "https://example.com/b.jpg"},
				},
				Segments: []Segment{{Mode: SegmentModeWalk, DistanceMeters: 100, DurationMinutes: 5, Instruction: "x"}, {Mode: SegmentModeWalk, DistanceMeters: 100, DurationMinutes: 5, Instruction: "y"}}, // want 1, got 2
			},
		},
	})
	if err == nil {
		t.Fatalf("expected validation error for segment count")
	}
	if !errors.As(err, &valErr) {
		t.Fatalf("expected ValidationError for segment count")
	}

	// Ensure nothing was written via unscoped count
	var count int64
	db.Unscoped().Model(&Trip{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 trips after validation failures, got %d", count)
	}
}

func TestTripService_CreateTrip_Rollback(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// Inject a failing create callback for Segment to force mid-transaction rollback
	cbName := "test_fail_segment_create"
	err := db.Callback().Create().Before("gorm:create").Register(cbName, func(tx *gorm.DB) {
		if tx.Statement.Model != nil {
			if _, ok := tx.Statement.Model.(*Segment); ok {
				tx.AddError(errors.New("injected failure"))
			}
			// Also handle non-pointer model type
			if tx.Statement.ReflectValue.IsValid() {
				if tx.Statement.ReflectValue.Type().Name() == "Segment" {
					tx.AddError(errors.New("injected failure"))
				}
			}
		}
	})
	if err != nil {
		t.Fatalf("register callback: %v", err)
	}
	t.Cleanup(func() {
		db.Callback().Create().Remove(cbName)
	})

	gen := NewMockGenerator()
	itins, _ := gen.Generate(ctx, CreateTripParams{Destination: "Rome", City: "Rome", DurationDays: 2})
	_, err = svc.CreateTrip(ctx, CreateTripParams{
		Destination:  "Rome",
		DurationDays: 2,
		Itineraries:  itins,
	})
	if err == nil {
		t.Fatalf("expected injected error")
	}
	// Rollback: no trip should remain, even via unscoped count
	var count int64
	db.Unscoped().Model(&Trip{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 trips after rollback, got %d", count)
	}
	db.Unscoped().Model(&Itinerary{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 itineraries after rollback, got %d", count)
	}
}

func TestTripService_CreateTrip_PlaceholderViaHook(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	created, err := svc.CreateTrip(ctx, CreateTripParams{
		Destination:  "Rome",
		DurationDays: 1,
		Itineraries: []*Itinerary{
			{
				Title: "T", Stops: []Stop{{Kind: StopKindLandmark, Title: "Empty Image", Body: "body", ImageURL: ""}},
			},
		},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(created.Itineraries) == 0 || len(created.Itineraries[0].Stops) == 0 {
		t.Fatalf("expected stops")
	}
	st := created.Itineraries[0].Stops[0]
	if st.ImageURL == "" {
		t.Errorf("expected placeholder ImageURL via hook")
	}
	if st.ImageSource != ImageSourcePlaceholder {
		t.Errorf("expected placeholder source, got %q", st.ImageSource)
	}
}

func TestTripService_GenerateAndSaveTrip_NilGenerator(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	created, err := svc.GenerateAndSaveTrip(ctx, CreateTripParams{
		Destination:  "Paris",
		DurationDays: 2,
	}, nil)
	if err != nil {
		t.Fatalf("nil generator: %v", err)
	}
	if created.Destination != "Paris" {
		t.Errorf("expected Paris")
	}
	if len(created.Itineraries) != 0 {
		t.Errorf("nil generator should produce no itineraries")
	}
}
