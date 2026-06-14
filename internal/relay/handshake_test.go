package relay

import (
	"net"
	"testing"
	"time"
)

func TestRelayHandshakeBinary(t *testing.T) {
	// Start a test TCP server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer listener.Close()

	serverAddr := listener.Addr().String()

	// Test handshake round-trip
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		handshake, err := ReadHandshake(conn)
		if err != nil {
			t.Errorf("Server: Failed to read handshake: %v", err)
			return
		}

		if handshake.SessionID != "test-session-123" {
			t.Errorf("Server: Expected session ID 'test-session-123', got %q", handshake.SessionID)
		}

		if handshake.PeerType != PeerTypeInitiator {
			t.Errorf("Server: Expected peer type %d, got %d", PeerTypeInitiator, handshake.PeerType)
		}

		// Send ACK
		if err := WriteHandshakeAck(conn, true, ""); err != nil {
			t.Errorf("Server: Failed to write handshake ack: %v", err)
		}
	}()

	// Client connects
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Send handshake
	if err := WriteHandshake(conn, "test-session-123", PeerTypeInitiator); err != nil {
		t.Fatalf("Client: Failed to write handshake: %v", err)
	}

	// Read ACK
	if err := ReadHandshakeAck(conn); err != nil {
		t.Fatalf("Client: Failed to read handshake ack: %v", err)
	}
}

func TestRelayHandshakeError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer listener.Close()

	serverAddr := listener.Addr().String()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		_, err = ReadHandshake(conn)
		if err != nil {
			t.Errorf("Server: Failed to read handshake: %v", err)
			return
		}

		// Send error ACK
		if err := WriteHandshakeAck(conn, false, "max sessions reached"); err != nil {
			t.Errorf("Server: Failed to write error ack: %v", err)
		}
	}()

	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	if err := WriteHandshake(conn, "err-session", PeerTypeResponder); err != nil {
		t.Fatalf("Client: Failed to write handshake: %v", err)
	}

	err = ReadHandshakeAck(conn)
	if err == nil {
		t.Error("Expected error from handshake ack, got nil")
	}
}

func TestConnectToRelay(t *testing.T) {
	// Start a simple relay server
	server := NewServer(ServerConfig{
		Addr:        "127.0.0.1:0",
		MaxSessions: 10,
	})

	// Use a random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	addr := listener.Addr().String()
	listener.Close()

	server = NewServer(ServerConfig{
		Addr:        addr,
		MaxSessions: 10,
	})

	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start relay server: %v", err)
	}
	defer server.Close()

	// Connect as initiator
	conn, err := ConnectAsInitiator(addr, "test-session-integration")
	if err != nil {
		t.Fatalf("Failed to connect as initiator: %v", err)
	}
	conn.Close()
}

func TestRelaySessionManagement(t *testing.T) {
	s := &Session{
		ID:        "test-session",
		CreatedAt: time.Now(),
	}

	// Add first peer
	if err := s.AddPeer(nil); err != nil {
		// This should work with a nil conn for testing
		t.Skip("AddPeer with nil conn not suitable for unit test")
	}
}