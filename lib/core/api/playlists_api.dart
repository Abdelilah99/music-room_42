import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:music_room/core/api/api_client.dart';
import 'package:music_room/core/models/playlist.dart';

class PlaylistsApi {
  final ApiClient _client;

  PlaylistsApi(this._client);

  // GET /playlists?q=<query> -> {"playlists": [...]}
  Future<List<Playlist>> list({String? query}) async {
    final res = await _client.dio.get(
      '/api/v1/playlists',
      queryParameters: (query != null && query.isNotEmpty) ? {'q': query} : null,
    );
    final raw = res.data['playlists'] as List<dynamic>? ?? [];
    return raw.map((p) => Playlist.fromJson(p as Map<String, dynamic>)).toList();
  }

  // POST /playlists -> 201 Playlist
  Future<Playlist> create({
    required String name,
    required String visibility,
    required int license,
  }) async {
    final res = await _client.dio.post('/api/v1/playlists', data: {
      'name': name,
      'visibility': visibility,
      'license': license,
    });
    return Playlist.fromJson(res.data as Map<String, dynamic>);
  }
}

final playlistsApiProvider = Provider<PlaylistsApi>(
  (ref) => PlaylistsApi(ref.watch(apiClientProvider)),
);
