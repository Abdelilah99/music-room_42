import 'dart:async';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:dio/dio.dart';
import 'package:music_room/core/api/api_client.dart';
import 'package:music_room/core/models/device.dart'; // Import the official Device model from dev

// Model for devices shared TO the user (Incoming)
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

class MyDevicesNotifier extends AsyncNotifier<List<Device>> {
  Dio get _dio => ref.read(apiClientProvider).dio;

  @override
  FutureOr<List<Device>> build() {
    return _fetchDevices();
  }

  Future<List<Device>> _fetchDevices() async {
    final response = await _dio.get('/api/v1/devices');
    final devicesData = response.data['devices'] as List? ?? [];
    return devicesData
        .map((item) => Device.fromJson(item as Map<String, dynamic>))
        .toList();
  }

  Future<void> refresh() async {
    state = const AsyncValue.loading();
    state = await AsyncValue.guard(() => _fetchDevices());
  }

  Future<void> grantDelegation(String deviceId, String friendId) async {
    try {
      await _dio.post(
        '/api/v1/devices/$deviceId/delegate',
        data: {'friend_user_id': friendId},
      );
      state = await AsyncValue.guard(() => _fetchDevices());
    } on DioException catch (e) {
      throw Exception(e.response?.data?['error'] ?? 'Could not delegate device control.');
    }
  }

  Future<void> revokeDelegation(String deviceId) async {
    try {
      await _dio.delete('/api/v1/devices/$deviceId/delegate');
      state = await AsyncValue.guard(() => _fetchDevices());
    } on DioException catch (e) {
      throw Exception(e.response?.data?['error'] ?? 'Could not revoke control.');
    }
  }
}

class DelegatedToMeNotifier extends AsyncNotifier<List<DelegatedDevice>> {
  Dio get _dio => ref.read(apiClientProvider).dio;

  @override
  FutureOr<List<DelegatedDevice>> build() {
    return _fetchDelegated();
  }

  Future<List<DelegatedDevice>> _fetchDelegated() async {
    final response = await _dio.get('/api/v1/devices/delegated');
    final devicesData = response.data['devices'] as List? ?? [];
    return devicesData
        .map((item) => DelegatedDevice.fromJson(item as Map<String, dynamic>))
        .toList();
  }

  Future<void> refresh() async {
    state = const AsyncValue.loading();
    state = await AsyncValue.guard(() => _fetchDelegated());
  }
}

final myDevicesProvider = AsyncNotifierProvider.autoDispose<MyDevicesNotifier, List<Device>>(MyDevicesNotifier.new);
final delegatedToMeProvider = AsyncNotifierProvider.autoDispose<DelegatedToMeNotifier, List<DelegatedDevice>>(DelegatedToMeNotifier.new);

// Real Friends fetch provider for the interactive picker sheet
final friendsProvider = FutureProvider.autoDispose<List<Map<String, dynamic>>>((ref) async {
  final dio = ref.read(apiClientProvider).dio;
  final response = await dio.get('/api/v1/friends');
  final list = response.data['friends'] as List? ?? response.data as List? ?? [];
  return list.map((item) => item as Map<String, dynamic>).toList();
});
