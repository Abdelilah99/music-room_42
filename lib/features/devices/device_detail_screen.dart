import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:music_room/core/api/web_socket_service.dart';
import 'package:music_room/core/models/device.dart';
import 'package:music_room/features/devices/device_control_provider.dart';
import 'package:music_room/shared/widgets/snackbar_helper.dart';

// Playback control surface for a single device.
//
// The owner connects to the device WebSocket and sees incoming commands
// reflected live. A delegate (reached from the delegated-devices list once
// that lands) sends commands too but does not open the socket - the server
// only lets the owner subscribe.
class DeviceDetailScreen extends ConsumerStatefulWidget {
  const DeviceDetailScreen({
    super.key,
    required this.deviceId,
    this.device,
    this.isOwner = true,
    this.ownerEmail,
  });

  final String deviceId;

  // The device, passed from the list. May be null on a cold deep link.
  final Device? device;

  // Whether the current user owns this device. Owners get the live socket;
  // delegates only send commands.
  final bool isOwner;

  // For the delegate view: the owner whose device is being controlled.
  final String? ownerEmail;

  @override
  ConsumerState<DeviceDetailScreen> createState() => _DeviceDetailScreenState();
}

class _DeviceDetailScreenState extends ConsumerState<DeviceDetailScreen> {
  String get _wsPath => '/api/v1/devices/${widget.deviceId}/ws';

  @override
  void initState() {
    super.initState();
    // Only the owner subscribes to the device socket. Done after the first
    // frame so the provider is alive (it is watched in build for its state).
    if (widget.isOwner) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) return;
        ref
            .read(wsProvider(_wsPath).notifier)
            .messageStream
            .listen(_onWsMessage);
      });
    }
  }

  void _onWsMessage(dynamic raw) {
    if (raw is! String) return;
    try {
      final data = jsonDecode(raw) as Map<String, dynamic>;
      if (data['type'] != 'command') return;
      final action = data['action'] as String?;
      if (action == null) return;
      final value = data['value'] as int?;
      ref
          .read(deviceControlProvider(widget.deviceId).notifier)
          .applyRemote(action, value);
    } catch (_) {
      // Ignore malformed frames.
    }
  }

  Future<void> _send(String action, {int? value}) async {
    final error = await ref
        .read(deviceControlProvider(widget.deviceId).notifier)
        .sendCommand(action, value: value, applyLocally: !widget.isOwner);
    if (!mounted || error == null) return;
    AppSnackBar.show(context, message: error, type: SnackBarType.error);
  }

  @override
  Widget build(BuildContext context) {
    final playback = ref.watch(deviceControlProvider(widget.deviceId));
    // Owner only: keep the socket connection alive while this screen is mounted.
    final wsState =
        widget.isOwner ? ref.watch(wsProvider(_wsPath)) : null;

    final title = widget.device?.name ?? 'Device';

    return Scaffold(
      appBar: AppBar(title: Text(title)),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          _banner(context),
          if (wsState != null) ...[
            const SizedBox(height: 8),
            _WsStatusChip(state: wsState),
          ],
          const SizedBox(height: 24),
          _NowPlaying(playback: playback),
          const SizedBox(height: 24),
          _controls(playback),
        ],
      ),
    );
  }

  Widget _banner(BuildContext context) {
    final theme = Theme.of(context);

    if (!widget.isOwner) {
      final owner = widget.ownerEmail ?? 'another user';
      return _InfoBanner(
        icon: Icons.cast_connected,
        text: "You are controlling $owner's device",
        color: theme.colorScheme.primary,
      );
    }

    final delegate = widget.device?.delegate;
    if (delegate != null) {
      return _InfoBanner(
        icon: Icons.cast_connected,
        text: 'Controlled by ${delegate.email}',
        color: theme.colorScheme.primary,
      );
    }
    return _InfoBanner(
      icon: Icons.person_outline,
      text: 'No active delegate. Only you can control this device.',
      color: theme.disabledColor,
    );
  }

  Widget _controls(PlaybackState playback) {
    final busy = playback.inFlight;

    return Column(
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceEvenly,
          children: [
            FilledButton.icon(
              onPressed: busy
                  ? null
                  : () => _send(playback.isPlaying ? 'pause' : 'play'),
              icon: Icon(playback.isPlaying ? Icons.pause : Icons.play_arrow),
              label: Text(playback.isPlaying ? 'Pause' : 'Play'),
            ),
            OutlinedButton.icon(
              onPressed: busy ? null : () => _send('next'),
              icon: const Icon(Icons.skip_next),
              label: const Text('Next'),
            ),
          ],
        ),
        const SizedBox(height: 24),
        Row(
          children: [
            const Icon(Icons.volume_down),
            Expanded(
              child: Slider(
                value: playback.volume.toDouble(),
                min: 0,
                max: 100,
                divisions: 100,
                label: '${playback.volume}',
                onChanged: busy
                    ? null
                    : (v) => ref
                        .read(deviceControlProvider(widget.deviceId).notifier)
                        .setVolumeLocal(v.round()),
                onChangeEnd: busy
                    ? null
                    : (v) => _send('volume', value: v.round()),
              ),
            ),
            const Icon(Icons.volume_up),
            const SizedBox(width: 8),
            SizedBox(
              width: 36,
              child: Text('${playback.volume}', textAlign: TextAlign.end),
            ),
          ],
        ),
      ],
    );
  }
}

class _NowPlaying extends StatelessWidget {
  const _NowPlaying({required this.playback});

  final PlaybackState playback;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      children: [
        Icon(
          playback.isPlaying ? Icons.graphic_eq : Icons.music_note,
          size: 48,
          color: theme.colorScheme.primary,
        ),
        const SizedBox(height: 8),
        Text('Track #${playback.trackPosition}', style: theme.textTheme.titleMedium),
        Text(
          playback.isPlaying ? 'Playing' : 'Paused',
          style: theme.textTheme.bodySmall?.copyWith(color: theme.hintColor),
        ),
      ],
    );
  }
}

class _InfoBanner extends StatelessWidget {
  const _InfoBanner({
    required this.icon,
    required this.text,
    required this.color,
  });

  final IconData icon;
  final String text;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        children: [
          Icon(icon, size: 18, color: color),
          const SizedBox(width: 8),
          Expanded(child: Text(text)),
        ],
      ),
    );
  }
}

class _WsStatusChip extends StatelessWidget {
  const _WsStatusChip({required this.state});

  final WsConnectionState state;

  @override
  Widget build(BuildContext context) {
    final (label, color) = switch (state) {
      WsConnectionState.connected => ('Live', Colors.green),
      WsConnectionState.connecting => ('Connecting...', Colors.orange),
      WsConnectionState.disconnected => ('Disconnected', Colors.grey),
      WsConnectionState.error => ('Connection lost', Colors.red),
    };
    return Row(
      children: [
        Icon(Icons.circle, size: 10, color: color),
        const SizedBox(width: 6),
        Text(label, style: Theme.of(context).textTheme.bodySmall),
      ],
    );
  }
}
