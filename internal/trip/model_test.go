package trip

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestTrip_InterestsList(t *testing.T) {
	trip := &Trip{
		Interests: "Architecture, History, Local Food",
	}
	list := trip.InterestsList()
	if len(list) != 3 {
		t.Fatalf("expected 3 items in list, got %d", len(list))
	}
	expected := []string{"Architecture", "History", "Local Food"}
	for i, exp := range expected {
		if list[i] != exp {
			t.Errorf("expected item %d to be '%s', got '%s'", i, exp, list[i])
		}
	}
}

func TestTrip_EmptyInterestsList(t *testing.T) {
	trip := &Trip{Interests: "   "}
	if list := trip.InterestsList(); list != nil {
		t.Errorf("expected nil for empty interests, got %v", list)
	}
}

func TestModel_NormalizedSchema_CreateAndPreload(t *testing.T) {
	dsn := fmt.Sprintf("file:memmodel_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&Trip{}, &Itinerary{}, &Stop{}, &Segment{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	trip := &Trip{Destination: "Rome, Italy", DurationDays: 3}
	if err := db.Create(trip).Error; err != nil {
		t.Fatalf("create trip: %v", err)
	}

	itin := &Itinerary{TripID: trip.ID, Title: "Classic Highlights", Theme: "Ancient", Summary: "summary", BestSeason: "Spring", Position: 0}
	if err := db.Create(itin).Error; err != nil {
		t.Fatalf("create itinerary: %v", err)
	}

	stops := []Stop{
		{ItineraryID: itin.ID, Position: 0, Kind: StopKindLandmark, Title: "Colosseum", Body: "body", ImageURL: "https://example.com/a.jpg"},
		{ItineraryID: itin.ID, Position: 1, Kind: StopKindHistory, Title: "Forum", Body: "body", ImageURL: "https://example.com/b.jpg"},
		{ItineraryID: itin.ID, Position: 2, Kind: StopKindFoodDrink, Title: "Trattoria", Body: "body", Tip: "order cacio e pepe", ImageURL: "https://example.com/c.jpg"},
	}
	for i := range stops {
		if err := db.Create(&stops[i]).Error; err != nil {
			t.Fatalf("create stop %d: %v", i, err)
		}
	}

	segment := Segment{ItineraryID: itin.ID, FromStopID: stops[0].ID, ToStopID: stops[1].ID, Position: 0, Mode: SegmentModeWalk, DistanceMeters: 500, DurationMinutes: 6, Instruction: "Walk"}
	if err := db.Create(&segment).Error; err != nil {
		t.Fatalf("create segment: %v", err)
	}
	segment2 := Segment{ItineraryID: itin.ID, FromStopID: stops[1].ID, ToStopID: stops[2].ID, Position: 1, Mode: SegmentModeTransit, DistanceMeters: 1200, DurationMinutes: 10, Instruction: "Transit"}
	if err := db.Create(&segment2).Error; err != nil {
		t.Fatalf("create segment2: %v", err)
	}

	var loaded Trip
	if err := db.Preload("Itineraries", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC") }).
		Preload("Itineraries.Stops", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC") }).
		Preload("Itineraries.Segments", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC") }).
		First(&loaded, trip.ID).Error; err != nil {
		t.Fatalf("preload: %v", err)
	}

	if len(loaded.Itineraries) != 1 {
		t.Fatalf("expected 1 itinerary, got %d", len(loaded.Itineraries))
	}
	if len(loaded.Itineraries[0].Stops) != 3 {
		t.Fatalf("expected 3 stops, got %d", len(loaded.Itineraries[0].Stops))
	}
	if len(loaded.Itineraries[0].Segments) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(loaded.Itineraries[0].Segments))
	}
	if loaded.Itineraries[0].Stops[0].Title != "Colosseum" {
		t.Errorf("wrong first stop title %q", loaded.Itineraries[0].Stops[0].Title)
	}
	if loaded.Itineraries[0].Stops[2].Tip != "order cacio e pepe" {
		t.Errorf("expected tip preserved")
	}
	// N-1 invariant
	if len(loaded.Itineraries[0].Segments) != len(loaded.Itineraries[0].Stops)-1 {
		t.Errorf("N-1 invariant broken")
	}
}

func TestModel_CascadeDelete(t *testing.T) {
	dsn := fmt.Sprintf("file:memcascade_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	_ = db.Exec("PRAGMA foreign_keys=ON")
	if err := db.AutoMigrate(&Trip{}, &Itinerary{}, &Stop{}, &Segment{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	trip := &Trip{Destination: "Test", DurationDays: 2}
	db.Create(trip)
	itin := &Itinerary{TripID: trip.ID, Title: "T", Position: 0}
	db.Create(itin)
	stop := Stop{ItineraryID: itin.ID, Position: 0, Kind: StopKindLandmark, Title: "S", Body: "b", ImageURL: "https://example.com/x.jpg"}
	db.Create(&stop)

	if err := db.Delete(trip).Error; err != nil {
		t.Fatalf("delete trip: %v", err)
	}
	var count int64
	db.Model(&Itinerary{}).Where("trip_id = ?", trip.ID).Count(&count)
	if count != 0 {
		t.Errorf("expected itineraries cascaded, got %d", count)
	}
}
