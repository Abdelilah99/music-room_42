import 'dart:async';

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:music_room/core/api/event_api.dart';
import 'package:music_room/core/api/events_api.dart';
import 'package:music_room/core/api/web_socket_service.dart';
import 'package:music_room/core/models/queue_track.dart';
import 'package:music_room/core/widgets/music_search_sheet.dart';
import 'package:music_room/features/track_vote/event_queue_provider.dart';

class EventDetailScreen extends ConsumerStatefulWidget {
  const EventDetailScreen({super.key, required this.eventId});

  final String eventId;

  @override
  ConsumerState<EventDetailScreen> createState() => _EventDetailScreenState();
}

class _EventDetailScreenState extends ConsumerState<EventDetailScreen> {
  late final String _wsPath;
  StreamSubscription<dynamic>? _wsSub;

  // True once the WS has connected at least once; gates the reconnecting banner
  // so it doesn't fire during the initial handshake.
  bool _everConnected = false;

  // Tracks whose vote POST is currently in flight — disables their button.
  final _votingTracks = <String>{};

  bool _suggesting = false;

  @override
  void initState() {
    super.initState();
    _wsPath = '/api/v1/ws/${widget.eventId}';
    // Set up the stream subscription after the first build so ref.watch has
    // already registered wsProvider as a dependency (keeping it alive).
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      _wsSub ??= ref
          .read(wsProvider(_wsPath).notifier)
          .messageStream
          .listen(_onWsMessage);
    });
  }

  @override
  void dispose() {
    _wsSub?.cancel();
    super.dispose();
  }

  void _onWsMessage(dynamic raw) {
    final tracks = parseQueueUpdate(raw);
    if (tracks != null) {
      ref.read(eventQueueProvider(widget.eventId).notifier).applyUpdate(tracks);
    }
  }

  Future<void> _handleSuggest() async {
    final track = await showMusicSearchSheet(context);
    if (track == null || !mounted) return;
    setState(() => _suggesting = true);
    try {
      await ref.read(eventsApiProvider).suggestTrack(
            widget.eventId,
            externalId: track.externalId,
            title: track.title,
            artist: track.artist,
          );
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Track added to queue')),
        );
      }
    } on DioException catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            e.response?.statusCode == 409
                ? 'Track is already in the queue'
                : 'Failed to add track, please try again',
          ),
        ),
      );
    } finally {
      if (mounted) setState(() => _suggesting = false);
    }
  }

  Future<void> _handleUpvote(String trackId) async {
    if (_votingTracks.contains(trackId)) return;
    setState(() => _votingTracks.add(trackId));
    try {
      final success = await ref
          .read(eventQueueProvider(widget.eventId).notifier)
          .upvote(trackId);
      if (!success && mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            backgroundColor: Colors.orange,
            behavior: SnackBarBehavior.floating,
            content: Row(
              children: const [
                Icon(Icons.warning_amber_rounded, color: Colors.white),
                SizedBox(width: 12),
                Expanded(
                  child: Text(
                    'You have already voted for this track',
                    style: TextStyle(
                        color: Colors.white, fontWeight: FontWeight.w500),
                  ),
                ),
              ],
            ),
          ),
        );
      }
    } catch (_) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Vote failed, please try again')),
        );
      }
    } finally {
      if (mounted) setState(() => _votingTracks.remove(trackId));
    }
  }

  @override
  Widget build(BuildContext context) {
    final connState = ref.watch(wsProvider(_wsPath));
    final queueAsync = ref.watch(eventQueueProvider(widget.eventId));

    // Record the first successful connection so the banner only shows on
    // subsequent reconnects, not during the initial handshake.
    ref.listen<WsConnectionState>(wsProvider(_wsPath), (_, next) {
      if (next == WsConnectionState.connected && !_everConnected) {
        setState(() => _everConnected = true);
      }
    });

    final showBanner = _everConnected &&
        (connState == WsConnectionState.connecting ||
            connState == WsConnectionState.error);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Live Queue'),
        actions: [
          Padding(
            padding: const EdgeInsets.only(right: 12),
            child: _WsDot(state: connState),
          ),
        ],
      ),
      floatingActionButton: FloatingActionButton(
        onPressed: _suggesting ? null : _handleSuggest,
        child: _suggesting
            ? const SizedBox(
                width: 24,
                height: 24,
                child: CircularProgressIndicator(
                    strokeWidth: 2, color: Colors.white),
              )
            : const Icon(Icons.add),
      ),
      body: Column(
        children: [
          AnimatedSwitcher(
            duration: const Duration(milliseconds: 200),
            child: showBanner
                ? _ReconnectBanner(
                    key: const ValueKey('banner'),
                    isError: connState == WsConnectionState.error,
                    onRetry: connState == WsConnectionState.error
                        ? () =>
                            ref.read(wsProvider(_wsPath).notifier).reconnect()
                        : null,
                  )
                : const SizedBox.shrink(key: ValueKey('no-banner')),
          ),
          Expanded(
            child: queueAsync.when(
              loading: () =>
                  const Center(child: CircularProgressIndicator()),
              error: (_, _) => Center(
                child: Padding(
                  padding: const EdgeInsets.all(16),
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      const Icon(Icons.error_outline,
                          size: 48, color: Colors.redAccent),
                      const SizedBox(height: 16),
                      const Text(
                        'Failed to load the queue. Check your connection.',
                        textAlign: TextAlign.center,
                      ),
                      const SizedBox(height: 16),
                      ElevatedButton.icon(
                        onPressed: () => ref
                            .invalidate(eventQueueProvider(widget.eventId)),
                        icon: const Icon(Icons.refresh),
                        label: const Text('Retry'),
                      ),
                    ],
                  ),
                ),
              ),
              data: (tracks) => tracks.isEmpty
                  ? const Center(
                      child: Padding(
                        padding: EdgeInsets.all(24),
                        child: Column(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            Icon(Icons.queue_music_outlined,
                                size: 64, color: Colors.grey),
                            SizedBox(height: 16),
                            Text(
                              'No tracks in the queue yet',
                              style: TextStyle(color: Colors.grey),
                            ),
                          ],
                        ),
                      ),
                    )
                  : ListView.separated(
                      itemCount: tracks.length,
                      separatorBuilder: (_, _) => const Divider(height: 1),
                      itemBuilder: (_, i) {
                        final track = tracks[i];
                        return _TrackTile(
                          track: track,
                          isVoting: _votingTracks.contains(track.id),
                          onUpvote: () => _handleUpvote(track.id),
                        );
                      },
                    ),
            ),
          ),
        ],
      ),
    );
  }
}

// ── Widgets ──────────────────────────────────────────────────────────────────

class _WsDot extends StatelessWidget {
  const _WsDot({required this.state});

  final WsConnectionState state;

  @override
  Widget build(BuildContext context) {
    final (color, label) = switch (state) {
      WsConnectionState.connected => (Colors.green, 'Connected'),
      WsConnectionState.connecting => (Colors.orange, 'Connecting…'),
      WsConnectionState.disconnected => (Colors.grey, 'Disconnected'),
      WsConnectionState.error => (Colors.red, 'Connection error'),
    };
    return Tooltip(
      message: label,
      child: CircleAvatar(radius: 6, backgroundColor: color),
    );
  }
}

class _ReconnectBanner extends StatelessWidget {
  const _ReconnectBanner({
    super.key,
    required this.isError,
    required this.onRetry,
  });

  final bool isError;
  final VoidCallback? onRetry;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final bgColor = isError ? scheme.errorContainer : scheme.tertiaryContainer;
    final fgColor =
        isError ? scheme.onErrorContainer : scheme.onTertiaryContainer;
    final label =
        isError ? 'Connection lost — live updates paused' : 'Reconnecting…';

    return Material(
      color: bgColor,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
        child: Row(
          children: [
            if (!isError)
              SizedBox(
                width: 14,
                height: 14,
                child: CircularProgressIndicator(
                  strokeWidth: 2,
                  color: fgColor,
                ),
              )
            else
              Icon(Icons.wifi_off, size: 16, color: fgColor),
            const SizedBox(width: 8),
            Expanded(
              child: Text(
                label,
                style: Theme.of(context)
                    .textTheme
                    .bodySmall
                    ?.copyWith(color: fgColor),
              ),
            ),
            if (onRetry != null)
              TextButton(
                style: TextButton.styleFrom(foregroundColor: fgColor),
                onPressed: onRetry,
                child: const Text('Retry'),
              ),
          ],
        ),
      ),
    );
  }
}

class _TrackTile extends StatelessWidget {
  const _TrackTile({
    required this.track,
    required this.isVoting,
    required this.onUpvote,
  });

  final QueueTrack track;
  final bool isVoting;
  final VoidCallback onUpvote;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;

    return ListTile(
      contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
      leading: CircleAvatar(
        backgroundColor: scheme.primaryContainer,
        foregroundColor: scheme.onPrimaryContainer,
        child: Text(
          '${track.votes}',
          style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 13),
        ),
      ),
      title: Text(
        track.name,
        style: const TextStyle(fontWeight: FontWeight.w600),
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
      ),
      subtitle: Text(
        track.artist,
        style: TextStyle(color: scheme.onSurfaceVariant),
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
      ),
      trailing: isVoting
          ? const SizedBox(
              width: 24,
              height: 24,
              child: CircularProgressIndicator(strokeWidth: 2),
            )
          : IconButton(
              icon: const Icon(Icons.thumb_up_alt_outlined),
              onPressed: onUpvote,
              tooltip: 'Upvote',
            ),
    );
  }
}
