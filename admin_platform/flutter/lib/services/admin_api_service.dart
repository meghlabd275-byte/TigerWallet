import 'dart:convert';
import 'package:http/http.dart' as http;
import '../models/admin_models.dart';

class ApiException implements Exception {
  final String message;
  final int? statusCode;

  ApiException(this.message, {this.statusCode});

  @override
  String toString() => 'ApiException: $message (Status: $statusCode)';
}

class AdminApiService {
  static const String baseUrl = 'https://api.tigerwallet.io/admin/v1';
  
  String? _authToken;

  void setAuthToken(String token) {
    _authToken = token;
  }

  Map<String, String> get _headers => {
    'Content-Type': 'application/json',
    'Accept': 'application/json',
    if (_authToken != null) 'Authorization': 'Bearer $_authToken',
  };

  Future<Map<String, dynamic>> _get(String endpoint) async {
    try {
      final response = await http.get(
        Uri.parse('$baseUrl$endpoint'),
        headers: _headers,
      );
      return _handleResponse(response);
    } catch (e) {
      throw ApiException('Network error: $e');
    }
  }

  Future<Map<String, dynamic>> _post(String endpoint, {Map<String, dynamic>? body}) async {
    try {
      final response = await http.post(
        Uri.parse('$baseUrl$endpoint'),
        headers: _headers,
        body: body != null ? jsonEncode(body) : null,
      );
      return _handleResponse(response);
    } catch (e) {
      throw ApiException('Network error: $e');
    }
  }

  Future<Map<String, dynamic>> _put(String endpoint, {Map<String, dynamic>? body}) async {
    try {
      final response = await http.put(
        Uri.parse('$baseUrl$endpoint'),
        headers: _headers,
        body: body != null ? jsonEncode(body) : null,
      );
      return _handleResponse(response);
    } catch (e) {
      throw ApiException('Network error: $e');
    }
  }

  Future<Map<String, dynamic>> _delete(String endpoint) async {
    try {
      final response = await http.delete(
        Uri.parse('$baseUrl$endpoint'),
        headers: _headers,
      );
      return _handleResponse(response);
    } catch (e) {
      throw ApiException('Network error: $e');
    }
  }

  Map<String, dynamic> _handleResponse(http.Response response) {
    if (response.statusCode >= 200 && response.statusCode < 300) {
      if (response.body.isEmpty) return {};
      return jsonDecode(response.body) as Map<String, dynamic>;
    } else if (response.statusCode == 401) {
      throw ApiException('Unauthorized', statusCode: 401);
    } else if (response.statusCode == 403) {
      throw ApiException('Forbidden', statusCode: 403);
    } else if (response.statusCode == 404) {
      throw ApiException('Not found', statusCode: 404);
    } else {
      throw ApiException('Server error', statusCode: response.statusCode);
    }
  }

  // Authentication
  Future<AdminUser> login(String email, String password) async {
    final response = await _post('/auth/login', body: {
      'email': email,
      'password': password,
    });
    final token = response['token'] as String;
    setAuthToken(token);
    return AdminUser.fromJson(response['admin'] as Map<String, dynamic>);
  }

  Future<void> logout() async {
    await _post('/auth/logout');
    _authToken = null;
  }

  // Users
  Future<ApiResponse<PlatformUser>> getUsers({
    int page = 1,
    int limit = 20,
    String? status,
    String? kycStatus,
    String? search,
  }) async {
    final queryParams = <String, String>{
      'page': page.toString(),
      'limit': limit.toString(),
      if (status != null) 'status': status,
      if (kycStatus != null) 'kyc_status': kycStatus,
      if (search != null) 'search': search,
    };
    
    final response = await _get('/users?${_buildQuery(queryParams)}');
    return ApiResponse(
      data: (response['data'] as List)
          .map((e) => PlatformUser.fromJson(e as Map<String, dynamic>))
          .toList(),
      pagination: Pagination.fromJson(response['pagination'] as Map<String, dynamic>),
    );
  }

  Future<PlatformUser> getUser(int id) async {
    final response = await _get('/users/$id');
    return PlatformUser.fromJson(response);
  }

  Future<PlatformUser> updateUser(int id, Map<String, dynamic> data) async {
    final response = await _put('/users/$id', body: data);
    return PlatformUser.fromJson(response);
  }

  Future<void> suspendUser(int id, String reason) async {
    await _post('/users/$id/suspend', body: {'reason': reason});
  }

  Future<void> banUser(int id, String reason) async {
    await _post('/users/$id/ban', body: {'reason': reason, 'permanent': true});
  }

  Future<void> activateUser(int id) async {
    await _post('/users/$id/activate');
  }

  // Transactions
  Future<ApiResponse<Transaction>> getTransactions({
    int page = 1,
    int limit = 20,
    String? status,
    String? type,
    String? chain,
    bool? flagged,
  }) async {
    final queryParams = <String, String>{
      'page': page.toString(),
      'limit': limit.toString(),
      if (status != null) 'status': status,
      if (type != null) 'type': type,
      if (chain != null) 'chain': chain,
      if (flagged != null) 'flagged': flagged.toString(),
    };
    
    final response = await _get('/transactions?${_buildQuery(queryParams)}');
    return ApiResponse(
      data: (response['data'] as List)
          .map((e) => Transaction.fromJson(e as Map<String, dynamic>))
          .toList(),
      pagination: Pagination.fromJson(response['pagination'] as Map<String, dynamic>),
    );
  }

  Future<Transaction> getTransaction(int id) async {
    final response = await _get('/transactions/$id');
    return Transaction.fromJson(response);
  }

  Future<void> flagTransaction(int id, String reason) async {
    await _post('/transactions/$id/flag', body: {'reason': reason});
  }

  Future<void> unflagTransaction(int id) async {
    await _post('/transactions/$id/unflag');
  }

  // KYC
  Future<ApiResponse<KYCApplication>> getKYCApplications({
    int page = 1,
    int limit = 20,
    String? status,
    int? level,
  }) async {
    final queryParams = <String, String>{
      'page': page.toString(),
      'limit': limit.toString(),
      if (status != null) 'status': status,
      if (level != null) 'level': level.toString(),
    };
    
    final response = await _get('/kyc?${_buildQuery(queryParams)}');
    return ApiResponse(
      data: (response['data'] as List)
          .map((e) => KYCApplication.fromJson(e as Map<String, dynamic>))
          .toList(),
      pagination: Pagination.fromJson(response['pagination'] as Map<String, dynamic>),
    );
  }

  Future<void> approveKYC(int id) async {
    await _post('/kyc/$id/approve');
  }

  Future<void> rejectKYC(int id, String reason) async {
    await _post('/kyc/$id/reject', body: {'reason': reason});
  }

  // Tokens
  Future<ApiResponse<Token>> getTokens({
    int page = 1,
    int limit = 20,
    String? chain,
    bool? isActive,
    String? search,
  }) async {
    final queryParams = <String, String>{
      'page': page.toString(),
      'limit': limit.toString(),
      if (chain != null) 'chain': chain,
      if (isActive != null) 'is_active': isActive.toString(),
      if (search != null) 'search': search,
    };
    
    final response = await _get('/tokens?${_buildQuery(queryParams)}');
    return ApiResponse(
      data: (response['data'] as List)
          .map((e) => Token.fromJson(e as Map<String, dynamic>))
          .toList(),
      pagination: Pagination.fromJson(response['pagination'] as Map<String, dynamic>),
    );
  }

  Future<void> activateToken(int id) async {
    await _post('/tokens/$id/activate');
  }

  Future<void> deactivateToken(int id) async {
    await _post('/tokens/$id/deactivate');
  }

  Future<void> verifyToken(int id) async {
    await _post('/tokens/$id/verify');
  }

  // Withdrawals
  Future<ApiResponse<WithdrawalRequest>> getWithdrawals({
    int page = 1,
    int limit = 20,
    String? status,
    String? token,
    String? chain,
  }) async {
    final queryParams = <String, String>{
      'page': page.toString(),
      'limit': limit.toString(),
      if (status != null) 'status': status,
      if (token != null) 'token': token,
      if (chain != null) 'chain': chain,
    };
    
    final response = await _get('/withdrawals?${_buildQuery(queryParams)}');
    return ApiResponse(
      data: (response['data'] as List)
          .map((e) => WithdrawalRequest.fromJson(e as Map<String, dynamic>))
          .toList(),
      pagination: Pagination.fromJson(response['pagination'] as Map<String, dynamic>),
    );
  }

  Future<void> approveWithdrawal(int id) async {
    await _post('/withdrawals/$id/approve');
  }

  Future<void> rejectWithdrawal(int id, String reason) async {
    await _post('/withdrawals/$id/reject', body: {'reason': reason});
  }

  Future<void> processWithdrawal(int id, String txHash) async {
    await _post('/withdrawals/$id/process', body: {'tx_hash': txHash});
  }

  // White Labels
  Future<ApiResponse<WhiteLabel>> getWhiteLabels({
    int page = 1,
    int limit = 20,
    String? status,
    String? search,
  }) async {
    final queryParams = <String, String>{
      'page': page.toString(),
      'limit': limit.toString(),
      if (status != null) 'status': status,
      if (search != null) 'search': search,
    };
    
    final response = await _get('/whitelabels?${_buildQuery(queryParams)}');
    return ApiResponse(
      data: (response['data'] as List)
          .map((e) => WhiteLabel.fromJson(e as Map<String, dynamic>))
          .toList(),
      pagination: Pagination.fromJson(response['pagination'] as Map<String, dynamic>),
    );
  }

  Future<void> activateWhiteLabel(int id) async {
    await _post('/whitelabels/$id/activate');
  }

  Future<void> suspendWhiteLabel(int id) async {
    await _post('/whitelabels/$id/suspend');
  }

  // Bots
  Future<ApiResponse<BotInstance>> getBots({
    int page = 1,
    int limit = 20,
    String? status,
    String? botType,
  }) async {
    final queryParams = <String, String>{
      'page': page.toString(),
      'limit': limit.toString(),
      if (status != null) 'status': status,
      if (botType != null) 'bot_type': botType,
    };
    
    final response = await _get('/bots?${_buildQuery(queryParams)}');
    return ApiResponse(
      data: (response['data'] as List)
          .map((e) => BotInstance.fromJson(e as Map<String, dynamic>))
          .toList(),
      pagination: Pagination.fromJson(response['pagination'] as Map<String, dynamic>),
    );
  }

  Future<void> startBot(int id) async {
    await _post('/bots/$id/start');
  }

  Future<void> stopBot(int id) async {
    await _post('/bots/$id/stop');
  }

  Future<void> pauseBot(int id) async {
    await _post('/bots/$id/pause');
  }

  // Fees
  Future<List<FeeConfig>> getFeeConfigs() async {
    final response = await _get('/fees');
    return (response['data'] as List)
        .map((e) => FeeConfig.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  Future<FeeConfig> updateFeeConfig(int id, Map<String, dynamic> data) async {
    final response = await _put('/fees/$id', body: data);
    return FeeConfig.fromJson(response);
  }

  // Blockchains
  Future<List<Blockchain>> getBlockchains() async {
    final response = await _get('/blockchains');
    return (response['data'] as List)
        .map((e) => Blockchain.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  Future<void> activateBlockchain(String id) async {
    await _post('/blockchains/$id/activate');
  }

  Future<void> deactivateBlockchain(String id) async {
    await _post('/blockchains/$id/deactivate');
  }

  // System
  Future<Map<String, List<SystemStatus>>> getSystemStatus() async {
    final response = await _get('/system/status');
    return {
      'services': (response['services'] as List)
          .map((e) => SystemStatus.fromJson(e as Map<String, dynamic>))
          .toList(),
      'databases': (response['databases'] as List)
          .map((e) => SystemStatus.fromJson(e as Map<String, dynamic>))
          .toList(),
      'networks': (response['networks'] as List)
          .map((e) => SystemStatus.fromJson(e as Map<String, dynamic>))
          .toList(),
    };
  }

  Future<AnalyticsData> getAnalyticsOverview() async {
    final response = await _get('/analytics/overview');
    return AnalyticsData.fromJson(response);
  }

  // Admins
  Future<ApiResponse<AdminUser>> getAdmins({
    int page = 1,
    int limit = 20,
    String? role,
    String? status,
  }) async {
    final queryParams = <String, String>{
      'page': page.toString(),
      'limit': limit.toString(),
      if (role != null) 'role': role,
      if (status != null) 'status': status,
    };
    
    final response = await _get('/admins?${_buildQuery(queryParams)}');
    return ApiResponse(
      data: (response['data'] as List)
          .map((e) => AdminUser.fromJson(e as Map<String, dynamic>))
          .toList(),
      pagination: Pagination.fromJson(response['pagination'] as Map<String, dynamic>),
    );
  }

  Future<AdminUser> createAdmin(Map<String, dynamic> data) async {
    final response = await _post('/admins', body: data);
    return AdminUser.fromJson(response);
  }

  Future<void> deleteAdmin(int id) async {
    await _delete('/admins/$id');
  }

  Future<void> suspendAdmin(int id) async {
    await _post('/admins/$id/suspend');
  }

  Future<void> activateAdmin(int id) async {
    await _post('/admins/$id/activate');
  }

  String _buildQuery(Map<String, String> params) {
    return params.entries
        .map((e) => '${e.key}=${Uri.encodeComponent(e.value)}')
        .join('&');
  }
}
