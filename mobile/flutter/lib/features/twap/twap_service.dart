/// TWAP Service for Flutter
/// Time-Weighted Average Price trading strategy

import 'dart:convert';
import 'package:http/http.dart' as http;

class TWAPService {
  static const String _baseUrl = 'http://localhost:8443/api/v1/twap';
  
  final http.Client _client;
  
  TWAPService({http.Client? client}) : _client = client ?? http.Client();
  
  /// Create a TWAP order
  Future<TWAPOrder> createOrder({
    required String tokenIn,
    required String tokenOut,
    required double totalAmount,
    required int numberOfTrades,
    required Duration interval,
    double? maxSlippage,
  }) async {
    final response = await _client.post(
      Uri.parse('$_baseUrl/orders'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        'tokenIn': tokenIn,
        'tokenOut': tokenOut,
        'totalAmount': totalAmount,
        'numberOfTrades': numberOfTrades,
        'intervalSeconds': interval.inSeconds,
        'maxSlippage': maxSlippage ?? 0.5,
      }),
    );
    
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return TWAPOrder.fromJson(data);
    }
    throw Exception('Failed to create TWAP order: ${response.body}');
  }
  
  /// Get TWAP order status
  Future<TWAPOrder> getOrder(String orderId) async {
    final response = await _client.get(
      Uri.parse('$_baseUrl/orders/$orderId'),
    );
    
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return TWAPOrder.fromJson(data);
    }
    throw Exception('Failed to get TWAP order: ${response.body}');
  }
  
  /// Cancel TWAP order
  Future<bool> cancelOrder(String orderId) async {
    final response = await _client.delete(
      Uri.parse('$_baseUrl/orders/$orderId'),
    );
    
    return response.statusCode == 200;
  }
  
  /// Pause TWAP order
  Future<bool> pauseOrder(String orderId) async {
    final response = await _client.post(
      Uri.parse('$_baseUrl/orders/$orderId/pause'),
      headers: {'Content-Type': 'application/json'},
    );
    
    return response.statusCode == 200;
  }
  
  /// Resume TWAP order
  Future<bool> resumeOrder(String orderId) async {
    final response = await _client.post(
      Uri.parse('$_baseUrl/orders/$orderId/resume'),
      headers: {'Content-Type': 'application/json'},
    );
    
    return response.statusCode == 200;
  }
  
  /// Get user's TWAP orders
  Future<List<TWAPOrder>> getUserOrders(String address) async {
    final response = await _client.get(
      Uri.parse('$_baseUrl/orders?user=$address'),
    );
    
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return (data['orders'] as List)
          .map((e) => TWAPOrder.fromJson(e))
          .toList();
    }
    throw Exception('Failed to get orders: ${response.body}');
  }
  
  /// Get estimated average price
  Future<PriceEstimate> estimatePrice({
    required String tokenIn,
    required String tokenOut,
    required double amount,
  }) async {
    final response = await _client.post(
      Uri.parse('$_baseUrl/estimate'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        'tokenIn': tokenIn,
        'tokenOut': tokenOut,
        'amount': amount,
      }),
    );
    
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return PriceEstimate.fromJson(data);
    }
    throw Exception('Failed to estimate price: ${response.body}');
  }
  
  void dispose() {
    _client.close();
  }
}

class TWAPOrder {
  final String orderId;
  final String user;
  final String tokenIn;
  final String tokenOut;
  final double totalAmount;
  final double filledAmount;
  final double remainingAmount;
  final int numberOfTrades;
  final int completedTrades;
  final Duration interval;
  final double maxSlippage;
  final TWAPStatus status;
  final List<TWAPTrade> trades;
  final DateTime createdAt;
  final DateTime? completedAt;
  
  TWAPOrder({
    required this.orderId,
    required this.user,
    required this.tokenIn,
    required this.tokenOut,
    required this.totalAmount,
    required this.filledAmount,
    required this.remainingAmount,
    required this.numberOfTrades,
    required this.completedTrades,
    required this.interval,
    required this.maxSlippage,
    required this.status,
    required this.trades,
    required this.createdAt,
    this.completedAt,
  });
  
  factory TWAPOrder.fromJson(Map<String, dynamic> json) {
    return TWAPOrder(
      orderId: json['orderId'],
      user: json['user'],
      tokenIn: json['tokenIn'],
      tokenOut: json['tokenOut'],
      totalAmount: double.parse(json['totalAmount'].toString()),
      filledAmount: double.parse(json['filledAmount'].toString()),
      remainingAmount: double.parse(json['remainingAmount'].toString()),
      numberOfTrades: json['numberOfTrades'],
      completedTrades: json['completedTrades'],
      interval: Duration(seconds: json['intervalSeconds']),
      maxSlippage: double.parse(json['maxSlippage'].toString()),
      status: TWAPStatus.values.firstWhere(
        (e) => e.name == json['status'],
        orElse: () => TWAPStatus.pending,
      ),
      trades: (json['trades'] as List?)
          ?.map((e) => TWAPTrade.fromJson(e))
          .toList() ?? [],
      createdAt: DateTime.parse(json['createdAt']),
      completedAt: json['completedAt'] != null 
          ? DateTime.parse(json['completedAt']) 
          : null,
    );
  }
  
  double get progressPercent => (completedTrades / numberOfTrades) * 100;
  double get averagePrice {
    if (trades.isEmpty) return 0;
    double total = 0;
    for (var trade in trades) {
      total += trade.price * trade.amount;
    }
    return filledAmount > 0 ? total / filledAmount : 0;
  }
}

class TWAPTrade {
  final String tradeId;
  final double amount;
  final double price;
  final String txHash;
  final DateTime timestamp;
  
  TWAPTrade({
    required this.tradeId,
    required this.amount,
    required this.price,
    required this.txHash,
    required this.timestamp,
  });
  
  factory TWAPTrade.fromJson(Map<String, dynamic> json) {
    return TWAPTrade(
      tradeId: json['tradeId'],
      amount: double.parse(json['amount'].toString()),
      price: double.parse(json['price'].toString()),
      txHash: json['txHash'],
      timestamp: DateTime.parse(json['timestamp']),
    );
  }
}

enum TWAPStatus {
  pending,
  active,
  paused,
  completed,
  cancelled,
}

class PriceEstimate {
  final double averagePrice;
  final double minPrice;
  final double maxPrice;
  final double estimatedSlippage;
  
  PriceEstimate({
    required this.averagePrice,
    required this.minPrice,
    required this.maxPrice,
    required this.estimatedSlippage,
  });
  
  factory PriceEstimate.fromJson(Map<String, dynamic> json) {
    return PriceEstimate(
      averagePrice: double.parse(json['averagePrice'].toString()),
      minPrice: double.parse(json['minPrice'].toString()),
      maxPrice: double.parse(json['maxPrice'].toString()),
      estimatedSlippage: double.parse(json['estimatedSlippage'].toString()),
    );
  }
}
