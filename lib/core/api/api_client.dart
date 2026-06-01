import 'package:dio/dio.dart';
import 'package:flutter_dotenv/flutter_dotenv.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class ApiClient {
  late final Dio dio;

  ApiClient() {
    final baseUrl = dotenv.env['API_BASE_URL'] ?? '';
    dio = Dio(BaseOptions(baseUrl: baseUrl));
  }
}

final apiClientProvider = Provider<ApiClient>((ref) => ApiClient());
