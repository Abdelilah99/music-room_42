import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:music_room/core/api/playlists_api.dart';
import 'package:music_room/core/api/profile_api.dart';
import 'package:music_room/core/models/playlist.dart';

final playlistsProvider =
    AsyncNotifierProvider<PlaylistsNotifier, List<Playlist>>(
        PlaylistsNotifier.new);

class PlaylistsNotifier extends AsyncNotifier<List<Playlist>> {
  String _query = '';

  PlaylistsApi get _api => ref.read(playlistsApiProvider);

  @override
  Future<List<Playlist>> build() => _api.list(query: _query);

  // Re-runs the list with a new search term. Keeps the previous results on
  // screen while the request is in flight (no spinner flash on each keystroke).
  // The query guard drops a slow in-flight response if a newer search has
  // already been issued, so the last result shown always matches the last term.
  Future<void> search(String query) async {
    _query = query;
    final result = await AsyncValue.guard(() => _api.list(query: query));
    if (_query == query) state = result;
  }

  Future<void> refresh() async {
    state = await AsyncValue.guard(() => _api.list(query: _query));
  }
}

// Resolves a playlist owner's display name from their id. Riverpod caches per
// id, so each distinct owner in the list is fetched only once. Falls back to a
// short id fragment if the profile cannot be loaded.
final playlistOwnerNameProvider =
    FutureProvider.family<String, String>((ref, ownerId) async {
  try {
    final profile = await ref.read(profileApiProvider).getUserProfile(ownerId);
    return profile.displayName;
  } catch (_) {
    return ownerId.length >= 8 ? ownerId.substring(0, 8) : ownerId;
  }
});
