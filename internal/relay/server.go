// Package relay implements a TCP relay server that forwards traffic between
// two peers when direct P2P connection is not possible.
package relay

import (
	"io"
	"log/slog"
	"net"
	"sync"
	"time"
)

const (
	// DefaultRelayPort is the default TCP port for the relay server.
	DefaultRelayPort = 10988

	// RelayBufferSize is the buffer size for data forwarding.
	RelayBufferSize = 32 * 1024 // 32 KiB

	// DefaultQuotaBytes is the free-tier monthly bandwidth quota (1 GB).
	DefaultQuotaBytes = 1 * 1024 * 1024 * 1024

	// DefaultQuotaRate is the free-tier bandwidth rate limit (1 Mbps = 125 KB/s).
	DefaultQuotaRate = 125 * 1024

	// HandshakeTimeout is the timeout for relay handshake.
	HandshakeTimeout = 30 * time.Second
)

// ServerConfig holds configuration for the relay server.
type ServerConfig struct {
	// Addr is the TCP address to listen on (e.g., ":10988").
	Addr string

	// MaxSessions is the maximum number of concurrent relay sessions.
	MaxSessions int

	// QuotaBytes is the monthly bandwidth quota per free-tier user (0 = unlimited).
	QuotaBytes int64

	// QuotaRate is the per-connection rate limit in bytes/sec (0 = unlimited).
	QuotaRate int
}

// DefaultRelayConfig returns a ServerConfig with defaults.
func DefaultRelayConfig() ServerConfig {
	return ServerConfig{
		Addr:        ":10988",
		MaxSessions: 1000,
		QuotaBytes:  DefaultQuotaBytes,
		QuotaRate:   DefaultQuotaRate,
	}
}

// Server is a TCP relay server.
type Server struct {
	config   ServerConfig
	listener net.Listener
	sessions map[string]*Session
	mu       sync.RWMutex
	done     chan struct{}
}

// NewServer creates a new relay server.
func NewServer(cfg ServerConfig) *Server {
	return &Server{
		config:   cfg,
		sessions: make(map[string]*Session),
		done:     make(chan struct{}),
	}
}

// Start begins accepting relay connections.
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.config.Addr)
	if err != nil {
		return err
	}
	s.listener = listener

	slog.Info("relay server starting", "addr", s.config.Addr)

	go s.acceptLoop()
	return nil
}

// Close stops the relay server and closes all active sessions.
func (s *Server) Close() error {
	close(s.done)

	s.mu.Lock()
	defer s.mu.Unlock()

	for id, session := range s.sessions {
		session.Close()
		delete(s.sessions, id)
	}

	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

// SessionCount returns the number of active relay sessions.
func (s *Server) SessionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions)
}

// acceptLoop accepts incoming connections and matches them to sessions.
func (s *Server) acceptLoop() {
	for {
		select {
		case <-s.done:
			return
		default:
		}

		conn, err := s.listener.Accept()
		if err != nil {
			if s.isClosed() {
				return
			}
			slog.Error("relay accept error", "err", err)
			continue
		}

		go s.handleConnection(conn)
	}
}

func (s *Server) isClosed() bool {
	select {
	case <-s.done:
		return true
	default:
		return false
	}
}

// handleConnection processes a new relay connection.
func (s *Server) handleConnection(conn net.Conn) {
	remoteAddr := conn.RemoteAddr().String()
	slog.Info("relay connection", "remote", remoteAddr)

	// Set handshake deadline
	conn.SetReadDeadline(time.Now().Add(HandshakeTimeout))

	// Read handshake
	handshake, err := ReadHandshake(conn)
	if err != nil {
		slog.Error("relay handshake failed", "remote", remoteAddr, "err", err)
		WriteHandshakeAck(conn, false, err.Error())
		conn.Close()
		return
	}

	slog.Info("relay handshake received",
		"session_id", handshake.SessionID,
		"peer_type", handshake.PeerType,
		"remote", remoteAddr,
	)

	// Clear deadline for normal operation
	conn.SetReadDeadline(time.Time{})

	s.mu.Lock()
	session, exists := s.sessions[handshake.SessionID]
	if !exists {
		// First peer to connect for this session
		if len(s.sessions) >= s.config.MaxSessions {
			s.mu.Unlock()
			slog.Warn("max sessions reached, rejecting", "session_id", handshake.SessionID)
			WriteHandshakeAck(conn, false, "max sessions reached")
			conn.Close()
			return
		}

		session = &Session{
			ID:         handshake.SessionID,
			CreatedAt:  time.Now(),
			QuotaBytes: s.config.QuotaBytes,
			QuotaRate:  s.config.QuotaRate,
		}
		s.sessions[handshake.SessionID] = session
	}
	s.mu.Unlock()

	// Assign this connection to the session
	if err := session.AddPeer(conn); err != nil {
		slog.Error("relay session full", "session_id", handshake.SessionID, "err", err)
		WriteHandshakeAck(conn, false, err.Error())
		conn.Close()

		// Clean up empty session
		s.mu.Lock()
		if session.PeerCount() == 0 {
			delete(s.sessions, handshake.SessionID)
		}
		s.mu.Unlock()
		return
	}

	// Send handshake acknowledgment
	if err := WriteHandshakeAck(conn, true, ""); err != nil {
		slog.Error("relay handshake ack failed", "session_id", handshake.SessionID, "err", err)
		conn.Close()
		return
	}

	// If session is complete (2 peers), start forwarding
	if session.PeerCount() == 2 {
		slog.Info("relay session established", "session_id", handshake.SessionID)

		go func() {
			s.relaySession(session)

			// Clean up after session ends
			s.mu.Lock()
			delete(s.sessions, handshake.SessionID)
			s.mu.Unlock()
			slog.Info("relay session ended", "session_id", handshake.SessionID)
		}()
	} else {
		slog.Info("relay session waiting for peer", "session_id", handshake.SessionID, "peer_count", session.PeerCount())
	}
}

// relaySession forwards data between the two peers of a session.
func (s *Server) relaySession(session *Session) {
	peers := session.GetPeers()
	if len(peers) != 2 {
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// Forward: peer[0] -> peer[1]
	go func() {
		defer wg.Done()
		n, err := io.Copy(peers[1], peers[0])
		session.AddBytes(n)
		if err != nil {
			slog.Debug("relay forward error", "direction", "0->1", "err", err)
		}
		peers[1].Close()
	}()

	// Forward: peer[1] -> peer[0]
	go func() {
		defer wg.Done()
		n, err := io.Copy(peers[0], peers[1])
		session.AddBytes(n)
		if err != nil {
			slog.Debug("relay forward error", "direction", "1->0", "err", err)
		}
		peers[0].Close()
	}()

	wg.Wait()
	session.Close()
}