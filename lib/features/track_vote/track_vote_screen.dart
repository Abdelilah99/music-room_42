import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:music_room/core/api/web_socket_service.dart';
import 'package:music_room/core/widgets/ws_shell.dart';

class TrackVoteScreen extends ConsumerWidget {
  const TrackVoteScreen({super.key});

  static const _hubPath = '/api/v1/ws/track-vote';

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final connState = ref.watch(wsProvider(_hubPath));

    return WsShell(
      title: 'Track Vote',
      state: connState,
      onRetry: () => ref.read(wsProvider(_hubPath).notifier).reconnect(),
      // TODO: replace with a real event list; this button is for manual testing only
      child: Center(
        child: FilledButton.icon(
          onPressed: () => context.push('/events/test-event-1'),
          icon: const Icon(Icons.queue_music_outlined),
          label: const Text('Open Event Queue'),
        ),
      ),
    );
  }
}
