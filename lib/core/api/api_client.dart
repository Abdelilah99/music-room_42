import 'dart:async';

import 'package:dio/dio.dart';
import 'package:flutter_dotenv/flutter_dotenv.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:music_room/core/services/token_storage.dart';
import 'package:music_room/features/auth/auth_provider.dart';

class _AuthInterceptor extends Interceptor {
  final TokenStorage _storage;
  final Dio _dio;
  final Future<void> Function() _onAuthExpired;

  // Serialises concurrent 401 responses: only one refresh attempt runs at a
  // time; other failing requests wait on the same Completer.
  Completer<String?>? _pendingRefresh;

  _AuthInterceptor({
    required TokenStorage storage,
    required Dio dio,
    required Future<void> Function() onAuthExpired,
  })  : _storage = storage,
        _dio = dio,
        _onAuthExpired = onAuthExpired;

  @override
  void onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) async {
    // Requests marked _retry already have the updated header set.
    if (options.extra['_retry'] != true) {
      final token = await _storage.getAccessToken();
      if (token != null) {
        options.headers['Authorization'] = 'Bearer $token';
      }
    }
    handler.next(options);
  }

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) async {
    final options = err.requestOptions;

    // Only intercept 401s that haven't already been through one retry cycle.
    if (err.response?.statusCode != 401 || options.extra['_retry'] == true) {
      handler.next(err);
      return;
    }

    // Another refresh is already in flight — queue behind it.
    if (_pendingRefresh != null) {
      final newToken = await _pendingRefresh!.future;
      if (newToken == null) {
        handler.next(err);
        return;
      }
      options.headers['Authorization'] = 'Bearer $newToken';
      options.extra['_retry'] = true;
      try {
        handler.resolve(await _dio.fetch(options));
      } catch (_) {
        handler.next(err);
      }
      return;
    }

    final completer = Completer<String?>();
    _pendingRefresh = completer;

    try {
      final refreshToken = await _storage.getRefreshToken();
      if (refreshToken == null) throw Exception('no stored refresh token');

      final res = await _dio.post(
        '/api/v1/auth/refresh',
        data: {'refresh_token': refreshToken},
        // Mark so onError does not try to refresh the refresh request itself.
        options: Options(extra: {'_retry': true}),
      );

      final newAccess = res.data['access_token'] as String;
      final newRefresh = res.data['refresh_token'] as String;
      await _storage.saveTokens(access: newAccess, refresh: newRefresh);

      completer.complete(newAccess);

      options.headers['Authorization'] = 'Bearer $newAccess';
      options.extra['_retry'] = true;
      handler.resolve(await _dio.fetch(options));
    } catch (_) {
      completer.complete(null);
      await _storage.clearTokens();
      await _onAuthExpired();
      handler.next(err);
    } finally {
      _pendingRefresh = null;
    }
  }
}

class ApiClient {
  late final Dio dio;

  ApiClient._({
    required String baseUrl,
    required TokenStorage tokenStorage,
    required Future<void> Function() onAuthExpired,
  }) {
    dio = Dio(BaseOptions(baseUrl: baseUrl));
    dio.interceptors.add(_AuthInterceptor(
      storage: tokenStorage,
      dio: dio,
      onAuthExpired: onAuthExpired,
    ));
  }
}

final apiClientProvider = Provider<ApiClient>((ref) {
  return ApiClient._(
    baseUrl: dotenv.env['API_BASE_URL'] ?? '',
    tokenStorage: ref.read(tokenStorageProvider),
    // Lazy callback: not evaluated at construction time, so no circular
    // dependency between apiClientProvider and authProvider.
    onAuthExpired: () =>
        ref.read(authProvider.notifier).forceUnauthenticated(),
  );
});
