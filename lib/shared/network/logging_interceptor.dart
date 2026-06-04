import 'dart:io';
import 'package:dio/dio.dart';
import 'package:device_info_plus/device_info_plus.dart';
import 'package:package_info_plus/package_info_plus.dart';

class LoggingInterceptor extends Interceptor {
  final DeviceInfoPlugin deviceInfo = DeviceInfoPlugin();

  @override
  void onRequest(RequestOptions options, RequestInterceptorHandler handler) async {
    // 1. Read app specification version string from platform metadata
    final PackageInfo packageInfo = await PackageInfo.fromPlatform();
    final String appVersion = packageInfo.version;

    // 2. Identify Host Platform Operating System 
    final String platform = Platform.isAndroid ? 'android' : 'ios';

    // 3. Extract exact Hardware Device Model safely
    String deviceModel = 'unknown';
    try {
      if (Platform.isAndroid) {
        final AndroidDeviceInfo androidInfo = await deviceInfo.androidInfo;
        deviceModel = androidInfo.model;
      } else if (Platform.isIOS) {
        final IosDeviceInfo iosInfo = await deviceInfo.iosInfo;
        deviceModel = iosInfo.utsname.machine;
      }
    } catch (_) {
      // Gracefully fall back if simulator capabilities restrict lookups
    }

    // 4. Inject metadata tracking requirements into headers
    options.headers['X-Platform'] = platform;
    options.headers['X-Device-Model'] = deviceModel;
    options.headers['X-App-Version'] = appVersion;

    return handler.next(options);
  }
}
