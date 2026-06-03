import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/api/friends_api.dart';
import '../../core/models/friendship.dart';
import '../../core/models/user.dart';

final friendsProvider =
    AsyncNotifierProvider<FriendsNotifier, FriendsData>(FriendsNotifier.new);

class FriendsNotifier extends AsyncNotifier<FriendsData> {
  FriendsApi get _api => ref.read(friendsApiProvider);

  @override
  Future<FriendsData> build() => _api.getFriends();

  Future<void> sendRequest(String userId) async {
    await _api.sendRequest(userId);
    ref.invalidateSelf();
  }

  Future<void> accept(String requestId) async {
    final old = state.valueOrNull;
    if (old == null) return;

    final req = old.incoming.firstWhere((r) => r.id == requestId);
    state = AsyncData(old.copyWith(
      incoming: old.incoming.where((r) => r.id != requestId).toList(),
      accepted: [...old.accepted, Friend(friendshipId: requestId, user: req.user)],
    ));

    try {
      await _api.accept(requestId);
    } catch (_) {
      state = AsyncData(old);
      rethrow;
    }
  }

  Future<void> reject(String requestId) async {
    final old = state.valueOrNull;
    if (old == null) return;

    state = AsyncData(old.copyWith(
      incoming: old.incoming.where((r) => r.id != requestId).toList(),
    ));

    try {
      await _api.remove(requestId);
    } catch (_) {
      state = AsyncData(old);
      rethrow;
    }
  }

  Future<void> cancelRequest(String requestId) async {
    final old = state.valueOrNull;
    if (old == null) return;

    state = AsyncData(old.copyWith(
      outgoing: old.outgoing.where((r) => r.id != requestId).toList(),
    ));

    try {
      await _api.remove(requestId);
    } catch (_) {
      state = AsyncData(old);
      rethrow;
    }
  }

  Future<void> unfriend(String friendshipId) async {
    final old = state.valueOrNull;
    if (old == null) return;

    state = AsyncData(old.copyWith(
      accepted: old.accepted.where((f) => f.friendshipId != friendshipId).toList(),
    ));

    try {
      await _api.remove(friendshipId);
    } catch (_) {
      state = AsyncData(old);
      rethrow;
    }
  }
}

// Family provider for user search — keyed by query string.
final userSearchProvider = FutureProvider.family<List<User>, String>((ref, query) async {
  if (query.trim().isEmpty) return [];
  return ref.read(friendsApiProvider).searchUsers(query);
});
