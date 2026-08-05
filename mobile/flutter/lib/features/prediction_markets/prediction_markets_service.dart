/// Prediction Markets Service for Flutter
/// Binary options and prediction market trading

import 'dart:convert';
import 'package:http/http.dart' as http;

class PredictionMarketsService {
  static const String _baseUrl = 'https://api.tigerwallet.com/v1/predictions';
  
  final http.Client _client;
  
  PredictionMarketsService({http.Client? client}) : _client = client ?? http.Client();
  
  /// Get all markets
  Future<List<PredictionMarket>> getMarkets({String status = 'active'}) async {
    final response = await _client.get(
      Uri.parse('$_baseUrl/markets?status=$status'),
    );
    
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return (data['markets'] as List)
          .map((e) => PredictionMarket.fromJson(e))
          .toList();
    }
    throw Exception('Failed to get markets: ${response.body}');
  }
  
  /// Get market details
  Future<PredictionMarket> getMarket(String marketId) async {
    final response = await _client.get(
      Uri.parse('$_baseUrl/markets/$marketId'),
    );
    
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return PredictionMarket.fromJson(data);
    }
    throw Exception('Failed to get market: ${response.body}');
  }
  
  /// Place a bet
  Future<PredictionBet> placeBet({
    required String marketId,
    required String outcome,
    required double amount,
    required String userAddress,
  }) async {
    final response = await _client.post(
      Uri.parse('$_baseUrl/bets'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        'marketId': marketId,
        'outcome': outcome,
        'amount': amount,
        'userAddress': userAddress,
      }),
    );
    
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return PredictionBet.fromJson(data);
    }
    throw Exception('Failed to place bet: ${response.body}');
  }
  
  /// Get user's bets
  Future<List<PredictionBet>> getUserBets(String address) async {
    final response = await _client.get(
      Uri.parse('$_baseUrl/bets?user=$address'),
    );
    
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return (data['bets'] as List)
          .map((e) => PredictionBet.fromJson(e))
          .toList();
    }
    throw Exception('Failed to get bets: ${response.body}');
  }
  
  /// Get market odds
  Future<MarketOdds> getOdds(String marketId) async {
    final response = await _client.get(
      Uri.parse('$_baseUrl/markets/$marketId/odds'),
    );
    
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return MarketOdds.fromJson(data);
    }
    throw Exception('Failed to get odds: ${response.body}');
  }
  
  /// Get market history (resolved markets)
  Future<List<MarketResolution>> getMarketHistory(String marketId) async {
    final response = await _client.get(
      Uri.parse('$_baseUrl/markets/$marketId/history'),
    );
    
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return (data['history'] as List)
          .map((e) => MarketResolution.fromJson(e))
          .toList();
    }
    throw Exception('Failed to get history: ${response.body}');
  }
  
  /// Claim winnings
  Future<String> claimWinnings(String betId) async {
    final response = await _client.post(
      Uri.parse('$_baseUrl/bets/$betId/claim'),
      headers: {'Content-Type': 'application/json'},
    );
    
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return data['txHash'];
    }
    throw Exception('Failed to claim: ${response.body}');
  }
  
  void dispose() {
    _client.close();
  }
}

class PredictionMarket {
  final String marketId;
  final String question;
  final String description;
  final String category;
  final String imageUrl;
  final String outcomeA;
  final String outcomeB;
  final double oddsA;
  final double oddsB;
  final double volume;
  final double liquidity;
  final int totalBets;
  final MarketStatus status;
  final DateTime endTime;
  final DateTime? resolutionTime;
  final String? resolvedOutcome;
  
  PredictionMarket({
    required this.marketId,
    required this.question,
    required this.description,
    required this.category,
    required this.imageUrl,
    required this.outcomeA,
    required this.outcomeB,
    required this.oddsA,
    required this.oddsB,
    required this.volume,
    required this.liquidity,
    required this.totalBets,
    required this.status,
    required this.endTime,
    this.resolutionTime,
    this.resolvedOutcome,
  });
  
  factory PredictionMarket.fromJson(Map<String, dynamic> json) {
    return PredictionMarket(
      marketId: json['marketId'],
      question: json['question'],
      description: json['description'],
      category: json['category'],
      imageUrl: json['imageUrl'],
      outcomeA: json['outcomeA'],
      outcomeB: json['outcomeB'],
      oddsA: double.parse(json['oddsA'].toString()),
      oddsB: double.parse(json['oddsB'].toString()),
      volume: double.parse(json['volume'].toString()),
      liquidity: double.parse(json['liquidity'].toString()),
      totalBets: json['totalBets'],
      status: MarketStatus.values.firstWhere(
        (e) => e.name == json['status'],
        orElse: () => MarketStatus.active,
      ),
      endTime: DateTime.parse(json['endTime']),
      resolutionTime: json['resolutionTime'] != null 
          ? DateTime.parse(json['resolutionTime']) 
          : null,
      resolvedOutcome: json['resolvedOutcome'],
    );
  }
  
  bool get isResolved => status == MarketStatus.resolved;
  bool get isExpired => DateTime.now().isAfter(endTime);
}

enum MarketStatus {
  active,
  closed,
  resolving,
  resolved,
  cancelled,
}

class PredictionBet {
  final String betId;
  final String marketId;
  final String userAddress;
  final String outcome;
  final double amount;
  final double potentialWin;
  final double actualWin;
  final double odds;
  final BetStatus status;
  final DateTime createdAt;
  final DateTime? claimedAt;
  
  PredictionBet({
    required this.betId,
    required this.marketId,
    required this.userAddress,
    required this.outcome,
    required this.amount,
    required this.potentialWin,
    required this.actualWin,
    required this.odds,
    required this.status,
    required this.createdAt,
    this.claimedAt,
  });
  
  factory PredictionBet.fromJson(Map<String, dynamic> json) {
    return PredictionBet(
      betId: json['betId'],
      marketId: json['marketId'],
      userAddress: json['userAddress'],
      outcome: json['outcome'],
      amount: double.parse(json['amount'].toString()),
      potentialWin: double.parse(json['potentialWin'].toString()),
      actualWin: double.parse(json['actualWin'].toString()),
      odds: double.parse(json['odds'].toString()),
      status: BetStatus.values.firstWhere(
        (e) => e.name == json['status'],
        orElse: () => BetStatus.pending,
      ),
      createdAt: DateTime.parse(json['createdAt']),
      claimedAt: json['claimedAt'] != null 
          ? DateTime.parse(json['claimedAt']) 
          : null,
    );
  }
  
  bool get isWinner => status == BetStatus.won;
  bool get canClaim => status == BetStatus.won && claimedAt == null;
}

enum BetStatus {
  pending,
  won,
  lost,
  claimed,
}

class MarketOdds {
  final String marketId;
  final double oddsA;
  final double oddsB;
  final double impliedProbabilityA;
  final double impliedProbabilityB;
  
  MarketOdds({
    required this.marketId,
    required this.oddsA,
    required this.oddsB,
    required this.impliedProbabilityA,
    required this.impliedProbabilityB,
  });
  
  factory MarketOdds.fromJson(Map<String, dynamic> json) {
    return MarketOdds(
      marketId: json['marketId'],
      oddsA: double.parse(json['oddsA'].toString()),
      oddsB: double.parse(json['oddsB'].toString()),
      impliedProbabilityA: double.parse(json['impliedProbabilityA'].toString()),
      impliedProbabilityB: double.parse(json['impliedProbabilityB'].toString()),
    );
  }
}

class MarketResolution {
  final String marketId;
  final String outcome;
  final double totalVolume;
  final int totalBets;
  final DateTime resolvedAt;
  
  MarketResolution({
    required this.marketId,
    required this.outcome,
    required this.totalVolume,
    required this.totalBets,
    required this.resolvedAt,
  });
  
  factory MarketResolution.fromJson(Map<String, dynamic> json) {
    return MarketResolution(
      marketId: json['marketId'],
      outcome: json['outcome'],
      totalVolume: double.parse(json['totalVolume'].toString()),
      totalBets: json['totalBets'],
      resolvedAt: DateTime.parse(json['resolvedAt']),
    );
  }
}
