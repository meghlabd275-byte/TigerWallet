/// Passkey Service for Flutter
/// WebAuthn/FIDO2 passwordless authentication

import 'dart:convert';
import 'package:http/http.dart' as http;

class PasskeyService {
  static const String _baseUrl = 'http://localhost:8443/api/v1/passkey';
  
  final http.Client _client;
  
  PasskeyService({http.Client? client}) : _client = client ?? http.Client();
  
  /// Register a new passkey
  Future<PasskeyCredential> register({
    required String username,
    required String domain,
  }) async {
    // Step 1: Get challenge from server
    final challengeResponse = await _client.post(
      Uri.parse('$_baseUrl/register'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        'username': username,
        'domain': domain,
      }),
    );
    
    if (challengeResponse.statusCode != 200) {
      throw Exception('Failed to start registration: ${challengeResponse.body}');
    }
    
    final challenge = jsonDecode(challengeResponse.body);
    
    // In a real implementation, this would use the web_authn package
    // to create a credential on the device
    final credential = await _createCredential(username, domain, challenge);
    
    // Step 2: Verify with server
    final verifyResponse = await _client.post(
      Uri.parse('$_baseUrl/register/verify'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        'username': username,
        'credential': credential,
      }),
    );
    
    if (verifyResponse.statusCode == 200) {
      final data = jsonDecode(verifyResponse.body);
      return PasskeyCredential.fromJson(data);
    }
    throw Exception('Failed to verify: ${verifyResponse.body}');
  }
  
  /// Authenticate with passkey
  Future<AuthResult> authenticate(String username) async {
    // Step 1: Get challenge
    final challengeResponse = await _client.post(
      Uri.parse('$_baseUrl/authenticate'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'username': username}),
    );
    
    if (challengeResponse.statusCode != 200) {
      throw Exception('Failed to start authentication: ${challengeResponse.body}');
    }
    
    final challenge = jsonDecode(challengeResponse.body);
    
    // Step 2: Get credential from device
    final credential = await _getCredential(username, challenge);
    
    // Step 3: Verify
    final verifyResponse = await _client.post(
      Uri.parse('$_baseUrl/authenticate/verify'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        'username': username,
        'credential': credential,
      }),
    );
    
    if (verifyResponse.statusCode == 200) {
      final data = jsonDecode(verifyResponse.body);
      return AuthResult.fromJson(data);
    }
    throw Exception('Failed to verify: ${verifyResponse.body}');
  }
  
  /// List user's passkeys
  Future<List<PasskeyCredential>> listCredentials(String username) async {
    final response = await _client.get(
      Uri.parse('$_baseUrl/credentials/$username'),
    );
    
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return (data['credentials'] as List)
          .map((e) => PasskeyCredential.fromJson(e))
          .toList();
    }
    throw Exception('Failed to list credentials: ${response.body}');
  }
  
  /// Delete a passkey
  Future<bool> deleteCredential(String credentialId) async {
    final response = await _client.delete(
      Uri.parse('$_baseUrl/credentials/$credentialId'),
    );
    
    return response.statusCode == 200;
  }
  
  /// Check if device supports passkeys
  Future<bool> isSupported() async {
    // In a real implementation, this would check platform capabilities
    return true;
  }
  
  // Private methods that would interact with platform-specific APIs
  Future<Map<String, dynamic>> _createCredential(
    String username,
    String domain,
    Map<String, dynamic> challenge,
  ) async {
    // This is a placeholder - in production, use the web_authn package
    return {
      'id': 'credential_id_${DateTime.now().millisecondsSinceEpoch}',
      'rawId': 'raw_credential_id',
      'type': 'public-key',
      'response': {
        'clientDataJSON': jsonEncode({
          'challenge': challenge['challenge'],
          'origin': domain,
          'type': 'webauthn.create',
        }),
        'attestationObject': 'attestation_object',
      },
    };
  }
  
  Future<Map<String, dynamic>> _getCredential(
    String username,
    Map<String, dynamic> challenge,
  ) async {
    // This is a placeholder - in production, use the web_authn package
    return {
      'id': 'credential_id',
      'rawId': 'raw_credential_id',
      'type': 'public-key',
      'response': {
        'clientDataJSON': jsonEncode({
          'challenge': challenge['challenge'],
          'origin': 'tigerwallet.com',
          'type': 'webauthn.get',
        }),
        'signature': 'signature',
      },
    };
  }
  
  void dispose() {
    _client.close();
  }
}

class PasskeyCredential {
  final String credentialId;
  final String username;
  final String domain;
  final DateTime createdAt;
  final DateTime? lastUsed;
  final String? credentialType;
  
  PasskeyCredential({
    required this.credentialId,
    required this.username,
    required this.domain,
    required this.createdAt,
    this.lastUsed,
    this.credentialType,
  });
  
  factory PasskeyCredential.fromJson(Map<String, dynamic> json) {
    return PasskeyCredential(
      credentialId: json['credentialId'],
      username: json['username'],
      domain: json['domain'],
      createdAt: DateTime.parse(json['createdAt']),
      lastUsed: json['lastUsed'] != null 
          ? DateTime.parse(json['lastUsed']) 
          : null,
      credentialType: json['credentialType'],
    );
  }
}

class AuthResult {
  final bool success;
  final String? token;
  final DateTime expiresAt;
  
  AuthResult({
    required this.success,
    this.token,
    required this.expiresAt,
  });
  
  factory AuthResult.fromJson(Map<String, dynamic> json) {
    return AuthResult(
      success: json['success'],
      token: json['token'],
      expiresAt: DateTime.parse(json['expiresAt']),
    );
  }
}
