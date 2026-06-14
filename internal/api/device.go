// Package api implements the REST API for CloudBridge server.
// It provides endpoints for device management and monitoring.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/cloudbridge/cloudbridge/internal/signal"
)

// DeviceAPI handles device-related REST API endpoints.
type DeviceAPI struct {
	hub *signal.Hub
	sm  *signal.SessionManager
}

// NewDeviceAPI creates a new DeviceAPI.
func NewDeviceAPI(hub *signal.Hub, sm *signal.SessionManager) *DeviceAPI {
	return &DeviceAPI{
		hub: hub,
		sm:  sm,
	}
}

// RegisterRoutes registers the API routes on the given mux.
func (api *DeviceAPI) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/devices", api.ListDevices)
	mux.HandleFunc("GET /api/v1/devices/{deviceId}", api.GetDevice)
	mux.HandleFunc("GET /api/v1/sessions", api.ListSessions)
	mux.HandleFunc("GET /api/v1/stats", api.GetStats)
}

// DeviceResponse is the JSON response for a device.
type DeviceResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Platform   string `json:"platform"`
	Version    string `json:"version"`
	Online     bool   `json:"online"`
	Registered string `json:"registered"`
	LastSeen   string `json:"last_seen"`
}

// ListDevices returns all registered devices.
func (api *DeviceAPI) ListDevices(w http.ResponseWriter, r *http.Request) {
	devices := api.hub.ListDevices()

	resp := make([]DeviceResponse, 0, len(devices))
	for _, d := range devices {
		resp = append(resp, DeviceResponse{
			ID:         d.ID,
			Name:       d.Name,
			Platform:   d.Platform,
			Version:    d.Version,
			Online:     true, // If it's in the hub, it's online
			Registered: d.Registered.Format("2006-01-02T15:04:05Z"),
			LastSeen:   d.LastSeen.Format("2006-01-02T15:04:05Z"),
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

// GetDevice returns a specific device by ID.
func (api *DeviceAPI) GetDevice(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("deviceId")
	if deviceID == "" {
		writeError(w, http.StatusBadRequest, "missing device ID")
		return
	}

	// Check if device is online
	client, online := api.hub.FindClientByDevice(deviceID)
	_ = client // Will be used for detailed info later

	if !online {
		writeError(w, http.StatusNotFound, "device not found: "+deviceID)
		return
	}

	dev, found := api.hub.FindDeviceByClient(client)
	if !found {
		writeError(w, http.StatusNotFound, "device not found: "+deviceID)
		return
	}

	resp := DeviceResponse{
		ID:         dev.ID,
		Name:       dev.Name,
		Platform:   dev.Platform,
		Version:    dev.Version,
		Online:     true,
		Registered: dev.Registered.Format("2006-01-02T15:04:05Z"),
		LastSeen:   dev.LastSeen.Format("2006-01-02T15:04:05Z"),
	}

	writeJSON(w, http.StatusOK, resp)
}

// SessionResponse is the JSON response for a session.
type SessionResponse struct {
	ID        string `json:"id"`
	DeviceID  string `json:"device_id"`
	Status    string `json:"status"`
	Protocols string `json:"protocols"`
	CreatedAt string `json:"created_at"`
}

// ListSessions returns all active sessions.
func (api *DeviceAPI) ListSessions(w http.ResponseWriter, r *http.Request) {
	sessions := api.sm.ListSessions()

	resp := make([]SessionResponse, 0, len(sessions))
	for _, s := range sessions {
		resp = append(resp, SessionResponse{
			ID:        s.ID,
			DeviceID:  s.DeviceID,
			Status:    string(s.Status),
			Protocols: "", // TODO: serialize protocols
			CreatedAt: s.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

// StatsResponse is the JSON response for server stats.
type StatsResponse struct {
	DevicesOnline int `json:"devices_online"`
	SessionsActive int `json:"sessions_active"`
}

// GetStats returns server statistics.
func (api *DeviceAPI) GetStats(w http.ResponseWriter, r *http.Request) {
	devices := api.hub.ListDevices()
	sessions := api.sm.ListSessions()

	resp := StatsResponse{
		DevicesOnline:  len(devices),
		SessionsActive: len(sessions),
	}

	writeJSON(w, http.StatusOK, resp)
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("json encode error", "err", err)
	}
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}