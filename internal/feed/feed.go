package feed

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/jusunglee/mta-go/internal/models"
	"github.com/jusunglee/mta-go/internal/store"
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
	apiKey         string
	store          *store.Store
	updateInterval time.Duration
	httpClient     *http.Client
	stopCh         chan struct{}
	wg             sync.WaitGroup
}

func NewManager(apiKey string, store *store.Store, updateInterval time.Duration) *Manager {
	return &Manager{
		apiKey:         apiKey,
		store:          store,
		updateInterval: updateInterval,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		stopCh: make(chan struct{}),
	}
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
		log.Printf("Initial update failed: %v", err)
	}

	ticker := time.NewTicker(m.updateInterval)
	defer ticker.Stop()

	// Main update loop - select pattern for clean shutdown
	for {
		select {
		case <-ticker.C:
			if err := m.update(); err != nil {
				log.Printf("Update failed: %v", err)
			}
		case <-m.stopCh:
			return
		}
	}
}

func (m *Manager) update() error {
	// TODO: Implement actual GTFS-RT parsing from FeedURLs
	// Currently using mock data for development/testing
	stations := m.createMockStations()
	alerts := m.createMockAlerts()

	// Atomically update store with new data
	m.store.UpdateStations(stations)
	m.store.UpdateAlerts(alerts)

	return nil
}

// fetchFeed retrieves GTFS-RT data from MTA API
// Currently unused - will be needed when implementing real feed parsing
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

// createMockStations creates mock station data for testing
// Uses real NYC subway station coordinates and route assignments
func (m *Manager) createMockStations() map[string]*models.Station {
	now := time.Now()
	stations := map[string]*models.Station{
		"127": {
			ID:       "127",
			Name:     "Times Sq-42 St",
			Location: models.Location{Lat: 40.755477, Lon: -73.987691},
			Routes:   []string{"N", "Q", "R", "W", "S", "1", "2", "3", "7"},
			Trains: models.TrainsByDirection{
				North: []models.Train{
					{Route: "N", Time: now.Add(2 * time.Minute)},
					{Route: "Q", Time: now.Add(5 * time.Minute)},
					{Route: "1", Time: now.Add(3 * time.Minute)},
				},
				South: []models.Train{
					{Route: "R", Time: now.Add(1 * time.Minute)},
					{Route: "W", Time: now.Add(4 * time.Minute)},
					{Route: "2", Time: now.Add(6 * time.Minute)},
				},
			},
			Stops: map[string]models.Location{
				"127N": {Lat: 40.755983, Lon: -73.986229},
				"127S": {Lat: 40.75529, Lon: -73.987495},
			},
			LastUpdate: now,
		},
		"631": {
			ID:       "631",
			Name:     "Grand Central-42 St",
			Location: models.Location{Lat: 40.751776, Lon: -73.976848},
			Routes:   []string{"4", "5", "6", "7", "S"},
			Trains: models.TrainsByDirection{
				North: []models.Train{
					{Route: "4", Time: now.Add(3 * time.Minute)},
					{Route: "5", Time: now.Add(5 * time.Minute)},
					{Route: "6", Time: now.Add(2 * time.Minute)},
				},
				South: []models.Train{
					{Route: "4", Time: now.Add(4 * time.Minute)},
					{Route: "6", Time: now.Add(1 * time.Minute)},
				},
			},
			Stops: map[string]models.Location{
				"631N": {Lat: 40.752769, Lon: -73.979189},
				"631S": {Lat: 40.751431, Lon: -73.976041},
			},
			LastUpdate: now,
		},
		"635": {
			ID:       "635",
			Name:     "14 St-Union Sq",
			Location: models.Location{Lat: 40.734673, Lon: -73.989951},
			Routes:   []string{"N", "Q", "R", "W", "4", "5", "6", "L"},
			Trains: models.TrainsByDirection{
				North: []models.Train{
					{Route: "N", Time: now.Add(2 * time.Minute)},
					{Route: "4", Time: now.Add(4 * time.Minute)},
					{Route: "L", Time: now.Add(3 * time.Minute)},
				},
				South: []models.Train{
					{Route: "Q", Time: now.Add(5 * time.Minute)},
					{Route: "6", Time: now.Add(2 * time.Minute)},
				},
			},
			Stops: map[string]models.Location{
				"635N": {Lat: 40.735736, Lon: -73.990568},
				"635S": {Lat: 40.734789, Lon: -73.99073},
			},
			LastUpdate: now,
		},
	}
	return stations
}

// createMockAlerts creates mock alert data for testing
// Simulates typical MTA service advisories
func (m *Manager) createMockAlerts() []models.Alert {
	now := time.Now()
	future := now.Add(2 * time.Hour)

	return []models.Alert{
		{
			ID:          "alert1",
			Header:      "Weekend Service Change",
			Description: "N/Q/R/W trains are running on a modified schedule this weekend",
			Routes:      []string{"N", "Q", "R", "W"},
			Stations:    []string{"127", "635"},
			ActivePeriods: []models.TimePeriod{
				{Start: &now, End: &future},
			},
		},
	}
}
