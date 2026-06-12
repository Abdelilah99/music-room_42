import 'dart:convert';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:music_room/core/api/api_client.dart';
import 'package:music_room/core/models/queue_track.dart';

class EventApi {
  final ApiClient _client;

  EventApi(this._client);

  Future<List<QueueTrack>> getQueue(String eventId) async {
    final res = await _client.dio.get('/api/v1/events/$eventId/queue');
    final list = (res.data as Map<String, dynamic>)['tracks'] as List<dynamic>? ?? [];
    return list
        .map((e) => QueueTrack.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  /// Throws [DioException] with status 409 when the user has already voted.
  Future<void> vote(String eventId, String trackId) async {
    await _client.dio.post('/api/v1/events/$eventId/tracks/$trackId/vote');
  }
}

final eventApiProvider = Provider<EventApi>(
  (ref) => EventApi(ref.watch(apiClientProvider)),
);

/// Parse a raw WS frame into a sorted queue list.
/// Returns null for any unrecognised or malformed frame.
List<QueueTrack>? parseQueueUpdate(dynamic raw) {
  Map<String, dynamic> msg;
  try {
    msg = raw is String
        ? jsonDecode(raw) as Map<String, dynamic>
        : raw as Map<String, dynamic>;
  } catch (_) {
    return null;
  }
  if (msg['type'] != 'queue_update') return null;
  final payload = msg['data'] as List<dynamic>?;
  if (payload == null) return null;
  return payload
      .map((e) => QueueTrack.fromJson(e as Map<String, dynamic>))
      .toList()
    ..sort((a, b) => b.votes.compareTo(a.votes));
}
