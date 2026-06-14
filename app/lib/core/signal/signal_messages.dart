/// Signal message types and payloads, aligned with Go's internal/protocol/messages.go
library;

import 'dart:convert';

// Message type constants
class MsgType {
  static const String register = 'register';
  static const String registerAck = 'register_ack';
  static const String heartbeat = 'heartbeat';
  static const String heartbeatAck = 'heartbeat_ack';
  static const String connectRequest = 'connect_request';
  static const String connectResponse = 'connect_response';
  static const String disconnect = 'disconnect';
  static const String sdpOffer = 'sdp_offer';
  static const String sdpAnswer = 'sdp_answer';
  static const String iceCandidate = 'ice_candidate';
  static const String transportNegotiate = 'transport_negotiate';
  static const String transportReady = 'transport_ready';
  static const String error = 'error';
}

/// Transport mode constants
class TransportMode {
  static const String webrtc = 'webrtc';
  static const String quic = 'quic';
  static const String relay = 'relay';
}

/// Signal message envelope
class SignalMessage {
  final String type;
  final Map<String, dynamic>? data;

  SignalMessage({required this.type, this.data});

  Map<String, dynamic> toJson() => {
        'type': type,
        if (data != null) 'data': data,
      };

  String encode() => jsonEncode(toJson());

  factory SignalMessage.fromJson(Map<String, dynamic> json) => SignalMessage(
        type: json['type'] as String,
        data: json['data'] as Map<String, dynamic>?,
      );

  factory SignalMessage.decode(String raw) {
    final json = jsonDecode(raw) as Map<String, dynamic>;
    return SignalMessage.fromJson(json);
  }
}

// --- Payload classes ---

class RegisterPayload {
  final String deviceId;
  final String deviceName;
  final String platform;
  final String version;
  final String publicKey;

  RegisterPayload({
    required this.deviceId,
    required this.deviceName,
    required this.platform,
    required this.version,
    this.publicKey = '',
  });

  Map<String, dynamic> toJson() => {
        'device_id': deviceId,
        'device_name': deviceName,
        'platform': platform,
        'version': version,
        'public_key': publicKey,
      };
}

class RegisterAckPayload {
  final String token;
  final int expiresAt;

  RegisterAckPayload({required this.token, required this.expiresAt});

  factory RegisterAckPayload.fromJson(Map<String, dynamic> json) =>
      RegisterAckPayload(
        token: json['token'] as String,
        expiresAt: json['expires_at'] as int,
      );
}

class HeartbeatPayload {
  final int timestamp;

  HeartbeatPayload({required this.timestamp});

  Map<String, dynamic> toJson() => {'timestamp': timestamp};
}

class HeartbeatAckPayload {
  final int serverTime;

  HeartbeatAckPayload({required this.serverTime});

  factory HeartbeatAckPayload.fromJson(Map<String, dynamic> json) =>
      HeartbeatAckPayload(serverTime: json['server_time'] as int);
}

class ConnectRequestPayload {
  final String sessionId;
  final String deviceId;
  final List<String> protocols;

  ConnectRequestPayload({
    required this.sessionId,
    required this.deviceId,
    required this.protocols,
  });

  Map<String, dynamic> toJson() => {
        'session_id': sessionId,
        'device_id': deviceId,
        'protocols': protocols,
      };
}

class ConnectResponsePayload {
  final String sessionId;
  final bool accepted;
  final String? reason;

  ConnectResponsePayload({
    required this.sessionId,
    required this.accepted,
    this.reason,
  });

  factory ConnectResponsePayload.fromJson(Map<String, dynamic> json) =>
      ConnectResponsePayload(
        sessionId: json['session_id'] as String,
        accepted: json['accepted'] as bool,
        reason: json['reason'] as String?,
      );
}

class DisconnectPayload {
  final String sessionId;
  final String? reason;

  DisconnectPayload({required this.sessionId, this.reason});

  Map<String, dynamic> toJson() => {
        'session_id': sessionId,
        if (reason != null) 'reason': reason,
      };
}

class SdpOfferPayload {
  final String sessionId;
  final String sdp;

  SdpOfferPayload({required this.sessionId, required this.sdp});

  Map<String, dynamic> toJson() => {
        'session_id': sessionId,
        'sdp': sdp,
      };

  factory SdpOfferPayload.fromJson(Map<String, dynamic> json) =>
      SdpOfferPayload(
        sessionId: json['session_id'] as String,
        sdp: json['sdp'] as String,
      );
}

class SdpAnswerPayload {
  final String sessionId;
  final String sdp;

  SdpAnswerPayload({required this.sessionId, required this.sdp});

  Map<String, dynamic> toJson() => {
        'session_id': sessionId,
        'sdp': sdp,
      };

  factory SdpAnswerPayload.fromJson(Map<String, dynamic> json) =>
      SdpAnswerPayload(
        sessionId: json['session_id'] as String,
        sdp: json['sdp'] as String,
      );
}

class IceCandidatePayload {
  final String sessionId;
  final String candidate;

  IceCandidatePayload({required this.sessionId, required this.candidate});

  Map<String, dynamic> toJson() => {
        'session_id': sessionId,
        'candidate': candidate,
      };

  factory IceCandidatePayload.fromJson(Map<String, dynamic> json) =>
      IceCandidatePayload(
        sessionId: json['session_id'] as String,
        candidate: json['candidate'] as String,
      );
}

class TransportNegotiatePayload {
  final String sessionId;
  final String mode;
  final String? relayEndpoint;

  TransportNegotiatePayload({
    required this.sessionId,
    required this.mode,
    this.relayEndpoint,
  });

  Map<String, dynamic> toJson() => {
        'session_id': sessionId,
        'mode': mode,
        if (relayEndpoint != null) 'relay_endpoint': relayEndpoint,
      };

  factory TransportNegotiatePayload.fromJson(Map<String, dynamic> json) =>
      TransportNegotiatePayload(
        sessionId: json['session_id'] as String,
        mode: json['mode'] as String,
        relayEndpoint: json['relay_endpoint'] as String?,
      );
}

class TransportReadyPayload {
  final String sessionId;
  final String mode;

  TransportReadyPayload({required this.sessionId, required this.mode});

  factory TransportReadyPayload.fromJson(Map<String, dynamic> json) =>
      TransportReadyPayload(
        sessionId: json['session_id'] as String,
        mode: json['mode'] as String,
      );
}

class ErrorPayload {
  final String code;
  final String message;

  ErrorPayload({required this.code, required this.message});

  factory ErrorPayload.fromJson(Map<String, dynamic> json) => ErrorPayload(
        code: json['code'] as String,
        message: json['message'] as String,
      );
}