package models

import (
	"testing"
	"time"
)

func TestStationConvertToResponse(t *testing.T) {
	station := &Station{
		ID:       "123",
		Name:     "Test Station",
		Location: Location{Lat: 40.755, Lon: -73.987},
		Routes:   []string{"N", "Q", "R"},
		Trains: TrainsByDirection{
			North: []Train{
				{Route: "N", Time: time.Now().Add(5 * time.Minute)},
				{Route: "Q", Time: time.Now().Add(10 * time.Minute)},
			},
			South: []Train{
				{Route: "R", Time: time.Now().Add(7 * time.Minute)},
			},
		},
		Stops: map[string]Location{
			"123N": {Lat: 40.756, Lon: -73.987},
			"123S": {Lat: 40.754, Lon: -73.987},
		},
		LastUpdate: time.Now(),
	}

	response := station.ConvertToResponse()

	// Check basic fields
	if response.ID != station.ID {
		t.Errorf("Expected ID %s, got %s", station.ID, response.ID)
	}
	if response.Name != station.Name {
		t.Errorf("Expected Name %s, got %s", station.Name, response.Name)
	}

	// Check location conversion
	if response.Location[0] != station.Location.Lat || response.Location[1] != station.Location.Lon {
		t.Errorf("Location mismatch: expected [%f, %f], got %v",
			station.Location.Lat, station.Location.Lon, response.Location)
	}

	// Check routes
	if len(response.Routes) != len(station.Routes) {
		t.Errorf("Expected %d routes, got %d", len(station.Routes), len(response.Routes))
	}

	// Check trains
	if len(response.N) != len(station.Trains.North) {
		t.Errorf("Expected %d northbound trains, got %d",
			len(station.Trains.North), len(response.N))
	}
	if len(response.S) != len(station.Trains.South) {
		t.Errorf("Expected %d southbound trains, got %d",
			len(station.Trains.South), len(response.S))
	}

	// Check stops conversion
	if len(response.Stops) != len(station.Stops) {
		t.Errorf("Expected %d stops, got %d", len(station.Stops), len(response.Stops))
	}
	for id, loc := range station.Stops {
		respLoc, exists := response.Stops[id]
		if !exists {
			t.Errorf("Stop %s missing in response", id)
			continue
		}
		if respLoc[0] != loc.Lat || respLoc[1] != loc.Lon {
			t.Errorf("Stop %s location mismatch", id)
		}
	}
}

func TestTimePeriod(t *testing.T) {
	now := time.Now()
	future := now.Add(1 * time.Hour)

	period := TimePeriod{
		Start: &now,
		End:   &future,
	}

	if period.Start == nil || period.End == nil {
		t.Error("TimePeriod pointers should not be nil")
	}

	if !period.End.After(*period.Start) {
		t.Error("End time should be after start time")
	}
}
