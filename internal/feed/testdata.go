package feed

import (
	"time"

	"github.com/jusunglee/mta-go/internal/models"
)

// CreateMockStations creates mock station data for testing
// Uses real NYC subway station coordinates and route assignments
func CreateMockStations() map[string]*models.Station {
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

// CreateMockAlerts creates mock alert data for testing
// Simulates typical MTA service advisories
func CreateMockAlerts() []models.Alert {
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