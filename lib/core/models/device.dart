class Device {
  final String id;
  final String name;
  final String model;
  final String? delegatedUserId;
  final String? delegatedUserEmail;

  Device({
    required this.id,
    required this.name,
    required this.model,
    this.delegatedUserId,
    this.delegatedUserEmail,
  });

  factory Device.fromJson(Map<String, dynamic> json) {
    final delegateJson = json['delegate'] as Map<String, dynamic>?;
    return Device(
      id: json['id']?.toString() ?? '',
      name: json['name']?.toString() ?? 'Unknown Device',
      model: json['model']?.toString() ?? 'Generic Model',
      delegatedUserId: delegateJson?['user_id']?.toString(),
      delegatedUserEmail: delegateJson?['email']?.toString(),
    );
  }
}

class DelegatedDevice {
  final String id;
  final String ownerEmail;
  final String deviceModel;

  DelegatedDevice({
    required this.id,
    required this.ownerEmail,
    required this.deviceModel,
  });

  factory DelegatedDevice.fromJson(Map<String, dynamic> json) {
    final ownerJson = json['owner'] as Map<String, dynamic>?;
    return DelegatedDevice(
      id: json['id']?.toString() ?? '',
      ownerEmail: ownerJson?['email']?.toString() ?? 'Unknown Owner',
      deviceModel: json['model']?.toString() ?? 'Generic Model',
    );
  }
}
