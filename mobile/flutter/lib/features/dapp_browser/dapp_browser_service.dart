// DApp Browser Service - Flutter Mobile
// Complete DApp browsing with Web3 injection

import 'dart:convert';
import 'package:http/http.dart' as http;

class DAppBrowserService {
  static const String API_BASE = 'http://localhost:8443/api/v1';
  String? _token;
  
  DAppBrowserService({String? token}) : _token = token;
  
  Map<String, String> get _headers => {
    'Content-Type': 'application/json',
    if (_token != null) 'Authorization': 'Bearer $_token',
  };
  
  // Get list of featured DApps
  Future<List<DApp>> getFeaturedDApps() async {
    final response = await http.get(
      Uri.parse('$API_BASE/dapps/featured'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((d) => DApp.fromJson(d)).toList();
    }
    throw Exception('Failed to load featured DApps');
  }
  
  // Get DApp categories
  Future<List<DAppCategory>> getCategories() async {
    final response = await http.get(
      Uri.parse('$API_BASE/dapps/categories'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((c) => DAppCategory.fromJson(c)).toList();
    }
    throw Exception('Failed to load categories');
  }
  
  // Search DApps
  Future<List<DApp>> searchDApps(String query) async {
    final response = await http.get(
      Uri.parse('$API_BASE/dapps/search?q=$query'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((d) => DApp.fromJson(d)).toList();
    }
    throw Exception('Failed to search DApps');
  }
  
  // Get DApp details
  Future<DApp> getDAppDetails(String dappId) async {
    final response = await http.get(
      Uri.parse('$API_BASE/dapps/$dappId'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return DApp.fromJson(data['data']);
    }
    throw Exception('Failed to load DApp details');
  }
  
  // Submit DApp for listing
  Future<bool> submitDApp({
    required String name,
    required String url,
    required String category,
    required String description,
    required String logoUrl,
  }) async {
    final response = await http.post(
      Uri.parse('$API_BASE/dapps/submit'),
      headers: _headers,
      body: json.encode({
        'name': name,
        'url': url,
        'category': category,
        'description': description,
        'logoUrl': logoUrl,
      }),
    );
    
    return response.statusCode == 201;
  }
  
  // Get user favorites
  Future<List<String>> getFavorites() async {
    final response = await http.get(
      Uri.parse('$API_BASE/dapps/favorites'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return List<String>.from(data['data']);
    }
    return [];
  }
  
  // Add to favorites
  Future<bool> addFavorite(String dappId) async {
    final response = await http.post(
      Uri.parse('$API_BASE/dapps/favorites'),
      headers: _headers,
      body: json.encode({'dappId': dappId}),
    );
    
    return response.statusCode == 200;
  }
  
  // Remove from favorites
  Future<bool> removeFavorite(String dappId) async {
    final response = await http.delete(
      Uri.parse('$API_BASE/dapps/favorites/$dappId'),
      headers: _headers,
    );
    
    return response.statusCode == 200;
  }
  
  // Get browsing history
  Future<List<DAppHistory>> getHistory() async {
    final response = await http.get(
      Uri.parse('$API_BASE/dapps/history'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((h) => DAppHistory.fromJson(h)).toList();
    }
    return [];
  }
  
  // Add to history
  Future<bool> addToHistory(String dappId, String url) async {
    final response = await http.post(
      Uri.parse('$API_BASE/dapps/history'),
      headers: _headers,
      body: json.encode({
        'dappId': dappId,
        'url': url,
      }),
    );
    
    return response.statusCode == 200;
  }
  
  // Clear history
  Future<bool> clearHistory() async {
    final response = await http.delete(
      Uri.parse('$API_BASE/dapps/history'),
      headers: _headers,
    );
    
    return response.statusCode == 200;
  }
}

class DApp {
  final String id;
  final String name;
  final String url;
  final String description;
  final String logoUrl;
  final String category;
  final double rating;
  final int users;
  final int volume24h;
  final bool isVerified;
  final List<String> chains;
  
  DApp({
    required this.id,
    required this.name,
    required this.url,
    required this.description,
    required this.logoUrl,
    required this.category,
    required this.rating,
    required this.users,
    required this.volume24h,
    required this.isVerified,
    required this.chains,
  });
  
  factory DApp.fromJson(Map<String, dynamic> json) {
    return DApp(
      id: json['id'] ?? '',
      name: json['name'] ?? '',
      url: json['url'] ?? '',
      description: json['description'] ?? '',
      logoUrl: json['logoUrl'] ?? '',
      category: json['category'] ?? '',
      rating: (json['rating'] ?? 0).toDouble(),
      users: json['users'] ?? 0,
      volume24h: json['volume24h'] ?? 0,
      isVerified: json['isVerified'] ?? false,
      chains: List<String>.from(json['chains'] ?? []),
    );
  }
}

class DAppCategory {
  final String id;
  final String name;
  final String icon;
  final int dappCount;
  
  DAppCategory({
    required this.id,
    required this.name,
    required this.icon,
    required this.dappCount,
  });
  
  factory DAppCategory.fromJson(Map<String, dynamic> json) {
    return DAppCategory(
      id: json['id'] ?? '',
      name: json['name'] ?? '',
      icon: json['icon'] ?? '',
      dappCount: json['dappCount'] ?? 0,
    );
  }
}

class DAppHistory {
  final String dappId;
  final String dappName;
  final String url;
  final DateTime visitedAt;
  
  DAppHistory({
    required this.dappId,
    required this.dappName,
    required this.url,
    required this.visitedAt,
  });
  
  factory DAppHistory.fromJson(Map<String, dynamic> json) {
    return DAppHistory(
      dappId: json['dappId'] ?? '',
      dappName: json['dappName'] ?? '',
      url: json['url'] ?? '',
      visitedAt: DateTime.parse(json['visitedAt']),
    );
  }
}
