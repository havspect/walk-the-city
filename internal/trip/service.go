package trip

import (
	"context"
	"errors"
	"fmt"
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
	var msgs []string
	for field, msg := range v.FieldErrors {
		msgs = append(msgs, fmt.Sprintf("%s: %s", field, msg))
	}
	return strings.Join(msgs, ", ")
}

// CreateTripParams holds inputs for creating a new city trip.
type CreateTripParams struct {
	Destination  string
	DurationDays int
	Notes        string
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

	if len(errs) > 0 {
		return &ValidationError{FieldErrors: errs}
	}
	return nil
}

// TripService defines the business operations for city trip management.
// This interface acts as the decoupled integration seam for HTTP handlers
// and future LLM itinerary generation agents.
type TripService interface {
	CreateTrip(ctx context.Context, params CreateTripParams) (*Trip, error)
	ListTrips(ctx context.Context) ([]Trip, error)
	GetTripByID(ctx context.Context, id uint) (*Trip, error)
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

	t := &Trip{
		Destination:  strings.TrimSpace(params.Destination),
		DurationDays: params.DurationDays,
		Notes:        strings.TrimSpace(params.Notes),
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
