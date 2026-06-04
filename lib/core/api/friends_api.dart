import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../models/friendship.dart';
import '../models/user.dart';
import 'api_client.dart';

class FriendsApi {
  final ApiClient _client;

  FriendsApi(this._client);

  Future<FriendsData> getFriends() async {
    final responses = await Future.wait([
      _client.dio.get('/api/v1/friends'),
      _client.dio.get('/api/v1/friends/requests'),
      _client.dio.get('/api/v1/friends/outgoing'),
    ]);

    List<T> parse<T>(dynamic res, String key, T Function(Map<String, dynamic>) f) =>
        ((res.data[key] as List<dynamic>?) ?? [])
            .map((e) => f(e as Map<String, dynamic>))
            .toList();

    return FriendsData(
      accepted: parse(responses[0], 'friends', Friend.fromEntry),
      incoming: parse(responses[1], 'requests', FriendRequest.fromEntry),
      outgoing: parse(responses[2], 'requests', FriendRequest.fromEntry),
    );
  }

  Future<List<User>> searchUsers(String q) async {
    final res = await _client.dio
        .get('/api/v1/users/search', queryParameters: {'q': q});
    final list = res.data['users'] as List<dynamic>? ?? [];
    return list.map((e) => User.fromJson(e as Map<String, dynamic>)).toList();
  }

  Future<void> sendRequest(String userId) async {
    await _client.dio
        .post('/api/v1/friends/request', data: {'addressee_id': userId});
  }

  Future<void> accept(String friendshipId) async {
    await _client.dio.post('/api/v1/friends/accept/$friendshipId');
  }

  // Covers reject (incoming), cancel (outgoing), and unfriend (accepted).
  Future<void> remove(String friendshipId) async {
    await _client.dio.delete('/api/v1/friends/$friendshipId');
  }
}

final friendsApiProvider = Provider<FriendsApi>(
  (ref) => FriendsApi(ref.watch(apiClientProvider)),
);
