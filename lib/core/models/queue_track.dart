class QueueTrack {
  final String id;
  final String externalId; // Deezer id, used to resolve the playback preview
  final String name;
  final String artist;
  final int votes;

  const QueueTrack({
    required this.id,
    required this.externalId,
    required this.name,
    required this.artist,
    required this.votes,
  });

  factory QueueTrack.fromJson(Map<String, dynamic> json) => QueueTrack(
        id: json['id'] as String,
        externalId: json['external_id'] as String? ?? '',
        name: json['title'] as String,
        artist: json['artist'] as String,
        votes: (json['vote_count'] as num).toInt(),
      );

  QueueTrack withVotes(int v) => QueueTrack(
        id: id,
        externalId: externalId,
        name: name,
        artist: artist,
        votes: v,
      );
}
