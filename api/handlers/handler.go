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
type Handler struct {
	client mta.Client
}

// NewHandler creates a new HTTP handler
func NewHandler(client mta.Client) *Handler {
	return &Handler{client: client}
}

// RegisterRoutes registers all routes
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/", h.handleIndex).Methods("GET")
	r.HandleFunc("/by-location", h.handleByLocation).Methods("GET")
	r.HandleFunc("/by-route/{route}", h.handleByRoute).Methods("GET")
	r.HandleFunc("/by-id/{ids}", h.handleByID).Methods("GET")
	r.HandleFunc("/routes", h.handleRoutes).Methods("GET")
	r.HandleFunc("/alerts", h.handleAlerts).Methods("GET")
}

// Response wraps API responses
type Response struct {
	Data    interface{} `json:"data"`
	Updated string      `json:"updated,omitempty"`
}

// ErrorResponse represents an error response
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
	// Convert stations to response format
	data := make([]models.StationResponse, len(stations))
	var lastUpdate time.Time

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
