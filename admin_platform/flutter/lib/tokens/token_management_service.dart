// Admin Platform Token Management Service - Flutter

import 'dart:convert';
import 'package:http/http.dart' as http;

class TokenManagementService {
  static const String API_BASE = 'https://admin-api.tigerwallet.com/api/v1';
  String? _token;
  
  TokenManagementService({String? token}) : _token = token;
  
  Map<String, String> get _headers => {
    'Content-Type': 'application/json',
    if (_token != null) 'Authorization': 'Bearer $_token',
  };
  
  // Get all tokens
  Future<List<Token>> getTokens({String? status}) async {
    String url = '$API_BASE/tokens';
    if (status != null) url += '?status=$status';
    
    final response = await http.get(Uri.parse(url), headers: _headers);
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((t) => Token.fromJson(t)).toList();
    }
    return [];
  }
  
  // Get token details
  Future<Token> getTokenDetails(String tokenId) async {
    final response = await http.get(
      Uri.parse('$API_BASE/tokens/$tokenId'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return Token.fromJson(data['data']);
    }
    throw Exception('Failed to get token');
  }
  
  // Add new token
  Future<Token> addToken({
    required String name,
    required String symbol,
    required String contractAddress,
    required int decimals,
    required String chain,
    required double maxSupply,
    bool isListed = false,
  }) async {
    final response = await http.post(
      Uri.parse('$API_BASE/tokens'),
      headers: _headers,
      body: json.encode({
        'name': name,
        'symbol': symbol,
        'contractAddress': contractAddress,
        'decimals': decimals,
        'chain': chain,
        'maxSupply': maxSupply,
        'isListed': isListed,
      }),
    );
    
    if (response.statusCode == 201) {
      final data = json.decode(response.body);
      return Token.fromJson(data['data']);
    }
    throw Exception('Failed to add token');
  }
  
  // Update token
  Future<Token> updateToken(String tokenId, Map<String, dynamic> updates) async {
    final response = await http.put(
      Uri.parse('$API_BASE/tokens/$tokenId'),
      headers: _headers,
      body: json.encode(updates),
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return Token.fromJson(data['data']);
    }
    throw Exception('Failed to update token');
  }
  
  // List token (make visible to users)
  Future<bool> listToken(String tokenId) async {
    final response = await http.post(
      Uri.parse('$API_BASE/tokens/$tokenId/list'),
      headers: _headers,
    );
    
    return response.statusCode == 200;
  }
  
  // Delist token
  Future<bool> delistToken(String tokenId) async {
    final response = await http.post(
      Uri.parse('$API_BASE/tokens/$tokenId/delist'),
      headers: _headers,
    );
    
    return response.statusCode == 200;
  }
  
  // Get trading pairs
  Future<List<TradingPair>> getTradingPairs({String? baseToken}) async {
    String url = '$API_BASE/pairs';
    if (baseToken != null) url += '?baseToken=$baseToken';
    
    final response = await http.get(Uri.parse(url), headers: _headers);
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((p) => TradingPair.fromJson(p)).toList();
    }
    return [];
  }
  
  // Create trading pair
  Future<TradingPair> createTradingPair({
    required String baseToken,
    required String quoteToken,
    required String pairAddress,
    required double minTradeAmount,
    required double maxTradeAmount,
  }) async {
    final response = await http.post(
      Uri.parse('$API_BASE/pairs'),
      headers: _headers,
      body: json.encode({
        'baseToken': baseToken,
        'quoteToken': quoteToken,
        'pairAddress': pairAddress,
        'minTradeAmount': minTradeAmount,
        'maxTradeAmount': maxTradeAmount,
      }),
    );
    
    if (response.statusCode == 201) {
      final data = json.decode(response.body);
      return TradingPair.fromJson(data['data']);
    }
    throw Exception('Failed to create pair');
  }
  
  // Update trading pair
  Future<bool> updateTradingPair(String pairId, Map<String, dynamic> updates) async {
    final response = await http.put(
      Uri.parse('$API_BASE/pairs/$pairId'),
      headers: _headers,
      body: json.encode(updates),
    );
    
    return response.statusCode == 200;
  }
}

class Token {
  final String id;
  final String name;
  final String symbol;
  final String contractAddress;
  final int decimals;
  final String chain;
  final double maxSupply;
  final double circulatingSupply;
  final bool isListed;
  final String status;
  final DateTime createdAt;
  
  Token({
    required this.id,
    required this.name,
    required this.symbol,
    required this.contractAddress,
    required this.decimals,
    required this.chain,
    required this.maxSupply,
    required this.circulatingSupply,
    required this.isListed,
    required this.status,
    required this.createdAt,
  });
  
  factory Token.fromJson(Map<String, dynamic> json) {
    return Token(
      id: json['id'] ?? '',
      name: json['name'] ?? '',
      symbol: json['symbol'] ?? '',
      contractAddress: json['contractAddress'] ?? '',
      decimals: json['decimals'] ?? 18,
      chain: json['chain'] ?? '',
      maxSupply: (json['maxSupply'] ?? 0).toDouble(),
      circulatingSupply: (json['circulatingSupply'] ?? 0).toDouble(),
      isListed: json['isListed'] ?? false,
      status: json['status'] ?? 'PENDING',
      createdAt: DateTime.parse(json['createdAt']),
    );
  }
}

class TradingPair {
  final String id;
  final String baseToken;
  final String quoteToken;
  final String pairAddress;
  final double price;
  final double volume24h;
  final double liquidity;
  final bool isActive;
  
  TradingPair({
    required this.id,
    required this.baseToken,
    required this.quoteToken,
    required this.pairAddress,
    required this.price,
    required this.volume24h,
    required this.liquidity,
    required this.isActive,
  });
  
  factory TradingPair.fromJson(Map<String, dynamic> json) {
    return TradingPair(
      id: json['id'] ?? '',
      baseToken: json['baseToken'] ?? '',
      quoteToken: json['quoteToken'] ?? '',
      pairAddress: json['pairAddress'] ?? '',
      price: (json['price'] ?? 0).toDouble(),
      volume24h: (json['volume24h'] ?? 0).toDouble(),
      liquidity: (json['liquidity'] ?? 0).toDouble(),
      isActive: json['isActive'] ?? true,
    );
  }
}
