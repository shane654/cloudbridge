// Package protocol defines the CloudBridge wire protocol for multiplexed
// stream transport over WebRTC DataChannel, QUIC, or Relay connections.
package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// Stream identifiers for well-known channels.
const (
	StreamControl StreamID = 0x0000
	StreamSSH     StreamID = 0x0001
	StreamShell   StreamID = 0x0002
	StreamRDP     StreamID = 0x0003
	StreamDocker  StreamID = 0x0004
	StreamVNC     StreamID = 0x0005
)

// Frame types carried in the Type field.
const (
	FrameData          FrameType = 0x01
	FrameOpenStream     FrameType = 0x02
	FrameCloseStream   FrameType = 0x03
	FrameWindowUpdate  FrameType = 0x04
	FramePing          FrameType = 0x05
	FramePong          FrameType = 0x06
)

// Header size: StreamID(2) + Type(1) + Length(4) = 7 bytes.
const FrameHeaderSize = 7

// MaxPayloadSize is the maximum payload length in a single frame.
const MaxPayloadSize = 64 * 1024 // 64 KiB

// StreamID identifies a multiplexed stream within a tunnel connection.
type StreamID uint16

// FrameType identifies the kind of frame.
type FrameType uint8

// Frame is the unit of transmission on the wire.
//
// Wire format:
//
//	+----------+--------+-----------+----------+
//	| StreamID | Type   | Length    | Payload  |
//	| 2 bytes  | 1 byte | 4 bytes   | N bytes  |
//	+----------+--------+-----------+----------+
type Frame struct {
	StreamID StreamID
	Type     FrameType
	Payload  []byte
}

// MarshalBinary encodes the frame into its wire format.
func (f *Frame) MarshalBinary() ([]byte, error) {
	if len(f.Payload) > MaxPayloadSize {
		return nil, fmt.Errorf("payload too large: %d > %d", len(f.Payload), MaxPayloadSize)
	}

	buf := make([]byte, FrameHeaderSize+len(f.Payload))
	binary.BigEndian.PutUint16(buf[0:2], uint16(f.StreamID))
	buf[2] = byte(f.Type)
	binary.BigEndian.PutUint32(buf[3:7], uint32(len(f.Payload)))
	copy(buf[7:], f.Payload)

	return buf, nil
}

// UnmarshalBinary decodes a frame from its wire format.
func (f *Frame) UnmarshalBinary(data []byte) error {
	if len(data) < FrameHeaderSize {
		return errors.New("frame too short")
	}

	f.StreamID = StreamID(binary.BigEndian.Uint16(data[0:2]))
	f.Type = FrameType(data[2])
	length := binary.BigEndian.Uint32(data[3:7])

	if len(data) < FrameHeaderSize+int(length) {
		return fmt.Errorf("frame truncated: expected %d bytes payload, got %d", length, len(data)-FrameHeaderSize)
	}
	if length > MaxPayloadSize {
		return fmt.Errorf("payload too large: %d > %d", length, MaxPayloadSize)
	}

	f.Payload = make([]byte, length)
	copy(f.Payload, data[FrameHeaderSize:FrameHeaderSize+length])

	return nil
}

// WriteTo writes the frame to an io.Writer in wire format.
// Returns the number of bytes written.
func (f *Frame) WriteTo(w io.Writer) (int64, error) {
	data, err := f.MarshalBinary()
	if err != nil {
		return 0, err
	}
	n, err := w.Write(data)
	return int64(n), err
}

// ReadFrom reads a frame from an io.Reader.
// It reads the header first, then the payload.
func ReadFrom(r io.Reader) (*Frame, error) {
	header := make([]byte, FrameHeaderSize)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, fmt.Errorf("reading frame header: %w", err)
	}

	streamID := StreamID(binary.BigEndian.Uint16(header[0:2]))
	frameType := FrameType(header[2])
	length := binary.BigEndian.Uint32(header[3:7])

	if length > MaxPayloadSize {
		return nil, fmt.Errorf("payload too large: %d > %d", length, MaxPayloadSize)
	}

	payload := make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return nil, fmt.Errorf("reading frame payload: %w", err)
		}
	}

	return &Frame{
		StreamID: streamID,
		Type:     frameType,
		Payload:  payload,
	}, nil
}

// String returns a human-readable description of the frame type.
func (t FrameType) String() string {
	switch t {
	case FrameData:
		return "DATA"
	case FrameOpenStream:
		return "OPEN_STREAM"
	case FrameCloseStream:
		return "CLOSE_STREAM"
	case FrameWindowUpdate:
		return "WINDOW_UPDATE"
	case FramePing:
		return "PING"
	case FramePong:
		return "PONG"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", t)
	}
}

// String returns a human-readable description of the stream.
func (s StreamID) String() string {
	switch s {
	case StreamControl:
		return "control"
	case StreamSSH:
		return "ssh"
	case StreamShell:
		return "shell"
	case StreamRDP:
		return "rdp"
	case StreamDocker:
		return "docker"
	case StreamVNC:
		return "vnc"
	default:
		return fmt.Sprintf("stream-%d", s)
	}
}