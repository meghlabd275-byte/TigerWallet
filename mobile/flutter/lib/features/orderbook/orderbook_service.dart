/// Orderbook Service for Flutter
/// Provides real-time orderbook data and trading

import 'dart:convert';
import 'dart:async';
import 'package:http/http.dart' as http;

class OrderbookService {
  static const String _baseUrl = 'https://api.tigerwallet.com/v1/orderbook';
  
  final http.Client _client;
  Timer? _pollTimer;
  final _orderbookController = StreamController<Orderbook>.broadcast();
  
  OrderbookService({http.Client? client}) : _client = client ?? http.Client();
  
  Stream<Orderbook> get orderbookStream => _orderbookController.stream;
  
  /// Get orderbook for a trading pair
  Future<Orderbook> getOrderbook(String baseToken, String quoteToken, {int depth = 20}) async {
    final response = await _client.get(
      Uri.parse('$_baseUrl/$baseToken/$quoteToken?depth=$depth'),
    );
    
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return Orderbook.fromJson(data);
    }
    throw Exception('Failed to get orderbook: ${response.body}');
  }
  
  /// Start polling for orderbook updates
  void startPolling(String baseToken, String quoteToken, {Duration interval = const Duration(seconds: 1)}) {
    stopPolling();
    _pollTimer = Timer.periodic(interval, (_) async {
      try {
        final orderbook = await getOrderbook(baseToken, quoteToken);
        _orderbookController.add(orderbook);
      } catch (e) {
        // Handle error silently
      }
    });
  }
  
  /// Stop polling
  void stopPolling() {
    _pollTimer?.cancel();
    _pollTimer = null;
  }
  
  /// Get recent trades
  Future<List<Trade>> getRecentTrades(String baseToken, String quoteToken, {int limit = 50}) async {
    final response = await _client.get(
      Uri.parse('$_baseUrl/$baseToken/$quoteToken/trades?limit=$limit'),
    );
    
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return (data['trades'] as List).map((e) => Trade.fromJson(e)).toList();
    }
    throw Exception('Failed to get trades: ${response.body}');
  }
  
  /// Get candlestick data
  Future<List<Candle>> getCandles(
    String baseToken,
    String quoteToken, {
    String interval = '1h',
    int limit = 100,
  }) async {
    final response = await _client.get(
      Uri.parse('$_baseUrl/$baseToken/$quoteToken/candles?interval=$interval&limit=$limit'),
    );
    
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return (data['candles'] as List).map((e) => Candle.fromJson(e)).toList();
    }
    throw Exception('Failed to get candles: ${response.body}');
  }
  
  void dispose() {
    stopPolling();
    _orderbookController.close();
    _client.close();
  }
}

class Orderbook {
  final String baseToken;
  final String quoteToken;
  final List<OrderbookEntry> bids;
  final List<OrderbookEntry> asks;
  final int sequence;
  final DateTime timestamp;
  
  Orderbook({
    required this.baseToken,
    required this.quoteToken,
    required this.bids,
    required this.asks,
    required this.sequence,
    required this.timestamp,
  });
  
  factory Orderbook.fromJson(Map<String, dynamic> json) {
    return Orderbook(
      baseToken: json['baseToken'],
      quoteToken: json['quoteToken'],
      bids: (json['bids'] as List)
          .map((e) => OrderbookEntry.fromJson(e))
          .toList(),
      asks: (json['asks'] as List)
          .map((e) => OrderbookEntry.fromJson(e))
          .toList(),
      sequence: json['sequence'],
      timestamp: DateTime.fromMillisecondsSinceEpoch(json['timestamp']),
    );
  }
  
  double get spread {
    if (bids.isEmpty || asks.isEmpty) return 0;
    return asks.first.price - bids.first.price;
  }
  
  double get spreadPercent {
    if (bids.isEmpty || asks.isEmpty) return 0;
    return (spread / asks.first.price) * 100;
  }
}

class OrderbookEntry {
  final double price;
  final double amount;
  final double quoteAmount;
  
  OrderbookEntry({
    required this.price,
    required this.amount,
    required this.quoteAmount,
  });
  
  factory OrderbookEntry.fromJson(Map<String, dynamic> json) {
    return OrderbookEntry(
      price: double.parse(json['price'].toString()),
      amount: double.parse(json['amount'].toString()),
      quoteAmount: double.parse(json['quoteAmount'].toString()),
    );
  }
}

class Trade {
  final String id;
  final double price;
  final double amount;
  final String side;
  final DateTime timestamp;
  
  Trade({
    required this.id,
    required this.price,
    required this.amount,
    required this.side,
    required this.timestamp,
  });
  
  factory Trade.fromJson(Map<String, dynamic> json) {
    return Trade(
      id: json['id'],
      price: double.parse(json['price'].toString()),
      amount: double.parse(json['amount'].toString()),
      side: json['side'],
      timestamp: DateTime.fromMillisecondsSinceEpoch(json['timestamp']),
    );
  }
}

class Candle {
  final DateTime openTime;
  final double open;
  final double high;
  final double low;
  final double close;
  final double volume;
  final int trades;
  
  Candle({
    required this.openTime,
    required this.open,
    required this.high,
    required this.low,
    required this.close,
    required this.volume,
    required this.trades,
  });
  
  factory Candle.fromJson(Map<String, dynamic> json) {
    return Candle(
      openTime: DateTime.fromMillisecondsSinceEpoch(json['openTime']),
      open: double.parse(json['open'].toString()),
      high: double.parse(json['high'].toString()),
      low: double.parse(json['low'].toString()),
      close: double.parse(json['close'].toString()),
      volume: double.parse(json['volume'].toString()),
      trades: json['trades'],
    );
  }
}
