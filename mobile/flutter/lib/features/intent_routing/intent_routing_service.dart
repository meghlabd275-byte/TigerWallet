/// Intent Routing Service for Flutter
/// Smart order routing with intent-based trading

import 'dart:convert';
import 'package:http/http.dart' as http;

class IntentRoutingService {
  static const String _baseUrl = 'https://api.tigerwallet.com/v1/intent';
  
  final http.Client _client;
  
  IntentRoutingService({http.Client? client}) : _client = client ?? http.Client();
  
  /// Create a trading intent
  Future<Intent> createIntent({
    required IntentType type,
    required String fromToken,
    required String toToken,
    required double amount,
    double? maxSlippage,
    IntentPriority priority = IntentPriority.medium,
  }) async {
    final response = await _client.post(
      Uri.parse('$_baseUrl/intents'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        'type': type.name,
        'fromToken': fromToken,
        'toToken': toToken,
        'amount': amount,
        'maxSlippage': maxSlippage ?? 1.0,
        'priority': priority.name,
      }),
    );
    
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return Intent.fromJson(data);
    }
    throw Exception('Failed to create intent: ${response.body}');
  }
  
  /// Get intent by ID
  Future<Intent> getIntent(String intentId) async {
    final response = await _client.get(
      Uri.parse('$_baseUrl/intents/$intentId'),
    );
    
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return Intent.fromJson(data);
    }
    throw Exception('Failed to get intent: ${response.body}');
  }
  
  /// Cancel an intent
  Future<bool> cancelIntent(String intentId) async {
    final response = await _client.delete(
      Uri.parse('$_baseUrl/intents/$intentId'),
    );
    
    return response.statusCode == 200;
  }
  
  /// Get user's intents
  Future<List<Intent>> getUserIntents(String address) async {
    final response = await _client.get(
      Uri.parse('$_baseUrl/intents?user=$address'),
    );
    
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return (data['intents'] as List)
          .map((e) => Intent.fromJson(e))
          .toList();
    }
    throw Exception('Failed to get intents: ${response.body}');
  }
  
  /// Find matching intents (for intent-based trading)
  Future<List<IntentMatch>> findMatches(String intentId) async {
    final response = await _client.get(
      Uri.parse('$_baseUrl/intents/$intentId/matches'),
    );
    
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return (data['matches'] as List)
          .map((e) => IntentMatch.fromJson(e))
          .toList();
    }
    throw Exception('Failed to find matches: ${response.body}');
  }
  
  /// Execute a matched intent
  Future<ExecutionResult> executeMatch(String matchId) async {
    final response = await _client.post(
      Uri.parse('$_baseUrl/matches/$matchId/execute'),
      headers: {'Content-Type': 'application/json'},
    );
    
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return ExecutionResult.fromJson(data);
    }
    throw Exception('Failed to execute: ${response.body}');
  }
  
  /// Get routing suggestion
  Future<RoutingSuggestion> getSuggestion({
    required String fromToken,
    required String toToken,
    required double amount,
  }) async {
    final response = await _client.post(
      Uri.parse('$_baseUrl/suggest'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        'fromToken': fromToken,
        'toToken': toToken,
        'amount': amount,
      }),
    );
    
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return RoutingSuggestion.fromJson(data);
    }
    throw Exception('Failed to get suggestion: ${response.body}');
  }
  
  void dispose() {
    _client.close();
  }
}

enum IntentType {
  swap,
  limitOrder,
  stopLoss,
  takeProfit,
}

enum IntentPriority {
  low,
  medium,
  high,
  urgent,
}

class Intent {
  final String intentId;
  final String user;
  final IntentType type;
  final String fromToken;
  final String toToken;
  final double amount;
  final double? limitPrice;
  final double maxSlippage;
  final IntentPriority priority;
  final IntentStatus status;
  final DateTime createdAt;
  final DateTime? expiresAt;
  final List<IntentMatch> matches;
  final ExecutionResult? result;
  
  Intent({
    required this.intentId,
    required this.user,
    required this.type,
    required this.fromToken,
    required this.toToken,
    required this.amount,
    this.limitPrice,
    required this.maxSlippage,
    required this.priority,
    required this.status,
    required this.createdAt,
    this.expiresAt,
    this.matches = const [],
    this.result,
  });
  
  factory Intent.fromJson(Map<String, dynamic> json) {
    return Intent(
      intentId: json['intentId'],
      user: json['user'],
      type: IntentType.values.firstWhere(
        (e) => e.name == json['type'],
        orElse: () => IntentType.swap,
      ),
      fromToken: json['fromToken'],
      toToken: json['toToken'],
      amount: double.parse(json['amount'].toString()),
      limitPrice: json['limitPrice'] != null 
          ? double.parse(json['limitPrice'].toString()) 
          : null,
      maxSlippage: double.parse(json['maxSlippage'].toString()),
      priority: IntentPriority.values.firstWhere(
        (e) => e.name == json['priority'],
        orElse: () => IntentPriority.medium,
      ),
      status: IntentStatus.values.firstWhere(
        (e) => e.name == json['status'],
        orElse: () => IntentStatus.pending,
      ),
      createdAt: DateTime.parse(json['createdAt']),
      expiresAt: json['expiresAt'] != null 
          ? DateTime.parse(json['expiresAt']) 
          : null,
      matches: (json['matches'] as List?)
          ?.map((e) => IntentMatch.fromJson(e))
          .toList() ?? [],
      result: json['result'] != null 
          ? ExecutionResult.fromJson(json['result']) 
          : null,
    );
  }
}

enum IntentStatus {
  pending,
  matching,
  matched,
  executing,
  completed,
  cancelled,
  expired,
}

class IntentMatch {
  final String matchId;
  final String intentId;
  final String counterparty;
  final double price;
  final double amount;
  final double gasSaved;
  final DateTime timestamp;
  
  IntentMatch({
    required this.matchId,
    required this.intentId,
    required this.counterparty,
    required this.price,
    required this.amount,
    required this.gasSaved,
    required this.timestamp,
  });
  
  factory IntentMatch.fromJson(Map<String, dynamic> json) {
    return IntentMatch(
      matchId: json['matchId'],
      intentId: json['intentId'],
      counterparty: json['counterparty'],
      price: double.parse(json['price'].toString()),
      amount: double.parse(json['amount'].toString()),
      gasSaved: double.parse(json['gasSaved'].toString()),
      timestamp: DateTime.parse(json['timestamp']),
    );
  }
}

class ExecutionResult {
  final String txHash;
  final double executedAmount;
  final double receivedAmount;
  final double slippage;
  final double gasUsed;
  final DateTime timestamp;
  
  ExecutionResult({
    required this.txHash,
    required this.executedAmount,
    required this.receivedAmount,
    required this.slippage,
    required this.gasUsed,
    required this.timestamp,
  });
  
  factory ExecutionResult.fromJson(Map<String, dynamic> json) {
    return ExecutionResult(
      txHash: json['txHash'],
      executedAmount: double.parse(json['executedAmount'].toString()),
      receivedAmount: double.parse(json['receivedAmount'].toString()),
      slippage: double.parse(json['slippage'].toString()),
      gasUsed: double.parse(json['gasUsed'].toString()),
      timestamp: DateTime.parse(json['timestamp']),
    );
  }
}

class RoutingSuggestion {
  final List<RouteStep> route;
  final double estimatedOutput;
  final double estimatedGas;
  final double estimatedTime;
  final double priceImpact;
  
  RoutingSuggestion({
    required this.route,
    required this.estimatedOutput,
    required this.estimatedGas,
    required this.estimatedTime,
    required this.priceImpact,
  });
  
  factory RoutingSuggestion.fromJson(Map<String, dynamic> json) {
    return RoutingSuggestion(
      route: (json['route'] as List)
          .map((e) => RouteStep.fromJson(e))
          .toList(),
      estimatedOutput: double.parse(json['estimatedOutput'].toString()),
      estimatedGas: double.parse(json['estimatedGas'].toString()),
      estimatedTime: double.parse(json['estimatedTime'].toString()),
      priceImpact: double.parse(json['priceImpact'].toString()),
    );
  }
}

class RouteStep {
  final String protocol;
  final String fromToken;
  final String toToken;
  final double fromAmount;
  final double toAmount;
  
  RouteStep({
    required this.protocol,
    required this.fromToken,
    required this.toToken,
    required this.fromAmount,
    required this.toAmount,
  });
  
  factory RouteStep.fromJson(Map<String, dynamic> json) {
    return RouteStep(
      protocol: json['protocol'],
      fromToken: json['fromToken'],
      toToken: json['toToken'],
      fromAmount: double.parse(json['fromAmount'].toString()),
      toAmount: double.parse(json['toAmount'].toString()),
    );
  }
}
