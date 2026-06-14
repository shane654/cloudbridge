/// Device model for CloudBridge.
library;

/// Represents a registered device on the CloudBridge server.
class Device {
  final String id;
  final String name;
  final String platform;
  final String version;
  final bool online;
  final DateTime? registered;
  final DateTime? lastSeen;

  Device({
    required this.id,
    required this.name,
    required this.platform,
    this.version = '',
    this.online = false,
    this.registered,
    this.lastSeen,
  });

  factory Device.fromJson(Map<String, dynamic> json) => Device(
        id: json['id'] as String,
        name: json['name'] as String,
        platform: json['platform'] as String,
        version: json['version'] as String? ?? '',
        online: json['online'] as bool? ?? false,
        registered: json['registered'] != null
            ? DateTime.tryParse(json['registered'] as String)
            : null,
        lastSeen: json['last_seen'] != null
            ? DateTime.tryParse(json['last_seen'] as String)
            : null,
      );

  /// Platform icon for UI
  String get platformIcon {
    switch (platform) {
      case 'linux':
        return '🐧';
      case 'windows':
        return '🪟';
      case 'darwin':
        return '🍎';
      default:
        return '💻';
    }
  }

  @override
  String toString() => 'Device(id: $id, name: $name, platform: $platform, online: $online)';
}