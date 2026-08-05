/**
 * TaxAnalyticsService - Flutter Implementation
 * Complete tax tracking and reporting for Master Wallet
 * Features: Cost basis, Capital gains/losses, Multi-jurisdiction
 */

import 'dart:convert';
import 'dart:math';

class TaxAnalyticsService {
  static TaxAnalyticsService? _instance;
  static TaxAnalyticsService get instance {
    _instance ??= TaxAnalyticsService._();
    return _instance!;
  }

  TaxAnalyticsService._();

  String _defaultJurisdiction = 'US';
  String _defaultMethod = 'FIFO';

  final Map<String, Transaction> _transactions = {};
  final Map<String, TaxLot> _taxLots = {};
  final Map<String, Map<int, double>> _tokenPrices = {};

  // ==================== Models ====================

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

  final Map<String, JurisdictionConfig> _jurisdictions = {
    'US': JurisdictionConfig(
      code: 'US',
      name: 'United States',
      shortTermThreshold: 365,
      capitalGainsRate: 20,
      incomeTaxRate: 37,
      reportingThreshold: 600,
      currency: 'USD',
    ),
    'UK': JurisdictionConfig(
      code: 'UK',
      name: 'United Kingdom',
      shortTermThreshold: 0,
      capitalGainsRate: 20,
      incomeTaxRate: 45,
      reportingThreshold: 0,
      currency: 'GBP',
    ),
    'EU': JurisdictionConfig(
      code: 'EU',
      name: 'European Union',
      shortTermThreshold: 365,
      capitalGainsRate: 25,
      incomeTaxRate: 45,
      reportingThreshold: 0,
      currency: 'EUR',
    ),
  };

  // ==================== Configuration ====================

  void setDefaultJurisdiction(String jurisdiction) {
    if (_jurisdictions.containsKey(jurisdiction)) {
      _defaultJurisdiction = jurisdiction;
    }
  }

  void setDefaultMethod(String method) {
    _defaultMethod = method;
  }

  JurisdictionConfig getJurisdictionConfig([String? jurisdiction]) {
    return _jurisdictions[jurisdiction ?? _defaultJurisdiction] ??
        _jurisdictions['US']!;
  }

  // ==================== Transaction Import ====================

  void importTransaction(Transaction tx) {
    _transactions[tx.id] = tx;

    if (tx.status == 'confirmed' && tx.valueUSD > 0 && tx.from != tx.to) {
      _createTaxLot(tx);
    }
  }

  void _createTaxLot(Transaction tx) {
    final lot = TaxLot(
      id: 'lot_${tx.id}',
      transactionId: tx.id,
      token: tx.token,
      amount: tx.amount,
      costBasis: tx.valueUSD.toString(),
      costBasisUSD: tx.valueUSD,
      acquisitionDate: tx.timestamp,
      holdingPeriod: 'short',
    );
    _taxLots[lot.id] = lot;
  }

  // ==================== Price Data ====================

  void updateTokenPrice(String token, int timestamp, double priceUSD) {
    _tokenPrices[token] ??= {};
    _tokenPrices[token]![timestamp] = priceUSD;
  }

  double getTokenPrice(String token, int timestamp) {
    final prices = _tokenPrices[token];
    if (prices == null || prices.isEmpty) return 0;

    double closestPrice = 0;
    int minDiff = 999999999;

    for (final entry in prices.entries) {
      final diff = (entry.key - timestamp).abs();
      if (diff < minDiff) {
        minDiff = diff;
        closestPrice = entry.value;
      }
    }

    return closestPrice;
  }

  // ==================== Tax Calculation ====================

  Map<String, dynamic> calculateCostBasis(
    String token,
    String sellAmount,
    String? method,
    int? timestamp,
  ) {
    final lots = _getAvailableLots(token, timestamp);
    final methodToUse = method ?? _defaultMethod;

    List<TaxLot> sortedLots = List.from(lots);
    switch (methodToUse) {
      case 'FIFO':
        sortedLots.sort((a, b) => a.acquisitionDate.compareTo(b.acquisitionDate));
        break;
      case 'LIFO':
        sortedLots.sort((a, b) => b.acquisitionDate.compareTo(a.acquisitionDate));
        break;
      case 'HIFO':
        sortedLots.sort((a, b) =>
            double.parse(b.costBasis).compareTo(double.parse(a.costBasis)));
        break;
      case 'AVERAGE':
        final totalAmount = lots.fold<double>(
            0, (sum, lot) => sum + double.parse(lot.amount));
        final totalCost =
            lots.fold<double>(0, (sum, lot) => sum + lot.costBasisUSD);
        final avgCost = totalAmount > 0 ? totalCost / totalAmount : 0;
        return {
          'costBasis': (double.parse(sellAmount) * avgCost).toString(),
          'lotsUsed': [],
        };
    }

    double remaining = double.parse(sellAmount);
    double totalCostBasis = 0;
    final lotsUsed = <Map<String, dynamic>>[];

    for (final lot in sortedLots) {
      if (remaining <= 0) break;

      final lotAmount = double.parse(lot.amount);
      final amountToUse = min(remaining, lotAmount);
      final costBasisForAmount =
          (amountToUse / lotAmount) * lot.costBasisUSD;

      totalCostBasis += costBasisForAmount;
      lotsUsed.add({
        'lotId': lot.id,
        'amount': amountToUse.toString(),
        'costBasis': costBasisForAmount.toString(),
      });

      remaining -= amountToUse;
    }

    return {
      'costBasis': totalCostBasis.toString(),
      'lotsUsed': lotsUsed,
    };
  }

  List<TaxLot> _getAvailableLots(String token, int? beforeTimestamp) {
    final cutoff = beforeTimestamp ?? DateTime.now().millisecondsSinceEpoch;
    return _taxLots.values
        .where((lot) =>
            lot.token == token && lot.acquisitionDate < cutoff)
        .toList();
  }

  TaxEvent processSale(Transaction saleTx, [String? method]) {
    final costBasisResult = calculateCostBasis(
      saleTx.token,
      saleTx.amount,
      method,
      saleTx.timestamp,
    );

    final proceeds = saleTx.valueUSD;
    final costBasis = double.parse(costBasisResult['costBasis']);
    final gainLoss = proceeds - costBasis;

    // Determine holding period
    final jurisdiction = getJurisdictionConfig();
    final lotsUsed = costBasisResult['lotsUsed'] as List;
    String holdingPeriod = 'short';

    if (lotsUsed.isNotEmpty) {
      final earliestAcquisition = lotsUsed
          .map((l) => _taxLots[l['lotId']]?.acquisitionDate ?? 0)
          .reduce((a, b) => a < b ? a : b);
      final daysSinceAcquisition =
          (saleTx.timestamp - earliestAcquisition) / (1000 * 60 * 60 * 24);
      holdingPeriod = daysSinceAcquisition >= jurisdiction.shortTermThreshold
          ? 'long'
          : 'short';
    }

    final event = TaxEvent(
      id: 'event_${saleTx.id}',
      type: 'SELL',
      token: saleTx.token,
      amount: saleTx.amount,
      proceeds: proceeds.toString(),
      proceedsUSD: proceeds,
      costBasis: costBasis.toString(),
      costBasisUSD: costBasis,
      gainLoss: gainLoss.toString(),
      gainLossUSD: gainLoss,
      transactionId: saleTx.id,
      timestamp: saleTx.timestamp,
      jurisdiction: _defaultJurisdiction,
    );

    // Update lots
    _updateLotsAfterSale(lotsUsed);

    return event;
  }

  void _updateLotsAfterSale(List<Map<String, dynamic>> lotsUsed) {
    for (final used in lotsUsed) {
      final lot = _taxLots[used['lotId']];
      if (lot != null) {
        final remaining = double.parse(lot.amount) - double.parse(used['amount']);
        if (remaining <= 0) {
          _taxLots.remove(lot.id);
        } else {
          _taxLots[lot.id] = TaxLot(
            id: lot.id,
            transactionId: lot.transactionId,
            token: lot.token,
            amount: remaining.toString(),
            costBasis: (double.parse(lot.costBasis) * (remaining / double.parse(lot.amount)))
                .toString(),
            costBasisUSD:
                lot.costBasisUSD * (remaining / double.parse(lot.amount)),
            acquisitionDate: lot.acquisitionDate,
            holdingPeriod: lot.holdingPeriod,
          );
        }
      }
    }
  }

  // ==================== Tax Summary ====================

  TaxSummary calculateTaxSummary(int startDate, int endDate, [String? jurisdiction]) {
    final events = getTaxEvents(startDate, endDate, jurisdiction);

    double totalProceeds = 0;
    double totalCostBasis = 0;
    double totalGainLoss = 0;
    double shortTermGain = 0;
    double shortTermLoss = 0;
    double longTermGain = 0;
    double longTermLoss = 0;
    double totalIncome = 0;
    double totalFees = 0;
    final tokenBreakdown = <String, TokenBreakdown>{};

    for (final event in events) {
      if (event.type == 'SELL') {
        totalProceeds += event.proceedsUSD;
        totalCostBasis += event.costBasisUSD;
        totalGainLoss += event.gainLossUSD;

        if (event.gainLossUSD > 0) {
          if (event.holdingPeriod == 'short') {
            shortTermGain += event.gainLossUSD;
          } else {
            longTermGain += event.gainLossUSD;
          }
        } else if (event.gainLossUSD < 0) {
          if (event.holdingPeriod == 'short') {
            shortTermLoss += event.gainLossUSD.abs();
          } else {
            longTermLoss += event.gainLossUSD.abs();
          }
        }

        final tokenData = tokenBreakdown[event.token] ??
            TokenBreakdown(proceeds: 0, costBasis: 0, gainLoss: 0, count: 0);
        tokenBreakdown[event.token] = TokenBreakdown(
          proceeds: tokenData.proceeds + event.proceedsUSD,
          costBasis: tokenData.costBasis + event.costBasisUSD,
          gainLoss: tokenData.gainLoss + event.gainLossUSD,
          count: tokenData.count + 1,
        );
      } else if (['REWARD', 'STAKING', 'MINING', 'AIRDROP'].contains(event.type)) {
        totalIncome += event.proceedsUSD;
      } else if (event.type == 'FEE') {
        totalFees += event.proceedsUSD;
      }
    }

    return TaxSummary(
      totalProceeds: totalProceeds,
      totalCostBasis: totalCostBasis,
      totalGainLoss: totalGainLoss,
      shortTermGain: shortTermGain,
      shortTermLoss: shortTermLoss,
      longTermGain: longTermGain,
      longTermLoss: longTermLoss,
      totalIncome: totalIncome,
      totalFees: totalFees,
      washSaleAdjustments: 0,
      tokenBreakdown: tokenBreakdown,
    );
  }

  List<TaxEvent> getTaxEvents(int startDate, int endDate, [String? jurisdiction]) {
    final events = <TaxEvent>[];

    for (final tx in _transactions.values) {
      if (tx.timestamp < startDate ||
          tx.timestamp > endDate ||
          (jurisdiction != null && _defaultJurisdiction != jurisdiction)) {
        continue;
      }

      if (tx.status == 'confirmed' && tx.valueUSD > 0) {
        events.add(processSale(tx));
      }
    }

    events.sort((a, b) => a.timestamp.compareTo(b.timestamp));
    return events;
  }

  // ==================== Report Generation ====================

  TaxReport generateTaxReport(int startDate, int endDate, [String? jurisdiction]) {
    final events = getTaxEvents(startDate, endDate, jurisdiction);
    final summary = calculateTaxSummary(startDate, endDate, jurisdiction);

    final transactions = _transactions.values
        .where((tx) => tx.timestamp >= startDate && tx.timestamp <= endDate)
        .toList()
      ..sort((a, b) => a.timestamp.compareTo(b.timestamp));

    return TaxReport(
      id: 'report_${DateTime.now().millisecondsSinceEpoch}',
      generatedAt: DateTime.now().millisecondsSinceEpoch,
      startDate: startDate,
      endDate: endDate,
      jurisdiction: jurisdiction ?? _defaultJurisdiction,
      summary: {
        'totalProceeds': summary.totalProceeds.toString(),
        'totalCostBasis': summary.totalCostBasis.toString(),
        'totalGainLoss': summary.totalGainLoss.toString(),
        'shortTermGainLoss':
            (summary.shortTermGain - summary.shortTermLoss).toString(),
        'longTermGainLoss':
            (summary.longTermGain - summary.longTermLoss).toString(),
        'totalIncome': summary.totalIncome.toString(),
        'totalFees': summary.totalFees.toString(),
      },
      events: events,
      transactions: transactions,
    );
  }

  String exportAsCSV(TaxReport report) {
    final lines = <String>[];
    lines.add(
        'Type,Token,Amount,Proceeds,Cost Basis,Gain/Loss,Date,Jurisdiction');

    for (final event in report.events) {
      lines.add(
          '${event.type},${event.token},${event.amount},${event.proceedsUSD},${event.costBasisUSD},${event.gainLossUSD},${DateTime.fromMillisecondsSinceEpoch(event.timestamp).toIso8601String()},${event.jurisdiction}');
    }

    return lines.join('\n');
  }

  String exportAsJSON(TaxReport report) {
    return jsonEncode({
      'id': report.id,
      'generatedAt': report.generatedAt,
      'startDate': report.startDate,
      'endDate': report.endDate,
      'jurisdiction': report.jurisdiction,
      'summary': report.summary,
      'events': report.events.map((e) => e.toJson()).toList(),
    });
  }
}
