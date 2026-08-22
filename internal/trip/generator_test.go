package trip

import (
	"context"
	"testing"
)

func TestMockGenerator_Rome(t *testing.T) {
	gen := NewMockGenerator()
	params := CreateTripParams{
		Destination:  "Rome, Lazio, Italy",
		City:         "Rome",
		Country:      "Italy",
		Month:        "September",
		DurationDays: 3,
		Pace:         "Moderate",
		Interests:    []string{"Architecture", "History", "Local Food"},
		Mobility:     "Walking + Transit",
	}

	it, err := gen.Generate(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected generation error: %v", err)
	}

	if it == nil {
		t.Fatal("expected non-nil Itinerary")
	}

	if len(it.Days) != 3 {
		t.Fatalf("expected 3 days, got %d", len(it.Days))
	}

	if len(it.HighlightCards) == 0 {
		t.Error("expected at least 1 highlight card")
	}

	if len(it.CoolFacts) == 0 {
		t.Error("expected at least 1 cool fact")
	}

	// Verify Day 1 content
	day1 := it.Days[0]
	if day1.DayNumber != 1 {
		t.Errorf("expected DayNumber 1, got %d", day1.DayNumber)
	}
	if len(day1.Morning) == 0 {
		t.Error("expected morning stops for Day 1")
	}
	if len(day1.Afternoon) == 0 {
		t.Error("expected afternoon stops for Day 1")
	}
	if len(day1.Evening) == 0 {
		t.Error("expected evening stops for Day 1")
	}
}

func TestMockGenerator_GenericCity(t *testing.T) {
	gen := NewMockGenerator()
	params := CreateTripParams{
		Destination:  "Kyoto, Japan",
		City:         "Kyoto",
		Country:      "Japan",
		Month:        "November",
		DurationDays: 2,
		Pace:         "Relaxed",
		Interests:    []string{"Architecture", "Culture"},
	}

	it, err := gen.Generate(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected generation error: %v", err)
	}

	if len(it.Days) != 2 {
		t.Fatalf("expected 2 days for Kyoto, got %d", len(it.Days))
	}

	if len(it.HighlightCards) == 0 {
		t.Error("expected highlight cards for Kyoto")
	}
}
