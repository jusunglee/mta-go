package mta

import (
	"time"

	"github.com/jusunglee/mta-go/internal/models"
)

// Client defines the interface for accessing MTA data
// Abstracts different data sources (local vs remote) behind common interface
type Client interface {
	GetStationsByLocation(lat, lon float64, limit int) ([]models.Station, error)
	GetStationsByRoute(route string) ([]models.Station, error)
	GetStationsByIDs(ids []string) ([]models.Station, error)

	GetRoutes() ([]string, error)

	GetServiceAlerts() ([]models.Alert, error)

	GetLastUpdate() time.Time
	GetLastStaticUpdate() time.Time
}

// Config holds configuration for the MTA client
// APIKey required for accessing MTA's GTFS-RT feeds
type Config struct {
	APIKey         string
	UpdateInterval time.Duration
	StationsFile   string
}

// DefaultConfig returns default configuration
// 60-second update interval balances freshness with API rate limits
func DefaultConfig() Config {
	return Config{
		UpdateInterval: 60 * time.Second,
		StationsFile:   "data/stations.json",
	}
}
