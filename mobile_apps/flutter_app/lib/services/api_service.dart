/**
 * TigerWallet API Service
 * Complete backend API connectivity
 */

import 'dart:convert';
import 'package:http/http.dart' as http;

class ApiException implements Exception {
  final String message;
  final int? statusCode;
  
  ApiException(this.message, {this.statusCode});
  
  @override
  String toString() => 'ApiException: $message (status: $statusCode)';
}

class ApiService {
  final String baseUrl;
  final http.Client _client;
  String? _authToken;
  
  ApiService({
    required this.baseUrl,
    http.Client? client,
  }) : _client = client ?? http.Client();
  
  void setAuthToken(String token) {
    _authToken = token;
  }
  
  void clearAuthToken() {
    _authToken = null;
  }
  
  Map<String, String> get _headers {
    final headers = {
      'Content-Type': 'application/json',
      'Accept': 'application/json',
    };
    if (_authToken != null) {
      headers['Authorization'] = 'Bearer $_authToken';
    }
    return headers;
  }
  
  // Wallet API
  Future<Map<String, dynamic>> getWalletInfo(String address) async {
    return _get('/wallet/$address');
  }
  
  Future<List<dynamic>> getWalletTokens(String address) async {
    return _get('/wallet/$address/tokens');
  }
  
  Future<List<dynamic>> getWalletTransactions(String address, {int page = 1, int limit = 20}) async {
    return _get('/wallet/$address/transactions?page=$page&limit=$limit');
  }
  
  // Swap API
  Future<Map<String, dynamic>> getSwapQuote({
    required String fromToken,
    required String toToken,
    required String amount,
    required String fromAddress,
  }) async {
    return _post('/swap/quote', {
      'fromToken': fromToken,
      'toToken': toToken,
      'amount': amount,
      'fromAddress': fromAddress,
    });
  }
  
  Future<Map<String, dynamic>> executeSwap({
    required String quoteId,
    required String privateKey,
  }) async {
    return _post('/swap/execute', {
      'quoteId': quoteId,
      'privateKey': privateKey,
    });
  }
  
  // Staking API
  Future<List<dynamic>> getStakingPools(String chainId) async {
    return _get('/staking/pools?chainId=$chainId');
  }
  
  Future<Map<String, dynamic>> stake({
    required String poolId,
    required String amount,
    required String privateKey,
  }) async {
    return _post('/staking/stake', {
      'poolId': poolId,
      'amount': amount,
      'privateKey': privateKey,
    });
  }
  
  Future<Map<String, dynamic>> unstake({
    required String stakeId,
    required String amount,
    required String privateKey,
  }) async {
    return _post('/staking/unstake', {
      'stakeId': stakeId,
      'amount': amount,
      'privateKey': privateKey,
    });
  }
  
  // Bridge API
  Future<Map<String, dynamic>> getBridgeQuote({
    required String fromChain,
    required String toChain,
    required String token,
    required String amount,
  }) async {
    return _post('/bridge/quote', {
      'fromChain': fromChain,
      'toChain': toChain,
      'token': token,
      'amount': amount,
    });
  }
  
  Future<Map<String, dynamic>> executeBridge({
    required String quoteId,
    required String privateKey,
  }) async {
    return _post('/bridge/execute', {
      'quoteId': quoteId,
      'privateKey': privateKey,
    });
  }
  
  // NFT API
  Future<List<dynamic>> getUserNFTs(String address) async {
    return _get('/nft/$address');
  }
  
  Future<Map<String, dynamic>> transferNFT({
    required String tokenId,
    required String toAddress,
    required String privateKey,
  }) async {
    return _post('/nft/transfer', {
      'tokenId': tokenId,
      'toAddress': toAddress,
      'privateKey': privateKey,
    });
  }
  
  // Price API
  Future<Map<String, dynamic>> getPrices(List<String> tokenAddresses) async {
    return _post('/prices', {
      'tokens': tokenAddresses,
    });
  }
  
  Future<Map<String, dynamic>> getPriceChart(String tokenAddress, {String interval = '24h'}) async {
    return _get('/prices/$tokenAddress/chart?interval=$interval');
  }
  
  // Market API
  Future<List<dynamic>> getMarkets() async {
    return _get('/markets');
  }
  
  Future<Map<String, dynamic>> getMarketDetails(String marketId) async {
    return _get('/markets/$marketId');
  }
  
  // Gas API
  Future<Map<String, dynamic>> getGasPrices(String chainId) async {
    return _get('/gas/$chainId');
  }
  
  Future<Map<String, dynamic>> estimateGas({
    required String chainId,
    required String from,
    required String to,
    required String value,
    required String data,
  }) async {
    return _post('/gas/estimate', {
      'chainId': chainId,
      'from': from,
      'to': to,
      'value': value,
      'data': data,
    });
  }
  
  // User API
  Future<Map<String, dynamic>> registerUser(Map<String, dynamic> userData) async {
    return _post('/users/register', userData);
  }
  
  Future<Map<String, dynamic>> getUserProfile() async {
    return _get('/users/me');
  }
  
  Future<Map<String, dynamic>> updateUserProfile(Map<String, dynamic> data) async {
    return _put('/users/me', data);
  }
  
  // HTTP Methods
  Future<dynamic> _get(String endpoint) async {
    final response = await _client.get(
      Uri.parse('$baseUrl$endpoint'),
      headers: _headers,
    );
    
    return _handleResponse(response);
  }
  
  Future<dynamic> _post(String endpoint, Map<String, dynamic> body) async {
    final response = await _client.post(
      Uri.parse('$baseUrl$endpoint'),
      headers: _headers,
      body: jsonEncode(body),
    );
    
    return _handleResponse(response);
  }
  
  Future<dynamic> _put(String endpoint, Map<String, dynamic> body) async {
    final response = await _client.put(
      Uri.parse('$baseUrl$endpoint'),
      headers: _headers,
      body: jsonEncode(body),
    );
    
    return _handleResponse(response);
  }
  
  Future<dynamic> _delete(String endpoint) async {
    final response = await _client.delete(
      Uri.parse('$baseUrl$endpoint'),
      headers: _headers,
    );
    
    return _handleResponse(response);
  }
  
  dynamic _handleResponse(http.Response response) {
    if (response.statusCode >= 200 && response.statusCode < 300) {
      if (response.body.isEmpty) return {};
      return jsonDecode(response.body);
    }
    
    String message = 'Request failed';
    try {
      final body = jsonDecode(response.body);
      message = body['message'] ?? body['error'] ?? message;
    } catch (_) {}
    
    throw ApiException(message, statusCode: response.statusCode);
  }
  
  void dispose() {
    _client.close();
  }
}
