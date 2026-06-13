import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:just_audio/just_audio.dart';
import 'package:music_room/core/api/music_api.dart';
import 'package:music_room/core/api/playlists_api.dart';
import 'package:music_room/core/api/web_socket_service.dart';
import 'package:music_room/core/models/device.dart';
import 'package:music_room/core/models/playlist.dart';
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

  // Held so it can be cancelled in dispose(); otherwise a frame can arrive
  // after teardown and write to a provider that is already gone.
  StreamSubscription<dynamic>? _wsSub;

  // The owner's device is the actual player. A playlist is cast to it, and the
  // PlaybackState (driven by the owner's controls and remote delegate commands)
  // is reconciled onto this audio player.
  AudioPlayer? _player;
  List<PlaylistTrack> _castTracks = const [];
  String? _castName;
  int _loadedPosition = 0; // PlaybackState.trackPosition currently loaded
  bool _casting = false; // a cast/load is in flight

  @override
  void initState() {
    super.initState();
    // Only the owner is the audio player.
    if (widget.isOwner) {
      _player = AudioPlayer();
    }
    // Both owner and delegate subscribe to the device socket, so each sees the
    // other's commands and now-playing live. Done after the first frame so the
    // provider is alive (it is watched in build for its state).
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      _wsSub = ref
          .read(wsProvider(_wsPath).notifier)
          .messageStream
          .listen(_onWsMessage);
    });
  }

  @override
  void dispose() {
    _wsSub?.cancel();
    _player?.dispose();
    super.dispose();
  }

  // The owner picks one of their playlists to play on this device.
  Future<void> _castPlaylist() async {
    final selected = await _showPlaylistPicker();
    if (selected == null || !mounted) return;
    setState(() => _casting = true);
    try {
      final detail = await ref.read(playlistsApiProvider).get(selected.id);
      if (!mounted) return;
      setState(() {
        _castTracks = detail.tracks;
        _castName = detail.playlist.name;
        _loadedPosition = 0;
      });
      if (detail.tracks.isEmpty && mounted) {
        AppSnackBar.show(context,
            message: 'That playlist has no tracks yet',
            type: SnackBarType.warning);
      }
    } catch (_) {
      if (mounted) {
        AppSnackBar.show(context,
            message: 'Could not load that playlist', type: SnackBarType.error);
      }
    } finally {
      if (mounted) setState(() => _casting = false);
    }
  }

  Future<Playlist?> _showPlaylistPicker() {
    return showModalBottomSheet<Playlist>(
      context: context,
      isScrollControlled: true,
      builder: (_) => const _PlaylistPickerSheet(),
    );
  }

  // Reconciles the audio player with the latest PlaybackState. Commands arrive
  // here whether the owner pressed a control or a delegate did (via the socket).
  void _reconcile(PlaybackState? prev, PlaybackState next) {
    final player = _player;
    if (player == null || _castTracks.isEmpty) return;

    if (prev?.volume != next.volume) {
      player.setVolume((next.volume / 100).clamp(0.0, 1.0));
    }

    // A track change (e.g. "next") loads and plays the new track.
    if (prev?.trackPosition != next.trackPosition) {
      unawaited(_loadAndPlay(next.trackPosition, play: next.isPlaying));
      return;
    }

    // Play/pause on the current track.
    if (prev?.isPlaying != next.isPlaying) {
      if (next.isPlaying) {
        if (_loadedPosition == next.trackPosition) {
          player.play();
        } else {
          unawaited(_loadAndPlay(next.trackPosition, play: true));
        }
      } else {
        player.pause();
      }
    }
  }

  Future<void> _loadAndPlay(int position, {required bool play}) async {
    final player = _player;
    if (player == null || _castTracks.isEmpty) return;
    // trackPosition is 1-based and grows past the end; wrap it onto the list.
    final idx = (position - 1) % _castTracks.length;
    final track = _castTracks[idx];
    try {
      final resolved =
          await ref.read(musicApiProvider).getTrack(track.externalId);
      final url = resolved.previewUrl;
      if (url == null || url.isEmpty) return;
      await player.setUrl(url);
      final volume = ref.read(deviceControlProvider(widget.deviceId)).volume;
      await player.setVolume((volume / 100).clamp(0.0, 1.0));
      _loadedPosition = position;
      // play() resolves at end-of-track, so it must not be awaited.
      if (play) unawaited(player.play());
      // Tell the delegate what is now playing.
      unawaited(ref
          .read(deviceControlProvider(widget.deviceId).notifier)
          .announceTrack('${track.title}  -  ${track.artist}'));
    } catch (_) {
      // Resolution/playback failed; leave the player idle.
    }
  }

  // The track the device is currently on, for the now-playing display.
  PlaylistTrack? _currentTrack(int position) {
    if (_castTracks.isEmpty) return null;
    return _castTracks[(position - 1) % _castTracks.length];
  }

  void _onWsMessage(dynamic raw) {
    if (raw is! String) return;
    try {
      final data = jsonDecode(raw) as Map<String, dynamic>;
      if (data['type'] != 'command') return;
      final action = data['action'] as String?;
      if (action == null) return;
      final value = data['value'] as int?;
      final track = data['track'] as String?;
      ref
          .read(deviceControlProvider(widget.deviceId).notifier)
          .applyRemote(action, value, track);
    } catch (_) {
      // Ignore malformed frames.
    }
  }

  Future<void> _send(String action, {int? value}) async {
    // When the socket is up, both owner and delegate rely on the WS echo to
    // update state (single source of truth). If it is down, apply locally so
    // the controls stay responsive.
    final socketUp =
        ref.read(wsProvider(_wsPath)) == WsConnectionState.connected;
    final error = await ref
        .read(deviceControlProvider(widget.deviceId).notifier)
        .sendCommand(action, value: value, applyLocally: !socketUp);
    if (!mounted || error == null) return;
    AppSnackBar.show(context, message: error, type: SnackBarType.error);
  }

  @override
  Widget build(BuildContext context) {
    final playback = ref.watch(deviceControlProvider(widget.deviceId));
    // Both sides keep the socket alive while this screen is mounted.
    final wsState = ref.watch(wsProvider(_wsPath));

    // Owner only: drive the real audio player from every PlaybackState change,
    // whether it came from the local controls or a remote delegate command.
    if (widget.isOwner) {
      ref.listen(deviceControlProvider(widget.deviceId), _reconcile);
    }

    final title = widget.device?.name ?? 'Device';
    final current =
        widget.isOwner ? _currentTrack(playback.trackPosition) : null;

    return Scaffold(
      appBar: AppBar(title: Text(title)),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          _banner(context),
          const SizedBox(height: 8),
          _WsStatusChip(state: wsState),
          if (widget.isOwner) ...[
            const SizedBox(height: 16),
            _castSection(context),
          ],
          const SizedBox(height: 24),
          _NowPlaying(
            playback: playback,
            track: current,
            label: playback.nowPlaying,
            hasCast: _castTracks.isNotEmpty,
          ),
          const SizedBox(height: 24),
          _controls(playback),
        ],
      ),
    );
  }

  Widget _castSection(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      decoration: BoxDecoration(
        color: scheme.surfaceContainerHighest,
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        children: [
          Icon(Icons.queue_music, size: 18, color: scheme.primary),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              _castName == null
                  ? 'No playlist cast to this device'
                  : 'Playing: $_castName',
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
          ),
          TextButton(
            onPressed: _casting ? null : _castPlaylist,
            child: _casting
                ? const SizedBox(
                    width: 16,
                    height: 16,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : Text(_castName == null ? 'Cast' : 'Change'),
          ),
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
  const _NowPlaying({
    required this.playback,
    this.track,
    this.label,
    this.hasCast = false,
  });

  final PlaybackState playback;
  final PlaylistTrack? track; // owner: resolved locally
  final String? label; // delegate: "Title - Artist" from the owner's broadcast
  final bool hasCast;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final status = playback.isPlaying ? 'Playing' : 'Paused';
    final title = track?.title ??
        label ??
        (hasCast ? 'Track #${playback.trackPosition}' : 'Nothing playing');
    final subtitle = track != null ? '${track!.artist}  -  $status' : status;

    return Column(
      children: [
        Icon(
          playback.isPlaying ? Icons.graphic_eq : Icons.music_note,
          size: 48,
          color: theme.colorScheme.primary,
        ),
        const SizedBox(height: 8),
        Text(
          title,
          style: theme.textTheme.titleMedium,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
        ),
        Text(
          subtitle,
          style: theme.textTheme.bodySmall?.copyWith(color: theme.hintColor),
        ),
      ],
    );
  }
}

class _PlaylistPickerSheet extends ConsumerStatefulWidget {
  const _PlaylistPickerSheet();

  @override
  ConsumerState<_PlaylistPickerSheet> createState() =>
      _PlaylistPickerSheetState();
}

class _PlaylistPickerSheetState extends ConsumerState<_PlaylistPickerSheet> {
  late final Future<List<Playlist>> _playlists;

  @override
  void initState() {
    super.initState();
    // Capture the fetch once so rebuilds don't restart it.
    _playlists = ref.read(playlistsApiProvider).list();
  }

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: MediaQuery.of(context).size.height * 0.5,
      child: FutureBuilder<List<Playlist>>(
        future: _playlists,
        builder: (context, snap) {
          if (snap.connectionState != ConnectionState.done) {
            return const Center(child: CircularProgressIndicator());
          }
          if (snap.hasError) {
            return const Center(child: Text('Failed to load playlists'));
          }
          final playlists = snap.data ?? const [];
          if (playlists.isEmpty) {
            return const Center(child: Text('No playlists yet'));
          }
          return ListView(
            children: [
              const Padding(
                padding: EdgeInsets.all(16),
                child: Text(
                  'Cast a playlist',
                  style: TextStyle(fontWeight: FontWeight.bold, fontSize: 16),
                ),
              ),
              for (final p in playlists)
                ListTile(
                  leading: const Icon(Icons.queue_music),
                  title: Text(p.name),
                  subtitle: Text(p.isPublic ? 'Public' : 'Private'),
                  onTap: () => Navigator.pop(context, p),
                ),
            ],
          );
        },
      ),
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
