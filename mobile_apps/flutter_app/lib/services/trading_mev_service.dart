import 'dart:convert';
import 'package:http/http.dart' as http;

/// Trading Service for Flutter App
/// Order Book, Trading Charts, Positions
class TradingService {
  static final TradingService _instance = TradingService._internal();
  factory TradingService() => _instance;
  TradingService._internal();

  final String _baseUrl = 'https://api.tigerwallet.com/v1/trading';

  // Order Book
  Future<OrderBook?> getOrderBook(String symbol, {int limit = 50}) async {
    final response = await http.get(
      Uri.parse('$_baseUrl/orderbook?symbol=$symbol&limit=$limit'),
      headers: {'Content-Type': 'application/json'},
    );
    if (response.statusCode == 200) {
      return OrderBook.fromJson(json.decode(response.body));
    }
    return null;
  }

  // Candlesticks
  Future<List<Candlestick>> getCandlesticks(String symbol, {String interval = '1h', int limit = 100}) async {
    final response = await http.get(
      Uri.parse('$_baseUrl/klines?symbol=$symbol&interval=$interval&limit=$limit'),
      headers: {'Content-Type': 'application/json'},
    );
    if (response.statusCode == 200) {
      final List<dynamic> data = json.decode(response.body);
      return data.map((e) => Candlestick.fromJson(e)).toList();
    }
    return [];
  }

  // Positions
  Future<List<Position>> getPositions(String walletAddress) async {
    final response = await http.get(
      Uri.parse('$_baseUrl/positions/$walletAddress'),
      headers: {'Content-Type': 'application/json'},
    );
    if (response.statusCode == 200) {
      final List<dynamic> data = json.decode(response.body);
      return data.map((e) => Position.fromJson(e)).toList();
    }
    return [];
  }

  // Open Orders
  Future<List<OpenOrder>> getOpenOrders(String walletAddress) async {
    final response = await http.get(
      Uri.parse('$_baseUrl/orders/$walletAddress?status=open'),
      headers: {'Content-Type': 'application/json'},
    );
    if (response.statusCode == 200) {
      final List<dynamic> data = json.decode(response.body);
      return data.map((e) => OpenOrder.fromJson(e)).toList();
    }
    return [];
  }

  // Place Market Order
  Future<String> placeMarketOrder(String walletAddress, String symbol, String side, double amount, {int leverage = 1}) async {
    final response = await http.post(
      Uri.parse('$_baseUrl/orders'),
      headers: {'Content-Type': 'application/json'},
      body: json.encode({
        'wallet_address': walletAddress,
        'symbol': symbol,
        'side': side,
        'type': 'market',
        'amount': amount,
        'leverage': leverage,
      }),
    );
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return data['txHash'];
    }
    throw Exception('Failed to place order');
  }

  // Place Limit Order
  Future<String> placeLimitOrder(String walletAddress, String symbol, String side, double price, double amount, {int leverage = 1}) async {
    final response = await http.post(
      Uri.parse('$_baseUrl/orders'),
      headers: {'Content-Type': 'application/json'},
      body: json.encode({
        'wallet_address': walletAddress,
        'symbol': symbol,
        'side': side,
        'type': 'limit',
        'price': price,
        'amount': amount,
        'leverage': leverage,
      }),
    );
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return data['txHash'];
    }
    throw Exception('Failed to place order');
  }

  // Cancel Order
  Future<bool> cancelOrder(String walletAddress, String orderId) async {
    final response = await http.delete(
      Uri.parse('$_baseUrl/orders/$orderId'),
      headers: {'Content-Type': 'application/json'},
      body: json.encode({'wallet_address': walletAddress}),
    );
    return response.statusCode == 200;
  }

  // Close Position
  Future<String> closePosition(String walletAddress, String positionId) async {
    final response = await http.post(
      Uri.parse('$_baseUrl/positions/$positionId/close'),
      headers: {'Content-Type': 'application/json'},
      body: json.encode({'wallet_address': walletAddress}),
    );
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return data['txHash'];
    }
    throw Exception('Failed to close position');
  }
}

// Models
class OrderBook {
  final List<List<double>> bids;
  final List<List<double>> asks;
  final int timestamp;
  final String symbol;

  OrderBook({required this.bids, required this.asks, required this.timestamp, required this.symbol});

  factory OrderBook.fromJson(Map<String, dynamic> json) {
    return OrderBook(
      bids: (json['bids'] as List).map((e) => (e as List).map((e) => (e as num).toDouble()).toList()).toList(),
      asks: (json['asks'] as List).map((e) => (e as List).map((e) => (e as num).toDouble()).toList()).toList(),
      timestamp: json['timestamp'],
      symbol: json['symbol'],
    );
  }
}

class Candlestick {
  final int timestamp;
  final double open;
  final double high;
  final double low;
  final double close;
  final double volume;

  Candlestick({required this.timestamp, required this.open, required this.high, required this.low, required this.close, required this.volume});

  factory Candlestick.fromJson(List<dynamic> json) {
    return Candlestick(
      timestamp: json[0],
      open: (json[1] as num).toDouble(),
      high: (json[2] as num).toDouble(),
      low: (json[3] as num).toDouble(),
      close: (json[4] as num).toDouble(),
      volume: (json[5] as num).toDouble(),
    );
  }
}

class Position {
  final String id;
  final String symbol;
  final String side;
  final double amount;
  final double entryPrice;
  final double currentPrice;
  final double unrealizedPnl;
  final int leverage;
  final double liquidationPrice;
  final double margin;

  Position({required this.id, required this.symbol, required this.side, required this.amount, required this.entryPrice, required this.currentPrice, required this.unrealizedPnl, required this.leverage, required this.liquidationPrice, required this.margin});

  factory Position.fromJson(Map<String, dynamic> json) {
    return Position(
      id: json['id'],
      symbol: json['symbol'],
      side: json['side'],
      amount: (json['amount'] as num).toDouble(),
      entryPrice: (json['entry_price'] as num).toDouble(),
      currentPrice: (json['current_price'] as num).toDouble(),
      unrealizedPnl: (json['unrealized_pnl'] as num).toDouble(),
      leverage: json['leverage'],
      liquidationPrice: (json['liquidation_price'] as num).toDouble(),
      margin: (json['margin'] as num).toDouble(),
    );
  }
}

class OpenOrder {
  final String id;
  final String symbol;
  final String side;
  final String type;
  final double price;
  final double amount;
  final double filledAmount;
  final String status;
  final int createdAt;

  OpenOrder({required this.id, required this.symbol, required this.side, required this.type, required this.price, required this.amount, required this.filledAmount, required this.status, required this.createdAt});

  factory OpenOrder.fromJson(Map<String, dynamic> json) {
    return OpenOrder(
      id: json['id'],
      symbol: json['symbol'],
      side: json['side'],
      type: json['type'],
      price: (json['price'] as num).toDouble(),
      amount: (json['amount'] as num).toDouble(),
      filledAmount: (json['filled_amount'] as num).toDouble(),
      status: json['status'],
      createdAt: json['created_at'],
    );
  }
}

/// MEV Protection Service
class MEVProtectionService {
  static final MEVProtectionService _instance = MEVProtectionService._internal();
  factory MEVProtectionService() => _instance;
  MEVProtectionService._internal();

  Future<SandwichDetection?> detectSandwichAttack(String txHash) async {
    final response = await http.get(
      Uri.parse('https://api.tigerwallet.com/v1/mev/detect-sandwich?tx=$txHash'),
      headers: {'Content-Type': 'application/json'},
    );
    if (response.statusCode == 200) {
      return SandwichDetection.fromJson(json.decode(response.body));
    }
    return null;
  }

  Future<SimulationResult?> simulateTransaction(String from, String to, String data, String value) async {
    final response = await http.post(
      Uri.parse('https://api.tigerwallet.com/v1/mev/simulate'),
      headers: {'Content-Type': 'application/json'},
      body: json.encode({'from': from, 'to': to, 'data': data, 'value': value}),
    );
    if (response.statusCode == 200) {
      return SimulationResult.fromJson(json.decode(response.body));
    }
    return null;
  }

  Future<String> submitWithProtection(String signedTx, {String protectionLevel = 'medium'}) async {
    final response = await http.post(
      Uri.parse('https://api.tigerwallet.com/v1/mev/submit'),
      headers: {'Content-Type': 'application/json'},
      body: json.encode({'signed_tx': signedTx, 'protection_level': protectionLevel}),
    );
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return data['txHash'];
    }
    throw Exception('Failed to submit transaction');
  }
}

class SandwichDetection {
  final bool detected;
  final String? frontRunTx;
  final String? backRunTx;
  final double? profit;
  final String severity;

  SandwichDetection({required this.detected, this.frontRunTx, this.backRunTx, this.profit, required this.severity});

  factory SandwichDetection.fromJson(Map<String, dynamic> json) {
    return SandwichDetection(
      detected: json['detected'],
      frontRunTx: json['front_run_tx'],
      backRunTx: json['back_run_tx'],
      profit: json['profit']?.toDouble(),
      severity: json['severity'],
    );
  }
}

class SimulationResult {
  final bool success;
  final int gasUsed;
  final String? error;

  SimulationResult({required this.success, required this.gasUsed, this.error});

  factory SimulationResult.fromJson(Map<String, dynamic> json) {
    return SimulationResult(
      success: json['success'],
      gasUsed: json['gas_used'],
      error: json['error'],
    );
  }
}

/// Session Keys Service
class SessionKeysService {
  static final SessionKeysService _instance = SessionKeysService._internal();
  factory SessionKeysService() => _instance;
  SessionKeysService._internal();

  Future<SessionKey?> generateSessionKey(String walletAddress, String dappUrl, List<String> permissions, {int expiresIn = 86400}) async {
    final response = await http.post(
      Uri.parse('https://api.tigerwallet.com/v1/session-keys'),
      headers: {'Content-Type': 'application/json'},
      body: json.encode({
        'wallet_address': walletAddress,
        'dapp_url': dappUrl,
        'permissions': permissions,
        'expires_in': expiresIn,
      }),
    );
    if (response.statusCode == 200) {
      return SessionKey.fromJson(json.decode(response.body));
    }
    return null;
  }

  Future<List<SessionKey>> getSessionKeys(String walletAddress) async {
    final response = await http.get(
      Uri.parse('https://api.tigerwallet.com/v1/session-keys/$walletAddress'),
      headers: {'Content-Type': 'application/json'},
    );
    if (response.statusCode == 200) {
      final List<dynamic> data = json.decode(response.body);
      return data.map((e) => SessionKey.fromJson(e)).toList();
    }
    return [];
  }

  Future<bool> revokeSessionKey(String walletAddress, String sessionKeyId) async {
    final response = await http.delete(
      Uri.parse('https://api.tigerwallet.com/v1/session-keys/$sessionKeyId'),
      headers: {'Content-Type': 'application/json'},
      body: json.encode({'wallet_address': walletAddress}),
    );
    return response.statusCode == 200;
  }
}

class SessionKey {
  final String id;
  final String key;
  final String dapp;
  final List<String> permissions;
  final int expiresAt;
  final int createdAt;

  SessionKey({required this.id, required this.key, required this.dapp, required this.permissions, required this.expiresAt, required this.createdAt});

  factory SessionKey.fromJson(Map<String, dynamic> json) {
    return SessionKey(
      id: json['id'],
      key: json['key'],
      dapp: json['dapp'],
      permissions: List<String>.from(json['permissions']),
      expiresAt: json['expires_at'],
      createdAt: json['created_at'],
    );
  }
}

/// Gas Optimization Service
class GasOptimizationService {
  static final GasOptimizationService _instance = GasOptimizationService._internal();
  factory GasOptimizationService() => _instance;
  GasOptimizationService._internal();

  Future<GasPrice?> getGasPrices({String chain = 'ethereum'}) async {
    final response = await http.get(
      Uri.parse('https://api.tigerwallet.com/v1/gas/prices?chain=$chain'),
      headers: {'Content-Type': 'application/json'},
    );
    if (response.statusCode == 200) {
      return GasPrice.fromJson(json.decode(response.body));
    }
    return null;
  }

  Future<List<OptimizationSuggestion>> getOptimizationSuggestions(String from, String to, String data) async {
    final response = await http.post(
      Uri.parse('https://api.tigerwallet.com/v1/gas/optimize'),
      headers: {'Content-Type': 'application/json'},
      body: json.encode({'from': from, 'to': to, 'data': data}),
    );
    if (response.statusCode == 200) {
      final List<dynamic> data = json.decode(response.body);
      return data.map((e) => OptimizationSuggestion.fromJson(e)).toList();
    }
    return [];
  }
}

class GasPrice {
  final int slow;
  final int standard;
  final int fast;
  final int instant;

  GasPrice({required this.slow, required this.standard, required this.fast, required this.instant});

  factory GasPrice.fromJson(Map<String, dynamic> json) {
    return GasPrice(
      slow: json['slow'],
      standard: json['standard'],
      fast: json['fast'],
      instant: json['instant'],
    );
  }
}

class OptimizationSuggestion {
  final String type;
  final double potentialSavings;
  final String recommendation;

  OptimizationSuggestion({required this.type, required this.potentialSavings, required this.recommendation});

  factory OptimizationSuggestion.fromJson(Map<String, dynamic> json) {
    return OptimizationSuggestion(
      type: json['type'],
      potentialSavings: (json['potential_savings'] as num).toDouble(),
      recommendation: json['recommendation'],
    );
  }
}
