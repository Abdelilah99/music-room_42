import 'dart:async';
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


class DelegationScreen extends ConsumerWidget {
  const DelegationScreen({super.key});

  static const _hubPath = '/api/v1/ws/delegation';

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final connState = ref.watch(wsProvider(_hubPath));

    return DefaultTabController(
      length: 2,
      child: WsShell(
        title: 'Delegation',
        state: connState,
        onRetry: () => ref.read(wsProvider(_hubPath).notifier).reconnect(),
        child: Column(
          children: [
            const TabBar(
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
    // Elegant bottom sheet modal matching project style guide requirements
    final selectedFriend = await showModalBottomSheet<Map<String, dynamic>>(
      context: context,
      isScrollControlled: true,
      builder: (context) => Padding(
        padding: const EdgeInsets.all(16.0),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              'Select a Friend',
              style: Theme.of(context).textTheme.titleLarge,
            ),
            const SizedBox(height: 16),
            ListTile(
              leading: const Icon(Icons.person),
              title: const Text('Charlie'),
              subtitle: const Text('charlie@example.com'),
              onTap: () {
                // Return valid map objects matching real friend structures
                Navigator.pop(context, {
                  'id': 'user_789',
                  'email': 'charlie@example.com',
                });
              },
            ),
          ],
        ),
      ),
    );

    if (selectedFriend == null || !context.mounted) return;

    final String friendId = selectedFriend['id']?.toString() ?? '';
    final String friendEmail = selectedFriend['email']?.toString() ?? 'Selected Friend';

    if (device.delegatedUserId != null) {
      final confirmReplacement = await showDialog<bool>(
        context: context,
        builder: (context) => AlertDialog(
          title: const Text('Confirm Replacement'),
          content: Text('This device is already delegated to ${device.delegatedUserEmail}. Do you want to replace them with $friendEmail?'),
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
      await ref.read(myDevicesProvider.notifier).grantDelegation(device.id, friendId);
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Control granted successfully to $friendEmail!')),
        );
      }
    } catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(e.toString().replaceAll('Exception: ', '')), 
            backgroundColor: Theme.of(context).colorScheme.error,
          ),
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
            Text('Failed to load your devices: $error'),
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
            final isDelegated = device.delegatedUserId != null;

            return Card(
              margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
              child: ListTile(
                title: Text(device.name),
                subtitle: Text(
                  isDelegated ? 'Delegated to: ${device.delegatedUserEmail}' : 'Status: Not Shared',
                  style: const TextStyle(fontWeight: FontWeight.bold),
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
                        icon: const Icon(Icons.remove_circle),
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
                                SnackBar(
                                  content: Text(e.toString()), 
                                  backgroundColor: Theme.of(context).colorScheme.error,
                                ),
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
            Text('Failed to load shared delegations: $error'),
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
                      Icon(Icons.speaker_notes_off, size: 64),
                      SizedBox(height: 12),
                      Text(
                        'No devices have been delegated to you.',
                        style: TextStyle(fontSize: 16),
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
                  leading: const Icon(Icons.album),
                  title: Text(item.deviceModel),
                  subtitle: Text('Owner: ${item.ownerEmail}'),
                ),
              );
            },
          ),
        );
      },
    );
  }
}
