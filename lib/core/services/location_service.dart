import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:geolocator/geolocator.dart';

class LocationResult {
  final double lat;
  final double lng;
  const LocationResult(this.lat, this.lng);
}

/// Thin wrapper over geolocator. Returns null when location is unavailable
/// (service off or permission denied) so callers can fall back to manual input.
class LocationService {
  const LocationService();

  Future<LocationResult?> currentPosition() async {
    if (!await Geolocator.isLocationServiceEnabled()) {
      return null;
    }

    var permission = await Geolocator.checkPermission();
    if (permission == LocationPermission.denied) {
      permission = await Geolocator.requestPermission();
    }
    if (permission == LocationPermission.denied ||
        permission == LocationPermission.deniedForever) {
      return null;
    }

    final pos = await Geolocator.getCurrentPosition(
      locationSettings: const LocationSettings(accuracy: LocationAccuracy.high),
    );
    return LocationResult(pos.latitude, pos.longitude);
  }
}

final locationServiceProvider = Provider<LocationService>((_) => const LocationService());
