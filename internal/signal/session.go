package signal

import (
	"log/slog"
	"sync"
	"time"
)

// Session represents an active connection session between an app and an agent.
type Session struct {
	ID         string
	Initiator  *Client // The app that initiated the connection
	Responder  *Client // The agent that accepted the connection
	DeviceID   string
	Protocols  []string
	CreatedAt  time.Time
	Status     SessionStatus
}

// SessionStatus represents the state of a session.
type SessionStatus string

const (
	SessionPending   SessionStatus = "pending"
	SessionAccepted  SessionStatus = "accepted"
	SessionRejected  SessionStatus = "rejected"
	SessionActive    SessionStatus = "active"
	SessionClosed    SessionStatus = "closed"
)

// SessionManager tracks active sessions and routes messages between peers.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session // sessionID -> Session
	byClient map[string][]*Session // clientID -> sessions
}

// NewSessionManager creates a new SessionManager.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
		byClient: make(map[string][]*Session),
	}
}

// CreateSession creates a new session for a connection request.
func (sm *SessionManager) CreateSession(sessionID string, initiator *Client, deviceID string, protocols []string) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session := &Session{
		ID:        sessionID,
		Initiator: initiator,
		DeviceID:  deviceID,
		Protocols: protocols,
		CreatedAt: time.Now(),
		Status:    SessionPending,
	}

	sm.sessions[sessionID] = session
	sm.byClient[initiator.ID] = append(sm.byClient[initiator.ID], session)

	slog.Info("session created",
		"session_id", sessionID,
		"initiator", initiator.ID,
		"device_id", deviceID,
		"protocols", protocols,
	)

	return session
}

// AcceptSession marks a session as accepted and records the responder.
func (sm *SessionManager) AcceptSession(sessionID string, responder *Client) (*Session, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[sessionID]
	if !ok {
		return nil, false
	}

	session.Responder = responder
	session.Status = SessionAccepted
	sm.byClient[responder.ID] = append(sm.byClient[responder.ID], session)

	slog.Info("session accepted",
		"session_id", sessionID,
		"responder", responder.ID,
	)

	return session, true
}

// RejectSession marks a session as rejected.
func (sm *SessionManager) RejectSession(sessionID string) (*Session, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[sessionID]
	if !ok {
		return nil, false
	}

	session.Status = SessionRejected
	slog.Info("session rejected", "session_id", sessionID)
	return session, true
}

// ActivateSession marks a session as fully active (transport established).
func (sm *SessionManager) ActivateSession(sessionID string) (*Session, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[sessionID]
	if !ok {
		return nil, false
	}

	session.Status = SessionActive
	slog.Info("session activated", "session_id", sessionID)
	return session, true
}

// CloseSession marks a session as closed.
func (sm *SessionManager) CloseSession(sessionID string) (*Session, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[sessionID]
	if !ok {
		return nil, false
	}

	session.Status = SessionClosed
	slog.Info("session closed", "session_id", sessionID)
	return session, true
}

// GetSession returns a session by ID.
func (sm *SessionManager) GetSession(sessionID string) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	session, ok := sm.sessions[sessionID]
	return session, ok
}

// GetPeer returns the other client in a session (given one client, find the other).
func (sm *SessionManager) GetPeer(sessionID string, client *Client) (*Client, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, ok := sm.sessions[sessionID]
	if !ok {
		return nil, false
	}

	if session.Initiator != nil && session.Initiator.ID == client.ID {
		return session.Responder, true
	}
	if session.Responder != nil && session.Responder.ID == client.ID {
		return session.Initiator, true
	}

	return nil, false
}

// RemoveClientSessions removes all sessions associated with a client (e.g., on disconnect).
func (sm *SessionManager) RemoveClientSessions(clientID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sessions := sm.byClient[clientID]
	for _, session := range sessions {
		session.Status = SessionClosed
		delete(sm.sessions, session.ID)
		slog.Info("session removed on disconnect", "session_id", session.ID, "client_id", clientID)
	}
	delete(sm.byClient, clientID)
}

// ListSessions returns all sessions (for debugging/monitoring).
func (sm *SessionManager) ListSessions() []Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make([]Session, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		result = append(result, *s)
	}
	return result
}

// RouteToPeer sends a message to the peer of a given client in a session.
// This is the core routing function for signaling messages.
func (sm *SessionManager) RouteToPeer(hub *Hub, sessionID string, sender *Client, msgType string, data []byte) bool {
	peer, ok := sm.GetPeer(sessionID, sender)
	if !ok || peer == nil {
		slog.Warn("no peer found for session routing",
			"session_id", sessionID,
			"sender", sender.ID,
		)
		return false
	}

	select {
	case peer.Send <- data:
		slog.Debug("routed message to peer",
			"session_id", sessionID,
			"from", sender.ID,
			"to", peer.ID,
			"type", msgType,
		)
		return true
	default:
		slog.Warn("peer send buffer full, dropping message",
			"peer_id", peer.ID,
			"session_id", sessionID,
		)
		return false
	}
}

// GenerateSessionID creates a new unique session identifier.
func GenerateSessionID() string {
	return "ses-" + time.Now().Format("20060102150405") + "-" + randomHex(4)
}

func randomHex(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = "0123456789abcdef"[time.Now().Nanosecond()%16]
	}
	return string(b)
}