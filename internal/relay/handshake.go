package relay

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

// Relay handshake protocol (binary):
//
// Client → Server:
//   Magic (4 bytes): "CBLD"
//   Version (1 byte): 0x01
//   SessionID length (2 bytes, big-endian)
//   SessionID (variable)
//   PeerType (1 byte): 0x01=initiator, 0x02=responder
//
// Server → Client:
//   Magic (4 bytes): "CBLD"
//   Status (1 byte): 0x00=ok, 0x01=error
//   If error: message length (2 bytes) + message (variable)

const (
	handshakeMagic    = "CBLD"
	handshakeVersion  = 0x01
	handshakeHeaderLen = 4 + 1 + 2 + 1 // magic + version + sessionID len + peerType

	PeerTypeInitiator  = 0x01
	PeerTypeResponder  = 0x02

	statusOK    = 0x00
	statusError = 0x01
)

// HandshakeMessage is the client handshake payload.
type HandshakeMessage struct {
	SessionID string
	PeerType  byte
}

// WriteHandshake sends a client handshake to the relay server.
func WriteHandshake(conn net.Conn, sessionID string, peerType byte) error {
	idBytes := []byte(sessionID)
	if len(idBytes) > 65535 {
		return fmt.Errorf("session ID too long: %d bytes", len(idBytes))
	}

	buf := make([]byte, 0, handshakeHeaderLen+len(idBytes))
	buf = append(buf, handshakeMagic...)
	buf = append(buf, handshakeVersion)
	buf = append(buf, byte(len(idBytes)>>8), byte(len(idBytes)))
	buf = append(buf, idBytes...)
	buf = append(buf, peerType)

	_, err := conn.Write(buf)
	return err
}

// ReadHandshake reads a client handshake from a relay connection.
func ReadHandshake(conn net.Conn) (*HandshakeMessage, error) {
	reader := bufio.NewReader(conn)

	// Read magic
	magic := make([]byte, 4)
	if _, err := io.ReadFull(reader, magic); err != nil {
		return nil, fmt.Errorf("read magic: %w", err)
	}
	if string(magic) != handshakeMagic {
		return nil, fmt.Errorf("invalid magic: %q", string(magic))
	}

	// Read version
	version, err := reader.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("read version: %w", err)
	}
	if version != handshakeVersion {
		return nil, fmt.Errorf("unsupported version: %d", version)
	}

	// Read session ID length
	idLenBuf := make([]byte, 2)
	if _, err := io.ReadFull(reader, idLenBuf); err != nil {
		return nil, fmt.Errorf("read session ID length: %w", err)
	}
	idLen := int(binary.BigEndian.Uint16(idLenBuf))

	if idLen == 0 || idLen > 256 {
		return nil, fmt.Errorf("invalid session ID length: %d", idLen)
	}

	// Read session ID
	sessionID := make([]byte, idLen)
	if _, err := io.ReadFull(reader, sessionID); err != nil {
		return nil, fmt.Errorf("read session ID: %w", err)
	}

	// Read peer type
	peerType, err := reader.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("read peer type: %w", err)
	}
	if peerType != PeerTypeInitiator && peerType != PeerTypeResponder {
		return nil, fmt.Errorf("invalid peer type: %d", peerType)
	}

	return &HandshakeMessage{
		SessionID: string(sessionID),
		PeerType:  peerType,
	}, nil
}

// WriteHandshakeAck sends a server handshake acknowledgment.
func WriteHandshakeAck(conn net.Conn, ok bool, errMsg string) error {
	buf := make([]byte, 0, 5)
	buf = append(buf, handshakeMagic...)

	if ok {
		buf = append(buf, statusOK)
		_, err := conn.Write(buf)
		return err
	}

	buf = append(buf, statusError)
	msgBytes := []byte(errMsg)
	if len(msgBytes) > 65535 {
		msgBytes = msgBytes[:65535]
	}
	buf = append(buf, byte(len(msgBytes)>>8), byte(len(msgBytes)))
	buf = append(buf, msgBytes...)

	_, err := conn.Write(buf)
	return err
}

// ReadHandshakeAck reads a server handshake acknowledgment.
func ReadHandshakeAck(conn net.Conn) error {
	reader := bufio.NewReader(conn)

	// Read magic
	magic := make([]byte, 4)
	if _, err := io.ReadFull(reader, magic); err != nil {
		return fmt.Errorf("read ack magic: %w", err)
	}
	if string(magic) != handshakeMagic {
		return fmt.Errorf("invalid ack magic: %q", string(magic))
	}

	// Read status
	status, err := reader.ReadByte()
	if err != nil {
		return fmt.Errorf("read ack status: %w", err)
	}

	if status == statusOK {
		return nil
	}

	// Read error message
	msgLenBuf := make([]byte, 2)
	if _, err := io.ReadFull(reader, msgLenBuf); err != nil {
		return fmt.Errorf("read error message length: %w", err)
	}
	msgLen := int(binary.BigEndian.Uint16(msgLenBuf))

	msgBytes := make([]byte, msgLen)
	if _, err := io.ReadFull(reader, msgBytes); err != nil {
		return fmt.Errorf("read error message: %w", err)
	}

	return fmt.Errorf("relay handshake failed: %s", string(msgBytes))
}