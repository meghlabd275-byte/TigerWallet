/**
 * AuthService - Flutter Implementation
 *
 * Thin REST client over the canonical Go backend (:8450) auth routes. Both
 * register and login target the canonical /api/v1/auth/* endpoints and return
 * the JWT + identity issued server-side. No local token minting, no hardcoded
 * users, no mock credentials. The backend is the sole source of truth.
 *
 * See master_wallet/CANONICAL_API_CONTRACT.md §Auth.
 */

import 'dart:convert';
import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;

class AuthService extends ChangeNotifier {
  static const String API_BASE = String.fromEnvironment(
    'MASTER_WALLET_API_URL',
    defaultValue: 'http://localhost:8450',
  );
  static const String _apiV1 = '$API_BASE/api/v1';

  String? _token;

  AuthService({String? token}) : _token = token;

  /// The JWT from the most recent successful register/login, if any.
  String? get token => _token;

  void setToken(String? token) {
    _token = token;
    notifyListeners();
  }

  Map<String, String> get _headers => {
        'Content-Type': 'application/json',
        if (_token != null) 'Authorization': 'Bearer $_token',
      };

  Exception _error(http.Response r, String op) =>
      Exception('auth $op failed (${r.statusCode}): ${r.body}');

  AuthSession _session(Map<String, dynamic> data) => AuthSession(
        token: data['token'] as String?,
        userId: data['user_id'] as String?,
        email: data['email'] as String?,
        role: data['role'] as String?,
      );

  /// Register a new account via POST /api/v1/auth/register.
  /// Body: {email, password, name}. On success the backend issues a JWT and
  /// returns {token, user_id, email, role}, which we cache for subsequent
  /// authenticated calls. Throws on any non-2xx / network error.
  Future<AuthSession> register({
    required String email,
    required String password,
    String? name,
  }) async {
    final r = await http.post(
      Uri.parse('$_apiV1/auth/register'),
      headers: _headers,
      body: jsonEncode({
        'email': email,
        'password': password,
        if (name != null) 'name': name,
      }),
    );
    if (r.statusCode != 200 && r.statusCode != 201) throw _error(r, 'register');
    final data = jsonDecode(r.body) as Map<String, dynamic>;
    final session = _session(data);
    if (session.token != null) {
      _token = session.token;
      notifyListeners();
    }
    return session;
  }

  /// Log in via POST /api/v1/auth/login.
  /// Body: {email, password}. Returns the server-issued JWT + identity.
  Future<AuthSession> login({
    required String email,
    required String password,
  }) async {
    final r = await http.post(
      Uri.parse('$_apiV1/auth/login'),
      headers: _headers,
      body: jsonEncode({'email': email, 'password': password}),
    );
    if (r.statusCode != 200) throw _error(r, 'login');
    final data = jsonDecode(r.body) as Map<String, dynamic>;
    final session = _session(data);
    if (session.token != null) {
      _token = session.token;
      notifyListeners();
    }
    return session;
  }

  /// Clear the cached JWT. The backend manages session validity via JWT
  /// expiry; there is no server-side logout endpoint in the canonical contract.
  void logout() {
    _token = null;
    notifyListeners();
  }
}

/// Identity + JWT returned by the canonical auth endpoints.
class AuthSession {
  final String? token;
  final String? userId;
  final String? email;
  final String? role;

  AuthSession({this.token, this.userId, this.email, this.role});

  bool get isAuthenticated => token != null && token!.isNotEmpty;
}
