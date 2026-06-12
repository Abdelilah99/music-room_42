import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:music_room/core/api/api_client.dart';
import 'package:music_room/core/models/device.dart';

class DevicesApi {
  final ApiClient _client;

  DevicesApi(this._client);

  // GET /devices -> {"devices": [...]}
  Future<List<Device>> list() async {
    final res = await _client.dio.get('/api/v1/devices');
    final raw = res.data['devices'] as List<dynamic>? ?? [];
    return raw.map((e) => Device.fromJson(e as Map<String, dynamic>)).toList();
  }

  // POST /devices -> 201 Device (409 if this user already has a device with the
  // same model).
  Future<Device> register({
    required String name,
    required String platform,
    required String model,
  }) async {
    final res = await _client.dio.post('/api/v1/devices', data: {
      'name': name,
      'platform': platform,
      'model': model,
    });
    return Device.fromJson(res.data as Map<String, dynamic>);
  }

  // DELETE /devices/:id
  Future<void> delete(String id) async {
    await _client.dio.delete('/api/v1/devices/$id');
  }

  // POST /devices/:id/command -> relays a playback command to the device's
  // WebSocket hub. `value` is only sent for the "volume" action (0-100).
  Future<void> sendCommand(
    String id, {
    required String action,
    int? value,
  }) async {
    final data = <String, dynamic>{'action': action};
    if (value != null) data['value'] = value;
    await _client.dio.post('/api/v1/devices/$id/command', data: data);
  }

  Future<void> grantDelegation(String deviceId, String friendId) async {
    await _client.dio.post(
      '/api/v1/devices/$deviceId/delegate',
      data: {'friend_user_id': friendId},
    );
  }

  Future<void> revokeDelegation(String deviceId) async {
    await _client.dio.delete('/api/v1/devices/$deviceId/delegate');
  }
}

final devicesApiProvider = Provider<DevicesApi>(
  (ref) => DevicesApi(ref.watch(apiClientProvider)),
);
