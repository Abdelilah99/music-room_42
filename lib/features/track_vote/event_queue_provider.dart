import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:music_room/core/api/event_api.dart';
import 'package:music_room/core/models/queue_track.dart';

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

  /// Returns null on success, or the server error code string on a known failure.
  /// Optimistic: increments the specific track immediately and re-sorts.
  /// Revert: decrements that same track — avoids overwriting concurrent WS updates.
  Future<String?> upvote(String trackId, {double? lat, double? lng}) async {
    final current = state.value;
    if (current == null) return null;

    state = AsyncData(_adjust(current, trackId, 1));

    try {
      await _api.vote(_eventId, trackId, lat: lat, lng: lng);
      return null;
    } on DioException catch (e) {
      // Revert by decrementing the current list (which may have been replaced
      // by a queue_update while the POST was in flight).
      state = AsyncData(_adjust(state.value ?? current, trackId, -1));
      final code = (e.response?.data as Map<String, dynamic>?)?['error'] as String?;
      if (code != null) return code;
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
