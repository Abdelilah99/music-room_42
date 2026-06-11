import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:music_room/core/api/events_api.dart';
import 'package:music_room/core/api/profile_api.dart';
import 'package:music_room/core/models/event.dart';

final eventsProvider =
    AsyncNotifierProvider<EventsNotifier, List<Event>>(EventsNotifier.new);

class EventsNotifier extends AsyncNotifier<List<Event>> {
  String _query = '';

  EventsApi get _api => ref.read(eventsApiProvider);

  @override
  Future<List<Event>> build() => _api.list(query: _query);

  // Re-runs the list with a new search term. Keeps the previous results
  // on screen while the request is in flight (no spinner flash on each keystroke).
  Future<void> search(String query) async {
    _query = query;
    state = await AsyncValue.guard(() => _api.list(query: _query));
  }

  Future<void> refresh() async {
    state = await AsyncValue.guard(() => _api.list(query: _query));
  }
}

// Fetches a single event by id. autoDispose so navigating away re-fetches on return.
final eventByIdProvider = FutureProvider.autoDispose.family<Event, String>(
  (ref, eventId) => ref.read(eventsApiProvider).get(eventId),
);

// Resolves an owner's display name from their id. Riverpod caches per id, so
// each distinct owner in the list is fetched only once. Falls back to a short
// id fragment if the profile cannot be loaded.
final ownerNameProvider =
    FutureProvider.family<String, String>((ref, ownerId) async {
  try {
    final profile = await ref.read(profileApiProvider).getUserProfile(ownerId);
    return profile.displayName;
  } catch (_) {
    return ownerId.length >= 8 ? ownerId.substring(0, 8) : ownerId;
  }
});
