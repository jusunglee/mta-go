package handlers

import (
	"testing"
	"time"

	"github.com/jusunglee/mta-go/internal/models"
)

// MockClient implements mta.Client for testing
type MockClient struct{}

func (m *MockClient) GetStationsByLocation(lat, lon float64, limit int) ([]models.Station, error) {
	return []models.Station{}, nil
}

func (m *MockClient) GetStationsByRoute(route string) ([]models.Station, error) {
	return []models.Station{}, nil
}

func (m *MockClient) GetStationsByIDs(ids []string) ([]models.Station, error) {
	return []models.Station{}, nil
}

func (m *MockClient) GetRoutes() ([]string, error) {
	return []string{"A", "B", "C"}, nil
}

func (m *MockClient) GetServiceAlerts() ([]models.Alert, error) {
	return []models.Alert{}, nil
}

func (m *MockClient) GetLastUpdate() time.Time {
	return time.Now()
}

func (m *MockClient) GetLastStaticUpdate() time.Time {
	return time.Now().Add(-1 * time.Hour)
}

func TestResponseTypes(t *testing.T) {
	client := &MockClient{}
	h := NewHandler(client)

	// Test that response metadata is populated correctly
	meta := h.getResponseMetadata()
	
	if meta.Updated == "" {
		t.Error("Expected Updated timestamp to be set")
	}
	
	if meta.StaticDataUpdated == "" {
		t.Error("Expected StaticDataUpdated timestamp to be set")
	}
	
	// Test that response types are properly typed
	routes, _ := client.GetRoutes()
	routesResponse := RoutesResponse{
		Data:             routes,
		ResponseMetadata: meta,
	}
	
	// Verify type safety - this wouldn't compile if types were wrong
	if len(routesResponse.Data) != 3 {
		t.Errorf("Expected 3 routes, got %d", len(routesResponse.Data))
	}
	
	// Test stations response structure
	stations := []models.Station{}
	stationsResponse := StationsResponse{
		Data:             make([]models.StationResponse, len(stations)),
		ResponseMetadata: meta,
	}
	
	if len(stationsResponse.Data) != 0 {
		t.Errorf("Expected 0 stations, got %d", len(stationsResponse.Data))
	}
}