/// Connection state machine for CloudBridge.
library;

/// Connection states following the P2P → Relay fallback pattern.
enum ConnectionState {
  /// No active connection
  idle,

  /// Establishing signal connection
  connecting,

  /// Probing for P2P connectivity (STUN/ICE)
  probing,

  /// P2P connection established (WebRTC DataChannel or QUIC)
  p2pConnected,

  /// Connecting to relay server
  relayConnecting,

  /// Relay connection established
  relayConnected,

  /// Connection ready for data transfer
  connected,

  /// Connection error
  error,

  /// Connection closed
  disconnected,
}

/// Extension for human-readable state names
extension ConnectionStateExtension on ConnectionState {
  String get displayName {
    switch (this) {
      case ConnectionState.idle:
        return 'Idle';
      case ConnectionState.connecting:
        return 'Connecting...';
      case ConnectionState.probing:
        return 'Probing P2P...';
      case ConnectionState.p2pConnected:
        return 'P2P Connected';
      case ConnectionState.relayConnecting:
        return 'Connecting Relay...';
      case ConnectionState.relayConnected:
        return 'Relay Connected';
      case ConnectionState.connected:
        return 'Connected';
      case ConnectionState.error:
        return 'Error';
      case ConnectionState.disconnected:
        return 'Disconnected';
    }
  }

  bool get isConnected =>
      this == ConnectionState.p2pConnected ||
      this == ConnectionState.relayConnected ||
      this == ConnectionState.connected;

  bool get isTransitioning =>
      this == ConnectionState.connecting ||
      this == ConnectionState.probing ||
      this == ConnectionState.relayConnecting;
}