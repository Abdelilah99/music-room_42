import 'dart:io';
import 'package:dio/dio.dart';
import 'package:music_room/core/services/device_info_service.dart';

class LoggingInterceptor extends Interceptor {
  @override
  void onRequest(RequestOptions options, RequestInterceptorHandler handler) {
    // Inject metadata synchronously without adding per-request overhead
    options.headers['X-Platform'] = Platform.isAndroid ? 'android' : 'ios';
    options.headers['X-Device-Model'] = DeviceInfoService.deviceName;
    options.headers['X-App-Version'] = DeviceInfoService.appVersion;
    
    handler.next(options);
  }
}
