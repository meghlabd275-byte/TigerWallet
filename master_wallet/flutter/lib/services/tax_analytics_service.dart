/**
 * TaxAnalyticsService - Flutter Implementation
 *
 * The canonical Go backend (:8450) does NOT provide a tax engine (no cost
 * basis, capital-gains, lot-tracking, or report-generation endpoints). Raw
 * data is therefore sourced from the canonical endpoints:
 *   - GET /api/v1/master-wallet/:id/transactions
 *   - GET /api/v1/price?coin_id=
 *   - GET /api/v1/master-wallet/:id/analytics/volume
 * Tax-specific computations (cost basis, tax events, tax summaries, reports)
 * fail closed rather than fabricating lots/gains locally, which would require
 * a client-side accounting engine the backend does not authorize.
 *
 * NO in-memory transaction/lot/price stores, NO simulated prices, NO fake
 * tax reports. The backend is the sole source of truth for transaction and
 * price data.
 */

import 'dart:convert';
import 'package:http/http.dart' as http;

class TaxAnalyticsService {
  static TaxAnalyticsService? _instance;
  static TaxAnalyticsService get instance {
    _instance ??= TaxAnalyticsService._();
    return _instance!;
  }

  TaxAnalyticsService._();

  static const String API_BASE = String.fromEnvironment(
    'MASTER_WALLET_API_URL',
    defaultValue: 'http://localhost:8450',
  );

  final String masterWalletId;
  String? _token;

  TaxAnalyticsService({required this.masterWalletId, String? token})
      : _token = token;

  void setToken(String? token) => _token = token;

  Map<String, String> get _headers => {
        'Content-Type': 'application/json',
        if (_token != null) 'Authorization': 'Bearer $_token',
      };

  String get _txBase => '$API_BASE/api/v1/master-wallet/$masterWalletId/transactions';

  Exception _err(http.Response r, String op) =>
      Exception('tax $op failed (${r.statusCode}): ${r.body}');

  Exception _unsupported(String op) => UnimplementedError(
        'tax $op is not supported by the canonical backend contract. '
        'The backend exposes no tax engine (cost basis, gains/losses, '
        'reports). Fetch transactions via getTransactions() and price via '
        'getTokenPrice() and perform any tax computation outside this '
        'client or wait for a backend tax API.');

  /// Fetch real transactions from the canonical backend for a date range.
  /// `startDate`/`endDate` are millisecond epochs; forwarded as query params.
  Future<List<Transaction>> getTransactions({
    int? startDate,
    int? endDate,
  }) async {
    final uri = Uri.parse(_txBase).replace(
      queryParameters: {
        // The canonical backend reads the wallet id from this query param;
        // keep it in sync with the :id path segment so the live query returns data.
        'master_wallet_id': masterWalletId,
        if (startDate != null) 'start': startDate.toString(),
        if (endDate != null) 'end': endDate.toString(),
      },
    );
    final r = await http.get(uri, headers: _headers);
    if (r.statusCode != 200) throw _err(r, 'getTransactions');
    final data = json.decode(r.body);
    final list = (data['transactions'] as List?) ?? const [];
    return list
        .map((t) => Transaction.fromJson(t as Map<String, dynamic>))
        .toList();
  }

  /// Fetch a real token price (USD + 24h change) from the canonical price
  /// endpoint. `coinId` is the CoinGecko-style id (e.g. 'ethereum').
  Future<double> getTokenPrice(String coinId) async {
    final uri = Uri.parse('$API_BASE/api/v1/price').replace(
      queryParameters: {'coin_id': coinId},
    );
    final r = await http.get(uri, headers: _headers);
    if (r.statusCode != 200) throw _err(r, 'getTokenPrice');
    final data = json.decode(r.body);
    final usd = data['usd'];
    if (usd == null) {
      throw Exception('price response missing usd value');
    }
    return (usd as num).toDouble();
  }

  // ==================== Tax-specific computations (unsupported) ====================

  void setDefaultJurisdiction(String jurisdiction) {
    throw _unsupported('setDefaultJurisdiction');
  }

  void setDefaultMethod(String method) {
    throw _unsupported('setDefaultMethod');
  }

  JurisdictionConfig getJurisdictionConfig([String? jurisdiction]) {
    throw _unsupported('getJurisdictionConfig');
  }

  void importTransaction(Transaction tx) {
    throw _unsupported('importTransaction');
  }

  Map<String, dynamic> calculateCostBasis(
    String token,
    String sellAmount,
    String? method,
    int? timestamp,
  ) {
    throw _unsupported('calculateCostBasis');
  }

  TaxEvent processSale(Transaction saleTx, [String? method]) {
    throw _unsupported('processSale');
  }

  TaxSummary calculateTaxSummary(
      int startDate, int endDate, [String? jurisdiction]) {
    throw _unsupported('calculateTaxSummary');
  }

  List<TaxEvent> getTaxEvents(
      int startDate, int endDate, [String? jurisdiction]) {
    throw _unsupported('getTaxEvents');
  }

  TaxReport generateTaxReport(
      int startDate, int endDate, [String? jurisdiction]) {
    throw _unsupported('generateTaxReport');
  }

  String exportAsCSV(TaxReport report) {
    throw _unsupported('exportAsCSV');
  }

  String exportAsJSON(TaxReport report) {
    throw _unsupported('exportAsJSON');
  }
}

// ==================== DTOs ====================

class Transaction {
  final String id;
  final String hash;
  final int timestamp;
  final int blockNumber;
  final String from;
  final String to;
  final String token;
  final String amount;
  final double valueUSD;
  final String fee;
  final double feeUSD;
  final String status;

  Transaction({
    required this.id,
    required this.hash,
    required this.timestamp,
    required this.blockNumber,
    required this.from,
    required this.to,
    required this.token,
    required this.amount,
    required this.valueUSD,
    required this.fee,
    required this.feeUSD,
    required this.status,
  });

  Map<String, dynamic> toJson() => {
        'id': id,
        'hash': hash,
        'timestamp': timestamp,
        'blockNumber': blockNumber,
        'from': from,
        'to': to,
        'token': token,
        'amount': amount,
        'valueUSD': valueUSD,
        'fee': fee,
        'feeUSD': feeUSD,
        'status': status,
      };

  factory Transaction.fromJson(Map<String, dynamic> json) => Transaction(
        id: json['id']?.toString() ?? json['hash']?.toString() ??
            json['tx_hash']?.toString() ?? '',
        hash: json['tx_hash']?.toString() ?? json['hash']?.toString() ?? '',
        timestamp: _parseTimestamp(json['created_at'] ??
            json['timestamp'] ??
            json['confirmed_at']),
        blockNumber: _asInt(json['blockNumber'] ?? json['block_number']) ?? 0,
        from: json['from']?.toString() ?? '',
        to: json['to']?.toString() ?? '',
        token: json['token']?.toString() ??
            json['token_symbol']?.toString() ??
            json['asset']?.toString() ??
            '',
        amount: json['amount']?.toString() ?? '0',
        valueUSD: _asDouble(json['valueUSD'] ?? json['value_usd']) ?? 0.0,
        fee: json['fee']?.toString() ?? '0',
        feeUSD: _asDouble(json['feeUSD'] ?? json['fee_usd']) ?? 0.0,
        status: json['status']?.toString() ?? 'unknown',
      );

  static int? _asInt(dynamic v) {
    if (v == null) return null;
    if (v is int) return v;
    return int.tryParse(v.toString());
  }

  static double? _asDouble(dynamic v) {
    if (v == null) return null;
    if (v is double) return v;
    if (v is int) return v.toDouble();
    return double.tryParse(v.toString());
  }

  static int _parseTimestamp(dynamic v) {
    if (v == null) return 0;
    if (v is int) return v;
    final parsed = DateTime.tryParse(v.toString());
    return parsed?.millisecondsSinceEpoch ?? 0;
  }
}

class TaxLot {
  final String id;
  final String transactionId;
  final String token;
  final String amount;
  final String costBasis;
  final double costBasisUSD;
  final int acquisitionDate;
  final String holdingPeriod;

  TaxLot({
    required this.id,
    required this.transactionId,
    required this.token,
    required this.amount,
    required this.costBasis,
    required this.costBasisUSD,
    required this.acquisitionDate,
    required this.holdingPeriod,
  });

  Map<String, dynamic> toJson() => {
        'id': id,
        'transactionId': transactionId,
        'token': token,
        'amount': amount,
        'costBasis': costBasis,
        'costBasisUSD': costBasisUSD,
        'acquisitionDate': acquisitionDate,
        'holdingPeriod': holdingPeriod,
      };
}

class TaxEvent {
  final String id;
  final String type;
  final String token;
  final String amount;
  final String proceeds;
  final double proceedsUSD;
  final String costBasis;
  final double costBasisUSD;
  final String gainLoss;
  final double gainLossUSD;
  final String transactionId;
  final int timestamp;
  final String jurisdiction;
  final String? holdingPeriod;

  TaxEvent({
    required this.id,
    required this.type,
    required this.token,
    required this.amount,
    required this.proceeds,
    required this.proceedsUSD,
    required this.costBasis,
    required this.costBasisUSD,
    required this.gainLoss,
    required this.gainLossUSD,
    required this.transactionId,
    required this.timestamp,
    required this.jurisdiction,
    this.holdingPeriod,
  });

  Map<String, dynamic> toJson() => {
        'id': id,
        'type': type,
        'token': token,
        'amount': amount,
        'proceeds': proceeds,
        'proceedsUSD': proceedsUSD,
        'costBasis': costBasis,
        'costBasisUSD': costBasisUSD,
        'gainLoss': gainLoss,
        'gainLossUSD': gainLossUSD,
        'transactionId': transactionId,
        'timestamp': timestamp,
        'jurisdiction': jurisdiction,
        'holdingPeriod': holdingPeriod,
      };
}

class TaxReport {
  final String id;
  final int generatedAt;
  final int startDate;
  final int endDate;
  final String jurisdiction;
  final Map<String, String> summary;
  final List<TaxEvent> events;
  final List<Transaction> transactions;

  TaxReport({
    required this.id,
    required this.generatedAt,
    required this.startDate,
    required this.endDate,
    required this.jurisdiction,
    required this.summary,
    required this.events,
    required this.transactions,
  });
}

class TaxSummary {
  final double totalProceeds;
  final double totalCostBasis;
  final double totalGainLoss;
  final double shortTermGain;
  final double shortTermLoss;
  final double longTermGain;
  final double longTermLoss;
  final double totalIncome;
  final double totalFees;
  final double washSaleAdjustments;
  final Map<String, TokenBreakdown> tokenBreakdown;

  TaxSummary({
    required this.totalProceeds,
    required this.totalCostBasis,
    required this.totalGainLoss,
    required this.shortTermGain,
    required this.shortTermLoss,
    required this.longTermGain,
    required this.longTermLoss,
    required this.totalIncome,
    required this.totalFees,
    required this.washSaleAdjustments,
    required this.tokenBreakdown,
  });
}

class TokenBreakdown {
  final double proceeds;
  final double costBasis;
  final double gainLoss;
  final int count;

  TokenBreakdown({
    required this.proceeds,
    required this.costBasis,
    required this.gainLoss,
    required this.count,
  });
}

class JurisdictionConfig {
  final String code;
  final String name;
  final int shortTermThreshold;
  final double capitalGainsRate;
  final double incomeTaxRate;
  final double reportingThreshold;
  final String currency;

  JurisdictionConfig({
    required this.code,
    required this.name,
    required this.shortTermThreshold,
    required this.capitalGainsRate,
    required this.incomeTaxRate,
    required this.reportingThreshold,
    required this.currency,
  });
}
