import 'dart:convert';
import 'package:http/http.dart' as http;

/// Cross-Chain Intent Router Service for Flutter App
/// Production-ready intent-based cross-chain transactions
class CrossChainIntentService {
  static final CrossChainIntentService _instance = CrossChainIntentService._internal();
  factory CrossChainIntentService() => _instance;
  CrossChainIntentService._internal();

  final String _baseUrl = 'http://localhost:8443/api/v1/intents';
  
  /// Get quotes for cross-chain intent
  Future<List<IntentQuote>> getQuotes({
    required int sourceChain,
    required int destinationChain,
    required String fromToken,
    required String toToken,
    required String fromAmount,
    required String sender,
  }) async {
    final response = await http.post(
      Uri.parse('$_baseUrl/quotes'),
      headers: {'Content-Type': 'application/json'},
      body: json.encode({
        'source_chain': sourceChain,
        'destination_chain': destinationChain,
        'from_token': fromToken,
        'to_token': toToken,
        'from_amount': fromAmount,
        'sender': sender,
      }),
    );
    
    if (response.statusCode == 200) {
      final List<dynamic> data = json.decode(response.body);
      return data.map((e) => IntentQuote.fromJson(e)).toList();
    }
    return [];
  }
  
  /// Get best quote
  Future<IntentQuote?> getBestQuote({
    required int sourceChain,
    required int destinationChain,
    required String fromToken,
    required String toToken,
    required String fromAmount,
    required String sender,
  }) async {
    final quotes = await getQuotes(
      sourceChain: sourceChain,
      destinationChain: destinationChain,
      fromToken: fromToken,
      toToken: toToken,
      fromAmount: fromAmount,
      sender: sender,
    );
    
    if (quotes.isEmpty) return null;
    
    // Sort by output amount (highest first)
    quotes.sort((a, b) => b.toAmount.compareTo(a.toAmount));
    return quotes.first;
  }
  
  /// Create a cross-chain intent
  Future<CrossChainIntent> createIntent({
    required int sourceChain,
    required int destinationChain,
    required String fromToken,
    required String toToken,
    required String fromAmount,
    required String toAmountMin,
    required String solver,
    required String solverSignature,
    int? deadline,
  }) async {
    final response = await http.post(
      Uri.parse('$_baseUrl/intents'),
      headers: {'Content-Type': 'application/json'},
      body: json.encode({
        'source_chain': sourceChain,
        'destination_chain': destinationChain,
        'from_token': fromToken,
        'to_token': toToken,
        'from_amount': fromAmount,
        'to_amount_min': toAmountMin,
        'solver': solver,
        'solver_signature': solverSignature,
        'deadline': deadline ?? (DateTime.now().millisecondsSinceEpoch ~/ 1000) + 3600,
      }),
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return CrossChainIntent.fromJson(data);
    }
    throw Exception('Failed to create intent');
  }
  
  /// Get intent status
  Future<IntentExecution?> getIntentStatus(String intentId) async {
    final response = await http.get(
      Uri.parse('$_baseUrl/intents/$intentId/status'),
      headers: {'Content-Type': 'application/json'},
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return IntentExecution.fromJson(data);
    }
    return null;
  }
  
  /// Monitor intent status (polling)
  Stream<IntentExecution> monitorIntent(String intentId, {int pollIntervalMs = 5000}) async* {
    while (true) {
      final status = await getIntentStatus(intentId);
      if (status != null) {
        yield status;
        
        // Stop monitoring if terminal state
        if (status.status == 'filled' || status.status == 'expired' || status.status == 'failed') {
          break;
        }
      }
      
      await Future.delayed(Duration(milliseconds: pollIntervalMs));
    }
  }
}

// Models

class CrossChainIntent {
  final String id;
  final String sender;
  final int sourceChain;
  final int destinationChain;
  final String fromToken;
  final String toToken;
  final String fromAmount;
  final String toAmountMin;
  final int deadline;
  final String signature;
  final int fillDeadline;

  CrossChainIntent({
    required this.id,
    required this.sender,
    required this.sourceChain,
    required this.destinationChain,
    required this.fromToken,
    required this.toToken,
    required this.fromAmount,
    required this.toAmountMin,
    required this.deadline,
    required this.signature,
    required this.fillDeadline,
  });

  factory CrossChainIntent.fromJson(Map<String, dynamic> json) {
    return CrossChainIntent(
      id: json['id'] ?? '',
      sender: json['sender'] ?? '',
      sourceChain: json['sourceChain'] ?? 0,
      destinationChain: json['destinationChain'] ?? 0,
      fromToken: json['fromToken'] ?? '',
      toToken: json['toToken'] ?? '',
      fromAmount: json['fromAmount'] ?? '',
      toAmountMin: json['toAmountMin'] ?? '',
      deadline: json['deadline'] ?? 0,
      signature: json['signature'] ?? '',
      fillDeadline: json['fillDeadline'] ?? 0,
    );
  }
}

class IntentQuote {
  final String intentId;
  final String solver;
  final String solverLogo;
  final String fromToken;
  final String toToken;
  final String fromAmount;
  final String toAmount;
  final String toAmountMin;
  final double priceImpact;
  final String gasCost;
  final int estimatedTime;
  final List<IntentRouteStep> route;

  IntentQuote({
    required this.intentId,
    required this.solver,
    required this.solverLogo,
    required this.fromToken,
    required this.toToken,
    required this.fromAmount,
    required this.toAmount,
    required this.toAmountMin,
    required this.priceImpact,
    required this.gasCost,
    required this.estimatedTime,
    required this.route,
  });

  factory IntentQuote.fromJson(Map<String, dynamic> json) {
    return IntentQuote(
      intentId: json['intentId'] ?? '',
      solver: json['solver'] ?? '',
      solverLogo: json['solverLogo'] ?? '',
      fromToken: json['fromToken'] ?? '',
      toToken: json['toToken'] ?? '',
      fromAmount: json['fromAmount'] ?? '',
      toAmount: json['toAmount'] ?? '',
      toAmountMin: json['toAmountMin'] ?? '',
      priceImpact: (json['priceImpact'] ?? 0).toDouble(),
      gasCost: json['gasCost'] ?? '',
      estimatedTime: json['estimatedTime'] ?? 0,
      route: json['route'] != null 
          ? (json['route'] as List).map((e) => IntentRouteStep.fromJson(e)).toList()
          : [],
    );
  }
}

class IntentRouteStep {
  final String protocol;
  final String protocolLogo;
  final String action;
  final String fromToken;
  final String toToken;
  final String fromAmount;
  final String toAmount;

  IntentRouteStep({
    required this.protocol,
    required this.protocolLogo,
    required this.action,
    required this.fromToken,
    required this.toToken,
    required this.fromAmount,
    required this.toAmount,
  });

  factory IntentRouteStep.fromJson(Map<String, dynamic> json) {
    return IntentRouteStep(
      protocol: json['protocol'] ?? '',
      protocolLogo: json['protocolLogo'] ?? '',
      action: json['action'] ?? '',
      fromToken: json['fromToken'] ?? '',
      toToken: json['toToken'] ?? '',
      fromAmount: json['fromAmount'] ?? '',
      toAmount: json['toAmount'] ?? '',
    );
  }
}

class IntentExecution {
  final String transactionHash;
  final String intentId;
  final String status;
  final String? sourceChainTxHash;
  final String? destinationChainTxHash;
  final String? filledAmount;
  final int? fillTimestamp;

  IntentExecution({
    required this.transactionHash,
    required this.intentId,
    required this.status,
    this.sourceChainTxHash,
    this.destinationChainTxHash,
    this.filledAmount,
    this.fillTimestamp,
  });

  factory IntentExecution.fromJson(Map<String, dynamic> json) {
    return IntentExecution(
      transactionHash: json['transactionHash'] ?? '',
      intentId: json['intentId'] ?? '',
      status: json['status'] ?? 'pending',
      sourceChainTxHash: json['sourceChainTxHash'],
      destinationChainTxHash: json['destinationChainTxHash'],
      filledAmount: json['filledAmount'],
      fillTimestamp: json['fillTimestamp'],
    );
  }
}

// Supported Chains

class ChainInfo {
  final int id;
  final String name;
  final String symbol;
  final String color;

  const ChainInfo({
    required this.id,
    required this.name,
    required this.symbol,
    required this.color,
  });

  static const List<ChainInfo> supportedChains = [
    ChainInfo(id: 1, name: 'Ethereum', symbol: 'ETH', color: '#627EEA'),
    ChainInfo(id: 56, name: 'BNB Chain', symbol: 'BNB', color: '#F3BA2F'),
    ChainInfo(id: 137, name: 'Polygon', symbol: 'MATIC', color: '#8247E5'),
    ChainInfo(id: 42161, name: 'Arbitrum', symbol: 'ETH', color: '#28A0F0'),
    ChainInfo(id: 10, name: 'Optimism', symbol: 'ETH', color: '#FF0420'),
    ChainInfo(id: 43114, name: 'Avalanche', symbol: 'AVAX', color: '#E84142'),
    ChainInfo(id: 8453, name: 'Base', symbol: 'ETH', color: '#0052FF'),
    ChainInfo(id: 324, name: 'zkSync Era', symbol: 'ETH', color: '#F3BA2F'),
    ChainInfo(id: 59144, name: 'Linea', symbol: 'ETH', color: '#000000'),
    ChainInfo(id: 534352, name: 'Scroll', symbol: 'ETH', color: '#C4AEF5'),
  ];
}
