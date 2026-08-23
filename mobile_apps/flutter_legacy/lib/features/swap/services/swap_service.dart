// Swap Service - DEX Aggregator & Trading
// Complete swap functionality with multi-hop routing

import '../../services/api_service.dart';

class SwapService {
  final ApiService _api = ApiService.instance;
  
  // Get swap quote
  Future<SwapQuote> getQuote({
    required String fromToken,
    required String toToken,
    required String amount,
    required String fromChainId,
    required String toChainId,
  }) async {
    final response = await _api.post('/swap/quote', body: {
      'fromToken': fromToken,
      'toToken': toToken,
      'amount': amount,
      'fromChainId': fromChainId,
      'toChainId': toChainId,
    });
    
    if (response.success) {
      return SwapQuote.fromJson(response.data);
    }
    throw Exception(response.error);
  }
  
  // Execute swap
  Future<SwapResult> executeSwap({
    required String quoteId,
    required String fromAddress,
    required String slippageTolerance,
  }) async {
    final response = await _api.post('/swap/execute', body: {
      'quoteId': quoteId,
      'fromAddress': fromAddress,
      'slippageTolerance': slippageTolerance,
    });
    
    if (response.success) {
      return SwapResult.fromJson(response.data);
    }
    throw Exception(response.error);
  }
  
  // Get swap status
  Future<SwapStatus> getStatus(String swapId) async {
    final response = await _api.get('/swap/status/$swapId');
    
    if (response.success) {
      return SwapStatus.fromJson(response.data);
    }
    throw Exception(response.error);
  }
  
  // Get supported tokens
  Future<List<SwapToken>> getSupportedTokens(String chainId) async {
    final response = await _api.get('/swap/tokens', queryParams: {'chainId': chainId});
    
    if (response.success) {
      return (response.data as List).map((t) => SwapToken.fromJson(t)).toList();
    }
    return [];
  }
  
  // Get popular pairs
  Future<List<SwapPair>> getPopularPairs() async {
    final response = await _api.get('/swap/popular-pairs');
    
    if (response.success) {
      return (response.data as List).map((p) => SwapPair.fromJson(p)).toList();
    }
    return [];
  }
}

class SwapQuote {
  final String id;
  final String fromToken;
  final String toToken;
  final String fromAmount;
  final String toAmount;
  final String fromChainId;
  final String toChainId;
  final String priceImpact;
  final String minimumReceived;
  final String route;
  final List<SwapRouteStep> routeSteps;
  final String estimatedTime;
  final String gasEstimate;
  
  SwapQuote({
    required this.id,
    required this.fromToken,
    required this.toToken,
    required this.fromAmount,
    required this.toAmount,
    required this.fromChainId,
    required this.toChainId,
    required this.priceImpact,
    required this.minimumReceived,
    required this.route,
    required this.routeSteps,
    required this.estimatedTime,
    required this.gasEstimate,
  });
  
  factory SwapQuote.fromJson(Map<String, dynamic> json) {
    return SwapQuote(
      id: json['id'],
      fromToken: json['fromToken'],
      toToken: json['toToken'],
      fromAmount: json['fromAmount'],
      toAmount: json['toAmount'],
      fromChainId: json['fromChainId'],
      toChainId: json['toChainId'],
      priceImpact: json['priceImpact'],
      minimumReceived: json['minimumReceived'],
      route: json['route'],
      routeSteps: (json['routeSteps'] as List?)
          ?.map((s) => SwapRouteStep.fromJson(s))
          .toList() ?? [],
      estimatedTime: json['estimatedTime'],
      gasEstimate: json['gasEstimate'],
    );
  }
}

class SwapRouteStep {
  final String protocol;
  final String fromToken;
  final String toToken;
  final String fromAmount;
  final String toAmount;
  
  SwapRouteStep({
    required this.protocol,
    required this.fromToken,
    required this.toToken,
    required this.fromAmount,
    required this.toAmount,
  });
  
  factory SwapRouteStep.fromJson(Map<String, dynamic> json) {
    return SwapRouteStep(
      protocol: json['protocol'],
      fromToken: json['fromToken'],
      toToken: json['toToken'],
      fromAmount: json['fromAmount'],
      toAmount: json['toAmount'],
    );
  }
}

class SwapResult {
  final String swapId;
  final String txHash;
  final String status;
  
  SwapResult({
    required this.swapId,
    required this.txHash,
    required this.status,
  });
  
  factory SwapResult.fromJson(Map<String, dynamic> json) {
    return SwapResult(
      swapId: json['swapId'],
      txHash: json['txHash'],
      status: json['status'],
    );
  }
}

enum SwapStatus {
  pending,
  confirmed,
  failed,
}

extension SwapStatusExtension on SwapStatus {
  static SwapStatus fromString(String status) {
    switch (status.toLowerCase()) {
      case 'confirmed':
        return SwapStatus.confirmed;
      case 'failed':
        return SwapStatus.failed;
      default:
        return SwapStatus.pending;
    }
  }
}

class SwapToken {
  final String address;
  final String symbol;
  final String name;
  final String decimals;
  final String logoUrl;
  final bool isVerified;
  
  SwapToken({
    required this.address,
    required this.symbol,
    required this.name,
    required this.decimals,
    required this.logoUrl,
    required this.isVerified,
  });
  
  factory SwapToken.fromJson(Map<String, dynamic> json) {
    return SwapToken(
      address: json['address'],
      symbol: json['symbol'],
      name: json['name'],
      decimals: json['decimals'],
      logoUrl: json['logoUrl'],
      isVerified: json['isVerified'] ?? false,
    );
  }
}

class SwapPair {
  final String fromToken;
  final String toToken;
  final String fromChainId;
  final String toChainId;
  final String liquidity;
  final String volume24h;
  
  SwapPair({
    required this.fromToken,
    required this.toToken,
    required this.fromChainId,
    required this.toChainId,
    required this.liquidity,
    required this.volume24h,
  });
  
  factory SwapPair.fromJson(Map<String, dynamic> json) {
    return SwapPair(
      fromToken: json['fromToken'],
      toToken: json['toToken'],
      fromChainId: json['fromChainId'],
      toChainId: json['toChainId'],
      liquidity: json['liquidity'],
      volume24h: json['volume24h'],
    );
  }
}
