package trip

import (
	"context"
	"testing"
)

func TestMockGenerator_Rome_ReturnsTwoItineraries(t *testing.T) {
	gen := NewMockGenerator()
	params := CreateTripParams{
		Destination:  "Rome, Lazio, Italy",
		City:         "Rome",
		Country:      "Italy",
		Month:        "September",
		DurationDays: 3,
		Pace:         "Moderate",
		Interests:    []string{"Architecture", "History", "Local Food"},
		Mobility:     "Walking + Public Transit",
	}

	itins, err := gen.Generate(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected generation error: %v", err)
	}
	if len(itins) != 2 {
		t.Fatalf("expected exactly 2 itineraries, got %d", len(itins))
	}

	for idx, itin := range itins {
		if len(itin.Stops) == 0 {
			t.Errorf("itinerary %d: expected stops", idx)
		}
		if len(itin.Segments) != len(itin.Stops)-1 {
			t.Errorf("itinerary %d: N-1 invariant broken: %d stops but %d segments", idx, len(itin.Stops), len(itin.Segments))
		}
		if itin.Title == "" || itin.Theme == "" {
			t.Errorf("itinerary %d: expected title/theme", idx)
		}
		for _, s := range itin.Stops {
			if s.ImageURL == "" {
				t.Errorf("itinerary %d stop %q: image url must never be empty", idx, s.Title)
			}
			if !ValidStopKinds[s.Kind] {
				t.Errorf("itinerary %d stop %q: invalid kind %q", idx, s.Title, s.Kind)
			}
			if s.Title == "" || s.Body == "" {
				t.Errorf("itinerary %d stop %q: title/body required", idx, s.Title)
			}
		}
		for _, seg := range itin.Segments {
			if !ValidSegmentModes[seg.Mode] {
				t.Errorf("itinerary %d segment %d: invalid mode %q", idx, seg.Position, seg.Mode)
			}
			if seg.DistanceMeters <= 0 || seg.DurationMinutes <= 0 {
				t.Errorf("itinerary %d segment %d: distance/duration must be positive", idx, seg.Position)
			}
		}
	}

	// Check scaling: 2-day vs 7-day (R10, AE5)
	paramsShort := params
	paramsShort.DurationDays = 2
	itinsShort, _ := gen.Generate(context.Background(), paramsShort)

	paramsLong := params
	paramsLong.DurationDays = 7
	itinsLong, _ := gen.Generate(context.Background(), paramsLong)

	if len(itinsLong[0].Stops) <= len(itinsShort[0].Stops) {
		t.Errorf("expected 7-day itinerary to have more stops than 2-day: short=%d long=%d", len(itinsShort[0].Stops), len(itinsLong[0].Stops))
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

	itins, err := gen.Generate(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected generation error: %v", err)
	}
	if len(itins) != 2 {
		t.Fatalf("expected 2 itineraries for Kyoto, got %d", len(itins))
	}
	for _, itin := range itins {
		if len(itin.Stops) == 0 {
			t.Error("expected stops for Kyoto")
		}
		if len(itin.Segments) != len(itin.Stops)-1 {
			t.Errorf("N-1 broken for Kyoto: %d stops %d segments", len(itin.Stops), len(itin.Segments))
		}
	}
}

func TestMockGenerator_MobilityInfluencesMode(t *testing.T) {
	gen := NewMockGenerator()
	params := CreateTripParams{
		Destination:  "Paris, France",
		City:         "Paris",
		DurationDays: 4,
		Mobility:     "Walking Only",
	}
	itins, _ := gen.Generate(context.Background(), params)
	for _, itin := range itins {
		for _, seg := range itin.Segments {
			if seg.Mode != SegmentModeWalk {
				t.Errorf("Walking Only should produce only walk segments, got %q", seg.Mode)
			}
		}
	}
}
