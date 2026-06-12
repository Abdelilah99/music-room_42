import 'dart:async';
import 'dart:convert';

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:music_room/core/api/playlists_api.dart';
import 'package:music_room/core/api/web_socket_service.dart';
import 'package:music_room/core/models/playlist.dart';
import 'package:music_room/core/widgets/friend_picker.dart';
import 'package:music_room/core/widgets/music_search_sheet.dart';
import 'package:music_room/features/playlist_editor/playlist_detail_provider.dart';
import 'package:music_room/features/playlist_editor/playlists_provider.dart';
import 'package:music_room/features/profile/profile_provider.dart';
import 'package:music_room/shared/widgets/snackbar_helper.dart';
import 'package:music_room/shared/widgets/state_widgets.dart';

class PlaylistEditorScreen extends ConsumerStatefulWidget {
  const PlaylistEditorScreen({super.key, required this.playlistId});

  final String playlistId;

  @override
  ConsumerState<PlaylistEditorScreen> createState() =>
      _PlaylistEditorScreenState();
}

class _PlaylistEditorScreenState extends ConsumerState<PlaylistEditorScreen> {
  StreamSubscription<dynamic>? _wsSub;
  // trackId → intended position; used to detect echo vs. genuine conflict.
  final Map<String, int> _pendingMoves = {};
  bool _inviting = false;
  bool _addingTrack = false;
  // Gates the reconnect banner — don't show it on the initial handshake.
  bool _everConnected = false;

  String get _wsPath => '/api/v1/playlists/${widget.playlistId}/ws';

  @override
  void initState() {
    super.initState();
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
    super.dispose();
  }

  void _onWsMessage(dynamic raw) {
    if (!mounted) return;
    final Map<String, dynamic> event;
    try {
      event =
          (raw is String ? jsonDecode(raw) : raw) as Map<String, dynamic>;
    } catch (_) {
      return;
    }

    final type = event['type'] as String?;
    final notifier =
        ref.read(playlistDetailProvider(widget.playlistId).notifier);

    switch (type) {
      case 'track_added':
        final track = PlaylistTrack.fromJson(
            event['track'] as Map<String, dynamic>);
        notifier.applyTrackAdded(track);
      case 'track_removed':
        notifier.applyTrackRemoved(event['track_id'] as String);
      case 'track_moved':
        final trackId = event['track_id'] as String;
        final position = (event['position'] as num).toInt();
        final pending = _pendingMoves.remove(trackId);
        notifier.applyTrackMoved(trackId, position);
        // Only show conflict snackbar when the server position differs from
        // our optimistic position — equal means it is our own echo.
        if (pending != null && pending != position && mounted) {
          AppSnackBar.show(
            context,
            message: 'Track moved by another collaborator',
            type: SnackBarType.warning,
          );
        }
    }
  }

  // onReorderItem already delivers the adjusted insertion index (no subtract-1 needed).
  void _onReorderItem(int oldIndex, int newIndex) {
    final tracks =
        ref.read(playlistDetailProvider(widget.playlistId)).value?.tracks;
    if (tracks == null || oldIndex >= tracks.length) return;

    final trackId = tracks[oldIndex].id;
    final newPosition = newIndex + 1; // 1-based

    setState(() => _pendingMoves[trackId] = newPosition);

    ref
        .read(playlistDetailProvider(widget.playlistId).notifier)
        .moveTrack(trackId, newPosition)
        .catchError((Object e) {
      if (!mounted) return;
      setState(() => _pendingMoves.remove(trackId));
      final isEditBlocked = e is DioException &&
          (e.response?.data as Map?)?['code'] == 'EDIT_NOT_ALLOWED';
      AppSnackBar.show(
        context,
        message: isEditBlocked
            ? 'You don\'t have permission to edit this playlist'
            : 'Failed to move track',
        type: SnackBarType.error,
      );
    });
  }

  Future<void> _removeTrack(Playlist playlist, PlaylistTrack track) async {
    try {
      await ref
          .read(playlistDetailProvider(widget.playlistId).notifier)
          .removeTrack(playlist.id, track.id);
      if (mounted) {
        AppSnackBar.show(
          context,
          message: '${track.title} removed',
          type: SnackBarType.success,
        );
      }
    } on DioException catch (e) {
      if (!mounted) return;
      final isEditBlocked =
          (e.response?.data as Map?)?['code'] == 'EDIT_NOT_ALLOWED';
      AppSnackBar.show(
        context,
        message: isEditBlocked
            ? 'You don\'t have permission to edit this playlist'
            : 'Failed to remove track',
        type: SnackBarType.error,
      );
    } catch (_) {
      if (mounted) {
        AppSnackBar.show(
          context,
          message: 'Failed to remove track',
          type: SnackBarType.error,
        );
      }
    }
  }

  Future<void> _handleInvite(Playlist playlist) async {
    final friend = await showFriendPicker(context);
    if (friend == null || !mounted) return;

    setState(() => _inviting = true);
    try {
      await ref
          .read(playlistsApiProvider)
          .inviteUser(playlist.id, friend.id);
      if (mounted) {
        AppSnackBar.show(
          context,
          message: '${friend.displayName} invited',
          type: SnackBarType.success,
        );
      }
    } on DioException catch (e) {
      if (!mounted) return;
      if (e.response?.statusCode == 409) {
        AppSnackBar.show(
          context,
          message: '${friend.displayName} is already invited',
          type: SnackBarType.warning,
        );
      } else {
        AppSnackBar.show(
          context,
          message: 'Failed to invite user',
          type: SnackBarType.error,
        );
      }
    } catch (_) {
      if (mounted) {
        AppSnackBar.show(
          context,
          message: 'Failed to invite user',
          type: SnackBarType.error,
        );
      }
    } finally {
      if (mounted) setState(() => _inviting = false);
    }
  }

  Future<void> _handleAddTrack(String playlistId) async {
    final track = await showMusicSearchSheet(context);
    if (track == null || !mounted) return;

    setState(() => _addingTrack = true);
    try {
      await ref.read(playlistsApiProvider).addTrack(
            playlistId,
            externalId: track.externalId,
            title: track.title,
            artist: track.artist,
          );
      // WS track_added echo will insert the track via applyTrackAdded.
    } on DioException catch (e) {
      if (!mounted) return;
      final data = e.response?.data;
      final isConflict = e.response?.statusCode == 409;
      final isEditBlocked =
          (data as Map?)?['code'] == 'EDIT_NOT_ALLOWED';
      AppSnackBar.show(
        context,
        message: isConflict
            ? '${track.title} is already in this playlist'
            : isEditBlocked
                ? 'You don\'t have permission to edit this playlist'
                : 'Failed to add track',
        type: isConflict || isEditBlocked
            ? SnackBarType.warning
            : SnackBarType.error,
      );
    } catch (_) {
      if (mounted) {
        AppSnackBar.show(
          context,
          message: 'Failed to add track',
          type: SnackBarType.error,
        );
      }
    } finally {
      if (mounted) setState(() => _addingTrack = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final connState = ref.watch(wsProvider(_wsPath));
    final detailAsync = ref.watch(playlistDetailProvider(widget.playlistId));
    final profileAsync = ref.watch(myProfileProvider);

    // Record the first successful connection so the banner only fires on
    // subsequent drops, not during the initial handshake.
    ref.listen<WsConnectionState>(wsProvider(_wsPath), (_, next) {
      if (next == WsConnectionState.connected && !_everConnected) {
        setState(() => _everConnected = true);
      }
    });

    final showBanner = _everConnected &&
        (connState == WsConnectionState.connecting ||
            connState == WsConnectionState.error);

    final playlist =
        detailAsync.maybeWhen(data: (d) => d.playlist, orElse: () => null);
    final currentUserId =
        profileAsync.maybeWhen(data: (p) => p.id, orElse: () => null);

    final canEdit = playlist != null;

    return Scaffold(
      appBar: AppBar(
        title: Text(playlist?.name ?? 'Playlist'),
        actions: [
          if (playlist != null && currentUserId == playlist.ownerId)
            IconButton(
              onPressed: _inviting ? null : () => _handleInvite(playlist),
              icon: const Icon(Icons.person_add_outlined),
              tooltip: 'Invite collaborator',
            ),
          Padding(
            padding: const EdgeInsets.only(right: 12),
            child: _WsDot(state: connState),
          ),
        ],
      ),
      floatingActionButton: canEdit
          ? FloatingActionButton(
              onPressed: _addingTrack
                  ? null
                  : () => _handleAddTrack(playlist.id),
              tooltip: 'Add track',
              child: _addingTrack
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
            child: detailAsync.when(
              loading: () => const AppLoadingWidget(),
              error: (_, _) => AppErrorWidget(
                message: 'Failed to load playlist. Check your connection.',
                onRetry: () =>
                    ref.invalidate(playlistDetailProvider(widget.playlistId)),
              ),
              data: (detail) => Column(
                children: [
                  _OwnerHeader(playlist: detail.playlist),
                  const Divider(height: 1),
                  Expanded(
                    child: detail.tracks.isEmpty
                        ? const AppEmptyStateWidget(
                            icon: Icons.queue_music_outlined,
                            message: 'No tracks yet',
                          )
                        : ReorderableListView.builder(
                            itemCount: detail.tracks.length,
                            onReorderItem: _onReorderItem,
                            itemBuilder: (_, i) {
                              final track = detail.tracks[i];
                              return _TrackTile(
                                key: ValueKey(track.id),
                                track: track,
                                onRemove: () =>
                                    _removeTrack(detail.playlist, track),
                              );
                            },
                          ),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _OwnerHeader extends ConsumerWidget {
  const _OwnerHeader({required this.playlist});

  final Playlist playlist;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final ownerName =
        ref.watch(playlistOwnerNameProvider(playlist.ownerId));
    final scheme = Theme.of(context).colorScheme;

    final name =
        ownerName.maybeWhen(data: (n) => n, orElse: () => null);
    final initial =
        (name != null && name.isNotEmpty) ? name[0].toUpperCase() : '?';

    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 12),
      child: Row(
        children: [
          CircleAvatar(
            radius: 16,
            backgroundColor: scheme.primaryContainer,
            child: Text(
              initial,
              style: TextStyle(
                fontWeight: FontWeight.bold,
                fontSize: 14,
                color: scheme.onPrimaryContainer,
              ),
            ),
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              'by ${name ?? '…'}',
              style: Theme.of(context).textTheme.bodyMedium,
              overflow: TextOverflow.ellipsis,
            ),
          ),
          const SizedBox(width: 8),
          _LicenseBadge(isOpen: playlist.isOpenLicense),
        ],
      ),
    );
  }
}

class _LicenseBadge extends StatelessWidget {
  const _LicenseBadge({required this.isOpen});

  final bool isOpen;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(
          isOpen ? Icons.public : Icons.lock_outline,
          size: 14,
          color: scheme.outline,
        ),
        const SizedBox(width: 4),
        Text(
          isOpen ? 'Anyone can edit' : 'Invited editors only',
          style: Theme.of(context)
              .textTheme
              .bodySmall
              ?.copyWith(color: scheme.outline),
        ),
      ],
    );
  }
}

class _TrackTile extends StatelessWidget {
  const _TrackTile({
    super.key,
    required this.track,
    required this.onRemove,
  });

  final PlaylistTrack track;
  final VoidCallback onRemove;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;

    return ListTile(
      contentPadding: const EdgeInsets.fromLTRB(16, 4, 8, 4),
      leading: CircleAvatar(
        backgroundColor: scheme.secondaryContainer,
        child: Text(
          '${track.position}',
          style: TextStyle(
            fontWeight: FontWeight.bold,
            fontSize: 13,
            color: scheme.onSecondaryContainer,
          ),
        ),
      ),
      title: Text(
        track.title,
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
        style: const TextStyle(fontWeight: FontWeight.w500),
      ),
      subtitle: Text(
        track.artist,
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
      ),
      trailing: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          IconButton(
            icon: Icon(Icons.delete_outline, color: scheme.error),
            onPressed: onRemove,
            tooltip: 'Remove',
          ),
          const Icon(Icons.drag_handle),
        ],
      ),
    );
  }
}

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
            Icon(
              isError ? Icons.wifi_off : Icons.sync,
              size: 18,
              color: fgColor,
            ),
            const SizedBox(width: 8),
            Expanded(
              child: Text(label,
                  style: TextStyle(color: fgColor, fontSize: 13)),
            ),
            if (onRetry != null)
              TextButton(
                onPressed: onRetry,
                style: TextButton.styleFrom(foregroundColor: fgColor),
                child: const Text('Retry'),
              ),
          ],
        ),
      ),
    );
  }
}
