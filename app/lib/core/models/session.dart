/// Session model for CloudBridge.
library;

/// Represents an active session between the app and a remote device.
class Session {
  final String id;
  final String deviceId;
  final String status;
  final List<String> protocols;
  final DateTime? createdAt;

  Session({
    required this.id,
    required this.deviceId,
    required this.status,
    this.protocols = const [],
    this.createdAt,
  });

  factory Session.fromJson(Map<String, dynamic> json) => Session(
        id: json['id'] as String,
        deviceId: json['device_id'] as String,
        status: json['status'] as String,
        protocols: (json['protocols'] as String?)?.split(',') ?? [],
        createdAt: json['created_at'] != null
            ? DateTime.tryParse(json['created_at'] as String)
            : null,
      );
}