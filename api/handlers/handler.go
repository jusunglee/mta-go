package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"time"

	"github.com/gorilla/mux"
	"github.com/jusunglee/mta-go/internal/models"
	"github.com/jusunglee/mta-go/pkg/mta"
)

// Handler handles HTTP requests
// Wraps MTA client with REST API endpoints
type Handler struct {
	client mta.Client
}

func NewHandler(client mta.Client) *Handler {
	return &Handler{client: client}
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/", h.handleIndex).Methods("GET")
	r.HandleFunc("/by-location", h.handleByLocation).Methods("GET")
	r.HandleFunc("/by-route/{route}", h.handleByRoute).Methods("GET")
	r.HandleFunc("/by-id/{ids}", h.handleByID).Methods("GET")
	r.HandleFunc("/routes", h.handleRoutes).Methods("GET")
	r.HandleFunc("/alerts", h.handleAlerts).Methods("GET")
}

// Base response metadata for all API responses
type ResponseMetadata struct {
	Updated           string `json:"updated,omitempty"`            // Real-time data update
	StaticDataUpdated string `json:"static_data_updated,omitempty"` // Static GTFS data update
}

// Specific response types for each endpoint
type StationsResponse struct {
	Data []models.StationResponse `json:"data"`
	ResponseMetadata
}

type RoutesResponse struct {
	Data []string `json:"data"`
	ResponseMetadata
}

type AlertsResponse struct {
	Data []models.Alert `json:"data"`
	ResponseMetadata
}

type InfoResponse struct {
	Data map[string]string `json:"data"`
	ResponseMetadata
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// getResponseMetadata creates metadata with update timestamps
func (h *Handler) getResponseMetadata() ResponseMetadata {
	meta := ResponseMetadata{}
	
	// Add real-time data update time
	if lastUpdate := h.client.GetLastUpdate(); !lastUpdate.IsZero() {
		meta.Updated = lastUpdate.Format(time.RFC3339)
	}
	
	// Add static data update time if available
	if staticUpdate := h.client.GetLastStaticUpdate(); !staticUpdate.IsZero() {
		meta.StaticDataUpdated = staticUpdate.Format(time.RFC3339)
	}
	
	return meta
}

func (h *Handler) handleIndex(w http.ResponseWriter, r *http.Request) {
	response := InfoResponse{
		Data: map[string]string{
			"title":  "mta-go",
			"readme": "Visit https://github.com/jusunglee/mta-go for more info",
		},
		ResponseMetadata: h.getResponseMetadata(),
	}
	h.writeJSON(w, response)
}

func (h *Handler) handleByLocation(w http.ResponseWriter, r *http.Request) {
	// Extract and validate coordinate parameters
	latStr := r.URL.Query().Get("lat")
	lonStr := r.URL.Query().Get("lon")

	if latStr == "" || lonStr == "" {
		h.writeError(w, "Missing lat/lon parameter", http.StatusBadRequest)
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		h.writeError(w, "Invalid lat parameter", http.StatusBadRequest)
		return
	}

	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		h.writeError(w, "Invalid lon parameter", http.StatusBadRequest)
		return
	}

	// Hardcoded limit of 5 stations for reasonable response size
	stations, err := h.client.GetStationsByLocation(lat, lon, 5)
	if err != nil {
		h.writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.writeStationsResponse(w, stations)
}

func (h *Handler) handleByRoute(w http.ResponseWriter, r *http.Request) {
	route := mux.Vars(r)["route"]

	stations, err := h.client.GetStationsByRoute(route)
	if err != nil {
		h.writeError(w, err.Error(), http.StatusNotFound)
		return
	}

	h.writeStationsResponse(w, stations)
}

func (h *Handler) handleByID(w http.ResponseWriter, r *http.Request) {
	// Parse comma-separated station IDs from URL path
	idsStr := mux.Vars(r)["ids"]
	ids := strings.Split(idsStr, ",")

	stations, err := h.client.GetStationsByIDs(ids)
	if err != nil {
		h.writeError(w, err.Error(), http.StatusNotFound)
		return
	}

	h.writeStationsResponse(w, stations)
}

func (h *Handler) handleRoutes(w http.ResponseWriter, r *http.Request) {
	routes, err := h.client.GetRoutes()
	if err != nil {
		h.writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := RoutesResponse{
		Data:             routes,
		ResponseMetadata: h.getResponseMetadata(),
	}
	
	h.writeJSON(w, response)
}

func (h *Handler) handleAlerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := h.client.GetServiceAlerts()
	if err != nil {
		h.writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := AlertsResponse{
		Data:             alerts,
		ResponseMetadata: h.getResponseMetadata(),
	}
	
	h.writeJSON(w, response)
}

func (h *Handler) writeStationsResponse(w http.ResponseWriter, stations []models.Station) {
	// Convert internal Station structs to API response format
	data := make([]models.StationResponse, len(stations))
	var lastUpdate time.Time

	// Track the most recent update time across all stations
	for i, station := range stations {
		data[i] = station.ConvertToResponse()
		if station.LastUpdate.After(lastUpdate) {
			lastUpdate = station.LastUpdate
		}
	}

	// Create response with proper typing
	response := StationsResponse{
		Data:             data,
		ResponseMetadata: h.getResponseMetadata(),
	}
	
	// Override with station-specific update time if more recent
	if !lastUpdate.IsZero() {
		response.Updated = lastUpdate.Format(time.RFC3339)
	}

	h.writeJSON(w, response)
}

func (h *Handler) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.writeError(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (h *Handler) writeError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}
