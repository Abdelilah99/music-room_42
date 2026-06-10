import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:music_room/core/api/profile_api.dart';
import 'package:music_room/core/models/user_profile.dart';

class MyProfileNotifier extends AsyncNotifier<UserProfile> {
  @override
  Future<UserProfile> build() => ref.read(profileApiProvider).getMyProfile();

  Future<void> patchSection({
    PublicInfo? publicInfo,
    FriendsInfo? friendsInfo,
    PrivateInfo? privateInfo,
    MusicPreferences? musicPreferences,
  }) async {
    await ref.read(profileApiProvider).updateMyProfile(
          publicInfo: publicInfo,
          friendsInfo: friendsInfo,
          privateInfo: privateInfo,
          musicPreferences: musicPreferences,
        );
    ref.invalidateSelf();
    await future;
  }
}

final myProfileProvider =
    AsyncNotifierProvider.autoDispose<MyProfileNotifier, UserProfile>(
  MyProfileNotifier.new,
);

// autoDispose ensures a fresh fetch on every screen entry (navigate away & back re-fetches).
final userProfileProvider =
    FutureProvider.autoDispose.family<UserProfile, String>(
  (ref, userId) => ref.read(profileApiProvider).getUserProfile(userId),
);
