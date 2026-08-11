// Prediction Markets Service - Flutter Mobile

import 'dart:convert';
import 'package:http/http.dart' as http;

class PredictionMarketsService {
  static const String API_BASE = 'http://localhost:8443/api/v1';
  String? _token;
  
  PredictionMarketsService({String? token}) : _token = token;
  
  Map<String, String> get _headers => {
    'Content-Type': 'application/json',
    if (_token != null) 'Authorization': 'Bearer $_token',
  };
  
  // Get markets
  Future<List<PredictionMarket>> getMarkets() async {
    final response = await http.get(
      Uri.parse('$API_BASE/prediction/markets'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((m) => PredictionMarket.fromJson(m)).toList();
    }
    return [];
  }
  
  // Place bet
  Future<Bet> placeBet(String marketId, String outcome, double amount) async {
    final response = await http.post(
      Uri.parse('$API_BASE/prediction/$marketId/bet'),
      headers: _headers,
      body: json.encode({'outcome': outcome, 'amount': amount}),
    );
    
    if (response.statusCode == 201) {
      final data = json.decode(response.body);
      return Bet.fromJson(data['data']);
    }
    throw Exception('Failed to place bet');
  }
  
  // Claim winnings
  Future<double> claimWinnings(String marketId) async {
    final response = await http.post(
      Uri.parse('$API_BASE/prediction/$marketId/claim'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] ?? 0).toDouble();
    }
    return 0;
  }
  
  // Get user bets
  Future<List<Bet>> getUserBets() async {
    final response = await http.get(
      Uri.parse('$API_BASE/prediction/bets'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((b) => Bet.fromJson(b)).toList();
    }
    return [];
  }
}

class PredictionMarket {
  final String id;
  final String question;
  final String description;
  final List<String> outcomes;
  final double totalVolume;
  final String endTime;
  final String status;
  final String? result;
  
  PredictionMarket({
    required this.id,
    required this.question,
    required this.description,
    required this.outcomes,
    required this.totalVolume,
    required this.endTime,
    required this.status,
    this.result,
  });
  
  factory PredictionMarket.fromJson(Map<String, dynamic> json) {
    return PredictionMarket(
      id: json['id'] ?? '',
      question: json['question'] ?? '',
      description: json['description'] ?? '',
      outcomes: List<String>.from(json['outcomes'] ?? []),
      totalVolume: (json['totalVolume'] ?? 0).toDouble(),
      endTime: json['endTime'] ?? '',
      status: json['status'] ?? 'ACTIVE',
      result: json['result'],
    );
  }
}

class Bet {
  final String id;
  final String marketId;
  final String outcome;
  final double amount;
  final double potentialWin;
  final String status;
  final DateTime createdAt;
  
  Bet({
    required this.id,
    required this.marketId,
    required this.outcome,
    required this.amount,
    required this.potentialWin,
    required this.status,
    required this.createdAt,
  });
  
  factory Bet.fromJson(Map<String, dynamic> json) {
    return Bet(
      id: json['id'] ?? '',
      marketId: json['marketId'] ?? '',
      outcome: json['outcome'] ?? '',
      amount: (json['amount'] ?? 0).toDouble(),
      potentialWin: (json['potentialWin'] ?? 0).toDouble(),
      status: json['status'] ?? 'PENDING',
      createdAt: DateTime.parse(json['createdAt']),
    );
  }
}

// RWA Trading Service
class RWATradingService {
  static const String API_BASE = 'http://localhost:8443/api/v1';
  String? _token;
  
  RWATradingService({String? token}) : _token = token;
  
  Map<String, String> get _headers => {
    'Content-Type': 'application/json',
    if (_token != null) 'Authorization': 'Bearer $_token',
  };
  
  // Get RWAs
  Future<List<RWA>> getRWAs() async {
    final response = await http.get(
      Uri.parse('$API_BASE/rwa/list'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((r) => RWA.fromJson(r)).toList();
    }
    return [];
  }
  
  // Buy RWA
  Future<RWAOrder> buyRWA(String rwaId, double amount) async {
    final response = await http.post(
      Uri.parse('$API_BASE/rwa/$rwaId/buy'),
      headers: _headers,
      body: json.encode({'amount': amount}),
    );
    
    if (response.statusCode == 201) {
      final data = json.decode(response.body);
      return RWAOrder.fromJson(data['data']);
    }
    throw Exception('Failed to buy');
  }
  
  // Sell RWA
  Future<RWAOrder> sellRWA(String rwaId, double amount) async {
    final response = await http.post(
      Uri.parse('$API_BASE/rwa/$rwaId/sell'),
      headers: _headers,
      body: json.encode({'amount': amount}),
    );
    
    if (response.statusCode == 201) {
      final data = json.decode(response.body);
      return RWAOrder.fromJson(data['data']);
    }
    throw Exception('Failed to sell');
  }
  
  // Get user holdings
  Future<List<RWAHolding>> getUserHoldings() async {
    final response = await http.get(
      Uri.parse('$API_BASE/rwa/holdings'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((h) => RWAHolding.fromJson(h)).toList();
    }
    return [];
  }
}

class RWA {
  final String id;
  final String name;
  final String type;
  final String description;
  final String assetAddress;
  final double price;
  final double totalSupply;
  final double marketCap;
  
  RWA({
    required this.id,
    required this.name,
    required this.type,
    required this.description,
    required this.assetAddress,
    required this.price,
    required this.totalSupply,
    required this.marketCap,
  });
  
  factory RWA.fromJson(Map<String, dynamic> json) {
    return RWA(
      id: json['id'] ?? '',
      name: json['name'] ?? '',
      type: json['type'] ?? '',
      description: json['description'] ?? '',
      assetAddress: json['assetAddress'] ?? '',
      price: (json['price'] ?? 0).toDouble(),
      totalSupply: (json['totalSupply'] ?? 0).toDouble(),
      marketCap: (json['marketCap'] ?? 0).toDouble(),
    );
  }
}

class RWAOrder {
  final String id;
  final String rwaId;
  final String type;
  final double amount;
  final double price;
  final String status;
  final DateTime createdAt;
  
  RWAOrder({
    required this.id,
    required this.rwaId,
    required this.type,
    required this.amount,
    required this.price,
    required this.status,
    required this.createdAt,
  });
  
  factory RWAOrder.fromJson(Map<String, dynamic> json) {
    return RWAOrder(
      id: json['id'] ?? '',
      rwaId: json['rwaId'] ?? '',
      type: json['type'] ?? '',
      amount: (json['amount'] ?? 0).toDouble(),
      price: (json['price'] ?? 0).toDouble(),
      status: json['status'] ?? '',
      createdAt: DateTime.parse(json['createdAt']),
    );
  }
}

class RWAHolding {
  final String rwaId;
  final String name;
  final double amount;
  final double value;
  
  RWAHolding({
    required this.rwaId,
    required this.name,
    required this.amount,
    required this.value,
  });
  
  factory RWAHolding.fromJson(Map<String, dynamic> json) {
    return RWAHolding(
      rwaId: json['rwaId'] ?? '',
      name: json['name'] ?? '',
      amount: (json['amount'] ?? 0).toDouble(),
      value: (json['value'] ?? 0).toDouble(),
    );
  }
}

// Gas Tracker Service
class GasTrackerService {
  static const String API_BASE = 'http://localhost:8443/api/v1';
  
  Future<GasPrice> getGasPrice(String chain) async {
    final response = await http.get(
      Uri.parse('$API_BASE/gas/price?chain=$chain'),
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return GasPrice.fromJson(data['data']);
    }
    throw Exception('Failed to get gas price');
  }
  
  Future<GasEstimate> estimateGas(String chain, String to, String data) async {
    final response = await http.post(
      Uri.parse('$API_BASE/gas/estimate'),
      headers: {'Content-Type': 'application/json'},
      body: json.encode({'chain': chain, 'to': to, 'data': data}),
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return GasEstimate.fromJson(data['data']);
    }
    throw Exception('Failed to estimate gas');
  }
}

class GasPrice {
  final String chain;
  final double slow;
  final double standard;
  final double fast;
  final double baseFee;
  
  GasPrice({
    required this.chain,
    required this.slow,
    required this.standard,
    required this.fast,
    required this.baseFee,
  });
  
  factory GasPrice.fromJson(Map<String, dynamic> json) {
    return GasPrice(
      chain: json['chain'] ?? '',
      slow: (json['slow'] ?? 0).toDouble(),
      standard: (json['standard'] ?? 0).toDouble(),
      fast: (json['fast'] ?? 0).toDouble(),
      baseFee: (json['baseFee'] ?? 0).toDouble(),
    );
  }
}

class GasEstimate {
  final double gasLimit;
  final double gasPrice;
  final double totalFee;
  
  GasEstimate({
    required this.gasLimit,
    required this.gasPrice,
    required this.totalFee,
  });
  
  factory GasEstimate.fromJson(Map<String, dynamic> json) {
    return GasEstimate(
      gasLimit: (json['gasLimit'] ?? 0).toDouble(),
      gasPrice: (json['gasPrice'] ?? 0).toDouble(),
      totalFee: (json['totalFee'] ?? 0).toDouble(),
    );
  }
}

// Orderbook Service
class OrderbookService {
  static const String API_BASE = 'http://localhost:8443/api/v1';
  
  Future<Orderbook> getOrderbook(String symbol) async {
    final response = await http.get(
      Uri.parse('$API_BASE/orderbook/$symbol'),
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return Orderbook.fromJson(data['data']);
    }
    throw Exception('Failed to get orderbook');
  }
  
  Future<bool> placeLimitOrder(String symbol, String side, double price, double quantity) async {
    final response = await http.post(
      Uri.parse('$API_BASE/orderbook/limit'),
      headers: {'Content-Type': 'application/json'},
      body: json.encode({
        'symbol': symbol, 'side': side, 'price': price, 'quantity': quantity
      }),
    );
    
    return response.statusCode == 201;
  }
}

class Orderbook {
  final String symbol;
  final List<OrderLevel> bids;
  final List<OrderLevel> asks;
  
  Orderbook({required this.symbol, required this.bids, required this.asks});
  
  factory Orderbook.fromJson(Map<String, dynamic> json) {
    return Orderbook(
      symbol: json['symbol'] ?? '',
      bids: (json['bids'] as List).map((b) => OrderLevel.fromJson(b)).toList(),
      asks: (json['asks'] as List).map((a) => OrderLevel.fromJson(a)).toList(),
    );
  }
}

class OrderLevel {
  final double price;
  final double quantity;
  
  OrderLevel({required this.price, required this.quantity});
  
  factory OrderLevel.fromJson(Map<String, dynamic> json) {
    return OrderLevel(
      price: (json['price'] ?? 0).toDouble(),
      quantity: (json['quantity'] ?? 0).toDouble(),
    );
  }
}

// TWAP Service
class TWAPService {
  static const String API_BASE = 'http://localhost:8443/api/v1';
  String? _token;
  
  TWAPService({String? token}) : _token = token;
  
  Map<String, String> get _headers => {
    'Content-Type': 'application/json',
    if (_token != null) 'Authorization': 'Bearer $_token',
  };
  
  Future<TWAPOrder> createTWAP(String symbol, double totalAmount, int intervals, String side) async {
    final response = await http.post(
      Uri.parse('$API_BASE/twap/create'),
      headers: _headers,
      body: json.encode({
        'symbol': symbol, 'totalAmount': totalAmount, 'intervals': intervals, 'side': side
      }),
    );
    
    if (response.statusCode == 201) {
      final data = json.decode(response.body);
      return TWAPOrder.fromJson(data['data']);
    }
    throw Exception('Failed to create TWAP');
  }
  
  Future<bool> cancelTWAP(String orderId) async {
    final response = await http.post(
      Uri.parse('$API_BASE/twap/$orderId/cancel'),
      headers: _headers,
    );
    
    return response.statusCode == 200;
  }
  
  Future<List<TWAPExecution>> getExecutions(String orderId) async {
    final response = await http.get(
      Uri.parse('$API_BASE/twap/$orderId/executions'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((e) => TWAPExecution.fromJson(e)).toList();
    }
    return [];
  }
}

class TWAPOrder {
  final String id;
  final String symbol;
  final double totalAmount;
  final int intervals;
  final int executedIntervals;
  final double executedAmount;
  final String side;
  final String status;
  
  TWAPOrder({
    required this.id,
    required this.symbol,
    required this.totalAmount,
    required this.intervals,
    required this.executedIntervals,
    required this.executedAmount,
    required this.side,
    required this.status,
  });
  
  factory TWAPOrder.fromJson(Map<String, dynamic> json) {
    return TWAPOrder(
      id: json['id'] ?? '',
      symbol: json['symbol'] ?? '',
      totalAmount: (json['totalAmount'] ?? 0).toDouble(),
      intervals: json['intervals'] ?? 0,
      executedIntervals: json['executedIntervals'] ?? 0,
      executedAmount: (json['executedAmount'] ?? 0).toDouble(),
      side: json['side'] ?? '',
      status: json['status'] ?? '',
    );
  }
}

class TWAPExecution {
  final String orderId;
  final double price;
  final double amount;
  final DateTime executedAt;
  
  TWAPExecution({required this.orderId, required this.price, required this.amount, required this.executedAt});
  
  factory TWAPExecution.fromJson(Map<String, dynamic> json) {
    return TWAPExecution(
      orderId: json['orderId'] ?? '',
      price: (json['price'] ?? 0).toDouble(),
      amount: (json['amount'] ?? 0).toDouble(),
      executedAt: DateTime.parse(json['executedAt']),
    );
  }
}

// Intent Routing Service
class IntentRoutingService {
  static const String API_BASE = 'http://localhost:8443/api/v1';
  
  Future<IntentRoute> findBestRoute(String intent) async {
    final response = await http.post(
      Uri.parse('$API_BASE/intent/route'),
      headers: {'Content-Type': 'application/json'},
      body: json.encode({'intent': intent}),
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return IntentRoute.fromJson(data['data']);
    }
    throw Exception('Failed to find route');
  }
}

class IntentRoute {
  final String intent;
  final List<String> steps;
  final double totalValue;
  final double estimatedOutput;
  final double gasFee;
  
  IntentRoute({required this.intent, required this.steps, required this.totalValue, required this.estimatedOutput, required this.gasFee});
  
  factory IntentRoute.fromJson(Map<String, dynamic> json) {
    return IntentRoute(
      intent: json['intent'] ?? '',
      steps: List<String>.from(json['steps'] ?? []),
      totalValue: (json['totalValue'] ?? 0).toDouble(),
      estimatedOutput: (json['estimatedOutput'] ?? 0).toDouble(),
      gasFee: (json['gasFee'] ?? 0).toDouble(),
    );
  }
}

// Security Scanner Service
class SecurityScannerService {
  static const String API_BASE = 'http://localhost:8443/api/v1';
  
  Future<SecurityReport> scanContract(String address, String chain) async {
    final response = await http.get(
      Uri.parse('$API_BASE/security/scan?address=$address&chain=$chain'),
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return SecurityReport.fromJson(data['data']);
    }
    throw Exception('Failed to scan');
  }
  
  Future<List<SecurityAlert>> getAlerts(String address) async {
    final response = await http.get(
      Uri.parse('$API_BASE/security/alerts?address=$address'),
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((a) => SecurityAlert.fromJson(a)).toList();
    }
    return [];
  }
}

class SecurityReport {
  final String address;
  final String chain;
  final String riskLevel;
  final double score;
  final List<String> issues;
  final List<String> recommendations;
  
  SecurityReport({required this.address, required this.chain, required this.riskLevel, required this.score, required this.issues, required this.recommendations});
  
  factory SecurityReport.fromJson(Map<String, dynamic> json) {
    return SecurityReport(
      address: json['address'] ?? '',
      chain: json['chain'] ?? '',
      riskLevel: json['riskLevel'] ?? 'UNKNOWN',
      score: (json['score'] ?? 0).toDouble(),
      issues: List<String>.from(json['issues'] ?? []),
      recommendations: List<String>.from(json['recommendations'] ?? []),
    );
  }
}

class SecurityAlert {
  final String id;
  final String type;
  final String severity;
  final String description;
  final DateTime createdAt;
  
  SecurityAlert({required this.id, required this.type, required this.severity, required this.description, required this.createdAt});
  
  factory SecurityAlert.fromJson(Map<String, dynamic> json) {
    return SecurityAlert(
      id: json['id'] ?? '',
      type: json['type'] ?? '',
      severity: json['severity'] ?? '',
      description: json['description'] ?? '',
      createdAt: DateTime.parse(json['createdAt']),
    );
  }
}
