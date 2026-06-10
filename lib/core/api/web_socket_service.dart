import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/foundation.dart';
import 'package:flutter_dotenv/flutter_dotenv.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:music_room/core/services/token_storage.dart';
import 'package:web_socket_channel/io.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

enum WsConnectionState { connecting, connected, disconnected, error }

// ── Core service (framework-agnostic) ────────────────────────────────────────

class WebSocketService {
  static const _maxAttempts = 3;

  final String _path;
  final Future<String?> Function() _getToken;
  final void Function(WsConnectionState) _onStateChange;

  final _controller = StreamController<dynamic>.broadcast();

  WebSocketChannel? _channel;
  StreamSubscription<dynamic>? _subscription;
  Timer? _retryTimer;

  bool _intentionalClose = false;
  int _attempt = 0;

  WebSocketService({
    required this._path,
    required this._getToken,
    required this._onStateChange,
  });

  Stream<dynamic> get stream => _controller.stream;

  void connect() {
    _intentionalClose = false;
    _attempt = 0;
    _openSocket();
  }

  void send(String message) {
    if (_channel == null) {
      debugPrint('WS send skipped: not connected to $_path');
      return;
    }
    _channel!.sink.add(message);
  }

  void disconnect() {
    _intentionalClose = true;
    _retryTimer?.cancel();
    _retryTimer = null;
    _subscription?.cancel();
    _subscription = null;
    _channel?.sink.close();
    _channel = null;
    _onStateChange(WsConnectionState.disconnected);
    if (!_controller.isClosed) _controller.close();
  }

  Future<void> _openSocket() async {
    _onStateChange(WsConnectionState.connecting);

    final base = dotenv.env['API_BASE_URL'] ?? '';
    final wsBase = base.replaceFirst(RegExp(r'^http'), 'ws');
    final uri = Uri.parse('$wsBase$_path');
    // Fresh token read on every connect/reconnect so a refreshed access token
    // is always used — avoids stale-token failures after the interceptor rotates it.
    final token = await _getToken();
    final headers = token != null
        ? <String, dynamic>{'Authorization': 'Bearer $token'}
        : <String, dynamic>{};

    late WebSocketChannel channel;
    try {
      channel = IOWebSocketChannel.connect(uri, headers: headers);
      await channel.ready;
    } catch (_) {
      if (!_intentionalClose) _scheduleReconnect();
      return;
    }

    if (_intentionalClose) {
      await channel.sink.close();
      return;
    }

    _channel = channel;
    _attempt = 0;
    _onStateChange(WsConnectionState.connected);

    _subscription?.cancel();
    _subscription = _channel!.stream.listen(
      (msg) {
        if (!_controller.isClosed) _controller.add(msg);
      },
      onError: (_) {
        if (!_intentionalClose) _scheduleReconnect();
      },
      onDone: () {
        if (!_intentionalClose) _scheduleReconnect();
      },
    );
  }

  void _scheduleReconnect() {
    _subscription?.cancel();
    _subscription = null;
    _channel = null;
    _attempt++;
    if (_attempt > _maxAttempts) {
      _onStateChange(WsConnectionState.error);
      return;
    }
    final delay = Duration(seconds: math.pow(2, _attempt - 1).toInt());
    _onStateChange(WsConnectionState.connecting);
    _retryTimer = Timer(delay, _openSocket);
  }
}

// ── Riverpod layer ────────────────────────────────────────────────────────────

class WsNotifier extends Notifier<WsConnectionState> {
  WsNotifier(this._path);
  final String _path;

  late final WebSocketService _service;

  @override
  WsConnectionState build() {
    _service = WebSocketService(
      path: _path,
      getToken: () => ref.read(tokenStorageProvider).getAccessToken(),
      onStateChange: (s) {
        if (ref.mounted) state = s;
      },
    );
    _service.connect();
    ref.onDispose(_service.disconnect);
    return WsConnectionState.connecting;
  }

  Stream<dynamic> get messageStream => _service.stream;

  void send(String message) => _service.send(message);

  /// Resets attempt counter and re-opens the connection. Use after error state.
  void reconnect() {
    _service.connect();
  }
}

/// autoDispose + family: each hub path gets its own connection, torn down
/// automatically when the last listener unsubscribes.
final wsProvider = NotifierProvider.autoDispose
    .family<WsNotifier, WsConnectionState, String>(
  (path) => WsNotifier(path),
);
