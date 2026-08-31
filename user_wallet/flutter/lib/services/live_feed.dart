/**
 * LiveFeedSocket — real WebSocket price feed over /api/v1/ws.
 * Streams ticker frames into a broadcast Stream; error frames surface as
 * payload, never fabricated prices.
 */

import 'dart:async';
import 'dart:convert';

import 'package:web_socket_channel/web_socket_channel.dart';

import 'user_wallet.dart';

class LiveFeedSocket {
  WebSocketChannel? _channel;
  final _controller = StreamController<Map<String, dynamic>>.broadcast();
  Stream<Map<String, dynamic>> get stream => _controller.stream;

  Future<void> connect(UserWalletService api) async {
    final base = await api.baseUrl();
    final wsBase = base.replaceFirst('http://', 'ws://').replaceFirst('https://', 'wss://');
    try {
      _channel = WebSocketChannel.connect(Uri.parse('$wsBase/api/v1/ws'));
    } catch (e) {
      _controller.add({'type': 'error', 'error': 'Live feed unavailable: $e'});
      return;
    }
    _channel!.stream.listen(
      (frame) {
        try {
          final data = jsonDecode(frame is String ? frame : utf8.decode(frame as List<int>));
          if (data is Map<String, dynamic>) _controller.add(data);
        } catch (_) {/* ignore malformed frames */}
      },
      onError: (_) {
        _controller.add({'type': 'error', 'error': 'Live feed connection lost'});
      },
    );
  }

  void dispose() {
    _channel?.sink.close();
    _controller.close();
  }
}
