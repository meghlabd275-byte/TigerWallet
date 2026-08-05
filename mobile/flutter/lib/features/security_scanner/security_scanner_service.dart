/// Security Scanner Service for Flutter
/// Scans contracts and addresses for security risks

import 'dart:convert';
import 'package:http/http.dart' as http;

class SecurityScannerService {
  static const String _baseUrl = 'https://api.tigerwallet.com/v1/security';
  
  final http.Client _client;
  
  SecurityScannerService({http.Client? client}) : _client = client ?? http.Client();
  
  /// Scan a contract address for vulnerabilities
  Future<ContractScanResult> scanContract(String address, String chain) async {
    final response = await _client.post(
      Uri.parse('$_baseUrl/scan/contract'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'address': address, 'chain': chain}),
    );
    
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return ContractScanResult.fromJson(data);
    }
    throw Exception('Failed to scan contract: ${response.body}');
  }
  
  /// Scan a token for risks
  Future<TokenScanResult> scanToken(String address, String chain) async {
    final response = await _client.post(
      Uri.parse('$_baseUrl/scan/token'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'address': address, 'chain': chain}),
    );
    
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return TokenScanResult.fromJson(data);
    }
    throw Exception('Failed to scan token: ${response.body}');
  }
  
  /// Check if an address is flagged
  Future<AddressCheck> checkAddress(String address) async {
    final response = await _client.get(
      Uri.parse('$_baseUrl/check/$address'),
    );
    
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return AddressCheck.fromJson(data);
    }
    throw Exception('Failed to check address: ${response.body}');
  }
  
  /// Simulate a transaction before execution
  Future<SimulationResult> simulateTransaction({
    required String from,
    required String to,
    required String data,
    required String value,
    required String chain,
  }) async {
    final response = await _client.post(
      Uri.parse('$_baseUrl/simulate'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        'from': from,
        'to': to,
        'data': data,
        'value': value,
        'chain': chain,
      }),
    );
    
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return SimulationResult.fromJson(data);
    }
    throw Exception('Failed to simulate: ${response.body}');
  }
  
  /// Get security alerts
  Future<List<SecurityAlert>> getSecurityAlerts() async {
    final response = await _client.get(
      Uri.parse('$_baseUrl/alerts'),
    );
    
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return (data['alerts'] as List)
          .map((e) => SecurityAlert.fromJson(e))
          .toList();
    }
    throw Exception('Failed to get alerts: ${response.body}');
  }
  
  void dispose() {
    _client.close();
  }
}

class ContractScanResult {
  final String address;
  final String chain;
  final String scanId;
  final int score;
  final String riskLevel;
  final List<SecurityIssue> issues;
  final bool whitelisted;
  final DateTime lastScanned;
  
  ContractScanResult({
    required this.address,
    required this.chain,
    required this.scanId,
    required this.score,
    required this.riskLevel,
    required this.issues,
    required this.whitelisted,
    required this.lastScanned,
  });
  
  factory ContractScanResult.fromJson(Map<String, dynamic> json) {
    return ContractScanResult(
      address: json['address'],
      chain: json['chain'],
      scanId: json['scanId'],
      score: json['score'],
      riskLevel: json['riskLevel'],
      issues: (json['issues'] as List)
          .map((e) => SecurityIssue.fromJson(e))
          .toList(),
      whitelisted: json['whitelisted'],
      lastScanned: DateTime.fromMillisecondsSinceEpoch(json['lastScanned'] * 1000),
    );
  }
}

class SecurityIssue {
  final String id;
  final String title;
  final String description;
  final String severity;
  final String category;
  
  SecurityIssue({
    required this.id,
    required this.title,
    required this.description,
    required this.severity,
    required this.category,
  });
  
  factory SecurityIssue.fromJson(Map<String, dynamic> json) {
    return SecurityIssue(
      id: json['id'],
      title: json['title'],
      description: json['description'],
      severity: json['severity'],
      category: json['category'],
    );
  }
}

class TokenScanResult {
  final String address;
  final String name;
  final String symbol;
  final String totalSupply;
  final int holders;
  final int transfers;
  final bool isMintable;
  final bool isPausable;
  final bool isBlacklisted;
  final int trustScore;
  final bool verified;
  
  TokenScanResult({
    required this.address,
    required this.name,
    required this.symbol,
    required this.totalSupply,
    required this.holders,
    required this.transfers,
    required this.isMintable,
    required this.isPausable,
    required this.isBlacklisted,
    required this.trustScore,
    required this.verified,
  });
  
  factory TokenScanResult.fromJson(Map<String, dynamic> json) {
    return TokenScanResult(
      address: json['address'],
      name: json['name'],
      symbol: json['symbol'],
      totalSupply: json['totalSupply'],
      holders: json['holders'],
      transfers: json['transfers'],
      isMintable: json['isMintable'],
      isPausable: json['isPausable'],
      isBlacklisted: json['isBlacklisted'],
      trustScore: json['trustScore'],
      verified: json['verified'],
    );
  }
}

class AddressCheck {
  final String address;
  final bool isFlagged;
  final bool isScam;
  final bool isPhishing;
  final int reports;
  final DateTime firstSeen;
  final DateTime lastActivity;
  final List<String> labels;
  
  AddressCheck({
    required this.address,
    required this.isFlagged,
    required this.isScam,
    required this.isPhishing,
    required this.reports,
    required this.firstSeen,
    required this.lastActivity,
    required this.labels,
  });
  
  factory AddressCheck.fromJson(Map<String, dynamic> json) {
    return AddressCheck(
      address: json['address'],
      isFlagged: json['isFlagged'],
      isScam: json['isScam'],
      isPhishing: json['isPhishing'],
      reports: json['reports'],
      firstSeen: DateTime.fromMillisecondsSinceEpoch(json['firstSeen'] * 1000),
      lastActivity: DateTime.fromMillisecondsSinceEpoch(json['lastActivity'] * 1000),
      labels: List<String>.from(json['labels']),
    );
  }
}

class SimulationResult {
  final bool success;
  final int gasUsed;
  final List<String> stateChanges;
  final bool reverts;
  final String revertReason;
  
  SimulationResult({
    required this.success,
    required this.gasUsed,
    required this.stateChanges,
    required this.reverts,
    required this.revertReason,
  });
  
  factory SimulationResult.fromJson(Map<String, dynamic> json) {
    return SimulationResult(
      success: json['success'],
      gasUsed: json['gasUsed'],
      stateChanges: List<String>.from(json['stateChanges']),
      reverts: json['reverts'],
      revertReason: json['revertReason'],
    );
  }
}

class SecurityAlert {
  final String id;
  final String title;
  final String description;
  final String severity;
  final List<String> affectedAddresses;
  final DateTime publishedAt;
  
  SecurityAlert({
    required this.id,
    required this.title,
    required this.description,
    required this.severity,
    required this.affectedAddresses,
    required this.publishedAt,
  });
  
  factory SecurityAlert.fromJson(Map<String, dynamic> json) {
    return SecurityAlert(
      id: json['id'],
      title: json['title'],
      description: json['description'],
      severity: json['severity'],
      affectedAddresses: List<String>.from(json['affectedAddresses']),
      publishedAt: DateTime.fromMillisecondsSinceEpoch(json['publishedAt'] * 1000),
    );
  }
}
