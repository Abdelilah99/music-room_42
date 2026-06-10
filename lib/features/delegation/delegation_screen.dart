import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:dio/dio.dart';
import 'package:music_room/core/api/web_socket_service.dart';
import 'package:music_room/core/widgets/ws_shell.dart';
import 'package:music_room/core/api/api_client.dart';


class Device {
  final String id;
  final String name;
  final String model;
  final String? delegatedToName;
  final String? delegatedToId;

  Device({
    required this.id,
    required this.name,
    required this.model,
    this.delegatedToName,
    this.delegatedToId,
  });

  factory Device.fromJson(Map<String, dynamic> json) {
    return Device(
      id: json['id']?.toString() ?? '',
      name: json['name']?.toString() ?? 'Unknown Device',
      model: json['model']?.toString() ?? 'Generic Model',
      delegatedToName: json['delegated_user_name']?.toString(), // Alignment with backend fields
      delegatedToId: json['delegated_friend_id']?.toString(),     // Alignment with backend fields
    );
  }
}

class DelegatedDevice {
  final String id;
  final String ownerName;
  final String deviceModel;

  DelegatedDevice({
    required this.id,
    required this.ownerName,
    required this.deviceModel,
  });

  factory DelegatedDevice.fromJson(Map<String, dynamic> json) {
    return DelegatedDevice(
      id: json['id']?.toString() ?? '',
      ownerName: json['owner_name']?.toString() ?? 'Unknown Owner',
      deviceModel: json['device_model']?.toString() ?? 'Generic Model',
    );
  }
}

class MyDevicesNotifier extends AutoDisposeAsyncNotifier<List<Device>> {
  Dio get _dio => ref.read(apiClientProvider).dio;

  @override
  Future<List<Device>> build() async {
    return _fetchDevices();
  }

  Future<List<Device>> _fetchDevices() async {
    try {
      final response = await _dio.get('/api/v1/devices');
      return (response.data as List)
          .map((item) => Device.fromJson(item as Map<String, dynamic>))
          .toList();
    } catch (err) {
      rethrow;
    }
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
      final errMsg = e.response?.data?['error'] ?? 'Could not delegate device control.';
      throw Exception(errMsg);
    }
  }

  Future<void> revokeDelegation(String deviceId) async {
    try {
      await _dio.delete('/api/v1/devices/$deviceId/delegate');
      state = await AsyncValue.guard(() => _fetchDevices());
    } on DioException catch (e) {
      final errMsg = e.response?.data?['error'] ?? 'Could not revoke control.';
      throw Exception(errMsg);
    }
  }
}

class DelegatedToMeNotifier extends AutoDisposeAsyncNotifier<List<DelegatedDevice>> {
  Dio get _dio => ref.read(apiClientProvider).dio;

  @override
  Future<List<DelegatedDevice>> build() async {
    return _fetchDelegated();
  }

  Future<List<DelegatedDevice>> _fetchDelegated() async {
    try {
      final response = await _dio.get('/api/v1/devices/delegated');
      return (response.data as List)
          .map((item) => DelegatedDevice.fromJson(item as Map<String, dynamic>))
          .toList();
    } catch (err) {
      rethrow;
    }
  }

  Future<void> refresh() async {
    state = const AsyncValue.loading();
    state = await AsyncValue.guard(() => _fetchDelegated());
  }
}

final myDevicesProvider = AsyncNotifierProvider.autoDispose<MyDevicesNotifier, List<Device>>(MyDevicesNotifier.new);
final delegatedToMeProvider = AsyncNotifierProvider.autoDispose<DelegatedToMeNotifier, List<DelegatedDevice>>(DelegatedToMeNotifier.new);


class DelegationScreen extends ConsumerWidget {
  const DelegationScreen({super.key});

  static const _hubPath = '/api/v1/ws/delegation';

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final connState = ref.watch(wsProvider(_hubPath));

    return DefaultTabController(
      length: 2,
      child: WsShell(
        title: 'Delegation', // Explicit parameter title integration preserved
        state: connState,
        onRetry: () => ref.read(wsProvider(_hubPath).notifier).reconnect(),
        child: Column(
          children: [
            const TabBar(
              labelColor: Colors.blue,
              unselectedLabelColor: Colors.grey,
              indicatorColor: Colors.blue,
              tabs: [
                Tab(icon: Icon(Icons.devices), text: 'My Devices'),
                Tab(icon: Icon(Icons.assignment_ind), text: 'Delegated to Me'),
              ],
            ),
            Expanded(
              child: TabBarView(
                children: [
                  const MyDevicesTab(),
                  const DelegatedToMeTab(),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}


class MyDevicesTab extends ConsumerWidget {
  const MyDevicesTab({super.key});

  void _openFriendPickerAndGrant(BuildContext context, WidgetRef ref, Device device) async {
    const String selectedFriendId = "user_789";
    const String selectedFriendName = "Charlie";

    if (!context.mounted) return;

    if (device.delegatedToId != null) {
      final confirmReplacement = await showDialog<bool>(
        context: context,
        builder: (context) => AlertDialog(
          title: const Text('Confirm Replacement'),
          content: Text('This device is already delegated to ${device.delegatedToName}. Do you want to replace them with $selectedFriendName?'),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: const Text('Cancel'),
            ),
            ElevatedButton(
              onPressed: () => Navigator.pop(context, true),
              child: const Text('Confirm'),
            ),
          ],
        ),
      );

      if (confirmReplacement != true) return;
    }

    try {
      await ref.read(myDevicesProvider.notifier).grantDelegation(device.id, selectedFriendId);
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Control granted successfully to $selectedFriendName!')),
        );
      }
    } catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(e.toString().replaceAll('Exception: ', '')), backgroundColor: Colors.red),
        );
      }
    }
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final myDevicesState = ref.watch(myDevicesProvider);

    return myDevicesState.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (error, stack) => Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Text('Failed to load your devices: $error', style: const TextStyle(color: Colors.red)),
            const SizedBox(height: 12),
            ElevatedButton(
              onPressed: () => ref.read(myDevicesProvider.notifier).refresh(),
              child: const Text('Retry'),
            ),
          ],
        ),
      ),
      data: (devices) => RefreshIndicator(
        onRefresh: () => ref.read(myDevicesProvider.notifier).refresh(),
        child: ListView.builder(
          physics: const AlwaysScrollableScrollPhysics(),
          itemCount: devices.length,
          itemBuilder: (context, index) {
            final device = devices[index];
            final isDelegated = device.delegatedToName != null;

            return Card(
              margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
              child: ListTile(
                title: Text(device.name),
                subtitle: Text(
                  isDelegated ? 'Delegated to: ${device.delegatedToName}' : 'Status: Not Shared',
                  style: TextStyle(
                    color: isDelegated ? Colors.green : Colors.grey,
                    fontWeight: FontWeight.bold,
                  ),
                ),
                trailing: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    ElevatedButton(
                      onPressed: () => _openFriendPickerAndGrant(context, ref, device),
                      child: Text(isDelegated ? 'Change' : 'Grant'),
                    ),
                    if (isDelegated) ...[
                      const SizedBox(width: 8),
                      IconButton(
                        icon: const Icon(Icons.remove_circle, color: Colors.red),
                        onPressed: () async {
                          try {
                            await ref.read(myDevicesProvider.notifier).revokeDelegation(device.id);
                            if (context.mounted) {
                              ScaffoldMessenger.of(context).showSnackBar(
                                const SnackBar(content: Text('Delegation revoked successfully.')),
                              );
                            }
                          } catch (e) {
                            if (context.mounted) {
                              ScaffoldMessenger.of(context).showSnackBar(
                                SnackBar(content: Text(e.toString()), backgroundColor: Colors.red),
                              );
                            }
                          }
                        },
                      ),
                    ]
                  ],
                ),
              ),
            );
          },
        ),
      ),
    );
  }
}

class DelegatedToMeTab extends ConsumerWidget {
  const DelegatedToMeTab({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final delegatedState = ref.watch(delegatedToMeProvider);

    return delegatedState.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (error, stack) => Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Text('Failed to load shared delegations: $error', style: const TextStyle(color: Colors.red)),
            const SizedBox(height: 12),
            ElevatedButton(
              onPressed: () => ref.read(delegatedToMeProvider.notifier).refresh(),
              child: const Text('Retry'),
            ),
          ],
        ),
      ),
      data: (items) {
        if (items.isEmpty) {
          return RefreshIndicator(
            onRefresh: () => ref.read(delegatedToMeProvider.notifier).refresh(),
            child: ListView(
              physics: const AlwaysScrollableScrollPhysics(),
              children: const [
                SizedBox(height: 140),
                Center(
                  child: Column(
                    children: [
                      Icon(Icons.speaker_notes_off, size: 64, color: Colors.grey),
                      SizedBox(height: 12),
                      Text(
                        'No devices have been delegated to you.',
                        style: TextStyle(color: Colors.grey, fontSize: 16),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          );
        }

        return RefreshIndicator(
          onRefresh: () => ref.read(delegatedToMeProvider.notifier).refresh(),
          child: ListView.builder(
            physics: const AlwaysScrollableScrollPhysics(),
            itemCount: items.length,
            itemBuilder: (context, index) {
              final item = items[index];
              return Card(
                margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                child: ListTile(
                  leading: const Icon(Icons.album, color: Colors.blue),
                  title: Text(item.deviceModel),
                  subtitle: Text('Owner: ${item.ownerName}'),
                ),
              );
            },
          ),
        );
      },
    );
  }
}
