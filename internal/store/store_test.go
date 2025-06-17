package store

import (
	"testing"
	"time"

	"github.com/jusunglee/mta-go/internal/models"
)

func TestStore(t *testing.T) {
	s := NewStore()

	// Test data
	stations := map[string]*models.Station{
		"123": {
			ID:       "123",
			Name:     "Times Square",
			Location: models.Location{Lat: 40.755, Lon: -73.987},
			Routes:   []string{"N", "Q", "R", "W", "S", "1", "2", "3", "7"},
			Stops:    make(map[string]models.Location),
		},
		"456": {
			ID:       "456",
			Name:     "Grand Central",
			Location: models.Location{Lat: 40.752, Lon: -73.977},
			Routes:   []string{"4", "5", "6", "S"},
			Stops:    make(map[string]models.Location),
		},
		"789": {
			ID:       "789",
			Name:     "Union Square",
			Location: models.Location{Lat: 40.735, Lon: -73.990},
			Routes:   []string{"N", "Q", "R", "W", "4", "5", "6", "L"},
			Stops:    make(map[string]models.Location),
		},
	}

	// Update stations
	s.UpdateStations(stations)

	t.Run("GetStationsByLocation", func(t *testing.T) {
		// Near Times Square
		results := s.GetStationsByLocation(40.755, -73.987, 2)
		if len(results) != 2 {
			t.Errorf("Expected 2 stations, got %d", len(results))
		}
		if results[0].ID != "123" {
			t.Errorf("Expected nearest station to be 123, got %s", results[0].ID)
		}
	})

	t.Run("GetStationsByRoute", func(t *testing.T) {
		results, err := s.GetStationsByRoute("N")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("Expected 2 stations on route N, got %d", len(results))
		}

		// Test non-existent route
		_, err = s.GetStationsByRoute("X")
		if err == nil {
			t.Error("Expected error for non-existent route")
		}
	})

	t.Run("GetStationsByIDs", func(t *testing.T) {
		results, err := s.GetStationsByIDs([]string{"123", "456"})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("Expected 2 stations, got %d", len(results))
		}

		// Test non-existent IDs
		_, err = s.GetStationsByIDs([]string{"999"})
		if err == nil {
			t.Error("Expected error for non-existent IDs")
		}
	})

	t.Run("GetRoutes", func(t *testing.T) {
		routes := s.GetRoutes()
		expectedRoutes := []string{"1", "2", "3", "4", "5", "6", "7", "L", "N", "Q", "R", "S", "W"}
		if len(routes) != len(expectedRoutes) {
			t.Errorf("Expected %d routes, got %d", len(expectedRoutes), len(routes))
		}
	})

	t.Run("UpdateAlerts", func(t *testing.T) {
		alerts := []models.Alert{
			{
				ID:          "alert1",
				Header:      "Test Alert",
				Description: "Test Description",
				Routes:      []string{"N", "Q"},
			},
		}
		s.UpdateAlerts(alerts)

		retrievedAlerts := s.GetServiceAlerts()
		if len(retrievedAlerts) != 1 {
			t.Errorf("Expected 1 alert, got %d", len(retrievedAlerts))
		}
		if retrievedAlerts[0].Header != "Test Alert" {
			t.Errorf("Expected alert header 'Test Alert', got '%s'", retrievedAlerts[0].Header)
		}
	})

	t.Run("GetLastUpdate", func(t *testing.T) {
		lastUpdate := s.GetLastUpdate()
		if time.Since(lastUpdate) > time.Minute {
			t.Error("Last update time is too old")
		}
	})
}

func TestDistance(t *testing.T) {
	// Test distance calculation
	// Times Square to Grand Central (approximately 0.97 km)
	dist := distance(40.755, -73.987, 40.752, -73.977)
	if dist < 0.9 || dist > 1.1 {
		t.Errorf("Expected distance ~1.0 km, got %.2f km", dist)
	}

	// Same location
	dist = distance(40.755, -73.987, 40.755, -73.987)
	if dist != 0 {
		t.Errorf("Expected distance 0, got %.2f", dist)
	}
}
