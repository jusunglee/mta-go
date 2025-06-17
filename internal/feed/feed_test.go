package feed

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/jusunglee/mta-go/internal/store"
)

func TestParseGTFSData(t *testing.T) {
	tests := []struct {
		name        string
		gtfsDir     string
		expectError bool
	}{
		{
			name:        "parse regular GTFS data",
			gtfsDir:     "testdata/gtfs_subway",
			expectError: false,
		},
		{
			name:        "parse supplemented GTFS data",
			gtfsDir:     "testdata/gtfs_supplemented",
			expectError: false,
		},
		{
			name:        "missing directory should fail",
			gtfsDir:     "testdata/nonexistent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a manager with a test store
			s := store.NewStore()
			m := &Manager{
				store: s,
			}

			err := m.parseGTFSData(tt.gtfsDir)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify stations were loaded
			routes := s.GetRoutes()
			if len(routes) == 0 {
				t.Error("No routes found after parsing GTFS data")
			}

			// Test a few stations by location (near major NYC hubs)
			stations := s.GetStationsByLocation(40.755, -73.987, 5) // Near Times Square
			if len(stations) == 0 {
				t.Error("No stations found near Times Square")
			}

			// Verify station data structure
			for _, station := range stations[:1] { // Test first station
				if station.ID == "" {
					t.Error("Station missing ID")
				}
				if station.Name == "" {
					t.Error("Station missing name")
				}
				if station.Location.Lat == 0 || station.Location.Lon == 0 {
					t.Error("Station missing valid coordinates")
				}
				if station.LastUpdate.IsZero() {
					t.Error("Station missing last update time")
				}
				// Routes should be populated by parseRoutes
				t.Logf("Station %s (%s) has %d routes: %v", 
					station.Name, station.ID, len(station.Routes), station.Routes)
			}
		})
	}
}

func TestParseStops(t *testing.T) {
	tests := []struct {
		name        string
		stopsFile   string
		expectError bool
		minStations int
	}{
		{
			name:        "parse regular GTFS stops",
			stopsFile:   "testdata/gtfs_subway/stops.txt",
			expectError: false,
			minStations: 100, // NYC subway has 400+ stations, so 100 is conservative
		},
		{
			name:        "parse supplemented GTFS stops",
			stopsFile:   "testdata/gtfs_supplemented/stops.txt",
			expectError: false,
			minStations: 100,
		},
		{
			name:        "missing file should fail",
			stopsFile:   "testdata/nonexistent.txt",
			expectError: true,
			minStations: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{}
			stations, err := m.parseStops(tt.stopsFile)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(stations) < tt.minStations {
				t.Errorf("Expected at least %d stations, got %d", tt.minStations, len(stations))
			}

			// Verify station data structure
			count := 0
			for stationID, station := range stations {
				if count >= 3 { // Test first few stations
					break
				}

				if station.ID != stationID {
					t.Errorf("Station ID mismatch: map key %s != station.ID %s", stationID, station.ID)
				}

				if station.Name == "" {
					t.Errorf("Station %s missing name", stationID)
				}

				if station.Location.Lat == 0 || station.Location.Lon == 0 {
					t.Errorf("Station %s missing valid coordinates: lat=%f, lon=%f", 
						stationID, station.Location.Lat, station.Location.Lon)
				}

				if len(station.Stops) == 0 {
					t.Errorf("Station %s missing stops", stationID)
				}

				// Verify stop coordinates are reasonable (NYC area)
				for stopID, location := range station.Stops {
					if location.Lat < 40.4 || location.Lat > 41.0 {
						t.Errorf("Stop %s has invalid latitude: %f", stopID, location.Lat)
					}
					if location.Lon < -74.5 || location.Lon > -73.5 {
						t.Errorf("Stop %s has invalid longitude: %f", stopID, location.Lon)
					}
				}

				count++
			}

			t.Logf("Successfully parsed %d stations from %s", len(stations), tt.stopsFile)
		})
	}
}

func TestParseRoutesFile(t *testing.T) {
	tests := []struct {
		name        string
		routesFile  string
		expectError bool
		expectedRoutes []string // Sample routes we expect to find
	}{
		{
			name:        "parse regular GTFS routes",
			routesFile:  "testdata/gtfs_subway/routes.txt",
			expectError: false,
			expectedRoutes: []string{"1", "2", "3", "4", "5", "6", "7", "A", "B", "C", "D", "E", "F", "G", "J", "L", "M", "N", "Q", "R", "W", "Z"},
		},
		{
			name:        "parse supplemented GTFS routes",
			routesFile:  "testdata/gtfs_supplemented/routes.txt",
			expectError: false,
			expectedRoutes: []string{"1", "2", "3", "4", "5", "6", "7", "A", "B", "C", "D", "E", "F", "G", "J", "L", "M", "N", "Q", "R", "W", "Z"},
		},
		{
			name:        "missing file should fail",
			routesFile:  "testdata/nonexistent.txt",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{}
			routes, err := m.parseRoutesFile(tt.routesFile)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(routes) == 0 {
				t.Error("No routes found")
			}

			// Check that we found expected routes
			routeNames := make(map[string]bool)
			for _, routeName := range routes {
				routeNames[routeName] = true
			}

			foundCount := 0
			for _, expectedRoute := range tt.expectedRoutes {
				if routeNames[expectedRoute] {
					foundCount++
				}
			}

			// We should find most of the expected routes (allowing for service changes)
			if foundCount < len(tt.expectedRoutes)/2 {
				t.Errorf("Found only %d of %d expected routes. Routes found: %v", 
					foundCount, len(tt.expectedRoutes), routes)
			}

			t.Logf("Successfully parsed %d routes, found %d/%d expected routes", 
				len(routes), foundCount, len(tt.expectedRoutes))
		})
	}
}

func TestParseTripsFile(t *testing.T) {
	tests := []struct {
		name        string
		tripsFile   string
		expectError bool
		minTrips    int
	}{
		{
			name:        "parse regular GTFS trips",
			tripsFile:   "testdata/gtfs_subway/trips.txt",
			expectError: false,
			minTrips:    1000, // Conservative estimate
		},
		{
			name:        "parse supplemented GTFS trips",
			tripsFile:   "testdata/gtfs_supplemented/trips.txt",
			expectError: false,
			minTrips:    1000,
		},
		{
			name:        "missing file should fail",
			tripsFile:   "testdata/nonexistent.txt",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{}
			routeTrips, err := m.parseTripsFile(tt.tripsFile)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			totalTrips := 0
			for routeID, trips := range routeTrips {
				totalTrips += len(trips)
				if len(trips) == 0 {
					t.Errorf("Route %s has no trips", routeID)
				}
			}

			if totalTrips < tt.minTrips {
				t.Errorf("Expected at least %d trips, got %d", tt.minTrips, totalTrips)
			}

			t.Logf("Successfully parsed %d routes with %d total trips", len(routeTrips), totalTrips)
		})
	}
}

func TestParseStopTimesFile(t *testing.T) {
	tests := []struct {
		name          string
		stopTimesFile string
		expectError   bool
		minStops      int
	}{
		{
			name:          "parse regular GTFS stop times",
			stopTimesFile: "testdata/gtfs_subway/stop_times.txt",
			expectError:   false,
			minStops:      10000, // Conservative estimate - this file is huge
		},
		{
			name:          "parse supplemented GTFS stop times",
			stopTimesFile: "testdata/gtfs_supplemented/stop_times.txt",
			expectError:   false,
			minStops:      10000,
		},
		{
			name:          "missing file should fail",
			stopTimesFile: "testdata/nonexistent.txt",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{}
			
			// Set a timeout for this test since stop_times.txt can be very large
			start := time.Now()
			
			tripStops, err := m.parseStopTimesFile(tt.stopTimesFile)
			
			elapsed := time.Since(start)
			t.Logf("Parsing took %v", elapsed)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			totalStops := 0
			for tripID, stops := range tripStops {
				totalStops += len(stops)
				if len(stops) == 0 {
					t.Errorf("Trip %s has no stops", tripID)
				}
			}

			if totalStops < tt.minStops {
				t.Errorf("Expected at least %d stop times, got %d", tt.minStops, totalStops)
			}

			t.Logf("Successfully parsed %d trips with %d total stop times", len(tripStops), totalStops)
		})
	}
}

func TestParseRoutes(t *testing.T) {
	tests := []struct {
		name        string
		gtfsDir     string
		expectError bool
	}{
		{
			name:        "associate routes with stations (regular GTFS)",
			gtfsDir:     "testdata/gtfs_subway",
			expectError: false,
		},
		{
			name:        "associate routes with stations (supplemented GTFS)",
			gtfsDir:     "testdata/gtfs_supplemented",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{}

			// First parse stops to get stations
			stations, err := m.parseStops(filepath.Join(tt.gtfsDir, "stops.txt"))
			if err != nil {
				t.Fatalf("Failed to parse stops: %v", err)
			}

			// Store original route counts (should be empty)
			originalRouteCounts := make(map[string]int)
			for stationID, station := range stations {
				originalRouteCounts[stationID] = len(station.Routes)
			}

			// Now associate routes
			err = m.parseRoutes(filepath.Join(tt.gtfsDir, "routes.txt"), stations)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify that stations now have routes
			stationsWithRoutes := 0
			totalRoutes := 0

			for stationID, station := range stations {
				if len(station.Routes) > originalRouteCounts[stationID] {
					stationsWithRoutes++
					totalRoutes += len(station.Routes)
				}
			}

			if stationsWithRoutes == 0 {
				t.Error("No stations have routes after parsing")
			}

			// Sample a few stations and log their routes
			count := 0
			for stationID, station := range stations {
				if len(station.Routes) > 0 && count < 5 {
					t.Logf("Station %s (%s) serves routes: %v", 
						station.Name, stationID, station.Routes)
					count++
				}
			}

			t.Logf("Successfully associated routes with %d/%d stations (total %d route assignments)", 
				stationsWithRoutes, len(stations), totalRoutes)
		})
	}
}

// Benchmark the most expensive operations
func BenchmarkParseStopTimes(b *testing.B) {
	m := &Manager{}
	stopTimesFile := "testdata/gtfs_subway/stop_times.txt"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := m.parseStopTimesFile(stopTimesFile)
		if err != nil {
			b.Fatalf("Error in benchmark: %v", err)
		}
	}
}

func BenchmarkParseGTFSData(b *testing.B) {
	gtfsDir := "testdata/gtfs_subway"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := store.NewStore()
		m := &Manager{store: s}
		
		err := m.parseGTFSData(gtfsDir)
		if err != nil {
			b.Fatalf("Error in benchmark: %v", err)
		}
	}
}

func TestStaticDataRefresh(t *testing.T) {
	s := store.NewStore()
	m := NewManager("test-key", s, time.Minute)
	
	// Test that refresh can be disabled
	m.SetStaticUpdateInterval(0)
	if m.staticUpdateInterval != 0 {
		t.Error("SetStaticUpdateInterval(0) should disable refresh")
	}
	
	// Test that refresh interval can be configured
	m.SetStaticUpdateInterval(2 * time.Hour)
	if m.staticUpdateInterval != 2*time.Hour {
		t.Errorf("Expected 2 hours, got %v", m.staticUpdateInterval)
	}
	
	// Test default interval
	m2 := NewManager("test-key", s, time.Minute)
	if m2.staticUpdateInterval != 6*time.Hour {
		t.Errorf("Expected default 6 hours, got %v", m2.staticUpdateInterval)
	}
	
	// Test GetLastStaticUpdate before any updates
	if !m.GetLastStaticUpdate().IsZero() {
		t.Error("GetLastStaticUpdate should return zero time before any updates")
	}
}