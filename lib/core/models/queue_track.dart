class QueueTrack {
  final String id;
  final String name;
  final String artist;
  final int votes;

  const QueueTrack({
    required this.id,
    required this.name,
    required this.artist,
    required this.votes,
  });

  factory QueueTrack.fromJson(Map<String, dynamic> json) => QueueTrack(
        id: json['id'] as String,
        name: json['name'] as String,
        artist: json['artist'] as String,
        votes: (json['votes'] as num).toInt(),
      );

  QueueTrack withVotes(int v) =>
      QueueTrack(id: id, name: name, artist: artist, votes: v);
}
