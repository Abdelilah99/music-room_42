// Mirrors the server Device and ActiveDelegate (server/internal/model/device.go).
class Device {
  final String id;
  final String userId;
  final String name;
  final String platform;
  final String model;
  final DateTime? createdAt;
  final DeviceDelegate? delegate; // null when there is no active delegate

  const Device({
    required this.id,
    required this.userId,
    required this.name,
    required this.platform,
    required this.model,
    this.createdAt,
    this.delegate,
  });

  bool get isDelegated => delegate != null;

  factory Device.fromJson(Map<String, dynamic> json) {
    DateTime? toDate(dynamic v) => v is String ? DateTime.tryParse(v) : null;

    final rawDelegate = json['delegate'];
    return Device(
      id: json['id'] as String,
      userId: json['user_id'] as String,
      name: json['name'] as String,
      platform: json['platform'] as String,
      model: json['model'] as String,
      createdAt: toDate(json['created_at']),
      delegate: rawDelegate is Map<String, dynamic>
          ? DeviceDelegate.fromJson(rawDelegate)
          : null,
    );
  }
}

// The friend a device is currently delegated to. The server exposes only the
// delegate's id and email, so that is all the client can show.
class DeviceDelegate {
  final String userId;
  final String email;

  const DeviceDelegate({required this.userId, required this.email});

  factory DeviceDelegate.fromJson(Map<String, dynamic> json) => DeviceDelegate(
        userId: json['user_id'] as String,
        email: json['email'] as String,
      );
}
