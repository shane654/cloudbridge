// Package tunnel defines the transport interface for CloudBridge connections.
// A Tunnel carries multiplexed streams between the app and the agent.
package tunnel

import (
	"io"
	"net"
)

// Transport is the interface for underlying data channels (WebRTC, QUIC, Relay TCP).
type Transport interface {
	// Open opens the transport connection.
	Open() error

	// Close closes the transport connection.
	Close() error

	// ReadFrame reads a frame from the transport.
	ReadFrame() ([]byte, error)

	// WriteFrame writes a frame to the transport.
	WriteFrame(data []byte) error

	// LocalAddr returns the local network address.
	LocalAddr() net.Addr

	// RemoteAddr returns the remote network address.
	RemoteAddr() net.Addr
}

// Tunnel manages a multiplexed connection over a Transport.
// It reads/writes protocol.Frame objects over a single transport.
type Tunnel struct {
	transport Transport
	closed    chan struct{}
}

// NewTunnel creates a new tunnel over the given transport.
func NewTunnel(transport Transport) *Tunnel {
	return &Tunnel{
		transport: transport,
		closed:    make(chan struct{}),
	}
}

// Open establishes the transport connection.
func (t *Tunnel) Open() error {
	return t.transport.Open()
}

// Close shuts down the transport connection.
func (t *Tunnel) Close() error {
	close(t.closed)
	return t.transport.Close()
}

// IsClosed returns whether the tunnel has been closed.
func (t *Tunnel) IsClosed() bool {
	select {
	case <-t.closed:
		return true
	default:
		return false
	}
}

// ReadWriteCloser wraps a stream within a tunnel as an io.ReadWriteCloser.
type ReadWriteCloser struct {
	readCh  chan []byte
	writeCh chan []byte
	closeCh chan struct{}
}

// NewReadWriteCloser creates a bidirectional stream.
func NewReadWriteCloser() *ReadWriteCloser {
	return &ReadWriteCloser{
		readCh:  make(chan []byte, 64),
		writeCh: make(chan []byte, 64),
		closeCh: make(chan struct{}),
	}
}

// Read reads data from the stream.
func (rwc *ReadWriteCloser) Read(p []byte) (n int, err error) {
	select {
	case data, ok := <-rwc.readCh:
		if !ok {
			return 0, io.EOF
		}
		n = copy(p, data)
		return n, nil
	case <-rwc.closeCh:
		return 0, io.EOF
	}
}

// Write writes data to the stream.
func (rwc *ReadWriteCloser) Write(p []byte) (n int, err error) {
	select {
	case rwc.writeCh <- p:
		return len(p), nil
	case <-rwc.closeCh:
		return 0, io.ErrClosedPipe
	}
}

// Close closes the stream.
func (rwc *ReadWriteCloser) Close() error {
	close(rwc.closeCh)
	return nil
}

// ReadCh returns the channel for incoming data.
func (rwc *ReadWriteCloser) ReadCh() chan []byte {
	return rwc.readCh
}

// WriteCh returns the channel for outgoing data.
func (rwc *ReadWriteCloser) WriteCh() chan []byte {
	return rwc.writeCh
}