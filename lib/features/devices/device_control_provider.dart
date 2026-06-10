import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:music_room/core/api/devices_api.dart';

// Local view of a device's playback, kept in sync from two directions:
// the controls on this screen (optimistic on send) and, for the owner, the
// `command` messages arriving over the device WebSocket.
class PlaybackState {
  final bool isPlaying;
  final int volume; // 0-100
  final int trackPosition; // 1-based index into the local player
  final bool inFlight; // a command request is in progress

  const PlaybackState({
    this.isPlaying = false,
    this.volume = 50,
    this.trackPosition = 1,
    this.inFlight = false,
  });

  PlaybackState copyWith({
    bool? isPlaying,
    int? volume,
    int? trackPosition,
    bool? inFlight,
  }) {
    return PlaybackState(
      isPlaying: isPlaying ?? this.isPlaying,
      volume: volume ?? this.volume,
      trackPosition: trackPosition ?? this.trackPosition,
      inFlight: inFlight ?? this.inFlight,
    );
  }
}

class DeviceControlNotifier extends Notifier<PlaybackState> {
  DeviceControlNotifier(this._deviceId);

  final String _deviceId;

  @override
  PlaybackState build() => const PlaybackState();

  // Applies a command to the local state without any network call. Used both
  // for optimistic updates on send and for commands received over the socket.
  void _apply(String action, int? value) {
    switch (action) {
      case 'play':
        state = state.copyWith(isPlaying: true);
      case 'pause':
        state = state.copyWith(isPlaying: false);
      case 'next':
        state = state.copyWith(trackPosition: state.trackPosition + 1);
      case 'volume':
        if (value != null) {
          state = state.copyWith(volume: value.clamp(0, 100));
        }
    }
  }

  // Handles a `command` message received over the device WebSocket (owner side).
  void applyRemote(String action, int? value) => _apply(action, value);

  // Moves the slider locally while dragging, without sending anything. The
  // command is sent once on drag end via [sendCommand].
  void setVolumeLocal(int value) {
    state = state.copyWith(volume: value.clamp(0, 100));
  }

  // Sends a command to the device. Returns null on success or a human-readable
  // message on failure. A second call while one is in flight is ignored so a
  // double tap cannot double-send.
  //
  // The owner does NOT apply the command locally here: the server broadcasts it
  // back over the device WebSocket and [applyRemote] applies it, which is the
  // single source of truth. Applying it here too would double-count a relative
  // action like "next". A caller with no socket (a delegate) passes
  // applyLocally: true so its controls still reflect what it sent.
  Future<String?> sendCommand(
    String action, {
    int? value,
    bool applyLocally = false,
  }) async {
    if (state.inFlight) return null;

    state = state.copyWith(inFlight: true);
    try {
      await ref
          .read(devicesApiProvider)
          .sendCommand(_deviceId, action: action, value: value);
      if (applyLocally) _apply(action, value);
      state = state.copyWith(inFlight: false);
      return null;
    } on DioException catch (e) {
      state = state.copyWith(inFlight: false);
      final data = e.response?.data;
      final msg = data is Map ? data['error'] as String? : null;
      return msg ?? 'Could not send the command';
    }
  }
}

final deviceControlProvider = NotifierProvider.autoDispose
    .family<DeviceControlNotifier, PlaybackState, String>(
  DeviceControlNotifier.new,
);
