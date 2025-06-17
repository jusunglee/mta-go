package models

import (
	"time"
)

type Location struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type Train struct {
	Route string    `json:"route"`
	Time  time.Time `json:"time"`
}

// TrainsByDirection separates trains by subway direction (North/South)
// This mirrors the MTA's directional conventions for NYC subway
type TrainsByDirection struct {
	North []Train `json:"N"`
	South []Train `json:"S"`
}

// Station represents a subway station with real-time data
// Trains field uses json:"-" to exclude from JSON serialization - use ConvertToResponse for API output
type Station struct {
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	Location   Location            `json:"location"`
	Routes     []string            `json:"routes"`
	Trains     TrainsByDirection   `json:"-"`
	Stops      map[string]Location `json:"stops"`
	LastUpdate time.Time           `json:"last_update"`
}

// StationResponse is the API response format for a station
// Uses [2]float64 arrays instead of Location structs for more compact JSON output
type StationResponse struct {
	ID         string                `json:"id"`
	Name       string                `json:"name"`
	Location   [2]float64            `json:"location"`
	Routes     []string              `json:"routes"`
	N          []Train               `json:"N"`
	S          []Train               `json:"S"`
	Stops      map[string][2]float64 `json:"stops"`
	LastUpdate time.Time             `json:"last_update"`
}

type Alert struct {
	ID            string       `json:"id"`
	Header        string       `json:"header"`
	Description   string       `json:"description"`
	Routes        []string     `json:"routes"`
	Stations      []string     `json:"stations"`
	ActivePeriods []TimePeriod `json:"active_periods"`
}

// TimePeriod represents a time range
// Uses pointers to allow nil values for open-ended periods
type TimePeriod struct {
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
}

type FeedInfo struct {
	LastUpdate time.Time `json:"last_update"`
	Routes     []string  `json:"routes"`
}

// ConvertToResponse converts internal Station to API response format
// Transforms Location structs to [lat, lon] arrays and expands nested train directions
func (s *Station) ConvertToResponse() StationResponse {
	// Convert Location structs to coordinate arrays for API response
	stops := make(map[string][2]float64)
	for id, loc := range s.Stops {
		stops[id] = [2]float64{loc.Lat, loc.Lon}
	}

	return StationResponse{
		ID:         s.ID,
		Name:       s.Name,
		Location:   [2]float64{s.Location.Lat, s.Location.Lon},
		Routes:     s.Routes,
		N:          s.Trains.North,
		S:          s.Trains.South,
		Stops:      stops,
		LastUpdate: s.LastUpdate,
	}
}
