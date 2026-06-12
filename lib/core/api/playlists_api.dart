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

  // GET /playlists/:id -> PlaylistWithTracks
  Future<PlaylistWithTracks> get(String playlistId) async {
    final res = await _client.dio.get('/api/v1/playlists/$playlistId');
    return PlaylistWithTracks.fromJson(res.data as Map<String, dynamic>);
  }

  // POST /playlists/:id/tracks -> 201 PlaylistTrack
  Future<PlaylistTrack> addTrack(
    String playlistId, {
    required String externalId,
    required String title,
    required String artist,
  }) async {
    final res = await _client.dio.post(
      '/api/v1/playlists/$playlistId/tracks',
      data: {'external_id': externalId, 'title': title, 'artist': artist},
    );
    return PlaylistTrack.fromJson(res.data as Map<String, dynamic>);
  }

  // DELETE /playlists/:id/tracks/:trackId -> 200
  Future<void> removeTrack(String playlistId, String trackId) async {
    await _client.dio
        .delete('/api/v1/playlists/$playlistId/tracks/$trackId');
  }

  // PATCH /playlists/:id/tracks/:trackId/position -> 200
  Future<void> moveTrack(
    String playlistId,
    String trackId, {
    required int position,
  }) async {
    await _client.dio.patch(
      '/api/v1/playlists/$playlistId/tracks/$trackId/position',
      data: {'position': position},
    );
  }

  // POST /playlists/:id/invites -> 201
  Future<void> inviteUser(String playlistId, String userId) async {
    await _client.dio.post(
      '/api/v1/playlists/$playlistId/invites',
      data: {'user_id': userId},
    );
  }
}

final playlistsApiProvider = Provider<PlaylistsApi>(
  (ref) => PlaylistsApi(ref.watch(apiClientProvider)),
);
