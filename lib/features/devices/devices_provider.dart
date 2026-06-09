import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:music_room/core/api/devices_api.dart';
import 'package:music_room/core/models/device.dart';
import 'package:music_room/core/services/device_info_service.dart';

final devicesProvider =
    AsyncNotifierProvider<DevicesNotifier, List<Device>>(DevicesNotifier.new);

class DevicesNotifier extends AsyncNotifier<List<Device>> {
  DevicesApi get _api => ref.read(devicesApiProvider);

  // Auto-registration runs at most once per provider lifetime, on the first
  // load. Pull-to-refresh reuses the same notifier so it never fires again.
  bool _autoRegisterAttempted = false;

  @override
  Future<List<Device>> build() async {
    final devices = await _api.list();
    return _maybeAutoRegister(devices);
  }

  // If the current device's model is not already registered, register it once
  // and append it to the list. Best-effort: a failure (e.g. a 409 race) just
  // leaves the fetched list as-is; the manual button stays available.
  Future<List<Device>> _maybeAutoRegister(List<Device> devices) async {
    if (_autoRegisterAttempted) return devices;
    _autoRegisterAttempted = true;

    final model = DeviceInfoService.model;
    if (model.isEmpty) return devices;
    if (devices.any((d) => d.model == model)) return devices;

    try {
      final created = await _api.register(
        name: _currentDeviceName(),
        platform: 'android',
        model: model,
      );
      return [...devices, created];
    } catch (_) {
      return devices;
    }
  }

  // A readable name for the current device, falling back to manufacturer+model
  // when the system device name is unavailable.
  String _currentDeviceName() {
    final name = DeviceInfoService.deviceName;
    if (name.isNotEmpty) return name;
    final manufacturer = DeviceInfoService.manufacturer;
    final model = DeviceInfoService.model;
    final composed =
        [manufacturer, model].where((s) => s.isNotEmpty).join(' ').trim();
    return composed.isEmpty ? 'Android device' : composed;
  }

  Future<void> refresh() async {
    state = await AsyncValue.guard(() => _api.list());
  }

  // Manual "Register this device" action. Returns null on success, or a
  // human-readable message on failure.
  Future<String?> registerCurrent() async {
    final model = DeviceInfoService.model;
    if (model.isEmpty) {
      return 'Could not read this device model';
    }
    try {
      final created = await _api.register(
        name: _currentDeviceName(),
        platform: 'android',
        model: model,
      );
      state = AsyncData([...(state.value ?? []), created]);
      return null;
    } on DioException catch (e) {
      if (e.response?.statusCode == 409) {
        return 'This device is already registered';
      }
      return _errorMessage(e) ?? 'Could not register this device';
    }
  }

  // Deletes a device and removes it from the list on success. Returns null on
  // success, or a human-readable message on failure.
  Future<String?> remove(String id) async {
    try {
      await _api.delete(id);
      final current = state.value ?? [];
      state = AsyncData(current.where((d) => d.id != id).toList());
      return null;
    } on DioException catch (e) {
      return _errorMessage(e) ?? 'Could not delete the device';
    }
  }

  String? _errorMessage(DioException e) {
    final data = e.response?.data;
    return data is Map ? data['error'] as String? : null;
  }
}
