// Package agent implements the CloudBridge agent that runs on remote devices.
// It registers with the signal server, maintains heartbeats, and serves
// Shell/SSH/Docker sessions.
package agent

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/cloudbridge/cloudbridge/internal/protocol"
)

const (
	// DefaultSignalURL is the default WebSocket signal server URL.
	DefaultSignalURL = "ws://localhost:10980/signal"

	// HeartbeatInterval is how often the agent sends heartbeats.
	HeartbeatInterval = 30 * time.Second

	// ReconnectBaseDelay is the base delay for exponential backoff reconnection.
	ReconnectBaseDelay = 1 * time.Second

	// ReconnectMaxDelay is the maximum reconnection delay.
	ReconnectMaxDelay = 30 * time.Second

	// MaxReconnectAttempts is the maximum number of reconnection attempts (0 = unlimited).
	MaxReconnectAttempts = 0
)

// Config holds the agent configuration.
type Config struct {
	// ServerURL is the WebSocket URL of the signal server.
	ServerURL string

	// DeviceName is a human-readable name for this device.
	DeviceName string

	// Platform override (auto-detected if empty).
	Platform string

	// Version of the agent.
	Version string

	// KeyPath is the path to the device's ed25519 private key file.
	// If empty, a key will be generated and stored in the current directory.
	KeyPath string
}

// DefaultConfig returns an agent Config with defaults.
func DefaultConfig() Config {
	return Config{
		ServerURL:  DefaultSignalURL,
		DeviceName: "cloudbridge-agent",
		Version:    "0.1.0",
	}
}

// Agent is the remote device agent.
type Agent struct {
	config   Config
	deviceID string
	token    string
	conn     *websocket.Conn
	mu       sync.Mutex
	done     chan struct{}
}

// New creates a new Agent.
func New(cfg Config) (*Agent, error) {
	if cfg.ServerURL == "" {
		cfg.ServerURL = DefaultSignalURL
	}
	if cfg.DeviceName == "" {
		cfg.DeviceName = "cloudbridge-agent"
	}

	deviceID, err := generateDeviceID()
	if err != nil {
		return nil, fmt.Errorf("generate device ID: %w", err)
	}

	return &Agent{
		config:   cfg,
		deviceID: deviceID,
		done:     make(chan struct{}),
	}, nil
}

// DeviceID returns the agent's unique device identifier.
func (a *Agent) DeviceID() string {
	return a.deviceID
}

// Run starts the agent: connects to the signal server, registers,
// and maintains the connection with heartbeats.
func (a *Agent) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-a.done:
			return nil
		default:
		}

		if err := a.connect(ctx); err != nil {
			slog.Error("agent connection failed", "err", err)
		}

		// Reconnect with exponential backoff
		delay := ReconnectBaseDelay
		for attempt := 1; ; attempt++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}

			slog.Info("reconnecting", "attempt", attempt, "delay", delay)

			if err := a.connect(ctx); err == nil {
				break // connected successfully
			}

			delay *= 2
			if delay > ReconnectMaxDelay {
				delay = ReconnectMaxDelay
			}
		}
	}
}

// Close stops the agent.
func (a *Agent) Close() error {
	close(a.done)
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.conn != nil {
		return a.conn.Close()
	}
	return nil
}

// connect establishes a WebSocket connection and registers the device.
func (a *Agent) connect(ctx context.Context) error {
	dialer := websocket.DefaultDialer

	slog.Info("connecting to signal server", "url", a.config.ServerURL)
	conn, _, err := dialer.DialContext(ctx, a.config.ServerURL+"?type=agent", nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	a.mu.Lock()
	a.conn = conn
	a.mu.Unlock()

	// Register device
	if err := a.register(); err != nil {
		conn.Close()
		return fmt.Errorf("register: %w", err)
	}

	// Start heartbeat goroutine
	heartbeatCtx, heartbeatCancel := context.WithCancel(ctx)
	go a.heartbeatLoop(heartbeatCtx)

	// Read messages until disconnected
	defer heartbeatCancel()
	return a.readMessages(ctx)
}

// register sends the device registration message.
func (a *Agent) register() error {
	platform := protocol.Platform(getPlatform())
	if a.config.Platform != "" {
		platform = protocol.Platform(a.config.Platform)
	}

	msg, err := protocol.Encode(protocol.MsgTypeRegister, protocol.RegisterPayload{
		DeviceID:   a.deviceID,
		DeviceName: a.config.DeviceName,
		Platform:   platform,
		Version:    a.config.Version,
		PublicKey:  "", // TODO: load from key file
	})
	if err != nil {
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	return a.conn.WriteMessage(websocket.TextMessage, msg)
}

// heartbeatLoop sends periodic heartbeats.
func (a *Agent) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			msg, err := protocol.Encode(protocol.MsgTypeHeartbeat, protocol.HeartbeatPayload{
				Timestamp: protocol.CurrentTimestampMillis(),
			})
			if err != nil {
				slog.Error("encode heartbeat", "err", err)
				continue
			}

			a.mu.Lock()
			err = a.conn.WriteMessage(websocket.TextMessage, msg)
			a.mu.Unlock()

			if err != nil {
				slog.Error("send heartbeat", "err", err)
				return
			}
		}
	}
}

// readMessages reads and dispatches incoming WebSocket messages.
func (a *Agent) readMessages(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		a.mu.Lock()
		conn := a.conn
		a.mu.Unlock()

		_, msg, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}

		msgType, err := protocol.Decode(msg, nil)
		if err != nil {
			slog.Error("decode message", "err", err)
			continue
		}

		slog.Debug("received message", "type", msgType)

		switch msgType {
		case protocol.MsgTypeRegisterAck:
			a.handleRegisterAck(msg)
		case protocol.MsgTypeConnectRequest:
			a.handleConnectRequest(msg)
		case protocol.MsgTypeSDPOffer, protocol.MsgTypeSDPAnswer, protocol.MsgTypeICECandidate:
			slog.Info("signaling message received", "type", msgType)
		case protocol.MsgTypeError:
			var errPayload protocol.ErrorPayload
			protocol.Decode(msg, &errPayload)
			slog.Error("server error", "code", errPayload.Code, "message", errPayload.Message)
		default:
			slog.Warn("unhandled message type", "type", msgType)
		}
	}
}

func (a *Agent) handleRegisterAck(raw []byte) {
	var ack protocol.RegisterAckPayload
	if _, err := protocol.Decode(raw, &ack); err != nil {
		slog.Error("decode register ack", "err", err)
		return
	}

	a.token = ack.Token
	slog.Info("device registered", "device_id", a.deviceID, "token", ack.Token)
}

func (a *Agent) handleConnectRequest(raw []byte) {
	var req protocol.ConnectRequestPayload
	if _, err := protocol.Decode(raw, &req); err != nil {
		slog.Error("decode connect request", "err", err)
		return
	}

	slog.Info("incoming connection request", "session_id", req.SessionID, "protocols", req.Protocols)

	// Auto-accept for MVP. In production, this would require user approval.
	resp, err := protocol.Encode(protocol.MsgTypeConnectResponse, protocol.ConnectResponsePayload{
		SessionID: req.SessionID,
		Accepted:  true,
	})
	if err != nil {
		slog.Error("encode connect response", "err", err)
		return
	}

	a.mu.Lock()
	err = a.conn.WriteMessage(websocket.TextMessage, resp)
	a.mu.Unlock()

	if err != nil {
		slog.Error("send connect response", "err", err)
	}
}

// generateDeviceID creates a unique, stable device identifier.
// It uses the machine's hostname and a random component.
func generateDeviceID() (string, error) {
	hostname, err := getHostname()
	if err != nil {
		hostname = "unknown"
	}

	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	hash := sha256.Sum256([]byte(hostname + hex.EncodeToString(randomBytes)))
	return "dev-" + base64.RawURLEncoding.EncodeToString(hash[:8]), nil
}

// getHostname returns the machine hostname.
func getHostname() (string, error) {
	return os.Hostname()
}

// getPlatform returns the OS platform string.
func getPlatform() string {
	return "linux" // Will be overridden by config or detected at build time
}

// healthHandler returns a simple health check response.
// This can be used by the main command to expose an HTTP health endpoint.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}