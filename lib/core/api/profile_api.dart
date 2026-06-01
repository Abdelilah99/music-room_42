import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:music_room/core/api/api_client.dart';
import 'package:music_room/core/models/user_profile.dart';

class ProfileApi {
  final ApiClient _client;
  const ProfileApi(this._client);

  Future<UserProfile> getMyProfile() async {
    final response = await _client.dio.get('/api/v1/users/me');
    return UserProfile.fromJson(response.data as Map<String, dynamic>);
  }

  Future<void> updateMyProfile({
    PublicInfo? publicInfo,
    FriendsInfo? friendsInfo,
    PrivateInfo? privateInfo,
    MusicPreferences? musicPreferences,
  }) async {
    final body = <String, dynamic>{};
    if (publicInfo != null) body['public_info'] = publicInfo.toJson();
    if (friendsInfo != null) body['friends_info'] = friendsInfo.toJson();
    if (privateInfo != null) body['private_info'] = privateInfo.toJson();
    if (musicPreferences != null) {
      body['music_preferences'] = musicPreferences.toJson();
    }
    if (body.isEmpty) return;
    await _client.dio.patch('/api/v1/users/me', data: body);
  }

  Future<UserProfile> getUserProfile(String userId) async {
    final response = await _client.dio.get('/api/v1/users/$userId');
    return UserProfile.fromJson(response.data as Map<String, dynamic>);
  }
}

final profileApiProvider = Provider<ProfileApi>(
  (ref) => ProfileApi(ref.watch(apiClientProvider)),
);
