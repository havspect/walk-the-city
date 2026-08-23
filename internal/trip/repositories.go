package trip

import (
	"context"
	"errors"
	"fmt"

	"github.com/havspect/walk-the-city/internal/repository"
	"gorm.io/gorm"
)

// TripRepository is the per-model persistence layer for Trip.
// It composes the generic Repository and adds the single ordered-preload chain.
type TripRepository struct {
	*repository.Repository[Trip]
}

// NewTripRepository creates a new TripRepository.
func NewTripRepository(db *gorm.DB) *TripRepository {
	return &TripRepository{Repository: repository.New[Trip](db)}
}

// orderedPreloads returns a query with the canonical ordered preloads.
// The preload chain exists exactly once, here, and is shared by both read methods.
func (r *TripRepository) orderedPreloads() gorm.ChainInterface[Trip] {
	return gorm.G[Trip](r.DB()).
		Preload("Itineraries", func(db gorm.PreloadBuilder) error {
			db.Order("position ASC")
			return nil
		}).
		Preload("Itineraries.Stops", func(db gorm.PreloadBuilder) error {
			db.Order("position ASC")
			return nil
		}).
		Preload("Itineraries.Segments", func(db gorm.PreloadBuilder) error {
			db.Order("position ASC")
			return nil
		})
}

// GetTripByID returns a trip with all itineraries, stops, and segments
// ordered by position. It maps gorm.ErrRecordNotFound to ErrNotFound.
func (r *TripRepository) GetTripByID(ctx context.Context, id uint) (*Trip, error) {
	t, err := r.orderedPreloads().Where("id = ?", id).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get trip %d: %w", id, err)
	}
	return &t, nil
}

// ListTrips returns all trips ordered by created_at DESC with the same preloads.
func (r *TripRepository) ListTrips(ctx context.Context) ([]Trip, error) {
	trips, err := r.orderedPreloads().Order("created_at DESC").Find(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list trips: %w", err)
	}
	return trips, nil
}

// ItineraryRepository is a thin composition of ChildRepository for Itinerary.
type ItineraryRepository struct {
	*repository.ChildRepository[Itinerary]
}

// NewItineraryRepository creates a new ItineraryRepository.
func NewItineraryRepository(db *gorm.DB) *ItineraryRepository {
	return &ItineraryRepository{ChildRepository: repository.NewChild[Itinerary](db, "trip_id")}
}

// StopRepository is a thin composition of ChildRepository for Stop.
type StopRepository struct {
	*repository.ChildRepository[Stop]
}

// NewStopRepository creates a new StopRepository.
func NewStopRepository(db *gorm.DB) *StopRepository {
	return &StopRepository{ChildRepository: repository.NewChild[Stop](db, "itinerary_id")}
}

// SegmentRepository is a thin composition of ChildRepository for Segment.
type SegmentRepository struct {
	*repository.ChildRepository[Segment]
}

// NewSegmentRepository creates a new SegmentRepository.
func NewSegmentRepository(db *gorm.DB) *SegmentRepository {
	return &SegmentRepository{ChildRepository: repository.NewChild[Segment](db, "itinerary_id")}
}
