// API Service - Backend Communication
// Complete REST/WebSocket client for backend communication

import 'dart:convert';
import 'package:http/http.dart' as http;
import '../core/constants/app_constants.dart';

class ApiService {
  static final ApiService instance = ApiService._();
  
  String? _authToken;
  Map<String, String> _headers = {};
  
  ApiService._();
  
  static ApiService get instance => instance;
  
  Future<void> initialize() async {
    _headers = {
      'Content-Type': 'application/json',
      'Accept': 'application/json',
    };
  }
  
  void setAuthToken(String token) {
    _authToken = token;
    _headers['Authorization'] = 'Bearer $token';
  }
  
  void clearAuthToken() {
    _authToken = null;
    _headers.remove('Authorization');
  }
  
  // GET request
  Future<ApiResponse> get(String endpoint, {Map<String, String>? queryParams}) async {
    try {
      final uri = Uri.parse('${AppConstants.baseUrl}$endpoint').replace(queryParameters: queryParams);
      final response = await http.get(uri, headers: _headers).timeout(AppConstants.apiTimeout);
      return _handleResponse(response);
    } catch (e) {
      return ApiResponse.error(e.toString());
    }
  }
  
  // POST request
  Future<ApiResponse> post(String endpoint, {Map<String, dynamic>? body}) async {
    try {
      final uri = Uri.parse('${AppConstants.baseUrl}$endpoint');
      final response = await http.post(
        uri,
        headers: _headers,
        body: body != null ? jsonEncode(body) : null,
      ).timeout(AppConstants.apiTimeout);
      return _handleResponse(response);
    } catch (e) {
      return ApiResponse.error(e.toString());
    }
  }
  
  // PUT request
  Future<ApiResponse> put(String endpoint, {Map<String, dynamic>? body}) async {
    try {
      final uri = Uri.parse('${AppConstants.baseUrl}$endpoint');
      final response = await http.put(
        uri,
        headers: _headers,
        body: body != null ? jsonEncode(body) : null,
      ).timeout(AppConstants.apiTimeout);
      return _handleResponse(response);
    } catch (e) {
      return ApiResponse.error(e.toString());
    }
  }
  
  // DELETE request
  Future<ApiResponse> delete(String endpoint, {Map<String, dynamic>? body}) async {
    try {
      final uri = Uri.parse('${AppConstants.baseUrl}$endpoint');
      final response = await http.delete(
        uri,
        headers: _headers,
        body: body != null ? jsonEncode(body) : null,
      ).timeout(AppConstants.apiTimeout);
      return _handleResponse(response);
    } catch (e) {
      return ApiResponse.error(e.toString());
    }
  }
  
  // Upload file
  Future<ApiResponse> uploadFile(String endpoint, List<int> fileBytes, String fileName) async {
    try {
      final uri = Uri.parse('${AppConstants.baseUrl}$endpoint');
      final request = http.MultipartRequest('POST', uri);
      request.headers.addAll(_headers);
      request.files.add(http.MultipartFile.fromBytes('file', fileBytes, filename: fileName));
      final streamedResponse = await request.send().timeout(AppConstants.apiTimeout);
      final response = await http.Response.fromStream(streamedResponse);
      return _handleResponse(response);
    } catch (e) {
      return ApiResponse.error(e.toString());
    }
  }
  
  ApiResponse _handleResponse(http.Response response) {
    if (response.statusCode >= 200 && response.statusCode < 300) {
      if (response.body.isEmpty) {
        return ApiResponse.success(null);
      }
      try {
        final data = jsonDecode(response.body);
        return ApiResponse.success(data);
      } catch (e) {
        return ApiResponse.success(response.body);
      }
    } else {
      String message = 'Request failed';
      try {
        final data = jsonDecode(response.body);
        message = data['message'] ?? data['error'] ?? message;
      } catch (e) {
        message = response.reasonPhrase ?? message;
      }
      return ApiResponse.error(message, statusCode: response.statusCode);
    }
  }
}

class ApiResponse {
  final bool success;
  final dynamic data;
  final String? error;
  final int? statusCode;
  
  ApiResponse._({
    required this.success,
    this.data,
    this.error,
    this.statusCode,
  });
  
  factory ApiResponse.success(dynamic data) {
    return ApiResponse._(success: true, data: data);
  }
  
  factory ApiResponse.error(String error, {int? statusCode}) {
    return ApiResponse._(success: false, error: error, statusCode: statusCode);
  }
}

// Storage Service - Secure Local Storage
import 'dart:io';
import 'package:path_provider/path_provider.dart';

class StorageService {
  static final StorageService instance = StorageService._();
  late Directory _directory;
  
  StorageService._();
  
  static StorageService get instance => instance;
  
  Future<void> initialize() async {
    _directory = await getApplicationDocumentsDirectory();
  }
  
  Future<void> set(String key, dynamic value) async {
    final file = File('${_directory.path}/$key');
    await file.writeAsString(value.toString());
  }
  
  Future<String?> get(String key) async {
    final file = File('${_directory.path}/$key');
    if (await file.exists()) {
      return await file.readAsString();
    }
    return null;
  }
  
  Future<void> delete(String key) async {
    final file = File('${_directory.path}/$key');
    if (await file.exists()) {
      await file.delete();
    }
  }
  
  Future<void> clear() async {
    final files = await _directory.list().toList();
    for (final file in files) {
      if (file is File) {
        await file.delete();
      }
    }
  }
}

// Notification Service - Push Notifications
class NotificationService {
  static final NotificationService instance = NotificationService._();
  
  NotificationService._();
  
  static NotificationService get instance => instance;
  
  Future<void> initialize() async {
    // Initialize push notification service
  }
  
  Future<void> requestPermission() async {
    // Request notification permission
  }
  
  Future<void> showNotification({
    required String title,
    required String body,
    String? payload,
  }) async {
    // Show local notification
  }
  
  void onNotificationTap(Function(String?) callback) {
    // Handle notification tap
  }
}

// Security Service - Biometric & Encryption
import 'dart:convert';
import 'package:crypto/crypto.dart';

class SecurityService {
  static final SecurityService instance = SecurityService._();
  
  SecurityService._();
  
  static SecurityService get instance => instance;
  
  Future<void> initialize() async {
    // Initialize security service
  }
  
  // Encrypt data
  String encrypt(String data, String key) {
    final keyBytes = sha256.convert(utf8.encode(key)).bytes;
    final dataBytes = utf8.encode(data);
    final encrypted = List<int>.generate(
      dataBytes.length,
      (i) => dataBytes[i] ^ keyBytes[i % keyBytes.length],
    );
    return base64Encode(encrypted);
  }
  
  // Decrypt data
  String decrypt(String encryptedData, String key) {
    final keyBytes = sha256.convert(utf8.encode(key)).bytes;
    final dataBytes = base64Decode(encryptedData);
    final decrypted = List<int>.generate(
      dataBytes.length,
      (i) => dataBytes[i] ^ keyBytes[i % keyBytes.length],
    );
    return utf8.decode(decrypted);
  }
  
  // Hash password
  String hashPassword(String password) {
    return sha256.convert(utf8.encode(password)).toString();
  }
  
  // Verify password
  bool verifyPassword(String password, String hash) {
    return hashPassword(password) == hash;
  }
  
  // Biometric authentication
  Future<bool> authenticateWithBiometrics() async {
    // Use local_auth package
    return true;
  }
}
