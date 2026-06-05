import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
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
      child: const Center(child: Text('Track Vote')),
    );
  }
}
