import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:music_room/core/api/playlists_api.dart';
import 'package:music_room/core/models/playlist.dart';

class PlaylistDetailNotifier extends AsyncNotifier<PlaylistWithTracks> {
  PlaylistDetailNotifier(this._playlistId);
  final String _playlistId;

  PlaylistsApi get _api => ref.read(playlistsApiProvider);

  @override
  Future<PlaylistWithTracks> build() => _api.get(_playlistId);

  // Dedupes by id so our own add-echo doesn't double-insert the track.
  void applyTrackAdded(PlaylistTrack track) {
    final current = state.value;
    if (current == null) return;
    if (current.tracks.any((t) => t.id == track.id)) return;
    state = AsyncData(current.copyWithTracks([...current.tracks, track]));
  }

  void applyTrackRemoved(String trackId) {
    final current = state.value;
    if (current == null) return;
    state = AsyncData(
      current.copyWithTracks(
        current.tracks.where((t) => t.id != trackId).toList(),
      ),
    );
  }

  // Removes the track, inserts it at the 1-based [position], then renumbers.
  void applyTrackMoved(String trackId, int position) {
    final current = state.value;
    if (current == null) return;
    var tracks = current.tracks.toList();
    final idx = tracks.indexWhere((t) => t.id == trackId);
    if (idx == -1) return;
    final track = tracks.removeAt(idx);
    final insertAt = (position - 1).clamp(0, tracks.length);
    tracks.insert(insertAt, track);
    tracks = [
      for (var i = 0; i < tracks.length; i++) tracks[i].withPosition(i + 1)
    ];
    state = AsyncData(current.copyWithTracks(tracks));
  }

  // Optimistic remove: applies locally then DELETEs on the server. Reverts on error.
  Future<void> removeTrack(String playlistId, String trackId) async {
    final snapshot = state.value;
    if (snapshot == null) return;
    applyTrackRemoved(trackId);
    try {
      await _api.removeTrack(playlistId, trackId);
    } catch (_) {
      state = AsyncData(snapshot);
      rethrow;
    }
  }

  // Optimistic move: applies locally then PATCHes the server. Reverts on error.
  Future<void> moveTrack(String trackId, int newPosition) async {
    final snapshot = state.value;
    if (snapshot == null) return;
    applyTrackMoved(trackId, newPosition);
    try {
      await _api.moveTrack(snapshot.playlist.id, trackId,
          position: newPosition);
    } catch (_) {
      state = AsyncData(snapshot);
      rethrow;
    }
  }
}

final playlistDetailProvider = AsyncNotifierProvider.autoDispose
    .family<PlaylistDetailNotifier, PlaylistWithTracks, String>(
  (playlistId) => PlaylistDetailNotifier(playlistId),
);
