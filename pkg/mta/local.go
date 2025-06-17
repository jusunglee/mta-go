package mta

import (
	"time"

	"github.com/jusunglee/mta-go/internal/feed"
	"github.com/jusunglee/mta-go/internal/models"
	"github.com/jusunglee/mta-go/internal/store"
)

// LocalClient implements the Client interface for local usage
// Manages in-memory data store and background feed updates
type LocalClient struct {
	store       *store.Store
	feedManager *feed.Manager
}

// NewLocal creates a new local MTA client
// Starts background feed manager for automatic data updates
func NewLocal(config Config) (*LocalClient, error) {
	s := store.NewStore()

	// TODO: Load static station data from stations.json file
	// Currently relies on feed manager to populate station data dynamically

	fm := feed.NewManager(config.APIKey, s, config.UpdateInterval)
	fm.Start()

	return &LocalClient{
		store:       s,
		feedManager: fm,
	}, nil
}

// Close gracefully shuts down the local client
// Must be called to stop background goroutines and prevent leaks
func (c *LocalClient) Close() {
	c.feedManager.Stop()
}

func (c *LocalClient) GetStationsByLocation(lat, lon float64, limit int) ([]models.Station, error) {
	return c.store.GetStationsByLocation(lat, lon, limit), nil
}

func (c *LocalClient) GetStationsByRoute(route string) ([]models.Station, error) {
	return c.store.GetStationsByRoute(route)
}

func (c *LocalClient) GetStationsByIDs(ids []string) ([]models.Station, error) {
	return c.store.GetStationsByIDs(ids)
}

func (c *LocalClient) GetRoutes() ([]string, error) {
	return c.store.GetRoutes(), nil
}

func (c *LocalClient) GetServiceAlerts() ([]models.Alert, error) {
	return c.store.GetServiceAlerts(), nil
}

func (c *LocalClient) GetLastUpdate() time.Time {
	return c.store.GetLastUpdate()
}
