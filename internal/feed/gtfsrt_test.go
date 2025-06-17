package feed

import (
	"testing"
	"time"

	"github.com/jusunglee/mta-go/internal/gtfsrt"
	"github.com/jusunglee/mta-go/internal/models"
	"github.com/jusunglee/mta-go/internal/store"
)

func TestExtractRouteFromID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"A20241201", "A"},
		{"N20241201", "N"},
		{"123_20241201", "123_"},
		{"1", "1"},
		{"A", "A"},
		{"", ""},
		{"123", "123"},
	}

	m := &Manager{}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := m.extractRouteFromID(tt.input)
			if result != tt.expected {
				t.Errorf("extractRouteFromID(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSortAndLimitTrains(t *testing.T) {
	now := time.Now()
	m := &Manager{}
	
	trains := []models.Train{
		{Route: "N", Time: now.Add(5 * time.Minute)},
		{Route: "Q", Time: now.Add(2 * time.Minute)},
		{Route: "N", Time: now.Add(5 * time.Minute)}, // Duplicate
		{Route: "R", Time: now.Add(1 * time.Minute)},
		{Route: "W", Time: now.Add(10 * time.Minute)},
	}

	result := m.sortAndLimitTrains(trains)

	// Should have 4 unique trains (removed 1 duplicate)
	if len(result) != 4 {
		t.Errorf("Expected 4 trains after deduplication, got %d", len(result))
	}

	// Should be sorted by time (R, Q, N, W)
	expectedOrder := []string{"R", "Q", "N", "W"}
	for i, train := range result {
		if train.Route != expectedOrder[i] {
			t.Errorf("Train %d: expected route %s, got %s", i, expectedOrder[i], train.Route)
		}
	}

	// Test with empty slice
	empty := m.sortAndLimitTrains([]models.Train{})
	if len(empty) != 0 {
		t.Error("Empty slice should remain empty")
	}
}

func TestProcessTripUpdate(t *testing.T) {
	m := &Manager{}
	
	// Create test stations
	stations := map[string]*models.Station{
		"R16": {
			ID:   "R16",
			Name: "Times Sq-42 St",
			Trains: models.TrainsByDirection{
				North: []models.Train{},
				South: []models.Train{},
			},
		},
	}

	// Create test trip update
	routeID := "N20241201"
	stopID := "R16N"
	arrivalTime := time.Now().Add(3 * time.Minute).Unix()
	
	tripUpdate := &gtfsrt.TripUpdate{
		Trip: &gtfsrt.TripDescriptor{
			RouteId: &routeID,
		},
		StopTimeUpdate: []*gtfsrt.StopTimeUpdate{
			{
				StopId: &stopID,
				Arrival: &gtfsrt.StopTimeEvent{
					Time: &arrivalTime,
				},
			},
		},
	}

	// Process the trip update
	m.processTripUpdate(tripUpdate, stations)

	// Verify the train was added
	station := stations["R16"]
	if len(station.Trains.North) != 1 {
		t.Errorf("Expected 1 northbound train, got %d", len(station.Trains.North))
	}

	if len(station.Trains.South) != 0 {
		t.Errorf("Expected 0 southbound trains, got %d", len(station.Trains.South))
	}

	train := station.Trains.North[0]
	if train.Route != "N" {
		t.Errorf("Expected route N, got %s", train.Route)
	}

	expectedTime := time.Unix(arrivalTime, 0)
	if !train.Time.Equal(expectedTime) {
		t.Errorf("Expected time %v, got %v", expectedTime, train.Time)
	}
}

func TestProcessAlert(t *testing.T) {
	// Create a real store for the manager
	s := store.NewStore()
	m := &Manager{store: s}
	
	headerText := "Service Alert"
	descriptionText := "Delays on N line"
	routeID := "N20241201"
	
	alert := &gtfsrt.Alert{
		HeaderText: &gtfsrt.TranslatedString{
			Translation: []*gtfsrt.TranslatedString_Translation{
				{
					Text: &headerText,
				},
			},
		},
		DescriptionText: &gtfsrt.TranslatedString{
			Translation: []*gtfsrt.TranslatedString_Translation{
				{
					Text: &descriptionText,
				},
			},
		},
		InformedEntity: []*gtfsrt.EntitySelector{
			{
				RouteId: &routeID,
			},
		},
	}

	// Process the alert
	m.processAlert(alert)
	
	// Verify the alert was processed
	alerts := s.GetServiceAlerts()
	if len(alerts) != 1 {
		t.Errorf("Expected 1 alert, got %d", len(alerts))
	}
	
	processedAlert := alerts[0]
	if processedAlert.Header != headerText {
		t.Errorf("Expected header %q, got %q", headerText, processedAlert.Header)
	}
	
	if processedAlert.Description != descriptionText {
		t.Errorf("Expected description %q, got %q", descriptionText, processedAlert.Description)
	}
	
	if len(processedAlert.Routes) != 1 || processedAlert.Routes[0] != "N" {
		t.Errorf("Expected routes [N], got %v", processedAlert.Routes)
	}
}

