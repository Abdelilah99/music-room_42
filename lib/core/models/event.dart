// Mirrors the server Event (server/internal/model/event.go).
class Event {
  final String id;
  final String ownerId;
  final String name;
  final String visibility; // "public" or "private"
  final double? lat;
  final double? lng;
  final double? radius; // km, only meaningful for license 2
  final DateTime? voteStart;
  final DateTime? voteEnd;
  final int license; // 0 open, 1 invite-only, 2 geofence + time window
  final DateTime? createdAt;

  const Event({
    required this.id,
    required this.ownerId,
    required this.name,
    required this.visibility,
    this.lat,
    this.lng,
    this.radius,
    this.voteStart,
    this.voteEnd,
    required this.license,
    this.createdAt,
  });

  bool get isPublic => visibility == 'public';

  factory Event.fromJson(Map<String, dynamic> json) {
    double? toDouble(dynamic v) => v == null ? null : (v as num).toDouble();
    DateTime? toDate(dynamic v) =>
        v is String ? DateTime.tryParse(v) : null;

    return Event(
      id: json['id'] as String,
      ownerId: json['owner_id'] as String,
      name: json['name'] as String,
      visibility: json['visibility'] as String,
      lat: toDouble(json['lat']),
      lng: toDouble(json['lng']),
      radius: toDouble(json['radius']),
      voteStart: toDate(json['vote_start']),
      voteEnd: toDate(json['vote_end']),
      license: (json['license'] as num).toInt(),
      createdAt: toDate(json['created_at']),
    );
  }
}
