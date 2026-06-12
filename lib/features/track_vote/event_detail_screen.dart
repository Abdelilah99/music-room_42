import 'dart:async';

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:music_room/core/api/event_api.dart';
import 'package:music_room/core/api/events_api.dart';
import 'package:music_room/core/api/web_socket_service.dart';
import 'package:music_room/core/models/event.dart';
import 'package:music_room/core/models/queue_track.dart';
import 'package:music_room/core/models/user.dart';
import 'package:music_room/core/services/location_service.dart';
import 'package:music_room/core/widgets/friend_picker.dart';
import 'package:music_room/core/widgets/music_search_sheet.dart';
import 'package:music_room/features/profile/profile_provider.dart';
import 'package:music_room/features/track_vote/event_queue_provider.dart';
import 'package:music_room/features/track_vote/events_provider.dart';
import 'package:music_room/shared/widgets/snackbar_helper.dart';

class EventDetailScreen extends ConsumerStatefulWidget {
  const EventDetailScreen({super.key, required this.eventId});

  final String eventId;

  @override
  ConsumerState<EventDetailScreen> createState() => _EventDetailScreenState();
}

class _EventDetailScreenState extends ConsumerState<EventDetailScreen> {
  late final String _wsPath;
  StreamSubscription<dynamic>? _wsSub;

  bool _everConnected = false;
  final _votingTracks = <String>{};
  bool _suggesting = false;
  bool _inviting = false;

  @override
  void initState() {
    super.initState();
    _wsPath = '/api/v1/ws/${widget.eventId}';
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
      ref.invalidate(eventQueueProvider(widget.eventId));
      if (mounted) {
        AppSnackBar.show(context, message: 'Track added to queue');
      }
    } on DioException catch (e) {
      if (!mounted) return;
      AppSnackBar.show(
        context,
        message: e.response?.statusCode == 409
            ? 'Track is already in the queue'
            : 'Failed to add track, please try again',
        type: SnackBarType.error,
      );
    } finally {
      if (mounted) setState(() => _suggesting = false);
    }
  }

  Future<void> _handleUpvote(String trackId, Event event) async {
    if (_votingTracks.contains(trackId)) return;

    double? lat, lng;
    if (event.license == 2) {
      final pos = await ref.read(locationServiceProvider).currentPosition();
      if (!mounted) return;
      if (pos == null) {
        AppSnackBar.show(
          context,
          message: 'Location access is required to vote at this event. Please enable location in settings.',
          type: SnackBarType.error,
        );
        return;
      }
      lat = pos.lat;
      lng = pos.lng;
    }

    setState(() => _votingTracks.add(trackId));
    try {
      final errorCode = await ref
          .read(eventQueueProvider(widget.eventId).notifier)
          .upvote(trackId, lat: lat, lng: lng);
      if (errorCode != null && mounted) {
        _showVoteError(errorCode, event);
      }
    } catch (_) {
      if (mounted) {
        AppSnackBar.show(
          context,
          message: 'Vote failed, please try again',
          type: SnackBarType.error,
        );
      }
    } finally {
      if (mounted) setState(() => _votingTracks.remove(trackId));
    }
  }

  void _showVoteError(String code, Event event) {
    final String message;
    switch (code) {
      case 'NOT_INVITED':
        message = 'You are not invited to vote in this event';
      case 'OUT_OF_RANGE':
        message = 'You must be at the event location to vote';
      case 'VOTING_CLOSED':
        final now = DateTime.now();
        final start = event.voteStart;
        if (start != null && now.isBefore(start)) {
          message = 'Voting is not open yet';
        } else {
          message = 'Voting has ended';
        }
      default:
        if (code.contains('already voted')) {
          message = 'You have already voted for this track';
        } else {
          message = 'Vote failed, please try again';
        }
    }
    AppSnackBar.show(context, message: message, type: SnackBarType.warning);
  }

  Future<void> _handleInvite() async {
    final User? friend = await showFriendPicker(context);
    if (friend == null || !mounted) return;
    setState(() => _inviting = true);
    try {
      await ref.read(eventsApiProvider).invite(widget.eventId, friend.id);
      if (mounted) {
        AppSnackBar.show(context, message: '${friend.displayName} invited successfully');
      }
    } on DioException catch (e) {
      if (!mounted) return;
      final code = (e.response?.data as Map<String, dynamic>?)?['error'] as String?;
      final message = code == 'ALREADY_INVITED'
          ? '${friend.displayName} is already invited'
          : 'Failed to invite ${friend.displayName}, please try again';
      AppSnackBar.show(context, message: message, type: SnackBarType.error);
    } finally {
      if (mounted) setState(() => _inviting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final connState = ref.watch(wsProvider(_wsPath));
    final queueAsync = ref.watch(eventQueueProvider(widget.eventId));
    final eventAsync = ref.watch(eventByIdProvider(widget.eventId));
    final myProfileAsync = ref.watch(myProfileProvider);

    ref.listen<WsConnectionState>(wsProvider(_wsPath), (_, next) {
      if (next == WsConnectionState.connected && !_everConnected) {
        setState(() => _everConnected = true);
      }
    });

    final showBanner = _everConnected &&
        (connState == WsConnectionState.connecting ||
            connState == WsConnectionState.error);

    final event = eventAsync.asData?.value;
    final currentUserId = myProfileAsync.asData?.value.id;
    final isOwner = event != null && currentUserId != null && event.ownerId == currentUserId;
    // canInteract: event loaded OK → user has access; hide controls if not
    final canInteract = event != null;

    return Scaffold(
      appBar: AppBar(
        title: Text(event?.name ?? 'Live Queue'),
        actions: [
          if (isOwner)
            _inviting
                ? const Padding(
                    padding: EdgeInsets.symmetric(horizontal: 16),
                    child: SizedBox(
                      width: 20,
                      height: 20,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    ),
                  )
                : IconButton(
                    icon: const Icon(Icons.person_add_outlined),
                    tooltip: 'Invite a friend',
                    onPressed: _handleInvite,
                  ),
          Padding(
            padding: const EdgeInsets.only(right: 12),
            child: _WsDot(state: connState),
          ),
        ],
      ),
      floatingActionButton: canInteract
          ? FloatingActionButton(
              onPressed: _suggesting ? null : _handleSuggest,
              tooltip: 'Suggest a track',
              child: _suggesting
                  ? const SizedBox(
                      width: 24,
                      height: 24,
                      child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white),
                    )
                  : const Icon(Icons.add),
            )
          : null,
      body: Column(
        children: [
          AnimatedSwitcher(
            duration: const Duration(milliseconds: 200),
            child: showBanner
                ? _ReconnectBanner(
                    key: const ValueKey('banner'),
                    isError: connState == WsConnectionState.error,
                    onRetry: connState == WsConnectionState.error
                        ? () => ref.read(wsProvider(_wsPath).notifier).reconnect()
                        : null,
                  )
                : const SizedBox.shrink(key: ValueKey('no-banner')),
          ),
          Expanded(
            child: queueAsync.when(
              loading: () => const Center(child: CircularProgressIndicator()),
              error: (_, _) => Center(
                child: Padding(
                  padding: const EdgeInsets.all(16),
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      const Icon(Icons.lock_outline, size: 48, color: Colors.grey),
                      const SizedBox(height: 16),
                      const Text(
                        'Queue unavailable. You may not have access to this event.',
                        textAlign: TextAlign.center,
                      ),
                      const SizedBox(height: 16),
                      ElevatedButton.icon(
                        onPressed: () => ref.invalidate(eventQueueProvider(widget.eventId)),
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
                            Icon(Icons.queue_music_outlined, size: 64, color: Colors.grey),
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
                          onUpvote: canInteract
                              ? () => _handleUpvote(track.id, event)
                              : null,
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
    final fgColor = isError ? scheme.onErrorContainer : scheme.onTertiaryContainer;
    final label = isError ? 'Connection lost — live updates paused' : 'Reconnecting…';

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
                child: CircularProgressIndicator(strokeWidth: 2, color: fgColor),
              )
            else
              Icon(Icons.wifi_off, size: 16, color: fgColor),
            const SizedBox(width: 8),
            Expanded(
              child: Text(
                label,
                style: Theme.of(context).textTheme.bodySmall?.copyWith(color: fgColor),
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
  final VoidCallback? onUpvote;

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
      trailing: onUpvote == null
          ? null
          : isVoting
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
