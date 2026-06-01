import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:music_room/core/api/auth_api.dart';
import 'package:music_room/core/services/token_storage.dart';

enum AuthStatus { authenticated, unauthenticated }

class AuthNotifier extends AsyncNotifier<AuthStatus> {
  @override
  Future<AuthStatus> build() async {
    final has = await ref.read(tokenStorageProvider).hasTokens();
    return has ? AuthStatus.authenticated : AuthStatus.unauthenticated;
  }

  Future<void> login({
    required String email,
    required String password,
  }) async {
    final result = await ref.read(authApiProvider).login(
          email: email,
          password: password,
        );
    await ref.read(tokenStorageProvider).saveTokens(
          access: result.accessToken,
          refresh: result.refreshToken,
        );
    state = const AsyncData(AuthStatus.authenticated);
  }

  Future<void> logout() async {
    final storage = ref.read(tokenStorageProvider);
    try {
      final rt = await storage.getRefreshToken();
      if (rt != null) {
        await ref.read(authApiProvider).logout(refreshToken: rt);
      }
    } catch (_) {}
    await storage.clearTokens();
    state = const AsyncData(AuthStatus.unauthenticated);
  }

  // Called by the auth interceptor when a token refresh attempt fails.
  void forceUnauthenticated() {
    ref.read(tokenStorageProvider).clearTokens();
    state = const AsyncData(AuthStatus.unauthenticated);
  }
}

final authProvider =
    AsyncNotifierProvider<AuthNotifier, AuthStatus>(AuthNotifier.new);
