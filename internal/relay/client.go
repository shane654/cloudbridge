package relay

import (
	"fmt"
	"net"
	"time"
)

// ConnectToRelay establishes a TCP connection to a relay server and performs
// the handshake. Returns the established connection.
func ConnectToRelay(addr string, sessionID string, peerType byte) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("dial relay %s: %w", addr, err)
	}

	// Set deadline for handshake
	conn.SetDeadline(time.Now().Add(HandshakeTimeout))

	if err := WriteHandshake(conn, sessionID, peerType); err != nil {
		conn.Close()
		return nil, fmt.Errorf("write handshake: %w", err)
	}

	if err := ReadHandshakeAck(conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("read handshake ack: %w", err)
	}

	// Clear deadline for normal operation
	conn.SetDeadline(time.Time{})

	return conn, nil
}

// ConnectAsInitiator connects to the relay as the initiator (app side).
func ConnectAsInitiator(addr string, sessionID string) (net.Conn, error) {
	return ConnectToRelay(addr, sessionID, PeerTypeInitiator)
}

// ConnectAsResponder connects to the relay as the responder (agent side).
func ConnectAsResponder(addr string, sessionID string) (net.Conn, error) {
	return ConnectToRelay(addr, sessionID, PeerTypeResponder)
}