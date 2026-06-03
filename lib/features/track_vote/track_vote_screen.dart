import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:music_room/core/api/web_socket_service.dart';
import 'package:music_room/core/services/token_storage.dart';
import 'package:music_room/core/widgets/ws_shell.dart';

class TrackVoteScreen extends ConsumerStatefulWidget {
  const TrackVoteScreen({super.key});

  @override
  ConsumerState<TrackVoteScreen> createState() => _TrackVoteScreenState();
}

class _TrackVoteScreenState extends ConsumerState<TrackVoteScreen> {
  static const _hubPath = '/api/v1/ws/track-vote';

  String? _token;
  bool _tokenLoaded = false;

  @override
  void initState() {
    super.initState();
    _loadToken();
  }

  Future<void> _loadToken() async {
    final token = await ref.read(tokenStorageProvider).getAccessToken();
    if (mounted) setState(() { _token = token; _tokenLoaded = true; });
  }

  @override
  Widget build(BuildContext context) {
    if (!_tokenLoaded) {
      return const Scaffold(body: Center(child: CircularProgressIndicator()));
    }

    final connState = ref.watch(wsProvider((_hubPath, _token)));

    return WsShell(
      title: 'Track Vote',
      state: connState,
      onRetry: () =>
          ref.read(wsProvider((_hubPath, _token)).notifier).reconnect(),
      child: const Center(child: Text('Track Vote')),
    );
  }
}
