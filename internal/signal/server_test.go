package signal

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/cloudbridge/cloudbridge/internal/protocol"
)

func TestSignalServerIntegration(t *testing.T) {
	// Create a test signal server
	hub := NewHub()
	sm := NewSessionManager()
	handler := NewHandler(hub, sm)

	server := &Server{
		config: ServerConfig{
			Addr:              ":0",
			Path:              "/signal",
			HeartbeatInterval: 30 * time.Second,
		},
		hub:            hub,
		sessionManager: sm,
		handler:        handler,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}

	// Create test HTTP server
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/signal" {
			server.handleWebSocket(w, r)
		}
	}))
	defer httpServer.Close()

	// Connect WebSocket client
	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/signal?type=agent"

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect WebSocket: %v", err)
	}
	defer ws.Close()

	// Test device registration
	regPayload := protocol.RegisterPayload{
		DeviceID:   "test-device-001",
		DeviceName: "Test Device",
		Platform:   protocol.PlatformLinux,
		Version:    "0.1.0",
		PublicKey:  "dGVzdA==",
	}

	msg, err := protocol.Encode(protocol.MsgTypeRegister, regPayload)
	if err != nil {
		t.Fatalf("Failed to encode register message: %v", err)
	}

	if err := ws.WriteMessage(websocket.TextMessage, msg); err != nil {
		t.Fatalf("Failed to send register message: %v", err)
	}

	// Read response
	_, response, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	msgType, err := protocol.Decode(response, nil)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if msgType != protocol.MsgTypeRegisterAck {
		t.Errorf("Expected register_ack, got %s", msgType)
	}

	var ack protocol.RegisterAckPayload
	if _, err := protocol.Decode(response, &ack); err != nil {
		t.Fatalf("Failed to decode register ack: %v", err)
	}

	if ack.Token == "" {
		t.Error("Expected non-empty token in register ack")
	}

	// Test heartbeat
	hbPayload := protocol.HeartbeatPayload{
		Timestamp: protocol.CurrentTimestampMillis(),
	}

	hbMsg, err := protocol.Encode(protocol.MsgTypeHeartbeat, hbPayload)
	if err != nil {
		t.Fatalf("Failed to encode heartbeat message: %v", err)
	}

	if err := ws.WriteMessage(websocket.TextMessage, hbMsg); err != nil {
		t.Fatalf("Failed to send heartbeat message: %v", err)
	}

	_, hbResponse, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read heartbeat response: %v", err)
	}

	hbType, err := protocol.Decode(hbResponse, nil)
	if err != nil {
		t.Fatalf("Failed to decode heartbeat response: %v", err)
	}

	if hbType != protocol.MsgTypeHeartbeatAck {
		t.Errorf("Expected heartbeat_ack, got %s", hbType)
	}

	// Verify device is registered in the hub
	devices := hub.ListDevices()
	if len(devices) != 1 {
		t.Errorf("Expected 1 device, got %d", len(devices))
	}

	if devices[0].ID != "test-device-001" {
		t.Errorf("Expected device ID test-device-001, got %s", devices[0].ID)
	}

	if devices[0].Name != "Test Device" {
		t.Errorf("Expected device name 'Test Device', got %s", devices[0].Name)
	}
}

func TestHubDeviceManagement(t *testing.T) {
	hub := NewHub()

	// Create mock client
	client := &Client{
		ID:      "test-client-1",
		IsAgent: true,
		Send:    make(chan []byte, 10),
		Close:   make(chan struct{}),
	}

	// Register client
	hub.RegisterClient(client)

	// Register device
	hub.RegisterDevice(client, &DeviceInfo{
		ID:       "dev-001",
		Name:     "Test-PC",
		Platform: "linux",
		Version:  "0.1.0",
	})

	// Test FindClientByDevice
	found, ok := hub.FindClientByDevice("dev-001")
	if !ok {
		t.Error("Expected to find device dev-001")
	}
	if found.ID != client.ID {
		t.Errorf("Expected client ID %s, got %s", client.ID, found.ID)
	}

	// Test FindDeviceByClient
	dev, ok := hub.FindDeviceByClient(client)
	if !ok {
		t.Error("Expected to find device for client")
	}
	if dev.ID != "dev-001" {
		t.Errorf("Expected device ID dev-001, got %s", dev.ID)
	}

	// Test ListDevices
	devices := hub.ListDevices()
	if len(devices) != 1 {
		t.Errorf("Expected 1 device, got %d", len(devices))
	}

	// Test RouteToDevice
	msg, _ := protocol.Encode(protocol.MsgTypeHeartbeatAck, protocol.HeartbeatAckPayload{
		ServerTime: protocol.CurrentTimestampMillis(),
	})

	if !hub.RouteToDevice("dev-001", msg) {
		t.Error("Expected to route message to device dev-001")
	}

	// Test UnregisterClient
	hub.UnregisterClient(client)

	// After unregister, device should be removed
	_, ok = hub.FindClientByDevice("dev-001")
	if ok {
		t.Error("Expected device to be removed after client unregister")
	}
}

func TestSessionManager(t *testing.T) {
	sm := NewSessionManager()
	hub := NewHub()

	// Create mock clients
	appClient := &Client{
		ID:      "app-1",
		IsAgent: false,
		Send:    make(chan []byte, 10),
		Close:   make(chan struct{}),
	}
	agentClient := &Client{
		ID:       "agent-1",
		IsAgent:  true,
		DeviceID: "dev-001",
		Send:     make(chan []byte, 10),
		Close:    make(chan struct{}),
	}

	hub.RegisterClient(appClient)
	hub.RegisterClient(agentClient)

	// Create session
	session := sm.CreateSession("ses-001", appClient, "dev-001", []string{"ssh", "shell"})
	if session.ID != "ses-001" {
		t.Errorf("Expected session ID ses-001, got %s", session.ID)
	}
	if session.Status != SessionPending {
		t.Errorf("Expected status pending, got %s", session.Status)
	}

	// Accept session
	accepted, ok := sm.AcceptSession("ses-001", agentClient)
	if !ok {
		t.Error("Expected to accept session ses-001")
	}
	if accepted.Status != SessionAccepted {
		t.Errorf("Expected status accepted, got %s", accepted.Status)
	}

	// Test GetPeer
	peer, ok := sm.GetPeer("ses-001", appClient)
	if !ok || peer.ID != agentClient.ID {
		t.Errorf("Expected peer %s, got %v", agentClient.ID, peer)
	}

	peer, ok = sm.GetPeer("ses-001", agentClient)
	if !ok || peer.ID != appClient.ID {
		t.Errorf("Expected peer %s, got %v", appClient.ID, peer)
	}

	// Test RouteToPeer
	msg, _ := protocol.Encode(protocol.MsgTypeConnectResponse, protocol.ConnectResponsePayload{
		SessionID: "ses-001",
		Accepted:  true,
	})

	if !sm.RouteToPeer(hub, "ses-001", agentClient, protocol.MsgTypeConnectResponse, msg) {
		t.Error("Expected to route message to peer")
	}

	// Close session
	sm.CloseSession("ses-001")
	session, _ = sm.GetSession("ses-001")
	if session.Status != SessionClosed {
		t.Errorf("Expected status closed, got %s", session.Status)
	}

	// Test RemoveClientSessions
	sm.CreateSession("ses-002", appClient, "dev-001", []string{"shell"})
	sm.RemoveClientSessions(appClient.ID)
	if _, ok := sm.GetSession("ses-002"); ok {
		t.Error("Expected session ses-002 to be removed")
	}
}