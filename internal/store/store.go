package store

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jusunglee/mta-go/internal/models"
)

// Store manages in-memory station and alert data
type Store struct {
	mu              sync.RWMutex
	stations        map[string]*models.Station
	stationsByRoute map[string][]*models.Station
	alerts          []models.Alert
	lastUpdate      time.Time
	routes          []string
}

// NewStore creates a new store instance
func NewStore() *Store {
	return &Store{
		stations:        make(map[string]*models.Station),
		stationsByRoute: make(map[string][]*models.Station),
		alerts:          []models.Alert{},
	}
}

// UpdateStations updates the station data
func (s *Store) UpdateStations(stations map[string]*models.Station) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stations = stations
	s.lastUpdate = time.Now()

	// Rebuild indices
	s.stationsByRoute = make(map[string][]*models.Station)
	routeSet := make(map[string]bool)

	for _, station := range stations {
		for _, route := range station.Routes {
			s.stationsByRoute[route] = append(s.stationsByRoute[route], station)
			routeSet[route] = true
		}
	}

	// Sort stations by name for each route
	for route := range s.stationsByRoute {
		sort.Slice(s.stationsByRoute[route], func(i, j int) bool {
			return s.stationsByRoute[route][i].Name < s.stationsByRoute[route][j].Name
		})
	}

	// Update routes list
	s.routes = make([]string, 0, len(routeSet))
	for route := range routeSet {
		s.routes = append(s.routes, route)
	}
	sort.Strings(s.routes)
}

// UpdateAlerts updates the service alerts
func (s *Store) UpdateAlerts(alerts []models.Alert) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alerts = alerts
}

// GetStationsByLocation returns stations near a location
func (s *Store) GetStationsByLocation(lat, lon float64, limit int) []models.Station {
	s.mu.RLock()
	defer s.mu.RUnlock()

	type stationDist struct {
		station  *models.Station
		distance float64
	}

	var stations []stationDist
	for _, station := range s.stations {
		dist := distance(lat, lon, station.Location.Lat, station.Location.Lon)
		stations = append(stations, stationDist{station, dist})
	}

	sort.Slice(stations, func(i, j int) bool {
		return stations[i].distance < stations[j].distance
	})

	result := make([]models.Station, 0, limit)
	for i := 0; i < limit && i < len(stations); i++ {
		result = append(result, *stations[i].station)
	}

	return result
}

// GetStationsByRoute returns all stations on a route
func (s *Store) GetStationsByRoute(route string) ([]models.Station, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	route = strings.ToUpper(route)
	stations, ok := s.stationsByRoute[route]
	if !ok {
		return nil, fmt.Errorf("route %s not found", route)
	}

	result := make([]models.Station, len(stations))
	for i, station := range stations {
		result[i] = *station
	}

	return result, nil
}

// GetStationsByIDs returns stations by their IDs
func (s *Store) GetStationsByIDs(ids []string) ([]models.Station, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]models.Station, 0, len(ids))
	for _, id := range ids {
		if station, ok := s.stations[id]; ok {
			result = append(result, *station)
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no stations found for given IDs")
	}

	return result, nil
}

// GetRoutes returns all available routes
func (s *Store) GetRoutes() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]string, len(s.routes))
	copy(result, s.routes)
	return result
}

// GetServiceAlerts returns all active service alerts
func (s *Store) GetServiceAlerts() []models.Alert {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]models.Alert, len(s.alerts))
	copy(result, s.alerts)
	return result
}

// GetLastUpdate returns the last update time
func (s *Store) GetLastUpdate() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastUpdate
}

// distance calculates the distance between two points using the Haversine formula
func distance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth's radius in kilometers

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}
