/// Connection manager orchestrates the full connection lifecycle:
/// Signal registration → Connect request → Transport negotiation → Data channel.
library;

import 'dart:async';

import 'package:uuid/uuid.dart';

import '../protocol/frame.dart' show StreamID;
import '../signal/signal_client.dart';
import '../signal/signal_messages.dart';
import 'connection_state.dart';
import 'relay_transport.dart';

/// Manages the connection lifecycle between the app and a remote device.
class ConnectionManager {
  final SignalClient signalClient;

  ConnectionState _state = ConnectionState.idle;
  String? _activeSessionId;
  RelayTransport? _relayTransport;
  String? _relayAddress;
  String? _lastError;

  final List<void Function(ConnectionState)> _stateListeners = [];
  final List<void Function(String, bool)> _connectionResultListeners = [];

  ConnectionManager({required this.signalClient}) {
    signalClient.addMessageHandler(_handleSignalMessage);
    signalClient.addStateListener(_handleSignalState);
  }

  ConnectionState get state => _state;
  String? get activeSessionId => _activeSessionId;
  RelayTransport? get relayTransport => _relayTransport;
  String? get lastError => _lastError;

  void addStateListener(void Function(ConnectionState) listener) {
    _stateListeners.add(listener);
  }

  void addConnectionResultListener(void Function(String, bool) listener) {
    _connectionResultListeners.add(listener);
  }

  void _setState(ConnectionState newState) {
    if (_state != newState) {
      _state = newState;
      for (final listener in _stateListeners) {
        listener(newState);
      }
    }
  }

  /// Connect to a remote device.
  void connectToDevice(String deviceId, List<String> protocols) {
    _lastError = null;
    _activeSessionId = 'ses-${const Uuid().v4().substring(0, 8)}';
    _setState(ConnectionState.connecting);

    signalClient.connectRequest(
      sessionId: _activeSessionId!,
      deviceId: deviceId,
      protocols: protocols,
    );
  }

  /// Disconnect the active session.
  void disconnect() {
    if (_activeSessionId != null) {
      signalClient.disconnect(sessionId: _activeSessionId!);
    }
    _relayTransport?.disconnect();
    _relayTransport = null;
    _activeSessionId = null;
    _lastError = null;
    _setState(ConnectionState.disconnected);
  }

  void _handleSignalMessage(SignalMessage message) {
    switch (message.type) {
      case MsgType.connectResponse:
        _handleConnectResponse(message.data);
        break;
      case MsgType.transportNegotiate:
        _handleTransportNegotiate(message.data);
        break;
      case MsgType.transportReady:
        _handleTransportReady(message.data);
        break;
      case MsgType.disconnect:
        _handleDisconnect(message.data);
        break;
      case MsgType.error:
        _handleError(message.data);
        break;
    }
  }

  void _handleSignalState(SignalConnectionState signalState) {
    if (signalState == SignalConnectionState.disconnected ||
        signalState == SignalConnectionState.error) {
      _lastError = 'Signal connection lost';
      _setState(ConnectionState.error);
    }
  }

  void _handleConnectResponse(Map<String, dynamic>? data) {
    if (data == null) return;
    final response = ConnectResponsePayload.fromJson(data);

    if (response.accepted) {
      _setState(ConnectionState.probing);

      // For MVP, go directly to relay
      signalClient.transportNegotiate(
        sessionId: response.sessionId,
        mode: TransportMode.relay,
        relayEndpoint: _relayAddress ?? 'localhost:10988',
      );
    } else {
      _lastError = response.reason ?? 'Connection rejected';
      _setState(ConnectionState.error);
      for (final listener in _connectionResultListeners) {
        listener(response.sessionId, false);
      }
    }
  }

  void _handleTransportNegotiate(Map<String, dynamic>? data) {
    if (data == null) return;
    final negotiate = TransportNegotiatePayload.fromJson(data);

    if (negotiate.mode == TransportMode.relay &&
        negotiate.relayEndpoint != null) {
      _relayAddress = negotiate.relayEndpoint;
      _connectToRelay();
    }
  }

  void _handleTransportReady(Map<String, dynamic>? data) {
    if (data == null) return;
    _setState(ConnectionState.connected);
  }

  void _handleDisconnect(Map<String, dynamic>? data) {
    _relayTransport?.disconnect();
    _relayTransport = null;
    _activeSessionId = null;
    _setState(ConnectionState.disconnected);
  }

  void _handleError(Map<String, dynamic>? data) {
    if (data == null) return;
    final error = ErrorPayload.fromJson(data);
    _lastError = error.message;
    _setState(ConnectionState.error);
  }

  Future<void> _connectToRelay() async {
    if (_activeSessionId == null || _relayAddress == null) return;

    _setState(ConnectionState.relayConnecting);

    _relayTransport = RelayTransport(
      address: _relayAddress!,
      sessionId: _activeSessionId!,
      isInitiator: true,
    );

    try {
      await _relayTransport!.connect();
      _setState(ConnectionState.relayConnected);

      // Open shell stream
      _relayTransport!.openStream(StreamID.shell);

      _setState(ConnectionState.connected);
    } catch (e) {
      _lastError = 'Relay connection failed: $e';
      _setState(ConnectionState.error);
    }
  }

  /// Set the relay server address.
  void setRelayAddress(String address) {
    _relayAddress = address;
  }
}