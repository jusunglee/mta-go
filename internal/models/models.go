package models

import (
	"time"
)

// Location represents a geographic coordinate
type Location struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// Train represents a train arrival
type Train struct {
	Route string    `json:"route"`
	Time  time.Time `json:"time"`
}

// TrainsByDirection groups trains by direction
type TrainsByDirection struct {
	North []Train `json:"N"`
	South []Train `json:"S"`
}

// Station represents a subway station with real-time data
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

// Alert represents a service alert
type Alert struct {
	ID            string       `json:"id"`
	Header        string       `json:"header"`
	Description   string       `json:"description"`
	Routes        []string     `json:"routes"`
	Stations      []string     `json:"stations"`
	ActivePeriods []TimePeriod `json:"active_periods"`
}

// TimePeriod represents a time range
type TimePeriod struct {
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
}

// FeedInfo contains metadata about the feed
type FeedInfo struct {
	LastUpdate time.Time `json:"last_update"`
	Routes     []string  `json:"routes"`
}

// ConvertToResponse converts a Station to StationResponse format
func (s *Station) ConvertToResponse() StationResponse {
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
