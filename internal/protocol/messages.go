package protocol

import (
	"encoding/json"
	"fmt"
	"time"
)

// Signal message types for the WebSocket control channel.
const (
	// Device management
	MsgTypeRegister       = "register"
	MsgTypeRegisterAck    = "register_ack"
	MsgTypeHeartbeat      = "heartbeat"
	MsgTypeHeartbeatAck   = "heartbeat_ack"

	// Connection lifecycle
	MsgTypeConnectRequest  = "connect_request"
	MsgTypeConnectResponse = "connect_response"
	MsgTypeDisconnect      = "disconnect"

	// WebRTC signaling
	MsgTypeSDPOffer      = "sdp_offer"
	MsgTypeSDPAnswer     = "sdp_answer"
	MsgTypeICECandidate  = "ice_candidate"

	// Transport negotiation
	MsgTypeTransportNegotiate = "transport_negotiate"
	MsgTypeTransportReady     = "transport_ready"

	// Errors
	MsgTypeError = "error"
)

// TransportMode defines the tunnel transport type.
type TransportMode string

const (
	TransportWebRTC TransportMode = "webrtc"
	TransportQUIC   TransportMode = "quic"
	TransportRelay  TransportMode = "relay"
)

// Platform identifies the remote device OS.
type Platform string

const (
	PlatformLinux   Platform = "linux"
	PlatformWindows Platform = "windows"
	PlatformDarwin  Platform = "darwin"
)

// SignalMessage is the envelope for all WebSocket messages.
// Each message carries a `type` field for dispatch.
type SignalMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// Encode wraps a payload into a SignalMessage and serializes it.
func Encode(msgType string, payload any) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	envelope := SignalMessage{
		Type: msgType,
		Data: data,
	}
	return json.Marshal(envelope)
}

// Decode parses a SignalMessage envelope and unmarshals the Data field.
func Decode(raw []byte, payload any) (string, error) {
	var envelope SignalMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return "", fmt.Errorf("unmarshal envelope: %w", err)
	}
	if payload != nil && envelope.Data != nil {
		if err := json.Unmarshal(envelope.Data, payload); err != nil {
			return "", fmt.Errorf("unmarshal payload: %w", err)
		}
	}
	return envelope.Type, nil
}

// --- Device management messages ---

// RegisterPayload is sent by an agent to register with the signal server.
type RegisterPayload struct {
	DeviceID   string   `json:"device_id"`
	DeviceName string   `json:"device_name"`
	Platform   Platform `json:"platform"`
	Version    string   `json:"version"`
	PublicKey  string   `json:"public_key"` // ed25519 public key (base64)
}

// RegisterAckPayload is the server's response to a registration.
type RegisterAckPayload struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"` // unix timestamp
}

// HeartbeatPayload is sent periodically by the agent.
type HeartbeatPayload struct {
	Timestamp int64 `json:"timestamp"` // unix milliseconds
}

// HeartbeatAckPayload is the server's heartbeat acknowledgment.
type HeartbeatAckPayload struct {
	ServerTime int64 `json:"server_time"` // unix milliseconds
}

// --- Connection lifecycle messages ---

// ConnectRequestPayload is sent by the app to request a connection to a device.
type ConnectRequestPayload struct {
	SessionID string   `json:"session_id"`
	DeviceID  string   `json:"device_id"`
	Protocols []string `json:"protocols"` // e.g., ["ssh", "shell"]
}

// ConnectResponsePayload is the agent's response to a connection request.
type ConnectResponsePayload struct {
	SessionID string `json:"session_id"`
	Accepted  bool   `json:"accepted"`
	Reason    string `json:"reason,omitempty"` // if rejected
}

// DisconnectPayload closes a session.
type DisconnectPayload struct {
	SessionID string `json:"session_id"`
	Reason    string `json:"reason,omitempty"`
}

// --- WebRTC signaling messages ---

// SDPOfferPayload carries a WebRTC SDP offer.
type SDPOfferPayload struct {
	SessionID string `json:"session_id"`
	SDP       string `json:"sdp"`
}

// SDPAnswerPayload carries a WebRTC SDP answer.
type SDPAnswerPayload struct {
	SessionID string `json:"session_id"`
	SDP       string `json:"sdp"`
}

// ICECandidatePayload carries an ICE candidate.
type ICECandidatePayload struct {
	SessionID string `json:"session_id"`
	Candidate string `json:"candidate"`
}

// --- Transport negotiation messages ---

// TransportNegotiatePayload proposes or confirms a transport mode.
type TransportNegotiatePayload struct {
	SessionID     string       `json:"session_id"`
	Mode          TransportMode `json:"mode"`
	RelayEndpoint string       `json:"relay_endpoint,omitempty"` // populated when mode=relay
}

// TransportReadyPayload confirms the transport is established.
type TransportReadyPayload struct {
	SessionID string       `json:"session_id"`
	Mode      TransportMode `json:"mode"`
}

// --- Error message ---

// ErrorPayload carries an error description.
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// --- Helper: CurrentTimestampMillis returns the current time in milliseconds. ---

func CurrentTimestampMillis() int64 {
	return time.Now().UnixMilli()
}