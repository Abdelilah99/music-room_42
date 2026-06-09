import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:music_room/core/models/device.dart';
import 'package:music_room/features/devices/devices_provider.dart';
import 'package:music_room/shared/widgets/snackbar_helper.dart';
import 'package:music_room/shared/widgets/state_widgets.dart';

class MyDevicesScreen extends ConsumerWidget {
  const MyDevicesScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final devicesAsync = ref.watch(devicesProvider);

    return Scaffold(
      appBar: AppBar(title: const Text('My Devices')),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => _registerCurrent(context, ref),
        icon: const Icon(Icons.add),
        label: const Text('Register this device'),
      ),
      body: devicesAsync.when(
        loading: () => const AppLoadingWidget(),
        error: (_, _) => AppErrorWidget(
          message: 'Could not load your devices.',
          onRetry: () => ref.read(devicesProvider.notifier).refresh(),
        ),
        data: (devices) => RefreshIndicator(
          onRefresh: () => ref.read(devicesProvider.notifier).refresh(),
          child: devices.isEmpty
              ? ListView(
                  // AlwaysScrollable so pull-to-refresh works on the empty state.
                  physics: const AlwaysScrollableScrollPhysics(),
                  children: const [
                    SizedBox(height: 120),
                    AppEmptyStateWidget(
                      icon: Icons.devices_outlined,
                      message: 'No devices registered yet.\n'
                          'Register this device to get started.',
                    ),
                  ],
                )
              : ListView.separated(
                  physics: const AlwaysScrollableScrollPhysics(),
                  padding: const EdgeInsets.fromLTRB(12, 12, 12, 88),
                  itemCount: devices.length,
                  separatorBuilder: (_, _) => const SizedBox(height: 8),
                  itemBuilder: (_, i) => _DeviceCard(
                    device: devices[i],
                    onDelete: () => _confirmDelete(context, ref, devices[i]),
                  ),
                ),
        ),
      ),
    );
  }

  Future<void> _registerCurrent(BuildContext context, WidgetRef ref) async {
    final error = await ref.read(devicesProvider.notifier).registerCurrent();
    if (!context.mounted) return;
    if (error == null) {
      AppSnackBar.show(context,
          message: 'Device registered', type: SnackBarType.success);
    } else {
      AppSnackBar.show(context, message: error, type: SnackBarType.error);
    }
  }

  Future<void> _confirmDelete(
      BuildContext context, WidgetRef ref, Device device) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Remove device'),
        content: Text('Remove "${device.name}" from your account?'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('Remove'),
          ),
        ],
      ),
    );
    if (confirmed != true) return;

    final error = await ref.read(devicesProvider.notifier).remove(device.id);
    if (!context.mounted) return;
    if (error != null) {
      AppSnackBar.show(context, message: error, type: SnackBarType.error);
    }
  }
}

class _DeviceCard extends StatelessWidget {
  const _DeviceCard({required this.device, required this.onDelete});

  final Device device;
  final VoidCallback onDelete;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final delegateText = device.delegate == null
        ? 'No active delegate'
        : 'Delegated to ${device.delegate!.email}';

    return Card(
      margin: EdgeInsets.zero,
      child: ListTile(
        leading: const Icon(Icons.smartphone_outlined),
        title: Text(device.name),
        subtitle: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const SizedBox(height: 2),
            Text('${device.model} - ${device.platform}'),
            const SizedBox(height: 2),
            Row(
              children: [
                Icon(
                  device.isDelegated
                      ? Icons.swap_horiz
                      : Icons.person_off_outlined,
                  size: 16,
                  color: device.isDelegated
                      ? theme.colorScheme.primary
                      : theme.disabledColor,
                ),
                const SizedBox(width: 4),
                Expanded(
                  child: Text(
                    delegateText,
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: device.isDelegated
                          ? theme.colorScheme.primary
                          : theme.disabledColor,
                    ),
                  ),
                ),
              ],
            ),
          ],
        ),
        isThreeLine: true,
        trailing: IconButton(
          icon: const Icon(Icons.delete_outline),
          tooltip: 'Remove device',
          onPressed: onDelete,
        ),
      ),
    );
  }
}
