package agent

import (
	"io"
	"log/slog"
	"sync"

	"github.com/cloudbridge/cloudbridge/internal/protocol"
	"github.com/cloudbridge/cloudbridge/internal/tunnel"
)

// SessionManager manages active tunnel sessions on the agent.
// Each session corresponds to a connection from a remote app.
type SessionManager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

// Session represents an active tunnel session with a remote app.
type Session struct {
	ID       string
	Tunnel   *tunnel.Tunnel
	Shell    io.ReadWriteCloser
	done     chan struct{}
}

// NewSessionManager creates a new session manager.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

// CreateSession sets up a new session with a relay transport.
func (sm *SessionManager) CreateSession(sessionID, relayAddr string) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Create relay transport as responder
	transport := tunnel.NewRelayTransportAsResponder(relayAddr, sessionID)
	tun := tunnel.NewTunnel(transport)

	if err := tun.Open(); err != nil {
		return nil, err
	}

	session := &Session{
		ID:     sessionID,
		Tunnel: tun,
		done:   make(chan struct{}),
	}

	sm.sessions[sessionID] = session
	slog.Info("session created", "session_id", sessionID)

	return session, nil
}

// StartShell starts a shell proxy for the session.
func (s *Session) StartShell() error {
	slog.Info("starting shell for session", "session_id", s.ID)

	// For now, we use the relay connection directly.
	// The frame-based multiplexing will be added in Phase 2.
	// For MVP, we write raw data through the relay transport.

	// This will be enhanced when we add the full frame protocol
	// to multiplex multiple streams over a single tunnel.
	return nil
}

// Close terminates the session.
func (s *Session) Close() error {
	close(s.done)
	if s.Shell != nil {
		s.Shell.Close()
	}
	if s.Tunnel != nil {
		return s.Tunnel.Close()
	}
	return nil
}

// RemoveSession removes a session from the manager.
func (sm *SessionManager) RemoveSession(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, ok := sm.sessions[sessionID]; ok {
		session.Close()
		delete(sm.sessions, sessionID)
		slog.Info("session removed", "session_id", sessionID)
	}
}

// GetSession returns a session by ID.
func (sm *SessionManager) GetSession(sessionID string) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	session, ok := sm.sessions[sessionID]
	return session, ok
}

// ListSessions returns all active sessions.
func (sm *SessionManager) ListSessions() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make([]*Session, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		result = append(result, s)
	}
	return result
}

// ProxyData bidirectionally copies data between a tunnel stream and a shell.
// This is the core data path for MVP.
func ProxyData(shell io.ReadWriteCloser, conn io.ReadWriteCloser) error {
	var wg sync.WaitGroup
	wg.Add(2)
	errCh := make(chan error, 2)

	// Shell -> Connection
	go func() {
		defer wg.Done()
		_, err := io.Copy(conn, shell)
		errCh <- err
	}()

	// Connection -> Shell
	go func() {
		defer wg.Done()
		_, err := io.Copy(shell, conn)
		errCh <- err
	}()

	wg.Wait()
	close(errCh)

	// Return the first error
	for err := range errCh {
		if err != nil {
			return err
		}
	}
	return nil
}

// ProxyDataFrames bidirectionally copies data using the frame protocol.
// This wraps raw data in protocol.Frame objects for multiplexed transport.
func ProxyDataFrames(shell io.ReadWriteCloser, transport tunnel.Transport, streamID protocol.StreamID) error {
	var wg sync.WaitGroup
	wg.Add(2)
	errCh := make(chan error, 2)

	// Shell -> Transport (wrap in frames)
	go func() {
		defer wg.Done()
		buf := make([]byte, protocol.MaxPayloadSize)
		for {
			n, err := shell.Read(buf)
			if err != nil {
				errCh <- err
				return
			}

			frame := &protocol.Frame{
				StreamID: streamID,
				Type:     protocol.FrameData,
				Payload:  buf[:n],
			}

			data, err := frame.MarshalBinary()
			if err != nil {
				errCh <- err
				return
			}

			if err := transport.WriteFrame(data); err != nil {
				errCh <- err
				return
			}
		}
	}()

	// Transport -> Shell (unwrap frames)
	go func() {
		defer wg.Done()
		for {
			data, err := transport.ReadFrame()
			if err != nil {
				errCh <- err
				return
			}

			frame := &protocol.Frame{}
			if err := frame.UnmarshalBinary(data); err != nil {
				errCh <- err
				return
			}

			if frame.StreamID == streamID && frame.Type == protocol.FrameData {
				if _, err := shell.Write(frame.Payload); err != nil {
					errCh <- err
					return
				}
			}
		}
	}()

	wg.Wait()

	shell.Close()

	// Return the first non-nil error
	for err := range errCh {
		if err != nil {
			return err
		}
	}
	return nil
}