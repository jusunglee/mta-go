package feed

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/jusunglee/mta-go/internal/gtfsrt"
	"github.com/jusunglee/mta-go/internal/models"
	"github.com/jusunglee/mta-go/internal/store"
	"google.golang.org/protobuf/proto"
)

// GTFS static data URLs from MTA
const (
	// Regular GTFS: Normal subway schedule, updated a few times per year
	GTFSRegularURL = "https://rrgtfsfeeds.s3.amazonaws.com/gtfs_subway.zip"
	// Supplemented GTFS: Includes service changes for next 7 days, updated hourly
	GTFSSupplementedURL = "https://rrgtfsfeeds.s3.amazonaws.com/gtfs_supplemented.zip"
)

// FeedURLs for NYC Subway GTFS-RT feeds
// Each URL corresponds to different subway lines as per MTA's feed grouping
var FeedURLs = []string{
	"https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs",      // 1234567S
	"https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-l",    // L
	"https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-nqrw", // NRQW
	"https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-bdfm", // BDFM
	"https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-ace",  // ACE
	"https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-jz",   // JZ
	"https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-g",    // G
}

// Manager handles feed fetching and processing
// Runs background goroutine to periodically fetch and parse MTA GTFS-RT data
type Manager struct {
	apiKey               string
	store                *store.Store
	updateInterval       time.Duration
	staticUpdateInterval time.Duration // How often to refresh static GTFS data
	httpClient           *http.Client
	stopCh               chan struct{}
	wg                   sync.WaitGroup
	gtfsDataDir          string    // Directory to store GTFS static data
	staticsLoaded        bool      // Track if static data has been loaded
	lastStaticUpdate     time.Time // When static data was last successfully updated
}

func NewManager(apiKey string, store *store.Store, updateInterval time.Duration) *Manager {
	return &Manager{
		apiKey:               apiKey,
		store:                store,
		updateInterval:       updateInterval,
		staticUpdateInterval: 6 * time.Hour, // Refresh static data every 6 hours
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		stopCh:      make(chan struct{}),
		gtfsDataDir: "data/gtfs", // Default directory for GTFS data
	}
}

// SetStaticUpdateInterval configures how often static GTFS data is refreshed
// Default is 6 hours. Set to 0 to disable automatic refresh (only load once).
func (m *Manager) SetStaticUpdateInterval(interval time.Duration) {
	m.staticUpdateInterval = interval
}

// GetLastStaticUpdate returns when static GTFS data was last successfully updated
// Returns zero time if static data hasn't been loaded yet
func (m *Manager) GetLastStaticUpdate() time.Time {
	return m.lastStaticUpdate
}

func (m *Manager) Start() {
	m.wg.Add(1)
	go m.updateLoop()
}

// Stop gracefully shuts down the feed update loop
// Waits for current update to complete before returning
func (m *Manager) Stop() {
	close(m.stopCh)
	m.wg.Wait()
}

func (m *Manager) updateLoop() {
	defer m.wg.Done()

	// Fetch initial data before starting periodic updates
	if err := m.update(); err != nil {
		slog.Error("Initial update failed", "error", err)
	}

	ticker := time.NewTicker(m.updateInterval)
	defer ticker.Stop()

	// Main update loop - select pattern for clean shutdown
	for {
		select {
		case <-ticker.C:
			if err := m.update(); err != nil {
				slog.Error("Update failed", "error", err)
			}
		case <-m.stopCh:
			return
		}
	}
}

func (m *Manager) update() error {
	// Load static GTFS data on first run OR if enough time has passed
	needsStaticUpdate := !m.staticsLoaded || 
		(m.staticUpdateInterval > 0 && !m.lastStaticUpdate.IsZero() && time.Since(m.lastStaticUpdate) > m.staticUpdateInterval)
	
	if needsStaticUpdate {
		if err := m.loadStaticGTFSData(); err != nil {
			if !m.staticsLoaded {
				// First load failed - this is critical
				return fmt.Errorf("failed to load initial static GTFS data: %w", err)
			}
			// Refresh failed but we have existing data - log warning and continue
			slog.Warn("Failed to refresh static GTFS data, continuing with existing data", 
				"error", err, "last_update", m.lastStaticUpdate)
		} else {
			// Success - update tracking variables
			m.staticsLoaded = true
			m.lastStaticUpdate = time.Now()
			slog.Info("Successfully refreshed static GTFS data", "update_time", m.lastStaticUpdate)
		}
	}

	// Fetch real-time data from all GTFS-RT feeds
	if err := m.updateRealTimeData(); err != nil {
		slog.Warn("Failed to update real-time data", "error", err)
		// Don't return error - static data should still be available
	}

	return nil
}

// updateRealTimeData fetches and processes GTFS-RT feeds for live train data
func (m *Manager) updateRealTimeData() error {
	// Get current stations from store to update with real-time data
	stations := make(map[string]*models.Station)

	// Get all routes and fetch stations for each to build a map
	routes := m.store.GetRoutes()
	for _, route := range routes {
		routeStations, err := m.store.GetStationsByRoute(route)
		if err != nil {
			continue
		}
		for _, station := range routeStations {
			if _, exists := stations[station.ID]; !exists {
				// Create a copy to avoid modifying store data directly
				stationCopy := station
				stationCopy.Trains = models.TrainsByDirection{
					North: []models.Train{},
					South: []models.Train{},
				}
				stations[station.ID] = &stationCopy
			}
		}
	}

	// Process each GTFS-RT feed
	for _, feedURL := range FeedURLs {
		if err := m.processFeed(feedURL, stations); err != nil {
			slog.Warn("Failed to process feed", "url", feedURL, "error", err)
			// Continue with other feeds
		}
	}

	// Sort and clean up train arrivals for each station
	for _, station := range stations {
		station.Trains.North = m.sortAndLimitTrains(station.Trains.North)
		station.Trains.South = m.sortAndLimitTrains(station.Trains.South)
		station.LastUpdate = time.Now()
	}

	// Update store with real-time data
	m.store.UpdateStations(stations)

	return nil
}

// processFeed fetches and parses a single GTFS-RT feed
func (m *Manager) processFeed(feedURL string, stations map[string]*models.Station) error {
	// Fetch the protobuf data
	data, err := m.fetchFeed(feedURL)
	if err != nil {
		return fmt.Errorf("failed to fetch feed: %w", err)
	}

	// Parse the protobuf message
	var feedMessage gtfsrt.FeedMessage
	if err := proto.Unmarshal(data, &feedMessage); err != nil {
		return fmt.Errorf("failed to unmarshal protobuf: %w", err)
	}

	// Process each entity in the feed
	for _, entity := range feedMessage.Entity {
		if entity.TripUpdate != nil {
			m.processTripUpdate(entity.TripUpdate, stations)
		}
		if entity.Alert != nil {
			m.processAlert(entity.Alert)
		}
	}

	return nil
}

// processTripUpdate processes a GTFS-RT trip update to extract arrival times
func (m *Manager) processTripUpdate(tripUpdate *gtfsrt.TripUpdate, stations map[string]*models.Station) error {
	if tripUpdate.Trip == nil || tripUpdate.Trip.RouteId == nil {
		return fmt.Errorf("trip update is missing required fields")
	}

	routeID := *tripUpdate.Trip.RouteId

	// Convert GTFS route ID to route name (e.g., "A20241201" -> "A")
	routeName := m.extractRouteFromID(routeID)
	if routeName == "" {
		return fmt.Errorf("invalid route ID: %s", routeID)
	}

	// Process each stop time update
	for _, stopTimeUpdate := range tripUpdate.StopTimeUpdate {
		if stopTimeUpdate.StopId == nil || stopTimeUpdate.Arrival == nil {
			return fmt.Errorf("stop time update is missing required fields")
		}

		stopID := *stopTimeUpdate.StopId

		// Extract parent station ID (remove direction suffix)
		parentStationID := stopID
		direction := ""
		if len(stopID) > 0 {
			lastChar := stopID[len(stopID)-1]
			if lastChar == 'N' || lastChar == 'S' {
				parentStationID = stopID[:len(stopID)-1]
				if lastChar == 'N' {
					direction = "North"
				} else {
					direction = "South"
				}
			}
		}

		// Find the station
		station, exists := stations[parentStationID]
		if !exists {
			return fmt.Errorf("station not found: %s", parentStationID)
		}

		// Calculate arrival time
		var arrivalTime time.Time
		if stopTimeUpdate.Arrival.Time != nil {
			arrivalTime = time.Unix(*stopTimeUpdate.Arrival.Time, 0)
		} else if stopTimeUpdate.Arrival.Delay != nil {
			// If only delay is provided, add it to current time
			// This is a simplification - ideally we'd use scheduled time + delay
			arrivalTime = time.Now().Add(time.Duration(*stopTimeUpdate.Arrival.Delay) * time.Second)
		} else {
			return fmt.Errorf("no usable time data")
		}

		// Skip past arrivals (more than 1 minute ago)
		if time.Since(arrivalTime) > time.Minute {
			return fmt.Errorf("arrival time is more than 1 minute ago")
		}

		// Create train arrival
		train := models.Train{
			Route: routeName,
			Time:  arrivalTime,
		}

		// Add to appropriate direction
		switch direction {
		case "North":
			station.Trains.North = append(station.Trains.North, train)
		case "South":
			station.Trains.South = append(station.Trains.South, train)
		default:
			return fmt.Errorf("invalid direction: %s", direction)
		}
	}

	return nil
}

// processAlert processes a GTFS-RT alert and adds it to the store
func (m *Manager) processAlert(alert *gtfsrt.Alert) {
	if alert.HeaderText == nil || len(alert.HeaderText.Translation) == 0 {
		return
	}

	// Extract alert text
	headerText := alert.HeaderText.Translation[0].Text
	if headerText == nil {
		return
	}

	descriptionText := ""
	if alert.DescriptionText != nil && len(alert.DescriptionText.Translation) > 0 && alert.DescriptionText.Translation[0].Text != nil {
		descriptionText = *alert.DescriptionText.Translation[0].Text
	}

	// Extract affected routes and stations
	var routes []string
	var stationIDs []string

	for _, entity := range alert.InformedEntity {
		if entity.RouteId != nil {
			routeName := m.extractRouteFromID(*entity.RouteId)
			if routeName != "" {
				routes = append(routes, routeName)
			}
		}
		if entity.StopId != nil {
			stopID := *entity.StopId
			// Extract parent station ID
			if len(stopID) > 0 && (stopID[len(stopID)-1] == 'N' || stopID[len(stopID)-1] == 'S') {
				stopID = stopID[:len(stopID)-1]
			}
			stationIDs = append(stationIDs, stopID)
		}
	}

	// Create alert model
	alertModel := models.Alert{
		ID:            fmt.Sprintf("rt_%d", time.Now().Unix()), // Generate unique ID
		Header:        *headerText,
		Description:   descriptionText,
		Routes:        routes,
		Stations:      stationIDs,
		ActivePeriods: []models.TimePeriod{}, // TODO: Parse active periods from alert.ActivePeriod
	}

	// Add active periods
	for _, period := range alert.ActivePeriod {
		timePeriod := models.TimePeriod{}
		if period.Start != nil {
			startTime := time.Unix(int64(*period.Start), 0)
			timePeriod.Start = &startTime
		}
		if period.End != nil {
			endTime := time.Unix(int64(*period.End), 0)
			timePeriod.End = &endTime
		}
		alertModel.ActivePeriods = append(alertModel.ActivePeriods, timePeriod)
	}

	// Get current alerts and add this one
	currentAlerts := m.store.GetServiceAlerts()
	currentAlerts = append(currentAlerts, alertModel)
	m.store.UpdateAlerts(currentAlerts)
}

// extractRouteFromID extracts route name from GTFS route ID
// E.g., "A20241201" -> "A", "N20241201" -> "N", "123_20241201" -> "123_"
func (m *Manager) extractRouteFromID(routeID string) string {
	// MTA route IDs often have the format: RouteNameYYYYMMDD
	// We want to extract just the route name part

	// Look for a pattern like YYYYMMDD (8 consecutive digits) at the end
	if len(routeID) >= 8 {
		// Check if the last 8 characters are digits (date pattern)
		isDate := true
		for i := len(routeID) - 8; i < len(routeID); i++ {
			if routeID[i] < '0' || routeID[i] > '9' {
				isDate = false
				break
			}
		}
		if isDate {
			return routeID[:len(routeID)-8]
		}
	}

	// Fallback: look for the first sequence of 4+ digits
	for i, char := range routeID {
		if char >= '0' && char <= '9' && i > 0 {
			// Check if this starts a sequence of at least 4 digits
			digitCount := 0
			for j := i; j < len(routeID) && routeID[j] >= '0' && routeID[j] <= '9'; j++ {
				digitCount++
			}
			if digitCount >= 4 {
				return routeID[:i]
			}
		}
	}

	// If no date pattern found, return the whole string
	// (might be a simple route name like "A", "1", or "SIR")
	return routeID
}

// fetchFeed retrieves GTFS-RT protobuf data from MTA API
func (m *Manager) fetchFeed(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	// MTA requires API key in x-api-key header
	req.Header.Set("x-api-key", m.apiKey)

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// loadStaticGTFSData downloads and parses GTFS static data
func (m *Manager) loadStaticGTFSData() error {
	// Create data directory if it doesn't exist
	if err := os.MkdirAll(m.gtfsDataDir, 0755); err != nil {
		return fmt.Errorf("failed to create GTFS data directory: %w", err)
	}

	// Download and extract GTFS data (prefer supplemented for current service changes)
	gtfsPath := filepath.Join(m.gtfsDataDir, "gtfs_supplemented.zip")
	if err := m.downloadFile(GTFSSupplementedURL, gtfsPath); err != nil {
		slog.Warn("Failed to download supplemented GTFS, trying regular", "error", err)
		// Fallback to regular GTFS
		gtfsPath = filepath.Join(m.gtfsDataDir, "gtfs_subway.zip")
		if err := m.downloadFile(GTFSRegularURL, gtfsPath); err != nil {
			return fmt.Errorf("failed to download GTFS data: %w", err)
		}
	}

	// Extract ZIP file
	extractDir := filepath.Join(m.gtfsDataDir, "extracted")
	if err := m.extractZip(gtfsPath, extractDir); err != nil {
		return fmt.Errorf("failed to extract GTFS data: %w", err)
	}

	// Parse GTFS data and populate store
	if err := m.parseGTFSData(extractDir); err != nil {
		return fmt.Errorf("failed to parse GTFS data: %w", err)
	}

	return nil
}

// downloadFile downloads a file from URL to local path
func (m *Manager) downloadFile(url, filepath string) error {
	resp, err := m.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// extractZip extracts a ZIP file to the specified directory
func (m *Manager) extractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	// Create destination directory
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	// Extract files
	for _, f := range r.File {
		path := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.FileInfo().Mode())
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.FileInfo().Mode())
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

// parseGTFSData reads GTFS CSV files and populates the store
func (m *Manager) parseGTFSData(gtfsDir string) error {
	// Parse stops.txt for station information
	stations, err := m.parseStops(filepath.Join(gtfsDir, "stops.txt"))
	if err != nil {
		return fmt.Errorf("failed to parse stops: %w", err)
	}

	// Parse routes.txt and associate with stations
	if err := m.parseRoutes(filepath.Join(gtfsDir, "routes.txt"), stations); err != nil {
		return fmt.Errorf("failed to parse routes: %w", err)
	}

	// Update store with parsed data
	m.store.UpdateStations(stations)
	m.store.UpdateAlerts([]models.Alert{}) // No static alerts in GTFS

	slog.Info("Loaded stations from GTFS data", "count", len(stations))
	return nil
}

// parseStops reads stops.txt and creates station data
// GTFS uses location_type=1 for parent stations and empty for platform stops
func (m *Manager) parseStops(stopsFile string) (map[string]*models.Station, error) {
	file, err := os.Open(stopsFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("empty stops file")
	}

	// Parse header to find column indices
	header := records[0]
	columns := make(map[string]int)
	for i, col := range header {
		columns[col] = i
	}

	requiredCols := []string{"stop_id", "stop_name", "stop_lat", "stop_lon"}
	for _, col := range requiredCols {
		if _, ok := columns[col]; !ok {
			return nil, fmt.Errorf("missing required column: %s", col)
		}
	}

	stations := make(map[string]*models.Station)
	platformStops := make([][]string, 0) // Store platform stops for second pass

	// First pass: Process parent stations (location_type=1)
	for _, record := range records[1:] {
		if len(record) != len(header) {
			continue // Skip incomplete records
		}

		stopID := record[columns["stop_id"]]
		stopName := record[columns["stop_name"]]
		latStr := record[columns["stop_lat"]]
		lonStr := record[columns["stop_lon"]]

		// Skip if essential data is missing
		if stopID == "" || stopName == "" || latStr == "" || lonStr == "" {
			continue
		}

		// Check if this is a parent station
		locationType := ""
		if locTypeCol, ok := columns["location_type"]; ok && locTypeCol < len(record) {
			locationType = record[locTypeCol]
		}

		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil {
			slog.Warn("Invalid latitude for stop", "stop_id", stopID, "error", err)
			continue
		}

		lon, err := strconv.ParseFloat(lonStr, 64)
		if err != nil {
			slog.Warn("Invalid longitude for stop", "stop_id", stopID, "error", err)
			continue
		}

		if locationType == "1" {
			// This is a parent station
			stations[stopID] = &models.Station{
				ID:         stopID,
				Name:       stopName,
				Location:   models.Location{Lat: lat, Lon: lon},
				Routes:     []string{},                 // Will be populated by parseRoutes
				Trains:     models.TrainsByDirection{}, // No static train data
				Stops:      make(map[string]models.Location),
				LastUpdate: time.Now(),
			}
		} else {
			// This is a platform stop, save for second pass
			platformStops = append(platformStops, record)
		}
	}

	// Second pass: Associate platform stops with parent stations
	parentStationCol := -1
	if col, ok := columns["parent_station"]; ok {
		parentStationCol = col
	}

	for _, record := range platformStops {
		stopID := record[columns["stop_id"]]
		latStr := record[columns["stop_lat"]]
		lonStr := record[columns["stop_lon"]]

		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil {
			continue
		}

		lon, err := strconv.ParseFloat(lonStr, 64)
		if err != nil {
			continue
		}

		// Find parent station
		var parentID string
		if parentStationCol >= 0 && parentStationCol < len(record) && record[parentStationCol] != "" {
			parentID = record[parentStationCol]
		} else {
			// Fallback: extract from stop ID (remove direction suffix)
			parentID = stopID
			if len(stopID) > 0 && (stopID[len(stopID)-1] == 'N' || stopID[len(stopID)-1] == 'S') {
				parentID = stopID[:len(stopID)-1]
			}
		}

		// Add platform stop to parent station
		if station, exists := stations[parentID]; exists {
			station.Stops[stopID] = models.Location{Lat: lat, Lon: lon}
		}
	}

	return stations, nil
}

// parseRoutes reads routes.txt and associates routes with stations
// Joins routes.txt -> trips.txt -> stop_times.txt to map routes to stations
func (m *Manager) parseRoutes(routesFile string, stations map[string]*models.Station) error {
	gtfsDir := filepath.Dir(routesFile)

	// Step 1: Parse routes.txt to get route_id -> route_short_name mapping
	routes, err := m.parseRoutesFile(routesFile)
	if err != nil {
		return fmt.Errorf("failed to parse routes file: %w", err)
	}

	// Step 2: Parse trips.txt to get route_id -> trip_ids mapping
	routeTrips, err := m.parseTripsFile(filepath.Join(gtfsDir, "trips.txt"))
	if err != nil {
		return fmt.Errorf("failed to parse trips file: %w", err)
	}

	// Step 3: Parse stop_times.txt to get trip_id -> stop_ids mapping
	tripStops, err := m.parseStopTimesFile(filepath.Join(gtfsDir, "stop_times.txt"))
	if err != nil {
		return fmt.Errorf("failed to parse stop_times file: %w", err)
	}

	// Step 4: Join the data to build route -> stations mapping
	stationRoutes := make(map[string]map[string]bool) // station_id -> set of routes

	for routeID, routeName := range routes {
		tripIDs, ok := routeTrips[routeID]
		if !ok {
			continue
		}

		for tripID := range tripIDs {
			stopIDs, ok := tripStops[tripID]
			if !ok {
				continue
			}

			for stopID := range stopIDs {
				// Extract parent station ID (remove direction suffix)
				parentID := stopID
				if len(stopID) > 0 && (stopID[len(stopID)-1] == 'N' || stopID[len(stopID)-1] == 'S') {
					parentID = stopID[:len(stopID)-1]
				}

				if stationRoutes[parentID] == nil {
					stationRoutes[parentID] = make(map[string]bool)
				}
				stationRoutes[parentID][routeName] = true
			}
		}
	}

	// Step 5: Update stations with route information
	for stationID, station := range stations {
		if routeSet, ok := stationRoutes[stationID]; ok {
			routes := make([]string, 0, len(routeSet))
			for route := range routeSet {
				routes = append(routes, route)
			}
			station.Routes = routes
		}
	}

	slog.Info("Mapped routes to stations", "station_count", len(stationRoutes))
	return nil
}

// parseRoutesFile reads routes.txt and returns route_id -> route_short_name mapping
func (m *Manager) parseRoutesFile(routesFile string) (map[string]string, error) {
	file, err := os.Open(routesFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("empty routes file")
	}

	// Parse header
	header := records[0]
	columns := make(map[string]int)
	for i, col := range header {
		columns[col] = i
	}

	routeIDCol, ok := columns["route_id"]
	if !ok {
		return nil, fmt.Errorf("missing route_id column")
	}

	routeNameCol, ok := columns["route_short_name"]
	if !ok {
		return nil, fmt.Errorf("missing route_short_name column")
	}

	routes := make(map[string]string)
	for _, record := range records[1:] {
		if len(record) > routeIDCol && len(record) > routeNameCol {
			routeID := record[routeIDCol]
			routeName := record[routeNameCol]
			if routeID != "" && routeName != "" {
				routes[routeID] = routeName
			}
		}
	}

	return routes, nil
}

// parseTripsFile reads trips.txt and returns route_id -> set of trip_ids mapping
func (m *Manager) parseTripsFile(tripsFile string) (map[string]map[string]bool, error) {
	file, err := os.Open(tripsFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("empty trips file")
	}

	// Parse header
	header := records[0]
	columns := make(map[string]int)
	for i, col := range header {
		columns[col] = i
	}

	routeIDCol, ok := columns["route_id"]
	if !ok {
		return nil, fmt.Errorf("missing route_id column")
	}

	tripIDCol, ok := columns["trip_id"]
	if !ok {
		return nil, fmt.Errorf("missing trip_id column")
	}

	routeTrips := make(map[string]map[string]bool)
	for _, record := range records[1:] {
		if len(record) > routeIDCol && len(record) > tripIDCol {
			routeID := record[routeIDCol]
			tripID := record[tripIDCol]
			if routeID != "" && tripID != "" {
				if routeTrips[routeID] == nil {
					routeTrips[routeID] = make(map[string]bool)
				}
				routeTrips[routeID][tripID] = true
			}
		}
	}

	return routeTrips, nil
}

// parseStopTimesFile reads stop_times.txt and returns trip_id -> set of stop_ids mapping
func (m *Manager) parseStopTimesFile(stopTimesFile string) (map[string]map[string]bool, error) {
	file, err := os.Open(stopTimesFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Parse header first
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	columns := make(map[string]int)
	for i, col := range header {
		columns[col] = i
	}

	tripIDCol, ok := columns["trip_id"]
	if !ok {
		return nil, fmt.Errorf("missing trip_id column")
	}

	stopIDCol, ok := columns["stop_id"]
	if !ok {
		return nil, fmt.Errorf("missing stop_id column")
	}

	tripStops := make(map[string]map[string]bool)

	// Process records one by one to handle large files efficiently
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading stop_times: %w", err)
		}

		if len(record) > tripIDCol && len(record) > stopIDCol {
			tripID := record[tripIDCol]
			stopID := record[stopIDCol]
			if tripID != "" && stopID != "" {
				if tripStops[tripID] == nil {
					tripStops[tripID] = make(map[string]bool)
				}
				tripStops[tripID][stopID] = true
			}
		}
	}

	return tripStops, nil
}

// sortAndLimitTrains sorts trains by arrival time and limits to next 10 arrivals
func (m *Manager) sortAndLimitTrains(trains []models.Train) []models.Train {
	if len(trains) == 0 {
		return trains
	}

	// Remove duplicates and sort by time
	trainMap := make(map[string]models.Train)
	for _, train := range trains {
		key := fmt.Sprintf("%s_%d", train.Route, train.Time.Unix())
		trainMap[key] = train
	}

	// Convert back to slice
	uniqueTrains := make([]models.Train, 0, len(trainMap))
	for _, train := range trainMap {
		uniqueTrains = append(uniqueTrains, train)
	}

	// Sort by arrival time
	for i := 0; i < len(uniqueTrains)-1; i++ {
		for j := i + 1; j < len(uniqueTrains); j++ {
			if uniqueTrains[i].Time.After(uniqueTrains[j].Time) {
				uniqueTrains[i], uniqueTrains[j] = uniqueTrains[j], uniqueTrains[i]
			}
		}
	}

	// Limit to next 10 arrivals
	if len(uniqueTrains) > 10 {
		uniqueTrains = uniqueTrains[:10]
	}

	return uniqueTrains
}
