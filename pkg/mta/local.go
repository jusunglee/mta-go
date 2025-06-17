package mta

import (
	"time"

	"github.com/jusunglee/mta-go/internal/feed"
	"github.com/jusunglee/mta-go/internal/models"
	"github.com/jusunglee/mta-go/internal/store"
)

// LocalClient implements the Client interface for local usage
type LocalClient struct {
	store       *store.Store
	feedManager *feed.Manager
}

// NewLocal creates a new local MTA client
func NewLocal(config Config) (*LocalClient, error) {
	s := store.NewStore()

	// TODO: Load stations from stations.json file
	// For now, we'll let the feed populate stations dynamically

	fm := feed.NewManager(config.APIKey, s, config.UpdateInterval)
	fm.Start()

	return &LocalClient{
		store:       s,
		feedManager: fm,
	}, nil
}

// Close stops the local client
func (c *LocalClient) Close() {
	c.feedManager.Stop()
}

// GetStationsByLocation returns stations near a location
func (c *LocalClient) GetStationsByLocation(lat, lon float64, limit int) ([]models.Station, error) {
	return c.store.GetStationsByLocation(lat, lon, limit), nil
}

// GetStationsByRoute returns all stations on a route
func (c *LocalClient) GetStationsByRoute(route string) ([]models.Station, error) {
	return c.store.GetStationsByRoute(route)
}

// GetStationsByIDs returns stations by their IDs
func (c *LocalClient) GetStationsByIDs(ids []string) ([]models.Station, error) {
	return c.store.GetStationsByIDs(ids)
}

// GetRoutes returns all available routes
func (c *LocalClient) GetRoutes() ([]string, error) {
	return c.store.GetRoutes(), nil
}

// GetServiceAlerts returns all active service alerts
func (c *LocalClient) GetServiceAlerts() ([]models.Alert, error) {
	return c.store.GetServiceAlerts(), nil
}

// GetLastUpdate returns the last update time
func (c *LocalClient) GetLastUpdate() time.Time {
	return c.store.GetLastUpdate()
}
