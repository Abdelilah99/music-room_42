import 'package:flutter_dotenv/flutter_dotenv.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

class WsClient {
  WebSocketChannel connect(String path) {
    final base = dotenv.env['API_BASE_URL'] ?? '';
    final wsBase = base.replaceFirst(RegExp(r'^http'), 'ws');
    return WebSocketChannel.connect(Uri.parse('$wsBase$path'));
  }
}
