package tunnel

import (
	"fmt"
	"net"
	"sync"

	"github.com/cloudbridge/cloudbridge/internal/protocol"
	"github.com/cloudbridge/cloudbridge/internal/relay"
)

// RelayTransport implements Transport over a TCP relay connection.
// It reads/writes protocol.Frame objects over the relay's binary stream.
type RelayTransport struct {
	addr      string
	sessionID string
	peerType  byte // PeerTypeInitiator or PeerTypeResponder
	conn      net.Conn
	mu        sync.Mutex
}

// NewRelayTransportAsInitiator creates a relay transport for the app (initiator) side.
func NewRelayTransportAsInitiator(addr, sessionID string) *RelayTransport {
	return &RelayTransport{
		addr:      addr,
		sessionID: sessionID,
		peerType:  relay.PeerTypeInitiator,
	}
}

// NewRelayTransportAsResponder creates a relay transport for the agent (responder) side.
func NewRelayTransportAsResponder(addr, sessionID string) *RelayTransport {
	return &RelayTransport{
		addr:      addr,
		sessionID: sessionID,
		peerType:  relay.PeerTypeResponder,
	}
}

// Open establishes the relay connection and completes the handshake.
func (rt *RelayTransport) Open() error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	conn, err := relay.ConnectToRelay(rt.addr, rt.sessionID, rt.peerType)
	if err != nil {
		return fmt.Errorf("relay connect: %w", err)
	}

	rt.conn = conn
	return nil
}

// Close closes the relay connection.
func (rt *RelayTransport) Close() error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	if rt.conn != nil {
		return rt.conn.Close()
	}
	return nil
}

// ReadFrame reads a frame from the relay connection.
func (rt *RelayTransport) ReadFrame() ([]byte, error) {
	frame, err := protocol.ReadFrom(rt.conn)
	if err != nil {
		return nil, fmt.Errorf("read frame from relay: %w", err)
	}

	data, err := frame.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("marshal frame: %w", err)
	}

	return data, nil
}

// WriteFrame writes a frame to the relay connection.
func (rt *RelayTransport) WriteFrame(data []byte) error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	frame := &protocol.Frame{}
	if err := frame.UnmarshalBinary(data); err != nil {
		return fmt.Errorf("unmarshal frame: %w", err)
	}

	if _, err := frame.WriteTo(rt.conn); err != nil {
		return fmt.Errorf("write frame to relay: %w", err)
	}

	return nil
}

// LocalAddr returns the local network address.
func (rt *RelayTransport) LocalAddr() net.Addr {
	if rt.conn != nil {
		return rt.conn.LocalAddr()
	}
	return nil
}

// RemoteAddr returns the remote network address.
func (rt *RelayTransport) RemoteAddr() net.Addr {
	if rt.conn != nil {
		return rt.conn.RemoteAddr()
	}
	return nil
}