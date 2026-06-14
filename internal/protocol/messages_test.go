package protocol

import (
	"encoding/json"
	"testing"
)

func TestEncodeDecodeRegister(t *testing.T) {
	reg := RegisterPayload{
		DeviceID:   "device-abc123",
		DeviceName: "My-PC",
		Platform:   PlatformLinux,
		Version:    "0.1.0",
		PublicKey:  "dGVzdHB1YmxpY2tleQ==",
	}

	data, err := Encode(MsgTypeRegister, reg)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	var got RegisterPayload
	msgType, err := Decode(data, &got)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if msgType != MsgTypeRegister {
		t.Errorf("msgType: got %q, want %q", msgType, MsgTypeRegister)
	}
	if got.DeviceID != reg.DeviceID {
		t.Errorf("DeviceID: got %q, want %q", got.DeviceID, reg.DeviceID)
	}
	if got.DeviceName != reg.DeviceName {
		t.Errorf("DeviceName: got %q, want %q", got.DeviceName, reg.DeviceName)
	}
	if got.Platform != reg.Platform {
		t.Errorf("Platform: got %q, want %q", got.Platform, reg.Platform)
	}
}

func TestEncodeDecodeConnectRequest(t *testing.T) {
	req := ConnectRequestPayload{
		SessionID: "sess-001",
		DeviceID:  "device-abc123",
		Protocols: []string{"ssh", "shell"},
	}

	data, err := Encode(MsgTypeConnectRequest, req)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	var got ConnectRequestPayload
	msgType, err := Decode(data, &got)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if msgType != MsgTypeConnectRequest {
		t.Errorf("msgType: got %q, want %q", msgType, MsgTypeConnectRequest)
	}
	if got.SessionID != req.SessionID {
		t.Errorf("SessionID: got %q, want %q", got.SessionID, req.SessionID)
	}
	if len(got.Protocols) != 2 {
		t.Errorf("Protocols: got %d, want 2", len(got.Protocols))
	}
}

func TestEncodeDecodeTransportNegotiate(t *testing.T) {
	tn := TransportNegotiatePayload{
		SessionID:     "sess-001",
		Mode:          TransportRelay,
		RelayEndpoint: "relay.cloudbridge.io:10988",
	}

	data, err := Encode(MsgTypeTransportNegotiate, tn)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	var got TransportNegotiatePayload
	msgType, err := Decode(data, &got)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if msgType != MsgTypeTransportNegotiate {
		t.Errorf("msgType: got %q, want %q", msgType, MsgTypeTransportNegotiate)
	}
	if got.Mode != TransportRelay {
		t.Errorf("Mode: got %q, want %q", got.Mode, TransportRelay)
	}
	if got.RelayEndpoint != tn.RelayEndpoint {
		t.Errorf("RelayEndpoint: got %q, want %q", got.RelayEndpoint, tn.RelayEndpoint)
	}
}

func TestDecodeWithNilPayload(t *testing.T) {
	// Message with no data (e.g., heartbeat_ack with embedded timestamp)
	hb := HeartbeatPayload{
		Timestamp: 1234567890,
	}

	data, err := Encode(MsgTypeHeartbeat, hb)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	msgType, err := Decode(data, nil)
	if err != nil {
		t.Fatalf("Decode with nil payload: %v", err)
	}
	if msgType != MsgTypeHeartbeat {
		t.Errorf("msgType: got %q, want %q", msgType, MsgTypeHeartbeat)
	}
}

func TestDecodeInvalidJSON(t *testing.T) {
	_, err := Decode([]byte("not json"), nil)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestEnvelopeStructure(t *testing.T) {
	reg := RegisterPayload{
		DeviceID:   "test-id",
		DeviceName: "test-device",
		Platform:   PlatformWindows,
	}

	data, err := Encode(MsgTypeRegister, reg)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	// Verify envelope structure
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}

	if string(envelope["type"]) != `"register"` {
		t.Errorf("type field: got %s, want \"register\"", string(envelope["type"]))
	}

	if _, ok := envelope["data"]; !ok {
		t.Error("data field missing from envelope")
	}
}