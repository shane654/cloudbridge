package signal

import (
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// DeviceInfo holds metadata about a registered device.
type DeviceInfo struct {
	ID         string
	Name       string
	Platform   string
	Version    string
	PublicKey  string
	Registered time.Time
	LastSeen   time.Time
}

// Client represents a connected WebSocket client (agent or app).
type Client struct {
	ID       string
	IsAgent  bool            // true for agents, false for apps
	DeviceID string          // populated after registration (agents only)
	UserID   string          // populated after auth (apps only)
	Conn     *websocket.Conn // WebSocket connection
	Send     chan []byte     // buffered channel for outgoing messages
	Close    chan struct{}
}

// Hub maintains the set of active clients and routes messages between them.
type Hub struct {
	mu       sync.RWMutex
	clients  map[string]*Client    // clientID -> Client
	devices  map[string]*DeviceInfo // deviceID -> DeviceInfo (registered agents)
	byDevice map[string]*Client     // deviceID -> Client (for agent lookup)
}

// NewHub creates a new Hub instance.
func NewHub() *Hub {
	return &Hub{
		clients:  make(map[string]*Client),
		devices:  make(map[string]*DeviceInfo),
		byDevice: make(map[string]*Client),
	}
}

// RegisterClient adds a client to the hub.
func (h *Hub) RegisterClient(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c.ID] = c
	slog.Info("client registered", "client_id", c.ID, "is_agent", c.IsAgent)
}

// UnregisterClient removes a client and cleans up associated device.
func (h *Hub) UnregisterClient(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.clients, c.ID)

	// If this client was an agent, remove its device registration.
	if c.IsAgent && c.DeviceID != "" {
		delete(h.byDevice, c.DeviceID)
		delete(h.devices, c.DeviceID)
		slog.Info("device unregistered", "device_id", c.DeviceID)
	}

	close(c.Send)
	slog.Info("client unregistered", "client_id", c.ID)
}

// RegisterDevice records an agent's device info.
// If the device already exists (same device ID), it updates the info and
// replaces the old client connection instead of creating a duplicate.
func (h *Hub) RegisterDevice(client *Client, info *DeviceInfo) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// If device already registered, update it instead of duplicating
	if existing, ok := h.devices[info.ID]; ok {
		// Clean up old client association
		if oldClient, oldOk := h.byDevice[info.ID]; oldOk {
			oldClient.DeviceID = ""
		}
		// Preserve original registration time
		info.Registered = existing.Registered
		slog.Info("device re-registered", "device_id", info.ID, "name", info.Name, "platform", info.Platform)
	} else {
		info.Registered = time.Now()
		slog.Info("device registered", "device_id", info.ID, "name", info.Name, "platform", info.Platform)
	}

	info.LastSeen = time.Now()
	h.devices[info.ID] = info
	h.byDevice[info.ID] = client
	client.DeviceID = info.ID
}

// UpdateHeartbeat refreshes the last-seen timestamp for a device.
func (h *Hub) UpdateHeartbeat(deviceID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if dev, ok := h.devices[deviceID]; ok {
		dev.LastSeen = time.Now()
	}
}

// FindDeviceByClient returns the device info for a given client's device.
func (h *Hub) FindDeviceByClient(c *Client) (*DeviceInfo, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if c.DeviceID == "" {
		return nil, false
	}
	dev, ok := h.devices[c.DeviceID]
	return dev, ok
}

// FindClientByDevice returns the client for a given device ID.
func (h *Hub) FindClientByDevice(deviceID string) (*Client, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	c, ok := h.byDevice[deviceID]
	return c, ok
}

// ListDevices returns all registered devices.
func (h *Hub) ListDevices() []DeviceInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]DeviceInfo, 0, len(h.devices))
	for _, dev := range h.devices {
		result = append(result, *dev)
	}
	return result
}

// RouteToDevice sends a message to the client associated with a device ID.
// Returns false if the device is not connected.
func (h *Hub) RouteToDevice(deviceID string, msg []byte) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	c, ok := h.byDevice[deviceID]
	if !ok {
		return false
	}

	select {
	case c.Send <- msg:
		return true
	default:
		slog.Warn("client send buffer full, dropping message", "client_id", c.ID, "device_id", deviceID)
		return false
	}
}

// RouteToClient sends a message to a specific client.
func (h *Hub) RouteToClient(clientID string, msg []byte) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	c, ok := h.clients[clientID]
	if !ok {
		return false
	}

	select {
	case c.Send <- msg:
		return true
	default:
		slog.Warn("client send buffer full, dropping message", "client_id", clientID)
		return false
	}
}