import 'package:dio/dio.dart';
import 'package:flutter_dotenv/flutter_dotenv.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

// Auth endpoints do not require a Bearer token, so this uses its own
// interceptor-free Dio instance to avoid a circular provider dependency.
class AuthApi {
  late final Dio _dio;

  AuthApi() {
    final baseUrl = dotenv.env['API_BASE_URL'] ?? '';
    _dio = Dio(BaseOptions(baseUrl: baseUrl));
  }

  Future<void> register({
    required String email,
    required String password,
  }) async {
    await _dio.post('/api/v1/auth/register', data: {
      'email': email,
      'password': password,
    });
  }

  Future<({String accessToken, String refreshToken})> login({
    required String email,
    required String password,
  }) async {
    final response = await _dio.post('/api/v1/auth/login', data: {
      'email': email,
      'password': password,
    });
    return (
      accessToken: response.data['access_token'] as String,
      refreshToken: response.data['refresh_token'] as String,
    );
  }

  Future<void> resendVerification({required String email}) async {
    await _dio.post('/api/v1/auth/resend-verification', data: {'email': email});
  }

  Future<void> forgotPassword({required String email}) async {
    await _dio.post('/api/v1/auth/forgot-password', data: {'email': email});
  }

  Future<({String accessToken, String refreshToken})> refresh({
    required String refreshToken,
  }) async {
    final response = await _dio.post('/api/v1/auth/refresh', data: {
      'refresh_token': refreshToken,
    });
    return (
      accessToken: response.data['access_token'] as String,
      refreshToken: response.data['refresh_token'] as String,
    );
  }

  Future<void> logout({required String refreshToken}) async {
    await _dio.post('/api/v1/auth/logout', data: {
      'refresh_token': refreshToken,
    });
  }
}

final authApiProvider = Provider<AuthApi>((ref) => AuthApi());
