package trip

import "context"

// TripGenerator defines the contract for generating structured city itineraries.
type TripGenerator interface {
	Generate(ctx context.Context, params CreateTripParams) (*Itinerary, error)
}
