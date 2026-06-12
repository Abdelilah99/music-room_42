import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:music_room/core/api/api_client.dart';
import 'package:music_room/core/models/event.dart';

class EventsApi {
  final ApiClient _client;

  EventsApi(this._client);

  // GET /events?q=<query> -> {"events": [...]}
  Future<List<Event>> list({String? query}) async {
    final res = await _client.dio.get(
      '/api/v1/events',
      queryParameters: (query != null && query.isNotEmpty) ? {'q': query} : null,
    );
    final raw = res.data['events'] as List<dynamic>? ?? [];
    return raw.map((e) => Event.fromJson(e as Map<String, dynamic>)).toList();
  }

  // GET /events/:id -> Event
  Future<Event> get(String eventId) async {
    final res = await _client.dio.get('/api/v1/events/$eventId');
    return Event.fromJson(res.data as Map<String, dynamic>);
  }

  // POST /events -> 201 Event
  Future<Event> create({
    required String name,
    required String visibility,
    required int license,
    double? lat,
    double? lng,
    double? radius,
    DateTime? voteStart,
    DateTime? voteEnd,
  }) async {
    final body = <String, dynamic>{
      'name': name,
      'visibility': visibility,
      'license': license,
      'lat': ?lat,
      'lng': ?lng,
      'radius': ?radius,
      if (voteStart != null) 'vote_start': voteStart.toUtc().toIso8601String(),
      if (voteEnd != null) 'vote_end': voteEnd.toUtc().toIso8601String(),
    };
    final res = await _client.dio.post('/api/v1/events', data: body);
    return Event.fromJson(res.data as Map<String, dynamic>);
  }

  // POST /events/:id/tracks -> 201 (track added) | 409 (already queued)
  Future<void> suggestTrack(
    String eventId, {
    required String externalId,
    required String title,
    required String artist,
  }) async {
    await _client.dio.post(
      '/api/v1/events/$eventId/tracks',
      data: {'external_id': externalId, 'title': title, 'artist': artist},
    );
  }

  // POST /events/:id/invites -> 201 (invited)
  Future<void> inviteUser(String eventId, String userId) async {
    await _client.dio.post(
      '/api/v1/events/$eventId/invites',
      data: {'user_id': userId},
    );
  }
}

final eventsApiProvider = Provider<EventsApi>(
  (ref) => EventsApi(ref.watch(apiClientProvider)),
);
