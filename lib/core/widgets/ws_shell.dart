import 'package:flutter/material.dart';
import 'package:music_room/core/api/web_socket_service.dart';

/// Drop-in scaffold wrapper for screens backed by a WebSocket hub.
/// Handles the loading/error/connected states so feature screens only
/// need to supply their content widget.
class WsShell extends StatelessWidget {
  const WsShell({
    super.key,
    required this.title,
    required this.state,
    required this.onRetry,
    required this.child,
    this.floatingActionButton,
  });

  final String title;
  final WsConnectionState state;
  final VoidCallback onRetry;
  final Widget child;
  final Widget? floatingActionButton;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(title),
        actions: [
          Padding(
            padding: const EdgeInsets.only(right: 12),
            child: _ConnectionDot(state: state),
          ),
        ],
      ),
      body: switch (state) {
        WsConnectionState.error => _ErrorView(onRetry: onRetry),
        _ => child,
      },
      floatingActionButton:
          state == WsConnectionState.error ? null : floatingActionButton,
    );
  }
}

class _ConnectionDot extends StatelessWidget {
  const _ConnectionDot({required this.state});
  final WsConnectionState state;

  @override
  Widget build(BuildContext context) {
    final (color, label) = switch (state) {
      WsConnectionState.connected    => (Colors.green, 'Connected'),
      WsConnectionState.connecting   => (Colors.orange, 'Connecting…'),
      WsConnectionState.disconnected => (Colors.grey, 'Disconnected'),
      WsConnectionState.error        => (Colors.red, 'Error'),
    };
    return Tooltip(
      message: label,
      child: CircleAvatar(radius: 6, backgroundColor: color),
    );
  }
}

class _ErrorView extends StatelessWidget {
  const _ErrorView({required this.onRetry});
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          const Icon(Icons.wifi_off, size: 48, color: Colors.red),
          const SizedBox(height: 12),
          const Text('Connection lost after 3 attempts.'),
          const SizedBox(height: 16),
          FilledButton.icon(
            onPressed: onRetry,
            icon: const Icon(Icons.refresh),
            label: const Text('Retry'),
          ),
        ],
      ),
    );
  }
}
