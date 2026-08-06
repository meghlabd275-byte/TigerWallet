/**
 * TigerWallet Admin - Dio HTTP Client Implementation
 */

import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../constants/app_constants.dart';
import 'api_client.dart';
import 'logger.dart';

class DioClient implements ApiClient {
  late final Dio _dio;
  final SharedPreferences _prefs;
  
  DioClient() : _prefs = throw UnimplementedError() {
    _initDio();
  }
  
  factory DioClient.withPrefs(SharedPreferences prefs) {
    return DioClient._internal(prefs);
  }
  
  DioClient._internal(this._prefs) {
    _initDio();
  }
  
  void _initDio() {
    _dio = Dio(BaseOptions(
      baseUrl: AppConstants.baseUrl,
      connectTimeout: const Duration(milliseconds: AppConstants.apiTimeout),
      receiveTimeout: const Duration(milliseconds: AppConstants.apiTimeout),
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      },
    ));
    
    // Add interceptors
    _dio.interceptors.add(InterceptorsWrapper(
      onRequest: (options, handler) async {
        // Add auth token
        final token = _prefs.getString(AppConstants.tokenKey);
        if (token != null) {
          options.headers['Authorization'] = 'Bearer $token';
        }
        Logger.d('API Request: ${options.method} ${options.path}');
        handler.next(options);
      },
      onResponse: (response, handler) {
        Logger.d('API Response: ${response.statusCode} ${response.requestOptions.path}');
        handler.next(response);
      },
      onError: (error, handler) {
        Logger.e('API Error: ${error.message}');
        
        // Handle 401 - Unauthorized
        if (error.response?.statusCode == 401) {
          _handleUnauthorized();
        }
        
        handler.next(error);
      },
    ));
    
    // Add logging interceptor in debug mode
    if (kDebugMode) {
      _dio.interceptors.add(LogInterceptor(
        requestBody: true,
        responseBody: true,
        error: true,
      ));
    }
  }
  
  void _handleUnauthorized() async {
    // Clear tokens and redirect to login
    await _prefs.remove(AppConstants.tokenKey);
    await _prefs.remove(AppConstants.refreshTokenKey);
  }
  
  String? get _token => _prefs.getString(AppConstants.tokenKey);
  String? get _refreshToken => _prefs.getString(AppConstants.refreshTokenKey);
  
  // Auth
  @override
  Future<Map<String, dynamic>> login(String email, String password) async {
    final response = await _dio.post(
      ApiEndpoints.login,
      data: {'email': email, 'password': password},
    );
    final data = response.data;
    
    // Save tokens
    await _prefs.setString(AppConstants.tokenKey, data['token']);
    await _prefs.setString(AppConstants.refreshTokenKey, data['refresh_token']);
    
    return data;
  }
  
  @override
  Future<void> logout() async {
    try {
      await _dio.post(ApiEndpoints.logout);
    } finally {
      await _prefs.remove(AppConstants.tokenKey);
      await _prefs.remove(AppConstants.refreshTokenKey);
    }
  }
  
  @override
  Future<Map<String, dynamic>> refreshToken(String refreshToken) async {
    final response = await _dio.post(
      ApiEndpoints.refresh,
      data: {'refresh_token': refreshToken},
    );
    final data = response.data;
    
    await _prefs.setString(AppConstants.tokenKey, data['token']);
    if (data['refresh_token'] != null) {
      await _prefs.setString(AppConstants.refreshTokenKey, data['refresh_token']);
    }
    
    return data;
  }
  
  @override
  Future<void> setup2FA() async {
    await _dio.post(ApiEndpoints.setup2FA);
  }
  
  @override
  Future<bool> verify2FA(String code) async {
    final response = await _dio.post(
      ApiEndpoints.verify2FA,
      data: {'code': code},
    );
    return response.data['verified'] ?? false;
  }
  
  @override
  Future<void> changePassword(String oldPassword, String newPassword) async {
    await _dio.post(
      '/auth/change-password',
      data: {'old_password': oldPassword, 'new_password': newPassword},
    );
  }
  
  // Admins
  @override
  Future<List<Map<String, dynamic>>> getAdmins() async {
    final response = await _dio.get(ApiEndpoints.admins);
    return List<Map<String, dynamic>>.from(response.data);
  }
  
  @override
  Future<Map<String, dynamic>> getAdmin(String id) async {
    final response = await _dio.get('${ApiEndpoints.admins}/$id');
    return response.data;
  }
  
  @override
  Future<Map<String, dynamic>> createAdmin(Map<String, dynamic> data) async {
    final response = await _dio.post(ApiEndpoints.admins, data: data);
    return response.data;
  }
  
  @override
  Future<Map<String, dynamic>> updateAdmin(String id, Map<String, dynamic> data) async {
    final response = await _dio.put('${ApiEndpoints.admins}/$id', data: data);
    return response.data;
  }
  
  @override
  Future<void> deleteAdmin(String id) async {
    await _dio.delete('${ApiEndpoints.admins}/$id');
  }
  
  @override
  Future<void> suspendAdmin(String id) async {
    await _dio.post('${ApiEndpoints.admins}/$id/suspend');
  }
  
  @override
  Future<void> activateAdmin(String id) async {
    await _dio.post('${ApiEndpoints.admins}/$id/activate');
  }
  
  // Users
  @override
  Future<Map<String, dynamic>> getUsers({int page = 1, int pageSize = 20, String? search, String? status}) async {
    final response = await _dio.get(
      ApiEndpoints.users,
      queryParameters: {
        'page': page,
        'page_size': pageSize,
        if (search != null) 'search': search,
        if (status != null) 'status': status,
      },
    );
    return response.data;
  }
  
  @override
  Future<Map<String, dynamic>> getUser(String id) async {
    final response = await _dio.get('${ApiEndpoints.users}/$id');
    return response.data;
  }
  
  @override
  Future<Map<String, dynamic>> updateUser(String id, Map<String, dynamic> data) async {
    final response = await _dio.put('${ApiEndpoints.users}/$id', data: data);
    return response.data;
  }
  
  @override
  Future<void> banUser(String id) async {
    await _dio.post('${ApiEndpoints.users}/$id/ban');
  }
  
  @override
  Future<void> unbanUser(String id) async {
    await _dio.post('${ApiEndpoints.users}/$id/unban');
  }
  
  @override
  Future<void> suspendUser(String id) async {
    await _dio.post('${ApiEndpoints.users}/$id/suspend');
  }
  
  // KYC
  @override
  Future<Map<String, dynamic>> getKycRequests({int page = 1, int pageSize = 20, String? status}) async {
    final response = await _dio.get(
      ApiEndpoints.kycRequests,
      queryParameters: {
        'page': page,
        'page_size': pageSize,
        if (status != null) 'status': status,
      },
    );
    return response.data;
  }
  
  @override
  Future<Map<String, dynamic>> getKyc(String id) async {
    final response = await _dio.get('${ApiEndpoints.kycRequests}/$id');
    return response.data;
  }
  
  @override
  Future<void> approveKyc(String id) async {
    await _dio.post('${ApiEndpoints.kycRequests}/$id/approve');
  }
  
  @override
  Future<void> rejectKyc(String id, String reason) async {
    await _dio.post(
      '${ApiEndpoints.kycRequests}/$id/reject',
      data: {'reason': reason},
    );
  }
  
  // Transactions
  @override
  Future<Map<String, dynamic>> getTransactions({int page = 1, int pageSize = 20, String? status, String? userId}) async {
    final response = await _dio.get(
      ApiEndpoints.transactions,
      queryParameters: {
        'page': page,
        'page_size': pageSize,
        if (status != null) 'status': status,
        if (userId != null) 'user_id': userId,
      },
    );
    return response.data;
  }
  
  @override
  Future<Map<String, dynamic>> getTransaction(String id) async {
    final response = await _dio.get('${ApiEndpoints.transactions}/$id');
    return response.data;
  }
  
  @override
  Future<void> flagTransaction(String id, String reason) async {
    await _dio.post(
      '${ApiEndpoints.transactions}/$id/flag',
      data: {'reason': reason},
    );
  }
  
  @override
  Future<void> unflagTransaction(String id) async {
    await _dio.post('${ApiEndpoints.transactions}/$id/unflag');
  }
  
  // Withdrawals
  @override
  Future<Map<String, dynamic>> getWithdrawals({int page = 1, int pageSize = 20, String? status}) async {
    final response = await _dio.get(
      ApiEndpoints.withdrawals,
      queryParameters: {
        'page': page,
        'page_size': pageSize,
        if (status != null) 'status': status,
      },
    );
    return response.data;
  }
  
  @override
  Future<Map<String, dynamic>> getWithdrawal(String id) async {
    final response = await _dio.get('${ApiEndpoints.withdrawals}/$id');
    return response.data;
  }
  
  @override
  Future<void> approveWithdrawal(String id) async {
    await _dio.post('${ApiEndpoints.withdrawals}/$id/approve');
  }
  
  @override
  Future<void> rejectWithdrawal(String id, String reason) async {
    await _dio.post(
      '${ApiEndpoints.withdrawals}/$id/reject',
      data: {'reason': reason},
    );
  }
  
  @override
  Future<void> processWithdrawal(String id) async {
    await _dio.post('${ApiEndpoints.withdrawals}/$id/process');
  }
  
  // Tokens
  @override
  Future<Map<String, dynamic>> getTokens({int page = 1, int pageSize = 20, String? search, String? status}) async {
    final response = await _dio.get(
      ApiEndpoints.tokens,
      queryParameters: {
        'page': page,
        'page_size': pageSize,
        if (search != null) 'search': search,
        if (status != null) 'status': status,
      },
    );
    return response.data;
  }
  
  @override
  Future<Map<String, dynamic>> getToken(String id) async {
    final response = await _dio.get('${ApiEndpoints.tokens}/$id');
    return response.data;
  }
  
  @override
  Future<Map<String, dynamic>> createToken(Map<String, dynamic> data) async {
    final response = await _dio.post(ApiEndpoints.tokens, data: data);
    return response.data;
  }
  
  @override
  Future<Map<String, dynamic>> updateToken(String id, Map<String, dynamic> data) async {
    final response = await _dio.put('${ApiEndpoints.tokens}/$id', data: data);
    return response.data;
  }
  
  @override
  Future<void> deleteToken(String id) async {
    await _dio.delete('${ApiEndpoints.tokens}/$id');
  }
  
  @override
  Future<void> verifyToken(String id) async {
    await _dio.post('${ApiEndpoints.tokens}/$id/verify');
  }
  
  // Pairs
  @override
  Future<Map<String, dynamic>> getPairs({int page = 1, int pageSize = 20, String? status}) async {
    final response = await _dio.get(
      ApiEndpoints.pairs,
      queryParameters: {
        'page': page,
        'page_size': pageSize,
        if (status != null) 'status': status,
      },
    );
    return response.data;
  }
  
  @override
  Future<Map<String, dynamic>> getPair(String id) async {
    final response = await _dio.get('${ApiEndpoints.pairs}/$id');
    return response.data;
  }
  
  @override
  Future<Map<String, dynamic>> createPair(Map<String, dynamic> data) async {
    final response = await _dio.post(ApiEndpoints.pairs, data: data);
    return response.data;
  }
  
  @override
  Future<Map<String, dynamic>> updatePair(String id, Map<String, dynamic> data) async {
    final response = await _dio.put('${ApiEndpoints.pairs}/$id', data: data);
    return response.data;
  }
  
  @override
  Future<void> haltPair(String id) async {
    await _dio.post('${ApiEndpoints.pairs}/$id/halt');
  }
  
  @override
  Future<void> activatePair(String id) async {
    await _dio.post('${ApiEndpoints.pairs}/$id/activate');
  }
  
  // Blockchains
  @override
  Future<List<Map<String, dynamic>>> getBlockchains() async {
    final response = await _dio.get(ApiEndpoints.blockchains);
    return List<Map<String, dynamic>>.from(response.data);
  }
  
  @override
  Future<Map<String, dynamic>> getBlockchain(String id) async {
    final response = await _dio.get('${ApiEndpoints.blockchains}/$id');
    return response.data;
  }
  
  @override
  Future<Map<String, dynamic>> createBlockchain(Map<String, dynamic> data) async {
    final response = await _dio.post(ApiEndpoints.blockchains, data: data);
    return response.data;
  }
  
  @override
  Future<Map<String, dynamic>> updateBlockchain(String id, Map<String, dynamic> data) async {
    final response = await _dio.put('${ApiEndpoints.blockchains}/$id', data: data);
    return response.data;
  }
  
  // Fees
  @override
  Future<List<Map<String, dynamic>>> getFees({int? chainId}) async {
    final response = await _dio.get(
      ApiEndpoints.fees,
      queryParameters: {if (chainId != null) 'chain_id': chainId},
    );
    return List<Map<String, dynamic>>.from(response.data);
  }
  
  @override
  Future<Map<String, dynamic>> createFee(Map<String, dynamic> data) async {
    final response = await _dio.post(ApiEndpoints.fees, data: data);
    return response.data;
  }
  
  @override
  Future<Map<String, dynamic>> updateFee(String id, Map<String, dynamic> data) async {
    final response = await _dio.put('${ApiEndpoints.fees}/$id', data: data);
    return response.data;
  }
  
  // White Labels
  @override
  Future<Map<String, dynamic>> getWhiteLabels({int page = 1, int pageSize = 20, String? status}) async {
    final response = await _dio.get(
      ApiEndpoints.whiteLabels,
      queryParameters: {
        'page': page,
        'page_size': pageSize,
        if (status != null) 'status': status,
      },
    );
    return response.data;
  }
  
  @override
  Future<Map<String, dynamic>> getWhiteLabel(String id) async {
    final response = await _dio.get('${ApiEndpoints.whiteLabels}/$id');
    return response.data;
  }
  
  @override
  Future<Map<String, dynamic>> createWhiteLabel(Map<String, dynamic> data) async {
    final response = await _dio.post(ApiEndpoints.whiteLabels, data: data);
    return response.data;
  }
  
  @override
  Future<Map<String, dynamic>> updateWhiteLabel(String id, Map<String, dynamic> data) async {
    final response = await _dio.put('${ApiEndpoints.whiteLabels}/$id', data: data);
    return response.data;
  }
  
  @override
  Future<void> activateWhiteLabel(String id) async {
    await _dio.post('${ApiEndpoints.whiteLabels}/$id/activate');
  }
  
  @override
  Future<void> suspendWhiteLabel(String id) async {
    await _dio.post('${ApiEndpoints.whiteLabels}/$id/suspend');
  }
  
  // Tickets
  @override
  Future<Map<String, dynamic>> getTickets({int page = 1, int pageSize = 20, String? status, String? priority}) async {
    final response = await _dio.get(
      ApiEndpoints.tickets,
      queryParameters: {
        'page': page,
        'page_size': pageSize,
        if (status != null) 'status': status,
        if (priority != null) 'priority': priority,
      },
    );
    return response.data;
  }
  
  @override
  Future<Map<String, dynamic>> getTicket(String id) async {
    final response = await _dio.get('${ApiEndpoints.tickets}/$id');
    return response.data;
  }
  
  @override
  Future<Map<String, dynamic>> createTicket(Map<String, dynamic> data) async {
    final response = await _dio.post(ApiEndpoints.tickets, data: data);
    return response.data;
  }
  
  @override
  Future<Map<String, dynamic>> updateTicket(String id, Map<String, dynamic> data) async {
    final response = await _dio.put('${ApiEndpoints.tickets}/$id', data: data);
    return response.data;
  }
  
  @override
  Future<void> assignTicket(String id, String adminId) async {
    await _dio.put(
      '${ApiEndpoints.tickets}/$id/assign',
      data: {'assigned_to': adminId},
    );
  }
  
  @override
  Future<void> addTicketMessage(String id, String message) async {
    await _dio.post(
      '${ApiEndpoints.tickets}/$id/messages',
      data: {'message': message},
    );
  }
  
  // Analytics
  @override
  Future<Map<String, dynamic>> getDashboardStats() async {
    final response = await _dio.get(ApiEndpoints.analyticsDashboard);
    return response.data;
  }
  
  @override
  Future<Map<String, dynamic>> getUserAnalytics({String? startDate, String? endDate}) async {
    final response = await _dio.get(
      ApiEndpoints.analyticsUsers,
      queryParameters: {
        if (startDate != null) 'start_date': startDate,
        if (endDate != null) 'end_date': endDate,
      },
    );
    return response.data;
  }
  
  @override
  Future<Map<String, dynamic>> getTransactionAnalytics({String? startDate, String? endDate}) async {
    final response = await _dio.get(
      ApiEndpoints.analyticsTransactions,
      queryParameters: {
        if (startDate != null) 'start_date': startDate,
        if (endDate != null) 'end_date': endDate,
      },
    );
    return response.data;
  }
  
  @override
  Future<Map<String, dynamic>> getRevenueAnalytics({String? startDate, String? endDate}) async {
    final response = await _dio.get(
      ApiEndpoints.analyticsRevenue,
      queryParameters: {
        if (startDate != null) 'start_date': startDate,
        if (endDate != null) 'end_date': endDate,
      },
    );
    return response.data;
  }
  
  @override
  Future<List<Map<String, dynamic>>> getVolumeChart({String? startDate, String? endDate, String? interval}) async {
    final response = await _dio.get(
      '/analytics/volume-chart',
      queryParameters: {
        if (startDate != null) 'start_date': startDate,
        if (endDate != null) 'end_date': endDate,
        if (interval != null) 'interval': interval,
      },
    );
    return List<Map<String, dynamic>>.from(response.data);
  }
  
  // Audit
  @override
  Future<Map<String, dynamic>> getAuditLogs({int page = 1, int pageSize = 50, String? adminId, String? action}) async {
    final response = await _dio.get(
      ApiEndpoints.auditLogs,
      queryParameters: {
        'page': page,
        'page_size': pageSize,
        if (adminId != null) 'admin_id': adminId,
        if (action != null) 'action': action,
      },
    );
    return response.data;
  }
  
  @override
  Future<String> exportAuditLogs(Map<String, dynamic> filters) async {
    final response = await _dio.post(
      ApiEndpoints.auditExport,
      data: filters,
    );
    return response.data.toString();
  }
  
  // Feature Flags
  @override
  Future<List<Map<String, dynamic>>> getFeatureFlags() async {
    final response = await _dio.get(ApiEndpoints.featureFlags);
    return List<Map<String, dynamic>>.from(response.data);
  }
  
  @override
  Future<Map<String, dynamic>> createFeatureFlag(Map<String, dynamic> data) async {
    final response = await _dio.post(ApiEndpoints.featureFlags, data: data);
    return response.data;
  }
  
  @override
  Future<Map<String, dynamic>> updateFeatureFlag(String id, Map<String, dynamic> data) async {
    final response = await _dio.put('${ApiEndpoints.featureFlags}/$id', data: data);
    return response.data;
  }
  
  @override
  Future<void> deleteFeatureFlag(String id) async {
    await _dio.delete('${ApiEndpoints.featureFlags}/$id');
  }
  
  // Notifications
  @override
  Future<List<Map<String, dynamic>>> getNotifications() async {
    final response = await _dio.get(ApiEndpoints.notifications);
    return List<Map<String, dynamic>>.from(response.data);
  }
  
  @override
  Future<void> markNotificationRead(String id) async {
    await _dio.put('${ApiEndpoints.notifications}/$id/read');
  }
  
  @override
  Future<void> broadcastNotification(String title, String message, String type) async {
    await _dio.post(
      ApiEndpoints.notificationBroadcast,
      data: {
        'title': title,
        'message': message,
        'type': type,
      },
    );
  }
  
  // Backups
  @override
  Future<List<Map<String, dynamic>>> getBackups() async {
    final response = await _dio.get(ApiEndpoints.backups);
    return List<Map<String, dynamic>>.from(response.data);
  }
  
  @override
  Future<Map<String, dynamic>> createBackup(String type) async {
    final response = await _dio.post(
      ApiEndpoints.backupCreate,
      data: {'type': type},
    );
    return response.data;
  }
  
  @override
  Future<void> restoreBackup(String id) async {
    await _dio.post('${ApiEndpoints.backups}/$id/restore');
  }
  
  @override
  Future<void> deleteBackup(String id) async {
    await _dio.delete('${ApiEndpoints.backups}/$id');
  }
  
  // Webhooks
  @override
  Future<List<Map<String, dynamic>>> getWebhooks() async {
    final response = await _dio.get(ApiEndpoints.webhooks);
    return List<Map<String, dynamic>>.from(response.data);
  }
  
  @override
  Future<Map<String, dynamic>> createWebhook(Map<String, dynamic> data) async {
    final response = await _dio.post(ApiEndpoints.webhooks, data: data);
    return response.data;
  }
  
  @override
  Future<Map<String, dynamic>> updateWebhook(String id, Map<String, dynamic> data) async {
    final response = await _dio.put('${ApiEndpoints.webhooks}/$id', data: data);
    return response.data;
  }
  
  @override
  Future<void> testWebhook(String id) async {
    await _dio.post('${ApiEndpoints.webhooks}/$id/test');
  }
  
  @override
  Future<void> deleteWebhook(String id) async {
    await _dio.delete('${ApiEndpoints.webhooks}/$id');
  }
}
