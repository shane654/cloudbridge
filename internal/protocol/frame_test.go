package protocol

import (
	"bytes"
	"testing"
)

func TestFrameMarshalUnmarshal(t *testing.T) {
	tests := []struct {
		name     string
		frame    Frame
		wantErr  bool
	}{
		{
			name:  "data frame on control stream",
			frame: Frame{StreamID: StreamControl, Type: FrameData, Payload: []byte("hello")},
		},
		{
			name:  "open stream frame",
			frame: Frame{StreamID: StreamSSH, Type: FrameOpenStream, Payload: nil},
		},
		{
			name:  "close stream frame",
			frame: Frame{StreamID: StreamShell, Type: FrameCloseStream, Payload: nil},
		},
		{
			name:  "window update frame",
			frame: Frame{StreamID: StreamSSH, Type: FrameWindowUpdate, Payload: []byte{0, 0, 0, 4}},
		},
		{
			name:  "ping frame",
			frame: Frame{StreamID: StreamControl, Type: FramePing, Payload: []byte{1, 2, 3, 4}},
		},
		{
			name:  "empty payload",
			frame: Frame{StreamID: StreamControl, Type: FrameData, Payload: []byte{}},
		},
		{
			name:  "large payload near limit",
			frame: Frame{StreamID: StreamDocker, Type: FrameData, Payload: make([]byte, MaxPayloadSize-1)},
		},
		{
			name:    "payload too large",
			frame:   Frame{StreamID: StreamControl, Type: FrameData, Payload: make([]byte, MaxPayloadSize+1)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.frame.MarshalBinary()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("MarshalBinary: %v", err)
			}

			got := &Frame{}
			if err := got.UnmarshalBinary(data); err != nil {
				t.Fatalf("UnmarshalBinary: %v", err)
			}

			if got.StreamID != tt.frame.StreamID {
				t.Errorf("StreamID: got %v, want %v", got.StreamID, tt.frame.StreamID)
			}
			if got.Type != tt.frame.Type {
				t.Errorf("Type: got %v, want %v", got.Type, tt.frame.Type)
			}
			if !bytes.Equal(got.Payload, tt.frame.Payload) {
				t.Errorf("Payload: got %v, want %v", got.Payload, tt.frame.Payload)
			}
		})
	}
}

func TestReadFromWriteTo(t *testing.T) {
	original := &Frame{
		StreamID: StreamSSH,
		Type:     FrameData,
		Payload:  []byte("ssh payload data"),
	}

	var buf bytes.Buffer
	if _, err := original.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	got, err := ReadFrom(&buf)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if got.StreamID != original.StreamID {
		t.Errorf("StreamID: got %v, want %v", got.StreamID, original.StreamID)
	}
	if got.Type != original.Type {
		t.Errorf("Type: got %v, want %v", got.Type, original.Type)
	}
	if !bytes.Equal(got.Payload, original.Payload) {
		t.Errorf("Payload: got %v, want %v", got.Payload, original.Payload)
	}
}

func TestFrameTypeString(t *testing.T) {
	tests := []struct {
		ft   FrameType
		want string
	}{
		{FrameData, "DATA"},
		{FrameOpenStream, "OPEN_STREAM"},
		{FrameCloseStream, "CLOSE_STREAM"},
		{FrameWindowUpdate, "WINDOW_UPDATE"},
		{FramePing, "PING"},
		{FramePong, "PONG"},
		{FrameType(0xFF), "UNKNOWN(255)"},
	}
	for _, tt := range tests {
		if got := tt.ft.String(); got != tt.want {
			t.Errorf("FrameType(%d).String() = %q, want %q", tt.ft, got, tt.want)
		}
	}
}

func TestStreamIDString(t *testing.T) {
	tests := []struct {
		sid  StreamID
		want string
	}{
		{StreamControl, "control"},
		{StreamSSH, "ssh"},
		{StreamShell, "shell"},
		{StreamRDP, "rdp"},
		{StreamDocker, "docker"},
		{StreamVNC, "vnc"},
		{StreamID(99), "stream-99"},
	}
	for _, tt := range tests {
		if got := tt.sid.String(); got != tt.want {
			t.Errorf("StreamID(%d).String() = %q, want %q", tt.sid, got, tt.want)
		}
	}
}

func TestReadFromTruncated(t *testing.T) {
	// Write a valid frame, then truncate the payload
	f := &Frame{StreamID: StreamSSH, Type: FrameData, Payload: []byte("hello world")}
	data, err := f.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	// Truncate by 1 byte
	truncated := data[:len(data)-1]
	_, err = ReadFrom(bytes.NewReader(truncated))
	if err == nil {
		t.Error("expected error for truncated frame, got nil")
	}
}