import 'dart:async';

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:geolocator/geolocator.dart';
import 'package:music_room/core/api/event_api.dart';
import 'package:music_room/core/api/events_api.dart';
import 'package:music_room/core/api/web_socket_service.dart';
import 'package:music_room/core/models/event.dart';
import 'package:music_room/core/models/queue_track.dart';
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

  // True once the WS has connected at least once; gates the reconnecting banner
  // so it doesn't fire during the initial handshake.
  bool _everConnected = false;

  // Tracks whose vote POST is currently in flight — disables their button.
  final _votingTracks = <String>{};

  bool _suggesting = false;
  bool _inviting = false;

  static const _location = LocationService();

  @override
  void initState() {
    super.initState();
    _wsPath = '/api/v1/events/${widget.eventId}/ws';
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
      ref.invalidate(eventQueueProvider(widget.eventId));
      if (mounted) {
        AppSnackBar.show(context, message: 'Track added to queue', type: SnackBarType.success);
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
    setState(() => _votingTracks.add(trackId));

    double? lat, lng;
    if (event.license == 2) {
      final gpsResult = await _location.currentPositionDetailed();
      if (!mounted) return;
      if (gpsResult is GpsError) {
        setState(() => _votingTracks.remove(trackId));
        _showGpsError(gpsResult.reason);
        return;
      }
      final ok = gpsResult as GpsSuccess;
      lat = ok.lat;
      lng = ok.lng;
    }

    try {
      final rejection = await ref
          .read(eventQueueProvider(widget.eventId).notifier)
          .upvote(trackId, lat: lat, lng: lng);
      if (rejection != null && mounted) {
        _showVoteRejection(rejection, event);
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

  void _showGpsError(GpsFailure reason) {
    final msg = switch (reason) {
      GpsFailure.serviceDisabled =>
        'Location services are disabled. This event verifies your physical presence to allow voting.',
      GpsFailure.permissionDeniedForever =>
        'Location access is permanently denied. Enable it in app settings to vote in this geofenced event.',
      GpsFailure.permissionDenied =>
        'Location permission is required to vote in this geofenced event.',
    };
    AppSnackBar.show(context, message: msg, type: SnackBarType.error);
    if (reason == GpsFailure.permissionDeniedForever) {
      Geolocator.openAppSettings();
    }
  }

  void _showVoteRejection(VoteRejection rejection, Event event) {
    final msg = switch (rejection) {
      VoteRejection.alreadyVoted => 'You have already voted for this track',
      VoteRejection.notInvited => 'You are not invited to vote in this event',
      VoteRejection.outOfRange => 'You must be at the event location to vote',
      VoteRejection.votingClosed => _votingClosedMessage(event),
      VoteRejection.missingCoords => 'Location is required to vote in this event',
    };
    AppSnackBar.show(
      context,
      message: msg,
      type: rejection == VoteRejection.alreadyVoted
          ? SnackBarType.warning
          : SnackBarType.error,
    );
  }

  String _votingClosedMessage(Event event) {
    final now = DateTime.now();
    if (event.voteStart != null && now.isBefore(event.voteStart!)) {
      return 'Voting is not open yet';
    }
    return 'Voting has ended';
  }

  Future<void> _handleInvite(Event event) async {
    final friend = await showFriendPicker(context);
    if (friend == null || !mounted) return;
    setState(() => _inviting = true);
    try {
      await ref.read(eventsApiProvider).inviteUser(event.id, friend.id);
      if (mounted) {
        AppSnackBar.show(
          context,
          message: '${friend.displayName} was invited',
          type: SnackBarType.success,
        );
      }
    } on DioException catch (e) {
      if (!mounted) return;
      final msg = e.response?.statusCode == 409
          ? '${friend.displayName} is already invited'
          : 'Failed to invite ${friend.displayName}. Try again.';
      AppSnackBar.show(context, message: msg, type: SnackBarType.error);
    } finally {
      if (mounted) setState(() => _inviting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final connState = ref.watch(wsProvider(_wsPath));
    final queueAsync = ref.watch(eventQueueProvider(widget.eventId));
    final eventAsync = ref.watch(eventDetailProvider(widget.eventId));
    final profileAsync = ref.watch(myProfileProvider);

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

    final event = eventAsync.maybeWhen(data: (e) => e, orElse: () => null);
    final currentUserId = profileAsync.maybeWhen(data: (p) => p.id, orElse: () => null);

    // canInteract: event loaded successfully → user has access to this event.
    // Hides vote + suggest controls when the event is inaccessible (e.g. a
    // private license-1 event the user is not invited to).
    final canInteract = eventAsync.hasValue;

    return Scaffold(
      appBar: AppBar(
        title: Text(event?.name ?? 'Live Queue'),
        actions: [
          if (event != null && currentUserId == event.ownerId)
            IconButton(
              icon: _inviting
                  ? const SizedBox(
                      width: 20,
                      height: 20,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Icon(Icons.person_add_outlined),
              tooltip: 'Invite a friend',
              onPressed: _inviting ? null : () => _handleInvite(event),
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
              child: _suggesting
                  ? const SizedBox(
                      width: 24,
                      height: 24,
                      child: CircularProgressIndicator(
                          strokeWidth: 2, color: Colors.white),
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
                        ? () =>
                            ref.read(wsProvider(_wsPath).notifier).reconnect()
                        : null,
                  )
                : const SizedBox.shrink(key: ValueKey('no-banner')),
          ),
          Expanded(
            child: _buildBody(eventAsync, queueAsync),
          ),
        ],
      ),
    );
  }

  Widget _buildBody(
    AsyncValue<Event> eventAsync,
    AsyncValue<List<QueueTrack>> queueAsync,
  ) {
    // Event fetch failed — show access error instead of the queue.
    if (eventAsync.hasError) {
      return _buildEventError(eventAsync.error!);
    }

    final event = eventAsync.maybeWhen(data: (e) => e, orElse: () => null);

    return queueAsync.when(
      loading: () => const Center(child: CircularProgressIndicator()),
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
                onPressed: () =>
                    ref.invalidate(eventQueueProvider(widget.eventId)),
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
                  onUpvote: event != null
                      ? () => _handleUpvote(track.id, event)
                      : () {},
                  showVoteButton: event != null,
                );
              },
            ),
    );
  }

  Widget _buildEventError(Object error) {
    final is4xx = error is DioException &&
        (error.response?.statusCode ?? 0) >= 400 &&
        (error.response?.statusCode ?? 0) < 500;
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              is4xx ? Icons.lock_outline : Icons.error_outline,
              size: 48,
              color: is4xx ? Colors.grey : Colors.redAccent,
            ),
            const SizedBox(height: 16),
            Text(
              is4xx
                  ? 'You are not invited to this event'
                  : 'Failed to load the event. Check your connection.',
              textAlign: TextAlign.center,
            ),
            if (!is4xx) ...[
              const SizedBox(height: 16),
              ElevatedButton.icon(
                onPressed: () =>
                    ref.invalidate(eventDetailProvider(widget.eventId)),
                icon: const Icon(Icons.refresh),
                label: const Text('Retry'),
              ),
            ],
          ],
        ),
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
    this.showVoteButton = true,
  });

  final QueueTrack track;
  final bool isVoting;
  final VoidCallback onUpvote;
  final bool showVoteButton;

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
      trailing: showVoteButton
          ? isVoting
              ? const SizedBox(
                  width: 24,
                  height: 24,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : IconButton(
                  icon: const Icon(Icons.thumb_up_alt_outlined),
                  onPressed: onUpvote,
                  tooltip: 'Upvote',
                )
          : null,
    );
  }
}
