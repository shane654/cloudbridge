package relay

import (
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Session represents a relay session between two peers.
type Session struct {
	ID         string
	CreatedAt  time.Time
	peers      [2]net.Conn
	peerCount  int
	bytes      atomic.Int64 // total bytes relayed
	QuotaBytes int64        // bandwidth quota (0 = unlimited)
	QuotaRate  int          // rate limit in bytes/sec (0 = unlimited)
	mu         sync.Mutex
	closed     bool
}

// AddPeer adds a connection to the session. Returns error if session is full.
func (s *Session) AddPeer(conn net.Conn) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.peerCount >= 2 {
		return &relayError{"session already has 2 peers"}
	}

	s.peers[s.peerCount] = conn
	s.peerCount++
	return nil
}

// PeerCount returns the number of connected peers.
func (s *Session) PeerCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.peerCount
}

// GetPeers returns a copy of the peer connections.
func (s *Session) GetPeers() []net.Conn {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]net.Conn, 0, s.peerCount)
	for i := 0; i < s.peerCount; i++ {
		if s.peers[i] != nil {
			result = append(result, s.peers[i])
		}
	}
	return result
}

// AddBytes adds to the total bytes relayed counter.
func (s *Session) AddBytes(n int64) {
	s.bytes.Add(n)
}

// BytesRelayed returns the total bytes relayed in this session.
func (s *Session) BytesRelayed() int64 {
	return s.bytes.Load()
}

// Close closes all peer connections in the session.
func (s *Session) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}
	s.closed = true

	for i := 0; i < s.peerCount; i++ {
		if s.peers[i] != nil {
			s.peers[i].Close()
		}
	}
}

// relayError is a simple error type for relay operations.
type relayError struct {
	msg string
}

func (e *relayError) Error() string {
	return "relay: " + e.msg
}