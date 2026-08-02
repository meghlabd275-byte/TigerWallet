import 'dart:convert';
import 'package:http/http.dart' as http;

/// DeFi Service for Flutter App
/// Production-ready DeFi protocol integrations
class DeFiService {
  static final DeFiService _instance = DeFiService._internal();
  factory DeFiService() => _instance;
  DeFiService._internal();

  final String _baseUrl = 'https://api.tigerwallet.com/v1/defi';
  
  // Aave Methods
  
  /// Get Aave pools
  Future<List<DeFiPool>> getAavePools({String chain = 'ethereum'}) async {
    final response = await http.get(
      Uri.parse('$_baseUrl/aave/pools?chain=$chain'),
      headers: {'Content-Type': 'application/json'},
    );
    
    if (response.statusCode == 200) {
      final List<dynamic> data = json.decode(response.body);
      return data.map((e) => DeFiPool.fromJson(e)).toList();
    }
    return [];
  }
  
  /// Supply to Aave
  Future<String> supplyToAave({
    required String walletAddress,
    required String poolAddress,
    required String tokenAddress,
    required String amount,
    required String chain,
  }) async {
    final response = await http.post(
      Uri.parse('$_baseUrl/aave/supply'),
      headers: {'Content-Type': 'application/json'},
      body: json.encode({
        'wallet_address': walletAddress,
        'pool_address': poolAddress,
        'token_address': tokenAddress,
        'amount': amount,
        'chain': chain,
      }),
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return data['txHash'];
    }
    throw Exception('Transaction failed');
  }
  
  /// Borrow from Aave
  Future<String> borrowFromAave({
    required String walletAddress,
    required String poolAddress,
    required String tokenAddress,
    required String amount,
    required int interestRateMode,
    required String chain,
  }) async {
    final response = await http.post(
      Uri.parse('$_baseUrl/aave/borrow'),
      headers: {'Content-Type': 'application/json'},
      body: json.encode({
        'wallet_address': walletAddress,
        'pool_address': poolAddress,
        'token_address': tokenAddress,
        'amount': amount,
        'interest_rate_mode': interestRateMode,
        'chain': chain,
      }),
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return data['txHash'];
    }
    throw Exception('Transaction failed');
  }
  
  // Uniswap Methods
  
  /// Get swap quote
  Future<SwapQuote> getSwapQuote({
    required String tokenIn,
    required String tokenOut,
    required String amount,
    String chain = 'ethereum',
  }) async {
    final response = await http.get(
      Uri.parse('$_baseUrl/uniswap/quote?tokenIn=$tokenIn&tokenOut=$tokenOut&amount=$amount&chain=$chain'),
      headers: {'Content-Type': 'application/json'},
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return SwapQuote.fromJson(data);
    }
    throw Exception('Failed to get quote');
  }
  
  /// Execute swap
  Future<String> executeSwap({
    required String walletAddress,
    required String tokenIn,
    required String tokenOut,
    required String amount,
    required String minOutput,
    required String chain,
  }) async {
    final response = await http.post(
      Uri.parse('$_baseUrl/uniswap/swap'),
      headers: {'Content-Type': 'application/json'},
      body: json.encode({
        'wallet_address': walletAddress,
        'token_in': tokenIn,
        'token_out': tokenOut,
        'amount': amount,
        'min_output': minOutput,
        'chain': chain,
      }),
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return data['txHash'];
    }
    throw Exception('Swap failed');
  }
  
  // Compound Methods
  
  /// Get Compound pools
  Future<List<DeFiPool>> getCompoundPools() async {
    final response = await http.get(
      Uri.parse('$_baseUrl/compound/pools'),
      headers: {'Content-Type': 'application/json'},
    );
    
    if (response.statusCode == 200) {
      final List<dynamic> data = json.decode(response.body);
      return data.map((e) => DeFiPool.fromJson(e)).toList();
    }
    return [];
  }
  
  // Yearn Vaults
  
  /// Get Yearn vaults
  Future<List<DeFiPool>> getYearnVaults() async {
    final response = await http.get(
      Uri.parse('$_baseUrl/yearn/vaults'),
      headers: {'Content-Type': 'application/json'},
    );
    
    if (response.statusCode == 200) {
      final List<dynamic> data = json.decode(response.body);
      return data.map((e) => DeFiPool.fromJson(e)).toList();
    }
    return [];
  }
  
  // Portfolio
  
  /// Get all positions for a wallet
  Future<List<DeFiPosition>> getAllPositions(String walletAddress) async {
    final response = await http.get(
      Uri.parse('$_baseUrl/positions/$walletAddress'),
      headers: {'Content-Type': 'application/json'},
    );
    
    if (response.statusCode == 200) {
      final List<dynamic> data = json.decode(response.body);
      return data.map((e) => DeFiPosition.fromJson(e)).toList();
    }
    return [];
  }
}

// Models

class DeFiPool {
  final String id;
  final String protocol;
  final String chain;
  final TokenInfo token0;
  final TokenInfo? token1;
  final double tvl;
  final double apy;
  final double? rewardsApy;
  final String poolAddress;

  DeFiPool({
    required this.id,
    required this.protocol,
    required this.chain,
    required this.token0,
    this.token1,
    required this.tvl,
    required this.apy,
    this.rewardsApy,
    required this.poolAddress,
  });

  factory DeFiPool.fromJson(Map<String, dynamic> json) {
    return DeFiPool(
      id: json['id'] ?? '',
      protocol: json['protocol'] ?? '',
      chain: json['chain'] ?? '',
      token0: TokenInfo.fromJson(json['token0'] ?? {}),
      token1: json['token1'] != null ? TokenInfo.fromJson(json['token1']) : null,
      tvl: (json['tvl'] ?? 0).toDouble(),
      apy: (json['apy'] ?? 0).toDouble(),
      rewardsApy: json['rewardsApy']?.toDouble(),
      poolAddress: json['poolAddress'] ?? '',
    );
  }
}

class TokenInfo {
  final String address;
  final String symbol;
  final String name;
  final int decimals;
  final String? logoUrl;

  TokenInfo({
    required this.address,
    required this.symbol,
    required this.name,
    required this.decimals,
    this.logoUrl,
  });

  factory TokenInfo.fromJson(Map<String, dynamic> json) {
    return TokenInfo(
      address: json['address'] ?? '',
      symbol: json['symbol'] ?? '',
      name: json['name'] ?? '',
      decimals: json['decimals'] ?? 18,
      logoUrl: json['logoUrl'],
    );
  }
}

class DeFiPosition {
  final String id;
  final String protocol;
  final String chain;
  final String poolAddress;
  final TokenInfo token0;
  final TokenInfo? token1;
  final double deposited0;
  final double? deposited1;
  final double valueUsd;
  final double apy;
  final List<Reward>? rewards;

  DeFiPosition({
    required this.id,
    required this.protocol,
    required this.chain,
    required this.poolAddress,
    required this.token0,
    this.token1,
    required this.deposited0,
    this.deposited1,
    required this.valueUsd,
    required this.apy,
    this.rewards,
  });

  factory DeFiPosition.fromJson(Map<String, dynamic> json) {
    return DeFiPosition(
      id: json['id'] ?? '',
      protocol: json['protocol'] ?? '',
      chain: json['chain'] ?? '',
      poolAddress: json['poolAddress'] ?? '',
      token0: TokenInfo.fromJson(json['token0'] ?? {}),
      token1: json['token1'] != null ? TokenInfo.fromJson(json['token1']) : null,
      deposited0: (json['deposited0'] ?? 0).toDouble(),
      deposited1: json['deposited1']?.toDouble(),
      valueUsd: (json['valueUsd'] ?? 0).toDouble(),
      apy: (json['apy'] ?? 0).toDouble(),
      rewards: json['rewards'] != null 
          ? (json['rewards'] as List).map((e) => Reward.fromJson(e)).toList()
          : null,
    );
  }
}

class Reward {
  final TokenInfo token;
  final double amount;
  final double valueUsd;
  final double apy;

  Reward({
    required this.token,
    required this.amount,
    required this.valueUsd,
    required this.apy,
  });

  factory Reward.fromJson(Map<String, dynamic> json) {
    return Reward(
      token: TokenInfo.fromJson(json['token'] ?? {}),
      amount: (json['amount'] ?? 0).toDouble(),
      valueUsd: (json['valueUsd'] ?? 0).toDouble(),
      apy: (json['apy'] ?? 0).toDouble(),
    );
  }
}

class SwapQuote {
  final TokenInfo fromToken;
  final TokenInfo toToken;
  final double fromAmount;
  final double toAmount;
  final double toAmountMin;
  final double priceImpact;
  final List<String> route;
  final double gasCostUsd;
  final String protocol;

  SwapQuote({
    required this.fromToken,
    required this.toToken,
    required this.fromAmount,
    required this.toAmount,
    required this.toAmountMin,
    required this.priceImpact,
    required this.route,
    required this.gasCostUsd,
    required this.protocol,
  });

  factory SwapQuote.fromJson(Map<String, dynamic> json) {
    return SwapQuote(
      fromToken: TokenInfo.fromJson(json['fromToken'] ?? {}),
      toToken: TokenInfo.fromJson(json['toToken'] ?? {}),
      fromAmount: (json['fromAmount'] ?? 0).toDouble(),
      toAmount: (json['toAmount'] ?? 0).toDouble(),
      toAmountMin: (json['toAmountMin'] ?? 0).toDouble(),
      priceImpact: (json['priceImpact'] ?? 0).toDouble(),
      route: List<String>.from(json['route'] ?? []),
      gasCostUsd: (json['gasCostUsd'] ?? 0).toDouble(),
      protocol: json['protocol'] ?? '',
    );
  }
}
