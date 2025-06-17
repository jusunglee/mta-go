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

// Response wraps API responses
// Includes optional timestamp for cache validation
type Response struct {
	Data    interface{} `json:"data"`
	Updated string      `json:"updated,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) handleIndex(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"title":  "mta-go",
		"readme": "Visit https://github.com/jusunglee/mta-go for more info",
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

	response := Response{
		Data:    routes,
		Updated: h.client.GetLastUpdate().Format(time.RFC3339),
	}
	h.writeJSON(w, response)
}

func (h *Handler) handleAlerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := h.client.GetServiceAlerts()
	if err != nil {
		h.writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := Response{
		Data:    alerts,
		Updated: h.client.GetLastUpdate().Format(time.RFC3339),
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

	response := Response{
		Data: data,
	}
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
