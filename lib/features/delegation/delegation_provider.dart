import 'dart:async';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:dio/dio.dart';
import 'package:music_room/core/api/api_client.dart';

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

final delegatedToMeProvider = AsyncNotifierProvider.autoDispose<DelegatedToMeNotifier, List<DelegatedDevice>>(DelegatedToMeNotifier.new);

final friendsProvider = FutureProvider.autoDispose<List<Map<String, dynamic>>>((ref) async {
  final dio = ref.read(apiClientProvider).dio;
  final response = await dio.get('/api/v1/friends');
  final list = response.data['friends'] as List? ?? response.data as List? ?? [];
  return list.map((item) => item as Map<String, dynamic>).toList();
});
