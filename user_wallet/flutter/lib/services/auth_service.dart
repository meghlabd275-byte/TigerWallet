/**
 * AuthService — JWT lifecycle for the UserWallet Flutter app.
 * Stores the token in SharedPreferences and propagates it to UserWalletService.
 * No registration is required by UserWallet (guest bootstrap allowed), but
 * explicit email register/login is supported — the directive's onboarding is
 * wallet create/import; auth here is only for JWT-scoped routes.
 */

import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'user_wallet.dart';

class AuthService extends ChangeNotifier {
  static const String _tokenKey = 'userwallet_jwt';
  String? _token;
  String? _email;
  bool _initialized = false;

  String? get token => _token;
  String? get email => _email;
  bool get isAuthenticated => _token != null;
  bool get initialized => _initialized;

  final UserWalletService api = UserWalletService();

  Future<void> initialize() async {
    if (_initialized) return;
    final prefs = await SharedPreferences.getInstance();
    _token = prefs.getString(_tokenKey);
    _email = prefs.getString('userwallet_email');
    api.setToken(_token);
    _initialized = true;
    notifyListeners();
  }

  Future<void> _persist(String? token, String? email) async {
    final prefs = await SharedPreferences.getInstance();
    if (token == null) {
      await prefs.remove(_tokenKey);
      await prefs.remove('userwallet_email');
    } else {
      await prefs.setString(_tokenKey, token);
      if (email != null) await prefs.setString('userwallet_email', email);
    }
  }

  Future<void> login(String email, String password, {bool register = false}) async {
    final res = register
        ? await api.register(email, password)
        : await api.login(email, password);
    final token = res?['token'] ?? res?['jwt'];
    if (token == null) {
      throw ApiException(401, 'Backend did not return a token');
    }
    _token = token.toString();
    _email = email;
    api.setToken(_token);
    await _persist(_token, _email);
    notifyListeners();
  }

  Future<void> guest() async {
    final res = await api.guest();
    final token = res?['token'] ?? res?['jwt'];
    if (token == null) {
      throw ApiException(401, 'Backend did not return a guest token');
    }
    _token = token.toString();
    _email = 'guest';
    api.setToken(_token);
    await _persist(_token, _email);
    notifyListeners();
  }

  Future<void> logout() async {
    _token = null;
    _email = null;
    api.setToken(null);
    await _persist(null, null);
    notifyListeners();
  }
}
