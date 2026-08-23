// P2P Trading Service - Flutter - Real API Connection
// Production-ready with NO mock data

import 'dart:convert';
import 'package:http/http.dart' as http;

class P2PApiClient {
  static const String baseUrl = 'http://localhost:8443/api/v1';
  String? _token;
  
  P2PApiClient({String? token}) : _token = token;
  
  Map<String, String> get _headers => {
    'Content-Type': 'application/json',
    if (_token != null) 'Authorization': 'Bearer $_token',
  };
  
  Future<List<P2PAdvert>> getAdverts({
    String? token, String? side, String? fiatCurrency,
    String? paymentMethod, int page = 1, int limit = 20,
  }) async {
    final queryParams = {
      if (token != null) 'token': token,
      if (side != null) 'side': side,
      if (fiatCurrency != null) 'fiatCurrency': fiatCurrency,
      if (paymentMethod != null && paymentMethod != 'All') 'paymentMethod': paymentMethod,
      'page': page.toString(), 'limit': limit.toString(),
    };
    
    final uri = Uri.parse('$baseUrl/p2p/adverts').replace(queryParameters: queryParams);
    final response = await http.get(uri, headers: _headers);
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((a) => P2PAdvert.fromJson(a)).toList();
    }
    throw Exception('Failed to load adverts');
  }
  
  Future<P2POrder> createOrder(String advertId, double amount) async {
    final response = await http.post(
      Uri.parse('$baseUrl/p2p/orders'), headers: _headers,
      body: json.encode({'advertId': advertId, 'amount': amount}),
    );
    
    if (response.statusCode == 200) {
      return P2POrder.fromJson(json.decode(response.body)['data']);
    }
    throw Exception('Failed to create order');
  }
  
  Future<List<P2POrder>> getOrders({String? status}) async {
    final uri = Uri.parse('$baseUrl/p2p/orders').replace(
      queryParameters: status != null ? {'status': status} : null,
    );
    final response = await http.get(uri, headers: _headers);
    
    if (response.statusCode == 200) {
      return (json.decode(response.body)['data'] as List).map((o) => P2POrder.fromJson(o)).toList();
    }
    throw Exception('Failed to load orders');
  }
  
  Future<void> markAsPaid(String orderId, String paymentProof) async {
    final response = await http.post(
      Uri.parse('$baseUrl/p2p/orders/$orderId/pay'), headers: _headers,
      body: json.encode({'paymentProof': paymentProof}),
    );
    if (response.statusCode != 200) throw Exception('Failed to mark as paid');
  }
  
  Future<void> confirmPayment(String orderId) async {
    final response = await http.post(
      Uri.parse('$baseUrl/p2p/orders/$orderId/confirm'), headers: _headers,
    );
    if (response.statusCode != 200) throw Exception('Failed to confirm');
  }
  
  Future<void> cancelOrder(String orderId, String reason) async {
    final response = await http.post(
      Uri.parse('$baseUrl/p2p/orders/$orderId/cancel'), headers: _headers,
      body: json.encode({'reason': reason}),
    );
    if (response.statusCode != 200) throw Exception('Failed to cancel');
  }
  
  Future<void> openDispute(String orderId, String reason) async {
    final response = await http.post(
      Uri.parse('$baseUrl/p2p/orders/$orderId/dispute'), headers: _headers,
      body: json.encode({'reason': reason}),
    );
    if (response.statusCode != 200) throw Exception('Failed to open dispute');
  }
  
  Future<List<PaymentMethod>> getPaymentMethods() async {
    final response = await http.get(Uri.parse('$baseUrl/p2p/payment-methods'), headers: _headers);
    if (response.statusCode == 200) {
      return (json.decode(response.body)['data'] as List).map((m) => PaymentMethod.fromJson(m)).toList();
    }
    throw Exception('Failed to load payment methods');
  }
  
  Future<List<FiatCurrency>> getFiatCurrencies() async {
    final response = await http.get(Uri.parse('$baseUrl/p2p/fiat-currencies'), headers: _headers);
    if (response.statusCode == 200) {
      return (json.decode(response.body)['data'] as List).map((c) => FiatCurrency.fromJson(c)).toList();
    }
    throw Exception('Failed to load currencies');
  }
}

class P2PAdvert {
  final String id, merchantId, username, avatar, side, token, fiatCurrency, paymentMethod;
  final double price, minAmount, maxAmount, availableAmount;
  final int ordersCompleted;
  final double completionRate, avgReleaseTime;
  final bool isOnline, isMerchant, isVerified;
  final String? merchantLevel;
  final double? collateralLocked;
  final double securityScore;

  P2PAdvert({required this.id, required this.merchantId, required this.username, required this.avatar,
    required this.side, required this.token, required this.fiatCurrency, required this.paymentMethod,
    required this.price, required this.minAmount, required this.maxAmount, required this.availableAmount,
    required this.ordersCompleted, required this.completionRate, required this.avgReleaseTime,
    required this.isOnline, required this.isMerchant, required this.isVerified, this.merchantLevel,
    this.collateralLocked, required this.securityScore});

  factory P2PAdvert.fromJson(Map<String, dynamic> json) => P2PAdvert(
    id: json['id'], merchantId: json['merchantId'], username: json['username'], avatar: json['avatar'],
    side: json['side'], token: json['token'], fiatCurrency: json['fiatCurrency'],
    paymentMethod: json['paymentMethod'], price: (json['price'] as num).toDouble(),
    minAmount: (json['minAmount'] as num).toDouble(), maxAmount: (json['maxAmount'] as num).toDouble(),
    availableAmount: (json['availableAmount'] as num).toDouble(), ordersCompleted: json['ordersCompleted'],
    completionRate: (json['completionRate'] as num).toDouble(), avgReleaseTime: (json['avgReleaseTime'] as num).toDouble(),
    isOnline: json['isOnline'] ?? false, isMerchant: json['isMerchant'] ?? false, isVerified: json['isVerified'] ?? false,
    merchantLevel: json['merchantLevel'], collateralLocked: json['collateralLocked'] != null ? (json['collateralLocked'] as num).toDouble() : null,
    securityScore: (json['securityScore'] as num?)?.toDouble() ?? 100,
  );
}

class P2POrder {
  final String id, advertId, side, token, fiatCurrency, paymentMethod, status;
  final double price, amount, fiatAmount;
  final double? buyerDeposit, sellerDeposit;
  final DateTime createdAt;

  P2POrder({required this.id, required this.advertId, required this.side, required this.token,
    required this.fiatCurrency, required this.paymentMethod, required this.price, required this.amount,
    required this.fiatAmount, required this.status, this.buyerDeposit, this.sellerDeposit, required this.createdAt});

  factory P2POrder.fromJson(Map<String, dynamic> json) => P2POrder(
    id: json['id'], advertId: json['advertId'], side: json['side'], token: json['token'],
    fiatCurrency: json['fiatCurrency'], paymentMethod: json['paymentMethod'],
    price: (json['price'] as num).toDouble(), amount: (json['amount'] as num).toDouble(),
    fiatAmount: (json['fiatAmount'] as num).toDouble(), status: json['status'],
    buyerDeposit: json['buyerDeposit'] != null ? (json['buyerDeposit'] as num).toDouble() : null,
    sellerDeposit: json['sellerDeposit'] != null ? (json['sellerDeposit'] as num).toDouble() : null,
    createdAt: DateTime.parse(json['createdAt']),
  );
}

class PaymentMethod {
  final String id, name, type;
  PaymentMethod({required this.id, required this.name, required this.type});
  factory PaymentMethod.fromJson(Map<String, dynamic> json) => PaymentMethod(id: json['id'], name: json['name'], type: json['type']);
}

class FiatCurrency {
  final String code, name, symbol;
  FiatCurrency({required this.code, required this.name, required this.symbol});
  factory FiatCurrency.fromJson(Map<String, dynamic> json) => FiatCurrency(code: json['code'], name: json['name'], symbol: json['symbol']);
}

final p2pApi = P2PApiClient();
