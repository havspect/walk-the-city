package trip

import "context"

// TripGenerator defines the contract for generating structured city itineraries.
// It returns one or more themed itineraries per trip (R20), each with Stops and Segments.
type TripGenerator interface {
	Generate(ctx context.Context, params CreateTripParams) ([]*Itinerary, error)
}
