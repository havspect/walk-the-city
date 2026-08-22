package trip

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"gorm.io/gorm"
)

var (
	// ErrNotFound indicates a requested trip was not found.
	ErrNotFound = errors.New("trip not found")
)

// ValidationError represents one or more invalid input fields.
type ValidationError struct {
	FieldErrors map[string]string
}

func (v *ValidationError) Error() string {
	keys := make([]string, 0, len(v.FieldErrors))
	for field := range v.FieldErrors {
		keys = append(keys, field)
	}
	sort.Strings(keys)

	var msgs []string
	for _, field := range keys {
		msgs = append(msgs, fmt.Sprintf("%s: %s", field, v.FieldErrors[field]))
	}
	return strings.Join(msgs, ", ")
}

// CreateTripParams holds inputs for creating a new city trip.
type CreateTripParams struct {
	Destination  string
	City         string
	Country      string
	Lat          float64
	Lon          float64
	Month        string
	DurationDays int
	Pace         string
	Interests    []string
	Mobility     string
	Notes        string
	Itineraries  []*Itinerary
}

// Validate verifies that the trip input parameters satisfy business constraints.
func (p CreateTripParams) Validate() *ValidationError {
	errs := make(map[string]string)

	dest := strings.TrimSpace(p.Destination)
	if dest == "" {
		errs["destination"] = "Destination is required"
	} else if len(dest) > 255 {
		errs["destination"] = "Destination cannot exceed 255 characters"
	}

	if p.DurationDays < 1 || p.DurationDays > 30 {
		errs["duration_days"] = "Duration must be between 1 and 30 days"
	}

	notes := strings.TrimSpace(p.Notes)
	if len(notes) > 2000 {
		errs["notes"] = "Notes cannot exceed 2000 characters"
	}

	if len(errs) > 0 {
		return &ValidationError{FieldErrors: errs}
	}
	return nil
}

// TripService defines the business operations for city trip management.
type TripService interface {
	CreateTrip(ctx context.Context, params CreateTripParams) (*Trip, error)
	ListTrips(ctx context.Context) ([]Trip, error)
	GetTripByID(ctx context.Context, id uint) (*Trip, error)
	GenerateAndSaveTrip(ctx context.Context, params CreateTripParams, gen TripGenerator) (*Trip, error)
}

type service struct {
	db *gorm.DB
}

// NewService constructs a default TripService backed by GORM.
func NewService(db *gorm.DB) TripService {
	return &service{db: db}
}

func validateItineraries(itins []*Itinerary) *ValidationError {
	errs := make(map[string]string)
	for i, itin := range itins {
		if itin == nil {
			continue
		}
		prefix := fmt.Sprintf("itineraries[%d]", i)
		if itin.Title == "" && len(itin.Stops) > 0 {
			// Title is expected for stacked sections; advisory but not hard fail (generator always sets it)
		}
		// Validate stops
		for j, st := range itin.Stops {
			if !ValidStopKinds[st.Kind] {
				errs[fmt.Sprintf("%s.stops[%d].kind", prefix, j)] = fmt.Sprintf("invalid stop kind %q", st.Kind)
			}
			if strings.TrimSpace(st.Title) == "" {
				errs[fmt.Sprintf("%s.stops[%d].title", prefix, j)] = "stop title is required"
			}
			if strings.TrimSpace(st.Body) == "" {
				errs[fmt.Sprintf("%s.stops[%d].body", prefix, j)] = "stop body is required"
			}
			if st.ImageSource != "" && !ValidImageSources[st.ImageSource] {
				errs[fmt.Sprintf("%s.stops[%d].image_source", prefix, j)] = fmt.Sprintf("invalid image_source %q", st.ImageSource)
			}
		}
		// N-1 invariant (R12): N stops => N-1 segments; 0/1 stop => 0 segments
		wantSegs := 0
		if len(itin.Stops) >= 2 {
			wantSegs = len(itin.Stops) - 1
		}
		if len(itin.Segments) != wantSegs {
			errs[fmt.Sprintf("%s.segments", prefix)] = fmt.Sprintf("expected %d segments for %d stops, got %d", wantSegs, len(itin.Stops), len(itin.Segments))
		}
		for j, seg := range itin.Segments {
			if !ValidSegmentModes[seg.Mode] {
				errs[fmt.Sprintf("%s.segments[%d].mode", prefix, j)] = fmt.Sprintf("invalid segment mode %q", seg.Mode)
			}
			if seg.DistanceMeters <= 0 {
				errs[fmt.Sprintf("%s.segments[%d].distance_meters", prefix, j)] = "distance must be > 0"
			}
			if seg.DurationMinutes <= 0 {
				errs[fmt.Sprintf("%s.segments[%d].duration_minutes", prefix, j)] = "duration must be > 0"
			}
		}
	}
	if len(errs) > 0 {
		return &ValidationError{FieldErrors: errs}
	}
	return nil
}

func (s *service) CreateTrip(ctx context.Context, params CreateTripParams) (*Trip, error) {
	if validationErr := params.Validate(); validationErr != nil {
		return nil, validationErr
	}
	if v := validateItineraries(params.Itineraries); v != nil {
		return nil, v
	}

	cityName := strings.TrimSpace(params.City)
	if cityName == "" {
		parts := strings.Split(params.Destination, ",")
		cityName = strings.TrimSpace(parts[0])
	}

	interestsStr := strings.Join(params.Interests, ", ")

	t := &Trip{
		Destination:  strings.TrimSpace(params.Destination),
		City:         cityName,
		Country:      strings.TrimSpace(params.Country),
		Lat:          params.Lat,
		Lon:          params.Lon,
		Month:        strings.TrimSpace(params.Month),
		DurationDays: params.DurationDays,
		Pace:         strings.TrimSpace(params.Pace),
		Interests:    interestsStr,
		Mobility:     strings.TrimSpace(params.Mobility),
		Notes:        strings.TrimSpace(params.Notes),
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(t).Error; err != nil {
			return fmt.Errorf("failed to create trip: %w", err)
		}

		for idx, itin := range params.Itineraries {
			if itin == nil {
				continue
			}
			itin.TripID = t.ID
			itin.Position = idx
			// Ensure segments/stops will be created with correct FK; we need to hold them before Create
			stops := itin.Stops
			segments := itin.Segments
			itin.Stops = nil
			itin.Segments = nil

			if err := tx.Create(itin).Error; err != nil {
				return fmt.Errorf("failed to create itinerary: %w", err)
			}

			// Create stops and remember IDs by position
			savedStops := make([]Stop, 0, len(stops))
			for pos, st := range stops {
				st.ItineraryID = itin.ID
				st.Position = pos
				// Ensure image URL never empty (R19)
				if strings.TrimSpace(st.ImageURL) == "" {
					st.ImageURL = placeholderImageURL(st.Title)
				}
				if err := tx.Create(&st).Error; err != nil {
					return fmt.Errorf("failed to create stop: %w", err)
				}
				savedStops = append(savedStops, st)
			}

			// Create segments; recompute From/To from saved stop IDs by position (N-1 invariant R12)
			for pos, seg := range segments {
				seg.ItineraryID = itin.ID
				seg.Position = pos
				if len(savedStops) > 0 {
					if pos < len(savedStops)-1 {
						seg.FromStopID = savedStops[pos].ID
						seg.ToStopID = savedStops[pos+1].ID
					} else if pos < len(savedStops) {
						// Defensive: if generator provided extra segments, clamp
						seg.FromStopID = savedStops[pos].ID
						if pos+1 < len(savedStops) {
							seg.ToStopID = savedStops[pos+1].ID
						}
					}
				}
				if err := tx.Create(&seg).Error; err != nil {
					return fmt.Errorf("failed to create segment: %w", err)
				}
			}

			// Restore for return value (with IDs)
			itin.Stops = savedStops
			// Reload segments with IDs for completeness
			var savedSegs []Segment
			if err := tx.Where("itinerary_id = ?", itin.ID).Order("position ASC").Find(&savedSegs).Error; err == nil {
				itin.Segments = savedSegs
			} else {
				itin.Segments = segments
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Reload with preloads for return value consistency
	return s.GetTripByID(ctx, t.ID)
}

func placeholderImageURL(title string) string {
	// Deterministic placeholder; real BFL Flux will upgrade in place (R17-R19)
	return "https://picsum.photos/seed/" + urlSafeSeed(title) + "/600/400"
}

func urlSafeSeed(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		s = "walk-the-city"
	}
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	// Keep alphanumeric and dash
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		return "walk-the-city"
	}
	if len(out) > 40 {
		out = out[:40]
	}
	return out
}

func (s *service) ListTrips(ctx context.Context) ([]Trip, error) {
	var trips []Trip
	err := s.db.WithContext(ctx).
		Preload("Itineraries", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC") }).
		Preload("Itineraries.Stops", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC") }).
		Preload("Itineraries.Segments", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC") }).
		Order("created_at DESC").Find(&trips).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list trips: %w", err)
	}
	return trips, nil
}

func (s *service) GetTripByID(ctx context.Context, id uint) (*Trip, error) {
	var t Trip
	err := s.db.WithContext(ctx).
		Preload("Itineraries", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC") }).
		Preload("Itineraries.Stops", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC") }).
		Preload("Itineraries.Segments", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC") }).
		First(&t, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get trip %d: %w", id, err)
	}
	return &t, nil
}

func (s *service) GenerateAndSaveTrip(ctx context.Context, params CreateTripParams, gen TripGenerator) (*Trip, error) {
	if validationErr := params.Validate(); validationErr != nil {
		return nil, validationErr
	}

	if gen != nil {
		itins, err := gen.Generate(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("itinerary generation failed: %w", err)
		}
		params.Itineraries = itins
	}

	return s.CreateTrip(ctx, params)
}
