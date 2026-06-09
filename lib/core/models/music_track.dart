class MusicTrack {
  final String externalId;
  final String title;
  final String artist;
  final String? coverUrl;

  const MusicTrack({
    required this.externalId,
    required this.title,
    required this.artist,
    this.coverUrl,
  });

  factory MusicTrack.fromJson(Map<String, dynamic> json) => MusicTrack(
        externalId: json['external_id'] as String,
        title: json['title'] as String,
        artist: json['artist'] as String,
        coverUrl: json['cover_url'] as String?,
      );
}
