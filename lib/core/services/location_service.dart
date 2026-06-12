import 'package:geolocator/geolocator.dart';

enum GpsFailure { serviceDisabled, permissionDenied, permissionDeniedForever }

sealed class GpsResult {}

final class GpsSuccess extends GpsResult {
  final double lat;
  final double lng;
  GpsSuccess(this.lat, this.lng);
}

final class GpsError extends GpsResult {
  final GpsFailure reason;
  GpsError(this.reason);
}

class LocationResult {
  final double lat;
  final double lng;
  const LocationResult(this.lat, this.lng);
}

/// Thin wrapper over geolocator. Use [currentPositionDetailed] for a typed
/// result with a failure reason; [currentPosition] for a nullable fallback.
class LocationService {
  const LocationService();

  /// Returns null when location is unavailable. Prefer [currentPositionDetailed]
  /// when the caller needs to explain why location failed.
  Future<LocationResult?> currentPosition() async {
    final result = await currentPositionDetailed();
    if (result is GpsSuccess) return LocationResult(result.lat, result.lng);
    return null;
  }

  /// Returns [GpsSuccess] or [GpsError] with a [GpsFailure] reason.
  Future<GpsResult> currentPositionDetailed() async {
    if (!await Geolocator.isLocationServiceEnabled()) {
      return GpsError(GpsFailure.serviceDisabled);
    }

    var permission = await Geolocator.checkPermission();
    if (permission == LocationPermission.denied) {
      permission = await Geolocator.requestPermission();
    }
    if (permission == LocationPermission.deniedForever) {
      return GpsError(GpsFailure.permissionDeniedForever);
    }
    if (permission == LocationPermission.denied) {
      return GpsError(GpsFailure.permissionDenied);
    }

    final pos = await Geolocator.getCurrentPosition(
      locationSettings: const LocationSettings(accuracy: LocationAccuracy.high),
    );
    return GpsSuccess(pos.latitude, pos.longitude);
  }
}
