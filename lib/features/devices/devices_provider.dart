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

    final model = _currentModel();
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

  // A stable, non-blank identifier for the current device. The server keys
  // device uniqueness on `model`, so this must resolve to the same value on
  // every launch, otherwise the same physical device can be registered more
  // than once. Trimmed so a whitespace-only platform value never slips through,
  // and falls back to a constant when the model is unavailable (e.g. on some
  // emulators) so we never send a blank model.
  String _currentModel() {
    final model = DeviceInfoService.model.trim();
    if (model.isNotEmpty) return model;
    final manufacturer = DeviceInfoService.manufacturer.trim();
    return manufacturer.isNotEmpty ? manufacturer : 'Unknown Android device';
  }

  // A readable name for the current device, falling back to the model when the
  // system device name is unavailable.
  String _currentDeviceName() {
    final name = DeviceInfoService.deviceName.trim();
    return name.isNotEmpty ? name : _currentModel();
  }

  Future<void> refresh() async {
    state = await AsyncValue.guard(() => _api.list());
  }

  // Manual "Register this device" action. Returns null on success, or a
  // human-readable message on failure.
  Future<String?> registerCurrent() async {
    final model = _currentModel();
    final existing = state.value ?? [];
    // Don't create a second row for a device that is already in the list.
    if (existing.any((d) => d.model == model)) {
      return 'This device is already registered';
    }
    try {
      final created = await _api.register(
        name: _currentDeviceName(),
        platform: 'android',
        model: model,
      );
      state = AsyncData([...existing, created]);
      return null;
    } on DioException catch (e) {
      if (e.response?.statusCode == 409) {
        // Registered on the server but missing from our list: reload so it shows.
        await refresh();
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
