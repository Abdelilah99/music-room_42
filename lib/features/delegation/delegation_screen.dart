import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:music_room/core/api/web_socket_service.dart';
import 'package:music_room/core/widgets/ws_shell.dart';
import 'package:music_room/core/models/device.dart';
import 'package:music_room/features/devices/devices_provider.dart'; 
import 'package:music_room/features/delegation/delegation_provider.dart';

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
        child: Padding(
          padding: const EdgeInsets.all(8.0),
          child: Column(
            children: [
              const TabBar(
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
      ),
    );
  }
}

class MyDevicesTab extends ConsumerWidget {
  const MyDevicesTab({super.key});

  void _openFriendPickerAndGrant(BuildContext context, WidgetRef ref, Device device) async {
    final selectedFriend = await showModalBottomSheet<Map<String, dynamic>>(
      context: context,
      isScrollControlled: true,
      builder: (context) => Consumer(
        builder: (context, ref, child) {
          final friendsState = ref.watch(friendsProvider);
          return Container(
            height: MediaQuery.of(context).size.height * 0.5,
            padding: const EdgeInsets.all(16.0),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('Select a Friend', style: Theme.of(context).textTheme.titleLarge),
                const SizedBox(height: 16),
                Expanded(
                  child: friendsState.when(
                    loading: () => const Center(child: CircularProgressIndicator()),
                    error: (err, _) => Center(child: Text('Failed to load friends: $err')),
                    data: (friends) {
                      if (friends.isEmpty) {
                        return const Center(child: Text('No friends found.'));
                      }
                      return ListView.builder(
                        itemCount: friends.length,
                        itemBuilder: (context, index) {
                          final friend = friends[index];
                          final email = friend['email']?.toString() ?? 'No Email';
                          
                          // [Critical Fix] Safely extract custom username through nested public_info objects
                          final username = friend['public_info']?['display_name']?.toString()
                              ?? friend['email']?.toString()
                              ?? 'Unknown';
                          
                          return ListTile(
                            leading: const Icon(Icons.person),
                            title: Text(username),
                            subtitle: Text(email),
                            onTap: () => Navigator.pop(context, friend),
                          );
                        },
                      );
                    },
                  ),
                ),
              ],
            ),
          );
        },
      ),
    );

    if (selectedFriend == null || !context.mounted) return;

    final String friendId = selectedFriend['id']?.toString() ?? selectedFriend['user_id']?.toString() ?? '';
    final String friendEmail = selectedFriend['email']?.toString() ?? 'Selected Friend';

    if (device.isDelegated) {
      final confirmReplacement = await showDialog<bool>(
        context: context,
        builder: (context) => AlertDialog(
          title: const Text('Confirm Replacement'),
          content: Text('This device is already delegated to ${device.delegate?.email}. Replace with $friendEmail?'),
          actions: [
            TextButton(onPressed: () => Navigator.pop(context, false), child: const Text('Cancel')),
            ElevatedButton(onPressed: () => Navigator.pop(context, true), child: const Text('Confirm')),
          ],
        ),
      );
      if (confirmReplacement != true) return;
    }

    try {
      await ref.read(devicesProvider.notifier).grantDelegation(device.id, friendId);
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Control granted to $friendEmail!')));
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
    final myDevicesState = ref.watch(devicesProvider);

    return myDevicesState.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (error, _) => Center(child: Text('Error loading devices: $error')),
      data: (devices) => RefreshIndicator(
        onRefresh: () => ref.read(devicesProvider.notifier).refresh(),
        child: ListView.builder(
          physics: const AlwaysScrollableScrollPhysics(),
          itemCount: devices.length,
          itemBuilder: (context, index) {
            final device = devices[index];
            final isDelegated = device.isDelegated;

            return Card(
              margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
              child: ListTile(
                title: Text(device.name),
                subtitle: Text(
                  isDelegated ? 'Delegated to: ${device.delegate?.email}' : 'Status: Not Shared',
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
                          final confirmRevoke = await showDialog<bool>(
                            context: context,
                            builder: (context) => AlertDialog(
                              title: const Text('Remove Delegation'),
                              content: const Text('Remove delegation for this device?'),
                              actions: [
                                TextButton(onPressed: () => Navigator.pop(context, false), child: const Text('Cancel')),
                                ElevatedButton(onPressed: () => Navigator.pop(context, true), child: const Text('Remove')),
                              ],
                            ),
                          );

                          if (confirmRevoke != true || !context.mounted) return;

                          try {
                            await ref.read(devicesProvider.notifier).revokeDelegation(device.id);
                            if (context.mounted) {
                              ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('Delegation revoked.')));
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
      error: (error, _) => Center(child: Text('Error loading shared delegations: $error')),
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
                      Text('No devices have been delegated to you.', style: TextStyle(fontSize: 16)),
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
