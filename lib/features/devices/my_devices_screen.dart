import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:music_room/core/models/device.dart';
import 'package:music_room/features/devices/devices_provider.dart';
import 'package:music_room/shared/widgets/snackbar_helper.dart';
import 'package:music_room/shared/widgets/state_widgets.dart';

class MyDevicesScreen extends ConsumerStatefulWidget {
  const MyDevicesScreen({super.key});

  @override
  ConsumerState<MyDevicesScreen> createState() => _MyDevicesScreenState();
}

class _MyDevicesScreenState extends ConsumerState<MyDevicesScreen> {
  // Ids of devices with a delete request in flight, so the row's button is
  // disabled and a second tap cannot fire a duplicate DELETE.
  final Set<String> _deleting = {};

  @override
  Widget build(BuildContext context) {
    final devicesAsync = ref.watch(devicesProvider);

    return Scaffold(
      appBar: AppBar(title: const Text('My Devices')),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: _registerCurrent,
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
                    isDeleting: _deleting.contains(devices[i].id),
                    onDelete: () => _confirmDelete(devices[i]),
                  ),
                ),
        ),
      ),
    );
  }

  Future<void> _registerCurrent() async {
    final error = await ref.read(devicesProvider.notifier).registerCurrent();
    if (!mounted) return;
    if (error == null) {
      AppSnackBar.show(context,
          message: 'Device registered', type: SnackBarType.success);
    } else {
      AppSnackBar.show(context, message: error, type: SnackBarType.error);
    }
  }

  Future<void> _confirmDelete(Device device) async {
    // Ignore taps on a row that is already being deleted.
    if (_deleting.contains(device.id)) return;

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
    if (_deleting.contains(device.id)) return;

    setState(() => _deleting.add(device.id));
    final error = await ref.read(devicesProvider.notifier).remove(device.id);
    if (!mounted) return;
    setState(() => _deleting.remove(device.id));
    if (error != null) {
      AppSnackBar.show(context, message: error, type: SnackBarType.error);
    }
  }
}

class _DeviceCard extends StatelessWidget {
  const _DeviceCard({
    required this.device,
    required this.isDeleting,
    required this.onDelete,
  });

  final Device device;
  final bool isDeleting;
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
        trailing: isDeleting
            ? const SizedBox(
                width: 24,
                height: 24,
                child: CircularProgressIndicator(strokeWidth: 2),
              )
            : IconButton(
                icon: const Icon(Icons.delete_outline),
                tooltip: 'Remove device',
                onPressed: onDelete,
              ),
      ),
    );
  }
}
