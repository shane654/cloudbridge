package signal

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// SendBufferSize is the number of messages buffered for each client.
	SendBufferSize = 256

	// PingInterval is how often we ping clients to check liveness.
	PingInterval = 30 * time.Second

	// PongWait is the maximum time we wait for a pong response.
	PongWait = 60 * time.Second
)

// ServerConfig holds configuration for the signal server.
type ServerConfig struct {
	// Addr is the address to listen on (e.g., ":10980").
	Addr string

	// Path is the WebSocket endpoint path (e.g., "/signal").
	Path string

	// HeartbeatInterval is how often agents should send heartbeats.
	HeartbeatInterval time.Duration
}

// DefaultConfig returns a ServerConfig with sensible defaults.
func DefaultConfig() ServerConfig {
	return ServerConfig{
		Addr:              ":10980",
		Path:              "/signal",
		HeartbeatInterval: 30 * time.Second,
	}
}

// Server is the WebSocket signaling server.
type Server struct {
	config         ServerConfig
	hub            *Hub
	sessionManager *SessionManager
	handler        *Handler
	upgrader       websocket.Upgrader
	httpServer     *http.Server
}

// NewServer creates a new signal server with the given config.
func NewServer(cfg ServerConfig) *Server {
	if cfg.HeartbeatInterval == 0 {
		cfg.HeartbeatInterval = 30 * time.Second
	}

	hub := NewHub()
	sessionManager := NewSessionManager()
	handler := NewHandler(hub, sessionManager)

	return &Server{
		config:         cfg,
		hub:            hub,
		sessionManager: sessionManager,
		handler:        handler,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin:     func(r *http.Request) bool { return true }, // TODO: validate origin in production
		},
	}
}

// Hub returns the underlying Hub for external access (e.g., REST API).
func (s *Server) Hub() *Hub {
	return s.hub
}

// SessionManager returns the underlying SessionManager.
func (s *Server) SessionManager() *SessionManager {
	return s.sessionManager
}

// Start begins listening for WebSocket connections.
// If apiHandler is provided, it will be called to register REST API routes.
func (s *Server) Start(apiRegister func(mux *http.ServeMux)) error {
	mux := http.NewServeMux()
	mux.HandleFunc(s.config.Path, s.handleWebSocket)

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Register REST API routes if provided
	if apiRegister != nil {
		apiRegister(mux)
	}

	s.httpServer = &http.Server{
		Addr:    s.config.Addr,
		Handler: mux,
	}

	slog.Info("signal server starting", "addr", s.config.Addr, "path", s.config.Path)

	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("signal server shutting down")
	return s.httpServer.Shutdown(ctx)
}

// handleWebSocket upgrades an HTTP connection to WebSocket and handles messages.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("websocket upgrade failed", "err", err)
		return
	}

	client := &Client{
		ID:      generateClientID(),
		IsAgent: r.URL.Query().Get("type") == "agent",
		Conn:    conn,
		Send:    make(chan []byte, SendBufferSize),
		Close:   make(chan struct{}),
	}

	s.hub.RegisterClient(client)
	slog.Info("websocket client connected", "client_id", client.ID, "is_agent", client.IsAgent)

	// Start writer goroutine
	go s.writePump(client)

	// Run reader in current goroutine (blocks until disconnected)
	s.readPump(client)

	// Clean up sessions and client registration
	s.sessionManager.RemoveClientSessions(client.ID)
	s.hub.UnregisterClient(client)
}

// readPump reads messages from the WebSocket connection.
func (s *Server) readPump(client *Client) {
	defer client.Conn.Close()

	client.Conn.SetReadLimit(64 * 1024) // 64KB max message size
	client.Conn.SetReadDeadline(time.Now().Add(PongWait))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(PongWait))
		return nil
	})

	for {
		_, msg, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {
				slog.Error("websocket read error", "client_id", client.ID, "err", err)
			}
			return
		}

		if err := s.handler.HandleMessage(client, msg); err != nil {
			slog.Error("handle message error", "client_id", client.ID, "err", err)
		}
	}
}

// writePump writes messages to the WebSocket connection.
func (s *Server) writePump(client *Client) {
	ticker := time.NewTicker(PingInterval)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := client.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				slog.Error("websocket write error", "client_id", client.ID, "err", err)
				return
			}

		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-client.Close:
			return
		}
	}
}

// generateClientID returns a unique client identifier.
// TODO: use crypto/rand or UUID for production.
func generateClientID() string {
	return "cl-" + time.Now().Format("20060102150405")
}