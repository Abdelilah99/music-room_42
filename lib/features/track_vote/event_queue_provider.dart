import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:music_room/core/api/event_api.dart';
import 'package:music_room/core/models/queue_track.dart';

enum VoteRejection { alreadyVoted, notInvited, outOfRange, votingClosed, missingCoords }

class EventQueueNotifier extends AsyncNotifier<List<QueueTrack>> {
  EventQueueNotifier(this._eventId);
  final String _eventId;

  EventApi get _api => ref.read(eventApiProvider);

  @override
  Future<List<QueueTrack>> build() => _api.getQueue(_eventId);

  /// Replace the whole queue from a WS queue_update message.
  void applyUpdate(List<QueueTrack> tracks) {
    state = AsyncData(tracks);
  }

  /// Returns null on success, or a [VoteRejection] for a known server refusal.
  /// Optimistic: increments the track immediately and reverts on any failure
  /// before classifying the error, so the revert covers all rejection types.
  Future<VoteRejection?> upvote(String trackId, {double? lat, double? lng}) async {
    final current = state.value;
    if (current == null) return null;

    state = AsyncData(_adjust(current, trackId, 1));

    try {
      await _api.vote(_eventId, trackId, lat: lat, lng: lng);
      return null;
    } on DioException catch (e) {
      state = AsyncData(_adjust(state.value ?? current, trackId, -1));

      final code = e.response?.statusCode;
      if (code == 409) return VoteRejection.alreadyVoted;
      if (code == 403 || code == 400) {
        final data = e.response?.data;
        final errorStr = data is Map ? data['error'] as String? : null;
        final rejection = switch (errorStr) {
          'NOT_INVITED' => VoteRejection.notInvited,
          'OUT_OF_RANGE' => VoteRejection.outOfRange,
          'VOTING_CLOSED' => VoteRejection.votingClosed,
          _ => code == 400 ? VoteRejection.missingCoords : null,
        };
        if (rejection != null) return rejection;
      }
      rethrow;
    }
  }

  List<QueueTrack> _adjust(List<QueueTrack> tracks, String trackId, int delta) {
    return [
      for (final t in tracks)
        if (t.id == trackId)
          t.withVotes(t.votes + delta < 0 ? 0 : t.votes + delta)
        else
          t,
    ]..sort((a, b) => b.votes.compareTo(a.votes));
  }
}

final eventQueueProvider = AsyncNotifierProvider.autoDispose
    .family<EventQueueNotifier, List<QueueTrack>, String>(
  (eventId) => EventQueueNotifier(eventId),
);
