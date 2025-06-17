package mta

import (
	"time"

	"github.com/jusunglee/mta-go/internal/models"
)

// Client defines the interface for accessing MTA data
type Client interface {
	// Station queries
	GetStationsByLocation(lat, lon float64, limit int) ([]models.Station, error)
	GetStationsByRoute(route string) ([]models.Station, error)
	GetStationsByIDs(ids []string) ([]models.Station, error)

	// Route information
	GetRoutes() ([]string, error)

	// Service alerts
	GetServiceAlerts() ([]models.Alert, error)

	// Metadata
	GetLastUpdate() time.Time
}

// Config holds configuration for the MTA client
type Config struct {
	APIKey         string
	UpdateInterval time.Duration
	StationsFile   string
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		UpdateInterval: 60 * time.Second,
		StationsFile:   "data/stations.json",
	}
}
