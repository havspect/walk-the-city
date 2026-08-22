package trip

import (
	"strings"
	"time"
)

// StopKind discriminates the uniform highlight-card type.
const (
	StopKindLandmark    = "Landmark"
	StopKindHistory     = "History"
	StopKindFoodDrink   = "FoodDrink"
	StopKindHiddenGem   = "HiddenGem"
	StopKindParkNature  = "ParkNature"
	StopKindNeighborhood = "Neighborhood"
	StopKindTrivia      = "Trivia"
)

// ValidStopKinds is the closed set of Stop.kind values (R4).
var ValidStopKinds = map[string]bool{
	StopKindLandmark:     true,
	StopKindHistory:      true,
	StopKindFoodDrink:    true,
	StopKindHiddenGem:    true,
	StopKindParkNature:   true,
	StopKindNeighborhood: true,
	StopKindTrivia:       true,
}

// SegmentMode is the transport mode between two consecutive stops (R13).
const (
	SegmentModeWalk    = "walk"
	SegmentModeTransit = "transit"
	SegmentModeBicycle = "bicycle"
	SegmentModeDrive   = "drive"
)

// ValidSegmentModes is the closed set of Segment mode values.
var ValidSegmentModes = map[string]bool{
	SegmentModeWalk:    true,
	SegmentModeTransit: true,
	SegmentModeBicycle: true,
	SegmentModeDrive:   true,
}

// Trip is the base entity. It owns many alternative Itineraries (R1).
type Trip struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Destination  string    `gorm:"size:255;not null" json:"destination"`
	City         string    `gorm:"size:255" json:"city"`
	Country      string    `gorm:"size:255" json:"country"`
	Lat          float64   `json:"lat"`
	Lon          float64   `json:"lon"`
	Month        string    `gorm:"size:50" json:"month"`
	DurationDays int       `gorm:"not null;default:1" json:"duration_days"`
	Pace         string    `gorm:"size:50" json:"pace"`
	Interests    string    `gorm:"type:text" json:"interests"`
	Mobility     string    `gorm:"size:100" json:"mobility"`
	Notes        string    `gorm:"type:text" json:"notes"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	Itineraries []Itinerary `gorm:"foreignKey:TripID;constraint:OnDelete:CASCADE" json:"itineraries"`
}

// Itinerary is a themed / alternative plan for a Trip (R2). Not tied to a calendar day.
type Itinerary struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	TripID     uint      `gorm:"not null;index;constraint:OnDelete:CASCADE" json:"trip_id"`
	Title      string    `gorm:"size:255" json:"title"`
	Theme      string    `gorm:"size:255" json:"theme"`
	Summary    string    `gorm:"type:text" json:"summary"`
	BestSeason string    `gorm:"type:text" json:"best_season"`
	Position   int       `gorm:"not null;default:0" json:"position"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	Stops    []Stop    `gorm:"foreignKey:ItineraryID;constraint:OnDelete:CASCADE" json:"stops"`
	Segments []Segment `gorm:"foreignKey:ItineraryID;constraint:OnDelete:CASCADE" json:"segments"`
}

// Stop is a uniform highlight card within an Itinerary (R4-R8). Every Stop has image+text (R6).
type Stop struct {
	ID                 uint    `gorm:"primaryKey" json:"id"`
	ItineraryID        uint    `gorm:"not null;index;constraint:OnDelete:CASCADE" json:"itinerary_id"`
	Position           int     `gorm:"not null" json:"position"`
	Kind               string  `gorm:"size:50;not null" json:"kind"`
	Title              string  `gorm:"size:255;not null" json:"title"`
	Body               string  `gorm:"type:text;not null" json:"body"`
	Neighborhood       string  `gorm:"size:255" json:"neighborhood"`
	Lat                float64 `json:"lat"`
	Lon                float64 `json:"lon"`
	WikiQuery          string  `gorm:"size:255" json:"wiki_query"`
	RecommendedMinutes int     `json:"recommended_minutes"`
	Tip                string  `gorm:"type:text" json:"tip"`

	// Image fields — never empty url (R19). Placeholder at creation, upgraded in place (R17-R19).
	ImageURL    string `gorm:"size:1024;not null" json:"image_url"`
	ImageAlt    string `gorm:"size:255" json:"image_alt"`
	ImageCredit string `gorm:"size:255" json:"image_credit"`
	ImageSource string `gorm:"size:50" json:"image_source"`
	ImagePrompt string `gorm:"type:text" json:"image_prompt"`
}

// Segment connects two consecutive Stops within one Itinerary (R11-R13).
type Segment struct {
	ID              uint   `gorm:"primaryKey" json:"id"`
	ItineraryID     uint   `gorm:"not null;index;constraint:OnDelete:CASCADE" json:"itinerary_id"`
	FromStopID      uint   `gorm:"not null;index" json:"from_stop_id"`
	ToStopID        uint   `gorm:"not null;index" json:"to_stop_id"`
	Position        int    `gorm:"not null" json:"position"`
	Mode            string `gorm:"size:50;not null" json:"mode"`
	DistanceMeters  int    `gorm:"not null" json:"distance_meters"`
	DurationMinutes int    `gorm:"not null" json:"duration_minutes"`
	Instruction     string `gorm:"type:text" json:"instruction"`
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
