/**
 * WebSocketService - Flutter Implementation
 * Real-time connection for Master Wallet
 */

import 'dart:async';
import 'dart:convert';
import 'package:web_socket_channel/web_socket_channel.dart';

enum ConnectionState {
  disconnected,
  connecting,
  connected,
  reconnecting,
  error,
}

class WebSocketService {
  static final WebSocketService _instance = WebSocketService._internal();
  factory WebSocketService() => _instance;
  WebSocketService._internal();
  
  WebSocketChannel? _channel;
  StreamSubscription? _subscription;
  Timer? _heartbeatTimer;
  Timer? _reconnectTimer;
  
  String? _walletId;
  String? _authToken;
  int _reconnectAttempts = 0;
  
  static const String WS_URL = '';
  static const int MAX_RECONNECT_ATTEMPTS = 10;
  static const Duration RECONNECT_DELAY = Duration(seconds: 5);
  
  ConnectionState _connectionState = ConnectionState.disconnected;
  ConnectionState get connectionState => _connectionState;
  
  final _stateController = StreamController<ConnectionState>.broadcast();
  Stream<ConnectionState> get stateStream => _stateController.stream;
  
  final _messageController = StreamController<String>.broadcast();
  Stream<String> get messageStream => _messageController.stream;
  
  final _balanceController = StreamController<BalanceUpdate>.broadcast();
  Stream<BalanceUpdate> get balanceStream => _balanceController.stream;
  
  final _transactionController = StreamController<TransactionUpdate>.broadcast();
  Stream<TransactionUpdate> get transactionStream => _transactionController.stream;
  
  /// Connect to WebSocket server
  void connect({required String walletId, String? token}) {
    _walletId = walletId;
    _authToken = token;
    _connect();
  }
  
  void _connect() {
    _connectionState = ConnectionState.connecting;
    _stateController.add(_connectionState);
    
    try {
      _channel = WebSocketChannel.connect(Uri.parse(WS_URL));
      
      _subscription = _channel!.stream.listen(
        _onMessage,
        onError: _onError,
        onDone: _onDone,
      );
      
      _connectionState = ConnectionState.connected;
      _stateController.add(_connectionState);
      _reconnectAttempts = 0;
      
      _authenticate();
      _startHeartbeat();
    } catch (e) {
      _connectionState = ConnectionState.error;
      _stateController.add(_connectionState);
      _scheduleReconnect();
    }
  }
  
  /// Disconnect from server
  void disconnect() {
    _stopHeartbeat();
    _cancelReconnect();
    _subscription?.cancel();
    _channel?.sink.close();
    _channel = null;
    
    _connectionState = ConnectionState.disconnected;
    _stateController.add(_connectionState);
  }
  
  /// Subscribe to balance updates
  void subscribeToBalance(int chainId) {
    _sendMessage('subscribe', 'balance', {'chainId': chainId});
  }
  
  /// Unsubscribe from balance updates
  void unsubscribeFromBalance(int chainId) {
    _sendMessage('unsubscribe', 'balance', {'chainId': chainId});
  }
  
  /// Subscribe to transaction updates
  void subscribeToTransactions(String address) {
    _sendMessage('subscribe', 'transactions', {'address': address});
  }
  
  /// Subscribe to ticker updates
  void subscribeToTicker(String pair) {
    _sendMessage('subscribe', 'ticker', {'pair': pair});
  }
  
  /// Subscribe to order book
  void subscribeToOrderBook(String pair) {
    _sendMessage('subscribe', 'orderbook', {'pair': pair});
  }
  
  void _authenticate() {
    _sendMessage('auth', 'auth', {
      'walletId': _walletId ?? '',
      'token': _authToken ?? '',
    });
  }
  
  void _sendMessage(String type, String channel, Map<String, dynamic> data) {
    final message = {
      'type': type,
      'channel': channel,
      'data': data,
      'timestamp': DateTime.now().millisecondsSinceEpoch,
    };
    
    _channel?.sink.add(jsonEncode(message));
  }
  
  void _onMessage(dynamic data) {
    try {
      final json = jsonDecode(data as String) as Map<String, dynamic>;
      final channel = json['channel'] as String?;
      final payload = json['data'] as Map<String, dynamic>? ?? {};
      
      _messageController.add(data as String);
      
      switch (channel) {
        case 'balance':
          _handleBalanceUpdate(payload);
          break;
        case 'transactions':
          _handleTransactionUpdate(payload);
          break;
      }
    } catch (e) {
      // Handle parse error
    }
  }
  
  void _handleBalanceUpdate(Map<String, dynamic> data) {
    _balanceController.add(BalanceUpdate(
      chainId: data['chainId'] as int? ?? 0,
      address: data['address'] as String? ?? '',
      balance: data['balance'] as String? ?? '0',
      token: data['token'] as String? ?? 'ETH',
      timestamp: data['timestamp'] as int? ?? 0,
    ));
  }
  
  void _handleTransactionUpdate(Map<String, dynamic> data) {
    _transactionController.add(TransactionUpdate(
      txHash: data['txHash'] as String? ?? '',
      from: data['from'] as String? ?? '',
      to: data['to'] as String? ?? '',
      amount: data['amount'] as String? ?? '0',
      status: data['status'] as String? ?? '',
      timestamp: data['timestamp'] as int? ?? 0,
    ));
  }
  
  void _onError(dynamic error) {
    _connectionState = ConnectionState.error;
    _stateController.add(_connectionState);
    _stopHeartbeat();
    _scheduleReconnect();
  }
  
  void _onDone() {
    _connectionState = ConnectionState.disconnected;
    _stateController.add(_connectionState);
    _stopHeartbeat();
    _scheduleReconnect();
  }
  
  void _scheduleReconnect() {
    if (_reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
      _connectionState = ConnectionState.error;
      _stateController.add(_connectionState);
      return;
    }
    
    _reconnectAttempts++;
    _connectionState = ConnectionState.reconnecting;
    _stateController.add(_connectionState);
    
    _reconnectTimer = Timer(
      RECONNECT_DELAY * _reconnectAttempts,
      _connect,
    );
  }
  
  void _cancelReconnect() {
    _reconnectTimer?.cancel();
    _reconnectTimer = null;
  }
  
  void _startHeartbeat() {
    _heartbeatTimer = Timer.periodic(const Duration(seconds: 15), (_) {
      _sendMessage('ping', 'heartbeat', {});
    });
  }
  
  void _stopHeartbeat() {
    _heartbeatTimer?.cancel();
    _heartbeatTimer = null;
  }
  
  void dispose() {
    disconnect();
    _stateController.close();
    _messageController.close();
    _balanceController.close();
    _transactionController.close();
  }
}

// Data Classes

class BalanceUpdate {
  final int chainId;
  final String address;
  final String balance;
  final String token;
  final int timestamp;
  
  BalanceUpdate({
    required this.chainId,
    required this.address,
    required this.balance,
    required this.token,
    required this.timestamp,
  });
}

class TransactionUpdate {
  final String txHash;
  final String from;
  final String to;
  final String amount;
  final String status;
  final int timestamp;
  
  TransactionUpdate({
    required this.txHash,
    required this.from,
    required this.to,
    required this.amount,
    required this.status,
    required this.timestamp,
  });
}
