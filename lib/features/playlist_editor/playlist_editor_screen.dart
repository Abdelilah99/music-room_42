import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:music_room/core/api/web_socket_service.dart';
import 'package:music_room/core/widgets/ws_shell.dart';

class PlaylistEditorScreen extends ConsumerWidget {
  const PlaylistEditorScreen({super.key});

  static const _hubPath = '/api/v1/ws/playlist';

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final connState = ref.watch(wsProvider(_hubPath));

    return WsShell(
      title: 'Playlist Editor',
      state: connState,
      onRetry: () => ref.read(wsProvider(_hubPath).notifier).reconnect(),
      child: const Center(child: Text('Playlist Editor')),
    );
  }
}
