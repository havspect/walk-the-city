package trip

import (
	"testing"
)

func TestTrip_ItineraryJSONSerialization(t *testing.T) {
	trip := &Trip{
		Destination:  "Rome, Italy",
		City:         "Rome",
		Country:      "Italy",
		Lat:          41.8933,
		Lon:          12.4829,
		Month:        "September",
		DurationDays: 3,
		Pace:         "Moderate",
		Interests:    "Architecture, History, Local Food",
		Mobility:     "Walking + Transit",
	}

	it := &Itinerary{
		Summary:    "A historic exploration of Rome in mild September weather.",
		BestSeason: "Autumn (September - October)",
		Days: []DayPlan{
			{
				DayNumber: 1,
				Theme:     "Ancient Foundations & Monti Living",
				Morning: []Stop{
					{
						Name:           "Colosseum & Ludus Magnus",
						Neighborhood:   "Celio",
						Category:       "Architecture",
						Description:    "Flavian amphitheater and adjacent gladiatorial training grounds.",
						WalkingMinutes: 10,
						TransitTip:     "Metro Line B to Colosseo station",
						Lat:            41.8902,
						Lon:            12.4922,
						WikiQuery:      "Colosseum",
					},
				},
				Afternoon: []Stop{
					{
						Name:           "Monti Historic Alleys & Gelaterie",
						Neighborhood:   "Monti",
						Category:       "Food & Living",
						Description:    "Artisan workshops and traditional gelato on Via Urbana.",
						WalkingMinutes: 8,
						TransitTip:     "Short stroll from Piazza Venezia",
						Lat:            41.8950,
						Lon:            12.4920,
						WikiQuery:      "Monti (rione of Rome)",
					},
				},
				Evening: []Stop{
					{
						Name:           "Piazza Madonna dei Monti",
						Neighborhood:   "Monti",
						Category:       "Food & Living",
						Description:    "Local piazza gathering with authentic aperitivo and supplì.",
						WalkingMinutes: 5,
						Lat:            41.8942,
						Lon:            12.4905,
						WikiQuery:      "Piazza della Madonna dei Monti",
					},
				},
			},
		},
		HighlightCards: []HighlightCard{
			{
				Category:       "Architecture",
				Title:          "Pantheon Concrete Dome",
				Neighborhood:   "Pigna",
				Story:          "Unreinforced concrete dome that has stood intact for almost 2,000 years.",
				Tip:            "Visit at midday to see sunlight illuminate the interior oculus.",
				WalkingMinutes: 12,
				TransitTip:     "Bus 64 or 70 from Termini",
				Lat:            41.8986,
				Lon:            12.4769,
				WikiQuery:      "Pantheon, Rome",
			},
			{
				Category:       "Food & Living",
				Title:          "Trastevere Backstreet Forno",
				Neighborhood:   "Trastevere",
				Story:          "Neighborhood bakery renowned for pizza al taglio and morning supplì.",
				Tip:            "Arrive before 1 PM for the freshest pizza bianca.",
				WalkingMinutes: 15,
				TransitTip:     "Tram 8 to Piazza Sonnino",
				Lat:            41.8885,
				Lon:            12.4700,
				WikiQuery:      "Trastevere",
			},
		},
		CoolFacts: []CoolFactCard{
			{
				Title:           "Nasone Public Water Fountains",
				Fact:            "Rome has over 2,500 continuous cold drinking water fountains nicknamed nasoni (big noses).",
				SeasonalityNote: "Crisp, cold water is especially refreshing in early September.",
			},
		},
	}

	err := trip.SetItinerary(it)
	if err != nil {
		t.Fatalf("SetItinerary failed: %v", err)
	}

	if trip.ItineraryJSON == "" {
		t.Fatal("expected non-empty ItineraryJSON")
	}

	loadedIt, err := trip.GetItinerary()
	if err != nil {
		t.Fatalf("GetItinerary failed: %v", err)
	}

	if loadedIt == nil {
		t.Fatal("expected non-nil Itinerary")
	}

	if loadedIt.Summary != it.Summary {
		t.Errorf("expected summary '%s', got '%s'", it.Summary, loadedIt.Summary)
	}

	if len(loadedIt.Days) != 1 {
		t.Fatalf("expected 1 day plan, got %d", len(loadedIt.Days))
	}

	if len(loadedIt.HighlightCards) != 2 {
		t.Fatalf("expected 2 highlight cards, got %d", len(loadedIt.HighlightCards))
	}

	if len(loadedIt.CoolFacts) != 1 {
		t.Fatalf("expected 1 cool fact, got %d", len(loadedIt.CoolFacts))
	}

	if loadedIt.HighlightCards[0].Title != "Pantheon Concrete Dome" {
		t.Errorf("expected card title 'Pantheon Concrete Dome', got '%s'", loadedIt.HighlightCards[0].Title)
	}
}

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

func TestTrip_EmptyItineraryJSON(t *testing.T) {
	trip := &Trip{}
	it, err := trip.GetItinerary()
	if err != nil {
		t.Fatalf("unexpected error on empty itinerary json: %v", err)
	}
	if it != nil {
		t.Errorf("expected nil itinerary for empty JSON, got %+v", it)
	}
}
