/// WebSocket signal client for CloudBridge.
/// Connects to the signal server, handles registration, heartbeats,
/// and message routing.
library;

import 'dart:async';

import 'package:web_socket_channel/web_socket_channel.dart';

import 'signal_messages.dart';

/// Callback for incoming signal messages
typedef SignalMessageHandler = void Function(SignalMessage message);

/// Connection state for the signal client
enum SignalConnectionState {
  disconnected,
  connecting,
  connected,
  registered,
  error,
}

/// SignalClient manages the WebSocket connection to the CloudBridge signal server.
class SignalClient {
  final String serverUrl;
  final bool isAgent;

  WebSocketChannel? _channel;
  Timer? _heartbeatTimer;
  Timer? _reconnectTimer;
  SignalConnectionState _state = SignalConnectionState.disconnected;
  String? _token;
  String? _deviceId;
  int _reconnectAttempts = 0;

  static const Duration _heartbeatInterval = Duration(seconds: 30);
  static const Duration _reconnectBaseDelay = Duration(seconds: 1);
  static const Duration _reconnectMaxDelay = Duration(seconds: 30);
  static const int _maxReconnectAttempts = 0; // unlimited

  final List<SignalMessageHandler> _messageHandlers = [];
  final List<void Function(SignalConnectionState)> _stateListeners = [];

  SignalClient({
    required this.serverUrl,
    this.isAgent = false,
  });

  SignalConnectionState get state => _state;
  String? get token => _token;
  String? get deviceId => _deviceId;

  void addMessageHandler(SignalMessageHandler handler) {
    _messageHandlers.add(handler);
  }

  void removeMessageHandler(SignalMessageHandler handler) {
    _messageHandlers.remove(handler);
  }

  void addStateListener(void Function(SignalConnectionState) listener) {
    _stateListeners.add(listener);
  }

  void removeStateListener(void Function(SignalConnectionState) listener) {
    _stateListeners.remove(listener);
  }

  void _setState(SignalConnectionState newState) {
    if (_state != newState) {
      _state = newState;
      for (final listener in _stateListeners) {
        listener(newState);
      }
    }
  }

  /// Connect to the signal server.
  Future<void> connect() async {
    if (_state == SignalConnectionState.connected ||
        _state == SignalConnectionState.registered) {
      return;
    }

    _setState(SignalConnectionState.connecting);

    try {
      final typeParam = isAgent ? 'type=agent' : 'type=app';
      final url = '$serverUrl?$typeParam';
      _channel = WebSocketChannel.connect(Uri.parse(url));

      _setState(SignalConnectionState.connected);
      _reconnectAttempts = 0;

      _channel!.stream.listen(
        _onMessage,
        onError: _onError,
        onDone: _onDone,
      );
    } catch (e) {
      _setState(SignalConnectionState.error);
      _scheduleReconnect();
    }
  }

  /// Register this device with the signal server.
  void register({
    required String deviceId,
    required String deviceName,
    required String platform,
    required String version,
    String publicKey = '',
  }) {
    _deviceId = deviceId;
    final payload = RegisterPayload(
      deviceId: deviceId,
      deviceName: deviceName,
      platform: platform,
      version: version,
      publicKey: publicKey,
    );

    _send(MsgType.register, payload.toJson());
  }

  /// Send a connect request to a remote device.
  void connectRequest({
    required String sessionId,
    required String deviceId,
    required List<String> protocols,
  }) {
    final payload = ConnectRequestPayload(
      sessionId: sessionId,
      deviceId: deviceId,
      protocols: protocols,
    );

    _send(MsgType.connectRequest, payload.toJson());
  }

  /// Send a connect response (accept/reject).
  void connectResponse({
    required String sessionId,
    required bool accepted,
    String? reason,
  }) {
    final payload = ConnectResponsePayload(
      sessionId: sessionId,
      accepted: accepted,
      reason: reason,
    );

    _send(MsgType.connectResponse, {
      'session_id': payload.sessionId,
      'accepted': payload.accepted,
      if (payload.reason != null) 'reason': payload.reason,
    });
  }

  /// Send a transport negotiate message.
  void transportNegotiate({
    required String sessionId,
    required String mode,
    String? relayEndpoint,
  }) {
    final payload = TransportNegotiatePayload(
      sessionId: sessionId,
      mode: mode,
      relayEndpoint: relayEndpoint,
    );

    _send(MsgType.transportNegotiate, payload.toJson());
  }

  /// Send a disconnect message.
  void disconnect({required String sessionId, String? reason}) {
    final payload = DisconnectPayload(sessionId: sessionId, reason: reason);
    _send(MsgType.disconnect, payload.toJson());
  }

  /// Send a raw message.
  void _send(String type, Map<String, dynamic>? data) {
    if (_channel == null) return;

    final msg = SignalMessage(type: type, data: data);
    _channel!.sink.add(msg.encode());
  }

  void _onMessage(dynamic message) {
    if (message is String) {
      try {
        final signalMsg = SignalMessage.decode(message);
        _handleSignalMessage(signalMsg);
      } catch (e) {
        // Ignore malformed messages
      }
    }
  }

  void _handleSignalMessage(SignalMessage message) {
    switch (message.type) {
      case MsgType.registerAck:
        _handleRegisterAck(message.data);
        break;
      case MsgType.heartbeatAck:
        // Heartbeat acknowledged, nothing special to do
        break;
      case MsgType.error:
        _handleError(message.data);
        break;
      default:
        // Forward to registered handlers
        for (final handler in _messageHandlers) {
          handler(message);
        }
        break;
    }
  }

  void _handleRegisterAck(Map<String, dynamic>? data) {
    if (data == null) return;
    final ack = RegisterAckPayload.fromJson(data);
    _token = ack.token;
    _setState(SignalConnectionState.registered);
  }

  void _handleError(Map<String, dynamic>? data) {
    if (data == null) return;
    // Error received from server - forward to handlers
    final _ = ErrorPayload.fromJson(data);
    // Forward error to handlers
    for (final handler in _messageHandlers) {
      handler(SignalMessage(type: MsgType.error, data: data));
    }
  }

  void _onError(dynamic error) {
    _setState(SignalConnectionState.error);
    _scheduleReconnect();
  }

  void _onDone() {
    _stopHeartbeat();
    _setState(SignalConnectionState.disconnected);
    _scheduleReconnect();
  }

  void _startHeartbeat() {
    _stopHeartbeat();
    _heartbeatTimer = Timer.periodic(_heartbeatInterval, (_) {
      final payload = HeartbeatPayload(
        timestamp: DateTime.now().millisecondsSinceEpoch,
      );
      _send(MsgType.heartbeat, payload.toJson());
    });
  }

  void _stopHeartbeat() {
    _heartbeatTimer?.cancel();
    _heartbeatTimer = null;
  }

  void _scheduleReconnect() {
    if (_reconnectAttempts >= _maxReconnectAttempts &&
        _maxReconnectAttempts > 0) {
      return;
    }

    final delay = Duration(
      milliseconds: (_reconnectBaseDelay.inMilliseconds *
              (1 << _reconnectAttempts))
          .clamp(0, _reconnectMaxDelay.inMilliseconds),
    );

    _reconnectTimer = Timer(delay, () {
      _reconnectAttempts++;
      connect();
    });
  }

  /// Disconnect from the signal server.
  void disconnectClient() {
    _stopHeartbeat();
    _reconnectTimer?.cancel();
    _channel?.sink.close();
    _channel = null;
    _setState(SignalConnectionState.disconnected);
  }

  /// Start heartbeat after successful registration.
  void startHeartbeat() {
    _startHeartbeat();
  }
}