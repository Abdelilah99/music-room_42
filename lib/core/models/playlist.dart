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
