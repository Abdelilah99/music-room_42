import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart' show defaultTargetPlatform, TargetPlatform;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:music_room/core/api/devices_api.dart';
import 'package:music_room/core/models/device.dart';
import 'package:music_room/core/services/device_info_service.dart';
import 'package:music_room/features/delegation/delegation_provider.dart'; // Import extensions

final devicesProvider =
    AsyncNotifierProvider<DevicesNotifier, List<Device>>(DevicesNotifier.new);

class DevicesNotifier extends AsyncNotifier<List<Device>> {
  DevicesApi get _api => ref.read(devicesApiProvider);

  bool _autoRegisterAttempted = false;

  @override
  Future<List<Device>> build() async {
    final devices = await _api.list();
    return _maybeAutoRegister(devices);
  }

  Future<List<Device>> _maybeAutoRegister(List<Device> devices) async {
    if (_autoRegisterAttempted) return devices;
    _autoRegisterAttempted = true;

    final model = _currentModel();
    if (devices.any((d) => d.model == model)) return devices;

    try {
      final created = await _api.register(
        name: _currentDeviceName(),
        platform: _currentPlatform(),
        model: model,
      );
      return [...devices, created];
    } catch (_) {
      return devices;
    }
  }

  String _currentPlatform() =>
      defaultTargetPlatform == TargetPlatform.iOS ? 'ios' : 'android';

  String _currentModel() {
    final model = DeviceInfoService.model.trim();
    if (model.isNotEmpty) return model;
    final manufacturer = DeviceInfoService.manufacturer.trim();
    return manufacturer.isNotEmpty ? manufacturer : 'Unknown Android device';
  }

  String _currentDeviceName() {
    final name = DeviceInfoService.deviceName.trim();
    return name.isNotEmpty ? name : _currentModel();
  }

  Future<void> refresh() async {
    state = await AsyncValue.guard(() => _api.list());
  }

  Future<void> grantDelegation(String deviceId, String friendId) async {
    try {
      final clientDio = ref.read(apiClientProvider).dio;
      await clientDio.post('/api/v1/devices/$deviceId/delegate', data: {'friend_user_id': friendId});
      await refresh(); 
    } on DioException catch (e) {
      throw Exception(_errorMessage(e) ?? 'Could not delegate device control.');
    }
  }

  Future<void> revokeDelegation(String deviceId) async {
    try {
      final clientDio = ref.read(apiClientProvider).dio;
      await clientDio.delete('/api/v1/devices/$deviceId/delegate');
      await refresh(); 
    } on DioException catch (e) {
      throw Exception(_errorMessage(e) ?? 'Could not revoke control.');
    }
  }

  Future<String?> registerCurrent() async {
    final model = _currentModel();
    final existing = state.value ?? [];
    if (existing.any((d) => d.model == model)) {
      return 'This device is already registered';
    }
    try {
      final created = await _api.register(
        name: _currentDeviceName(),
        platform: _currentPlatform(),
        model: model,
      );
      state = AsyncData([...existing, created]);
      return null;
    } on DioException catch (e) {
      if (e.response?.statusCode == 409) {
        await refresh();
        return 'This device is already registered';
      }
      return _errorMessage(e) ?? 'Could not register this device';
    }
  }

  Future<String?> remove(String id) async {
    try {
      await _api.delete(id);
      _dropFromList(id);
      return null;
    } on DioException catch (e) {
      if (e.response?.statusCode == 404) {
        _dropFromList(id);
        return null;
      }
      return _errorMessage(e) ?? 'Could not delete the device';
    }
  }

  void _dropFromList(String id) {
    final current = state.value ?? [];
    state = AsyncData(current.where((d) => d.id != id).toList());
  }

  String? _errorMessage(DioException e) {
    final data = e.response?.data;
    return data is Map ? data['error'] as String? : null;
  }
}
