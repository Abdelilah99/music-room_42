// Mirrors the server Playlist (server/internal/model/playlist.go).
class Playlist {
  final String id;
  final String ownerId;
  final String name;
  final String visibility; // "public" or "private"
  final int license; // 0 = anyone edits, 1 = invited only
  final DateTime? createdAt;

  const Playlist({
    required this.id,
    required this.ownerId,
    required this.name,
    required this.visibility,
    required this.license,
    this.createdAt,
  });

  bool get isPublic => visibility == 'public';

  // License 0 lets anyone with access edit; license 1 limits editing to the
  // owner and invited collaborators.
  bool get isOpenLicense => license == 0;

  factory Playlist.fromJson(Map<String, dynamic> json) {
    DateTime? toDate(dynamic v) => v is String ? DateTime.tryParse(v) : null;

    return Playlist(
      id: json['id'] as String,
      ownerId: json['owner_id'] as String,
      name: json['name'] as String,
      visibility: json['visibility'] as String,
      license: (json['license'] as num).toInt(),
      createdAt: toDate(json['created_at']),
    );
  }
}

// Mirrors server PlaylistTrack (server/internal/model/playlist.go).
class PlaylistTrack {
  final String id;
  final String playlistId;
  final String externalId;
  final String title;
  final String artist;
  final int position; // 1-based
  final String? addedBy;
  final DateTime? createdAt;

  const PlaylistTrack({
    required this.id,
    required this.playlistId,
    required this.externalId,
    required this.title,
    required this.artist,
    required this.position,
    this.addedBy,
    this.createdAt,
  });

  factory PlaylistTrack.fromJson(Map<String, dynamic> json) {
    DateTime? toDate(dynamic v) => v is String ? DateTime.tryParse(v) : null;
    return PlaylistTrack(
      id: json['id'] as String,
      playlistId: json['playlist_id'] as String,
      externalId: json['external_id'] as String,
      title: json['title'] as String,
      artist: json['artist'] as String,
      position: (json['position'] as num).toInt(),
      addedBy: json['added_by'] as String?,
      createdAt: toDate(json['created_at']),
    );
  }

  PlaylistTrack withPosition(int newPosition) => PlaylistTrack(
        id: id,
        playlistId: playlistId,
        externalId: externalId,
        title: title,
        artist: artist,
        position: newPosition,
        addedBy: addedBy,
        createdAt: createdAt,
      );
}

// Mirrors server PlaylistWithTracks: Playlist fields + tracks array at top level.
class PlaylistWithTracks {
  final Playlist playlist;
  final List<PlaylistTrack> tracks;
  // Whether the current user may edit (add/remove/reorder) this playlist.
  // Server-computed: owner, license-0, or an invited editor of a license-1.
  final bool canEdit;

  const PlaylistWithTracks({
    required this.playlist,
    required this.tracks,
    this.canEdit = false,
  });

  factory PlaylistWithTracks.fromJson(Map<String, dynamic> json) {
    return PlaylistWithTracks(
      playlist: Playlist.fromJson(json),
      tracks: (json['tracks'] as List<dynamic>? ?? [])
          .map((t) => PlaylistTrack.fromJson(t as Map<String, dynamic>))
          .toList(),
      canEdit: json['can_edit'] as bool? ?? false,
    );
  }

  PlaylistWithTracks copyWithTracks(List<PlaylistTrack> tracks) =>
      PlaylistWithTracks(playlist: playlist, tracks: tracks, canEdit: canEdit);
}
