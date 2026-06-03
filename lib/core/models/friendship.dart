import 'user.dart';

// An accepted friend — we keep friendshipId for unfriend calls.
class Friend {
  final String friendshipId;
  final User user;

  const Friend({required this.friendshipId, required this.user});

  // Parses a server FriendEntry: {friendship_id, user_id, email, public_info}
  factory Friend.fromEntry(Map<String, dynamic> json) {
    final pub = json['public_info'] as Map<String, dynamic>?;
    return Friend(
      friendshipId: json['friendship_id'] as String,
      user: User(
        id: json['user_id'] as String,
        email: json['email'] as String,
        name: pub?['display_name'] as String?,
      ),
    );
  }
}

// A pending friend request (incoming or outgoing).
class FriendRequest {
  final String id;
  final User user;

  const FriendRequest({required this.id, required this.user});

  // Parses a server FriendEntry: {friendship_id, user_id, email, public_info}
  factory FriendRequest.fromEntry(Map<String, dynamic> json) {
    final pub = json['public_info'] as Map<String, dynamic>?;
    return FriendRequest(
      id: json['friendship_id'] as String,
      user: User(
        id: json['user_id'] as String,
        email: json['email'] as String,
        name: pub?['display_name'] as String?,
      ),
    );
  }
}

class FriendsData {
  final List<Friend> accepted;
  final List<FriendRequest> incoming;
  final List<FriendRequest> outgoing;

  const FriendsData({
    this.accepted = const [],
    this.incoming = const [],
    this.outgoing = const [],
  });

  FriendsData copyWith({
    List<Friend>? accepted,
    List<FriendRequest>? incoming,
    List<FriendRequest>? outgoing,
  }) =>
      FriendsData(
        accepted: accepted ?? this.accepted,
        incoming: incoming ?? this.incoming,
        outgoing: outgoing ?? this.outgoing,
      );
}
