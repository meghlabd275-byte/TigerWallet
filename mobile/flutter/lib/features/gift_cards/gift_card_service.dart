// Gift Card Service - Flutter Mobile
// Complete gift card system with real backend

import 'dart:convert';
import 'package:http/http.dart' as http;

class GiftCardService {
  static const String API_BASE = 'https://api.tigerwallet.com/api/v1';
  String? _token;
  
  GiftCardService({String? token}) : _token = token;
  
  Map<String, String> get _headers => {
    'Content-Type': 'application/json',
    if (_token != null) 'Authorization': 'Bearer $_token',
  };
  
  // Get available templates
  Future<List<GiftCardTemplate>> getTemplates() async {
    final response = await http.get(
      Uri.parse('$API_BASE/giftcards/templates'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((t) => GiftCardTemplate.fromJson(t)).toList();
    }
    return [];
  }
  
  // Create gift card
  Future<GiftCard> createGiftCard({
    required String token,
    required double amount,
    String? templateId,
  }) async {
    final response = await http.post(
      Uri.parse('$API_BASE/giftcards'),
      headers: _headers,
      body: json.encode({
        'token': token,
        'amount': amount,
        'templateId': templateId,
      }),
    );
    
    if (response.statusCode == 201) {
      final data = json.decode(response.body);
      return GiftCard.fromJson(data['data']);
    }
    throw Exception('Failed to create gift card');
  }
  
  // Redeem gift card
  Future<GiftCard> redeemGiftCard(String code) async {
    final response = await http.post(
      Uri.parse('$API_BASE/giftcards/redeem'),
      headers: _headers,
      body: json.encode({'code': code}),
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return GiftCard.fromJson(data['data']);
    }
    throw Exception('Failed to redeem gift card');
  }
  
  // Check balance
  Future<GiftCard?> checkBalance(String code) async {
    final response = await http.get(
      Uri.parse('$API_BASE/giftcards/$code/balance'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return GiftCard.fromJson(data['data']);
    }
    return null;
  }
  
  // Get user's created cards
  Future<List<GiftCard>> getCreatedCards() async {
    final response = await http.get(
      Uri.parse('$API_BASE/giftcards/created'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((c) => GiftCard.fromJson(c)).toList();
    }
    return [];
  }
  
  // Get user's redeemed cards
  Future<List<GiftCard>> getRedeemedCards() async {
    final response = await http.get(
      Uri.parse('$API_BASE/giftcards/redeemed'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((c) => GiftCard.fromJson(c)).toList();
    }
    return [];
  }
}

class GiftCardTemplate {
  final String id;
  final String name;
  final String imageUrl;
  final bool isActive;
  
  GiftCardTemplate({
    required this.id,
    required this.name,
    required this.imageUrl,
    required this.isActive,
  });
  
  factory GiftCardTemplate.fromJson(Map<String, dynamic> json) {
    return GiftCardTemplate(
      id: json['id'] ?? '',
      name: json['name'] ?? '',
      imageUrl: json['imageUrl'] ?? '',
      isActive: json['isActive'] ?? false,
    );
  }
}

class GiftCard {
  final String id;
  final String code;
  final String token;
  final double amount;
  final String? templateId;
  final String status;
  final DateTime? expiresAt;
  final DateTime createdAt;
  final DateTime? redeemedAt;
  final String? redeemedBy;
  
  GiftCard({
    required this.id,
    required this.code,
    required this.token,
    required this.amount,
    this.templateId,
    required this.status,
    this.expiresAt,
    required this.createdAt,
    this.redeemedAt,
    this.redeemedBy,
  });
  
  factory GiftCard.fromJson(Map<String, dynamic> json) {
    return GiftCard(
      id: json['id'] ?? '',
      code: json['code'] ?? '',
      token: json['token'] ?? '',
      amount: (json['amount'] ?? 0).toDouble(),
      templateId: json['templateId'],
      status: json['status'] ?? 'ACTIVE',
      expiresAt: json['expiresAt'] != null ? DateTime.parse(json['expiresAt']) : null,
      createdAt: DateTime.parse(json['createdAt']),
      redeemedAt: json['redeemedAt'] != null ? DateTime.parse(json['redeemedAt']) : null,
      redeemedBy: json['redeemedBy'],
    );
  }
}
