// Package stun implements a STUN server per RFC 5389 for NAT type detection.
// It responds to Binding Requests with the client's reflexive address.
package stun

import (
	"encoding/binary"
	"fmt"
	"log/slog"
	"net"
	"time"
)

// STUN message types (RFC 5389).
const (
	StunBindingRequest  = 0x0001
	StunBindingResponse = 0x0101
)

// STUN attribute types.
const (
	AttrMappedAddress = 0x0001
	AttrXORMappedAddress = 0x0020
)

// STUN magic cookie (RFC 5389).
const StunMagicCookie = 0x2112A442

// ServerConfig holds configuration for the STUN server.
type ServerConfig struct {
	// Addr is the UDP address to listen on (e.g., ":10978").
	Addr string
}

// DefaultSTUNConfig returns a ServerConfig with defaults.
func DefaultSTUNConfig() ServerConfig {
	return ServerConfig{
		Addr: ":10978",
	}
}

// Server is a STUN server that responds to Binding Requests.
type Server struct {
	config ServerConfig
	conn   *net.UDPConn
	done   chan struct{}
}

// NewServer creates a new STUN server.
func NewServer(cfg ServerConfig) *Server {
	return &Server{
		config: cfg,
		done:   make(chan struct{}),
	}
}

// Start begins listening for STUN requests.
func (s *Server) Start() error {
	addr, err := net.ResolveUDPAddr("udp", s.config.Addr)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	s.conn = conn

	slog.Info("STUN server starting", "addr", s.config.Addr)

	go s.serve()
	return nil
}

// Close stops the STUN server.
func (s *Server) Close() error {
	close(s.done)
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

// serve reads STUN messages and responds to Binding Requests.
func (s *Server) serve() {
	buf := make([]byte, 576) // Minimum MTU for STUN messages

	for {
		select {
		case <-s.done:
			return
		default:
		}

		s.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, remoteAddr, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			if s.isClosed() {
				return
			}
			slog.Error("STUN read error", "err", err)
			continue
		}

		if err := s.handleMessage(buf[:n], remoteAddr); err != nil {
			slog.Debug("STUN message handling error", "err", err, "remote", remoteAddr)
		}
	}
}

// isClosed checks if the server has been closed.
func (s *Server) isClosed() bool {
	select {
	case <-s.done:
		return true
	default:
		return false
	}
}

// handleMessage processes a STUN message and responds if it's a Binding Request.
func (s *Server) handleMessage(data []byte, remoteAddr *net.UDPAddr) error {
	if len(data) < 20 {
		return &stunError{"message too short"}
	}

	// Parse STUN header
	msgType := binary.BigEndian.Uint16(data[0:2])
	msgLength := binary.BigEndian.Uint16(data[2:4])
	magicCookie := binary.BigEndian.Uint32(data[4:8])

	if magicCookie != StunMagicCookie {
		return &stunError{"invalid magic cookie"}
	}

	// Only handle Binding Requests
	if msgType != StunBindingRequest {
		return &stunError{fmt.Sprintf("unsupported message type: 0x%04x", msgType)}
	}

	if len(data) < 20+int(msgLength) {
		return &stunError{"message truncated"}
	}

	// Transaction ID (bytes 8-20)
	transactionID := make([]byte, 12)
	copy(transactionID, data[8:20])

	// Build Binding Response with XOR-MAPPED-ADDRESS attribute
	response, err := s.buildBindingResponse(transactionID, remoteAddr)
	if err != nil {
		return err
	}

	_, err = s.conn.WriteToUDP(response, remoteAddr)
	return err
}

// buildBindingResponse creates a STUN Binding Response with XOR-MAPPED-ADDRESS.
func (s *Server) buildBindingResponse(transactionID []byte, mappedAddr *net.UDPAddr) ([]byte, error) {
	ip := mappedAddr.IP.To4()
	if ip == nil {
		// Handle IPv6
		ip = mappedAddr.IP.To16()
	}

	port := uint16(mappedAddr.Port)

	// XOR-MAPPED-ADDRESS attribute value:
	//   0x00 (reserved) | family (0x01=IPv4, 0x02=IPv6) | xor-port | xor-ip
	var family byte = 0x01 // IPv4
	xorPort := port ^ uint16(StunMagicCookie>>16)

	// Build XOR key from magic cookie bytes
	cookieBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(cookieBytes, StunMagicCookie)

	var xorIP []byte
	if len(ip) == 4 {
		family = 0x01
		xorIP = make([]byte, 4)
		for i := 0; i < 4; i++ {
			xorIP[i] = ip[i] ^ cookieBytes[i]
		}
	} else {
		family = 0x02
		xorIP = make([]byte, 16)
		// For IPv6, XOR with magic cookie + transaction ID
		for i := 0; i < 4; i++ {
			xorIP[i] = ip[i] ^ cookieBytes[i]
		}
		for i := 4; i < 16; i++ {
			xorIP[i] = ip[i] ^ transactionID[i-4]
		}
	}

	// Attribute value: reserved(1) + family(1) + xor-port(2) + xor-ip(4 or 16)
	attrValueLen := 4 + len(xorIP)
	attrValue := make([]byte, attrValueLen)
	attrValue[0] = 0x00 // reserved
	attrValue[1] = family
	binary.BigEndian.PutUint16(attrValue[2:4], xorPort)
	copy(attrValue[4:], xorIP)

	// Attribute: type(2) + length(2) + value
	attrLen := 4 + len(attrValue)
	attr := make([]byte, attrLen)
	binary.BigEndian.PutUint16(attr[0:2], AttrXORMappedAddress)
	binary.BigEndian.PutUint16(attr[2:4], uint16(len(attrValue)))
	copy(attr[4:], attrValue)

	// STUN Response: type(2) + length(2) + magic(4) + transactionID(12) + attributes
	msgLength := len(attr)
	response := make([]byte, 20+msgLength)

	binary.BigEndian.PutUint16(response[0:2], StunBindingResponse)
	binary.BigEndian.PutUint16(response[2:4], uint16(msgLength))
	binary.BigEndian.PutUint32(response[4:8], StunMagicCookie)
	copy(response[8:20], transactionID)
	copy(response[20:], attr)

	return response, nil
}

// stunError is a simple error type for STUN processing errors.
type stunError struct {
	msg string
}

func (e *stunError) Error() string {
	return "stun: " + e.msg
}