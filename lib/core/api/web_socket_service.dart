import 'dart:async';
import 'dart:math' as math;

import 'package:flutter_dotenv/flutter_dotenv.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:web_socket_channel/io.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

enum WsConnectionState { connecting, connected, disconnected, error }

// ── Core service (framework-agnostic) ────────────────────────────────────────

class WebSocketService {
  static const _maxAttempts = 3;

  final String _path;
  final String? _token;
  final void Function(WsConnectionState) _onStateChange;

  final _controller = StreamController<dynamic>.broadcast();

  WebSocketChannel? _channel;
  StreamSubscription<dynamic>? _subscription;
  Timer? _retryTimer;

  bool _intentionalClose = false;
  int _attempt = 0;

  WebSocketService({
    required this._path,
    this._token,
    required this._onStateChange,
  });

  Stream<dynamic> get stream => _controller.stream;

  void connect() {
    _intentionalClose = false;
    _attempt = 0;
    _openSocket();
  }

  void send(String message) => _channel?.sink.add(message);

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
    final headers = _token != null
        ? <String, dynamic>{'Authorization': 'Bearer $_token'}
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
    debugPrint('[WS:$_path] retry $_attempt/$_maxAttempts in ${delay.inSeconds}s');
    _onStateChange(WsConnectionState.connecting);
    _retryTimer = Timer(delay, _openSocket);
  }
}

// ── Riverpod layer ────────────────────────────────────────────────────────────

class WsNotifier extends StateNotifier<WsConnectionState> {
  late final WebSocketService _service;

  WsNotifier(String path, String? token) : super(WsConnectionState.connecting) {
    _service = WebSocketService(
      path: path,
      token: token,
      onStateChange: (s) {
        if (mounted) state = s;
      },
    );
    _service.connect();
  }

  Stream<dynamic> get messageStream => _service.stream;

  void send(String message) => _service.send(message);

  /// Resets attempt counter and re-opens the connection. Use after error state.
  void reconnect() {
    _service.connect();
  }

  @override
  void dispose() {
    _service.disconnect();
    super.dispose();
  }
}

/// autoDispose + family: each (path, token) pair gets its own hub connection.
/// The connection is torn down automatically when the last listener
/// unsubscribes — i.e. when the screen is removed from the widget tree.
final wsProvider = StateNotifierProvider.autoDispose
    .family<WsNotifier, WsConnectionState, (String, String?)>(
  (ref, args) => WsNotifier(args.$1, args.$2),
);
