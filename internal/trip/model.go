package trip

import (
	"strings"

	"gorm.io/gorm"
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

// ImageSource constants for Stop images (R17-R19).
const (
	ImageSourcePlaceholder = "placeholder"
	ImageSourceBFLFlux     = "bfl_flux"
)

// ValidImageSources is the allowed set for Stop.ImageSource.
var ValidImageSources = map[string]bool{
	ImageSourcePlaceholder: true,
	ImageSourceBFLFlux:     true,
}

// Trip is the base entity. It owns many alternative Itineraries (R1).
type Trip struct {
	gorm.Model
	Destination  string `gorm:"size:255;not null" json:"destination"`
	City         string `gorm:"size:255" json:"city"`
	Country      string `gorm:"size:255" json:"country"`
	Lat          float64 `json:"lat"`
	Lon          float64 `json:"lon"`
	Month        string `gorm:"size:50" json:"month"`
	DurationDays int    `gorm:"not null;default:1" json:"duration_days"`
	Pace         string `gorm:"size:50" json:"pace"`
	Interests    string `gorm:"type:text" json:"interests"`
	Mobility     string `gorm:"size:100" json:"mobility"`
	Notes        string `gorm:"type:text" json:"notes"`

	Itineraries []Itinerary `gorm:"foreignKey:TripID;constraint:OnDelete:CASCADE" json:"itineraries"`
}

// Itinerary is a themed / alternative plan for a Trip (R2). Not tied to a calendar day.
type Itinerary struct {
	gorm.Model
	TripID     uint   `gorm:"not null;index;constraint:OnDelete:CASCADE" json:"trip_id"`
	Title      string `gorm:"size:255" json:"title"`
	Theme      string `gorm:"size:255" json:"theme"`
	Summary    string `gorm:"type:text" json:"summary"`
	BestSeason string `gorm:"type:text" json:"best_season"`
	Position   int    `gorm:"not null;default:0" json:"position"`

	Stops    []Stop    `gorm:"foreignKey:ItineraryID;constraint:OnDelete:CASCADE" json:"stops"`
	Segments []Segment `gorm:"foreignKey:ItineraryID;constraint:OnDelete:CASCADE" json:"segments"`
}

// Stop is a uniform highlight card within an Itinerary (R4-R8). Every Stop has image+text (R6).
type Stop struct {
	gorm.Model
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

// BeforeCreate ensures every Stop has a non-empty ImageURL (R19). Direct DB writes
// bypassing TripService still get a deterministic placeholder.
func (s *Stop) BeforeCreate(tx *gorm.DB) error {
	if strings.TrimSpace(s.ImageURL) == "" {
		title := strings.TrimSpace(s.Title)
		if title == "" {
			title = "walk-the-city"
		}
		seed := strings.ToLower(title)
		seed = strings.ReplaceAll(seed, " ", "-")
		var b strings.Builder
		for _, r := range seed {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
				b.WriteRune(r)
			}
		}
		out := b.String()
		if out == "" {
			out = "walk-the-city"
		}
		if len(out) > 40 {
			out = out[:40]
		}
		s.ImageURL = "https://picsum.photos/seed/" + out + "/600/400"
	}
	if strings.TrimSpace(s.ImageSource) == "" {
		s.ImageSource = ImageSourcePlaceholder
	}
	return nil
}

// Trip soft-delete cascades to itineraries (and their stops/segments) to preserve
// the hard-delete cascade semantics expected before gorm.Model was introduced.
func (t *Trip) AfterDelete(tx *gorm.DB) error {
	if t.ID == 0 {
		return nil
	}
	// Use the same delete mode (soft vs hard) as the parent delete.
	// Load itineraries in the current scope so soft-deleted parents still find children.
	var itineraries []Itinerary
	if err := tx.Where("trip_id = ?", t.ID).Find(&itineraries).Error; err != nil {
		return err
	}
	// Fallback to Unscoped when the parent was hard-deleted and children are already excluded
	// by foreign-key cascade or previous soft deletes.
	if len(itineraries) == 0 {
		if err := tx.Unscoped().Where("trip_id = ?", t.ID).Find(&itineraries).Error; err != nil {
			return err
		}
	}
	for i := range itineraries {
		if err := tx.Delete(&itineraries[i]).Error; err != nil {
			return err
		}
	}
	return nil
}

// Itinerary soft-delete cascades to stops and segments.
func (it *Itinerary) AfterDelete(tx *gorm.DB) error {
	if it.ID == 0 {
		return nil
	}
	if err := tx.Where("itinerary_id = ?", it.ID).Delete(&Stop{}).Error; err != nil {
		return err
	}
	if err := tx.Where("itinerary_id = ?", it.ID).Delete(&Segment{}).Error; err != nil {
		return err
	}
	return nil
}

// Segment connects two consecutive Stops within one Itinerary (R11-R13).
type Segment struct {
	gorm.Model
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
