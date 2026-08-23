// Launchpad Service - Flutter Mobile
// Token launchpad and IDO platform

import 'dart:convert';
import 'package:http/http.dart' as http;

class LaunchpadService {
  static const String API_BASE = 'http://localhost:8443/api/v1';
  String? _token;
  
  LaunchpadService({String? token}) : _token = token;
  
  Map<String, String> get _headers => {
    'Content-Type': 'application/json',
    if (_token != null) 'Authorization': 'Bearer $_token',
  };
  
  // Get active launches
  Future<List<Launch>> getActiveLaunches() async {
    final response = await http.get(
      Uri.parse('$API_BASE/launchpad/active'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((l) => Launch.fromJson(l)).toList();
    }
    return [];
  }
  
  // Get upcoming launches
  Future<List<Launch>> getUpcomingLaunches() async {
    final response = await http.get(
      Uri.parse('$API_BASE/launchpad/upcoming'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((l) => Launch.fromJson(l)).toList();
    }
    return [];
  }
  
  // Get launch details
  Future<Launch> getLaunchDetails(String launchId) async {
    final response = await http.get(
      Uri.parse('$API_BASE/launchpad/$launchId'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return Launch.fromJson(data['data']);
    }
    throw Exception('Failed to get launch');
  }
  
  // Participate in IDO
  Future<Participation> participate(String launchId, double amount) async {
    final response = await http.post(
      Uri.parse('$API_BASE/launchpad/$launchId/participate'),
      headers: _headers,
      body: json.encode({'amount': amount}),
    );
    
    if (response.statusCode == 201) {
      final data = json.decode(response.body);
      return Participation.fromJson(data['data']);
    }
    throw Exception('Failed to participate');
  }
  
  // Claim tokens
  Future<bool> claimTokens(String launchId) async {
    final response = await http.post(
      Uri.parse('$API_BASE/launchpad/$launchId/claim'),
      headers: _headers,
    );
    
    return response.statusCode == 200;
  }
  
  // Get user participations
  Future<List<Participation>> getUserParticipations() async {
    final response = await http.get(
      Uri.parse('$API_BASE/launchpad/participations'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((p) => Participation.fromJson(p)).toList();
    }
    return [];
  }
}

class Launch {
  final String id;
  final String name;
  final String symbol;
  final String description;
  final String website;
  final String whitepaper;
  final String tokenAddress;
  final String saleToken;
  final double price;
  final double hardCap;
  final double softCap;
  final double raised;
  final String startTime;
  final String endTime;
  final String status;
  final List<String> links;
  
  Launch({
    required this.id,
    required this.name,
    required this.symbol,
    required this.description,
    required this.website,
    required this.whitepaper,
    required this.tokenAddress,
    required this.saleToken,
    required this.price,
    required this.hardCap,
    required this.softCap,
    required this.raised,
    required this.startTime,
    required this.endTime,
    required this.status,
    required this.links,
  });
  
  factory Launch.fromJson(Map<String, dynamic> json) {
    return Launch(
      id: json['id'] ?? '',
      name: json['name'] ?? '',
      symbol: json['symbol'] ?? '',
      description: json['description'] ?? '',
      website: json['website'] ?? '',
      whitepaper: json['whitepaper'] ?? '',
      tokenAddress: json['tokenAddress'] ?? '',
      saleToken: json['saleToken'] ?? '',
      price: (json['price'] ?? 0).toDouble(),
      hardCap: (json['hardCap'] ?? 0).toDouble(),
      softCap: (json['softCap'] ?? 0).toDouble(),
      raised: (json['raised'] ?? 0).toDouble(),
      startTime: json['startTime'] ?? '',
      endTime: json['endTime'] ?? '',
      status: json['status'] ?? 'UPCOMING',
      links: List<String>.from(json['links'] ?? []),
    );
  }
}

class Participation {
  final String id;
  final String launchId;
  final double amount;
  final double tokenAmount;
  final String status;
  final DateTime participatedAt;
  final DateTime? claimedAt;
  
  Participation({
    required this.id,
    required this.launchId,
    required this.amount,
    required this.tokenAmount,
    required this.status,
    required this.participatedAt,
    this.claimedAt,
  });
  
  factory Participation.fromJson(Map<String, dynamic> json) {
    return Participation(
      id: json['id'] ?? '',
      launchId: json['launchId'] ?? '',
      amount: (json['amount'] ?? 0).toDouble(),
      tokenAmount: (json['tokenAmount'] ?? 0).toDouble(),
      status: json['status'] ?? 'PENDING',
      participatedAt: DateTime.parse(json['participatedAt']),
      claimedAt: json['claimedAt'] != null ? DateTime.parse(json['claimedAt']) : null,
    );
  }
}
