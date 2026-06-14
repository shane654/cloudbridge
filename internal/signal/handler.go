package signal

import (
	"encoding/json"
	"log/slog"

	"github.com/cloudbridge/cloudbridge/internal/protocol"
)

// Handler processes incoming WebSocket messages from clients.
type Handler struct {
	hub            *Hub
	sessionManager *SessionManager
}

// NewHandler creates a new message handler.
func NewHandler(hub *Hub, sessionManager *SessionManager) *Handler {
	return &Handler{
		hub:            hub,
		sessionManager: sessionManager,
	}
}

// HandleMessage dispatches an incoming message to the appropriate handler.
func (h *Handler) HandleMessage(client *Client, raw []byte) error {
	var envelope protocol.SignalMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		slog.Error("invalid message format", "client_id", client.ID, "err", err)
		return h.sendError(client, "invalid_message", "invalid message format")
	}

	slog.Debug("received message", "client_id", client.ID, "type", envelope.Type)

	switch envelope.Type {
	case protocol.MsgTypeRegister:
		return h.handleRegister(client, envelope.Data)
	case protocol.MsgTypeHeartbeat:
		return h.handleHeartbeat(client, envelope.Data)
	case protocol.MsgTypeConnectRequest:
		return h.handleConnectRequest(client, envelope.Data)
	case protocol.MsgTypeConnectResponse:
		return h.handleConnectResponse(client, envelope.Data)
	case protocol.MsgTypeSDPOffer:
		return h.handleSDPOffer(client, envelope.Data)
	case protocol.MsgTypeSDPAnswer:
		return h.handleSDPAnswer(client, envelope.Data)
	case protocol.MsgTypeICECandidate:
		return h.handleICECandidate(client, envelope.Data)
	case protocol.MsgTypeTransportNegotiate:
		return h.handleTransportNegotiate(client, envelope.Data)
	case protocol.MsgTypeDisconnect:
		return h.handleDisconnect(client, envelope.Data)
	default:
		slog.Warn("unknown message type", "type", envelope.Type, "client_id", client.ID)
		return h.sendError(client, "unknown_type", "unknown message type: "+envelope.Type)
	}
}

func (h *Handler) handleRegister(client *Client, data json.RawMessage) error {
	var reg protocol.RegisterPayload
	if err := json.Unmarshal(data, &reg); err != nil {
		return h.sendError(client, "invalid_register", "invalid register payload")
	}

	slog.Info("device registering", "device_id", reg.DeviceID, "name", reg.DeviceName, "platform", reg.Platform)

	h.hub.RegisterDevice(client, &DeviceInfo{
		ID:        reg.DeviceID,
		Name:      reg.DeviceName,
		Platform:  string(reg.Platform),
		Version:   reg.Version,
		PublicKey: reg.PublicKey,
	})

	ack, err := protocol.Encode(protocol.MsgTypeRegisterAck, protocol.RegisterAckPayload{
		Token:     "token-" + reg.DeviceID, // TODO: generate real JWT token
		ExpiresAt: 0,                       // TODO: set proper expiration
	})
	if err != nil {
		return err
	}

	select {
	case client.Send <- ack:
		return nil
	default:
		slog.Warn("client send buffer full", "client_id", client.ID)
		return nil
	}
}

func (h *Handler) handleHeartbeat(client *Client, data json.RawMessage) error {
	var hb protocol.HeartbeatPayload
	if err := json.Unmarshal(data, &hb); err != nil {
		return h.sendError(client, "invalid_heartbeat", "invalid heartbeat payload")
	}

	h.hub.UpdateHeartbeat(client.DeviceID)

	ack, err := protocol.Encode(protocol.MsgTypeHeartbeatAck, protocol.HeartbeatAckPayload{
		ServerTime: protocol.CurrentTimestampMillis(),
	})
	if err != nil {
		return err
	}

	select {
	case client.Send <- ack:
		return nil
	default:
		slog.Warn("client send buffer full", "client_id", client.ID)
		return nil
	}
}

func (h *Handler) handleConnectRequest(client *Client, data json.RawMessage) error {
	var req protocol.ConnectRequestPayload
	if err := json.Unmarshal(data, &req); err != nil {
		return h.sendError(client, "invalid_connect_request", "invalid connect request payload")
	}

	slog.Info("connect request",
		"from", client.ID,
		"device_id", req.DeviceID,
		"session_id", req.SessionID,
		"protocols", req.Protocols,
	)

	// Create a session tracking the connection
	h.sessionManager.CreateSession(req.SessionID, client, req.DeviceID, req.Protocols)

	// Forward the request to the target device's agent
	dataBytes, err := protocol.Encode(protocol.MsgTypeConnectRequest, req)
	if err != nil {
		return err
	}

	if !h.hub.RouteToDevice(req.DeviceID, dataBytes) {
		// Device is offline, clean up session and notify requester
		h.sessionManager.RejectSession(req.SessionID)
		return h.sendError(client, "device_offline", "device is not online: "+req.DeviceID)
	}
	return nil
}

func (h *Handler) handleConnectResponse(client *Client, data json.RawMessage) error {
	var resp protocol.ConnectResponsePayload
	if err := json.Unmarshal(data, &resp); err != nil {
		return h.sendError(client, "invalid_connect_response", "invalid connect response payload")
	}

	slog.Info("connect response",
		"device_id", client.DeviceID,
		"session_id", resp.SessionID,
		"accepted", resp.Accepted,
	)

	if resp.Accepted {
		h.sessionManager.AcceptSession(resp.SessionID, client)
	} else {
		h.sessionManager.RejectSession(resp.SessionID)
	}

	// Route response back to the app that initiated the connection
	dataBytes, err := protocol.Encode(protocol.MsgTypeConnectResponse, resp)
	if err != nil {
		return err
	}

	if !h.sessionManager.RouteToPeer(h.hub, resp.SessionID, client, protocol.MsgTypeConnectResponse, dataBytes) {
		return h.sendError(client, "route_failed", "failed to route response to peer")
	}
	return nil
}

func (h *Handler) handleSDPOffer(client *Client, data json.RawMessage) error {
	var offer protocol.SDPOfferPayload
	if err := json.Unmarshal(data, &offer); err != nil {
		return h.sendError(client, "invalid_sdp_offer", "invalid SDP offer payload")
	}

	slog.Info("SDP offer relayed", "session_id", offer.SessionID)

	dataBytes, err := protocol.Encode(protocol.MsgTypeSDPOffer, offer)
	if err != nil {
		return err
	}

	h.sessionManager.RouteToPeer(h.hub, offer.SessionID, client, protocol.MsgTypeSDPOffer, dataBytes)
	return nil
}

func (h *Handler) handleSDPAnswer(client *Client, data json.RawMessage) error {
	var answer protocol.SDPAnswerPayload
	if err := json.Unmarshal(data, &answer); err != nil {
		return h.sendError(client, "invalid_sdp_answer", "invalid SDP answer payload")
	}

	slog.Info("SDP answer relayed", "session_id", answer.SessionID)

	dataBytes, err := protocol.Encode(protocol.MsgTypeSDPAnswer, answer)
	if err != nil {
		return err
	}

	h.sessionManager.RouteToPeer(h.hub, answer.SessionID, client, protocol.MsgTypeSDPAnswer, dataBytes)
	return nil
}

func (h *Handler) handleICECandidate(client *Client, data json.RawMessage) error {
	var ice protocol.ICECandidatePayload
	if err := json.Unmarshal(data, &ice); err != nil {
		return h.sendError(client, "invalid_ice_candidate", "invalid ICE candidate payload")
	}

	slog.Debug("ICE candidate relayed", "session_id", ice.SessionID)

	dataBytes, err := protocol.Encode(protocol.MsgTypeICECandidate, ice)
	if err != nil {
		return err
	}

	h.sessionManager.RouteToPeer(h.hub, ice.SessionID, client, protocol.MsgTypeICECandidate, dataBytes)
	return nil
}

func (h *Handler) handleTransportNegotiate(client *Client, data json.RawMessage) error {
	var tn protocol.TransportNegotiatePayload
	if err := json.Unmarshal(data, &tn); err != nil {
		return h.sendError(client, "invalid_transport_negotiate", "invalid transport negotiate payload")
	}

	slog.Info("transport negotiate", "session_id", tn.SessionID, "mode", tn.Mode)

	dataBytes, err := protocol.Encode(protocol.MsgTypeTransportNegotiate, tn)
	if err != nil {
		return err
	}

	h.sessionManager.RouteToPeer(h.hub, tn.SessionID, client, protocol.MsgTypeTransportNegotiate, dataBytes)
	return nil
}

func (h *Handler) handleDisconnect(client *Client, data json.RawMessage) error {
	var disc protocol.DisconnectPayload
	if err := json.Unmarshal(data, &disc); err != nil {
		return h.sendError(client, "invalid_disconnect", "invalid disconnect payload")
	}

	slog.Info("disconnect", "client_id", client.ID, "session_id", disc.SessionID)

	dataBytes, err := protocol.Encode(protocol.MsgTypeDisconnect, disc)
	if err != nil {
		return err
	}

	h.sessionManager.RouteToPeer(h.hub, disc.SessionID, client, protocol.MsgTypeDisconnect, dataBytes)
	h.sessionManager.CloseSession(disc.SessionID)
	return nil
}

func (h *Handler) sendError(client *Client, code string, message string) error {
	errMsg, err := protocol.Encode(protocol.MsgTypeError, protocol.ErrorPayload{
		Code:    code,
		Message: message,
	})
	if err != nil {
		slog.Error("failed to encode error message", "err", err)
		return err
	}

	select {
	case client.Send <- errMsg:
	default:
		slog.Warn("client send buffer full, error not delivered", "client_id", client.ID)
	}
	return nil
}