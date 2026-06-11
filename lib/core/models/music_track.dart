class MusicTrack {
  final String externalId;
  final String title;
  final String artist;
  final String? coverUrl;
  final String? previewUrl;
  final String? album;

  const MusicTrack({
    required this.externalId,
    required this.title,
    required this.artist,
    this.coverUrl,
    this.previewUrl,
    this.album,
  });

  factory MusicTrack.fromJson(Map<String, dynamic> json) => MusicTrack(
        externalId: json['external_id'] as String,
        title: json['title'] as String,
        artist: json['artist'] as String,
        coverUrl: json['cover_url'] as String?,
        previewUrl: json['preview_url'] as String?,
        album: json['album'] as String?,
      );
}
