import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:just_audio/just_audio.dart';
import 'package:music_room/core/api/music_api.dart';
import 'package:music_room/core/models/queue_track.dart';
import 'package:music_room/features/track_vote/event_queue_provider.dart';

// Snapshot of the now-playing state for an event.
class EventPlayerState {
  final String? trackId;
  final String? title;
  final String? artist;
  final bool playing;
  final bool loading;

  const EventPlayerState({
    this.trackId,
    this.title,
    this.artist,
    this.playing = false,
    this.loading = false,
  });

  bool get isActive => trackId != null;

  EventPlayerState copyWith({
    String? trackId,
    String? title,
    String? artist,
    bool? playing,
    bool? loading,
  }) =>
      EventPlayerState(
        trackId: trackId ?? this.trackId,
        title: title ?? this.title,
        artist: artist ?? this.artist,
        playing: playing ?? this.playing,
        loading: loading ?? this.loading,
      );
}

// Plays the event queue as a live chain: always the highest-voted track that
// hasn't played yet, advancing automatically when a preview ends. Preview URLs
// are resolved from Deezer just before playing (they are short-lived).
class EventPlayerNotifier extends Notifier<EventPlayerState> {
  EventPlayerNotifier(this._eventId);
  final String _eventId;

  final AudioPlayer _player = AudioPlayer();
  final Set<String> _playedIds = {};
  // Kept in sync with the queue provider so auto-advance always picks from the
  // current, vote-sorted order rather than a stale read.
  List<QueueTrack> _liveQueue = const [];

  @override
  EventPlayerState build() {
    _liveQueue = ref.read(eventQueueProvider(_eventId)).value ?? const [];
    ref.listen(eventQueueProvider(_eventId), (_, next) {
      _liveQueue = next.value ?? const [];
    });

    final sub = _player.processingStateStream.listen((s) {
      if (s == ProcessingState.completed) _onCompleted();
    });
    ref.onDispose(() {
      sub.cancel();
      _player.dispose();
    });
    return const EventPlayerState();
  }

  // The next track to play: highest-voted one not yet played this session.
  QueueTrack? _nextTrack() {
    for (final t in _liveQueue) {
      if (!_playedIds.contains(t.id)) return t;
    }
    return null;
  }

  // Play/pause button entry point.
  Future<void> toggle() async {
    if (state.loading) return;
    if (state.playing) {
      await _player.pause();
      state = state.copyWith(playing: false);
      return;
    }
    if (state.trackId != null) {
      // play() resolves only at end-of-track, so it must not be awaited.
      state = state.copyWith(playing: true);
      unawaited(_player.play());
      return;
    }
    await _playNext();
  }

  // Skip the current track.
  Future<void> skip() async {
    final current = state.trackId;
    if (current != null) _playedIds.add(current);
    await _playNext();
  }

  Future<void> _playNext() async {
    final track = _nextTrack();
    if (track == null) {
      await _player.stop();
      state = const EventPlayerState();
      return;
    }
    await _play(track);
  }

  Future<void> _play(QueueTrack track) async {
    state = EventPlayerState(
      trackId: track.id,
      title: track.name,
      artist: track.artist,
      loading: true,
    );
    try {
      final resolved =
          await ref.read(musicApiProvider).getTrack(track.externalId);
      final url = resolved.previewUrl;
      if (url == null || url.isEmpty) {
        // No preview for this track; mark it played and move on.
        _playedIds.add(track.id);
        await _playNext();
        return;
      }
      await _player.setUrl(url);
      // play() resolves only at end-of-track, so start it without awaiting and
      // mark the state playing immediately; auto-advance happens via the
      // processingState 'completed' listener.
      state = state.copyWith(loading: false, playing: true);
      unawaited(_player.play());
    } catch (_) {
      // Resolution or playback failed; skip this track and try the next.
      _playedIds.add(track.id);
      await _playNext();
    }
  }

  void _onCompleted() {
    final current = state.trackId;
    if (current != null) _playedIds.add(current);
    _playNext();
  }
}

final eventPlayerProvider =
    NotifierProvider.autoDispose.family<EventPlayerNotifier, EventPlayerState, String>(
  (eventId) => EventPlayerNotifier(eventId),
);
