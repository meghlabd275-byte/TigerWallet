/**
 * TigerWallet Admin - Auth Provider
 */

import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../../core/constants/app_constants.dart';
import '../../../core/network/dio_client.dart';

class AuthState {
  final bool isAuthenticated;
  final String? token;
  final Map<String, dynamic>? user;
  final bool isLoading;
  final String? error;
  
  const AuthState({
    this.isAuthenticated = false,
    this.token,
    this.user,
    this.isLoading = false,
    this.error,
  });
  
  AuthState copyWith({
    bool? isAuthenticated,
    String? token,
    Map<String, dynamic>? user,
    bool? isLoading,
    String? error,
  }) {
    return AuthState(
      isAuthenticated: isAuthenticated ?? this.isAuthenticated,
      token: token ?? this.token,
      user: user ?? this.user,
      isLoading: isLoading ?? this.isLoading,
      error: error,
    );
  }
}

class AuthNotifier extends StateNotifier<AuthState> {
  final DioClient _apiClient;
  final SharedPreferences _prefs;
  
  AuthNotifier(this._apiClient, this._prefs) : super(const AuthState());
  
  Future<void> login(String email, String password) async {
    state = state.copyWith(isLoading: true, error: null);
    
    try {
      final response = await _apiClient.login(email, password);
      
      await _prefs.setString(AppConstants.tokenKey, response['token']);
      await _prefs.setString(AppConstants.refreshTokenKey, response['refresh_token']);
      
      state = state.copyWith(
        isAuthenticated: true,
        token: response['token'],
        user: response['admin'],
        isLoading: false,
      );
    } catch (e) {
      state = state.copyWith(
        isLoading: false,
        error: e.toString(),
      );
    }
  }
  
  Future<void> logout() async {
    state = state.copyWith(isLoading: true);
    
    try {
      await _apiClient.logout();
    } finally {
      await _prefs.remove(AppConstants.tokenKey);
      await _prefs.remove(AppConstants.refreshTokenKey);
      
      state = const AuthState();
    }
  }
  
  Future<void> checkAuth() async {
    final token = _prefs.getString(AppConstants.tokenKey);
    
    if (token != null) {
      state = state.copyWith(
        isAuthenticated: true,
        token: token,
      );
    }
  }
}

final authStateProvider = StateNotifierProvider<AuthNotifier, AuthState>((ref) {
  final prefs = ref.watch(sharedPreferencesProvider);
  final apiClient = DioClient.withPrefs(prefs);
  return AuthNotifier(apiClient, prefs);
});

final sharedPreferencesProvider = Provider<SharedPreferences>((ref) {
  throw UnimplementedError('SharedPreferences not initialized');
});
