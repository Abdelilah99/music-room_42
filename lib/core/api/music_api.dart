import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:music_room/core/api/api_client.dart';
import 'package:music_room/core/models/music_track.dart';

class MusicApi {
  final ApiClient _client;

  MusicApi(this._client);

  // GET /music/search?q=<query> -> array or {"tracks": [...]}
  Future<List<MusicTrack>> search(String query) async {
    final res = await _client.dio.get(
      '/api/v1/music/search',
      queryParameters: {'q': query},
    );
    final raw = res.data;
    final list = raw is List
        ? raw
        : (raw as Map<String, dynamic>)['tracks'] as List<dynamic>? ?? [];
    return list
        .map((e) => MusicTrack.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  // GET /music/tracks/:id -> MusicTrack with a fresh preview_url for playback.
  Future<MusicTrack> getTrack(String externalId) async {
    final res = await _client.dio.get('/api/v1/music/tracks/$externalId');
    return MusicTrack.fromJson(res.data as Map<String, dynamic>);
  }
}

final musicApiProvider = Provider<MusicApi>(
  (ref) => MusicApi(ref.watch(apiClientProvider)),
);
