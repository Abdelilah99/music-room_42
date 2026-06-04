import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:music_room/core/api/web_socket_service.dart';
import 'package:music_room/core/widgets/ws_shell.dart';

class DelegationScreen extends ConsumerWidget {
  const DelegationScreen({super.key});

  static const _hubPath = '/api/v1/ws/delegation';

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final connState = ref.watch(wsProvider(_hubPath));

    return WsShell(
      title: 'Delegation',
      state: connState,
      onRetry: () => ref.read(wsProvider(_hubPath).notifier).reconnect(),
      child: const Center(child: Text('Delegation')),
    );
  }
}
