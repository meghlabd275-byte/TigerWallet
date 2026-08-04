import 'dart:convert';
import 'package:http/http.dart' as http;

class ApiService {
  static const String baseUrl = 'https://admin-api.tigerwallet.com';
  static String? _token;

  static void setToken(String token) {
    _token = token;
  }

  static Map<String, String> get _headers => {
    'Content-Type': 'application/json',
    if (_token != null) 'Authorization': 'Bearer $_token',
  };

  // Auth
  static Future<Map<String, dynamic>> login(String email, String password) async {
    final response = await http.post(
      Uri.parse('$baseUrl/api/v1/auth/login'),
      headers: _headers,
      body: jsonEncode({'email': email, 'password': password}),
    );
    final data = jsonDecode(response.body);
    if (data['token'] != null) {
      setToken(data['token']);
    }
    return data;
  }

  static Future<void> logout() async {
    await http.post(Uri.parse('$baseUrl/api/v1/auth/logout'), headers: _headers);
    _token = null;
  }

  static Future<Map<String, dynamic>> getCurrentAdmin() async {
    final response = await http.get(Uri.parse('$baseUrl/api/v1/auth/me'), headers: _headers);
    return jsonDecode(response.body);
  }

  // Users
  static Future<Map<String, dynamic>> getUsers({int page = 1, int limit = 20, String? status, String? search}) async {
    final queryParams = {'page': page.toString(), 'limit': limit.toString()};
    if (status != null) queryParams['status'] = status;
    if (search != null) queryParams['search'] = search;
    
    final response = await http.get(
      Uri.parse('$baseUrl/api/v1/users').replace(queryParameters: queryParams),
      headers: _headers,
    );
    return jsonDecode(response.body);
  }

  static Future<Map<String, dynamic>> getUser(String id) async {
    final response = await http.get(Uri.parse('$baseUrl/api/v1/users/$id'), headers: _headers);
    return jsonDecode(response.body);
  }

  static Future<Map<String, dynamic>> updateUser(String id, Map<String, dynamic> data) async {
    final response = await http.put(
      Uri.parse('$baseUrl/api/v1/users/$id'),
      headers: _headers,
      body: jsonEncode(data),
    );
    return jsonDecode(response.body);
  }

  static Future<void> suspendUser(String id, String reason) async {
    await http.post(
      Uri.parse('$baseUrl/api/v1/users/$id/suspend'),
      headers: _headers,
      body: jsonEncode({'reason': reason}),
    );
  }

  static Future<void> banUser(String id, String reason) async {
    await http.post(
      Uri.parse('$baseUrl/api/v1/users/$id/ban'),
      headers: _headers,
      body: jsonEncode({'reason': reason}),
    );
  }

  // KYC
  static Future<Map<String, dynamic>> getKYCSubmissions({int page = 1, int limit = 20, String? status, int? level}) async {
    final queryParams = {'page': page.toString(), 'limit': limit.toString()};
    if (status != null) queryParams['status'] = status;
    if (level != null) queryParams['level'] = level.toString();
    
    final response = await http.get(
      Uri.parse('$baseUrl/api/v1/kyc').replace(queryParameters: queryParams),
      headers: _headers,
    );
    return jsonDecode(response.body);
  }

  static Future<void> approveKYC(String id, {String? notes}) async {
    await http.post(
      Uri.parse('$baseUrl/api/v1/kyc/$id/approve'),
      headers: _headers,
      body: jsonEncode(notes != null ? {'notes': notes} : {}),
    );
  }

  static Future<void> rejectKYC(String id, String reason) async {
    await http.post(
      Uri.parse('$baseUrl/api/v1/kyc/$id/reject'),
      headers: _headers,
      body: jsonEncode({'reason': reason}),
    );
  }

  // Tokens
  static Future<Map<String, dynamic>> getTokens({int page = 1, int limit = 20, String? status, String? chain}) async {
    final queryParams = {'page': page.toString(), 'limit': limit.toString()};
    if (status != null) queryParams['status'] = status;
    if (chain != null) queryParams['chain'] = chain;
    
    final response = await http.get(
      Uri.parse('$baseUrl/api/v1/tokens').replace(queryParameters: queryParams),
      headers: _headers,
    );
    return jsonDecode(response.body);
  }

  static Future<Map<String, dynamic>> createToken(Map<String, dynamic> data) async {
    final response = await http.post(
      Uri.parse('$baseUrl/api/v1/tokens'),
      headers: _headers,
      body: jsonEncode(data),
    );
    return jsonDecode(response.body);
  }

  static Future<Map<String, dynamic>> updateToken(String id, Map<String, dynamic> data) async {
    final response = await http.put(
      Uri.parse('$baseUrl/api/v1/tokens/$id'),
      headers: _headers,
      body: jsonEncode(data),
    );
    return jsonDecode(response.body);
  }

  static Future<void> deleteToken(String id) async {
    await http.delete(Uri.parse('$baseUrl/api/v1/tokens/$id'), headers: _headers);
  }

  // Pairs
  static Future<Map<String, dynamic>> getPairs({int page = 1, int limit = 20, String? status, String? chain}) async {
    final queryParams = {'page': page.toString(), 'limit': limit.toString()};
    if (status != null) queryParams['status'] = status;
    if (chain != null) queryParams['chain'] = chain;
    
    final response = await http.get(
      Uri.parse('$baseUrl/api/v1/pairs').replace(queryParameters: queryParams),
      headers: _headers,
    );
    return jsonDecode(response.body);
  }

  static Future<Map<String, dynamic>> createPair(Map<String, dynamic> data) async {
    final response = await http.post(
      Uri.parse('$baseUrl/api/v1/pairs'),
      headers: _headers,
      body: jsonEncode(data),
    );
    return jsonDecode(response.body);
  }

  // Transactions
  static Future<Map<String, dynamic>> getTransactions({int page = 1, int limit = 20, String? status, String? type}) async {
    final queryParams = {'page': page.toString(), 'limit': limit.toString()};
    if (status != null) queryParams['status'] = status;
    if (type != null) queryParams['type'] = type;
    
    final response = await http.get(
      Uri.parse('$baseUrl/api/v1/transactions').replace(queryParameters: queryParams),
      headers: _headers,
    );
    return jsonDecode(response.body);
  }

  // Withdrawals
  static Future<Map<String, dynamic>> getWithdrawals({int page = 1, int limit = 20, String? status}) async {
    final queryParams = {'page': page.toString(), 'limit': limit.toString()};
    if (status != null) queryParams['status'] = status;
    
    final response = await http.get(
      Uri.parse('$baseUrl/api/v1/withdrawals').replace(queryParameters: queryParams),
      headers: _headers,
    );
    return jsonDecode(response.body);
  }

  static Future<void> approveWithdrawal(String id) async {
    await http.post(Uri.parse('$baseUrl/api/v1/withdrawals/$id/approve'), headers: _headers);
  }

  static Future<void> rejectWithdrawal(String id, String reason) async {
    await http.post(
      Uri.parse('$baseUrl/api/v1/withdrawals/$id/reject'),
      headers: _headers,
      body: jsonEncode({'reason': reason}),
    );
  }

  // Chains
  static Future<Map<String, dynamic>> getChains() async {
    final response = await http.get(Uri.parse('$baseUrl/api/v1/chains'), headers: _headers);
    return jsonDecode(response.body);
  }

  static Future<Map<String, dynamic>> createChain(Map<String, dynamic> data) async {
    final response = await http.post(
      Uri.parse('$baseUrl/api/v1/chains'),
      headers: _headers,
      body: jsonEncode(data),
    );
    return jsonDecode(response.body);
  }

  // Fees
  static Future<Map<String, dynamic>> getFees() async {
    final response = await http.get(Uri.parse('$baseUrl/api/v1/fees'), headers: _headers);
    return jsonDecode(response.body);
  }

  static Future<Map<String, dynamic>> createFee(Map<String, dynamic> data) async {
    final response = await http.post(
      Uri.parse('$baseUrl/api/v1/fees'),
      headers: _headers,
      body: jsonEncode(data),
    );
    return jsonDecode(response.body);
  }

  // White Labels
  static Future<Map<String, dynamic>> getWhiteLabels({int page = 1, int limit = 20, String? status}) async {
    final queryParams = {'page': page.toString(), 'limit': limit.toString()};
    if (status != null) queryParams['status'] = status;
    
    final response = await http.get(
      Uri.parse('$baseUrl/api/v1/white-labels').replace(queryParameters: queryParams),
      headers: _headers,
    );
    return jsonDecode(response.body);
  }

  static Future<Map<String, dynamic>> createWhiteLabel(Map<String, dynamic> data) async {
    final response = await http.post(
      Uri.parse('$baseUrl/api/v1/white-labels'),
      headers: _headers,
      body: jsonEncode(data),
    );
    return jsonDecode(response.body);
  }

  // Dashboard
  static Future<Map<String, dynamic>> getDashboard() async {
    final response = await http.get(Uri.parse('$baseUrl/api/v1/dashboard'), headers: _headers);
    return jsonDecode(response.body);
  }

  // Tickets
  static Future<Map<String, dynamic>> getTickets({int page = 1, int limit = 20, String? status, String? priority, String? category}) async {
    final queryParams = {'page': page.toString(), 'limit': limit.toString()};
    if (status != null) queryParams['status'] = status;
    if (priority != null) queryParams['priority'] = priority;
    if (category != null) queryParams['category'] = category;
    
    final response = await http.get(
      Uri.parse('$baseUrl/api/v1/tickets').replace(queryParameters: queryParams),
      headers: _headers,
    );
    return jsonDecode(response.body);
  }

  static Future<Map<String, dynamic>> createTicket(Map<String, dynamic> data) async {
    final response = await http.post(
      Uri.parse('$baseUrl/api/v1/tickets'),
      headers: _headers,
      body: jsonEncode(data),
    );
    return jsonDecode(response.body);
  }

  static Future<Map<String, dynamic>> updateTicket(String id, Map<String, dynamic> data) async {
    final response = await http.put(
      Uri.parse('$baseUrl/api/v1/tickets/$id'),
      headers: _headers,
      body: jsonEncode(data),
    );
    return jsonDecode(response.body);
  }

  static Future<void> addTicketMessage(String ticketId, String message) async {
    await http.post(
      Uri.parse('$baseUrl/api/v1/tickets/$ticketId/messages'),
      headers: _headers,
      body: jsonEncode({'message': message}),
    );
  }

  // Knowledge Base
  static Future<Map<String, dynamic>> getKnowledgeBaseArticles() async {
    final response = await http.get(Uri.parse('$baseUrl/api/v1/knowledge-base'), headers: _headers);
    return jsonDecode(response.body);
  }

  static Future<Map<String, dynamic>> createKnowledgeBaseArticle(Map<String, dynamic> data) async {
    final response = await http.post(
      Uri.parse('$baseUrl/api/v1/knowledge-base'),
      headers: _headers,
      body: jsonEncode(data),
    );
    return jsonDecode(response.body);
  }

  // Approval Workflows
  static Future<Map<String, dynamic>> getWorkflows() async {
    final response = await http.get(Uri.parse('$baseUrl/api/v1/workflows'), headers: _headers);
    return jsonDecode(response.body);
  }

  static Future<Map<String, dynamic>> createWorkflow(Map<String, dynamic> data) async {
    final response = await http.post(
      Uri.parse('$baseUrl/api/v1/workflows'),
      headers: _headers,
      body: jsonEncode(data),
    );
    return jsonDecode(response.body);
  }

  static Future<Map<String, dynamic>> getApprovalRequests({int page = 1, int limit = 20, String? status, String? workflowId}) async {
    final queryParams = {'page': page.toString(), 'limit': limit.toString()};
    if (status != null) queryParams['status'] = status;
    if (workflowId != null) queryParams['workflow_id'] = workflowId;
    
    final response = await http.get(
      Uri.parse('$baseUrl/api/v1/approval-requests').replace(queryParameters: queryParams),
      headers: _headers,
    );
    return jsonDecode(response.body);
  }

  static Future<void> approveRequest(String id) async {
    await http.post(Uri.parse('$baseUrl/api/v1/approval-requests/$id/approve'), headers: _headers);
  }

  static Future<void> rejectRequest(String id, String reason) async {
    await http.post(
      Uri.parse('$baseUrl/api/v1/approval-requests/$id/reject'),
      headers: _headers,
      body: jsonEncode({'reason': reason}),
    );
  }

  // Dashboards
  static Future<Map<String, dynamic>> getComplianceDashboard() async {
    final response = await http.get(Uri.parse('$baseUrl/api/v1/dashboard/compliance'), headers: _headers);
    return jsonDecode(response.body);
  }

  static Future<Map<String, dynamic>> getFinanceDashboard() async {
    final response = await http.get(Uri.parse('$baseUrl/api/v1/dashboard/finance'), headers: _headers);
    return jsonDecode(response.body);
  }

  static Future<Map<String, dynamic>> getSecurityDashboard() async {
    final response = await http.get(Uri.parse('$baseUrl/api/v1/dashboard/security'), headers: _headers);
    return jsonDecode(response.body);
  }

  // Notifications
  static Future<Map<String, dynamic>> getNotifications({int page = 1, int limit = 20, String? status}) async {
    final queryParams = {'page': page.toString(), 'limit': limit.toString()};
    if (status != null) queryParams['status'] = status;
    
    final response = await http.get(
      Uri.parse('$baseUrl/api/v1/notifications').replace(queryParameters: queryParams),
      headers: _headers,
    );
    return jsonDecode(response.body);
  }

  static Future<void> markNotificationRead(String id) async {
    await http.put(Uri.parse('$baseUrl/api/v1/notifications/$id/read'), headers: _headers);
  }

  static Future<void> sendNotification(Map<String, dynamic> data) async {
    await http.post(
      Uri.parse('$baseUrl/api/v1/notifications'),
      headers: _headers,
      body: jsonEncode(data),
    );
  }

  static Future<void> broadcastNotification(Map<String, dynamic> data) async {
    await http.post(
      Uri.parse('$baseUrl/api/v1/notifications/broadcast'),
      headers: _headers,
      body: jsonEncode(data),
    );
  }

  // API Keys
  static Future<Map<String, dynamic>> getAPIKeys() async {
    final response = await http.get(Uri.parse('$baseUrl/api/v1/api-keys'), headers: _headers);
    return jsonDecode(response.body);
  }

  static Future<Map<String, dynamic>> createAPIKey(Map<String, dynamic> data) async {
    final response = await http.post(
      Uri.parse('$baseUrl/api/v1/api-keys'),
      headers: _headers,
      body: jsonEncode(data),
    );
    return jsonDecode(response.body);
  }

  static Future<void> revokeAPIKey(String id) async {
    await http.post(Uri.parse('$baseUrl/api/v1/api-keys/$id/revoke'), headers: _headers);
  }

  // Webhooks
  static Future<Map<String, dynamic>> getWebhooks() async {
    final response = await http.get(Uri.parse('$baseUrl/api/v1/webhooks'), headers: _headers);
    return jsonDecode(response.body);
  }

  static Future<Map<String, dynamic>> createWebhook(Map<String, dynamic> data) async {
    final response = await http.post(
      Uri.parse('$baseUrl/api/v1/webhooks'),
      headers: _headers,
      body: jsonEncode(data),
    );
    return jsonDecode(response.body);
  }

  static Future<Map<String, dynamic>> updateWebhook(String id, Map<String, dynamic> data) async {
    final response = await http.put(
      Uri.parse('$baseUrl/api/v1/webhooks/$id'),
      headers: _headers,
      body: jsonEncode(data),
    );
    return jsonDecode(response.body);
  }

  static Future<void> deleteWebhook(String id) async {
    await http.delete(Uri.parse('$baseUrl/api/v1/webhooks/$id'), headers: _headers);
  }

  // Market Maker Bots
  static Future<Map<String, dynamic>> getMarketMakerBots({int page = 1, int limit = 20, String? status}) async {
    final queryParams = {'page': page.toString(), 'limit': limit.toString()};
    if (status != null) queryParams['status'] = status;
    
    final response = await http.get(
      Uri.parse('$baseUrl/api/v1/market-maker').replace(queryParameters: queryParams),
      headers: _headers,
    );
    return jsonDecode(response.body);
  }

  static Future<void> startBot(String id) async {
    await http.post(Uri.parse('$baseUrl/api/v1/market-maker/$id/start'), headers: _headers);
  }

  static Future<void> stopBot(String id) async {
    await http.post(Uri.parse('$baseUrl/api/v1/market-maker/$id/stop'), headers: _headers);
  }

  static Future<void> pauseBot(String id) async {
    await http.post(Uri.parse('$baseUrl/api/v1/market-maker/$id/pause'), headers: _headers);
  }

  // Admins (Super Admin)
  static Future<Map<String, dynamic>> getAdmins({int page = 1, int limit = 20, String? role}) async {
    final queryParams = {'page': page.toString(), 'limit': limit.toString()};
    if (role != null) queryParams['role'] = role;
    
    final response = await http.get(
      Uri.parse('$baseUrl/api/v1/admins').replace(queryParameters: queryParams),
      headers: _headers,
    );
    return jsonDecode(response.body);
  }

  static Future<Map<String, dynamic>> createAdmin(Map<String, dynamic> data) async {
    final response = await http.post(
      Uri.parse('$baseUrl/api/v1/admins'),
      headers: _headers,
      body: jsonEncode(data),
    );
    return jsonDecode(response.body);
  }

  static Future<Map<String, dynamic>> updateAdmin(String id, Map<String, dynamic> data) async {
    final response = await http.put(
      Uri.parse('$baseUrl/api/v1/admins/$id'),
      headers: _headers,
      body: jsonEncode(data),
    );
    return jsonDecode(response.body);
  }

  static Future<void> deleteAdmin(String id) async {
    await http.delete(Uri.parse('$baseUrl/api/v1/admins/$id'), headers: _headers);
  }

  // Audit Logs
  static Future<Map<String, dynamic>> getAuditLogs({int page = 1, int limit = 20, String? adminId, String? action}) async {
    final queryParams = {'page': page.toString(), 'limit': limit.toString()};
    if (adminId != null) queryParams['admin_id'] = adminId;
    if (action != null) queryParams['action'] = action;
    
    final response = await http.get(
      Uri.parse('$baseUrl/api/v1/audit-logs').replace(queryParameters: queryParams),
      headers: _headers,
    );
    return jsonDecode(response.body);
  }
}
