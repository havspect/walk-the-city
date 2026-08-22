package trip

import (
	"encoding/json"
	"strings"
	"time"
)

// Trip represents a persistent city trip plan in the database.
type Trip struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Destination   string    `gorm:"size:255;not null" json:"destination"`
	City          string    `gorm:"size:255" json:"city"`
	Country       string    `gorm:"size:255" json:"country"`
	Lat           float64   `json:"lat"`
	Lon           float64   `json:"lon"`
	Month         string    `gorm:"size:50" json:"month"`
	DurationDays  int       `gorm:"not null;default:1" json:"duration_days"`
	Pace          string    `gorm:"size:50" json:"pace"`
	Interests     string    `gorm:"type:text" json:"interests"`
	Mobility      string    `gorm:"size:100" json:"mobility"`
	Notes         string    `gorm:"type:text" json:"notes"`
	ItineraryJSON string    `gorm:"type:text" json:"itinerary_json"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Itinerary encapsulates the structured schedule, highlight cards, and trivia facts.
type Itinerary struct {
	Summary        string          `json:"summary"`
	BestSeason     string          `json:"best_season"`
	Days           []DayPlan       `json:"days"`
	HighlightCards []HighlightCard `json:"highlight_cards"`
	CoolFacts      []CoolFactCard  `json:"cool_facts"`
}

// DayPlan represents a daily schedule broken into Morning, Afternoon, and Evening stops.
type DayPlan struct {
	DayNumber int    `json:"day_number"`
	Theme     string `json:"theme"`
	Morning   []Stop `json:"morning"`
	Afternoon []Stop `json:"afternoon"`
	Evening   []Stop `json:"evening"`
}

// Stop represents an individual stop or waypoint within a daily itinerary.
type Stop struct {
	Name           string  `json:"name"`
	Neighborhood   string  `json:"neighborhood"`
	Category       string  `json:"category"`
	Description    string  `json:"description"`
	WalkingMinutes int     `json:"walking_minutes"`
	TransitTip     string  `json:"transit_tip"`
	Lat            float64 `json:"lat"`
	Lon            float64 `json:"lon"`
	WikiQuery      string  `json:"wiki_query"`
}

// HighlightCard represents a standalone visual card highlighting architecture, history, or local food.
type HighlightCard struct {
	Category       string  `json:"category"`
	Title          string  `json:"title"`
	Neighborhood   string  `json:"neighborhood"`
	Story          string  `json:"story"`
	Tip            string  `json:"tip"`
	WalkingMinutes int     `json:"walking_minutes"`
	TransitTip     string  `json:"transit_tip"`
	Lat            float64 `json:"lat"`
	Lon            float64 `json:"lon"`
	WikiQuery      string  `json:"wiki_query"`
}

// CoolFactCard represents a trivia, cultural quirk, or seasonal recommendation.
type CoolFactCard struct {
	Title           string `json:"title"`
	Fact            string `json:"fact"`
	SeasonalityNote string `json:"seasonality_note"`
}

// GetItinerary deserializes the stored ItineraryJSON into an Itinerary struct.
func (t *Trip) GetItinerary() (*Itinerary, error) {
	if strings.TrimSpace(t.ItineraryJSON) == "" {
		return nil, nil
	}
	var it Itinerary
	if err := json.Unmarshal([]byte(t.ItineraryJSON), &it); err != nil {
		return nil, err
	}
	return &it, nil
}

// SetItinerary serializes an Itinerary struct into the ItineraryJSON string field.
func (t *Trip) SetItinerary(it *Itinerary) error {
	if it == nil {
		t.ItineraryJSON = ""
		return nil
	}
	data, err := json.Marshal(it)
	if err != nil {
		return err
	}
	t.ItineraryJSON = string(data)
	return nil
}

// InterestsList parses the comma-separated Interests string into a slice of trimmed strings.
func (t *Trip) InterestsList() []string {
	if strings.TrimSpace(t.Interests) == "" {
		return nil
	}
	raw := strings.Split(t.Interests, ",")
	list := make([]string, 0, len(raw))
	for _, item := range raw {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			list = append(list, trimmed)
		}
	}
	return list
}
