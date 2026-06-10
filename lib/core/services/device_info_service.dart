import 'package:flutter/services.dart';
import 'package:package_info_plus/package_info_plus.dart';

// Call init() once at startup. All getters are synchronous after that.
class DeviceInfoService {
  static const _channel = MethodChannel('com.musicroom/device_info');

  static String _model = '';
  static String _manufacturer = '';
  static String _deviceName = '';
  static String _appVersion = '';

  static Future<void> init() async {
    try {
      final results = await Future.wait([
        _channel.invokeMethod<String>('getModel'),
        _channel.invokeMethod<String>('getManufacturer'),
        _channel.invokeMethod<String>('getDeviceName'),
        PackageInfo.fromPlatform(),
      ]);
      _model        = (results[0] as String?) ?? '';
      _manufacturer = (results[1] as String?) ?? '';
      _deviceName   = (results[2] as String?) ?? '';
      _appVersion   = (results[3] as PackageInfo).version;
    } catch (_) {
      // MethodChannel not available on this platform — getters return empty strings
    }
  }

  static String get model        => _model;
  static String get manufacturer => _manufacturer;
  static String get deviceName   => _deviceName;
  static String get appVersion   => _appVersion;
}
