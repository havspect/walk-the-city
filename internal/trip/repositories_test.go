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

func setupTripDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:memtrip_repo_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Exec("PRAGMA foreign_keys=ON").Error; err != nil {
		t.Fatalf("pragma: %v", err)
	}
	if err := db.AutoMigrate(&Trip{}, &Itinerary{}, &Stop{}, &Segment{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestTripRepository_GetAndList(t *testing.T) {
	db := setupTripDB(t)
	tripRepo := NewTripRepository(db)
	itineraryRepo := NewItineraryRepository(db)
	stopRepo := NewStopRepository(db)
	segmentRepo := NewSegmentRepository(db)
	ctx := context.Background()

	// Empty list
	trips, err := tripRepo.ListTrips(ctx)
	if err != nil {
		t.Fatalf("list empty: %v", err)
	}
	if len(trips) != 0 {
		t.Fatalf("expected 0 trips, got %d", len(trips))
	}

	// Create two trips via repository Create (low-level)
	t1 := &Trip{Destination: "London", DurationDays: 2}
	if err := tripRepo.Create(ctx, t1); err != nil {
		t.Fatalf("create t1: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	t2 := &Trip{Destination: "Rome", DurationDays: 5}
	if err := tripRepo.Create(ctx, t2); err != nil {
		t.Fatalf("create t2: %v", err)
	}

	// Add itineraries out of order to test preload ordering
	itin2 := &Itinerary{TripID: t1.ID, Title: "B", Position: 1}
	itin1 := &Itinerary{TripID: t1.ID, Title: "A", Position: 0}
	if err := itineraryRepo.Create(ctx, itin2); err != nil {
		t.Fatalf("create itin2: %v", err)
	}
	if err := itineraryRepo.Create(ctx, itin1); err != nil {
		t.Fatalf("create itin1: %v", err)
	}
	// Stops out of order
	s2 := &Stop{ItineraryID: itin1.ID, Position: 1, Kind: StopKindHistory, Title: "Forum", Body: "body", ImageURL: "https://example.com/b.jpg"}
	s1 := &Stop{ItineraryID: itin1.ID, Position: 0, Kind: StopKindLandmark, Title: "Colosseum", Body: "body", ImageURL: "https://example.com/a.jpg"}
	s3 := &Stop{ItineraryID: itin1.ID, Position: 2, Kind: StopKindFoodDrink, Title: "Trattoria", Body: "body", ImageURL: "https://example.com/c.jpg"}
	for _, s := range []*Stop{s2, s1, s3} {
		if err := stopRepo.Create(ctx, s); err != nil {
			t.Fatalf("create stop %q: %v", s.Title, err)
		}
	}
	// Segments out of order
	seg2 := &Segment{ItineraryID: itin1.ID, FromStopID: s1.ID, ToStopID: s2.ID, Position: 1, Mode: SegmentModeTransit, DistanceMeters: 800, DurationMinutes: 10, Instruction: "Transit"}
	seg1 := &Segment{ItineraryID: itin1.ID, FromStopID: s1.ID, ToStopID: s1.ID, Position: 0, Mode: SegmentModeWalk, DistanceMeters: 500, DurationMinutes: 6, Instruction: "Walk"}
	// Correct seg1 endpoints to s1->s2, seg2 to s2->s3 after IDs known
	// Reload stops to get IDs ordered
	stops, _ := stopRepo.ListByParent(ctx, itin1.ID)
	// stops are ordered by position ASC: Colosseum, Forum, Trattoria
	if len(stops) != 3 {
		t.Fatalf("expected 3 stops, got %d", len(stops))
	}
	seg1.FromStopID = stops[0].ID
	seg1.ToStopID = stops[1].ID
	seg2.FromStopID = stops[1].ID
	seg2.ToStopID = stops[2].ID
	if err := segmentRepo.Create(ctx, seg2); err != nil {
		t.Fatalf("create seg2: %v", err)
	}
	if err := segmentRepo.Create(ctx, seg1); err != nil {
		t.Fatalf("create seg1: %v", err)
	}

	// GetTripByID should return ordered preloads
	loaded, err := tripRepo.GetTripByID(ctx, t1.ID)
	if err != nil {
		t.Fatalf("getByID: %v", err)
	}
	if len(loaded.Itineraries) != 2 {
		t.Fatalf("expected 2 itineraries, got %d", len(loaded.Itineraries))
	}
	if loaded.Itineraries[0].Title != "A" || loaded.Itineraries[1].Title != "B" {
		t.Errorf("itineraries not ordered by position: %+v", loaded.Itineraries)
	}
	if len(loaded.Itineraries[0].Stops) != 3 {
		t.Fatalf("expected 3 stops, got %d", len(loaded.Itineraries[0].Stops))
	}
	if loaded.Itineraries[0].Stops[0].Title != "Colosseum" {
		t.Errorf("stops not ordered: %q", loaded.Itineraries[0].Stops[0].Title)
	}
	if len(loaded.Itineraries[0].Segments) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(loaded.Itineraries[0].Segments))
	}
	if loaded.Itineraries[0].Segments[0].Position != 0 || loaded.Itineraries[0].Segments[1].Position != 1 {
		t.Errorf("segments not ordered")
	}

	// ListTrips ordered by created_at DESC
	list, err := tripRepo.ListTrips(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 trips, got %d", len(list))
	}
	if list[0].ID != t2.ID || list[1].ID != t1.ID {
		t.Errorf("expected DESC order")
	}
	// List also preloads
	if len(list[1].Itineraries) != 2 {
		t.Errorf("list should preload itineraries")
	}
}

func TestTripRepository_GetTripByID_NotFound(t *testing.T) {
	db := setupTripDB(t)
	tripRepo := NewTripRepository(db)
	_, err := tripRepo.GetTripByID(context.Background(), 99999)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestChildRepositories_ListAndCount(t *testing.T) {
	db := setupTripDB(t)
	tripRepo := NewTripRepository(db)
	itineraryRepo := NewItineraryRepository(db)
	stopRepo := NewStopRepository(db)
	segmentRepo := NewSegmentRepository(db)
	ctx := context.Background()

	trip := &Trip{Destination: "Test", DurationDays: 2}
	if err := tripRepo.Create(ctx, trip); err != nil {
		t.Fatalf("create trip: %v", err)
	}
	// Itinerary ListByTrip
	itins := []*Itinerary{
		{TripID: trip.ID, Title: "T1", Position: 1},
		{TripID: trip.ID, Title: "T0", Position: 0},
	}
	for _, it := range itins {
		if err := itineraryRepo.Create(ctx, it); err != nil {
			t.Fatalf("create itin: %v", err)
		}
	}
	list, err := itineraryRepo.ListByParent(ctx, trip.ID)
	if err != nil {
		t.Fatalf("list itineraries: %v", err)
	}
	if len(list) != 2 || list[0].Position != 0 {
		t.Errorf("itinerary order failed: %+v", list)
	}
	c, _ := itineraryRepo.CountByParent(ctx, trip.ID)
	if c != 2 {
		t.Errorf("count itineraries expected 2, got %d", c)
	}

	// Stop ListByItinerary
	itinID := list[0].ID
	stops := []*Stop{
		{ItineraryID: itinID, Position: 2, Kind: StopKindLandmark, Title: "C", Body: "b", ImageURL: "https://example.com/c.jpg"},
		{ItineraryID: itinID, Position: 0, Kind: StopKindLandmark, Title: "A", Body: "b", ImageURL: "https://example.com/a.jpg"},
		{ItineraryID: itinID, Position: 1, Kind: StopKindLandmark, Title: "B", Body: "b", ImageURL: "https://example.com/b.jpg"},
	}
	for _, s := range stops {
		if err := stopRepo.Create(ctx, s); err != nil {
			t.Fatalf("create stop: %v", err)
		}
	}
	slist, err := stopRepo.ListByParent(ctx, itinID)
	if err != nil {
		t.Fatalf("list stops: %v", err)
	}
	if len(slist) != 3 || slist[0].Position != 0 || slist[1].Position != 1 {
		t.Errorf("stop order failed")
	}
	c, _ = stopRepo.CountByParent(ctx, itinID)
	if c != 3 {
		t.Errorf("stop count expected 3, got %d", c)
	}

	// Segment ListByItinerary
	segs := []*Segment{
		{ItineraryID: itinID, FromStopID: slist[0].ID, ToStopID: slist[1].ID, Position: 1, Mode: SegmentModeWalk, DistanceMeters: 300, DurationMinutes: 4, Instruction: "s1"},
		{ItineraryID: itinID, FromStopID: slist[0].ID, ToStopID: slist[0].ID, Position: 0, Mode: SegmentModeWalk, DistanceMeters: 200, DurationMinutes: 3, Instruction: "s0"},
	}
	// fix seg positions: create in reverse order
	if err := segmentRepo.Create(ctx, segs[0]); err != nil {
		t.Fatalf("create seg: %v", err)
	}
	if err := segmentRepo.Create(ctx, segs[1]); err != nil {
		t.Fatalf("create seg2: %v", err)
	}
	segList, _ := segmentRepo.ListByParent(ctx, itinID)
	if len(segList) != 2 || segList[0].Position != 0 {
		t.Errorf("segment order failed")
	}
	c, _ = segmentRepo.CountByParent(ctx, itinID)
	if c != 2 {
		t.Errorf("segment count expected 2, got %d", c)
	}

	// Unknown parent returns empty
	empty, _ := stopRepo.ListByParent(ctx, 99999)
	if len(empty) != 0 {
		t.Errorf("expected empty for unknown parent")
	}
}

func TestStopRepository_PlaceholderHook(t *testing.T) {
	db := setupTripDB(t)
	tripRepo := NewTripRepository(db)
	itineraryRepo := NewItineraryRepository(db)
	stopRepo := NewStopRepository(db)
	ctx := context.Background()

	trip := &Trip{Destination: "HookTest", DurationDays: 1}
	if err := tripRepo.Create(ctx, trip); err != nil {
		t.Fatalf("create trip: %v", err)
	}
	itin := &Itinerary{TripID: trip.ID, Title: "T", Position: 0}
	if err := itineraryRepo.Create(ctx, itin); err != nil {
		t.Fatalf("create itin: %v", err)
	}
	stop := &Stop{ItineraryID: itin.ID, Position: 0, Kind: StopKindLandmark, Title: "Empty Image Stop", Body: "body", ImageURL: ""}
	if err := stopRepo.Create(ctx, stop); err != nil {
		t.Fatalf("create stop: %v", err)
	}
	if stop.ImageURL == "" {
		t.Errorf("expected placeholder ImageURL, got empty")
	}
	if stop.ImageSource != ImageSourcePlaceholder {
		t.Errorf("expected placeholder source, got %q", stop.ImageSource)
	}
	// Verify persisted value also has placeholder
	found, err := stopRepo.FindByID(ctx, stop.ID)
	if err != nil {
		t.Fatalf("find stop: %v", err)
	}
	if found.ImageURL == "" {
		t.Errorf("persisted stop missing placeholder")
	}
}

func TestTripRepository_Delete_Cascade(t *testing.T) {
	db := setupTripDB(t)
	tripRepo := NewTripRepository(db)
	itineraryRepo := NewItineraryRepository(db)
	stopRepo := NewStopRepository(db)
	ctx := context.Background()

	trip := &Trip{Destination: "Cascade", DurationDays: 1}
	if err := tripRepo.Create(ctx, trip); err != nil {
		t.Fatalf("create trip: %v", err)
	}
	itin := &Itinerary{TripID: trip.ID, Title: "T", Position: 0}
	if err := itineraryRepo.Create(ctx, itin); err != nil {
		t.Fatalf("create itin: %v", err)
	}
	stop := &Stop{ItineraryID: itin.ID, Position: 0, Kind: StopKindLandmark, Title: "S", Body: "b", ImageURL: "https://example.com/x.jpg"}
	if err := stopRepo.Create(ctx, stop); err != nil {
		t.Fatalf("create stop: %v", err)
	}

	// Delete via repository (load-then-delete so hooks see ID)
	n, err := tripRepo.Delete(ctx, trip.ID)
	if err != nil {
		t.Fatalf("delete trip: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1, got %d", n)
	}
	_, err = tripRepo.FindByID(ctx, trip.ID)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("trip should be soft-deleted, find should return ErrRecordNotFound, got %v", err)
	}
	// Itineraries should be cascade soft-deleted
	list, _ := itineraryRepo.ListByParent(ctx, trip.ID)
	if len(list) != 0 {
		t.Errorf("expected itineraries cascaded, got %d", len(list))
	}
	c, _ := itineraryRepo.CountByParent(ctx, trip.ID)
	if c != 0 {
		t.Errorf("expected itinerary count 0 after cascade, got %d", c)
	}
	// Stops also cascaded via itinerary hook
	slist, _ := stopRepo.ListByParent(ctx, itin.ID)
	if len(slist) != 0 {
		t.Errorf("expected stops cascaded, got %d", len(slist))
	}
	// Verify via unscoped that rows still exist but soft-deleted
	var count int64
	db.Unscoped().Model(&Itinerary{}).Where("trip_id = ?", trip.ID).Count(&count)
	if count != 1 {
		t.Errorf("unscoped itinerary count expected 1, got %d", count)
	}
}
