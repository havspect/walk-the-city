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
	Itinerary    *Itinerary
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

func (s *service) CreateTrip(ctx context.Context, params CreateTripParams) (*Trip, error) {
	if validationErr := params.Validate(); validationErr != nil {
		return nil, validationErr
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

	if params.Itinerary != nil {
		if err := t.SetItinerary(params.Itinerary); err != nil {
			return nil, fmt.Errorf("failed to encode itinerary: %w", err)
		}
	}

	if err := s.db.WithContext(ctx).Create(t).Error; err != nil {
		return nil, fmt.Errorf("failed to create trip: %w", err)
	}

	return t, nil
}

func (s *service) ListTrips(ctx context.Context) ([]Trip, error) {
	var trips []Trip
	err := s.db.WithContext(ctx).Order("created_at DESC").Find(&trips).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list trips: %w", err)
	}
	return trips, nil
}

func (s *service) GetTripByID(ctx context.Context, id uint) (*Trip, error) {
	var t Trip
	err := s.db.WithContext(ctx).First(&t, id).Error
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
		it, err := gen.Generate(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("itinerary generation failed: %w", err)
		}
		params.Itinerary = it
	}

	return s.CreateTrip(ctx, params)
}
